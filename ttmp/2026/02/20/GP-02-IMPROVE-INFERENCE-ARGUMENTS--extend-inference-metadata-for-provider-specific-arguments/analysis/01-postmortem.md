---
Title: Postmortem
Ticket: GP-02-IMPROVE-INFERENCE-ARGUMENTS
Status: active
Topics:
    - inference
    - metadata
    - architecture
    - geppetto
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/inference/engine/inference_config.go
      Note: Core types and resolution helpers
    - Path: pkg/inference/engine/turnkeys.go
      Note: Typed Turn.Data key declarations
    - Path: pkg/steps/ai/claude/helpers.go
      Note: Claude engine wiring
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: OpenAI Responses engine wiring
    - Path: pkg/steps/ai/openai/helpers.go
      Note: OpenAI Chat engine wiring
    - Path: pkg/steps/ai/gemini/engine_gemini.go
      Note: Gemini engine wiring
    - Path: pkg/steps/ai/settings/settings-step.go
      Note: StepSettings.Inference field
ExternalSources: []
Summary: "Postmortem for GP-02: what was built, what went well, what to watch"
LastUpdated: 2026-02-20T08:13:52.951025019-05:00
WhatFor: "Quick summary of outcomes and lessons from the InferenceConfig implementation"
WhenToUse: "When reviewing the feature for merge or planning follow-up work"
---

# Postmortem: Layered InferenceConfig for Per-Turn Provider Arguments

## What We Built

A layered configuration system that gives callers per-turn control over provider-specific LLM parameters (thinking budget, reasoning effort, temperature, etc.) while supporting engine-level defaults.

**Before:** All inference parameters were fixed at engine construction time via `StepSettings`. No way to adjust thinking budget, reasoning effort, or other provider-specific knobs per inference call.

**After:** Three-layer resolution — `Turn.Data > StepSettings.Inference > StepSettings.Chat > defaults` — using the existing typed `DataKey[T]` pattern. Callers can set engine-wide defaults *and* override per turn.

## Scope

| Area | Detail |
|------|--------|
| New file | `inference/engine/inference_config.go` — 3 config types + 3 resolution helpers |
| Modified files | 9 (keys, settings, all 4 provider engines) |
| Lines changed | +363 / -17 |
| Commits | `36d93f1` (code), `b5ae820` (docs) |

### Cross-Provider Fields

| Field | Claude | OpenAI Responses | OpenAI Chat | Gemini |
|-------|--------|-----------------|-------------|--------|
| ThinkingBudget | `thinking.budget_tokens` | `reasoning.max_tokens` | -- | -- |
| ReasoningEffort | -- | `reasoning.effort` | -- | -- |
| Temperature | `temperature` | `temperature` | `temperature` | `temperature` |
| TopP | `top_p` | `top_p` | `top_p` | `top_p` |
| MaxResponseTokens | `max_tokens` | `max_output_tokens` | `max_tokens` | `max_output_tokens` |
| Stop | `stop_sequences` | `stop_sequences` | `stop` | -- |
| Seed | -- | -- | `seed` | -- |

### Bonus: Completed Existing Gap

`KeyStructuredOutputConfig` was declared in `turnkeys.go` but never read by any engine. All four engines now consume it from Turn.Data, giving callers per-turn structured output control.

## What Went Well

1. **The existing typed key pattern scaled cleanly.** Adding `KeyInferenceConfig`, `KeyClaudeInferenceConfig`, and `KeyOpenAIInferenceConfig` was mechanical — the `DataK[T]` infrastructure handled serialization, validation, and type safety.

2. **Engine-level defaults were a good design refinement.** The user's feedback to add `StepSettings.Inference` (not just Turn.Data) made the system much more ergonomic: configure once at engine creation, override only when needed.

3. **No import cycles.** `settings` already imported `engine`, so adding `Inference *engine.InferenceConfig` to `StepSettings` was cycle-safe. This was verified before implementation.

4. **Backward compatibility preserved.** All existing tests passed without changes. When no InferenceConfig keys are set, behavior is identical to before.

## Issues Encountered

### 1. Import Name Collision (Build Error)

Both `claude/helpers.go` and `openai/helpers.go` had a local variable `engine` (the model name string) that shadowed the `engine` package import. The compiler error was misleading: `engine.ResolveInferenceConfig undefined (type string has no field or method ResolveInferenceConfig)`.

**Fix:** Aliased the import to `infengine`.

**Lesson:** When adding package-level function calls to existing code, check for local variable name collisions with the package name.

### 2. Embedded Field Lint (Pre-Commit Rejection)

The Gemini `genai.GenerativeModel` embeds `GenerationConfig`, so `model.GenerationConfig.Temperature` triggers staticcheck QF1008 ("could remove embedded field from selector"). First commit attempt was rejected by the pre-commit hook.

**Fix:** Simplified to `model.Temperature`, `model.TopP`, `model.MaxOutputTokens`.

**Lesson:** When accessing fields on types you didn't write, check whether the parent is an embedded struct.

## What to Watch

1. **Resolution is full-replacement, not per-field merge.** If a caller sets `KeyInferenceConfig` with only `ThinkingBudget`, any engine-level `Temperature` from `StepSettings.Inference` is lost. This is intentional (simpler) but may surprise callers who expect partial overrides.

2. **Claude thinking + temperature constraint.** Claude requires `temperature = 1.0` (or unset) when thinking is enabled. The current code does not enforce this — the caller is responsible.

3. **No unit tests for the new wiring.** The implementation compiles and existing tests pass, but there are no dedicated tests for InferenceConfig resolution or per-engine Turn.Data override behavior. These should be added.

## Follow-Up Work

- Add unit tests for InferenceConfig resolution and per-engine wiring
- Consider per-field merging if partial Turn.Data overrides are needed
- Add `GeminiInferenceConfig` for provider-specific Gemini parameters (safety settings, function calling mode)
- Validate Claude thinking + temperature interaction at the config level
