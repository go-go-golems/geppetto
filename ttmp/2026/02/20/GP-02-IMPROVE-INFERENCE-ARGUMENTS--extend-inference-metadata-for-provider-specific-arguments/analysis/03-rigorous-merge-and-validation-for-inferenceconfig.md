---
Title: Rigorous Merge and Validation for InferenceConfig
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
      Note: MergeInferenceConfig + ResolveInferenceConfig field-level merge semantics
    - Path: pkg/steps/ai/claude/helpers.go
      Note: Claude override block + post-override validation
    - Path: pkg/steps/ai/openai/helpers.go
      Note: OpenAI Chat override block + reasoning model guards
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: OpenAI Responses override block + reasoning model guards
    - Path: pkg/steps/ai/gemini/engine_gemini.go
      Note: Gemini override block (no constraints currently)
ExternalSources: []
Summary: 'Analysis of how to refactor InferenceConfig override resolution from scattered per-provider imperative code into a structured merge + validate pipeline, addressing field-level merge and constraint enforcement.'
LastUpdated: 2026-02-20T11:52:25-05:00
WhatFor: Planning a principled refactor of InferenceConfig resolution
WhenToUse: When implementing the merge/validation refactor
---

# Analysis: Rigorous Merge and Validation for InferenceConfig

## Original Bugs (Now Fixed)

Status update (2026-02-20):
- Bug 1 (full-replacement merge) fixed in commit `3bb7a62`.
- Bug 2 (reasoning-model penalty bypass) fixed in commit `3bb7a62`.
- Stop explicit-clear leak across builders fixed in commit `2e0b55e`.

### Bug 1: Full-replacement instead of field-level merge (P1)

`ResolveInferenceConfig` returns the Turn.Data config as-is when present, dropping
all engine defaults:

```go
func ResolveInferenceConfig(t *turns.Turn, engineDefault *InferenceConfig) *InferenceConfig {
    if t != nil {
        if cfg, ok, err := KeyInferenceConfig.Get(t.Data); err == nil && ok {
            return &cfg  // ← engine defaults entirely lost
        }
    }
    return engineDefault
}
```

A caller who sets `KeyInferenceConfig` with only `Stop: ["<END>"]` will lose
engine-level `ThinkingBudget`, `Temperature`, etc. This contradicts the documented
"layered" semantics.

### Bug 2: PresencePenalty/FrequencyPenalty bypass reasoning-model guard (P2)

In `openai/helpers.go`, the `OpenAIInferenceConfig` override block writes
`PresencePenalty` and `FrequencyPenalty` unconditionally:

```go
if oaiCfg.PresencePenalty != nil {
    req.PresencePenalty = float32(*oaiCfg.PresencePenalty)
}
```

But the base-settings path zeroes these for reasoning models (o1/o3/o4/gpt-5).
Same class of bypass as the temperature/top_p issue fixed in commit `0c06789`.

## Current State: How Overrides Work

### Resolution layers (intended)

```
Turn.Data InferenceConfig
    ↓ (field-level merge)
StepSettings.Inference
    ↓ (override)
StepSettings.Chat / StepSettings.OpenAI / StepSettings.Claude
    ↓ (defaults)
Provider API request
```

### Where constraints are enforced today

Each provider has its own imperative code that enforces model-specific rules:

| Constraint | Provider | Where | How |
|------------|----------|-------|-----|
| temperature XOR top_p | Claude | `helpers.go:304` | Post-override error |
| temperature=1 when thinking | Claude | `helpers.go:308` | Post-override error |
| No temp/topP for reasoning models | OpenAI Chat | `helpers.go:484-489` | Guard before write |
| No temp/topP for reasoning models | OpenAI Responses | `helpers.go:189-194` | Guard before write |
| No penalties for reasoning models | OpenAI Chat | Suppress | Yes (sanitized) |
| No N>1 for reasoning models | OpenAI Chat | `helpers.go:378` | Base settings zeroed |
| maxTokens→maxCompletionTokens | OpenAI Chat | `helpers.go:492` | Conditional field |

### Problems with the current approach

1. **Constraints are scattered.** Each provider has ad-hoc guards mixed into the
   override-application code. New constraints require touching the right provider
   file and knowing where to insert the check.

2. **Easy to miss.** We've already had multiple rounds of bug fixes (commit `0c06789`,
   commit `3bb7a62`, and commit `2e0b55e`) for the same pattern: "override block bypasses guard". Each new
   field or provider-specific config adds more surface area for this class of bug.

