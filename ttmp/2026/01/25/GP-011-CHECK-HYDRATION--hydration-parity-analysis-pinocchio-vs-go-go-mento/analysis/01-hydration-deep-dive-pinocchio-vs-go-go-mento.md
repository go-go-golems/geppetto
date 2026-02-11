---
Title: 'Hydration deep dive: pinocchio vs go-go-mento'
Ticket: GP-011-CHECK-HYDRATION
Status: active
Topics:
    - hydration
    - webchat
    - persistence
    - events
    - backend
    - frontend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-mento/go/pkg/persistence/timelinehydration/projector.go
      Note: Event projection and user message entity helper
    - Path: go-go-mento/go/pkg/persistence/timelinehydration/repo.go
      Note: Version-aware upsert and hydration query ordering
    - Path: go-go-mento/go/pkg/webchat/stream_coordinator.go
      Note: Redis XID extraction for ordering versions
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Frontend hydration gating and snapshot insertion order
    - Path: pinocchio/pkg/webchat/router.go
      Note: Hydration endpoints and user message persistence
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: SEM frame projection into timeline entities
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: SQLite timeline schema and ordering rules
ExternalSources: []
Summary: Deep comparative analysis of hydration in pinocchio vs go-go-mento, with ordering gap diagnosis and fix blueprint.
LastUpdated: 2026-01-25T09:36:12.636004258-05:00
WhatFor: Explain how hydration works in both repos and identify ordering regressions for user messages in pinocchio.
WhenToUse: Use when debugging hydration ordering, aligning pinocchio with the original go-go-mento design, or planning fixes to timeline persistence.
---


# Hydration Deep Dive: pinocchio vs go-go-mento

## Executive Summary (Short)

Pinocchio implements hydration in two layers: a transient SEM-frame buffer (`/hydrate`) and an optional durable SQLite projection store (`/timeline`). The durable path orders a full snapshot by `created_at_ms`, which defaults to the *first upsert time* per entity. User messages are inserted via a separate `/chat` path (not the SEM stream), so their timestamps are decoupled from the event stream and can collide at millisecond precision, falling back to lexicographic `entity_id` ordering. On restore, the frontend preserves snapshot order without re-sorting, so any collision or ordering drift becomes visible as misordered user messages.

Go-go-mento treats hydration as a full subsystem (projector → service → repo) with explicit versioning. When Redis is present, it converts stream XIDs into monotonic versions; otherwise it uses time plus a version tracker to avoid collisions. User messages are persisted and broadcast with the *same version* as their SEM frames, so ordering and UI reconstruction stay consistent. The most likely gap in pinocchio is the lack of a stable, monotonic version assigned to user messages and the absence of stream-derived ordering in the durable timeline path.

The fix is to give pinocchio a first-class notion of “ordering version” (derived from Redis XID or a monotonic counter), assign it to user messages and SEM-projected entities, and use it in `/timeline` ordering. Pinocchio already has the pieces (event stream, sem buffer, timeline projector); it needs the same ordering discipline that go-go-mento enforces.

---

# Chapter 1 — Hydration in Pinocchio (Textbook Treatment)

Hydration in pinocchio is the process of reconstructing conversational UI state from server-side state, with two complementary mechanisms: **(1) short‑lived SEM frame replay** and **(2) durable timeline snapshots**. Think of the system as a pipeline that transforms engine events into UI‑meaningful entities, while providing a way to rehydrate the UI after disconnection or reload.

## 1.1 Conceptual Model

Pinocchio’s web chat has three layers of state:

1) **Event stream:** the engine emits Geppetto events (LLM partials, tool calls, logs). These are routed via Watermill (in‑memory or Redis transport).
2) **Semantic frames:** the event stream is converted to SEM frames (JSON envelopes). These are streamed to the browser in real time and buffered for quick replays.
3) **Timeline entities:** a canonical projection of those events is persisted into a timeline store for durable hydration.

Hydration means “rebuilding the UI state from a stored representation.” Pinocchio provides two representations:

- **SEM frames (transient):** `/hydrate` returns a buffered subset of SEM frames (used when the timeline store is not enabled). This is a fast, memory‑only replay path. (`pinocchio/pkg/webchat/router.go` + `pinocchio/pkg/webchat/sem_buffer.go`)
- **Timeline snapshots (durable):** `/timeline` returns a projection stored in SQLite (or other `TimelineStore`). This is the long‑lived “actual hydration” path. (`pinocchio/pkg/webchat/timeline_store_sqlite.go` + `router.go`)

