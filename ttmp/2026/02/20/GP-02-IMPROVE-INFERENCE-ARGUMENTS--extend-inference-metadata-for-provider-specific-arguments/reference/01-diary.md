---
Title: Diary
Ticket: GP-02-IMPROVE-INFERENCE-ARGUMENTS
Status: active
Topics:
    - inference
    - metadata
    - architecture
    - geppetto
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/inference/engine/inference_config.go
      Note: NEW - Core InferenceConfig
    - Path: pkg/inference/engine/turnkeys.go
      Note: Added KeyInferenceConfig
    - Path: pkg/steps/ai/claude/api/completion.go
    - Path: pkg/steps/ai/claude/api/messages.go
      Note: Added ThinkingParam struct and Thinking field to MessageRequest (commit 36d93f1)
    - Path: pkg/steps/ai/claude/helpers.go
      Note: Wired InferenceConfig + ClaudeInferenceConfig + StructuredOutputConfig reading (commit 36d93f1)
    - Path: pkg/steps/ai/gemini/engine_gemini.go
      Note: Wired InferenceConfig reading for Temperature
    - Path: pkg/steps/ai/openai/helpers.go
      Note: Wired InferenceConfig + OpenAIInferenceConfig + StructuredOutputConfig reading (commit 36d93f1)
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: Wired InferenceConfig + OpenAIInferenceConfig + StructuredOutputConfig; added reasoning.max_tokens
    - Path: pkg/steps/ai/settings/settings-step.go
      Note: Added Inference *engine.InferenceConfig field to StepSettings (commit 36d93f1)
    - Path: pkg/turns/keys.go
      Note: Added value key constants for inference configs (commit 36d93f1)
ExternalSources: []
Summary: 'Implementation diary for GP-02: extending inference metadata with per-turn provider-specific arguments'
LastUpdated: 2026-02-20T08:07:05.719217457-05:00
WhatFor: Tracking the implementation journey of the layered InferenceConfig system
WhenToUse: When reviewing, continuing, or understanding the inference config work
---


# Diary

## Goal

Capture the implementation journey of extending geppetto's inference pipeline to support per-turn provider-specific arguments (thinking budget, reasoning effort, temperature overrides, etc.) using a layered InferenceConfig system that builds on the existing typed Turn.Data key pattern.

## Step 1: Codebase Exploration and Architecture Analysis

The first step was a deep exploration of geppetto's inference pipeline to understand how settings currently flow from callers to provider API requests, and to identify the gaps preventing per-turn control over provider-specific parameters.

### Prompt Context

**User prompt (verbatim):** "Analyze the geppetto/ codebase and our goal is to create a new ticket docmgr GP-02-IMPROVE-INFERENCE-ARGUMENTS. I want to make sure we can extend the metadata passed from the inference calls down to actual provider API calls in order to pass provider specific (or generalization like thinking budget) down to the actual LLM APIs. See for https://developers.openai.com/api/reference/resources/responses/methods/create I think a model like the typed turn metadata key design pattern would be useful here. The goal is to then store your analysis in a document in this newly created ticket."

**Assistant interpretation:** Explore the geppetto inference pipeline, catalog the gap between what provider APIs accept and what geppetto exposes, then design an extension using the existing typed Turn.Data key pattern.

**Inferred user intent:** Enable callers to control provider-specific LLM parameters (especially thinking/reasoning budget) on a per-turn basis, without breaking the existing settings architecture.

### What I did

- Explored the inference call chain: `Session.StartInference()` -> `Builder.Build()` -> `runner.RunInference()` -> `toolloop.Loop.RunLoop()` -> `eng.RunInference()` (provider-specific)
- Mapped the existing `StepSettings` structure (Chat, OpenAI, Claude, Gemini sub-structs)
- Studied the typed Turn.Data key pattern: `turns.DataK[T](namespace, value, version)` creating keys like `geppetto.tool_config@v1`
- Cataloged every parameter in Claude Messages API, OpenAI Responses API, OpenAI Chat Completions API, and Gemini API against what geppetto currently supports
- Fetched the OpenAI Responses API reference for the complete parameter list
- Created docmgr ticket GP-02-IMPROVE-INFERENCE-ARGUMENTS
- Wrote comprehensive analysis document `design/01-analysis-inference-arguments.md`

