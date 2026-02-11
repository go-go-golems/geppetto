---
Title: Diary
Ticket: MO-003-UNIFY-INFERENCE
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
    - Path: geppetto/pkg/inference/middleware/systemprompt_middleware.go
      Note: Idempotent system prompt insertion (Step 5)
    - Path: geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/analysis/11-go-go-mento-webchat-conversation-manager-and-run-alignment.md
      Note: Analysis doc produced in Step 7
    - Path: geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/design-doc/03-inferencestate-enginebuilder-core-architecture.md
      Note: |-
        Design doc produced in Step 8
        Updated in Step 9 to introduce Session API
    - Path: go-go-mento/go/pkg/webchat/conversation_manager.go
      Note: Primary lifecycle manager analyzed in Step 7
    - Path: moments/backend/pkg/webchat/conversation_debugtap.go
      Note: Moments DebugTap for pre-inference snapshots (Step 2)
    - Path: moments/backend/pkg/webchat/router.go
      Note: Moments DebugTap wiring (Step 2)
    - Path: pinocchio/pkg/inference/runner/runner.go
      Note: Shared inference runner introduced in Step 3.
    - Path: pinocchio/pkg/ui/backend.go
      Note: TUI backend refactor to use shared runner (Step 3).
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Removed redundant snapshot/update helpers after runner migration (Step 4).
    - Path: pinocchio/pkg/webchat/router.go
      Note: Webchat run loop now uses shared runner (Step 4).
ExternalSources: []
Summary: Diary for MO-003 implementation steps.
LastUpdated: 2026-01-16T15:22:40-05:00
WhatFor: Track implementation steps for unifying inference between pinocchio TUI and webchat.
WhenToUse: Update after each implementation step or significant discovery.
---







# Diary

## Goal

Track the step-by-step implementation of shared inference orchestration across pinocchio TUI and webchat, with follow-on guidance for migrating Moments later.

## Step 0: Create MO-003 ticket and migrate docs into it

I created the MO-003 ticket workspace and moved the unification-related analysis/design docs out of MO-002 so future work is scoped correctly. This set a clean baseline for the unification work and made sure the new ticket has the right context for planning and execution.

I also created the task list for MO-003 and uploaded the moved analysis/design docs to reMarkable so they remain accessible after the ticket re-org.

**Commit (code):** 5541dc5 — "Docs: reorganize MO-003 ticket and update diary"

### What I did
- Created MO-003 ticket workspace via `docmgr ticket create-ticket`.
- Moved analysis + design docs from MO-002 to MO-003 (and moved analysis 01–06 back to MO-002 later).
- Created MO-003 tasks and updated the doc tree.
- Uploaded the analysis + design docs to reMarkable under the new ticket path.

### Why
- The unification work is distinct from the MO-002 bugfix scope and needs a dedicated ticket.
- Consolidating the docs under MO-003 prevents confusion and keeps later edits focused.

### What worked
- Doc moves preserved relative paths and updated ticket frontmatter automatically.
- reMarkable uploads mirrored the new ticket structure successfully.

### What didn't work
- N/A

### What I learned
- `docmgr doc move` is safe for ticket re-orgs as long as the destination subdirs exist.

### What was tricky to build
- Coordinating the doc move + reMarkable upload without breaking references.

### What warrants a second pair of eyes
- Confirm the doc move list matches the intended scope for MO-003 vs MO-002.

### What should be done in the future
- N/A

### Code review instructions
- Review ticket structure under `geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/`.

### Technical details
- Doc move command: `docmgr doc move --doc <path> --dest-ticket MO-003-UNIFY-INFERENCE`.
- reMarkable upload used `--mirror-ticket-structure`.

## Step 1: Add prompt-resolution + step-mode analysis updates

I wrote and refined the prompt-resolution analysis doc, then expanded it to include Moments step mode details and explicit tool result event emission. This captures the reasoning around why tool result events live in the tool loop and clarifies that inference engines emit only inference events.

Because an earlier upload already existed, I re-uploaded the updated analysis under a new name to reMarkable, then removed the temporary copy locally.

**Commit (code):** 5541dc5 (initial doc); updates after that are not yet committed.

