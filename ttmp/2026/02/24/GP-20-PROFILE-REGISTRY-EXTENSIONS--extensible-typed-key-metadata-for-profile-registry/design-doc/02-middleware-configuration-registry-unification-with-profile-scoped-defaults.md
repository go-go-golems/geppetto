---
Title: Middleware Configuration Registry Unification with Profile-Scoped Defaults
Ticket: GP-20-PROFILE-REGISTRY-EXTENSIONS
Status: active
Topics:
    - architecture
    - geppetto
    - pinocchio
    - chat
    - frontend
    - persistence
    - migration
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/registry.go
      Note: Existing registry pattern with duplicate registration protection.
    - Path: geppetto/pkg/inference/middleware/middleware.go
      Note: |-
        Core middleware function type that should remain unchanged.
        Core middleware function contract preserved by proposed design
    - Path: geppetto/pkg/inference/tools/registry.go
      Note: |-
        Existing interface+registry pattern to mirror for middleware definitions.
        Reference interface+registry pattern used as unification baseline
    - Path: geppetto/pkg/profiles/service.go
      Note: Effective runtime resolution and override handling.
    - Path: geppetto/pkg/profiles/types.go
      Note: RuntimeSpec and MiddlewareUse payload model for profile-scoped defaults.
    - Path: geppetto/pkg/sections/sections.go
      Note: Existing glazed layered-source pattern used as the reference integration style.
    - Path: geppetto/ttmp/2026/02/24/GP-20-PROFILE-REGISTRY-EXTENSIONS--extensible-typed-key-metadata-for-profile-registry/sources/local/middleware-config-proposals.md
      Note: Imported proposal source analyzed and synthesized
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go
      Note: |-
        Current app composer that hardcodes middleware chain and ignores profile middleware list.
        Current hardcoded middleware chain that should consume profile runtime middlewares
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: |-
        Current profile middleware override parsing and runtime build flow.
        Current ad-hoc middleware override parsing to replace with registry-based decoding
    - Path: pinocchio/pkg/inference/runtime/engine.go
      Note: Current map-based middleware factory composition point.
    - Path: pinocchio/pkg/sem/registry/registry.go
      Note: Existing typed registration pattern to mirror for middleware config decode.
    - Path: pinocchio/pkg/webchat/router.go
      Note: Router middleware/tool registration and composition wiring.
ExternalSources:
    - local:middleware-config-proposals.md
Summary: Research and design synthesis for unifying middleware configuration around a typed middleware definition registry, aligned with existing registry patterns and profile-scoped defaults.
LastUpdated: 2026-02-25T00:18:00-05:00
WhatFor: Define how to integrate the imported middleware configuration proposals with the current Geppetto/Pinocchio/Go-Go-OS architecture.
WhenToUse: Use when implementing middleware configuration refactors, profile-scoped middleware defaults, or app-level middleware extensions without binary flags.
---


# Middleware Configuration Registry Unification with Profile-Scoped Defaults

## Executive Summary

The imported proposal (`sources/local/middleware-config-proposals.md`) is directionally correct and should be adopted with minimal adaptation to the current codebase: keep middleware runtime shape as `func(HandlerFunc) HandlerFunc`, but introduce a first-class middleware definition registry for config schema, layered parsing, validation, and instantiation.

The recommended target is:

1. adopt "Proposal 2" as the default architecture (typed middleware definitions + glazed values layering),
2. add "Proposal 3-lite" fields (`id`, `enabled`) to middleware uses for stable instance semantics and future patching,
3. perform a hard cutover (no compatibility shim) so the middleware path has a single architecture,
4. keep JSON-schema export (Proposal 4) as a second phase.

This unifies design patterns already used across the repo:

- `profiles.Registry` for profile lifecycle,
- `tools.ToolRegistry` for tool registration,
- event and SEM handler registries for decoder/translator extension.

It also solves the current gap where middleware config is still ad-hoc (`map[string]MiddlewareBuilder`, `any` config blobs, manual per-request parsing), which prevents robust profile-scoped middleware defaults from scaling.

## Problem Statement

### What Works Today

