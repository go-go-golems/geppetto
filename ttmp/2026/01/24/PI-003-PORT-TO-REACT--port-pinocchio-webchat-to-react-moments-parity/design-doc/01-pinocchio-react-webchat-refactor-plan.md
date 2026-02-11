---
Title: Pinocchio → React Webchat Refactor Plan
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-mento/web/src/sem/registry.ts
      Note: Reference SEM handler registry design
    - Path: pinocchio/cmd/web-chat/web/.storybook/main.ts
      Note: Storybook config (React+Vite)
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.stories.tsx
      Note: Storybook ChatWidget stories (ScenarioBasic)
    - Path: pinocchio/pkg/sem/registry/registry.go
      Note: Type-based SEM handler registry used by translator
    - Path: pinocchio/pkg/webchat/router.go
      Note: |-
        HTTP + WS endpoints and conversation/run wiring
        /chat now queues + returns 202; drains queue after inference
        GET /hydrate endpoint for hydration gating (commit f696ce4)
    - Path: pinocchio/pkg/webchat/sem_buffer.go
      Note: In-memory SEM frame buffer for hydration
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Registry-only SEM translator; stable IDs; protobuf shaping
    - Path: pinocchio/pkg/webchat/send_queue.go
      Note: Backend-owned send serialization queue + idempotency records
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Single-writer coordination and fan-out patterns
    - Path: pinocchio/proto/sem/base/llm.proto
      Note: Canonical protobuf schema for llm.* SEM payloads
    - Path: pinocchio/proto/sem/base/tool.proto
      Note: Canonical protobuf schema for tool.* SEM payloads (includes customKind)
ExternalSources: []
Summary: Step-by-step plan for refactoring Pinocchio’s current webchat (backend + UI) toward the Moments/go-go-mento React + Redux Toolkit + SEM streaming architecture, including Storybook workflows and explicit non-goals (no switch fallback, no legacy protocols, no sink-owned conversation state).
LastUpdated: 2026-01-24T16:43:48.402294285-05:00
WhatFor: Concrete refactor plan to move Pinocchio’s current web chat UI toward the Moments/go-go-mento React + RTK Toolkit + SEM streaming architecture, with explicit non-goals (no switch fallbacks, no backward-compat payload aliases, no sink-owned conversation state).
WhenToUse: Use when planning or implementing the Pinocchio React webchat port; treat as the step-by-step roadmap and Storybook workflow guide.
---






# Pinocchio → React Webchat Refactor Plan

## Executive Summary

Pinocchio already has a working webchat backend (`pinocchio/pkg/webchat/**`) that emits SEM frames and a lightweight Preact timeline UI (`pinocchio/cmd/web-chat/web/**`). Moments/go-go-mento represent the “mature” version of this architecture: React + Redux Toolkit timeline entities, a SEM handler registry that produces normalized widget entities, a singleton WS manager with hydration gating, and rich widgets driven by structured extraction.

This document proposes a refactor path that:
- Adopts a **protobuf-first SEM schema** for *all* event families, compiled to **Go + TypeScript** (Buf), so the UI and backend share one canonical contract.
- Establishes **one canonical SEM contract** (no dual protocols, no fallback switches).
- Moves “busy / run-in-progress / queue semantics” **to the backend** (not a client retry queue).
- Treats sink-driven conversation state mutation as a **smell** (state should be owned by middleware/engine/projection components).
- Builds a **single reusable `ChatWidget`** and a Storybook workflow to iterate on individual widgets and full end-to-end scenarios.

## Problem Statement

Pinocchio’s current web UI is functional, but it is not set up for “Moments-class” affordances:
- The UI is not React/RTK-based; it is a small Preact app using a Zustand store (`pinocchio/cmd/web-chat/web/src/store.js`).
- The backend SEM mapping is implemented in a registry-only translator (`pinocchio/pkg/webchat/sem_translator.go`) and emits SEM frames with protobuf-authored `event.data` payloads (protojson). The legacy TL envelope (`{ tl: true, ... }`) has been removed.
- Client-side behavior (send path, streaming behavior, widget coverage) will drift unless we adopt the registry-and-widget architecture that Moments stabilizes.

