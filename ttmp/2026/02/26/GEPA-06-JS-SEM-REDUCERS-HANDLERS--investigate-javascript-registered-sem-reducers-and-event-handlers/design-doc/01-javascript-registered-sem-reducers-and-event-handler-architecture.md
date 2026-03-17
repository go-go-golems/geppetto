---
Title: JavaScript-Registered SEM Reducers and Event Handler Architecture
Ticket: GEPA-06-JS-SEM-REDUCERS-HANDLERS
Status: active
Topics:
    - gepa
    - event-streaming
    - js-vm
    - sem
    - pinocchio
    - geppetto
    - go-go-os
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/runtime/registerChatModules.ts
      Note: Runtime JS module registration surface for chat extension
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts
      Note: |-
        Frontend SEM handler registration and dispatch semantics
        Frontend SEM handler map behavior and default llm.delta handling
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/state/timelineSlice.ts
      Note: Timeline reducer semantics and version merge behavior
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/go-go-os/packages/engine/src/plugin-runtime/stack-bootstrap.vm.js
      Note: |-
        Current sandbox plugin host API; no SEM subscription surface
        Sandbox plugin runtime API surface lacks SEM subscription
    - Path: geppetto/pkg/events/chat-events.go
      Note: |-
        Canonical geppetto event types (start/partial/final etc.)
        Canonical geppetto event type taxonomy
    - Path: geppetto/pkg/js/modules/geppetto/api_events.go
      Note: |-
        JS event collector sink, wildcard support, payload mapping
        JS event collector sink and wildcard semantics
    - Path: geppetto/pkg/js/modules/geppetto/api_sessions.go
      Note: |-
        JS run handle and on(eventType, callback) subscription model
        JavaScript run handle and event subscription API
    - Path: go-go-gepa/cmd/gepa-runner/plugin_loader.go
      Note: |-
        GEPA-04 plugin event hook options.emitEvent/options.events.emit
        GEPA-04 plugin event hook injection
    - Path: go-go-gepa/cmd/gepa-runner/plugin_stream.go
      Note: |-
        GEPA stream CLI output wrapper (stream-event)
        GEPA stream-event CLI emission
    - Path: go-go-gepa/pkg/dataset/generator/plugin_loader.go
      Note: GEPA-04 dataset generator event hook path
    - Path: go-go-gepa/pkg/jsbridge/emitter.go
      Note: |-
        GEPA plugin event envelope model (plugin_event)
        GEPA plugin event envelope structure
    - Path: go-go-gepa/ttmp/2026/02/26/GEPA-06-JS-SEM-REDUCERS-HANDLERS--investigate-javascript-registered-sem-reducers-and-event-handlers/scripts/js-sem-reducer-handler-prototype.js
      Note: Prototype for reducer/handler composition semantics
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Backend translator registration and mapping
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: |-
        SEM pass-through and geppetto-event-to-SEM translation boundary
        SEM pass-through and translation boundary
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: |-
        Backend SEM projection to TimelineEntityV2
        Backend SEM projection logic and built-in llm handlers
    - Path: pinocchio/pkg/webchat/timeline_registry.go
      Note: Backend timeline handler registration contract
ExternalSources: []
Summary: Exhaustive architecture study of JavaScript-registered SEM reducers and JavaScript event handlers across geppetto, pinocchio, and go-go-os, including current capability, layer boundaries, and implementation roadmap.
LastUpdated: 2026-02-26T18:23:00-05:00
WhatFor: Determine where SEM projection belongs, what JS handler capabilities already exist, and what to build next
WhenToUse: Use when implementing JS-driven event reactions or JS-defined SEM projection/reducer behavior
---












# JavaScript-Registered SEM Reducers and Event Handler Architecture

## 1. Executive Summary

This document answers your two target capabilities:

1. register JavaScript SEM reducers/projection code,
2. register JavaScript event handlers that react to `llm.delta` or other SEM/geppetto events.

It also answers the architecture boundary question:

> Is SEM projection done in `pinocchio` or in `geppetto`?

