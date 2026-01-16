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
    - Path: pinocchio/pkg/inference/runner/runner.go
      Note: Shared inference runner introduced in Step 1
    - Path: pinocchio/pkg/ui/backend.go
      Note: TUI backend refactor to use shared runner
ExternalSources: []
Summary: Diary for MO-003 implementation steps.
LastUpdated: 2026-01-16T15:12:30-05:00
WhatFor: Track implementation steps for unifying inference between pinocchio TUI and webchat.
WhenToUse: Update after each implementation step or significant discovery.
---



# Diary

## Goal

Track the step-by-step implementation of shared inference orchestration across pinocchio TUI and webchat, with follow-on guidance for migrating Moments later.

## Step 1: Create shared runner and migrate TUI backend

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
