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


# The Webchat System: A Textbook Treatment

> *"The key to understanding any complex system is to identify the data that flows through it and the transformations applied at each stage."*

---

## Preface: How to Read This Document

This document explains the pinocchio webchat system as a **composed pipeline of cooperating components**. Each component is introduced with its purpose, its interfaces, and the invariants it maintains. The goal is not merely to describe *what* the code does, but to illuminate *why* it is structured as it is—so that you can reason about the system confidently when extending or debugging it.

If you are short on time, read **Section 2 (Architecture Overview)** and **Section 4 (The Request-Response Dance)**. They contain the essential mental model. The remaining sections fill in details that become important as you work more deeply with the code.

---

## 1. The Problem We Are Solving

Consider the challenge of building a web-based chat interface to a large language model. The user types a prompt; the system must:

1. **Accept the request** via HTTP
2. **Route it** to the appropriate inference engine (there may be multiple "profiles" with different configurations)
3. **Stream the response** token-by-token back to the browser
4. **Persist enough state** that if the user refreshes the page, they can resume where they left off
5. **Handle concurrent users** efficiently, without blocking or corrupting shared state

This is a **stateful streaming server**—a fundamentally different beast from a typical request/response web application. The webchat system solves this problem through a layered architecture where each layer has a single, clear responsibility.

A useful one-line summary of the entire system:

> **HTTP starts work, WebSocket streams results, timeline snapshots repair state.**

---

## 2. Architecture Overview: The Big Picture

### 2.1 The Four Layers

The webchat system can be understood as four cooperating layers:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              LAYER 4: UI                                │
│  ChatWidget.tsx renders messages; wsManager.ts handles WS + hydration   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                      SEM frames (WebSocket) + Timeline API
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          LAYER 3: HTTP/WS Gateway                        │
│  Router (router.go) accepts requests, upgrades WS, serves static assets  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ConvManager, ConnectionPool, StreamCoordinator
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       LAYER 2: Conversation Runtime                      │
│  getOrCreateConv orchestrates Engine/Session/Subscriber per conversation │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                       Events via EventRouter (Watermill)
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         LAYER 1: Inference Core                          │
│  Geppetto engines, sessions, tool loop, middlewares                      │
└─────────────────────────────────────────────────────────────────────────┘
```

**Layer 1** (Inference Core) knows nothing about HTTP—it simply runs prompts through LLM inference and emits typed events.

**Layer 2** (Conversation Runtime) manages the lifecycle of conversations: creating them, rebuilding them when configuration changes, and wiring event flow.

**Layer 3** (HTTP/WS Gateway) translates external requests into conversation operations and streams results back to clients.

**Layer 4** (UI) renders the chat and handles user input, consuming the SEM event protocol over WebSocket.

This layering ensures that each component can be understood, tested, and replaced independently.

### 2.2 Data Flow Diagram

```
┌────────────────────────────────────────────────────────────────┐
│                          Browser                                │
│                                                                │
│  ChatWidget ─────── POST /chat ────────────────┐               │
│       ▲                 ▲                      │               │
│       │                 │                      │               │
│       │           GET /timeline                │               │
│       │                 │                      │               │
│       └───── WS /ws ◀───┴── SEM frames ◀───────┘               │
│           (wsManager.ts)                                       │
└────────────────────────────────────────────────────────────────┘
                          │
                          │ WebSocket (SEM protocol)
                          ▼
┌────────────────────────────────────────────────────────────────┐
│                         Backend                                 │
│                                                                │
│  Router (router.go)                                            │
│   ├─ EngineFromReqBuilder (engine_from_req.go)                 │
│   ├─ Conversation Manager (conversation.go)                    │
│   │   ├─ Session + Engine                                      │
│   │   ├─ ConnectionPool (connection_pool.go)                   │
│   │   └─ StreamCoordinator (stream_coordinator.go)             │
│   ├─ SEM Translator (sem_translator.go)                        │
│   └─ Timeline Projector (timeline_projector.go)                │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

### 2.3 Key Concepts

Before diving into the code, we must establish precise vocabulary. Confusion over terminology is the enemy of understanding.

**Conversation** (`conv_id`): A persistent identity that survives page reloads. A single conversation may contain many user prompts and assistant responses. Think of it as a "chat thread."

