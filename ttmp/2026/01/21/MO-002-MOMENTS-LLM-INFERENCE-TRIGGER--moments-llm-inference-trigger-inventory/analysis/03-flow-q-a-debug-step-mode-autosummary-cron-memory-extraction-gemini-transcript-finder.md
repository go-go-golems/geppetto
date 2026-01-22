---
Title: 'Flow Q&A: debug step mode, autosummary cron, memory extraction, Gemini transcript finder'
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
    - Path: moments/backend/pkg/autosummary/cron_worker.go
      Note: Autosummary cron batch loop
    - Path: moments/backend/pkg/autosummary/summary_client.go
      Note: Autosummary calls /rpc/v1/chat/coaching-conversation-summary with internal JWT and new conv_id
    - Path: moments/backend/pkg/doclens/retrieval/gemini.go
      Note: GeminiTranscriptFinder Drive query + ranking (not an LLM call)
    - Path: moments/backend/pkg/memory/llm_extractor.go
      Note: OpenAICompleter implementation (direct OpenAI chat.completions call)
    - Path: moments/backend/pkg/memory/transcript_extractor.go
      Note: memories.extract_from_transcript queues transcript content for ingestion
    - Path: moments/backend/pkg/webchat/loops.go
      Note: ToolCallingLoop (RunInference loop) and step-mode pause points
    - Path: moments/backend/pkg/webchat/router.go
      Note: Chat handlers
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T09:31:38.742713789-05:00
WhatFor: ""
WhenToUse: ""
---

# Flow Q&A (Moments): debug step mode, autosummary cron, memories extraction, Gemini transcript finder

This document answers the specific “how does X work?” questions that came up while inventorying all Moments endpoints that can trigger Geppetto inference. It focuses on **actual control/data flow** and “where does state live?”.

## Terms (so the rest is unambiguous)

- **Conversation (`conv`)**: an in-memory `*webchat.Conversation` stored in a `ConvManager` map (not persisted). See `moments/backend/pkg/webchat/conversation.go:24`.
- **Turn**: a `*turns.Turn` inside the conversation (`conv.Turn`). This is the mutable “LLM transcript/blocks + tool calls” state that the loop updates. See `moments/backend/pkg/webchat/conversation.go:24`.
- **Geppetto inference**: `engine.Engine.RunInference(ctx, turn)` invoked by `ToolCallingLoop`. See `moments/backend/pkg/webchat/loops.go:25`.
- **Step mode**: a pause/continue mechanism around tool execution driven by `StepController` (per conversation). See `moments/backend/pkg/webchat/step_controller.go:9`.

---

## 1) How do `debug/step-mode` and `debug/continue` find the conversation and resume it?

### Conversation lookup (what it is and where it lives)

The debug endpoints accept a `conv_id` (and `pause_id` for continue) and then look up the conversation **in memory**:

- `debug/continue` → `conv := r.getConvByID(conv_id)` → `conv.StepCtrl.Continue(pause_id)` in `moments/backend/pkg/webchat/router.go:359`.
- `debug/step-mode` → `conv := r.getConvByID(conv_id)`; creates/enables `conv.StepCtrl` in `moments/backend/pkg/webchat/router.go:401`.

`getConvByID` is a simple map lookup in `ConvManager` (`r.cm.conns[id]`) — no DB, no redis, no filesystem. See `moments/backend/pkg/webchat/conversation.go:138`.

Implication (food-inspector note): **if the server process restarts**, the conversation map is lost and `debug/continue` will return 404 even if the client “remembers” the same `conv_id`.

### Ownership + auth checks

Even when debug endpoints are enabled, handlers require a session and enforce ownership:

- Session from context and current user check: `moments/backend/pkg/webchat/router.go:378`.
- Ownership check: `conv.OwnerUserID` must match session user id, or is adopted on first `step-mode` call. See `moments/backend/pkg/webchat/router.go:435`.

### Where do “turns” come from / how does it “resume turns to access”?

