---
Title: PI-010 Webchat Debug Work Review
Ticket: PI-010-DEBUG-WEB-AGENT-SETUP
Status: active
Topics:
    - backend
    - cli
    - debugging
    - webchat
    - websocket
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/router.go
      Note: |-
        Webchat router with debug logs and /chat + /ws handlers
        Webchat router with ws/chat handling and debug logs
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Timeline upsert behavior for SEM events
    - Path: pinocchio/pkg/webchat/timeline_store_memory.go
      Note: In-memory timeline fallback
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: Durable timeline persistence used for hydration
    - Path: web-agent-example/cmd/web-agent-debug/chat.go
      Note: |-
        CLI /chat implementation
        CLI chat request handling
    - Path: web-agent-example/cmd/web-agent-debug/main.go
      Note: CLI harness entrypoint
    - Path: web-agent-example/cmd/web-agent-debug/run.go
      Note: |-
        Combined chat/ws/timeline run command
        Combined ws/chat/timeline workflow
    - Path: web-agent-example/cmd/web-agent-debug/timeline.go
      Note: CLI timeline fetch + summary
    - Path: web-agent-example/cmd/web-agent-debug/ws.go
      Note: |-
        CLI websocket client implementation
        CLI websocket client
    - Path: web-agent-example/cmd/web-agent-example/engine_from_req.go
      Note: |-
        External engine builder (no-cookie profile override)
        No-cookie engine builder for external use
    - Path: web-agent-example/cmd/web-agent-example/main.go
      Note: Web-agent-example server wiring
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-05T03:36:00-05:00
WhatFor: Review PI-010 webchat debugging work for correctness, ergonomics, and externalization readiness.
WhenToUse: When assessing webchat integration quality or preparing external webchat use.
---


# PI-010 Webchat Debug Work Review

## Executive Summary

PI-010 produced a focused debugging harness and better observability around webchat streaming. The CLI (`web-agent-debug`) now simulates the browser’s `/chat` → `/ws` → `/timeline` flow and can be used to reproduce “empty websocket” reports from the terminal. Alongside that, debug logging in `pinocchio/pkg/webchat/router.go` surfaces the **resolved `conv_id`/profile**, which is where the real mismatch bugs tend to hide.

The work is **functionally correct and valuable**, but still somewhat **internal‑friendly** rather than **externally‑friendly**. To make webchat a clean, third‑party‑facing subsystem, we should take the current CLI + router behavior and add:

- A stable external contract for `/chat`, `/ws`, `/timeline` that does **not** require cookies.
- A formal “webchat client API” (Go package + JS type schema) to prevent drift.
- A clearer separation between **routing**, **profiling**, and **session identity**.

The remainder of this review explains exactly what we built, where it is strong, and how to harden it for external use.

---

## What Was Actually Built (Concrete Artifacts)

### 1) Router Debug Logs (Pinocchio)

File: `pinocchio/pkg/webchat/router.go`

Key change: debug logs that make the `conv_id` + profile resolution explicit during `/ws` and `/chat`.

**Why it matters:** most “empty websocket” issues come from subtle mismatches:

- `/chat` uses one `conv_id` (created in the client),
- `/ws` uses another (or none),
- `profile` falls back to cookie and points at the wrong registry entry.

The new logs reveal these mismatches without guesswork.

### 2) The CLI Harness (`web-agent-debug`)

Files:

- `web-agent-example/cmd/web-agent-debug/main.go`
- `web-agent-example/cmd/web-agent-debug/chat.go`
- `web-agent-example/cmd/web-agent-debug/ws.go`
- `web-agent-example/cmd/web-agent-debug/timeline.go`
- `web-agent-example/cmd/web-agent-debug/run.go`

This tool is effectively a scripted browser:

- `chat`: POST `/chat` (with overrides)
- `ws`: connect to `/ws`, print SEM frames
- `timeline`: fetch hydration snapshot
- `run`: orchestrate all three in a single smoke test

The `run` command is the most valuable part for debugging in practice.

