# Tasks

## Completed

- [x] Create ticket workspace `GP-002-TURNS-BLOCKS-SCHEMA`.
- [x] Write deferred planning analysis for normalized `turns + blocks` persistence.

## Deferred Implementation Tasks (Not started)

- [x] Finalize canonical hash material/rules for block content hashing.
- [ ] Implement sqlite schema migration for `turns`, `blocks`, and `turn_block_membership`.
- [ ] Implement payload backfill command from legacy `turns.payload` rows.
- [ ] Cut read/write paths to normalized tables and delete payload-only storage path.
- [ ] Add parity tests (legacy payload decode vs normalized rehydrate) and query benchmarks.
