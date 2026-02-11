---
Title: ToolCallingLoop struct + step control integration
Ticket: GP-07-STEP-CONTROLLER-INTEGRATION
Status: active
Topics:
    - geppetto
    - backend
    - conversation
    - events
    - websocket
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/session/execution.go
      Note: |-
        ExecutionHandle cancel/wait model; step waits must observe ctx cancellation
        ExecutionHandle cancellation semantics that pause waits must observe
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: |-
        Current builder-owned dispatch between single-pass inference and tool calling loop
        Runner currently calls RunToolCallingLoop; wiring point for ToolCallingLoop struct
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: |-
        Current `RunToolCallingLoop` function and its phase boundaries
        Current RunToolCallingLoop; proposed refactor target
    - Path: moments/backend/pkg/webchat/loops.go
      Note: |-
        Existing step-mode pause points and debugger.pause emission (conversation-coupled)
        Baseline pause points and debugger.pause events (conversation-coupled)
    - Path: moments/backend/pkg/webchat/step_controller.go
      Note: |-
        Baseline StepController (no ctx cancellation support)
        Baseline StepController API to generalize into cancellation-safe stepcontrol
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T22:13:34.199547884-05:00
WhatFor: ""
WhenToUse: ""
---


# ToolCallingLoop struct + step control integration

## Prompt context (why this doc exists)

GP-07’s prior analysis focused on how Moments implements step mode today and how to integrate stepping into Geppetto’s session model.

This follow-up proposes a specific architectural move:

- Step control should be **neither**:
  - a middleware (engine-level wrapping), **nor**
  - a webchat conversation field (app-owned mutable state),
- and should instead be integrated directly into the canonical tool-calling loop.

Concretely:

- Convert `toolhelpers.RunToolCallingLoop` (function) into a small struct with a `RunLoop` method.
- Use the same functional-options (“With*”) constructor pattern as `session.NewToolLoopEngineBuilder(...)`.
- Define what “StepController” becomes in this world, where it is wired in, and how webchat finds and continues the correct paused execution.

## Current state (as of 2026-01-23)

### Where the tool loop lives

In Geppetto today, the tool-calling loop is the function:

- `geppetto/pkg/inference/toolhelpers/helpers.go`: `RunToolCallingLoop(ctx, eng, turn, registry, cfg)`

and it exposes stable phase boundaries via snapshot hooks:

- `pre_inference`
- `post_inference`
- `post_tools`

`session.ToolLoopEngineBuilder` chooses between:

- single-pass inference (`eng.RunInference`), and
- tool loop (`toolhelpers.RunToolCallingLoop`),

inside the session runner (`toolLoopRunner.RunInference`).

### Where “step control” lives in Moments

In `moments/backend/pkg/webchat/loops.go`, stepping is implemented by reaching into webchat conversation state:

- `conv.StepCtrl` (conversation-owned)
- pause points in `ToolCallingLoop` (Moments’ function)
- emits `debugger.pause`
- blocks on `StepCtrl.Wait(...)` (does not observe `ctx.Done()`)

This creates two problems for Pinocchio/Geppetto:

1. The pause gate is **not** part of the canonical Geppetto tool loop, so step behavior is easy to drift or forget.
2. The pause gate is **conversation-coupled**, which is exactly what we want to avoid.

## Design goals

1. Make stepping a **first-class tool loop feature**, not a middleware or app-owned hook.
2. Ensure pausing is **cancellation-safe**:
   - if `ExecutionHandle.Cancel()` is called, a paused run must unblock immediately (no 30s hang).
3. Keep the tool loop reusable across:
   - Pinocchio webchat (in-memory or Redis)
   - Moments (authorized tool executor)
   - any other app using `session.ToolLoopEngineBuilder`
4. Keep the “With*” construction style consistent with existing patterns:
   - `session.NewToolLoopEngineBuilder(...)`
   - `toolhelpers.ToolConfig.With...`

## Proposal: turn the tool loop into a struct

### New type (toolhelpers)

Introduce a loop object that owns configuration and optional integration points:

