---
Title: Moments React Chat Widget Architecture
Ticket: PI-003-PORT-TO-REACT
Status: active
Topics:
    - react
    - webchat
    - moments
    - pinocchio
    - geppetto
    - frontend
    - backend
    - websocket
    - redux
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-mento/go/pkg/webchat/stream_coordinator.go
      Note: Reference implementation of event stream consumption + SEM translation + WS broadcast callbacks
    - Path: moments/backend/pkg/app/sink_registry.go
      Note: Sink pipeline builder; structuredsink extractors (avoid sink-owned conversation state)
    - Path: moments/backend/pkg/webchat/router.go
      Note: Go webchat HTTP + WS handlers; attaches sinks via events.WithEventSinks
    - Path: moments/web/src/platform/chat/hooks/useChatStream.ts
      Note: WebSocket connection + SEM envelope parsing + routing to Redux
    - Path: moments/web/src/platform/chat/state/chatQueueSlice.ts
      Note: Queue-based send path with 409 retry/backoff
    - Path: moments/web/src/platform/sem/registry.ts
      Note: Frontend SEM handler registry (type -> add/upsert cmd)
    - Path: moments/web/src/platform/timeline/state/timelineSlice.ts
      Note: Normalized timeline entity store and streaming helpers
ExternalSources:
    - https://redux-toolkit.js.org/
    - https://redux-toolkit.js.org/rtk-query/overview
    - https://watermill.io/docs/
    - https://github.com/gorilla/websocket
    - https://docs.aws.amazon.com/elasticloadbalancing/latest/application/application-load-balancers.html
Summary: Deep analysis of the Moments/go-go-mento React chat widget + Redux Toolkit state model + SEM WebSocket protocol, and the Go backend pipeline that converts Geppetto events (including structuredsink extractor outputs) into SEM frames and widget updates.
LastUpdated: 2026-01-24T13:52:51.728817619-05:00
WhatFor: Use as a reference architecture to port/upgrade Pinocchio webchat to React with Moments-style streaming UI, SEM event routing, and rich timeline widgets.
WhenToUse: When implementing or debugging the React chat UI, SEM event schema, Go webchat router, sink extractors, or the end-to-end streaming pipeline.
---


# Moments / go-go-mento Webchat: React + Redux Toolkit + SEM WebSocket — A Deep Architectural Analysis

## Abstract

The Moments and go-go-mento webchat systems implement a particular (and unusually powerful) design pattern: **an event-sourced UI** for chat. The backend emits a stream of typed events (LLM token deltas, tool lifecycle, structured “widget” outputs, debug pauses, etc.). The frontend treats those events as a protocol (“SEM”), routes them through a handler registry, and turns them into a normalized set of timeline entities stored in Redux. Rendering is then a pure function of those entities: each entity kind has a registered widget renderer.

This document explains that system end-to-end, emphasizing the “contracts” (schemas, IDs, ordering, upsert semantics) that must remain consistent if you want Pinocchio to achieve Moments parity.

## Scope and what this is (and is not)

This document focuses on:
- **React widget wiring**: where chat mounts, how the chat timeline is rendered, and how “widgets” appear.
- **Redux Toolkit patterns**: timeline entity model, queue-based send path, and RTK Query API slices.
- **WebSocket + SEM protocol**: envelope shape, keepalive, error handling, and registry-based routing.
- **Go backend pipeline**: how Geppetto emits events, how sinks (including structuredsink extractors) produce typed events, how those become SEM frames, and how frames are broadcast.

It does not attempt to explain the entire Moments product or all middleware/tool behavior; it zooms in on the webchat streaming pipeline and the parts of the system that shape the chat UX.

## Primary sources (code and local docs)

This analysis is based on:
- Moments frontend: `moments/web/src/platform/chat/`, `moments/web/src/platform/sem/`, `moments/web/src/platform/timeline/`, `moments/web/src/store/`, `moments/web/src/features/chat/`
- Moments backend: `moments/backend/pkg/webchat/`, `moments/backend/pkg/app/sink_registry.go`, `moments/backend/pkg/sem/`, `moments/backend/pkg/inference/middleware/*/*extractor*.go`
- go-go-mento frontend: `go-go-mento/web/src/ws/`, `go-go-mento/web/src/sem/`, `go-go-mento/web/src/store/`
- go-go-mento backend: `go-go-mento/go/pkg/webchat/`, `go-go-mento/go/pkg/sem/registry/`, `go-go-mento/go/pkg/persistence/timelinehydration/`
- Local documentation that already exists in-repo (highly relevant):
  - `go-go-mento/docs/reference/webchat/frontend-integration.md`
  - `go-go-mento/docs/reference/webchat/sem-and-widgets.md`
  - `go-go-mento/docs/reference/webchat/backend-internals.md`
  - `go-go-mento/docs/reference/webchat/engine-builder.md`
  - `go-go-mento/docs/reference/persistence/timeline-hydration.md`
  - Moments migration tickets: `moments/ttmp/MIGRATE_CHAT-migrate-webchat-into-moments/README.md`, `moments/ttmp/IMPORT-MIDDLEWARE-port-middleware-and-structured-event-emitters-from-mento-playground-go-pkg-webchat-into-webchat/README.md`

## External references (selected)

These are the external “background” documents most useful for the parts that are not project-specific:
- Redux Toolkit docs: https://redux-toolkit.js.org/
- RTK Query docs: https://redux-toolkit.js.org/rtk-query/overview
- Watermill docs (used for pub/sub abstractions): https://watermill.io/docs/
- Gorilla WebSocket (Go WS implementation): https://github.com/gorilla/websocket
- AWS ALB basics (idle timeout/connection behavior context): https://docs.aws.amazon.com/elasticloadbalancing/latest/application/application-load-balancers.html

## A mental model: the chat UI as a state machine over events

Before touching any code, it helps to name the core objects:

1. **Conversation**: identified by a stable `conv_id`. The user’s browser can “attach” and “detach” from it; the server can run inference runs “inside” it.
2. **Event stream**: a time-ordered sequence of events emitted by inference and middleware. Some are “primitive” (token deltas), some are “structured” (a multiple-choice question widget, a planning trace, a team-member suggestion, etc.).
3. **SEM**: the on-the-wire encoding of those events for the web UI.
4. **Timeline entities**: the frontend’s persisted state, keyed by `(conv_id, entity_id)` and typed by `kind`. Updates are generally **upserts**: new data merges into existing entity state.
5. **Widgets**: React renderers selected by `entity.kind`.

The principle is simple:

