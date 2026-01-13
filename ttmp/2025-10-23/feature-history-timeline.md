# Feature Timeline: Responses API → Debug Taps

_Source data: `ttmp/2025-10-23/git-history-and-code-index.db` (built from `task/add-gpt5-responses-to-geppetto`). Commit order is chronological._

## 1. OpenAI Responses API foundations (2025-10-16 → 2025-10-20)

1. **db50680 – 2025-10-16 – “Add more logging / info for reasoning tokens”**  
   Expanded the OpenAI tools example and the early responses engine (`cmd/examples/openai-tools/main.go`, `pkg/steps/ai/openai/engine_openai_responses.go`) so the CLI surfaces raw reasoning tokens while streaming. The symbol index captures new helper functions such as `RunIntoWriter` and enhanced buffering inside `runResponses`.
2. **490eb2f – 2025-10-16 – “Add thinking boundaries”**  
   Tightened event emission boundaries for “thinking” segments by wiring additional hooks in `pkg/events/step-printer-func.go` and the responses engine. This lays groundwork for exposing intermediate reasoning to clients.
3. **7caa1e4 – 2025-10-17 – “Add streaming of thinking tokens…”**  
   Broadened streaming support across seven files, including `pkg/events/chat-events.go` and new helpers in `pkg/steps/ai/openai/helpers_responses.go`, ensuring errors and incremental events can flow through the responses pipeline. The example CLI was updated again to demo the richer stream.
4. **2e61de0 – 2025-10-17 – “Add some debugging info and fallbacks”**  
   Captured design notes under `ttmp/2025-10-17/` while hardening the inference factory (`pkg/inference/engine/factory/factory.go`) and responses helpers to gracefully handle missing metadata when the API falls back to legacy behaviour.
5. **b7363d1 – 2025-10-17 – “Move openai responses to its own package and engine”**  
   The responses implementation was extracted into a dedicated package (`pkg/steps/ai/openai_responses/{engine.go,helpers.go}`) and removed from the generic OpenAI engine. Factory wiring was updated so the new engine can be constructed in isolation. Planning docs in `ttmp/2025-10-17/03-...` describe the migration strategy.
6. **dd5ceb2 – 2025-10-17 – “Update plan and docs”**  
   Synced high-level documentation (`pkg/doc/topics/04-events.md`, `06-inference-engines.md`, `07-tools.md`) with the new responses engine structure and event flow.
7. **e2b071d – 2025-10-17 – “Add more design work…”**  
   Produced multiple design documents covering conversation state, middleware chaining, and encrypted reasoning. These live under `ttmp/2025-10-17/04-` to `07-` and serve as specs for the next implementation step.
8. **8c48301 – 2025-10-17 – “Add handling encrypted thinking blocks for stateless continuation”**  
   Implemented encrypted block handling in `pkg/steps/ai/openai_responses/{engine.go,helpers.go}` and added key utilities in `pkg/turns/{keys.go,types.go}` so stateless resumes can decrypt prior context.
9. **20e7ea5 – 2025-10-17 – “Implement stateless responses with encrypted block content”**  
   Rounded out the stateless story by finalising the helpers/engine logic and documenting the flow in `ttmp/2025-10-17/02-responses-api-changelog-and-developer-guide.md`.
10. **f484f79 – 2025-10-17 – “I think this is parallel tool calling?”**  
    Tweaked `pkg/steps/ai/openai_responses/helpers.go` to prepare for parallel tool execution—primarily wiring flags within request envelopes.
11. **8beb585 – 2025-10-17 – “Add flag for parallel tools and also server tools”**  
    Extended the example runner and engine to toggle parallel tool execution via new request options.
12. **239d207 – 2025-10-17 – “Make web_search server side tool work in the responses API”**  
    Added `ttmp/.../08-unified-events-for-advanced-model-functionality.md` and patched the engine/example so server-side search tools integrate with the responses stream.
13. **fdc5991 – 2025-10-20 – “Add more documents about unified anthropic/openai events”**  
    Documented event mappings for OpenAI and Anthropic under `ttmp/2025-10-17/09-11-*.md`, solidifying the shared event vocabulary.
14. **24a3814 – 2025-10-20 – “Add web search events and more mapped events”**  
    Propagated the unified event schema into runtime (`pkg/events/chat-events.go`, `event-router.go`, `step-printer-func.go`) and synced docs. Removed the obsolete `TODO.md` marker.