### What I did
- Authored `analysis/09-prompt-resolution-in-router-and-middlewares.md` with prompt-resolver call sites and prompt slot behavior.
- Added a new section explaining step mode and explicit tool result events.
- Uploaded the updated doc to reMarkable under a new name to avoid overwrite errors.

### Why
- We needed a single reference that explains the router vs middleware resolution split and event emission responsibilities.
- The new step-mode section clarifies why tool events are emitted at the loop level.

### What worked
- The doc update captured the exact call sites and clarified the architectural boundary.
- reMarkable upload succeeded after using a new filename.

### What didn't work
- Initial upload failed with: `entry already exists (use --force to recreate, --content-only to replace content)`.

### What I learned
- The reMarkable upload tool requires a new filename or `--force` for updates.

### What was tricky to build
- Keeping the doc consistent with both router and middleware code paths while adding step-mode context.

### What warrants a second pair of eyes
- Verify the tool-result emission explanation matches all tool-loop call sites.

### What should be done in the future
- Commit the updated analysis doc changes in MO-003.

### Code review instructions
- Review `geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/analysis/09-prompt-resolution-in-router-and-middlewares.md`.

### Technical details
- reMarkable upload used a temporary filename `09-prompt-resolution-in-router-and-middlewares-step-mode.md`.

## Step 2: Add Moments webchat debugtap for pre-inference turn snapshots

I added a DebugTap implementation for Moments webchat that writes each pre-inference Turn snapshot to `/tmp/conversations/<conv-id>/NN-before-inference.yaml`. This uses the existing DebugTap interface in geppetto’s Responses engine to persist turns without adding extra hooks in the loop.

The change is currently implemented in the Moments backend but still needs a commit.

**Commit (code):** N/A (pending)

### What I did
- Added `moments/backend/pkg/webchat/conversation_debugtap.go` implementing `engine.DebugTap`.
- Wired it into `moments/backend/pkg/webchat/router.go` to attach the tap on the run context.
- Added env override `MOMENTS_CONVERSATION_TAP_DIR` (set to `off`/`false` to disable).

### Why
- We needed a standardized way to capture the pre-inference Turn in Moments without duplicating snapshot hooks.

### What worked
- The tap integrates cleanly with the engine’s `OnTurnBeforeConversion` hook.

### What didn't work
- N/A

### What I learned
- The Responses engine already emits YAML snapshots through DebugTap, so the webchat just needs to attach it.

### What was tricky to build
- Ensuring we sanitize conversation IDs for filesystem safety.

### What warrants a second pair of eyes
- Verify that the tap doesn’t introduce performance issues under high throughput.

### What should be done in the future
- Commit the Moments debugtap changes after review.

### Code review instructions
- Review `moments/backend/pkg/webchat/conversation_debugtap.go` and the router wiring.

### Technical details
- Output path format: `/tmp/conversations/<conv-id>/<NN>-before-inference.yaml`.

## Step 3: Create shared runner and migrate TUI backend

I introduced a shared inference runner in pinocchio and refactored the TUI backend to use it. This consolidates snapshot creation, optional tool-loop execution, and conversation-state updates into a single helper so the TUI no longer hand-rolls its own snapshot/update logic.

This step establishes the pattern we will reuse for webchat in the next step: build a snapshot from `ConversationState`, run the engine (with or without tools), and persist the updated state consistently.

**Commit (code):** 2df3b2c — "Add shared inference runner for TUI"

### What I did
- Added `pinocchio/pkg/inference/runner/runner.go` with shared Run/Snapshot/Update helpers.
- Swapped the TUI engine backend to call `runner.Run` instead of direct `RunInference` and local snapshot/update helpers.
- Removed unused snapshot/update methods from the TUI backend after the runner migration.

### Why
- We want one shared orchestration path for inference across TUI and webchat.
- Centralizing snapshot + update behavior reduces divergence and eliminates duplicate logic.

### What worked
- The runner cleanly encapsulates snapshot + update and supports optional tool loops.
- Pinocchio tests and lint ran cleanly during the commit hooks.

### What didn't work
- N/A

