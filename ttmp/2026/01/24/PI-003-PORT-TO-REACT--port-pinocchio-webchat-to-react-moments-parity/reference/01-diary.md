---
Title: Diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/analysis/01-moments-react-chat-widget-architecture.md
      Note: Main architecture narrative produced from this research
    - Path: geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/reference/01-diary.md
      Note: Research diary capturing the file/command trail
    - Path: pinocchio/cmd/web-chat/web/src/store.js
      Note: Adjusted to read event.data for smoke-testing
    - Path: pinocchio/pkg/webchat/router.go
      Note: ws.hello/ws.pong payload now in event.data
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Step 15 implementation (commit 949beb9)
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-24T13:52:51.900577155-05:00
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Keep a highly detailed, step-by-step research diary for PI-003-PORT-TO-REACT: how the Moments / go-go-mento React chat UI works end-to-end (Redux Toolkit + RTK Query + WebSocket SEM frames) and how the Go backend turns Geppetto events + structured sinks (“extractors”) into SEM frames and widget updates.

## Step 1: Create ticket workspace and establish a research map

Created a new `docmgr` ticket workspace and seeded it with an analysis document and this diary document, then confirmed that both `go-go-mento/` and `moments/` exist locally in the monorepo workspace so the “deep analysis” can be based on actual source, not guesswork.

I also did a quick inventory pass to locate the likely “hot paths” for chat: React/Redux store composition, WebSocket handling, SEM handler registries, timeline entity reducers/renderers, and the Go webchat router/streaming pipeline. This avoided prematurely reading low-signal code.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket PI-003-PORT-TO-REACT . 

Do a deep analysis of how the react chat widget is setup in go-go-mento / moments, especially how SEM events are routed and extracted from sinks and passed over the to the JS side of things, and how the chat widget is wired up, focus on rtk-toolkit and websocket communication and how the backend in go transforms incoming geppetto events into updates to widgets and streaming chat and all that. 

Write a detailed document in a detailed textbook style as if written by peter norvig, do search for existing documentation and link it, describe all of it because I want to seriously improve the webchat in pinocchio to have all these nice affordances and run with react. 

Store the document in the docmgr ticket. 

Keep a very detailed diary of your research and the thinking you are doing and where you are looking at and why, in addition to all of the other stuff. Add at intermediate steps during your research."

**Assistant interpretation:** Create a new docmgr ticket, then write (and store) a comprehensive architecture document explaining the Moments/go-go-mento React chat widget and the Go backend streaming pipeline, emphasizing SEM event routing, sink extractors, Redux Toolkit patterns, and WebSocket behavior; keep a detailed research diary while doing it.

**Inferred user intent:** Build a faithful “reference architecture” (Moments parity) to port/upgrade Pinocchio’s webchat into a React-based, affordance-rich, streaming chat system with observable event flows and maintainable state management.

**Commit (code):** N/A

### What I did
- Ran `docmgr ticket create-ticket --ticket PI-003-PORT-TO-REACT ...` and added two docs: `analysis/01-moments-react-chat-widget-architecture.md` and `reference/01-diary.md`.
- Confirmed relevant source trees exist locally: `moments/`, `moments/web/`, `moments/backend/`, `go-go-mento/web/`, `go-go-mento/go/`.
- Located the “primary” UI + WS files by using ripgrep and opening the first-hop README/docs and core hooks.

### Why
- Ticket-first keeps outputs discoverable and reviewable.
- A “map first, then drill down” approach reduces false starts when reading unfamiliar code.

### What worked
- `docmgr` was already initialized and created the ticket workspace under `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/`.
- Both Moments and go-go-mento codebases are present, enabling side-by-side comparison and lineage tracing.

### What didn't work
- N/A

### What I learned
- The monorepo already contains both the “source” (go-go-mento) and “port” (moments), so most “search” can be done locally (plus targeted external references for technologies like RTK Query, Watermill, WebSockets).

### What was tricky to build
- N/A (setup-only step)

### What warrants a second pair of eyes
- N/A (no code changes)

### What should be done in the future
- N/A

### Code review instructions
- Start with `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/analysis/01-moments-react-chat-widget-architecture.md` once populated.
- Use this diary to reproduce the research trail and file selection rationale.

### Technical details
- Ticket workspace root: `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/`
- Docs created:
  - `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/analysis/01-moments-react-chat-widget-architecture.md`
  - `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/reference/01-diary.md`

## Step 2: Identify the frontend “spine” (Redux Toolkit + SEM registry + WebSocket) in Moments and go-go-mento

Mapped the client-side architecture by following the dataflow from (1) user input → (2) HTTP POST “start chat” → (3) WebSocket stream of SEM frames → (4) SEM registry routing → (5) Redux timeline entity updates → (6) widget renderers. I intentionally read these in the order the system executes, because it exposes the “contracts” (message schema, entity IDs, upsert semantics) that have to match across frontend and backend.

I also compared Moments to go-go-mento to see which parts are direct ports and which are simplified. This matters for Pinocchio: if Moments is a partial port, you may want to copy the more robust go-go-mento patterns (e.g., singleton WS manager + hydration gating) rather than re-derive them.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket PI-003-PORT-TO-REACT . 

Do a deep analysis of how the react chat widget is setup in go-go-mento / moments, especially how SEM events are routed and extracted from sinks and passed over the to the JS side of things, and how the chat widget is wired up, focus on rtk-toolkit and websocket communication and how the backend in go transforms incoming geppetto events into updates to widgets and streaming chat and all that. 

Write a detailed document in a detailed textbook style as if written by peter norvig, do search for existing documentation and link it, describe all of it because I want to seriously improve the webchat in pinocchio to have all these nice affordances and run with react. 

Store the document in the docmgr ticket. 

Keep a very detailed diary of your research and the thinking you are doing and where you are looking at and why, in addition to all of the other stuff. Add at intermediate steps during your research."

**Assistant interpretation:** Build a precise, code-grounded description of the React chat widget wiring (Redux/RTK Query + WebSocket SEM frames) and how those events become timeline entities/widgets.

**Inferred user intent:** Capture the “true” event/state model so Pinocchio can adopt it without regressions (especially around streaming, tool widgets, and step-debugger UX).

**Commit (code):** N/A

### What I did
- Moments UI entry points:
  - Located `moments/web/src/features/chat/ChatPage.tsx` and `moments/web/src/features/chat/ChatPageWithSidebar.tsx` to see how chats are mounted.
  - Followed the sidebar path: `moments/web/src/platform/chat/hooks/useSidebarChat.ts` → `moments/web/src/platform/chat/hooks/useChatStream.ts`.
  - Traced Redux wiring: `moments/web/src/store/store.ts`, `moments/web/src/store/api/apiSlice.ts`, `moments/web/src/store/api/rpcSlice.ts`, and `moments/web/src/platform/api/chatApi.ts`.
  - Traced timeline rendering: `moments/web/src/platform/timeline/Timeline.tsx`, `moments/web/src/platform/timeline/registry.ts`, `moments/web/src/platform/timeline/state/timelineSlice.ts`, `moments/web/src/platform/timeline/types.ts`, and the widgets under `moments/web/src/platform/timeline/widgets/`.
  - Traced SEM routing: `moments/web/src/platform/sem/registry.ts` and handlers under `moments/web/src/platform/sem/handlers/`.
