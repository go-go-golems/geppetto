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

This redesign is now a **hard-cut ideal API model**, not an incremental compatibility layer. We do not need to preserve the current JavaScript object/map API as a first-class public contract. The new contract should make JavaScript manipulate Go-owned domain objects through explicit, typed Go wrapper values wherever possible. JavaScript should call methods on Go-backed builders, sessions, turns, tools, engines, embedding models, and result objects; it should not assemble loosely shaped maps that are decoded later.

The current `require("geppetto")` module already proves that Go-backed state can be carried through goja, but its `__geppetto_ref` mechanism is a transitional implementation detail rather than the ideal model. The ideal model is: **Go constructs the object, Go owns the object state, Go validates every mutation, JavaScript only receives a method surface and explicit serialization boundaries such as `toJSON()` or `snapshot()`**.

The proposed public API is therefore a clean break: `gp.agent()`, `gp.turn()`, `gp.engine()`, `gp.tool()`, `gp.schema`, and `gp.embeddings()` produce typed Go wrappers. The old `turns.newTurn(map)`, `engines.fromConfig(map)`, `runner.run({ ... })`, and raw JavaScript registry APIs should be removed or moved to an intentionally named `gp.unsafe`/`gp.compat` package only if a concrete need appears. The default API should feel fluid in JavaScript but behave like a typed Go SDK.

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
- Preserving current JavaScript map/object constructors as public compatibility APIs.
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
| - current transitional JS facades around Go refs                         |
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

### Current wrapper/reference pattern: useful evidence, not the target model

The current module uses a hidden reference mechanism for several facade objects. Go creates a JavaScript object, attaches methods/properties, and stores an unexported Go pointer under `__geppetto_ref`. Later, when JavaScript passes that object back into a Go method, `getRef(...)` recovers the original Go reference instead of relying on `goja.Value.Export()`, which would usually produce a `map[string]any` and lose identity.

```text
Current transitional pattern

Go *engineRef / *builderRef / *sessionRef
        |
        v
custom JS facade object + hidden __geppetto_ref
        |
        v
later Go API calls getRef(...) to recover identity
```

This answers what `__geppetto_ref` is for: it preserves Go identity behind a hand-authored JavaScript facade. It is useful for the current implementation, but it should not be the conceptual API model. The ideal model is not “plain JS object plus hidden pointer”; it is “a Go object exposed into goja with an intentionally designed JavaScript method surface.”

The practical difference is important:

- Transitional hidden-ref facade: JavaScript sees an ordinary object that secretly points at Go state.
- Ideal Go wrapper object: JavaScript receives an object whose methods are Go methods/adapters and whose state is explicitly Go-owned.

The redesign should keep hidden refs only as an implementation fallback if goja requires them for a custom facade. It should not rely on hidden refs as the principal architecture for new domain objects.

### Pros and cons: hidden refs vs direct Go wrappers

| Approach | Pros | Cons | Recommendation |
|---|---|---|---|
| Hidden `__geppetto_ref` on JS facade | Ergonomic custom JS shape; preserves identity across calls; avoids exposing raw Go internals; can hide non-enumerable implementation detail. | Easy to lose by spreading/cloning/serializing; debugging is confusing; creates visible object vs hidden state split; every API must remember to recover the ref; still encourages facade objects that can drift from Go state. | Accept only as a low-level implementation detail where goja needs a custom JS object shape. Do not design the public model around it. |
| Direct Go wrapper values exposed to JS | JavaScript manipulates Go-owned state; validation happens on every method call; fewer map decode paths; identity is natural; easier to reason about lifecycle. | Requires carefully designed wrapper types so Go naming/internals do not leak; needs explicit `toJSON()`/snapshot methods; async methods must respect runtime-owner threading. | Preferred default for all new public API objects. |
| Plain JavaScript maps decoded later | Very easy to prototype; serializable; familiar to JS users. | Weak typing; delayed errors; typos become runtime surprises; no object identity; provider/tool/runtime invariants are hard to enforce. | Remove from the ideal public API except explicit `gp.unsafe.fromObject(...)` style escape hatches. |

### Ideal object model

The new API should expose Go-owned wrappers with JavaScript-friendly method names. The JavaScript side should mutate Go state by calling methods; it should not mutate exported maps in place.

```text
Ideal hard-cut model

JavaScript call: gp.turn().system("...").user("...").build()
        |
        v
Go wrapper method validates input immediately
        |
        v
Go-owned *turns.Turn is mutated/cloned/finalized
        |
        v
JS receives another Go wrapper or explicit immutable snapshot
```

Rules:

1. **No public mutable maps for domain objects.** Turns, blocks, engines, tools, schemas, agents, sessions, embeddings, and results are wrappers.
2. **Methods are the mutation boundary.** Every method validates its arguments before changing Go state.
3. **Serialization is explicit.** Use `toJSON()`, `snapshot()`, `yaml()`, or `debug()` when JavaScript needs data.
4. **Build/finalize boundaries are explicit.** Builders can be mutable; built objects should be immutable or copy-on-write.
5. **Escape hatches are named as unsafe.** If raw object import is needed, use names such as `gp.unsafe.turnFromObject(obj)`.

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

