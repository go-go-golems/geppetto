---
Title: 'Legacy and unused geppetto functionality: cleanup analysis and implementation guide'
Ticket: GP-45-REMOVE-LEGACY-UNUSED-FUNCTIONALITY
Status: active
Topics:
    - geppetto
    - architecture
    - cleanup
    - migration
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/01-profiles.md
      Note: Docs drift around removed policy and override features.
    - Path: geppetto/pkg/inference/engine/run_with_result.go
      Note: |-
        Legacy scalar inference metadata mirroring.
        Legacy scalar inference metadata mirroring
    - Path: geppetto/pkg/profiles/adapters.go
      Note: |-
        Thin string-wrapper helpers with no meaningful production usage found.
        Likely unused adapter helpers
    - Path: geppetto/pkg/profiles/extensions.go
      Note: |-
        Extension normalization and registry interfaces.
        Lightly integrated extension normalization
    - Path: geppetto/pkg/profiles/middleware_extensions.go
      Note: |-
        Middleware config projection into typed extensions.
        Likely test-only middleware extension projection
    - Path: geppetto/pkg/profiles/source_chain.go
      Note: Chained registry implementation with some apparently unused fields.
    - Path: geppetto/pkg/profiles/stack_trace.go
      Note: |-
        Always-on debug path tracing.
        Always-on profile stack trace generation
    - Path: geppetto/pkg/sections/profile_registry_source.go
      Note: |-
        Migration-era profile loading bridge.
        Migration shim for profile loading
    - Path: geppetto/pkg/sections/sections.go
      Note: Bootstrap logic and compatibility comments around profile settings.
    - Path: geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go
      Note: |-
        Legacy runtime metadata shape normalization.
        Legacy runtime metadata shape normalization
ExternalSources: []
Summary: Detailed inventory and phased cleanup plan for legacy support, unused helpers, and over-complex subsystems found during the geppetto profile-runtime review.
LastUpdated: 2026-03-17T16:55:00-04:00
WhatFor: Use this document to decide what legacy or low-value machinery can be safely removed from geppetto and in what order.
WhenToUse: Use when planning cleanup work after GP-44 or when onboarding a new engineer to geppetto architecture and technical debt.
---


# Legacy and unused geppetto functionality: cleanup analysis and implementation guide

## Executive Summary

The geppetto repository contains three different kinds of cleanup candidates:

1. Clearly intentional backward-compatibility support that preserves older shapes or APIs.
2. Machinery that appears lightly integrated or test-only from current in-repo usage.
3. Features that may be conceptually useful, but are currently implemented in a way that is more complex than their demonstrated application value.

This document organizes those findings into a practical cleanup plan. It does not assume that every suspicious subsystem should be deleted immediately. Instead, it teaches the relevant architecture first, then classifies each candidate by confidence and risk so a new intern can tell the difference between:

- "delete now",
- "delete after downstream grep",
- "keep, but simplify or gate behind debug mode",
- "docs are wrong even if code stays."

The highest-confidence findings are:

- legacy runtime metadata normalization in `runtimeattrib`,
- legacy scalar inference metadata mirroring in `run_with_result`,
- migration-shim profile loading and help text,
- stale docs that still describe removed override policy and request-time mutation,
- thin wrapper adapters in `pkg/profiles/adapters.go`.

The strongest medium-confidence findings are:

- extension normalization and codec plumbing that looks only lightly integrated,
- middleware-extension projection helpers that appear test-only,
- always-on stack trace generation that may be more debug payload than runtime code really needs,
- extra `ChainedRegistry` fields that appear unused after construction.

## Problem Statement

Architecture cleanup is easy to get wrong when "legacy", "unused", and "over-complex" are treated as if they are the same thing. They are not.

In this repository:

- some code clearly exists to preserve old behavior during migration,
- some code is still documented as active even though the implementation has already hard-cut it away,
- some subsystems seem to have been designed for extensibility that current applications do not visibly exploit,
- some fields and helper types are simply vestigial.

If all of these are lumped into one bucket, engineers either delete too much at once or avoid cleanup entirely because the work feels risky. The right approach is to separate them.

## Scope

### In scope

- Backward-compatibility support in runtime metadata and inference metadata.
- Migration-era shims around profile loading and settings bootstrapping.
- Unused or minimally useful helper layers.
- Documentation drift that teaches removed behavior as current behavior.
- Complexity hotspots where the implementation looks larger than the demonstrated use.

### Out of scope

- Large-scale replacement of the profile system.
- Removing extension support solely because current usage is unclear.
- Breaking downstream apps without first confirming real consumers.

## System Orientation For A New Intern

Before evaluating cleanup candidates, understand the main subsystems involved.

### 1. Profiles and registries

The `pkg/profiles` package stores and resolves runtime presets. The core concepts are:

- `ProfileRegistry`: a named collection of profiles with a default profile slug.
- `Profile`: a named preset with a stack, runtime payload, metadata, and extensions.
- `RuntimeSpec`: the actual runtime data stored on a profile.
- `StoreRegistry`: the main resolver implementation.
- `ChainedRegistry`: a precedence-aware wrapper that loads multiple registry sources.

Relevant files:

- [pkg/profiles/types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go)
- [pkg/profiles/registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go)
- [pkg/profiles/service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go)
- [pkg/profiles/source_chain.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go)

### 2. Sections bootstrapping

The `pkg/sections` package is where profile-registry settings are wired into command parsing and defaults/config/env precedence. This package matters because migration shims often live at bootstrapping boundaries.

Relevant files:

- [pkg/sections/sections.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go)
- [pkg/sections/profile_registry_source.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_registry_source.go)

### 3. JS module API

The JS bindings in `pkg/js/modules/geppetto` expose profiles, engines, turns, and schema-discovery features. This matters because low-value behavior often persists through JS-facing surfaces longer than through core Go internals.

Relevant files:

- [pkg/js/modules/geppetto/module.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go)
- [pkg/js/modules/geppetto/api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go)
- [pkg/js/modules/geppetto/api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go)
- [pkg/js/modules/geppetto/api_schemas.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_schemas.go)

### 4. Turn metadata and inference metadata

Runtime and inference metadata are written to turns and then sometimes mirrored or normalized into event payloads. This is a classic place for compatibility code to linger.

Relevant files:

- [pkg/inference/engine/run_with_result.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/run_with_result.go)
- [pkg/steps/ai/runtimeattrib/runtimeattrib.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go)
- [pkg/turns/keys_gen.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/turns/keys_gen.go)

### 5. Documentation layer

Geppetto carries a large internal documentation corpus in `pkg/doc` and `ttmp`. Cleanup work must include docs because stale docs are a compatibility layer of their own: they preserve old ideas in engineers' heads even after the code is gone.

## Findings By Category

## A. Clear backward-compatibility support

### A1. Runtime metadata normalization accepts multiple historical shapes

