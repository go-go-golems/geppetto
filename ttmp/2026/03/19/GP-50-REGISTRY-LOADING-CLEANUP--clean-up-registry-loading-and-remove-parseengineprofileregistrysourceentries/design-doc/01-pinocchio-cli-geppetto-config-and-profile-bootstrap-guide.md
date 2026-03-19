---
Title: Pinocchio CLI Geppetto config and profile bootstrap guide
Ticket: GP-50-REGISTRY-LOADING-CLEANUP
Status: active
Topics:
    - profiles
    - glazed
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/sections/profile_sections.go
      Note: Shared profile section and parser-time middleware bootstrap
    - Path: glazed/pkg/config/resolve.go
      Note: Default application config file search order
    - Path: pinocchio/cmd/pinocchio/main.go
      Note: Root CLI bootstrap and loaded-command entrypoint wiring
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: Loaded-command runtime path and local profile fallback logic
    - Path: pinocchio/pkg/cmds/helpers/profile_runtime.go
      Note: Thin-command runtime helper for base settings plus selected profile resolution
    - Path: pinocchio/pkg/cmds/loader.go
      Note: Schema/default construction for YAML-loaded commands
    - Path: pinocchio/pkg/cmds/profile_base_settings.go
      Note: Base-settings derivation by stripping profile parse steps
ExternalSources: []
Summary: Detailed intern-facing guide to the current Pinocchio and Geppetto CLI bootstrap paths, why loaded commands became confusing, and a recommended unified runtime/bootstrap model for config, profile selection, and engine creation.
LastUpdated: 2026-03-19T08:23:37.199663368-04:00
WhatFor: Explain the current and historical command-loading/runtime-resolution architecture and propose a clear implementation plan for standardizing Glazed-based Geppetto CLI bootstrapping across Pinocchio.
WhenToUse: Use when adding or refactoring any Pinocchio or Geppetto CLI command that should load baseline inference settings, expose profile selection, and construct an engine with minimal custom plumbing.
---


# Pinocchio CLI Geppetto config and profile bootstrap guide

## Executive Summary

This document explains how Pinocchio currently bootstraps Geppetto-backed CLI commands, why that flow became hard to reason about, and how to simplify it for future commands.

The short version is:

- there are currently multiple bootstrap paths,
- they do not all agree on where baseline inference settings come from,
- they do not all agree on how profile selection is defaulted,
- and they do not all share one helper for turning parsed values into an engine.

Today there are at least three important paths:

1. loaded YAML commands, which are parsed through `BuildCobraCommandWithGeppettoMiddlewares(...)` and then do additional runtime profile merging inside `pinocchio/pkg/cmds/cmd.go`;
2. profile-only app commands, such as the simple chat agent, which use `ResolveFinalInferenceSettings(...)` from `pinocchio/pkg/cmds/helpers/profile_runtime.go`;
3. older or transitional helpers, such as `ParseGeppettoLayers(...)`, which rebuild parts of the middleware stack manually.

That split is the main reason the system feels confusing.

The recommended direction is:

- standardize on Glazed/Cobra command construction,
- standardize on the shared Geppetto `profile-settings` section from `geppetto/pkg/sections/profile_sections.go`,
- standardize on one runtime helper that returns:
  - base inference settings,
  - resolved profile selection,
  - final merged inference settings,
  - and optionally a ready-to-use engine,
- keep loaded-command support, but make it consume the same runtime helper instead of reimplementing profile fallback logic inside `cmd.go`.

There is one important clarification for the future design:

- current code treats `config.yaml` and `profiles.yaml` as different things,
- but the desired operator model in this ticket wants `profiles.yaml` to be the main baseline inference source.

That distinction must become explicit in the implementation and in the docs.

## Problem Statement

You want a simple and repeatable contract for Geppetto-backed CLI commands:

- use the Glazed framework,
- load a baseline inference configuration automatically,
- add the shared `profile` and `profile-registries` surface,
- and let the caller construct an engine quickly from the selected profile.

That sounds straightforward, but the current system grew in layers over time.

The current architecture mixes:

- command-schema defaults loaded from YAML command files,
- Glazed middleware-based config parsing,
- profile bootstrap in Geppetto middleware,
- separate helper-based profile bootstrap in Pinocchio,
- and older ad-hoc fallback behavior in individual commands.

For a new intern, the following questions are all harder than they should be:

- “Where do my base inference settings come from?”
- “Which file wins: `config.yaml` or `profiles.yaml`?”
- “Where is `default` profile injected?”
- “What helper should I use if I just want an engine?”

