---
Title: Engine profiles architecture and migration guide
Ticket: GP-49-ENGINE-PROFILES
Status: active
Topics:
    - geppetto
    - architecture
    - inference
    - profile-registry
    - config
    - javascript
    - pinocchio
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed proposal for reintroducing Geppetto-owned engine profiles, renaming StepSettings to InferenceSettings, and moving runtime behavior entirely to application code."
LastUpdated: 2026-03-18T18:12:00-04:00
WhatFor: "Use this guide when implementing the clean split between Geppetto engine configuration and application runtime configuration."
WhenToUse: "Use when replacing RuntimeSpec-based profiles, planning the InferenceSettings rename, or preparing the hard-cut migration across Geppetto, Pinocchio, and JS APIs."
---

# Engine profiles architecture and migration guide

## Executive Summary

Geppetto currently has an unresolved architectural split. `GP-43` removed `runtime.step_settings_patch`, which was the correct response to a confusing patch-based configuration model, but the remaining profile system still mixes two different concerns:

1. engine configuration, which belongs to Geppetto, and
2. application runtime behavior, which belongs to applications such as Pinocchio, GEC-RAG, and custom runner hosts.

This document proposes a hard cut:

- rename `StepSettings` to `InferenceSettings`
- reintroduce Geppetto-owned profiles, but only as `EngineProfile`
- remove `RuntimeSpec` from the Geppetto profile model entirely
- move `system_prompt`, `middlewares`, `tools`, `runtimeKey`, and `runtimeFingerprint` out of Geppetto core
- make applications own runtime composition and runtime identity
- perform the migration with no backwards compatibility wrappers
- publish a separate migration playbook in `glazed/pkg/doc/...` so downstream codebases can migrate systematically

The key design principle is simple:

```text
Geppetto owns engine configuration.
Applications own runtime behavior.
```

If implemented cleanly, this should make the system more understandable than both:

- the old mixed `runtime + step_settings_patch` profile model
- the current post-GP-43 world where too much engine configuration responsibility was pushed outward into applications

## Problem Statement And Scope

### The problem

The current system still uses the word "profile" for a document that carries:

- engine-adjacent semantics in some places
- prompt and middleware behavior in other places
- tool filtering metadata in other places
- runtime identity and attribution implications in still other places

That was already conceptually muddy before `GP-43`, and it remains muddy now. The difference is that the old patch mechanism was removed, so the current ambiguity is easier to see.

This becomes especially obvious in the Pinocchio JS example path:

- `gp.runner.resolveRuntime({})` resolves selected profile runtime metadata
- `pinocchio.engines.fromDefaults(...)` builds an engine from base settings plus explicit overrides
- those two paths are not meant to be the same thing

That split is correct at one level, but the current profile model no longer offers a Geppetto-owned way to say "this profile should configure the engine."

### Scope of this ticket

This ticket covers:

- the conceptual redesign
- the new profile format and API shape
- the `StepSettings` to `InferenceSettings` rename
- the Geppetto/app boundary
- the migration plan

This ticket does not implement code. It defines the target architecture and the work slices.

## Current-State Analysis

### 1. `StepSettings` is still the core engine configuration object

The current engine configuration object lives in:

- [settings-step.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/settings-step.go)

Observed facts:

- `type StepSettings struct` is still the main configuration object for provider/model/client/inference settings.
- `NewStepSettings()` and `NewStepSettingsFromParsedValues(...)` are still the primary constructors.

This object is no longer a "step" in any meaningful architectural sense. It configures the engine and the inference client. The name is legacy terminology.

### 2. Geppetto engine construction still centers on `StepSettings`

The engine factory path still uses `StepSettings` directly:

- [api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go)
- [factory helpers/tests references](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/factory/helpers_test.go)

Observed facts:

- JS `gp.engines.fromConfig(...)` ultimately builds from `StepSettings`.
- Engine construction in Go continues to use `NewEngineFromStepSettings(...)`.

This is strong evidence that engine configuration is still a Geppetto-owned concept, even after the profile-patch removal.

