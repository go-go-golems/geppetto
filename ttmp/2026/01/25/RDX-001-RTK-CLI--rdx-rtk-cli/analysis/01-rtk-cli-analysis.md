---
Title: RTK CLI Analysis (SocketCluster + Redux DevTools)
Ticket: RDX-001-RTK-CLI
Status: active
Topics: []
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: rdx/cmd/rdx/commands.go
      Note: SocketCluster command entrypoints
    - Path: rdx/cmd/rdx/socketcluster.go
      Note: SocketCluster login and subscription
    - Path: rdx/cmd/rdx/socketcluster_commands.go
      Note: SocketCluster command implementations
    - Path: rdx/pkg/rtk/path.go
      Note: Dot-path utilities
    - Path: rdx/pkg/rtk/socketcluster.go
      Note: SocketCluster relay message parsing
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-26T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# RTK CLI Analysis (SocketCluster + Redux DevTools)

## Goal

Implement a Go CLI (RDX) that connects to the **official Redux DevTools remote server** (`redux-devtools` / `remotedev-server`) over SocketCluster and exposes practical debugging commands (`list`, `tail`, `state`, `watch`) with structured output via Glazed.

## Source Summary (Official Protocol)

The official remote DevTools stack uses SocketCluster as the transport layer and relays **Redux DevTools messages** (lifted state and control events) over channels such as `log` and `respond`. The client (`@redux-devtools/remote`) uses SocketCluster to login and transmit message types like `STATE`, `ACTION`, `START`, `STOP`, `DISPATCH`, `SYNC`, `IMPORT`, and `UPDATE`. The server (`remotedev-server`) routes messages between app clients and monitor clients via these channels.

## Conceptual Model

### 1) Entities

- **App client**: emits `ACTION` and `STATE` payloads (lifted state) to the server via SocketCluster.
- **Monitor client**: subscribes to the server and can request sync or dispatch time-travel actions.
- **Server**: SocketCluster hub that routes messages by channel and socket id.

### 2) Transport and Channels

- SocketCluster endpoint: `ws://<host>:<port>/socketcluster/`
- Login event: `login` with role (`master` for monitor, `monitor` for viewer)
- Channel subscriptions:
  - `log` (messages from app clients)
  - `respond` (messages from monitors)
  - `sc-<socketId>` (server internal routing)

### 3) DevTools Message Types (High‑Level)

- **`STATE`**: lifted state snapshot (JSAN string)
- **`ACTION`**: action + resulting state (JSAN strings)
- **`START` / `STOP`**: monitor lifecycle
- **`DISPATCH`**: time‑travel, reset, import, etc.
- **`SYNC` / `IMPORT` / `UPDATE`**: state synchronization between app and monitor
- **`DISCONNECTED`**: monitor disconnect signal

These messages are part of the Redux DevTools remote protocol and are emitted over SocketCluster events.

## CLI Command Semantics (SocketCluster)

- `rdx list`: observes live SocketCluster traffic within a timeout window and reports instance ids encountered.
- `rdx tail <instance-id>`: streams `ACTION` messages for the instance and optionally includes decoded state.
- `rdx state <instance-id> [path]`: waits for a `STATE`/`ACTION` payload and extracts a dot‑path.
- `rdx watch <instance-id> <path>`: emits rows when a state path changes.

## Design Translation: Go + Glazed

### 1) Glazed Command Layer

Glazed commands provide structured output (JSON/YAML/CSV/table) using `types.Row`. Each command implements `cmds.GlazeCommand` and accesses flags via `parsedLayers.InitializeStruct`.

### 2) SocketCluster Client

The CLI uses a SocketCluster Go client to:

- Dial `/socketcluster/` endpoint
- `login` as monitor
- Subscribe to the `log` channel
- Parse relay messages into a typed struct

### 3) Payload Decoding

SocketCluster relay messages embed JSON strings for `payload` (lifted state) and `action`. The CLI decodes those into `interface{}` and extracts `action.type` when present.

### 4) Path Resolution

State queries use a simple dot‑path resolver against `map[string]interface{}` values. Arrays are currently treated as non‑indexable for deterministic behavior.

## Error Handling and Timeouts

1. **List/state commands**: use a timeout (default 5 seconds). Timeout can be disabled with `--timeout-seconds=0`.
2. **Tail/watch**: stream until context cancellation.
3. **Server errors**: surfaced as Go errors when available.

## Security and Safety Considerations

- Default server URL targets localhost SocketCluster endpoint.
- Optional token flag is preserved for future server‑side auth policies, but the official server does not enforce it by default.

## Implementation Structure

```
rdx/
  cmd/rdx/
    main.go                     # cobra + glazed bootstrapping
    commands.go                 # list, tail, state, watch (SocketCluster only)
    socketcluster.go            # SocketCluster login/subscription
    socketcluster_commands.go   # protocol-specific command handlers
  pkg/rtk/
    socketcluster.go            # relay message parsing
    path.go                     # dot-path utilities
```

### Command Output Shapes

- `list`: instance_id, app, last_seen, last_seen_unix_ms, meta
- `tail`: instance_id, action_type, timestamp, timestamp_unix_ms, action (+ state if requested)
- `state`: instance_id, path, value, timestamp, timestamp_unix_ms
- `watch`: instance_id, path, value, action_type, timestamp, timestamp_unix_ms

## Extension Points (Future Work)

1. **Array indexing** for dot paths (e.g., `items.0.id`).
2. **Diff output** in `tail` or `watch` for summarized state changes.
3. **Report store access** (GraphQL) for historical sessions.
4. **Time travel** helpers (emit DISPATCH commands).

## Conclusion

The RDX CLI is now aligned with the official Redux DevTools SocketCluster protocol. Its design keeps the CLI lightweight while enabling structured, scriptable inspection of live Redux DevTools streams.
