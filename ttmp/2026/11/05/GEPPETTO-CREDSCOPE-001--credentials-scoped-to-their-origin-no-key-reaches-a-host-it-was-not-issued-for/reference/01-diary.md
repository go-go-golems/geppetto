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

## Step 2: Phase 1 — Origin, PinnedTransport, PinClient and credentials.Scoped

This step adds the two primitives the design rests on. `pkg/security/origin.go` defines `Origin`, `ParseOrigin`/`OriginOf`, `Allows`, `PinnedTransport` and `PinClient`, plus the sentinel `ErrOriginNotAllowed`. `pkg/steps/ai/credentials/scoped.go` adds `Scoped`, which wraps a bearer token source so it refuses to produce a token for a foreign `Request.BaseURL`. Nothing uses them yet. Phases 2 and 3 wire them in.

### Prompt Context

**User prompt (verbatim):** see Step 1.

**Assistant interpretation:** Implement guide Phase 1.

**Inferred user intent:** Build the enforcement mechanism once, before touching any engine.

### What I did
- `Origin{Scheme, Host, Port}`:
  - scheme and host are lower-case, and the port is always explicit;
  - `String()` uses `net.JoinHostPort`, so IPv6 hosts are bracketed.
- `PinnedTransport.RoundTrip`:
  - refuses before calling `Next` and closes the request body (RoundTripper contract);
  - the error names only scheme and host (`redactedTarget`), plus the allowed list.
- `PinClient(base, allowed…)` copies `base` (nil means a new client), so pinning `http.DefaultClient` never mutates the global.
- `credentials.Scoped` returns one of two wrapper types, so `UnauthorizedBearerTokenSource` is preserved only when the wrapped source has it. An empty or unparsable `BaseURL` is refused.
- Tests:
  - `pkg/security/origin_test.go`: normalization table; rejects `""`, `" "`, `"/"`, `"//"`, relative, `ftp`, `file`; refusal before sending (the counting `Next` sees only the allowed request); no query in the error; body closed;
  - an `httptest` cross-origin 307 with `x-api-key`: the target is never contacted, while a same-origin redirect still works;
  - the base client is not modified;
  - `scoped_test.go`.

### Why
- Pinning lives in the transport because `http.Client` calls `Transport.RoundTrip` once per redirect hop. A check before `Do` cannot see redirects.

### What worked
- `GOWORK=off go test ./pkg/security/ ./pkg/steps/ai/credentials/ -count=1`: both packages pass.

### What didn't work
- No failures observed.

### What I learned
- `url.Parse("https://")` succeeds with an empty host, so the host check in `OriginOf` is required, not redundant.

### What was tricky to build
- A nil `Next` must mean `http.DefaultTransport` *read at request time*, not captured at construction. Otherwise a pinned client built before a test (or the probe) swaps `http.DefaultTransport` would bypass the swap.

### What warrants a second pair of eyes
- Deviation from the guide's API sketch: `ParseOrigin` *ignores* user info instead of rejecting it, because user info is not part of the origin and rejecting it could break proxy-style URLs. Recorded for the guide's status section.

### What should be done in the future
- N/A

### Code review instructions
- `pkg/security/origin.go`, `pkg/steps/ai/credentials/scoped.go` and their tests.

### Technical details
- Error shape: `request to https://other.example refused (allowed: https://gw.example:443): origin not allowed for this credential`.

## Step 3: Phase 2 — the Pi bridge resolves endpoints by provider, pins the client, scopes the source

The Pi bridge now computes a model's endpoint and its credential's scope from the same place, the provider. `endpoint(m)` (new `pkg/steps/ai/providers/pi/endpoint.go`) returns the model's normalized `baseUrl`, else the provider's own default from `providerDefaultBaseURLs`, else `*NoEndpointError`. `Resolve` and `Decider` use the result verbatim. They pin the engine's HTTP client to its origin and wrap the token source in `credentials.Scoped`. The protocol-keyed defaults (`first(m.BaseURL, "https://api.openai.com/v1")` and the Anthropic and Codex equivalents) are gone. The LunaRoute special case from #444 became one row of the table.

This closes inventory rows 1, 2 and 6 for the Pi path, and the redirect part of row 7 for Pi-built Claude and Gemini engines: their clients are pinned, so `x-api-key` and `?key=` cannot follow a redirect.

### Prompt Context

**User prompt (verbatim):** see Step 1.

**Assistant interpretation:** Implement guide Phase 2.

**Inferred user intent:** Make Pi-built engines and deciders unable to send a key anywhere but the provider's own origin.

### What I did
- `endpoint.go`:
  - `providerDefaultBaseURLs` covers `openai`, `anthropic`, `google`, `openai-codex` and `lunaroute` (`LunaRouteProvider`);
  - `NoEndpointError{Model, Provider, Reason}`; its message contains "no baseUrl";
  - `normalizeBaseURL` trims space and trailing slashes;
  - a URL that is not absolute http(s) is a `NoEndpointError` too.
- `resolve.go`:
  - `endpoint(m)` runs before `headers()` and before `credential()`;
  - `s.Client.HTTPClient = security.PinClient(withHeaders(…), origin)`;
  - `src = credentials.Scoped(src, origin)`;
  - every `BaseUrls[...]` assignment uses `baseURL`.
- `decision.go`: `endpoint(m)` (errors also wrap `decision.ErrInvalidRequest`), `credentials.Scoped`, pinned client.
- `credscope_test.go`, with a recording `http.RoundTripper` passed through `WithHTTPClient`:
  - foreign providers without a usable `baseUrl` (`""`, `" "`, `"/"`, `"api.z.ai/v1"`) fail with `NoEndpointError`, and **no host is contacted**;
  - `openai` keeps `api.openai.com`;
  - a 307 from `gateway.example` to `other.example` is refused with `security.ErrOriginNotAllowed`, and `other.example` is never contacted;
  - the decider uses the LunaRoute default, refuses `fireworks`, and stays on `gateway.example`.

### Why
- The default URL now belongs to the key's provider by construction (guide §7, "the Pi bridge resolves endpoints by provider").
- Running `endpoint()` first means no credential or header value is resolved for a request that will never be sent.

### What worked
- Pre-change check: I restored `resolve.go` and `decision.go` from `HEAD` and ran `go test ./pkg/steps/ai/providers/pi/ -run 'TestResolve|TestDeciderUsesProviderEndpoint'`. Three tests FAIL there (`TestResolveRefusesProtocolDefaultForForeignProviders`, `TestResolvePinsClientAcrossRedirects`, `TestDeciderUsesProviderEndpoint`), and all pass with the change.
- `GOWORK=off go test ./... -count=1`: no failures. `make lint`: 0 issues.

### What didn't work
- No failures observed beyond the intended pre-change failures.

### What I learned
- The existing `TestDeciderErrors` cases from #444 pass unchanged. The table entry for `lunaroute` and the `NoEndpointError` message ("no baseUrl") preserve their contract.

### What was tricky to build
- Codex: its route already accepts only `https://chatgpt.com/backend-api`. Pinning its client to the same origin is consistent, and the OAuth refresh uses its own client (`oauth.NewClient(a.codexConfig())`), not `s.Client.HTTPClient`, so refreshes to `auth.openai.com` are not affected.
- Gemini: before, an empty `baseUrl` left `gemini-base-url` unset (SDK default). Now it is set to the `google` table entry, or to the model's URL. Behaviour only changes for non-`google` Gemini-protocol providers without `baseUrl`, which now fail.

### What warrants a second pair of eyes
- `catalog.go:147` (`m.BaseURL = first(pc.BaseURL, m.BaseURL)`): a `models.json` provider `baseUrl` of `" "` wins over a valid stored URL, and then normalizes to missing. That fails closed (an error, not a leak), but a user could find it surprising.

### What should be done in the future
- N/A for this phase.

### Code review instructions
- `pkg/steps/ai/providers/pi/endpoint.go`, then the `resolve.go` diff around the `endpoint(m)` call, then `decision.go`.
- `GOWORK=off go test ./pkg/steps/ai/providers/pi/ -count=1`.

### Technical details
- Error: `pi: fireworks/m has no baseUrl (and provider fireworks has no default endpoint); set baseUrl for provider fireworks in models.json`.