### What I learned
- The TUI backend had redundant snapshot/update helpers that can now be removed safely.
- A single Run entrypoint is sufficient for TUI use cases without tool execution.

### What was tricky to build
- Ensuring the runner handles nil state safely while preserving run IDs when available.

### What warrants a second pair of eyes
- Confirm that the new runner’s default ToolConfig behavior is correct for future webchat usage.

### What should be done in the future
- Migrate pinocchio webchat to use the runner (next step).

### Code review instructions
- Start in `pinocchio/pkg/inference/runner/runner.go` to review the shared orchestration logic.
- Review `pinocchio/pkg/ui/backend.go` for the TUI switch-over.

### Technical details
- Commit: `2df3b2c`
- Hooks ran: `go test ./...`, `go generate ./...`, `go build ./...`, `golangci-lint run`, `go vet`.

## Step 4: Migrate pinocchio webchat to shared runner

I refactored the webchat run loop to use the shared runner instead of doing snapshotting and tool-loop orchestration inline. This removes duplicate snapshot/update helpers on the conversation and centralizes event sink + snapshot hook wiring in one place.

This step aligns the webchat flow with the TUI path introduced in Step 3 and makes subsequent unification (e.g., shared runner options) straightforward.

**Commit (code):** 0fdcb56 — "Use shared runner in webchat"

### What I did
- Rewired both webchat handlers to call `runner.Run` with tool registry, tool config, event sinks, and snapshot hooks.
- Removed the now-unused per-conversation snapshot/update helpers and state mutex.
- Added runner import and centralized error handling in the run loop.

### Why
- Webchat was duplicating inference orchestration logic that now lives in the shared runner.
- A single run entrypoint reduces future divergence and simplifies reasoning about ordering.

### What worked
- The webchat run loop now uses the same snapshot + update path as the TUI.
- Pre-commit tests and lint passed after cleanup of unused helpers.

### What didn't work
- Initial commit attempt failed with compile/lint errors:
  - `undefined: runner` and `declared and not used: updatedTurn` in `pkg/webchat/router.go`.
  - Unused `stateMu`, `snapshotForPrompt`, `updateStateFromTurn` in `pkg/webchat/conversation.go`.
- Fix: added the runner import, removed unused updatedTurn binding, and deleted the unused helpers.

### What I learned
- The webchat conversation helpers were only needed for snapshot/update and can be fully replaced by the runner.

### What was tricky to build
- Ensuring webchat’s system-prompt filtering is preserved by passing `FilterBlocks` in the runner Update options.

### What warrants a second pair of eyes
- Confirm no other code paths relied on the removed conversation snapshot/update helpers.

### What should be done in the future
- Start the Moments migration plan by mapping its tool loop to the runner’s options.

### Code review instructions
- Review `pinocchio/pkg/webchat/router.go` for the new runner usage and option wiring.
- Review `pinocchio/pkg/webchat/conversation.go` for removal of redundant helpers.

### Technical details
- Commit: `0fdcb56`
- Hooks ran: `go test ./...`, `go generate ./...`, `go build ./...`, `golangci-lint run`, `go vet`.

## Step 5: Make system prompt middleware idempotent

I updated the geppetto system prompt middleware to detect a previously-inserted system prompt block and skip reinsertion. This eliminates the need to strip systemprompt blocks when persisting conversation state and makes the middleware safe to run on snapshots that already contain a system prompt.

**Commit (code):** 4594a4b — "Make system prompt middleware idempotent"

### What I did
- Added a metadata check for `middleware=systemprompt` in the system prompt middleware.
- Skipped reinsertion when a prior systemprompt block is already present.

### Why
- Persisting system prompt blocks should not cause prompt duplication across turns.
- Idempotency lets us remove the `FilterBlocks` workaround in pinocchio webchat.

### What worked
- Tests and lint passed in the geppetto pre-commit hooks.

### What didn't work
- N/A

### What I learned
- The middleware already tags inserted blocks with `middleware=systemprompt`, making this check straightforward.

### What was tricky to build
- Ensuring the skip path preserves middleware ordering by returning `next(ctx, t)` immediately.

