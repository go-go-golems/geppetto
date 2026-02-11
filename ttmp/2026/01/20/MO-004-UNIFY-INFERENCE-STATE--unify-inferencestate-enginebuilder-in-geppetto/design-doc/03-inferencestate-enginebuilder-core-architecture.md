---
Title: InferenceState + EngineBuilder Core Architecture
Ticket: MO-004-UNIFY-INFERENCE-STATE
Status: active
Topics:
    - inference
    - architecture
    - webchat
    - prompts
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-mento/go/pkg/webchat/engine_builder.go
      Note: EngineBuilder for composing engines
    - Path: go-go-mento/go/pkg/webchat/inference_state.go
      Note: Existing InferenceState reused as core
    - Path: go-go-mento/go/pkg/webchat/loops.go
      Note: ToolCallingLoop retained as core execution path
    - Path: go-go-mento/go/pkg/webchat/turns_persistence.go
      Note: Persistence hook inspiration
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-16T19:21:31-05:00
WhatFor: Define a shared inference core around InferenceState + EngineBuilder across TUI/console/webchat.
WhenToUse: When refactoring or unifying inference loops across repos and frontends.
---



# InferenceState + EngineBuilder Core Architecture

## Executive Summary

We should unify inference across TUI, console, and webchat by elevating **InferenceState** and **EngineBuilder** to the core abstraction. The runner is defined as a thin, reusable loop that executes inference (with tools) over a `turns.Turn`, updates `InferenceState`, and emits events. Conversation lifecycle and UI concerns remain downstream. This mirrors go-go-mento’s clean webchat design but without injecting or coupling to `ConversationManager`.

The design reuses the existing `ToolCallingLoop` pattern and allows callers to plug in persistence behavior via a small interface. This keeps the core inference logic deterministic while enabling different UIs (websocket, TUI, CLI) and storage strategies to compose the loop without rewriting it.

---

## Problem Statement

1. **Inference orchestration is duplicated across UIs**
   - TUI, webchat, and console flows each rebuild loops and persistence logic, which drifts over time.

2. **Engine composition is not centralized**
   - Different entry points build engines differently, leading to inconsistent middleware and tool behavior.

3. **Conversation lifecycle vs inference core is conflated**
   - Current patterns mix lifecycle (connections, eviction) with inference execution, making reuse harder.

4. **Persistence is embedded in some loops**
   - go-go-mento’s webchat embeds persistence inside the tool loop; pinocchio TUI does not persist at all. We need a pluggable persistence hook.

---

## Proposed Solution

### 1) Core types

Reuse the existing types from go-go-mento webchat:

- `InferenceState` (current run state, engine, last turn, running state)
- `EngineBuilder` (profiles + middleware + tool registry -> engine)

**Existing references:**
- `go-go-mento/go/pkg/webchat/inference_state.go`
- `go-go-mento/go/pkg/webchat/engine_builder.go`

### 2) Inference core owns the loop (ToolCallingLoop)

Use the existing `ToolCallingLoop` logic as the canonical execution path. The loop takes a seed `Turn` and returns the updated `Turn` after inference + tool execution.

**Key rule:** the loop remains pure with respect to lifecycle (no ConversationManager).

### 3) Inference core exposes persistence hooks

Define a minimal persistence interface that can be implemented by webchat, TUI, or console.

```go
// Package inferencecore

type TurnPersister interface {
    PersistTurn(ctx context.Context, state *InferenceState, t *turns.Turn, turnIndex int) error
}
```

The core loop can optionally call this persister at the end of a successful run, or when emitting snapshot hooks.

### 4) Single execution surface (who calls this?)

The *caller* is always the outermost “driver” that receives user input and triggers an inference run:

- **Webchat router / handler goroutine**
  - Example (go-go-mento): `Router.chatHandler` starts a goroutine and calls `ToolCallingLoop(...)` with `conv.Eng` and `conv.Turn` (`go-go-mento/go/pkg/webchat/router.go` + `go-go-mento/go/pkg/webchat/loops.go`).
  - In a unified design: the router owns a `Session` (see below) and calls `session.RunInference(ctx, seed)`.