### 2. The current hidden-ref mechanism proves identity preservation is needed

`attachRef` and `getRef` show why plain JavaScript objects are insufficient: when goja exports an object, pointer identity and non-enumerable Go state can disappear. The current mechanism is good evidence that Geppetto needs Go-owned identity across calls.

However, hidden refs should not become the primary object model. The redesign should move one level cleaner: explicit Go wrapper values should be the public objects, and hidden refs should be treated as a private compatibility/implementation trick only when a custom facade is unavoidable.

### 3. The current namespace taxonomy reveals useful domain boundaries

The current namespaces are not the final API, but they identify real domain boundaries:

- construction: turns, engines, tools, middleware;
- orchestration: sessions, runs, agents;
- metadata: engine profiles, schemas, events.

The hard-cut redesign should rename and reshape these boundaries around typed constructors (`gp.turn`, `gp.engine`, `gp.tool`, `gp.agent`, `gp.embeddings`) instead of preserving the current namespace names for compatibility.

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

const inference = gp.inferenceProfiles.resolve("assistant");

const agent = gp.agent()
  .inference(inference)
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

This keeps the split visible: `inference` comes from inference settings/profile resolution, while system prompt, tools, middleware, and events are configured directly on the JS agent API.

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

### 3. Replace raw model config maps with inference settings builders

Current `engines.fromConfig(options)` accepts a dynamic map with keys like `apiType`, `provider`, `model`, `temperature`, `topP`, `timeoutMs`, `apiKey`, `baseURL`, and `modelInfo`. This is simple but weakly typed, and it mixes credentials into JavaScript.

Target shape:

```javascript
const settings = gp.inferenceSettings()
  .provider("openai-responses")
  .model("gpt-5-mini")
  .credentialRef("openai-main")
  .temperature(0.2)
  .timeoutMs(30000)
  .build();

const engine = gp.engine().inference(settings).build();
```

The public API should not expose `apiKey` or `apiKeyEnv`. Credential lookup is a host Go responsibility behind symbolic credential references.
### 4. Make embeddings a first-class module capability

`geppetto/pkg/js/embeddings-js.go` is older and separate from `require("geppetto")`. It registers a global object with sync, async, and callback-style embedding methods using an event loop. The TODO already says to move registration into a runtime engine context, pass context into wrappers for cancellation, remove callback-style embeddings, and remove stale JS conversation/runtime coupling.

Target shape:

```javascript
const embedder = gp.embeddings()
  .provider("openai")
  .model("text-embedding-3-small")
  .credentialRef("openai-main")
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

- remove permissive public constructors from the ideal API;
- make `gp.turn()` and any standalone `gp.block()` builder strict by default;
- provide only explicitly named unsafe import methods if raw object ingestion is genuinely required.

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

### `profiles` must mean Geppetto inference registries only

A new user will expect a generic “profile” to describe the whole agent runtime, including prompt, middleware, tools, memory, and provider. That is exactly what Geppetto should avoid. Within this module, profile resolution is named `inferenceProfiles`, and it loads/resolves **Geppetto engine profile registries** only. A resolved profile returns a Go-backed `InferenceSettings` object.

Recommendation:

- Replace `profiles` / `engineProfiles` naming with `inferenceProfiles` in JavaScript.
- `gp.inferenceProfiles.load("profiles.yaml")` loads Geppetto registry YAML/SQLite sources using `engineprofiles.RegistrySourceSpec` internally.
- `registry.resolve("assistant")` returns a Go-backed `InferenceSettings` object.
- `gp.inferenceProfiles.resolve("assistant")` is a convenience over the host-default Geppetto registry chain, if the host configured one.
- Do not load Pinocchio unified config documents (`app:`, `profile:`, inline `profiles:` overlays) in this JS API. If Pinocchio wants to supply defaults, it should pass Geppetto registry sources or a ready `engineprofiles.RegistryReader` to the module host options.
- Do not add `agent.profile(...)`; use `agent.inference(settings)`.
- System prompt, middlewares, tools, tool loop policy, and event handlers are configured through the JS API (`gp.agent()`), not through Geppetto inference profiles.

### `runner.resolveRuntime` does not run or resolve engines

The word “runner” suggests execution, but `resolveRuntime` only resolves app-owned runtime inputs. The method is useful but too low-level for first contact.

Recommendation:

- Keep low-level runner concepts as Go internals or expose them under clearer wrapper names such as `sessionBuilder` only if needed.
- Introduce `agent().run(...)` / `chat().ask(...)` for ordinary use.

### Tools have two separate stories

Current API supports JavaScript tools and imported Go tools, which is good. However, the recommended path for typed tool schemas is not clear. Tool `parameters` accepts raw JSON Schema. Users need a builder that can produce JSON Schema while validating common mistakes.

## Proposed Architecture

### Design principles

1. **Hard cut to Go-owned wrappers.** New public objects are Go wrapper values with JavaScript-friendly methods, not object literals carrying hidden metadata.
2. **No map-first constructors.** Do not design around `map[string]any` options. Use typed builders and explicit setters.
3. **Strict by default.** Unknown keys, unknown block kinds, invalid enum values, out-of-range settings, missing required fields, and disallowed host capabilities fail immediately.
4. **JavaScript mutates through methods only.** State transitions happen through Go methods so invariants are enforced at the boundary.
5. **Explicit serialization boundaries.** `toJSON()`, `snapshot()`, `toYAML()`, or `debug()` produce plain objects for inspection and persistence; those plain objects are not the live domain objects.
6. **Immutable built objects.** Builders may be mutable; built turns/engines/agents/results should be immutable or copy-on-write from JavaScript’s perspective.
7. **Opinionated happy path first.** `gp.agent()` and `gp.embeddings()` should cover common scripts without requiring users to understand sessions, runner runtimes, profile internals, and event sinks.
8. **Host policy is part of construction.** Network, credentials, tools, filesystem access, and profile registries are host-owned capabilities exposed through typed services, especially in xgoja bundles.
9. **Unsafe APIs are explicit.** Any raw-object import/export API must live under `gp.unsafe` or similarly clear names.

### New public API layers

```text
Layer 1: Opinionated facade
  gp.agent(), gp.chat(), gp.embeddings()

