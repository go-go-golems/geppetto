---
Title: Profile registry + middleware schema parity analysis for JS bindings
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
    - Path: geppetto/pkg/inference/middlewarecfg/debug_payload.go
      Note: Deterministic debug payload API for schema resolution
    - Path: geppetto/pkg/inference/middlewarecfg/definition.go
      Note: Middleware JSON schema contract surface
    - Path: geppetto/pkg/inference/middlewarecfg/registry.go
      Note: ListDefinitions source for schema catalog generation
    - Path: geppetto/pkg/inference/middlewarecfg/resolver.go
      Note: Schema-based coercion and precedence logic
    - Path: geppetto/pkg/js/modules/geppetto/api_engines.go
      Note: fromProfile implementation semantics for gap analysis
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Current JS export and option boundaries missing profile/schema APIs
    - Path: geppetto/pkg/js/modules/geppetto/plugins_module.go
      Note: Plugin module scope evidence
    - Path: geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl
      Note: Type-level evidence of missing profile/schema namespaces
    - Path: geppetto/pkg/profiles/extensions.go
      Note: Extension schema discovery capability interfaces
    - Path: geppetto/pkg/profiles/middleware_extensions.go
      Note: Typed-key middleware config mapping foundation
    - Path: geppetto/pkg/profiles/registry.go
      Note: Registry service contract used as Go parity baseline
    - Path: geppetto/pkg/profiles/service.go
      Note: Resolve and CRUD semantics that JS bindings currently cannot access
    - Path: geppetto/pkg/sections/profile_registry_source.go
      Note: Registry-backed profile loading in section middleware
    - Path: geppetto/pkg/sections/sections.go
      Note: Registry middleware wired into runtime configuration flow
ExternalSources: []
Summary: Gap analysis and implementation plan to expose profile registry + middleware/extension schema discovery to require("geppetto") JS bindings.
LastUpdated: 2026-02-25T00:00:00Z
WhatFor: Align JS bindings with Go profile-registry and schema-discovery capabilities used by app APIs.
WhenToUse: Use when implementing or reviewing profile/schema parity work in pkg/js/modules/geppetto.
---


# Profile registry + middleware schema parity analysis for JS bindings

> Note: This document captures the initial parity-first recommendation. The final direction is now hard cutover and is documented in `design-doc/02-unified-final-js-api-design-inference-first.md`.
>
> Update (GP-31 alignment): `engines.fromProfile` is now registry-backed in code. Remaining GP-21 implementation gap is the missing `gp.profiles`/`gp.schemas` namespaces and runtime-surface cleanup (remove runtime registry selector inputs from JS runtime APIs).

## Executive Summary

`geppetto/pkg/profiles` and `geppetto/pkg/inference/middlewarecfg` now provide the Go-side primitives required for profile-registry runtime resolution and middleware/extension schema discovery, but `require("geppetto")` does not expose these capabilities.

The current JS API can create engines/sessions, compose middleware, and manage tool registries, but cannot:

1. query or mutate profile registries,
2. resolve an effective profile from registry data,
3. discover middleware JSON schemas,
4. discover extension JSON schemas.

This leaves JS consumers unable to participate in the same profile-driven runtime + schema-driven UX flow that Go application APIs already support and document.

## Problem Statement and Scope

### Problem

We have Go runtime support for:

1. registry-first profiles (`profiles.Registry`, `ResolveEffectiveProfile`),
2. middleware schema contracts (`middlewarecfg.Definition.ConfigJSONSchema()`),
3. extension schema-capable codecs (`profiles.ExtensionSchemaCodec` + `ExtensionCodecLister`).

But JS bindings stop at inference/session middleware composition and do not expose registry/schema operations.

### Scope

In scope:

1. `geppetto/pkg/profiles/*`
2. `geppetto/pkg/inference/middlewarecfg/*`
3. `geppetto/pkg/js/modules/geppetto/*`
4. JS API docs/types under `pkg/doc/topics/13-js-api-reference.md`, `pkg/doc/topics/14-js-api-user-guide.md`, `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`, `pkg/doc/types/geppetto.d.ts`

Out of scope:

1. pinocchio web-chat HTTP handler implementation (`/api/chat/*`) in external app repos,
2. frontend code outside this repository.

## Current-State Architecture (Evidence-Backed)

### A) Go profile-registry domain is fully present

