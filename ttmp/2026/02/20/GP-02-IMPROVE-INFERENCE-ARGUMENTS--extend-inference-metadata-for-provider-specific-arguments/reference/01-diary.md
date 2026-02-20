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

## Step 4: Glazed Section for InferenceConfig — CLI/Config/Env/Profile Support

With the InferenceConfig types and engine wiring in place from Step 3, the next gap was that `StepSettings.Inference` was the only settings field not wired into the glazed parameter system — no CLI flags, no YAML config loading, no env vars, no profiles.

### Prompt Context

**User prompt (verbatim):** "Analyze how to add a glazed Schema/Section to load the inference settings from config, like the other chat parameter. Create a new analysis document in the ticket and upload to remarkable tablet."

Followed by: "are you sure that pointers become non-nil with glazed? how do you know?"

Then: "alright, add tasks to the ticket and then implement them, committing as you go, keeping a diary"

**Assistant interpretation:** Design and implement the glazed section integration so InferenceConfig is loadable from CLI flags, YAML config, env vars, and profiles.

**Commit (code):** 71e8154 — "feat(settings): add glazed section for InferenceConfig"

### What I did

1. **Analysis:** Wrote `analysis/02-glazed-section-for-inferenceconfig.md` exploring three options for where to place the section and the nil-pointer semantics question.

2. **Experiment:** Created `scripts/test_glazed_pointer_nil/` to verify whether glazed's `DecodeSectionInto` leaves pointer fields nil when the YAML flag definition has no `default:` key.

3. **Corrected my assumption:** I initially assumed glazed always populates pointer fields (making nil detection impossible), which would have required a wrapper struct with zero-to-nil mapping. The experiment proved this wrong — omitting `default:` from the YAML preserves nil pointers. This simplified the design significantly.

4. **Implementation:**
   - Added `yaml:` + `glazed:` tags to `engine.InferenceConfig` fields
   - Created `flags/inference.yaml` with slug `ai-inference` and **no default values**
   - Created `settings-inference.go` with `InferenceValueSection` and `AiInferenceSlug`
   - Registered section in `CreateGeppettoSections()`
   - Added `AiInferenceSlug` to env var whitelist in `GetCobraCommandGeppettoMiddlewares()`
   - Added `glazed:"ai-inference"` tag to `StepSettings.Inference`
   - Added `DecodeSectionInto(AiInferenceSlug, ss.Inference)` in `UpdateFromParsedValues()`

### Why

- Without the glazed section, users had no way to set inference overrides from CLI, config files, environment, or profiles
- Every other settings struct in StepSettings had a corresponding section — Inference was the only gap
- This enables use cases like: `pinocchio chat --inference-thinking-budget 8192`

### What worked

- **Omitting defaults preserves nil pointers.** The experiment confirmed that glazed's `FieldValuesFromDefaults()` skips definitions where `Default == nil`, and `DecodeInto()` skips fields not in the FieldValues map. Pointer fields stay nil unless explicitly provided.
- **Direct tagging on engine.InferenceConfig** — no wrapper struct needed. The `glazed:` tag is just a struct field annotation with no import requirement.
- **The existing section registration pattern** was easy to follow: create section, add to slice, add to whitelist, add decode call.

### What didn't work

1. **gofmt lint failure on first commit attempt.** The experiment script `main.go` had Unicode arrows (`→`) in comments. While valid Go, `gofmt` reformatted them and the pre-commit hook rejected the diff. Fixed by running `gofmt -w` on the file.

2. **Initial incorrect analysis.** I assumed glazed always populated pointer fields from defaults, which would have required a wrapper struct (`InferenceSettings`) with value types and a `ToInferenceConfig()` mapping function. The user correctly questioned this assumption, leading to the experiment that disproved it. The analysis document was updated to reflect the simpler approach.

### What I learned

- **Verify assumptions about library behavior experimentally.** Reading source code is good, but running a quick experiment in `scripts/` is faster and more reliable for behavioral questions.
- **Glazed nil semantics:** `default:` key present (even with zero value like `0`) → pointer becomes non-nil. `default:` key absent → pointer stays nil. This is the critical distinction for override-layer settings.
- **Choice fields without defaults:** Glazed supports `type: choice` with `choices:` but no `default:` — the pointer stays nil until the user explicitly provides a value.

### What was tricky to build

Nothing in the implementation was tricky — the pattern was well-established and the experiment removed all ambiguity. The tricky part was the analysis: correctly identifying that the nil-preservation question was the key design decision, and then verifying it experimentally rather than guessing.

