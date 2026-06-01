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

This is a reasonable taxonomy, but it splits common user intent across too many namespaces. A user who wants “run a turn with system text, user content, and a tool-enabled runtime” must understand profiles, engines, runner runtimes, sessions, tools, turns, and event collectors before they can write the final script.

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

const registry = gp.inferenceProfiles.load("./profiles.yaml");
const inference = registry.resolve("assistant");

const agent = gp.agent()
  .inference(inference)
  .tools((tools) => tools
    .js("lookup", "Lookup a document")
      .input(gp.schema.object({ q: gp.schema.string().required() }))
      .handle(({ q }) => ({ text: search(q) }))
  )
  .events((ev) => ev.onTextDelta((d) => console.log(d.text)))
  .build();

const turn = gp.turn()
  .system("Answer in one short paragraph.")
  .user("What changed in this repository?")
  .build();

const result = await agent.run(turn);
console.log(result.text());
```

This keeps the split visible: `inference` comes from inference settings/profile resolution; tools, middleware, and events are configured on the JS agent API; and all message content, including system text, lives in explicit `Turn` objects.

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

### 3. Replace raw model config maps with registry-resolved inference settings

Current `engines.fromConfig(options)` accepts a dynamic map with keys like `apiType`, `provider`, `model`, `temperature`, `topP`, `timeoutMs`, `apiKey`, `baseURL`, and `modelInfo`. This is simple but weakly typed, it mixes credentials into JavaScript, and it encourages scripts to treat model parameters as ad-hoc object literals.

Target shape:

```javascript
const registry = gp.inferenceProfiles.load("./profiles.yaml");
const settings = registry.resolve("assistant");
const engine = gp.engine().inference(settings).build();
```

For the first pass, the public API should not expose `gp.inferenceSettings()` or any model-parameter builder. Provider/model/sampling/token/base URL/model metadata changes happen in Geppetto registry files. JavaScript may inspect registry-resolved Go-owned `InferenceSettings` wrappers through `toJSON`, `clone`, and redacted `debug`, but should not mutate model settings directly. The public API should also not expose `apiKey` or `apiKeyEnv`. Credential lookup is a host Go responsibility behind symbolic credential references.
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

- No single explicit `agent().run(turn)` runtime facade.
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
- Introduce `agent().run(turn)` for ordinary runtime execution and keep all message content, including system text, in explicit turns.

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
  gp.agent(), gp.embeddings()

Layer 2: Typed Go wrapper builders
  gp.turn(), gp.engine(), gp.tool(), gp.schema, gp.runtime()

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
  export function turn(): TurnBuilder;
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
- `engines.fromConfig(map)`, `inferenceSettings()` builder APIs, and other map/model-parameter constructors;
- top-level `createBuilder`, `createSession`, and `runInference`;
- `runner.run({ ... })` as the ordinary user entrypoint;
- tool registration through raw `ToolSpec` maps;
- callback-style embedding APIs.

Internally, old helpers can survive temporarily to support tests or implementation plumbing, but the public `.d.ts`, tutorials, examples, and generated xgoja docs should describe only the hard-cut contract above plus explicit `unsafe` imports.

### Naming and responsibility boundaries

Use names that make the separation explicit:

| Concept | Public name | Owns | Must not own |
|---|---|---|---|
| Inference settings | `InferenceSettings` | provider, API type, model, sampling, token limits, base URL, model metadata, credential reference, resolved from Geppetto registries | turn content, tools, middleware chain, JS callbacks, app runtime policy, JS-side model-parameter mutation |
| Inference profile catalog | `gp.inferenceProfiles` | loading/resolving named `InferenceSettings` from Geppetto registry sources (`profiles.yaml`, SQLite, SQLite DSN, or host-provided `RegistryReader`) | agent configuration, tool registration, turn/system content, Pinocchio app config documents |
| Engine | `Engine`, `gp.engine()` | compiled Geppetto inference engine built from `InferenceSettings` | app/session/tool policy |
| Agent runtime | `AgentBuilder`, `Agent` | JS-configured middlewares, JS/Go tools, tool loop policy, event handling, run defaults, engine/inference selection | message content, system prompts, user prompts, provider/profile lookup by string |
| Host credentials | `CredentialRef` / `credentialRef(name)` | names an API credential resolved by the Go host | raw API key strings, environment variable lookup from JS |

In Geppetto, the unqualified word “profile” should not mean “full agent profile”. It should mean **inference profile** only: a named source of inference settings. If a user wants a full application profile containing prompt, tools, middleware, memory, UI policy, or agent presets, they should build that profile system in their own app and pass the resulting pieces into this module.

The default host integration should use Geppetto registry sources. In a Pinocchio host, that can mean passing Pinocchio's `profiles.yaml` file if it is a Geppetto registry file, or passing a prebuilt `engineprofiles.RegistryReader`. Geppetto JS should not parse Pinocchio unified config documents directly.

### Credential policy: no environment variables in JS

The public JavaScript API must not expose `apiKey`, `apiKeyEnv`, `fromEnv`, or equivalent methods. JS scripts should never fetch provider credentials directly from environment variables. Instead, scripts choose a named credential reference and the Go host resolves that reference according to host policy.

Allowed:

```javascript
const settings = gp.inferenceProfiles
  .load("./profiles.yaml")
  .resolve("assistant");
