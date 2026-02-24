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


## 2026-02-24

Follow-up cutover alignment: commit 79d3516 updated web debug UI to current_runtime_key, commit c0084df updated go-go-os integration helper, and commit e47df35 added runtime-switch per-turn persistence tests (final persister + snapshot hook). Validation rerun: go test ./pkg/webchat ./pkg/persistence/chatstore ./cmd/web-chat -count=1, vitest debugApi test, and go-go-os integration package tests.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go — Conversation debug payload parser update
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts — Conversation current_runtime_key client parsing
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/turn_persister_runtime_test.go — Runtime-switch regressions now covered


## 2026-02-24

Documentation completion commit dbdcd12: added runtime-truth migration playbook, SQL validation queries, backfill troubleshooting, debug API runtime history snippets, and current_runtime_key terminology updates.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/doc/topics/webchat-debugging-and-ops.md — Runtime history operational checks
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/doc/topics/webchat-profile-registry.md — Conversation pointer vs per-turn truth guidance
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/doc/topics/webchat-runtime-truth-migration-playbook.md — Primary GP-26 migration/reference playbook


## 2026-02-24

Closeout block: commit 6678135 added go-go-os runtime-switch integration test with per-turn runtime assertions; validated /tmp/turns.db and /tmp/timeline3.db migration/runtime state; ran docmgr doctor (all checks passed); linked GP-26 follow-up from GP-24/GP-25 ticket docs; set GP-26 status to complete.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/index.md — Cross-ticket outcome link
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-25-MIGRATION-DOCS-RELEASE--migration-tooling-docs-and-release/index.md — Cross-ticket outcome link
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-26-PER-TURN-RUNTIME-TRUTH--per-turn-runtime-truth-and-conversation-current-runtime-semantics/index.md — Ticket status finalized to complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go — App-level runtime-switch persistence assertions