The specific problems are:

### 1. There are multiple sources of truth for runtime bootstrap

`PinocchioCommand.RunIntoWriter(...)` in `pinocchio/pkg/cmds/cmd.go` builds final runtime settings in one way, while `ResolveFinalInferenceSettings(...)` in `pinocchio/pkg/cmds/helpers/profile_runtime.go` builds them in another way.

Those paths overlap heavily, but they do not share one core implementation.

### 2. Baseline config and profile-registry fallback use different default files

`glazed/pkg/config/resolve.go` resolves the default application config file as:

- `$XDG_CONFIG_HOME/pinocchio/config.yaml`
- `$HOME/.pinocchio/config.yaml`
- `/etc/pinocchio/config.yaml`

That is current behavior, not a proposal.

By contrast, `geppetto/pkg/sections/profile_sections.go` and several Pinocchio helpers look for the default registry file at:

- `$XDG_CONFIG_HOME/pinocchio/profiles.yaml`

Those are not the same file or the same concept, but the distinction is not surfaced clearly.

### 3. Loaded YAML commands and explicit app commands do not share one mental model

Loaded commands go through:

- `PinocchioCommandLoader`,
- `BuildCobraCommandWithGeppettoMiddlewares`,
- then runtime merging in `PinocchioCommand.RunIntoWriter`.

Profile-only commands usually go through:

- a command description with `NewProfileSettingsSection()`,
- then `ResolveFinalInferenceSettings(...)`,
- then `factory.NewEngineFromSettings(...)`.

Both are valid patterns today, but they are not documented as two deliberate architectures.

### 4. Transitional helpers still exist

`ParseGeppettoLayers(...)` in `pinocchio/pkg/cmds/helpers/parse-helpers.go` reconstructs a partial middleware chain manually and uses older string-based profile-registry handling.

It still works for some example-style code, but it is not the model we should ask new code to follow.

### 5. Profile defaulting is inconsistent

The Geppetto Cobra middleware bootstrap path sets:

- `profile = "default"` when no explicit profile is provided.

But `ResolveProfileSettings(...)` in Pinocchio helpers does not always inject `"default"` on its own; some callers set it manually, some do not, and some rely on registry default behavior.

That inconsistency is subtle and error-prone.

## Proposed Solution

Define one standard bootstrap contract for Geppetto-backed CLI commands and then make the different command families plug into it.

The standard contract should be:

1. every CLI command is built with Glazed/Cobra;
2. every command that needs profile selection attaches the shared section from `geppetto/pkg/sections/profile_sections.go`;
3. every command resolves runtime state through one helper that returns a rich bootstrap object;
4. engine creation is a one-liner on top of that bootstrap object.

### Recommended runtime abstraction

Introduce one shared runtime object in Pinocchio, conceptually like this:

```go
type GeppettoRuntime struct {
    ConfigFiles     []string
    ProfileSettings geppettosections.ProfileSettings
    BaseSettings    *settings.InferenceSettings
    ResolvedProfile *engineprofiles.ResolvedEngineProfile
    FinalSettings   *settings.InferenceSettings
    Engine          engine.Engine
    Close           func()
}
```

And expose helpers along these lines:

```go
func ResolveGeppettoRuntime(
    ctx context.Context,
    parsed *values.Values,
    opts ...RuntimeOption,
) (*GeppettoRuntime, error)

func CreateEngineFromParsed(
    ctx context.Context,
    parsed *values.Values,
    ef factory.EngineFactory,
    opts ...RuntimeOption,
) (engine.Engine, *GeppettoRuntime, error)
```

The exact names can vary, but the interface should make the intended workflow obvious:

- parse once,
- resolve once,
- create engine once.

### Recommended operator model

For future CLI commands, the intended behavior should be:

- app-level configuration still comes from Pinocchio config search paths,
- profile selection comes from the shared `profile-settings` section,
- engine-specific baseline inference values come from the selected engine profile registry,
- and the common case uses the `default` profile from the default registry file.

If the long-term intention is “`profiles.yaml` is the operator-facing baseline for inference behavior”, then the code should reflect that explicitly.

That means the design should make a clean distinction between:

- **application config**
  - repositories
  - other non-inference app settings
  - lives in `config.yaml`

- **inference runtime baseline**
  - model/provider/provider-specific knobs
  - lives in `profiles.yaml`
  - most often under a `default` profile

### Recommended command patterns

There should be two officially supported command patterns.