### What warrants a second pair of eyes

- **Choice field behavior without defaults:** The YAML defines `inference-reasoning-effort` as `type: choice` with `choices: [low, medium, high]` but no default. I haven't tested whether glazed's Cobra flag binding handles this correctly (e.g., does `--help` show the choices? Does omitting the flag work?). Worth verifying in a real Cobra command.

### What should be done in the future

- Add an integration test that creates a command with the ai-inference section, parses known flags, and verifies the InferenceConfig fields are correctly populated (non-nil for provided flags, nil for omitted ones)
- Consider whether to expose `ClaudeInferenceConfig` and `OpenAIInferenceConfig` as sections too, or keep them as Turn.Data-only

### Code review instructions

**Start with:**
1. `pkg/steps/ai/settings/flags/inference.yaml` — verify no `default:` keys
2. `pkg/inference/engine/inference_config.go` — verify `glazed:` tag names match YAML flag names
3. `pkg/steps/ai/settings/settings-step.go` — verify `DecodeSectionInto` call

**Validate with:**
```bash
go build ./... && go test ./...
# Run the experiment:
go run ./ttmp/.../scripts/test_glazed_pointer_nil/
```

### Technical details

Files created/modified:
- `pkg/steps/ai/settings/flags/inference.yaml` (NEW) — 8 flags, no defaults
- `pkg/steps/ai/settings/settings-inference.go` (NEW) — ValueSection wrapper
- `pkg/inference/engine/inference_config.go` (MODIFIED) — added yaml:/glazed: tags
- `pkg/steps/ai/settings/settings-step.go` (MODIFIED) — glazed tag + decode call
- `pkg/sections/sections.go` (MODIFIED) — section registration + env whitelist
- `scripts/test_glazed_pointer_nil/main.go` (NEW) — experiment confirming nil behavior

## Step 5: Fix Sampling-Guard Bypass in InferenceConfig Overrides

A code review surfaced three P1 bugs where the InferenceConfig override blocks could reintroduce sampling parameters that earlier model-specific guards had deliberately omitted, causing provider API rejections.

### Prompt Context

**User prompt (verbatim):** Three review comments identifying that InferenceConfig overrides in Claude, OpenAI Responses, and OpenAI Chat helpers could bypass model-specific sampling guards (temperature/top_p exclusivity for Claude, reasoning-model omission for OpenAI).

**Assistant interpretation:** Fix all three provider helpers to respect model constraints after applying InferenceConfig overrides.

**Commit:** 0c06789 — "fix(providers): respect model constraints after InferenceConfig overrides"

### What I did

1. **Claude (`helpers.go`):** Added post-override re-validation:
   - Check that at most one of `req.Temperature`/`req.TopP` is set (Claude requires exclusivity)
   - Check that when thinking is enabled, temperature is 1.0 or unset (Claude constraint)
   - Both return errors with clear messages for fail-fast behavior

2. **OpenAI Responses (`helpers.go`):**
   - Extracted inline reasoning-model check into `isResponsesReasoningModel()` helper
   - Refactored the base-settings check to use the helper (reducing duplication)
   - Guarded temperature/top_p overrides behind `!isResponsesReasoningModel(req.Model)`

3. **OpenAI Chat (`helpers.go`):**
   - Cached `isReasoningModel(engine)` result into `reasoning` local
   - Guarded temperature/top_p overrides behind `!reasoning`

### Why

The original InferenceConfig override blocks were added after the model-specific guards in the request-building flow. They ran unconditionally, which meant:
- Claude: An InferenceConfig with both Temperature and TopP set would bypass the exclusivity error that fires on base chat settings
- OpenAI: A reasoning model (o1/o3/o4/gpt-5) with temperature/top_p overrides would send parameters the API rejects

### What worked

- All three fixes follow the same pattern: check the model constraint before applying the override
- The OpenAI Responses refactor (extracting `isResponsesReasoningModel`) also cleaned up the inline check in the base settings path

### What didn't work

N/A — all fixes compiled and passed lint on first attempt.

### What I learned

- Override blocks that run late in request construction must re-validate any constraints that earlier code enforced. The "apply overrides last" pattern is convenient but can bypass guards if not careful.
- The three providers had slightly different guard patterns (Claude: exclusivity error, OpenAI Responses: inline model prefix check, OpenAI Chat: `isReasoningModel` helper). Standardizing on extracted helpers reduces future risk.

