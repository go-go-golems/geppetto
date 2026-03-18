---
Title: 'Remove RuntimeKeyFallback from geppetto: analysis, design, and implementation guide'
Ticket: GP-44-REMOVE-RUNTIMEKEYFALLBACK
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/examples/js/geppetto/09_profiles_resolve_stack_precedence.js
      Note: Example still prints resolved runtimeKey.
    - Path: geppetto/pkg/doc/topics/13-js-api-reference.md
      Note: JS reference still documents runtimeKey as a supported option.
    - Path: geppetto/pkg/js/modules/geppetto/api_engines.go
      Note: |-
        engines.fromProfile JS binding.
        JS engines.fromProfile runtimeKey handling without engine impact
    - Path: geppetto/pkg/js/modules/geppetto/api_profiles.go
      Note: |-
        profiles.resolve JS binding.
        JS profiles.resolve runtimeKeyFallback handling
    - Path: geppetto/pkg/js/modules/geppetto/module_test.go
      Note: |-
        JS test coverage that still asserts runtimeKey behavior.
        Tests that currently preserve runtimeKey semantics
    - Path: geppetto/pkg/profiles/registry.go
      Note: |-
        Public resolver request and response types.
        Defines ResolveInput and ResolvedProfile RuntimeKey surface
    - Path: geppetto/pkg/profiles/service.go
      Note: |-
        Only implementation that reads RuntimeKeyFallback and emits RuntimeKey.
        Contains the only RuntimeKeyFallback behavior
    - Path: geppetto/pkg/profiles/source_chain.go
      Note: Pass-through wrapper over ResolveInput for chained registry lookups.
ExternalSources: []
Summary: Evidence-backed guide for removing RuntimeKeyFallback from geppetto, including resolver architecture, API impact, phased implementation plan, test strategy, and documentation cleanup.
LastUpdated: 2026-03-17T16:50:00-04:00
WhatFor: Use this document to implement or review the removal of RuntimeKeyFallback without accidentally changing profile selection or engine behavior.
WhenToUse: Use when onboarding to the profile resolver, implementing the RuntimeKeyFallback cleanup, or auditing the JS profile APIs.
---


# Remove RuntimeKeyFallback from geppetto: analysis, design, and implementation guide

## Executive Summary

`RuntimeKeyFallback` is part of the profile-resolution API, but not part of the actual profile-resolution algorithm. In the current implementation, the resolver uses it only to populate `ResolvedProfile.RuntimeKey`; if the field is absent, the resolver derives the runtime key from the selected profile slug. No other code path in `geppetto/` uses this value to decide which registry to read, which profile to select, how to merge a profile stack, how to build `EffectiveStepSettings`, how to compute the runtime fingerprint, or how to instantiate an engine.

That makes `RuntimeKeyFallback` a good hard-cut cleanup candidate. It is still present in Go types, JS input handling, JS typings, tests, examples, and docs, but the behavior it controls is just an output label. The recommended change is to remove it entirely from the resolver request and response APIs, then clean up all call sites and docs in one focused change.

For a new intern, the key insight is this: the real unit of runtime selection in the profile system is `registry slug + profile slug + stack precedence`, not `runtime key`. `RuntimeKeyFallback` exists next to the system, not in the center of it.

## Problem Statement

The profile resolver exposes more surface area than the active runtime model actually needs. `ResolveInput` currently includes `RuntimeKeyFallback`, and `ResolvedProfile` currently includes `RuntimeKey`, but the resolver uses that input only as a label-selection shortcut:

- `ResolveInput` defines `RuntimeKeyFallback` in [registry.go:17-24](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go#L17).
- `ResolvedProfile` includes `RuntimeKey` in [registry.go:26-37](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go#L26).
- `ResolveEffectiveProfile` assigns `runtimeKey := in.RuntimeKeyFallback`, then falls back to `ParseRuntimeKey(profileSlug.String())` when the field is empty in [service.go:153-183](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L153).

The question for this ticket is not whether the existing code works. It does. The question is whether it adds meaningful capability. The answer, based on the repository evidence, is no.

The cost of keeping it is spread across:

- public Go API surface,
- JS API surface,
- TypeScript declarations,
- tests that protect output-only labeling behavior,
- examples that teach the field as if it were important,
- docs that encourage consumers to think in terms of `runtimeKey`.

That is exactly the kind of low-value machinery that makes architecture harder for new engineers to understand.

## Scope

### In scope

- Remove `ResolveInput.RuntimeKeyFallback`.
- Remove `ResolvedProfile.RuntimeKey`.
- Remove JS `profiles.resolve(...).runtimeKeyFallback`.
- Remove JS `profiles.resolve(...).runtimeKey`.
- Remove JS `engines.fromProfile(...).runtimeKey`.
- Update tests, examples, docs, and type declarations.

### Out of scope

- Changing registry-stack precedence behavior.
- Changing profile-slug fallback behavior when a registry default profile is selected.
- Changing `RuntimeFingerprint`.
- Changing turn metadata formats outside any references that currently mention runtime keys.
- Larger legacy cleanup work captured in GP-45.

## Current-State Architecture

### Resolver entrypoint

Applications enter the profile resolver through the `profiles.RegistryReader` interface in [registry.go:39-45](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go#L39). The main method is:

```go
ResolveEffectiveProfile(ctx context.Context, in ResolveInput) (*ResolvedProfile, error)
```

`StoreRegistry` is the default implementation in [service.go:17-22](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L17). `ChainedRegistry` is a wrapper that resolves profile slugs by stack precedence and then forwards to the aggregate store-backed resolver in [source_chain.go:37-45](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go#L37) and [source_chain.go:201-221](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go#L201).

### What the resolver actually does

The actual resolution sequence in `ResolveEffectiveProfile` is:

1. Resolve the registry slug.
2. Load the registry.
3. Resolve the profile slug, including registry-default fallback.
4. Expand the profile stack.
5. Merge stack layers.
6. Produce `EffectiveRuntime`.
7. Apply the runtime step-settings patch to any base step settings.
8. Build provenance metadata and runtime fingerprint.
9. Return `ResolvedProfile`.

In pseudocode, the live behavior is:

```go
func ResolveEffectiveProfile(in ResolveInput) ResolvedProfile {
    registry := loadRegistry(resolveRegistrySlug(in.RegistrySlug))
    profileSlug := resolveProfileSlugForRegistry(in.ProfileSlug, registry)
    layers := ExpandProfileStack(registry, profileSlug)
    mergedRuntime, trace := MergeProfileStackLayersWithTrace(layers)
    effectiveStepSettings := ApplyRuntimeStepSettingsPatch(in.BaseStepSettings, mergedRuntime.StepSettingsPatch)
    runtimeFingerprint := fingerprint(registry, layers, mergedRuntime, effectiveStepSettings)

    // Detached label logic
    runtimeKey := in.RuntimeKeyFallback
    if runtimeKey == zero {
        runtimeKey = ParseRuntimeKey(profileSlug.String())
    }

    return ResolvedProfile{
        RegistrySlug: registrySlug,
        ProfileSlug: profileSlug,
        RuntimeKey: runtimeKey,
        EffectiveRuntime: mergedRuntime,
        EffectiveStepSettings: effectiveStepSettings,
        RuntimeFingerprint: runtimeFingerprint,
        Metadata: provenance,
    }
}
```

The important architectural observation is that the real algorithm ends before the runtime-key label code begins.

### JS entrypoints that still expose it

Two JS surfaces still expose runtime-key inputs:

1. `gp.profiles.resolve(...)` accepts both `runtimeKeyFallback` and `runtimeKey` aliases in [api_profiles.go:155-169](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go#L155).
2. `gp.engines.fromProfile(..., opts)` accepts `runtimeKey` in [api_engines.go:248-258](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go#L248).

`profiles.resolve` returns the runtime key because `encodeResolvedProfile` copies `resolved.RuntimeKey.String()` into the JS object in [api_profiles.go:31-46](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go#L31).

`engines.fromProfile`, however, does not preserve runtime key in engine metadata. Its metadata payload only includes:

- `profileRegistry`
- `profileSlug`
- `runtimeFingerprint`
- `resolvedMetadata`

See [api_engines.go:275-283](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go#L275).

This is the strongest evidence that runtime key is not part of meaningful runtime behavior.

### Examples and tests that keep the field alive

The field survives mostly because examples and tests still mention it:

- JS test assertions around `runtimeKeyFallback` and `resolved.runtimeKey` in [module_test.go:813-820](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module_test.go#L813).
- More stack-resolution JS tests passing `runtimeKeyFallback` in [module_test.go:893-896](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module_test.go#L893).
- Example code showing `resolved.runtimeKey` in [09_profiles_resolve_stack_precedence.js:3-21](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/09_profiles_resolve_stack_precedence.js#L3).
- Docs describing `opts.runtimeKey` in [13-js-api-reference.md:129-144](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md#L129).

In other words, the field is preserved by documentation gravity more than by runtime necessity.

## Evidence That RuntimeKeyFallback Is Not Functionally Necessary

### It does not participate in profile selection

Registry selection and profile lookup are driven by:

- `RegistrySlug`,
- `ProfileSlug`,
- registry default profile fallback,
- chain precedence when a profile is looked up without an explicit registry.

That logic lives in [service.go:121-145](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L121) and [source_chain.go:201-221](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go#L201). `RuntimeKeyFallback` is not read anywhere in that selection path.

### It does not participate in runtime composition

The stack merge logic in [stack_merge.go:11-76](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_merge.go#L11) produces:

- merged `StepSettingsPatch`,
- merged `SystemPrompt`,
- merged `Middlewares`,
- merged `Tools`,
- merged `Extensions`.

There is no runtime-key field in `RuntimeSpec`. This matters because it means the field is not an attribute of stored profile runtime data.

### It does not participate in engine construction

`engineFromResolvedProfile` clones `resolved.EffectiveStepSettings`, applies provider defaults, and calls `enginefactory.NewEngineFromStepSettings(ss)` in [api_engines.go:264-272](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go#L264). No code path there reads `ResolvedProfile.RuntimeKey`.

### It does not participate in runtime fingerprinting

`runtimeFingerprint(...)` hashes registry slug, profile metadata, stack lineage, merged runtime payload, and step metadata in [service.go:224-249](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L224). Runtime key is absent from the fingerprint payload. That means deleting the field should not change fingerprint stability for identical runtime inputs.

## Problem With Keeping It

The main harm is conceptual. A new engineer sees `RuntimeKeyFallback` and reasonably assumes:

- there is a runtime registry keyed by runtime key,
- runtime key might control which model/provider is selected,
- runtime key might feed into cache or provenance identity,
- runtime key might be the app-facing stable selector while profile slug is an internal implementation detail.

None of those are true in the current code.

The field also creates a false API taxonomy:

```text
registrySlug -> real selector
profileSlug -> real selector
runtimeKeyFallback -> looks like selector, is not selector
```

That is exactly the kind of ambiguity that slows down onboarding and increases the chance of wrong architectural changes.

## Proposed Solution

### Design goal

Make profile resolution explicitly profile-driven:

- input: registry slug, profile slug, base step settings
- output: selected registry/profile, effective runtime, effective step settings, runtime fingerprint, metadata

No separate runtime-key label should survive in the resolver contract.

### Target API shape

Proposed Go API:

```go
type ResolveInput struct {
    RegistrySlug      RegistrySlug
    ProfileSlug       ProfileSlug
    BaseStepSettings  *settings.StepSettings
}

type ResolvedProfile struct {
    RegistrySlug          RegistrySlug
    ProfileSlug           ProfileSlug
    EffectiveStepSettings *settings.StepSettings
    EffectiveRuntime      RuntimeSpec
    RuntimeFingerprint    string
    Metadata              map[string]any
}
```

Proposed JS behavior:

- `gp.profiles.resolve(...)` accepts only resolver-relevant inputs.
- returned JS object no longer contains `runtimeKey`.
- `gp.engines.fromProfile(..., opts)` rejects no new cases, but silently loses support for `opts.runtimeKey` because it is no longer meaningful.

### Implementation note

This should be a hard cut, not a deprecation shim. The repository already uses hard-cut cleanup language in multiple places, and keeping a compatibility alias for an output-only label would extend the migration tail without preserving real value.

## Detailed Implementation Plan

### Phase 1: Remove core resolver fields

Files:

- [pkg/profiles/registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go)
- [pkg/profiles/service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go)

Steps:

1. Remove `RuntimeKeyFallback` from `ResolveInput`.
2. Remove `RuntimeKey` from `ResolvedProfile`.
3. Delete the local runtime-key assignment block in `ResolveEffectiveProfile`.
4. Return the rest of the resolved payload unchanged.

Pseudocode:

```go
// delete:
runtimeKey := in.RuntimeKeyFallback
if runtimeKey.IsZero() {
    runtimeKey = ParseRuntimeKey(profileSlug.String())
}

// and remove RuntimeKey: runtimeKey from return payload
```

### Phase 2: Remove JS input and output surface

Files:

- [pkg/js/modules/geppetto/api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go)
- [pkg/js/modules/geppetto/api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go)

Steps:

1. Remove runtime-key parsing from `profiles.resolve`.
2. Remove `runtimeKey` from `encodeResolvedProfile`.
3. Remove runtime-key parsing from `engines.fromProfile`.
4. Keep the rest of the flow intact.

New `profiles.resolve` pseudocode:

```go
func profilesResolve(opts map[string]any) JSValue {
    in := ResolveInput{}
    in.RegistrySlug = parseOptionalRegistrySlug(opts["registrySlug"])
    in.ProfileSlug = parseOptionalProfileSlug(opts["profileSlug"])
    if requestOverrides present {
        throw "removed"
    }
    resolved := registry.ResolveEffectiveProfile(ctx, in)
    return encodeResolvedProfileWithoutRuntimeKey(resolved)
}
```

### Phase 3: Update tests

Files:

- [pkg/profiles/service_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service_test.go)
- [pkg/js/modules/geppetto/module_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module_test.go)

Steps:

1. Remove assertions about `resolved.RuntimeKey`.
2. Remove JS inputs that pass `runtimeKeyFallback`.
3. Remove JS assertions that check `resolved.runtimeKey`.
4. Retain tests that prove the real behavior:
   - registry resolution,
   - profile resolution,
   - fingerprint presence,
   - stack precedence,
   - engine construction from step settings.

### Phase 4: Update types, examples, and docs

Files:

- [pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl)
- [pkg/doc/types/geppetto.d.ts](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/types/geppetto.d.ts)
- [examples/js/geppetto/09_profiles_resolve_stack_precedence.js](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/09_profiles_resolve_stack_precedence.js)
- [pkg/doc/topics/13-js-api-reference.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md)
- [pkg/doc/topics/14-js-api-user-guide.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/14-js-api-user-guide.md)
- [pkg/doc/topics/01-profiles.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/01-profiles.md)

Steps:

1. Remove runtime-key fields from typings.
2. Rewrite examples to show only registry/profile/fingerprint outputs.
3. Update docs to explain that profile resolution is registry-first and profile-first, without a separate runtime-key selector.

## Validation Strategy

### Required tests

Run:

```bash
go test ./pkg/profiles ./pkg/js/modules/geppetto
```

### Required grep validation

Run:

```bash
rg -n "RuntimeKeyFallback|runtimeKeyFallback|runtimeKey" geppetto/pkg geppetto/examples/js geppetto/pkg/doc
```

Expected result:

- no remaining `RuntimeKeyFallback`,
- no JS resolver/engine runtime-key option docs,
- any remaining `runtime_key` references should belong only to turn metadata compatibility code, not profile resolution.

### Behavior checks for reviewers

After implementation, reviewers should verify:

1. `profiles.resolve({ profileSlug: "assistant" })` still selects the correct profile.
2. `engines.fromProfile("assistant")` still builds the same engine as before.
3. `runtimeFingerprint` is still populated.
4. `profile.stack.lineage` and `profile.stack.trace` are unchanged.

## Risks

### Risk 1: Hidden downstream consumer outside this repo

The main real risk is an out-of-repo consumer that reads `ResolvedProfile.RuntimeKey` as an output-only label. The repository evidence says no in-repo runtime flow depends on it, but a downstream app could still display it.

Mitigation:

- do one repo-wide and mono-repo-wide search before landing,
- mention the API removal explicitly in changelog or PR notes.

### Risk 2: Docs drift after code removal

The repository already has documentation drift around removed resolver features. If implementation lands without a coordinated doc cleanup, the old concept will linger in onboarding material.

Mitigation:

- treat docs and typings as part of the same change,
- fail the work item if examples still mention runtime key.

## Alternatives Considered

### Alternative A: Keep the field but deprecate it

Rejected because:

- it is already low-value,
- deprecation would preserve mental overhead,
- the codebase is already moving through hard-cut cleanup patterns.

### Alternative B: Keep `RuntimeKey` as output but remove `RuntimeKeyFallback`

This is viable, but not preferred. The output field would still suggest a meaningful runtime identity layer even though it would just mirror the selected profile slug. If the field is not carrying distinct meaning, removing both input and output is cleaner.

### Alternative C: Rename runtime key to profile label

Rejected because the resolver already has the real selected identity: `ProfileSlug`. Adding another synonym would worsen API semantics.

## Open Questions

1. Do any downstream repositories outside `geppetto/` render `ResolvedProfile.RuntimeKey` in UI or logs?
2. Is there any product requirement for a human-facing label distinct from profile slug? If yes, that should likely be handled through `Profile.DisplayName`, not through resolver input.

## Reference Map For A New Intern

If you are new to this codebase, read in this order:

1. [pkg/profiles/registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go) to learn the public resolver API.
2. [pkg/profiles/service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go) to see the actual resolution algorithm.
3. [pkg/profiles/source_chain.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go) to understand stack-precedence lookup.
4. [pkg/js/modules/geppetto/api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go) for the JS resolver surface.
5. [pkg/js/modules/geppetto/api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go) for profile-to-engine wiring.
6. [pkg/js/modules/geppetto/module_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module_test.go) for concrete usage examples.

## References

- [pkg/profiles/registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go)
- [pkg/profiles/service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go)
- [pkg/profiles/source_chain.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go)
- [pkg/profiles/service_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service_test.go)
- [pkg/js/modules/geppetto/api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go)
- [pkg/js/modules/geppetto/api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go)
- [pkg/js/modules/geppetto/module_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module_test.go)
- [examples/js/geppetto/09_profiles_resolve_stack_precedence.js](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/09_profiles_resolve_stack_precedence.js)
- [pkg/doc/topics/13-js-api-reference.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md)
