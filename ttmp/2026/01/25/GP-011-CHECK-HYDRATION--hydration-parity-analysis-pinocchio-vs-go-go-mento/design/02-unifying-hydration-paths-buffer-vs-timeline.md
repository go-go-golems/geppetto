---
Title: Unifying hydration paths (buffer vs timeline)
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
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Hydration gating and ordering on client
    - Path: pinocchio/pkg/webchat/sem_buffer.go
      Note: In-memory SEM replay buffer
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Projection into timeline entities
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: Durable snapshot store and ordering
ExternalSources: []
Summary: Analysis of whether to unify pinocchio’s /hydrate (SEM buffer) and /timeline (durable projection) paths, with design options and recommendations.
LastUpdated: 2026-01-25T10:12:00-05:00
WhatFor: "Define a single hydration path with canonical timeline projections and in-memory fallback when no DB is configured."
WhenToUse: "Use when consolidating hydration logic, removing /hydrate, or deciding how to keep SEM frames while using timeline projections as the source of truth."
---

# Unifying Hydration Paths (Buffer vs Timeline)

## 1. Problem Statement

Pinocchio currently has two hydration paths:

- **/hydrate** — in‑memory buffer of SEM frames (transient replay)
- **/timeline** — durable SQLite projection snapshots (canonical state)

This creates dual representations and two ordering semantics. The question is whether we can unify these paths and use a single representation that is efficient, race‑free, and consistent across live and restored sessions.

## 2. Why There Are Two Paths Today

Each path addresses different constraints:

### /hydrate (SEM frames, in‑memory)
- **Pros:**
  - Minimal latency and no DB dependencies.
  - Replay of *exact* event stream (great for UI correctness in the moment).
  - Works when the durable timeline store is disabled.
- **Cons:**
  - Not durable; limited buffer window.
  - SEM frames are verbose; hard to query and merge without order metadata.

### /timeline (projected entities, durable)
- **Pros:**
  - Durable; good for reload and offline resume.
  - Compact entity state suitable for UI reconstruction.
- **Cons:**
  - Requires projections and ordering contracts.
  - Potential mismatch vs live SEM stream unless ordering is aligned.

The split exists because the timeline projection is optional, and SEM replay is a cheap fallback.

## 3. Can We Unify? Yes, But Only if One Representation Becomes Canonical

Unification is feasible if we pick **one canonical representation** and ensure it supports both:

- **Fast, incremental live updates** (streaming)
- **Durable, reload‑safe snapshots** (hydrate)

There are two viable approaches:

### Approach A — Canonical Event Log (SEM frames) + Projected Cache

Use SEM frames as the only “truth,” and derive projections (entities) on demand or from cache.

**How it works:**
- Persist an append‑only SEM log (Redis stream or DB table).
- Hydration reads from that log and replays into the frontend (or into a server‑side projector).
- Optionally keep an in‑memory projection cache to serve fast `/timeline` responses.

**Pros:**
- Single source of truth.
- Simple mental model: replay events to rebuild state.

**Cons:**
- Heavy to hydrate if log is large.
- Requires retention and replay controls.

**Best if:** you want full event‑sourcing and are ok with replay costs or have efficient snapshotting.

### Approach B — Canonical Projection (timeline entities) + Event Stream as Delta

Make the **timeline projection** the canonical representation and treat SEM frames as transient delivery.

**How it works:**
- All events update the timeline store with monotonic versions.
- `/timeline` becomes the single hydration interface.
- The WS stream carries delta events that map directly to timeline updates.

**Pros:**
- Hydration is always fast; no replay for long histories.
- Frontend doesn’t need to understand raw SEM to rebuild state.

**Cons:**
- Requires strong ordering/version contract.
- Losing the exact event stream may reduce debugging fidelity.

**Best if:** UI correctness and fast reloads are more important than replaying raw events.

## 4. Size‑Limited In‑Memory Projection Cache (Your Suggestion)

Yes: you can maintain a **size‑limited cache of projected entities** in memory and use it for hydration when the durable store is disabled or for fast rehydration in the hot path.

### Two variants

1) **In‑memory projection cache only (no DB):**
   - Simpler but not durable.
   - Suitable for development or ephemeral sessions.

2) **Hybrid cache + DB:**
   - DB is durable source of truth.
   - Cache mirrors most recent entities for faster hydration.
   - Cache can be keyed by version and size‑limited (LRU).

### What this buys you
- **Single representation** for hydration and live updates (entities)
- Potentially eliminate `/hydrate` entirely, or use it only as a debug endpoint

