---
Title: 'Textbook: The New Webchat Router'
Ticket: PI-007-WEBCHAT-BACKEND-REFACTOR
Status: complete
Topics:
    - webchat
    - backend
    - bugfix
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/router.go
      Note: Primary router anatomy described
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Stream sequencing and cursor logic
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Timeline projection described
ExternalSources: []
Summary: A deep, Norvig-style walkthrough of the refactored webchat router, its lifecycle, and its surrounding architecture.
LastUpdated: 2026-02-04T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# The New Webchat Router: A Norvig-Style Textbook

This chapter teaches the new webchat router as if we were reading a carefully annotated system design, not a pile of source files. We move from “what” to “why” to “how,” and we dwell on the subtle interfaces where correctness and clarity are won or lost. You’ll find clear symbols, worked examples, concise pseudocode, and small exercises. The aim is to let you read the code with a mental model that is both accurate and delightful.

---

## 1. The Big Picture: A Router as a Coordination Machine

A webchat server is not a single function. It is a coordinated dance between three kinds of “time”: request time (HTTP), streaming time (WebSocket frames), and durability time (timeline storage for hydration). The refactored router is the conductor of that dance.

The goal of the refactor was to **separate concerns** cleanly and make the boundaries visible: *UI vs API*, *conversation lifecycle vs request lifecycle*, and *stream ordering vs persistence*. What you get is a system that is easier to embed, easier to reason about, and much harder to accidentally break.

**Core idea**: the router is not “the app.” It is the *transport* between the app and the outside world, with a few key invariants:

- Every conversation has one place where it is created and owned (`ConvManager`).
- Every stream has a monotonic ordering (`event.seq`) suitable for hydration and merging.
- Every handler can be mounted under a prefix without rewriting paths.

---

## 2. A Map of the System (Symbols First)

We use a small dictionary of symbols to make the prose precise:

- **`Router`**: HTTP + WS front door, owns mux and handler registration.
- **`Server`**: The wrapper that runs the router and the event loop.
- **`ConvManager`** (`cm`): Owns conversation creation, eviction, and lifecycle.
- **`Conversation`**: Holds engine, stream coordinator, connection pool, and timeline projector.
- **`StreamCoordinator`**: Consumes events → emits SEM frames with ordering.
- **`ConnectionPool`**: Manages WebSocket clients; broadcasts are non-blocking.
- **`TimelineProjector`**: Applies SEM frames to durable timeline storage.
- **`TimelineStore`**: SQLite or in-memory projection store; uses explicit versions.

The new structure makes these separations explicit; the most important learning step is to see where *responsibility* lives, not just where *code* lives.

---

## 3. Callout: Fundamental Invariant—Monotonic Ordering

> **Fundamental**: The system relies on a single monotonic sequence `event.seq` to order timeline entities across live streaming and hydration.

This is the root of a surprisingly large set of design choices. If you understand this, you understand the fix that resolved the hydration ordering bug.

### Why it matters

- Hydration orders entities by `version`, and `version` comes from `event.seq`.
- If `seq` for assistant events is tiny but user messages are time-based, hydration will place user messages last.
- Therefore, **every producer of timeline data must use the same scale**.

The refactor enforces this by requiring explicit versions on every timeline upsert and by ensuring the stream coordinator derives a time-based fallback sequence when Redis stream IDs are missing.

---

## 4. The Router in Action: End-to-End Flow

Let’s follow a request from user prompt to UI update:

```
User submits prompt
   │
   ▼
POST /chat ─────► Router.handleChatRequest
   │               ├─ BuildEngineFromReq
   │               ├─ ConvManager.GetOrCreate
   │               ├─ PrepareRun
   │               └─ startRunForPrompt
   │
   ▼
Engine emits events → StreamCoordinator
   │                    ├─ derive StreamCursor (seq + stream_id)
   │                    ├─ SemanticEventsFromEventWithCursor
   │                    └─ onFrame callbacks
   │
   ▼
ConnectionPool broadcasts frames to WS clients
   │
   ▼
TimelineProjector consumes SEM frames and stores snapshots
```

Now the same, but for hydration:

```
Browser reloads page
   │
   ▼
GET /timeline?conv_id=... ───► Router.timelineHandler
   │                             └─ TimelineStore.GetSnapshot
   ▼
Browser merges hydrated entities by version
```

Notice how **both streaming and hydration are pinned to the same version scale**.

---

## 5. The Router as a Physical Artifact (Key Files)

Here is a concrete list of the files you will read in order:

1. `pinocchio/pkg/webchat/router.go`
2. `pinocchio/pkg/webchat/server.go`
3. `pinocchio/pkg/webchat/conversation.go`
4. `pinocchio/pkg/webchat/stream_coordinator.go`
5. `pinocchio/pkg/webchat/connection_pool.go`
6. `pinocchio/pkg/webchat/timeline_projector.go`
7. `pinocchio/pkg/webchat/timeline_store.go`

If you want to learn the new router, read them exactly in this order. That order mirrors the flow of responsibility from outside-in.

---

## 6. The Router’s Anatomy

### 6.1 Constructor and Handlers

The router’s constructor wires its world, but it does not start it. That’s intentional.

**Symbols to note**:

- `NewRouter(ctx, parsed, staticFS)`
- `registerUIHandlers`
- `registerAPIHandlers`
- `APIHandler()` and `UIHandler()`