```

Model/provider parameters are edited in the Geppetto registry file, not through JavaScript setters.

Forbidden in the public API:

```javascript
gp.inferenceSettings();                    // forbidden for now
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
  .tool("read_file", (t) => t
    .description("Read a repository file")
    .input(gp.schema.object({ path: gp.schema.string().required() }))
    .handler(({ path }) => fs.readFile(path, "utf8")))
  .goTool("grep")
  .events((events) => events
    .onStart((ev) => console.log("start", ev.inferenceId))
    .onTextDelta((ev) => process.stdout.write(ev.text)))
  .build();

const turn = gp.turn()
  .system("You are a careful code reviewer.")
  .user("Summarize the JS binding architecture.")
  .build();

const result = await agent.run(turn);
console.log(result.text());
```

Go-backed pieces:

- `agentBuilderRef`: stores a selected `InferenceSettings`/engine plus runtime policy, tool builder, event config, default run options.
- `agentRef`: stores compiled engine/session factory/tool registry/event sink policy.
- `runResultRef`: stores original/effective/output turns, usage, run metadata, text extraction helpers, and tool call summaries.

### Agent execution contract: explicit turns only

`Agent` should not expose `ask(prompt)` and should not own `.system(...)`. The execution API is intentionally turn-first:

```typescript
interface Agent {
  run(turn: Turn, options?: RunOptions): Promise<RunResult>;
  stream(turn: Turn, options?: RunOptions): RunHandle;
}
```

Why:

- Every run has an explicit input `Turn` that can be saved, inspected, replayed, and diffed.
- Multimodal input (images, files, prior assistant messages, tool results) is naturally represented in the turn.
- System prompts are normal turn content, not hidden agent policy.
- `gp.engine().run(turn)` and `gp.agent().run(turn)` share the same mental model.

`agent.run(turn)` must never mutate the caller's turn in place. It should return a result with traceability helpers:

```javascript
const result = await agent.run(turn);

result.inputTurn();     // clone/snapshot of the original user-supplied turn
result.effectiveTurn(); // after agent runtime metadata/tool config/middleware setup
result.outputTurn();    // final turn after inference/tool loop
result.text();          // convenience text extraction from outputTurn
```

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

Multimodal input should also be turn-first:

```javascript
const turn = gp.turn()
  .system("You are a careful visual reasoning assistant.")
  .user((m) => m
    .text("What is in this screenshot?")
    .imageFile("./screenshot.png"))
  .build();