#### Pattern A: profile-only commands

Use this when the command only needs:

- normal app config loading,
- `--profile`,
- `--profile-registries`,
- and a final engine.

This should be the common default for app commands such as:

- agents,
- web apps,
- small utilities,
- most new Pinocchio commands.

The implementation should be:

- attach `NewProfileSettingsSection()`,
- parse with Glazed/Cobra,
- call the shared runtime/bootstrap helper,
- create the engine.

#### Pattern B: full Geppetto schema commands

Use this when the command intentionally exposes the full Geppetto inference surface or embeds per-command YAML defaults.

This includes:

- loaded YAML commands,
- commands that expose full inference/provider flags,
- advanced examples.

The implementation should still use the same runtime helper, but the helper must accept parsed values that already include:

- command-local defaults,
- profile-applied values,
- config/env/flag overrides.

That means the helper must know how to derive **base settings from parsed values** by stripping profile-originated parse steps instead of rerunning a hidden parse that would lose command-local defaults.

## Design Decisions

### Decision 1: the shared profile section is the canonical CLI surface

Use:

- `geppetto/pkg/sections/profile_sections.go`
- `ProfileSettings`
- `ProfileSettingsSectionSlug`
- `NewProfileSettingsSection(...)`

Rationale:

- it gives one flag definition,
- one type contract,
- one place for default registry fallback behavior,
- and one place for future semantics such as default profile selection or stronger validation.

### Decision 2: keep Glazed parsing as the first-class mechanism

Do not build new command flows around ad-hoc environment-variable parsing or raw `flag` parsing unless the command is intentionally a tiny standalone tool.

Rationale:

- Glazed already knows how to merge defaults, config, env, args, and Cobra flags,
- it already records parse logs and sources,
- and Pinocchio already depends heavily on it.

### Decision 3: treat `parsedValues` as the canonical runtime input

The runtime helper should consume `*values.Values`, not a dozen primitive parameters.

Rationale:

- loaded commands already parse into `values.Values`,
- explicit commands already parse into `values.Values`,
- parse logs are necessary when stripping profile-originated values,
- and this avoids duplicating resolution logic at the call site.

### Decision 4: separate app config from inference baseline explicitly

Current code already distinguishes these implicitly:

- `config.yaml` is discovered by `ResolveAppConfigPath(...)`,
- `profiles.yaml` is discovered by `defaultPinocchioProfileRegistriesIfPresent()`.

The future design should document that split on purpose.

Recommended rule:

- `config.yaml` is for application bootstrap and non-inference configuration,
- `profiles.yaml` is for inference behavior and profile selection.

If the team really wants “the baseline for all Geppetto inference settings lives in `~/.pinocchio/profiles.yaml`”, then that should mean:

- the default registry file is required or strongly expected,
- its `default` profile carries the baseline inference settings,
- and commands resolve `default` unless the operator chooses another profile.

### Decision 5: engine creation should be a thin layer

Commands should not manually repeat:

- config-file search,
- profile-registry search,
- profile slug normalization,
- registry parsing,
- registry-chain creation,
- merge logic.

They should do:

```go
runtime, err := pinhelpers.ResolveGeppettoRuntime(ctx, parsed)
if err != nil { ... }
defer runtime.Close()

eng, err := factory.NewEngineFromSettings(runtime.FinalSettings)
if err != nil { ... }
```

or the single-call equivalent.

### Decision 6: loaded commands remain supported, but stop owning custom profile fallback logic

`PinocchioCommand.RunIntoWriter(...)` currently contains its own profile fallback path.

That code exists for good historical reasons, but it is the wrong long-term ownership boundary.

The runtime helper should own:

- profile selection extraction,
- base settings derivation,
- profile resolution,
- final settings merge,
- engine creation helpers.

`PinocchioCommand` should own only:

- prompt/message rendering,
- UI mode selection,
- turn construction,
- and actual command execution.

### Design Diagram: current state

