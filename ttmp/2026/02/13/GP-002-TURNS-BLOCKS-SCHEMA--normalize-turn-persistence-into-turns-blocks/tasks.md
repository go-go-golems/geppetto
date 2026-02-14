# Tasks

## Completed

- [x] Create ticket workspace `GP-002-TURNS-BLOCKS-SCHEMA`.
- [x] Write deferred planning analysis for normalized `turns + blocks` persistence.

## Deferred Implementation Tasks

- [x] Finalize canonical hash material/rules for block content hashing.
- [x] Implement sqlite schema migration for `turns`, `blocks`, and `turn_block_membership`.
- [x] (Superseded) Implement payload backfill command from legacy `turns.payload` rows.
- [x] Cut read/write paths to normalized tables and delete payload-only storage path.
- [ ] Add normalized read/write validation and query benchmarks for fresh-db workflow.

## Decisions

- 2026-02-14: Legacy/backfill path removed by directive; migration now targets fresh DBs only.
