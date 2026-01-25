---
Title: 'Bug analysis: hydration blank + duplicate user messages'
Ticket: GP-012-WEBCHAT-HYDRATION-BUG
Status: active
Topics:
    - hydration
    - webchat
    - frontend
    - backend
    - bug
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx
      Note: Optimistic user message insertion and conv_id URL behavior
    - Path: pinocchio/cmd/web-chat/web/src/sem/registry.ts
      Note: timeline.upsert event handling
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Hydration fetch logic and silent failure handling
    - Path: pinocchio/cmd/web-chat/web/vite.config.ts
      Note: Dev proxy for /timeline
    - Path: pinocchio/pkg/webchat/router.go
      Note: User message upsert id and /timeline handler
    - Path: pinocchio/pkg/webchat/timeline_upsert.go
      Note: WS timeline delta emission
ExternalSources: []
Summary: Root-cause analysis of missing hydration on reload and duplicated user messages after first round; includes likely causes, evidence to gather, and fix options.
LastUpdated: 2026-01-25T10:24:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Bug Analysis: Hydration Blank + Duplicate User Messages

## 1) Symptom Summary

**User‑reported symptoms (Jan 25, 2026):**

- Reloading the page shows **no hydrated conversation** (empty timeline), despite having just chatted.
- After the first prompt, the **user message appears twice** (same content, two entries).
- Console warning: `Selector selectTimelineEntities returned a different result ...` (memoization warning).

## 2) Expected vs. Actual

### Expected
- `/timeline` returns a snapshot; UI shows prior messages after reload.
- User message appears **once**, then assistant response.

### Actual
- Hydration shows empty list on reload.
- User message duplicates after first round (live and/or after hydration).

## 3) Environment and Recent Changes

- Pinocchio web chat was updated to **Approach B**:
  - timeline projections are canonical
  - SEM frames retained for live streaming
  - `/hydrate` removed; `/timeline` is sole hydration path
  - timeline upserts emitted over WS as `timeline.upsert` events
  - in‑memory TimelineStore used when DB not configured

This change introduces two new failure modes:

1) **Hydration depends on `/timeline`**; any proxy or request issue yields empty UI.
2) **Optimistic user message** is now a separate entity from the canonical timeline user message, unless reconciled.

## 4) Reproduction Steps (as inferred)

1) Start web chat in dev (Vite at :5173, backend at :8080).
2) Send a prompt (e.g., “yo”).
3) Observe UI shows user message twice.
4) Reload page; hydration shows empty list.

## 5) Primary Hypotheses (Hydration Blank)

### H1 — `/timeline` request never reaches backend (proxy/basePrefix)

- The network log shown contains only Vite asset loads; no `/timeline` request.
- If `basePrefix` resolves to a non‑proxied path, the request can hit Vite and return 404.

**Evidence to check:**
- Devtools Network: look for `/timeline?conv_id=...` and response status.
- Check Vite proxy config for `/timeline` (it exists, but verify it’s loaded).
- Confirm `basePrefixFromLocation()` returns `''` (root), not `'/something'`.

**Likely fix if true:**
- Ensure Vite proxy includes `/timeline` (already present) and the app is served at root.
- Add logging in `wsManager.hydrate()` when `/timeline` fails (currently silent).

### H2 — Timeline store is empty (in‑memory store reset)

- With no DB configured, timeline data lives only in memory.
- If backend restarts (auto‑reload, rebuild, go run), it loses all entities.

**Evidence to check:**
- Backend logs for restart or rebuild.
- Confirm the Go process stayed alive between send and reload.

**Likely fix if true:**
- Enable SQLite timeline store (`--timeline-db`) for persistence.
- Add a banner if the in‑memory store is in use (dev warning).

### H3 — conv_id mismatch

- If the URL `conv_id` is missing or changed, `/timeline` is queried with a new conversation that has no entities.

**Evidence to check:**
- On reload, inspect URL query param `conv_id`.
- Compare to backend logs (conv_id on /chat, on /timeline).

**Likely fix if true:**
- Ensure `setConvIdInLocation()` runs before or after /chat.
- Confirm /chat doesn’t replace conv_id unexpectedly.

## 6) Primary Hypotheses (Duplicate User Messages)

### D1 — Optimistic local user message + canonical timeline user message

Current behavior:

- ChatWidget adds **optimistic user message** with ID `user-${Date.now()}`.
- Backend persists user message with ID `user-${turnID}` and emits `timeline.upsert`.
- These IDs do not match, so **two entities** appear.

**Evidence to check:**
- Redux timeline state after first prompt; compare IDs for user messages.

**Likely fix:** (choose one)

1) **Reconcile optimistic message**: after /chat returns, replace the optimistic entity with the canonical ID (`user-${turnID}`) or delete the optimistic entry.
2) **Send a client_message_id** with /chat and use it as the user entity ID so timeline upsert updates the same entity.
3) **Disable optimistic echo** once timeline upserts are active (trades latency for correctness).

### D2 — Timeline snapshot contains duplicates

Less likely, but possible if user messages are inserted twice on the backend. Check that `/chat` isn’t called twice or that timeline store isn’t upserting twice with different IDs.

**Evidence to check:**
- Server logs for `startRunForPrompt` called twice.
- Timeline store entity IDs for user messages.

## 7) Secondary Issue: Redux Selector Warning

`selectTimelineEntities` returns a new array on each call, causing a warning about memoization. This is **not the root cause** of hydration failure, but it increases render noise and can mask other console warnings.

**Fix option:**
- Use `createSelector` to memoize or select `state.timeline.order` and map in the component.

## 8) Diagnostic Checklist (Concrete)

### Backend

- Add logging to `/timeline` handler:
  - conv_id, since_version, entity count
- Add log when `timeline.upsert` is emitted for user messages
- Confirm timeline store choice (SQLite vs in‑memory) on startup

### Frontend

- Log when `/timeline` fetch fails (status code + body)
- Log `conv_id` used by hydration
- Track entity IDs after first prompt to confirm D1

## 9) Proposed Fixes (Prioritized)

### Fix A — Deduplicate user messages (high impact)

- Add `client_message_id` to /chat request; backend uses it for user message entity ID.
- Or reconcile optimistic user message with the returned `turn_id`.

### Fix B — Make hydration failure visible (medium impact)

- In `wsManager.hydrate()`, log or surface when `/timeline` fails.
- This will reveal if the issue is a 404, CORS, or empty response.

### Fix C — Persist timeline store (if needed)

- For dev stability: run backend with `--timeline-db /tmp/pinocchio-timeline.db`.
- For production: ensure persistent storage configured.

## 10) Where to Start (Code Pointers)

- `pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx` — optimistic user message insertion and conv_id URL behavior
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts` — hydration fetch /timeline (silent failure)
- `pinocchio/cmd/web-chat/web/src/sem/registry.ts` — handling `timeline.upsert` events
- `pinocchio/pkg/webchat/router.go` — user message upsert ID (`user-${turnID}`)
- `pinocchio/pkg/webchat/timeline_upsert.go` — WS emission of timeline deltas

---

## Recommended Next Investigation

1) Capture `/timeline` request/response on reload (status + payload).
2) Inspect Redux timeline state after first prompt to confirm duplicate IDs.
3) Decide on reconciliation strategy (client_message_id vs post‑chat replacement).

This should confirm whether the root cause is **missing hydration data** (request/response) or **duplicate entity IDs** (optimistic echo).
