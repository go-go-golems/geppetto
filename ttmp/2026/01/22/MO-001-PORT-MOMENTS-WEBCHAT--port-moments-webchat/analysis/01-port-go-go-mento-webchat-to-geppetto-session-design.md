---
Title: Port go-go-mento webchat to Geppetto session design
Ticket: MO-001-PORT-MOMENTS-WEBCHAT
Status: active
Topics:
    - webchat
    - moments
    - session-refactor
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/01/21/MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/design-doc/01-session-refactor-sessionid-enginebuilder-executionhandle.md
      Note: Target session/ExecutionHandle design for port
    - Path: go-go-mento/go/pkg/webchat/conversation_manager.go
      Note: ConversationManager (get-or-create
    - Path: go-go-mento/go/pkg/webchat/loops.go
      Note: Custom authorized tool loop + step-mode pauses
    - Path: go-go-mento/go/pkg/webchat/router.go
      Note: Moments webchat router (ws/http/cancel/run loop)
    - Path: go-go-mento/go/pkg/webchat/stream_coordinator.go
      Note: StreamCoordinator (subscriber→event/frame
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Pinocchio ws reader/broadcast (candidate for ConnectionPool/StreamCoordinator)
    - Path: pinocchio/pkg/webchat/router.go
      Note: Pinocchio webchat entrypoint using geppetto session
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T10:52:15.884835829-05:00
WhatFor: ""
WhenToUse: ""
---


# Port go-go-mento webchat to Geppetto session design

## Scope / Goal

Analyze what would need to change in `go-go-mento/go/pkg/webchat` to use the
new Geppetto **Session + EngineBuilder + ExecutionHandle** model (MO-007), and
propose a reconciliation plan with Pinocchio’s current webchat implementation.

Explicit goals:

1. Move “moments webchat” toward the MO-007 session model (no `InferenceState`,
   no ad-hoc cancel plumbing, “one active inference at a time” semantics).
2. Reconcile/simplify duplication between:
   - `go-go-mento/go/pkg/webchat` (strong websocket + conversation manager)
   - `pinocchio/pkg/webchat` (already using `geppetto/pkg/inference/session`)
3. Preserve the best parts of go-go-mento:
   - `StreamCoordinator` (subscriber → event → frames, ordering + version extraction)
   - `ConnectionPool` (broadcast, idle, drop-on-error)
   - `ConversationManager` (get-or-create, eviction, config signature rebuilds)
4. Align naming/semantics: **SessionID vs ConversationID vs RunID/InferenceID**
   so that streaming, persistence, and cancellation are coherent.

Non-goals (for this ticket’s analysis):

- Implement the port or do code moves right now.
- Decide on exact UI payload schema changes (SEM/timeline), beyond identifying
  “must not break” expectations and where normalization should live.

## Reference: MO-007 session model (Geppetto)

MO-007 establishes these core rules:

- `session.Session` represents a long-lived multi-turn conversation (stable `SessionID`).
- One active inference at a time.
- `StartInference(ctx)` returns an `ExecutionHandle` (cancel + wait).
- Event sinks are injected via context (`events.WithEventSinks`), not engine config.
- Tool loop orchestration lives in an EngineBuilder/InferenceRunner (e.g. `session.ToolLoopEngineBuilder`).

Implication: any webchat server wanting “cancel”, “run in progress”, “streaming
events”, “persistence”, and “tools” should treat these as:

- **Session manager concerns** (own Session instances, map conv_id → Session, enforce exclusivity),
- **Runner concerns** (construct engine/middleware, attach sinks/hook/persister, run tool loop),
- **Transport concerns** (ws connection pooling, event subscription, frame translation).

## Current state: go-go-mento webchat architecture (what’s “good” and what’s “legacy”)

### “Good”: transport + lifecycle orchestration

The following pieces are strong and worth keeping conceptually:

- `ConnectionPool`: centralizes websocket connection management, broadcast,
  drop-on-write-error, and idle timeout.
- `StreamCoordinator`: centralizes subscriber consumption and ensures ordered
  delivery to:
  - `onEvent` (hydration/persistence), and
  - `onFrame` (websocket broadcast).
  It also extracts an ordering-friendly “version” from Redis stream IDs and
  caches it for downstream hydration.
- `ConversationManager`: provides:
  - `GetOrCreate(convID, profileSlug, overrides, req)` with rebuild-on-signature-change,
  - stream attach/start,
  - idle eviction loop, and
  - lifecycle hooks like “load persisted turns on resume”.

These are exactly the kinds of concerns Pinocchio’s current webchat lacks.

### “Legacy”: pre-MO-007 inference lifecycle

go-go-mento still holds inference lifecycle via `InferenceState`:

- `StartRun()` / `FinishRun()` / `SetCancel()` / `CancelRun()`
- `running bool` + `context.CancelFunc` manually managed per conversation

It is *conceptually* the older “InferenceState” model MO-007 replaced with
`ExecutionHandle`. The webchat router starts goroutines itself and manually
publishes an `EventInterrupt` on cancellation.

### Hybrid: engine composition vs run orchestration

go-go-mento has its own `EngineBuilder` that returns:

`(engine.Engine, events.EventSink, EngineConfig, error)`

and a custom `ToolCallingLoop` that does:

- `eng.RunInference` iteration
- tool call extraction + execution via an **authorized** tool executor
- step-mode pause points (debugger pause events)
- optional persistence via router hooks

Under MO-007, this logic should live behind an `InferenceRunner` returned by a
`session.EngineBuilder`, and the webchat router should no longer directly own
the goroutine/cancel bookkeeping.

## Current state: Pinocchio webchat architecture (what’s “good” and what’s missing)

### “Good”: it already uses `session.Session`

Pinocchio’s `/chat` endpoint already:

- uses `session.Session`,
- sets `conv.Sess.Builder = &session.ToolLoopEngineBuilder{...}`,
- calls `handle, _ := conv.Sess.StartInference(...)` and `handle.Wait()` in a goroutine.

So Pinocchio is *structurally aligned* with MO-007 already.

### Missing: the strong transport/session manager layer

Pinocchio’s current webchat is comparatively minimal:

- Websocket connections are stored as a `map[*websocket.Conn]bool` with separate locks.
- Broadcast writes happen without pooling semantics (drop-on-error, writer serialization, etc.).
- Streaming reader is a per-conversation goroutine reading from Watermill subscriber directly.
- There is no `StreamCoordinator` abstraction; no ordering/version extraction.
- There is no “ConversationManager” layer with:
  - eviction policies beyond a simple idle timer per conversation,
  - rebuild-on-config-change signature logic,
  - load/persist turn history as a first-class concern.

### Naming/semantics mismatch: SessionID vs run_id vs conv_id

Pinocchio sets `Session.SessionID` to a generated `runID`, while also taking a
`conv_id` as the topic identifier for events.

This diverges from MO-007’s vocabulary:

- `SessionID` should be stable and typically correspond to “conversation ID”.
- “inference/run” should be a separate identifier (e.g. `ExecutionHandle.InferenceID`)
  that can be used for UI correlation and per-inference filtering.

This mismatch matters when you want:

- persistence keyed by “conversation id”,
- multiple turns over time under one session,
- event streaming that can include multiple inferences safely.

## Delta analysis: what must change in go-go-mento to be “MO-007-native”

This section assumes we want go-go-mento’s webchat to adopt MO-007 directly
(even if the longer-term plan is to consolidate into Pinocchio’s package).

### 1) Replace `InferenceState` with `session.Session` + `ExecutionHandle`

Change:

- Remove `InferenceState` fields from `Conversation` (or reduce `Conversation`
  to transport-only state).
- Move “running/cancel” semantics to `ExecutionHandle`.
- Replace:
  - `conv.StartRun()` → `handle, err := conv.Session.StartInference(ctx)`
  - `conv.CancelRun()` → `handle.Cancel()`
  - `conv.FinishRun()` → “inference goroutine ends; handle done; session appends turn”

Implications:

- Cancellation no longer needs a stored `CancelFunc`. The handle owns it.
- “Run in progress” checks become `Session.IsRunning()` (or “active handle not nil”).
- The router/manager must define when a Turn is appended (MO-007 suggests appending on success).
- If you still need “always emit an interrupt event on cancel”, that becomes a
  runner/session concern (see below).

### 2) Decide the stable identifier strategy (SessionID, conv_id, run_id)

Strong recommendation for MO-007 alignment:

- Treat `conv_id` (UI-visible) as the **SessionID**.
- Treat “inference id” as what the UI currently calls `run_id`.
- Ensure `events.EventMetadata.RunID` is set to the inference id (not the conversation id).

This enables:

- stable persistence keys (`SessionID == conv_id`),
- safe streaming of historical events per conversation topic,
- accurate cancellation targeting (cancel the active `ExecutionHandle` for that session),
- and a clean server-side rule: one active inference at a time per session.

This will require updating:

- how go-go-mento generates and returns `run_id`,
- how the websocket stream filters (if it filters) and what the UI expects.

### 3) Move custom loop semantics behind an `InferenceRunner`

go-go-mento’s `ToolCallingLoop` has two key “non-standard” behaviors:

1. Authorized tool execution (needs identity session).
2. Step-mode pausing at well-defined phases.

Under MO-007, the clean reconciliation is:

- Keep the custom loop implementation (for now).
- Wrap it in a `session.EngineBuilder` implementation that returns a custom `InferenceRunner`.

That runner’s `RunInference` would:

- attach `events.WithEventSinks(ctx, ...)`,
- attach snapshot hook (for persistence/debug),
- attach identity session / request-scoped values (if needed),
- run the custom loop,
- persist turn snapshots via a `TurnPersister` abstraction (not via `RouterFromContext`).

This prevents the HTTP handler layer from becoming the orchestration layer.

### 4) Standardize event emission on cancel and terminal states

go-go-mento currently emits an `EventInterrupt` in a defer when it observes `context.Canceled`.

Under MO-007, we should decide one consistent behavior across products:

- If an inference is canceled, always emit a terminal UI event that:
  - causes “generating…” to end,
  - provides stable correlation (inference id + turn id),
  - is never duplicated.

Best place:

- In the runner wrapper (the piece that owns the inference goroutine), not in HTTP handlers.

Pinocchio would benefit from the same behavior (today it relies on provider behavior + tool loop).

### 5) Persistence and “load turns on resume” should become first-class runner/session concerns

go-go-mento currently:

- persists turns via `RouterFromContext(...)` inside the loop,
- loads the most recent persisted turn and seeds `InferenceState.Turn`.

In MO-007 terms:

- turn persistence should be an injected `TurnPersister` dependency of the runner.
- loading turns should be a `SessionManager` concern that constructs/rehydrates
  `session.Session` with a turn list.

This is also where “turn version from Redis stream id” can be integrated cleanly:

- Either propagate an “event version” into persistence APIs explicitly, or
- write snapshots with monotonic time but keep event ordering separate.

### 6) EngineBuilder/config signature logic becomes a “Session configuration” concept

go-go-mento’s rebuild-on-signature-change behavior is valuable, but it should be
expressed in MO-007 terms as:

- Session has stable identity.
- SessionManager stores the “current desired config signature”.
- If a request changes profile/overrides in a way that changes the signature:
  - cancel active inference (or reject until idle),
  - rebuild runner/builder for subsequent runs,
  - keep transport subscription stable (topic based on conv id / session id).

## Reconciliation plan: converge go-go-mento and Pinocchio webchat

The core observation:

- Pinocchio already has the *right inference abstraction* (MO-007 session),
  but lacks the “product-grade” session/connection/stream management layer.
- go-go-mento has the *right transport/session management layer*, but still uses
  the pre-MO-007 inference lifecycle and custom loop wiring.

So the lowest-risk convergence direction is:

1) Bring go-go-mento’s **ConnectionPool + StreamCoordinator + EventTranslator**
   patterns into `pinocchio/pkg/webchat` (or a shared package).
2) Bring go-go-mento’s **ConversationManager** concepts into Pinocchio as a
   `SessionManager` that owns `session.Session` instances (instead of `InferenceState`).
