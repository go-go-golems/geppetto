---
Title: 'Compendium: sinks, sessions, conversation state, lifecycle, engines, tool loops (Q&A + diagrams)'
Ticket: MO-006-CLEANUP-CANCELLATION-LIFECYCLE
Status: active
Topics:
    - inference
    - architecture
    - events
    - webchat
    - tui
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/context.go
      Note: Context sink plumbing used throughout the Q&A
    - Path: geppetto/pkg/inference/core/session.go
      Note: Session runner patterns (RunInference vs RunInferenceStarted)
    - Path: geppetto/pkg/inference/state/state.go
      Note: InferenceState stores conversation state + lifecycle today
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Canonical RunToolCallingLoop signature
    - Path: go-go-mento/go/pkg/webchat/router.go
      Note: go-go-mento cancel path emits interrupt on context.Canceled
    - Path: moments/backend/pkg/webchat/loops.go
      Note: Moments custom tool loop (step mode + explicit tool events)
    - Path: pinocchio/pkg/ui/backend.go
      Note: TUI lifecycle/cancel pattern
    - Path: pinocchio/pkg/webchat/router.go
      Note: Webchat lifecycle/cancel pattern
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-21T12:55:00-05:00
WhatFor: Single comprehensive reference capturing the project’s key Q&A about sinks, sessions, conversation state, lifecycle/cancellation, engines, and tool loops (with diagrams and pseudocode).
WhenToUse: When onboarding, reviewing refactors (MO-004/005/006), or debugging missing/duplicate events, hanging UIs, or cancellation semantics.
---


## Preface (context and scope)

This document consolidates a set of practical architecture questions and answers that came up while implementing and debugging:

- OpenAI Responses streaming behavior and strict input validation,
- event sink plumbing (Watermill vs context/session sinks),
- multi-turn “chat” state management and turn construction,
- tool-calling loops (single-pass vs multi-step inference),
- cancellation and “run lifecycle” semantics across TUI and webchat.

The intent is Norvig-style clarity:

1) define terms precisely,
2) specify invariants and state machines,
3) show sequence/timing diagrams,
4) show pseudocode that matches current code,
5) propose a naming/organization scheme that reduces confusion.

### Primary reference implementations (current code)

- geppetto (shared inference primitives):
  - `geppetto/pkg/inference/state/state.go` (current `InferenceState`)
  - `geppetto/pkg/inference/core/session.go` (current `Session` runner)
  - `geppetto/pkg/inference/toolhelpers/helpers.go` (tool loop core)
  - `geppetto/pkg/events/context.go` (context sink plumbing)
  - `geppetto/pkg/inference/middleware/sink_watermill.go` (Watermill sink)
  - provider engines publish via context (e.g. `geppetto/pkg/steps/ai/openai_responses/engine.go`)

- pinocchio (TUI + webchat):
  - `pinocchio/pkg/ui/backend.go` (Bubble Tea runner pattern)
  - `pinocchio/pkg/ui/runtime/builder.go` (chat program builder)
  - `pinocchio/pkg/webchat/router.go` (webchat run loop pattern)
  - `pinocchio/pkg/webchat/loops.go` (thin wrapper around tool loop)

- moments and go-go-mento (webchat patterns):
  - `moments/backend/pkg/webchat/router.go`, `moments/backend/pkg/webchat/loops.go`
  - `go-go-mento/go/pkg/webchat/router.go`, `go-go-mento/go/pkg/webchat/loops.go`

### Ticket context

This compendium synthesizes the work and analysis in:

- MO-004: “Unify InferenceState + Session + EngineBuilder”
- MO-005: “Cleanup sinks (remove engine.WithSink; use Session/context sinks)”
- MO-006: “Cleanup cancellation lifecycle semantics (Run vs Conversation vs Inference)”

## 0) Glossary (terms we will use consistently)

### Conversation

Long-lived interaction state (a chat thread/session). It persists across multiple user prompts and multiple inference executions.

### Inference (execution)

A short-lived, cancelable computation that advances the conversation state by calling the model (and possibly tools). From the UI perspective, “generating” corresponds to an active inference.

### Turn (in this codebase)

In `geppetto/pkg/turns/types.go`, a `Turn` is a container of ordered **Blocks**:

- `Turn.Blocks []Block` is the canonical history representation in the current implementation.
- `Turn.RunID` is currently used as a “conversation-like identifier” in many chat contexts, which is a source of confusion.

In other words: **the current system models multi-turn chat as one growing Turn** (append blocks over time), not `[]Turn`.