The key consequence: **the ordering semantics are different across these paths.** `/hydrate` uses SEM sequence numbers. `/timeline` uses `created_at_ms` or `version` depending on whether the request is full or incremental.

## 1.2 End‑to‑End Flow (Server)

A pinocchio conversation is created or reused via `Router.getOrCreateConv` (`pinocchio/pkg/webchat/conversation.go`). Each conversation binds:

- an engine (`Eng`),
- an event sink (`Sink`),
- a subscriber (`sub`) to the Watermill topic, and
- a stream coordinator that reads events and emits SEM frames to clients.

When an event arrives:

1) **Stream coordinator** translates the event into SEM frames (`EventTranslator` + SEM registry).
2) Each frame is **broadcast** to WebSocket clients.
3) The frame is also **buffered** in a per‑conversation `semFrameBuffer`.
4) If a timeline store exists, the frame is **projected** into a timeline entity by `TimelineProjector.ApplySemFrame` and **persisted** (`TimelineStore.Upsert`).

This is the heart of pinocchio’s hydration pipeline: the stream powers live UI updates and gradually builds a durable projection.

## 1.3 Hydration Path A: `/hydrate` (SEM Frame Replay)

The `/hydrate` endpoint returns buffered SEM frames as JSON, filtered by `since_seq` and optionally limited (`pinocchio/pkg/webchat/router.go`). The server constructs a response containing:

- `frames`: sem envelopes, in buffer order,
- `last_seq`: the highest SEM sequence observed,
- `queue_depth`: inflight server queue, and
- session metadata.

On the frontend, `wsManager.ts` uses `/hydrate` only if `/timeline` is unavailable. It sorts frames by `seq` and replays them through `handleSem`, ensuring deterministic ordering by the **SEM sequence numbers** (which come from the event registry/translator, not the DB). (`pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`)

**Invariant:** `/hydrate` ordering reflects the SEM stream order, not any persisted ordering.

## 1.4 Hydration Path B: `/timeline` (Durable Projection)

When the timeline store is enabled, `/timeline` returns a `TimelineSnapshotV1` produced by `TimelineStore.GetSnapshot` (`pinocchio/pkg/webchat/timeline_store_sqlite.go`). The SQLite store persists:

- `timeline_versions`: per‑conversation version counter,
- `timeline_entities`: per entity state with `created_at_ms`, `updated_at_ms`, and `version`.

**Ordering rules:**

- Full snapshot (`sinceVersion == 0`): `ORDER BY created_at_ms ASC, entity_id ASC`
- Incremental snapshot: `ORDER BY version ASC`

This is a critical design choice: **full hydration order is driven by `created_at_ms`**, not the version counter.

**Entity timestamps:**

`Upsert` sets `created_at_ms` on the first insertion of an entity. If the entity’s `CreatedAtMs` is not explicitly provided, it defaults to `time.Now()` at the time of upsert. (`pinocchio/pkg/webchat/timeline_store_sqlite.go`)

**Implication:** The *first time* an entity is seen determines its position in the restored timeline.

## 1.5 Timeline Projection (SEM → Entities)

`TimelineProjector` consumes SEM frames and writes `TimelineEntityV1` snapshots (`pinocchio/pkg/webchat/timeline_projector.go`). It covers:

- LLM start/delta/final events (assistant or thinking roles)
- Tool calls and results
- Planning and thinking‑mode middleware events

It is careful about consistency:

- Maintains per‑message role caches
- Throttles `llm.delta` writes to avoid DB churn

**Note:** The projector does **not** handle user messages. User messages are inserted via the `/chat` handler (see below), not via SEM frames.

## 1.6 User Messages: A Separate Path

In `startRunForPrompt`, after a user prompt is appended to the session, pinocchio manually inserts a user message entity into the timeline store:

```go
// pinocchio/pkg/webchat/router.go
_, _ = r.timelineStore.Upsert(..., &timelinepb.TimelineEntityV1{
  Id:   "user-" + turnID,
  Kind: "message",
  Snapshot: &timelinepb.TimelineEntityV1_Message{
    Message: &timelinepb.MessageSnapshotV1{Role: "user", Content: prompt},
  },
})
```

