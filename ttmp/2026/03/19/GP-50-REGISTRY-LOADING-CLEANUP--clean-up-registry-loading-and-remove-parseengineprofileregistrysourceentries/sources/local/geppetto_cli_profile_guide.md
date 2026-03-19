---
Title: Imported Source - Geppetto CLI Profile Guide
Ticket: GP-50-REGISTRY-LOADING-CLEANUP
Status: active
Topics:
    - profiles
    - glazed
    - cleanup
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: Loaded-command runtime path discussed extensively in the imported guide
    - Path: pinocchio/pkg/cmds/helpers/profile_runtime.go
      Note: Thin bootstrap/runtime helper path contrasted against loaded commands
    - Path: pinocchio/pkg/cmds/loader.go
      Note: Historical loader behavior and default propagation issues analyzed in the guide
ExternalSources:
    - /home/manuel/Downloads/geppetto_cli_profile_guide.md
Summary: Imported external design guide that separates baseline CLI config, engine profile registry overlays, and deferred runtime-profile concerns for Pinocchio and Geppetto command bootstrapping.
LastUpdated: 2026-03-19T10:25:00-04:00
WhatFor: Preserve the imported proposal that now serves as the preferred basis for simplifying CLI config loading and engine profile overlay behavior.
WhenToUse: Use when planning or implementing shared Geppetto-backed CLI bootstrap helpers, especially for separating baseline config from engine profile registries.
---

# Pinocchio / Geppetto CLI profile-loading architecture guide

## Purpose

This guide explains how the current Pinocchio + Geppetto + Glazed command stack works, why it feels confusing, how it worked historically, and how to implement a cleaner, reusable pattern for **all CLI commands that use Geppetto**.

The target reader is a new intern who needs to understand:

- how Glazed parsing works,
- how Pinocchio repository-loaded commands are built,
- how Geppetto sections become `InferenceSettings`,
- how engine profile registries are selected and resolved,
- how chat-time profile switching works,
- and how to build a simpler API for “give me the selected engine quickly”.

---

## Executive summary

The current system has **three different layers of configuration** that are easy to conflate.

First, there is a **baseline CLI/app config layer** loaded through Glazed sections and middlewares. That is where config files, environment variables, defaults, and flags feed values such as `ai-chat.ai-engine`, `openai-chat.openai-api-key`, or `ai-inference.reasoning-effort` into parsed command values.

Second, there is an **engine profile registry layer**. This is the Geppetto engine-profile system in `geppetto/pkg/engineprofiles`. A selected profile slug from `profile-settings.profile` and a registry-source list from `profile-settings.profile-registries` resolve to a final `InferenceSettings` overlay. This is now an **engine-only** concept. It should decide engine/provider/model/client settings, not tools, prompts, or middleware behavior.

Third, there is an **application runtime layer**. That covers Pinocchio-specific concerns such as prompts, tool registries, middleware selection, system prompt injection, and interactive chat behavior. This layer is currently spread across Pinocchio code, not owned by Geppetto profiles.

The biggest source of confusion is that there are currently **multiple entry-point styles** doing similar work in different ways:

- repository-loaded YAML commands go through `pinocchio/pkg/cmds/loader.go`, `pinocchio/pkg/cmds/cobra.go`, and `pinocchio/pkg/cmds/cmd.go`
- JS and some agent/demo entry points use `pinocchio/pkg/cmds/helpers/profile_runtime.go`
- some older examples call `factory.NewEngineFromParsedValues(...)`, which is fast but **not profile-registry-aware**

The clean long-term model should be:

```text
baseline config/defaults/env/flags
  -> base InferenceSettings
  -> optional engine profile registry overlay
  -> final InferenceSettings
  -> engine

separate Pinocchio runtime config
  -> tools + middleware + prompts + UI behavior
```

That means the reusable abstraction you want is not just “parse Geppetto sections”, but:

1. build or obtain a **baseline** `InferenceSettings`
2. resolve `profile` + `profile-registries`
3. overlay the selected engine profile onto the baseline
4. expose helpers to build an engine from the final settings
5. optionally keep the baseline around for runtime profile switching in chat

---

## The short mental model

If you only remember one picture, remember this one.

```text
Glazed command schema
  ├─ default section (command args/flags for prompt variables)
  ├─ pinocchio helper section (chat/output/UI knobs)
  ├─ geppetto inference sections
  │    ├─ ai-chat
  │    ├─ ai-client
  │    ├─ openai-chat / claude-chat / gemini-chat / embeddings / ai-inference
  │    └─ profile-settings
  └─ parsed values
       ├─ defaults
       ├─ config files
       ├─ env vars
       ├─ args / cobra flags
       └─ provenance log per field

parsed values
  -> base InferenceSettings
  -> resolve selected engine profile
  -> merge profile onto base
  -> final InferenceSettings
  -> engine factory
  -> run / chat / JS / agent
```

---

## Core vocabulary

### Glazed section

A **section** is a named group of fields in a command schema. Examples:

- `ai-chat`
- `openai-chat`
- `profile-settings`
- `geppetto-helpers`

Relevant APIs:

- `glazed/pkg/cmds/schema/schema.go`
- `glazed/pkg/cmds/values/section-values.go`

### Glazed middleware

A **middleware** is a parser source or transformer that updates parsed values. Typical sources are:

- defaults
- config files
- env vars
- args
- Cobra flags

Important detail: middlewares are **executed in reverse order** of the list you pass to `sources.Execute(...)`, which is why code typically lists high-precedence sources first and defaults last. See `glazed/pkg/cmds/sources/middlewares.go:37-67`.

### Parsed values

Parsed values are the per-section field values after Glazed middlewares run. They are stored in `*values.Values` and can be decoded into structs section-by-section with `DecodeSectionInto(...)`. See `glazed/pkg/cmds/values/section-values.go:246-260`.

### Parse provenance

Each field value stores a log of where it came from. This is why Pinocchio can later strip profile-derived values and recompute a baseline. See `glazed/pkg/cmds/fields/field-value.go:11-17`, `:67-83`, and `:113-118`.

### InferenceSettings

`InferenceSettings` is the in-memory object Geppetto uses to create engines. It combines:

- provider/API settings
- chat/model settings
- client settings
- embeddings settings
- inference-time settings

See `geppetto/pkg/steps/ai/settings/settings-inference.go:53-68`.

### Engine profile registry

An engine profile registry is a named set of engine profiles. A profile resolves to a merged `InferenceSettings`. See:

- `geppetto/pkg/engineprofiles/types.go:34-53`
- `geppetto/pkg/engineprofiles/registry.go:17-29`
- `geppetto/pkg/engineprofiles/service.go:105-145`

### Base settings vs final settings

This distinction matters a lot.

- **Base settings**: what you get from defaults + config + env + flags before applying a selected engine profile
- **Final settings**: base settings plus the selected engine profile overlay

Chat profile switching needs both. The active chat session uses final settings, but `/profile` switching needs the original baseline so it can re-apply a different profile later.

---

## The files that matter most

### Pinocchio

- `pinocchio/pkg/cmds/loader.go`
  - loads YAML-defined Pinocchio commands
  - adds Geppetto sections to command descriptions
  - historically felt like the “magic” entry point

- `pinocchio/pkg/cmds/cobra.go`
  - builds Cobra commands with Geppetto middlewares
  - uses `geppetto/pkg/sections.GetCobraCommandGeppettoMiddlewares`

- `pinocchio/pkg/cmds/cmd.go`
  - runtime execution path for a `PinocchioCommand`
  - decodes helpers, resolves settings, resolves profile registries, creates engines, runs chat

- `pinocchio/pkg/cmds/profile_base_settings.go`
  - recomputes a profile-free baseline from parsed values
  - important for chat-time profile switching

- `pinocchio/pkg/cmds/helpers/profile_runtime.go`
  - newer shared helper for “base settings + profile selection + final settings”
  - currently used in JS and some non-loaded command paths

- `pinocchio/cmd/pinocchio/main.go`
  - root command setup
  - repository loading
  - `run-command file.yaml`
  - JS subcommand wiring

### Geppetto

- `geppetto/pkg/sections/sections.go`
  - creates Geppetto settings sections
  - defines the Cobra middleware constructor for Geppetto-enabled commands

- `geppetto/pkg/sections/profile_sections.go`
  - profile-settings section and a near-duplicate profile middleware helper

- `geppetto/pkg/engineprofiles/*`
  - engine registry types, loaders, stack expansion, merge rules, source chaining

- `geppetto/pkg/inference/engine/factory/*`
  - creates engines from `InferenceSettings`

### Glazed

- `glazed/pkg/cmds/sources/middlewares.go`
  - middleware execution model

- `glazed/pkg/cmds/fields/field-value.go`
  - per-field parse provenance

- `glazed/pkg/config/resolve.go`
  - app config file discovery

- `glazed/pkg/appconfig/options.go`
  - old Glazed `profiles.yaml` bootstrap helper (`WithProfile(...)`)
  - important historical reference for the “bootstrap parse before profile middleware” pattern

---

## Current repository-loaded command flow

This is the path the user usually perceives as “the Pinocchio command system”.

### Step 1: YAML command loading

The loader reads a YAML prompt/command file and turns it into a `PinocchioCommandDescription`. See `pinocchio/pkg/cmds/loader.go:37-62`.

Then it reads the same YAML again with `settings.NewInferenceSettingsFromYAML(...)` and uses the result as defaults when constructing Geppetto sections. See `pinocchio/pkg/cmds/loader.go:65-73`.

Then it prepends the Pinocchio helper section and builds a `PinocchioCommand`. See `pinocchio/pkg/cmds/loader.go:74-106`.

#### Important observation

This is the historical reason loaded commands “felt special”: the command loader did not just load prompt text. It also tried to seed Geppetto settings defaults directly from the command YAML.

### Step 2: Cobra command creation

`pinocchio/pkg/cmds/cobra.go:12-21` builds the Cobra command and tells Glazed to use `geppetto/pkg/sections.GetCobraCommandGeppettoMiddlewares` for parsing.

That means every loaded command gets:

- standard Geppetto settings sections
- profile-settings section
- the Geppetto config/env/flag parsing behavior

### Step 3: Bootstrap parsing before the full parse

`geppetto/pkg/sections/sections.go:130-299` does a bootstrap parse before returning the main middleware chain.

It first bootstraps `command-settings` to discover config files. See `sections.go:176-210`.

Then it bootstraps `profile-settings` so `profile` and `profile-registries` can come from:

