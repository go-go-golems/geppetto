---
Title: Scopedjs and demo review cleanup analysis design and implementation guide
Ticket: GP-36
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - js-bindings
    - pinocchio
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/schema.go
      Note: Core public API surface for scopedjs
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/tool.go
      Note: Registration behavior and one of the main lifecycle mismatches
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/eval.go
      Note: Eval contract, option merging, and rejection handling behavior
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/pinocchio/cmd/examples/scopedjs-tui-demo/main.go
      Note: TUI demo shell with heavy duplication from scopeddb demo
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/pinocchio/cmd/examples/scopedjs-tui-demo/renderers.go
      Note: TUI demo renderer layer with shared helper duplication
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/pinocchio/cmd/examples/scopeddb-tui-demo/main.go
      Note: Comparison baseline for duplicated demo shell logic
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/pinocchio/cmd/examples/scopeddb-tui-demo/renderers.go
      Note: Comparison baseline for duplicated renderer helper logic
ExternalSources: []
Summary: Thorough review of the scopedjs package and scopedjs demo work since origin/main, with findings, cleanup strategy, and an intern-oriented architecture guide.
LastUpdated: 2026-03-16T22:12:00-04:00
WhatFor: Help a new engineer understand what landed since origin/main, what is solid, what is fuzzy, and how to simplify or harden the design without breaking the useful parts.
WhenToUse: Use when reviewing scopedjs, planning cleanup work, or onboarding an intern to the scopedjs and scopedjs-demo code paths.
---

# Scopedjs and demo review cleanup analysis design and implementation guide

## Executive Summary

This review covers the work added since `origin/main` in two repositories:

- `geppetto`: the new `pkg/inference/tools/scopedjs` package, its examples, and its docs
- `pinocchio`: the new `cmd/examples/scopedjs-tui-demo` demo and its ticket docs

The high-level conclusion is positive: the new work proves the `scopedjs` concept end to end, has decent test coverage for initial behavior, and is already useful enough to power a real demo. The package is small, the host-side API is understandable, and the examples make the intended adoption path concrete.

The cleanup need is not that the feature is bad. The cleanup need is that the first implementation leaves several architectural edges fuzzy:

1. runtime lifecycle semantics are misleading, especially around `StateMode`
2. lazy registrations lose most of the model-facing capability description
3. eval option merging is not expressive enough to override booleans safely
4. the Pinocchio demo copied a large amount of `scopeddb` demo shell and renderer scaffolding instead of extracting a shared demo harness

That means the right next step is not a rewrite. The right next step is a focused cleanup pass that:

- clarifies lifecycle semantics in API and docs
- makes lazy and prebuilt description quality consistent
- removes duplicated demo infrastructure
- trims dead or misleading abstractions before more adopters build on them

## Problem Statement

The newly landed `scopedjs` package is the first reusable abstraction for exposing a prepared JavaScript runtime as one Geppetto tool. At the same time, the Pinocchio TUI demo proves that the package can drive a user-facing workflow. Because the feature was built in several slices, there is now enough code to review for long-term quality.

The user asked specifically for a cleanup-oriented review of:

- duplicated code
- deprecated or removable compatibility shims
- messy or unidiomatic code
- fuzzy architecture
- migration and backwards-compatibility baggage that can be removed

That means this document is not only a design guide. It is also a maintenance audit. The goal is to separate:

- what should remain as stable package surface
- what should be refactored into shared helpers
- what should be clarified before other teams depend on it

## Scope of Review

### Commits reviewed

#### Geppetto

- `6221675` `feat(scopedjs): add core api and description layer`
- `e4253c5` `feat(scopedjs): add runtime build and eval execution`
- `cf45f92` `feat(scopedjs): register prebuilt and lazy eval tools`
- `9d63530` `feat(scopedjs): add runnable examples and adoption docs`

#### Pinocchio

