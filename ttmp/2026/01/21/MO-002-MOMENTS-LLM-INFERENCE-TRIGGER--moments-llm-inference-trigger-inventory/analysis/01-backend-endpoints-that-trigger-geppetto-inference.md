---
Title: 'Backend: Endpoints That Trigger Geppetto Inference'
Ticket: MO-002-MOMENTS-LLM-INFERENCE-TRIGGER
Status: active
Topics:
    - moments
    - backend
    - frontend
    - llm
    - inference
    - geppetto
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: moments/backend/cmd/moments-server/serve.go
      Note: Registers chat routes on mux router (primary HTTP entrypoint)
    - Path: moments/backend/pkg/admin/cron_trigger.go
      Note: Admin cron trigger endpoint that can indirectly start autosummary
    - Path: moments/backend/pkg/autosummary/http_handlers.go
      Note: Admin retry endpoint that can indirectly re-trigger chat inference
    - Path: moments/backend/pkg/autosummary/summary_client.go
      Note: Indirect trigger path; calls /rpc/v1/chat/coaching-conversation-summary
    - Path: moments/backend/pkg/webchat/loops.go
      Note: Only backend call site of geppetto Engine.RunInference (ToolCallingLoop)
    - Path: moments/backend/pkg/webchat/router.go
      Note: HTTP handlers for /rpc/v1/chat*
    - Path: moments/config/crontab.yaml
      Note: Schedule names/topics that determine which cron triggers lead to chat inference
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-21T12:47:20.299010605-05:00
WhatFor: ""
WhenToUse: ""
---


# Backend: Endpoints That Trigger Geppetto Inference

## Scope + Definitions

This document answers:

> “In the Moments **backend**, which **HTTP endpoints** can cause a **Geppetto inference** to happen?”

### What counts as “Geppetto inference”

For this audit, “Geppetto inference” means: execution reaches `engine.Engine.RunInference(ctx, turn)` from `github.com/go-go-golems/geppetto/pkg/inference/engine`.

That is the *moment the backend hands a Turn to the Geppetto inference engine*, which in turn will call an LLM provider (directly or via Geppetto steps).

### What counts as an “endpoint that can trigger it”

An endpoint is considered a **trigger** if calling it can (under some conditions) result in `RunInference(...)` being invoked:

- **Direct trigger**: the endpoint’s request path starts a run loop that calls `RunInference(...)`.
- **Control trigger**: the endpoint does not itself call `RunInference(...)`, but can *resume* an already-started run such that the pending `RunInference(...)` or subsequent `RunInference(...)` executes.
- **Indirect trigger**: the endpoint causes some worker/subsystem to call a direct-trigger endpoint (e.g., “admin triggers cron” → “cron worker calls chat endpoint”).

This document focuses on Geppetto inference triggers. It includes a small appendix for “LLM calls that are *not* Geppetto inference” because they are easy to confuse during incidents.

## Executive Summary (the short list)

### Direct Geppetto inference triggers

- `POST /rpc/v1/chat`
- `POST /rpc/v1/chat/{profile}`

### Inference-control endpoints (only when enabled)

These are only registered when `services.mento.webchat.enable-debug-endpoints=true`:

- `POST /rpc/v1/chat/debug/step-mode` (enables/disables step pauses)
- `POST /rpc/v1/chat/debug/continue` (resumes a paused step)

### Indirect triggers (they don’t call Geppetto directly, but can cause it)

- `POST /api/v1/admin/cron/{name}/trigger` (when `{name}` is `transcript-auto-summary`)
- `POST /api/v1/admin/autosummary/jobs/{id}/retry` (resets a failed job so cron will re-run it, which calls chat)

## Methodology (“food inspector” checklist)

1. Find the **ground truth call sites** of Geppetto inference in backend code:
   - Search for `RunInference(` in `moments/backend`.
2. For each call site, identify which HTTP handlers can reach it.
3. Enumerate all relevant routes and any gating conditions (feature flags, settings).
4. Validate “indirect” triggers by tracing worker/cron flows until they hit a direct trigger endpoint.

### Ground truth finding: only one `RunInference(...)` call site

The only direct call site in `moments/backend` is:

- `moments/backend/pkg/webchat/loops.go` → `ToolCallingLoop(...)` → `eng.RunInference(ctx, currentTurn)`

This makes the webchat subsystem the single ingress funnel for Geppetto inference in Moments backend.

## High-level Architecture (how an inference actually happens)

### A. User/API call path (direct triggers)

```
Client
  |
  |  POST /rpc/v1/chat[/<profile>]   (JSON body with prompt + conv_id + context)
  v
moments-server (mux router)
  |
  v
webchat.Router.handleChatRequest(...)
  |
  |  spawn goroutine (async run)
  v
webchat.ToolCallingLoop(...)
  |
  |  i = 0..N
  |    RunInference()  <-- Geppetto inference boundary (LLM call happens downstream)
  |    Extract tool calls
  |    Execute tools
  |    Append tool results
  v
Publishes events to Watermill topic "chat:<conv_id>"
  |
  v
WS clients (GET /rpc/v1/chat/ws) receive semantic event frames
```

