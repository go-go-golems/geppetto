---
Title: Diary
Ticket: GP-020-DEBUG-UI-REALTIME-FOLLOW
Status: active
Topics:
    - frontend
    - debugging
    - websocket
    - webchat
    - timeline
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts
      Note: Exploration source for debug-ui data model and conversation metadata
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/components/SessionList.tsx
    - Path: pinocchio/cmd/web-chat/web/src/sem/registry.ts
      Note: Exploration source for SEM event type support
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Exploration source for websocket ordering and buffering semantics
    - Path: pinocchio/pkg/webchat/router.go
      Note: Snapshot hook persists turns but does not websocket-stream full turn snapshots
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Current websocket semantic event classes
    - Path: pinocchio/pkg/webchat/timeline_upsert.go
      Note: Evidence for timeline.upsert websocket emission
    - Path: pinocchio/pkg/webchat/turn_persister.go
      Note: Turn persistence path used for stored snapshots
ExternalSources: []
Summary: Exploration diary for realtime websocket follow planning in debug UI.
LastUpdated: 2026-02-14T12:12:00-05:00
WhatFor: Record how realtime follow architecture and tasks were derived.
WhenToUse: Use when implementing GP-020 or reviewing assumptions around live attach behavior.
---



# Diary

## Goal
Record the exploration process used to define the realtime websocket follow ticket and its implementation sequencing.

## Step 1: Trace websocket and SEM plumbing for reuse in debug UI
I focused on finding existing websocket and protobuf decoding logic that can be reused instead of re-invented. The key constraint was a read-only debug UI that can follow live activity from another tab.

The output of this step is a concrete architecture choice: a debug-ui-specific websocket manager reusing SEM decoding/mapping patterns, with conversation-scoped connect/hydrate/replay lifecycle.

### Prompt Context
**User prompt (verbatim):** "Create a new ticket to integrate it in the UI, and add a detailed implementation guide to the ticket. 

Also make a list of other functionality that might not be integrated in the UI.

Also make a ticket + implementation plan to add realtime websocket connection to the debug UI when selecting a socket, to be able to \"follow\" realtime conversations (happening in another tab).

Upload both implementation plans to the remarkable tablet. Keep a diary while exploring /"

**Assistant interpretation:** Create a second ticket specifically for websocket follow mode, including a detailed implementation plan and diary-backed reasoning.

**Inferred user intent:** Prepare implementation-ready design work so live-follow capabilities can be added without ambiguity and without coupling to legacy behavior.

**Commit (code):** N/A (documentation-only step)

### What I did
- Inspected `src/ws/wsManager.ts` lifecycle (connect, hydrate, buffer, replay, disconnect).
- Inspected `src/sem/registry.ts` for existing SEM event decode and timeline upsert behaviors.
- Inspected debug-ui API layer (`src/debug-ui/api/debugApi.ts`) and conversation summary shape (`ws_connections`).
- Created ticket workspace `GP-020-DEBUG-UI-REALTIME-FOLLOW` and authored detailed implementation plan.

### Why
- Reusing existing websocket and SEM primitives reduces integration risk and implementation time.
- Needed to preserve read-only debug semantics while still applying live updates.

### What worked
- Existing `wsManager` provided a clear hydration-before-replay ordering model.
- Existing SEM registry confirmed support for timeline widgets/custom kinds through proto-backed mapping.

### What didn't work
- N/A during this exploration step (no runtime implementation attempted yet).

### What I learned
- Debug UI has enough conversation metadata today to expose follow affordances (`ws_connections`).
- The main missing piece is not backend capability but debug-ui store lifecycle wiring for live streams.

### What was tricky to build
- The tricky part is merge semantics: combining hydrated snapshots and live stream updates without duplicates or stale overwrites.
- This is why the plan explicitly includes monotonic dedupe keys and a pause mode.

### What warrants a second pair of eyes
- Choice between RTK Query cache patching vs dedicated live overlay slice.
- Whether follow mode URL persistence should be enabled by default.

