---
Title: Geppetto go-go-goja API Review and Builder Design Guide
Ticket: GP-GOJA-API-2026-06-01
Status: active
Topics:
    - geppetto
    - js-bindings
    - goja
    - inference
    - intern-onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/js/modules/geppetto/api_runner.go
      Note: Current runtime/runner split and prepared run flow reviewed for agent facade design
    - Path: geppetto/pkg/js/modules/geppetto/api_sessions.go
      Note: Current builder/session/run handle implementation that new fluent API should reuse
    - Path: geppetto/pkg/js/modules/geppetto/api_tools_registry.go
      Note: Current JS and Go tool registry bridge reviewed for typed tool builder design
    - Path: geppetto/pkg/js/modules/geppetto/codec.go
      Note: Current loose JS map to Go turn/block codec and target for strict builder validation
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Native module export wiring and runtime state reviewed for builder API design
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: xgoja runtime module creation reviewed for Geppetto provider host-service integration
    - Path: pinocchio/pkg/chatapp/runtime_inference.go
      Note: Downstream application runtime consumption reviewed to shape agent/event API requirements
ExternalSources: []
Summary: Review of the current Geppetto go-go-goja JavaScript binding architecture and a phased design for an opinionated Go-backed fluid builder API.
LastUpdated: 2026-06-01T10:26:25.090988456-04:00
WhatFor: Use when redesigning require("geppetto") for xgoja/goja scripts, standalone LLM apps, agents, and embeddings workflows.
WhenToUse: Before implementing new Geppetto JS bindings or changing the builder, runner, engine, turn, tool, profile, xgoja provider, or embedding APIs.
---


# Geppetto go-go-goja API Review and Builder Design Guide

## Executive Summary

The current `require("geppetto")` module already contains many of the right building blocks for an LLM scripting API: it is a native goja module, exposes Go-backed references for engines/builders/sessions/tool registries, has deterministic tests, ships generated TypeScript declarations, and has examples that walk from turns to sessions, middleware, tools, profiles, and streaming handles. The good news is that this is not a greenfield rewrite. The runtime has enough structure to evolve toward a safer and more elegant API.

The main design problem is that the JavaScript-facing surface still looks like a collection of dynamic map adapters around Go internals. Turns, blocks, profile options, tool parameters, runtime settings, and event payloads are mostly accepted as untyped JavaScript objects and decoded later. That makes scripts easy to start, but it weakens the central goal for this ticket: most construction should happen on the Go side so Geppetto can enforce strong typing, runtime validation, canonical defaults, provider-specific invariants, and stable object identity.

This document proposes a staged redesign centered on a **Go-backed fluid builder API**. JavaScript should primarily compose opaque Go-owned handles such as `gp.turn().user(...).system(...).build()`, `gp.agent().profile(...).system(...).tools(...).run(...)`, `gp.tool(...).schema(...).handler(...)`, and `gp.embeddings().profile(...).embed(...)`. Plain JavaScript objects should remain available as escape hatches, but not as the preferred construction path. The target API should feel like JavaScript while behaving like a typed Go SDK.

## Problem Statement and Scope

### Requested outcome

The user asked for a new ticket to examine and improve the current Geppetto go-go-goja API and JS bindings. The desired end state is:

- an elegant, fluid builder-based API;
- most operations and object construction performed on the Go side;
- strong typing and runtime validation instead of relying on JavaScript maps;
- an opinionated, customizable LLM inference API for agents, LLM scripts, and embeddings scripts;
- good support for xgoja standalone applications that bundle modules;
- an intern-facing analysis/design/implementation guide with prose, bullets, diagrams, pseudocode, API references, and file references;
- ticket storage, diary maintenance, and reMarkable delivery.

### In scope

This review covers:

- `geppetto/pkg/js/modules/geppetto`: the current native `require("geppetto")` module.
- `geppetto/pkg/js/runtime`: the Geppetto runtime wrapper over go-go-goja.
- `geppetto/pkg/js/embeddings-js.go`: the older embedding wrapper that should inform the new embedding API.
- `geppetto/pkg/doc/types/geppetto.d.ts` and docs/examples for the public contract.
- `go-go-goja/pkg/xgoja`: provider/module mechanics for generated standalone runtimes.
- `pinocchio/pkg/inference/runtime` and `pinocchio/pkg/chatapp`: downstream runtime composition and UI/event usage that should shape API requirements.

### Out of scope for the first implementation pass

- Replacing Geppetto inference engines.
- Rewriting provider event vocabularies.
- Replacing the Pinocchio webchat architecture.
- Removing all map-based compatibility immediately.
- Implementing browser TypeScript UI bindings.

