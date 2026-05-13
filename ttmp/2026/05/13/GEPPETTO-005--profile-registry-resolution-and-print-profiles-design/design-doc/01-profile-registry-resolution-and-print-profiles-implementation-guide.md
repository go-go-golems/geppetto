---
Title: Profile registry resolution and print-profiles implementation guide
Ticket: GEPPETTO-005
Status: active
Topics:
    - geppetto
    - pinocchio
    - coinvault
    - profiles
    - cli
    - configuration
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: ../../../../../../../2026-03-16--gec-rag/cmd/coinvault/cmds/chat_send.go
      Note: First application integration for --print-profiles early exit
    - Path: ../../../../../../../2026-03-16--gec-rag/cmd/coinvault/cmds/profile_settings.go
      Note: CoinVault custom profile settings merger and local defaults
    - Path: ../../../../../../../2026-03-16--gec-rag/internal/webchat/profiles.go
      Note: CoinVault inference profile registry opener and HTTP profile APIs
    - Path: ../../../../../../../2026-03-16--gec-rag/internal/webchat/sessionstream/sessionstream_runtime_resolver.go
      Note: CoinVault runtime composition from application profile plus inference profile
    - Path: ../../../../../../../pinocchio/pkg/cmds/profilebootstrap/profile_selection.go
      Note: Pinocchio wrapper around Geppetto bootstrap with config docs and inline profiles
    - Path: ../../../../../../../pinocchio/pkg/configdoc/profiles.go
      Note: Converts Pinocchio inline profiles into a Geppetto registry and composes registries
    - Path: pkg/cli/bootstrap/profile_introspection.go
      Note: Implemented reusable report construction
    - Path: pkg/cli/bootstrap/profile_introspection_test.go
      Note: Covers report building
    - Path: pkg/cli/bootstrap/profile_registry.go
      Note: Constructs ResolvedProfileRegistryChain from profile settings
    - Path: pkg/cli/bootstrap/profile_runtime.go
      Note: Central Geppetto CLI profile runtime resolution from config/env/defaults plus explicit CLI
    - Path: pkg/engineprofiles/service.go
      Note: Resolves selected engine profiles
    - Path: pkg/engineprofiles/source_chain.go
      Note: Parses profile registry sources and builds ChainedRegistry precedence/defaults
    - Path: pkg/sections/profile_introspection_section.go
      Note: Implemented the Glazed profile-introspection flags
    - Path: pkg/sections/profile_sections.go
      Note: Defines the generic --profile and --profile-registries Glazed section
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---



# Profile registry resolution and `--print-profiles` implementation guide

## Executive summary

This document explains how Geppetto, Pinocchio, and CoinVault currently load and resolve inference profiles, and proposes where to implement a `--print-profiles` CLI flag.

The short answer is:

- **The central profile model lives in Geppetto** under `pkg/engineprofiles`.
- **The central reusable CLI bootstrap path also lives in Geppetto** under `pkg/cli/bootstrap`.
- **Pinocchio mostly uses that bootstrap path**, then adds Pinocchio-specific unified config documents and inline profiles.
- **CoinVault partially uses Geppetto profile flags**, but still has custom profile resolution glue for `profile-registry.local.yaml`, `application-profiles.yaml`, HTTP cookies, and its chat runtime.
- The best long-term location for `--print-profiles` is therefore **Geppetto's CLI bootstrap/profile introspection layer**, with a small adapter for CoinVault's current custom path.

The proposed feature should answer three user questions:

1. **What profile registry sources were requested?**
2. **What registries and profiles were actually loaded, and which default registry/profile won?**
3. **For a selected profile, what stack layers and final merged inference settings/runtime overlays will the app use?**

A good first implementation can print a concise table of loaded profiles and defaults. A second implementation can add `--print-profile-resolution` or `--print-profiles=resolved` to show lineage and merged settings.

## Problem statement

The current UX makes it hard to understand why a command used a particular model/profile. Users pass flags such as:

```bash
--profile-registries ./profile-registry.local.yaml \
--profile gpt-5-low
```

or rely on implicit defaults such as:

- `~/.config/pinocchio/profiles.yaml`
- project-level `.pinocchio.yml`
- CoinVault's `./profile-registry.local.yaml`
- CoinVault's `./application-profiles.local.yaml`
- registry default profile slugs
- CLI overrides
- HTTP body/cookie profile selection

But there is no uniform CLI command or flag that prints the loaded registry stack and effective selection.

This creates three recurring debugging problems:

- A command appears to use the wrong model because a profile source higher in precedence shadowed another source.
- A profile slug exists in one registry but the default registry points somewhere else.
- App-specific layers, especially Pinocchio inline profiles and CoinVault application profiles, are confused with Geppetto inference profiles.

## Vocabulary

### Engine profile

An **engine profile** is the Geppetto concept for named LLM/inference settings. It lives in `geppetto/pkg/engineprofiles` and is represented by `EngineProfile`.

Important fields include:

- `Slug`
- `DisplayName`
- `Description`
- `Stack`
- `InferenceSettings`
- `Extensions`
- `Metadata`

Engine profiles are about **how to call the inference engine**: provider, model, API type, reasoning settings, middleware/runtime extensions, and related settings.

### Engine profile registry

An **engine profile registry** is a named collection of engine profiles. It is represented by `EngineProfileRegistry`.

Important fields include:

- `Slug`
- `DisplayName`
- `DefaultEngineProfileSlug`
- `Profiles`
- `Metadata`

