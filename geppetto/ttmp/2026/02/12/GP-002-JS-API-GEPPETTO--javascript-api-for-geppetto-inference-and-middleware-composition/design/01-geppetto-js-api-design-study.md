---
Title: Geppetto JS API Design Study
Ticket: GP-002-JS-API-GEPPETTO
Status: active
Topics:
    - geppetto
    - javascript
    - goja
    - api-design
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/examples/generic-tool-calling/main.go
      Note: Practical composition example with toolloop and middleware
    - Path: cmd/gen-turns/main.go
      Note: Turns mapper generator used for kind/key ID tables
    - Path: pkg/inference/middleware/middleware.go
      Note: Canonical middleware chain contract
    - Path: pkg/inference/session/session.go
      Note: Session lifecycle and single-active inference invariants
    - Path: pkg/inference/tools/base_executor.go
      Note: Tool execution semantics and policy hooks
    - Path: pkg/inference/tools/registry.go
      Note: Tool registry contract used by toolloop and external callers
    - Path: pkg/inference/toolloop/enginebuilder/builder.go
      Note: Engine assembly seam for middleware/tools/sinks
    - Path: pkg/inference/toolloop/loop.go
      Note: Tool orchestration loop semantics and iteration controls
    - Path: pkg/js/embeddings-js.go
      Note: Existing Goja bridge patterns for sync/async/cancel
    - Path: pkg/turns/block_kind_gen.go
      Note: Generated canonical BlockKind mappings
    - Path: pkg/turns/keys_gen.go
      Note: Generated canonical turn/block/data key mappings
    - Path: pkg/turns/spec/turns_codegen.yaml
      Note: Source-of-truth schema for generated kind/key mappers
    - Path: pkg/turns/types.go
      Note: Turn and block data model
ExternalSources: []
Summary: Exhaustive design study for a Goja-powered JS API for Geppetto inference, engines, middleware composition, and turn/block manipulation.
LastUpdated: 2026-02-12T09:43:33-05:00
WhatFor: Define implementation-ready options and recommendation for Geppetto JavaScript API architecture.
WhenToUse: Use when implementing or reviewing JS API surfaces for inference/session/middleware/turns in Geppetto.
---


# Geppetto JavaScript API Design Study

## 1. Executive Summary

This study defines how to add a first-class JavaScript API to Geppetto (Go + Goja), with explicit support for:

- Running inference.
- Creating engines.
- Creating middleware in JavaScript.
- Assembling middleware pipelines from JavaScript that can include both JS and Go middleware.
- Manipulating turns and blocks safely.

The current Geppetto architecture already provides strong extension seams:

- `session.EngineBuilder` for assembly.
- `middleware.Middleware` (`func(HandlerFunc) HandlerFunc`) for pipeline composition.
- `toolloop` for tool orchestration.
- `turns.Turn` / `turns.Block` as canonical interaction state.

The missing piece is a dedicated JS runtime adapter and module contract. The recommended architecture is a **hybrid API style**:

1. A minimal functional core for correctness and composability.
2. A fluent builder layer for ergonomics.
3. A declarative middleware registry surface for long-lived embedding apps.

This hybrid approach matches current Geppetto internals, minimizes risk, and supports gradual evolution. It also integrates cleanly with the `go-go-goja` native module pattern (`modules.NativeModule` + `require("geppetto")`).

## 2. Scope and Success Criteria

### 2.1 Scope

In-scope:

- API shape for JS callers embedded in Goja runtimes.
- Mapping between JS API and existing Geppetto runtime types.
- Error, cancellation, and async behavior.
- Middleware composition including Go and JS middleware.
- Turn/block manipulation and metadata/data access.
- Design variants and tradeoffs.
- Practical experiment results validating behavior and performance constraints.

Out-of-scope:

- Full production implementation in this ticket.
- Provider-specific behavior redesign.
- Sandbox/security policy implementation details beyond API hooks.
- Building a separate Node runtime package (though compatibility is discussed).

### 2.2 Success Criteria

A proposed API is successful if it:

- Preserves existing inference/session/toolloop invariants.
- Makes middleware composition explicit and predictable.
- Avoids hidden data races or object aliasing surprises.
- Supports both blocking and async workflows.
- Keeps Go/JS boundary overhead manageable.
- Is testable using deterministic integration tests.
- Can be implemented incrementally without destabilizing existing users.

## 3. Methodology

This study combines:

1. Static architecture mapping from live Geppetto source.
2. Prior internal design context review.
3. Goja module authoring pattern analysis.
4. Direct Goja experiments for boundary cost and semantics.
5. Option comparison across API styles.

### 3.1 Primary Code Surfaces Studied

Core engine/session/pipeline:

- `pkg/inference/engine/engine.go`
- `pkg/inference/session/session.go`
- `pkg/inference/session/execution.go`
- `pkg/inference/toolloop/enginebuilder/builder.go`
- `pkg/inference/toolloop/enginebuilder/options.go`
- `pkg/inference/toolloop/loop.go`
- `pkg/inference/toolloop/step_controller.go`

Middleware and tools:

- `pkg/inference/middleware/middleware.go`
- `pkg/inference/middleware/systemprompt_middleware.go`
- `pkg/inference/middleware/reorder_tool_results_middleware.go`
- `pkg/inference/middleware/logging_middleware.go`
- `pkg/inference/tools/registry.go`
- `pkg/inference/tools/config.go`
- `pkg/inference/tools/definition.go`
- `pkg/inference/tools/base_executor.go`

Turn model:

- `pkg/turns/types.go`
- `pkg/turns/helpers_blocks.go`
- `pkg/turns/keys.go`
- `pkg/turns/key_families.go`
- `pkg/turns/toolblocks/toolblocks.go`

Current JS bridge:

- `pkg/js/embeddings-js.go`
- `pkg/js/TODO.md`

Factory/provider usage examples:

- `cmd/examples/simple-inference/main.go`
- `cmd/examples/simple-streaming-inference/main.go`
- `cmd/examples/middleware-inference/main.go`
- `cmd/examples/generic-tool-calling/main.go`
- `cmd/examples/openai-tools/main.go`

Event routing:

- `pkg/events/event-router.go`

Goja module pattern references:

- `~/.codex/skills/go-go-goja-module-authoring/SKILL.md`
- `.../references/goja-git-pattern.md`
- `.../references/validation-checklist.md`
- `$(go env GOMODCACHE)/github.com/go-go-golems/go-go-goja@v0.0.4/modules/common.go`
- `$(go env GOMODCACHE)/github.com/go-go-golems/go-go-goja@v0.0.4/engine/runtime.go`

## 4. Current Architecture Findings

### 4.1 Engine Contract Is Minimal and Strong

`engine.Engine` is intentionally small:

```go
type Engine interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

Implication for JS API:

- JS should not bypass this abstraction.
- JS-facing engine wrappers should adapt to this contract rather than introducing an incompatible parallel graph.

### 4.2 Session Owns Concurrency and Identity Invariants

`session.Session` enforces:

- Stable `SessionID`.
- Append-only turn history.
- Single active inference at a time.
- In-place mutation of latest turn during inference.
- Explicit cancel/wait handle lifecycle.

Implication:

- JS API must expose or respect “single active run per session”.
- JS should not present a misleading multi-concurrent API unless backed by multiple sessions.

### 4.3 Engine Builder Is the Assembly Seam

`toolloop/enginebuilder.Builder` already composes:

- Base engine.
- Middleware chain.
- Optional tool registry + loop configuration.
- Event sinks.
- Snapshot hooks.
- Step controller.
- Persister.

Implication:

- JS API should primarily “configure a builder” and then run via session.
- This avoids reimplementing orchestration logic in JS.

### 4.4 Middleware Chain Semantics Are Clear

`middleware.Chain(handler, mws...)` applies reverse wrapping order:

- `Chain(h, m1, m2, m3)` => `m1(m2(m3(h)))`.

Implication:

- JS middleware API must preserve this order exactly.
- Any divergence would produce debugging confusion when mixing Go and JS middleware.

### 4.5 Tool Loop Is Already Separated from Provider Engines

`toolloop.Loop`:

- Performs inference step.
- Extracts pending tool calls from turn blocks.
- Executes tools.
- Appends `tool_use` blocks.
- Iterates with max iteration guard.

Implication:

- JS API can expose tool behavior as configuration and middleware hooks, not as provider-specific hacks.

### 4.6 Turns/Blocks Are Suitable as JS Data Model

`turns.Turn` and `turns.Block` already represent conversation state in a provider-agnostic format:

- Blocks by kind (`user`, `llm_text`, `tool_call`, `tool_use`, `system`, `reasoning`, `other`).
- Opaque typed key wrappers for metadata/data.

Implication:

- JS API should treat turns/blocks as first-class objects.
- Must decide carefully how to expose typed metadata/data keys to JS ergonomically without erasing type guarantees.

### 4.7 Existing JS Bridge Demonstrates Patterns, but Is Narrow

`pkg/js/embeddings-js.go` demonstrates:

- Sync function export.
- Promise-based async export.
- Callback-style async export.
- Cancellation by returning cancel closure.

Implication:

- Inference API can reuse these patterns.
- But current approach is ad hoc; a module-wide convention is needed.

## 5. Goja and Module Authoring Constraints

From the `go-go-goja-module-authoring` guidance and module source pattern:

- Define JS contract first.
- Keep domain logic outside module loader glue.
- Implement `modules.NativeModule` with:
  - `Name() string`
  - `Doc() string`
  - `Loader(*goja.Runtime, *goja.Object)`
- Register with `modules.Register` and load via `require()`.
- Prefer lowerCamel JSON keys for JS-facing APIs.
- Integration-test `require("<module>")` with real runtime.

### 5.1 Why This Matters for Geppetto

A `require("geppetto")` module gives:

- Stable discoverable JS entrypoint.
- Coexistence with other modules (`fs`, `exec`, etc.).
- A natural migration path from current global injection style.

## 6. Experiment Results

Three experiments were created under ticket scripts:

- `scripts/goja_callback_bridge_experiment.go`
- `scripts/goja_middleware_composition_experiment.go`
- `scripts/goja_eventloop_async_experiment.go`

### 6.1 Experiment A: JS→Go Callback Boundary Cost

What was measured:

- Pure JS loop.
- JS loop calling trivial Go scalar function.
- JS loop calling Go function with object payload conversion (`map[string]any`).

Representative results from repeated runs:

- Pure JS: ~156ns to ~320ns per iteration.
- JS→Go scalar: ~733ns to ~1162ns per call.
- JS→Go object payload: ~5131ns to ~10872ns per call.

Observed multiplier vs pure JS:

- Scalar Go callback: ~3x to ~5x.
- Object conversion callback: ~20x to ~42x.

Design implication:

- Do not cross JS↔Go boundary per token for high-volume streams.
- Prefer batched events, coarse callback granularity, or buffered sink strategies.
- Keep hot-path transformations on one side when possible.

### 6.2 Experiment B: Mixed JS+Go Middleware Composition

Validated:

- JS middleware composition order works as expected.
- JS middleware can invoke Go post-processor and mutate result.
- JS thrown exception is surfaced back to Go as error string with stack location.

Result example:

- Final text transformed by JS + Go: `"HELLO WORLD [GO-POST]"`.
- Error path message: `Error: middleware exploded at <eval>:...`.

Hot-loop benchmark (3 middleware layers, 30k iterations):

- ~15us to ~29us per operation depending on run.

Design implication:

- Mixed middleware is viable.
- Error wrapping and stack preservation should be intentional in adapter.
- Repeated tiny middleware crossings are costly; keep operations meaningful.

### 6.3 Experiment C: Async + Cancellation with EventLoop

Validated with `goja_nodejs/eventloop`:

- Promise resolves from goroutine through loop scheduling.
- Cancel path can reject promise deterministically.
- Event order in test showed canceled branch arriving before successful branch when timing dictated.

Design implication:

- Async inference API should use event loop scheduling for resolve/reject.
- Cancellation should be explicit and mapped to predictable rejection semantics.

## 7. API Style Alternatives

This section evaluates multiple API styles requested in the ticket.

### 7.1 Style A: Minimal Functional API

Shape:

```js
const geppetto = require("geppetto");

const engine = geppetto.createEngine({ provider: "openai_responses", profile: "4o-mini" });
const session = geppetto.createSession({ engine });

let turn = geppetto.turn()
  .system("You are concise")
  .user("Hello")
  .build();

turn = session.run(turn);
```

Pros:

- Small surface area.
- Easy to reason about.
- Maps directly to current Go contracts.

Cons:

- Less discoverable for complex builder settings.
- Middleware composition can feel low-level.

Best fit:

- Power users, scripts, tests.

### 7.2 Style B: Fluent Builder API

Shape:

```js
const geppetto = require("geppetto");

const engine = geppetto
  .engineBuilder()
  .provider("openai_responses")
  .profile("4o-mini")
  .middleware(geppetto.middleware.systemPrompt("You are concise"))
  .middleware(geppetto.middleware.js((ctx, turn, next) => next(ctx, turn)))
  .toolRegistry(myRegistry)
  .maxIterations(5)
  .build();

const session = geppetto.session({ engine });
const result = session.runTurn(geppetto.turn().user("Hello").build());
```

Pros:

- Very ergonomic.
- Good discoverability.
- Easy progressive configuration.

Cons:

- Can hide defaults if not documented well.
- Risks builder bloat over time.

Best fit:

- App code and integration layers.

### 7.3 Style C: Declarative Graph/Manifest API

Shape:

```js
const app = geppetto.fromSpec({
  engine: { provider: "openai_responses", profile: "4o-mini" },
  middlewares: [
    { type: "system_prompt", prompt: "You are concise" },
    { type: "js", name: "uppercase", fn: (ctx, turn, next) => { /* ... */ } }
  ],
  tools: { enabled: true, maxIterations: 6 }
});
```

Pros:

- Serializable configuration.
- Nice for persisted workflows.
- Good for config-driven products.

Cons:

- Function-valued fields are hard to serialize reliably.
- Debugging generated pipelines can be opaque.
- Overkill for many script uses.

Best fit:

- Hosted systems, GUI builders, persisted agents.

### 7.4 Style D: Plugin Registry API

Shape:

```js
geppetto.plugins.registerMiddleware("myMw", fn);
geppetto.plugins.registerEngineFactory("myProvider", factoryFn);

const engine = geppetto.engineBuilder()
  .provider("myProvider")
  .middleware("myMw")
  .build();
```

Pros:

- Extensible ecosystem model.
- Useful for large organizations.

Cons:

- Namespace/versioning complexity.
- Harder lifecycle and deterministic loading.

Best fit:

- Later-phase plugin architecture, not day-1 core.

### 7.5 Style E: Event-Driven Session Object (Observer-centric)

Shape:

```js
const run = session.runAsync(turn);
run.on("event", e => ...);
run.on("done", result => ...);
run.cancel();
```

Pros:

- Natural for streaming UIs.
- Aligns with existing event sinks.

Cons:

- Adds lifecycle/state machine complexity.
- Easy to misuse if session concurrency rules are hidden.

Best fit:

- Optional async layer over blocking core.

## 8. Style Comparison Matrix

| Style | Ergonomics | Debuggability | Runtime Cost | Fits Current Architecture | Implementation Risk |
|---|---:|---:|---:|---:|---:|
| A Minimal Functional | Medium | High | Low | High | Low |
| B Fluent Builder | High | Medium-High | Low-Medium | High | Medium |
| C Declarative Graph | Medium | Medium | Medium | Medium | Medium-High |
| D Plugin Registry | Medium | Low-Medium | Medium | Medium | High |
| E Event-Driven Session | High (for async apps) | Medium | Medium | High (as wrapper) | Medium |

Recommendation: **Hybrid A + B + selective E**.

- A for stable low-level core.
- B for user ergonomics.
- E as optional async wrapper to avoid forcing complexity on all users.

## 9. Recommended JS API (Hybrid)

### 9.1 Top-Level Module Contract

```js
const geppetto = require("geppetto");

// Core factories
geppetto.createEngine(options)
geppetto.createSession(options)
geppetto.turn()

// Middleware namespaces
geppetto.middleware.systemPrompt(prompt)
geppetto.middleware.reorderToolResults()
geppetto.middleware.logging(opts)
geppetto.middleware.js(fn)

// Optional builder
geppetto.engineBuilder()
```

### 9.2 Engine Creation

Minimal form:

```js
const engine = geppetto.createEngine({
  provider: "openai_responses",
  profile: "4o-mini",
  middlewares: [
    geppetto.middleware.systemPrompt("You are concise")
  ],
  tools: {
    enabled: true,
    maxIterations: 5,
    maxParallelTools: 2,
    toolChoice: "auto"
  }
});
```

Fluent equivalent:

```js
const engine = geppetto
  .engineBuilder()
  .provider("openai_responses")
  .profile("4o-mini")
  .use(geppetto.middleware.systemPrompt("You are concise"))
  .tools({ enabled: true, maxIterations: 5, maxParallelTools: 2 })
  .build();
```

### 9.3 Session + Inference

Blocking:

```js
const session = geppetto.createSession({ engine });
const turn = geppetto.turn().user("Explain CAP theorem briefly").build();
const out = session.run(turn);
```

Async handle:

```js
const handle = session.runAsync(turn);
handle.cancel();
const out = await handle.wait();
```

Streaming wrapper (optional, not replacing blocking core):

```js
const run = session.runStream(turn);
run.on("event", e => console.log(e.type));
run.on("done", out => console.log(out));
run.on("error", err => console.error(err));
```

### 9.4 Middleware API

Canonical JS middleware signature:

```ts
type JsMiddleware = (ctx: RunContext, turn: Turn, next: (ctx: RunContext, turn: Turn) => Turn) => Turn;
```

Registration:

```js
const mw = geppetto.middleware.js((ctx, turn, next) => {
  const out = next(ctx, turn);
  for (const b of out.blocks) {
    if (b.kind === "llm_text" && b.payload?.text) {
      b.payload.text = b.payload.text.toUpperCase();
    }
  }
  return out;
});
```

Mixing JS and Go middleware:

```js
const engine = geppetto.engineBuilder()
  .use(geppetto.middleware.systemPrompt("You are concise"))    // Go middleware
  .use(geppetto.middleware.js(myJsMw))                          // JS middleware
  .use(geppetto.middleware.reorderToolResults())                // Go middleware
  .build();