## Current-State Architecture

### High-level system map

```text
+------------------------- JavaScript script --------------------------+
| const gp = require("geppetto")                                      |
| turns / engines / profiles / runner / middlewares / tools / events  |
+-----------------------------------+---------------------------------+
                                    |
                                    v
+------------------------- Geppetto goja module -----------------------+
| pkg/js/modules/geppetto                                             |
| - module exports                                                     |
| - Go-backed refs via hidden __geppetto_ref                           |
| - JS object decode/encode codecs                                     |
| - builder/session/runner/tool/profile/event adapters                 |
+-----------------------------------+---------------------------------+
                                    |
                                    v
+-------------------------- Geppetto Go runtime -----------------------+
| inference engines, sessions, toolloop enginebuilder, middleware,     |
| events, turns, profiles, tools                                       |
+-----------------------------------+---------------------------------+
                                    |
                                    v
+---------------------- Hosts: lab, xgoja, Pinocchio ------------------+
| - geppetto JS runtime helper                                         |
| - xgoja generated binaries and provider modules                      |
| - Pinocchio runtime builder and chatapp event sink                   |
+---------------------------------------------------------------------+
```

### Native module installation

`geppetto/pkg/js/modules/geppetto/module.go` defines the core module name, options, runtime state, and export installation. The module name is hard-coded as `geppetto`, and `Register` attaches a native module loader to a `goja_nodejs/require.Registry`.

Key observed files:

- `module.go`: `ModuleName`, `Options`, `NewLoader`, `Register`, `moduleRuntime`, and `installExports`.
- `provider/provider.go`: xgoja provider wrapper for the Geppetto module.
- `runtime/runtime.go`: convenience runtime builder that registers `require("geppetto")`.

Evidence:

- `module.go` exposes top-level namespaces and functions: `createBuilder`, `createSession`, `runInference`, `turns`, `engines`, `profiles`, `runner`, `schemas`, `middlewares`, `events`, and `tools`.
- `runtime/runtime.go` builds a go-go-goja runtime with implicit defaults disabled and registers the Geppetto module as a runtime module.
- `provider/provider.go` defines xgoja provider config fields: `profile`, `registry`, `allowNetwork`, and `allowTools`.

### Go-backed references

The strongest part of the current design is the hidden reference mechanism. Objects returned to JavaScript can carry non-enumerable Go references in `__geppetto_ref`, and later APIs can recover those references instead of trusting a serialized JavaScript map.

```text
Go engineRef/toolRegistryRef/sessionRef/builderRef
        |
        v
JS object with non-enumerable __geppetto_ref
        |
        v
later API call recovers Go pointer using getRef(...)
```

This is exactly the pattern the redesign should expand. Today it is used for objects such as engines, builders, sessions, resolved profiles, runner runtimes, prepared runs, and tool registries. The main gap is that turns, blocks, model options, embedding requests, and many nested settings are still primarily map-shaped values.

### Current JS namespaces

The generated declarations in `geppetto/pkg/doc/types/geppetto.d.ts` document the public surface. The current module exposes:

- `turns`: normalize and construct turn/block maps.
- `engines`: create echo/config/profile/function engines.
- `profiles`: list, resolve, connect, and disconnect engine profile registries.
- `runner`: resolve runtime metadata, prepare runs, run synchronously, start streaming.
- `schemas`: list middleware and extension schemas.
- `middlewares`: create JS or named Go middleware references.
- `events`: create collectors.
- `tools`: create registries.
- top-level `createBuilder`, `createSession`, `runInference`.

This is a reasonable taxonomy, but it splits common user intent across too many namespaces. A user who wants “ask a model with a system prompt and a tool” must understand profiles, engines, runner runtimes, sessions, tools, turns, and event collectors before they can write the final script.

### Current construction flow

A typical current script looks like this:

```javascript
const gp = require("geppetto");

const engine = gp.engines.fromConfig({
  apiType: "openai",
  model: "gpt-4.1-mini",
  apiKey: ENV.OPENAI_API_KEY,
});

const runtime = gp.runner.resolveRuntime({
  systemPrompt: "Answer tersely.",
  runtimeKey: "demo",
});

const out = gp.runner.run({
  engine,
  runtime,
  prompt: "hello",
});
```

Internally this is converted to Go engines, middleware, sessions, turns, and enginebuilder options. The flow works, but the JavaScript API makes the user assemble conceptual pieces that Geppetto could assemble more safely with typed builders.

### Current xgoja integration

`go-go-goja` has an xgoja framework for generated standalone binaries. The important pieces are:

- `providerapi.Module` declares a module name, default alias, config schema, and factory.
- `app.RuntimeFactory.NewRuntime` reads a runtime profile, resolves provider modules, builds go-go-goja runtime modules, and registers module aliases.
- Commands such as `eval`, `run`, `repl`, and `jsverbs` execute JavaScript against a selected runtime profile.

The Geppetto provider already follows the xgoja provider style, but there is an integration risk: the provider requires host services for `GeppettoOptions`, while the generic xgoja `RuntimeFactory` path currently constructs `ModuleContext` with context/name/as/config but not host services. For generated Geppetto binaries, the design should explicitly define where host services come from and how credentials/tools/profile registries are allowed.

### Current Pinocchio integration pattern

Pinocchio is important because it shows how a real app composes runtime pieces:

- `pinocchio/pkg/inference/runtime/engine.go` builds an engine from settings and middlewares.
- `pinocchio/pkg/inference/runtime/composer.go` separates app-owned runtime requests from composed runtime outputs.
- `pinocchio/pkg/chatapp/runtime_inference.go` receives a runtime engine, wraps event sinks, creates a Geppetto session, appends history/prompt, starts inference, and publishes chat UI events.
- `pinocchio/pkg/chatapp/runtime_sink.go` maps Geppetto runtime events to chatapp protobuf UI events.

This confirms that a good JS binding should not only “call an LLM”. It should expose a clean host/app boundary for runtime policy, sinks, tool permissions, persistence, and profile provenance.

## What Is Good

### 1. The module is structured around Go-native runtime state

`moduleRuntime` centralizes VM, runtime owner, bridge, registries, middleware factories, defaults, schemas, event sinks, snapshot hooks, and persisters. This is good because the runtime already has a place to store host-provided capabilities.

Why it matters:

- A builder API can be added without inventing a second runtime container.
- Host policy can be enforced in one place.
- xgoja and lab hosts can share the same module options.

### 2. Hidden Go references are the right bridge pattern

`attachRef` and `getRef` avoid losing Go identity when JavaScript passes objects back into Go. `applyBuilderOptions` even contains a comment explaining why it reads live goja object properties instead of `Export()` for ref-carrying fields.

This pattern should become the primary object model. Builder methods should return Go-backed handles, not plain object maps, whenever the value represents a validated domain object.

### 3. There is a useful namespace taxonomy

The current namespaces are easy to explain:

- construction: `turns`, `engines`, `tools`, `middlewares`;
- orchestration: `createBuilder`, `createSession`, `runner`;
- metadata: `profiles`, `schemas`, `events`.

The redesign can keep this taxonomy for lower-level APIs while adding a higher-level `agent()` / `chat()` / `embeddings()` facade.

### 4. Tests exercise runtime integration

`module_test.go` contains integration tests for requiring the module, turns, engines, sessions, tools, events, profiles, runners, streaming handles, and registry lifecycle. `dts_parity_test.go` checks generated TypeScript declaration parity against runtime exports. `runtime_test.go` validates that `NewRuntime` exposes `require("geppetto")` and default module behavior.

The redesign should preserve this test strategy and add new builder contract tests.

### 5. Documentation and examples exist

The repository already has:

- API reference: `geppetto/pkg/doc/topics/13-js-api-reference.md`.
- User guide: `geppetto/pkg/doc/topics/14-js-api-user-guide.md`.
- Getting started tutorial: `geppetto/pkg/doc/tutorials/05-js-api-getting-started.md`.
- Example scripts under `geppetto/examples/js/geppetto`.
- Generated declaration file: `geppetto/pkg/doc/types/geppetto.d.ts`.

This gives the intern concrete places to update after implementation.

## What Could Be Better

### 1. Introduce one opinionated happy path

Today the user chooses among `createSession`, `createBuilder`, `runInference`, and `runner.*`. Each is legitimate, but there is no single obvious default.

Proposed happy path:

```javascript
const gp = require("geppetto");

const agent = gp.agent()
  .profile("assistant")
  .system("Answer in one short paragraph.")
  .tools((tools) => tools
    .js("lookup", "Lookup a document")
      .input(gp.schema.object({ q: gp.schema.string().required() }))
      .handle(({ q }) => ({ text: search(q) }))
  )
  .events((ev) => ev.onTextDelta((d) => console.log(d.text)))
  .build();

const result = agent.ask("What changed in this repository?");
console.log(result.text());
```

This hides the lower-level split while still allowing access to engine, runtime, session, turns, and event streams when needed.

### 2. Make turns and blocks Go-backed builders

Current turn helpers return plain JavaScript objects that are decoded later. That is flexible, but it means invalid shapes can travel far before errors appear.

