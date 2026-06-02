---
Title: EventEmitter runAsync code review and intern guide
Ticket: GP-GOJA-STREAM-EVENTS-2026-06-01
Status: active
Topics:
    - geppetto
    - goja
    - js-bindings
    - streaming
    - events
DocType: analysis
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: cmd/examples/geppetto-js-run/main.go
      Note: example runner promise waiting review
    - Path: pkg/js/modules/geppetto/api_agent.go
      Note: runAsync implementation
    - Path: pkg/js/modules/geppetto/api_event_emitters.go
      Note: EventEmitter EventSink adapter and lifecycle review
    - Path: pkg/js/modules/geppetto/api_event_payloads.go
      Note: JavaScript event payload contract review
    - Path: pkg/js/modules/geppetto/provider/provider.go
      Note: xgoja provider integration gap review
    - Path: pkg/js/runtime/runtime.go
      Note: runtime jsevents manager installation and resolver wiring
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-02T10:25:50.44241158-04:00
WhatFor: ""
WhenToUse: ""
---


# EventEmitter `runAsync` Code Review and Intern Guide

## Executive summary

This document reviews the EventEmitter streaming work that added builder-level JavaScript EventEmitter support and the new `agent.runAsync(turn, options?)` method to Geppetto's go-go-goja JavaScript bindings. It is written for a new intern who needs to understand the architecture, how events travel through the system, what the implementation currently gets right, and where the design is still messy, under-tested, or incomplete.

The implemented public shape is intentionally small:

```js
const gp = require("geppetto");
const EventEmitter = require("events");

const events = new EventEmitter();
events.on("text-delta", ev => process.stdout.write(ev.delta));
events.on("inference-error", ev => console.error(ev.message || ev.error));

const agent = gp.agent()
  .inference(settings)
  .events(events)
  .build();

const handle = agent.runAsync(turn, { timeoutMs: 120000 });
const result = await handle.promise;
```

That shape is the right first-pass direction. It avoids the `handle.on(...)` late-listener race, keeps listeners registered before inference starts, and uses Geppetto's existing `events.EventSink` injection point. The implementation is also significantly better after the runner fix: async example scripts now surface missing-profile errors instead of exiting silently.

However, this code is not done. The most important issues are lifecycle management, host integration coverage, owner-thread safety inside `runAsync`, incomplete event payload coverage, lack of real-provider streaming validation, and weak documentation of what is guaranteed versus best-effort. The current work should be treated as a useful prototype that needs cleanup before it becomes a stable API contract.

## What changed

Relevant commits:

- `35c994e5` — `Add Geppetto JS EventEmitter runAsync`
- `e3a47348` — `Diary: record EventEmitter runAsync implementation`
- `0d16c88b` — `Document Geppetto JS runAsync event examples`
- `5ce221a5` — `Fix JS example runner promise waiting`

Major files:

| File | Role |
|---|---|
| `pkg/js/modules/geppetto/api_agent.go` | Adds `.events(...)` usage on agents and `agent.runAsync(...)`. |
| `pkg/js/modules/geppetto/api_event_emitters.go` | Wraps a go-go-goja `EventEmitter` as a Geppetto `events.EventSink`. |
| `pkg/js/modules/geppetto/api_event_payloads.go` | Encodes Geppetto event structs into JavaScript payload objects. |
| `pkg/js/modules/geppetto/api_events.go` | Keeps the older `gp.events.collector()` sink and now reuses shared payload encoding. |
| `pkg/js/runtime/runtime.go` | Installs `jsevents.Install()` and passes a typed resolver into the Geppetto module. |
| `pkg/js/modules/geppetto/module.go` | Adds typed EventEmitter manager fields to module options/runtime state. |
| `cmd/examples/geppetto-js-run/main.go` | Runs profile-backed JS examples and now waits for returned goja Promises. |
| `examples/js/geppetto/31_event_emitter_run_async.js` | Minimal real-provider EventEmitter + `runAsync` example. |
| `examples/js/geppetto/32_event_emitter_progress_summary.js` | Event counting/progress example. |
| `examples/js/geppetto/33_event_emitter_multiturn_run_async.js` | Multi-turn `runAsync` example using one builder-level emitter. |
| `pkg/doc/topics/13-js-api-reference.md` | Public JS reference for `runAsync` and EventEmitter payloads. |

