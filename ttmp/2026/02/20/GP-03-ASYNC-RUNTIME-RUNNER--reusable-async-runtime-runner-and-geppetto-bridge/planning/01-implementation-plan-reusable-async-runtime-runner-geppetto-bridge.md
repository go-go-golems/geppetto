---
Title: 'Implementation Plan: Reusable Async Runtime Runner + Geppetto Bridge'
Ticket: GP-03-ASYNC-RUNTIME-RUNNER
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - inference
    - middleware
    - tools
DocType: planning
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/js/modules/geppetto/api.go
      Note: Owner-thread migration scope
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Runner option wiring
    - Path: go-go-goja/pkg/runtimeowner/runner.go
      Note: Final runner behavior and owner-context design
    - Path: go-go-goja/pkg/runtimeowner/types.go
      Note: Final API signatures summarized in implementation outcome
ExternalSources: []
Summary: Detailed implementation analysis and phased build plan for a reusable runtime-owner async runner in go-go-goja and geppetto bridge integration, with intern-focused context and pseudocode.
LastUpdated: 2026-02-20T10:42:59.667100576-05:00
WhatFor: ""
WhenToUse: ""
---


# Implementation Plan: Reusable Async Runtime Runner + Geppetto Bridge

## 1) Purpose and audience

This document is the implementation map for ticket `GP-03-ASYNC-RUNTIME-RUNNER`.

It is written for:

- maintainers implementing the runner and geppetto bridge,
- reviewers who need to validate architecture and risk,
- new developers (including interns) who need full context and explicit next steps.

The goal is to explain not only what we will build, but why each piece exists, how pieces connect, and where bugs can reappear if boundaries are violated.

---

## Implementation outcome (2026-02-20)

The runner + geppetto integration scope for GP-03 is now implemented.

Code commits:

- `go-go-goja` commit `03a723b`: reusable `pkg/runtimeowner` runner + tests.
- `geppetto` commit `aad992c`: bridge wiring and callback-path migration.

Final implemented files:

- `go-go-goja/pkg/runtimeowner/types.go`
- `go-go-goja/pkg/runtimeowner/errors.go`
- `go-go-goja/pkg/runtimeowner/runner.go`
- `go-go-goja/pkg/runtimeowner/runner_test.go`
- `go-go-goja/pkg/runtimeowner/runner_race_test.go`
- `geppetto/pkg/js/runtimebridge/bridge.go`
- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/modules/geppetto/api.go`
- `geppetto/pkg/js/modules/geppetto/module_test.go`
- `geppetto/cmd/examples/geppetto-js-lab/main.go`

Final key signatures:

- `runtimeowner.NewRunner(vm *goja.Runtime, scheduler Scheduler, opts Options) Runner`
- `type runtimeowner.Runner interface { Call; Post; Shutdown; IsClosed }`
- `geppetto.Options.Runner runtimeowner.Runner`
- `func (m *moduleRuntime) callOnOwner(ctx context.Context, op string, fn func(context.Context) (any, error)) (any, error)`
- `func (m *moduleRuntime) postOnOwner(ctx context.Context, op string, fn func(context.Context)) error`

Notes from implementation:

- Async callback/value boundary calls are now routed through the runner bridge.
- New async regression tests for `runAsync` and `start` with JS callbacks pass under `-race`.
- Section 7.2 (`go-go-goja` bootstrap API changes) was intentionally deferred out of this ticket scope.

---

## 2) Scope and constraints

### In scope

- Build a reusable runtime-owner async runner in `go-go-goja`.
- Integrate geppetto to use that runner for all async VM-touching boundaries.
- Improve `go-go-goja` runtime/bootstrap API so third-party module registration is straightforward.
- Add race/stress/integration tests.

### Out of scope

- Feature additions to inference semantics unrelated to runtime safety.
- UI changes.
- Large model/provider behavior changes.

### Constraint confirmed by user

No backward compatibility is required for this ticket.

Implication:

- We can change API shapes directly instead of layering compatibility wrappers.
- We should still preserve clarity and migration notes, but we do not need compatibility shims.

---

## 3) Why this work exists

`goja` runtime objects are not safe for concurrent use by arbitrary goroutines. In our current setup, async inference paths can eventually execute JS callbacks (engines from function, JS middleware, JS tool handlers, JS tool hooks) while background goroutines are active.

Without a strict runtime ownership model, these patterns risk race conditions and hard-to-reproduce panics.

The architecture fix is to centralize all runtime interaction through a single owner-thread runner with explicit call semantics.

---

## 4) Mental model (for new developers)

Think in terms of two worlds:

1. **Go concurrency world**
- goroutines, network calls, tool execution, waiting, retries.

2. **JS runtime world**
- a single-threaded runtime context where JS values/functions are valid.

The bug class appears when code from world 1 reaches into world 2 directly.

The runner is the border checkpoint:

- all requests from world 1 to world 2 pass through one queue/dispatcher,
- world 2 executes them on owner thread,
- results are sent back in controlled form.

```text
+-----------------------------+         +-------------------------------+
| Go concurrency world        |  Call   | Runtime Runner                |
| (goroutines, tools, I/O)    +-------->+ (serialize + lifecycle guard) |
+-----------------------------+         +---------------+---------------+
                                                        |
                                                        v
                                            +-----------------------------+
                                            | goja runtime owner context  |
                                            +-----------------------------+
