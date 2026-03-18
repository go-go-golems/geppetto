---
Title: Opinionated JavaScript API design and implementation guide
Ticket: GP-46-OPINIONATED-JS-APIS
Status: active
Topics:
    - geppetto
    - javascript
    - js-bindings
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/js/geppetto/20_events_collector_sink.js
      Note: Representative current streaming example
    - Path: examples/js/geppetto/22_runner_run.js
      Note: Simple blocking runner example
    - Path: examples/js/geppetto/23_runner_profile_run.js
      Note: Profile-driven runner example
    - Path: examples/js/geppetto/24_runner_start_handle.js
      Note: Top-level runner.start handle example
    - Path: pkg/doc/topics/13-js-api-reference.md
      Note: |-
        Current JS API contract documentation
        Reference docs now teach gp.runner as the default API
    - Path: pkg/doc/topics/14-js-api-user-guide.md
      Note: |-
        Current user-facing JS composition guidance
        User guide now teaches runner-first workflow
    - Path: pkg/js/modules/geppetto/api_builder_options.go
      Note: Dense low-level builder options that motivate a higher-level runner API
    - Path: pkg/js/modules/geppetto/api_engines.go
      Note: Explicit engine construction boundary
    - Path: pkg/js/modules/geppetto/api_events.go
      Note: Current event collector sink behavior
    - Path: pkg/js/modules/geppetto/api_middlewares.go
      Note: Go and JS middleware materialization gap
    - Path: pkg/js/modules/geppetto/api_profiles.go
      Note: Profile resolution and runtime stack binding
    - Path: pkg/js/modules/geppetto/api_runner.go
      Note: Implementation of the new gp.runner namespace and runtime/prepared-run assembly
    - Path: pkg/js/modules/geppetto/api_sessions.go
      Note: Current session and builder assembly path
    - Path: pkg/js/modules/geppetto/api_tools_registry.go
      Note: Current tool registry behavior and future filter point
    - Path: pkg/js/modules/geppetto/module.go
      Note: |-
        Top-level JS module exports and namespace boundaries
        Top-level JS export surface now includes gp.runner
    - Path: pkg/js/modules/geppetto/module_test.go
      Note: Current JS module behavior coverage
    - Path: pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl
      Note: |-
        Current generated JS typing surface
        TypeScript template updated with runner contracts
ExternalSources: []
Summary: Design-first guide for adding an additive opinionated JS runner layer above Geppetto's current low-level builder/session module.
LastUpdated: 2026-03-18T10:30:00-04:00
WhatFor: Explain the current JS module architecture, identify the gaps, and propose a concrete opinionated JS API with a phased implementation plan.
WhenToUse: Use when designing or implementing a higher-level JavaScript API on top of require("geppetto").
---



# Opinionated JavaScript API design and implementation guide

## Executive Summary

Geppetto's current JavaScript module already has the core pieces needed for serious work:

- explicit engine construction,
- low-level session and builder composition,
- tool registry assembly,
- event collector sinks,
- profile registry resolution,
- generated TypeScript declarations,
- strong example and test coverage.

The problem is not capability. The problem is that the default JS surface is still advanced-by-default.

Today, a script that wants to do something real usually has to assemble several concepts manually:

1. resolve profile metadata,
2. build an engine explicitly,
3. build or import a tool registry,
4. wire a builder or session,
5. attach event sinks,
6. choose between `run`, `runAsync`, and `start`,
7. append a turn and wait for completion.

That is a reasonable low-level API. It is not yet a great opinionated API.

The recommendation in this document is to add a new additive `gp.runner` namespace to `require("geppetto")`, rather than trying to further grow the existing `createBuilder` / `createSession` surface. The new namespace should mirror the cleaned-up Go boundary:

- engines remain explicit,
- profiles resolve runtime metadata only,
- the runner layer consumes already-resolved execution inputs,
- `prepare`, `run`, and `start` become the default app-facing JS workflow,
- the existing builder/session layer remains available as the advanced escape hatch.

This is the best fit for the codebase after the Go simplification work in GP-40 through GP-45.

## Problem Statement and Scope

### The problem

The JavaScript module currently exposes a flexible but fragmented API. The main exports are installed in `pkg/js/modules/geppetto/module.go`, which registers:

- `createBuilder`
- `createSession`
- `runInference`
- `turns`
- `engines`
- `profiles`
- `schemas`
- `middlewares`
- `events`
- `tools`

