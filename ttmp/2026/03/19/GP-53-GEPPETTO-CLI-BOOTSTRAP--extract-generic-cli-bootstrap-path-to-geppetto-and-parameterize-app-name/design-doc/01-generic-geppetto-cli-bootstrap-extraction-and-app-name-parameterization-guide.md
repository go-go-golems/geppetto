---
Title: Generic Geppetto CLI Bootstrap Extraction and App Name Parameterization Guide
Ticket: GP-53-GEPPETTO-CLI-BOOTSTRAP
Status: active
Topics:
    - profiles
    - glazed
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/cmds/profilebootstrap/engine_settings.go
      Note: Current generic resolved engine-settings implementation candidate to move into Geppetto
    - Path: ../../../../../../../pinocchio/pkg/cmds/profilebootstrap/profile_selection.go
      Note: Current generic profile-selection implementation candidate to move into Geppetto
    - Path: cmd/examples/internal/runnerexample/inference_settings.go
      Note: Current registry-only example helper that demonstrates a second Geppetto bootstrap style
    - Path: pkg/inference/engine/factory/helpers.go
      Note: Current direct parsed-values engine helper that solves a narrower problem than the proposed bootstrap package
    - Path: pkg/sections/profile_sections.go
      Note: Older profile bootstrap path with app-specific assumptions that should not be the long-term home
    - Path: pkg/sections/sections.go
      Note: Older Geppetto bootstrap path that still hardcodes Pinocchio assumptions
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-19T11:42:23.788747828-04:00
WhatFor: Define how to extract the generic CLI config/profile/bootstrap path from Pinocchio into Geppetto while making app name, env prefix, and config discovery configurable.
WhenToUse: Use when implementing or reviewing the follow-up refactor that moves generic CLI bootstrap responsibilities into Geppetto.
---


# Generic Geppetto CLI Bootstrap Extraction and App Name Parameterization Guide

## Executive Summary

Pinocchio now has a cleaner CLI bootstrap model than Geppetto itself. The new code in `pinocchio/pkg/cmds/profilebootstrap` resolves config files, profile selection, optional engine-profile overlay, and final engine settings in one explicit flow. The problem is that the implementation is still Pinocchio-owned even though most of the logic is generic Geppetto logic with only a small amount of app-specific configuration.

This ticket proposes extracting that generic path into a Geppetto package, parameterizing the app name and related bootstrap settings, and then making Pinocchio consume that Geppetto package rather than owning its own implementation. The target is not to move every existing Geppetto command to the new path immediately. The target is to create one reusable bootstrap API that can serve:

- applications that expose full Geppetto inference sections
- applications that expose only `profile-settings`
- applications that want baseline config loading plus optional engine-profile overlay

The extraction should produce a package whose callers provide app-specific knobs such as app name, env prefix, config-file mapper, and baseline section construction, while Geppetto owns the generic resolution algorithm and the resolved result structures.

## Problem Statement

There are currently three partially overlapping bootstrap models in the codebase.

First, many Geppetto example commands simply parse full AI sections and call `factory.NewEngineFromParsedValues(...)`. That works for direct inference flags, but it does not model the newer baseline-config plus profile-overlay story explicitly. The representative implementation is in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/factory/helpers.go`.

Second, the Geppetto registry-only examples decode the shared `profile-settings` section and then call a custom example helper, `ResolveInferenceSettingsFromRegistry(...)`, from `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/internal/runnerexample/inference_settings.go`. That helper is fine for examples, but it is not a reusable CLI bootstrap contract. It does not model baseline config files, config precedence, or resolved config/profile metadata.

Third, Geppetto still contains older shared bootstrap code in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go` and `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_sections.go`. That code is shared in name, but it is not actually generic. It hardcodes Pinocchio assumptions:

- `PINOCCHIO` as the environment prefix
- `pinocchio` as the config app name
- an implicit default profile of `"default"`
- an implicit fallback to `~/.config/pinocchio/profiles.yaml`

