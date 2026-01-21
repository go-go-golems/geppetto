---
Title: Diary
Ticket: MO-007-SESSION-REFACTOR
Status: active
Topics:
    - inference
    - architecture
    - events
    - webchat
    - tui
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../bobatea/pkg/chat/model.go
      Note: Removed WithAutoStartBackend/StartBackendMsg/startBackend; inference only starts via submit()
    - Path: ../../../../../../../bobatea/pkg/chat/user_messages.go
      Note: Removed StartBackendMsg
    - Path: ../../../../../../../bobatea/pkg/eventbus/eventbus.go
      Note: Switch to AddConsumerHandler to satisfy Watermill deprecation lint
    - Path: ../../../../../../../pinocchio/pkg/cmds/cmd.go
      Note: |-
        Fix chat start: stop using bobatea autoStartBackend; rely on submit() to start inference (commit da5f276)
        Disable bobatea autoStartBackend; submit() starts inference (commit da5f276)
        Removed last WithAutoStartBackend call site
    - Path: ../../../../../../../pinocchio/pkg/inference/enginebuilder/parsed_layers.go
      Note: Updated ParsedLayersEngineBuilder to match engine factory signature change (commit 6ce03ff)
    - Path: cmd/examples/generic-tool-calling/main.go
      Note: Streaming + tool loop example migrated (commit 5cd95af)
    - Path: cmd/examples/simple-inference/main.go
      Note: Example migrated to session.Session + ToolLoopEngineBuilder (commit 5cd95af)
    - Path: pkg/inference/engine/factory/factory.go
      Note: Removed engine option plumbing; provider constructors no longer accept sinks/options (commit d6a0f54)
    - Path: pkg/inference/session/builder.go
      Note: EngineBuilder/InferenceRunner interfaces (commit 158e4be)
    - Path: pkg/inference/session/execution.go
      Note: ExecutionHandle cancel/wait contract (commit 158e4be)
    - Path: pkg/inference/session/session.go
      Note: Async Session lifecycle + StartInference invariants (commit 158e4be)
    - Path: pkg/inference/session/session_test.go
      Note: Unit tests for session lifecycle + ToolLoopEngineBuilder (commit 158e4be)
    - Path: pkg/inference/session/tool_loop_builder.go
      Note: Standard ToolLoopEngineBuilder wiring (middleware+sinks+snapshots+tool loop) (commit 158e4be)
    - Path: pkg/steps/ai/gemini/engine_gemini.go
      Note: Removed engine-config sink bridge; Gemini uses context sinks only (commit d6a0f54)
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-21T13:51:22.625347026-05:00
WhatFor: ""
WhenToUse: ""
---







# Diary

## Goal

Track the implementation of `MO-007-SESSION-REFACTOR`: introducing a new `Session` + `ExecutionHandle` lifecycle and a standard `ToolLoopEngineBuilder` that composes `engine.Engine` + `middleware.Middleware` and runs the canonical tool-calling loop with sinks/snapshots wired via context.

## Context

MO-007 is intended to supersede prior “cleanup sinks” and “cancellation lifecycle” tickets by standardizing:

- **Session** = long-lived multi-turn interaction (`SessionID`), owns turn history.
- **Inference** = one blocking step, cancelable via context, executed asynchronously by `Session.StartInference`.
- **EngineBuilder/Runner** = stable composition point for base provider engine + middleware + tool loop + hooks.

## Quick Reference

Key packages (new):

- `geppetto/pkg/inference/session`:
  - `Session.StartInference(ctx) (*ExecutionHandle, error)` (async)
  - `ExecutionHandle.Cancel()`, `ExecutionHandle.Wait() (*turns.Turn, error)`
  - `ToolLoopEngineBuilder` (standard builder for chat-style apps)

## Usage Examples

N/A (implementation in progress)

## Step 1: Add session package + ToolLoopEngineBuilder and tests

