---
Title: 'Model catalog: design and implementation guide'
Ticket: GEPPETTO-288--model-catalog-yaml-for-model-defaults-costs-capabilities
Status: active
Topics:
    - geppetto
    - inference
    - yaml
    - config
    - profile-registry
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-04T10:37:10.832889935-05:00
WhatFor: ""
WhenToUse: ""
---

# Model Catalog (YAML) for Geppetto

Date: 2026-03-04  
Audience: new intern (first week in the repo)  
Goal: explain *what exists today*, *what we want to add*, and *exactly where/how to implement it*.

---

## 0) Problem Statement (what we’re trying to fix)

Geppetto currently treats “model selection” as a pair of runtime settings:

- `ai-engine` (model slug / model name string)
- `ai-api-type` (provider selection: `openai`, `openai-responses`, `claude`, `gemini`, …)

This works, but it has a major gap:

- There is **no canonical in-repo registry** of “known models” (by slug) that encodes:
  - max output tokens / context window,
  - recommended provider (`openai` vs `openai-responses` for reasoning-capable OpenAI models),
  - supported “thinking” / “reasoning” controls and valid levels,
  - pricing (input/output token costs, cache-related token pricing),
  - default settings we want to apply when the user didn’t specify an override.

As a result, model-specific behaviors are scattered as heuristics and provider-specific constraints:

- “Reasoning model” detection is currently **prefix-based** (`o1`, `o3`, `o4`, `gpt-5`) and duplicated:
  - `pkg/inference/engine/factory/factory.go` (warning)
  - `pkg/steps/ai/openai/helpers.go` (sanitization and `max_completion_tokens`)
  - `pkg/steps/ai/openai_responses/helpers.go` (omit sampling params)
  - `pkg/js/modules/geppetto/api_engines.go` (`inferAPIType`)

We want a single source of truth: a **YAML Model Catalog** with a **local override** file so users can add brand-new model slugs without waiting for a code release.

---

## 1) Current Architecture Tour (what exists today)

This section is intentionally concrete: you should be able to open each file and understand where it fits.

### 1.1 StepSettings is the “runtime configuration object”

Core file:

- `pkg/steps/ai/settings/settings-step.go`

Key types:

- `settings.StepSettings`
- `settings.ChatSettings`
- `engine.InferenceConfig` (defaults + per-turn overrides)

Important fields (simplified):

```go
type StepSettings struct {
  Chat      *ChatSettings         // ai-engine, ai-api-type, max_response_tokens, temperature, top_p, …
  OpenAI    *openai.Settings       // OpenAI-specific flags (reasoning_effort, reasoning_summary, …)
  Claude    *claude.Settings
  Gemini    *gemini.Settings
  API       *APISettings           // api keys + base urls
  Inference *engine.InferenceConfig // engine-level defaults for per-turn overrides
}

type ChatSettings struct {
  Engine            *string        // yaml:"engine" glazed:"ai-engine"
  ApiType           *types.ApiType // yaml:"api_type" glazed:"ai-api-type"
  MaxResponseTokens *int           // glazed:"ai-max-response-tokens"
  Temperature       *float64       // glazed:"ai-temperature"
  TopP              *float64       // glazed:"ai-top-p"
}
```

Where defaults come from:

- the `flags/*.yaml` embedded “ValueSection” defaults (Glazed schema)
  - `pkg/steps/ai/settings/flags/chat.yaml` (defaults: engine=`gpt-4`, api-type=`openai`)
  - `pkg/steps/ai/settings/openai/chat.yaml` (defaults for OpenAI flags, including `openai-reasoning-effort`)
  - `pkg/steps/ai/settings/flags/inference.yaml` (no defaults: designed to remain nil unless set)

### 1.2 Profiles can set StepSettings patches (profile-first is preferred)

Core files:

- `pkg/profiles/service.go` (resolves profile stack and applies step-settings patches)
- `pkg/profiles/runtime_settings_patch_resolver.go` (applies `runtime.step_settings_patch`)
- documentation: `pkg/doc/topics/01-profiles.md`

How it works:

- profiles store a `runtime.step_settings_patch` map like:

```yaml
runtime:
  step_settings_patch:
    ai-chat:
      ai-engine: gpt-4o-mini
      ai-api-type: openai
    ai-inference:
      inference-max-response-tokens: 1024
```

Then `profiles.StoreRegistry.ResolveEffectiveProfile(...)` merges the stack and calls:

- `ApplyRuntimeStepSettingsPatch(base, effectivePatch)` → returns *effective* `StepSettings`.

### 1.3 Engine selection uses ApiType, then model heuristics

Core file:

- `pkg/inference/engine/factory/factory.go`

Today:

- provider is selected from `settings.Chat.ApiType`
- if provider is `openai` *and* the model looks like a reasoning model (prefix match), Geppetto logs a warning suggesting `openai-responses`.

It does **not** auto-switch providers today.

### 1.4 Provider engines map StepSettings to provider wire requests

These files matter because “defaults” and “capabilities” must match what providers accept:

- OpenAI Chat Completions:
  - `pkg/steps/ai/openai/engine_openai.go`
  - `pkg/steps/ai/openai/helpers.go`
  - Special case: reasoning models use `MaxCompletionTokens` and reject sampling params.

- OpenAI Responses:
  - `pkg/steps/ai/openai_responses/engine.go`
  - `pkg/steps/ai/openai_responses/helpers.go`
  - Supports:
    - `reasoning.effort` (low/medium/high)
    - `reasoning.summary` (auto/concise/detailed)
    - `reasoning.max_tokens` (thinking budget)
  - Special case: reasoning models omit `temperature` and `top_p`.

- Claude Messages:
  - `pkg/steps/ai/claude/engine_claude.go`
  - `pkg/steps/ai/claude/helpers.go`
  - Supports “thinking budget” via `thinking.budget_tokens`
  - Constraint: if thinking is enabled, Claude requires `temperature=1.0` (or unset).

- Gemini:
  - `pkg/steps/ai/gemini/engine_gemini.go`
  - Uses `GenerationConfig.MaxOutputTokens` etc.

### 1.5 Per-turn inference overrides already exist

Core file:

- `pkg/inference/engine/inference_config.go`

Resolution order:

```
Turn.Data overrides
  > StepSettings.Inference defaults
    > StepSettings.Chat fields (temperature/top_p/max_response_tokens)
```

This is important: the Model Catalog should primarily set **StepSettings defaults**, not per-turn overrides.

---

## 2) Design Goals and Non-Goals

### Goals

- A **canonical “model slug → metadata” registry** shipped with Geppetto.
- A **local YAML override** for new models / corrections without a release.
- A stable API to ask: “Given model slug X, what are the defaults/capabilities/costs?”
- A resolver that can:
  - choose a default provider (`ai-api-type`) based on model slug,
  - choose a default output token cap (`ai-max-response-tokens`) per model,
  - determine whether sampling params are allowed (temperature/top_p),
  - determine supported thinking controls and levels,
  - compute cost from usage tokens (future, but schema-ready now).

### Non-goals (for v1)

- Automatically scraping provider docs/pricing from the internet at runtime.
- Full “model discovery” across providers.
- Perfect correctness of pricing data (we design the system so pricing can be updated locally).

---

## 3) Proposed Solution: a YAML-backed Model Catalog

### 3.1 Where it lives in the repo

Add a new package:

- `pkg/models` (or `pkg/modelcatalog`; pick one and be consistent)

And ship a built-in catalog file embedded into the binary:

- `pkg/models/catalog.yaml` (embedded with `//go:embed`)

Add an optional local override file that is loaded at runtime (if present):

- Recommended path: `${XDG_CONFIG_HOME:-~/.config}/pinocchio/models.yaml`
  - This matches how profile registries are treated in `pkg/sections/sections.go`.
  - If you want this to be app-agnostic, support `GEPPETTO_MODEL_CATALOG` as an override env var.

### 3.2 YAML schema (v1)

Keep the schema strict enough to be useful, but flexible for future additions.

Top-level:

```yaml
version: 1
models:
  <model_slug>:
    display_name: "Human friendly name"
    provider:
      default_api_type: openai|openai-responses|claude|gemini
      # Optional: some slugs exist on multiple providers / proxies
      compatible_api_types: [openai, openai-responses]

    limits:
      context_window_tokens: 128000
      max_output_tokens: 16384

    defaults:
      # Applied when StepSettings.Chat.MaxResponseTokens is nil
      max_response_tokens: 2048

      # Optional: some models want a different default sampling
      temperature: 0.7
      top_p: 1.0

      # Optional defaults for reasoning/thinking controls
      reasoning_effort: medium
      reasoning_summary: auto
      thinking_budget_tokens: 0

    capabilities:
      # “reasoning models reject sampling params” becomes data-driven
      allow_temperature: true
      allow_top_p: true

      thinking:
        # none | openai_reasoning | claude_thinking
        kind: none
        supported_effort: [low, medium, high]
        supported_summary: [auto, concise, detailed]
        max_thinking_budget_tokens: 0

    pricing:
      currency: USD
      # Prices are per 1M tokens to avoid lots of decimals.
      input_per_million: 0.0
      output_per_million: 0.0
      # Optional / provider-specific:
      cached_input_per_million: 0.0
      cache_write_input_per_million: 0.0
```

Notes:

- Many fields can be optional (pointers) so overrides can be partial.
- “context window” and “max output” are separate; we often only need “max output”.
- The catalog is designed to support pricing, even if v1 only uses it for reporting.

### 3.3 Merge rules (built-in + local override)

We want a deterministic merge that supports:

- adding new model slugs,
- overriding specific fields for existing slugs.

Recommended merge semantics:

1) Load built-in catalog (embedded YAML) into typed structs.
2) Load local override catalog (optional) into typed structs.
3) Merge:
   - models are keyed by slug
   - for a model present in both:
     - each field is overridden if the override field is non-nil
     - for slices: override slice when override slice is non-nil (even empty)
     - for nested structs: merge recursively using the same rule

Pseudocode:

```text
mergeModel(base, ov):
  out = base.clone()
  if ov.display_name != nil: out.display_name = ov.display_name
  out.provider = mergeProvider(base.provider, ov.provider)
  out.limits = mergeLimits(base.limits, ov.limits)
  out.defaults = mergeDefaults(base.defaults, ov.defaults)
  out.capabilities = mergeCapabilities(base.capabilities, ov.capabilities)
  out.pricing = mergePricing(base.pricing, ov.pricing)
  return out
```

### 3.4 Validation rules (fail fast)

Validation should happen at load time and include:

- `version == 1` (or a supported version list)
- model slug is non-empty and trimmed
- `provider.default_api_type` is one of `types.ApiType*` supported values
- token limits and defaults are non-negative
- supported thinking levels/summaries are from allowed sets:
  - effort: `low|medium|high`
  - summary: `auto|concise|detailed` (or `""` meaning “off” if you want)
- if `capabilities.thinking.kind == openai_reasoning`:
  - recommended api type should usually include `openai-responses`
- if a model disallows temperature/top_p, defaults shouldn’t set them (or resolver must sanitize).

---

## 4) How the Model Catalog should be used (integration points)

### 4.1 One “NormalizeStepSettings” entry point (most important)

Add a function that:

- inspects the selected model slug (`StepSettings.Chat.Engine`)
- looks up model metadata in the catalog
- applies defaults only when the user didn’t set explicit values

Recommended file:

- `pkg/models/normalize.go` (or `pkg/steps/ai/settings/normalize.go`, but keep model logic out of settings if possible)

What it should do:

1) Resolve effective model slug:
   - `engine := strings.TrimSpace(*ss.Chat.Engine)`
   - if empty, do nothing
2) Lookup `ModelSpec` from catalog (exact match for v1)
3) Apply:
   - Provider:
     - if `ss.Chat.ApiType == nil` or `*ss.Chat.ApiType == "auto"`:
       - set to `spec.provider.default_api_type`
   - Max response tokens:
     - if `ss.Chat.MaxResponseTokens == nil` and `spec.defaults.max_response_tokens != nil`:
       - set it
   - Inference defaults:
     - if `ss.Inference == nil`: initialize it
     - if `ss.Inference.ReasoningEffort == nil` and `spec.defaults.reasoning_effort != nil`:
       - set it
     - if `ss.Inference.ThinkingBudget == nil` and `spec.defaults.thinking_budget_tokens != nil`:
       - set it
