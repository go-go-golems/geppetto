---
Title: Model Metadata Design and Implementation Guide
Ticket: GP-70
Status: active
Topics:
    - geppetto
    - engine-profiles
    - inference
    - model-metadata
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/engineprofiles/types.go:Core EngineProfile and EngineProfileRegistry types
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/engineprofiles/extensions.go:Profile extension key system for typed extensions
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/steps/ai/settings/settings-inference.go:InferenceSettings top-level struct
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/steps/ai/settings/settings-chat.go:ChatSettings with Engine and ApiType
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/inference/engine/inference_config.go:InferenceConfig for per-turn overrides
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/inference/engine/factory/factory.go:StandardEngineFactory dispatching by ApiType
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/engineprofiles/stack_merge.go:Stack merge for InferenceSettings and Extensions
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/engineprofiles/source_chain.go:ChainedRegistry loading YAML/SQLite sources
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/turns/inference_result.go:InferenceResult and InferenceUsage
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/pinocchio/pkg/inference/runtime/profile_runtime.go:ProfileRuntime extension example
ExternalSources: []
Summary: "Design and implementation guide for adding structured model metadata to engine profiles, covering model capabilities, costs, context windows, and a metadata grab-bag."
LastUpdated: 2026-05-06T10:50:00.000000000-04:00
WhatFor: "Onboarding reference for an intern implementing model metadata in the geppetto/pinocchio profile system"
WhenToUse: "When implementing or reviewing the model metadata feature"
---

# Model Metadata Design and Implementation Guide

## Executive Summary

This document describes how to add structured **model metadata** to the geppetto engine profile system. Model metadata captures static, model-level properties—such as whether a model supports reasoning/thinking, what input modalities it accepts (text, image, audio, video), its context window size (both the absolute token limit and the quality high-water-mark beyond which quality degrades), its maximum output tokens, and per-token cost rates (input, output, cache read, cache write). These fields are both typed Go structs for compile-time safety and a `map[string]any` metadata grab-bag for vendor-specific or forward-compatible data.

The feature integrates into the existing `EngineProfile` system by adding a new `ModelInfo` struct inside `InferenceSettings`, mirroring the existing pattern of `ChatSettings`, `OpenAI.Settings`, etc. It also introduces a profile extension key (`geppetto.model_info@v1`) so that model metadata can be layered and merged through the profile stack just like `ProfileRuntime` is today.

The primary consumers of this data are:

1. **Context window budgeting** — upstream callers (pinocchio TUI, web-chat) can trim prompt content to fit within the model's quality ceiling rather than its hard limit.
2. **Cost tracking** — post-inference cost computation using `InferenceResult.Usage` × model cost rates.
3. **UI rendering** — showing model capabilities (reasoning badge, input type icons, context window gauge) in the profile switcher and model picker.
4. **API schema discovery** — the JS module surface can expose model metadata so prompt scripts can make model-aware decisions.

---

## Problem Statement

Today, geppetto's engine profiles know *which* model to call (`ChatSettings.Engine`) and *how* to call it (`ApiType`, API keys, base URLs, temperature, etc.), but they know nothing *about* the model itself. The system cannot answer basic questions like:

- *Does this model support extended thinking/reasoning?*
- *Can this model accept images? Audio? Video?*
- *What is the context window size, and at what token count does quality degrade?*
- *How much does a single input token cost? Output token? Cache read? Cache write?*
- *What is the maximum number of output tokens?*

Without this metadata, every consumer must either hard-code model-specific heuristics (see `factory.isReasoningModel()` which pattern-matches model name prefixes) or simply ignore the information. This leads to:

- **Duplicated heuristics** scattered across the codebase (e.g., `isReasoningModel()` in factory.go, `inferAPIType()` in api_engines.go).
- **No cost accounting** — usage tokens are tracked in `InferenceResult.Usage` but there is no cost model to convert tokens → dollars.
- **No context window awareness** — the system cannot warn when a prompt exceeds the model's quality ceiling, leading to silently degraded output.
- **No capability discovery** — UIs and scripts cannot render model capabilities without out-of-band knowledge.

The goal is to make model metadata a first-class part of the profile system, loaded from YAML alongside existing `inference_settings`, typed for compile-time safety, and extensible via a grab-bag for vendor-specific properties.

---

## Current-State Architecture

Before we design the solution, let's walk through every component you need to understand. If you are new to this codebase, read each subsection carefully—the design depends on understanding how these pieces fit together.

### The Three Repos

The workspace at `/home/manuel/workspaces/2026-05-06/add-model-metadata/` contains three Go modules in a Go workspace (`go.work`):

| Repo | Path | Purpose |
|------|------|---------|
| **geppetto** | `./geppetto` | Core AI inference library: settings, engine factory, provider implementations, turns, events |
| **pinocchio** | `./pinocchio` | Application layer: TUI, web-chat, profile management, JS scripting runtime |
| **glazed** | `./glazed` | Generic CLI framework: Cobra integration, structured output, help system |

**Model metadata lives in geppetto** because it is a property of the inference model, not the application. Pinocchio consumes it.

### Engine Profiles (geppetto/pkg/engineprofiles/)

The engine profile system is the *central configuration mechanism* for AI inference in this codebase. Let's walk through it in detail.

#### Types (types.go)

The core types are:

```
EngineProfileRegistry         // A named collection of profiles (e.g., "provider-openai")
  Slug: RegistrySlug          // e.g., "provider-openai"
  DisplayName: string         // e.g., "Provider defaults"
  Description: string
  DefaultEngineProfileSlug    // Which profile to use by default
  Profiles: map[EngineProfileSlug]*EngineProfile
  Metadata: RegistryMetadata

EngineProfile                 // A named engine preset
  Slug: EngineProfileSlug     // e.g., "default", "assistant", "analyst"
  DisplayName: string
  Description: string
  Stack: []EngineProfileRef   // Layer stack for composition
  InferenceSettings: *InferenceSettings  // The actual engine config
  Metadata: EngineProfileMetadata         // Provenance (source, version, timestamps)
  Extensions: map[string]any              // Typed extension grab-bag
```

Key points:

- `InferenceSettings` is the same struct used throughout the inference pipeline—profile resolution produces a ready-to-use `InferenceSettings`.
- `Extensions` is a `map[string]any` keyed by *extension keys* like `"pinocchio.webchat_runtime@v1"`. This is how app-specific data (e.g., Pinocchio's middleware list) is stored alongside engine settings without polluting the core types.
- `Stack` enables layering: a profile can reference another profile as a base, and the system merges them.

#### Stack Merge (stack_merge.go)

When a profile has a `Stack`, the system walks the stack from base to leaf, merging `InferenceSettings` at each layer:

```
MergeEngineProfileStackLayers(layers []EngineProfileStackLayer) → StackMergeResult
  StackMergeResult.InferenceSettings = merged settings
  StackMergeResult.Extensions = merged extensions
```

Extensions merge recursively (maps merge key-by-key; scalars overwrite). This is how `ProfileRuntime` data flows through the stack.

#### Profile Extension Keys (extensions.go)

Extensions use a **typed key system** to avoid string-typed chaos:

```go
type ProfileExtensionKey[T any] struct { id ExtensionKey }

// Key format: "namespace.feature@vN" (e.g., "pinocchio.webchat_runtime@v1")
```

Operations:

- `Get(profile *EngineProfile) → (T, bool, error)` — decode from `profile.Extensions`
- `Set(profile *EngineProfile, value T) error` — encode into `profile.Extensions`
- `Delete(profile *EngineProfile)` — remove from `profile.Extensions`

The type parameter `T` gives you compile-time type safety. The runtime representation is always `map[string]any` inside `profile.Extensions`, but the codec round-trips through JSON marshal/unmarshal to get typed Go structs.

**Example from pinocchio** (`pinocchio/pkg/inference/runtime/profile_runtime.go`):

```go
var WebChatProfileRuntimeExtension = gepprofiles.MustProfileExtensionKey[ProfileRuntime](
    "pinocchio", "webchat_runtime", 1,
)
```

This is the pattern we will follow for model metadata.

#### Registry, Store, and Chained Loading

The registry interface (`Registry`) provides read access to profiles:

```go
type Registry interface {
    ListRegistries(ctx) → []RegistrySummary
    GetRegistry(ctx, slug) → *EngineProfileRegistry
    ListEngineProfiles(ctx, slug) → []*EngineProfile
    GetEngineProfile(ctx, registrySlug, profileSlug) → *EngineProfile
    ResolveEngineProfile(ctx, ResolveInput) → *ResolvedEngineProfile
}
```

`ResolveEngineProfile` is the important one—it walks the stack, merges settings, and returns a `ResolvedEngineProfile`:

```go
type ResolvedEngineProfile struct {
    RegistrySlug      RegistrySlug
    EngineProfileSlug EngineProfileSlug
    InferenceSettings *InferenceSettings
    StackLineage      []ResolvedProfileStackEntry
    Metadata          map[string]any
}
```

The `ChainedRegistry` loads profiles from multiple sources (YAML files, SQLite databases) and chains them together. Source specs look like:

- `yaml:/path/to/profiles.yaml`
- `sqlite:/path/to/profiles.db`
- Plain path (auto-detected as YAML or SQLite)

### InferenceSettings (geppetto/pkg/steps/ai/settings/)

`InferenceSettings` is the top-level configuration struct that an engine consumes:

```go
type InferenceSettings struct {
    API        *APISettings           // API keys and base URLs
    Chat       *ChatSettings          // Model name, API type, temperature, etc.
    OpenAI     *openai.Settings       // OpenAI-specific overrides
    Client     *ClientSettings        // Timeout, proxy, organization
    Claude     *claude.Settings       // Claude-specific overrides
    Gemini     *gemini.Settings       // Gemini-specific overrides
    Ollama     *ollama.Settings       // Ollama-specific overrides
    Embeddings *config.EmbeddingsConfig
    Inference  *engine.InferenceConfig // Per-turn inference overrides (thinking budget, etc.)
}
```

`ChatSettings` is where the model identity lives today:

```go
type ChatSettings struct {
    Engine            *string          // Model name (e.g., "gpt-4o-mini")
    ApiType           *types.ApiType   // Provider (e.g., "openai", "claude")
    MaxResponseTokens *int
    Temperature       *float64
    TopP              *float64
    Stop              []string
    // ... caching, structured output, etc.
}
```

Key observation: **all scalar fields use pointer types** (`*string`, `*int`, `*float64`). A nil pointer means "not set, use default" or "don't override during merge". This is critical for the stack merge semantics.

### InferenceConfig (geppetto/pkg/inference/engine/)

`InferenceConfig` provides per-turn overrides for inference parameters:

```go
type InferenceConfig struct {
    ThinkingBudget    *int     // e.g., 8192 for Claude thinking budget
    ReasoningEffort   *string  // e.g., "low", "medium", "high" for OpenAI
    ThinkingType      *string  // e.g., "enabled" for DeepSeek
    ReasoningSummary  *string  // e.g., "auto", "concise", "detailed" for OpenAI
    Temperature       *float64
    TopP              *float64
    MaxResponseTokens *int
    Stop              []string
    Seed              *int
}
```

This is used both as a default (on `InferenceSettings.Inference`) and as a per-turn override (on `Turn.Data`). The resolution order is: Turn.Data → InferenceSettings.Inference → InferenceSettings.Chat fields.

### Engine Factory (geppetto/pkg/inference/engine/factory/)

`StandardEngineFactory.CreateEngine(settings)` dispatches by `settings.Chat.ApiType`:

```
"openai"           → openai.NewOpenAIEngine(settings)
"open-responses"   → openai_responses.NewEngine(settings)
"claude"           → claude.NewClaudeEngine(settings)
"gemini"           → gemini.NewGeminiEngine(settings)
```

Today, it has a hardcoded `isReasoningModel()` that pattern-matches model name prefixes (`o1`, `o3`, `o4`, `gpt-5`). This is the kind of heuristic that model metadata should replace.

### Profile YAML Format

Profiles are YAML files on disk. A typical profile file looks like:

```yaml
slug: provider-openai
display_name: Provider defaults
description: Provider baseline engine settings and credentials.
profiles:
  default:
    slug: default
    display_name: Provider default
    inference_settings:
      api:
        api_keys:
          openai-api-key: demo-openai-key
      chat:
        api_type: openai
        engine: gpt-4o-mini
```

Team and user layers stack on top:

```yaml
slug: team-agent
display_name: Team profiles
profiles:
  assistant:
    slug: assistant
    stack:
      - registry_slug: provider-openai
        profile_slug: default
    inference_settings:
      chat:
        engine: gpt-5-mini
```

Extensions appear alongside `inference_settings`:

```yaml
    inference_settings:
      chat:
        engine: gpt-5-mini
    extensions:
      pinocchio.webchat_runtime@v1:
        system_prompt: "You are a helpful assistant."
        middlewares:
          - name: agent-mode
            enabled: true
```

### JS Module Surface (geppetto/pkg/js/modules/geppetto/)

The geppetto JS module exposes engine and profile objects to JavaScript:

- `geppetto.engines.fromConfig(opts)` — creates an engine from options
- `geppetto.engines.fromProfile(opts)` — resolves a profile and creates an engine
- `geppetto.engines.fromResolvedProfile(resolved)` — creates an engine from a resolved profile

The JS module is how Pinocchio scripts create and use engines. Model metadata must be accessible through this surface.

### InferenceResult (geppetto/pkg/turns/)

After inference, the system stores an `InferenceResult` on the turn:

```go
type InferenceResult struct {
    Provider   string
    Model      string
    StopReason string
    FinishClass InferenceFinishClass
    Truncated  bool
    Usage      *InferenceUsage   // input_tokens, output_tokens, cached_tokens, etc.
    MaxTokens  *int
    DurationMs *int64
    RequestID  string
    ResponseID string
    Extra      map[string]any
}

type InferenceUsage struct {
    InputTokens              int
    OutputTokens             int
    CachedTokens             int
    CacheCreationInputTokens int
    CacheReadInputTokens     int
}
```

This is where cost computation will connect: `Usage × ModelInfo.Cost = total cost`.

---

## Gap Analysis

The current system has these specific gaps relative to the desired outcome:

| # | Gap | Evidence | Impact |
|---|-----|----------|--------|
| 1 | **No model metadata struct** | `InferenceSettings` has no field for model capabilities, costs, or context windows | Every consumer must hard-code or guess model properties |
| 2 | **No context window awareness** | No `contextWindow` or `qualityHighWatermark` fields anywhere in the codebase | Prompts can silently exceed quality limits; no trimming logic possible |
| 3 | **No cost model** | `InferenceResult.Usage` tracks tokens but no cost rates exist | No cost accounting without out-of-band data |
| 4 | **No input modality tracking** | No `input` field (text, image, audio, video) | Cannot filter or render model capabilities |
| 5 | **No reasoning capability flag** | `isReasoningModel()` in factory.go hard-codes name prefixes | Breaks for new models; duplicates across callers |
| 6 | **No model metadata extension** | `EngineProfile.Extensions` has no `geppetto.model_info@v1` key | Cannot layer model info through the profile stack |

---

## Proposed Architecture

### New Types

We introduce a `ModelInfo` struct that lives in two places:

1. **Inside `InferenceSettings`** as a first-class field (for direct access from resolved settings).
2. **As a profile extension** (`geppetto.model_info@v1`) for stack-merge semantics.

The dual placement ensures:
- Direct access from `InferenceSettings` (no extension decoding needed for the common path).
- Stack merge through `Extensions` (layered model info merges like other extensions).

#### ModelInfo Struct

`ModelInfo` describes static, model-level properties independent of per-request inference parameters. It is loaded from profile YAML alongside `inference_settings` and can be layered through the profile stack.

**Design rules** (follow the same patterns as `ChatSettings` and `InferenceConfig`):

- All scalar fields use pointer types so that `nil` means "not set" during merge.
- Slice and map fields use value types (nil = absent, empty = explicitly empty).
- Every struct has a `Clone()` method.
- All fields carry `json` and `yaml` struct tags with `omitempty`.

**Fields**:

| Field | Type | Purpose |
|-------|------|---------|
| `ID` | `*string` | Canonical model identifier matching `ChatSettings.Engine` (e.g., "DeepSeek-V4-Pro") |
| `Name` | `*string` | Human-readable display name (e.g., "DeepSeek V4 Pro") |
| `Reasoning` | `*bool` | Whether the model supports extended thinking/reasoning. When true, `InferenceConfig.ThinkingBudget`/`ReasoningEffort` are relevant |
| `Input` | `[]InputModality` | Modalities the model accepts ("text", "image", "audio", "video", "pdf") |
| `ContextWindow` | `*int` | Absolute maximum tokens the model can process in one request |
| `QualityHighWatermark` | `*int` | Token count beyond which output quality degrades. When nil, assume quality is stable up to `ContextWindow` |
| `MaxOutputTokens` | `*int` | Maximum tokens the model can generate in one response. Maps to `ChatSettings.MaxResponseTokens` defaults |
| `Cost` | `*ModelCost` | Per-token pricing in USD per million tokens |
| `Metadata` | `map[string]any` | Untyped grab-bag for vendor-specific or forward-compatible properties |

The implementor should define the exact Go struct with appropriate struct tags, following the existing patterns in `ChatSettings` and `InferenceConfig`.

#### InputModality Type

`InputModality` is a string type representing an input modality a model can accept. Known values ("text", "image", "audio", "video", "pdf") should be defined as constants. Using a string type (not an enum) preserves forward compatibility—new modalities from providers (e.g., "3d", "code") round-trip through YAML/JSON without code changes.

#### ModelCost Struct

`ModelCost` holds per-token pricing in USD per million tokens. It has four fields: `Input`, `Output`, `CacheRead`, `CacheWrite` — all `float64`.

**Key design decision**: `ModelCost` uses value types (not pointers) for its fields. Rationale: a nil `ModelInfo.Cost` means "cost unknown/not specified" while a `ModelCost` with all zeros means "this model is free." This avoids the nil-vs-zero ambiguity that would arise if cost fields were pointer types.

The implementor should decide the exact approach—this decision (value types for Cost) is recommended but open to revisiting during implementation if a different tradeoff makes more sense.

### InferenceSettings Integration

Add a `ModelInfo *ModelInfo` field to `InferenceSettings`, with the YAML tag `model_info,omitempty`. Follow the same pattern as the existing fields (`Chat`, `Inference`, etc.)—a pointer type means "absent by default, mergeable through the stack."

The implementor should also decide on a `glazed` section tag (e.g., `glazed:"ai-model-info"`) if CLI flag exposure is desired, though cost and context window fields are unlikely to be set from CLI flags.

### Profile Extension Key

Consider registering an extension key for ModelInfo (e.g., `geppetto.model_info@v1`) if the JS module or external consumers need typed access through the extension system. However, see the design decision below—**the extension key is optional and read-only**, not the source of truth.

The implementor should decide whether an extension key is needed based on JS module requirements. If the resolved profile object can expose `modelInfo` directly from `InferenceSettings.ModelInfo`, an extension key may be unnecessary.

**Design decision**: Having ModelInfo in both `InferenceSettings` and `Extensions` creates a synchronization problem. Instead, we choose *one* canonical location. Let's look at the options:

**Option A: ModelInfo only in InferenceSettings**
- Pro: Simple. Follows the existing pattern (Chat, OpenAI, Claude are all in InferenceSettings).
- Pro: No extension key needed. Stack merge already handles InferenceSettings.
- Con: `InferenceSettings` is about *how to call* the engine, not *what the model is*. ModelInfo is semantically different.

**Option B: ModelInfo only as a profile extension**
- Pro: Clean separation between "how to call" and "what it is."
- Pro: Extension merge semantics already work.
- Con: Every consumer must decode the extension. No direct access from InferenceSettings.

**Option C: ModelInfo in InferenceSettings + extension key for backward compat**
- Pro: Direct access from InferenceSettings. Extension key for JS/external consumers.
- Con: Two copies. Must keep in sync.

**Decision: Option A** — `ModelInfo` goes directly into `InferenceSettings` as a first-class field. The stack merge already handles `InferenceSettings` field-by-field, so `ModelInfo` merges naturally when we implement `MergeModelInfo`. We also add an extension key for the JS module surface (read-only, derived from `InferenceSettings.ModelInfo` after resolution), but the extension is *not* the source of truth.

### Merge Semantics for ModelInfo

ModelInfo merges like other `InferenceSettings` fields: overlay wins for set fields, nil fields fall back to base.

**Merge rules** (pseudocode):

- If base is nil → return overlay.Clone()
- If overlay is nil → return base.Clone()
- Clone base into result
- For each pointer field: if overlay.field is set, copy it into result (replacing base)
- For `Input` slice: if overlay.Input is non-nil, replace entirely (not append)
- For `Cost`: if overlay.Cost is non-nil, replace wholesale (not field-by-field) — avoids nil-vs-zero ambiguity
- For `Metadata` map: merge recursively (maps merge key-by-key; scalars overwrite), following the pattern in `mergeExtensionValue()` in `stack_merge.go`

**Important**: `ModelCost` is replaced wholesale, not field-by-field. If you override cost, you must specify all rates. This is a deliberate design tradeoff—see the "Risks" section.

The implementor should follow the pattern of `MergeInferenceConfig` in `inference_config.go`.

### YAML Format

Model metadata appears under `inference_settings.model_info` in profile YAML:

```yaml
slug: provider-openai
display_name: Provider defaults
profiles:
  default:
    slug: default
    inference_settings:
      api:
        api_keys:
          openai-api-key: demo-openai-key
      chat:
        api_type: openai
        engine: gpt-4o-mini
      model_info:
        id: gpt-4o-mini
        name: GPT-4o Mini
        reasoning: false
        input:
          - text
          - image
        context_window: 128000
        quality_high_watermark: 128000
        max_output_tokens: 16384
        cost:
          input: 0.15
          output: 0.60
          cache_read: 0.075
          cache_write: 0.30
        metadata:
          output_modalities:
            - text
          fine_tunable: true
```

A team override can layer on top:

```yaml
slug: team-agent
profiles:
  assistant:
    slug: assistant
    stack:
      - registry_slug: provider-openai
        profile_slug: default
    inference_settings:
      chat:
        engine: gpt-5
      model_info:
        id: gpt-5
        name: GPT-5
        reasoning: true
        context_window: 1000000
        quality_high_watermark: 500000
        max_output_tokens: 32768
        cost:
          input: 2.50
          output: 10.00
          cache_read: 1.25
          cache_write: 5.00
```

After stack merge, the `assistant` profile has `model_info` from the team layer (overlay) since all fields are set.

### Quality High-Watermark Semantics

The `quality_high_watermark` field is the most novel part of this design. Here's how it works:

**Problem**: Some models advertise very large context windows (e.g., 1M tokens) but produce noticeably lower-quality output when the input exceeds a certain threshold (e.g., 500K tokens). The system should know about this threshold so it can:

1. Warn the user when their prompt exceeds the quality ceiling.
2. Trim context strategically (drop old turns, summarize, etc.) to stay within the quality window.
3. Show a gauge in the UI that distinguishes "safe" from "degraded" context usage.

**Semantics**:

- `quality_high_watermark` ≤ `context_window` always. If `quality_high_watermark > context_window`, validation fails.
- When `quality_high_watermark` is nil, assume quality is stable up to `context_window`.
- The budgeting logic should target `quality_high_watermark` as the "effective context" for prompt construction, using `context_window` as the hard ceiling for API calls.

**Pseudocode for context budgeting**:

```
func EffectiveContextLimit(info *ModelInfo) int {
    if info == nil || info.ContextWindow == nil {
        return 0  // unknown, caller must use a default
    }
    if info.QualityHighWatermark != nil && *info.QualityHighWatermark < *info.ContextWindow {
        return *info.QualityHighWatermark
    }
    return *info.ContextWindow
}

func HardContextLimit(info *ModelInfo) int {
    if info == nil || info.ContextWindow == nil {
        return 0
    }
    return *info.ContextWindow
}
```

### Cost Computation

With `ModelInfo.Cost` and `InferenceResult.Usage`, computing cost is a matter of multiplying each usage bucket by the corresponding cost rate:

**Pseudocode**:

```
ComputeCost(info, usage) → float64 or nil:
  if info is nil or info.Cost is nil or usage is nil → return nil
  total = 0
  total += Cost.Input  × usage.InputTokens / 1_000_000
  total += Cost.Output × usage.OutputTokens / 1_000_000
  total += Cost.CacheRead  × usage.CacheReadInputTokens / 1_000_000
  total += Cost.CacheWrite × usage.CacheCreationInputTokens / 1_000_000
  return total
```

All rates are in **USD per million tokens**. The implementor should decide whether this is a method on `ModelInfo`, a standalone function, or computed inline after inference.

### Replacing isReasoningModel()

The factory's `isReasoningModel()` can be replaced by checking `ModelInfo.Reasoning`. The approach:

1. When `ModelInfo` is available and `Reasoning` is set, use it directly (data-driven).
2. When `ModelInfo` is nil or `Reasoning` is nil, fall back to the existing name-prefix heuristic as a compatibility path.
3. Eventually, remove the heuristic once all profiles carry `model_info.reasoning`.

The implementor should decide the exact wiring—whether `ModelInfo` is passed to the factory as a separate argument, extracted from `InferenceSettings.ModelInfo` inside the factory, or accessed through some other mechanism.

### JS Module Surface

The JS module needs to expose `ModelInfo`:

```javascript
// From a resolved profile:
const resolved = geppetto.profiles.resolve({ profileSlug: "assistant" });
const info = resolved.modelInfo;

// Access typed fields:
console.log(info.id);                    // "gpt-5"
console.log(info.reasoning);             // true
console.log(info.input);                 // ["text", "image"]
console.log(info.contextWindow);         // 1000000
console.log(info.qualityHighWatermark);  // 500000
console.log(info.maxOutputTokens);       // 32768
console.log(info.cost.input);            // 2.50
console.log(info.cost.output);           // 10.00
console.log(info.metadata);             // { output_modalities: ["text"], ... }
```

This requires adding `modelInfo` to the `newResolvedEngineProfileObject()` function in `api_runtime_metadata.go`.

### GetMetadata Integration

`InferenceSettings.GetMetadata()` (in `settings-inference.go`) should include model metadata. When `ModelInfo` is present, flatten its fields into the metadata map using prefixed keys (e.g., `"ai-model-id"`, `"ai-model-reasoning"`, `"ai-model-context-window"`, `"ai-model-quality-high-watermark"`, `"ai-model-max-output-tokens"`, `"ai-model-cost-input"`, `"ai-model-cost-output"`, `"ai-model-cost-cache-read"`, `"ai-model-cost-cache-write"`, `"ai-model-input"`).

The implementor should follow the existing `GetMetadata()` pattern of only including non-nil/non-zero values, and should decide the exact key naming convention.

---

## Data Flow Diagrams

### Profile Resolution with Model Metadata

```
┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│ provider YAML    │    │ team YAML        │    │ user YAML        │
│ (base layer)     │    │ (overlay 1)      │    │ (overlay 2)      │
│                  │    │                  │    │                  │
│ chat.engine:     │    │ chat.engine:     │    │ (no override)    │
│   gpt-4o-mini   │    │   gpt-5          │    │                  │
│ model_info:      │    │ model_info:      │    │                  │
│   context_window │    │   context_window │    │                  │
│   : 128000       │    │   : 1000000      │    │                  │
│   cost.input:    │    │   cost.input:    │    │                  │
│     0.15         │    │     2.50         │    │                  │
└────────┬─────────┘    └────────┬─────────┘    └────────┬─────────┘
         │                       │                        │
         └───────────┬───────────┘                        │
                     │ Stack Merge                        │
                     │                                    │
                     │  MergeInferenceSettings(base, o1)  │
                     │  → merged InferenceSettings         │
                     │                                    │
                     └──────────┬─────────────────────────┘
                                │ Stack Merge
                                │
                                │  MergeInferenceSettings(merged, o2)
                                │  → final InferenceSettings
                                │
                    ┌───────────┴───────────┐
                    │ ResolvedEngineProfile │
                    │                       │
                    │ InferenceSettings:    │
                    │   chat.engine: gpt-5  │
                    │   model_info:         │
                    │     context_window:   │
                    │       1000000         │
                    │     cost.input: 2.50  │
                    └───────────┬───────────┘
                                │
                    ┌───────────┴───────────┐
                    │ CreateEngine(settings)│
                    │ + ModelInfo available  │
                    │ for budgeting, cost,   │
                    │ UI rendering          │
                    └───────────────────────┘
```

### Cost Computation Flow

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ InferenceResult │     │ ModelInfo       │     │ ComputedCost    │
│                 │     │                 │     │                 │
│ Usage:          │     │ Cost:           │     │ total =         │
│   InputTokens:  │  ×  │   Input: 2.50   │  =  │   2.50×1500/1M  │
│     1500        │     │   Output: 10.00 │     │  + 10.00×500/1M │
│   OutputTokens: │     │   CacheRead:    │     │  + 1.25×800/1M  │
│     500         │     │     1.25        │     │  + 5.00×200/1M  │
│   CacheRead:    │     │   CacheWrite:   │     │ = $0.01425      │
│     800         │     │     5.00        │     │                 │
│   CacheCreation:│     │                 │     │                 │
│     200         │     │ (per 1M tokens) │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### Context Window Budgeting Flow

```
┌──────────────────┐
│ ModelInfo        │
│                  │
│ ContextWindow:   │
│   1,000,000      │──── Hard limit (API reject above this)
│                  │
│ QualityHighWater-│
│   mark: 500,000  │──── Effective limit (budget prompts to this)
│                  │
│ MaxOutputTokens: │
│   32,768         │──── Reserve this many tokens for output
│                  │
└────────┬─────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│ Prompt Budgeter                              │
│                                              │
│ Available input tokens =                      │
│   EffectiveContextLimit - MaxOutputTokens     │
│   = 500,000 - 32,768                         │
│   = 467,232                                  │
│                                              │
│ If prompt > 467,232:                         │
│   → Trim/summarize old turns                 │
│   → Warn: "Quality may degrade above 500K"   │
│                                              │
│ Hard ceiling =                               │
│   ContextWindow - MaxOutputTokens             │
│   = 1,000,000 - 32,768                      │
│   = 967,232                                  │
│                                              │
│ If prompt > 967,232:                         │
│   → ERROR: Exceeds hard context limit        │
└─────────────────────────────────────────────┘
```

---

## Implementation Plan

### Phase 1: Core Types and Merge (geppetto only)

**Goal**: Define the types, implement merge, wire into `InferenceSettings`.

**Files to create**:

1. `geppetto/pkg/steps/ai/settings/model_info.go` — `ModelInfo`, `InputModality`, `ModelCost` structs + `MergeModelInfo`, `Clone`, `ComputeCost`, `EffectiveContextLimit`, `HardContextLimit` helpers.

**Files to modify**:

2. `geppetto/pkg/steps/ai/settings/settings-inference.go` — Add `ModelInfo *ModelInfo` field to `InferenceSettings`. Update `Clone()`, `GetMetadata()`, `GetSummary()`.

3. `geppetto/pkg/engineprofiles/inference_settings_merge.go` — Add `mergeModelInfo` to the `mergeInferenceSettings` path (since it's part of `InferenceSettings`, the YAML round-trip merge should handle it automatically, but verify and add explicit test).

4. `geppetto/pkg/steps/ai/settings/flags/inference.yaml` — Add `ai-model-info` glazed section flags (if needed for CLI).

**Test files**:

5. `geppetto/pkg/steps/ai/settings/model_info_test.go` — Unit tests for `MergeModelInfo`, `Clone`, `ComputeCost`, `EffectiveContextLimit`.

6. `geppetto/pkg/steps/ai/settings/settings-inference_test.go` — Add test cases for `InferenceSettings` with `ModelInfo` merge.

### Phase 2: Profile Integration (geppetto only)

**Goal**: Model metadata loads from YAML profiles and flows through resolution.

**Files to modify**:

7. `geppetto/pkg/engineprofiles/codec_yaml_runtime.go` — No changes needed (YAML codec already handles all `InferenceSettings` fields via struct tags), but add a test with model_info in YAML.

8. `geppetto/pkg/engineprofiles/stack_merge_test.go` — Add test for ModelInfo stack merge.

**Test files**:

9. `geppetto/pkg/engineprofiles/stack_merge_model_info_test.go` — Focused test for ModelInfo through profile stack.

**Example YAML fixture**:

```yaml
slug: test-provider
profiles:
  default:
    slug: default
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4o-mini
      model_info:
        id: gpt-4o-mini
        name: GPT-4o Mini
        reasoning: false
        input:
          - text
          - image
        context_window: 128000
        quality_high_watermark: 128000
        max_output_tokens: 16384
        cost:
          input: 0.15
          output: 0.60
          cache_read: 0.075
          cache_write: 0.30
```

### Phase 3: Engine Factory Integration (geppetto only)

**Goal**: Replace `isReasoningModel()` heuristics with `ModelInfo.Reasoning`.

**Files to modify**:

10. `geppetto/pkg/inference/engine/factory/factory.go` — Add `ModelInfo` to the engine creation path. Use `info.Reasoning` when available, fall back to name-prefix heuristic.

11. `geppetto/pkg/inference/engine/inference_config_sanitize.go` — Update `SanitizeForReasoningModel` to accept `ModelInfo` for decision.

### Phase 4: JS Module Surface (geppetto only)

**Goal**: Expose ModelInfo to JavaScript.

**Files to modify**:

12. `geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go` — Add `modelInfo` to `newResolvedEngineProfileObject()`. Add `newModelInfoObject()` helper.

13. `geppetto/pkg/js/modules/geppetto/api_engines.go` — Expose `modelInfo` on engine objects.

### Phase 5: Cost Computation (geppetto only)

**Goal**: Compute and persist cost on inference results.

**Files to modify**:

14. `geppetto/pkg/turns/inference_result.go` — Add `Cost *float64` field to `InferenceResult`.

15. `geppetto/pkg/inference/engine/inference_result_metadata.go` — After inference, if `ModelInfo` is available, compute cost and store it.

### Phase 6: Pinocchio Integration

**Goal**: Use ModelInfo in Pinocchio for context budgeting and UI rendering.

**Files to modify**:

16. `pinocchio/pkg/ui/profileswitch/` — Show model capabilities in the profile picker (reasoning badge, input type icons, context window gauge, cost indicator).

17. `pinocchio/pkg/cmds/profilebootstrap/engine_settings.go` — Pass `ModelInfo` through to engine creation.

18. `pinocchio/cmd/web-chat/` — Use `EffectiveContextLimit` for prompt trimming.

---

## Testing Strategy

### Unit Tests

| Test File | What It Tests |
|-----------|--------------|
| `model_info_test.go` | `MergeModelInfo` (base nil, overlay nil, both set, partial overlay), `Clone`, `ComputeCost` (all usage fields, zero usage, nil cost), `EffectiveContextLimit`, `HardContextLimit` |
| `settings-inference_test.go` | `InferenceSettings` round-trip with `ModelInfo`, merge with `ModelInfo` |
| `stack_merge_model_info_test.go` | Full profile stack merge with `model_info` in YAML |

### Integration Tests

| Test | What It Tests |
|------|--------------|
| Profile YAML with model_info | Load a YAML profile with model_info, resolve it, verify `InferenceSettings.ModelInfo` is populated |
| Stack merge with model_info | Load two profile layers with different model_info, verify overlay wins |
| JS module model_info | From JS, resolve a profile, access `resolved.modelInfo` |

### Validation Tests

| Test | What It Tests |
|------|--------------|
| quality_high_watermark > context_window | Must fail validation |
| negative cost | Must fail validation |
| empty model_info | Must round-trip correctly (all fields nil) |

---

## API Reference

### New Go Types

All new types live in `geppetto/pkg/steps/ai/settings/model_info.go`.

**`InputModality`** — string type with known constants: `text`, `image`, `audio`, `video`, `pdf`. Unrecognized values round-trip.

**`ModelCost`** — struct with `Input`, `Output`, `CacheRead`, `CacheWrite` (all `float64`, value types). USD per million tokens.

**`ModelInfo`** — struct with the fields described in the table above. All scalar fields are pointer types. Includes `Clone()` and `Validate()` methods.

The implementor should define the exact struct tags (`json`, `yaml`, `glazed`) following the existing patterns in `ChatSettings` and `InferenceConfig`.

### New Helper Functions

All helpers live in `geppetto/pkg/steps/ai/settings/model_info.go`.

| Function | Signature Sketch | Purpose |
|----------|------------------|----------|
| `NewModelInfo` | `() *ModelInfo` | Constructor with all fields nil |
| `ModelInfo.Clone` | `(m *ModelInfo) *ModelInfo` | Deep copy |
| `ModelCost.Clone` | `(m *ModelCost) *ModelCost` | Deep copy |
| `MergeModelInfo` | `(base, overlay *ModelInfo) *ModelInfo` | Overlay merge following `MergeInferenceConfig` pattern |
| `ModelInfo.ComputeCost` | `(m *ModelInfo, usage *InferenceUsage) *float64` | Token usage × cost rates |
| `ModelInfo.EffectiveContextLimit` | `(m *ModelInfo) int` | Returns `quality_high_watermark` if set, else `context_window` |
| `ModelInfo.HardContextLimit` | `(m *ModelInfo) int` | Returns `context_window` |
| `ModelInfo.Validate` | `(m *ModelInfo) error` | Invariants (e.g., `quality_high_watermark <= context_window`, non-negative costs) |

The implementor should decide exact signatures (methods vs functions, receiver vs first-arg, return types) based on what feels idiomatic in this codebase.

### Modified Structs

**`InferenceSettings`** gains a `ModelInfo *ModelInfo` field (YAML: `model_info,omitempty`). The implementor should also update `Clone()`, `GetMetadata()`, and `GetSummary()` to handle the new field.

**`InferenceResult`** gains a `Cost *float64` field (JSON/YAML: `cost,omitempty`) to persist the computed cost after inference.

### YAML Schema Addition

```yaml
# Under inference_settings:
model_info:
  id: string                  # canonical model ID
  name: string                # display name
  reasoning: bool             # supports extended thinking
  input: [string]             # "text" | "image" | "audio" | "video" | "pdf"
  context_window: int         # absolute max tokens
  quality_high_watermark: int # quality degrades above this
  max_output_tokens: int      # max response tokens
  cost:
    input: float              # USD per 1M input tokens
    output: float             # USD per 1M output tokens
    cache_read: float         # USD per 1M cache-read tokens
    cache_write: float        # USD per 1M cache-write tokens
  metadata:                   # untyped grab-bag
    key: value
```

---

## Risks and Alternatives

### Risks

| Risk | Mitigation |
|------|-----------|
| YAML backward compatibility — existing profiles without model_info must still load | All ModelInfo fields are optional (pointer types). Missing `model_info` key produces nil, which is valid. |
| Stack merge of cost fields — partial cost overlay (e.g., only override input cost) requires careful merge | ModelCost is replaced wholesale (not field-by-field). If you override cost, you must specify all rates. This avoids the "nil vs zero" ambiguity. |
| Model metadata becomes stale — providers change pricing or context windows | Model metadata is in YAML profiles (version-controlled). Updates are profile edits. No runtime API calls to fetch metadata (out of scope). |
| Extension key vs InferenceSettings dual source of truth | Decision: InferenceSettings is the source of truth. Extension key is read-only projection for JS. |

### Alternatives Considered

1. **Separate ModelCatalog service** — A standalone service that maps model names to metadata. Rejected because it adds a new service dependency and doesn't compose with the profile stack. Metadata belongs in the profile layer where it merges naturally.

2. **ModelInfo as a profile extension only** — Rejected because it requires decoding an extension for every consumer. Direct field access on InferenceSettings is more ergonomic and follows the existing pattern.

3. **ModelCost with pointer fields** — Rejected because it creates nil-vs-zero ambiguity for "free" vs "unknown." A nil `Cost` means unknown; a `Cost{}` with zeros means free.

4. **Context window as a single field** — Rejected because it cannot express quality degradation. The two-field approach (context_window + quality_high_watermark) captures the real-world behavior of long-context models.

---

## Open Questions

1. **Should ModelInfo auto-populate from an external model catalog API?** — Out of scope for v1, but the `Metadata` grab-bag can store vendor-specific identifiers that a future catalog service could resolve.

2. **Should we validate model_info.id against chat.engine?** — A warning (not error) if they don't match. Out of scope for v1.

3. **Should cost be USD or allow other currencies?** — USD per million tokens is the industry standard. If needed, add a `currency` field to `ModelCost` in a future version.

4. **How does this interact with the glazed CLI flag system?** — The `glazed:"ai-model-info"` tag enables CLI flag generation. We need to decide if model metadata should be settable from CLI flags (probably not for cost/context window, but maybe for reasoning/input).

---

## References

### Key Files

| File | Purpose |
|------|---------|
| `geppetto/pkg/engineprofiles/types.go` | Core EngineProfile, EngineProfileRegistry types |
| `geppetto/pkg/engineprofiles/extensions.go` | Profile extension key system |
| `geppetto/pkg/engineprofiles/stack_merge.go` | Stack merge for InferenceSettings and Extensions |
| `geppetto/pkg/engineprofiles/inference_settings_merge.go` | YAML round-trip merge for InferenceSettings |
| `geppetto/pkg/engineprofiles/source_chain.go` | ChainedRegistry loading YAML/SQLite |
| `geppetto/pkg/engineprofiles/codec_yaml_runtime.go` | YAML codec for engine profiles |
| `geppetto/pkg/steps/ai/settings/settings-inference.go` | InferenceSettings struct |
| `geppetto/pkg/steps/ai/settings/settings-chat.go` | ChatSettings with Engine and ApiType |
| `geppetto/pkg/inference/engine/inference_config.go` | InferenceConfig, merge, resolve helpers |
| `geppetto/pkg/inference/engine/factory/factory.go` | StandardEngineFactory dispatching |
| `geppetto/pkg/inference/engine/inference_config_sanitize.go` | SanitizeForReasoningModel |
| `geppetto/pkg/inference/engine/inference_result_metadata.go` | BuildInferenceResultFromEventMetadata |
| `geppetto/pkg/turns/inference_result.go` | InferenceResult, InferenceUsage |
| `geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go` | JS resolved profile object |
| `geppetto/pkg/js/modules/geppetto/api_engines.go` | JS engine creation |
| `pinocchio/pkg/inference/runtime/profile_runtime.go` | ProfileRuntime extension example |
| `pinocchio/pkg/ui/profileswitch/` | Profile picker UI |

### Existing Patterns to Follow

- **Pointer fields for optional scalars** — All optional fields on `ModelInfo` should use pointer types, following `ChatSettings` and `InferenceConfig`.
- **Clone methods** — Every struct should have a `Clone()` method, following `EngineProfile.Clone()`.
- **Merge function** — `MergeModelInfo` should follow the pattern of `MergeInferenceConfig`.
- **Profile extension key** — If needed, follow `WebChatProfileRuntimeExtension` pattern (see `pinocchio/pkg/inference/runtime/profile_runtime.go`).
- **YAML tags** — `yaml:"field_name,omitempty"` on all fields, following `InferenceSettings`.
- **JSON tags** — `json:"field_name,omitempty"` on all fields for API/JS consumption.
- **Validation** — `Validate()` method returning `error`, following `ValidateRegistry`.

### Data Example

The example from the ticket:

```json
{
  "id": "DeepSeek-V4-Pro",
  "name": "DeepSeek V4 Pro",
  "reasoning": true,
  "input": ["text"],
  "contextWindow": 262144,
  "maxTokens": 32768,
  "cost": {
    "input": 0,
    "output": 0,
    "cacheRead": 0,
    "cacheWrite": 0
  }
}
```

Maps to our YAML (the Go struct is left to the implementor):

```yaml
model_info:
  id: DeepSeek-V4-Pro
  name: DeepSeek V4 Pro
  reasoning: true
  input:
    - text
  context_window: 262144
  max_output_tokens: 32768
  cost:
    input: 0
    output: 0
    cache_read: 0
    cache_write: 0
```