## Architecture: the pieces an intern must understand

### 1. goja is single-owner-threaded

A goja runtime is not generally safe to manipulate from arbitrary goroutines. go-go-goja wraps a goja runtime in a `RuntimeOwner`. Code that needs to call JavaScript callbacks must schedule work onto that owner.

The important API idea is:

```go
owner.Post(ctx, "operation.name", func(ctx context.Context, vm *goja.Runtime) {
    // Safe to touch goja values and call JS functions here.
})
```

`agent.run(...)` is synchronous and blocks the caller until inference completes. If JavaScript calls `agent.run(...)`, the goja owner is occupied by the original JS-to-Go call while Go waits. That means JS EventEmitter callbacks cannot run live while `run(...)` is waiting. This is why `runAsync(...)` exists.

### 2. Geppetto events are published through `events.EventSink`

The Geppetto event subsystem has a simple sink abstraction. Downstream provider/tool-loop code publishes events to sinks stored in the run context.

Evidence:

- `pkg/events/context.go:15-24` attaches sinks to context.
- `pkg/events/context.go:40-49` publishes to each sink best-effort and ignores individual sink errors.

Conceptually:

```go
ctx = events.WithEventSinks(ctx, sinkA, sinkB)
...
events.PublishEventToContext(ctx, events.NewTextDeltaEvent(...))
```

### 3. Agent builders already had an event-sink slot

`gp.agent().events(...)` appends an `events.EventSink` to the builder. When the agent is built, those sinks are copied into `agentRef.eventSinks`.

Evidence:

- `api_agent.go:152-160` parses `.events(sink)` and appends it.
- `api_agent.go:198-208` copies builder sinks to the built agent.
- `api_agent.go:266-281` builds sessions with those sinks.

That is why builder-level EventEmitter support is a good first pass: it uses the existing sink path instead of inventing a parallel callback mechanism.

### 4. go-go-goja already has connected EventEmitters

The `go-go-goja/pkg/jsevents` package provides a manager that can adopt JavaScript-created EventEmitter objects and later emit into them from Go goroutines.

Evidence from go-go-goja:

- `pkg/jsevents/manager.go:112-121` adopts JS-created EventEmitters on the owner thread.
- `pkg/jsevents/manager.go:161-186` schedules `EmitterRef.EmitWithBuilder(...)` calls onto the runtime owner.
- `pkg/jsevents/manager.go:209-230` closes refs and unregisters them from the manager.

The Geppetto adapter in `api_event_emitters.go` stores a `*jsevents.EmitterRef`, implements `events.EventSink`, and calls `EmitWithBuilder`.

### 5. Runtime setup is split into module registration and runtime initialization

go-go-goja runtime modules register before runtime initializers run. The `jsevents.Install()` initializer creates the manager later. That is why the implementation uses a typed lazy resolver rather than reading the manager directly during module registration.

Evidence:

- `pkg/js/runtime/runtime.go:37-55` installs an `EventEmitterManagerResolver` closure that looks up `jsevents.RuntimeValueKey` later.
- `pkg/js/runtime/runtime.go:71-75` installs `jsevents.Install()` if no initializer with ID `jsevents.manager` is already present.
- `pkg/js/modules/geppetto/module.go:47-49` exposes the typed manager fields in `geppetto.Options`.
- `pkg/js/modules/geppetto/module.go:95-97` stores them in `moduleRuntime`.

This is better than exposing a generic `RuntimeValues map[string]any`, but it still couples Geppetto's core module directly to go-go-goja's `jsevents` package.

## Current event flow

```text
JavaScript setup
  ├─ const events = new EventEmitter()
  ├─ events.on("text-delta", callback)
  └─ gp.agent().inference(settings).events(events).build()
                         │
                         ▼
Geppetto JS module
  ├─ agent().events(value)
  ├─ requireEventSink(value)
  ├─ newEventEmitterSinkFromValue(value)
  └─ jsevents.Manager.AdoptEmitterOnOwner(value)
                         │
                         ▼
Built agent
  └─ agentRef.eventSinks contains jsEventEmitterSink
                         │
                         ▼
agent.runAsync(turn)
  ├─ creates goja Promise + handle
  ├─ starts goroutine
  ├─ buildSession() injects event sinks
  ├─ StartInference(ctx)
  └─ wait for ExecutionHandle
                         │
                         ▼
Provider/tool loop
  └─ events.PublishEventToContext(ctx, ev)
                         │
                         ▼
jsEventEmitterSink.PublishEvent(ev)
  ├─ encodeGeppettoEventPayload(ev)
  ├─ emit "event"
  └─ emit type-specific name such as "text-delta"
                         │
                         ▼
go-go-goja EventEmitter
  └─ owner.Post(...) invokes JS listeners on the goja owner
```

