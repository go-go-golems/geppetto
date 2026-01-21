---
Title: 'Session refactor: SessionID + EngineBuilder + ExecutionHandle'
Ticket: MO-007-SESSION-REFACTOR
Status: active
Topics:
    - inference
    - architecture
    - events
    - webchat
    - tui
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/context.go
      Note: Context sink plumbing (post-WithSink world)
    - Path: geppetto/pkg/inference/core/session.go
      Note: Old Session runner to be removed
    - Path: geppetto/pkg/inference/engine/options.go
      Note: engine.WithSink to be deleted
    - Path: geppetto/pkg/inference/state/state.go
      Note: Old InferenceState to be replaced
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Canonical RunToolCallingLoop used by standard builder
    - Path: pinocchio/pkg/ui/backend.go
      Note: TUI migration target
    - Path: pinocchio/pkg/webchat/router.go
      Note: Webchat migration target
ExternalSources: []
Summary: 'Replace InferenceState/core.Session/engine.WithSink with a single Session+ExecutionHandle model: SessionID multi-turn state + EngineBuilder that runs a tool-loop inference runner.'
LastUpdated: 2026-01-21T19:15:00-05:00
WhatFor: Supersede MO-005 (sink cleanup) and MO-006 (cancellation lifecycle) with a single coherent Session/Execution abstraction.
WhenToUse: Before implementing MO-007 or refactoring callers (pinocchio TUI/webchat, examples, moments) to the new session model.
---


# Session refactor: SessionID + EngineBuilder + ExecutionHandle

## Executive Summary

We want one clean abstraction that supersedes:

- `InferenceState` (which mixes conversation state with in-flight inference lifecycle),
- `core.Session` (runner/config bag that is not a “session” in the product sense),
- `engine.WithSink` (engine-config sinks) and the provider-engine “bridge sinks into ctx” glue,
- the confusing “StartRun/FinishRun/HasCancel” lifecycle split (`RunInference` vs `RunInferenceStarted`).

This design proposes:

1) A **Session** (with `SessionID`) representing a long-lived multi-turn interaction.
2) A **Session.EngineBuilder** that returns an object which can perform inference via:
   - `StartInference(ctx, turn) -> (turn, error)`
3) A **Session.StartInference(ctx) -> (ExecutionHandle, error)** that:
   - instantiates an engine via the builder,
   - starts inference on the session’s current input turn,
   - returns a handle to **cancel** and **wait**.
4) A standard **EngineBuilder** implementation that takes:
   - tool registry + tool config,
   - middleware chain,
   - snapshot hook,
   - turn persister,
   - event sinks,
   and whose `StartInference` triggers the canonical tool-calling loop.

We explicitly do **not** keep backwards compatibility; the migration is “change it all at once”.

## Problem Statement

Today we have overlapping, confusing concepts and multiple ways to “start inference”:

- “Session” (`geppetto/pkg/inference/core.Session`) is a runner/config bag, not a multi-turn session abstraction.
- “InferenceState” stores both long-lived state (engine + turn) and short-lived lifecycle state (running + cancel).
- Cancellation uses a mix of `StartRun/FinishRun/SetCancel/HasCancel`, with two runner entrypoints (`RunInference` vs `RunInferenceStarted`).
- Event sinks exist in both engine config (`engine.WithSink`) and context (`events.WithEventSinks`), with bridging code to avoid missing events and a persistent risk of duplicate delivery.

These problems show up as real bugs and friction:

- Web/TUI can hang in “generating” if cancellation doesn’t produce an explicit terminal signal.
- Developers misunderstand “RunID” vs “Conversation” vs “Inference”, because the naming is overloaded.
- Call sites have to choose between multiple plumbing strategies, leading to drift and duplication.

## Proposed Solution

### Vocabulary

- **Session**: long-lived multi-turn state (what product engineers call a conversation).
- **Inference**: a short-lived, cancelable execution that advances the session.
- **ExecutionHandle**: the concrete handle to cancel/wait a running inference.

