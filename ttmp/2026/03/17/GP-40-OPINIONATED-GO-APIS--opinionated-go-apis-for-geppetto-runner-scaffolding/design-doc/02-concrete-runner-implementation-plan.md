---
Title: Concrete Runner Implementation Plan
Ticket: GP-40-OPINIONATED-GO-APIS
Status: active
Topics:
    - geppetto
    - go-api
    - architecture
    - go
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/session/session.go
      Note: Existing session lifecycle that the new runner will wrap
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: Existing low-level builder that the new runner will assemble internally
    - Path: geppetto/pkg/inference/middlewarecfg/resolver.go
      Note: Existing middleware-use resolution that the runner should reuse
    - Path: geppetto/pkg/inference/tools/definition.go
      Note: Existing function-to-tool conversion that the runner should expose ergonomically
    - Path: pinocchio/pkg/inference/runtime/engine.go
      Note: Current Pinocchio-only ToolRegistrar and engine helper to be replaced or mirrored
ExternalSources: []
Summary: Detailed, implementation-first guide for building the new Geppetto runner package step by step after the cleanup work that moved runtime resolution and policy out of Geppetto core.
LastUpdated: 2026-03-18T03:22:00-04:00
WhatFor: Use as the concrete build plan for implementing `pkg/inference/runner` in reviewable slices with clear responsibilities, test coverage, and migration targets.
WhenToUse: Use when actively implementing GP-40 or onboarding an intern to the concrete code changes needed for the new runner package.
---

# Concrete Runner Implementation Plan

## Executive Summary

This document translates the GP-40 architecture proposal into a concrete implementation sequence for a new public Geppetto package:

```text
geppetto/pkg/inference/runner
```

The most important architectural rule is simple:

- applications own runtime resolution and policy,
- the new runner owns assembly and execution.

That means the runner package should not accept profile patches, profile registries, request overrides, or other partial runtime fragments. It should accept a fully resolved runtime input and turn that into:

- a provider engine,
- a middleware-wrapped execution stack,
- an optional tool loop,
- a prepared session,
- and a simple sync or async run API.

This hard boundary is now realistic because the recent cleanup work removed the old mixed model from Geppetto core:

- request overrides are gone,
- `AllowedTools` is gone from Geppetto core,
- `StepSettingsPatch` is gone,
- `RuntimeKeyFallback` is gone,
- runtime attribution is now canonical-only.

So the right implementation plan is no longer "add a higher-level wrapper around profile-driven Geppetto runtime resolution." The right plan is "add a higher-level wrapper around Geppetto execution primitives that consumes resolved runtime input from the application."

## What The New Package Is

The new package is not:

- a replacement for `session`,
- a replacement for `toolloop`,
- a replacement for `enginebuilder`,
- or a new profile system.

The new package is:

- the batteries-included assembly layer above those primitives.

The package should give users one obvious place to start when they want to build:

- a simple CLI,
- a streaming server,
- a web chat backend,
- or a custom outer loop that still wants shared preparation.

## Current System Pieces You Must Understand First

Before implementing the runner package, an intern should understand five existing pieces.

### 1. `session.Session`

File:
- [session.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/session/session.go)

What it does:
- owns a stable `SessionID`
- stores append-only turn history
- ensures only one inference is active at a time
- appends prompt turns through `AppendNewTurnFromUserPrompt(...)`
- starts inference through `StartInference(...)`

Why it matters:
- the new runner should build on `Session`, not hide or replace it.
- `Prepare(...)` should return a prepared `Session` so advanced callers can still drive custom outer loops.

### 2. `enginebuilder.Builder`

Files:
- [builder.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/enginebuilder/builder.go)
- [options.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/enginebuilder/options.go)

What it does:
- wraps a base engine with middleware
- attaches event sinks and snapshot hooks
- chooses single-pass or tool-loop execution based on whether a registry exists
- persists the final turn if a persister is configured

Why it matters:
- the runner package should assemble an `enginebuilder.Builder` internally.
- it should not invent a parallel execution stack.

### 3. `toolloop.Loop` and configs

Files:
- [loop.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/loop.go)
- [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/config.go)
- [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go)

What it does:
- runs iterative tool-calling inference
- executes pending tools
- appends tool results
- repeats until done or max iterations is reached

Why it matters:
- the runner package should expose loop and tool config defaults, but still rely on the existing loop implementation.

### 4. `middlewarecfg`

Files:
- [definition.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/middlewarecfg/definition.go)
- [resolver.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/middlewarecfg/resolver.go)
- [chain.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/middlewarecfg/chain.go)