## Public API contract as currently documented

### Agent construction

```ts
interface AgentBuilder {
  events(sink: EventEmitterLike | any): AgentBuilder;
  build(): Agent;
}
```

The `any` part still exists because Go sinks can be passed as host-provided refs. For normal JavaScript code, the intended value is a go-go-goja EventEmitter from `require("events")`.

### Async execution

```ts
interface Agent {
  run(turn: TurnWrapper, options?: RunOptions): RunResult;
  runAsync(turn: TurnWrapper, options?: RunOptions): AgentAsyncHandle;
}

interface AgentAsyncHandle {
  promise: Promise<RunResult>;
  cancel(): void;
  close(): void;
}

interface RunOptions {
  timeoutMs?: number;
  tags?: Record<string, any>;
}
```

Intentionally absent:

- `agent.stream(...)`
- `agent.runAsync(turn, { events })`
- `handle.on(...)`

That absence is a good design decision for now. It avoids making late listener registration look safe when it is not.

## What is good about the implementation

### The public API is small and coherent

The final API uses one event registration mechanism and one async execution method. This is much easier to teach and test than the earlier mixture of `stream`, per-run emitters, and `handle.on`.

Good decision:

```js
const events = new EventEmitter();
events.on("text-delta", onDelta);
const agent = gp.agent().events(events).build();
const handle = agent.runAsync(turn);
```

Avoided bad first-pass decision:

```js
const handle = agent.runAsync(turn);
handle.on("text-delta", onDelta); // racy if run has already started
```

### It reuses Geppetto's existing event-sink path

The builder stores sinks and session construction passes those sinks to the inference engine. This avoids a separate JavaScript-only event path that would need to be kept in sync with the Go event system.

Evidence:

- `api_agent.go:152-160`: `.events(...)` appends the sink.
- `api_agent.go:266-281`: `buildSession()` forwards the sink list.

### It uses go-go-goja's connected-emitter model

The adapter does not call JavaScript callbacks directly from provider goroutines. Instead, `PublishEvent` calls `EmitterRef.EmitWithBuilder`, which schedules delivery on the runtime owner.

Evidence:

- `api_event_emitters.go:52-69`: `PublishEvent` uses `EmitWithBuilder`.
- go-go-goja `manager.go:161-186`: `EmitWithBuilder` uses `owner.Post`.

### It fixed the silent example problem

The example runner now exports a returned JS Promise on the owner thread and waits for it. Missing profile errors are now visible as `script promise rejected: GoError: profile not found` instead of appearing as zero output.

Evidence:

- `cmd/examples/geppetto-js-run/main.go:134-143`: converts a returned Promise value with `value.Export().(*goja.Promise)`.
- `cmd/examples/geppetto-js-run/main.go:148-196`: polls the promise state and surfaces rejection.

## Findings: design and implementation risks

### Finding 1 — High: builder-level EventEmitter refs are never closed

`jsEventEmitterSink` has a `Close(ctx)` method, and go-go-goja `EmitterRef.Close` unregisters the ref from the manager. But built agents do not expose `agent.close()`, and `runAsync` does not close builder-level sinks after a run. That means every builder-level EventEmitter adoption can leave a manager ref alive until the whole runtime closes.

Evidence:

- `api_event_emitters.go:72-83` implements `Close`.
- `api_agent.go:198-208` copies sinks into `agentRef`.
- `api_agent.go:212-248` exposes only `name`, `run`, and `runAsync`; no `close` method.
- go-go-goja `manager.go:209-230` shows `Close` is what unregisters refs from the manager map.

Why this matters:

- Long-lived runtimes that create many agents can accumulate connected emitter refs.
- JavaScript can remove listeners, but the Go manager still retains the adopted emitter ref unless `Close` is called.
- The API reference says the current lifetime is builder/agent-level, but no explicit release mechanism exists.

