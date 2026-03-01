---
Title: Investigation Evidence and Experiments
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: workspaces/2026-03-01/generate-js-types/geppetto/cmd/gen-meta/main.go
      Note: Current bespoke schema-template d.ts generator evidence
    - Path: workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/module.go
      Note: Runtime export assembly evidence
    - Path: workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl
      Note: Template complexity evidence for declaration richness
    - Path: workspaces/2026-03-01/generate-js-types/geppetto/pkg/spec/geppetto_codegen.yaml
      Note: Schema source for geppetto d.ts output/template mapping
    - Path: workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts/scripts/03_run_gap_experiments.sh
      Note: Experiment runner for reproducible gap reports
    - Path: workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts/sources/experiments/01-dts-surface-report.md
      Note: Measured declaration surface complexity output
    - Path: workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts/sources/experiments/02-tsgen-capability-report.md
      Note: Measured tsgen model capability output
    - Path: workspaces/2026-03-01/generate-js-types/go-go-goja/cmd/gen-dts/main.go
      Note: Registration-based generator behavior evidence
    - Path: workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/spec/types.go
      Note: Descriptor model limits evidence
ExternalSources: []
Summary: Evidence-backed investigation of current geppetto d.ts generation and go-go-goja registration tsgen limits for true replacement.
LastUpdated: 2026-03-01T14:47:03-05:00
WhatFor: Establish concrete constraints and migration requirements before replacing geppetto's bespoke d.ts generator.
WhenToUse: Use when planning or reviewing the true replacement of geppetto d.ts generation onto registration-based tsgen.
---


# Investigation Evidence and Experiments

## Goal

Build an evidence-backed baseline for replacing geppetto's bespoke `cmd/gen-meta --section js-dts` pipeline with the registration-based `go-go-goja/cmd/gen-dts` system, and identify concrete technical gaps that must be closed for a true replacement.

## Context

Two generation systems currently coexist:

1. Geppetto uses a schema + template generator that emits rich declarations into `pkg/doc/types/geppetto.d.ts`.
2. Go-go-goja uses registry-discovered modules implementing `TypeScriptDeclarer`, rendered through `pkg/tsgen` descriptors.

The key question is not whether we can produce some `.d.ts`, but whether we can replace the geppetto system without relying on raw passthrough fallback as the primary authoring mechanism.

## Quick Reference

### High-level finding

A true replacement requires extending `go-go-goja/pkg/tsgen/spec.Module` beyond function + raw-text oriented descriptors. Geppetto's declaration surface is significantly richer (interfaces, type aliases, grouped const objects, callback-rich signatures, nested object schemas).

### Evidence index