3) Port moments-specific concerns (identity auth, step-mode, hydration/persistence)
   as optional hooks/adapters layered on top of that shared core.

### Where should the shared code live?

Options:

1) `geppetto/pkg/webchat` (best “shared library” location across products)
   - Pros: geppetto already defines events/turns/session; reuse is natural.
   - Cons: pinocchio currently owns the web UI assets + router entrypoints.

2) `pinocchio/pkg/webchat` becomes the “canonical” implementation
   - Pros: already exists; referenced by MO-007 as migration target.
   - Cons: moments/go-go-mento is conceptually not “pinocchio”; importing pinocchio
     may feel wrong unless pinocchio is treated as a host app, not a library.

3) Keep separate but align concepts
   - Pros: no code moves required immediately.
   - Cons: duplication and drift continue (and “porting” becomes “reimplementing”).

Recommendation:

- Long-term: extract the core into `geppetto/pkg/webchatcore` (or similar).
- Short-term: implement the improvements in `pinocchio/pkg/webchat`, then extract once stable.

### What to port from go-go-mento into Pinocchio first (high ROI)

1) `ConnectionPool` semantics
   - broadcast drop-on-error
   - `SendToOne` for “hello/pong”
   - idle timer triggers `StreamCoordinator.Stop()` (not just cancels reader)