> The backend emits events.  
> The frontend turns events into entities.  
> Entities render as widgets.

Everything else is engineering to keep that principle correct under streaming, retries, reconnects, and multiple feature-specific event types.

---

# Part I — Frontend architecture

## 1. Where chat mounts: “page” and “sidebar” forms

Moments has at least two “shells” for chat:

1) **Full page chat (POC)**: `moments/web/src/features/chat/ChatPage.tsx`
- Establishes/chooses a `convId`.
- Connects WebSocket streaming via `useChatStream`.
- Sends messages via `enqueueChatMessage` (a Redux thunk) which drives RTK Query chat endpoints.
- Renders timeline entities via the `Timeline` component.

2) **Sidebar chat**: `moments/web/src/features/chat/ChatPageWithSidebar.tsx`
- Uses `useSidebarChat` which wraps the same underlying primitives (timeline state + send queue + WS stream).
- Renders the “sidebar chat” UI component: `moments/web/src/platform/chat/SidebarChat/SidebarChat.tsx` (which itself delegates actual message/tool widget rendering to `Timeline`).

go-go-mento also implements “chat-in-a-sidebar” patterns; compare:
- `go-go-mento/web/src/hooks/useSidebarChat.ts`
- `go-go-mento/web/src/components/SidebarChat/SidebarChat.tsx`

### The key architectural point

**SidebarChat is intentionally “dumb”:** it renders whatever timeline entities it is given. The interesting work is upstream:
- managing the conversation id,
- establishing the WebSocket stream,
- sending messages with retry,
- routing SEM events into timeline entities.

That separation is what allows “chat everywhere” in the product (any page can embed the sidebar widget and pass context).

## 2. Redux Toolkit store composition (Moments)

The Moments Redux store is composed in `moments/web/src/store/store.ts`:
- `apiSlice` (REST-like endpoints under `/api/v1`)
- `rpcSlice` (RPC-like endpoints under `/rpc/v1`)
- `auth` slice
- `timeline` slice (the conversation entity store)
- `chatQueue` slice (send queue and 409 retry behavior)
- plus feature/platform reducers such as `documents`

### Why RTK Query is used twice (apiSlice vs rpcSlice)

Moments distinguishes between:
- `/api/v1` endpoints (identity / integrations / various REST paths) in `apiSlice`,
- `/rpc/v1` endpoints (webchat and “RPC” calls) in `rpcSlice`.

Both base queries share the same high-level design: `fetchBaseQuery` plus a `prepareHeaders` that adds `Authorization: Bearer <token>` using either Redux auth state or `tokenService.getAccessToken()`.

## 3. The send path: from UI input to backend inference

Moments uses a deliberate “two-channel” design:

- **HTTP POST** starts a run (user prompt is delivered to backend):
  - Implemented as RTK Query mutations in `moments/web/src/platform/api/chatApi.ts`:
    - `startChat` → `POST /rpc/v1/chat`
    - `startChatWithProfile` → `POST /rpc/v1/chat/{profile}`
  - Wrapped in a send queue for robustness:
    - `moments/web/src/platform/chat/state/chatQueueSlice.ts`
    - `enqueueChatMessage` queues the message and calls `processQueue` to execute it.
    - `processQueue` handles `409 Conflict` (interpreted as “run in progress”) by retrying shortly, rather than failing immediately.

- **WebSocket** streams the response as SEM events:
  - Implemented in `moments/web/src/platform/chat/hooks/useChatStream.ts`.

This split is not incidental: WebSocket is a poor fit for “starting a run” when you care about idempotence, retries, and request/response error semantics. HTTP is a better fit. WebSocket is then used where it shines: streaming server → client events.

## 4. The timeline slice: the core data model

The timeline slice is the heart of the frontend architecture:
- Moments: `moments/web/src/platform/timeline/state/timelineSlice.ts`
- go-go-mento: `go-go-mento/web/src/store/timeline/timelineSlice.ts`

The state is normalized:

```ts
byConvId: {
  [convId]: {
    byId: { [entityId]: TimelineEntity },
    order: string[],
    displayMode?: 'chat' | 'ui'
  }
}
```

Key reducers (Moments):
- `addEntity`: insert once (no duplicates).
- `upsertEntity`: create if missing; else merge props if `kind` matches; else replace.
- `appendMessageText` / `finalizeMessage`: optimized helpers for streaming assistant text.
- `appendEntityArrayItem`: used for “chunked” widgets (e.g., summary chunks) that accumulate multiple pieces.

### A rule worth stating explicitly: “entity identity” is sacred

The UI’s correctness depends on stable `entity.id` values:
- LLM text streaming must use the same `id` across `llm.start` → `llm.delta` → `llm.final`.
- Tool lifecycle must share the same `tool_call_id` across start/delta/result/done.
- Structured widgets must map “started/update/completed” events to a stable item ID.

If IDs drift, the UI will duplicate widgets instead of updating them.

## 5. Rendering: widget registry and visibility modes

Rendering is dispatched by entity kind:
- Registry: `moments/web/src/platform/timeline/registry.ts`
- Registration side-effects: `moments/web/src/platform/timeline/registerAll.ts`
- Renderer selection: `moments/web/src/platform/timeline/Timeline.tsx`

Each widget registers:
- a renderer function for an entity kind (e.g., `message`, `tool_call`, `planning`, `thinking_mode`, etc.),
- a visibility mode (`normal` vs `debug`) so the UI can hide low-signal entities in non-debug mode.

This solves a common problem in streaming agent UIs: “the backend can emit many events; the user should see a curated subset by default.”

## 6. SEM event routing on the frontend (Moments)

Moments routes WebSocket events using a registry-first strategy:
- Registry: `moments/web/src/platform/sem/registry.ts`
- Handlers: `moments/web/src/platform/sem/handlers/*`
- Call site: `moments/web/src/platform/chat/hooks/useChatStream.ts`

The WebSocket handler parses JSON frames and expects this envelope:

```json
{ "sem": true, "event": { "type": "llm.delta", "id": "...", ... } }
```

Routing algorithm (conceptually):
1. Parse JSON; ignore non-JSON frames.
2. If `{ error: "..." }`, dispatch an error message entity.
3. If `{ sem: true, event: ... }`, attempt `handleSem(...)`:
   - if a handler exists and returns a command, dispatch it (`add` or `upsert`).
   - else fall back to a legacy `switch` on `ev.type` inside `useChatStream` (covers many core types like `llm.*`, `tool.*`, and several widget families).

