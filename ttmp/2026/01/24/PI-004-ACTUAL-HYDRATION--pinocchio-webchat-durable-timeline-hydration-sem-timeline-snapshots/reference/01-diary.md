---
Title: Diary
Ticket: PI-004-ACTUAL-HYDRATION
Status: active
Topics:
    - backend
    - pinocchio
    - webchat
    - hydration
    - timeline
    - protobuf
    - websocket
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: GET /timeline + env config + user message snapshot (commit b1f908b)
    - Path: ../../../../../../../pinocchio/pkg/webchat/timeline_projector.go
      Note: SEM frame -> sem.timeline snapshot projector (commit b1f908b)
    - Path: ../../../../../../../pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: SQLite schema + upsert + versioned snapshot queries (commit b1f908b)
    - Path: ../../../../../../../pinocchio/proto/sem/timeline/planning.proto
      Note: PlanningSnapshotV1 schema (commit 244757b)
    - Path: ../../../../../../../pinocchio/proto/sem/timeline/tool.proto
      Note: Tool snapshot extensions (result_raw/custom_kind/done) (commit 244757b)
    - Path: ../../../../../../../pinocchio/proto/sem/timeline/transport.proto
      Note: Transport protos for snapshots/upserts (commit d97efe1)
    - Path: ttmp/2026/01/24/PI-004-ACTUAL-HYDRATION--pinocchio-webchat-durable-timeline-hydration-sem-timeline-snapshots/design-doc/01-durable-hydration-via-sem-timeline-snapshots.md
      Note: Design doc (updated to choose SQLite + multi-conversation schema)
    - Path: ttmp/2026/01/24/PI-004-ACTUAL-HYDRATION--pinocchio-webchat-durable-timeline-hydration-sem-timeline-snapshots/tasks.md
      Note: Expanded implementation tasks + acceptance criteria
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-24T19:38:46.981538914-05:00
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Keep a detailed, step-by-step diary for PI-004-ACTUAL-HYDRATION: designing (and later implementing) “actual hydration” for Pinocchio webchat via durable `sem.timeline.*` snapshot semantics.

## Context

This ticket exists because the current Pinocchio hydration (`GET /hydrate`) is a bounded replay of buffered SEM frames. It is useful for reconnect gating but is not durable nor a canonical “UI state snapshot.” The design here centers `sem.timeline.*` protobuf schemas as the canonical snapshot layer.

## Quick Reference

### Current baseline

- `GET /hydrate?conv_id=...` returns buffered SEM frames (JSON) + `last_seq`.
- Buffer is in-memory only: `pinocchio/pkg/webchat/sem_buffer.go`.

### Proposed endpoint (draft)

```text
GET /timeline?conv_id=...&since_version=...&limit=...
-> { conv_id, version, entities: [...], server_time_ms }
```

### Snapshot schemas (source of truth)

- `pinocchio/proto/sem/timeline/message.proto`
- `pinocchio/proto/sem/timeline/tool.proto`
- `pinocchio/proto/sem/timeline/status.proto`
- `pinocchio/proto/sem/timeline/middleware.proto`

## Usage Examples

N/A (design ticket; implementation will add playbooks as code lands)

## Related

- PI-003 React webchat baseline ticket:
  - `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/`

## Step 1: Create PI-004 ticket and write the design doc

Created a separate ticket for “actual hydration” and wrote the first version of the design doc that specifies `sem.timeline.*` as the canonical snapshot layer, proposes a `GET /timeline` endpoint, and outlines a server-side projection store with monotonic versioning.

### Prompt Context

**User prompt (verbatim):**

> Create a separate ticket for implementing the actual hydration, and write a detailed implementation/design doc for that in this new ticket.
>
> Then return to this ticket, and implement 2.
>
> Check tasks off as you go, write diary, commit to git.

**Assistant interpretation:** Create a new docmgr ticket (PI-004) for hydration design/implementation, write a detailed design doc there, then return to PI-003 to implement backend emission for planning/thinking-mode SEM events.

**Inferred user intent:** Separate “durable hydration” work (timeline snapshot architecture) from “React webchat parity” work so each can be implemented/reviewed independently.

**Commit (docs):** pending

### What I did
- Created ticket: `PI-004-ACTUAL-HYDRATION`
- Added docs:
  - `design-doc/01-durable-hydration-via-sem-timeline-snapshots.md`
  - `reference/01-diary.md`
- Drafted an implementation-oriented design doc:
  - Defines goals and “actual hydration” definition
  - Proposes a snapshot endpoint `GET /timeline`
  - Proposes a projection store with versioning
  - Captures open questions (WS payload types, persistence choice)