- **TUI backend**
  - The TUI event loop takes submitted user input and triggers an inference call.
  - In a unified design: the TUI backend owns a `Session` per chat tab and calls `session.RunInference(ctx, seed)`, typically with a nil persister.

- **Console / CLI command**
  - A command handler constructs a session, runs once (or loops), prints the final output, then exits.
  - In a unified design: CLI builds a session, calls `session.RunInference(ctx, seed)`, and optionally persists to file/stdout.

So the correct API surface is not a giant free function with many arguments. Instead we should capture those dependencies once in a small struct (session/runner), and expose a method that is basically `RunInference(ctx, seed) -> (turn, error)`.

---

## Proposed “Session” struct to reduce call complexity

The following struct captures the “ambient” parameters that do not vary per inference call (engine, loop settings, persistence). The only per-call input is the seed `Turn`.

```go
// Package inferencecore
type Runner interface {
    // RunInference executes one full inference run (single-pass or tool-loop, depending on configuration).
    RunInference(ctx context.Context, seed *turns.Turn) (*turns.Turn, error)
}

type Session struct {
    State     *InferenceState
    Registry  geptools.ToolRegistry // nil => no tools (single-pass)
    LoopOpts  LoopOptions
    Persister TurnPersister // optional
}

func (s *Session) RunInference(ctx context.Context, seed *turns.Turn) (*turns.Turn, error)
```

This is the method that UI layers call. All other configuration is handled at Session construction time (EngineBuilder + profile config + middleware/tool composition).

---

## Architecture Diagram

```
          +------------------+
          |  EngineBuilder   |  (profile + middleware + tools)
          +---------+--------+
                    |
                    v
+-----------+    +------------------+    +------------------+
|  UI Layer | -> |  Inference Core  | -> | TurnPersister?   |
| (TUI/WS)  |    | ToolCallingLoop  |    | (optional hook)  |
+-----------+    +------------------+    +------------------+
           \            |
            \           v
             \    InferenceState
              \  (RunID + Turn + Engine)
```

---

## Core API (pseudocode)

```go
// Core state container
// (reuse existing go-go-mento webchat type)
type InferenceState struct {
    RunID string
    Turn  *turns.Turn
    Eng   engine.Engine
    // running + cancel omitted for brevity
}

// Persistence hook
// (caller supplies implementation or nil)
type TurnPersister interface {
    // PersistTurn persists a turn. The persister receives the run identifier explicitly.
    // Turn identity (TurnID) can be derived from t.ID.
    // Note: runID is intentionally redundant with t.RunID; implementations should prefer t.RunID when set.
    PersistTurn(ctx context.Context, runID string, t *turns.Turn) error
}

// Loop options
// (reuse ToolCallingLoop options / max iterations / timeout)
type LoopOptions struct {
    MaxIterations int
    TimeoutSeconds int
}

type Session struct {
    State     *InferenceState
    Registry  geptools.ToolRegistry
    LoopOpts  LoopOptions
    Persister TurnPersister
}

func (s *Session) RunInference(ctx context.Context, seed *turns.Turn) (*turns.Turn, error) {
    if s == nil || s.State == nil {
        return nil, errors.New("session/state is nil")
    }
    if seed == nil {
        seed = s.State.Turn
    }
    if seed == nil {
        seed = &turns.Turn{}
    }

    // If no registry is provided, this is a single-pass inference.
    if s.Registry == nil {
        updated, err := s.State.Eng.RunInference(ctx, seed)
        if updated != nil {
            s.State.Turn = updated
        }
        return updated, err
    }

    updated, err := ToolCallingLoop(ctx, s.State.Eng, seed, s.Registry, map[string]any{
        "max_iterations":  s.LoopOpts.MaxIterations,
        "timeout_seconds": s.LoopOpts.TimeoutSeconds,
    })

    if updated != nil {
        s.State.Turn = updated
    }

    if err == nil && s.Persister != nil {
        runID := s.State.RunID
        if updated != nil && updated.RunID != "" {
            runID = updated.RunID
        }
        _ = s.Persister.PersistTurn(ctx, runID, updated)
    }
    return updated, err
}
```