Meanwhile, Pinocchio now has a cleaner explicit bootstrap layer in:

- `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go`
- `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go`

That code is closer to the desired long-term architecture, but it is still in the wrong package. It ties a generic problem to one application package name and one app-specific config/env naming scheme.

The result is unnecessary complexity:

- Geppetto does not own its own best generic bootstrap path.
- Pinocchio contains reusable logic that other Geppetto-backed apps should use.
- Old Geppetto shared code still bakes in Pinocchio-specific assumptions.
- There is no single Geppetto-level data structure for “resolved config files + selected profile + base settings + final settings + resolved engine profile”.

## Proposed Solution

Create a new Geppetto package that owns the generic CLI bootstrap contract and make the application-specific parts explicit inputs.

Recommended package:

```text
geppetto/pkg/cli/bootstrap
```

The package should own four categories of things.

### 1. Resolved data structures

Move the generic shapes from Pinocchio into Geppetto-level types.

```go
type ProfileSettings struct {
    Profile           string   `glazed:"profile"`
    ProfileRegistries []string `glazed:"profile-registries"`
}

type ResolvedCLIProfileSelection struct {
    ProfileSettings
    ConfigFiles []string
}

type ResolvedCLIEngineSettings struct {
    BaseInferenceSettings  *settings.InferenceSettings
    FinalInferenceSettings *settings.InferenceSettings
    ProfileSelection       *ResolvedCLIProfileSelection
    ResolvedEngineProfile  *profiles.ResolvedEngineProfile
    ConfigFiles            []string
    Close                  func()
}
```

These types are application-agnostic. They should not mention Pinocchio.

### 2. Parameterized app configuration

Define an options/config structure that captures only the app-specific parts.

```go
type AppBootstrapConfig struct {
    AppName          string
    EnvPrefix        string
    ConfigFileMapper sources.ConfigFileMapper
    NewProfileSection func() (schema.Section, error)
    BuildBaseSections func() ([]schema.Section, error)
}
```

There are several acceptable shapes here. The important thing is not the exact name, but the boundary:

- `AppName` controls config-path discovery through `glazed/pkg/config`
- `EnvPrefix` controls `sources.FromEnv(...)`
- `ConfigFileMapper` controls how app config files are converted into section maps
- `NewProfileSection` allows callers to reuse the shared Geppetto `profile-settings` section or provide a compatible specialized version
- `BuildBaseSections` allows callers to define which hidden baseline sections participate in base inference settings resolution

The package should validate that `AppName`, `EnvPrefix`, and the section builder callbacks are present.

### 3. Generic resolution functions

The package should expose explicit resolution stages instead of one giant helper.

Recommended API:

```go
func ResolveCLIConfigFiles(cfg AppBootstrapConfig, parsed *values.Values) ([]string, error)
func ResolveCLIProfileSelection(cfg AppBootstrapConfig, parsed *values.Values) (*ResolvedCLIProfileSelection, error)
func ResolveBaseInferenceSettings(cfg AppBootstrapConfig, parsed *values.Values) (*settings.InferenceSettings, []string, error)
func ResolveCLIEngineSettings(ctx context.Context, cfg AppBootstrapConfig, parsed *values.Values) (*ResolvedCLIEngineSettings, error)
func ResolveCLIEngineSettingsFromBase(ctx context.Context, cfg AppBootstrapConfig, base *settings.InferenceSettings, parsed *values.Values, baseConfigFiles []string) (*ResolvedCLIEngineSettings, error)
func NewEngineFromResolvedCLIEngineSettings(engineFactory factory.EngineFactory, resolved *ResolvedCLIEngineSettings) (engine.Engine, error)
```

The call sequence should stay explicit:

```text
parsed values
    |
    v
ResolveCLIConfigFiles
    |
    v
ResolveCLIProfileSelection
    |
    v
ResolveBaseInferenceSettings
    |
    v
ResolveCLIEngineSettings / ResolveCLIEngineSettingsFromBase
    |
    v
NewEngineFromResolvedCLIEngineSettings
```