- defaults
- config files
- environment variables
- Cobra flags

See `sections.go:212-252`.

This bootstrap parse exists because of an older class of bugs: profile selection cannot be resolved correctly if you instantiate profile-aware logic before env/config/flags are applied.

### Step 4: Main parse into `parsedValues`

The middleware chain is returned in precedence order and later executed via Glazed. See:

- `geppetto/pkg/sections/sections.go:254-298`
- `glazed/pkg/cmds/sources/middlewares.go:37-67`

Operationally, the main effective precedence is:

```text
defaults < config files < env < args < cobra flags
```

### Step 5: `RunIntoWriter` turns parsed values into settings

Inside `pinocchio/pkg/cmds/cmd.go`, `RunIntoWriter(...)` does the main runtime work.

First it decodes the helper/UI settings from `geppetto-helpers`. See `cmd.go:206-211`.

Then it creates a fresh `InferenceSettings` and updates it from parsed values. See `cmd.go:213-221`.

Then it resolves `profile` and `profile-registries` from the `profile-settings` section, with compatibility fallbacks to the default section and environment variables. See `cmd.go:223-250`.

Then it computes a baseline using `baseSettingsFromParsedValues(...)`. See `cmd.go:252-258` and `profile_base_settings.go:17-77`.

Then, if profile registries are configured, it creates a profile-switch manager, resolves the selected engine profile, and replaces `stepSettings` with the resolved merged settings. See `cmd.go:259-271`.

### Step 6: Engine creation

Actual engine creation happens later in `runEngineAndCollectMessages(...)`:

- `engine, err := rc.EngineFactory.CreateEngine(rc.InferenceSettings)`
- file: `pinocchio/pkg/cmds/cmd.go:453-489`

The engine factory itself chooses a provider based on `settings.Chat.ApiType`. See `geppetto/pkg/inference/engine/factory/factory.go:47-94`.

### Step 7: Chat-time profile switching

If chat mode is active and profile registries exist, Pinocchio builds a `profileswitch.Manager` from the **baseline settings** plus the registry sources. See `pinocchio/pkg/cmds/cmd.go:608-753` and `pinocchio/pkg/ui/profileswitch/manager.go:17-39`, `:118-178`.

That is why the code needs both:

- a final resolved settings object for the initial run
- a profile-free baseline for future `/profile` switches

---

## ASCII diagram: current loaded-command flow

```text
YAML command file
  |
  |  pinocchio/pkg/cmds/loader.go
  v
CommandDescription + Geppetto sections + helper section
  |
  |  pinocchio/pkg/cmds/cobra.go
  v
Cobra command with GetCobraCommandGeppettoMiddlewares
  |
  |  geppetto/pkg/sections/sections.go
  |    - bootstrap command-settings
  |    - bootstrap profile-settings
  |    - return full middleware chain
  v
parsedValues (*values.Values)
  |
  |  pinocchio/pkg/cmds/cmd.go / RunIntoWriter
  |    - decode helper settings
  |    - decode stepSettings from parsedValues
  |    - compute baseSettings
  |    - resolve selected engine profile
  |    - merge onto baseSettings
  v
final InferenceSettings
  |
  |  engine factory
  v
engine
  |
  v
run / chat / interactive session
```

---

## How it worked in the past

There are two different “past systems” that are easy to mix up.

### Past system 1: loader-based command defaults

Historically, repository-loaded commands looked like they “already knew” their Geppetto settings because the loader itself tried to derive those defaults from the command YAML. That behavior is still visible in `pinocchio/pkg/cmds/loader.go:65-73`.

That is what your message is pointing at when you say “this used to work with `pinocchio/pkg/cmds/loader.go` loaded commands”.

The mental model was:

```text
command YAML
  -> prompt + flags + arguments
  -> maybe some embedded inference defaults
  -> Geppetto sections initialized from those defaults
```

That made loaded commands feel more self-contained than ad hoc Cobra commands.

### Past system 2: old Glazed `profiles.yaml` middleware

Glazed used to have a classic profile system based on:

- `profile-settings.profile`
- `profile-settings.profile-file`
- `GatherFlagsFromProfiles(...)`

That system loaded a file shaped like:

```yaml
development:
  ai-chat:
    ai-engine: gpt-4o-mini
  openai-chat:
    openai-api-key: ...
```

and applied it as a mid-precedence bundle of field overrides.

The relevant code is:

- `glazed/pkg/cmds/sources/profiles.go:13-80`
- `glazed/pkg/appconfig/options.go:174-315`

### Why the old profile system broke

The old system had a circularity problem:

- profile selection itself was parsed through Glazed
- but profile middleware needed to know the selected profile before it could run
- if the middleware was instantiated too early, it captured only default values

This issue is documented clearly in:

- `pinocchio/ttmp/2025/12/15/IMPROVE-PROFILES-001--fix-profile-system-interdependency-issues/analysis/01-profile-system-interdependency-health-inspection.md`

The fix was the bootstrap-parse pattern:

1. pre-parse enough settings to know the config file and selected profile
2. then build the real middleware chain

### What changed recently

Geppetto later made a hard architectural split:

- engine profiles are now **engine-only**
- runtime behavior no longer belongs in Geppetto profile resolution

See:

- `geppetto/pkg/doc/topics/01-profiles.md`
- `geppetto/pkg/engineprofiles/*`