- `61a1b61` `feat(scopedjs-demo): scaffold runtime fixtures and smoke tests`
- `7313e2b` `feat(scopedjs-demo): wire pinocchio command shell`
- `2f7be40` `feat(scopedjs-demo): render eval calls and results`
- `e65d08f` `feat(scopedjs-demo): polish runtime behavior and demo guide`
- `5c46394` `docs(GP-033): record final validation and acceptance`

### Surface area size

The review diff is roughly:

- `geppetto/pkg/inference/tools/scopedjs/*.go`: 1,355 LoC including tests
- `pinocchio/cmd/examples/scopedjs-tui-demo/*.go`: 1,243 LoC including tests
- new docs and examples on top of that

Two numbers are especially important:

- `pinocchio/cmd/examples/scopedjs-tui-demo/main.go`: 233 lines
- `pinocchio/cmd/examples/scopeddb-tui-demo/main.go`: 233 lines

That exact size match is the first signal that some shell logic was copied rather than factored.

## Proposed Solution

The proposed cleanup strategy is:

1. keep the core `scopedjs` package shape
2. clarify or simplify lifecycle semantics
3. separate static capability description from dynamic runtime construction
4. fix the option override model before more callers depend on it
5. extract duplicated example and renderer infrastructure in Pinocchio

This is a cleanup-and-hardening pass, not a redesign from scratch.

## Current-State Architecture

### Geppetto package structure

The new package lives under:

```text
pkg/inference/tools/scopedjs/
  schema.go
  builder.go
  runtime.go
  eval.go
  description.go
  tool.go
  helpers.go
  *_test.go
```

The intent of each file is reasonably clean:

- `schema.go`
  - public types
- `builder.go`
  - host-side registration of modules, globals, bootstrap helpers, and docs
- `runtime.go`
  - convert builder state into a live runtime
- `eval.go`
  - execute model-provided JS inside that runtime
- `description.go`
  - synthesize model-facing tool prose
- `tool.go`
  - register the runtime as a Geppetto tool

This is a good package layout for a first feature. A new engineer can navigate it without deep repository context.

### Runtime flow

The current runtime flow looks like this:

```text
host application
    |
    v
EnvironmentSpec[Scope, Meta]
    |
    v
Configure(ctx, builder, scope)
    |
    v
Builder state
  modules
  globals
  bootstrap files
  helper docs
    |
    v
BuildRuntime(...)
    |
    v
BuildResult
  Runtime
  Meta
  Manifest
  Cleanup
    |
    +--> RegisterPrebuilt(...)
    |
    +--> NewLazyRegistrar(...)
            |
            v
          RunEval(...)
```

That architecture is sensible. The problem is not the broad shape. The problem is that some of the names imply stronger lifecycle semantics than the implementation currently enforces.

### Pinocchio demo structure

The new demo lives under:

```text
cmd/examples/scopedjs-tui-demo/
  main.go
  environment.go
  fake_data.go
  renderers.go
  *_test.go
  README.md
```

This matches the existing `scopeddb` demo shape closely:

- one file for shell wiring
- one file for environment and runtime composition
- one file for fixtures
- one file for custom timeline renderers

That consistency is useful for onboarding. The downside is that the implementation copied large chunks of existing shell and renderer infrastructure instead of introducing a shared example harness.

## Review Findings

### Finding 1: `StateMode` is an API promise that the runtime does not actually implement

Problem: The public API suggests three meaningful runtime lifecycle modes, but the implementation does not enforce them. Today `StateMode` mostly affects generated prose in the tool description, not actual runtime reuse behavior.

Where to look:

- `geppetto/pkg/inference/tools/scopedjs/schema.go:25-38`
- `geppetto/pkg/inference/tools/scopedjs/description.go:32-39`
- `geppetto/pkg/inference/tools/scopedjs/tool.go:10-23`
- `geppetto/pkg/inference/tools/scopedjs/tool.go:36-63`

Example:

```go
type StateMode string

const (
    StatePerCall    StateMode = "per_call"
    StatePerSession StateMode = "per_session"
    StateShared     StateMode = "shared"
)
```

```go
switch opts.StateMode {
case StatePerCall:
    parts = append(parts, "Each call uses a fresh runtime.")
case StatePerSession:
    parts = append(parts, "Runtime state persists within the current session.")
case StateShared:
    parts = append(parts, "Runtime state may be shared across calls.")
}
```

```go
func(ctx context.Context, in EvalInput) (EvalOutput, error) {
    return RunEval(ctx, handle.Runtime, in, evalOpts)
}
```

Why it matters: `RegisterPrebuilt(...)` always reuses the same `handle.Runtime` for every call. That is shared runtime state whether the description says so or not. If `evalOpts.StateMode` is left at the default `StatePerCall`, the description claims "Each call uses a fresh runtime" while the closure actually uses one persistent runtime instance. `NewLazyRegistrar(...)` is build-per-call, but `StatePerSession` and `StateShared` still are not implemented there either.

Cleanup sketch:

```text
Option A: remove the fake enum and expose honest runtime reuse semantics
Option B: keep the enum but actually implement runtime strategies
```

Lower-risk cleanup:

```go
type RegistrationMode string

const (
    RegistrationPrebuiltShared RegistrationMode = "prebuilt_shared"
    RegistrationLazyPerCall    RegistrationMode = "lazy_per_call"
)
```

### Finding 2: lazy registrations lose most of the model-facing capability description

Problem: Lazy registrations currently build the tool description from an empty manifest, so the model does not get module, global, or helper documentation in the dynamic-scope case.

Where to look:

- `geppetto/pkg/inference/tools/scopedjs/tool.go:45-48`
- `geppetto/pkg/inference/tools/scopedjs/description.go:17-28`

Example:

```go
def, err := tools.NewToolFromFunc(
    spec.Tool.Name,
    BuildDescription(spec.Tool.Description, EnvironmentManifest{}, evalOpts),
    func(ctx context.Context, in EvalInput) (EvalOutput, error) {
        scope, err := resolve(ctx)
        ...
        handle, err := BuildRuntime(ctx, spec, scope)
```

Why it matters: `BuildDescription(...)` is designed to enrich the tool description with available modules, globals, helpers, and bootstrap files. But `NewLazyRegistrar(...)` throws that away by passing `EnvironmentManifest{}`. As a result, prebuilt tools advertise the real runtime shape while lazy tools advertise only the manual summary and notes.

Cleanup sketch:

```go
type EnvironmentSpec[Scope any, Meta any] struct {
    RuntimeLabel string
    Tool         ToolDefinitionSpec
    DefaultEval  EvalOptions
    Describe     func(b *Builder) error
    Configure    func(ctx context.Context, b *Builder, scope Scope) (Meta, error)
}
```

Use `Describe(...)` for static capability docs and `Configure(...)` for scope-bound runtime state.

### Finding 3: `EvalOptions` merging is not expressive enough to override booleans cleanly

Problem: `resolveEvalOptions(...)` can override durations and integers in both directions, but `CaptureConsole` is effectively one-way. You can turn it on with an override, but you cannot explicitly turn it off if the base spec turned it on.

Where to look:

- `geppetto/pkg/inference/tools/scopedjs/eval.go:235-256`
- `geppetto/pkg/inference/tools/scopedjs/schema.go:33-46`

Example:

```go
if override.CaptureConsole {
    base.CaptureConsole = true
}
```

Why it matters: `bool` does not distinguish between "false because the caller explicitly wants false" and "false because the caller did not specify anything". Today that means a default `CaptureConsole=true` cannot be overridden to false through the public override path.

Cleanup sketch:

```go
type EvalOptionOverrides struct {
    Timeout        *time.Duration
    MaxOutputChars *int
    CaptureConsole *bool
    StateMode      *StateMode
}
```

