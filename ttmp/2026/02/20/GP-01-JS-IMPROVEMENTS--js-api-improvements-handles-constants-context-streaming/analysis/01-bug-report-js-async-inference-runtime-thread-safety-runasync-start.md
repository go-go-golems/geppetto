---
Title: 'Bug Report: JS Async Inference Runtime Thread-Safety (runAsync/start)'
Ticket: GP-01-JS-IMPROVEMENTS
Status: active
Topics:
    - bug
    - geppetto
    - go
    - inference
    - middleware
    - tools
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../go/pkg/mod/github.com/dop251/goja@v0.0.0-20250630131328-58d95d85e994/builtin_promise.go
      Note: Upstream NewPromise goroutine-safety warning and event-loop usage pattern
    - Path: ../../../../../../../../../../go/pkg/mod/github.com/dop251/goja_nodejs@v0.0.0-20240728170619-29b559befffc/eventloop/eventloop.go
      Note: RunOnLoop semantics and loop threading guarantees
    - Path: geppetto/pkg/inference/session/session.go
      Note: StartInference goroutine model underpinning async execution
    - Path: geppetto/pkg/inference/tools/base_executor.go
      Note: Parallel tool execution behavior relevant to JS tool callback safety
    - Path: geppetto/pkg/inference/tools/config.go
      Note: Default MaxParallelTools value impacting concurrency risk
    - Path: geppetto/pkg/js/modules/geppetto/api.go
      Note: Primary async/session API and all JS callback invocation sites analyzed
    - Path: geppetto/pkg/js/modules/geppetto/codec.go
      Note: toJSValue conversion path that can construct goja values off-loop
    - Path: go-go-goja/README.md
      Note: Comparison baseline for documented async pattern in companion repo
ExternalSources: []
Summary: Detailed analysis of goja event-loop safety bug in geppetto JS async inference, comparison with goja/goja_nodejs patterns, and fix options.
LastUpdated: 2026-02-20T09:28:25.437545341-05:00
WhatFor: ""
WhenToUse: ""
---


# Bug Report: JS Async Inference Runtime Thread-Safety (runAsync/start)

## 1) Executive Summary

There is a real concurrency bug in the JS async inference path in `pkg/js/modules/geppetto/api.go`: `runAsync()` launches `sr.runSync(...)` in a background goroutine, while `runSync()` can execute JS-backed callbacks (engine, middleware, tool handlers, tool hooks) that directly call `goja` callables.

This violates the `goja` runtime safety contract whenever the JS/event loop can continue running concurrently.

The review comment is correct in substance:

- `runAsync()` currently does:
  - create JS Promise,
  - `go func() { out, err := sr.runSync(...) ... }`,
  - settle promise via `RunOnLoop(...)`.
- But `runSync()` can cross back into JS during inference, and those callback invocations are not marshaled to the loop thread.
- Result: runtime can be touched concurrently, causing races and possible panics/corruption.

Important expansion: this is not isolated to `runAsync()`. `start()` also runs inference in goroutines and has the same JS-callback boundary problem. In addition, several code paths construct `goja.Value` off-loop.

Recommended direction:

- Keep inference/background work asynchronous.
- Marshal every JS runtime boundary (calling JS callbacks and creating JS values) onto a single VM thread using `RunOnLoop` + synchronous handoff.
- Treat `runAsync`/`start` fix as part of a larger “JS boundary serialization” fix, not a single-line goroutine removal.

---

## 2) Scope and Evidence Base

This report is based on current local code in:

- `geppetto/pkg/js/modules/geppetto/api.go`
- `geppetto/pkg/js/modules/geppetto/codec.go`
- `geppetto/pkg/inference/session/session.go`
- `geppetto/pkg/inference/toolloop/enginebuilder/builder.go`
- `geppetto/pkg/inference/toolloop/loop.go`
- `geppetto/pkg/inference/tools/base_executor.go`
- `geppetto/pkg/js/embeddings-js.go`
- `geppetto/cmd/examples/geppetto-js-lab/main.go`

And upstream/runtime references:

- `goja/README.md`
- `goja/builtin_promise.go`
- `goja/typedarrays.go`
- `goja/runtime.go`
- `goja_nodejs/eventloop/eventloop.go`
- `goja_nodejs/eventloop/eventloop_test.go`