### Why

- Need to understand the full settings flow before designing the extension
- Need to know exactly which provider parameters are unsupported to scope the work
- The existing KeyToolConfig/KeyStructuredOutputConfig pattern is the natural extension point

### What worked

- The exploration quickly revealed a clean pattern: typed Turn.Data keys are already used for ToolConfig and StructuredOutputConfig, making the extension natural
- Import graph analysis showed that `settings` -> `engine` is already established (no cycle risk)
- Found that `KeyStructuredOutputConfig` was declared but never consumed — an existing gap to complete

### What didn't work

- N/A (exploration phase)

### What I learned

- Turn.Data keys for engine types must live in `inference/engine` (not `turns`) to avoid import cycles since `engine` imports `turns`
- The `GenerationConfig` field in genai's model is embedded, so `model.GenerationConfig.Temperature` should be simplified to `model.Temperature`
- Claude's `Metadata` struct existed in the API types but was always set to nil — no path for `user_id`
- Claude had no `Thinking` field in `MessageRequest` at all

### What was tricky to build

N/A (exploration phase)

### What warrants a second pair of eyes

- The analysis document's completeness — are there provider parameters we missed that callers will need?

### What should be done in the future

N/A

### Code review instructions

- Review `design/01-analysis-inference-arguments.md` for the gap analysis and proposed design

### Technical details

Key files studied:
- `turns/key_families.go` — DataKey[T] generic infrastructure
- `inference/engine/turnkeys.go` — existing KeyToolConfig, KeyStructuredOutputConfig declarations
- `inference/engine/types.go` — ToolConfig type
- Provider helpers: `claude/helpers.go`, `openai/helpers.go`, `openai_responses/helpers.go`, `gemini/engine_gemini.go`

## Step 2: Design Decision — Layered InferenceConfig with Engine Defaults

After the analysis, the design needed a key refinement: the user wanted InferenceConfig to be settable at engine creation time (via StepSettings) so engines can be configured with high thinking budget from the start, with Turn.Data providing per-turn overrides.

### Prompt Context

**User prompt (verbatim):** "So I think the inference config should be passed at engine creation time, and can be overriden by Turn data if present. That way I can configure an engine to have high thinking from the start"

**Assistant interpretation:** Add `Inference *engine.InferenceConfig` to `StepSettings` for engine-level defaults, with Turn.Data as per-turn override.

**Inferred user intent:** Support configuring thinking budget (and other inference parameters) at engine construction time as defaults, while still allowing per-turn overrides for specific inference calls.

### What I did

- Refined the design to a three-layer resolution: Turn.Data > StepSettings.Inference > StepSettings.Chat fields > defaults
- Verified the import graph: `settings-chat.go` already imports `engine`, so adding `Inference *engine.InferenceConfig` to StepSettings is cycle-safe
- Wrote the implementation plan with this layered approach

### Why

- Engine-level defaults mean callers don't need to set Turn.Data keys on every turn — they configure once
- Per-turn overrides allow dynamic adjustment (e.g., higher thinking budget for complex turns)
- This is ergonomically clean: set-and-forget for common cases, explicit override for special cases

### What worked

- The import graph already supported this direction
- `StepSettings.Clone()` and `GetMetadata()` patterns were straightforward to extend

### What didn't work

- N/A

### What I learned

- The layered design is more ergonomic than Turn.Data-only approach
- Resolution helpers (`ResolveInferenceConfig(turn, engineDefault)`) cleanly encapsulate the precedence logic

### What was tricky to build

N/A (design phase)

### What warrants a second pair of eyes

- The resolution precedence: Turn.Data fully replaces StepSettings.Inference (no per-field merge). This is the simplest approach but means a per-turn override must specify all desired fields, not just the ones to change.

### What should be done in the future

- Consider per-field merging if callers need to override just one field while keeping engine defaults for others

### Code review instructions

