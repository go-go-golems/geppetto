---
Title: Remove StepSettingsPatch and move final StepSettings resolution to callers: design and implementation guide
Ticket: GP-43-REMOVE-STEPSETTINGSPATCH
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - config
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/types.go
      Note: Defines RuntimeSpec and currently stores StepSettingsPatch
    - Path: geppetto/pkg/profiles/registry.go
      Note: Defines BaseStepSettings and EffectiveStepSettings on profile resolution APIs
    - Path: geppetto/pkg/profiles/service.go
      Note: Applies StepSettingsPatch during ResolveEffectiveProfile
    - Path: geppetto/pkg/profiles/runtime_settings_patch_resolver.go
      Note: Houses StepSettingsPatch apply/merge logic slated for deletion
    - Path: geppetto/pkg/sections/profile_registry_source.go
      Note: Converts resolved profile patches into Glazed section source maps
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Applies profile StepSettingsPatch during runtime composition
    - Path: 2026-03-16--gec-rag/internal/webchat/runtime.go
      Note: Applies profile StepSettingsPatch during runtime composition
    - Path: temporal-relationships/internal/extractor/httpapi/run_chat_transport.go
      Note: Applies profile StepSettingsPatch during runtime composition
    - Path: pinocchio/pkg/cmds/helpers/profile_runtime.go
      Note: Uses EffectiveStepSettings as a caller-facing helper result
ExternalSources: []
Summary: Detailed plan for deleting StepSettingsPatch from Geppetto profile runtime, removing RuntimeKeyFallback in the same hard cut, and moving final runtime construction and caching to callers.
LastUpdated: 2026-03-17T22:35:00-04:00
WhatFor: Use this guide to remove StepSettingsPatch and RuntimeKeyFallback cleanly from the Geppetto profile system and migrate downstream applications to caller-owned final runtime resolution.
WhenToUse: Use when implementing, reviewing, or validating the hard cut that removes StepSettingsPatch, EffectiveStepSettings, BaseStepSettings, and RuntimeKeyFallback.
---

# Remove StepSettingsPatch and move final StepSettings resolution to callers: design and implementation guide

## Executive Summary

`StepSettingsPatch` should be deleted entirely.

The core problem is that Geppetto profiles currently do two jobs at once:

1. select runtime-facing app policy such as system prompt, middleware uses, and tool names,
2. partially configure provider engine settings by patching `*settings.StepSettings`.

That second job is the wrong boundary. Final `StepSettings` resolution belongs to the caller. The caller already owns config files, environment variables, app-level caching, engine identity, and runtime bootstrap policy. Geppetto should not merge partial step-setting fragments inside profile resolution and then return `EffectiveStepSettings` as if profile resolution were an engine-factory layer.

The target architecture is:

```text
caller config/profile logic
  -> final *settings.StepSettings
  -> app-owned runtime key/fingerprint
  -> optional app runtime metadata and caching
  -> Geppetto engine/toolloop/session primitives
```

After this change:

- `RuntimeSpec.StepSettingsPatch` is removed.
- `ResolveInput.BaseStepSettings` is removed.
- `ResolveInput.RuntimeKeyFallback` is removed.
- `ResolvedProfile.EffectiveStepSettings` is removed.
- `ApplyRuntimeStepSettingsPatch` and `MergeRuntimeStepSettingsPatches` are deleted.
- Callers are responsible for producing final `*settings.StepSettings` and runtime identity before building engines.

## Problem Statement

Today the profile system still treats engine settings as patchable runtime payload:

- `RuntimeSpec` stores `StepSettingsPatch` in `geppetto/pkg/profiles/types.go`.
- `ResolveInput` accepts `BaseStepSettings`, and `ResolvedProfile` returns `EffectiveStepSettings` in `geppetto/pkg/profiles/registry.go`.
- `ResolveEffectiveProfile` applies `StepSettingsPatch` to `BaseStepSettings` in `geppetto/pkg/profiles/service.go`.
- Pinocchio, GEC-RAG, and Temporal Relationships still apply profile patch data during runtime composition instead of receiving final resolved settings up front.

