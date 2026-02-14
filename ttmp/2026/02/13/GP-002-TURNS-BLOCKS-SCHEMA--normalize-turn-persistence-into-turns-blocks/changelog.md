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
