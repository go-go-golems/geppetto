---
Title: 'Frontend: Redux Mutations + Inference Trigger Points'
Ticket: MO-002-MOMENTS-LLM-INFERENCE-TRIGGER
Status: active
Topics:
    - moments
    - backend
    - frontend
    - llm
    - inference
    - geppetto
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: moments/web/src/features/document-finding/DocumentFindingPage.tsx
      Note: Auto-starts inference to find transcripts/documents via startWithHiddenMessage
    - Path: moments/web/src/features/drive1on1/SummaryPage.tsx
      Note: Auto-starts summary inference via startWithHiddenMessage; chooses summary profile
    - Path: moments/web/src/features/team-select/TeamSelectPage.tsx
      Note: Auto-starts team inference via startWithHiddenMessage and sends follow-up instructions
    - Path: moments/web/src/platform/api/chatApi.ts
      Note: Defines startChat/startChatWithProfile mutations that hit /rpc/v1/chat*
    - Path: moments/web/src/platform/chat/hooks/useChatStream.ts
      Note: WebSocket consumer for /rpc/v1/chat/ws; streams inference output into timeline
    - Path: moments/web/src/platform/chat/hooks/useSidebarChat.ts
      Note: Feature-level chat hook; sendMessage/startWithHiddenMessage both enqueue inference
    - Path: moments/web/src/platform/chat/state/chatQueueSlice.ts
      Note: Central inference trigger; dispatches RTK Query startChat/startChatWithProfile and handles 409 retry
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-21T12:54:51.648898611-05:00
WhatFor: ""
WhenToUse: ""
---


# Frontend: Redux Mutations + Inference Trigger Points

## Scope

This document is a frontend-side companion to the backend report:

- `analysis/01-backend-endpoints-that-trigger-geppetto-inference.md`

It answers two questions:

1) **“All Redux mutations”** in the Moments frontend (interpreted as):
   - **RTK Query `builder.mutation(...)` endpoints** (network mutations), and
   - **Redux Toolkit slice action creators** (state mutations) that are the primary write-paths.

2) **Every place in the frontend codebase where a Geppetto inference can be triggered**, now that we know the backend trigger endpoints are primarily `POST /rpc/v1/chat*`.

## Key Backend Trigger Endpoints (from backend report)

Geppetto inference is directly triggered by:

- `POST /rpc/v1/chat`
- `POST /rpc/v1/chat/{profile}`

Inference-adjacent/control endpoints:

- `GET /rpc/v1/chat/ws` (streaming attachment; not a direct trigger)
- `POST /rpc/v1/chat/debug/step-mode` (control plane; only if enabled server-side)
- `POST /rpc/v1/chat/debug/continue` (control plane; only if enabled server-side)

## How the Frontend Triggers Inference (high-level)

### Primary send path (queue + RTK Query)

```
UI event (user sends message OR auto-start)
  |
  v
dispatch(enqueueChatMessage(...))                 (Redux slice)
  |
  v
dispatch(processQueue(convId))                   (thunk)
  |
  | chooses one:
  |   - chatApi.endpoints.startChat.initiate(...)             -> POST /rpc/v1/chat
  |   - chatApi.endpoints.startChatWithProfile.initiate(...)  -> POST /rpc/v1/chat/{profile}
  v
backend starts ToolCallingLoop -> Engine.RunInference -> LLM
```

### Streaming path (WebSocket)

```
useChatStream({ convId, profile, draftBundleId, enabled })
  |
  v
ws(s)://<host>/rpc/v1/chat/ws?conv_id=...&profile=...&draft_bundle_id=...&access_token=...
  |
  v
dispatch timelineSlice.addEntity / upsertEntity / appendMessageText ...
```

The WebSocket is crucial for UX, but it does not itself start inference; it receives events produced by backend runs started by `POST /rpc/v1/chat*`.

## A) RTK Query Mutation Inventory (network mutations)

Below is the exhaustive list of `builder.mutation(...)` definitions in `moments/web/src` at time of writing.

### `/api/v1` mutations (`moments/web/src/store/api/apiSlice.ts`)

Base URL: `/api/v1`

- `discoverTeam` → `POST /api/v1/coach/discover-team`
- `startGoogleOAuth` → `POST /api/v1/oauth/state`
- `startSlackOAuth` → `POST /api/v1/oauth/state` (provider=`slack`)
- `startGitHubOAuth` → `POST /api/v1/oauth/state/github`
- `startLinearOAuth` → `POST /api/v1/oauth/state/linear`
- `disconnectGoogle` → `POST /api/v1/google/disconnect`
- `disconnectSlack` → `POST /api/v1/oauth/revoke` (provider=`slack`)
- `revokeProviderOAuth` → `POST /api/v1/oauth/revoke`
- `updatePrompt` → `PUT /api/v1/admin/prompts/:key`
- `resetPrompt` → `POST /api/v1/admin/prompts/:key/reset`
- `assignPersonRole` → `POST /api/v1/admin/rbac/person_roles`
- `unassignPersonRole` → `DELETE /api/v1/admin/rbac/person_roles`

### `/api/v1` mutations injected into `apiSlice`

