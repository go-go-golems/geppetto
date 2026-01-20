---
Title: 'Analysis: Engine sinks vs context sinks; PinocchioCommand runner; Session run lifecycle (StartRun/FinishRun/HasCancel)'
Ticket: MO-004-UNIFY-INFERENCE-STATE
Status: active
Topics:
    - inference
    - architecture
    - webchat
    - prompts
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/context.go
      Note: |-
        Context sink attachment + PublishEventToContext
        Explains context sinks and PublishEventToContext
    - Path: geppetto/pkg/events/sink.go
      Note: EventSink interface
    - Path: geppetto/pkg/inference/core/session.go
      Note: |-
        Session runner; RunInference vs RunInferenceStarted; ctx wiring
        Explains RunInference vs RunInferenceStarted and HasCancel
    - Path: geppetto/pkg/inference/engine/options.go
      Note: |-
        Engine config sinks (engine.WithSink / Config.EventSinks)
        Explains engine-config sinks (engine.WithSink)
    - Path: geppetto/pkg/inference/middleware/sink_watermill.go
      Note: WatermillSink EventSink implementation
    - Path: geppetto/pkg/inference/state/state.go
      Note: |-
        InferenceState lifecycle (StartRun/FinishRun/HasCancel/CancelRun)
        Explains StartRun/FinishRun/CancelRun semantics
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Provider engine attaches config sinks into ctx at RunInference start
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: |-
        PinocchioCommand runner (blocking + chat paths)
        PinocchioCommand runner inference paths
    - Path: pinocchio/pkg/ui/backend.go
      Note: |-
        Bubble Tea backend uses StartRun + RunInferenceStarted
        Bubble Tea run start/cancel pattern
    - Path: pinocchio/pkg/webchat/router.go
      Note: Webchat uses StartRun + RunInferenceStarted + Watermill sinks
ExternalSources: []
Summary: Explains event sink plumbing (engine-config sinks vs context sinks), PinocchioCommand inference paths, and Session/InferenceState run lifecycle (StartRun/FinishRun/HasCancel).
LastUpdated: 2026-01-20T21:10:00-05:00
WhatFor: Make it clear which layer owns lifecycle/publishing responsibilities and why Session has two run entrypoints.
WhenToUse: When adding a new runner (TUI/webchat) or changing event/middleware/inference orchestration.
---


# Analysis: Engine sinks vs context sinks; PinocchioCommand runner; Session run lifecycle (StartRun/FinishRun/HasCancel)

## Goal

This document explains:

1) how inference events flow end-to-end (including why we have two sink injection points),
2) what `PinocchioCommand` is and how it executes inference in blocking and chat modes,
3) why `core.Session` has `RunInference` vs `RunInferenceStarted`,
4) what `InferenceState.StartRun/FinishRun/HasCancel/CancelRun` do, and why.

It’s written to answer these questions directly:

- What are **engine config sinks** used for? Can we remove them given we now commonly use sinks from context?
- What is the **PinocchioCommand runner**, where is it used, and how does it do inference?
- Why do we have **RunInference vs RunInferenceStarted**, and when is one used vs the other?
- Why **HasCancel**? What are the two ways of starting a run, and can we unify them?
- What are **StartRun and FinishRun**, where are they defined, and what do they do?

## Glossary

- **Event**: Structured inference progress output (start, partial/streaming, tool call/use, final, error).
- **EventSink**: Destination for events.
- **Context sinks**: Sinks attached to `context.Context` using `events.WithEventSinks`.
- **Engine-config sinks**: Sinks attached to an engine’s `engine.Config` via `engine.WithSink(...)`.
- **Session**: `geppetto/pkg/inference/core.Session` (runner/orchestrator).
- **InferenceState**: `geppetto/pkg/inference/state.InferenceState` (per-session run state).

## 1) Event sinks: interface and publishing model

### 1.1 `EventSink` contract

`geppetto/pkg/events/sink.go` defines the core interface:

```go
type EventSink interface {
    PublishEvent(event Event) error
}
```

Concrete implementations include:

- `middleware.WatermillSink` (`geppetto/pkg/inference/middleware/sink_watermill.go`): publishes JSON events to a Watermill topic.
- Various printer/adapter sinks (e.g. step printers) that render events to stdout or convert them into UI messages.

### 1.2 Context sinks: the unified publishing mechanism

`geppetto/pkg/events/context.go` provides:

- `events.WithEventSinks(ctx, sinks...) context.Context`
- `events.PublishEventToContext(ctx, event)`

Publishing is intentionally “best effort”: if there are no sinks, it’s a no-op; if a sink fails, the error is ignored so inference isn’t disrupted.

Conceptually:

```
publishers (engine/tool loop/middleware)
  |
  v
events.PublishEventToContext(ctx, ev)
  |
  v
events.GetEventSinks(ctx) -> []EventSink -> PublishEvent(ev)
```

This is critical because tool loops and middlewares do not (and should not) depend on provider-engine configuration types.