Important details:

- The entity ID is derived from `turnID` (e.g., `user-<turn>`).
- `CreatedAtMs` is **not** set; it defaults to `time.Now()` at upsert.
- This message is **not** emitted as a SEM frame.

Therefore the **user message’s position** in the durable snapshot is determined by the time at which the router writes it—not by any stream ordering or sequence number.

## 1.7 Frontend Ordering Rules

The frontend hydration path is simple but strict:

1) `/timeline` is fetched.
2) Entities are inserted in **server‑provided order** into a timeline slice.
3) The UI renders `state.order` directly (no re‑sorting by timestamps).

In `timelineSlice.ts`, the order array is insertion order, and upserts do **not** reorder items. (`pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`)

Therefore, the ordering policy of `/timeline` becomes the canonical UI order after reload.

## 1.8 Observed Ordering Risk

The pinocchio store uses millisecond timestamps. If two user messages land in the same millisecond, the full snapshot ordering falls back to `entity_id`. UUID‑ish IDs are not meaningful for chronological ordering, so ties can reorder messages.

**When are collisions likely?**

- rapid multi‑prompt submission,
- queued prompts being processed in a tight loop,
- offline persistence or batch insertion,
- concurrency in `Upsert` for the same conversation.

Because user messages are inserted outside the SEM stream, they lack any “global ordering” metadata that would align them with assistant events. This is the key difference from go‑go‑mento.

---

# Chapter 2 — Hydration in Go‑Go‑Mento (Textbook Treatment)

Go‑go‑mento treats hydration as a **standalone subsystem** with explicit interfaces and ordering guarantees. It separates event handling, projection, versioning, and persistence into distinct layers. The design is based on a principle common in event‑sourced systems: **stable IDs and monotonic versions** are the basis for deterministic reconstruction.

## 2.1 Core Abstractions

The hydration subsystem lives in `go/pkg/persistence/timelinehydration/` and consists of:

1) **Projector** — translates events into `ProjectedEntity` snapshots (`projector.go`).
2) **Service** — enforces monotonic versions (via a version tracker) and orchestrates persistence (`service.go`).
3) **Repository** — version‑aware upsert and query logic (Postgres SQL) (`repo.go`).
4) **Aggregator** — glue function that applies projection and persists synchronously; supports version override (`aggregator.go`).

These layers are wired into the webchat router via `streamOnEvent` (`go/pkg/webchat/conversation.go`).

## 2.2 Event Stream to Timeline: The Canonical Pipeline

The go‑go‑mento event pipeline is organized around **stable correlation IDs** and **monotonic versions**:

1) **Event ingest** via `StreamCoordinator` (Redis or in‑memory). (`go/pkg/webchat/stream_coordinator.go`)
2) **Redis XID extraction** (if present). Redis stream IDs are parsed into a version (`timestamp + seq`) and cached temporarily.
3) **Correlation** — `correlation.ExtractID` produces a stable entity ID for the event. (`go/pkg/webchat/correlation/extract.go`)
4) **Projection** — the `BasicProjector` builds a `ProjectedEntity` with `Version` and a payload snapshot.
5) **Service layer** — `Service.Upsert` ensures versions are monotonic per entity using a `versionTracker` and sets `started_at`/`updated_at`.
6) **Repository upsert** — Postgres upsert with `GREATEST(version)` ensures idempotence.

**Ordering invariant:** Each entity update has a monotonically non‑decreasing version; cross‑entity ordering is derived from `updated_at` and version.

## 2.3 Versioning and Redis Ordering

When Redis Streams are used, their XIDs carry ordering information (`milliseconds‑sequence`). The stream coordinator extracts XIDs and stores a temporary mapping from event ID to version (`event_version_cache.go`). The router’s `streamOnEvent` then applies this override via `timelinehydration.HandleEventWithVersion`.

This gives three critical properties:

- **Stable ordering within a millisecond:** the `seq` component breaks ties.
- **Consistent ordering across replay:** hydration from Redis or a persisted stream reuses the same XID ordering.
- **Graceful fallback without Redis:** if no XID is present, time‑based versions are still made monotonic by the version tracker.

This is a direct answer to the “not necessarily using redis transport” caveat: the system remains robust because the version tracker enforces monotonicity even without Redis.

## 2.4 User Messages: Persisted + Broadcast with Version

