---
Title: Diary
Ticket: MO-004-UNIFY-INFERENCE-STATE
Status: active
Topics:
    - inference
    - architecture
    - webchat
    - prompts
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/builder/builder.go
      Note: New shared EngineBuilder interface
    - Path: geppetto/pkg/inference/core/session.go
      Note: New shared Session Runner
    - Path: geppetto/pkg/inference/state/state.go
      Note: New shared InferenceState
    - Path: geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/design-doc/03-inferencestate-enginebuilder-core-architecture.md
      Note: Primary design doc being implemented in MO-004
ExternalSources: []
Summary: Implementation diary for moving InferenceState/EngineBuilder into geppetto and unifying callers.
LastUpdated: 2026-01-20T00:00:00Z
WhatFor: Track the step-by-step work for MO-004.
WhenToUse: Update after each meaningful implementation/debug step and each commit.
---



# Diary

## Goal

Move the core inference-session primitives (InferenceState + EngineBuilder contract + Runner interface and Session implementation) into geppetto so TUI/CLI/webchat can share a single inference orchestration core.

## Step 1: Create MO-004 ticket workspace and diary

This step created a clean ticket workspace dedicated to moving InferenceState/EngineBuilder into geppetto and unifying call sites. Separating this from MO-003 keeps the document-heavy API exploration distinct from the concrete implementation work that follows.

**Commit (code):** N/A

### What I did
- Created ticket `MO-004-UNIFY-INFERENCE-STATE` with docmgr.
- Created a new diary doc for MO-004.

### Why
- MO-004 is the execution phase: move types into geppetto and start wiring apps to them.

### What worked
- Ticket + diary created successfully.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Review ticket scaffold under `geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/`.

### Technical details
- `docmgr ticket create-ticket --ticket MO-004-UNIFY-INFERENCE-STATE ...`

## Step 2: Implement geppetto-owned InferenceState + Runner Session scaffolding

This step begins the actual code extraction into geppetto. I implemented a geppetto-owned `InferenceState` (run/cancel bookkeeping + current turn + engine handle) and a geppetto-owned `Session` that implements a minimal `Runner` interface (`RunInference(ctx, seed)`).

The Session captures stable dependencies (tool registry/config, event sinks, snapshot hook, optional persister) so call sites don’t pass a long list of arguments each time. This mirrors the working shape we saw in go-go-mento webchat, but keeps it UI-agnostic.

**Commit (code):** N/A

### What I did
- Added `geppetto/pkg/inference/state/state.go` implementing `InferenceState`.
- Added `geppetto/pkg/inference/core/session.go` implementing:
  - `Runner` interface
  - `Session.RunInference` supporting single-pass and tool-loop modes
  - event sinks + snapshot hook wiring via context
  - cancellation via `InferenceState.CancelRun()`
- Added `geppetto/pkg/inference/builder/builder.go` defining a geppetto-level `EngineBuilder` interface (no lifecycle injection).

### Why
- These primitives are shared across TUI/CLI/webchat. They belong in geppetto.
- A Session object matches real usage (long-lived per conversation/tab) and keeps the per-call API small.

### What worked
- The Session uses geppetto’s existing tool loop (`toolhelpers.RunToolCallingLoop`) and event sink context propagation.

### What didn't work
- N/A

### What I learned
- `toolhelpers.RunToolCallingLoop` already provides the canonical tool-loop core; our Session just needs to supply registry/config and hook context.

### What was tricky to build
- Ensuring cancellation is safe: StartRun + SetCancel + FinishRun + deferred cancel.

### What warrants a second pair of eyes
- Confirm the EngineBuilder interface shape is general enough for pinocchio and moments, not just go-go-mento.

### What should be done in the future
- Migrate go-go-mento webchat’s local InferenceState to a thin alias over geppetto’s InferenceState.