- Profiles can carry runtime defaults (`system_prompt`, `middlewares`, `tools`) in `geppetto/pkg/profiles/types.go`.
- Profile stores are already durable and backend-agnostic (YAML + SQLite JSON payload rows).
- Runtime composition is app-owned and pluggable via `RuntimeBuilder`.

### What Is Missing

Middleware configuration currently has no unified typed contract layer:

- `pinocchio/pkg/inference/runtime/engine.go` consumes `[]MiddlewareSpec` plus `map[string]MiddlewareBuilder`.
- `MiddlewareBuilder` takes `cfg any` and returns middleware directly; decode/validation is app-local.
- `pinocchio/cmd/web-chat/runtime_composer.go` manually parses middleware overrides (`[]any -> []MiddlewareSpec`) with no middleware-specific schema validation.
- `go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go` hardcodes middleware instances and does not consume profile-provided middleware list, so profile-scoped middleware defaults are partially inert.

### Architectural Consequences

- No self-describing middleware config schema for CLI/UI/API introspection.
- No provenance chain for middleware config values (defaults/profile/config/env/flags/request).
- No uniform validation boundary for profile CRUD and runtime composition.
- Divergence risk between pinocchio and go-go-os runtime behavior.

## Current-State Research Findings

### A. Existing Registry and Interface Patterns Are Mature

The codebase already favors "interface + registry + in-memory implementation":

- `geppetto/pkg/profiles/registry.go`: `Registry` interface with `StoreRegistry`.
- `geppetto/pkg/inference/tools/registry.go`: `ToolRegistry` with thread-safe in-memory implementation.
- `geppetto/pkg/events/registry.go`: codec/encoder registries with duplicate-guard semantics.
- `pinocchio/pkg/sem/registry/registry.go`: typed registration (`RegisterByType[T]`).
- `pinocchio/pkg/webchat/timeline_registry.go`: event-type handler registry.

Conclusion: introducing a middleware definition registry is aligned with existing architecture and team patterns.

### B. Middleware Runtime Type Should Stay As-Is

`geppetto/pkg/inference/middleware/middleware.go` defines:

```go
type Middleware func(HandlerFunc) HandlerFunc
```

This is simple, composable, and already integrated into engine builders. Replacing it is unnecessary. Configuration/introspection concerns should be solved in a separate registry layer.

### C. Profile Runtime Already Carries Middleware Defaults, But Decode Is Untyped

`RuntimeSpec` includes:

```go
Middlewares []MiddlewareUse // {Name string, Config any}
```

The profile service validates middleware names are non-empty but does not validate middleware-specific config shape. That validation should move into a middleware definition registry codec layer.

### D. There Is One Structural Inconsistency To Resolve

`Router.RegisterMiddleware(...)` stores factories in `r.mwFactories`, but this map is currently not consumed by the default `convRuntimeComposer` path. Middleware composition happens inside app runtime composers, not the router map.

This is a design smell and must be resolved. Two concrete options are viable:

#### Option D1: Centralize Middleware Definitions in Runtime Composer Stack

Make runtime-composer infrastructure the single owner of middleware definitions and config decoding.

Pseudo:

```go
type ProfileRuntimeComposer struct {
    parsed *values.Values
    mwReg  middlewarecfg.Registry
    deps   middlewarecfg.BuildDeps
}

func (c *ProfileRuntimeComposer) Compose(ctx context.Context, req ConversationRuntimeRequest) (ComposedRuntime, error) {
    uses := toMiddlewareUses(req.ResolvedProfileRuntime)

    schema, err := c.mwReg.SchemaForUses(uses)
    if err != nil { return ComposedRuntime{}, err }
    parsed := values.New()

    err = sources.Execute(schema, parsed,
        middlewarecfg.SourceFromDefinitionDefaults(uses),
        middlewarecfg.SourceFromProfileConfig(uses),
        middlewarecfg.SourceFromRequestOverrides(req.RuntimeOverrides),
    )
    if err != nil { return ComposedRuntime{}, err }

    chain, err := c.mwReg.BuildChain(ctx, c.deps, uses, parsed)
    if err != nil { return ComposedRuntime{}, err }

    eng, err := buildEngineWithChain(chain)
    if err != nil { return ComposedRuntime{}, err }
    return ComposedRuntime{Engine: eng}, nil
}
```

