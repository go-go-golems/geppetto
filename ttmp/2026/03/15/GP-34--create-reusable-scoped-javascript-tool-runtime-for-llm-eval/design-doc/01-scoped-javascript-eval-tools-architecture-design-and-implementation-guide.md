---
Title: Scoped JavaScript eval tools architecture, design, and implementation guide
Ticket: GP-34
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - backend
    - js-bindings
    - go-api
    - security
    - inference
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/tools/definition.go
      Note: Canonical Geppetto tool definition and schema generation surface
    - Path: geppetto/pkg/inference/tools/registry.go
      Note: |-
        Tool registry contract the new package must target
        Tool registry contract that scopedjs must target
    - Path: geppetto/pkg/inference/tools/scopeddb/schema.go
      Note: |-
        Current reusable pattern for scoped package API shape
        Reusable scoped package API precedent
    - Path: geppetto/pkg/inference/tools/scopeddb/tool.go
      Note: |-
        Current reusable pattern for prebuilt and lazy registration helpers
        Reusable prebuilt and lazy registration precedent
    - Path: geppetto/pkg/js/modules/geppetto/api_tools_registry.go
      Note: |-
        Existing JS-side tool registry bridge
        Existing JS tool registry bridge and overlap boundary
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: |-
        Existing runtime-owned geppetto native module
        Current runtime-owned geppetto native module
    - Path: geppetto/pkg/js/runtime/runtime.go
      Note: |-
        Geppetto-owned JS runtime bootstrap on top of go-go-goja
        Current Geppetto JS runtime bootstrap
    - Path: geppetto/pkg/js/runtimebridge/bridge.go
      Note: |-
        Owner-thread bridge for invoking JS safely from Go
        Owner-thread bridge for safe JS invocation from Go
    - Path: go-go-goja/engine/factory.go
      Note: |-
        Underlying runtime factory builder and runtime initialization flow
        Underlying runtime factory composition flow
    - Path: go-go-goja/engine/module_specs.go
      Note: ModuleSpec and RuntimeInitializer contracts
    - Path: go-go-goja/modules/common.go
      Note: |-
        Native module registry and module documentation interface
        Native module interface and doc surface
    - Path: go-go-goja/modules/fs/fs.go
      Note: |-
        Concrete example of a documented native module
        Concrete documented native module example
    - Path: go-go-goja/pkg/jsdoc/export/export.go
      Note: |-
        JSDocEx export surface for JSON/YAML/Markdown/SQLite
        JSDocEx export surface
    - Path: go-go-goja/pkg/jsdoc/extract/extract.go
      Note: JSDocEx extraction pipeline
    - Path: go-go-goja/pkg/jsdoc/model/model.go
      Note: |-
        JSDocEx documentation model types
        JSDocEx symbol and package doc model
    - Path: go-go-goja/pkg/jsdoc/model/store.go
      Note: JSDocEx aggregate documentation store
    - Path: go-go-goja/pkg/runtimeowner/runner.go
      Note: |-
        Runtime owner thread safety and scheduling model
        Owner-thread scheduler and concurrency model
ExternalSources: []
Summary: Detailed intern-oriented architecture and implementation guide for a reusable Geppetto package that exposes configured goja runtimes as single LLM-facing eval tools.
LastUpdated: 2026-03-16T00:12:01.7634972-04:00
WhatFor: Give a new implementer enough system context, design rationale, API detail, and file-level guidance to build the scoped JavaScript eval tool package without first rediscovering the Geppetto and go-go-goja architecture.
WhenToUse: Use when implementing GP-34, reviewing the design, or onboarding to the Geppetto JavaScript runtime, module, and documentation stack.
---


# Scoped JavaScript eval tools architecture, design, and implementation guide

## Executive Summary

This ticket proposes a new reusable Geppetto package for exposing a prepared goja runtime as one LLM-facing tool such as `eval_dbserver`. The immediate precedent is the reusable scoped database package in `geppetto/pkg/inference/tools/scopeddb`, which extracted a repeated application pattern into a shared package while keeping the application-owned data-loading logic local to the application. GP-34 should do the same for JavaScript runtime tools.

As of the 2026-03-16 implementation checkpoints, the package has landed at `geppetto/pkg/inference/tools/scopedjs`. The public package name is `scopedjs`, and the runnable adoption examples live under:

- `geppetto/cmd/examples/scopedjs-tool`
- `geppetto/cmd/examples/scopedjs-dbserver`

Today the lower-level building blocks already exist:

- Geppetto already knows how to define and register tools through `ToolDefinition`, `ToolRegistry`, and `tools.NewToolFromFunc`.
- Geppetto already knows how to create an owned goja runtime and register the `geppetto` native module through `pkg/js/runtime/runtime.go`.
- `go-go-goja` already provides the underlying runtime factory, native module registration model, runtime ownership rules, and a documentation extraction/export subsystem named JSDocEx.

What does not exist yet is the package that assembles those parts into a reusable application pattern. Right now an application that wants to expose a runtime-scoped JavaScript environment to an LLM would need to re-decide:

- how the runtime is built,
- how modules and globals are registered,
- how helper scripts are loaded,
- how the tool input and output are shaped,
- how runtime state is scoped,
- and how the model learns what is available in the environment.

The proposed package, tentatively `geppetto/pkg/inference/tools/scopedjs`, should own those reusable mechanics. The application should only need to provide the app-owned parts: which modules and globals exist, which scripts are preloaded, how the scope is derived from context, and what documentation should be surfaced.

## Problem Statement

The user story is straightforward: an application wants to expose a prepared JavaScript environment to the model as a single tool. For example:

- register a filesystem module,
- inject a database client or scoped database object,
- load helper JS files,
- register an Obsidian control module,
- register a webserver module,
- and then expose everything as a single tool such as `eval_dbserver`.

The model should not have to understand Geppetto internals. It should see a single tool with a clear contract and a tool description that explains:

- what modules can be required,
- what globals already exist,
- which helpers were preloaded,
- what the tool is supposed to be used for,
- and a few starter snippets.

### Why the current system is not enough

The current system is powerful but fragmented.

#### Fragment 1: generic tool registration exists, but not the JS runtime pattern

`geppetto/pkg/inference/tools/definition.go` and `registry.go` already provide generic tool registration. This is enough to expose one function as one tool, but it does not tell an implementer how to build and scope an owned runtime environment that the function executes against.

#### Fragment 2: JS runtime bootstrapping exists, but not the LLM-facing tool contract

`geppetto/pkg/js/runtime/runtime.go` and the `go-go-goja/engine` package already create the runtime and wire `require`. That is necessary, but it still leaves open:

- which modules to register,
- how to document them,
- whether the runtime is rebuilt per call or kept per session,
- and what JSON input and output shape the tool should use.

#### Fragment 3: JS-to-Go bridging exists, but not the environment packaging abstraction

`geppetto/pkg/js/modules/geppetto/module.go`, `api_tools_registry.go`, and `runtimebridge/bridge.go` already show safe owner-thread interactions and JS-defined tools. Those pieces are real and useful, but they operate one layer below the desired feature. The user request is not "let JS define more tools." The request is "let the host expose one prepared runtime as one reusable tool."

#### Fragment 4: documentation exists, but is not connected to tool descriptions

`go-go-goja/modules/common.go` requires modules to provide `Doc() string`, and JSDocEx can extract much richer structured documentation from JS source into a `DocStore`. But there is currently no reusable flow that turns that documentation into the LLM-facing description of a runtime eval tool.

### The actual design problem

The design problem is not "how do we run JavaScript?" That already works.

The design problem is:

1. package the runtime environment cleanly,
2. scope it safely,
3. document it automatically,
4. register it as one tool,
5. and give applications both a prebuilt and a lazy path analogous to `scopeddb`.

If GP-34 does not solve those five problems together, it will create another low-level helper instead of a reusable application pattern.

## Proposed Solution

Create a new package, tentatively:

```text
geppetto/pkg/inference/tools/scopedjs
```

This package should mirror the successful separation already used by `scopeddb`.

### Responsibility split

#### Application-owned responsibilities

The application should own:

- the decision about which environment exists,
- the scope data needed to build it,
- which goja modules to register,
- which globals or host objects to inject,
- which JS bootstrap files or inline scripts to load,
- and the human-facing documentation for those modules/globals/helpers.

#### Geppetto-owned responsibilities

The shared package should own:

- the public package API,
- the LLM-facing eval tool registration,
- the JSON schema for tool inputs,
- the result envelope for outputs, console logs, and errors,
- runtime lifecycle and cleanup rules,
- tool description generation,
- and the prebuilt plus lazy registration helpers.

### High-level architecture

```text
               app code
                  |
                  v
        +---------------------+
        | EnvironmentSpec     |
        | - tool metadata     |
        | - eval options      |
        | - Configure(...)    |
        +---------------------+
                  |
                  v
        +---------------------+
        | scopedjs Builder    |
        | - modules           |
        | - globals           |
        | - bootstrap scripts |
        | - docs/manifest     |
        +---------------------+
                  |
                  v
        +---------------------+
        | BuildRuntime(...)   |
        | - go-go-goja factory|
        | - require registry  |
        | - runtime owner     |
        | - bootstrap load    |
        +---------------------+
                  |
                  v
        +---------------------+
        | RegisterPrebuilt /  |
        | NewLazyRegistrar    |
        +---------------------+
                  |
                  v
        +---------------------+
        | ToolDefinition      |
        | name: eval_xxx      |
        | input: EvalInput    |
        | output: EvalOutput  |
        +---------------------+
                  |
                  v
             provider tool call
```

