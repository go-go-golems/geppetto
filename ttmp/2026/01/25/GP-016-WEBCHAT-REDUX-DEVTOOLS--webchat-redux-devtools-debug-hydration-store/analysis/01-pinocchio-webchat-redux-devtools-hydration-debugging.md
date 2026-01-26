---
Title: 'Pinocchio webchat: Redux DevTools + hydration debugging'
Ticket: GP-016-WEBCHAT-REDUX-DEVTOOLS
Status: active
Topics:
    - frontend
    - hydration
    - bug
    - websocket
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx
      Note: Optimistic user message + ensureHydrated flow
    - Path: pinocchio/cmd/web-chat/web/src/store/store.ts
      Note: Redux store wiring (configureStore)
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Hydration path (timeline clear + snapshot replay)
    - Path: pinocchio/cmd/web-chat/web/vite.config.ts
      Note: Vite DEV proxy config + env wiring
ExternalSources: []
Summary: How to enable Redux DevTools (local or remote) in pinocchio webchat and use it to debug hydration + timeline dedup.
LastUpdated: 2026-01-25T17:52:11.37473733-05:00
WhatFor: Guide for wiring Redux DevTools in Vite dev mode and inspecting hydration behavior in pinocchio webchat.
WhenToUse: When investigating hydration gaps, duplicate user messages, or missing timeline entities in pinocchio webchat.
---


# Pinocchio webchat: Redux DevTools + hydration debugging

## Context

- Related tickets: GP-012-WEBCHAT-HYDRATION-BUG, GP-013-WEBCHAT-HYDRATION-RENDERING.
- Pinocchio webchat is a React + RTK app (`pinocchio/cmd/web-chat/web`).
- Hydration flow lives in the WS manager (`src/ws/wsManager.ts`) and Chat widget (`src/chat/ChatWidget.tsx`).

## Current wiring (so we know what to watch)

### Store setup

- Store is `configureStore` with reducers for `app`, `timeline`, and `errors`.
- Default Redux DevTools (browser extension) is enabled by RTK automatically in dev builds.

### Hydration flow summary

On the first message:

1. Optimistic user message is added (`timeline/addEntity`) with a local `user-<ts>` id.
2. WS is connected with `hydrate: false` to avoid wiping optimistic state.
3. `/chat` returns `conv_id` and `session_id`.
4. `ensureHydrated()` clears the timeline and refetches `/timeline` to rehydrate with canonical entities.
5. Buffered WS events are then applied in-order.

Actions worth filtering in DevTools:

- `timeline/clear`
- `timeline/addEntity`
- `timeline/upsertEntity`
- `app/setConvId`
- `app/setRunId`
- `app/setWsStatus`
- `app/setLastSeq`

## Redux DevTools options

### Option A: browser extension (simplest)

RTK already enables the extension in dev, so you can just open the Redux DevTools extension while running `npm run dev` and the Go server. This is enough for most hydration debugging.

### Option B: remote devtools (more flexible)

Remote devtools is useful if you want a dedicated UI, a second browser, or a persistent timeline while reloading. It has three moving parts:

1. **App** (pinocchio webchat) emits Redux actions/state.
2. **Bridge server** (`@redux-devtools/cli`) relays actions over WS.
3. **Monitor UI** (extension in Remote mode or the CLI-provided UI).

`app  <--ws-->  redux-devtools (CLI server)  <--ws-->  monitor UI`

## Proposed pinocchio wiring for remote devtools (DEV-only)

### 1) Add dependencies (web frontend)

From `pinocchio/cmd/web-chat/web`:

- Runtime: `@redux-devtools/remote`
- Dev dependency: `@redux-devtools/cli`

### 2) Gate remote devtools via Vite DEV mode

Use `import.meta.env.DEV` and a `VITE_*` env flag so prod builds stay clean. Example pattern:

```ts
import { configureStore } from '@reduxjs/toolkit';
import { devToolsEnhancer } from '@redux-devtools/remote';

const enableRemoteDevtools =
  import.meta.env.DEV && import.meta.env.VITE_REMOTE_DEVTOOLS === '1';

export const store = configureStore({
  reducer: {
    app: appSlice.reducer,
    timeline: timelineSlice.reducer,
    errors: errorsSlice.reducer,
  },
  devTools: !enableRemoteDevtools,
  enhancers: (getDefaultEnhancers) =>
    enableRemoteDevtools
      ? getDefaultEnhancers().concat(devToolsEnhancer({ realtime: true, hostname: 'localhost', port: 8000 }))
      : getDefaultEnhancers(),
});
```

Notes:

- `devTools: !enableRemoteDevtools` prevents double-devtools connections.
- Vite will expose `VITE_REMOTE_DEVTOOLS=1` to `import.meta.env`.
- You can also expose `VITE_REMOTE_DEVTOOLS_HOST` / `VITE_REMOTE_DEVTOOLS_PORT` if needed.

### 3) Start the bridge server

From the same frontend directory (or repo root):

```bash
npx redux-devtools --hostname=localhost --port=8000 --open=browser
```

If the UI opens but doesn’t connect, choose “Use custom (local) server” in its settings with `localhost:8000`.

### 4) Start Vite + Go backend

- Vite: `npm run dev` in `pinocchio/cmd/web-chat/web`
- Backend: run the pinocchio webchat Go server (as usual)
- Ensure `VITE_BACKEND_ORIGIN` points to the Go server if needed

## Hydration debugging checklist

- Confirm `app/convId` is set immediately after `/chat` returns.
- Look for `timeline/clear` followed by `timeline/upsertEntity` bursts when `ensureHydrated()` runs.
- If you see duplicates, check if two entities share the same `id` or if optimistic `user-<ts>` messages remain after hydration.
- Compare `app/lastSeq` and the order of `timeline/upsertEntity` actions to spot out-of-order WS replays.

## Trace notes (2026-01-25)

Inputs:

- Redux DevTools export: `/home/manuel/Downloads/state.json` (pre-hydration).
- Timeline snapshot (GET `/timeline`): conv_id `ccb79b63-8381-4a03-90aa-4ed89f83e2d2`, version 12.

Observations:

- `state.json` shows three user entities in `timeline.byId` before hydration:
  - `user-d5b6c425-...` (content `hello`, createdAt `1769401816114`)
  - `user-1769401818088` (content `hello`, createdAt `1769401818088`) ← optimistic
  - `user-93fd974d-...` (content `hello`, version 5, createdAt `0`) ← server entity
- `/timeline` includes **two** user `hello` messages with distinct ids:
  - `user-d5b6c425-...` at `1769401816114`
  - `user-93fd974d-...` at `1769401818097`
- This indicates:
  - The double-widget state pre-hydration is caused by optimistic `user-<ts>` plus server user entity (different ids).
  - The server timeline also has two separate `hello` prompts (distinct ids, ~2s apart), which may reflect double submit or two distinct sends.
  - The WS timeline upsert for `user-93fd...` carried `createdAtMs=0` (client stored `createdAt: 0`), even though GET `/timeline` returns proper timestamps.

Implications / next checks:

- If you only submitted `hello` once, investigate a duplicate POST `/chat` (check browser network log, idempotency key reuse, or double submit).
- Consider giving the optimistic user message a stable id that matches the server id, or merging it on the first user `timeline.upsert`.
- The WS `timeline.upsert` path may be missing `created_at_ms` for user entities (or sending `0`); verify server-side payloads.

## Suggested next steps

- Wire the DEV-only remote devtools toggle behind `VITE_REMOTE_DEVTOOLS=1`.
- Verify that hydration clears the optimistic message and replays the canonical user entity once.
- If duplicates persist, consider using server-provided message IDs for the optimistic message so upserts collapse cleanly.