A registry can be loaded from YAML or SQLite.

### Registry source

A **registry source** is an input location passed by CLI/config, such as:

```text
./profile-registry.yaml
yaml:./profiles.yaml
sqlite:/tmp/profiles.db
sqlite-dsn:file:/tmp/profiles.db?_foreign_keys=on
```

Geppetto parses these into `RegistrySourceSpec` values.

### Chained registry

A **chained registry** is Geppetto's multi-source registry. It loads multiple registry sources into one read interface, preserves source precedence, and resolves ambiguous profile slugs by searching top-of-stack sources first.

The type is `engineprofiles.ChainedRegistry`.

### Profile stack

An engine profile can reference base profiles through `Stack`. Stack resolution expands a selected profile into base-to-leaf layers and merges them deterministically.

Example conceptual YAML:

```yaml
profiles:
  base-openai:
    inference_settings:
      chat:
        engine: gpt-4.1-mini
  gpt-5-low:
    stack:
      - profile: base-openai
    inference_settings:
      chat:
        engine: gpt-5
      inference:
        reasoning_effort: low
```

The effective settings are:

```text
base-openai settings
  overlaid by gpt-5-low settings
```

### Pinocchio unified config document

Pinocchio adds `.pinocchio.yml` documents with top-level keys:

- `app`
- `profile`
- `profiles`

These can specify active profiles, imported registries, repositories, and inline profiles.

### CoinVault application profile

CoinVault has a separate concept called **application profile**. It is not the same as a Geppetto engine profile.

Application profiles control CoinVault prompt/tool behavior:

- selected system prompt
- prompt template
- enabled tool names

They live in `application-profiles.yaml` and are loaded by `internal/appprofiles`.

## High-level architecture

```text
                                ┌──────────────────────────────┐
                                │          CLI command          │
                                │ --profile, --profile-...      │
                                │ --print-profiles (proposed)   │
                                └──────────────┬───────────────┘
                                               │
                                               ▼
                         ┌─────────────────────────────────────────┐
                         │ Geppetto profile settings / bootstrap   │
                         │ pkg/sections + pkg/cli/bootstrap        │
                         └──────────────┬──────────────────────────┘
                                        │
                         registry source entries
                                        │
                                        ▼
                         ┌─────────────────────────────────────────┐
                         │ geppetto/pkg/engineprofiles             │
                         │ ParseRegistrySourceSpecs                │
                         │ NewChainedRegistryFromSourceSpecs       │
                         │ ResolveEngineProfile                    │
                         └──────────────┬──────────────────────────┘
                                        │
                    ┌───────────────────┴───────────────────┐
                    ▼                                       ▼
       ┌─────────────────────────┐             ┌─────────────────────────┐
       │ Pinocchio adapters       │             │ CoinVault adapters       │
       │ .pinocchio.yml           │             │ local profile registry   │
       │ inline profiles          │             │ application profiles     │
       │ profilebootstrap         │             │ webchat.OpenProfiles     │
       └─────────────────────────┘             └─────────────────────────┘
```

The key point is that **profile registry parsing, loading, chain precedence, stack expansion, and inference-setting merge semantics are centralized in Geppetto**. Application-specific layers select when and how to call that central code.

## File map

### Geppetto core profile model

| File | Responsibility |
|---|---|
| `geppetto/pkg/engineprofiles/types.go` | Data model for `EngineProfileRegistry`, `EngineProfile`, refs, metadata, and extensions. |
| `geppetto/pkg/engineprofiles/registry.go` | Public read interfaces: `RegistryReader`, `Registry`, `ResolveInput`, `ResolvedEngineProfile`. |
| `geppetto/pkg/engineprofiles/service.go` | `StoreRegistry`; applies default registry/profile slug and resolves a profile into stack+merged settings. |
| `geppetto/pkg/engineprofiles/source_chain.go` | Parses source specs and builds `ChainedRegistry` from YAML/SQLite sources. |
| `geppetto/pkg/engineprofiles/stack_resolver.go` | Expands profile stacks into deterministic base-to-leaf layers. |
| `geppetto/pkg/engineprofiles/stack_merge.go` | Merges stack layers. Later layers override earlier layers. |
| `geppetto/pkg/engineprofiles/inference_settings_merge.go` | Merges base and overlay `InferenceSettings`. |

### Geppetto CLI profile bootstrap

| File | Responsibility |
|---|---|
| `geppetto/pkg/sections/profile_sections.go` | Defines the generic Glazed `profile-settings` section with `--profile` and `--profile-registries`. |
| `geppetto/pkg/cli/bootstrap/profile_selection.go` | Resolves profile settings from parsed values and applies default registry source discovery. |
| `geppetto/pkg/cli/bootstrap/profile_registry_defaults.go` | Finds default `~/.config/<app>/profiles.yaml` if it exists. |
| `geppetto/pkg/cli/bootstrap/profile_registry.go` | Builds `ResolvedProfileRegistryChain` from profile settings. |
| `geppetto/pkg/cli/bootstrap/profile_runtime.go` | Resolves profile settings from config/env/defaults plus explicit CLI parsed values. |
| `geppetto/pkg/cli/bootstrap/engine_settings.go` | Merges hidden/base inference settings with resolved profile settings. |

### Pinocchio profile-specific code

