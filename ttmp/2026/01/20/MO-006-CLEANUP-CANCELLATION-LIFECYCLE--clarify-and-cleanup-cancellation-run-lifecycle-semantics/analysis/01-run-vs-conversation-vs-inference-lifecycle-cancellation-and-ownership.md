---
Title: 'Run vs Conversation vs Inference: lifecycle, cancellation, and ownership'
Ticket: MO-006-CLEANUP-CANCELLATION-LIFECYCLE
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
    - Path: geppetto/pkg/inference/core/session.go
      Note: |-
        Session lifecycle split (RunInference vs RunInferenceStarted)
        Runner lifecycle split (RunInference vs RunInferenceStarted)
    - Path: geppetto/pkg/inference/state/state.go
      Note: |-
        Current InferenceState (StartRun/FinishRun/CancelRun/HasCancel)
        Current lifecycle/cancel API (StartRun/FinishRun/CancelRun)
    - Path: go-go-mento/go/pkg/webchat/conversation.go
      Note: go-go-mento conversation embeds its own InferenceState with StartRun/FinishRun/CancelRun
    - Path: go-go-mento/go/pkg/webchat/router.go
      Note: |-
        go-go-mento webchat start/cancel endpoints using conversation state
        go-go-mento cancel path emits interrupt on context.Canceled
    - Path: moments/backend/pkg/webchat/conversation.go
      Note: Moments conversation stores running + cancel separately (not geppetto InferenceState)
    - Path: moments/backend/pkg/webchat/router.go
      Note: |-
        Moments webchat starts run loop and stores cancel; emits interrupt events
        Moments run loop cancellation and cleanup
    - Path: pinocchio/pkg/ui/backend.go
      Note: |-
        TUI run start/cancel pattern (Bubble Tea command)
        TUI Start/Interrupt/Kill patterns
    - Path: pinocchio/pkg/webchat/router.go
      Note: |-
        Webchat run start/cancel pattern (HTTP handler + goroutine)
        Webchat StartRun + goroutine + FinishRun
ExternalSources: []
Summary: Clarifies terminology and ownership for Conversation vs Inference, documents current StartRun/FinishRun/CancelRun usage, and proposes a simpler lifecycle model where a Conversation holds state and an Inference is the cancelable unit.
LastUpdated: 2026-01-20T21:40:00-05:00
WhatFor: Reduce lifecycle confusion/bugs by making run/inference cancellation and ownership explicit and uniform across TUI and webchat.
WhenToUse: When refactoring cancellation, adding new runners, or debugging stuck 'running'/'generating' states.
---


# Run vs Conversation vs Inference: lifecycle, cancellation, and ownership

## Goal

The current code uses “run” in multiple senses:

- “RunID” on a `Turn` that behaves like a *conversation identifier* in chat systems.
- “StartRun / FinishRun / CancelRun” methods that actually gate and cancel a single *inference execution* (a model call or tool loop).
- Separate “run” concepts in other parts of the ecosystem (e.g. go-go-mento indexing/task runs) that are not “chat conversations”.

This document:

1) defines a precise mental model and vocabulary (Conversation vs Inference),
2) documents current behavior in geppetto + pinocchio + go-go-mento + moments (including cancellation),
3) proposes a cleaned-up ownership model and API shape that matches how chat products work:
   - a **Conversation** is long-lived state (history, engine, ids, connections),
   - an **Inference** is the short-lived, cancelable computation that advances the state.

The intent is to make this “obvious”, in the sense that once you see the model you can predict where a bug must be (e.g., “inference finished but conversation stuck in running state”).

## Glossary (recommended vocabulary)

### Conversation (a.k.a. chat thread, session)

A long-lived interaction state that persists across multiple user prompts.

Invariants:

- It has stable identity: `ConversationID`.
- It owns the accumulated state: “history” (currently captured as a single `*turns.Turn` with blocks).
- It is *not itself cancelable*; it can be abandoned/evicted, but cancellation is about computations, not state.

### Inference (a.k.a. inference execution, model step)

A short-lived computation that consumes the current conversation state and produces an updated state.

Invariants:

- Only one inference should run at a time per conversation (unless explicitly designed otherwise).
- It is cancelable.
- It has outcomes: `completed`, `errored`, or `cancelled`.

### Tool loop inference

One inference execution may internally call the provider multiple times (tool-calling loop), but from the UI perspective this is still one inference that can be cancelled as a unit.

### IDs (recommendation)

Avoid overloading “RunID”:

- `ConversationID`: stable thread/session identifier (maps to websocket topic, URLs, storage keys).
- `InferenceID`: identifier for one inference execution (optional; helpful for tracing).
- `TurnID`: identifier of a `turns.Turn` snapshot (already present as `t.ID`).

## Current implementation: what exists today

### A. geppetto `InferenceState` (misleading name)

File: `geppetto/pkg/inference/state/state.go`

```go
type InferenceState struct {
    RunID string       // often used like a conversation id
    Turn  *turns.Turn  // accumulated history/state
    Eng   engine.Engine

    running bool
    cancel  context.CancelFunc
}
```

Methods:

- `StartRun() error` — sets `running=true` and rejects concurrent inferences.
- `FinishRun()` — clears `running=false` and drops cancel func.
- `SetCancel(cancel)` — stores a cancel func for the *current inference*.
- `CancelRun() error` — calls the stored cancel func if `running=true`.
- `HasCancel()` — used to avoid overwriting a previously stored cancel func.

Observation:

- This struct is doing **two jobs**:
  1) holding long-lived conversation state (`RunID`, `Turn`, `Eng`)
  2) tracking a short-lived inference execution (`running`, `cancel`)

This is the core source of conceptual friction: “run” methods live on a thing that also stores the conversation.

### B. geppetto `core.Session` (runner)

File: `geppetto/pkg/inference/core/session.go`

`Session` orchestrates an inference execution and can run:

- single-pass inference (`Eng.RunInference`)
- tool-loop inference (`toolhelpers.RunToolCallingLoop`)

It has two entrypoints:

- `RunInference(ctx, seed)` — owns lifecycle: calls `StartRun`, sets cancel, defers `FinishRun`.
- `RunInferenceStarted(ctx, seed)` — assumes caller already called `StartRun`; sets cancel only if missing.

This split exists for UI scheduling patterns (e.g. Bubble Tea), but it also makes it less clear who owns lifecycle cleanup in each call site.

### C. pinocchio TUI cancellation model

File: `pinocchio/pkg/ui/backend.go`

Pattern:

1) `Start(prompt)` calls `inf.StartRun()` synchronously (so the UI can reject a second submission immediately).
2) It creates `runCtx, cancel := context.WithCancel(ctx)` and stores cancel via `inf.SetCancel(cancel)`.
3) It returns a `tea.Cmd` (a thunk) that actually performs inference asynchronously.
4) The thunk defers `cancel(); inf.FinishRun()`.

Cancellation:

- UI can call `Interrupt()` which calls `inf.CancelRun()` (best-effort).

Key observation:

- The “run” is actually the **inference execution**; the conversation lives on in `inf.Turn`.

### D. pinocchio webchat cancellation model

File: `pinocchio/pkg/webchat/router.go`

Pattern (simplified):

1) HTTP handler calls `conv.Inf.StartRun()`; if already running, returns HTTP 409.
2) It spawns a goroutine that:
   - creates `runCtx, runCancel := context.WithCancel(baseCtx)`
   - stores cancel via `conv.Inf.SetCancel(runCancel)`
   - defers `runCancel(); conv.Inf.FinishRun()`
   - calls `Session.RunInferenceStarted(runCtx, seed)`

Cancellation:

- (If there is a cancel endpoint) it would call `conv.Inf.CancelRun()` which cancels the stored `runCancel`.

### E. moments webchat cancellation model (different implementation)

Files:

- `moments/backend/pkg/webchat/conversation.go`
- `moments/backend/pkg/webchat/router.go`

Moments does not use geppetto’s `InferenceState`. It holds:

- `running bool`
- `cancel context.CancelFunc`

and on run completion it does:

```go
runCancel()
conv.running = false
conv.cancel = nil
```

Cancellation:

- by calling `conv.cancel()` (through some endpoint or internal control).

Important detail: moments explicitly emits an interrupt event when the loop ends due to `context.Canceled` (see the go-go-mento snippet below for a clearer version).

### F. go-go-mento webchat cancellation model (clearer “interrupt event” handling)

Files:

- `go-go-mento/go/pkg/webchat/conversation.go`
- `go-go-mento/go/pkg/webchat/router.go`

go-go-mento’s webchat `Conversation` embeds its own `InferenceState` type (similar to geppetto’s), and in the run goroutine it:

- stores cancel via `conv.SetCancel(runCancel)`
- defers cleanup via `conv.FinishRun()`
- when `ToolCallingLoop` returns `context.Canceled`, it publishes an explicit interrupt event so the UI can finalize a “generating” entity.

