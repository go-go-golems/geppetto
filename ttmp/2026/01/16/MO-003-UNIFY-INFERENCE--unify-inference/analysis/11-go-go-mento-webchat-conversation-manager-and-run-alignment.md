---
Title: Go-go-mento Webchat Conversation Manager and Run Alignment
Ticket: MO-003-UNIFY-INFERENCE
Status: active
Topics:
    - inference
    - architecture
    - webchat
    - prompts
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-mento/go/pkg/webchat/conversation.go
      Note: Conversation container and context helpers
    - Path: go-go-mento/go/pkg/webchat/conversation_manager.go
      Note: Conversation lifecycle and engine recomposition
    - Path: go-go-mento/go/pkg/webchat/engine_builder.go
      Note: Engine + sink composition
    - Path: go-go-mento/go/pkg/webchat/inference_state.go
      Note: RunID + current Turn storage
    - Path: go-go-mento/go/pkg/webchat/loops.go
      Note: ToolCallingLoop and step mode
    - Path: go-go-mento/go/pkg/webchat/stream_coordinator.go
      Note: Event streaming and SEM frame translation
    - Path: go-go-mento/go/pkg/webchat/turns_loader.go
      Note: Load persisted turns to seed state
    - Path: go-go-mento/go/pkg/webchat/turns_persistence.go
      Note: Persist finalized turns
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-16T19:07:03-05:00
WhatFor: Understand go-go-mento webchat architecture and align it with Run-centric conversation state.
WhenToUse: When deciding how to unify pinocchio/moments around the go-go-mento webchat model.
---


# Go-go-mento Webchat Conversation Manager and Run Alignment

## Executive Summary

Go-go-mento’s webchat stack already operates in a *turn-centric* way: each active conversation stores a single `turns.Turn` as the current snapshot (`InferenceState.Turn`) and persists finalized turns to a database. The `ConversationManager` is the lifecycle coordinator: it owns conversation lookup/creation, engine recomposition, websocket connection pools, and streaming hydration.

This aligns well with the new “Run‑centric” proposal if we treat `turns.Run` as the canonical container for *turn history* and use the current `InferenceState.Turn` as the “last turn.” Go-go-mento already does most of this, but the run concept is implicit: it uses `RunID` fields and a persisted turns table instead of an explicit in‑memory `Run` object. The unification work is therefore not a full rewrite; it’s a formalization: name the run container, formalize append semantics, and make the runner interface match the webchat flow.

---

## Architectural Map (go-go-mento webchat)

**Primary components and responsibilities**

- **Router** (`go-go-mento/go/pkg/webchat/router.go`)
  - HTTP + websocket entry points
  - Dispatches `ChatHandler` requests
  - Orchestrates run loop (ToolCallingLoop)
  - Bridges between web sockets and inference engine

- **ConversationManager** (`go-go-mento/go/pkg/webchat/conversation_manager.go`)
  - Lifecycle of conversations (create/reuse/evict)
  - Engine composition via `EngineBuilder`
  - Stream coordinator management
  - Connection pool management
  - Optional turn loading via `TurnsLoader`

- **Conversation** (`go-go-mento/go/pkg/webchat/conversation.go`)
  - Holds per-conversation state, including
    - `InferenceState` (current turn, run, engine, running flag)
    - Identity context
    - Event sink
    - Connection pool and stream coordinator
    - Profile + engine config signature
    - Step controller

- **InferenceState** (`go-go-mento/go/pkg/webchat/inference_state.go`)
  - The core “current state” container
  - Holds `RunID`, `Turn *turns.Turn`, `Eng engine.Engine`
  - Tracks running + cancel handles

- **EngineBuilder** (`go-go-mento/go/pkg/webchat/engine_builder.go`)
  - Builds `engine.Engine` + `events.EventSink` from profiles
  - Composes middleware/tool pipeline
  - Injects `ConversationManager` for extractor wrappers

- **ToolCallingLoop** (`go-go-mento/go/pkg/webchat/loops.go`)
  - Orchestrates inference + tool calls
  - Handles step mode pauses
  - Persists final turn via router hook

