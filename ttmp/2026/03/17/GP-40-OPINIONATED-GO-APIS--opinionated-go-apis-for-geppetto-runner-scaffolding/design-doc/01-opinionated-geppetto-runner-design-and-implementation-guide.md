---
Title: Opinionated Geppetto Runner Design and Implementation Guide
Ticket: GP-40-OPINIONATED-GO-APIS
Status: active
Topics:
    - geppetto
    - pinocchio
    - go-api
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: 2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go
      Note: CoinVault session/bootstrap duplication motivating the new API
    - Path: geppetto/pkg/inference/middlewarecfg/resolver.go
      Note: Profile-driven middleware resolution already available in Geppetto
    - Path: geppetto/pkg/inference/session/session.go
      Note: Session lifecycle and turn history invariants
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: Current low-level composition hub that the proposed runner should wrap
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Tool loop orchestration and tool execution behavior
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Pinocchio runtime composition that overlaps with the proposed Geppetto layer
    - Path: temporal-relationships/internal/extractor/gorunner/loop.go
      Note: Advanced custom outer-loop use case that still needs shared preparation
ExternalSources: []
Summary: Detailed analysis and design proposal for a new opinionated Go runner layer on top of Geppetto's existing session, toolloop, middleware, and tools primitives, updated after the hard cuts that removed profile patches, runtime-key fallback, and other mixed runtime configuration paths from Geppetto core.
LastUpdated: 2026-03-18T03:12:00-04:00
WhatFor: Evidence-backed design and migration guide for a new opinionated Go runner layer on top of Geppetto, with final runtime resolution, runtime identity, and registry filtering treated as app-owned preparation rather than Geppetto core concerns.
WhenToUse: Use when designing or implementing a simpler public Go API for inference sessions, tool loops, middleware resolution, and app-owned resolved runtime composition on top of Geppetto core.
---


# Opinionated Geppetto Runner Design and Implementation Guide

## Executive Summary

Geppetto already has solid low-level primitives for inference: `session.Session` owns turn history and active-run lifecycle, `toolloop/enginebuilder.Builder` wires engines, middlewares, tools, sinks, snapshots, and persistence, and `toolloop.Loop` runs iterative tool execution. Those primitives are good building blocks, but they are still too low-level for the common "make me a useful CLI or chat loop in a few lines" use case.

Observed evidence:

- `session.Session` is intentionally generic and delegates all policy to an `EngineBuilder` in `geppetto/pkg/inference/session/session.go:21-35` and `geppetto/pkg/inference/session/builder.go:9-20`.
- `enginebuilder.Builder` carries many concerns at once: base engine, middlewares, tool registry, loop config, tool config, tool executor, event sinks, snapshot hooks, step control, and persistence in `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:31-74`.
- The builder switches between single-pass inference and a tool loop depending on whether a registry is present in `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:190-218`.
- The same hand-wiring pattern is repeated in Geppetto examples, in Pinocchio wrappers, and again in downstream applications such as CoinVault, CozoDB Editor, and Temporal Relationships.

The recommendation is to add a new opinionated Go runner layer above `session` and `enginebuilder`, not instead of them. The new layer should preserve the current low-level APIs while offering:

1. A very small default surface for static CLI tools and simple chat loops.
2. A slightly larger advanced surface for profile-driven runtimes, registered tools, middleware definitions, event sinks, snapshots, and persistence.
3. Shared abstractions that currently live in Pinocchio but are general enough to belong in Geppetto, especially tool registrars and runtime composition contracts.

The design below recommends a new package, shown with the placeholder alias `oprunner`, that can collapse repetitive app code into a few lines while still exposing escape hatches for complex applications.

## Problem Statement And Scope

The requested outcome is an opinionated runner for Geppetto so that people can scaffold powerful CLI tools that run inference tool loops with registered tools and middlewares "in a couple of lines of code."

This design is in scope:

- A Go-first runner API for common CLI and chat-style inference flows.
- Registry construction from Go functions and registrar callbacks.
- Tool-loop defaults that are safe and sensible but overridable.
- Middleware composition from static middleware and profile-driven middleware definitions.
- Session bootstrap, prompt append, run start, wait, sink propagation, snapshots, persistence, and step mode.
- Migration guidance for current Geppetto, Pinocchio, and downstream users.

This design is explicitly out of scope:

- Replacing `session.Session`, `toolloop.Loop`, `enginebuilder.Builder`, or `tools.ToolRegistry`.
- Hiding the Turn model. Advanced users should still be able to work directly with `turns.Turn`.
- Replacing application-owned persistence schemas or app-specific outer loops.
- Creating a new profile format inside Geppetto core. Profile selection and profile-to-runtime resolution are now application concerns, not runner-core concerns.

## Current-State Architecture

### 1. Core Geppetto stack

The current runtime stack is layered and reasonably clean:

```text
application code
  -> session.Session
      -> EngineBuilder.Build(...)
          -> enginebuilder.Builder
              -> middleware chain around base engine
              -> optional toolloop.Loop
                  -> tools.ToolRegistry + ToolExecutor
                      -> provider engine
```

The important detail is that Geppetto already has the right primitives. The missing piece is a higher-level public API that assembles them consistently.

### 2. Session lifecycle

`session.Session` is the canonical long-lived interaction object:

- Stable `SessionID`, turn history, and single-active-inference invariant are defined in `geppetto/pkg/inference/session/session.go:21-35`.
- `AppendNewTurnFromUserPrompt` clones the previous turn, drops the old turn ID, appends new user blocks, and adds the seed turn to history in `geppetto/pkg/inference/session/session.go:44-94`.
- `StartInference` stamps session and inference metadata, builds a blocking runner from the configured builder, runs asynchronously, and merges the result back into the latest turn in `geppetto/pkg/inference/session/session.go:180-273`.

This is a good low-level API, but it is not a "batteries included" user-facing runner API. The caller still has to construct the builder, the seed turn, the registry, and the wiring around each run.

### 3. Engine builder

`enginebuilder.Builder` is Geppetto's composition hub:

- The builder fields show all the moving pieces applications need to assemble in `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:31-74`.
- `Build` normalizes default loop and tool configs and returns a runner bound to one session ID in `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:78-112`.
- `RunInference` attaches event sinks and snapshot hooks to context, stamps missing turn IDs and metadata, and then chooses single-pass or tool-loop execution in `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:158-240`.
- The option helpers in `geppetto/pkg/inference/toolloop/enginebuilder/options.go:13-89` make this more ergonomic, but the caller still has to know which options matter for a given use case.

This builder is intentionally low-level. That is the right role for it. It should remain the flexible assembly primitive beneath the new opinionated runner.

### 4. Tool loop behavior

The tool loop is clear and serializable:

- `toolloop.Loop` owns the engine, registry, configs, executor, step controller, pause timeout, and snapshot hook in `geppetto/pkg/inference/toolloop/loop.go:20-32`.
- `RunLoop` injects the registry into context, stores serializable tool configuration and tool definition snapshots on `Turn.Data`, runs iterative inference, extracts pending tool calls, executes them, appends tool results, and repeats until no tool calls remain or max iterations is reached in `geppetto/pkg/inference/toolloop/loop.go:92-173`.
- Tool execution uses the context registry and a configurable executor in `geppetto/pkg/inference/toolloop/loop.go:273-310`.
- Step-mode pause/continue behavior is owned by `StepController` in `geppetto/pkg/inference/toolloop/step_controller.go:47-207`.

This is exactly the sort of logic an opinionated runner should reuse instead of re-implementing.

### 5. Tool subsystem

The tool subsystem is already suitable for a better public surface:

- `tools.NewToolFromFunc` turns Go functions into tool definitions with inferred JSON Schema in `geppetto/pkg/inference/tools/definition.go:34-95`.
- `ToolRegistry` and `InMemoryToolRegistry` provide registration, listing, merging, and cloning in `geppetto/pkg/inference/tools/registry.go:8-142`.
- `DefaultToolExecutor` and `BaseToolExecutor` handle call execution, retries, event publishing, masking, and concurrency in `geppetto/pkg/inference/tools/executor.go:7-32` and `geppetto/pkg/inference/tools/base_executor.go:30-269`.
- `ToolConfig` still captures execution policy knobs the higher-level API should surface, such as tool choice, parallelism, retries, timeouts, and error behavior in `geppetto/pkg/inference/tools/config.go:5-129`. Tool selection itself should now be treated as app-owned registry filtering rather than a core Geppetto allowlist.

The main missing piece is not capability. It is a better "tool registrar" abstraction for building registries from app-owned catalog entries and lazy dataset-backed tools.

### 6. Middleware subsystem

Middleware is also already structured correctly:

- The core middleware contract is small and composable in `geppetto/pkg/inference/middleware/middleware.go:8-23`.
- Important built-ins already exist: system prompt middleware in `geppetto/pkg/inference/middleware/systemprompt_middleware.go:11-95`, logging in `geppetto/pkg/inference/middleware/logging_middleware.go:11-87`, and tool-result adjacency repair in `geppetto/pkg/inference/middleware/reorder_tool_results_middleware.go:10-139`.
- Schema-first middleware composition is available through `middlewarecfg.Definition`, `Resolver`, and `BuildChain` in `geppetto/pkg/inference/middlewarecfg/definition.go:9-40`, `geppetto/pkg/inference/middlewarecfg/resolver.go:15-204`, and `geppetto/pkg/inference/middlewarecfg/chain.go:12-76`.

This is significant because the opinionated runner does not need a new middleware system. It needs to own the integration point that turns runtime middleware uses into concrete middleware instances.

### 7. The runner boundary is now cleaner than it was during the original GP-40 writeup

The most important architectural change since this ticket started is that Geppetto no longer needs to own profile-driven partial runtime resolution:

- request overrides are gone,
- `AllowedTools` is gone from Geppetto core,
- `StepSettingsPatch` is gone,
- `RuntimeKeyFallback` is gone,
- runtime attribution is now canonical-only.

That changes the recommended GP-40 boundary materially.

The runner should no longer accept profile-like partial runtime fragments from Geppetto core. Instead, the app should hand Geppetto one fully resolved runtime input:

- final `*settings.StepSettings`,
- final system prompt,
- final middleware uses or concrete middleware instances,
- final tool registrars or already filtered registry inputs,
- app-owned runtime identity metadata if the app cares about it.

In other words:

```text
app config / profile selection / caching
  -> resolved runtime input
  -> Geppetto opinionated runner
      -> session + enginebuilder + toolloop
```

not:

```text
app
  -> Geppetto profiles
  -> patch/override/runtime resolution
  -> runner
```

### 8. The JS API already exposes a higher-level builder

There is an important asymmetry in Geppetto today:

- The JS module already offers `createSession`, `runInference`, `withTools`, `withToolLoop`, `withToolHooks`, `withEventSink`, and `withPersister` in `geppetto/pkg/js/modules/geppetto/api_sessions.go:18-245` and `geppetto/pkg/js/modules/geppetto/api_builder_options.go:14-259`.
- The JS tool-loop settings map directly to `LoopConfig` and `ToolConfig` in `geppetto/pkg/js/modules/geppetto/api_builder_options.go:191-259`.

Inference: Geppetto already accepts that users want a higher-level composition API. Go currently lacks an equally ergonomic surface.

## Evidence From Real Usage

### Geppetto examples repeat the same manual setup

The examples are straightforward, but they also show the current ergonomics problem:

- `geppetto/cmd/examples/simple-inference/main.go:111-165` builds an engine, creates middleware, creates a session, sets `sess.Builder`, appends a seed turn, starts inference, waits, and prints.
- `geppetto/cmd/examples/middleware-inference/main.go:128-244` repeats that pattern and manually toggles the tool registry, loop config, and tool config.
- `geppetto/cmd/examples/openai-tools/main.go:316-398` manually creates tools, registers them, chooses tool config, creates a session, sets builder options, appends a turn, and waits.

The documentation also teaches the low-level path directly in `geppetto/pkg/doc/topics/07-tools.md:108-174` and `geppetto/pkg/doc/topics/09-middlewares.md:65-205`.

### Pinocchio has started building app-facing wrappers already

Observed wrappers in Pinocchio:

- `pinocchio/pkg/ui/backends/toolloop/backend.go:23-89` is effectively a mini opinionated runner for Bubble Tea chats.
- `pinocchio/pkg/ui/profileswitch/backend.go:19-239` builds profile-switched sessions by rewriting the builder around a session.
- `pinocchio/pkg/inference/runtime/engine.go:16-46` defines `ToolRegistrar` and a helper that builds an engine from settings plus middlewares, but it actually returns a builder-built runner as an `engine.Engine`.
- `pinocchio/cmd/web-chat/runtime_composer.go:17-177` resolves profile middleware config, composes middlewares, builds an engine, computes runtime fingerprints, and passes app-owned tool selection metadata downstream.

Inference: Pinocchio is already compensating for the lack of a shared Go runner layer in Geppetto.

### CoinVault duplicates session preparation and registry filtering

CoinVault's configurable runner is one of the clearest examples of the missing abstraction:

- The runner prepares runtime composition separately from execution in `2026-03-16--gec-rag/internal/webchat/configurable_loop_runner.go:22-84`.
- Session preparation resolves runtime, builds an allowed-tools registry, loads or creates the seed turn, appends the user prompt, publishes a user event, and finally creates an `enginebuilder.Builder` with sinks, snapshot hook, step control, and persistence in `2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go:89-156`.
- Tool catalogs are app-owned and filtered by allowed tool names in `2026-03-16--gec-rag/internal/webchat/tool_catalog.go:14-154`.
- Simple function tools are turned into registrars by app-local helpers in `2026-03-16--gec-rag/internal/webchat/tools.go:45-52`.

This is exactly the kind of composition the opinionated runner should centralize.

### CozoDB Editor uses Geppetto directly for streaming inference

`2026-03-14--cozodb-editor/backend/pkg/hints/engine.go:91-158` shows a simpler but still repetitive pattern:

- create seed turn,
- assemble event sinks and filtering sink,
- create session,
- create builder with base engine, event sinks, and step controller,
- append turn,
- start and wait,
- post-process the final turn into a response.

This application does not need tool registration, but it still wants the same session and sink wiring. The new API should support this lighter path too.

### Temporal Relationships needs the same pieces plus a custom outer loop

Temporal Relationships has the most specialized control flow:

- `temporal-relationships/internal/extractor/gorunner/loop.go:46-108` creates a session, maybe builds a scoped tool registry, configures an `enginebuilder.Builder`, and then runs an outer loop over repeated prompts.
- It persists turn snapshots before and after each inference in `temporal-relationships/internal/extractor/gorunner/loop.go:156-174`.
- It uses lazy scoped database tool registrars built on Geppetto's `scopeddb.NewLazyRegistrar` in `temporal-relationships/internal/extractor/entityhistory/tool.go:19-45`, `.../transcripthistory/tool.go:19-45`, `.../runturnhistory/tool.go:12-38`, and `geppetto/pkg/inference/tools/scopeddb/tool.go:37-79`.
- The run-chat API layer also composes runtime settings and app-owned tool filtering inputs in `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go:160-225`.

This app should probably keep its custom outer loop, but it should still be able to reuse the same runtime composition and session/bootstrap logic from the new opinionated runner.

## Gap Analysis

The main gaps are architectural, not algorithmic.

### Gap 1: repeated manual assembly

Every app manually answers the same questions:

- How do I create the base engine?
- Which middlewares do I always apply?
- How do I turn profile middleware uses into concrete middleware?
- How do I build a registry from a catalog of registrars?
- How do I filter tools by allowed tool list?
- How do I seed the first turn versus append a follow-up prompt?
- How do I attach sinks, snapshot hooks, persisters, and step control?
- How do I expose a sync "run once" path and an async "start and wait" path?

The answers are currently scattered across examples and app code.

### Gap 2: shared abstractions live in the wrong layer

