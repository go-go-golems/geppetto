## TODOs for OpenAI Responses API integration

This document tracks the remaining implementation and hardening tasks to complete the Responses API integration in Geppetto.

### Routing, Engine, and Settings
- [ ] Route tool-call execution result blocks back into Responses input
  - Add `assistant` function_call and `tool` tool_result input items in `buildInputItemsFromTurn`
  - Preserve call ordering; pair `tool_call` and `tool_use` by `id`
- [ ] Add unit tests for `requiresResponses(model)` routing and `ApiTypeOpenAIResponses`

### Streaming and Events
- [ ] Emit tool-call events for Responses during streaming for each function call (complete across multiple parallel calls)
- [ ] Verify “reasoning-summary” delta events across long streams; guard against interleaved summaries
- [ ] Add accumulator tests: reasoning summary, output text, function_call args
- [ ] Document new `EventTypePartialThinking` and ensure printers support it (they do)

### Tooling
- [ ] Harden `PrepareToolsForResponses` against complex JSON Schema (nested refs, enums, defaults)
- [ ] Support parallel tool calls in Responses: concurrency hints, iterative loop
- [ ] Verify models and tool schema limits (name length, schema size) and log guardrails

### Error handling and Observability
- [ ] Add typed error mapping for common Responses codes (invalid_value, insufficient_quota, missing_required_parameter)
- [ ] Enrich `EventError` with provider code and param when present
- [ ] Add redact filters for logs (never print API keys)

### Model Compat and Parameters
- [ ] Enforce omission of `temperature` and `top_p` for o3/o4 in all code paths
- [ ] Add matrix tests across `o4-mini`, `o4`, `o3`, and a non-reasoning model
- [ ] Verify `reasoning.effort` and `reasoning.summary` semantics ("auto" vs explicit)

### CLI & Examples
- [ ] openai-tools: add a flag to dump YAML request on demand (in addition to trace)

### Documentation
- [ ] Update main docs to include Responses API and new flags (`ai-api-type`, OpenAI settings)
- [ ] Add developer guide: “How Geppetto handles Responses SSE”
- [ ] Include troubleshooting section for common 400 errors (tool_choice invalid, params not supported)

### Testing
- [ ] Golden tests for SSE parsing (reasoning, output_text, function_call args)
- [ ] Integration test for tool loop with Responses (mock HTTP server)
- [ ] Back-compat tests to ensure Chat path unaffected

### Future Enhancements
- [ ] Multi-modal content (images, files) mapping for Responses input
- [ ] Encrypted reasoning re-hydration (if/when usable in later turns)
- [ ] Configurable tool-choice for vendor-internal tools (file_search) without breaking function tools


