---
Title: Web Agent Debug UI Migration Analysis for Pinocchio
Ticket: GP-001-ADD-DEBUG-UI
Status: active
Topics:
    - frontend
    - geppetto
    - migration
    - conversation
    - events
    - architecture
DocType: planning
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/llm-runner/serve.go
      Note: Offline artifact viewer reference implementation and UI fallback behavior
    - Path: pinocchio/pkg/persistence/chatstore/timeline_store.go
      Note: Canonical timeline projection storage contract for live level-2 debug mode
    - Path: pinocchio/pkg/persistence/chatstore/turn_store.go
      Note: Canonical turn snapshot storage contract for live level-2 debug mode
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Conversation manager runtime state used for live conversation listing and selection
    - Path: pinocchio/pkg/webchat/router.go
      Note: Existing live routes and handler patterns used for debug API consolidation
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Projection update pipeline used for timeline/debug inspector data
    - Path: web-agent-example/cmd/web-agent-debug/web/src/api/debugApi.ts
      Note: Current frontend contract expectations and mismatch points
ExternalSources: []
Summary: Deep migration analysis and execution plan to move debug UI ownership into pinocchio with two supported modes: offline viewer (filesystem and sqlite) and live level-2 inspector.
LastUpdated: 2026-02-13T18:06:00-05:00
WhatFor: Provide no-backwards-compatibility migration strategy to make pinocchio the single owner of debug UI runtime + reusable React/RTK inspector package stack.
WhenToUse: Use when implementing GP-001 with pinocchio ownership and when reviewing scope, sequence, and deletion plan.
---

# GP-001 Analysis: Move Debug UI Into Pinocchio (Simplified)

## Executive Decision (Updated)

We should move debug UI ownership into `pinocchio`, not `geppetto`.

Target scope is explicitly:

1. **Offline viewer** (artifact directory and/or sqlite snapshot inspection; no live runtime dependency).
2. **Level-2 live debug inspector** (conversation/timeline/turn/event inspection backed by runtime stores).

This simplifies migration significantly:

- no cross-repo ownership split between geppetto and pinocchio,
- no need to re-home `webchat`/`chatstore` into geppetto first,
- one backend/runtime owner for live inspection (`pinocchio`).

No backward compatibility shims are required.

## Scope Definition

### Offline viewer (required)

- Input:
  - artifact directories (`final_turn*.yaml`, `events*.ndjson`, logs/raw payload files), and
  - sqlite files containing persisted turns/timelines.
- Behavior: read-only historical inspection.
- Runtime dependency:
  - none on `webchat` live runtime, and
  - optional read dependency on `chatstore` sqlite schema when sqlite is provided.
- Suggested backend basis:
  - adapt `llm-runner`-style artifact parsing into pinocchio-owned debug API, and
  - add offline sqlite readers for `chatstore` timeline/turn stores.

### Level-2 live debug (required)

- Input: active conversation runtime + persisted timeline/turn snapshots.
- Behavior: inspect conversations, turns, timeline projection entities, and buffered events.
- Runtime dependency: `pinocchio/pkg/webchat` + `pinocchio/pkg/persistence/chatstore`.
- Excludes full chat UX rework (this is inspector/debug tooling, not chat product UX).

## Hard Evidence Snapshot (Current State)

### Surface area and complexity

- `web-agent-debug` frontend TS/TSX files: `79`.
- `web-agent-debug` runtime TS/TSX LOC (excluding stories/tests/mocks): `3677`.
- `web-agent-debug` Go harness LOC: `1003` across `8` files.
- `pinocchio/pkg/webchat` Go LOC: `5239` across `31` files.
- `pinocchio/pkg/persistence/chatstore` Go files: `8`.

### Existing backend route reality vs UI expectations

Frontend currently expects `/debug/*` resources via RTK API:
- `web-agent-example/cmd/web-agent-debug/web/src/api/debugApi.ts:47`

`pinocchio/webchat` currently exposes core live routes:
- `/ws`: `pinocchio/pkg/webchat/router.go:500`
- `/timeline`: `pinocchio/pkg/webchat/router.go:671`
- `/turns`: `pinocchio/pkg/webchat/router.go:732`
- `/chat`: `pinocchio/pkg/webchat/router.go:823`
- step-debug control endpoints: `pinocchio/pkg/webchat/router.go:386`, `pinocchio/pkg/webchat/router.go:423`, `pinocchio/pkg/webchat/router.go:457`

