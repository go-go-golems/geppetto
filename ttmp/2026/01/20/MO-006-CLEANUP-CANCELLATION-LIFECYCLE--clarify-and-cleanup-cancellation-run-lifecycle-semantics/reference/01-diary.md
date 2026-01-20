---
Title: Diary
Ticket: MO-006-CLEANUP-CANCELLATION-LIFECYCLE
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
  - Path: geppetto/pkg/inference/state/state.go
    Note: Current StartRun/FinishRun/CancelRun/HasCancel implementation
  - Path: geppetto/pkg/inference/core/session.go
    Note: Runner lifecycle split (RunInference vs RunInferenceStarted)
  - Path: pinocchio/pkg/ui/backend.go
    Note: TUI start/cancel/finish pattern (Bubble Tea)
  - Path: pinocchio/pkg/webchat/router.go
    Note: Webchat start/cancel/finish pattern (HTTP + goroutine)
  - Path: moments/backend/pkg/webchat/conversation.go
    Note: Moments stores running+cancel on Conversation (different shape)
  - Path: moments/backend/pkg/webchat/router.go
    Note: Moments run loop lifecycle and cancel handling
  - Path: go-go-mento/go/pkg/webchat/conversation.go
    Note: go-go-mento conversation embeds its own inference state
  - Path: go-go-mento/go/pkg/webchat/router.go
    Note: go-go-mento run loop cancels and emits interrupt on context.Canceled
ExternalSources: []
Summary: "Implementation diary for MO-006: clarify and propose cleanup for cancellation/run lifecycle semantics."
LastUpdated: 2026-01-20T21:50:00-05:00
WhatFor: "Track investigation + documentation steps for cleaning up StartRun/FinishRun/CancelRun semantics."
WhenToUse: "Update after each meaningful investigation/design change and each commit."
---

# Diary

## Goal

Clarify and simplify cancellation/run lifecycle semantics across geppetto/pinocchio/moments/go-go-mento by aligning implementation terminology with the product reality: **Conversation is state; Inference is the cancelable computation.**

## Step 1: Create MO-006 ticket workspace, tasks, and docs

This step created a dedicated ticket workspace so the lifecycle cleanup discussion is not spread across other refactor tickets. The goal is to be able to point reviewers to a single “textbook” analysis before we start changing APIs.

**Commit (code):** N/A

### What I did
- Created ticket `MO-006-CLEANUP-CANCELLATION-LIFECYCLE`.
- Added an analysis doc and a diary doc.
- Added initial tasks (document current behavior; propose cleaned model; add cancel tests).

### Why
- The current naming (`StartRun/FinishRun/CancelRun`) is ambiguous and risks bugs (stuck “running” state, missing interrupt signals).

### What worked
- Ticket scaffold created successfully.

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
- Review ticket scaffold under:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/`

### Technical details
- Commands:
  - `docmgr ticket create-ticket --ticket MO-006-CLEANUP-CANCELLATION-LIFECYCLE ...`
  - `docmgr doc add --ticket MO-006-CLEANUP-CANCELLATION-LIFECYCLE ...`
  - `docmgr task add --ticket MO-006-CLEANUP-CANCELLATION-LIFECYCLE ...`

## Step 2: Audit lifecycle + cancellation behavior and write the “Run vs Conversation vs Inference” analysis

This step gathered the actual call sites for `StartRun/FinishRun/CancelRun/HasCancel` across the main runners (pinocchio TUI + webchat), compared them with moments and go-go-mento’s webchat structures, and then wrote a detailed analysis that proposes a clearer abstraction boundary: conversation state vs inference execution.

The key conceptual correction is that *a conversation is not something you cancel*; you cancel an in-flight inference (which might be a single provider call or a tool loop). Once that boundary is explicit, the rest of the lifecycle API becomes much simpler to reason about.

**Commit (code):** N/A

### What I did
- Located lifecycle/cancellation call sites across code:
  - `geppetto/pkg/inference/state/state.go`
  - `geppetto/pkg/inference/core/session.go`
  - `pinocchio/pkg/ui/backend.go`
  - `pinocchio/pkg/webchat/router.go`
  - `moments/backend/pkg/webchat/*`
  - `go-go-mento/go/pkg/webchat/*`
- Wrote the analysis doc:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/analysis/01-run-vs-conversation-vs-inference-lifecycle-cancellation-and-ownership.md`

### Why
- The current code stores both conversation state and in-flight inference lifecycle fields in one struct, which makes “Run” ambiguous and encourages incorrect mental models.
- Web UIs need a reliable terminal signal on cancellation; otherwise they can remain stuck in “generating”.

### What worked
- The audit found consistent patterns:
  - TUI/webchat “claim run first” (StartRun) before async execution.
  - Cancellation is always a context cancel (stored somewhere reachable).

### What didn't work
- N/A

### What I learned
- Moments and go-go-mento already treat cancellation as “cancel the current run loop’s context,” but naming still conflates “run” with “conversation”.

### What was tricky to build
- Separating “chat run” semantics from go-go-mento’s non-chat “runs” (task/indexing runs), which use CancelRun in a different domain.

### What warrants a second pair of eyes
- Confirm the proposed API direction (ConversationState + structured StartInference handle) is compatible with Bubble Tea and HTTP handlers without reintroducing races.

### What should be done in the future
- Prototype a `StartInference(ctx) (runCtx, finish, err)` style API and migrate pinocchio TUI to it first.

### Code review instructions
- Start with the analysis doc:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/analysis/01-run-vs-conversation-vs-inference-lifecycle-cancellation-and-ownership.md`

### Technical details
- Search command used:
  - `rg -n "StartRun\\(|FinishRun\\(|CancelRun\\(|HasCancel\\(" -S geppetto pinocchio moments go-go-mento --glob '!**/ttmp/**' --glob '!**/pkg/doc/**'`
