# Changelog

## 2026-02-10

- Initial workspace created


## 2026-02-10

Step 2 (bug 1): added regression test for pre-reasoning assistant context loss, confirmed failure, and fixed helper index selection logic.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/helpers.go — Fix skip logic to only omit one assistant block
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/helpers_test.go — Regression coverage for assistant pre-context preservation


## 2026-02-10

Step 3 (bug 2): added streaming regression test, reproduced success-on-error behavior, and changed streaming tail to return streamErr before final event.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/engine.go — Return error instead of success when stream reports provider failure
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/engine_test.go — Regression coverage for SSE error propagation

