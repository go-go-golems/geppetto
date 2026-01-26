---
Title: RTK DevTools CLI Design (SocketCluster)
Ticket: RDX-001-RTK-CLI
Status: active
Topics: []
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../../../tmp/remote-devtools/package/src/devTools.ts
      Note: Remote client protocol behavior
    - Path: ../../../../../../../../../../../../tmp/remotedev-server/package/README.md
      Note: Server options and report store
    - Path: glazed/pkg/doc/tutorials/05-build-first-command.md
      Note: Glazed command construction reference
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-26T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# RTK DevTools CLI Design (SocketCluster)

## Goal

Design a production-grade CLI that connects to the official Redux DevTools remote server over SocketCluster and provides query and streaming functionality with Glazed-structured output.

## Inputs and Sources (Reviewed)

- Official remotedev server README (options, report store, GraphQL endpoint).  
- Redux DevTools remote integration docs (SocketCluster event names and remote usage).  
- @redux-devtools/cli options and monitor UI flow.  
- Stack Overflow discussion for React Native/Expo Hermes networking caveats.  
- Scaler article for high-level DevTools behavior (time travel, action history).

## Requirements

### Functional

- Connect to `ws://<host>:<port>/socketcluster/`.
- Login as a monitor client (SocketCluster `login` event).
- Subscribe to the server's monitor channel and receive DevTools relay messages.
- Commands:
  - `list`: observe live sessions and summarize instance ids
  - `tail <instance-id>`: stream action events
  - `state <instance-id> [path]`: output latest state or subpath
  - `watch <instance-id> <path>`: emit rows on value change

### Non-functional

- Structured output formats via Glazed (`--output json|yaml|csv|table`).
- Deterministic behavior under missing or malformed payloads.
- Clear error messages for handshake, login, and decode failures.

## Protocol Overview (SocketCluster + DevTools)

### Transport and Channels

- SocketCluster WebSocket endpoint: `/socketcluster/`.
- Monitor login: emit `login` and receive the channel name (typically `log`).
- Channel usage:
  - `log`: stream of relay messages from app clients.
  - `respond`: stream of monitor responses (if implementing time travel).
  - `sc-<socketId>`: internal routing for specific socket.

### Relay Message Shapes

Relay messages include a type field and JSAN-encoded payload/action strings. The core types are:

- `STATE`: lifted state snapshot
- `ACTION`: action and resulting lifted state
- `START`, `STOP`: monitor state
- `DISPATCH`: time travel or state control
- `SYNC`, `IMPORT`, `UPDATE`: state sync operations
- `DISCONNECTED`: monitor disconnect

For CLI usage, the minimum requirement is to correctly parse `STATE` and `ACTION` to extract `action.type` and decode the payload.

## CLI Architecture

### Components

1. **SocketCluster monitor client**
   - Connect, login, subscribe
   - Reconnect-safe (best effort)

2. **Relay message decoder**
   - Parse into typed struct
   - Decode `payload` and `action` JSON strings

3. **Command handlers**
   - Map relay messages to Glazed rows
   - Enforce timeouts and cancellation

### Data Flow

```
SocketCluster -> login -> channel subscription -> relay messages
     |                                                |
     |----------------------------------------------->|
                 Command handlers -> Glazed output
```

## Command Behavior Details

### list

- Listen for live relay messages for a short window (`--timeout-seconds`).
- Cache unique instance ids (from relay message fields).
- Emit rows: `instance_id`, `app`, `last_seen`, `meta`.
- Limitation: official server does not provide a separate list API, so list is inferred from traffic.

### tail

- Stream `ACTION` messages for a given instance id.
- Decode action payload; extract `action.type`.
- Optional: include decoded state with `--include-state`.

### state

- Wait for the first `STATE` or `ACTION` message for the instance.
- Decode lifted state and resolve dot-path (if provided).
- Emit a single row.

### watch

- Track a dot-path value and emit when it changes.
- Handle missing path behavior via `--allow-missing`.
- Use `reflect.DeepEqual` for comparisons.

## Glazed Integration

- Each command implements `cmds.GlazeCommand` and emits `types.Row`.
- Use Glazed parameter layers for output formatting and debug flags.
- Provide rich help and examples for each command.

## Error Handling Strategy

- **Bad handshake**: explain URL shape (`/socketcluster/`).
- **Login failure**: surface server error from ack.
- **Decode failure**: include raw payload string in output for debugging.
- **Timeout**: explicit message (`no instances observed before timeout`).

## Testing Approach

- Unit tests for dot-path resolution and action type extraction.
- Smoke test against a local `redux-devtools` server.
- Capture and replay relay messages as fixtures.

## References

- https://github.com/zalmoxisus/remotedev-server
- https://github.com/reduxjs/redux-devtools/blob/main/docs/Integrations/Remote.md
- https://www.npmjs.com/package/@redux-devtools/cli
- https://www.scaler.com/topics/react/redux-devtools/
- https://stackoverflow.com/questions/76595014/redux-devtools-with-expo-49-beta-react-native-hermes-engine