This step introduces `geppetto/pkg/inference/session` as the next home for MO-007’s lifecycle primitives. The focus was to get a minimal, testable implementation in place: a `Session` that owns turn history and can start an async inference, an `ExecutionHandle` for cancel/wait, and a standard `ToolLoopEngineBuilder` that composes a base `engine.Engine` with `middleware.Middleware` and runs either a single inference or the tool loop with snapshot/persistence wiring.

I also added a small unit test suite to lock down the new semantics (append-on-success, cancel behavior, single active inference) and to validate the builder’s integration with the tool loop + snapshot hook + persister.

**Commit (code):** 158e4be — "Session: add ToolLoopEngineBuilder and lifecycle tests"

### What I did
- Added `geppetto/pkg/inference/session/builder.go` with `EngineBuilder` + `InferenceRunner` interfaces.
- Added `geppetto/pkg/inference/session/execution.go` implementing `ExecutionHandle` (cancel + wait).
- Added `geppetto/pkg/inference/session/session.go` implementing `Session` with async `StartInference`.
- Added `geppetto/pkg/inference/session/tool_loop_builder.go` implementing `ToolLoopEngineBuilder`:
  - wraps `engine.Engine` with `middleware.NewEngineWithMiddleware(...)`
  - injects `events.EventSink` via `events.WithEventSinks(ctx, ...)`
  - injects snapshot hook via `toolhelpers.WithTurnSnapshotHook(ctx, ...)`
  - runs either `eng.RunInference(...)` or `toolhelpers.RunToolCallingLoop(...)`
  - best-effort persists final turn
- Added unit tests in `geppetto/pkg/inference/session/session_test.go`.
- Ran `go test ./geppetto/... -count=1`.

### Why
- MO-007 needs a single, shared lifecycle surface that both TUI and webchat can call into.
- The “tool loop + middleware + sinks + snapshots” wiring should live at the builder/runner layer, not in provider-engine config knobs (`engine.WithSink`), so providers can stay strict and UIs can stay downstream.

### What worked
- `go test ./geppetto/... -count=1` passes.
- New tests cover:
  - append output turn on success
  - cancellation propagates `context.Canceled`
  - only one active inference at a time
  - tool loop path calls snapshot hook and invokes persister

### What didn't work
- Initial `git commit` failed due to formatting enforced by pre-commit hooks:
  - Command: `git -C geppetto commit -m "Session: add ToolLoopEngineBuilder and lifecycle tests"`
  - Error: `pkg/inference/session/session_test.go:169:1: File is not properly formatted (gofmt)`
  - Error: `pkg/inference/session/tool_loop_builder.go:145:1: File is not properly formatted (gofmt)`
  - Fix: `gofmt -w geppetto/pkg/inference/session/session_test.go geppetto/pkg/inference/session/tool_loop_builder.go`

### What I learned
- The new `Session.StartInference` needs to be careful about lock usage; calling a lock-taking helper (`Latest()`) while holding `s.mu` is a deadlock footgun. The tests help keep this honest.

### What was tricky to build
- Getting the lifecycle split right:
  - `Session.StartInference` must be async and return immediately.
  - `InferenceRunner.RunInference` must be blocking and rely on context for cancellation.
- Testing “tool loop + snapshot hook” without requiring real tool calls: using a non-nil registry and a base engine that emits no tool calls still exercises `pre_inference`/`post_inference` hook phases.

### What warrants a second pair of eyes
- Whether the “append output turn only on success” policy is correct for all frontends (webchat might want to persist partial/error turns for UX/debugging).
- Whether persister failures should remain best-effort (ignored) or should be surfaced to the caller in some contexts.

### What should be done in the future
- Migrate callers off `geppetto/pkg/inference/core.Session` and `geppetto/pkg/inference/state.InferenceState` to the new `session.Session` + `ToolLoopEngineBuilder`.
- Remove `engine.WithSink` callsites and engine-config sink wiring once all callers use context sinks.

