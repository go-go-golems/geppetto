---
Title: 'Postmortem: Session refactor (MO-007)'
Ticket: MO-007-SESSION-REFACTOR
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
    - Path: ../../../../../../../bobatea/pkg/chat/model.go
      Note: WithAutoStartBackend / StartBackendMsg mismatch
    - Path: ../../../../../../../pinocchio/pkg/cmds/cmd.go
      Note: Pinocchio disables autoStartBackend; uses submit path
    - Path: cmd/llm-runner/main.go
      Note: CLI that runs fixtures and records artifacts
    - Path: pkg/inference/engine/factory/factory.go
      Note: Provider engine selection after removing engine options
    - Path: pkg/inference/fixtures/fixtures.go
      Note: Inference fixtures used by llm-runner
    - Path: pkg/inference/session/session.go
      Note: Session lifecycle invariants and StartInference implementation
    - Path: pkg/inference/session/tool_loop_builder.go
      Note: Standard builder that wires middleware
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-21T15:09:35.701453027-05:00
WhatFor: ""
WhenToUse: ""
---


# Postmortem: Session refactor (MO-007)

## Goal

Replace the legacy inference lifecycle stack (`InferenceState`, `core.Session`, engine-config sinks via `engine.WithSink`) with a single, consistent model:

- `session.Session` owns multi-turn history (a chat “conversation”)
- `Session.StartInference(ctx)` starts one inference asynchronously and returns an `ExecutionHandle`
- `InferenceRunner.RunInference(ctx, seed)` is the single blocking provider/loop entrypoint
- event sinks are **context-only** (`events.WithEventSinks`), never “configured into an engine”

The downstream intent is that **TUI and webchat** become thin UIs that submit prompts and render events, while inference orchestration is centralized and shared.

This document is a “fleshed out diary”: it restates the work in chronological implementation steps, but focuses on engineering decisions, pitfalls, and what to review.

## Outcome summary (what shipped)

### New core

- `geppetto/pkg/inference/session`:
  - `Session` + `ExecutionHandle` (cancel/wait, single active inference invariant)
  - `EngineBuilder` / `InferenceRunner` interfaces
  - `ToolLoopEngineBuilder` standard runner:
    - wraps provider `engine.Engine` with `middleware.Middleware`
    - attaches sinks via `events.WithEventSinks(ctx, ...)`
    - attaches snapshot hook via `toolhelpers.WithTurnSnapshotHook(ctx, ...)`
    - runs `toolhelpers.RunToolCallingLoop(...)` when configured

### Migrations

- geppetto examples migrated to use `session.Session` / `ToolLoopEngineBuilder`.
- pinocchio TUI backend migrated to session semantics (no more `InferenceState`).
- pinocchio webchat migrated to session semantics (no `StartRun/FinishRun`).
- pinocchio agent “tool-loop backend” migrated to session semantics.

### Sink cleanup (the important plumbing simplification)

- Deleted `engine.WithSink` and the entire “engine options/config sinks” mechanism.
- Updated provider constructors (OpenAI Chat, OpenAI Responses, Claude, Gemini) to rely on context sinks only.
- Updated engine factory API to no longer accept options.

### Real-world validation

Verified with:

- `go test ./... -count=1` in both `geppetto/` and `pinocchio/`
- real OpenAI calls via geppetto examples (single-pass, streaming, tool loop, Responses thinking)
- pinocchio Bubble Tea chat in tmux, multi-turn, with OpenAI Responses `gpt-5-mini` (thinking events delivered and UI completes correctly)

## Chronological implementation steps (what changed, why, and how)

### Step 1: Introduce `session.Session` + `ExecutionHandle` + `ToolLoopEngineBuilder` (geppetto)

Primary goal: put a small, testable lifecycle “kernel” in one package.

Key design choices:

- **One active inference at a time**: avoids subtle UI races and simplifies cancellation semantics.
- **Session owns turns**: `Session.Append(seed)` happens before starting inference. The builder/runner uses the latest turn as input.
- **Async start, blocking wait**: UIs need “start now, render events while it runs, then apply final result”.

Added unit tests to lock down invariants:

- cancel triggers `context.Canceled`
- output turn is appended to session history only on success
- builder/runner uses snapshot hook and persister in the tool-loop path

### Step 2: Migrate geppetto examples to the session model

Goal: prove the new lifecycle is viable in real call paths and clarify the intended “shape” for other apps.

The main migration pattern:

1) Build seed turn (system + user prompt)
2) `sess.Append(seed)`
3) `handle, _ := sess.StartInference(ctx)`
4) `out, err := handle.Wait()` (or stream via sinks)

### Step 3: Migrate pinocchio TUI (Bubble Tea) to session semantics

Goal: make the TUI a pure downstream consumer:

- submit prompt
- start inference once
- render event stream
- finalize on completion