Recommended fix:

```go
type agentRef struct {
    eventSinks []events.EventSink
    closeOnce sync.Once
}

func (a *agentRef) close(ctx context.Context) error {
    var ret error
    a.closeOnce.Do(func() {
        for _, sink := range a.eventSinks {
            if closer, ok := sink.(interface{ Close(context.Context) error }); ok {
                if err := closer.Close(ctx); err != nil && ret == nil {
                    ret = err
                }
            }
        }
    })
    return ret
}
```

Expose it in JavaScript:

```js
const agent = gp.agent().events(events).build();
try {
  const result = await agent.runAsync(turn).promise;
} finally {
  agent.close();
}
```

If `agent.close()` is too much API surface, register a runtime closer that closes module-created `jsEventEmitterSink` refs when the runtime shuts down. That still does not solve mid-runtime churn, but it prevents final leaks.

### Finding 2 — High: `runAsync` builds sessions on a background goroutine, but some session-building code touches goja state

`agent.runAsync` starts a goroutine and calls `a.startRun(input, opts)` inside that goroutine. `startRun` calls `a.buildSession()`. Most of `buildSession` is Go-only, but one branch calls `a.api.vm.ToValue(a.loopOptions)`.

Evidence:

- `api_agent.go:384-400`: goroutine calls `a.startRun(...)` and waits.
- `api_agent.go:291-314`: `startRun` calls `a.buildSession()`.
- `api_agent.go:278-280`: `buildSession()` calls `a.api.vm.ToValue(a.loopOptions)` when loop options are present.

Why this matters:

- goja runtimes must be touched on the owner thread.
- This path is probably untested with `.toolLoop(...)` plus `runAsync`.
- Similar risks may exist if later builder options call helpers that assume owner-thread execution.

Recommended fix:

Split session preparation from inference waiting:

```go
func (a *agentRef) startAsync(input *turns.Turn, opts runOptions) goja.Value {
    // Still on owner thread here.
    started, startErr := a.startRun(input, opts)
    if startErr != nil { return rejectedPromise(startErr) }

    go func() {
        out, err := started.handle.Wait()
        settlePromiseOnOwner(out, err)
    }()

    return handleObj
}
```

If `StartInference` itself is fast and only starts a goroutine, doing setup on the owner thread is acceptable. If it can block, split `prepareRunOnOwner` from `startInferenceInBackground`, but keep all `goja.Value` conversion on owner.

Minimum tests to add:

- `agent.toolLoop({...}).runAsync(turn)` does not panic/race.
- Run with Go race detector if feasible: `go test -race ./pkg/js/modules/geppetto -run TestAgentRunAsync`.

### Finding 3 — High: xgoja provider integration likely does not wire the EventEmitter manager

`pkg/js/runtime.NewRuntime` wires `EventEmitterManagerResolver`. The xgoja provider path returns `geppettomodule.NewLoader(opts)` using options from host services and config. It does not itself receive a go-go-goja `RuntimeModuleContext`, so it cannot install the resolver the same way.

Evidence:

- `pkg/js/runtime/runtime.go:37-55` wires the resolver in the custom runtime path.
- `pkg/js/modules/geppetto/provider/provider.go:59-72` returns `geppettomodule.NewLoader(opts)` from provider host options and config.
- `pkg/js/modules/geppetto/provider/provider.go:22-24` only requires `GeppettoOptions(...)`; there is no EventEmitter manager contract.

Why this matters:

- Examples run through `cmd/examples/geppetto-js-run`, which uses `pkg/js/runtime.NewRuntime`, so they work there.
- Generated xgoja hosts using the provider module may expose `require("geppetto")` without `EventEmitterManagerResolver`; then `.events(new EventEmitter())` fails with `geppetto events: jsevents manager is not installed`.
- This is exactly the kind of host-path mismatch that becomes confusing for users: examples work, generated app fails.

Recommended options:

1. Extend provider `HostServices` with an optional typed interface:

```go
type EventEmitterManagerProvider interface {
    EventEmitterManagerResolver() func() (*jsevents.Manager, bool)
}
```