## 2) Engine-config sinks vs context sinks

### 2.1 What “engine-config sinks” are

`geppetto/pkg/inference/engine/options.go` defines:

- `type Config struct { EventSinks []events.EventSink }`
- `engine.WithSink(sink)` appends to `Config.EventSinks`

Historically, this existed so code that *creates the engine* could also decide where the engine should publish events, without needing to thread sinks through every single `RunInference` call.

Example: `pinocchio/pkg/cmds/cmd.go` uses `engine.WithSink(middleware.NewWatermillSink(...))` in blocking mode.

### 2.2 Why engine-config sinks and context sinks diverged

Before the recent unification work:

- tool loops / middleware generally published via context (because it’s composable),
- provider engines sometimes published directly to `Config.EventSinks` (because it was “convenient”),
- and sometimes also to context.

That split means a shared runner like `core.Session` would either:

- attach sinks to context (good for tool-loop events) but miss “engine-config-only” engine events, or
- rely on engine-config sinks (good for engine events) but miss context-published tool-loop events.

### 2.3 The current unification: engines attach their config sinks into the run context

Provider engines now begin `RunInference` by making config sinks available through the run context:

```go
if len(e.config.EventSinks) > 0 {
    ctx = events.WithEventSinks(ctx, e.config.EventSinks...)
}
```

Example: `geppetto/pkg/steps/ai/openai_responses/engine.go`.

Then engines publish exclusively via:

```go
events.PublishEventToContext(ctx, ev)
```

Net effect:

- engine-config sinks still work (for existing call sites),
- tool-loop and middleware events share the same sink set,
- a `core.Session` can attach per-run sinks to context and all publishers see them.

### 2.4 Can we remove engine-config sinks?

**Capability-wise: yes. Pragmatically: not yet without significant churn.**

#### Why we might remove them eventually

- Fewer ways to do the same thing.
- Lower risk of duplicate event delivery (same sink added on engine + on ctx).
- Engines become simpler “pure inference” units; wiring becomes the runner’s job.

#### What makes removal expensive

- Many call sites currently build engines with `engine.WithSink(...)` and then call `Eng.RunInference(...)` directly.
- Engine factories accept engine options; removing `WithSink` would force call sites to add context sinks around every inference call, or migrate those flows to `core.Session`.

#### Recommended near-term convention (to avoid duplicates)

Per application/entrypoint, pick one strategy:

- Strategy A: configure sinks on the engine (via `engine.WithSink`) and let engines attach them into ctx at runtime.
- Strategy B: do not configure engine sinks; attach sinks via `core.Session.EventSinks` (or explicit `events.WithEventSinks`) and keep engines sink-agnostic.

Do not do both with the same sink instance unless you intentionally want duplication.

## 3) `core.Session`: what it wires and why it exists

`geppetto/pkg/inference/core/session.go` provides a single orchestration object with stable dependencies:

- `State *InferenceState` (engine + current turn + run id + cancel + running flag)
- tool loop inputs: `Registry`, `ToolConfig`
- event plumbing: `EventSinks`
- debugging/persistence: `SnapshotHook`, optional `Persister`

### 3.1 What Session does per run (pseudocode)

`RunInference` (full lifecycle owner):

```go
StartRun()
runCtx, cancel := context.WithCancel(ctx)
SetCancel(cancel)
defer cancel()
defer FinishRun()
return RunInferenceStarted(runCtx, seed)
```

`RunInferenceStarted` (caller already started run):

- If state has no cancel: create one (so Interrupt works).
- Attach sinks to ctx (`events.WithEventSinks`).
- Attach snapshot hook to ctx (`toolhelpers.WithTurnSnapshotHook`).
- Execute either:
  - single pass: `Eng.RunInference`
  - tool loop: `toolhelpers.RunToolCallingLoop`
- Update `State.Turn` (and `State.RunID` if present on the returned turn).

## 4) `InferenceState`: StartRun / FinishRun / CancelRun / HasCancel

`geppetto/pkg/inference/state/state.go` defines:

```go
type InferenceState struct {
    RunID string
    Turn  *turns.Turn
    Eng   engine.Engine

    mu      sync.Mutex
    running bool
    cancel  context.CancelFunc
}
```

### 4.1 `StartRun()`: “single-run guard”

- Sets `running = true` (under mutex).
- If already running, returns `ErrInferenceRunning`.

Used to prevent:

- double-submit in TUI,
- concurrent POST /chat calls for the same conversation.

### 4.2 `FinishRun()`: “cleanup”

- Clears `running` and `cancel`.
- Puts the conversation/session back into an idle state.

### 4.3 `CancelRun()` + `HasCancel()`: cancellation wiring

- `SetCancel(cancel)` stores the cancel function for the currently active run.
- `CancelRun()` calls that cancel function **only** if the run is active.
- `HasCancel()` reports whether a cancel is currently stored.

## 5) Why `RunInference` vs `RunInferenceStarted` (and why `HasCancel` exists)

