---
Title: StepController deep dive + session integration
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
      Note: ExecutionHandle cancel/wait model (pause must observe ctx cancellation)
    - Path: geppetto/pkg/inference/session/session.go
      Note: Current Session API that stepping must integrate into
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: ToolLoopEngineBuilder wiring point for step gating
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Canonical tool loop phases where step gating can hook
    - Path: geppetto/ttmp/2026/01/21/MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/design-doc/01-session-refactor-sessionid-enginebuilder-executionhandle.md
      Note: Design rationale for MO-007 (no backwards-compat assumption aligns with this ticket)
    - Path: geppetto/ttmp/2026/01/21/MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/reference/01-diary.md
      Note: Background on the session/execution model we’re integrating stepping into
    - Path: moments/backend/pkg/mentoevents/events.go
      Note: Defines debugger.pause event schema used by Moments
    - Path: moments/backend/pkg/webchat/loops.go
      Note: Shows pause points (after_inference/after_tools) and pause event emission
    - Path: moments/backend/pkg/webchat/step_controller.go
      Note: Source-of-truth for Moments StepController primitive
    - Path: pinocchio/pkg/webchat/forwarder.go
      Note: Event->SEM mapping (must add pause event mapping)
    - Path: pinocchio/pkg/webchat/router.go
      Note: Webchat entry point that starts inference and would host debug endpoints
ExternalSources: []
Summary: "Deep dive into Moments StepController semantics and a proposed (breaking-change-friendly) design to integrate stepping/pause/continue into Geppetto session.Session and the canonical tool loop, including Pinocchio webchat event forwarding considerations."
LastUpdated: 2026-01-22T17:50:37.225000076-05:00
WhatFor: Explain how StepController works in moments/backend today and propose a concrete, breaking-change-friendly design for integrating stepping into the current Geppetto session.Session + tool loop architecture (as used by Pinocchio webchat).
WhenToUse: Use when implementing step-mode/pause/continue in Pinocchio webchat and when deciding where stepping should live in the Geppetto session abstraction (Session vs ExecutionHandle vs tool loop).
---



# StepController deep dive + session integration

## Executive summary

`moments/backend` implements “step mode” using a small **StepController** primitive plus two **pause points** in the tool loop:

- Pause **after inference** when there are pending tool calls (“review next action”).
- Pause **after tool execution** (“review tool results”).

Each pause emits a `debugger.pause` event with a fresh `pause_id`, and the server blocks until either:
- the UI calls a “continue” endpoint with that `pause_id`, or
- a fixed timeout elapses (30s), at which point the server auto-continues.

Pinocchio webchat now runs on the MO-007 stack (`geppetto/pkg/inference/session`). The most direct way to backport step-mode behavior is to:

1) **Add a StepController concept to the session execution model**, ideally scoped to the active inference (`ExecutionHandle`) but reachable through `session.Session` for webchat handlers.
2) **Insert stepping at tool-loop phase boundaries** (`post_inference` and `post_tools`) and emit a pause event into existing context sinks.

No backwards-compatibility constraints mean we can change public APIs (e.g., `StartInference` options), adjust hook signatures, and update Pinocchio’s event forwarder.

## Goals / non-goals

### Goals

- Provide “step mode” semantics in Pinocchio webchat equivalent to Moments:
  - deterministic pause points aligned with the tool loop
  - server-side block until “continue”
  - a pause event contract that the UI can observe
  - safe behavior under cancellation (no deadlocks / no 30s “stuck pause” on cancel)
- Integrate stepping into the **current** session abstraction (`session.Session` + `ExecutionHandle`) rather than ad-hoc webchat-only code.
- Allow breaking API changes to keep the design clean.

### Non-goals

- Implementing the full UI/UX for step mode here (this doc defines the contract + integration plan).
- Preserving legacy names/fields like `run_id` vs `session_id` (we can update consumers).

## How StepController works in Moments (today)

### The primitive

In `moments/backend/pkg/webchat/step_controller.go`, StepController is a mutex-protected gate:

- State:
  - `enabled bool`
  - `waiters map[pauseID]chan struct{}`
- Operations:
  - `Enable()` / `Disable()` (disable drains/continues all waiters)
  - `Pause(pauseID) -> deadline` (registers a waiter, returns `now+30s`)
  - `Wait(pauseID, timeout)` (blocks until continue or timeout; on timeout it calls `Continue`)
  - `Continue(pauseID)` (closes the waiter channel)

Notably:
- It does **not** accept `context.Context`; cancellation is not observed except by timeout.
- It does **not** enforce ownership/authorization; that happens in HTTP handlers.

### Where it is invoked (pause points)

In `moments/backend/pkg/webchat/loops.go`, StepController is used in the tool loop:

1) After inference (only if pending tool calls exist):
   - `pause_id := uuid.NewString()`
   - `deadline := StepCtrl.Pause(pause_id)`
   - emit `debugger.pause` with `at="after_inference"` and meta (e.g., `pending_tools`)
   - `StepCtrl.Wait(pause_id, 30s)`