Missing today for the UI:
- canonical conversation list/detail endpoints,
- canonical event-list endpoint from sem buffer,
- canonical turn-detail-by-phase endpoint.

### Data-shape mismatches to fix

1. `turns` response shape:
- backend returns envelope with `items`: `pinocchio/pkg/webchat/router.go:722`
- frontend expects `TurnSnapshot[]`: `web-agent-example/cmd/web-agent-debug/web/src/api/debugApi.ts:72`

2. turn payload shape:
- persisted as YAML string: `pinocchio/pkg/persistence/chatstore/turn_store.go:12`
- frontend expects parsed turn object: `web-agent-example/cmd/web-agent-debug/web/src/types/index.ts:39`

3. timeline entity shape:
- backend emits protobuf oneof entities (`message`, `toolCall`, `toolResult`, etc.): `pinocchio/pkg/sem/pb/proto/sem/timeline/transport.pb.go:24`
- frontend currently uses flattened `props` model: `web-agent-example/cmd/web-agent-debug/web/src/types/index.ts:124`

## Clarifications (Requested)

### 1) `TurnSnapshot[]` vs \"envelope with items\"

- Backend `/turns` currently returns an envelope object:
  - `{ conv_id, session_id, phase, since_ms, items: TurnSnapshot[] }`
  - source: `pinocchio/pkg/webchat/router.go:722`
- Frontend `getTurns` currently types the response directly as `TurnSnapshot[]` with no transform:
  - source: `web-agent-example/cmd/web-agent-debug/web/src/api/debugApi.ts:72`

Relation:

- `items` is the actual array payload.
- Envelope fields are query/context metadata for pagination/filter provenance.

Design update:

- Standardize debug API to envelope form for both offline and live endpoints.
- Frontend must retain and use envelope metadata (`conv_id`, `session_id`, `phase`, `since_ms`, paging hints) in RTK state and inspector context views.
- Components that only need row arrays can read `items`, but API-layer transforms must not discard envelope metadata.

### 2) Do we have a persisted blocks table?

No, not in current `chatstore` turn persistence.

- `TurnSnapshot` stores one serialized `payload` string per `(conv_id, session_id, turn_id, phase, created_at_ms)`:
  - `pinocchio/pkg/persistence/chatstore/turn_store.go:6`
- SQLite schema has a single `turns` table with `payload TEXT`; there is no `blocks` table:
  - `pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go:48`

Implication:

- block-level inspection is reconstructed by deserializing the turn payload (YAML/serde), not by querying normalized block rows.

### 3) What is a flattened `props` model, and are timeline entities arbitrary?

Flattened `props` model in current frontend means:

- every timeline entity is represented as:
  - `{ id, kind, created_at, updated_at?, version?, props: Record<string, unknown> }`
  - source: `web-agent-example/cmd/web-agent-debug/web/src/types/index.ts:124`
- all snapshot-specific fields are collapsed into generic `props`.

Backend timeline entities are protobuf oneof-based, not generic arbitrary JSON:

- `TimelineEntityV1` has fixed oneof snapshot variants (`message`, `tool_call`, `tool_result`, etc.):
  - `pinocchio/pkg/sem/pb/proto/sem/timeline/transport.pb.go:24`
  - `pinocchio/pkg/sem/pb/proto/sem/timeline/transport.pb.go:216`
- custom timeline behavior is extensible via handler registration:
  - `pinocchio/pkg/webchat/timeline_registry.go:28`

Important nuance:

- third parties can register handlers for new SEM event types and set `kind` labels freely,
- but payload shape is still constrained to the protobuf snapshot variants unless proto definitions are extended and regenerated.

### 4) Why do we care about projector/upsert if debug UI is read-only?

The UI is read-only, but **live level-2 data is write-fed by the projector pipeline**.

- `TimelineProjector.ApplySemFrame` converts incoming SEM frames into timeline store upserts:
  - `pinocchio/pkg/webchat/timeline_projector.go:30`
