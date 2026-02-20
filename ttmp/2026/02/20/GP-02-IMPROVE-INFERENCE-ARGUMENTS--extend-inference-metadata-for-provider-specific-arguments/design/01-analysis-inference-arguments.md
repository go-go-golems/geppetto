---
title: "Analysis: Extending Inference Arguments via Typed Turn.Data Keys"
type: analysis
status: active
topics:
  - inference
  - metadata
  - architecture
  - typed-keys
related_files:
  - geppetto/pkg/turns/key_families.go
  - geppetto/pkg/turns/keys.go
  - geppetto/pkg/inference/engine/turnkeys.go
  - geppetto/pkg/inference/engine/types.go
  - geppetto/pkg/inference/engine/structured_output.go
  - geppetto/pkg/steps/ai/settings/settings-step.go
  - geppetto/pkg/steps/ai/claude/helpers.go
  - geppetto/pkg/steps/ai/claude/api/messages.go
  - geppetto/pkg/steps/ai/openai/helpers.go
  - geppetto/pkg/steps/ai/openai_responses/helpers.go
  - geppetto/pkg/steps/ai/gemini/engine_gemini.go
---

# Analysis: Extending Inference Arguments via Typed Turn.Data Keys

## 1. Problem Statement

Geppetto's inference pipeline currently configures LLM parameters once at engine construction time via `StepSettings`. There is **no mechanism to pass per-turn provider-specific arguments** dynamically through the inference call chain.

Modern LLM APIs expose rich parameter surfaces that callers need per-request control over:

- **Claude**: Extended thinking (`thinking.type`, `thinking.budget_tokens`)
- **OpenAI Responses**: Reasoning effort, reasoning token budget, store, service tier, prompt caching
- **OpenAI Chat Completions**: Seed, service tier
- **Gemini**: Safety settings, function calling mode

The gap is especially acute for **thinking/reasoning budget** — a key capability for advanced models — which has no path from caller to API request today.

## 2. Current Architecture

### 2.1 Inference Call Chain

```
Session.StartInference()
  → Builder.Build() → InferenceRunner
    → runner.RunInference(ctx, turn)
      → toolloop.Loop.RunLoop(ctx, turn)
        → eng.RunInference(ctx, turn) [provider-specific]
```

### 2.2 Settings Flow

Each provider engine receives `*settings.StepSettings` at construction time:

```go
type StepSettings struct {
    API        *APISettings
    Chat       *ChatSettings        // engine, temperature, topP, maxTokens, stop, stream
    OpenAI     *openai.Settings     // N, penalties, reasoning effort/summary
    Claude     *claude.Settings     // TopK, UserID
    Gemini     *gemini.Settings     // (empty)
    Client     *ClientSettings
    Embeddings *EmbeddingsConfig
}
```

Engines read from `e.settings` when building API requests. These settings are **immutable for the engine's lifetime** — there is no per-turn override mechanism.

### 2.3 Existing Typed Turn.Data Keys Pattern

The codebase already has a proven pattern for per-turn configuration using **typed Turn.Data keys**:

**Key construction** (`turns/key_families.go`):
```go
type DataKey[T any] struct { id TurnDataKey }

func DataK[T any](namespace, value string, version uint16) DataKey[T]
```

**Key format**: `namespace.value@vN` (e.g., `geppetto.tool_config@v1`)

**Typed access**:
- `key.Get(turn.Data) → (T, found bool, err error)`
- `key.Set(&turn.Data, value T) → error` (validates JSON serializability)

**Existing keys** (`inference/engine/turnkeys.go`):
```go
var KeyToolConfig            = turns.DataK[ToolConfig](turns.GeppettoNamespaceKey, turns.ToolConfigValueKey, 1)
var KeyStructuredOutputConfig = turns.DataK[StructuredOutputConfig](turns.GeppettoNamespaceKey, turns.StructuredOutputConfigValueKey, 1)
```