There is no “turn resume by id” lookup. The “turns to access” are whatever is currently in memory:

- The active run loop uses `conv.Turn` as its state container and updates it as it proceeds. See `moments/backend/pkg/webchat/router.go:944` and `moments/backend/pkg/webchat/router.go:948`.
- Debug endpoints do not fetch or mutate `conv.Turn` directly; they only affect blocking/unblocking via `StepCtrl`.

So “resume” is literally: **unblock the goroutine currently paused inside `ToolCallingLoop`**.

### Step mode control flow (sequence diagram)

```text
Client (UI)                      Moments backend                     Geppetto
-----------                      ---------------                     --------
POST /rpc/v1/chat {conv_id,...}  handleChatRequest()                  (none)
                                 getOrCreateConv() -> conv.Turn
                                 start goroutine run loop
                                                                RunInference(turn)
                                                                -> produces pending tool calls
                                 ToolCallingLoop notices pending tools
                                 StepCtrl.Pause() -> emits debugger.pause(pause_id)
                                 StepCtrl.Wait(pause_id)  <--- blocks here

WebSocket stream receives debugger.pause(pause_id)

POST /rpc/v1/chat/debug/continue {conv_id,pause_id}
                                 getConvByID(conv_id)
                                 StepCtrl.Continue(pause_id) -> closes channel
                                 StepCtrl.Wait returns; loop continues
```

### The actual “pause points”

Step mode pauses in two places in `ToolCallingLoop`:

1. **After inference** (only if there are pending tool calls): `moments/backend/pkg/webchat/loops.go:69`.
2. **After tool execution**, before next inference iteration: `moments/backend/pkg/webchat/loops.go:158`.

Each pause emits a `debugger.pause` event with a fresh `pause_id` and then blocks up to 30s:

- Create pause + publish event: `moments/backend/pkg/webchat/loops.go:74`.
- Wait with timeout: `moments/backend/pkg/webchat/loops.go:81`.

---

## 2) How does `transcript-auto-summary` (cron) manage a conversation, and how many inferences does it run?

### Where is it triggered “from”?

There are two ways the autosummary worker can be triggered:

1. **In-process cron service** (if enabled): `App.initCronService` starts `CronService.StartAndWatch` reading `config/crontab.yaml`. See `moments/backend/pkg/app/app.go:485`.
2. **Manual admin trigger**: `POST /api/v1/admin/cron/{name}/trigger`. See `moments/backend/pkg/admin/cron_trigger.go:41`.

The schedule itself is defined here:

- `transcript-auto-summary` → topic `autosummary.transcript-summary` with args `batch_size: 5`, `max_transcripts_per_coach: 3`. See `moments/config/crontab.yaml:29`.

The Moments server subscribes to that topic and hands messages to the autosummary cron worker:

- Subscriber loop: `moments/backend/cmd/moments-server/serve.go:666`.

### What “conversation” does it use?

Autosummary doesn’t “resume” a user’s existing chat conversation. For each transcript it summarizes it:

- Creates a fresh `ConvID: uuid.NewString()` in the chat request. See `moments/backend/pkg/autosummary/summary_client.go:115`.
- Calls `POST /rpc/v1/chat/coaching-conversation-summary`. See `moments/backend/pkg/autosummary/summary_client.go:130`.
- Auths using an **internal JWT** with `SessionID: "autosummary-cron"`. See `moments/backend/pkg/autosummary/summary_client.go:194`.

So each transcript summary is a brand new “mini chat run” with its own `conv_id`.

### How many Geppetto inferences does it run?

There are two relevant “loops”:

1. **Autosummary cron loop (outer)**: coaches × transcripts
   - `batch_size` coaches per cron tick. Parsed at `moments/backend/pkg/autosummary/cron_worker.go:100`.
   - `max_transcripts_per_coach` per coach per tick. Parsed at `moments/backend/pkg/autosummary/cron_worker.go:105`.
   - Then the worker calls `GenerateSummary` once per selected transcript: `moments/backend/pkg/autosummary/cron_worker.go:290`.