### B. Autosummary path (indirect trigger)

```
Admin (or scheduler)
  |
  | POST /api/v1/admin/cron/transcript-auto-summary/trigger
  v
cron.CronService publishes Watermill event to topic "autosummary.transcript-summary"
  |
  v
autosummary.CronWorker.HandleMessage(...)
  |
  | for each eligible transcript:
  |   HTTPSummaryClient.GenerateSummary(...)
  |     POST /rpc/v1/chat/coaching-conversation-summary   <-- direct trigger endpoint
  v
Geppetto inference happens via webchat.ToolCallingLoop (same as A)
```

## Endpoint Inventory (with exact triggers and caveats)

### 1) `POST /rpc/v1/chat` (direct trigger)

**Where it’s registered**
- `moments/backend/cmd/moments-server/serve.go` registers `chatRouter.ChatHandler()` at `POST /rpc/v1/chat`.
- Handler implementation: `moments/backend/pkg/webchat/router.go` → `registerHTTPHandlers()` → `r.chatHandler`.

**Authentication**
- In `serve.go`, handler is wrapped with `identityclient.RequireSession(...)`.
- So a valid session is required for the request to be accepted.

**Why it triggers Geppetto inference**
- `r.chatHandler` calls `r.handleChatRequest(...)`.
- `handleChatRequest` spawns a goroutine that executes `ToolCallingLoop(...)`.
- `ToolCallingLoop` calls `eng.RunInference(...)` unconditionally on its first iteration.

**Important “trigger conditions”**
- If the conversation is already running (`conv.running == true`), the endpoint returns `409 Conflict` and does **not** start a second inference run.
- If the requested profile slug is unknown, it returns `404` and does not run inference.

**Profile selection rules (this endpoint is sneaky)**

`POST /rpc/v1/chat` does **not** take a `{profile}` path param. It chooses a profile by:

1. (Inside `handleChatRequest`) If no explicit profile was provided:
   - If an existing conversation with that `conv_id` exists, reuse `conv.ProfileSlug`.
2. Else, if a cookie `chat_profile` exists, use that.
3. Else, default to `"default"`.

Additionally, `handleChatRequest` can still extract a profile from:
- Query param `?profile=...` (highest priority),
- Path parsing (if the path contains `/chat/<profile>`),
…because `profileSlugFromRequest(req)` is used by the `POST /rpc/v1/chat/{profile}` handler. For `POST /rpc/v1/chat` specifically, the handler passes only the cookie-derived slug; then `handleChatRequest` does the “existing conv / cookie / default” fallback.

**Pseudocode (actual control flow)**

```text
POST /rpc/v1/chat:
  profileSlug := cookie("chat_profile") or ""
  handleChatRequest(req, profileSlug)

handleChatRequest(req, profileSlug):
  body := decode JSON
  convID := body.conv_id or new UUID

  if profileSlug == "":
    if conv exists for convID:
      profileSlug = conv.ProfileSlug
  if profileSlug == "":
    profileSlug = cookie("chat_profile")
  if profileSlug == "":
    profileSlug = "default"

  conv := getOrCreateConv(convID, profileSlug, buildEngineAndSink)
  if conv.running: return 409
  conv.running = true

  append user prompt block to conv.turn
  build tool registry (filtered by profile.DefaultTools)
  attach session, profile prompt, metadata, overrides

  spawn goroutine:
    ToolCallingLoop(ctxWithSession, conv.Engine, conv.Turn, perProfileLoopLimits)
    conv.running = false
```

**What a “trigger” looks like at runtime**
- Request returns immediately with `{ run_id, conv_id }` while the inference run continues asynchronously.
- Streaming output is delivered by event publications and (optionally) WS subscriptions.

---

### 2) `POST /rpc/v1/chat/{profile}` (direct trigger)

**Where it’s registered**
- `moments/backend/cmd/moments-server/serve.go` registers at `POST /rpc/v1/chat/{profile}`.
- Handler implementation: `moments/backend/pkg/webchat/router.go` → `r.chatProfileHandler`.

**Authentication**
- Wrapped with `identityclient.RequireSession(...)`.

**Why it triggers Geppetto inference**
- Same as `POST /rpc/v1/chat`, except profile selection is explicit.

**Profile selection rules**

`profileSlugFromRequest(req)` chooses (in this priority order):
1. Query param `?profile=...` (wins even if the path param is different)
2. Mux vars: `{profile}` path param
3. Fallback parsing of `.../chat/<profile>` from path string

This means the “explicit” endpoint can still be overridden by query string.

**Configured profiles (App defaults)**

Profiles are registered in `moments/backend/pkg/app/profiles.go` and exposed by:
- `GET /rpc/v1/chat/profiles` (non-trigger endpoint, listed here for completeness)

At time of writing, the profile slugs include:
- `default`
- `doclens`
- `team-select`
- `drive1on1`
- `drive1on1-summary`
- `coaching-conversation-summary`
- `coaching-conversations-summary`
- `find-coaching-transcripts`
- `thinking-mode`
- `presidential-debate`
- `questionnaire`

---

