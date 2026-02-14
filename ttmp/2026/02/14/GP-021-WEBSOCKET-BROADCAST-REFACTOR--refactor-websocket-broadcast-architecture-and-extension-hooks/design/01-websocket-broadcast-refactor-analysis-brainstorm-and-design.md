---
Title: 'WebSocket Broadcast Refactor: Analysis, Brainstorm, and Design'
Ticket: GP-021-WEBSOCKET-BROADCAST-REFACTOR
Status: active
Topics:
    - backend
    - websocket
    - architecture
    - events
    - webchat
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/context.go
      Note: event sink context publish path
    - Path: geppetto/pkg/inference/middleware/sink_watermill.go
      Note: watermill sink publishes events to router topic
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: attaches EventSinks to inference run context
    - Path: geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/sources/ws-broadcast-paths.txt
      Note: experiment output for broadcast call graph
    - Path: geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/sources/ws-hookability-audit.txt
      Note: experiment output for extension seam audit
    - Path: geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/sources/ws-protocol-surface.txt
      Note: experiment output for protocol and event-type inventory
    - Path: pinocchio/pkg/webchat/connection_pool.go
      Note: transport fanout primitive and backpressure
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: SEM stream callback and broadcast orchestration
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: builds conv sink to topic chat:<conv_id>
    - Path: pinocchio/pkg/webchat/router.go
      Note: /ws handler and control frame behavior
    - Path: pinocchio/pkg/webchat/router_options.go
      Note: existing option-based extension surface
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: event subscription and SEM frame generation
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: persisted projection path and upsert callback
    - Path: pinocchio/pkg/webchat/timeline_registry.go
      Note: custom timeline handler extension seam
    - Path: pinocchio/pkg/webchat/timeline_upsert.go
      Note: timeline upsert websocket emission
ExternalSources: []
Summary: Deep analysis and refactor design for websocket broadcast architecture, connection profiles, and server-side extension hooks without direct ConnectionPool access.
LastUpdated: 2026-02-14T17:30:00-05:00
WhatFor: Guide websocket refactor work to support per-connection filtering/profiles and pluggable backend event producers.
WhenToUse: Use when implementing websocket routing/fanout changes, subscription semantics, and extension APIs.
---



# WebSocket Broadcast Refactor: Analysis, Brainstorm, and Design

## 1. Objective
This document analyzes the current websocket broadcast architecture in `pinocchio/pkg/webchat` and proposes a refactor plan that enables:

1. Per-connection websocket profiles and filtering controls.
2. Pluggable backend-side event producers that can publish websocket frames without directly depending on `ConnectionPool` internals.
3. Clear layering between event ingestion, semantic frame translation, projection upserts, and client fanout.

The design intentionally treats this as an architectural refactor ticket, not an incremental bugfix. The goal is to leave the system with explicit extension seams and predictable broadcast semantics.

## 2. Current Architecture (What Exists Today)

### 2.1 `/ws` lifecycle and join flow
The websocket endpoint is implemented in `router.go` (`/ws` handler). It currently reads only:

1. `conv_id`
2. `profile`

The request is normalized through `BuildEngineFromReq`, then the conversation is obtained via `GetOrCreate`. The connection is added to `ConvManager`/`ConnectionPool`. On connect, a targeted `ws.hello` message is sent via `SendToOne`. The read loop handles ping/pong and then idles until disconnect.

Key consequence: connection capabilities are currently implicit and global to conversation membership; no per-connection subscription state is tracked.

### 2.2 How broadcast frames are produced
There are two active broadcast producers:

1. SEM stream fanout from `StreamCoordinator` callback path in `conversation.go`.
   - Source: event router subscriber -> `StreamCoordinator.consume`
   - Transform: `SemanticEventsFromEventWithCursor`
   - Fanout: `conv.pool.Broadcast(frame)`

2. Projection upsert fanout in `timeline_upsert.go`.
   - Source: timeline projector persisted upsert callback
   - Payload: SEM `timeline.upsert` envelope (`TimelineUpsertV1` payload)
   - Fanout: `conv.pool.Broadcast(...)`

Targeted per-connection sends are used only for control frames (`ws.hello`, `ws.pong`) via `SendToOne`.

