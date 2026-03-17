---
Title: Implementation diary
Ticket: GP-38
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - js-bindings
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/eval.go
      Note: Existing eval sequencing reviewed before introducing the wrapper
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/tool.go
      Note: Prebuilt path targeted by the cleanup
ExternalSources: []
Summary: Chronological diary for the scopedjs shared-runtime execution cleanup.
LastUpdated: 2026-03-17T10:05:04.177173061-04:00
WhatFor: Record the exact commands, reasoning, implementation choices, and validation results for the shared-runtime wrapper cleanup.
WhenToUse: Use when reviewing how the cleanup was executed or when extending the same runtime-wrapper shape later for per-session pooling.
---

# Implementation diary

## Goal

Replace the implicit shared-runtime execution pattern in `scopedjs` with a clearer explicit wrapper that serializes eval calls on one reused runtime.

## Context

The immediate trigger was a concurrency bug in the prebuilt path: concurrent evals on the same runtime can interleave between `prepareEval(...)`, `executeEval(...)`, `waitForPromise(...)`, and `cleanupEval(...)`. The user asked specifically for the longer clear-term shape, not just a local mutex hidden in `RegisterPrebuilt(...)`.

## Quick Reference

### Initial commands

```bash
docmgr ticket create-ticket --root geppetto/ttmp --ticket GP-38 --title "Refactor scopedjs shared-runtime execution around a serialized runtime wrapper" --topics geppetto,tools,architecture,js-bindings
docmgr doc add --root geppetto/ttmp --ticket GP-38 --doc-type design-doc --title "Serialized shared-runtime executor cleanup plan" --summary "Design and implementation guide for moving scopedjs shared-runtime execution behind a serialized runtime wrapper."
docmgr doc add --root geppetto/ttmp --ticket GP-38 --doc-type reference --title "Implementation diary" --summary "Chronological diary for the scopedjs shared-runtime execution cleanup."
```

### Code paths reviewed before coding

- `geppetto/pkg/inference/tools/scopedjs/eval.go`
- `geppetto/pkg/inference/tools/scopedjs/tool.go`
- `geppetto/pkg/inference/tools/scopedjs/schema.go`
- `geppetto/pkg/inference/tools/scopedjs/tool_test.go`
- `geppetto/pkg/doc/tutorials/07-build-scopedjs-eval-tools.md`

### Planned implementation shape

- add a `RuntimeExecutor` wrapper with a mutex
- expose it on `BuildResult`
- switch `RegisterPrebuilt(...)` to use it
- add a concurrency regression test proving whole-eval serialization
- update the scopedjs tutorial

## Usage Examples

### Intended future prebuilt call path

```text
BuildRuntime(...)
  -> BuildResult{ Runtime, Executor, ... }

RegisterPrebuilt(...)
  -> handle.Executor.RunEval(...)
```

### Why this matters

This keeps the rule visible:

- raw runtime = low-level object
- executor = safe reused-runtime eval path

## Detailed Notes

### 1. Confirmed the bug is broader than console

The first code read was `eval.go`. The key conclusion was that `console` restoration is only the visible symptom. The real problem is that `RunEval(...)` spans multiple owner calls, so overlapping callers on one runtime can interleave across temporary input globals, console replacement, promise waiting, and cleanup.

### 2. Chose a wrapper instead of a local mutex

I considered a closure-local mutex inside `RegisterPrebuilt(...)`, but rejected it because it would bury the important lifecycle rule in one registrar implementation. A named wrapper is the better long-term shape and matches the user request.

### 3. Kept compatibility as a design constraint

I decided not to replace `BuildResult.Runtime`. The cleanup should add a clearer path, not force every caller to rewrite immediately. The wrapper can live alongside the raw runtime.

### 4. Implemented the wrapper in package code

I added:

- `pkg/inference/tools/scopedjs/executor.go`

That file now holds the explicit reused-runtime abstraction:

- `RuntimeExecutor`
- `NewRuntimeExecutor(...)`
- `(*RuntimeExecutor).RunEval(...)`

I also updated:

- `schema.go` so `BuildResult` exposes `Executor`
- `runtime.go` so `BuildRuntime(...)` populates that field
- `tool.go` so `RegisterPrebuilt(...)` and the lazy path both evaluate through the wrapper

### 5. Added a whole-lifecycle serialization regression test

Instead of testing only console output, I added a stronger prebuilt concurrency test in `tool_test.go`.

The test:

- starts one prebuilt eval that enters a pending promise and sets a shared `phase = "running"`
- starts a second eval on the same prebuilt runtime
- verifies the second eval does not complete before the first is released
- releases the first promise from the test
- verifies the second eval observes `phase = "done"`

That proves serialization across the full eval lifecycle, not just one console-specific detail.

### 6. Updated the public tutorial

I updated `pkg/doc/tutorials/07-build-scopedjs-eval-tools.md` so the developer-facing guidance now reflects the new shape:

- `BuildResult.Runtime` remains the raw runtime handle
- `BuildResult.Executor` is the safe reused-runtime eval wrapper
- `RegisterPrebuilt(...)` already uses the wrapper internally

### 7. Ran focused validation

Commands:

```bash
go test ./pkg/inference/tools/scopedjs
go test ./pkg/doc
docmgr doctor --root geppetto/ttmp --ticket GP-38 --stale-after 30
```

Results:

- both passed
- ticket validation passed

## Related

- Design guide: [../design-doc/01-serialized-shared-runtime-executor-cleanup-plan.md](../design-doc/01-serialized-shared-runtime-executor-cleanup-plan.md)