Go‑go‑mento persists user messages explicitly and broadcasts a `user.message` SEM frame with the same version/timestamp (`go/pkg/webchat/router.go`). The flow is:

- Build a user message entity with `NewUserMessageProjectedEntity` (role = user, status = completed).
- Persist it via `hydrSvc.Upsert`, using a version derived from `time.Now()` (or a Redis override if used).
- Emit a SEM frame that includes `version` and `timestamp`.

This ensures that *both* durable hydration and live streaming agree on user message ordering and IDs.

## 2.5 Repository Ordering and Incremental Hydration

The repository query for hydration is:

```sql
SELECT ... FROM wc_timeline_entities
WHERE timeline_id = $1
ORDER BY updated_at NULLS LAST, version
```

The frontend uses `selectTimelineMaxVersion` to track the max version already seen and requests incremental hydration via `sinceVersion`. (`web/src/store/timeline/timelineSlice.ts` + `web/src/hooks/useTimelineHydration.ts`)

Ordering is therefore determined by `updated_at` (which the service sets to the version), with `version` as a deterministic tie‑breaker. This is precisely the invariant pinocchio lacks.

## 2.6 Offline Hydration (CLI)

Go‑go‑mento also provides a CLI command (`mento-service webchat hydrate`) for offline replay from Redis or JSONL (`go/cmd/mento-service/cmds/webchat/hydrate.go`). The offline flow uses the same projector and service logic, so the **ordering semantics remain identical** to the live system.

This contributes to trust: the same ordering logic is exercised whether events arrive live or through replay.

---

# Comparative Diagnosis: What’s Missing or “Off” in Pinocchio

Below are the design differences that map directly to the reported problem (user message ordering after hydration):

## 3.1 Ordering Source

- **Pinocchio:** full snapshot ordering uses `created_at_ms`, which defaults to the first upsert time (millisecond precision); no explicit event ordering is carried into the durable store.
- **Go‑go‑mento:** ordering uses explicit versions derived from Redis XID or a monotonic tracker, and persists `updated_at` alongside `version`.

**Impact:** Pinocchio’s ordering is sensitive to millisecond collisions and to the *moment of persistence*, not the moment of the user’s action.

## 3.2 User Message Insertion Path

- **Pinocchio:** user messages are inserted directly in `/chat` via `TimelineStore.Upsert`, without SEM frames or version correlation.
- **Go‑go‑mento:** user messages are persisted and broadcast with an explicit version, part of the same ordering domain as other timeline entities.

**Impact:** Pinocchio user messages are not integrated into the stream ordering domain. Their ordering can drift relative to assistant messages and relative to each other.

## 3.3 Tie‑breaking

- **Pinocchio:** `ORDER BY created_at_ms, entity_id` (lexicographic) for full hydration.
- **Go‑go‑mento:** `ORDER BY updated_at, version` with monotonic versioning.

**Impact:** Lexicographic entity IDs can reorder messages in ways that are unrelated to time or user intent.

## 3.4 Redis Transport (Optional)

- **Pinocchio:** Redis (if enabled) is used for event transport but does not propagate an ordering version into the timeline store.
- **Go‑go‑mento:** Redis XID is explicitly translated into a version and used for ordering, while a version tracker handles non‑Redis cases.

**Impact:** Pinocchio loses the strongest available ordering signal when Redis is in use, and has no tie‑breaking strategy when it is not.

## 3.5 Frontend Ordering Semantics

Both systems preserve insertion order on the client. This makes server ordering **the single source of truth**.

**Therefore the server ordering rule is the root of the user‑message ordering bug.**

---

# Fix Blueprint: Bringing Pinocchio In Line with Go‑Go‑Mento

The right fix is to **define a single ordering domain** and use it consistently for all timeline entities, including user messages. The design below mirrors go‑go‑mento’s proven pattern.

## 4.1 Introduce a Monotonic Ordering Version

**Goal:** A single, monotonic value that determines timeline order and survives hydration.

Implementation idea (pinocchio‑friendly):

- Add a `version` (or `ordering_version`) to `TimelineEntityV1` or treat `CreatedAtMs` as the ordering version.
- On write, set `CreatedAtMs` (and optionally `UpdatedAtMs`) explicitly to the ordering version.
- When Redis is enabled, derive the ordering version from the Redis XID (`timestamp + seq`).
- When Redis is not enabled, use a per‑conversation monotonic counter or a time‑plus‑sequence scheme to avoid collisions.

