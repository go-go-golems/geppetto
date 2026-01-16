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
