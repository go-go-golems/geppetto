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
      Note: ResolveInferenceConfig (currently full-replacement, needs field-level merge)
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
LastUpdated: 2026-02-20T09:34:58.801351685-05:00
WhatFor: Planning a principled refactor of InferenceConfig resolution
WhenToUse: When implementing the merge/validation refactor
---

# Analysis: Rigorous Merge and Validation for InferenceConfig

## Two Open Bugs

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
    ↓ (field-level merge, but currently full-replacement)
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
| No penalties for reasoning models | OpenAI Chat | **MISSING** | Bug #2 |
| No N>1 for reasoning models | OpenAI Chat | `helpers.go:378` | Base settings zeroed |
| maxTokens→maxCompletionTokens | OpenAI Chat | `helpers.go:492` | Conditional field |

### Problems with the current approach

1. **Constraints are scattered.** Each provider has ad-hoc guards mixed into the
   override-application code. New constraints require touching the right provider
   file and knowing where to insert the check.

2. **Easy to miss.** We've already had two rounds of bug fixes (commit `0c06789` and
   the pending P2) for the same pattern: "override block bypasses guard". Each new
   field or provider-specific config adds more surface area for this class of bug.

3. **Merge is wrong.** `ResolveInferenceConfig` does full-replacement, not field-level
   merge. The "layered" documentation is aspirational, not actual.

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
    if len(turnCfg.Stop) > 0 {
        merged.Stop = turnCfg.Stop
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

### Recommendation

**Option A** for constraints (centralized functions in `engine` package) combined
with field-level merge. This addresses both bugs and reduces the scattered-guard
problem without over-engineering.

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

### Step 1: Add `MergeInferenceConfig` (fixes Bug #1)

- Add `MergeInferenceConfig(turn, default)` to `inference_config.go`
- Update `ResolveInferenceConfig` to use it
- Unit test: merge with various nil/non-nil field combinations

### Step 2: Add validation/sanitize functions

- `ValidateClaudeInferenceConfig(cfg) error`
- `SanitizeForReasoningModel(cfg) *InferenceConfig`
- `SanitizeOpenAIForReasoningModel(cfg) *OpenAIInferenceConfig`
- Unit tests for each

### Step 3: Refactor provider override blocks

- **Claude:** Replace inline constraints with `ValidateClaudeInferenceConfig` call
- **OpenAI Chat:** Call `SanitizeForReasoningModel` + `SanitizeOpenAIForReasoningModel` before applying (fixes Bug #2)
- **OpenAI Responses:** Call `SanitizeForReasoningModel` before applying
- **Gemini:** No constraints currently; leave as-is

### Step 4: Move `isReasoningModel` to shared location

Both `openai/helpers.go` and `openai_responses/helpers.go` have their own copy
of the same prefix-check logic. Consider moving to a shared helper in `engine`
or a new `providers` package. (Optional — could be a follow-up.)

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `pkg/inference/engine/inference_config.go` | MODIFY | Add MergeInferenceConfig, refactor ResolveInferenceConfig |
| `pkg/inference/engine/inference_config_validate.go` | CREATE | ValidateClaudeInferenceConfig, SanitizeForReasoningModel, SanitizeOpenAIForReasoningModel |
| `pkg/inference/engine/inference_config_test.go` | CREATE | Tests for merge + validation |
| `pkg/steps/ai/claude/helpers.go` | MODIFY | Replace inline constraints with ValidateClaudeInferenceConfig |
| `pkg/steps/ai/openai/helpers.go` | MODIFY | Add sanitize calls, fix penalty bug |
| `pkg/steps/ai/openai_responses/helpers.go` | MODIFY | Use SanitizeForReasoningModel |

## Open Questions

1. **Should merge be recursive for provider-specific configs?** Currently
   `ClaudeInferenceConfig` and `OpenAIInferenceConfig` have no engine-level
   defaults (Turn.Data only). If engine defaults are added, they'd need merge
   functions too. Design the merge pattern once and apply consistently.

2. **Suppress vs. warn?** When a reasoning model drops temperature from an
   override, should we log a warning? Currently silent. A warning would help
   callers notice their override was ignored, but could be noisy in normal
   operation.

3. **Should SanitizeForReasoningModel live in `engine` or in the provider package?**
   It references model-name prefixes which are provider-specific knowledge. But
   the InferenceConfig types live in `engine`. Putting the sanitizer in `engine`
   with the model prefix list is pragmatic but couples `engine` to provider
   details. Alternatively, providers could call a generic `ClearSamplingFields`
   and handle the model check themselves.