This is the go‑go‑mento pattern: time‑based versions + version tracker + optional Redis override.

## 4.2 Persist User Messages Through the Same Ordering Path

Two viable strategies:

1) **Emit user message SEM frames** (like go‑go‑mento’s `user.message`) so the timeline projector handles them in the same pipeline as other events. The frame should carry an explicit `version` (derived from the same ordering domain).

2) **Keep the direct `Upsert` but assign explicit version + created_at:**
   - Compute `version := nextTimelineVersion()` (or XID‑based).
   - Call `TimelineStore.Upsert` with `CreatedAtMs = version` and `UpdatedAtMs = version`.

Either approach removes the implicit “time of persistence” ordering.

## 4.3 Change Full Snapshot Ordering to Use Version

Today, full snapshots use `created_at_ms`. If you treat `created_at_ms` as the ordering version, then this can remain. But the ordering field must be monotonic.

An alternate approach is to **store a dedicated `version` column** and order by it for full snapshots (mirroring go‑go‑mento). This is more explicit and easier to reason about.

## 4.4 Add a Version Tracker for Non‑Redis Modes

When Redis is not used, timestamps can collide. A lightweight version tracker (in‑memory per process) ensures monotonicity and prevents same‑ms reorderings.

Go‑go‑mento’s `versionTracker` is a minimal template for this: if a candidate version is <= the previous value, increment it.

## 4.5 Make Ordering Observable and Debuggable

Expose in `/timeline`:

- `since_version` semantics (already present),
- `snapshot.version` representing the latest ordering value,
- `created_at_ms`/`updated_at_ms` explicitly set for every entity.

This makes it easy to audit ordering on the client and in logs.

---

# Probable Root Cause for the Reported Bug

The likely cause of “user message ordering off after DB hydration” is:

1) **User messages are inserted in a separate path** with `CreatedAtMs` defaulted to `time.Now()` at persistence time.
2) **Full snapshot ordering uses `created_at_ms`**, which is not guaranteed to reflect user message submission order, especially under rapid submissions or queueing.
3) **If two user messages share the same millisecond**, the ordering falls back to `entity_id` (lexicographic), which is arbitrary with respect to chronology.

This is consistent with the observation that ordering problems are more likely when Redis transport is not used (no external ordering signal) and when user messages are added to hydration as a separate path.

---

# Actionable Implementation Checklist (Pinocchio)

1) **Introduce a per‑conversation monotonic ordering version** (time + seq or counter).
2) **Use that version as `CreatedAtMs` (and `UpdatedAtMs`) for all timeline entities**, including user messages inserted via `/chat`.
3) **Optionally add `user.message` SEM frames** and teach `TimelineProjector` to project them.
4) **If Redis transport is enabled, derive versions from Redis XIDs** (as go‑go‑mento does in `stream_coordinator.go`).
5) **Adjust `/timeline` ordering** to rely on the monotonic version rather than lexical entity IDs.

This preserves the existing UI assumptions (insertion order is authoritative) while ensuring that hydration order is deterministic and correct.

---

# Appendix: Key Files Consulted

## Pinocchio

- `pinocchio/pkg/webchat/router.go` — `/hydrate`, `/timeline`, user message upsert in `startRunForPrompt`
- `pinocchio/pkg/webchat/timeline_store_sqlite.go` — SQLite schema, ordering rules
- `pinocchio/pkg/webchat/timeline_projector.go` — SEM → timeline entities
- `pinocchio/pkg/webchat/conversation.go` — stream wiring, SEM buffer, timeline projector hook
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts` — hydration gating and snapshot application
- `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts` — insertion order semantics

## Go‑Go‑Mento

- `go-go-mento/go/pkg/persistence/timelinehydration/{aggregator,projector,service,repo}.go`
- `go-go-mento/go/pkg/webchat/{conversation,router,stream_coordinator,event_version_cache}.go`
- `go-go-mento/web/src/hooks/useTimelineHydration.ts`
- `go-go-mento/web/src/store/timeline/timelineSlice.ts`
- `go-go-mento/docs/reference/persistence/timeline-hydration.md`
- `go-go-mento/go/cmd/mento-service/cmds/webchat/hydrate.go`
