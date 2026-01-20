---
Title: 'Postmortem: InferenceState + Session + EngineBuilder Unification'
Ticket: MO-004-UNIFY-INFERENCE-STATE
Status: active
Topics:
    - inference
    - architecture
    - webchat
    - prompts
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/builder/builder.go
      Note: EngineBuilder interface
    - Path: geppetto/pkg/inference/core/session.go
      Note: Session runner (RunInference/RunInferenceStarted)
    - Path: geppetto/pkg/inference/state/state.go
      Note: InferenceState lifecycle and cancel
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Provider sink attachment and context publishing
    - Path: pinocchio/pkg/ui/backend.go
      Note: TUI backend run orchestration via Session
    - Path: pinocchio/pkg/webchat/router.go
      Note: Webchat run orchestration via Session
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-20T15:30:59.730629874-05:00
WhatFor: ""
WhenToUse: ""
---


# Postmortem: MO-004 Inference Core Unification (InferenceState + Session + EngineBuilder)

## Executive summary

This implementation unified the “inference core” (what owns the engine, what owns the accumulated turn snapshot, how the tool-loop is orchestrated, and how events are emitted) across:

- `geppetto` (shared inference foundation)
- `pinocchio` (TUI + webchat + an agent example)

The central changes are:

1. Introduced **geppetto-owned inference primitives**:
   - `state.InferenceState` — long-lived engine + run lifecycle + current `*turns.Turn`.
   - `core.Session` — the minimal runner that executes inference (single-pass or tool-loop), updates `InferenceState`, wires cancellation, and optionally attaches sinks/hooks.
   - `builder.EngineBuilder` — a stable engine composition seam.
2. Standardized **event emission** by making provider engines attach their configured sinks into the run context and publish only via context, so engine events and tool-loop events share one path.
3. Migrated pinocchio’s callers away from local “runner/conversation-state” implementations and toward the shared geppetto core.

Net effect: fewer “turn reconstruction” footguns, fewer ordering/validation bugs (notably with strict provider validation like OpenAI Responses), and fewer per-UI custom run loops.

## Scope and non-scope

### In scope

- Create a stable geppetto “inference core” surface that:
  - is UI-agnostic (TUI/CLI/webchat)
  - supports both single-pass inference and tool loops
  - supports cancellation in a consistent way
  - supports event streaming in a consistent way
- Migrate:
  - geppetto `cmd/examples/*` to the new core
  - pinocchio TUI (`pkg/ui`) to the new core
  - pinocchio webchat (`pkg/webchat`) to the new core
  - pinocchio agent example (`cmd/agents/simple-chat-agent`) to the new core

### Explicitly deferred

- `go-go-mento` webchat migration: it’s currently pinned to older APIs and pulling it into the workspace caused broad unrelated compile failures. This should be handled as a separate “port go-go-mento up to current geppetto” effort before attempting migration.
- `moments` migration: an analysis doc exists for how to migrate `moments/backend/pkg/webchat/router.go` and friends, but implementation is intentionally deferred until the pinocchio + examples path is stable.

## Timeline / main commits

This is intentionally “commit-first”, since code review/archaeology is easiest from the hashes:

- `453e6af`: Introduce geppetto inference core (`InferenceState`, `core.Session`, `builder.EngineBuilder`).
- `3206cef`: Provider engines attach configured sinks into context and publish via context only.
- `e009123`: Migrate geppetto `cmd/examples/*` to EngineBuilder + InferenceState + Session.
- `03a3043`: Migrate pinocchio agent example to EngineBuilder + InferenceState + Session.
- `550b073`: Migrate pinocchio TUI + webchat to geppetto InferenceState + Session.
- `1a835e5`: Remove obsolete pinocchio local runner package.

Ticket-level narrative is tracked in:

- `reference/01-diary.md` (step-by-step diary with commands and failures)
- `changelog.md` (summary, commit hashes, and file notes)

## What we introduced (new primitives and why)

### 1) `geppetto/pkg/inference/state.InferenceState`

**File:** `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/state/state.go`

**Responsibilities**

- Own the long-lived values that define “an inference session”:
  - `Eng engine.Engine` — the composed provider engine (possibly middleware-wrapped).
  - `Turn *turns.Turn` — the current canonical “snapshot” of the conversation state.
  - `RunID string` — per-session run identity used in event metadata filtering.
