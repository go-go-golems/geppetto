---
Title: Unified final JS API design for profile registry and schema-first middleware (inference-first)
Ticket: GP-21-PROFILE-MW-REGISTRY-JS
Status: active
Topics:
    - profile-registry
    - js-bindings
    - go-api
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../add-menus/go-go-os/ttmp/2026/02/24/OS-09-JS-ENGINE-API-DESIGN--js-engine-factory-profile-registry-and-middleware-schema-api-design/design-doc/01-comprehensive-js-api-design-for-engine-factories-profile-registry-and-schema-first-middleware.md
      Note: |-
        External full JS API proposal to merge with GP-21 findings
        External full JS API proposal merged into final recommendation
    - Path: pkg/inference/middlewarecfg/chain.go
      Note: Deterministic middleware build ordering and enable semantics
    - Path: pkg/inference/middlewarecfg/resolver.go
      Note: |-
        Canonical schema-first config precedence and trace model
        Schema-first precedence and trace model reused by final API
    - Path: pkg/inference/session/session.go
      Note: Single-active-inference and cancellation invariants
    - Path: pkg/inference/toolloop/enginebuilder/builder.go
      Note: Inference runtime composition contracts
    - Path: pkg/js/modules/geppetto/api_builder_options.go
      Note: Builder option plumbing for tools/hooks/sinks/persister
    - Path: pkg/js/modules/geppetto/api_engines.go
      Note: Existing fromProfile semantics and step-settings engine creation targeted for hard-cutover replacement
    - Path: pkg/js/modules/geppetto/api_middlewares.go
      Note: JS middleware context contract (sessionId/inferenceId/tags/deadline)
    - Path: pkg/js/modules/geppetto/api_sessions.go
      Note: |-
        Session/run/start/runAsync behavior and run-handle lifecycle
        Inference lifecycle contract preserved by final factory design
    - Path: pkg/js/modules/geppetto/module.go
      Note: |-
        Current export surface and module options
        Current JS export and options boundary for new namespace insertion points
    - Path: pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl
      Note: Type-level public JS API contract
    - Path: pkg/profiles/extensions.go
      Note: Extension schema codec interfaces
    - Path: pkg/profiles/middleware_extensions.go
      Note: Typed-key middleware config mapping
    - Path: pkg/profiles/registry.go
      Note: Registry read/write/resolve interfaces
    - Path: pkg/profiles/service.go
      Note: |-
        Effective profile resolution, override policy, runtime fingerprint
        ResolveEffectiveProfile policy and runtime fingerprint source of truth
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/various/inspect_from_profile_semantics.out
      Note: |-
        Runtime evidence for existing fromProfile precedence behavior
        Runtime evidence for existing engines.fromProfile precedence behavior
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/various/inspect_inference_surface.out
      Note: |-
        Runtime evidence for current inference/session API surface
        Runtime evidence of existing inference/session API surface
ExternalSources: []
Summary: Merged recommendation that combines GP-21 Go-parity work with the external full JS API proposal, with a hard-cutover inference-first JS API for geppetto bindings.
LastUpdated: 2026-02-25T00:00:00Z
WhatFor: Define the final JS API direction before implementation/commit, with explicit inference-runtime alignment and type-inference strategy.
WhenToUse: Use as the authoritative design reference before coding GP-21 JS API changes.
---


# Unified final JS API design for profile registry and schema-first middleware (inference-first)

## Executive Summary

This document merges:

1. the existing GP-21 parity analysis in this repository, and
2. the external comprehensive JS API proposal from OS-09.

Final recommendation:

1. **Adopt GP-21 parity primitives as mandatory base**:
   - `gp.profiles` for registry CRUD + resolve,
   - `gp.schemas` for middleware/extension schema discovery,
   - Go-owned resolution path (`profiles.Registry`, `middlewarecfg.*`) as source of truth.