### Documentation architecture

The documentation pipeline should be treated as part of the product, not an optional extra.

```text
NativeModule.Doc()        JSDocEx DocStore        app-authored notes
         |                      |                        |
         +----------+-----------+------------------------+
                    |
                    v
             EnvironmentManifest
                    |
                    v
             BuildDescription(...)
                    |
                    v
      ToolDefinition.Description for eval_xxx
```

The critical design principle is that the model should receive the environment contract at tool-advertisement time, not only in out-of-band prose docs.

## System Walkthrough For A New Intern

This section explains the current system in the order that a new implementer should learn it.

### 1. Learn the Geppetto tool system first

Relevant files:

- `geppetto/pkg/inference/tools/definition.go`
- `geppetto/pkg/inference/tools/registry.go`
- `geppetto/pkg/inference/tools/scopeddb/schema.go`
- `geppetto/pkg/inference/tools/scopeddb/tool.go`

#### What `ToolDefinition` is

`ToolDefinition` is Geppetto's generic representation of a tool that can be advertised to a provider and executed by the tool loop. It contains:

- `Name`
- `Description`
- `Parameters` (JSON schema)
- `Function`
- `Examples`
- `Tags`
- `Version`

That means the new scoped JS package does not need a new tool system. It needs to produce a normal `ToolDefinition`.

#### What `tools.NewToolFromFunc` does

`tools.NewToolFromFunc(...)` is important because it derives the JSON schema from a Go function's input type. The new package should use this exactly like `scopeddb` does.

Conceptually:

```pseudo
tool := NewToolFromFunc(
  name,
  description,
  func(ctx, input EvalInput) -> EvalOutput
)
registry.RegisterTool(tool.Name, tool)
```

#### Why `scopeddb` is the right template

`scopeddb` already solved the packaging problem for another bounded tool type.

Its design is worth studying before touching any JS code:

- `DatasetSpec[Scope, Meta]` says what a scoped dataset is.
- `BuildInMemory(...)` and `BuildFile(...)` construct a scoped resource.
- `RegisterPrebuilt(...)` registers an already-built resource.
- `NewLazyRegistrar(...)` builds the resource from context on demand.

That is exactly the pattern GP-34 should reuse, except the resource is a goja runtime instead of a SQLite database.

### 2. Learn the Geppetto JS runtime wrapper next

Relevant files:

- `geppetto/pkg/js/runtime/runtime.go`
- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/runtimebridge/bridge.go`

#### What `pkg/js/runtime/runtime.go` does

This file is Geppetto's current top-level JS bootstrap. It:

1. builds a go-go-goja factory,
2. creates a runtime,
3. creates a `require.Registry`,
4. optionally enables default go-go-goja modules,
5. registers the `geppetto` native module,
6. stores the `require` module on the runtime,
7. runs additional runtime initializers.

This means GP-34 does not need to invent runtime creation from scratch. It can either:

- build directly on this file,
- or extract a slightly thinner helper if the current bootstrap is too Geppetto-module-specific.

#### What `module.go` tells us

`geppetto/pkg/js/modules/geppetto/module.go` shows how runtime-bound APIs are exposed to JavaScript. The main ideas:

- the module is registered as `require("geppetto")`,
- it keeps runtime-scoped state in `moduleRuntime`,
- it uses a runtime owner bridge when it needs to invoke JS safely from Go,
- and it can expose tool registry helpers, profiles, engines, sessions, and more.

This file is not the place to implement GP-34, but it proves that Geppetto already has real runtime-scoped host integration patterns.

#### Why `runtimebridge` matters

Goja runtimes are not generally safe to manipulate from arbitrary goroutines. `runtimebridge/bridge.go` wraps the go-go-goja runtime owner so Go code can safely:

- call JS on the owner thread,
- post asynchronous work to the owner thread,
- and convert Go values to JS values inside the right runtime context.

Any scoped eval tool will need this discipline. If the implementation bypasses the owner thread, it will eventually become flaky or racy.

### 3. Learn the go-go-goja runtime engine below Geppetto

Relevant files:

- `go-go-goja/engine/factory.go`
- `go-go-goja/engine/module_specs.go`
- `go-go-goja/engine/runtime.go`
- `go-go-goja/pkg/runtimeowner/runner.go`

#### What the `FactoryBuilder` does

The go-go-goja engine package builds runtimes in two phases.

Phase 1: factory composition.

- collect static module registrations (`ModuleSpec`)
- collect runtime initializers (`RuntimeInitializer`)
- build an immutable factory

Phase 2: runtime creation.

- create a new VM
- create and start the event loop
- create the runtime owner
- enable `require`
- enable console
- run runtime initializers

Conceptually:

```pseudo
builder := engine.NewBuilder()
builder.WithModules(...)
builder.WithRuntimeInitializers(...)
factory := builder.Build()
runtime := factory.NewRuntime(ctx)
```

This is the execution substrate that GP-34 should package rather than duplicate.

#### What `ModuleSpec` and `RuntimeInitializer` mean

`ModuleSpec` is static registration applied when building the factory.

`RuntimeInitializer` is per-runtime work applied after the runtime exists.

That distinction is useful for GP-34:

- require-able native modules belong in module specs,
- runtime-bound globals and per-runtime bootstrap work belong in runtime initializers.

### 4. Learn the native module model

Relevant files:

- `go-go-goja/modules/common.go`
- `go-go-goja/modules/fs/fs.go`

The `modules.NativeModule` interface is:

```go
type NativeModule interface {
    Name() string
    Doc() string
    Loader(*goja.Runtime, *goja.Object)
}
```

This is important because it already carries both:

- runtime behavior via `Loader(...)`,
- and human-facing documentation via `Doc() string`.

The `fs` module is a useful concrete example:

- `Name()` returns `"fs"`
- `Doc()` returns a human-readable description of the module
- `Loader(...)` populates the module exports with Go functions

GP-34 should reuse that documentation surface instead of inventing a second unrelated module-doc mechanism. The likely direction is:

- accept `Doc()` strings as the minimum module doc source,
- allow richer docs later via JSDocEx or structured descriptors.

### 5. Learn the JSDocEx system separately from runtime execution

Relevant files:

- `go-go-goja/pkg/jsdoc/model/model.go`
- `go-go-goja/pkg/jsdoc/model/store.go`
- `go-go-goja/pkg/jsdoc/extract/extract.go`
- `go-go-goja/pkg/jsdoc/batch/batch.go`
- `go-go-goja/pkg/jsdoc/export/export.go`
- `go-go-goja/pkg/jsdoc/server/server.go`
- `go-go-goja/cmd/goja-jsdoc/doc/01-jsdoc-system.md`

JSDocEx is not the runtime system. It is the documentation extraction and export system. That distinction matters.

#### What JSDocEx gives us

It can:

- parse JS source,
- extract structured symbol docs, examples, and package docs,
- build a `DocStore`,
- export that store as JSON, YAML, Markdown, or SQLite,
- and serve it over HTTP with a browser UI.

#### Why GP-34 should care

GP-34 is not trying to build a documentation browser. But it does need a robust way to surface environment docs to the model. JSDocEx is useful because it already provides:

- a structured doc model (`SymbolDoc`, `Example`, `Package`, `DocStore`),
- a batch builder,
- and export formats that could be embedded into tool descriptions, debug logs, or precomputed manifests.

The key insight is:

`NativeModule.Doc()` is the minimum viable documentation path. JSDocEx is the scalable path.

## Proposed Package API

This section is the concrete API recommendation.

### Package location

Recommended:

```text
geppetto/pkg/inference/tools/scopedjs
```

Rejected names:

- `jsruntime`: too generic and overlaps with existing runtime bootstrap code.
- `evaltool`: describes the tool but not the scoping/environment pattern.
- `sandboxjs`: implies stronger isolation than goja currently guarantees.

### Core types

```go
package scopedjs

type ToolDescription struct {
	Summary         string
	Modules         []ModuleDoc
	Globals         []GlobalDoc
	Helpers         []HelperDoc
	BootstrapFiles  []string
	StarterSnippets []string
	Notes           []string
}

type ToolDefinitionSpec struct {
	Name        string
	Description ToolDescription
	Tags        []string
	Version     string
}

type EvalOptions struct {
	Timeout        time.Duration
	MaxOutputChars int
	CaptureConsole bool
	StateMode      StateMode
}

type StateMode string

const (
	StatePerCall    StateMode = "per_call"
	StatePerSession StateMode = "per_session"
	StateShared     StateMode = "shared"
)

type ScopeResolver[Scope any] func(ctx context.Context) (Scope, error)

type EnvironmentSpec[Scope any, Meta any] struct {
	RuntimeLabel string
	Tool         ToolDefinitionSpec
	DefaultEval  EvalOptions

	Configure func(ctx context.Context, b *Builder, scope Scope) (Meta, error)
}