Important subtlety:

Bubble Tea UIs often start work inside a returned `tea.Cmd`.
However, to keep the “already running” check correct (and to avoid needing “RunInferenceStarted”), pinocchio now starts inference synchronously in `Backend.Start(...)` and returns a `tea.Cmd` that only blocks on `ExecutionHandle.Wait()`.

### Step 4: Migrate pinocchio webchat to session semantics

Goal: delete the “pre-start lifecycle split” and use a single `IsRunning()` / `StartInference()`.

Important subtlety:

The old webchat had a “wait until router is running before starting” intent. The new code uses:

```text
wait router.Running() (best-effort, bounded) → StartInference
```

This preserves the intent without recreating “StartRun/FinishRun”.

### Step 5: Delete `engine.WithSink` / engine option plumbing (geppetto + pinocchio)

Goal: eliminate the confusing duality of:

- “engine-config sinks” (options passed at engine construction time), and
- “context sinks” (events attached to ctx)

Before this change, provider engines sometimes had code like:

```text
if engine.Config.EventSinks non-empty:
    ctx = events.WithEventSinks(ctx, those sinks...)
```

That bridge had two failure modes:

- duplicates (if sinks are also placed on ctx by caller)
- confusion (developers don’t know which is authoritative)

After this change:

- provider engines publish only via `events.PublishEventToContext(ctx, ...)`
- callers attach sinks only via `events.WithEventSinks(...)` (usually by configuring `ToolLoopEngineBuilder.EventSinks`)
- the engine factory no longer accepts options at all

This step was mechanical but cross-cutting (provider constructors, engine factory interface, helpers, example builders, pinocchio builder).

### Step 6: Real-world tests and the pinocchio chat “auto-start hang”

We ran live inference tests and discovered a UI lifecycle mismatch around `WithAutoStartBackend`.

See the dedicated section below; the short version:

- `WithAutoStartBackend(true)` triggers `StartBackendMsg` at model init time.
- `bobatea/pkg/chat/model.go:startBackend()` currently transitions the UI into “streaming state” but does not start the backend inference (it is a no-op in the new prompt flow).
- That leaves the UI stuck in “already streaming” forever.

Pinocchio now always uses `WithAutoStartBackend(false)` and relies on normal submit flow to start inference.

## What “inference fixtures” are (and what they’re for)

Inference fixtures live in `geppetto/pkg/inference/fixtures` and are used by `geppetto/cmd/llm-runner`.

They are not unit test fixtures; they are an “E2E harness” for running a recorded turn (and optional follow-ups) and persisting artifacts to disk for analysis, debugging, and reproducibility.

Key components:

- `fixtures.LoadFixtureOrTurn(path)`:
  - reads YAML that can be either:
    - a full `turns.Turn` document, or
    - a wrapper document: `{ turn: ..., followups: [...] }`
- `fixtures.ExecuteFixture(ctx, turn, followups, settings, opts)`:
  - executes:
    1) Run inference on the initial turn
    2) Append follow-up blocks one-by-one and re-run inference each time
  - writes artifacts:
    - `input_turn.yaml`, `final_turn.yaml`, `events.ndjson`, plus per-followup `events-N.ndjson` and `final_turn_N.yaml`
  - optionally captures:
    - raw provider data via `engine.DebugTap` (see `fixtures/rawtap.go`)
    - HTTP via VCR cassettes (`github.com/dnaeon/go-vcr/recorder`)
    - logs to `logs.jsonl`
- `fixtures.BuildReport(outDir)`:
  - creates a simple Markdown summary report from the recorded artifacts

Fixtures are primarily for:

- debugging provider protocol issues (Responses validation, tool ordering, thinking)
- capturing minimal repros that can be shipped as YAML + cassette
- inspecting event ordering when a UI “hangs”

## The `WithAutoStartBackend` issue (what happened, why, and whether it’s “fixed”)

### What `WithAutoStartBackend(true)` does

In bobatea chat:

- `WithAutoStartBackend(true)` causes `Init()` to enqueue a `StartBackendMsg`.
- The model handles `StartBackendMsg` by calling `startBackend()`.

### Why it caused a hang

`startBackend()` currently:

- sets `StateStreamCompletion`
- blurs input / updates keybindings
- returns a “backend command” which is explicitly a no-op in the “new prompt flow”

So the UI ends up in “streaming state” while no inference has actually been started. Then:

- pressing tab triggers `submit()`
- `submit()` checks `backend.IsFinished()`
- because the model thinks it is streaming, submit is ignored/blocked

### Is the behavior “the same now”?

- In **bobatea** itself: yes. `WithAutoStartBackend(true)` still triggers `StartBackendMsg`, and `startBackend()` still does not start inference.
- In **pinocchio**: we changed the wiring so pinocchio no longer uses `WithAutoStartBackend(true)`.
  - Pinocchio now relies on `ReplaceInputTextMsg` + `SubmitMessageMsg` (submit path), which *does* call `backend.Start(...)`.