```

### 9.5 Turn and Block Manipulation API

Builder-style create:

```js
const t = geppetto.turn()
  .id("optional-id")
  .system("You are concise")
  .user("Find weather in Paris and London")
  .build();
```

Direct block helpers:

```js
t.addBlock(geppetto.block.userText("Hello"));
t.addBlock(geppetto.block.toolCall({ id: "c1", name: "get_weather", args: { location: "Paris" } }));
```

Metadata/data helpers (preferred over direct map poking):

```js
t.meta.set("geppetto.session_id@v1", sessionId);
const sid = t.meta.getString("geppetto.session_id@v1");

t.data.set("geppetto.agent_mode@v1", "research");
```

Rationale:

- Keep low-level raw object access available.
- Provide helpers for typed key usage to reduce errors.

## 10. JS↔Go Data Model Mapping

### 10.1 Canonical JS Shapes

Turn:

```ts
interface Turn {
  id?: string;
  blocks: Block[];
  metadata?: Record<string, unknown>;
  data?: Record<string, unknown>;
}
```

Block:

```ts
interface Block {
  id?: string;
  kind: "user" | "llm_text" | "tool_call" | "tool_use" | "system" | "reasoning" | "other";
  role?: string;
  payload?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
}
```

### 10.2 Conversion Rules

Go→JS:

- `turns.Turn` exported as plain JS object.
- `BlockKind` mapped to stable string values.
- Opaque metadata/data wrappers exported as key-value maps.

JS→Go:

- Validate required fields (`kind`, payload shape where required).
- Convert string kinds to enum with fallback/error policy.
- Rehydrate metadata/data into wrapper maps.

### 10.3 Mutation Semantics

Recommended default:

- Middleware sees a mutable turn object.
- Mutations are intentional and reflected in output.
- Session-level in-place mutation behavior remains unchanged.

Optional helper:

- `turn.clone()` to support defensive middleware authoring.

## 11. Error Model

### 11.1 Principles

- Never swallow JS errors silently.
- Preserve JS stack trace context when crossing to Go.
- Wrap with phase context (`decode`, `middleware`, `encode`, `run`) on Go side.

### 11.2 Proposed Behavior

If JS middleware throws:

- Catch `*goja.Exception`.
- Convert to Go error with:
  - original message,
  - stack string,
  - middleware name if available.
- Surface through `RunInference` error return.

If Go middleware or engine returns error:

- Reject async promise or return thrown error in sync invocation.

## 12. Cancellation and Async Model

### 12.1 Baseline

Session already supports cancellation via `ExecutionHandle.Cancel()`.

### 12.2 JS API

Blocking API:

- `session.run(turn)` throws on error.

Async handle API:

- `const h = session.runAsync(turn)`
- `await h.wait()`
- `h.cancel()`

Streaming API (optional):

- Provides event callbacks and cancel.
- Internally driven by event sink + router pattern.

### 12.3 EventLoop Requirement

For async functions exported to JS:

- Resolve/reject promises on Goja event loop using `RunOnLoop`.
- Avoid resolving promises directly from background goroutine.

## 13. Performance Guidance and Guardrails

Given experiment data:

- JS→Go object calls are expensive relative to pure JS loops.
- Avoid per-token boundary calls.
- Prefer:
  - batch event delivery,
  - fewer larger payload crossings,
  - local processing on one side before crossing.

### 13.1 Practical Guardrails

- Middleware callbacks should operate on turn-level or block-group-level boundaries, not token-level by default.
- Streaming callbacks should include optional buffering/windowing.
- Provide instrumentation counters for boundary crossings in debug mode.

## 14. Implementation Architecture (Go Side)

### 14.1 Package Layout Proposal

```text
pkg/
  inference/
    jsbridge/
      contract.go          # shared adapter interfaces
      codec.go             # Turn/Block JS<->Go conversion
      errors.go            # error wrapping + stack capture
      runtime_context.go   # session/run context surface
      middleware_js.go     # JS middleware adapter
      engine_builder_js.go # JS builder wrappers
  js/
    modules/
      geppetto/
        module.go          # NativeModule registration + exports
        api_engine.go
        api_session.go
        api_turn.go
        api_middleware.go
        api_events.go
        docs.go
```

### 14.2 Service/Adapter Boundary

Keep business logic in existing packages:

- `pkg/inference/...`
- `pkg/turns/...`

Keep JS glue thin:

- decode args,
- call services,
- encode results,
- wrap errors.

This follows go-go-goja guidance and avoids logic drift.

### 14.3 Module Skeleton

```go
type module struct{}

func (m *module) Name() string { return "geppetto" }
func (m *module) Doc() string  { return "Geppetto inference/session/middleware module" }
func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    _ = exports.Set("createEngine", makeCreateEngine(vm))
    _ = exports.Set("createSession", makeCreateSession(vm))
    _ = exports.Set("turn", makeTurnBuilder(vm))
    _ = exports.Set("middleware", makeMiddlewareNamespace(vm))
}

func init() {
    modules.Register(&module{})
}
```

## 15. Middleware Composition Mechanics

### 15.1 Adapter Strategy

JS middleware function should be wrapped into Go middleware once at registration/build time:

- `func JsMiddlewareAdapter(vm *goja.Runtime, jsFn goja.Callable) middleware.Middleware`

Execution pipeline:

1. Convert incoming `*turns.Turn` to JS turn object.
2. Build JS `next` callable that calls wrapped Go next.
3. Invoke JS middleware.
4. Convert returned turn object back to Go turn.
5. Validate and propagate errors.

### 15.2 Identity and Cloning

Use policy option per middleware wrapper:

- `mutate` (default): JS receives direct mutable object view.
- `copy`: JS receives clone; adapter diffs/returns new output.

Default should be `mutate` for predictable performance and alignment with current session behavior.

## 16. Turn/Block API Detailed Options

### 16.1 Option 1: Plain Object First

- Expose turn/block as plain JS objects.
- Helpers optional.

Pros:

- Familiar JS ergonomics.

Cons:

- Easier to make shape mistakes.

### 16.2 Option 2: Class Wrapper + Plain Export

- `Turn` and `Block` class wrappers with methods.
- `toJSON()` returns plain object.

Pros:

- Better affordances and method docs.

Cons:

- More implementation complexity in Goja.

### 16.3 Recommendation

Use hybrid:

- Plain object canonical transport format.
- Builder/helper factories for correctness.

## 17. Engine and Provider Configuration Surface

### 17.1 Minimal Provider Options

```js
createEngine({
  provider: "openai_responses" | "openai" | "claude" | "gemini",
  profile: "4o-mini",
  overrides: {
    model: "...",
    temperature: 0.2,
    apiKey: "...",
    baseUrl: "..."
  }
})
```

### 17.2 Tool Config Surface

Mirror existing tool config fields in lowerCamel:

- `enabled`
- `toolChoice`
- `maxIterations`
- `executionTimeout`
- `maxParallelTools`
- `allowedTools`
- `toolErrorHandling`
- `retryConfig`

## 18. Event and Streaming API Avenues

Three viable approaches:

1. **Callback hooks in runAsync options**
   - simplest implementation.
2. **EventEmitter-like run handle**
   - best ergonomics for apps.
3. **Async iterator stream**
   - modern and composable.

Recommendation:

- Start with callback + EventEmitter-like handle.
- Add async iterator once stable.

Reason:

- Async iterator in Goja requires careful promise scheduling and backpressure semantics.

## 19. Compatibility With Existing Go Middleware

Expose a middleware catalog namespace in JS that wraps current Go middleware constructors:

```js
geppetto.middleware.systemPrompt(prompt)
geppetto.middleware.reorderToolResults()
geppetto.middleware.logging({ level: "info" })
```

This allows users to combine existing trusted middleware with JS custom middleware.

## 20. Security and Safety Considerations

Even for embedded JS:

- Validate middleware and tool names.
- Limit unsafe host function exposure.
- Add optional middleware execution timeout/circuit breaker.
- Redact sensitive fields in emitted events/logs.

For hosted/multi-tenant future:

- Runtime sandbox policy hooks.
- Quotas on memory/time/cross-boundary calls.

## 21. Testing Strategy

### 21.1 Unit Tests

- JS codec tests for turn/block conversion.
- Middleware adapter tests:
  - happy path,
  - thrown exceptions,
  - invalid returned shapes.

### 21.2 Integration Tests

- `require("geppetto")` loads in runtime.
- End-to-end run with fake engine.
- Mixed Go+JS middleware order verification.
- Cancellation behavior for async run.

### 21.3 Existing Inference Safety Net

Keep running package tests:

- `go test ./pkg/inference/... ./pkg/turns/... -count=1`

(Executed in this study; all passing.)

## 22. Incremental Delivery Plan

### Phase 1: Foundation

- Add module skeleton (`require("geppetto")`).
- Add turn/block codec.
- Add `createSession`, `run`, `runAsync` basics.

### Phase 2: Middleware

- Add JS middleware adapter.
- Add Go middleware wrappers in namespace.
- Add composition tests.

### Phase 3: Builder + Tools

- Add fluent engine builder.
- Expose tool config + registry bindings.

### Phase 4: Streaming Enhancements

- Add event-driven run handle.
- Optional async iterator API.

### Phase 5: Hardening

- Performance counters.
- Better diagnostics.
- Rich docs and examples.

## 23. Risk Register

### 23.1 Risk: Hidden Mutation Bugs

Cause:

- Shared turn object references across JS and Go.

Mitigation:

- Document mutate semantics.
- Add optional clone mode for middleware wrapper.
- Add tests for aliasing behavior.

### 23.2 Risk: Boundary Overhead in High-Frequency Streams

Cause:

- Frequent JS↔Go object conversion.

Mitigation:

- Batch boundaries.
- Avoid per-token object crossing.
- Instrument call counts.

### 23.3 Risk: Error Context Loss

Cause:

- Poor wrapping of JS exception details.

Mitigation:

- Standard wrapper with stack capture.
- Include middleware name and phase.

### 23.4 Risk: Divergent API From Internal Contracts

Cause:

- Inventing JS abstractions that bypass enginebuilder/session semantics.

Mitigation:

- Keep JS API thin over existing seams.
- Do not reimplement orchestration logic in JS.

## 24. Recommended Public API Sketch (Consolidated)

```js
const geppetto = require("geppetto");

