---
Title: 'Webchat package reorg: structure and dependency proposal'
Ticket: PI-005-WEBCHAT-REORG
Status: active
Topics:
    - webchat
    - refactor
    - analysis
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/connection_pool.go
      Note: Runtime connection management for websocket clients
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: |-
        Conversation lifecycle and runtime wiring
        Conversation runtime wiring
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: |-
        Engine composition logic and middleware/tool wiring
        Engine composition
    - Path: pinocchio/pkg/webchat/engine_config.go
      Note: EngineConfig signature and settings metadata
    - Path: pinocchio/pkg/webchat/engine_from_req.go
      Note: |-
        Request policy resolver for conv/profile/overrides
        Request policy
    - Path: pinocchio/pkg/webchat/router.go
      Note: |-
        HTTP/WS entrypoints and orchestration
        HTTP/WS orchestration
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: |-
        Event -> SEM frame translation
        SEM translation
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Event consumption and stream fan-out
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: |-
        SEM frames -> timeline projection
        Timeline projection
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: SQLite-backed timeline store
ExternalSources: []
Summary: |
    Research and proposal for reorganizing pinocchio/pkg/webchat into coherent
    subpackages (engine/sem/timeline/runtime/http) without import cycles.
LastUpdated: 2026-01-28T11:10:21-05:00
WhatFor: Provide a dependency-safe package map and migration plan for reorganizing webchat.
WhenToUse: Before refactoring the webchat package layout.
---


# Webchat Package Reorg: A Textbook Treatment

> "Architecture is the art of making the right dependencies possible, and the wrong dependencies impossible."

This document analyzes the current `pinocchio/pkg/webchat` package and proposes a clean reorganization into focused subpackages (`engine/`, `sem/`, `timeline/`, `runtime/`, `http/`). The goal is to improve comprehensibility and maintainability **without introducing import cycles**.

The proposal is conservative: it reorganizes by existing functional boundaries and introduces explicit dependency rules rather than refactoring behavior.

---

## 1) Problem Statement

The `webchat` package has grown into a monolith containing:

- Engine composition and request policy logic
- Conversation runtime, queueing, and connection management
- Semantic event translation and streaming infrastructure
- Timeline projection and persistence
- HTTP/WS routing

All of this currently sits in a single package, which makes it easy to create accidental coupling. The objective is to:

1) Split the package into coherent subpackages,
2) Keep dependencies unidirectional, and
3) Preserve the current behavior with minimal churn.

---

## 2) Current File Inventory (Clusters Emerge Naturally)

The existing files already fall into clear conceptual clusters:

### Engine Policy and Composition
- `engine_builder.go`
- `engine_config.go`
- `engine_from_req.go`
- `engine.go`
- `loops.go`

### Conversation Runtime
- `conversation.go`
- `connection_pool.go`
- `send_queue.go`

### Semantic Events and Streaming
- `sem_translator.go`
- `sem_buffer.go`
- `stream_coordinator.go`

### Timeline and Projection
- `timeline_projector.go`
- `timeline_store.go`
- `timeline_store_memory.go`
- `timeline_store_sqlite.go`
- `timeline_upsert.go`

### HTTP/Server
- `router.go`
- `server.go`

### Shared Types
- `types.go`

This is a strong signal: the package already *wants* to be separated.

---

## 3) Dependency Constraints (The Non‑Negotiables)

To avoid import cycles, we need explicit layering rules. A practical rule set is:

```
http -> runtime -> engine
http -> runtime -> sem
http -> runtime -> timeline
sem  -> (no runtime, no http)
engine -> (no runtime, no http)
timeline -> sem (optional), but not runtime/http
```

**Why these constraints matter:**
- `runtime` must orchestrate engines, sem streams, and timeline projection.
- `sem` should be a pure translation layer, not aware of runtime or HTTP.
- `timeline` may need to parse SEM frames, but should not depend on runtime types.
- `http` is the top layer and can depend on everything else.

These rules make it impossible to create a cycle if followed strictly.

---

## 4) Proposed Package Layout

### 4.1 New Package Tree

```
pinocchio/pkg/webchat/
  http/
    router.go
    server.go
  engine/
    builder.go
    config.go
    from_req.go
    loops.go
    engine.go
  runtime/
    conversation.go
    connection_pool.go
    send_queue.go
  sem/
    translator.go
    buffer.go
    stream_coordinator.go
  timeline/
    projector.go
    store.go
    store_memory.go
    store_sqlite.go
    upsert.go
  types.go (or moved into engine/runtime as appropriate)
```

### 4.2 Where `types.go` Goes

`types.go` defines `Profile`, `Router`, and a few core interfaces. This is the main risk for cycles.

**Recommendation:**
- Split `types.go` into:
  - `engine/types.go` (profiles, middleware/tool registries)
  - `http/router_types.go` (router struct and config)
  - `runtime/types.go` (conversation interface)