Observed in [runtimeattrib.go:10-70](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go#L10).

The function accepts:

- a plain string runtime key,
- a map with `runtime_key`,
- a map with `key`,
- a map with `slug`,
- both dotted and underscored profile/registry/version forms.

That is explicit legacy support. It is useful only if older producers still emit those shapes.

Pseudocode summary:

```go
switch runtimeMeta := KeyTurnMetaRuntime.Get(...) {
case string:
    extra["runtime_key"] = runtimeMeta
case map:
    extra["runtime_key"] = firstNonEmpty(runtime_key, key, slug)
    extra["profile.slug"] = firstNonEmpty(profile.slug, profile_slug)
    extra["profile.registry"] = firstNonEmpty(profile.registry, registry_slug)
    extra["profile.version"] = firstPositive(profile.version, profile_version)
}
```

Assessment:

- Category: backward compatibility
- Confidence: high
- Delete now?: no, not before downstream confirmation
- Best next step: find current producers of `KeyTurnMetaRuntime`

### A2. Inference metadata still mirrors legacy scalar keys

Observed in [run_with_result.go:185-210](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/run_with_result.go#L185).

`MirrorLegacyInferenceKeys` writes:

- provider,
- model,
- stop reason,
- usage

back into old scalar metadata keys "during migration". The name and comment make the intent explicit.

Assessment:

- Category: backward compatibility
- Confidence: high
- Delete now?: no, not before searching downstream readers of those scalar keys
- Best next step: inspect references to `KeyTurnMetaProvider`, `KeyTurnMetaModel`, `KeyTurnMetaStopReason`, and `KeyTurnMetaUsage`

### A3. Profile flag loading bridge is explicitly a migration shim

Observed in [profile_registry_source.go:14-18](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_registry_source.go#L14).

The comment says:

- "drop-in compatible",
- "for migration use".

That does not automatically mean it should be deleted right away. It does mean this is not the clean conceptual end-state API.

Assessment:

- Category: migration shim
- Confidence: high
- Delete now?: not without verifying how commands use it

### A4. CLI help still frames direct engine/provider flags as migration escape hatches

Observed in [chat.yaml:5-17](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/flags/chat.yaml#L5).

This is a softer form of legacy support: the code path may still be useful, but the help text says the system is still in a migration phase. If that migration is effectively complete, this wording keeps the architecture looking more transitional than it really is.

Assessment:

- Category: migration/documentation lag
- Confidence: medium-high
- Delete now?: wording only, after product-owner confirmation

## B. Docs drift that preserves removed features

### B1. Request overrides are documented as active even though JS APIs reject them

Observed in:

- [api_profiles.go:170-171](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go#L170)
- [api_engines.go:259-260](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go#L259)
- [01-profiles.md:137-153](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/01-profiles.md#L137)
- [13-js-api-reference.md:131-144](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md#L131)
- [14-js-api-user-guide.md:157-171](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/14-js-api-user-guide.md#L157)

This is a strong cleanup candidate even if no code changes land. The docs are simply wrong relative to the implementation.

Assessment:

- Category: stale documentation
- Confidence: very high
- Delete now?: yes, together with GP-44 or as its own doc cleanup

### B2. Profile docs still describe `PolicySpec` override controls as live model

The docs still refer to:

- `allow_overrides`
- `allowed_override_keys`
- `denied_override_keys`
- `requestOverrides`

But the current `pkg/profiles/types.go` no longer exposes `PolicySpec` at all in the inspected version.

Assessment:

- Category: stale documentation
- Confidence: very high
- Delete now?: yes

## C. Likely unused or low-value helpers

### C1. `pkg/profiles/adapters.go` is probably pure ceremony

Observed in [adapters.go:1-30](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/adapters.go#L1).

The file just wraps parse/string methods:

- `RegistrySlugFromString`
- `ProfileSlugFromString`
- `RuntimeKeyFromString`
- `RegistrySlugToString`
- `ProfileSlugToString`
- `RuntimeKeyToString`

Repo search found only the file itself and its tests, not meaningful production call sites.

Assessment:

- Category: likely unused helper
- Confidence: high
- Delete now?: probably yes after one wider repo search

### C2. `StoreRegistry.extensionCodecs` appears written but not read

Observed in [service.go:18-33](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L18).

`WithExtensionCodecRegistry(...)` stores a registry on `StoreRegistry`, but the field does not appear to be consumed elsewhere in current runtime code. That suggests a partially integrated feature or leftover hook.

Assessment:

- Category: likely unused integration hook
- Confidence: medium-high
- Delete now?: only after confirming there is no future write-path that relies on it

### C3. `NormalizeProfileExtensions` appears test-oriented from current in-repo usage

Observed in [extensions.go:242-270](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions.go#L242).

Search results showed tests around it, but I did not find production callers outside `pkg/profiles` tests. That makes it suspicious, though not conclusively dead if other repositories call it.

Assessment:

- Category: likely unused helper
- Confidence: medium
- Delete now?: no, first confirm downstream usage

### C4. Middleware-extension projection helpers appear test-only

Observed in [middleware_extensions.go:53-146](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/middleware_extensions.go#L53).

Search results showed tests, docs, and descriptions, but no production callers. This is exactly the kind of feature that can remain intellectually expensive while adding no active value.

Assessment:

- Category: likely unused subsystem
- Confidence: medium-high
- Delete now?: after a wider usage search

## D. Over-complex but potentially still useful

### D1. `ChainedRegistry` implementation carries apparently unused fields

Observed in [source_chain.go:39-45](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go#L39).

The struct stores:

- `aggregateStore`
- `registryOwners`

Search results did not show them being read after construction. The main active fields appear to be:

- `aggregate`
- `precedenceTopFirst`
- `defaultRegistrySlug`
- `sources`

This is not necessarily a big deal, but it is a signal that the implementation has accumulated bookkeeping beyond the active algorithm.

Assessment:

- Category: local over-complexity
- Confidence: medium-high
- Delete now?: yes for unused fields, after verifying tests

### D2. Stack trace metadata is always built, even though it looks debug-oriented

Observed in:

- [stack_merge.go:71-76](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_merge.go#L71)
- [service.go:166-172](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L166)
- [stack_trace.go:48-170](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_trace.go#L48)

The system always computes a path-level trace and embeds `profile.stack.trace` into metadata. That is valuable for debugging and docs, but it may be more detail than normal runtime callers need on every resolve.

Assessment:

- Category: over-complex / potentially over-eager debug behavior
- Confidence: medium
- Delete now?: no; better candidate for "make opt-in" than immediate deletion

### D3. JS profile-registry runtime switching keeps a mini ownership lifecycle

Observed in [module.go:67-107](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go#L67) and [api_profiles.go:230-268](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go#L230).

This machinery may be justified, but it is more elaborate than a simple binding:

- base registry,
- current registry,
- closers,
- ownership flag,
- current source list,
- restore-on-disconnect semantics.

The subtle issue I found is that `baseProfileRegistrySpec` is stored on the runtime struct but I did not find code that initializes it from options, so the "restore baseline source list" part looks weaker than the rest of the ownership model.

Assessment:

- Category: over-complex integration
- Confidence: medium
- Delete now?: no
- Best next step: inspect actual host usage and whether connected-source restoration needs refinement

## Architecture Diagram

This simplified diagram shows where most cleanup candidates sit:

```text
CLI / App Boot
   |
   v
sections bootstrap --------------------> migration shims / help-text lag
   |
   v
profiles.Registry / ChainedRegistry ---> unused fields / stack trace always-on / thin adapters
   |
   v
JS APIs --------------------------------> stale options / stale docs / registry-switch lifecycle complexity
   |
   v
Turns + Event Metadata -----------------> legacy scalar mirroring / legacy shape normalization
```

## Proposed Cleanup Strategy

Do not remove everything in one PR. Use phased hard cuts.

### Phase 1: Fix documentation drift

This is the safest cleanup and has the highest signal-to-risk ratio.

Remove or rewrite docs that still describe:

- request overrides,
- override policy fields,
- `runtimeKey` as a meaningful runtime-control concept.

### Phase 2: Remove obviously vestigial helpers

Candidates:

- `pkg/profiles/adapters.go`
- unused `ChainedRegistry` fields

These are small, easy-to-review changes if wider grep still shows no meaningful consumers.

### Phase 3: Decide on compatibility horizon

Before touching runtime metadata compatibility code, explicitly answer:

1. Which downstream systems still read old scalar inference keys?
2. Which downstream systems still emit old runtime metadata shapes?

If the answer is "none", then remove:

- `MirrorLegacyInferenceKeys`
- extra legacy branches in `AddRuntimeAttributionToExtra`

### Phase 4: Audit lightly integrated extensibility layers

Before removing extension normalization or middleware-extension projection:

1. grep the full mono-repo,
2. inspect downstream repos,
3. decide whether the feature is strategic or accidental.

If strategic:

- document the supported flow clearly.

If accidental:

- remove the machinery and simplify the profile model.

### Phase 5: Revisit always-on debug payload generation

If performance or payload size matter, consider:

```go
if debugTracingEnabled {
    metadata["profile.stack.trace"] = trace.BuildDebugPayload()
}
```

That should only happen after confirming whether callers rely on the trace always being present.

## Detailed Task Guidance

### For documentation cleanup

Files:

- [pkg/doc/topics/01-profiles.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/01-profiles.md)
- [pkg/doc/topics/13-js-api-reference.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md)
- [pkg/doc/topics/14-js-api-user-guide.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/14-js-api-user-guide.md)

Action:

- align all prose with live code,
- remove references to removed request-time mutation,
- explain the current hard-cut model directly.

### For legacy metadata cleanup

Files:

- [pkg/inference/engine/run_with_result.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/run_with_result.go)
- [pkg/steps/ai/runtimeattrib/runtimeattrib.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go)

Action:

- map current readers and writers first,
- reduce accepted shapes only when canonical producers/readers are confirmed.

### For extensions cleanup

Files:

- [pkg/profiles/extensions.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions.go)
- [pkg/profiles/middleware_extensions.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/middleware_extensions.go)
- [pkg/js/modules/geppetto/api_schemas.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_schemas.go)

Action:

- distinguish "used for schema discovery" from "used in runtime mutation",
- delete unused write/normalize paths only after that distinction is explicit.

## Risks And Alternatives

### Risk: removing compatibility code before the migration is truly over

This is the main risk for runtime metadata and inference metadata cleanup.

Mitigation:

- search the larger workspace,
- ask downstream owners if needed,
- stage deprecations or short-lived release notes if necessary.

### Risk: deleting strategically valuable but currently dormant extension hooks

Some extension or middleware-extension machinery may exist because the repository wants to grow into it, not because it already uses it heavily.

Mitigation:

- require an explicit owner decision before deletion,
- prefer "document or delete" over leaving ambiguous half-integration.

### Alternative: keep everything and just fix docs

This is safer short term, but it preserves code complexity and onboarding cost. It should only be chosen for subsystems with confirmed downstream consumers.

## Open Questions

1. Which downstream repositories still read legacy scalar inference keys?
2. Which downstream repositories still emit old runtime metadata shapes?
3. Is extension normalization part of a planned future write path, or is it just unused infrastructure?
4. Should `profile.stack.trace` be permanent API output, or is it really debug material?

## Recommended Split Into Follow-Up Tickets

If this ticket becomes too large, split it into:

1. doc sync cleanup,
2. legacy metadata cleanup,
3. extension and middleware-extension audit,
4. chained-registry simplification.

## References

- [pkg/steps/ai/runtimeattrib/runtimeattrib.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go)
- [pkg/inference/engine/run_with_result.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/run_with_result.go)
- [pkg/sections/profile_registry_source.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_registry_source.go)
- [pkg/sections/sections.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go)
- [pkg/profiles/adapters.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/adapters.go)
- [pkg/profiles/extensions.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions.go)
- [pkg/profiles/middleware_extensions.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/middleware_extensions.go)
- [pkg/profiles/source_chain.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go)
- [pkg/profiles/stack_trace.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_trace.go)
- [pkg/doc/topics/01-profiles.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/01-profiles.md)