const engine = geppetto
  .engineBuilder()
  .provider("openai_responses")
  .profile("4o-mini")
  .use(geppetto.middleware.systemPrompt("You are concise"))
  .use(geppetto.middleware.js((ctx, turn, next) => {
    const out = next(ctx, turn);
    for (const b of out.blocks) {
      if (b.kind === "llm_text" && b.payload?.text) {
        b.payload.text = b.payload.text.trim();
      }
    }
    return out;
  }))
  .tools({ enabled: true, maxIterations: 5, toolChoice: "auto" })
  .build();

const session = geppetto.createSession({ engine });

const turn = geppetto.turn()
  .system("You are concise")
  .user("Use get_weather for Paris and summarize")
  .build();

const out = session.run(turn);
console.log(out.blocks);

const h = session.runAsync(turn);
// h.cancel();
const out2 = await h.wait();
```

## 25. Implementation Notes for First Cut

### 25.1 Keep v1 Scope Focused

Must-have in v1:

- `require("geppetto")` module.
- `createEngine`, `createSession`, `turn` builder.
- JS middleware adapter.
- Wrappers for 2-3 Go middleware constructors.
- Blocking + async run handle.

Defer for v2:

- Full event stream API with async iterator.
- Plugin registry architecture.
- Advanced manifest serialization.

### 25.2 Documentation Requirements

Include in v1 docs:

- Mutability and concurrency semantics.
- Middleware order semantics.
- Error behavior examples.
- Cancellation examples.
- Performance guardrails (do not callback per token with objects).

## 26. Final Recommendation

Adopt a **Goja native module** (`require("geppetto")`) with a **hybrid functional + fluent API** layered over existing `session` + `enginebuilder` + `middleware` + `turns` architecture.

This is the highest-confidence path because:

- It reuses proven internals.
- It supports the requested JS capabilities directly.
- It keeps implementation risk bounded.
- It can be delivered incrementally with strong testability.

## 27. Appendix A: Experiment Commands

```bash
go run geppetto/ttmp/2026/02/12/GP-002-JS-API-GEPPETTO--javascript-api-for-geppetto-inference-and-middleware-composition/scripts/goja_callback_bridge_experiment.go

go run geppetto/ttmp/2026/02/12/GP-002-JS-API-GEPPETTO--javascript-api-for-geppetto-inference-and-middleware-composition/scripts/goja_middleware_composition_experiment.go

go run geppetto/ttmp/2026/02/12/GP-002-JS-API-GEPPETTO--javascript-api-for-geppetto-inference-and-middleware-composition/scripts/goja_eventloop_async_experiment.go

go test ./pkg/inference/... ./pkg/turns/... -count=1
```

## 28. Appendix B: Extended API Variant Sketches

### Variant B1: Strict Functional Core

```js
const builder = geppetto.newBuilder({ provider: "openai_responses" });
const runner = geppetto.buildRunner(builder);
const turn2 = geppetto.run(runner, turn1);
```

Best for minimalism, weaker for discoverability.

### Variant B2: Session-First API

```js
const session = geppetto.session({ provider: "openai_responses", profile: "4o-mini" });
session.use(geppetto.middleware.systemPrompt("..."));
session.use(geppetto.middleware.js(fn));
const out = session.ask("Hello");
```

Ergonomic but can hide engine/session boundaries; acceptable as syntactic sugar only.

### Variant B3: Spec-Driven API

```js
const runtime = geppetto.runtime({
  provider: "openai_responses",
  middlewares: ["systemPrompt", "reorderToolResults"],
  jsMiddlewares: [{ name: "trim", fn: ... }]
});
```

Good for persisted specs; should be additive, not primary surface.

## 29. Appendix C: Concrete Engineering Checklist

- [ ] Create `pkg/js/modules/geppetto/module.go` implementing `modules.NativeModule`.
- [ ] Add JS turn/block codec with validation.
- [ ] Implement JS middleware adapter + error wrapping.
- [ ] Implement engine builder wrapper with Go middleware catalog.
- [ ] Implement session wrapper exposing `run`, `runAsync`, `cancel`, `wait`.
- [ ] Add `require("geppetto")` integration tests.
- [ ] Add mixed middleware ordering tests.
- [ ] Add docs/examples for each API style.

## 30. Closing Note

The requested capabilities are fully achievable without rewriting Geppetto core inference. The project should prioritize a disciplined adapter layer that mirrors internal contracts, then iterate on ergonomics. The experiments in this ticket show the model is viable and clarify where performance and lifecycle guardrails are required.

## 31. Behavioral Contract Deep Dive (Must-Preserve Semantics)

This section spells out implementation contracts observed in source/tests that a JS API must not violate.

### 31.1 Session Contracts

From `pkg/inference/session/session.go` and tests:

- `SessionID` is required for active inference.
- A session can have at most one active inference (`ErrSessionAlreadyActive`).
- Inference runs against latest appended turn.
- Session writes `session_id` and `inference_id` turn metadata.
- If runner returns a different turn pointer, session copies result back into canonical latest turn.

JS consequence:

- JS API must either:
  - preserve single-active behavior and surface a clear error, or
  - internally queue runs explicitly (not currently present in Go code).
- “Run multiple prompts in parallel in one session” should be a documented anti-pattern unless users spawn multiple sessions.

### 31.2 Execution Handle Contracts

From `pkg/inference/session/execution.go`:

- Cancel is idempotent.
- Wait returns final output or error.
- IsRunning reflects completion channel state.

JS consequence:

- Async handles should expose exactly these semantics:
  - `cancel()` multiple times is safe.
  - `wait()` resolves once.
  - `isRunning()` is informative, not transactional.

### 31.3 Middleware Chain Contracts

From `pkg/inference/middleware/middleware.go`:

- Middleware is pure function wrapping handler.
- Chain order is deterministic (reverse wrapping).

JS consequence:

- Must preserve order with mixed middleware list.
- Must not auto-sort middleware by type or priority unless explicitly requested by user.

### 31.4 Toolloop Contracts

From `pkg/inference/toolloop/loop.go`:

- Loop requires non-nil engine + registry.
- Loop writes tool config into turn data (`engine.KeyToolConfig`).
- Loop extracts pending tool calls by block analysis.
- Loop appends tool_use blocks after tool execution.
- Loop stops on no pending calls or max iterations.
- Optional pause points via step controller.

JS consequence:

- JS API should expose loop config and tool config distinctly.
- JS middleware should not be encouraged to mutate loop bookkeeping keys arbitrarily.
- Step/pause API can be optional but should be compatible with current mechanism.

### 31.5 Turn Data/Metadata Contracts

From `pkg/turns/types.go` and `pkg/turns/key_families.go`:

- Typed keys validate JSON serializability on `Set`.
- `Get` returns `(value, ok, error)` and can report type mismatch.
- Wrapper maps are opaque by design.

JS consequence:

- JS helpers should encourage key-safe access.
- Direct raw map writes should remain possible for advanced users, but docs should mark this as “unsafe mode”.

## 32. Detailed JavaScript Contract Proposal (Type Shapes)

Below is a concrete TS-like contract for `require("geppetto")`.

### 32.1 Top-Level Types

```ts
export interface GeppettoModule {
  createEngine(opts: EngineOptions): EngineHandle;
  createSession(opts: SessionOptions): SessionHandle;
  engineBuilder(): EngineBuilder;

  turn(): TurnBuilder;
  block: BlockFactories;

  middleware: MiddlewareCatalog;
  keys: KeyHelpers;
  errors: ErrorHelpers;
}
```

### 32.2 Engine and Session Types

```ts
export interface EngineHandle {
  readonly id: string;
  readonly provider: string;

  run(turn: Turn, opts?: RunOptions): Turn;
  runAsync(turn: Turn, opts?: RunOptions): RunHandle;

  // Optional advanced API
  buildSession(opts?: SessionOptions): SessionHandle;
}

export interface SessionHandle {
  readonly sessionId: string;

  append(turn: Turn): void;
  latest(): Turn | null;

  run(turn?: Turn, opts?: RunOptions): Turn;
  runAsync(turn?: Turn, opts?: RunOptions): RunHandle;
  cancelActive(): void;

