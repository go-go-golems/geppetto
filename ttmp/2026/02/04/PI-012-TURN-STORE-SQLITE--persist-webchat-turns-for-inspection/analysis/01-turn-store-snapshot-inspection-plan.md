---
Title: Turn Store + Snapshot Inspection Plan
Ticket: PI-012-TURN-STORE-SQLITE
Status: active
Topics:
  - backend
  - webchat
  - sqlite
  - debugging
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
---

# Turn Store + Snapshot Inspection Plan

## Goal

Make it possible to inspect the exact LLM input blocks (system/user/assistant/tool) by persisting turns. Provide two inspection paths:

1. **Immediate snapshots** via an env-controlled hook (already present, but not integrated into the runbook).
2. **Durable SQLite turn store** (new) for structured queries and later analysis, independent of the timeline store.

## Background and Current Flow

**Where turns live today**

- Webchat creates/updates a `turns.Turn` inside `geppetto/pkg/inference/session`.
- The **run loop** (`enginebuilder.Builder`) accepts a `TurnPersister` that can be called after a successful run.
- Webchat already has a **snapshot hook** (file-based) wired through `PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR` in `pinocchio/pkg/webchat/router.go`.

**Why this matters**

- Timeline entities are derived from SEM events and are often lossy.
- When we want to verify whether a middleware actually injected a system prompt, we need to see the **actual blocks** that went to the engine.

## Existing Snapshot Hook (File-Based)

There is already a snapshot hook:

- `snapshotHookForConv` in `pinocchio/pkg/webchat/router.go`
- Enabled by setting `PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR`.
- Writes YAML turns to: `<dir>/<conv_id>/<run_id>/<timestamp>-<phase>-<turn_id>.yaml`

This is good for immediate debugging, but not queryable and not durable across environments.

## Proposed SQLite Turn Store

### Design Overview

We add a `TurnStore` in `pinocchio/pkg/webchat` alongside the existing `TimelineStore`.

The store will accept `turns.Turn` snapshots and persist them as YAML (or JSON) with metadata for quick lookup.

**Flow diagram**

```
[Chat Request]
   -> build Turn (system/user blocks)
   -> middleware pipeline
   -> engine
   -> updated Turn
        -> TurnPersister (final turn)
        -> SnapshotHook (per-phase)
             -> TurnStore
```

### API Sketch

```go
// pinocchio/pkg/webchat/turn_store.go

type TurnStore interface {
    Save(ctx context.Context, convID, runID, turnID, phase string, createdAtMs int64, payload []byte) error
}

// WithTurnStore wires a TurnStore into the router.
func WithTurnStore(store TurnStore) RouterOption
```

**Notes**

- `phase` will capture `pre|post|final|...` from the snapshot hook.
- `payload` is YAML (via `turns/serde.ToYAML`) to preserve block structure + metadata.

### SQLite Schema

```sql
CREATE TABLE IF NOT EXISTS turns (
    conv_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    turn_id TEXT NOT NULL,
    phase TEXT NOT NULL,
    created_at_ms INTEGER NOT NULL,
    payload TEXT NOT NULL,
    PRIMARY KEY (conv_id, run_id, turn_id, phase, created_at_ms)
);

CREATE INDEX IF NOT EXISTS idx_turns_conv ON turns (conv_id, created_at_ms DESC);
CREATE INDEX IF NOT EXISTS idx_turns_run ON turns (run_id, created_at_ms DESC);
```

### Retrieval / Inspection

We will optionally add a `/turns` endpoint for debugging, analogous to `/timeline`:

```
GET /turns?conv_id=...&limit=...&phase=...&since_ms=...
```

Response (JSON):

```json
{
  "conv_id": "...",
  "items": [
    {
      "turn_id": "...",
      "phase": "pre",
      "created_at_ms": 123,
      "payload": "<YAML>"
    }
  ]
}
```

This endpoint is strictly for debugging and can be disabled by leaving the store unconfigured.

## Implementation Plan

### Phase 1: Snapshots for immediate inspection

- Add a runbook note for setting `PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR`.
- Update tmux launch to include the env var.

### Phase 2: Turn store (SQLite)

- Add `TurnStore` interface and SQLite implementation.
- Add config flags: `turns-dsn`, `turns-db`.
- Wire store to:
  - `SnapshotHook` for per-phase snapshots.
  - `Builder.Persister` for final turn snapshots.

### Phase 3: Debug endpoint

- Add `GET /turns` handler in `webchat.Router`.
- Document usage.

## Risks and Constraints

- Snapshot payloads can be large (YAML). We should cap or allow pruning if size becomes a problem.
- Storing **every phase** might be excessive; we can gate phases (default: `final` only) or allow config.
- The endpoint should not be used for user-facing UI (debug only).

## Pseudocode (Wiring)

```go
store := NewSQLiteTurnStore(dsn)

hook := func(ctx context.Context, t *turns.Turn, phase string) {
    payload := serde.ToYAML(t)
    store.Save(ctx, conv.ID, conv.RunID, t.ID, phase, time.Now().UnixMilli(), payload)
}

builder.Persister = TurnStorePersister{Store: store, Phase: "final"}
```

## Acceptance Criteria

- We can turn on file snapshots and see injected blocks for a given `conv_id`.
- SQLite store captures at least the final turn for each run.
- Optional `/turns` endpoint returns YAML payloads for debugging.
- Documentation clarifies how to enable/inspect turns.