2. Or change provider API integration so module factories receive runtime values/initializers in a typed way.
3. Or document clearly that xgoja hosts must inject `EventEmitterManager` or `EventEmitterManagerResolver` through `GeppettoOptions`.

Minimum test:

- Provider-created Geppetto module in a go-go-goja runtime with `require("events")` and `jsevents.Install()` can call `.events(new EventEmitter())` successfully.

### Finding 4 — Medium: event sink errors are best-effort and currently disappear

`PublishEventToContext` ignores individual sink errors. `jsEventEmitterSink.PublishEvent` returns errors when `EmitWithBuilder` fails, but callers usually do not see them.

Evidence:

- `pkg/events/context.go:40-49`: `_ = sink.PublishEvent(event)`.
- `api_event_emitters.go:63-67`: `EmitWithBuilder` errors are returned.

Why this matters:

- If the runtime is closed, the emitter ref is closed, or scheduling is rejected, event delivery may silently stop.
- JavaScript code may observe no events and no error.
- This looks like a provider streaming issue even when the real problem is sink delivery.

Recommended fix:

- Keep inference best-effort, but add observability.
- Wrap JS sinks with logging on publish failure.
- Count failed emissions in a debug field or internal metric.

Pseudocode:

```go
func (s *jsEventEmitterSink) PublishEvent(ev events.Event) error {
    if err := s.emit(ev); err != nil {
        s.api.logger.Warn().Err(err).Str("event", string(ev.Type())).Msg("js event emit failed")
        return err
    }
    return nil
}
```

If `PublishEventToContext` remains best-effort, the sink must log before returning.

### Finding 5 — Medium: event payload coverage is incomplete and hand-maintained

`encodeGeppettoEventPayload` maps common text/reasoning/tool/error events, but many canonical events only get the generic fields. For example, run lifecycle events, provider metadata updates, provider call finished fields, web search events, citation events, and file search events do not get rich payloads.

Evidence:

- `api_event_payloads.go:33-96` lists the currently mapped concrete event types.
- `pkg/events/chat-events.go` defines many more canonical event constants beyond the mapped set.
- `api_event_payloads.go:103-116` emits all event types by name, even when payload detail is missing.

Why this matters:

- Docs may imply event payloads are stable and rich, but many event types will only have `type`, ids, metadata, correlation, and raw payload.
- New event structs can be added without updating JS payload encoding.
- Tests cover only `text-delta`, `tool-result-ready`, and `error` mapping.

Recommended fix:

Add a payload contract table generated or tested against the canonical event registry. At minimum, add tests for:

- `provider-call-started`
- `provider-call-metadata-updated`
- `provider-call-finished`
- `run-started`
- `run-finished`
- `run-failed`

Possible long-term design:

```go
type JSEventPayloadEncoder interface {
    EncodeJSEventPayload() map[string]any
}
```

Then event structs can own their JavaScript payload conversion, or a central registry can map event types to encoder functions.

### Finding 6 — Medium: no `runAsync` lifecycle events are emitted by the adapter

The design originally discussed lifecycle events such as `stream-start`, `stream-result`, `stream-error`, and `stream-close`. The final implementation does not emit lifecycle events from `runAsync`; it only forwards whatever the provider/tool loop publishes.

Evidence:

- `api_agent.go:384-416` starts, waits, and settles the promise without publishing adapter lifecycle events.
- `api_event_payloads.go` has no adapter lifecycle event types.

Why this matters:

- If an engine/provider emits no Geppetto events, JavaScript sees no events until the promise resolves.
- Users cannot reliably show "run started" / "run closed" progress from the JS adapter itself.
- Examples rely on provider lifecycle events like `provider-call-started`, but those are provider/tool-loop events, not adapter-level guarantees.

Recommendation:

Add adapter lifecycle events only after deciding names and payload stability. Good first choice:

- `runasync-start`
- `runasync-result`
- `runasync-error`
- `runasync-close`

Avoid using `stream-*` names now that the API is `runAsync`.

### Finding 7 — Medium: `runAsync` rejection values are strings, not JavaScript Error objects

When startup or wait fails, the code rejects the Promise with `err.Error()` converted to a JS value.

Evidence:

- `api_agent.go:390-392`: `reject(a.api.vm.ToValue(err.Error()))`.
- `api_agent.go:401-404`: `reject(a.api.vm.ToValue(waitErr.Error()))`.