### 2.3 Where the fanout primitive lives
`ConnectionPool` owns client channels, backpressure behavior, write deadlines, and drop-on-full policy. Its API is intentionally small:

1. `Broadcast([]byte)`
2. `SendToOne(conn, []byte)`
3. connection add/remove/lifecycle

It does not maintain connection metadata beyond the `wsConn` handle and send buffer.

### 2.4 Current extension seams
The code already has some extension hooks, but they are asymmetrical:

1. `WithTimelineUpsertHook`: lets callers customize timeline-upsert behavior.
2. `RegisterTimelineHandler`: allows custom projector handlers for specific SEM event types.
3. `WithEventSinkWrapper`, `WithBuildSubscriber`, `EngineFromReqBuilder`: allow dependency injection around event ingestion and request policy.

Missing seam: there is no first-class “websocket broadcaster” interface exposed to backend modules. Publishing non-projector custom websocket events generally requires either:

1. touching conversation internals and pool directly, or
2. piggybacking on existing event translator/projector chains.

### 2.5 Debug UI coupling status
Current frontend websocket consumer is the `webchat` app store `wsManager`; debug-ui currently relies on debug APIs. Nevertheless, the same `/ws` endpoint and SEM framing are relevant to future debug attach/follow mode.

### 2.6 End-to-end data path: inference/middleware -> websocket
This is the exact runtime chain today:

1. Engine + middlewares emit typed events via `events.PublishEventToContext(...)`.
2. `toolloop/enginebuilder.Builder` attaches event sinks to run context (`events.WithEventSinks`).
3. Webchat conversation builder provides `conv.Sink` in `EventSinks`.
4. `conv.Sink` is a `WatermillSink` to topic `chat:<conv_id>`.
5. `StreamCoordinator` subscribes to `chat:<conv_id>` and reads messages.
6. For each message:
   - decode event (`events.NewEventFromJson`)
   - translate to SEM envelopes (`SemanticEventsFromEventWithCursor`)
   - invoke `onFrame` callback for each envelope
7. `onFrame` callback (wired by `Conversation`) performs three side effects:
   - `conv.pool.Broadcast(frame)` (client fanout)
   - `conv.semBuf.Add(frame)` (debug event buffer)
   - `conv.timelineProj.ApplySemFrame(...)` (projection persistence path)
8. `TimelineProjector.ApplySemFrame` upserts projection entities into timeline store.
9. Projector `onUpsert` callback emits `timeline.upsert` websocket SEM frame.

This is why websocket fanout is currently tightly coupled to conversation callback wiring.

### 2.7 `onFrame` and `timelineProj` semantics
Definitions in current code:

1. `onFrame`: callback owned by `Conversation` and injected into `StreamCoordinator`.
   - Signature: `(events.Event, StreamCursor, []byte)`.
   - Responsibility today: orchestrate fanout + buffering + projection.
2. `timelineProj`: per-conversation `*TimelineProjector`.
   - Consumes SEM frames, maps known event types to `TimelineEntityV1`, persists them.
   - Can call `onUpsert` hook to emit websocket `timeline.upsert`.

`StreamCoordinator` itself does not know `ConnectionPool` or projector internals; it delegates via callbacks.

## 3. Experimental Findings
Three reproducible scripts were added under `scripts/` and outputs are in `sources/`:

1. `scripts/01-trace-ws-broadcast-paths.sh`
2. `scripts/02-inventory-ws-protocol-surface.sh`
3. `scripts/03-hookability-audit.sh`

### 3.1 Broadcast callsite inventory
From `sources/ws-broadcast-paths.txt`:

1. Broadcast callsites in production code are concentrated in `conversation.go` and `timeline_upsert.go`.
2. `/ws` handler itself uses targeted `SendToOne` for hello/pong.
3. Broadcast graph is simple but hardcoded.

This concentration is good for refactorability, but it means all behavior changes currently require editing core `webchat` files.

### 3.2 Protocol surface inventory
From `sources/ws-protocol-surface.txt`:

1. `/ws` query params parsed today: `conv_id`, `profile`.
2. Translator emits a broad SEM type set: `log`, `llm.*`, `tool.*`, `agent.mode`, `debugger.pause`, `thinking.mode.*`.
3. Timeline projection updates are additionally emitted as `timeline.upsert`.

Implication: the protocol is already rich enough to support profile-based filtering without inventing new event taxonomy.

### 3.3 Hookability audit
From `sources/ws-hookability-audit.txt`:

1. Strong hook seams exist around timeline projection and subscriber/router construction.
2. No generalized websocket publisher registry or broker abstraction exists.
3. Direct `ConnectionPool` interaction is the dominant way to reach sockets.

Implication: extension ergonomics are currently inversion-of-control in some layers, but not in fanout layer.

## 4. Key Architectural Problems

### Problem A: Broadcast path is not policy-aware per connection
`Broadcast` sends identical frames to all clients for a conversation. There is no built-in support for:

1. “debug” vs “chat” subscription profiles
2. event category filtering
3. opt-in high-volume channels (e.g. turn snapshots)

### Problem B: Backend producers must know too much
To publish websocket events today, new code tends to require access to conversation/pool internals or must be wedged into existing translator/projector paths.

### Problem C: No explicit subscription contract
The protocol has no declared subscription model beyond “joined conversation”. This blocks clean rollout of optional channels (debug-only streams, dense payloads, etc.).

### Problem D: Cross-layer concerns are mixed
`conversation.go` callback currently does all of the following in one closure:

1. client fanout
2. sem buffering for debug APIs
3. timeline projector application

The closure works, but it is doing orchestration work that should be captured as named pipeline steps.

## 5. Design Goals and Non-Goals

### 5.1 Goals
1. Introduce explicit websocket subscription/profile semantics at connection time.
2. Decouple event production from direct `ConnectionPool` usage.
3. Preserve existing default behavior for current clients (at least during migration).
4. Keep conversation-level isolation (still scoped by `conv_id`).
5. Maintain write-path safety (bounded buffers and drop policy).

### 5.2 Non-goals
1. Replacing SEM payload format.
2. Replacing watermill/event router ingestion.
3. Full transport rewrite (e.g., gRPC streaming).
4. Immediate rollout of every potential debug stream.

## 6. Brainstormed Refactor Options

### Option 1: Thin wrapper around `ConnectionPool`
Create a `ConversationBroadcaster` interface and implement it over existing pool. Add profile filtering outside the pool.

Pros:
1. Low-risk migration.
2. Minimal touching of low-level write code.

Cons:
1. Filtering logic may become fragmented if done ad hoc.
2. Harder to reason about per-connection state lifecycle.

### Option 2: Metadata-aware pool (stateful clients)
Extend pool client state with subscription/profile metadata and dispatch decisions inside pool methods.

Pros:
1. Single place for fanout policy.
2. Efficient filtering during iteration.

Cons:
1. `ConnectionPool` becomes policy-heavy.
2. Harder to keep pool as simple transport primitive.

### Option 3: Broker + transport split (recommended)
Add an intermediate conversation-scoped broker that owns:

1. subscription registry
2. event classification
3. profile filtering
4. dispatch to transport adapter (`ConnectionPool`)

`ConnectionPool` remains transport-only. Producers publish into broker, not pool.

Pros:
1. Clean layering.
2. Best extensibility for backend modules.
3. Easier testing of policy without real sockets.

Cons:
1. Larger upfront refactor.
2. Requires careful migration wiring.

Recommendation: Option 3.

## 7. Proposed Target Architecture

### 7.1 New core interfaces

```go
type WSFrame struct {
    ConvID    string
    Type      string
    ID        string
    Seq       uint64
    StreamID  string
    Payload   []byte // JSON envelope (current sem format)
    Channel   string // e.g. "sem", "timeline", "debug.turn_snapshot"
    Priority  string // normal|high|low (optional)
}

type WSSubscription struct {
    ConnectionID string
    ConvID       string
    Profile      string            // e.g. "chat", "debug-lite", "debug-full"
    Channels     map[string]bool   // explicit channel opts
    Filters      map[string]string // optional key/value filters
}

type ConversationWSPublisher interface {
    Publish(ctx context.Context, frame WSFrame) error
}

type ConversationWSBroker interface {
    Register(sub WSSubscription, conn wsConn) error
    Unregister(connectionID string)
    Publish(ctx context.Context, frame WSFrame) error
}
```

