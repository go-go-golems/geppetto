---
Title: Investigation diary
Ticket: GP-34
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - backend
    - js-bindings
    - go-api
    - security
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/tools/scopeddb/tool.go
      Note: |-
        Registration precedent inspected first
        First concrete code precedent inspected
    - Path: geppetto/pkg/js/modules/geppetto/api_tools_registry.go
      Note: |-
        Existing JS tool registry surface inspected for overlap and gaps
        Existing JS tool registry inspected for overlap and gaps
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Existing native module inspected for runtime bridge usage
    - Path: geppetto/pkg/js/runtime/runtime.go
      Note: |-
        Runtime bootstrap inspected for the JS side
        Runtime bootstrap inspected for JS-side reuse
    - Path: geppetto/ttmp/2026/03/15/GP-33--extract-scoped-db-tool-pattern-into-reusable-geppetto-package/index.md
      Note: |-
        Immediate predecessor ticket for the reusable scoped DB pattern
        Immediate predecessor ticket used as the package-structure precedent
ExternalSources: []
Summary: Short diary of the initial GP-34 ticket creation and repository inspection focused on the new scoped DB package and current JS runtime wiring.
LastUpdated: 2026-03-15T23:14:16.477036579-04:00
WhatFor: Preserve the initial command trail and concrete repository findings that shaped the GP-34 ticket scope.
WhenToUse: Use when revisiting why GP-34 was scoped as a reusable eval-style JS runtime tool instead of a generic JS cleanup task.
---


# Investigation diary

## Goal

Create a new docmgr ticket for the proposed scoped JavaScript tool runtime, inspect the codebase enough to anchor the ticket to real implementation seams, and restate the task in precise Geppetto terms.

## Context

The immediate precedent is the ticket and implementation work for the reusable scoped DB tool pattern. The user asked for the analogous JS-side abstraction: instead of scoping an SQLite snapshot and exposing a query tool, scope a goja runtime and expose one eval tool with registered modules, globals, bootstrap scripts, and documentation.

## Quick Reference

Files inspected first:

- `geppetto/ttmp/2026/03/15/GP-33--extract-scoped-db-tool-pattern-into-reusable-geppetto-package/index.md`
- `geppetto/pkg/inference/tools/scopeddb/tool.go`
- `geppetto/pkg/inference/tools/registry.go`
- `geppetto/pkg/js/runtime/runtime.go`
- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/modules/geppetto/api_tools_registry.go`
- `geppetto/pkg/js/runtimebridge/bridge.go`
- `geppetto/pkg/doc/topics/13-js-api-reference.md`

Main findings:

- `scopeddb` already demonstrates the exact reusable registration pattern to mirror.
- Geppetto already owns a reusable goja runtime bootstrap and owner-thread bridge.
- JS can already define tools dynamically, but there is no package that exposes one prepared runtime as one LLM-facing tool.
- Documentation for JS capabilities already exists and should be folded into the tool description story.

## Usage Examples

Use this diary when continuing GP-34 work to avoid restarting from a blank slate. The next design step should build directly on the scoped DB package shape and decide the JS equivalent of:

- dataset spec,
- prebuilt registrar,
- lazy registrar,
- and auto-generated tool description.

## Related

- `../analysis/01-scoped-javascript-tool-runtime-analysis-and-proposal.md`
- `../design-doc/01-scoped-javascript-eval-tools-architecture-design-and-implementation-guide.md`
- `../index.md`

## 2026-03-16 Implementation Start

### Goal

Turn the design ticket into executable work by expanding GP-34 into granular implementation slices and then starting the first real package checkpoint.

### What changed in approach

The work is now explicitly split into four slices:

1. package skeleton and pure-data API,
2. runtime construction and eval execution,
3. tool registration and integration,
4. examples, docs, and finish.

This is deliberate. The first slice is chosen so it can be implemented, tested, and committed independently before the runtime execution logic is introduced.

### New technical finding

While preparing the implementation plan, I found an important reusable reference in `go-go-goja/pkg/repl/evaluators/javascript/evaluator.go`. It already contains owner-thread evaluation and promise-settling logic for goja runtimes. That means the future `scopedjs` runtime-execution slice should probably reuse or adapt that logic instead of inventing its own promise polling behavior from scratch.

## 2026-03-16 Slice 1 Checkpoint

### Goal

Land the first real code checkpoint: the package skeleton and pure-data API layer, without yet attempting runtime execution or tool registration.

### Files added

- `geppetto/pkg/inference/tools/scopedjs/schema.go`
- `geppetto/pkg/inference/tools/scopedjs/builder.go`
- `geppetto/pkg/inference/tools/scopedjs/description.go`
- `geppetto/pkg/inference/tools/scopedjs/helpers.go`
- `geppetto/pkg/inference/tools/scopedjs/schema_test.go`
- `geppetto/pkg/inference/tools/scopedjs/description_test.go`

### What was implemented

- Core public types for the proposed package:
  - tool metadata,
  - eval options,
  - state modes,
  - environment spec,
  - build result,
  - eval input/output,
  - manifest doc types.
- A builder that records:
  - custom module registrations,
  - native modules,
  - globals,
  - runtime initializers,
  - bootstrap sources/files,
  - helper docs.
- Description rendering that turns the manifest plus tool notes into one LLM-facing description string.
- Unit tests for:
  - default eval options,
  - builder validation,
  - manifest cloning,
  - description formatting.

### Verification

I verified the first slice with:

```bash
env GOWORK=off GOCACHE=/tmp/geppetto-go-build go test ./pkg/inference/tools/scopedjs ./pkg/js/runtime
```

Notes:

- `GOWORK=off` was necessary because the workspace `go.work` file currently lags the `go 1.26.1` requirement declared by sibling modules.
- `GOCACHE=/tmp/geppetto-go-build` was necessary because the default build cache location was not writable in the earlier sandboxed run.

### Outcome

The package now has a stable type-and-description layer that can be committed independently. The next slice should use this foundation to implement runtime construction, bootstrap loading, owned evaluation, and promise handling.

## 2026-03-16 Slice 2 Checkpoint

### Goal

Implement the actual runtime build and eval path so `scopedjs` is no longer only a design skeleton.

### Main implementation decisions

- `BuildRuntime(...)` now builds directly on `go-go-goja/engine.NewBuilder()` rather than the higher-level Geppetto runtime wrapper.
- Builder-owned custom modules are converted into `engine.ModuleSpec` adapters.
- Builder-owned globals are installed via generated runtime initializers.
- Bootstrap scripts are loaded after runtime construction on the runtime owner thread.
- `RunEval(...)` executes user code by wrapping it in an async function body, then waiting for returned promises to settle.
- Promise waiting logic was adapted from `go-go-goja/pkg/repl/evaluators/javascript/evaluator.go`.

### Important API correction

During implementation, `GlobalBinding` had to change shape. The earlier first-slice type accepted `func(context.Context, *gojengine.Runtime) error`, but that was awkward because globals are installed during runtime initialization when the code naturally has a `*gojengine.RuntimeContext`, not a fully wrapped `*gojengine.Runtime`. The type is now aligned with that reality and takes `*gojengine.RuntimeContext`.

### Files added in this slice

- `geppetto/pkg/inference/tools/scopedjs/runtime.go`
- `geppetto/pkg/inference/tools/scopedjs/eval.go`
- `geppetto/pkg/inference/tools/scopedjs/runtime_test.go`

### Verification

I verified the runtime slice with:

```bash
env GOWORK=off GOCACHE=/tmp/geppetto-go-build go test ./pkg/inference/tools/scopedjs
```

Coverage in this checkpoint includes:

- runtime build with native module + global + bootstrap source + bootstrap file,
- eval returning structured objects,
- console capture,
- async promise fulfillment,
- promise rejection surfaced in `EvalOutput.Error`,
- timeout surfaced in `EvalOutput.Error`.

### Outcome

`scopedjs` now has a working runtime/eval core. The next slice should wire this into the Geppetto tool registry with `RegisterPrebuilt(...)` and `NewLazyRegistrar(...)`, then add provider-schema tests and app-facing examples.
