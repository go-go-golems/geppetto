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
RelatedFiles:
    - Path: repo://pkg/decision/decision.go
      Note: Core types
    - Path: repo://pkg/decision/factory/settings_factory.go
      Note: Factory
    - Path: repo://pkg/decision/systemone/provider.go
      Note: Adapter
    - Path: repo://pkg/js/modules/geppetto/api_decider.go
      Note: JS binding
ExternalSources: []
Summary: Chronological implementation diary for GEPPETTO-DECISION-001 (decision models as a Geppetto modality).
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

## Step 2: Phase 1 and 2 — pkg/decision core and the System One adapter

This step built the transport-neutral core (`pkg/decision`) and the LunaRoute adapter (`pkg/decision/systemone`), and ran the adapter against the live gateway. Both packages pass their unit tests and the repository's `lintmax` pre-commit hook; the live test returned answers from djev in 311 ms.

### Prompt Context
**User prompt (verbatim):** see Step 1
**Assistant interpretation:** Implement guide phases 1 and 2.
**Inferred user intent:** A working, safe decision primitive before wiring settings and callers.
**Commit (code):** f637be88 — "feat(decision): pkg/decision core — question model types, validation, errors, Gate (GEPPETTO-DECISION-001 P1)"; 33f11d69 — "feat(decision): LunaRoute System One adapter (pkg/decision/systemone) with fake-server and opt-in live tests (GEPPETTO-DECISION-001 P2)"