### Short answer

1. **SEM projection is primarily in `pinocchio` backend and frontend clients (`go-go-os` / `pinocchio` web TS), not in `geppetto`.**
2. **`geppetto` already supports JavaScript event handlers for its own event stream** (`start`/`partial`/`final`, tool events, etc.) through the JS run handle `on(eventType, callback)` API.
3. **Backend JavaScript-registered SEM reducers in `pinocchio` do not exist today.** Backend reducer/projection handlers are Go functions.
4. **Frontend JavaScript SEM registration does exist today** (runtime TS/JS modules can call `registerSem(...)`, register renderers, dispatch reducers).
5. **GEPA-04 already added plugin stream events in `go-go-gepa`,** but those are currently plugin-stream envelopes (`plugin_event`) and CLI streaming, not automatic SEM projection.

> [!IMPORTANT]
> The missing piece for your request is a server-side JS reducer/handler runtime in `pinocchio` (or an equivalent dynamic extension path), plus explicit bridge decisions from GEPA plugin events into SEM/timeline conventions.

---

## 2. Direct Answers to Your Questions

## 2.1 Can we register JavaScript SEM reducers/projection code now?

1. **Backend (`pinocchio`): no** (current registration APIs are Go-only handler functions).
2. **Frontend (`go-go-os` app bundle JS/TS): yes** (register SEM handlers + dispatch state updates).
3. **Frontend sandbox plugins (`QuickJS` stack bundle): no** (no SEM subscription API exposed there yet).

## 2.2 Can we register JavaScript event handlers to react to `llm.delta` / SEM?

1. **Geppetto JS runtime: yes for geppetto event type `partial` (delta-equivalent).**
2. **Literal geppetto event type `llm.delta`: no** in geppetto core (that name is SEM-side convention, mostly downstream of Pinocchio translation).
3. **Go-go-os SEM handler registration: yes** for `llm.delta` and any custom SEM event type in app runtime modules.
4. **Pinocchio backend JS handlers for SEM: no today.**

## 2.3 Is projection in pinocchio or geppetto?

1. **Geppetto**: produces runtime events (`start`, `partial`, `final`, etc.) and supports event sinks.
2. **Pinocchio**: converts incoming event stream to SEM and performs backend timeline projection.
3. **Go-go-os / pinocchio web frontend**: performs client-side SEM handling and reducer dispatch for UI state.

So the practical answer is: **projection is in Pinocchio and frontend runtimes; Geppetto is upstream event producer infrastructure.**

---

## 3. Fundamentals and Layer Map

## 3.1 Terminology

1. **Geppetto event**: typed runtime event in `geppetto/pkg/events` (`start`, `partial`, `final`, tool events, logs).
2. **SEM event**: normalized semantic envelope used by chat stream/timeline/websocket APIs (`llm.delta`, `timeline.upsert`, etc.).
3. **Projection/reducer**: logic that maps event stream into timeline entities / UI state.
4. **Handler**: callback reacting to event side effects.

## 3.2 Layered architecture

```mermaid
flowchart TD
  A[Geppetto Engines] --> B[Geppetto events.Event stream]
  B --> C[Pinocchio StreamCoordinator]
  C --> D[SEM envelopes]
  D --> E[Pinocchio TimelineProjector]
  D --> F[WebSocket clients]
  E --> G[/api/timeline snapshots]
  F --> H[go-go-os WsManager]
  H --> I[go-go-os semRegistry handlers]
  I --> J[Redux timeline/chat reducers]
```

### Interpretation

1. `geppetto` is event-source and sink/subscription infrastructure.
2. `pinocchio` is the central SEM normalization + backend projection layer.
3. `go-go-os` is client projection/render layer with runtime JS registration support.

---

## 4. Current-State Evidence by Layer

## 4.1 Geppetto: JS handlers for geppetto events already exist

## 4.1.1 Event type model

Geppetto canonical LLM stream events are:

1. `start`
2. `partial`
3. `final`

