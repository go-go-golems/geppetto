---
Title: Ideal app-facing API and hard-cut implementation plan for caller-owned runtime resolution
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
    - Path: geppetto/pkg/profiles/registry.go
      Note: Current ResolveInput and ResolvedProfile contract still expose BaseStepSettings, EffectiveStepSettings, and RuntimeKeyFallback
    - Path: geppetto/pkg/profiles/service.go
      Note: ResolveEffectiveProfile currently derives EffectiveStepSettings and synthesizes RuntimeKey from RuntimeKeyFallback
    - Path: geppetto/pkg/profiles/runtime_settings_patch_resolver.go
      Note: Houses the patch-based helper model slated for deletion
    - Path: geppetto/pkg/steps/ai/settings/settings-step.go
      Note: Defines concrete StepSettings that should become the caller-owned engine configuration object
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Representative caller path currently converting resolved profile runtime into final engine settings
    - Path: pinocchio/pkg/webchat/llm_loop_runner.go
      Note: Shows the actual Geppetto boundary where app-owned engine and registry are handed to the tool loop/session layer
    - Path: 2026-03-16--gec-rag/internal/webchat/runtime.go
      Note: Second representative caller path currently applying StepSettingsPatch during runtime composition
ExternalSources: []
Summary: Detailed target API and hard-cut migration plan for moving final runtime resolution fully to callers and deleting StepSettingsPatch plus RuntimeKeyFallback in the same series.
LastUpdated: 2026-03-17T22:35:00-04:00
WhatFor: Use this document to understand the ideal long-term API, why Geppetto should stop owning profile-to-engine resolution, and how to execute the hard cut safely without compatibility shims.
WhenToUse: Use when implementing GP-43, reviewing the caller-owned runtime boundary, or deciding how GP-45 RuntimeKeyFallback cleanup folds into the same cut.
---

# Ideal app-facing API and hard-cut implementation plan for caller-owned runtime resolution

## Executive Summary

The ideal API is simple:

- the application resolves a fully concrete inference runtime,
- Geppetto receives that concrete runtime and runs inference,
- Geppetto does not own profile patching, runtime identity fallback, or partial engine-configuration logic.

That means GP-43 should be implemented as a hard cut with these removals in one series:

- `RuntimeSpec.StepSettingsPatch`
- `ResolveInput.BaseStepSettings`
- `ResolvedProfile.EffectiveStepSettings`
- patch-merge/apply helpers
- `ResolveInput.RuntimeKeyFallback`

This document argues that `RuntimeKeyFallback` should be folded into the same implementation series rather than left for a separate conceptual layer. Once the app owns final runtime resolution, runtime identity is naturally app-owned too. Keeping a Geppetto-level fallback string after that would preserve the old blurred boundary.

## The ideal boundary

### What the app should do

The app should do all of the following before it asks Geppetto to run inference:

- select a profile or runtime preset,
- load defaults, config files, and environment variables,
- resolve provider credentials and model settings,
- build final `*settings.StepSettings`,
- resolve middleware uses into built middleware values,
- decide which tools are exposed,
- decide runtime identity and cache keys.

### What Geppetto should do

Geppetto should do only these things:

- build an `engine.Engine` from final settings,
- wrap the engine in middleware,
- run inference,
- run the tool loop,
- persist and emit events if the caller wires those pieces in.

### Ideal object shape

In an ideal world the caller would have a fully concrete runtime object like this:

```go
type ResolvedInferenceRuntime struct {
    Key         string
    Fingerprint string

    StepSettings *settings.StepSettings
    SystemPrompt string
    Middlewares  []middleware.Middleware
    ToolRegistry tools.ToolRegistry

    Metadata map[string]any
}
```

This is intentionally not a Geppetto core type requirement. Different apps may have slightly different structs. The important point is that the shape is concrete and caller-owned.

### Ideal Geppetto-side construction API

The most elegant Geppetto helper is small:

```go
type EngineSpec struct {
    StepSettings *settings.StepSettings
    SystemPrompt string
    Middlewares  []middleware.Middleware
}

func BuildEngine(ctx context.Context, spec EngineSpec) (engine.Engine, error)
```

Then session/toolloop wiring stays exactly where it belongs:

```go
eng, err := geppetto.BuildEngine(ctx, geppetto.EngineSpec{
    StepSettings: rt.StepSettings,
    SystemPrompt: rt.SystemPrompt,
    Middlewares:  rt.Middlewares,
})
if err != nil { ... }

sess := session.NewSession()
sess.Builder = &enginebuilder.Builder{
    Base:       eng,
    Registry:   rt.ToolRegistry,
    LoopConfig: &loopCfg,
    ToolConfig: &toolCfg,
    EventSinks: []events.EventSink{sink},
}
```