This is a key behavioral invariant for UIs:

> when inference stops (cancelled), the UI should see an explicit end-of-run signal, not just “no more deltas”.

## What the user mental model is (and why it clashes)

You said:

> I understand a Run to be the same as a "Conversation", it's a long running interaction where multiple inferences are run one after the other, and an inference is the thing that can be cancelled. a Run as such is just keeping the state.

That model matches how most chat systems are designed.

But in our code today:

- the *state holder* (`InferenceState`) also contains the *in-flight inference lifecycle* fields (`running`, `cancel`),
- method names say “Run” but actually refer to the inference execution, not the conversation.

So the main fix is not “more code” but “make naming and ownership consistent with the real abstraction boundaries.”

## A precise state machine for the current system

Let’s define the current state machine that’s implicitly encoded by `InferenceState`:

### Conversation-level state (long-lived)

```
Conversation := (RunID, Turn, Engine, ...)
```

This exists as long as the user keeps the tab open / websocket connected / conversation retained.

### Inference-level state (short-lived, cancelable)

```
InferenceExecution := (running, cancel)
```

Transitions:

```
StartRun()           -> running=true
SetCancel(cancel)    -> cancel != nil
CancelRun()          -> cancel() invoked; downstream ctx becomes Done()
FinishRun()          -> running=false; cancel=nil
```

Key invariant required by UIs:

- Every path out of an inference must ultimately reach a “finished” terminal in the UI:
  - final response
  - error response
  - interrupt/cancel response

If cancellation happens but no interrupt event is emitted, the UI can remain stuck in “generating”.

## Timing diagrams (who does what, when)

### Diagram 1: Pinocchio TUI (Bubble Tea) typical inference

```
time →

UI thread:   Start(prompt)
              | StartRun()
              | runCtx, cancel := WithCancel(ctx)
              | SetCancel(cancel)
              | return tea.Cmd --------------------------------------------------+
                                                                               |
tea.Cmd goroutine:                                                             |
              +----> seed := snapshotForPrompt(prompt)                          |
              |      sess.RunInferenceStarted(runCtx, seed)                     |
              |      (provider emits deltas via sinks)                          |
              |      updated turn assigned                                      |
              +----> defer cancel(); FinishRun() ------------------------------+
```

Cancellation:

```
UI thread: Interrupt() -> CancelRun() -> cancel() -> ctx.Done() -> engine returns context.Canceled
```

Risk points:

- If engine/tool-loop returns `context.Canceled` but nobody emits an interrupt event, UI may not finalize the in-flight message.
- If FinishRun is not called (panic/early return), UI can be stuck in “running” and reject new prompts.

### Diagram 2: Webchat (HTTP handler + goroutine)

```
HTTP handler thread:  POST /chat
                      | StartRun() else 409
                      | spawn goroutine ---------------------------------------+
                                                                               |
run goroutine:                                                                  |
                      +--> runCtx, runCancel := WithCancel(baseCtx)            |
                      |    SetCancel(runCancel)                                |
                      |    sess.RunInferenceStarted(runCtx, seed)              |
                      |    ...                                                 |
                      +--> defer runCancel(); FinishRun() ---------------------+
```

Cancellation:

```
HTTP handler thread: POST /cancel -> conv.CancelRun() -> runCancel()
```

## Ownership diagrams (what should own what)

### Today (blended ownership)

```
InferenceState
  - Conversation identity (RunID)
  - Conversation history (Turn)
  - Conversation engine (Eng)
  - Inference execution lock (running)
  - Inference cancel handle (cancel)
```

This is compact, but it conflates the two levels.

### Proposed (separated ownership)

```
ConversationState
  - ConversationID
  - Turn (history)
  - Engine
  - (optional) shared config: tool registry, prompt profile, etc.

InferenceExecution (ephemeral; created per inference)
  - InferenceID
  - cancel func
  - started_at / finished_at
  - status: running|completed|errored|cancelled
```

The key idea: **cancellation belongs to the InferenceExecution**, not to the ConversationState.

The conversation persists; an inference comes and goes.

## Proposed API cleanup (textbook-style spec)

### 1) Rename and redefine concepts

- Rename `InferenceState` → `ConversationState` (or `Conversation`), because it’s primarily state.
- Rename `StartRun/FinishRun/CancelRun` → `StartInference/FinishInference/CancelInference`.

Rationale:

- Names should reflect the cancelable unit (inference), not the long-lived state (conversation).

### 2) Define explicit invariants

#### Invariant: single in-flight inference per conversation

```
StartInference succeeds iff there is no active inference.
```

#### Invariant: cancellation must be observable downstream

If an inference ends due to cancellation, downstream should receive an explicit terminal event (interrupt), not just an absence of more tokens.

This can be implemented:

- at the runner level (Session/tool-loop wrapper), because it has full control over lifecycle and sinks,
- or at a lower provider-engine level (less ideal, because cancellation may occur before an engine emits anything).

### 3) Prefer a “run handle” over Start/Finish methods

The most robust pattern is:

```go
type InferenceHandle struct {
    Cancel func()
    Wait   func() error // optional
}

func (c *ConversationState) StartInference(parent context.Context) (runCtx context.Context, h InferenceHandle, err error)
```

Where:

- `StartInference` is atomic:
  - checks “not already running”
  - creates `runCtx, cancel := context.WithCancel(parent)`
  - stores cancel in the handle
  - marks running
  - returns `runCtx` and handle
- `h.Cancel()` cancels *this* inference.
- `h.Wait()` (optional) blocks until finished.
- `h` also includes a `Finish()` method or a returned `finish()` closure to ensure cleanup cannot be forgotten.

This eliminates the “you must remember to call FinishRun in all code paths” class of bugs by making it natural to use `defer finish()`.

### 4) Collapse `RunInference` vs `RunInferenceStarted`

If the ConversationState exposes a single `StartInference` that returns `(runCtx, finish)`:

- Bubble Tea and HTTP handlers can “claim” the inference (fail fast) and get a cancel function immediately.
- The runner doesn’t need a second entrypoint (`RunInferenceStarted`) because the lifecycle is already started and structured.

Example pseudocode (TUI):

```go
runCtx, finish, err := conv.StartInference(ctx)
if err != nil { return alreadyRunning }
return tea.Cmd(func() tea.Msg {
    defer finish()
    return sess.RunInference(runCtx, seed)
})
```

No separate `HasCancel`, no “started vs not started” split.

### 5) Where does `RunID` live?

If we adopt the “Run == Conversation” model:

- (Update 2026-01-22 / GP-02) remove `Turn.RunID` entirely and store the conversation/session id in
  `Turn.Metadata` via `turns.KeyTurnMetaSessionID` (legacy `run_id` may remain as a serialized/log
  field name, but it maps to SessionID),
- make all webchat routes / topics key off `ConversationID`.

If other domains (batch indexing) have their own “run id” concept, keep it separate (`IndexingRunID`, etc).

## Where should this be implemented?

### Minimal-invasive cleanup (geppetto-only)

We can improve clarity without immediately touching moments/go-go-mento by:

1) renaming in geppetto (`InferenceState` → `ConversationState`) and updating call sites in pinocchio,
2) replacing StartRun/FinishRun/SetCancel/HasCancel with a structured `StartInference` API returning closures,
3) moving “emit interrupt event on cancel” into `core.Session` tool-loop wrapper (so UIs don’t hang).

### Broader unification (cross-repo)

Once geppetto’s API is cleaned, moments and go-go-mento can migrate by:

- mapping their local state fields (`running`, `cancel`) onto the new ConversationState API,
- deleting their local inference-state duplicates.

## Practical checklist: what to verify after changes

- Double-submit behavior:
  - second prompt while inference running is rejected cleanly (409 or “already running”) and UI does not corrupt state.
- Cancellation behavior:
  - Cancel during streaming ends with a terminal event and UI finalizes the in-flight message.
  - Cancel when idle is a no-op with a clear error (`not running`) and no side effects.
- Cleanup behavior:
  - Every inference always resets “running” state even on error/panic.
- Tool-loop behavior:
  - Cancel during a tool loop cancels the whole loop; no tools continue to execute after cancel.

## Summary (the “Norvig paragraph”)

The simplest way to reason about these systems is to separate *state* from *computation*. A conversation is state: a durable record of what has happened so far and the engine configuration that interprets it. An inference is computation: a transient process that advances the conversation state and may be cancelled. When we attach “run” methods to a structure that also stores the conversation, we introduce ambiguity about what is being started, what is being cancelled, and what must be cleaned up. The remedy is to model the cancelable unit explicitly, return a structured handle that guarantees cleanup, and enforce a small set of invariants: one inference at a time per conversation, cancellation produces an explicit terminal signal, and all exit paths execute the same cleanup logic.
