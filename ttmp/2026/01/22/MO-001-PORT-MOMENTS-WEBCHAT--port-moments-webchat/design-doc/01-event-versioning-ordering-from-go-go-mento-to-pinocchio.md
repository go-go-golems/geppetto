---
Title: Event versioning + ordering (from go-go-mento to pinocchio)
Ticket: MO-001-PORT-MOMENTS-WEBCHAT
Status: active
Topics:
    - webchat
    - moments
    - session-refactor
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-mento/go/pkg/webchat/connection_pool.go
      Note: Reference ConnectionPool implementation
    - Path: go-go-mento/go/pkg/webchat/event_translator.go
      Note: Reference SEM translator with stable IDs
    - Path: go-go-mento/go/pkg/webchat/event_version_cache.go
      Note: Reference eventID->version cache
    - Path: go-go-mento/go/pkg/webchat/ordering_middleware.go
      Note: Reference block ordering middleware
    - Path: go-go-mento/go/pkg/webchat/stream_coordinator.go
      Note: Reference StreamCoordinator implementation and XID extraction
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Current pinocchio reader/broadcast implementation to be replaced
    - Path: pinocchio/pkg/webchat/forwarder.go
      Note: Current pinocchio SEM mapping; add version fields
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T11:18:00.721670764-05:00
WhatFor: ""
WhenToUse: ""
---


# Event versioning + ordering (from go-go-mento to pinocchio)

## Executive Summary

We want to move the “good transport” parts of moments’ (go-go-mento) webchat into
Pinocchio without bringing in DB persistence yet, while still improving:

- **Ordering**: deterministic client ordering of events across websocket reconnects,
  redis stream consumption, and “late” frames.
- **Versioning**: a lightweight, monotonic version/cursor that can be attached to
  events/frames so the UI and future persistence/hydration layers have a stable
  notion of “what happened when”.

The concrete proposal is to port (or re-create) go-go-mento’s `StreamCoordinator`
and “stream-derived version” mechanism into Pinocchio as a first-class concept,
and to standardize where ordering guarantees are enforced:

- **Transport ordering**: in the subscriber/reader (consume stream sequentially).
- **Frame ordering**: translator emits frames in a stable order for each event.
- **Turn/block ordering**: a middleware (or pre-run normalizer) ensures blocks in
  a Turn are in the canonical order before calling the provider.

This doc is written to be implementation-adjacent: diagrams, invariants, and
pseudocode are provided, with references to the current implementations.

## Problem Statement

Pinocchio’s current webchat has a working MO-007-aligned inference loop (Session
→ ExecutionHandle), but its websocket and streaming layer is minimal:

- Reader loop is per conversation and tightly coupled to websocket broadcasting.
- There is no concept of a “stream cursor/version” available to downstream code.
- Ordering is implicitly “whatever arrives first”; this fails on reconnect or if
  we ever add replay/resume.

go-go-mento’s webchat has the opposite: strong streaming infrastructure:

- `StreamCoordinator` consumes a Watermill subscriber, translates events into
  SEM frames, and calls `onEvent` and `onFrame` synchronously to preserve ordering.
- It attempts to extract a monotonic **version** from Redis stream IDs (XIDs) via
  Watermill metadata and passes that version to hydration/persistence.

We want the best of both:

- Keep Pinocchio’s MO-007 inference model.
- Move go-go-mento’s stream/ordering/versioning improvements into Pinocchio.

Constraints and explicit product choices for this port:

- Middleware list application order in Pinocchio should match go-go-mento:
  apply **in reverse**, so “first in list” reads like “outermost wrapper”.
- Geppetto’s tool loop should accept a pluggable `ToolExecutor` (needed by moments);
  this impacts how “tool events” are generated and thus ordering guarantees.
- No DB persistence for now; the versioning work should still be useful without it.

## Proposed Solution

### 1) Introduce “stream ordering” as an explicit subsystem (StreamCoordinator)

Port the conceptual structure of:

- `go-go-mento/go/pkg/webchat/stream_coordinator.go`
- `go-go-mento/go/pkg/webchat/connection_pool.go`

into Pinocchio (either directly, or reimplemented with the same contract).

The key contract is that **one goroutine** consumes the subscriber stream, and
events are processed sequentially:

```
subscriber.Subscribe(ctx, topic) -> ch
for msg := range ch:
    event := Decode(msg)
    onEvent(event)  // synchronous
    for frame in Translate(event):
        onFrame(event, frame) // synchronous
    msg.Ack()
```

This yields a very strong and simple invariant:

- If the subscriber produces messages in order for a topic, then all downstream
  effects (`onEvent` side effects and frames) are processed in that order.