4) Sanitize settings based on capabilities:
   - if `allow_temperature=false`: set `ss.Chat.Temperature=nil` (unless we decide to error instead)
   - if `allow_top_p=false`: set `ss.Chat.TopP=nil`

Diagram: normalization placement in the runtime pipeline

```text
CLI flags / profile patch / JS opts
          |
          v
   settings.StepSettings (raw)
          |
          v
   ModelCatalog.NormalizeStepSettings(ss)   <--- NEW
          |
          v
 engine/factory.CreateEngine(ss)
          |
          v
 provider engine builds requests
```

### 4.2 Replace duplicated heuristics with catalog queries

Today, “reasoning model” is inferred by prefix checks in several places.

Migration plan:

- Keep existing heuristics as *fallback* for unknown models (v1).
- For known models:
  - derive `isReasoning` from catalog capabilities (e.g. `allow_temperature=false`)
  - derive “recommended provider” from `provider.default_api_type`

Concretely, aim to remove or narrow these functions:

- `pkg/inference/engine/factory/factory.go:isReasoningModel`
- `pkg/steps/ai/openai/helpers.go:isReasoningModel`
- `pkg/steps/ai/openai_responses/helpers.go:isResponsesReasoningModel`
- `pkg/js/modules/geppetto/api_engines.go:inferAPIType`

Replace with something like:

```go
spec, ok := catalog.Lookup(model)
if ok {
  // use spec.capabilities.* and spec.provider.*
} else {
  // fallback heuristics for unknown models
}
```

### 4.3 Provide a “Describe model” API for UIs and tooling

Add a small Go API for callers:

```go
type Catalog interface {
  Lookup(slug string) (ModelSpec, bool)
  List() []ModelSpec
}
```

This enables:

- CLI commands: `geppetto models list`, `geppetto models show <slug>`
- UI usage: show max tokens, thinking levels, recommended provider
- runtime behavior: validate user overrides

### 4.4 (Optional v1) Cost computation hook

Geppetto already reports token usage in:

- `pkg/events/metadata.go` (`events.Usage`)

Where to compute cost:

- At “inference complete” time, we have `model` and `usage` in event metadata.
- Add a helper:

```go
func ComputeCostUSD(spec ModelSpec, usage events.Usage) (float64, bool)
```

Then attach:

- `metadata.Extra["cost_usd"] = ...` (or add a typed field later).

This is optional for v1, but design the YAML schema so it’s easy later.

---

## 5) Concrete Implementation Plan (step-by-step for an intern)

### Step A: Add the new package and types

Files to add:

- `pkg/models/types.go`
- `pkg/models/catalog_loader.go`
- `pkg/models/merge.go`
- `pkg/models/validate.go`
- `pkg/models/normalize.go`
- `pkg/models/catalog.yaml` (embedded defaults)

Key types (sketch):

```go
type CatalogFile struct {
  Version int `yaml:"version"`
  Models map[string]*ModelSpec `yaml:"models"`
}

type ModelSpec struct {
  Slug        string  `yaml:"-"` // key in map; populate after load
  DisplayName *string `yaml:"display_name,omitempty"`
  Provider    *ProviderSpec `yaml:"provider,omitempty"`
  Limits      *LimitsSpec `yaml:"limits,omitempty"`
  Defaults    *DefaultsSpec `yaml:"defaults,omitempty"`
  Capabilities *CapabilitiesSpec `yaml:"capabilities,omitempty"`
  Pricing     *PricingSpec `yaml:"pricing,omitempty"`
}
```

Implementation detail:

- Prefer pointers inside structs so “not specified” merges cleanly.
- After YAML load, set `ModelSpec.Slug` from the map key (canonical slug).

### Step B: Embed the built-in YAML catalog

Pattern to copy:

- `pkg/steps/ai/settings/openai/settings.go` embeds `chat.yaml` via `//go:embed`.

Add:

```go
//go:embed catalog.yaml
var builtInCatalogYAML []byte
```

### Step C: Implement local override loading

Implement:

```go
func LoadDefaultCatalog(ctx context.Context) (*Catalog, error)
```

Suggested behavior:

- load built-in
- if env var `GEPPETTO_MODEL_CATALOG` is set → load that YAML and merge
- else if `${XDG_CONFIG_HOME:-~/.config}/pinocchio/models.yaml` exists → load and merge

Be explicit about precedence in docs and tests.

### Step D: Normalize StepSettings before engine creation

Edit:

- `pkg/inference/engine/factory/helpers.go` or `pkg/inference/engine/factory/factory.go`

Recommended approach:

- in `NewEngineFromStepSettings`, call:
  - `models.NormalizeStepSettings(stepSettings, models.DefaultCatalog())`

This ensures:

- CLI usage,
- profile resolution usage,
- JS module usage,
all get consistent defaults.

### Step E: Add an “auto” provider option

To safely support “model chooses provider”, introduce a sentinel `ai-api-type=auto`.

Changes:

- `pkg/steps/ai/types/types.go`: add `ApiTypeAuto ApiType = "auto"`
- `pkg/steps/ai/settings/flags/chat.yaml`: include `"auto"` in choices and set default to `"auto"`
  - This is a behavior change, but `auto` should resolve to `openai` for most models, so it’s backward-safe.
- `StandardEngineFactory.CreateEngine(...)` must never see `provider=="auto"`; normalization must resolve it first.

### Step F: Replace prefix heuristics in hot paths (incremental)

Do this carefully:

1) Keep the old heuristic as fallback for unknown models.
2) For known models in catalog, use catalog capabilities instead.

This reduces regressions when users use a new model slug not in catalog yet.

### Step G: Add tests

Suggested test files:

- `pkg/models/catalog_loader_test.go`
- `pkg/models/merge_test.go`
- `pkg/models/normalize_test.go`

Test cases:

- loading built-in only
- override adds a new model
- override patches an existing model’s `max_response_tokens`
- `ai-api-type=auto` resolves to `openai-responses` for a reasoning model
- unknown model keeps fallback behavior

Also consider updating:

- `pkg/sections/profile_registry_source_test.go` and/or engine factory tests if defaults change.

---

## 6) Operational Guidance (how users will maintain overrides)

### Built-in catalog

- Lives in `pkg/models/catalog.yaml`
- Updated by PRs when we want to ship new defaults.

### Local override catalog

- Recommended path: `~/.config/pinocchio/models.yaml`
- Users can add:
  - new model slugs,
  - pricing changes,
  - max token changes,
  - provider defaults.

Example override file:

```yaml
version: 1
models:
  my-new-model-2026-02-01:
    provider:
      default_api_type: openai-responses
    limits:
      max_output_tokens: 8192
    defaults:
      max_response_tokens: 4096
    capabilities:
      allow_temperature: false
      allow_top_p: false
      thinking:
        kind: openai_reasoning
        supported_effort: [low, medium, high]
        supported_summary: [auto, concise, detailed]
```

---

## 7) Appendix: Where to look when debugging

When something looks “wrong” (wrong provider, wrong max tokens, thinking params rejected), check:

- Provider selection:
  - `pkg/inference/engine/factory/factory.go`
  - JS path: `pkg/js/modules/geppetto/api_engines.go`

- Request-building (provider wire params):
  - OpenAI chat: `pkg/steps/ai/openai/helpers.go`
  - OpenAI Responses: `pkg/steps/ai/openai_responses/helpers.go`
  - Claude: `pkg/steps/ai/claude/helpers.go`
  - Gemini: `pkg/steps/ai/gemini/engine_gemini.go`

- Defaults / schema:
  - `pkg/steps/ai/settings/flags/chat.yaml`
  - `pkg/steps/ai/settings/flags/inference.yaml`

- Profile patch application:
  - `pkg/profiles/runtime_settings_patch_resolver.go`
  - `pkg/profiles/service.go`

---

## 8) Suggested ticket breakdown (what to implement first)

Minimal v1 milestone:

- Add Model Catalog package + YAML embed + local override load
- Add `ai-api-type=auto` sentinel and normalization step
- Apply model-based defaults for:
  - provider (auto → recommended)
  - max response tokens (if unset)
  - sampling param allowance (sanitize)
- Add tests that cover the merge + normalization behavior
- Keep prefix heuristics as fallback for unknown slugs