2. **Adopt selected OS-09 ergonomics as an inference-first composition layer**:
   - `createEngineFactory` concept,
   - deterministic merge/precedence model,
   - structured error model,
   - middleware patch builder,
   - debug plan metadata.
3. **Adapt OS-09 API shape to Geppetto runtime constraints**, instead of porting it verbatim:
   - keep `require("geppetto")` module style,
   - compose into existing `Builder`/`Session`/`run|start|runAsync` lifecycle,
   - preserve current cancellation, tags, deadline, event, and tool-loop semantics.
4. **Apply a hard cutover for profile APIs**:
   - remove legacy model-centric `engines.fromProfile` semantics,
   - make profile-based engine composition registry-first only.

This gives “best of both”: no policy drift from Go runtime contracts, plus a high-level API that is usable for real inference workflows.

## Problem Statement and Scope

### Problem

The repository already supports profile registry and schema-first middleware configuration in Go, but JS bindings still expose only engine/session/tool primitives and not the profile/schema APIs.

Separately, OS-09 proposes a rich JS API (`createProfileRegistry`, `defineMiddleware`, `createEngineFactory`, `middlewarePatch`, `createProfileClient`) with strong ergonomics.

We need one final JS API design that:

1. fits Geppetto’s current runtime model,
2. preserves inference behavior,
3. avoids duplicating backend policy logic,
4. is ready for implementation in `pkg/js/modules/geppetto`.

### Scope

In scope:

1. final JS API design for `require("geppetto")` module,
2. merge strategy between GP-21 and OS-09 approaches,
3. inference-runtime compatibility requirements,
4. TypeScript declaration strategy (including config type inference ergonomics),
5. phased implementation plan in this repository.

Out of scope:

1. implementing the code changes (this is pre-commit design),
2. introducing frontend-specific package layout from OS-09 (`packages/engine/src/runtime-factory/*`),
3. rewriting external app HTTP handlers.

## Current Inference Runtime Constraints (Evidence)

The final API must not break these existing invariants:

1. `gp` currently exposes inference primitives and lifecycle methods:
   - exports: `createBuilder`, `createSession`, `runInference`, `engines`, `middlewares`, `tools` (`module.go:105-138`),
   - runtime experiment confirms this exact surface (`various/inspect_inference_surface.out`).
2. `Session` lifecycle supports sync + async + cancelable run handle:
   - `run`, `start`, `runAsync`, `cancelActive` (`api_sessions.go:377-425`, `429-605`).
3. Session enforces single active inference:
   - `ErrSessionAlreadyActive` and `active` handle guard (`session/session.go:12-35`, `199-203`, `232-239`).
4. Run context propagation exists and is already relied upon:
   - run tags + timeout/deadline (`api_sessions.go:571-605`),
   - session/inference IDs in context (`session/context.go:13-46`),
   - middleware JS context includes `sessionId`, `inferenceId`, `deadlineMs`, `tags` (`api_middlewares.go:81-114`).
5. Engine composition already supports tool-loop and observability hooks:
   - `WithMiddlewares`, `WithToolRegistry`, `WithLoopConfig`, `WithToolConfig`, `WithToolExecutor`, `WithEventSinks`, `WithSnapshotHook`, `WithPersister` (`api_sessions.go:208-245`).

Design implication: any new “factory” API must compile down to this exact builder/session machinery, not bypass it.

## What Each Proposal Gets Right

### GP-21 strengths (must keep)

1. Correct source of truth:
   - use `profiles.Registry` (`profiles/registry.go:57-78`) and `ResolveEffectiveProfile` (`profiles/service.go:128-178`) instead of recreating resolution logic in JS.
2. Correct schema contracts:
   - middleware schema from `Definition.ConfigJSONSchema()` (`middlewarecfg/definition.go:35-39`),
   - precedence/coercion/trace from `middlewarecfg.Resolver` (`middlewarecfg/resolver.go:15-204`),
   - deterministic build order from `middlewarecfg.BuildChain` (`middlewarecfg/chain.go:37-76`).