- **Persistence** (`go-go-mento/go/pkg/webchat/turns_loader.go`, `turns_persistence.go`)
  - Loads last persisted turn to seed inference state on resume
  - Persists finalized turns (blocks + metadata)

- **Streaming Layer**
  - `StreamCoordinator` (`stream_coordinator.go`) pulls events from Redis and translates to SEM frames
  - `EventTranslator` creates per-event UI frames
  - `ConnectionPool` fans frames out to websocket clients

---

## Conversation Lifecycle in go-go-mento

### 1) Router bootstraps the system

The router initializes core registries, then constructs `ConversationManager`:

```
Router.NewRouter()
 ├─> EngineBuilder (profiles + middleware/tool registry)
 └─> ConversationManager (Builder, Redis subscriber factory, timeline loader, turn loader)
```

**Key code:**
- `go-go-mento/go/pkg/webchat/router.go` `NewRouter(...)`
- `go-go-mento/go/pkg/webchat/engine_builder.go`

### 2) ConversationManager owns create/reuse

`ConversationManager.GetOrCreate`:

- Builds engine config signature for requested profile + overrides.
- Reuses existing conversation if signature matches.
- If changed, recomposes engine + sink, reattaches stream coordinator.
- Ensures timeline hydration + loads persisted turns.
- Initializes `InferenceState` if missing.

**Key code:**
- `go-go-mento/go/pkg/webchat/conversation_manager.go` `GetOrCreate(...)`

### 3) The conversation itself is “current turn” + run metadata

`Conversation` embeds `*InferenceState` which holds:

```go
RunID string
Turn  *turns.Turn
Eng   engine.Engine
```

This is already a “run + last turn” model; the history is not kept in memory, only the most recent Turn.

### 4) Turn persistence is explicit, not implicit

- On run completion, `ToolCallingLoop` asks the router to persist the final turn.
- Persistence uses a turn accumulator so tool snapshots can be persisted accurately.
- DB persistence stores blocks + metadata; later, `turns_loader` reconstructs the latest turn.

**Key code:**
- `go-go-mento/go/pkg/webchat/turns_persistence.go`
- `go-go-mento/go/pkg/webchat/turns_loader.go`

---

## Run Loop Anatomy (go-go-mento)

### Core run flow

1. `ChatHandler` receives a prompt.
2. `ConversationManager.GetOrCreate` returns a Conversation.
3. Router sets up tool registry and metadata on `conv.Turn.Data`.
4. A goroutine starts `ToolCallingLoop`.
5. Loop executes inference, tool calls, pauses (step mode), and returns a final `Turn`.
6. Router updates `conv.Turn` with returned `Turn`.
7. Final turn is persisted (if DB available).

### Pseudocode (simplified)

```go
conv := conversationManager.GetOrCreate(...)
conv.Turn.Data[ToolRegistry] = registry

runCtx := WithConversation(WithRouter(ctx, r), conv)
runCtx = events.WithEventSinks(runCtx, conv.Sink)

updated, err := ToolCallingLoop(runCtx, conv.Eng, conv.Turn, registry, opts)
if updated != nil { conv.Turn = updated }
```

### ToolCallingLoop structure

```
for i in range(maxIterations):
  updated = eng.RunInference(ctx, currentTurn)
  calls = ExtractPendingToolCalls(updated)
  if no calls:
     persist(updated)
     return updated
  results = ExecuteToolCalls(calls)
  AppendToolResults(updated, results)
  currentTurn = updated
```

**Step mode** is injected after inference and after tool execution. It pauses the loop and emits a debugger pause event.

---

## Streaming / Websocket Flow

**Event flow:**

```
Engine -> WatermillSink -> Redis stream -> StreamCoordinator -> EventTranslator -> ConnectionPool -> WebSocket
```

**Key types:**

- `WatermillSink` builds event stream (from geppetto middleware)
- `StreamCoordinator` consumes Redis topic `chat:<conv-id>`
- `EventTranslator` converts events into SEM frames
- `ConnectionPool` fans out frames to active sockets

This architecture allows the inference loop to be decoupled from websocket streaming, but still supports immediate event propagation and timeline hydration.