- go-go-mento UI entry points (for comparison):
  - WebSocket lifecycle: `go-go-mento/web/src/ws/manager.ts` (singleton manager), `go-go-mento/web/src/ws/hooks.ts` (`useConversation`), and `go-go-mento/web/src/hooks/useTimelineHydration.ts`.
  - SEM routing robustness: `go-go-mento/web/src/sem/handlers/llm.ts` and `go-go-mento/web/src/sem/handlers/userMessage.ts`.

### Why
- These files define the frontend’s “contracts”: SEM envelope shape, entity IDs, dispatch semantics (`add` vs `upsert`), and the widget catalog. Anything that changes in the backend must preserve these contracts, or the UI will drop/duplicate events.

### What worked
- The Moments frontend has a clear separation:
  - **Transport**: `useChatStream` opens `/rpc/v1/chat/ws` and parses `{ sem: true, event: ... }`.
  - **Routing**: `handleSem` maps event `type` to a handler that emits `{ kind: 'add'|'upsert', entity }`.
  - **State**: `timelineSlice` provides idempotent `addEntity` / `upsertEntity` and streaming helpers (append/finalize).
  - **Presentation**: `Timeline` renders `entity.kind` via a registry of widget renderers.
- The go-go-mento frontend includes additional production-hardening:
  - A singleton WS manager with ref counting, hydration gating, and generation counters to avoid duplicate sockets under React StrictMode.
  - A more robust LLM streaming handler that tolerates missing `llm.start`, empty deltas, and prefers `cumulative` when present.

### What didn't work
- Moments does not currently show a timeline hydration layer (HTTP GET snapshots) analogous to go-go-mento’s `useTimelineHydration`, which may matter for Pinocchio if reload-resume is a requirement.

### What I learned
- The Moments client already speaks the “SEM envelope” protocol (`{ sem: true, event: {...} }`) and routes most “rich UI” event types (thinking mode, planning, multiple-choice, etc.) through a handler registry; the widget system is essentially a typed event-sourced UI.
- go-go-mento’s WS manager/hydration pattern is likely the most transferable “battle-tested” piece for a Pinocchio React port.

### What was tricky to build
- The critical subtlety is **entity identity**: everything assumes stable IDs (`ev.id`, `item_id`, etc.) so repeated events upsert into the same widget rather than creating duplicates.

### What warrants a second pair of eyes
- Whether Moments intentionally dropped timeline hydration (DB snapshots) or it simply hasn’t been ported yet; that affects design goals for Pinocchio (reload persistence vs purely in-memory).

### What should be done in the future
- N/A (this step is analysis-only)

### Code review instructions
- Validate the traced execution order by starting at:
  - `moments/web/src/platform/chat/hooks/useSidebarChat.ts`
  - `moments/web/src/platform/chat/hooks/useChatStream.ts`
  - `moments/web/src/platform/sem/registry.ts`
  - `moments/web/src/platform/timeline/state/timelineSlice.ts`
  - `moments/web/src/platform/timeline/widgets/`

### Technical details
- Moments transport endpoint (client-side): `/rpc/v1/chat/ws` (constructed in `useChatStream` with `conv_id`, optional `profile`, optional `draft_bundle_id`, and optional `access_token` query params).
- Moments send path (client-side): Redux queue thunk `enqueueChatMessage` calling `chatApi.startChat` / `chatApi.startChatWithProfile` via RTK Query.
- go-go-mento hydration endpoint (client-side): `GET /rpc/v1/conversations/{convId}/timeline?sinceVersion=...` (via `go-go-mento/web/src/store/api/chatTimelineApi.ts`).

## Step 3: Trace backend “event → sink → SEM → WebSocket → widgets” in go-go-mento and Moments

Completed the backend pipeline trace by starting from the HTTP “start chat” handler, following where the run context is built and where the event sink is attached, and then tracking how events flow through the sink stack (including structuredsink extractors) into SEM frames and finally into WebSocket broadcasts.

I did this in both codebases because the Moment’s implementation is “the target” for Pinocchio parity, while go-go-mento contains a more decomposed (and in places more battle-tested) architecture (StreamCoordinator/ConnectionPool, timeline hydration).

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket PI-003-PORT-TO-REACT . 

Do a deep analysis of how the react chat widget is setup in go-go-mento / moments, especially how SEM events are routed and extracted from sinks and passed over the to the JS side of things, and how the chat widget is wired up, focus on rtk-toolkit and websocket communication and how the backend in go transforms incoming geppetto events into updates to widgets and streaming chat and all that. 

Write a detailed document in a detailed textbook style as if written by peter norvig, do search for existing documentation and link it, describe all of it because I want to seriously improve the webchat in pinocchio to have all these nice affordances and run with react. 

Store the document in the docmgr ticket. 

Keep a very detailed diary of your research and the thinking you are doing and where you are looking at and why, in addition to all of the other stuff. Add at intermediate steps during your research."

**Assistant interpretation:** Explain the backend mechanics that produce the frontend’s SEM stream: where events originate, how sink wrappers/extractors produce structured events, and how those are translated and broadcast to the browser.

**Inferred user intent:** Avoid a superficial “we use websockets” explanation; capture the actual extraction/translation machinery so Pinocchio can reliably reproduce the same rich widgets and debugging affordances.

**Commit (code):** N/A

### What I did
- go-go-mento backend deep path:
  - Webchat HTTP/WS routing: `go-go-mento/go/pkg/mento/http/public_server.go` (route registration) and `go-go-mento/go/pkg/webchat/router.go` (handlers).
  - Run context wiring: confirmed `events.WithEventSinks(runCtx, conv.Sink)` in `go-go-mento/go/pkg/webchat/router.go`.
  - Sink composition: `go-go-mento/go/pkg/webchat/engine_builder.go` and `go-go-mento/go/pkg/webchat/sink_wrapper.go` (Watermill sink + FilteringSink extractors + extra wrappers).
  - Streaming bridge: `go-go-mento/go/pkg/webchat/stream_coordinator.go` and `go-go-mento/go/pkg/webchat/connection_pool.go`.
  - SEM translation: `go-go-mento/go/pkg/webchat/event_translator.go` plus handler registry `go-go-mento/go/pkg/sem/registry/registry.go`.
  - “Accumulator sinks” updating `Turn.Data`: `go-go-mento/go/pkg/webchat/suggestions_sink.go` and `go-go-mento/go/pkg/webchat/doc_suggestions_sink.go`.
  - Timeline hydration endpoints (go-go-mento): `go-go-mento/go/pkg/persistence/timelinehydration/rpc.go` (`GET /rpc/v1/conversations/{convId}/timeline`).
