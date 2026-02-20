---
Title: Glazed Section for InferenceConfig
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
      Note: InferenceConfig, ClaudeInferenceConfig, OpenAIInferenceConfig types
    - Path: pkg/steps/ai/settings/settings-step.go
      Note: StepSettings.Inference field (currently no glazed tag or section)
    - Path: pkg/steps/ai/settings/settings-chat.go
      Note: ChatSettings with glazed tags + ChatValueSection pattern to follow
    - Path: pkg/steps/ai/settings/flags/chat.yaml
      Note: ai-chat section YAML definition to reference as template
    - Path: pkg/steps/ai/settings/openai/settings.go
      Note: OpenAI ValueSection pattern to follow
    - Path: pkg/sections/sections.go
      Note: CreateGeppettoSections() and GetCobraCommandGeppettoMiddlewares()
ExternalSources: []
Summary: 'Analysis of how to add a glazed schema/section for InferenceConfig so inference settings are loadable from YAML config, CLI flags, environment variables, and profiles — the same way all other chat parameters work.'
LastUpdated: 2026-02-20T08:53:59.047008108-05:00
WhatFor: Planning the implementation of glazed section integration for InferenceConfig
WhenToUse: When implementing the glazed section for inference settings
---

# Analysis: Glazed Section for InferenceConfig

## Problem

`StepSettings.Inference *engine.InferenceConfig` was added in GP-02 commit `36d93f1` but is currently **not wired into the glazed section/flag system**. This means:

- No CLI flags (e.g., `--inference-thinking-budget 8192`)
- No config file loading (e.g., `inference:` YAML section)
- No environment variable support (e.g., `PINOCCHIO_INFERENCE_THINKING_BUDGET`)
- No profile support (e.g., `high-thinking:` profile with inference settings)

Every other settings struct in `StepSettings` — Chat, Client, Claude, OpenAI, Gemini, Ollama, Embeddings — has a corresponding glazed section. `Inference` is the only gap.

## Current Pattern

All existing settings follow the same four-part pattern:

### 1. YAML flag definition (embedded)

```yaml
# flags/xxx.yaml
slug: section-slug
name: Human-readable section name
flags:
  - name: flag-name
    type: int|float|string|bool|choice|stringList
    help: Description
    default: value
```

### 2. Settings struct with `glazed:` tags

```go
type Settings struct {
    Field *Type `yaml:"yaml_name,omitempty" glazed:"flag-name"`
}
```

The `glazed:` tag value must match the `name:` in the YAML flag definition exactly.

### 3. ValueSection wrapper

```go
//go:embed "flags/xxx.yaml"
var settingsYAML []byte

const SectionSlug = "section-slug"

type ValueSection struct {
    *schema.SectionImpl `yaml:",inline"`
}

func NewValueSection(opts ...schema.SectionOption) (*ValueSection, error) {
    ret, err := schema.NewSectionFromYAML(settingsYAML, opts...)
    // ...
}
```

### 4. Registration in three places

| Location | What | Why |
|----------|------|-----|
| `sections.CreateGeppettoSections()` | Create section, initialize defaults from StepSettings | Makes section available to commands |
| `sections.GetCobraCommandGeppettoMiddlewares()` | Add slug to env var whitelist | Enables `PINOCCHIO_*` env var loading |
| `settings.UpdateFromParsedValues()` | `DecodeSectionInto(slug, target)` | Populates StepSettings from parsed values |

## Design Decision: Where Should InferenceConfig Live?

There are three options for where to define the InferenceConfig section:

### Option A: New section in `inference/engine/` package

Place the YAML and ValueSection next to `inference_config.go` in `inference/engine/`.

**Pro:** Keeps type + section co-located.
**Con:** The `engine` package currently has no glazed dependency. Adding `schema` import just for a section wrapper adds coupling. Also breaks the convention that all settings sections live under `steps/ai/settings/`.

### Option B: New sub-package `steps/ai/settings/inference/`

Follow the exact same pattern as `settings/claude/`, `settings/openai/`, `settings/gemini/`.

```
steps/ai/settings/inference/
    inference.yaml      # flag definitions
    settings.go         # InferenceSettings struct + ValueSection
```

**Pro:** Follows established convention exactly. Clean separation.
**Con:** Introduces a wrapper struct (`inference.Settings`) that mirrors `engine.InferenceConfig`, requiring mapping between them. Or it imports `engine.InferenceConfig` directly and adds `glazed:` tags... but `engine.InferenceConfig` currently uses `json:` tags, not `yaml:`+`glazed:` tags. The struct would need tag changes.

