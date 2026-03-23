---
Title: 'Investigation and Design: Geppetto true replacement onto registration-based gen-dts'
Ticket: GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT
Status: active
Topics:
    - architecture
    - codegen
    - geppetto
    - goja
    - js-bindings
    - migration
    - tooling
    - typescript
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/dts_parity_test.go
      Note: Parity gate retained in migration strategy
    - Path: workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/module.go
      Note: Runtime registration constraints informing migration design
    - Path: workspaces/2026-03-01/generate-js-types/go-go-goja/cmd/gen-dts/main.go
      Note: Target of proposed option plumbing and generator flow changes
    - Path: workspaces/2026-03-01/generate-js-types/go-go-goja/modules/typing.go
      Note: Target of proposed option-aware declarer API
    - Path: workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/render/dts_renderer.go
      Note: Target of proposed renderer expansion
    - Path: workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/spec/types.go
      Note: Target of proposed declaration model expansion
    - Path: workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/validate/validate.go
      Note: Target of proposed validator expansion
ExternalSources: []
Summary: Detailed true-replacement design for migrating geppetto d.ts generation onto registration-based go-go-goja gen-dts with tsgen model extensions.
LastUpdated: 2026-03-01T14:47:03-05:00
WhatFor: Define the architecture, APIs, migration phases, and validation gates required to replace geppetto's bespoke d.ts generator.
WhenToUse: Use when implementing or reviewing the GP-34 true replacement effort.
---


# Investigation and Design: Geppetto true replacement onto registration-based gen-dts

## Executive Summary

This document proposes a full replacement of geppetto's schema/template `d.ts` generation with the registration-based `go-go-goja/cmd/gen-dts` pipeline, while preserving type richness and runtime parity guarantees.

The core design decision is to evolve `go-go-goja/pkg/tsgen` from a function-centric descriptor model (`Functions + RawDTS`) into a declaration-graph model that first-class represents interfaces, type aliases, constants/object namespaces, and callable signatures. Without this evolution, migration would degenerate into embedding large raw declaration strings in `RawDTS`, which is not a true replacement.

The migration is phased to minimize risk:

1. Add tsgen declaration model support and renderer/validator coverage.
2. Add option-aware declarer interfaces and command option plumbing.
3. Implement geppetto declaration descriptor provider from runtime metadata + declaration schema builders.
4. Run dual generation and parity checks until outputs converge.
5. Cut over geppetto generation command to registration-based path and retire `cmd/gen-meta --section js-dts` usage for geppetto declarations.

## Problem Statement and Scope

### Problem

Geppetto currently produces `pkg/doc/types/geppetto.d.ts` through a custom schema + template pipeline (`cmd/gen-meta`). The broader ecosystem now has a registration-based generator (`go-go-goja/cmd/gen-dts`) intended to collect declaration descriptors from modules. These systems are not aligned structurally.

Evidence:

- Geppetto declaration source is schema/template-driven (`geppetto_codegen.yaml`, `geppetto.d.ts.tmpl`, `cmd/gen-meta/main.go:412-423`).
- Registration generator expects modules implementing `TypeScriptDeclarer` and emits from `spec.Module` descriptors (`go-go-goja/cmd/gen-dts/main.go:103-173`, `modules/typing.go:5-9`).
- `spec.Module` currently models only `Functions` and `RawDTS` (`pkg/tsgen/spec/types.go:26-32`).

### Scope

In scope:

1. True replacement architecture and migration plan.
2. Required changes to tsgen model, renderer, validator, and command options.
3. Required changes to geppetto declaration authoring path.
4. Validation and rollout strategy.

Out of scope:

1. Immediate implementation in this ticket (this is investigation/design only).
2. Frontend consumer migration details outside declaration generation contracts.
3. General non-geppetto module refactors unless needed for compatibility.

## Current-State Analysis (Evidence-Based)

### A. Geppetto generation pipeline

1. `cmd/gen-meta` owns multi-section generation including `js-dts`.
- `--section js-dts` is part of command contract (`cmd/gen-meta/main.go:139-185`).

