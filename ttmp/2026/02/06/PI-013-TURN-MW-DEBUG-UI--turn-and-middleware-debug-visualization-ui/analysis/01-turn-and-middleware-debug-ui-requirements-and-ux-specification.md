---
Title: Turn and Middleware Debug UI Requirements and UX Specification
Ticket: PI-013-TURN-MW-DEBUG-UI
Status: active
Topics:
    - websocket
    - middleware
    - turns
    - events
    - frontend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/04-events.md
      Note: Event sink and streaming model documentation
    - Path: geppetto/pkg/doc/topics/08-turns.md
      Note: Canonical turns and blocks architecture and key semantics
    - Path: geppetto/pkg/doc/topics/09-middlewares.md
      Note: Middleware composition model and ordering guidance
    - Path: geppetto/pkg/events/structuredsink/filtering_sink.go
      Note: Structured sink extraction and malformed-policy behaviors
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: Context wiring for sinks and snapshot hooks
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Snapshot phases and tool loop iteration lifecycle
    - Path: pinocchio/cmd/web-chat/web/src/sem/registry.ts
      Note: Frontend SEM handler and aggregation behavior
    - Path: pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts
      Note: Client timeline upsert/rekey logic
    - Path: pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx
      Note: Current rendering shell and component extension points
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Hydration and buffered replay behavior
    - Path: pinocchio/pkg/webchat/router.go
      Note: HTTP endpoints
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Geppetto event to SEM mapping logic
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Event sequencing and SEM frame dispatch
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: SEM to timeline entity projection and throttling details
    - Path: pinocchio/pkg/webchat/timeline_store.go
      Note: Timeline projection storage contract
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: Versioned timeline snapshot retrieval and upsert semantics
    - Path: pinocchio/pkg/webchat/turn_store.go
      Note: Turn snapshot storage contract
    - Path: pinocchio/pkg/webchat/turn_store_sqlite.go
      Note: Durable snapshot schema and query filters
    - Path: pinocchio/proto/sem/timeline/transport.proto
      Note: Canonical timeline entity and snapshot wire contract
ExternalSources: []
Summary: Comprehensive requirements and UX specification for a web UI that visualizes turn snapshots, middleware mutations, structured sink extraction, SEM event flows, and timeline projection behavior.
LastUpdated: 2026-02-06T21:10:00-05:00
WhatFor: Provide a detailed designer/developer handoff for building a middleware and turn-debug visualization UI.
WhenToUse: Use when implementing or designing tooling to inspect block/turn mutation paths, event design, and inference-time behavior in Pinocchio/Geppetto.
---


# Turn and Middleware Debug UI Requirements and UX Specification

## 1. Executive summary

This document specifies a dedicated debugging web application for understanding how Geppetto turns/blocks evolve through middleware and tool-loop phases, and how those transformations surface as SEM events and timeline entities in Pinocchio webchat.

The target outcome is not "another chat UI." It is a developer instrument panel for:

- Seeing exact before/after turn snapshots at controlled phases.
- Explaining which middleware changed what, where, and why.
- Correlating turn-level mutations with event emissions and timeline projection writes.
- Inspecting structured sink extraction and event-schema evolution.
- Accelerating debugging of hydration, ordering, and inference behavior.

The implementation should support high-fidelity forensic workflows (postmortem) and live streaming workflows (inference in progress).

## 2. Scope and boundaries

### In scope

- Turn and block mutation visualization for a single inference run and across runs.
- Middleware chain introspection and per-middleware diffing.
- Tool-loop phase snapshoting (`pre_inference`, `post_inference`, `post_tools`, `final`).
- Correlation of turns with events (`session_id`, `inference_id`, `turn_id`, message/event IDs).
- Timeline projection visibility (`/timeline`, `timeline.upsert`, projector aggregation behavior).
- Turn snapshot visibility (`/turns`, persisted YAML payloads by phase).
- Structured sink and typed extraction visibility for custom structured payloads.
- UI affordances for new event design and schema rollout validation.

