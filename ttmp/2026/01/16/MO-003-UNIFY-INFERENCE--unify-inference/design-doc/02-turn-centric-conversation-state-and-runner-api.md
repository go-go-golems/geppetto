---
Title: Turn-Centric Conversation State and Runner API
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
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-16T18:36:59.366713189-05:00
WhatFor: ""
WhenToUse: ""
---

# Turn-Centric Conversation State and Runner API

## Executive Summary

We should invert the current model: treat `turns.Turn` as the canonical unit of conversation state and store a *sequence of Turns* as the conversation history. The inference runner should only accept a `Turn` and return a `Turn`, while the history object is responsible for appending results and building the next seed turn. This removes the current `**ConversationState` signature, makes the runner API stable, and cleanly separates concerns: *turn construction and persistence* outside; *inference execution* inside.

This design aligns pinocchio TUI, pinocchio webchat, and moments webchat around the same core contract. Moments-specific prompt resolution can live in a prompt/seed builder that happens **before** the runner is invoked. The runner itself remains generic (single-pass or tool-loop). The result is a unified, composable architecture where UI code is strictly downstream of the inference loop and state logic lives in a single `ConversationHistory` abstraction.

## Problem Statement

### Current issues

1. **Awkward runner signature**
   - `Run(ctx, eng, state **ConversationState, seed *Turn, opts ...)` mixes ownership: it both mutates state and accepts a seed. The `**` pointer is a smell and forces every caller into a fragile pattern.

2. **State is block-based, not turn-based**
   - `ConversationState` stores an evolving slice of blocks (`[]turns.Block`) and reconstructs a Turn on demand. This works, but it makes the *last turn* implicit rather than explicit, and it forces callers to rebuild seeds even though previous turns already represent the canonical snapshot.

3. **Prompt construction is ambiguous**
   - Prompt insertion (user text or system prompts) happens in multiple places: a runner helper (pinocchio) and middlewares (geppetto/moments). It is not obvious whether the caller or the runner owns seed construction.

4. **Unification with moments is blocked by prompt resolver**
   - Moments uses a `PromptResolver` to replace prompt tags/slots. This makes state construction more complex and is not naturally modeled inside the current runner or `ConversationState` API.

5. **Responses API enforces strict ordering**
   - Response model adjacency rules (e.g., reasoning blocks) require careful ordering. The more we mix “state mutation” and “run” into one function, the harder it is to guarantee a correct ordering pipeline.

## Proposed Solution

### 1) Make `turns.Turn` the canonical unit

`turns.Turn` already represents a complete snapshot for a single inference call (`Blocks`, `Data`, `Metadata`, `RunID`). We should treat each inference result as a new Turn in a history list.

**Current type (reference):**
- `geppetto/pkg/turns/types.go` `type Turn struct { ID, RunID, Blocks, Metadata, Data }`

### 2) Introduce `ConversationHistory` (turn sequence)

**New type (conceptual):**

```go
// geppetto/pkg/conversation/history.go

type ConversationHistory struct {
    ID      string
    RunID   string
    Turns   []turns.Turn
    Meta    turns.Metadata
    Data    turns.Data
    Version int64
}

func (h *ConversationHistory) Last() *turns.Turn
func (h *ConversationHistory) Append(t *turns.Turn) error
```

**Key principle:** the *last* turn is the canonical snapshot of the conversation used for inference. History preserves prior snapshots for audit/debugging and optional downstream features (time-travel, replay, diff).

### 3) Make the runner interface pure

**Runner API becomes:**

```go
type Runner interface {
    Run(ctx context.Context, seed *turns.Turn) (*turns.Turn, error)
}
```

The runner is *stateless* and unaware of conversation history. It only transforms input turn -> output turn.

### 4) Seed building happens outside runner

Seed construction becomes a separate concern with explicit steps:

```go
func BuildSeed(history *ConversationHistory, prompt string, opts SeedOptions) (*turns.Turn, error)
```

This function:
- clones the last Turn (or empty Turn if new)
- appends user input as a user block
- optionally runs pre-run middlewares (system prompt injection, prompt resolution, custom tags)
- validates ordering constraints before calling the runner

### 5) History integration owns persistence

```go
func (h *ConversationHistory) Run(ctx context.Context, runner Runner, seed *turns.Turn) (*turns.Turn, error) {
    out, err := runner.Run(ctx, seed)
    if err != nil {
        return nil, err
    }
    if err := h.Append(out); err != nil {
        return nil, err
    }
    return out, nil
}
```

This keeps persistence responsibility inside `ConversationHistory` (or a small adapter around it) and removes any need for `**ConversationState`.

---

## Design Decisions

### Decision 1: History stores full Turns, not deltas

**Rationale:**
- A Turn already holds the fully expanded block list; storing full snapshots makes responses-API validation easier and removes ambiguity about what the current state is.
- Deltas are harder to reason about and would require patch logic to reconstruct the full context on every request.

### Decision 2: Runner does not mutate history

**Rationale:**
- Decoupling makes `Runner` composable (tool loop vs single inference vs replay) and allows UI layers to call it consistently.
- History appends results *after* a successful run, making failure behavior predictable.

### Decision 3: Prompt resolution lives in seed building

**Rationale:**
- Moments uses `PromptResolver` to replace prompt tags; this is a pure “input construction” concern. Keeping it outside the runner avoids polluting inference logic with app-specific dependencies.
- This matches the pinocchio TUI/webchat design where prompt conversion is already upstream of `Run`.

### Decision 4: Validation is a seed-time concern

