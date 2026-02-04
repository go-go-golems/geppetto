---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/router.go
      Note: /chat and /ws handlers referenced in CLI design
    - Path: web-agent-example/cmd/web-agent-debug/chat.go
      Note: CLI /chat implementation
    - Path: web-agent-example/cmd/web-agent-debug/main.go
      Note: CLI harness entrypoint
    - Path: web-agent-example/cmd/web-agent-debug/run.go
      Note: CLI smoke test command
    - Path: web-agent-example/cmd/web-agent-debug/timeline.go
      Note: CLI /timeline implementation
    - Path: web-agent-example/cmd/web-agent-debug/ws.go
      Note: CLI websocket implementation
    - Path: web-agent-example/pkg/thinkingmode/middleware.go
      Note: Custom middleware config referenced by CLI overrides
    - Path: web-agent-example/web/src/App.tsx
      Note: Frontend overrides payload structure
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---



# CLI Harness for /chat + WebSocket Streaming

This design document proposes a small CLI tool that emulates the web client’s `/chat` + `/ws` behavior so we can diagnose “websocket is empty” reports and verify streaming in a controlled, scriptable environment. The tool should be usable by someone with no frontend context and should output a clear, chronological view of what happens during chat submission and streaming updates.

The goal is not to replace the frontend; it is to make the *wire protocol observable* and reproducible without a browser.

---

## 1) Problem Statement

We have a live web-agent example where hydration works (timeline data appears on reload), but the websocket stream appears empty in the UI. A dedicated CLI should:

- Generate or accept a `conv_id`.
- Perform `POST /chat` with overrides (including custom thinking-mode middleware).
- Establish `GET /ws` with matching query params + cookies.
- Log the inbound WS frames and correlate them with `/chat` response + timeline snapshots.

This allows us to answer: “Is the backend streaming frames and are we *actually subscribed* to the correct conversation?”

---

## 2) Scope and Non‑Goals

**In scope**

- CLI that can do:
  - `POST /chat` and log response.
  - `GET /ws` and log/pretty print frames.
  - Optional: `GET /timeline` to confirm persisted events.
- Introspect and decode “SEM envelope” JSON frames (the `{ sem: true, event: { type, id, data } }` structure).
- Support middleware overrides in `/chat` payload (custom thinking mode).
- Provide debug “sanity checks” for common failure modes.

**Out of scope**

- Browser automation (Playwright is already a separate tool).
- Full UI simulation or hydration rendering in CLI.
- Replacing or replicating the complete frontend store logic.

---

## 3) Current Server Contract (Observed)

### `/chat` (HTTP POST)

Location: `pinocchio/pkg/webchat/router.go` lines ~620–735.

- Method: `POST`
- Request body:

```json
{
  "conv_id": "<uuid>",
  "prompt": "hello",
  "overrides": {
    "middlewares": [
      { "name": "webagent-thinking-mode", "config": { "mode": "fast" } }
    ]
  }
}
```

- Server behavior:
  - Builds an engine from request policy.
  - Creates/loads conversation by `conv_id` + profile.
  - Starts run and returns a JSON response.

### `/ws` (WebSocket)

Location: `pinocchio/pkg/webchat/router.go` lines ~462–560.

- URL: `/ws?conv_id=<id>&profile=<slug>`
- If `profile` missing, server falls back to `chat_profile` cookie.
- On connect, server sends a greeting frame:

```json
{
  "sem": true,
  "event": {
    "type": "ws.hello",
    "id": "ws.hello:<conv_id>:<ts>",
    "data": "<protojson-encoded payload>"
  }
}
```

- Ping/pong support:
  - Client may send `"ping"` or a JSON event of type `ws.ping`.
  - Server replies with `ws.pong` frame.

### `/timeline` (HTTP GET)

Location: `pinocchio/pkg/webchat/router.go` lines ~560–620.

- URL: `/timeline?conv_id=<id>&since_version=<n>&limit=<n>`
- Returns timeline snapshot JSON (protojson).
- Used for hydration; should include `webagent_thinking_mode` entity if custom events are persisted.

---

## 4) CLI Requirements

### 4.1 Core Commands

Proposed binary: `web-agent-debug` or a subcommand of `web-agent-example`.

**Command: `chat`**

