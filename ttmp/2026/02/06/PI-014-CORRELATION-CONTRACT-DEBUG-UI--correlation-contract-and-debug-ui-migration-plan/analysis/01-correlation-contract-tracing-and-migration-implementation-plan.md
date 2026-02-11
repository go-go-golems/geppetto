---
Title: Correlation Contract, Tracing, and Migration Implementation Plan
Ticket: PI-014-CORRELATION-CONTRACT-DEBUG-UI
Status: active
Topics:
    - backend
    - middleware
    - turns
    - events
    - frontend
    - websocket
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Current seq/stream_id assignment and SEM envelope enrichment
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Current SEM envelope shape and event translation
    - Path: pinocchio/pkg/webchat/router.go
      Note: Existing route layout and turn/timeline endpoints
    - Path: pinocchio/pkg/webchat/turn_store.go
      Note: Turn query model and storage contract
    - Path: pinocchio/pkg/webchat/turn_store_sqlite.go
      Note: Query limits and sorting behavior
    - Path: pinocchio/pkg/webchat/turn_persister.go
      Note: Final snapshot persistence path
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: SnapshotHook invocation phases
    - Path: pinocchio/pkg/webchat/engine.go
      Note: Middleware chain assembly and built-ins
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: BuildFromConfig and sink wrapping extension point
    - Path: geppetto/ttmp/2026/02/06/PI-013-TURN-MW-DEBUG-UI--turn-and-middleware-debug-visualization-ui/analysis/05-architecture-and-implementation-plan-for-debug-ui.md
      Note: Design doc to align and update
ExternalSources: []
Summary: Detailed implementation plan for inference-scoped event-to-snapshot correlation, tracing architecture, endpoint migration, separate debug app location, and design-doc updates required by PI-013 follow-up decisions.
LastUpdated: 2026-02-07T11:25:00-05:00
WhatFor: Provide implementation-ready engineering guidance for Critical 1 and related High/Medium decisions from the PI-013 review.
WhenToUse: Use before starting implementation of debug correlation, middleware tracing, and endpoint/frontend migration work.
---

# Correlation Contract, Tracing, and Migration Implementation Plan

## Scope and explicit decisions

This ticket captures and operationalizes the following decisions:

1. Critical 1: implement a first-class correlation/join contract between events and snapshots.
2. High 3: migrate timeline/turn APIs under `/debug/*` with no backwards compatibility.
3. High 4: build a fully separate debug app at `web-agent-example/cmd/web-agent-debug`.
4. Medium 1: update the design document with identity-aware diffing.
5. Medium 4: update the design document with acceptance tests.
6. High 2: explicitly out of scope (sanitization concerns deferred).

## Problem statement (Critical 1)

Current data can be inspected, but reliable joining is weak:

- turn snapshots in `TurnStore` are time-indexed rows;
- SEM envelopes carry `seq` and `stream_id` but do not carry a full explicit correlation object;
- UI often has to infer relationships heuristically.

For debugging middleware and timeline behavior, implicit joins are not acceptable. We need deterministic joins.

## Correlation contract (target)

### Envelope contract v2

All outgoing SEM envelopes used by debug views MUST include a `correlation` object:

```json
{
  "sem": true,
  "event": {
    "type": "llm.delta",
    "id": "...",
    "seq": 1707053365100000000,
    "stream_id": "1707053365100-0",
    "data": {"...": "..."}
  },
  "correlation": {
    "conv_id": "conv_...",
    "session_id": "sess_...",
    "inference_id": "inf_...",
    "turn_id": "turn_..."
  }
}
```

Rules:

- `conv_id`: required for debug routes.
- `session_id`, `inference_id`, `turn_id`: required when available; empty string only when unknown.
- `correlation` is inference-scoped; it does not encode middleware snapshot phase metadata.

### Snapshot contract v2

Extend snapshot records with deterministic join fields:

- `source` (`hook` or `persister`)
- `inference_id` (nullable)
- `seq_hint` (last known event seq at capture time, nullable)

## Join model

Primary key strategy in UI/backend APIs:

1. Primary join key: `(conv_id, session_id, inference_id, turn_id)`.
2. Secondary temporal key: `event.seq` to `seq_hint` proximity.
3. Snapshot phase remains a snapshot-store concern only (view/filter in snapshot APIs, not in event correlation).

No index-position-based correlation is allowed for core joins.

## Backend implementation plan

### A. Correlation emission

Files:

- `pinocchio/pkg/webchat/stream_coordinator.go`
- `pinocchio/pkg/webchat/sem_translator.go`
- `pinocchio/pkg/webchat/router.go`

Work:

1. Extend SEM envelope builder to attach `correlation` object.
2. Populate correlation from event metadata (`session`, `inference`, `turn`) and conversation context (`conv_id`).
3. Ensure seq assignment remains monotonic and unchanged.

### B. Snapshot enrichment

Files:

- `pinocchio/pkg/webchat/turn_store.go`
- `pinocchio/pkg/webchat/turn_store_sqlite.go`
- `pinocchio/pkg/webchat/turn_persister.go`
- `pinocchio/pkg/webchat/router.go`

Work:

1. Add new snapshot fields in store model and schema migration.
2. Keep existing `final` persister path, but mark `source=persister`.
3. SnapshotHook captures continue with `source=hook`, and also emit `phase=final` snapshot at completion.
4. Add strict ordering guarantees on retrieval (`created_at_ms`, tie-breaker `id`).