### What it doesn’t buy you
- **Race‑free delivery** unless you also implement versioned resume (see prior design doc).

## 5. Suggested Unification Strategy (Your Decision)

### Recommendation: Canonical Projection + Monotonic Versioning + In‑Memory Store Fallback

Make `/timeline` the universal hydration path, and unify live updates with the same representation:

1) **All events update the timeline store** with monotonic versions.
2) **The timeline store always exists** (durable DB when configured; **in‑memory store** otherwise).
3) **WS stream sends timeline‑delta events** (entity upserts with version).
4) **Client merges deltas with snapshot** using version + entity ID.
5) **/hydrate is removed** (single hydration path: `/timeline`).

This yields one consistent path across dev and prod while preserving live SEM streaming.

### In‑Memory Store as the Default (No‑DB Mode)

When no durable DB is configured, use an in‑memory `TimelineStore` that implements the same interface and ordering rules as the SQLite/Postgres store. This becomes the **default store**, not merely a cache.

Key properties:
- **Same schema semantics:** `conv_id`, `entity_id`, `version`, `created_at_ms`, `updated_at_ms`.
- **Size limits:** enforce per‑conversation and global caps (LRU or ring eviction).
- **Identical ordering:** full snapshot order and incremental order must match the durable store.

This guarantees a single hydration path in all environments.

## 6. What to Do with SEM Frames

Even if you unify hydration on projections, SEM frames still provide value:

- Fine‑grained UI streaming (token deltas)
- Debugging and instrumentation

Two options:

1) **Keep SEM frames for live UI only** (not for hydration). They are ephemeral.
2) **Translate SEM frames into timeline deltas** and send those over WS, possibly in addition to the raw SEM frames for debugging.

Option 2 offers a clean “single source of truth” for state changes, with SEM as optional diagnostics.

## 7. Risks & Tradeoffs

- **Projection drift:** If timeline projector misses an event type, UI state will diverge. Needs testing.
- **Increased server work:** Continuous projection + DB writes for streaming deltas.
- **Loss of raw stream:** If you remove SEM replay entirely, you lose a detailed audit trail (unless you persist it).

## 8. Concrete Options Matrix

| Option | Canonical Representation | Hydration Path | Live Updates | Durability | Complexity |
|---|---|---|---|---|---|
| Current | SEM buffer + timeline | /hydrate + /timeline | SEM frames | Partial | Medium |
| A | SEM log | Replay SEM | SEM frames | Strong if log | High |
| B (recommended) | Timeline projection | **/timeline only** | Timeline deltas + SEM | Strong (durable) / Ephemeral (in‑mem) | Medium |
| Hybrid (cache) | Timeline + in‑mem cache | /timeline from cache | Timeline deltas + SEM | Strong | Medium‑High |

## 9. Recommended Next Steps (Pinocchio)

1) **Define a version contract** (monotonic version for all entities + user messages).
2) **Make timeline projection the canonical hydration representation.**
3) **Send WS deltas that mirror timeline updates** (alongside SEM).
4) **Implement an in‑memory TimelineStore** used when no DB is configured.
5) **Remove /hydrate** (single hydration path: `/timeline`).

## 10. Clarification: Keep SEM Frames + Canonical Timeline Projections

Per your decision:

- **SEM frames stay** as a live event stream (for streaming UX and debugging).
- **Timeline projections are canonical** for hydration and durable state.

This implies a dual‑stream contract, but not dual sources of truth:

1) **Canonical source of truth:** timeline entities with monotonic versions.
2) **Live stream (SEM):** ephemeral, but **must be mappable** to timeline updates.

### Practical implications

- **Every SEM frame that affects UI state must correspond to a timeline entity update** (same entity ID + version).
- **SEM stream can deliver fine‑grained deltas** (token streaming, tool progress) without being authoritative for hydration.
- **Clients merge by version**: the timeline snapshot establishes state; SEM frames are applied only if their version is newer.

### Suggested wire behavior (single path)

- `/timeline` always exists and returns `snapshot_version` (from durable or in‑memory store).
- WS connects and continues to stream **SEM frames**.
- Server also emits compact **timeline‑deltas** alongside SEM for deterministic merges.

### Why this still unifies hydration

Hydration is unified on **one representation** (timeline entities), while SEM is treated as a **delivery format**, not a second state store. The SEM stream remains valuable for UX and debugging, but correctness and ordering are governed by the timeline version contract and the (durable or in‑memory) timeline store.