### 4. Thin app wrappers

Once the generic package exists, Pinocchio should become a thin wrapper:

```go
var PinocchioBootstrapConfig = geppettobootstrap.AppBootstrapConfig{
    AppName:   "pinocchio",
    EnvPrefix: "PINOCCHIO",
    ConfigFileMapper: pinocchioConfigFileMapper,
    NewProfileSection: geppettosections.NewProfileSettingsSection,
    BuildBaseSections: sections.CreateGeppettoSections,
}
```

Then the current `pinocchio/pkg/cmds/profilebootstrap` package can either:

- become a compatibility wrapper over Geppetto bootstrap, or
- be deleted after call sites migrate.

### Control-flow diagram

```text
                        +-----------------------+
                        | AppBootstrapConfig    |
                        | - AppName             |
                        | - EnvPrefix           |
                        | - ConfigFileMapper    |
                        | - BuildBaseSections   |
                        +-----------+-----------+
                                    |
                     +--------------+--------------+
                     |                             |
                     v                             v
          ResolveCLIProfileSelection     ResolveBaseInferenceSettings
                     |                             |
                     +--------------+--------------+
                                    |
                                    v
                      ResolveCLIEngineSettings
                                    |
                   +----------------+----------------+
                   |                                 |
                   v                                 v
          BaseInferenceSettings             FinalInferenceSettings
                   |                                 |
                   +----------------+----------------+
                                    |
                                    v
                 NewEngineFromResolvedCLIEngineSettings
```

### Behavioral rules

The new Geppetto package should preserve the post-cutover rules already enforced in Pinocchio:

- No implicit `profiles.yaml` fallback.
- No implicit default profile injection.
- Baseline config plus direct flags is valid with no registry.
- Selecting a profile without any profile registries is a validation error.
- Config-file precedence stays explicit and observable through `ConfigFiles`.

## Design Decisions

### Decision 1: The new home should be a new Geppetto package, not `pkg/sections`

`pkg/sections` already mixes three concerns:

- section construction
- Cobra/Glazed middleware assembly
- bootstrap-time profile/config resolution

That package is too coupled to the older middleware-oriented path. The new bootstrap package should be a narrower package centered on resolved CLI state, not section assembly.

### Decision 2: Application identity must be explicit

Hardcoding `"pinocchio"` and `"PINOCCHIO"` is exactly the bug this extraction is trying to remove. App identity must be caller-provided.

### Decision 3: Keep section construction separate from resolution

`CreateGeppettoSections(...)` is still useful for building hidden baseline sections. It should remain a section-construction utility. The new package should call it through a callback, not absorb it.

### Decision 4: Preserve staged resolution rather than returning only an engine

Returning only `engine.Engine` hides too much information. The resolved structures are valuable because they preserve:

- config-file provenance
- whether a profile overlay was used
- the resolved engine profile metadata
- the base vs final inference settings split

That observability is one of the main improvements of the new Pinocchio path and should not be lost in the Geppetto extraction.

### Decision 5: Migrate Pinocchio first, then optionally migrate Geppetto examples

The first value of the new package is making Pinocchio depend on Geppetto instead of owning the logic. Only after that should we decide which Geppetto examples should migrate from `NewEngineFromParsedValues(...)` or `runnerexample.ResolveInferenceSettingsFromRegistry(...)`.

## Alternatives Considered

### Alternative A: Leave the code in Pinocchio and let other apps copy it

Rejected because it repeats the current mistake: generic Geppetto logic would remain app-owned.

### Alternative B: Expand `factory.NewEngineFromParsedValues(...)`

Rejected because `factory.NewEngineFromParsedValues(...)` intentionally solves a narrower problem: build an engine from already-parsed AI sections. It is not the right home for config-file discovery, profile-selection metadata, or resolved engine-profile overlay state.

### Alternative C: Keep using `pkg/sections.GetCobraCommandGeppettoMiddlewares(...)`

Rejected because that path still carries older Pinocchio-specific assumptions and mixes bootstrap with middleware assembly. It is a migration source, not the final abstraction.

