---
Title: Refactor Postmortem
Ticket: GP-02-REMOVE-TURN-RUN-ID
Status: active
Topics:
    - geppetto
    - turns
    - inference
    - refactor
    - design
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../moments/backend/pkg/artifact/buffer.go
      Note: Final runID→sessionID signature cleanup in moments
    - Path: pkg/events/chat-events.go
      Note: Legacy run_id JSON field + canonical SessionID
    - Path: pkg/inference/session/session.go
      Note: SessionID/InferenceID/TurnID invariants
ExternalSources: []
Summary: Engineering report for the GP-02 RunID→SessionID/InferenceID refactor (what changed, why, and lessons learned).
LastUpdated: 2026-01-22T16:55:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Refactor Postmortem

## Goal

Produce an engineering postmortem for the GP-02 refactor that removed `Turn.RunID` and standardized
correlation IDs across Geppetto + downstream repos (Pinocchio + Moments), including what changed,
what was tricky, what went well, and what I would do differently next time.

## Context

Historically, the codebase used the term **RunID** inconsistently:

- Sometimes it meant “the long-lived multi-turn session id” (what the new session API calls `SessionID`).
- Sometimes it was (implicitly) overloaded to mean “the id for a single inference execution”.
- Some stores/loggers/events used `run_id` as a serialized/log field name, even when the semantic
  intent was “session id”.

This ambiguity made it hard to reason about:

- event stream filtering and correlation
- persistence keys (SQLite/debug stores, Redis streams)
- debugging (what is stable across turns vs per-inference)
- compatibility between repos evolving at different speeds

The refactor introduced a stable, explicit model:

- `SessionID` — stable multi-turn id (stored on `Turn.Metadata` via typed key)
- `InferenceID` — unique per single inference execution (per `RunInference` call)
- `TurnID` — per-turn id (stored on `Turn.ID`)

…and retained legacy `run_id` where needed for log field naming / API compatibility.

## Quick Reference

### Canonical semantics

- **SessionID**: long-lived multi-turn id, stored on `Turn.Metadata`:
  - `turns.KeyTurnMetaSessionID` (YAML key string: `geppetto.session_id@v1`)
- **InferenceID**: unique per inference execution, stored on `Turn.Metadata`:
  - `turns.KeyTurnMetaInferenceID` (YAML key string: `geppetto.inference_id@v1`)
- **TurnID**: per-turn id stored on `Turn.ID`; must never be empty during inference execution.

### Mapping old → new

| Old name | New name | Notes |
|---|---|---|
| `turns.Turn.RunID` | `turns.KeyTurnMetaSessionID` | Removed `RunID` field entirely |
| `events.EventMetadata.RunID` | `events.EventMetadata.SessionID` | Code must populate from Turn metadata |
| “run loop id” | `SessionID` + `InferenceID` | session is stable; inference is per call |
| log field `run_id` | still `run_id` | retained as legacy log field name, but value is SessionID |

### Invariants (enforced/relied on)

- `Session.StartInference` fails if `SessionID` is empty.
- `Session.StartInference` ensures `Turn.ID` exists before invoking the runner.
- The tool loop runner ensures `Turn.ID` exists and propagates it to the returned turn if missing.
- Engines populate `events.EventMetadata.{SessionID,InferenceID,TurnID}` from `Turn.Metadata` + `Turn.ID`.

### Validation commands

```bash
cd geppetto && go test ./... -count=1
cd pinocchio && go test ./... -count=1
cd moments/backend && go test ./... -count=1
cd moments/backend && make lint
```

### Key commits (workspaces)

- Geppetto:
  - `122c963` — ensure `TurnID`/`InferenceID` always set/propagated (session + tool loop runner)
  - `bdc03c1` — remove `NewEngineWithMiddleware` helper and standardize on tool-loop builder middleware
- Pinocchio:
  - `25c61b3` — adapt to `events.EventMetadata.SessionID/InferenceID`
  - `91a59ee` — rename SQLite debug store columns to `session_id` (no DB compat)
  - `2886551` — webchat: return `session_id`/`inference_id`/`turn_id`; start EngineConfig/Builder wiring
- Moments:
  - `6bb64356` — rename `RunID` → `SessionID` across moments/backend (and wire `InferenceID`)
  - `80d3a087` — rename remaining `runID` function params in artifact buffering to `sessionID`

## Usage Examples

### How to seed a session turn correctly