This changed the meaning of “profile” substantially.

The new profile system is now about:

```text
profile slug + registry stack
  -> resolved InferenceSettings
  -> engine
```

not:

```text
profile slug
  -> prompt + tools + middleware + system prompt + engine settings
```

That is the right architectural direction, but it created a transition period where:

- loaded commands still felt loader-centric
- JS/helper paths introduced new resolution helpers
- some code still expected old-style profile semantics

---

## The most important source of confusion: there are three different YAML/config shapes

This is the single most useful thing to explain to a new intern.

### Shape 1: app config file (`config.yaml`)

This is the Glazed config-file shape used by `sources.FromFiles(...)` plus the config mapper.

Example:

```yaml
profile-settings:
  profile: default
  profile-registries:
    - ~/.config/pinocchio/profiles.yaml

ai-chat:
  ai-api-type: openai
  ai-engine: gpt-5-mini

ai-client:
  timeout: 30s

openai-chat:
  openai-api-key: ${OPENAI_API_KEY}

ai-inference:
  inference-reasoning-effort: medium
```

This shape is section-based. The keys are **Glazed section slugs**, and the nested keys are **field names**.

Relevant code:

- config discovery: `glazed/pkg/config/resolve.go:9-47`
- config mapping: `geppetto/pkg/sections/sections.go:138-163`
- duplicate helper mapper: `pinocchio/pkg/cmds/helpers/profile_runtime.go:238-259`

### Shape 2: engine profile registry (`profiles.yaml`)

This is the engine-profile registry shape used by `geppetto/pkg/engineprofiles`.

Example:

```yaml
slug: workspace
profiles:
  default:
    slug: default
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-5-mini
      inference:
        reasoning_effort: medium
```

This is **not** the same as the config-file shape.

Relevant code:

- `geppetto/pkg/engineprofiles/types.go:34-53`
- `geppetto/pkg/engineprofiles/service.go:105-145`
- `geppetto/pkg/engineprofiles/source_chain.go:46-75`

### Shape 3: loader-embedded inference defaults in command YAML

`pinocchio/pkg/cmds/loader.go` calls `settings.NewInferenceSettingsFromYAML(...)` on the whole command YAML file. That means loader-embedded defaults use the `InferenceSettings` YAML shape, not the Glazed config shape.

That shape looks more like:

```yaml
chat:
  api_type: openai
  engine: gpt-5-mini
client:
  timeout_second: 30
openai:
  api_key: ...
inference:
  reasoning_effort: medium
```

Relevant code:

- `pinocchio/pkg/cmds/loader.go:65-73`
- `geppetto/pkg/steps/ai/settings/settings-inference.go:116-129`

### Why this matters

A new developer can easily assume that one file shape is accepted everywhere. It is not.

Right now, the system mixes:

- section-based Glazed config files
- engine-profile registry files
- loader-side inference-settings parsing

That alone explains a large fraction of the confusion.

---

## Current pain points and design smells

### 1. The same conceptual work exists in multiple places

There are at least three distinct bootstrap patterns in active use:

- loaded commands: `loader.go` + `cmd.go`
- JS and agents: `pkg/cmds/helpers/profile_runtime.go`
- some examples: direct `factory.NewEngineFromParsedValues(...)`

That means there is no single “this is how Pinocchio resolves an engine” function.

### 2. `factory.NewEngineFromParsedValues(...)` is not enough for registry-based profiles

`geppetto/pkg/inference/engine/factory/helpers.go:9-14` is a convenient helper, but it only decodes `InferenceSettings` from parsed sections. It does **not** resolve engine profile registries.

So for your desired API, this helper is too low-level.

### 3. `PinocchioCommand.RunIntoWriter(...)` and `profile_runtime.go` solve similar problems differently

`RunIntoWriter(...)` resolves from already-parsed command values, which is good because it preserves command-local defaults.

`ResolveBaseInferenceSettings(...)` in `profile_runtime.go` re-parses hidden Geppetto sections from env/config/defaults, which is useful for JS/agents, but does not naturally preserve loader-provided command-local defaults.

This difference is subtle and important.

### 4. Registry-required vs registry-optional behavior is inconsistent

`geppetto/pkg/sections/sections.go:245-251` currently errors if no `profile-registries` are configured.

But `pinocchio/pkg/cmds/helpers/profile_runtime.go:127-131` allows “no registry” and simply returns the baseline settings.

That is an inconsistency in behavior.

If your target architecture is “baseline config file plus optional engine profile overlay”, the parser-side hard error is probably too strict.

### 5. The loader re-parses the whole YAML file to discover settings defaults

This is a legacy-looking pattern in `pinocchio/pkg/cmds/loader.go:65-73`.

It works as a convenient hack, but it has problems:

- it is not obvious to readers
- it creates a third file shape
- it couples unrelated YAML fields to inference parsing
- it makes the command description format harder to reason about

### 6. Helper section prepending is redundant

The helper section is added in both:

- `pinocchio/pkg/cmds/loader.go:74-80`
- `pinocchio/pkg/cmds/cmd.go:176-195`

Because schema sections are keyed by slug, this is mostly harmless, but it is still redundant and adds cognitive noise.

### 7. Default path conventions are not fully aligned

Config-file discovery supports:

- `$XDG_CONFIG_HOME/pinocchio/config.yaml`
- `$HOME/.pinocchio/config.yaml`
- `/etc/pinocchio/config.yaml`

See `glazed/pkg/config/resolve.go:9-47`.

Default engine-profile registry discovery only checks:

- `$XDG_CONFIG_HOME/pinocchio/profiles.yaml`

See `geppetto/pkg/sections/profile_sections.go:55-66`.

That mismatch contributes to confusion, especially because users often think in terms of `~/.pinocchio/...`.

---

## What `cmd.go` is really doing

This is worth spelling out because it is the file you explicitly asked about.

### `NewPinocchioCommand(...)`

File: `pinocchio/pkg/cmds/cmd.go:176-195`

It does one thing of note: it prepends the helper section. It does **not** add Geppetto sections. Those must already be present on the description.

That means Geppetto readiness is not guaranteed by `NewPinocchioCommand(...)` itself.

### `RunIntoWriter(...)`

File: `pinocchio/pkg/cmds/cmd.go:200-340`

This is the real “resolved runtime bootstrap” function for loaded commands.

Conceptually it does:

```text
parsed values
  -> decode helper settings
  -> decode raw inference settings
  -> decode profile selection
  -> compute profile-free baseline
  -> resolve selected engine profile
  -> final settings
  -> RunWithOptions(...)
```

The key engine-profile resolution block is `cmd.go:223-271`.

### `runEngineAndCollectMessages(...)`

File: `pinocchio/pkg/cmds/cmd.go:453-489`

This is where the engine is actually created from the resolved `InferenceSettings`.

### `runChat(...)`

File: `pinocchio/pkg/cmds/cmd.go:491-840`

This is where baseline settings become important again, because interactive profile switching uses:

```text
base settings + profile registry manager -> new final settings per switch
```

The profile-switch path is `cmd.go:608-753`.

---

## What the current helper package is already doing

`pinocchio/pkg/cmds/helpers/profile_runtime.go` is important because it is already very close to the reusable abstraction you want.

It provides:

- `ResolveBaseInferenceSettings(...)` — build baseline settings from config/env/defaults
- `ResolveEngineProfileSettings(...)` — resolve `profile` and `profile-registries`
- `ResolveFinalInferenceSettings(...)` — resolve the selected engine profile and merge it onto the baseline

Relevant lines:

- `profile_runtime.go:37-65`
- `profile_runtime.go:83-113`
- `profile_runtime.go:115-170`

### Why it is not yet the complete answer

Because it is a **hidden reparse** helper.

That is correct for:

- JS commands that do not already have full Geppetto sections in a Glazed command schema
- agent/demo entry points that only have `config-file`, `profile`, and `profile-registries`

But it is not a perfect replacement for `RunIntoWriter(...)` because loaded commands may have command-local defaults already present in parsed values. A hidden reparse would ignore those unless you explicitly carry them in.

So the long-term abstraction needs **two entry points**, not one.

---

## Recommended target design

The clean architecture is:

```text
Layer A: baseline config
  - config file
  - env
  - command defaults
  - flags
  -> base InferenceSettings

Layer B: engine profile selection
  - profile
  - profile-registries
  -> resolved engine profile
  -> final InferenceSettings

Layer C: Pinocchio runtime policy
  - prompt/system prompt
  - tools
  - middlewares
  - chat/UI behavior
  -> runtime/session assembly
```

### Design rule 1

Keep **engine profiles engine-only**.

That matches the Geppetto direction documented in `geppetto/pkg/doc/topics/01-profiles.md`.

### Design rule 2

Do not use `profiles.yaml` for two unrelated meanings.

If you want a baseline app config file, do **not** make it the same conceptual file as the engine-profile registry.

Recommended naming:

- baseline app config: `config.yaml`
- engine-profile registry: `profiles.yaml`

If you want a legacy home-dir fallback, use:

- `~/.pinocchio/config.yaml`
- `~/.pinocchio/profiles.yaml`

but keep the **semantics** separate.

### Design rule 3

Add a single explicit Pinocchio-owned “resolve final engine settings” API.

Geppetto should remain generic.

Pinocchio is the right place to own:

- app config path discovery
- command-local defaults preservation
- profile selection bootstrap
- final engine creation convenience helpers

### Design rule 4

Expose both base settings and final settings.

This is required for chat profile switching and good debugging.

---

## Proposed reusable API

A practical package could live in:

```text
pinocchio/pkg/cmds/enginebootstrap
```

or

```text
pinocchio/pkg/cmds/helpers
```

with clearer naming.

### Proposed result type

```go
type ResolvedCLIEngine struct {
    BaseSettings       *settings.InferenceSettings
    FinalSettings      *settings.InferenceSettings
    Profile            string
    ProfileRegistries  []string
    ResolvedProfile    *engineprofiles.ResolvedEngineProfile
    ConfigFiles        []string
    Close              func()
}
```

### Proposed helper 1: for already-parsed commands

Use this for repository-loaded commands and any Glazed command that already has Geppetto sections in its schema.

```go
func ResolveCLIEngineFromParsedValues(
    ctx context.Context,
    parsed *values.Values,
) (*ResolvedCLIEngine, error)
```

This function should:

1. decode base settings from **the already-parsed values**
2. resolve `profile` + `profile-registries`
3. compute `BaseSettings`
4. resolve and merge the selected engine profile
5. return both base and final settings