- Own run lifecycle:
  - `StartRun()` / `FinishRun()` — single-run-at-a-time guard.
  - `SetCancel()` / `CancelRun()` — cancellation surface for UIs.

**Key design decision:** the UI should not be reconstructing turns from rendered state. The inference state is canonical.

### 2) `geppetto/pkg/inference/core.Session` + `core.Runner`

**File:** `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/core/session.go`

**Responsibilities**

- Provide one API:
  - `RunInference(ctx, seed) (*turns.Turn, error)`
- Optionally run in tool-loop mode if a registry is provided:
  - `toolhelpers.RunToolCallingLoop(ctx, Eng, seed, Registry, ToolConfig)`
- Attach consistent “shared behaviors”:
  - attach sinks to context (`events.WithEventSinks`)
  - attach snapshot hook (`toolhelpers.WithTurnSnapshotHook`)
  - cancellation wiring (store cancel function in InferenceState)
  - persist updated state back to `InferenceState.Turn`

**Added compromise:** `RunInferenceStarted(ctx, seed)` exists.

Why it exists:

- Some UIs need to mark “running” before returning an async command (Bubble Tea), or before launching a goroutine (HTTP handlers) to avoid a “double start” race.

This introduces more surface area than a pure `RunInference` interface, but it prevented breaking the Bubble Tea “Start returns a Cmd” pattern.

### 3) `geppetto/pkg/inference/builder.EngineBuilder`

**File:** `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/builder/builder.go`

**Responsibilities**

- Define a stable seam where UIs can say:
  - “I have a profile slug and overrides; give me an engine + sink + a comparable config signature”
- Avoid injecting app lifecycle concerns (no ConversationManager injection).

This interface is intentionally small and slightly “opaque”:

```text
Build(convID, profileSlug, overrides) -> (engine, sink, EngineConfig, error)
BuildConfig(profileSlug, overrides) -> EngineConfig
BuildFromConfig(convID, config) -> (engine, sink, error)
```

In practice, this gives us a canonical place to encode profile/override rules without duplicating them in multiple UIs.

## What we modified (call sites and providers)

### 1) Provider engines: unify event publishing

**Files**

- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/engine.go`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai/engine_openai.go`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/claude/engine_claude.go`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/gemini/engine_gemini.go`

**Before**

There were effectively two different event paths:

- Engine publishes some events to the context sink (via `events.PublishEventToContext`)
- Tool loop publishes to context sinks
- Engine also publishes to “config sinks” (engine options), bypassing context

This created a bad choice for shared orchestration:

- If Session attaches sinks to context, tool-loop events are emitted, but engine events might be double-emitted or inconsistent depending on config.
- If Session does not attach sinks, tool-loop events can be silently dropped if the UI relied on engine config sinks only.

**After**

Provider engines now attach their configured sinks to the run context early in `RunInference`, and publish events only through the context mechanism.

This means:

- Tool-loop and provider streaming events share one path.
- UIs can choose a single strategy and stick with it:
  - “configure sinks on engine” OR “attach sinks on session/context”

### 2) geppetto cmd/examples: moved to the new core

**Files**

- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/internal/examplebuilder/builder.go`
- Multiple `geppetto/cmd/examples/*/main.go`

**What changed**

- Examples now instantiate a minimal EngineBuilder and always run inference via `InferenceState` + `core.Session`.
- Tool-loop examples (generic tool calling) now use `Session.Registry` + `Session.ToolConfig` rather than directly calling `toolhelpers.RunToolCallingLoop`.

This provides working reference code for downstream consumers.

### 3) pinocchio: TUI and webchat now use shared core

**TUI**

- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/ui/backend.go`

Key changes:

- `EngineBackend` now stores `*state.InferenceState` (`e.inf`) instead of a `ConversationState`.
- Seed construction is explicit: clone stored snapshot turn + append user text.
- Cancellation uses `InferenceState.CancelRun()`.
- Uses `core.Session.RunInferenceStarted` to avoid Bubble Tea’s “double start” race.

**Webchat**

- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/conversation.go`

Key changes:

- Conversation stores `Inf *state.InferenceState` instead of `State *conversation.ConversationState`.
- The `/chat` handler calls `conv.Inf.StartRun()` *before* launching the goroutine to prevent concurrent runs.
- Seed construction is explicit (`seedForPrompt`): clone `conv.Inf.Turn` + append prompt.
- Tool-loop execution uses `core.Session` with `Registry + ToolConfig`, and `EventSinks` includes the `WatermillSink`.

### 4) pinocchio: agent example moved to new core + EngineBuilder

**Files**

- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/inference/enginebuilder/parsed_layers.go`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/agents/simple-chat-agent/main.go`

Key changes:

- Tool-loop backend now uses:
  - `InferenceState` as the owner of `Turn` + cancel/running state
  - `core.Session` as the executor (tool-loop mode)
- Added a pinocchio-local ParsedLayers EngineBuilder (a minimal implementation).

### 5) pinocchio cleanup: removed obsolete runner package

- Deleted `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/inference/runner/runner.go`

This was intentionally “no backwards compatibility alias”: once migrated, the old path was removed to prevent regression and drift.

## Key architecture changes (before vs after)

### Before: per-UI inference orchestration patterns drifted

```text
pinocchio TUI:
  UI state -> reduceHistory() -> seed Turn -> engine.RunInference()

pinocchio webchat:
  ConversationState.SnapshotForPrompt -> seed Turn -> runner.Run() -> toolhelpers.RunToolCallingLoop()

agent example:
  own loop + own cancel/running tracking

providers:
  tool loop publishes events to ctx; engines publish to ctx + config sinks (double-path)
```

### After: shared core drives runs, UIs are downstream

```text
UI layer (TUI / web / agent)
  |
  | build seed = clone(InferenceState.Turn) + append(prompt block)
  v
core.Session.RunInference*
  |
  | (optional tool loop)
  v
engine.RunInference  -> emits events via context sinks
  |
  v
InferenceState.Turn = updatedTurn
```

`RunInference*` means either `RunInference` or `RunInferenceStarted`, depending on whether the UI needs to mark “running” before deferring execution.

## Issues encountered and how they were handled

### 1) Event sinks “double-path” caused either missing tool events or duplicate publishing

Symptom:

- Tool-loop events (tool call/execute/result) did not reliably appear in all UIs unless the UI also attached sinks to context.
- Attaching sinks at both levels could duplicate provider events.

Fix:

- Make engines attach their configured sinks into the run context and publish via context only (see commit `3206cef`).

Tradeoff:

- The “canonical” sink wiring strategy needs to be documented per app (engine-config sinks vs session-attached sinks). Both now work, but mixing can duplicate.

### 2) Bubble Tea “Start returns Cmd” requires marking the run started before execution

Symptom:

- `Start()` can be called again before the returned command runs, if the “running” flag isn’t set early.

Fix:

- Added `Session.RunInferenceStarted(...)`.
- In Bubble Tea backends, do:
  - `InferenceState.StartRun()` before returning the Cmd
  - `RunInferenceStarted` inside the Cmd
  - `FinishRun` in defer inside the Cmd

Tradeoff:

- Slightly larger API surface than the ideal minimal runner.

### 3) Formatting/lint hook failures

Symptom:

- Geppetto commit failed due to `gofmt` issue in newly added file.

Fix:

- Ran `gofmt -w` and re-committed.

Notes:

- Pinocchio’s pre-commit runs `npm install/build` for web-chat frontend and reported `npm audit` vulnerabilities. These were not addressed as part of this ticket (they are unrelated to inference unification).

### 4) Dependency churn risk

Symptom:

- Go tooling sometimes rewrote `go.mod/go.sum` unintentionally during earlier phases.

Mitigation:

- Avoided committing unrelated dependency churn; reverted when it happened.

### 5) go-go-mento incompatibility

Symptom:

- Adding `go-go-mento/go` to the workspace caused a large set of unrelated compile errors (older API alignment).

Decision:

- Explicitly deferred; do not “drive-by fix” unrelated repo compatibility under this ticket.

## Compromises and “we knowingly left this imperfect”

### 1) `RunInferenceStarted` exists

This is a pragmatic concession to Bubble Tea and “mark running before goroutine” patterns. The “pure interface” would ideally stay smaller.

Suggested eventual improvement:

- Make the runner API explicitly model:
  - `Start(ctx) (RunHandle, error)` and `RunHandle.Wait()`

But we did not implement this because it would be a larger refactor and would risk scope creep.

### 2) Duplicate “ParsedLayers EngineBuilder” implementations exist

We introduced:

- `geppetto/cmd/examples/internal/examplebuilder` (example-only)
- `pinocchio/pkg/inference/enginebuilder` (pinocchio-only)

This duplication is acceptable short-term, but it should eventually converge into:

- a shared helper in geppetto (if generally useful), or
- a pinocchio EngineBuilder that also handles profiles/overrides (superseding the parsed-layers one).

### 3) Seed construction uses local “cloneTurn” helpers

Both pinocchio TUI and webchat now clone turns locally.

This is safe but a bit repetitive. It may be worth adding:

- `turns.CloneTurn(t *Turn) *Turn` (deep-enough for Blocks/Data/Metadata)

…but we did not do this to keep the migration scoped and avoid touching shared turn primitives without strong need.

## Testing performed (and what “real world” tests remain)

### Automated

- `geppetto`: `go test ./... -count=1`
- `pinocchio`: `go test ./... -count=1`
- Both repos passed their pre-commit hooks (with the note that pinocchio’s hook also runs a frontend build and prints npm audit warnings).

### Recommended “real world” validations (manual)

1) Pinocchio webchat multi-turn with OpenAI Responses thinking models:

```bash
go run ./cmd/web-chat --log-level DEBUG
```

In the browser webchat:

- Send a first message (expect streaming)
- Send a second message (ensure no “reasoning item must be followed” 400s)

2) Pinocchio TUI multi-turn:

```bash
go run ./cmd/pinocchio code professional "hello" --ai-engine gpt-5-mini --chat --ai-api-type openai-responses
```

- Send multiple messages; verify generating state completes each time.

3) Agent example:

```bash
go run ./cmd/agents/simple-chat-agent
```

Verify:

- Tool call events are visible
- Tool results appear and the loop continues
- Interrupt/Kill cancels cleanly

## What needs cleanup and review (actionable checklist)

### High priority review

- **Cancel ownership rules**: verify there is no path where `StartRun` is set but `FinishRun` is skipped (panic/early returns).
  - Review `core.Session.RunInferenceStarted` + pinocchio webchat goroutine defers.
- **Event sink duplication**: standardize per app whether sinks are attached via engine config or session context.
  - Pinocchio webchat currently attaches sinks via Session.
  - Pinocchio TUI engines historically used `engine.WithSink`; confirm we are not double-attaching.

### Medium priority cleanup

- Consider introducing a shared `turns.Clone` helper to remove duplicated `cloneTurn` functions.
- Consider consolidating ParsedLayers EngineBuilder into a shared helper if it’s broadly used.
- Decide whether `InferenceState.HasCancel()` is the right API, or whether cancel should always be set by Session (and UIs shouldn’t set it directly).

### Ticket tasks still open (MO-004)

As of this postmortem:

- Add a reference persister implementation(s) (no-op + filesystem persister for debugging).
- Add targeted tests for `Session.RunInference` in both single-pass and tool-loop modes.
- Document migration notes / breaking changes in the design docs.
- Implement moments migration when ready (analysis exists).

## Migration notes for moments (preview)

See analysis:

- `analysis/01-moments-webchat-router-migration-to-geppetto-inferencestate-session.md`

Key warning:

- moments’ “step mode” and tool-loop orchestration is more complex than pinocchio’s; the migration should start by swapping `Conversation` state ownership to `InferenceState` and only then decide whether to reuse `core.Session` tool-loop or keep a custom loop initially.

## Appendix: Reference diagrams

### Sequence diagram: pinocchio webchat `/chat` after migration

```text
HTTP handler
  |
  | conv := getOrCreateConv()
  | conv.Inf.StartRun()
  |
  +--> goroutine:
        runCtx := context.WithCancel(baseCtx)
        conv.Inf.SetCancel(cancel)
        seed := clone(conv.Inf.Turn) + append(user prompt)
        sess := core.Session{State: conv.Inf, Registry: tools, ...}
        sess.RunInferenceStarted(runCtx, seed)
        defer FinishRun()
```

### Event flow (goal state)

```text
toolhelpers.RunToolCallingLoop
  publishes -> events.PublishEventToContext
provider engine
  publishes -> events.PublishEventToContext
engine config sinks
  are attached into ctx at start of RunInference
so both paths converge into the same sink list
```