This “registry-first + legacy switch fallback” pattern is a **Moments implementation detail** (a migration artifact), not a desirable steady state. In go-go-mento, the primary routing mechanism is the SEM registry (`web/src/sem/registry.ts`); the only “switch” routing you’ll usually see is in mocks (e.g., `web/src/hooks/useMockChatStream.ts`).

For Pinocchio: treat the Moments switch fallback as an anti-pattern. Do not replicate it; make the registry complete.

### 6.1 Moments SEM handler catalog (what the browser knows how to do)

At a glance, Moments’ frontend SEM registry recognizes (at least) the following event families (registered in `moments/web/src/platform/sem/handlers/*`):

**Core streaming + tools**

| SEM type | Primary frontend handling | Primary effect on Redux timeline |
|---|---|---|
| `llm.start`, `llm.delta`, `llm.final` | Legacy switch in `moments/web/src/platform/chat/hooks/useChatStream.ts` (should be moved into a SEM handler) | Creates/updates a `message` entity for streaming assistant text via `appendMessageText` / `finalizeMessage`. |
| `tool.start`, `tool.delta`, `tool.result`, `tool.done` | Both `moments/web/src/platform/sem/handlers/tool.ts` and the legacy switch in `useChatStream` (duplication; legacy) | Creates/updates a `tool_call` entity (and sometimes `tool_result` entities). |

**Planning / execution (aggregated)**

| SEM type | Frontend handler | Entity kind |
|---|---|---|
| `planning.start`, `planning.iteration`, `planning.reflection`, `planning.complete` | `moments/web/src/platform/sem/handlers/planning.ts` | `planning` (an in-memory aggregate keyed by run id; re-rendered as a single evolving widget). |
| `execution.start`, `execution.complete` | `moments/web/src/platform/sem/handlers/planning.ts` | Still the same `planning` entity (execution state nested under the planning aggregate). |

**“Thinking mode” and related introspection widgets**

| SEM type prefix | Frontend handler | Entity kind |
|---|---|---|
| `thinking.mode.*` | `moments/web/src/platform/sem/handlers/thinkingMode.ts` | `thinking_mode` |
| `mode.evaluation.*` | `moments/web/src/platform/sem/handlers/modeEvaluation.ts` | `mode_evaluation` |
| `next.thinking.mode.*` | `moments/web/src/platform/sem/handlers/nextThinkingMode.ts` | `next_thinking_mode` |
| `inner.thoughts.*` | `moments/web/src/platform/sem/handlers/innerThoughts.ts` | `inner_thoughts` |

**Structured “selection” side channels (often page-consumed, not necessarily chat-rendered)**

| SEM type prefix | Frontend handler | Entity kind(s) | Notes |
|---|---|---|---|
| `document.*` | `moments/web/src/platform/sem/handlers/docLens.ts` | `doc_suggestion`, `doc_suggestion_removed`, plus `status` errors | These entities are commonly consumed by page-specific UI (e.g., document pickers). They are *not* registered with a dedicated timeline widget renderer today, so they may only show up via the “unknown entity” fallback in debug mode. |
| `team.member.*` | `moments/web/src/platform/sem/handlers/teamSelection.ts` | `team_candidate`, `team_candidate_removed`, plus `status` errors | Same pattern: timeline is used as a general conversation-scoped event store, not solely a chat transcript. |
| `question.multiple_choice.*` | `moments/web/src/platform/sem/handlers/question.ts` | `multiple_choice` | This *does* have a widget renderer (`MultipleChoiceWidget`). |

**Summaries / artifacts / memory**

| SEM type | Frontend handler | Entity kind(s) | Notes |
|---|---|---|---|
| `summary.chunk.started` | `moments/web/src/platform/sem/handlers/summaryChunk.ts` | `editable_summary` | Creates an “editable summary” entity keyed by stable item id (chunk suffix stripped). |
| `summary.chunk.completed` | Legacy switch in `moments/web/src/platform/chat/hooks/useChatStream.ts` | `editable_summary` (chunks appended) | Appends chunk text into an array field using `appendEntityArrayItem`. |
| `artifact.persisted` | Registered but handled via legacy switch | `editable_summary` updated | The handler is registered as a no-op so the event reaches the legacy switch that updates the *most recent* editable summary with `artifact_id`. |
| `memory.extraction.completed` | `moments/web/src/platform/sem/handlers/memory.ts` | `memory_validation` | Emits a validation card only when extraction succeeded and yields a persisted `memory_id`. |

**Operational / safety**

| SEM type | Frontend handler | Entity kind |
|---|---|---|
| `debug.pause` | `moments/web/src/platform/sem/handlers/debugger.ts` | `debug_pause` |
| `run.limit.exceeded` | `moments/web/src/platform/sem/handlers/runLimit.ts` | `run_limit` |

Two meta-observations for Pinocchio:
1. Not all timeline entities are meant to render in the chat transcript; some are meant to drive page-local UI state.
2. Moments still contains a “legacy switch” in `moments/web/src/platform/chat/hooks/useChatStream.ts` that duplicates handler logic for many event families (`llm.*`, `tool.*`, summary chunks, artifacts, team analysis, etc.). It exists as a migration convenience (and to paper over incomplete registry coverage), but it is not necessary in principle: a complete registry makes the switch redundant and eliminates a major source of behavioral drift.

### 6.2 Worked end-to-end trace: “send a message, stream a response, show widgets”

Here is the canonical lifecycle for a single user prompt (glossing over auth and UI shell differences).

**(A) Browser: user sends a message**

1. User hits Enter in the SidebarChat input.
2. `useSidebarChat.sendMessage`:
   - immediately dispatches a `message` entity for the user (optimistic UI),
   - enqueues a send request into `chatQueueSlice` (Moments: retries transient conflicts like `409 Conflict`),
   - relies on WebSocket events for assistant streaming + widgets.

Pinocchio port note: do **not** reproduce the `chatQueueSlice` retry queue; push those semantics into the backend and keep the UI’s send path “thin.”

**(B) Backend: start run + attach sink**

3. HTTP handler `POST /rpc/v1/chat` (or `/rpc/v1/chat/{profile}`) creates/gets the conversation.
4. A goroutine starts the run:
   - builds `runCtx`,
   - does `runCtx = events.WithEventSinks(runCtx, conv.Sink)`,
   - runs the ToolCallingLoop (Geppetto engine + middleware + tools).

**(C) Backend: events emitted**

5. The engine emits:
   - `EventPartialCompletionStart` → SEM `llm.start`
   - repeated `EventPartialCompletion` → SEM `llm.delta` (delta + optional cumulative)
   - tool lifecycle events (`EventToolCall`, `EventToolResult`, etc.) → SEM `tool.*`