### What should be done in the future
1. Implement debug websocket adapter first, then wire controls.
2. Add two-tab manual validation workflow to avoid regressions in attach/detach behavior.

### Code review instructions
- Start with websocket lifecycle in `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`.
- Review SEM decode handlers in `pinocchio/cmd/web-chat/web/src/sem/registry.ts`.
- Compare with debug-ui API and store boundaries to validate proposed adapter seam.

### Technical details
- Key design requirement: hydrate once via HTTP, then replay buffered websocket frames by sequence order.
- Read-only guarantee: debug UI sends no control frames/messages; it only subscribes and renders.

## Step 2: Narrow follow scope to generic timeline upserts only
I validated the backend emit path to check whether debug UI can rely directly on projected entities over websocket. The result is that it can: `timeline.upsert` is already emitted after persistence, which matches your desired model.

I also verified the turn/block question: full turn snapshots are not currently streamed over websocket. They are persisted and queried via debug HTTP endpoints, so turn/block websocket streaming should be a later ticket.

### Prompt Context
**User prompt (verbatim):** "yeah, update the document. for now we just want generic timeline entities and upsert them. We'll do debug specific projection and hydration later on. Do we also receive the turns/blocks over websocket (maybe in a debug only mode)?"

**Assistant interpretation:** Update the GP-020 plan to focus on generic timeline upserts only, and confirm whether websocket currently carries full turns/blocks.

**Inferred user intent:** Keep current implementation scope minimal and aligned with what backend already provides, deferring heavier debug-specific streaming designs.

**Commit (code):** N/A (documentation-only step)

### What I did
- Verified websocket projection emit path in `pinocchio/pkg/webchat/timeline_upsert.go`.
- Verified projector persistence ordering in `pinocchio/pkg/webchat/timeline_projector.go`.
- Verified current websocket SEM translator event types in `pinocchio/pkg/webchat/sem_translator.go`.
- Verified turn snapshot persistence path in `pinocchio/pkg/webchat/router.go` (`snapshotHookForConv`) and `pinocchio/pkg/webchat/turn_persister.go`.
- Updated GP-020 design doc and tasks to use bootstrap + `timeline.upsert` follow only.

### Why
- Needed to avoid adding scope for events/turns websocket handling when projected timeline upserts already satisfy the immediate requirement.

### What worked
- Backend architecture cleanly supports this scope: persisted projection + websocket upsert broadcast.

### What didn't work
- Initial file probes used wrong filenames (`snapshots.go`, `turn_store_persister.go`) and failed with `No such file or directory`.
- Corrected to `router.go` snapshot hook and `turn_persister.go`.

### What I learned
- `timeline.upsert` is the right primary stream for debug follow mode right now.
- Full turns/blocks are currently API-backed (`/api/debug/turns`, `/api/debug/turn/...`), not websocket-backed snapshots.

### What was tricky to build
- Distinguishing "semantic event stream includes llm/tool updates" from "full persisted turn snapshots are streamed" required checking both emitters and persistence hooks.

### What warrants a second pair of eyes
- Whether to eventually add a debug-only websocket event type for turn snapshots (`turn.snapshot`) or keep turns strictly HTTP-sourced.

### What should be done in the future
1. Add a follow-up ticket if we want debug-only websocket turn/block streaming.
2. Keep the immediate implementation strictly generic timeline upsert to reduce risk.

### Code review instructions
- Start at `pinocchio/pkg/webchat/timeline_upsert.go` and `pinocchio/pkg/webchat/timeline_projector.go`.
- Confirm non-streaming of full turns in `pinocchio/pkg/webchat/router.go` and `pinocchio/pkg/webchat/turn_persister.go`.
- Review plan/task updates in GP-020 design/tasks docs.

### Technical details
- Upsert emission flow is persist-first then broadcast (`store.Upsert` then `onUpsert` -> `conv.pool.Broadcast`).
- Websocket payloads include `timeline.upsert`, plus SEM llm/tool/log-style frames, but not full turn YAML/blocks snapshots.