And comparative repo:

- `go-go-goja/README.md`
- `go-go-goja/engine/runtime.go`
- `go-go-goja/modules/common.go`

---

## 3) Current Async Architecture in Geppetto JS Module

### 3.1 Public session run APIs

In `pkg/js/modules/geppetto/api.go`:

- `session.run(...)` calls `sr.runSync(...)` directly.
- `session.start(...)` returns a run handle with promise/events/cancel and launches background goroutine.
- `session.runAsync(...)` returns a promise and launches background goroutine.

Relevant code:

- `run()` around `api.go:412-434`
- `start()` around `api.go:628-713`
- `runAsync()` around `api.go:480-503`
- `runSync()` around `api.go:464-478`

### 3.2 `runSync` internals

`runSync(seed, opts)` currently does:

1. optionally `session.Append(seed)`
2. build context
3. `session.StartInference(ctx)`
4. `handle.Wait()`

So `runSync` itself is synchronous from the caller’s perspective, but it delegates to `session.StartInference`, which is asynchronous internally.

### 3.3 `session.StartInference` behavior

In `pkg/inference/session/session.go`:

- `StartInference` creates runner, creates `ExecutionHandle`, and then starts a goroutine (`session.go:242`) that runs `runner.RunInference(...)`.
- That means inference body is not executed on the caller goroutine.

This is key: `runSync()` is blocking, but the actual inference execution is happening in a goroutine.

### 3.4 Where inference can call JS directly

In `pkg/js/modules/geppetto/api.go`, JS callables are invoked directly in multiple inference-layer components:

1. JS engine callback:
- `jsCallableEngine.RunInference(...)` calls `e.fn(...)` directly (`api.go:116-133`).

2. JS middleware callback:
- `jsMiddleware(...)` wrapper calls `fn(...)` directly (`api.go:1598-1673`, especially `1660`).

3. JS tool handler callback:
- tool registry wrapper calls `handler(...)` directly (`api.go:1755-1801`, especially `1796`).

4. JS tool hooks:
- `beforeToolCall` via `e.hooks.Before(...)` (`api.go:1080-1126`, especially `1099`)
- `afterToolCall` via `e.hooks.After(...)` (`api.go:1128-1177`, especially `1151`)
- `onToolError` via `e.hooks.OnError(...)` (`api.go:1179-1250`, especially `1216`)

None of these call sites currently force execution onto loop thread.

---

## 4) The Bug: Why `runAsync` is Unsafe

### 4.1 Current `runAsync` implementation

`runAsync` does:

- create promise on VM (`api.go:484`)
- spawn goroutine (`api.go:486`)
- inside goroutine call `sr.runSync(seed, runOptions{})` (`api.go:487`)
- use `loop.RunOnLoop` only to resolve/reject promise (`api.go:488-499`)

So only promise settlement is loop-serialized. Inference itself is not.

### 4.2 Race timeline

Representative unsafe timeline:

1. JS calls `session.runAsync(...)`.
2. `runAsync` spawns goroutine G1.
3. loop/event thread continues to process JS (e.g. timers, other handlers, next operations).
4. G1 enters inference path; during inference, JS-backed engine/middleware/tool hook calls into `goja` directly.
5. loop thread and G1 can now both touch runtime concurrently.

This is exactly the class of race/panic called out by the review note.

### 4.3 Why this violates goja contract

Upstream goja states runtime is not goroutine-safe:

- `goja/README.md:99-103`: runtime can only be used by a single goroutine at a time.
- `goja/builtin_promise.go:610-612`: promise objects and resolvers are not goroutine-safe; use event loop pattern.
- `goja/runtime.go:2430-2431`: APIs not safe for concurrent use, only VM goroutine.
- `goja/typedarrays.go:101-102`: “may not be called concurrently” style constraints are explicit.

So settling promise on loop but invoking JS callables off-loop is not sufficient.

---

## 5) Upstream Pattern: How goja/goja_nodejs Async Is Intended

### 5.1 Event-loop execution model

`goja_nodejs/eventloop` provides a queue/owner-thread model:

- `RunOnLoop` schedules function on loop context (`eventloop.go:314-320`).
- docs repeatedly state runtime values must not be used outside callback boundaries (`eventloop.go:159-161`, `187-189`, `316-317`).