6. If the model outputs tagged structured blocks, the **FilteringSink extractors** emit typed events too (e.g., multiple-choice, thinking mode, summary chunks). Those typed events are also translated into SEM frames.

**(D) Transport + WebSocket**

7. Events flow through the event transport (Redis Streams or event router), are read by the conversation reader, and are translated into SEM frames.
8. The WebSocket broadcasts each SEM frame to every connected browser for that `conv_id`.

**(E) Browser: SEM → Redux → widgets**

9. `useChatStream` receives frames:
   - routes them through `handleSem` registry,
   - dispatches `addEntity`/`upsertEntity` to the timeline slice,
   - and the `Timeline` component re-renders the updated entity set.

The key invariant is that all updates are stable-ID upserts: the UI is continuously refining a small set of entities as new evidence (events) arrives.

---

# Part II — Backend architecture

## 7. Where events come from: Geppetto and the run context

Both go-go-mento and Moments are built around Geppetto:
- A run is started (via HTTP `/rpc/v1/chat` or `/rpc/v1/chat/{profile}`).
- A conversation has a **sink** (`events.EventSink`) attached.
- The run context is constructed with that sink:
  - go-go-mento: `go-go-mento/go/pkg/webchat/router.go` uses `events.WithEventSinks(runCtx, conv.Sink)`
  - moments: `moments/backend/pkg/webchat/router.go` does the same.

The essential idea:

> Geppetto (engine + middleware + tools) emits `events.Event` instances.  
> Those events are published to the sinks present in the run context.  
> The sink pipeline transports them to any web clients and (optionally) persistence.

## 8. Sink pipelines: WatermillSink + structuredsink FilteringSink + (avoid) stateful accumulator sinks

### 8.1 WatermillSink (transport)

Both systems use a Watermill-backed sink as the “base sink” for webchat:
- It publishes events to a per-conversation topic, typically `chat:{conv_id}`.
- That topic is backed by Redis Streams when configured; Moments can also run in-memory using `events.EventRouter`.

### 8.2 structuredsink FilteringSink (structured event extraction)

The phrase “SEM events are routed and extracted from sinks” refers to the following mechanism:

1. The LLM response may include **tagged blocks** like `<mento:team_member:v1> ... </mento:team_member:v1>`, `<thinking:mode:v1> ...`, `<question:multiple_choice:v1> ...`, `<mento:summary_chunk:v1> ...`.
2. A structuredsink **Extractor** recognizes tags (`TagPackage`, `TagType`, `TagVersion`) and builds an ExtractorSession per `item_id`.
3. The session:
   - parses the block incrementally (often as YAML using a debounced snapshot parser),
   - emits typed events like `EventTeamMemberAdded`, `EventThinkingModeUpdate`, `EventMultipleChoiceCompleted`, `EventSummaryChunkCompleted`, etc.

Examples in Moments backend:
- Thinking mode extractors: `moments/backend/pkg/inference/middleware/thinkingmode/extractor.go`
  - tags are `thinking:mode:v1`, `thinking:mode_evaluation:v1`, `thinking:next_thinking_mode:v1`, `thinking:inner_thoughts:v1`
- Team member extractor: `moments/backend/pkg/inference/middleware/teamselection/extractor.go`
  - tags are `mento:team_member:v1`, plus removal events
  - on completion, it resolves identity (person IDs) using the conversation’s session
- Multiple-choice extractor: `moments/backend/pkg/inference/middleware/question/extractor.go`
  - tags are `question:multiple_choice:v1`
- Summary chunk extractor: `moments/backend/pkg/inference/middleware/summary/chunk_extractor.go`
  - tags are `mento:summary_chunk:v1`

### 8.3 “Accumulator” sinks (existing pattern; avoid mutating conversation state from sinks)

Both systems have historically used a pattern where a sink wrapper updates derived conversation fields (e.g., `conv.Turn.Data`) as typed events are observed:
- Team suggestions: `Turn.Data["mento.team.suggestions"]`
  - go-go-mento: `go-go-mento/go/pkg/webchat/suggestions_sink.go`
  - moments: `moments/backend/pkg/teamchat/team_suggestions_sink.go`
- Doc suggestions: `Turn.Data["mento.docs.suggestions"]`
  - go-go-mento: `go-go-mento/go/pkg/webchat/doc_suggestions_sink.go`
  - moments: `moments/backend/pkg/doclens/doc_suggestions_sink.go`

This works, but it is a **design smell**: sinks exist to *transport and enrich events*, not to be “hidden writers” of authoritative conversation state. When sinks mutate conversation state, you create hard-to-reason-about coupling:
- ordering bugs (a state read in middleware depends on which sink wrappers are configured),
- implicit dependencies (tools/middleware now depend on sink configuration),
- and poor testability (you need sink pipelines active to reproduce state).

Preferred pattern (especially for Pinocchio):
- Keep sinks **pure** (observe/emit), and update conversation state in one of:
  - explicit middleware (single-owner write path),
  - the engine loop (single-threaded coordinator),
  - or a dedicated “projection” component that consumes events and owns derived state (not the sink).

## 9. Event → SEM translation on the backend (SEM registry)

### 9.1 The SEM envelope

Both backends ultimately produce frames like:

```json
{ "sem": true, "event": { "type": "tool.start", "id": "...", ... } }
```

The translator function is called `SemanticEventsFromEvent` in both codebases:
- go-go-mento: `go-go-mento/go/pkg/webchat/event_translator.go`
- moments: `moments/backend/pkg/webchat/forwarder.go`

### 9.2 How protobuf ends up “inside” SEM frames (protobuf-shaped JSON payloads)

The transport format for SEM over WebSocket is **JSON**, but many SEM payloads are authored as protobuf messages and then converted to JSON at the boundary.

Concretely (go-go-mento backend):
1) A typed Geppetto/mento event is handled by a SEM handler.
2) The handler constructs a protobuf message for the SEM payload (e.g., planning events):
   - `go-go-mento/go/pkg/webchat/handlers/planning.go` builds `semMw.PlanningStarted`, `semMw.PlanningIteration`, etc.
3) The protobuf message is converted into a `map[string]any` using `protojson`:
   - `go-go-mento/go/pkg/webchat/handlers/helpers.go` implements `pbToMap(msg proto.Message)` as `protojson.Marshal` followed by `json.Unmarshal`.
4) That map is assigned to the SEM frame’s `data` field and wrapped in `{ sem: true, event: ... }`.