Evidence: `chat-events.go` (`EventTypeStart`, `EventTypePartialCompletion`, `EventTypeFinal`).

## 4.1.2 JS event callback API

`session.start(...)` returns handle with:

1. `promise`
2. `cancel()`
3. `on(eventType, callback)`

Evidence: `api_sessions.go` around handle object construction and `on(...)` registration.

## 4.1.3 Collector semantics

`jsEventCollector` implements `events.EventSink` and dispatches:

1. exact event-type listeners,
2. wildcard `*` listeners,
3. payload enrichment for partial/tool/final/error.

Evidence: `api_events.go`.

### Practical implication

`geppetto` already has JavaScript event reaction capability; what it does not do is expose Pinocchio SEM projections directly.

---

## 4.2 Pinocchio: backend SEM translation and timeline projection are Go-based

## 4.2.1 Stream -> SEM pipeline

`StreamCoordinator`:

1. accepts already-SEM envelopes,
2. or decodes geppetto JSON and translates to SEM.

Evidence: `stream_coordinator.go` around payload detection/translation paths.

## 4.2.2 Timeline projection

`TimelineProjector.ApplySemFrame`:

1. validates SEM envelope shape,
2. executes custom timeline handlers,
3. falls back to built-in event-type switch (`llm.start`, `llm.delta`, `llm.final`, etc.),
4. upserts `TimelineEntityV2`.

Evidence: `timeline_projector.go`.

## 4.2.3 Registration interfaces are Go-only

Backend extension points:

1. `RegisterTimelineHandler(eventType string, handler TimelineSemHandler)`
2. `semregistry.RegisterByType(eventType, handler)`

Both accept Go function types. There is no JS runtime loader for these server-side handlers currently.

Evidence: `timeline_registry.go`, `registry.go`, startup registrations in `cmd/web-chat/main.go`.

---

## 4.3 Go-go-os: frontend SEM registration exists, with caveats

## 4.3.1 SEM registry behavior

`semRegistry.ts`:

1. `registerSem(type, handler)` writes into map,
2. `handleSem` resolves exact type and invokes one handler.

Evidence: `semRegistry.ts` lines around `handlers` map and `handleSem`.

## 4.3.2 Defaults include llm.delta

`registerDefaultSemHandlers()` registers:

1. `timeline.upsert` projection,
2. `llm.start`, `llm.delta`, `llm.final`, etc.

Evidence: `semRegistry.ts` default registration block.

## 4.3.3 Runtime JS extensibility exists for app-bundle modules

`registerChatRuntimeModule(...)` can register handlers/renderers/normalizers at runtime.

Evidence: `registerChatModules.ts`, inventory app boot wiring.

## 4.3.4 Caveat: single-handler override semantics

`registerSem` currently uses `Map.set(type, handler)` so new handler replaces old one for that type (no built-in composition stack).

Evidence: `semRegistry.ts` map behavior.

## 4.3.5 Sandbox plugins do not currently get SEM APIs

QuickJS stack plugin bootstrap surface does not expose `subscribeSem` or reducer registration hooks.

Evidence: `stack-bootstrap.vm.js` and plugin authoring typings.

---

## 4.4 GEPA-04 baseline (your note, confirmed)

Your note is correct and now validated in code:

1. `go-go-gepa` plugin methods receive event hooks:
   - `options.emitEvent(payload)`
   - `options.events.emit(payload)`
2. CLI commands support `--stream` and print `stream-event { ... }` envelopes.
3. `jsbridge.Emitter` annotates plugin event sequence/timestamp/plugin method metadata.

Evidence:

1. `go-go-gepa/cmd/gepa-runner/plugin_loader.go`
2. `go-go-gepa/pkg/dataset/generator/plugin_loader.go`
3. `go-go-gepa/pkg/jsbridge/emitter.go`
4. `go-go-gepa/cmd/gepa-runner/plugin_stream.go`
5. `go-go-gepa/cmd/gepa-runner/stream_cli_integration_test.go`

