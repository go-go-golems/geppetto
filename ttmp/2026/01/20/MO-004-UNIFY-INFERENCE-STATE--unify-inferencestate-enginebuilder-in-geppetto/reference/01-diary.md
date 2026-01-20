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
