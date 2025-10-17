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

### Post-extraction follow-ups (completed/updated)
- [x] Verification: Both Chat Completions and Responses engines work correctly after refactoring
- [x] Documentation: Updated inference engines, events, and tools docs with Responses API details
- [~] Include tool_call and tool_result blocks in Responses `input` on subsequent turns
  - **Status**: Implemented but API rejected `function_call` and `tool_result` content types
  - **Issue**: OpenAI Responses API doesn't support these types in `input.content[]`
  - **Supported types**: `input_text`, `input_image`, `output_text`, `refusal`, `input_file`, `computer_screenshot`, `summary_text`
  - **Next**: Need to investigate conversation state management (see `04-conversation-state-management-design-proposals.md`)
  - **Likely solution**: Use conversation_id or previous_response_id instead of sending tool history
- [x] Design proposals for conversation state management created in `04-conversation-state-management-design-proposals.md`
- [x] Middleware coherence analysis created in `05-middleware-and-chained-responses-problem.md`
  - **Key insight**: Need to track which blocks came from which response_id
  - **Recommendation**: Solution 0 (Track Response Boundaries) using existing helpers
  - Leverages `SnapshotBlockIDs` and `NewBlocksNotIn` from middleware package
- [ ] Add unit tests and a minimal golden SSE test for stability
- [ ] Implement conversation state management (Hybrid approach from Proposal 6 in doc 04)
- [ ] Implement response boundary tracking (Solution 0 from doc 05)

### Code pointers
- Chat engine: `pkg/steps/ai/openai/engine_openai.go`
- Responses engine: `pkg/steps/ai/openai_responses/engine.go`
- Responses helpers: `pkg/steps/ai/openai_responses/helpers.go`
- Factory wiring: `pkg/inference/engine/factory/factory.go`
- Example runner: `cmd/examples/openai-tools/main.go`

If anything fails, re-run with `--log-level trace` and review the “Responses: request YAML” and SSE lines for quick diagnosis.