### What I did
- `pkg/decision/{doc,decision,errors,gate,validate,answer_json}.go`: types (`Request`, `Question`, `Option`, `Answer`, `Response`, `Usage`, `Model`, `Provider`), `ValidateRequest`, `ValidateQuestion`, `ValidateResponse`, sentinels plus `ProviderError`/`NewProviderError`/`KindForStatus`, `Gate`.
- `pkg/decision/systemone/{protocol,provider}.go`: `New(Options)`, `Decide`, ordered `encodeOptions`, pointer-based wire answers (`mapAnswers`, `levelProbabilities`), `providerError` with `safeCode` and `retryAfter`, bounded reads, redirect rejection.
- Tests: `decision_test.go` (22 request-validation cases, 10 response cases, error mapping, Gate, JSON), `systemone/provider_test.go` (happy path, option order, 10 error statuses, 10 invalid responses, limits, redirects, cancellation, constructor validation), `live_test.go` (opt-in).
- Mutation check: replaced `encodeOptions` with a `map[string]string` marshal; `TestDecideKeepsOptionOrder` failed with `criteria":{"a":"say \"hi\"","m":"","z":""}`; restored.
- Live: `GEPPETTO_LIVE_SYSTEMONE=1 LUNAROUTE_API_KEY=… go test ./pkg/decision/systemone -run TestLive -v -count=1` (key read from Pi's models.json inline, not printed).

### Why
- Pointer fields in `wireAnswer` make "the provider omitted `noul`" an invalid response instead of a silent probability of 0.
- Unknown top-level response fields are tolerated (`latency_ms`, `future_field` in the fixture) while answers are validated strictly (guide §4.5).
- Provider error codes pass through `safeCode` (`[a-z0-9_]{1,64}`), so a server that put a token into `code` cannot leak it; the test plants `Bearer sk-12345` there.

### What worked
- `go test ./pkg/decision/...` → `ok` for both packages; `.bin/golangci-lint run ./pkg/decision/...` → `0 issues`.
- Live: `model=djev request_id=01a1a8f1-4e15-7542-a508-d68a61c3d54b duration=311ms usage={InputTokens:209 OutputTokens:16}`; `route` → `semantic` (0.929, confidence 0.755), `complexity` → 1.947.

### What didn't work
- First commit attempt of P1 failed the pre-commit `make lintmax`: `pkg/decision/decision_test.go:59:1: File is not properly formatted (gofmt)`, `errors.go:67:2: QF1002: could use tagged switch on status`, `validate.go:57:6: QF1001: could apply De Morgan's law`. Fixed with gofmt, a tagged switch and an `idRune` helper.
- `answer_json` test first draft did not compile: `decision_test.go:179:27: invalid map key type Answer` (a leftover map literal keyed by a struct with a map field); removed.
- `TestDecide` failed on cost: `usage/cost &{InputTokens:228 OutputTokens:16} 0x350d1fd9e420` because `228*0.042/1e6` is an exact constant expression in the test and a rounded float at runtime; compared with a 1e-15 tolerance.
- P2 lint: `QF1001` again in `safeCode` and `ST1013` (`418` → `http.StatusTeapot`); fixed.

### What I learned
- With `omitempty`, a noul answer of probability 0 serialised as `{"type":"noul"}`. Added `Answer.MarshalJSON` that writes exactly the fields of the answer's type, zeros included.
- The live `needs_retrieval` noul, asked without criteria and without a corpus description in the state, returned 0.084; the Step 1 probe with criteria and `corpus` in the state returned 0.994 for the same query. Decisions depend strongly on what the state and criteria say; RAG callers should describe the corpus in the state.

### What was tricky to build
- Choice criteria order. Go maps cannot carry it, so `encodeOptions` writes the object bytes itself, marshalling each key and value separately for escaping. The mutation check proves the test detects a regression.
- Score probabilities are keyed `"0".."n-1"`; `levelProbabilities` rejects padded keys like `"02"` (`strconv.Itoa(i) != k`) so a key cannot alias another level.

### What warrants a second pair of eyes
- `ProviderError.Message` is kept for 400/404 (validation messages). A provider that echoed request content into a 400 message would expose it to callers' logs.
- On a non-2xx response the body is read under the same limit and parsed leniently; an oversized error body yields a truncated parse, not `ErrResponseTooLarge`.

### What should be done in the future
- N/A for these phases.

### Code review instructions
- Start at `pkg/decision/decision.go` and `validate.go`, then `systemone/provider.go` (`Decide`, `encodeRequest`, `mapAnswers`, `providerError`).
- `go test ./pkg/decision/... -count=1`; live test as above.

### Technical details
- Defaults: base `https://gw.lunaroute.com/v1`, endpoint `/systemone`, request cap 8 MiB, response cap 1 MiB, client timeout 60 s when no client is injected.
- Status → sentinel: 400/415 invalid request; 401/402/403 not granted; 404 model not found; 413 too large; 429 rate limited; others unavailable.

## Step 3: Phases 3–6 — settings and profiles, Pi, JavaScript, docs and smoke example

This step connected the primitive to everything that configures or calls it. Decisions now come from `InferenceSettings` and engine profiles through a factory, from Pi's agent directory through `AgentDir.Decider`, and from JavaScript through `gp.decider`. A help page and a live smoke example document the result. The full geppetto suite passes (`go test ./...`: 55 packages `ok`, exit 0), and `make lintmax` reports 0 issues.

### Prompt Context
**User prompt (verbatim):** see Step 1
**Assistant interpretation:** Implement guide phases 3–6.
**Inferred user intent:** Decisions should be configurable and callable like rerank, and usable from rag-workbench through Pi without extra setup.
**Commit (code):** dff707c9 (P3 settings/factory/profiles), a37bbe30 (P4 Pi Decider), bf13aeea (P5 gp.decider + d.ts), 4229f55f (P6 help page + smoke example)

### What I did
- `pkg/decision/config/{settings.go,flags/decision.yaml}`: `DecisionConfig` (omitempty YAML tags), `DecisionSlug`, flags `decision-type`, `decision-engine`, byte limits, `systemone-api-key`, `systemone-base-url`.
- `pkg/decision/factory/settings_factory.go`: `NewSettingsFactory`, `NewSettingsFactoryFromInferenceSettings`, `NewProvider`, `ValidateInferenceSettingsForDecision`; names `APIKeyName`, `BaseURLName`, `PolicyName` (`decision`).
- `pkg/steps/ai/settings/settings-inference.go`: `Decision` field, nil default, metadata, clone, guarded decode, `apiSlugs`. `pkg/sections/sections.go` registers the section with `NewDecisionConfig()` defaults. `pkg/cli/bootstrap/inference_debug.go` maps `decision.*`.
- Tests: `factory/settings_factory_test.go` (rejections, validation, a YAML profile building a working provider with cost, clone/metadata, outbound policy); `engineprofiles/stack_merge_decision_test.go` (stacking, partial overlay); `sections/sections_test.go` (section flags, flag→settings decode, host without the section).
- `pkg/steps/ai/providers/pi/decision.go`: `DecisionOptions`, `(*AgentDir).Decider`; `decision_test.go`.
- `pkg/js/modules/geppetto/api_decider.go`: `gp.decider`, `decide`, `decideAsync`, `model`; strict decoding through `goja.Object.Keys()`; `spec/geppetto.d.ts.tmpl` and the regenerated `pkg/doc/types/geppetto.d.ts`; `api_decider_test.go`.
- `pkg/doc/topics/16-decisions.md`; `cmd/examples/decision-pi-smoke/main.go` (Glazed command `run`).

### Why
- Decoding the decision section only when it is registered (`parsedValues.Get`) keeps hosts that assemble their own section lists working. Glazed's `DecodeSectionInto` returns `section decision not found` for a missing section, and rerank has no such guard.
- `systemone-api-key` and `systemone-base-url` are flags of the decision section, so the existing `*-api-key`/`*-base-url` wildcard decode puts them into `APISettings` with no new code.

### What worked
- Live through Pi: `go run ./cmd/examples/decision-pi-smoke run --query "how does the rebalance law preserve product placements?"` → route `semantic` 0.924, needs_retrieval 0.909, complexity 1.885, 231 input tokens, 403 ms.
- Passages: rebalance passage 0.999, cafeteria passage 0.0013, 635 ms.

### What didn't work
- `TestDecisionFlagsDecodeIntoInferenceSettings` first failed with `decision: &config.DecisionConfig{Type:"systemone", Engine:"djev", MaxRequestBytes:0, MaxResponseBytes:0}`. `sections.go` initialised defaults from an empty struct, as rerank does, which zeroed the YAML defaults. Fixed by seeding from `NewDecisionConfig()`.
- `TestDeciderErrors` failed with `header x-team of lunaroute/djev: environment variable TEAM is not set`. The fixture's models.json header needs `TEAM`; set it in the test env.
- `TestGeneratedDTSMatchesRuntimeExportSurface` failed with `extra: [decider]`. Added declarations to `spec/geppetto.d.ts.tmpl` and ran `go generate` in `pkg/js/modules/geppetto`.
- `TestDeciderDecide` failed on JSON key order (`"needs":{"probability":0,"type":"noul"}` vs expected order). Go maps converted to JS objects have random key order. The test now extracts fields in JS.
- The first P6 commit was blocked by glazed-lint: `cmd/examples/decision-pi-smoke/main.go:38:25: use Glazed config/env middleware or an explicit command field instead of os.Getenv in CLI code` and five `define CLI flags with cmds.WithFlags(fields.New(...)) instead of raw Cobra/pflag/flag APIs`. Rewrote it as a Glazed `GlazeCommand` (rows per answer) through `examplecmd`.

### What I learned
- `pi.DefaultDir("")` already resolves `~/.pi/agent`, so the example needs no environment lookup.
- Pi's `withHeaders(nil, …)` returns an `http.Client` without a timeout. The systemone adapter's 60 s default applies only when no client is injected, so Pi-built deciders rely on the caller's context deadline.

### What was tricky to build
- Preserving choice-option order across JavaScript. `Export()` of a JS object yields `map[string]any`, so the decoder walks objects with `Keys()` and reads arrays (`ClassName() == "Array"`) as names or `{name, description}`. The test checks the wire body contains `"criteria":{"z":"last letter","semantic":"concepts","keyword":"identifiers"}`.

### What warrants a second pair of eyes
- The `hasDecision` guard in `UpdateFromParsedValues` changes behaviour relative to rerank: a missing section is "not configured" rather than an error.
- `Decider` does not check that the Pi model is a decision model (Pi cannot tell). A chat model fails at the server with `ErrInvalidRequest`.

### What should be done in the future
- Capability discovery from `GET /v1/models` (`capabilities.systemone`) so hosts can list decision models.
- A timeout on Pi-built HTTP clients, or documentation that callers must set deadlines.

### Code review instructions
- `git log --oneline f637be88^..4229f55f`; start with `pkg/decision/factory/settings_factory.go`, the `settings-inference.go` diff, `pi/decision.go`, `js/modules/geppetto/api_decider.go`.
- `go test ./... && make lintmax`; live: `go run ./cmd/examples/decision-pi-smoke run --format json`.

### Technical details
- JS response shape: `{provider, model, answers: {id: {type, probability | choice, confidence, probabilities | score, confidence, levelProbabilities}}, usage: {inputTokens, outputTokens}, cost?, requestId?, durationMs?}`.

## Step 4: Address the Codex review on PR #444

PR #444 (`wesen:task/geppetto-decision-001` → `main`) got three automated review findings from `chatgpt-codex-connector`. All three were real and are fixed in one commit, with regression tests. While running the tests repeatedly I also found that `TestDeciderDecide` was still flaky, and fixed that too.

### Prompt Context

**User prompt (verbatim):** "Tackle this https://github.com/go-go-golems/geppetto/pull/444 (code review comments)"

**Assistant interpretation:** Fix each review finding on PR #444, push the fixes, and reply to each comment.

### What I did
- **P1, `pkg/steps/ai/providers/pi/decision.go`:**
  - `Decider` used to fall back to `systemone.DefaultBaseURL` for any catalog entry without a `baseUrl`. That would send, for example, an OpenAI credential to `gw.lunaroute.com`.
  - Now the fallback applies only when `m.Provider == LunaRouteProvider` (`"lunaroute"`). Any other provider without a base URL is rejected with `decision.ErrInvalidRequest` before its credential is even looked up.
  - Test: `amazon-bedrock/claude` in the fixture has no `baseUrl`.
- **P2, `pkg/decision/validate.go`:**
  - A choice answer must now carry a probability for every declared option. Before, a nil or partial `probabilities` map passed validation, and a missing option silently read as 0.
  - Tests: "partial option probabilities" and "no option probabilities".
  - djev always returns the full distribution, including entries such as `9.5e-06` (see `sources/01-systemone-probe-results.txt`), so live behaviour does not change.
  - The fixture in `TestDecideKeepsOptionOrder` returned only `{"z":1}` and had to be completed.
- **P2, `pkg/js/modules/geppetto/api_decider.go`:**
  - `images` must be an array; `undefined`/`null` is still allowed. `levels` must be an array too.
  - Before, `exportSlice` returned nil for a string, so the images were silently dropped.
  - Added an `isArray` helper. Tests cover both cases.
- **Docs:** `16-decisions.md` now states the base-URL rule and the full-distribution requirement.

### Why
- P1 is a credential leak to a third-party host.
- The other two hand callers a "validated" answer or request that silently lacks data they relied on.

### What worked
- `GOWORK=off go test ./... -count=1` passes.
- `make lint` passes (golangci-lint 0 issues, geppetto-lint, glazed-lint).
- `go test -run TestDecider -count=50 ./pkg/js/modules/geppetto/` passes.

### What didn't work
- `TestDecideKeepsOptionOrder` failed after the P2 fix: `decision answer "q": 1 option probabilities for 3 options: invalid decision response`. Its fixture answered with an incomplete map, so I completed it with `{"z":1,"a":0,"m":0}`.
- `TestDeciderDecide` failed in about 5 of 30 runs: `... "usage":{"outputTokens":3,"inputTokens":12} ... does not contain "usage":{"inputTokens":12,"outputTokens":3}` (and the same for `model`).
  - Step 3's fix had covered only the answers. `model` and `usage` are also Go maps, so their key order is random.
  - The test now reads those fields individually. This flake was there before the PR's CI run, which passed by chance.

### What I learned
- An unvalidated fallback default combined with a per-provider credential lookup turns a convenience into a credential leak. A default endpoint has to be tied to the provider it belongs to.

### What was tricky to build
- Ordering in `Decider`: the base-URL check must come before `a.credential(...)`. That way a misconfigured provider fails with the base-URL error, not a missing-credential error, and no credential is resolved for a request that is never sent.

### What warrants a second pair of eyes
- `LunaRouteProvider = "lunaroute"` is the provider name in the user's Pi config. A user who registers LunaRoute under another name must set `baseUrl` explicitly.

### What should be done in the future
- N/A

### Code review instructions
- `git show HEAD` on `task/geppetto-decision-001`. Start with `pi/decision.go`, then the `validateAnswer` choice branch.

### Technical details
- Review comment ids: 4484703632 (P1, Pi URL fallback), 4484703647 (P2, choice probabilities), 4484703659 (P2, images).

## Step 5: Second Codex review — URL normalization and null probabilities

After step 4, the user asked Codex to re-review PR #444. It reported two more findings against `194f1287`. Both are fixed here, kept small, because the user decided the architectural fix (credentials scoped to their origin) goes in a separate stacked PR, GEPPETTO-CREDSCOPE-001. This step only makes #444 safe on its own.

### Prompt Context

**User prompt (verbatim):** "yes, stack it. Create a new tickte and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
 then implement, and commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill). Diary format from skill:L Mandatory"