### Recommended follow-up

Either:

1) remove `StartBackendMsg`/`startBackend()` entirely from bobatea, or
2) re-implement `startBackend()` so it starts inference by calling backend.Start with the current input value (but that implies a defined “where does prompt text live” contract).

Right now it’s a footgun: it changes state but doesn’t do the work.

**Status (as implemented):** bobatea commit `c2a08dc` removed `StartBackendMsg` / `startBackend()` / `WithAutoStartBackend`, and pinocchio commit `930b461` removed the last call site.

## What needs careful review (and what I was unhappy with / what was tricky)

### 1) Concurrency / invariants in `session.Session`

Review:

- `Session.StartInference` locking and the “one active inference” invariant.
- ensuring `Append(...)` semantics are correct for all callers (TUI is serialized; webchat is concurrent).

The tradeoff:

- We chose simplicity: one active inference per session.
- This needs to be enforced consistently and tested under webchat concurrency.

### 2) EngineBuilder boundaries and “who owns what”

Tricky bit:

- Tool loop needs multiple inputs: base engine, middleware chain, tool registry, tool config, snapshot hook, persister, sinks.
- We intentionally push sinks/snapshots into context at the runner level.

What to review:

- `ToolLoopEngineBuilder` wiring is the single new “center of gravity”. It should remain small and deterministic.

### 3) Removal of engine options (potential external breakage)

We removed `engine.Option`/`engine.WithSink` entirely. This is good internally, but it can break external downstream repos or consumers who used the old engine factory signature.

What to review:

- any other modules outside this workspace that import geppetto and call `factory.CreateEngine(..., engine.WithSink(...))`

### 4) OpenAI Responses include/encrypted behavior (model-specific gotcha)

In real runs, using OpenAI Responses with models that do not support `include=reasoning.encrypted_content` can produce:

> `Encrypted content is not supported with this model.`

This is not caused by MO-007 directly, but the playbook run surfaced it.
We used `gpt-5-mini` successfully.

### 5) The bobatea “auto-start” semantics mismatch

This one is “I’m unhappy with it”:

- `WithAutoStartBackend` looks like “start inference on init”
- but it currently only flips UI state and does not start the backend

Pinocchio works because we stopped using it, but the underlying mismatch still exists and could bite other callers.

## How to validate (commands)

### Unit tests

```bash
cd geppetto && go test ./... -count=1
cd pinocchio && go test ./... -count=1
```

### Real-world smoke (OpenAI key required)

```bash
cd geppetto
go run ./cmd/examples/simple-inference simple-inference "Say hello." --pinocchio-profile 4o-mini --ai-engine gpt-4o-mini --ai-api-type openai
go run ./cmd/examples/simple-streaming-inference simple-streaming-inference "Write one sentence about penguins." --pinocchio-profile 4o-mini --ai-engine gpt-4o-mini --ai-api-type openai --verbose
go run ./cmd/examples/generic-tool-calling generic-tool-calling "What's the weather in Paris and what is 2+2?" --pinocchio-profile 4o-mini --ai-engine gpt-4o-mini --ai-api-type openai --tools-enabled --max-iterations 3
go run ./cmd/examples/openai-tools test-openai-tools --ai-api-type openai-responses --ai-engine gpt-5-mini --mode thinking --prompt "What is 23*17 + 55?"
```

Pinocchio TUI multi-turn (tmux):

```bash
cd pinocchio
rm -f /tmp/pinocchio-tui.log
tmux kill-session -t pinocchio-tui-smoke 2>/dev/null || true
tmux new-session -d -s pinocchio-tui-smoke \
  "go run ./cmd/pinocchio code professional 'hello' --ai-api-type openai-responses --ai-engine gpt-5-mini --chat --log-level debug --log-file /tmp/pinocchio-tui.log --with-caller"
sleep 10
tmux send-keys -t pinocchio-tui-smoke 'What is 2+2?' Tab
sleep 12
tmux send-keys -t pinocchio-tui-smoke 'And what is 3+3?' Tab
sleep 14
tmux send-keys -t pinocchio-tui-smoke M-q
sleep 1
tmux kill-session -t pinocchio-tui-smoke 2>/dev/null || true
tail -n 120 /tmp/pinocchio-tui.log
```

## Follow-ups / cleanup suggestions (not done here)

- Decide what to do with bobatea `WithAutoStartBackend`:
  - remove it, or make it actually start inference.
- Update documentation that references `engine.WithSink` to instead recommend context sinks or Session builder sinks.
- Consider adding a small integration test for webchat concurrency (`Session.IsRunning()` gating), if we want hard guarantees.