This is powerful, but it creates three onboarding problems:

1. There is no single obvious "small default" path for building a real script.
2. The profile-to-runtime-to-execution path stops halfway through.
3. The low-level builder and session API becomes the path of least resistance even for simple scripts.

### Why this matters now

This problem became more important after the Go API simplification. On the Go side, Geppetto now has a clear opinionated layer in `pkg/inference/runner`. On the JS side, the public module still reflects the older lower-level assembly mindset.

That mismatch makes the platform feel inconsistent:

- Go: "use the runner unless you need advanced control"
- JS: "assemble everything manually, then use sessions/builders directly"

### Scope of this ticket

This ticket is about the Geppetto JS module exposed through `require("geppetto")`.

In scope:

- the current JS module API shape,
- how scripts currently assemble runtime behavior,
- how profile resolution interacts with execution,
- how an opinionated JS layer should look,
- phased implementation guidance for the new layer.

Out of scope:

- changing the Go runner API again,
- redesigning the low-level session/builder contracts from scratch,
- adding profile-based engine setting overlays back into the system,
- changing Goja or host-runtime ownership semantics.

## Current-State Architecture

This section is evidence-based. Every major claim here maps to concrete files in the current tree.

### 1. The module exports a low-level composition surface

The top-level JS module is installed in `pkg/js/modules/geppetto/module.go`.

That file shows the current public contract:

```text
require("geppetto")
  -> version
  -> createBuilder(opts?)
  -> createSession(opts)
  -> runInference(engine, turn, opts?)
  -> turns.*
  -> engines.*
  -> profiles.*
  -> schemas.*
  -> middlewares.*
  -> events.*
  -> tools.*
```

This is important because it shows that the current default entry points are still:

- builder-oriented (`createBuilder`),
- session-oriented (`createSession`),
- or a one-shot helper (`runInference`).

There is no dedicated runner namespace today.

Relevant files:

- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
- `geppetto/pkg/doc/topics/13-js-api-reference.md`

### 2. Sessions and builders are the real execution assembly layer

The execution path is implemented in `pkg/js/modules/geppetto/api_sessions.go`.

That file does three related things:

- builds sessions from a `builderRef`,
- exposes chainable builder methods,
- exposes session lifecycle methods such as `run`, `runAsync`, and `start`.

The current JS builder supports methods such as:

- `withEngine`
- `useMiddleware`
- `useGoMiddleware`
- `withTools`
- `withToolLoop`
- `withToolHooks`
- `withPersister`
- `withEventSink`
- `withSnapshotHook`
- `buildSession`

From an intern's perspective, this is the "expert assembly" layer. It is roughly analogous to wiring `enginebuilder.Builder` directly on the Go side.

That means the current JS default is already a layer below the new Go default.

Relevant files:

- `geppetto/pkg/js/modules/geppetto/api_sessions.go`
- `geppetto/pkg/js/modules/geppetto/api_builder_options.go`
- `geppetto/pkg/doc/topics/13-js-api-reference.md`
- `geppetto/pkg/doc/topics/14-js-api-user-guide.md`

### 3. Engines are built explicitly from config or JS callbacks

The engine helpers live in `pkg/js/modules/geppetto/api_engines.go`.

The key fact here is that JS now follows the simplified post-GP-43 boundary:

- `gp.engines.fromConfig(...)` builds an engine from explicit config
- `gp.engines.fromFunction(fn)` builds a deterministic JS-backed engine
- `gp.engines.echo(...)` builds a tiny local engine for smoke tests

`fromConfig(...)` explicitly maps JS options into `StepSettings` and does not use profile registries to create engine settings.

This is the right architectural direction. It means the opinionated JS design should not try to reintroduce profile-based engine configuration. It should build on top of this explicit engine boundary.

Relevant files:

- `geppetto/pkg/js/modules/geppetto/api_engines.go`
- `geppetto/pkg/doc/playbooks/07-wire-provider-credentials-for-js-and-go-runner.md`
- `geppetto/examples/js/geppetto/06_live_profile_inference.js`

### 4. Profiles already resolve runtime metadata, but execution does not consume that metadata directly

The profile APIs live in `pkg/js/modules/geppetto/api_profiles.go`.

The important current behavior is:

- `profiles.resolve(...)` returns resolved runtime metadata,
- the result includes `effectiveRuntime`,
- `effectiveRuntime` contains `system_prompt`, `middlewares`, and `tools`,
- scripts can inspect this result today,
- but the module does not provide a first-class path that takes this result and feeds it directly into execution assembly.

This is the key halfway-house problem in the current JS surface.

The docs already teach the simplified conceptual boundary:

```text
profiles.resolve(...)
  -> runtime metadata only

engines.fromConfig(...)
  -> explicit engine
```

But after that, the script still has to manually translate the pieces into builder/session assembly.

Relevant files:

- `geppetto/pkg/js/modules/geppetto/api_profiles.go`
- `geppetto/pkg/doc/topics/14-js-api-user-guide.md`
- `geppetto/examples/js/geppetto/10_engines_from_profile_metadata.js`
- `geppetto/examples/js/geppetto/19_profiles_connect_stack_runtime.js`

### 5. Middleware and tool configuration are flexible but still low-level

The builder option parser in `api_builder_options.go` and the middleware/tool helpers in `api_middlewares.go` and `api_tools_registry.go` expose a dense execution configuration surface.

This is useful, but it also means the caller has to understand several layers at once:

- middleware instances or JS middleware functions,
- named Go middleware factories,
- tool registries,
- tool-loop settings,
- tool hooks,
- event sinks,
- snapshot hooks,
- persisters.

This is exactly the kind of complexity that a higher-level `gp.runner` layer should absorb.

One especially important current gap is that profile middleware uses are visible as metadata, but the module does not yet have a small first-class helper that says:

```text
take resolved profile middleware uses
  -> materialize them through Go middleware factories
  -> attach them to execution
```

Relevant files:

- `geppetto/pkg/js/modules/geppetto/api_builder_options.go`
- `geppetto/pkg/js/modules/geppetto/api_middlewares.go`
- `geppetto/pkg/js/modules/geppetto/api_tools_registry.go`

### 6. Streaming exists, but the ergonomic surface is still session-first

Streaming in JS works today through:

- `session.start(...)`
- the event collector sink from `events.collector()`
- callbacks attached to the returned handle or sink

That is functional and useful, but it is still session-oriented, not runner-oriented.

The current streaming example in `examples/js/geppetto/20_events_collector_sink.js` demonstrates this clearly:

```javascript
const sink = gp.events.collector().on("*", ...);

const session = gp.createBuilder()
  .withEngine(engine)
  .withTools(registry, { enabled: true, maxIterations: 3 })
  .withEventSink(sink)
  .buildSession();

session.append(...);
const out = session.run();
```

This is a good advanced example. It is not the smallest possible high-level API.

Relevant files:

- `geppetto/pkg/js/modules/geppetto/api_events.go`
- `geppetto/pkg/js/modules/geppetto/api_sessions.go`
- `geppetto/examples/js/geppetto/20_events_collector_sink.js`

### 7. The tests and docs confirm the current philosophy

The JS docs and tests are solid. They show that the current API is deliberate.

The tests in `module_test.go` cover:

- turns,
- constants,
- sessions,
- builder composition,
- JS + Go middlewares,
- tool registries,
- async behavior,
- profile registry resolution,
- schema catalogs,
- event collectors.

The user guide and tutorial currently teach:

- script-first iteration,
- explicit engine construction,
- builder/session composition,
- registry resolution and inspection,
- event collectors and Go tools.

This means the opinionated design should be additive and respectful. We are not fixing a broken API. We are adding a smaller default path above a capable advanced API.

Relevant files:

- `geppetto/pkg/js/modules/geppetto/module_test.go`
- `geppetto/pkg/doc/topics/13-js-api-reference.md`
- `geppetto/pkg/doc/topics/14-js-api-user-guide.md`
- `geppetto/pkg/doc/tutorials/05-js-api-getting-started.md`

## Gap Analysis

The table below summarizes the main mismatch between current behavior and the desired post-Go-runner experience.

| Area | Current state | Desired state |
|---|---|---|
| Default entry point | `createBuilder` / `createSession` / `runInference` | `gp.runner.run` / `gp.runner.start` / `gp.runner.prepare` |
| Profile resolution to execution | manual translation by caller | first-class runner path |
| Runtime metadata usage | inspectable but not directly consumable | resolved runtime object consumed by runner |
| Tool filtering | caller manually manages registry contents | runner can filter from runtime `toolNames` without mutating caller registry |
| Streaming ergonomics | session-first | runner-first |
| Documentation framing | advanced API appears as default | advanced API remains available but no longer the default teaching path |

