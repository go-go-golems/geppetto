---
Title: Dispatch/Update/Import/Sync Spec
Ticket: RDX-003-DISPATCH-SYNC
Status: active
Topics: []
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../../../tmp/remote-devtools/package/src/devTools.ts
      Note: Control message shapes and monitor behavior
    - Path: ../../../../../../../../../../../../tmp/remotedev-server/package/lib/worker.js
      Note: Channel routing for respond/log
    - Path: rdx/cmd/rdx/socketcluster.go
      Note: SocketCluster login
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-26T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Dispatch/Update/Import/Sync Spec

## Goal

Implement full **control-channel functionality** for the Redux DevTools remote protocol, enabling dispatch, state sync, import/export, and update operations from the CLI.

## Context

- The official DevTools protocol supports control messages used by monitor UIs.
- These include `DISPATCH`, `SYNC`, `IMPORT`, and `UPDATE` (plus `START/STOP`).
- The CLI currently only reads from the `log` channel and does not emit control messages.

Related tickets:
- RDX-001-RTK-CLI (SocketCluster transport)
- RDX-002-HIST-REPORTS (report replay may emit control messages)
- RDX-004-DIFF-SNAPSHOT (may use update/sync flows)

## Requirements

### Functional

1. **Emit control messages** to the correct channel (`respond`).
2. Support **time travel** and **state replacement** via `DISPATCH` and `IMPORT`.
3. Provide **sync operations** to fetch the latest lifted state.
4. Provide **update** to request new state from the app client.

### Non-Functional

- No backward compatibility with the custom JSON protocol.
- Prefer clarity over performance.

## Protocol Notes (From Remote Client Behavior)

The remote monitor uses SocketCluster messages that include:

- `UPDATE` — request the app to send the current lifted state.
- `IMPORT` — replace lifted state with an imported payload.
- `SYNC` — synchronize state between app and monitor.
- `DISPATCH` — time-travel or control action with a payload describing the change.

The CLI must:

- Login with **monitor credentials** (likely `master`) to receive `respond` channel updates.
- Emit control messages to the server so they route to app clients.

## Command Surface

### `rdx control update <instance-id>`

- Sends `{ type: "UPDATE" }` to the control channel.
- Waits for the next `STATE` payload from the instance.

### `rdx control sync <instance-id>`

- Sends `{ type: "SYNC", ... }` with optional last-known state hash.
- Emits a structured `STATE` row when received.

### `rdx control import <instance-id> --file <path>`

- Reads a lifted state JSON file and sends `{ type: "IMPORT", state: <json> }`.
- Can optionally verify success by listening for `STATE` confirmation.

### `rdx control dispatch <instance-id> --payload <json>`

- Sends a DevTools `DISPATCH` with a typed payload, e.g. `{ type: "JUMP_TO_ACTION", ... }`.
- Supports common helpers:
  - `rdx control jump --index N`
  - `rdx control reset`
  - `rdx control rollback`

## Data Model and Encoding

- Use the lifted-state model from `@redux-devtools/instrument`.
- Payloads are often JSAN-encoded strings; CLI should accept both raw JSON and JSAN input.

## Error Handling

- If no response is received within timeout, return an explicit error.
- If the app ignores the message, log that no confirmation was received.
- Validate JSON payloads before sending.

## Implementation Plan

1. Add a control-channel client in `rdx/pkg/rtk/socketcluster_control.go`.
2. Implement command group `rdx control` with subcommands above.
3. Provide helpers to build common DISPATCH payloads.
4. Update docs and examples.

## Files and Symbols to Touch

- `rdx/cmd/rdx/commands.go` (register control commands)
- `rdx/cmd/rdx/socketcluster.go` (login as master when needed)
- `rdx/cmd/rdx/socketcluster_commands.go` (shared message parse)

## Open Questions

- Exact message shapes for `DISPATCH` variants (JUMP_TO_ACTION, IMPORT_STATE, RESET, etc.).
- Whether the server expects JSAN for some payload fields.

## Out of Scope

- UI integration or TUI mode.
- Server-side changes.

## Deliverables

- Full control command set.
- Documented examples of time travel and import.
- Test fixtures for each control message type.