- Review the resolution logic in `inference_config.go` — `ResolveInferenceConfig()`, `ResolveClaudeInferenceConfig()`, `ResolveOpenAIInferenceConfig()`

### Technical details

Resolution order:
```
Turn.Data InferenceConfig > StepSettings.Inference > StepSettings.Chat fields > defaults
```

## Step 3: Implementation — Types, Keys, and Engine Wiring

The bulk implementation: creating the InferenceConfig types, declaring Turn.Data keys, adding the `Inference` field to StepSettings, and wiring all four provider engines to read per-turn overrides.

### Prompt Context

**User prompt (verbatim):** (same as Step 1 — the implementation was part of the approved plan)

**Assistant interpretation:** Implement the layered InferenceConfig system across all provider engines.

**Inferred user intent:** Complete the feature so callers can control thinking budget, reasoning effort, and other provider-specific parameters both at engine creation and per-turn.

**Commit (code):** 36d93f1 — "feat(inference): add per-turn InferenceConfig with engine-level defaults"

### What I did

- Created `inference/engine/inference_config.go` with:
  - `InferenceConfig` (cross-provider: ThinkingBudget, ReasoningEffort, Temperature, TopP, MaxResponseTokens, Stop, Seed)
  - `ClaudeInferenceConfig` (UserID, TopK)
  - `OpenAIInferenceConfig` (N, PresencePenalty, FrequencyPenalty, Store, ServiceTier)
  - Resolution helpers: `ResolveInferenceConfig()`, `ResolveClaudeInferenceConfig()`, `ResolveOpenAIInferenceConfig()`
- Added 3 value key constants to `turns/keys.go`
- Added 3 typed key declarations to `inference/engine/turnkeys.go`
- Added `Inference *engine.InferenceConfig` to `StepSettings`, updated `Clone()` and `GetMetadata()`
- Claude engine:
  - Added `ThinkingParam` struct and `Thinking *ThinkingParam` field to `MessageRequest`
  - Expanded `Metadata` struct with `UserID` field
  - Wired InferenceConfig reading in `MakeMessageRequestFromTurn` (ThinkingBudget, Temperature, TopP, MaxResponseTokens, Stop)
  - Wired ClaudeInferenceConfig reading (UserID, TopK)
  - Wired KeyStructuredOutputConfig from Turn.Data (completing existing gap)
- OpenAI Responses engine:
  - Added `MaxTokens *int` to `reasoningParam` struct
  - Added `Store *bool` and `ServiceTier *string` to `responsesRequest`
  - Wired InferenceConfig reading (ThinkingBudget -> reasoning.max_tokens, ReasoningEffort, ReasoningSummary, Temperature, TopP, MaxResponseTokens, Stop)
  - Wired OpenAIInferenceConfig reading (Store, ServiceTier)
  - Wired KeyStructuredOutputConfig from Turn.Data
- OpenAI Chat engine:
  - Wired InferenceConfig reading (Temperature, TopP, MaxResponseTokens, Stop, Seed)
  - Wired OpenAIInferenceConfig reading (N, PresencePenalty, FrequencyPenalty)
  - Wired KeyStructuredOutputConfig from Turn.Data
- Gemini engine:
  - Wired InferenceConfig reading (Temperature, TopP, MaxResponseTokens)

### Why

- Enables callers to control provider-specific parameters dynamically
- Completes the existing KeyStructuredOutputConfig gap across all engines
- Follows the established Turn.Data key pattern for consistency

### What worked

- The existing typed key infrastructure (`DataK[T]`, `Get`/`Set`) made adding new keys trivial
- The resolution helper pattern cleanly separates the precedence logic from engine code
- All existing tests continued to pass, confirming backward compatibility

### What didn't work

1. **Import name collision**: `engine.ResolveInferenceConfig` failed to compile in `claude/helpers.go` and `openai/helpers.go` because the package import `engine` was shadowed by a local variable `engine` (the model name string). Fixed by aliasing the import to `infengine`.

2. **Lint failure on first commit attempt**: Pre-commit hook (golangci-lint/staticcheck QF1008) rejected `model.GenerationConfig.Temperature` in `gemini/engine_gemini.go` — since `GenerationConfig` is an embedded field, the selector should be simplified to `model.Temperature`. Fixed by removing the embedded field from the selector.

