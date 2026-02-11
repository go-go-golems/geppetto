---
Title: Replace Turn.RunID with SessionID in Turn metadata
Ticket: GP-02-REMOVE-TURN-RUN-ID
Status: active
Topics:
    - geppetto
    - turns
    - inference
    - refactor
    - design
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Enforces typed-key usage and restricts key constructors to key-definition files
    - Path: geppetto/pkg/doc/topics/08-turns.md
      Note: Docs show run_id as top-level field; will need update to metadata session id key
    - Path: geppetto/pkg/events/chat-events.go
      Note: EventMetadata exposes RunID; engines populate it from turns today
    - Path: geppetto/pkg/inference/session/session.go
      Note: Session.Append currently injects RunID; will switch to setting SessionID in Turn.Metadata
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: Runner currently injects RunID; needs decision/update for SessionID-in-metadata approach
    - Path: geppetto/pkg/turns/keys.go
      Note: Defines canonical typed keys; add KeyTurnMetaSessionID here
    - Path: geppetto/pkg/turns/types.go
      Note: Defines turns.Turn including RunID; would remove field and rely on Metadata
ExternalSources: []
Summary: Design analysis and migration checklist for removing turns.Turn.RunID and replacing it with a typed SessionID stored in turns.Turn.Metadata and set at session.Append time.
LastUpdated: 2026-01-22T09:57:28.592760319-05:00
WhatFor: ""
WhenToUse: ""
---


# Replace `turns.Turn.RunID` with a `SessionID` stored in `turns.Turn.Metadata`

## Goal

Remove the top-level `RunID` field from `geppetto/pkg/turns/types.go` and instead store the session correlation identifier in `Turn.Metadata`, set automatically when a `Turn` is added to a `session.Session` via `Append`.

This resolves the naming mismatch (`RunID` vs `SessionID`) by making the `Turn` model “session-independent” at construction time while still allowing session correlation once a turn becomes part of a session history.

Additional decisions (requested):

- Add `session.NewSession()` that generates a new `SessionID` (UUID).
- `Session.StartInference` must fail if the session has no seed turn, or if the seed turn is “empty” (0 blocks).

## Current State (What Exists Today)

### Types and invariants

- `geppetto/pkg/turns/types.go`:
  - `type Turn struct { ID string; Blocks []Block; Metadata Metadata; Data Data }`
  - Session correlation is stored in `Turn.Metadata` (typed key `turns.KeyTurnMetaSessionID`, id `geppetto.session_id@v1`).
- `geppetto/pkg/inference/session/session.go`:
  - `type Session struct { SessionID string; Turns []*turns.Turn; ... }`
  - `Session.Append` sets `turns.KeyTurnMetaSessionID` on appended turns when missing.
  - `Session.StartInference` fails if the session has no seed turn or the seed turn is empty (0 blocks).

### Where `Turn.RunID` is used (Geppetto code)

The load-bearing usages are “correlation” (events/logging/persistence) and “copying/cloning”:

- Engines populate `events.EventMetadata.RunID` from `t.RunID`:
  - `geppetto/pkg/steps/ai/openai_responses/engine.go`
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `geppetto/pkg/steps/ai/claude/engine_claude.go`
  - `geppetto/pkg/steps/ai/gemini/engine_gemini.go`
- Middleware uses `t.RunID` for logging context:
  - `geppetto/pkg/inference/middleware/logging_middleware.go`
  - `geppetto/pkg/inference/middleware/systemprompt_middleware.go`
- Tool-loop runner and session enforce `RunID` presence:
  - `geppetto/pkg/inference/session/session.go`
  - `geppetto/pkg/inference/session/tool_loop_builder.go`
- Tests and docs assume `Turn.RunID` exists and round-trips in YAML:
  - `geppetto/pkg/turns/serde/serde_test.go`
  - `geppetto/pkg/inference/session/session_test.go`
  - `geppetto/pkg/inference/toolhelpers/helpers_test.go`
  - `geppetto/pkg/doc/topics/08-turns.md`

Downstream, `events.EventMetadata.RunID` is used by sinks and structured event filtering, but those are decoupled from `turns.Turn` (they consume event metadata, not turns).

## How Turn Metadata Keys Work (So We Can Store `SessionID` There)

### Key format and typed access

- `Turn.Metadata` is an opaque wrapper around a `map[TurnMetadataKey]any`.
- Key identity strings are intended to be canonical: `namespace.value@vN` (see `turns.NewKeyString`).
- “Geppetto-owned” typed keys are defined centrally in `geppetto/pkg/turns/keys.go` using:
  - `TurnMetaK[T](namespace, value, version)` which produces a `TurnMetaKey[T]`
  - `TurnMetaKey[T].Get(m turns.Metadata)` and `.Set(m *turns.Metadata, value T)`