What it does:
- lets applications describe middleware by configuration instead of only by direct Go construction
- resolves middleware-use payloads against definitions
- builds real middleware instances from app-owned dependencies

Why it matters:
- the runner package should reuse this instead of adding a new middleware configuration system.

### 5. tools and registries

Files:
- [definition.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/definition.go)
- [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/registry.go)

What it does:
- converts Go functions into tools
- stores tool definitions in registries
- provides cloning and merging

Why it matters:
- the runner should make function tools and registrars easier to use.
- it should still produce ordinary `tools.ToolRegistry` values.

## The New Public API Boundary

This is the recommended public boundary for the first implementation.

```go
type ToolRegistrar func(ctx context.Context, reg tools.ToolRegistry) error

type Runtime struct {
    StepSettings *settings.StepSettings
    SystemPrompt string

    MiddlewareUses []profiles.MiddlewareUse
    Middlewares    []middleware.Middleware

    ToolNames      []string
    ToolRegistrars []ToolRegistrar

    RuntimeKey         string
    RuntimeFingerprint string
    ProfileVersion     uint64
}

type StartRequest struct {
    SessionID string
    Prompt    string
    SeedTurn  *turns.Turn
    Runtime   Runtime

    EventSinks   []events.EventSink
    SnapshotHook toolloop.SnapshotHook
    Persister    enginebuilder.TurnPersister
}

type PreparedRun struct {
    Runtime  Runtime
    Engine   engine.Engine
    Registry tools.ToolRegistry
    Session  *session.Session
    Turn     *turns.Turn
}
```

Important notes:

- `Runtime` is resolved input, not a request to resolve something later.
- `RuntimeKey`, `RuntimeFingerprint`, and `ProfileVersion` are optional app-owned metadata. The runner should carry them, not derive them.
- `EventSinks`, `SnapshotHook`, and `Persister` belong in the request because streaming and persistence are often per-run concerns, not only runner-global concerns.

## Recommended Package Layout

Start with this file structure:

```text
geppetto/pkg/inference/runner/
  types.go
  options.go
  tools.go
  middleware.go
  prepare.go
  run.go
  errors.go
  runner_test.go
```

Recommended responsibilities:

- `types.go`
  - public types
  - interfaces
  - request/result structs
- `options.go`
  - runner construction
  - default configs
  - `Option` functions
- `tools.go`
  - `ToolRegistrar`
  - `WithFuncTool(...)`
  - registry construction and filtering
- `middleware.go`
  - middleware resolution/building
  - system-prompt and default middleware helpers
- `prepare.go`
  - `Prepare(...)`
  - session bootstrap
  - builder assembly
- `run.go`
  - `Start(...)`
  - `Run(...)`
- `errors.go`
  - clear package-specific errors
- `runner_test.go`
  - end-to-end behavior tests

## Step-By-Step Implementation Sequence

This sequence is the most important part of the guide. Follow it in order.

### Phase 1: Create the package skeleton

Create the package directory and initial files.

Files:
- `geppetto/pkg/inference/runner/types.go`
- `geppetto/pkg/inference/runner/options.go`
- `geppetto/pkg/inference/runner/errors.go`

Implement first:
- `Runner`
- `Option`
- `Runtime`
- `StartRequest`
- `PreparedRun`
- package-level errors

Recommended errors:

```go
var (
    ErrRunnerNil           = errors.New("runner is nil")
    ErrRuntimeStepSettings = errors.New("runtime step settings are nil")
    ErrRuntimeComposerNil  = errors.New("runtime composer is nil")
    ErrPromptAndSeedEmpty  = errors.New("prompt and seed turn are both empty")
)
```

Design rule:
- do not add a composer yet unless a direct runtime-only constructor feels obviously incomplete.

Why this phase exists:
- it freezes the public boundary before we write behavior.
- it lets tests and later slices compile against stable types.

### Phase 2: Implement tool registration and filtering

Files:
- `geppetto/pkg/inference/runner/tools.go`

Implement:
- `ToolRegistrar`
- `FuncTool(...) ToolRegistrar`
- `MustFuncTool(...) ToolRegistrar`
- `buildRegistry(...)`
- `filterRegistry(...)`

Pseudocode:

```go
func buildRegistry(ctx context.Context, regs []ToolRegistrar, names []string) (tools.ToolRegistry, error) {
    reg := tools.NewInMemoryToolRegistry()
    for _, registrar := range regs {
        if registrar == nil {
            continue
        }
        if err := registrar(ctx, reg); err != nil {
            return nil, err
        }
    }
    if len(names) == 0 {
        return reg, nil
    }
    return filterRegistry(reg, names)
}
```