`moments/web/src/store/api/analyticsApi.ts` (injects into `apiSlice`)

- `trackAnalyticsEvent` → `POST /api/v1/pa/events`

### `/rpc/v1` base mutations (`moments/web/src/store/api/rpcSlice.ts`)

Base URL: `/rpc/v1`

Chat control-plane:
- `chatDebugStepMode` → `POST /rpc/v1/chat/debug/step-mode`
- `chatDebugContinue` → `POST /rpc/v1/chat/debug/continue`

Relationships:
- `upsertRelationshipByEmail` → `POST /rpc/v1/relationships.upsert_by_email`
- `upsertRelationship` → `POST /rpc/v1/relationships.upsert`
- `updateRelationship` → `POST /rpc/v1/relationships.update`
- `deleteRelationship` → `POST /rpc/v1/relationships.delete`

Prompts:
- `createPrompt` → `POST /rpc/v1/prompts.create`
- `updatePrompt` → `POST /rpc/v1/prompts.update`
- `promotePrompt` → `POST /rpc/v1/prompts.promote`
- `archivePrompt` → `POST /rpc/v1/prompts.archive`
- `rollbackPrompt` → `POST /rpc/v1/prompts.rollback`
- `importPrompt` → `POST /rpc/v1/prompts.import`

Profile editor:
- `createBundle` → `POST /rpc/v1/profile_editor.bundles.create`
- `updateBundle` → `POST /rpc/v1/profile_editor.bundles.update`
- `archiveBundle` → `POST /rpc/v1/profile_editor.bundles.archive`
- `upsertEntry` → `POST /rpc/v1/profile_editor.entries.upsert`
- `deleteEntry` → `POST /rpc/v1/profile_editor.entries.delete`
- `publishBundle` → `POST /rpc/v1/profile_editor.bundles.publish`

### `/rpc/v1` injected mutations (feature/platform APIs)

Chat API (`moments/web/src/platform/api/chatApi.ts`, injects into `rpcSlice`)

- `startChat` → `POST /rpc/v1/chat`  ✅ **Direct Geppetto inference trigger**
- `startChatWithProfile` → `POST /rpc/v1/chat/{profile}` ✅ **Direct Geppetto inference trigger**

Person context (`moments/web/src/platform/api/personContextApi.ts`)

- `upsertPersonContext` → `POST /rpc/v1/person_context.upsert`

Memory (`moments/web/src/platform/memory/api/memoryApi.ts`)

- `createMemory` → `POST /rpc/v1/memories.create`
- `searchMemories` → `POST /rpc/v1/memories.search`
- `deleteMemory` → `POST /rpc/v1/memories.delete`
- `recallMemory` → `POST /rpc/v1/memories.recall`
- `approveMemory` → `POST /rpc/v1/memories.approve`

Artifacts (`moments/web/src/features/coach-artifacts/api/artifactApi.ts`)

- `updateArtifact` → `POST /rpc/v1/artifact.update`

Artifact feedback (`moments/web/src/features/coach-artifacts/api/artifactFeedbackApi.ts`)

- `createArtifactFeedback` → `POST /rpc/v1/artifact/feedback.create`

## B) Redux Slice “Mutation” Inventory (state writes)

This is the list of slice action creators (the primary “state mutation” APIs) used across chat and related UX.

### `authSlice` (`moments/web/src/store/auth/authSlice.ts`)

- `setLoading`
- `setSession`
- `setUserProfile`
- `clearSession`
- `expireSession`
- `setError`
- `setAppUserId`

### `timelineSlice` (`moments/web/src/platform/timeline/state/timelineSlice.ts`)

- `setActiveConversation`
- `setConversationDisplayMode`
- `addEntity`
- `addEntities`
- `updateEntity`
- `appendEntityArrayItem`
- `upsertEntity`
- `appendMessageText`
- `finalizeMessage`
- `updateToolProgress`
- `removeEntity`
- `removeEntitiesByKind`
- `clearConversation`
- `clearAll`

### `chatQueueSlice` (`moments/web/src/platform/chat/state/chatQueueSlice.ts`)

- `enqueue`
- `dequeue`
- `setProcessing`
- `bumpRetry`
- `clearQueue`

Also note the exported thunks (not slice reducers, but critical behavior):
- `enqueueChatMessage(...)`
- `enqueueChatMessageAndWait(...)`
- `processQueue(convId)`

### `documentsSlice` (`moments/web/src/platform/documents/state/documentsSlice.ts`)

- `upsertDocuments`
- `setDocuments`
- `clearDocuments`
- `setSelectedDocumentId`
- `selectDocument`
- `deselectDocument`
- `clearSelectedDocuments`

## C) Exhaustive Frontend Inference Trigger Points

This section lists every place the frontend can cause the backend to execute a Geppetto inference, i.e. every code path that results in calling:

- `chatApi.endpoints.startChat...` (→ `POST /rpc/v1/chat*`)

### C1) The central trigger: `chatQueueSlice.processQueue`

File: `moments/web/src/platform/chat/state/chatQueueSlice.ts`