If you keep `types.go` at the root, it must not import subpackages. It should only define pure types.

---

## 5) Proposed Dependency Graph (No Cycles)

```
             ┌──────────────┐
             │   http       │
             └──────┬───────┘
                    │
        ┌───────────┼───────────┐
        ▼           ▼           ▼
    runtime       engine       sem
        │           │           │
        └───────────┼───────────┘
                    ▼
                 timeline
```

Interpretation:

- `http` depends on everything (top-level orchestration).
- `runtime` depends on `engine`, `sem`, and `timeline`.
- `timeline` may depend on `sem` if it needs SEM parsing helpers.
- `engine` and `sem` remain independent from runtime and http.

If `timeline` can be made independent of `sem` by accepting decoded events or an interface, that is even cleaner.

---

## 6) File‑to‑Package Mapping (Concrete Proposal)

| Current File | New Package | Rationale |
| --- | --- | --- |
| `router.go` | `webchat/http` | HTTP/WS entrypoints, routing, orchestration |
| `server.go` | `webchat/http` | Server wrapper around router |
| `conversation.go` | `webchat/runtime` | Conversation lifecycle, engine wiring |
| `connection_pool.go` | `webchat/runtime` | WebSocket connection management |
| `send_queue.go` | `webchat/runtime` | Per-conversation queueing/idempotency |
| `engine_builder.go` | `webchat/engine` | Engine composition logic |
| `engine_config.go` | `webchat/engine` | Config + signature |
| `engine_from_req.go` | `webchat/engine` | Request policy resolution |
| `engine.go` | `webchat/engine` | Engine wiring helpers |
| `loops.go` | `webchat/engine` | Run loop strategies |
| `sem_translator.go` | `webchat/sem` | Event -> SEM translation |
| `sem_buffer.go` | `webchat/sem` | Buffered SEM frames |
| `stream_coordinator.go` | `webchat/sem` | Event consumption + fan-out |
| `timeline_projector.go` | `webchat/timeline` | SEM -> timeline projection |
| `timeline_store*.go` | `webchat/timeline` | Timeline persistence |
| `timeline_upsert.go` | `webchat/timeline` or `webchat/http` | Hook emission; see below |
| `types.go` | split | avoid cycles |

### Timeline Upsert Placement

`timeline_upsert.go` currently attaches to `Router` (HTTP layer). Consider splitting:

- `timeline` package owns projection + store
- `http` package emits "timeline.upsert" events when configured

If `timeline_upsert.go` depends on router or HTTP concerns, keep it in `http` (or `runtime`) rather than `timeline`.

---

## 7) Import Cycle Risk Analysis

The most likely cycle risks come from:

1) `Router` type living in the root package and embedding runtime/engine concerns.
2) `TimelineProjector` needing to call back into router or runtime.
3) `EngineFromReqBuilder` reaching into conversation lookup interfaces.

### Mitigation Patterns

**Pattern A: Narrow interfaces**

Example:

```go
type ConversationLookup interface {
    GetConversation(convID string) (*Conversation, bool)
}
```

Ensure `engine` only depends on small interfaces, not runtime packages.

**Pattern B: Inversion via callbacks**

Have timeline projectors accept callback functions instead of importing router/runtime types.

**Pattern C: Move shared types down**

Define shared interfaces in a low-level `webchat/core` or `webchat/types` package that has *no dependencies*.

---

## 8) Proposed Refactor Sequence (Low Risk)

1) **Create subpackages** with type aliases or re-export stubs (optional).
2) Move sem files into `webchat/sem` and update imports.
3) Move timeline files into `webchat/timeline` and update imports.
4) Move engine files into `webchat/engine` and update imports.
5) Move runtime files into `webchat/runtime` and update imports.
6) Move router/server into `webchat/http`.
7) Split `types.go` last (most sensitive).

This order reduces cycle risk and keeps the most central files for last.

---

## 9) Optional Enhancements (If Desired)

If you want even cleaner boundaries:

- Introduce `webchat/core` with shared interfaces (`Profile`, `ConversationLookup`, `TimelineStore`).
- Rename `EngineFromReqBuilder` to `RequestPolicyResolver` to clarify its role.
- Make `timeline` independent of SEM by accepting a parsed event struct rather than raw frames.

These are not required for the reorg, but they make the package structure even more self‑describing.

---

## 10) Summary: The Design in One Paragraph

The `webchat` package can be cleanly split into five subpackages—`http`, `runtime`, `engine`, `sem`, and `timeline`—with explicit unidirectional dependencies that prevent import cycles. The `http` layer orchestrates everything, the `runtime` layer binds engine and streaming to conversation lifecycle, `engine` handles configuration and composition, `sem` handles translation and streaming, and `timeline` handles persistence and projection. The key risk is shared types; splitting or isolating them avoids cycles. This reorg requires minimal behavioral change and primarily improves clarity and modularity.
