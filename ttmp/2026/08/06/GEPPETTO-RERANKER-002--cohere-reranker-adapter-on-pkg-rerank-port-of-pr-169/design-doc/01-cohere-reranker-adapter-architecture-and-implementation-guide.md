---
Title: 'Cohere reranker adapter: architecture and implementation guide'
Ticket: GEPPETTO-RERANKER-002
Status: active
Topics:
    - geppetto
    - inference
    - providers
    - architecture
    - embeddings
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/js/modules/geppetto/api_reranker.go
      Note: Goja surface that gains cohere support for free via the factory
    - Path: repo://pkg/rerank/config/settings.go
      Note: RerankConfig reused by the cohere provider (type/engine/byte limits)
    - Path: repo://pkg/rerank/factory/settings_factory.go
      Note: Factory to extend with the cohere provider type and API key resolution
    - Path: repo://pkg/rerank/llamacpp/provider.go
      Note: Reference adapter whose pipeline and strictness the Cohere provider mirrors
    - Path: repo://pkg/rerank/rerank.go
      Note: Core Provider interface and Request/Response types the Cohere adapter implements
    - Path: repo://pkg/security/outbound_url.go
      Note: Outbound URL policy applied to the Cohere endpoint
ExternalSources:
    - https://docs.cohere.com/reference/rerank
    - https://github.com/go-go-golems/geppetto/pull/169
Summary: 'Intern-facing architecture and implementation guide for porting the PR #169 Cohere reranker onto the modern pkg/rerank primitive as pkg/rerank/cohere, discarding the deprecated embeddings.Reranker API.'
LastUpdated: 2026-08-06T00:00:00-04:00
WhatFor: Step-by-step implementation guide for the Cohere rerank adapter, factory/config/validation wiring, tests, and docs.
WhenToUse: Use when implementing or reviewing the Cohere rerank provider for geppetto.
---


# Cohere reranker adapter: architecture and implementation guide

## 1. Executive summary

Geppetto has three model-service primitives: inference (`pkg/inference/engine`), embeddings (`pkg/embeddings`), and reranking (`pkg/rerank`). The reranking primitive shipped in GEPPETTO-RERANKER-001 with a single provider: a strict llama.cpp `/v1/rerank` adapter. The design explicitly reserved a slot for hosted providers — *"Consider Cohere or Jina adapters only after the first local provider is complete"* (GEPPETTO-RERANKER-001 investigation diary).

This ticket adds the second provider: **a Cohere v2 `/rerank` adapter in `pkg/rerank/cohere`**, implemented against the transport-neutral `rerank.Provider` interface, constructed through the existing settings factory from engine profiles and `InferenceSettings`, and automatically exposed to JavaScript through the existing `gp.reranker(settings)` Goja API.

The work is a **port, not a merge**. Pull request #169 (May 2025) added a Cohere reranker as `pkg/embeddings.CohereReranker` behind a small `embeddings.Reranker` interface. That interface predates `pkg/rerank` and is strictly weaker: no caller-owned document IDs, no usage/cost/duration propagation, no deterministic ordering guarantees, no bounded IO, no outbound URL policy, no profile integration. We salvage the protocol knowledge (endpoint shape, headers, response DTO) and **discard the legacy API**. Nothing from PR #169 lands as-is; PR #169 should be closed once this ticket ships.

By the end of this guide you will understand:

- what reranking is and where it sits in geppetto's architecture;
- the exact contract of `pkg/rerank` (types, invariants, sentinel errors);
- how the llama.cpp adapter works, as the reference implementation to mirror;
- the Cohere v2 rerank HTTP API;
- how settings, profiles, the factory, and the JavaScript API fit together;
- the precise files to create or edit, with pseudocode and a test matrix.

## 2. Problem statement and scope

### 2.1 Problem

Geppetto can rerank documents only against a self-hosted llama.cpp server. Users who want a hosted, zero-ops reranker (Cohere `rerank-v3.5` and successors) have no supported path. PR #169 attempted this a year ago but targeted an API surface (`pkg/embeddings`) that no longer exists in that form.

### 2.2 In scope

1. New package `pkg/rerank/cohere`: a strict Cohere v2 `/rerank` adapter implementing `rerank.Provider`.
2. Factory wiring: `pkg/rerank/factory` recognizes `rerank.type: cohere`, resolves the API key and optional base-URL override from `InferenceSettings.API` maps, and constructs the provider.
3. Validation wiring: `ValidateInferenceSettingsForRerank` accepts `cohere` and produces profile-oriented diagnostics for missing keys.
4. Unit tests with a mock HTTP server, mirroring the llamacpp test style; factory/validation tests; one Goja integration test proving the factory path needs no JS changes.
5. Documentation: extend the reranking topic guide (`pkg/doc/topics/15-reranking.md`) with a Cohere section.

