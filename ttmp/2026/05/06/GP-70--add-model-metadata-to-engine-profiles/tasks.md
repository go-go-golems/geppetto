# Tasks

## Phase 1: Core Types and Merge (geppetto only)

- [x] Create `geppetto/pkg/steps/ai/settings/model_info.go` — define `InputModality` string type with constants (text, image, audio, video, pdf), `ModelCost` struct (Input, Output, CacheRead, CacheWrite float64), `ModelInfo` struct with all fields from the design table (ID, Name, Reasoning, Input, ContextWindow, QualityHighWatermark, MaxOutputTokens, Cost, Metadata). All scalar fields pointer-typed. Add `Clone()` methods for both structs. Add `Validate()` on ModelInfo (quality_high_watermark ≤ context_window, non-negative costs). Add `NewModelInfo()` constructor.
- [x] Add `EffectiveContextLimit()` and `HardContextLimit()` helper methods on `ModelInfo` — EffectiveContextLimit returns quality_high_watermark if set and less than context_window, else context_window. HardContextLimit returns context_window. Return 0 when ModelInfo is nil or field is nil.
- [x] Add `ModelInfo *ModelInfo` field to `InferenceSettings` in `geppetto/pkg/steps/ai/settings/settings-inference.go`. Update `Clone()` to deep-copy ModelInfo. Update `GetMetadata()` to flatten ModelInfo fields into the metadata map with `ai-model-` prefixed keys. Update `GetSummary()` to display model info summary lines. Update `NewInferenceSettings()` if needed.
- [x] Implement `MergeModelInfo(base, overlay *ModelInfo) *ModelInfo` following the `MergeInferenceConfig` pattern: overlay wins for set pointer fields, nil falls back to base. Input slice replaced entirely. Cost replaced wholesale. Metadata map merged recursively (key-by-key, scalars overwrite). Wire into `mergeInferenceSettings()` in `geppetto/pkg/engineprofiles/inference_settings_merge.go` — verify YAML round-trip merge handles ModelInfo automatically or add explicit handling.
- [x] Decide on `ai-model-info` glazed section flags in `geppetto/pkg/steps/ai/settings/flags/inference.yaml` — implemented as no new CLI flags for v1 because profile-loaded metadata should remain profile/catalog data, not per-invocation flags.
- [x] Write `geppetto/pkg/steps/ai/settings/model_info_test.go` — test Clone, Validate (quality_high_watermark > context_window should error, negative cost should error), EffectiveContextLimit, HardContextLimit, MergeModelInfo (base nil, overlay nil, both set, partial overlay, Metadata recursive merge), ComputeCost (all usage fields, zero usage, nil cost, nil info).
- [x] Add test cases to `geppetto/pkg/steps/ai/settings/settings-inference_test.go` for InferenceSettings round-trip with ModelInfo, InferenceSettings.Clone() preserving ModelInfo, and InferenceSettings merge with ModelInfo.

## Phase 2: Profile Integration (geppetto only)

- [x] Create a profile YAML fixture with model_info under `inference_settings` in a test (use the example fixture from the design doc with gpt-4o-mini). Load it via `DecodeEngineProfileYAMLSingleRegistry()`, verify `InferenceSettings.ModelInfo` is populated with correct values. This tests that the YAML codec handles ModelInfo via struct tags without code changes.
- [x] Write `geppetto/pkg/engineprofiles/stack_merge_model_info_test.go` — test full profile stack merge with model_info: base layer has one ModelInfo, overlay layer has a different ModelInfo, verify overlay wins for set fields, nil fields fall back to base. Test partial overlay (only override context_window, keep base cost). Test Metadata recursive merge through the stack.
- [x] Add model_info to at least one existing example profile YAML file (e.g., `geppetto/examples/js/geppetto/profiles/10-provider-openai.yaml`) so the feature has a real-world reference.
- [x] Verify `ResolvedEngineProfile.InferenceSettings.ModelInfo` is populated after `ResolveEngineProfile()` — this should work automatically since ModelInfo is part of InferenceSettings, but add an explicit test.