### What warrants a second pair of eyes
- Verify no downstream code depends on the previous append behavior when a system block exists.

### What should be done in the future
- Remove `FilterBlocks` usage in pinocchio webchat runner options (next step).

### Code review instructions
- Review `geppetto/pkg/inference/middleware/systemprompt_middleware.go`.

### Technical details
- Commit: `4594a4b`
- Hooks ran: `go test ./...`, `go generate ./...`, `go build ./...`, `golangci-lint run`, `go vet`.

## Step 6: Decouple runner from prompt and remove systemprompt filtering

I refactored the shared runner to accept a pre-built Turn snapshot instead of a raw prompt string, then updated the TUI and webchat call sites to build the snapshot explicitly. This moves prompt-to-block construction out of `Run`, per the new contract, and makes the runner a pure execution + update routine.

With the system prompt middleware now idempotent, I removed the systemprompt filtering logic from webchat persistence. The webchat run loop now persists the full updated Turn without stripping systemprompt blocks.

**Commit (code):** f0f8ad3 — "Decouple runner from prompt and drop systemprompt filtering"

### What I did
- Updated `runner.Run` to accept a `seed *turns.Turn` and require callers to build snapshots.
- Updated TUI backend to call `runner.SnapshotForPrompt` and pass the seed into `Run`.
- Updated webchat run loops to build a snapshot first, then call `Run`.
- Removed the `filterSystemPromptBlocks` helper from webchat conversation state handling.

### Why
- The runner should not implicitly convert user prompts into blocks; that belongs to the caller.
- Idempotent system prompt middleware makes state filtering unnecessary.

### What worked
- Tests and lint passed after refactor.
- Webchat snapshot/error handling now happens before the run loop executes.

### What didn't work
- N/A

### What I learned
- The shared runner works best as a pure orchestrator once snapshot creation is externalized.

### What was tricky to build
- Ensuring error handling in webchat restores `conv.running`/`conv.cancel` when snapshot building fails.

### What warrants a second pair of eyes
- Confirm that removing systemprompt filtering does not reintroduce prompt duplication (should be safe now that middleware is idempotent).

### What should be done in the future
- Centralize snapshot creation + registry wiring in a shared helper (next unification step).

### Code review instructions
- Review `pinocchio/pkg/inference/runner/runner.go` for the new Run signature.
- Review `pinocchio/pkg/ui/backend.go` and `pinocchio/pkg/webchat/router.go` for updated call sites.
- Review `pinocchio/pkg/webchat/conversation.go` for removal of filtering helpers.

### Technical details
- Commit: `f0f8ad3`
- Hooks ran: `go test ./...`, `go generate ./...`, `go build ./...`, `golangci-lint run`, `go vet`.

## Step 7: Analyze go-go-mento webchat conversation manager vs Run-centric model

This step focused on understanding the go-go-mento webchat architecture in detail so we can anchor unification work to an existing, clean design. I traced the conversation lifecycle, the inference loop, streaming, and persistence paths, then mapped them to our current Run‑centric proposal to see where we are aligned and where we are still diverging.

The result is a new analysis document that explains the webchat architecture in textbook detail, with diagrams and pseudocode, and explicitly answers how `ConversationManager` and `InferenceState` already behave like a “run + current turn” model. This should clarify why the Run-centric approach is not a new direction, but rather a formalization of how go-go-mento already works.

**Commit (code):** N/A

### What I did
- Read core webchat files to map responsibilities and the inference lifecycle:
  - `go-go-mento/go/pkg/webchat/conversation_manager.go`
  - `go-go-mento/go/pkg/webchat/conversation.go`
  - `go-go-mento/go/pkg/webchat/inference_state.go`
  - `go-go-mento/go/pkg/webchat/loops.go`
  - `go-go-mento/go/pkg/webchat/turns_loader.go`
  - `go-go-mento/go/pkg/webchat/turns_persistence.go`
  - `go-go-mento/go/pkg/webchat/engine_builder.go`
  - `go-go-mento/go/pkg/webchat/stream_coordinator.go`
- Wrote a new analysis doc that explains the architecture and alignment with Run‑centric conversation state.
- Captured the full flow from Router → ConversationManager → InferenceState → ToolCallingLoop → persistence → streaming.