### 3. Geppetto profiles currently model application runtime behavior, not engine settings

The current profile data model is:

- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/types.go)
- [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/service.go)
- [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/stack_merge.go)

Observed facts:

- `RuntimeSpec` contains:
  - `SystemPrompt`
  - `Middlewares`
  - `Tools`
- `ResolveEffectiveProfile(...)` produces `EffectiveRuntime`, not engine settings.
- `runtimeFingerprint(...)` fingerprints prompt/middleware/tool runtime data.

This means the current Geppetto profile system is already oriented toward app runtime behavior, not engine construction.

### 4. JS runtime assembly materializes prompt, middleware, and tool selection from profiles

The JS runtime metadata and runner paths are in:

- [api_runtime_metadata.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go)
- [api_runner.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runner.go)

Observed facts:

- resolved profile runtime becomes:
  - system prompt middleware
  - materialized middleware chain
  - tool-name filtering
  - runtime metadata stamping

That is application/runtime orchestration behavior, not engine configuration.

### 5. Pinocchio already uses separate paths for runtime resolution and engine construction

Relevant files:

- [profile_runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go)
- [js.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/cmds/js.go)
- [module.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/js/modules/pinocchio/module.go)
- [runner-profile-demo.js](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/examples/js/runner-profile-demo.js)

Observed facts:

- Pinocchio resolves profile selection through helper/config logic.
- `gp.runner.resolveRuntime({})` consumes the selected/default profile stack for runtime metadata.
- `pinocchio.engines.fromDefaults(...)` builds the engine from hidden base settings plus explicit overrides.

This is the clearest live evidence that "runtime profile" and "engine settings" are separate concerns.

### 6. Glazed already uses hard-cut migration playbooks for large API shifts

Relevant file:

- [migrating-to-facade-packages.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/glazed/pkg/doc/tutorials/migrating-to-facade-packages.md)

Observed facts:

- Glazed already documents hard-cut migrations without compatibility shims.
- The style is explicit, exhaustive, and stepwise.

This matters because the requested migration strategy here is similar: no wrappers, but a strong migration document.

## Gap Analysis

The current system has three architectural gaps.

### Gap 1: Geppetto has no first-class engine profile abstraction

After `GP-43`, Geppetto no longer has a reusable way to say:

```text
load engine preset X
stack it with engine preset Y
produce final engine settings
```

That is still a Geppetto concern and should not be rebuilt ad hoc in every application.

### Gap 2: Geppetto profiles still model app runtime behavior

The current `RuntimeSpec` lives in Geppetto core, but:

- prompt semantics are application-facing
- middleware names are application-facing
- tool filtering is application-facing
- runtime fingerprints are application-facing

So the existing Geppetto profile system is still holding app policy that should live above Geppetto.

### Gap 3: the `StepSettings` name hides the real boundary

`StepSettings` suggests:

- an older execution model
- legacy command semantics
- a lower-level lifecycle notion that is no longer central

But the object really means:

```text
all provider/model/client/inference settings needed to build an engine
```

So the name actively obscures the new design.

## Proposed Architecture

## Core decision

Adopt the following boundary:

```text
Geppetto:
  EngineProfile
  InferenceSettings
  engine profile registry loading
  engine profile resolution
  engine construction

Application:
  system prompt
  middleware selection
  tool filtering / registries
  runtime key / runtime fingerprint
  runtime caching
  session / conversation policy
```

### New names

Rename:

- `StepSettings` -> `InferenceSettings`

Introduce:

- `EngineProfile`
- `EngineProfileRegistry`
- `ResolvedEngineProfile`

Delete from Geppetto profiles:

- `RuntimeSpec`
- runtime middleware merge logic
- runtime tool merge logic
- runtime fingerprinting in the Geppetto profile layer

### What happens to the current profile registry stack

The current `pkg/profiles` subsystem should not be treated as one indivisible thing. It contains both:

1. useful registry mechanics, and
2. mixed runtime-profile semantics that should be removed.