On the frontend, the corresponding handlers treat `ev.data` as **protobuf-shaped JSON** and rehydrate it using the generated schema:
- go-go-mento example: `go-go-mento/web/src/sem/handlers/planning.ts` uses `@bufbuild/protobuf` `fromJson(PlanningStartedSchema, ev.data as JsonObject)`.

This is a useful mental model:
- **protobuf is the schema and authoring format**, and
- **JSON is the wire format**.

In Moments, the Go backend SEM handlers (`moments/backend/pkg/sem/handlers/**`) generally construct JSON maps directly rather than assembling protobuf messages. The frontend still uses the same `fromJson(schema, ev.data)` pattern for many event families; that works as long as Moments emits JSON that matches the schema the UI expects.

### 9.3 Registry-first translation

Both backends route events through a **type-based registry**:

- go-go-mento: `go-go-mento/go/pkg/sem/registry/registry.go`
  - supports type-based handlers and message-based handlers for `events.EventInfo`
  - includes panic recovery and simple hit/miss/error metrics
- moments: `moments/backend/pkg/sem/registry/registry.go`
  - simpler type-based handler list per event type

In both cases, the registry defines the “compiler” from Go event types to SEM frames.

### 9.4 Handler registration patterns (Moments)

Moments distributes SEM handler registration across feature packages:
- Core: `moments/backend/pkg/sem/handlers/handlers.go` registers LLM/tool/log/runlimit/inference handlers.
- Analytics: `moments/backend/pkg/analytics/sem_handlers.go` registers summary chunk and artifact persisted frames.
- Doclens/teamchat may register additional SEM handlers for their typed events.

Examples:
- LLM streaming events map:
  - `events.EventPartialCompletionStart` → `llm.start`
  - `events.EventPartialCompletion` → `llm.delta` (delta + cumulative)
  - `events.EventFinal`/`events.EventInterrupt` → `llm.final`
  - Implemented in `moments/backend/pkg/sem/handlers/llm_handlers.go`
- Tool events map:
  - `events.EventToolCall` → `tool.start`
  - `events.EventToolResult` → `tool.result` + `tool.done`
  - Implemented in `moments/backend/pkg/sem/handlers/tool_handlers.go`
  - Note the use of an in-memory cache to enrich `tool.result` with `tool_name` and parsed `input`.

### 9.4 Handler registration patterns (go-go-mento)

go-go-mento registers a webchat-focused set of SEM handlers in `go-go-mento/go/pkg/webchat/event_translator.go` via `init()`:

```go
handlers.RegisterTeamAnalysisHandlers()
handlers.RegisterPlanningHandlers()
handlers.RegisterThinkingModeHandlers()
...
```

The net effect is the same: when the stream coordinator reads an event from the event transport, it can translate it into one or more SEM frames.

## 10. Transport and streaming: Redis streams and/or in-memory event routing

### 10.1 go-go-mento: Redis Streams + consumer groups + StreamCoordinator

go-go-mento’s backend uses a dedicated `StreamCoordinator` and a `ConnectionPool`:
- `go-go-mento/go/pkg/webchat/stream_coordinator.go`
- `go-go-mento/go/pkg/webchat/connection_pool.go`

Pipeline:
1. Inference publishes Geppetto events to the sink (ultimately Watermill publisher).
2. Events appear on a Redis Stream (`chat:{conv_id}`).
3. `StreamCoordinator.consume()` reads Watermill messages and:
   - decodes the event,
   - optionally records a monotonic version derived from the Redis XID,
   - calls `onEvent(event)` (timeline hydration path),
   - translates event to SEM frames,
   - calls `onFrame(event, frame)` to broadcast to WebSocket clients.

This design is documented in `go-go-mento/docs/reference/webchat/backend-internals.md`.

### 10.2 Moments: EventRouter (Redis or in-memory) + per-conversation reader

Moments backend creates an `events.EventRouter` in `moments/backend/pkg/webchat/router.go`:
- It attempts Redis pub/sub via appconfig; otherwise falls back to an in-memory gochannel router.
- Each conversation has a `startReader` goroutine (`moments/backend/pkg/webchat/conversation.go`) that:
  - subscribes to `chat:{conv_id}`,
  - decodes events,
  - calls `SemanticEventsFromEvent(e)` to get frames,
  - writes them to connected WebSocket clients.

The underlying principle is the same; the structuring is different (go-go-mento isolates streaming into coordinator/pool types, Moments does it directly inside `Conversation`).

## 11. WebSocket endpoints, auth, and keepalive

### 11.1 Endpoint shape

Frontend connects to:

```
GET /rpc/v1/chat/ws?conv_id=<id>[&profile=<slug>][&draft_bundle_id=<uuid>][&access_token=<jwt>]
```

Moments client builds this URL in `moments/web/src/platform/chat/hooks/useChatStream.ts`.

Moments backend accepts an `access_token` query parameter and mirrors it into the `Authorization` header if the header is absent (so the websocket can build a session despite browser limitations on WS headers): `moments/backend/pkg/webchat/router.go`.

### 11.2 Keepalive

Both codebases send periodic pings from the browser to avoid idle timeouts in load balancers:
- Moments client: sends `'ping'` every ~30s (`PING_INTERVAL_MS = 30_000`) in `useChatStream`.
- Moments backend: if it reads `"ping"` on the websocket read loop, it writes `"pong"` back (non-JSON); this resets idle timers.
- go-go-mento does something similar but uses SEM-style `ws.hello` / `ws.pong` frames and supports more ping formats.

### 11.3 Concurrency constraints (important for correctness)

Gorilla WebSocket’s rule of thumb is: **do not concurrently call write methods on the same connection from multiple goroutines**. The systems handle this differently:
- go-go-mento serializes broadcasts via a single `ConnectionPool` mutex, so concurrent broadcasters are safe.
- Moments uses a `connsMu` and carefully locks around `"pong"` writes; correctness depends on only a single broadcaster goroutine doing the streaming writes.

If you port this to Pinocchio, treat “single-writer” as a first-class invariant: a connection should have exactly one “write serialization mechanism.”

---

# Part III — Cross-cutting contracts and invariants

## 12. The SEM protocol as a contract

The most important “pin” to preserve for compatibility is the SEM envelope:

```json
{ "sem": true, "event": { "type": "<dotted>", "id": "<string>", ... } }
```

Required fields:
- `event.type`: a dotted event name (`llm.delta`, `tool.start`, `thinking.mode.update`, etc.).
- `event.id`: a stable identifier that the UI uses as an entity id or as an item id.