```

---

## 5) High-level architecture decisions

### Decision A: Put generic runner in `go-go-goja`

Reason:

- Runner semantics are generic to `goja` embedding, not geppetto-specific.
- Other modules can reuse it later.

### Decision B: Keep geppetto-specific adaptation in geppetto

Reason:

- Geppetto callback contracts (turn codec, tool hook payloads, run-handle semantics) are domain-specific.
- Generic runner should not import geppetto types.

### Decision C: Introduce explicit bootstrap extension in `go-go-goja`

Reason:

- Third-party module registration needs a pre-enable hook for registry wiring.
- Current `engine.New()` hides registry creation and enable sequence.

### Decision D: Break APIs cleanly (no compatibility layer)

Reason:

- User requested no backward compatibility burden.
- Clean API is easier to reason about than dual-path legacy bridges.

---

## 6) Proposed code layout

### `go-go-goja` new files

- `go-go-goja/pkg/runtimeowner/types.go`
- `go-go-goja/pkg/runtimeowner/runner.go`
- `go-go-goja/pkg/runtimeowner/errors.go`
- `go-go-goja/pkg/runtimeowner/runner_test.go`
- `go-go-goja/pkg/runtimeowner/runner_race_test.go`
- `go-go-goja/engine/runtime_ext.go` (bootstrap API)
- `go-go-goja/engine/bootstrap_test.go`

### `geppetto` new/changed files

- `geppetto/pkg/js/modules/geppetto/module.go` (options refactor)
- `geppetto/pkg/js/modules/geppetto/api.go` (bridge usage)
- `geppetto/pkg/js/modules/geppetto/codec.go` (safe conversion usage)
- `geppetto/pkg/js/runtimebridge/bridge.go` (new adapter layer)
- `geppetto/pkg/js/runtimebridge/bridge_test.go`
- `geppetto/pkg/js/modules/geppetto/module_test.go` (new async race-oriented tests)

---

## 7) Detailed API design

## 7.1 Generic runner API (go-go-goja)

### Scheduler abstraction

```go
// go-go-goja/pkg/runtimeowner/types.go
package runtimeowner

import "github.com/dop251/goja"

type Scheduler interface {
    RunOnLoop(func(*goja.Runtime)) bool
}
```

### Runner interface

```go
package runtimeowner

import "context"

type Runner interface {
    Call(ctx context.Context, op string, fn func(*goja.Runtime) (any, error)) (any, error)
    Post(ctx context.Context, op string, fn func(*goja.Runtime)) error
    Shutdown(ctx context.Context) error
    IsClosed() bool
}
```

### Options

```go
package runtimeowner

import (
    "time"
)

type Options struct {
    Name string
    MaxWait time.Duration
    RecoverPanics bool
}
```

### Errors

```go
var (
    ErrClosed = errors.New("runtime runner closed")
    ErrScheduleRejected = errors.New("runtime schedule rejected")
    ErrCanceled = errors.New("runtime call canceled")
    ErrPanicked = errors.New("runtime call panicked")
)
```

## 7.2 Better bootstrap API (go-go-goja)

```go
// go-go-goja/engine/runtime_ext.go
package engine