Layer 2: Typed Go wrapper builders
  gp.inferenceSettings(), gp.turn(), gp.engine(), gp.tool(), gp.schema, gp.runtime()

Layer 3: Explicit unsafe/import APIs
  gp.unsafe.turnFromObject(), gp.unsafe.inferenceSettingsFromObject(), gp.unsafe.debugExport()

Layer 4: Host Go APIs
  Options, HostServices, provider config, xgoja runtime factory
```

### Final public module contract after hard cut

The default `require("geppetto")` export should be intentionally small:

```typescript
declare module "geppetto" {
  export function agent(): AgentBuilder;
  export function chat(): ChatBuilder;
  export function turn(): TurnBuilder;
  export function inferenceSettings(): InferenceSettingsBuilder;
  export function engine(): EngineBuilder;
  export function tool(name: string): ToolBuilder;
  export function toolRegistry(): ToolRegistryBuilder;
  export function embeddings(): EmbeddingBuilder;
  export const schema: SchemaNamespace;
  export const inferenceProfiles: InferenceProfileNamespace;
  export const events: EventNamespace;
  export const unsafe: UnsafeNamespace;
}

interface InferenceProfileNamespace {
  /** Load one or more Geppetto registry sources (YAML, SQLite file, or SQLite DSN). */
  load(source: string | string[]): InferenceRegistry;

  /** Resolve against the host-default Geppetto registry chain, if configured. */
  resolve(input?: string | ResolveInferenceInput): InferenceSettings;

  /** Return the host-default registry wrapper, if configured. */
  default(): InferenceRegistry;
}

interface InferenceRegistry {
  listRegistries(): RegistrySummary[];
  listProfiles(registrySlug?: string): InferenceProfileSummary[];
  resolve(input?: string | ResolveInferenceInput): InferenceSettings;
  close(): void;
}

interface ResolveInferenceInput {
  registry?: string;
  profile?: string;
}
```

Names to remove from the default public contract:

- `turns.newTurn`, `turns.newUserBlock`, `turns.normalize` as first-class APIs;
- `engines.fromConfig(map)` and other map-first constructors;
- top-level `createBuilder`, `createSession`, and `runInference`;
- `runner.run({ ... })` as the ordinary user entrypoint;
- tool registration through raw `ToolSpec` maps;
- callback-style embedding APIs.

Internally, old helpers can survive temporarily to support tests or implementation plumbing, but the public `.d.ts`, tutorials, examples, and generated xgoja docs should describe only the hard-cut contract above plus explicit `unsafe` imports.

### Naming and responsibility boundaries

Use names that make the separation explicit:

| Concept | Public name | Owns | Must not own |
|---|---|---|---|
| Inference settings | `InferenceSettings`, `gp.inferenceSettings()` | provider, API type, model, sampling, token limits, base URL, model metadata, credential reference | system prompt, tools, middleware chain, JS callbacks, app runtime policy |
| Inference profile catalog | `gp.inferenceProfiles` | loading/resolving named `InferenceSettings` from Geppetto registry sources (`profiles.yaml`, SQLite, SQLite DSN, or host-provided `RegistryReader`) | agent configuration, tool registration, system prompts, Pinocchio app config documents |
| Engine | `Engine`, `gp.engine()` | compiled Geppetto inference engine built from `InferenceSettings` | app/session/tool policy |
| Agent runtime | `AgentBuilder`, `Agent` | system prompt, JS-configured middlewares, JS/Go tools, tool loop policy, event handling, run defaults | provider/model/profile lookup except by accepting explicit `InferenceSettings` or an inference-profile selector |
| Host credentials | `CredentialRef` / `credentialRef(name)` | names an API credential resolved by the Go host | raw API key strings, environment variable lookup from JS |

In Geppetto, the unqualified word “profile” should not mean “full agent profile”. It should mean **inference profile** only: a named source of inference settings. If a user wants a full application profile containing prompt, tools, middleware, memory, UI policy, or agent presets, they should build that profile system in their own app and pass the resulting pieces into this module.

The default host integration should use Geppetto registry sources. In a Pinocchio host, that can mean passing Pinocchio's `profiles.yaml` file if it is a Geppetto registry file, or passing a prebuilt `engineprofiles.RegistryReader`. Geppetto JS should not parse Pinocchio unified config documents directly.

### Credential policy: no environment variables in JS

The public JavaScript API must not expose `apiKey`, `apiKeyEnv`, `fromEnv`, or equivalent methods. JS scripts should never fetch provider credentials directly from environment variables. Instead, scripts choose a named credential reference and the Go host resolves that reference according to host policy.

Allowed:

```javascript
const settings = gp.inferenceSettings()
  .provider("openai-responses")
  .model("gpt-5-mini")
  .credentialRef("openai-main")
  .temperature(0.2)
  .build();