3. Correct extension schema merge model:
   - explicit schemas + middleware-derived typed-key schemas + codec schemas (`profiles/extensions.go:148-170`, `middleware_extensions.go:27-39`).
4. Correct identification of current gap:
   - no `profiles`/`schemas` namespace in JS exports and types (`module.go:105-138`, `geppetto.d.ts.tmpl`).

### OS-09 strengths (adopt selectively)

1. Better high-level ergonomics:
   - factory + patch builder pattern is cleaner than ad-hoc overrides.
2. Strong deterministic merge language:
   - explicit layer ordering and override policy.
3. Better error contract:
   - explicit machine-readable codes.
4. Better debugging contract:
   - expose resolved chain/layers/runtime fingerprint for investigations.
5. Better migration framing:
   - supports incremental adoption.

## What Not to Port Verbatim from OS-09

1. Do not create a parallel profile model separate from Go `profiles.Profile` schema; this causes drift.
2. Do not make runtime overrides default-open; Go policy already defines guardrails (`profiles/service.go:332-378`).
3. Do not force a separate frontend package structure into this repository.
4. Do not bypass existing `Session` execution path.

## Unified Final API (Recommended)

### 1) Required parity surface

### `gp.profiles`

```ts
profiles.listRegistries(): RegistrySummary[]
profiles.getRegistry(registrySlug?: string): ProfileRegistry
profiles.listProfiles(registrySlug?: string): Profile[]
profiles.getProfile(profileSlug: string, registrySlug?: string): Profile
profiles.resolve(input?: ResolveInput): ResolvedProfile
profiles.createProfile(profile: Profile, opts?: { registrySlug?: string; write?: WriteOptions }): Profile
profiles.updateProfile(profileSlug: string, patch: ProfilePatch, opts?: { registrySlug?: string; write?: WriteOptions }): Profile
profiles.deleteProfile(profileSlug: string, opts?: { registrySlug?: string; write?: WriteOptions }): void
profiles.setDefaultProfile(profileSlug: string, opts?: { registrySlug?: string; write?: WriteOptions }): void
```

Backed directly by `profiles.Registry` operations.

### `gp.schemas`

```ts
schemas.listMiddlewares(): Array<{
  name: string;
  version?: number;
  displayName?: string;
  description?: string;
  schema: Record<string, any>;
}>

schemas.listExtensions(): Array<{
  key: string;
  displayName?: string;
  description?: string;
  schema: Record<string, any>;
}>
```

Backed by:

1. `middlewarecfg.DefinitionRegistry.ListDefinitions()`,
2. typed-key middleware wrapper schemas,
3. `ExtensionCodecLister` + `ExtensionSchemaCodec`.

### 2) Inference-first composition surface

Add a focused factory namespace that composes through existing builder/session runtime.

### `gp.factories`

```ts
type FactoryCreateInput = {
  profile?: string;
  registry?: string;
  runtimeKeyFallback?: string;
  requestOverrides?: Record<string, any>;
  middlewarePatch?: MiddlewarePatch | ((b: MiddlewarePatchBuilder) => MiddlewarePatchBuilder);
  runDefaults?: { timeoutMs?: number; tags?: Record<string, any> };
  debug?: boolean;
};

interface EngineFactory {
  plan(input?: FactoryCreateInput): ComposedPlan;
  createBuilder(input?: FactoryCreateInput): Builder;
  createSession(input?: FactoryCreateInput): Session;
  createEngine(input?: FactoryCreateInput): Engine;
}

factories.createEngineFactory(options?: {
  defaultRegistrySlug?: string;
  defaultProfileSlug?: string;
  allowRuntimeOverridesByDefault?: boolean;
}): EngineFactory;

factories.middlewarePatch(): MiddlewarePatchBuilder;
```

### Why this shape

