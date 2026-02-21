---
Title: 'Intern Guide: Reusable Async Runtime Safety Runner (go-go-goja + geppetto)'
Ticket: GP-01-JS-IMPROVEMENTS
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
    - Path: geppetto/cmd/examples/geppetto-js-lab/main.go
      Note: Reference manual wiring pattern (event loop + registry + geppetto)
    - Path: geppetto/pkg/js/modules/geppetto/api.go
      Note: Async paths and callback boundaries to migrate via runner
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Geppetto registration/options surface and loop dependency
    - Path: go-go-goja/engine/runtime.go
      Note: Current runtime bootstrap flow and module enable point
    - Path: go-go-goja/modules/common.go
      Note: Generic NativeModule registry conventions and EnableAll
    - Path: go-go-goja/modules/database/database.go
      Note: Database module example used in third-party integration section
ExternalSources: []
Summary: Deep guide for building a reusable async VM-safety runner, integrating it into geppetto, and wiring third-party apps that combine go-go-goja modules (including database) with geppetto.
LastUpdated: 2026-02-20T09:55:03.387505785-05:00
WhatFor: ""
WhenToUse: ""
---


# Intern Guide: Reusable Async Runtime Safety Runner (go-go-goja + geppetto)

## Why this document exists

This guide is for developers new to this codebase (especially interns and first-time contributors) who need to understand three related things:

1. Why async safety is a hard requirement when embedding goja JavaScript.
2. How to build a reusable VM-safety runner in `go-go-goja` (so multiple modules can share one safe pattern).
3. How geppetto should consume that runner, and what a third-party app should look like when it wants both `database` (from `go-go-goja`) and `geppetto` in one runtime.

This is intentionally verbose and implementation-oriented. It combines architecture prose, concrete signatures, pseudocode, and diagrams.

---

## Current state (quick orientation)

Before designing anything new, it helps to know the current building blocks.

### go-go-goja today

`go-go-goja` currently provides:

- a module registry abstraction via `modules.NativeModule`, `modules.Register`, `modules.EnableAll` in `go-go-goja/modules/common.go`.
- a runtime bootstrap helper in `go-go-goja/engine/runtime.go` (`New`, `NewWithOptions`, `NewWithConfig`) that:
  - creates `vm := goja.New()`,
  - creates a `require.Registry`,
  - enables all registered modules,
  - enables console,
  - returns `(*goja.Runtime, *require.RequireModule)`.

This is a useful host/bootstrap library, but its current `engine.New...` API does not expose the registry pre-enable hook needed to inject custom modules like geppetto in a first-class way.

### geppetto today

Geppetto’s JS module API is in `geppetto/pkg/js/modules/geppetto`.

The module registration point is:

- `geppetto/pkg/js/modules/geppetto/module.go`
- `func Register(reg *require.Registry, opts Options)`

and `Options` now expects:

- `Runner runtimeowner.Runner`
- optional tool registry and middleware factories.

This is good, explicit, and runtime-friendly.

### the mismatch

- `go-go-goja` module system is based on `NativeModule.Loader(vm, moduleObj)`.
- geppetto is currently a registry-level module registration (`Register(reg, opts)`) rather than a `NativeModule` implementation.
- `go-go-goja/engine.New...` creates and enables a registry internally, so there is no direct app hook for `geppetto.Register(reg, opts)` before enable.

This mismatch is why third-party apps currently need either:

- manual runtime assembly (recommended today for geppetto integration), or
- future extensions to `go-go-goja` bootstrap APIs.

---

## Problem statement: async safety, in one sentence

When code can run concurrently, all goja runtime interactions must be serialized onto one runtime owner thread; otherwise you risk races/panics and undefined behavior.

For geppetto this is especially relevant because async inference paths can trigger JS callbacks (engine functions, middleware, tool handlers, tool hooks).

---

## Design goals for the reusable runner

We want a reusable component that is generic enough for go-go-goja modules and specific enough to enforce safety invariants.

### Functional goals