> [!NOTE]
> This solves plugin-event streaming in GEPA runner. It does **not** by itself provide SEM reducer registration or automatic timeline projection in Pinocchio.

---

## 5. Capability Matrix

| Capability | Geppetto | Pinocchio Backend | go-go-os Frontend |
| --- | --- | --- | --- |
| JS event handler subscription for live events | Yes (`session.start().on`) | No | Yes (app-bundle TS runtime) |
| Literal `llm.delta` event name | No (`partial` equivalent) | Yes (SEM) | Yes (SEM) |
| SEM translation | No | Yes | N/A |
| Timeline projection reducer | No | Yes (Go handlers + built-ins) | Yes (Redux + sem handlers) |
| JS-registered backend reducers | No | No | N/A |
| JS-registered frontend reducers/projection | N/A | N/A | Yes (app code), No (sandbox plugins) |

---

## 6. Why This Feels Ambiguous (Root Cause)

The same concept (streaming model tokens/events) exists under different names at different layers:

1. Geppetto emits `partial`.
2. Pinocchio SEM emits `llm.delta` after translation.
3. Frontend consumes `llm.delta` and/or `timeline.upsert`.

That naming split makes it seem like "JS handling for llm.delta" should be in geppetto, but geppetto is intentionally upstream and protocol-agnostic for SEM.

---

## 7. Design Options for Your Requested Functionality

## Option A: Frontend-only JS customization (fastest)

1. Register custom handlers with `registerSem(...)` in go-go-os app runtime modules.
2. Register custom renderers/normalizers.
3. Optionally consume raw envelope bus for additive observers.

Pros:

1. immediate,
2. no backend runtime changes,
3. great for UI experimentation.

Cons:

1. no server-side projection control,
2. no effect on `/api/timeline` persisted model unless backend also emits compatible entities.

## Option B: Backend Go handlers + minimal JS hooks (moderate)

1. Keep pinocchio projection in Go.
2. Add explicit hook events that call into a constrained JS callback runtime for side effects, not core projection.

Pros:

1. safer operationally,
2. lower blast radius.

Cons:

1. not full JS-defined reducers.

## Option C: Full backend JS reducer runtime in pinocchio (target for your request)

1. Add goja runtime hosting JS reducer modules.
2. Expose registration API `registerSemReducer(eventType, reducer)` and `onSem(eventType, handler)`.
3. Allow reducer outputs to upsert one/many timeline entities.

Pros:

1. fulfills requirement directly,
2. dynamic server-side projection behavior.

Cons:

1. highest complexity and risk,
2. needs strict sandboxing and lifecycle controls.

---

## 8. Recommended Architecture

Recommendation: staged hybrid.

## Stage 1 (now)

1. Use existing GEPA-04 plugin stream events.
2. Bridge selected GEPA events into Pinocchio SEM using backend Go mappings.
3. Use go-go-os runtime module registration for JS-side reactions and rendering.

## Stage 2

1. Add **composable** frontend SEM handler chains (instead of single overwrite map).
2. Add typed helper for geppetto `partial` <-> SEM `llm.delta` aliasing in JS API docs.

## Stage 3 (if still needed)

1. Implement backend JS reducer runtime in Pinocchio with strong guardrails.

This sequence gives value early while deferring risky infrastructure until necessary.

---

## 9. Proposed API Contracts

## 9.1 Geppetto-side handler ergonomics

```js
const handle = session.start(seedTurn, { timeoutMs: 30000 });

handle
  .on("partial", (ev) => {
    // ev.delta, ev.completion
  })
  .on("*", (ev) => {
    // global observer
  });

const out = await handle.promise;
```

Potential compatibility alias (proposed):

```js
handle.on("llm.delta", cb); // internally mapped to "partial"
```

## 9.2 Frontend composable SEM handlers (proposed improvement)

Current risk:

```ts
registerSem("llm.delta", handlerA);
registerSem("llm.delta", handlerB); // replaces A today
```

Proposed API:

```ts
const offA = addSemHandler("llm.delta", handlerA, { order: 10 });
const offB = addSemHandler("llm.delta", handlerB, { order: 20 });
const offWild = addSemHandler("*", wildcardHandler);
```