### Main pain points

#### Pain point 1: Too many concepts must be assembled manually

The caller currently has to understand:

- engine construction,
- builder options,
- event sinks,
- tool loop settings,
- tool registries,
- runtime metadata resolution.

This is too much for "simple script" use cases.

#### Pain point 2: Profiles stop at metadata

The current JS profile API is correct conceptually, but it is not ergonomic enough.

It answers:

- "what runtime metadata did this profile resolve to?"

But it does not answer:

- "how do I turn that metadata into an executable inference run without wiring a builder manually?"

#### Pain point 3: Streaming is more advanced than it should be

`session.start()` and event collector handles are useful, but they are lower-level than necessary for common cases.

The JS equivalent of the Go runner should make the streaming path look like:

```javascript
const handle = gp.runner.start({ ... });
handle.on("*", ...);
const out = await handle.wait();
```

not:

```javascript
const session = gp.createBuilder()...buildSession();
session.append(...);
const handle = session.start();
```

#### Pain point 4: Middleware uses from profile resolution are not first-class execution inputs

This is the most subtle gap in the current design.

The module can:

- list middleware schemas,
- create Go middleware references manually,
- resolve profile runtime metadata that contains middleware uses.

But there is no small path that says:

```text
profile middleware uses -> actual execution middlewares
```

That is a real missing layer.

## Design Goals

The proposed JS opinionated API should satisfy these goals.

### Goal 1: Mirror the new Go runner mental model

The JS API should feel conceptually aligned with the Go runner:

- caller owns engine creation explicitly,
- profiles supply runtime metadata,
- a runner layer assembles execution,
- sync and streaming paths are obvious.

### Goal 2: Keep the low-level API available

`createBuilder`, `createSession`, and `runInference` should remain available.

They are still valuable for:

- tests,
- hosts,
- exotic integrations,
- advanced tuning,
- backwards-compatible script support.

### Goal 3: Make the small path obvious

A new engineer should be able to ask:

> "I have an engine, maybe a profile, maybe a tool registry. How do I just run this?"

and get a single obvious answer.

### Goal 4: Preserve the explicit engine boundary

The new design must not turn profiles back into engine-setting overlays.

Profiles should continue to resolve:

- system prompt,
- middleware uses,
- tool names,
- runtime identity metadata.

Engines should continue to be built explicitly.

### Goal 5: Support both blocking and event-driven scripts

The opinionated layer must work for:

- one-shot scripts,
- streaming CLIs,
- event-driven hosts,
- deterministic tests.

## Recommended Design

### Recommendation: add a new `gp.runner` namespace

The best design is to add a new namespace:

```javascript
const gp = require("geppetto");

gp.runner.resolveRuntime(...)
gp.runner.prepare(...)
gp.runner.run(...)
gp.runner.start(...)
```

This is better than overloading the existing top-level APIs because it creates a clear conceptual split:

- low-level layer:
  - `createBuilder`
  - `createSession`
  - `runInference`
- opinionated app-facing layer:
  - `runner.resolveRuntime`
  - `runner.prepare`
  - `runner.run`
  - `runner.start`

### Why `gp.runner` is the best name

`gp.runner` works for three reasons:

1. It mirrors the Go naming directly.
2. It explains its purpose immediately.
3. It avoids overloading already-crowded top-level exports.

Names that are less good:

- `gp.app`: too generic
- `gp.chat`: too narrow
- `gp.runtime`: overloaded and ambiguous
- `gp.sessions2`: migration-smelling and awkward

## Proposed API Shape

### 1. `gp.runner.resolveRuntime(input)`

Purpose:

- resolve profile metadata,
- merge direct runtime metadata,
- materialize a runtime object suitable for execution.

Proposed input:

```typescript
interface RunnerResolveRuntimeInput {
  profile?: {
    registrySlug?: string;
    profileSlug?: string;
  };
  systemPrompt?: string;
  middlewares?: Array<MiddlewareRef | MiddlewareFn>;
  toolNames?: string[];
  runtimeKey?: string;
  runtimeFingerprint?: string;
  profileVersion?: number;
}
```

Proposed output:

```typescript
interface RunnerResolvedRuntime {
  systemPrompt?: string;
  middlewares: Array<MiddlewareRef | MiddlewareFn>;
  toolNames?: string[];
  runtimeKey?: string;
  runtimeFingerprint?: string;
  profileVersion?: number;
  metadata?: Record<string, any>;
}
```

Behavior:

- if `profile` is provided, call `profiles.resolve(...)`
- pull `effectiveRuntime.system_prompt`
- materialize `effectiveRuntime.middlewares` through Go middleware factories
- copy `effectiveRuntime.tools` into `toolNames`
- preserve runtime identity metadata
- merge any direct explicit runtime additions from the caller

Important constraint:

This function should not build engines. It resolves runtime metadata only.

### 2. `gp.runner.prepare(input)`

Purpose:

- assemble the underlying session and seed turn,
- apply runtime metadata,
- filter tools,
- return a prepared execution object.

Proposed input:

```typescript
interface RunnerRequest {
  engine: Engine;
  prompt?: string;
  seedTurn?: Turn;
  runtime?: RunnerResolvedRuntime;
  tools?: ToolRegistry;
  toolLoop?: ToolLoopSettings;
  eventSink?: EventSink;
  eventSinks?: EventSink[];
  snapshotHook?: SnapshotHook;
  persister?: TurnPersister;
  timeoutMs?: number;
  tags?: Record<string, any>;
}
```

Proposed output:

```typescript
interface PreparedRun {
  session: Session;
  turn: Turn;
  runtime?: RunnerResolvedRuntime;
  run(): Turn;
  start(): RunHandle;
}
```

This mirrors the Go `Prepare(...)` idea and gives advanced callers an inspection/customization step without forcing them down to `createBuilder`.

### 3. `gp.runner.run(input)`

Purpose:

- the simple blocking path.

Shape:

```javascript
const out = gp.runner.run({
  engine,
  prompt: "Explain the answer briefly.",
  runtime,
  tools: registry,
  toolLoop: { enabled: true, maxIterations: 4 },
});
```

Internally, this is just:

```text
resolve request
  -> prepare
  -> session.run(turn, runOptions)
  -> return final turn
```

### 4. `gp.runner.start(input)`

Purpose:

- the streaming/event-driven path.

Shape:

```javascript
const handle = gp.runner.start({
  engine,
  prompt: "Use the weather tool if needed.",
  runtime,
  tools: registry,
  toolLoop: { enabled: true, maxIterations: 4 },
  eventSink: gp.events.collector().on("*", console.log),
});

handle.on("*", (ev) => console.log(ev.type));
const out = await handle.wait();
```

Important note:

The new `start(...)` should return a runner-oriented handle, not force the user to first construct a session and then call `session.start()`.

## Internal Mechanics

This section explains how the new layer should be built on top of existing code.

### Internal flow

```text
Runner request
  -> require explicit engine
  -> resolve runtime metadata (optional)
  -> materialize middlewares
  -> materialize/filer tools
  -> build builderRef internally
  -> build session
  -> append prompt or seed turn
  -> run or start
```

### Pseudocode

```go
func (m *moduleRuntime) runnerRun(call goja.FunctionCall) goja.Value {
    req := decodeRunnerRequest(call.Arguments[0])

    prepared, err := m.prepareRunnerRequest(req)
    if err != nil {
        panic(m.vm.NewGoError(err))
    }

    out, err := prepared.session.runSync(prepared.turn, prepared.runOptions)
    if err != nil {
        panic(m.vm.NewGoError(err))
    }

    return m.encodeTurnValue(out)
}
```

```go
func (m *moduleRuntime) prepareRunnerRequest(req runnerRequest) (*preparedRun, error) {
    if req.Engine == nil {
        return nil, fmt.Errorf("runner request requires engine")
    }

    runtime, err := m.resolveRunnerRuntime(req.RuntimeInput)
    if err != nil {
        return nil, err
    }

    middlewares, err := m.materializeRunnerMiddlewares(runtime)
    if err != nil {
        return nil, err
    }

    registry, err := m.materializeFilteredRegistry(req.Tools, runtime.ToolNames)
    if err != nil {
        return nil, err
    }

    builder := m.newBuilderRef()
    builder.base = req.Engine
    builder.middlewares = middlewares
    builder.registry = registry
    builder.loopCfg, builder.toolCfg = materializeToolLoop(req.ToolLoop)
    builder.eventSinks = mergedEventSinks(...)
    builder.snapshotHook = req.SnapshotHook
    builder.persister = req.Persister

    sessionRef, err := builder.buildSession()
    if err != nil {
        return nil, err
    }

    turn := materializeSeedTurn(req.Prompt, req.SeedTurn, runtime)
    return &preparedRun{sessionRef, turn, runtime, req.RunOptions}, nil
}
```

