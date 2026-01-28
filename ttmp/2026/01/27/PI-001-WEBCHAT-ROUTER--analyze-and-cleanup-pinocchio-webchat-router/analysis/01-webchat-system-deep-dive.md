---
Title: Webchat System Deep Dive
Ticket: PI-001-WEBCHAT-ROUTER
Status: active
Topics:
    - analysis
    - webchat
    - refactor
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: |-
        Frontend WS + timeline hydration behavior
        Frontend WS + timeline hydration
    - Path: ../../../../../../../pinocchio/pkg/webchat/connection_pool.go
      Note: |-
        WebSocket lifecycle and idle detection
        WebSocket pool
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
      Note: |-
        Conversation lifecycle, getOrCreateConv, stream setup
        Conversation lifecycle and getOrCreateConv
    - Path: ../../../../../../../pinocchio/pkg/webchat/engine_from_req.go
      Note: |-
        Request policy for conv_id/profile/overrides
        Request policy for conv/profile/overrides
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: |-
        HTTP/WS handlers, request flow, run loop, timeline endpoints
        HTTP/WS handlers and run loop
    - Path: ../../../../../../../pinocchio/pkg/webchat/stream_coordinator.go
      Note: |-
        Event consumption and SEM frame emission
        Event stream consumption and SEM frames
    - Path: ../../../../../../../pinocchio/pkg/webchat/timeline_projector.go
      Note: |-
        SEM -> durable timeline projection
        SEM to timeline projection
ExternalSources: []
Summary: A full, didactic walkthrough of the pinocchio webchat backend/frontend flow, from HTTP/WS entrypoints to SEM and timeline projection.
LastUpdated: 2026-01-27T19:45:00-05:00
WhatFor: Reference-quality understanding of how the webchat system currently works.
WhenToUse: When changing router, conversation lifecycle, or frontend integration.
---


# Webchat System Deep Dive (Textbook-Style)

## 0. Reading Map (Why this document exists)

This document explains the webchat system as a composed pipeline. Each component is introduced with its purpose, interfaces, and the invariants it maintains. It intentionally blends prose, pseudocode, and diagrams so you can reason both about control flow and about data flow.

If you only read one section, read **Section 2 (Architecture Overview)** and **Section 4 (Request and Event Flow)**. They contain the key mental model.

---

## 1. System at a Glance

**The webchat system is a stateful streaming chat server with a small frontend**. It is built on top of Geppetto's inference engines and emits **SEM (Semantic Events)** over WebSocket to the browser. It also maintains an optional **durable timeline** (SQLite or in-memory) for hydration and reload.

A useful one-line summary:

> **HTTP starts work, WebSocket streams results, timeline snapshots repair state.**

---

## 2. Architecture Overview

### 2.1 Big-Picture Diagram