2. Schema is explicit about both output and template paths.
- `outputs.geppetto_dts` and `templates.geppetto_dts` (`geppetto_codegen.yaml:5-15`).

3. `generateJSDTS` renders template using computed enum groups.
- `generateJSDTS` path (`cmd/gen-meta/main.go:412-423`).

4. Template contains rich declaration categories:
- 41 exported interfaces, 3 type aliases, grouped const object exports, callback signatures, and promise-bearing APIs (measured by experiments under `sources/experiments/`).

### B. Geppetto runtime export surface assembly

1. Module registration is option-driven (`Register(reg, opts)`) and not global-init style (`module.go:49-55`).
2. Export surface is assembled in imperative code (`installExports`) with nested namespaces (`module.go:137-190`).

### C. go-go-goja registration tsgen path

1. Generator discovers modules through global default registry (`main.go:43`, `modules/common.go:95-111`).
2. Descriptor interface has no options (`modules/typing.go:5-9`).
3. Descriptor model lacks first-class declarations beyond function signatures and raw lines (`spec/types.go:26-32`).
4. Renderer supports `export function` plus `RawDTS` passthrough (`dts_renderer.go:63-81`).
5. Validator validates function-centric descriptor data (`validate.go:34-59`).

## Gap Analysis

### Gap 1: Declaration model mismatch

- Geppetto declaration surface: broad declaration graph.
- tsgen model: function declarations + raw text escape hatch.

Consequence:
- True replacement is blocked without model expansion.

### Gap 2: Option-driven API surfaces are not representable in declarer contract

- Geppetto runtime behavior is option-influenced (`Options` in module registration).
- `TypeScriptDeclarer` currently cannot receive generation options.

Consequence:
- Cannot cleanly represent variant API surfaces, feature toggles, or compatibility modes through standard interface.

### Gap 3: Validation asymmetry

- Current validator checks function declarations and type refs.
- No structural validation for interfaces, type aliases, namespaces, constant groups.

Consequence:
- Regression risk increases as declaration richness grows.

### Gap 4: Registration model mismatch

- go-go-goja relies on package `init()` registration for module discovery.
- Geppetto registration is explicit runtime call with options.

Consequence:
- Need a generator-focused module registration adapter that does not alter runtime semantics.

## Design Goals and Non-Goals

### Goals

1. First-class declaration modeling in tsgen for all geppetto declaration categories.
2. Deterministic generation and check mode.
3. Option-aware descriptor generation.
4. Runtime parity safety net preserved.
5. Migration path with dual-run confidence before cutover.

### Non-goals

1. Removing all raw declaration support (`RawDTS`) globally; keep as escape hatch, not primary path.
2. Forcing geppetto runtime registration API to become init/global-driven.

## Proposed Solution

## 1) Extend tsgen declaration model

Introduce richer `spec.Module` structure.

Proposed additions (conceptual):

```go
// pkg/tsgen/spec/types.go

type Module struct {
    Name        string
    Description string

    // Existing
    Functions   []Function
    RawDTS      []string

    // New
    Interfaces  []InterfaceDecl
    TypeAliases []TypeAliasDecl
    Consts      []ConstDecl
    Namespaces  []NamespaceDecl // optional, for grouped values/functions/types
}

type InterfaceDecl struct {
    Name    string
    Extends []TypeRef
    Fields  []Field
    Calls   []CallSignature
}

type TypeAliasDecl struct {
    Name string
    Type TypeRef
}

type ConstDecl struct {
    Name     string
    Type     TypeRef
    Readonly bool
}

type NamespaceDecl struct {
    Name      string
    Functions []Function
    Consts    []ConstDecl
    Types     []TypeAliasDecl
}
```

Rationale:

- Enables declaration-graph generation without dropping to raw strings.
- Keeps function model intact for existing modules.
- Enables incremental adoption in other modules.

## 2) Extend renderer

`pkg/tsgen/render/dts_renderer.go` should render declaration categories in deterministic order.

