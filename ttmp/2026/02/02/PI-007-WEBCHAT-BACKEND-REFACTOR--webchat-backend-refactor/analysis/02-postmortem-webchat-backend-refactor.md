---
Title: 'Postmortem: Webchat Backend Refactor'
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
    - Path: pinocchio/pkg/doc/topics/webchat-frontend-integration.md
      Note: Doc updates summarized
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Seq derivation + fallback described
    - Path: pinocchio/pkg/webchat/timeline_store.go
      Note: Explicit versioning in postmortem
ExternalSources: []
Summary: Detailed engineering postmortem of the webchat backend refactor, including subsequent hydration ordering bugfixes.
LastUpdated: 2026-02-03T21:30:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Postmortem: Webchat Backend Refactor

## Executive Summary

We refactored the webchat backend to simplify routing, make the UI/API separation explicit, centralize conversation lifecycle responsibilities, and harden streaming/timeline behavior under load. The work removed legacy compatibility shims, clarified time-based ordering of SEM events, and introduced predictable eviction and non-blocking WebSocket broadcasting.

The refactor exposed (and then fixed) a hydration ordering bug: when Redis stream IDs were absent, assistant events were assigned tiny local sequences while user messages used time-based versions, causing user messages to appear at the bottom after hydration. We corrected the StreamCoordinator fallback sequence generation to use time-based monotonic values, restoring consistent ordering.

## Goals

- Simplify webchat routing and handler ownership without backwards-compatibility shims.
- Make UI and API handlers composable and mountable under arbitrary roots.
- Consolidate conversation lifecycle operations in a single manager.
- Ensure streaming behavior is robust under slow clients and long-lived sessions.
- Align timeline storage versions with stream ordering so hydration reflects true message order.

## Scope and Timeline

- **Refactor work**: PI-007-WEBCHAT-BACKEND-REFACTOR
- **Bugfix follow-on**: PI-008-ADDRESS-WEBCHAT-BUGS

## Major Changes

### 1) Router Mounting and Handler Separation

**Problem**: The previous routing layout mixed UI and API handler setup and relied on ambiguous mounting behavior. Root prefix handling could leave paths broken and required caller-side fixes.

**Change**:
- `Router.Mount` now wraps handlers with `http.StripPrefix`, and a base path redirect ensures `/prefix` → `/prefix/` works.
- Added explicit `UIHandler()` and `APIHandler()` for callers who want to split UI and backend endpoints.
- `NewServer` / `NewRouter` now accept an `fs.FS` instead of internal path assumptions.

**Why it mattered**:
- Enables embedding and reverse-proxying under a prefix without path hacks.
- Allows hosting UI assets separately while still using the same backend.

### 2) ConversationManager Ownership

**Problem**: The router owned too much conversation lifecycle logic (creation, connection management, cleanup), which made it hard to reason about concurrency and ownership.

**Change**:
- `ConvManager` now owns `GetOrCreate`, `AddConn`, and `RemoveConn` responsibilities.
- Router delegates to the manager for lifecycle operations.

**Why it mattered**:
- Single ownership point for conversation lifecycle reduces duplication and subtle race potential.

### 3) Send Queue Refactor

**Problem**: Queue lifecycle logic was embedded in router code, and behavior depended on implicit state ordering.

**Change**:
- Introduced `PrepareRun`, `ClaimNextQueued`, `RunPreparation` and related utilities in `send_queue.go`.
- Router became a thin orchestrator rather than the queue implementation.

**Why it mattered**:
- Clarifies how queued runs start, how idempotency is enforced, and how queue state is observed.

### 4) Stream Ordering via Redis IDs (and Time-Based Fallback)

**Problem**: Without Redis stream metadata, `event.seq` fell back to a tiny local counter. Timeline versions for assistant events then sorted before time-based user-message versions, breaking hydration ordering.

**Change**:
- StreamCoordinator now derives `event.seq` from Redis stream IDs when available.
- When not available, it falls back to a **time-based monotonic seq** (`UnixMillis * 1_000_000`) instead of a local counter.

