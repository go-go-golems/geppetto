---
Title: Moments Webchat Router Migration to geppetto InferenceState/Session
Ticket: MO-004-UNIFY-INFERENCE-STATE
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
    - Path: geppetto/pkg/inference/core/session.go
      Note: Target shared runner session
    - Path: geppetto/pkg/inference/state/state.go
      Note: Target shared inference state
    - Path: moments/backend/pkg/webchat/conversation.go
      Note: Moments conversation state container to refactor
    - Path: moments/backend/pkg/webchat/router.go
      Note: Moments webchat router to migrate
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-20T00:00:00Z
WhatFor: Map moments webchat router state/loop to the new geppetto InferenceState + core.Session runner.
WhenToUse: Before refactoring moments webchat to use the shared inference core.
---


# Moments Webchat Router Migration to geppetto InferenceState/Session

## Executive Summary

`moments/backend/pkg/webchat` currently implements its own per-conversation inference state (`Conversation.RunID`, `Conversation.Turn`, `Conversation.Eng`, plus `running/cancel` bookkeeping), plus its own lifecycle manager (`ConvManager`) and engine composition path in `router.go`. The proposed MO-004 direction is to move the *inference-session core* into geppetto:

- `geppetto/pkg/inference/state.InferenceState` becomes the canonical state container.
- `geppetto/pkg/inference/core.Session` becomes the canonical runner (implements `Runner.RunInference(ctx, seed)`), handling:
  - single-pass inference and/or tool-loop inference via registry
  - event sink wiring
  - snapshot hook wiring
  - cancel/run lifecycle via `InferenceState.StartRun/CancelRun/FinishRun`

This document explains how moments webchat currently works (textbook-level), then lays out a concrete migration plan to adopt the shared geppetto state+runner without entangling webchat lifecycle (`ConvManager`, websockets, idle timers) into the inference core.

---

## Scope

In scope:
- `moments/backend/pkg/webchat/router.go` and `conversation.go` state/run loop wiring.
- Mapping moments’ prompt resolution + profile system prompt insertion to “seed turn construction” and middleware responsibilities.
- How to plug moments persistence and streaming into the new runner via event sinks + optional persister.

Out of scope:
- Refactoring moments’ `PromptResolver` itself.
- Rewriting moments’ websocket streaming (`startReader`, SEM conversion).

---

## Current moments webchat architecture

### Key files and what they do

- `moments/backend/pkg/webchat/router.go`
  - Builds a per-conversation engine + sink in the websocket handler.
  - Owns HTTP routes for `/chat`, `/ws`, debug endpoints.
  - Resolves the base profile prompt via `PromptResolver` and inserts a system block.

- `moments/backend/pkg/webchat/conversation.go`
  - Defines `Conversation` (state + websocket streams).
  - Defines `ConvManager` (map of live conversations).
  - Starts a per-conversation subscriber reader that broadcasts events to connected sockets.

### Data structures today

#### Conversation

`Conversation` is currently both:
- the **lifecycle container** (connections, readers, idle timers, ownership), and
- the **inference state container** (run id, engine, current turn, running/cancel).

From `moments/backend/pkg/webchat/conversation.go`:

- Identity/lifecycle:
  - `ID`, `ProfileSlug`, `EngConfigSig`
  - websocket connection set + locks
  - subscriber + read loop management
  - identity session + owner id + step controller

- Inference core (the part we want to replace with geppetto InferenceState):
  - `RunID string`
  - `Turn *turns.Turn`
  - `Eng engine.Engine`
  - `Sink events.EventSink`
  - `running bool` / `cancel context.CancelFunc` / `mu sync.Mutex`

#### ConvManager

`ConvManager` is a map-based lifecycle manager (thread-safe) with:
- `GetConversation(convID)`
- `FindConversationByRunID(runID)`

It is analogous to go-go-mento’s `ConversationManager`, but moments keeps a slimmer lifecycle implementation.

---

## Current inference event flow

### “Build engine and join conversation” (websocket handler)

In `moments/backend/pkg/webchat/router.go`, the websocket handler does:

1. Determine `conv_id`, `profile`, and optional `draft_bundle_id`.
2. Build `engine.Engine` + `events.EventSink` + `message.Subscriber`:
   - Sink uses `geppetto/pkg/inference/middleware.NewWatermillSink` (watermill publisher + topic)
   - Additional sink pipeline built via injected `SinkBuilder(profileSlug, watermillSink, r.cm)`
   - Engine built via `composeEngineFromSettings(...)` using profile middlewares
3. `getOrCreateConv(...)` stores the conversation with `conv.RunID` and `conv.Turn`.
4. Resolve the profile prompt slug via `PromptResolver` and insert the system prompt block:
   - `resolveProfilePrompt(...)` -> resolved text
   - `EnsureProfileSystemPromptBlock(conv.Turn, resolved)`
5. Start subscriber reader and attach websocket connection.

### “Run inference” (chat handler)

The chat handler (`handleChatRequest`) runs a loop that:
- appends the user prompt to `conv.Turn`
- invokes inference engine (`RunInference` or tool loop)
- uses event sinks to stream deltas
- stores the updated `conv.Turn` back into the conversation

(Details are spread across `router.go` and helper files; the key is that the state lives on `Conversation`.)

---

## Target architecture with geppetto InferenceState + Session

### Separation of concerns

We split the current `Conversation` responsibilities into:

1) **Lifecycle / transport (moments-owned, webchat package)**
- websocket connections
- subscriber reader loop
- idle timers / cleanup
- identity session + owner