### Code review instructions
- Start with:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/state/state.go`
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/core/session.go`
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/builder/builder.go`

### Technical details
- Session single-pass: `state.Eng.RunInference(ctx, seed)`
- Session tool-loop: `toolhelpers.RunToolCallingLoop(ctx, state.Eng, seed, registry, cfg)`

## Step 3: Analyze moments webchat router migration to shared InferenceState

This step mapped the current moments webchat state and inference loop wiring (router + conversation structs) to the new geppetto-owned inference core. The key finding is that moments currently conflates lifecycle/transport and inference state inside `Conversation`; migrating cleanly means replacing the `RunID/Turn/Eng/running/cancel` fields with a single `*state.InferenceState` and driving inference via a `core.Session` runner.

I captured the current flow (WS join builds engine/sink, prompt resolver inserts system prompt, chat handler mutates Turn then runs inference) and then provided a concrete migration plan that keeps ConvManager and websocket streaming unchanged while moving just the inference-session core to geppetto.

**Commit (code):** N/A

### What I did
- Read moments webchat router and conversation implementation.
- Wrote a detailed analysis doc explaining current structure and a step-by-step migration plan to `geppetto/pkg/inference/state` + `geppetto/pkg/inference/core`.

### Why
- We need to migrate moments in a controlled way after we have a shared inference core in geppetto.

### What worked
- The mapping is straightforward because moments already stores the minimal triple (RunID, Turn, Eng), which matches InferenceState.

### What didn't work
- N/A

### What I learned
- Moments does profile prompt resolution at websocket join time and inserts a system block directly into the Turn; keeping that behavior is fine as long as system prompt insertion remains idempotent.

### What was tricky to build
- Identifying which parts are lifecycle-only (connections/readers) vs inference-core (run/cancel, turn/engine storage).

### What warrants a second pair of eyes
- Confirm whether moments tool-loop behavior differs materially from geppetto toolhelpers (step mode, tool auth), so we don’t force unification too early.

### What should be done in the future
- After migrating state, revisit whether moments should use `core.Session` tool-loop path or keep a custom loop with an injected executor.

### Code review instructions
- Review the analysis doc:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/analysis/01-moments-webchat-router-migration-to-geppetto-inferencestate-session.md`

### Technical details
- Primary current files:
  - `moments/backend/pkg/webchat/router.go`
  - `moments/backend/pkg/webchat/conversation.go`

## Step 4: Unify event sink publishing via context (engines attach config sinks to ctx)

This step addressed a practical blocker for a unified runner: tool loops and middleware publish events via `events.PublishEventToContext`, while provider engines also publish events via both engine-config sinks and context sinks. That double-path makes it hard to safely attach sinks in a shared Session without either missing tool-loop events or double-publishing inference events.

I changed the provider engines to attach their configured sinks into the run context at the start of `RunInference`, and then publish events only through `events.PublishEventToContext`. This preserves the “engine has configured sinks” UX (e.g., pinocchio builds engines with `engine.WithSink(...)`) while ensuring that *all* context-published events (including tool-loop events) are delivered to the same sinks.

**Commit (code):** N/A

### What I did
- Updated provider engines to:
  - call `ctx = events.WithEventSinks(ctx, e.config.EventSinks...)` at the start of `RunInference`
  - remove direct loops over `e.config.EventSinks` in `publishEvent`
- Applied this to:
  - `geppetto/pkg/steps/ai/openai_responses/engine.go`
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `geppetto/pkg/steps/ai/claude/engine_claude.go`
  - `geppetto/pkg/steps/ai/gemini/engine_gemini.go`

### Why
- A shared runner/session needs a single event publishing mechanism.
- Tool loops publish via context; engines had a separate “config sinks” path.
- Attaching config sinks to context makes tool-loop events and engine events flow through the same sinks without requiring callers to add context sinks manually.

### What worked
- `go test ./...` passes in geppetto after the changes.

### What didn't work
- N/A

### What I learned
- The provider engines already publish to context; the missing piece was ensuring the engine-config sinks were available through the same context path.

### What was tricky to build
- Avoiding double-publish when both config sinks and context sinks are used: this change makes it easier to standardize on “configure sinks on the engine” and let the engine attach them to context.

### What warrants a second pair of eyes
- Verify that any callers that *also* attach the same sink to context do not end up with duplicates (we should standardize on one mechanism per app).