  isRunning(): boolean;
}

export interface RunHandle {
  readonly sessionId: string;
  readonly inferenceId: string;

  wait(): Promise<Turn>;
  cancel(): void;
  isRunning(): boolean;

  on(event: "event" | "done" | "error", fn: (payload: any) => void): RunHandle;
}
```

### 32.3 Turn and Block Types

```ts
export interface Turn {
  id?: string;
  blocks: Block[];
  metadata?: Record<string, unknown>;
  data?: Record<string, unknown>;
}

export interface Block {
  id?: string;
  kind: BlockKind;
  role?: string;
  payload?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
}

export type BlockKind =
  | "user"
  | "llm_text"
  | "tool_call"
  | "tool_use"
  | "system"
  | "reasoning"
  | "other";
```

### 32.4 Builder Types

```ts
export interface TurnBuilder {
  id(v: string): TurnBuilder;
  system(text: string): TurnBuilder;
  user(text: string): TurnBuilder;
  assistant(text: string): TurnBuilder;
  toolCall(spec: ToolCallSpec): TurnBuilder;
  toolUse(spec: ToolUseSpec): TurnBuilder;
  block(b: Block): TurnBuilder;
  metadata(k: string, v: unknown): TurnBuilder;
  data(k: string, v: unknown): TurnBuilder;
  build(): Turn;
}

export interface EngineBuilder {
  provider(name: ProviderName): EngineBuilder;
  profile(name: string): EngineBuilder;
  settings(overrides: Record<string, unknown>): EngineBuilder;

  use(mw: MiddlewareHandle): EngineBuilder;
  tools(config: ToolConfig): EngineBuilder;
  toolRegistry(reg: ToolRegistryHandle): EngineBuilder;

  eventSink(sink: EventSinkHandle): EngineBuilder;

  build(): EngineHandle;
}
```

### 32.5 Middleware Types

```ts
export interface MiddlewareCatalog {
  js(fn: JsMiddleware, opts?: JsMiddlewareOptions): MiddlewareHandle;

  // Wrappers for existing Go middleware
  systemPrompt(prompt: string): MiddlewareHandle;
  reorderToolResults(): MiddlewareHandle;
  logging(opts?: LoggingMiddlewareOptions): MiddlewareHandle;
}

export interface MiddlewareHandle {
  readonly name: string;
}

export type JsMiddleware = (
  ctx: RunContext,
  turn: Turn,
  next: (ctx: RunContext, turn: Turn) => Turn
) => Turn;

export interface RunContext {
  sessionId?: string;
  inferenceId?: string;
  traceId?: string;

  // minimal but extensible context
  extra?: Record<string, unknown>;
}
```

### 32.6 Tool Types

```ts
export interface ToolRegistryHandle {
  registerTool(name: string, def: ToolDefinition): void;
  listTools(): ToolDefinition[];
}

export interface ToolDefinition {
  name: string;
  description: string;
  parameters: Record<string, unknown>; // JSON schema
  run: (args: any, ctx?: ToolRunContext) => any | Promise<any>;
}

export interface ToolConfig {
  enabled?: boolean;
  toolChoice?: "auto" | "none" | "required";
  maxIterations?: number;
  executionTimeout?: string; // duration string for JS API, parsed in Go
  maxParallelTools?: number;
  allowedTools?: string[];
  toolErrorHandling?: "continue" | "abort" | "retry";
  retryConfig?: {
    maxRetries?: number;
    backoffBase?: string;
    backoffFactor?: number;
  };
}
```

## 33. Alternative API Styles: Extended Evaluation

This section expands each style with richer examples and failure-mode analysis.

### 33.1 Style A (Functional): Strengths and Weak Points

Strengths:

- Very close to Go internals; easy to maintain.
- Good for scripted workflows and tests.
- Easier to reason about resource ownership.

Weak points:

- Long option objects can become noisy.
- Middleware composition requires manual array assembly and naming discipline.
- Users new to Geppetto may miss available options.

Failure modes:

- Huge “god option” objects with mixed concerns.
- Implicit defaults hidden in implementation.

Mitigations:

- Validate option schema with detailed errors.
- Provide `geppetto.describeDefaults()` helper.

### 33.2 Style B (Fluent): Strengths and Weak Points

Strengths:

- Guides users toward valid combinations.
- Easier onboarding and discoverability.

Weak points:

- Builder state mutation can produce subtle bugs if reused.
- Harder serialization compared to plain config objects.

Failure modes:

- Reusing one builder instance across unrelated engine builds.
- Forgetting `.build()` and passing builder by mistake.

Mitigations:

- Freeze builder after `.build()`.
- Add explicit `clone()` if reuse is intended.
- Strong runtime type checks.

### 33.3 Style C (Declarative): Strengths and Weak Points

Strengths:

- Great for persisted agent definitions.
- Easier for UI-driven workflow editors.

Weak points:

- Executable function components (JS middleware) are hard to serialize safely.
- Configuration/version migrations become mandatory.

Failure modes:

- Loading stale persisted specs after API updates.

Mitigations:

- Add `specVersion` and migration hooks.
- Limit declarative mode to supported deterministic subset.

### 33.4 Style D (Plugin Registry): Strengths and Weak Points

Strengths:

- Scales to ecosystem contributions.

Weak points:

- Requires governance (naming/versioning/loading lifecycle).
- Hard to guarantee deterministic ordering/compatibility.

Failure modes:

- Duplicate plugin names across modules.
- Runtime collisions in shared environments.

Mitigations:

- Prefix namespace convention (`org.plugin`).
- Explicit plugin lockfile/allowlist.

### 33.5 Style E (Event-Driven): Strengths and Weak Points

Strengths:

- Natural fit for live interfaces.
- Hooks into existing event sink ecosystem.

Weak points:

- More complex lifecycle.
- Must document relation to session single-active rule.

Failure modes:

- Attaching event listeners too late and missing early events.
- Unbounded event buffers.

Mitigations:

- Provide `run.on(...)` registration before start in one API call.
- Add bounded buffers and overflow strategies.

## 34. Cross-Language Middleware Assembly Algorithm

This is a concrete algorithm for mixing Go and JS middleware handles.

Input:

- Ordered middleware list from builder.
- Each item tagged as `go` or `js`.

Output:

- `[]middleware.Middleware` consumable by `enginebuilder.WithMiddlewares`.

Algorithm:

1. Iterate middleware list in user order.
2. For `go` handles, append underlying Go middleware directly.
3. For `js` handles, create Go adapter closure:
   - captures runtime pointer,
   - captures callable,
   - captures adapter options.
4. Adapter closure when called:
   - decode `*turns.Turn` -> JS object,
   - create JS `next` closure that invokes downstream Go handler,
   - call JS function,
   - decode JS return -> `*turns.Turn`,
   - return result.
5. Fail fast on decode/encode errors with middleware index/name context.

Pseudocode:

```go
func BuildMixedMiddlewares(specs []MiddlewareSpec, vm *goja.Runtime) ([]middleware.Middleware, error) {
    out := make([]middleware.Middleware, 0, len(specs))
    for i, s := range specs {
        switch s.Kind {
        case SpecKindGo:
            out = append(out, s.GoMW)
        case SpecKindJS:
            mw, err := NewJSMiddlewareAdapter(vm, s.JSCallable, s.Options)
            if err != nil {
                return nil, fmt.Errorf("middleware[%d:%s]: %w", i, s.Name, err)
            }
            out = append(out, mw)
        default:
            return nil, fmt.Errorf("middleware[%d:%s]: unknown kind", i, s.Name)
        }
    }
    return out, nil
}
```

## 35. Turn/Block Manipulation Cookbook

This section gives concrete manipulations expected from users.

### 35.1 Insert System Prompt if Missing

```js
const ensureSystem = geppetto.middleware.js((ctx, turn, next) => {
  const hasSystem = turn.blocks.some(b => b.kind === "system");
  if (!hasSystem) {
    turn.blocks.unshift(geppetto.block.systemText("You are concise"));
  }
  return next(ctx, turn);
});
```

### 35.2 Rewrite Assistant Output

```js
const trimAssistant = geppetto.middleware.js((ctx, turn, next) => {
  const out = next(ctx, turn);
  for (const b of out.blocks) {
    if (b.kind === "llm_text" && typeof b.payload?.text === "string") {
      b.payload.text = b.payload.text.trim();
    }
  }
  return out;
});
```

### 35.3 Filter Tool Calls by Allowlist

```js
const allowed = new Set(["get_weather", "calculator"]);