### Out of scope (initial release)

- Replacing existing chat widget UX.
- Production end-user feature polish for non-technical audiences.
- Full data lineage back to provider raw HTTP payloads.
- Mutation replay with deterministic re-execution in the provider.

## 3. Current system inventory (as implemented)

This section is grounded in current repository state on February 6, 2026.

### 3.1 Canonical Turn model

Source: `geppetto/pkg/turns/types.go`, `geppetto/pkg/turns/keys.go`, `geppetto/pkg/doc/topics/08-turns.md`

Current facts:

- A `Turn` is ordered `[]Block` plus opaque typed-key stores `Turn.Metadata` and `Turn.Data`.
- `Block.Kind` includes `user`, `llm_text`, `tool_call`, `tool_use`, `system`, `reasoning`, `other`.
- Typed keys exist for Turn metadata (`session_id`, `inference_id`, usage, model, etc.), block metadata (`middleware`, `agentmode_tag`, etc.), and Turn data (`agent_mode`, tool config, etc.).
- `Turn.Clone()` deep-copies block slices/maps and wrapper maps (shallow value copy).
- Tool-loop pending-call detection is ID-based: `tool_call.id` unresolved by matching `tool_use.id`.

Implication for debug UI:

- The core artifact to visualize is the full Turn snapshot plus typed-key stores and per-block metadata.
- ID-based tracking is crucial: block identity is not only order, it is also stable IDs.

### 3.2 Middleware composition and mutation behavior

Source: `geppetto/pkg/inference/middleware/middleware.go`, `geppetto/pkg/inference/toolloop/enginebuilder/builder.go`, `pinocchio/pkg/webchat/engine.go`, `geppetto/pkg/doc/topics/09-middlewares.md`

Current facts:

- Middleware chain order is deterministic: `Chain(m1,m2,m3)` executes as `m1(m2(m3(handler)))`.
- Pinocchio engine composition always prepends `ToolResultReorderMiddleware`, appends requested profile/override middlewares, then system prompt middleware.
- Middlewares mutate the same `*turns.Turn` pointer in-place (conventional behavior), then call `next`.
- No built-in generic "per-middleware before/after snapshot hook" currently exists in middleware chain itself.
- Existing middlewares that materially mutate blocks/data include:
  - `systemprompt` (insert/replace system block, set block metadata).
  - `tool result reorder` (reorders block sequence for tool adjacency constraints).
  - `agentmode` (inserts user prompt block, updates Turn.Data mode keys, appends system switch notice, emits agent-mode events).
  - `sqlitetool` (registers runtime tool; mostly affects tool availability and downstream behavior).

Implication for debug UI:

- The biggest observability gap is per-middleware mutation attribution.
- We can currently see pre/post loop phases, but not which middleware caused each mutation unless inferred indirectly.

### 3.3 Tool-loop snapshot hooks and phases

Source: `geppetto/pkg/inference/toolloop/loop.go`, `geppetto/pkg/inference/toolloop/context.go`, `pinocchio/pkg/webchat/router.go`, `pinocchio/pkg/webchat/turn_store.go`, `pinocchio/pkg/webchat/turn_store_sqlite.go`, `pinocchio/pkg/webchat/turn_persister.go`

Current facts:

- Snapshot hook phases currently emitted by tool loop:
  - `pre_inference`
  - `post_inference`
  - `post_tools`
- Final persistence happens separately via `TurnPersister` using phase `final`.
- Pinocchio snapshot hook persists YAML snapshots into optional `TurnStore` and optional filesystem directory.
- `TurnStore` schema currently stores:
  - `conv_id`, `run_id`, `turn_id`, `phase`, `created_at_ms`, `payload`
- Query path exists via `GET /turns` with filters (`conv_id` or `run_id`, `phase`, `since_ms`, `limit`).

Limitations:

- `phase` is a free string, no explicit schema/version.
- No sequence index to guarantee strict order when timestamps collide.
- No middleware name attached to snapshots.
- No diff artifact persisted, only full YAML payload.

