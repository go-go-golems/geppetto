# Tasks

## Phase 1: Core Types and Merge (geppetto only)

- [ ] Create `geppetto/pkg/steps/ai/settings/model_info.go` — define `InputModality` string type with constants (text, image, audio, video, pdf), `ModelCost` struct (Input, Output, CacheRead, CacheWrite float64), `ModelInfo` struct with all fields from the design table (ID, Name, Reasoning, Input, ContextWindow, QualityHighWatermark, MaxOutputTokens, Cost, Metadata). All scalar fields pointer-typed. Add `Clone()` methods for both structs. Add `Validate()` on ModelInfo (quality_high_watermark ≤ context_window, non-negative costs). Add `NewModelInfo()` constructor.
- [ ] Add `EffectiveContextLimit()` and `HardContextLimit()` helper methods on `ModelInfo` — EffectiveContextLimit returns quality_high_watermark if set and less than context_window, else context_window. HardContextLimit returns context_window. Return 0 when ModelInfo is nil or field is nil.
- [ ] Add `ModelInfo *ModelInfo` field to `InferenceSettings` in `geppetto/pkg/steps/ai/settings/settings-inference.go`. Update `Clone()` to deep-copy ModelInfo. Update `GetMetadata()` to flatten ModelInfo fields into the metadata map with `ai-model-` prefixed keys. Update `GetSummary()` to display model info summary lines. Update `NewInferenceSettings()` if needed.
- [ ] Implement `MergeModelInfo(base, overlay *ModelInfo) *ModelInfo` following the `MergeInferenceConfig` pattern: overlay wins for set pointer fields, nil falls back to base. Input slice replaced entirely. Cost replaced wholesale. Metadata map merged recursively (key-by-key, scalars overwrite). Wire into `mergeInferenceSettings()` in `geppetto/pkg/engineprofiles/inference_settings_merge.go` — verify YAML round-trip merge handles ModelInfo automatically or add explicit handling.
- [ ] Add `ai-model-info` glazed section flags in `geppetto/pkg/steps/ai/settings/flags/inference.yaml` if CLI flag exposure is desired (decide per field — cost and context window likely don't need CLI flags; reasoning might).
- [ ] Write `geppetto/pkg/steps/ai/settings/model_info_test.go` — test Clone, Validate (quality_high_watermark > context_window should error, negative cost should error), EffectiveContextLimit, HardContextLimit, MergeModelInfo (base nil, overlay nil, both set, partial overlay, Metadata recursive merge), ComputeCost (all usage fields, zero usage, nil cost, nil info).
- [ ] Add test cases to `geppetto/pkg/steps/ai/settings/settings-inference_test.go` for InferenceSettings round-trip with ModelInfo, InferenceSettings.Clone() preserving ModelInfo, and InferenceSettings merge with ModelInfo.

## Phase 2: Profile Integration (geppetto only)

- [ ] Create a profile YAML fixture with model_info under `inference_settings` in a test (use the example fixture from the design doc with gpt-4o-mini). Load it via `DecodeEngineProfileYAMLSingleRegistry()`, verify `InferenceSettings.ModelInfo` is populated with correct values. This tests that the YAML codec handles ModelInfo via struct tags without code changes.
- [ ] Write `geppetto/pkg/engineprofiles/stack_merge_model_info_test.go` — test full profile stack merge with model_info: base layer has one ModelInfo, overlay layer has a different ModelInfo, verify overlay wins for set fields, nil fields fall back to base. Test partial overlay (only override context_window, keep base cost). Test Metadata recursive merge through the stack.
- [ ] Add model_info to at least one existing example profile YAML file (e.g., `geppetto/examples/js/geppetto/profiles/10-provider-openai.yaml`) so the feature has a real-world reference.
- [ ] Verify `ResolvedEngineProfile.InferenceSettings.ModelInfo` is populated after `ResolveEngineProfile()` — this should work automatically since ModelInfo is part of InferenceSettings, but add an explicit test.

## Phase 3: Engine Factory Integration (geppetto only)

- [ ] Update `geppetto/pkg/inference/engine/factory/factory.go` — when `settings.ModelInfo` is available and `Reasoning` is set, use that instead of `isReasoningModel()`. Keep `isReasoningModel()` as a fallback when ModelInfo is nil or Reasoning is nil. Consider making `isReasoningModel()` a private helper rather than removing it entirely.
- [ ] Update `geppetto/pkg/inference/engine/inference_config_sanitize.go` — `SanitizeForReasoningModel` and `SanitizeOpenAIForReasoningModel` should accept `ModelInfo` (or a `reasoning bool`) as input for the sanitization decision, rather than requiring the caller to determine reasoning status externally. Update all callers.
- [ ] Update `geppetto/pkg/js/modules/geppetto/api_engines.go` `inferenceSettingsFromEngineOptions()` — when building InferenceSettings from JS options, check if the resolved profile's ModelInfo provides reasoning info, and use it instead of `inferAPIType()` heuristics where applicable.

## Phase 4: JS Module Surface (geppetto only)

- [ ] Update `geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go` — add `modelInfo` property to `newResolvedEngineProfileObject()`. Create a `newModelInfoObject()` helper that converts ModelInfo to a goja JS object with all typed fields (id, name, reasoning, input, contextWindow, qualityHighWatermark, maxOutputTokens, cost.input/output/cacheRead/cacheWrite, metadata). Handle nil ModelInfo gracefully.
- [ ] Update `geppetto/pkg/js/modules/geppetto/api_engines.go` — expose `modelInfo` on engine objects created from resolved profiles, so JS scripts can access `engine.modelInfo` after creating an engine.
- [ ] Write a JS module integration test: resolve a profile with model_info, access `resolved.modelInfo.reasoning`, `resolved.modelInfo.cost.input`, etc. from JavaScript, verify values round-trip correctly.

## Phase 5: Cost Computation (geppetto only)

- [ ] Add `Cost *float64` field to `InferenceResult` in `geppetto/pkg/turns/inference_result.go` with json/yaml/mapstructure tags (`cost,omitempty`).
- [ ] Implement cost computation logic: after inference completes and `InferenceResult` is built, if `ModelInfo` is available with non-nil `Cost`, compute total cost from `InferenceUsage × ModelCost` rates (per 1M tokens). Store the result in `InferenceResult.Cost`. Decide where this wiring lives — likely in `RunInferenceWithResult()` or a post-inference hook.
- [ ] Update `geppetto/pkg/inference/engine/inference_result_metadata.go` — `BuildInferenceResultFromEventMetadata` should accept optional `ModelInfo` and compute cost if available. Or add a separate `ComputeAndStampCost()` function.
- [ ] Update the `turnkeys_gen.go` if the new `Cost` field on `InferenceResult` needs a typed key for Turn.Metadata access.
- [ ] Write test: given a ModelInfo with known costs and an InferenceUsage with known token counts, verify computed cost matches expected value. Test nil ModelInfo (cost should remain nil). Test zero-cost model (free model with all-zero rates).

## Phase 6: Pinocchio Integration

- [ ] Update `pinocchio/pkg/ui/profileswitch/` — in the profile picker, render model capabilities: reasoning badge (✨ or similar), input modality icons (📝🖼️🎵), context window gauge (bar showing used/effective/hard), cost indicator (💰 per 1M tokens). These should read from the resolved profile's `InferenceSettings.ModelInfo`.
- [ ] Update `pinocchio/pkg/cmds/profilebootstrap/engine_settings.go` — pass `ModelInfo` through to engine creation path so the factory can use `Reasoning` instead of heuristics.
- [ ] Update `pinocchio/cmd/web-chat/` — use `EffectiveContextLimit()` for prompt trimming. Before sending a prompt, check if the estimated token count exceeds the effective context limit and warn/trim. Use `HardContextLimit()` as the absolute ceiling.
- [ ] Update `pinocchio/cmd/web-chat/profiles/` — the web-chat profile types and resolver should expose ModelInfo through the profile API so the frontend can render model capabilities.

## Cross-cutting

- [ ] Add `model_info` to the profile YAML files used in integration tests and CI smoke tests so the feature is exercised in automated pipelines.
- [ ] Update any documentation or README that describes the profile YAML format to include the new `model_info` section.
