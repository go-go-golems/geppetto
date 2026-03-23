---
Title: Investigation and migration guide for tight webchat service extraction
Ticket: PI-021-WEBCHAT-SERVICE-EXTRACTION
Status: active
Topics:
    - webchat
    - pinocchio
    - backend
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/webchat/chat_service.go
      Note: Current thin wrapper under review
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation_service.go
      Note: Concrete implementation behind current chat wrapper
    - Path: ../../../../../../../pinocchio/pkg/webchat/http/api.go
      Note: Canonical transport contract for chat
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: Router construction and current wrapper instantiation
    - Path: ../../../../../../../pinocchio/pkg/webchat/server.go
      Note: Server construction APIs including NewFromRouter
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-06T09:32:46.44097265-05:00
WhatFor: ""
WhenToUse: ""
---


# Investigation and migration guide for tight webchat service extraction

## Executive Summary

`pinocchio/pkg/webchat` currently mixes three concerns that are easier to understand separately: conversation execution, HTTP transport handlers, and optional router/UI embedding helpers. The current exported surface suggests a deeper split than the implementation actually provides, which makes the package harder to migrate into the tighter shared chat-service shape discussed in the OS-level design work.

This ticket proposes an incremental cleanup. The first priority is to simplify the implementation without changing the active `/chat`, `/ws`, and `/api/timeline` contract. The initial code slice should collapse the redundant `ChatService` wrapper into a zero-cost compatibility surface over `ConversationService`, then continue by removing dead construction seams and alias-only subpackages.

## Problem Statement

The package is carrying legacy surface area that obscures the real architecture.

Observed examples:

- `ChatService` is presented as a separate service layer, but it currently just forwards into `ConversationService`.
- `Server` and the package docs say that applications own `/chat` and `/ws`, but the package still exposes router convenience APIs and embedded static UI helpers as if they were part of the preferred composition path.
- Alias-only packages under `pkg/webchat/{chat,stream,timeline,bootstrap}` expand the public surface without adding behavior.
- `NewFromRouter` preserves an older construction seam even though the newer docs point users to `NewServer(...)` plus explicit handler constructors.

The practical effect is that a new engineer cannot easily tell:

- which APIs are canonical,
- which ones are compatibility leftovers,
- and which pieces should survive the longer-term extraction into a smaller shared chat backend.

## Current State

### Canonical active integration path

The current package and downstream app wiring indicate the canonical shape is:

```text
runtime builder + request resolver
              |
              v
        webchat.NewServer(...)
              |
              v
      app-owned handler mounting
        - POST /chat          -> webhttp.NewChatHandler(...)
        - GET  /ws            -> webhttp.NewWSHandler(...)
        - GET  /api/timeline  -> webhttp.NewTimelineHandler(...)
```

That path matters because it clarifies the real boundaries:

- `ConversationService` materializes or reuses conversations and submits prompts.
- `StreamHub` owns websocket attachment and stream lifecycle.
- `webchat/http` owns request/response transport contracts.
- apps own route registration and any higher-level feature APIs.

### Active versus legacy surface

Active and still structurally important:

- `pinocchio/pkg/webchat/conversation_service.go`
- `pinocchio/pkg/webchat/stream_hub.go`
- `pinocchio/pkg/webchat/http/api.go`
- `pinocchio/pkg/webchat/router_timeline_api.go`
- `pinocchio/pkg/webchat/http/profile_api.go` for integrations that truly use profile APIs

Likely legacy, transitional, or overextended:

- `pinocchio/pkg/webchat/chat_service.go`
- `pinocchio/pkg/webchat/server.go:NewFromRouter`
- `pinocchio/pkg/webchat/chat/api.go`
- `pinocchio/pkg/webchat/stream/api.go`
- `pinocchio/pkg/webchat/timeline/api.go`
- `pinocchio/pkg/webchat/bootstrap/api.go`
- `Router.Mount`, `Router.Handle`, `Router.HandleFunc`, and `Router.Handler` as non-canonical utility mux helpers

### Why static UI is not part of the extracted target

The long-term shared backend service should not depend on serving embedded UI. `cmd/web-chat` may continue to ship a bundled demo frontend, but the extracted backend service used by OS apps should focus on transport and runtime composition only.

That means the shared target should prioritize:

- chat submission
- websocket streaming
- timeline hydration
- optional profile listing if the frontend needs a dropdown

and should not require:

- `staticFS`
- `/`
- `/assets/*`
- a bundled frontend owned by the shared Go package

## Proposed Solution

Migrate `webchat` toward a smaller, more honest surface by removing redundant abstraction layers before introducing any new packaging.

### Target shape

The target shared backend shape is:

```text
ChatService / ConversationService
  - SubmitPrompt(...)
  - ResolveAndEnsureConversation(...)  // if still needed on the chat side

StreamHub
  - ResolveAndEnsureConversation(...)
  - AttachWebSocket(...)

TimelineService
  - Snapshot(...)

webhttp
  - NewChatHandler(...)
  - NewWSHandler(...)
  - NewTimelineHandler(...)
```

The migration should preserve behavior but reduce indirection.

### Phase 1: collapse the `ChatService` wrapper

The safest first change is not a rename campaign. It is to make `ChatService` a zero-cost compatibility layer over `ConversationService` and remove the forwarding wrapper behavior.

Desired effect:

- no behavior change,
- fewer places to keep in sync,
- clearer alignment between the type name and the actual implementation.

### Phase 2: remove obsolete construction seams

Remove `NewFromRouter` after verifying there are no in-repo callers and update any docs that still mention the older `NewRouter + NewFromRouter` setup.