```

Forbidden in the public API:

```javascript
gp.engine().apiKey("sk-...");             // forbidden
gp.engine().apiKeyEnv("OPENAI_API_KEY"); // forbidden
gp.credentials.fromEnv("OPENAI_API_KEY"); // forbidden
```

A host may implement `credentialRef("openai-main")` by reading a key from an environment variable, keychain, config file, vault, or OS secret store. That is a Go-side host decision, not a JavaScript API feature.

Host-side shape:

```go
type InferenceRegistryProvider interface {
    DefaultInferenceRegistry(ctx context.Context) (engineprofiles.RegistryReader, func(), error)
}

type CredentialResolver interface {
    ResolveCredential(ctx context.Context, ref CredentialRef, provider string) (ResolvedCredential, error)
}

type GeppettoHostServices interface {
    InferenceRegistryProvider() InferenceRegistryProvider // Geppetto registry source/reader only.
    Credentials() CredentialResolver                      // JS never sees raw API keys.
    GoTools() tools.ToolRegistry                          // Optional host-approved Go tools.
}
```

A Pinocchio host can implement `InferenceRegistryProvider` by passing Geppetto registry files (for example `pinocchio/profiles.yaml` when it uses Geppetto registry format) or by constructing a `engineprofiles.RegistryReader` itself. Pinocchio-level concepts such as `app.repositories`, agent presets, UI behavior, app mode, and persisted chat runtime remain Pinocchio/application concerns.

### Geppetto registry YAML as the supported profile file format

The JavaScript API should stick to Geppetto registry files. A registry file has a registry slug, an optional default profile slug, and profile entries containing `inference_settings`.

```yaml
slug: local
default_profile_slug: assistant
profiles:
  assistant:
    display_name: Assistant
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-5-mini
  cheap:
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4o-mini
```

Meaning:

- `slug` identifies the registry when multiple registries are loaded.
- `default_profile_slug` selects the profile used when no profile is supplied for that registry.
- keys under `profiles:` are the resolvable inference profile names.
- profile `stack:` entries may layer settings using existing Geppetto stack resolution.

Supported JS usage:

```javascript
const registry = gp.inferenceProfiles.load("./profiles.yaml");
const settings = registry.resolve("assistant");
const engine = gp.engine().inference(settings).build();
```

Multiple Geppetto registry sources should use existing source-chain semantics:

```javascript
const registry = gp.inferenceProfiles.load([
  "./base-profiles.yaml",
  "./team-overrides.yaml",
  "sqlite:./local-profiles.sqlite",
]);

const settings = registry.resolve({ profile: "assistant" });
```

Unsupported by this API:

```yaml
app:
  repositories: [~/prompts]
profile:
  active: assistant
profiles:
  assistant:
    inference_settings: ...
```

That is a Pinocchio unified config document, not a Geppetto registry file. If an application wants that richer setup, it should resolve it in application code and pass Geppetto either a registry source, a `RegistryReader`, or a final `InferenceSettings` object.

### Proposed `agent()` API

```javascript
const registry = gp.inferenceProfiles.load("./pinocchio/profiles.yaml");
const inference = registry.resolve("assistant");

const agent = gp.agent()
  .name("repo-reviewer")
  .inference(inference)
  .system("You are a careful code reviewer.")
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

- `agentBuilderRef`: stores a selected `InferenceSettings`/engine plus runtime policy, tool builder, event config, system prompt, and default run options.
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

### Proposed `inferenceSettings()` and `engine()` APIs

Build or resolve inference settings first, then compile an engine from those settings. This makes the boundary explicit: inference settings describe provider/model behavior; agents describe prompt/tools/middleware/runtime behavior.

```javascript
const settings = gp.inferenceSettings()
  .provider("openai-responses")
  .model("gpt-5-mini")
  .credentialRef("openai-main")
  .baseURL("https://api.openai.com/v1")
  .temperature(0.2)
  .topP(0.9)
  .timeoutMs(30000)
  .modelInfo((m) => m
    .contextWindow(128000)
    .maxOutputTokens(8192)
    .reasoning(true))
  .build();

const engine = gp.engine()
  .inference(settings)
  .build();
```