---

## Conversation State Management (go-go-mento)

**In-memory:**
- Only the last `turns.Turn` is stored in `InferenceState.Turn`.
- `RunID` is stored both on `InferenceState.RunID` and (often) `Turn.RunID`.

**Persisted:**
- Turns are persisted to DB after completion.
- Loading only reconstructs the latest turn, not the full history.

This is equivalent to a **Run container where only `Run.Last()` is in memory**, with full history living in storage.

---

## Alignment with Run-Centric Model

### What already aligns

- **Turn‑centric core**: inference always runs against a `*turns.Turn`.
- **Run identity**: `RunID` is already threaded through turns and event metadata.
- **Append semantics**: final turns are appended (persisted) explicitly.

### What differs

- **No explicit `turns.Run` object**: run history is implicit via DB, not in memory.
- **No explicit append API**: persistence is performed in router/loop, not on a run object.
- **Current Turn is “source of truth”**: this mirrors a Run container with only the last turn in memory.

### How to unify

**Option A: Use `turns.Run` as a thin wrapper**

- Store `turns.Run` on `Conversation` (or `InferenceState`) instead of `Turn`.
- `turns.Run.Turns[len-1]` becomes the current snapshot.
- Persistence still happens at the end of a loop (but now append comes from run).

**Option B: Keep current model but formalize the Run contract**

- Continue storing only the latest `Turn`.
- Treat DB as the canonical run history.
- Create helper functions:
  - `RunAddUserPrompt(runID, *Turn, prompt)`
  - `RunAppendPersisted(runID, *Turn)`

Both options are consistent with the webchat flow; Option A gives a clearer type boundary that aligns with our new “Run‑centric” design.

---

## How go-go-mento’s Webchat Structure Guides Unification

### Core principles we should adopt

- **ConversationManager as the lifecycle gate**
  - Centralizes creation, reuse, and eviction.
  - Allows engine recomposition when profile/overrides change.

- **Single “current turn” in memory**
  - Avoids unbounded memory growth.
  - Uses persistence for history and replay.

- **Explicit streaming pipeline**
  - Events are pushed to Redis and translated into UI frames in one place.
  - Keeps inference loop independent of websocket details.

- **EngineBuilder as a stable seam**
  - Profiles -> middlewares -> engine is deterministic.
  - Router stays lean and does not embed composition logic.

### Mapping to the Run-centric design

```
Run (turns.Run)
  - Run.ID == RunID
  - Run.Turns[-1] == current snapshot
  - Run.Append(newTurn) -> persist

ConversationManager
  - owns Run
  - owns Engine + Sink
  - owns streaming + connection pool

Runner
  - Run(ctx, turn) -> turn
```

This is the same conceptual flow as go-go-mento today, with the addition of an explicit Run container instead of implicit DB history.

---

## Suggested Unification Steps (pinocchio + moments)

1. **Adopt the ConversationManager pattern** in pinocchio webchat:
   - Move connection pool + stream coordinator into a manager object.
   - Centralize engine recomposition when profile overrides change.

2. **Use `turns.Run` as the state holder**
   - Replace ad‑hoc conversation state with a Run container.
   - Keep only latest turn in memory; persist history as needed.

3. **Unify runner interface**
   - `Run(ctx, seed *turns.Turn) (*turns.Turn, error)` as the only inference contract.

4. **Move prompt resolution upstream**
   - For moments, resolve prompt text before adding to the current turn.

---

## Key Source References

- `go-go-mento/go/pkg/webchat/conversation_manager.go`
- `go-go-mento/go/pkg/webchat/conversation.go`
- `go-go-mento/go/pkg/webchat/inference_state.go`
- `go-go-mento/go/pkg/webchat/loops.go`
- `go-go-mento/go/pkg/webchat/turns_loader.go`
- `go-go-mento/go/pkg/webchat/turns_persistence.go`
- `go-go-mento/go/pkg/webchat/engine_builder.go`
- `go-go-mento/go/pkg/webchat/stream_coordinator.go`
- `go-go-mento/go/pkg/webchat/connection_pool.go`
