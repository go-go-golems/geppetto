# Tasks

## Schema and Migration

- [x] Add `runtime_key` column to `turns` table with non-null default in sqlite migration.
- [x] Add `inference_id` column to `turns` table with non-null default in sqlite migration.
- [x] Extend `ensureTurnsTableColumns` with additive migration guards for both new columns.
- [x] Add migration SQL to backfill `inference_id` from `turn_metadata_json`.
- [x] Add migration SQL to best-effort backfill `runtime_key` from turn metadata runtime key.
- [x] Decide fallback behavior when runtime metadata is missing during backfill.
- [x] Add index `turns_by_conv_runtime_updated`.
- [x] Add index `turns_by_conv_inference_updated`.
- [x] Add unit test for migrating an old DB missing both columns.
- [x] Add unit test for backfill behavior with metadata present.
- [x] Add unit test for backfill behavior with metadata absent.

## TurnStore API and Model

- [x] Extend `chatstore.TurnSnapshot` with `RuntimeKey` field.
- [x] Extend `chatstore.TurnSnapshot` with `InferenceID` field.
- [x] Extend `TurnStore.Save` contract to accept save options for runtime/inference projection.
- [x] Update all `TurnStore` implementations to match the new save signature.
- [x] Keep backward payload compatibility by preserving metadata JSON projection.
- [x] Add tests that `List` returns runtime and inference columns correctly.

## Persistence Wiring

- [x] Update `turnStorePersister` to pass conversation runtime key explicitly on save.
- [x] Add inference-id extraction fallback from `turns.KeyTurnMetaInferenceID`.
- [x] Ensure turn save path sets runtime key for both final-persister and snapshot hook paths.
- [x] Ensure empty runtime handling follows policy (error or empty sentinel).
- [x] Add tests for persister behavior when runtime changes mid-conversation.

## Conversation Semantics Cleanup

- [x] Define canonical naming: `current_runtime_key` for conversation-level runtime pointer.
- [x] Update conversation response DTOs to expose `current_runtime_key`.
- [x] Preserve old `runtime_key` response alias only if required by callers, otherwise remove.
- [ ] Document that conversation runtime is latest pointer, not history.
- [x] Ensure conversation index persistence updates current runtime on profile switch.

## Debug API and Query Surface

- [x] Add per-item `runtime_key` and `inference_id` to `/api/debug/turns` response payload.
- [x] Add per-phase `runtime_key` and `inference_id` to `/api/debug/turn/:conv/:session/:turn` payload.
- [x] Update debug API tests for new response fields.
- [x] Update any frontend debug parsers/types that assume old shape.
- [x] Add API documentation snippets for querying runtime history by turn.

## Runtime Switch Correctness

- [ ] Add integration test: start with runtime `inventory`, send turn, verify persisted turn runtime.
- [ ] Add integration test: switch profile/runtime to `planner` in same conversation.
- [ ] Add integration test: send second turn and verify persisted runtime switched to `planner`.
- [ ] Verify conversation current runtime is `planner` after switch.
- [ ] Verify first turn remains `inventory` and is not rewritten.
- [ ] Add regression test to prevent turn runtime overwrite by subsequent turns.

## Migration Playbook and Docs

- [x] Update migration playbook with semantic model (`turn runtime` vs `conversation current runtime`).
- [x] Add SQL examples for operators validating migrated DBs.
- [x] Add troubleshooting note for incomplete runtime backfill on old records.
- [x] Update API reference docs with `current_runtime_key` terminology.
- [x] Add release note entry for semantic change and consumer impact.

## Rollout and Validation

- [x] Run `go test ./pkg/persistence/chatstore -count=1` and capture results.
- [x] Run `go test ./pkg/webchat -count=1` and capture results.
- [x] Run integration tests for profile switch and runtime persistence.
- [ ] Validate with sample conversation DBs (`/tmp/timeline3.db` and `/tmp/turns.db` style checks).
- [x] Confirm no compatibility env flags are introduced for this semantic change.
- [x] Record validation evidence in changelog.

## Closeout

- [ ] Run `docmgr doctor --ticket GP-26-PER-TURN-RUNTIME-TRUTH`.
- [ ] Update ticket status to `complete` once all validations pass.
- [ ] Link GP-26 outcome from GP-24/GP-25 docs where runtime semantics are referenced.
