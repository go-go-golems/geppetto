## Forward Plan: OpenAI Responses API engine (next steps for tomorrow)

This document gives a concrete plan to continue the OpenAI Responses API integration work. It summarizes the current state, clarifies open items, and provides detailed, step-by-step tasks a new developer can pick up immediately.

### Current state (what works now)
- New engine `openai_responses` created and wired in the factory.
  - Path: `pkg/steps/ai/openai_responses/`
  - Constructor: `openai_responses.NewEngine(settings, options...)`
- Streaming implemented for Responses SSE:
  - Reasoning summary boundaries (`thinking-started/ended`) and deltas (via `EventTypePartialThinking`).
  - Output text deltas (`EventTypePartialCompletion`).
  - Function call arguments streaming and finalization (`response.function_call_arguments.*`, `output_item.done`).
- Tool calls:
  - Tool definitions included in request (`tools` array).
  - Tool call events emitted; `tool_call` blocks appended to `turn` to drive middleware.
  - Example `openai-tools` shows end-to-end tool execution with Responses.
- Observability:
  - Trace-level YAML dump of full Responses request payload.
  - Concise input preview and counts of tool blocks.
  - Clear error surfacing for SSE `error` and `response.failed`.

### What’s still missing (high priority)
1) Include tool_call and tool_result in Responses input for the next turn.
   - We currently re-send only the original user text. We need to include both the assistant’s function_call and the subsequent tool_result in `input` on the next request.

2) Tests and stabilization:
   - Unit tests for building Requests with/without tools; omission of sampling params for `o3/o4`.
   - SSE parsing golden tests (reasoning summary, text, function_call args).
   - Back-compat check: Chat engine unaffected.

3) Docs polish and examples:
   - Add a concise “Responses input schema” section to dev docs once tool blocks are included.
   - Provide one minimal reasoning-only example and one tools round-trip example.

### Detailed tasks (step-by-step)
1) Add tool_call and tool_result input items to `buildInputItemsFromTurn`
   - File: `pkg/steps/ai/openai_responses/helpers.go`
   - Function: `buildInputItemsFromTurn(t *turns.Turn) []responsesInput`
   - Extend mapping:
     - For `turns.BlockKindToolCall` → add `role: "assistant"` and content part `{ type: "function_call", name, arguments, call_id }`.
     - For `turns.BlockKindToolUse` → add `role: "tool"` and content part `{ type: "tool_result", tool_call_id, content }` (stringified result).
   - Ensure order is preserved (blocks already in turn order). Pair by `id` as needed.
   - Note: Keep existing user/system text as `input_text` parts.

2) Add concise input previews for tool items
   - In `openai_responses/engine.go` where we print the `input_preview`, for new part types, print `type`, `id/call_id` and first bytes of `arguments`/`result`.

3) Add tests
   - Create a small test suite under `pkg/steps/ai/openai_responses/`:
     - `helpers_test.go`: table tests for `buildInputItemsFromTurn` with turns containing user → tool_call → tool_use.
     - `sse_parser_test.go`: golden tests on minimal SSE event sequences (reasoning summary, function_call args, completed) to ensure events are emitted in order.
   - Keep tests deterministic; mock SSE by feeding lines into the flush logic or by factoring parser into a helper.

4) CLI/example improvements
   - `cmd/examples/openai-tools/main.go`:
     - Add a `--dump-request-yaml` flag that forces YAML printing even if not at trace (useful in CI/teaching).
     - Provide `--mode=thinking` and `--mode=tools` (already present) with prompts that exercise function_call.

5) Documentation updates
   - Update `pkg/doc/topics/06-inference-engines.md` to include `openai_responses` engine: when it is used, capabilities, and flags.
   - Add a short “Responses SSE events” section listing the events we handle.
   - Expand troubleshooting with common 400s (sampling params, tool_choice type).

### Testing matrix and commands
- Thinking only (Responses):
```
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type=openai-responses \
  --ai-engine=o4-mini \
  --mode=thinking \
  --prompt='Prove n odd sum equals n^2, stream reasoning summary' \
  --log-level info
```

- Tools round-trip (Responses):
```
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type=openai-responses \
  --ai-engine=o4-mini \
  --mode=tools \
  --prompt='Please use get_weather to check the weather in San Francisco, in celsius.' \
  --log-level trace --verbose
```

- Chat path sanity (unchanged):
```
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type=openai \
  --ai-engine=gpt-4o-mini-2024-07-18 \
  --mode=tools \
  --prompt='Use get_weather...' \
  --log-level info
```

### Known pitfalls and gotchas
- Sampling params (`temperature`, `top_p`) are rejected by o3/o4; ensure they are omitted for these models.
- `tool_choice` values like `auto` are not valid for function tools in Responses; omit it entirely.
- Errors often arrive as SSE `error` or `response.failed`; we already surface them. Keep an eye on `code`/`param` fields.
- The model may emit multiple function calls in sequence; ensure the turn preserves each call and result pair.

### Code pointers
- Engine: `pkg/steps/ai/openai_responses/engine.go`
- Request building: `pkg/steps/ai/openai_responses/helpers.go`
- Factory wiring: `pkg/inference/engine/factory/factory.go`
- Events and printers: `pkg/events/*.go`
- Example runner: `cmd/examples/openai-tools/main.go`

### Ready-to-pick tasks for tomorrow
- Implement tool_call/tool_result input items in `buildInputItemsFromTurn` and verify via trace YAML.
- Add the `--dump-request-yaml` flag in the example.
- Write minimal tests for helper and an SSE parsing golden.

If you get stuck on SSE behaviors or model quirks, enable `--log-level trace` and scroll to the “Responses: request YAML” block and SSE lines. When in doubt, try a short prompt to reproduce quickly.


