---
Title: Hydration race-free ordering and resume strategy
Ticket: GP-011-CHECK-HYDRATION
Status: active
Topics:
    - hydration
    - webchat
    - persistence
    - events
    - backend
    - frontend
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-mento/go/pkg/persistence/timelinehydration/service.go
      Note: Monotonic version tracker
    - Path: go-go-mento/go/pkg/webchat/stream_coordinator.go
      Note: Redis XID to version mapping for resume
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Current hydration/WS sequencing and lack of resume handshake
    - Path: pinocchio/pkg/webchat/router.go
      Note: Hydration endpoints and timeline snapshot watermark candidate
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: Ordering and version fields in durable snapshot
ExternalSources: []
Summary: Answer whether monotonic ordering alone avoids hydration races and propose a resume strategy that prevents event loss during snapshot+stream overlap.
LastUpdated: 2026-01-25T10:05:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Hydration Race‑Free Ordering and Resume Strategy

## 1. Question Restated (and Short Answer)

**Question:** “Will monotonic ordering by id avoid hydration race conditions when restoring from disk while new SEM events are arriving? Can the websocket ask for events starting at `hydration.last_seen_sem_id`?”

**Short answer:** **Monotonic ordering is necessary but not sufficient.** Ordering lets you *merge* a snapshot with live events deterministically, but it does not guarantee you *received every event* that occurred during the race window. To be race‑free, you also need a **resume mechanism with retention** (buffer/log) and a **snapshot watermark** (a version/seq that the snapshot is guaranteed to include). With those two pieces, “start at last_seen” becomes correct.

If the server does not retain events, no ordering scheme alone can guarantee that the client didn’t miss events between “snapshot read” and “stream subscribed.”

## 2. Why Ordering Alone Doesn’t Eliminate Races

### 2.1 The Classic Race Window

There is a window between:

1. **Snapshot read** (hydrate from DB), and
2. **Live stream subscription** (WebSocket open).

Any events that arrive in that window can be lost **unless** the server replays them. Monotonic ordering can tell you *where* those events should fit, but not *whether you got them*.

### 2.2 Loss vs. Misorder

- **Ordering solves misorder:** If every event has a monotonic version, you can sort or ignore duplicates.
- **Retention solves loss:** If you can request “events since X,” the server can fill the gap.

Therefore you need **both**.

## 3. Minimum Invariants for Race‑Free Hydration

To make the “start at last_seen_sem_id” idea correct, the system must ensure:

1) **Snapshot watermark:** The hydration response must include a `snapshot_version` (or `last_seq`) representing the highest version/seq that is guaranteed included in the snapshot.

2) **Durable or buffered event log:** The server must retain events for at least a short window so it can replay all events with `version > snapshot_version`.

3) **Monotonic versioning:** Every event (including user messages) must carry a monotonically increasing version in the same ordering domain.

If any of these are missing, you can still experience gaps.

## 4. Design Options (with Pros/Cons)

### Option A — Snapshot + Resume from Server Buffer (Short‑term Retention)

**Mechanism:**
- Server keeps a per‑conversation ring buffer of SEM frames with versions.
- `/timeline` returns `snapshot_version`.
- WS handshake includes `since_version`, server replays frames from buffer and then streams live.

**Pros:**
- Minimal infrastructure changes.
- Works without Redis.

**Cons:**
- Buffer is finite; long disconnects can still lose events.

**When acceptable:**
- If you only need to survive short hydration gaps (seconds/minutes), not long offline periods.

### Option B — Redis Stream (Durable Event Log)

**Mechanism:**
- Treat Redis stream as the authoritative log.
- Snapshot returns `last_xid` or derived `snapshot_version`.
- WS resume requests events from Redis starting after that XID.

**Pros:**
- Strong ordering and durable retention.
- Natural resume semantics.

**Cons:**
- Requires Redis transport and XID plumbing to the client.

### Option C — DB Event Log (Append‑Only Table)

**Mechanism:**
- Persist SEM events to an append‑only table with a monotonic sequence.
- `/timeline` is a projection; resume reads the event log from `since_version`.

**Pros:**
- Fully durable; no external system required.

**Cons:**
- More storage and operational complexity.

### Option D — “Connect WS First, Then Hydrate” (Client‑buffered)