**Usage pattern** — toolloop sets config on turn, engine reads it:
```go
// toolloop/loop.go — sets before inference
engine.KeyToolConfig.Set(&t.Data, engineToolConfig(maxIterations, l.toolCfg))

// engine_openai.go — reads during request building
if toolCfg, ok, err := engine.KeyToolConfig.Get(t.Data); err == nil && ok { ... }
```

### 2.4 Identified Gaps

| Area | Current State | Gap |
|------|--------------|-----|
| `KeyStructuredOutputConfig` | Declared but **never read by any engine** | Engines read from `chatSettings.StructuredOutputConfig()` directly |
| Claude `Metadata` field | Exists in `MessageRequest` but **always set to nil** | No path for `user_id` metadata |
| Claude `Thinking` | **Not in request struct** | Cannot enable extended thinking |
| OpenAI `reasoning.max_tokens` | **Not in request struct** | Cannot set thinking token budget |
| Per-turn temperature/topP/maxTokens | Only via StepSettings (engine-lifetime) | Cannot adjust per turn |

## 3. Provider API Parameter Catalog

### 3.1 OpenAI Responses API (Complete)

From the [Responses API reference](https://developers.openai.com/api/reference/resources/responses/methods/create):

| Parameter | Type | Currently Supported | Notes |
|-----------|------|:---:|-------|
| `model` | string | Yes | Via `ChatSettings.Engine` |
| `input` | string/array | Yes | Built from Turn blocks |
| `instructions` | string | No | System message via middleware instead |
| `temperature` | number | Yes | Via `ChatSettings.Temperature` |
| `top_p` | number | Yes | Via `ChatSettings.TopP` |
| `max_output_tokens` | number | Yes | Via `ChatSettings.MaxResponseTokens` |
| `reasoning` | object | Partial | `.effort` and `.summary` via `openai.Settings`; **`.max_tokens` NOT supported** |
| `tools` | array | Yes | Via tool registry |
| `tool_choice` | string/object | Partial | Via ToolConfig |
| `parallel_tool_calls` | boolean | Yes | Via `openai.Settings.ParallelToolCalls` |
| `text` | object | Yes | Via structured output settings |
| `stream` | boolean | Yes | Via `ChatSettings.Stream` |
| `include` | array | Partial | Hardcoded to `reasoning.encrypted_content` |
| `store` | boolean | **No** | |
| `metadata` | object | **No** | Up to 16 key-value pairs |
| `service_tier` | string | **No** | |
| `truncation` | string | **No** | |
| `seed` | number | **No** | (Not listed in Responses but exists in Chat Completions) |
| `background` | boolean | **No** | |
| `prompt_cache_key` | string | **No** | |
| `prompt_cache_retention` | string | **No** | |
| `safety_identifier` | string | **No** | |
| `top_logprobs` | number | **No** | |
| `context_management` | array | **No** | |
| `conversation` | string/object | **No** | Managed by geppetto's turn system |

### 3.2 Claude Messages API

| Parameter | Type | Currently Supported | Notes |
|-----------|------|:---:|-------|
| `model` | string | Yes | Via `ChatSettings.Engine` |
| `messages` | array | Yes | Built from Turn blocks |
| `max_tokens` | int | Yes | Via `ChatSettings.MaxResponseTokens` |
| `temperature` | float | Yes | Via `ChatSettings.Temperature` |
| `top_p` | float | Yes | Via `ChatSettings.TopP` |
| `top_k` | int | Partial | In `claude.Settings` but not wired to request |
| `stop_sequences` | array | Yes | Via `ChatSettings.Stop` |
| `system` | string | Yes | Via system blocks |
| `tools` | array | Yes | Via tool registry |
| `stream` | bool | Yes | Via `ChatSettings.Stream` |
| `metadata` | object | **No** | Field exists in struct but always `nil` |
| `thinking` | object | **No** | `{type: "enabled", budget_tokens: N}` |
| `output_format` | object | Yes | Via structured output |

### 3.3 OpenAI Chat Completions API

| Parameter | Type | Currently Supported | Notes |
|-----------|------|:---:|-------|
| `model` | string | Yes | |
| `messages` | array | Yes | |
| `temperature` | float | Yes | |
| `top_p` | float | Yes | |
| `max_tokens` | int | Yes | |
| `n` | int | Yes | Via `openai.Settings.N` |
| `stop` | array | Yes | |
| `presence_penalty` | float | Yes | Via `openai.Settings` |
| `frequency_penalty` | float | Yes | Via `openai.Settings` |
| `stream` | bool | Yes | |
| `seed` | int | **No** | |
| `tools` | array | Yes | |
| `tool_choice` | string/object | Yes | |
| `logit_bias` | map | Partial | Has known type bug |

## 4. Proposed Design: Layered InferenceConfig

### 4.1 Architecture

Three layers of configuration:

1. **`StepSettings.Inference`** — Engine-level defaults set at construction time
2. **Turn.Data `InferenceConfig`** — Per-turn overrides (cross-provider abstractions)
3. **Turn.Data provider-specific configs** (`ClaudeInferenceConfig`, `OpenAIInferenceConfig`) — Per-turn provider-specific fields

```
Engine-level default (StepSettings):
  ┌─────────────────────────────────────┐
  │  StepSettings.Inference             │ ← Set at engine creation time
  │  *engine.InferenceConfig            │   (e.g., ThinkingBudget: 8192)
  └─────────────────────────────────────┘

Per-turn overrides (Turn.Data):
  ┌─────────────────────────────────────┐
  │  InferenceConfig (cross-provider)   │ ← ThinkingBudget, ReasoningEffort,
  │  geppetto.inference_config@v1       │   Temperature, TopP, MaxResponseTokens, Seed
  └─────────────────────────────────────┘
  ┌─────────────────────────────────────┐
  │  ClaudeInferenceConfig (optional)   │ ← UserID, TopK
  │  geppetto.claude_inference_config@v1│
  └─────────────────────────────────────┘
  ┌─────────────────────────────────────┐
  │  OpenAIInferenceConfig (optional)   │ ← N, Penalties, Store, ServiceTier
  │  geppetto.openai_inference_config@v1│
  └─────────────────────────────────────┘

Engine reads at request-build time:
  ┌──────────────────────────────────────────────┐
  │  1. Check Turn.Data for InferenceConfig      │
  │  2. Fall back to StepSettings.Inference      │
  │  3. Fall back to StepSettings.Chat fields    │
  │  4. Check Turn.Data for provider-specific    │
  └──────────────────────────────────────────────┘
```

### 4.2 Cross-Provider Field Mapping

| InferenceConfig Field | Claude | OpenAI Responses | OpenAI Chat | Gemini |
|-----------------------|--------|-----------------|-------------|--------|
| `ThinkingBudget` | `thinking.budget_tokens` | `reasoning.max_tokens` | ignored | ignored |
| `ReasoningEffort` | ignored | `reasoning.effort` | ignored | ignored |
| `ReasoningSummary` | ignored | `reasoning.summary` | ignored | ignored |
| `Temperature` | `temperature` | `temperature` | `temperature` | `temperature` |
| `TopP` | `top_p` | `top_p` | `top_p` | `top_p` |
| `MaxResponseTokens` | `max_tokens` | `max_output_tokens` | `max_tokens` | `max_output_tokens` |
| `Stop` | `stop_sequences` | `stop_sequences` | `stop` | ignored |
| `Seed` | ignored | ignored | `seed` | ignored |

### 4.3 Override Precedence

```
Turn.Data InferenceConfig  >  StepSettings.Inference  >  StepSettings.Chat fields  >  defaults
```

Resolution is handled by `engine.ResolveInferenceConfig(turn, settings.Inference)`:
- `ResolveInferenceConfig` reads Turn.Data and merges field-by-field over `StepSettings.Inference`
- Nil fields in Turn.Data preserve engine defaults
- `Stop` has explicit three-state semantics:
  - `Stop == nil` preserves inherited stop sequences
  - `Stop == []string{}` explicitly clears inherited stop sequences
  - `Stop == []string{"..."}` explicitly replaces inherited stop sequences
- Engines then apply non-nil resolved fields and still fall back to ChatSettings when unset

### 4.4 Type Definitions

```go
// inference/engine/inference_config.go

type InferenceConfig struct {
    ThinkingBudget    *int     `json:"thinking_budget,omitempty"`
    ReasoningEffort   *string  `json:"reasoning_effort,omitempty"`
    ReasoningSummary  *string  `json:"reasoning_summary,omitempty"`
    Temperature       *float64 `json:"temperature,omitempty"`
    TopP              *float64 `json:"top_p,omitempty"`
    MaxResponseTokens *int     `json:"max_response_tokens,omitempty"`
    Stop              []string `json:"stop,omitempty"`
    Seed              *int     `json:"seed,omitempty"`
}

type ClaudeInferenceConfig struct {
    UserID *string `json:"user_id,omitempty"`
    TopK   *int    `json:"top_k,omitempty"`
}

type OpenAIInferenceConfig struct {
    N                *int     `json:"n,omitempty"`
    PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
    FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
    Store            *bool    `json:"store,omitempty"`
    ServiceTier      *string  `json:"service_tier,omitempty"`
}
```

### 4.5 Key Declarations

```go
// inference/engine/turnkeys.go
var KeyInferenceConfig       = turns.DataK[InferenceConfig](turns.GeppettoNamespaceKey, "inference_config", 1)
var KeyClaudeInferenceConfig = turns.DataK[ClaudeInferenceConfig](turns.GeppettoNamespaceKey, "claude_inference_config", 1)
var KeyOpenAIInferenceConfig = turns.DataK[OpenAIInferenceConfig](turns.GeppettoNamespaceKey, "openai_inference_config", 1)
```

### 4.6 Usage Examples

**Engine-level default** (set once at construction time):
```go
settings.Inference = &engine.InferenceConfig{
    ThinkingBudget:  intPtr(8192),
    ReasoningEffort: strPtr("high"),
}
eng, _ := claude.NewClaudeEngine(settings)
// All inference calls through this engine will use thinking budget 8192
```

**Per-turn override** (dynamic, per inference call):
```go
// Override thinking budget for this specific turn
cfg := engine.InferenceConfig{
    ThinkingBudget: intPtr(16384),
}
engine.KeyInferenceConfig.Set(&turn.Data, cfg)
// This turn uses 16384; other turns still use engine default 8192
```

**Resolution in engine** (handled automatically):
```go
// engine.ResolveInferenceConfig performs field-level merge:
// Turn.Data overrides non-nil fields, preserving engine defaults for nil fields.
infCfg := engine.ResolveInferenceConfig(t, e.settings.Inference)
if infCfg != nil && infCfg.ThinkingBudget != nil {
    req.Thinking = &api.ThinkingParam{
        Type: "enabled", BudgetTokens: *infCfg.ThinkingBudget,
    }
}
```

## 5. Implementation Summary

### Files to Create
- `geppetto/pkg/inference/engine/inference_config.go` — Type definitions

### Files to Modify
- `geppetto/pkg/turns/keys.go` — Value key constants
- `geppetto/pkg/inference/engine/turnkeys.go` — Typed key declarations
- `geppetto/pkg/steps/ai/claude/api/messages.go` — Add Thinking field to MessageRequest
- `geppetto/pkg/steps/ai/claude/api/completion.go` — Expand Metadata struct
- `geppetto/pkg/steps/ai/claude/helpers.go` — Read Turn.Data in MakeMessageRequestFromTurn
- `geppetto/pkg/steps/ai/openai_responses/helpers.go` — Read Turn.Data in buildResponsesRequest
- `geppetto/pkg/steps/ai/openai/helpers.go` — Read Turn.Data in MakeCompletionRequestFromTurn
- `geppetto/pkg/steps/ai/gemini/engine_gemini.go` — Read Turn.Data in RunInference

### Bonus: Complete existing KeyStructuredOutputConfig wiring
All four engines should also read `engine.KeyStructuredOutputConfig` from Turn.Data, completing the already-declared but unwired key.