---

## How This Replaces ConversationManager Injection

**Rule:** EngineBuilder should remain free of `ConversationManager` injection.

- The inference core composes engines purely from profile + middleware + tool registry.
- Lifecycle management (connections, eviction, stream coordinator) stays in the UI/router layer.
- If a UI needs to correlate runs to connections, it does so via `InferenceState.RunID` and external maps.

This mirrors go-go-mento’s current organization but removes the “engine builder needs conversation manager” dependency, making it usable in TUI/CLI contexts.

---

## Persistence Strategy (caller-provided)

The persistence hook is intentionally small so different environments can implement it:

- **Webchat**: persist turns to DB + broadcast event frames.
- **TUI**: no persistence (nil persister).
- **Console**: optionally persist to local file or stdout.

Example implementation for webchat:

```go
type WebchatPersister struct { router *Router }
func (p *WebchatPersister) PersistTurn(ctx context.Context, runID string, t *turns.Turn) error {
    // Implementation decides indexing/versioning strategy.
    // This could write (runID, turnID=t.ID, blocks=t.Blocks) to DB or filesystem.
    return nil
}
```

---

## Design Decisions

### Decision 1: Inference core uses existing InferenceState

**Rationale:** This type already exists in go-go-mento and cleanly expresses “run ID + current turn + engine”. It is shared and stable.

### Decision 2: ToolCallingLoop remains canonical

**Rationale:** The loop already supports tool execution, step mode, snapshot hooks, and event emission. Reusing it avoids divergence.

### Decision 3: Persistence is pluggable

**Rationale:** Different UIs need different persistence behavior. A small interface allows reuse without coupling.

### Decision 4: No ConversationManager injection

**Rationale:** Conversation lifecycle is orthogonal to inference execution. Keeping the engine builder free of lifecycle dependencies makes it reusable.

### Decision 5: InferenceState and EngineBuilder live in geppetto

**Rationale:** These are core primitives shared across multiple applications/repos. Placing them in geppetto (the shared inference foundation) prevents duplicating “core” state/build logic in pinocchio/moments/go-go-mento.

---

## Alternatives Considered

1. **Keep ConversationManager in EngineBuilder**
   - Pros: convenience for webchat sinks.
   - Cons: breaks reuse in TUI/CLI; couples engine building to lifecycle.

2. **Create a new runner type instead of reusing ToolCallingLoop**
   - Pros: fresh API.
   - Cons: duplicates logic and risks drift with existing tool loop.

3. **Embed persistence directly in the loop**
   - Pros: fewer call sites.
   - Cons: enforces DB-centric behavior and makes TUI/CLI awkward.

---

## Implementation Plan

1. **Extract InferenceState + EngineBuilder to a shared package**
   - Keep go-go-mento as reference, but move the types into **geppetto** for pinocchio/moments/go-go-mento to share.

2. **Define inference core package**
   - Expose `Session.RunInference(ctx, seed)` with ToolCallingLoop + persistence hook.

3. **Refactor pinocchio TUI + webchat**
   - Replace local runner with inference core.
   - Supply a nil persister in TUI, or a filesystem persister for debug.

4. **Refactor moments webchat**
   - Use the same inference core, with a DB persister and upstream prompt resolution.

---

## Open Questions

- Should `TurnPersister` receive additional identifiers beyond `runID` (e.g., conversation ID) for webchat contexts?
- Should `EngineBuilder` become a geppetto-level “engine factory” interface to make it easier to implement in pinocchio vs moments?

---

## References

- `go-go-mento/go/pkg/webchat/inference_state.go`
- `go-go-mento/go/pkg/webchat/engine_builder.go`
- `go-go-mento/go/pkg/webchat/loops.go`
- `go-go-mento/go/pkg/webchat/turns_persistence.go`
- `go-go-mento/go/pkg/webchat/conversation_manager.go`