### What was tricky to build

Nothing technically tricky — the fix pattern was straightforward once the bug was identified. The key insight was recognizing that all three bugs shared the same root cause: overrides applied after guards.

### What warrants a second pair of eyes

- **Claude thinking + temperature = 1.0 constraint:** The Claude API docs say temperature must be 1.0 when thinking is enabled. The current check rejects any temperature != 1.0. Should we auto-coerce to 1.0 instead of erroring? Current behavior: fail-fast with a clear error message.
- **OpenAI Responses reasoning model list:** The `isResponsesReasoningModel` check uses the same prefix list as the OpenAI Chat `isReasoningModel`. If these diverge in the future (different providers support different model families), they'd need separate maintenance.

### What should be done in the future

- Add unit tests that specifically exercise InferenceConfig overrides on reasoning models to prevent regression
- Consider consolidating the reasoning-model detection into a shared helper (both OpenAI packages use the same prefix list)

### Code review instructions

**Start with:** Each provider's InferenceConfig override block — look for the guard checks added before temperature/top_p assignment:
- `claude/helpers.go` — post-override validation block (~line 302-308)
- `openai_responses/helpers.go` — `isResponsesReasoningModel()` helper + `overrideAllowSampling` guard
- `openai/helpers.go` — `reasoning` local + `!reasoning` guards

### Technical details

All three bugs shared the same pattern:
```
1. Base settings applied with model constraints (guard A)
2. InferenceConfig overrides applied WITHOUT checking constraints
3. Request sent with invalid parameter combination → provider rejects
```

Fix pattern:
```
1. Base settings applied with model constraints (guard A)
2. InferenceConfig overrides applied WITH same constraint check
3. Request sent with valid parameters
```

## Step 6: Field-Level Merge, Sanitize Helpers, and Helper→Method Conversion

After the sampling-guard fixes in Step 5, a code review identified three remaining issues: (1) `ResolveInferenceConfig` used full-replacement instead of field-level merge, losing engine defaults when a caller set only one Turn.Data field; (2) reasoning-model constraint enforcement was duplicated inline across providers; (3) provider helper functions were standalone, making it awkward to add engine-level context. An analysis document (Option A/B/C) was written, and Option B was chosen.

### Prompt Context

**User prompt (verbatim):** "yes, then add tasks and build the clean solution."

**Assistant interpretation:** Implement the refactoring plan from analysis doc 03 using Option B (convert helpers to methods on engine structs).

**Commit:** 3bb7a62 — "refactor(providers): field-level merge, sanitize helpers, helper-to-method conversion"

### What I did

1. **MergeInferenceConfig** (`inference/engine/inference_config.go`):
   - Added `MergeInferenceConfig(turnCfg, engineDefault *InferenceConfig) *InferenceConfig` for field-level merge
   - Deep-copies pointer values and slices to prevent mutation of inputs
   - Updated `ResolveInferenceConfig` to use `MergeInferenceConfig` instead of full-replacement

2. **Sanitize helpers** (`inference/engine/inference_config_sanitize.go` — NEW):
   - `SanitizeForReasoningModel(cfg *InferenceConfig)` — clears Temperature, TopP
   - `SanitizeOpenAIForReasoningModel(cfg *OpenAIInferenceConfig)` — clears PresencePenalty, FrequencyPenalty, N

3. **Helper→method conversion** (Option B):
   - Claude: `MakeMessageRequestFromTurn(s, t)` → `(e *ClaudeEngine) MakeMessageRequestFromTurn(t)`
   - OpenAI Chat: `MakeCompletionRequestFromTurn(s, t)` → `(e *OpenAIEngine) MakeCompletionRequestFromTurn(t)`
   - OpenAI Responses: `buildResponsesRequest(s, t)` → `(e *Engine) buildResponsesRequest(t)`
   - Each now uses `e.settings` instead of a parameter

4. **Bug #2 fix**: OpenAI Chat PresencePenalty/FrequencyPenalty bypassed reasoning-model guards when set via `OpenAIInferenceConfig`. Fixed by upfront `SanitizeOpenAIForReasoningModel` before applying fields.

5. **Tests** (`inference/engine/inference_config_test.go` — NEW):
   - 6 tests for MergeInferenceConfig (both nil, turn nil, default nil, overrides, all fields, mutation safety)
   - 2 tests for SanitizeForReasoningModel (nil, clears fields)
   - 2 tests for SanitizeOpenAIForReasoningModel (nil, clears fields)
   - New test in openai/helpers_test.go for reasoning model penalty sanitization