### Option C: Define section in `settings/` package directly (Recommended)

Add the YAML file to `settings/flags/inference.yaml` and the ValueSection to `settings/settings-inference.go`, keeping `InferenceConfig` in `engine` but adding a thin settings wrapper in the `settings` package (which already imports `engine`).

**Pro:** No new sub-package. The `settings` package already imports `engine` (verified: `settings-chat.go` imports `engine` for `StructuredOutputConfig`). Follows the same pattern as `ChatSettings` which lives directly in `settings/`.
**Con:** Requires either a wrapper struct with glazed tags that maps to/from `engine.InferenceConfig`, or adding `yaml:`+`glazed:` tags directly to `engine.InferenceConfig`.

### Recommended: Option C, variant — Add tags directly to `engine.InferenceConfig`

The simplest approach: add `yaml:` and `glazed:` tags directly to `engine.InferenceConfig`, `ClaudeInferenceConfig`, and `OpenAIInferenceConfig`. Then create the section in `settings/`.

This works because:
- `glazed` has no special import requirement for the struct — tags are just struct field annotations
- The `settings` package already imports `engine`
- `DecodeSectionInto()` uses the `glazed:` tag to match YAML flag names to struct fields
- No wrapper struct needed, no mapping code

## Implementation Plan

### Step 1: Add `yaml:` and `glazed:` tags to InferenceConfig types

**File:** `pkg/inference/engine/inference_config.go`

```go
type InferenceConfig struct {
    ThinkingBudget    *int     `json:"thinking_budget,omitempty"    yaml:"thinking_budget,omitempty"    glazed:"inference-thinking-budget"`
    ReasoningEffort   *string  `json:"reasoning_effort,omitempty"   yaml:"reasoning_effort,omitempty"   glazed:"inference-reasoning-effort"`
    ReasoningSummary  *string  `json:"reasoning_summary,omitempty"  yaml:"reasoning_summary,omitempty"  glazed:"inference-reasoning-summary"`
    Temperature       *float64 `json:"temperature,omitempty"        yaml:"temperature,omitempty"        glazed:"inference-temperature"`
    TopP              *float64 `json:"top_p,omitempty"              yaml:"top_p,omitempty"              glazed:"inference-top-p"`
    MaxResponseTokens *int     `json:"max_response_tokens,omitempty" yaml:"max_response_tokens,omitempty" glazed:"inference-max-response-tokens"`
    Stop              []string `json:"stop,omitempty"               yaml:"stop,omitempty"               glazed:"inference-stop"`
    Seed              *int     `json:"seed,omitempty"               yaml:"seed,omitempty"               glazed:"inference-seed"`
}
```

Note: the `inference-` prefix avoids collision with existing `ai-temperature`, `ai-top-p` etc. in the `ai-chat` section. This is intentional — the InferenceConfig overrides are a **separate knob** from the base chat settings, and having distinct flag names makes the override semantics explicit.

### Step 2: Create YAML flag definition

**New file:** `pkg/steps/ai/settings/flags/inference.yaml`

```yaml
slug: ai-inference
name: Inference override settings
description: >
  Per-request inference parameter overrides. These take precedence over
  base chat settings and map to provider-specific API parameters
  (thinking budget, reasoning effort, temperature, etc.).
  All fields default to nil (unset) — only explicitly provided values
  take effect as overrides.
flags:
  - name: inference-thinking-budget
    type: int
    help: >
      Token budget for model thinking/reasoning. Maps to Claude
      thinking.budget_tokens and OpenAI Responses reasoning.max_tokens.
  - name: inference-reasoning-effort
    type: choice
    choices: ["low", "medium", "high"]
    help: >
      Reasoning effort level. Maps to OpenAI Responses reasoning.effort.
  - name: inference-reasoning-summary
    type: choice
    choices: ["auto", "concise", "detailed"]
    help: >
      Reasoning summary mode. Maps to OpenAI Responses reasoning.summary.
  - name: inference-temperature
    type: float
    help: Override temperature for this inference call
  - name: inference-top-p
    type: float
    help: Override top-p sampling for this inference call
  - name: inference-max-response-tokens
    type: int
    help: Override max output tokens for this inference call
  - name: inference-stop
    type: stringList
    help: Override stop sequences for this inference call
  - name: inference-seed
    type: int
    help: Seed for reproducibility (OpenAI Chat Completions)
```