### 5.2 Canonical Promise pattern

Upstream `goja` `NewPromise` comment shows canonical async pattern (`builtin_promise.go:606-628`):

- create promise on loop/VM context,
- run blocking work in goroutine,
- call `resolve/reject` back on loop via `RunOnLoop`.

`goja_nodejs` test uses the same shape (`eventloop_test.go:516-533`).

### 5.3 Critical nuance

This pattern assumes the background goroutine does not touch VM/JS runtime directly. It does external/blocking work only, then posts result back.

Geppetto `runAsync` violates that assumption when inference path itself includes JS callbacks.

---

## 6) How This Compares to go-go-goja

### 6.1 go-go-goja core runtime setup

`go-go-goja/engine/runtime.go` sets up plain `goja.Runtime` + `require` modules (`runtime.go:50-65`) and is mostly synchronous.

`go-go-goja/modules/common.go` is module registration plumbing; no async/event-loop orchestration by default (`common.go:77-83`).

### 6.2 Async guidance in go-go-goja docs

`go-go-goja/README.md` contains an “Asynchronous APIs” section (`README.md:137+`) describing the same recommended pattern: background work + `RunOnLoop` for resolve/callback.

This is conceptually aligned with upstream goja/goja_nodejs guidance.

### 6.3 What it means for this ticket

So, yes: the intended approach in goja ecosystem is loop-serialized runtime interaction.

But geppetto’s async inference path differs materially because inference may execute JS callables in the background worker path itself. That part is not “the standard pattern”.

---

## 7) Important Expansion: `start()` Has the Same Core Problem

`start()` in `api.go:628-713` also runs inference in goroutine and only marshals promise/event notifications back via `RunOnLoop`.

Any direct JS callback inside inference (engine/middleware/tool handlers/hooks) can still run off-loop and race with active loop JS.

So a fix only in `runAsync()` is incomplete.

---

## 8) Additional Runtime-Safety Hazards Observed

These are relevant because they impact a robust fix scope.

### 8.1 JS value construction off-loop

`toJSValue` in `codec.go:405-449` allocates JS objects/arrays via `m.vm.NewObject()`, `m.vm.NewArray()`, `m.vm.ToValue(...)`.

In async paths, some payloads are converted before `RunOnLoop`, e.g. event collector:

- `PublishEvent` computes `payload := c.encodeEvent(ev)` before posting to loop (`api.go:558-560`).
- `encodeEvent` calls `c.api.toJSValue(payload)` (`api.go:567-626`).

So VM object creation can happen off-loop.

### 8.2 Tool parallelism can amplify races

`tools.DefaultToolConfig()` sets `MaxParallelTools: 3` (`tools/config.go:24`).

`BaseToolExecutor.ExecuteToolCalls` uses `executeParallel` with goroutines (`base_executor.go:243-259`).

If JS tool handlers/hooks are involved and direct callback invocation remains off-loop, concurrency pressure increases.

### 8.3 Session inference always goroutine-driven

Even “sync” run path calls `StartInference`, which itself launches goroutine (`session.go:242-270`).

This is fine for Go-only engines/middleware, but requires strict JS boundary marshalling when JS callables are in play.

---

## 9) Fix Options

## Option A: Serialize entire async inference onto loop thread when JS callbacks are possible

Approach:

- In `runAsync`/`start`, if configuration includes JS-backed components, schedule whole `runSync` call through `RunOnLoop` instead of goroutine inference execution.

Pros:

- Simple mental model; no off-loop JS callback risk.
- Fastest patch for immediate correctness.

Cons:

- `runSync` can block for long periods (provider network, tool execution).
- Loop thread becomes blocked; JS responsiveness degrades.
- Streaming/event callbacks may stall behind long inference.

Usefulness:

- Good emergency safety patch, not ideal long-term UX.

## Option B (recommended): Keep inference async, but marshal every JS boundary call to loop thread synchronously

Approach:

- Introduce helper(s) that execute JS callbacks and JS value conversion on loop thread and synchronously return results/errors to caller goroutine.
- Replace all direct callable invocations in inference-time codepaths.

Pros:

- Preserves async inference and responsiveness.
- Enforces goja single-threaded runtime access.
- Works for `runAsync`, `start`, and any future async path.

Cons:

- Wider refactor than one-line fix.
- Needs careful deadlock handling and panic-to-error handling.

Usefulness:

- Best balance of correctness + behavior.

## Option C: Hard guard/fail-fast for async + JS-backed components

Approach:

- Detect JS engine/middleware/tool/hooks and reject `runAsync`/`start` with clear error until safe dispatcher exists.

Pros:

- Very safe and explicit.
- Minimal code change.

Cons:

- Regressive for users expecting async with JS callbacks.
- Does not solve deeper architecture.

Usefulness:

- Good temporary mitigation, especially if release pressure is high.

## Option D: Larger architectural shift to one runtime-owner actor

Approach:

- Move all runtime interaction behind dedicated owner goroutine/dispatcher abstraction.

Pros:

- Strong architecture for long-term safety.

Cons:

- Larger migration, likely beyond this ticket scope.

Usefulness:

- Good long-term direction; too heavy for immediate bug.

---

## 10) Recommended Plan (Pragmatic)

### Phase 1 (Immediate safety)

1. Apply Option C or minimal Option A as short-term guard.

- If JS-backed callbacks configured and safe dispatcher not enabled:
  - reject `runAsync/start` with actionable error, or
  - run whole inference on loop thread (documented blocking behavior).

2. Add targeted race test that reproduces current bug (expect fails before fix under `-race`, passes after).

### Phase 2 (Real fix)

Implement Option B dispatcher and remove temporary guard.

Core implementation tasks:

1. Add `moduleRuntime` helper for synchronous loop execution:

- `callOnLoopSync(fn func(*goja.Runtime) (T, error)) (T, error)` pattern (generic or typed variants).
- If already on loop goroutine (or loop unavailable), define explicit behavior to avoid deadlock.

2. Route all inference-time JS callable invocations through helper:

- `jsCallableEngine.RunInference`
- `jsMiddleware`
- JS tool handler wrapper
- tool hooks (`Before/After/OnError`)

3. Route async-path JS value conversion through helper where needed:

- event payload encoding in collector
- any `toJSValue/encodeTurnValue` usage executed from background goroutines

4. Keep promise settlement on loop (already done), but ensure no other runtime access happens outside loop.

5. Audit `start()` path in parallel with `runAsync()`; do not fix one without the other.

### Phase 3 (Stabilization)

1. Add stress/race test matrix.
2. Validate tool parallel settings with JS handlers/hooks.
3. Document safe async contract in JS API docs.

---

## 11) Proposed Detailed Technical Shape for Option B

A workable design without major architecture rewrite:

### 11.1 New helper API

At `moduleRuntime` layer, add methods conceptually like:

- `invokeJS(callable goja.Callable, args ...goja.Value) (goja.Value, error)`
- `toJSValueSafe(v any) (goja.Value, error)`

Behavior:

- If no loop configured: preserve current sync behavior (or return explicit error for async-only paths).
- If loop configured: schedule on `RunOnLoop`, wait on result channel.
- Recover panic from callback and convert to Go error where appropriate.

### 11.2 Replace direct callable usage

Replace these direct calls:

- `e.fn(...)`
- `fn(...)` in middleware
- `handler(...)` in tool registry wrapper
- `hooks.Before/After/OnError(...)`

with `invokeJS(...)` helper.

### 11.3 Encode payloads on loop

Where payload is currently built as JS value before loop scheduling (e.g., event collector), move conversion inside loop callback.

Current unsafe shape:

- `payload := c.encodeEvent(ev)` off-loop
- then `RunOnLoop(... cb(payload) ...)`

Safe shape:

- inside `RunOnLoop`: build payload JS value and then call listener.

### 11.4 Cancellation and deadlock notes

- Waiting goroutine must select on completion + context cancellation when applicable.
- If loop is stopped/terminated and cannot accept jobs, return deterministic error.
- Avoid calling synchronous loop helper from loop callback itself unless helper detects and executes inline.

---

## 12) Testing Strategy

Add tests that verify safety behavior, not just API shape.

### 12.1 Repro race test for async + JS engine callback

- Configure runtime with loop and `engines.fromFunction` JS callback.
- Use `runAsync` while also queueing additional loop work.
- Assert completion and run under `go test -race`.

### 12.2 Repro race test for `start()` + JS middleware/hook