type BuildResult[Meta any] struct {
	Runtime  *gojengine.Runtime
	Meta     Meta
	Manifest EnvironmentManifest
	Cleanup  func() error
}
```

### Why `Configure` is the right callback

`scopeddb` uses `Materialize(...)` to let the application fill a freshly built database. The JS equivalent should be a `Configure(...)` callback that receives a mutable builder.

That callback should be the place where the application says:

- add module `fs`,
- inject global `db`,
- add global `obsidian`,
- load `bootstrap/router.js`,
- attach helper docs,
- return any useful `Meta`.

### Builder API

```go
type Builder struct {
	AddModule(name string, register func(*require.Registry) error, doc ModuleDoc)
	AddNativeModule(mod modules.NativeModule)
	AddGlobal(name string, bind GlobalBinding, doc GlobalDoc)
	AddInitializer(init gojengine.RuntimeInitializer)
	AddBootstrapSource(name string, src string)
	AddBootstrapFile(path string)
	AddHelper(name string, signature string, description string)
}

type GlobalBinding func(ctx context.Context, rt *gojengine.Runtime) error

type ModuleDoc struct {
	Name        string
	Description string
	Exports     []string
}

type GlobalDoc struct {
	Name        string
	Type        string
	Description string
}

type HelperDoc struct {
	Name        string
	Signature   string
	Description string
}

type EnvironmentManifest struct {
	Modules        []ModuleDoc
	Globals        []GlobalDoc
	Helpers        []HelperDoc
	BootstrapFiles []string
}
```

### Important implementation detail about globals

Globals are not the same thing as require modules.

- require modules are registered against the `require.Registry` before runtime creation,
- globals are set on the runtime after it exists and therefore should be implemented as runtime initializers or owner-thread bindings.

That is why the builder should store them separately.

### Tool input and output

Recommended first-pass tool I/O:

```go
type EvalInput struct {
	Code  string         `json:"code"`
	Input map[string]any `json:"input,omitempty"`
}

type ConsoleLine struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

type EvalOutput struct {
	Result     any           `json:"result,omitempty"`
	Console    []ConsoleLine `json:"console,omitempty"`
	Error      string        `json:"error,omitempty"`
	DurationMs int64         `json:"durationMs,omitempty"`
}
```

#### Why this input shape is recommended

It is intentionally small.

- `Code` is the actual JS body the model wants to run.
- `Input` is optional structured data the host can inject as a variable such as `input`.

This avoids overdesigning the first version around subcommands or multiple execution modes. The tool can always grow later if needed.

### Execution model

Recommended execution convention:

```pseudo
async function __scoped_eval__(input) {
  // model-provided code goes here
}
return await __scoped_eval__(input)
```

That gives the model a clean way to:

- use `await`,
- return a structured result with `return`,
- and avoid relying on "last expression" semantics.

### Registration helpers

The package should expose helpers parallel to `scopeddb`.

```go
func BuildRuntime[Scope any, Meta any](
	ctx context.Context,
	spec EnvironmentSpec[Scope, Meta],
	scope Scope,
) (*BuildResult[Meta], error)

func RegisterPrebuilt[Scope any, Meta any](
	reg tools.ToolRegistry,
	spec EnvironmentSpec[Scope, Meta],
	handle *BuildResult[Meta],
	opts EvalOptions,
) error

func NewLazyRegistrar[Scope any, Meta any](
	spec EnvironmentSpec[Scope, Meta],
	resolve ScopeResolver[Scope],
	opts EvalOptions,
) func(reg tools.ToolRegistry) error
```

These should behave like their `scopeddb` counterparts:

- `RegisterPrebuilt(...)` uses an already-built runtime handle.
- `NewLazyRegistrar(...)` resolves scope from context and builds on demand.

## Detailed Execution Flow

This section shows how one tool call should execute.

### Prebuilt runtime case

```text
app boot
  -> BuildRuntime(scope)
  -> RegisterPrebuilt(toolRegistry, handle)

provider calls eval_dbserver
  -> tool executor invokes tool fn
  -> tool fn posts execution to runtime owner
  -> runtime executes wrapped async function
  -> console is captured
  -> result is exported to Go
  -> EvalOutput returned
```

Pseudocode:

```pseudo
function RegisterPrebuilt(reg, spec, handle, opts):
  description = BuildDescription(spec.Tool.Description, handle.Manifest, opts)
  def = NewToolFromFunc(spec.Tool.Name, description, func(ctx, in EvalInput) -> EvalOutput:
    return RunEval(ctx, handle.Runtime, in, opts)
  )
  reg.RegisterTool(def.Name, def)