### 3) No‑cookie Engine Builder (Web Agent Example)

File: `web-agent-example/cmd/web-agent-example/engine_from_req.go`

This isolates the engine request logic **from cookie‑based profile selection**, which is a key step toward external embedding. This is the “first cut” at making webchat more library‑friendly.

---

## How the System Works (With Pseudocode)

### /chat → /ws → /timeline at a glance

```
client submits chat:
  POST /chat { conv_id, prompt, overrides }

server does:
  engineInput = BuildEngineFromReq(req)
  conv = ConvManager.GetOrCreate(engineInput.ConvID)
  run inference
  emit SEM events -> websocket clients + timeline

client subscribes:
  GET /ws?conv_id=...&profile=...
  stream SEM frames

client hydrates:
  GET /timeline?conv_id=...&since_version=...
```

### Server side (router logic)

```pseudo
handleWS(req):
  log query conv_id/profile
  buildInput = engineFromReq(req)
  log resolved conv_id/profile
  pool = wsPool[conv_id]
  pool.add(conn)
  send ws.hello
  loop: handle ping/pong, forward events

handleChat(req):
  body = decode JSON
  buildInput = engineFromReq(req)
  ensure conv exists
  run inference with overrides
  return JSON response
```

### Client side (CLI harness)

```pseudo
run command:
  conv_id = ensureConvID()
  ws = connect /ws?conv_id
  start ws reader
  POST /chat with same conv_id
  sleep
  GET /timeline
  print summary
```

This is correct and matches the internal API. The review below focuses on external‑use improvements.

---

## Review of Correctness and Observability

### Strong Points (What’s Good)

- **End‑to‑end reproduction**: the CLI proves the streaming pipeline works without the browser.
- **Explicit logging** in `router.go` surfaces the highest‑value debugging data (resolved `conv_id`, profile, override presence).
- **Timeline validation**: being able to correlate websocket events with persisted snapshots is essential.
- **No-cookie engine builder**: solid first move toward a clean external API.

### Remaining Friction (Where It’s Thin)

- **Profile selection still mixes sources**:
  - In `pinocchio/pkg/webchat`, profile can come from URL path, query params, or cookies.
  - This is convenient for internal webchat, but confusing for external consumers.
- **No stable contract docs** for `/chat` and `/ws` (what fields are required? which are optional? what happens on mismatch?).
- **No explicit versioning** of the webchat API. This makes external integration risky.
- **Debug output is informal**: logs help developers, but external users need structured error messages and stable failure modes.

---

## External‑Use Readiness Review

Below is the most important section for future externalization. It is blunt by design.

### 1) Session Identity: `conv_id` Should Be First‑Class

Currently:

- `/chat` can create a `conv_id` if missing (server‑side).
- `/ws` requires `conv_id` in query.
- cookies can still sneak in for profile selection.

**Risk:** external clients can accidentally connect a websocket to a different conversation than their `/chat` call. This is the most common “empty websocket” bug.

**Recommendation:**

- Treat `conv_id` as mandatory in both `/chat` and `/ws`.
- If missing, return a **422** with a JSON error payload:

```json
{
  "error": "missing_conv_id",
  "message": "conv_id is required for /chat and /ws"
}
```

### 2) Profile Resolution Should Be Explicit

Right now:

- `/chat/<profile>` influences selection.
- `/ws` uses `profile` query or cookie.

For external usage, this is too implicit. You want a single rule:

- Either **all profile selection comes from body** (`overrides.profile`),
- Or **all selection comes from a path or query param**.

**Recommendation:**

- Introduce a consistent `profile` field in the `/chat` body and the `/ws` query param.
- Deprecate cookie‑based profile resolution for external use.

### 3) Document the SEM Envelope Contract

The CLI prints SEM frames, but external developers need a **formal schema**. Example:

```ts
interface SemEnvelope {
  sem: true
  event: {
    type: string
    id: string
    data: unknown
  }
}
```

**Recommendation:**

