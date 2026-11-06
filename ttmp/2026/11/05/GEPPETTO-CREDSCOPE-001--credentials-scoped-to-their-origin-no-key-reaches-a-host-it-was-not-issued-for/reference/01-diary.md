---
Title: Diary
Ticket: GEPPETTO-CREDSCOPE-001
Status: active
Topics:
    - security
    - credentials
    - providers
    - geppetto
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-05T21:50:28.062800021-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

<!-- What is the purpose of this reference document? -->

## Context

<!-- Provide background context needed to use this reference -->

## Quick Reference

<!-- Provide copy/paste-ready content, API contracts, or quick-look tables -->

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->

# Diary

## Goal

Make it structurally impossible for a provider credential to reach a host it was not issued for. Today Geppetto resolves secrets and endpoints separately, and each side has its own fallbacks. This ticket ties each credential to an origin and has the HTTP client enforce it, stacked on PR #444 (branch `task/geppetto-credscope-001`, based on `task/geppetto-decision-001` at `4ab5c800`).

## Step 1: Inventory, leak probe, and the intern guide

The two Codex findings on PR #444 were both credential pairings that a later default undid. After them, the user asked whether the bug class could be ruled out architecturally, and whether it existed beyond the decision and chat paths. I surveyed every place that attaches a credential or chooses an endpoint. Then I wrote a probe that reproduces each path in-process, and wrote the design guide around the evidence.

The probe confirmed six leak paths without any real network traffic. Two were unexpected: on redirects, Go forwards custom headers. So Pi's `models.json` headers and Claude's `x-api-key` follow a 307 to another host, while `Authorization` is stripped.

### Prompt Context

**User prompt (verbatim):** "yes, stack it. Create a new tickte and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
 then implement, and commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill). Diary format from skill:L Mandatory"

**Assistant interpretation:** Create a ticket and a branch stacked on #444, write an intern-level design guide, upload it, implement it in phases with commits, and keep an investigation-format diary.

**Inferred user intent:** Remove the bug class across all modalities, not one call site at a time, with a reviewable trail.

### What I did
- Fixed the two remaining #444 comments first (`4ab5c800`, DECISION-001 diary step 5), so #444 is safe without this branch.
- Created the branch `task/geppetto-credscope-001` and the ticket via `docmgr ticket create-ticket --ticket GEPPETTO-CREDSCOPE-001 …`, with a design doc and this diary.
- Surveyed endpoints and credentials:
  - `grep` for hard-coded `https://` constants and for `BaseURL == ""` fallbacks;
  - YAML `*-base-url` defaults;
  - `EnsureHTTPClient` call sites and `CheckRedirect` users;
  - `credentials.Request{` construction sites.
- Read `credentials/bearer.go`, `static.go`, `security/outbound_url.go`, `pi/resolve.go`, `openai/chat_stream.go`, `openai_responses/provider_settings.go`, `claude/api/completion.go`, `gemini/engine_gemini.go`, `openaicodex/codex.go`, `embeddings/settings_factory.go`, `embeddings/openai.go` and the rerank factory.
- Wrote `scripts/01-leak-probe/main.go`:
  - it replaces `http.DefaultTransport` with a recorder that answers 400 (or 307 for chosen hosts), and drives each path with marker secrets;
  - output: `sources/01-leak-probe-output.txt`.
- Summarized the real Pi catalog with a Python one-off that reads provider names, model counts and URL hosts only, never keys. All 222 models have a `baseUrl`.
- Wrote the guide: `design-doc/01-intern-guide-credential-scoping-in-geppetto-analysis-design-and-implementation.md`.

### Why
- Evidence before design: the inventory decides whether a per-call-site fix is enough. It is not. The four underlying causes are in guide §5.
- An in-process recorder makes the probe safe to run with any configuration. No marker can leave the machine.

### What worked
- Command: `GOWORK=off go run ./ttmp/2026/11/05/GEPPETTO-CREDSCOPE-001--…/scripts/01-leak-probe`. Result:
  - `FW-SECRET` was sent to `api.openai.com` and `ZAI-SECRET` to `api.anthropic.com`;
  - `TEAM-SECRET` and `CL-SECRET` were sent to `other.example` after a 307;
  - `GW-SECRET` was sent to `api.openai.com` from embeddings;
  - `OR-SECRET` was paired with the `openai` slot's URL;
  - the #444 decider fix holds (no request).

### What didn't work
- `echo ===RESP` in zsh failed with `(eval):1: ==RESP not found`: zsh expands a leading `=` as a command path. I used `echo "--RESP"` instead.
- `grep -rn --include=*.go` failed with `no matches found: --include=*.go` (zsh glob). I quoted the pattern.

### What I learned
- `http.Client` forwards every header on a redirect except `Authorization`, `WWW-Authenticate` and `Cookie` (to a different domain). Custom auth headers are not protected.
- The OpenAI embeddings provider never reads `openai-base-url`: `NewOpenAIProvider(apiKey, …)` calls `openai.NewClient(apiKey)`. This leak needs no Pi involvement.
- Codex already pins its origin (`openaicodex.Route.Resolve` accepts only `https://chatgpt.com/backend-api`). The design generalizes that precedent.

### What was tricky to build
- Driving engines without a network: the engines use `settings.EnsureHTTPClient`, which returns `http.DefaultClient` when nothing is configured, and the Pi bridge's `withHeaders(nil, …)` builds an `&http.Client{}`. Both end at `http.DefaultTransport`, so replacing that one variable intercepts every path, including the go-openai client used by embeddings.
- Ending each request without retries: a 400 with a JSON error body makes every engine stop after one attempt. A 401 would trigger the unauthorized-refresh path.

### What warrants a second pair of eyes
- The decision to *refuse* (not strip) out-of-scope requests, and to pin the whole client rather than filter headers (guide §7).
- Making OpenAI embeddings honour `openai-base-url` is a behaviour change (guide §7).

### What should be done in the future
- Cross-profile stack merges and per-layer pairing (guide §11). Deferred.

### Code review instructions
- Read the guide §4.4 inventory and §6 design. Run the probe.

### Technical details
- Probe markers: `FW-SECRET`, `ZAI-SECRET`, `GW-SECRET`, `OR-SECRET`, `CL-SECRET`, `TEAM-SECRET`.
- Redirect simulation: the recorder answers `307` with `Location: https://other.example/…` for `gateway.example`.
