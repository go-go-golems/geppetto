---
Title: 'Deadlock Analysis: owner thread run() + async callback scheduling'
Ticket: GP-03-ASYNC-RUNTIME-RUNNER
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - inference
    - middleware
    - tools
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/js/modules/geppetto/api.go
      Note: Deadlock timeline and callback boundary behavior
    - Path: geppetto/pkg/js/modules/geppetto/module_test.go
      Note: Test harness change that triggered deadlock
    - Path: go-go-goja/pkg/runtimeowner/runner.go
      Note: Owner-context and scheduler behavior discussed in analysis
ExternalSources: []
Summary: Detailed analysis of the GP-03 deadlock encountered when synchronous session.run() executed on the runtime owner thread while background inference attempted to schedule JS callback work on that same owner queue.
LastUpdated: 2026-02-20T11:27:37.84312788-05:00
WhatFor: ""
WhenToUse: ""
---


# Deadlock Analysis: owner thread `run()` + async callback scheduling

## Executive Summary

The deadlock happened when test code executed JavaScript through `runtimeowner.Runner.Call(...)`, which means the entire JS script ran on the goja owner loop thread. Inside that script, `session.run()` was called synchronously. `session.run()` blocks until inference completes. Inference executes in a background goroutine and eventually reaches JS middleware/engine callbacks that were migrated to `callOnOwner(...)`, which schedules work back onto the same owner loop thread.

At that point we had a classic wait cycle:

- Owner thread is blocked waiting for inference to finish.
- Inference goroutine is blocked waiting for owner thread to run a scheduled callback.

No participant can progress, so test timed out.

## Context

The GP-03 fix intentionally moved async VM-touching operations onto owner-thread scheduling. That migration is correct for race safety. The deadlock was introduced by a test harness change, not by the bridge migration itself:

- Before: tests called `vm.RunString(...)` directly from test goroutine.
- During migration attempt: tests switched to `runner.Call(... vm.RunString(...))`.
- Result: synchronous `run()` execution now occupied owner thread while background callback work needed owner thread.

## The Exact Failure Signal

Failing command:

```bash
cd geppetto && go test ./pkg/js/modules/geppetto -count=1 -timeout=45s -v
```

Observed failure:

- `panic: test timed out after 45s`
- Hot goroutines showed:
  - one blocked in `runtimeowner.(*runner).Call` waiting on result channel;
  - one blocked in `sessionRef.runSync` waiting `handle.Wait()`;
  - one background inference goroutine trying to re-enter owner via `callOnOwner` from JS middleware path.

This confirms a circular wait rather than a panic/race.

## Step-by-Step Timeline

1. Test executes `mustRunJS(...)`.
2. `mustRunJS(...)` enters `runner.Call("module_test.RunString", fn)`.
3. Scheduler runs `vm.RunString(script)` on owner thread.
4. Script invokes `s.run()`.
5. `s.run()` -> `runSync()` -> `StartInference()` and then `Wait()` (blocking).
6. Inference goroutine reaches JS middleware/engine callback.
7. Callback path calls `callOnOwner(ctx, "middleware.fromJS", ...)`.
8. `callOnOwner` schedules with runner/scheduler and waits for owner completion.
9. Owner is still blocked in step 5 waiting for inference.
10. Inference cannot complete because step 8 cannot run.

Deadlock cycle closes.

## Concurrency Graph

```text
Owner loop thread
  └─ runner.Call(module_test.RunString)
      └─ vm.RunString(...)
          └─ JS s.run()
              └─ runSync() waits on handle.Wait()  [BLOCKED]

Inference goroutine
  └─ engine/middleware JS callback path
      └─ callOnOwner(...)
          └─ scheduler.RunOnLoop(...) + wait result  [BLOCKED]

Cycle:
Owner waits for inference
Inference waits for owner
```

## Why Owner-Context Fast-Path Did Not Break the Cycle

`runtimeowner` has owner-context fast-path logic: if a call is already on owner context, execute inline instead of queueing. That protects nested callbacks that happen *within* owner-owned execution.

Here, inference callback executed in a different goroutine created by session inference (`StartInference`), not the owner loop goroutine. Therefore `isOwnerContext(ctx)` was false and normal scheduling path was used. That was correct behavior.

So the deadlock was not a bug in owner-context detection; it was a usage-level reentrancy inversion caused by running a blocking API on the owner thread.

## Why Direct `vm.RunString(...)` in Tests Avoided Deadlock

With test host-thread execution:

- test goroutine calls `vm.RunString(...)` (not via owner-runner scheduling wrapper);
- `s.run()` blocks test goroutine, not owner loop queue;
- owner loop remains available for inference callback scheduling;
- callback executes, inference completes, `run()` returns.

This is why reverting test/script execution to host-thread call resolved the hang while preserving the async safety migration.

## Root Cause Statement

Root cause is **blocking owner-loop occupancy by synchronous API (`run()`) while downstream work requires owner-loop scheduling to complete**.

More generally:

- any API that blocks waiting for background work must not run on the same serialized owner executor if that background work can call back into owner.

## Safety Rule Derived

Do not execute synchronous `session.run()` through `runtimeowner.Runner.Call(...)` (or any owner-loop wrapper) when middleware/engine/tool hooks may schedule owner callbacks.

Equivalent policy:

- `run()` may run off-owner.
- callback internals may hop to owner.
- owner must remain dispatchable while `run()` is waiting.

## Practical Prevention Options

### Option 1 (current applied fix)

Keep test harness host-thread execution (`vm.RunString` / `vm.RunScript`) and only route callback boundaries via bridge helpers.

### Option 2

Document/guard: detect owner-thread invocation for synchronous `run()` and return explicit error (fast fail) instead of deadlocking.

Pseudo-guard idea:

```go
if runtimeowner.IsOwnerContext(ctx) {
    return nil, errors.New("session.run cannot execute on runtime owner thread")
}
```

### Option 3

Introduce a fully async public API path in tests/callers (`start()` / `runAsync()`) and avoid synchronous blocking APIs inside owner-runner calls.

## Why This Does Not Invalidate GP-03 Migration

The GP-03 migration goal was race safety at async boundaries (engine callbacks, middleware, JS tool handlers/hooks, event sink, promise settlement). Those paths remain correct and verified with `-race` after the harness rollback.

The deadlock was created by changing the harness execution model for synchronous scripts, not by the boundary migration design itself.

## Verification After Fix

Commands run after rollback:

```bash
cd geppetto && go test ./pkg/js/modules/geppetto -count=1 -timeout=180s
cd geppetto && go test ./pkg/js/modules/geppetto -race -count=1 -timeout=240s
cd geppetto && go test ./cmd/examples/geppetto-js-lab -count=1
```

All passed.

## Actionable Guidance for New Developers

If you are integrating new JS callback paths:

1. Route all VM value creation/callback invocation through `callOnOwner` / `postOnOwner`.
2. Avoid calling blocking synchronous flows from inside owner-runner calls.
3. If a flow waits on background goroutines and those goroutines call JS, keep owner queue unblocked.
4. Add tests that exercise mixed sync+async behavior and run with `-race`.

## Related Files

- `geppetto/pkg/js/modules/geppetto/api.go`
- `geppetto/pkg/js/modules/geppetto/module_test.go`
- `go-go-goja/pkg/runtimeowner/runner.go`
- `geppetto/pkg/js/runtimebridge/bridge.go`