2) After tools (always, as long as step mode enabled):
   - same pattern with `at="after_tools"`

### Event contract (“debugger.pause”)

In `moments/backend/pkg/mentoevents/events.go` the pause event is:

- Type: `"debugger.pause"`
- Fields:
  - `pause_id` (string)
  - `at` (string)
  - `summary` (string)
  - `deadline_ts` (ms since epoch)
  - `meta` (object)

### How the UI resumes execution

In `moments/backend/pkg/webchat/router.go`:

- A debug “continue” endpoint receives `{conv_id, pause_id}`.
- The handler checks:
  - there is an identity session
  - the user matches `conv.OwnerUserID`
- Then it calls `conv.StepCtrl.Continue(pause_id)`.

Step mode is enabled either via a debug step-mode endpoint or via a run request override (`overrides.step_mode=true`).

## What changes in Pinocchio/Geppetto post-MO-007

### Current execution model (Geppetto)

MO-007 introduces:

- `session.Session`:
  - owns `SessionID` and turn history (`Turns []*turns.Turn`)
  - enforces “only one active inference”
  - starts inference via `StartInference(ctx) -> *ExecutionHandle`
- `ExecutionHandle`:
  - `Cancel()` + `Wait()`
  - cancellation is via context (`context.WithCancel`)
- `ToolLoopEngineBuilder`:
  - injects sinks (`events.WithEventSinks`)
  - injects snapshot hook (`toolhelpers.WithTurnSnapshotHook`)
  - runs either a single inference or `toolhelpers.RunToolCallingLoop`

### Canonical tool loop phases (Geppetto)

`geppetto/pkg/inference/toolhelpers/RunToolCallingLoop` already defines stable “hookable” phases:

- `pre_inference`
- `post_inference`
- `post_tools`

This maps cleanly to Moments:

- Moments `after_inference` ~= Geppetto `post_inference` (and can be conditioned on `pending_tools > 0`)
- Moments `after_tools` ~= Geppetto `post_tools`

### Pinocchio webchat uses session.Session directly

`pinocchio/pkg/webchat/router.go` appends a seed turn and calls `conv.Sess.StartInference(...)` with a `ToolLoopEngineBuilder` configured for sinks, registry, tool config, and snapshot hook.

That makes `session.Session` the natural integration point for step control (there is no separate “conversation runner” abstraction anymore).

## Integration design options

Because we do not require backwards compatibility, the primary decision is “where should stepping live”:

### Option A: Implement stepping as a blocking SnapshotHook (minimal code motion)

Approach:
- Keep toolhelpers unchanged.
- Implement StepController in the session layer and provide a wrapper snapshot hook:
  - call the original snapshot hook (for persistence/debug snapshots)
  - additionally, at `post_inference` and `post_tools`, emit `debugger.pause` and block on StepController.

Pros:
- No import-cycle risk (session already depends on toolhelpers; toolhelpers doesn’t need to depend on session).
- No need to touch provider engines or middleware.
- Easy to prototype and backport quickly.

Cons:
- SnapshotHook’s name/intent becomes overloaded (it becomes both “snapshot” and “execution gating”).
- Step behavior becomes “opt-in” via “did we remember to attach the hook”.
- Single-pass inference runs (Registry==nil) won’t naturally hit tool-loop phases unless we add more hook calls.

### Option B: Make step gating a first-class tool loop feature (cleanest semantics)

Approach:
- Add an explicit “step gate” hook to `toolhelpers.RunToolCallingLoop` (and optionally to single-pass inference runner).
- Define a small interface in a package that does not create cycles (e.g., `geppetto/pkg/inference/stepcontrol`):
  - context attach/get helpers
  - a controller interface/type used by toolhelpers
- `session.Session` (or `ToolLoopEngineBuilder`) attaches the step gate to the run context.

Pros:
- Clear semantics: tool loop *knows* it can pause, and it pauses consistently.
- Avoids abusing SnapshotHook for gating.
- Easier to test at the tool-loop level (unit tests for pause/resume/cancel).

Cons:
- Requires modifying geppetto core packages and updating call sites/tests.

### Option C: Re-implement the tool loop inside session (closest to Moments structure, highest churn)

Approach:
- Replace `toolhelpers.RunToolCallingLoop` usage with a session-owned tool loop that matches Moments.

Pros:
- Maximum control / parity.

Cons:
- Duplicates logic and risks drift vs Geppetto’s canonical tool loop.
- Biggest code motion for the least incremental benefit.

## Recommended approach

Prefer **Option B** (first-class step gating) unless we need a very fast prototype, in which case Option A can bootstrap the UI contract.

The key additional improvement vs Moments is: **Wait must observe context cancellation** (so `handle.Cancel()` can reliably unblock a paused run immediately).

## Proposed API and contracts (breaking changes allowed)

### 1) Define a controller interface that can be driven by webchat