Pinocchio’s current reader loop is similar, but lacks modularity and doesn’t
extract version/cursor information. This change isolates that.

### 2) Define “event stream version” and how it is extracted

We define a “stream version” as a monotonically increasing value associated with
the *delivery* order of events on a topic. In Redis Streams this is naturally
the stream XID (e.g. `1700000000000-42`).

In go-go-mento today:

- `StreamCoordinator` looks for XID-ish metadata keys (`xid`, `redis_xid`, …) on
  the Watermill message, parses `timestamp-seq`, and computes `version := ts + seq`.
- It stores `(event_id -> version)` in a transient cache (`event_version_cache.go`).
- Later, the hydration callback (`streamOnEvent`) retrieves and deletes that
  version and passes it into persistence.

References:

- `go-go-mento/go/pkg/webchat/stream_coordinator.go`
- `go-go-mento/go/pkg/webchat/event_version_cache.go`
- `go-go-mento/go/pkg/webchat/conversation.go` (`streamOnEvent`)

For Pinocchio (no DB yet), we propose a slightly more explicit approach:

1. Extract a `StreamCursor` from the subscriber message:
   - Prefer the raw XID string (e.g. `1700000000000-42`).
2. Derive a `StreamVersion` representation used for ordering:
   - Either keep the XID string (recommended; preserves total ordering and avoids collisions),
   - Or derive an `int64` (only if required by existing consumers).
3. Attach this to:
   - the decoded `events.Event` *as metadata extra*, and/or
   - the emitted SEM frames (as `event.version` or `event.stream_id`).

Because we’re not persisting yet, the immediate payoff is:

- UI can order events deterministically even if frames arrive late or are replayed.
- If we later add persistence, the “version wiring” is already designed.

#### What about transports with no XID (in-memory / non-Redis)?

Some transports (e.g. the in-memory Watermill router / Go channels) do not have
a stream-native cursor like Redis XID. We still need deterministic ordering and
deduping *within a single server lifetime*.

The rule is:

- If `stream_id` exists (Redis Streams), it is the authoritative cursor.
- Otherwise, the server synthesizes an in-memory monotonic sequence `seq` per
  `conv_id` (assigned at consume-time, not publish-time).

Concretely, the StreamCoordinator owns:

- `seq uint64` counter per conversation stream, incremented once per consumed message.

And each emitted SEM envelope carries:

- `stream_id` (optional)
- `seq` (always)

This gives us:

- Redis mode: stable ordering across reconnect/replay (`stream_id`).
- Memory mode: stable ordering for the lifetime of the process (`seq`), which is
  sufficient as long as we’re not doing persistence/replay for that transport.

Pseudocode sketch:

```go
type Cursor struct {
    StreamID string // optional
    Seq      uint64 // always present (fallback)
}

func (sc *StreamCoordinator) consume(ctx context.Context) {
    for msg := range ch {
        ev := decode(msg.Payload)
        cur := Cursor{
            StreamID: extractXID(msg.Metadata), // "" if not present
            Seq:      sc.seq.Add(1),
        }
        for _, frame := range Translate(ev, cur) {
            onFrame(ev, frame)
        }
        msg.Ack()
    }
}
```

### 3) Clarify the “three orderings” (and where each is enforced)

To avoid mixing concerns, we treat ordering at three layers.

#### A) Stream ordering (subscriber → events)

Goal: ensure we *process* events in topic order.

- Implemented by `StreamCoordinator.consume(...)` reading one channel sequentially.

#### B) Frame ordering (event → SEM frames)

Goal: given one event, produce frames in a deterministic sequence.

- Example: for a tool call result, emit `tool.result` then `tool.done`.
- Implemented in the translator (`EventTranslator`/`SemanticEventsFromEvent`).

References:

- Pinocchio: `pinocchio/pkg/webchat/forwarder.go`
- go-go-mento: `go-go-mento/go/pkg/webchat/event_translator.go`

#### C) Turn/block ordering (Turn → engine input)

Goal: the Turn given to the provider has a canonical ordering of blocks that:

- keeps system context first,
- puts user/tool contexts in predictable positions,
- keeps tool results adjacent to their call groups (UX and provider correctness),
- prevents “tool call noise” leaking into assistant text streams.

go-go-mento has a dedicated ordering middleware for blocks:

- `go-go-mento/go/pkg/webchat/ordering_middleware.go`

Pinocchio currently relies on `middleware.NewToolResultReorderMiddleware()` and
does not have “section-based” ordering. We should port the concept (not
necessarily the exact metadata keys) to Pinocchio as a normalizer that runs
*before* inference.

### 4) Add a lightweight “version field” to SEM envelopes (no DB required)

We extend the semantic event envelope with either:

- `event.stream_id` (string XID), and/or
- `event.version` (int64, derived).

