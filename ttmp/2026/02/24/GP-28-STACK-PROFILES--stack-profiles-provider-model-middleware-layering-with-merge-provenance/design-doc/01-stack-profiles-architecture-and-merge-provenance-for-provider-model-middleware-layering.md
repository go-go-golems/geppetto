---
Title: Stack profiles architecture and merge provenance (v3 hard-cutover geppetto-first)
Ticket: GP-28-STACK-PROFILES
Status: active
Topics:
    - profile-registry
    - stack-profiles
    - merge-provenance
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/registry.go
      Note: Resolver contract and ResolveInput/ResolvedProfile baseline
    - Path: geppetto/pkg/profiles/service.go
      Note: Current single-profile resolve flow and runtime fingerprint path
    - Path: geppetto/pkg/profiles/types.go
      Note: Profile domain model to extend with stack references
    - Path: geppetto/pkg/profiles/runtime_settings_patch_resolver.go
      Note: Existing deep-merge helper reused for stacked step_settings_patch
    - Path: geppetto/pkg/profiles/validation.go
      Note: Validation boundaries for stack refs/cycle/depth constraints
    - Path: geppetto/pkg/profiles/overlay.go
      Note: Overlay abstraction to remove from implementation scope
    - Path: geppetto/pkg/inference/middlewarecfg/resolver.go
      Note: Path-level deterministic trace pattern to reuse for profile field provenance
    - Path: geppetto/pkg/inference/middlewarecfg/debug_payload.go
      Note: Stable debug payload serialization contract for trace output
    - Path: geppetto/pkg/inference/middlewarecfg/source.go
      Note: Canonical source-layer precedence model
    - Path: geppetto/pkg/js/modules/geppetto/api_engines.go
      Note: Current model-centric fromProfile path to replace with registry-backed semantics
    - Path: pinocchio/cmd/web-chat/profile_policy.go
      Note: Downstream web-chat profile API + resolver adaptation target
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Downstream runtime composer adaptation target
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go
      Note: Downstream strict resolver adaptation target
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go
      Note: Downstream runtime composer adaptation target
ExternalSources: []
Summary: v3 stack-profile implementation spec: geppetto-first hard cutover, multi-registry profile stacks, middlewarecfg-style per-field provenance trace, and policy-gated request-time overrides, with overlay removed.
LastUpdated: 2026-02-26T00:20:00-05:00
WhatFor: Execute stack profile support with deterministic multi-registry layering, auditable field trace, and safe request-time override controls.
WhenToUse: Use as the implementation plan for GP-28 and sequencing reference for downstream pinocchio/go-go-os adoption.
---

# Stack profiles architecture and merge provenance (v3 hard-cutover geppetto-first)

## Executive summary

This v3 updates GP-28 with the final requested scope:

1. Hard cutover, no compatibility layer.
2. Geppetto core first, downstream apps later.
3. Multiple profile registries are first-class in stack references.
4. Provenance uses middlewarecfg-style per-field trace (not minimal winners-only).
5. Request-time overrides remain supported and are enforced by `PolicySpec`.
6. Overlay abstraction is removed from the stack-profile implementation path.

## Final requirements

1. Support stack composition across multiple registries.
2. Emit deterministic per-path trace history for merged fields.
3. Apply request-time overrides only when policy allows.
4. Keep runtime fingerprint lineage-aware for cache correctness.
5. Keep downstream pinocchio/go-go-os changes sequenced after Geppetto core stabilization.

## Current-state constraints

1. Resolver is single-profile today (`geppetto/pkg/profiles/registry.go`, `geppetto/pkg/profiles/service.go`).
2. Overlay exists but is not required for stack implementation and is not part of desired target architecture (`geppetto/pkg/profiles/overlay.go`).
3. Downstream apps already resolve runtime in app-owned resolver/composer boundaries and can adopt the new core behavior after Geppetto is complete:
   - `pinocchio/cmd/web-chat/profile_policy.go`,
   - `pinocchio/cmd/web-chat/runtime_composer.go`,
   - `go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go`,
   - `go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go`.

## Scope (v3)

## In scope

1. Add persisted stack refs to `Profile` with optional cross-registry reference.
2. Resolve stack across registries with cycle/depth validation.
3. Merge runtime/policy/extensions deterministically.
4. Apply request-time overrides per `PolicySpec`.
5. Emit middlewarecfg-style per-field trace provenance.
6. Include layer lineage in runtime fingerprint.
7. Hard-cut JS semantics to registry-backed profile resolution.
8. Remove overlay implementation from this path.

## Out of scope

1. Backward-compatible dual semantics for `engines.fromProfile`.
2. Overlay-backed composition or store-chain fallback.
3. Immediate pinocchio/go-go-os implementation in this ticket.

## Data model

Add stack refs to profile domain:

```go
type ProfileRef struct {
    RegistrySlug RegistrySlug `json:"registry_slug,omitempty" yaml:"registry_slug,omitempty"`
    ProfileSlug  ProfileSlug  `json:"profile_slug" yaml:"profile_slug"`
}

type Profile struct {
    // existing fields
    Stack []ProfileRef `json:"stack,omitempty" yaml:"stack,omitempty"`
}
```

Semantics:

1. Empty `RegistrySlug` means current registry.
2. Non-empty `RegistrySlug` allows cross-registry layering.
3. `Stack` order is low -> high precedence.
4. Leaf profile is applied last.

## Resolver and merge behavior

## Stack expansion

1. Expand declared refs depth-first while preserving declared order.
2. Resolve each ref against `(registry_slug || current_registry, profile_slug)`.
3. Detect cycles across `(registry, profile)` identity chain.
4. Enforce max depth guard (recommended 32).
5. Deduplicate by first occurrence in expanded order.