15. **bae23df – 2025-10-20 – “Fix input_text/output_text in input blocks”**  
    Corrected block conversion edge cases within the responses helpers/engine and recorded migration analysis under `ttmp/2025-10-20/01-02-*.md`.
16. **d051bdf – 2025-10-21 – “Add tests for converting blocks into openai responses”**  
    Landed unit and e2e tests (`helpers_test.go`, `helpers_e2e_test.go`) verifying block-to-response translation, ensuring the newly added encryption pathways remain safe.

## 2. Supporting serialization & runner scaffolding (2025-10-21)

17. **c1dead0 – 2025-10-21 – “Add turns/blocks serde”**  
    Added `pkg/turns/serde/serde.go` and supporting docs to serialise turns/blocks, enabling fixture capture/replay.
18. **92823ff – 2025-10-21 – “Add e2e runner with recording capabilities”**  
    Introduced `cmd/e2e-responses-runner` with YAML fixtures and wire-up in `go.mod` to record responses sessions.
19. **e5cdac0 – 2025-10-21 – “Add fixture code to pkg”**  
    Moved fixture loading logic into `pkg/inference/fixtures/fixtures.go`, with accompanying design notes (`ttmp/2025-10-20/05-...`).
20. **057410f – 2025-10-21 – “Pimp up the fixture runner”**  
    Enhanced the e2e runner & fixtures plus the responses engine to support richer playback scenarios.

## 3. Debug taps & LLM runner (2025-10-21 → 2025-10-23)

21. **696aaa6 – 2025-10-21 – “Add debug taps...”**  
    Replaced the e2e runner with the new `cmd/llm-runner` (fixtures, README, main entrypoint) and introduced `pkg/inference/engine/debugtap.go` plus `pkg/inference/fixtures/rawtap.go` to mirror provider traffic. Responses helpers were adjusted to emit tap events.
22. **5908d75 – 2025-10-21 – “Improve UI for debugging requests”**  
    Added the HTTP API (`cmd/llm-runner/api.go`), `serve.go`, templated UI, and an accompanying TypeScript SPA under `cmd/llm-runner/web/…` (43 files total). Existing fixtures/README were updated to explain how to inspect captured traffic.
23. **a4d3e5b – 2025-10-23 – “Update go.mod”**  
    Pulled in UI/runtime dependencies required by the runner updates and adjusted docs referencing the new modules.
24. **facd1b8 – 2025-10-23 – “Better correlation of turn to backend request”**  
    Added cassette fixtures (`cmd/llm-runner/cassettes/simple.yaml`) and tightened correlation logic across the runner API, server, and front-end components so events line up with backend requests.
25. **e648548 – 2025-10-23 – “Remove e2e-responses-runner before deleting it”**  
    Temporarily restored the legacy runner files to stage their removal cleanly for downstream consumers.
26. **2b0e150 – 2025-10-23 – “Remove e2e-responses-runner”**  
    Deleted the obsolete runner once the llm-runner fully replaced it.
27. **eaa263f – 2025-10-23 – “Format the source code”**  
    gofmt’d all touched files (runner, events, examples) after the large refactor.
28. **fd0f919 – 2025-10-23 – “Fix linting”**  
    Addressed golangci-lint findings across examples, middleware, responses helpers, and the new llm-runner.

## 4. Event extensibility (2025-10-22)

29. **e130785 – 2025-10-22 – “Add event extensibility”**  
    Created `pkg/events/registry.go`, augmented `pkg/events/context.go` and `pkg/inference/engine/debugtap.go` to allow external registry injection, and captured extensive design notes under `ttmp/2025-10-21/geppetto-2025-10-21-geppetto-events-extensibility-design/`.

## 5. Generic tool executor (2025-10-22)

30. **b21e6f9 – 2025-10-22 – “Add generic tool executor”**  
    Added `pkg/inference/tools/base_executor.go`, refactored `executor.go`, and produced extensibility notes under `ttmp/2025-10-22/0{1,2}-*.md`, enabling downstream consumers to plug in bespoke tool backends while reusing the shared execution façade.

---

These commits, captured in the SQLite index, chart the progression from the initial OpenAI responses implementation, through the build-out of debugging taps and the `llm-runner`, to the broader platform abstractions (event registries and generic tool execution) that followed.