## 9.3 Pinocchio backend JS reducer API (proposed)

```js
// sem-reducers.js loaded at server startup
registerSemReducer("llm.delta", (state, event) => {
  return {
    upserts: [
      {
        id: event.id,
        kind: "message",
        props: {
          role: "assistant",
          content: event.data.cumulative,
          streaming: true,
        },
      },
    ],
    state,
  };
});

onSem("*", (event) => {
  // side-effect observer: metrics/logging/traces
});
```

Host-side Go interface sketch:

```go
type JSReducerRuntime interface {
    AddReducer(eventType string, reducer JSReducer)
    AddHandler(eventType string, handler JSEventHandler)
    Dispatch(ctx context.Context, ev TimelineSemEvent, now int64) (upserts []*timelinepb.TimelineEntityV2, handled bool, err error)
}
```

---

## 10. Prototype Evidence from This Ticket

Ticket script added:

- `scripts/js-sem-reducer-handler-prototype.js`

What it demonstrates:

1. current single-map handler model causes override (`["custom:hello"]`),
2. composable model supports reducer chains + typed handlers + wildcard handlers.

Observed output confirms:

1. overwrite risk in current semantics,
2. composable API solves additive extension use case.

---

## 11. Implementation Plan (Detailed)

## Phase 0: Contract and vocabulary alignment

1. Define canonical event naming table:
   - geppetto `partial` <-> sem `llm.delta`.
2. Define supported extension surfaces by layer.
3. Decide whether backend JS reducer runtime is required for first milestone.

## Phase 1: Immediate developer ergonomics

1. Document geppetto JS `partial` handling as delta equivalent.
2. Add alias support or helper docs for `llm.delta` mapping in JS bindings.
3. Add go-go-os helper module patterns for SEM extensions.

## Phase 2: Frontend handler composition hardening

1. Replace `Map<string, Handler>` with chain list per type.
2. Add wildcard and unsubscribe semantics.
3. Add `SemContext.getState` for state-aware reducers.

## Phase 3: GEPA stream to SEM bridge

1. Take GEPA-04 plugin stream events.
2. Map selected event classes to SEM (`gepa.*` or `timeline.upsert`).
3. Feed through Pinocchio stream backend and timeline projector.

## Phase 4: Optional backend JS reducer runtime in Pinocchio

1. Add loader and sandbox for reducer scripts.
2. Register reducers/handlers at startup with capability controls.
3. Integrate into `handleTimelineHandlers` path before built-ins.
4. Provide deterministic precedence and fallback behavior.

## Phase 5: Stability/operations

1. Timeouts + panic guards for JS reducer callbacks.
2. Metrics around reducer latency/error/drop counts.
3. Rate limit and payload size protection.

---

## 12. Test Strategy

## 12.1 Unit tests

1. Geppetto alias mapping tests (`partial` vs `llm.delta`).
2. Frontend sem handler chain tests:
   - order,
   - wildcard,
   - unsubscribe,
   - no default override regressions.
3. Backend JS reducer runtime tests:
   - valid upsert generation,
   - malformed reducer output rejection,
   - timeout behavior.

## 12.2 Integration tests

1. GEPA plugin emits event via GEPA-04 hooks -> bridged SEM -> frontend receives.
2. `llm.delta` handler reacts without breaking built-in stream state transitions.
3. Reconnect/hydration path remains consistent with custom reducers.

## 12.3 Regression tests

1. Existing default SEM handlers still function.
2. Timeline snapshot version semantics unchanged.
3. No duplicate timeline entities introduced by dual handlers.

---

## 13. Risks and Mitigations

## 13.1 Risk: confused semantics (`partial` vs `llm.delta`)

Mitigation:

1. explicit aliasing rules,
2. docs and helper constants,
3. validation warnings when unknown synonyms are used.

## 13.2 Risk: handler override regressions in frontend

Mitigation:

1. move to composable chain API,
2. keep `registerDefaultSemHandlers()` idempotent/additive behavior,
3. add tests for extension-safe defaults.

## 13.3 Risk: backend JS runtime safety

Mitigation:

1. strict execution budget,
2. no unrestricted host APIs,
3. fallback to Go built-ins on error,
4. structured error telemetry.

## 13.4 Risk: event flood from GEPA streams

Mitigation:

1. coalesce high-frequency progress events,
2. emit coarse-grained milestones for timeline,
3. keep raw verbose events in debug channel only.

---

## 14. Decision Guidance

If your immediate goal is developer workflows and prototyping:

1. use existing geppetto JS `on("partial")` and go-go-os `registerSem(...)` now,
2. leverage GEPA-04 stream events as source,
3. bridge only essential events into SEM/timeline first.

If your goal is dynamic backend projection logic owned by JS teams:

1. plan dedicated Pinocchio backend JS reducer runtime as a controlled subsystem,
2. do it after frontend composition and event naming are stabilized.

---

## 15. External Documentation

1. Goja runtime: <https://github.com/dop251/goja>
2. goja_nodejs event loop: <https://github.com/dop251/goja_nodejs>
3. Protobuf JSON mapping: <https://protobuf.dev/programming-guides/proto3/#json>
4. Gorilla websocket docs: <https://pkg.go.dev/github.com/gorilla/websocket>
5. Watermill pub/sub docs: <https://watermill.io/docs/getting-started/>

---

## 16. Open Questions

1. Do you want backend JS reducers to be hot-reloadable or startup-only?
2. Should custom JS reducers be allowed to mutate historical entities, or append-only upserts?
3. Should GEPA plugin events map directly to `timeline.upsert`, or first to typed `gepa.*` SEM events with dedicated projectors?
4. Is sandbox plugin runtime in go-go-os intended to ever consume live SEM directly, or should that remain app-runtime-only?
5. Should we standardize on `llm.delta` nomenclature at all layers, or preserve geppetto `partial` upstream and map only at boundaries?

---

## 17. Key Evidence References

### Geppetto

1. `geppetto/pkg/events/chat-events.go:15`
2. `geppetto/pkg/events/chat-events.go:17`
3. `geppetto/pkg/js/modules/geppetto/api_sessions.go:503`
4. `geppetto/pkg/js/modules/geppetto/api_sessions.go:530`
5. `geppetto/pkg/js/modules/geppetto/api_events.go:19`
6. `geppetto/pkg/js/modules/geppetto/api_events.go:47`
7. `geppetto/pkg/js/modules/geppetto/api_events.go:103`

### Pinocchio

1. `pinocchio/pkg/webchat/stream_coordinator.go:152`
2. `pinocchio/pkg/webchat/stream_coordinator.go:176`
3. `pinocchio/pkg/webchat/sem_translator.go:132`
4. `pinocchio/pkg/webchat/timeline_projector.go:82`
5. `pinocchio/pkg/webchat/timeline_projector.go:116`
6. `pinocchio/pkg/webchat/timeline_projector.go:145`
7. `pinocchio/pkg/webchat/timeline_registry.go:21`
8. `pinocchio/pkg/webchat/timeline_registry.go:29`

### go-go-os

1. `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts:50`
2. `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts:55`
3. `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts:71`
4. `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts:320`
5. `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts:338`
6. `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/runtime/registerChatModules.ts:38`
7. `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/plugin-runtime/stack-bootstrap.vm.js:122`

### go-go-gepa (GEPA-04 baseline)

1. `go-go-gepa/cmd/gepa-runner/plugin_loader.go:297`
2. `go-go-gepa/cmd/gepa-runner/plugin_loader.go:337`
3. `go-go-gepa/pkg/dataset/generator/plugin_loader.go:147`
4. `go-go-gepa/pkg/jsbridge/emitter.go:48`
5. `go-go-gepa/cmd/gepa-runner/plugin_stream.go:20`
6. `go-go-gepa/cmd/gepa-runner/stream_cli_integration_test.go:112`