- Support async/background work while keeping VM access serialized.
- Provide request/response call semantics to VM thread.
- Support fire-and-forget posts (notifications/events).
- Support cancellation and clean shutdown behavior.
- Surface deterministic errors when runtime/loop is not available.

### Safety goals

- No direct VM calls outside the runner boundary in async paths.
- Reentrancy-safe behavior (no deadlocks when called from owner thread).
- Panic capture and conversion to Go errors.

### Developer-experience goals

- API simple enough to read in 5 minutes.
- Minimal ceremony for common usage (`Call`, `InvokeCallable`, `ToValue`).
- Rich logging/metrics hooks.
- Works for geppetto and future modules without geppetto-specific types.

---

## Conceptual model: Runtime Owner + Dispatcher

Think of the runner as a small RPC layer into the VM owner thread.

```text
                        +----------------------------------+
background goroutines   |   Runtime Safety Runner          |
(tool exec, inference,  |  (single entry point for VM)     |
network I/O)            |                                  |
      |                 |  queue -> run on VM owner -> ret |
      | Call/Post       +------------------+---------------+
      +------------------------------------|
                                           v
                                 +--------------------+
                                 | goja Runtime owner |
                                 | (event loop thread)|
                                 +--------------------+
```

Key idea: callers do not touch `vm` directly; they submit closures that the runner executes on the owner context.

---

## Implemented reusable package in go-go-goja

### Suggested package and files

Place a reusable component under:

- `go-go-goja/pkg/runtimeowner/runner.go`
- `go-go-goja/pkg/runtimeowner/types.go`
- `go-go-goja/pkg/runtimeowner/errors.go`
- `go-go-goja/pkg/runtimeowner/runner_test.go`
- `go-go-goja/pkg/runtimeowner/runner_race_test.go`

This keeps it neutral and reusable across modules.

### Core interface design

```go
// go-go-goja/pkg/runtimeowner/types.go
package runtimeowner

import (
    "context"
    "github.com/dop251/goja"
)

type Scheduler interface {
    // Must schedule fn on VM owner context.
    // Returns false when scheduler is terminated/unavailable.
    RunOnLoop(fn func(*goja.Runtime)) bool
}

type Runner interface {
    Call(ctx context.Context, op string, fn CallFunc) (any, error)
    Post(ctx context.Context, op string, fn PostFunc) error
    Shutdown(ctx context.Context) error
    IsClosed() bool
}
```

Notes for interns:

- We use `any` in baseline API to keep it Go-version-friendly and easy to consume.
- You can add typed wrappers on top (`CallValue[T]`) later.

### Concrete implementation signature

```go
// go-go-goja/pkg/runtimeowner/runner.go
func NewRunner(vm *goja.Runtime, scheduler Scheduler, opts Options) Runner

type Options struct {
    Name string
    MaxWait int64 // milliseconds
    RecoverPanics bool
}
```

### Error set

```go
var (
    ErrClosed           = errors.New("runtime runner closed")
    ErrScheduleRejected = errors.New("runtime schedule rejected")
    ErrCanceled         = errors.New("runtime call canceled")
    ErrPanicked         = errors.New("runtime call panicked")
)
```

---

## Runner behavior in detail

### `Call` behavior

`Call` means: schedule VM closure and synchronously wait for result.

Behavior contract:

- If runner is closed: return `ErrClosed`.
- If caller context is canceled before completion: return `ErrCanceled` + `ctx.Err()` cause.
- If scheduler rejects (loop terminated/stopped): return `ErrScheduleRejected`.
- If closure panics: recover and return `ErrPanicked` when panic recovery is enabled.
- If called from owner context: execute inline to avoid deadlock.

Pseudocode:

```go
func (r *runner) Call(ctx context.Context, op string, fn func(*goja.Runtime) (any, error)) (any, error) {
    if r.closed.Load() {
        return nil, ErrClosed
    }

    if r.isOwnerContext(ctx) {
        return safeInvoke(op, r.vm, fn)
    }

    done := make(chan result, 1)
    accepted := r.scheduler.RunOnLoop(func(vm *goja.Runtime) {
        v, err := safeInvoke(op, vm, fn)
        done <- result{v: v, err: err}
    })
    if !accepted {
        return nil, ErrScheduleRejected
    }

    select {
    case <-ctx.Done():
        return nil, wrap(ErrCanceled, ctx.Err())
    case out := <-done:
        return out.v, out.err
    }
}
```

### `Post` behavior

`Post` means: enqueue VM closure, do not wait for result.

Use cases:

- event emission,
- promise settlement when caller does not need sync result,
- low-priority logging hooks.

Behavior contract:

- returns immediate error if runner closed/scheduler rejected,
- closure panic is recovered and logged.

### owner context detection

The current implementation uses runner-internal context markers, not explicit thread IDs. Whenever work executes on owner via `Call`/`Post`, the runner stamps the context and nested calls fast-path inline.

---

## Helpers that geppetto will need on top

To keep geppetto integration clean, add helper wrappers in geppetto (or in a small shared adapter package):

- `InvokeCallable(ctx, callable, this, args...)`
- `ToJSValue(ctx, payload)`
- `CreatePromise(ctx)`
- `ResolvePromise(ctx, token, value)`
- `RejectPromise(ctx, token, err)`

Conceptual signatures:

```go
// geppetto/pkg/js/runtimebridge/bridge.go (proposed)
type Bridge interface {
    Call(ctx context.Context, op string, fn func(*goja.Runtime) (any, error)) (any, error)
}

func InvokeCallable(ctx context.Context, b Bridge, fn goja.Callable, this goja.Value, args ...goja.Value) (goja.Value, error)
func ToJSValue(ctx context.Context, b Bridge, payload any) (goja.Value, error)
```

Why not put these in go-go-goja?

- they embed assumptions about how you want to marshal payloads and errors in geppetto.
- keep base runner generic, keep business semantics local.

---

## How geppetto should consume the runner

This is the critical part.

### Current geppetto call sites to migrate

From `geppetto/pkg/js/modules/geppetto/api.go`, async-sensitive VM boundaries include:

- JS engine callback invocation.
- JS middleware callback invocation.
- JS tool handler callback invocation.
- JS tool hooks (`before/after/onError`).
- async event payload conversion and listener callback invocations.

### Implemented geppetto runtime struct changes

Current `Options` uses `Runner runtimeowner.Runner`.

Refactor direction:

```go
type Options struct {
    Runner runtimeowner.Runner
    GoToolRegistry tools.ToolRegistry
    GoMiddlewareFactories map[string]MiddlewareFactory
    Logger zerolog.Logger
}
```

At runtime init:

- runner is injected once in options.
- module runtime builds one bridge from the runner and reuses it for all callback/value boundaries.

Then instantiate a runner/bridge once per module runtime and reuse everywhere.

### Migration pattern per call site

For each direct callback:

- before:
  - call `fn(...)` directly from whichever goroutine you are on.
- after:
  - call `bridge.InvokeCallable(...)` which delegates to runner.

For each off-loop value conversion:

- before:
  - call `m.toJSValue(...)` directly.
- after:
  - build Go DTO first,
  - convert in runner `Call` closure.

### Expected outcome

- async inference stays async,
- VM access becomes serialized,
- race risk drops to architecture-level guardrails rather than manual vigilance.

---

## What should go-go-goja expose for third-party apps

Third-party app developers need one of two integration levels:

1. **Current model (works now):** manual runtime + registry wiring.
2. **Future convenience model (recommended enhancement):** bootstrap API that lets app mutate registry before enable.

### 1) Current model (works today)

Even without changing go-go-goja engine APIs, apps can do this:

- create VM manually,
- create `require.Registry`,
- call `modules.EnableAll(reg)` (register built-in modules like `database`),
- call `geppetto.Register(reg, opts)`,
- enable registry on VM.

Pseudocode:

```go
package main

import (
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/console"
    "github.com/dop251/goja_nodejs/eventloop"
    "github.com/dop251/goja_nodejs/require"
    "github.com/go-go-golems/go-go-goja/pkg/runtimeowner"

    ggmodules "github.com/go-go-golems/go-go-goja/modules"
    _ "github.com/go-go-golems/go-go-goja/modules/database" // ensure database init registration

    gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
)

func buildRuntime() (*goja.Runtime, *require.RequireModule, *eventloop.EventLoop, error) {
    vm := goja.New()
    console.Enable(vm)

    loop := eventloop.NewEventLoop()
    go loop.Start()
    runner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
        Name: "notes-assistant",
        RecoverPanics: true,
    })

    reg := require.NewRegistry()

    // 1) Register go-go-goja native modules, including database.
    ggmodules.EnableAll(reg)

    // 2) Register geppetto explicitly.
    gp.Register(reg, gp.Options{
        Runner: runner,
        // GoToolRegistry: your tool registry,
    })

    req := reg.Enable(vm)
    return vm, req, loop, nil
}
```

JS side in this app:

```javascript
const db = require("database");
const gp = require("geppetto");

db.configure("sqlite3", ":memory:");
db.exec("create table notes(id integer primary key, text text)");
db.exec("insert into notes(text) values (?)", "hello from db");

const rows = db.query("select * from notes");
console.log(rows);

const s = gp.createSession({ engine: gp.engines.echo({ reply: "READY" }) });
s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("hi")] }));
const out = s.run();
console.log(out.blocks[out.blocks.length - 1]);
```

### 2) Future convenience model (recommended for go-go-goja)

Add an engine constructor that exposes a pre-enable registry hook.

Proposed filename/signature:

- `go-go-goja/engine/runtime_ext.go`

```go
type BootstrapOption func(*Bootstrap)

type Bootstrap struct {
    RuntimeConfig RuntimeConfig
    RequireOptions []require.Option

    // Hook app can use to register custom modules (geppetto, etc.)
    BeforeEnable func(reg *require.Registry)
}

func NewWithBootstrap(b Bootstrap) (*goja.Runtime, *require.RequireModule, error)
```

Usage:

```go
vm, req, err := engine.NewWithBootstrap(engine.Bootstrap{
    RuntimeConfig: engine.DefaultRuntimeConfig(),
    BeforeEnable: func(vm *goja.Runtime, reg *require.Registry) {
        runner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
            Name: "app-runtime",
            RecoverPanics: true,
        })
        gp.Register(reg, gp.Options{Runner: runner, GoToolRegistry: toolReg})
    },
})
```

This keeps go-go-goja generic and makes geppetto integration ergonomic.

---

## Architecture diagrams for the recommended end state

### High-level composition

```text
+-------------------------------------------------------------+
| Third-party app                                              |
|                                                             |
|  +------------------+    +-------------------------------+  |
|  | go-go-goja       |    | geppetto                      |  |
|  | runtime bootstrap|    | require("geppetto") module    |  |
|  +---------+--------+    +---------------+---------------+  |
|            |                             |                  |
|            +-------------+---------------+                  |
|                          v                                  |
|                 +------------------+                        |
|                 | require.Registry |                        |
|                 +--------+---------+                        |
|                          |                                  |
|                          v                                  |
|                    +-----------+                            |
|                    | goja VM   |                            |
|                    +-----------+                            |
|                          ^                                  |
|                          |                                  |
|               +----------+-----------+                      |
|               | Runtime Safety Runner|                      |
|               | (owner-thread bridge)|                      |
|               +----------------------+                      |
+-------------------------------------------------------------+
```

### Async inference control flow with safe runner

```text
JS calls runAsync/start
        |
        v
geppetto starts background inference goroutine
        |
        v
needs JS callback (engine/mw/tool hook)
        |
        v
bridge.Call(ctx, op, fn(vm)->result)
        |
        v
runner schedules fn on VM owner context
        |
        v
callback executes safely on owner thread
        |
        v
result returned to background goroutine
        |
        v
inference continues; promise/event settlement also via runner
```