**Fundamental**: the router now has explicit UI and API handler registries. That means you can mount these parts independently, embed the API in another service, or serve UI elsewhere.

### 6.2 Mounting Under a Prefix

The refactor introduced a true mount function:

```go
func (r *Router) Mount(mux *http.ServeMux, prefix string) {
    prefix = strings.TrimRight(prefix, "/")
    mux.Handle(prefix+"/", http.StripPrefix(prefix, r.mux))
    mux.HandleFunc(prefix, func(w http.ResponseWriter, r0 *http.Request) {
        http.Redirect(w, r0, prefix+"/", http.StatusPermanentRedirect)
    })
}
```

**Why it matters**: prefix mounting now *preserves internal paths* without rewriting every handler. It also handles the “missing trailing slash” case gracefully.

---

## 7. A Norvig-Style Algorithmic View (Pseudocode)

### 7.1 Handling a Chat Request

```pseudo
function handleChatRequest(req):
    input, body ← BuildEngineFromReq(req)
    conv ← ConvManager.GetOrCreate(input.conv_id, input.profile, overrides)

    prep ← conv.PrepareRun(idempotency_key, profile, overrides, body.prompt)
    if prep.Start == false:
        return prep.Response

    resp ← startRunForPrompt(conv, profile, overrides, body.prompt)
    return resp
```

**Observe**: This makes queueing explicit. The router is now the orchestrator; queue policy lives in the conversation.

### 7.2 Stream Consumption and Sequencing

```pseudo
function consume():
    for msg in subscriber:
        event ← decode(msg)
        seq ← derive seq from stream_id OR time-based monotonic fallback
        cursor ← { stream_id, seq }
        for frame in SemanticEventsFromEventWithCursor(event, cursor):
            onFrame(event, cursor, frame)
        ack(msg)
```

**Key design**: `seq` is used by the timeline projector and by the UI for ordering. If `seq` is ever out of scale, everything breaks downstream.

---

## 8. The Timeline: Durable Truth

### 8.1 Explicit Versions Only

Timeline upserts require `version` explicitly. This decision eliminates invisible auto-increment logic and aligns all timeline updates with the stream sequence.

**Signature**:

```go
Upsert(ctx, convID, version, entity) error
```

### 8.2 Timeline Projector

The projector reads SEM frames and applies them to the store. It expects `event.seq` on every frame. If `seq` is absent or zero, the frame is ignored (by design).

This is not a convenience choice; it’s a *safety property*. The timeline store must never receive an unordered version.

---

## 9. Callout: Concurrency and Backpressure

> **Fundamental**: WebSocket writes must never block the stream coordinator.

The refactor moved `ConnectionPool` to a non-blocking model: each client has a buffered channel and a writer goroutine. A slow client does not stall the entire conversation; it is dropped when its buffer overflows.

This is a design that favors **overall system responsiveness** over the guarantee that every client receives every frame. That is the correct choice for chat UIs where reconnection + hydration is the intended recovery path.

---

## 10. Exercises (Short, Concrete)

**Exercise 1**: Trace the lifecycle for a brand-new conversation.
- Which functions are called in order from `POST /chat` to the first `llm.delta` frame?
- Which object creates the StreamCoordinator?

**Exercise 2**: Find the monotonic sequencing logic.
- Identify where `event.seq` is derived.
- Explain how the fallback path avoids decreasing sequences.

**Exercise 3**: Identify where timeline hydration is defined.
- Where does `/timeline` read from?
- Which component owns the on-disk DB path?

---

## 11. Quiz (With Answers)

**Q1**: Why is `TimelineStore.Upsert` required to take a version?

**A**: Because the ordering needs to match the stream sequence (`event.seq`). If the store auto-incremented, versions could diverge from stream order, causing hydration inconsistencies.

**Q2**: What happens if Redis stream metadata is missing?

**A**: The StreamCoordinator uses a time-based monotonic sequence (`UnixMillis * 1_000_000`) to keep ordering consistent with user-message inserts.

**Q3**: Why separate `APIHandler()` and `UIHandler()`?

**A**: To allow embedding or serving UI assets separately from the API/WS endpoints, and to make root prefix mounting predictable.

---

## 12. Diagram: New Router Structure

```
┌───────────────┐
│   Router      │
│  (mux + API)  │
└──────┬────────┘
       │ UIHandler() / APIHandler()
       │
       ▼
┌───────────────────────────────┐
│ ConversationManager (cm)       │
│ - GetOrCreate                  │
│ - Eviction loop                │
└──────┬────────────────────────┘
       │ owns
       ▼
┌───────────────────────────────┐
│ Conversation                   │
│ - StreamCoordinator            │
│ - ConnectionPool               │
│ - TimelineProjector            │
└──────┬────────────────────────┘
       │ stream events → seq
       ▼
┌───────────────────────────────┐
│ TimelineStore (SQLite/mem)     │
│ - Upsert(version, entity)      │
│ - GetSnapshot                  │
└───────────────────────────────┘
```

---

## 13. A Final Mental Model

Think of the router as **a traffic cop** that guarantees ordering and separation of concerns rather than business logic. Conversations live in a manager; streams live in a coordinator; persistence is explicit. If you keep these boundaries in mind, the code reads like a well-factored textbook rather than a state machine.

And, as Norvig would say, *clarity is not a luxury; it is a compiler for your future self.*

