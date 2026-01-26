---
Title: Diary
Ticket: GP-016-WEBCHAT-REDUX-DEVTOOLS
Status: active
Topics:
    - frontend
    - hydration
    - bug
    - websocket
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/01/25/GP-016-WEBCHAT-REDUX-DEVTOOLS--webchat-redux-devtools-debug-hydration-store/analysis/01-pinocchio-webchat-redux-devtools-hydration-debugging.md
      Note: Trace analysis and setup notes
    - Path: geppetto/ttmp/2026/01/25/GP-016-WEBCHAT-REDUX-DEVTOOLS--webchat-redux-devtools-debug-hydration-store/scripts/inspect_actions_for_id.py
      Note: Script to inspect actions for a specific entity id
    - Path: geppetto/ttmp/2026/01/25/GP-016-WEBCHAT-REDUX-DEVTOOLS--webchat-redux-devtools-debug-hydration-store/scripts/inspect_add_entity_actions.py
      Note: Script to list timeline/addEntity actions
    - Path: geppetto/ttmp/2026/01/25/GP-016-WEBCHAT-REDUX-DEVTOOLS--webchat-redux-devtools-debug-hydration-store/scripts/inspect_state_user_entries.py
      Note: Script to list user entities in state.json
    - Path: pinocchio/cmd/web-chat/web/.env.local
      Note: Remote devtools env toggles
    - Path: pinocchio/cmd/web-chat/web/package-lock.json
      Note: NPM lockfile updates
    - Path: pinocchio/cmd/web-chat/web/package.json
      Note: Added devtools dependencies
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx
      Note: Rekey optimistic user message using turn_id from /chat
    - Path: pinocchio/cmd/web-chat/web/src/sem/registry.ts
      Note: Use Date.now for live SEM event timestamps
    - Path: pinocchio/cmd/web-chat/web/src/store/store.ts
      Note: DEV-only remote Redux DevTools wiring
    - Path: pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts
      Note: Added rekey reducer to merge optimistic and server entities
ExternalSources: []
Summary: Diary of Redux DevTools wiring + hydration/duplicate message debugging for pinocchio webchat.
LastUpdated: 2026-01-26T09:22:00-05:00
WhatFor: Track debugging steps, fixes, and validation for the webchat hydration + Redux DevTools work.
WhenToUse: When reviewing what changed for Redux DevTools and user message deduplication fixes.
---



# Diary

## Goal

Capture the work to wire Redux DevTools (remote) for pinocchio webchat and fix the optimistic/user-message duplication seen before hydration, including trace analysis and commands run.

## Step 1: Remote DevTools wiring + optimistic rekey + trace analysis

I wired remote Redux DevTools for the pinocchio webchat frontend and added a DEV-only config path so we can inspect hydration actions without enabling devtools in production. I also added a rekey mechanism to collapse the optimistic user message into the server’s canonical `user-<turn_id>` entity, which removes the duplicate widget seen in pre-hydration traces.

I validated the trace from `/home/manuel/Downloads/state.json`, documented it in the GP-016 analysis doc, and stored the small inspection scripts in the ticket scripts folder. I also ran the tmux setup to keep the Redux DevTools bridge and Vite running together for live inspection.

### Prompt Context

**User prompt (verbatim):** "Cool, i think it's fixed. Write diary, commit"

**Assistant interpretation:** Write a diary entry for the work done and commit the changes.

**Inferred user intent:** Capture the debugging work in a structured doc and finalize it with commits.

**Commit (code):** b6bf48a — "Webchat: enable remote devtools + rekey user messages"

### What I did
- Added `@redux-devtools/remote` and `@redux-devtools/cli` dependencies in the webchat frontend.
- Added DEV-only remote devtools wiring in `store.ts` with `VITE_REMOTE_DEVTOOLS*` env toggles and created `.env.local`.
- Implemented `timeline/rekeyEntity` and used `/chat`’s `turn_id` to rekey optimistic user messages.
- Documented trace findings in the GP-016 analysis doc and created scripts to inspect Redux DevTools exports.
- Ran tmux with the Redux DevTools bridge (`npx redux-devtools`) and Vite (`npm run dev`).