## Why the current API is inelegant

Today Geppetto profiles still expose a patch-shaped, partially resolved engine boundary:

```text
ResolveInput.BaseStepSettings
  + RuntimeSpec.StepSettingsPatch
  + RuntimeKeyFallback
  -> ResolvedProfile.EffectiveStepSettings
  -> runtime composer
  -> engine
```

This is inelegant because:

- it mixes app config resolution with profile metadata resolution,
- it makes `ResolvedProfile` partly about engine config and partly about app runtime metadata,
- it forces apps to understand Geppetto’s section-patch format,
- it turns runtime identity into something Geppetto synthesizes rather than something the caller owns.

`RuntimeKeyFallback` is especially revealing here. It exists because Geppetto is still trying to invent runtime identity inside profile resolution instead of receiving runtime identity from the caller.

## Hard-cut recommendation

Do not preserve backward compatibility here.

The hard-cut version is cleaner and easier to explain:

1. Callers build final `*settings.StepSettings`.
2. Callers compute runtime key/fingerprint.
3. Profile resolution returns only profile/runtime metadata.
4. Geppetto consumes final runtime values and runs.

### Explicit removals

From Geppetto core:

- `RuntimeSpec.StepSettingsPatch`
- `ResolveInput.BaseStepSettings`
- `ResolveInput.RuntimeKeyFallback`
- `ResolvedProfile.EffectiveStepSettings`
- `ApplyRuntimeStepSettingsPatch`
- `MergeRuntimeStepSettingsPatches`

From caller flows and docs:

- `effectiveStepSettings`
- `effectiveRuntime.step_settings_patch`
- examples that treat profile resolution as StepSettings construction
- examples that treat runtime identity as a Geppetto fallback concern

## API before and after

### Current style

```go
base := loadBaseStepSettings()

resolved, err := registry.ResolveEffectiveProfile(ctx, profiles.ResolveInput{
    ProfileSlug:        profileSlug,
    BaseStepSettings:   base,
    RuntimeKeyFallback: profiles.MustRuntimeKey(profileSlug.String()),
})
if err != nil { ... }

eng, err := BuildEngineFromSettingsWithMiddlewares(
    ctx,
    resolved.EffectiveStepSettings,
    resolved.EffectiveRuntime.SystemPrompt,
    resolvedMiddlewares,
)
```

### Ideal style

```go
rt, err := app.ResolveInferenceRuntime(ctx, request)
if err != nil { ... }

eng, err := geppetto.BuildEngine(ctx, geppetto.EngineSpec{
    StepSettings: rt.StepSettings,
    SystemPrompt: rt.SystemPrompt,
    Middlewares:  rt.Middlewares,
})
if err != nil { ... }

sess := session.NewSession()
sess.Builder = &enginebuilder.Builder{
    Base:       eng,
    Registry:   rt.ToolRegistry,
    LoopConfig: &loopCfg,
    ToolConfig: &toolCfg,
}
```

### Resulting mental model

```text
app request/config/profile
  -> ResolvedInferenceRuntime
  -> Geppetto BuildEngine + session/toolloop
```

That is much easier to teach than:

```text
base settings
  -> geppetto profile resolution
  -> patch maps
  -> effective settings
  -> runtime fallback keys
  -> engine
```

## Implementation plan

## Phase 1: freeze the target contract

Define the target Geppetto-side contract explicitly.

`ResolvedProfile` should shrink to:

```go
type ResolvedProfile struct {
    RegistrySlug       RegistrySlug
    ProfileSlug        ProfileSlug
    RuntimeFingerprint string
    EffectiveRuntime   RuntimeSpec
    Metadata           map[string]any
}
```

And `RuntimeSpec` should shrink to:

```go
type RuntimeSpec struct {
    SystemPrompt string
    Middlewares  []MiddlewareUse
    Tools        []string
}
```

No `StepSettingsPatch`.
No runtime-key fallback input.

Caller-side runtime identity becomes explicit in app code:

```go
type AppRuntimeIdentity struct {
    Key         string
    Fingerprint string
}
```

## Phase 2: add caller-owned resolution helpers in apps first

Do this before deleting Geppetto fields.

### Pinocchio

Create an app-owned helper along these lines:

```go
type ResolvedWebchatRuntime struct {
    RuntimeKey         string
    RuntimeFingerprint string

    StepSettings *settings.StepSettings
    SystemPrompt string
    Middlewares  []middleware.Middleware
    ToolNames    []string
    Metadata     map[string]any
}
```

Responsibilities:

- load base settings from flags/env/config,
- resolve profile runtime metadata,
- build final `StepSettings` itself,
- compute runtime key/fingerprint itself.

### GEC-RAG

Create an equivalent app-owned runtime resolver in the webchat package.

### Temporal Relationships

Move final `StepSettings` creation into its command/runtime bootstrap layer before run-chat transport creation.

## Phase 3: switch callers to the new object shape

Once caller-owned runtime resolution exists:

- `runtime_composer.go` in Pinocchio should accept final `StepSettings`, not `ResolvedProfileRuntime.StepSettingsPatch`,
- GEC-RAG runtime composition should consume final `StepSettings`,
- helper functions that currently return `EffectiveStepSettings` should return app-owned resolved runtime/config instead.

At this point Geppetto profile resolution is no longer on the critical path for engine configuration.

## Phase 4: delete the Geppetto patch and fallback APIs

Files to change:

- `geppetto/pkg/profiles/types.go`
- `geppetto/pkg/profiles/registry.go`
- `geppetto/pkg/profiles/service.go`
- `geppetto/pkg/profiles/runtime_settings_patch_resolver.go`
- `geppetto/pkg/sections/profile_registry_source.go`
- JS profile bindings and type definitions

Tasks:

1. Remove `StepSettingsPatch` from `RuntimeSpec`.
2. Remove `BaseStepSettings` from `ResolveInput`.
3. Remove `RuntimeKeyFallback` from `ResolveInput`.
4. Remove `EffectiveStepSettings` from `ResolvedProfile`.
5. Delete patch helper functions and their tests.
6. Delete fallback runtime-key synthesis from `ResolveEffectiveProfile`.

After this phase, any caller that still expects Geppetto to create final StepSettings or runtime identity will fail fast, which is the intended hard-cut behavior.

## Phase 5: remove compatibility surfaces and stale examples

Delete or rewrite:

- JS APIs that expose `effectiveStepSettings`,
- JS examples that teach `runtimeKeyFallback`,
- docs that teach `runtime.step_settings_patch`,
- docs that describe profile resolution as an engine-settings builder.

This is where the old mental model is actually removed from the repo.

## Phase 6: validation

Validate three things separately:

1. App-owned runtime resolution produces final `StepSettings`.
2. Geppetto still builds/runs engines correctly from final settings.
3. No stale profile-patch or runtime-fallback references remain.

Concrete validation passes:

- `geppetto/pkg/profiles`
- `geppetto/pkg/js/modules/geppetto`
- `geppetto/pkg/sections`
- Pinocchio webchat runtime composition and profile switching
- GEC-RAG runtime composition
- Temporal Relationships run-chat runtime bootstrap

## Rationale for folding RuntimeKeyFallback into GP-43

If GP-43 lands without removing `RuntimeKeyFallback`, the system still sends a mixed message:

- engine settings are caller-owned,
- but runtime identity is still partly Geppetto-owned.

That is not a clean boundary.

The same architectural reasoning applies to both:

- `StepSettingsPatch` is Geppetto inventing engine configuration,
- `RuntimeKeyFallback` is Geppetto inventing runtime identity.

Both belong to the caller.

So the clean hard cut is:

```text
caller owns final engine config
caller owns runtime identity
Geppetto runs concrete runtime values
```

## Open questions

These questions should be answered once implementation starts, but they do not change the target design.

### Should Geppetto ship a convenience helper for engine construction?

Probably yes, but it should be thin:

- input: final `StepSettings` + prompt + middleware values
- output: `engine.Engine`

It should not reintroduce profile-aware resolution logic.

### Should Geppetto ship a generic runtime resolver?

No, probably not.

That would risk recreating the same too-smart shared abstraction we are removing.

### Should apps standardize on one runtime struct?

Not required.

Pinocchio, GEC-RAG, and Temporal Relationships can each have app-specific resolved runtime structs, as long as they all converge on the same Geppetto boundary:

- final `StepSettings`
- concrete middleware values
- concrete tool registry

## Recommendation

Implement GP-43 as the main hard-cut ticket and explicitly absorb the `RuntimeKeyFallback` removal into it.

Do not split the conceptual cleanup into:

- “remove patch settings now”
- “remove runtime identity fallback later”

The elegant API emerges only when both are gone.
