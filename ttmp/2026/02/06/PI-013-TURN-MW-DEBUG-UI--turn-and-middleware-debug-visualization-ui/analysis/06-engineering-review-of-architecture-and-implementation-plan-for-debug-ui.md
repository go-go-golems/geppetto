---
Title: Engineering Review of Architecture and Implementation Plan for Debug UI
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
    - Path: geppetto/ttmp/2026/02/06/PI-013-TURN-MW-DEBUG-UI--turn-and-middleware-debug-visualization-ui/analysis/05-architecture-and-implementation-plan-for-debug-ui.md
      Note: Proposal under review
    - Path: pinocchio/cmd/web-chat/web/src/App.tsx
      Note: Existing frontend entry-point architecture
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Existing hydration and live stream ingestion path
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Conversation and ConvManager state and locking model
    - Path: pinocchio/pkg/webchat/engine.go
      Note: Middleware composition order and built-in middleware insertion
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: Sink wrapping extension point and engine build path
    - Path: pinocchio/pkg/webchat/router.go
      Note: Existing API surface, debug gating, snapshot and persister wiring
    - Path: pinocchio/pkg/webchat/sem_buffer.go
      Note: Current SEM frame retention model (raw bytes only)
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: SEM envelope shape and metadata propagation limits
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: seq and stream_id enrichment model
    - Path: pinocchio/pkg/webchat/turn_persister.go
      Note: Existing final-phase persistence
    - Path: pinocchio/pkg/webchat/turn_store.go
      Note: Turn query model limitations
    - Path: pinocchio/pkg/webchat/turn_store_sqlite.go
      Note: Query ordering, limit defaults, and schema constraints
ExternalSources: []
Summary: Deep engineering review of proposal 05, including architecture fit, correctness gaps, risk assessment, and a corrected phased plan aligned with current Pinocchio and Geppetto internals.
LastUpdated: 2026-02-07T10:20:00-05:00
WhatFor: Validate feasibility and technical correctness before implementation starts, and prevent rework caused by incorrect assumptions about correlation, middleware tracing, and frontend/backend integration.
WhenToUse: Use this review as the engineering decision record before converting proposal 05 into implementation tickets.
---


# Engineering Review: Proposal 05

## Scope and review method

This review evaluates `analysis/05-architecture-and-implementation-plan-for-debug-ui.md` against actual implementation behavior in Geppetto and Pinocchio as of February 7, 2026.

Review method:

1. Verify each core proposal assumption against code.
2. Identify correctness and architecture-fit gaps.
3. Rank findings by severity (critical, high, medium).
4. Provide concrete remediation and a corrected phased plan.

## Executive verdict

The proposal is strong on product direction and UX ambition, and it captures the right problem space. However, there are several critical technical mismatches with the current runtime and data model that would cause rework if implementation starts as written.

Recommendation: proceed only after incorporating the critical and high-priority corrections in this review.

## What is strong in the proposal

- Clear phased rollout structure and screen-by-screen decomposition.
- Correct focus on middleware attribution, turn snapshots, and event/projection correlation as the core debug value.
- Good callout that structured sink visibility is a distinct track and can be deferred.
- Correct identification that current `TurnStore` and `semFrameBuffer` are useful starting points.

## Findings

### Critical 1: Correlation model in proposal does not match current SEM envelope reality

Problem:
The proposal assumes cross-highlighting can be driven by `event.seq` plus `inference_id` and `turn_id` present in event metadata. That is not true for current envelopes.

Evidence:

- Proposal assumption: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md:742`
- Proposal assumption: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md:805`
- Current SEM wrapper only includes `sem` and `event`: `pinocchio/pkg/webchat/sem_translator.go:41`
- Cursor enrichment adds only `seq` and optional `stream_id`: `pinocchio/pkg/webchat/stream_coordinator.go:229`
- Turn snapshots are time-based rows, not sequence-linked rows: `pinocchio/pkg/webchat/turn_store_sqlite.go:142`

Impact:
Cross-lane correlation will be unreliable or impossible for many event types. Implementation will either add brittle heuristics or silently produce wrong links.

Required remediation:

1. Add explicit correlation fields into SEM envelopes (for example `session_id`, `inference_id`, `turn_id`) at emission time.
2. Add a stable join key strategy between snapshots and event stream (event seq and/or explicit turn-phase snapshot sequence).
3. Treat this as Phase 0 contract work before UI implementation.

### Critical 2: Middleware tracing integration path is not compatible with current engine composition

Problem:
The proposal introduces tracing through an `r.debugEnabled` path and direct wrapping assumptions that do not match current composition flow.

Evidence:

- Proposed integration uses `r.debugEnabled`: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md:1006`
- `Router` has no `debugEnabled` field: `pinocchio/pkg/webchat/types.go:76`
- Current debug behavior is environment-gated per endpoint, not router-global state: `pinocchio/pkg/webchat/router.go:385`
- Actual middleware assembly is done in `composeEngineFromSettings`, which also injects built-ins (`tool result reorder`, system prompt): `pinocchio/pkg/webchat/engine.go:22`

Impact:
Tracing inserted at the wrong layer will miss built-in middleware or create inconsistent layer ordering versus runtime behavior.

Required remediation:

1. Implement tracing in the same path where middleware chain is materialized (`composeEngineFromSettings` or adjacent).
2. Include both requested and built-in middleware layers in trace output, with explicit `layer_kind` and `name`.
3. Use the existing env-gated debug model instead of introducing a new router flag unless deliberately refactored.

### Critical 3: Proposal treats `final` snapshot as missing, but final persistence already exists

Problem:
The proposal asks to add a fourth tool-loop snapshot phase `final`. Current system already persists `final` via persister.

Evidence:

- Proposal claim: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md:177`
- Tool-loop snapshot hook phases currently pre/post/post_tools: `geppetto/pkg/inference/toolloop/loop.go:126`
- Final persistence is already wired: `pinocchio/pkg/webchat/router.go:1026`
- Persister default phase is `final`: `pinocchio/pkg/webchat/turn_persister.go:55`

Impact:
If implemented literally, this creates duplicate or ambiguous final records and complicates diff logic.

Required remediation:

1. Keep current final persister behavior.
2. Document that `final` is a persisted phase, not a loop snapshot phase.
3. If exact phase lineage is needed, add a `source` field (`hook` vs `persister`) instead of duplicating records.

### High 1: Run-summary endpoint design risks incorrect output due to TurnStore query model

Problem:
Proposal suggests deriving runs from `turnStore.List(conv_id)` and grouping in memory.

Evidence:

- Proposal approach: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md:282`
- Default list limit is 200 unless overridden: `pinocchio/pkg/webchat/turn_store_sqlite.go:112`
- Rows are returned `ORDER BY created_at_ms DESC`: `pinocchio/pkg/webchat/turn_store_sqlite.go:145`

Impact:
Run summaries can silently omit older runs and produce misleading counts/timestamps.

Required remediation:

1. Add explicit `DistinctRuns` API in store with dedicated SQL, or
2. Add separate run index table, updated on save.

### High 2: Conversation detail endpoint can leak sensitive config and may not be serializable as proposed

Problem:
Proposal asks to store and return full `EngineConfig` from `Conversation`.

Evidence:

- Proposal change: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md:254`
- Current `Conversation` stores signature only: `pinocchio/pkg/webchat/conversation.go:39`
- `EngineConfig` includes `StepSettings`: `pinocchio/pkg/webchat/engine_config.go:17`
- Signature intentionally uses sanitized `StepMetadata`: `pinocchio/pkg/webchat/engine_config.go:25`

Impact:
Returning full config can expose provider credentials or internal runtime config and may produce unstable payloads.

Required remediation:

1. Define `DebugEngineConfigView` (sanitized) as explicit response DTO.
2. Populate from safe fields (`profile_slug`, `system_prompt`, middleware/tool names, step metadata only).

### High 3: Endpoint gating and route shape are inconsistent with current server behavior

Problem:
Proposal states all debug endpoints are gated and mounted under `/debug/*`, but existing `/turns` and `/timeline` are non-debug routes and part of normal API surface.

Evidence:

- Proposal statement: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md:187`
- Existing `/turns` route: `pinocchio/pkg/webchat/router.go:728`
- Existing `/timeline` route: `pinocchio/pkg/webchat/router.go:667`
- Existing debug gating only covers step control routes: `pinocchio/pkg/webchat/router.go:386`

Impact:
Mixed assumptions can create brittle frontend base URLs and unexpected behavior across dev/prod.

Required remediation:

1. Decide one of two models explicitly:
   - model A: keep `/turns` and `/timeline` public-internal, add new `/debug/*` only for new endpoints.
   - model B: migrate all diagnostic endpoints under `/debug/*` and provide compatibility aliases.
2. Capture this decision in API contract and tests.

### High 4: Frontend architecture plan underuses existing app foundation and may duplicate infra

Problem:
Proposal positions a separate SPA rooted at `/debug/`, but current frontend already has Redux, WS ingestion, timeline hydration, and SEM mapping.

Evidence:

- Proposal statement: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md:181`
- Existing app entry: `pinocchio/cmd/web-chat/web/src/App.tsx:1`
- Existing store and API middleware setup: `pinocchio/cmd/web-chat/web/src/store/store.ts:1`
- Existing WS + hydration + buffering path: `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts:56`

Impact:
A separate app can increase build/deploy complexity and duplicate state pipelines that already exist.

Required remediation:

1. Default to route-level module split in existing app (`/debug/*` in React Router), not separate SPA.
2. Reuse existing ws/timeline ingestion where possible and add debug slices incrementally.

### Medium 1: Proposed diff algorithm by index will misclassify moved blocks

Problem:
Diffing only by index does not preserve block identity across reorder operations.

Evidence:

- Proposal diff algorithm: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md:772`
- Middleware reorder exists by design: `pinocchio/pkg/webchat/engine.go:25`

Impact:
UI will report false add/remove changes where blocks were only moved.

Required remediation:

1. Use stable block identifiers where available (`block.id`, tool call IDs).
2. Fall back to content hashing and kind-role matching when IDs are missing.
3. Present reorder as first-class diff type.

### Medium 2: Event history strategy is insufficient for forensic/debug sessions

Problem:
Proposal relies on in-memory `semFrameBuffer` for `/debug/events`. Buffer is capped and reset on process lifecycle.

Evidence:

- Current buffer model: `pinocchio/pkg/webchat/sem_buffer.go:9`
- Capacity trimming behavior: `pinocchio/pkg/webchat/sem_buffer.go:33`

Impact:
Designers and developers will lose event history on restart or high-volume sessions, reducing value of a forensic tool.

Required remediation:

1. For MVP, mark `/debug/events` as live/ephemeral only in UI.
2. Plan optional durable `EventStore` for postmortem mode in later phase.

### Medium 3: `semFrameBuffer` current implementation can become hot under heavy traffic

Problem:
Current drop behavior reallocates/copies on overflow.

Evidence:

- Overflow path creates a new slice on every trim: `pinocchio/pkg/webchat/sem_buffer.go:35`

Impact:
High-rate streams can cause avoidable allocation churn.

Required remediation:

1. If debug endpoints depend on higher retention, replace with ring-buffer indexing instead of copy-trim.
2. Expose capacity and usage stats for observability.

### Medium 4: Proposal is missing concrete acceptance tests for correctness-critical paths

Problem:
Plan includes phases but not measurable correctness gates for correlation, ordering, and middleware attribution.

Impact:
Likely regressions in exactly the areas this UI needs to clarify.

Required remediation:

Add explicit acceptance tests:

1. Correlation contract test: event envelope includes session/inference/turn metadata and joins to snapshot API.
2. Ordering test: seq monotonicity preserved through stream/hydrate/replay.
3. Middleware trace test: known middleware chain yields expected pre/post deltas per layer.
4. Snapshot test: `final` appears once per inference and does not duplicate hook phases.

## Corrected implementation blueprint

### Phase 0: Observability contract alignment (new mandatory phase)

- Finalize correlation schema across events, turn snapshots, and timeline entities.
- Define sanitized debug DTOs (`ConversationDebug`, `EngineConfigDebug`, `TurnPhaseBundle`, `EventEnvelopeDebug`).
- Decide endpoint gating model and route namespace policy.

Exit criteria:

- JSON contract documented.
- At least one integration test proves event-turn correlation for a real run.

### Phase 1: Backend read APIs with contract-safe payloads

- Add `/debug/conversations`, `/debug/conversation/{id}`, `/debug/conversation/{id}/runs`.
- Add `/debug/events/{conv_id}` with explicit "ephemeral" semantics.
- Add `/debug/turn/{conv_id}/{run_id}/{turn_id}` backed by `TurnID` query support.
- Keep `/turns` and `/timeline` behavior stable unless migration is explicit.

### Phase 2: UI debug module inside existing frontend app

- Add debug routes to existing frontend project instead of separate SPA.
- Reuse existing WS manager and timeline hydration foundation.
- Ship screens in this order:
  - Session overview (minimal)
  - Turn inspector
  - Snapshot diff (identity-aware)

### Phase 3: Middleware tracing

- Instrument chain where middlewares are actually composed.
- Include built-in and configured middleware layers.
- Persist trace rows with correlation fields and layer metadata.
- Add `/debug/mw-trace` endpoints and golden tests.

### Phase 4: Structured sink tracing

- Implement sink tracing via existing event sink wrapper extension point:
  - `pinocchio/pkg/webchat/engine_builder.go:95`
  - `pinocchio/pkg/webchat/router_options.go:125`
- Capture raw vs filtered vs extracted streams with malformed-policy outcomes.
- Add `GET /debug/sink-trace/{conv_id}/{turn_id}`.

## Final recommendation

Proposal 05 should be treated as directionally correct but not implementation-ready.

Decision:

- Approve UX direction.
- Require architecture revision using this review before ticket breakdown and coding.

## Suggested immediate edits to proposal 05

1. Replace the current cross-highlighting section with a contract-first correlation section.
2. Remove "add 4th final snapshot phase" and document existing final persister.
3. Replace separate SPA assumption with route-module plan in existing frontend unless product explicitly wants hard separation.
4. Add explicit security section for debug payload sanitization.
5. Add acceptance tests and operational constraints (event history is ephemeral in MVP).