### Enforcement (turnsdatalint)

`geppetto/pkg/analysis/turnsdatalint` enforces:

1. No raw/untyped string map indexing into typed-key maps (`Turn.Data`, `Turn.Metadata`, `Block.Metadata`, `Run.Metadata`).
2. Key constructors `DataK/TurnMetaK/BlockMetaK` can only be called in key-definition files (like `pkg/turns/keys.go`), ensuring a single canonical key definition per concept.

Implication: the new SessionID metadata key must be defined in `geppetto/pkg/turns/keys.go` (or another allowed key-definition file), and all readers must use the typed key variable.

## Proposed Replacement Design

### New metadata key (namespaced + versioned)

Add a new Turn metadata key:

- Canonical key id: `geppetto.session_id@v1`
- Definition site: `geppetto/pkg/turns/keys.go`

Proposed addition:

```go
// geppetto/pkg/turns/keys.go
const (
    // Turn.Metadata
    TurnMetaSessionIDValueKey = "session_id"
)

var (
    KeyTurnMetaSessionID = TurnMetaK[string](GeppettoNamespaceKey, TurnMetaSessionIDValueKey, 1)
)
```

Naming: use `session_id` (not `run_id`) to avoid perpetuating the ambiguity. Event payloads can continue to call this value `run_id` for now (see “Migration: events”).

### `Turn` becomes session-independent

Remove the `RunID` field from `turns.Turn`:

```go
// geppetto/pkg/turns/types.go
type Turn struct {
    ID       string   `yaml:"id,omitempty"`
    Blocks   []Block  `yaml:"blocks"`
    Metadata Metadata `yaml:"metadata,omitempty"`
    Data     Data     `yaml:"data,omitempty"`
}
```

This makes session correlation an optional annotation (metadata) rather than a required top-level field.

## API Signature Changes (Explicit)

### `turns.Turn`

Before:

```go
type Turn struct {
    ID     string  `yaml:"id,omitempty"`
    RunID  string  `yaml:"run_id,omitempty"`
    Blocks []Block `yaml:"blocks"`
    Metadata Metadata `yaml:"metadata,omitempty"`
    Data     Data     `yaml:"data,omitempty"`
}
```

After:

```go
type Turn struct {
    ID     string  `yaml:"id,omitempty"`
    Blocks []Block `yaml:"blocks"`
    Metadata Metadata `yaml:"metadata,omitempty"`
    Data     Data     `yaml:"data,omitempty"`
}
```

### `session.Session.Append`

Signature can remain:

```go
func (s *Session) Append(t *turns.Turn)
```

but the behavior changes from setting `t.RunID` to setting `turns.KeyTurnMetaSessionID` in `t.Metadata`.

### `TurnPersister`

Change the interface to remove the redundant `runID` parameter; the persister should read the session id from `Turn.Metadata`:

```go
type TurnPersister interface {
    PersistTurn(ctx context.Context, t *turns.Turn) error
}
```

This implies the caller (session/runner) must ensure `turns.KeyTurnMetaSessionID` is set on the turn before calling `PersistTurn`.

## Where the SessionID is Set (Append-time)

### Key requirement from the ticket

Set the SessionID *when adding the Turn to the Session* (`Session.Append`), not at Turn construction time. This ensures turns are “portable” until associated with a specific session.

### Pseudocode for `Session.Append`

```go
// geppetto/pkg/inference/session/session.go
func (s *Session) Append(t *turns.Turn) {
    if s == nil || t == nil {
        return
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    if s.SessionID != "" {
        // Only set when missing; do not overwrite an explicit caller-provided value.
        if _, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && !ok {
            _ = turns.KeyTurnMetaSessionID.Set(&t.Metadata, s.SessionID)
        }
    }

    s.Turns = append(s.Turns, t)
}
```

### Pseudocode for `Session.StartInference` seed behavior

Today, when there is no seed turn, `StartInference` creates a `Turn` with `RunID: s.SessionID`.

After removal:

```go
// New requirement: do NOT auto-seed; caller must seed a non-empty turn first.
if input == nil || len(input.Blocks) == 0 {
    return nil, ErrSessionEmptyTurn
}
```

## Reading the SessionID (Replacing `t.RunID` Reads)

### Recommended access pattern

Replace `t.RunID` reads with:

```go
sid, _, err := turns.KeyTurnMetaSessionID.Get(t.Metadata)
```

In logging and event code, it may be preferable to treat decode errors as “missing” and log them, rather than failing the inference.

### Optional helper (if we want less boilerplate)

If we find repeated boilerplate, add helper(s) in `turns`:

```go
func SessionID(t *Turn) (string, bool, error) {
    if t == nil {
        return "", false, nil
    }
    return KeyTurnMetaSessionID.Get(t.Metadata)
}
```

This avoids sprinkling nil checks and typed-key Get logic.

## Tool Loop / Runner Considerations

`ToolLoopEngineBuilder.Build(ctx, sessionID)` injects `sessionID` into `Turn.Metadata` (via `turns.KeyTurnMetaSessionID`) inside `RunInference` when missing.

After removing `Turn.RunID`, there are two reasonable options:

1. **Strict “Append-only injection” (matches this ticket’s stated intent):**
   - Remove runner injection entirely.
   - Require callers to seed/append their turn into a `Session` before calling `RunInference` so the input turn already carries `KeyTurnMetaSessionID`.
   - Update `ToolLoopEngineBuilder_RunsToolLoopAndPersists` test accordingly.
2. **Keep runner best-effort injection for non-session usage (behavioral parity):**
   - Runner sets `KeyTurnMetaSessionID` on the input (and possibly output) turn when `sessionID != ""` and the key is missing.
   - This preserves “correlation ids exist” behavior for callers using the runner without `session.Session`.

Given the ecosystem (examples, tests, and ad-hoc runner usage), option (2) is safer, but it weakens the ticket’s “only Append sets it” rule. If we choose (1), we should update docs/examples to clearly show “seed via Session”.

## Migration Notes (High-Level)

### Engines and middleware

Update engines/middleware to treat “run id” correlation as “session id from turn metadata”:

```go
// old
metadata.RunID = t.RunID

// new (still populates EventMetadata.RunID for downstream compatibility)
if sid, ok, _ := turns.KeyTurnMetaSessionID.Get(t.Metadata); ok {
    metadata.RunID = sid
}
```

### Tests

Update tests that:

- set `Turn.RunID` directly (replace with `KeyTurnMetaSessionID.Set(&turn.Metadata, ...)` or seed via `Session.Append`)
- assert `out.RunID == ...` (replace with metadata lookup)
- round-trip YAML `run_id` (replace with round-tripping the metadata key `geppetto.session_id@v1`)

### Documentation/YAML examples

Docs currently show:

```yaml
run_id: run_abc
metadata: {}
```

After this change, `run_id` should be removed, and session correlation (if needed in fixtures) should move to metadata:

```yaml
metadata:
  geppetto.session_id@v1: sess_abc
```

Note: old YAML fixtures containing `run_id:` will silently ignore the field after removal (YAML unmarshal ignores unknown fields by default). That’s convenient but can hide mistakes; consider adding a lint step for YAML fixtures if this becomes a problem.

## File-by-File Implementation Checklist (Concrete)

### Core data model

- `geppetto/pkg/turns/types.go`
  - Remove `Turn.RunID`.
- `geppetto/pkg/turns/keys.go`
  - Add `TurnMetaSessionIDValueKey` + `KeyTurnMetaSessionID`.

### Session + runner

- `geppetto/pkg/inference/session/session.go`
  - `Append`: set `KeyTurnMetaSessionID` (instead of `t.RunID`).
  - `StartInference`: seed turn without `RunID` and rely on `Append`.
- `geppetto/pkg/inference/session/tool_loop_builder.go`
  - Decide: keep or remove session-id injection in runner.
  - Update `TurnPersister` comment to avoid referencing `t.RunID`.

### Correlation producers (events/logging)

- Engines:
  - `geppetto/pkg/steps/ai/openai_responses/engine.go`
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `geppetto/pkg/steps/ai/claude/engine_claude.go`
  - `geppetto/pkg/steps/ai/gemini/engine_gemini.go`
- Middleware:
  - `geppetto/pkg/inference/middleware/logging_middleware.go`
  - `geppetto/pkg/inference/middleware/systemprompt_middleware.go`

### Tests/docs

- `geppetto/pkg/turns/serde/serde_test.go`
- `geppetto/pkg/inference/session/session_test.go`
- `geppetto/pkg/inference/toolhelpers/helpers_test.go`
- `geppetto/pkg/doc/topics/08-turns.md`

## Open Questions / Decisions Needed

1. **Should `Session.Append` mutate the passed-in Turn or append a snapshot copy?**
   - Mutating is simplest but means “a turn becomes session-bound” as a side effect of appending.
   - Copying (and possibly deep-copying slices/maps) aligns better with “append-only snapshots” and avoids external mutation leaks.
2. **Do we keep `events.EventMetadata.RunID` naming for now?**
   - Keeping it avoids a downstream rename cascade; semantics become “session id used for correlation”.
   - Renaming would be clearer but is likely a follow-up ticket.
3. **Tool-loop runner behavior outside sessions:**
   - Do we require session-seeded turns or keep runner best-effort injection?