Core behaviors needed:

- Enable/disable stepping
- Pause at a named “phase” with metadata, returning a `pause_id` and an expected deadline for the UI
- Wait for continue, but stop waiting if context is canceled
- Continue by `pause_id`

Suggested interface (package name illustrative):

```go
// package stepcontrol
type Phase string

const (
    PhaseAfterInference Phase = "after_inference" // or align with toolhelpers phase names
    PhaseAfterTools     Phase = "after_tools"
)

type Pause struct {
    PauseID    string
    Phase      Phase
    Summary    string
    DeadlineMs int64
    Meta       map[string]any
}

type Controller interface {
    Enabled() bool
    Disable()                 // must unblock any waiters
    Continue(pauseID string)  // idempotent

    // MaybePause emits an event (via ctx sinks) and blocks until continued/canceled/timeout.
    MaybePause(ctx context.Context, p Pause) error
}
```

Notes:
- `MaybePause` being “event emission + wait” keeps the call sites simple.
- The controller can internally generate `pause_id` to avoid requiring the caller to do so.

### 2) Integrate into session.Session / ExecutionHandle

Key design choice: stepping is *execution-level*, but webchat needs to reach it.

Suggested session shape:

- `session.Session` owns a `StepController` (or a factory/config used to create per-execution controllers).
- `ExecutionHandle` exposes step control for the active inference.
- `session.Session` exposes a safe delegation method for webchat handlers:
  - `Session.Continue(pauseID string) error` (delegates to active execution controller)

Because we can break APIs, we can redesign `StartInference` to accept options:

```go
type StartOptions struct {
    StepController stepcontrol.Controller // optional
}

func (s *Session) StartInference(ctx context.Context, opts StartOptions) (*ExecutionHandle, error)
```

or:

```go
func (s *Session) StartInference(ctx context.Context) (*ExecutionHandle, error)
func (s *Session) EnableStepping()
func (s *Session) DisableStepping()
func (s *Session) Continue(pauseID string) error
```

The options-based design is clearer for “step mode is per run”.

### 3) Hook step gating into the tool loop phases

If using Option B:
- `toolhelpers.RunToolCallingLoop` calls `stepcontrol.MaybePause(...)` at:
  - `post_inference` (conditioned on pending tool calls)
  - `post_tools`

If using Option A:
- `ToolLoopEngineBuilder` (or webchat) injects a snapshot hook that:
  - delegates to the original hook
  - additionally runs the pause+wait logic

### 4) Event: represent pauses in a way Pinocchio will forward

Pinocchio’s `SemanticEventsFromEvent` currently only handles known Go event types and drops unknown events.

Therefore, a pause event must be one of:

- a new typed Geppetto event (recommended), e.g. `events.EventDebuggerPause`, or
- a generic fallback path in the forwarder (“if e.Type() == 'debugger.pause' then …”).

To stay close to Moments, reuse:

- Event type string: `"debugger.pause"`
- Payload fields: `pause_id`, `at`/`phase`, `summary`, `deadline_ts`, `meta`

Then map it to a SEM frame such as:

```json
{ "sem": true, "event": { "type": "debugger.pause", "id": "<pause_id>", "at": "after_tools", "deadlineTs": 123, "meta": {...} } }
```

## Webchat integration notes (Pinocchio)

Pinocchio webchat will need:

- A way to enable step mode for a run (`overrides.step_mode=true` is one viable entry point).
- A way to continue:
  - `POST /debug/continue { conv_id, pause_id }` (Moments-style), calling `conv.Sess.Continue(pause_id)`.
- Gating/authorization:
  - Pinocchio doesn’t appear to have identity middleware like Moments; for dev, this can be gated by an env var or restricted to localhost.

## Risks / sharp edges

- **Cancellation while paused**: if the pause wait does not observe `ctx.Done()`, `Cancel()` will not stop a paused run promptly (Moments currently relies on timeout).
- **Event forwarding**: without a forwarder mapping, pause events will be dropped and step mode will appear “broken”.
- **Phase definition drift**: if we rely on string phase names, it’s easy to regress pause placement. Prefer typed constants.
- **Multiple pauses**: session only allows one active inference, but the controller should still be robust if multiple pause IDs are registered (idempotent continues, disable drains).

## Concrete implementation plan (for follow-up work)

1) Add a step control package (or session-local type if using Option A).
2) Add unit tests:
   - pause emits an event and blocks until continue
   - cancel unblocks immediately
   - disable drains and unblocks
3) Wire step gating into tool loop phases (Option B preferred).
4) Add a typed pause event (or forwarder fallback) and update Pinocchio forwarder mapping.
5) Add webchat endpoints for enable/continue (dev-gated).

## Open questions

- Should step mode be per-session (sticky) or per-inference (options)?
- Should the pause boundary include more structured metadata (tool names/args summaries) to allow better UI review?
- Do we need additional pause points (e.g., before inference, after final answer) for non-tool runs?
