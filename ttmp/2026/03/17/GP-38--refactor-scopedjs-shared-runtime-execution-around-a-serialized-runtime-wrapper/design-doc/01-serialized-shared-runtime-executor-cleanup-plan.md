---
Title: Serialized shared-runtime executor cleanup plan
Ticket: GP-38
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - js-bindings
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/eval.go
      Note: Multi-phase eval logic that must not interleave on a shared runtime
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/tool.go
      Note: Prebuilt registration currently calling RunEval on the raw runtime
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/schema.go
      Note: BuildResult API that can expose the clearer wrapper without breaking compatibility
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/tool_test.go
      Note: Best place to add prebuilt concurrency regression tests
ExternalSources: []
Summary: Design and implementation guide for moving scopedjs shared-runtime execution behind a serialized runtime wrapper.
LastUpdated: 2026-03-17T10:05:04.010389796-04:00
WhatFor: Explain why shared-runtime execution needs an explicit wrapper and how to implement it without destabilizing existing scopedjs APIs.
WhenToUse: Use when implementing or reviewing the shared-runtime serialization cleanup, especially if future per-session pooling will reuse the same wrapper.
---

# Serialized shared-runtime executor cleanup plan

## Executive Summary

`scopedjs` currently exposes a raw `*gojengine.Runtime` through `BuildResult.Runtime`, and `RegisterPrebuilt(...)` directly calls `RunEval(...)` against that runtime. That is easy to read, but it leaves shared-runtime execution semantics implicit. The current implementation bug shows why that is not enough: one eval call is split into several runtime-owner calls, so concurrent callers can interleave those phases on the same reused runtime.

The cleanup in this ticket is to introduce one explicit wrapper type that owns serialized evaluation for a reused runtime. `BuildRuntime(...)` should populate that wrapper, `RegisterPrebuilt(...)` should use it, and the developer docs should explain that this wrapper is the right abstraction whenever one runtime instance survives across calls.

## Problem Statement

### The current unsafe shape

Today the shared-runtime path looks like this:

```text
BuildRuntime(...)
    -> BuildResult{ Runtime: rawRuntime, ... }

RegisterPrebuilt(...)
    -> tool closure
    -> RunEval(ctx, handle.Runtime, in, opts)
```

The problem is in `RunEval(...)` itself. It is not one atomic runtime operation. It is a sequence:

1. prepare input globals and optional console replacement
2. execute wrapped JavaScript
3. poll promise state if the script returned a pending promise
4. restore console and delete temporary globals

That means two callers on the same reused runtime can overlap like this:

```text
call A: prepare
call A: execute -> promise pending
call B: prepare
call B: execute
call B: cleanup
call A: cleanup
```

This is what makes stale console restoration possible. It is also more general than console:

- temporary globals can overlap
- cleanup from one eval can restore state expected by another eval
- any future reused-runtime feature would inherit the same bug if it uses raw `RunEval(...)`

### Why this is architectural, not just a local bug

The immediate symptom is console corruption, but the actual missing abstraction is “safe execution on a reused runtime.” That concept deserves a named type.

If the code continues to pass raw runtimes around for shared execution, every future lifecycle mode has to rediscover the same rule:

- prebuilt shared runtime needs serialization
- per-session pooled runtime needs one lock per session runtime
- any cached runtime manager needs the same behavior

This is the signal that the system wants a wrapper.

## Proposed Solution

### New wrapper type

Add a new type, for example:

```go
type RuntimeExecutor struct {
    Runtime *gojengine.Runtime
    mu      sync.Mutex
}

func NewRuntimeExecutor(rt *gojengine.Runtime) *RuntimeExecutor

func (r *RuntimeExecutor) RunEval(
    ctx context.Context,
    in EvalInput,
    opts EvalOptions,
) (EvalOutput, error)
```

Behavior:

- holds one runtime
- serializes one entire `RunEval(...)` call with a mutex
- exposes the raw runtime for compatibility and exceptional low-level access

### BuildResult integration

Keep compatibility:

```go
type BuildResult[Meta any] struct {
    Runtime   *gojengine.Runtime
    Executor  *RuntimeExecutor
    Meta      Meta
    Manifest  EnvironmentManifest
    Cleanup   func() error
}
```

Why add instead of replace:

- existing code may still read `BuildResult.Runtime`
- examples and tests can migrate gradually
- the wrapper becomes the preferred path without a breaking API cut

### RegisterPrebuilt integration

Change:

```go
return RunEval(ctx, handle.Runtime, in, evalOpts)
```

to something conceptually like:

```go
executor := executorFromBuildResult(handle)
return executor.RunEval(ctx, in, evalOpts)
```

Where `executorFromBuildResult(...)` falls back safely if a caller manually built a `BuildResult` with only `Runtime` populated.

## Design Decisions

### Decision 1: add a wrapper instead of only adding a mutex inside RegisterPrebuilt

Why:

- a closure-local mutex would fix the bug, but it would hide the important abstraction
- future per-session pooling needs the same concept on pool entries
- a named type makes the ownership model visible in the API

### Decision 2: keep `BuildResult.Runtime` for compatibility

Why:

- this is cleanup work, not a breaking redesign
- callers may inspect or use the raw runtime for bootstrap/debug/test code
- the wrapper should become preferred, not mandatory overnight

### Decision 3: lock around the entire `RunEval(...)`, not just console replacement

Why:

- the shared-state hazard spans `prepare`, `execute`, `waitForPromise`, and `cleanup`
- fixing only console would leave other interleavings in place
- the full eval call is the correct isolation unit for reused-runtime execution

### Decision 4: keep lazy behavior unchanged

Why:

- lazy registration already builds one runtime per call
- there is no reuse to protect across callers
- API consistency is useful, but semantics must not change

It is fine if lazy code uses the wrapper internally for uniformity, but it should not add new lifecycle behavior.

## Alternatives Considered

### Alternative A: add a mutex local to RegisterPrebuilt only

Pros:

- smallest patch
- fixes the immediate bug

Cons:

- leaves no reusable abstraction for per-session runtime pooling
- hides the serialization rule in one registrar implementation
- does not improve the public shape of `BuildResult`

Rejected because the user explicitly asked for the longer clear-term shape, not only the smallest fix.

### Alternative B: stack or namespace console wrappers so concurrent evals can coexist

Rejected because:

- it addresses only one visible symptom
- temporary input globals and cleanup sequencing still overlap
- shared reused-runtime execution still lacks a coherent ownership model

### Alternative C: make `RunEval(...)` itself globally synchronized

Rejected because:

- `RunEval(...)` is also used with fresh runtimes
- the needed lock belongs to the runtime instance, not to the package globally
- different runtimes should still be able to execute concurrently

## Implementation Plan

### Phase 1: Add the wrapper type

Steps:

1. Add a new file such as `executor.go`.
2. Define `RuntimeExecutor`.
3. Add a constructor that returns nil or a usable wrapper for a non-nil runtime.
4. Add a `RunEval(...)` method that locks and delegates to the existing package-level `RunEval(...)`.

Pseudocode:

```go
type RuntimeExecutor struct {
    Runtime *gojengine.Runtime
    mu      sync.Mutex
}

func NewRuntimeExecutor(rt *gojengine.Runtime) *RuntimeExecutor {
    if rt == nil {
        return nil
    }
    return &RuntimeExecutor{Runtime: rt}
}

func (r *RuntimeExecutor) RunEval(ctx context.Context, in EvalInput, opts EvalOptions) (EvalOutput, error) {
    if r == nil || r.Runtime == nil {
        return EvalOutput{}, fmt.Errorf("runtime is nil")
    }
    r.mu.Lock()
    defer r.mu.Unlock()
    return RunEval(ctx, r.Runtime, in, opts)
}
```

### Phase 2: Expose it from BuildResult

Steps:

1. Add `Executor *RuntimeExecutor` to `BuildResult`.
2. Populate it in `BuildRuntime(...)` with `NewRuntimeExecutor(rt)`.
3. Keep `Runtime` untouched.

### Phase 3: Use it in prebuilt registration

Steps:

1. Add a helper to choose `handle.Executor` when available.
2. Fall back to `NewRuntimeExecutor(handle.Runtime)` if the field is nil.
3. Make `RegisterPrebuilt(...)` call the wrapper.

### Phase 4: Add regression tests

Test shape:

- create a prebuilt tool with a bootstrap helper that returns a pending promise and marks a shared phase variable
- start one call that waits on that promise
- start a second call on the same tool
- release the first promise later
- assert the second call observes serialized ordering instead of interleaving

This is stronger than only testing console restoration because it proves whole-eval serialization.

### Phase 5: Update docs

At minimum update:

- `pkg/doc/tutorials/07-build-scopedjs-eval-tools.md`

The tutorial should explain:

- `BuildResult.Runtime` is the raw runtime handle
- `BuildResult.Executor` is the safe wrapper for reused-runtime evals
- `RegisterPrebuilt(...)` already uses the executor
- future runtime-pool features will likely reuse the same abstraction

## Intern Notes

If you are a new intern working on this cleanup, keep these rules in mind:

- Do not rewrite `RunEval(...)` unless a test proves you need to.
- Add the wrapper around the existing function first.
- Write the concurrency regression test before trying more ambitious refactors.
- Preserve the current lazy vs prebuilt semantics.
- Avoid API churn that would force every example and caller to change immediately.

## Open Questions

- Should the wrapper expose any method beyond `RunEval(...)`, or is that premature?
- Should examples eventually prefer `handle.Executor` explicitly, or is it enough that `RegisterPrebuilt(...)` uses it?

## References

- `geppetto/pkg/inference/tools/scopedjs/eval.go`
- `geppetto/pkg/inference/tools/scopedjs/tool.go`
- `geppetto/pkg/inference/tools/scopedjs/schema.go`
- `geppetto/pkg/inference/tools/scopedjs/tool_test.go`
- `geppetto/ttmp/2026/03/16/GP-36--review-and-cleanup-scopedjs-and-scopedjs-demo-work-since-origin-main/design-doc/01-scopedjs-and-demo-review-cleanup-analysis-design-and-implementation-guide.md`
