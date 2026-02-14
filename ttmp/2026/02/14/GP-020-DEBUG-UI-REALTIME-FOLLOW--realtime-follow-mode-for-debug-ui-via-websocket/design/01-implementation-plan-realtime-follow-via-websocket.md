---
Title: 'Implementation Plan: Realtime Follow via WebSocket'
Ticket: GP-020-DEBUG-UI-REALTIME-FOLLOW
Status: active
Topics:
    - frontend
    - debugging
    - websocket
    - webchat
    - timeline
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/components/SessionList.tsx
      Note: Add attach/follow controls based on ws_connections
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/store/store.ts
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/store/uiSlice.ts
      Note: Add follow-mode state and status reducers
    - Path: pinocchio/cmd/web-chat/web/src/sem/registry.ts
      Note: Reference SEM decode handlers and timeline upserts
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Reference websocket lifecycle for connect/hydrate/replay
    - Path: pinocchio/pkg/webchat/router_debug_routes.go
      Note: Bootstrap source via /api/debug/timeline and turns endpoints
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Persisted projection source and upsert ordering
    - Path: pinocchio/pkg/webchat/timeline_upsert.go
      Note: Server emits websocket timeline.upsert frames after persistence
ExternalSources: []
Summary: Implementation plan to add read-only websocket follow mode in debug UI when selecting a socket-backed live conversation.
LastUpdated: 2026-02-14T12:12:00-05:00
WhatFor: Plan realtime conversation following from debug UI with socket attach semantics.
WhenToUse: Use when implementing socket selection and live stream follow in debug-ui runtime.
---



# Implementation Plan: Realtime Follow via WebSocket

## Goal
Enable debug UI to attach read-only to a live conversation over websocket and continuously follow persisted timeline projection updates (`timeline.upsert`) while another tab is producing activity.

## Intended UX
1. User selects a conversation with active sockets in sidebar.
2. User enables "Follow live" mode for that conversation/session.
3. UI bootstraps timeline entities from debug API, then applies incoming websocket `timeline.upsert` frames incrementally.
4. User can pause follow mode and inspect current snapshot without mutating backend state.

## Current State (Code Facts)
1. Debug UI uses RTK Query polling-style HTTP endpoints (`/api/debug/*`) only.
2. Existing websocket manager lives in non-debug `webchat` (`src/ws/wsManager.ts`) and writes to `store/timelineSlice` in the webchat store, not debug-ui store.
3. Server already broadcasts persisted projection updates as websocket SEM frames (`event.type = "timeline.upsert"`) from `timeline_upsert.go`.
4. `ConversationSummary` already includes `ws_connections`, so sidebar has enough signal to present live-attach affordance.
5. Full turn/block snapshots are persisted to turn storage and exposed via debug HTTP routes (`/api/debug/turns`, `/api/debug/turn/...`), not streamed as full snapshots over websocket today.

## Architecture Decision
Use a debug-ui scoped websocket client that listens for `timeline.upsert` and upserts generic timeline entities into debug-ui state. Keep debug UI read-only: no command sending, no mutation messages.

## Implementation Guide

### Phase 1: Add follow-mode state and connection target
1. Extend debug-ui UI state with:
   - `followLive: boolean`
   - `followStatus: 'idle' | 'connecting' | 'bootstrapping' | 'connected' | 'error' | 'closed'`
   - `selectedSocketConvId` (usually same as selected conversation)
2. Add actions/selectors in `uiSlice.ts` for toggling follow and status updates.
3. Persist follow preference in URL/search params only if needed (`follow=1`) to keep sharing links deterministic.

### Phase 2: Create debug websocket adapter (timeline upsert only)
1. Add `src/debug-ui/ws/` with:
   - `debugTimelineWsManager.ts` (connect/disconnect/bootstrap/buffer/flush)
2. Reuse timeline proto decode/mapping for `TimelineUpsertV1` only.
3. Bootstrap flow:
   - initial GET (`/api/debug/timeline?conv_id=...`)
   - record high-water version from snapshot
   - replay buffered websocket envelopes in sequence order, dropping frames at or below high-water
4. On conversation switch, ensure hard disconnect then reconnect for new conv id.

### Phase 3: Wire UI controls
1. Add follow controls in `SessionList` row actions for conversations with `ws_connections > 0`.
2. Add follow status badge in app header and timeline page.
3. Disable follow toggle when no socket-backed conversation is selected.
4. Add explicit pause/resume and reconnect actions.

### Phase 4: Merge live updates into existing debug data model
1. Add reducers/selectors to upsert live timeline entities into debug-ui state.
2. Resolve cache strategy:
   - Option A: maintain separate `liveOverlay` slice and merge at selector layer.
   - Option B: patch RTK Query caches via `api.util.updateQueryData`.
3. Use monotonic keys (timeline version/entity id) to avoid duplicates.

### Phase 5: Read-only guarantees and error handling
1. No outbound websocket messages from debug UI.
2. Drop unknown SEM event types safely, keep them visible in raw inspector.
3. Add retry/backoff for socket reconnect with explicit user override.
4. On parse/decode errors, surface to follow status panel but keep existing snapshot rendered.

### Phase 6: Testing
1. Unit tests for debug websocket manager:
   - connect/disconnect lifecycle
   - bootstrap-before-replay ordering
   - duplicate suppression
2. Integration tests with mocked websocket envelopes:
   - follow mode updates timeline pane in near real-time
   - pause mode stops applying incoming frames
3. Manual test:
   - open live conversation in tab A
   - open debug UI in tab B
   - attach follow mode and verify updates without refresh

## Detailed Task Breakdown
1. Add follow-mode fields/actions/selectors in debug-ui state.
2. Implement `debugWsManager` with conversation-scoped lifecycle.
3. Build `timeline.upsert` decode/upsert path for generic timeline entities.
4. Add session list and header follow controls.
5. Integrate bootstrap + live merge strategy with deterministic dedupe.
6. Add reconnect/backoff and status presentation.
7. Add tests (unit + integration) for follow lifecycle.
8. Validate with two-tab manual scenario and document expected behavior.

## Acceptance Criteria
1. Selecting a socket-backed conversation and enabling follow connects websocket successfully.
2. New projected timeline entities appear in debug UI without page reload.
3. Pause stops live application while retaining current state.
4. Reconnect and conversation switching do not leak stale updates.
5. Read-only behavior is preserved (no side-effect commands sent).

## Risks and Mitigations
1. Risk: Drift between webchat ws manager and debug ws manager.
   Mitigation: extract shared low-level helpers after first stable implementation.
2. Risk: Duplicate events due to hydrate + stream overlap.
   Mitigation: dedupe by timeline version/entity key before state write.
3. Risk: Overwriting inspection context while user navigates detail views.
   Mitigation: pause option plus per-panel sticky selection IDs.

## Out of Scope
1. Bi-directional debugging controls (pause backend, inject events).
2. Historical replay scrubbing over websocket.
3. Streaming full turn/block snapshots over websocket (debug-only or otherwise).
4. Debug-specific projector/hydration semantics beyond generic timeline upserts.