```

### Lazy runtime case

```text
provider calls eval_dbserver
  -> tool fn starts
  -> resolve scope from ctx
  -> BuildRuntime(scope)
  -> defer Cleanup()
  -> RunEval(...)
  -> return EvalOutput
```

Pseudocode:

```pseudo
function NewLazyRegistrar(spec, resolve, opts):
  return func(reg):
    def = NewToolFromFunc(spec.Tool.Name, description, func(ctx, in EvalInput):
      scope = resolve(ctx)
      handle = BuildRuntime(ctx, spec, scope)
      defer handle.Cleanup()
      return RunEval(ctx, handle.Runtime, in, opts)
    )
    reg.RegisterTool(def.Name, def)
```

## Design Decisions

### Decision A: mirror `scopeddb` instead of inventing a new package style

Rationale:

- the pattern already works in this codebase,
- it is familiar to maintainers,
- and it gives applications both prebuilt and lazy paths.

### Decision B: default to per-call runtime state

Recommended default:

- `StateMode = per_call`

Rationale:

- easiest to reason about,
- avoids stale mutable state across unrelated tool calls,
- avoids session ownership complexity in v1,
- and better matches the bounded-scoped-tool philosophy.

`per_session` should exist only as an explicit second mode. `shared` should be the most advanced and least default mode.

### Decision C: use the existing go-go-goja runtime owner, not ad-hoc locking

Rationale:

- `pkg/runtimeowner/runner.go` already defines the safe execution model,
- `runtimebridge` already exposes the necessary bridge APIs,
- and bypassing them would reintroduce concurrency hazards the existing code already solved.

### Decision D: support both minimal and rich documentation sources

Rationale:

- the minimal path is immediate: `NativeModule.Doc()` plus app-authored prose,
- the rich path can later incorporate JSDocEx `DocStore` content for helpers and bootstrap files.

This keeps v1 implementable without blocking on a full documentation pipeline redesign.

### Decision E: the tool description is part of the runtime contract

Rationale:

If the tool description does not explain the environment, the model must infer everything by trial and error. That is expensive, brittle, and unnecessary. The package should generate descriptions deliberately, exactly as `scopeddb` enumerates allowed tables and starter queries.

## Tool Description Strategy

The description builder should produce prose like:

```text
Execute JavaScript inside the scoped dbserver runtime.
Available modules: fs, webserver, obsidian.
Available globals: db, appConfig.
Bootstrap files already loaded: bootstrap/router.js, bootstrap/sql_helpers.js.
Use return to provide the final result.
Starter snippets: const rows = await db.query(...); return rows.
```

### Suggested description algorithm

```pseudo
function BuildDescription(desc, manifest, opts):
  parts = []
  parts.append(ensureSentence(desc.Summary or "Execute JavaScript inside a prepared scoped runtime"))

  if manifest.Modules not empty:
    parts.append("Available modules: " + join(module names))

  if manifest.Globals not empty:
    parts.append("Available globals: " + join(global names))

  if manifest.Helpers not empty:
    parts.append("Helpers: " + render short helper signatures)

  if manifest.BootstrapFiles not empty:
    parts.append("Bootstrap files already loaded: " + join(file names))

  parts.append("Use return to provide the final result.")

  if opts.StateMode == per_call:
    parts.append("Each call runs in a fresh runtime.")

  for note in desc.Notes:
    parts.append(note)

  if desc.StarterSnippets not empty:
    parts.append("Starter snippets: " + join(snippets, " | "))

  return join(parts, " ")