Why this matters:

- JS callers get a string, not an `Error` with `message`, `stack`, `name`, or structured cause.
- Examples use `String(err)`, which hides the problem.
- Better JS APIs reject with Error objects.

Recommended fix:

```go
func (m *moduleRuntime) newJSError(err error) goja.Value {
    if err == nil { return goja.Undefined() }
    return m.vm.NewGoError(err) // or construct Error with extra fields
}
```

Then reject with the error value on owner:

```go
_ = reject(a.api.newJSError(err))
```

### Finding 8 — Medium: `runAsync` uses `context.Background()` rather than runtime lifetime context

Run contexts are derived from `context.Background()` in `sessionRef.buildRunContext`.

Evidence:

- `api_sessions.go:59-76`: `ctx := context.Background()`.
- `api_agent.go:390` and `api_agent.go:401` post promise settlement with `context.Background()`.
- `api_event_emitters.go:63` emits with `context.Background()`.

Why this matters:

- Closing the runtime does not necessarily cancel in-flight provider work unless the provider itself observes another cancellation path.
- Event emission may be attempted after runtime close; this probably returns errors, but those are best-effort and may be swallowed.
- Long-running provider calls can outlive the JS runtime that initiated them.

Recommended fix:

Use the runtime lifetime context available through go-go-goja runtime services, or add it to `moduleRuntime` options. Build run context from runtime lifetime plus run timeout:

```go
base := m.runtimeLifetimeContext
if base == nil { base = context.Background() }
ctx := base
if timeoutMs > 0 { ctx, cancel = context.WithTimeout(ctx, timeout) }
```

### Finding 9 — Medium: `runAsync` can panic if an engine returns `(nil, nil)`

`startAsync` clones `out` when resolving. There is no nil check.

Evidence:

- `api_agent.go:400`: `out, waitErr := started.handle.Wait()`.
- `api_agent.go:406-411`: `outputTurn: out.Clone()`.

Why this matters:

- Most engines should return a turn or an error, but defensive code should not panic in the owner-post callback.
- A panic in the owner-post callback may be swallowed or reported poorly depending on runtime owner recovery.

Recommended fix:

```go
if waitErr == nil && out == nil {
    waitErr = fmt.Errorf("agent.runAsync: engine returned nil turn and nil error")
}
```

Add the same check to `runSync`.

### Finding 10 — Medium: EventEmitter delivery ordering and duplication semantics are not specified

For each Geppetto event, `jsEventEmitterSink.PublishEvent` emits two EventEmitter notifications: generic `event` and type-specific event name.

Evidence:

- `api_event_emitters.go:59-68`: loops over `eventEmitterNamesForPayload(payload)`.
- `api_event_payloads.go:103-116`: returns `[]string{"event", eventType}`.

Why this matters:

- Generic and type-specific listeners see the same payload object cloned separately.
- Ordering across different Geppetto events depends on provider goroutine scheduling and owner queue ordering.
- Tests only verify a simple single-event order.

Recommended documentation:

- For a single `PublishEvent` call, Geppetto attempts `event` first and the type-specific name second.
- Across concurrent publishers, ordering is best-effort and should not be used as a strict clock.
- Use `sequence`, `correlation`, and provider ids when ordering matters.

Recommended tests:

- Multiple sequential deltas preserve `sequence` and observed order in a single publisher.
- Two different event sinks both receive events.

### Finding 11 — Low/Medium: `EventEmitterManagerResolver` is typed but still awkward API surface

The move away from `RuntimeValues map[string]any` was correct. But `geppetto.Options` now directly imports `go-go-goja/pkg/jsevents` and exposes both a concrete manager and a resolver.

Evidence:

- `module.go:47-49`: `EventEmitterManager` and `EventEmitterManagerResolver`.
- `runtime.go:46-53`: resolver closure.

Why this matters:

- Core Geppetto JS module options are now coupled to go-go-goja's `jsevents` package.
- Two ways to configure the same dependency can confuse host implementers.
- The resolver exists only because of initialization order; without a comment, future maintainers may not understand it.

Recommendation:

Add explicit comments:

```go
// EventEmitterManager is used when a host can provide the connected-emitter
// manager directly. EventEmitterManagerResolver is used by go-go-goja runtime
// modules because runtime initializers install the manager after module
// registration. Prefer EventEmitterManager when available; otherwise use the resolver.
```

