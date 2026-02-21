# Changelog

## 2026-02-20

- Initial workspace created


## 2026-02-20

Captured runtime snapshots and correlated OpenAI 400 with turn/block evidence for missing reasoning predecessor.

### Related Files

- /home/manuel/workspaces/2026-02-14/hypercard-add-webchat/geppetto/ttmp/2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/sources/snapshots/gpt-5.log — Snapshot of failing run logs
- /home/manuel/workspaces/2026-02-14/hypercard-add-webchat/geppetto/ttmp/2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/sources/snapshots/turns.db — Snapshot used to reconstruct failing turn


## 2026-02-20

Patched OpenAI Responses input builder to preserve reasoning/function_call adjacency across full turn history and added regression tests.

### Related Files

- /home/manuel/workspaces/2026-02-14/hypercard-add-webchat/geppetto/pkg/steps/ai/openai_responses/helpers.go — Bug fix for ordered reasoning preservation
- /home/manuel/workspaces/2026-02-14/hypercard-add-webchat/geppetto/pkg/steps/ai/openai_responses/helpers_test.go — New test for older reasoning/tool_call chains