**Run** (also called **Session**, identified by `run_id` or `session_id`): A specific execution context within a conversation. Currently, a conversation creates one run at initialization and reuses it. The run holds the turn history.

**Turn**: A single prompt-response pair. Each user message starts a turn; the assistant's response (possibly including tool calls) completes it.

**Profile**: A named configuration bundle specifying system prompt, available tools, middlewares, and LLM settings. Profiles allow the same server to offer different "personalities."

**SEM (Semantic Events)**: The canonical event protocol between backend and frontend. Each SEM frame is a JSON envelope with `sem: true` and an `event` object containing `type`, `id`, and `data`.

**Timeline**: A durable projection of conversation history, stored in SQLite or in-memory. The timeline allows clients to "catch up" after reconnecting.

---

## 3. Component Responsibilities

Each source file in the webchat package has a distinct role. Understanding these roles is essential for navigating the codebase.

### 3.1 Router (`router.go`)

**Role:** The composition root—the entry point that wires everything together.

The Router is created via `NewRouter(ctx, parsed, staticFS)`. It:

- Builds the underlying **EventRouter** (Watermill-based, either in-memory or Redis-backed)
- Initializes the **ProfileRegistry** and **ConvManager**
- Configures the **TimelineStore** (SQLite or in-memory)
- Registers all HTTP endpoints via `registerHTTPHandlers()`

The Router owns the `mux` (an `http.ServeMux`) and provides `Handler()` to integrate with any HTTP server.

```go
// The Router is the system's nervous system.
type Router struct {
    baseCtx       context.Context
    mux           *http.ServeMux
    router        *events.EventRouter  // Watermill event bus
    cm            *ConvManager         // Active conversations
    timelineStore TimelineStore        // Durable hydration store
    profiles      ProfileRegistry      // Named configurations
    // ... other fields
}
```

**Key insight:** The Router does not process events itself. It delegates to the appropriate Conversation, which in turn delegates to its StreamCoordinator and TimelineProjector. This separation keeps the Router code manageable.

### 3.2 Conversation Lifecycle (`conversation.go`)

**Role:** Manages per-conversation state and streaming attachments.

The central function here is `getOrCreateConv(convID, profileSlug, overrides)`. This function makes two critical decisions:

1. **Should we reuse an existing conversation?** If a conversation with this ID already exists, we might be able to reuse it.
2. **Should we rebuild because configuration changed?** If the profile or overrides differ, we must recreate the engine and stream.

```go
// Pseudocode for the decision logic
cfg := BuildConfig(profile, overrides)
newSig := cfg.Signature()

if conversation exists:
    if profile changed OR signature changed:
        rebuild engine, sink, subscriber
        restart stream coordinator
    return conversation

else:
    create new Conversation
    build engine, sink, subscriber
    start stream
    store in ConvManager
    return conversation
```

The **EngineConfig.Signature()** is a JSON representation of the configuration (excluding secrets). Comparing signatures determines whether a rebuild is necessary.

Each `Conversation` struct holds:
- `Sess` — the Geppetto Session managing turn history
- `Eng` — the inference Engine
- `Sink` — the EventSink publishing to the EventRouter
- `pool` — the ConnectionPool of active WebSocket clients
- `stream` — the StreamCoordinator consuming events
- `timelineProj` — the TimelineProjector persisting to the TimelineStore

### 3.3 Request Policy (`engine_from_req.go`)

**Role:** Determine conversation ID, profile, and overrides from incoming HTTP/WS requests.

The `EngineFromReqBuilder` interface abstracts request parsing. The default implementation:

- For **POST /chat**: Parses JSON body, generates `conv_id` if missing, resolves profile from path/cookie/existing conversation
- For **GET /ws**: Extracts `conv_id` and `profile` from query params, falls back to cookies

```go
type EngineBuildInput struct {
    ConvID      string
    ProfileSlug string
    Overrides   map[string]any
}
```

**Why an interface?** This allows integrators to customize request parsing—for example, to extract conversation ID from a JWT token or enforce organization-specific policies.

### 3.4 Streaming Core (`stream_coordinator.go`)

**Role:** Subscribe to the event topic and translate events into SEM frames.

The StreamCoordinator is the bridge between the Watermill event bus and WebSocket clients. It:

1. Subscribes to the topic `chat:<conv_id>`
2. Decodes each message into a Geppetto event
3. Calls `SemanticEventsFromEventWithCursor()` to translate into SEM frames
4. Invokes the `onFrame` callback to broadcast

```go
func (sc *StreamCoordinator) consume(ctx context.Context) {
    ch, _ := sc.subscriber.Subscribe(ctx, topicForConv(sc.convID))
    for msg := range ch {
        ev, _ := events.NewEventFromJson(msg.Payload)
        cur := StreamCursor{StreamID: extractStreamID(msg), Seq: sc.seq.Add(1)}
        
        for _, frame := range SemanticEventsFromEventWithCursor(ev, cur) {
            sc.onFrame(ev, cur, frame)
        }
        msg.Ack()
    }
}
```

**Sequence numbers** (`seq`) are injected into each frame. This allows the frontend to order frames deterministically, even if network delivery is reordered.

### 3.5 WebSocket Management (`connection_pool.go`)

**Role:** Maintain WebSocket connections and broadcast SEM frames.

The ConnectionPool is beautifully simple:

```go
type ConnectionPool struct {
    conns       map[*websocket.Conn]struct{}
    idleTimer   *time.Timer
    idleTimeout time.Duration
    onIdle      func()
}
```

It supports:
- `Add(conn)` — register a new connection
- `Remove(conn)` — unregister (and close) a connection  
- `Broadcast(data)` — send to all connections
- **Idle detection** — when no connections remain, starts a timer; if it expires, calls `onIdle()` to stop the stream

**Why idle detection?** Without it, streams would run forever even after all clients disconnect, wasting resources.

### 3.6 SEM Translation (`sem_translator.go`)

**Role:** Convert Geppetto events into the SEM wire protocol.

The translator uses a registry pattern (`semregistry`) to map event types to handler functions. Each handler returns zero or more SEM frames:

```go
semregistry.RegisterByType[*events.EventPartialCompletion](func(ev *events.EventPartialCompletion) ([][]byte, error) {
    data, _ := protoToRaw(&sempb.LlmDelta{Id: id, Delta: ev.Delta, Cumulative: ev.Completion})
    return [][]byte{wrapSem(map[string]any{"type": "llm.delta", "id": id, "data": data})}, nil
})
```

**SEM frame envelope:**
```json
{
  "sem": true,
  "event": {
    "type": "llm.delta",
    "id": "abc-123",
    "seq": 42,
    "data": { "delta": "Hello", "cumulative": "Hello, world" }
  }
}
```

The translator also maintains small caches for:
- **Message IDs** — ensuring stable IDs across `llm.start`, `llm.delta`, and `llm.final` events
- **Tool call metadata** — remembering tool names for result events

### 3.7 Timeline Projection (`timeline_projector.go`)

**Role:** Convert SEM frames into durable timeline entities for hydration.

The TimelineProjector watches the SEM stream and maintains persistent snapshots:

```go
func (p *TimelineProjector) ApplySemFrame(ctx context.Context, frame []byte) error {
    switch env.Event.Type {
    case "llm.start", "llm.thinking.start":
        // Create a message entity with streaming=true
    case "llm.delta", "llm.thinking.delta":
        // Update content (throttled to reduce write churn)
    case "llm.final":
        // Finalize with streaming=false
    case "tool.start", "tool.done", "tool.result":
        // Track tool call lifecycle
    case "planning.start", "planning.iteration", "planning.complete":
        // Aggregate into a single planning entity
    }
}
```

**Throttling:** During streaming, `llm.delta` events arrive rapidly (potentially every token). Writing to the database on every delta would be wasteful. The projector throttles writes to at most once per 250ms per message.

**Planning aggregation:** Multiple planning events (`start`, `iteration`, `reflection`, `complete`) are aggregated into a single timeline entity, keeping the timeline clean.

### 3.8 Frontend Integration (`wsManager.ts`)

**Role:** Connect to WebSocket, hydrate from timeline, and dispatch SEM events to Redux.

The WsManager implements a careful hydration protocol:

1. **Connect** WebSocket to `/ws?conv_id=...`
2. **Buffer** incoming frames until hydrated
3. **Fetch** `/timeline?conv_id=...` to get persisted state
4. **Apply** timeline snapshot to Redux store
5. **Sort** buffered frames by `seq` and apply them
6. **Process** new frames directly

```typescript
// Hydration ensures correct state reconstruction
async hydrate(args, nonce) {
    dispatch(timelineSlice.actions.clear());
    const snap = await fetch(`${basePrefix}/timeline?conv_id=${convId}`);
    applyTimelineSnapshot(snap, dispatch);
    
    // Now apply buffered frames in sequence order
    this.buffered.sort((a, b) => seqFromEnvelope(a) - seqFromEnvelope(b));
    for (const fr of this.buffered) {
        handleSem(fr, dispatch);
    }
}
```

**Why this dance?** When reconnecting, some events may have been missed. The timeline provides the "truth" up to a certain point, and buffered frames fill in anything that arrived during hydration.

---

## 4. The Request-Response Dance

Let us trace what happens when a user sends a message.

### 4.1 Phase 1: HTTP Request Arrives

```
Browser → POST /chat { "conv_id": "abc", "prompt": "Hello" }
```

The router's `handleChatRequest` function:

1. Calls `EngineFromReqBuilder.BuildEngineFromReq(req)` to parse the request
2. Calls `getOrCreateConv(convID, profileSlug, overrides)` to ensure the conversation exists
3. Checks idempotency to handle duplicate requests

```go
// Simplified pseudocode
input, body, _ := builder.BuildEngineFromReq(req)
conv, _ := router.getOrCreateConv(input.ConvID, input.ProfileSlug, input.Overrides)

if conv.isBusy() {
    conv.enqueue(prompt)
    return 202 Accepted, { "status": "queued" }
}

conv.runningKey = idempotencyKey
response := router.startRunForPrompt(conv, profileSlug, overrides, body.Prompt)
return 200 OK, response
```

### 4.2 Phase 2: Inference Begins

`startRunForPrompt` does the heavy lifting:

1. Builds the tool registry from registered factories
2. Appends a new turn from the user prompt
3. Persists the user message to the timeline (if enabled)
4. Creates an `enginebuilder.Builder` with the engine, tools, and event sinks
5. Calls `conv.Sess.StartInference(ctx)` to begin inference

```go
seed, _ := conv.Sess.AppendNewTurnFromUserPrompt(prompt)

// Persist user message to timeline
entity := &timelinepb.TimelineEntityV1{
    Id:   "user-" + turnID,
    Kind: "message",
    Snapshot: &timelinepb.TimelineEntityV1_Message{...},
}
timelineStore.Upsert(ctx, convID, entity)

// Start inference
handle, _ := conv.Sess.StartInference(ctx)
```

### 4.3 Phase 3: Events Flow

As the LLM generates tokens, the engine emits events:

```
Engine → EventSink → EventRouter → Watermill topic "chat:abc"
                                         ↓
                                  StreamCoordinator subscribes
                                         ↓
                              Translate to SEM frames
                                         ↓
                              ConnectionPool.Broadcast()
                                         ↓
                              WebSocket clients receive frames
```

Simultaneously, the TimelineProjector receives the same SEM frames and persists them:

```
SEM frame → TimelineProjector.ApplySemFrame()
                    ↓
            TimelineStore.Upsert()
                    ↓
            timeline.upsert event → WebSocket
```

### 4.4 Phase 4: Completion

When inference completes:

1. The goroutine waiting on `handle.Wait()` resumes
2. `finishRun()` clears the running state
3. `tryDrainQueue()` checks if there are queued prompts and processes them

```go
go func() {
    _, waitErr := handle.Wait()
    router.finishRun(conv, idempotencyKey, handle.InferenceID, turnID, waitErr)
    router.tryDrainQueue(conv)  // Process next queued prompt
}()
```

---

## 5. The Engine Configuration System

### 5.1 Profiles: Named Configurations

A **Profile** bundles configuration for a chat "mode":

```go
type Profile struct {
    Slug           string           // Unique identifier
    DefaultPrompt  string           // System prompt
    DefaultTools   []string         // Available tools
    DefaultMws     []MiddlewareUse  // Middleware chain
    LoopName       string           // Run loop strategy
    AllowOverrides bool             // Can clients customize?
}
```

