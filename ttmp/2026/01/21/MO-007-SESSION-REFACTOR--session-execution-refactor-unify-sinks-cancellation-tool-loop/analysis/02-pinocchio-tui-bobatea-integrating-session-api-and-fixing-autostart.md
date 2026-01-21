---
Title: 'Pinocchio TUI + bobatea: integrating Session API and fixing autostart'
Ticket: MO-007-SESSION-REFACTOR
Status: active
Topics:
    - inference
    - architecture
    - events
    - webchat
    - tui
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../bobatea/pkg/chat/backend.go
      Note: Backend.Start requires prompt string
    - Path: ../../../../../../../bobatea/pkg/chat/model.go
      Note: StartBackendMsg/startBackend() path and the no-op backend start
    - Path: ../../../../../../../pinocchio/pkg/cmds/cmd.go
      Note: Pinocchio auto-submits via ReplaceInputTextMsg + SubmitMessageMsg
    - Path: ../../../../../../../pinocchio/pkg/ui/backend.go
      Note: Pinocchio EngineBackend implements chat.Backend using session.Session
    - Path: pkg/inference/session/session.go
      Note: Session lifecycle and StartInference/ExecutionHandle
    - Path: pkg/inference/session/tool_loop_builder.go
      Note: ToolLoopEngineBuilder wiring
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-21T15:16:54.620909714-05:00
WhatFor: ""
WhenToUse: ""
---


# Pinocchio TUI + bobatea: integrating Session API and fixing autostart

## Goal

Explain:

1) why `bobatea`’s `startBackend()` is effectively a no-op (and why that caused the “stuck generating” symptom),
2) how the pinocchio TUI pipeline is wired end-to-end with the **new Session API**,
3) how we should cut over cleanly (no backwards compatibility) so that:
   - the only way an inference starts is via `Backend.Start(ctx, prompt)`,
   - pinocchio “start in chat mode” remains supported,
   - bobatea doesn’t contain lifecycle knobs that don’t actually start inference.

This document intentionally focuses on pinocchio TUI (Bubble Tea) and the bobatea chat widget; webchat is out of scope except where shared primitives matter.

## Key context: bobatea’s `Backend` contract

The chat widget’s backend contract is explicit:

```go
type Backend interface {
    Start(ctx context.Context, prompt string) (tea.Cmd, error)
    Interrupt()
    Kill()
    IsFinished() bool
}
```

`Start(ctx, prompt)` is the *only* backend entrypoint that has enough information to run a prompt-based inference, because it includes the user prompt string.

## Why `startBackend()` is a no-op (and why it hung)

### The control message

`bobatea/pkg/chat/user_messages.go` defines a `StartBackendMsg`.

When the chat model is constructed with `WithAutoStartBackend(true)`, `Init()` enqueues `StartBackendMsg`:

```go
if m.autoStartBackend {
    cmds = append(cmds, func() tea.Msg { return StartBackendMsg{} })
}
```

### The handler

In `bobatea/pkg/chat/model.go`, handling `StartBackendMsg` calls `startBackend()`:

```go
case StartBackendMsg:
    return m, m.startBackend()
```

### What `startBackend()` does today

`startBackend()` transitions the UI into “streaming” state (`StateStreamCompletion`) and returns a `tea.Batch(refreshCmd, backendCmd)`.

But **the backend command is explicitly a no-op**:

```go
backendCmd := func() tea.Msg {
    log.Debug().Msg("BACKEND START COMMAND EXECUTING (no-op in new prompt flow)")
    return nil
}
```

This is the core mismatch:

- `StartBackendMsg` *looks* like “start inference”
- but it doesn’t call `Backend.Start(...)` (because it has no prompt string to pass)

### Why that caused “stuck generating”

The submit path (`SubmitMessageMsg` → `submit()`):