We want to seriously improve Pinocchio webchat by converging on the Moments/go-go-mento architecture:
- React UI with a normalized timeline entity store and a widget registry.
- WebSocket streaming with a single WS manager and hydration gating.
- Backend extraction + semantic events that drive widgets (not ad-hoc UI state).
- A first-class Storybook workflow so widget work can proceed without running the whole product.
- A protobuf-first SEM contract compiled to Go+TS so “what events exist” is explicit, versionable, and type-checked across the boundary.

## Prerequisites (core concepts you must understand)

The implementation will go faster (and with fewer “mysterious duplicates”) if the implementor has a clear mental model of these fundamental concepts.

### Concept 1: “Event-sourced UI” vs “render-as-you-go”

Fundamental idea:
- The backend streams **semantic events** (SEM) and the frontend treats them as a protocol.
- The UI is not “rendering a chat transcript”; it is materializing and updating a **set of entities** (widgets) over time.

Implications:
- **Stable IDs are non-negotiable.** Every SEM event that mutates an entity must carry an ID that the UI uses consistently.
- Handlers must be **idempotent** (replay/hydration should not duplicate widgets).

### Concept 2: Normalized entity state + renderer registry (React/Redux Toolkit)

The frontend should keep one canonical state model:
- `timelineSlice` stores `byId` and `order` per conversation, and updates via `addEntity` / `upsertEntity` / append operations.
- A widget registry maps `entity.kind` → renderer component (pure view).

Implications:
- UI correctness depends on handler purity and stable IDs, not on component local state.
- Debug mode is “data visibility”: render entities, don’t hide them in control flow.

### Concept 3: WebSocket lifecycle + hydration gating

WebSocket is for streaming deltas, but reloads need hydration:
- Hydrate a snapshot (timeline entities + version) first.
- Then connect WS (or connect and gate application of deltas until hydration completes).

Implications:
- Without hydration gating you will see duplicates, race conditions, or “missing early events”.
- Use a singleton WS manager to avoid React StrictMode double-connect bugs.

### Concept 4: Protobuf-first SEM payloads (protojson boundary)

In go-go-mento, protobuf is the schema/authoring format and JSON is the wire format:
- Go authoring: create protobuf messages → `protojson.Marshal` → `data` map.
- TS consuming: parse `ev.data` using `fromJson(MessageSchema, ev.data)`.

Implications:
- “Adding a new SEM event” means updating `.proto`, regenerating Go+TS, and implementing both sides.
- Parse failures are contract violations; treat them as first-class errors during development.

### Concept 5: Event transport vs state ownership (don’t let sinks own conversation state)

Sinks and sink wrappers are for transport and extraction. If a component needs derived state:
- own the state in middleware/engine/projection components,
- do not rely on “sink writes into `Turn.Data`” as an implicit side-effect.

Implications:
- It must be possible to explain “where state comes from” without referencing sink ordering.

## Documents to read (in recommended order)

This plan assumes the implementor has read the following. These are the fastest way to “download” the architecture into your head:

1) Ticket architecture analysis (this ticket):
- `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/analysis/01-moments-react-chat-widget-architecture.md`

2) go-go-mento “prior art” docs (high-signal):
- `go-go-mento/docs/reference/webchat/frontend-integration.md`
- `go-go-mento/docs/reference/webchat/sem-and-widgets.md`
- `go-go-mento/docs/reference/webchat/backend-internals.md`
- `go-go-mento/docs/reference/webchat/engine-builder.md`

3) Moments web docs (practical implementation patterns):
- `moments/web/docs/event-driven-widgets.md` (typed widgets + protobuf + buf workflow)
- `moments/web/docs/component-development.md` (Storybook and component conventions)

4) Pinocchio existing webchat notes (useful to understand current baseline and pitfalls):
- `pinocchio/ttmp/2025-08-22/02-backend-semantic-event-mapping.md`
- `pinocchio/cmd/web-chat/README.md`