Target shape:

```javascript
const turn = gp.turn()
  .system("You are a code reviewer.")
  .user("Review this diff.")
  .metadata("trace_id", "abc")
  .data("tool_config", gp.toolConfig().choice("auto"))
  .build();
```

Go should validate block kind, role, payload schema, metadata keys, data keys, and tool config as methods are called. The resulting JS object can expose convenience accessors but should carry a `*turns.Turn` ref.

### 3. Replace raw model config maps with profile/config builders

Current `engines.fromConfig(options)` accepts a dynamic map with keys like `apiType`, `provider`, `model`, `temperature`, `topP`, `timeoutMs`, `apiKey`, `baseURL`, and `modelInfo`. This is simple but weakly typed.

Target shape:

```javascript
const engine = gp.engine()
  .openaiResponses()
  .model("gpt-5-mini")
  .apiKeyEnv("OPENAI_API_KEY")
  .temperature(0.2)
  .timeoutMs(30000)
  .build();
```

The implementation can keep `engines.fromConfig` as an adapter, but the preferred API should be builder-first.

### 4. Make embeddings a first-class module capability

`geppetto/pkg/js/embeddings-js.go` is older and separate from `require("geppetto")`. It registers a global object with sync, async, and callback-style embedding methods using an event loop. The TODO already says to move registration into a runtime engine context, pass context into wrappers for cancellation, remove callback-style embeddings, and remove stale JS conversation/runtime coupling.

Target shape:

```javascript
const embedder = gp.embeddings()
  .profile("text-embedding-3-small")
  .batchSize(64)
  .build();

const vector = await embedder.embed("hello");
const many = await embedder.embedMany(["a", "b"]);
```

Embeddings should share the same module options and runtime owner bridge as inference.

### 5. Make xgoja host services explicit

The xgoja provider config already has security-relevant fields (`allowNetwork`, `allowTools`), but the provider requires host services. Generated standalone apps need a clear host implementation path.

Target Go shape:

```go
type GeppettoHost struct {
    Profiles profiles.RegistryReader
    Tools    tools.ToolRegistry
    Secrets  SecretResolver
    Policy   RuntimePolicy
}

func (h *GeppettoHost) GeppettoOptions(ctx context.Context, cfg provider.Config) (geppetto.Options, error) {
    if !cfg.AllowNetwork { /* disallow network engines */ }
    if !cfg.AllowTools { /* do not expose Go tools */ }
    return geppetto.Options{...}, nil
}
```

The xgoja runtime factory also needs a path to pass host services into provider `ModuleContext`.

## What Is Bad or Risky

### 1. Too many unchecked JavaScript maps

The codec accepts `map[string]any` for turn data, metadata, payloads, block shorthands, engine options, tool parameters, runner runtime metadata, and extension schemas. This makes the API permissive but undermines strong runtime validation.

The risk is not only type safety. It is also UX: errors appear at execution time instead of construction time, often as generic “must be an object” or provider factory errors.

### 2. `parseBlockKind` silently downgrades unknown kinds

Unknown block kinds are parsed as `BlockKindOther`. That may be useful for compatibility, but it is dangerous for an opinionated API. A typo like `"tol_call"` should fail in strict builder paths.

Recommendation:

- keep permissive behavior in `turns.normalize` or `turns.fromObject`;
- make `gp.turn().blockKind(...)` and named builders strict by default;
- provide `.unsafeBlock(object)` or `.fromLooseObject(object)` for migration.

### 3. Callback-based embeddings are stale

`generateEmbeddingWithCallbacks` returns a cancel function and manually schedules callbacks. This is a less elegant API for modern scripts, and it lives outside the current `geppetto` native module. Promise-based APIs with runtime-owner posting should be the default.

### 4. Lower-level and higher-level concepts are mixed

The current `runner` is app/runtime-oriented, while `profiles` are engine-oriented, and the docs explicitly describe a “hard cut” between engine profiles and runtime behavior. This separation is architecturally sound, but it is confusing for simple scripts.

The solution is not to merge internals again. The solution is to add a high-level facade that composes both sides explicitly and transparently.

### 5. xgoja provider host dependency needs tightening

The Geppetto provider requires host services, but the generic runtime factory path shown in `go-go-goja/pkg/xgoja/app/factory.go` does not pass host services in the `ModuleContext`. If this remains unresolved, Geppetto xgoja bundles will fail unless a custom path injects host services.

This is a review-critical integration point for standalone bundled applications.

## What Is Missing

### User-facing API gaps

