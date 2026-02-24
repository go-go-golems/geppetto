---
Title: JSON Schema First with Glazed-Style Value Provenance
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
    - Path: /home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/fields/field-value.go
      Note: Current Glazed value/log model (`FieldValue.Log []ParseStep`) to preserve in new model.
    - Path: /home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/fields/parse.go
      Note: ParseStep contract (`Source`, `Value`, `Metadata`) and default source semantics.
    - Path: /home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/sources/update.go
      Note: Layer merge behavior and `UpdateWithLog` flow used as provenance baseline.
    - Path: /home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/values/section-values.go
      Note: Existing section/value container and decode behavior.
    - Path: /home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/json-schema.go
      Note: Current JSON schema export path and its limitations.
    - Path: geppetto/ttmp/2026/02/24/GP-20-PROFILE-REGISTRY-EXTENSIONS--extensible-typed-key-metadata-for-profile-registry/design-doc/02-middleware-configuration-registry-unification-with-profile-scoped-defaults.md
      Note: Parent design that this schema/provenance model extends.
ExternalSources: []
Summary: Hard-cutover proposal for making JSON Schema canonical immediately and implementing ParseStep-equivalent provenance tracking from day one.
LastUpdated: 2026-02-25T00:42:00-05:00
WhatFor: Define a concrete architecture for JSON-schema-first middleware configuration with source-layer traceability equivalent to current Glazed parse logs.
WhenToUse: Use when implementing middleware config resolution, UI form generation, and audit/debug tooling that must explain where each resolved value came from.
---

# JSON Schema First with Glazed-Style Value Provenance

## Executive Summary

Yes, we can go straight to JSON Schema and still keep the same auditability behavior Glazed provides today.  
The key is to treat provenance tracking as a first-class layer independent of Glazed sections, not as a side effect of Glazed-only parsing.

Recommended direction:

1. JSON Schema becomes canonical for middleware parameter contracts.
2. A new resolver engine applies layered sources (defaults/profile/env/flags/request) against schema paths.
3. Every write is recorded in a provenance ledger equivalent to `FieldValue.Log` (but path-based and schema-native).
4. Glazed sections become an adapter (for CLI help/Cobra ergonomics), not the canonical contract source.
5. Hard cutover: no dual-path compatibility mode with “Glazed primary” middleware configs.

## Problem Statement

Current state has two separate concerns coupled together:

- parameter shape + coercion + validation (mostly Glazed field definitions),
- provenance tracking (`FieldValue.Log []ParseStep` in Glazed).

If we move to JSON Schema without designing a provenance model, we lose one of the strongest operational features: "why is this final value what it is?".

Today’s baseline behavior in Glazed:

- `fields.FieldValue` carries the final typed value plus an ordered `Log` of parse steps.
- each `ParseStep` stores `Source`, `Value`, and optional `Metadata`.
- source middlewares (`FromMap`, `FromEnv`, etc.) call `UpdateWithLog(...)`, preserving history.

This must be preserved, not regressed, in the JSON-schema-first system.

## Proposed Solution

### 1. Canonical Parameter Model = JSON Schema (Draft 2020-12)

Each middleware definition exposes:

```go
type Definition interface {
    Name() string
    Description() string
    ConfigJSONSchema() *jsonschema.Schema
    BuildFromAny(ctx context.Context, deps BuildDeps, cfg any) (middleware.Middleware, error)
}
```

Schema contains defaults, required fields, enums, descriptions, and constraints.

### 2. Provenance Ledger (Glazed Log Equivalent)

Introduce path-based value tracking:

```go
type ValueSetStep struct {
    Source     string                 `json:"source"`      // defaults/profile/env/cobra/request
    Layer      string                 `json:"layer"`       // optional finer grain
    Path       string                 `json:"path"`        // JSON pointer, e.g. /middlewares/systemPrompt/prompt
    RawValue   any                    `json:"raw_value"`   // incoming value
    Coerced    any                    `json:"coerced"`     // value after schema coercion
    Metadata   map[string]any         `json:"metadata,omitempty"`
}

type ResolvedValue struct {
    Path  string         `json:"path"`
    Value any            `json:"value"`
    Steps []ValueSetStep `json:"steps"`
}

type ResolvedConfig struct {
    Values map[string]any            `json:"values"` // materialized object
    Trace  map[string]*ResolvedValue `json:"trace"`  // path -> resolution history
}
```

This is structurally the same concept as `FieldValue.Log`, but schema/path-oriented.

### 3. Source Application Engine

Resolver algorithm:

1. compile schema for active middleware instances,
2. ingest each source in precedence order,
3. validate/coerce source payload against schema fragments,
4. apply writes by JSON pointer path,
5. append a `ValueSetStep` per write,
6. produce final typed object and trace.

Pseudocode:

```pseudo
resolved = new ResolvedConfig(schema)

for source in [defaults, profile, config, env, flags, request]:
  payload = source.load()
  writes = schemaProjectAndCoerce(schema, payload) // -> [{path, raw, coerced, metadata}]
  for w in writes:
    resolved.values[w.path] = w.coerced
    resolved.trace[w.path].steps.append({
      source: source.name,
      layer: source.layer,
      path: w.path,
      raw_value: w.raw,
      coerced: w.coerced,
      metadata: w.metadata
    })

cfg = materializeObjectFromPaths(resolved.values)
validateWholeObject(schema, cfg)
return resolved
```

### 4. Glazed Adapter Layer (Optional but Recommended)

For CLI compatibility and good help UX, generate Glazed sections from JSON Schema:

```go
func GlazedSectionFromJSONSchema(instanceKey string, sch *jsonschema.Schema) (schema.Section, error)
```

This adapter is one-way derived output. Canonical source remains JSON Schema.

### 4.1 Hard Cutover Scope

Cutover means:

- remove ad-hoc middleware decode/merge paths,
- do not keep a fallback "legacy glazed-only middleware schema" code path,
- migrate pinocchio and go-go-os composers in the same rollout window,
- keep one runtime behavior and one test matrix.

### 5. Unified Contract for APIs + UI + Runtime

Because JSON Schema is canonical:

- profile CRUD can validate config immediately,
- UI can render forms directly from schema,
- runtime resolver uses same schema for coercion,
- provenance trace can be returned in debug endpoints.

No duplicated contract definitions.

## Design Decisions

### Decision 1: JSON Schema Is Canonical, Not Export-Only

Rationale:

- one contract across backend, UI, and runtime;
- removes drift between section definitions and API models.

### Decision 2: Provenance Is Independent of Glazed Implementation Types

Rationale:

- preserves observability even if Glazed internals evolve;
- allows path-based tracking for nested objects/arrays naturally.

### Decision 3: Keep Source Semantics from Glazed

Rationale:

- team already understands layered precedence;
- existing mental model and test expectations remain valid.

### Decision 4: Glazed Is an Adapter, Not the Source of Truth

Rationale:

- keeps Cobra/help ergonomics where needed,
- prevents dual-authoring schema definitions.

### Decision 5: ParseStep-Equivalent Provenance Is Mandatory in Phase 1

Rationale:

- preserves current debugging power from day one of cutover,
- avoids temporary observability regressions.

### Decision 6: Hard Cutover Instead of Transitional Dual Model

Rationale:

- lower long-term complexity,
- no ambiguity about which contract is authoritative,
- fewer edge cases and fewer compatibility tests.

## Alternatives Considered

### A) Keep Glazed Canonical and Continue Exporting JSON Schema

Pros:

- lower near-term implementation cost.

Cons:

- JSON schema remains secondary and may drift,
- harder to guarantee UI/API/runtime contract parity.

### B) JSON Schema Canonical But Drop Provenance

Rejected:

- loses critical debugging and compliance capability,
- regression versus current `FieldValue.Log` behavior.

### C) Dual Canonical Models (Glazed + JSON Schema)

Rejected:

- high maintenance overhead,
- persistent mismatch risk.

## Implementation Plan

1. Add `middlewarecfg/schema` package with JSON-schema-first middleware contracts.
2. Add `middlewarecfg/resolve` package implementing layered source resolution + provenance ledger (ParseStep-equivalent).
3. Implement source adapters:
   - defaults from schema,
   - profile runtime config,
   - request overrides,
   - optional env/flags for CLI mode.
4. Refactor pinocchio/go-go-os runtime composers to consume resolver output.
5. Remove legacy ad-hoc middleware decode/override parsing paths.
6. Expose debug API to return resolved middleware config with trace (`path -> steps`).
7. Add Glazed adapter generator (`JSON schema -> section`) for CLI/help surfaces.
8. Add tests:
   - precedence,
   - coercion,
   - required/default semantics,
   - trace completeness and ordering.
9. Remove duplicated/legacy ad-hoc parser code.

## Open Questions

1. Which JSON schema validator/coercer should be canonical (`santhosh-tekuri/jsonschema`, `xeipuuv/gojsonschema`, or custom + `invopop/jsonschema` generation only)?
2. Should trace include timestamps and actor IDs for compliance trails?
3. How should array patch semantics be represented in provenance (replace vs merge)?
4. Do we expose full trace in production API responses or only behind debug/admin endpoints?
5. Should we persist traces for runtime replay/debug, or keep in-memory only?

## References

- Glazed log model: `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/fields/field-value.go`
- Parse step structure: `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/fields/parse.go`
- Source merge behavior: `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/sources/update.go`
- Values container: `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/values/section-values.go`
- Existing JSON schema export implementation: `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/json-schema.go`
- Parent middleware unification design: `design-doc/02-middleware-configuration-registry-unification-with-profile-scoped-defaults.md`