- Moments backend deep path:
  - Webchat router + conversation lifecycle: `moments/backend/pkg/webchat/router.go` and `moments/backend/pkg/webchat/conversation.go`.
  - Sink registry and extractors: `moments/backend/pkg/app/sink_registry.go` plus representative extractors:
    - `moments/backend/pkg/inference/middleware/thinkingmode/extractor.go`
    - `moments/backend/pkg/inference/middleware/teamselection/extractor.go`
    - `moments/backend/pkg/inference/middleware/question/extractor.go`
    - `moments/backend/pkg/inference/middleware/summary/chunk_extractor.go`
    - `moments/backend/pkg/memory/extractor.go`
  - SEM translation in Moments: `moments/backend/pkg/webchat/forwarder.go` plus handler registry `moments/backend/pkg/sem/registry/registry.go` and core handler registration `moments/backend/pkg/sem/handlers/*`.
  - Feature SEM handlers (examples): `moments/backend/pkg/analytics/sem_handlers.go` for `summary.chunk.*` and `artifact.persisted`.
  - Accumulator sinks: `moments/backend/pkg/doclens/doc_suggestions_sink.go` and `moments/backend/pkg/teamchat/team_suggestions_sink.go`.

### Why
- The “rich webchat” experience (tool widgets, structured blocks, planning traces, thinking mode panels, etc.) depends on the exact backend event→SEM mapping and on stable ID contracts. If Pinocchio reimplements only the UI without reproducing the backend extraction and registry logic, the widgets will never appear reliably.

### What worked
- Clear end-to-end event flow emerged in both systems:
  1) HTTP `POST /rpc/v1/chat...` starts a run and sets up a `runCtx`.
  2) `runCtx` includes the conversation’s sink via `events.WithEventSinks`.
  3) Geppetto emits events (LLM streaming, tool lifecycle, plus typed structured events from extractors).
  4) The sink pipeline transports those events via Watermill (Redis Streams or event router).
  5) A reader loop (StreamCoordinator in go-go-mento; startReader in Moments) decodes events, translates them into SEM frames, and broadcasts to all connected WebSocket clients for that conversation.
- The structuredsink extractor mechanism is concrete and inspectable:
  - Extractors identify `<pkg:type:ver>` blocks and emit typed events on start/update/completion.
  - Some extractors resolve identity context (e.g., team member extractor resolves person IDs using the conversation’s session).
  - Some extractors persist to DB (e.g., memory extractor) and still emit SEM-visible events.

### What didn't work
- There are (understandable) “version skew” mismatches between docs and code:
  - Some existing docs mention `/api/conversations/.../timeline`, while go-go-mento’s actual registered hydration route is `/rpc/v1/conversations/{convId}/timeline` (as implemented by `timelinehydration/rpc.go`).

### What I learned
- “SEM events routed and extracted from sinks” is literally:
  - **Extraction**: `structuredsink.NewFilteringSink(baseSink, ..., extractors...)` recognizes tagged output and emits additional typed events into the same sink stream.
  - **Routing**: a registry (`semregistry`) maps Go event concrete types to SEM frames; frontend registry maps SEM `type` to timeline entity upserts.
- The system is intentionally layered so that adding a new widget typically requires:
  1) an extractor (if the widget is derived from LLM-tagged structured output),
  2) a backend SEM handler (event → SEM frame),
  3) a frontend SEM handler (SEM frame → timeline entity),
  4) a widget renderer for the new entity kind.

### What was tricky to build
- WebSocket concurrency constraints are easy to violate if you have multiple goroutines writing to the same connection (e.g., ping responses plus stream broadcasts). go-go-mento’s `ConnectionPool` design makes serialization obvious; Moments relies more on call-graph discipline plus locks in specific places.
- The extractors that “reach back” into the conversation manager for session resolution (team selection, memory) have subtle coupling: you must maintain a reliable mapping from event metadata (session id / topic / conv id) back to the live conversation.

### What warrants a second pair of eyes
- Confirm the “single broadcaster goroutine” assumption in Moments backend (i.e., that only the reader loop writes SEM frames), because `connsMu.RLock()` does not serialize concurrent writers.
- Decide whether Pinocchio wants go-go-mento’s timeline hydration semantics; Moments does not obviously expose the same HTTP hydration endpoint today.

### What should be done in the future
- N/A (analysis-only)

### Code review instructions
- For go-go-mento backend streaming: start at `go-go-mento/go/pkg/webchat/stream_coordinator.go` and then read `go-go-mento/go/pkg/webchat/event_translator.go`.
- For sink extraction in Moments: start at `moments/backend/pkg/app/sink_registry.go`, then open one extractor (e.g., `moments/backend/pkg/inference/middleware/thinkingmode/extractor.go`) and then the corresponding SEM handler in `moments/backend/pkg/sem/handlers/`.

### Technical details
- go-go-mento: sink composition entry point is `go-go-mento/go/pkg/webchat/engine_builder.go` → `middleware.NewWatermillSink(publisher, topicForConv(convID))` → wrapper stack (FilteringSink extractors, suggestion sinks).
- go-go-mento: translation is `go-go-mento/go/pkg/webchat/event_translator.go` (registry first, switch fallback).
- Moments: sink composition is `moments/backend/pkg/app/sink_registry.go` (profile-gated wrappers).
- Moments: translation is `moments/backend/pkg/webchat/forwarder.go` (registry first; if not handled, drop).

## Step 4: Collect existing documentation and write the textbook-style architecture doc

Located and used the existing go-go-mento webchat documentation as the “prior art,” then wrote a unifying architecture document in this ticket that (a) points to those docs, and (b) explains Moments-specific deltas and the extractor/SEM plumbing in one place.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket PI-003-PORT-TO-REACT . 

Do a deep analysis of how the react chat widget is setup in go-go-mento / moments, especially how SEM events are routed and extracted from sinks and passed over the to the JS side of things, and how the chat widget is wired up, focus on rtk-toolkit and websocket communication and how the backend in go transforms incoming geppetto events into updates to widgets and streaming chat and all that. 

Write a detailed document in a detailed textbook style as if written by peter norvig, do search for existing documentation and link it, describe all of it because I want to seriously improve the webchat in pinocchio to have all these nice affordances and run with react. 

Store the document in the docmgr ticket. 

Keep a very detailed diary of your research and the thinking you are doing and where you are looking at and why, in addition to all of the other stuff. Add at intermediate steps during your research."

**Assistant interpretation:** Produce a single comprehensive narrative doc, link existing internal docs, and add external references for the core technologies used.

**Inferred user intent:** Have one “canonical” doc you can hand to yourself (or teammates) while porting Pinocchio, without constantly re-deriving the dataflow from code.

**Commit (code):** N/A

### What I did
- Found the existing go-go-mento webchat doc set:
  - `go-go-mento/docs/reference/webchat/frontend-integration.md`
  - `go-go-mento/docs/reference/webchat/sem-and-widgets.md`
  - `go-go-mento/docs/reference/webchat/backend-internals.md`
  - `go-go-mento/docs/reference/webchat/engine-builder.md`
  - `go-go-mento/docs/reference/persistence/timeline-hydration.md`
- Added external reference links for Redux Toolkit, RTK Query, Watermill, Gorilla WebSocket, and AWS ALB context.
- Wrote the consolidated architecture doc into:
  - `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/analysis/01-moments-react-chat-widget-architecture.md`

