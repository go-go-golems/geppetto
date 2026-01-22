---
Title: Diary
Ticket: MO-002-MOMENTS-LLM-INFERENCE-TRIGGER
Status: active
Topics:
    - moments
    - backend
    - frontend
    - llm
    - inference
    - geppetto
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-21T12:41:50.228822904-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Record an audit-style investigation of all Moments backend endpoints and frontend code paths that can trigger a Geppetto LLM inference, including intermediary steps and dead ends.

## Step 1: Create Ticket Workspace + Diary

Created the docmgr ticket workspace and initialized a dedicated diary doc so all subsequent investigation steps, commands, and findings are recorded in one place.

This step intentionally did not change product code; it only set up the documentation scaffolding needed for a thorough, reproducible audit.

### What I did
- Ran `docmgr status --summary-only` to confirm docmgr root and config.
- Ran `docmgr ticket create-ticket --ticket MO-002-MOMENTS-LLM-INFERENCE-TRIGGER ...` to create the ticket workspace.
- Ran `docmgr doc add --ticket MO-002-MOMENTS-LLM-INFERENCE-TRIGGER --doc-type reference --title "Diary"` to create this diary document.

### Why
- Keep the audit trail (commands + rationale) discoverable and reviewable alongside the final reports.

### What worked
- Ticket workspace created under `geppetto/ttmp/2026/01/21/...`.
- Diary document created at `geppetto/ttmp/2026/01/21/.../reference/01-diary.md`.

### What didn't work
- N/A

### What I learned
- This repo’s docmgr root is `geppetto/ttmp` (per `docmgr status --summary-only`), so ticket docs live alongside Geppetto’s documentation artifacts.

### What was tricky to build
- N/A (no code changes).

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- N/A (docs-only scaffolding).

### Technical details
- Commands run:
  - `docmgr status --summary-only`
  - `docmgr ticket create-ticket --ticket MO-002-MOMENTS-LLM-INFERENCE-TRIGGER --title "Moments: LLM inference trigger inventory" --topics moments,backend,frontend,llm,inference,geppetto`
  - `docmgr doc add --ticket MO-002-MOMENTS-LLM-INFERENCE-TRIGGER --doc-type reference --title "Diary"`

## Step 2: Backend Recon — Find Geppetto Inference Entrypoints

Mapped the backend’s HTTP route registrations and then worked backwards from “where does an actual `geppetto/pkg/inference` engine execute” to identify which endpoints can start or resume a Geppetto inference.

The key discovery is that the only direct `Engine.RunInference(...)` call in the Moments backend is inside the webchat run loop, so all Geppetto LLM calls funnel through the webchat chat endpoints (plus any endpoints that intentionally control those runs).

### What I did
- Scanned `moments/backend/cmd/moments-server/serve.go` for route wiring and “long response” hints.
- Grepped the backend for Geppetto inference imports and `RunInference(` call sites.
- Read the webchat router and loop implementation to confirm when `RunInference` is invoked.
- Checked the autosummary subsystem to see how it triggers summary generation (and whether it calls webchat).
- Located cron/admin endpoints that can indirectly kick off autosummary runs.

### Why
- “Endpoints that can trigger a Geppetto inference” is best answered by locating the exact Go call site(s) where inference happens and enumerating all HTTP ingress paths that reach those call sites (directly or via orchestration endpoints).

### What worked
- Found the only `RunInference(` call site at `moments/backend/pkg/webchat/loops.go` in `ToolCallingLoop`.
- Confirmed `ToolCallingLoop` is started by the HTTP chat handler in `moments/backend/pkg/webchat/router.go` (`handleChatRequest` starts a goroutine and calls `ToolCallingLoop`).
- Confirmed autosummary triggers Geppetto inference by calling `POST /rpc/v1/chat/coaching-conversation-summary` via `moments/backend/pkg/autosummary/summary_client.go`.
- Identified cron/admin endpoints used to trigger autosummary processing (`/api/v1/admin/cron/{name}/trigger`) and to reset jobs for retry (`/api/v1/admin/autosummary/jobs/{id}/retry`).

### What didn't work
- N/A (no blockers; this was investigation-only).

### What I learned
- `GET /rpc/v1/chat/ws` is primarily a streaming attachment endpoint (subscribe + forward events); the actual LLM inference is initiated by `POST /rpc/v1/chat*` handlers which call `ToolCallingLoop`.
- Cron schedules are defined in `moments/config/crontab.yaml`; the “transcript-auto-summary” schedule publishes to `autosummary.transcript-summary`, and the autosummary worker uses the HTTP summary client to call the chat endpoint.