### Phase 3: delete alias-only subpackages

Remove or strongly deprecate:

- `webchat/chat`
- `webchat/stream`
- `webchat/timeline`
- `webchat/bootstrap`

### Phase 4: tighten router utility surface

Decide whether `Mount`, `Handle`, `HandleFunc`, and `Handler` should survive as exported API or move out of the shared path entirely.

### Phase 5: keep the shared target independent of static UI

For the eventual extracted chat-service package, static UI must be optional or absent. App shells should own UI composition.

## Design Decisions

### Decision: use behavior-preserving slices

Rationale:

- `pinocchio` already has active downstream consumers.
- smaller diffs are easier to validate and easier to explain in the diary.
- it keeps cleanup work from being confused with semantics changes.

### Decision: treat `webchat/http` as canonical

Rationale:

- it expresses the active transport contract used by real integrations.
- it already isolates the important seams:
  - request resolution
  - prompt submission
  - websocket attach
  - timeline snapshot reads

### Decision: remove accidental layers before inventing new ones

Rationale:

- the package already has too many overlapping names (`server`, `router`, `chat service`, `conversation service`, `bootstrap`).
- adding another abstraction before trimming the current surface would worsen the confusion.

### Decision: separate “demo app” concerns from “shared backend” concerns

Rationale:

- `cmd/web-chat` can remain a richer example app.
- the shared backend target should stay small and app-composable.

## Alternatives Considered

### Rewrite everything into a new package immediately

Rejected for now.

Reason:

- too much movement at once,
- harder review,
- more migration risk for downstream callers.

### Keep the current exported surface and only add docs

Rejected.

Reason:

- the problem is not just missing documentation; it is real surface-area sprawl.

### Remove all legacy APIs in one commit

Rejected for now.

Reason:

- not safe enough without staged validation and diary-backed checkpoints.

## Implementation Plan

### Migration backlog

1. Finish the baseline investigation and record the cleanup sequence in this ticket.
2. Collapse `ChatService` into a zero-cost compatibility layer over `ConversationService`.
3. Validate the change with focused `pkg/webchat` and `cmd/web-chat` tests.
4. Record the step in the diary and changelog, then commit the code.
5. Remove `NewFromRouter`.
6. Validate and commit that removal.
7. Remove alias-only subpackages.
8. Reassess router utility methods and static UI helpers.

### Detailed task sequencing

#### Task 1: collapse the wrapper layer

Primary files:

- `pinocchio/pkg/webchat/chat_service.go`
- `pinocchio/pkg/webchat/conversation_service.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/server.go`
- `pinocchio/pkg/webchat/types.go`
- `pinocchio/pkg/webchat/router_options.go`

Pseudocode:

```go
// before
svc := NewConversationService(...)
r.chatService = NewChatServiceFromConversation(svc)

// after
svc := NewConversationService(...)
r.chatService = svc

// compatibility surface
type ChatService = ConversationService

func NewChatService(cfg ChatServiceConfig) (*ConversationService, error) {
    return NewConversationService(cfg)
}

func NewChatServiceFromConversation(svc *ConversationService) *ConversationService {
    return svc
}
```

Validation:

- `go test ./pkg/webchat/...`
- targeted tests for `cmd/web-chat` if needed

#### Task 2: remove `NewFromRouter`

Primary files:

- `pinocchio/pkg/webchat/server.go`
- `pinocchio/pkg/doc/topics/webchat-http-chat-setup.md`
- any tests or docs still mentioning `NewFromRouter`

Validation:

- repo-wide grep for `NewFromRouter(`
- targeted `go test`

#### Task 3: remove alias-only subpackages

Primary files:

- `pinocchio/pkg/webchat/chat/api.go`
- `pinocchio/pkg/webchat/stream/api.go`
- `pinocchio/pkg/webchat/timeline/api.go`
- `pinocchio/pkg/webchat/bootstrap/api.go`

Validation:

- repo-wide import grep
- `go test ./...` in `pinocchio` if feasible

#### Task 4: tighten the router surface

Questions to answer:

- does anything outside tests still use `Mount`, `Handle`, `HandleFunc`, or `Handler`?
- should those helpers survive only as demo-app utilities?

Possible end state:

```go
srv.ChatService()
srv.StreamHub()
srv.TimelineService()
```

with no implied mux ownership in the shared path.

### Review checklist per step

- Does the change reduce indirection or surface area?
- Does it preserve `/chat`, `/ws`, and `/api/timeline` behavior?
- Does it avoid introducing new compatibility layers?
- Are the diary and changelog updated in the same implementation slice?

## Open Questions

### External consumer risk

We have strong in-repo evidence that some exported symbols are dead, but we do not yet have a full map of external consumers outside this workspace. Some deletions may need a short deprecation window if `pinocchio` is imported elsewhere.

### Final naming

There is still a naming decision ahead:

- keep `ConversationService` as the main concrete type and use `ChatService` as a compatibility label, or
- eventually rename more aggressively so the package presents one obvious term.

The recommended first steps do not require settling that question yet.

### Static UI inside `pinocchio`

The long-term shared extraction target should not depend on embedded UI. The remaining question is whether static UI helpers remain only for `cmd/web-chat` or are removed from `pkg/webchat` entirely in a later cleanup.

## References

- `pinocchio/pkg/webchat/chat_service.go`
- `pinocchio/pkg/webchat/conversation_service.go`
- `pinocchio/pkg/webchat/http/api.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/server.go`
- `pinocchio/pkg/doc/topics/webchat-http-chat-setup.md`