**Assistant interpretation:** Fix the two open comments on #444 minimally. Then do the credential-scoping work as a new ticket and branch stacked on #444.

**Inferred user intent:** Keep #444 reviewable and safe, and remove the bug class in a dedicated change.

### What I did
- **P1, comment 4485078058:**
  - Added `systemone.NormalizeBaseURL`, the exact normalization `New` applies.
  - `pi.Decider` now checks the normalized value, so `" "`, `"/"` and `" // "` count as missing for non-LunaRoute providers.
  - Test: a `fireworks/djev` model with each of those URLs must fail with "no baseUrl".
  - I confirmed the test fails when the old raw check is restored.
- **P2, comment 4485078074:**
  - `wireAnswer.Probabilities` is now `map[string]*float64`.
  - `optionProbabilities` and `levelProbabilities` reject JSON `null` with `ErrInvalidResponse`.
  - Tests: "null option probability" and "null level probability" in `TestDecideInvalidResponses`.

### Why
- P1: the step-4 guard checked the raw string while `New` used the normalized one, so the value that was checked differed from the value that was used. Sharing one normalization function closes that gap for this call site. The general fix is GEPPETTO-CREDSCOPE-001.
- P2: `encoding/json` decodes `null` into a `float64` as 0 without an error, so the completeness check from step 4 could not see it.

### What worked
- `GOWORK=off go test ./pkg/decision/... ./pkg/steps/ai/providers/pi/ -count=1` passes.

### What didn't work
- No failures observed.

### What I learned
- Pi's `buildCatalog` keeps non-empty but unusable `baseUrl` values such as `" "` as they are. Each consumer normalizes differently.

### What was tricky to build
- Nothing beyond keeping the check and the use on the same function.

### What warrants a second pair of eyes
- Whether to remove `systemone.New`'s default URL entirely once CREDSCOPE lands (planned there).

### What should be done in the future
- GEPPETTO-CREDSCOPE-001: tie each credential to its origin across all engines (inventory in that ticket).

### Code review instructions
- `git show HEAD`: `systemone/provider.go` (`NormalizeBaseURL`, `optionProbabilities`), `pi/decision.go`.

### Technical details
- Review comment ids: 4485078058, 4485078074.