Profiles are registered at startup:

```go
router.AddProfile(&Profile{
    Slug:          "coding",
    DefaultPrompt: "You are a helpful coding assistant.",
    DefaultTools:  []string{"calculator", "web_search"},
    AllowOverrides: true,
})
```

### 5.2 EngineConfig: The Build Specification

When a request arrives, `BuildConfig(profileSlug, overrides)` merges the profile defaults with any request-specific overrides:

```go
type EngineConfig struct {
    ProfileSlug  string
    SystemPrompt string
    Middlewares  []MiddlewareUse
    Tools        []string
    StepSettings *settings.StepSettings
}
```

The `Signature()` method produces a deterministic string representation (excluding secrets). This signature is compared to decide whether to rebuild the engine when a conversation is reused with different settings.

### 5.3 Override Handling

Clients can customize behavior if `Profile.AllowOverrides` is true:

```json
{
  "prompt": "Explain quantum computing",
  "conv_id": "abc",
  "overrides": {
    "system_prompt": "You are a physics professor.",
    "middlewares": [{"name": "planning", "config": {...}}],
    "tools": ["calculator"]
  }
}
```

The router validates overrides and rejects unknown or malformed values.

---

## 6. Event Translation: From Engine Events to SEM

### 6.1 The Translation Registry

Geppetto's engine emits strongly-typed Go events. The SEM translator converts these to the wire protocol using a registry of handlers:

```go
// Registration pattern
semregistry.RegisterByType[*events.EventPartialCompletion](func(ev) ([][]byte, error) {
    // Convert to SEM frame
})
```

### 6.2 Common SEM Event Types

| Geppetto Event | SEM Type | Description |
|----------------|----------|-------------|
| `EventPartialCompletionStart` | `llm.start` | Response begins |
| `EventPartialCompletion` | `llm.delta` | New tokens |
| `EventFinal` | `llm.final` | Response complete |
| `EventThinkingPartial` | `llm.thinking.delta` | Reasoning tokens |
| `EventToolCall` | `tool.start` | Tool invocation |
| `EventToolResult` | `tool.result` + `tool.done` | Tool output |
| `EventPlanningStart` | `planning.start` | Agentic planning |
| `EventExecutionComplete` | `execution.complete` | Run finished |

### 6.3 Sequence Numbers and Ordering

Each SEM frame receives a `seq` number from the StreamCoordinator:

```go
cur := StreamCursor{
    StreamID: extractStreamID(msg),  // Redis stream ID or empty
    Seq:      sc.seq.Add(1),         // Monotonic counter
}
```

The frontend uses `seq` to:
- Order frames during hydration
- Detect duplicates
- Debug timing issues

---

## 7. Timeline: Durable State for Hydration

### 7.1 Why Timelines Exist

Without timelines, refreshing the browser would lose all conversation history. The timeline provides a **durable projection** of conversation state that survives server restarts and page reloads.

### 7.2 Timeline Entities

The timeline stores entities, not raw events:

```protobuf
message TimelineEntityV1 {
  string id = 1;
  string kind = 2;  // "message", "tool_call", "planning", etc.
  oneof snapshot {
    MessageSnapshotV1 message = 10;
    ToolCallSnapshotV1 tool_call = 11;
    PlanningSnapshotV1 planning = 12;
    // ...
  }
}
```

Entities are identified by stable IDs and **upserted** (inserted or updated). This allows streaming updates to accumulate into a final state.

### 7.3 The Hydration Protocol

```
Client                              Server
   │                                   │
   │ ──── WS /ws?conv_id=abc ────────▶ │
   │ ◀──── Buffer frames ──────────────│
   │                                   │
   │ ──── GET /timeline?conv_id=abc ─▶ │
   │ ◀──── TimelineSnapshotV1 ─────────│
   │                                   │
   │  [Apply snapshot to Redux store]  │
   │  [Sort buffered frames by seq]    │
   │  [Apply buffered frames]          │
   │                                   │
   │ ◀──── Real-time SEM frames ───────│
```

This protocol ensures:
- No events are lost during hydration
- Events are applied in the correct order
- The timeline provides a consistent base state

---

## 8. Concurrency and Queueing

### 8.1 One Run at a Time

