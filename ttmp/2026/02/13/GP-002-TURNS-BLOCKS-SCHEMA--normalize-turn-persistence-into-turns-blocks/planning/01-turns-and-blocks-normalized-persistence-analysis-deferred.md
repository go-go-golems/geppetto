---
Title: Turns and Blocks Normalized Persistence Analysis (Deferred)
Ticket: GP-002-TURNS-BLOCKS-SCHEMA
Status: active
Topics:
    - backend
    - persistence
    - turns
    - architecture
    - migration
    - pinocchio
DocType: planning
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/persistence/chatstore/turn_store.go
      Note: Current turn snapshot contract showing payload-string storage
    - Path: pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go
      Note: Current sqlite turns schema with payload column and indexes
    - Path: pinocchio/pkg/webchat/turn_persister.go
      Note: Current persister write path and serialization behavior
    - Path: geppetto/pkg/turns/types.go
      Note: Canonical block and turn model used by normalized schema
ExternalSources: []
Summary: Deferred design analysis for replacing serialized turn payload storage with normalized turns + blocks persistence using block id + content hash identity.
LastUpdated: 2026-02-13T18:30:00-05:00
WhatFor: Define the future normalized schema and migration plan without implementing it during GP-001.
WhenToUse: Use when implementing GP-002 to replace current payload-string turn snapshots with queryable normalized rows.
---


# GP-002 Analysis: Normalize Turn Persistence Into Turns + Blocks (Deferred)

## Executive Summary

Yes, we can move to a normalized `turns + blocks` schema and use `(block_id, content_hash)` to avoid collisions when block IDs are reused with different content.

Recommendation for future implementation:

1. Replace `turns.payload TEXT` snapshots with normalized tables under a no-backwards-compatibility migration.
2. Keep `turns` as the logical conversation/session/turn identity.
3. Add `blocks` as deduplicated block content rows keyed by `(block_id, content_hash)`.
4. Add a relation table for ordered, phase-specific membership of blocks in each persisted turn snapshot.

This is deferred out of GP-001 because GP-001 is focused on debug UI migration/contract convergence and does not require storage normalization to ship.

## Current State (Why Change)

Current turn persistence stores each snapshot as one serialized YAML payload string:

- `TurnSnapshot.Payload` is a string (`turn_store.go`).
- sqlite `turns` table has `payload TEXT` and no blocks table (`turn_store_sqlite.go`).
- write path serializes full turns with `serde.ToYAML(...)` before insert (`turn_persister.go`).

Consequences:

1. Block-level filters/queries require deserialize-then-scan.
2. Duplicate block content is repeated across snapshots/phases.
3. Cannot index block traits (`kind`, tool name, role, etc.) natively.
4. Harder to build performant inspectors over large persisted datasets.

## Proposed Schema (Future)

## 1) `turns` (logical turn identity)

One row per `(conv_id, session_id, turn_id)`.

Suggested columns:

- `conv_id TEXT NOT NULL`
- `session_id TEXT NOT NULL`
- `turn_id TEXT NOT NULL`
- `turn_created_at_ms INTEGER NOT NULL`
- `turn_metadata_json TEXT NOT NULL DEFAULT '{}'`
- `turn_data_json TEXT NOT NULL DEFAULT '{}'`
- `updated_at_ms INTEGER NOT NULL`
- `PRIMARY KEY (conv_id, session_id, turn_id)`

## 2) `blocks` (content-addressed block rows)

One row per unique `(block_id, content_hash)`.

Suggested columns:

- `block_id TEXT NOT NULL`
- `content_hash TEXT NOT NULL` (sha256 over canonicalized block content)
- `kind TEXT NOT NULL`
- `role TEXT NOT NULL DEFAULT ''`
- `payload_json TEXT NOT NULL DEFAULT '{}'`
- `block_metadata_json TEXT NOT NULL DEFAULT '{}'`
- `first_seen_at_ms INTEGER NOT NULL`
- `PRIMARY KEY (block_id, content_hash)`

## 3) `turn_block_membership` (ordered phase snapshots)

Maps turn snapshots/phases to block rows in deterministic order.

Suggested columns:

- `conv_id TEXT NOT NULL`
- `session_id TEXT NOT NULL`
- `turn_id TEXT NOT NULL`
- `phase TEXT NOT NULL`
- `snapshot_created_at_ms INTEGER NOT NULL`
- `ordinal INTEGER NOT NULL`
- `block_id TEXT NOT NULL`
- `content_hash TEXT NOT NULL`
- `PRIMARY KEY (conv_id, session_id, turn_id, phase, snapshot_created_at_ms, ordinal)`
- `FOREIGN KEY (conv_id, session_id, turn_id) REFERENCES turns(...)`
- `FOREIGN KEY (block_id, content_hash) REFERENCES blocks(...)`

Required indexes:

1. `(conv_id, session_id, phase, snapshot_created_at_ms DESC)` for turn listing.
2. `(block_id, content_hash)` is primary for block lookup.
3. `(kind, role)` on `blocks` for inspector filters.

## Block Identity and Hashing Rules

Composite identity is `(block_id, content_hash)`.

Hash input should be canonical JSON over:

- `kind`
- `role`
- `payload`
- `metadata`

Hashing notes:

1. Canonicalize map key order before hashing.
2. Treat absent/empty payload as `{}`.
3. Use lowercase hex SHA-256.
4. Keep hash algorithm/version constant in schema docs.

Why not `block_id` alone:

- middlewares/providers can reuse IDs across updates;
- same ID with changed payload must not overwrite old content;
- composite key avoids accidental clashes while still deduplicating exact repeats.

## Read/Write Behavior

Write path (future):

1. Upsert `turns` row for logical identity.
2. For each block in order:
- canonicalize block JSON
- compute `content_hash`
- upsert into `blocks`
3. Insert `turn_block_membership` rows for the current phase snapshot.

Read path (future):

1. Query membership rows for requested turn/phase ordered by `ordinal`.
2. Join `blocks` by `(block_id, content_hash)`.
3. Rehydrate `Turn.Blocks[]` deterministically.
4. Return same API DTO shape expected by debug endpoints.

## Migration Strategy (No Backwards Compatibility)

When GP-002 is implemented:

1. Introduce new schema in-place (new tables + indexes).
2. Backfill from legacy `turns.payload` by:
- parsing YAML into `turns.Turn`
- writing normalized rows
3. Cut read paths over to normalized tables.
4. Delete legacy payload-column code and migration fallbacks.
5. Remove old indexes tied to payload-only access patterns.

No dual-read or long-lived compatibility shim is required per ticket direction.

## Why This Is Deferred From GP-001

GP-001 objective is debug UI migration into pinocchio (offline + live level-2) with clear API contracts. That can ship using current persistence as long as handlers decode payloads.

Doing storage normalization in GP-001 would increase scope substantially:

1. schema migration + backfill complexity,
2. new query codepaths and tests,
3. higher rollout risk unrelated to UI ownership/cutover.

So GP-002 exists to isolate that risk and sequence it after debug UI cutover.

## Risks and Mitigations (For GP-002)

1. Risk: hash instability due to non-canonical serialization.
- Mitigation: one canonical JSON routine + fixture tests with golden hashes.

2. Risk: metadata churn reduces dedupe ratio.
- Mitigation: start with full-fidelity hash; if needed, split stable/ephemeral metadata in a follow-up ticket.

3. Risk: backfill failures on malformed payload rows.
- Mitigation: produce per-row error report and continue backfill with explicit skip accounting.

4. Risk: query regressions.
- Mitigation: add SQL benchmarks and parity tests comparing legacy vs normalized rehydration for sampled datasets.

## Acceptance Criteria (When GP-002 Is Picked Up)

1. Legacy payload-only turn storage path removed.
2. Normalized schema (`turns`, `blocks`, `turn_block_membership`) is the only persistence path.
3. Rehydrated turns are parity-verified against legacy payload decode for golden fixtures.
4. Offline sqlite viewer queries block-level filters without full payload deserialization.
5. Performance/regression tests and migration/backfill runbook are documented.

## Out of Scope (This Ticket Now)

1. No implementation in GP-001.
2. No temporary compatibility layer.
3. No changes to timeline projection schema in this ticket.