### Why
- Remote DevTools + trace scripts make it easier to reason about hydration and duplicate entities across reloads.
- Rekeying optimistic messages ensures the client uses canonical IDs so WS updates and hydration collapse duplicates.

### What worked
- Devtools bridge + Vite combination provided action/state traces.
- Rekeying via `turn_id` eliminates the optimistic/server duplicate before hydration completes.

### What didn't work
- Initial commit failed due to linting: Biome reported unsorted imports in `src/store/store.ts` after `git commit` triggered `npm run lint`.
- First install of `@redux-devtools/cli` timed out at 10s; reran with a longer timeout.

### What I learned
- The UI `#...` label is the `createdAt` field, which uses WS `seq` live but Unix ms on hydration; that’s why the numbers differ after reload.

### What was tricky to build
- Ensuring optimistic IDs are reconciled with server IDs without losing ordering or props required careful rekeying and merge logic.

### What warrants a second pair of eyes
- The rekey reducer merges props and order updates; confirm it behaves correctly when a canonical entity already exists.
- Ensure the `/chat` response always includes `turn_id`; queued paths may skip it and need a fallback strategy.

### What should be done in the future
- N/A

### Code review instructions
- Start with `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts` and `pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx`.
- Verify `pinocchio/cmd/web-chat/web/src/store/store.ts` for DEV-only devtools wiring.
- Validate by running `npm run check` in `pinocchio/cmd/web-chat/web` and confirming no duplicate user message appears before hydration.

### Technical details
- Commands run:
  - `npm install @redux-devtools/remote`
  - `npm install -D @redux-devtools/cli` (first attempt timed out at 10s)
  - `npx redux-devtools --hostname=localhost --port=8000`
  - `npm run dev`
- Trace scripts stored in: `geppetto/ttmp/2026/01/25/GP-016-WEBCHAT-REDUX-DEVTOOLS--webchat-redux-devtools-debug-hydration-store/scripts/`

## Step 2: Human-readable sent-at timestamps

I switched the card header metadata from the `#<createdAt>` shorthand to a human-readable “sent at” timestamp. To avoid showing a seq number as a timestamp in the live stream, I also adjusted the client-side event timestamp to use `Date.now()` consistently for live SEM events.

This makes the UI consistent across live streaming and hydration, and it aligns with the request to see readable timestamps instead of mixed seq/time values.

### Prompt Context

**User prompt (verbatim):** "sent at timestamp, readable by humans."

**Assistant interpretation:** Update the UI to display human-readable timestamps rather than `#<createdAt>` values.

**Inferred user intent:** Make chat message metadata readable and consistent across live stream and reload.

**Commit (code):** bea2adc — "Webchat: show sent-at timestamps"

### What I did
- Added `fmtSentAt` in `ChatWidget.tsx` and replaced all `#...` card metadata with “sent at” timestamps.
- Updated `createdAtFromEvent` to always use `Date.now()` so live events get real timestamps.

### Why
- `createdAt` mixed seq and epoch millis, which made the UI inconsistent and confusing after reload.

### What worked
- UI now shows a readable time for both live messages and hydrated snapshots.

### What didn't work
- N/A

### What I learned
- The display path was reusing `createdAt` for two different concepts (seq and time).

### What was tricky to build
- Ensuring live-stream timestamps don’t regress ordering or hydration replacement behavior.

### What warrants a second pair of eyes
- Confirm that using `Date.now()` for live SEM events doesn’t break any ordering assumptions.

### What should be done in the future
- N/A

### Code review instructions
- Check `pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx` for `fmtSentAt` usage.
- Check `pinocchio/cmd/web-chat/web/src/sem/registry.ts` for `createdAtFromEvent`.
- Validate in the browser that live messages and reloaded messages display readable “sent at” times.

### Technical details
- No new dependencies or scripts.
