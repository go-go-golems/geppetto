---
title: Investigation diary
doc_type: reference
status: active
intent: long-term
topics:
  - geppetto
  - goja
  - js-bindings
  - turns
  - architecture
owners:
  - manuel
created: 2026-06-02
updated: 2026-06-02
---

# Diary

## Goal

Record the correction and design work for the missing JavaScript Turn append/continuation API ticket.

## Step 1: Correct the second storage-adjacent ticket to turn continuation ergonomics

The earlier ticket split created a timeline-storage boundary ticket, but the requested second concern was actually the JS turn continuation ergonomics: adding a `turn()` API shape that can start from an existing turn, append new blocks, and return a new Go-owned turn. I created the correct ticket and wrote an implementation guide for that API.

The design preserves the hard-cut rules. It does not add hidden chat state, does not add `agent.ask()`, and does not accept plain JS turn-shaped objects. Instead, `gp.turn(existingTurn).user(...).build()` is a pure clone-and-append operation over Go-owned `Turn` wrappers.

### Prompt Context

**User prompt (verbatim):** "so the one ticket should have been about adding the turn() api where we can append new blocks and get a new turn out of it. and the other one the one for turns store."

**Assistant interpretation:** The previous timeline-storage ticket was not the intended second ticket; create the correct turn append/continuation API ticket while keeping the turn-store ticket as the other requested ticket.

**Inferred user intent:** Make the ticket set match the actual desired future work: one ticket for JS turn continuation ergonomics and one ticket for JS turn-store persistence.

**Commit (code):** N/A — documentation/design only.

### What I did
- Created ticket `GP-JS-TURN-APPEND-2026-06-02`.
- Added design doc `design-doc/01-javascript-turn-append-api-design-and-implementation-guide.md`.
- Read the current JS turn builder implementation in `pkg/js/modules/geppetto/api_turn_builder.go`.
- Read `pkg/turns/types.go` for `Turn.Clone()` semantics.
- Read `pkg/inference/session/session.go` for the existing continuation rule that clears cloned `Turn.ID` before appending a new user prompt.
- Used `examples/js/geppetto/30_real_provider_multiturn.js` as evidence for the ergonomic gap.

### Why
- Current built turn wrappers only support `toJSON()` and `clone()`.
- Multi-turn scripts must manually reconstruct prior context instead of starting from `result.outputTurn()`.
- The desired behavior is a value transformation, not a hidden chat API.

### What worked
- Existing builder methods (`system`, `user`, `assistant`, `metadata`, `build`) are already the right append vocabulary.
- The implementation can be small: update `gp.turn(...)` to accept an optional base `TurnWrapper` and reuse existing builder machinery.
- The Go session layer already provides the key ID-clearing precedent.

### What didn't work
- The previous ticket split included a timeline-storage ticket as one of the pair. That ticket may still be useful as supplementary boundary documentation, but it was not the intended replacement for the turn append API ticket.

### What I learned
- `gp.turn(existing)` should likely clear the copied `Turn.ID` by default, while `turn.clone()` should preserve the ID.
- The API can improve multi-turn ergonomics without compromising explicit-turn execution.

### What was tricky to build
- The tricky part is distinguishing exact clone semantics from continuation semantics. `turn.clone()` should preserve identity because it says "clone"; `gp.turn(existing).user(...).build()` should create a new turn with prior context and therefore should not retain the previous persisted turn ID.
- The guide resolves this with an explicit behavior table and tests.

### What warrants a second pair of eyes
- Whether ID clearing should happen immediately in `gp.turn(existing)` or only once the first append method is called.
- Whether convenience methods on `TurnWrapper` should be added in the first implementation or deferred.

### What should be done in the future
- Implement `gp.turn(existingTurn)`.
- Add tests for wrapper-only base validation, immutable append, ID clearing, and multimodal continuation.
- Update multi-turn examples and TypeScript declarations.

### Code review instructions
- Start with `pkg/js/modules/geppetto/api_turn_builder.go`.
- Compare the ID rule with `pkg/inference/session/session.go:AppendNewTurnFromUserPrompts`.
- Validate with `go test ./pkg/js/modules/geppetto -count=1` and the hard-cut public surface contract test.

### Technical details
- Preferred API: `gp.turn(previousTurn).assistant(...).user(...).build()`.
- Explicit non-goal: no `agent.ask()`, no hidden agent transcript, no plain JS object import.
- Related ticket that remains part of the intended pair: `GP-JS-TURNSTORE-2026-06-02`.