### Finding 4: the Pinocchio demo duplicated the scopeddb demo shell almost line for line

Problem: The new `scopedjs` demo reused the `scopeddb` demo by copying it rather than extracting a shared shell helper. The result works, but it leaves Pinocchio with two example commands that will drift.

Where to look:

- `pinocchio/cmd/examples/scopedjs-tui-demo/main.go:30-233`
- `pinocchio/cmd/examples/scopeddb-tui-demo/main.go:30-233`

Example:

```diff
- accountID
+ workspaceID

- buildDemoRegistry(ctx, demoScope{AccountID: accountID})
+ buildDemoRegistry(ctx, workspaceID)

- chat.WithTitle("scopeddb support history demo")
+ chat.WithTitle("scopedjs project ops demo")
```

Why it matters: The shared logic includes profile resolution, event router setup, Watermill sink wiring, Bubble Tea program bootstrapping, and status bar style construction. If any of that changes, Pinocchio now has at least two copies to update.

Cleanup sketch:

```text
pinocchio/cmd/examples/internal/tuidemo/
  shell.go
  profiles.go
  statusbar.go
```

```go
type DemoSpec struct {
    Title            string
    TimelineRegister func(*timeline.Registry)
    StatusBar        func() string
    BuildBackend     func(ctx context.Context) (chat.Backend, io.Closer, error)
}
```

### Finding 5: the TUI renderer layer also copied shared helper infrastructure instead of isolating only the scopedjs-specific parts

Problem: The `scopedjs` renderer is not just inspired by the `scopeddb` renderer. It copied a large amount of general renderer scaffolding and then replaced the data formatting logic.

Where to look:

- `pinocchio/cmd/examples/scopedjs-tui-demo/renderers.go`
- `pinocchio/cmd/examples/scopeddb-tui-demo/renderers.go`

Example:

```go
func registerDemoRenderers(r *timeline.Registry) {
    r.RegisterModelFactory(base_renderers.NewLLMTextFactory())
    r.RegisterModelFactory(base_renderers.PlainFactory{})
    r.RegisterModelFactory(...)
    r.RegisterModelFactory(...)
    r.RegisterModelFactory(base_renderers.LogEventFactory{})
}
```

Why it matters: The result is a maintenance trap. Shell-level renderer behavior will drift between examples, and bug fixes in markdown rendering must be copied manually.

Cleanup sketch:

```text
pinocchio/pkg/ui/exampletimeline/
  markdown_entity.go
  tool_call_renderer.go
  tool_result_renderer.go
  style.go
```

Then example-specific files only implement:

```go
func buildEvalToolCallMarkdown(...) string
func formatEvalResultMarkdown(...) string
```

### Finding 6: examples and demo each carry their own fake modules instead of establishing a clear reusable test-double layer

Problem: The `webserver` and `obsidian` demo modules exist in both `geppetto/cmd/examples/scopedjs-dbserver/main.go` and `pinocchio/cmd/examples/scopedjs-tui-demo/environment.go`. They are not identical, but they are conceptually the same kind of fake capability layer.

Where to look:

- `geppetto/cmd/examples/scopedjs-dbserver/main.go:20-63`
- `pinocchio/cmd/examples/scopedjs-tui-demo/environment.go:40-96`

Why it matters: There is no obvious answer to which fake modules are canonical examples, which are intentionally demo-specific, and where a new example should get fake modules from.

Cleanup sketch:

```text
geppetto/pkg/inference/tools/scopedjs/scopedjstest/
  webserver.go
  obsidian.go
  fixtures.go
```

Then examples can either use those fakes directly or wrap them with demo-specific behavior.

## Design Decisions

### Decision 1: prefer honest semantics over feature-shaped enums

If a feature flag or enum suggests behavior that is not actually implemented, remove or rename it. In this review, `StateMode` is the clearest example.

### Decision 2: split static description from dynamic construction