### Alternative D: Move the Pinocchio package wholesale into Geppetto without parameterization

Rejected because that would just transplant the bug. The whole point is to make app name, env prefix, and config mapping configurable.

## Implementation Plan

### Phase 1: Add the Geppetto bootstrap package

Create a new package in Geppetto and port the generic pieces from:

- `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go`
- `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go`

Do not import Pinocchio.

### Phase 2: Introduce explicit app config

Add the app-config structure and make all config/env/section construction flow through it.

The first supported app config should reproduce current Pinocchio behavior except for the app-specific naming:

```go
AppName: "pinocchio"
EnvPrefix: "PINOCCHIO"
```

### Phase 3: Migrate Pinocchio to the Geppetto package

Change `pinocchio/pkg/cmds/profilebootstrap` into wrappers or remove it entirely. Target files:

- `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go`
- `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go`
- `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_selection.go`
- `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_engine_settings.go`

### Phase 4: Add focused tests in Geppetto

Add tests in the new Geppetto package for:

- config discovery with `AppName`
- env-prefix handling with `EnvPrefix`
- no implicit registry fallback
- profile-without-registries validation
- base-only mode
- profile overlay merge behavior
- `ResolveCLIEngineSettingsFromBase(...)` preserving caller-provided base settings

### Phase 5: Evaluate Geppetto call-site migration

Once the package is stable, classify Geppetto call sites:

- keep direct `NewEngineFromParsedValues(...)` where full AI sections are already the desired UX
- migrate registry-only examples to the new bootstrap API if they should also model config-file discovery and resolved metadata
- deprecate or simplify `GetCobraCommandGeppettoMiddlewares(...)` / `GetProfileSettingsMiddleware(...)` if they become redundant

### Suggested migration order

1. Add Geppetto bootstrap package and tests.
2. Migrate Pinocchio wrappers to call it.
3. Migrate one thin Pinocchio command and one loaded Pinocchio path as proof.
4. Remove duplicated Pinocchio implementation.
5. Reassess Geppetto examples and old `pkg/sections` bootstrap helpers.

### Pseudocode sketch

```go
cfg := bootstrap.AppBootstrapConfig{
    AppName:          "pinocchio",
    EnvPrefix:        "PINOCCHIO",
    ConfigFileMapper: pinocchioConfigFileMapper,
    NewProfileSection: sections.NewProfileSettingsSection,
    BuildBaseSections: sections.CreateGeppettoSections,
}

resolved, err := bootstrap.ResolveCLIEngineSettings(ctx, cfg, parsed)
if err != nil {
    return err
}
defer maybeClose(resolved.Close)

eng, err := bootstrap.NewEngineFromResolvedCLIEngineSettings(nil, resolved)
if err != nil {
    return err
}
```

## Open Questions

The following questions should be resolved during implementation rather than blocking ticket creation.

- Should the new package also own the shared config-file mapper helper, or should callers always provide one?
- Should `ResolveBaseInferenceSettings(...)` accept a callback that returns sections, or a prebuilt schema, or a callback that returns a baseline `InferenceSettings` default object?
- Should Pinocchio keep its compatibility wrapper package for one release cycle, or hard-cut to the Geppetto package immediately?
- Should the existing Geppetto `pkg/sections` bootstrap helpers be deprecated in code comments once the new package exists?
- Should Geppetto examples continue using `runnerexample.ResolveInferenceSettingsFromRegistry(...)`, or should that helper itself become a thin wrapper over the new package?

## References

- Current Pinocchio generic bootstrap candidate:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go`
- Current Pinocchio loaded-command consumer:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd.go`
- Current Pinocchio loader defaults source:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/loader.go`
- Current Geppetto direct-engine helper:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/factory/helpers.go`
- Current Geppetto registry-only example helper:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/internal/runnerexample/inference_settings.go`
- Current Geppetto older shared bootstrap path:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_sections.go`
- Related upstream ticket that motivated this follow-up:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries`
