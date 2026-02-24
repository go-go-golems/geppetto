---
Title: Implementation Plan - Middleware JSON Schema and ParseStep Resolver
Ticket: GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP
Status: active
Topics:
    - architecture
    - backend
    - geppetto
    - pinocchio
    - middleware
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/middleware/middleware.go
      Note: Middleware runtime type to preserve.
    - Path: geppetto/pkg/sections/sections.go
      Note: Existing layered precedence semantics used as compatibility baseline.
    - Path: geppetto/pkg/profiles/types.go
      Note: Profile `RuntimeSpec.Middlewares` source model.
    - Path: geppetto/pkg/profiles/validation.go
      Note: Validation surface for middleware uses and schema decode failures.
    - Path: geppetto/pkg/profiles/step_settings_mapper.go
      Note: Legacy symbol likely replaced by resolver naming.
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Existing ad-hoc parse path that must be removed.
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go
      Note: Secondary runtime composer to migrate to shared resolver.
ExternalSources: []
Summary: Detailed implementation plan for a JSON-schema-first middleware config resolver with source provenance equivalent to Glazed ParseStep logs.
LastUpdated: 2026-02-24T13:12:02-05:00
WhatFor: Define the core middleware configuration engine consumed by profile runtime composition across applications.
WhenToUse: Use when implementing middleware contract definitions, layered value resolution, and debug provenance APIs.
---

# Implementation Plan - Middleware JSON Schema and ParseStep Resolver

## Executive Summary

This ticket performs a hard cutover to one middleware configuration architecture:

1. every middleware declares JSON Schema as its config contract,
2. one resolver applies layered sources with deterministic precedence,
3. resolver records per-path provenance steps (source/layer/raw/coerced metadata),
4. runtime composers build middleware chain from resolved typed config only.

No compatibility mode is kept for legacy ad-hoc parsing. One model reduces ambiguity and long-term maintenance cost.

## Problem Statement

Middleware configuration currently exists as partially structured `any` blobs with application-local decoding logic. This creates drift across binaries:

- Pinocchio and Go-Go-OS parse middleware overrides differently,
- frontend/CLI cannot rely on one machine-readable schema,
- debugging why a value was chosen is difficult without parse provenance,
- profile-scoped defaults remain fragile when layered with request/config overrides.

The project already depends on layered-value semantics (defaults + profile + config + env + flags + request). We need those semantics in a middleware-specific resolver that is schema-native and introspectable.

## Requirements

Functional:

- middleware definitions expose canonical JSON Schema,
- resolver accepts layered source payloads and returns final config object per middleware instance,
- resolver produces provenance log equivalent to Glazed ParseStep history,
- runtime builder consumes only resolver output.

Non-functional:

- deterministic output ordering,
- clear validation errors with path/source context,
- no runtime regressions in middleware chain order,
- extensible to UI form generation and debug APIs.

## Proposed Solution

### 1. Middleware Definition Contract

Create `geppetto/pkg/inference/middlewarecfg` with definition interface:

```go
type Definition interface {
    Name() string
    Description() string
    ConfigJSONSchema() map[string]any
    Build(ctx context.Context, deps BuildDeps, cfg any) (middleware.Middleware, error)
}
```

`BuildDeps` carries app-owned dependencies (DB handles, services, publishers, etc.) to avoid global state.

### 2. Instance Model

Middleware uses remain profile-owned, with stable instance semantics:

```go
type Use struct {
    Name    string `json:"name" yaml:"name"`
    ID      string `json:"id,omitempty" yaml:"id,omitempty"`
    Enabled *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
    Config  any    `json:"config,omitempty" yaml:"config,omitempty"`
}
```

`ID` prevents collisions when the same middleware appears multiple times with different configs.

### 3. Resolver and Provenance

The resolver is path-based and schema-driven.