`ToolRegistrar` currently lives in Pinocchio, not Geppetto, in `pinocchio/pkg/inference/runtime/engine.go:16-17`. CoinVault uses that type in its own tool catalog in `2026-03-16--gec-rag/internal/webchat/tool_catalog.go:14-18`. That is a sign that the abstraction is generally useful and should move down into Geppetto.

### Gap 3: runtime composition is split awkwardly

Geppetto owns the execution stack and `middlewarecfg`, but app-facing runtime composition contracts are still scattered across Pinocchio and downstream apps. The new runner should expose a small resolved-runtime boundary on the Geppetto side so downstream code does not need Pinocchio types just to express "here is the engine config I already resolved."

### Gap 4: no standard bootstrap story for app-owned tool filtering

After the removal of core `AllowedTools` enforcement, Geppetto now correctly trusts the tool registry it is given. The remaining duplication is that multiple applications still rebuild the same app-owned filtering and registry-construction steps themselves before handing a registry to Geppetto, for example `ToolCatalog.BuildRegistry` in `2026-03-16--gec-rag/internal/webchat/tool_catalog.go:132-154`.

That means the opinionated runner should not resurrect a core allowlist concept. It should instead make registry filtering a first-class app-owned preparation step that happens before the registry reaches Geppetto core.

### Gap 5: the current Go surface is less ergonomic than the JS surface

The JS module already lets callers build sessions and configure tool loops in a small number of calls. Go users still copy boilerplate from examples. That asymmetry is avoidable.

## Design Goals

1. Keep the existing low-level primitives intact.
2. Make the default path easy enough for small CLIs and prototypes.
3. Make the advanced path strong enough for web chat, profile-driven runtimes, lazy registrars, snapshots, persistence, and step mode.
4. Reuse `settings.StepSettings`, `middlewarecfg`, `toolloop`, and `tools` instead of inventing new parallel systems.
5. Keep application-specific outer loops possible.
6. Make migration incremental.

## Non-Goals

1. Do not hide `turns.Turn` from advanced users.
2. Do not force every caller into profile-driven composition.
3. Do not move all app logic into Geppetto.
4. Do not make `enginebuilder.Builder` itself responsible for high-level app policy.

## Design Options

### Option A: improve documentation only

Description:

- Keep the current API.
- Add more examples and playbooks.

Advantages:

- Very low implementation cost.
- No new public surface area.

Problems:

- Does not remove boilerplate.
- Does not centralize repeated patterns already duplicated across multiple apps.
- Does not solve the mismatch between low-level primitives and common user goals.

Conclusion:

- Not sufficient.

### Option B: add a few helper functions on top of `enginebuilder`

Description:

- Add helpers such as `RunPrompt`, `BuildRegistry`, or `RunWithTools`.

Advantages:

- Small change.
- Easy to explain.

Problems:

- Too small for profile-driven middleware resolution, runtime composition, snapshots, persistence, and async start/wait flows.
- Tends to accumulate ad hoc helpers instead of a coherent public layer.

Conclusion:

- Better than today, but still too weak for the downstream evidence set.

### Option C: add a dedicated opinionated runner layer above `session` and `enginebuilder`

Description:

- Introduce a new Geppetto package that composes engines, middlewares, tool registrars, resolved runtime inputs, sessions, and lifecycle hooks into a standard app-facing surface.

Advantages:

- Matches the actual usage patterns in Pinocchio and downstream apps.
- Preserves low-level escape hatches.
- Makes it possible to move generally useful abstractions out of Pinocchio and into Geppetto.
- Gives both simple and advanced callers a single place to start.

Problems:

- More API design work.
- Needs careful boundary discipline to avoid swallowing app-specific logic.

Conclusion:

- Recommended.

### Option D: keep the opinionated layer in Pinocchio

Description:

- Treat Geppetto as low-level only and let Pinocchio remain the "app framework."

Advantages:

- Smaller Geppetto surface.

Problems:

- Downstream non-Pinocchio apps still need the same abstractions.
- Shared runtime concepts already depend on Geppetto types.
- The requested feature is specifically "an opinionated runner for Geppetto."

Conclusion:

- Not recommended as the primary answer.

## Recommended Architecture

### Package boundary

Recommended new package family:

```text
geppetto/pkg/inference/opinionated/
  runner.go
  runtime.go
  registrars.go
  middleware.go
  prepare.go
  result.go
```

I am using `opinionated` as a placeholder name. The exact path is a naming decision, not a design dependency. The important boundary is:

- `session`, `toolloop`, `tools`, and `middlewarecfg` remain foundational packages.
- `opinionated` becomes the batteries-included assembly layer.

### Core types

The following API sketch is the recommended shape.