### Why
- The existing go-go-mento docs are already very close to what Pinocchio needs; the “value add” is stitching them to Moments and to the actual extractor/SEM translation code paths in this monorepo.

### What worked
- The consolidated doc now serves as a “table of contents + deep pointers” into both codebases and their existing documentation.

### What didn't work
- N/A

### What I learned
- The go-go-mento docs are effectively a ready-made playbook for building a robust streaming chat UI; Moments is a practical instance with some structural changes (event router, sink registry in app, etc.).

### What was tricky to build
- Keeping the document readable while still giving enough file-level precision to be actionable.

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- Relate the key source files to the doc via `docmgr doc relate` so the ticket has a tight “RelatedFiles” list.

### Code review instructions
- Review the architecture doc in the ticket and then spot-check the linked files for correctness.

## Step 5: Upload the ticket docs to reMarkable

### What I did
- Confirmed the `remarquee` tool and cloud auth were working:
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
- Chose to bundle the ticket docs into a single PDF (for a single clickable ToC and easier reading on-device):
  - `.../index.md`
  - `.../analysis/01-moments-react-chat-widget-architecture.md`
  - `.../reference/01-diary.md`
- Did a dry-run to confirm ToC titles + output path:
  - `remarquee upload bundle --dry-run <paths...> --name "PI-003-PORT-TO-REACT - Moments Webchat Architecture" --remote-dir "/ai/2026/01/24/PI-003-PORT-TO-REACT" --toc-depth 2`
- Uploaded the bundle:
  - `remarquee upload bundle <paths...> --name "PI-003-PORT-TO-REACT - Moments Webchat Architecture" --remote-dir "/ai/2026/01/24/PI-003-PORT-TO-REACT" --toc-depth 2`
- Verified it exists in the destination folder:
  - `remarquee cloud ls /ai/2026/01/24/PI-003-PORT-TO-REACT --long --non-interactive`

### Why
- Bundling keeps the “architecture doc + diary + index” together, and the ToC makes it easy to jump between sections on reMarkable.

### What worked
- End-to-end upload succeeded; the file shows up in `/ai/2026/01/24/PI-003-PORT-TO-REACT`.

### What didn't work
- N/A

## Step 6: Update the port design (single component + keyboard profile) and overwrite the reMarkable PDF

### Prompt context
- User request:
  - “we want one consistent component, not split up in 5 smaller ones.”
  - “can we determine which keyboard is what programmatically? to differentiate between ADV and non-ADV?”
  - “Update the design, overwrite the pdf on remarkable.”

### What I changed (design)
- Updated the architecture document’s porting guidance to explicitly recommend **one consistent `ChatWidget` root component** as the only integration surface, with layout variants controlled via props (rather than multiple “chat widgets” with diverging semantics).
- Added a section on **ADV vs non-ADV keyboards** that separates:
  - what the browser can reliably detect (key events + layout hints),
  - what requires a native host (Electron/OS HID enumeration),
  - and the recommended approach: treat “ADV” as a **capability profile** with an explicit user override (and optional host/backend override), not as a browser-detectable hardware identity.
- Updated the “port checklist” accordingly (single `ChatWidget`, keyboard profile support).

Changed file:
- `analysis/01-moments-react-chat-widget-architecture.md`

### Why
- Multiple component trees for “chat” almost always drift: subtle differences in store wiring, websocket lifecycle, retry handling, and shortcut semantics accumulate and become product bugs.
- Keyboard-device identification is a privacy-sensitive area in browsers; the safest UX is a capability profile with explicit user control.

### Upload (overwrite)
- Rebuilt and overwrote the existing reMarkable bundle PDF (explicit overwrite requested; note this can delete existing annotations on-device).

Commands run:
- Dry-run:
  - `remarquee upload bundle --dry-run <index.md> <architecture.md> <01-diary.md> --name "PI-003-PORT-TO-REACT - Moments Webchat Architecture" --remote-dir "/ai/2026/01/24/PI-003-PORT-TO-REACT" --toc-depth 2`
- Upload (overwrite):
  - `remarquee upload bundle --force <index.md> <architecture.md> <01-diary.md> --name "PI-003-PORT-TO-REACT - Moments Webchat Architecture" --remote-dir "/ai/2026/01/24/PI-003-PORT-TO-REACT" --toc-depth 2`
- Verify:
  - `remarquee cloud ls /ai/2026/01/24/PI-003-PORT-TO-REACT --long --non-interactive`

## Step 7: Add an “evolution” section (go-go-mento → Moments) using git history

### Prompt context
- User request: “Add a section to the document that examines how go-go-mento evolved into moments … Be detailed. You can use git logs as well.”

### What I did (research trail)
1) Confirmed why `git log` didn’t work at the workspace root:
   - This workspace is not itself a git repo; `moments/`, `go-go-mento/`, and `geppetto/` are separate worktrees with their own `.git` files.
2) Pulled “anchor commits” in `moments/` that represent major migration and refactor phases:
   - `git -C moments show -n 1 --name-status f24d0669` (bulk web UI import; introduces `PORT-WEB`)
   - `git -C moments show -n 1 --name-status 96dca9c1` (adds `backend/pkg/webchat/**`; introduces `MIGRATE_CHAT`)
   - `git -C moments show -n 1 --name-status 0bbe7b0a` (ports middlewares/sinks; introduces `IMPORT-MIDDLEWARE`)
   - `git -C moments show -n 1 --name-status fa93d59f` (structured extraction fixes; introduces `SEM-EXTRACTORS` ticket workspace)
   - `git -C moments show -n 1 --name-status 9281c31b` (frontend platform refactor moving chat/timeline under `web/src/platform/**`)
   - `git -C moments show -n 1 --name-status c22124f2` and `9388e8ea` (SEM registry/handlers refactor)
3) Pulled “anchor commits” in `go-go-mento/` to show what existed earlier (and what got matured there):
   - `git -C go-go-mento show -n 1 --name-status 655175867` (introduces SidebarChat)
   - `git -C go-go-mento show -n 1 --name-status b26741ddc` (StreamCoordinator + EventTranslator rewrite)
4) Read the migration tickets in Moments that explicitly describe the intended porting strategy and phasing:
   - `moments/ttmp/PORT-WEB-port-mento-web-ui-into-moments/**`
   - `moments/ttmp/MIGRATE_CHAT-migrate-webchat-into-moments/**`
   - `moments/ttmp/IMPORT-MIDDLEWARE-port-middleware-and-structured-event-emitters-from-mento-playground-go-pkg-webchat-into-webchat/**`

### What I changed (document)
- Added a new historical section to the architecture doc explaining:
  - the “copy first, refactor second” migration pattern,
  - the frontend path (SidebarChat and Timeline preserved, later moved under a platform layer),
  - the backend path (webchat package port + middleware/sink/extractor parity + SEM handler registry refactor),
  - and why those phases matter for a Pinocchio React port.

Changed file:
- `analysis/01-moments-react-chat-widget-architecture.md`

