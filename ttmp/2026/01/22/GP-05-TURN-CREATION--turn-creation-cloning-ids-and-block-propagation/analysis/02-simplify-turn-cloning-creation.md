---
Title: Simplify turn cloning/creation
Ticket: GP-05-TURN-CREATION
Status: active
Topics:
    - geppetto
    - turns
    - inference
    - design
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md
      Note: Docs updated to use Turn.Clone in UI seed pattern
    - Path: geppetto/pkg/doc/topics/08-turns.md
      Note: Docs updated to remove Block.TurnID
    - Path: geppetto/pkg/inference/session/session.go
      Note: StartInference now uses Turn.Clone for safe per-inference input copies
    - Path: geppetto/pkg/turns/types.go
      Note: Defines turns.Turn.Clone() and removes Block.TurnID
    - Path: pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go
      Note: Agent snapshotForPrompt uses Turn.Clone
    - Path: pinocchio/pkg/ui/backend.go
      Note: UI snapshotForPrompt/SetSeedTurn use Turn.Clone
    - Path: pinocchio/pkg/webchat/router.go
      Note: seedForPrompt clones latest turn via Turn.Clone
ExternalSources: []
Summary: Where turn cloning happens today; which clones are still necessary; and how to simplify/centralize cloning/seed creation with turns.Turn.Clone().
LastUpdated: 2026-01-23T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---



# Simplify turn cloning/creation

## Goal

Identify where `turns.Turn` instances are being cloned/created today, determine which of those clones
are actually necessary (vs redundant), and standardize the cloning implementation so call sites
aren’t hand-rolling partial copies.

This document is intentionally focused on *simplification and correctness of cloning*; it does not
try to decide the “fresh `Turn.ID` per inference vs stable `Turn.ID` across snapshots” policy (that
choice is discussed elsewhere in the ticket).

## Key observation: there are two distinct “clone moments”

## Update (2026-01-23)

With `Session.AppendNewTurnFromUserPrompt(...)` in place, and with `Session.StartInference(...)`
running against the latest appended turn **in-place**, the cloning story is now intentionally:

- **Clone once at prompt creation time** (to avoid mutating history while building the next prompt).
- **Do not clone again inside `StartInference`** (middlewares/engines may mutate the turn and we want
  those mutations to be the persisted “latest turn” for the next prompt).

### A) Caller-side “seed for prompt” cloning (required)

Callers (Pinocchio webchat / Bubble Tea / agent backends) build a **new seed turn** by cloning the
latest snapshot and appending a new user block. This clone is required because:

- `sess.Latest()` returns a pointer to historical state owned by the session.
- appending blocks without cloning risks mutating history (slice aliasing / capacity reuse).
- even though `StartInference` now runs in-place, we still need a safe way to create the next prompt
  turn without mutating the previous one.

Preferred mechanism:

- `Session.AppendNewTurnFromUserPrompt(...)`

### B) `Session.StartInference` internal cloning (no longer done)

`Session.StartInference` now runs against the latest appended turn in-place. This makes it easier
for middlewares to intentionally mutate the turn (e.g. normalization, system-prompt insertion,
tool-config adjustments) and have those changes become the next prompt’s base.

## What changed in this ticket (simplification)

### 1) `Block.TurnID` removed

`turns.Block` no longer carries `TurnID`. It was unused in practice, and its presence encouraged
partial “best-effort” propagation ideas that were not implemented consistently.

If block-level attribution is needed in the future, it should likely be expressed via
`Block.Metadata` typed keys rather than a dedicated struct field.

### 2) `turns.Turn.Clone()` added and used

A single canonical clone implementation now lives on `turns.Turn`:

- deep-copies the `[]Block` slice
- deep-copies each `Block.Payload` map
- clones `Turn.Metadata`, `Turn.Data`, and each `Block.Metadata` (shallow map copies)

This removes duplicated (and previously inconsistent) “cloneTurn” helpers across repositories.

## Detailed code locations (and why each exists)

### `geppetto/pkg/inference/session/session.go` — `(*Session).StartInference`

Behavior:

- reads latest historical turn from `s.Turns`
- stamps `geppetto.session_id@v1` + `geppetto.inference_id@v1` on the latest appended turn
- runs inference against that turn in-place (middlewares may mutate it)
- does **not** append a new “output turn”; the latest turn is the canonical updated state (if a
  runner returns a different pointer, the session copies the result into the latest turn)

Why it exists:

- drives inference and stamps per-inference metadata.

### `pinocchio/pkg/webchat/router.go` — prompt handlers

Behavior:

- calls `conv.Sess.AppendNewTurnFromUserPrompt(prompt)` (clone latest + append user block + append)
- calls `StartInference`

Why its cloning is required:

- avoids mutating the latest historical snapshot in-place while constructing a new prompt snapshot.

Note: With `StartInference` now running in-place, prompt creation is the one “safe clone moment” we
still want to centralize and standardize.

### `pinocchio/pkg/ui/backend.go` — `Start` / `SetSeedTurn`

Behavior:

- uses `sess.AppendNewTurnFromUserPrompt(prompt)` in `Start`
- clones an externally-provided seed turn before attaching it to a fresh session (`SetSeedTurn`)

Why its cloning is required:

- same reason as webchat: seed construction must not mutate session history or shared input objects.

### `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go` — `Start`

Same prompt-append pattern as UI and webchat; cloning is required for safe seed construction.

### Non-clone helpers (do not need to use `Turn.Clone`)

Some helpers intentionally copy only block slices (not a full turn clone), e.g. middleware diffing
helpers that return a `[]turns.Block` view. These should remain simple slice copies because they are
not “turn cloning”.

## Recommendations for further simplification (follow-up ideas)

1) Prefer a single prompt→turn API:
   - standardize UIs on `Session.AppendNewTurnFromUserPrompt(...)` for follow-up prompts.
   - keep `Session.Append(...)` for initial seed turns / externally loaded state.

2) Decide and enforce a single `Turn.ID` policy:
   - today many callers preserve `Turn.ID` when cloning snapshots; if we want fresh IDs per
     inference, that needs an explicit, centralized policy (and likely a migration).

3) If block-level attribution is still desired:
   - add typed keys for block inference attribution and stamp them at creation time (or post-run),
     rather than re-introducing `Block.TurnID`.
