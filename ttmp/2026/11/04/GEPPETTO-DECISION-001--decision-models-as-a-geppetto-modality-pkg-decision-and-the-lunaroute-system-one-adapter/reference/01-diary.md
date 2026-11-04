---
Title: Diary
Ticket: GEPPETTO-DECISION-001
Status: active
Topics:
    - geppetto
    - decisions
    - go-api
    - js-bindings
    - profiles
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: "Chronological implementation diary for GEPPETTO-DECISION-001 (decision models as a Geppetto modality)."
LastUpdated: 2026-11-04T18:29:00.885012716-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Record, step by step, how decision-model support (pkg/decision, the LunaRoute System One adapter, settings, Pi credentials and the JS binding) was researched, designed and implemented in Geppetto, with the evidence, failures and review notes for each step.

## Step 1: Research decision models, probe System One, write the intern guide

This step established what "Jev-like decision models" are and what LunaRoute actually serves, before any design. The probe confirmed that `djev` answers typed `noul`/`choice`/`score` questions on `POST /v1/systemone` in about half a second, that one request can batch many questions over a shared state, and gave us the exact error envelope. With that evidence the ticket, the intern guide and the phase plan were written.

The work happened in the WSM workspace `~/workspaces/2026-11-02/rag-workbench`, whose `go.work` now also uses `./rag-workbench`; the geppetto worktree there was switched to a new branch `task/geppetto-decision-001`.

### Prompt Context
**User prompt (verbatim):** "alright, do it, work from there, add it to the go.work, and what I want you to do then is research jev like decisions models and the API on lunaroute.com which has similar models.

Then, create a docmgr ticket in geppetto and add support for decision models (similar to how embeddings is another modality). Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Then once implemented, create a ticket in rag-workbench and integrate decision models / classifiers like jev into the JS API and into the RAG pipelines, as a way to route queries or classify things or whatever.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


You can search for use cases of jev like models for RAG online if you want, for inspiration.

Then implement. commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)

All subagents if used should write their own diary in the diary format of the skill as well.

Diary MANDATORY."
**Assistant interpretation:** Move work into the workspace; research Jev/djev and LunaRoute's decision API; design and implement a decision modality in Geppetto with a ticket, intern guide, reMarkable upload and diary; then do the same for rag-workbench integration.
**Inferred user intent:** Make fast, typed, calibrated classification a first-class Geppetto capability so RAG pipelines can route and gate on it.
**Commit (code):** none yet (research and documentation step)

### What I did
- Workspace: fast-forwarded `rag-workbench` worktree `task/rag-workbench` from `a1e444b` to `6ee9fc3`; `go work use ./rag-workbench && go work use` (go.work now `go 1.26.6`, six modules); `go build/vet/test ./...` in rag-workbench passed against local geppetto and go-go-goja.
- Read LunaRoute docs: `docs.lunaroute.com/models/`, `/api/systemone/`, `/api/models/`, `/api/embeddings-rerank/`; third-party Jev write-ups (getmaxim/bifrost, jevkit on PyPI) and an Adaptive-RAG/CRAG/Self-RAG search for RAG use cases.
- Listed Pi providers/model ids without secrets; found `lunaroute/djev` and `clef-flash` in `models-store.json` (recorded as `openai-completions`).
- Wrote `scripts/01-systemone-probe/probe.py` (reads the key from `~/.pi/agent/models.json`, never prints it) and request bodies; recorded results in `sources/01-systemone-probe-results.txt` and `sources/02-models-systemone-capability.txt`.
- Read geppetto: `pkg/rerank/{doc,rerank,validate,errors}.go`, `pkg/rerank/cohere/provider.go`, `pkg/rerank/factory/settings_factory.go`, `pkg/rerank/config/settings.go`, `pkg/steps/ai/settings/settings-inference.go` (rerank integration points), `pkg/sections/sections.go`, `pkg/js/modules/geppetto/api_reranker*.go`, `pkg/steps/ai/providers/pi/{resolve,catalog}.go`, `pkg/steps/ai/credentials/bearer.go`.
- Created ticket GEPPETTO-DECISION-001 (new vocab topic `decisions`), the design doc and this diary.