Host-default profile resolution should look like this when the host provides a Geppetto registry chain:

```javascript
const settings = gp.inferenceProfiles.resolve("assistant");
const engine = gp.engine().inference(settings).build();
```

Explicit registry loading should look like this when the script chooses the source:

```javascript
const registry = gp.inferenceProfiles.load("./profiles.yaml");
const settings = registry.resolve({ profile: "assistant" });
const engine = gp.engine().inference(settings).build();
```

Validation examples:

- provider must be one of known API types or a registered extension provider;
- model must not be empty;
- temperature must be in provider-supported range;
- timeout must be positive;
- credential reference must be a non-empty symbolic name accepted by host policy;
- raw API keys and environment variable references are rejected because the JavaScript API does not own credentials.

### Proposed `embeddings()` API

```javascript
const embedder = gp.embeddings()
  .provider("openai")
  .model("text-embedding-3-small")
  .credentialRef("openai-main")
  .dimensions(1536)
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
    inference *inferenceSettingsRef
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
    // Hard-cut public surface: export the new typed wrapper constructors.
    // Old map-first namespaces should not be exported unless deliberately
    // placed under gp.unsafe for debugging/imports.
    m.mustSet(exports, "agent", m.newAgentBuilder)
    m.mustSet(exports, "chat", m.newChatBuilder)
    m.mustSet(exports, "inferenceSettings", m.newInferenceSettingsBuilder)
    m.mustSet(exports, "inferenceProfiles", m.newInferenceProfilesNamespace())
    m.mustSet(exports, "turn", m.newTurnBuilder)
    m.mustSet(exports, "engine", m.newEngineBuilder)
    m.mustSet(exports, "tool", m.newToolBuilder)
    m.mustSet(exports, "toolRegistry", m.newToolRegistryBuilder)
    m.mustSet(exports, "embeddings", m.newEmbeddingBuilder)
    m.mustSet(exports, "schema", m.newSchemaNamespace())
    m.mustSet(exports, "unsafe", m.newUnsafeNamespace())
}
```

### Builder method pattern

```go
type TurnBuilderJS struct {
    api  *moduleRuntime
    turn *turns.Turn
}

func (b *TurnBuilderJS) User(text string) *TurnBuilderJS {
    if strings.TrimSpace(text) == "" {
        panic(b.api.vm.NewTypeError("turn.user(text): text must not be empty"))
    }
    turns.AppendBlock(b.turn, turns.NewUserTextBlock(text))
    return b
}

func (b *TurnBuilderJS) Build() *TurnJS {
    if err := validateTurnStrict(b.turn); err != nil {
        panic(b.api.vm.NewGoError(err))
    }
    return &TurnJS{api: b.api, turn: b.turn.Clone()}
}

func (m *moduleRuntime) newTurnBuilder(call goja.FunctionCall) goja.Value {
    // Prefer exposing a Go wrapper value whose methods mutate Go state.
    // If goja requires a custom JS object, keep any hidden ref inside this
    // helper and do not make it part of the conceptual API model.
    return m.vm.ToValue(&TurnBuilderJS{api: m, turn: &turns.Turn{}})
}
```

### Strict import codec for `gp.unsafe`

```go
func (m *moduleRuntime) unsafeTurnFromObject(raw goja.Value) (*TurnJS, error) {
    // Even unsafe import validates strictly by default. A separate method name
    // such as unsafe.looseTurnFromObject would be required for permissive
    // coercion.
    turn, err := decodeTurnStrict(raw.Export())
    if err != nil {
        return nil, err
    }
    return &TurnJS{api: m, turn: turn}, nil
}
```

Normal builders should bypass decoding entirely by constructing Go values directly. Object import exists only for explicit unsafe/debug boundaries.

## Detailed Implementation Task List

This plan assumes the JS API sticks to existing Geppetto registry formats and does **not** implement Pinocchio config-document parsing in Geppetto JS. Pinocchio can still be a host that supplies a Geppetto registry source or a ready `engineprofiles.RegistryReader`.

### Phase 0: Contract lock and baseline inventory

Goal: freeze the intended public API before touching implementation code.

Tasks:

1. Add an API contract test that describes the final top-level `require("geppetto")` keys:
   - `agent`
   - `chat`
   - `inferenceSettings`
   - `inferenceProfiles`
   - `turn`
   - `engine`
   - `tool`
   - `toolRegistry`
   - `embeddings`
   - `schema`
   - `events`
   - `unsafe`
2. Add negative API inventory checks that old map-first names are absent from the public contract:
   - `turns.newTurn`
   - `engines.fromConfig`
   - top-level `createBuilder`
   - top-level `createSession`
   - top-level `runInference`
   - `runner.run` as the ordinary public entrypoint
3. Document that `gp.inferenceProfiles.load(...)` accepts Geppetto registry sources only:
   - YAML registry file path
   - `yaml:PATH`
   - `yaml://PATH`
   - SQLite file path
   - `sqlite:PATH`
   - `sqlite-dsn:DSN`