### Tool loop (tool-calling loop)

A higher-level inference loop that may call the provider engine multiple times and execute tools between calls until the model stops requesting tool calls or an iteration limit is reached.

## 1) Q&A: where is “the sequence of turns” kept?

### Q: Where does the sequence of turns live? Do we even have a `[]Turn`?

**A (current code):** We do *not* keep an explicit `[]Turn` in the geppetto/pinocchio inference core. The “history” is the ordered list of blocks inside a single `*turns.Turn`.

Evidence:

- `geppetto/pkg/turns/types.go` defines:

```go
type Turn struct {
    ID     string
    RunID  string
    Blocks []Block
    Metadata Metadata
    Data Data
}
```

- `geppetto/pkg/inference/state/state.go` stores one `Turn`:

```go
type InferenceState struct {
    RunID string
    Turn  *turns.Turn
    Eng   engine.Engine
    // plus running/cancel fields
}
```

As a result, “multi-turn” is represented by repeatedly appending blocks:

- append user block,
- engine adds assistant blocks (and maybe tool_call blocks),
- tool loop appends tool result blocks,
- repeat for next user message.

### Q: What if we want “each inference produces a new Turn”?

**A (proposed model):** Then `ConversationState` should store `[]Turn` (or `[]*Turn`), and each inference execution returns a new Turn (immutable snapshot) derived from previous state + input.

That change is a major semantic shift but removes a lot of confusion:

- It makes “turn” correspond to “one inference execution”.
- It makes persistence/debugging easier (append-only snapshots).

## 2) Q&A: what are “config sinks” vs “context sinks”? Can we remove engine.WithSink?

### Q: What are “config sinks” in the engine used for?

**A:** Engine-config sinks are the legacy injection point:

- `engine.WithSink(sink)` stores sinks in `engine.Config.EventSinks`.
- Historically, provider engines published directly to those sinks.

This was convenient for call sites that construct engines via factories and want event routing “baked into” the engine instance.

### Q: What are “context sinks” used for?

**A:** Context sinks are the unified, composable event routing mechanism:

- `events.WithEventSinks(ctx, sinks...)` attaches sinks to the context.
- `events.PublishEventToContext(ctx, ev)` publishes to all sinks on the context.

Tool loops and middleware publish via context because they should not depend on engine configuration types.

### Q: Why do we still have both?

**A:** Compatibility. Many call sites still create engines with `engine.WithSink(...)`.

To unify publishing, provider engines now bridge engine-config sinks into context at the start of `RunInference` (so tool-loop/middleware events also reach those sinks).

### Q: Should we remove engine.WithSink?

**A:** Yes, it’s a good cleanup direction, but only if we move sink ownership to `core.Session` (or explicitly wrap contexts) and ensure we do not accidentally duplicate sinks across layers.

Key pitfalls:

- `events.WithEventSinks` is append-only; attaching the same sink multiple times yields duplicates.
- Long-lived base contexts carrying sinks can accumulate duplicates across runs.

Recommended invariant:

> Base contexts are sinkless; each inference creates a fresh run context and attaches sinks exactly once (preferably by Session/Runner).

See MO-005 for a complete inventory and migration plan.

## 3) Q&A: what is PinocchioCommand runner and how does it do inference?

### Q: What is PinocchioCommand runner for and where is it used?

**A:** `pinocchio/pkg/cmds/cmd.go` defines `PinocchioCommand`, which bridges:

```
CLI args + parsed layers + templates -> seed Turn -> inference run -> output or TUI/webchat
```

It supports:

- blocking mode (run once, print output),
- interactive mode (run once, then optionally enter chat),
- chat mode (Bubble Tea program).

### Q: How does it do inference (blocking)?

Simplified logic:

1) Create engine via factory.
2) Construct seed Turn (system prompt + blocks + user prompt).
3) Call `engine.RunInference(ctx, seed)`.

Historically it used `engine.WithSink(watermillSink)` so events flow to the router; MO-005 proposes moving sink wiring to context/session.

### Q: How does it do inference (chat)?

It creates and runs an event router and a Bubble Tea program.

The actual per-prompt inference in chat mode is performed by the UI backend (`pinocchio/pkg/ui/backend.go`) which uses `InferenceState` + `core.Session`.

## 4) Q&A: why do we have RunInference vs RunInferenceStarted? Why HasCancel?

### Q: Why both RunInference and RunInferenceStarted?

**A:** There are two real-world start patterns:

#### Pattern A: “Runner owns lifecycle” (blocking, tests)