### Code review instructions
- Start at `geppetto/pkg/inference/session/tool_loop_builder.go` (runner wiring: middleware + sinks + snapshots + tool loop).
- Then review `geppetto/pkg/inference/session/session.go` (async lifecycle + concurrency invariants).
- Validate with `go test ./geppetto/... -count=1`.

## Step 2: Migrate geppetto examples off core.Session/InferenceState

This step moves the public “small examples” in `geppetto/cmd/examples/*` to the new MO-007 primitives (`session.Session` + `session.ToolLoopEngineBuilder`). The goal is to keep the examples as the first always-green consumer surface that validates the new lifecycle, before tackling pinocchio’s TUI/webchat and finally deleting the legacy packages.

The examples now create a `runID`, seed a `turns.Turn`, instantiate a `session.Session` with a `ToolLoopEngineBuilder` (including event sinks for streaming examples), and then execute inference via `StartInference(...).Wait()`.

**Commit (code):** 5cd95af — "Examples: switch to session.Session and ToolLoopEngineBuilder"

### What I did
- Updated these programs to use `geppetto/pkg/inference/session`:
  - `geppetto/cmd/examples/simple-inference/main.go`
  - `geppetto/cmd/examples/simple-streaming-inference/main.go`
  - `geppetto/cmd/examples/middleware-inference/main.go`
  - `geppetto/cmd/examples/openai-tools/main.go`
  - `geppetto/cmd/examples/claude-tools/main.go`
  - `geppetto/cmd/examples/generic-tool-calling/main.go`
- Removed usage of:
  - `geppetto/pkg/inference/state.NewInferenceState`
  - `geppetto/pkg/inference/core.Session`
- Ran `go test ./... -count=1` (within `geppetto/`) via pre-commit.

### Why
- This keeps a clear “known good” reference for how to wire a chat-style run using MO-007 primitives.
- It reduces the blast radius of later deletions: once examples are migrated, we can remove legacy packages more confidently.

### What worked
- All examples compile; pre-commit `test` and `lint` pass on commit.

### What didn't work
- N/A

### What I learned
- The examples previously already relied on context sinks (not engine-config sinks), so the migration to `ToolLoopEngineBuilder` is mostly mechanical.

### What was tricky to build
- Avoiding accidental “double orchestration” in examples that already use middleware-managed tools (keep `ToolLoopEngineBuilder.Registry == nil` there).

### What warrants a second pair of eyes
- Whether examples should standardize on `builder.Build(...).RunInference(...)` instead of `Session.StartInference(...).Wait()` to avoid nested goroutines in apps that already use an errgroup (not a correctness issue, but worth deciding for consistency).

### What should be done in the future
- Update pinocchio examples and pinocchio UI (TUI/webchat) to stop using `core.Session`/`InferenceState`.

### Code review instructions
- Review one non-streaming and one streaming example for the new wiring:
  - `geppetto/cmd/examples/simple-inference/main.go`
  - `geppetto/cmd/examples/simple-streaming-inference/main.go`
- Validate with `go test ./... -count=1` in `geppetto/`.

## Step 3: Migrate pinocchio TUI backend to Session/ExecutionHandle (and drop engine.WithSink there)

This step migrates the pinocchio Bubble Tea chat backend off `InferenceState` and `core.Session.RunInferenceStarted(...)`. The new behavior starts inference immediately (so “already running” checks happen synchronously) and returns a Bubble Tea `Cmd` that simply blocks on the `ExecutionHandle.Wait()` result.

As part of this, the TUI runtime builder stops constructing the provider engine with `engine.WithSink(uiSink)`. Instead, it passes `uiSink` into `ui.NewEngineBackend(...)`, which wires it into `session.ToolLoopEngineBuilder.EventSinks`, so provider engines publish streaming events via context sinks only.