- checks `m.backend.IsFinished()`, and refuses to submit while streaming,
- transitions UI state to streaming and then calls `Backend.Start(ctx, userMessage)` inside `backendCmd`,
- emits `BackendFinishedMsg` when the backend completes, which calls `finishCompletion()` to unblur input and return to `StateUserInput`.

`StartBackendMsg` bypasses *all* of that. It sets the UI into `StateStreamCompletion` without starting inference and without any guaranteed `BackendFinishedMsg`. That leaves the UI in “generating” forever, and normal submit is blocked as “already streaming”.

### Why it was implemented this way

This looks like a partial refactor: previously, the model likely had a “prompt-driven start” path that could start without a submit (e.g., conversation-seeded inference or a pre-attached prompt), but the current backend API requires a string prompt. During “new prompt flow” changes, `StartBackendMsg` was retained for UI state orchestration but the actual inference start moved exclusively into `submit()`.

From a product behavior standpoint: starting inference should be a submit, not a UI state toggle. So the current “no-op” implementation is consistent with “submit owns start”, but the presence of `WithAutoStartBackend(true)` creates a misleading integration hook.

## The pinocchio TUI pipeline with the new Session API (end-to-end)

This is the path that now works and should be the only supported path going forward.

### Components (who owns what)

- **pinocchio CLI command**:
  - builds router + sinks,
  - constructs the Bubble Tea program and chat model,
  - sends initial input manipulation messages (prefill prompt),
  - sends `SubmitMessageMsg` when it wants an automatic first inference.

- **bobatea chat model**:
  - owns UI state machine (`StateUserInput` ↔ `StateStreamCompletion`),
  - handles submit, and calls `Backend.Start(ctx, userMessage)` to actually start inference.

- **pinocchio EngineBackend** (implements `bobatea/pkg/chat.Backend`):
  - owns `session.Session` and an `EngineBuilder` (usually `session.ToolLoopEngineBuilder`),
  - starts inference using the new Session API and returns a `tea.Cmd` that blocks on `ExecutionHandle.Wait()`,
  - listens to event router “ui” events and translates them into timeline entity updates.

- **geppetto session + tool loop**:
  - `Session` enforces “one active inference per session” and manages turn list,
  - `ToolLoopEngineBuilder` composes base provider engine + middleware and runs the tool calling loop,
  - event sinks are attached to context (not to the engine).

### Sequence diagram (user submit)

```text
User presses TAB
  ↓
bobatea chat model: submit()
  - create user message entity (timeline)
  - set StateStreamCompletion
  - backendCmd calls Backend.Start(ctx, prompt)
  ↓
pinocchio EngineBackend.Start(ctx, prompt)
  - clone seed turn from Session.Latest()
  - append user prompt block(s)
  - Session.Append(seedTurn)
  - Session.Builder = ToolLoopEngineBuilder{Base, Middleware, Registry, ...}
  - handle := Session.StartInference(ctx)         // async
  - return tea.Cmd: func() tea.Msg { handle.Wait(); return BackendFinishedMsg{} }
  ↓
geppetto runner (in goroutine)
  - ToolLoopEngineBuilder.RunInference(ctx, seedTurn)
  - attach sinks: ctx = events.WithEventSinks(ctx, uiSink, ...)
  - run tool loop or single inference
  - provider emits events → PublishEventToContext → sink publishes to router topic "ui"
  ↓
pinocchio EventRouter handler delivers ui events to EngineBackend.EventHandler()
  - EngineBackend translates to timeline events:
    - UIEntityCreated for assistant message
    - UIEntityUpdated for partial content
    - UIEntityCompleted on final
  ↓
bobatea chat model consumes timeline events, updates view
  ↓
ExecutionHandle.Wait returns
  ↓
EngineBackend returns BackendFinishedMsg
  ↓
bobatea chat model finishCompletion() unblurs input, StateUserInput
```

### Key invariants

- **All inference start must happen in `Backend.Start(ctx, prompt)`**.
- **Only submit starts inference** (submit owns prompt text; “start backend” does not).
- **The UI should never enter StateStreamCompletion without an in-flight inference**.