Optional fields depend on the event family:
- LLM: `delta`, `cumulative`, `text`, `metadata`
- Tool: `name`, `input`, `patch`, `result`, plus optional derived label/summaries
- Structured widgets: `data` containing an `item_id` or `itemId` and payload snapshots

## 13. Upsert semantics and idempotence

The frontend prefers to treat the stream as potentially “replayed”:
- A handler should be idempotent when applied multiple times.
- “Add” should be used when creating a brand new entity once; “upsert” is used for updates.
- Array accumulation (e.g., summary chunks) uses explicit append operations rather than repeated full replacements.

This is what makes it possible (in go-go-mento) to mix WebSocket streaming with DB hydration snapshots without visible duplication.

## 14. Debug affordances

The architecture intentionally supports a set of “developer affordances”:
- **Debug visibility mode**: hide noisy entity kinds in normal mode.
- **WS debug toggles** (go-go-mento): query param `?ws_debug=1`, global `window.__WS_DEBUG__`, localStorage `__WS_DEBUG__`.
- **Step debugger**: endpoints exist in both systems (`/rpc/v1/chat/debug/step-mode`, `/rpc/v1/chat/debug/continue`) and produce `debug_pause` events that the UI renders.

For Pinocchio parity, decide which affordances you want; most are composable and do not require deep product coupling.

---

# Part IV — Practical guidance for Pinocchio porting

## 15. What to copy (high leverage)

If the goal is “Pinocchio webchat with Moments parity,” the high-leverage components are:

Frontend:
- The timeline entity model (`timelineSlice`) and widget registry design.
- The SEM handler registry pattern (**registry-only**; do not ship a switch fallback).
- go-go-mento’s singleton WS manager + hydration gating if you need production hardening and reload persistence.
- A **single “all-in-one” `ChatWidget` root component** that can be embedded anywhere, with internal layout composition but one consistent external contract (props, store wiring, and capabilities).

Backend:
- The sink pipeline pattern: Watermill sink + structuredsink FilteringSink extractors (and keep sinks pure; avoid sink-owned conversation state).
- The SEM registry mapping event types → SEM frames (and the discipline to keep it the only source of truth).
- A single-writer broadcast mechanism for WebSockets.

### 15.1 A single consistent `ChatWidget` component (avoid “5 different chats”)

When porting to React, it is tempting to mirror every host context as a distinct component tree (“sidebar chat,” “modal chat,” “debug chat,” “inline chat,” etc.). This is a trap: you end up with five subtly different event-handling behaviors, five sets of keyboard shortcuts, and five places where “what is the current conversation?” is computed differently.

Instead, treat webchat as **one product component** with a stable, minimal external interface:
- `ChatWidget` is the *only* integration surface.
- Everything else (timeline, composer, tool widgets, debug overlays) is internal implementation detail.

Concretely:
- `ChatWidget` owns the **WS lifecycle** (connect, reconnect, keepalive, disconnect), either directly or by delegating to a singleton `wsManager`.
- `ChatWidget` owns the **store wiring**:
  - it ensures the RTK store exists,
  - it registers SEM handlers once,
  - and it feeds incoming SEM frames into `timelineSlice`.
- `ChatWidget` is parameterized by “host needs” via props rather than forked component trees.

Recommended external API shape (illustrative, not prescriptive):
- `conversationKey?: string` (or `conversationId?: string`) — which timeline to show/hydrate.
- `profile?: string` — which backend profile/run mode to use (if relevant).
- `variant?: "sidebar" | "modal" | "full"` — affects layout only (not semantics).
- `capabilities?: { debug?: boolean; }` — controls affordances, not logic.
- `onEvent?: (semEvent) => void` — optional integration hook for logging/analytics (never drives core state).

This is how you keep semantics stable while allowing multiple presentations.

### 15.2 Pinocchio port principles (explicitly no fallback switch, no backwards compatibility)

The Moments/go-go-mento lineage contains a number of “migration crutches” (legacy switches, duplicate handlers, legacy alias fields) that helped the systems evolve without breaking in-flight behavior. For Pinocchio, you have the advantage of starting a fresh integration: take the *contracts* and *architecture*, but do not import the crutches.

Concretely:

1) **No frontend switch fallback**
- Do not implement `switch (ev.type)` as a secondary routing mechanism.
- All SEM events must route through a single registry (`registerSem` / `handleSem`).
- Unknown event types should be treated as an explicit error or a single generic “unknown widget” path (for debug), not as “silently ignored unless a switch handles it.”

2) **No backwards-compatibility payload aliases**
- Do not send “legacy alias” fields like `analysis_id` or redundant keys “just in case.”
- Choose one canonical schema per event family and stick to it.
- If you use protobuf schemas, treat protojson field names (camelCase) as the contract.

3) **No frontend `chatQueueSlice` conflict/retry semantics**
- The “run in progress / 409 retry/backoff” behavior belongs on the backend: the server is the authority on concurrency and queuing.
- The UI can still be optimistic about rendering the user message, but it should not implement a hidden retry queue to achieve correctness.

4) **Do adopt the WS manager + hydration gating**
- Keep the singleton WS manager pattern (one socket per conversation/session, generation-counters for React StrictMode).
- Keep hydration gating so timeline snapshots and WS deltas combine without duplication.

5) **Do not use sinks to update conversation state**
- Treat “sink mutates `Turn.Data`” as a design smell; keep sinks pure and move derived state updates into middleware/engine/projection components.

## 16. Common pitfalls (and how to avoid them)

1) **Entity IDs drift** → duplicated widgets  
   Fix: define stable entity IDs at the source (backend) and keep handler logic pure.

2) **Concurrent websocket writes** → intermittent corruption/panics  
   Fix: enforce one write serialization mechanism.

3) **StrictMode double mount creates duplicate sockets** (React dev)  
   Fix: use generation counters + ref counting as in go-go-mento `wsManager`, or ensure only one hook instance can connect.

4) **Missing hydration** → reload loses chat state  
   Fix: adopt go-go-mento’s timeline hydration (DB snapshots + `sinceVersion`), or explicitly accept in-memory-only behavior.

## 17. A minimal “port checklist”

- Frontend:
  - [ ] Implement `TimelineEntity` types and a normalized timeline slice.
  - [ ] Implement SEM registry and core handlers for `llm.*` and `tool.*`.
  - [ ] Implement a single `ChatWidget` root component (layout variants via props; semantics shared).
  - [ ] Implement timeline rendering using a widget registry (`Timeline` inside `ChatWidget`).
  - [ ] Implement WebSocket connect + keepalive + error surfacing.