### 2.3 Out of scope (explicitly dropped legacy)

- The `embeddings.Reranker` interface, `RerankOption` functional options, and `RankResult` type from PR #169. **Not ported, not shimmed.** Per repository policy we do not add backwards-compatibility adapters.
- `max_tokens_per_doc` request control. The core `rerank.Request` has no per-document truncation knob; callers pre-truncate document text. Recorded as DR-4.
- A Cohere **embeddings** provider (also in PR #169). Separate concern; tracked separately.
- A live Cohere API test in CI. Live tests remain opt-in, following the llamacpp precedent.
- Any ragkit/ragopt adapter. Downstream RAG integration is deferred (RESEARCHCTL-015 lineage).

## 3. Background: what reranking is

Reranking is the second stage of a retrieval pipeline. A cheap first stage (BM25, vector similarity) returns a few hundred candidate documents; a **cross-encoder reranker** then scores each `(query, document)` pair jointly and reorders the candidates by true relevance. Cross-encoders are far more accurate than first-stage retrieval but far more expensive, so they run only over the shortlist.

```
                retrieval pipeline
┌──────────┐   ┌──────────────────┐   ┌────────────────┐
│  query   │──▶│ first stage       │──▶│  top-K (K~100) │
└──────────┘   │ (BM25 / vector)   │   │  candidates    │
               └──────────────────┘   └───────┬────────┘
                                              ▼
                                     ┌────────────────┐
                                     │  RERANKER       │  ◀── this ticket
                                     │  cross-encoder  │      (Cohere adapter)
                                     └───────┬────────┘
                                             ▼
                                     top-N reordered results
                                     + scores, usage, cost
```

Geppetto owns the reranker box and nothing else. It deliberately knows nothing about chunks, evidence, fusion, hydration, citations, or evaluation — those belong to the calling application (e.g. a RAG system). The dependency direction is:

```
application (RAG system)
    └─▶ geppetto/pkg/rerank          (transport-neutral core)
            ├─▶ pkg/rerank/llamacpp  (local provider, exists)
            └─▶ pkg/rerank/cohere    (hosted provider, THIS TICKET)

geppetto/pkg/rerank ──✗──▶ any RAG or application package
```

## 4. Current-state architecture

This section maps every part of the system you will touch. All file references are real; open them as you read.

### 4.1 The core: `pkg/rerank`

The core package defines the transport-neutral contract. It has no HTTP code and no provider imports.

**Types (`pkg/rerank/rerank.go`):**

- `Document{ID, Text}` — one candidate. `ID` is the durable, caller-controlled identity. Application metadata and first-stage scores stay **outside** the provider request by design (reduces accidental disclosure to providers).
- `Request{Model, Query, Documents, TopN}` — one rerank call. `TopN` is explicit; the package never silently defaults cardinality. Callers needing one score per document set `TopN == len(Documents)`.
- `Result{DocumentID, Index, Score, Rank}` — one scored document mapped back to caller identity. Scores are raw provider values (may be negative; **not** probabilities).
- `Usage{InputTokens, TotalTokens}` — provider-reported token consumption. A **nil `*Usage` means "provider did not report usage"** — distinct from zero.
- `Response{Provider, Model, Results, Usage, Cost, RequestID, DurationMs}` — the rich response built for scientific runs: what answered, how much it cost, how long it took. `Cost` is nil when pricing is unknown; nil and zero are intentionally distinguishable.
- `Model{Provider, Name}` — provider identity.
- `Provider` interface (`rerank.go:93-100`):

```go
type Provider interface {
    Rerank(ctx context.Context, in Request) (Response, error)
    Model() Model
}
```

**Validation (`pkg/rerank/validate.go`):** `ValidateRequest(in, providerModel)` rejects empty queries, zero documents, empty/duplicate document IDs, empty text, `TopN` outside `[1, len(Documents)]`, and request/provider model conflicts. Note the safety contract: errors **never echo caller document IDs or text** — error messages identify inputs by position only.

**Ordering (`pkg/rerank/order.go`):** `ValidateAndMapResults(documents, topN, raw)` is the heart of response handling. Providers return index/score pairs; this function validates them (exact cardinality, index in range, no duplicates, finite scores, both fields present) and maps indices back to caller `DocumentID`s. Then `SortResults` orders deterministically: score descending → input index ascending → document ID ascending. `AssignRanks` numbers from 1. Response order is **never** treated as durable identity.

**Errors (`pkg/rerank/errors.go`):** five sentinel categories for caller classification: `ErrInvalidRequest`, `ErrInvalidResponse`, `ErrUnavailable`, `ErrRequestTooLarge`, `ErrResponseTooLarge`. The package-level safety contract: sentinel-derived errors must never contain query text, document text, authorization headers, endpoint userinfo, or provider response bodies.

### 4.2 The reference adapter: `pkg/rerank/llamacpp`

The llama.cpp adapter is the implementation you should mirror. Read it fully before writing code: `pkg/rerank/llamacpp/provider.go` (373 lines) and `protocol.go`.

Its pipeline, which the Cohere adapter will replicate almost step for step:

```
Rerank(ctx, in)
 1. rerank.ValidateRequest(in, p.Model())           # caller input
 2. effectiveModel = rerank.ResolveModel(in, ...)
 3. strip caller IDs: documents[i] = doc.Text        # IDs never leave the process
 4. json.Marshal(request{Model, Query, Documents, TopN})
 5. reject if len(payload) > MaxRequestBytes         # ErrRequestTooLarge
 6. POST endpoint (context-aware), Content-Type: application/json
 7. non-2xx → drain body bounded, ErrUnavailable     # body never surfaced
 8. readAtMost(body, MaxResponseBytes)               # ErrResponseTooLarge
 9. decodeStrict: DisallowUnknownFields + reject trailing JSON
10. rerank.ValidateAndMapResults(in.Documents, in.TopN, raw)
11. response model mismatch → ErrInvalidResponse
12. build Response{Provider, Model, Results, Usage, Cost, RequestID, DurationMs}
```

Key adapter behaviors to copy:

- **Construction-time validation** (`New`): required `BaseURL` and `Model`, no silent localhost default, strict base-URL hygiene (scheme, host, no userinfo, no query/fragment, no dot segments), byte limits defaulting to 2 MiB request / 1 MiB response, endpoint built with `url.JoinPath` and re-validated under `security.ValidateOutboundURL`.
- **Redirect rejection**: an injected `*http.Client` is shallow-cloned and its `CheckRedirect` is replaced; the caller's client is never mutated (`cloneClientWithRedirectRejection`).
- **Transport error redaction** (`redactTransportError`): `net/url` errors can embed redirect targets, proxy URLs, userinfo, or query strings — the original error is deliberately discarded and replaced with the stable `ErrUnavailable` sentinel.
- **Pointer DTO fields** (`protocol.go`): `*int` index / `*float64` score distinguish "missing" from "valid zero"; the adapter rejects responses with missing fields via `RawResult.HasIndex/HasScore`.

### 4.3 Configuration: `pkg/rerank/config` + `InferenceSettings`

`RerankConfig` (`pkg/rerank/config/settings.go`) holds semantic config only:

```go
type RerankConfig struct {
    Type             string `yaml:"type,omitempty" glazed:"rerank-type"`
    Engine           string `yaml:"engine,omitempty" glazed:"rerank-engine"`
    MaxRequestBytes  int64  `yaml:"max_request_bytes,omitempty" glazed:"rerank-max-request-bytes"`
    MaxResponseBytes int64  `yaml:"max_response_bytes,omitempty" glazed:"rerank-max-response-bytes"`
}
```

Endpoints and credentials are **not** stored here. They live in the existing `InferenceSettings.API` maps (`BaseUrls`, `APIKeys`, `AllowHTTP`, `AllowLocalNetworks`) and `InferenceSettings.Client` (timeout, proxy) — exactly like embeddings — so profile composition stays uniform across all model-service primitives. `InferenceSettings.Rerank` is wired at `pkg/steps/ai/settings/settings-inference.go:87`.

The Glazed section (`pkg/rerank/config/flags/rerank.yaml`) exposes the four fields as CLI/YAML flags and is registered in `pkg/sections/sections.go:95-128` and `pkg/cli/bootstrap/inference_debug.go`. **No new section is needed for Cohere** — it reuses `rerank.type`, `rerank.engine`, and the API maps.

Engine profiles overlay rerank config automatically (see `pkg/engineprofiles/stack_merge_rerank_test.go`); a profile stack can therefore bind `type: cohere`, `engine: rerank-v3.5`, and the API key maps without any new merge code.

### 4.4 The factory: `pkg/rerank/factory`

`pkg/rerank/factory/settings_factory.go` breaks the import cycle between the core types and the adapters:

```
pkg/rerank            (core: Request/Response/Provider, no provider imports)
pkg/rerank/llamacpp ─▶ pkg/rerank
pkg/rerank/cohere   ─▶ pkg/rerank          (new)
pkg/rerank/factory  ─▶ core + ALL adapters + settings   (the only place that knows every provider)
```

Today (`settings_factory.go:17-19, 64-67, 70-106`):

- provider registry: a single constant `rerankProviderLlamaCpp = "llamacpp"`;
- `SupportedProviders()` returns `["llamacpp"]`;
- `NewProvider()` switches on `config.Type`, resolves the base URL from `api.BaseUrls["rerank-base-url"]`, builds the outbound URL policy from `api.AllowHTTP["rerank"]` / `api.AllowLocalNetworks["rerank"]`, obtains an HTTP client via `settings.EnsureHTTPClient(f.client)` (`pkg/steps/ai/settings/http_client.go:98`), and constructs the llamacpp provider;
- `ValidateInferenceSettingsForRerank(s)` gives profile-oriented errors before construction (missing `rerank` section, missing type/engine, unsupported type, missing base URL).

### 4.5 JavaScript: `gp.reranker(settings)`

`pkg/js/modules/geppetto/api_reranker.go` exposes `gp.reranker(settings)` where `settings` must be a registry-resolved `InferenceSettings` wrapper. The builder calls `rerankfactory.NewSettingsFactoryFromInferenceSettings(...)` and `factory.NewProvider()` — **construction goes through the factory, so a new provider type needs zero JavaScript changes.** JavaScript can never supply endpoints, credentials, HTTP clients, or provider callbacks; it only sees `rerank(query, documents, options)`, `rerankAsync(...)`, and `model()`. This is a deliberate security property of the 001 design.

### 4.6 Outbound security: `pkg/security`

`security.ValidateOutboundURL(rawURL, OutboundURLOptions{AllowHTTP, AllowLocalNetworks})` rejects non-http(s) schemes and local-network targets unless explicitly allowed. For llama.cpp (a localhost server), profiles opt into HTTP/local networks. For Cohere (a hosted SaaS), the defaults are already correct: HTTPS required, local networks denied.

### 4.7 The legacy artifact being discarded (PR #169)

For completeness, here is what PR #169 added and why each piece dies:

| PR #169 artifact | Fate | Reason |
|---|---|---|
| `embeddings.Reranker` interface (`rerank.go`) | **Discard** | Superseded by `rerank.Provider`; weaker in every dimension (no doc IDs, no usage, no ordering guarantee) |
| `RerankOption`/`WithTopN`/`WithMaxTokensPerDoc` | **Discard** | `TopN` is now an explicit `Request` field; truncation is caller-side (DR-4) |
| `RankResult{Index, Document, Score}` | **Discard** | Echoing document text into results violates the core's identity model; `Result{DocumentID, Index, Score, Rank}` replaces it |
| `CohereReranker` HTTP code | **Salvage protocol only** | Endpoint, headers (`Authorization: Bearer`, `X-Client-Name`), request/response DTO shape are correct and reused |
| Unbounded `io.ReadAll`-style decode, error bodies in messages | **Discard** | Violates bounded-IO and safe-error contracts |
| `ttmp/2025-05-13` Cohere API notes | **Salvage** | Folded into §5 of this guide |

## 5. The Cohere v2 rerank API (reference)

Source: PR #169's captured notes (`ttmp/2025-05-13/01-cohere-rerank-api.md`) and the official reference at <https://docs.cohere.com/reference/rerank>.

**Endpoint:** `POST https://api.cohere.com/v2/rerank`, `Content-Type: application/json`.

**Headers:**

| Header | Required | Notes |
|---|---|---|
| `Authorization: Bearer <key>` | yes | Trial and production keys |
| `X-Client-Name` | no | We send `go-go-golems/geppetto` (carried over from PR #169) |

**Request:**

```json
{
  "model": "rerank-v3.5",
  "query": "what is the capital of the united states",
  "documents": ["Carson City is ...", "Washington, D.C. ..."],
  "top_n": 2
}
```

- `model`, `query`, `documents` required; `top_n` optional on the wire (our core always sends it — cardinality is explicit in geppetto).
- `max_tokens_per_doc` (default 4096) exists on the wire but we do not expose it (DR-4).
- Cohere recommends < 1000 documents per request.

**Response (200):**

```json
{
  "id": "9f7c...-request-id",
  "results": [ { "index": 3, "relevance_score": 0.98 }, { "index": 1, "relevance_score": 0.71 } ],
  "meta": {
    "api_version": { "version": "2.0", "is_experimental": false },
    "billed_units": { "search_units": 1 }
  }
}
```

- `id` maps directly to `rerank.Response.RequestID` (llama.cpp had to fish it from a header; Cohere puts it in the body).
- `relevance_score` is normalized to `[0, 1]` for Cohere — but the core still treats scores as opaque floats; no probability interpretation anywhere.
- `meta.billed_units.search_units` counts **searches**, not tokens. See DR-3 for the usage/cost mapping decision.

**Errors:** non-200 with a JSON body `{"message": "..."}`. Status codes include 400/401/403/404/422/429/498/499/500/501/503/504. We classify every non-2xx as `ErrUnavailable` and **never** surface the body (it may echo request content).

## 6. Proposed design

### 6.1 Package layout

```
pkg/rerank/cohere/
├── doc.go            # package documentation (threat model + invariants)
├── protocol.go       # request/response wire DTOs (pointer fields)
├── provider.go       # Provider: New + Rerank + helpers
├── provider_test.go  # mock-server unit tests (see §8)
└── live_test.go      # opt-in live test, gated on COHERE_API_KEY env var
```

Everything else is small edits: one new case in the factory, extended validation, factory tests, docs.

### 6.2 Provider options and construction

Mirroring llamacpp's `Options`, with the hosted-provider deltas:

```go
// pkg/rerank/cohere/provider.go

type Options struct {
    APIKey           string   // required; never logged, never in errors
    BaseURL          string   // optional; default "https://api.cohere.com"
    Model            string   // required, e.g. "rerank-v3.5"
    HTTPClient       *http.Client
    OutboundURL      security.OutboundURLOptions
    MaxRequestBytes  int64    // default 2 MiB
    MaxResponseBytes int64    // default 1 MiB
    CostPerSearch    *float64 // optional; see DR-3
}

func New(options Options) (*Provider, error)
```

Pseudocode for `New`:

```
func New(opts):
    apiKey = trim(opts.APIKey); if empty → ErrInvalidRequest("cohere api key is required")
    baseURL = trim(opts.BaseURL); if empty → baseURL = DefaultBaseURL  // hosted default IS unambiguous
    parsed = url.Parse(baseURL)                       // never wrap parse errors (may contain userinfo)
    validateBaseURL(parsed)                           # same hygiene rules as llamacpp
    model = trim(opts.Model); if empty → ErrInvalidRequest
    maxReq/maxResp = defaults when zero, reject negative
    endpoint = url.JoinPath(baseURL, "v2/rerank")
    security.ValidateOutboundURL(endpoint, opts.OutboundURL)   # HTTPS + no-local by default
    client = cloneClientWithRedirectPolicy(opts.HTTPClient)    # see DR-2
    return &Provider{...}
```

One deliberate difference from llamacpp: `BaseURL` has a **default** (`https://api.cohere.com`). A hosted provider has one canonical endpoint, so requiring every profile to spell it out is noise; the override exists for proxies and tests. The default is still HTTPS-validated through the same outbound policy.

### 6.3 The Rerank pipeline

```
func (p *Provider) Rerank(ctx, in):
    started = now()
    rerank.ValidateRequest(in, p.Model())                 # same core validation
    effectiveModel = rerank.ResolveModel(in, p.Model())
    texts = [doc.Text for doc in in.Documents]            # caller IDs stay local

    payload = json.Marshal(request{Model, Query, Documents: texts, TopN})
    if len(payload) > p.maxRequestBytes → ErrRequestTooLarge

    req = POST p.endpoint with ctx, body=payload
    req.Header["Content-Type"] = "application/json"
    req.Header["Accept"]       = "application/json"
    req.Header["Authorization"] = "Bearer " + p.apiKey     # header only; never in errors
    req.Header["X-Client-Name"] = "go-go-golems/geppetto"

    resp = p.client.Do(req)
    if err → redactTransportError()                       # discard original, ErrUnavailable
    if resp.StatusCode not 2xx:
        drainBounded(resp.Body, p.maxResponseBytes)
        → "cohere endpoint returned status %d: ErrUnavailable"

    raw, tooLarge, err = readAtMost(resp.Body, p.maxResponseBytes)
    if tooLarge → ErrResponseTooLarge
    wire = decodeStrict(raw)                              # unknown fields + trailing data rejected
    if err → ErrInvalidResponse

    results = rerank.ValidateAndMapResults(in.Documents, in.TopN, toRawResults(wire.Results))

    return rerank.Response{
        Provider:   ProviderName,                        # "cohere"
        Model:      effectiveModel,
        Results:    results,
        Usage:      nil,                                 # DR-3: search_units are not tokens
        Cost:       computeSearchCost(wire.Meta, p.costPerSearch),   # DR-3
        RequestID:  wire.ID,                             # body-carried request id
        DurationMs: &durationMs,
    }
```

Helpers `drainBounded`, `readAtMost`, `decodeStrict`, `toRawResults`, and the client-cloning function are small and duplicated per adapter by design (each adapter owns its strictness; premature sharing couples providers). Follow the llamacpp file structure so reviewers can diff the two adapters mechanically.

### 6.4 Wire DTOs (`protocol.go`)

```go
type request struct {
    Model     string   `json:"model"`
    Query     string   `json:"query"`
    Documents []string `json:"documents"`
    TopN      int      `json:"top_n"`
}

type response struct {
    ID      string   `json:"id,omitempty"`
    Results []item   `json:"results"`
    Meta    *meta    `json:"meta,omitempty"`
}

type item struct {
    Index          *int     `json:"index"`
    RelevanceScore *float64 `json:"relevance_score"`
}

type meta struct {
    APIVersion   *apiVersion  `json:"api_version,omitempty"`
    BilledUnits  *billedUnits `json:"billed_units,omitempty"`
}

type billedUnits struct {
    SearchUnits int `json:"search_units"`
}
```

`decodeStrict` uses `DisallowUnknownFields`, so adding fields to the DTO later is a conscious act — silent API drift surfaces as test failures, not misparsed data.

### 6.5 Factory and validation wiring

Edits to `pkg/rerank/factory/settings_factory.go`:

```go
const (
    rerankProviderLlamaCpp = "llamacpp"
    rerankProviderCohere   = "cohere"          // NEW
)

func (f *SettingsFactory) SupportedProviders() []string {
    return []string{rerankProviderLlamaCpp, rerankProviderCohere}   // NEW
}
```

New case in `NewProvider()`:

```
case rerankProviderCohere:
    apiKey = trim(f.api.APIKeys["cohere-api-key"])          # embeddings uses the same key name
    if apiKey == "" → ErrInvalidRequest(
        "selected rerank profile has no cohere-api-key; " +
        "set inference_settings.api.api_keys.cohere-api-key")
    baseURL = trim(f.api.BaseUrls["cohere-base-url"])       # optional override; "" = default
    outbound = resolveOutboundURLOptions()                  # same "rerank" policy keys
    httpClient = settings.EnsureHTTPClient(f.client)
    return cohere.New(cohere.Options{
        APIKey: apiKey, BaseURL: baseURL, Model: engine,
        HTTPClient: httpClient, OutboundURL: outbound,
        MaxRequestBytes: f.config.MaxRequestBytes,
        MaxResponseBytes: f.config.MaxResponseBytes,
        CostPerSearch: f.resolveSearchCost(),               # nil until priced; see DR-3
    })
```

And in `ValidateInferenceSettingsForRerank`: accept `cohere` as a type, require the API key for it (instead of the llamacpp `rerank-base-url` requirement), keep llamacpp's checks unchanged. Both provider diagnostics must mention the exact YAML path the user needs to fix — that is the established style.

A minimal profile for users then looks like:

```yaml
inference_settings:
  api:
    api_keys:
      cohere-api-key: "${COHERE_API_KEY}"
  rerank:
    type: cohere
    engine: rerank-v3.5
```

and JavaScript gets it for free: `gp.reranker(resolvedSettings).rerank(query, docs, { top_n: 5 })`.

### 6.6 Decision records

**DR-1 — Port to `pkg/rerank/cohere`, do not resurrect `embeddings.Reranker`.**
Context: PR #169's interface predates the 001 primitive. Options: (a) merge PR #169 as-is; (b) keep both interfaces; (c) port protocol knowledge onto `rerank.Provider` and delete the legacy. Decision: (c). Rationale: two rerank APIs would fracture callers; the legacy interface cannot express doc IDs, usage, or deterministic ordering; repository policy forbids compatibility shims without explicit request. Consequences: PR #169 is closed unmerged; ragkit integration later adapts `rerank.Provider` → `rag.Reranker`. Status: accepted.

**DR-2 — Reject redirects for the Cohere adapter, same as llamacpp.**
Context: the 001 diary notes hosted providers "may legitimately redirect" and suggests extracting redirect policy when a hosted provider is added. Options: allow redirects / keep rejection / make it an option. Decision: keep rejection. Rationale: the canonical endpoint is fixed and HTTPS; a redirect from `api.cohere.com` is an anomaly worth failing closed on; silent redirect-following can move credentials across origins (Go strips `Authorization` on cross-host redirect, but failing closed is still clearer to audit). Consequences: if Cohere ever moves the endpoint, construction-time `BaseURL` override is the migration path. Status: accepted.

**DR-3 — `Usage` stays nil; `Cost` computed from search units only when a per-search rate is configured.**
Context: Cohere reports `meta.billed_units.search_units` (count of searches), not tokens. `rerank.Usage` models tokens. Options: (a) stuff search units into `Usage.InputTokens`; (b) leave `Usage` nil and add optional per-search cost; (c) extend the core `Usage` struct. Decision: (b). Rationale: (a) corrupts the token semantics that scientific runs aggregate on; (c) changes a frozen core API for one provider's billing quirk — the core comment says nil means "not reported", which is honest here. Consequences: `Response.Usage == nil` for Cohere; `Response.Cost` is nil unless `CostPerSearch` is configured (default nil → unknown). ModelInfo has no search-rate field today, so the factory passes nil; wiring a rate is a future config change, not a core change. Status: accepted.

**DR-4 — No `max_tokens_per_doc` passthrough.**
Context: the Cohere wire supports it; PR #169 exposed it. The core `Request` has no per-document options. Decision: omit; callers pre-truncate `Document.Text`. Rationale: keeps the core request transport-neutral; truncation policy is an application concern (it interacts with chunking). Status: accepted.

**DR-5 — `BaseURL` defaults to `https://api.cohere.com`; override via `cohere-base-url`.**
Context: llamacpp has no default (a generic library must not point at localhost silently). A hosted provider has exactly one canonical endpoint. Decision: default with override. Status: accepted.

## 7. Phased implementation plan

Each phase ends with a commit. Run `gofmt`/`go build`/`go test` before every commit.

**Phase 1 — adapter core.** Create `pkg/rerank/cohere/{doc.go,protocol.go,provider.go}`. No factory changes yet. Exit gate: `go build ./pkg/rerank/cohere/` and a compile-time `var _ rerank.Provider = (*Provider)(nil)`.

**Phase 2 — adapter tests.** `provider_test.go` with `httptest.Server`: happy path (scores sorted, IDs mapped, ranks assigned, RequestID from body), empty-documents rejection, TopN bounds, duplicate/out-of-range/non-finite score rejection, missing index/score rejection, non-2xx classification, oversized request/response, trailing-JSON rejection, redirect rejection, transport-error redaction, authorization header sent. Mirror `pkg/rerank/llamacpp/provider_test.go` structure. Exit gate: `go test ./pkg/rerank/cohere/ -race`.

**Phase 3 — factory + validation.** Edit `pkg/rerank/factory/settings_factory.go` (new constant, `SupportedProviders`, `NewProvider` case, `ValidateInferenceSettingsForRerank` branches). Extend `settings_factory_test.go`: cohere happy path from `InferenceSettings`, missing-key diagnostic, unsupported-type message now lists both providers, base-URL override honored. Exit gate: `go test ./pkg/rerank/...`.

**Phase 4 — Goja parity proof.** One test in `pkg/js/modules/geppetto` (extend `api_reranker_test.go` pattern): construct a registry-resolved `InferenceSettings` with `rerank.type: cohere` and a mock base URL, call `gp.reranker(settings).rerank(...)`, assert results. This proves DR-adjacent claim "the JS wrapper needs no change". Exit gate: `go test ./pkg/js/modules/geppetto/ -run Rerank`.

**Phase 5 — docs + live test.** Extend `pkg/doc/topics/15-reranking.md` with a Cohere section (profile YAML, JS snippet, security notes). Add `live_test.go` gated on `COHERE_API_KEY` (skip by default). Exit gate: docs render, `go vet ./...`, `golangci-lint run ./pkg/rerank/...`.

## 8. Test matrix

| # | Scenario | Layer | Expected |
|---|---|---|---|
| 1 | Happy path, 4 docs, TopN=2 | adapter | 2 results, sorted desc, ranks 1-2, IDs mapped, RequestID set |
| 2 | Happy path, TopN=len(docs) | adapter | full cardinality, complete scores |
| 3 | Equal scores | adapter | tie broken by index then ID (deterministic) |
| 4 | Empty query / no docs / empty ID / dup ID / empty text | adapter | `ErrInvalidRequest`, no ID echoed in message |
| 5 | TopN=0 or TopN>len | adapter | `ErrInvalidRequest` |
| 6 | Model conflict (request vs provider) | adapter | `ErrInvalidRequest` |
| 7 | Response count ≠ TopN | adapter | `ErrInvalidResponse` |
| 8 | Missing index / missing score in a result | adapter | `ErrInvalidResponse` (pointer DTO) |
| 9 | Index out of range / duplicate index | adapter | `ErrInvalidResponse` |
| 10 | NaN/Inf score | adapter | `ErrInvalidResponse` (craft via `httptest` literal body) |
| 11 | Non-2xx status | adapter | `ErrUnavailable`, body not in error |
| 12 | Body > MaxResponseBytes | adapter | `ErrResponseTooLarge` |
| 13 | Payload > MaxRequestBytes | adapter | `ErrRequestTooLarge` (set tiny limit) |
| 14 | Trailing JSON garbage | adapter | `ErrInvalidResponse` |
| 15 | Unknown field in response | adapter | `ErrInvalidResponse` |
| 16 | Server 302 to another host | adapter | redirect rejected → `ErrUnavailable` |
| 17 | Server hangs / connection reset | adapter | redacted `ErrUnavailable`, no URL in message |
| 18 | Request carries `Authorization: Bearer` + `X-Client-Name` | adapter | asserted by mock server |
| 19 | Constructor: missing key, missing model, bad scheme, userinfo URL, negative limits | adapter | `ErrInvalidRequest` |
| 20 | Factory: `type: cohere` happy path from InferenceSettings | factory | provider constructed, model identity correct |
| 21 | Factory: missing `cohere-api-key` | factory | diagnostic names `inference_settings.api.api_keys.cohere-api-key` |
| 22 | Factory: unsupported type | factory | error lists `llamacpp` and `cohere` |
| 23 | Validation: profile diagnostics for cohere | factory | profile-oriented messages |
| 24 | Goja: `gp.reranker(settings)` with cohere config | js | rerank works through factory, no JS change |
| 25 | Live Cohere call | live (opt-in) | skipped without `COHERE_API_KEY` |

## 9. Risks, alternatives, open questions

**Risks.**
- *Cohere API drift*: mitigated by strict decoding (unknown fields fail tests loudly) and the opt-in live test.
- *Key leakage*: the API key lives in memory and one header; the sentinel-error contract plus redacted transport errors keep it out of logs. Reviewers should grep the new package for `apiKey` usages — there should be exactly two (field assignment, header set).
- *Strict decoding vs. `meta` evolution*: if Cohere adds `meta` fields we must add them to the DTO. Acceptable: explicit beats silent.

**Alternatives considered.** Merging PR #169 (rejected, DR-1); allowing redirects (rejected, DR-2); token-faking usage (rejected, DR-3); a shared HTTP strictness package for both adapters (deferred — two call sites do not justify the abstraction; revisit if a third provider lands).

**Open questions.**
- Should `RerankConfig` gain a `cost_per_search` field so profiles can price Cohere rerank? Deferred; DR-3 keeps the door open without a core change.
- Jina/voyage adapters: the same factory pattern applies; not this ticket.

## 10. References

Key files to read, in order:

1. `pkg/rerank/rerank.go` — core types and `Provider` interface
2. `pkg/rerank/validate.go`, `order.go`, `errors.go` — invariants
3. `pkg/rerank/llamacpp/provider.go`, `protocol.go` — reference adapter
4. `pkg/rerank/config/settings.go`, `flags/rerank.yaml` — configuration
5. `pkg/rerank/factory/settings_factory.go` — construction + validation
6. `pkg/security/outbound_url.go` — outbound policy
7. `pkg/js/modules/geppetto/api_reranker.go` — JS surface (no changes needed)
8. `pkg/doc/topics/15-reranking.md` — user-facing docs to extend
9. PR #169: <https://github.com/go-go-golems/geppetto/pull/169> — legacy artifact (protocol source, do not merge)
10. GEPPETTO-RERANKER-001 ticket (`ttmp/2026/07/18/...`) — the primitive's design history
11. Cohere rerank reference: <https://docs.cohere.com/reference/rerank>