| File | Responsibility |
|---|---|
| `pinocchio/pkg/cmds/profilebootstrap/profile_selection.go` | Pinocchio wrapper around Geppetto bootstrap; adds Pinocchio app config, config docs, inline profiles, and composition. |
| `pinocchio/pkg/cmds/profilebootstrap/engine_settings.go` | Resolves final engine settings for Pinocchio commands. |
| `pinocchio/pkg/configdoc/types.go` | Defines `.pinocchio.yml` document schema: `app`, `profile`, `profiles`. |
| `pinocchio/pkg/configdoc/load.go` | Loads and validates Pinocchio config docs. |
| `pinocchio/pkg/configdoc/merge.go` | Merges multiple Pinocchio config documents by precedence. |
| `pinocchio/pkg/configdoc/profiles.go` | Converts inline profiles to a Geppetto registry and composes imported+inline registries. |
| `pinocchio/cmd/web-chat/profiles/resolver.go` | HTTP request-time profile selection and runtime-plan resolution for web-chat. |
| `pinocchio/pkg/inference/runtime/runtime_plan.go` | Resolves Pinocchio runtime extensions from resolved engine profiles. |

### CoinVault profile-specific code

| File | Responsibility |
|---|---|
| `2026-03-16--gec-rag/cmd/coinvault/cmds/profile_settings.go` | CoinVault-specific profile settings, local file defaults, and merging with Geppetto `profile-settings`. |
| `2026-03-16--gec-rag/cmd/coinvault/cmds/chat_send.go` | Mounts command flags/sections and passes profile settings into the local runner. |
| `2026-03-16--gec-rag/cmd/coinvault/cmds/serve.go` | Mounts server flags/sections and passes profile settings into HTTP server runtime. |
| `2026-03-16--gec-rag/internal/webchat/profiles.go` | Opens Geppetto profile registries for CoinVault and exposes HTTP profile APIs. |
| `2026-03-16--gec-rag/internal/webchat/sessionstream/sessionstream_runtime_resolver.go` | Selects application profile + inference profile and composes the runtime. |
| `2026-03-16--gec-rag/internal/appprofiles/store.go` | Loads CoinVault application profiles from `application-profiles.yaml`. |

## Current Geppetto resolution path

### 1. CLI section definition

Geppetto defines the generic profile section in:

```text
geppetto/pkg/sections/profile_sections.go
```

It exposes:

```go
type ProfileSettings struct {
    Profile           string   `glazed:"profile"`
    ProfileRegistries []string `glazed:"profile-registries"`
}
```

The corresponding flags are:

```text
--profile
--profile-registries
```

This section is intentionally generic. It does not know about Pinocchio, CoinVault, web-chat, application profiles, SQL tools, or cookies.

### 2. CLI/config/env/default resolution

The reusable bootstrap layer lives in:

```text
geppetto/pkg/cli/bootstrap
```

The important call is:

```go
ResolveCLIProfileRuntime(ctx, cfg, parsed)
```

Conceptual pseudocode:

```go
func ResolveCLIProfileRuntime(ctx, cfg, parsed):
    profileSection := cfg.NewProfileSection()
    schema := schema(profileSection)

    resolvedValues := values.New()

    configMiddleware, configFiles := resolveConfigMiddleware(cfg, parsed)

    sources.Execute(
        schema,
        resolvedValues,
        FromEnv(cfg.EnvPrefix),
        configMiddleware,
        FromDefaults,
    )

    if parsed != nil:
        resolvedValues.Merge(parsed)  // explicit CLI values win

    settings := ResolveProfileSettings(resolvedValues)
    settings = PrepareProfileSettingsForRuntime(cfg, settings)

    registryChain := ResolveProfileRegistryChain(ctx, settings)

    return ResolvedCLIProfileRuntime{
        ProfileSettings: settings,
        ConfigFiles: configFiles,
        ProfileRegistryChain: registryChain,
    }
```

### 3. Default profile registry source

If no `profile-registries` are configured, Geppetto's bootstrap can discover:

```text
~/.config/<app>/profiles.yaml
```

This lives in:

```text
geppetto/pkg/cli/bootstrap/profile_registry_defaults.go
```

The behavior is intentionally app-name-aware:

```go
path := filepath.Join(os.UserConfigDir(), cfg.AppName, "profiles.yaml")
```

For Pinocchio, that is usually:

```text
~/.config/pinocchio/profiles.yaml
```

For another app using the same bootstrap, it would be:

```text
~/.config/<that-app>/profiles.yaml
```

### 4. Registry source parsing

`profile-registries` eventually becomes source specs through:

```go
engineprofiles.ParseRegistrySourceSpecs(entries)
```

Supported forms include:

```text
yaml:PATH
yaml://PATH
sqlite:PATH
sqlite-dsn:DSN
PATH.yaml
PATH.db
```

If no prefix is present, Geppetto infers YAML vs SQLite by extension/header.

### 5. Chained registry construction

The chain builder is:

```go
engineprofiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
```

It loads every source and creates one aggregate read interface.

Important behavior:

- Duplicate registry slugs across sources are rejected.
- The default registry is chosen from the last loaded source that had registries.
- Profile slug lookup without an explicit registry searches top-of-stack sources first.

Conceptually:

```go
for source in sources:
    registries := load(source)
    for reg in registries:
        if reg.Slug already loaded:
            error duplicate registry slug
        aggregate[reg.Slug] = reg

// Later sources have higher precedence for ambiguous profile lookup.
precedenceTopFirst := reverse(sourceRegistryOrder)
defaultRegistry := firstRegistryFromLastSource
```