### Why

- **Field-level merge** means callers can set `ThinkingBudget` on Turn.Data without losing the engine-level `Temperature` default
- **Sanitize helpers** centralize constraint enforcement — each provider checks model type and calls the sanitizer, replacing scattered inline guards
- **Method conversion** gives helpers access to `e.settings` naturally, eliminating the need to pass settings as a parameter

### What worked

- Option B was clean: engine structs are lightweight (`settings *StepSettings`), so test setup is just `newTestEngine(ss)` — one extra line
- No external callers of the helper functions existed (only in-package tests and the engine's RunInference), so the refactor was fully contained
- Upfront sanitize pattern is clearer than per-field guards — impossible to miss a field

### What didn't work

1. **MergeInferenceConfig pointer sharing**: Initial implementation used shallow copy, sharing pointers with input structs. `TestMergeInferenceConfig_DoesNotMutateInputs` caught it — mutating `*got.Temperature = 999.0` also changed `turn.Temperature`. Fixed by deep-copying: `v := *turnCfg.Temperature; merged.Temperature = &v`
2. **Unused import after method conversion**: Converting `MakeMessageRequestFromTurn` to a method removed the direct use of the `settings` package in `claude/helpers.go`. Same for `openai_responses/helpers.go`. Fixed by removing the imports.
3. **gofmt alignment**: Helper functions like `func intPtr(v int) *int { return &v }` aligned with different column spacing than gofmt expected. Pre-commit hook caught it.
4. **Unused `float64Ptr`**: After removing `float64Ptr` usage from openai test (penalty values now come from `StepSettings` not `InferenceConfig`), the function became unused. Lint caught it.

### What I learned

- Deep-copy is essential for merge functions that produce a new config from two inputs — shallow copy creates aliased pointers that break mutation safety
- The upfront-sanitize pattern (sanitize config, then apply all fields unconditionally) is cleaner than the guard-per-field pattern (check model type before each field assignment)
- When converting a standalone function to a method, always check for now-unused imports

### What was tricky to build

Nothing technically tricky. The mutation-safety bug was subtle but the test caught it immediately. The key design decision (Option B vs A vs C) was resolved in the analysis phase.

### What warrants a second pair of eyes

- **Merge semantics**: Turn.Data fields override individual engine defaults. An empty `Stop` slice in Turn.Data does NOT clear the engine default (only a non-empty slice overrides). This might surprise callers who want to explicitly clear stop sequences.
- **Sanitize placement**: Providers check `isReasoningModel()` themselves and conditionally call the sanitizer. Model-name knowledge stays in provider packages, which is correct, but means new providers must remember to add the check.

### What should be done in the future

- Integration tests that exercise the full Turn.Data → merge → sanitize → request pipeline
- Consider a `ClearStop` sentinel value if callers need to explicitly clear stop sequences

### Code review instructions

**Start with:**
1. `pkg/inference/engine/inference_config.go` — `MergeInferenceConfig` deep-copy logic
2. `pkg/inference/engine/inference_config_sanitize.go` — sanitize helpers
3. `pkg/inference/engine/inference_config_test.go` — mutation safety test
4. Any provider helper (e.g., `openai/helpers.go`) — method signature + upfront sanitize pattern

**Validate with:**
```bash
go build ./... && go test ./...
```

### Technical details

Files modified:
- `pkg/inference/engine/inference_config.go` — MergeInferenceConfig + updated ResolveInferenceConfig
- `pkg/inference/engine/inference_config_sanitize.go` (NEW) — sanitize functions
- `pkg/inference/engine/inference_config_test.go` (NEW) — 10 tests
- `pkg/steps/ai/claude/{helpers,engine_claude,helpers_test}.go` — method conversion
- `pkg/steps/ai/openai/{helpers,engine_openai,helpers_test}.go` — method conversion + Bug #2 fix
- `pkg/steps/ai/openai_responses/{helpers,engine,helpers_test}.go` — method conversion + sanitize
- Analysis doc 03 updated with Option B recommendation

## Related

- [Analysis: Extending Inference Arguments via Typed Turn.Data Keys](../design/01-analysis-inference-arguments.md)
- [Analysis: Glazed Section for InferenceConfig](../analysis/02-glazed-section-for-inferenceconfig.md)
- [Analysis: Rigorous Merge and Validation for InferenceConfig](../analysis/03-rigorous-merge-and-validation-for-inferenceconfig.md)
