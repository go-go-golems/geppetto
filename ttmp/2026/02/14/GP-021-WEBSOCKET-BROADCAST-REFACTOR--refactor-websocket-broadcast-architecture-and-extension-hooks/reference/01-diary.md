---
Title: Diary
Ticket: GP-021-WEBSOCKET-BROADCAST-REFACTOR
Status: active
Topics:
    - backend
    - websocket
    - architecture
    - events
    - webchat
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/scripts/01-trace-ws-broadcast-paths.sh
      Note: repro script for broadcast path inventory
    - Path: geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/scripts/02-inventory-ws-protocol-surface.sh
      Note: repro script for protocol/event inventory
    - Path: geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/scripts/03-hookability-audit.sh
      Note: repro script for extension seam audit
    - Path: geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/sources/ws-broadcast-paths.txt
      Note: recorded output for script 01
    - Path: geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/sources/ws-hookability-audit.txt
      Note: recorded output for script 03
    - Path: geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/sources/ws-protocol-surface.txt
      Note: recorded output for script 02
    - Path: pinocchio/pkg/webchat/connection_pool.go
    - Path: pinocchio/pkg/webchat/conversation.go
    - Path: pinocchio/pkg/webchat/router.go
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
    - Path: pinocchio/pkg/webchat/timeline_upsert.go
ExternalSources: []
Summary: Implementation diary for websocket broadcast refactor analysis and design work.
LastUpdated: 2026-02-14T17:33:00-05:00
WhatFor: Track exploration, experiments, and decisions used to produce GP-021 design.
WhenToUse: Use when reviewing or continuing websocket refactor planning work.
---


# Diary

## Goal
Document how websocket broadcast internals were explored, what experiments were run, and how the refactor proposal was derived.

## Step 1: Build a concrete map of current websocket fanout
I started by answering the direct architectural question: who actually calls `Broadcast`, from where, and through which data path. The objective was to avoid speculative design and anchor the ticket in concrete callsites.

This step revealed that fanout is highly concentrated and therefore tractable to refactor, but it also confirmed that subscription/profile semantics are absent at the pool level.

### Prompt Context
**User prompt (verbatim):** "Let's brainstorm a bit about the websocket thing. We could have connection with different profiles for filtering / adding stuff, but also... how does the ws currently \"receive\" things to broadcast? who calls Broadcast? I also want to explore the possibility for backend side systems to \"hook\" into the broadcast mechanism to add their own events easily, without necessarily having to access the ConnectionPool. 

Create a separate ticket for refactoring the websocket part, and add a detailed analysis and design/brainstorm document, 5+ pages. Then upload to remarkable. You can run experiments, store the scripts in scripts/ of said new ticket."

**Assistant interpretation:** Create a new ticket focused on websocket refactor, run reproducible experiments, produce a long analysis/design doc, and upload it to reMarkable.

**Inferred user intent:** Turn an ad-hoc architecture discussion into an actionable, evidence-based design package with explicit migration path.

**Commit (code):** N/A (docs/scripts only)

### What I did
- Created ticket `GP-021-WEBSOCKET-BROADCAST-REFACTOR`.
- Inspected `router.go`, `conversation.go`, `stream_coordinator.go`, `connection_pool.go`, `timeline_upsert.go`, and option/hook files.
- Identified primary fanout sites:
  - SEM frame fanout in stream callback (`conversation.go`).
  - timeline upsert fanout (`timeline_upsert.go`).
  - targeted `SendToOne` for hello/pong in `/ws` handler (`router.go`).

### Why
- Needed a precise call graph before proposing extension API boundaries.

### What worked
- Static inventory with `rg` gave a complete and quick broadcast call map.

### What didn't work
- N/A in this step.

### What I learned
- Existing extension points exist around projector and builder wiring, but not at websocket fanout abstraction level.
- `ConnectionPool` is transport-oriented and unaware of per-connection profile semantics.

### What was tricky to build
- Main subtlety was distinguishing “event production hooks” from “fanout hooks”; they are not the same layer in current code.

### What warrants a second pair of eyes
- Whether profile filtering should live inside pool or in a broker layer above pool.

### What should be done in the future
1. Keep `ConnectionPool` transport-only if possible.
2. Add explicit broker/publisher interfaces for policy and extension.

### Code review instructions
- Start with `pinocchio/pkg/webchat/router.go` `/ws` section.
- Follow into `pinocchio/pkg/webchat/conversation.go` stream callback closure.
- Validate low-level fanout in `pinocchio/pkg/webchat/connection_pool.go`.