```

Implementation notes:

- `TurnBuilder` should hold a `*turns.Turn` internally.
- Named methods call existing Go constructors such as `turns.NewUserTextBlock` and `turns.NewToolCallBlock`.
- `.metadata(key, value)` should validate canonical keys or namespaced keys.
- `.data(key, value)` should validate known keys where possible.
- multimodal user messages use a Go-owned `MessageBuilder` with `.text(...)`, `.imageFile(...)`, `.imageURL(...)`, and `.imageBytes(...)` methods.
- raw object import belongs only under `gp.unsafe`, not normal `TurnBuilder`.

### Proposed registry-resolved `InferenceSettings` and `engine()` APIs

For the first implementation pass, JavaScript does **not** get an `inferenceSettings()` builder. All provider/model/sampling/token/base URL/model metadata changes happen in Geppetto registry files. JavaScript loads a registry, resolves a profile, receives a Go-owned `InferenceSettings` wrapper, and passes that wrapper to the engine or agent.

```javascript
const registry = gp.inferenceProfiles.load("./profiles.yaml");
const settings = registry.resolve({ profile: "assistant" });
const engine = gp.engine().inference(settings).build();
```

The `InferenceSettings` wrapper may expose read-only utility methods:

```javascript
settings.toJSON(); // detached, redacted snapshot
settings.clone();  // Go-owned clone wrapper
settings.debug();  // redacted diagnostic view
```

It should not expose mutating setters such as `.model(...)`, `.temperature(...)`, `.apiKey(...)`, or `.apiKeyEnv(...)`. If a script needs different model parameters, it should select another registry profile or load another registry file.

Host-default profile resolution should look like this when the host provides a Geppetto registry chain:

```javascript
const settings = gp.inferenceProfiles.resolve("assistant");
const engine = gp.engine().inference(settings).build();
```

Validation examples:

- registry source must be a supported Geppetto registry source;
- resolved profile must contain valid `InferenceSettings`;
- credential references must be symbolic and host-resolvable at engine-build time;
- raw API keys and environment variable references are rejected because the JavaScript API does not own credentials;
- plain JavaScript objects are rejected by `engine().inference(...)`.

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
const result = await agent.run(gp.turn().user("hello").build());

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

This plan assumes the JS API uses existing Geppetto registry formats and does **not** expose a JavaScript `inferenceSettings()` builder in the first pass. Model/provider parameters are changed through registry files. JavaScript receives Go-owned `InferenceSettings` wrappers from `gp.inferenceProfiles.resolve(...)` / `registry.resolve(...)` and may inspect them through `toJSON`, `clone`, and `debug`.

### Phase 0: Contract lock and baseline inventory

Goal: freeze the intended public API before implementation. This phase should not break the current default test suite. Put hard-cut contract tests behind an explicit build tag until the implementation is ready to flip them into the normal suite.

Tasks:

1. Add a build-tagged contract test file, for example `hardcut_contract_test.go` with `//go:build geppetto_js_hardcut_contract`.
2. In that test, assert the final top-level `require("geppetto")` keys:
   - `agent`
   - `inferenceProfiles`
   - `turn`
   - `engine`
   - `tool`
   - `toolRegistry`
   - `embeddings`
   - `schema`
   - `events`
   - `unsafe`
3. In the same test, assert removed names are absent from the hard-cut public surface:
   - `chat`
   - `inferenceSettings` builder
   - `turns.newTurn` and map-first turn helpers
   - `engines.fromConfig` and map-first engine helpers
   - top-level `createBuilder`
   - top-level `createSession`
   - top-level `runInference`
   - ordinary `runner.run`
4. Add a package-level contract comment explaining that `InferenceSettings` objects are produced by registry resolution only in the first pass.
5. Document accepted `gp.inferenceProfiles.load(...)` source forms in the contract test comments and design docs:
   - YAML path
   - `yaml:PATH`
   - `yaml://PATH`
   - SQLite path
   - `sqlite:PATH`
   - `sqlite-dsn:DSN`
6. Run baseline focused tests without the hard-cut build tag:
   - `go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`
7. Optionally run the build-tagged contract test and record that it fails until Phase 1+ exports exist:
   - `go test -tags geppetto_js_hardcut_contract ./pkg/js/modules/geppetto -run TestHardCutPublicSurfaceContract -count=1`

Acceptance criteria:

- Default package tests remain green.
- The build-tagged contract test captures the desired hard-cut surface.
- The task list and design doc explicitly say there is no `gp.inferenceSettings()` builder in the first pass.

### Phase 1: Go-owned `InferenceSettings` result wrapper

Goal: implement the read-only Go wrapper returned by profile resolution.

Tasks:

1. Add `api_inference_settings.go` in `geppetto/pkg/js/modules/geppetto`.
2. Define immutable/copy-on-write `InferenceSettingsJS` wrapper around cloned `*settings.InferenceSettings`.
3. Store provenance metadata on the wrapper:
   - registry slug
   - profile slug
   - stack lineage
   - source metadata
4. Implement read-only helper methods:
   - `toJSON()` returns detached redacted snapshot
   - `clone()` returns another Go-owned wrapper
   - `debug()` returns redacted diagnostics
   - optional getters such as `model()` / `provider()` if useful and read-only
5. Explicitly do not implement mutating setters:
   - no `.provider(...)`
   - no `.model(...)`
   - no `.temperature(...)`
   - no `.credentialRef(...)`
   - no `.apiKey(...)`
   - no `.apiKeyEnv(...)`