**Rationale:**
- Responses API adjacency rules (reasoning -> text/tool) must be validated before calling the engine. It should happen during seed build or immediately before runner invocation, not after.

---

## How This Impacts Unification with Moments

### Current moments shape (high level)

- Prompt resolution happens in middleware via `PromptResolver` (see `moments/backend/pkg/app/app.go` and middleware registration in webchat registries).
- Webchat router constructs an engine with middlewares/tooling and then runs inference inside the router loop.

### Turn-centric design fits moments if we:

1. **Provide a prompt-resolution hook in seed building**
   - Seed builder calls `PromptResolver.Resolve(...)` to obtain prompt text.
   - The resolved prompt is inserted as a system/user block in the seed before the runner is invoked.

2. **Keep middlewares focused on block mutation**
   - Middleware still works as Turn->Turn transformation, but prompt resolution can be modeled as a pre-middleware pass or as a special middleware in the seed builder pipeline.

3. **Let moments’ router own the History**
   - The router (or ConversationManager) holds `ConversationHistory` and uses it for each user message.
   - No changes to tool loop logic are needed, as the runner interface is unchanged.

### Unification outcomes

- Pinocchio TUI and pinocchio webchat become *clients* of a common runner + history API.
- Moments webchat can adopt the same API while still using its prompt resolver by plugging it into the seed builder stage.
- All three code paths converge on: **History -> BuildSeed -> Runner -> Append**.

---

## Architecture Diagram (proposed)

```
User Input
    |
    v
[ConversationHistory]
    |
    | BuildSeed(prompt)
    v
  Seed Turn  ----->  [Seed pipeline]
                    - prompt resolver (moments)
                    - system prompt middleware
                    - validation
    |
    v
 [Runner: Run(ctx, seed)]
    |
    v
 Output Turn
    |
    v
History.Append(output)
```

### Sequence diagram (webchat/TUI)

```
Client        UI/Router         History        SeedBuilder      Runner
  |               |               |                |              |
  | prompt        |               |                |              |
  |-------------->|               |                |              |
  |               | BuildSeed()   |                |              |
  |               |-------------> | clone+append   |              |
  |               |               |--------------> | (resolve)    |
  |               |               | <--------------|              |
  |               | Run(seed)     |                |              |
  |               |---------------------------------------------->|
  |               |<----------------------------------------------|
  |               | Append(turn)  |                |              |
  |               |-------------> |                |              |
  | response      |               |                |              |
  |<--------------|               |                |              |
```

---

## Pseudocode (core interfaces)

```go
// Runner API (pure)
type Runner interface {
    Run(ctx context.Context, seed *turns.Turn) (*turns.Turn, error)
}

// History stores turns
func (h *ConversationHistory) Append(t *turns.Turn) error {
    if t == nil {
        return errors.New("turn is nil")
    }
    h.Turns = append(h.Turns, *t)
    h.Version++
    if t.RunID != "" {
        h.RunID = t.RunID
    }
    return nil
}

// BuildSeed clones last turn and appends user prompt
func BuildSeed(h *ConversationHistory, prompt string, opts SeedOptions) (*turns.Turn, error) {
    seed := &turns.Turn{}
    if last := h.Last(); last != nil {
        seed = CloneTurn(last)
    }
    if prompt != "" {
        seed.Blocks = append(seed.Blocks, turns.NewUserTextBlock(prompt))
    }
    return ApplySeedPipeline(seed, opts)
}
```

---

## Alternatives Considered

1. **Keep ConversationState as blocks**
   - Pros: minimal changes.
   - Cons: keeps seed construction implicit and perpetuates the awkward runner signature.

2. **Keep block state but remove `**`**
   - Pros: less invasive.
   - Cons: still conflates “snapshot building” and “running” with state mutation; no clean solution for moments prompt resolution.

3. **Store deltas only (append-only block diffs)**
   - Pros: memory efficient.
   - Cons: hard to validate ordering constraints, and increases complexity for replay/debugging.

---

## Implementation Plan

### Phase 1: Introduce turn history abstraction
- Add `ConversationHistory` type and helper methods.
- Add `BuildSeed` utilities and shared validation helpers.

### Phase 2: Adapter for existing ConversationState
- Provide conversion helpers:
  - `HistoryFromState(state *ConversationState) *ConversationHistory`
  - `StateFromHistory(history *ConversationHistory) *ConversationState` (if needed temporarily)

### Phase 3: Migrate pinocchio TUI + webchat
- Replace runner signatures to use `Runner` interface and `ConversationHistory`.
- Move all prompt insertion to seed building functions.

### Phase 4: Migrate moments webchat
- Introduce a moments-specific seed builder that injects prompt resolver output.
- Replace current in-router run loop with `history.BuildSeed` + `runner.Run`.

### Phase 5: Remove old ConversationState (optional)
- Once all consumers use turn history, deprecate and remove block-based state type.

---

## Open Questions

- Do we want to persist *all* turns or only the last turn plus audit logs?
- Should tool-loop intermediate turns be appended to history or only the final result?
- Do we need a per-turn “seed provenance” record (e.g., which prompt resolver slug was used)?

---

## References

- `geppetto/pkg/turns/types.go` (Turn and Block definitions)
- `geppetto/pkg/conversation/state.go` (current block-based state)
- `pinocchio/pkg/inference/runner/runner.go` (current runner signature)
- `moments/backend/pkg/app/app.go` (PromptResolver wiring)
- `moments/docs/backend/app-initialization.md` (PromptResolver lifecycle)