- Purpose: Do `POST /chat` and print response.
- Flags:
  - `--conv-id <uuid>` (optional, generates if absent)
  - `--prompt <string>`
  - `--profile <slug>` (default: `default`)
  - `--thinking-mode <fast|slow|custom>` (writes middleware overrides)
  - `--backend <url>` (default: `http://localhost:8080`)
  - `--cookie <k=v>` (optional cookie injection)
  - `--json` (print raw JSON response)

**Command: `ws`**

- Purpose: Connect to `/ws` and stream frames.
- Flags:
  - `--conv-id <uuid>` (required)
  - `--profile <slug>` (optional)
  - `--backend <url>`
  - `--timeout <duration>`
  - `--ping-interval <duration>`
  - `--pretty` (pretty print frames)
  - `--filter-type <prefix>` (e.g. `webagent.thinking`) 

**Command: `timeline`**

- Purpose: Fetch snapshot and show entity kinds.
- Flags:
  - `--conv-id <uuid>`
  - `--since-version <n>`
  - `--limit <n>`

**Command: `run`** (or `smoke`)

- Purpose: Combined workflow:
  1. Open WebSocket.
  2. POST /chat.
  3. Log streaming frames.
  4. Fetch /timeline.

---

## 5) CLI Behavior Details

### 5.1 Conversation ID Handling

- If `--conv-id` is not supplied, generate a UUID.
- Always print the chosen `conv_id` first so users can reuse it.

### 5.2 Override Construction

If `--thinking-mode` is set, build the `overrides` section like this:

```json
{
  "overrides": {
    "middlewares": [
      { "name": "webagent-thinking-mode", "config": { "mode": "fast" } }
    ]
  }
}
```

This mirrors the UI behavior (see `web-agent-example/web/src/App.tsx`).

### 5.3 WebSocket Connection Strategy

Common “empty stream” causes:

- Conv ID mismatch: UI and backend have different `conv_id` values.
- Missing `profile` or cookie mismatch; server rejects or attaches to a different profile.
- WebSocket opened *before* conversation exists. (The server should still accept, but conv may be idle.)

Mitigation in CLI:

- Option A: Start `/ws` first, then `/chat` with same `conv_id`.
- Option B: POST `/chat` first, then connect `/ws`.
- Print `ws.hello` frame; absence indicates handshake failure.

### 5.4 Frame Decoding

For frames shaped like:

```json
{"sem": true, "event": { "type": "...", "id": "...", "data": "..." }}
```

We should:

- Detect `sem=true`.
- Parse `event.type` and `event.id`.
- Attempt to decode `event.data` as JSON if possible; otherwise print base64/raw.

### 5.5 Timeline Correlation

After streaming, call `/timeline?conv_id=<id>` and list:

- entity kind counts
- first/last timestamps

This helps confirm that the server is emitting events even if the WS connection is silent.

---

## 6) Architecture Sketch

```
         ┌────────────────────┐
         │   web-agent-debug  │
         │  (CLI executable)  │
         └─────────┬──────────┘
                   │
        ┌──────────┴──────────┐
        │ HTTP /chat request  │
        │   JSON payload      │
        └──────────┬──────────┘
                   │
                   ▼
         ┌────────────────────┐
         │ pinocchio webchat  │
         │  /chat, /ws, /timeline
         └─────────┬──────────┘
                   │
        ┌──────────┴──────────┐
        │ WebSocket /ws frames│
        │   (SEM envelope)    │
        └─────────────────────┘
```

---

## 7) Pseudocode

### 7.1 Combined “run” flow

```pseudo
function run(prompt, thinkingMode):
    convID = userConvID or uuid()
    print("conv_id:", convID)

    ws = connect_ws("/ws?conv_id=" + convID + "&profile=default")
    spawn(ws_reader(ws))

    payload = { conv_id: convID, prompt: prompt }
    if thinkingMode:
        payload.overrides.middlewares = [{ name: "webagent-thinking-mode", config: { mode: thinkingMode } }]

    resp = POST /chat with payload
    print(resp)

    sleep(2s)
    snapshot = GET /timeline?conv_id=convID
    print(snapshot summary)

    close ws
```

### 7.2 WebSocket reader

```pseudo
function ws_reader(ws):
    while ws open:
        msg = ws.read()
        if msg is JSON and msg.sem == true:
            print("SEM", msg.event.type, msg.event.id)
            print(decode(msg.event.data))
        else:
            print("RAW", msg)
```