4.1) Pinocchio SEM protobuf toolchain (the “source of truth” for payload schemas):
- `pinocchio/buf.yaml`, `pinocchio/buf.gen.yaml`
- `pinocchio/proto/sem/**`
- Generated outputs:
  - `pinocchio/pkg/sem/pb/proto/sem/**` (Go)
  - `pinocchio/web/src/sem/pb/proto/sem/**` (TS; used by the main `pinocchio/web` app)
  - `pinocchio/cmd/web-chat/web/src/sem/pb/proto/sem/**` (TS; used by the embedded `cmd/web-chat` React app)

5) Moments migration tickets (historical context; why code is shaped the way it is):
- `moments/ttmp/PORT-WEB-port-mento-web-ui-into-moments/design/01-proposed-implementation-guide-port-web-ui-into-moments.md`
- `moments/ttmp/MIGRATE_CHAT-migrate-webchat-into-moments/design/01-architecture-plan-migrate-webchat.md`
- `moments/ttmp/IMPORT-MIDDLEWARE-port-middleware-and-structured-event-emitters-from-mento-playground-go-pkg-webchat-into-webchat/design/01-migration-plan-middleware-and-structured-event-emitters.md`

## Proposed Solution

The proposed solution is a staged refactor of both backend and frontend. “Staged” is critical: we preserve a working system while we move toward the desired architecture, but we explicitly avoid shipping long-lived compatibility crutches (switch fallbacks, duplicate protocols, redundant alias fields).

### A. Canonical contract (SEM over WebSocket + snapshot hydration, with protobuf-defined payloads)

The canonical on-the-wire contract is SEM frames:

```json
{ "sem": true, "event": { "type": "llm.delta", "id": "…", "data": {…}, "metadata": {…} } }
```

Where “`data`” is always the protojson encoding of a protobuf message defined for that SEM event type (compiled to Go and TS from the same `.proto` source). This is the model used in go-go-mento for many event families already:
- Protos live under `go-go-mento/proto/sem/**`.
- Codegen is driven by `go-go-mento/buf.yaml` + `go-go-mento/buf.gen.yaml`.
- TS handlers rehydrate via `@bufbuild/protobuf` `fromJson(MessageSchema, ev.data)` (example: `go-go-mento/web/src/sem/handlers/planning.ts`).
- Go handlers author payloads as protobuf messages and convert to JSON at the boundary (example: `go-go-mento/go/pkg/webchat/handlers/helpers.go`).

We also need snapshot hydration so reloads don’t lose context, and so the UI can safely reattach:

```text
GET /rpc/v1/chat/timeline?conversation_id=...&since_version=...
→ { entities: [...], version: N }
```

Frontend combines:
1) hydrate snapshot (if any),
2) then apply WS deltas,
3) using idempotent upsert semantics keyed by stable IDs.

### A.1 Schema toolchain: Buf generates Go + TS from a single proto source

Pinocchio should adopt the same toolchain pattern as go-go-mento:
- Source of truth: `pinocchio/proto/sem/**.proto`
- Config: `pinocchio/buf.yaml` + `pinocchio/buf.gen.yaml`
- Generated Go: `pinocchio/pkg/sem/pb/proto/sem/**` (module `github.com/go-go-golems/pinocchio`, so `option go_package = "github.com/go-go-golems/pinocchio/pkg/sem/pb/proto/sem/<ns>;<alias>"`)
- Generated TS: inside the new React package, e.g. `pinocchio/web/src/sem/pb/proto/sem/**` (exact path depends on the chosen UI repo layout)

One-command regeneration (pattern):

```bash
cd pinocchio
buf generate
```

This makes “what SEM events exist” explicit: adding a new event type implies adding a `.proto`, regenerating, and then implementing one Go handler + one TS handler against typed messages.

### B. Backend: make Pinocchio a “clean SEM server”

Target invariants for Pinocchio backend:
- Emits **SEM only** (remove `{ tl: true, ... }` once the React UI is the supported client).
- Uses a **registry** for event → SEM mapping (no monolithic switch; no “legacy alias keys”).
- Owns **message send serialization/queuing** (no frontend retry queue).
- Avoids sinks mutating conversation state (derived state belongs to middleware/engine/projection).
 - Authors SEM `data` payloads as **protobuf messages for every SEM type**, then uses protojson at the boundary.