6. Add tests proving snapshots are detached and mutating JS snapshots does not mutate Go-owned settings.
7. Add negative tests proving the forbidden mutating/credential methods are absent.

Acceptance criteria:

- `InferenceSettingsJS` is usable as a Go-owned wrapper.
- JavaScript cannot tweak model parameters directly.

### Phase 2: Geppetto registry loader wrapper

Goal: expose existing `engineprofiles` registry source loading to JS without Pinocchio.

Tasks:

1. Add or rewrite `api_inference_profiles.go` for the new `gp.inferenceProfiles` namespace.
2. Implement `gp.inferenceProfiles.load(source)` for string and string-array source inputs.
3. Use `engineprofiles.ParseRegistrySourceSpecs` and `engineprofiles.NewChainedRegistryFromSourceSpecs` internally.
4. Implement `InferenceRegistryJS` wrapper with registry reader, optional closer, and source metadata.
5. Implement `registry.resolve(input)` for string profile names and typed `{ registry, profile }` input snapshots.
6. Wrap `ResolvedEngineProfile.InferenceSettings` as `InferenceSettingsJS` with provenance metadata.
7. Implement `registry.listRegistries`, `registry.listProfiles`, and `registry.close`.
8. Implement host-default `gp.inferenceProfiles.resolve(...)` and clear error when no default registry is configured.
9. Add temporary registry YAML tests for `slug`, `default_profile_slug`, `profiles.default`, stacks, multiple source precedence, and invalid source errors.
10. Add explicit rejection/documentation test for Pinocchio unified config docs with `app:` as unsupported by `gp.inferenceProfiles.load(...)`.

Acceptance criteria:

- `gp.inferenceProfiles.load("profiles.yaml").resolve("assistant")` returns `InferenceSettingsJS`.
- No Pinocchio package import is introduced into Geppetto.

### Phase 3: Engine builder integration

Goal: compile engines only from registry-resolved Go-owned inference settings.

Tasks:

1. Add/update `EngineBuilderJS`.
2. Implement `gp.engine().inference(settings).build()` accepting only `InferenceSettingsJS` or trusted Go settings wrappers.
3. Remove/withhold public `fromConfig(map)` and map-first engine constructors.
4. Resolve symbolic credentials through host `CredentialResolver` only inside Go-side engine build.
5. Ensure raw credentials are never visible through JS snapshots/debug output.
6. Add tests for building engines from registry-resolved settings.
7. Add tests rejecting plain JS objects passed as inference settings.
8. Add tests for missing credential resolver and redacted debug output.

Acceptance criteria:

- Engine construction starts from registry-resolved `InferenceSettingsJS`.
- Credential resolution happens on Go side only.

### Phase 4: Agent API integration — explicit turns only

Goal: configure runtime behavior from JS while keeping all message content in explicit turns.

Tasks:

1. Add/update `api_agent.go`.
2. Implement `gp.agent()` builder methods: `name`, `inference`, `engine`, `middleware`, `goMiddleware`, `tool`, `goTool`, `toolLoop`, `events`, `runDefaults`, `build`.
3. Explicitly do **not** implement `agent.ask(prompt)`, `agent.system(prompt)`, `agent.profile(name)`, or first-pass `agent.inferenceProfile(name)`.
4. Ensure `.inference(...)` accepts `InferenceSettingsJS`, not profile names or JS maps.
5. Implement `agent.run(turn, options?)` requiring a Go-owned `Turn` wrapper.
6. Ensure `agent.run` clones input turn, applies runtime/tool/middleware setup to an effective turn, and never mutates caller input.
7. Implement `agent.stream(turn, options?)` with explicit turn requirement and runtime-owner-safe event/cancel/promise behavior.
8. Implement `RunResultJS` helpers: `inputTurn`, `effectiveTurn`, `outputTurn`, `text`, `usage`, `stopReason`, `events`, `toJSON`.
9. Add tests for fake/echo engine `agent.run(turn)`, explicit system block in turn, no `agent.ask`, no `agent.system`, JS tools, Go tools, middleware ordering, non-mutating input turns, and result turn traceability.

Acceptance criteria:

- Users compose runtime behavior with `gp.agent()` and message content with `gp.turn()`.
- Inference profile resolution supplies only settings.
- There is no hidden prompt-to-turn conversion in agent APIs.

### Phase 5: Tool, schema, turn, and multimodal message wrappers

Goal: remove remaining map-first construction from everyday scripts.

Tasks:

1. Implement `gp.schema` builders for object, string, integer, number, boolean, array, enum, required/default/min/max helpers.
2. Implement `gp.tool(name)` builder with `description`, `input`, `handler`, `build`.
3. Implement `gp.toolRegistry()` wrapper with `add`, `addGo`, `list`, `call`.
4. Implement `gp.turn()` builder with `system`, `user` string shorthand, `user(messageBuilderFn)`, `assistant`, `toolCall`, `toolResult`, `metadata`, `build`.
5. Implement `MessageBuilder` with `text`, `imageFile`, `imageURL`, and `imageBytes`.
6. Ensure all built schema/tool/turn/message objects are Go-owned wrappers with explicit snapshots.
7. Add invalid construction tests for schema/tool/turn/message wrappers.
8. Add multimodal image tests with deterministic fake provider or codec-level assertions.

Acceptance criteria:

- Example scripts no longer construct turn/block/tool maps directly.

### Phase 6: xgoja and host integration

Goal: make generated standalone binaries able to expose the same API safely.

Tasks:

1. Update Geppetto xgoja provider config schema for registry source configuration: `profileRegistries`, `defaultProfile`, and optional `allowRegistryLoad`.
2. Add host service wiring for default registry reader, credential resolver, and approved Go tool registry.
3. Ensure `allowRegistryLoad` defaults to safe host policy.
4. Add xgoja tests for requiring Geppetto, resolving default registry profiles, explicit registry load allow/deny policy, and host-only credential resolution.

Acceptance criteria:

- xgoja standalone apps can bundle Geppetto and profile registries without Pinocchio imports.

### Phase 7: Documentation, examples, and declaration generation

Goal: make the hard-cut API teachable and type-visible.

Tasks:

1. Update TypeScript declarations for `InferenceSettings`, `InferenceProfileNamespace`, `InferenceRegistry`, `EngineBuilder`, `AgentBuilder`, `ToolBuilder`, `SchemaNamespace`, `TurnBuilder`, `MessageBuilder`, and `RunResult`.
2. Update `dts_parity_test.go` for final top-level exports and absence of removed names.
3. Add examples: `01_load_registry_resolve_profile.js`, `02_engine_from_registry_profile.js`, `03_agent_from_registry_profile.js`, `04_tools_and_schema.js`, `05_multimodal_turn.js`, `06_embeddings_with_registry_profile.js`.
4. Document Geppetto registry YAML fields: `slug`, `default_profile_slug`, `profiles.<slug>`, `stack`, `inference_settings`.
5. Document that Pinocchio unified config docs are application-side and not loaded by `gp.inferenceProfiles.load(...)`.

Acceptance criteria:

- New examples run with deterministic/fake engines where possible.
- Live examples self-skip unless host credential refs are available.

### Phase 8: Cleanup and hard-cut removal

Goal: complete the hard cut.

Tasks:

1. Remove public exports for old map-first namespaces or move intentionally to `gp.unsafe`.
2. Remove docs/examples that teach old APIs as normal usage.
3. Keep internal codecs only where needed for snapshots/import tests.
4. Add regression tests that old public names stay absent.
5. Run focused tests: `go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`.
6. Run broader Geppetto and xgoja tests if registry/core/provider wiring changed.

Acceptance criteria:

- Public JS API is hard-cut, Go-wrapper based, explicit-turn based, and Geppetto-registry based.
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
- `agent().inference(fakeInferenceSettings).run(gp.turn().user("x").build())` or `agent().engine(echoEngine).run(turn)` returns deterministic text.
- `agent().tool(...).run(turn)` executes a JS tool through the existing toolloop.
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
| `gp.turns.newTurn({ blocks })` | `gp.turn().system(...).user(...).build()` |
| `gp.turns.newUserBlock(text)` | `gp.turn().user(text)` or `gp.block().user(text)` if standalone block builders are needed |
| `gp.engines.fromConfig(map)` | `gp.inferenceProfiles.load(...).resolve(...)` plus `gp.engine().inference(settings).build()` |
| `gp.createBuilder(options)` | `gp.agent()` for common flows, `gp.sessionBuilder()` for low-level flows |
| `gp.createSession(options)` | `gp.agent().buildSession()` or `gp.session(engine).build()` |
| `gp.runner.run({ engine, runtime, prompt })` | `gp.agent().inference(settings).tool(...).run(gp.turn().system(...).user(...).build())` |
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

Slice 3: gp.agent().engine(echo).run(gp.turn().user("hello").build())
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