- Publish a small schema doc alongside the frontend package.
- Export a TypeScript type from the webchat frontend package.

### 4) Publish a “Webchat Client” SDK

Currently, the CLI is the only “client.” External devs need more:

- A Go `webchatclient` package (mirrors CLI but as library).
- A JS `@pwchat/client` package that manages `/chat` + `/ws` + `/timeline` and returns an observable stream.

**Recommendation:**

- Refactor `web-agent-debug` into a reusable client library and a thin CLI wrapper.
- Provide explicit config types and default values.

### 5) Versioned API Surface

External users will want stability. Introduce a version prefix:

```
POST /v1/chat
GET  /v1/ws
GET  /v1/timeline
```

This gives you room for future adjustments without breaking compatibility.

### 6) OAuth / API key / Auth Hooks

Webchat currently assumes “local dev.” Externalization means:

- Pluggable auth hooks (`WithAuthMiddleware`?)
- Proper CORS behavior

**Recommendation:**

- Provide a `RouterOption` to install auth middleware or to validate request headers.
- Add doc examples for secure usage.

---

## Design Improvements (Concrete)

### A) Explicit Engine Build Contract

Today we have `EngineFromReqBuilder` and a custom builder in web‑agent‑example. This is good, but not fully documented.

**Proposed pattern:**

```go
builder := webchat.NewEngineFromReqBuilder(
  webchat.WithConvIDResolver(...),
  webchat.WithProfileResolver(...),
  webchat.WithOverridesDecoder(...),
)

router, _ := webchat.NewRouter(
  ctx,
  parsed,
  staticFS,
  webchat.WithEngineFromReqBuilder(builder),
)
```

Make each resolver composable and testable. This makes external embedding significantly easier.

### B) Make “profile cookies” opt‑in

For external embed, implicit cookies are noise. Provide:

```go
webchat.WithProfileCookie(false)
```

Then only explicit profile selection is used.

### C) Surface an Official CLI for Debugging

The `web-agent-debug` harness is excellent and should be promoted to a first‑class `pinocchio webchat debug` command. That makes external users discover it and trust it.

---

## Things the Junior Implementation Got Right

Even if you aren’t happy with the engineering experience, the core technical choices are solid:

- **Protocol parity with the frontend**: CLI uses exactly the same endpoints and payload shape as the UI.
- **Minimal moving parts**: No hidden state; the CLI is reproducible.
- **Unit‑test coverage** (small but relevant): each CLI command was run with `go test` after implementation.

---

## Things That Should Be Tightened

These are not wrong, but they are places that should be refined before external use.

- **Error semantics**: return structured error JSON, not generic text.
- **Consistency**: remove hidden cookie fallback in external profiles.
- **Schema versioning**: formalize `/chat` response shape and SEM event schemas.
- **Client library**: let external developers use a “real API,” not a manual set of requests.

---

## Suggested Next Steps (Externalization Roadmap)

1. **Write a stable webchat API spec** (docs + TS types).
2. **Refactor CLI into library + thin CLI wrapper**.
3. **Introduce versioned endpoints**.
4. **Add a “no cookie” mode** in pinocchio router (or make it the default for embedded use).
5. **Provide a single official external integration guide** with step‑by‑step usage examples.

---

## Appendix: Minimal External Client (Pseudo)

```pseudo
client = NewWebchatClient("https://my-server")
conv_id = uuid()

ws = client.connectWS(conv_id, profile="default")
ws.onEvent = renderSemEvent

client.chat(conv_id, "hello", overrides={middlewares:[...]})

snapshot = client.timeline(conv_id)
render(snapshot)
```

This is the ergonomic shape external developers want. The PI‑010 CLI already does this; we should package it properly.

---

## Closing Note

The PI‑010 work is good engineering in the “debugger harness” category. It solves the immediate operational pain (empty websocket) and gives developers a reproducible workflow. If we now formalize the API and turn the harness into a supported external client, the webchat system becomes a genuinely reusable subsystem rather than an internal tool.