- Backend:
  - [ ] Provide `/rpc/v1/chat` + `/rpc/v1/chat/{profile}` and `/rpc/v1/chat/ws`.
  - [ ] Serialize/queue “send message” semantics server-side (no frontend `chatQueueSlice`).
  - [ ] Make message submission idempotent (dedupe key) and return stable IDs early.
  - [ ] Attach `events.WithEventSinks(ctx, conv.Sink)` for runs.
  - [ ] Wrap base sink with FilteringSink extractors needed for rich widgets.
  - [ ] Register SEM handlers for all event types you intend to show.
  - [ ] Ensure safe websocket broadcasting (single writer).

---

# Part V — How go-go-mento evolved into Moments (migration timeline + architectural deltas)

This section is intentionally “historical and comparative”: it is less about what the system is *now*, and more about how the current Moments implementation acquired its shape, and which parts are inherited versus genuinely new.

The core claim (useful for Pinocchio porting) is:
- **go-go-mento** is the older integrated system where the webchat architecture (SEM stream → widget timeline) was exercised heavily and documented.
- **Moments** is the successor system that **ported** that UI and then **ported + refactored** the Go backend webchat stack into the `moments-server` architecture, with additional integrations and extensibility work.

In practice, this evolution happened as a combination of:
- Straight code copy (large “pull over web” commits),
- A staged backend port (webchat package + router + config),
- Subsequent hardening/refactors (SEM registry, extractors, feature gating),
- And packaging/organizational refactors (frontend “platform” modules).

## 18. Lineage model: not a git-merge, but a porting lineage (with deliberate “phase gates”)

It is important to understand that “go-go-mento → Moments” is not necessarily a clean git ancestor/descendant relationship (i.e., not a simple fork with a merge base). Instead, the history strongly suggests a **porting lineage**:

1) There existed a working UI architecture (SidebarChat + timeline widgets + SEM handlers) and a working backend event-to-SEM forwarding architecture (sinks, translators/forwarders, and connection management).

2) Moments then adopted that architecture via two migration tracks:
- **Frontend track** (port web UI and preserve semantics).
- **Backend track** (port webchat backend into `moments/backend/pkg/webchat`, integrate with Moments server/router/config).

3) Only after parity was achieved did Moments undertake structural refactors:
- Moving web chat building blocks into a `platform/` layer (frontend),
- Moving SEM handler logic into a dedicated package and switching to a registry-first mapping model (backend).

This is exactly the migration strategy you would use if you cared about “keeping the UI usable” while you rewired backend internals: **copy first, refactor second**.

## 19. Frontend evolution: “SidebarChat + Timeline” preserved, then platformized

### 19.1 go-go-mento establishes the SidebarChat + timeline pattern

In go-go-mento, the SidebarChat appears as an explicit component introduction:
- `2025-10-15` — `655175867`: “Create initial sidebar chat component” (adds `web/src/components/SidebarChat/SidebarChat.tsx` and related story/README). The same commit modifies timeline widgets (`MessageWidget`, `Timeline`, `ToolCallWidget`, `ToolResultWidget`), which is a strong signature of the “sidebar chat renders a timeline of widgets” architecture.

You can read this as: SidebarChat is not “a chat UI” so much as “a container around a timeline renderer + a composer,” where most of the sophistication lives in the timeline entity registry and SEM handler mapping.

### 19.2 Moments imports the web UI and preserves the surface area

In Moments, the earliest “web UI import” event is a single massive commit:
- `2025-11-13` — `f24d0669`: “First cut pull over web” (adds `web/…` in bulk and introduces docmgr-managed docs, including web docs like `docs/web/event-driven-widgets.md` and a `ttmp/PORT-WEB…` ticket).

This commit is significant because it is exactly the “copy first” phase:
- SidebarChat is present under `web/src/components/SidebarChat/SidebarChat.tsx`.
- Timeline widgets and SEM handler code are present as well.
- The build toolchain is preserved (Vite, Storybook, tests), consistent with the `PORT-WEB` ticket’s written plan.

In other words: Moments did not “invent a new UI.” It imported an existing one, then began adapting it to the new repo and backend endpoints.

### 19.3 Moments later refactors the frontend into a “platform” layer

After the initial import and subsequent iteration, Moments performs a structural refactor:
- `2025-12-09` — `9281c31b`: “Move platform pieces”

This commit renames/moves the chat pieces into a platform namespace:
- `web/src/components/SidebarChat/SidebarChat.tsx` → `web/src/platform/chat/SidebarChat/SidebarChat.tsx`
- `web/src/hooks/useChatStream.ts` → `web/src/platform/chat/hooks/useChatStream.ts`
- `web/src/store/chatQueue/chatQueueSlice.ts` → `web/src/platform/chat/state/chatQueueSlice.ts`
- Timeline entities and widgets similarly move under `web/src/platform/timeline/**`

This is an “engineering-maturity” move: once you believe the chat UI is a reusable platform feature rather than a page-local toy, you extract it into a platform layer that can be imported by many pages without duplicating semantics.

## 20. Backend evolution: port webchat into Moments server, then refactor SEM mapping and extraction

### 20.0 go-go-mento hardens the streaming backend before the Moments port

In go-go-mento, you can see the backend streaming layer being actively hardened and reorganized in the same time window as the Moments backend porting work begins:

- `2025-11-18` — `b26741ddc`: “Implement StreamCoordinator and rewrite EventTranslator”
  - Renames the webchat “forwarder” into a more explicit translator (`go/pkg/webchat/forwarder.go` → `go/pkg/webchat/event_translator.go`).
  - Updates `go/pkg/webchat/stream_coordinator.go`, which is part of the “single-writer / coordinated fan-out” story (i.e., prevent concurrent websocket writes and centralize broadcast semantics).
  - Updates the web SEM proto bindings (`web/src/sem/pb/proto/...`) in the same commit, reinforcing the idea that backend translation and frontend handlers are a coupled protocol evolution.

Even if the exact types differ in Moments, the *shape* of the work is the same: once you are beyond toy streaming, you centralize translation and coordination so the UI receives a clean, stable semantic stream.

### 20.1 Moments ports the webchat backend as a new package

The first major backend port lands here:
- `2025-11-14` — `96dca9c1`: “Progress towards porting webchat”

This commit adds (among others):
- `backend/pkg/webchat/{conversation,engine,forwarder,loops,router,settings,step_controller,types}.go`
- A dedicated migration ticket workspace: `ttmp/MIGRATE_CHAT-migrate-webchat-into-moments/**`
- Router/server wiring changes (`backend/cmd/moments-server/serve.go`, config, and appconfig registrations)