### What I learned

- In Go, when a package import name collides with a local variable name, the local variable wins. Import aliasing (`infengine`) is the clean fix.
- The genai library uses an embedded `GenerationConfig` struct in its model type, so the linter correctly flags redundant selectors like `model.GenerationConfig.Temperature`.
- Pre-commit hooks that run linting are valuable — they caught the embedded field issue before the commit landed.

### What was tricky to build

**Import aliasing in claude/helpers.go and openai/helpers.go**: Both files had a local variable named `engine` (the model engine string from `ChatSettings.Engine`). When I added `engine "github.com/.../inference/engine"`, the Go compiler couldn't distinguish between the package and the variable. The error message was confusing: `engine.ResolveInferenceConfig undefined (type string has no field or method ResolveInferenceConfig)` — it looked like a method resolution issue when it was actually a name collision. The fix was straightforward once diagnosed: alias the import to `infengine`.

**Gemini embedded field lint**: The staticcheck QF1008 diagnostic was new to me. The Gemini `genai.GenerativeModel` embeds `GenerationConfig`, so accessing `model.GenerationConfig.Temperature` is valid Go but unnecessarily verbose. The linter (correctly) suggests `model.Temperature`. This applies to all three fields I set: Temperature, TopP, MaxOutputTokens.

### What warrants a second pair of eyes

- **Claude thinking + temperature interaction**: When ThinkingBudget is set, Claude may have restrictions on temperature (must be 1.0 or unset). The current code doesn't enforce this constraint. The caller is responsible for valid parameter combinations.
- **OpenAI reasoning model max_tokens**: For OpenAI reasoning models (o1/o3), `max_completion_tokens` is used instead of `max_tokens`. The InferenceConfig override for MaxResponseTokens goes through the existing reasoning-model-aware code path, but this interaction should be verified.
- **Resolution semantics**: Turn.Data InferenceConfig fully replaces StepSettings.Inference (no per-field merge). If a caller sets `KeyInferenceConfig` with only `ThinkingBudget`, the engine-level Temperature default from `StepSettings.Inference` is lost. This is intentional (simpler) but worth confirming is the desired behavior.

### What should be done in the future

- Add unit tests specifically for the InferenceConfig resolution and per-engine wiring
- Consider per-field merging of InferenceConfig if callers need partial overrides
- Wire the Gemini-specific InferenceConfig (safety settings, function calling mode) when needed
- Add GeminiInferenceConfig for provider-specific Gemini parameters

### Code review instructions

**Where to start:**
1. `pkg/inference/engine/inference_config.go` — the core types and resolution helpers
2. `pkg/inference/engine/turnkeys.go` — new typed key declarations
3. `pkg/steps/ai/claude/helpers.go:277-323` — Claude engine wiring (InferenceConfig + ClaudeInferenceConfig + StructuredOutputConfig)
4. `pkg/steps/ai/openai_responses/helpers.go` — OpenAI Responses engine wiring
5. `pkg/steps/ai/openai/helpers.go` — OpenAI Chat engine wiring

**How to validate:**
```bash
go build ./...
go test ./... -count=1
```

### Technical details

Cross-provider field mapping:

| InferenceConfig Field | Claude | OpenAI Responses | OpenAI Chat | Gemini |
|-----------------------|--------|-----------------|-------------|--------|
| ThinkingBudget | thinking.budget_tokens | reasoning.max_tokens | ignored | ignored |
| ReasoningEffort | ignored | reasoning.effort | ignored | ignored |
| Temperature | temperature | temperature | temperature | temperature |
| TopP | top_p | top_p | top_p | top_p |
| MaxResponseTokens | max_tokens | max_output_tokens | max_tokens | max_output_tokens |
| Stop | stop_sequences | stop_sequences | stop | ignored |
| Seed | ignored | ignored | seed | ignored |

## Related

- [Analysis: Extending Inference Arguments via Typed Turn.Data Keys](../design/01-analysis-inference-arguments.md)