---

## 8) Files and Symbols to Reference

**Server endpoints**

- `pinocchio/pkg/webchat/router.go`:
  - `/ws` handler (websocket, ws.hello, ws.ping/pong)
  - `/chat` handler (POST, BuildEngineFromReq, PrepareRun)
  - `/timeline` handler (snapshot JSON)

**Client overrides**

- `web-agent-example/web/src/App.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx` (payload format)

**Custom thinking mode**

- `web-agent-example/pkg/thinkingmode/middleware.go`
- `web-agent-example/pkg/thinkingmode/events.go`
- `web-agent-example/pkg/thinkingmode/sem.go`
- `web-agent-example/pkg/thinkingmode/timeline.go`

---

## 9) Debugging Playbook (CLI‑first)

1. **Confirm backend is alive**
   - `curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/chat`
   - Expected: `405` for GET, `200` for POST.

2. **Open WebSocket manually**
   - `web-agent-debug ws --conv-id <id>`
   - Expected: `ws.hello` frame.

3. **Send prompt**
   - `web-agent-debug chat --conv-id <id> --prompt "hello" --thinking-mode fast`

4. **Observe streaming**
   - Use `web-agent-debug ws` output to confirm streaming events appear.

5. **Correlate with timeline**
   - `web-agent-debug timeline --conv-id <id>`
   - Check `webagent_thinking_mode` entity appears.

---

## 10) CLI UX Examples

**Single run**

```bash
web-agent-debug run \
  --backend http://localhost:8080 \
  --prompt "hello" \
  --thinking-mode fast
```

**Separate chat + websocket**

```bash
web-agent-debug ws --conv-id 4212f97e-ca69-45ac-9d90-290cdb483fde
web-agent-debug chat --conv-id 4212f97e-ca69-45ac-9d90-290cdb483fde --prompt "hello"
```

---

## 11) Open Questions

- Should the CLI be hosted in `web-agent-example` or `pinocchio/cmd/web-chat`?
- Do we need a “profile selector” for multi‑profile environments?
- Should the CLI support Redis stream correlation when WS is silent?

---

## 12) Suggested Implementation Plan

1. Create `cmd/web-agent-debug` (or add to `web-agent-example`):
   - `chat`, `ws`, `timeline`, `run` subcommands.
2. Implement HTTP client with cookie jar and JSON body support.
3. Implement WS client (use `gorilla/websocket` to align with server).
4. Add SEM frame decoding and pretty printer.
5. Add timeline snapshot summarizer.

---

## 13) Mini‑Exercises

1. Modify `web-agent-debug ws` to send `ws.ping` every 5s.
2. Add a `--filter-type` flag that only prints matching SEM events.
3. Add `--raw` option that prints raw JSON frames without decoding.
4. Write a test that verifies `ws.hello` is received within 2s after connect.

---

## 14) Summary

A CLI harness gives us an objective, scriptable way to validate `/chat` and `/ws` behavior. It will make “empty websocket” debugging reproducible, and it will help validate custom middlewares like thinking‑mode events without requiring the browser UI.

---

## 15) Implementation Status (2026-02-04)

The CLI harness is now implemented in `web-agent-example/cmd/web-agent-debug` with the four subcommands described above. This section documents current usage so an intern can run it immediately.

### Quickstart

```bash
# from repo root
cd web-agent-example

# send a prompt (auto-generates conv_id if omitted)
go run ./cmd/web-agent-debug chat --prompt "hello" --thinking-mode fast

# connect to websocket (replace conv_id)
go run ./cmd/web-agent-debug ws --conv-id <conv_id>

# fetch timeline summary (replace conv_id)
go run ./cmd/web-agent-debug timeline --conv-id <conv_id>

# one-shot smoke test (ws + chat + timeline)
go run ./cmd/web-agent-debug run --prompt "hello" --thinking-mode fast
```

### Debug Server Logs

When diagnosing streaming issues, run the server with debug logs:

```bash
go run ./cmd/web-agent-example serve \
  --addr :8080 \
  --timeline-db /tmp/web-agent-example-timeline.db \
  --log-level debug
```

### Notes

- The `ws` command defaults to sending a "ping" every 5 seconds.
- Use `--filter-type webagent.thinking` to focus on custom events.
- `run` will wait 2 seconds before fetching `/timeline` (override with `--timeline-delay`).
