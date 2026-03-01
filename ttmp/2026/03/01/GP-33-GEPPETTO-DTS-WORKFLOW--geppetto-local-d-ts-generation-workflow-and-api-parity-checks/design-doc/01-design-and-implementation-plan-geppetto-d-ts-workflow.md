---
Title: 'Design and Implementation Plan: Geppetto d.ts Workflow'
Ticket: GP-33-GEPPETTO-DTS-WORKFLOW
Status: active
Topics:
    - geppetto
    - js-bindings
    - tooling
    - typescript
    - goja
    - codegen
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/cmd/gen-meta/main.go
      Note: Existing authoritative YAML/template d.ts generator
    - Path: /home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/spec/geppetto_codegen.yaml
      Note: Schema that drives geppetto.d.ts generation
    - Path: /home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/module.go
      Note: Runtime export surface to validate against generated declarations
    - Path: /home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/doc/types/geppetto.d.ts
      Note: Generated declaration target
ExternalSources: []
Summary: Add a geppetto-local gen-dts command with check mode and parity testing against runtime exports, while continuing to use the existing YAML/template generator as the source of truth.
LastUpdated: 2026-03-01T14:00:00-05:00
WhatFor: Define implementation for a reliable local/CI d.ts workflow in geppetto and verify declaration/runtime alignment.
WhenToUse: Use when updating geppetto JS API, schema templates, or declaration generation workflows.
---

# Design and Implementation Plan: Geppetto d.ts Workflow

## Executive Summary

Geppetto already has a bespoke TypeScript declaration generator (`cmd/gen-meta --section js-dts`) driven by `pkg/spec/geppetto_codegen.yaml` and `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`. The gap is developer workflow and validation: there is no repo-local `gen-dts` command with first-class `--check` behavior, and no automated parity check confirming declarations still match the runtime `require("geppetto")` export surface.

This ticket introduces a geppetto-local `cmd/gen-dts` command that wraps existing `gen-meta` generation, adds deterministic check mode, and introduces an API parity test that compares runtime exports with generated declaration exports. The design intentionally does not replace the existing YAML/template generator in this iteration.

## Problem Statement

Current issues:

1. External/integration repos cannot rely on `go-go-goja/cmd/gen-dts` to discover geppetto APIs, because geppetto does not register through `go-go-goja/modules.Register` and does not implement `TypeScriptDeclarer`.
2. Geppetto's own d.ts generation path exists, but is hidden behind `go generate` and not exposed as an explicit local command (`make gen-dts`, `make check-dts`).
3. There is no explicit test asserting that generated `pkg/doc/types/geppetto.d.ts` remains aligned with the runtime exports from `pkg/js/modules/geppetto/module.go`.

Impact:

- Developer confusion about which generator to run.
- Potential declaration drift during API changes.
- Harder CI enforcement for declaration correctness.

## Proposed Solution

Implement a geppetto-local declaration workflow that uses the existing generator as source-of-truth:

1. Add `cmd/gen-dts` in geppetto that invokes `cmd/gen-meta --section js-dts`.
2. Add `--check` mode to `cmd/gen-dts` by generating to a temporary output path (via temporary schema rewrite) and byte-comparing against committed `pkg/doc/types/geppetto.d.ts`.
3. Add a runtime parity test in `pkg/js/modules/geppetto` that:
   - loads `require("geppetto")` in a real runtime,
   - extracts runtime export keys (top-level + namespaced objects),
   - verifies matching declaration entries exist in generated `pkg/doc/types/geppetto.d.ts`.
4. Add Makefile targets:
   - `gen-dts` (write/update mode)
   - `check-dts` (drift detection)
5. Document usage in README (or nearest operator-facing doc section).

## Design Decisions

1. Keep `cmd/gen-meta` + YAML/template pipeline as authoritative for geppetto declarations.
Reason: this pipeline already encodes geppetto-specific types and has broad coverage in `geppetto.d.ts`.

2. Add repo-local wrapper command instead of forcing geppetto onto `go-go-goja/modules.Register` in this ticket.
Reason: geppetto module registration is runtime-option-driven (`Runner`, registries, hooks). A forced registration model change is a larger architecture migration and should be a separate ticket.

3. Validate declaration/runtime parity at export-surface level (not full TS AST equivalence).
Reason: we need robust drift detection without building a full TypeScript parser in Go for this iteration.

4. Use non-destructive check mode (temp schema/output) rather than mutating tracked files.
Reason: predictable CI behavior and no accidental working tree changes when running check mode.

## Alternatives Considered

1. Port geppetto fully to `go-go-goja/modules.Register` and `TypeScriptDeclarer` now.
Rejected for this ticket because geppetto's runtime module initialization model depends on runtime-owned options and is not a trivial registration swap.

2. Parse `.d.ts` into full AST and compare semantics against runtime reflection.
Rejected as over-engineered for immediate workflow needs.

3. Reuse `go-go-goja/cmd/gen-dts` directly without geppetto-local wrapper.
Rejected because current discovery mechanism cannot see geppetto module registration path.

## Implementation Plan

Phase 1: Workflow command

1. Add `geppetto/cmd/gen-dts/main.go` with:
   - `--schema` (default `pkg/spec/geppetto_codegen.yaml`)
   - `--check` flag
2. Write mode:
   - run `go run ./cmd/gen-meta --schema <schema> --section js-dts`
3. Check mode:
   - load schema,
   - rewrite `outputs.geppetto_dts` to temp path,
   - run `gen-meta` on temp schema,
   - compare temp output to committed output.

Phase 2: Parity validation

1. Add test file in `pkg/js/modules/geppetto`.
2. Use existing runtime helper setup to load `require("geppetto")`.
3. Assert declaration file contains exports for:
   - top-level constants/functions (`version`, `consts`, `createBuilder`, `createSession`, `runInference`)
   - namespaced export objects (`turns`, `engines`, `profiles`, `schemas`, `middlewares`, `tools`)
   - expected method names inside each namespace.

Phase 3: DX and CI hooks

1. Add `gen-dts` and `check-dts` targets to Makefile.
2. Add short README instructions for local and CI usage.
3. Run generation + tests and confirm no drift.

## Testing Strategy

1. Command smoke tests:
   - `go run ./cmd/gen-dts`
   - `go run ./cmd/gen-dts --check`
2. Runtime parity test:
   - `go test ./pkg/js/modules/geppetto -run TestGeneratedDTSMatchesRuntimeExports -count=1`
3. Full affected tests:
   - `go test ./cmd/gen-dts ./pkg/js/modules/geppetto -count=1`

## Open Questions

1. Should future migration expose a reusable library API from `go-go-goja/cmd/gen-dts` so geppetto can share code instead of wrapping `gen-meta` subprocess execution?
2. Should parity validation later include type-shape checks for signatures, not just export presence?

## References

1. `cmd/gen-meta/main.go`
2. `pkg/spec/geppetto_codegen.yaml`
3. `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
4. `pkg/doc/types/geppetto.d.ts`
5. `pkg/js/modules/geppetto/module.go`