- conversation stream loop calls projector on each frame:
  - `pinocchio/pkg/webchat/conversation.go:230`

So:

- UI does not write timeline data,
- but live mode depends on that write path to keep `TimelineStore` current.
- offline mode can read already persisted sqlite snapshots without live projector activity.

## What We Need From Webchat for Level-2 (Minimal Slice)

Not all of `webchat` is required for inspector functionality.

Required live-debug subset:

1. Conversation registry/snapshot view from `ConvManager`:
- `pinocchio/pkg/webchat/conversation.go:57`

2. Event buffer snapshot source (`semBuf`):
- `pinocchio/pkg/webchat/sem_buffer.go:39`

3. Timeline projection source + upsert behavior:
- `pinocchio/pkg/webchat/timeline_projector.go:30`
- `pinocchio/pkg/webchat/timeline_upsert.go:10`

4. Turn/timeline store access handlers:
- `pinocchio/pkg/webchat/router.go:620`
- `pinocchio/pkg/webchat/router.go:674`

5. Turn persistence hook wiring:
- `pinocchio/pkg/webchat/turn_persister.go:21`

Can be out-of-scope for initial level-2 inspector if desired:

- full chat request handling path (`/chat`) beyond identification context,
- websocket ping/pong UX behavior details,
- profile cookie mechanics except where needed to label conversations.

## Chatstore vs Existing Geppetto Stores (Why Pinocchio Ownership Helps)

`chatstore` provides queryable debug-domain stores:

- Timeline store (`Upsert`, `GetSnapshot`):
  - `pinocchio/pkg/persistence/chatstore/timeline_store.go:13`
- Turn store (`Save`, `List` with filters):
  - `pinocchio/pkg/persistence/chatstore/turn_store.go:25`
- Durable SQLite implementations with migrations/indexes:
  - `pinocchio/pkg/persistence/chatstore/timeline_store_sqlite.go:46`
  - `pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go:43`

By contrast, geppetto currently has:

- in-memory session turn history (`Session.Turns`) and execution orchestration,
- write-side persister hook interface (`TurnPersister`) but no standard read/query API for live inspector views,
- event routing transport, not a debug-query store abstraction.

Therefore keeping debug UI runtime in pinocchio avoids unnecessary store re-architecture in geppetto.

## Target End State (No Compatibility)

## 1) Ownership and location

All debug UI product/runtime ownership moves under `pinocchio/`.

- Go command: `pinocchio/cmd/debug-ui` (or equivalent pinocchio-owned command)
- Debug backend package: `pinocchio/pkg/debugapi`
- Web workspace: `pinocchio/web/` (apps + reusable packages)

`web-agent-example/cmd/web-agent-debug` is deleted after cutover.

## 2) Unified pinocchio debug API namespace

Define canonical namespace:

- `GET /api/debug/conversations`
- `GET /api/debug/conversations/:convId`
- `GET /api/debug/conversations/:convId/sessions`
- `GET /api/debug/turns?conv_id=&session_id=&phase=&since_ms=&limit=`
- `GET /api/debug/turn/:convId/:sessionId/:turnId`
- `GET /api/debug/events/:convId?since_seq=&type=&limit=`
- `GET /api/debug/timeline?conv_id=&since_version=&limit=`
- `POST /api/debug/step/enable`
- `POST /api/debug/step/disable`
- `POST /api/debug/continue`

No `/debug/*` compatibility routes.

## 3) Two-mode backend model

### Mode A: Offline

- Endpoint family reads from:
  - filesystem artifacts (yaml/ndjson/log/raw), and/or
  - sqlite snapshots (`chatstore` timeline/turn data files).
- Reuses existing artifact parsing concepts from `llm-runner`, plus sqlite readers hosted in pinocchio.

### Mode B: Live Level-2

- Endpoint family reads webchat runtime snapshots and chatstore-backed timelines/turns.
- Uses `ConvManager`, `semBuf`, `TimelineStore`, and `TurnStore`.

Both modes feed the same frontend DTO contract to keep UI reusable, including shared metadata envelopes used by frontend filters/context state.

## 4) Reusable React/RTK package architecture (pinocchio-owned)