import (
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/require"
)

type Bootstrap struct {
    RuntimeConfig RuntimeConfig
    RequireOptions []require.Option

    // Optional hooks
    BeforeBuiltinModules func(vm *goja.Runtime, reg *require.Registry) error
    BeforeEnable func(vm *goja.Runtime, reg *require.Registry) error
    AfterEnable func(vm *goja.Runtime, req *require.RequireModule) error
}

func NewWithBootstrap(b Bootstrap) (*goja.Runtime, *require.RequireModule, error)
```

Notes:

- `New()` can remain as convenience wrapper internally calling `NewWithBootstrap(defaults)`.
- Since compatibility is not required, we can also choose to replace `NewWithConfig` behavior, but wrapper strategy remains cleaner.

## 7.3 Geppetto module options refactor

Current geppetto options include `Loop *eventloop.EventLoop`.

Target shape (breaking change allowed):

```go
// geppetto/pkg/js/modules/geppetto/module.go
package geppetto

import "github.com/go-go-golems/go-go-goja/pkg/runtimeowner"

type Options struct {
    Runner runtimeowner.Runner
    GoToolRegistry tools.ToolRegistry
    GoMiddlewareFactories map[string]MiddlewareFactory
    Logger zerolog.Logger
}
```

This change forces callers to pass explicit runtime runner; geppetto stops accepting raw event loop directly.

---

## 8) Runner implementation details

## 8.1 Core algorithm: `Call`

Pseudo implementation:

```go
func (r *runner) Call(ctx context.Context, op string, fn func(*goja.Runtime) (any, error)) (any, error) {
    if r.closed.Load() {
        return nil, ErrClosed
    }

    if ctx == nil {
        ctx = context.Background()
    }

    // channel buffered to avoid goroutine leak in edge cases
    resultCh := make(chan callResult, 1)

    accepted := r.scheduler.RunOnLoop(func(vm *goja.Runtime) {
        var out any
        var err error

        if r.opts.RecoverPanics {
            defer func() {
                if rec := recover(); rec != nil {
                    err = fmt.Errorf("%w: op=%s panic=%v", ErrPanicked, op, rec)
                }
                resultCh <- callResult{value: out, err: err}
            }()
        }

        out, err = fn(vm)
        if !r.opts.RecoverPanics {
            resultCh <- callResult{value: out, err: err}
        }
    })

    if !accepted {
        return nil, ErrScheduleRejected
    }

    select {
    case <-ctx.Done():
        return nil, fmt.Errorf("%w: %v", ErrCanceled, ctx.Err())
    case res := <-resultCh:
        return res.value, res.err
    }
}
```

## 8.2 Core algorithm: `Post`

```go
func (r *runner) Post(ctx context.Context, op string, fn func(*goja.Runtime)) error {
    if r.closed.Load() {
        return ErrClosed
    }

    accepted := r.scheduler.RunOnLoop(func(vm *goja.Runtime) {
        defer recoverLog(op)
        fn(vm)
    })

    if !accepted {
        return ErrScheduleRejected
    }
    return nil
}
```

## 8.3 Shutdown semantics

```go
func (r *runner) Shutdown(ctx context.Context) error {
    if !r.closed.CompareAndSwap(false, true) {
        return nil
    }
    // Optional: wait for in-flight call accounting to drain
    // depending on implementation complexity chosen.
    return nil
}
```

Important simplification:

- Keep `Shutdown` local to runner state; lifecycle of event loop itself remains managed by caller.

---

## 9) Geppetto bridge implementation details

## 9.1 Bridge responsibilities

Bridge handles geppetto-specific VM interactions:

- invoking JS callables safely,
- converting Go payloads to JS values on runtime owner,
- converting return values back into geppetto codec paths,
- preserving geppetto error context labels.

## 9.2 Bridge interface and helpers

```go
// geppetto/pkg/js/runtimebridge/bridge.go
package runtimebridge

type Bridge struct {
    runner runtimeowner.Runner
    vm *goja.Runtime
}

func New(r runtimeowner.Runner, vm *goja.Runtime) *Bridge