### Why this matters (my interpretation)
- If Pinocchio wants Moments-grade affordances, it should follow the same migration discipline:
  - preserve semantic contracts (SEM envelope, stable IDs, timeline widgets),
  - wire it into the host product,
  - then refactor toward maintainable registries and platform modules.

## Step 8: Overwrite the reMarkable PDF bundle with the updated “evolution” section

### Prompt context
- User request: “yes” (overwrite the reMarkable PDF so it includes Part V).

### What I did
- Dry-run:
  - `remarquee upload bundle --dry-run <index.md> <architecture.md> <01-diary.md> --name "PI-003-PORT-TO-REACT - Moments Webchat Architecture" --remote-dir "/ai/2026/01/24/PI-003-PORT-TO-REACT" --toc-depth 2`
- Upload (overwrite):
  - `remarquee upload bundle --force <index.md> <architecture.md> <01-diary.md> --name "PI-003-PORT-TO-REACT - Moments Webchat Architecture" --remote-dir "/ai/2026/01/24/PI-003-PORT-TO-REACT" --toc-depth 2`
- Verify:
  - `remarquee cloud ls /ai/2026/01/24/PI-003-PORT-TO-REACT --long --non-interactive`

### Result
- The bundle PDF in `/ai/2026/01/24/PI-003-PORT-TO-REACT` is updated in-place.

## Step 9: Refine the architecture + Pinocchio port rules (protobuf, no switch fallback, backend-owned queue, no sink-owned state) and add a Pinocchio refactor plan + Storybook workflow

This step tightens the “Pinocchio porting” guidance so it matches what you actually want to build: a strict React/RTK implementation with no legacy crutches, and a backend that owns correctness (queue/serialization) rather than pushing it into frontend retries. In addition, it adds an explicit Pinocchio-side implementation plan document so the ticket is not just analysis but a concrete roadmap.

I treated this as “make the implicit decisions explicit”: where Moments/go-go-mento currently include migration artifacts (legacy switch fallbacks, duplicate handler paths, sink-driven derived state), the Pinocchio plan should *not* inherit them by accident.

### Prompt Context

**User prompt (verbatim):**

```text
- how does protobuf go into the SEM events / event structs
- what's the legacy switch you mention in the primary frontend handling in core streaming + tools (what 's the fallback switch in useChatStream as well? are thse necessary?)
- update t6o say that there shouldn't be sinks updating conversation state, that's a smell / design pattern mistake. 

in the port to pinocchio part: update to say: no switch fallback, no backwards compatibility. The chatQueueSlice semantics should be handled on the backend side, not in the frontend. 
- we do want ws manager + hydration gating
- remove the adv vs nonadv keyboard section, i was mixixng up two projects. 

Once you update the design document, create a new implementation planning/design document that explains the refactoring steps on the pinocchio side to move towards the react solution in detail. 

We also want to have a section that details storybook work so we can work on individual chatwidgets .

Use pseudocode and prose paragraphs and bullet points and refer to existing source and documentation files and all that.

Analyyze and store in ticket and upload to remarkable.

Keep a detailed diary.
```

**Assistant interpretation:** Update the main architecture doc to clarify protobuf → SEM encoding, explain the `useChatStream` legacy switch and whether it’s needed, remove the keyboard section, and change Pinocchio port guidance to “no fallbacks / no backwards compat / backend owns queue semantics / keep ws manager + hydration gating / no sink-owned conversation state”. Then add a new design/plan doc for Pinocchio refactoring including a Storybook workflow.

**Inferred user intent:** You want a strict, maintainable React port for Pinocchio with Moments-class affordances, but without importing migration baggage that makes the system hard to reason about.

**Commit (code):** N/A

### What I did
- Answered “protobuf in SEM” by reading the actual implementation:
  - Backend uses `protojson.Marshal` + `json.Unmarshal` to turn protobuf payload messages into JSON maps for SEM `data` (`go-go-mento/go/pkg/webchat/handlers/helpers.go`), and frontend rehydrates that JSON using Bufbuild schemas (`go-go-mento/web/src/sem/handlers/planning.ts`).
- Identified the exact “legacy switch fallback” in Moments:
  - `moments/web/src/platform/chat/hooks/useChatStream.ts` routes via `handleSem` first, then falls back to `switch (ev.type)` which still handles core families (`llm.*`, `tool.*`, summary chunks, artifacts, etc.).
- Updated the primary architecture doc:
  - Removed the ADV/non-ADV keyboard section.
  - Added a “protobuf-shaped JSON” explanation section.
  - Reframed sink-owned conversation state updates as a design smell (and called out the existing accumulator sinks as historical/legacy rather than a target pattern for Pinocchio).
  - Updated the Pinocchio port principles to explicitly require: no switch fallback, no backwards-compat payload aliases, no frontend `chatQueueSlice`, backend-owned send serialization/queue semantics, keep ws manager + hydration gating.
- Created a new design doc in the ticket with a detailed refactor plan (including Storybook workflows and scenario playback pseudocode):
  - `design-doc/01-pinocchio-react-webchat-refactor-plan.md`
- Updated the ticket index to link the new plan doc.
- Related the most relevant source files to the new plan doc via `docmgr doc relate` (Pinocchio backend, Pinocchio current Preact UI, Moments `useChatStream`, go-go-mento protobuf-to-SEM helper, go-go-mento SEM registry).

### Why
- “Legacy switch” fallbacks and sink-owned state are the kinds of accidental complexity that make ports brittle. Since you want “no backwards compatibility,” Pinocchio can start clean: registry-only routing, canonical schemas, and backend-owned correctness.

### What worked
- The code paths for protobuf → SEM were straightforward to audit and document: protobuf is used as schema/authoring, JSON is the wire format.
- Pinocchio already has a strong backend baseline (`pinocchio/pkg/webchat/**`), which makes the plan much more actionable than “rewrite everything”.

### What didn't work
- N/A

### What I learned
- Moments’ “registry-first, switch fallback” is not an abstract idea — it is literally duplicated logic in `useChatStream`. If the registry were complete, the switch could disappear.
- Sink-based accumulators (doc/team suggestions) are convenient but create hidden coupling; this aligns with your instinct that it’s a design smell.

### What was tricky to build
- Avoiding mischaracterizing go-go-mento: it is registry-driven on the frontend, but the “switch” patterns mostly show up in mock streams (and in Moments’ real hook), so the doc needed to be precise about where the switch actually lives.

### What warrants a second pair of eyes
- The “backend-owned queue semantics” API sketch in the plan doc: you’ll likely want to sanity-check it against Pinocchio’s existing router endpoints and desired UX (queued vs rejected while busy).

### What should be done in the future
- Decide whether Pinocchio will adopt the protobuf schema approach (protojson + Bufbuild `fromJson`) for structured widgets, or standardize on JSON-only payloads with TS types.

### Code review instructions
- Start with the updated architecture doc and the new plan doc:
  - `analysis/01-moments-react-chat-widget-architecture.md`
  - `design-doc/01-pinocchio-react-webchat-refactor-plan.md`
- Spot-check the referenced code for correctness:
  - `go-go-mento/go/pkg/webchat/handlers/helpers.go` (protojson boundary)
  - `moments/web/src/platform/chat/hooks/useChatStream.ts` (legacy switch fallback)
  - `pinocchio/pkg/webchat/forwarder.go` (current SEM mapping switch)

