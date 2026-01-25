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
RelatedFiles: []
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

**User prompt (verbatim):** "Create a separate ticket for implementing the actual hydration, and write a detailed implementation/design doc for that in this new ticket. \n\nThen return to this ticket, and implement 2. \n\nCheck tasks off as you go, write diary, commit to git."

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
