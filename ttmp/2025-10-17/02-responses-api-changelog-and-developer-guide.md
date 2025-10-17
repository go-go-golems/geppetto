## OpenAI Responses API: Changelog & Developer Guide

### Purpose and Scope
This doc captures the work to integrate OpenAI’s Responses API into Geppetto’s inference engine, with streaming, reasoning, and tool-calling support. It explains what changed, why, how to test it, and what to watch out for. It is written for a new developer to onboard and continue the work.

### What Changed (High-level)
- Added a new provider type `openai-responses` and a dedicated engine package `openai_responses` (factory selects it). Chat engine stays pure Chat Completions.
- Implemented SSE streaming for Responses: output text deltas, reasoning summary deltas, function call arguments, and completion metadata.
- Introduced thinking events: `thinking-started/ended`, per-delta `EventTypePartialThinking` for reasoning summary, and pretty printers.
- Implemented tool support for Responses: request `tools` schema, function call detection from SSE, ToolCall event emission, and Turn tool_call blocks.
- Improved observability: trace YAML dump of the exact request, concise input preview, detailed SSE logs, and surfaced provider errors.

### Key Files and Symbols
- `pkg/steps/ai/openai_responses/engine.go`: Responses engine (SSE, events, tools).
- `pkg/steps/ai/openai_responses/helpers.go`: Build Responses request, input items, tools mapping.
- `pkg/steps/ai/openai/engine_openai.go`: Router selecting Responses vs Chat.
- `pkg/inference/engine/factory/factory.go`: Provider validation and fallback key for `openai-responses`.
- `pkg/events/chat-events.go`: New `EventTypePartialThinking` + constructors.
- `cmd/examples/openai-tools/main.go`: Example runner, printers, tool middleware wiring.

### Detailed Changes
1) Provider and Routing
- Added `ApiTypeOpenAIResponses` and supported it in the engine factory to instantiate `openai_responses.NewEngine`. Validation allows fallback to `openai-api-key`; no separate base URL required.
- Chat engine no longer routes; provider selection is done in the factory.

2) Request Building
- `buildResponsesRequest` builds `model`, `input`, `max_output_tokens`, `stop_sequences`, and reasoning (`effort`, `summary`).
- Omits `temperature`/`top_p` for `o3/o4` models (they reject sampling params).
- `PrepareToolsForResponses` converts registry tools to Responses schema: `[{type:function, name, description, parameters}]`.
- Omit `tool_choice` for function tools (avoid invalid vendor values reserved for built-ins like `file_search`).

3) Streaming and Events
- Parses SSE: `response.output_item.added/done`, `response.output_text.delta`, `response.reasoning_summary_text.delta`, `response.function_call_arguments.*`, `response.completed`, and `error/response.failed`.
- Emits:
  - Info events for phase boundaries (`thinking-started/ended`, `output-started/ended`).
  - `EventTypePartialThinking` for reasoning summary deltas.
  - `EventTypePartialCompletion` for output text deltas.
  - `EventToolCall` when a function call completes (with `call_id`, `name`, `arguments`).
- Accumulates usage (input/output tokens, reasoning tokens) and carries on the Final event.

4) Tool Calls (Responses)
- Reads function call arguments from SSE deltas, finalizes on `output_item.done` with `type:function_call`.
- Publishes `EventToolCall` immediately, and appends a `tool_call` block to the `turn` so the middleware can execute.
- After middleware appends `tool_use` (tool_result) block to the same `turn`, the engine is re-invoked and continues, showing a full round-trip.

5) Observability and Debugging
- Logs a concise `input_preview` (role, part type, text length, head snippet) and tool block counts for each request.
- At trace level, dumps full YAML of the request payload right before sending.
- Surfaces streaming errors (`error`, `response.failed`) as `EventError`, including code and message.

### What Worked
- Streaming of reasoning summary and output deltas; partials print well without noisy labels.
- Tool calls over Responses: arguments stream reliably and finalize with `output_item.done`.
- Error surfacing clarifies quota/misconfiguration issues.

### What Didn’t (and Fixes)
- Sampling params on o4/o3 → invalid parameter errors; fixed by omitting `temperature`, `top_p`.
- `tool_choice: auto` rejected; removed tool_choice for function tools.
- Wrong tool schema (nested `function`) rejected; fixed to top-level name/description/parameters.
- Usage not populated early; now parsed on `response.completed` from nested fields.

### Open Questions / Caveats
- **Tool History Issue**: Attempted to include `function_call` and `tool_result` content types in `input`, but API rejected them. Supported content types are: `input_text`, `input_image`, `output_text`, `refusal`, `input_file`, `computer_screenshot`, `summary_text`. This suggests the Responses API uses a different state management model (conversation_id or previous_response_id) instead of explicit tool history. See `04-conversation-state-management-design-proposals.md` for design proposals.
- **Current Limitation**: Tool calls repeat in a loop because the model doesn't see previous tool results. This will be fixed once conversation state management is implemented.
- Parallel tool calls and `parallel_tool_calls` semantics need UX design and tests.
- Encrypted reasoning: we include the flag, but no hydration mechanism is implemented yet.

### How to Test
1) Thinking only
```
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type=openai-responses \
  --ai-engine=o4-mini \
  --mode=thinking \
  --prompt='Prove n odd sum equals n^2, stream reasoning summary.' \
  --log-level info
```
Expect: “Thinking started”, streaming reasoning summary text, token summary line.