```go
// package toolhelpers
type ToolCallingLoop struct {
    eng      engine.Engine
    registry tools.ToolRegistry
    cfg      ToolConfig

    // Optional overrides / integration points
    executor      tools.ToolExecutor
    snapshotHook  SnapshotHook          // if non-nil, overrides ctx-attached hook
    stepper       Stepper               // optional pause/continue gate
    pauseEmitter  PauseEmitter          // optional: publish debugger.pause event
    pauseTimeout  time.Duration         // default: 30s (auto-continue)
}

type ToolCallingLoopOption func(*ToolCallingLoop)

func NewToolCallingLoop(opts ...ToolCallingLoopOption) *ToolCallingLoop

func WithToolCallingEngine(eng engine.Engine) ToolCallingLoopOption
func WithToolCallingRegistry(reg tools.ToolRegistry) ToolCallingLoopOption
func WithToolCallingConfig(cfg ToolConfig) ToolCallingLoopOption
func WithToolCallingExecutor(exec tools.ToolExecutor) ToolCallingLoopOption
func WithToolCallingSnapshotHook(h SnapshotHook) ToolCallingLoopOption
func WithToolCallingStepper(s Stepper) ToolCallingLoopOption
func WithToolCallingPauseEmitter(e PauseEmitter) ToolCallingLoopOption
func WithToolCallingPauseTimeout(d time.Duration) ToolCallingLoopOption
```

and replace `RunToolCallingLoop(...)` with:

```go
func (l *ToolCallingLoop) RunLoop(ctx context.Context, initial *turns.Turn) (*turns.Turn, error)
```

### Why a struct helps

- We stop threading unrelated integration knobs through a single free function signature.
- Step mode wiring becomes explicit: `WithToolCallingStepper(...)`.
- We can add new cross-cutting integrations later (e.g., metrics, tracing, run-limit behavior) without multiplying parameters.

### Backwards compatibility

GP-07 explicitly allows breaking changes.

However, if we want a gentler transition, we can keep:

- `RunToolCallingLoop(...)` as a thin wrapper that constructs `NewToolCallingLoop(...)` and calls `RunLoop`.

## What “StepController” becomes

The critical shift is: the pause gate should not be “owned by conversation”.

Instead it should be “owned by the currently running inference”, and should:

- register pause waiters,
- allow continue by `pause_id`,
- unblock on:
  - continue, OR
  - timeout, OR
  - `ctx.Done()` (cancellation), OR
  - step mode disabled.

### Proposed interface (stepcontrol package)

To keep `toolhelpers` clean and avoid importing session/webchat packages, define a small package:

- `geppetto/pkg/inference/stepcontrol`

Example:

```go
// package stepcontrol
type Phase string

const (
    PhaseAfterInference Phase = "after_inference"
    PhaseAfterTools     Phase = "after_tools"
)

type PauseRequest struct {
    Phase   Phase
    Summary string
    Extra   map[string]any
}

type Pause struct {
    PauseID     string
    DeadlineMs  int64
    Phase       Phase
}

type Controller interface {
    Enabled() bool

    // Pause registers a pause and returns the pause token. It should be cheap and non-blocking.
    Pause(ctx context.Context, req PauseRequest) (Pause, error)

    // Wait blocks until continued/timeout/cancel/disabled.
    Wait(ctx context.Context, pauseID string, timeout time.Duration) error

    Continue(pauseID string) bool
    Disable() // should unblock all waiters immediately
}
```

Then `toolhelpers.ToolCallingLoop` can depend only on the interface via an adapter:

```go
// package toolhelpers
type Stepper interface {
    Enabled() bool
    Pause(ctx context.Context, req StepPauseRequest) (pauseID string, deadlineMs int64, err error)
    Wait(ctx context.Context, pauseID string, timeout time.Duration) error
}
```

The key requirement: `Wait` must `select` on `<-ctx.Done()` so `ExecutionHandle.Cancel()` unblocks immediately.

### Where it is wired in

Recommendation:

- `session.ToolLoopEngineBuilder` grows an optional field (or option) for step control, and passes it down into the runner.
- The runner, when `Registry != nil`, creates a `toolhelpers.ToolCallingLoop` with `WithToolCallingStepper(...)`.

This keeps the responsibility aligned:

- Session owns “inference execution configuration”.
- Tool loop owns “where the pause points are”.

## Where the pause points live (in the tool loop)

### After inference (only if there are pending tool calls)

In `RunLoop`, immediately after `post_inference` snapshot:

- compute `pending := toolblocks.ExtractPendingToolCalls(updated)`
- if `len(pending) > 0` and `stepper.Enabled()`:
  - create pause id
  - emit pause event (optional)
  - `stepper.Wait(ctx, pauseID, pauseTimeout)`

### After tools (always, but only if enabled)

After appending tool results and `post_tools` snapshot:

- if `stepper.Enabled()`:
  - pause + wait

## Emitting `debugger.pause` (who does it)

We want the tool loop to be able to emit a pause event *without* knowing anything about Pinocchio/Moments webchat types.

Two viable approaches:

### A) Tool loop publishes a Geppetto-native event

Add a Geppetto event type:

- `events.EventDebuggerPause` with `Type_ = "debugger.pause"`

Then `ToolCallingLoop` can call `events.PublishEventToContext(ctx, ev)` at pause points.

Pros:
- tool loop owns the semantics and a single event type is used everywhere.

