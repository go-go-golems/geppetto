---
Title: Step controller integration (from go-go-mento to pinocchio)
Ticket: MO-001-PORT-MOMENTS-WEBCHAT
Status: active
Topics:
    - webchat
    - moments
    - session-refactor
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Candidate for hook + ToolExecutor injection
    - Path: go-go-mento/go/pkg/webchat/handlers/debugger.go
      Note: Reference SEM mapping for debug.pause
    - Path: go-go-mento/go/pkg/webchat/loops.go
      Note: Reference pause points + pause event emission
    - Path: go-go-mento/go/pkg/webchat/router.go
      Note: Reference debug.continue and step-mode endpoints
    - Path: go-go-mento/go/pkg/webchat/step_controller.go
      Note: Reference StepController implementation
    - Path: pinocchio/pkg/webchat/forwarder.go
      Note: Pinocchio SEM mapping; add debug.pause mapping
    - Path: pinocchio/pkg/webchat/router.go
      Note: Pinocchio webchat endpoints to extend
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T11:18:09.531896279-05:00
WhatFor: ""
WhenToUse: ""
---


# Step controller integration (from go-go-mento to pinocchio)

## Executive Summary

go-go-mento’s webchat has a “step mode” that lets operators pause an agentic run
at well-defined points (after inference, after tool execution), inspect what the
agent is about to do, and then explicitly continue.

We want to port this capability into Pinocchio’s webchat as a separable feature
that does **not** require DB persistence and does not undermine MO-007’s
Session/ExecutionHandle model.

The proposed integration consists of:

- A small `StepController` state machine per conversation/session.
- A “pause protocol” expressed as semantic events (`debug.pause`) over the
  existing event/SEM streaming path.
- One or two HTTP endpoints to continue and to toggle step mode.
- Hook points in the tool loop (or a thin wrapper around it) that can:
  - decide whether to pause,
  - emit a pause event,
  - block until continued (or timeout auto-continues).

## Problem Statement

Pinocchio’s webchat currently supports:

- Multi-provider inference via Geppetto,
- Tool calling via `session.ToolLoopEngineBuilder` / `toolhelpers.RunToolCallingLoop`,
- Streaming UI updates over websockets.

What it lacks is “operator control” over the tool loop:

- No way to pause between “LLM proposes tool calls” and “tools execute”.
- No way to inspect the tool calls and approve/deny/continue.
- No way to pause after tools to inspect results before letting the LLM continue.

go-go-mento already implemented an MVP step controller that:

- Maintains per-pause waiters,
- Emits a pause event with a deadline,
- Provides HTTP endpoints to continue and enable/disable step mode,
- Auto-continues after a timeout to avoid wedging.

References:

- `go-go-mento/go/pkg/webchat/step_controller.go`
- `go-go-mento/go/pkg/webchat/router.go` (`debug.continue`, `step-mode` handlers)
- `go-go-mento/go/pkg/webchat/loops.go` (pause points + event emission)

## Proposed Solution

### 1) Port the StepController primitive (mostly as-is)

The existing StepController is intentionally tiny:

- `Enable()` / `Disable()` / `IsEnabled()`
- `Pause(pauseID) -> deadline`
- `Wait(pauseID, timeout)` (auto-continues)
- `Continue(pauseID)`

Key semantics:

- Disabling step mode should drain any pending waits (otherwise a running
  inference can wedge forever).
- Wait should always have a timeout; step mode is a debugging tool, not a new
  correctness-critical protocol.

In Pinocchio, this should live alongside webchat state (likely in a manager
object), not inside Geppetto’s core session package.

### 2) Define a “pause protocol” over events (SEM)

We treat a pause as an observable event, not an out-of-band websocket message.

Event payload (conceptual):

- `pause_id` (string, opaque)
- `at` (phase string, e.g. `after_inference`, `after_tools`)
- `summary` (human readable)
- `deadline_ts` (ms since epoch)
- `meta` (optional object; e.g. list of tool names)

In go-go-mento, the event is `mentoevents.EventDebuggerPause`, and the SEM
translation produces:

```json
{"sem": true, "event": {"type":"debug.pause","id":"<pause_id>", "at":"…", "deadlineTs":…, "meta":{...}}}
```

Pinocchio does not need the moments-specific event type. We can:

1) Add a small “debug pause” event type to a Pinocchio-local package, or
2) Encode pause as a standard `events.EventInfo` with structured fields and have
   the forwarder translate it, or
3) Add a generic `events.EventDebugPause` to Geppetto if we want it shared.

Given we want to tackle this separately and keep it optional, (2) is the lowest
friction: use `EventInfo` or `EventLog` with a dedicated message key and fields.

### 3) Add step-mode HTTP endpoints (continue + toggle)

Minimal endpoints matching go-go-mento’s semantics:

- `POST /api/debug/continue` with `{ "conv_id": "...", "pause_id": "..." }`
- `POST /api/debug/step-mode` with `{ "conv_id": "...", "enabled": true|false }`

In Pinocchio, we don’t currently have identity/ownership gating like moments.
So by default:

- No authorization gating (local dev / single-user assumption), or
- Gate behind an env flag, or
- Gate behind a shared secret header if needed.

We can add ownership later if/when pinocchio webchat gains identity.

### 4) Add pause hook points to the tool loop

The key architectural choice is *where* to implement “pause points”.

Option A (minimal diff, pinocchio-local):

- Keep using `session.ToolLoopEngineBuilder`, but wrap the underlying call to
  `toolhelpers.RunToolCallingLoop` with a pinocchio-specific runner that:
  - runs one engine step,
  - pauses if tool calls are pending,
  - executes tools,
  - pauses after tools,
  - loops.

This mirrors go-go-mento’s `ToolCallingLoop` and keeps the step mode concerns
out of Geppetto core.

Option B (shared improvement):

- Extend `toolhelpers.RunToolCallingLoop` (or create a new variant) to accept a
  “hook interface” with phase callbacks:

```go
type LoopHook interface {
    OnPhase(ctx context.Context, t *turns.Turn, phase string, data map[string]any) error
}
```

Then pinocchio can provide a hook implementation that:

- checks if step mode enabled,
- emits debug.pause event,
- blocks until continued.

Because you also want toolhelpers to accept a pluggable `ToolExecutor`, Option B
is attractive: both “authorized tools” and “step pausing” become configurable
without forking the loop.

### 5) Ensure middleware list is applied in reverse order

Step mode becomes much easier to reason about if the middleware stack order is
consistent between moments and pinocchio.

As decided, Pinocchio should apply middleware lists in reverse so that
`[A, B, C]` reads as “A wraps B wraps C wraps base”.

## Design Decisions

### Decision: Auto-continue timeout is required

Rationale:

- Step mode is an operator tool; it must not wedge the system permanently.
- Timeouts provide a “fail open” path.

### Decision: Pause protocol uses normal event/SEM path

Rationale:

- The UI already consumes SEM events.
- We keep one transport rather than introducing a parallel websocket message schema.

### Decision: No DB persistence integration (yet)

Rationale:

- The immediate goal is live debugging, not replay.
- It keeps the step controller implementation small and deployable.

### Decision: Prefer extending Geppetto toolhelpers (hooks + ToolExecutor)

Rationale:

- Avoids divergence: otherwise moments and pinocchio keep separate loop implementations.
- Makes step mode a “pluggable capability” rather than a fork.

## Alternatives Considered

### Alternative: Implement step mode purely in the UI (client-side pause)

Rejected because:

- Tool execution happens server-side; the server must gate it.
- Client-only pausing doesn’t prevent tools from executing.

### Alternative: Pause by blocking the websocket reader

Rejected because:

- It couples control flow to transport.
- It breaks multi-connection scenarios and reconnect behavior.

### Alternative: Add DB-backed “pause state” persistence now

Rejected for this phase: it increases scope and requires a correctness model for
resume/retry that we don’t need yet.

## Implementation Plan

1) Add a `StepController` implementation to Pinocchio webchat (ported from moments).
2) Add endpoints:
   - `/api/debug/continue`
   - `/api/debug/step-mode`
3) Define the pause SEM event mapping:
   - either via a dedicated event type, or via `EventInfo` fields.
4) Add tool loop hook points:
   - Prefer: extend `toolhelpers.RunToolCallingLoop` to accept:
     - `tools.ToolExecutor` (for moments/authorized tools)
     - a hook interface (for step pauses)
5) Add a pinocchio-level wiring layer:
   - per conversation/session, store StepController and enable/disable state.
6) (Optional) Add a minimal UI affordance:
   - show pause cards with “continue” action that calls the endpoint.

## Open Questions

1) What are the exact pause phases we want to support?
   - `after_inference` and `after_tools` are the MVP.
2) Should step mode be per conversation, per profile, or global?
3) What is the minimum viable authorization story for “continue”?
   - none (dev), env flag, or shared secret header.

## References

- go-go-mento step controller + pause emission:
  - `go-go-mento/go/pkg/webchat/step_controller.go`
  - `go-go-mento/go/pkg/webchat/loops.go`
  - `go-go-mento/go/pkg/webchat/router.go`
- go-go-mento SEM pause mapping:
  - `go-go-mento/go/pkg/webchat/handlers/debugger.go`
- Pinocchio webchat baseline:
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/forwarder.go`
- Geppetto tool loop (candidate for hook/executor injection):
  - `geppetto/pkg/inference/toolhelpers/helpers.go`