### What was tricky to build
- Distinguishing “LLM usage” vs “Geppetto inference”: several subsystems (e.g., memory extraction) may call LLM providers directly, but Geppetto inference specifically routes through `engine.Engine.RunInference` (webchat).

### What warrants a second pair of eyes
- Validate that no non-webchat HTTP handlers construct and run a Geppetto engine via a different path (I relied on `rg` for `RunInference(` and geppetto inference imports).

### What should be done in the future
- N/A

### Code review instructions
- Start at `moments/backend/pkg/webchat/loops.go` (`ToolCallingLoop`) and `moments/backend/pkg/webchat/router.go` (`handleChatRequest`).
- Then review `moments/backend/pkg/autosummary/summary_client.go` for the internal chat call, and `moments/backend/pkg/admin/cron_trigger.go` / `moments/config/crontab.yaml` for cron-trigger paths.

### Technical details
- Commands run (representative):
  - `rg -n "github\\.com/go-go-golems/geppetto/pkg/inference" moments/backend -S`
  - `rg -n "RunInference\\(" moments/backend/pkg -S`
  - `sed -n '1,420p' moments/backend/cmd/moments-server/serve.go`
  - `sed -n '1,980p' moments/backend/pkg/webchat/router.go`
  - `sed -n '1,240p' moments/backend/pkg/webchat/loops.go`
  - `sed -n '1,220p' moments/backend/pkg/autosummary/summary_client.go`
  - `sed -n '1,220p' moments/config/crontab.yaml`

## Step 3: Write Backend Report + Upload to reMarkable

Turned the investigation into a structured, audit-style report with explicit definitions (“what counts as Geppetto inference”), an endpoint inventory, pseudocode, and diagrams showing the direct and indirect trigger flows.

Then uploaded the report as a PDF to the ticket folder on reMarkable, capturing the sandbox networking failure and the successful retry with network-enabled execution.

### What I did
- Created and wrote the backend report document:
  - `geppetto/ttmp/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER--moments-llm-inference-trigger-inventory/analysis/01-backend-endpoints-that-trigger-geppetto-inference.md`
- Added docmgr `RelatedFiles` links for the main call sites and route wiring.
- Ran `remarquee upload md --dry-run ...` to validate PDF generation + target remote dir.
- Uploaded the report to `/ai/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER/`.

### Why
- The deliverable needs to be reviewable by backend and frontend engineers and usable during incidents; “list of endpoints” alone isn’t enough without call graphs, conditions, and control-plane details.
- Uploading to reMarkable makes it easy to review away from the editor and annotate.

### What worked
- The report cleanly reduces the trigger set to webchat `/rpc/v1/chat*` plus explicit control-plane and indirect cron/admin triggers.
- The final upload succeeded and produced `01-backend-endpoints-that-trigger-geppetto-inference.pdf` in the ticket folder on reMarkable.

### What didn't work
- First upload attempt failed in the sandbox due to missing DNS/network access:
  - `dial tcp: lookup internal.cloud.remarkable.com: no such host`
  - `dial tcp: lookup webapp-prod.cloud.remarkable.engineering: no such host`

### What I learned
- `remarquee upload md` supports `--remote-dir` and `--dry-run`, but not a custom `--name` flag (PDF filename derives from the markdown filename).
- Networked uploads require running outside the network-restricted sandbox.

### What was tricky to build
- Being precise about “trigger” vs “adjacent”: e.g., `GET /rpc/v1/chat/ws` is essential for streaming UX, but it does not start inference; the report calls this out explicitly.

### What warrants a second pair of eyes
- Confirm the control-plane semantics match expectations in production (step-mode/continue endpoints are gated by config and require ownership).

### What should be done in the future
- N/A

### Code review instructions
- Read the backend report doc:
  - `geppetto/ttmp/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER--moments-llm-inference-trigger-inventory/analysis/01-backend-endpoints-that-trigger-geppetto-inference.md`

### Technical details
- Commands run:
  - `docmgr doc add --ticket MO-002-MOMENTS-LLM-INFERENCE-TRIGGER --doc-type analysis --title "Backend: Endpoints That Trigger Geppetto Inference"`
  - `docmgr doc relate --doc geppetto/ttmp/2026/01/21/.../analysis/01-backend-endpoints-that-trigger-geppetto-inference.md --file-note "...:..."`
  - `remarquee upload md --dry-run geppetto/ttmp/2026/01/21/.../analysis/01-backend-endpoints-that-trigger-geppetto-inference.md --remote-dir "/ai/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER"`
  - `remarquee upload md geppetto/ttmp/2026/01/21/.../analysis/01-backend-endpoints-that-trigger-geppetto-inference.md --remote-dir "/ai/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER" --non-interactive`