### 6. Resolving a profile

The central call is:

```go
registry.ResolveEngineProfile(ctx, engineprofiles.ResolveInput{
    RegistrySlug:      selectedRegistry,
    EngineProfileSlug: selectedProfile,
})
```

In `StoreRegistry.ResolveEngineProfile(...)`:

1. If no registry is provided, use the registry default.
2. If no profile is provided, use the registry's `DefaultEngineProfileSlug`, or `default` if present.
3. Expand the profile stack using `ExpandEngineProfileStack(...)`.
4. Merge stack layers using `MergeEngineProfileStackLayers(...)`.
5. Return `ResolvedEngineProfile` with lineage and metadata.

### 7. Merging base settings with profile settings

Geppetto has two merges that are easy to confuse:

1. **Stack merge**: base profile -> leaf profile.
2. **Runtime base merge**: command/env/config base inference settings -> resolved profile inference settings.

The second merge happens in:

```text
geppetto/pkg/cli/bootstrap/engine_settings.go
```

Conceptual pseudocode:

```go
base := ResolveBaseInferenceSettings(cfg, parsed)
profileRuntime := ResolveCLIProfileRuntime(ctx, cfg, parsed)
resolved := profileRuntime.Registry.ResolveEngineProfile(ctx, profileRuntime.DefaultProfileResolve)
final := engineprofiles.MergeInferenceSettings(base, resolved.InferenceSettings)
```

Overlay wins for scalars. Nested maps merge recursively. Cost objects are replaced as a unit.

## Current Pinocchio resolution path

Pinocchio uses Geppetto's bootstrap but adds a unified config document system.

The adapter starts in:

```text
pinocchio/pkg/cmds/profilebootstrap/profile_selection.go
```

The Pinocchio bootstrap config declares:

```go
AppName:   "pinocchio"
EnvPrefix: "PINOCCHIO"
NewProfileSection: geppettosections.NewProfileSettingsSection
BuildBaseSections: geppettosections.CreateGeppettoSections
ConfigPlanBuilder: pinocchioConfigPlanBuilder
```

Pinocchio config files use this schema:

```yaml
app:
  repositories: []
profile:
  active: gpt-5-low
  registries:
    - ./profiles.yaml
profiles:
  local-profile:
    display_name: Local profile
    inference_settings: {}
    extensions: {}
```

Pinocchio's config plan searches paths such as:

```text
system app config
home app config
XDG app config
git-root .pinocchio.yml
git-root .pinocchio.override.yml
cwd .pinocchio.yml
cwd .pinocchio.override.yml
explicit --config-file
```

### Inline profiles

Pinocchio can define inline profiles inside `.pinocchio.yml`. These are converted to a normal Geppetto registry by:

```text
pinocchio/pkg/configdoc/profiles.go
```

The inline registry slug is:

```text
config-inline
```

Composition behavior:

```text
inline profiles first, imported registries second
```

That means inline profiles can be resolved through the same `geppetto/pkg/engineprofiles.Registry` interface.

### Pinocchio-specific runtime extensions

Pinocchio runtime overlays are not part of Geppetto's base inference settings. They live in profile extensions and are resolved in:

```text
pinocchio/pkg/inference/runtime/runtime_plan.go
```

A resolved profile may therefore affect:

- inference settings
- system prompt
- tools
- middleware list
- runtime fingerprint/profile version

## Current CoinVault resolution path

CoinVault is similar but not identical.

It mounts Geppetto's generic profile section, but it has its own profile settings resolver:

```text
2026-03-16--gec-rag/cmd/coinvault/cmds/profile_settings.go
```

CoinVault has two distinct profile families:

### 1. Inference profiles

These are Geppetto engine profiles selected by:

```text
--profile-registries
--registry
--profile
```

They control model/provider/inference behavior.

CoinVault resolves them through:

```go
webchat.OpenInferenceProfiles(ctx, opts.ProfileRegistries, opts.Registry, opts.Profile)
```

in:

```text
2026-03-16--gec-rag/internal/webchat/profiles.go
```

This function calls the same Geppetto core profile APIs:

```go
entries := gepprofiles.ParseEngineProfileRegistrySourceEntries(registrySources)
specs := gepprofiles.ParseRegistrySourceSpecs(entries)
chain := gepprofiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
```

But it is a CoinVault-local wrapper, not Geppetto bootstrap.

### 2. Application profiles

These are CoinVault prompt/tool profiles selected by:

```text
--application-profiles
--application-profile
```

They are loaded by:

```text
2026-03-16--gec-rag/internal/appprofiles/store.go
```

Application profiles select:

- prompt text or prompt template
- tool list
- default application behavior

They do **not** select the LLM provider/model directly.

### CoinVault runtime composition

The core runtime resolver is:

```text
2026-03-16--gec-rag/internal/webchat/sessionstream/sessionstream_runtime_resolver.go
```

It does:

```go
appSlug, appProfile := resolveApplicationProfile(in.ApplicationProfile)
profilePlan := resolveInferenceRuntimePlan(ctx, req, in)
systemPrompt := RenderApplicationProfileSystemPromptWithSQLDoc(appProfile, sqlDocs)
runtimeProfile := ProfileRuntime{SystemPrompt: systemPrompt, Tools: appProfile.Tools}

if profilePlan != nil:
    inferenceSettings = profilePlan.InferenceSettings
    runtimeProfile = profilePlan.Runtime.Clone()
    runtimeProfile.SystemPrompt = systemPrompt
    runtimeProfile.Tools = appProfile.Tools
```