**Why it mattered**:
- Aligns the version scale for assistant and user events, preserving correct ordering both in live streams and on hydration.

### 5) TimelineStore: Explicit Versions Only (No Backwards Compatibility)

**Problem**: The timeline store previously auto-incremented versions, which could diverge from stream ordering. The user explicitly requested no backwards compatibility shims.

**Change**:
- `TimelineStore.Upsert` now **requires** an explicit version (from `event.seq`).
- SQLite and in-memory implementations accept the version and use it directly.
- `TimelineProjector` extracts `event.seq` from SEM frames and passes it into the store.
- Manual user-message inserts use a time-based seq value.

**Why it mattered**:
- Ensures timeline versions reflect the same ordering as streaming events and the UI merge logic.

### 6) Eviction Loop and ConnectionPool Resilience

**Problem**: Idle conversations were not evicted deterministically, and slow clients could stall broadcasts.

**Change**:
- Added eviction config in `ConvManager` with periodic scanning.
- WebSocket ConnectionPool now uses per-connection writer goroutines and buffered queues, with drop-on-full backpressure.

**Why it mattered**:
- Prevents memory leaks from idle conversations.
- Guarantees that one slow client cannot stall the entire conversation stream.

## Bugfixes and Validation

### Hydration Ordering Bug (PI-008)

**Symptom**: On hydration, all user messages appeared at the bottom, and first user input could appear below the first assistant response.

**Root cause**:
- Assistant events were stored with tiny sequence values (1, 2, 3…) when stream IDs were missing.
- User messages were stored with large time-based version values.
- SQLite ordered by version, so assistants were always listed first.

**Fix**:
- StreamCoordinator now uses a time-based, monotonic sequence in the fallback path.

**Validation**:
- Playwright-driven UI test: user message appears above assistant response before and after hydration.
- SQLite inspection: both user and assistant versions are time-based in fresh conversations.

## Documentation Updates

Documentation was refreshed across the webchat suite to match the refactor and bugfixes:

- `/timeline` is now the canonical hydration path.
- `event.seq` behavior and fallback ordering is documented.
- StreamCoordinator and ConnectionPool behaviors now match actual code.
- Backend reference no longer mentions stale translator patterns.

## Testing and Tooling

- Targeted unit tests for StreamCoordinator sequence derivation and fallback behavior.
- Timeline store tests updated to require explicit versions.
- Pre-commit hooks ensured `go test`, `go generate`, `go build`, `golangci-lint`, and `go vet` ran on changes.

## Risks and Trade-offs

- **No backwards compatibility**: Some legacy behaviors (like `/hydrate` or implicit timeline versions) were removed. This was intentional and documented, but may surprise older integrations.
- **Sequence monotonicity**: When mixing Redis-derived seq with time-based fallback, ordering is enforced by monotonic guard rails rather than strict adherence to stream IDs. This favors UI ordering correctness over preserving raw stream ID values.

## Lessons Learned

- When storage ordering is tied to stream ordering, **version scale** must be consistent across all producers. Small integer fallbacks are not compatible with time-based versions.
- Handler separation (UI vs API) makes embedding and proxying much simpler, but requires deliberate documentation updates.
- Synchronous projection is valuable for ordering, but it amplifies the impact of slow storage; proactive eviction and non-blocking broadcast are necessary complements.

## Follow-ups (Completed)

- Added time-based seq fallback in StreamCoordinator.
- Updated all webchat docs to align with refactor changes.
- Captured hydration evidence and Playwright validation in PI-008.

## Appendix: Key Files Updated

- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/conversation.go`
- `pinocchio/pkg/webchat/stream_coordinator.go`
- `pinocchio/pkg/webchat/connection_pool.go`
- `pinocchio/pkg/webchat/timeline_projector.go`
- `pinocchio/pkg/webchat/timeline_store.go`
- `pinocchio/pkg/doc/topics/webchat-*`
- `pinocchio/cmd/web-chat/README.md`