Implication for debug UI:

- Base required data exists for phase-level timeline.
- To achieve middleware-level introspection, new snapshot points and metadata are required.

### 3.4 Event sinks and structured sink affordances

Source: `geppetto/pkg/events/context.go`, `geppetto/pkg/events/chat-events.go`, `geppetto/pkg/events/registry.go`, `geppetto/pkg/events/structuredsink/filtering_sink.go`, `geppetto/pkg/doc/topics/04-events.md`

Current facts:

- Events are published via context-attached sinks (`events.WithEventSinks` + `PublishEventToContext`).
- Event model supports many event types and custom type registration via codec/factory registry.
- `structuredsink.FilteringSink` can:
  - remove tagged structured blocks from streamed text,
  - emit extractor-typed events (`OnStart`, `OnRaw`, `OnCompleted`),
  - support malformed policies and capture limits.
- Structured sink is a key extension point for new event designs and inference output extraction.

Limitations:

- No built-in visualization of extraction sessions, capture states, malformed handling, or dropped payloads.
- No first-class UI for comparing raw text stream vs filtered text vs extracted typed events.

Implication for debug UI:

- A sink-focused lane is required: raw event stream, filtered stream, extracted artifacts, and policy outcomes.

### 3.5 SEM translation and timeline projection pipeline

Source: `pinocchio/pkg/webchat/stream_coordinator.go`, `pinocchio/pkg/webchat/sem_translator.go`, `pinocchio/pkg/webchat/timeline_projector.go`, `pinocchio/pkg/webchat/timeline_store.go`, `pinocchio/pkg/webchat/timeline_store_sqlite.go`, `pinocchio/proto/sem/timeline/*.proto`

Current facts:

- StreamCoordinator subscribes to watermill topic, decodes events, assigns monotonic sequence, and emits SEM envelopes.
- SEM translator maps Geppetto (and Pinocchio typed) events to SEM event types (`llm.*`, `tool.*`, `thinking.mode.*`, `planning.*`, etc.).
- Timeline projector consumes SEM frames and writes projection entities into `TimelineStore` via upsert.
- Projection entities are typed protobuf oneof snapshots (`message`, `tool_call`, `tool_result`, `thinking_mode`, `planning`, etc.).
- Hydration API: `GET /timeline?conv_id=<>&since_version=<>&limit=<>`.
- Real-time push: `timeline.upsert` SEM events to connected clients.

Important behavior details:

- `llm.delta` writes are throttled (250ms) in projector to reduce DB churn.
- Planning events are aggregated in-memory into one `planning` entity per `run_id`.
- Upsert store preserves `created_at_ms` for existing entities and updates `updated_at_ms`.

Limitations:

- No UI for projector-internal aggregation state.
- No visibility into dropped/throttled interim deltas.
- No explicit provenance chain linking a specific turn mutation to a specific timeline upsert.

Implication for debug UI:

- We need side-by-side views for event-level truth and projection-level truth.
- We need explicit correlation primitives, not inferred visual matching only.

### 3.6 Existing frontend capabilities

Source: `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`, `pinocchio/cmd/web-chat/web/src/sem/registry.ts`, `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`, `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`, `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.stories.tsx`

Current facts:

- Existing webchat frontend is a timeline card renderer, not a debug inspector.
- It hydrates from `/timeline`, then applies buffered SEM frames.
- It supports many kinds (`message`, `tool_call`, `tool_result`, `thinking_mode`, `planning`, etc.) with cards.
- Storybook scenarios exist for replay of SEM sequences.

Limitations:

- No visualization of `/turns` snapshots.
- No middleware chain/phase panel.
- No diff viewer for block-level changes.
- No sink/extractor inspector.
- No event schema explorer.
- Planning card currently emphasizes generic props display and does not provide forensic depth.

Implication for debug UI:

- Existing renderer can be a reference component library, but a dedicated debug IA is required.

### 3.7 Existing CLI affordances