```text
                    CURRENT STATE

        YAML command file / explicit Glazed command
                          |
                          v
      +-----------------------------------------------+
      | Command schema / sections / defaults          |
      |                                               |
      | Loaded command path:                          |
      |   PinocchioCommandLoader                      |
      |   -> NewInferenceSettingsFromYAML             |
      |   -> CreateGeppettoSections(defaults)         |
      +-----------------------------------------------+
                          |
                          v
      +-----------------------------------------------+
      | Cobra + Glazed parser                         |
      |                                               |
      | Loaded command path:                          |
      |   BuildCobraCommandWithGeppettoMiddlewares    |
      |   -> GetCobraCommandGeppettoMiddlewares       |
      |                                               |
      | Explicit command path:                        |
      |   plain cli.BuildCobraCommand                 |
      +-----------------------------------------------+
                          |
                          v
                     parsedValues
                          |
             +------------+-------------+
             |                          |
             v                          v
   PinocchioCommand.RunIntoWriter   ResolveFinalInferenceSettings
   (local runtime logic)            (helper runtime logic)
             |                          |
             v                          v
      final inference settings     final inference settings
             |                          |
             +------------+-------------+
                          |
                          v
                    engine creation
```

### Design Diagram: recommended state

```text
                  RECOMMENDED STATE

      Any Glazed/Cobra command with parsedValues
                          |
                          v
      +-----------------------------------------------+
      | Shared runtime bootstrap helper               |
      |                                               |
      | ResolveGeppettoRuntime(ctx, parsed, opts...)  |
      |   - resolve config files                      |
      |   - resolve profile settings                  |
      |   - derive base settings                      |
      |   - resolve selected profile                  |
      |   - merge final settings                      |
      |   - optionally create engine                  |
      +-----------------------------------------------+
                          |
                          v
          +---------------+----------------+
          |                                |
          v                                v
   PinocchioCommand                 app commands / agents / web
   (turn/UI execution)              (simple engine consumers)
```

## Current State: detailed walkthrough

### 1. Command discovery and loader path

`pinocchio/pkg/cmds/loader.go` is the loader for YAML-defined Pinocchio commands.

What it does:

- reads a YAML command file into `PinocchioCommandDescription`,
- parses embedded inference defaults with `settings.NewInferenceSettingsFromYAML(...)`,
- turns those defaults into Glazed sections via `geppettosections.CreateGeppettoSections(...)`,
- prepends the helper section,
- and creates a `PinocchioCommand`.

Important lines:

- `pinocchio/pkg/cmds/loader.go:64-78`
- `pinocchio/pkg/cmds/loader.go:80-123`

The subtle but important point is that this loader injects **command-local defaults** into the schema. That matters later when computing base settings for a loaded command.

### 2. Root CLI bootstrapping for loaded commands

`pinocchio/cmd/pinocchio/main.go` has two key responsibilities for loaded commands:

- load the YAML file in `run-command`,
- build a Cobra command using `BuildCobraCommandWithGeppettoMiddlewares(...)`.

Important lines:

- `pinocchio/cmd/pinocchio/main.go:68-100`
- `pinocchio/cmd/pinocchio/main.go:161-167`

The root command also attaches the shared `profile-settings` section to the root command itself, which gives inherited `--profile` and `--profile-registries` flags for regular Cobra subcommands.

### 3. Geppetto middleware bootstrap

The parser-time bootstrap lives in Geppetto.

Relevant APIs:

- `geppetto/pkg/sections/sections.go:37-116`
- `geppetto/pkg/sections/profile_sections.go:68-96`
- `geppetto/pkg/sections/profile_sections.go:98-267`

What happens in the middleware path:

- full Geppetto sections are created,
- command settings are bootstrapped from Cobra/env/defaults,
- config files are resolved,
- profile settings are bootstrapped from Cobra/env/config/defaults,
- the middleware chain is then assembled in precedence order.

This is a real design improvement because it resolves profile selection early enough to create the profile middleware consistently.

### 4. Application config path resolution

Current config-path behavior is defined in:

- `glazed/pkg/config/resolve.go:9-47`

Current default search order is:

- `$XDG_CONFIG_HOME/pinocchio/config.yaml`
- `$HOME/.pinocchio/config.yaml`
- `/etc/pinocchio/config.yaml`

This means **current code does not treat `profiles.yaml` as the default application config file**.

### 5. Default profile-registry path resolution

Current default profile-registry fallback is defined in:

- `geppetto/pkg/sections/profile_sections.go:55-66`

Current fallback behavior is:

- look for `$XDG_CONFIG_HOME/pinocchio/profiles.yaml`

That means the system currently has separate default discovery for:

- app config
- profile registries

### 6. Runtime helper path for profile-only commands

`pinocchio/pkg/cmds/helpers/profile_runtime.go` is the cleanest current implementation of “I only need profile selection plus final inference settings”.

Important lines:

- `ResolveBaseInferenceSettings`: `pinocchio/pkg/cmds/helpers/profile_runtime.go:37-65`
- `ResolveEngineProfileSettings`: `pinocchio/pkg/cmds/helpers/profile_runtime.go:83-113`
- `ResolveFinalInferenceSettings`: `pinocchio/pkg/cmds/helpers/profile_runtime.go:115-170`

This helper:

- reconstructs hidden Geppetto sections,
- loads base inference settings from config/env/defaults,
- resolves the shared profile section,
- loads the selected profile from registry sources,
- merges base settings with profile settings,
- and returns a closer if registry resources must be closed.

This is already close to the desired thin-command model.

### 7. Runtime path inside `PinocchioCommand`

`pinocchio/pkg/cmds/cmd.go` is different.

Important lines:

- `stepSettings.UpdateFromParsedValues(...)`: `pinocchio/pkg/cmds/cmd.go:213-221`
- local profile fallback block: `pinocchio/pkg/cmds/cmd.go:223-250`
- base settings derivation: `pinocchio/pkg/cmds/cmd.go:252-258`
- runtime registry resolution: `pinocchio/pkg/cmds/cmd.go:259-272`
- transfer into run context: `pinocchio/pkg/cmds/cmd.go:317-335`

This path is more complicated because loaded commands already have a fully parsed schema with command-local defaults and possibly profile-applied parse steps.

The good part is:

- `baseSettingsFromParsedValues(parsedValues)` correctly strips `source == "profiles"` from parse logs and preserves all other parsed defaults and overrides.

The confusing part is:

- `cmd.go` still owns profile fallback extraction itself rather than delegating to a shared runtime helper.

### 8. Legacy partial-bootstrap helper

`pinocchio/pkg/cmds/helpers/parse-helpers.go` contains `ParseGeppettoLayers(...)`.

This helper:

- reconstructs parts of the config/env/default chain manually,
- uses older string-based registry parsing,
- and does not represent the best current design.

It is useful as historical evidence, but it should be treated as a transitional helper.

## Historical Evolution

This section matters because the current design only makes sense if you see how it evolved.

### Phase 1: loaded commands and repository loading were the main abstraction

Earlier Pinocchio leaned heavily on repository-loaded YAML commands.

Relevant historical point:

- commit `37d2fb4` (four months ago) added more explicit config-file loading behavior in `cmd/pinocchio/main.go`.

That older root flow used:

- repository loading,
- `cli.WithCobraMiddlewaresFunc(...)`,
- and earlier profile-settings plumbing directly in the repository loader setup.

### Phase 2: runtime profile resolution became a separate concern

Later, Pinocchio needed:

- clearer profile resolution,
- better engine profile support,
- and simpler app commands that were not always repository-loaded YAML commands.

Relevant historical points:

- commit `64d2f39` introduced `pkg/cmds/helpers/profile_runtime.go`
- commit `cf01006` migrated CLI and JS profile loading to engine profiles

This is where the helper runtime path became first-class.

### Phase 3: loaded commands still needed custom handling

Even after helper-based profile runtime existed, loaded commands still had a special problem:

- their schema already contained command-local inference defaults,
- so simply re-running hidden config/env/default parsing was not enough.

That gap was fixed in:

- commit `d6fd5e6` (`fix loaded command engine profile resolution`)

That commit added runtime merging logic inside `pinocchio/pkg/cmds/cmd.go` and a test proving loaded commands can resolve a selected engine profile correctly.

### Phase 4: shared section cleanup

Most recently:

- commit `279f8c9` moved Pinocchio to the shared Geppetto `profile-settings` section and `[]string` registries.

This cleaned up flag ownership, but it did not yet fully unify runtime bootstrap ownership.

## Why the current system feels confusing

The system feels confusing because “bootstrap” is actually spread across several layers.

### Layer A: schema construction

The loader constructs the command schema and can inject YAML-defined inference defaults.

### Layer B: parser bootstrap

The Cobra/Glazed parser can add config/env/default/profile-registry values before runtime starts.

### Layer C: runtime reconstruction

Some commands rebuild base settings later from hidden sections or from parsed-value logs.

### Layer D: engine creation

Commands or helpers finally turn the final merged inference settings into an engine.

The most important concrete confusion points are:

- `CreateGeppettoSections(...)` is used both to define real CLI surfaces and to perform hidden bootstrap parsing.
- `ResolveFinalInferenceSettings(...)` is elegant for profile-only commands, but it does not directly explain how loaded-command YAML defaults are preserved.
- `PinocchioCommand.RunIntoWriter(...)` preserves loaded-command semantics, but it duplicates ownership that should eventually belong to shared runtime helpers.
- `ParseGeppettoLayers(...)` is still present and can mislead readers into thinking partial manual middleware reconstruction is the normal model.
- `RunContext.ProfileRegistries` still stores a comma-joined string, even though the shared section now uses `[]string`.