---

## Concrete migration plan (engineering work breakdown)

### Phase A: foundation in go-go-goja

- Add `pkg/runtimeowner` runner package with tests.
- Add minimal metrics/log hooks.
- Add docs and examples in `go-go-goja` showing safe async pattern.

Deliverables:

- `go-go-goja/pkg/runtimeowner/*`
- `go-go-goja/ttmp/...` design note
- unit tests + race tests

### Phase B: geppetto bridge integration

- Add bridge wrapper package in geppetto.
- Wire bridge into module runtime init.
- Replace direct VM callback/value call sites.
- Make `Options.Runner` the required async wiring surface.

Deliverables:

- `geppetto/pkg/js/runtimebridge/*` (or equivalent)
- updated `module.go` options
- updated callback/hook/event paths

### Phase C: app ergonomics in go-go-goja

- Add `engine.NewWithBootstrap` or similar API hook.
- Add sample app combining `database` + `geppetto`.

Deliverables:

- `go-go-goja/engine/runtime_ext.go`
- integration example in `go-go-goja/cmd/...` or docs

### Phase D: hardening

- add cross-repo race/stress tests.
- add CI checks for forbidden direct VM access in async paths.

---

## Testing strategy for interns (what to run, what to look for)

### Unit tests

- Runner `Call` happy path.
- cancellation path.
- scheduler rejection path.
- panic recovery path.
- owner-thread reentrancy path.

### Integration tests

- geppetto async run with JS engine function.
- geppetto async start with JS middleware and tool hooks.
- high parallel tools with JS handlers.
- database + geppetto coexistence in one runtime.

### Race detector

Run race suites in both repos:

```bash
cd go-go-goja && go test ./... -race
cd geppetto && go test ./... -race
```

What to look for:

- no data race reports touching goja runtime internals,
- deterministic behavior under repeated runs,
- no deadlocks/hangs under cancellation/shutdown.

---

## Common pitfalls and how to avoid them

1. Calling `goja.Callable` directly from background goroutines.
- Fix: always go through runner bridge.

2. Building `goja.Value` off owner context.
- Fix: keep DTO in Go; convert inside runner `Call` closure.

3. Using `engine.New()` then trying to register geppetto afterward.
- Fix: manual runtime/registry wiring today, or use future bootstrap hook.

4. Nested sync calls causing deadlocks.
- Fix: owner-thread detection and inline execution path.

5. Ignoring loop termination/shutdown.
- Fix: deterministic errors and clear lifecycle handling in runner.

---

## Minimal signature checklist (implemented)

These signatures are enough to implement the first usable version.

### go-go-goja

- `pkg/runtimeowner/types.go`
  - `type Scheduler interface { RunOnLoop(func(*goja.Runtime)) bool }`
  - `type Runner interface { Call(...); Post(...); Shutdown(...); IsClosed() bool }`

- `pkg/runtimeowner/runner.go`
  - `func NewRunner(vm *goja.Runtime, scheduler Scheduler, opts Options) Runner`

- `engine/runtime_ext.go` (recommended)
  - `func NewWithBootstrap(b Bootstrap) (*goja.Runtime, *require.RequireModule, error)`

### geppetto

- `pkg/js/modules/geppetto/module.go`
  - `type Options struct { Runner runtimeowner.Runner ... }`

- `pkg/js/runtimebridge/bridge.go` (proposed)
  - `func (b *Bridge) InvokeCallable(...)`
  - `func (b *Bridge) ToJSValue(...)`

- `pkg/js/modules/geppetto/api.go`
  - replace direct callback invocations with bridge calls.

---

## Example “third-party app architecture” narrative

Imagine an app named `notes-assistant`.

It wants:

- `database` module from go-go-goja for local state,
- `geppetto` module for inference/tool orchestration,
- one JavaScript runtime so scripts can access both modules.

The app startup would:

- construct VM and event loop,
- create require registry,
- enable go-go-goja modules,
- register geppetto with loop and tool registry,
- enable registry on VM,
- run scripts that `require("database")` and `require("geppetto")`.

Operationally, async inference callbacks inside geppetto would route through the shared runner, while database calls remain regular Go module calls.

This gives a coherent runtime model for scripts and a safer concurrency model for native code.

---

## Closing guidance for the new intern

If you are starting implementation tomorrow, begin with this order:

1. Build and test the generic runner package in `go-go-goja` first.
2. Add one geppetto callback path through the bridge (e.g. JS engine callback) and verify race tests.
3. Migrate remaining callback paths iteratively.
4. Add a tiny third-party sample app that combines `database` + `geppetto`.
5. Only then optimize APIs and polish ergonomics.

The north star is simple:

- “async is fine,
- concurrent VM access is not.”

Design everything around that invariant and the rest of the architecture becomes much easier to reason about.


---

## Addendum (v2): `NativeModule` vs `NativeModule.Loader()` and a better third-party registration API

This section answers two practical architecture questions that usually come up when people first read `go-go-goja`:

1. What exactly is `NativeModule`?
2. How is `NativeModule.Loader()` different from the module itself?
3. What should a better `go-go-goja` API look like so third-party modules (like geppetto) register easily?

### 1) What is `NativeModule`?

In `go-go-goja/modules/common.go`, `NativeModule` is the module contract/interface:

```go
type NativeModule interface {
    Name() string
    Doc() string
    Loader(*goja.Runtime, *goja.Object)
}
```

Conceptually, `NativeModule` is the **module descriptor + installer contract**. It tells the registry:

- what the module is called (`Name`),
- what docs/help text it has (`Doc`),
- how to install exports into Node-style module `exports` (`Loader`).

So “what is NativeModule?”

- It is not a JS value.
- It is not the runtime.
- It is a Go-side module definition that can be registered once and then enabled on one or more registries.

### 2) What is `NativeModule.Loader()` specifically?

`Loader(vm, moduleObj)` is a callback invoked by `require` when module code is being initialized.

Responsibilities of `Loader`:

- read `exports := moduleObj.Get("exports")`,
- set exported functions/objects on `exports`,
- optionally capture `vm` for closure-based helpers,
- do lightweight setup needed for JS-facing API.

Non-responsibilities of `Loader`:

- it should not be your app bootstrap,
- it should not create global runtime wiring across unrelated modules,
- it should not perform long-blocking operations on VM thread.

A simple mental model:

- `NativeModule` = “module type/class”.
- `Loader` = “module constructor/install hook for a runtime”.

### 3) Why geppetto looks a little different today

Geppetto currently exposes `Register(reg, opts)` directly (in `geppetto/pkg/js/modules/geppetto/module.go`) and then internally calls `reg.RegisterNativeModule(...)`.

That means geppetto is still a native module, but registration is done via its own package-level function because it needs explicit options (loop/tool registry/middleware factories/logger).

So this is valid and intentional:

- `go-go-goja` typical module path: `modules.Register(&m{})` during init.
- geppetto path: app calls `gp.Register(reg, gp.Options{...})` when it has runtime-specific dependencies.

### 4) Why third-party registration feels awkward today

`go-go-goja/engine.New()` currently does all of this internally:

1. `vm := goja.New()`
2. `reg := require.NewRegistry(...)`
3. `modules.EnableAll(reg)`
4. `req := reg.Enable(vm)`

This is convenient for built-ins, but there is no first-class pre-enable hook for app-specific module registration.

That is why apps using geppetto either:

- manually build runtime+registry, or
- need new bootstrap APIs in go-go-goja.

### 5) Better API shape for third-party modules

A better API should satisfy both:

- keep simple `engine.New()` for quick starts,
- allow deterministic customization points for third-party module wiring.

Recommended additions:

#### A. Bootstrap struct with hooks