Source: `pinocchio/cmd/web-chat/timeline/*.go`

Current facts:

- CLI commands exist for timeline inspection (`list`, `snapshot`, `entities`, `entity`, `stats`, `verify`).
- No parallel CLI for turn snapshots currently.

Implication for debug UI:

- CLI is useful for backend validation, but UI can provide richer correlation and interaction.

## 4. Primary users and jobs-to-be-done

### 4.1 Middleware developer

Needs to answer:

- Which middleware mutated this turn/block?
- Was mutation intended (insert/replace/reorder)?
- Did mutation break downstream tool matching or role ordering?
- Did metadata/data keys drift unexpectedly?

### 4.2 Structured sink developer

Needs to answer:

- Did extractor detect open/close tags correctly?
- Were malformed blocks handled per policy?
- What did raw stream contain vs forwarded filtered stream?
- Which typed events were emitted and with what payload?

### 4.3 Event schema designer (SEM / typed events)

Needs to answer:

- Is new event type translated correctly from source event?
- Is payload shape stable and consumable by UI/projection?
- Are IDs/seq/session/turn correlations preserved?
- Does projector map event to intended timeline entity shape?

### 4.4 Inference debugging user (workflow engineer)

Needs to answer:

- Why did this run diverge from expected behavior?
- Where did ordering or hydration mismatch occur?
- What happened between user prompt and final answer in exact steps?

## 5. Problem statement and opportunity

Current observability is split across:

- Logs,
- raw events,
- timeline projection,
- persisted turn YAML snapshots,
- frontend card rendering.

This fragmentation slows debugging and encourages guesswork. A unified debug UI can reduce issue triage time dramatically by making causal chains visible:

- Turn mutation timeline,
- middleware action attribution,
- event emission timeline,
- projection materialization timeline,
- frontend hydration application timeline.

## 6. Product principles for the debug UI

1. Evidence-first: show concrete payloads and diffs, not only summaries.
2. Correlation-first: every pane must support session/inference/turn/event linking.
3. Loss visibility: explicitly show throttled, dropped, filtered, malformed, or ignored data.
4. Replayability: a captured run should be inspectable offline from persisted artifacts.
5. Designer-ready: component vocabulary should map cleanly to visual system decisions.

## 7. Information architecture (proposed)

### 7.1 Global layout

- Left rail: Run/Conversation selector + filters.
- Center workspace: synchronized multi-lane timeline.
- Right inspector: deep payload viewer + diff + metadata/data keys + provenance.
- Bottom panel: query/log console and export tools.

### 7.2 Core lanes

1. Turn snapshot lane
- Rows = snapshots.
- Columns = phase, timestamp, turn_id, block count, change summary.

2. Middleware lane
- Rows = middleware invocation boundaries.
- Nodes = before/after states and emitted events.

3. Event lane
- Rows = ordered events/SEM frames.
- Visual tokens for event family and payload schema version.

4. Projection lane
- Rows = timeline entity upserts with version.
- Highlights for entity kind and mutation type.

5. Hydration lane (optional)
- For frontend-focused debugging: snapshot fetch, clear, replay buffer, live apply.

### 7.3 Inspector tabs

- Raw payload
- Normalized payload
- Diff (semantic + raw)
- Key/value stores (`Turn.Data`, `Turn.Metadata`, `Block.Metadata`)
- Correlations (IDs and links)
- Validation (schema checks, invariants)

## 8. Functional requirements

### 8.1 Run selection and filtering

FR-001: User can load runs by `conv_id`, `run_id/session_id`, date range.

FR-002: User can filter by middleware name, event type, phase, entity kind.

FR-003: User can search by ID (`turn_id`, event/message ID, tool call ID).

### 8.2 Turn and block snapshot inspection

FR-010: UI must display persisted turn snapshots from `/turns` with ordering controls (asc/desc by time).

FR-011: UI must decode YAML payload into structured turn model with key-family awareness.

FR-012: UI must show block list per snapshot with stable IDs, kind, role, payload, metadata.