- `@pinocchio/debug-contract`
- `@pinocchio/debug-api`
- `@pinocchio/debug-state`
- `@pinocchio/debug-components`
- `@pinocchio/debug-app`

This keeps inspector widgets reusable across offline/live modes without duplicating app shell code.

## Migration Phases (Simplified)

## Phase A: Contract freeze in pinocchio

1. Define one DTO contract for both offline and live level-2 modes.
2. Prefer protojson timeline shape + centralized frontend adapter layer.
3. Require frontend state selectors to preserve envelope metadata (not array-only coercion).

## Phase B: Live level-2 endpoints in pinocchio

1. Add conversation snapshot/list endpoint from `ConvManager`.
2. Add events endpoint using `semBuf.Snapshot()` decoding.
3. Add turn detail endpoint by aggregating `TurnStore.List` + YAML decode.
4. Expose timeline via unified `/api/debug/timeline` namespace.

## Phase C: Offline endpoints in pinocchio

1. Port/adapt artifact parsing logic into `pinocchio/pkg/debugapi/offline`.
2. Add sqlite readers for offline turn/timeline inspection (`chatstore`-compatible files).
3. Add `runs` + `run detail` endpoints that map filesystem and sqlite sources to shared DTOs.

## Phase D: Frontend move and package split (pinocchio web)

1. Move runtime UI from `web-agent-debug` into pinocchio web workspace.
2. Extract contract/api/state/components packages.
3. Remove dead endpoints, duplicated types, and incomplete state wiring.
4. Remove runtime inline styles and tighten checks.

## Phase E: Cutover and deletion

1. Delete `web-agent-example/cmd/web-agent-debug` (web + proxy harness).
2. Remove all `/debug/*` compatibility assumptions.
3. Keep `llm-runner` independent unless separately deprecated; no forced coupling in this ticket.

## Suggested Directory Layout After Migration

```text
pinocchio/
  cmd/
    debug-ui/
      main.go
      serve.go
      gen_frontend.go
      static/...
  pkg/
    debugapi/
      contracts/
      live/
      offline/
      handlers/
  web/
    package.json
    apps/
      debug-ui/
    packages/
      debug-contract/
      debug-api/
      debug-state/
      debug-components/
```

## Why This Is Simpler

1. Runtime and stores stay with existing owner (`pinocchio`).
2. No immediate store abstraction redesign in geppetto.
3. One backend can support offline + live with one shared DTO contract.
4. `web-agent-debug` deletion becomes straightforward replacement, not a cross-repo transplant.

## Risks and Mitigations

1. Risk: mixing offline and live logic tightly.
- Mitigation: strict package split `offline/` vs `live/` with shared contract only.

2. Risk: event endpoint decode cost.
- Mitigation: `limit` and `since_seq` filters; bounded decode.

3. Risk: timeline oneof complexity in UI.
- Mitigation: centralized adapter in `debug-contract` package.

4. Risk: frontend workspace drift.
- Mitigation: single pinocchio web workspace and CI checks (`build`, `typecheck`, `test`).

## Readiness Criteria (Definition of Done)

1. Pinocchio serves debug UI with two supported modes: offline + live level-2.
2. Live inspector uses real conversation/timeline/turn/event data from webchat/chatstore.
3. Offline viewer reads artifact directories and/or sqlite snapshots through shared DTO endpoints.
4. `web-agent-debug` command/frontend is deleted.
5. No compatibility routes retained.
6. Frontend uses metadata envelopes in filters/context panels rather than discarding non-`items` fields.
7. CI covers backend contract tests + frontend package/app checks.

## Appendix: Validation commands run during analysis

Passing:

- `go test ./pinocchio/pkg/webchat`
- `go test ./geppetto/cmd/llm-runner`
- `npm run -s check:helpers:dedupe` in `web-agent-debug/web`
- `npm run -s check:styles:inline-runtime` in `web-agent-debug/web`

Failing (toolchain state, not design regression):

- `npm run -s test:unit` in `web-agent-debug/web` -> `sh: 1: vitest: not found`
- `npm run -s build` in both web apps -> `TS2688 Cannot find type definition file for 'vite/client'` (and `node` in `llm-runner` build)