### Why
- The team wants to unify around the go-go-mento webchat structure; we needed a precise map of how it manages state and turns.
- Comparing it against the Run‑centric proposal highlights that go-go-mento already stores the “current turn” and keeps history in persistence.

### What worked
- The code structure in go-go-mento cleanly exposes lifecycle, loop, and streaming responsibilities, making the analysis straightforward.
- The analysis doc makes the “Run vs current turn” alignment explicit.

### What didn't work
- N/A

### What I learned
- go-go-mento is already Turn-centric: `InferenceState.Turn` is the single canonical snapshot in memory.
- A “Run” concept exists implicitly via `RunID` and persisted turns, even without an explicit `turns.Run` container.

### What was tricky to build
- Keeping the analysis exhaustive while separating lifecycle concerns (conversation manager) from streaming (event translator/stream coordinator) and persistence.

### What warrants a second pair of eyes
- Validate that the analysis accurately reflects how persisted turns are reconstructed and how RunID is threaded through events.

### What should be done in the future
- Decide whether to replace `InferenceState.Turn` with `turns.Run` or keep the current “last turn + DB history” model and formalize it with helper APIs.

### Code review instructions
- Start with the new analysis doc:
  - `geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/analysis/11-go-go-mento-webchat-conversation-manager-and-run-alignment.md`
- Cross-check against the webchat files listed in “What I did.”

### Technical details
- Primary entry point: `go-go-mento/go/pkg/webchat/router.go` (run loop + ToolCallingLoop).
- State container: `go-go-mento/go/pkg/webchat/inference_state.go`.
- Lifecycle coordinator: `go-go-mento/go/pkg/webchat/conversation_manager.go`.

## Step 8: Draft InferenceState + EngineBuilder core design

This step distilled the unification strategy down to the two most stable primitives in the existing codebase: `InferenceState` and `EngineBuilder`. I wrote a design document that treats these as the shared core for all UIs, preserves the existing ToolCallingLoop, and explicitly removes ConversationManager from the inference path.

The proposal makes persistence a caller‑supplied hook rather than a baked‑in side effect. This keeps the inference loop reusable for TUI, console, and webchat without hidden DB coupling while still allowing webchat to persist turns when needed.

**Commit (code):** N/A

### What I did
- Wrote a design doc specifying the shared inference core, centered on `InferenceState` + `EngineBuilder`.
- Defined a persistence hook interface so webchat can persist turns without the inference core depending on DB logic.
- Kept the existing ToolCallingLoop as the canonical execution path.

### Why
- We need a single inference core usable across TUI, console, and webchat.
- Conversation lifecycle (connections, eviction, stream coordinator) should remain downstream of inference.
- Persistence should be optional and provided by the caller.

### What worked
- The design cleanly separates inference from lifecycle and persistence.
- The proposal reuses existing types instead of introducing new ones.

### What didn't work
- N/A

### What I learned
- go-go-mento’s `InferenceState` already captures the minimal state we need: RunID, current Turn, engine.
- ToolCallingLoop can remain the standard execution path if we expose a persistence hook.

### What was tricky to build
- Making the persistence hook minimal while still supporting webchat’s turn persistence requirements.

### What warrants a second pair of eyes
- Confirm the proposed RunInference signature is sufficient for all existing call sites.
- Verify that removing ConversationManager injection doesn’t break any middleware or sink composition assumptions.

### What should be done in the future
- Decide where to place the shared inference core package (geppetto vs new shared module).

### Code review instructions
- Review the design doc:
  - `geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/design-doc/03-inferencestate-enginebuilder-core-architecture.md`

### Technical details
- The design explicitly retains `ToolCallingLoop` from `go-go-mento/go/pkg/webchat/loops.go`.

## Step 9: Refine Inference core API to a Session method

This step tightened the proposed inference-core interface after realizing the earlier `RunInference(ctx, state, seed, registry, loopOpts, persister)` function was too argument-heavy and didn’t match how real callers are structured. In practice, the callers (webchat router goroutine, TUI backend, CLI command handler) already have long-lived objects for state and configuration, so we should capture those once and expose a simple per-call method.

