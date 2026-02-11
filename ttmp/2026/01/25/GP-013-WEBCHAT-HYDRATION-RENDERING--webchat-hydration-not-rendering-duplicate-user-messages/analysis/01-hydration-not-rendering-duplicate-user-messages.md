---
Title: Hydration not rendering + duplicate user messages
Ticket: GP-013-WEBCHAT-HYDRATION-RENDERING
Status: active
Topics:
    - bug
    - frontend
    - hydration
    - websocket
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx
      Note: Optimistic user echo and hydration triggering
    - Path: pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts
      Note: Snapshot mapping logic
    - Path: pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts
      Note: Selector warning and state shape
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Hydration fetch/parse path and missing isObject
    - Path: pinocchio/pkg/webchat/router.go
      Note: /timeline handler and JSON shape
ExternalSources: []
Summary: Identify why /timeline hydration does not render and why user messages duplicate; propose fixes.
LastUpdated: 2026-01-25T15:54:03-05:00
WhatFor: ""
WhenToUse: ""
---


# Hydration Not Rendering + Duplicate User Messages

## Goal

Explain why `/timeline` hydration does not render on reload and why user messages appear twice, using the provided timeline response and a code-level audit. Provide concrete hypotheses, root-cause candidates, and fixes.

## Context

Symptoms reported in the web UI:
- On reload, hydration shows nothing in the timeline even though `/timeline` returns entities.
- After the first round, user input appears twice (and duplicates also appear when hydrating).
- Console warning: `Selector selectTimelineEntities returned a different result when called with the same parameters`.

Timeline response sample (from the user):

```json
{"convId":"507f7093-fe94-4375-a7ec-a2495f14b24b","version":"8","serverTimeMs":"1769369096065","entities":[{"id":"user-05232b5b-9346-4a2d-8ae6-5040acb63767","kind":"message","createdAtMs":"1769368551779","updatedAtMs":"1769368551779","message":{"schemaVersion":1,"role":"user","content":"hello"}},{"id":"1d192928-c4c4-49ee-8060-416e603e2d08","kind":"message","createdAtMs":"1769368551781","updatedAtMs":"1769368552914","message":{"schemaVersion":1,"role":"assistant","content":"Hello! How can I assist you today?"}},{"id":"user-2e8a3395-07e2-4da0-8d0b-c8995df76339","kind":"message","createdAtMs":"1769368553770","updatedAtMs":"1769368553770","message":{"schemaVersion":1,"role":"user","content":"yo"}},{"id":"d88dcd3e-b4e8-4c9a-aea6-ea35fbe0af64","kind":"message","createdAtMs":"1769368553771","updatedAtMs":"1769368554405","message":{"schemaVersion":1,"role":"assistant","content":"Yo! What's up? How can I help you?"}}]}
```

Observed network behavior (user report): no explicit `/timeline` request visible in the waterfall; WebSocket connects successfully.

## Current Hydration Path (Frontend)

Hydration lives in `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`:
- `connect()` calls `hydrate()` unless `hydrate: false`.
- `hydrate()` calls `GET /timeline?conv_id=...`, parses JSON via `fromJson(TimelineSnapshotV1Schema, ...)`, then `applyTimelineSnapshot()` into the Redux timeline slice.
- The slice is cleared before hydration.

Important details:
- `hydrate()` wraps the JSON parse in a `try/catch` and silently ignores errors.
- There is no logging if hydration fails or if `res.ok` is false.
- `applyTimelineSnapshot()` uses `timelineEntityFromProto()` to map protobuf entities to render entities.

## Immediate Root-Cause Candidate (High Confidence)

### Missing `isObject()` in `wsManager.ts`

`wsManager.ts` calls `isObject(j)` inside `hydrate()`, but there is no local definition or import. This triggers a `ReferenceError`, which is caught and silently ignored:

```ts
if (isObject(j)) {
  const snap = fromJson(...);
  applyTimelineSnapshot(...);
}
```

This means:
- `/timeline` can return valid JSON, but hydration fails before parsing.
- The timeline slice is already cleared, leaving it empty.
- On reload, the UI renders nothing.