### 7.2 Layering
1. Producers (stream callback, timeline upsert hook, future debug hooks) call `Publish` on broker.
2. Broker decides eligible subscriptions.
3. Broker dispatches bytes to transport adapter (pool abstraction).
4. Pool remains responsible for channel buffering, deadlines, and socket closure.

### 7.3 Connection profile model
At `/ws` connect, allow query params like:

1. `ws_profile=chat|debug-lite|debug-full`
2. `channels=sem,timeline,debug.turn_snapshot`
3. `filter_types=timeline.upsert,llm.final` (optional advanced override)

Default if omitted: current behavior (`chat` profile with existing channels).

### 7.4 Event classification strategy
Map outgoing frames to channels using deterministic rules:

1. `timeline.upsert` -> `timeline`
2. `ws.hello/ws.pong` -> `control`
3. `llm.*`, `tool.*`, `log`, etc. -> `sem`
4. future `turn.snapshot` -> `debug.turn_snapshot`

Classification can happen once near producer boundary or centrally in broker.

### 7.5 Backend hook strategy (no ConnectionPool dependency)
Introduce router-level registration for websocket emitters:

```go
type WSEmitterFactory func(conv *Conversation, pub ConversationWSPublisher) error

func WithWSEmitterFactory(f WSEmitterFactory) RouterOption
```

Flow:
1. Conversation created.
2. Broker/publisher bound.
3. Registered emitter factories initialized.
4. Emitters publish via interface only.

This allows backend systems to contribute events without importing/knowing `ConnectionPool`.

## 8. Specific Answer to “Who calls Broadcast today?”
Today `Broadcast` is called by:

1. SEM frame fanout closure in `conversation.go` (stream callback path).
2. Timeline upsert emitter in `timeline_upsert.go`.

`SendToOne` is called by:

1. `/ws` handler on hello.
2. `/ws` ping/pong response path.

No other production fanout paths are present in current inventory scripts.

## 9. Ownership and reference map
Practical ownership/reference structure:

1. `Router` owns `*ConvManager` and builder/hook dependencies.
2. `ConvManager` owns `map[string]*Conversation`.
3. `Conversation` owns:
   - `*ConnectionPool`
   - `*StreamCoordinator`
   - optional `*TimelineProjector`
   - `events.EventSink` used by inference run context
4. `StreamCoordinator` owns subscriber + callbacks, but does not own pool/projector.
5. `ConnectionPool` owns socket clients/channels and transport-level write/drop policy.

This ownership shape is stable and should be preserved while extracting broker/publisher abstractions.

## 10. Supporting Turn Snapshots over WS (Future Channel)

### 10.1 Current state
Turn snapshots are persisted via `snapshotHookForConv` and turn persister/store paths. They are exposed via debug HTTP routes (`/api/debug/turns`, `/api/debug/turn/...`). They are not currently emitted as websocket snapshot frames.

### 10.2 Proposed channel design
Add optional debug channel frame:

1. `event.type = "turn.snapshot"`
2. `data = {conv_id, session_id, turn_id, phase, created_at_ms, inference_id, payload}`
3. Classified as `debug.turn_snapshot`

### 10.3 Two-signal gating model (recommended)
Emit turn snapshot websocket frames only when both signals are true:

1. Producer intent (request/runtime policy): this inference is debug-snapshot-enabled.
2. Consumer subscription: at least one websocket client for this conversation has `debug.turn_snapshot` channel enabled.

This prevents accidental high-volume emissions while keeping explicit control.

### 10.4 Producer intent source (`EngineFromReqBuilder` path)
Use request policy resolution to carry debug intent:

1. Extend request build output with typed debug options (e.g. `DebugOptions.EmitTurnSnapshots`).
2. `startInferenceForPrompt` receives this option and composes hooks accordingly.
3. Snapshot emit hook runs after successful persistence save (persist-first ordering).

This keeps policy selection in request/build layer, not in low-level transport code.

### 10.5 Consumer subscription source (`/ws` path)
At websocket connect, parse connection capabilities:

1. `ws_profile=debug-full`, or
2. explicit `channels=debug.turn_snapshot`

This avoids forcing high-volume snapshot payloads on normal chat clients.

### 10.6 Clean emission API (no direct pool dependency)
Snapshot persistence hook should publish via websocket publisher interface (broker-backed), not by touching `ConnectionPool` directly from persistence code.

## 11. Migration Plan

### Phase 0: Instrument and baseline
1. Keep existing behavior.
2. Add metrics/log labels around fanout source and frame type.
3. Validate no regressions.

### Phase 1: Introduce broker behind existing behavior
1. Create broker abstraction and default implementation.
2. Wire existing producers to publish through broker.
3. Broker initially broadcasts to all to preserve behavior.

### Phase 2: Add subscription metadata
1. Parse `ws_profile/channels` on connect.
2. Store subscription record for each socket.
3. Add channel classification and filtering.

### Phase 3: Expose backend emitter hooks
1. Add `WithWSEmitterFactory` (or equivalent).
2. Provide helpers for constructing SEM envelopes and publishing.
3. Document extension contract.

### Phase 4: Optional debug channels
1. Implement `debug.turn_snapshot` emitter.
2. Gate by producer intent + channel subscription.
3. Add client support only in debug UI paths.

### Phase 5: Cleanup
1. Remove remaining direct `conv.pool.Broadcast` usage outside broker internals.
2. Finalize policy boundaries and tests.

## 12. Testing Strategy

### Unit tests
1. Broker dispatch eligibility by profile/channel matrix.
2. Subscription add/remove lifecycle.
3. Drop behavior unaffected by filtering logic.
4. Channel classification for known event types.

### Integration tests
1. Two clients same conversation, different profiles -> different received frame sets.
2. Timeline upsert reaches timeline subscribers.
3. Existing chat client with default params receives current behavior.
4. Optional debug turn snapshot channel only emits when producer intent + subscription are both set.

### Load/safety tests
1. Backpressure under high `llm.delta` rate.
2. Burst of timeline upserts.
3. Connect/disconnect churn with idle timers.

## 13. Operational Considerations

### 12.1 Observability
Add counters and logs for:

1. frames published by source and type
2. frames delivered by channel/profile
3. dropped frames / dropped connections
4. subscription counts by profile

### 12.2 Security and policy
Potential policy controls:

1. restrict debug profiles by auth context
2. reject unsupported channel requests
3. enforce maximum channel combinations per connection

### 12.3 Compatibility envelope
Even if we later remove legacy modes, migration should include one stable default profile (`chat`) matching today’s behavior to keep existing frontend intact while debug profile work lands.

## 14. Open Questions
1. Should profile selection be query params only, or also negotiable via first client frame?
2. Should filtering happen at frame-envelope level (`event.type`) or after protobuf decode?
3. Is a single broker per conversation sufficient, or do we need broker shards by channel for performance?
4. How long do we keep `semBuf` as-is once broker metrics/history exist?
5. Should debug UI bootstrap/catch-up remain HTTP-only or gain broker-backed replay endpoint?

## 15. Recommended Decision
Adopt the broker + transport split (Option 3), then stage in profile-aware filtering and backend emitter hooks. This gives a clean answer to both goals:

1. Connection-specific websocket behavior via profiles/channels.
2. Backend-side pluggability without requiring direct `ConnectionPool` access.

The first implementation increment should preserve today’s payload semantics and default fanout, then layer policy gradually.

## 16. Initial Implementation Task List
1. Introduce broker interfaces and default implementation.
2. Wrap `ConnectionPool` behind a transport adapter used by broker.
3. Route existing stream callback and timeline upsert through broker publish API.
4. Add connection IDs and subscription records at `/ws` connect.
5. Parse `ws_profile` and `channels` query params with validation.
6. Add frame channel classification helper and filtering rules.
7. Add router option for websocket emitter factories.
8. Add tests for profile/channel filtering and default compatibility.
9. Add observability counters/log fields for publication and dispatch.
10. Document protocol and extension contract for backend teams.
11. Add typed debug options in request builder output for producer-intent gating.
12. Wire turn snapshot emission through publisher API with persist-first ordering.
