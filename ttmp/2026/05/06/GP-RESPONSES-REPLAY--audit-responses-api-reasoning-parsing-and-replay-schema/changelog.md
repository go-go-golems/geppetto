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


## 2026-05-06

Implemented incoming Responses parser hardening: openai_responses block metadata, per-reasoning encrypted-content state, terminal reasoning_text content merge, and streaming parser regressions.

### Related Files

- pkg/steps/ai/openai_responses/engine.go — Parser stores item metadata
- pkg/steps/ai/openai_responses/engine_test.go — Regression tests for metadata capture
- pkg/steps/ai/openai_responses/helpers.go — Added openai_responses block metadata keys and reasoning output content helpers
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/reference/01-implementation-diary.md — Implementation diary step 3


## 2026-05-06

Updated Geppetto docs for Responses reasoning_text replay, item_id safety, openai_responses block metadata, and redacted request previews.

### Related Files

- pkg/doc/topics/06-inference-engines.md — Document Responses reasoning replay and metadata semantics
- pkg/doc/topics/08-turns.md — Document Reasoning block payload keys and openai_responses metadata
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/reference/01-implementation-diary.md — Implementation diary step 4


## 2026-05-06

Re-uploaded the updated GP-RESPONSES-REPLAY audit guide to reMarkable at /ai/2026/05/06/GP-RESPONSES-REPLAY.

### Related Files

- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/design/01-responses-reasoning-parsing-replay-audit.md — Updated design guide uploaded to reMarkable
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/reference/01-implementation-diary.md — Implementation diary step 5


## 2026-05-06

Live OpenAI Responses test rejected reasoning_text input content (input[].content maximum length 0), so replay now omits plaintext reasoning text while still parsing/storing it locally; targeted openai_responses tests pass.

### Related Files

- pkg/steps/ai/openai_responses/helpers.go — Reasoning replay omits plaintext reasoning_text input content after live provider rejection
- pkg/steps/ai/openai_responses/helpers_test.go — Updated replay regression to assert plaintext-only reasoning is omitted
- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/reference/01-implementation-diary.md — Implementation diary step 6 records live failure and policy change


## 2026-05-06

Ran live web-chat smoke tests after restart: the old user session still fails because it predates payload.item_id capture and no migration is intended; a fresh two-turn Responses session succeeded with encrypted reasoning/item_id replay.

### Related Files

- ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/reference/01-implementation-diary.md — Implementation diary step 7 records live restart and smoke-test results