## Merge order

1. Expanded stack layers from base to leaf.
2. Request-time overrides last.

## Field rules

1. `runtime.step_settings_patch`: deep merge via `MergeRuntimeStepSettingsPatches`.
2. `runtime.system_prompt`: last non-empty string wins.
3. `runtime.tools`: replace-on-write when provided.
4. `runtime.middlewares`: merge by middleware instance key.
5. `extensions`: deep map merge; scalar/list replace.
6. `policy`: restrictive merge.

Middleware key rule:

1. `name#id` preferred.
2. `name[index]` fallback only when id missing.

## Request-time overrides (explicit)

Request overrides are allowed, but only per effective `PolicySpec`:

1. Keep existing allow/deny gate semantics.
2. Keep denied/allowed key checks.
3. Keep deterministic canonicalization of override keys.
4. Return policy violations as explicit resolver errors.

This retains flexibility while preserving safety.

## Provenance (middlewarecfg trace pattern)

Adopt the same trace model style used by `middlewarecfg.Resolver`:

```go
type ProfileParseStep struct {
    RegistrySlug string         `json:"registry_slug"`
    ProfileSlug  string         `json:"profile_slug"`
    Version      uint64         `json:"version"`
    Source       string         `json:"source,omitempty"`
    Path         string         `json:"path"`
    Operation    string         `json:"operation"`
    Raw          any            `json:"raw,omitempty"`
    Value        any            `json:"value,omitempty"`
    Metadata     map[string]any `json:"metadata,omitempty"`
}

type ProfilePathTrace struct {
    Path  string             `json:"path"`
    Value any                `json:"value,omitempty"`
    Steps []ProfileParseStep `json:"steps"`
}

type ProfileResolutionProvenance struct {
    Layers       []ResolvedProfileLayer      `json:"layers"`
    OrderedPaths []string                    `json:"ordered_paths,omitempty"`
    Trace        map[string]ProfilePathTrace `json:"trace,omitempty"`
}
```

Requirements:

1. Deterministic ordering for paths and steps.
2. Stable serialization for debug tooling and tests.
3. Layer identity always includes registry + profile + version.

## Fingerprint and caching

Runtime fingerprint must include stack lineage and effective runtime output:

1. Ordered layer identity (`registry/profile/version`).
2. Effective runtime payload after stack merge + request overrides.
3. Effective step settings.

Goal: fingerprint changes whenever any relevant upstream layer changes so cache keys for prebuilt runtimes remain correct.

## Hard-cutover decisions

1. No compatibility path for legacy model-centric `engines.fromProfile` semantics.
2. No overlay abstraction in stack implementation.
3. No store-overlay fallback composition path.

## Implementation plan (sequenced)

### Phase 1 — Geppetto core stack resolver and provenance

1. Extend `pkg/profiles/types.go` (`ProfileRef`, `Profile.Stack`).
2. Add stack expansion helper (`stack_resolver.go`) with multi-registry resolution + cycle/depth checks.
3. Update `pkg/profiles/service.go`:
   - stack-aware merge,
   - policy-gated request overrides,
   - full field trace provenance,
   - lineage-aware fingerprint.
4. Extend validation and tests (`pkg/profiles/validation.go`, `pkg/profiles/*_test.go`).
5. Remove overlay code from near-term implementation path:
   - delete `pkg/profiles/overlay.go`,
   - delete overlay tests,
   - clean references.

Exit criteria:

1. `go test ./pkg/profiles` green with stack+trace+policy coverage.
2. Non-stack behavior remains stable.

### Phase 2 — Geppetto JS API hard cutover

1. Update `pkg/js/modules/geppetto/api_engines.go` to registry-backed semantics.
2. Expose stack/provenance-rich resolve outputs.
3. Update `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`.
4. Add JS tests for multi-registry stack resolution and provenance/fingerprint outputs.

Exit criteria:

1. No legacy model/env fallback in `engines.fromProfile`.
2. JS tests green for stack semantics.

### Phase 3 — Downstream adoption tickets

Create follow-on implementation tickets for:

1. Pinocchio CLI/web-chat adaptation:
   - `pinocchio/cmd/web-chat/profile_policy.go`,
   - `pinocchio/cmd/web-chat/runtime_composer.go`.
2. go-go-os adaptation:
   - `go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go`,
   - `go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go`,
   - integration tests.

Exit criteria:

1. Runtime key/fingerprint semantics remain deterministic and version-sensitive under stacked profiles.
2. Policy-gated request overrides remain enforced.

## Risks and mitigations

1. Risk: complex multi-registry cycle handling.
   - Mitigation: explicit `(registry,profile)` graph cycle detection with test fixtures.
2. Risk: provenance payload size.
   - Mitigation: deterministic trace with optional debug trimming later, but keep full trace in core.
3. Risk: middleware merge ambiguity without IDs.
   - Mitigation: strict validation/recommendation around IDs for stacked middleware definitions.
4. Risk: downstream drift while core is in progress.
   - Mitigation: geppetto-first sequencing and downstream follow-up tickets.

## Open questions

1. Should middleware `id` become mandatory in stacked profiles when multiple instances of same name exist?
2. Should provenance trace be always returned or gated behind a debug flag at API layer while always available internally?

## Final recommendation

Yes, this returns close to the original stronger design, with one key simplification retained:

1. keep multi-registry stacks,
2. keep middlewarecfg-style field trace provenance,
3. keep policy-gated request-time overrides,
4. remove overlay abstraction and avoid using it as stack mechanism,
5. execute Geppetto core first, then downstream consumers.
