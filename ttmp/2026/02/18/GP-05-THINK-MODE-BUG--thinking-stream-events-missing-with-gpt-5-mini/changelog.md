# Changelog

## 2026-02-18

- Initial workspace created
## 2026-02-18

- Created ticket workspace `GP-05-THINK-MODE-BUG` with diary and analysis documents.
- Added reproduction script to verify engine routing:
  - `scripts/inspect_engine_selection.go`
- Added reproduction script to exercise Responses SSE handling with trace logs:
  - `scripts/repro_thinking_stream_events.go`
- Captured reproducibility artifacts:
  - `sources/inspect_engine_selection.out`
  - `sources/repro_thinking_stream_events.trace.log`
- Wrote detailed bug report identifying two causes:
  - default provider path remains `openai` unless `ai-api-type=openai-responses` is set
  - missing handler branches for `response.reasoning_text.delta` and `response.reasoning_text.done`

## 2026-02-18

Completed reproducibility investigation: added deterministic engine-selection and SSE trace harnesses, captured outputs, and documented dual root causes for missing thinking stream events.

### Related Files

- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/analysis/01-bug-report-missing-thinking-stream-events.md — Detailed findings and fix guidance
- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/scripts/inspect_engine_selection.go — Engine routing reproduction
- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/scripts/repro_thinking_stream_events.go — SSE event-shape reproduction

## 2026-02-18

- Re-ran investigation against real OpenAI API using key sourced from local Pinocchio config.
- Added live trace captures for:
  - `ai-api-type=openai`
  - `ai-api-type=openai-responses` with `openai-reasoning-summary=auto`
  - `ai-api-type=openai-responses` with `openai-reasoning-summary=""`
- Updated analysis conclusions:
  - primary operational causes are provider routing + summary configuration
  - `response.reasoning_text.*` parser gap remains latent compatibility risk.

## 2026-02-18

- Implemented fixes:
  - Auto-route reasoning models (`gpt-5*`, `o1*`, `o3*`, `o4*`) from `openai` to `openai-responses`.
  - Added Responses SSE handling for `response.reasoning_text.delta` and `response.reasoning_text.done`.
- Added regression tests:
  - `pkg/inference/engine/factory/factory_test.go`
  - `pkg/steps/ai/openai_responses/engine_test.go`
- Verified:
  - `go test ./pkg/inference/engine/factory -count=1`
  - `go test ./pkg/steps/ai/openai_responses -count=1`
  - real API post-fix trace now shows auto-routing + `partial-thinking` in `openai` mode for `gpt-5-mini`.

## 2026-02-18

Validated findings against real OpenAI API and uploaded updated v2 ticket/artifact bundles to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_gpt5mini_openai.trace.log — Live openai path trace
- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_gpt5mini_openai_responses.trace.log — Live responses summary trace
- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_gpt5mini_openai_responses_no_summary.trace.log — Live responses no-summary trace

## 2026-02-18

Implemented fixes (auto-route gpt-5/o* to Responses + reasoning_text SSE handling), added regression tests, and validated with full go test suite and post-fix live traces.

### Related Files

- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/pkg/inference/engine/factory/factory.go — Auto-routing fix
- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/pkg/inference/engine/factory/factory_test.go — Auto-routing regression test
- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/pkg/steps/ai/openai_responses/engine.go — reasoning_text SSE handling
- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/pkg/steps/ai/openai_responses/engine_test.go — reasoning_text streaming regression test
- /home/manuel/workspaces/2026-02-18/thinking-mode-broken/geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_after_fix_gpt5mini_openai.trace.log — Post-fix real API evidence


## 2026-02-25

Ticket closed