**Mechanism:**
- Client opens WS first and buffers all incoming events locally.
- Client then hydrates the snapshot.
- Client applies buffered events after `snapshot_version`.

**Pros:**
- No server retention needed.

**Cons:**
- Only race‑free if the WS connection is established *before* the snapshot; events that arrive before WS connect are still at risk. Not safe if network latency is high or WS connect is delayed.

## 5. Recommended Approach (Safe and Practical)

A combined **Snapshot + Resume** design is the most robust and aligns with the original go‑go‑mento intent:

### 5.1 Protocol Contract

1) **Hydration response** (`/timeline` or `/hydrate`):
   - includes `snapshot_version` (monotonic ordering domain)
   - includes `server_time` for debugging

2) **WebSocket connect** includes:
   - `since_version = snapshot_version`
   - optional `resume_token` if you support reconnections

3) **Server behavior:**
   - On connect, replay all events with `version > since_version` from its buffer or log
   - Then switch to live streaming

This makes the client’s “start at last_seen” idea correct, but only because the server promises to replay from a retained stream.

### 5.2 Monotonic Version Definition

Use one ordering domain across:

- timeline entities (DB snapshots)
- SEM frames (WS)
- user messages

Suggested version sources:

- **Redis XID** when available (timestamp + seq) — strongest ordering.
- **Local monotonic counter** per conversation otherwise (time + sequence or atomic counter).

The key is not “ordering by ID,” but “ordering by monotonic version that is shared between snapshot and stream.”

## 6. Answering the Specific Idea: “events start at hydration.last_seen_sem_id”

This is **correct** if and only if:

- `last_seen_sem_id` is a **monotonic version** (not a UUID).
- The server **retains events** for replay after that version.
- The snapshot is guaranteed to include all events up to that version.

If any of these are not true, you can still miss events and the system is not race‑free.

## 7. Practical Implications for Pinocchio

Pinocchio currently has:

- **/hydrate** (SEM buffer replay) with `last_seq` but no server‑side resume handshake for WS.
- **/timeline** (SQLite snapshots) ordered by `created_at_ms`, not a strict monotonic version.
- User messages inserted via `/chat` without a shared version.

To support race‑free hydration, pinocchio needs:

1) **A shared monotonic version** for all entities and SEM frames.
2) **A snapshot watermark** returned by `/timeline`.
3) **WS resume** using `since_version` and a server replay buffer/log.

If Redis is enabled, reuse Redis XID → version (like go‑go‑mento). If not, introduce a per‑conversation version counter.

## 8. Suggested Implementation Sketch (Pinocchio)

### 8.1 Add a Version Source

- Introduce `nextVersion(convID)` that returns a monotonic value.
- If Redis stream XIDs exist, use those as the version.

### 8.2 Write Versions into the Timeline Store

- `TimelineStore.Upsert` should take `CreatedAtMs/UpdatedAtMs` explicitly as `version`.
- For user messages inserted via `/chat`, set `CreatedAtMs = version`.

### 8.3 Include `snapshot_version` in `/timeline`

- Populate snapshot with the max version in the store.

### 8.4 WS Resume Handshake

- Add `?since_version=` to `/ws` or a `resume` message on connect.
- Server replays buffered SEM frames with `version > since_version`.
- Then stream live frames.

## 9. Edge Cases and Pitfalls

- **Clock skew:** If versions are time‑based, multiple nodes with skew can misorder. Prefer Redis XID or a centralized counter.
- **Partial snapshots:** If snapshot is not transactionally consistent, its watermark can be wrong. Store the max version in the same transaction.
- **Duplicate delivery:** Clients should de‑dup by version+entity ID; expected in replay scenarios.

## 10. Conclusion

Monotonic ordering by itself does not eliminate hydration race conditions. It is a *necessary* property for merging state, but it is **not a delivery guarantee**. To be race‑free, you must pair monotonic ordering with a **resume protocol** and **event retention**. If you can supply those, then “start at hydration.last_seen_sem_id” is exactly the right idea and becomes a correct, robust solution.

---

## Open Questions (to resolve before implementation)

1) Do we want to make Redis mandatory for race‑free hydration, or do we accept a bounded in‑memory replay window?
2) Should `/timeline` be strongly consistent (transactional snapshot with watermark)?
3) What is the maximum tolerated offline gap for resume (minutes, hours)?

These answers determine whether we need a durable event log vs a short ring buffer.