### C. Endpoint migration (High 3, no compatibility)

Decision:

- Remove `/turns` and `/timeline` legacy API surface.
- New canonical endpoints:
  - `GET /debug/turns`
  - `GET /debug/timeline`

Migration plan:

1. Update router registrations.
2. Update all frontend consumers.
3. Update docs and examples.
4. Do not provide aliases.

### D. Separate debug app location (High 4)

Decision:

- Build separate app at `web-agent-example/cmd/web-agent-debug`.

Work breakdown:

1. Create independent Vite/React app in that path.
2. Use RTK/RTK Query state and API patterns consistent with existing webchat frontend guidelines.
3. Reuse the same CSS/styling framework and design-token conventions as existing webchat.
4. Keep webchat app unchanged except API backend additions.
5. Run debug app independently in dev and build as standalone artifact.

## Critical 2 tracing design (explicit answer)

### Question

Is tracing done by inserting one tracing middleware, or multiple wrappers at different chain points?

### Answer

Use per-layer wrappers generated during chain composition.

- Not one global middleware at the outer edge.
- Wrap each middleware layer with a tracer wrapper that captures:
  - `pre` turn clone,
  - call duration,
  - `post` turn,
  - error.

This behaves like multiple generated wrappers, one for each layer index, preserving exact chain topology (including built-ins).

Implementation note:

- Instrument at middleware assembly point (`composeEngineFromSettings` path), so built-in middlewares are represented in trace output.

## Critical 3 persister/store relation (explicit answer)

### Question

Does the final persister write to the same store tracing would use?

### Answer

No, by design they should be separate stores.

- Final persister writes final turn snapshots to `TurnStore`.
- Middleware tracing writes per-layer before/after records to `MiddlewareTraceStore`.

Rationale:

- different cardinality and query patterns,
- avoids overloading snapshot table with tracing payload explosion,
- keeps tooling simple (`TurnStore` for phase snapshots, `MiddlewareTraceStore` for chain diffs).

## SnapshotHook call path (explicit answer)

### Question

How is `SnapshotHook` called?

### Answer

Call chain:

1. Router prompt entrypoint (currently named `startRunForPrompt` in code; legacy symbol) constructs `hook := snapshotHookForConv(...)` and injects it into `enginebuilder.Builder`.
2. Tool loop executes in `geppetto/pkg/inference/toolloop/loop.go`.
3. Loop calls `l.snapshot(ctx, t, phase)` at:
   - `pre_inference`
   - `post_inference`
   - `post_tools`
   - `final`
4. `l.snapshot` calls `snapshotHook` if present (or context hook fallback).
5. Separately, final turn persistence is still done via `Persister` (`phase=final`).

## High 1 better design (explicit answer)

### Question

What is the better design for session summaries?

### Recommendation

Add a dedicated `DistinctSessions` store method with SQL aggregation now, and reserve materialized session tables for later if needed.

Why:

- avoids incorrect summaries from `List(... limit 200)` behavior,
- low implementation risk,
- gives exact `count/min/max` data directly from DB.

Suggested SQL shape:

```sql
SELECT session_id,
       COUNT(*) AS snapshot_count,
       MIN(created_at_ms) AS first_snapshot_ms,
       MAX(created_at_ms) AS last_snapshot_ms
FROM turns
WHERE conv_id = ?
GROUP BY session_id
ORDER BY last_snapshot_ms DESC;
```

## Design document updates required in PI-013

Update `analysis/05-architecture-and-implementation-plan-for-debug-ui.md` with these edits:

1. replace index-based block diff with identity-aware/reorder-aware diff.
2. add acceptance tests section (correlation, ordering, trace coverage, migration checks).
3. set frontend location to `web-agent-example/cmd/web-agent-debug`.
4. set endpoint migration policy to `/debug/turns` + `/debug/timeline` only (no compatibility).
5. call `SnapshotHook` with `phase=final` in addition to the `Persister` final write.

## Phase plan

### Phase 1: Contract and storage

- correlation envelope v2
- snapshot schema v2 (`source`, `inference_id`, `seq_hint`, join fields)
- `DistinctSessions` API

### Phase 2: Endpoint migration

- ship `/debug/turns` and `/debug/timeline`
- remove legacy routes
- update backend tests

### Phase 3: Tracing store and wrappers

- per-layer tracing wrappers
- `MiddlewareTraceStore` schema + endpoints

### Phase 4: Separate debug app bootstrapping

- scaffold `web-agent-example/cmd/web-agent-debug`
- basic list/inspect/diff flows

### Phase 5: Validation and hardening

- acceptance test matrix
- docs sync in PI-013 design doc

## Acceptance criteria

1. Every debug event exposed to UI has deterministic correlation fields.
2. Snapshot->event join works without fuzzy heuristics in golden test fixtures.
3. `/turns` and `/timeline` legacy routes are removed.
4. Middleware chain trace includes built-in and configured middlewares in actual execution order.
5. PI-013 design doc reflects migrated decisions.

## Out of scope

- High 2 security/sanitization hardening (explicitly deferred).
- Durable EventStore postmortem persistence (handled in PI-015).
- full SEM pipeline perf program (handled in PI-016).