- No single `agent()` or `chat()` facade.
- No Go-backed `TurnBuilder` / `BlockBuilder`.
- No strict mode vs loose migration mode distinction.
- No first-class `embeddings()` namespace in `require("geppetto")`.
- No schema builder for tool input definitions.
- No result object with typed helpers such as `.text()`, `.turn()`, `.usage()`, `.toolCalls()`.
- No standard secret/provider policy abstraction for xgoja standalone binaries.
- No concise “one page happy path” doc for intern/project users.

### Implementation gaps

- Need Go structs for JS-facing builder state.
- Need validation methods that return precise TypeErrors at construction time.
- Need new TypeScript declarations generated from the builder surface.
- Need examples for agent, embeddings, xgoja bundle, and migration from current maps.
- Need tests that prove invalid builder calls fail early.
- Need host-services propagation in go-go-goja xgoja runtime construction or an explicit Geppetto xgoja host adapter.

## What Is Confusing

### `profiles` means engine profiles, not full runtime profiles

The docs now say the `profiles` namespace is really an engine profiles namespace. A new user will still expect a “profile” to describe the whole agent runtime, including prompt, middleware, tools, and provider.

Recommendation:

- Keep `profiles` as-is for compatibility.
- Add `engineProfiles` as a clearer alias.
- In the high-level facade, use `.profile("assistant")` but document that it resolves an engine profile plus optional app runtime policy supplied by the host.

### `runner.resolveRuntime` does not run or resolve engines

The word “runner” suggests execution, but `resolveRuntime` only resolves app-owned runtime inputs. The method is useful but too low-level for first contact.

Recommendation:

- Keep `runner` for advanced use.
- Introduce `agent().run(...)` / `chat().ask(...)` for ordinary use.

### Tools have two separate stories

Current API supports JavaScript tools and imported Go tools, which is good. However, the recommended path for typed tool schemas is not clear. Tool `parameters` accepts raw JSON Schema. Users need a builder that can produce JSON Schema while validating common mistakes.

## Proposed Architecture

### Design principles

1. **Builder-first, object-literal-second.** Preferred APIs should be methods on Go-backed builders. Object literals remain for migration and advanced escapes.
2. **Strict by default.** Builder calls should validate names, enum values, numeric ranges, required fields, and host policy immediately.
3. **Opaque Go handles for domain objects.** Engines, agents, turns, tools, embedding models, sessions, and results should carry Go refs.
4. **Small JavaScript surface, rich Go internals.** JavaScript should compose, not reimplement Geppetto’s internal data model.
5. **Opinionated defaults with explicit overrides.** Default to safe tool policy, host profile resolution, standard event collection, timeouts, and cancellation.
6. **xgoja-ready.** The API must work in generated standalone binaries with explicit host policy for secrets, network, tools, profiles, and filesystem access.

### New public API layers

```text
Layer 1: Opinionated facade
  gp.agent(), gp.chat(), gp.embeddings()

Layer 2: Typed builders
  gp.turn(), gp.engine(), gp.tools.builder(), gp.schema, gp.runtime()

Layer 3: Existing advanced APIs
  gp.turns.*, gp.engines.*, gp.runner.*, gp.profiles.*, gp.tools.createRegistry()

Layer 4: Host Go APIs
  Options, HostServices, provider config, xgoja runtime factory
```

### Proposed `agent()` API

```javascript
const agent = gp.agent()
  .name("repo-reviewer")
  .profile("assistant")
  .system("You are a careful code reviewer.")
  .temperature(0.1)
  .maxTokens(2000)
  .tool("read_file", (t) => t
    .description("Read a repository file")
    .input(gp.schema.object({ path: gp.schema.string().required() }))
    .handler(({ path }) => fs.readFile(path, "utf8")))
  .goTool("grep")
  .events((events) => events
    .onStart((ev) => console.log("start", ev.inferenceId))
    .onTextDelta((ev) => process.stdout.write(ev.text)))
  .build();

const result = await agent.ask("Summarize the JS binding architecture.");
console.log(result.text());
```

Go-backed pieces:

- `agentBuilderRef`: stores engine/profile config, runtime policy, tool builder, event config, default run options.
- `agentRef`: stores compiled engine/session factory/tool registry/event sink policy.
- `runResultRef`: stores final `*turns.Turn`, usage, run metadata, text extraction helpers, tool call summaries.

### Proposed `turn()` API

```javascript
const turn = gp.turn()
  .id("turn-1")
  .system("You are concise.")
  .user("Hello")
  .assistant("Hi")
  .toolCall("call-1", "lookup", { q: "abc" })
  .toolResult("call-1", { answer: 42 })
  .metadata("trace_id", "trace-123")
  .build();
```