Concrete refactors:

1) Replace the monolithic forwarder switch with a registry pattern
- Current: `pinocchio/pkg/webchat/sem_translator.go` emits SEM frames via a registry-only mapping (no switch fallback).
- Target: `pinocchio/pkg/sem/registry` (or reuse `moments/backend/pkg/sem/registry`) where each typed event registers its mapping function:

```go
// pseudocode
registry.RegisterByType[*events.EventPartialCompletion](func(ev *events.EventPartialCompletion, md events.EventMetadata) [][]byte {
  return []{ WrapSem(map{ type:"llm.delta", id: semID(md), data: { delta: ev.Delta, cumulative: ev.Completion }, metadata: md.LLMInferenceData }) }
})
```

This reduces drift and makes coverage auditable (you can list registered types).

1b) Make the registry handlers protobuf-first (no untyped JSON maps)

Instead of building arbitrary `map[string]any` payloads, every handler should:
- construct a protobuf message for the event family,
- serialize it with `protojson` into the SEM frame’s `data`,
- and ensure the frontend handler uses the same schema to parse the payload.

Pseudocode sketch:

```go
// pseudocode
msg := &sembase.ToolStart{Id: toolId, Name: toolName, Input: structpbInput}
data := pbToMap(msg) // protojson.Marshal + json.Unmarshal
ev := map[string]any{ "type": "tool.start", "id": toolId, "data": data, "metadata": md.LLMInferenceData }
return [][]byte{ WrapSem(ev) }
```

2) Move “send queue” semantics into the server
- Current UI approaches often do: optimistic UI + retry-on-409 in the client.
- Target: a server-side queue per conversation/session:
  - Pinocchio implements this on `POST /chat` with an idempotency key + per-conversation queue:
    - If no run is in progress: start inference immediately.
    - If a run is in progress: enqueue the prompt server-side and return `202 Accepted` (no client retry queue).
    - If the same idempotency key is submitted again: return the previously computed response (queued/running/completed/error).

Concrete API behavior (Pinocchio `POST /chat`):

- **Idempotency key input** (first match wins):
  - HTTP header `Idempotency-Key`
  - HTTP header `X-Idempotency-Key`
  - JSON body `idempotency_key`
  - If absent, the server generates a UUID and returns it as `idempotency_key` in the response.

- **Response when started (200 OK)**:

```json
{
  "status": "started",
  "idempotency_key": "...",
  "conv_id": "...",
  "session_id": "...",
  "run_id": "...",
  "turn_id": "...",
  "inference_id": "..."
}
```

- **Response when queued (202 Accepted)**:

```json
{
  "status": "queued",
  "queue_position": 2,
  "queue_depth": 2,
  "idempotency_key": "...",
  "conv_id": "...",
  "session_id": "...",
  "run_id": "..."
}
```

- **Repeat submissions** with the same idempotency key return the cached response (status transitions to `running` / `completed` / `error` as the server processes the queued item).

Implementation anchor (Pinocchio): `pinocchio/pkg/webchat/router.go`, `pinocchio/pkg/webchat/send_queue.go`.

2b) Add a hydration endpoint so reloads can replay a recent SEM history

Pinocchio now exposes a best-effort hydration endpoint backed by an in-memory per-conversation SEM frame buffer.

- Endpoint: `GET /hydrate?conv_id=<id>&profile=<slug>`
- Optional query params:
  - `since_seq=<uint64>`: only return frames with `event.seq > since_seq`
  - `limit=<int>`: return at most N frames (most recent)

Response (shape):

```json
{
  "conv_id": "...",
  "session_id": "...",
  "run_id": "...",
  "server_time": 1730000000000,
  "frames": [ { "sem": true, "event": { "...": "..." } } ],
  "frame_count": 123,
  "last_seq": 456,
  "last_stream_id": "1729012345-0",
  "queue_depth": 0,
  "running_idempotency": ""
}
```

Notes:

- This endpoint is intentionally **SEM-frame based**, not “timeline entity snapshot” based, so the frontend can reuse its own reducers/handlers to rebuild state.
- It is currently **not durable** (in-memory); durability/persistence is a follow-up if we need reloads to survive server restarts.
- It is designed to pair with a singleton WS manager + hydration gating:
  1) `GET /hydrate` → replay frames into the store
  2) open WS (or open WS early but gate processing until hydration completes)
  3) apply WS deltas with idempotent upsert semantics

Implementation anchor (Pinocchio): `pinocchio/pkg/webchat/router.go`, `pinocchio/pkg/webchat/sem_buffer.go`.

3) Eliminate “sink-owned conversation state”
- If a feature needs derived state (team suggestions, doc suggestions, memory extraction results), implement it as:
  - explicit middleware that writes to `Turn.Data`,
  - or a projection service owned by the engine/router,
  - not as a sink wrapper that mutates the conversation.

This keeps “what state exists?” independent of “which sink wrappers are enabled?”.

### C. Frontend: one reusable `ChatWidget` (React + RTK Toolkit) with no switch fallback

We create a new React module for Pinocchio webchat that follows Moments’ architecture:
- A normalized timeline slice (`timelineSlice`) with idempotent `addEntity` / `upsertEntity`.
- A SEM handler registry (`registerSem` / `handleSem`) that maps SEM events into timeline commands.
- A widget registry (`registry.ts` + `registerAll.ts`) that renders entities by kind.
- A singleton WS manager + hydration gating.

Non-negotiables (explicit):
- No `switch (ev.type)` fallback in the WebSocket hook.
- No “legacy alias keys” in payloads.
- No client-side send queue/retry semantics (no `chatQueueSlice` equivalent).
- All SEM handlers parse `ev.data` via **generated protobuf schemas** (Bufbuild `fromJson(MessageSchema, ...)`) and treat parse failures as contract violations.

### D. Storybook-first development workflow (widgets + scenarios)

We treat Storybook as a primary development loop for both:
- individual widgets (MessageWidget, ToolCallWidget, EditableSummaryWidget, PlanningWidget, …), and
- full “scenario playback” stories that simulate a realistic SEM stream.

This is patterned after the Moments/go-go-mento approach:
- `moments/web/src/platform/chat/SidebarChat/SidebarChat.stories.tsx`
- `moments/web/src/stories/withMockWsScenario.tsx`
- `moments/web/src/stories/withInitialTimeline.tsx`

For Pinocchio, the Storybook harness should provide:
1) A preconfigured RTK store with the timeline slice.
2) A “scenario runner” that replays SEM frames over time (deterministic).
3) A WS-mock layer (optional), but ideally a direct “inject SEM frame” helper so stories are fast and stable.

Pseudo-API for stories:

```ts
// pseudocode
const scenario = [
  { atMs: 0, frame: { sem: true, event: { type: "llm.start", id: "m1" } } },
  { atMs: 50, frame: { sem: true, event: { type: "llm.delta", id: "m1", data: { delta: "Hel" } } } },
  { atMs: 100, frame: { sem: true, event: { type: "llm.delta", id: "m1", data: { delta: "lo" } } } },
  { atMs: 200, frame: { sem: true, event: { type: "tool.start", id: "t1", data: { name: "calc", input: { expr: "1+1" } } } } },
  { atMs: 400, frame: { sem: true, event: { type: "tool.result", id: "t1", data: { result: "2" } } } },
  { atMs: 450, frame: { sem: true, event: { type: "tool.done", id: "t1" } } },
  { atMs: 600, frame: { sem: true, event: { type: "llm.final", id: "m1", data: { text: "Hello." } } } },
];
```

Stories then mount:
- `<ChatWidget conversationId="story" variant="sidebar" />`
- start scenario playback,
- and assert/visually inspect resulting timeline widgets.

## Design Decisions

1) **Single protocol: SEM only**
- Rationale: having both `{ tl: true, ... }` and `{ sem: true, ... }` doubles maintenance and invites drift. Pinocchio should converge on SEM as the sole UI protocol.

2) **Use a JSON envelope for SEM (protobuf payloads in `data`)**
- Decision: keep a JSON envelope `{ sem: true, event: { type, id, data, metadata } }` and standardize on protojson for `data` for every SEM type.
- Rationale: keeps the wire format human-debuggable while still making schemas canonical and type-checked via generated protobuf code (Go + TS).