Longer-term: define a small local interface instead of exposing `*jsevents.Manager` if Geppetto wants to reduce dependency coupling.

### Finding 12 — Low/Medium: `gp.events.collector()` is now legacy-ish and under-positioned

The older `gp.events.collector()` remains. It implements `events.EventSink` and uses Go-owned JS callback lists. It is no longer the recommended streaming API, but it is still exported.

Evidence:

- `api_events.go` still implements `eventsCollector`.
- `module.go` still exports `events.collector`.
- New docs prefer EventEmitter, but the old collector is not clearly marked as lower-level/legacy.

Why this matters:

- Users may choose collector and then wonder why examples use EventEmitter.
- Collector callbacks and EventEmitter callbacks have different semantics and lifecycle behavior.

Recommendation:

Document `gp.events.collector()` as:

- internal/advanced Go `EventSink` interop;
- not the recommended JS streaming interface;
- subject to deprecation once EventEmitter coverage is complete.

### Finding 13 — Low: examples are useful but not automated against real providers

The example scripts are good manual smoke tests, but no automated test runs them against a provider registry.

Evidence:

- `examples/js/geppetto/31_event_emitter_run_async.js`
- `examples/js/geppetto/32_event_emitter_progress_summary.js`
- `examples/js/geppetto/33_event_emitter_multiturn_run_async.js`

Why this matters:

- The earlier zero-output issue was only discovered manually.
- Provider behavior differs; real-provider examples should at least be occasionally run and observed.

Recommendation:

Add a manual smoke target:

```bash
make smoke-js-events PROFILE_REGISTRIES="$HOME/.config/pinocchio/profiles.yaml" PROFILE=default
```

Or add a script:

```bash
examples/js/geppetto/run_event_emitter_examples.sh
```

The script should run `31`, `32`, and optionally `33`, and should fail if there is no final JSON output.

## Deprecated or confusing code inventory

### `agent.stream(...)`

Status: removed from current public API and TypeScript declarations.

Keep it removed. Do not reintroduce it as an alias unless there is a migration requirement. If an alias is added, it must be documented as deprecated and tested to behave identically to `runAsync`.

### `handle.on(...)`

Status: not exposed by `runAsync`.

Keep it absent. It is racy unless the implementation uses a prepared-but-not-started handle or attaches an internal emitter before returning.

### `agent.runAsync(turn, { events })`

Status: intentionally deferred.

Do not add this until there is a clear need for per-run isolation. If it is added, `buildSession(extraSinks...)` must be refactored carefully and all EventEmitter adoption/closing must happen on the owner thread.

### `gp.events.collector()`

Status: still exported.

Treat it as advanced/legacy interop. Prefer EventEmitter in examples and docs. Consider adding a warning section in the API reference.

## Missing functionality checklist

Required before considering this stable:

- [ ] Add `agent.close()` or equivalent sink lifecycle cleanup.
- [ ] Ensure all `runAsync` session preparation that touches goja runs on the owner thread.
- [ ] Wire EventEmitter manager support for xgoja provider-created modules, or document exactly how hosts must provide it.
- [ ] Improve Promise rejection values from strings to JavaScript `Error` objects.
- [ ] Use runtime lifetime context instead of `context.Background()` for runs and event posts where possible.
- [ ] Add nil-output defensive checks for `run` and `runAsync`.
- [ ] Add richer event payload mapping for provider lifecycle and run lifecycle events.
- [ ] Add tests for provider lifecycle payloads, ordering assumptions, manager-missing failures, xgoja provider integration, and `toolLoop(...).runAsync(...)`.
- [ ] Add a manual smoke script for real-provider EventEmitter examples.

Nice-to-have after stabilization:

- [ ] Per-run EventEmitter sinks: `agent.runAsync(turn, { events })`.
- [ ] Adapter lifecycle events with `runasync-*` names.
- [ ] Structured event TypeScript union types.
- [ ] Metrics/logging for dropped EventEmitter emissions.

## Suggested cleanup plan

### Phase 1: Safety and lifecycle

Goal: prevent leaks and owner-thread misuse.

Tasks:

1. Add comments to `EventEmitterManager` and `EventEmitterManagerResolver` explaining initialization order.
2. Add `agent.close()` or runtime closer-backed sink cleanup.
3. Move run preparation that may touch goja onto the owner thread.
4. Replace `context.Background()` in run contexts with runtime lifetime context where feasible.
5. Add nil-output defensive checks.

Validation:

```bash
go test ./pkg/js/modules/geppetto -run 'TestAgentRunAsync|TestEventEmitterSink' -count=1
go test -race ./pkg/js/modules/geppetto -run TestAgentRunAsync -count=1
```

### Phase 2: Host integration

Goal: make examples, runtime.NewRuntime, and xgoja provider paths consistent.

Tasks:

1. Add a provider integration test that creates a provider-backed Geppetto module and calls `.events(new EventEmitter())`.
2. Add optional provider host service interface for EventEmitter manager/resolver if needed.
3. Document host requirements for EventEmitter support.

Validation:

```bash
go test ./pkg/js/modules/geppetto/provider -count=1
GOWORK=off go test ./pkg/js/modules/geppetto/provider -count=1
```

### Phase 3: Payload contract hardening

Goal: make event payload behavior predictable and durable.

Tasks:

1. Add table-driven tests for every canonical event type currently expected by JS users.
2. Add provider lifecycle payload fields (`usage`, `stopReason`, `durationMs`, etc.).
3. Decide whether event payload keys are camelCase, snake_case, or mixed; normalize if possible.
4. Generate or validate TypeScript event payload declarations.

Validation:

```bash
go test ./pkg/js/modules/geppetto -run TestEventPayload -count=1
```

### Phase 4: Documentation and smoke tests

Goal: avoid repeating the zero-output confusion.

Tasks:

1. Add `examples/js/geppetto/run_event_emitter_examples.sh`.
2. Update docs with a troubleshooting section:
   - missing profile => visible `profile not found` error;
   - no deltas => provider may not stream text deltas;
   - no events at all => verify EventEmitter manager support and default modules.
3. Run real-provider smoke and record observed event types in ticket diary.

Validation:

```bash
go run ./cmd/examples/geppetto-js-run run \
  --script examples/js/geppetto/32_event_emitter_progress_summary.js \
  --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
  --profile default \
  --timeout-ms 120000
```

## Intern-oriented API reference

### Minimal usage

```js
const gp = require("geppetto");
const EventEmitter = require("events");

const settings = gp.inferenceProfiles.resolve("default");
const events = new EventEmitter();

events.on("event", ev => console.log("generic", ev.type));
events.on("text-delta", ev => console.log("delta", ev.delta));
events.on("inference-error", ev => console.error(ev.message));

const agent = gp.agent()
  .inference(settings)
  .events(events)
  .build();

const turn = gp.turn().user("Hello").build();
const handle = agent.runAsync(turn, { timeoutMs: 120000 });
const result = await handle.promise;

console.log(result.text());
```

### Cancellation

```js
const handle = agent.runAsync(turn, { timeoutMs: 120000 });

// Later:
handle.cancel(); // also available as handle.close()

try {
  await handle.promise;
} catch (err) {
  console.error("run was canceled or failed", err);
}
```

### Events to listen for

Start with these:

```js
events.on("event", ev => counts[ev.type] = (counts[ev.type] || 0) + 1);
events.on("provider-call-started", ev => console.log("provider started", ev.inferenceId));
events.on("provider-call-finished", ev => console.log("provider finished", ev.inferenceId));
events.on("text-delta", ev => process.stdout.write(ev.delta));
events.on("tool-result-ready", ev => console.log(ev.toolResult));
events.on("inference-error", ev => console.error(ev.message || ev.error));
```

Do not assume `text-delta` always fires. Some providers only produce final text.

## Review conclusion

The work is directionally correct: builder-level EventEmitter registration plus `runAsync` is the right first stable shape. It avoids the known `handle.on` race, uses Geppetto's existing event-sink abstraction, and integrates with go-go-goja's owner-thread-safe EventEmitter manager.

The main problems are not the basic API shape; they are lifecycle, host integration, and hardening. Before this should be treated as stable API surface, the next engineer should close builder-level emitter refs, make `runAsync` session preparation owner-thread-safe, prove xgoja provider integration, harden event payload contracts, and add a real-provider smoke script so silent failures do not recur.
