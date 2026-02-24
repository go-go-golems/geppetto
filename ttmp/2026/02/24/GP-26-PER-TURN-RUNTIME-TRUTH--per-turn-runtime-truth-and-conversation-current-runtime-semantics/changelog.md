# Changelog

## 2026-02-24

- Initial workspace created


## 2026-02-24

- Added detailed implementation design and granular execution tasks for per-turn runtime truth.
- Clarified target semantics: per-turn runtime is authoritative; conversation runtime is current pointer only.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-26-PER-TURN-RUNTIME-TRUTH--per-turn-runtime-truth-and-conversation-current-runtime-semantics/design-doc/01-implementation-plan-per-turn-runtime-truth-and-conversation-current-runtime.md — Detailed implementation strategy and data-model decisions.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-26-PER-TURN-RUNTIME-TRUTH--per-turn-runtime-truth-and-conversation-current-runtime-semantics/tasks.md — Granular task checklist for schema, API, migration, testing, and rollout.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-26-PER-TURN-RUNTIME-TRUTH--per-turn-runtime-truth-and-conversation-current-runtime-semantics/index.md — Ticket overview updated with intent and links.

## 2026-02-24

Implementation diary update: audited GP-26 scope/tasks and inspected current code paths for turn store, turn persister, conversation index projection, and debug routes. Confirmed turn store API/schema still lacks first-class runtime_key/inference_id columns and identified remaining runtime_key conversation fields to rename to current_runtime_key semantics in debug surfaces with no compatibility wrappers.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/persistence/chatstore/turn_store.go — Current TurnSnapshot/TurnStore contract baseline
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go — Schema/migration baseline lacking runtime/inference columns
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/router_debug_routes.go — Conversation/runtime field naming and debug payload baseline
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/turn_persister.go — Save path currently not passing runtime/inference projection


## 2026-02-24

Backfilled implementation diary: created reference/01-diary.md and recorded GP-26 baseline audit + prompt-context entries.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-26-PER-TURN-RUNTIME-TRUTH--per-turn-runtime-truth-and-conversation-current-runtime-semantics/reference/01-diary.md — Added formal diary log for GP-26


## 2026-02-24

Implemented hard cutover commit d39acba: added turns.runtime_key + turns.inference_id columns/indexes/backfill, expanded TurnStore save contract, wired webchat+CLI persisters, and switched conversation debug field to current_runtime_key (no alias). Validation: go test ./pkg/persistence/chatstore ./pkg/webchat ./pkg/cmds -count=1 and go test ./... -count=1 in pinocchio.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go — Schema+backfill+list projection implementation
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/persistence/chatstore/turn_store_sqlite_test.go — Migration/backfill behavior tests
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/router_debug_routes.go — Conversation runtime field rename and turn detail enrichment