This preserves command-local defaults.

### Proposed helper 2: for lightweight/manual entry points

Use this for JS commands, agents, or small utilities that only have `config-file` / `profile` / `profile-registries` but no full Geppetto schema.

```go
func ResolveCLIEngineFromBootstrap(
    ctx context.Context,
    parsed *values.Values,
) (*ResolvedCLIEngine, error)
```

This is essentially a renamed and generalized version of today’s `ResolveFinalInferenceSettings(...)` helper.

### Proposed helper 3: fast engine creation

```go
func (r *ResolvedCLIEngine) NewEngine() (engine.Engine, error) {
    return enginefactory.NewEngineFromSettings(r.FinalSettings)
}
```

This is the convenience API you are explicitly asking for.

### Proposed helper 4: selection-only resolution

```go
type ProfileSelection struct {
    Profile           string
    ProfileRegistries []string
    ConfigFiles       []string
}

func ResolveProfileSelection(parsed *values.Values) (ProfileSelection, error)
```

This should be the one place that knows:

- default registry discovery
- config file discovery
- env fallbacks
- compatibility behavior

That logic is currently duplicated across:

- `geppetto/pkg/sections/sections.go`
- `geppetto/pkg/sections/profile_sections.go`
- `pinocchio/pkg/cmds/cmd.go`
- `pinocchio/pkg/cmds/helpers/profile_runtime.go`

---

## Pseudocode for the recommended implementation

### Parsed-values path

```go
func ResolveCLIEngineFromParsedValues(
    ctx context.Context,
    parsed *values.Values,
) (*ResolvedCLIEngine, error) {
    if parsed == nil {
        return nil, errors.New("parsed values are required")
    }

    // 1. final-looking settings from current parsed values
    raw, err := settings.NewInferenceSettings()
    if err != nil {
        return nil, err
    }
    if err := raw.UpdateFromParsedValues(parsed); err != nil {
        return nil, err
    }

    // 2. profile-free baseline for chat switching / merge base
    base, err := baseSettingsFromParsedValues(parsed)
    if err != nil {
        return nil, err
    }

    // 3. profile selection
    sel, err := ResolveProfileSelection(parsed)
    if err != nil {
        return nil, err
    }

    // 4. no registry => base/final are the same
    if len(sel.ProfileRegistries) == 0 {
        return &ResolvedCLIEngine{
            BaseSettings:      base,
            FinalSettings:     raw,
            Profile:           sel.Profile,
            ProfileRegistries: nil,
        }, nil
    }

    // 5. registry resolution
    specs, err := engineprofiles.ParseRegistrySourceSpecs(sel.ProfileRegistries)
    if err != nil {
        return nil, err
    }
    chain, err := engineprofiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
    if err != nil {
        return nil, err
    }

    input := engineprofiles.ResolveInput{}
    if sel.Profile != "" {
        input.EngineProfileSlug = engineprofiles.MustEngineProfileSlug(sel.Profile)
    }

    resolved, err := chain.ResolveEngineProfile(ctx, input)
    if err != nil {
        _ = chain.Close()
        return nil, err
    }

    final, err := engineprofiles.MergeInferenceSettings(base, resolved.InferenceSettings)
    if err != nil {
        _ = chain.Close()
        return nil, err
    }

    return &ResolvedCLIEngine{
        BaseSettings:      base,
        FinalSettings:     final,
        Profile:           sel.Profile,
        ProfileRegistries: append([]string(nil), sel.ProfileRegistries...),
        ResolvedProfile:   resolved,
        Close: func() {
            _ = chain.Close()
        },
    }, nil
}
```

### Bootstrap path

```go
func ResolveCLIEngineFromBootstrap(
    ctx context.Context,
    parsed *values.Values,
) (*ResolvedCLIEngine, error) {
    // Reuse today’s hidden-bootstrap logic:
    // - resolve config files
    // - build hidden Geppetto sections
    // - parse defaults/config/env
    // - resolve profile selection
    // - merge selected engine profile
}
```

### Engine creation helper

```go
func NewEngineFromResolvedCLIEngine(
    resolved *ResolvedCLIEngine,
) (engine.Engine, error) {
    if resolved == nil || resolved.FinalSettings == nil {
        return nil, errors.New("resolved engine settings missing")
    }
    return enginefactory.NewEngineFromSettings(resolved.FinalSettings)
}
```

---

## What should change in `PinocchioCommand.RunIntoWriter(...)`

Today it contains inline engine/bootstrap logic.

Recommended refactor:

1. keep helper/UI decoding in `RunIntoWriter(...)`
2. move engine/bootstrap resolution into the new shared helper
3. keep only command execution and run-mode assembly in `RunIntoWriter(...)`

The code should become conceptually:

```go
func (g *PinocchioCommand) RunIntoWriter(...) error {
    helpers := decodeHelpers(parsedValues)

    resolved, err := enginebootstrap.ResolveCLIEngineFromParsedValues(ctx, parsedValues)
    if err != nil {
        return err
    }
    if resolved.Close != nil {
        defer resolved.Close()
    }

    _, err = g.RunWithOptions(ctx,
        run.WithInferenceSettings(resolved.FinalSettings),
        run.WithBaseSettings(resolved.BaseSettings),
        run.WithProfileSelection(resolved.Profile, strings.Join(resolved.ProfileRegistries, ",")),
        ...
    )
    return err
}
```

