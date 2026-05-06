# Changelog

## 2026-05-06

- Initial workspace created


## 2026-05-06

Created GP-RESPONSES-REPLAY audit ticket, moved archived OpenAI Responses documentation into sources/, wrote intern-oriented design/implementation guide for reasoning parsing and replay, and corrected a misleading replay comment in helpers.go.

### Related Files

- pkg/steps/ai/openai_responses/helpers.go — Correct comment to acknowledge official reasoning_text schema while keeping current omission policy
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/design/01-responses-reasoning-parsing-replay-audit.md — Detailed audit
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/sources/openai-reasoning-guide.md — Archived official reasoning guide snapshot
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/sources/openai-reasoning-items-cookbook.md — Archived official reasoning-items cookbook snapshot
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/sources/openai-responses-create-api-reference.md — Archived Responses create API reference snapshot
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/sources/openai-responses-object-api-reference.md — Archived Responses object API reference snapshot


## 2026-05-06

Uploaded the Responses reasoning parsing/replay audit guide to reMarkable at /ai/2026/05/06/GP-RESPONSES-REPLAY.


## 2026-05-06

Updated design decisions: keep `item_id` because it matches the Responses spec, do not add migration/backwards-compatibility work, store OpenAI-specific response metadata under `metadata[openai_responses.*]`, leave middleware reordering out of scope, and implement reasoning_text replay directly rather than behind a capability flag.


## 2026-05-06

Implemented request-side reasoning_text replay and redacted Responses input previews; added regression tests and validated with go test ./pkg/steps/ai/openai_responses -count=1.

### Related Files

- pkg/steps/ai/openai_responses/engine.go — Request input summary logging now uses typed redacted preview
- pkg/steps/ai/openai_responses/helpers.go — Reasoning items now replay payload.text as content reasoning_text and expose redacted preview helpers
- pkg/steps/ai/openai_responses/helpers_test.go — Regression tests for reasoning_text replay
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/reference/01-implementation-diary.md — Implementation diary step 2