Proposed order inside `declare module`:

1. consts
2. interfaces
3. type aliases
4. namespaces
5. functions
6. raw passthrough (last resort)

Determinism rules:

1. Sort by declaration name.
2. Stable field ordering.
3. Preserve explicit insertion order only where semantically required (if any).

## 3) Extend validator

`pkg/tsgen/validate/validate.go` should validate new declaration categories:

1. duplicate name collisions across declaration namespaces,
2. empty identifiers,
3. invalid call signatures,
4. unresolved named type references (best-effort static checks where possible),
5. namespace/member collision checks.

## 4) Add option-aware declarer contract

Retain compatibility by adding optional interface:

```go
type TypeScriptDeclarer interface {
    TypeScriptModule() *spec.Module
}

type TypeScriptDeclarerWithOptions interface {
    TypeScriptModuleWithOptions(opts map[string]any) (*spec.Module, error)
}
```

`cmd/gen-dts` changes:

1. add repeatable flags (example):
- `--module-option geppetto.profile=full`
- `--module-option geppetto.feature.profiles=true`

2. parse flags into `map[moduleName]map[string]any`.
3. resolution logic:
- if module implements `TypeScriptDeclarerWithOptions`, pass module-specific option map.
- else use legacy `TypeScriptModule()`.

## 5) Introduce geppetto descriptor provider (generation adapter)

Create a generator-facing module package in geppetto that:

1. Implements `modules.NativeModule` for registry identity,
2. Implements `TypeScriptDeclarerWithOptions` for declaration generation,
3. Does not replace runtime `Register(reg, opts)` semantics used in production execution.

Recommended package (example):
- `geppetto/pkg/js/modules/geppetto/tsdescriptor`

Responsibilities:

1. Derive declaration model from canonical geppetto declaration schema/builders,
2. Keep declaration source as structured Go data, not monolithic raw string,
3. Include option switches for slim/full profiles if needed.

## 6) Dual-generation and convergence gates

During migration:

1. Generate declaration with old path and new path.
2. Compare normalized outputs.
3. Keep runtime parity test as non-negotiable gate.

Normalization strategy:

1. run prettier/formatter for stable whitespace, or
2. compare parsed declaration AST fingerprints (preferred medium-term).

## Pseudocode and Key Flows

### A. `go-go-goja/cmd/gen-dts` module descriptor resolution flow

```go
for _, module := range modules.ListDefaultModules() {
    if !selected(module.Name()) { continue }

    opts := optionsByModule[module.Name()]

    switch m := module.(type) {
    case modules.TypeScriptDeclarerWithOptions:
        desc, err := m.TypeScriptModuleWithOptions(opts)
    case modules.TypeScriptDeclarer:
        desc := m.TypeScriptModule()
    default:
        if strict { return error }
        continue
    }

    validate.Module(desc)
    bundle.Modules = append(bundle.Modules, desc)
}

validate.Bundle(bundle)
render.Bundle(bundle)
```

### B. Geppetto migration generation command flow

```go
// geppetto cmd/gen-dts (post replacement)
opts := parseFlags()
out, err := exec.Command("go", "run", "../go-go-goja/cmd/gen-dts", ...module opts...)
if check {
    compare(out, committedFile)
} else {
    write(out)
}
```

### C. Cutover verification flow

```go
old := generateWithGenMetaJSDTS()
new := generateWithRegistrationTSGen()
if normalize(old) != normalize(new) {
    fail("cutover mismatch")
}
runRuntimeParityTest() // d.ts export groups vs runtime keys
```

## Phased Implementation Plan

### Phase 0: Foundation and ADR updates

1. Add GP-34 references and acceptance criteria to docs.
2. Lock migration constraints and invariants.

Deliverables:

- finalized design doc,
- agreed declaration model extension spec.

### Phase 1: tsgen model expansion

Changes:

1. `pkg/tsgen/spec` model additions,
2. helper constructors,
3. backward-compatible compatibility tests.

Acceptance:

1. existing modules still generate unchanged outputs,
2. new declaration unit tests pass.