4. Run baseline tests:
   - `go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`
5. Do not implement new behavior until the contract tests fail for the expected missing API.

Acceptance criteria:

- Contract test exists.
- Old public API names are listed as intentionally removed.
- Baseline package tests pass before implementation begins.

### Phase 1: Go-owned `InferenceSettings` wrapper

Goal: make inference settings a real Go-owned JS object.

Tasks:

1. Add `api_inference_settings.go` in `geppetto/pkg/js/modules/geppetto`.
2. Define `InferenceSettingsBuilderJS` with Go-owned builder state:
   - `api *moduleRuntime`
   - `settings *settings.InferenceSettings`
   - `credentialRef string`
   - optional metadata/provenance fields
3. Define immutable/copy-on-write `InferenceSettingsJS`:
   - wraps cloned `*settings.InferenceSettings`
   - stores symbolic credential reference
   - exposes read-only getters and `toJSON()` snapshots
4. Implement builder methods:
   - `.provider(name)` / provider-specific convenience methods if desired
   - `.model(name)`
   - `.credentialRef(name)`
   - `.baseURL(url)` if allowed by policy
   - `.temperature(value)`
   - `.topP(value)`
   - `.maxTokens(value)`
   - `.timeoutMs(value)`
   - `.modelInfo(fnOrBuilder)`
   - `.build()`
5. Implement validation:
   - provider/model required where necessary
   - numeric ranges checked immediately
   - timeout must be positive
   - credential ref must be symbolic/non-empty
   - raw API key fields are impossible to set
6. Implement snapshot methods:
   - `settings.toJSON()`
   - `settings.clone()`
   - optional `settings.debug()` with redacted credential reference
7. Add tests proving JS receives a Go wrapper:
   - methods exist on returned object
   - `toJSON()` returns a detached snapshot
   - mutating the snapshot does not mutate the Go object
   - object spread/JSON stringify does not become the live state
8. Add negative credential tests:
   - no `.apiKey(...)`
   - no `.apiKeyEnv(...)`
   - no `.fromEnv(...)`

Acceptance criteria:

- `gp.inferenceSettings().provider(...).model(...).credentialRef(...).build()` returns a usable Go wrapper.
- No public JS path accepts a raw API key or env var name.

### Phase 2: Geppetto registry loader wrapper

Goal: expose existing `engineprofiles` registry source loading to JS without Pinocchio.

Tasks:

1. Add `api_inference_profiles.go` or replace the existing `api_profiles.go` public surface.
2. Define `InferenceProfilesNamespaceJS` with methods:
   - `load(source string | string[]): InferenceRegistryJS`
   - `resolve(input?: string | ResolveInferenceInput): InferenceSettingsJS`
   - `default(): InferenceRegistryJS`
3. Define `InferenceRegistryJS` wrapper:
   - owns `engineprofiles.RegistryReader` or `engineprofiles.Registry`
   - owns optional closer
   - knows whether it is host-default or script-loaded
4. Implement `load(...)` using existing Geppetto functions:
   - normalize JS string/array input
   - call `engineprofiles.ParseRegistrySourceSpecs(entries)`
   - call `engineprofiles.NewChainedRegistryFromSourceSpecs(ctx, specs)`
   - wrap returned registry and close function
5. Implement `registry.resolve(...)`:
   - accept string profile slug
   - accept object `{ registry, profile }` only as typed argument decoding, not as live domain state
   - call `ResolveEngineProfile(ctx, ResolveInput)`
   - wrap `ResolvedEngineProfile.InferenceSettings` as `InferenceSettingsJS`
   - attach provenance metadata (`registry`, `profile`, `stackLineage`) to the wrapper
6. Implement listing helpers:
   - `registry.listRegistries()` returns snapshots
   - `registry.listProfiles(registrySlug?)` returns snapshots
7. Implement `registry.close()` for script-loaded chains.
8. Implement host-default resolution:
   - module options accept `EngineProfileRegistry` / `RegistryReader` as they do today
   - `gp.inferenceProfiles.resolve(...)` uses host-default reader
   - if no host default exists, throw a clear error: `no default inference profile registry configured; use gp.inferenceProfiles.load(path)`
9. Add tests with temporary Geppetto registry YAML:
   - single registry load
   - `slug` disambiguation
   - `default_profile_slug`
   - fallback to `profiles.default`
   - stacked profile resolution
   - multiple source precedence
   - invalid source error messages
10. Add explicit non-goal tests/documentation:
   - Pinocchio unified config docs with `app:` are rejected by the Geppetto registry YAML decoder
   - error message says to use Geppetto registry YAML or application-side setup

Acceptance criteria:

- `gp.inferenceProfiles.load("profiles.yaml").resolve("assistant")` returns `InferenceSettingsJS`.
- No Pinocchio package import is introduced into Geppetto.

### Phase 3: Engine builder integration

Goal: compile engines only from Go-owned inference settings.

Tasks:

1. Add or update `EngineBuilderJS`.
2. Implement `.inference(settings)` accepting only:
   - `InferenceSettingsJS`
   - possibly a trusted Go `*settings.InferenceSettings` host object
3. Remove/withhold public `.fromConfig(map)` and `.provider(...).model(...)` shortcuts on `engine()` unless they delegate through `gp.inferenceSettings()` builder internally.
4. On `.build()`:
   - clone settings
   - resolve symbolic credential refs via host `CredentialResolver`
   - inject resolved credentials into settings only on Go side
   - call existing engine factory
   - return Go-owned `EngineJS`
5. Add tests:
   - engine builds from `gp.inferenceSettings()`
   - engine builds from `gp.inferenceProfiles.load(...).resolve(...)`
   - engine rejects plain JS objects
   - engine fails clearly when credential resolver is missing
   - engine never exposes raw credential in `toJSON()`/debug output

Acceptance criteria:

- `gp.engine().inference(settings).build()` is the only ordinary engine construction path.
- Credential resolution happens on Go side only.

### Phase 4: Agent API integration

Goal: configure runtime behavior from JS while keeping inference settings separate.

Tasks:

1. Add/update `api_agent.go`.
2. Implement `gp.agent()` builder methods:
   - `.name(name)`
   - `.inference(settings)`
   - `.engine(engine)` for advanced prebuilt engine injection
   - `.system(prompt)`
   - `.middleware(middleware)` / `.goMiddleware(name, optionsBuilder?)`
   - `.tool(name, builderFn)`
   - `.goTool(name)`
   - `.toolLoop(configBuilder)`
   - `.events(eventBuilderFn)`
   - `.runDefaults(optionsBuilder)`
   - `.build()`
3. Ensure `.inference(...)` accepts `InferenceSettingsJS`, not profile names or JS maps.
4. Keep optional convenience out of the first pass:
   - do not add `agent.profile(...)`
   - do not add `agent.inferenceProfile(...)` unless later UX evidence demands it
5. Implement `agent.ask(prompt)`:
   - constructs a Go-owned turn from prompt
   - applies system prompt/middleware/tools
   - runs session/engine
   - returns `RunResultJS`
6. Implement `RunResultJS` helpers:
   - `.text()`
   - `.turn()`
   - `.usage()`
   - `.stopReason()`
   - `.events()`
   - `.toJSON()`
7. Add tests:
   - echo/fake engine ask path
   - system prompt applied from JS, not profile
   - JS tool execution
   - Go tool import when host registry allows it
   - middleware order
   - result text extraction

Acceptance criteria:

- Users compose runtime behavior with `gp.agent()`, not profile files.
- Inference profile resolution supplies only settings.

### Phase 5: Tool/schema/turn wrappers

Goal: remove remaining map-first construction from everyday scripts.

Tasks:

1. Implement `gp.schema` builders:
   - object
   - string
   - integer
   - number
   - boolean
   - array
   - enum
   - required/default/min/max helpers
2. Implement `gp.tool(name)` builder:
   - `.description(...)`
   - `.input(schema)`
   - `.handler(fn)`
   - `.build()`
3. Implement `gp.toolRegistry()` wrapper:
   - `.add(tool)`
   - `.addGo(name)`
   - `.list()`
   - `.call(name, args)` with args validated against schema where possible
4. Implement `gp.turn()` builder:
   - `.system(text)`
   - `.user(text)`
   - `.assistant(text)`
   - `.toolCall(id, name, args)`
   - `.toolResult(id, result)`
   - `.metadata(key, value)`
   - `.build()`
5. Ensure all built objects are Go-owned wrappers with explicit snapshots.
6. Add tests for invalid schema/tool/turn construction.

Acceptance criteria:

- Example scripts no longer construct turn/block/tool maps directly.

### Phase 6: xgoja and host integration

Goal: make generated standalone binaries able to expose the same API safely.

Tasks:

1. Update Geppetto xgoja provider config schema to include registry source configuration:
   - `profileRegistries?: string[]`
   - `defaultProfile?: string`
   - possibly `allowRegistryLoad?: boolean`
2. Ensure `allowRegistryLoad` defaults to safe host policy.
3. Add host service wiring for:
   - default registry reader
   - credential resolver
   - approved Go tool registry
4. Do not let JS read environment variables for credentials.
5. Add xgoja runtime tests:
   - generated runtime can `require("geppetto")`
   - default registry profile resolves
   - explicit `gp.inferenceProfiles.load(...)` obeys allow/deny policy
   - credential refs are resolved by host only

Acceptance criteria:

- xgoja standalone apps can bundle Geppetto and profile registries without Pinocchio imports.

### Phase 7: Documentation, examples, and declaration generation

Goal: make the hard-cut API teachable and type-visible.

Tasks:

1. Update TypeScript declarations:
   - `InferenceSettingsBuilder`
   - `InferenceSettings`
   - `InferenceProfileNamespace`
   - `InferenceRegistry`
   - `EngineBuilder`
   - `AgentBuilder`
   - `ToolBuilder`
   - `SchemaNamespace`