Outcome:

- `Router.RegisterMiddleware(...)` either disappears, or is replaced with `RegisterMiddlewareDefinition(...)` that writes into the same registry used by composer.
- No shadow registry on router with dead state.

#### Option D2: Remove Dead Registration API, Keep App-Composers Ad-Hoc

Delete unused middleware-registration surfaces and accept app-local map factories for now.

Pseudo:

```go
// remove from Router:
// mwFactories map[string]MiddlewareBuilder
// func (r *Router) RegisterMiddleware(...)
// remove forwarding from Server too

composer := newProfileRuntimeComposer(parsed, middlewareFactories) // app-owned
srv := webchat.NewServer(ctx, parsed, fs, webchat.WithRuntimeComposer(composer))
```

Outcome:

- Clarifies current ownership boundary immediately.
- Does not by itself deliver typed schema/validation/provenance.

#### Recommendation for D

Adopt **D1** and fold the cleanup part of D2 into it as a hard cutover:

- centralize middleware definition registry in runtime composer path,
- remove dead router/server middleware registration API instead of maintaining aliases or compatibility wrappers.

## Proposed Solution

### 1. Introduce `middlewarecfg` Package (Core Abstraction)

Create `geppetto/pkg/inference/middlewarecfg` with:

```go
type Use struct {
    Name    string `json:"name" yaml:"name"`
    ID      string `json:"id,omitempty" yaml:"id,omitempty"`
    Enabled *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
    Config  any    `json:"config,omitempty" yaml:"config,omitempty"`
}

type BuildDeps struct {
    // app-owned injected dependencies (logger, stores, services, etc.)
}

type Definition interface {
    Name() string
    Description() string
    Section(instanceKey string) (schema.Section, error)
    Build(ctx context.Context, deps BuildDeps, parsed *values.Values, instanceKey string) (middleware.Middleware, error)
}

type Registry interface {
    Register(Definition) error
    Get(name string) (Definition, bool)
    SchemaForUses(uses []Use) (*schema.Schema, error)
    BuildChain(ctx context.Context, deps BuildDeps, uses []Use, parsed *values.Values) ([]middleware.Middleware, error)
}
```

### 2. Hard Cutover Profile Middleware Shape (No Compatibility Shims)

Extend `geppetto/pkg/profiles/types.go` `MiddlewareUse` with:

- `ID` (stable instance key),
- `Enabled` (tri-state default/true/false).

Cutover rules:

- migrate all in-repo producers/consumers in one pass,
- remove legacy adapter paths (`map[string]MiddlewareBuilder` compatibility layers),
- no dual-path runtime support in core packages.

### 3. Standardize Layering for Middleware Config

Adopt the same precedence model already used in Geppetto sections workflows:

1. middleware definition defaults,
2. profile-scoped middleware config (`RuntimeSpec.Middlewares`),
3. config file / env / CLI layers (if command mode),
4. request overrides (if allowed by policy).

In web APIs without command mode, apply the subset that exists:

- definition defaults -> profile defaults -> request overrides.

### 4. Integrate With Profile Resolution, Not Replace It

Keep `profiles.ResolveEffectiveProfile` as the owner of high-level runtime merge policy. Then add middlewarecfg validation/build in runtime composition stage:

```text
resolve profile runtime -> obtain []MiddlewareUse
-> middlewarecfg.SchemaForUses
-> sources.Execute(layered middleware values)
-> registry.BuildChain
-> engine builder
```

### 5. App-Owned Middleware Remains App-Owned

App packages (pinocchio cmd/web-chat, go-go-os inventory server, third-party binaries) register their middleware definitions and dependencies. Core Geppetto only provides:

- base interfaces,
- helper typed-definition generic,
- optional built-in definitions for geppetto-owned middlewares.

This matches current ownership boundaries while removing ad-hoc config parsing.

### 6. Full JSON Schema Model: How Middleware Parameters Are Described

In a JSON-schema-first phase, each middleware definition provides a canonical parameter schema that describes its config contract:

- property names,
- types (`string`, `integer`, `boolean`, `array`, `object`),
- required fields,
- default values,
- descriptions/examples,
- constraints (enum/min/max/pattern).

