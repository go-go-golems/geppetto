## Forward Plan: Extract openai-responses into its own engine (status + next steps)

This plan focuses on completing and validating the extraction of the OpenAI Responses API into its own engine, separate from the Chat Completions engine.

### Objectives
- Keep `pkg/steps/ai/openai` strictly for Chat Completions.
- House all Responses API logic in `pkg/steps/ai/openai_responses`.
- Use the engine factory to select between engines based on `ai-api-type`.

### Extraction tasks (status)
- [x] Create `openai_responses` engine package and move Responses code
  - Code now under: `pkg/steps/ai/openai_responses/{engine.go,helpers.go}`
  - New type: `openai_responses.Engine` with `NewEngine(settings, options...)`
- [x] Update engine factory to use `openai_responses` for `ai-api-type=openai-responses`
  - File: `pkg/inference/engine/factory/factory.go`
  - Branch selects `openai_responses.NewEngine` for `openai-responses`
- [x] Strip Responses routing from OpenAI chat engine
  - File: `pkg/steps/ai/openai/engine_openai.go`
  - Removed in-engine routing; factory now decides
- [ ] Build and verify both Chat and Responses paths via examples
  - Run commands below; capture logs and ensure both paths work as expected

### Verification steps (to do now)
1) Chat engine still works (tools, streaming):
```
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type=openai \
  --ai-engine=gpt-4o-mini-2024-07-18 \
  --mode=tools \
  --prompt='Please use get_weather to check the weather in San Francisco, in celsius.' \
  --log-level info
```
Expect: tool_call → tool_result → final assistant answer (no Responses YAML in logs).

2) Responses engine works (thinking + tools):
```
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type=openai-responses \
  --ai-engine=o4-mini \
  --mode=tools \
  --prompt='Please use get_weather to check the weather in San Francisco, in celsius.' \
  --log-level trace --verbose
```
Expect: reasoning boundaries, function_call args streamed, ToolCall events, request YAML printed at trace, tool_result loop continues.

### Post-extraction follow-ups (next)
- Include tool_call and tool_result blocks in Responses `input` on subsequent turns
  - Add to `buildInputItemsFromTurn` in `openai_responses/helpers.go`:
    - `assistant` function_call item (name, call_id, arguments)
    - `tool` tool_result item (tool_call_id, content)
- Add unit tests and a minimal golden SSE test for stability
- Update docs: engine selection, troubleshooting, example commands

### Code pointers
- Chat engine: `pkg/steps/ai/openai/engine_openai.go`
- Responses engine: `pkg/steps/ai/openai_responses/engine.go`
- Responses helpers: `pkg/steps/ai/openai_responses/helpers.go`
- Factory wiring: `pkg/inference/engine/factory/factory.go`
- Example runner: `cmd/examples/openai-tools/main.go`

If anything fails, re-run with `--log-level trace` and review the “Responses: request YAML” and SSE lines for quick diagnosis.