The migration should keep the first category and delete or replace the second.

#### What should survive

These pieces are still valuable even after the profile-model redesign:

- registry source parsing from [source_chain.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/source_chain.go)
- read-only YAML registry loading
- read-only SQLite registry loading from [sqlite_store.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/sqlite_store.go)
- stacked registry reads and slug lookup
- default-profile selection semantics, if they are truly about engine selection
- provenance/version metadata, if retained for engine-profile debugging

These are infrastructure concerns. They are still useful when the payload becomes engine-only.

#### What should be deleted

These parts encode the old mixed runtime model and should not survive:

- `RuntimeSpec` in [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/types.go)
- `ResolvedProfile.EffectiveRuntime` in [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/registry.go)
- runtime stack merge rules in [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/stack_merge.go)
- runtime fingerprinting in [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/service.go)
- validation logic that only exists for prompt, middleware, or tool runtime payloads

Those surfaces are tied to the assumption that Geppetto profiles are executable runtime presets. GP-49 is explicitly rejecting that assumption.

#### What should be transformed

Current conceptual flow:

```text
ProfileRegistry
  -> Profile
  -> ResolvedProfile
  -> EffectiveRuntime
```

Target conceptual flow:

```text
EngineProfileRegistry
  -> EngineProfile
  -> ResolvedEngineProfile
  -> InferenceSettings
```

That means the current registry API should be narrowed and renamed, not preserved under the old mixed-model names.

#### Package recommendation

There are two realistic package strategies:

1. keep `pkg/profiles`, but repurpose it around engine-only profiles
2. hard-rename `pkg/profiles` to `pkg/engineprofiles`

Keeping `pkg/profiles` reduces import churn, but it hides the conceptual change. Renaming to `pkg/engineprofiles` creates more churn, but makes the new boundary explicit everywhere.

Because this migration is already intended as a no-compat hard cut, a package rename to `pkg/engineprofiles` is probably the cleaner end state.

#### Target API shape

Today the reader interface is effectively:

```go
type RegistryReader interface {
    ListRegistries(ctx context.Context) ([]RegistrySummary, error)
    GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*ProfileRegistry, error)
    ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error)
    GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, error)
    ResolveEffectiveProfile(ctx context.Context, in ResolveInput) (*ResolvedProfile, error)
}
```

The target interface should be conceptually closer to:

```go
type EngineRegistryReader interface {
    ListRegistries(ctx context.Context) ([]EngineRegistrySummary, error)
    GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*EngineProfileRegistry, error)
    ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*EngineProfile, error)
    GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*EngineProfile, error)
    ResolveEngineProfile(ctx context.Context, in ResolveInput) (*ResolvedEngineProfile, error)
}
```

And the result should terminate at final engine configuration:

```go
type ResolvedEngineProfile struct {
    RegistrySlug      RegistrySlug
    ProfileSlug       ProfileSlug
    InferenceSettings *settings.InferenceSettings
    Metadata          map[string]any
}
```

### Why `InferenceSettings` and not `EngineSettings`

The user explicitly prefers `InferenceSettings`, and that name is defensible:

- it is broader than provider-only or model-only naming
- it covers provider/model/client/inference controls
- it maps naturally to the existing package placement under `steps/ai/settings`

The important point is that the old "step" name goes away completely.

## New Geppetto data model

### `InferenceSettings`

This is the renamed `StepSettings`.

It should continue to hold:

- provider API type
- model/engine name
- client timeouts
- base URL / API key references
- inference options such as temperature/max tokens/reasoning settings

It should not hold:

- system prompt
- middleware configuration
- tool filtering policy
- runtime keys/fingerprints

### `EngineProfile`

Proposed shape:

```go
type EngineProfile struct {
    Slug      EngineProfileSlug
    Stack     []EngineProfileRef
    Settings  InferenceSettingsSpec
    Metadata  map[string]any
}
```

Important rules:

- `Settings` must be explicit typed fields or a tightly scoped engine-settings schema
- no generic patch maps
- no section-name-based mutation
- no app runtime fields

