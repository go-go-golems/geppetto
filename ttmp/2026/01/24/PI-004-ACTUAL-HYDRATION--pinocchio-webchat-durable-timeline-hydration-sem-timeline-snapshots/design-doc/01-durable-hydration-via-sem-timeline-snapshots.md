---
Title: Durable Hydration via sem.timeline Snapshots
Ticket: PI-004-ACTUAL-HYDRATION
Status: active
Topics:
    - backend
    - pinocchio
    - webchat
    - hydration
    - timeline
    - protobuf
    - websocket
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/router.go
      Note: Current GET /hydrate returns buffered SEM frames; baseline to replace/augment
    - Path: pinocchio/pkg/webchat/sem_buffer.go
      Note: Current in-memory SEM frame buffer used for hydration gating
    - Path: pinocchio/proto/sem/timeline/message.proto
      Note: sem.timeline snapshot schema for message entities
    - Path: pinocchio/proto/sem/timeline/middleware.proto
      Note: sem.timeline snapshot schema for middleware entities (thinking_mode
    - Path: pinocchio/proto/sem/timeline/status.proto
      Note: sem.timeline snapshot schema for status entities
    - Path: pinocchio/proto/sem/timeline/tool.proto
      Note: sem.timeline snapshot schema for tool entities
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-24T19:38:46.832334453-05:00
WhatFor: ""
WhenToUse: ""
---


# Durable Hydration via sem.timeline Snapshots

## Executive Summary

Pinocchio webchat currently “hydrates” by replaying a bounded buffer of **raw SEM frames** returned by `GET /hydrate` (see `pinocchio/pkg/webchat/router.go`). This is useful for reconnect gating, but it is not “actual hydration” in the sense of a durable, canonical snapshot of *UI state*:

- The replay history is limited (buffer size).
- The replay is not durable across server restarts.
- The UI must re-run the entire projection logic from scratch every time it reconnects.
- The server has no canonical representation of “the current timeline state” to serve quickly.

This design introduces a durable hydration mechanism that uses **`sem.timeline.*` snapshot payloads** (protobuf-defined, compiled to Go+TS) as the canonical UI projection layer. The backend maintains a **timeline projection store** (in-memory + optional persistence), serves a **snapshot endpoint** (`GET /timeline`), and can optionally stream **snapshot deltas** on the WebSocket alongside (or instead of) raw engine SEM frames.

The goal is: a reload/reconnect can reconstruct the timeline in O(number_of_entities) rather than O(number_of_events), while still preserving the current “SEM over WS” architecture and not requiring frontend retry queues.

## Problem Statement

### What “hydration” must accomplish

Hydration is the process by which a new client (or a client that refreshed/reconnected) obtains enough state to render the current conversation timeline, then safely transitions into consuming WebSocket deltas without:

- duplicating entities,
- missing early frames emitted during the reconnection window,
- relying on client-side retry queues,
- requiring the server to replay unbounded history.

### What we have today (and why it’s insufficient)

Today, Pinocchio’s `GET /hydrate` returns a snapshot of buffered SEM frames:

- It’s an **in-memory ring buffer** of JSON SEM envelopes (already projected/translated by the webchat translator).
- This works as a **reconnect gate** (hydrate-first, then WS).
- It does not provide a canonical, durable “current timeline state”.

In practice, this means:

- Reloading after a long conversation may lose earlier context (buffer limit).
- Restarting the server loses all hydration state (buffer cleared).
- “Event replay hydration” is computationally heavier and depends on correctness of frontend projection handlers.

### “Actual hydration” definition for this ticket

For this ticket, “actual hydration” means:

1) The backend owns a canonical timeline projection (entities + stable IDs + ordering + version).
2) The client can request a snapshot of entities and render immediately.
3) The client can then consume WS deltas *starting from a version/cursor*.
4) The snapshot protocol is protobuf-defined and compiled to Go and TS.

## Proposed Solution

We add a new **timeline projection subsystem** to Pinocchio webchat:

### A. Canonical snapshot payloads: `sem.timeline.*`

We standardize “snapshot state” as a separate family of payload schemas under `pinocchio/proto/sem/timeline/*.proto`:

- `sem.timeline.MessageSnapshotV1` (`message.proto`)
- `sem.timeline.ToolCallSnapshotV1`, `sem.timeline.ToolResultSnapshotV1` (`tool.proto`)
- `sem.timeline.StatusSnapshotV1` (`status.proto`)
- `sem.timeline.ThinkingModeSnapshotV1`, etc (`middleware.proto`)

These schemas describe **UI entity props**, not raw engine execution events. They are meant to be:

- stable,
- idempotent,
- forward-compatible (additive fields),
- sufficient for immediate rendering.

### B. Timeline entity envelope (server internal and on-the-wire)

We introduce a server-side “timeline entity” concept:

```text
TimelineEntity {
  id: string            // stable ID, used for upsert
  kind: string          // message|tool_call|tool_result|planning|thinking_mode|...
  created_at_ms: int64  // stable ordering anchor (or logical seq)
  updated_at_ms: int64
  props: oneof snapshot payload (sem.timeline.*) encoded as protojson
}
```

On the wire, we keep the existing JSON SEM envelope, but define a **snapshot type namespace**:

```json
{ "sem": true, "event": { "type": "timeline.upsert", "id": "<entity-id>", "data": { ... } } }
```

Where `event.data` is protojson for a “timeline upsert” message that includes:

- `kind`
- `created_at_ms` / `updated_at_ms`
- `snapshot` payload (oneof, or kind-specific)

Note: this ticket can either (a) define a new protobuf message `sem.timeline.TimelineUpsertV1`, or (b) treat each kind as its own snapshot message and ship it with a type like `timeline.message.upsert`, `timeline.tool_call.upsert`, etc. The “one envelope” option is usually easier for clients.

### C. Snapshot endpoint: `GET /timeline`

We add:

```text
GET /timeline?conv_id=...&since_version=...&limit=...
-> {
  conv_id,
  version,           // monotonic timeline version
  entities: [...],   // array of timeline.upsert SEM frames or raw entities
  server_time_ms
}
```

Properties:

- `version` increments on each entity upsert in the projection store.
- `since_version` allows incremental rehydration and pagination.
- response is idempotent and safe to replay.

### D. Projection store (in-memory + optional persistence)

We maintain a per-conversation store:

```text
TimelineStore {
  version: uint64
  by_id: map[string]TimelineEntity
  order: []string     // stable order for rendering
}
```

How it is updated:

- The stream coordinator (or a parallel projector) consumes engine SEM frames and updates the store via a projection function.
- For example:
  - `llm.*` → upsert message snapshot
  - `tool.*` → upsert tool call + result snapshots
  - `planning.*` → upsert planning snapshot

Persistence options (choose one in implementation):

1) **In-memory only** (fast; loses state on restart; still “actual” within process).
2) **SQLite per conversation** (durable; simplest local persistence; good for Pinocchio).
3) **Redis** (durable/shared; fits existing redisstreams optionality).

### E. Client hydration flow (no retry queue)

Client algorithm:

1) `GET /timeline?conv_id=...` → apply snapshot entities (idempotent upserts).
2) Connect WS with `since_version=<snapshot.version>` (or include a header/query).
3) WS streams deltas:
   - either raw engine SEM events (current design),
   - and/or `timeline.upsert` snapshots that update the projection incrementally.

The invariant is: the frontend should never need to implement “chatQueueSlice semantics”; the backend projection + idempotent upserts provide a simpler contract.

## Design Decisions

### Decision 1: `sem.timeline.*` represents *projection state*, not raw events

Rationale:
- Keeps the schema stable even when engine event shapes change.
- Allows snapshot-based hydration without replaying long event histories.

### Decision 2: Versioned snapshots (monotonic `version`)

Rationale:
- Avoids “since_seq” ambiguity across reconnects and server restarts.
- Enables incremental hydration and WS catch-up semantics.

### Decision 3: Keep JSON SEM envelope at the transport boundary

Rationale:
- Matches the rest of the system.
- Allows inspection/debugging via browser/devtools.
- Uses protobuf via protojson for strong schema ownership.

## Alternatives Considered

### Alternative A: Keep current `GET /hydrate` (frame replay) only

Rejected because it is not durable, bounded, and heavier to replay.

### Alternative B: Server emits only raw events; client is the canonical projector

Rejected because it creates correctness and performance problems:
- all clients must re-run the projection logic,
- server cannot provide a canonical “current state” snapshot,
- harder to evolve UI semantics without coordinated frontend changes.

### Alternative C: Protobuf binary over WS/HTTP

Rejected because it increases operational friction and removes easy inspectability. We keep JSON envelope + protojson payloads.

## Implementation Plan

1) **Define the transport for timeline snapshots**
   - Add `sem.timeline.TimelineUpsertV1` and `sem.timeline.TimelineSnapshotV1` protos (or choose per-kind event types).
   - Run `cd pinocchio && buf generate`.

2) **Implement a projection store**
   - In-memory projection per conversation.
   - Stable order and idempotent upsert semantics.

3) **Update webchat streaming path to update the projection**
   - Consume existing SEM frames and project into `sem.timeline.*` entities.
   - Keep current raw SEM broadcasting for now (dual-stream acceptable internally), but do not introduce frontend fallback logic.

4) **Add `GET /timeline` endpoint**
   - Returns snapshot + version (+ incremental support via `since_version`).

5) **Update frontend WS manager to hydrate via `GET /timeline`**
   - Apply snapshot entities.
   - Connect WS with `since_version`.

6) **(Optional) Persist projection store**
   - SQLite-backed store (append-only events or entity snapshots).
   - Ensure durability across server restart.

7) **Add tests**
   - Projection correctness tests (event stream → snapshot entities).
   - Version monotonicity.
   - Hydration + WS delta gating (no duplicates; no missing updates).

## Open Questions

1) Should the WS stream carry:
   - only raw engine SEM events, or
   - only `timeline.upsert` snapshots, or
   - both (raw for debug; timeline for UI)?

2) Should `GET /timeline` return:
   - a JSON array of entities, or
   - a JSON array of SEM frames (`{sem:true,event:{type:"timeline.upsert",...}}`)?

3) Persistence:
   - in-memory only (fastest to ship), or
   - SQLite (durable local), or
   - Redis (shared/durable)?

4) Ordering contract:
   - use server `version` for ordering, or
   - keep `created_at_ms` and let UI order by created time + stable tie-breaks?

## References

- `pinocchio/pkg/webchat/router.go` (`GET /hydrate`) and `pinocchio/pkg/webchat/sem_buffer.go` (current buffer-based gating)
- `pinocchio/proto/sem/timeline/*.proto` (snapshot payload schemas)
- Ticket PI-003 (React webchat + hydration gating baseline): `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/`