```go
type ToolRegistrar func(ctx context.Context, reg tools.ToolRegistry) error

type RuntimeRequest struct {
    RuntimeKey         string
    RuntimeFingerprint string
    ProfileVersion     uint64

    StepSettings *settings.StepSettings
    SystemPrompt string

    // One of these two middleware inputs may be used.
    MiddlewareUses []middlewarecfg.Use
    Middlewares    []middleware.Middleware

    ToolNames      []string
    ToolRegistrars []ToolRegistrar
}

type ComposedRuntime struct {
    Engine             engine.Engine
    RuntimeKey         string
    RuntimeFingerprint string

    SeedSystemPrompt string
    Middlewares      []middleware.Middleware
    ToolNames        []string
    ToolRegistrars   []ToolRegistrar

    EventSinks       []events.EventSink
    SnapshotHook     toolloop.SnapshotHook
    Persister        enginebuilder.TurnPersister
    StepController   *toolloop.StepController
    StepPauseTimeout time.Duration

    LoopConfig toolloop.LoopConfig
    ToolConfig tools.ToolConfig
}

type RuntimeComposer interface {
    Compose(ctx context.Context, req RuntimeRequest) (ComposedRuntime, error)
}

type StartRequest struct {
    SessionID string
    Prompt    string
    SeedTurn  *turns.Turn
    Runtime   RuntimeRequest
}

type PreparedRun struct {
    Runtime  ComposedRuntime
    Registry tools.ToolRegistry
    Session  *session.Session
    SeedTurn *turns.Turn
    TurnID   string
}

type Runner struct {
    composer RuntimeComposer
}

func New(opts ...Option) *Runner
func (r *Runner) Prepare(ctx context.Context, req StartRequest) (*PreparedRun, error)
func (r *Runner) Start(ctx context.Context, req StartRequest) (*PreparedRun, *session.ExecutionHandle, error)
func (r *Runner) Run(ctx context.Context, req StartRequest) (*PreparedRun, *turns.Turn, error)
```

This structure intentionally supports three modes:

1. Tiny callers skip a composer entirely and give the runner a direct resolved runtime input.
2. Slightly larger callers use `Run`.
3. Advanced callers use `Prepare` and `Start` so they can inspect the runtime, attach their own bookkeeping, or run custom outer loops.

### Why `Prepare` matters

`Prepare` is the main difference between a toy helper and a serious app-facing runner.

CoinVault already separates preparation from execution in `2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go:89-156`. Temporal Relationships also needs app-owned outer-loop logic in `temporal-relationships/internal/extractor/gorunner/loop.go:129-205`.

If the new runner only offered `RunPrompt(ctx, "hello")`, it would be too small. `Prepare` lets advanced apps reuse the shared bootstrap while keeping control of the final execution policy.

### Tool registration story

The new runner should standardize two forms of tool input:

1. Function tools:

```go
oprunner.WithFuncTool("calc", "A calculator", calculatorTool)
```

2. Registrars:

```go
oprunner.WithToolRegistrars(
    entityhistory.NewLazyQueryToolRegistrar(...),
    transcripthistory.NewLazyQueryToolRegistrar(...),
)
```

Internally:

- The runner creates a fresh `tools.InMemoryToolRegistry`.
- It executes all registrars into that registry.
- It filters the resulting registry by `ToolNames` if present.
- It stores the filtered registry in `PreparedRun.Registry`.

This directly captures the patterns now split across CoinVault and Temporal Relationships.

### Middleware composition story

The opinionated runner should support three middleware sources:

1. Static Go middleware passed in options.
2. Middleware definitions plus app-provided `MiddlewareUses`.
3. App-owned composer output that already returns concrete middleware.

Recommended rule:

- If the composer already returns `[]middleware.Middleware`, use them directly.
- If it returns `MiddlewareUses` plus a `middlewarecfg.DefinitionRegistry`, resolve and build them centrally.

This lets the runner use Geppetto's existing `middlewarecfg` system rather than re-implementing Pinocchio's `ProfileRuntimeComposer`.

### Session bootstrap rules

The runner should define one standard bootstrap path:

1. If `SeedTurn` is provided, clone and normalize it.
2. Otherwise create an empty seed turn.
3. If there is no prior system block and `SeedSystemPrompt` is non-empty, rely on `middleware.NewSystemPromptMiddleware` instead of hard-coding system-block construction in multiple places.
4. Create `session.Session` with the requested `SessionID` or a generated one.
5. Append the seed turn if needed.
6. Append the prompt turn through `AppendNewTurnFromUserPrompt` for follow-up consistency.

Why prefer middleware for system prompt insertion?

- It keeps one canonical behavior source.
- It preserves block provenance tagging.
- It matches existing Pinocchio runtime behavior in `pinocchio/pkg/inference/runtime/engine.go:34-40`.

### Engine-builder assembly inside the opinionated runner

Internally the new runner should always end by creating an `enginebuilder.Builder`, not a parallel execution stack.

Pseudo-flow:

```go
func (r *Runner) Prepare(ctx context.Context, req StartRequest) (*PreparedRun, error) {
    runtime := r.composer.Compose(ctx, req.Runtime)
    reg := buildRegistry(runtime.ToolRegistrars, runtime.ToolNames)

    sess := session.NewSession()
    if req.SessionID != "" {
        sess.SessionID = req.SessionID
    }

    seed := normalizeSeed(req.SeedTurn)
    if seed != nil {
        sess.Append(seed)
    }

    appended, err := sess.AppendNewTurnFromUserPrompt(req.Prompt)
    if err != nil { return nil, err }

    builder := enginebuilder.New(
        enginebuilder.WithBase(runtime.Engine),
        enginebuilder.WithMiddlewares(runtime.Middlewares...),
        enginebuilder.WithToolRegistry(reg),
        enginebuilder.WithLoopConfig(runtime.LoopConfig),
        enginebuilder.WithToolConfig(runtime.ToolConfig),
        enginebuilder.WithEventSinks(runtime.EventSinks...),
        enginebuilder.WithSnapshotHook(runtime.SnapshotHook),
        enginebuilder.WithPersister(runtime.Persister),
        enginebuilder.WithStepController(runtime.StepController),
        enginebuilder.WithStepPauseTimeout(runtime.StepPauseTimeout),
    )

    sess.Builder = builder
    return &PreparedRun{Runtime: runtime, Registry: reg, Session: sess, SeedTurn: appended, TurnID: appended.ID}, nil
}
```

