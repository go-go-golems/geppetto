---
Title: Diary
Ticket: PI-021-WEBCHAT-SERVICE-EXTRACTION
Status: active
Topics:
    - webchat
    - pinocchio
    - backend
    - cleanup
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/webchat/chat_service.go
      Note: Implementation diary tracks wrapper collapse and later cleanup
    - Path: ../../../../../../../pinocchio/pkg/webchat/server.go
      Note: Diary will record construction cleanup steps here
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-06T09:32:46.446167578-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture the webchat service migration in small, reviewable steps, including exact commands, validation, failures, and follow-up risks.

## Step 1: Create The Ticket And Define The Migration Sequence

This step created the `geppetto/ttmp` ticket workspace for the webchat service extraction and converted the earlier architecture discussion into a concrete migration plan. The main goal was to turn a conceptual direction into an executable sequence before mutating `pinocchio`.

The core conclusion of this step is that the first code change should be structural rather than semantic: collapse `ChatService` into a zero-cost compatibility layer over `ConversationService`, preserve the current handler contract, and then continue removing obsolete construction seams and alias packages.

### Prompt Context

**User prompt (verbatim):** "Ok, make a ticket with docmgr --root geppetto/ttmp and make a detailed investigation/implementation doc and add tasks.

Then go back to our main topic, the extraction of a chat service.

I like the last tight core documentation, save for needing th eUI. Now write out a detailed set of tasks on how to do the migration, and start working on the tasks one by one, committing appropriately as you go, keeping a detailed implementation diary."

**Assistant interpretation:** Create a `docmgr` ticket in `geppetto/ttmp`, write the investigation and migration documents, seed an explicit task list, and then start the implementation with commit-sized cleanup steps.

**Inferred user intent:** Turn the earlier architecture discussion into an actual execution plan and begin simplifying `pinocchio/pkg/webchat` immediately.

**Commit (code):** N/A

### What I did

- Created ticket `PI-021-WEBCHAT-SERVICE-EXTRACTION` under `geppetto/ttmp`.
- Added the primary design document and the implementation diary.
- Wrote the migration guide with:
  - current-state analysis
  - a tighter target shape
  - phased cleanup tasks
  - per-step validation guidance
- Replaced the placeholder task list with explicit implementation tasks.

### Why

- The migration direction had already been discussed, but it was not yet captured in a single `geppetto` ticket with an executable sequence.
- A local ticket keeps the cleanup program close to the repositories that implement it and makes continuation easier.

### What worked

- `docmgr` supports `--root`, so the ticket could be created under `geppetto/ttmp` without changing the global default docs root.
- The wrapper-collapse step emerged as a clear first implementation slice: it reduces indirection without changing the transport contract.

### What didn't work

- `docmgr status --summary-only` still pointed to the global `openai-app-server/ttmp` root, so every command for this ticket must pass `--root /home/manuel/workspaces/2026-03-02/os-openai-app-server/geppetto/ttmp`.
- The initial grep for `Ticket: PI-021` exited with status `1` because the ticket did not exist yet; that was expected but worth recording as part of the setup trail.

### What I learned

- The existing `geppetto/ttmp` ticket history already uses `PI-*` for Pinocchio/webchat work, so `PI-021` fits the local naming pattern.
- The current webchat cleanup naturally decomposes into:
  - wrapper collapse,
  - construction cleanup,
  - alias-package removal,
  - router-surface reduction.

### What was tricky to build

- The tricky part was distinguishing “small and canonical” from “small and legacy.” `webchat/http/api.go` is compact enough that it can look like leftover glue, but it is actually the active transport contract. The design doc had to make that distinction explicit so later cleanup steps do not delete the wrong layer.

### What warrants a second pair of eyes

- The deletion steps for alias packages and older constructors may affect consumers outside the visible workspace.
- The longer-term treatment of embedded static UI still needs a deliberate choice once the transport cleanup is done.

### What should be done in the future

- Implement the first code step from `tasks.md`: collapse `ChatService` into a zero-cost compatibility layer over `ConversationService`.

### Code review instructions

- Start with the migration guide and verify that the ordering reflects the current codebase accurately.
- Then inspect `tasks.md` and confirm the first code slice is small enough for a focused review.

### Technical details

- Ticket root: `/home/manuel/workspaces/2026-03-02/os-openai-app-server/geppetto/ttmp`
- Ticket id: `PI-021-WEBCHAT-SERVICE-EXTRACTION`
- Planned first code slice:

```go
type ChatService = ConversationService

func NewChatService(cfg ChatServiceConfig) (*ConversationService, error) {
    return NewConversationService(cfg)
}

func NewChatServiceFromConversation(svc *ConversationService) *ConversationService {
    return svc
}
```
