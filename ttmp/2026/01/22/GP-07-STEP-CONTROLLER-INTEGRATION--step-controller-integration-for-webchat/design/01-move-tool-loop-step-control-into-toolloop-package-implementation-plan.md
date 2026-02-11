---
Title: Move tool loop + step control into toolloop package (implementation plan)
Ticket: GP-07-STEP-CONTROLLER-INTEGRATION
Status: active
Topics:
    - geppetto
    - backend
    - conversation
    - events
    - websocket
    - architecture
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/chat-events.go
      Note: Place to add EventDebuggerPause (new Geppetto-native pause event)
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: Will switch from toolhelpers loop to toolloop package
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Current RunToolCallingLoop source to migrate
    - Path: pinocchio/pkg/webchat/forwarder.go
      Note: Needs SEM mapping for debugger.pause
    - Path: pinocchio/pkg/webchat/router.go
      Note: Needs step-mode enable + continue endpoints wired to shared StepController
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T22:30:41.523116012-05:00
WhatFor: ""
WhenToUse: ""
---


# Move tool loop + step control into `toolloop` package

## Prompt Context

**User prompt (verbatim):**
```
ok, let's create tasks to implement this design. 

Also, I want to move the tool calling loop into its own package, so that we can call the Options WithEngine/WithRegistry/... instead of WithToolCallingEngine/WithToolCallingRegistry/... Move stepcontrol into that package too, since it's an integral part of the tool loop. Use option A for the debugger.pause and we'll move it over in moments later on. 

The Session cannot own the pause controller for all pauses, a session can be paused, but you can't manage all paused sessions through a session. The StepController laready has all the information needed, so to trigger a continue (say through HTTP), we would just need to wire up a handler to the step controller. Does that make sense ?

Create a design document and implementation plan that lays out exactly how to implement this, architecturally and with pseudocode and then laying out each step precisely.
```

**Assistant interpretation:** Produce a concrete design + step-by-step implementation plan for implementing step-mode pause/continue by (1) moving the tool calling loop into a new package with `WithEngine/WithRegistry/...` options, (2) moving step control primitives into that same package, (3) making the tool loop itself publish a Geppetto-native `debugger.pause` event, and (4) wiring HTTP “continue” directly to a shared StepController service (not session-owned, not conversation-owned), then update GP-07 tasks to match.

**Inferred user intent:** Make the architecture cleaner and more reusable: a canonical tool loop package that owns stepping semantics and events, while a web layer can control pausing/continuing without stashing step state in sessions or conversations.

## Summary

This doc proposes and plans a refactor to implement GP-07 step-mode by:

1. Introducing a new package: `geppetto/pkg/inference/toolloop`
2. Moving the tool-calling loop implementation out of `toolhelpers` into `toolloop` as a struct:
   - `toolloop.Loop` (or `toolloop.ToolCallingLoop`) with `RunLoop(ctx, turn)` method
   - functional options: `toolloop.WithEngine`, `toolloop.WithRegistry`, ...
3. Moving step control into that same package as a cancellation-safe service:
   - `toolloop.StepController` manages pauses across many in-flight runs (not tied to one `session.Session`)
4. Making the tool loop publish a Geppetto-native event:
   - `events.EventDebuggerPause` with type `"debugger.pause"` published via `events.PublishEventToContext`
5. Wiring webchat HTTP endpoints directly to the shared `toolloop.StepController` instance:
   - continue handler calls `StepController.Continue(pause_id)` after auth checks

This matches the constraints:

- Step mode is not middleware.
- Step mode is not stored in webchat conversation state.
- Step mode is not session-owned (sessions don’t manage other sessions’ pauses).

## Architecture

### New package: `geppetto/pkg/inference/toolloop`

Responsibilities:

- Execute the canonical tool-calling loop (LLM inference → tool calls → tool results → repeat).
- Define pause points and step-mode semantics (after inference when tools pending; after tools).
- Provide a cancellation-safe step controller primitive used by the tool loop.

Non-responsibilities:

- Webchat routing/auth (Pinocchio/Moments own that).
- Identity/session resolution (apps provide any identity context and set enablement/config).

### Event publication (“Option A”)

The tool loop publishes a Geppetto-native pause event directly (not via callback emitter).

- Add `events.EventDebuggerPause` (type `"debugger.pause"`) to `geppetto/pkg/events`.
- Tool loop publishes:
  - after inference (only when pending tools exist and step mode enabled)
  - after tool results appended (when enabled)

This decouples Pinocchio from Moments’ `mentoevents` and sets a canonical event contract in Geppetto.

### Step controller ownership model

`toolloop.StepController` is a shared service (e.g., owned by a Router/app singleton), not owned by a session or conversation.

Rationale:

- A single process may have many active sessions; pausing is a cross-session coordination problem.
- A session being paused is a property of an in-flight execution, but the control plane (HTTP continue) needs to find it by pause_id.

Mechanism:

- Each pause registers a `pause_id` in `StepController`.
- Continue handler calls `StepController.Continue(pause_id)` directly.
- `StepController` stores pause metadata to allow the handler to authorize the resume.

## Proposed APIs (pseudocode)

### `toolloop.Loop`

```go
package toolloop

type Loop struct {
    eng      engine.Engine
    registry tools.ToolRegistry
    cfg      Config

    executor tools.ToolExecutor
    step     *StepController
    pauseTimeout time.Duration

    snapshotHook toolhelpers.SnapshotHook // optional override (else uses ctx-attached hook)
}

type Option func(*Loop)

func New(opts ...Option) *Loop

func WithEngine(eng engine.Engine) Option
func WithRegistry(reg tools.ToolRegistry) Option
func WithConfig(cfg Config) Option
func WithExecutor(exec tools.ToolExecutor) Option
func WithStepController(sc *StepController) Option
func WithPauseTimeout(d time.Duration) Option
func WithSnapshotHook(h toolhelpers.SnapshotHook) Option

func (l *Loop) RunLoop(ctx context.Context, initial *turns.Turn) (*turns.Turn, error)
```

### `toolloop.StepController`

We need:

- enable/disable step mode per “scope” (minimum: per session_id; optionally per conversation_id)
- pause creation that returns `pause_id` + `deadline_ms`
- wait that is cancellation-safe (`select` on ctx.Done)
- continue by pause_id
- metadata per pause to support authorization (owner/session/conv)

```go
package toolloop

type StepScope struct {
    SessionID string
    ConversationID string
    Owner string
}

type PauseMeta struct {
    PauseID     string
    Phase       string // "after_inference" | "after_tools"
    Summary     string
    DeadlineMs  int64

    SessionID string
    InferenceID string
    TurnID string
    Scope StepScope

    Extra map[string]any
}

type StepController struct {
    enabled map[string]StepScope      // keyed by session id
    waiters map[string]*pauseWaiter   // keyed by pause_id
}

func NewStepController() *StepController

func (s *StepController) Enable(scope StepScope)
func (s *StepController) DisableSession(sessionID string)
func (s *StepController) IsEnabled(sessionID string) (StepScope, bool)

func (s *StepController) Pause(ctx context.Context, meta PauseMeta) (PauseMeta, bool)
func (s *StepController) Wait(ctx context.Context, pauseID string, timeout time.Duration) error
func (s *StepController) Continue(pauseID string) (PauseMeta, bool)
```

Semantic requirements:

- `Wait` must unblock on:
  - `Continue(pauseID)`
  - timeout (auto-continue)
  - `ctx.Done()` (cancel)
  - disable (drain)

### `events.EventDebuggerPause`

```go
package events

const EventTypeDebuggerPause EventType = "debugger.pause"

type EventDebuggerPause struct {
    EventImpl
    PauseID    string         `json:"pause_id"`
    Phase      string         `json:"phase"`
    Summary    string         `json:"summary"`
    DeadlineMs int64          `json:"deadline_ms"`
    Extra      map[string]any `json:"extra,omitempty"`
}
```

Event metadata must include:

- `session_id`
- `inference_id`
- `turn_id`

## How `RunLoop` integrates stepping (pseudocode)

```go
for i := 0; i < cfg.MaxIterations; i++ {
    snapshot("pre_inference")
    updated, err := eng.RunInference(ctx, t)
    snapshot("post_inference")

    pending := toolblocks.ExtractPendingToolCalls(updated)
    if len(pending) > 0 && stepEnabledForTurn(updated) {
        pm := buildPauseMeta(updated, "after_inference", "Review next action", map[string]any{
            "pending_tools": len(pending),
        })
        pm, _ = step.Pause(ctx, pm)
        events.PublishEventToContext(ctx, &events.EventDebuggerPause{...})
        _ = step.Wait(ctx, pm.PauseID, pauseTimeout)
    }
    if len(pending) == 0 {
        return updated, nil
    }

    // execute tools + append results
    ...
    snapshot("post_tools")

    if stepEnabledForTurn(updated) {
        pm := buildPauseMeta(updated, "after_tools", "Review tool results", nil)
        pm, _ = step.Pause(ctx, pm)
        events.PublishEventToContext(ctx, &events.EventDebuggerPause{...})
        _ = step.Wait(ctx, pm.PauseID, pauseTimeout)
    }

    t = updated
}
```

`stepEnabledForTurn` uses `turn.Metadata` `session_id` to look up enablement in `StepController`.

## Pinocchio webchat wiring (high-level)

### Where the shared StepController lives

- Add a field on `pinocchio/pkg/webchat.Router`:
  - `stepCtrl *toolloop.StepController`

This is router-owned (not conversation-owned, not session-owned).

### Enabling step mode

- At run start (`POST /chat`), if overrides request step mode:
  - call `Router.stepCtrl.Enable(StepScope{SessionID: conv.RunID, ConversationID: conv.ID, Owner: userID})`

### Continuing

- `POST /debug/continue` handler:
  - validate auth using returned `PauseMeta` (Owner / ConversationID / SessionID)
  - call `Router.stepCtrl.Continue(pause_id)`

No need to find the session/handle first.

## Implementation plan (precise steps)

1. **Create package skeleton**
   - Add `geppetto/pkg/inference/toolloop/` with:
     - `loop.go` (Loop struct + options + RunLoop)
     - `config.go` (Config derived from existing `toolhelpers.ToolConfig`)
     - `step_controller.go` (ctx-aware StepController)
2. **Move tool loop implementation**
   - Port logic from `toolhelpers.RunToolCallingLoop` into `toolloop.(*Loop).RunLoop`.
   - Decide what stays in `toolhelpers` (e.g., SnapshotHook context helpers) vs what moves.
3. **Add pause event type (Option A)**
   - Add `events.EventDebuggerPause`.
   - Publish it from `toolloop.RunLoop` at pause points.
4. **Wire tool loop into session runner**
   - Update `session/tool_loop_builder.go` runner path:
     - replace `toolhelpers.RunToolCallingLoop(...)` with `toolloop.New(...).RunLoop(...)`.
5. **Add StepController wiring point**
   - Provide the `*toolloop.StepController` to the runner (breaking changes OK):
     - either via builder field, or via context attachment helper in `toolloop`.
6. **Pinocchio forwarder support**
   - Extend `pinocchio/pkg/webchat/forwarder.go` to map `"debugger.pause"` to SEM frames.
7. **Pinocchio endpoints**
   - Add dev-gated endpoints:
     - enable/disable step mode (by conv/session)
     - continue by `pause_id`
   - Continue handler calls `Router.stepCtrl.Continue(pause_id)` (not session).
8. **Tests**
   - Unit tests in `geppetto/pkg/inference/toolloop` for:
     - wait unblocks on continue
     - wait unblocks on ctx cancel
     - disable drains waiters
   - Unit tests for loop pause points (fake engine returning tool calls).
9. **Moments migration (deferred)**
   - Later: replace `moments/backend/pkg/webchat/loops.go` with the new package and Geppetto pause event.