I updated the design doc to introduce a `Session` struct that holds `InferenceState`, tool registry, loop settings, and optional persistence hooks. The key outcome is that the per-call interface becomes the straightforward `session.RunInference(ctx, seed) -> (turn, error)` and matches the mental model of “run inference on this turn snapshot”.

**Commit (code):** N/A

### What I did
- Updated the `InferenceState + EngineBuilder core` design doc to add an explicit “who calls this?” section.
- Replaced the free function signature with a `Session` struct + method `RunInference(ctx, seed)`.
- Documented how persistence hooks fit, including options for intermediate snapshot handling.

### Why
- The original signature pushed too many stable dependencies (registry, loop opts, persister) into a per-call function.
- Real call sites naturally want to hold those dependencies at a higher level (per conversation, per TUI tab, per CLI invocation).

### What worked
- The `Session` API reduces surface complexity without losing configurability.
- The design now maps directly to go-go-mento’s existing structure (InferenceState + ToolCallingLoop).

### What didn't work
- N/A

### What I learned
- “RunInference(ctx, turn) -> turn” only works cleanly once a session/runner object exists to capture the non-turn arguments.

### What was tricky to build
- Keeping persistence pluggable without leaking conversation lifecycle concepts (ConversationManager, websocket pools) into the inference core.

### What warrants a second pair of eyes
- Ensure the single-pass (no tools) path and the tool-loop path are both correctly represented in the design.
- Validate the proposed snapshot-hook options (context-based vs explicit hook struct) against current `ToolCallingLoop` behavior.

### What should be done in the future
- Decide whether to generalize `ToolCallingLoop` hooks via explicit structs (Option B) and update call sites accordingly.

### Code review instructions
- Review the updated design doc:
  - `geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/design-doc/03-inferencestate-enginebuilder-core-architecture.md`

## Step 10: Tighten Session API (Runner interface, persister runID, geppetto ownership)

This step continued refining the inference-core API to match how we actually want to consume it across UIs. After removing the unnecessary TurnIndex hook, I updated the design to make the per-call surface an explicit interface (`Runner`) and to ensure persistence gets the one identifier it truly needs: the `runID`.

I also locked in the ownership decision: `InferenceState` and `EngineBuilder` belong in geppetto (the shared inference foundation), not in app repos. That keeps the “core” truly shared and prevents future drift.

**Commit (code):** N/A

### What I did
- Updated the design doc to:
  - Make `RunInference(ctx, seed) (*turns.Turn, error)` an explicit `Runner` interface.
  - Change `TurnPersister` to receive `runID` explicitly and derive `turnID` from `t.ID`.
  - Remove the snapshot-vs-final persister split discussion (single persister interface only).
  - Specify that `InferenceState` and `EngineBuilder` live in geppetto.
- Updated the upload script for the doc to use `remarquee upload md --force` for safe overwrites.

### Why
- UIs want to depend on a minimal runner interface, not a concrete Session type.
- Persistence needs stable correlation IDs. Passing `runID` is sufficient, and persisters can also consult `t.RunID`/`t.ID`.
- We do not want separate persistence interfaces for snapshots vs finals; that complexity belongs in the caller/implementation.

### What worked
- The design doc now has a crisp API boundary: `Runner` + `Session` implements it.

### What didn't work
- N/A

### What I learned
- Passing `runID` into persistence is the right level of explicitness; `turnID` is already part of the turn snapshot.

### What was tricky to build
- Keeping the interface minimal while still covering webchat needs without reintroducing lifecycle coupling.

### What warrants a second pair of eyes
- Confirm the persister signature is sufficient for the existing DB persistence code paths (which currently use a turn index) and decide where that indexing should live.

### What should be done in the future
- If turn-indexing remains required by a DB schema, make it an internal detail of the DB persister implementation.

### Code review instructions
- Review the updated design doc:
  - `geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/design-doc/03-inferencestate-enginebuilder-core-architecture.md`
- Review the force-upload helper:
  - `geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/scripts/upload-doc-03-inferencestate-enginebuilder-core-architecture.sh`