const enforceToolAllowlist = geppetto.middleware.js((ctx, turn, next) => {
  const out = next(ctx, turn);
  out.blocks = out.blocks.filter(b => {
    if (b.kind !== "tool_call") return true;
    const name = b.payload?.name;
    return typeof name === "string" && allowed.has(name);
  });
  return out;
});
```

### 35.4 Attach Custom Metadata

```js
const traceMw = geppetto.middleware.js((ctx, turn, next) => {
  turn.metadata = turn.metadata || {};
  turn.metadata["acme.trace@v1"] = ctx.traceId || "none";
  return next(ctx, turn);
});
```

### 35.5 Normalize Tool Result Placement

Most users should prefer wrapped Go middleware:

```js
.use(geppetto.middleware.reorderToolResults())
```

This avoids reimplementing delicate adjacency rules in JS.

## 36. Tool Registry from JS: Design Options

### 36.1 Option A: JS-Defined Tools Backed by Go Wrapper

JS defines runtime function and schema; adapter wraps into `tools.ToolDefinition`.

Pros:

- Very flexible.

Cons:

- Async tool functions need careful event loop integration.

### 36.2 Option B: Go-Defined Tools Referenced by Name in JS

JS only selects from pre-registered Go tools.

Pros:

- Stronger control and safety.

Cons:

- Less dynamic scripting capability.

### 36.3 Option C: Hybrid

- Allow both.
- Default to Go-registered tools in production contexts.
- Enable JS-defined tools for local scripting.

Recommendation: Hybrid with policy gates.

## 37. Debug and Observability Hooks

The engine already supports `DebugTap` via context (`pkg/inference/engine/debugtap.go`).

JS API can expose this as:

```js
const run = session.runAsync(turn, {
  debugTap: {
    onHttp(req) { ... },
    onHttpResponse(resp) { ... },
    onSse(event, data) { ... },
    onProviderObject(name, data) { ... },
    onTurnBeforeConversion(yaml) { ... }
  }
});
```

Implementation note:

- Wrap JS callbacks in non-fatal best-effort invocations.
- Never let debug callback errors break inference by default.

## 38. Deployment Modes and API Guidance

### 38.1 Embedded CLI Script Mode

Characteristics:

- Short-lived process.
- Single runtime.
- Human-driven execution.

API guidance:

- Prefer simple blocking APIs.
- Minimal event streaming.

### 38.2 Long-Lived Service Mode

Characteristics:

- Many sessions.
- Long runtime lifetime.

API guidance:

- Prefer explicit session lifecycle management.
- Provide cleanup/dispose methods.
- Guard against middleware accumulation leaks.

### 38.3 Multi-Tenant/Hosted Mode (Future)

Characteristics:

- Untrusted script potential.
- Resource isolation needs.

API guidance:

- Restrict host API surface.
- Enforce quotas/timeouts.
- Audit plugin and tool registration.

## 39. Validation Matrix (Exhaustive)

This matrix should be converted into actual tests during implementation.

### 39.1 Engine/Session Lifecycle

- create engine with valid provider.
- create engine with invalid provider -> error.
- create session and run single turn.
- runAsync wait success.
- runAsync cancel before completion.
- run while active -> `already active` error.

### 39.2 Middleware Composition

- single JS middleware transform.
- multiple JS middlewares order validation.
- mixed Go+JS order validation.
- JS middleware throws.
- JS middleware returns invalid shape.
- JS middleware omits return.

### 39.3 Turn/Block Manipulation

- user/system/assistant builder functions.
- tool_call/tool_use helper creation.
- reasoning block preservation.
- metadata/data roundtrip.
- invalid block kind handling.

### 39.4 Toolloop Integration

- tool call extraction and execution.
- max iteration reached error.
- allowed tools filtering.
- tool error handling modes (`continue`, `abort`, `retry`).

### 39.5 Async and Events

- promise resolve path.
- promise reject path.
- cancellation reject path.
- event listener receives start/partial/final/error in order.

### 39.6 Performance Regression Checks

- boundary call count per run.
- mean run latency with N middleware layers.
- object conversion overhead sanity thresholds.

## 40. Rollout and Governance Model

### 40.1 API Stability Levels

- `stable`: core `createEngine/createSession/turn` and Go middleware wrappers.
- `beta`: JS middleware adapter semantics.
- `experimental`: async iterator streaming and plugin registry.

### 40.2 Versioning Rules

- LowerCamel key policy enforced for JS-facing options.
- Breaking shape changes require explicit migration notes.
- Add compatibility test fixtures for major APIs.

### 40.3 Documentation Ownership

- API reference with examples.
- behavior contracts section (single-active session, mutation semantics).
- troubleshooting guide (errors at JS↔Go boundary).

## 41. Concrete First Implementation Backlog (Suggested)

1. `pkg/js/modules/geppetto/module.go` with basic exports.
2. `pkg/inference/jsbridge/codec.go` for `Turn`/`Block` conversion.
3. `pkg/inference/jsbridge/middleware_js.go` for adapter.
4. `pkg/inference/jsbridge/errors.go` for exception wrapping.
5. `pkg/js/modules/geppetto/api_engine.go` + `api_session.go` wrappers.
6. Integration test package:
   - load module,
   - run fake engine,
   - verify mixed middleware.

## 42. Additional Example: End-to-End Composition Script

```js
const geppetto = require("geppetto");

