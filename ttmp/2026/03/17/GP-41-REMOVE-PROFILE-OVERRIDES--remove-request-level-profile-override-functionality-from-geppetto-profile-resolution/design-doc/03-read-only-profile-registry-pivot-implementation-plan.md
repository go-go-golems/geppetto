---
Title: 'GP-41 pivot: read-only profile registry implementation plan'
Ticket: GP-41-REMOVE-PROFILE-OVERRIDES
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - pinocchio
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/registry.go
      Note: Registry interface currently mixes read and write responsibilities
    - Path: geppetto/pkg/profiles/service.go
      Note: StoreRegistry still exposes mutation methods that this pivot removes
    - Path: geppetto/pkg/profiles/source_chain.go
      Note: ChainedRegistry currently supports routed writes and refresh logic
    - Path: geppetto/pkg/js/modules/geppetto/api_profiles.go
      Note: JS profile namespace still exposes mutation APIs
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Module runtime still carries writer-specific fields
    - Path: pinocchio/pkg/webchat/http/profile_api.go
      Note: Pinocchio currently exposes profile CRUD/default endpoints on top of Geppetto mutation APIs
ExternalSources: []
Summary: Implementation plan for the GP-41 pivot that makes profile registries read-only, removes profile mutation APIs from Geppetto core and JS, and treats mutability as a future app-owned layer if it is ever needed again.
LastUpdated: 2026-03-17T15:55:00-04:00
WhatFor: Use this document to execute the GP-41 pivot from override removal toward a fully read-only registry service.
WhenToUse: Use when implementing the next GP-41 commits, reviewing the scope change, or onboarding someone into the new read-only registry direction.
---

# GP-41 pivot: read-only profile registry implementation plan

## Executive Summary

GP-41 started as a ticket to remove request-level profile overrides. During implementation, the work exposed a larger simplification opportunity: the Geppetto profile registry abstraction is doing both read-time runtime resolution and write-time profile mutation, even though current downstream usage is overwhelmingly read-oriented. The user-directed pivot is to remove mutation APIs from the registry layer entirely and keep profile registries read-only.

That means the new target architecture is:

- Geppetto profile registries are read-only.
- Geppetto profile resolution is read-only.
- Geppetto JS profile APIs are read-only.
- Pinocchio and any other app code should not expose CRUD/defaulting endpoints that depend on the Geppetto registry layer.
- If writable registries are ever needed again, they should be added as a separate app-owned layer on top of store primitives rather than being part of the default registry abstraction.

This plan subsumes the earlier Phase 2 idea of shrinking `PolicySpec` to just `read_only`. Once profile mutation leaves the registry service entirely, `PolicySpec` itself no longer has a purpose and should be deleted.

## Why this pivot is the right simplification

Removing only `request_overrides` still leaves a surprising amount of complexity behind:

- `PolicySpec` exists mostly to control mutation or overrides.
- `StoreRegistry` still exposes CRUD and default-profile mutation paths.
- `ChainedRegistry` still owns routed write behavior and refresh logic.
- the JS `profiles` namespace still exposes mutation methods and mutation option types.
- Pinocchio still mirrors those mutation endpoints in its profile HTTP API.

The underlying product reality is much simpler:

- apps load profile registries,
- apps resolve profiles,
- apps build engines from resolved runtime data,
- apps do not currently depend on end-user profile editing as a core product workflow.

The clean design is to make the registry abstraction match that reality.

## Target architecture

### Before

```text
ProfileStore (mutable persistence)
  -> StoreRegistry (read + write + resolve)
  -> ChainedRegistry (read + routed write + refresh)
  -> JS module (read + write APIs)
  -> Pinocchio profile API (read + write endpoints)
```

### After

```text
ProfileStore (mutable persistence primitives may still exist internally)
  -> StoreRegistry (read + resolve only)
  -> ChainedRegistry (read + resolve only)
  -> JS module (read-only profile APIs)
  -> Pinocchio profile API (read-only profile endpoints, or removed entirely if unused)
```

## Scope of the pivot

In scope:

- remove `ProfilePatch`, `WriteOptions`, and `RegistryWriter` from Geppetto `pkg/profiles`
- remove `CreateProfile`, `UpdateProfile`, `DeleteProfile`, and `SetDefaultProfile` from `StoreRegistry`
- remove routed-write behavior from `ChainedRegistry`
- remove JS mutation APIs from `pkg/js/modules/geppetto`
- remove `PolicySpec` and `policy` fields now that mutation policy has no remaining job
- remove downstream code in Pinocchio and tests in GEC-RAG that only exist to satisfy the old writable registry interface

Out of scope:

- removing low-level mutable store primitives such as `UpsertProfile` from store backends
- removing profile resolution itself
- removing profile registry loading from YAML or SQLite
- removing `runtime.step_settings_patch` in this ticket

## Implementation slices

### Slice 1: Geppetto core registry becomes read-only

Files:

- [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go)
- [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go)
- [source_chain.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go)
- [errors.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/errors.go)

Changes:

- delete `ProfilePatch`
- delete `WriteOptions`
- delete `RegistryWriter`
- make `Registry` equal to the read-only surface
- delete `StoreRegistry.CreateProfile`
- delete `StoreRegistry.UpdateProfile`
- delete `StoreRegistry.DeleteProfile`
- delete `StoreRegistry.SetDefaultProfile`
- delete `ChainedRegistry` write-routing methods and owner refresh logic
- remove `ErrPolicyViolation` and `PolicyViolationError`

Why first:

- this is the architectural boundary change
- downstream cleanup becomes mechanical once the core writer surface is gone

### Slice 2: Delete policy from the profile domain model

Files:

- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go)
- [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_merge.go)
- [stack_trace.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_trace.go)
- [validation.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/validation.go)

Changes:

- delete `PolicySpec`
- delete `Profile.Policy`
- delete `policy` fields from stack merge outputs and trace outputs
- remove policy-related validation and tests

Why second:

- mutation policy only makes sense while mutation exists on the registry surface
- once Slice 1 lands, `policy` becomes dead schema

### Slice 3: Remove JS mutation APIs

Files:

- [module.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go)
- [api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go)
- [geppetto.d.ts.tmpl](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl)
- [geppetto.d.ts](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/types/geppetto.d.ts)

Changes:

- remove `ProfileRegistryWriter` from module options/runtime state
- remove `profiles.createProfile`
- remove `profiles.updateProfile`
- remove `profiles.deleteProfile`
- remove `profiles.setDefaultProfile`
- remove mutation-related TypeScript declarations
- keep read-only methods such as:
  - `listRegistries`
  - `getRegistry`
  - `listProfiles`
  - `getProfile`
  - `resolve`
  - `connectStack`
  - `disconnectStack`
  - `getConnectedSources`

### Slice 4: Remove downstream writable API layers

Files:

- [profile_api.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/http/profile_api.go)
- Pinocchio tests covering profile CRUD/default routes
- GEC-RAG fake registries that still implement writer methods in tests

Changes:

- remove Pinocchio profile CRUD/default endpoints or reduce them to read-only endpoints only
- remove downstream error handling for `PolicyViolationError`
- simplify fake registries and tests to satisfy only the read-only registry interface

### Slice 5: Documentation and examples cleanup

Files:

- Geppetto docs and JS examples that still mention CRUD or policy
- Pinocchio docs describing writable profile APIs
- ticket docs updated to reflect the completed pivot

Changes:

- delete mutation examples that no longer exist
- rewrite docs to describe registries as read-only
- document that writable profile editing, if reintroduced later, should live above the store layer

## Practical pseudocode for the new boundary

### New registry interface

```go
type Registry interface {
	ListRegistries(ctx context.Context) ([]RegistrySummary, error)
	GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*ProfileRegistry, error)
	ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error)
	GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, error)
	ResolveEffectiveProfile(ctx context.Context, in ResolveInput) (*ResolvedProfile, error)
}
```

### Read-only JS namespace

```ts
interface ProfilesNamespace {
  listRegistries(): RegistrySummary[]
  getRegistry(registrySlug?: string): ProfileRegistry
  listProfiles(registrySlug?: string): Profile[]
  getProfile(profileSlug: string, registrySlug?: string): Profile
  resolve(input?: ResolveInput): ResolvedProfile
  connectStack(sources: ProfileRegistrySources): ConnectedProfileStack
  disconnectStack(): void
  getConnectedSources(): string[]
}
```

## Risks to watch

- Pinocchio may currently assume writable profile routes exist, even if real product users are not depending on them.
- Some tests may use writable registries as a convenient setup tool; those should shift to direct store seeding or static fixtures.
- Documentation drift is likely because the writable path exists in several separate guides.

## Recommended commit boundaries

1. `make profile registry service read-only`
2. `remove policy from geppetto profile model`
3. `remove geppetto js profile mutation apis`
4. `remove pinocchio writable profile api`
5. `clean docs and tests for read-only registries`

## Reviewer checklist

- Does `profiles.Registry` expose only read operations?
- Are all `PolicySpec` and `policy` fields removed from the public profile model?
- Does the JS API expose only read-only profile methods?
- Are Pinocchio and GEC-RAG no longer depending on writer methods or policy-specific errors?
- Do the remaining docs consistently describe registries as read-only?