### Materializing runtime middlewares

This is one of the most important internal helpers.

Rules:

1. If the caller already passed explicit middleware objects, keep them.
2. If the resolved runtime contains profile middleware uses, convert them through `resolveGoMiddleware(name, options)`.
3. If the resolved runtime contains a system prompt, apply it as a system-prompt middleware.
4. Preserve ordering deterministically.

That flow should be documented explicitly because it bridges the biggest current gap between profile resolution and execution.

### Filtering tools by `toolNames`

The opinionated layer should not mutate the caller's registry in place.

Instead:

1. create a fresh in-memory registry,
2. copy tool definitions from the source registry,
3. include only tools allowed by `runtime.toolNames` if such a list exists.

This matters because the runner should remain safe for repeated calls and shared registries.

## Practical Examples

### Example 1: Small blocking script

```javascript
const gp = require("geppetto");

const runtime = gp.runner.resolveRuntime({
  profile: { profileSlug: "assistant" },
});

const out = gp.runner.run({
  engine: gp.engines.fromConfig({
    apiType: "openai",
    model: "gpt-4.1-mini",
    apiKey: ENV.OPENAI_API_KEY,
  }),
  prompt: "Summarize why explicit runtime boundaries help.",
  runtime,
});

console.log(out.blocks[out.blocks.length - 1].payload.text);
```

This is the simplest "real" shape the JS module should support.

### Example 2: Streaming tool-loop assistant

```javascript
const gp = require("geppetto");

const reg = gp.tools.createRegistry();
reg.register({
  name: "weather",
  description: "Look up a short weather summary",
  handler: ({ location }) => ({ summary: `Sunny in ${location}` }),
});

const sink = gp.events.collector()
  .on("*", (ev) => console.log("event", ev.type));

const handle = gp.runner.start({
  engine: gp.engines.fromConfig({
    apiType: "openai",
    model: "gpt-4.1-mini",
    apiKey: ENV.OPENAI_API_KEY,
  }),
  prompt: "If needed, use the weather tool for Berlin.",
  runtime: gp.runner.resolveRuntime({
    profile: { profileSlug: "assistant" },
  }),
  tools: reg,
  toolLoop: { enabled: true, maxIterations: 4 },
  eventSink: sink,
});

handle.on("*", (ev) => console.log("handle", ev.type));
const out = await handle.wait();
```

This is the JS equivalent of the Go streaming runner shape.

### Example 3: Deterministic test harness

```javascript
const gp = require("geppetto");

const engine = gp.engines.fromFunction((turn) => {
  turn.blocks.push(gp.turns.newAssistantBlock("READY"));
  return turn;
});

const out = gp.runner.run({
  engine,
  prompt: "say ready",
});

assert(out.blocks[out.blocks.length - 1].payload.text === "READY");
```

This preserves the current deterministic testing strengths while removing builder boilerplate.

## Design Decisions

### Decision 1: make the new API additive

Rationale:

- the current low-level API is still useful,
- the examples and tests already depend on it,
- additive introduction is lower risk,
- docs can shift the default path without forcing a hard break.

### Decision 2: keep engines explicit

Rationale:

- matches the Go simplification,
- prevents profile/runtime confusion,
- keeps provider credential handling honest,
- aligns with current `fromConfig(...)` design.

### Decision 3: use a new namespace instead of growing `createSession`

Rationale:

- `createSession` already carries too many concepts,
- builder options are already dense,
- a namespace gives room for `resolveRuntime`, `prepare`, `run`, and `start`,
- it avoids making a low-level function pretend to be a high-level API.

### Decision 4: let profile resolution contribute runtime metadata only

Rationale:

- consistent with the current docs,
- consistent with Go,
- avoids reintroducing profile-patched engine settings.

## Alternatives Considered

### Alternative A: keep the current API and improve docs only

Why it is not enough:

- docs can reduce confusion,
- but they cannot remove the boilerplate,
- and they cannot make the advanced layer feel like the small default path.

### Alternative B: overload `createSession(opts)` with more sugar

Why it is not recommended:

- the option object is already crowded,
- it further blurs "builder/session" and "runner" responsibilities,
- it makes the advanced layer even harder to reason about.

### Alternative C: add top-level `run` and `start` functions only

Why it is weaker than `gp.runner`:

- it pollutes the module top level,
- it leaves no clear place for `resolveRuntime` and `prepare`,
- it makes the public surface flatter and harder to scan.

### Alternative D: add `gp.app` or `gp.chat`

Why it is not recommended:

- `app` is too vague,
- `chat` is too narrow,
- `runner` maps directly to the Go concept and current problem.

## Detailed Implementation Plan

This is the recommended implementation order.

### Phase 1: add private runner internals

Files to add:

- `pkg/js/modules/geppetto/api_runner.go`
- `pkg/js/modules/geppetto/api_runner_runtime.go`
- optionally `pkg/js/modules/geppetto/api_runner_registry.go`

Tasks:

1. define private request/result structs,
2. implement runtime resolution helper,
3. implement middleware materialization helper,
4. implement registry clone/filter helper.

Goal:

- get the core internals correct before exposing public JS names.

### Phase 2: export `gp.runner`

Update:

- `pkg/js/modules/geppetto/module.go`
- `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
- generated `pkg/doc/types/geppetto.d.ts`

Public API:

- `gp.runner.resolveRuntime`
- `gp.runner.prepare`
- `gp.runner.run`
- `gp.runner.start`

Goal:

- make the new surface visible without changing the old one.

### Phase 3: test the new surface

Update:

- `pkg/js/modules/geppetto/module_test.go`

Add tests for:

1. minimal blocking run,
2. deterministic JS engine run,
3. profile resolution to runtime metadata consumption,
4. middleware use materialization,
5. tool filtering by `toolNames`,
6. event-driven start path,
7. failure behavior when engine is missing,
8. failure behavior when profile middleware names are unknown.

Goal:

- ensure the new layer is not just convenient, but correct.

### Phase 4: add example scripts

Update or add:

- `examples/js/geppetto/21_runner_blocking_profile.js`
- `examples/js/geppetto/22_runner_streaming_tools.js`
- `examples/js/geppetto/23_runner_deterministic_test.js`

Goal:

- provide concrete source-of-truth scripts for new users.

### Phase 5: reframe docs

Update:

- `pkg/doc/topics/13-js-api-reference.md`
- `pkg/doc/topics/14-js-api-user-guide.md`
- `pkg/doc/tutorials/05-js-api-getting-started.md`

Documentation changes:

- make `gp.runner` the default recommendation,
- relabel `createBuilder` / `createSession` as advanced,
- add migration notes for current JS callers.

### Phase 6: decide on deprecation framing

This can be delayed.

Possible policy:

- keep `createBuilder` / `createSession` fully supported,
- document them as advanced,
- do not remove them until the new runner layer is proven in real scripts.

## Implementation Tasks for a New Intern

This section is intentionally detailed and procedural.

### Task 1: understand the current module shape

Read these files in order:

1. `geppetto/pkg/js/modules/geppetto/module.go`
2. `geppetto/pkg/js/modules/geppetto/api_sessions.go`
3. `geppetto/pkg/js/modules/geppetto/api_builder_options.go`
4. `geppetto/pkg/js/modules/geppetto/api_engines.go`
5. `geppetto/pkg/js/modules/geppetto/api_profiles.go`

Your goal is to answer:

- where the JS exports are installed,
- how sessions are built,
- how engines are built,
- how profiles are resolved.

### Task 2: prototype runtime resolution privately

Before adding public exports, write internal helpers that can:

- accept profile input,
- resolve profile runtime metadata,
- map middleware uses into middleware instances,
- preserve runtime identity metadata.

Do not expose this publicly first. Prove the internal mechanics with tests.

### Task 3: implement tool filtering carefully

Do not mutate caller registries in place.

Build a helper that:

- creates a fresh in-memory registry,
- copies only allowed tools,
- leaves the original registry unchanged.

Why this matters:

- callers may reuse registries across runs,
- mutation would produce surprising cross-run behavior.

### Task 4: implement `prepare`

This is the most important function.

It should:

1. validate the request,
2. create or clone the seed turn,
3. append a prompt block if needed,
4. configure event sinks and hooks,
5. build the internal session,
6. return a prepared object.

Everything else becomes simpler once `prepare` is correct.

### Task 5: implement `run` and `start`

`run`:

- should be a thin blocking wrapper above `prepare`.

`start`:

- should be the streaming wrapper,
- should integrate cleanly with current run-handle semantics,
- should make event subscription easy.

### Task 6: update typings and docs

Do not leave the new API undocumented.

Update:

- TypeScript template,
- reference docs,
- user guide,
- example scripts.

### Task 7: decide what stays advanced

Do not remove low-level functions casually.

After the new layer is in place, explicitly document:

- when to use `gp.runner`,
- when to keep using `createBuilder`,
- when to use `createSession`,
- when `runInference` is still appropriate.

## Risks

### Risk 1: re-blurring engine and profile responsibilities

This is the biggest architectural risk.

Mitigation:

- require explicit engines,
- keep profile resolution metadata-only,
- do not add profile-based engine setting reconstruction.

### Risk 2: overloading the runner request object

If `gp.runner.run({...})` accepts too many optional shapes, it will become a new version of the current builder options problem.

Mitigation:

- keep request types narrow,
- prefer separate `resolveRuntime` and `prepare` helpers over giant overloaded objects.

### Risk 3: hidden Goja reference handling bugs

The JS module depends on non-enumerable hidden references (`__geppetto_ref`) to preserve Go object identity.

Mitigation:

- reuse existing `requireEngineRef`, `requireToolRegistry`, and related helpers,
- avoid export/encode/decode round-trips for ref-carrying objects when not needed.

### Risk 4: incorrect middleware use materialization

The profile runtime may reference Go middleware names that the current host did not register.

Mitigation:

- fail clearly and early,
- include explicit tests for unknown middleware names,
- document host requirements.

## Open Questions

These questions should stay visible until implementation starts.

1. Should `gp.runner.resolveRuntime(...)` accept both direct runtime fields and profile input, or should those be separate helpers?
2. Should `gp.runner.prepare(...)` be public in v1, or should JS start with only `run` and `start`?
3. Should `gp.runner.start(...)` return the same handle contract as `session.start(...)`, or a slightly richer runner-specific handle?
4. Should the JS runner surface support direct engine config objects, or should it require a prebuilt `Engine` object only?
5. Do we want a companion `gp.runner.fromConfig(...)` convenience, or does that risk repeating the same layering mistakes?

## Validation Strategy

Before calling the new design complete, validate all of the following:

### Unit and integration coverage

- module tests cover the new runner namespace
- deterministic JS engine tests still pass
- event collector streaming tests still pass
- profile runtime resolution tests still pass

### Documentation coverage

- `geppetto.d.ts` matches the new runtime
- JS API reference documents `gp.runner`
- user guide recommends `gp.runner` first
- examples demonstrate both blocking and streaming usage

### Manual smoke checks

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/21_runner_blocking_profile.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/22_runner_streaming_tools.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/23_runner_deterministic_test.js
```

## References

### Primary implementation files

- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/modules/geppetto/api_sessions.go`
- `geppetto/pkg/js/modules/geppetto/api_builder_options.go`
- `geppetto/pkg/js/modules/geppetto/api_engines.go`
- `geppetto/pkg/js/modules/geppetto/api_profiles.go`
- `geppetto/pkg/js/modules/geppetto/api_middlewares.go`
- `geppetto/pkg/js/modules/geppetto/api_tools_registry.go`
- `geppetto/pkg/js/modules/geppetto/api_events.go`
- `geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
- `geppetto/pkg/js/modules/geppetto/module_test.go`

### Primary docs and examples

- `geppetto/pkg/doc/topics/13-js-api-reference.md`
- `geppetto/pkg/doc/topics/14-js-api-user-guide.md`
- `geppetto/pkg/doc/tutorials/05-js-api-getting-started.md`
- `geppetto/examples/js/geppetto/06_live_profile_inference.js`
- `geppetto/examples/js/geppetto/10_engines_from_profile_metadata.js`
- `geppetto/examples/js/geppetto/20_events_collector_sink.js`

### Related design context

- `geppetto/pkg/inference/runner/`
- `geppetto/pkg/doc/topics/10-runner.md`