func (b *Bridge) InvokeCallable(ctx context.Context, op string, fn goja.Callable, this goja.Value, args ...goja.Value) (goja.Value, error)
func (b *Bridge) ToJSValue(ctx context.Context, op string, convert func(*goja.Runtime) (goja.Value, error)) (goja.Value, error)
func (b *Bridge) Run(ctx context.Context, op string, fn func(*goja.Runtime) error) error
```

Pseudo implementation for callable:

```go
func (b *Bridge) InvokeCallable(ctx context.Context, op string, fn goja.Callable, this goja.Value, args ...goja.Value) (goja.Value, error) {
    outAny, err := b.runner.Call(ctx, op, func(vm *goja.Runtime) (any, error) {
        v, callErr := fn(this, args...)
        if callErr != nil {
            return nil, callErr
        }
        return v, nil
    })
    if err != nil {
        return nil, err
    }
    v, ok := outAny.(goja.Value)
    if !ok {
        return nil, fmt.Errorf("bridge %s: expected goja.Value, got %T", op, outAny)
    }
    return v, nil
}
```

## 9.3 Integration call-site map in geppetto

Replace direct VM touchpoints in:

- `jsCallableEngine.RunInference`
- `jsMiddleware` function callback invocation
- tool handler wrapper callback invocation
- tool hooks: `Before`, `After`, `OnError`
- event collector payload conversion and listener callback execution
- promise resolve/reject sites in async paths

Rule for reviewers:

- any `goja.Callable(...)`, `vm.NewObject()`, `vm.NewArray()`, `vm.ToValue(...)` in async-callable paths must be in runner-managed closures.

---

## 10) Third-party app model after changes

### Desired usage

Third-party app can:

- create runtime via `go-go-goja` bootstrap,
- register built-ins and custom modules in hook,
- pass a runner to geppetto,
- use `database` and `geppetto` in one JS runtime.

Pseudocode:

```go
loop := eventloop.NewEventLoop()
go loop.Start()

var runner runtimeowner.Runner

vm, req, err := engine.NewWithBootstrap(engine.Bootstrap{
    RuntimeConfig: engine.DefaultRuntimeConfig(),
    BeforeEnable: func(vm *goja.Runtime, reg *require.Registry) error {
        // construct runner once vm + loop available
        runner = runtimeowner.NewRunner(vm, loop, runtimeowner.Options{Name: "app-runtime"})

        gp.Register(reg, gp.Options{
            Runner: runner,
            GoToolRegistry: myTools,
        })
        return nil
    },
})
if err != nil {
    return err
}
defer runner.Shutdown(context.Background())
_ = req
```

Script example:

```javascript
const db = require("database");
const gp = require("geppetto");

db.configure("sqlite3", ":memory:");

const s = gp.createSession({ engine: gp.engines.echo({ reply: "READY" }) });
s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("hi")] }));

const handle = s.start();
handle.on("*", (ev) => console.log(ev.type));
await handle.promise;
```

---

## 11) Phased implementation plan

## Phase 0: repository prep

- Create ticket docs and task plan (done).
- Freeze current async bug references for regression comparison.

## Phase 1: build generic runner in go-go-goja

Deliverables:

- `pkg/runtimeowner` package with tests.

Checklist:

1. Implement `Scheduler`, `Runner`, `Options`, errors.
2. Implement `Call`, `Post`, `Shutdown`.
3. Add unit tests for:
- happy path,
- canceled context,
- scheduler rejection,
- panic recovery,
- closed runner.
4. Add race-oriented tests that run many concurrent `Call`s.

## Phase 2: add bootstrap API in go-go-goja

Deliverables:

- `engine.NewWithBootstrap`.

Checklist:

1. Implement hook-aware constructor.
2. Ensure built-in module registration still works.
3. Add tests covering:
- hook order,
- hook error propagation,
- custom module registration before enable.

## Phase 3: add geppetto bridge

Deliverables:

- `pkg/js/runtimebridge` in geppetto.

Checklist:

1. Implement callable/value helper methods.
2. Add bridge-level tests with fake runner.

## Phase 4: migrate geppetto async boundaries

Deliverables:

- all target call sites routed through bridge.

Checklist:

1. engine callback path migrated.
2. middleware callback path migrated.
3. tool handler path migrated.
4. tool hook paths migrated.
5. event collector conversion/callback path migrated.
6. promise settlement path aligned to bridge helper.

## Phase 5: stress + regression tests

Deliverables:

- race/stress suites in geppetto and any needed harness tests.

Checklist:

1. async with JS engine callback under `-race`.
2. async with JS middleware/hooks under `-race`.
3. parallel tools + JS hooks under `-race`.
4. integration composition with `database` + `geppetto`.

## Phase 6: docs/examples polish

Deliverables:

- updated docs in both repos,
- one runnable example for third-party composition.

---

## 12) Pseudocode-heavy checklist by file

### `go-go-goja/pkg/runtimeowner/types.go`

```go
package runtimeowner