```go
sess := session.NewSession()
seed := &turns.Turn{}
turns.AppendBlock(seed, turns.NewUserTextBlock("hello"))
sess.Append(seed) // appends snapshot; ensures KeyTurnMetaSessionID is set on the turn
handle, err := sess.StartInference(ctx)
```

### How to build event metadata inside an engine/middleware

```go
md := events.EventMetadata{ID: uuid.New()}
if sid, ok, _ := turns.KeyTurnMetaSessionID.Get(t.Metadata); ok { md.SessionID = sid }
if iid, ok, _ := turns.KeyTurnMetaInferenceID.Get(t.Metadata); ok { md.InferenceID = iid }
md.TurnID = t.ID
```

### How to log without reintroducing RunID ambiguity

```go
log.Info().
  Str("run_id", md.SessionID).     // legacy field name
  Str("session_id", md.SessionID).
  Str("inference_id", md.InferenceID).
  Str("turn_id", md.TurnID).
  Msg("...")
```

## Engineering Report

### What changed (high-level)

- **Geppetto core**
  - Removed `turns.Turn.RunID` and migrated session correlation to typed metadata keys.
  - Introduced/standardized `InferenceID` via typed metadata key (`geppetto.inference_id@v1`).
  - Tightened session invariants (`SessionID` required; seed turn must exist and have blocks).
  - Ensured `Turn.ID` exists before inference, and is preserved across runner outputs.
- **Pinocchio**
  - Updated event metadata handling and started returning canonical IDs from `/chat` endpoints.
  - Updated SQLite debug store schema to `session_id`/`inference_id` (no migration/back-compat).
- **Moments**
  - Renamed `RunID` identifiers (fields, methods, interfaces) → `SessionID` throughout backend.
  - Updated all `events.EventMetadata` construction to use `SessionID` (and wired `InferenceID`).
  - Updated SEM handler caches from “run→turn” to “session→turn”.

### What was tricky

- **Root cause wasn’t mechanical**: “RunID” wasn’t a simple rename; it was a conflated concept with
  real downstream behaviors (stream filtering, stores, UI assumptions).
- **Propagation guarantees**: many subsystems assumed IDs already existed; removing a struct field
  exposed “missing id” edge cases. The right fix was to enforce invariants at the session/runner
  boundary (not try to patch dozens of call sites ad hoc).
- **Copy semantics**: session history is append-only; the session must not mutate historical turns
  in-place when tagging per-inference metadata (hence defensive copies).
- **Downstream repos**: because this was a breaking API change (`Turn.RunID` removed), keeping the
  workspace green required coordinating Pinocchio + Moments updates.
- **Git hooks/tooling friction**: moments’ `lefthook.yml` decoding issue forced commits using
  `LEFTHOOK=0` (this is operational debt, not a product change, but it impacts iteration speed).

### What went well

- The typed key infrastructure (`TurnMetaK[...]`) made the metadata migration auditable and less error-prone.
- Introducing `InferenceID` as a first-class key made it possible to reason about “one inference”
  even when there’s a long-lived session.
- Tests/lint were able to validate end-to-end quickly (especially in `moments/backend`, which has a
  strong `make lint` + custom analyzers).

### What I would do better next time

1. **Start with a compatibility matrix** (per consumer: CLI examples, webchat servers, sinks,
   stores, web UI) to decide explicitly where legacy `run_id` must remain and where it can be
   removed immediately.
2. **Add a small “ID contract” package/doc earlier**, including:
   - canonical meanings (session vs inference vs turn)
   - where each ID is sourced from (metadata vs turn struct)
   - whether it is required vs best-effort
3. **Automate the mechanical renames** (codemod) once the conceptual mapping is fixed.
4. **Add tests that assert invariants**, not just compilation:
   - “TurnID never empty during inference”
   - “InferenceID stable throughout one tool loop execution”
   - “Event metadata always contains SessionID+TurnID, InferenceID when available”
5. **Fix the tooling friction early** (moments lefthook) so it doesn’t distort iteration costs.

### Known follow-ups

- Complete the “docs sweep” task (remaining references to `Turn.RunID` in older docs).
- Decide how to migrate moments/web from legacy `run_id` to canonical `session_id` while preserving
  the planning widget’s per-run correlation key (see
  `geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/analysis/03-moments-web-run-id-usage-audit.md`).

## Related

- `geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/reference/01-diary.md`
- `geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/analysis/01-replace-turn-runid-with-sessionid-in-turn-metadata.md`
- `geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/analysis/02-moments-backend-rename-runid-to-sessionid.md`
- `geppetto/pkg/events/chat-events.go` (EventMetadata struct + legacy JSON handling)