That would make `cmd.go` much easier to reason about.

---

## What should change in the loader

The loader is doing something clever but not explicit enough.

### Current behavior

`pinocchio/pkg/cmds/loader.go:65-73` re-parses the whole YAML file using `settings.NewInferenceSettingsFromYAML(...)` and uses that as Geppetto section defaults.

### Recommended behavior

Make command-local inference defaults an **explicit field** in the command YAML.

For example:

```yaml
name: my-command
short: demo
prompt: |
  Say hello to {{ .name }}

inference-defaults:
  chat:
    api_type: openai
    engine: gpt-5-mini
  inference:
    reasoning_effort: medium
```

and then in Go:

```go
type PinocchioCommandDescription struct {
    Name              string                       `yaml:"name"`
    Short             string                       `yaml:"short"`
    Prompt            string                       `yaml:"prompt,omitempty"`
    InferenceDefaults *settings.InferenceSettings  `yaml:"inference-defaults,omitempty"`
}
```

Then the loader would use that field directly instead of re-parsing the whole file.

This removes a lot of invisible behavior.

### Alternative if you want section-shaped defaults instead

You could also define:

```yaml
geppetto-defaults:
  ai-chat:
    ai-api-type: openai
    ai-engine: gpt-5-mini
  ai-inference:
    inference-reasoning-effort: medium
```

That matches config-file shape more closely.

Either option is better than “decode the entire command YAML again and hope inference defaults fall out of it”.

---

## What to do about `~/.pinocchio/profiles.yaml`

Your prompt specifically mentions:

```text
load a basic config file (per default the ~/.pinocchio/profiles.yaml)
```

I strongly recommend **not** using that exact meaning unless you are intentionally redefining the current architecture.

### Why not

In the current codebase, `profiles.yaml` already means **engine profile registry**. See:

- `geppetto/pkg/sections/profile_sections.go:55-66`
- `pinocchio/README.md` engine profile docs

If you also use `profiles.yaml` as a baseline config file for section-based settings, you are overloading the filename with two incompatible schemas.

### Recommended file semantics

Use:

```text
~/.config/pinocchio/config.yaml
~/.pinocchio/config.yaml           (legacy fallback)
```

for baseline config, and keep:

```text
~/.config/pinocchio/profiles.yaml
~/.pinocchio/profiles.yaml         (optional legacy fallback if you add one)
```

for engine-profile registries.

### Recommended search order

For config file:

1. `$XDG_CONFIG_HOME/pinocchio/config.yaml`
2. `$HOME/.pinocchio/config.yaml`
3. `/etc/pinocchio/config.yaml`

For profile registries:

1. explicit `--profile-registries`
2. `PINOCCHIO_PROFILE_REGISTRIES`
3. `profile-settings.profile-registries` from config
4. `$XDG_CONFIG_HOME/pinocchio/profiles.yaml`
5. optional legacy `$HOME/.pinocchio/profiles.yaml`

That would align config and registry behavior much better.

---

## How to handle profiles that configure middlewares and tools

This is where it is easy to accidentally reintroduce the old confusion.

### Recommendation: do not overload engine profiles for runtime policy

The current Geppetto direction is clear:

- engine profiles decide engine settings
- application code decides runtime behavior

That is the cleanest separation.

### Best option: separate Pinocchio runtime profiles

If you want profiles for:

- tools
- middleware chains
- system prompts
- starter prompts
- output modes
- web/chat UX behavior

then create a **separate Pinocchio-owned runtime profile layer**.

For example:

```yaml
runtime-profiles:
  assistant:
    system-prompt: |
      You are a careful coding assistant.
    tools:
      include:
        - calculator
        - shell
        - search
    middlewares:
      - tool-result-reorder
      - tracing
```

That can live in:

- `runtime-profiles.yaml`
- `runtime-settings.*` section in config
- or a dedicated Pinocchio runtime registry package

### Why this is better

It keeps the mental model clean:

```text
engine profile -> what engine to build
runtime profile -> how the app session should behave
```

### Alternative: use engine profile `Extensions`

Technically, `EngineProfile` has an `Extensions` field. See `geppetto/pkg/engineprofiles/types.go:35-43`. Stack merging for extensions also exists in `geppetto/pkg/engineprofiles/stack_merge.go:10-42`.

But there is an important limitation:

- current `ResolvedEngineProfile` does **not** expose merged extensions in its public result shape
- `StoreRegistry.ResolveEngineProfile(...)` currently returns merged `InferenceSettings` plus metadata only

See `geppetto/pkg/engineprofiles/service.go:127-145`.

So if you want runtime/tool/middleware settings to come out of engine profile resolution, you would need to extend the public API.

### Practical advice

Use a separate Pinocchio runtime profile system unless you intentionally want to widen the Geppetto profile API again.

---

## Suggested implementation phases

### Phase 1: centralize profile selection

Create one helper that resolves:

- config files
- `profile`
- `profile-registries`
- defaults/fallbacks

and delete duplicated logic from:

- `pinocchio/pkg/cmds/cmd.go`
- `pinocchio/pkg/cmds/helpers/profile_runtime.go`
- `geppetto/pkg/sections/profile_sections.go`
- `geppetto/pkg/sections/sections.go`