## Step 4: Phase 3 — pinned clients on the settings path; Responses slot pairing; embeddings honour openai-base-url

Every engine and adapter that attaches a settings slot's credential now sends through `settings.EnsurePinnedHTTPClient(cs, baseURL…)`. That is `EnsureHTTPClient` pinned to the origin of the URL actually used, so a redirect or any other origin is refused. Two resolution bugs are fixed on the way. The Responses engine now resolves the key and the URL as a pair from one slot (inventory row 4). OpenAI embeddings now read `openai-base-url` instead of always calling OpenAI (row 3).

### Prompt Context

**User prompt (verbatim):** see Step 1.

**Assistant interpretation:** Implement guide Phase 3.

**Inferred user intent:** Close the profile-configured paths, which need no Pi involvement.

### What I did
- `settings.EnsurePinnedHTTPClient` (`pkg/steps/ai/settings/http_client.go`): an error when given no URL or an unparsable one; the host client is copied, never modified.
- Pinned clients wired into:
  - OpenAI chat (`chat_stream.go`, to the slot base URL);
  - the Responses engine (to the transport's resolved URL) and Responses token count;
  - Claude: the engine and token count, which before used `&http.Client{}` unless one was injected;
  - Gemini modern engine (to the base URL, or `geminiDefaultBaseURL` when unset);
  - embeddings: OpenAI and Cohere (to the base URL or `defaultCohereBaseURL`);
  - rerank: llama.cpp and Cohere (to the base URL or `cohere.DefaultBaseURL`);
  - the decision factory (to `NormalizeBaseURL(baseURL)` or `systemone.DefaultBaseURL`).
  - Pinning errors in rerank and decision also wrap their `ErrInvalidRequest`.
- Responses: `resolveResponsesSlot(api, sourced)`:
  - the slot is the first with a key; with a token source, or when no slot has a key (Codex), the first with a URL;
  - `openai-responses` may use `openai-base-url`; only the OpenAI-named slots default to `api.openai.com`;
  - an `open-responses` key without its own URL is an error;
  - `responsesAPIKey`/`responsesBaseURL` remain as wrappers over it.
- Embeddings OpenAI: `NewOpenAIProvider(…, opts ...OpenAIOption)` with `WithOpenAIBaseURL` and `WithOpenAIHTTPClient` (variadic, so existing callers compile). The factory resolves `options.baseURL`, then `openai-base-url`, then `https://api.openai.com/v1`, validates it with `ValidateOutboundURL(…, "embeddings")` as Cohere does, and pins.
- Tests:
  - new `TestResponsesSlotPairsKeyWithItsOwnURL`: the leak case is an error; five pairing cases, including Codex and a token source;
  - `TestResponsesAPIKeyPrefersOpenResponsesKey` now gives the open-responses slot a URL;
  - the embeddings client-injection test now asserts a pinned copy (same `Timeout`, `PinnedTransport.Next` is the host transport, origin `https://proxy.example:443`, host client unmodified).

### Why
- Pinning to the URL actually used covers redirect leaks for all profile-configured keys (row 7: Claude `x-api-key`, Gemini `x-goog-api-key`).
- For profile keys, slot pairing is the only signal of where a key belongs, so pairing has to be fixed where it was broken (Responses, embeddings).

### What worked
- `GOWORK=off go test ./pkg/rerank/... ./pkg/decision/... ./pkg/embeddings/... ./pkg/steps/... ./pkg/inference/... ./pkg/js/... -count=1`: no failures after the test updates.
- `make lint`: 0 issues.

### What didn't work
- First Responses attempt: choosing the slot only by key broke Codex. Codex supplies its credential through a transport middleware with `DisableDefaultBearer`, so no slot has a key and the openai slot (api.openai.com) was picked instead of `open-responses-base-url`. Fixed: with no key in any slot, the slot is chosen by URL.
- `TestResponsesAPIKeyPrefersOpenResponsesKey` failed with `provider_settings_test.go:265: expected open-responses key, got ""`. That is the intended change (an open-responses key with no open-responses URL), so I updated the test's fixture.
- `TestNewSettingsFactoryFromInferenceSettingsSkipsBlankLocalCohereValuesAndInjectsClient` failed with `settings_factory_test.go:190: provider did not retain the HTTP client injected through InferenceSettings.Client`. The test compared client identity; it now checks the pinned copy's properties.
- `go build` failed with `pkg/embeddings/settings_factory.go:173:13: undefined: security`: a missing import, added.

### What I learned
- `open-responses` is the canonical Responses slot name and `openai-responses` a legacy alias (`types.go:7–8`, `factory.go:230–245`). Only `openai`, `claude` and `gemini` have YAML URL defaults, so `openai-base-url` is always populated in CLI apps. That is what made the old cross-slot fallback send other vendors' keys to OpenAI.

### What was tricky to build
- Pin target versus adapter default: when a profile names no URL, the adapter (Cohere, System One, the Gemini SDK) applies its own default. The pin must use the *same* default, or every request would be refused. Each call site therefore pins to `configured-or-adapter-default`, using the adapter's exported constant where one exists.

### What warrants a second pair of eyes
- Behaviour changes:
  1. OpenAI embeddings honour `openai-base-url`;
  2. OpenAI embeddings now enforce the outbound URL policy (`allow_http.embeddings`, `allow_local_networks.embeddings` for local servers);
  3. an `open-responses` key needs `open-responses-base-url`.
- Gemini SDK: the cache client may be built without our `HTTPClient` (guide §11). This was not verified in this step.

### What should be done in the future
- Phase 4 matrix: end-to-end rows for chat, Claude redirect, embeddings and Responses.

### Code review instructions
- `settings/http_client.go` (`EnsurePinnedHTTPClient`), `openai_responses/provider_settings.go` (`resolveResponsesSlot`), `embeddings/settings_factory.go` (the openai case), then the one-line call-site changes.

### Technical details
- The Responses slot error message: `open-responses-api-key is set but open-responses-base-url is not; the key is not sent to another slot's URL or to OpenAI's default`.

## Step 5: Phase 4 — the cross-path regression matrix

`pkg/steps/ai/credscope` is a test-only package (just `doc.go`, no production code) holding `TestNoCredentialLeavesItsOrigin`, a matrix of 38 rows. Each row drives a real path (an engine, adapter, factory, or the Pi bridge) with the marker `MATRIX-SECRET` as its credential, through a recording transport. The invariant: every contacted host is in the row's allowed set, a refused configuration contacts nothing, and a configured row really does reach its own origin, so a row cannot pass by never exercising the credential. On the #444 head the matrix fails 14 rows; on this branch all 38 pass.

### Prompt Context

**User prompt (verbatim):** see Step 1.

**Assistant interpretation:** Implement guide Phase 4.

**Inferred user intent:** Make the bug class regress loudly. New engines must add a row.

### What I did
- The rows:
  - 12 paths, each with a plain row and a "307 to other.example" row: OpenAI chat; Claude; Gemini; Responses (`openai` slot and `open-responses` slot); embeddings (OpenAI, Cohere); rerank Cohere; decision System One; Pi `openai-completions` with a secret `x-team` header; Pi `anthropic-messages`; Pi decider;
  - refusal rows for `" "` and `"/"` base URLs (OpenAI chat, Claude, Pi engine, Pi decider), Pi foreign providers without a `baseUrl` (both protocols), and an `open-responses` key with only `openai-base-url`;
  - default rows: OpenAI embeddings, Pi first-party `openai`, Pi `lunaroute` decider.
- A `TestMain` installs a **tripwire** as `http.DefaultTransport`. Any request from a path that ignores the client it was given is recorded there and fails the row, and nothing leaves the process.
- Ran the same test files against `4ab5c800` in a temporary worktree (`git worktree add --detach … 4ab5c800`, files copied in, then removed):
  - `sources/02-matrix-before.txt`: 14 FAIL;
  - `sources/03-matrix-after.txt`: 38 PASS.

### Why
- The matrix tests the invariant, not each fix: a row asserts *where the marker went*, whatever mechanism stops it.
- The tripwire turns "the path built its own client" into a test failure instead of a network call.

### What worked
- Before (14 FAIL): every redirect row of OpenAI chat, Claude, Gemini, both Responses slots, Cohere embeddings, Pi openai and Pi anthropic contacted `other.example`. The Pi foreign-provider rows contacted `api.openai.com`/`api.anthropic.com`. The open-responses row contacted `api.openai.com`. All three OpenAI-embeddings rows bypassed the client.
- After: `GOWORK=off go test ./pkg/steps/ai/credscope/ -count=1` → `ok`.
- Before this change, the decision, rerank-Cohere and Pi-decider redirect rows already passed: those adapters set `CheckRedirect` to reject redirects themselves. Pinning is the second line of defence there.

### What didn't work
- The first matrix run failed `decision_systemone` with `expected a request to [gateway.example], none was sent (err selected profile is not decision-capable: missing inference_settings.decision.type …)`. `NewDecisionConfig()` leaves `Type` empty; the row now sets `systemone`/`djev`. The "must reach its own origin" assertion caught this. Without it the row would have passed vacuously.
- **The first before-run made real network requests.** Before I added the tripwire, the three OpenAI-embeddings rows on the old code ignored the recorder (go-openai's own client) and sent requests to the real `https://api.openai.com/v1/embeddings` with the dummy marker `MATRIX-SECRET` as the bearer token. OpenAI answered `401 … Incorrect API key provided: MATRIX-S*CRET`. No real credential was involved, but this is exactly inventory row 3 happening for real. The tripwire was added immediately afterwards, and the re-run (`sources/02-matrix-before.txt`) sent nothing.

### What I learned
- Every adapter whose author thought about redirects (System One, Cohere rerank, llama.cpp) already rejected them. The engines written against `EnsureHTTPClient` did not. Pinning centralizes that knowledge.

### What was tricky to build
- Making each row both drive the path and end promptly. The recorder answers 400 with a JSON error that every engine parses as a terminal error. A 401 would trigger unauthorized-refresh paths, and 5xx would trigger retries.

### What warrants a second pair of eyes
- The matrix replaces `http.DefaultTransport` for the whole test binary. That is safe because the package contains only this test, which is not parallel.

### What should be done in the future
- New credential-carrying paths must add rows. The package doc says so.

### Code review instructions
- `pkg/steps/ai/credscope/credscope_test.go`: `rows()`, then `TestNoCredentialLeavesItsOrigin`.
- To reproduce "before": `git worktree add --detach /tmp/x 4ab5c800 && cp -r pkg/steps/ai/credscope /tmp/x/pkg/steps/ai/ && (cd /tmp/x && GOWORK=off go test ./pkg/steps/ai/credscope/)`.

### Technical details
- Row shape: `row{name, run func(*testing.T, *http.Client) error, allowed []string, redirectFrom string}`.

## Step 6: Phase 5 — help topic, decisions doc, after-state probe, guide status

This step documents the behaviour for users and closes the evidence loop. There is a new Glazed help topic, `pkg/doc/topics/17-credential-scope.md` (slug `geppetto-credential-scope`), and `16-decisions.md` now describes the provider-keyed endpoint. The probe re-run on this branch is saved as `sources/04-leak-probe-after.txt`. The guide has a new §13 with the implementation status, the deviations and the open items.

### Prompt Context

**User prompt (verbatim):** see Step 1.

**Assistant interpretation:** Implement guide Phase 5 and close the ticket's documentation.

**Inferred user intent:** Leave users and reviewers able to understand the new errors and behaviour changes.

### What I did
- The help topic covers:
  - the error shape;
  - the scope sources table and the provider default table;
  - behaviour to be aware of (redirects, Responses slots, embeddings base URL and outbound policy);
  - guidance for code (`EnsurePinnedHTTPClient`, `PinClient`, adding matrix rows).
- Re-ran the probe: `GOWORK=off go run ./ttmp/…/scripts/01-leak-probe`.
- Checked rag-workbench, a downstream user of `pi.Resolve`/`Decider` and `NewOpenAIProvider`:
  - `go build ./...` passes through the workspace `go.work`;
  - `go test ./pkg/ragjs/ ./pkg/llm/... ./pkg/server/ ./cmd/... -count=1` passes.
- Guide §13: the commits table, the evidence, the deviations and the open items.

### Why
- The behaviour changes (redirects refused, open-responses keys needing their own URL, embeddings honouring `openai-base-url`) produce new errors, and each needs a user-facing explanation that names the fix.

### What worked
- Probe after: every leak row is either refused before sending, or has its marker only at the configured host (`sources/04-leak-probe-after.txt`).
- `GOWORK=off go test ./pkg/doc/... -count=1`: ok. The topic is embedded through `pkg/doc/doc.go`.

### What didn't work
- `go run ./cmd/examples/decision-pi-smoke help geppetto-credential-scope` printed ``Unknown help topic [`geppetto-credential-scope`]``. The example binary does not load Geppetto's doc FS. `help geppetto-decision-package` gives the same answer, so this is not specific to the new page. I changed the cross-reference in `16-decisions.md` from "`geppetto help …`" to "the `geppetto-credential-scope` help topic", because Geppetto has no `geppetto` binary.

### What I learned
- Geppetto's help topics are surfaced by host binaries (for example pinocchio) that load `pkg/doc`, not by the examples.

### What was tricky to build
- N/A (documentation).

### What warrants a second pair of eyes
- Whether the behaviour changes need a release note in addition to the help topic.

### What should be done in the future
- Verify the Gemini SDK cache client (guide §11).

### Code review instructions
- `pkg/doc/topics/17-credential-scope.md`; guide §13.

### Technical details
- Evidence files: `sources/01-leak-probe-output.txt` (before), `02-matrix-before.txt`, `03-matrix-after.txt`, `04-leak-probe-after.txt`.


## Step 7: Design 02 — make a bound connection the request capability

The user approved the bound-connection direction and requested implementation, appropriate commits and a detailed diary. The existing GEPPETTO-CREDSCOPE-001 ticket covers precisely this invariant, so I added design document 02 and new tasks there. The starting tree is clean at `47984a78` on `task/geppetto-credscope-001`.

The design keeps current configuration sources as the trusted declaration boundary and replaces the runtime representation: one opaque connection owns endpoint, authentication, transport and URL policy. It also closes the deferred embeddings layer pairing and verifies the modern Gemini SDK cache path. It does not invent a secret URI backend merely to reproduce illustrative pseudocode.

### Prompt Context
**User prompt (verbatim):** "alright, i don't know if there's already a ticket we can use, and add a second design doc, then implement. commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)"
**Assistant interpretation:** Reuse the credential-scoping ticket, write a second concrete design, implement the capability boundary and document the work with milestone commits.
**Inferred user intent:** Remove repeated key/URL/client assembly so ordinary feature code cannot accidentally send credentials to a fallback endpoint.

### What I did
- Read the docmgr and diary skills and the full investigation-entry format.
- Checked branch, clean status, recent commits and the latest diary checkpoint.
- Inspected settings maps, HTTP-client ownership, engine constructors and request paths, Pi resolution, embeddings merges, direct adapter constructors and the existing restricted transport middleware.
- Used `docmgr doc add` to create design-doc/02 and `docmgr task add` to add implementation milestones.
- Wrote the design including source boundaries, route validation order, SDK ownership, replay policy and trust limits.

### Why
- The existing origin guard is valuable but a new path still has to remember several independent pieces. Binding them once removes that repeated assembly.
- Reusing the restricted header middleware preserves the existing Codex model without another authentication extension system.
- Source settings and runtime capability must have distinct lifetimes; changing a public settings map after resolution must not change a credential's destination.

### What worked
- `git status --short --branch`: clean on the expected ticket branch at `47984a78`.
- `docmgr status --summary-only`: recognized the repository configuration and existing ticket tree.
- `docmgr doc add` and all five task additions succeeded.

### What didn't work
- Inventory attempted `cat pkg/steps/ai/transport/route.go` and `cat pkg/steps/ai/transport/middleware.go`; both reported `No such file or directory`. `rg --files` showed the package is implemented in `transport.go`, which I read instead. No source changes or validation failures occurred.

### What I learned
- Active Gemini inference already uses the modern SDK. The legacy cache warning concerns unused setup code; the modern cache path still needs direct verification.
- Pi's public settings currently contain raw Anthropic/Gemini keys despite the surrounding `Resolved` documentation saying settings are without secrets.
- Responses currently pins the resolved route's origin, which is too late if routing itself changes the origin.

### What was tricky to build
- Defining a useful boundary without a new configuration registry: the binding is created at each existing trusted input boundary and cannot be rebound by the request encoder.
- SDKs expect an HTTP client; the facade must preserve the guard and authentication without exposing the underlying transport or double-following redirects.

### What warrants a second pair of eyes
- Credential origin declarations are trusted configuration, not a claim that issuer ownership can be inferred from opaque API-key bytes.
- Host-provided RoundTrippers remain trusted code and cannot be sandboxed by an origin wrapper.

### What should be done in the future
- Implement the five milestones in tasks.md; replace design checkpoint status with actual delivery evidence.

### Code review instructions
- Read design-doc/02 first, then the `connection` package as it is added.
- Preserve prior diary steps as historical evidence; subsequent steps use this full investigation format.

### Technical details
- Existing reusable extension points: `transport.HeaderWriter`, `HeaderSet`, `Middleware`, `Chain`, and `ResolveAndValidate`.
- Existing public inputs remain the source boundary; no backwards compatibility layer is planned.

#### Commit storage correction

The initial commit command in the original linked checkout failed before staging: `fatal: Unable to create '/home/manuel/code/wesen/go-go-golems/geppetto/.git/worktrees/geppetto31/index.lock': Read-only file system`. Its common Git directory is outside the writable roots. I cloned the clean branch locally into the writable sibling `geppetto-bound-connections`, copied only this step's four documentation changes, and restored those four original-checkout paths to their starting state. Implementation and commits now live on `task/geppetto-bound-connections-001` in that isolated clone; the original checkout is preserved. This is a filesystem restriction, not a code failure or an automatic approval rejection.


## Step 8: Implement the bound connection and shared request executor

Added `pkg/steps/ai/connection`, whose immutable runtime object owns the provider's declared base URL, named operation routes, authentication, default headers, URL policy and private HTTP client. The reusable restricted transport middleware from #444 supplies header injection and response classification. Encoders submit an operation and body; the connection resolves, validates, authenticates, sends and governs one pre-stream unauthorized replay.

The user clarified that the purpose is to streamline the way authenticated API calls carry credentials, auth, routes and headers. Named routes are therefore part of the connection API, and the shared executor replaces duplicated send/retry loops rather than merely wrapping each existing pinned client.

### Prompt Context
**User prompt (verbatim):** "the goal is really to streamline this all and make a very clear way of passing credential auth routes headers around, for API calls to auth'ed routes"
**Assistant interpretation:** Make this a coherent API-call boundary, including route selection and headers, rather than only another credential guard.
**Inferred user intent:** Callers should pass one connection and an operation/payload without understanding credential or transport assembly.
**Commit (design):** `565dc266` — "docs(credscope): design bound connections and record implementation plan"

### What I did
- Added `connection.Config`, opaque `Connection` and `Authentication`, named `Routes`, `Relative`, `NewRequest`, `Do`, `DoWithObserver` and `SDKClient`.
- Added static and renewable token authentication with generic sensitive header carriers (`Authorization`, `x-api-key`, or another declared header).
- Reused transport header restrictions, per-invocation middleware attempt state and response hooks.
- Validated actual request origin and Host before credential lookup; validated every redirected network request with a private final guard.
- Snapshotted the declaration's headers, route map, client struct and default transport reference.
- Kept SDK placeholder credentials separate from real credentials. The SDK facade runs through the shared executor and does not reprocess redirects or cookies.
- Redacted auth/default headers both for observers and the returned `Response.Request` so successful response metadata does not hand credentials back to encoders.
- Added adversarial unit tests for foreign routes, direct foreign URLs, Host overrides, configuration mutation, redaction, redirects (301/302/303/307/308), SDK requests, same-origin redirects, a redirect callback that changes the host, concurrent bounded refresh, non-replayable bodies, invalid declarations and route traversal.

### Why
- Resolving a route must never redefine the credential scope. Scope comes from the declaration before a route resolver runs.
- Auth middleware can only set declared headers; the connection owns all routing and sending.
- One shared request executor removes the need for each engine to implement authentication refresh and redirect handling.
- The original client struct and route/header maps may be reused by the host without mutating an established connection.

### What worked
- `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache go test ./pkg/steps/ai/connection`: passed.
- Recording transports exercise real `net/http` redirect behavior without network requests.
- Concurrent requests each get independent initial/replacement tokens and at most two attempts.

### What didn't work
- The first plain `GOWORK=off go test ./pkg/steps/ai/connection` failed at setup because Go attempted to write `/home/manuel/.cache/go-build/...`: `read-only file system`. Subsequent Go commands use the writable `/tmp/geppetto-bound-go-cache` through an explicit `GOCACHE` assignment. No environment variables are read by new code.

### What I learned
- A redacted observer alone is insufficient: `http.Response.Request` otherwise exposes the authenticated request to its caller.
- Wrapping an SDK client requires one owner for redirects and cookies. The SDK facade leaves these to the inner bound connection.

### What was tricky to build
- The HTTP request body has a different lifetime from the reusable payload. Retry requires `GetBody`; non-replayable bodies do not trigger credential refresh at all.
- A host redirect callback runs before the final RoundTripper. The origin/Host guard must therefore be inside the private network transport as well as before authentication.
- Capturing the binding's base URL in a private context value prevents token sources from being keyed by an operation path or an SDK-rewritten URL.

### What warrants a second pair of eyes
- SDK facade redirect ownership and successful response redaction.
- Trust assumptions for injected network transports and custom route/middleware implementations.

### What should be done in the future
- Integrate engines and adapters with this executor, replacing their independent credential and client assembly.

### Code review instructions
- Read `connection/connection.go` from `New` through `Resolve`, `DoWithObserver` and `SDKClient`, then `auth.go`.
- Run `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache go test -race ./pkg/steps/ai/connection`.

### Technical details
- Sources receive `{Provider: declaration.Provider, BaseURL: declaration.BaseURL}`; custom source scope refusals and cancellation identities survive sanitized errors.
- `Authentication`'s zero value permits explicitly anonymous APIs; a configured token source cannot silently become anonymous.
- This is a core checkpoint; production paths have not yet been migrated.


## Step 9: Migrate inference, token counting, Pi and transcription to connections

OpenAI chat and Responses now use the connection's common authentication and bounded replay implementation. Claude's API client accepts only a connection, and its former key/URL/client/auth setters are removed. Gemini uses a bound modern SDK factory shared by all SDK services. Whisper transcription now takes the same connection used by chat or embeddings, rather than building an ambient OpenAI client from a raw key. Pi captures one source-bound connection and passes it directly to engine constructors.

The migration removes independent credential resolution from payload code and stops storing Anthropic/Gemini credentials in Pi's public resolved settings. Settings are snapshotted without credential maps or host clients for encoders. Responses validates a custom route against the original declaration's origin during construction, before any credential lookup.

### Prompt Context
**User prompt (verbatim):** "is this used for all APIs like embeddings and rerank and whisper (?) and llm and stuff> for anthropic and openai and gemini?"
**Assistant interpretation:** Apply the same setup to all existing AI API families, explicitly including Whisper transcription, embeddings and reranking.
**Inferred user intent:** Avoid a collection of separate authentication pipelines per capability or SDK.
**Commit (core):** `9b8a0f48` — "feat(connection): bind API routes, authentication and request execution"

### What I did
- Added `settings.ResolveAPIConnection` to select one slot's key/endpoint together and `BindConnection` to apply host client settings at the declaration boundary.
- Added `settings.PayloadSettings` so encoders do not retain API credential maps or host clients.
- Added `WithConnection` options for OpenAI, Responses, Claude and Gemini; conflicting separate auth options are rejected.
- Replaced OpenAI/Responses credential/retry loops with `connection.NewRequest` and `Do`; kept streaming decoding unchanged.
- Removed secret-returning Responses helpers and its duplicated bearer middleware. The existing `RequestTransport` is resolved into connection routes and middleware at construction.
- Changed Claude's low-level API client to `NewClient(connection)` and removed its mutable authentication, endpoint and client fields/setters.
- Migrated Claude and Responses token counting and added explicit connection injection.
- Added `providers/openaisdk` and `providers/geminisdk` so SDK constructors receive only an inert placeholder and the bound HTTP facade.
- Changed `NewTranscriptionClient` to take a connection and return a constructor error.
- Removed unused legacy Gemini API-key query transport/client setup. Active inference already used the modern SDK.
- Changed Pi to bind provider endpoint, scoped source and custom headers once, capture the connection, and keep raw keys out of resolved settings. System One gained a connection input so Pi decision calls use the same boundary.
- Updated tests for the explicit constructor contracts instead of restoring removed wrappers or setters.
- Added recording tests for Whisper multipart uploads and redirects; modern Gemini generation, cache, counting, embeddings and files; and mutation of Pi resolved settings across all four protocols.

### Why
- Shared ownership must replace repeated assembly, not just add another wrapper around each old client.
- A protocol route cannot become a new credential authority. Custom Responses routes are checked against the source binding.
- SDK client factories make the injected transport mandatory and keep real credential material out of SDK configuration.
- Payload encoders should depend on model/settings data, without requiring unrelated client settings to exist.

### What worked
- `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache go build ./pkg/steps/ai/... ./pkg/decision/...`: passed after removing a stale unused import.
- The original 38-row credential matrix passes with all migrated inference/Pi paths reaching their intended marker hosts.
- Recording checks passed for connection ownership, OpenAI refresh, Responses source errors/slot selection, Pi redirects/settings mutation, Codex routing/refresh and Anthropic auth headers.
- Modern Gemini's `client.go:427–435` assigns the same internal API client to Models, Caches, Files and other services. Recording tests exercise generation, cache, counting, embeddings and files directly, including refused cross-origin redirects.

### What didn't work
- The first affected-package build reported `pkg/steps/ai/claude/api/completion.go:7:2: "fmt" imported and not used`; removed the import.
- The first test compilation reported removed internal APIs (for example `undefined: resolveBearerToken`, `unknown field apiKey in struct literal of type chatStreamConfig`, `client.SetBearerAuthorization undefined`, `undefined: responsesRequestTransport` and `undefined: geminiClientOptions`). Updated behavioral tests to construct explicit connections and removed tests of obsolete legacy setup.
- After credential-free settings were introduced, matrix rows returned `missing client settings` without making a request. OpenAI and Claude payload builders had checked for a client despite never using one. Removed those obsolete checks; the matrix then passed.
- Existing socket-based tests failed before exercising code: `httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted`. The affected command was `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache go test ./pkg/steps/ai/openai/... ./pkg/steps/ai/openai_responses/... ./pkg/steps/ai/claude/... ./pkg/steps/ai/gemini/... ./pkg/steps/ai/providers/pi/... ./pkg/decision/... ./pkg/steps/ai/credscope/...`. New integration tests use recording transports; existing socket tests will be compile-checked and the execution restriction remains disclosed.
- An unused `turns` test import remained after moving invalid-route rejection to construction; removed it.
- The first Whisper test used `result.Text`; the compiler reported `result.Text undefined (type *Transcription has no field or method Text)`. The existing API returns `Transcription.Response.Text` and stores provider errors in `Transcription.Error`, so the test now checks that actual contract.
- A tool invocation adding integration tests was interrupted by user steering before execution; verified the files were absent and applied the additions once on resume.

### What I learned
- Transcription was a real additional leak surface: `NewTranscriptionClient(apiKey)` constructed an SDK client with no host endpoint or client injection.
- The modern Gemini cache path shares its bound client; the old warning was attached to unused legacy setup.
- Some tests only need an engine's observation helpers. Those tests can construct the observation state directly rather than requiring invalid runnable credentials to be accepted.

### What was tricky to build
- Pi's settings remain public and mutable for model configuration, so the closure must capture the connection independently. Mutation tests demonstrate that changing URL maps and key slots cannot move the resolved authority.
- Claude requires three carrier choices: `x-api-key`, gateway dual auth, and subscription OAuth with identity headers. These are connection configuration choices, not request-encoder branches.
- Returning a private SDK facade preserves host timeouts/proxies while preventing SDK config from acquiring the original mutable transport.

### What warrants a second pair of eyes
- Public constructor changes: Claude API client and transcription now require a connection; invalid network settings are rejected during engine construction.
- Source-bound origin checks around custom Responses routes and the canonical Codex route.
- The broader existing suite still needs execution in an environment that permits loopback test sockets.

### What should be done in the future
- Complete embeddings/rerank/direct-provider migration, make embeddings layer pairing atomic and update help and final validation.

### Code review instructions
- Start at `settings/connection.go`, then OpenAI and Responses engine constructors and reduced request senders.
- Inspect `claude/connection.go`, `claude/api/completion.go`, `providers/pi/resolve.go`, and the two SDK factories.
- Review new recording tests in `providers/geminisdk/client_test.go`, `openai/transcribe_test.go` and `providers/pi/credscope_test.go`.

### Technical details
- No new environment reads or dependencies were added.
- Static configuration is resolved once; renewable credentials remain request-time lookups within the connection.
- This checkpoint removes more request setup code than it adds; provider payloads and SSE reducers retain their existing responsibilities.


## Step 10: Complete provider migration and atomic embedding credential selection

Embedding payload clients now take a connection directly, including OpenAI, Cohere and Ollama. The OpenAI SDK factory is shared with Whisper. Rerank providers (Cohere, llama.cpp and Qwen3 completion) and System One execute through the same connection and accept a prebound connection without independently replacing its endpoint or policy. Their direct configuration constructors also establish the same binding, so skipping a settings factory does not skip enforcement.

The settings factory now selects embedding credential layers atomically. A nonblank local key or URL replaces the provider's whole pair; an incomplete local pair is refused. Per-call credential overrides have the same contract. Provider endpoint constants now live in one provider-keyed defaults package. The expanded regression matrix has 64 cases, including direct bound constructors, local providers with auth, counters, Whisper and Gemini cache.

### Prompt Context
**User prompt (verbatim):** see Step 7 for implementation/diary requirements; Step 8 for the streamlined authenticated-call goal; Step 9 for API-family scope. The user subsequently said "continue" while this work was active.
**Assistant interpretation:** Finish applying the common request boundary across the requested API families and close the deferred layer-pairing gap.
**Inferred user intent:** Each request encoder should receive one API authority, regardless of modality or SDK.
**Commit (inference):** `aa77b9b3` — "refactor(ai): share bound connections across inference, Pi and audio"

### What I did
- Changed OpenAI, Cohere and Ollama embedding constructors to accept a connection. Removed separate network/client options from these payload clients.
- Added a complete connection option to the embedding factory for host-owned bindings.
- Replaced embedding-local and per-call field merges with atomic pair selection and actionable errors for partial bindings.
- Preserved parsed-section provenance rather than decoding top-level and local credentials into the same struct. Added matching local URL inputs for credential-specific CLI sections.
- Snapshotted source configuration in the factory and retained URL policy selection at the binding boundary.
- Migrated hosted/local rerank request senders to `NewRequest`/`Do`; added prebound connection input to local providers so authenticated gateways need no new auth path.
- Centralized strict redirect rejection as `connection.Config.RejectRedirects` and removed duplicated client-cloning/route-sending helpers.
- Removed unused `settings.EnsurePinnedHTTPClient`: consumers now bind the full connection rather than assembling and separately pinning a client.
- Added `providers/defaults` as the single source of canonical runtime vendor endpoints; Pi still defaults by provider identity, never protocol.
- Normalized and copied default header names/values in the connection, including multiple values.
- Removed the OpenAI SDK placeholder: this SDK supports an empty configured token, and the connection installs real auth when needed. Only SDKs requiring a nonempty key use the inert placeholder.
- Updated behavioral tests to check the sent credential and destination instead of reading private raw-key/client fields.
- Fixed an old embedding batch test that claimed to mock its SDK but actually attempted a network call with a dummy key; it now uses an in-process recorder.
- Extended the credential matrix from 38 to 64 cases. New refusal rows cover a foreign Responses route and a local embedding key with an inherited parent URL.

### Why
- Direct constructors must establish or accept the same authority as factory-created providers.
- Layer provenance is lost after independent map merges; refusal must happen while the source layers are still distinguishable.
- An already bound connection owns URL policy; applying fresh default policy from an otherwise empty provider options struct would incorrectly reject a deliberately local connection.
- Central provider defaults reduce endpoint duplication while preserving the provider-versus-protocol distinction.

### What worked
- `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache go test ./... -run '^$'`: every repository package and test compiled, including examples and ticket probes.
- `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache go build ./...`: passed. It emitted a nonfatal VCS stat-cache warning under the read-only module cache.
- `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache GOPROXY=off go build -buildvcs=false ./...`: passed without that metadata-cache warning.
- Focused connection/matrix/embedding checks passed: all 64 credential paths, 16 atomic-pair cases across OpenAI/Cohere, factory fixtures, model/dimension behavior and in-process batch tests.
- Explicit connections with local-network permission are validated by their own policy rather than a second empty adapter policy.

### What didn't work
- The first embedding factory build reported `undefined: err` after the shared connection construction replaced the provider switch. Added one function-local error variable.
- The first local-rerank build reported `qwen3/provider.go:224:25: p.client undefined`; changed the remaining completion request to `p.connection.Do`.
- Embedding test compilation initially referenced removed fields (`ollamaProvider.baseURL`, `cohereProvider.apiKey`, `cohereProvider.httpClient`, `cohereProvider.endpoint`). Updated tests to observe requests and non-secret route metadata instead of restoring those fields.
- Expanding the route validation cases exposed `a?`: `TestInvalidDeclarationsAndRelativeRoutes` expected refusal but got nil. `url.Parse` records an empty query delimiter as `ForceQuery`; added that explicit check.
- The whole-repository build emitted `go: writing stat cache: open /home/manuel/go/pkg/mod/cache/download/github.com/go-go-golems/geppetto/@v/v0.13.14-0.20261106223354-aa77b9b365b5.info...tmp: read-only file system`. The build ultimately returned exit 0; disabling VCS stamping avoids that sandbox-only write.

### What I learned
- OpenAI's SDK accepts an empty constructor token and does not install an Authorization header for it, so it needs no inert placeholder.
- Ollama and local rerank can use authenticated gateways through the same connection even though their original adapters had no API-key fields.
- A source factory and a payload client are different boundaries: raw configuration may enter the factory/constructor, while the reusable request implementation holds only the connection.

### What was tricky to build
- Preserving local Ollama's existing explicit HTTP/local-network policy while making hosted providers fail closed.
- Keeping provider byte limits, strict response validation and pricing independent from network authority during the rerank migration.
- Ensuring a complete local override replaces both fields, while blank local entries leave the parent binding alone.
- Removing redundant route helpers without losing their traversal checks; those checks now live in the shared connection's `Relative` resolver.

### What warrants a second pair of eyes
- Constructor changes in embeddings and the new requirement for paired local credential overrides.
- Selection of the outbound policy when an adapter receives an already bound connection.
- Full runtime execution of existing loopback tests remains restricted by the environment; their code compiles and new recording checks exercise the migrated paths.

### What should be done in the future
- Audit remaining credential-source HTTP calls and provider-specific auth helpers so they cannot reintroduce an ambient client or an issuer/default mismatch.
- Complete the public usage guide, final lint/race/downstream checks and design status.

### Code review instructions
- Start at `embeddings/settings_factory.go` and `embeddings/connection_test.go` for source pairing.
- Read `credscope/bound_paths_test.go` for direct constructor coverage.
- Review rerank constructor connection branches, the shared strict redirect setting and `providers/defaults/endpoints.go`.

### Technical details
- The 64-row matrix uses marker credentials and a tripwire on the ambient transport; no real service is called.
- `Connection` remains provider-neutral; only source resolvers import canonical vendor defaults.
- An explicit connection is exclusive with separate endpoint/key/client inputs, while model and response-limit options remain independent.


## Step 11: Bind credential-source HTTP and issuer-specific authentication

A final audit of HTTP call sites found that OAuth token grants still used the context/ambient HTTP client directly, and the Umans helper returned an unbound credential option that could inherit Anthropic's default endpoint. Those are authentication paths too, so they now use the same connection boundary.

OAuth continues to own grant/body encoding and the existing explicit HTTP(S)/local token-endpoint policy, while a bound connection owns sending and rejects redirects. Umans now exposes a connection bound to its own issuer endpoint and dual-auth headers; the old unbound `ClaudeOptions` helper is removed rather than retained as a compatibility wrapper.

### Prompt Context
**User prompt (verbatim):** see Steps 7–9.
**Assistant interpretation:** Finish the audit of authenticated API routes, including built-in credential acquisition and issuer-specific bindings.
**Inferred user intent:** A new or auxiliary auth path should not bypass the common setup or accidentally inherit a protocol vendor's endpoint.
**Commit (providers):** `5b0386c9` — "refactor(providers): use bound requests and atomic credential pairs"

### What I did
- Added a private bound token client in `credentials/oauth` for both authorization-code exchange and refresh grants.
- Preserved the host's context-injected timeout/proxy/TLS transport while refusing every token-endpoint redirect.
- Marked OAuth Authorization headers sensitive for connection diagnostics; grant credential bodies remain owned by the credential source.
- Replaced `umans.ClaudeOptions(key)` with `umans.NewConnection(key, client)` bound to `https://api.code.umans.ai`, Anthropic routes and dual-auth carriers.
- Added Umans to the canonical provider defaults catalogue.
- Added in-process grant tests for successful exchange/refresh and redirect refusal, checking both HTTP Basic and form auth where the OAuth library negotiates the style.
- Added a Umans test that sends the bound key to its issuer and refuses `api.anthropic.com`.
- Hardened generic header handling for nil request headers and declarations with no header values.
- Built the repository and Glazed custom vet tools into writable `/tmp` paths for final lint.

### Why
- Credential refresh calls carry secrets in their bodies and must have a fixed destination just as inference calls do.
- A provider-specific auth helper knows the issuer and should return the full binding rather than leave its endpoint to later settings fallback.
- The connection's role is network authority; OAuth grant semantics remain in the dedicated protocol package.

### What worked
- `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache go test ./pkg/steps/ai/credentials/oauth ./pkg/steps/ai/providers/umans -run 'Test(TokenGrants|ConnectionRequires)' -count=1`: passed.
- Both OAuth grant types contact only the configured token issuer, including under 307 redirect responses.
- The Umans connection sends both verified headers to its canonical issuer and rejects a request to the Anthropic origin.
- Both lint-tool binaries built successfully from installed module sources; no dependency changes were needed.

### What didn't work
- The first OAuth redirect test assumed every exchange attempt uses Basic auth. `TestTokenGrantsUseBoundConnection/exchange/redirect` failed with `Should be true` at the Basic-auth assertion. Inspection showed `x/oauth2` retries a failed automatic header-auth probe with form auth at the same configured token endpoint. Updated the test to validate both legitimate credential carriers and the bounded two probes, while continuing to assert that no foreign origin is contacted.

### What I learned
- OAuth's auth-style negotiation is separate from the connection's one unauthorized credential replay. A grant source can perform two bounded issuer requests while probing auth style; neither follows redirects.
- The existing OAuth URL contract allows explicitly configured local/plain-HTTP token endpoints. This migration retains that declared policy while adding origin/redirect enforcement.

### What was tricky to build
- `oauth2.Config.Exchange` chooses its HTTP client through context, so its injected value must be the sealed SDK facade, not the raw host client.
- A source-specific helper must capture the issuer at its creation boundary. Passing a raw key as a Claude option reproduced the original provider-versus-protocol default problem.

### What warrants a second pair of eyes
- OAuth automatic auth-style negotiation and the explicit token-endpoint policy retained from the existing API.
- Public removal of `umans.ClaudeOptions`; callers now pass `claude.WithConnection(umansConnection)` or reuse the connection for token counting.

### What should be done in the future
- Complete final public docs and validation evidence. No further production HTTP path was identified outside the shared boundary in the migrated AI packages.

### Code review instructions
- Read `oauth.tokenConnection`, the client substitution in `ExchangeAuthorizationCode` and `Refresh`, and `umans.NewConnection`.
- Run the source-grant recording tests and the 64-row matrix.

### Technical details
- OAuth's browser authorization URL generation remains a non-network operation.
- Arbitrary host-provided credential sources and network transports are trusted code; the connection does not sandbox their implementation.


## Step 12: Remove factory auth duplication and preserve binding units through profile stacks

The public factory layer still performed its own key/URL checks before constructing an engine. That prevented the intended use of one prebound connection without credential maps. The generic engine and token-counter factories now accept a connection directly and leave network/auth validation to its constructor. Counter constructors also return binding errors immediately instead of keeping a latent error until the first request.

Inspection of profile stack resolution found another place where a valid connection could be built from an already mixed key/URL pair: recursive merging of `APISettings` and embedding credentials. Added atomic credential-slot overlay semantics there too. Changing an existing authenticated pair requires both fields; blank values preserve the original pair. Initial declarations and anonymous local endpoints retain their source semantics.

### Prompt Context
**User prompt (verbatim):** see Steps 7–9.
**Assistant interpretation:** Make the connection usable through the normal public factory APIs and keep available configuration provenance until a binding is established.
**Inferred user intent:** The common path should be sufficient for applications, and ordinary profile composition should not undo credential pairing.
**Commit (auth sources):** `4739946b` — "fix(auth): bind OAuth grants and issuer-specific credentials to connections"

### What I did
- Added `factory.WithConnection` to the standard engine factory and removed independent credential-map/base-URL validation there.
- Kept model/protocol configuration checks and normalized protocol aliases/defaults once without mutating caller settings.
- Added connection/source options to the token-counter factory; replaced its separate unbound-source entrypoint.
- Fixed the Responses counter factory path that previously ignored an injected source.
- Changed both token-counter constructors to return `(*TokenCounter, error)` and updated all repository callers/tests. Invalid bindings fail at construction.
- Added minimal payload-settings checks to engine/counter constructors even when a connection is supplied.
- Added a nil-API guard in `InferenceSettings.GetMetadata`, required when a caller supplies only model settings plus a connection.
- Added `Pi.Resolved.Connection()` so callers can reuse the opaque authority without retrieving a secret or client.
- Added `settings.CredentialSlots` / `OverlayCredentialSlots` and used them after ordinary profile merging for top-level API and embedding credential sections.
- Added factory recording tests with no API/client settings, plus profile pair/blank/immutability cases.
- Used the matching cached golangci-lint source schema for offline configuration verification.

### Why
- A factory that insists on separate raw credentials defeats the new runtime capability and encourages duplicate assembly.
- Credential validation and fallback decisions should happen once at the source boundary, rather than once in a factory and again in an engine.
- Profile provenance is available during merging. A bound client cannot later detect that independently merged settings already contain the wrong pair.
- Blank overlays must not erase an endpoint and leave its old credential available to a later default.

### What worked
- Full `go test ./pkg/engineprofiles ./pkg/cli/bootstrap -count=1`: passed, including registry/stack/bootstrap cases.
- Factory/matrix recording checks passed, including four inference protocols and both counters with `API=nil` and `Client=nil`.
- All 64 credential matrix cases continue to pass after eager counter validation and factory cleanup.
- Configuration verification with `golangci-lint config verify --schema /home/manuel/go/pkg/mod/github.com/golangci/golangci-lint/v2@v2.11.2/jsonschema/golangci.next.jsonschema.json`: passed without network access.

### What didn't work
- `make lint` stopped at remote JSON-schema loading: `dial tcp: lookup golangci-lint.run: no such host`. The installed tool has a hidden `--schema` option; its matching module sources include the schema, so the actual configuration verification can run locally.
- First analyzer run found `settings/connection.go: variable copy has same name as predeclared identifier (predeclared)`. Renamed the payload snapshot variable.
- Second analyzer run found `engineprofiles/credential_merge_test.go: ineffectual assignment to gotKeys (ineffassign)`. Removed the unnecessary initial map allocations.
- An initial factory test selection also included the pre-existing TLS-server observation test and hit the known socket restriction. Narrowed the selection to constructor/factory recording tests; all selected tests passed.

### What I learned
- The highest-level API matters: accepting a connection in leaf constructors is insufficient if the application factory still demands raw key maps.
- Recursive profile merges were a concrete remaining instance of the same bug class, rather than only an abstract future concern.
- The installed golangci-lint schema override is hidden from normal help; source inspection established the supported flag before using it.

### What was tricky to build
- Separating protocol/model validation from network authority without leaving a second credential fallback chain in the factory.
- Making pair merging preserve blank overlays, initial declarations and anonymous endpoints while refusing partial changes to an existing credential-bearing binding.
- Passing settings without API maps through event metadata code that previously assumed those maps always existed.

### What warrants a second pair of eyes
- Public counter constructor/factory signature changes and their immediate-error contract.
- Source semantics for profile overlays: complete authenticated pairs are required when overriding an existing binding.
- The factory path no longer performs credential checks separately; connection resolution is the authoritative validation.

### What should be done in the future
- Complete the public guide and final verification after this code checkpoint. Arbitrary caller-premerged maps remain trusted declarations; no library can reconstruct provenance that a host discarded before calling it.

### Code review instructions
- Read `engine/factory.CreateEngine`, the reduced validation method, and the connection-only factory tests.
- Read `settings/credential_slots.go` and `engineprofiles/credential_merge_test.go` before the profile merge integration.
- Inspect the changed token-counter constructor signatures and Pi's opaque connection accessor.

### Technical details
- No compatibility wrappers were added for removed token-counter or Umans APIs.
- Credential overlay errors name slot fields, never their values.
- Existing byte limits, SSE decoding and response validation remain owned by their payload implementations.


## Step 13: Publish the public API guide and hand off the implementation

Finished the second design document with implementation results, constructor/configuration changes, commit boundaries and validation limits. The user-facing credential-scope help topic now explains the single connection API and maps it to every supported AI API family. Updated embedding examples and added compiled generic/shared-connection examples. The existing ticket remains active for review, with its implementation tasks checked.

The completed validation gate passes, and the downstream rag-workbench builds against this isolated implementation. Golangci-lint reports zero issues; the separate repository vet tool reports six calls in an unchanged generated key-definition file. The user's latest instruction is to avoid repeated exhaustive checks at each commit, so this checkpoint records the evidence already obtained and wraps up without expanding validation.

### Prompt Context
**User prompt (verbatim):** "how far are we?"
**User follow-up (verbatim):** "ook, we don't need to check veverything at every commit, this will get figured out as we use it"
**Assistant interpretation:** Report progress, finish the authorized implementation/documentation, and avoid further repeated broad verification.
**Inferred user intent:** Prioritize a coherent, usable implementation and reviewable commits; let ordinary use reveal additional integration details.
**Commit (factory/profile):** `dd84c625` — "refactor(factory): pass connections and merge credential bindings atomically"

### What I did
- Rewrote `pkg/doc/topics/17-credential-scope.md` around the bound connection, named routes, auth/header carriers, sharing, source configuration, profile/embedding pairing and troubleshooting.
- Updated the embedding topic/tutorial constructor examples and documented the complete provider base binding.
- Added `connection/example_test.go`: a generic authenticated API operation and one connection shared by chat, embeddings and Whisper.
- Added compile-time interface assertions for the connection's diagnostic interfaces and route resolvers.
- Updated design 02's delivered API, commit table, migration effects, evidence and trust limits; added a continuation link to the historical design 01.
- Filled the ticket overview with direct design/diary/evidence links and related the focused implementation files through docmgr.
- Added reproducible recording-validation and offline-schema lint scripts under the ticket.
- Saved successful validation output, actual lint output and a concise receipt of the downstream build.
- Checked all ticket implementation tasks, updated its changelog, and reviewed staging and whitespace before the final commit.
- Verified the original linked checkout is clean and the parent workspace file is preserved.

### Why
- The API needs to be clear to application authors: bind endpoint/auth/routes/headers once, then pass the opaque connection with model/payload settings.
- Constructor and credential-overlay behavior changes must be documented where callers find the relevant API examples.
- Evidence should distinguish a passing analyzer run from a different custom tool's diagnostics and from environment-blocked tests.
- The latest user instruction favors completing useful work over repeating broad checks at every commit.

### What worked
- `scripts/02-validate-bound-connections.sh`: exit 0. Includes all 64 credential matrix cases, focused source/SDK/factory/embedding tests, full profile/bootstrap/doc tests, core/matrix/SDK/issuer race checks, repository-wide test compilation and repository-wide build. Output: `sources/05-bound-connection-validation.txt`.
- The compiled generic and shared connection examples pass.
- Offline golangci-lint schema verification passes; the final golangci-lint run reports `0 issues.`.
- Downstream `GOWORK=/tmp/geppetto-bound-validation.work GOCACHE=/tmp/geppetto-bound-go-cache GOFLAGS=-buildvcs=false GOPROXY=off go build ./...` in rag-workbench: exit 0, no diagnostics. The temporary workspace selects the isolated Geppetto implementation and existing sibling modules.
- `docmgr doc relate`, task completion and changelog updates succeeded.

### What didn't work
- The custom repository vet stage of `scripts/03-lint-with-local-schema.sh` reports six `do not call turns.DataK outside key-definition files` diagnostics at `pkg/inference/engine/turnkeys_gen.go:9–14`. That generated file and the analyzer are unchanged by this work. The script returns exit 1 and stops before the final Glazed vet stage; a successful full `make lint` run is not claimed. Raw output is retained in `sources/06-bound-connection-lint.txt`.
- Existing socket-based tests cannot open loopback listeners in this environment, as recorded in Step 9. Their packages compile; recording tests exercise the changed request boundaries.
- `git diff --check` found one extra blank line at the end of design 02; removed it before staging.

### What I learned
- The public API is now useful through both direct clients and generic factories; callers can omit credential/client maps when they supply a connection.
- Verification boundaries should track meaningful implementation milestones. A commit alone is not a reason to repeat a broad gate.

### What was tricky to build
- Explaining source declarations, runtime authority and payload configuration without teaching callers the removed assembly paths.
- Keeping the historical PR #445 diary/design intact while clearly identifying the new implementation and its broader coverage.
- Reporting lint truthfully: the nine configured golangci analyzers pass, while a separate unchanged generated-key check still reports diagnostics.

### What warrants a second pair of eyes
- Deliberate public API changes: bound embedding/transcription/Claude clients, eager counter errors, connection factory options and the replaced Umans/source helper APIs.
- Complete pair requirements for credential overrides and profile stacks.
- Remaining trust limits: raw declarations from hosts, custom Go transports/sources, DNS/proxy trust and tenant path isolation.

### What should be done in the future
- Exercise the new APIs during normal use, as requested. Run the existing socket tests in a normal environment when useful.
- Review the independent generated-key vet diagnostics in their own context; no unrelated generated code was changed here.

### Code review instructions
- Start with design 02 and `connection/example_test.go`, then read the public usage topic.
- Review the six preceding commits as bounded implementation stages; the final commit contains the public guide, examples and evidence.
- Worktree: `/home/manuel/workspaces/2026-11-02/rag-workbench/geppetto-bound-connections`; branch: `task/geppetto-bound-connections-001`.

### Technical details
- No remote messages, push, PR publication, real credential lookup or live AI API requests were performed.
- The isolated checkout is necessary because the original common Git metadata is read-only in this environment. The original checkout is preserved.
- Evidence files are readable text; the downstream receipt explicitly identifies itself as a summary of the completed command.

## Step 14: Address PR #446 OAuth endpoint and SystemOne timeout findings

The review identified two migration regressions: a complete OAuth token URL was incorrectly used as a query-free connection base, and SystemOne's default 60-second timeout disappeared with its old redirect helper. Fixed both without adding compatibility wrappers or changing caller-supplied connection policy.

The new recording tests reproduce both failures before the implementation changes and pass afterward, including the race detector. The OAuth cases cover exchange and refresh with and without query parameters, plus refused foreign redirects; SystemOne covers default, explicit positive, and explicit zero timeouts.

### Prompt Context
**User prompt (verbatim):** "ok, fix, push"
**Assistant interpretation:** Fix the two review findings discussed immediately before this request and publish the follow-up to PR #446.
**Inferred user intent:** Preserve valid OAuth endpoints and bounded default decision requests while continuing to streamline request ownership.
**Source implementation:** local `b8566b6e`, published with the identical tree as PR head `ba79888f8f8b9e41a0e58c6db430a6517b4e5e2e`.

### What I did
- In `oauth.Client.tokenConnection`, parse a copy of TokenURL and remove RawQuery/ForceQuery only from the binding declaration. Exchange and Refresh retain the complete original request URL.
- In `systemone.New`, supply a 60-second HTTP client only when constructing a connection without an injected client. Leave explicit clients and complete supplied connections unchanged.
- Expand the recording OAuth grant matrix with repeated query values, encoded content, and a bare flag, checking the raw query exactly at the transport boundary.
- Add `TestDirectClientTimeoutPolicy`, which observes the actual outgoing request context deadline without waiting or opening a socket. Check that explicit client settings are preserved and the constructor does not mutate the injected client's redirect policy.

### Why
- Request query parameters and origin authority are different inputs. Keeping query-free base validation preserves the connection contract without discarding valid grant parameters.
- Timeout is service request policy that was accidentally coupled to the removed redirect helper. Restoring the default at construction retains the useful policy, without changing all connections globally.

### What worked
- Before the fixes, the targeted command failed with `OAuth token endpoint binding failed` on query-bearing endpoints and `request has no client deadline` for the default SystemOne client.
- After the fixes, `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache GOFLAGS=-buildvcs=false GOPROXY=off go test ./pkg/steps/ai/credentials/oauth ./pkg/decision/systemone -run 'TestTokenGrantsUseBoundConnection|TestDirectClientTimeoutPolicy' -count=1` passed both packages.
- The same command with `-race` passed both packages. Recording transports made no live provider requests.

### What didn't work
- `GOWORK=off GOCACHE=/tmp/geppetto-bound-go-cache GOFLAGS=-buildvcs=false GOPROXY=off go test ./pkg/steps/ai/credentials/oauth -count=1` hit the existing environment restriction: `httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted`, in `TestExchangeAuthorizationCodeNormalizesCredential`. This is not reported as a passing full OAuth suite. The focused recording regressions and race checks execute without listeners.

### What I learned
- Consolidating request execution does not automatically carry over service defaults formerly hidden in transport helpers.
- A complete operation URL can validly contain query parameters even when a connection's base declaration cannot.

### What was tricky to build
- Keeping the original token URL byte representation for requests, including repeated parameters and encoding, while removing query state from only the binding copy.
- Verifying a 60-second default without a 60-second delay: inspect the context deadline supplied to the recording transport, and separately prove explicit zero remains unbounded.

### What warrants a second pair of eyes
- Verify that no TokenURL mutation reaches `c.config.Endpoint.TokenURL` and that redirects remain rejected for both grant types.
- Confirm default policy is applied only in the nil-client construction branch; an already bound connection remains authoritative.

### What should be done in the future
- Consider connection-only provider constructors and factory-owned source/default resolution as a separate simplification, not part of this bounded review fix.

### Code review instructions
- Review `oauth.go: tokenConnection`, `connection_test.go: TestTokenGrantsUseBoundConnection`, `systemone/provider.go: New`, and `timeout_test.go: TestDirectClientTimeoutPolicy`.
- Run the focused command above, optionally with `-race`; no provider access or loopback listeners are required.

### Technical details
- Review threads: https://github.com/go-go-golems/geppetto/pull/446#discussion_r4498812896 and https://github.com/go-go-golems/geppetto/pull/446#discussion_r4498812911.
- PR #446 targets the upstream mirror `stack/geppetto-445-credscope` at #445's head `47984a78` and therefore isolates the continuation's changes.
- Remote publication uses the GitHub connector because shell DNS resolution is unavailable. Local source and published commit IDs can differ; compare tree identities for content equivalence.