```go
type ParseStep struct {
    Source   string         `json:"source"`
    Layer    string         `json:"layer,omitempty"`
    Path     string         `json:"path"`        // JSON pointer
    RawValue any            `json:"raw_value"`
    Value    any            `json:"value"`       // coerced
    Metadata map[string]any `json:"metadata,omitempty"`
}

type ResolvedPath struct {
    Path  string      `json:"path"`
    Value any         `json:"value"`
    Log   []ParseStep `json:"log"`
}

type ResolvedInstance struct {
    Use      Use                      `json:"use"`
    Config   map[string]any           `json:"config"`
    Trace    map[string]*ResolvedPath `json:"trace"`
}
```

Resolver algorithm:

1. compile schema for each active instance,
2. apply sources in fixed precedence order,
3. coerce/validate at write time by path,
4. append `ParseStep`,
5. validate final object,
6. return resolved config plus trace.

Pseudo:

```pseudo
for instance in active_uses:
  schema = registry.schema(instance.name)
  state = newResolvedState(schema)
  for source in precedence:
    writes = project(source.payload(instance), schema)
    for w in writes:
      state.apply(path=w.path, value=w.coerced)
      state.log(path=w.path, step=ParseStep{...})
  state.validateFinal()
  emit ResolvedInstance
```

### 4. Source Precedence

Canonical order (low -> high):

1. schema defaults,
2. profile middleware config,
3. persisted app config file,
4. environment values,
5. command flags,
6. request runtime overrides.

This matches established Glazed mental model while being schema-first.

### 5. Runtime Build

`BuildChain` consumes resolved instances:

```go
func (r *Registry) BuildChain(
    ctx context.Context,
    deps BuildDeps,
    resolved []ResolvedInstance,
) ([]middleware.Middleware, error)
```

Only enabled instances are built. Build errors include middleware name + instance ID + path-level root cause.

### 6. Glazed Adapter (Optional Output, Not Canonical Input)

Provide adapter:

```go
func SectionFromSchema(instanceKey string, schema map[string]any) (*schema.Section, error)
```

This preserves CLI/help surfaces without making sections canonical.

## Design Decisions

1. JSON Schema is the only canonical config contract for middleware parameters.
2. ParseStep-style provenance is mandatory from day one, not deferred.
3. Runtime middleware function type remains unchanged.
4. Resolver is path-based to support nested objects/arrays and clear debugging.
5. Hard cutover removes dual parsing paths and environment toggles.

## Alternatives Considered

### A. Keep ad-hoc map decoding in each runtime composer

Rejected because it guarantees long-term drift and no shared introspection.

### B. Keep Glazed sections as canonical and export JSON Schema secondarily

Rejected because it duplicates contract sources and invites mismatch.

### C. JSON Schema canonical but no provenance ledger

Rejected because it regresses debugability and violates existing ParseStep expectations.

## Implementation Plan

### Phase A - Package Scaffold

1. Add `middlewarecfg` package with definition registry and instance model.
2. Add registration APIs and duplicate name guards.
3. Add middleware-use validation (`name`, optional `id`, unique instance key).

### Phase B - Resolver Core

1. Add schema projection/coercion helpers.
2. Add layered source interface and canonical precedence implementation.
3. Add path-based state store and parse-step log capture.
4. Add final object materialization and validation.

### Phase C - Runtime Integration

1. Add `BuildChain` to instantiate middlewares from resolved configs.
2. Integrate into Pinocchio runtime composer.
3. Integrate into Go-Go-OS runtime composer.
4. Delete ad-hoc middleware override parsing logic and legacy map builder path.

### Phase D - Introspection and Debug

1. Add resolver debug object serialization (for tests and optional debug endpoint).
2. Add path/source metadata in error responses/log events.
3. Add deterministic formatting for trace output in assertions.

### Phase E - Verification

1. Unit tests for precedence and coercion.
2. Unit tests for trace log order and metadata.
3. Integration tests for middleware chain ordering and runtime behavior parity across apps.
4. Regression tests for invalid payload and conflict scenarios.

## Open Questions

1. Which schema validator/coercer package is preferred in this repository baseline?
2. Should parse trace be exposed only in debug endpoints or in standard API responses under an opt-in flag?
3. Should array writes be replace-only in phase 1 or support path-level merge operators?

## References

- `geppetto/pkg/inference/middleware/middleware.go`
- `geppetto/pkg/sections/sections.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`
- `go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go`