This has four costs:

1. Profile resolution becomes partly responsible for engine configuration.
2. Callers carry both profile runtime data and final step settings through the same flow.
3. The system encourages patch-shaped intermediate state rather than final concrete settings.
4. Caching becomes awkward because the app and Geppetto both participate in resolution of what should really be one final engine config.

The user intent behind this ticket is to simplify the system by making engine configuration and runtime identity concrete earlier and higher in the stack.

## Proposed Solution

Delete `StepSettingsPatch` from the profile runtime model, remove `RuntimeKeyFallback`, and move final runtime construction to callers.

### New boundary

Geppetto profiles should only resolve:

- registry/profile identity,
- profile metadata,
- system prompt,
- middleware uses,
- tool names.

Callers should resolve:

- config files,
- defaults,
- environment variables,
- provider credentials,
- any app-level overrides,
- final `*settings.StepSettings`.
- runtime key and fingerprint.

Then callers either:

- build an engine directly with `factory.NewEngineFromStepSettings(...)`, or
- pass final `*settings.StepSettings` into a thin app-owned engine factory.

### Resulting conceptual model

```text
Before:
  base StepSettings + profile patch -> EffectiveStepSettings -> engine

After:
  caller resolves final StepSettings -> engine
  profile resolution only returns app/runtime metadata
```

### New caller-side data shape

A caller-oriented shape like this is enough:

```go
type ResolvedEngineConfig struct {
    RuntimeKey   string
    StepSettings *settings.StepSettings
    SystemPrompt string
    ToolNames    []string
    MiddlewareUses []profiles.MiddlewareUse
    Fingerprint  string
}
```

The exact struct can differ per app, but the point is that `StepSettings` is final and concrete, not patch-shaped.

## Design Decisions

### Decision 1: remove `StepSettingsPatch` entirely, not just deprecate it silently

The patch mechanism is not just awkward syntax. It represents the wrong architectural boundary. Leaving it in place as a quiet legacy field would prolong the ambiguity.

### Decision 2: callers own final StepSettings and caching

The application already knows:

- where its config files are,
- what defaults it wants,
- what environment it trusts,
- what runtime identities it wants to cache.

That makes the app the natural owner of final step settings and runtime/cache keys.

### Decision 3: keep profile runtime fields that are clearly app/runtime policy

This ticket does not propose removing:

- `SystemPrompt`
- `Middlewares`
- `Tools`

Those are still coherent as profile-level runtime metadata, even if some apps later decide to own more of that too.

### Decision 4: do not force one caller-side resolver abstraction

Different apps may prefer different resolver shapes:

- Pinocchio may have one conversation-runtime resolver,
- CLI helpers may have a simpler `ResolveEffectiveStepSettings` replacement,
- Temporal Relationships may keep command-runtime helpers.

Geppetto should not replace `StepSettingsPatch` with another too-smart shared resolver abstraction.

## Current State

### Geppetto core

Current profile API surfaces:

- `RuntimeSpec.StepSettingsPatch` in `geppetto/pkg/profiles/types.go`
- `ResolveInput.BaseStepSettings` in `geppetto/pkg/profiles/registry.go`
- `ResolveInput.RuntimeKeyFallback` in `geppetto/pkg/profiles/registry.go`
- `ResolvedProfile.EffectiveStepSettings` in `geppetto/pkg/profiles/registry.go`
- `ApplyRuntimeStepSettingsPatch` and `MergeRuntimeStepSettingsPatches` in `geppetto/pkg/profiles/runtime_settings_patch_resolver.go`

Resolution path:

```text
ResolveInput.BaseStepSettings
  + EffectiveRuntime.StepSettingsPatch
  -> ApplyRuntimeStepSettingsPatch(...)
  -> ResolvedProfile.EffectiveStepSettings
```

### Downstream apps

Main consumers today:

- Pinocchio web chat applies `ResolvedProfileRuntime.StepSettingsPatch` in `pinocchio/cmd/web-chat/runtime_composer.go`.
- GEC-RAG applies it in `2026-03-16--gec-rag/internal/webchat/runtime.go`.
- Temporal Relationships applies it in `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go`.
- Pinocchio helper code and examples use `ResolvedProfile.EffectiveStepSettings` directly.