2. **Chat inference loop (inner)**: per transcript summary chat run
   - `handleChatRequest` sets `maxIterations := 5` unless overridden by profile. See `moments/backend/pkg/webchat/router.go:934`.
   - `ToolCallingLoop` performs up to `cfg.MaxIterations` calls to `eng.RunInference`. See `moments/backend/pkg/webchat/loops.go:63`.

Concrete upper bound per cron tick based on current `crontab.yaml`:

- Up to `5 coaches * 3 transcripts = 15` chat runs per cron message. See `moments/config/crontab.yaml:33`.
- Each chat run can do up to `5` `RunInference` calls by default. See `moments/backend/pkg/webchat/router.go:934` and `moments/backend/pkg/webchat/loops.go:63`.
- Upper bound: `15 * 5 = 75` Geppetto inference calls per cron tick (ignoring early exits when no tools remain).

Reality check / “gotchas”:

- If the profile causes the loop to exit earlier (no pending tools), it returns before hitting 5 iterations. See `moments/backend/pkg/webchat/loops.go:84`.
- If the profile sets `MaxIterations`, the bound changes. (Example: other profiles set it higher, e.g. `MaxIterations: 20` in `moments/backend/pkg/app/profiles.go:82`.)

### Autosummary’s job/skip logic (why it might run fewer)

Autosummary is job-tracked and deliberately skips transcripts that are:

- Already completed/exhausted.
- Still processing/pending (unless stale).
- Failed but still within retry delay.

This logic is in `shouldSkipTranscript`. See `moments/backend/pkg/autosummary/cron_worker.go:410`.

---

## 3) Who calls `autosummary/.../retry`, and how does that interact with chat?

### Who calls it?

It’s an **admin-only** endpoint:

- `POST /api/v1/admin/autosummary/jobs/:id/retry` handled by `AdminHandler.handleRetryJob`. See `moments/backend/pkg/autosummary/http_handlers.go:275`.
- It is only registered when autosummary is enabled and `app.InternalJWT != nil` (admin auth middleware). See `moments/backend/cmd/moments-server/serve.go:299`.

I did not find a Moments web frontend call site for this route during the prior frontend scan; it appears intended for an admin dashboard / manual operations rather than normal chat UX.

### What does it do (and what it does NOT do)?

`/retry` **does not run any inference** and does not call `/rpc/v1/chat`.

It mutates DB state to make cron pick the job up again:

- Resets `retry_count` to `0`, sets `last_attempted_at` to epoch, clears error/completed fields. See `moments/backend/pkg/autosummary/repository.go:205`.
- Keeps job status as `failed` so `shouldSkipTranscript` will consider it eligible immediately (bypassing retry delay). See `moments/backend/pkg/autosummary/repository.go:207`.

Interaction with chat:

- The *next* cron attempt that sees this transcript will call `GenerateSummary(...)` again, which calls `/rpc/v1/chat/coaching-conversation-summary` with a new `conv_id`. See `moments/backend/pkg/autosummary/cron_worker.go:290` and `moments/backend/pkg/autosummary/summary_client.go:130`.
- There is no “resume the same chat conversation” semantics here; it is an independent server-driven run.

---

## 4) What is the handler for `/rpc/v1/chat/coaching-conversation-summary`?

It is not a separate handler. It’s the **profile path variant** of the same chat handler:

- `/rpc/v1/chat` → `ChatHandler` (no explicit profile in path). See `moments/backend/cmd/moments-server/serve.go:139`.
- `/rpc/v1/chat/{profile}` → `ChatProfileHandler`. See `moments/backend/cmd/moments-server/serve.go:142`.

The profile slug is parsed from:

- Query param `?profile=...` or `{profile}` path var or `/chat/<slug>` path parsing. See `moments/backend/pkg/webchat/router.go:642`.

So:

- `/rpc/v1/chat/coaching-conversation-summary` is equivalent to calling `/rpc/v1/chat/{profile}` with `{profile} = coaching-conversation-summary`.