## What pinocchio should do (best integration plan; cutover, no compatibility)

### 1) Remove `WithAutoStartBackend` from pinocchio usage (already done)

Pinocchio should always pass `WithAutoStartBackend(false)` and trigger “auto-start” by sending:

- `ReplaceInputTextMsg{Text: <rendered prompt>}`
- `SubmitMessageMsg{}`

This is both more explicit and consistent with submit semantics.

### 2) Decide what to do with bobatea’s autostart API

Because `WithAutoStartBackend(true)` currently produces a broken state, we should cut over by deleting it (no compatibility) or redefining it.

Preferred options:

**Option A (recommended): delete it**

- Remove `StartBackendMsg` entirely.
- Remove `WithAutoStartBackend(...)` option and the `autoStartBackend` field.
- Document: “If you want automatic first inference, programmatically send `ReplaceInputTextMsg` and `SubmitMessageMsg` after the program starts.”

This aligns with “chat bots, not generic inference”: starting inference is always a submit.

**Option B: make it an “auto-submit” instead of “auto-start”**

- Keep an option, but implement it by enqueuing `SubmitMessageMsg` (not `StartBackendMsg`) *after* there is meaningful input text.
- This still has complexity because pinocchio typically fills input after router is running, not at `Init()`.
  - You’d need an internal “when input becomes non-empty, auto-submit once” state.

Option A is simpler and fits the cutover rule.

### 3) Clarify the pinocchio “start in chat mode” product behavior

Pinocchio currently has two different concepts that used to be conflated:

- “start chat UI immediately” (open chat model)
- “start an inference immediately” (auto-submit prompt)

The right model is:

- chat UI starts immediately,
- inference starts only if pinocchio explicitly submits a prompt.

So pinocchio should:

1) build and run the program,
2) seed the backend turn after router is running,
3) if `StartInChat` and prompt is non-empty, send Replace+Submit messages.

### 4) Strengthen the contract between pinocchio runtime and EngineBackend

With the new Session API, `EngineBackend.Start` is the single orchestration point. We should keep it that way and avoid any “side channel” start.

Suggested cleanup (cutover-safe):

- Make it impossible to put the chat model into `StateStreamCompletion` other than via `submit()` (which calls `Backend.Start`).
- Remove any pinocchio paths that tried to start inference by directly calling backend methods or sending `StartBackendMsg`.

## Open questions / review points

### Should bobatea own any orchestration at all?

Right now, bobatea is UI-only and the backend is fully external. That’s good.

If we remove `StartBackendMsg`, bobatea becomes “pure”:

- it submits prompts,
- it renders timeline events,
- it returns to idle when it receives `BackendFinishedMsg`.

### Should EngineBackend.Start start inference synchronously or inside a Cmd?

Pinocchio currently starts inference synchronously and returns a Cmd that waits.

This avoids races where multiple submits happen before “running” is recorded, and makes “already running” errors deterministic.

## Appendix: concrete file pointers (most relevant)

- bobatea:
  - `bobatea/pkg/chat/backend.go` — Backend interface (Start requires prompt string)
  - `bobatea/pkg/chat/model.go` — `submit()` vs `startBackend()` behavior
  - `bobatea/pkg/chat/user_messages.go` — `StartBackendMsg`, `SubmitMessageMsg`
- pinocchio:
  - `pinocchio/pkg/cmds/cmd.go` — program wiring + Replace/Submit auto-submit
  - `pinocchio/pkg/ui/backend.go` — EngineBackend implements chat.Backend and uses session API
  - `pinocchio/pkg/ui/runtime/builder.go` — constructs base engine + EngineBackend + router/sinks
- geppetto:
  - `geppetto/pkg/inference/session/session.go` — Session lifecycle, one active inference invariant
  - `geppetto/pkg/inference/session/tool_loop_builder.go` — canonical builder to run tool loop w/ ctx sinks