## Recommended Implementation Model

### The model to teach a new intern

Tell the intern to think about the system in exactly three steps:

1. **Parse** command inputs into `values.Values`.
2. **Resolve** runtime settings from those parsed values.
3. **Construct** an engine from the resolved final settings.

Everything else is plumbing.

### The one standard workflow

For new CLI commands, the intended flow should be:

```text
Define command with Glazed sections
    -> parse into values.Values
    -> resolve shared runtime object
    -> create engine
    -> run command-specific logic
```

### Pseudocode for the desired standard helper

```go
func ResolveGeppettoRuntime(ctx context.Context, parsed *values.Values, opts ...RuntimeOption) (*GeppettoRuntime, error) {
    profile := resolveProfileSettingsWithDefaults(parsed)
    configFiles := resolveConfigFiles(parsed)

    var base *settings.InferenceSettings
    if hasParsedGeppettoSections(parsed) {
        base = baseSettingsFromParsedValues(parsed)
    } else {
        base = executeHiddenGeppettoBaseParse(configFiles, opts...)
    }

    final := base
    var resolved *engineprofiles.ResolvedEngineProfile
    var closer func()

    if len(profile.ProfileRegistries) > 0 {
        chain, closeFn := openRegistryChain(profile.ProfileRegistries)
        closer = closeFn

        slug := profile.Profile
        if strings.TrimSpace(slug) == "" {
            slug = "default"
        }

        resolved = resolveSelectedProfile(ctx, chain, slug)
        final = MergeInferenceSettings(base, resolved.InferenceSettings)
    }

    return &GeppettoRuntime{
        ConfigFiles:     configFiles,
        ProfileSettings: profile,
        BaseSettings:    base,
        ResolvedProfile: resolved,
        FinalSettings:   final,
        Close:           closer,
    }, nil
}

func CreateEngineFromParsed(ctx context.Context, parsed *values.Values, ef factory.EngineFactory) (engine.Engine, *GeppettoRuntime, error) {
    rt, err := ResolveGeppettoRuntime(ctx, parsed)
    if err != nil {
        return nil, nil, err
    }
    if ef == nil {
        ef = factory.NewStandardEngineFactory()
    }
    eng, err := ef.CreateEngine(rt.FinalSettings)
    if err != nil {
        if rt.Close != nil {
            rt.Close()
        }
        return nil, nil, err
    }
    rt.Engine = eng
    return eng, rt, nil
}
```

### Practical interpretation of the requested operator model

The desired contract can be implemented as:

- **Glazed framework**
  - every command is a Glazed command or a Cobra command with Glazed sections
- **baseline inference source**
  - default to profile-registry path discovery
  - load the `default` profile from `profiles.yaml`
- **shared profile surface**
  - always use `NewProfileSettingsSection()`
- **fast engine construction**
  - `CreateEngineFromParsed(...)` should be the one obvious helper

### How to handle `config.yaml` versus `profiles.yaml`

This is the point where the implementation guide should be explicit.

#### Option A: keep dual files

- `config.yaml` remains application config
- `profiles.yaml` remains profile registry config
- baseline inference behavior comes from the `default` profile in `profiles.yaml`

This is my recommended option.

It matches the current code structure better, because:

- repository discovery already lives in app config,
- profile registry lookup already lives in `profiles.yaml`,
- and engine/inference behavior naturally belongs in a registry/profile document.

#### Option B: move all inference baseline to `config.yaml`

- `config.yaml` owns the Geppetto inference sections
- profiles become optional overlays

This is simpler in theory, but it weakens the “profile-first” operator model and keeps the split between config-driven commands and profile-driven commands alive longer.

### How loaded commands should fit the model

Loaded commands should not be a special runtime architecture.

They should only be special in one narrow way:

- they may contribute command-local inference defaults through the loader schema.

That means the shared runtime helper must support:

- deriving base settings from parsed values when those parsed values already contain real command defaults.

That is the real architectural insight behind `baseSettingsFromParsedValues(...)`.

## API and File Reference Map

### Core parser/bootstrap APIs

- `geppetto/pkg/sections/CreateGeppettoSections(...)`
  - `geppetto/pkg/sections/sections.go:37-116`