```
┌────────────────────────────────────────────────────────────┐
│                          Browser                           │
│                                                            │
│  UI (ChatWidget.tsx) ── POST /chat ───────────────┐         │
│           ▲                 ▲                      │         │
│           │                 │                      │         │
│           │           GET /timeline                │         │
│           │                 │                      │         │
│           └───── WS /ws ◀───┴── SEM frames ◀────────┘         │
│               (wsManager.ts)                                 │
└────────────────────────────────────────────────────────────┘
                          │
                          │ WebSocket (SEM)
                          ▼
┌────────────────────────────────────────────────────────────┐
│                         Backend                             │
│                                                            │
│  Router (router.go)                                        │
│   ├─ EngineFromReqBuilder (engine_from_req.go)             │
│   ├─ Conversation Manager (conversation.go)                │
│   │   ├─ Session + Engine                                  │
│   │   ├─ ConnectionPool (connection_pool.go)               │
│   │   └─ StreamCoordinator (stream_coordinator.go)         │
│   ├─ SEM Translator (sem_translator.go)                    │
│   └─ Timeline Projector (timeline_projector.go)            │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 2.2 Key Concepts (Callouts)

**Callout: Conversation vs Run**
- **Conversation**: keyed by `conv_id`, persistent identity across messages and reloads.
- **Run** (aka session): a specific inference execution with `run_id` / `session_id`.

**Callout: Event Router**
- Internally uses Geppetto's `events.EventRouter` with Watermill.
- Can be in-memory or Redis Streams-backed (see `pinocchio/pkg/redisstream/router.go`).

**Callout: SEM (Semantic Events)**
- A canonical JSON envelope with `sem: true` and `event` payloads.
- Produced by `EventTranslator` in `sem_translator.go` and further enhanced by `StreamCoordinator` with sequence numbers.

---

## 3. Core Components (Responsibility Map)

Below is a map of important types, their source files, and their responsibilities.

### 3.1 Router (pinocchio/pkg/webchat/router.go)

**Role:** Composition root for the webchat system.

Responsibilities:
- Build the EventRouter (Redis or in-memory).
- Register HTTP endpoints (static assets, `/chat`, `/ws`, `/timeline`, profiles, debug).
- Own the `ConvManager` (conversation lifecycle).
- Provide `RunEventRouter` to embed in other servers.

Key symbols:
- `NewRouter(ctx, parsed, staticFS)`
- `registerHTTPHandlers()`
- `BuildHTTPServer()`
- `RunEventRouter(ctx)`

### 3.2 Conversation Lifecycle (pinocchio/pkg/webchat/conversation.go)

**Role:** Owns per-conversation state and streaming attachments.

Responsibilities:
- Create or reuse conversations based on `conv_id` and engine config signature.
- Initialize `Session` + `Engine` + `EventSink`.
- Start and stop the `StreamCoordinator` (for SEM streaming).
- Attach a `ConnectionPool` (WS connections) with idle shutdown.

Key symbols:
- `getOrCreateConv(convID, profileSlug, overrides)`
- `Conversation{...}`
- `ConvManager.GetConversation`

### 3.3 Request Policy (pinocchio/pkg/webchat/engine_from_req.go)

**Role:** Decide conv_id, profile, and overrides from HTTP or WS requests.

Responsibilities:
- Parse and validate inputs.
- Use cookie `chat_profile` or existing conversation profile when missing.
- Enforce profile existence and return typed errors (`RequestBuildError`).

Key symbols:
- `DefaultEngineFromReqBuilder.BuildEngineFromReq`
- `profileSlugFromChatRequest`

### 3.4 Streaming Core (pinocchio/pkg/webchat/stream_coordinator.go)

**Role:** Convert event stream into SEM frames and dispatch to callbacks.

Responsibilities:
- Subscribe to the event topic (`chat:<conv_id>`).
- Decode Geppetto events from JSON payloads.
- Translate events into SEM frames and attach `seq` numbers.

Key symbols:
- `StreamCoordinator.Start/Stop/Close`
- `SemanticEventsFromEventWithCursor`

### 3.5 WebSocket Management (pinocchio/pkg/webchat/connection_pool.go)

**Role:** Maintain WS connections and broadcast SEM frames.

Responsibilities:
- Add/remove connections.
- Broadcast frames to all connections.
- Detect idle (no connections) and trigger cleanup.

Key symbols:
- `ConnectionPool.Add/Remove/Broadcast`

### 3.6 Timeline Projection (pinocchio/pkg/webchat/timeline_projector.go)

**Role:** Convert SEM frames into durable timeline entities.

Responsibilities:
- Decode SEM frames.
- Maintain caches for streaming role resolution and planning aggregation.
- Upsert into TimelineStore (SQLite or in-memory).
- Provide `timeline.upsert` events for live UIs.

Key symbols:
- `TimelineProjector.ApplySemFrame`
- `TimelineStore` implementations

### 3.7 Frontend Integration (pinocchio/cmd/web-chat/web)

**Role:** Chat UI and WS integration.

Key behaviors:
- Generates or persists `conv_id` in URL query params.
- Opens `WS /ws?conv_id=...` via `wsManager.ts`.
- Hydrates from `GET /timeline`.
- Consumes SEM frames and updates timeline store.

---

## 4. Request and Event Flow (End-to-End)

### 4.1 HTTP: Start a Chat Run (`POST /chat`)

**Call path:**

1. `router.go` handles `POST /chat`.
2. `EngineFromReqBuilder` resolves `conv_id`, profile, overrides.
3. `getOrCreateConv` creates or reuses Conversation (with engine signature checks).
4. Run loop is started (`startRunForPrompt`).
5. Inference emits events into the EventRouter.
6. StreamCoordinator picks events up and sends SEM to WS clients.

**Pseudocode:**

```
handleChatRequest(req):
  input, body = BuildEngineFromReq(req)
  conv = getOrCreateConv(input.conv_id, input.profile, input.overrides)

  if conv busy:
    enqueue prompt + return 202

  mark conv running
  response = startRunForPrompt(conv, input.profile, input.overrides, body.prompt)
  return response
```

**Important invariants:**
- At most one run is active per conversation. If already running, the request is queued.
- The idempotency key is stored and reused to return the same response for duplicate requests.

### 4.2 WebSocket: Join a Conversation (`GET /ws?conv_id=...`)

**Call path:**

1. Router upgrades to WebSocket.
2. `EngineFromReqBuilder` resolves conv/profile (from query or cookie).
3. `getOrCreateConv` ensures conversation exists.
4. Connection is added to `ConnectionPool`.
5. WS loop listens for incoming messages (ping/pong).
6. Outbound SEM frames are broadcast by StreamCoordinator callbacks.

**Pseudocode:**

```
wsHandler(req):
  conn = upgrade(req)
  input = BuildEngineFromReq(req)
  conv = getOrCreateConv(input.conv_id, input.profile)
  addConn(conv, conn)
  send WS hello frame
  while conn open:
    if ping -> send pong
```

### 4.3 Event Path: Engine -> SEM -> UI

**Pipeline:**

```
Engine emits Event -> EventRouter -> StreamCoordinator
  -> SemanticEventsFromEventWithCursor (SEM frames)
  -> ConnectionPool.Broadcast(frame)
  -> wsManager.handleSem(frame)
  -> timelineSlice updates UI state