Tool descriptions should not depend on whether a runtime is prebuilt or lazily constructed. If the model-facing capability docs are static, represent them statically. If they are scope-dependent, represent that intentionally.

### Decision 3: extract shared demo infrastructure before building a third example

Two similar examples are enough evidence that a reusable harness exists. The next example should not create a third copy of:

- profile resolution
- Watermill + Bubble Tea shell wiring
- markdown timeline entity scaffolding

## Alternatives Considered

### Alternative A: leave the package as-is and fix only the JS error bug

This would be the fastest path but the wrong cleanup choice. The misleading lifecycle semantics and lazy-description gap are higher-level API issues that will become harder to change once more code depends on them.

### Alternative B: perform a large package redesign immediately

This would likely overcorrect. The existing package shape is good enough to preserve. The real need is targeted cleanup, not a new abstraction stack.

### Alternative C: extract demo helpers first and leave scopedjs untouched

This would improve Pinocchio but leave the most important Geppetto API mismatches in place. That order is backwards.

## Implementation Plan

### Phase 1: clarify scopedjs lifecycle semantics

Goal: make package semantics honest.

Tasks:

1. decide whether `StateMode` survives
2. if it survives, implement real runtime strategies
3. otherwise replace it with clearer registration and runtime reuse language
4. update description generation and docs to match reality
5. add tests that call prebuilt tools multiple times and assert the intended state behavior

### Phase 2: separate description planning from runtime construction

Goal: make lazy and prebuilt tool descriptions equally informative.

Tasks:

1. add a manifest-only planning path
2. teach `NewLazyRegistrar(...)` to use it
3. add tests asserting lazy tool descriptions contain module, global, and helper details
4. update docs and examples

### Phase 3: fix option override semantics

Goal: make `EvalOptions` override behavior explicit and testable.

Tasks:

1. replace ambiguous boolean merging
2. add tests for overriding `CaptureConsole` in both directions
3. document override precedence clearly

### Phase 4: extract shared Pinocchio example shell

Goal: remove copy-pasted example scaffolding.

Tasks:

1. extract profile resolution helper
2. extract Bubble Tea, event router, and backend shell wiring
3. move shared status bar style helpers if needed
4. shrink `scopeddb` and `scopedjs` example `main.go` files to demo-specific wiring only

### Phase 5: extract shared renderer scaffolding

Goal: keep only content-formatting logic example-specific.

Tasks:

1. extract common timeline entity models and style helpers
2. leave only `build...Markdown(...)` and `format...Markdown(...)` in example files
3. update tests to target the shared helper package where appropriate

## Intern Guide: How the system fits together

This section is for a new intern who needs to work on the cleanup without having lived through the original implementation.

### Start with the package boundary

`scopedjs` is not a general JavaScript runtime framework. It is a Geppetto tool adapter for prepared JavaScript runtimes.

That means its job is narrower than it first appears:

- it does not decide what your application-specific modules do
- it does not decide what scope means
- it does not decide what a good `db` facade looks like

It does decide:

- how a host app describes a prepared runtime
- how that runtime is bootstrapped
- how the final tool input and output schema looks
- how description text is generated

If you keep that boundary in mind, cleanup decisions become easier.

### Key types and what they mean

#### `EnvironmentSpec`

This is the top-level host-side configuration object.

```go
type EnvironmentSpec[Scope any, Meta any] struct {
    RuntimeLabel string
    Tool         ToolDefinitionSpec
    DefaultEval  EvalOptions
    Configure    func(ctx context.Context, b *Builder, scope Scope) (Meta, error)
}
```

Think of it as:

```text
how the runtime should be described
+ how the runtime should be built
```

#### `Builder`

This is the mutable construction surface passed into `Configure(...)`.

You use it to register:

- modules
- globals
- bootstrap JavaScript
- helper docs

It is not the runtime itself. It is more like a build plan.

#### `BuildResult`

This is the output of `BuildRuntime(...)`.