2. Update `dts_parity_test.go` for the final top-level export set.
3. Add examples:
   - `01_inference_settings_builder.js`
   - `02_load_registry_resolve_profile.js`
   - `03_engine_from_registry_profile.js`
   - `04_agent_from_registry_profile.js`
   - `05_tools_and_schema.js`
   - `06_embeddings_with_credential_ref.js`
4. Add docs explaining Geppetto registry YAML:
   - `slug`
   - `default_profile_slug`
   - `profiles.<slug>`
   - `stack`
   - `inference_settings`
5. Add migration docs stating that Pinocchio unified config docs are application-side and not loaded by `gp.inferenceProfiles.load(...)`.

Acceptance criteria:

- New examples run with deterministic/fake engines where possible.
- Live examples self-skip unless host credential refs are available.

### Phase 8: Cleanup and removal

Goal: complete the hard cut.

Tasks:

1. Remove public exports for old map-first namespaces or move intentionally to `gp.unsafe`.
2. Remove docs that teach old APIs as normal usage.
3. Keep internal codecs only where needed for snapshots/import tests.
4. Add regression tests that old public names stay absent.
5. Run full validation:
   - `go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`
   - broader Geppetto tests if implementation touched registry/core packages
   - xgoja tests if provider wiring changed

Acceptance criteria:

- Public JS API is hard-cut, Go-wrapper based, and Geppetto-registry based.
- Pinocchio is not imported into Geppetto JS.

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
- `agent().inference(fakeInferenceSettings).ask("x")` or `agent().engine(echoEngine).ask("x")` returns deterministic text.
- `agent().tool(...).ask(...)` executes a JS tool through the existing toolloop.
- `embeddings().embed(...)` resolves a Promise on the owner thread with a fake provider.
- xgoja runtime with host services can load `require("geppetto")`.

### Documentation tests / examples

- Every new example should run under `geppetto-js-lab` where possible.
- Live provider examples must use host-provided credential references and self-skip when the host cannot resolve those references.
- TypeScript declarations must pass `dts_parity_test.go`.

## Hard Cutover Strategy

This ticket now assumes there is no legacy JavaScript API that must be preserved as a public contract. That changes the implementation strategy: remove confusing dynamic entrypoints rather than wrapping them in a preferred/legacy story.

### Public API removal/renaming plan

| Current API | Hard-cut replacement |
|---|---|
| `gp.turns.newTurn({ blocks })` | `gp.turn().user(...).system(...).build()` |
| `gp.turns.newUserBlock(text)` | `gp.turn().user(text)` or `gp.block().user(text)` if standalone block builders are needed |
| `gp.engines.fromConfig(map)` | `gp.inferenceSettings().provider(...).model(...).credentialRef(...).build()` plus `gp.engine().inference(settings).build()` |
| `gp.createBuilder(options)` | `gp.agent()` for common flows, `gp.sessionBuilder()` for low-level flows |
| `gp.createSession(options)` | `gp.agent().buildSession()` or `gp.session(engine).build()` |
| `gp.runner.run({ engine, runtime, prompt })` | `gp.agent().inference(settings).system(...).tool(...).ask(prompt)` |
| `gp.tools.createRegistry().register(map)` | `gp.tool(name).description(...).input(schema).handler(fn).build()` and `gp.toolRegistry().add(tool)` |
| Global embedding registration | `gp.embeddings().provider(...).model(...).build()` |

### Escape hatches

If implementation discovers a real need for raw object import/export, it should be intentionally marked unsafe:

```javascript
const turn = gp.unsafe.turnFromObject(obj);     // validates then wraps, no silent coercion
const obj = turn.toJSON();                      // explicit snapshot, not live state
const engine = gp.unsafe.engineFromConfig(obj); // only for tests/migration/debugging
```

Rules for unsafe APIs:

- They are not used in tutorials except migration/debug appendices.
- They validate strictly unless the method name says `loose`.
- They return Go wrappers, never live mutable maps.
- They are easy to grep and remove later.

### Cutover phases

1. Add the new wrapper API behind the final names.
2. Convert examples and docs immediately to the new API.
3. Remove or hide old map-first exports from the public `.d.ts`.
4. Keep internal helper functions if useful, but do not expose them as endorsed JS APIs.
5. Add tests that fail if removed map-first names reappear accidentally.

## Risks and Review-Critical Points

1. **Runtime-owner threading.** Async builder methods, event callbacks, tool handlers, and embeddings must only interact with goja on the owner thread.
2. **xgoja host services.** The standalone generated binary story depends on reliable host-service injection.
3. **Validation drift.** JS builders must reuse Go domain validation, not duplicate rules manually.
4. **Too much facade magic.** The agent facade must remain explainable and provide escape hatches to advanced APIs.
5. **TypeScript drift.** Generated declarations and runtime exports must remain in sync.
6. **Provider credentials.** xgoja bundles and Geppetto hosts must expose only symbolic credential references to JS; raw API keys and environment-variable lookup belong exclusively to host Go code.

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