3. **Merge semantics are now correct in code.** This section is retained for postmortem context:
   `ResolveInferenceConfig` now delegates to `MergeInferenceConfig` with field-level merge.

4. **No single source of truth** for what each provider accepts. The model-prefix
   checks (`isReasoningModel`, `isResponsesReasoningModel`) are duplicated across
   packages with the same logic.

## Proposed Design: Merge + Validate Pipeline

### Layer 1: Field-Level Merge (in `engine` package)

Replace `ResolveInferenceConfig` with a merge function:

```go
// MergeInferenceConfig returns a new InferenceConfig where turn-level fields
// take precedence over engine defaults. Nil fields in the turn config fall
// back to the corresponding engine default field.
func MergeInferenceConfig(turnCfg, engineDefault *InferenceConfig) *InferenceConfig {
    if turnCfg == nil {
        return engineDefault
    }
    if engineDefault == nil {
        return turnCfg
    }
    merged := *engineDefault // shallow copy of defaults
    if turnCfg.ThinkingBudget != nil {
        merged.ThinkingBudget = turnCfg.ThinkingBudget
    }
    if turnCfg.ReasoningEffort != nil {
        merged.ReasoningEffort = turnCfg.ReasoningEffort
    }
    if turnCfg.ReasoningSummary != nil {
        merged.ReasoningSummary = turnCfg.ReasoningSummary
    }
    if turnCfg.Temperature != nil {
        merged.Temperature = turnCfg.Temperature
    }
    if turnCfg.TopP != nil {
        merged.TopP = turnCfg.TopP
    }
    if turnCfg.MaxResponseTokens != nil {
        merged.MaxResponseTokens = turnCfg.MaxResponseTokens
    }
    if turnCfg.Stop != nil {
        merged.Stop = make([]string, len(turnCfg.Stop))
        copy(merged.Stop, turnCfg.Stop)
    }
    if turnCfg.Seed != nil {
        merged.Seed = turnCfg.Seed
    }
    return &merged
}
```

`ResolveInferenceConfig` changes to:

```go
func ResolveInferenceConfig(t *turns.Turn, engineDefault *InferenceConfig) *InferenceConfig {
    var turnCfg *InferenceConfig
    if t != nil {
        if cfg, ok, err := KeyInferenceConfig.Get(t.Data); err == nil && ok {
            turnCfg = &cfg
        }
    }
    return MergeInferenceConfig(turnCfg, engineDefault)
}
```

The same pattern applies to `ClaudeInferenceConfig` and `OpenAIInferenceConfig`:
their Resolve functions currently only read Turn.Data (no engine default), but
if engine defaults are added later, they'd follow the same merge pattern.

### Layer 2: Provider Constraint Validation

Rather than scattering guards inside each override block, define constraints
declaratively and validate the merged config against them.

**Option A: Centralized constraint functions (recommended for now)**

Add validation functions to the `engine` package that providers call after
merging:

```go
// ValidateClaudeInferenceConfig checks Claude-specific constraints.
func ValidateClaudeInferenceConfig(cfg *InferenceConfig) error {
    if cfg == nil {
        return nil
    }
    if cfg.Temperature != nil && cfg.TopP != nil {
        return errors.New("Claude requires only one of temperature/top_p")
    }
    if cfg.ThinkingBudget != nil && *cfg.ThinkingBudget > 0 {
        if cfg.Temperature != nil && *cfg.Temperature != 1.0 {
            return fmt.Errorf("Claude thinking requires temperature=1.0, got %.2f",
                *cfg.Temperature)
        }
    }
    return nil
}

// SanitizeForReasoningModel clears fields that reasoning models reject.
// Returns a copy; does not mutate the input.
func SanitizeForReasoningModel(cfg *InferenceConfig) *InferenceConfig {
    if cfg == nil {
        return nil
    }
    sanitized := *cfg
    sanitized.Temperature = nil
    sanitized.TopP = nil
    return &sanitized
}
```

Provider code becomes:

```go
// Claude
infCfg := engine.ResolveInferenceConfig(t, s.Inference)
if err := engine.ValidateClaudeInferenceConfig(infCfg); err != nil {
    return nil, err
}
// apply fields...

// OpenAI (reasoning model)
infCfg := engine.ResolveInferenceConfig(t, settings.Inference)
if isReasoningModel(engine) {
    infCfg = engine.SanitizeForReasoningModel(infCfg)
}
// apply fields...
```