```

## JSDocEx Integration Plan

JSDocEx should not be required for v1, but the package should be designed so it can benefit from it cleanly.

### Stage 1: minimal documentation path

Sources:

- `NativeModule.Doc()`
- builder-provided `ModuleDoc`, `GlobalDoc`, `HelperDoc`
- builder-provided notes and starter snippets

Benefits:

- simple,
- no extra parsing step,
- enough for first production use.

### Stage 2: optional structured documentation path

Potential integration:

- accept a `*model.DocStore`,
- or accept a precomputed documentation manifest generated from it,
- or run JSDocEx during build to extract docs from helper JS files.

Possible use cases:

- include helper function parameter docs,
- include examples extracted from bootstrap JS,
- export environment docs to SQLite or Markdown for debugging or operator docs,
- show runtime docs in both the tool description and separate human-facing docs.

### Important boundary

JSDocEx is for documentation extraction and export. It should not become the runtime configuration system itself. Runtime configuration should stay in `EnvironmentSpec` and `Builder`.

## Security And Safety Considerations

This package is not a security sandbox in the operating-system sense. It is a scoped runtime composition pattern. That distinction must be written down clearly.

### What the package can realistically control

- which modules are registered,
- which globals are available,
- which scripts are preloaded,
- how long eval can run before timeout,
- whether the runtime is fresh or persistent,
- how much console output is returned.

### What the package cannot guarantee alone

- full host isolation if dangerous modules are registered,
- filesystem or network safety if the provided modules are powerful,
- policy correctness if application-provided globals expose too much capability.

### Practical rule for implementers

Treat every registered module and global as a capability grant. If you give the runtime a full filesystem module and a full database handle, the model has those powers. GP-34 should make that power explicit and documented, not implicit.

## Alternatives Considered

### Alternative 1: do nothing and let each app wire its own runtime tool

Rejected because:

- the same packaging decisions would be repeated in every app,
- documentation quality would drift,
- and there would be no standard prebuilt/lazy registration pattern.

### Alternative 2: expose every module as a separate tool instead of one eval tool

Rejected as the primary solution because:

- it loses the coherence of a single programmable environment,
- it becomes awkward for workflows where the model wants to compose multiple capabilities in one script,
- and it does not match the motivating `db + webserver + obsidian` use case.

This may still be complementary in some applications, but it is not the requested product.

### Alternative 3: implement the feature inside `pkg/js/runtime` only

Rejected because:

- `pkg/js/runtime` is runtime bootstrap infrastructure,
- not the right home for a tool-specific scoped environment abstraction,
- and not parallel to the `pkg/inference/tools/scopeddb` pattern.

### Alternative 4: make JSDocEx mandatory for all environment docs

Rejected for v1 because:

- it adds unnecessary coupling to the first implementation,
- not all host-provided modules will have JS source to parse,
- and `NativeModule.Doc()` already provides a workable baseline.

## Implementation Plan

This section is intentionally detailed for a new intern.

### Phase 1: create the package skeleton

Create:

```text
geppetto/pkg/inference/tools/scopedjs/
  schema.go
  description.go
  eval.go
  tool.go
  docs.go
  schema_test.go
  description_test.go
  eval_test.go
  tool_test.go
```

Responsibilities:

- `schema.go`: core types (`EnvironmentSpec`, `BuildResult`, docs structs, options)
- `description.go`: description rendering
- `eval.go`: runtime build and execution helpers
- `tool.go`: `RegisterPrebuilt` and `NewLazyRegistrar`
- `docs.go`: optional doc manifest assembly helpers

### Phase 2: define the builder and runtime build path

Implement:

- builder data structures,
- collection of module specs and runtime initializers,
- bootstrap script loading,
- manifest accumulation for docs,
- cleanup handling.

Implementation advice:

- do not immediately over-abstract this,
- store plain slices for modules, globals, bootstrap files, and docs,
- keep the transformation into go-go-goja specs explicit and readable.

### Phase 3: implement `BuildRuntime(...)`

Algorithm:

```pseudo
function BuildRuntime(ctx, spec, scope):
  ensure spec.Configure exists
  b = NewBuilder()
  meta = spec.Configure(ctx, b, scope)

  engineBuilder = gojengine.NewBuilder()
  engineBuilder.WithModules(b.moduleSpecs...)
  engineBuilder.WithRuntimeInitializers(b.runtimeInitializers...)
  factory = engineBuilder.Build()
  runtime = factory.NewRuntime(ctx)

  run bootstrap scripts on owner thread

  return BuildResult{
    Runtime: runtime,
    Meta: meta,
    Manifest: b.Manifest(),
    Cleanup: runtime.Close,
  }
```

Key subtlety:

bootstrap scripts should execute only after:

- require has been enabled,
- modules have been registered,
- globals have been installed.

### Phase 4: implement `RunEval(...)`

Responsibilities:

- create a wrapped async function,
- inject `input`,
- execute on the runtime owner,
- capture console if configured,
- export the result back to Go,
- normalize exceptions and timeouts into `EvalOutput`.

Pseudocode:

```pseudo
function RunEval(ctx, runtime, in, opts):
  start = now()
  wrapped = "async function __scoped_eval__(input) { " + in.Code + "\n }\n__scoped_eval__(input)"

  ret = runtime.Owner.Call(ctx, "scopedjs.eval", func(ctx, vm):
    set runtime variable input = in.Input
    execute wrapped JS
    await promise resolution if needed
    collect result and console lines
    return exported value
  )

  return EvalOutput{
    Result: ret,
    Console: captured,
    DurationMs: elapsed,
  }