2) `StreamCoordinator` semantics
   - centralized subscriber consumption
   - synchronous `onEvent` and `onFrame` callbacks for ordering guarantees
   - version extraction from Redis stream ID and a defined propagation strategy

3) `EventTranslator` quality improvements
   - stable message IDs correlated by RunID/TurnID
   - stripping tool-call noise from assistant streams
   - registry-based translation hooks (optional)

4) `SessionManager` / “ConversationManager” semantics
   - map `conv_id` → session state and transport
   - eviction loop (not just per-conv idle timers)
   - rebuild-on-config-signature-change
   - load persisted turns on resume (if pinocchio adds persistence)

### What moments/go-go-mento can contribute back after convergence

Moments-specific features can be treated as optional add-ons:

- Identity/ownership model (cancel authorization, owner adoption).
- Step-mode pause/continue (debug endpoints + pause events).
- Timeline hydration/persistence integration.
- Specialized SEM handlers / renderer registry.

The key is to define a minimal set of interfaces so Pinocchio can “host” those
without importing moments internals.

## Design implications and tricky points

### Middleware ordering mismatch (Pinocchio vs go-go-mento)

- Pinocchio composes middleware in list order, which makes later entries wrap earlier ones.
- go-go-mento applies in reverse, so the first declared middleware becomes outermost.