So CoinVault intentionally overlays application-profile prompt/tools on top of inference-profile runtime/inference settings.

## Where should `--print-profiles` live?

### Recommendation

Implement the reusable feature in **Geppetto**, with one or two small app adapters.

The best location is not the low-level `engineprofiles` package alone, because `engineprofiles` does not know about CLI config/env/default discovery. It only knows how to parse explicit source specs and resolve registries.

The best central location is:

```text
geppetto/pkg/cli/bootstrap
```

with a small public API that can be used by:

- Geppetto example commands
- Pinocchio profilebootstrap commands
- Pinocchio web-chat CLI/server commands
- CoinVault `chat send` and `serve`

The flag definition can live in one of two places:

#### Option A: Add `--print-profiles` to `profile-settings`

File:

```text
geppetto/pkg/sections/profile_sections.go
```

Pros:

- Users naturally expect profile introspection next to `--profile` and `--profile-registries`.
- Any command mounting the profile section automatically gets the flag.
- Simple UX.

Cons:

- `ProfileSettings` currently contains only selection inputs. Adding an action flag mixes selection state with command behavior.
- Existing code decoding `ProfileSettings` must tolerate the extra field. This is normally fine, but semantically it is a different concern.

#### Option B: Add a new `profile-introspection` section

Files:

```text
geppetto/pkg/sections/profile_introspection_section.go
geppetto/pkg/cli/bootstrap/profile_introspection.go
```

Pros:

- Cleaner separation between selection and introspection behavior.
- Can grow additional flags without polluting `ProfileSettings`:
  - `--print-profiles`
  - `--print-profile-resolution`
  - `--print-profile-format table|json|yaml`
  - `--print-profile-settings`
- Commands opt in explicitly.

Cons:

- Apps must remember to mount the section.
- Slightly more boilerplate.

### Preferred design

Use **Option B internally**, while exposing the flag as plain `--print-profiles`.

That means:

- Add a new Geppetto section for introspection action flags.
- Add helper functions in Geppetto bootstrap that consume existing `ProfileSettings` plus resolved registry chain.
- Let apps decide whether to run this as an early-exit command behavior.

This avoids making `ProfileSettings` responsible for command flow control.

## Proposed command UX

### Minimal UX

```bash
coinvault chat send \
  --profile-registries ./profile-registry.local.yaml \
  --print-profiles
```

Expected output:

```text
Profile sources:
  1. ./profile-registry.local.yaml (yaml)

Registries:
  default *
    default profile: gpt-5-low
    profiles: 4

Profiles:
  registry  default  slug          model      api_type          description
  default   *        gpt-5-low     gpt-5     openai-responses  GPT-5 low reasoning
  default            gpt-5-nano    gpt-5-nano openai-responses Nano profile
```

No LLM call should be made. No database should be required unless the app chooses to validate DB before flags. For CoinVault specifically, `--print-profiles` should be handled **before** database startup checks.

### JSON UX

If the command already supports Glazed output, `--print-profiles --with-glaze-output --output json` should emit structured rows or a structured document.

Example document shape:

```json
{
  "sources": [
    {"raw": "./profile-registry.local.yaml", "kind": "yaml", "path": "./profile-registry.local.yaml"}
  ],
  "default_registry": "default",
  "selected_profile": "gpt-5-low",
  "registries": [
    {"slug": "default", "default_profile_slug": "gpt-5-low", "profile_count": 4}
  ],
  "profiles": [
    {
      "registry": "default",
      "slug": "gpt-5-low",
      "is_default": true,
      "display_name": "GPT-5 Low",
      "description": "OpenAI Responses GPT-5 profile with low reasoning effort.",
      "version": 1
    }
  ]
}
```

### Resolved-profile UX

A later enhancement should include:

```bash
--print-profile-resolution
```

or:

```bash
--print-profiles=resolved
```

Expected output:

```text
Resolved profile: default/gpt-5-low

Lineage:
  1. default/base-openai        version 3  source profiles.yaml
  2. default/gpt-5-low          version 7  source profiles.local.yaml

Merged inference settings:
  chat.engine: gpt-5
  chat.api_type: openai-responses
  inference.reasoning_effort: low
  inference.reasoning_summary: concise

Runtime extensions:
  pinocchio.webchat_runtime@v1:
    tools: [sql_query, sql_doc]
    middlewares: [...]
```

Do not print provider API keys or secrets. The implementation must redact sensitive settings.

## Proposed API design

### Data structures

Create a new package file:

```text
geppetto/pkg/cli/bootstrap/profile_introspection.go
```

Suggested types:

```go
type ProfileIntrospectionSettings struct {
    PrintProfiles bool   `glazed:"print-profiles"`
    PrintProfileResolution bool `glazed:"print-profile-resolution"`
    ProfileOutput string `glazed:"profile-output"`
}

type ProfileRegistryReport struct {
    Sources []ProfileRegistrySourceReport `json:"sources" yaml:"sources"`
    DefaultRegistry string `json:"default_registry" yaml:"default_registry"`
    SelectedProfile string `json:"selected_profile,omitempty" yaml:"selected_profile,omitempty"`
    Registries []ProfileRegistrySummaryReport `json:"registries" yaml:"registries"`
    Profiles []ProfileSummaryReport `json:"profiles" yaml:"profiles"`
    Resolution *ProfileResolutionReport `json:"resolution,omitempty" yaml:"resolution,omitempty"`
}

type ProfileRegistrySourceReport struct {
    Raw string `json:"raw" yaml:"raw"`
    Kind string `json:"kind" yaml:"kind"`
    Path string `json:"path,omitempty" yaml:"path,omitempty"`
    DSN string `json:"dsn,omitempty" yaml:"dsn,omitempty"`
}

type ProfileSummaryReport struct {
    Registry string `json:"registry" yaml:"registry"`
    Slug string `json:"slug" yaml:"slug"`
    DisplayName string `json:"display_name,omitempty" yaml:"display_name,omitempty"`
    Description string `json:"description,omitempty" yaml:"description,omitempty"`
    IsDefault bool `json:"is_default,omitempty" yaml:"is_default,omitempty"`
    Version uint64 `json:"version,omitempty" yaml:"version,omitempty"`
    Model string `json:"model,omitempty" yaml:"model,omitempty"`
    APIType string `json:"api_type,omitempty" yaml:"api_type,omitempty"`
}

type ProfileResolutionReport struct {
    Registry string `json:"registry" yaml:"registry"`
    Profile string `json:"profile" yaml:"profile"`
    Lineage []ResolvedProfileStackEntry `json:"lineage" yaml:"lineage"`
    InferenceSettings map[string]any `json:"inference_settings,omitempty" yaml:"inference_settings,omitempty"`
    Extensions map[string]any `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}
```

### Helper functions

Suggested API:

```go
func NewProfileIntrospectionSection() (schema.Section, error)

func ResolveProfileIntrospectionSettings(parsed *values.Values) ProfileIntrospectionSettings

func BuildProfileRegistryReport(
    ctx context.Context,
    cfg AppBootstrapConfig,
    parsed *values.Values,
    opts ProfileRegistryReportOptions,
) (*ProfileRegistryReport, func(), error)
```

Options:

```go
type ProfileRegistryReportOptions struct {
    IncludeResolution bool
    IncludeMergedSettings bool
    RedactSecrets bool
}
```

Pseudocode:

```go
func BuildProfileRegistryReport(ctx, cfg, parsed, opts):
    runtime, err := ResolveCLIProfileRuntime(ctx, cfg, parsed)
    if err != nil:
        return nil, nil, err

    chain := runtime.ProfileRegistryChain
    if chain == nil || chain.Registry == nil:
        return empty report with config file/source diagnostics

    report.Sources = source specs from runtime.ProfileSettings.ProfileRegistries
    report.DefaultRegistry = chain.DefaultRegistrySlug.String()

    registries := chain.Registry.ListRegistries(ctx)
    for each registrySummary:
        report.Registries append summary
        reg := chain.Registry.GetRegistry(ctx, registrySummary.Slug)
        profiles := chain.Registry.ListEngineProfiles(ctx, registrySummary.Slug)
        for profile in profiles:
            report.Profiles append profile summary

    if opts.IncludeResolution:
        resolved := chain.Registry.ResolveEngineProfile(ctx, chain.DefaultProfileResolve)
        report.Resolution = from resolved

    redact secrets
    return report, runtime.Close, nil
```

### Rendering

Do not force every application to hand-write table rendering.

Two approaches are possible:

1. Return typed rows and let Glazed render them.
2. Provide a small plain-text renderer for classic CLI output.

Recommended first implementation:

```go
func RenderProfileRegistryReportText(w io.Writer, report *ProfileRegistryReport) error
func ProfileRegistryReportRows(report *ProfileRegistryReport) []types.Row
```

This allows both classic and Glazed output.

## How this wires into applications

### Geppetto-native commands

Commands that already use `bootstrap.AppBootstrapConfig` should:

1. Add the new introspection section.
2. Decode `ProfileIntrospectionSettings` early.
3. If `PrintProfiles`, call `BuildProfileRegistryReport` and exit before inference.

Pseudocode:

```go
func Run(ctx, parsed):
    introspection := bootstrap.ResolveProfileIntrospectionSettings(parsed)
    if introspection.PrintProfiles:
        report, close, err := bootstrap.BuildProfileRegistryReport(ctx, cfg, parsed, opts)
        defer close()
        if err != nil: return err
        return bootstrap.RenderProfileRegistryReportText(os.Stdout, report)

    // existing command behavior
```

### Pinocchio commands

Pinocchio can either expose the Geppetto helper directly or wrap it in `pkg/cmds/profilebootstrap`.

Preferred wrapper:

```go
func NewProfileIntrospectionSection() (schema.Section, error) {
    return bootstrap.NewProfileIntrospectionSection()
}

func BuildProfileRegistryReport(ctx context.Context, parsed *values.Values, opts bootstrap.ProfileRegistryReportOptions) (...) {
    return bootstrap.BuildProfileRegistryReport(ctx, pinocchioBootstrapConfig(), parsed, opts)
}
```

But Pinocchio has inline profiles. The generic Geppetto report should not ignore them.

Pinocchio's `ResolveCLIProfileRuntime` currently composes imported registries with inline profiles. Therefore Pinocchio needs either:

- a Pinocchio-specific report builder that calls `profilebootstrap.ResolveCLIProfileRuntime`, or
- a Geppetto report builder extension point that lets apps provide their own `ResolveCLIProfileRuntime` function.

For an intern, the safer implementation is:

```go
// In Pinocchio profilebootstrap:
func BuildProfileRegistryReport(ctx, parsed, opts):
    runtime, err := ResolveCLIProfileRuntime(ctx, parsed) // Pinocchio wrapper, includes inline profiles
    return bootstrap.BuildReportFromResolvedProfileRuntime(ctx, runtime, opts)