Note: **no `default:` keys** — this is what keeps pointer fields nil in the target struct.

**Defaults and nil semantics:** Since `InferenceConfig` uses pointer fields where `nil` means "not set / don't override", we need the YAML flag definitions to preserve this.

### Experimentally Verified: Omit `default:` to Keep Pointers Nil

An experiment (`scripts/test_glazed_pointer_nil/`) confirmed that glazed's `DecodeSectionInto` and `InitializeStructFromFieldDefaults` **leave pointer fields nil when the YAML definition has no `default:` key**:

| YAML definition | Struct field | Result after decode |
|----------------|-------------|-------------------|
| `default: 42` | `*int` | non-nil → 42 |
| no `default:` key | `*int` | **nil** |
| `default: 0` | `*int` | non-nil → 0 |
| no `default:` key | `*float64` | **nil** |
| no `default:` key | `*string` | **nil** |
| `default: ""` | `*string` | non-nil → "" |

**The mechanism:** `FieldValuesFromDefaults()` skips definitions where `Default == nil`. If a field isn't in the FieldValues map, `DecodeInto()` skips it entirely, leaving the pointer untouched (nil).

**Conclusion: No wrapper struct needed.** We can use `engine.InferenceConfig` directly with `glazed:` tags and omit `default:` from the YAML. Pointer fields stay nil unless the user explicitly provides a value via CLI flag, config file, env var, or profile. This is exactly the "nil = don't override" semantics we need.

### Recommended: Direct tagging (no wrapper)

### Step 3: Create ValueSection and settings file

**New file:** `pkg/steps/ai/settings/settings-inference.go`

Since glazed preserves nil pointers when no default is provided, we can decode directly
into `engine.InferenceConfig` — no wrapper struct needed.

```go
package settings

import (
    _ "embed"

    "github.com/go-go-golems/glazed/pkg/cmds/schema"
)

//go:embed "flags/inference.yaml"
var inferenceSettingsYAML []byte

const AiInferenceSlug = "ai-inference"

type InferenceValueSection struct {
    *schema.SectionImpl `yaml:",inline"`
}

func NewInferenceValueSection(options ...schema.SectionOption) (*InferenceValueSection, error) {
    ret, err := schema.NewSectionFromYAML(inferenceSettingsYAML, options...)
    if err != nil {
        return nil, err
    }
    return &InferenceValueSection{SectionImpl: ret}, nil
}
```

### Step 4: Wire into CreateGeppettoSections

**File:** `pkg/sections/sections.go`

Add after the embeddings section creation:

```go
inferenceSection, err := settings.NewInferenceValueSection()
if err != nil {
    return nil, err
}
// Note: no InitializeDefaultsFromStruct here because
// InferenceSettings is a value-type wrapper, not the same
// as engine.InferenceConfig. Defaults come from the YAML.

result := []schema.Section{
    chatSection,
    clientSection,
    claudeSection,
    geminiSection,
    openaiSection,
    embeddingsSection,
    inferenceSection,  // NEW
}
```

### Step 5: Wire into env var whitelist

**File:** `pkg/sections/sections.go` — `GetCobraCommandGeppettoMiddlewares()`

Add `settings.AiInferenceSlug` to the whitelist:

```go
sources.WrapWithWhitelistedSections(
    []string{
        settings.AiChatSlug,
        settings.AiClientSlug,
        openai.OpenAiChatSlug,
        claude.ClaudeChatSlug,
        gemini.GeminiChatSlug,
        embeddingsconfig.EmbeddingsSlug,
        settings.AiInferenceSlug,      // NEW
        cli.ProfileSettingsSlug,
    },
    sources.FromEnv("PINOCCHIO", ...),
)
```

### Step 6: Wire into UpdateFromParsedValues

**File:** `pkg/steps/ai/settings/settings-step.go`

Add after the existing decode calls. Since we omit defaults from the YAML,
`DecodeSectionInto` only sets pointer fields the user explicitly provided —
all others stay nil. We can decode directly into `InferenceConfig`.

