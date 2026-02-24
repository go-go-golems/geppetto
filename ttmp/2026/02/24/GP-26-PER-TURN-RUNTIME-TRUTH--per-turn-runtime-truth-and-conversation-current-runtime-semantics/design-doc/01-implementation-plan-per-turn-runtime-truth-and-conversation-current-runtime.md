---
Title: Implementation Plan - Per-turn Runtime Truth and Conversation Current Runtime
Ticket: GP-26-PER-TURN-RUNTIME-TRUTH
Status: active
Topics:
    - architecture
    - backend
    - chat
    - persistence
    - pinocchio
    - migration
DocType: design-doc
Intent: long-term
Owners: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-24T15:34:22.109563377-05:00
WhatFor: Align runtime persistence semantics with profile switching by making turns authoritative and conversation runtime explicitly current-only.
WhenToUse: Use when implementing schema, persistence, and API changes related to runtime/profile selection and turn history.
RelatedFiles:
    - Path: pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go
      Note: Turn persistence schema and migration path where per-turn runtime fields will live.
    - Path: pinocchio/pkg/persistence/chatstore/turn_store.go
      Note: TurnStore interface and TurnSnapshot API contract.
    - Path: pinocchio/pkg/persistence/chatstore/timeline_store_sqlite.go
      Note: Conversation projection storage that currently carries runtime_key.
    - Path: pinocchio/pkg/webchat/turn_persister.go
      Note: Runtime and metadata values are attached here before turn snapshots are saved.
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Conversation struct currently exposes RuntimeKey field used as latest runtime pointer.
    - Path: pinocchio/pkg/webchat/router_debug_routes.go
      Note: Debug APIs that surface conversation and turn persistence data.
---

# Implementation Plan - Per-turn Runtime Truth and Conversation Current Runtime

## Executive Summary

Runtime can change within a single conversation after profile selection changes. Therefore, conversation-level `runtime_key` cannot be the source of truth for historical analysis.

This design makes runtime truth explicit:

1. store runtime per turn as authoritative history,
2. keep conversation runtime as current pointer only,
3. expose this distinction in APIs, docs, and tests,
4. backfill existing rows so migration is deterministic.

Outcome: no ambiguity when debugging runtime switches, better analytics semantics, and cleaner composition guarantees.

## Problem Statement

Current behavior mixes two different concepts:

- "runtime used for this specific inference turn",
- "runtime currently selected for this conversation/session".

When profile changes happen mid-conversation, conversation-level `runtime_key` reflects only latest state and overwrites historical context. This causes:

- incorrect assumptions in debugging and support,
- fragile analytics ("which runtime generated this response?"),
- confusion for future features like per-profile starter suggestions and profile-scoped middleware defaults.

The model needs a hard semantic split.

## Proposed Solution

Introduce a two-level runtime model.

### 1) Authoritative per-turn runtime

Persist runtime on each turn row in `turns.db`.

New `turns` columns:

- `runtime_key TEXT NOT NULL DEFAULT ''`
- `inference_id TEXT NOT NULL DEFAULT ''`

`conv_id` is already first-class and remains part of primary key.

Runtime for a turn is set at persistence time from the conversation runtime active during that inference execution.

Inference ID is mirrored from typed turn metadata into first-class column for queryability.

### 2) Conversation runtime as current pointer

Conversation store keeps runtime as current-only pointer (latest selected/effective runtime). It is denormalized and should not be treated as history.

Option A: keep column name `runtime_key` and document semantics as "current runtime".

Option B: rename to `current_runtime_key` in conversation-facing APIs and docs (column rename optional; API aliasing possible).

Recommendation: introduce API/DTO naming `current_runtime_key` now to remove ambiguity. DB rename can be deferred if needed.

### 3) Query and API contract updates

Turn queries and debug endpoints should return per-turn runtime explicitly.

Example shape:

```json
{
  "conv_id": "c-1",
  "session_id": "s-1",
  "turn_id": "t-2",
  "phase": "final",
  "runtime_key": "planner",
  "inference_id": "inf-123",
  "created_at_ms": 1700000000000
}
```

Conversation summary shape:

```json
{
  "conv_id": "c-1",
  "session_id": "s-1",
  "current_runtime_key": "planner"
}
```

### 4) Backfill and migration behavior

Migration adds the columns with safe defaults.

Backfill strategy:

- `inference_id` from `turn_metadata_json['geppetto.inference_id@v1']` when present,
- `runtime_key` best-effort from `turn_metadata_json['geppetto.runtime@v1']` if available,
- otherwise fallback to conversation current runtime at migration time or empty string.

Backfill is opportunistic and documented as potentially incomplete for older data without metadata.

## Design Decisions

1. Per-turn runtime is canonical source of truth.
Rationale: only per-turn storage can represent mid-conversation switches.

2. Conversation runtime remains denormalized.
Rationale: fast list queries and bootstrap behavior still need current runtime without scanning turns.

3. First-class columns for runtime/inference are preferred over metadata-only.
Rationale: indexed filters and simpler SQL for debugging and operations.

4. Hard semantic cutover with no compatibility env toggles.
Rationale: avoid dual meaning and drift.

5. Keep metadata keys in payload for portability.
Rationale: serialized turn payloads remain self-contained and backwards-readable.

## Alternatives Considered

1. Keep runtime only in conversation table.
Rejected: loses turn-level truth after switches.

2. Keep runtime only in `turn_metadata_json`.
Rejected: difficult to query/index and slower for debug endpoints.

3. Duplicate full runtime history in conversation table JSON blob.
Rejected: denormalized history is harder to maintain and reason about than turn-local persistence.

4. Introduce compatibility mode with old and new semantics.
Rejected: prolongs confusion and doubles test matrix.

## Implementation Plan

### Phase 1: Storage and schema

1. Add `runtime_key` and `inference_id` columns to `turns`.
2. Add migration logic in `ensureTurnsTableColumns`.
3. Add indexes:
- `turns_by_conv_runtime_updated (conv_id, runtime_key, updated_at_ms DESC)`
- `turns_by_conv_inference_updated (conv_id, inference_id, updated_at_ms DESC)`
4. Add migration/backfill SQL and tests for existing DB upgrades.

### Phase 2: Persistence wiring

1. Extend turn-save call path to carry runtime key and inferred inference id.
2. Update `turnStorePersister` to pass conversation runtime.
3. Keep metadata-to-column sync in `persistNormalizedSnapshot` fallback logic.
4. Ensure snapshot load/list includes runtime and inference fields in `TurnSnapshot`.

### Phase 3: API semantics and naming

1. Add `runtime_key` and `inference_id` to debug turn list responses.
2. Update conversation summary/detail responses to expose `current_runtime_key`.
3. Document field semantics in comments and docs.
4. Update frontend debug types if needed.

### Phase 4: Behavior and invariants

1. Guarantee per-turn runtime is set before persistence on each inference.
2. Continue updating conversation current runtime on profile selection and runtime rebuild.
3. Add tests for mid-conversation switch sequence:
- turn1 runtime `inventory`
- profile switch
- turn2 runtime `planner`
- conversation current runtime `planner`.

### Phase 5: Migration and rollout

1. Run migrations against existing local fixture DBs.
2. Verify old rows are backfilled where possible and new rows always populated.
3. Update migration playbook and troubleshooting docs.
4. Cutover without compatibility flags.

### Phase 6: Validation

1. Unit tests for schema and backfill.
2. Integration test in Pinocchio chat flow.
3. Optional parity test in Go-Go-OS if debug endpoints consume fields.
4. Manual SQL validation checklist.

Manual SQL smoke example:

```sql
SELECT turn_id, runtime_key, inference_id, updated_at_ms
FROM turns
WHERE conv_id = '93a0cf71-39f9-4df3-b60d-94ce12330014'
ORDER BY updated_at_ms ASC;
```

Expected: runtime changes visible per row across the switch point.

## Open Questions

1. Should we rename DB column `timeline_conversations.runtime_key` to `current_runtime_key` now, or keep DB stable and rename at API layer first?
2. Do we want to persist `profile_version` per turn alongside runtime for stronger reproducibility?
3. Should per-turn runtime become mandatory non-empty for all new writes, with explicit error if unavailable?
4. Should we expose runtime-switch events explicitly in timeline stream to simplify UI inspection?

## References

- `pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go`
- `pinocchio/pkg/persistence/chatstore/turn_store.go`
- `pinocchio/pkg/persistence/chatstore/timeline_store_sqlite.go`
- `pinocchio/pkg/webchat/turn_persister.go`
- `pinocchio/pkg/webchat/conversation.go`
- `pinocchio/pkg/webchat/router_debug_routes.go`