type Scheduler interface { RunOnLoop(func(*goja.Runtime)) bool }

type Runner interface {
    Call(context.Context, string, func(*goja.Runtime) (any, error)) (any, error)
    Post(context.Context, string, func(*goja.Runtime)) error
    Shutdown(context.Context) error
    IsClosed() bool
}

type Options struct {
    Name string
    MaxWait time.Duration
    RecoverPanics bool
}
```

### `go-go-goja/pkg/runtimeowner/runner.go`

```go
package runtimeowner

type runner struct {
    vm *goja.Runtime
    scheduler Scheduler
    opts Options
    closed atomic.Bool
}

func NewRunner(vm *goja.Runtime, s Scheduler, opts Options) Runner {
    if opts.Name == "" { opts.Name = "runtime" }
    return &runner{vm: vm, scheduler: s, opts: opts}
}

func (r *runner) IsClosed() bool { return r.closed.Load() }

// Call/Post/Shutdown as described earlier
```

### `go-go-goja/engine/runtime_ext.go`

```go
func NewWithBootstrap(b Bootstrap) (*goja.Runtime, *require.RequireModule, error) {
    cfg := b.RuntimeConfig
    vm := goja.New()

    reg := require.NewRegistry(b.RequireOptions...)

    if b.BeforeBuiltinModules != nil {
        if err := b.BeforeBuiltinModules(vm, reg); err != nil {
            return nil, nil, err
        }
    }

    modules.EnableAll(reg)

    if b.BeforeEnable != nil {
        if err := b.BeforeEnable(vm, reg); err != nil {
            return nil, nil, err
        }
    }

    req := reg.Enable(vm)

    if b.AfterEnable != nil {
        if err := b.AfterEnable(vm, req); err != nil {
            return nil, nil, err
        }
    }

    console.Enable(vm)
    return vm, req, nil
}
```

### `geppetto/pkg/js/runtimebridge/bridge.go`

```go
package runtimebridge

type Bridge struct {
    runner runtimeowner.Runner
}

func (b *Bridge) Call(ctx context.Context, op string, fn func(*goja.Runtime) (any, error)) (any, error) {
    return b.runner.Call(ctx, op, fn)
}

func (b *Bridge) InvokeCallable(ctx context.Context, op string, fn goja.Callable, this goja.Value, args ...goja.Value) (goja.Value, error) {
    out, err := b.Call(ctx, op, func(vm *goja.Runtime) (any, error) {
        v, e := fn(this, args...)
        if e != nil { return nil, e }
        return v, nil
    })
    if err != nil { return nil, err }
    return out.(goja.Value), nil
}
```

### `geppetto/pkg/js/modules/geppetto/module.go`

```go
type Options struct {
    Runner runtimeowner.Runner
    GoToolRegistry tools.ToolRegistry
    GoMiddlewareFactories map[string]MiddlewareFactory
    Logger zerolog.Logger
}