Implementation notes:

- `TurnBuilder` should hold a `*turns.Turn` internally.
- Named methods call existing Go constructors such as `turns.NewUserTextBlock` and `turns.NewToolCallBlock`.
- `.metadata(key, value)` should validate canonical keys or namespaced keys.
- `.data(key, value)` should validate known keys where possible.
- `.fromObject(obj)` may call the existing codec in permissive mode.

### Proposed `engine()` API

```javascript
const engine = gp.engine()
  .provider("openai-responses")
  .model("gpt-5-mini")
  .apiKeyEnv("OPENAI_API_KEY")
  .baseURL("https://api.openai.com/v1")
  .temperature(0.2)
  .topP(0.9)
  .timeoutMs(30000)
  .modelInfo((m) => m
    .contextWindow(128000)
    .maxOutputTokens(8192)
    .reasoning(true))
  .build();
```

Validation examples:

- provider must be one of known API types or a registered extension provider;
- model must not be empty;
- temperature must be in provider-supported range;
- timeout must be positive;
- API key literal should be discouraged in xgoja unless host policy allows it.

### Proposed `embeddings()` API

```javascript
const embedder = gp.embeddings()
  .provider("openai")
  .model("text-embedding-3-small")
  .dimensions(1536)
  .apiKeyEnv("OPENAI_API_KEY")
  .timeoutMs(10000)
  .build();

const one = await embedder.embed("hello");
const batch = await embedder.embedMany(["hello", "world"]);
console.log(one.dimensions, one.values.length);
```

Result helpers:

```javascript
one.values();       // number[]
one.dimensions();   // number
one.model();        // model metadata
one.toJSON();       // stable object
```

This should replace the global `RegisterEmbeddings` pattern with a namespace in `require("geppetto")`.

### Proposed `schema` API for tools

```javascript
const params = gp.schema.object({
  query: gp.schema.string().min(1).required(),
  limit: gp.schema.integer().min(1).max(20).default(5),
});
```

The schema builder should return a Go-backed schema ref wrapping `jsonschema.Schema`. Tool registration can still export JSON Schema to Geppetto tool definitions.

### Proposed result API

```javascript
const result = await agent.ask("hello");

result.text();        // assistant text after current run
result.turn();        // Go-backed turn object / JS view
result.usage();       // normalized usage
result.stopReason();  // provider stop reason
result.events();      // collected events if enabled
result.toJSON();      // stable serializable object
```

This removes the need for every script to inspect block arrays manually.

## Internal Go Design Sketch

### New ref types

```go
type agentBuilderRef struct {
    api *moduleRuntime
    name string
    engineBuilder *engineBuilderRef
    runtimeBuilder *runtimePolicyBuilderRef
    turnDefaults *turnBuilderRef
    toolBuilder *toolRegistryBuilderRef
    eventBuilder *eventBuilderRef
    runDefaults runOptions
}

type agentRef struct {
    api *moduleRuntime
    engine engine.Engine
    runtime *runnerResolvedRuntimeRef
    tools tools.ToolRegistry
    eventSinks []events.EventSink
    defaults runOptions
}

type turnBuilderRef struct {
    api *moduleRuntime
    turn *turns.Turn
    strict bool
}

type runResultRef struct {
    api *moduleRuntime
    turn *turns.Turn
    events []StreamEvent
    metadata map[string]any
}
```

### Export wiring pseudocode

```go
func (m *moduleRuntime) installExports(exports *goja.Object) {
    // existing exports stay
    m.mustSet(exports, "agent", m.newAgentBuilder)
    m.mustSet(exports, "chat", m.newChatBuilder)
    m.mustSet(exports, "turn", m.newTurnBuilder)
    m.mustSet(exports, "engine", m.newEngineBuilder)
    m.mustSet(exports, "embeddings", m.newEmbeddingBuilder)
    m.mustSet(exports, "schema", m.newSchemaNamespace())
}
```

### Builder method pattern

```go
func (m *moduleRuntime) newTurnBuilder(call goja.FunctionCall) goja.Value {
    ref := &turnBuilderRef{api: m, turn: &turns.Turn{}, strict: true}
    o := m.vm.NewObject()
    m.attachRef(o, ref)

    m.mustSet(o, "user", func(call goja.FunctionCall) goja.Value {
        text, err := requiredString(call, 0, "user(text)")
        if err != nil { panic(m.vm.NewTypeError(err.Error())) }
        turns.AppendBlock(ref.turn, turns.NewUserTextBlock(text))
        return o
    })

    m.mustSet(o, "build", func(goja.FunctionCall) goja.Value {
        if err := validateTurn(ref.turn); err != nil { panic(m.vm.NewGoError(err)) }
        return m.newTurnObject(ref.turn.Clone())
    })

    return o
}
```