- `Session.RunInference(ctx, seed)` does:
  - `StartRun()`
  - create `runCtx` and store cancel
  - defer `FinishRun()`
  - run inference

#### Pattern B: “UI claims run first” (Bubble Tea, HTTP)

- UIs often must reject immediately if already running (before starting a goroutine or returning a `tea.Cmd`).
- So the UI calls `StartRun()` itself, stores a cancel func, and then calls `RunInferenceStarted` later.

This is used in:

- pinocchio TUI: `pinocchio/pkg/ui/backend.go`
- pinocchio webchat: `pinocchio/pkg/webchat/router.go`

### Q: Why HasCancel?

**A:** `RunInferenceStarted` must not overwrite a cancel func that the UI stored earlier; otherwise UI interrupt/cancel would stop working.

So `RunInferenceStarted` creates a cancel only if missing.

### Q: Can we unify the two run entrypoints?

**A:** Yes; one robust approach is to replace “StartRun + SetCancel + FinishRun” with an atomic `StartInference(...)` that returns `(runCtx, finish)` (or an `Execution` handle). Then the runner only has one entrypoint that assumes lifecycle is already structured.

MO-006 documents this direction in detail.

## 5) Q&A: cancellation semantics — what happens when inference stops?

### Q: Who can cancel inference?

**A:** Any upstream component that holds the `Execution` cancel handle (or can access it from the conversation state) can cancel. Today, cancellation is implemented as context cancellation.

Examples:

- Pinocchio TUI: `Interrupt()` calls `inf.CancelRun()`.
- Webchat: cancel endpoints call `conv.CancelRun()` (go-go-mento) or `conv.cancel()` (moments).

### Q: What must happen on cancel to avoid UI hangs?

**A:** Cancellation must be observable downstream.

If a UI is rendering “generating…” based on `start`/`delta` events, it needs a terminal signal:

- final response, or
- error event, or
- interrupt/cancel event.

Otherwise it may remain stuck.

go-go-mento webchat explicitly emits an interrupt event when an inference ends with `context.Canceled` (see `go-go-mento/go/pkg/webchat/router.go`).

This is a key invariant for robust UIs:

> If inference terminates early (canceled), emit an explicit terminal event so consumers can finalize the in-flight UI entity.

## 6) Q&A: would it make sense to merge Session and InferenceState?

### Q: Should we merge Session and InferenceState?

**A:** Not as-is. It can reduce boilerplate but risks conflating:

- mutable conversation state (history + engine),
- runner configuration (tools/sinks/hooks/persist),
- in-flight inference lifecycle (cancel/running).

Given MO-005’s direction (sinks should live on Session/Runner, not on mutable state), keeping them separate is generally cleaner.

Better compromise:

- Keep `ConversationState` minimal and focused on history + ids + engine builder/config.
- Keep `Runner`/`Session` focused on “how to run inference” (tools/sinks/policies).
- Introduce an explicit `Execution` handle to represent a cancelable inference.

## 7) Q&A: proposed structs (state, runner, execution)

This section reflects an evolution of the discussion: we want to reduce ambiguity by making “conversation vs inference” explicit.

### Proposed “three-struct model”

#### ConversationState (long-lived; holds history)

Two variants:

**Variant A (current system): one growing Turn**

```go
type ConversationState struct {
    ConversationID string
    Turn *turns.Turn
}
```

**Variant B (your preferred model): list of Turns (one per inference)**

```go
type ConversationState struct {
    ConversationID string
    Turns []*turns.Turn // append-only snapshots
}
```

#### Runner (policy + mechanics)

```go
type Runner struct {
    Tools        tools.ToolRegistry
    ToolConfig   toolhelpers.ToolConfig
    EventSinks   []events.EventSink
    SnapshotHook toolhelpers.SnapshotHook
    Persister    TurnPersister
}
```

#### Execution (cancelable inference)

```go
type Execution struct {
    InferenceID string
    Input  *turns.Turn
    Output *turns.Turn
    Err    error

    ctx    context.Context
    cancel context.CancelFunc
    done   chan struct{}
}
```

### Where EngineBuilder and Engine fit

Preferred placement (especially if we adopt `[]Turn` snapshots):

- `EngineBuilder` belongs to the conversation/profile configuration layer (because engines may need rebuilding when profile/middlewares/tools/sinks change).
- `Engine` is constructed per inference execution (or is a stable per-conversation engine if guaranteed not to drift).

## 8) Q&A: where does tool loop fit? What are the signatures?

### Q: What is the signature of Engine?

In geppetto:

```go
type Engine interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

### Q: What is the signature of the tool-calling loop?

The canonical helper (Turn-based) is:

`geppetto/pkg/inference/toolhelpers/helpers.go`:

```go
func RunToolCallingLoop(
    ctx context.Context,
    eng engine.Engine,
    initialTurn *turns.Turn,
    registry tools.ToolRegistry,
    config ToolConfig,
) (*turns.Turn, error)
```

Webchat packages sometimes wrap or customize it:

- pinocchio wrapper: `pinocchio/pkg/webchat/loops.go`
- moments custom loop: `moments/backend/pkg/webchat/loops.go` (step mode, explicit tool events)
- go-go-mento custom loop: `go-go-mento/go/pkg/webchat/loops.go` (step mode, persistence hooks)

### Q: Where should tool loop live in the (state/runner/execution) model?

**A:** In the Runner. The tool loop is part of “how inference is executed”, not conversation state.

So:

- Runner decides whether to run single-pass or tool loop based on whether a registry/config is present.
- Execution captures the input Turn and the output Turn (or error/canceled).
- ConversationState stores the resulting Turn snapshot(s).

## 9) Diagrams: end-to-end sequences (TUI, webchat)

### 9.1 Pinocchio TUI submission (Bubble Tea)

```
time →

UI thread:  Start(prompt)
             | StartRun()  (reject if already running)
             | runCtx, cancel := WithCancel(ctx)
             | SetCancel(cancel)
             | return tea.Cmd ----------------------------------------------+
                                                                           |
tea.Cmd:                                                                    |
             +--> seed := snapshot + append user block                      |
             |    sess.RunInferenceStarted(runCtx, seed)                    |
             |    (events published to sinks)                               |
             +--> defer cancel(); FinishRun() -----------------------------+
```

### 9.2 Webchat submission (HTTP handler + goroutine)

```
HTTP handler: POST /chat
              | StartRun() else 409
              | spawn goroutine -------------------------------------------+
                                                                          |
goroutine:                                                                 |
              +--> runCtx, cancel := WithCancel(baseCtx)                    |
              |    SetCancel(cancel)                                        |
              |    sess.RunInferenceStarted(runCtx, seed)                   |
              +--> defer cancel(); FinishRun() ----------------------------+
```

## 10) Recommendations (organization and naming to reduce confusion)

If the product intent is “chat bots and agents”, and you want the words to match realities:

### Rename the long-lived object

- `InferenceState` → `ConversationState` (or just `Conversation`)

because it primarily stores conversation state (history + engine config), not “inference”.

### Rename the short-lived lifecycle operations

- `StartRun/FinishRun/CancelRun` → `StartInference/FinishInference/CancelInference`

because the cancelable thing is the inference execution, not the conversation.

### Stop overloading RunID in chat contexts

- Prefer `ConversationID` for chat-thread identity.
- Optionally add `InferenceID` for traceability.

### Collapse lifecycle split into an Execution handle

Instead of:

```
StartRun()
SetCancel(cancel)
RunInferenceStarted(...)
FinishRun()
```

prefer:

```go
exec, err := runner.StartInference(ctx, conversation, seed)
// exec.Cancel() possible immediately
updated, err := exec.Wait()
```

This makes ownership explicit, reduces “forget to call FinishRun” errors, and makes UI cancellation paths obvious.

## Appendix A: “Why strict providers exposed ordering bugs”

Providers like OpenAI Responses perform strict validation on input item ordering and pairing. When history mutation logic produces an invalid sequence, requests fail with 400s that are hard to interpret unless you can snapshot and inspect the exact constructed history.

This is why:

- snapshot hooks are valuable,
- history mutation must be deterministic and idempotent,
- and it’s important to clearly define where history lives and how it is advanced.

## Appendix B: What to read next

- MO-004 analysis: sinks vs context sinks; PinocchioCommand runner; Session lifecycle
  - `geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/analysis/03-analysis-engine-sinks-vs-context-sinks-pinocchiocommand-runner-session-run-lifecycle-startrun-finishrun-hascancel.md`

- MO-005 analysis: remove engine.WithSink; use Session/context sinks
  - `geppetto/ttmp/2026/01/20/MO-005-CLEANUP-SINKS--cleanup-engine-withsink-usage-move-sink-wiring-to-context-session/analysis/01-sink-cleanup-removing-engine-withsink-and-standardizing-on-context-session-sinks.md`

- MO-006 analysis: run vs conversation vs inference; cancellation semantics
  - `geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/analysis/01-run-vs-conversation-vs-inference-lifecycle-cancellation-and-ownership.md`