FR-013: UI must compute and display snapshot-to-snapshot diffs:

- block inserted/removed,
- block reordered,
- payload field changed,
- metadata key changed,
- Turn.Data / Turn.Metadata changes.

FR-014: UI must support snapshot baseline selection (compare arbitrary A vs B, not only adjacent).

### 8.3 Middleware mutation attribution

FR-020: UI must support middleware invocation visualization (chain order and nesting).

FR-021: For each middleware step, UI must show before/after diff summary and affected block IDs.

FR-022: UI must support attribution confidence levels when direct middleware-level snapshots are unavailable (inferred vs explicit).

FR-023: UI must flag potentially dangerous mutations (ID loss, orphaned tool calls, metadata drift).

### 8.4 Event and SEM inspection

FR-030: UI must show event stream ordered by seq with source timestamps and correlation IDs.

FR-031: UI must decode typed event payloads (including custom events via registry metadata).

FR-032: UI must show SEM envelope details (`type`, `id`, `seq`, `stream_id`, `data`).

FR-033: UI must allow toggling raw JSON/proto-view/semantic summary view.

FR-034: UI must expose event lineage: source event -> SEM frame(s).

### 8.5 Projection and hydration inspection

FR-040: UI must display timeline snapshot version and entities from `/timeline`.

FR-041: UI must show incremental updates since selected version.

FR-042: UI must display entity-level upsert history where available, including `created_at_ms`, `updated_at_ms`, version.

FR-043: UI must correlate timeline.upsert events with projector writes and entity diffs.

FR-044: UI must explicitly indicate projector throttling/aggregation effects (e.g., coalesced deltas).

### 8.6 Structured sink and extraction workflows

FR-050: UI must show raw streamed text, filtered streamed text, and extracted payloads side by side.

FR-051: UI must visualize extractor sessions (`OnStart`, `OnRaw`, `OnCompleted`) and outcomes.

FR-052: UI must show malformed-policy behavior (error/reconstruct/ignore) with justification.

FR-053: UI must show capture byte limits and truncation outcomes when triggered.

### 8.7 Event design and schema evolution tools

FR-060: UI must include an event-schema explorer with payload examples and validation status.

FR-061: User must be able to pin two runs and compare event-shape changes by type.

FR-062: UI must detect unknown or unhandled event types and report sink/projection/UI handler gaps.

FR-063: UI should provide "new event checklist" hints:

- event registered?
- SEM mapping present?
- timeline mapping present?
- frontend renderer present?
- hydration compatibility verified?

### 8.8 Export and collaboration

FR-070: Export selected run debug package as JSON/Markdown bundle.

FR-071: Export visual diff snapshots as images for ticket/docs.

FR-072: Generate shareable permalink to current filter and selected artifact.

## 9. Instrumentation requirements (backend)

The current data is sufficient for phase-level inspection but insufficient for robust middleware attribution.

### 9.1 Snapshot model extensions (recommended)

IR-001: Extend `TurnSnapshot` persisted payload with structured metadata envelope:

- `snapshot_id` (UUID)
- `phase` (enum string)
- `phase_seq` (monotonic integer per run)
- `middleware_name` (optional)
- `cause` (`toolloop`, `middleware`, `persister`, `manual`)
- `correlation` (`session_id`, `inference_id`, `turn_id`)

IR-002: Persist optional compact diff against previous snapshot to accelerate UI.

IR-003: Add explicit ordering column in DB (`seq`) to avoid timestamp collision ambiguity.

### 9.2 Middleware-level hooks

IR-010: Introduce middleware wrapper instrumentation that can emit:

- `middleware.before(<name>)`
- `middleware.after(<name>)`

with turn snapshot metadata.

IR-011: Ensure wrapper can be enabled selectively by middleware name to control write volume.

IR-012: Include middleware execution duration and error state.

### 9.3 Correlation completeness

IR-020: Guarantee `session_id`, `inference_id`, and `turn_id` are set at every snapshot/event boundary.