```

This suggests splitting the Geppetto implementation into two layers:

```go
BuildProfileRegistryReportFromBootstrap(ctx, cfg, parsed, opts)
BuildProfileRegistryReportFromRuntime(ctx, runtime, opts)
```

### CoinVault commands

CoinVault currently does not use Geppetto's full bootstrap path. It does:

```go
profileSettings := resolveProfileSettings(vals)
profileDeps, err := webchat.OpenInferenceProfiles(ctx, profileSettings.ProfileRegistries, profileSettings.Registry, profileSettings.Profile)
```

Therefore CoinVault can use the central Geppetto report types/renderers but will need a local bridge:

```go
func buildCoinVaultProfileReport(ctx context.Context, vals *values.Values) (*bootstrap.ProfileRegistryReport, func(), error) {
    settings := resolveProfileSettings(vals)
    deps, err := webchat.OpenInferenceProfiles(ctx, settings.ProfileRegistries, settings.Registry, settings.Profile)
    if err != nil { return nil, nil, err }

    report, err := bootstrap.BuildProfileRegistryReportFromRegistry(ctx, bootstrap.ProfileRegistryReportInput{
        SourceEntries: parse entries from settings.ProfileRegistries,
        Registry: deps.Registry,
        DefaultRegistrySlug: deps.DefaultRegistrySlug,
        DefaultProfileSlug: deps.DefaultProfileSlug,
    })
    return report, deps.Close, err
}
```

CoinVault should handle `--print-profiles` before database startup. That matters because profile introspection does not require the application MySQL database.

In `chat_send.go`, the early-exit shape should be:

```go
func (c *ChatSendCommand) Run(ctx context.Context, vals *values.Values) error {
    if wantsPrintProfiles(vals) {
        return runPrintProfiles(ctx, vals, os.Stdout)
    }

    settings, req, opts, timeout, err := prepareChatSend(vals)
    ... existing behavior ...
}
```

Do the same in `serve.go` if desired.

### CoinVault application-profile report

Because CoinVault has application profiles, a separate optional flag may be useful:

```text
--print-application-profiles
```

Do not overload `--print-profiles` to print both kinds unless the output labels them very clearly.

Recommended CoinVault behavior:

- `--print-profiles`: inference profiles only.
- `--print-application-profiles`: CoinVault app prompt/tool profiles.
- `--print-all-profiles`: both, clearly separated.

## Security requirements

Profile files often contain provider API keys. The implementation must not print secrets.

Never print raw `InferenceSettings` without redaction.

Redact fields whose names include:

```text
key
api_key
api-key
token
secret
password
credential
authorization
```

Pseudocode:

```go
func redactAny(v any) any:
    switch typed := v.(type):
    case map[string]any:
        out := map[string]any{}
        for k, child := range typed:
            if isSensitiveKey(k):
                out[k] = "***REDACTED***"
            else:
                out[k] = redactAny(child)
        return out
    case []any:
        return map(redactAny, typed)
    default:
        return typed
```

Also avoid logging raw registry YAML in debug output.

## Implementation plan for a new intern

### Phase 1: Add generic Geppetto introspection types

Files:

```text
geppetto/pkg/cli/bootstrap/profile_introspection.go
geppetto/pkg/cli/bootstrap/profile_introspection_test.go
```

Implement:

- report structs
- redaction helper
- classic text renderer
- rows helper if needed

Tests:

- redacts nested API keys
- prints empty report cleanly
- marks default registry/profile

### Phase 2: Add profile introspection section

File:

```text
geppetto/pkg/sections/profile_introspection_section.go
```

Fields:

```text
--print-profiles              bool
--print-profile-resolution    bool
--profile-output              string, optional, default table/text
```

Keep this separate from `ProfileSettings` unless there is a strong reason to couple them.

### Phase 3: Build reports from resolved runtime

Add a function that takes already-resolved profile runtime/registry inputs.

Suggested minimal API:

```go
type ProfileRegistryReportInput struct {
    SourceEntries []string
    Registry gepprofiles.Registry
    DefaultRegistrySlug gepprofiles.RegistrySlug
    DefaultProfileSlug gepprofiles.EngineProfileSlug
    ResolveInput gepprofiles.ResolveInput
}