### Why
- Rerank is the newest and most carefully specified model-service primitive (explicit invariants, safety contract, factory, profile tests, JS binding), so copying its layering gives decisions the same guarantees and makes the code familiar.
- The probe was needed because the docs omit exact JSON and because Pi lists djev as a chat model; designing from evidence avoided guessing field names.

### What worked
- `GET /v1/models`: two entries with `"capabilities": {"systemone": true, "vision": true}` — `djev` (32768 context) and `clef-flash` (8192).
- `POST /v1/systemone` with djev, three questions: `200` in 428 ms, `route` → `semantic` (p 0.979, confidence 0.907), `needs_retrieval` noul 0.994, `complexity` score 1.9994; usage 228 in / 16 out.
- Both `LUNAROUTE-API-KEY` and `Authorization: Bearer` headers returned `200`.
- Twelve passage-relevance noul questions in one request: `200` in 639 ms, 659 input tokens; six relevant passages 0.99–1.0, six irrelevant 0.0–0.02.
- Error envelopes observed exactly: unknown field → `400 systemone_invalid_request "unknown parameter \"options\""`; chat id `glm-5.3` → `400 "the requested model cannot serve System One"`; `nope` → `404 model_not_found`; choice without criteria → `400 "choice question \"a\" is missing \"criteria\""`; empty questions → `400 "\"questions\" must contain at least one question"`.

### What didn't work
- `clef-flash` with the same request: `502 {"error":{"code":"systemone_backend_error","message":"the System One backend could not be reached","request_id":"01a1a8e2-9ae4-7a62-a6a6-874509972682"}}`. Treated as unavailable; djev is the target.
- First `ls` of the workspace appeared to lack `slop-computer`; a later listing showed it present (it had been added by WSM at 18:19). No action needed.
- `grep -rn ... --include=*.go` failed in zsh with `no matches found: --include=*.go` (glob expansion); quoting the pattern (`--include='*.go'`) fixed it.

### What I learned
- The documented `latency_ms` response field is absent in practice; the adapter must not require it.
- `X-Request-Id` is returned as a header on every response, including errors.
- Pi's catalog cannot tell decision models from chat models, so the Pi entry point must not filter on `Supported`/API.
- Score probabilities and legend are keyed by level-index strings; choice probabilities by option name.

### What was tricky to build
- Nothing built yet. The design-level trap identified: Go maps lose option order and `encoding/json` sorts keys, while decision models are order-sensitive. The design therefore models options as a slice and hand-encodes the `criteria` object (guide §4.3).

### What warrants a second pair of eyes
- The decision to surface provider `message` text for 400/404 only (capped at 200 bytes) is a deliberate narrowing of rerank's "no provider bodies in errors" rule (guide §4.4).
- Tolerating unknown top-level response fields departs from the rerank adapters' fully strict decoding (guide §4.5).

### What should be done in the future
- Model discovery from `GET /v1/models` capabilities (non-goal here).
- An OpenRouter/TypeSafe adapter for Jev proper, if credentials become available.

### Code review instructions
- Read the guide `design-doc/01-intern-guide-decision-models-in-geppetto-analysis-design-and-implementation.md` Parts 2 and 4.
- Re-run the probe: `cd scripts/01-systemone-probe && python3 -I probe.py models && python3 -I probe.py post q1.json`.

### Technical details
- Endpoint `POST https://gw.lunaroute.com/v1/systemone`; request `{model, state, images?, questions: {id: {type, instructions, criteria}}}`; response `{model, answers: {id: {...}}, usage: {input_tokens, output_tokens}}`.
- Route body cap 8 MiB; images: data URLs only, ≤ 6 attachments, ≤ 5 MiB each; choice 1–255 options; score ≤ 10 levels.
