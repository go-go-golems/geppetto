---
Title: Session Management in Geppetto
Slug: geppetto-sessions
Short: Managing multi-turn interactions with session.Session — turn history, inference lifecycle, and async execution.
Topics:
- geppetto
- sessions
- turns
- architecture
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

# Session Management in Geppetto

## Why Sessions?

A single inference call processes one Turn. But most applications involve multiple exchanges: the user asks a question, the model responds, the user asks a follow-up, and so on. **Sessions** manage this multi-turn lifecycle.

A Session provides:

- **Turn history** — an append-only list of Turn snapshots, each a complete record of one inference cycle.
- **Safe turn creation** — clones the latest Turn and appends the new user prompt, preventing accidental mutation of historical snapshots.
- **Exclusive inference** — only one inference runs at a time per session, enforced by mutex.
- **Async execution** — inference runs in a goroutine; callers get an `ExecutionHandle` to wait or cancel.

## Core Concepts

### The Session Struct

```go
type Session struct {
    SessionID string          // Stable identifier for this session
    Turns     []*turns.Turn   // Append-only history of turn snapshots
    Builder   EngineBuilder   // Creates inference runners (wires engine, middleware, tools, etc.)
}
```

`SessionID` is auto-generated (UUID) when you create a session. `Turns` grows as the conversation progresses. `Builder` is the bridge to the inference pipeline — it produces a runner that knows how to execute inference with the right engine, middleware, event sinks, and tools.

### How Turns Grow

Each new Turn starts as a **clone** of the previous Turn's final state, with the new user prompt appended:

```
Turn 1 (seed):            [system, user₁]
Turn 1 (after inference): [system, user₁, llm_text₁]

Turn 2 = clone(Turn 1) + user₂:
Turn 2 (seed):            [system, user₁, llm_text₁, user₂]
Turn 2 (after inference): [system, user₁, llm_text₁, user₂, tool_call, tool_use, llm_text₂]
```

This means every Turn is a complete snapshot — you can examine any Turn in isolation and see the full context the model had.

### The ExecutionHandle

When you start inference, you get back an `ExecutionHandle` immediately (inference runs asynchronously):

```go
type ExecutionHandle struct {
    SessionID   string
    InferenceID string
    Input       *turns.Turn
}
```

Use it to:
- **Wait**: `result, err := handle.Wait()` — blocks until inference completes.
- **Cancel**: `handle.Cancel()` — cancels the in-flight inference via context cancellation.
- **Check**: `handle.IsRunning()` — non-blocking check.

## Basic Usage

```go
import "github.com/go-go-golems/geppetto/pkg/inference/session"

// 1. Create a session
sess := session.NewSession()
sess.Builder = myEngineBuilder // see EngineBuilder below

// 2. Add the first user prompt (creates the seed Turn)
turn, err := sess.AppendNewTurnFromUserPrompt("What's the weather in Paris?")

// 3. Run inference
handle, err := sess.StartInference(ctx)
if err != nil {
    // ErrSessionAlreadyActive if another inference is running
    // ErrSessionEmptyTurn if the turn has no blocks
}

// 4. Wait for the result
result, err := handle.Wait()
// result is the completed Turn (same pointer as sess.Latest())

// 5. Continue the conversation
turn2, _ := sess.AppendNewTurnFromUserPrompt("What about tomorrow?")
handle2, _ := sess.StartInference(ctx)
result2, _ := handle2.Wait()
```

### Multiple Prompts

You can append multiple user prompts at once (useful for multi-modal input or batch scenarios):

```go
turn, err := sess.AppendNewTurnFromUserPrompts("Summarize this:", "Also extract key dates.")
```

## The EngineBuilder Interface

The Session delegates all inference pipeline construction to an `EngineBuilder`:

```go
type EngineBuilder interface {
    Build(ctx context.Context, sessionID string) (InferenceRunner, error)
}

type InferenceRunner interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

The Builder is responsible for wiring together:
- The inference engine (OpenAI, Claude, Gemini, etc.)
- Middleware chain (system prompt, agent mode, tool reorder, etc.)
- Event sinks (for streaming events)
- Tool registry (for tool calling)
- Snapshot hooks (for debugging)
- Persistence (for storing completed turns)

The canonical implementation is `enginebuilder.Builder` from `geppetto/pkg/inference/toolloop/enginebuilder/`.

## How StartInference Works

Understanding the internal flow helps with debugging:

1. **Validates** session state (not nil, has turns, no active inference).
2. **Sets metadata** on the latest Turn: `SessionID`, `InferenceID`, `TurnID`.
3. **Calls `Builder.Build()`** to create an `InferenceRunner`.
4. **Creates an `ExecutionHandle`** with a cancellable context.
5. **Launches a goroutine** that calls `runner.RunInference(ctx, turn)`.
6. **On completion**: stores the result in the handle, clears the active state.

The latest Turn in `sess.Turns` is **mutated in place** by the runner and middleware. This is intentional: middleware modifications (system prompt updates, block reordering, tool results) become part of the Turn's final state and serve as the base for the next Turn.

## Cancellation

```go
// Cancel via the handle
handle.Cancel()

// Or cancel via the session (cancels whatever is active)
err := sess.CancelActive()
```

Cancellation propagates via `context.Context`: the engine, tool loop, and middleware all see the cancellation and can clean up.

## Concurrency Model

- **One active inference at a time.** If you call `StartInference()` while another is running, you get `ErrSessionAlreadyActive`.
- **Thread-safe.** All Session methods are protected by a mutex. Multiple goroutines can safely call `AppendNewTurnFromUserPrompt` and `StartInference` (though only one inference runs at a time).
- **Wait from any goroutine.** Multiple callers can `Wait()` on the same handle — they all receive the same result.

## Error Handling

| Error | When | What to do |
|-------|------|------------|
| `ErrSessionNil` | Session pointer is nil | Check initialization |
| `ErrSessionNoID` | SessionID is empty | Ensure `NewSession()` was used |
| `ErrSessionAlreadyActive` | Inference already running | Wait for current inference or cancel it |
| `ErrSessionEmptyTurn` | Latest turn has no blocks | Append user prompt before starting inference |
| `ErrSessionNoBuilder` | Builder is nil | Set `sess.Builder` before calling StartInference |

## Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/session"            // Session, ExecutionHandle
    "github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder" // Canonical EngineBuilder
    "github.com/go-go-golems/geppetto/pkg/turns"                        // Turn, Block types
)
```

## See Also

- [Turns and Blocks](08-turns.md) — The Turn data model that sessions manage
- [Inference Engines](06-inference-engines.md) — How engines process Turns; see "Complete Runtime Flow"
- [Middlewares](09-middlewares.md) — Middleware applied during inference
- [Events](04-events.md) — Streaming events emitted during inference
- Implementation: `geppetto/pkg/inference/session/session.go`