## Step 4: Frontend Audit — RTK Query Mutations + Inference Trigger Call Sites

Enumerated every RTK Query mutation across the frontend, then traced how the chat UI actually starts inference (and how it streams results). The core is a central “chat send queue” thunk that dispatches RTK Query mutations to `/rpc/v1/chat` or `/rpc/v1/chat/{profile}` and retries on `409 Conflict` (backend run in progress).

Then I cataloged every explicit and implicit (auto-start) entrypoint that hits that queue, and wrote it up as a second report document in the ticket.

### What I did
- Located Redux store setup and RTK Query API slices (`apiSlice`, `rpcSlice`) and all `injectEndpoints` extensions.
- Extracted all `builder.mutation(...)` definitions (network mutations) across the codebase.
- Identified the central inference trigger point:
  - `platform/chat/state/chatQueueSlice.ts` → `processQueue` dispatches:
    - `chatApi.endpoints.startChat.initiate` → `POST /rpc/v1/chat`
    - `chatApi.endpoints.startChatWithProfile.initiate` → `POST /rpc/v1/chat/{profile}`
- Found all call sites that enqueue chat messages or auto-start hidden messages, and recorded them in a dedicated frontend report doc.

### Why
- Backend triggers are mostly `/rpc/v1/chat*`, but the frontend might hit them from many UI entrypoints (sidebar chat, widgets, auto-start flows). Enumerating these prevents “mystery inferences” during debugging and incident response.

### What worked
- Found a single “choke point” for inference triggering: `enqueueChatMessage` → `processQueue`.
- Confirmed WS streaming uses `/rpc/v1/chat/ws` with `access_token` query param, matching backend behavior.
- Wrote a complete list of RTK Query mutations and inference trigger sites into:
  - `geppetto/ttmp/2026/01/21/.../analysis/02-frontend-redux-mutations-inference-trigger-points.md`

### What didn't work
- N/A (investigation-only).

### What I learned
- Several feature pages auto-start inference without explicit user typing via `startWithHiddenMessage(...)`:
  - Team select (“Find my team”)
  - Document finding (“Find me documents…”)
  - Summary page (“Please summarize the following documents…”)

### What was tricky to build
- Distinguishing “places that connect to streaming” (`useChatStream`) from “places that start inference” (queue → `POST /rpc/v1/chat*`). The report makes this separation explicit.

### What warrants a second pair of eyes
- Confirm that the “debug continue” mutation is intentionally unused in the current UI (it exists in `rpcSlice` but has no call sites).

### What should be done in the future
- N/A

### Code review instructions
- Start with the frontend report doc:
  - `geppetto/ttmp/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER--moments-llm-inference-trigger-inventory/analysis/02-frontend-redux-mutations-inference-trigger-points.md`
- Then review the trigger implementation:
  - `moments/web/src/platform/chat/state/chatQueueSlice.ts`
  - `moments/web/src/platform/api/chatApi.ts`
  - `moments/web/src/platform/chat/hooks/useChatStream.ts`

### Technical details
- Commands run (representative):
  - `rg -n "configureStore\\(|createSlice\\(|createApi\\(|injectEndpoints\\(" moments/web/src`
  - `rg -n "builder\\.mutation" moments/web/src`
  - `rg -n "enqueueChatMessage\\(|enqueueChatMessageAndWait\\(" moments/web/src`
  - `rg -n "startWithHiddenMessage\\(" moments/web/src`

## Step 5: Upload Frontend Report to reMarkable

Uploaded the frontend audit report (Redux mutations + inference trigger points) as a PDF to the same ticket folder on reMarkable so the backend and frontend reports can be reviewed together.

### What I did
- Ran `remarquee upload md --dry-run ...` to confirm the PDF output name and remote dir.
- Uploaded `02-frontend-redux-mutations-inference-trigger-points.pdf` to `/ai/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER/`.

### Why
- Keeps the two-ticket artifacts (backend + frontend) in a single reMarkable folder for easier review/annotation.

### What worked
- Upload succeeded with network-enabled execution.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Review the uploaded PDFs in reMarkable under `/ai/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER/`.

### Technical details
- Commands run:
  - `remarquee upload md --dry-run geppetto/ttmp/2026/01/21/.../analysis/02-frontend-redux-mutations-inference-trigger-points.md --remote-dir "/ai/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER"`
  - `remarquee upload md geppetto/ttmp/2026/01/21/.../analysis/02-frontend-redux-mutations-inference-trigger-points.md --remote-dir "/ai/2026/01/21/MO-002-MOMENTS-LLM-INFERENCE-TRIGGER" --non-interactive`
