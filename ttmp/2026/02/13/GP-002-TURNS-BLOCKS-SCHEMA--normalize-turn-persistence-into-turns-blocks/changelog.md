# Changelog

## 2026-02-13

- Initial workspace created
- Added deferred planning analysis for normalized `turns + blocks` schema with `(block_id, content_hash)` identity and no-backward-compatibility migration strategy.

## 2026-02-14

Step 1: finalized canonical block hash material/rules and added deterministic hash tests in pinocchio chatstore (commit 61ae8f23d31cd4528a13d65020988c9a8eea08c3).

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/block_hash.go — Define canonical JSON material and SHA-256 content hash rules
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/persistence/chatstore/block_hash_test.go — Prove determinism and change-sensitivity of hash computation