### Phase 2: renderer and validator expansion

Changes:

1. render paths for interfaces/type aliases/consts/namespaces,
2. validator coverage for new declarations.

Acceptance:

1. deterministic output snapshot tests,
2. invalid descriptor tests reject malformed declarations.

### Phase 3: option-aware declarer interface

Changes:

1. add new interface in `modules/typing.go`,
2. `cmd/gen-dts` option parsing and module dispatch.

Acceptance:

1. legacy modules unaffected,
2. new options path covered by tests.

### Phase 4: geppetto descriptor provider

Changes:

1. geppetto declaration builder using new spec model,
2. registration adapter for generator discovery,
3. conversion of current declaration artifacts into structured descriptors.

Acceptance:

1. generated output semantically equivalent to current geppetto.d.ts,
2. dual-generation compare gate passes.

### Phase 5: cutover and cleanup

Changes:

1. switch geppetto `gen-dts` implementation to registration-based path,
2. deprecate `cmd/gen-meta --section js-dts` for geppetto declarations,
3. preserve historical templates until one release cycle validates stability.

Acceptance:

1. CI uses new generation path,
2. runtime parity tests pass,
3. no downstream TypeScript regressions.

## Testing and Validation Strategy

### Unit tests

1. spec model constructors and serialization.
2. renderer snapshot tests for new declaration categories.
3. validator negative tests for malformed declarations.
4. option parsing tests for `cmd/gen-dts`.

### Integration tests

1. module discovery + generation across mixed old/new declarers.
2. geppetto declaration generation end-to-end.

### Parity tests

1. existing geppetto runtime export parity test remains required.
2. add normalized old-vs-new d.ts comparison test until cutover completion.

### CI gates

1. `--check` on generated d.ts.
2. full go test suite including module parity.
3. optional TypeScript compile smoke tests in downstream consumers.

## Risks, Alternatives, and Open Questions

### Risks

1. Scope expansion in tsgen can impact existing users if not backward compatible.
2. Declaration equivalence is hard to assert with pure text diff due to formatting/noise.
3. Option-driven descriptors can create non-deterministic outputs if option defaults drift.

Mitigations:

1. feature-flag new declaration model with compatibility tests,
2. use normalized or AST-based comparison for cutover,
3. require explicit option inputs in CI and command wrappers.

### Alternatives considered

1. Use `RawDTS` only for geppetto and call it migrated.
- Rejected for true replacement objective; keeps bespoke declaration content, only changes transport.

2. Keep current geppetto bespoke generator permanently.
- Rejected because it prevents unified registration-based tooling strategy and shared maintenance.

3. Auto-derive declaration model purely from runtime export reflection.
- Partially useful for parity, but insufficient for rich static type shapes unless runtime metadata is greatly expanded.

### Open questions

1. Do we require AST-level equivalence between old/new declarations or semantic compile equivalence is sufficient?
2. Should geppetto maintain one canonical schema source for both runtime and declaration generation, or move canonical source entirely to Go descriptor builders?
3. How many option profiles should be supported in v1 of option-aware declarers?

## Implementation Readiness Checklist

1. tsgen model RFC approved.
2. option-aware declarer contract approved.
3. geppetto descriptor package location and ownership confirmed.
4. dual-generation convergence pipeline defined.
5. rollback strategy documented.

## References

- Geppetto schema:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/spec/geppetto_codegen.yaml`
- Geppetto generator implementation:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/cmd/gen-meta/main.go`
- Geppetto declaration template:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
- Geppetto runtime export registration:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/module.go`
- Current geppetto local wrapper command:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/cmd/gen-dts/main.go`
- Runtime parity test:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/dts_parity_test.go`
- Registration generator:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/cmd/gen-dts/main.go`
- Registry/declarer contracts:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/modules/common.go`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/modules/typing.go`
- tsgen model/renderer/validator:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/spec/types.go`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/render/dts_renderer.go`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/validate/validate.go`
- GP-34 experiment artifacts:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts/sources/experiments/`