- Added an initial task list to `tasks.md`

## Step 2: Decide and design the projection store as SQLite (multi-conversation)

This step narrows the design from “possible persistence options” to a single concrete choice: a SQLite-backed projection store that supports multiple conversations in one database file. This choice is made to unlock real durability (across restarts) without introducing another infrastructure dependency, and to keep hydration semantics canonical on the backend rather than “whatever the browser last saw”.

The key change is that the backend becomes responsible for maintaining an upsertable set of timeline entities keyed by `(conv_id, entity_id)` with a monotonic per-conversation `version`. That `version` becomes the durable cursor for both snapshot responses and future WS catch-up.

### Prompt Context

**User prompt (verbatim):**
```text
ok, what ever, let's work on PI-004.

design projection store as sqlite (for multiple conversations, not just one). 

Create detailed tasks and add to ticket. 

Then implement, check off tasks as you go, keep a detailed diary (in PI-004) and commit as you go.
```

**Assistant interpretation:** Switch focus to PI-004, choose SQLite as the persistence backend for the timeline projection store (supporting multiple conversations), expand the ticket tasks with implementation detail, then start implementing with incremental commits and diary updates.

**Inferred user intent:** Make “actual hydration” durable and operationally real (restart-safe, multi-conversation) so the React webchat can hydrate from canonical server state rather than replaying ephemeral event buffers.

**Commit (docs):** pending

### What I did
- Updated PI-004 design doc to explicitly choose SQLite as the persistence mechanism and describe a multi-conversation schema and transaction semantics:
  - `design-doc/01-durable-hydration-via-sem-timeline-snapshots.md`
- Expanded PI-004 tasks into a detailed checklist with acceptance criteria:
  - `tasks.md`

### Why
- SQLite is the fastest path to “durable hydration” in Pinocchio: it’s restart-safe and doesn’t require a Redis dependency for the snapshot store.
- Multi-conversation is a hard requirement for real use: any store keyed only by “current conversation” will break as soon as we run multiple chat sessions concurrently.

### What worked
- The existing design doc already had the right conceptual shape (projection + versioned snapshot endpoint); it only needed a concrete persistence decision and schema.

### What didn't work
- N/A (design/doc step)

### What I learned
- The right durable cursor for hydration is not the WS buffer sequence; it’s a per-conversation projection `version` that increments on every entity upsert.

### What was tricky to build
- N/A (no code yet)

### What warrants a second pair of eyes
- The schema/API boundary decision: whether to store payloads as protojson strings (inspectable) or binary protobuf (smaller/faster). The current design chooses protojson for simplicity and alignment with existing wire format.

### What should be done in the future
- Once the implementation lands, add a short “ops doc” for enabling the DB, choosing a file path, and understanding growth/retention behavior.

### Code review instructions
- Review doc changes:
  - `geppetto/ttmp/2026/01/24/PI-004-ACTUAL-HYDRATION--pinocchio-webchat-durable-timeline-hydration-sem-timeline-snapshots/design-doc/01-durable-hydration-via-sem-timeline-snapshots.md`
  - `geppetto/ttmp/2026/01/24/PI-004-ACTUAL-HYDRATION--pinocchio-webchat-durable-timeline-hydration-sem-timeline-snapshots/tasks.md`

## Step 3: Implement SQLite projection store + SEM→snapshot projector + GET /timeline

This step lands the first working version of “actual hydration” in Pinocchio: a durable, SQLite-backed projection store that maintains a per-conversation entity set and serves it via `GET /timeline`. The key shift is that we stop treating hydration as “replay a bounded event buffer” and instead persist the *projection state* as canonical, queryable data keyed by `(conv_id, entity_id)` with a monotonic per-conversation `version`.

The implementation intentionally starts with a minimal but useful projection coverage (messages, tools, thinking-mode, planning/execution) and is designed to scale to additional widgets by extending the projector mappings and `sem.timeline.*` snapshot schemas.

### Prompt Context

**User prompt (verbatim):**
```text
ok, what ever, let's work on PI-004.

design projection store as sqlite (for multiple conversations, not just one). 

Create detailed tasks and add to ticket. 

Then implement, check off tasks as you go, keep a detailed diary (in PI-004) and commit as you go.
```

**Assistant interpretation:** Implement the PI-004 design: add transport protos, implement a SQLite-backed multi-conversation projection store, project SEM frames into `sem.timeline.*` snapshots, and expose a `GET /timeline` endpoint; add tests; commit incrementally and check tasks off.