**Commit (code):**
- geppetto: 388e976 — "Session: add IsRunning helper"
- pinocchio: 0c6041a — "TUI: use geppetto session.Session and context sinks"

### What I did
- Added `Session.IsRunning()` to `geppetto/pkg/inference/session/session.go` so UIs can check running state without reaching into internals.
- Updated `pinocchio/pkg/ui/backend.go`:
  - replaced `*state.InferenceState` + `core.Session` with `*session.Session`
  - rewired `Start()` to:
    1) build a seed turn (clone latest + append user prompt),
    2) append it to the session,
    3) call `Session.StartInference(ctx)` immediately,
    4) return a `tea.Cmd` that waits on `ExecutionHandle.Wait()`
  - rewired `Interrupt()/Kill()/IsFinished()` to use `Session.CancelActive()` / `Session.IsRunning()`
- Updated `pinocchio/pkg/ui/runtime/builder.go`:
  - removed `engine.WithSink(uiSink)` when constructing the provider engine
  - passed `uiSink` into `ui.NewEngineBackend(eng, uiSink)` so events flow via context sinks.

### Why
- The “pre-start lifecycle split” (`StartRun` + `RunInferenceStarted`) is exactly the complexity MO-007 is trying to eliminate.
- TUI should become a downstream consumer that depends only on Session semantics and context sinks.

### What worked
- `go test ./... -count=1` passes in `pinocchio/` after migration.

### What didn't work
- N/A (note: pinocchio’s pre-commit hook runs `npm audit` during lint and reports vulnerabilities, but this step did not address them).

### What I learned
- Starting inference immediately in `Backend.Start()` (instead of inside the returned `tea.Cmd`) is the simplest way to keep “already running” checks correct without needing a secondary `RunInferenceStarted` API.

### What was tricky to build
- Preserving turn continuity: we must append the “prompt turn” to the session before calling `StartInference`, since the session runner uses the latest turn as input.

### What warrants a second pair of eyes
- Whether appending the “prompt turn” before `StartInference` is acceptable in the presence of races (it is safe in the TUI because Start is serialized by Bubble Tea, but webchat may need stronger guarantees).

### What should be done in the future
- Apply the same pattern to pinocchio webchat router and any remaining tool-loop backends.

### Code review instructions
- Start at:
  - `pinocchio/pkg/ui/backend.go`
  - `pinocchio/pkg/ui/runtime/builder.go`
- Validate with `go test ./... -count=1` in `pinocchio/`.

## Step 4: Migrate pinocchio webchat router off InferenceState/core.Session

This step migrates pinocchio’s webchat conversation state and run loop wiring to the new session model. It removes `InferenceState.StartRun/FinishRun/SetCancel` usage and replaces the ad-hoc “wait for router running then RunInferenceStarted” goroutine with an explicit `session.Session` per conversation and a standard `ToolLoopEngineBuilder` per inference.

The key behavioral change is that the HTTP handler now:
1) checks `Session.IsRunning()` for conflict,
2) appends the “prompt turn” into the session history,
3) calls `Session.StartInference(...)` (async),
4) spawns a goroutine to `Wait()` only for logging purposes.

**Commit (code):** pinocchio d3c0684 — "Webchat: replace InferenceState/core.Session with session.Session"

### What I did
- Updated `pinocchio/pkg/webchat/conversation.go`:
  - replaced `Inf *state.InferenceState` with `Sess *session.Session`
  - initialize session with a base `ToolLoopEngineBuilder` that sets `Base` and `EventSinks`
- Updated `pinocchio/pkg/webchat/router.go`:
  - replaced `core.Session` usage with `session.ToolLoopEngineBuilder` + `Session.StartInference`
  - replaced “run in progress” gating from `StartRun()` to `Sess.IsRunning()`
  - updated `seedForPrompt` to clone from `Sess.Latest()` instead of `Inf.Turn`