func Register(reg *require.Registry, opts Options) {
    if opts.Runner == nil {
        panic("geppetto: Runner is required")
    }
    // register as before
}
```

### `geppetto/pkg/js/modules/geppetto/api.go` migration pattern

Before:

```go
ret, err := fn(goja.Undefined(), jsTurn, m.vm.ToValue(nextFn), m.toJSValue(ctxPayload))
```

After:

```go
retVal, err := m.bridge.InvokeCallable(ctx, "js.middleware.invoke", fn,
    goja.Undefined(),
    jsTurn,
    m.bridgeMustToValue(ctx, "js.middleware.next", nextFn),
    m.bridgeMustToValue(ctx, "js.middleware.ctx", ctxPayload),
)
```

Pseudo helper:

```go
func (m *moduleRuntime) bridgeMustToValue(ctx context.Context, op string, payload any) goja.Value {
    v, err := m.bridge.ToJSValue(ctx, op, payload)
    if err != nil {
        panic(m.vm.NewGoError(err))
    }
    return v
}
```

---

## 13) Testing plan in detail

## 13.1 go-go-goja runner tests

- `TestRunnerCallSuccess`
- `TestRunnerCallContextCanceled`
- `TestRunnerCallSchedulerRejected`
- `TestRunnerCallPanicRecovered`
- `TestRunnerPost`
- `TestRunnerShutdown`
- `TestRunnerConcurrentCallsRace`

Pseudo race test:

```go
func TestRunnerConcurrentCallsRace(t *testing.T) {
    loop := eventloop.NewEventLoop()
    go loop.Start()
    defer loop.Stop()

    vm := goja.New()
    r := NewRunner(vm, loop, Options{RecoverPanics: true})

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            _, err := r.Call(context.Background(), "inc", func(vm *goja.Runtime) (any, error) {
                return i + 1, nil
            })
            require.NoError(t, err)
        }(i)
    }
    wg.Wait()
}
```

## 13.2 go-go-goja bootstrap tests

- `TestNewWithBootstrapHooksOrder`
- `TestNewWithBootstrapBeforeEnableCanRegisterModule`
- `TestNewWithBootstrapPropagatesHookError`

## 13.3 geppetto bridge tests

- fake runner to ensure operation names and payloads are passed correctly.
- real loop-backed runner tests for callable invocation.

## 13.4 geppetto async regression tests

- `runAsync` with `engines.fromFunction`.
- `start` with JS middleware + hooks.
- tool parallelism with JS handlers.
- event collector callback path.

All executed with race detector in CI.

---

## 14) Definition of done

Ticket is done when all are true:

1. `go-go-goja` has reusable runner package with passing unit and race tests.
2. `go-go-goja` exposes bootstrap API for third-party module registration.
3. geppetto requires runner and has no direct async VM boundary calls outside bridge.
4. geppetto async/race integration tests pass.
5. docs and examples show `database` + `geppetto` composition.

---

## 15) Risk register and mitigations

### Risk: deadlocks from nested calls

Mitigation:

- keep call closures short,
- avoid nested `Call` inside `Call` unless explicitly designed,
- add tests for reentrancy edge behavior.

### Risk: cancellation leaks

Mitigation:

- buffered result channels,
- strict select on `ctx.Done()`,
- avoid goroutines waiting forever on unconsumed results.

### Risk: geppetto migration misses a call site

Mitigation:

- search patterns in review checklist:
  - `goja.AssertFunction(` invocation paths,
  - direct `fn(goja.Undefined(),` patterns,
  - direct `vm.NewObject()/NewArray()/ToValue` in async paths.

### Risk: third-party app confusion around new API

Mitigation:

- keep a minimal quick-start and one full example.
- include migration notes in both repos.

---

## 16) Reviewer checklist

For PR review, validate these points explicitly:

- Runner package has zero geppetto imports.
- Geppetto bridge hides runner internals from business logic.
- Module options no longer accept raw loop; runner is explicit.
- No direct async callback invocation bypassing bridge.
- Race tests fail before patch / pass after patch (or equivalent demonstration).
- Bootstrap hook ordering and error propagation are tested.

---

## 17) Execution sequence for the intern (day-by-day)

### Day 1

- implement runner package skeleton and tests.
- get `go test ./...` passing in `go-go-goja`.

### Day 2

- add bootstrap API and tests.
- create tiny custom-module registration test fixture.

### Day 3

- add geppetto bridge package.
- migrate first path: JS engine callback.

### Day 4

- migrate middleware/tool hooks/tool handlers.
- run targeted geppetto tests.

### Day 5

- add race/stress tests.
- add docs and end-to-end example with `database` + `geppetto`.

If schedule compresses, prioritize correctness boundary first (runner + geppetto callback migration), then bootstrap ergonomics.

---

## 18) Final guiding principle

Everything in this ticket enforces one invariant:

- **No background goroutine may touch JS runtime state directly.**

If every change preserves that invariant, the architecture remains correct even as features evolve.