```go
type BuildResult[Meta any] struct {
    Runtime  *gojengine.Runtime
    Meta     Meta
    Manifest EnvironmentManifest
    Cleanup  func() error
}
```

This object is important because it bundles both the live runtime and the description metadata. That combined role is part of why the lazy description problem exists today.

### How registration works today

There are two public registration paths:

#### `RegisterPrebuilt(...)`

You already built the runtime, and the tool calls reuse it.

#### `NewLazyRegistrar(...)`

The tool call resolves scope from context, builds a runtime, runs eval once, and cleans up.

Those are useful strategies. The mistake is pretending they are fully parameterized by `StateMode` when they are really hard-coded registration behaviors.

### How eval works today

The runtime execution path in `eval.go` does these things:

1. normalize options
2. inject `input` into a unique global variable
3. optionally replace `console`
4. wrap the model code in an async function
5. execute the code
6. if a promise is returned, poll it until it settles
7. return `EvalOutput`

Diagram:

```text
EvalInput
  code
  input
    |
    v
prepareEval(...)
  inject input var
  replace console
    |
    v
executeEval(...)
  RunString(async wrapper)
    |
    +--> plain value -> export -> Result
    |
    +--> promise -> waitForPromise(...)
                     |
                     +--> fulfilled -> Result
                     +--> rejected  -> Error
```

That path is the right place to reason about:

- promise rejection formatting
- console capture behavior
- truncation behavior
- timeout behavior

### How the Pinocchio demo uses the package

The demo is useful because it shows what an adopter actually has to write.

The flow is:

```text
fake fixture data
    |
    v
demoScope + demoMeta
    |
    v
demoEnvironmentSpec()
    |
    v
buildDemoRegistry(...)
    |
    v
toolloop backend + Bubble Tea chat model
    |
    v
timeline renderers for tool call / tool result
```

This is why the demo has real value beyond package tests. It proves:

- the tool contract is usable in an agent loop
- the model can read the description and write JS
- structured rendering is feasible

But it is also why duplicated shell logic should be cleaned up. Demos become templates for future work.

## Testing and Validation Strategy

### Package-level tests

Keep:

- build tests
- manifest tests
- eval tests
- registration tests

Add:

- prebuilt multi-call state behavior tests
- lazy description completeness tests
- option override tests for booleans

### Example-level tests

Keep direct smoke tests for:

- module access
- global access
- bootstrap helpers
- structured return values

### Demo-level tests

Keep focused renderer tests, but move generic renderer plumbing tests into shared helper packages once extracted.

## Open Questions

### Open question 1: should `scopedjs` support real per-session runtime caches?

Possible, but not required for the first cleanup. The more important need is to stop advertising semantics that do not yet exist.

### Open question 2: should rejection values preserve stack traces?

That came up separately during the Pinocchio demo work. It is related to eval behavior, but it is not required to solve the broader cleanup items in this ticket.

## References

### Geppetto

- `pkg/inference/tools/scopedjs/schema.go`
- `pkg/inference/tools/scopedjs/builder.go`
- `pkg/inference/tools/scopedjs/runtime.go`
- `pkg/inference/tools/scopedjs/eval.go`
- `pkg/inference/tools/scopedjs/description.go`
- `pkg/inference/tools/scopedjs/tool.go`
- `pkg/inference/tools/scopedjs/runtime_test.go`
- `pkg/inference/tools/scopedjs/tool_test.go`
- `cmd/examples/scopedjs-tool/main.go`
- `cmd/examples/scopedjs-dbserver/main.go`

### Pinocchio

- `cmd/examples/scopedjs-tui-demo/main.go`
- `cmd/examples/scopedjs-tui-demo/environment.go`
- `cmd/examples/scopedjs-tui-demo/renderers.go`
- `cmd/examples/scopeddb-tui-demo/main.go`
- `cmd/examples/scopeddb-tui-demo/renderers.go`