**Pro:** Validation logic is testable in isolation, lives next to the config types,
and providers just call the right function.

**Con:** Still requires each provider to know which validation to call. But this is
inherent — each provider has different rules.

**Option B: Constraint registry (over-engineered for now)**

Define constraints as data:

```go
type Constraint struct {
    Name    string
    Check   func(*InferenceConfig) error
}

var ClaudeConstraints = []Constraint{
    {Name: "temp-xor-topp", Check: checkTempXorTopP},
    {Name: "thinking-temp", Check: checkThinkingTemp},
}
```

**Pro:** Can iterate and report all violations at once.
**Con:** Indirection without clear benefit given only 4 providers. YAGNI.

**Option C: Apply via typed Provider interface method**

Add a method to the Engine interface:

```go
type Engine interface {
    RunInference(ctx, t) error
    ValidateInferenceConfig(cfg *InferenceConfig) error  // NEW
}
```

**Pro:** Polymorphic dispatch, each engine defines its own rules.
**Con:** InferenceConfig validation happens at request-build time inside helpers,
not at the Engine level. The Engine doesn't build requests — helpers do. This
would require restructuring the call chain.

### Recommendation (updated after discussion)

**Option B** — make helpers methods on their engine structs — combined with
centralized sanitize/validate functions in the `engine` package and field-level
merge.

After discussion, Option B is preferred because:

1. **Helpers become methods** on ClaudeEngine / OpenAIEngine / Engine (responses).
   They lose their `settings` parameter (access via `e.settings`), but gain
   natural access to the engine struct for validation dispatch.

2. **Standalone testability is not lost.** The engine structs are lightweight
   (`settings *StepSettings` + optional adapter). Constructing one for tests is
   one extra line: `e := &Engine{settings: settings}`.

3. **No signature gymnastics.** No extra parameters, no function-pointer injection,
   no Turn.Data mutation. The helpers call sanitize/validate at the right internal
   point using centralized functions from the `engine` package.

4. **Sanitize functions live in `engine` but don't check model names.** Providers
   call `SanitizeForReasoningModel(cfg)` / `SanitizeOpenAIForReasoningModel(cfg)`
   after their own `isReasoningModel()` check. This keeps model-name knowledge in
   the provider and generic field-clearing in `engine`.

5. **Claude validation stays post-application.** Claude constraints involve the
   interaction between ChatSettings and InferenceConfig (e.g., ChatSettings.TopP +
   InferenceConfig.Temperature = conflict). These are correctly validated on the
   final request state, not on InferenceConfig alone.

### Layer 3: OpenAIInferenceConfig penalty guard

For the specific P2 bug, `SanitizeForReasoningModel` should also handle the
OpenAI-specific config:

```go
func SanitizeOpenAIForReasoningModel(cfg *OpenAIInferenceConfig) *OpenAIInferenceConfig {
    if cfg == nil {
        return nil
    }
    sanitized := *cfg
    sanitized.PresencePenalty = nil
    sanitized.FrequencyPenalty = nil
    sanitized.N = nil // reasoning models also reject N>1
    return &sanitized
}
```

## Inventory of All Constraints