IR-021: Normalize naming (`run_id` legacy alias retained, but canonical key = `session_id`).

### 9.4 Projection visibility

IR-030: Emit optional projector debug events for:

- throttle skip decisions,
- aggregation updates,
- unknown SEM event drop reasons.

IR-031: Add debug counters per run (events received, writes performed, writes skipped).

## 10. UX concepts and affordances for designer handoff

This section is intentionally explicit to help visual/interaction design produce a high-signal tool.

### 10.1 Concept A: "Forensic Timeline"

Primary interaction:

- horizontally aligned lanes (snapshots, middlewares, events, projection).
- selecting any node highlights correlated nodes in all lanes.
- right pane explains delta in plain language + raw diff.

Affordances:

- color coding by artifact class (turn/event/projection/sink).
- dashed connectors for inferred correlations, solid for explicit IDs.
- compact badges for phases and middleware names.

### 10.2 Concept B: "Turn Microscope"

Primary interaction:

- two snapshot panes (A/B) with synchronized block list.
- inline field-level diff, metadata key diff, and block movement arrows.

Affordances:

- "explain this diff" summary panel:
  - added blocks,
  - removed blocks,
  - reordered block IDs,
  - payload key changes,
  - typed-key changes.

### 10.3 Concept C: "Event Lab"

Primary interaction:

- event stream table with schema-aware rendering.
- per-event validation status and mapping path.

Affordances:

- toggle raw/decoded/proto-ish view.
- show unhandled events as red rail markers.
- compare two runs by event-type histogram and payload key diff.

### 10.4 Concept D: "Structured Sink Console"

Primary interaction:

- tri-column streaming viewer:
  - raw text,
  - filtered text,
  - extracted typed objects.

Affordances:

- live parser state (idle/capturing).
- malformed policy action banner.
- extractor session list with lifecycle marks.

### 10.5 Designer notes: visual language

- Favor dense information cards over large empty chat bubbles.
- Make IDs copyable with one click.
- Add sticky correlation chip row (session, inference, turn).
- Keep monospace for payload/diff/code views; humanized summaries for headers.
- Support dark/light, but prioritize readability of dense tables/diffs.

## 11. Proposed data contracts for the debug UI

### 11.1 Turn snapshot API (existing + proposed)

Current `/turns` response provides raw items with YAML payload string.

Recommended additive fields for future:

- `seq`
- `snapshot_id`
- `middleware_name`
- `cause`
- `session_id`
- `inference_id`

### 11.2 Middleware trace API (new)

Potential endpoint:

- `GET /middleware-trace?conv_id=&run_id=&turn_id=`

Response elements:

- middleware invocation list,
- before/after snapshot refs,
- duration,
- error state.

### 11.3 Event trace API (optional materialized index)

Potential endpoint:

- `GET /events?conv_id=&run_id=&since_seq=`

This can be synthesized from existing stream capture or explicit persistence.

### 11.4 Correlation graph model

Each artifact node should include:

- `node_id`
- `node_type`
- `session_id`
- `inference_id`
- `turn_id`
- `timestamp`

Edges should include:

- `edge_type` (`emits`, `mutates`, `projects_to`, `hydrates_from`, `derived_from`)
- `confidence` (`explicit`, `inferred`)

## 12. Non-functional requirements

NFR-001: UI must handle at least 10k events and 1k snapshots in a run without freezing.

NFR-002: Diff computation should be incremental/memoized for large turns.

NFR-003: Sensitive payload controls (redaction toggles, safe-copy mode) required before broad team rollout.

NFR-004: All debug artifacts should be exportable without requiring live backend connectivity.

NFR-005: Must support deterministic ordering even when timestamps are identical.

NFR-006: Feature flags for high-volume instrumentation in production-like environments.

## 13. Validation strategy

### 13.1 Golden scenarios

- Pure text inference (no tools).
- Tool-calling run with multiple iterations.
- Middleware-induced mode switch run.
- Structured sink extraction run with malformed payload.
- Planning + execution typed event run.
- Hydration replay with out-of-order buffered frames.

