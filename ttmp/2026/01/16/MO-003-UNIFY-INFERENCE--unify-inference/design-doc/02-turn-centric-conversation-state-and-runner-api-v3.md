---
Title: Turn-Centric Conversation State and Runner API (v3)
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
      Note: Defines Turn and Run types referenced in design
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


# Turn-Centric Conversation State and Runner API (v3)

## Executive Summary

We should treat `turns.Turn` as the canonical unit of conversation state and store a *sequence of Turns* as the conversation history. The inference runner should only accept a `Turn` and return a `Turn`. History is responsible for appending results and managing the “current” turn that will be passed to the runner. This removes the current `**ConversationState` signature and makes the runner API stable.

Key updates in this version:
- `ConversationHistory` only stores `RunID` and `Turns` (no ID, no metadata/data, no version).
- Seed creation is moved into the `ConversationHistory` constructor; there is no `Seed()` method.
- There are no pre-run middlewares or local ordering validation. All ordering enforcement is handled by the engine/middleware stack (e.g., OpenAI Responses, Claude).

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

### 2) Introduce `ConversationHistory` (turn sequence)

**New type (conceptual):**

```go
// geppetto/pkg/conversation/history.go

type ConversationHistory struct {
    RunID string
    Turns []turns.Turn
}

func NewConversationHistory(seed *turns.Turn) *ConversationHistory
func (h *ConversationHistory) Last() *turns.Turn
func (h *ConversationHistory) Append(t *turns.Turn) error
func (h *ConversationHistory) AddUserPrompt(prompt string) error
```

**Key principle:** the *last* turn is the canonical snapshot used for inference. History preserves prior snapshots for audit/debugging and optional features (replay, diff).

### 3) Move seed construction into the constructor

There is **no `Seed()` method**. Instead, seed construction happens when a new `ConversationHistory` is created, and subsequent user input is appended via `AddUserPrompt` on the current last turn.

```go
func NewConversationHistory(seed *turns.Turn) *ConversationHistory {
    h := &ConversationHistory{}
    if seed != nil {
        h.Turns = append(h.Turns, *seed)
        if seed.RunID != "" {
            h.RunID = seed.RunID
        }
    }
    return h
}
```

### 4) Runner interface stays pure

**Runner API:**

```go
type Runner interface {
    Run(ctx context.Context, seed *turns.Turn) (*turns.Turn, error)
}
```

The runner is stateless and unaware of conversation history. It only transforms input turn -> output turn.

### 5) No pre-run middleware or local ordering validation

There are **no pre-run middlewares** and **no ordering validation** in history or seed construction. Ordering is enforced by the engine/provider stack (`EngineWithMiddleware` and the underlying API). This avoids duplicating provider rules locally.

---

## Relationship to Existing `turns.Run`

There is already a `Run` type in the `turns` package:

- `geppetto/pkg/turns/types.go`:

```go
// Run captures a multi-turn session.
type Run struct {
    ID    string
    Name  string
    Turns []Turn
    Metadata map[RunMetadataKey]interface{}
}
```

The proposed `ConversationHistory` is effectively a *stripped-down* run container. We can:

- **Option A (reuse `turns.Run`)**: treat `Run.ID` as `RunID` and ignore `Name`/`Metadata` for now.
- **Option B (wrapper)**: create a thin `ConversationHistory` that holds `RunID` + `Turns` and can be converted to/from `turns.Run`.

This is an explicit decision point. If we already have `Run`, it may be better to reuse it and avoid new types.

---

## How This Impacts Unification with Moments

### Current moments shape (high level)

- Prompt resolution happens via `PromptResolver` to map slugs -> prompt text.
- Webchat router constructs an engine with middleware/tooling and runs inference.

### Turn-centric design fits moments if we:

1. **Resolve prompt text before calling `AddUserPrompt`**
   - The router or profile layer resolves prompt text and passes the resulting string into `AddUserPrompt(...)`.

2. **Keep prompt resolution outside the runner**
   - Prompt resolver is not part of history or runner; it is a pre-run concern owned by moments routing/profile logic.

3. **Let moments’ router own the History**
   - Router holds a `ConversationHistory` per conversation and appends new turns returned by the runner.

### Unification outcomes

- Pinocchio TUI and pinocchio webchat become clients of the same `ConversationHistory` + `Runner` API.
- Moments webchat aligns with the same contract by resolving prompts *before* adding user input to the history.
- All code paths converge on: **NewHistory(seed) -> AddUserPrompt -> Runner.Run -> History.Append**.

---

## Architecture Diagram (proposed)

```
User Input
    |
    v
[ConversationHistory]
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
History.Append(output)
```

### Sequence diagram (webchat/TUI)

```
Client        UI/Router         History           Runner
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

// History stores turns only
func NewConversationHistory(seed *turns.Turn) *ConversationHistory {
    h := &ConversationHistory{}
    if seed != nil {
        h.Turns = append(h.Turns, *seed)
        if seed.RunID != "" {
            h.RunID = seed.RunID
        }
    }
    return h
}

func (h *ConversationHistory) Append(t *turns.Turn) error {
    if t == nil {
        return errors.New("turn is nil")
    }
    h.Turns = append(h.Turns, *t)
    if t.RunID != "" {
        h.RunID = t.RunID
    }
    return nil
}

func (h *ConversationHistory) AddUserPrompt(prompt string) error {
    if prompt == "" {
        return nil
    }
    if len(h.Turns) == 0 {
        h.Turns = append(h.Turns, turns.Turn{})
    }
    last := &h.Turns[len(h.Turns)-1]
    last.Blocks = append(last.Blocks, turns.NewUserTextBlock(prompt))
    return nil
}
```

---

## Alternatives Considered

1. **Keep ConversationState as blocks**
   - Pros: minimal changes.
   - Cons: still conflates seed construction and state mutation; keeps runner signature awkward.

2. **Local ordering validation in history**
   - Pros: early detection.
   - Cons: duplicates provider logic and risks divergence with real API behavior.

3. **Pre-run middleware in history**
   - Pros: predictable local pipeline.
   - Cons: duplicates the actual inference middleware stack.

---

## Implementation Plan

### Phase 1: Introduce turn history abstraction
- Add `ConversationHistory` with only `RunID` and `Turns`.
- Add constructor `NewConversationHistory(seed *turns.Turn)`.
- Add methods: `Last`, `AddUserPrompt`, `Append`.

### Phase 2: Decide on `turns.Run`
- Either reuse `turns.Run` as the history container or create a thin wrapper.

### Phase 3: Migrate pinocchio TUI + webchat
- Replace runner signatures to use `Runner` interface and `ConversationHistory`.
- Move prompt resolution to the caller (before `AddUserPrompt`).

### Phase 4: Migrate moments webchat
- Resolve prompt text in router/profile, then call `AddUserPrompt`.
- Keep middleware/ordering enforcement inside `EngineWithMiddleware`.

---

## Open Questions

- Should tool-loop intermediate turns be appended to history or only the final result?
- Should `ConversationHistory` be replaced by `turns.Run`?
- Do we need a per-turn provenance record (e.g., prompt slug) for moments debugging?

---

## References

- `geppetto/pkg/turns/types.go` (Turn and Run definitions)
- `geppetto/pkg/conversation/state.go` (current block-based state)
- `pinocchio/pkg/inference/runner/runner.go` (current runner signature)
- `moments/backend/pkg/app/app.go` (PromptResolver wiring)
- `moments/docs/backend/app-initialization.md` (PromptResolver lifecycle)