### Core interfaces and structs

#### Session

Session represents a long-lived sequence of interactions.

Key properties:

- `SessionID` is stable (used for persistence, tracing, routing).
- Session stores its multi-turn state as a list of Turns (append-only snapshots).
- Session enforces “one active inference at a time”.

Proposed shape:

```go
type Session struct {
    SessionID string
    Turns []*turns.Turn

    // Builder produces a runner for this session (tools/middleware/sinks configured outside Session).
    Builder EngineBuilder

    mu sync.Mutex
    active *ExecutionHandle
}
```

We deliberately avoid storing extra profile metadata inside Session. Profiles/middleware selection live outside and are compiled into the EngineBuilder implementation.

#### EngineBuilder and InferenceRunner

We separate building from running:

```go
type EngineBuilder interface {
    Build(ctx context.Context, sessionID string) (InferenceRunner, error)
}

type InferenceRunner interface {
    StartInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

This keeps the core API extremely small: “given a Turn, return a new Turn”.

#### ExecutionHandle

ExecutionHandle represents one in-flight inference.

```go
type ExecutionHandle struct {
    SessionID   string
    InferenceID string

    Input *turns.Turn

    done chan struct{}

    mu sync.Mutex
    cancel context.CancelFunc
    out *turns.Turn
    err error
}