1. Reuses existing JS idioms (`Builder`, `Session`, `Engine`) rather than introducing an unrelated object graph.
2. Makes inference workflows first-class:
   - one-liner `createSession(...).run(...)`,
   - preserves `start()` streaming handle behavior.
3. Keeps OS-09 ergonomics (factory + patch builder) where they deliver real value.

### 3) Hard cutover behavior and naming

1. `engines.fromProfile(...)` is redefined as **registry profile resolution**, not model/provider precedence.
2. Calls that relied on `profile`/env/model fallback behavior are breaking changes and must migrate to:
   - `engines.fromConfig(...)` for direct provider/model construction, or
   - `factories.createEngineFactory(...).createEngine/createSession(...)` for profile-driven runtime composition.
3. If profile registry is not configured, `engines.fromProfile(...)` throws `PROFILE_REGISTRY_NOT_CONFIGURED`.
4. The old precedence path (`api_engines.go:81-94`) is removed in cutover implementation.

## Merge and Precedence Model (Final)

For factory composition, deterministic order:

1. Base `StepSettings` defaults (host/module defaults).
2. `profiles.resolve(...)` result (`EffectiveRuntime` + `EffectiveStepSettings`).
3. Optional request overrides, but only if profile policy allows.
4. Middleware patch operations.
5. Final middleware schema validation/coercion using `middlewarecfg.Resolver`.

For direct `engines.fromProfile(...)` after cutover:

1. resolve profile via `profiles.Registry.ResolveEffectiveProfile(...)`,
2. construct engine from resolved effective step settings,
3. no env/model fallback path.

For middleware config sources, preserve canonical source layers from `middlewarecfg/source.go` and reuse `Resolver` trace payloads.

## Inference-Critical Runtime Rules

1. Factory output must route through `enginebuilder.New(...)` + `session.NewSession()` path (same as `builderRef.buildSession`).
2. `run`, `start`, and `runAsync` semantics must be unchanged.
3. Context metadata must be preserved:
   - session ID, inference ID, tags, deadline.
4. Tool-loop config/hook behavior must remain compatible with current `Builder` options.
5. Cancellation must remain context-driven (execution handle cancel).
6. Hard cutover must not alter `run/start/runAsync` observable semantics.

## Structured Error Model (Final)

Adopt OS-09 style but align codes with Go contracts and module dependency states.

```ts
class GeppettoFactoryError extends Error {
  code:
    | "PROFILE_REGISTRY_NOT_CONFIGURED"
    | "MIDDLEWARE_DEFINITION_REGISTRY_NOT_CONFIGURED"
    | "EXTENSION_SCHEMA_PROVIDER_NOT_CONFIGURED"
    | "PROFILE_NOT_FOUND"
    | "REGISTRY_NOT_FOUND"
    | "POLICY_VIOLATION"
    | "MIDDLEWARE_NOT_FOUND"
    | "SCHEMA_VALIDATION_FAILED"
    | "OVERRIDE_NOT_ALLOWED"
    | "MERGE_CONFLICT"
    | "INVALID_PATCH"
    | "INFERENCE_COMPOSITION_FAILED";
  details?: Record<string, any>;
}
```

Rules:

1. Missing host dependencies are deterministic type errors at call-site.
2. Resolver validation failures must include middleware name, path, and schema type where possible.
3. Policy rejection should map to `POLICY_VIOLATION`/`OVERRIDE_NOT_ALLOWED` and preserve original reason.

## Observability and Debug Contract (Final)

Add a stable `plan()` return and optional `debug` payload.

```ts
interface ComposedPlan {
  registrySlug: string;
  profileSlug: string;
  runtimeKey: string;
  runtimeFingerprint: string;
  resolvedRuntime: {
    systemPrompt?: string;
    tools?: string[];
    middlewares?: Array<{ name: string; id?: string; enabled?: boolean; config?: Record<string, any> }>;
    stepSettingsPatch?: Record<string, any>;
  };
  middlewareDebug?: Array<{
    key: string;
    sources: Array<{ name: string; layer: string }>;
    paths: Array<{ path: string; value: any; steps: Array<{ source: string; layer: string; path: string }> }>;
  }>;
  mergeLayers: Array<{ layer: string; applied: boolean; details?: string }>;
}
```

