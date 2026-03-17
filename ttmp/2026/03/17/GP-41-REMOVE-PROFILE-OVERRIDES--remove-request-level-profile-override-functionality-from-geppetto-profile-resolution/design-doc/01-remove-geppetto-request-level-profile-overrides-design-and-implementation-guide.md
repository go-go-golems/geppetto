---
Title: 'Remove Geppetto request-level profile overrides: design and implementation guide'
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
    - Path: 2026-03-16--gec-rag/internal/webchat/resolver.go
      Note: GEC-RAG forwards request_overrides into Geppetto profile resolution
    - Path: geppetto/pkg/profiles/registry.go
      Note: Public ResolveInput API currently exposing RequestOverrides
    - Path: geppetto/pkg/profiles/service.go
      Note: Core request override resolution and final step settings application
    - Path: geppetto/pkg/profiles/stack_merge.go
      Note: Override policy merge behavior across profile stacks
    - Path: geppetto/pkg/profiles/types.go
      Note: RuntimeSpec and PolicySpec definitions that would be simplified
    - Path: pinocchio/cmd/web-chat/profile_policy.go
      Note: Pinocchio forwards request_overrides into Geppetto profile resolution
    - Path: temporal-relationships/internal/extractor/httpapi/run_chat_transport.go
      Note: Control case using profile resolution without request overrides
ExternalSources: []
Summary: Evidence-backed implementation guide for removing request-level profile overrides from Geppetto. Explains current architecture, downstream usage, proposed API simplification, migration steps, tests, docs cleanup, and practical examples for new contributors.
LastUpdated: 2026-03-17T14:20:00-04:00
WhatFor: Use this guide to understand the current override system, why it should be removed, and how to implement the removal safely.
WhenToUse: Use when onboarding to GP-41, implementing the removal, reviewing the change, or auditing downstream impact.
---


# Remove Geppetto request-level profile overrides: design and implementation guide

## Executive Summary