This preserves today's core engine behavior while eliminating repeated application glue.

### Relationship to Pinocchio runtime abstractions

The design should absorb or mirror the following concepts currently in Pinocchio:

- `ToolRegistrar` from `pinocchio/pkg/inference/runtime/engine.go:16-17`
- `ConversationRuntimeRequest`, `ComposedRuntime`, and `RuntimeBuilder` from `pinocchio/pkg/inference/runtime/composer.go:11-43`

Recommendation:

- Move these to Geppetto if possible.
- If moving them immediately is too disruptive, add equivalent Geppetto definitions and make Pinocchio adapt to them.
- Do not make Geppetto core depend on profile-resolution types to express these contracts.

That avoids a future where downstream apps need Pinocchio as a type-only dependency to use a Geppetto runner.

## Practical API Examples

### Example 1: tiny CLI with two function tools

This is the "couple of lines" target.

```go
runner := oprunner.Must(
    oprunner.FromParsedValues(parsed),
    oprunner.WithSystemPrompt("You are a concise ops assistant."),
    oprunner.WithFuncTool("calc", "A calculator", calculatorTool),
    oprunner.WithFuncTool("inventory_summary", "Summarize inventory", inventorySummaryTool),
)

_, turn, err := runner.Run(ctx, oprunner.StartRequest{
    Prompt: "Use inventory_summary and then tell me the shop overview.",
})
```

Why this matters:

- This replaces the repeated session, builder, registry, and wait boilerplate seen in `geppetto/cmd/examples/openai-tools/main.go:316-398` and `geppetto/cmd/examples/middleware-inference/main.go:220-244`.

### Example 2: profile-driven web chat

```go
runner := oprunner.Must(
    oprunner.WithRuntimeComposer(profileComposer),
    oprunner.WithToolRegistrars(toolCatalog.Registrars()...),
    oprunner.WithMiddlewareDefinitions(defRegistry, buildDeps),
)

prep, handle, err := runner.Start(ctx, oprunner.StartRequest{
    SessionID: req.SessionID,
    Prompt:    req.Prompt,
    Runtime: oprunner.RuntimeRequest{
        RuntimeKey:         resolved.RuntimeKey,
        RuntimeFingerprint: resolved.RuntimeFingerprint,
        ProfileVersion:     resolved.ProfileVersion,
        StepSettings:       resolved.StepSettings,
        SystemPrompt:       resolved.SystemPrompt,
        MiddlewareUses:     resolved.MiddlewareUses,
        ToolNames:          resolved.ToolNames,
    },
})
```

Why this matters:

- This collapses the manual composition now spread across `pinocchio/cmd/web-chat/runtime_composer.go:35-177` and `2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go:92-156`.

### Example 3: custom outer loop with shared preparation

```go
prep, err := runner.Prepare(ctx, oprunner.StartRequest{
    SessionID: extractionRun.GeppettoSessionID,
    Prompt:    firstPrompt,
    Runtime:   runtimeReq,
})
if err != nil { return err }

for i := 0; i < cfg.MaxIterations; i++ {
    handle, err := prep.Session.StartInference(iterCtx)
    if err != nil { return err }
    turn, err := handle.Wait()
    if err != nil { return err }
    if shouldStop(turn) { break }
    _, err = prep.Session.AppendNewTurnFromUserPrompt(cfg.ContinuePrompt)
    if err != nil { return err }
}
```

Why this matters:

- Temporal Relationships still needs its custom outer stop policy, but it no longer needs to duplicate runtime composition and registry bootstrap logic.

### Example 4: simple streaming HTTP server with SSE

The opinionated API should not force everything through `Run(...)`. A simple streaming server still wants:

- async `Start(...)`,
- one or more event sinks,
- normal session lifecycle and tool-loop behavior,
- the ability to wait for completion after the stream is already flowing.

```go
func handleChatStream(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming unsupported", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    sink := events.EventSinkFunc(func(ev events.Event) error {
        payload, err := json.Marshal(ev)
        if err != nil {
            return err
        }
        if _, err := fmt.Fprintf(w, "event: %s\n", ev.Type()); err != nil {
            return err
        }
        if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
            return err
        }
        flusher.Flush()
        return nil
    })

    runtimeReq := oprunner.RuntimeRequest{
        RuntimeKey:         resolved.RuntimeKey,
        RuntimeFingerprint: resolved.RuntimeFingerprint,
        StepSettings:       resolved.StepSettings,
        SystemPrompt:       resolved.SystemPrompt,
        MiddlewareUses:     resolved.MiddlewareUses,
        ToolNames:          resolved.ToolNames,
    }

    prep, handle, err := runner.Start(ctx, oprunner.StartRequest{
        SessionID: req.SessionID,
        Prompt:    req.Prompt,
        Runtime:   runtimeReq,
        EventSinks: []events.EventSink{
            sink,
        },
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    _ = prep

    turn, err := handle.Wait()
    if err != nil {
        _, _ = fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
        flusher.Flush()
        return
    }

    b, _ := json.Marshal(turn)
    _, _ = fmt.Fprintf(w, "event: completed\ndata: %s\n\n", b)
    flusher.Flush()
}
```

Why this matters:

- This is the simplest realistic streaming-server story for web chat, internal tools, or a CLI daemon.
- The runner still owns the hard parts like session bootstrap and tool-loop assembly, while the server owns only transport concerns.

### Example 5: event-driven worker using a channel sink

Some applications do not stream over HTTP directly. They want to feed inference events into another subsystem such as:

- a WebSocket hub,
- a message bus,
- a GUI event loop,
- a test harness collecting events deterministically.

That still fits naturally if the opinionated runner exposes sinks on `StartRequest` or `Runner` options.

```go
type channelSink struct {
    ch chan events.Event
}

func (s *channelSink) PublishEvent(ev events.Event) error {
    s.ch <- ev
    return nil
}

func runStreamingJob(ctx context.Context, prompt string) error {
    evCh := make(chan events.Event, 128)
    sink := &channelSink{ch: evCh}

    go func() {
        defer close(evCh)
        for ev := range evCh {
            switch e := ev.(type) {
            case *events.PartialCompletionEvent:
                log.Printf("partial: %s", e.Delta)
            case *events.ToolCallEvent:
                log.Printf("tool call: %s", e.ToolCall.Name)
            case *events.FinalEvent:
                log.Printf("final: %s", e.Text)
            }
        }
    }()

    _, handle, err := runner.Start(ctx, oprunner.StartRequest{
        Prompt: prompt,
        Runtime: oprunner.RuntimeRequest{
            StepSettings: settings,
            SystemPrompt: "You are a concise assistant.",
            ToolRegistrars: []oprunner.ToolRegistrar{
                weatherRegistrar,
                inventoryRegistrar,
            },
        },
        EventSinks: []events.EventSink{sink},
    })
    if err != nil {
        return err
    }

    _, err = handle.Wait()
    return err
}
```

Why this matters:

- It shows the runner can still be event-driven and transport-agnostic.
- The opinionated API is not "synchronous only"; `Run(...)` is just the small default path, while `Start(...)` remains the streaming/event path.

## Detailed Rationale For The Recommended Design

### Rationale 1: preserve the low-level stack

This design does not ask Geppetto to abandon its clean lower layers. It formalizes how applications already use them.

### Rationale 2: centralize the right abstractions

The abstractions that should move into the opinionated layer are the ones repeated across applications:

- tool registrars,
- runtime composers,
- app-owned registry filtering,
- session bootstrap,
- builder assembly,
- sync and async run entrypoints,
- event-sink wiring for streaming and event-driven applications.

The abstractions that should stay low-level are:

- Turn mutation,
- tool execution semantics,
- provider engines,
- middleware contracts,
- step controller implementation.

### Rationale 3: align Go with existing JS ergonomics

The JS surface already exposes a higher-level builder language. Adding a Go opinionated runner makes the ecosystem more internally consistent.

### Rationale 4: avoid a false choice between "simple" and "powerful"

The new layer should not force a simple CLI API on advanced apps. That is why `Prepare` is part of the recommendation.

### Rationale 5: preserve event-driven integration

The opinionated runner should not accidentally become a blocking convenience wrapper only. Many real applications are event-driven:

- web chat streams tokens to browsers,
- terminal UIs react to partial updates,
- workers forward inference lifecycle events into other systems.

That means `Start(...)` plus explicit `EventSinks` is not an advanced afterthought. It is a first-class path in the API design.

## Proposed File-Level Implementation Plan

### Phase 1: add the new Geppetto package

Create:

- `geppetto/pkg/inference/opinionated/runner.go`
- `geppetto/pkg/inference/opinionated/runtime.go`
- `geppetto/pkg/inference/opinionated/registrars.go`
- `geppetto/pkg/inference/opinionated/prepare.go`
- `geppetto/pkg/inference/opinionated/result.go`

Responsibilities:

- option parsing,
- runtime composer contracts,
- registry construction and filtering,
- session bootstrap,
- builder assembly,
- run/start helpers.

### Phase 2: move or mirror shared abstractions from Pinocchio

Candidates:

- `ToolRegistrar`
- runtime request/result interfaces
- middleware-definition integration helpers

Likely touch points:

- `pinocchio/pkg/inference/runtime/composer.go`
- `pinocchio/pkg/inference/runtime/engine.go`

Goal:

- make Pinocchio depend on the new Geppetto layer instead of owning parallel types.
- keep profile selection and final `StepSettings` resolution on the Pinocchio side.

### Phase 3: add first-party examples

Replace boilerplate-heavy examples with short opinionated-runner examples:

- `geppetto/cmd/examples/simple-inference/main.go`
- `geppetto/cmd/examples/middleware-inference/main.go`
- `geppetto/cmd/examples/openai-tools/main.go`

Also add a new playbook under `geppetto/pkg/doc/playbooks/`.

### Phase 4: migrate Pinocchio adapters

Candidates:

- `pinocchio/pkg/ui/backends/toolloop/backend.go`
- `pinocchio/pkg/ui/profileswitch/backend.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`

Goal:

- shrink these packages into app-owned policy adapters instead of session/bootstrap frameworks.

### Phase 5: migrate downstream apps incrementally

Good first migrations:

- `2026-03-14--cozodb-editor/backend/pkg/hints/engine.go`
- `2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go`
- `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go`

Keep `temporal-relationships/internal/extractor/gorunner/loop.go` partially custom until the outer loop policy is proven compatible.

## Testing And Validation Strategy

### Unit tests

Add unit tests for:

1. building a registry from function tools and registrars,
2. app-owned registry filtering by `ToolNames`,
3. composing app-provided middleware uses through `middlewarecfg`,
4. preserving sinks, snapshot hooks, persisters, and step controller settings,
5. `Prepare`, `Start`, and `Run` lifecycle behavior,
6. seed-turn and follow-up prompt behavior,
7. nil-runtime and nil-engine failures.

### Compatibility tests

Add migration-style tests that compare old and new behavior for:

- simple inference,
- tool loop inference,
- profile runtime composition,
- lazy scoped database tools.

### Documentation validation

Update the docs so the first recommended path for Go users becomes the new opinionated runner, while still documenting the low-level stack for advanced cases.

## Risks

### Risk 1: package boundary drift

If the new package starts owning too much app logic, Geppetto becomes harder to reason about. Keep it focused on runtime assembly.

### Risk 2: partial duplication with Pinocchio during migration

This will happen for a while. Plan for adapters rather than a flag day rewrite.

### Risk 3: unclear naming

The package name matters less than the surface. Avoid names that sound lower-level than they are.

### Risk 4: reintroducing profile concerns into runner-core

They should not. The runner should consume resolved runtime input, not re-grow Geppetto-owned profile resolution. `Prepare` stays essential, but the bigger boundary discipline is keeping profile/app policy outside.

## Alternatives Considered And Rejected

Rejected alternatives:

1. More examples only.
2. A pile of helper functions with no shared runtime model.
3. Keeping the whole abstraction in Pinocchio.
4. Expanding `enginebuilder.Builder` into an app framework.

## Open Questions

1. Should Geppetto own the runtime-composer interfaces directly, or should the opinionated package merely adapt existing Pinocchio interfaces during the first phase?
2. Should registry filtering happen only after registration, or should the runner also offer a pre-registration catalog filter hook for expensive lazy registrars?
3. Should the default middleware set always include `NewToolResultReorderMiddleware`, matching Pinocchio's `BuildEngineFromSettingsWithMiddlewares` behavior in `pinocchio/pkg/inference/runtime/engine.go:34-40`?
4. Should the new API provide a `RunTurn` method in addition to `RunPrompt` to support non-text seed turns and replay-style workflows?
5. Should `fixtures.ExecuteFixture` in `geppetto/pkg/inference/fixtures/fixtures.go:96-252` eventually migrate to the new runner or remain intentionally separate because it is a recording-oriented utility?

## References

Core Geppetto runtime:

- `geppetto/pkg/inference/session/session.go`
- `geppetto/pkg/inference/session/builder.go`
- `geppetto/pkg/inference/toolloop/enginebuilder/builder.go`
- `geppetto/pkg/inference/toolloop/enginebuilder/options.go`
- `geppetto/pkg/inference/toolloop/loop.go`
- `geppetto/pkg/inference/toolloop/step_controller.go`
- `geppetto/pkg/inference/tools/definition.go`
- `geppetto/pkg/inference/tools/registry.go`
- `geppetto/pkg/inference/tools/base_executor.go`
- `geppetto/pkg/inference/tools/config.go`
- `geppetto/pkg/inference/middleware/middleware.go`
- `geppetto/pkg/inference/middleware/systemprompt_middleware.go`
- `geppetto/pkg/inference/middleware/reorder_tool_results_middleware.go`
- `geppetto/pkg/inference/middlewarecfg/definition.go`
- `geppetto/pkg/inference/middlewarecfg/resolver.go`
- `geppetto/pkg/inference/middlewarecfg/chain.go`
- `geppetto/pkg/profiles/types.go`

Geppetto examples and docs:

- `geppetto/cmd/examples/simple-inference/main.go`
- `geppetto/cmd/examples/middleware-inference/main.go`
- `geppetto/cmd/examples/openai-tools/main.go`
- `geppetto/pkg/doc/topics/07-tools.md`
- `geppetto/pkg/doc/topics/09-middlewares.md`
- `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md`

Geppetto JS API:

- `geppetto/pkg/js/modules/geppetto/api_sessions.go`
- `geppetto/pkg/js/modules/geppetto/api_builder_options.go`
- `geppetto/pkg/js/modules/geppetto/api_tools_registry.go`
- `geppetto/pkg/js/modules/geppetto/api_middlewares.go`

Pinocchio:

- `pinocchio/pkg/inference/runtime/composer.go`
- `pinocchio/pkg/inference/runtime/engine.go`
- `pinocchio/pkg/ui/backends/toolloop/backend.go`
- `pinocchio/pkg/ui/profileswitch/backend.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`

Downstream applications:

- `2026-03-14--cozodb-editor/backend/pkg/hints/engine.go`
- `2026-03-16--gec-rag/internal/webchat/configurable_loop_runner.go`
- `2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go`
- `2026-03-16--gec-rag/internal/webchat/tool_catalog.go`
- `2026-03-16--gec-rag/internal/webchat/tools.go`
- `2026-03-16--gec-rag/internal/webchat/runtime.go`
- `2026-03-16--gec-rag/internal/webchat/resolver.go`
- `temporal-relationships/internal/extractor/gorunner/loop.go`
- `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go`
- `temporal-relationships/internal/extractor/entityhistory/tool.go`
- `temporal-relationships/internal/extractor/transcripthistory/tool.go`
- `temporal-relationships/internal/extractor/runturnhistory/tool.go`

## Problem Statement

<!-- Describe the problem this design addresses -->

## Proposed Solution

<!-- Describe the proposed solution in detail -->

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