The webchat system enforces a crucial invariant: **at most one inference run is active per conversation at any time.**

This simplifies reasoning about state—you never have to worry about interleaved responses from multiple runs.

```go
// In handleChatRequest
if conv.isBusy() {
    pos := conv.enqueue(queuedChat{...})
    return 202 Accepted, { "status": "queued", "queue_position": pos }
}

conv.runningKey = idempotencyKey
// Start the run
```

### 8.2 Idempotency

Duplicate requests (same idempotency key) return the same response:

```go
if rec, ok := conv.getRecord(idempotencyKey); ok {
    return rec.Response
}
```

This handles network retries gracefully.

### 8.3 Queue Draining

When a run completes, the queue is automatically drained:

```go
func (r *Router) tryDrainQueue(conv *Conversation) {
    for {
        if conv.isBusy() { return }
        q, ok := conv.dequeue()
        if !ok { return }
        
        conv.runningKey = q.IdempotencyKey
        r.startRunForPrompt(conv, q.ProfileSlug, q.Overrides, q.Prompt)
        return  // Next item processed when this run finishes
    }
}
```

---

## 9. Operational Concerns

### 9.1 Configuration Options

Key settings from `RouterSettings`:

| Setting | Description |
|---------|-------------|
| `addr` | HTTP listen address |
| `idle-timeout-seconds` | Stop stream after N seconds without clients |
| `timeline-dsn` | SQLite connection string for durable timeline |
| `timeline-db` | Alternative: just the file path |
| `timeline-inmem-max-entities` | Size limit for in-memory store |
| `emit-planning-stubs` | Inject placeholder planning events for UI testing |

### 9.2 Debug Endpoints

When `PINOCCHIO_WEBCHAT_DEBUG=1`:

- `POST /debug/step/enable` — Enable step-by-step mode for a session
- `POST /debug/step/disable` — Disable step mode
- `POST /debug/continue` — Continue past a pause point

These endpoints provide manual control over the tool loop for debugging.

### 9.3 Redis vs In-Memory Events

The EventRouter can use either:
- **In-memory** (default) — Simpler, good for development and single-server deployment
- **Redis Streams** — Required for multi-server deployments; events persist across server restarts

---

## 10. Summary: What the System Guarantees

1. **Per-conversation serialization:** Only one run is active at a time; queued requests are processed in order.

2. **Event ordering:** SEM frames include sequence numbers; the frontend can reconstruct order.

3. **Durability (when enabled):** Timeline snapshots survive server restarts and browser refreshes.

4. **Profile isolation:** Different profiles can coexist, each with its own system prompt, tools, and middlewares.

5. **Clean separation:** HTTP/WS gateway, conversation runtime, and inference core operate independently.

If you modify the `Router`, `getOrCreateConv`, or `StreamCoordinator`, you are changing the core control plane. Proceed with care, write tests, and understand the invariants above.

---

## Appendix A: File Map

| File | Primary Responsibility |
|------|------------------------|
| `router.go` | Composition root, HTTP handlers |
| `conversation.go` | Conversation lifecycle, `getOrCreateConv` |
| `engine_from_req.go` | Request parsing and policy |
| `engine_config.go` | Config struct and signature generation |
| `engine_builder.go` | Engine/sink construction |
| `stream_coordinator.go` | Event subscription and SEM translation |
| `connection_pool.go` | WebSocket management |
| `sem_translator.go` | Event → SEM frame conversion |
| `timeline_projector.go` | SEM → timeline entity projection |
| `timeline_store_sqlite.go` | SQLite timeline implementation |
| `timeline_store_memory.go` | In-memory timeline implementation |
| `types.go` | Shared type definitions |

---

## Appendix B: Glossary

**Conversation** — A persistent chat thread identified by `conv_id`.

**Engine** — A Geppetto inference engine configured for a specific profile.

**EventRouter** — Watermill-based pub/sub system for internal event distribution.

**Hydration** — The process of restoring client state from the timeline after reconnection.

**Profile** — A named configuration bundle (system prompt, tools, middlewares).

**Run/Session** — A single execution context within a conversation.

**SEM** — Semantic Events protocol; the JSON wire format between backend and frontend.

**Timeline** — Durable storage of conversation state for hydration.

**Turn** — A single prompt-response pair within a conversation.