The accompanying design doc in the `MIGRATE_CHAT` ticket spells out the intended strategy:
- Mirror a known endpoint surface (`/rpc/v1/chat`, `/rpc/v1/chat/ws`, debug endpoints),
- Prefer HTTP for “start chat” and WS for streaming events,
- And feature-gate optional dependencies so Phase 1 can ship without pulling in every subsystem.

This is the backend analogue of “copy first”: make the package exist, wire it into the server, and preserve semantics.

### 20.2 Moments ports middleware + structured event emitters to reach parity

The next major port focuses on missing “middleware and structured event emitters”:
- `2025-11-14` — `0bbe7b0a`: “Pull a set of middlewares over”

This commit adds multiple webchat middlewares and sinks, notably:
- `backend/pkg/webchat/sink_wrapper.go` (sink wrapping is where extractor plumbing typically lives)
- `backend/pkg/webchat/*_middleware.go` and multiple `*_sink.go` files
- A dedicated ticket workspace: `ttmp/IMPORT-MIDDLEWARE-port-middleware-and-structured-event-emitters-from-mento-playground-go-pkg-webchat-into-webchat/**`

Conceptually, this phase is “teach Moments webchat to emit the same *rich* events the UI expects,” which is where most ports fail if you treat streaming as “just text deltas.”

### 20.3 Moments hardens structured extraction and brings over missing extractor paths

One of the most telling “port pains” is captured in:
- `2025-11-18` — `fa93d59f`: “Fix structured event extraction”

This commit adds explicit extractor code and adjusts sink wrapping/forwarding:
- `backend/pkg/inference/middleware/teamselection/extractor.go` and related files
- updates to `backend/pkg/webchat/sink_wrapper.go` and `backend/pkg/webchat/forwarder.go`
- a focused ticket workspace documenting the extractor port: `ttmp/2025/11/18/SEM-EXTRACTORS-port-structured-block-extractors-for-team-member-parsing/**`

This is exactly the kind of migration step you should expect when going from “basic streaming chat” to “event-driven widgets”: you discover missing extractors, then you implement them as FilteringSink wrappers and ensure the forwarder emits SEM events with stable IDs.

### 20.4 Moments refactors SEM mapping into a dedicated registry/handler package

After parity and incremental fixes, Moments does the deeper refactor:
- `2025-12-09` — `c22124f2`: “Refactor sem forwarder/handler into registry pattern”
- `2025-12-10` — `9388e8ea`: “Remove deprecated init registration on handlers”

This change is architecturally meaningful:
- SEM handler logic moves out of ad-hoc webchat handler files into `backend/pkg/sem/handlers/**`.
- The forwarder becomes “registry-first,” which reduces the risk of drift as new event types are added.

If you compare this to go-go-mento’s evolution, you see the same storyline: early versions often have a “switch statement” translator; mature versions converge on a registry pattern to keep the mapping centralized and testable.

### 20.5 Ongoing renames and semantic cleanup (example: RunID → SessionID)

Finally, you see Moments-specific product semantics and API cleanup:
- `2026-01-22` — `6bb64356`: “moments/backend: rename RunID to SessionID”

These are not merely cosmetic: they reflect that the webchat system is being integrated into a broader product, and its identifiers must align with product-level semantics (session, identity, routing).

## 21. Comparative summary: what is inherited vs what is new in Moments

Inherited (conceptual invariants that survived the migration):
- **SEM stream as the lingua franca** between backend and UI: the UI reacts to semantic event frames, not to “raw logs.”
- **Timeline-as-state**: the UI’s durable model is a normalized set of entities keyed by stable IDs; widgets are pure renderers.
- **Extractor sinks**: complex widgets require deriving structured events from raw logs/blocks, and the right place to do that is the sink pipeline.
- **“Copy then refactor” discipline**: preserve semantics first, then reorganize code (platform layer; registry patterns).

New/extended in Moments (signals of a system being productized):
- **Server integration**: webchat routes integrated into `moments-server` and identity/session enforcement.
- **Appconfig-based feature gates**: the backend is designed to run with optional dependencies (Redis vs in-memory, doclens/workflows integrations).
- **Dedicated SEM handler package**: `backend/pkg/sem/handlers/**` becomes a cross-cutting layer usable beyond webchat.
- **Frontend “platform” namespace**: chat/timeline become reusable platform modules.

## 22. Why this history matters for Pinocchio

If you want “Moments-class webchat affordances” in Pinocchio, the migration history suggests a strategy that has already worked once:

1) Treat go-go-mento as a reference implementation of *behavior and contracts* (SEM envelopes, entity IDs, widget semantics, debug affordances).
2) Treat Moments as the reference implementation of *how to integrate and scale it inside a larger product* (server wiring, appconfig gating, registry refactors, platformization).
3) Port in phases: copy/establish semantics first, then refactor for Pinocchio’s product needs.

# Appendix — “Existing documentation” you should read first

If you are going to implement or significantly refactor Pinocchio’s webchat, the following in-repo documents are already close to what you want; this document should be read as a unifying commentary plus Moments-specific deltas:

- go-go-mento frontend integration: `go-go-mento/docs/reference/webchat/frontend-integration.md`
- go-go-mento SEM + widget catalog: `go-go-mento/docs/reference/webchat/sem-and-widgets.md`
- go-go-mento backend internals (StreamCoordinator/ConnectionPool): `go-go-mento/docs/reference/webchat/backend-internals.md`
- go-go-mento EngineBuilder: `go-go-mento/docs/reference/webchat/engine-builder.md`
- go-go-mento timeline hydration: `go-go-mento/docs/reference/persistence/timeline-hydration.md`
- Moments migration tickets:
  - `moments/ttmp/PORT-WEB-port-mento-web-ui-into-moments/README.md`
  - `moments/ttmp/MIGRATE_CHAT-migrate-webchat-into-moments/README.md`
  - `moments/ttmp/IMPORT-MIDDLEWARE-port-middleware-and-structured-event-emitters-from-mento-playground-go-pkg-webchat-into-webchat/README.md`
  - `moments/ttmp/2025/11/18/SEM-EXTRACTORS-port-structured-block-extractors-for-team-member-parsing/README.md`
  - `moments/ttmp/2025/12/09/REFACTOR-HTTP-ROUTER--refactor-http-router-into-core-component/design-doc/02-refactor-2-webchat-sem-forwarder-registry-plan.md`