If we unify code, we must pick one ordering and document it clearly; otherwise
the same profile definition behaves differently across servers.

### Tool execution customization

Pinocchio’s `toolhelpers.RunToolCallingLoop` uses `tools.NewDefaultToolExecutor(...)`.
Moments needs an authorized executor (and may need per-tool policies).

Two viable reconciliation paths:

1) Keep moments’ custom tool loop behind a custom `InferenceRunner`.
2) Extend geppetto’s toolhelpers/session builder to accept a `tools.ToolExecutor`
   (or executor factory) so both products share the same orchestration logic.

Path (1) is less invasive initially; path (2) is better long-term.

### Cancellation UX (terminal event guarantees)

If the UI relies on a terminal “llm.final”/interrupt to exit “generating…”,
the server must ensure it emits one on cancel even if the provider returns early
or never produces a final chunk.

This should be consistent across:

- moments webchat,
- pinocchio webchat,
- and any future UI clients.

### Persistence + replay vs “live stream only”

go-go-mento includes:

- persisting turns (blocks) and loading recent turns on resume.

Pinocchio currently:

- has snapshot-to-files support (debugging), but no DB persistence or replay.

If we want “reconnect and see history”, Pinocchio needs:

- a persisted turn store (DB or files),
- a replay protocol (either “send initial snapshot on ws.hello” or “client fetch history via HTTP”),
- and a consistent versioning model (event-derived versions vs time-based).

### Versioning and ordering

go-go-mento extracts a monotonic-ish version from Redis stream IDs and threads
it into hydration logic via a transient cache.

If we unify:

- either formalize “event stream id/version” as part of the event metadata/frame envelope, or
- keep the transient cache but confine it to the server and document that it is best-effort.

## Concrete “port plan” (proposed sequencing)

This is an implementation-oriented outline derived from the above.

1) **Fix SessionID semantics in Pinocchio**
   - Set `Session.SessionID = conv_id` (stable conversation id).
   - Use a per-inference ID for `run_id`/event `RunID`.

2) **Introduce `SessionManager` to Pinocchio webchat**
   - Replace `ConvManager` with a manager that owns:
     - session instances,
     - active handles,
     - engine config signature,
     - connection pool,
     - stream coordinator.

3) **Move websocket handling to ConnectionPool + StreamCoordinator**
   - Replace direct read loop/broadcast in `pinocchio/pkg/webchat/conversation.go`.
   - Add ws.hello + ws.ping/ws.pong support (optional but very useful operationally).

4) **Unify translator**
   - Either port go-go-mento’s `EventTranslator` behavior to Pinocchio’s forwarder,
     or extract a shared translator with hooks.

5) **Port moments-specific features as adapters**
   - Authorized tool execution hook.
   - Step controller/pause endpoints.
   - Turn persistence and load-on-resume.

6) **Deprecate/retire go-go-mento webchat**
   - Replace with a thin wrapper around the shared/Pinocchio implementation, or
   - keep as “application wiring” only (no duplicated core logic).

## Open questions / decisions to make explicitly

1) What is the canonical definition of `run_id`?
   - per-inference (recommended), or per-conversation (legacy behavior)?
2) Do we want ws connections to receive:
   - “only latest inference events”, or “all session events”?
3) Where does “history replay” live:
   - WS initial frames, or HTTP fetch + WS live stream?
4) Do we extend geppetto’s `toolhelpers` to accept a custom executor?