- `geppetto/pkg/sections/NewProfileSettingsSection(...)`
  - `geppetto/pkg/sections/profile_sections.go:68-96`
- `geppetto/pkg/sections/GetProfileSettingsMiddleware(...)`
  - `geppetto/pkg/sections/profile_sections.go:98-267`
- `pinocchio/pkg/cmds/BuildCobraCommandWithGeppettoMiddlewares(...)`
  - `pinocchio/pkg/cmds/cobra.go:12-25`

### Core runtime APIs

- `pinocchio/pkg/cmds/helpers/ResolveFinalInferenceSettings(...)`
  - `pinocchio/pkg/cmds/helpers/profile_runtime.go:115-170`
- `pinocchio/pkg/cmds/profile_base_settings.go:12-78`
  - strips profile-originated parse steps to recover a base settings snapshot
- `pinocchio/pkg/cmds/cmd.go:200-340`
  - current loaded-command runtime path
- `factory.NewEngineFromSettings(...)`
  - used by several direct commands after settings resolution

### Core loader/root APIs

- `pinocchio/pkg/cmds/loader.go:37-123`
- `pinocchio/cmd/pinocchio/main.go:68-100`
- `pinocchio/cmd/pinocchio/main.go:142-168`

### Current config path semantics

- `glazed/pkg/config/resolve.go:9-47`

### Representative command styles

- Loaded YAML command path:
  - `pinocchio/pkg/cmds/loader.go`
  - `pinocchio/pkg/cmds/cmd.go`
- Thin profile-only command:
  - `pinocchio/cmd/agents/simple-chat-agent/main.go`
- Transitional helper/example:
  - `pinocchio/cmd/examples/simple-chat/main.go`
  - `pinocchio/pkg/cmds/helpers/parse-helpers.go`
- App-style custom command with manual base resolver:
  - `pinocchio/cmd/web-chat/main.go`

## Alternatives Considered

### Alternative 1: keep the current split and just document it better

Rejected as the primary answer.

Why:

- documentation helps,
- but the ownership split is still real,
- and future commands would still have to choose among multiple overlapping helpers.

### Alternative 2: move everything into Geppetto middleware only

Partially attractive, but not enough.

Why:

- parser middleware is the right place for config/env/flag precedence,
- but runtime still needs a clean way to return base settings, final settings, resolved profile metadata, and closers.
- commands still need a friendly “give me an engine” helper.

### Alternative 3: eliminate YAML loader defaults entirely

Rejected.

Why:

- loaded command files are still valuable,
- they can still supply low-precedence command defaults,
- and deleting that feature would solve the architecture problem by removing a user-visible capability.

### Alternative 4: keep inference baseline in `config.yaml` only

Not recommended given the desired operator model.

Why:

- it weakens the “profile registry as runtime baseline” story,
- it makes profile-first workflows feel secondary,
- and it preserves more of the current ambiguity than necessary.

## Implementation Plan

### Phase 1: document and codify the standard contract

Goal:

- make the intended model explicit before refactoring code further.

Concrete work:

- land this guide,
- update help/docs for future commands if needed,
- decide whether the long-term operator baseline is:
  - `config.yaml` for app config and `profiles.yaml` for inference, or
  - a different arrangement.

### Phase 2: introduce one shared runtime bootstrap helper

Suggested new file:

- `pinocchio/pkg/cmds/helpers/runtime_bootstrap.go`

Suggested responsibilities:

- resolve profile settings from parsed values,
- normalize default profile selection,
- derive base settings from parsed values when available,
- otherwise run hidden base parsing,
- resolve selected engine profile,
- merge final settings,
- optionally create an engine.

### Phase 3: move `cmd.go` onto the shared runtime helper

Refactor:

- `pinocchio/pkg/cmds/cmd.go`

Target end state:

- `RunIntoWriter(...)` no longer owns local profile fallback extraction,
- it asks the shared runtime helper for:
  - `BaseSettings`,
  - `FinalSettings`,
  - `ProfileSettings`,
  - `ResolvedProfile`,
  - and possibly `Engine`.

Important detail:

- preserve the current `baseSettingsFromParsedValues(...)` behavior for loaded commands,
- because that is what keeps command-local YAML defaults intact.

### Phase 4: move thin commands onto the same helper

Refactor representative commands:

- `pinocchio/cmd/agents/simple-chat-agent/main.go`
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/cmd/pinocchio/cmds/js.go`
- any remaining example commands that still use `ParseGeppettoLayers(...)`

Target:

- thin commands call the same bootstrap helper,
- then call `factory.NewEngineFromSettings(...)` or the convenience engine helper.

### Phase 5: retire or narrow transitional helpers

Candidates:

- `pinocchio/pkg/cmds/helpers/parse-helpers.go`
- local runtime-specific config/profile logic in app commands

Possible outcome:

- keep `ParseGeppettoLayers(...)` only for legacy examples,
- or remove it entirely after call sites are migrated.

### Phase 6: normalize the profile contract

Choose one rule and apply it everywhere:

- empty profile becomes `"default"`, or
- empty profile means “use registry default”.

Recommendation:

- use `"default"` explicitly for CLI/operator-facing behavior.

Why:

- it is easier to document,
- easier to test,
- and more predictable across registries.

### Phase 7: convert `RunContext.ProfileRegistries` to `[]string`

Refactor:

- `pinocchio/pkg/cmds/run/context.go`

Current issue:

- the shared section now uses `[]string`,
- but the run context still stores a comma-joined string.

That is an old boundary that should eventually disappear.

### Phase 8: add focused tests around the unified helper

Recommended test matrix:

- loaded command with YAML defaults plus selected profile
- profile-only command with default registry fallback
- explicit `--config-file` overriding app config
- explicit `--profile` overriding default profile
- empty profile using `"default"`
- no registry configured
- registry chain merge order

Suggested files:

- `pinocchio/pkg/cmds/helpers/profile_runtime_test.go`
- new tests beside the new bootstrap helper
- `pinocchio/pkg/cmds/cmd_profile_registry_test.go`

## Open Questions

### 1. Should inference baseline live in `profiles.yaml` only?

This document recommends:

- keep app config in `config.yaml`,
- keep inference runtime baseline in `profiles.yaml`,
- use `default` profile as the baseline inference contract.

That seems most aligned with the requested operator model, but it still needs an explicit project decision.

### 2. Should loaded YAML commands still be allowed to embed inference defaults?

I think yes, as low-precedence command-local defaults.

But once the shared runtime helper exists, the team should decide whether that is still desirable or whether loaded commands should be prompt-focused only.

### 3. Should profile resolution always require a registry file?

Current code already trends toward “yes”.

If that remains the intended direction, then the docs and helper error messages should describe it clearly.

### 4. Should application config search prefer XDG or legacy `~/.pinocchio/`?

Current `ResolveAppConfigPath(...)` prefers XDG first and only then falls back to `~/.pinocchio/config.yaml`.

That is reasonable, but if user-facing docs keep mentioning `~/.pinocchio/profiles.yaml`, they should also mention the XDG path explicitly to avoid confusion.

### 5. Should Geppetto own more of this bootstrap logic directly?

Right now the runtime/bootstrap split is shared between Geppetto and Pinocchio.

It may be worth asking whether:

- Geppetto should own parser/bootstrap for inference settings and profile sections,
- while Pinocchio owns command-specific UI/runtime behavior.

That is a plausible longer-term cleanup, but it is larger than this ticket.

## References

### Key files

- `pinocchio/pkg/cmds/cmd.go`
- `pinocchio/pkg/cmds/loader.go`
- `pinocchio/pkg/cmds/profile_base_settings.go`
- `pinocchio/pkg/cmds/helpers/profile_runtime.go`
- `pinocchio/pkg/cmds/helpers/parse-helpers.go`
- `pinocchio/pkg/cmds/cobra.go`
- `pinocchio/cmd/pinocchio/main.go`
- `pinocchio/cmd/agents/simple-chat-agent/main.go`
- `pinocchio/cmd/examples/simple-chat/main.go`
- `pinocchio/cmd/web-chat/main.go`
- `geppetto/pkg/sections/sections.go`
- `geppetto/pkg/sections/profile_sections.go`
- `geppetto/pkg/steps/ai/settings/settings-inference.go`
- `glazed/pkg/config/resolve.go`

### Key historical commits

- `37d2fb4` `:arrow_up: :books: Investigate profile and config file loading and use at least new config file loader`
- `64d2f39` `own runtime step settings outside profile resolution`
- `cf01006` `migrate pinocchio cli and js to engine profiles`
- `d6fd5e6` `fix loaded command engine profile resolution`
- `279f8c9` `refactor(profiles): reuse shared profile settings section`

### Related ticket docs

- `analysis/01-registry-loading-cleanup-analysis-and-migration-inventory.md`
- `analysis/02-reusable-profile-settings-section-migration-analysis.md`