1. Profile runtime model includes middleware + extensions:
   - `RuntimeSpec.Middlewares`: `pkg/profiles/types.go:17`
   - `Profile.Extensions`: `pkg/profiles/types.go:59`
2. Service abstraction is explicit and CRUD-capable:
   - `Registry` interface: `pkg/profiles/registry.go:57-78`
   - `ResolveEffectiveProfile`: `pkg/profiles/registry.go:63`
3. Concrete store-backed implementation exists:
   - `StoreRegistry.ResolveEffectiveProfile`: `pkg/profiles/service.go:128-178`
   - create/update/delete/default flows: `pkg/profiles/service.go:180-280`
4. CLI section resolution already migrated to registry abstraction:
   - `GatherFlagsFromProfileRegistry`: `pkg/sections/profile_registry_source.go:17-94`
   - wired in command middleware chain: `pkg/sections/sections.go:283-297`

### B) Middleware schema and extension schema primitives exist in Go

1. Middleware definitions expose JSON schema contract:
   - `Definition.ConfigJSONSchema()`: `pkg/inference/middlewarecfg/definition.go:35-39`
2. Definition registry supports lookup/list for schema catalog building:
   - `ListDefinitions()`: `pkg/inference/middlewarecfg/registry.go:12-16, 74-98`
3. Schema-aware resolver exists with layered precedence + coercion/validation:
   - resolver core: `pkg/inference/middlewarecfg/resolver.go:99-204`
   - per-path trace/debug payload: `pkg/inference/middlewarecfg/debug_payload.go:15-69`
4. Extension codecs can expose schemas and can be listed:
   - `ExtensionSchemaCodec.JSONSchema()`: `pkg/profiles/extensions.go:148-153`
   - `ExtensionCodecLister.ListCodecs()`: `pkg/profiles/extensions.go:166-170`
   - in-memory list implementation: `pkg/profiles/extensions.go:218-240`
5. Middleware config has typed-key extension mapping support:
   - `MiddlewareConfigExtensionKey`: `pkg/profiles/middleware_extensions.go:27-39`
   - project inline config to typed-key extensions: `pkg/profiles/middleware_extensions.go:53-80`
6. Base repo currently has no production (`non-test`) `ConfigJSONSchema()` middleware definition implementations:
   - `rg -n "func \\(.*\\) ConfigJSONSchema\\(\\) map\\[string\\]any" geppetto/pkg --glob '!**/*_test.go'` returned no matches
   - implication: concrete schema catalogs are expected to be application-owned today

### C) Go-side API contract is documented for app endpoints

Docs describe application schema discovery endpoints and payload contracts:

1. `/api/chat/schemas/middlewares`, `/api/chat/schemas/extensions`: `pkg/doc/topics/01-profiles.md:226-279`
2. same contracts in middleware topic: `pkg/doc/topics/09-middlewares.md:263-312`
3. operational runbook with contract checks: `pkg/doc/playbooks/06-operate-sqlite-profile-registry.md:55-77`

### D) JS bindings currently do not expose profile/schema APIs

1. Top-level exports are limited to version/builder/session/inference/turns/engines/middlewares/tools:
   - `installExports`: `pkg/js/modules/geppetto/module.go:105-138`
2. Runtime inventory experiment confirms no `profiles` or `schemas` namespace:
   - script: `ttmp/.../scripts/inspect_geppetto_exports.js`
   - output: `ttmp/.../various/inspect_geppetto_exports.out`
3. `geppetto/plugins` module exists but only for extractor plugin helpers:
   - loader exports: `pkg/js/modules/geppetto/plugins_module.go:21-85`
   - runtime inventory: `ttmp/.../scripts/inspect_geppetto_plugins_exports.js`
4. JS module options include tool registry + middleware factories, but no profile registry or schema providers:
   - `Options` struct: `pkg/js/modules/geppetto/module.go:33-41`
5. `engines.fromProfile(...)` does not consult profile registry service; it builds step settings from simple precedence + env/model inference:
   - precedence and default model fallback: `pkg/js/modules/geppetto/api_engines.go:81-94`
   - step-settings construction path: `pkg/js/modules/geppetto/api_engines.go:96-210`

## Gap Analysis Against Requested Outcome

### Gap 1: No profile-registry read/write surface in JS

Impact:

1. JS code cannot list/get/create/update/delete profiles.
2. JS cannot call canonical `ResolveEffectiveProfile` behavior.
3. JS cannot participate in profile editor workflows without out-of-band HTTP clients.

Evidence:

- missing exports in `module.go:105-138`
- no profile methods in `geppetto.d.ts.tmpl:245-264`

### Gap 2: No middleware/extension schema discovery surface in JS

Impact:

1. JS cannot build schema-driven profile editors from the same catalog contract documented for Go app APIs.
2. JS runtime cannot validate/form-generate against middleware schemas locally.

Evidence:

- schema primitives exist in Go (`middlewarecfg/definition.go`, `profiles/extensions.go`) but are not wired into JS module options/exports.

### Gap 3: No host dependency injection for registry/schema providers

Impact:

Even if JS export methods were added, `Options` currently has no slots for `profiles.Registry`, `middlewarecfg.DefinitionRegistry`, or extension schema provider/lister.

Evidence:

- `Options` field list in `module.go:33-41`

### Gap 4: Profile naming in JS (`fromProfile`) is model-centric, not registry-centric

Impact:

Method name suggests registry profile semantics but current behavior is provider model selection precedence with env fallback.

Evidence:

- `profileFromPrecedence`: `api_engines.go:81-94`
- engine construction from settings, not `profiles.Registry`: `api_engines.go:96-227`

## Proposed Solution

### Design Goals

1. Add JS parity for profile registry + schema discovery without breaking existing JS scripts.
2. Keep host opt-in wiring: features available when host provides registry/definition dependencies.
3. Reuse existing Go domain/service types; do not duplicate profile logic in JS.

### Proposed JS API Surface

#### 1) `gp.profiles` namespace

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

#### 2) `gp.schemas` namespace

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

### Host Wiring Additions

Extend module options (Go side):

```go
type Options struct {
  // existing fields...
  ProfileRegistry             profiles.Registry
  MiddlewareDefinitionRegistry middlewarecfg.DefinitionRegistry
  ExtensionCodecRegistry      profiles.ExtensionCodecRegistry
  ExtensionSchemas            map[string]map[string]any // optional explicit override catalog
}
```

### Runtime Behavior Rules

1. If a namespace method is called but dependency is unset, throw deterministic error:
   - `profile registry not configured`
   - `middleware definition registry not configured`
2. Preserve existing top-level APIs unchanged.
3. `engines.fromProfile` remains backward compatible; do not silently change semantics in first parity patch.

## Pseudocode and Key Flows

### A) Middleware schema list flow

```go
func (m *moduleRuntime) schemasListMiddlewares(...) goja.Value {
  if m.middlewareDefinitionRegistry == nil {
    panic(typeError("middleware definition registry not configured"))
  }
  defs := m.middlewareDefinitionRegistry.ListDefinitions()
  rows := []map[string]any{}
  for _, def := range defs {
    rows = append(rows, map[string]any{
      "name": def.Name(),
      "schema": copyStringAnyMap(def.ConfigJSONSchema()),
      // optional metadata adapter hooks if available
    })
  }
  return m.toJSValue(rows)
}
```

### B) Extension schema list flow

```go
func (m *moduleRuntime) schemasListExtensions(...) goja.Value {
  catalog := map[string]map[string]any{}

  // 1) explicit app-provided extension schemas
  merge(catalog, m.extensionSchemas)

  // 2) middleware-derived typed-key schemas (if definitions available)
  for _, def := range m.middlewareDefinitionRegistry.ListDefinitions() {
    key := profiles.MiddlewareConfigExtensionKey(def.Name())
    catalog[key.String()] = middlewareConfigWrapperSchema(def.ConfigJSONSchema())
  }

  // 3) codec registry schemas
  if lister, ok := m.extensionCodecRegistry.(profiles.ExtensionCodecLister); ok {
    for _, codec := range lister.ListCodecs() {
      if schemaCodec, ok := codec.(profiles.ExtensionSchemaCodec); ok {
        key := codec.Key().String()
        if _, exists := catalog[key]; !exists {
          catalog[key] = copyStringAnyMap(schemaCodec.JSONSchema())
        }
      }
    }
  }

  return m.toJSValue(sortedRows(catalog))
}
```

### C) Profile resolve flow

```go
func (m *moduleRuntime) profilesResolve(call goja.FunctionCall) goja.Value {
  reg := requireProfileRegistry(m)
  input := decodeResolveInput(call.Argument(0))
  // Convert string slugs -> typed slugs via profiles.Parse* adapters.
  resolved, err := reg.ResolveEffectiveProfile(ctx, input)
  if err != nil { panic(goError(err)) }
  return m.toJSValue(encodeResolvedProfile(resolved))
}
```

