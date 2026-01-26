---
Title: Minimal SocketCluster Client Plan
Ticket: RDX-007-SC-CLIENT
Status: active
Topics: [rdx, cli, network]
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: rdx/cmd/rdx/socketcluster.go
      Note: Current monitor implementation depends on sacOO7 client
    - Path: rdx/cmd/rdx/socketcluster_commands.go
      Note: Commands that consume SocketCluster monitor
    - Path: rdx/pkg/rtk/socketcluster.go
      Note: Relay message parsing helpers
    - Path: rdx/third_party/socketcluster-client-go/scclient/client.go
      Note: Upstream implementation used as protocol reference
    - Path: rdx/third_party/socketcluster-client-go/scclient/models/handshake.go
      Note: Handshake payload structure
    - Path: rdx/third_party/socketcluster-client-go/scclient/models/event.go
      Note: Emit/subscribe payloads and #publish channel messages
ExternalSources: []
Summary: "Define a minimal SocketCluster client to replace sacOO7 dependency for RDX."
LastUpdated: 2026-01-26T12:20:00-05:00
WhatFor: "Guide implementation of an in-repo SocketCluster client"
WhenToUse: "When removing third-party client dependency and stabilizing RDX monitoring"
---

# Minimal SocketCluster Client Plan

## Goal

Replace the current `github.com/sacOO7/socketcluster-client-go` dependency with a **minimal in-repo SocketCluster client** that implements only the features RDX needs (`list`, `tail`, `state`, `watch`). This avoids upstream panics and gives us full control over message parsing.

## Current State

- `rdx/cmd/rdx/socketcluster.go` uses the sacOO7 client.
- The library panics on nil/invalid messages (observed in `parser.GetMessageDetails`).
- We only need a narrow slice of SocketCluster behavior: connect, handshake, heartbeat, login ack, subscribe, publish delivery.

## Minimal Protocol Requirements

Based on the upstream client and observed server behavior:

1. **WebSocket connect** to `ws://host/socketcluster/`.
2. **Handshake**: send `{ "event": "#handshake", "data": { "authToken": null }, "cid": <id> }`.
3. **Heartbeat**: if server sends `"#1"`, reply with `"#2"`.
4. **Emit with ack** (used for login): send `{ "event": "login", "data": "monitor", "cid": <id> }` and wait for response `{ "rid": <id>, "data": <channel> }`.
5. **Subscribe**: send `{ "event": "#subscribe", "data": { "channel": "log" }, "cid": <id> }` (ack optional).
6. **Publish**: server sends `{ "event": "#publish", "data": { "channel": "log", "data": <payload> } }`.

We can ignore auth token set/remove and other advanced features.

## Minimal Client API (Proposed)

Package: `rdx/pkg/rtk/scclient`

```go
// New creates a client for a SocketCluster server.
New(url string) *Client

// Connect establishes the websocket, sends handshake, and starts read loop.
Connect(ctx context.Context) error

// Close shuts down the websocket.
Close() error

// EmitAck sends an event and waits for an ack response (rid == cid).
EmitAck(ctx context.Context, event string, data interface{}) (interface{}, error)

// Subscribe subscribes to a channel.
Subscribe(ctx context.Context, channel string) error

// OnPublish registers a handler for #publish events.
OnPublish(func(channel string, data interface{}))
```

## Replacement Plan

1. **Add `rdx/pkg/rtk/scclient`** implementing the minimal API.
2. Update `rdx/cmd/rdx/socketcluster.go` to use the new client:
   - `EmitAck` for `login` to retrieve channel.
   - `Subscribe` for channel.
   - `OnPublish` callback to parse relay messages.
3. Remove the `third_party/socketcluster-client-go` vendor and the `go.mod replace`.
4. Run tests (`go test ./...`) and manually confirm `rdx list` no longer panics.

## Error Handling & Robustness

- Validate JSON parsing; ignore malformed messages.
- Time out `EmitAck` using context deadlines.
- Guard concurrency with a mutex for the pending ack map.

## Deliverables

- Minimal SocketCluster client in `rdx/pkg/rtk/scclient`.
- `socketcluster.go` migrated to the new client.
- Removal of sacOO7 dependency and vendored copy.
- Diary updates and documented tasks.