Does it run a “single inference tool calling loop”?

- It runs **one `ToolCallingLoop(...)` per request** in a goroutine. See `moments/backend/pkg/webchat/router.go:874`.
- That loop can execute **multiple Geppetto inferences** (iterations) and tool calls until completion or safety cap. See `moments/backend/pkg/webchat/loops.go:63`.

The profile descriptor for `coaching-conversation-summary` is defined at `moments/backend/pkg/app/profiles.go:201`.

---

## 5) How does `conv.StepCtrl` work, and what is `conv`?

### What is `conv`?

`conv` is a per-conversation in-memory struct holding:

- identity/session (`Session`, `OwnerUserID`)
- the active Geppetto `Engine`
- the active `Turn`
- streaming sinks/subscriber state
- optional `StepCtrl` for step mode

See `moments/backend/pkg/webchat/conversation.go:24`.

It is created (or reused) by `getOrCreateConv`, which also composes/recomposes the engine for the requested profile. See `moments/backend/pkg/webchat/conversation.go:215`.

### What is `StepCtrl`?

`StepCtrl` is `*StepController`, which is basically:

- a boolean `enabled`
- a map `waiters[pause_id] -> chan struct{}`

See `moments/backend/pkg/webchat/step_controller.go:9`.

Mechanics (pseudocode approximating the implementation):

```text
Enable():
  enabled = true

Pause(pauseID):
  if not enabled: return now
  waiters[pauseID] = make(chan)
  return now + 30s (deadline for UI)

Wait(pauseID, timeout):
  ch := waiters[pauseID]
  if ch is nil: return
  select:
    case <-ch: return
    case <-time.After(timeout):
      Continue(pauseID)

Continue(pauseID):
  ch := waiters[pauseID]
  delete(waiters, pauseID)
  close(ch) (if non-nil)
```

See `moments/backend/pkg/webchat/step_controller.go:20`.

How step mode is enabled:

- Via debug endpoint (`POST /rpc/v1/chat/debug/step-mode`). See `moments/backend/pkg/webchat/router.go:401`.
- Or by request override (`body.Overrides["step_mode"] == true`). See `moments/backend/pkg/webchat/router.go:866`.

---

## 6) How does `memories.extract_from_transcript` work, and how does it call LLM providers?

### High-level behavior

`memories.extract_from_transcript` is an RPC endpoint that **queues** transcript content for extraction; it does not synchronously run the extraction in the request handler:

- Route: `/rpc/v1/memories.extract_from_transcript`. See `moments/backend/pkg/memory/rpc.go:665`.
- Handler calls `TranscriptExtractor.ExtractFromTranscript(...)`. See `moments/backend/pkg/memory/rpc.go:695`.

`ExtractFromTranscript` does:

1. Dedup: checks if transcript already processed for person; returns completed if so (unless `reprocess`). See `moments/backend/pkg/memory/transcript_extractor.go:77`.
2. Fetch transcript content from an existing **artifact** whose `Sources` JSON contains the transcript drive file id. See `moments/backend/pkg/memory/transcript_extractor.go:100`.
3. Enqueue an ingestion job into the document-ingestion queue: `IngestAsyncFull(... sourceType="transcript" ...)`. See `moments/backend/pkg/memory/transcript_extractor.go:106`.

### Where does the LLM call happen?

The LLM call happens asynchronously inside the **document worker** loop, one chunk at a time:

- Worker reads jobs from a channel and processes them. See `moments/backend/pkg/memory/document_ingester.go:301`.
- For each chunk, it calls `w.extractor.ExtractFromText(ctx, chunk)`. See `moments/backend/pkg/memory/document_ingester.go:355`.

The extractor is wired in `moments-server`:

- It constructs an OpenAI completer and uses it if configured, otherwise uses a noop extractor. See `moments/backend/cmd/moments-server/serve.go:200`.

The extraction path is:

- `LLMDocumentExtractor.ExtractFromText` → `LLMCompleter.Complete(...)`. See `moments/backend/pkg/memory/llm_extractor.go:53`.
- `LLMCompleter` is a simple interface (`Complete(ctx,prompt)`). See `moments/backend/pkg/memory/classifier.go:11`.
- `OpenAICompleter` implements `Complete` by calling `https://api.openai.com/v1/chat/completions`. See `moments/backend/pkg/memory/llm_extractor.go:194`.

Food-inspector note: this is **not** Geppetto/`/rpc/v1/chat` and therefore is **not** counted as a “Geppetto inference trigger” in the earlier inventory; it is a separate LLM integration.

### LLM call count for a transcript extraction

`DocumentWorker` calls the extractor once per content chunk. See `moments/backend/pkg/memory/document_ingester.go:355`.

So the number of OpenAI calls is roughly:

```text
num_llm_calls ≈ number_of_chunks(transcript_markdown, chunk_size)
```

(Chunk size is configured on the ingester; see `memCfg.IngestionChunkSize` wiring in `moments/backend/cmd/moments-server/serve.go:188`.)

### Diagram: transcript extraction pipeline

```text
Client
  |
  | POST /rpc/v1/memories.extract_from_transcript { transcript_drive_file_id }
  v
RPC handler (memory)
  |
  | TranscriptExtractor.ExtractFromTranscript()
  |  - find transcript content in Artifact.Sources
  |  - enqueue DocumentJob (sourceType="transcript")
  v
DocumentWorker (async goroutine)
  |
  | for each chunk:
  |    LLMDocumentExtractor.ExtractFromText(chunk)
  |      -> OpenAICompleter.Complete(prompt)  (OpenAI chat.completions)
  |    -> create draft memories
  v
DB (memories + job status)
```

---

## 7) How does `GeminiTranscriptFinder` work? Does it call Gemini?

Despite the name, `GeminiTranscriptFinder` does **not** call Gemini (LLM). It is a Drive-backed search + ranking helper:

- It uses a Drive query like “name contains Transcript / Notes by Gemini / … and trashed=false” and optional modifiedTime bounds. See `moments/backend/pkg/doclens/retrieval/gemini.go:164`.
- It calls the Google Drive REST API via `GoogleDriveService.SearchFiles`. See `moments/backend/pkg/doclens/retrieval/gemini.go:95` and `moments/backend/pkg/google/auth/drive_service.go:131`.
- It ranks results either by:
  - keyword match in titles, or
  - recency (modifiedTime),
  depending on query + sort mode. See `moments/backend/pkg/doclens/retrieval/gemini.go:225`.

### Core algorithm (pseudocode)

```text
Search(personID, query, topK, sortMode, from, to):
  driveQuery := buildGeminiDriveQuery(from, to)
  files := driveSvc.SearchFiles(personID, driveQuery, limit=computeSearchLimit(topK))
  docs := convertDriveFiles(files)

  keywords := normalizeKeywords(query)  # up to 5 tokens
  if sortMode == auto:
    sortMode = keyword if query non-empty else recency

  if sortMode == keyword and keywords non-empty:
    docs = sort by keywordScore(title, keywords) desc, then modifiedTime desc
  else:
    docs = sort by modifiedTime desc

  return docs[:clampTopK(topK)]
```

### Where is it used?

The most obvious use is team discovery:

- `DiscoverTeamFromTranscripts` calls `transcriptFinder.Search(...)` with `DefaultMaxTranscripts` and recency sort. See `moments/backend/pkg/teamdiscovery/service.go:63`.
- Then it enumerates the returned Drive file IDs and fetches permissions to infer teammates. See `moments/backend/pkg/teamdiscovery/service.go:107`.

So: `GeminiTranscriptFinder` is a “drive transcript locator” plus ranking heuristic; the “Gemini” label appears to refer to **Gemini-generated notes** that live in Drive (e.g., “Notes by Gemini”), not a Gemini model invocation.
