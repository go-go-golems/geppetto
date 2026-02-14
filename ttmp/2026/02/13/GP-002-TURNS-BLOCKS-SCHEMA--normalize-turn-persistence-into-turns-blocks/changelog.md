# Changelog

## 2026-02-13

- Initial workspace created
- Added deferred planning analysis for normalized `turns + blocks` schema with `(block_id, content_hash)` identity and no-backward-compatibility migration strategy.

## 2026-02-14

Step 1: finalized canonical block hash material/rules and added deterministic hash tests in pinocchio chatstore (commit 61ae8f23d31cd4528a13d65020988c9a8eea08c3).

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/block_hash.go — Define canonical JSON material and SHA-256 content hash rules
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/block_hash_test.go — Prove determinism and change-sensitivity of hash computation


## 2026-02-14

Step 2: added normalized sqlite schema migration (`turns`, `blocks`, `turn_block_membership`) while preserving legacy snapshots in `turn_snapshots`; updated offline sqlite run scanning accordingly (commit da65342b58800ca440f9dcaf11e5c6c693b0b968).

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go — Create normalized tables and migrate legacy turns->turn_snapshots
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/turn_store_sqlite_test.go — Validate schema creation and legacy migration behavior
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/debug_offline.go — Query proper snapshot table for offline sqlite runs after schema migration

## 2026-02-14

Step 3: implemented snapshot payload backfill API and added `web-chat turns backfill` command to populate normalized tables from `turn_snapshots.payload` (commit c2058f6971ea8f86bdb3e83ad4b15740671bf1f4).

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/main.go — Registers turns command group under web-chat root CLI
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/turns/backfill.go — CLI surface for running backfill from sqlite turn snapshots
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/turns/turns.go — Adds turns command group and attaches backfill subcommand
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/turn_store_backfill.go — BackfillNormalizedFromSnapshots implementation with upsert logic for turns, blocks, and membership rows
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/turn_store_backfill_test.go — Backfill behavior tests (happy path, dry-run, parse-error continuation)

## 2026-02-14

Step 4: per directive (“we don't need legacy backfill code! you can kill it. we'll just start from fresh dbs.”), removed legacy/backfill codepaths and switched turn persistence/runtime inspection to normalized tables only (commit 19dae3b56df1d40e173c2365c64947443a0273f1).

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go — Normalized-only migration/save/list implementation (no turn_snapshots/backfill path)
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/turn_store_sqlite_test.go — Updated tests for normalized-only persistence behavior
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/debug_offline.go — Offline run scan now reads normalized tables
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/debug_offline_test.go — Fixtures updated for normalized-only turn snapshots
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/cmds/chat_persistence_test.go — CLI persister fixture updated for normalized-only snapshot behavior
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/main.go — Removed turns backfill command registration

## 2026-02-14

Step 5: added normalized read/write validation depth and a baseline list-query benchmark for fresh-db workflow (commit 0fe29f438db2d2db1b0c1ef2f1cdf3ff9e7ce7fe).

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/turn_store_sqlite_benchmark_test.go — Added BenchmarkSQLiteTurnStore_ListByConversation baseline benchmark for normalized query path
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/turn_store_sqlite_test.go — Added stronger rehydration assertion for normalized save/list payload shape


## 2026-02-14

Ticket closed