| # | Constraint | Provider(s) | Type | Currently Enforced? |
|---|-----------|-------------|------|-------------------|
| 1 | At most one of temperature/top_p | Claude | Error | Yes (post-override) |
| 2 | temperature=1.0 when thinking | Claude | Error | Yes (post-override) |
| 3 | No temperature for reasoning models | OpenAI Chat, Responses | Suppress | Yes (guards) |
| 4 | No top_p for reasoning models | OpenAI Chat, Responses | Suppress | Yes (guards) |
| 5 | No presence_penalty for reasoning models | OpenAI Chat | Suppress | **No (Bug #2)** |
| 6 | No frequency_penalty for reasoning models | OpenAI Chat | Suppress | **No (Bug #2)** |
| 7 | No N>1 for reasoning models | OpenAI Chat | Suppress | Partial (base only) |
| 8 | maxTokens→maxCompletionTokens for reasoning | OpenAI Chat | Remap | Yes |
| 9 | temperature [0,2] range | All (API enforced) | API error | Not checked locally |
| 10 | top_p [0,1] range | All (API enforced) | API error | Not checked locally |

### Error vs. Suppress vs. Remap

- **Error:** Return an error to the caller (Claude constraints). The caller made
  an invalid combination; fail fast.
- **Suppress:** Silently drop the unsupported field (OpenAI reasoning models).
  The model just doesn't support the parameter; sending it causes an API error
  but it's not a semantic mistake by the caller.
- **Remap:** Convert one field to another (maxTokens→maxCompletionTokens).

The distinction matters: Claude constraints are *semantic errors* (the caller
asked for something contradictory), while OpenAI reasoning-model constraints are
*compatibility suppression* (the parameter exists but this model doesn't support it).

## Implementation Plan

### Step 1: Add `MergeInferenceConfig` + sanitize functions to `engine` package

- Add `MergeInferenceConfig(turnCfg, engineDefault)` to `inference_config.go`
- Update `ResolveInferenceConfig` to use `MergeInferenceConfig`
- Add `SanitizeForReasoningModel(cfg) *InferenceConfig` (clears Temperature, TopP)
- Add `SanitizeOpenAIForReasoningModel(cfg) *OpenAIInferenceConfig` (clears penalties, N)
- Unit tests for merge + sanitize

### Step 2: Convert Claude helper to method on ClaudeEngine

- `MakeMessageRequestFromTurn(s, t)` → `(e *ClaudeEngine) MakeMessageRequestFromTurn(t)`
- Uses `e.settings` instead of parameter
- Uses MergeInferenceConfig (via updated ResolveInferenceConfig)
- Claude post-application validation stays on request state (catches ChatSettings +
  InferenceConfig cross-source conflicts)

### Step 3: Convert OpenAI Chat helper to method on OpenAIEngine

- `MakeCompletionRequestFromTurn(settings, t)` → `(e *OpenAIEngine) MakeCompletionRequestFromTurn(t)`
- Uses `e.settings` instead of parameter
- For reasoning models: sanitize merged InferenceConfig + OpenAIInferenceConfig
  upfront (replaces per-field guards, **fixes Bug #2**)

### Step 4: Convert OpenAI Responses helper to method on Engine

- `buildResponsesRequest(s, t)` → `(e *Engine) buildResponsesRequest(t)`
- Uses `e.settings` instead of parameter
- For reasoning models: sanitize merged InferenceConfig upfront

### Step 5: Update Gemini

- Already inline in RunInference; just benefits from MergeInferenceConfig in
  ResolveInferenceConfig.

### Step 6: Update tests

- Update `helpers_test.go` in each package to construct engine structs

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `pkg/inference/engine/inference_config.go` | MODIFY | Add MergeInferenceConfig, refactor ResolveInferenceConfig |
| `pkg/inference/engine/inference_config_sanitize.go` | CREATE | SanitizeForReasoningModel, SanitizeOpenAIForReasoningModel |
| `pkg/inference/engine/inference_config_test.go` | CREATE | Tests for merge + sanitize |
| `pkg/steps/ai/claude/helpers.go` | MODIFY | Convert to method, use merged config |
| `pkg/steps/ai/claude/helpers_test.go` | MODIFY | Construct ClaudeEngine for tests |
| `pkg/steps/ai/claude/engine_claude.go` | MODIFY | Update call site |
| `pkg/steps/ai/openai/helpers.go` | MODIFY | Convert to method, sanitize for reasoning models |
| `pkg/steps/ai/openai/helpers_test.go` | MODIFY | Construct OpenAIEngine for tests |
| `pkg/steps/ai/openai/engine_openai.go` | MODIFY | Update call site |
| `pkg/steps/ai/openai_responses/helpers.go` | MODIFY | Convert to method, sanitize for reasoning models |
| `pkg/steps/ai/openai_responses/helpers_test.go` | MODIFY | Construct Engine for tests |
| `pkg/steps/ai/openai_responses/engine.go` | MODIFY | Update call site |

## Open Questions

1. **Should merge be recursive for provider-specific configs?** Currently
   `ClaudeInferenceConfig` and `OpenAIInferenceConfig` have no engine-level
   defaults (Turn.Data only). If engine defaults are added, they'd need merge
   functions too. Design the merge pattern once and apply consistently.

2. **Suppress vs. warn?** When a reasoning model drops temperature from an
   override, should we log a warning? Currently silent. A warning would help
   callers notice their override was ignored, but could be noisy in normal
   operation.

3. **Resolved: Where do sanitize functions live?** Sanitize functions live in
   `engine` but do NOT check model names. They only clear fields. Providers
   call `isReasoningModel()` themselves and conditionally invoke the sanitizer.
   This keeps model-name knowledge in the provider package.