### 13.2 Assertions to expose in UI

- Pending tool calls resolved by matching tool_use IDs.
- No orphaned tool_result without tool_call.
- Event seq monotonicity in lane ordering.
- Projection version monotonicity.
- Snapshot phase progression consistency.

## 14. Phased implementation plan

### Phase 0: Read-only aggregator UI (fastest value)

- Consume existing `/turns`, `/timeline`, and live SEM stream.
- Implement run selector, lanes, snapshot diff viewer.
- No backend changes yet.

### Phase 1: Middleware attribution instrumentation

- Add middleware before/after snapshot hooks.
- Persist sequence and invocation metadata.
- Render middleware lane with explicit attribution.

### Phase 2: Structured sink and event-design workbench

- Add sink session visualization and malformed diagnostics.
- Add event-schema explorer and "unhandled event" detection panel.

### Phase 3: Correlation graph and export hardening

- Add graph view + permalink + export package.
- Integrate with ticket/report workflows.

## 15. Risks and mitigations

Risk: Instrumentation overhead increases write volume.
Mitigation: per-feature flags, sampling, and phase filtering.

Risk: Too much data overwhelms users.
Mitigation: progressive disclosure, defaults, saved views.

Risk: Correlation ambiguity without explicit middleware-level traces.
Mitigation: show confidence labels; prioritize Phase 1 hooks.

Risk: Sensitive content leakage in debug artifacts.
Mitigation: redaction policies, safe export mode, role-based access controls if deployed.

## 16. Open design questions

1. Should middleware-level snapshots be persisted always or only in debug mode?
2. Where should raw event persistence live (watermill sidecar vs webchat store)?
3. Should the debug UI run embedded in webchat or as a separate app route/package?
4. Is "projection write history" required, or is latest upsert view sufficient?
5. How aggressively should turn snapshot payloads be compressed/normalized?

## 17. Handoff checklist for web designer

The design handoff should include:

- A main debug workspace wireframe with 4 synchronized lanes.
- A detailed turn diff panel (block movement + key changes).
- An event payload inspector with schema tabs.
- A structured-sink console prototype (raw/filtered/extracted).
- State charts for loading, live streaming, hydration replay, and error modes.
- Visual treatment for confidence (explicit vs inferred links).
- Density modes (comfortable vs compact) for debugging sessions.

## 18. Suggested screen inventory (MVP)

1. Run Browser
- Find runs by conv/session/date and open workspace.

2. Run Workspace
- Multi-lane synchronized timeline and right inspector.

3. Snapshot Diff Modal
- Arbitrary snapshot A/B compare with semantic diff.

4. Event Lab
- Event stream table + schema status + payload tree.

5. Structured Sink Lab
- Streaming compare console + extractor lifecycle panel.

6. Export/Share Dialog
- Select artifact set and generate package/permalink.

## 19. Appendix A: Current phase/event mappings

### Tool-loop phases currently observed

- `pre_inference`
- `post_inference`
- `post_tools`
- `final` (via persister)

### Common SEM event families in webchat flow

- LLM: `llm.start`, `llm.delta`, `llm.final`, `llm.thinking.*`
- Tools: `tool.start`, `tool.delta`, `tool.result`, `tool.done`
- Planning: `planning.start`, `planning.iteration`, `planning.reflection`, `planning.complete`
- Execution: `execution.start`, `execution.complete`
- Middleware/UI support: `thinking.mode.*`, `agent.mode`, `debugger.pause`, `timeline.upsert`

## 20. Appendix B: Summary of key gaps to close

1. No explicit middleware-level before/after snapshots.
2. No sequence field in turn snapshots for deterministic ordering.
3. No first-class raw-vs-filtered-vs-extracted sink visualization.
4. No event schema explorer highlighting unhandled new types.
5. No unified UI connecting turn mutations to projection outcomes.

Addressing these will produce a high-leverage developer tool for middleware, sink, and event-design work.