### Strict vs loose codec

```go
type DecodeMode int
const (
    DecodeLoose DecodeMode = iota
    DecodeStrict
)

func (m *moduleRuntime) decodeBlockWithMode(raw any, mode DecodeMode) (turns.Block, error) {
    // Loose: existing parseBlockKind -> other.
    // Strict: unknown kind returns error.
}
```

Existing functions can call loose mode. New builders should call strict mode or bypass decoding entirely by constructing Go values directly.

## Implementation Plan

### Phase 0: Stabilize evidence and tests

1. Keep the evidence script in this ticket as a reference.
2. Run baseline tests:
   - `go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`
3. Add a small API inventory test for new export names once implemented.

Acceptance criteria:

- existing tests pass;
- docs identify current behavior and migration plan;
- no API changes yet.

### Phase 1: Add Go-backed turn and schema builders

Files to start with:

- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/modules/geppetto/api_turn_builders.go` (new)
- `geppetto/pkg/js/modules/geppetto/api_schema_builders.go` (new)
- `geppetto/pkg/js/modules/geppetto/codec.go`
- `geppetto/pkg/js/modules/geppetto/module_test.go`
- `geppetto/pkg/doc/types/geppetto.d.ts`

Tasks:

1. Add `gp.turn()` export.
2. Add builder methods: `id`, `user`, `system`, `assistant`, `reasoning`, `toolCall`, `toolResult`, `metadata`, `data`, `build`, `toJSON`.
3. Add strict validation helpers.
4. Add `gp.schema` object/string/integer/number/boolean/array helpers.
5. Add tests for valid construction and early invalid failures.
6. Update TypeScript declarations and parity tests.

### Phase 2: Add engine and embedding builders

Files to start with:

- `api_engines.go`
- `api_embeddings.go` (new)
- `geppetto/pkg/js/embeddings-js.go` (migration reference)
- embedding provider packages under `geppetto/pkg/embeddings`
- runtime owner bridge helpers

Tasks:

1. Add `gp.engine()` fluent builder.
2. Internally emit `InferenceSettings` without passing through `map[string]any`.
3. Add `gp.embeddings()` builder.
4. Implement Promise-based `embed` and `embedMany` using runtime owner posting.
5. Mark callback embedding API as legacy in docs.

### Phase 3: Add `agent()` facade

Files to start with:

- `api_agent.go` (new)
- `api_runner.go`
- `api_sessions.go`
- `api_tools_registry.go`
- `api_events.go`

Tasks:

1. Add `gp.agent()` export.
2. Let the agent builder compose engine/profile, runtime prompt/middleware, tools, events, and run defaults.
3. Add `agent.ask`, `agent.run`, `agent.start`, and `agent.session` methods.
4. Add `RunResult` helpers.
5. Ensure streaming handles remain runtime-owner safe.

### Phase 4: Tighten xgoja provider host services

Files to start with:

- `geppetto/pkg/js/modules/geppetto/provider/provider.go`
- `go-go-goja/pkg/xgoja/providerapi/module.go`
- `go-go-goja/pkg/xgoja/app/factory.go`
- `go-go-goja/pkg/xgoja/app/host.go`

Tasks:

1. Decide whether `RuntimeFactory` owns host services or receives them per runtime.
2. Pass `Host` through `ModuleContext` for provider module creation.
3. Add a Geppetto host implementation example.
4. Add tests that a generated runtime with a Geppetto module can require it when host services are present and fails with a clear error when absent.

### Phase 5: Documentation and examples

Files to update:

- `geppetto/pkg/doc/topics/13-js-api-reference.md`
- `geppetto/pkg/doc/topics/14-js-api-user-guide.md`
- `geppetto/pkg/doc/tutorials/05-js-api-getting-started.md`
- `geppetto/examples/js/geppetto/README.md`
- new examples under `geppetto/examples/js/geppetto/fluent/`

Examples to add:

1. `01_turn_builder.js`
2. `02_agent_echo.js`
3. `03_agent_tool.js`
4. `04_agent_streaming.js`
5. `05_embeddings.js`
6. `06_xgoja_bundle.md` or runnable fixture

## Testing Strategy

### Unit tests

- Builder methods return the same object for chainability.
- Invalid enum values fail immediately.
- Missing required fields fail immediately.
- Numeric range validation works.
- Strict turn builder rejects unknown block kinds.
- Schema builder emits valid JSON Schema.

### Runtime integration tests

- `require("geppetto").turn().user("x").build()` round-trips through sessions.
- `agent().engine(gp.engines.echo()).ask("x")` returns deterministic text.
- `agent().tool(...).ask(...)` executes a JS tool through the existing toolloop.
- `embeddings().embed(...)` resolves a Promise on the owner thread with a fake provider.
- xgoja runtime with host services can load `require("geppetto")`.

### Documentation tests / examples

- Every new example should run under `geppetto-js-lab` where possible.
- Live provider examples must self-skip when credentials are missing.
- TypeScript declarations must pass `dts_parity_test.go`.

## Migration Strategy

Do not remove current APIs in the first implementation pass. Instead:

1. Add new fluent APIs next to current namespaces.
2. Update docs to say “preferred” vs “advanced/compatibility”.
3. Add warnings only in documentation, not runtime deprecations yet.
4. Convert examples gradually.
5. After adoption, consider moving permissive constructors under `gp.loose` or marking them legacy.

Compatibility mapping:

| Current API | Preferred API |
|---|---|
| `gp.turns.newTurn({ blocks })` | `gp.turn().user(...).build()` |
| `gp.turns.newUserBlock(text)` | `gp.turn().user(text)` |
| `gp.engines.fromConfig(map)` | `gp.engine().provider(...).model(...).build()` |
| `gp.tools.createRegistry().register(map)` | `gp.agent().tool(name, builderFn)` or `gp.tools.builder()` |
| `gp.runner.run({ engine, runtime, prompt })` | `gp.agent().engine(engine).runtime(runtime).ask(prompt)` |
| global embedding object | `gp.embeddings().build()` |

## Risks and Review-Critical Points

1. **Runtime-owner threading.** Async builder methods, event callbacks, tool handlers, and embeddings must only interact with goja on the owner thread.
2. **xgoja host services.** The standalone generated binary story depends on reliable host-service injection.
3. **Validation drift.** JS builders must reuse Go domain validation, not duplicate rules manually.
4. **Too much facade magic.** The agent facade must remain explainable and provide escape hatches to advanced APIs.
5. **TypeScript drift.** Generated declarations and runtime exports must remain in sync.
6. **Provider credentials.** xgoja bundles need an explicit secret policy so scripts do not normalize unsafe API-key literals.

## Intern Onboarding Guide: How to Work on This Safely

Start by reading these files in order:

1. `geppetto/pkg/js/modules/geppetto/module.go` — export wiring and runtime state.
2. `geppetto/pkg/js/modules/geppetto/api_types.go` — ref structs and core types.
3. `geppetto/pkg/js/modules/geppetto/codec.go` — current object map conversion.
4. `geppetto/pkg/js/modules/geppetto/api_sessions.go` — builder/session/run handles.
5. `geppetto/pkg/js/modules/geppetto/api_runner.go` — current runner orchestration.
6. `geppetto/pkg/js/modules/geppetto/api_tools_registry.go` — JS and Go tool registration.
7. `geppetto/pkg/js/runtime/runtime.go` — runtime creation.
8. `go-go-goja/pkg/xgoja/app/factory.go` — generated runtime module loading.
9. `pinocchio/pkg/chatapp/runtime_inference.go` — real app consumption of Geppetto runtime engines/events.

Then implement in small vertical slices:

```text
Slice 1: gp.turn().user("hello").build()
  -> export -> builder ref -> Go turn -> JS view -> test -> d.ts

Slice 2: gp.schema.object(...)
  -> schema refs -> JSON Schema export -> tool registration test

Slice 3: gp.agent().engine(echo).ask("hello")
  -> agent builder -> session run -> result.text() -> test

Slice 4: xgoja host-service injection
  -> provider context host -> geppetto provider test
```

Do not start with live provider inference. Use echo engines and fake embedding providers first.

## References

### Ticket artifacts

- `sources/01-code-evidence.md` — line-numbered evidence snapshot generated by `scripts/01-collect-evidence.sh`.
- `reference/01-investigation-diary.md` — chronological investigation diary.

### Core Geppetto files

- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/module.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_types.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/codec.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_sessions.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_runner.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_tools_registry.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/runtime/runtime.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/embeddings-js.go`

### xgoja files

- `/home/manuel/workspaces/2026-06-01/geppetto-js/go-go-goja/pkg/xgoja/providerapi/module.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/go-go-goja/pkg/xgoja/app/factory.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/go-go-goja/pkg/xgoja/app/host.go`

### Downstream app files

- `/home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/inference/runtime/engine.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/inference/runtime/composer.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/chatapp/runtime_inference.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/chatapp/runtime_sink.go`