function redactEmailsMw(ctx, turn, next) {
  const out = next(ctx, turn);
  const emailRE = /[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/ig;

  for (const b of out.blocks) {
    if (b.kind === "llm_text" && typeof b.payload?.text === "string") {
      b.payload.text = b.payload.text.replace(emailRE, "[redacted-email]");
    }
  }
  return out;
}

const engine = geppetto.engineBuilder()
  .provider("openai_responses")
  .profile("4o-mini")
  .use(geppetto.middleware.systemPrompt("You are a compliance-safe assistant"))
  .use(geppetto.middleware.js(redactEmailsMw))
  .use(geppetto.middleware.reorderToolResults())
  .tools({ enabled: true, maxIterations: 4, toolChoice: "auto" })
  .build();

const session = geppetto.createSession({ engine });

let turn = geppetto.turn()
  .user("Ask get_weather for Paris and include contact: test@example.com")
  .build();

turn = session.run(turn);
print(JSON.stringify(turn, null, 2));
```

## 43. Why the Recommendation Is Defensible

The recommendation is defensible because it follows existing system boundaries instead of fighting them:

- Session already encodes concurrency/lifecycle invariants.
- Engine builder already assembles middleware/tools/events.
- Middleware already has a clean compositional contract.
- Turn/block data model already supports rich transformations.

The design effort should therefore optimize for adapter quality, error clarity, and ergonomics, not architectural replacement.

## 44. Decision Summary

- **Use native module entrypoint**: `require("geppetto")`.
- **Adopt hybrid API style**: functional core + fluent builder + optional event wrapper.
- **Expose both Go and JS middleware in one ordered pipeline**.
- **Treat turn/block as canonical JS domain objects with helpers**.
- **Preserve session/toolloop invariants exactly**.
- **Enforce lowerCamel JS key naming**.
- **Plan async/streaming with event-loop-safe promise resolution**.

This provides the requested feature set while keeping risk and migration complexity manageable.

## 45. Extended Implementation Blueprint (Concrete Work Packages)

This section translates the recommendation into concrete engineering slices that can be staffed and reviewed independently.

### 45.1 Work Package A: Module Bootstrap

Deliverables:

- New package: `pkg/js/modules/geppetto`.
- `module.go` implementing `modules.NativeModule`.
- `Doc()` returning concise API synopsis.
- `Loader()` wiring stable exports.

Acceptance checks:

- `require("geppetto")` succeeds in runtime integration test.
- `Object.keys(require("geppetto"))` contains expected core exports.
- No inference logic in loader itself.

### 45.2 Work Package B: Codec Layer

Deliverables:

- `pkg/inference/jsbridge/codec_turn.go`.
- `pkg/inference/jsbridge/codec_block.go`.
- `pkg/inference/jsbridge/codec_meta.go`.

Codec contract:

- `DecodeTurn(vm, v goja.Value) (*turns.Turn, error)`
- `EncodeTurn(vm, t *turns.Turn) (goja.Value, error)`

Validation requirements:

- Reject invalid block kind values with precise errors.
- Reject non-object payload/metadata/data where object is required.
- Keep unknown keys intact for forward compatibility.

Edge-case requirements:

- Missing `blocks` -> decode as empty slice or error (choose and document).
- `null` payload -> convert to nil map.
- Non-JSON-serializable values should fail before crossing into typed key wrappers.

### 45.3 Work Package C: JS Middleware Adapter

Deliverables:

- `pkg/inference/jsbridge/middleware_js.go`.
- Adapter options struct for future extension:

```go
type JSMiddlewareOptions struct {
    Name          string
    CloneBefore   bool
    Timeout       time.Duration
    RecoverPanics bool
}
```

Core adapter behavior:

- Convert Go turn to JS object.
- Build JS `next` callable bridging to Go chain.
- Invoke JS callable.
- Convert returned value back to Go turn.
- Wrap any JS exception with middleware name + stack.

Timeout strategy (optional v1.1):

- For long-running JS middleware in service mode, support timeout enforcement.
- If timeout occurs, return decorated error and cancel context.

### 45.4 Work Package D: Engine Builder JS Wrapper

Deliverables:

- `api_engine_builder.go`.
- `api_middleware_catalog.go`.
- `api_tools.go`.

Behavior:

- Hold temporary builder state in Go struct referenced by JS object.
- `build()` emits immutable engine handle.
- Attempting to mutate builder after build should return explicit error.

Error examples to surface:

- Missing provider.
- Invalid tool choice string.
- Unresolvable middleware handle.

### 45.5 Work Package E: Session Wrapper

Deliverables:

- `api_session.go`.
- `api_run_handle.go`.

Core methods:

- `append(turn)`
- `latest()`
- `run(turn?)`
- `runAsync(turn?)`
- `cancelActive()`
- `isRunning()`

Async handle details:

- Keep Go `*session.ExecutionHandle` reference.
- `wait()` returns JS promise.
- `cancel()` delegates to handle cancel.
- Promise resolution/rejection scheduled on loop.

### 45.6 Work Package F: Event Exposure

Deliverables:

- `api_events.go`.
- Minimal run-event emitter wrapper.

Pattern:

- Attach optional event sink at run invocation.
- Forward events to JS callback list.
- Keep bounded queue to avoid runaway memory.

Potential queue policy options:

- `drop_oldest`
- `drop_newest`
- `block` (least safe in embedded mode)

Recommendation:

- default `drop_oldest` with counter metric.

### 45.7 Work Package G: Integration and Conformance Tests

Test layers:

1. Codec unit tests.
2. Middleware adapter unit tests.
3. Runtime integration tests (`require("geppetto")`).
4. End-to-end fake engine tests.
5. Optional provider-backed smoke tests in CI with secrets.

Mandatory golden tests:

- Mixed middleware ordering golden output.
- Error path stack-preservation golden output.
- Async cancel golden output.

## 46. Migration Scenarios and Compatibility Notes

### 46.1 Scenario A: Existing Go App Adds JS Middleware

Starting point:

- Existing Go code builds engine/session with Go middleware only.

Migration path:

1. Initialize Goja runtime and load module.
2. Expose selected Go middleware wrappers in JS.
3. Register one JS middleware function.
4. Use mixed middleware builder from JS.

Compatibility concern:

- Ensure existing Go middleware ordering remains stable when JS middleware inserted.

### 46.2 Scenario B: Script-Only Runtime

Starting point:

- User wants to script everything from JS in embedded runtime.

Migration path:

1. Use `require("geppetto")` only.
2. Build engine/session from JS.
3. Register tools and middleware from JS.

Compatibility concern:

- Error messages must be clear enough for non-Go users.
- Need high-quality docs for provider settings and defaults.

### 46.3 Scenario C: Gradual Replacement of Ad-Hoc JS Wrappers

Starting point:

- Existing `pkg/js/embeddings-js.go` style wrappers.

Migration path:

1. Introduce `geppetto` module next to existing wrappers.
2. Mark old wrappers as legacy in docs.
3. Migrate examples to module-first style.
4. Deprecate ad-hoc globals after one release cycle.

Compatibility concern:

- Keep old wrappers working until docs and examples are migrated.

### 46.4 Scenario D: Hosted Product with Persisted Agent Configs

Starting point:

- Need stable serialized model of workflows.

Migration path:

1. Use fluent builder for runtime assembly.
2. Add export/import of subset declarative spec.
3. Add `specVersion` and migration tooling.

Compatibility concern:

- JS function-valued middleware cannot be serialized naively.
- Need policy for “named middleware references” in persisted specs.

## 47. Advanced Error Taxonomy Proposal

Introduce structured error codes at JS boundary:

- `GEPPETTO_JS_DECODE_ERROR`
- `GEPPETTO_JS_ENCODE_ERROR`
- `GEPPETTO_JS_MIDDLEWARE_THROW`
- `GEPPETTO_JS_MIDDLEWARE_TIMEOUT`
- `GEPPETTO_JS_INVALID_OPTION`
- `GEPPETTO_JS_SESSION_ACTIVE`
- `GEPPETTO_JS_RUN_CANCELED`

Error object shape:

```ts
interface GeppettoError extends Error {
  code: string;
  phase?: "decode" | "middleware" | "engine" | "encode" | "session";
  middlewareName?: string;
  cause?: unknown;
}
```

Benefits:

- Easier programmatic handling in JS.
- Easier support and issue triage.

## 48. Performance Budget Proposal

Define rough budgets for embedded production usage (tunable by app):

- JS middleware layers per run: target <= 8.
- JS↔Go object crossings per run: target <= 200 for non-streaming workloads.
- Event callback rate:
  - token-level: discouraged by default,
  - chunk-level: recommended.

Add debug counters:

- `bridge.calls.scalar`
- `bridge.calls.object`
- `bridge.turn.decodes`
- `bridge.turn.encodes`
- `bridge.middleware.invocations`

Expose via debug API:

```js
const stats = geppetto.debug.bridgeStats();
```

## 49. Documentation Plan for Adoption

Recommended docs set:

1. Quickstart: 5-minute setup.
2. Core concepts: engine/session/turn/middleware.
3. Middleware cookbook (10 common patterns).
4. Tools cookbook.
5. Async and cancellation guide.
6. Troubleshooting guide (with error code table).
7. Performance guide (boundary-crossing dos/donts).

Each cookbook recipe should include:

- Problem statement.
- Minimal snippet.
- “Why this works with Geppetto internals.”
- Pitfalls.

## 50. Long-Horizon Evolution (Post-v1)

Potential future upgrades once core is stable:

- Native async iterator stream support.
- Declarative runtime graphs with validated schemas.
- Middleware capability descriptors (e.g., “reads metadata”, “mutates blocks”).
- Sandbox policy DSL for hosted multi-tenant use.
- Plugin signature verification for third-party modules.

These should remain secondary to stabilizing v1 adapter semantics.

## 51. Final Acceptance Checklist for This Design

The design should be considered complete/implementable when reviewers agree that:

- Core API shapes map directly to existing Go seams.
- Mixed middleware semantics are unambiguous.
- Error and cancellation behavior are explicit.
- Turn/block manipulation is ergonomic but safe.
- Performance guidance is backed by measured data.
- Rollout path is incremental and testable.

This document satisfies that bar and can be used directly to create implementation tickets.

## 52. Glossary and Terminology Alignment

To reduce ambiguity across implementation and docs, these terms should be used consistently.

- **Engine**: A provider-specific inference executor implementing `RunInference(ctx, *turns.Turn) (*turns.Turn, error)`.
- **Session**: A long-lived runtime object that owns `SessionID`, turn history, and active run exclusivity.
- **Inference**: One execution of a session runner against the latest turn (or explicitly supplied turn).
- **ExecutionHandle / RunHandle**: Cancelable/waitable object representing one in-flight inference.
- **Turn**: Ordered container of blocks plus turn-scoped metadata and data maps.
- **Block**: Atomic unit in a turn (user/system/assistant text, tool call/use, reasoning, other).
- **Middleware**: A function wrapper around the inference handler that can inspect/transform input or output turn state.
- **Toolloop**: Orchestration layer that iterates between model inference and tool execution until completion/limit.
- **Tool Registry**: Named catalog of callable tool definitions.
- **Tool Config**: Policy and limits governing tool calling behavior.
- **Event Sink**: Destination for structured runtime events.
- **Debug Tap**: Low-level optional observer for request/response/provider diagnostics.
- **JS Bridge**: Adapter layer that converts between Go runtime values and JS runtime values and vice versa.
- **Native Module**: A Goja module registered via `modules.NativeModule` and imported from JS via `require()`.

Terminology rules for docs and code comments:

1. Use **session** for lifecycle/concurrency context and **engine** for provider execution logic.
2. Use **turn mutation** explicitly when data is modified in place.
3. Use **adapter** for cross-language glue and **service/domain** for core business logic.
4. Use **blocking run** vs **async run** consistently; avoid vague terms like "start" without lifecycle details.
5. Use **middleware ordering** phrasing tied to the actual chain rule: first listed middleware is outermost wrapper.

Naming policy for JS-facing API keys:

- Use lowerCamel for all options and return objects (for example `maxIterations`, `toolChoice`, `executionTimeout`).
- Keep enum values lowercase snake or lowercase strings aligned with existing Go string constants where possible (`"auto"`, `"required"`, `"continue"`).
- Reserve PascalCase for constructor-like functions only if absolutely necessary (recommended: avoid).

By enforcing this vocabulary early, implementation, tests, and docs remain aligned, and downstream adopters avoid category errors when moving between Go and JS layers.

## 53. V2 Update: Codegen-Backed Turn/Block Contract (GP-003 Knowledge)

This section incorporates concrete implementation knowledge from ticket `GP-003-JS-TURNS-CODEGEN`.

### 53.1 What Changed

Turn/block mapper code is now generated from a schema:

- Schema source: `pkg/turns/spec/turns_codegen.yaml`
- Generator: `cmd/gen-turns/main.go`
- Generated outputs:
  - `pkg/turns/block_kind_gen.go`
  - `pkg/turns/keys_gen.go`

This moves BlockKind string mapping and typed key ID constants from handwritten code to generated code.

### 53.2 Why It Matters for JS API

The JS bridge should stop hardcoding key/kind strings in multiple places and instead rely on the generated mappers as canonical conversion tables.

Practical implications:

1. JS decode paths become table-driven and less error-prone.
2. JS encode paths preserve canonical naming consistently.
3. Adding new block kinds or metadata/data keys is now a schema + generate operation, not manual edits across multiple files.

### 53.3 Hybrid Model for Turns/Blocks (Implementation-Side Clarification)

The hybrid model means:

- **External JS contract** stays plain-object first for ergonomics.
- **Internal Go contract** stays strongly typed (`turns.Turn`, `turns.Block`, typed keys).
- **Bridge codec** is responsible for deterministic projection between the two.

The generated mappers become the stable seam between plain JS string fields and internal typed constants.

### 53.4 Pseudocode: Decode Turn (JS -> Go)

```go
func decodeTurn(js any) (*turns.Turn, error) {
    obj := mustObject(js)
    t := &turns.Turn{
        ID:       strOrEmpty(obj["id"]),
        Blocks:   []turns.Block{},
        Metadata: map[string]any{},
        Data:     map[string]any{},
    }

    for _, b := range asArray(obj["blocks"]) {
        bo := mustObject(b)

        // kind mapping uses generated canonical mapper
        k := turns.BlockKindFromString(strOrEmpty(bo["kind"])) // generated
        if k == "" {
            k = turns.BlockKindOther // generated fallback
        }

        block := turns.Block{
            Kind:     k,
            Content:  asStringPtr(bo["content"]),
            Role:     asStringPtr(bo["role"]),
            ID:       strOrEmpty(bo["id"]),
            Metadata: decodeBlockMeta(bo["metadata"]), // keyed via generated block-meta keys
            Data:     decodeData(bo["data"]),          // keyed via generated data keys
        }
        t.Blocks = append(t.Blocks, block)
    }

    t.Metadata = decodeTurnMeta(obj["metadata"]) // generated turn-meta keys
    t.Data = decodeData(obj["data"])             // generated data keys
    return t, nil
}
```

### 53.5 Pseudocode: Encode Turn (Go -> JS)

```go
func encodeTurn(t *turns.Turn) map[string]any {
    out := map[string]any{
        "id":       t.ID,
        "blocks":   []any{},
        "metadata": encodeTurnMeta(t.Metadata), // generated turn-meta keys
        "data":     encodeData(t.Data),         // generated data keys
    }

    blocks := out["blocks"].([]any)
    for _, b := range t.Blocks {
        bo := map[string]any{
            "id":       b.ID,
            "kind":     b.Kind.String(), // generated canonical string mapping
            "content":  ptrToAny(b.Content),
            "role":     ptrToAny(b.Role),
            "metadata": encodeBlockMeta(b.Metadata), // generated block-meta keys
            "data":     encodeData(b.Data),          // generated data keys
        }
        blocks = append(blocks, bo)
    }
    out["blocks"] = blocks
    return out
}
```

### 53.6 Mapper Codegen Lifecycle and Extension Policy

When adding a new block kind or key family entry:

1. Update `pkg/turns/spec/turns_codegen.yaml`.
2. Run `go generate ./pkg/turns`.
3. Run tests (`go test ./cmd/gen-turns ./pkg/turns/...`).
4. Update JS-facing docs/examples if externally visible.

Policy:

- Generated files own constant names/values for block kinds and key IDs.
- Handwritten files keep only non-generated business constants.
- JS API should reject unknown required keys but tolerate unknown optional keys under `other`/passthrough paths when safe.

### 53.7 Failure Modes and Guardrails

Primary risks:

- JS bridge assuming stale key/kind constants after schema changes.
- Handwritten string comparisons bypassing generated mappers.

Guardrails:

- Add codec tests that enumerate generated key/kind tables.
- Add a CI check that fails if `go generate ./pkg/turns` changes tracked files.
- Keep bridge decode/encode functions centralized (single source of conversion truth).

## 54. V2 Update: JS Tool Registration, ToolLoop Configuration, and Go Tool Interop

This section adds a concrete design for:

1. Registering JS tools from JS.
2. Configuring and enabling toolloop from JS.
3. Calling registered tools from JS, including tools registered on the Go side.

### 54.1 Requirements

- A single effective registry must be usable by toolloop regardless of tool origin (Go or JS).
- JS must be able to:
  - define new tools,
  - reference existing Go tools by name,
  - run direct tool calls outside inference for testing.
- Go callers must be able to execute JS-defined tools through the same registry contract.

### 54.2 Proposed JS API Surface

```js
const gp = require("geppetto");

const reg = gp.tools.createRegistry();

// register JS-native tool
reg.register({
  name: "js.normalize_text",
  description: "Normalize whitespace and casing",
  parameters: {
    type: "object",
    properties: { text: { type: "string" } },
    required: ["text"]
  },
  handler: async ({ text }) => ({ value: text.trim().toLowerCase() })
});

// expose existing Go tools into same registry view
reg.useGoTools(["weather.lookup", "calendar.create_event"]);

// direct invocation path from JS (works for JS or Go-backed tools)
const probe = await reg.call("weather.lookup", { city: "Berlin" });

const engine = gp.engines.openai({ model: "gpt-4.1-mini" });
const session = gp.createSession({
  engine,
  tools: reg,
  toolLoop: {
    enabled: true,
    maxIterations: 6,
    toolChoice: "auto",
    maxParallelTools: 3
  }
});
```

### 54.3 Registry Composition Model

Implementation model:

- `ToolRegistryBridge` wraps:
  - a Go registry (`tools.ToolRegistry`) for Go-defined tools,
  - a JS tool table for JS handlers,
  - optional name aliasing and collision policy.
- `GetTool(name)` resolves in deterministic order:
  1. explicit JS override namespace (if enabled),
  2. Go registry,
  3. JS registry default scope.
- Resolved tool is returned as a normal `tools.ToolDefinition` so existing `tools.Executor` and `toolloop` remain unchanged.

Collision policy recommendation:

- default deny on duplicate names across origins,
- allow explicit override with `reg.register(..., { replace: true })`,
- encourage prefixes (`js.` / `go.`) in mixed deployments.

### 54.4 Pseudocode: Bridge Registry (JS + Go)

```go
type ToolRegistryBridge struct {
    goRegistry tools.ToolRegistry
    jsTools    map[string]jsToolSpec // schema + goja callable
}

func (r *ToolRegistryBridge) RegisterJSTool(name string, spec jsToolSpec) error {
    if name == "" { return errNameEmpty }
    if _, exists := r.jsTools[name]; exists { return errDuplicate }
    r.jsTools[name] = spec
    return nil
}

func (r *ToolRegistryBridge) GetTool(name string) (*tools.ToolDefinition, error) {
    if def, err := r.goRegistry.GetTool(name); err == nil {
        return def, nil
    }
    jsSpec, ok := r.jsTools[name]
    if !ok {
        return nil, fmt.Errorf("tool not found: %s", name)
    }
    return buildToolDefinitionFromJS(name, jsSpec), nil
}
```

### 54.5 Pseudocode: Direct Call From JS to Registered Tool

```go
func (api *ToolsAPI) Call(name string, argsJSON []byte) (any, error) {
    call := tools.ToolCall{ID: uuid.NewString(), Name: name, Arguments: argsJSON}
    exec := tools.NewDefaultToolExecutor(api.toolCfg)
    res, err := exec.ExecuteToolCall(api.ctx, call, api.registry) // registry may resolve to Go or JS
    if err != nil { return nil, err }
    if res == nil { return nil, fmt.Errorf("missing tool result") }
    if res.Error != "" { return nil, fmt.Errorf(res.Error) }
    return res.Result, nil
}
```

### 54.6 Pseudocode: Configure ToolLoop From JS

```go
func sessionFromJS(opts SessionOptions) (*session.Session, error) {
    b := &enginebuilder.Builder{
        Base:        opts.Engine,
        Middlewares: opts.Middlewares,
    }
    if opts.Tools != nil {
        b.Registry = opts.Tools.Registry
        b.ToolConfig = decodeToolConfig(opts.ToolLoop)
        b.LoopConfig = &toolloop.LoopConfig{
            MaxIterations: opts.ToolLoop.MaxIterations,
        }
    }
    return session.New(b), nil
}
```

### 54.7 Pseudocode: Go Caller Invoking a JS-Defined Tool

```go
func callPossiblyJSTool(ctx context.Context, registry tools.ToolRegistry) error {
    exec := tools.NewDefaultToolExecutor(tools.DefaultToolConfig())
    call := tools.ToolCall{
        ID: "manual-1",
        Name: "js.normalize_text",
        Arguments: []byte(`{"text":"  A  B  "}`),
    }
    res, err := exec.ExecuteToolCall(ctx, call, registry)
    if err != nil { return err }
    if res.Error != "" { return fmt.Errorf(res.Error) }
    // res.Result should contain JS handler output, marshaled through bridge
    return nil
}
```

### 54.8 Security and Policy Constraints

- Enforce `ToolConfig.AllowedTools` before execution for both JS and Go origins.
- Apply timeout/cancellation consistently using context propagation in JS callable wrapper.
- Validate JSON args against declared tool schema before handler execution.
- Emit tool start/result events for JS tools exactly as for Go tools.
- Optionally gate JS direct `reg.call()` in production mode to reduce misuse.

### 54.9 Incremental Implementation Tasks

1. Add JS-facing `tools.createRegistry()` and `register/useGoTools/call/list` API.
2. Implement `ToolRegistryBridge` that satisfies `tools.ToolRegistry`.
3. Add JS tool wrapper -> `tools.ToolDefinition` conversion.
4. Wire builder/session options to attach registry + loop config from JS.
5. Add mixed-registry integration tests:
   - JS tool called by model through toolloop,
   - Go tool called directly from JS via `reg.call`,
   - Go executor calling JS-defined tool by name,
   - collision and allowlist policy enforcement.

This closes the remaining design gap: tools become first-class, bidirectional, and origin-agnostic across JS and Go.