### Documentation and migration layer

Many docs and tests still teach:

- `runtime.step_settings_patch` in YAML,
- `resolved.EffectiveStepSettings` as the main result of profile resolution,
- `BaseStepSettings` as an input to profile resolution.

That makes this a broad but conceptually straightforward cleanup.

## Target Architecture

### New Geppetto profile result

`ResolvedProfile` should shrink to something closer to:

```go
type ResolvedProfile struct {
    RegistrySlug RegistrySlug
    ProfileSlug  ProfileSlug
    EffectiveRuntime   RuntimeSpec
    RuntimeFingerprint string
    Metadata           map[string]any
}
```

Where `RuntimeSpec` no longer includes `StepSettingsPatch`.

### New caller workflow

```text
caller loads base/default config
  -> caller resolves final StepSettings
  -> caller computes runtime key/fingerprint
  -> caller resolves profile runtime metadata
  -> caller builds engine from final StepSettings
  -> caller applies system prompt / middleware / tools policy
  -> Geppetto runs inference
```

### Minimal Geppetto responsibility

Geppetto should only need:

- `engine.Engine`
- middlewares
- optional tool registry
- loop/session options

It should not need:

- patch maps,
- base step settings,
- effective step settings,
- runtime-key fallback hints,
- profile-owned engine configuration merge logic.

## Detailed Implementation Plan

## Phase 1: freeze the target API

1. Decide the exact reduced `ResolvedProfile` shape.
2. Decide whether to keep `RuntimeSpec` as-is minus `StepSettingsPatch`.
3. Decide the caller-owned runtime identity shape.
4. Hard-cut decision: no temporary compatibility adapter in Geppetto core.

Deliverable:

- a concrete list of fields to remove from `types.go` and `registry.go`, including `RuntimeKeyFallback`.

## Phase 2: remove patch support from Geppetto profiles

Files to change:

- `geppetto/pkg/profiles/types.go`
- `geppetto/pkg/profiles/registry.go`
- `geppetto/pkg/profiles/service.go`
- `geppetto/pkg/profiles/stack_merge.go`
- `geppetto/pkg/profiles/stack_trace.go`
- `geppetto/pkg/profiles/validation.go`
- `geppetto/pkg/profiles/runtime_settings_patch_resolver.go` (delete)

Tasks:

1. Remove `StepSettingsPatch` from `RuntimeSpec`.
2. Remove `BaseStepSettings` from `ResolveInput`.
3. Remove `RuntimeKeyFallback` from `ResolveInput`.
4. Remove `EffectiveStepSettings` from `ResolvedProfile`.
5. Delete patch application from `ResolveEffectiveProfile`.
6. Delete runtime-key fallback synthesis from `ResolveEffectiveProfile`.
7. Delete patch merge support from stack merge.
8. Delete patch-related trace output.
9. Delete patch-specific validation and override parsing.

## Phase 3: remove caller-facing patch outputs

Files to change:

- `geppetto/pkg/sections/profile_registry_source.go`
- `geppetto/pkg/js/modules/geppetto/api_profiles.go`
- `geppetto/pkg/js/modules/geppetto/api_engines.go`
- generated JS type definitions and tests

Tasks:

1. Stop exporting `effectiveStepSettings`.
2. Stop exporting `effectiveRuntime.step_settings_patch`.
3. Stop exporting or parsing `runtimeKeyFallback` for profile-resolution APIs.
4. Remove code that reconstructs Glazed source maps from profile patches.

This is an important design consequence:

- profile resolution no longer doubles as a StepSettings source.
- profile resolution no longer invents runtime identity.

## Phase 4: migrate downstream apps

### Pinocchio

Replace:

- profile patch application in `cmd/web-chat/runtime_composer.go`
- helper functions returning `EffectiveStepSettings`

With:

- caller-owned final step settings resolution before profile runtime composition.

Likely path:

```text
resolve base/default config
  -> resolve final StepSettings
  -> resolve profile runtime metadata
  -> compose engine from final StepSettings + profile runtime metadata
```