- Use JS middleware/tool hooks that mutate state.
- Run async handle, subscribe events, concurrently schedule loop work.
- Under race detector, ensure no race reports.

### 12.3 Parallel tool scenario

- Configure multiple pending tool calls with JS handlers and `maxParallelTools > 1`.
- Validate deterministic completion and no race.

### 12.4 Event collector payload safety

- Ensure event publishing path does not construct goja values off-loop during async run.

### 12.5 Regression tests for non-JS paths

- Go-only engine/middleware/tools with async methods should keep throughput/behavior.

---

## 13) Answer to “Is this how go-go-goja/goja do it?”

Short answer:

- They do not execute VM interactions concurrently from background goroutines.
- They use event-loop marshalling (`RunOnLoop`) for VM interactions (especially promise resolve/reject and callback invocation).
- Geppetto’s current async inference path is only partially applying that pattern (promise settlement), but not applying it to all inference-time JS callback boundaries.

So current geppetto implementation is not fully aligned with the intended goja/goja_nodejs runtime-safety model when JS callbacks can execute during async inference.

---

## 14) Risk/Impact Assessment

Severity: High (P1 classification is justified).

Potential impacts:

- data races,
- sporadic panics,
- nondeterministic JS behavior,
- difficult-to-reproduce production instability under async load.

Most exposed usage:

- `runAsync/start` with `engines.fromFunction`, JS middleware, JS tool handlers, or JS tool hooks.

Less exposed:

- Go-only inference components (still should be verified, but primary bug path is JS-bound callbacks).

---

## 15) Proposed Immediate Ticket Actions

1. Add explicit subtask: “Unify JS-bound runtime access via loop dispatcher for async inference paths (`runAsync` + `start`).”
2. Add explicit subtask: “Move async event payload JS conversion to loop thread.”
3. Add explicit subtask: “Race test matrix for async JS callback paths (`-race`).”
4. If needed for near-term release, add temporary guard disabling async when JS-bound callbacks are configured until dispatcher lands.

---

## Appendix A: Key Code References

Geppetto:

- `pkg/js/modules/geppetto/api.go:464-503` (`runSync`, `runAsync`)
- `pkg/js/modules/geppetto/api.go:628-713` (`start` run-handle async path)
- `pkg/js/modules/geppetto/api.go:111-133` (`jsCallableEngine.RunInference`)
- `pkg/js/modules/geppetto/api.go:1598-1673` (`jsMiddleware`)
- `pkg/js/modules/geppetto/api.go:1755-1801` (JS tool handler invocation)
- `pkg/js/modules/geppetto/api.go:1080-1250` (JS tool hooks)
- `pkg/js/modules/geppetto/api.go:540-565` (event collector publish)
- `pkg/js/modules/geppetto/codec.go:405-449` (`toJSValue` VM object conversion)
- `pkg/inference/session/session.go:180-273` (`StartInference` goroutine execution)
- `pkg/inference/toolloop/enginebuilder/builder.go:158-208` (runner inference path)
- `pkg/inference/tools/base_executor.go:243-259` (parallel tool execution)
- `pkg/inference/tools/config.go:18-33` (default parallel tools = 3)

Upstream:

- `goja/README.md:99-103` (runtime not goroutine-safe)
- `goja/builtin_promise.go:606-628` (`NewPromise` warning + event-loop pattern)
- `goja/runtime.go:2430-2431` (concurrency safety note)
- `goja/typedarrays.go:101-102` (must not be called concurrently)
- `goja_nodejs/eventloop/eventloop.go:314-320` (`RunOnLoop`)
- `goja_nodejs/eventloop/eventloop.go:316-317` (runtime values stay in callback scope)
- `goja_nodejs/eventloop/eventloop_test.go:516-533` (canonical promise resolve on loop)

Comparison repo:

- `go-go-goja/engine/runtime.go:50-65` (synchronous runtime setup)
- `go-go-goja/modules/common.go:77-83` (module registration)
- `go-go-goja/README.md:137-203` (documented async pattern using loop marshalling)

---

## Appendix B: One-line Bottom Line

Fixing only the `go func(){ runSync(...) }` line in `runAsync` is not sufficient unless all inference-time JS callback boundaries are loop-serialized; otherwise the same class of goja runtime race remains (including in `start()`).