### 3) `GET /rpc/v1/chat/ws` (NOT a direct trigger, but inference-adjacent)

**Where it’s registered**
- `moments/backend/cmd/moments-server/serve.go` registers `chatRouter.WebsocketHandler()` at `GET /rpc/v1/chat/ws`.

**What it does**
- Upgrades to websocket and attaches the connection to a conversation.
- Starts a per-conversation subscriber that forwards events to the websocket client.

**Why it does *not* trigger inference by itself**
- The WS handler does not call `ToolCallingLoop` and does not call `RunInference`.
- It can be present during an inference, but it does not start one.

**Security detail**
- The handler supports passing bearer token via query param (`access_token` or `token`) because browsers can’t set headers for WS the same way.

---

### 4) `POST /rpc/v1/chat/debug/step-mode` (control trigger; gated)

**Route availability**
- Only registered when `services.mento.webchat.enable-debug-endpoints=true`.
- Registration happens inside `moments/backend/pkg/webchat/router.go` during `registerHTTPHandlers()`.

**What it does**
- Enables or disables “step mode” on an existing conversation.
- Step mode causes `ToolCallingLoop` to pause at:
  - after `RunInference` when there are pending tool calls
  - after tool execution completes

**Why it can “trigger” inference**
- It can change a running conversation from “free running” to “pausing”, but it does not itself call `RunInference`.
- It is best classified as a **control endpoint**: it affects whether an inference proceeds automatically.

**Authorization**
- Requires session.
- Enforces conversation ownership: the first caller can “adopt ownership” if `OwnerUserID` is empty; subsequent callers must match `OwnerUserID`.

---

### 5) `POST /rpc/v1/chat/debug/continue` (control trigger; gated)

**Route availability**
- Only registered when `services.mento.webchat.enable-debug-endpoints=true`.

**What it does**
- Calls `conv.StepCtrl.Continue(pause_id)`, which releases `ToolCallingLoop` from a pause.

**Why it can “trigger” inference**
- If the loop is paused *before* a subsequent `RunInference` call, continuing can directly cause the next iteration (and thus another `RunInference`) to proceed.
- It is a control-plane trigger rather than a start trigger.

**Authorization**
- Requires session and strict owner match; unlike step-mode, it does not adopt ownership.

---

### 6) `POST /api/v1/admin/cron/{name}/trigger` (indirect trigger; admin-gated)

**Where it’s registered**
- `moments/backend/pkg/admin/cron_trigger.go` under `/api/v1/admin/cron`.
- Wired from `moments/backend/pkg/identityserver/server.go` when identity admin is enabled and a `CronService` exists.

**What it does**
- Publishes the schedule event to the schedule’s configured Watermill topic.

**When it can lead to Geppetto inference**
- If `{name}` is `transcript-auto-summary` (from `moments/config/crontab.yaml`), it publishes to topic `autosummary.transcript-summary`.
- The autosummary worker consumes that topic and uses `HTTPSummaryClient.GenerateSummary(...)`, which calls:
  - `POST /rpc/v1/chat/coaching-conversation-summary`

**Non-Geppetto cron schedule note**
- `transcript-memory-extraction` publishes to `memory.transcript-extraction` (memory subsystem; not Geppetto inference by this document’s definition).

---

### 7) `POST /api/v1/admin/autosummary/jobs/{id}/retry` (indirect trigger; admin-gated)

**Where it’s registered**
- `moments/backend/pkg/autosummary/http_handlers.go` under `/api/v1/admin/autosummary`.

**What it does**
- Resets a failed autosummary job so the next cron run will pick it up.
- The actual summary generation (and Geppetto inference) happens later in the autosummary worker via the chat endpoint call.

## Appendix A: “LLM calls” that are not Geppetto inference (easy to confuse)

These are not counted as Geppetto inference triggers because they do not call `geppetto/pkg/inference/engine.Engine.RunInference(...)` from the Moments backend.

Examples:
- `POST /rpc/v1/memories.extract_from_transcript` starts transcript memory extraction jobs; the extractor path may use provider-specific clients (e.g., OpenAI) directly rather than Geppetto.
- Team discovery uses `retrieval.NewGeminiTranscriptFinder(...)` for transcript search (Gemini), which is LLM-adjacent but not Geppetto inference.

## Appendix B: Pointers (code locations that matter)

- Route wiring:
  - `moments/backend/cmd/moments-server/serve.go`
  - `moments/backend/pkg/identityserver/server.go` (admin + cron wiring)
- Webchat inference:
  - `moments/backend/pkg/webchat/router.go` (`handleChatRequest`, debug endpoints, WS handler)
  - `moments/backend/pkg/webchat/loops.go` (`ToolCallingLoop` and `RunInference` call)
  - `moments/backend/pkg/app/profiles.go` (profile slugs + default middleware/tools)
- Autosummary:
  - `moments/backend/pkg/autosummary/summary_client.go` (calls `POST /rpc/v1/chat/coaching-conversation-summary`)
  - `moments/backend/pkg/autosummary/cron_worker.go` (invokes summary client)
  - `moments/config/crontab.yaml` (schedule names/topics)