Filtering rules:
- if `ToolNames` is empty, keep everything
- if `ToolNames` is non-empty, keep only those tools
- fail clearly if a requested tool name is missing

Why this phase exists:
- tool registration is one of the main pieces of repeated boilerplate today.
- it is independent from session/bootstrap logic and is easy to test first.

### Phase 3: Implement middleware resolution and default middleware policy

Files:
- `geppetto/pkg/inference/runner/middleware.go`

Implement:
- `resolveMiddlewares(...)`
- optional `WithMiddlewareDefinitions(...)`
- optional `WithMiddlewareBuildDeps(...)`

Recommended policy:

1. start with explicit `Runtime.Middlewares`
2. if empty and `Runtime.MiddlewareUses` is non-empty, resolve them through `middlewarecfg`
3. prepend `middleware.NewToolResultReorderMiddleware()`
4. append `middleware.NewSystemPromptMiddleware(...)` if `SystemPrompt` is non-empty

Diagram:

```text
Runtime.Middlewares provided?
  yes -> use them
  no  -> Runtime.MiddlewareUses provided?
           yes -> resolve with middlewarecfg
           no  -> none

then:
  prepend reorder middleware
  append system-prompt middleware when configured
```

Why this phase exists:
- middleware policy is one of the biggest places where app code diverges today.
- centralizing it early makes later example migration much cleaner.

### Phase 4: Implement engine construction from final settings

Files:
- `geppetto/pkg/inference/runner/prepare.go`

Implement:
- `buildEngine(...)`

Use:
- `factory.NewEngineFromStepSettings(...)`

Do not:
- reintroduce Pinocchio’s `BuildEngineFromSettingsWithMiddlewares(...)` helper into Geppetto unchanged
- create a special profile-driven path

The runner should:
- create the base engine from final `StepSettings`
- keep middleware separate until builder assembly

Why:
- `enginebuilder.Builder` already applies middleware.
- duplicating the Pinocchio helper would blur the boundary again.

### Phase 5: Implement `Prepare(...)`

Files:
- `geppetto/pkg/inference/runner/prepare.go`

This is the most important function in the package.

Responsibilities:
- validate runtime input
- build engine
- resolve middlewares
- build/filter registry
- create session
- append or normalize seed turn
- append prompt turn
- assemble `enginebuilder.Builder`
- return `PreparedRun`

Recommended pseudocode:

```go
func (r *Runner) Prepare(ctx context.Context, req StartRequest) (*PreparedRun, error) {
    if req.Runtime.StepSettings == nil {
        return nil, ErrRuntimeStepSettings
    }

    eng, err := buildEngine(ctx, req.Runtime.StepSettings)
    if err != nil {
        return nil, err
    }

    mws, err := r.resolveMiddlewares(ctx, req.Runtime)
    if err != nil {
        return nil, err
    }

    reg, err := buildRegistry(ctx, req.Runtime.ToolRegistrars, req.Runtime.ToolNames)
    if err != nil {
        return nil, err
    }

    sess := session.NewSession()
    if req.SessionID != "" {
        sess.SessionID = req.SessionID
    }

    if req.SeedTurn != nil {
        sess.Append(req.SeedTurn.Clone())
    }

    turn, err := sess.AppendNewTurnFromUserPrompt(req.Prompt)
    if err != nil {
        return nil, err
    }

    builder := enginebuilder.New(
        enginebuilder.WithBase(eng),
        enginebuilder.WithMiddlewares(mws...),
        enginebuilder.WithToolRegistry(reg),
        enginebuilder.WithLoopConfig(r.loopCfg),
        enginebuilder.WithToolConfig(r.toolCfg),
        enginebuilder.WithEventSinks(mergeSinks(r.eventSinks, req.EventSinks)...),
        enginebuilder.WithSnapshotHook(firstNonNil(req.SnapshotHook, r.snapshotHook)),
        enginebuilder.WithPersister(firstPersister(req.Persister, r.persister)),
    )
    sess.Builder = builder

    return &PreparedRun{
        Runtime:  req.Runtime,
        Engine:   eng,
        Registry: reg,
        Session:  sess,
        Turn:     turn,
    }, nil
}
```

Critical implementation note:
- `Prepare(...)` should be usable without ever calling `Start(...)`.
- that is what makes custom outer loops possible.

### Phase 6: Implement `Start(...)` and `Run(...)`

Files:
- `geppetto/pkg/inference/runner/run.go`

Implement:
- `Start(ctx, req) (*PreparedRun, *session.ExecutionHandle, error)`
- `Run(ctx, req) (*PreparedRun, *turns.Turn, error)`

Pseudocode:

```go
func (r *Runner) Start(ctx context.Context, req StartRequest) (*PreparedRun, *session.ExecutionHandle, error) {
    prep, err := r.Prepare(ctx, req)
    if err != nil {
        return nil, nil, err
    }
    handle, err := prep.Session.StartInference(ctx)
    if err != nil {
        return prep, nil, err
    }
    return prep, handle, nil
}

func (r *Runner) Run(ctx context.Context, req StartRequest) (*PreparedRun, *turns.Turn, error) {
    prep, handle, err := r.Start(ctx, req)
    if err != nil {
        return prep, nil, err
    }
    turn, err := handle.Wait()
    return prep, turn, err
}
```

Why:
- `Run(...)` is the small path
- `Start(...)` is the streaming/event-driven path
- `Prepare(...)` is the custom-outer-loop path

Those three functions together are the core user-facing package story.

### Phase 7: Add a composer interface only if it still earns its keep

After the direct runtime path exists and tests pass, decide whether to add:

```go
type Composer interface {
    Compose(ctx context.Context, req RuntimeRequest) (Runtime, error)
}
```

Recommendation:
- do not start with this
- add it only after the direct runtime path lands

Reason:
- it is easier to design the right abstraction once the direct path is real.
- starting with a composer risks recreating the old “Geppetto resolves app policy” shape too early.

### Phase 8: Migrate first-party examples

Good first migrations:
- [simple-inference/main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/simple-inference/main.go)
- [middleware-inference/main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/middleware-inference/main.go)
- [openai-tools/main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/openai-tools/main.go)

Migration goals:
- reduce boilerplate
- keep behavior equivalent
- prove the public API is actually better to use

### Phase 9: Add event-driven example coverage

The package must prove it still supports:
- SSE
- WebSockets
- TUI event loops
- background workers forwarding events elsewhere

That means:
- one unit or integration test should use `Start(...)`
- one example should show an event sink

This is important because the package should not read as a blocking-only convenience layer.

## Tests You Must Add

### Unit tests

Add tests for:

- `FuncTool(...)` registration
- registry filtering by `ToolNames`
- middleware resolution from direct middleware
- middleware resolution from middleware uses
- system prompt middleware insertion
- reorder middleware insertion
- `Prepare(...)` with no tools
- `Prepare(...)` with tools
- `Start(...)` async path
- `Run(...)` blocking path
- event sink propagation
- per-run sink overrides
- nil `StepSettings` failure

### Example-shaped tests

Add at least one test for each major use shape:

1. Tiny CLI shape
2. Event-driven `Start(...)` shape
3. Custom outer loop with `Prepare(...)`

### Migration tests

When examples are migrated, compare:
- old output/behavior
- new output/behavior

Do not assume the new package is correct just because it compiles.

## Recommended Commit Sequence

Use small commits in this order:

1. `add gp-40 concrete implementation docs`
2. `add runner package types and options`
3. `add runner tool registration and filtering`
4. `add runner middleware resolution`
5. `add runner prepare start and run`
6. `add runner tests`
7. `migrate geppetto examples to runner`
8. `record gp-40 implementation progress`

That sequence keeps the implementation reviewable and matches the task board.

## Risks And How To Avoid Them

### Risk 1: the runner becomes profile-aware again

Avoid by:
- accepting only resolved runtime input
- keeping composer optional
- keeping profile registries and profile selection outside the package

### Risk 2: middleware policy becomes surprising

Avoid by:
- making default middleware insertion explicit in docs and tests
- deciding once whether reorder middleware is always-on

### Risk 3: event-driven support becomes second-class

Avoid by:
- adding `Start(...)` from the first implementation slice
- exposing per-run sinks
- adding event-driven examples/tests immediately

### Risk 4: example migration reveals a missing option too late

Avoid by:
- migrating first-party examples before downstream apps
- letting those examples drive the missing option surface

## Open Questions For Implementation

1. Should `StartRequest` carry per-run loop/tool config overrides, or should that wait until after the first examples are migrated?
2. Should reorder middleware be always-on in the runner, matching current Pinocchio engine composition?
3. Should `Runner` expose helper constructors like `NewFromParsedValues(...)`, or should that stay out of the first cut?
4. Should the eventual composer interface live in the same package, or in a small subpackage like `runnercompose`?

## Final Recommendation

Start simple and direct:

1. Create `pkg/inference/runner`
2. Accept resolved runtime input only
3. Implement `Prepare`, `Start`, and `Run`
4. Reuse `enginebuilder`, `toolloop`, `middlewarecfg`, and `tools`
5. Migrate Geppetto examples
6. Add a composer only after the direct path proves itself

That is the cleanest path from analysis to a usable public API.