### Phase 2: centralize final engine settings resolution

Introduce:

- `ResolveCLIEngineFromParsedValues(...)`
- `ResolveCLIEngineFromBootstrap(...)`

These should be the only two supported pathways.

### Phase 3: centralize engine creation

Add a helper that takes the resolved result and calls `enginefactory.NewEngineFromSettings(...)`.

Then stop using `factory.NewEngineFromParsedValues(...)` in any code path that should honor engine profiles.

### Phase 4: refactor `RunIntoWriter(...)`

Move engine bootstrap logic out of `cmd.go` and keep only execution logic there.

### Phase 5: make loader defaults explicit

Replace the implicit “parse whole command YAML twice” logic with an explicit field such as `inference-defaults`.

### Phase 6: add runtime profiles if needed

Only after engine profiles are clean should you add a second runtime-profile system for tools and middleware.

---

## Tests you should add

### 1. Loaded command preserves command-local defaults

Create a YAML-loaded command with local default engine settings and verify the resolved base/final settings include them.

### 2. Config-only path works without registries

Verify a command can run from `config.yaml` alone when no profile registry is configured, assuming baseline settings contain enough engine info.

### 3. Registry overlay wins over baseline

Baseline says `gpt-4o-mini`; selected engine profile says `gpt-5-mini`; verify final engine is `gpt-5-mini`.

### 4. CLI flag wins over config-selected profile

Config says `profile: default`; CLI flag says `--profile fast`; verify `fast` is used.

### 5. Chat switching reuses baseline

Start with one profile, switch to another, then verify the second profile overlays onto the same baseline rather than onto the first profile’s already-merged settings.

### 6. JS/bootstrap path matches loaded-command semantics

As far as possible, verify the JS helper path resolves the same final engine settings as a full loaded command when given equivalent config/profile inputs.

### 7. Legacy path fallback tests

If you add `~/.pinocchio/profiles.yaml` as a fallback registry location, test both XDG and legacy home-dir paths.

---

## A concrete “good future state” example

### Baseline config: `~/.config/pinocchio/config.yaml`

```yaml
profile-settings:
  profile: default
  profile-registries:
    - ~/.config/pinocchio/profiles.yaml

ai-chat:
  ai-api-type: openai
  ai-engine: gpt-4o-mini

ai-client:
  timeout: 30s

openai-chat:
  openai-api-key: ${OPENAI_API_KEY}

ai-inference:
  inference-reasoning-effort: low
```

### Engine registry: `~/.config/pinocchio/profiles.yaml`

```yaml
slug: workspace
profiles:
  default:
    slug: default
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4o-mini

  fast:
    slug: fast
    stack:
      - profile_slug: default
    inference_settings:
      chat:
        engine: gpt-4.1-mini
      inference:
        reasoning_effort: low

  deep:
    slug: deep
    stack:
      - profile_slug: default
    inference_settings:
      chat:
        engine: gpt-5
      inference:
        reasoning_effort: high
```

### Optional runtime profile file: `~/.config/pinocchio/runtime-profiles.yaml`

```yaml
profiles:
  coding:
    system-prompt: |
      You are a precise Go coding assistant.
    tools:
      include:
        - shell
        - grep
        - calculator
    middlewares:
      - tool-result-reorder
      - trace
```

### Runtime assembly

```text
config.yaml -> base settings
profiles.yaml + selected profile -> final engine settings
runtime-profiles.yaml + selected runtime profile -> tools + middleware + prompts
```

That is a clean split.

---

## What I would tell a new intern on day one

1. Start by reading these files in order:
   - `pinocchio/pkg/cmds/loader.go`
   - `pinocchio/pkg/cmds/cmd.go`
   - `pinocchio/pkg/cmds/profile_base_settings.go`
   - `pinocchio/pkg/cmds/helpers/profile_runtime.go`
   - `geppetto/pkg/sections/sections.go`
   - `geppetto/pkg/sections/profile_sections.go`
   - `geppetto/pkg/engineprofiles/types.go`
   - `geppetto/pkg/engineprofiles/source_chain.go`
   - `geppetto/pkg/engineprofiles/service.go`
   - `geppetto/pkg/inference/engine/factory/factory.go`

2. Do not assume “profile” means one thing. Ask which one of these is meant:
   - baseline config defaults
   - engine profile registry selection
   - runtime/tool/middleware preset

3. When debugging, print both:
   - base settings
   - final settings

4. Never use `factory.NewEngineFromParsedValues(...)` if the command is supposed to honor engine profile registries.

5. Prefer a single shared helper instead of re-implementing profile/config resolution in each command.

---

## Bottom line

If your goal is:

- use Glazed everywhere,
- load baseline config once,
- add Geppetto `profile` + `profile-registries`,
- and create an engine quickly from the selected profile,

then the right move is to standardize on a Pinocchio-owned abstraction that returns:

- `BaseSettings`
- `FinalSettings`
- `ResolvedProfile`
- `Profile` / `ProfileRegistries`
- `Close()`

and then make loaded commands, JS commands, and agent/demo commands all consume that same abstraction.

The main architectural decision to protect is this one:

```text
engine profiles choose engines
runtime profiles choose middleware/tools/prompts
```

If you keep that separation explicit, the rest of the refactor becomes much easier.