### 5.1 The two real-world “start patterns”

There are two valid usage patterns in our apps:

#### Pattern A: synchronous “do everything”

The runner owns the entire lifecycle:

- `StartRun` inside runner
- create cancel inside runner
- `FinishRun` inside runner

That’s `Session.RunInference(ctx, seed)`.

This is a good fit for:

- CLI blocking commands
- tests
- any place where you don’t need to return early to a UI loop.

#### Pattern B: asynchronous UI “claim run first”

The UI/handler wants to:

- reject immediately if a run is already active (`StartRun`),
- ensure `CancelRun` works immediately (store cancel before async work),
- run inference on a goroutine / `tea.Cmd`,
- call `FinishRun` when the async work completes.

That’s:

```
StartRun()
SetCancel(cancel)
go/tea.Cmd: RunInferenceStarted(...)
defer FinishRun()
```

This is used by:

- Pinocchio TUI backend: `pinocchio/pkg/ui/backend.go` (`EngineBackend.Start`)
- Pinocchio webchat: `pinocchio/pkg/webchat/router.go` (HTTP handler spawns goroutine)

### 5.2 Why `RunInferenceStarted` checks `HasCancel`

If `RunInferenceStarted` always created a new cancel context and overwrote state:

- it could clobber a cancel that the UI stored earlier,
- meaning UI “Interrupt” would cancel the wrong context (or do nothing).

So `RunInferenceStarted` does:

```go
if !State.HasCancel() {
    runCtx, cancel := context.WithCancel(ctx)
    State.SetCancel(cancel)
    defer cancel()
}
```

This respects a caller-provided cancel, but still guarantees a cancel exists for non-UI callers.

### 5.3 Can we unify the two run entrypoints?

Yes, but only by choosing a single owner for run lifecycle.

Two plausible directions:

- **Make Session the only lifecycle owner** (remove `RunInferenceStarted`): simpler API, but worse fit for Bubble Tea/webchat where “claim run before async work” is valuable.
- **Move cancel creation into `StartRun`** (so `StartRun` returns a `runCtx` + `finish()`): removes `HasCancel` and compresses lifecycle into one structured helper, but is a breaking API change.

The current implementation keeps both because Pattern B is essential for our UIs.

## 6) PinocchioCommand runner: what it is, where it’s used, and how it runs inference

### 6.1 What it is

`pinocchio/pkg/cmds/cmd.go` defines `PinocchioCommand`, which is a Glazed command wrapper that:

- renders templated strings (system prompt, blocks/messages, user prompt),
- builds a seed `turns.Turn` from those inputs,
- chooses run mode: blocking vs interactive vs chat,
- wires event routing/sinks when needed.

It is the “glue” between:

```
CLI args + parsed layers -> rendered prompt(s) -> Turn -> inference run -> output/UI
```

### 6.2 Blocking mode inference path

In blocking mode, it:

1) creates an engine via `EngineFactory.CreateEngine(stepSettings, options...)`
2) builds a seed turn via `g.buildInitialTurn(...)`
3) calls `engine.RunInference(ctx, seed)`

If an event router is active, it attaches a Watermill sink as an **engine-config sink**:

- `options = append(options, engine.WithSink(middleware.NewWatermillSink(...)))`

So inference events are published to Watermill and can be printed/consumed.

### 6.3 Chat mode inference path

In chat mode, `PinocchioCommand`:

- enables streaming in settings,
- starts the event router,
- constructs a Bubble Tea program via `runtime.NewChatBuilder().BuildProgram()`.

The actual per-prompt inference in chat mode is performed by the backend (`pinocchio/pkg/ui/backend.go`) which uses `InferenceState` + `core.Session`.

## Appendix: sequence diagrams

### A.1 Pinocchio TUI submit

```
User hits <Tab>
  |
  v
EngineBackend.Start(ctx, prompt)                         pinocchio/pkg/ui/backend.go
  | StartRun(); SetCancel(cancel)
  | return tea.Cmd
  v
tea.Cmd executes
  |
  v
Session.RunInferenceStarted(runCtx, seed)                geppetto/pkg/inference/core/session.go
  | attach ctx sinks, snapshot hook
  | Eng.RunInference or ToolCallingLoop(...)
  v
provider engine publishes events via PublishEventToContext(ctx, ev)
  |
  v
WatermillSink -> event router -> UI subscriber -> timeline updates
```

### A.2 Pinocchio webchat POST /chat

```
HTTP POST /chat
  |
  v
router handler                                       pinocchio/pkg/webchat/router.go
  | conv.Inf.StartRun() (409 if running)
  | spawn goroutine
  v
goroutine:
  | SetCancel(runCancel); defer FinishRun
  | sess := Session{State: conv.Inf, EventSinks: [conv.Sink], Registry: ...}
  | sess.RunInferenceStarted(runCtx, seedForPrompt(...))
  v
events -> watermill topic -> websocket forwarder -> browser UI
```