This aligns with the report: timeline response exists but nothing renders.

## Secondary Factors (Medium Confidence)

### Duplicate user messages from optimistic echo + canonical entity mismatch

In `ChatWidget.tsx`, user messages are optimistically added with ID `user-${Date.now()}`. The backend emits a canonical user message with ID `user-${turnID}`. These IDs never match, so both messages remain.

With hydration failing, the optimistic echo is never reconciled against canonical entities, leading to a persistent duplicate.

### `version` and `createdAtMs` are `bigint` in proto

`TimelineSnapshotV1.version` and `TimelineEntityV1.createdAtMs` are `uint64` in proto, parsed into `bigint` by bufbuild. `timelineEntityFromProto()` expects numbers and ignores `version` if it is not a JS `number`. This does not cause blank rendering, but it reduces ordering/dedup safety for incremental upserts.

### Selector warning is real but not causal

`selectTimelineEntities` creates a new array on every call, which triggers the redux warning. This can cause extra renders but does not explain the empty timeline.

## Additional Observations: Silent Errors

### Try/catch blocks that swallow errors

There are multiple `try/catch` blocks with empty catches in the web chat frontend; this hides hydration errors and WS parse failures.

- `wsManager.ts`: 3 empty catches (WS message JSON parse, WS close, and `/timeline` hydrate).
- `ChatWidget.tsx`: 4 empty catches (URL parsing, URL rewrite, WS connect best-effort, hydrate best-effort).

These are the primary reason the `ReferenceError` from missing `isObject` is invisible in the console.

### Build-time visibility

The default frontend build (`vite build`) does not run `tsc`, so a missing symbol like `isObject` is not reported at build time. Only `npm run typecheck` (or `pnpm run typecheck`) would surface `Cannot find name 'isObject'`.

## Backend Considerations (Low Confidence)

`/timeline` is served by `pinocchio/pkg/webchat/router.go`. It always returns a JSON snapshot if the timeline store is enabled (in-memory by default now). The response sample demonstrates that the server is emitting valid proto JSON.

Unless `conv_id` is missing or wrong, backend behavior appears correct.

## Why the Timeline Response Should Parse

The sample JSON uses lowerCamel field names (`createdAtMs`, `serverTimeMs`, `message`) consistent with `protojson.Marshal` and `UseProtoNames=false`. `fromJson(TimelineSnapshotV1Schema, ...)` expects that shape and should map oneof fields correctly (message is the field name of the oneof). The parsing should work once the `isObject` bug is fixed.

## Recommended Fixes

### Must-Fix
1) Define or remove `isObject` in `wsManager.ts`.
   - Easiest: inline `if (j && typeof j === 'object')`.
   - Or import `isObject` from `timelineMapper` or a shared utils file.

2) Add hydration error logging (even in dev) to avoid silent failures.
   - Example: log when `res.ok` is false, or when JSON parse fails.

### Should-Fix
3) Reconcile optimistic user messages.
   - Option A (preferred): include a client-generated `client_message_id` in `/chat` body, and use it as timeline entity ID so optimistic and canonical match.
   - Option B: remove optimistic echo and rely on immediate SEM/timeline updates.
   - Option C: add reconciliation logic when a canonical user message arrives (match by role + content + near timestamp; drop optimistic).

4) Normalize proto `bigint` to numbers (or store bigint in Redux) so versioned upserts remain ordered.

5) Memoize `selectTimelineEntities` with `createSelector` to remove Redux warning (not critical to correctness).

## Debug Checklist (If Issue Persists After Fix #1)

- Confirm `/timeline` request is visible in the network log on reload.
- Verify `conv_id` is set in the URL and the same ID is used in WS URL and `/timeline` query.
- Log the parsed snapshot in `hydrate()` to verify mapping.
- Confirm `timelineSlice.order` grows after hydration.
- Verify no runtime errors in `timelineMapper` (e.g., `snapshot` shape) with console logs.

## Files to Inspect

- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx`
- `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go`