func BuildProfileRegistryReportFromRegistry(ctx context.Context, in ProfileRegistryReportInput, opts ProfileRegistryReportOptions) (*ProfileRegistryReport, error)
```

This makes it usable by Geppetto bootstrap, Pinocchio wrappers, and CoinVault custom glue.

### Phase 4: Build reports from Geppetto bootstrap

Add:

```go
func BuildProfileRegistryReport(ctx context.Context, cfg AppBootstrapConfig, parsed *values.Values, opts ProfileRegistryReportOptions) (*ProfileRegistryReport, func(), error)
```

Internally:

```go
runtime, err := ResolveCLIProfileRuntime(ctx, cfg, parsed)
return BuildProfileRegistryReportFromRegistry(...runtime...), runtime.Close, err
```

### Phase 5: Add Pinocchio wrapper

File:

```text
pinocchio/pkg/cmds/profilebootstrap/profile_introspection.go
```

Use Pinocchio's own `ResolveCLIProfileRuntime(ctx, parsed)` so inline `.pinocchio.yml` profiles are included.

### Phase 6: Wire to CoinVault `chat send`

Files:

```text
2026-03-16--gec-rag/cmd/coinvault/cmds/chat_send.go
2026-03-16--gec-rag/cmd/coinvault/cmds/serve.go
2026-03-16--gec-rag/cmd/coinvault/cmds/profile_settings.go
```

Add the introspection section to command sections.

Important ordering:

```go
if printProfiles {
    // do this before NewLocalRunner, before DB checks, before LLM calls
    return runCoinVaultPrintProfiles(ctx, vals, os.Stdout)
}
```

### Phase 7: Add tests

Geppetto tests:

- YAML source loads and prints profiles.
- Multiple source precedence is reported correctly.
- Default registry/profile is marked.
- Stack lineage appears in resolved report.
- Redaction works.

CoinVault tests:

- `chat send --print-profiles` does not require a DB.
- local `profile-registry.local.yaml` preference is reflected.
- `--registry` selects the default registry used in the report.
- `--profile` selects the profile shown as selected.

Pinocchio tests:

- imported registry appears.
- inline `.pinocchio.yml` profile appears.
- inline profile wins when resolving by unqualified profile slug.

## Proposed output examples

### Table output

```text
Profile sources
  1. yaml ./profile-registry.local.yaml

Default selection
  registry: default
  profile:  gpt-5-low

Registries
  * default  profiles=4  default_profile=gpt-5-low

Profiles
  default  *  gpt-5-low        gpt-5       openai-responses  GPT-5 low reasoning
  default     gpt-5-nano-low   gpt-5-nano  openai-responses  GPT-5 nano low reasoning
```

### Resolution output

```text
Resolved profile
  registry: default
  profile:  gpt-5-low

Stack lineage
  1. default/base-openai   version=1 source=profile-registry.yaml
  2. default/gpt-5-low     version=3 source=profile-registry.local.yaml

Merged settings summary
  chat.engine: gpt-5
  chat.api_type: openai-responses
  inference.reasoning_effort: low
  inference.reasoning_summary: concise
```

## Key design decisions

### Decision 1: Put the reusable feature in Geppetto

Reason: Geppetto owns the core profile registry abstractions and generic bootstrap resolution. A `--print-profiles` implementation should not be duplicated in every app.

### Decision 2: Let Pinocchio wrap the report builder

Reason: Pinocchio has inline profiles from `.pinocchio.yml`. A purely generic Geppetto bootstrap report would miss those unless the report builder accepts already-composed registries.

### Decision 3: Let CoinVault use a bridge first

Reason: CoinVault currently has a custom profile-settings resolver and application-profile layer. Refactoring CoinVault to use Geppetto bootstrap is probably worthwhile, but it is larger than adding `--print-profiles`. The bridge gets the UX quickly while still using central Geppetto report code.

### Decision 4: Keep inference profiles separate from application profiles

Reason: CoinVault application profiles are prompt/tool profiles, not provider/model profiles. Mixing them under one flag would increase confusion.

### Decision 5: Redact secrets by default

Reason: Local profile registries commonly include API keys. A profile printer must be safe to paste into bug reports.

## Open questions

1. Should `--print-profiles` output be a boolean flag or a mode flag?

   Options:

   ```text
   --print-profiles
   --print-profiles=summary|resolved|settings
   ```

   Recommendation: start with boolean `--print-profiles` plus separate `--print-profile-resolution`.

2. Should Geppetto's generic `ProfileSettings` grow a `Registry` field?

   CoinVault currently has `--registry` separately. Geppetto's `ResolveInput` supports registry slug selection, but the generic CLI profile section only has `--profile` and `--profile-registries`. Adding `--registry` centrally would align HTTP submit bodies and CLI behavior.

   Recommendation: consider adding a generic `--registry` in a follow-up. It is conceptually part of profile selection.

3. Should CoinVault migrate fully to Geppetto bootstrap?

   Recommendation: not as part of the first `--print-profiles` implementation. First expose the introspection report, then evaluate a cleanup ticket for reducing CoinVault's custom resolver.

## Practical debugging commands today

Until `--print-profiles` exists, these are useful manual probes.

List CoinVault events/profiles indirectly through SQLite or HTTP once running:

```bash
curl http://localhost:8080/api/chat/profiles
```

Inspect a local registry file without printing secrets:

```bash
yq '.profiles | keys' profile-registry.local.yaml
```

Inspect Geppetto source parsing behavior in tests:

```bash
go test ./pkg/engineprofiles -run 'Source|Chain|Resolve' -count=1
```

Inspect CoinVault profile settings tests:

```bash
cd 2026-03-16--gec-rag
go test ./cmd/coinvault/cmds -run ProfileSettings -count=1
```

## Final recommendation

For the intern implementing this feature:

1. Start in Geppetto with reusable report structs and renderers.
2. Build the report from a generic `gepprofiles.Registry` plus source/default/selection metadata.
3. Add a bootstrap wrapper for apps that use Geppetto `AppBootstrapConfig`.
4. Add a Pinocchio wrapper that includes inline config profiles.
5. Add a CoinVault bridge that reuses CoinVault's existing `resolveProfileSettings` and `OpenInferenceProfiles` but uses the central Geppetto report renderer.
6. Wire `--print-profiles` as an early-exit path before database checks and before LLM calls.
7. Redact secrets in all output.

This path improves user-visible debugging immediately while keeping the core profile-introspection logic centralized in Geppetto.