## Phased Implementation Plan

### Phase 1: Runtime plumbing + minimal APIs

1. Add option fields in `pkg/js/modules/geppetto/module.go`.
2. Add new files:
   - `pkg/js/modules/geppetto/api_profiles.go`
   - `pkg/js/modules/geppetto/api_schemas.go`
3. Wire `profiles` + `schemas` namespaces in `installExports`.
4. Add explicit errors for missing host dependencies.

### Phase 2: Type/docs parity

1. Extend `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl` with new namespaces and payload interfaces.
2. Regenerate `pkg/doc/types/geppetto.d.ts` via existing generation flow.
3. Update:
   - `pkg/doc/topics/13-js-api-reference.md`
   - `pkg/doc/topics/14-js-api-user-guide.md`
   with profile/schema examples and host wiring requirements.

### Phase 3: Tests and examples

1. Add JS module tests for:
   - missing dependency errors,
   - happy-path list/get/resolve/profile CRUD,
   - schema list payload shape.
2. Add/extend `cmd/examples/geppetto-js-lab` host wiring for profile/schema dependencies in test harness.
3. Add example scripts under `examples/js/geppetto/` for schema discovery and registry CRUD smoke tests.

### Phase 4: Optional semantic cleanup

1. Evaluate renaming/clarifying `engines.fromProfile` semantics (model selector vs registry profile).
2. If renamed, keep alias and deprecation window.

## Testing and Validation Strategy

1. Unit tests:
   - `go test ./pkg/js/modules/geppetto`
   - `go test ./pkg/profiles ./pkg/inference/middlewarecfg`
2. Script-level smoke checks (already added in ticket workspace):
   - `scripts/inspect_geppetto_exports.js`
   - `scripts/inspect_geppetto_plugins_exports.js`
3. Contract checks for new schema payloads:
   - deterministic ordering,
   - typed-key format for extension keys,
   - valid JSON object `schema` field.

## Risks, Alternatives, Open Questions

### Risks

1. App-specific metadata (`display_name`, `description`, `version`) may not exist in raw `middlewarecfg.Definition` today.
2. Base Geppetto may legitimately return empty middleware schema catalogs unless host apps register concrete definitions.
3. Not all middleware schemas map cleanly to UI fields; schema consumers must handle nested/complex JSON schema.
4. Mixing explicit extension schema overrides with codec-derived schemas needs stable precedence rules.

### Alternatives Considered

1. Expose only opaque HTTP bridge helpers in JS.
   - Rejected: leaks transport concerns into module and duplicates app API clients.
2. Keep schema discovery Go-only and let JS consume HTTP endpoints externally.
   - Rejected: prevents embedded JS runtimes from parity with host runtime capabilities.
3. Change `engines.fromProfile` semantics immediately to use registry service.
   - Rejected initially: high compatibility risk; better as explicit new API and migration.

### Open Questions

1. Where should middleware display metadata come from in base Geppetto (`Definition` extension interface vs host-side decorator)?
2. Should JS profile CRUD include optimistic concurrency defaults (`expected_version`) or require explicit write options each call?
3. Should schema namespaces live under `gp.profiles.schemas` or top-level `gp.schemas`? (proposal uses top-level for parity with documented endpoint grouping).

## References (Key Evidence)

1. `pkg/profiles/registry.go`
2. `pkg/profiles/service.go`
3. `pkg/profiles/extensions.go`
4. `pkg/profiles/middleware_extensions.go`
5. `pkg/inference/middlewarecfg/definition.go`
6. `pkg/inference/middlewarecfg/registry.go`
7. `pkg/inference/middlewarecfg/resolver.go`
8. `pkg/inference/middlewarecfg/debug_payload.go`
9. `pkg/sections/profile_registry_source.go`
10. `pkg/sections/sections.go`
11. `pkg/js/modules/geppetto/module.go`
12. `pkg/js/modules/geppetto/api_engines.go`
13. `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
14. `pkg/js/modules/geppetto/plugins_module.go`
15. `pkg/doc/topics/01-profiles.md`
16. `pkg/doc/topics/09-middlewares.md`
17. `pkg/doc/playbooks/06-operate-sqlite-profile-registry.md`
18. `ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_geppetto_exports.js`
19. `ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/various/inspect_geppetto_exports.out`
20. `ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_geppetto_plugins_exports.js`
