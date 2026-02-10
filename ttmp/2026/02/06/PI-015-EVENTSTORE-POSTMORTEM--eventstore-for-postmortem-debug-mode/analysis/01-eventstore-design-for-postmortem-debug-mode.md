---
Title: EventStore Design for Postmortem Debug Mode
Ticket: PI-015-EVENTSTORE-POSTMORTEM
Status: active
Topics:
    - backend
    - events
    - persistence
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/sem_buffer.go
      Note: Current in-memory only SEM retention and limits
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Event->SEM frame production and sequencing
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: SEM envelope generation source
    - Path: pinocchio/pkg/webchat/router.go
      Note: Existing debug and query endpoints
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: Existing SQLite durability and snapshot query patterns
ExternalSources: []
Summary: Detailed design for a durable EventStore to support postmortem debug workflows beyond in-memory semFrameBuffer limits.
LastUpdated: 2026-02-07T11:00:00-05:00
WhatFor: Define architecture, schema, API, and rollout plan for persistent event history in debug mode.
WhenToUse: Use when implementing or reviewing postmortem debug support for webchat/SEM event streams.
---

# EventStore Design for Postmortem Debug Mode

## Why this ticket exists

Current MVP behavior keeps recent SEM frames in an in-memory circular buffer. This is acceptable for live debugging but insufficient for postmortem analysis:

- restarts lose history,
- high-throughput sessions drop older frames,
- no durable timeline for incident forensics.

This ticket defines a durable EventStore specifically for postmortem mode.

## Goals

1. Persist SEM envelopes durably for selected conversations/runs.
2. Preserve sequence and correlation fields for deterministic replay and joins.
3. Keep write path low-overhead so inference latency does not regress significantly.
4. Enable operator workflows: search, export, replay windows.

## Non-goals

- Replace timeline projector or timeline store.
- Store provider raw HTTP traces.
- Full observability backend replacement (this is debug-focused).

## Data model

### Entity: stored_event

Required fields:

- `conv_id` TEXT NOT NULL
- `session_id` TEXT NULL
- `inference_id` TEXT NULL
- `turn_id` TEXT NULL
- `event_id` TEXT NULL
- `event_type` TEXT NOT NULL
- `seq` INTEGER NOT NULL
- `stream_id` TEXT NULL
- `source` TEXT NOT NULL (`sem` | `timeline_upsert` | `debug`)
- `payload_json` TEXT NOT NULL (full envelope)
- `received_at_ms` INTEGER NOT NULL
- `ingested_at_ms` INTEGER NOT NULL

Primary key:

- `(conv_id, seq, source)`

Indexes:

- `(conv_id, seq)`
- `(conv_id, event_type, seq)`
- `(conv_id, inference_id, seq)`
- `(conv_id, turn_id, seq)`
- `(conv_id, received_at_ms)`

## Storage implementation

Proposed files:

- `pinocchio/pkg/webchat/event_store.go`
- `pinocchio/pkg/webchat/event_store_sqlite.go`

Interface:

```go
type EventStore interface {
    Save(ctx context.Context, ev StoredEvent) error
    List(ctx context.Context, q EventQuery) ([]StoredEvent, error)
    Export(ctx context.Context, q EventQuery, w io.Writer) error
    DeleteBefore(ctx context.Context, convID string, beforeMs int64) (int64, error)
    Close() error
}
```

## Ingestion architecture

### Recommended write path

Use async ingestion queue per conversation:

1. `StreamCoordinator` produces SEM envelope.
2. Envelope is pushed to bounded ingestion channel.
3. Dedicated writer goroutine batch-flushes to SQLite (`N` records or `T` ms).
4. On backpressure, apply policy (`drop_oldest` or `block`) configurable in debug mode.

Rationale:

- avoids synchronously blocking the event stream on each DB write,
- supports controlled backpressure behavior,
- keeps sequence ordering intact.

## Query/API surface

New debug endpoints:

- `GET /debug/events/history/:conv_id`
- `GET /debug/events/history/:conv_id/export`
- `DELETE /debug/events/history/:conv_id?before_ms=...`

Query params for `history`:

- `since_seq`
- `until_seq`
- `type`
- `inference_id`
- `turn_id`
- `limit`
- `order` (`asc` or `desc`)

## Postmortem workflows enabled

1. Incident replay
- fetch from `since_seq` around failure window,
- replay through debug UI timeline/event panes.

2. Drift diagnosis
- compare durable events vs snapshot store vs timeline projection.

3. Bug report export
- JSONL export keyed by `conv_id` + `seq` range.

## Performance and operational constraints

- default disabled unless debug/eventstore flag is enabled,
- batch writes + WAL mode mandatory,
- retention policy required (`max_age_days` and/or `max_rows_per_conv`),
- explicit metrics:
  - ingestion queue depth,
  - dropped events,
  - write latency p50/p95,
  - DB size growth.

## Rollout plan

### Phase A: minimal persistence

- schema + store implementation
- async ingestion from SEM envelopes
- basic history endpoint

### Phase B: export + retention

- export endpoint
- retention job and admin endpoint

### Phase C: UI integration

- postmortem mode toggle in debug UI
- fallback to in-memory mode when EventStore is unavailable

## Risks

1. Write amplification under high token streams.
Mitigation: batching + selective capture options (drop selected event types).

2. Disk growth.
Mitigation: retention and per-conv caps.

3. Ordering ambiguity if ingest pipeline reorders.
Mitigation: preserve `seq` ordering and keyed primary constraints.

## Acceptance criteria

1. Event history survives process restart.
2. Query by `conv_id + seq range` returns deterministic ordered results.
3. Export endpoint produces complete JSONL for selected window.
4. Inference runs show no material latency regression in benchmark mode.
5. Retention policy successfully trims old rows.