### GEC-RAG

Move final step settings resolution into resolver/bootstrap code so runtime composition no longer receives patch-shaped config.

### Temporal Relationships

Move final step settings resolution into command/runtime setup helpers and pass final settings into run-chat composition.

## Phase 5: remove legacy docs and migration guidance

Update:

- README files
- profile YAML examples
- playbooks
- migration docs
- legacy profile migration code

Key requirement:

- stop teaching `runtime.step_settings_patch` as a supported profile feature.

## Phase 6: validate behavior

Validation must prove:

1. engines still build correctly from final caller-owned settings,
2. profile selection still affects system prompt, tools, and middlewares,
3. no app still depends on `EffectiveStepSettings`,
4. no docs or examples still advertise `step_settings_patch`.

## Detailed Implementation Tasks

The checklist in `tasks.md` is the implementation plan. It should be executed phase by phase with review checkpoints between:

1. Geppetto core API break,
2. JS/sections/doc surface cleanup,
3. downstream app migrations,
4. final deletion of compatibility scaffolding,
5. final doc/test validation.

## Pseudocode Sketch

### Before

```go
resolved, err := registry.ResolveEffectiveProfile(ctx, profiles.ResolveInput{
    ProfileSlug:      profile,
    BaseStepSettings: base,
})
if err != nil {
    return err
}

eng, err := factory.NewEngineFromStepSettings(resolved.EffectiveStepSettings)
```

### After

```go
finalStepSettings, err := app.ResolveFinalStepSettings(ctx, appConfig, env, profile)
if err != nil {
    return err
}

resolved, err := registry.ResolveEffectiveProfile(ctx, profiles.ResolveInput{
    ProfileSlug: profile,
})
if err != nil {
    return err
}

eng, err := factory.NewEngineFromStepSettings(finalStepSettings)
if err != nil {
    return err
}

eng = applyProfileRuntimeMetadata(eng, resolved.EffectiveRuntime)
```

The exact helper names will differ, but the main change is that the final step settings come from caller code, not profile patch application.

## Alternatives Considered

### Alternative 1: keep StepSettingsPatch but stop using it in main apps

Rejected because the wrong abstraction would remain in the public API and docs.

### Alternative 2: replace StepSettingsPatch with another typed Geppetto profile settings object

Rejected because that still keeps final engine settings resolution inside Geppetto profiles.

### Alternative 3: keep EffectiveStepSettings but remove only request-time patch overrides

Rejected because it leaves the core architectural confusion intact. It is cleaner than today, but still not the right boundary.

## Risks

### Risk 1: migration blast radius

This touches Geppetto, Pinocchio, GEC-RAG, Temporal Relationships, tests, docs, and example YAML.

Mitigation:

- implement in phases,
- migrate callers before deleting helpers they still use.

### Risk 2: caller-side duplication

If no shared app helper exists, multiple apps may each re-implement final step settings resolution.

Mitigation:

- allow app-owned shared helpers where useful,
- but keep them outside Geppetto core profile resolution.

### Risk 3: legacy YAML breakage

Existing profile files with `step_settings_patch` will fail after removal.

Mitigation:

- decide explicitly between hard break, migration script, or temporary validator warning.

## Open Questions

1. Should Geppetto still expose any helper for caller-side final step settings resolution, or should that move entirely to Pinocchio/other apps?
2. Should profile YAMLs containing `step_settings_patch` fail immediately, or should one migration command be provided first?
3. Should `Tools` and `Middlewares` remain in `RuntimeSpec`, or does this cleanup suggest an even thinner profile runtime object later?

## References

- `geppetto/pkg/profiles/types.go`
- `geppetto/pkg/profiles/registry.go`
- `geppetto/pkg/profiles/service.go`
- `geppetto/pkg/profiles/runtime_settings_patch_resolver.go`
- `geppetto/pkg/sections/profile_registry_source.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`
- `2026-03-16--gec-rag/internal/webchat/runtime.go`
- `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go`
- `pinocchio/pkg/cmds/helpers/profile_runtime.go`