### Technical details
- Key files consulted:
  - `go-go-mento/go/pkg/webchat/handlers/helpers.go`
  - `go-go-mento/web/src/sem/handlers/planning.ts`
  - `moments/web/src/platform/chat/hooks/useChatStream.ts`
  - `moments/backend/pkg/app/sink_registry.go`
  - `pinocchio/pkg/webchat/forwarder.go`
  - `pinocchio/cmd/web-chat/web/src/store.js`

## Step 10: Upload the updated ticket bundle to reMarkable (and fix a pandoc failure)

This step is the “distribution step”: after updating the architecture doc and adding the Pinocchio refactor plan, I bundled the ticket docs into a single PDF for reMarkable reading. The only complication was a LaTeX/pandoc failure caused by a literal `\n` sequence in the diary’s “verbatim prompt” string, which LaTeX interprets as an undefined control sequence.

After converting that prompt into a fenced `text` block (no backslashes), the bundle build succeeded and the updated PDF was uploaded.

### Prompt Context

**User prompt (verbatim):** "Analyyze and store in ticket and upload to remarkable."

**Assistant interpretation:** Rebuild the bundle PDF from the updated ticket docs and upload it to the reMarkable folder for this ticket.

**Inferred user intent:** You want the updated analysis + plan available for offline review on the device.

**Commit (code):** N/A

### What I did
- Verified `remarquee` tooling and account were available.
- Attempted upload with a new bundle name (to include the new design doc alongside the analysis + diary):
  - `remarquee upload bundle --force <index.md> <architecture.md> <plan.md> <01-diary.md> --name "PI-003-PORT-TO-REACT - Moments+Pinocchio Webchat" --remote-dir "/ai/2026/01/24/PI-003-PORT-TO-REACT" --toc-depth 2`
- Hit pandoc/LaTeX error:
  - `! Undefined control sequence. l.4642 events / event structs\n`
- Fixed the diary prompt formatting by replacing the literal `\n` sequences with a fenced `text` block.
- Re-ran:
  - Dry-run: `remarquee upload bundle --dry-run ...`
  - Upload (overwrite for that filename): `remarquee upload bundle --force ...`
- Verified remote contents:
  - `remarquee cloud ls /ai/2026/01/24/PI-003-PORT-TO-REACT --long --non-interactive`

### Why
- Bundling keeps the analysis, plan, and diary together with a clickable ToC for device reading.

### What worked
- Upload succeeded after fixing the pandoc/LaTeX issue.

### What didn't work
- Initial pandoc build failed due to a literal backslash sequence (`\n`) in the diary’s prompt string.

### What I learned
- For reMarkable bundle PDFs, avoid literal backslash sequences in normal prose; use fenced code blocks for “verbatim prompt” content when it contains backslashes or other LaTeX-sensitive sequences.

### What was tricky to build
- The failure surfaced late (at PDF generation time), so the fix required editing the diary rather than the “main” analysis/plan docs.

### What warrants a second pair of eyes
- Whether you want to keep the older PDF (`PI-003-PORT-TO-REACT - Moments Webchat Architecture`) or replace it with the new combined bundle as the single canonical device document.

### What should be done in the future
- If you want exactly one on-device doc, re-upload using the original filename with `--force` (this will overwrite and may remove existing annotations on the device).

### Code review instructions
- Verify the new plan doc reads as a coherent implementation roadmap:
  - `design-doc/01-pinocchio-react-webchat-refactor-plan.md`
- Confirm the bundle includes all four docs by checking the PDF ToC on-device.

### Technical details
- Remote folder: `/ai/2026/01/24/PI-003-PORT-TO-REACT`
- Bundle name uploaded: `PI-003-PORT-TO-REACT - Moments+Pinocchio Webchat`

## Step 11: Update the Pinocchio port plan to be protobuf-first for all SEM events

This step refines the Pinocchio port plan to make the SEM contract fully explicit and enforceable: every SEM event family should be defined in protobuf, and the schemas should be compiled to both Go (backend authoring/validation) and TypeScript (frontend parsing/typing). This is consistent with the proven go-go-mento approach (Buf + `protoc-gen-go` + `protoc-gen-es`) and removes a large class of “stringly-typed drift” failures.

### Prompt Context

**User prompt (verbatim):** "Update the porting plan to incorporate defining all SEM events as protobuf and compile them to go and TS."

**Assistant interpretation:** Modify the Pinocchio refactor plan doc so protobuf is the source of truth for SEM event payload schemas, with a Buf-based generation workflow to Go + TS.

**Inferred user intent:** Enforce a single canonical, versionable contract for SEM events so backend and frontend evolve together without hand-maintained JSON shape assumptions.

**Commit (code):** N/A

### What I did
- Located the existing reference implementation of the proto toolchain and SEM proto sources:
  - `go-go-mento/buf.yaml`, `go-go-mento/buf.gen.yaml`
  - `go-go-mento/proto/sem/**`
  - Observed generated outputs in both Go and TS (`go-go-mento/go/pkg/sem/pb/**`, `go-go-mento/web/src/sem/pb/**`).
- Updated the Pinocchio plan doc to:
  - explicitly require protobuf-defined payloads for **every** SEM event type,
  - describe adopting the same Buf workflow in Pinocchio (`pinocchio/proto/sem/**` → generated Go+TS),
  - add a concrete “schema toolchain” subsection and update open questions accordingly (envelope message vs `{type,id,data}` with protojson data).

Changed file:
- `design-doc/01-pinocchio-react-webchat-refactor-plan.md`

### Why
- A protobuf-first SEM schema makes event families explicit, type-checked, and easier to evolve safely across the backend/frontend boundary.

## Step 12: Overwrite the reMarkable bundle PDF with the protobuf-first update

### Prompt Context

**User prompt (verbatim):** "yes"

**Assistant interpretation:** Overwrite the existing reMarkable bundle PDF so the on-device document includes the protobuf-first changes to the port plan.

**Inferred user intent:** Keep the device copy as the canonical up-to-date reading artifact.

**Commit (code):** N/A

### What I did
- Dry-run:
  - `remarquee upload bundle --dry-run <index.md> <architecture.md> <plan.md> <01-diary.md> --name "PI-003-PORT-TO-REACT - Moments+Pinocchio Webchat" --remote-dir "/ai/2026/01/24/PI-003-PORT-TO-REACT" --toc-depth 2`
- Upload (overwrite):
  - `remarquee upload bundle --force <index.md> <architecture.md> <plan.md> <01-diary.md> --name "PI-003-PORT-TO-REACT - Moments+Pinocchio Webchat" --remote-dir "/ai/2026/01/24/PI-003-PORT-TO-REACT" --toc-depth 2`
- Verify:
  - `remarquee cloud ls /ai/2026/01/24/PI-003-PORT-TO-REACT --long --non-interactive`

### Result
- The PDF `PI-003-PORT-TO-REACT - Moments+Pinocchio Webchat` is updated in-place under `/ai/2026/01/24/PI-003-PORT-TO-REACT`.