2) Tools round-trip
```
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type=openai-responses \
  --ai-engine=o4-mini \
  --mode=tools \
  --prompt='Please use get_weather to check the weather in San Francisco, in celsius.' \
  --log-level trace --verbose
```
Expect: function_call arguments deltas → ToolCall event → middleware executes → tool_result block → next turn repeats until completion. With `--log-level trace`, YAML of requests is printed.

3) Error visibility
Use an invalid key or provoke a 400/429 to see `[error]` lines emitted by the printer and detailed logs.

### Developer Notes
- Files to read first: `engine_openai_responses.go`, `helpers_responses.go`, `engine_openai.go` (routing), `chat-events.go` (events), `openai-tools/main.go` (example), `factory.go` (provider wiring).
- When editing SSE parsing, keep logs at trace for high-frequency paths; avoid burdening stdout in normal info mode.
- Maintain Go guidelines from `config/llm/go-guidelines.md` (pure helpers, clear DTOs, early returns).

### Next Steps
- **Priority**: Implement conversation state management to fix tool loop issue (see `04-conversation-state-management-design-proposals.md`). Recommended approach is Proposal 6 (Hybrid: Turn.Data + Accessors).
- Add integration tests with a mock SSE server to stabilize function-call parsing across versions.
- Expand docs with conversation state management examples once implemented.
- Consider adding `ConversationManager` (Proposal 5) for high-level multi-turn workflows.

### Related Documents
- `01-things-todo-for-the-responses-api.md`: Detailed task list (partially completed)
- `03-forward-plan-responses-api.md`: Forward plan with verification steps and next tasks
- `04-conversation-state-management-design-proposals.md`: Six design proposals for managing conversation state (NEW)


### Stateless Encrypted Reasoning (NEW)

- Always request encrypted reasoning: Requests now force `include: ["reasoning.encrypted_content"]` so the provider returns an opaque reasoning blob on each response.
- New block kind: Added `BlockKindReasoning` in `pkg/turns/types.go` and `PayloadKeyEncryptedContent` in `pkg/turns/keys.go` to store `encrypted_content`.
- Parsing: The Responses engine appends a `reasoning` block when `response.output_item.done` has `type: reasoning`, storing `encrypted_content` if present.
- Input building: The helper converts the current `Turn` to Responses input by:
  - Emitting prior `user/system/assistant` text as `input_text` parts.
  - Emitting the latest `reasoning` block as an item: `{type: "reasoning", id, encrypted_content, summary: []}`.
  - Echoing tool history as items: `{type: "function_call", call_id, name, arguments}` and `{type: "function_call_output", call_id, output}`.
  - Only the most recent reasoning item is included to satisfy vendor ordering rules.
- Function call output: For function results, we now set `call_id` (not `tool_call_id`) to match Responses expectations.

### Example Runner Updates

- No CLI flag needed: Stateless continuation is the default (encrypted reasoning is always requested). The `openai-tools` example remains unchanged at the interface.
- Test invocations (same as before):
  - Thinking: `--ai-api-type=openai-responses --ai-engine=o4-mini --mode=thinking` OK.
  - Tools: `--ai-api-type=openai-responses --ai-engine=o4-mini --mode=tools` now streams a function_call, executes the tool via middleware, and sends back `function_call_output` together with the latest reasoning item.

### Verified Behavior (from real runs)

- Thinking mode: boundaries and token usage printed; reasoning tokens surfaced in final summary.
- Tools round-trip (first cycle):
  - Function call streamed (arguments deltas) → `output_item.done` fired → ToolCall event emitted.
  - Middleware executed mock `get_weather` → tool result block appended.
  - Next request included: user text, latest reasoning item (with blob), function_call, function_call_output.
- Subsequent continuation: vendor returned 400 when a `reasoning` item wasn’t followed by the required companion item. This indicates Responses expects the `reasoning` item to be accompanied by the other items from the same originating response as a contiguous group.

### Known Limitation (to address next)

- Grouping requirement: When echoing back items, the `reasoning` item must be followed by its associated items from the same response (e.g., the `function_call`). We currently include only the latest reasoning item and echo tool history, but we don’t yet ensure contiguous grouping per response boundary.
- Planned fix: Implement response-boundary grouping in input building so that we send:
  1) prior conversation context,
  2) the latest reasoning item,
  3) immediately followed by the items from the same response (e.g., `function_call`, `function_call_output`).

### File Diffs (high-level)

- `pkg/turns/types.go`: add `BlockKindReasoning` and string mapping.
- `pkg/turns/keys.go`: add `PayloadKeyEncryptedContent`.
- `pkg/steps/ai/openai_responses/helpers.go`:
  - Force `include: ["reasoning.encrypted_content"]`.
  - Extend request/response DTOs.
  - Build input with latest `reasoning` item, `function_call`, `function_call_output` (using `call_id`).
- `pkg/steps/ai/openai_responses/engine.go`:
  - Parse `reasoning` items from SSE and non-streaming.
  - Append `BlockKindReasoning` with `encrypted_content`.
  - Remove duplicate fallback reasoning append.
- `cmd/examples/openai-tools/main.go`: no flag needed; unchanged behavior, runs fine with new defaults.

### Testing Notes

- Thinking: OK.
- Tools (first round): OK.
- Tools (second round): 400 “reasoning item provided without its required following item” observed; will be addressed by grouping contiguous items from the same response in the builder.