func (h *ExecutionHandle) Cancel()
func (h *ExecutionHandle) Wait() (*turns.Turn, error)
func (h *ExecutionHandle) IsRunning() bool
```

#### Session.StartInference

Signature per requirement:

```go
func (s *Session) StartInference(ctx context.Context) (*ExecutionHandle, error)
```

Behavior:

1) Claim the right to run (reject if already active).
2) Determine input turn: `input := s.Latest()` (typically the latest Turn already includes the new user prompt block).
3) Build runner: `runner := s.Builder.Build(ctx, s.SessionID)`.
4) Create `runCtx, cancel := context.WithCancel(ctx)`.
5) Spawn goroutine that calls `runner.StartInference(runCtx, input)` and stores result.
6) On success, append the returned turn to `s.Turns`.
7) Close `done`.

Minimal helpers:

```go
func (s *Session) Latest() *turns.Turn
func (s *Session) Append(t *turns.Turn)
func (s *Session) CancelActive() error
```

### Standard EngineBuilder: tool-loop runner

We provide a standard EngineBuilder implementation that runs the tool-calling loop.

Inputs (by requirement):

- registry
- middlewares
- snapshot hook
- turn persister
- event sinks
- tool config

Design sketch:

```go
type ToolLoopEngineBuilder struct {
    // builds a provider engine with middleware chain already applied
    EngineFactory EngineFactoryLike

    Registry     tools.ToolRegistry
    ToolConfig   toolhelpers.ToolConfig

    EventSinks   []events.EventSink
    SnapshotHook toolhelpers.SnapshotHook
    Persister    TurnPersister
}
```

Build returns an `InferenceRunner` whose `StartInference`:

1) attaches sinks and snapshot hook to ctx (context sinks only; no engine.WithSink),
2) runs the tool loop (`toolhelpers.RunToolCallingLoop`) with registry + tool config,
3) persists the final turn via Persister (and optionally snapshots via hook),
4) returns the final updated Turn.

Pseudo:

```go
func (r *ToolLoopRunner) StartInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    runCtx := ctx
    runCtx = events.WithEventSinks(runCtx, r.EventSinks...)
    if r.SnapshotHook != nil {
        runCtx = toolhelpers.WithTurnSnapshotHook(runCtx, r.SnapshotHook)
    }
    updated, err := toolhelpers.RunToolCallingLoop(runCtx, r.Engine, t, r.Registry, r.ToolConfig)
    if err == nil && r.Persister != nil && updated != nil {
        _ = r.Persister.PersistTurn(runCtx, r.SessionID, updated)
    }
    return updated, err
}
```

This eliminates engine-config sinks and unifies tool-loop + engine events through the same context sinks.

## Design Decisions

### Decision: “Session” is the product abstraction

We use Session to mean “multi-turn long running interaction”. This replaces the overloaded use of “Run” for chat state.

### Decision: “Inference is cancelable” via ExecutionHandle

We model cancellation as a property of an in-flight inference execution, not of the session itself.

### Decision: event sinks are only injected via context

The standard builder/runner owns sink injection (`events.WithEventSinks`) and we remove `engine.WithSink`.

### Decision: no backwards compatibility

We remove old APIs and migrate call sites in one refactor step to avoid carrying conceptual debt forward.

## Alternatives Considered

### Keep InferenceState/core.Session and only rename methods

Rejected: keeps the blended responsibilities (conversation + lifecycle) and retains the two-entrypoint runner split.

### Merge runner config into Session state

Rejected: makes Session mutable in more dimensions (tools/sinks/hooks) and increases concurrency hazards; instead we keep those in the EngineBuilder.

### Keep engine.WithSink as a convenience

Rejected: reintroduces ambiguity and increases duplicate delivery risk; tool loops/middleware already publish via ctx sinks.

## Implementation Plan

This is a step-by-step plan to refactor the codebase. No compatibility layer is required.

### Phase 1: Implement new core types in geppetto

1) Add new package(s):
   - `geppetto/pkg/inference/session` (Session + ExecutionHandle)
   - `geppetto/pkg/inference/runner` (EngineBuilder + InferenceRunner + standard ToolLoopEngineBuilder)
2) Add unit tests:
   - execution cancellation semantics
   - sink injection semantics (no duplicates)
   - tool-loop runner uses registry and returns updated turn

### Phase 2: Migrate geppetto examples

Update `geppetto/cmd/examples/*` to use:

- Session + standard EngineBuilder
- Session.StartInference(ctx) + handle.Wait()

Validate using the existing playbook:

- `geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--.../reference/02-playbook-testing-inference-via-geppetto-pinocchio-examples.md`

### Phase 3: Migrate pinocchio TUI and webchat

- Replace `InferenceState` usage with `Session`.
- UI stores `ExecutionHandle` while generating; cancel calls `handle.Cancel()`.
- Webchat start returns 409 if there is an active handle.

### Phase 4: Remove old APIs

Delete:

- `geppetto/pkg/inference/state` (InferenceState)
- `geppetto/pkg/inference/core/session.go` (old runner Session)
- `geppetto/pkg/inference/engine/options.go` and all `engine.WithSink` usage
- provider-engine code paths that bridge config sinks into ctx

Update docs and the playbook where it referenced old APIs.

### Phase 5: Migrate moments/go-go-mento (follow-up)

Moments’ prompt resolver remains external; it supplies a builder that composes the correct middleware chain and tool executor behavior.

## Open Questions

1) How does Session choose its input Turn for StartInference?
   - Minimal: callers append user blocks and update session state prior to calling StartInference.
   - Better: Session exposes `AppendUserPrompt(prompt)` which creates a new Turn snapshot and makes it the latest.

2) Do we emit an explicit interrupt event on cancel to prevent UI hangs?
   - Likely yes, in the standard runner wrapper when `errors.Is(err, context.Canceled)`.

## References

- MO-005 sink cleanup analysis:
  - `geppetto/ttmp/2026/01/20/MO-005-CLEANUP-SINKS--cleanup-engine-withsink-usage-move-sink-wiring-to-context-session/analysis/01-sink-cleanup-removing-engine-withsink-and-standardizing-on-context-session-sinks.md`
- MO-006 cancellation lifecycle analysis + compendium:
  - `geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/analysis/01-run-vs-conversation-vs-inference-lifecycle-cancellation-and-ownership.md`
  - `geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/analysis/02-compendium-sinks-sessions-conversation-state-lifecycle-engines-tool-loops-q-a-diagrams.md`