Pseudo shape:

```go
type Definition interface {
    Name() string
    Description() string
    ConfigJSONSchema() *jsonschema.Schema
    BuildFromAny(ctx context.Context, deps BuildDeps, cfg any) (middleware.Middleware, error)
}
```

Example schema (system prompt middleware):

```json
{
  "type": "object",
  "properties": {
    "prompt": {
      "type": "string",
      "description": "System prompt text to inject or replace",
      "minLength": 1
    },
    "mode": {
      "type": "string",
      "enum": ["replace-first", "append"],
      "default": "replace-first"
    }
  },
  "required": ["prompt"],
  "additionalProperties": false
}
```

In this setup, middleware parameters are no longer "opaque `any`"; they are documented and machine-validated contracts consumable by:

- profile CRUD validation,
- web UI form generation,
- API clients and SDKs,
- runtime compose decode.

Cutover note: JSON Schema is the canonical middleware parameter contract immediately.  
Glazed sections are generated adapters for CLI/help only, and ParseStep-style provenance tracking is implemented in the schema resolver layer from day one.

## Design Decisions

### Decision 1: Adopt Proposal 2 as Core

Reason:

- integrates directly with existing glazed schemas and source layering,
- provides config provenance (`FieldValue.Log`) without new infrastructure,
- gives type-safe decode at middleware boundaries.

### Decision 2: Add Proposal 3-Lite (`id`, `enabled`) Early

Reason:

- avoids schema churn later when patch/reorder semantics are needed,
- enables multi-instance middleware cleanly,
- improves UI/editor stability with persistent instance identity.

### Decision 3: Defer Full JSON-Schema-First Model

Reason:

- phase-1 objective is runtime correctness and profile integration,
- glazed is already deeply integrated in command stack,
- JSON schema export can be layered later from typed config structs.

### Decision 4: Preserve Existing Middleware Function Type

Reason:

- minimal runtime risk,
- no churn in engine/inference execution stack,
- decouples runtime semantics from configuration concerns.

### Decision 5: Align Registry Semantics With Existing Registries

Use same principles as tools/events registries:

- duplicate registration returns error,
- thread-safe reads/writes,
- deterministic behavior for tests (clear/reset hooks in test-only paths).

### Decision 6: Hard Cutover, No Backward Compatibility Branch

Reason:

- simpler code and test matrix,
- avoids long-lived dual pathways,
- matches current migration direction (remove env/legacy toggles, single behavior path).

## Integration Diagram

```text
Profile Registry (YAML/SQLite/API)
  RuntimeSpec.Middlewares[] + Config(any)
              |
              v
ResolveEffectiveProfile (profiles service)
              |
              v
Runtime Composer (app-owned)
  uses -> middlewarecfg.Registry.SchemaForUses(uses)
       -> sources.Execute(layered values)
       -> middlewarecfg.Registry.BuildChain(...)
              |
              v
enginebuilder.Builder{Middlewares: ...}
```

## Pseudocode: Unified Runtime Compose

```pseudo
function ComposeRuntime(req):
  resolved = profileRegistry.ResolveEffectiveProfile(req.registry, req.profile, req.overrides)

  uses = convertRuntimeMiddlewares(resolved.EffectiveRuntime.Middlewares)

  schema = middlewareRegistry.SchemaForUses(uses)
  parsed = values.New()

  layeredSources = [
    SourceFromMiddlewareDefaults(uses),   // derived from definition defaults
    SourceFromProfileConfig(uses),        // runtime.middlewares[].config
    SourceFromRequestOverrides(req),      // when policy allows
  ]
  sources.Execute(schema, parsed, layeredSources...)

  mws = middlewareRegistry.BuildChain(ctx, deps, uses, parsed)

  return BuildEngine(stepSettings, systemPrompt, mws, allowedTools)
```

## Concrete Unification Work Across Repositories

### Geppetto

- Add `middlewarecfg` package (definitions, registry, typed helper).
- Extend profile middleware type with optional instance identity fields.
- Add middleware config validation hooks for known definitions (optional strict mode).

### Pinocchio

