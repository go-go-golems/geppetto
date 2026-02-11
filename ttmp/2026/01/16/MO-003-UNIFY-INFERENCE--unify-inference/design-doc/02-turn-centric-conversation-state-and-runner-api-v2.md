---
Title: Turn-Centric Conversation State and Runner API (v2)
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
      Note: Defines Turn
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


# Turn-Centric Conversation State and Runner API (v2)

## Executive Summary

We should invert the current model: treat `turns.Turn` as the canonical unit of conversation state and store a *sequence of Turns* as the conversation history. The inference runner should only accept a `Turn` and return a `Turn`, while the history object is responsible for appending results and producing the next seed turn. This removes the current `**ConversationState` signature, makes the runner API stable, and cleanly separates concerns: *turn construction and persistence* outside; *inference execution* inside.

This design aligns pinocchio TUI, pinocchio webchat, and moments webchat around the same core contract. Moments-specific prompt resolution stays upstream of the runner by producing a prompt string that gets appended to a seed turn. The runner remains generic (single-pass or tool-loop). UI code is strictly downstream of inference and state logic lives in a single `ConversationHistory` abstraction.

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

`turns.Turn` already represents a complete snapshot for a single inference call (`Blocks`, `Data`, `Metadata`, `RunID`). Each inference result becomes a new Turn in a history list.

**Reference type:**
- `geppetto/pkg/turns/types.go` `type Turn struct { ID, RunID, Blocks, Metadata, Data }`

### 2) Introduce `ConversationHistory` (turn sequence)

**New type (conceptual):**

```go
// geppetto/pkg/conversation/history.go

type ConversationHistory struct {
    ID    string
    RunID string
    Turns []turns.Turn
}

func (h *ConversationHistory) Last() *turns.Turn
func (h *ConversationHistory) Append(t *turns.Turn) error
```

**Key principle:** the *last* turn is the canonical snapshot used for inference. History preserves prior snapshots for audit/debugging and optional features (replay, diff).

### 3) Make the runner interface pure

**Runner API becomes:**

```go
type Runner interface {
    Run(ctx context.Context, seed *turns.Turn) (*turns.Turn, error)
}
```

The runner is stateless and unaware of conversation history. It only transforms input turn -> output turn.

### 4) Seed building lives on `ConversationHistory`

Seed construction is a method on the history type (no separate global builder).

```go
func (h *ConversationHistory) Seed(opts SeedOptions) (*turns.Turn, error)
func (h *ConversationHistory) AddUserPrompt(prompt string) error
```

- `Seed(...)` clones `Last()` (or returns empty Turn) and returns a seed.
- `AddUserPrompt(...)` appends a user block to the most recent turn or to a fresh empty turn if none exist.

### 5) All ordering/validation handled by engine/middleware

There are **no pre-run middlewares** and **no ordering validation** in `ConversationHistory` or seed building. Ordering/validation is handled by the provider/engine stack (`EngineWithMiddleware`) and the underlying API (e.g., OpenAI Responses or Claude).

This keeps history operations simple and ensures that any provider-specific constraints are enforced at the appropriate lower level.

---

## Design Decisions

### Decision 1: History stores full Turns, not deltas

**Rationale:**
- A Turn is the unit required by the engine provider. Storing full snapshots makes the history usable immediately for inference without extra reconstruction.

### Decision 2: History does not store metadata/data/version

**Rationale:**
- Metadata and data are already on each Turn. The history’s job is ordering, not enrichment.
- `RunID` is the only top-level state required to coordinate runs.

### Decision 3: Seed building has no middleware or validation

**Rationale:**
- Middleware chains (system prompt injection, prompt resolver expansion, tool-call sequencing) belong in the inference layer. Seed building should not replicate or preempt middleware behavior.
- Ordering validation (reasoning adjacency) should be the provider’s responsibility to enforce or reject; local validation risks diverging from provider rules.

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
- All code paths converge on: **History.Seed -> Runner.Run -> History.Append**.

---

## Architecture Diagram (proposed)

```
User Input
    |
    v
[ConversationHistory]
    |
    | Seed(opts)
    v
  Seed Turn
    |
    v
 [Runner: Run(ctx, seed)]  (engine+middlewares enforce ordering)
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
  |               | Seed()        |                 |
  |               |-------------> | clone/empty     |
  |               | AddUserPrompt |                 |
  |               |-------------> | append block    |
  |               | Run(seed)     |                 |
  |               |-------------------------------->|
  |               |<--------------------------------|
  |               | Append(turn)  |                 |
  |               |-------------> |                 |
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

func (h *ConversationHistory) Seed(opts SeedOptions) (*turns.Turn, error) {
    if last := h.Last(); last != nil {
        return CloneTurn(last), nil
    }
    return &turns.Turn{}, nil
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

2. **Local ordering validation in history/seed**
   - Pros: catches errors earlier.
   - Cons: duplicates provider logic and risks divergence with actual API requirements.

3. **Pre-run middlewares in seed builder**
   - Pros: predictable local composition.
   - Cons: duplicates inference middleware chain; harder to keep in sync across repos.

---

## Implementation Plan

### Phase 1: Introduce turn history abstraction
- Add `ConversationHistory` with only `RunID` and `Turns`.
- Add methods: `Last`, `Seed`, `AddUserPrompt`, `Append`.

### Phase 2: Adapter for existing ConversationState
- Provide conversion helper(s) if necessary for gradual migration.

### Phase 3: Migrate pinocchio TUI + webchat
- Replace runner signatures to use `Runner` interface and `ConversationHistory`.
- Move prompt resolution to the caller (before `AddUserPrompt`).

### Phase 4: Migrate moments webchat
- Resolve prompt text in router/profile, then call `AddUserPrompt`.
- Keep middleware/ordering enforcement inside `EngineWithMiddleware`.

---

## Open Questions

- Should tool-loop intermediate turns be appended to history or only the final result?
- Do we need a per-turn provenance record (e.g., prompt slug) for moments debugging?

---

## References

- `geppetto/pkg/turns/types.go` (Turn and Block definitions)
- `geppetto/pkg/conversation/state.go` (current block-based state)
- `pinocchio/pkg/inference/runner/runner.go` (current runner signature)
- `moments/backend/pkg/app/app.go` (PromptResolver wiring)
- `moments/docs/backend/app-initialization.md` (PromptResolver lifecycle)