### What should be done in the future
- Update docs/design to recommend a single sink wiring strategy per app (engine config sinks OR explicit context sinks, but not both).

### Code review instructions
- Review:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/engine.go`
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/claude/engine_claude.go`
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/gemini/engine_gemini.go`

### Technical details
- Engines now publish via context only; config sinks are attached to context at the start of `RunInference`.

## Step 5: Make SystemPrompt middleware idempotent (remove need for history filtering)

This step made the system prompt middleware safe to run repeatedly on a growing conversation snapshot. Previously, we relied on downstream “filter blocks before persistence” logic to prevent system prompt duplication, which is fragile and makes provider validation (like OpenAI Responses “reasoning item must be followed”) more likely to break when turns are reconstructed incorrectly.

By making the middleware idempotent, the persisted turn can remain “complete” (including system blocks), and we can avoid special casing system prompt blocks during state persistence.

**Commit (code):** 4594a4b — "Make system prompt middleware idempotent"

### What I did
- Updated the system prompt middleware to detect whether it already inserted the system prompt and avoid inserting again.
- Recorded this decision in git history and removed reliance on “filter system blocks out of persisted history” as a correctness mechanism.

### Why
- We want a single persisted “conversation snapshot turn” that can be reused as-is across runs.
- Provider APIs (Responses) validate item ordering more aggressively; duplicated or mis-ordered synthetic blocks are easy footguns.

### What worked
- Repeated runs no longer accumulate redundant system prompt blocks.

### What didn't work
- N/A

### What I learned
- Middleware idempotency is the right place to fix duplication, not persistence-time filtering.

### What was tricky to build
- Deciding what constitutes “already inserted”: system prompt equality vs. metadata tagging.

### What warrants a second pair of eyes
- Confirm the idempotency detection is strict enough (no false positives) but robust across minor formatting differences.

### What should be done in the future
- Standardize a metadata marker on injected blocks so idempotency can be keyed on a stable tag rather than string equality.

### Code review instructions
- Review the system prompt middleware implementation and its matching logic.

### Technical details
- Commit includes the concrete idempotency check and any associated unit tests (if present).

## Step 6: Pinocchio: replace reduceHistory with a persisted conversation snapshot

This step migrated pinocchio’s chat history handling away from `reduceHistory()` (flattening prior UI entities into a single synthetic Turn) and toward maintaining a single persisted “conversation snapshot” that is extended each run. This aligns the TUI and webchat with the stricter ordering requirements of the OpenAI Responses API and eliminates a class of bugs where the reconstructed input accidentally drops or misorders blocks.

The core idea is: store the full “canonical” turn (system/user/assistant/tool blocks) between runs, and on each new prompt, append a new user block to that snapshot to form the seed.

**Commit (code):** ccf9c61 — "Replace reduceHistory with ConversationState"

### What I did
- Replaced the `reduceHistory`-driven turn reconstruction in pinocchio with a persisted conversation snapshot abstraction.
- Ensured the seed turn for each run is derived from the stored snapshot + appended user prompt.

### Why
- Provider validation (Responses) makes ordering errors fatal (400) instead of “best effort”.
- We want one canonical place that “what the conversation is” is stored; not derived from UI render state.

### What worked
- Multi-turn chat stops relying on UI timeline reconstruction.

### What didn't work
- N/A

### What I learned
- UI render state and inference state should not be conflated; inference needs its own source of truth.

### What was tricky to build
- Making sure the snapshot includes all relevant block kinds (tool calls/uses, reasoning, etc.) without leaking UI-only artifacts.

### What warrants a second pair of eyes
- Confirm we never drop tool-use blocks when persisting state (those are required for Responses ordering).

### What should be done in the future
- Replace pinocchio’s local runner patterns with the geppetto Session/InferenceState core (the actual MO-004 goal).

### Code review instructions
- Review the pinocchio changes that removed `reduceHistory` and introduced snapshot-based seeding.

### Technical details
- This step sets up the later migration to geppetto’s `InferenceState` by treating the persisted turn as the single source of truth.

## Step 7: Pinocchio: unify runner shape and remove history filtering

This step cleaned up pinocchio’s runner API and removed “filter system prompt blocks” behavior that was compensating for non-idempotent middleware. The runner now focuses on: “run inference (maybe tool-loop) with a seed turn, update the stored snapshot state,” with fewer special cases.

This also removed temporary debug tap hooks that were useful during the Responses ordering investigation but shouldn’t remain in the production path.

**Commit (code):** f0f8ad3 — "Decouple runner from prompt and drop systemprompt filtering"

### What I did
- Simplified pinocchio runner code to accept a pre-built seed (snapshot + prompt already applied by caller).
- Removed “filter system prompt blocks” update option (now handled by idempotent middleware).
- Removed webchat debug tap hooks once the issue was understood.

### Why
- “Prompt construction” is caller/UI responsibility; runner should not have to know about prompt strings.
- Filtering blocks at persistence time is brittle and breaks provider input validation assumptions.

### What worked
- Cleaner separation between “build seed” and “run inference”.

### What didn't work
- N/A

### What I learned
- A small runner interface is only stable if we move all seed construction upstream.

### What was tricky to build
- Ensuring the cleaned-up runner still supports tool-loop runs without extra state threading.

### What warrants a second pair of eyes
- Confirm no call sites still pass (or depend on) filtered state updates.

### What should be done in the future
- Delete pinocchio runner entirely and migrate call sites to geppetto `core.Session` (MO-004).

### Code review instructions
- Review pinocchio runner changes and ensure no remaining `reduceHistory` usage.

### Technical details
- This commit prepares pinocchio to be migrated to geppetto’s `InferenceState` + `Session`.

## Step 8: WIP — migrate pinocchio TUI/webchat to geppetto InferenceState + core.Session

This step starts the real MO-004 consumer migration: stop using pinocchio’s `conversation.ConversationState`/runner and instead use geppetto’s `InferenceState` and `core.Session` runner. The TUI is the first target because it’s the simplest execution model (single-pass inference, engine already has a sink configured).

Along the way, I discovered a subtle but important API constraint: Bubble Tea requires marking the backend “running” before returning the command; otherwise `Start()` can be called again before the command executes. That means `Session.RunInference()` (which calls `StartRun`) can’t be used as-is for Bubble Tea without a “started run” variant.

**Commit (code):** N/A (in progress in working tree)

### What I did
- Investigated pinocchio call sites with:
  - `rg -n "ConversationState|SnapshotForPrompt|runner\\.Run\\(" pinocchio -S`
  - `sed -n '1,220p' pinocchio/pkg/inference/runner/runner.go`
  - `sed -n '1,220p' pinocchio/pkg/ui/backend.go`
  - `sed -n '1,220p' pinocchio/pkg/webchat/conversation.go`
- Began refactoring:
  - `pinocchio/pkg/ui/backend.go` to store a `*state.InferenceState` and call `core.Session` instead of pinocchio runner.
  - `pinocchio/pkg/webchat/conversation.go` to replace `ConversationState` with `InferenceState`.
- Added `core.Session.RunInferenceStarted(...)` to support “run already marked started” flows.
- Added `InferenceState.HasCancel()` and adjusted Session cancellation wiring to ensure cancellation works both when Session starts the run and when the caller starts the run.

### Why
- pinocchio’s TUI and webchat should share the same geppetto inference core.
- We need to support cancellation and “already running” checks uniformly across UIs.

### What worked
- The shape of the migration is straightforward: the persisted snapshot becomes `InferenceState.Turn`, and each prompt becomes “append user block, run”.

### What didn't work
- First patch attempt failed because my `apply_patch` hunk didn’t match the exact file contents (expected import block differed). I re-opened `pinocchio/pkg/ui/backend.go` and re-applied the patch with the correct context.

### What I learned
- Bubble Tea backends need a “mark started now, run later” interface; it’s not just a convenience.

### What was tricky to build
- Session cancellation ownership: avoid losing cancel when the caller establishes `context.WithCancel` ahead of time.

### What warrants a second pair of eyes
- The final “who owns cancel + who calls FinishRun” rules for Session vs. caller; we need to keep this consistent across TUI/webchat.

### What should be done in the future
- Once pinocchio migration is complete, delete `pinocchio/pkg/inference/runner` and migrate remaining call sites.

### Code review instructions
- Focus on:
  - `pinocchio/pkg/ui/backend.go` (migration to `InferenceState` + `core.Session`)
  - `pinocchio/pkg/webchat/conversation.go` (state swap)
  - `geppetto/pkg/inference/core/session.go` and `geppetto/pkg/inference/state/state.go` (new `RunInferenceStarted` + cancel changes)

### Technical details
- New Session API:
  - `RunInference(ctx, seed)` (starts run + sets cancel)
  - `RunInferenceStarted(ctx, seed)` (assumes run already started; sets cancel only if missing)

## Step 9: Migrate geppetto cmd/examples to EngineBuilder + InferenceState + Session

This step updated the `geppetto/cmd/examples/*` programs to consistently use the same “inference core” abstractions we’re standardizing on: `EngineBuilder` for engine construction, `InferenceState` as the long-lived state holder, and `core.Session` as the thin runner. This makes the examples both a validation surface and a reference implementation for downstream apps (pinocchio, moments).

I also had to align the Session API with Bubble Tea needs by adding a “run already started” variant and by making cancel wiring robust to “caller already created a cancel context”.

**Commit (code):** e009123 — "Examples: adopt EngineBuilder + Session runner"

### What I did
- Added `geppetto/cmd/examples/internal/examplebuilder/builder.go` implementing `builder.EngineBuilder` for examples.
- Updated example commands to run via `InferenceState` + `core.Session` instead of calling provider engines or `toolhelpers.RunToolCallingLoop` directly:
  - `cmd/examples/simple-inference/main.go`
  - `cmd/examples/simple-streaming-inference/main.go`
  - `cmd/examples/generic-tool-calling/main.go`
  - `cmd/examples/middleware-inference/main.go`
  - `cmd/examples/openai-tools/main.go`
  - `cmd/examples/claude-tools/main.go`
- Extended the Session/state core:
  - `InferenceState.HasCancel()` for detecting whether a caller already stored a cancel.
  - `Session.RunInferenceStarted(...)` for “already marked started” UIs.

### Why
- Examples should demonstrate the “blessed” composition style that UIs will use.
- We need a single place to encode the rules for cancellation, tool-loop orchestration, and event sink wiring.

### What worked
- `go test ./...` passed in `geppetto/`.
- Pre-commit lint/test hooks passed after running gofmt.

### What didn't work
- `git commit` initially failed because `cmd/examples/internal/examplebuilder/builder.go` was not gofmt’d:
  - Reported by `golangci-lint` as: `File is not properly formatted (gofmt)`
  - Fixed by running: `gofmt -w cmd/examples/internal/examplebuilder/builder.go`

### What I learned
- If we want tool-loop events to always publish, we need a clear “where sinks live” rule. For examples, we now build the sink alongside the engine but attach it via the Session run context.

### What was tricky to build
- Avoiding duplicate event delivery when both the engine config and the run context attach the same sinks. (For examples, we prefer Session-attached sinks.)

### What warrants a second pair of eyes
- Confirm the Session cancel ownership rules are sound across:
  - “Session starts the run” calls
  - “Caller starts the run (Bubble Tea)” calls

### What should be done in the future
- Add tests for the Session “started” path and cancel behavior (edge cases: cancel set late, cancel unset, multiple StartRun attempts).

### Code review instructions
- Start with:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/internal/examplebuilder/builder.go`
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/core/session.go`
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/state/state.go`
- Then scan example changes in `geppetto/cmd/examples/*/main.go`.

### Technical details
- Example EngineBuilder usage:
  - `engBuilder.Build("", profileSlug, overrides)` returns `(engine, sink, config)`.
- Session usage:
  - `sess := &core.Session{State: inf, EventSinks: []events.EventSink{sink}, Registry: reg, ToolConfig: &cfg}`
  - `updated, err := sess.RunInference(ctx, seed)`
