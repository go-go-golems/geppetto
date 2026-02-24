# Tasks

## Schema and Migration

- [ ] Add `runtime_key` column to `turns` table with non-null default in sqlite migration.
- [ ] Add `inference_id` column to `turns` table with non-null default in sqlite migration.
- [ ] Extend `ensureTurnsTableColumns` with additive migration guards for both new columns.
- [ ] Add migration SQL to backfill `inference_id` from `turn_metadata_json`.
- [ ] Add migration SQL to best-effort backfill `runtime_key` from turn metadata runtime key.
- [ ] Decide fallback behavior when runtime metadata is missing during backfill.
- [ ] Add index `turns_by_conv_runtime_updated`.
- [ ] Add index `turns_by_conv_inference_updated`.
- [ ] Add unit test for migrating an old DB missing both columns.
- [ ] Add unit test for backfill behavior with metadata present.
- [ ] Add unit test for backfill behavior with metadata absent.

## TurnStore API and Model

- [ ] Extend `chatstore.TurnSnapshot` with `RuntimeKey` field.
- [ ] Extend `chatstore.TurnSnapshot` with `InferenceID` field.
- [ ] Extend `TurnStore.Save` contract to accept save options for runtime/inference projection.
- [ ] Update all `TurnStore` implementations to match the new save signature.
- [ ] Keep backward payload compatibility by preserving metadata JSON projection.
- [ ] Add tests that `List` returns runtime and inference columns correctly.

## Persistence Wiring

- [ ] Update `turnStorePersister` to pass conversation runtime key explicitly on save.
- [ ] Add inference-id extraction fallback from `turns.KeyTurnMetaInferenceID`.
- [ ] Ensure turn save path sets runtime key for both final-persister and snapshot hook paths.
- [ ] Ensure empty runtime handling follows policy (error or empty sentinel).
- [ ] Add tests for persister behavior when runtime changes mid-conversation.

## Conversation Semantics Cleanup

- [ ] Define canonical naming: `current_runtime_key` for conversation-level runtime pointer.
- [ ] Update conversation response DTOs to expose `current_runtime_key`.
- [ ] Preserve old `runtime_key` response alias only if required by callers, otherwise remove.
- [ ] Document that conversation runtime is latest pointer, not history.
- [ ] Ensure conversation index persistence updates current runtime on profile switch.

## Debug API and Query Surface

- [ ] Add per-item `runtime_key` and `inference_id` to `/api/debug/turns` response payload.
- [ ] Add per-phase `runtime_key` and `inference_id` to `/api/debug/turn/:conv/:session/:turn` payload.
- [ ] Update debug API tests for new response fields.
- [ ] Update any frontend debug parsers/types that assume old shape.
- [ ] Add API documentation snippets for querying runtime history by turn.

## Runtime Switch Correctness

- [ ] Add integration test: start with runtime `inventory`, send turn, verify persisted turn runtime.
- [ ] Add integration test: switch profile/runtime to `planner` in same conversation.
- [ ] Add integration test: send second turn and verify persisted runtime switched to `planner`.
- [ ] Verify conversation current runtime is `planner` after switch.
- [ ] Verify first turn remains `inventory` and is not rewritten.
- [ ] Add regression test to prevent turn runtime overwrite by subsequent turns.

## Migration Playbook and Docs

- [ ] Update migration playbook with semantic model (`turn runtime` vs `conversation current runtime`).
- [ ] Add SQL examples for operators validating migrated DBs.
- [ ] Add troubleshooting note for incomplete runtime backfill on old records.
- [ ] Update API reference docs with `current_runtime_key` terminology.
- [ ] Add release note entry for semantic change and consumer impact.

## Rollout and Validation

- [ ] Run `go test ./pkg/persistence/chatstore -count=1` and capture results.
- [ ] Run `go test ./pkg/webchat -count=1` and capture results.
- [ ] Run integration tests for profile switch and runtime persistence.
- [ ] Validate with sample conversation DBs (`/tmp/timeline3.db` and `/tmp/turns.db` style checks).
- [ ] Confirm no compatibility env flags are introduced for this semantic change.
- [ ] Record validation evidence in changelog.

## Closeout

- [ ] Run `docmgr doctor --ticket GP-26-PER-TURN-RUNTIME-TRUTH`.
- [ ] Update ticket status to `complete` once all validations pass.
- [ ] Link GP-26 outcome from GP-24/GP-25 docs where runtime semantics are referenced.
