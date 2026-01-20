---
Title: Turn-Centric Conversation State and Runner API (v4)
Ticket: MO-003-UNIFY-INFERENCE
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
    - Path: geppetto/pkg/conversation/state.go
      Note: Current block-based conversation state being replaced
    - Path: geppetto/pkg/turns/types.go
      Note: Defines Turn and Run
    - Path: moments/backend/pkg/app/app.go
      Note: PromptResolver wiring relevant to upstream prompt resolution
    - Path: pinocchio/pkg/inference/runner/runner.go
      Note: Current runner signature motivating redesign
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-16T18:36:59.366713189-05:00
WhatFor: ""
WhenToUse: ""
---


# Turn-Centric Conversation State and Runner API (v4)

## Executive Summary

We should treat `turns.Turn` as the canonical unit of conversation state and store a *sequence of Turns* as the conversation history. Instead of introducing a new `ConversationHistory` type, we should reuse the existing `turns.Run` structure, which already models a multi‑turn session. The inference runner should accept a `Turn` and return a `Turn`. The run/history object (`turns.Run`) is responsible for appending results and managing the current turn passed to the runner. This removes the current `**ConversationState` signature and makes the runner API stable.

Key updates in this version:
- Replace `ConversationHistory` with `turns.Run` (no new type).
- `turns.Run` provides the sequence of `Turns` and can carry `RunID` (via `Run.ID`).
- Seed creation is handled by a constructor/helper that initializes a `turns.Run` with an initial seed turn.
- There are no pre-run middlewares or local ordering validation. All ordering enforcement is handled by the engine/middleware stack (OpenAI Responses, Claude, etc.).

## Problem Statement

### Current issues

1. **Awkward runner signature**
   - `Run(ctx, eng, state **ConversationState, seed *Turn, opts ...)` mixes ownership: it mutates state *and* accepts a seed. The `**` pointer is fragile and forces every caller into a specific pattern.

2. **State is block-based, not turn-based**
   - `ConversationState` stores an evolving slice of blocks and reconstructs a Turn on demand. The *last turn* is implicit and callers must rebuild seeds every time.

3. **Prompt construction is ambiguous**
   - Prompt insertion (user text or system prompts) happens in multiple places, making it unclear who owns seed construction.

4. **Unification with moments is blocked by prompt resolver**
   - Moments uses a `PromptResolver` to produce prompt text from slugs. This should be a pre-run concern, but the current architecture hides where it belongs.

## Proposed Solution

### 1) Make `turns.Turn` the canonical unit

`turns.Turn` represents a complete snapshot for a single inference call (`Blocks`, `Metadata`, `Data`, `RunID`). Each inference result becomes a new Turn in a history list.

**Reference type:**
- `geppetto/pkg/turns/types.go` `type Turn struct { ID, RunID, Blocks, Metadata, Data }`

### 2) Use `turns.Run` as the history container

`turns.Run` already exists and captures a multi-turn session.

**Existing type:**

```go
// geppetto/pkg/turns/types.go
// Run captures a multi-turn session.
type Run struct {
    ID    string
    Name  string
    Turns []Turn
    Metadata map[RunMetadataKey]interface{}
}
```

**Decision:** reuse `turns.Run` as the conversation history object. We can ignore `Name` and `Metadata` initially and treat `Run.ID` as the `RunID` used for correlation.

### 3) Seed creation is handled by a constructor/helper

Seed construction happens when a `turns.Run` is created (or reset). There is no `Seed()` method on the run; the run is initialized with a seed turn and then mutated via `AddUserPrompt` and `Append` operations.

```go
func NewRun(seed *turns.Turn) *turns.Run {
    r := &turns.Run{}
    if seed != nil {
        r.Turns = append(r.Turns, *seed)
        if seed.RunID != "" {
            r.ID = seed.RunID
        }
    }
    return r
}
```

### 4) Runner interface stays pure

**Runner API:**

```go
type Runner interface {
    Run(ctx context.Context, seed *turns.Turn) (*turns.Turn, error)
}
```

The runner is stateless and unaware of run/history. It only transforms input turn -> output turn.

### 5) No pre-run middleware or local ordering validation

There are **no pre-run middlewares** and **no ordering validation** in run/seed construction. Ordering is enforced by the engine/provider stack (`EngineWithMiddleware` and the underlying API). This avoids duplicating provider rules locally.

---

## How This Impacts Unification with Moments

### Current moments shape (high level)