Use existing `ResolvedConfig.BuildDebugPayload()` (`middlewarecfg/debug_payload.go:22-69`) where available.

## TypeScript Inference Strategy

Goal: keep runtime Go-owned, but provide useful compile-time inference for middleware config editing.

### Baseline (mandatory)

1. Strongly typed payload interfaces for profiles/schemas/factory inputs.
2. Narrow string unions for `code` fields and patch operation discriminants.

### Advanced inference (recommended)

Expose generic patch builder utilities that infer config types by middleware name from a user-provided map:

```ts
type MiddlewareConfigMap = Record<string, Record<string, any>>;

interface MiddlewarePatchBuilder<TMap extends MiddlewareConfigMap = Record<string, Record<string, any>>> {
  configure<K extends keyof TMap & string>(target: K, partialConfig: Partial<TMap[K]>): this;
  replace<K extends keyof TMap & string>(target: K, use: { name: K; id?: string; config?: Partial<TMap[K]> }): this;
  // ...append/prepend/enable/disable/remove/build
}

function middlewarePatch<TMap extends MiddlewareConfigMap = Record<string, Record<string, any>>>(): MiddlewarePatchBuilder<TMap>;
```

This gives practical config-type inference without claiming impossible automatic type derivation from arbitrary JSON Schema.

### What we intentionally avoid

1. No fake “schema-to-TypeScript automatic inference” promise for arbitrary JSON Schema.
2. No duplicated local profile schema types that diverge from Go payloads.

## Host Wiring Additions (Go side)

Extend `pkg/js/modules/geppetto.Options` and `moduleRuntime`:

```go
type Options struct {
  // existing fields...
  ProfileRegistry              profiles.Registry
  MiddlewareDefinitionRegistry middlewarecfg.DefinitionRegistry
  ExtensionCodecRegistry       profiles.ExtensionCodecRegistry
  ExtensionSchemas             map[string]map[string]any
  // optional defaults for factory
  DefaultRegistrySlug          profiles.RegistrySlug
  DefaultProfileSlug           profiles.ProfileSlug
}
```

If a dependency is absent, expose methods but throw explicit deterministic errors when called.

## File-Level Implementation Plan

### Phase 1: parity primitives + cutover scaffolding

1. Add option/runtime fields in `module.go`.
2. Add `api_profiles.go`.
3. Add `api_schemas.go`.
4. Wire namespaces in `installExports`.
5. Add tests for:
   - missing dependency errors,
   - CRUD + resolve happy paths,
   - schema list payload shape.
6. Add cutover guard tests for `engines.fromProfile` (registry-required behavior).

### Phase 2: inference-first factory

1. Add `api_factories.go` for:
   - `createEngineFactory`,
   - `middlewarePatch` builder,
   - `plan/createBuilder/createSession/createEngine`.
2. Route output through existing `builderRef.buildSession` path.
3. Reuse `middlewarecfg.Resolver` + `BuildChain` for middleware config resolution.
4. Add structured error type translation.

### Phase 3: TypeScript and docs

1. Update `spec/geppetto.d.ts.tmpl` with new namespaces/types/generics.
2. Regenerate `pkg/doc/types/geppetto.d.ts`.
3. Update docs:
   - `pkg/doc/topics/13-js-api-reference.md`,
   - `pkg/doc/topics/14-js-api-user-guide.md`.
4. Document **breaking migration** from legacy `engines.fromProfile` semantics to cutover behavior.
5. Remove legacy wording from JS API reference and examples.

### Phase 4: cutover implementation