**Inferred user intent:** Make hydration durable and canonical (restart-safe) so the frontend can hydrate from server-owned UI state rather than replaying ephemeral SEM frame buffers.

**Commits (code):**
- `pinocchio@d97efe1` — "sem.timeline: add transport protos (snapshot/upsert)"
- `pinocchio@244757b` — "sem.timeline: add planning snapshot + extend tool snapshots"
- `pinocchio@b1f908b` — "webchat: sqlite timeline projection store + /timeline"

### What I did
- Added transport schemas for durable timeline snapshots:
  - `pinocchio/proto/sem/timeline/transport.proto` (`TimelineEntityV1`, `TimelineUpsertV1`, `TimelineSnapshotV1`)
  - Regenerated Go+TS with Buf (outputs under `pinocchio/pkg/sem/pb/...` and both TS targets).
- Extended `sem.timeline.*` snapshot coverage to match current widgets:
  - Added `pinocchio/proto/sem/timeline/planning.proto` (`PlanningSnapshotV1`)
  - Extended `pinocchio/proto/sem/timeline/tool.proto` to preserve `result_raw`, `custom_kind`, and `done`.
- Implemented the SQLite projection store (multi-conversation, versioned):
  - `pinocchio/pkg/webchat/timeline_store_sqlite.go` (schema + upsert + snapshot queries)
  - `pinocchio/pkg/webchat/timeline_store_sqlite_test.go` (monotonic versions, limit, ordering)
- Implemented a per-conversation projector from SEM frames to timeline snapshot entities:
  - `pinocchio/pkg/webchat/timeline_projector.go`
  - Covers: `llm.*`, `tool.*`, `thinking.mode.*`, `planning.*`, `execution.*`
  - Throttles high-frequency `llm.delta` DB writes (250ms per message ID) to reduce churn.
- Wired the projector into the existing streaming pipeline:
  - `pinocchio/pkg/webchat/conversation.go` calls `timelineProj.ApplySemFrame(...)` for each SEM frame after translation.
- Added `GET /timeline` and store enablement:
  - `pinocchio/pkg/webchat/router.go`:
    - `GET /timeline?conv_id=...&since_version=...&limit=...` returns protojson `TimelineSnapshotV1`
    - enables store via env vars:
      - `PINOCCHIO_WEBCHAT_TIMELINE_DSN` (full sqlite DSN), or
      - `PINOCCHIO_WEBCHAT_TIMELINE_DB` (file path; DSN derived with WAL + busy_timeout)
    - writes a user message snapshot (`user-<turn_id>`) at run start so hydration can include user prompts.

### Why
- A SQLite-backed store is the simplest way to make hydration restart-safe and multi-conversation without requiring Redis.
- Versioned upserts allow clients to request “delta since N” without reprocessing unbounded history.

### What worked
- The store tests pass as part of `go test ./...` and validate monotonic versions and incremental query semantics.
- Pinocchio pre-commit hooks ran (Go test + go generate + lint) and succeeded for each commit.

### What didn't work
- A Go parsing footgun: calling a method on a composite literal requires parentheses. This initially broke formatting/build in the SQLite store:
  - Fix: `(protojson.UnmarshalOptions{...}).Unmarshal(...)`.

### What I learned
- For full snapshots, ordering by `created_at_ms` (not “last update version”) avoids the “streaming message jumps to the end” effect when rendering.
- For incremental snapshots, ordering by `version ASC` is the simplest and most predictable catch-up behavior.

### What was tricky to build
- Mapping `tool.result` into a stable “tool_result entity id” that matches the frontend convention:
  - `tool_call_id + ":custom"` when `custom_kind` is present, else `tool_call_id + ":result"`.
- Avoiding DB churn during token streaming required explicit throttling, otherwise every `llm.delta` would become a write.

### What warrants a second pair of eyes
- Whether the 250ms throttle is the right default and whether it should be configurable.
- Whether we should store a separate append-only “audit table” of raw SEM frames for debugging and postmortems (not required for snapshots, but useful).

### What should be done in the future
- Implement PI-004 Task #5: update the React frontend hydration path to use `GET /timeline` and version-based WS catch-up gating.

### Code review instructions
- Start here (backend behavior):
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/conversation.go`
- Then review store and projection:
  - `pinocchio/pkg/webchat/timeline_store_sqlite.go`
  - `pinocchio/pkg/webchat/timeline_projector.go`
- Validate:
  - `cd pinocchio && buf generate`
  - `cd pinocchio && go test ./... -count=1`