- Replace ad-hoc middleware override parsing in `cmd/web-chat/runtime_composer.go`.
- Build middleware chain through `middlewarecfg.Registry`.
- Keep app-owned middleware registration in command layer, but register definitions instead of raw `func(any) Middleware`.

### Go-Go-OS Inventory Server

- Update `internal/pinoweb/runtime_composer.go` to consume `ResolvedProfileRuntime.Middlewares` rather than hardcoded-only middleware list.
- Register inventory middleware definitions in a registry and apply profile-scoped defaults.
- Keep strict mode for request overrides where desired; this is policy, not architecture.

## Hard Cutover Plan

1. Introduce `middlewarecfg` and migrate pinocchio/go-go-os composers in the same change window.
2. Remove `Router.RegisterMiddleware`, `Server.RegisterMiddleware`, and `Router.mwFactories`.
3. Update all app bootstrap paths to register definitions into composer-owned middleware registry.
4. Remove legacy map-factory parsing code and tests that exercise it.
5. Update docs/examples to only show the new definition-registry path.

## Risks and Mitigations

- Risk: schema/decoder mismatch between middleware definition and stored profile config.  
  Mitigation: validate at profile create/update (or at runtime compose with explicit error mapping).

- Risk: registry ownership drift across packages.  
  Mitigation: define clear namespace and ownership conventions for middleware names and config keys.

- Risk: duplicate middleware registrations in large app binaries.  
  Mitigation: duplicate registration must return deterministic errors.

- Risk: go-go-os runtime behavior changes due to now honoring profile middlewares.  
  Mitigation: keep explicit defaults and add integration tests asserting chain order and generated events.

## Alternatives Considered

### A) Keep Current `map[string]MiddlewareBuilder` + manual parsing

Rejected:

- no schema/provenance,
- repeated parsing code in each app,
- poor extensibility for profile-scoped defaults.

### B) Hard-code all middleware config fields into `RuntimeSpec`

Rejected:

- causes core schema churn,
- couples app-specific middleware evolution to geppetto core release cycle.

### C) JSON-Schema-only Pipeline First

Deferred:

- useful for UI/API interoperability but adds translation work for CLI layering.
- better as phase 2 once typed glazed-first pipeline is stable.

## Implementation Plan

1. Create `middlewarecfg` package in Geppetto with registry + typed definition helper.
2. Extend `profiles.MiddlewareUse` with optional `ID` and `Enabled`.
3. Add tests: registry duplicate handling, section generation, typed decode, build chain order.
4. Refactor `pinocchio/cmd/web-chat/runtime_composer.go` to use middlewarecfg registry flow.
5. Refactor `go-go-os/internal/pinoweb/runtime_composer.go` to use profile-provided middleware uses.
6. Remove dead router/server middleware registration API and state.
7. Add profile API tests for new middleware use fields and config validation.
8. Update docs/help pages for middleware definition registry and profile-scoped defaults.
9. Add end-to-end tests validating same middleware config behavior in pinocchio and go-go-os.

## Open Questions

1. Should middleware definition lookup happen during profile CRUD writes (strict) or only at runtime compose (lazy)?
2. Should unknown middleware names in profile runtime be a hard error or warning+skip in non-strict modes?
3. Should middleware definitions be registered via composer constructor options only, or via an explicit shared bootstrap registry module?
4. Do we want cross-app shared "builtin middleware definition packs" (for example geppetto core + pinocchio extras)?
5. Should middleware config keys eventually be migrated into typed-key extension payloads for fully namespaced versioning?

## References

- Imported proposal: `sources/local/middleware-config-proposals.md`
- Middleware runtime type: `geppetto/pkg/inference/middleware/middleware.go`
- Profile runtime payload model: `geppetto/pkg/profiles/types.go`
- Profile override parsing today: `geppetto/pkg/profiles/service.go`
- Current pinocchio runtime composer: `pinocchio/cmd/web-chat/runtime_composer.go`
- Current go-go-os runtime composer: `go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go`
- Existing registry patterns:
  - `geppetto/pkg/inference/tools/registry.go`
  - `geppetto/pkg/events/registry.go`
  - `pinocchio/pkg/sem/registry/registry.go`