### Why
- Webchat was one of the main reasons `RunInferenceStarted` existed; MO-007’s goal is to delete that complexity entirely.
- The tool loop orchestration belongs in a standard builder/runner, not scattered across routers.

### What worked
- `go test ./... -count=1` passes in `pinocchio/` after migration.

### What didn't work
- N/A

### What I learned
- Even for HTTP-triggered inference, starting the async inference immediately and returning a `run_id` is simpler than managing a separate “run started” flag.

### What was tricky to build
- Preserving the “don’t start before router handlers are running” intent without reintroducing a separate `StartRun` API: this step uses a small best-effort wait on `r.router.Running()` (2s) before starting inference.

### What warrants a second pair of eyes
- Whether `conv.Sess.Builder = ...` should be set under a conversation/session lock to avoid any possible concurrent reads (in practice, there is only one active inference per session).

### What should be done in the future
- Remove remaining `engine.WithSink` call sites outside the TUI/webchat runtime (e.g., CLI paths and redis examples) once the new session plumbing is used everywhere.

### Code review instructions
- Start at:
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/conversation.go`
- Validate with `go test ./... -count=1` in `pinocchio/`.

## Step 5: Delete engine option/config sink plumbing (engine.WithSink) across geppetto/pinocchio

This step removes the last remnants of “engine-configured sinks” from the provider engine stack. Previously, some provider constructors accepted `options ...engine.Option` and would “bridge” any configured sinks into the context at the start of `RunInference` so that downstream publishers (middleware, tool loops) could see them. That bridge became both redundant (we standardized on context sinks) and a frequent source of confusion about which path is authoritative.

The change makes the model explicit and strict: provider engines publish via context (`events.PublishEventToContext`), and the app layer is responsible for attaching sinks to `ctx` (typically via `session.ToolLoopEngineBuilder.EventSinks` or directly via `events.WithEventSinks`). This eliminates the risk of duplicate delivery and removes an entire class of “why did my sink not receive events?” plumbing issues.

**Commit (code):**
- geppetto d6a0f54 — "Remove engine option/config sink plumbing"
- pinocchio 6ce03ff — "Update ParsedLayersEngineBuilder for new engine factory API"

### What I did
- Updated provider constructors to stop accepting `options ...engine.Option`:
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `geppetto/pkg/steps/ai/openai_responses/engine.go`
  - `geppetto/pkg/steps/ai/claude/engine_claude.go`
  - `geppetto/pkg/steps/ai/gemini/engine_gemini.go`
- Deleted `geppetto/pkg/inference/engine/options.go` and removed the `engine.Option`/`engine.Config` types.
- Updated `geppetto/pkg/inference/engine/factory` and helpers to stop threading options:
  - `CreateEngine(settings)` now has no variadic options
  - `factory.NewEngineFromParsedLayers(parsedLayers)` no longer takes options
- Updated `geppetto/cmd/examples/internal/examplebuilder/builder.go` and `pinocchio/pkg/inference/enginebuilder/parsed_layers.go` to match the new factory API.
- Ran:
  - `go test ./... -count=1` in `geppetto/`
  - `go test ./... -count=1` in `pinocchio/`

### Why
- The new MO-007 world standardizes on context sinks only (`events.WithEventSinks`), with sink selection controlled by the app/session layer.
- `engine.WithSink` was a legacy convenience that required subtle bridging behavior inside provider engines and complicated reasoning about “who publishes where”.

### What worked
- `go test ./... -count=1` passes in both repos after removing options/sinks.
- Pinocchio still receives streaming events correctly (via `session.ToolLoopEngineBuilder.EventSinks` and context sinks).

### What didn't work
- N/A (pinocchio pre-commit lint still reports `npm audit` vulnerabilities, but that’s unrelated to this refactor and was not addressed).

### What I learned
- Keeping provider engines “pure” (no config sink fields) reduces coupling: the provider engines can focus solely on request/streaming/protocol correctness.

### What was tricky to build
- Finding and updating the last compiled call sites that threaded `engine.Option` through builders (`examplebuilder` and pinocchio’s `enginebuilder`).

### What warrants a second pair of eyes
- Review the provider constructors’ new signatures and ensure no external module relies on `engine.Option` (we fixed all in-workspace call sites, but downstream consumers outside the workspace may need a coordinated update).

### What should be done in the future
- Update `.md` docs that still mention `engine.WithSink` to instead recommend `Session.EventSinks` or explicit `events.WithEventSinks` usage.

### Code review instructions
- Start at:
  - `geppetto/pkg/inference/engine/factory/factory.go`
  - `geppetto/pkg/steps/ai/openai_responses/engine.go`
  - `geppetto/pkg/steps/ai/gemini/engine_gemini.go`
- Validate with:
  - `go test ./... -count=1` in `geppetto/` and `pinocchio/`

## Step 6: Real-world smoke tests (geppetto examples + pinocchio TUI) and fix chat auto-start hang

This step closes the loop on MO-007 by exercising real provider calls (OpenAI Chat Completions + OpenAI Responses) and the multi-turn pinocchio TUI. The core goal was to validate that the Session/ExecutionHandle changes didn’t just compile, but actually drive streaming events through the router and render correctly in Bubble Tea.

During the first tmux run, pinocchio chat appeared to “freeze” immediately without ever starting inference. Debug logs showed that the chat model was dispatching `StartBackendMsg` at init (via `WithAutoStartBackend(true)`), but `startBackend()` is now explicitly a no-op in the new prompt flow. That meant the UI transitioned into the streaming state without ever calling `backend.Start(...)`, so subsequent submits were blocked. The fix was to stop using `WithAutoStartBackend` from pinocchio and rely exclusively on the submit path (`SubmitMessageMsg` → `submit()` → `backend.Start(...)`).

**Commit (code):** pinocchio da5f276 — "Fix chat start: disable model autoStartBackend"

### What I did
- Ran real provider calls via geppetto examples:
  - `cd geppetto && go run ./cmd/examples/simple-inference simple-inference "Say hello." --pinocchio-profile 4o-mini --ai-engine gpt-4o-mini --ai-api-type openai`
  - `cd geppetto && go run ./cmd/examples/simple-streaming-inference simple-streaming-inference "Write one sentence about penguins." --pinocchio-profile 4o-mini --ai-engine gpt-4o-mini --ai-api-type openai --verbose`
  - `cd geppetto && go run ./cmd/examples/generic-tool-calling generic-tool-calling "What's the weather in Paris and what is 2+2?" --pinocchio-profile 4o-mini --ai-engine gpt-4o-mini --ai-api-type openai --tools-enabled --max-iterations 3`
  - `cd geppetto && go run ./cmd/examples/openai-tools test-openai-tools --ai-api-type openai-responses --ai-engine gpt-5-mini --mode thinking --prompt "What is 23*17 + 55?"`
- Reproduced pinocchio chat “no inference starts” in tmux, then fixed it:
  - `pinocchio/pkg/cmds/cmd.go` now always passes `bobatea_chat.WithAutoStartBackend(false)`.
- Re-ran pinocchio chat in tmux with OpenAI Responses + gpt-5-mini; verified:
  - streaming thinking events (`EventThinkingPartial`)
  - multiple submits (two follow-up questions)
  - final completion event delivered and UI entity marked completed

### Why
- `go test ./...` is not sufficient for inference lifecycle work; we need to validate actual event routing and multi-turn UI runs.

### What worked
- geppetto examples produced correct output (single-pass, streaming, tools).
- OpenAI Responses thinking mode works with `--ai-engine gpt-5-mini`.
- pinocchio TUI successfully ran multi-turn in tmux once `WithAutoStartBackend` was disabled.

### What didn't work
- Initial pinocchio TUI run “froze” due to `WithAutoStartBackend(true)` triggering `StartBackendMsg` while `startBackend()` no longer starts backend.

### What I learned
- Any caller still using `WithAutoStartBackend(true)` is now “footgun”-level risky unless `startBackend()` is re-wired to call backend.Start (or removed entirely).

### What was tricky to build
- Distinguishing two separate “auto-start” concepts:
  - model init auto-start (`WithAutoStartBackend` → `StartBackendMsg`), and
  - auto-submit a rendered prompt (`ReplaceInputTextMsg` + `SubmitMessageMsg`).

### What warrants a second pair of eyes
- Confirm that no other pinocchio paths still pass `WithAutoStartBackend(true)` (or depend on it), since the behavior is now inconsistent with the new prompt flow.

### What should be done in the future
- Consider deleting `StartBackendMsg`/`startBackend()` from bobatea entirely (or re-introducing backend.Start there), now that submit() is the canonical start path.

### Code review instructions
- Start at:
  - `pinocchio/pkg/cmds/cmd.go`
  - `bobatea/pkg/chat/model.go` (`Init()` autoStartBackend and `startBackend()` behavior)
- Validate by rerunning the geppetto commands above and the pinocchio tmux smoke in the MO-004 playbook.

## Related

<!-- Link to related documents or resources -->

## Step 7: Remove bobatea AutoStartBackend/StartBackendMsg (cutover cleanup)

This step finishes the cleanup that the prior analysis proposed: `WithAutoStartBackend`/`StartBackendMsg`/`startBackend()` were misleading and actively dangerous after the “new prompt flow” refactor, because they flipped the UI into streaming state without actually calling `Backend.Start(...)`. Now that pinocchio can reliably auto-submit a first prompt by sending `ReplaceInputTextMsg` + `SubmitMessageMsg`, the “autostart backend” concept is unnecessary.

I removed those APIs from bobatea entirely and deleted pinocchio’s last remaining call site. This makes the model contract crisp: **only `SubmitMessageMsg` triggers inference**, and any “automatic first inference” is implemented by auto-submitting a prompt.

**Commit (code):**
- bobatea c2a08dc — "Remove chat AutoStartBackend/StartBackendMsg"
- pinocchio 930b461 — "Stop using removed bobatea AutoStartBackend"

### What I did
- Deleted `StartBackendMsg` from `bobatea/pkg/chat/user_messages.go`.
- Deleted `WithAutoStartBackend`, the `autoStartBackend` field, the `StartBackendMsg` init hook, and `startBackend()` from `bobatea/pkg/chat/model.go`.
- Updated pinocchio to stop referencing `WithAutoStartBackend` in `pinocchio/pkg/cmds/cmd.go`.
- Ran:
  - `go test ./... -count=1` in `bobatea/`
  - `go test ./... -count=1` in `pinocchio/`

### Why
- The StartBackend path had no prompt to pass to `Backend.Start(ctx, prompt string)`, so it could not be correct.
- Removing it reduces lifecycle ambiguity and eliminates a class of “UI stuck generating” hangs.

### What worked
- Both repos build/tests pass.

### What didn't work
- Bobatea’s pre-commit lint initially blocked the commit due to an unrelated Watermill deprecation warning; fixed by switching to `AddConsumerHandler`.

### What I learned
- A UI state transition named “start backend” is too easy to misuse unless it is *guaranteed* to start inference.

### What was tricky to build
- Coordinating cutover across repos because pinocchio imports bobatea directly (no compatibility layer allowed).

### What warrants a second pair of eyes
- Whether any non-workspace consumers depend on `WithAutoStartBackend` (we removed it with no compatibility).

### What should be done in the future
- N/A

### Code review instructions
- Review:
  - `bobatea/pkg/chat/model.go`
  - `bobatea/pkg/chat/user_messages.go`
  - `pinocchio/pkg/cmds/cmd.go`
- Validate:
  - `go test ./... -count=1` in `bobatea/` and `pinocchio/`