- Geppetto schema-driven generator:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/cmd/gen-meta/main.go`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/spec/geppetto_codegen.yaml`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
- Geppetto runtime exports:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/js/modules/geppetto/module.go`
- Registration-based generator:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/cmd/gen-dts/main.go`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/modules/common.go`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/modules/typing.go`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/spec/types.go`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/render/dts_renderer.go`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/validate/validate.go`
- Experiment scripts and outputs:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts/scripts/`
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts/sources/experiments/`

## Current-State Architecture (Evidence)

### Geppetto d.ts pipeline (bespoke)

1. YAML schema defines outputs/templates plus enum/key families and JS export groups.
- `outputs.geppetto_dts` and `templates.geppetto_dts` are explicit schema keys (`geppetto_codegen.yaml:5-15`).
- JS-facing enum/group metadata is modeled in schema (`geppetto_codegen.yaml:170-220`).

2. `cmd/gen-meta` parses schema and validates structure/consistency.
- Schema model includes both outputs and templates (`cmd/gen-meta/main.go:17-40`).
- `--section js-dts` is an explicit generation mode (`cmd/gen-meta/main.go:139-185`).
- Validation enforces required outputs/templates and export source references (`cmd/gen-meta/main.go:201-352`).

3. `generateJSDTS` renders template file with enum render data.
- d.ts generation is template-based (`cmd/gen-meta/main.go:412-423`).

4. Generated file is large and structurally rich.
- `geppetto.d.ts` currently ~497 lines (`wc -l` result from investigation command).
- Template includes many interfaces/types/functions and grouped constants (`geppetto.d.ts.tmpl:1-422`).

### Geppetto runtime export assembly

1. Registration function is option-driven and runtime-specific.
- `Register(reg, opts)` takes `Options` carrying middleware/tool/profile runtime dependencies (`module.go:32-55`).

2. Export surface is assembled imperatively in `installExports`.
- Top-level exports and grouped namespaces (`turns`, `engines`, `profiles`, `schemas`, `middlewares`, `tools`) are assigned via `mustSet` calls (`module.go:137-190`).

Implication:
- Geppetto export surface is runtime code first, with declaration generation currently done by schema/template logic and parity validation tests.

### Go-go-goja registration tsgen pipeline

1. `cmd/gen-dts` discovers modules via global registry.
- Reads `modules.ListDefaultModules()` (`go-go-goja/cmd/gen-dts/main.go:43`).
- Optional module filtering via `--module` (`go-go-goja/cmd/gen-dts/main.go:58-62,82-101`).

2. Module descriptors are optional and function-oriented.
- Module must implement `TypeScriptDeclarer` to contribute descriptor (`go-go-goja/cmd/gen-dts/main.go:128-136`, `modules/typing.go:5-9`).

3. Descriptor model is constrained.
- `spec.Module` fields are `Name`, `Description`, `Functions`, `RawDTS` (`pkg/tsgen/spec/types.go:26-32`).
- No first-class interface/type-alias/const group collections.

4. Renderer directly emits function declarations plus raw lines.
- Function rendering path: `renderFunction` (`pkg/tsgen/render/dts_renderer.go:63-108`).
- Raw passthrough path: `module.RawDTS` (`pkg/tsgen/render/dts_renderer.go:73-81`).

5. Validator validates function signatures and TypeRefs, not higher declaration categories.
- `validate.Module` iterates `module.Functions` (`pkg/tsgen/validate/validate.go:34-59`).

## Experiment Methodology

### Script inventory

- `scripts/01_probe_dts_surface.py`
  - Measures top-level export category counts and grouped-object members in `geppetto.d.ts`.
- `scripts/02_probe_tsgen_capabilities.py`
  - Inspects `tsgen` model fields and renderer/validator switch coverage.
- `scripts/03_run_gap_experiments.sh`
  - Runs both probes and materializes reports under `sources/experiments/`.

### Run command

```bash
/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts/scripts/03_run_gap_experiments.sh /home/manuel/workspaces/2026-03-01/generate-js-types
```

### Generated artifacts

- `sources/experiments/01-dts-surface-report.md`
- `sources/experiments/02-tsgen-capability-report.md`
- `sources/experiments/README.md`

## Experiment Results

### A. Geppetto surface complexity

From `01-dts-surface-report.md`:

- Lines: 498
- Top-level exports:
  - `const`: 8
  - `interface`: 41
  - `type`: 3
  - `function`: 3
- Grouped `const` object namespaces detected:
  - `consts` (10 members)
  - `turns` (8 members)
  - `engines` (4 members)
  - `profiles` (12 members)
  - `schemas` (2 members)
  - `middlewares` (2 members)
  - `tools` (1 member)
- Feature signals:
  - inline callbacks: 7
  - `Promise<...>`: 2
  - `Record<...>`: 28
  - `readonly`: 53
  - literal unions: 5

Interpretation:
- Geppetto declarations are not a small function-list API; they encode a broad and semantically rich type graph.

### B. tsgen model constraints

From `02-tsgen-capability-report.md`:

- `spec.Module` fields: `Name`, `Description`, `Functions`, `RawDTS`
- First-class support flags:
  - interfaces: `False`
  - type aliases: `False`
  - const declarations: `False`
- Renderer:
  - direct function rendering: `True`
  - raw passthrough: `True`
  - dedicated interface/type-alias render path: `False`

Interpretation:
- Current tsgen can represent geppetto fully only by heavily using `RawDTS`, which is not a true structural replacement.

## Gap Matrix

| Requirement for true replacement | Geppetto need | Current tsgen status | Gap severity |
| --- | --- | --- | --- |
| First-class interface modeling | High | Missing | Critical |
| First-class type alias modeling | Medium/High | Missing | High |
| First-class grouped const object modeling | High | Missing | Critical |
| Option-driven descriptor generation | High (runtime/config variants) | Missing in `TypeScriptDeclarer` | High |
| Validation for non-function declarations | High | Missing | High |
| Deterministic check-mode support | Required | Present (`--check`) | Low |
| Module registration discovery | Required | Present | Low |

## Inferred Constraints (explicitly inferred)

1. Inference: a drop-in cutover without tsgen model extension would force geppetto to maintain declaration text in `RawDTS`, effectively moving bespoke content rather than replacing it.
- Basis: `spec.Module` shape and renderer behavior (`types.go:26-32`, `dts_renderer.go:73-81`).

2. Inference: option-driven declaration surfaces (if geppetto needs feature/profile dependent signatures) require an additional declarer interface or command-level option channel.
- Basis: current declarer has no options (`modules/typing.go:5-9`).

3. Inference: runtime parity checks should remain mandatory post-migration to guard registration/descriptor drift.
- Basis: existing geppetto parity test already compares d.ts surface vs runtime exports (`dts_parity_test.go:24-50`).

## What This Means for Replacement Scope

A true replacement is a productized tsgen evolution project, not a thin adapter:

1. Extend tsgen spec to model declaration categories natively.
2. Extend renderer + validator accordingly.
3. Add option-aware declarer API.
4. Port geppetto declaration source of truth into structured descriptors.
5. Keep parity and check-mode gates through cutover.

## Usage Examples

### Re-run all experiments

```bash
TICKET=/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts
$TICKET/scripts/03_run_gap_experiments.sh /home/manuel/workspaces/2026-03-01/generate-js-types
```

### Inspect generated reports

```bash
sed -n '1,260p' $TICKET/sources/experiments/01-dts-surface-report.md
sed -n '1,260p' $TICKET/sources/experiments/02-tsgen-capability-report.md
```

### Reconfirm line-anchored evidence

```bash
nl -ba /home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/cmd/gen-meta/main.go | sed -n '137,185p'
nl -ba /home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/spec/types.go | sed -n '20,40p'
nl -ba /home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/render/dts_renderer.go | sed -n '49,110p'
```

## Related

- Design plan based on this evidence:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts/design-doc/01-investigation-and-design-geppetto-true-replacement-onto-registration-based-gen-dts.md`
- Chronological execution log:
  - `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/ttmp/2026/03/01/GP-34-GEPPETTO-GEN-DTS-TRUE-REPLACEMENT--true-replacement-design-geppetto-on-go-go-goja-registration-based-gen-dts/reference/01-implementation-diary.md`