### `EngineProfileRegistry`

Proposed shape:

```go
type EngineProfileRegistry struct {
    Slug               EngineRegistrySlug
    DefaultProfileSlug EngineProfileSlug
    Profiles           map[EngineProfileSlug]*EngineProfile
    Metadata           RegistryMetadata
}
```

This is conceptually similar to the existing profile registry, but the contents are engine-only.

### `ResolvedEngineProfile`

Proposed shape:

```go
type ResolvedEngineProfile struct {
    RegistrySlug         EngineRegistrySlug
    ProfileSlug          EngineProfileSlug
    StackTrace           []EngineProfileStackLayer
    EffectiveSettings    *InferenceSettings
    EngineProfileVersion uint64
}
```

The resolved object should give callers final engine settings and provenance, but not runtime prompt/tool/middleware data.

## New YAML format

### Current problem

Current profiles still look conceptually like:

```yaml
profiles:
  assistant:
    runtime:
      system_prompt: ...
      middlewares: ...
      tools: ...
```

That format should not survive in Geppetto core.

### Proposed engine profile YAML

Recommended direction:

```yaml
slug: team-models
default_profile_slug: gpt-5-mini

engine_profiles:
  gpt-5-mini:
    slug: gpt-5-mini
    settings:
      chat:
        api_type: openai-responses
        engine: gpt-5-mini
      client:
        timeout_seconds: 60

  claude-fast:
    slug: claude-fast
    settings:
      chat:
        api_type: claude
        engine: claude-3-5-haiku-latest
```

Alternative if you want to keep the current top-level key name:

```yaml
slug: team-models
profiles:
  gpt-5-mini:
    settings:
      ...
```

But I recommend `engine_profiles` because it forces the mental distinction.

### Hard-cut rules

The new YAML format must not include:

- `runtime:`
- `system_prompt`
- `middlewares`
- `tools`
- `step_settings_patch`

If any of those appear, decoding should fail with a validation error that points users to the migration playbook.

## New API shape

### Geppetto-side resolution API

Proposed API:

```go
func (r *Service) ResolveEngineProfile(
    ctx context.Context,
    in ResolveEngineProfileInput,
) (*ResolvedEngineProfile, error)
```

Where:

```go
type ResolveEngineProfileInput struct {
    RegistrySlug EngineRegistrySlug
    ProfileSlug  EngineProfileSlug
}
```

The resolved result should expose final `*InferenceSettings`.

### Engine construction

Rename:

```go
func NewEngineFromStepSettings(ss *StepSettings) (engine.Engine, error)
```

to one of:

```go
func NewEngineFromSettings(ss *InferenceSettings) (engine.Engine, error)
```

or:

```go
func NewEngineFromInferenceSettings(ss *InferenceSettings) (engine.Engine, error)
```

I recommend `NewEngineFromSettings(...)` once the type name is clear.

### Application-owned runtime composition

Applications should compose runtime separately:

```go
type AppRuntime struct {
    SystemPrompt string
    Middlewares  []middleware.Middleware
    ToolRegistry tools.ToolRegistry
    RuntimeKey   string
    Fingerprint  string
}
```

Then runtime assembly becomes:

```text
resolve engine profile -> final InferenceSettings -> build engine
resolve app runtime -> prompt + middleware + tools + runtime metadata
hand engine + app runtime pieces into runner/session layers
```

## Pseudocode and key flows

### Flow A: Engine profile resolution

```text
engine-profile registry YAML
  -> parse EngineProfileRegistry
  -> validate engine-only schema
  -> resolve default/selected profile
  -> merge stacked engine profiles
  -> materialize final InferenceSettings
  -> return ResolvedEngineProfile
```

Pseudocode:

```go
func ResolveEngineProfile(ctx context.Context, in ResolveEngineProfileInput) (*ResolvedEngineProfile, error) {
    reg := loadRegistry(in.RegistrySlug)
    prof := pickProfile(reg, in.ProfileSlug, reg.DefaultProfileSlug)
    stack := resolveProfileStack(reg, prof)
    merged := mergeEngineSettings(stack)
    settings := BuildInferenceSettings(merged)
    return &ResolvedEngineProfile{
        RegistrySlug:      reg.Slug,
        ProfileSlug:       prof.Slug,
        StackTrace:        stack,
        EffectiveSettings: settings,
    }, nil
}
```

### Flow B: Application runtime composition

```text
application request / config / state
  -> select app prompt
  -> select middlewares
  -> build filtered tool registry
  -> compute runtime identity
  -> call runner/session with engine + app runtime
```

Pseudocode:

```go
engProfile := gepprofiles.ResolveEngineProfile(...)
eng := enginefactory.NewEngineFromSettings(engProfile.EffectiveSettings)

appRuntime := ResolveAppRuntime(req)

runner.Run(ctx, runner.Runtime{
    Engine:      eng,
    Middlewares: appRuntime.Middlewares,
    ToolRegistry: appRuntime.ToolRegistry,
    RuntimeKey:  appRuntime.RuntimeKey,
})
```

### Flow C: Pinocchio JS after the redesign

```javascript
const gp = require("geppetto");
const pinocchio = require("pinocchio");

const engineProfile = pinocchio.engines.resolveProfile({ profile: "gpt-5-mini" });
const engineInfo = pinocchio.engines.inspectProfile(engineProfile);
const engine = pinocchio.engines.fromProfile(engineProfile);

const runtime = pinocchio.runtime.resolve({
  systemPrompt: "...",
  toolNames: ["calc"],
});

const out = gp.runner.run({
  engine,
  runtime,
  prompt: "Hello",
});
```

This example is illustrative, not a required first API shape. The key point is the split between engine profile resolution and app runtime resolution.

## Design Decisions

### Decision 1: Hard cut, no wrappers

Rationale:

- the old and new models encode different architectural assumptions
- compatibility wrappers would preserve confusion longer
- previous migrations in this ecosystem have already used hard-cut playbooks successfully

Consequence:

- rename symbols directly
- reject old YAML directly
- update downstream apps directly
- provide migration docs, not compatibility layers

### Decision 2: Rename `StepSettings` directly to `InferenceSettings`

Rationale:

- the current name is legacy terminology
- keeping the old type name would weaken the conceptual cleanup
- if the type is important enough to keep, it is important enough to name correctly

### Decision 3: Remove runtime behavior from Geppetto profiles

Rationale:

- prompt/middleware/tool behavior is application-owned
- those concepts vary by host application
- keeping them in Geppetto core encourages the wrong abstraction boundary

### Decision 4: Keep engine profiles in Geppetto

Rationale:

- engine configuration is still shared infrastructure
- provider/model/client configuration belongs to Geppetto
- applications should not have to reinvent registry loading and profile stacking for engine settings

## Alternatives Considered

### Alternative A: restore `StepSettingsPatch`

Rejected because:

- patch maps were hard to reason about
- typing and validation happened too late
- the old design mixed unrelated concerns

### Alternative B: keep the current post-GP-43 model and push everything to apps

Rejected because:

- it gives applications too much engine-configuration responsibility
- it removes a legitimately reusable Geppetto concern
- it makes shared engine presets harder to manage

### Alternative C: keep current `Profile` shape but add a separate engine section

Rejected because:

- it still keeps engine and runtime behavior in one document
- the word "profile" stays overloaded
- it makes migrations and ownership less obvious

## Implementation Plan

## Phase 1: Introduce new terminology and target model

1. Rename `StepSettings` to `InferenceSettings` in Geppetto.
2. Rename constructors and engine factory entrypoints.
3. Update docs, tests, and examples.

Files likely affected:

- [settings-step.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/settings-step.go)
- [api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go)
- [sections.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go)
- all references found via `rg -n "StepSettings|NewStepSettings|NewEngineFromStepSettings"`

## Phase 2: Add new engine profile model in Geppetto