3) **No frontend switch fallback**
- Rationale: a fallback switch duplicates logic, hides missing handler coverage, and leads to inconsistent semantics across widgets. Registry-only routing makes coverage explicit.

4) **Backend owns send serialization**
- Rationale: “run in progress” is a server fact; implementing retry queues in the client creates correctness bugs, and every client must replicate them. Centralize in the backend.

5) **No sink-owned conversation state**
- Rationale: sinks are event transport/enrichment; state mutation should be explicit and single-owner.

6) **Storybook as a core workflow**
- Rationale: widget development is easiest when it is decoupled from the full system (auth, server startup, real websocket). Scenario playback stories provide “unit tests you can see.”

## Alternatives Considered

1) Keep the current Preact + Zustand UI and “incrementally add widgets”
- Rejected: the Moments parity goal is explicitly React + RTK Toolkit + widget registry + Storybook workflows; incremental patches tend to recreate ad-hoc switches and drift.

2) Implement retry/queue semantics in the frontend (like Moments `chatQueueSlice`)
- Rejected: user intent is to move these semantics server-side; also, client queues are notoriously hard to keep correct across reloads, multiple tabs, and websocket reconnects.

3) Keep sink-driven derived state updates
- Rejected: this introduces hidden coupling and makes system behavior depend on sink configuration order.

## Implementation Plan

This plan assumes you want to reach “usable React chat” quickly, then iterate toward full Moments parity.

### Phase 0 — Contracts and invariants (paper cuts first)
- [x] Decide canonical SEM field naming (protojson camelCase vs snake_case) and enforce it.
- [x] Decide “stable ID” rules (which event generates which entity ID) and document them.
- [ ] Define “unknown event” behavior (debug-only generic widget).

#### Canonical SEM Field Naming

Pinocchio SEM uses **protojson output** for `event.data` (and for `event.metadata` when present), so keys are **lowerCamelCase** as produced by protojson.

Implications:

- protobuf fields like `max_tokens` become JSON keys like `maxTokens`
- `google.protobuf.Struct` becomes a JSON object with natural JSON key casing (whatever you put into the `Struct`)
- the SEM envelope itself stays JSON (hand-built), but its *payload* is protobuf-authored

#### Stable ID Rules (Pinocchio)

Stability of `event.id` is a correctness requirement: streaming and incremental updates must refer to the same entity across frames, websocket reconnects, and pagination/hydration.

Pinocchio rules:

1) **Prefer explicit UUID**: if Geppetto `EventMetadata.message_id` is set, it becomes `event.id` (canonical).

2) **Otherwise derive a stable LLM ID** from correlation metadata:

   - if `EventMetadata.inference_id` is present: `event.id = "llm-" + inference_id`
   - else if `EventMetadata.turn_id` is present: `event.id = "llm-" + turn_id`
   - else if `EventMetadata.session_id` is present: `event.id = "llm-" + session_id`
   - else: `event.id = "llm-" + <random uuid>`

3) **Thinking stream IDs**: thinking events reuse the base LLM ID with a suffix:

   - `event.id = <base_id> + ":thinking"`

4) **Tool IDs**: tool events use the tool call ID supplied by Geppetto/provider:

   - `tool.start`, `tool.delta`, `tool.result`, `tool.done`: `event.id = tool_call_id`

5) **Other IDs**:

   - `log`: prefer `message_id` if present, else `log-<random uuid>`
   - `agent.mode`: `agentmode-<turn_id>-<random uuid>` (best-effort; future: make it deterministic if we need idempotent upsert)
   - `debugger.pause`: `event.id = pause_id`

Implementation anchor (Pinocchio): `pinocchio/pkg/webchat/sem_translator.go`.

### Phase 1 — Backend cleanup to support a strict React client
- [ ] Add/standardize a SEM registry for Pinocchio event → SEM mappings (eliminate monolithic switch long-term).
  - Start by extracting the existing cases in `pinocchio/pkg/webchat/forwarder.go` into per-type handlers (now lives in `pinocchio/pkg/webchat/sem_translator.go`).