```

**Key detail:** `StreamCoordinator` injects `seq` and `stream_id` into SEM frames so the frontend can order buffered frames deterministically.

### 4.4 Timeline Hydration (`GET /timeline`)

- The frontend requests `/timeline?conv_id=...` before consuming buffered SEM frames.
- `TimelineProjector` writes to `TimelineStore` (SQLite or in-memory) while events stream.
- The GET returns a `TimelineSnapshotV1` containing entities and a monotonic version.

---

## 5. Conversation Lifecycle in Detail

### 5.1 Creation and Reuse: `getOrCreateConv`

`getOrCreateConv` is the heart of lifecycle management. It performs **two decisions**:

1. Should this conv reuse existing engine/session?
2. Should it rebuild because config or profile changed?

**Decision logic:**

```
cfg = BuildConfig(profile, overrides)
newSig = cfg.Signature()

if conv exists:
  if profile or signature changed:
    rebuild engine/sink/subscriber
    restart stream coordinator
  return conv

else:
  create new Conversation
  build engine/sink/subscriber
  start stream
  store in ConvManager
```

**Important invariants:**
- Conversation uses a stable `conv_id` and a single `run_id` (SessionID) for its lifetime.
- Rebuilding changes engine and stream, but the conversation ID remains constant.
- Timeline projector and SEM buffer are attached if a timeline store exists.

### 5.2 Queue and Idempotency

**Why:** Prevent overlapping runs in the same conversation, while allowing multiple requests.

Key fields:
- `runningKey`: the active idempotency key.
- `queue`: FIFO of pending prompts.
- `requests`: map of idempotency key -> status response.

If a new request arrives while a run is active:
- It is queued and returns HTTP 202 (Accepted).
- The queue is drained automatically when the run finishes (`tryDrainQueue`).

---

## 6. Engine Composition and Overrides

### 6.1 EngineConfig (pinocchio/pkg/webchat/engine_config.go)

The engine config determines how the engine is built:
- `ProfileSlug`
- `SystemPrompt`
- `Middlewares`
- `Tools`
- `StepSettings` (provider/model/credentials)

**Signature:** The config is serialized (excluding secrets) and used to decide whether a conversation must rebuild.

### 6.2 Override Handling

Overrides can be supplied via request body:

```
{
  "overrides": {
    "system_prompt": "...",
    "middlewares": [{"name":"agentmode", "config": {...}}],
    "tools": ["calculator"]
  }
}
```

If `Profile.AllowOverrides` is false, overrides are rejected.

---

## 7. Streaming and SEM Translation

### 7.1 Event Translation

The `EventTranslator` turns Geppetto events into SEM frames using a registry (`semregistry`).

**Conceptual algorithm:**

```
frames = semregistry.Handle(event)
for frame in frames:
  add seq/stream_id
  broadcast
```

### 7.2 WS Frames and SEM Envelope

All frames follow this envelope shape:

```json
{
  "sem": true,
  "event": {
    "type": "llm.delta",
    "id": "...",
    "data": { ... }
  }
}
```

The frontend uses `sem/registry.ts` to map frames into timeline entities.

---

## 8. Timeline Projection (Durable Hydration)

The TimelineProjector writes **projection entities** into a `TimelineStore` as SEM frames pass through. This allows the frontend to reload the page and recover history.

**Key mechanisms:**
- Streaming text (`llm.delta`) is throttled (at most once per 250ms per message).
- Message roles are inferred from `llm.start`/`llm.thinking.start` events.
- Planning aggregation produces a single entity per run.

**Store options:**
- SQLite (`NewSQLiteTimelineStore`) for durable storage.
- In-memory (`NewInMemoryTimelineStore`) for ephemeral sessions.

---

## 9. Frontend Flow (Webchat UI)

**Conversation identity:**
- `conv_id` is stored in URL query params.
- On first message, the frontend pins the conv_id in the URL.

**Hydration strategy:**
1. Open WS connection.
2. Fetch `/timeline` snapshot.
3. Apply snapshot to local state.
4. Replay buffered SEM frames in sequence order.

**Pseudocode:**

```
connectWS(convId):
  ws = new WebSocket(`/ws?conv_id=${convId}`)
  buffer frames until hydrated

hydrate(convId):
  snapshot = GET /timeline?conv_id=...
  apply snapshot entities
  sort buffered frames by seq and apply
```

---

## 10. Operational Concerns and Debugging

**Key settings:**
- `idle-timeout-seconds`: stops stream if no WS clients.
- `timeline-dsn` / `timeline-db`: enables durable timeline snapshots.
- `emit-planning-stubs`: injects stub planning events for demo UI.

**Debug endpoints (gated):**
- `/debug/step/enable`
- `/debug/step/disable`
- `/debug/continue`

These provide manual control over step-mode execution.

---

## 11. Summary: What the System Guarantees

- **Per-conversation serialization**: only one run at a time; queued requests are deterministic.
- **Event streaming**: SEM frames appear in-order with `seq` metadata.
- **Hydration**: timeline snapshots provide durable rehydration when enabled.
- **Composable profiles**: system prompt + middlewares + tools per profile.

If you change any of: `Router`, `getOrCreateConv`, or `StreamCoordinator`, you are changing the core control plane for the system.