## Phase 3: Engine Factory Integration (geppetto only)

- [x] Update `geppetto/pkg/inference/engine/factory/factory.go` — when `settings.ModelInfo` is available and `Reasoning` is set, use that instead of `isReasoningModel()`. Keep `isReasoningModel()` as a fallback when ModelInfo is nil or Reasoning is nil. Consider making `isReasoningModel()` a private helper rather than removing it entirely.
- [x] Update reasoning sanitization call sites to derive the reasoning decision from `ModelInfo.Reasoning` when present, falling back to existing name heuristics.
- [x] Update `geppetto/pkg/js/modules/geppetto/api_engines.go` `inferenceSettingsFromEngineOptions()` — when building InferenceSettings from JS options, check if the resolved profile's ModelInfo provides reasoning info, and use it instead of `inferAPIType()` heuristics where applicable.

## Phase 4: JS Module Surface (geppetto only)

- [x] Update `geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go` — add `modelInfo` property to `newResolvedEngineProfileObject()`. Create a `newModelInfoObject()` helper that converts ModelInfo to a goja JS object with all typed fields (id, name, reasoning, input, contextWindow, qualityHighWatermark, maxOutputTokens, cost.input/output/cacheRead/cacheWrite, metadata). Handle nil ModelInfo gracefully.
- [x] Update `geppetto/pkg/js/modules/geppetto/api_engines.go` — expose `modelInfo` on engine objects created from resolved profiles, so JS scripts can access `engine.modelInfo` after creating an engine.
- [x] Write a JS module integration test: resolve a profile with model_info, access `resolved.modelInfo.reasoning`, `resolved.modelInfo.cost.input`, etc. from JavaScript, verify values round-trip correctly.

## Phase 5: Cost Computation (geppetto only)

- [x] Add `Cost *float64` field to `InferenceResult` in `geppetto/pkg/turns/inference_result.go` with json/yaml/mapstructure tags (`cost,omitempty`).
- [x] Implement cost computation logic: after inference completes and `InferenceResult` is built, if `ModelInfo` is available with non-nil `Cost`, compute total cost from `InferenceUsage × ModelCost` rates (per 1M tokens). Store the result in `InferenceResult.Cost`. Decide where this wiring lives — likely in `RunInferenceWithResult()` or a post-inference hook.
- [x] Add separate model-cost stamping helper (`ApplyModelInfoCost`) and call it from provider result persistence paths rather than adding a settings dependency to `inference/engine`.
- [x] Verify `turnkeys_gen.go` does not need changes because `Cost` is a field on the existing `InferenceResult` payload, not a new turn metadata key.
- [x] Write test: given a ModelInfo with known costs and an InferenceUsage with known token counts, verify computed cost matches expected value. Test nil ModelInfo (cost should remain nil). Test zero-cost model (free model with all-zero rates).

## Phase 6: Pinocchio Integration

- [x] Update `pinocchio/pkg/ui/profileswitch/` — in the profile picker, render model capabilities: reasoning badge (✨ or similar), input modality icons (📝🖼️🎵), context window gauge (bar showing used/effective/hard), cost indicator (💰 per 1M tokens). These should read from the resolved profile's `InferenceSettings.ModelInfo`.
- [x] Verify `pinocchio/pkg/cmds/profilebootstrap/engine_settings.go` passes `ModelInfo` through automatically via merged `InferenceSettings`; no code change required there.
- [x] Expose `ModelInfo` through the web-chat profile API so the frontend/runtime can use `EffectiveContextLimit()` and `HardContextLimit()`; actual prompt trimming remains dependent on the existing token-budgeting layer and is not implemented in this pass.
- [x] Update `pinocchio/cmd/web-chat/profiles/` — the web-chat profile types and resolver should expose ModelInfo through the profile API so the frontend can render model capabilities.

## Cross-cutting

- [x] Add `model_info` to the profile YAML files used in integration tests and CI smoke tests so the feature is exercised in automated pipelines.
- [x] Update documentation through the GP-70 design document and example profile YAML; no separate README profile-format document was found in the repo.