Cons:
- needs a Geppetto event definition; Moments currently uses `mentoevents.NewEventDebuggerPause`.

### B) Tool loop calls a callback (PauseEmitter)

Keep toolhelpers independent by accepting a callback:

```go
type PauseEmitter interface {
    EmitDebuggerPause(ctx context.Context, pauseID string, phase string, summary string, deadlineMs int64, extra map[string]any)
}
```

`session.ToolLoopEngineBuilder` (or Pinocchio webchat) can pass an emitter that publishes to context sinks using whatever event type is desired.

Pros:
- tool loop doesn’t need to import events packages at all.
- can support both Geppetto-native and Moments-native event schemas.

Cons:
- splits the “pause semantics” between loop and caller; easy to forget wiring.

Recommendation for GP-07:
- Prefer **A** long-term (one canonical event), but **B** can be used to ship incrementally.

## “Find the right conversation” / routing `Continue(pause_id)`

Moving StepController out of conversation state forces us to decide how the server finds the paused execution.

There are three patterns:

### Pattern 1: Continue goes through the active ExecutionHandle (Session-owned)

Add to `session.Session` or `ExecutionHandle`:

- `Continue(pauseID string) error`

Webchat continues by locating the conversation to reach its `*session.Session`:

- `conv, ok := convManager.Get(conv_id)`
- `err := conv.Sess.Continue(pause_id)`

Pros:
- no global registry; simple mental model (“continue the active inference for this session”).
Cons:
- requires `conv_id` routing in webchat (still fine in Pinocchio).
- cannot continue without first finding the session.

### Pattern 2: Global PauseRegistry (pause_id → handle)

Maintain a registry (process-wide) keyed by `pause_id`:

```go
type PauseRegistry interface {
    Register(pauseID string, ctrl Controller, meta PauseMeta)
    Continue(pauseID string) bool
}
```

Pause event includes `pause_id` only, and the continue endpoint only needs `pause_id`.

Pros:
- simplest client contract.
- decouples continue from `conv_id`.
Cons:
- must manage lifecycle (cleanup on completion/cancel/disable).
- needs careful security checks: continue endpoint must validate that the caller is allowed to resume that pause (requires meta: owner/session/conv).

### Pattern 3: PauseID encodes session/conv identity

Make pause IDs structured:

- `pause_id = "<session_id>:<uuid>"`

Then the continue endpoint can parse `session_id` from the token and route accordingly.

Pros:
- no registry needed just to route.
Cons:
- still needs some mapping from `session_id` to active handle (session manager) and access control.

Recommendation:

- For Pinocchio webchat, **Pattern 1** is simplest and aligns with existing conv/session ownership checks.
- If we want a future-proof UI protocol (pause_id-only), add **Pattern 2** later (with a strong auth story).

## Proposed wiring sequence (Pinocchio)

1. Pinocchio webchat run start (`/chat`):
   - `conv.Sess.StartInference(...)` starts a run and returns `handle`.
2. If step mode is enabled:
   - configure `ToolLoopEngineBuilder` to pass a `stepcontrol.Controller` into the tool loop.
3. Tool loop hits pause point:
   - emits `debugger.pause` with `(session_id, inference_id, pause_id, phase, deadline_ms, extra...)`
   - blocks on controller wait, cancellation-safe.
4. Continue endpoint:
   - validates ownership (as Moments does)
   - calls `conv.Sess.Continue(pause_id)` (Session delegates to active handle/controller).

## Risks / sharp edges

- **Cancellation semantics**: any pause wait must observe `ctx.Done()`; otherwise `ExecutionHandle.Cancel()` will look “broken”.
- **Multiple pauses per inference**: controller should allow multiple outstanding pauses, but it’s safer to keep “one active pause at a time” (disabling step mode should drain all).
- **Event metadata correctness**: pause events should carry `session_id` and `inference_id` derived from `turn.Metadata` (not separate run fields).
- **Tool loop parity**: ensure phase ordering and “only pause after_inference when tools pending” match Moments to avoid UX surprises.

## Next steps (implementation-oriented)

This doc is design-only. A concrete implementation plan would likely be:

1. Add `geppetto/pkg/inference/stepcontrol` with a cancellation-safe controller.
2. Refactor `toolhelpers.RunToolCallingLoop` into `ToolCallingLoop.RunLoop`.
3. Add a minimal pause emitter (either Geppetto-native event or callback-based).
4. Extend `session.ToolLoopEngineBuilder`/runner to accept and wire step control.
5. Add `Session.Continue(pause_id)` and/or `ExecutionHandle.Continue(pause_id)`.
6. Update Pinocchio forwarder and add webchat endpoints (tasks already tracked in GP-07).
