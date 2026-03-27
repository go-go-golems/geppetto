# Tasks

- [x] Create ticket `GP-56-OPEN-RESPONSES` and seed the workspace
- [x] Inspect the current Geppetto Responses, event, turn, and tool-loop architecture
- [x] Review the Open Responses public contract and identify protocol differences from current OpenAI Responses support
- [x] Write an intern-focused design / analysis / implementation guide
- [x] Record the investigation diary

## Execution Plan

This ticket is being executed in small, reviewable slices. Each slice should end with:

- updated diary notes,
- focused tests,
- a git commit with a narrow scope.

## Phase 1: Provider Plumbing and Compatibility

Goal: introduce a first-class `open-responses` provider name without breaking existing `openai-responses` profiles, flags, or JS module callers.

- [x] Add `ApiTypeOpenResponses` to the shared provider type list in `pkg/steps/ai/types/types.go`
- [x] Treat `open-responses` and `openai-responses` as equivalent in the engine factory in `pkg/inference/engine/factory/factory.go`
- [x] Treat `open-responses` and `openai-responses` as equivalent in the token counter factory in `pkg/inference/tokencount/factory/factory.go`
- [x] Accept `open-responses` in CLI chat settings in `pkg/steps/ai/settings/flags/chat.yaml`
- [x] Update the JS engine option resolver in `pkg/js/modules/geppetto/api_engines.go` so:
  - reasoning-capable models infer `open-responses`,
  - `apiType: "open-responses"` works,
  - `apiType: "openai-responses"` remains a compatibility alias,
  - API key and base URL aliases are populated for both names
- [x] Add or update focused tests for provider selection and compatibility aliases in:
  - `pkg/inference/engine/factory/factory_test.go`
  - `pkg/inference/tokencount/factory/factory_test.go`
  - `pkg/steps/ai/openai_responses/provider_settings_test.go`
  - `pkg/js/modules/geppetto/module_test.go`
- [x] Commit Phase 1 as a provider-plumbing change set

## Phase 2: Responses Engine Decoupling

Goal: stop hardcoding OpenAI-only assumptions in the Responses runtime so other Open Responses providers can reuse the same engine contract.

- [x] Identify all OpenAI-specific assumptions in `pkg/steps/ai/openai_responses/engine.go`
- [x] Identify all OpenAI-specific assumptions in `pkg/steps/ai/openai_responses/helpers.go`
- [x] Extract provider lookup rules for:
  - base URL selection,
  - API key selection,
  - provider labeling in inference results,
  - default endpoint construction
- [ ] Introduce a provider-neutral Responses configuration layer while preserving current OpenAI behavior
- [ ] Ensure follow-up tool call replay still preserves reasoning/tool adjacency
- [x] Add focused engine and helper regression tests for the refactor
- [ ] Commit Phase 2 as a Responses-core refactor

## Phase 3: Reasoning Persistence Expansion

Goal: preserve richer reasoning state in turn blocks instead of storing only encrypted reasoning content.

- [x] Audit current reasoning block payload fields in:
  - `pkg/turns/keys_gen.go`
  - `pkg/turns/helpers_blocks.go`
  - `pkg/steps/ai/openai_responses/helpers.go`
- [x] Design the canonical reasoning payload shape for:
  - raw reasoning text,
  - reasoning summary text,
  - encrypted reasoning content,
  - item IDs and provider metadata
- [x] Extend block-writing helpers to persist the richer reasoning payload
- [x] Extend request-building helpers to replay the richer payload back into follow-up requests without losing OpenAI compatibility
- [x] Add regression tests for serialized turn blocks and replayed reasoning items
- [x] Commit Phase 3 as a reasoning-persistence change set

## Phase 4: Streaming Event Normalization

Goal: normalize Open Responses reasoning stream variants so Geppetto emits stable internal events even when upstream providers use different event names.

- [ ] Audit current SSE handling in `pkg/steps/ai/openai_responses/engine.go`
- [ ] Add normalization for at least:
  - `response.reasoning.delta`
  - `response.reasoning_text.delta`
  - `response.reasoning_text.done`
  - `response.reasoning_summary_text.delta`
- [ ] Confirm how normalized events map onto existing internal event types in `pkg/events/chat-events.go`
- [ ] Preserve compatibility with current `partial-thinking` / `reasoning-text-delta` event consumers
- [ ] Add fixture-driven regression tests covering alias event streams
- [ ] Commit Phase 4 as an event-normalization change set

## Phase 5: Fixtures, Examples, and Documentation Follow-Through

Goal: leave the system easy to test and easy to understand for the next engineer.

- [ ] Add example config or profile material demonstrating `open-responses`
- [ ] Add end-to-end trace fixtures for at least one non-OpenAI Open Responses provider
- [ ] Update ticket docs with implementation notes, gotchas, and follow-up risks discovered during coding
- [ ] Upload the refreshed ticket bundle to reMarkable after implementation stabilizes
- [ ] Run `docmgr doctor --ticket GP-56-OPEN-RESPONSES --stale-after 30`
- [ ] Commit Phase 5 as a fixtures-and-docs change set