### Technical details
- High-signal command used:
  - `rg -n "Broadcast\(|SendToOne\(|emitTimelineUpsert\(|onFrame|SemanticEventsFromEventWithCursor" pinocchio/pkg/webchat -g'*.go'`

## Step 2: Run reproducible experiments and capture outputs
I added scripts under ticket-local `scripts/` to make the investigation repeatable by others. This keeps the design verifiable and prevents knowledge from living only in prose.

The scripts generated inventories under ticket-local `sources/`, which were used directly in the design analysis.

### Prompt Context
**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Back the design document with repeatable script outputs, not only manual source reading.

**Inferred user intent:** Make the brainstorming artifact auditable and easy to revisit.

**Commit (code):** N/A (docs/scripts only)

### What I did
- Added scripts:
  - `scripts/01-trace-ws-broadcast-paths.sh`
  - `scripts/02-inventory-ws-protocol-surface.sh`
  - `scripts/03-hookability-audit.sh`
- Executed all three scripts.
- Captured outputs:
  - `sources/ws-broadcast-paths.txt`
  - `sources/ws-protocol-surface.txt`
  - `sources/ws-hookability-audit.txt`

### Why
- Needed deterministic evidence for callsite and extension-seam claims.

### What worked
- Scripts ran cleanly and produced expected inventories.
- Output confirms `/ws` query surface currently only includes `conv_id` and `profile`.

### What didn't work
- N/A during script execution.

### What I learned
- Broadcast producers are very few and centralized.
- Hookability exists for timeline and request/build layers, but not for generic websocket emitters.

### What was tricky to build
- Path resolution for ticket-local script execution needed stable workspace-relative handling to avoid brittle hardcoding.

### What warrants a second pair of eyes
- Whether script inventories should be promoted into a shared tooling area once this pattern is reused.

### What should be done in the future
1. Keep these scripts updated when websocket architecture changes.
2. Add CI guard script once refactor begins to assert expected fanout boundaries.

### Code review instructions
- Review script contents first, then inspect generated `sources/*.txt` outputs.
- Confirm that design assertions map back to output lines.

### Technical details
- Execution command:
  - `./01-trace-ws-broadcast-paths.sh && ./02-inventory-ws-protocol-surface.sh && ./03-hookability-audit.sh`

## Step 3: Produce 5+ page design/brainstorm document
After building the code and experiment baseline, I authored the long-form design with architecture options, recommended approach, migration phases, and explicit answers to the original questions about broadcast ownership and backend extensibility.

The proposed direction is broker + transport split: backend systems publish through interfaces and avoid direct `ConnectionPool` dependency.

### Prompt Context
**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Deliver a substantial design artifact, not a short note.

**Inferred user intent:** Enable implementation work with clear tradeoff analysis and staged rollout plan.

**Commit (code):** N/A (docs/scripts only)

### What I did
- Wrote `design/01-websocket-broadcast-refactor-analysis-brainstorm-and-design.md`.
- Included:
  - architecture of current system
  - inventory findings
  - problem statements
  - option analysis and recommendation
  - proposed interfaces
  - migration/testing/ops plans
  - turn-snapshot channeling strategy as future optional channel

### Why
- Needed a complete implementation-ready plan rather than fragmented notes.

### What worked
- Existing architecture was clear enough to support concrete interface proposals.
- Experiment outputs were sufficient to justify claims.

### What didn't work
- N/A.

### What I learned
- Refactor can be staged with low initial risk by preserving existing payload/default behavior while introducing broker abstraction first.

### What was tricky to build
- Balancing future flexibility (profiles/channels/plugins) against minimal invasive migration path required explicit phase boundaries.

### What warrants a second pair of eyes
- Recommended interface design (`ConversationWSBroker` / publisher contracts).
- Whether profile filters should operate on envelope type or on decoded payload semantics.

### What should be done in the future
1. Approve interface and layering direction.
2. Split implementation into phase tickets (broker introduction, profile filtering, plugin emitters, optional debug channels).

### Code review instructions
- Read the design doc top-to-bottom once for architecture flow.
- Then cross-check all claims against scripts in `sources/` and source files in `pinocchio/pkg/webchat/`.

### Technical details
- Primary deliverable path:
  - `geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/design/01-websocket-broadcast-refactor-analysis-brainstorm-and-design.md`