- [ ] Add server-side “send serialization”:
  - only one run executes per conversation at a time,
  - additional messages enqueue (or explicit “busy” errors with a server-side queue endpoint).
- [ ] Add timeline hydration endpoint(s) backed by persistence (if persistence exists) or in-memory snapshots (short term).
- [ ] Remove timeline-envelope output (`TimelineEventsFromEvent`) once React UI is the supported path.
- [ ] Ensure sinks are transport/extraction only; move derived state updates to middleware/engine/projection components.

### Phase 2 — Frontend foundation (React package + store + WS manager)
- [x] Choose the Pinocchio React frontend location and scaffold tooling.
  - Current choice: reuse and replace the existing `pinocchio/cmd/web-chat/web` Vite app with React + TypeScript + RTK.
  - `go generate ./pinocchio/cmd/web-chat` still builds the web assets into `pinocchio/cmd/web-chat/static/dist`.
- [ ] Implement Storybook dev loop for widget work.
  - Run: `cd pinocchio/cmd/web-chat/web && npm run storybook`
  - First story: `WebChat/ChatWidget` (includes a `ScenarioBasic` that replays a few SEM frames without a backend).
- [ ] Implement `timelineSlice` (normalized entities, add/upsert/append operations).
- [ ] Implement `sem/registry.ts` and register the minimum handlers:
  - `llm.start/delta/final`, `tool.start/delta/result/done`, `debug.pause`.
- [ ] Implement `wsManager` singleton + hydration gating.
- [ ] Implement `ChatWidget` root component as the only integration surface.

### Phase 3 — Widgets and rich event families
- [ ] Port the widget registry and baseline widgets from Moments (Message, ToolCall/Result, Status).
- [ ] Add structured widgets as needed (Planning, ThinkingMode, MultipleChoice, EditableSummary).
- [ ] Ensure every SEM type is handled via registry (no switch fallback).

### Phase 4 — Storybook workflow (widgets + scenarios)
- [ ] Add Storybook for the Pinocchio React package.
- [ ] Add story “fixtures”:
  - initial hydration state stories,
  - deterministic SEM scenario playback stories,
  - widget-only stories (render a single entity kind with representative props).
- [ ] Add a small “scenario runner” helper to drive time-based SEM playback.

### Phase 5 — Decommission legacy UI paths
- [ ] Deprecate `pinocchio/cmd/web-chat/web/**` (the Preact/Zustand UI) once React parity is reached.
- [ ] Remove any server compatibility behavior that only exists for the legacy client (timeline envelope, duplicate naming, etc.).

## Open Questions

1) How should “queued while busy” behave UX-wise: hard disable input, show queued count, or accept input and show “pending” state?
2) What persistence layer is the source of truth for hydration (DB vs Redis vs in-memory snapshot)?
3) How should multi-tab behavior work (one active writer, many readers, per-session ownership)?

## References

- Ticket architecture analysis: `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/analysis/01-moments-react-chat-widget-architecture.md`
- Pinocchio backend webchat (current): `pinocchio/pkg/webchat/sem_translator.go`, `pinocchio/pkg/sem/registry/registry.go`, `pinocchio/pkg/webchat/router.go`, `pinocchio/pkg/webchat/stream_coordinator.go`
- Pinocchio current web UI (Preact/Zustand): `pinocchio/cmd/web-chat/web/src/store.js`, `pinocchio/cmd/web-chat/web/src/app.js`
- Moments platform chat hook (shows legacy switch to avoid in Pinocchio): `moments/web/src/platform/chat/hooks/useChatStream.ts`
- go-go-mento SEM registry + protobuf-shaped JSON usage:
  - `go-go-mento/go/pkg/webchat/handlers/helpers.go`
  - `go-go-mento/web/src/sem/registry.ts`
  - `go-go-mento/web/src/sem/handlers/planning.ts`
- go-go-mento Buf toolchain and SEM proto sources (template for Pinocchio):
  - `go-go-mento/buf.yaml`
  - `go-go-mento/buf.gen.yaml`
  - `go-go-mento/proto/sem/**`