Example envelope:

```json
{
  "sem": true,
  "event": {
    "type": "llm.delta",
    "id": "…",
    "run_id": "…",
    "turn_id": "…",
    "stream_id": "1700000000000-42",
    "version": 1700000000042
  }
}
```

The UI can treat `(conv_id, stream_id)` as a dedupe key and use `stream_id` as
the primary ordering comparator.

### 5) Keep middleware list application order in reverse (explicit policy)

go-go-mento composes middlewares by iterating from last to first so that a list:

`[A, B, C]`

means “A wraps (B wraps (C wraps base))”.

Pinocchio’s `composeEngineFromSettings(...)` currently applies in list order.
We should standardize on go-go-mento’s behavior in Pinocchio, because it matches
how humans read middleware stacks (“top to bottom” as outer to inner).

Reference:

- go-go-mento: `go-go-mento/go/pkg/webchat/engine.go` (reverse application)
- pinocchio: `pinocchio/pkg/webchat/engine.go` (currently forward)

## Design Decisions

### Decision: Prefer Redis XID string as the primary “version”

Rationale:

- Redis stream IDs are already totally ordered within a stream.
- Converting to `int64` loses information (`ts-seq`), can collide, and invites subtle bugs.
- The UI can compare XIDs lexicographically if we normalize, or parse them if needed.

### Decision: Surface ordering/version in SEM frames

Rationale:

- Without DB, the UI still benefits immediately from stable ordering on reconnect.
- Debugging becomes easier (“why is this out of order?” becomes inspectable).

### Decision: Synchronous processing in StreamCoordinator

Rationale:

- Preserves ordering naturally.
- Avoids a class of races (hydration/persistence seeing events out of order).
- Makes “exactly once per subscriber message” semantics easier to reason about.

### Decision: Keep “turn/block ordering” as a middleware/normalizer

Rationale:

- Provider input correctness should not depend on incidental block append order.
- Keeping ordering local (before inference) reduces surprises in tool loops/middleware.

## Alternatives Considered

### Alternative: Don’t add versioning; rely on websocket frame arrival order

Rejected because reconnect/resume breaks ordering, and it becomes impossible to
dedupe or replay cleanly later.

### Alternative: Use server `time.Now().UnixMilli()` as version

Rejected because:

- multiple events can share the same timestamp,
- server time is not a reliable proxy for stream order,
- and it fails in distributed setups.

### Alternative: Maintain a per-conversation counter in memory only

Feasible for non-Redis mode, but insufficient for Redis Streams:

- a restarted server loses the counter state,
- and it does not match the stream’s authoritative order.

Use it only as a fallback when no stream cursor is available (e.g. in-memory router).

## Implementation Plan

1) Add `StreamCoordinator` + `ConnectionPool` to Pinocchio webchat (or a shared package).
   - Start with a straight port of the go-go-mento interfaces.
2) Teach the reader to extract `stream_id` from subscriber message metadata.
   - Store it in an in-memory map keyed by message/event id, *or* attach it to
     the event metadata extra field directly.
3) Update Pinocchio’s `SemanticEventsFromEvent` to include version fields in envelopes.
4) Add a block-ordering middleware/normalizer to Pinocchio webchat.
   - Port the section-based ordering idea (`ordering_middleware.go`).
5) Update Pinocchio’s engine composition to apply middlewares in reverse order.
6) (Related, but required for moments): extend `toolhelpers.RunToolCallingLoop`
   to accept a pluggable `tools.ToolExecutor` so authorized execution can be
   used without reimplementing the entire loop.

## Open Questions

1) Should `stream_id` live in:
   - the event metadata (`EventMetadata.Extra`), or
   - only in the SEM envelope, or
   - both?
2) If we keep `version` as `int64`, what exact mapping should we use from `ts-seq`?
   - (Recommend: don’t; keep `stream_id` string.)
3) In non-Redis mode, what is the authoritative version?
   - likely a per-conversation monotonic counter in memory.

## References

- MO-007 session design (reference architecture):
  - `geppetto/ttmp/2026/01/21/MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/design-doc/01-session-refactor-sessionid-enginebuilder-executionhandle.md`
- go-go-mento implementations:
  - `go-go-mento/go/pkg/webchat/stream_coordinator.go`
  - `go-go-mento/go/pkg/webchat/connection_pool.go`
  - `go-go-mento/go/pkg/webchat/event_version_cache.go`
  - `go-go-mento/go/pkg/webchat/event_translator.go`
  - `go-go-mento/go/pkg/webchat/ordering_middleware.go`
- Pinocchio implementations:
  - `pinocchio/pkg/webchat/conversation.go`
  - `pinocchio/pkg/webchat/forwarder.go`