- Prompt resolution happens via `PromptResolver` to map slugs -> prompt text.
- Webchat router constructs an engine with middleware/tooling and runs inference.

### Turn-centric design fits moments if we:

1. **Resolve prompt text before calling `AddUserPrompt`**
   - The router or profile layer resolves prompt text and passes the resulting string into `AddUserPrompt(...)`.

2. **Keep prompt resolution outside the runner**
   - Prompt resolver is not part of run or runner; it is a pre-run concern owned by moments routing/profile logic.

3. **Let moments’ router own the Run**
   - Router holds a `turns.Run` per conversation and appends new turns returned by the runner.

### Unification outcomes

- Pinocchio TUI and pinocchio webchat become clients of the same `turns.Run` + `Runner` API.
- Moments webchat aligns with the same contract by resolving prompts *before* adding user input to the run.
- All code paths converge on: **NewRun(seed) -> AddUserPrompt -> Runner.Run -> Run.Append**.

---

## Architecture Diagram (proposed)

```
User Input
    |
    v
[turns.Run]
    |
    | AddUserPrompt(prompt)
    v
 Current Turn (Last)
    |
    v
 [Runner: Run(ctx, last)]  (engine+middlewares enforce ordering)
    |
    v
 Output Turn
    |
    v
Run.Append(output)
```

### Sequence diagram (webchat/TUI)

```
Client        UI/Router         Run              Runner
  |               |               |                 |
  | prompt        |               |                 |
  |-------------->|               |                 |
  |               | AddUserPrompt |                 |
  |               |-------------> | append block    |
  |               | Run(last)     |                 |
  |               |-------------------------------->|
  |               |<--------------------------------|
  |               | Append(turn)  |                 |
  | response      |               |                 |
  |<--------------|               |                 |
```

---

## Pseudocode (core interfaces)

```go
// Runner API (pure)
type Runner interface {
    Run(ctx context.Context, seed *turns.Turn) (*turns.Turn, error)
}

func NewRun(seed *turns.Turn) *turns.Run {
    r := &turns.Run{}
    if seed != nil {
        r.Turns = append(r.Turns, *seed)
        if seed.RunID != "" {
            r.ID = seed.RunID
        }
    }
    return r
}

func RunAppend(r *turns.Run, t *turns.Turn) error {
    if t == nil {
        return errors.New("turn is nil")
    }
    r.Turns = append(r.Turns, *t)
    if t.RunID != "" {
        r.ID = t.RunID
    }
    return nil
}

func RunAddUserPrompt(r *turns.Run, prompt string) error {
    if prompt == "" {
        return nil
    }
    if len(r.Turns) == 0 {
        r.Turns = append(r.Turns, turns.Turn{})
    }
    last := &r.Turns[len(r.Turns)-1]
    last.Blocks = append(last.Blocks, turns.NewUserTextBlock(prompt))
    return nil
}
```

---

## Alternatives Considered

1. **Introduce a new `ConversationHistory` type**
   - Pros: minimal coupling with existing `turns.Run`.
   - Cons: reinvents a structure that already exists and increases cognitive load.

2. **Keep ConversationState as blocks**
   - Pros: minimal changes.
   - Cons: still conflates seed construction and state mutation; keeps runner signature awkward.

---

## Implementation Plan

### Phase 1: Use `turns.Run`
- Add helper functions for `NewRun`, `RunAddUserPrompt`, `RunAppend` (or methods on `turns.Run`).
- Replace `ConversationState` usage in pinocchio TUI/webchat with `turns.Run`.

### Phase 2: Migrate pinocchio TUI + webchat
- Replace runner signatures to use `Runner` interface and `turns.Run`.
- Move prompt resolution to the caller (before `RunAddUserPrompt`).

### Phase 3: Migrate moments webchat
- Resolve prompt text in router/profile, then call `RunAddUserPrompt`.
- Keep middleware/ordering enforcement inside `EngineWithMiddleware`.

---

## Open Questions

- Should tool-loop intermediate turns be appended to the run or only the final result?
- Do we want to formalize helper methods on `turns.Run` instead of free functions?

---

## References

- `geppetto/pkg/turns/types.go` (Turn and Run definitions)
- `geppetto/pkg/conversation/state.go` (current block-based state)
- `pinocchio/pkg/inference/runner/runner.go` (current runner signature)
- `moments/backend/pkg/app/app.go` (PromptResolver wiring)
- `moments/docs/backend/app-initialization.md` (PromptResolver lifecycle)