## Step 13: Make the implementation plan self-contained (reading list + concept callouts) and create detailed ticket tasks

This step turns the plan into something an implementor can pick up cold: it adds an ordered reading list and “fundamental concepts” callouts that explain the architectural primitives (event-sourced UI, normalized entities, hydration gating, protojson boundary). It also replaces the placeholder tasks with a detailed, phase-ordered checklist that `docmgr task list` can index.

### Prompt Context

**User prompt (verbatim):** "make sure you properly reference documents to read in the new implementation plan so that an implementor has all the information necessary and all the context to get tgoing. Add callouts for the fundamental concepts / architecture knowledge needed. Update the implementation plan, create tasks in the ticket (detailed)"

**Assistant interpretation:** Improve the plan doc by adding explicit “what to read” references and conceptual prerequisites, then write a detailed task list in the ticket so execution is straightforward.

**Inferred user intent:** You want the ticket to be directly actionable by an engineer without them needing to rediscover the context or hunt for source docs.

**Commit (code):** N/A

### What I did
- Updated the Pinocchio refactor plan doc with:
  - a “Prerequisites (core concepts you must understand)” section,
  - and a “Documents to read (in recommended order)” section that points to the most relevant in-repo docs (go-go-mento webchat docs, Moments web docs, and Pinocchio’s existing semantic mapping notes).
- Replaced the placeholder task list with a detailed checklist in `tasks.md`, with acceptance criteria and explicit references.
- Verified `docmgr task list --ticket PI-003-PORT-TO-REACT` indexes the tasks as discrete items.

Changed files:
- `design-doc/01-pinocchio-react-webchat-refactor-plan.md`
- `tasks.md`

### Why
- The plan was directionally correct but still assumed the reader already knew the “why” behind key invariants. The added callouts make the architecture teachable.
- A detailed task list prevents the project from degenerating into “random refactors” and keeps the port aligned with the explicit non-goals (no switch fallback, no legacy protocols, protobuf-first).

### What worked
- `docmgr` successfully indexed the rewritten `tasks.md` into 15 distinct tasks.

### What didn't work
- N/A

### What I learned
- The Moments and go-go-mento doc sets already cover most conceptual prerequisites; what was missing was an explicit “reading order” and “these are the invariants” list inside the plan doc.

### What warrants a second pair of eyes
- Whether the task list should be split further (e.g., separate “proto schema authoring” tasks per event family) or kept at the current “phase” granularity.

## Step 14: Lock JSON envelope decision and implement Pinocchio SEM protobuf toolchain (Buf) + check off tasks

This step begins execution of the ticket’s task list. You specified the envelope decision (“use json envelope”), so I locked that into the implementation plan, checked off the corresponding ticket task, and then implemented the first concrete backend-side deliverable: a Pinocchio-local protobuf schema tree and Buf-based Go+TS code generation, committed to the Pinocchio git repo.

The focus was deliberately “foundational plumbing” rather than a behavior change: we now have a canonical proto source tree and generated artifacts, but we did not yet change Pinocchio’s live SEM event shapes.

### Prompt Context

**User prompt (verbatim):**

```text
- use json envelope
- start working, check off tasks, commit to git, keep diary
```

**Assistant interpretation:** Decide and document that Pinocchio will use the JSON SEM envelope, then begin implementing the plan, checking off tasks as they’re completed, committing code changes, and recording the work in this diary.

**Inferred user intent:** Start converting the plan from “paper” into committed infrastructure, with traceable progress and explicit decisions.

**Commit (code):** `c07ea5c03b50a8254343127ca8dbb2901f84d7f4` — "pinocchio: add SEM protos + buf Go/TS generation"

### What I did
- Updated the implementation plan doc to explicitly record the decision: **SEM uses a JSON envelope** with protobuf payloads in `event.data` (protojson).
- Checked off ticket tasks in `tasks.md` via docmgr:
  - Task 2 (envelope decision)
  - Task 4 (proto source tree)
  - Task 5 (Buf toolchain)
- Implemented Pinocchio protobuf toolchain + schemas (copied from go-go-mento as a baseline):
  - Added `pinocchio/proto/sem/**` by copying `go-go-mento/proto/sem/**` and updating `option go_package` to Pinocchio’s module path.
  - Added `pinocchio/buf.yaml` and `pinocchio/buf.gen.yaml` (Go + TS codegen).
  - Ran `cd pinocchio && buf lint && buf generate`.
  - Verified Go compilation via `cd pinocchio && go test ./...`.
- Committed the changes in the Pinocchio git repo:
  - `git -C pinocchio commit -m "pinocchio: add SEM protos + buf Go/TS generation"`

### Why
- The protobuf toolchain is a prerequisite for making “all SEM events are protobuf-defined” real: without a stable schema + codegen pipeline, the system drifts back toward hand-maintained JSON shapes.
- Locking the JSON envelope decision early prevents later refactors from multiplying protocol variants.

### What worked
- `buf lint` passes with a minimal exceptions list (we are not enforcing version-suffixed packages yet).
- `buf generate` produced both Go (`pinocchio/pkg/sem/pb/**`) and TS (`pinocchio/web/src/sem/pb/**`) artifacts.
- `go test ./...` passed, and the Pinocchio pre-commit hook also ran and succeeded during the commit.

### What didn't work
- Initially, I tried adding `build.roots` to `buf.yaml`; Buf v1 rejected that config. Removing it and relying on lint exceptions fixed the issue.

### What I learned
- The go-go-mento proto layout and package naming conventions require either (a) relaxed lint rules, or (b) a directory/package restructuring. For now, relaxed lint rules are the pragmatic choice so we can proceed.

### What was tricky to build
- Making Buf lint pass without forcing an immediate “rename packages to .v1 + move files into per-package directories” migration (which would be a large, noisy change).

### What warrants a second pair of eyes
- Whether we want to enforce versioned protobuf package names (`.v1`) now, or defer until after the first end-to-end Pinocchio React client is working.

## Step 15: Replace Pinocchio SEM switch with registry-only translator + protobuf-authored payloads (stable IDs, data envelope)

This step turns the “protobuf-first + registry-only” plan into real backend behavior. I removed the monolithic SEM `switch` in Pinocchio and replaced it with a registry-backed translator that emits **only** the JSON SEM envelope and puts all payload fields into **protobuf-authored `event.data`** (protojson). In the same pass I fixed a correctness issue: previously, when `message_id` was missing, streaming LLM events could produce *different* IDs per frame, which breaks UI coalescing; the new translator derives stable IDs from `inference_id` / `turn_id` / `session_id`.

I kept the work “always green”: I ran `buf lint && buf generate`, added a unit test for ID stability + tool result shaping, ran `go test ./...`, and let the Pinocchio pre-commit hook run its full suite (including a Vite build for the web-chat frontend).

### Prompt Context

**User prompt (verbatim):** "continue , make sure you compile and test as you go along, to make sure we are on track"

**Assistant interpretation:** Continue executing the ticket plan (starting with backend SEM refactors), and continuously compile/test so we don’t drift or break the system.