2) **Inference-session core (geppetto-owned)**
- `InferenceState` (RunID, Turn, Eng, running/cancel)
- `Session` runner (RunInference method)
- tool loop orchestration
- event sinks + snapshot hook

### Proposed structure for moments Conversation

Instead of storing fields directly:

```go
RunID  string
Turn   *turns.Turn
Eng    engine.Engine
running bool
cancel context.CancelFunc
```

we store:

```go
Inference *state.InferenceState
Runner    core.Runner // typically *core.Session
Sink      events.EventSink // (optional; can also be attached via Session.EventSinks)
```

The runner is created once per conversation after engine composition.

---

## Where the pieces map (moments -> geppetto)

### 1) Running / cancellation

Today:
- `Conversation.running`, `Conversation.cancel`, guarded by `Conversation.mu`.

Target:
- `InferenceState.StartRun/FinishRun/CancelRun` is used instead.
- The UI/lifecycle layer calls `conv.Inference.CancelRun()`.

### 2) “Current Turn” storage

Today:
- `Conversation.Turn` is mutated and replaced.

Target:
- `InferenceState.Turn` is the canonical last turn.
- Call sites always read/write `conv.Inference.Turn`.

### 3) Engine storage

Today:
- `Conversation.Eng`.

Target:
- `InferenceState.Eng`.

### 4) Event sinks

Today:
- sink created at join time; run contexts are typically wrapped with `events.WithEventSinks`.

Target:
- `core.Session.EventSinks` is the single place we attach sinks to the run context.

### 5) Tool loop

Target options:

- If moments already uses `toolhelpers.RunToolCallingLoop` patterns, use geppetto `core.Session` with `Registry` and `ToolConfig`.
- If moments uses a custom loop (step mode, custom tool execution), either:
  - keep the moments loop temporarily, but still store state in `InferenceState`, or
  - extend `core.Session` to accept an injected “tool executor” contract later.

For MO-004, the most valuable first step is moving state/cancel/run semantics and event sink wiring. Tool loop unification can follow.

---

## Prompt resolution and system prompt insertion under the new model

### What happens today

- Moments resolves a base profile prompt slug (optionally with draft bundle) at *websocket join time*.
- It inserts a system block into the conversation’s Turn (`EnsureProfileSystemPromptBlock`).

### How this should look after migration

We keep the same high-level behavior:

- The router resolves prompt text and produces a seed Turn with the system prompt block.
- Then the conversation’s `InferenceState.Turn` is set to that Turn.

Important note:
- System prompt insertion should remain **idempotent** (middleware or helper ensures it is not duplicated) so persisting turns does not cause prompt growth.

---

## Migration plan (concrete steps)

### Step A: Introduce InferenceState into moments Conversation

- Replace fields:
  - `RunID`, `Turn`, `Eng`, `running`, `cancel`
- With:
  - `Inference *state.InferenceState`

Update code:
- `getOrCreateConv` should initialize `InferenceState` with:
  - `RunID`
  - seed `Turn` (with `RunID` set)
  - engine

### Step B: Use core.Session for inference execution

- Create a `core.Session` per conversation:
  - `State: conv.Inference`
  - `EventSinks: []events.EventSink{conv.Sink}` (or include additional sinks)
  - optionally `Registry + ToolConfig`
- Replace direct `conv.running/cancel` logic with `conv.Inference` methods.

### Step C: Persistence and “side effects”

- Moments currently uses event streaming; persistence may be separate.
- If moments has turn persistence, implement `core.TurnPersister` in moments and attach it to the Session.

### Step D: Keep ConvManager and websocket plumbing unchanged

- `ConvManager` remains a simple map of `Conversation` objects.
- Reader loops and websocket broadcast logic remain unchanged.

---

## Risks / gotchas

- **Concurrency:** moments currently serializes writes to websocket connections via `connsMu` but inference run state is guarded by `Conversation.mu`. When migrating to `InferenceState`, ensure only one inference run is active per conversation (`StartRun` enforces it).

- **Session correlation source of truth (update 2026-01-22 / GP-02):** `turns.Turn.RunID` no longer exists in Geppetto. Prefer a single long-lived `SessionID` owned by the conversation/session, and store it on `Turn.Metadata` via `turns.KeyTurnMetaSessionID` (legacy log/API name may remain `run_id`).

- **Tool loop differences:** if moments tool loop behavior differs from geppetto toolhelpers loop, don’t force unification in the first pass; focus on state/runner interface first.

---

## Diagrams

### Current moments (simplified)

```
WS join -> build engine/sink -> conversation{RunID, Turn, Eng, Sink}
                  |
                  v
Chat request -> mutate conv.Turn -> eng.RunInference(ctx, conv.Turn) -> conv.Turn = updated
                  |
                  v
Events -> sink -> watermill -> subscriber -> ws frames
```

### Target moments (simplified)

```
WS join -> build engine/sink -> conv.Inference = NewInferenceState(runID, seedTurn, eng)
                               conv.Runner   = &core.Session{State: conv.Inference, EventSinks: [conv.Sink], ...}

Chat request -> seed := conv.Inference.Turn (+ append user block)
             -> conv.Runner.RunInference(ctx, seed)
             -> conv.Inference.Turn updated

Events -> Session.EventSinks -> watermill -> subscriber -> ws frames
```

---

## Source references

- `moments/backend/pkg/webchat/router.go`
- `moments/backend/pkg/webchat/conversation.go`
- `geppetto/pkg/inference/state/state.go`
- `geppetto/pkg/inference/core/session.go`