```go
// go-go-goja/engine/bootstrap.go (proposed)
type Bootstrap struct {
    RuntimeConfig RuntimeConfig
    RequireOptions []require.Option

    // Called after registry creation, before modules.EnableAll (optional).
    BeforeBuiltinModules func(reg *require.Registry)

    // Called after modules.EnableAll, before reg.Enable(vm) (optional).
    BeforeEnable func(vm *goja.Runtime, reg *require.Registry)

    // Optional final hook after reg.Enable(vm).
    AfterEnable func(vm *goja.Runtime, req *require.RequireModule)
}

func NewWithBootstrap(b Bootstrap) (*goja.Runtime, *require.RequireModule, error)
```

Why two hooks (`BeforeBuiltinModules`, `BeforeEnable`)?

- Some apps may want to override/replace built-ins.
- Others just want to add extra modules after built-ins are loaded.

#### B. Functional options variant

If you prefer option functions:

```go
type RuntimeOption func(*runtimeBuild)

func WithRequireOption(opt require.Option) RuntimeOption
func WithRegistryMutator(fn func(vm *goja.Runtime, reg *require.Registry)) RuntimeOption
func WithModuleRegistrar(fn func(reg *require.Registry)) RuntimeOption

func NewExt(opts ...RuntimeOption) (*goja.Runtime, *require.RequireModule, error)
```

This is flexible but usually less discoverable than a typed bootstrap struct for newcomers.

#### C. Optional module composer helper

For very frequent module composition patterns:

```go
type ModuleRegistrar interface {
    RegisterTo(reg *require.Registry) error
}

func ComposeRegistrars(rs ...ModuleRegistrar) func(*require.Registry) error
```

Then adapters can be written once per third-party module.

### 6) Adapter pattern for geppetto under improved API

Because geppetto currently uses `Register(reg, opts)`, the app can provide a registrar closure:

```go
func GeppettoRegistrar(opts gp.Options) func(vm *goja.Runtime, reg *require.Registry) {
    return func(vm *goja.Runtime, reg *require.Registry) {
        gp.Register(reg, opts)
    }
}
```

This keeps geppetto decoupled from go-go-goja internals and still gives ergonomic app bootstrap.

### 7) Example: ideal third-party app setup (`database` + `geppetto`)

With proposed bootstrap API:

```go
loop := eventloop.NewEventLoop()
go loop.Start()

vm, req, err := engine.NewWithBootstrap(engine.Bootstrap{
    RuntimeConfig: engine.DefaultRuntimeConfig(),
    BeforeEnable: func(vm *goja.Runtime, reg *require.Registry) {
        runner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
            Name: "notes-assistant",
            RecoverPanics: true,
        })
        gp.Register(reg, gp.Options{
            Runner: runner,
            GoToolRegistry: toolRegistry,
        })
    },
})
if err != nil { /* handle */ }
_ = req
```

JS runtime usage:

```javascript
const db = require("database");
const gp = require("geppetto");

// database module still works
// geppetto module also available in same runtime
```

### 8) API quality checklist (what “better” means)

When reviewing the improved go-go-goja API, check these criteria:

- Third-party modules can register without forking `engine.New()`.
- Order of registration is explicit and documented.
- Built-in path remains one-liner for simple users.
- Hook errors propagate clearly (no silent partial setup).
- Tests cover module collision scenarios (same module name registered twice).
- Documentation includes a complete geppetto + database example.

### 9) Migration strategy for minimal disruption

1. Add `NewWithBootstrap` without changing `New`.
2. Re-implement `New` as thin wrapper over `NewWithBootstrap` default config.
3. Add docs and examples.
4. Migrate advanced apps (geppetto users) gradually.

This keeps backward compatibility while unlocking clean third-party module integration.

### 10) Final takeaway for interns

If you remember one sentence:

- `NativeModule` describes **what** a module is,
- `Loader` describes **how** it is installed into one runtime,
- and a good engine API decides **when** third-party module registration hooks run.

That separation of concerns is what makes a JavaScript embedding ecosystem scalable.