**Inferred user intent:** Maintain tight feedback loops while moving from design → implementation, ensuring each chunk is shippable and reviewable.

**Commit (code):** `949beb9d01bd967b95c85c2dc22d39a13b5ab2a0` — "webchat: registry-only SEM + protobuf payloads"

### What I did
- Implemented a global SEM handler registry and rewired Pinocchio webchat to be **registry-only** (no fallback switch):
  - Added `pinocchio/pkg/sem/registry/registry.go`
  - Added `pinocchio/pkg/webchat/sem_translator.go` and removed the switch-based mapping from `pinocchio/pkg/webchat/forwarder.go`
- Implemented and documented **stable ID rules** in code:
  - Prefer `EventMetadata.message_id` when set
  - Else derive from `inference_id`, else `turn_id`, else `session_id`, else random UUID
  - Thinking stream IDs use `:thinking` suffix
- Defined protobuf schemas for the core SEM event families used by Pinocchio today and generated Go+TS code via Buf:
  - New: `pinocchio/proto/sem/base/{llm,metadata,log,agent,debugger,ws}.proto`
  - Updated: `pinocchio/proto/sem/base/tool.proto` (added `custom_kind`)
  - Generated: `pinocchio/pkg/sem/pb/proto/sem/base/*.pb.go` and `pinocchio/web/src/sem/pb/proto/sem/base/*_pb.ts`
- Updated backend event emission to conform to the planned wire contract:
  - Envelope: `{ "sem": true, "event": { "type": "...", "id": "...", "data": <protojson>, "metadata": <protojson?> } }`
  - `event.data` and `event.metadata` now use protojson lowerCamelCase field names.
- Updated websocket greeting/pong events to put their payload in `event.data` (protobuf-authored):
  - `pinocchio/pkg/webchat/router.go`
- Updated the existing Pinocchio “web-chat” frontend (Zustand store) so it reads `event.data.*` instead of top-level fields:
  - `pinocchio/cmd/web-chat/web/src/store.js`
- Added tests for the new behavior:
  - `pinocchio/pkg/webchat/sem_translator_test.go` asserts stable IDs and calc tool result `customKind`.

### Why
- Registry-only routing enforces handler coverage and prevents accidental “half-migrations” where a legacy switch keeps masking missing SEM handlers.
- Protobuf-authored `event.data` is the simplest way to make schemas canonical across Go and TypeScript while keeping the wire format inspectable.
- Stable IDs are required for correct streaming message coalescing (without stable IDs the UI can’t upsert).

### What worked
- `cd pinocchio && buf lint && buf generate`
- `cd pinocchio && go test ./...`
- Pinocchio pre-commit hook completed successfully and ran the repo’s standard checks.

### What didn't work
- The Pinocchio pre-commit hook surfaced `npm audit` vulnerabilities during the web-chat build step (2 moderate, 1 high). The hook did not fail the commit, but this should be tracked separately.

### What I learned
- Moving to `event.data` changes field casing expectations: protojson emits lowerCamelCase (e.g., `maxTokens`), which is a deliberate break from the prior snake_case `LLMInferenceData` JSON tags.

### What was tricky to build
- Ensuring “registry-only” does not silently drop events: handlers that return `(nil, nil)` count as “found” but produce no frames; that is correct for `EventInfo` messages we intentionally ignore, but should be used sparingly elsewhere.
- Keeping the legacy web-chat frontend working enough for smoke testing while changing the SEM payload contract.

### What warrants a second pair of eyes
- The protobuf package organization: everything in `proto/sem/base/*` compiles into a single Go package `base`, which is convenient but may become unwieldy as SEM families grow.
- Whether `agent.mode` IDs should be made deterministic (vs random) for idempotent upsert semantics in future hydration scenarios.

### What should be done in the future
- Consider adding protobuf schemas for any remaining SEM event families (and enforce “no ad-hoc JSON” in `event.data`).
- Decide whether to gate or fail CI on `npm audit` output for the embedded web frontend build step.

### Code review instructions
- Start with `pinocchio/pkg/webchat/sem_translator.go` (registry registrations + protobuf shaping + stable IDs).
- Check the schema changes in `pinocchio/proto/sem/base/*.proto` and confirm generated output is expected.
- Validate with:
  - `cd pinocchio && buf lint && buf generate`
  - `cd pinocchio && go test ./...`

### Technical details
- Stable ID resolution logic is based on `EventMetadata.{message_id,inference_id,turn_id,session_id}` and is implemented in the translator (not in sinks).
- Tool result “custom kind” is now a protobuf field (`custom_kind`) emitted as protojson `customKind`, used to hint renderers (e.g. `calc_result`).

## Step 16: Remove legacy TL protocol from Pinocchio (SEM-only stream)

This step removes the last “second protocol” in Pinocchio webchat: the legacy TL envelope (`{ tl: true, event: ... }`). It’s dead weight now that SEM is registry-only and protobuf-authored; keeping it around invites drift and creates implicit compatibility obligations. After deleting the TL forwarder, I re-ran Go tests to confirm there are no hidden call sites, then checked off the corresponding ticket task and updated the plan docs to stop referencing the removed file.

### Prompt Context

**User prompt (verbatim):** "continue , make sure you compile and test as you go along, to make sure we are on track"

**Assistant interpretation:** Keep executing the next backend cleanup items while maintaining a compile/test feedback loop.

**Inferred user intent:** Remove legacy surface area early so the React port targets one protocol and one contract.

**Commit (code):** `a407483122762f68f0460fc3af2e09c095c870c0` — "webchat: remove legacy TL protocol"

### What I did
- Deleted the TL envelope code path from Pinocchio:
  - Removed `pinocchio/pkg/webchat/forwarder.go` (it only contained `TimelineEventsFromEvent` / `{ tl: true }` framing).
- Verified Go compilation and tests:
  - `cd pinocchio && go test ./...`
- Checked off docmgr Task #7 (remove legacy TL protocol) and updated ticket docs to stop referencing the deleted file:
  - `tasks.md`
  - `design-doc/01-pinocchio-react-webchat-refactor-plan.md`

### Why
- Maintaining two protocols doubles maintenance cost and creates ambiguity about which one is canonical.
- The port goal is React + registry-only SEM; TL is not part of that architecture.

### What worked
- Removing the file did not break any builds/tests; it appears TL was already unused by live code.

### What didn't work
- N/A

### What I learned
- Pinocchio’s legacy TL envelope had already become “documentation-only”; deleting it was low-risk and clarifies future work.

### What was tricky to build
- N/A (straight deletion + verify).

### What warrants a second pair of eyes
- Confirm no downstream consumers (outside this repo) still expect TL frames; if any exist, they should migrate to SEM rather than reintroducing TL.

### What should be done in the future
- Remove any remaining docs in `pinocchio/ttmp/**` that imply TL is supported, or move them to an “archive” section.

### Code review instructions
- Verify deletion: `pinocchio/pkg/webchat/forwarder.go` no longer exists.
- Validate: `cd pinocchio && go test ./...`

### Technical details
- TL envelope removal is a hard break by design; new clients should be SEM-only and registry-driven.
