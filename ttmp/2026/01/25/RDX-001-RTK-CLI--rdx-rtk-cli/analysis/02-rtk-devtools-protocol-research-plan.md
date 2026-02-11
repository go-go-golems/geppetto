---
Title: RTK DevTools Protocol Research Plan
Ticket: RDX-001-RTK-CLI
Status: active
Topics: []
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../../../tmp/remote-devtools/package/src/devTools.ts
      Note: Primary client-side protocol behavior
    - Path: ../../../../../../../../../../../../tmp/remotedev-server/package/README.md
      Note: Server configuration and report store details
    - Path: ../../../../../../../../../../../../tmp/remotedev-server/package/lib/worker.js
      Note: Server channel routing and login flow
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-26T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# RTK DevTools Protocol Research Plan

## Goal

Provide a detailed, actionable plan for an intern to produce a full-featured, authoritative document describing the Redux DevTools “remote” protocol (SocketCluster + lifted state), so engineering can implement or interoperate with Redux DevTools features confidently.

## Scope

The final document should cover:

- Wire protocol fundamentals (SocketCluster transport, event names, handshake, channels).
- The DevTools message schema (STATE/ACTION/START/STOP/DISPATCH/SYNC/IMPORT/UPDATE/etc).
- Data encoding (JSAN, lifted state shape, action sanitization).
- Protocol roles (app client, monitor UI, server).
- Compatibility and version notes across @redux-devtools/* packages.
- Practical integration guidance and test vectors.

## Summary of Existing Local Sources (Verified)

The following local sources were already inspected and should be treated as primary references in the write‑up:

1. **`@redux-devtools/remote` client implementation**
   - Source: `/tmp/remote-devtools/package/src/devTools.ts`
   - Used to infer:
     - SocketCluster login flow (`login` with `master` for monitor)
     - Event names (`log`, `log-noid`, `respond`)
     - Relay message shapes (`STATE`, `ACTION`, `START`, `STOP`, `DISPATCH`, `SYNC`, `IMPORT`, `UPDATE`, `DISCONNECTED`)
     - JSAN encoding of lifted state and action payloads

2. **`remotedev-server` (official server) implementation**
   - Source: `/tmp/remotedev-server/package/lib/worker.js`
   - Used to infer:
     - `login` behavior and channel routing
     - Publish/subscribe flows via `sc-*` channels
     - Emit and subscribe middleware for `log`, `respond`, and `report`

3. **`remotedev-server` README**
   - Source: `/tmp/remotedev-server/package/README.md`
   - Used to infer:
     - Server configuration flags
     - Existence of `/graphiql` for stored reports
     - Report storage and DB options

These sources are enough to draft a credible protocol description. The intern should still verify against upstream repositories and docs.

## Research Plan (Step‑by‑Step)

### Phase 1 — Map the Transport Layer

**Objective:** Describe SocketCluster framing and the event topology used by the DevTools remote ecosystem.

1) **Read SocketCluster basics** (official docs + protocol overview).
2) **Confirm default endpoint shape**: `/socketcluster/` path.
3) **Document SocketCluster primitives** used:
   - `login` request/response
   - event transmission (`emit` and `transmit`)
   - channel subscriptions (`subscribe`)
4) **Extract and diagram** from `worker.js`:
   - Channel routing logic (monitor vs app)
   - `sc-<socketId>` exchange subscriptions
   - `log`, `respond`, and `report` behavior

**Deliverable:** A concise transport diagram with message direction arrows and channel names.

### Phase 2 — Extract the DevTools Message Schema

**Objective:** Produce a complete schema catalog of message types and fields.

1) **Identify all message types** in `devTools.ts`:
   - `STATE`, `ACTION`, `START`, `STOP`, `ERROR`, `DISPATCH`, `SYNC`, `IMPORT`, `UPDATE`, `DISCONNECTED`
2) **For each message type**:
   - Required fields
   - Optional fields
   - Example payload (use real values from code)
   - How the receiver interprets the message
3) **Document message serialization**:
   - JSAN encoding of lifted state and actions
   - Lifted state shape (`@redux-devtools/instrument`)

**Deliverable:** A table of message types with example JSON payloads and semantics.

### Phase 3 — Lifted State Deep‑Dive

**Objective:** Explain the lifted state model clearly enough to reimplement or interpret DevTools UI features.

1) **Read `@redux-devtools/instrument`** to capture:
   - Shape of lifted state (actionsById, stagedActionIds, committedState, etc.)
   - How `IMPORT`, `SYNC`, and `DISPATCH` affect it
2) **Produce diagrams** for:
   - Timeline/staging concepts
   - Time travel / state recomputation pipeline

**Deliverable:** A conceptual and structural description of lifted state with field‑by‑field descriptions.

### Phase 4 — Monitor vs App Roles

**Objective:** Clarify how monitors (UI) and apps behave differently.

1) **Monitor role:**
   - Login as `master` and subscribe to `respond`
   - Send `UPDATE` or `DISPATCH` to sync and time travel
2) **App role:**
   - Emit `log` or `log-noid` until id is assigned
   - Handle `DISPATCH` / `ACTION` / `IMPORT` coming from monitor
3) **Identify any role assumptions** baked into server routing.

**Deliverable:** Role‑based sequence diagrams for app and monitor.

### Phase 5 — Report Store and GraphQL

**Objective:** Determine what is stored and how the GraphQL endpoint can be used.

1) **Read server store implementation** (`lib/store.js`) to map:
   - Stored fields: id, title, payload, action, meta, user, instanceId, etc.
2) **Locate GraphQL schema** (`/graphiql`, server code) for available queries.
3) **Document how reports link to DevTools UI** (action history replay).

**Deliverable:** A section on report persistence and API surface.

### Phase 6 — Compatibility Matrix

**Objective:** Note version and compatibility concerns.

1) **Review package versions** for:
   - `@redux-devtools/remote`
   - `remotedev-server`
   - `redux-devtools` CLI
2) **Record differences** in message shapes or transport in older versions.
3) **List unstable or undocumented behaviors** (e.g., `log-noid`, fallback paths).

**Deliverable:** Compatibility matrix with warnings and version notes.

### Phase 7 — Practical Integration Recipes

**Objective:** Provide copy‑paste‑ready integration patterns.

1) **App integration** (Redux and RTK) with `@redux-devtools/remote`.
2) **Monitor integration** (DevTools UI) with `redux-devtools` CLI.
3) **CLI integration** (this repo) with SocketCluster mode.

**Deliverable:** A “How to connect” section with exact commands and options.

### Phase 8 — Test Vectors and Debugging

**Objective:** Provide real test cases for validating protocol implementations.

1) **Record sample relay messages** from a live session.
2) **Provide JSON test fixtures** for each message type.
3) **Document common errors**:
   - Bad handshake (wrong URL)
   - Stale channel subscription
   - Missing JSAN decode

**Deliverable:** A test appendix with fixtures and troubleshooting steps.

## Research Questions to Answer (Checklist)

- What fields does `ACTION` carry in the remote protocol (besides `action` and `payload`)?
- How does the app learn its socket id and instance id?
- What exactly is sent over `log` vs `respond`?
- How are `START/STOP` used by the monitor?
- Which messages are required for time‑travel to work?
- How do `SYNC` and `IMPORT` differ in semantics?

## Proposed Outline of the Final Guide

1. Overview and terminology
2. Transport layer (SocketCluster)
3. Message catalog (with examples)
4. Lifted state model
5. Roles and flows (app vs monitor)
6. Report store + GraphQL
7. Compatibility notes
8. Integration recipes
9. Test vectors
10. Troubleshooting

## Deliverables for the Intern

- A markdown guide in the ticket’s `analysis/` directory
- A diagram pack (sequence diagrams + transport map)
- JSON fixtures for automated tests
- A short README section with integration commands

## Success Criteria

- Engineers can implement a DevTools‑compatible client based solely on the document
- Protocol coverage includes time travel and dispatch
- Reproducible test vectors and troubleshooting guidance exist