This ticket proposes removing request-level profile override functionality from Geppetto profile resolution. In the current design, profile resolution accepts a `RequestOverrides` map, checks override policy, normalizes key names, merges override values into the resolved `RuntimeSpec`, and only then produces `EffectiveStepSettings`. That behavior is implemented in [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L128) and [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go#L34).

The problem is not that the code is wrong. The problem is that the code supports a degree of per-request runtime customization that current downstream products do not actually depend on as an everyday operating model. Pinocchio, GEC-RAG, and Temporal Relationships primarily select a profile and then use the resolved runtime wholesale. Temporal Relationships does not even expose `request_overrides` on its run-chat HTTP surface. The extra override layer therefore adds policy complexity, validation code, JS API surface, HTTP payload fields, documentation burden, and tests for behavior that is largely unused in practice.

The recommended design is simple:

1. Make profile resolution purely profile-based.
2. Remove request-level override plumbing from Geppetto core and JS bindings.
3. Treat profiles as opinionated runtime presets, not partially mutable templates.
4. Let downstream applications keep any truly application-specific request customization outside the profile resolver.

This document explains the current system, the evidence for removal, the implementation plan, the migration steps, the tests to update, and the practical consequences for Pinocchio and GEC-RAG.

## Problem Statement

### What is a profile in this system?

A Geppetto profile is a named runtime preset. The stored runtime payload lives in `RuntimeSpec`, which currently includes:

- `step_settings_patch`
- `system_prompt`
- `middlewares`
- `tools`

That structure is defined in [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go#L13).

Profiles can also carry a `PolicySpec`, which currently includes request override controls:

- `allow_overrides`
- `allowed_override_keys`
- `denied_override_keys`
- `read_only`

Those policy fields are also defined in [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go#L21).

### What is request-level override functionality?

The current profile resolver accepts this input:

```go
type ResolveInput struct {
    RegistrySlug       RegistrySlug
    ProfileSlug        ProfileSlug
    RuntimeKeyFallback RuntimeKey

    BaseStepSettings *settings.StepSettings
    RequestOverrides map[string]any
}
```

See [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go#L34).

When a caller passes `RequestOverrides`, the resolver:

1. normalizes override keys,
2. checks allow/deny policy,
3. parses supported override payloads,
4. mutates the resolved `RuntimeSpec`,
5. applies the final `StepSettingsPatch`,
6. computes a runtime fingerprint that now depends on the override payload.

The key call site is in [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L155), where `resolveRuntimeSpec(stackMerge.Runtime, stackMerge.Policy, in.RequestOverrides)` is used before `ApplyRuntimeStepSettingsPatch(...)`.

### Why is this a problem?

Because the codebase is paying the complexity cost of a flexible override system, but the concrete products are mostly using profiles as fixed presets.

That mismatch creates several forms of accidental complexity:

- extra API surface in Go and JS,
- extra policy merge logic,
- extra validation branches,
- extra HTTP request fields,
- extra documentation that interns must learn,
- extra tests for behavior that is not central to product workflows,
- extra ambiguity about whether the source of truth is the stored profile or the request payload.

For a new engineer, this is especially confusing because the apparent model becomes:

```text
stored profile
  -> stacked profile merge
  -> request override policy merge
  -> request override parsing
  -> runtime mutation
  -> final step settings patch application
  -> engine construction
```

But the actual product behavior is usually just:

```text
selected profile
  -> resolve effective runtime
  -> apply runtime
  -> run
```

## Scope

### In scope

- Request override plumbing in Geppetto profile resolution.
- Override-related policy fields and merge behavior.
- JS bindings that expose request overrides.
- Downstream request contracts that only forward request overrides into Geppetto.
- Tests and docs that encode override behavior.

### Out of scope

- Removing profile-based `runtime.step_settings_patch`.
- Reworking profile storage format.
- Removing profile stacks.
- Removing downstream profile switching.

## Current-State Architecture

### Core profile resolution path

The main resolver logic is in [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L128).

The sequence is:

1. Load registry and selected profile.
2. Expand the stack.
3. Merge stack layers into one runtime and one policy.
4. Apply request overrides against the merged runtime and merged policy.
5. Apply the final `runtime.step_settings_patch` to `BaseStepSettings`.
6. Produce `ResolvedProfile`.

The core logic looks like this conceptually:

```go
func ResolveEffectiveProfile(in ResolveInput) ResolvedProfile {
    layers := ExpandProfileStack(...)
    merged := MergeProfileStackLayersWithTrace(layers)
    effectiveRuntime := resolveRuntimeSpec(
        merged.Runtime,
        merged.Policy,
        in.RequestOverrides,
    )
    effectiveStepSettings := ApplyRuntimeStepSettingsPatch(
        in.BaseStepSettings,
        effectiveRuntime.StepSettingsPatch,
    )
    return ResolvedProfile{
        EffectiveRuntime: effectiveRuntime,
        EffectiveStepSettings: effectiveStepSettings,
    }
}
```

### Override keys supported today

The resolver currently supports four request override keys:

- `system_prompt`
- `middlewares`
- `tools`
- `step_settings_patch`

Those constants are defined in [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L15), and they are parsed in [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L344).

### Policy complexity that exists only because of overrides

The current `PolicySpec` includes override-specific fields in [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go#L21).

Those fields then drive:

- validation in [validation.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/validation.go#L84),
- restrictive policy merge logic in [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_merge.go#L141),
- request override enforcement in `service.go`,
- several override-specific tests in [service_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service_test.go#L524).

This is important. Removing request overrides does not just delete one field. It removes an entire sub-system.

### JS binding surfaces

The Geppetto JS bindings currently expose request overrides in at least two places:

- `profiles.resolve({... requestOverrides ...})` in [api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go#L223)
- `engines.fromProfile(..., { requestOverrides })` in [api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go#L248)

This means override removal must be treated as an API simplification, not only an internal refactor.

## Downstream Usage Analysis

This section answers the practical question: who actually uses this functionality?

### Pinocchio

Pinocchio’s web chat request body still carries `request_overrides` in [api.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/http/api.go#L21).

Its resolver forwards those values directly into Geppetto in [profile_policy.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/profile_policy.go#L244) and [profile_policy.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/profile_policy.go#L298).

The frontend widget can emit `request_overrides` if `buildOverrides()` returns anything in [ChatWidget.tsx](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx#L170).

But the actual runtime composer behavior is still coarse-grained. It resolves a profile runtime and then applies that result wholesale in [runtime_composer.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/runtime_composer.go#L73).

Observed conclusion:

- Pinocchio exposes override plumbing.
- Pinocchio composes runtimes from resolved profiles, not from per-request override-heavy workflows.
- Override support is part of the request contract, but not central to the runtime composition model.

### GEC-RAG

GEC-RAG exposes `request_overrides` in its frontend API type in [chatApi.ts](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/web/src/api/chatApi.ts#L3).

Its resolver forwards `body.RequestOverrides` into Geppetto in [resolver.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/resolver.go#L169) and [resolver.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/resolver.go#L268).

But its runtime path also behaves as profile selection plus resolved runtime application in [runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/runtime.go#L159).

Observed conclusion:

- GEC-RAG carries override support through its transport and resolver surface.
- Its real runtime model is still profile selection, application profile merging, and then resolved runtime application.

### Temporal Relationships

Temporal Relationships does not expose `request_overrides` on its run-chat HTTP request type. The request shape in [run_chat_handlers.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_chat_handlers.go#L20) includes `profile` and tool setup, but no override map.

Its resolver constructs `ResolveInput{BaseStepSettings: r.baseStepSettings}` and optionally a `ProfileSlug` before calling Geppetto in [run_chat_transport.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_chat_transport.go#L540).

Observed conclusion:

- Temporal Relationships uses profile selection only.
- It is direct evidence that the request override machinery is not required for all downstream integrations.

### Concrete behavioral summary

The important distinction is:

- profile switching is actively used,
- `runtime.step_settings_patch` inside profiles is actively used,
- request-level runtime override functionality is mostly carried as a capability, not used as a core product pattern.

## Why removal is the right direction

### Design principle

Profiles should be opinionated runtime presets. A request should typically choose one profile, not partially rewrite it.

This leads to a simpler mental model:

```text
request
  -> choose profile
  -> resolve profile
  -> apply resolved runtime
  -> run inference loop
```

Instead of:

```text
request
  -> choose profile
  -> merge stack
  -> merge policy
  -> validate override allow/deny sets
  -> parse request override map
  -> mutate runtime
  -> apply step settings patch
  -> run inference loop
```

### Benefits

- Fewer concepts for new contributors.
- Narrower API surface in Go and JS.
- Less policy machinery.
- Cleaner downstream HTTP contracts.
- Smaller test matrix.
- Clearer provenance and runtime fingerprint behavior.

### Tradeoff

The tradeoff is that callers lose a generic escape hatch for request-specific prompt/tool/middleware/step-setting mutation through the profile resolver.

That is acceptable here because:

- the concrete product paths already operate at profile granularity,
- any truly application-specific per-request customization can live in app code instead of the core profile subsystem,
- the resulting system becomes easier to maintain and explain.

## Proposed Design

### High-level decision

Remove request-level profile overrides from Geppetto entirely.

That means:

1. `ResolveInput` no longer accepts `RequestOverrides`.
2. `PolicySpec` no longer carries override allow/deny fields.
3. `resolveRuntimeSpec` becomes a profile-runtime normalization helper, not an override application helper.
4. Stack merge logic no longer merges override policy fields.
5. JS bindings stop accepting `requestOverrides`.
6. Downstream HTTP request types stop forwarding `request_overrides` into profile resolution.

### Proposed API shape

Current:

```go
type ResolveInput struct {
    RegistrySlug       RegistrySlug
    ProfileSlug        ProfileSlug
    RuntimeKeyFallback RuntimeKey
    BaseStepSettings   *settings.StepSettings
    RequestOverrides   map[string]any
}
```

Proposed:

```go
type ResolveInput struct {
    RegistrySlug       RegistrySlug
    ProfileSlug        ProfileSlug
    RuntimeKeyFallback RuntimeKey
    BaseStepSettings   *settings.StepSettings
}
```

Current `PolicySpec`:

```go
type PolicySpec struct {
    AllowOverrides      bool
    AllowedOverrideKeys []string
    DeniedOverrideKeys  []string
    ReadOnly            bool
}
```

Proposed `PolicySpec`:

```go
type PolicySpec struct {
    ReadOnly bool
}
```

If keeping backward-compatible YAML decode is temporarily required, an intermediate form could ignore old fields during decode but not use them anywhere. That is a migration choice. It is not the preferred end state.

## Design Rationale

### Rationale 1: preserve the useful part

The useful part of profiles is not the request override mechanism. The useful part is:

- named runtime presets,
- stack composition,
- stored prompt/tool/middleware/model defaults,
- stable profile selection across apps.

Those remain intact after removal.

### Rationale 2: delete complexity at its source

If override removal only happened in Pinocchio and GEC-RAG, the Geppetto core would still retain:

- override key constants,
- policy fields,
- policy merge,
- policy validation,
- request override parsing,
- override-specific tests,
- JS binding support.

That would not actually simplify the architecture. The complexity must be removed at the Geppetto core layer.

### Rationale 3: put application-specific mutability in application code

If one application someday needs a per-request prompt suffix, a per-request tool subset, or a temporary debug setting, that application can do it directly in its own request handling or runtime composition layer.

That is a better separation of concerns:

- Geppetto profiles provide stored presets.
- Applications own ad hoc request mutation behavior.

## Alternatives Considered

### Alternative A: keep request overrides, but reduce the number of supported keys

Example:

- allow only `system_prompt`
- remove `middlewares`, `tools`, and `step_settings_patch`

Why rejected:

- It still keeps policy machinery alive.
- It still leaves the confusing mental model in place.
- It still requires downstream request contracts to explain override behavior.

### Alternative B: keep overrides only in Pinocchio/GEC-RAG, remove them from Geppetto core

Why rejected:

- Pinocchio and GEC-RAG currently rely on Geppetto to interpret the override map.
- If the applications really need custom request mutation, they should implement it explicitly rather than pretending it is a profile concept.

### Alternative C: deprecate first, remove later

This is a valid rollout strategy if compatibility matters. The end state is still full removal.

Possible deprecation path:

1. Stop documenting request overrides.
2. Add warnings if request overrides are passed.
3. Remove usage in downstream apps.
4. Delete the core support.

This may be useful if external callers rely on Pinocchio web chat APIs, but it is a rollout decision, not the architectural recommendation.

## Practical Real-World Examples

### Example 1: Pinocchio web chat

Desired behavior:

- user selects `analyst`
- server resolves profile `analyst`
- runtime composer applies profile prompt, tools, middleware, and `step_settings_patch`
- chat runs

What is not needed:

- request body mutating `system_prompt`
- request body injecting ad hoc middleware definitions
- request body patching model/provider values through the resolver

### Example 2: GEC-RAG application profile plus inference profile

Desired behavior:

- application profile controls domain prompt and tool allowlist
- inference profile controls model/provider patch
- server merges those stored profile concepts
- runtime runs

What is not needed:

- a third request-level override layer mutating the merged result

### Example 3: Temporal Relationships run-chat

Desired behavior:

- session stores a selected profile
- request names tool setup and session context
- transport resolves the profile and uses it

This flow already works without exposing request overrides on the wire.

## System Diagram

### Current system

```text
HTTP / JS request
        |
        v
ResolveInput{
  ProfileSlug,
  BaseStepSettings,
  RequestOverrides,
}
        |
        v
ExpandProfileStack
        |
        v
MergeProfileStackLayers
  - Runtime merge
  - Policy merge
        |
        v
resolveRuntimeSpec(...)
  - normalize override keys
  - enforce override policy
  - parse override payloads
  - mutate runtime
        |
        v
ApplyRuntimeStepSettingsPatch(...)
        |
        v
ResolvedProfile
        |
        v
Runtime composer / engine builder
```

### Proposed system

```text
HTTP / JS request
        |
        v
ResolveInput{
  ProfileSlug,
  BaseStepSettings,
}
        |
        v
ExpandProfileStack
        |
        v
MergeProfileStackLayers
  - Runtime merge
  - ReadOnly merge
        |
        v
ApplyRuntimeStepSettingsPatch(...)
        |
        v
ResolvedProfile
        |
        v
Runtime composer / engine builder
```

## File-by-File Implementation Guide

This section is written for a new intern. It explains not just what to change, but why each file matters.

### 1. Geppetto core types

Start with [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go#L34) and [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go#L13).

What to change:

- remove `RequestOverrides` from `ResolveInput`
- remove override-related policy fields from `PolicySpec`
- update clone logic accordingly

Why:

- these types define the public shape of the subsystem,
- deleting the fields here makes the simplification visible and enforceable.

### 2. Geppetto resolver implementation

Study [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go#L128).

What to change:

- remove the override constants,
- delete `resolveRuntimeSpec(base, policy, requestOverrides)` or replace it with a simpler no-override helper,
- remove request override parsing helpers,
- remove override policy enforcement from the resolution path,
- compute the resolved runtime directly from the merged stack runtime.

Conceptual pseudocode:

```go
func (r *StoreRegistry) ResolveEffectiveProfile(ctx context.Context, in ResolveInput) (*ResolvedProfile, error) {
    registry := loadRegistry(...)
    profileSlug := resolveProfileSlug(...)
    stackLayers := ExpandProfileStack(...)
    stackMerge, stackTrace := MergeProfileStackLayersWithTrace(stackLayers)

    effectiveRuntime := cloneRuntimeSpec(stackMerge.Runtime)
    effectiveStepSettings := ApplyRuntimeStepSettingsPatch(
        in.BaseStepSettings,
        effectiveRuntime.StepSettingsPatch,
    )

    return &ResolvedProfile{
        EffectiveRuntime: effectiveRuntime,
        EffectiveStepSettings: effectiveStepSettings,
        RuntimeFingerprint: runtimeFingerprint(...),
    }, nil
}
```

### 3. Stack policy merge

See [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_merge.go#L141).

What to change:

- remove `AllowOverrides`, `AllowedOverrideKeys`, and `DeniedOverrideKeys` merge behavior,
- simplify the policy merge function so it only preserves what still matters, likely `ReadOnly`.

Why:

- otherwise you leave dead policy logic in place.

### 4. Policy validation

See [validation.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/validation.go#L84).

What to change:

- remove override allow/deny validation,
- keep only validation for fields that still exist.

Why:

- validation should match the actual model, not a legacy shape.

### 5. JS binding cleanup

See [api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go#L223) and [api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go#L240).

What to change:

- remove `requestOverrides` parsing,
- update any TypeScript declaration templates and tests,
- update examples that demonstrate request overrides.

Why:

- a simplified core API should not be contradicted by the JS layer.

### 6. Pinocchio cleanup

Start with:

- [api.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/http/api.go#L21)
- [profile_policy.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/profile_policy.go#L230)
- [ChatWidget.tsx](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx#L162)

What to change:

- remove `RequestOverrides` from the chat request contract,
- remove resolver forwarding of overrides,
- decide whether `buildOverrides` remains for application-owned behavior or is deleted,
- update docs and tests accordingly.

Important design note:

If Pinocchio wants a non-profile request customization later, it should apply it after profile resolution in Pinocchio-owned code, not via Geppetto profile resolver overrides.

### 7. GEC-RAG cleanup

Start with:

- [chatApi.ts](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/web/src/api/chatApi.ts#L3)
- [resolver.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/resolver.go#L160)

What to change:

- remove `request_overrides` from frontend request typing,
- stop forwarding overrides into Geppetto resolution,
- keep the application-profile plus inference-profile merge, which is still a valid stored-profile composition pattern.

### 8. Tests

Start with [service_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service_test.go#L524) and search for `RequestOverrides`, `allow_overrides`, and `allowed_override_keys`.

What to do:

- delete tests that validate override behavior,
- rewrite any tests whose purpose should now be expressed in pure profile terms,
- add tests that protect the new simpler contract.

Recommended post-removal tests:

1. resolving a plain profile still yields the expected runtime and step settings
2. stacked profile merge still works
3. runtime fingerprint changes on meaningful profile changes
4. read-only policy still behaves correctly if retained
5. JS `profiles.resolve` and `engines.fromProfile` still work without override options

### 9. Documentation

Search results already show several docs and examples that mention:

- `request_overrides`
- request override policy
- override examples in JS

These docs must be updated so new contributors do not learn a removed concept.

Minimum documentation cleanup targets:

- Geppetto profile docs and JS API docs
- Pinocchio web chat setup docs
- examples that currently demonstrate override behavior

## Recommended Implementation Phases

### Phase 1: core Geppetto API cleanup

Files:

- [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go)
- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go)
- [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go)
- [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_merge.go)
- [validation.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/validation.go)

Goal:

- remove the feature at the core layer,
- keep profile resolution and stack merging intact.

### Phase 2: JS and example cleanup

Files:

- [api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go)
- [api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go)
- Geppetto examples and docs found by the inventory script

Goal:

- remove stale exposure of the deleted concept.

### Phase 3: downstream cleanup

Files:

- [api.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/http/api.go)
- [profile_policy.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/profile_policy.go)
- [ChatWidget.tsx](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx)
- [chatApi.ts](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/web/src/api/chatApi.ts)
- [resolver.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/resolver.go)

Goal:

- align HTTP contracts and frontend types with the simpler profile-only model.

### Phase 4: documentation and final audit

Goal:

- remove obsolete docs,
- update examples,
- ensure no remaining surface teaches request overrides as a supported concept.

## Testing and Validation Strategy

### Core tests

Run targeted tests around profile resolution:

```bash
go test ./geppetto/pkg/profiles/... -count=1
go test ./geppetto/pkg/js/modules/geppetto/... -count=1
```

### Downstream tests

Run targeted tests for Pinocchio and GEC-RAG request resolution:

```bash
go test ./pinocchio/cmd/web-chat/... -count=1
go test ./2026-03-16--gec-rag/internal/webchat/... -count=1
```

If frontend request types or components are updated, run the relevant frontend checks used by those repos.

### Manual verification checklist

1. Pinocchio chat still works with profile selection only.
2. GEC-RAG chat still resolves application profile plus inference profile correctly.
3. Temporal Relationships behavior is unchanged.
4. No docs or help text still describe request overrides as a supported contract.

## Risks and Mitigations

### Risk 1: external callers depend on `request_overrides`

Observed evidence shows the code exposes this field, especially in Pinocchio and GEC-RAG. Even if internal product flows do not rely on it heavily, external automation might.

Mitigation:

- decide whether to hard-remove immediately or deprecate for one release,
- mention the contract change explicitly in docs and changelogs.

### Risk 2: stale tests and examples keep reintroducing the concept

Mitigation:

- update examples and JS types in the same change set,
- use the inventory script in this ticket to search again before merge.

### Risk 3: policy removal breaks old YAML files

If old registries still contain override policy fields, decode behavior may need a short compatibility period.

Mitigation:

- either tolerate unknown legacy fields temporarily,
- or provide a migration note and fixture updates in the same patch.

## Open Questions

1. Should `PolicySpec` immediately shrink to only `ReadOnly`, or should legacy override fields remain ignored for one release?
2. Should Pinocchio keep a separate app-owned request customization hook, or should `buildOverrides` disappear entirely?
3. Are there external API consumers of Pinocchio web chat that require a brief deprecation period?

## Quick-start instructions for a new intern

If you are new to this codebase, follow this order:

1. Read [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go) and [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go) to understand the public model.
2. Read [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go) to see the resolver pipeline.
3. Read [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_merge.go) and [validation.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/validation.go) to understand the complexity added by overrides.
4. Read the downstream resolver files in Pinocchio and GEC-RAG to see how the feature is forwarded rather than deeply used.
5. Run the ticket-local inventory script from `scripts/`.
6. Make the core Geppetto type and resolver changes first.
7. Update JS bindings and downstream request contracts second.
8. Finish with docs and tests.

## References

- [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go)
- [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go)
- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go)
- [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_merge.go)
- [validation.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/validation.go)
- [service_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service_test.go)
- [api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go)
- [api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go)
- [api.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/http/api.go)
- [profile_policy.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/profile_policy.go)
- [ChatWidget.tsx](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx)
- [chatApi.ts](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/web/src/api/chatApi.ts)
- [resolver.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/resolver.go)
- [run_chat_handlers.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_chat_handlers.go)
- [run_chat_transport.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_chat_transport.go)