**Trigger mechanics**
- Always sends through a queue to avoid overlapping runs.
- Retries on `409 Conflict` (backend “run in progress”) by rescheduling after 250ms.
- Chooses endpoint based on whether `profile` is non-empty:
  - **Non-empty profile** → `startChatWithProfile` → `POST /rpc/v1/chat/{profile}`
  - **Empty profile** → `startChat` → `POST /rpc/v1/chat` (lets backend preserve conversation profile)

**Pseudocode**

```text
processQueue(convId):
  if conversation busy: retry later
  item := queue[0]
  if item.profile != "":
    call POST /rpc/v1/chat/{profile} with { prompt, conv_id, ... }
  else:
    call POST /rpc/v1/chat with { prompt, conv_id, ... }
  if 409: retry later (don’t drop)
  else if error: drop + emit system error message
  else: dequeue and process next
```

### C2) Direct `enqueueChatMessage(...)` call sites (explicit triggers)

These are code paths that dispatch `enqueueChatMessage` directly.

- `moments/web/src/features/chat/ChatPage.tsx`:
  - `handleSubmit()` dispatches `enqueueChatMessage({ convId, profile, text, stepMode })`.
  - Also toggles chat step mode via `useChatDebugStepModeMutation` (control-plane).

- `moments/web/src/platform/chat/hooks/useSidebarChat.ts`:
  - `sendMessage(text)` dispatches `enqueueChatMessage(...)`.
  - `startWithHiddenMessage(text)` dispatches `enqueueChatMessage(...)` without adding a visible user entity first.
  - Any feature using `useSidebarChat` inherits these trigger points.

- `moments/web/src/platform/timeline/widgets/MultipleChoiceWidget.tsx`:
  - Sends an “I choose: ...” message via `enqueueChatMessage({ profile: '' })` (explicitly omits profile to preserve backend profile).
  - Special case: transcript selection questions do **not** send an LLM message (they navigate directly).

- `moments/web/src/platform/timeline/widgets/RunLimitWidget.tsx`:
  - Calls `enqueueChatMessageAndWait(...)` to send “Please continue…”.
  - This triggers another `POST /rpc/v1/chat/{profileSlug}` run in the same conversation.

### C3) `useSidebarChat(...)` feature entrypoints (indirect triggers)

Any use of `SidebarChat` with `onSendMessage={chat.sendMessage}` is a user-facing trigger point:

- `moments/web/src/features/chat/ChatPageWithSidebar.tsx` → sidebar “Chat Assistant”
- `moments/web/src/features/profile-editor/ProfileDetailPage.tsx` → sidebar “Test Chat”
- `moments/web/src/features/document-finding/DocumentFindingPage.tsx` → sidebar “Document Assistant”
- `moments/web/src/features/drive1on1/SummaryPage.tsx` → sidebar chat on Summary page
- `moments/web/src/features/team-select/TeamSelectPage.tsx` → sidebar “Team Assistant”

### C4) Auto-start (no explicit user action)

These pages automatically call `startWithHiddenMessage(...)` once connected and in an initial/empty state:

- `moments/web/src/features/team-select/TeamSelectPage.tsx`
  - Auto-start: `startWithHiddenMessage('Find my team')` when connected + empty timeline + no existing relationships.

- `moments/web/src/features/document-finding/DocumentFindingPage.tsx`
  - Auto-start: “Find me documents…” when connected + empty timeline + no selected document.
  - Uses profile `find-coaching-transcripts`.

- `moments/web/src/features/drive1on1/SummaryPage.tsx`
  - Auto-start: “Please summarize the following documents: …” when connected + has documents + empty timeline.
  - Selects profile:
    - single doc → `coaching-conversation-summary`
    - multiple docs → `coaching-conversations-summary`

### C5) WebSocket event streaming (inference-adjacent)

WebSocket code does not start inference, but is the main consumer of inference output.

Files:
- `moments/web/src/platform/chat/hooks/useChatStream.ts` (WS implementation)
- `moments/web/src/features/chat/ChatPage.tsx` (uses `useChatStream`)
- `moments/web/src/platform/chat/hooks/useSidebarChat.ts` (uses `useChatStream`)
- `moments/web/src/features/agents/AgentsTestPage.tsx` (uses `useChatStream` for workflow planning events)

Connection URL shape (frontend):

```text
ws(s)://<host>/rpc/v1/chat/ws?
  conv_id=<convId>
  &profile=<profile>            (omitted if default)
  &draft_bundle_id=<uuid>       (optional)
  &access_token=<bearer token>  (optional, provided when available)
```

This matches backend behavior: the WS handler accepts token via query param and maps it to `Authorization: Bearer ...`.

### C6) Chat debug endpoints (control-plane)

The frontend has wiring to call step-mode (and defines “continue”, though it is not referenced by UI code at time of writing):

- `moments/web/src/features/chat/ChatPage.tsx` → `useChatDebugStepModeMutation` (toggle)
- `moments/web/src/features/chat/ChatPageWithSidebar.tsx` → `useChatDebugStepModeMutation` (toggle)

Backend caveat:
- These endpoints only exist when the backend config enables them (`enable-debug-endpoints`); otherwise they will 404.