```

### Phase 5: implement description rendering

Start simple.

Inputs:

- `ToolDescription`
- `EnvironmentManifest`
- `EvalOptions`

Outputs:

- one clear description string

Test cases:

- no docs supplied,
- only modules,
- modules plus globals,
- bootstrap files present,
- state mode notes,
- starter snippets and notes.

### Phase 6: register as a normal Geppetto tool

This should be intentionally boring. Follow `scopeddb/tool.go`.

- create a `ToolDefinition` from `EvalInput -> EvalOutput`
- copy tags and version
- register it in a `ToolRegistry`

If this part becomes complicated, the design has probably leaked too much runtime-building complexity into the registration layer.

### Phase 7: write examples and docs

Add:

- one small example with `fs`,
- one example with a fake `db` global,
- one example showing lazy context-based build,
- and one documentation page or tutorial in `geppetto/pkg/doc`.

The examples are not optional. This package is too architectural to ship without a readable example.

### Phase 8: test the intended modes

At minimum test:

- prebuilt runtime registration,
- lazy runtime registration,
- per-call fresh state,
- timeout path,
- JS exception path,
- bootstrap script execution,
- module/global visibility,
- description generation,
- console capture.

## Suggested Test Matrix

```text
Case                               Expectation
---------------------------------  ------------------------------------------
fresh runtime per call             global mutations do not persist
persistent runtime mode            mutations persist when explicitly enabled
missing Configure callback         build fails clearly
module registration failure        build fails clearly
bootstrap syntax error             tool returns structured error
runtime timeout                    tool returns timeout-style error
console capture enabled            output includes console lines
console capture disabled           console lines omitted
lazy scope resolver error          tool returns structured error, not panic
```

## Suggested File-Level Work Breakdown

For a new intern, the most effective reading and implementation order is:

1. `geppetto/pkg/inference/tools/scopeddb/schema.go`
2. `geppetto/pkg/inference/tools/scopeddb/tool.go`
3. `geppetto/pkg/inference/tools/definition.go`
4. `go-go-goja/engine/factory.go`
5. `go-go-goja/engine/module_specs.go`
6. `go-go-goja/pkg/runtimeowner/runner.go`
7. `geppetto/pkg/js/runtime/runtime.go`
8. `geppetto/pkg/js/runtimebridge/bridge.go`
9. `go-go-goja/modules/common.go`
10. `go-go-goja/modules/fs/fs.go`
11. `go-go-goja/pkg/jsdoc/model/*`
12. `go-go-goja/pkg/jsdoc/extract/extract.go`

That reading order moves from the target package pattern outward, rather than from the deepest infrastructure upward.

## Open Questions

The following questions should stay visible until implementation starts:

1. Should v1 support only `per_call`, with `per_session` explicitly deferred?
2. Should bootstrap scripts be allowed to mutate shared runtime state in persistent modes, or should they always rerun on each call?
3. Should `EvalInput` remain raw `code + input`, or do we need a richer request envelope immediately?
4. Should JSDocEx integration appear in v1, or should v1 use only `Doc()` plus structured builder docs?
5. Do we want a separate helper to expose host Go tools inside the runtime as callable JS functions, or is that out of scope for GP-34?

## Recommended Starting Point For Implementation

If I were handing this to a new intern, I would tell them:

1. Copy the structural shape of `scopeddb`, not its database details.
2. Build the smallest possible `EnvironmentSpec`, `Builder`, and `EvalInput/EvalOutput`.
3. Make `per_call` mode work first.
4. Use `NativeModule.Doc()` plus app-authored docs first.
5. Add JSDocEx-driven enrichment only after the core tool works end to end.

That path keeps the first implementation coherent and testable.

## References

- `geppetto/pkg/inference/tools/definition.go`
- `geppetto/pkg/inference/tools/registry.go`
- `geppetto/pkg/inference/tools/scopeddb/schema.go`
- `geppetto/pkg/inference/tools/scopeddb/tool.go`
- `geppetto/pkg/js/runtime/runtime.go`
- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/modules/geppetto/api_tools_registry.go`
- `geppetto/pkg/js/runtimebridge/bridge.go`
- `go-go-goja/engine/factory.go`
- `go-go-goja/engine/module_specs.go`
- `go-go-goja/pkg/runtimeowner/runner.go`
- `go-go-goja/modules/common.go`
- `go-go-goja/modules/fs/fs.go`
- `go-go-goja/pkg/jsdoc/model/model.go`
- `go-go-goja/pkg/jsdoc/model/store.go`
- `go-go-goja/pkg/jsdoc/extract/extract.go`
- `go-go-goja/pkg/jsdoc/export/export.go`
- `go-go-goja/cmd/goja-jsdoc/doc/01-jsdoc-system.md`

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