```go
func (ss *StepSettings) UpdateFromParsedValues(parsedValues *values.Values) error {
    // ... existing decodes ...

    // Decode inference overrides directly into InferenceConfig.
    // Fields without defaults in the YAML stay nil (= don't override).
    if ss.Inference == nil {
        ss.Inference = &engine.InferenceConfig{}
    }
    err = parsedValues.DecodeSectionInto(AiInferenceSlug, ss.Inference)
    if err != nil {
        return err
    }

    return nil
}
```

### Step 7: Update StepSettings.Inference `glazed:` tag

**File:** `pkg/steps/ai/settings/settings-step.go`

```go
Inference *engine.InferenceConfig `yaml:"inference,omitempty" glazed:"ai-inference"`
```

Adding the `glazed:` tag tells the section system which section slug corresponds to this field.

## Config File Format

After implementation, users can configure inference settings in their pinocchio config:

```yaml
# ~/.config/pinocchio/config.yaml
ai-chat:
  ai-engine: claude-sonnet-4-5
  ai-api-type: claude

ai-inference:
  inference-thinking-budget: 8192
  inference-reasoning-effort: high
```

And in profiles:

```yaml
# ~/.config/pinocchio/profiles.yaml
high-thinking:
  ai-chat:
    ai-engine: claude-sonnet-4-5
    ai-api-type: claude
  ai-inference:
    inference-thinking-budget: 16384

fast:
  ai-chat:
    ai-engine: claude-haiku-4-5
    ai-api-type: claude
  ai-inference:
    inference-thinking-budget: 0
    inference-reasoning-effort: low
```

And via CLI:

```bash
pinocchio chat --inference-thinking-budget 8192 --inference-temperature 0.5
```

And via environment:

```bash
export PINOCCHIO_INFERENCE_THINKING_BUDGET=8192
pinocchio chat
```

## Overlap with Existing Settings

Some `InferenceConfig` fields overlap with existing settings:

| InferenceConfig | Existing Setting | Section |
|----------------|-----------------|---------|
| Temperature | `ai-temperature` | `ai-chat` |
| TopP | `ai-top-p` | `ai-chat` |
| MaxResponseTokens | `ai-max-response-tokens` | `ai-chat` |
| Stop | `ai-stop` | `ai-chat` |
| ReasoningEffort | `openai-reasoning-effort` | `openai-chat` |
| ReasoningSummary | `openai-reasoning-summary` | `openai-chat` |
| Seed | (none) | -- |
| ThinkingBudget | (none) | -- |

**Resolution semantics:** The engine code already handles this — `ResolveInferenceConfig()` returns Turn.Data first, then `StepSettings.Inference`, then engine code falls through to `StepSettings.Chat` fields. So:

- `--ai-temperature 0.7` sets the base temperature in ChatSettings
- `--inference-temperature 0.5` sets an override in InferenceConfig that takes precedence
- Both can coexist; the inference value wins when present

This makes `--inference-*` flags useful for "I want to override for this specific run without changing my base config".

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `pkg/steps/ai/settings/flags/inference.yaml` | **CREATE** | YAML flag definitions for ai-inference section (no defaults) |
| `pkg/steps/ai/settings/settings-inference.go` | **CREATE** | ValueSection wrapper only (no wrapper struct needed) |
| `pkg/inference/engine/inference_config.go` | MODIFY | Add `yaml:` + `glazed:` tags to InferenceConfig fields |
| `pkg/steps/ai/settings/settings-step.go` | MODIFY | Add `glazed:` tag to Inference field, decode in UpdateFromParsedValues |
| `pkg/sections/sections.go` | MODIFY | Add inferenceSection to CreateGeppettoSections, add slug to env whitelist |

## Open Questions

1. **Should `inference-temperature` and `ai-temperature` be unified?** Currently they're separate flags with separate semantics. This is intentional (inference is an override layer) but could confuse users. An alternative is to drop the overlapping fields from InferenceConfig and only expose truly new fields (ThinkingBudget, ReasoningEffort, ReasoningSummary, Seed). The overlap fields would only be available via Turn.Data for programmatic per-turn control.

2. **Should we add ClaudeInferenceConfig and OpenAIInferenceConfig to the section too?** Currently these are Turn.Data-only. Adding them as flags would mean `--inference-claude-user-id`, `--inference-openai-store`, etc. This could be done as a follow-up or kept as Turn.Data-only for programmatic use.

3. **Profile interaction with inference section.** When a profile sets `ai-inference` values, these override config-file values (profiles > config). This is correct behavior but should be documented.