1. Replace `api_engines.go` `fromProfile` precedence implementation with registry-backed resolution.
2. Remove legacy env/model precedence helper for `fromProfile`.
3. Keep `fromConfig` as explicit non-profile escape hatch.
4. Add explicit migration error messages for common misuses.

### Phase 5: examples and validation

1. Extend `cmd/examples/geppetto-js-lab/main.go` wiring for profile/schema deps.
2. Add ticket-style scripts/examples for:
   - factory plan output,
   - policy rejection,
   - middleware schema validation errors with path traces,
   - start/runAsync behavior through factory-created sessions,
   - cutover `fromProfile` behavior checks.

## Testing and Validation Strategy

1. Unit tests:
   - profile namespace behavior,
   - schema namespace behavior,
   - patch builder deterministic operation ordering,
   - structured error code mapping.
2. Integration tests:
   - factory -> session.run path parity,
   - factory -> session.start handle semantics parity,
   - tags/timeouts preserved into middleware context.
3. Regression tests:
   - existing non-factory APIs unchanged,
   - `engines.fromProfile` now fails without registry wiring and no longer uses env/model fallback.

## Risks and Mitigations

1. Risk: policy drift if JS re-implements profile resolution.
   - Mitigation: always call `profiles.Registry.ResolveEffectiveProfile`.
2. Risk: hard-cutover breakage in scripts relying on legacy `engines.fromProfile` fallback.
   - Mitigation: explicit migration guide, targeted error messages, and example rewrites.
3. Risk: factory bypasses existing lifecycle and breaks cancellation/streaming.
   - Mitigation: factory returns existing `Builder/Session` objects.
4. Risk: type overpromises for schema inference.
   - Mitigation: explicit generic map strategy, no automatic schema inference claims.

## Decision Summary

Final design decision:

1. **Yes** to GP-21 parity namespaces (`profiles`, `schemas`).
2. **Yes** to OS-09 composition ergonomics (`createEngineFactory`, patch builder, error/debug contract), but adapted to Geppetto runtime.
3. **No** to copying OS-09 package layout and parallel profile models.
4. **Yes** hard cutover: legacy `engines.fromProfile` semantics are removed.

This is the recommended pre-commit blueprint for implementing the final JS API in this repository.

## References

1. `pkg/js/modules/geppetto/module.go`
2. `pkg/js/modules/geppetto/api_sessions.go`
3. `pkg/js/modules/geppetto/api_builder_options.go`
4. `pkg/js/modules/geppetto/api_middlewares.go`
5. `pkg/js/modules/geppetto/api_engines.go`
6. `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
7. `pkg/inference/session/session.go`
8. `pkg/inference/session/context.go`
9. `pkg/inference/toolloop/enginebuilder/builder.go`
10. `pkg/inference/middlewarecfg/definition.go`
11. `pkg/inference/middlewarecfg/registry.go`
12. `pkg/inference/middlewarecfg/source.go`
13. `pkg/inference/middlewarecfg/resolver.go`
14. `pkg/inference/middlewarecfg/chain.go`
15. `pkg/inference/middlewarecfg/debug_payload.go`
16. `pkg/profiles/registry.go`
17. `pkg/profiles/service.go`
18. `pkg/profiles/extensions.go`
19. `pkg/profiles/middleware_extensions.go`
20. `pkg/doc/topics/01-profiles.md`
21. `ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/design-doc/01-profile-registry-middleware-schema-parity-analysis-for-js-bindings.md`
22. `/home/manuel/workspaces/2026-02-24/add-menus/go-go-os/ttmp/2026/02/24/OS-09-JS-ENGINE-API-DESIGN--js-engine-factory-profile-registry-and-middleware-schema-api-design/design-doc/01-comprehensive-js-api-design-for-engine-factories-profile-registry-and-schema-first-middleware.md`
23. `ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/various/inspect_inference_surface.out`
24. `ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/various/inspect_from_profile_semantics.out`