1. Add `EngineProfile`, `EngineProfileRegistry`, and `ResolvedEngineProfile`.
2. Add validation and YAML codec support.
3. Add stack merge rules for engine-only settings.

Files likely added/changed:

- `geppetto/pkg/engineprofiles/engine_*`
- `geppetto/pkg/engineprofiles/types.go`
- `geppetto/pkg/engineprofiles/validation.go`
- `geppetto/pkg/engineprofiles/service.go`

## Phase 3: Remove mixed runtime behavior from Geppetto profiles

1. Delete `RuntimeSpec` from the Geppetto profile model.
2. Delete runtime merge logic from profile resolution.
3. Delete profile-runtime fingerprinting from Geppetto core.
4. Delete JS runtime/profile materialization paths that depend on `EffectiveRuntime`.

Files likely affected:

- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/types.go)
- [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/service.go)
- [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/stack_merge.go)
- [api_runtime_metadata.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go)
- [api_runner.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runner.go)

## Phase 4: Move runtime behavior to applications

1. Pinocchio defines its own runtime config model.
2. Pinocchio JS / webchat use Geppetto engine profiles only for engine construction.
3. Prompt/middleware/tool/runtime-key logic stays Pinocchio-owned.

Likely Pinocchio files:

- [profile_runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go)
- [module.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/js/modules/pinocchio/module.go)
- [js.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/cmds/js.go)

## Phase 5: Publish migration material

1. Add a dedicated migration playbook under `glazed/pkg/doc/...`
2. Make the playbook hard-cut and exhaustive
3. Link it from Geppetto and downstream docs

Suggested playbook content:

- old mixed profile YAML vs new engine profile YAML
- symbol rename map:
  - `StepSettings` -> `InferenceSettings`
  - `NewStepSettings` -> `NewInferenceSettings`
  - etc.
- where prompt/middlewares/tools must move in app code
- common migration pitfalls

## Testing Strategy

### Geppetto

- unit tests for new engine profile validation
- unit tests for engine profile stack resolution
- YAML decode/encode tests
- engine factory tests after the rename

### JS

- update JS tests to stop depending on `EffectiveRuntime`
- test new engine-profile-facing JS APIs only if such APIs are added

### Pinocchio

- command tests for engine profile loading
- JS command smoke tests
- runtime resolver tests to ensure app-owned runtime behavior still works

### Docs

- `docmgr doctor` on the ticket
- help/doc validation for any new migration playbook

## Risks, Alternatives, And Open Questions

### Risk 1: `InferenceSettings` rename is broad

The rename will touch a large number of packages and docs. This is acceptable, but it should be done as a deliberate sweep rather than piecemeal.

### Risk 2: `EngineProfile` could regress into patch-shaped config

If the implementation reintroduces generic patch maps, the redesign will fail. The new model must stay typed or narrowly schema-defined.

### Risk 3: downstream apps may have hidden dependencies on `RuntimeSpec`

The migration should assume some applications still mentally rely on Geppetto-owned runtime behavior even if the code already partially split away from it.

### Open question 1

Should the YAML key be `profiles:` or `engine_profiles:`?

My recommendation is `engine_profiles:` because it forces clearer meaning during the cutover.

### Open question 2

Should Geppetto expose a shared helper for app runtime assembly?

My recommendation is no for the first cut. Keep runtime composition app-owned until genuine duplication appears.

### Open question 3

Should `InferenceSettings` stay in the current package path?

Short term: yes, likely keep the package path stable even if the type name changes.

Long term: maybe move it to a less legacy path, but that is secondary to the conceptual cleanup.

## References

- [settings-step.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/settings-step.go)
- [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/service.go)
- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/types.go)
- [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/stack_merge.go)
- [api_runtime_metadata.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go)
- [api_runner.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runner.go)
- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/runner/types.go)
- [profile_runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go)
- [module.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/js/modules/pinocchio/module.go)
- [js.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/cmds/js.go)
- [migrating-to-facade-packages.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/glazed/pkg/doc/tutorials/migrating-to-facade-packages.md)
