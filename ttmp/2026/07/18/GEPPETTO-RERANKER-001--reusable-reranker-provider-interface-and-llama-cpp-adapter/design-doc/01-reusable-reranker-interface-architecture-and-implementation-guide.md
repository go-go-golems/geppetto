---
Title: Reusable reranker interface architecture and implementation guide
Ticket: GEPPETTO-RERANKER-001
Status: active
Topics:
    - architecture
    - geppetto
    - inference
    - intern-onboarding
    - providers
    - security
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/ttmp/2026/07/15/RAGEVAL-RERANK-001--reranking-stage-for-the-immutable-ttc-rag-laboratory/scripts/02-llamacpp-bge-reranker-probe-results.md
      Note: Real llama.cpp BGE request, response, usage, score, and cardinality evidence
    - Path: repo://pkg/embeddings/embeddings.go
      Note: Nearest reusable model-service Provider interface
    - Path: repo://pkg/embeddings/settings_factory.go
      Note: Settings-backed provider selection and construction pattern
    - Path: repo://pkg/engineprofiles/inference_settings_merge.go
      Note: Profile stack overlay and YAML-map round-trip behavior
    - Path: repo://pkg/js/modules/geppetto/api_embeddings.go
      Note: Profile-resolved synchronous model-service wrapper precedent
    - Path: repo://pkg/js/modules/geppetto/api_session.go
      Note: Cancellable asynchronous Promise handle and owner-thread settlement precedent
    - Path: repo://pkg/js/modules/geppetto/module.go
      Note: Goja module runtime, hidden references, runtime owner bridge, and top-level export installation
    - Path: repo://pkg/security/outbound_url.go
      Note: Outbound URL scheme and local-network policy
    - Path: repo://pkg/steps/ai/settings/http_client.go
      Note: Injected client, timeout, and proxy construction behavior
    - Path: repo://pkg/steps/ai/settings/settings-inference.go
      Note: InferenceSettings primitive sections, API policy, initialization, and cloning
ExternalSources:
    - https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md
    - https://docs.cohere.com/reference/rerank
Summary: Intern-facing architecture and phased implementation guide for a transport-neutral Geppetto reranker API, profile-backed construction, strict llama.cpp adapter, typed synchronous/asynchronous Goja API, usage reporting, outbound security, and downstream RAG integration.
LastUpdated: 2026-07-18T18:20:00-04:00
WhatFor: Define and implement reranking as a reusable Geppetto model-service primitive alongside inference and embeddings rather than embedding provider transport inside a RAG application.
WhenToUse: Read before adding pkg/rerank, changing InferenceSettings for reranking, implementing a llama.cpp rerank client, adding require("geppetto") reranking, or adapting Geppetto reranking into a retrieval system.
---



# Reusable reranker interface architecture and implementation guide

## Executive summary

Geppetto already provides two reusable model-service primitives. `pkg/inference/engine` defines provider-neutral text and multimodal inference over turns. `pkg/embeddings` defines provider-neutral single and batch embedding generation. A retrieval system now needs a third primitive: given one query and an ordered set of documents, score and reorder those documents with a cross-encoder reranker.

The RAG DSL v2 implementation in `rag-evaluation-system` has a native reranking operator and a narrow `ragoperators.Reranker` interface, but it deliberately does not own general provider protocols. A previous prototype proved a llama.cpp `/v1/rerank` adapter against a real BGE reranker. Moving the reusable transport and provider abstraction into Geppetto gives applications one consistent place for model service clients while leaving retrieval identity, evidence lineage, collapse, fusion, hydration, and evaluation in the RAG domain.

This ticket will add `pkg/rerank` with:

- a transport-neutral `Provider` interface;
- typed request, document, result, response, usage, and model records;
- strict request and response validation;
- deterministic identity mapping and ordering;
- a version-specific llama.cpp HTTP adapter;
- bounded requests and responses;
- context cancellation, injected HTTP clients, proxy behavior, outbound URL policy, and redirect checks;
- `InferenceSettings.Rerank` configuration and engine-profile round trips;
- a settings-backed provider factory;
- a typed `require("geppetto")` API with synchronous and cancellable asynchronous execution;
- generated TypeScript declarations, export-surface parity, hard-cut tests, and runnable JavaScript examples;
- unit, conformance, security, cancellation, profile, Goja, and live opt-in tests;
- documentation and a downstream RAG adapter example.

The Goja layer is a typed wrapper over the same profile-resolved Go provider. It does not expose credentials, arbitrary endpoint construction, provider callbacks, retrieval pipelines, vector search, result persistence, or RAG-specific evidence types.

## 1. Problem statement

Reranking differs from generation and embedding in shape, but it has the same infrastructure concerns:

- resolve a model and provider from configuration;
- construct a safe HTTP client;
- validate an outbound endpoint;
- propagate cancellation and deadlines;
- map provider-specific JSON to stable Go records;
- preserve provider usage and model identity;
- distinguish unknown cost from a known zero cost;
- reject malformed or incomplete output;
- keep credentials and response bodies out of errors.

Without a Geppetto primitive, every application must implement these concerns separately. The current RAG ticket could put a llama.cpp client under `pkg/ragproviders`, but that would make a general model-service protocol application-specific and would prevent Pinocchio, search tools, and future Geppetto users from sharing it.

The design must avoid the opposite error: Geppetto must not absorb retrieval semantics. It should not know about chunks, parent units, reciprocal-rank fusion, citations, relevance judgments, or researchctl traces. Geppetto receives caller-owned document IDs and text and returns scores mapped back to those IDs. The caller decides what those documents mean.

## 2. Scope

### 2.1 In scope

- A new `pkg/rerank` Go package.
- A generic `Provider` interface.
- Durable caller-controlled document identity.
- Provider-neutral usage and cost fields.
- Deterministic validation and ordering helpers.
- A llama.cpp adapter for `/v1/rerank`.
- Strict bounded HTTP behavior.
- Reuse of Geppetto's `pkg/security` outbound URL policy.
- Reuse of Geppetto's injected/shared client settings.
- Reranker settings under `InferenceSettings`.
- Engine-profile YAML, clone, stack merge, and round-trip support.
- Factory construction from direct config and resolved inference settings.
- A `require("geppetto")` `reranker(settings)` factory.
- Synchronous `rerank(...)` and cancellable `rerankAsync(...)` methods.
- Precise TypeScript declarations and runtime/declaration parity tests.
- Package documentation and Go/JavaScript examples.
- An adapter sketch for `ragoperators.Reranker`.

### 2.2 Non-goals

- No BM25, vector search, fusion, collapse, hydration, or evaluation.
- No RAG contract or researchctl dependency.
- No automatic text truncation in the first interface.
- No tokenization library in the first package.
- No implicit fallback to another provider or algorithm.
- No persistent ranking cache in the first implementation.
- No JavaScript-supplied provider callback, endpoint, credential, or arbitrary transport options.
- No JavaScript retrieval pipeline, fusion, persistence, or RAG lifecycle API.
- No generic provider plugin registry.
- No Cohere or Jina implementation in the first milestone; their APIs inform interface portability.
- No compatibility wrapper for deleted RAG prototype packages.

## 3. Concepts

### 3.1 Retrieval candidate

A retrieval candidate is an application value selected before reranking. It may be a chunk, document, product, code symbol, or another text-bearing record. Geppetto does not interpret it. The rerank request carries only:

- a caller-controlled unique ID;
- exact text submitted to the provider.

Application metadata and first-stage retrieval scores remain outside the provider request. This reduces accidental disclosure and keeps the interface reusable.

### 3.2 Cross-encoder reranker

A cross encoder scores the query and each candidate together. This is different from embedding retrieval, where query and document vectors are produced independently and compared later. The reranker returns a score per selected candidate. Scores are provider- and model-specific. They may be negative and must not be interpreted as probabilities unless the provider contract explicitly says so.

### 3.3 Input order and provider index

HTTP reranker protocols commonly identify results by the zero-based position of the submitted document. Array position is transport identity, not application identity. The adapter must retain this mapping:

```text
request.Documents[i].ID
    -> provider request documents[i]
    -> provider response result.index == i
    -> response.Results[*].DocumentID
```

A caller must never infer application identity from response order alone.

### 3.4 Top N

`TopN` controls response cardinality. Some providers return only the highest-scoring `TopN` documents. A caller that needs one score per input must request `TopN == len(Documents)`. The package validates the actual response against the requested cardinality.

### 3.5 Usage and cost

Reranking usually consumes input tokens but produces no generated output tokens. Provider responses may report prompt or total tokens. The package therefore needs a rerank-specific usage value rather than pretending reranking is chat inference.

A nil cost means pricing is unknown. A pointer to zero means the provider is explicitly free or local under the selected pricing policy.

## 4. Current Geppetto architecture

### 4.1 Inference engine

`pkg/inference/engine/engine.go` defines:

```go
type Engine interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

This interface is appropriate for conversational or structured generation. It is not appropriate for reranking because a rerank response is a scored relation between one query and many documents, not an assistant turn.

### 4.2 Embedding provider

`pkg/embeddings/embeddings.go` is the closest structural precedent:

```go
type Provider interface {
    GenerateEmbedding(context.Context, string) ([]float32, error)
    GenerateBatchEmbeddings(context.Context, []string) ([][]float32, error)
    GetModel() EmbeddingModel
}
```

`pkg/embeddings/settings_factory.go` adds provider selection, options, direct config, inference-settings construction, and optional cache wrappers. `pkg/embeddings/config/settings.go` adds a Glazed/YAML section. Reranking should follow the same package-level structure while correcting known limitations:

- return a typed response rather than scores without usage;
- inject or construct a bounded HTTP client;
- distinguish invalid config from unsupported provider;
- reject unknown cache/provider behavior rather than silently returning a base provider;
- make outbound URL and redirect policy explicit.

### 4.3 Inference settings and profile registry

`pkg/steps/ai/settings.InferenceSettings` contains API, chat, provider, client, embedding, inference, and model metadata sections. Engine profiles persist and merge this value. The new rerank section belongs here because a profile must be able to describe a reranking provider independently of chat and embeddings.

Profile merge uses YAML-to-map overlay semantics in `pkg/engineprofiles/inference_settings_merge.go`. Adding a typed `Rerank` field automatically participates in recursive overlay, but clone, initialization, YAML round-trip, tests, and documentation still need explicit updates.

### 4.4 HTTP and outbound security

`pkg/steps/ai/settings.EnsureHTTPClient` centralizes timeout, proxy, environment-proxy, and injected-client behavior. `pkg/security.ValidateOutboundURL` rejects unsupported schemes and local-network targets unless explicitly allowed.

The reranker adapter must use both mechanisms. It also needs redirect validation because validating the initial URL does not make an arbitrary redirect safe.

### 4.5 Model metadata and pricing

`pkg/steps/ai/settings.ModelInfo` records context limits and per-million-token costs for chat inference. The first reranker provider can use the profile's model ID and input-token cost, but reranking may eventually need a separate price-per-search-unit model. The core response therefore carries provider-reported usage and an optional computed cost without making `ModelInfo` part of the low-level package API.

## 5. Downstream RAG requirements

The active RAG operator interface currently has this shape:

```go
type Reranker interface {
    Rerank(context.Context, RerankRequest) ([]RerankScore, error)
}

type RerankRequest struct {
    Model, InputTemplate, Truncation, Tokenization string
    Query                                          string
    Candidates                                     []Evidence
    Results                                        int
}

type RerankScore struct {
    ChunkID string
    Score   float64
}
```

The RAG operator owns model-manifest resolution, tokenization/truncation policy matching, candidate windowing, complete-score validation, deterministic sorting, trace construction, and evidence identities. A Geppetto adapter only needs to:

1. map each hydrated evidence item to `rerank.Document{ID: chunkID, Text: sourceText}`;
2. set `TopN` to the number of submitted candidates when complete scores are required;
3. invoke a Geppetto provider;
4. map `rerank.Result.DocumentID` and score to `ragoperators.RerankScore`;
5. add Geppetto usage to the RAG usage/trace model.

This dependency direction is correct:

```text
rag-evaluation-system/pkg/ragproviders
    -> geppetto/pkg/rerank

geppetto/pkg/rerank
    -X-> rag-evaluation-system
```

## 6. External protocol evidence

### 6.1 llama.cpp

The official llama.cpp server documentation exposes reranking routes including `/reranking`, `/rerank`, `/v1/rerank`, and `/v1/reranking`. A reranker model is served with reranking enabled and rank pooling. The endpoint accepts a query, document strings, and `top_n`.

A real local probe on 2026-07-16 used:

```text
server: llama-server 1 (cb295bf59), Darwin arm64
model:  qllama/bge-reranker-v2-m3:q4_k_m
route:  POST /v1/rerank
flags:  --embedding --pooling rank --rerank
```

The response contained:

```json
{
  "model": "qllama/bge-reranker-v2-m3:q4_k_m",
  "object": "list",
  "usage": {"prompt_tokens": 96, "total_tokens": 96},
  "results": [
    {"index": 0, "relevance_score": -3.32784366607666},
    {"index": 1, "relevance_score": -9.837879180908203},
    {"index": 2, "relevance_score": -11.012685775756836}
  ]
}
```

The scores were negative and correctly ordered. Therefore the adapter validates only finiteness, not a `[0,1]` range.

### 6.2 Cohere portability check

Cohere's v2 rerank API also accepts a query, document list, model, and top-N limit and returns ordered results with document indices and relevance scores. This confirms that query + indexed documents + top-N + scores is a portable core. Provider-specific authentication, request options, and usage remain adapter concerns.

## 7. Proposed package layout

```text
pkg/rerank/
  rerank.go                 public request/response/provider/model API
  validate.go               request and response invariants
  order.go                  deterministic ordering and rank assignment
  errors.go                 stable sentinel/wrapped errors
  options.go                shared provider construction options
  config/
    settings.go             RerankConfig
    flags/rerank.yaml       Glazed fields
  llamacpp/
    provider.go             Provider implementation
    protocol.go             strict wire DTOs
    provider_test.go        httptest conformance/security tests
    live_test.go            opt-in real server probe
  settings_factory.go       config and InferenceSettings construction
  settings_factory_test.go
  rerank_test.go

pkg/js/modules/geppetto/
  api_reranker.go           profile-resolved provider wrapper and sync API
  api_reranker_async.go     Promise handle, cancellation, owner-thread settlement
  api_reranker_test.go      decoding, mapping, errors, sync behavior
  api_reranker_async_test.go cancellation, runtime close, Promise settlement
  module.go                 top-level `reranker` export
  module_hardcut_test.go    required public surface
  dts_parity_test.go        generated declaration/runtime export parity
  spec/geppetto.d.ts.tmpl   precise reranker declarations

examples/js/geppetto/hardcut/
  07_reranker_with_registry_profile.js

pkg/doc/topics/
  11-reranking.md           Go and JavaScript concepts, profiles, safety, examples
```

The numbering of the documentation topic should be adjusted to the existing topic index rather than assumed.

## 8. Public API design

### 8.1 Core records

```go
package rerank

type Document struct {
    ID   string `json:"id" yaml:"id"`
    Text string `json:"text" yaml:"text"`
}

type Request struct {
    Model     string     `json:"model" yaml:"model"`
    Query     string     `json:"query" yaml:"query"`
    Documents []Document `json:"documents" yaml:"documents"`
    TopN      int        `json:"top_n" yaml:"top_n"`
}

type Result struct {
    DocumentID string  `json:"document_id" yaml:"document_id"`
    Index      int     `json:"index" yaml:"index"`
    Score      float64 `json:"score" yaml:"score"`
    Rank       int     `json:"rank" yaml:"rank"`
}

type Usage struct {
    InputTokens int `json:"input_tokens,omitempty" yaml:"input_tokens,omitempty"`
    TotalTokens int `json:"total_tokens,omitempty" yaml:"total_tokens,omitempty"`
}

type Response struct {
    Provider   string   `json:"provider" yaml:"provider"`
    Model      string   `json:"model" yaml:"model"`
    Results    []Result `json:"results" yaml:"results"`
    Usage      *Usage   `json:"usage,omitempty" yaml:"usage,omitempty"`
    Cost       *float64 `json:"cost,omitempty" yaml:"cost,omitempty"`
    RequestID  string   `json:"request_id,omitempty" yaml:"request_id,omitempty"`
    DurationMs *int64   `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty"`
}

type Model struct {
    Provider string `json:"provider" yaml:"provider"`
    Name     string `json:"name" yaml:"name"`
}
```

`Request.Model` is explicit even when a provider instance has a configured model. Construction should either require it to equal the provider model or allow it to be empty and fill it from `Model()`. The recommended first behavior is strict equality when non-empty and provider default when empty.

### 8.2 Provider interface

```go
type Provider interface {
    Rerank(context.Context, Request) (Response, error)
    Model() Model
}
```

The response is richer than `[]Result` because usage, model, request ID, duration, and cost are first-class observations.

### 8.3 Request invariants

`ValidateRequest` rejects:

- empty query after trimming;
- zero documents;
- empty or duplicate document IDs;
- empty document text;
- `TopN < 1`;
- `TopN > len(Documents)`;
- a request model that conflicts with the configured provider model;
- encoded requests larger than the provider limit.

Do not silently default `TopN`. Explicit cardinality is important to callers that require complete scores.

### 8.4 Response invariants

The provider adapter rejects:

- response count not equal to `TopN`;
- missing index or score;
- index outside the submitted document array;
- duplicate response indices;
- NaN or infinite score;
- model mismatch when the response declares a model;
- trailing JSON values;
- response body larger than the configured limit.

After validation, results are sorted:

1. score descending;
2. input index ascending for equal scores;
3. document ID ascending as a final deterministic tie break.

Ranks are assigned from one after sorting.

### 8.5 Error model

Define sentinel categories so callers can classify failures without parsing complete strings:

```go
var (
    ErrInvalidRequest  = errors.New("invalid rerank request")
    ErrInvalidResponse = errors.New("invalid rerank response")
    ErrUnavailable     = errors.New("rerank provider unavailable")
    ErrRequestTooLarge = errors.New("rerank request too large")
    ErrResponseTooLarge = errors.New("rerank response too large")
)
```

Wrap these with safe context. Never include authorization headers, endpoint userinfo, document text, query text, or an unbounded provider body.

### 8.6 Goja API

The module adds one top-level factory consistent with `gp.embeddings(settings)`:

```javascript
const gp = require("geppetto");
const settings = gp.inferenceProfiles
  .load("~/.config/pinocchio/profiles.yaml")
  .resolve("bge-reranker-local");

const reranker = gp.reranker(settings);
const response = reranker.rerank(
  "How does TTC calculate a payroll adjustment?",
  [
    {id: "chunk-001", text: "A payroll adjustment corrects wages or deductions."},
    {id: "chunk-002", text: "Cypress trees tolerate dry conditions."}
  ],
  {topN: 2}
);

for (const result of response.results) {
  console.log(result.rank, result.documentId, result.score);
}
```

The settings argument must be the hidden-reference `InferenceSettings` wrapper returned by `inferenceProfiles.resolve`. JavaScript cannot supply a base URL, API key, HTTP client, local-network exception, or provider implementation directly. Those capabilities remain in host/profile configuration.

#### JavaScript surface

```typescript
export interface RerankDocument {
    id: string;
    text: string;
}

export interface RerankOptions {
    topN: number;
    model?: string;
}

export interface RerankResult {
    documentId: string;
    index: number;
    score: number;
    rank: number;
}

export interface RerankUsage {
    inputTokens?: number;
    totalTokens?: number;
}

export interface RerankResponse {
    provider: string;
    model: string;
    results: RerankResult[];
    usage?: RerankUsage;
    cost?: number;
    requestId?: string;
    durationMs?: number;
}

export interface RerankAsyncHandle {
    promise: Promise<RerankResponse>;
    cancel(): void;
    close(): void;
}

export interface RerankerProvider {
    rerank(query: string, documents: RerankDocument[], options: RerankOptions): RerankResponse;
    rerankAsync(query: string, documents: RerankDocument[], options: RerankOptions): RerankAsyncHandle;
    model(): {provider: string; name: string};
}

export function reranker(settings: InferenceSettings): RerankerProvider;
```

`topN` remains required in JavaScript for the same reason it is required in Go: complete-score and partial-score requests must be distinguishable. Documents must be `{id, text}` objects; accepting plain strings would discard durable identity.

#### Wrapper implementation

`api_reranker.go` follows `api_embeddings.go`:

```go
type rerankerRef struct {
    api      *moduleRuntime
    provider rerank.Provider
}

func (m *moduleRuntime) rerankerBuilder(call goja.FunctionCall) goja.Value {
    settingsRef, err := m.requireInferenceSettingsRef(call.Argument(0))
    if err != nil {
        panic(m.vm.NewGoError(err))
    }
    factory, err := rerank.NewSettingsFactoryFromInferenceSettings(settingsRef.settings)
    if err != nil {
        panic(m.vm.NewGoError(err))
    }
    provider, err := factory.NewProvider()
    if err != nil {
        panic(m.vm.NewGoError(err))
    }
    return m.newRerankerObject(&rerankerRef{api: m, provider: provider})
}
```

The provider wrapper is stored under `hiddenRefKey`; its property remains non-enumerable, non-writable, and non-configurable.

`rerank(...)` strictly decodes JavaScript values into the public Go `rerank.Request`, invokes the provider with the runtime lifetime context, and returns a plain camelCase response object. It rejects unknown option keys, sparse arrays, non-object documents, duplicate IDs, non-string text, non-integral `topN`, and values outside JavaScript's safe integer range.

#### Asynchronous execution

A network request can block the Goja owner thread. The module therefore also exposes `rerankAsync(...)`, modeled on `session.runAsync`:

```javascript
const handle = reranker.rerankAsync(query, documents, {topN: documents.length});
try {
  const response = await handle.promise;
} finally {
  handle.close();
}
```

Implementation flow:

```text
owner thread: decode and deep-copy JS request
  -> create context.WithCancel(runtime lifetime)
  -> create Promise and {promise,cancel,close} handle
  -> goroutine: provider.Rerank(ctx, copiedRequest)
  -> runtime bridge Post: convert response and resolve Promise
  -> on error: create safe GoError and reject Promise on owner thread
  -> cancel/close/runtime shutdown: cancel provider context
```

No `goja.Value`, object, callable, Promise resolver, or VM method may be accessed from the provider goroutine. Request data is converted to ordinary Go values before launching. Promise settlement and response conversion occur through `moduleRuntime.postOnOwner` or the existing runtime bridge.

The synchronous method remains useful for bounded command-style scripts and matches the existing embeddings API. Event-loop applications should use `rerankAsync`.

#### Goja tests

Required coverage includes:

- top-level `reranker` export and hard-cut surface;
- generated DTS top-level parity;
- profile-resolved construction;
- model metadata;
- exact request decoding and caller ID preservation;
- malformed documents/options and missing settings;
- safe error messages without query/document text;
- synchronous result conversion;
- Promise resolution and rejection on the owner thread;
- cancellation reaching the provider;
- runtime lifetime cancellation;
- repeated `cancel`/`close` idempotence;
- no goroutine touching Goja values;
- runnable hard-cut example.

The existing DTS parity test checks top-level exports but not every interface method. Add explicit runtime method-surface assertions for `RerankerProvider` and `RerankAsyncHandle`, or extend parity tooling deliberately.

## 9. llama.cpp adapter design

### 9.1 Options

```go
package llamacpp

type Options struct {
    BaseURL          string
    Model            string
    HTTPClient       *http.Client
    OutboundURL      security.OutboundURLOptions
    MaxRequestBytes  int64
    MaxResponseBytes int64
    CostPerMTokens   *float64
}

func New(options Options) (*Provider, error)
```

Defaults may exist for byte limits, but not for model or base URL. A generic library must not silently point at localhost.

### 9.2 Endpoint construction

Parse and normalize `BaseURL` at construction. Reject:

- missing scheme or host;
- non-HTTP(S) scheme;
- URL userinfo;
- query or fragment;
- path traversal or ambiguous path joining.

Construct the endpoint with `url.JoinPath(baseURL, "v1", "rerank")`, validate the final URL, and retain a redacted origin for diagnostics.

### 9.3 Request DTO

```go
type request struct {
    Model     string   `json:"model"`
    Query     string   `json:"query"`
    Documents []string `json:"documents"`
    TopN      int      `json:"top_n"`
}
```

Caller IDs never enter the provider payload. The adapter retains a local index-to-ID table.

### 9.4 Response DTO

```go
type response struct {
    Model   string `json:"model,omitempty"`
    Object  string `json:"object,omitempty"`
    Usage   *struct {
        PromptTokens int `json:"prompt_tokens"`
        TotalTokens  int `json:"total_tokens"`
    } `json:"usage,omitempty"`
    Results []struct {
        Index          *int     `json:"index"`
        RelevanceScore *float64 `json:"relevance_score"`
    } `json:"results"`
}
```

Pointer fields distinguish a missing zero from a valid zero.

### 9.5 Transport pseudocode

```go
func (p *Provider) Rerank(ctx context.Context, in rerank.Request) (rerank.Response, error) {
    started := time.Now()
    if err := rerank.ValidateRequest(in, p.Model()); err != nil {
        return Response{}, err
    }

    payload := encodeBounded(in)
    req := http.NewRequestWithContext(ctx, POST, p.endpoint, payload)
    req.Header.Set("Content-Type", "application/json")

    httpResponse, err := p.client.Do(req)
    if err != nil {
        return Response{}, redactTransportError(err)
    }
    defer httpResponse.Body.Close()

    if httpResponse.StatusCode/100 != 2 {
        drainBounded(httpResponse.Body)
        return Response{}, safeStatusError(httpResponse.StatusCode)
    }

    raw := readAtMost(httpResponse.Body, p.maxResponseBytes+1)
    if len(raw) > p.maxResponseBytes {
        return Response{}, ErrResponseTooLarge
    }
    wire := decodeStrict(raw)
    results := validateMapAndSort(in.Documents, in.TopN, wire.Results)

    usage := mapUsage(wire.Usage)
    return Response{
        Provider: "llama.cpp",
        Model: p.model,
        Results: results,
        Usage: usage,
        Cost: computeInputCost(usage, p.costPerMTokens),
        DurationMs: millisecondsSince(started),
    }, nil
}
```

### 9.6 Redirect policy

The provider must not rely on `http.Client`'s default redirect behavior after validating only the initial URL. The implementation should clone or wrap the client with a `CheckRedirect` function that validates every redirect target under the same outbound options. The caller's injected transport and timeout must remain intact.

An acceptable alternative for the first llama.cpp adapter is to reject every redirect. Local model endpoints should not redirect, and rejection is easier to reason about.

## 10. Settings and profile integration

### 10.1 RerankConfig

```go
package config

type RerankConfig struct {
    Type             string `yaml:"type,omitempty" glazed:"rerank-type"`
    Engine           string `yaml:"engine,omitempty" glazed:"rerank-engine"`
    MaxRequestBytes  int64  `yaml:"max_request_bytes,omitempty" glazed:"rerank-max-request-bytes"`
    MaxResponseBytes int64  `yaml:"max_response_bytes,omitempty" glazed:"rerank-max-response-bytes"`
}
```

Endpoint and credential values should use the existing `InferenceSettings.API` maps:

```yaml
inference_settings:
  api:
    base_urls:
      rerank-base-url: http://127.0.0.1:18012
    allow_http:
      rerank: true
    allow_local_networks:
      rerank: true
  client:
    timeout_second: 60
    proxy_from_environment: false
  rerank:
    type: llamacpp
    engine: qllama/bge-reranker-v2-m3:q4_k_m
    max_request_bytes: 2097152
    max_response_bytes: 1048576
  model_info:
    id: sha256:a4a2faf8f2d866cfa528c975bd0018095e8aa5de0d0a5279193524f0fa26956a
    name: qllama/bge-reranker-v2-m3:q4_k_m
    context_window: 8192
    cost:
      input: 0
      output: 0
```

The local endpoint is an operational profile value. Scientific callers should separately record exact model and adapter identities in their own manifests.

### 10.2 InferenceSettings extension

```go
Rerank *rerankconfig.RerankConfig `yaml:"rerank,omitempty" glazed:"rerank"`
```

Update:

- `NewInferenceSettings`;
- `InferenceSettings.Clone`;
- parsed Glazed sections;
- YAML normalization where needed;
- engine-profile clone and stack tests;
- profile print/summary output;
- documentation.

Do not make rerank config mandatory for chat or embedding profiles.

### 10.3 Factory

```go
type ProviderFactory interface {
    NewProvider(options ...ProviderOption) (Provider, error)
    SupportedProviders() []string
}

func NewSettingsFactory(config *config.RerankConfig, api *settings.APISettings, client *settings.ClientSettings, modelInfo *settings.ModelInfo) *SettingsFactory
func NewSettingsFactoryFromInferenceSettings(*settings.InferenceSettings) (*SettingsFactory, error)
```

The first factory supports only `llamacpp`. Unknown types fail explicitly.

Construction flow:

```text
resolved engine profile
  -> final InferenceSettings
  -> validate RerankConfig + API endpoint policy
  -> settings.EnsureHTTPClient
  -> llamacpp.New
  -> rerank.Provider
```

## 11. Downstream RAG adapter

The adapter belongs in the RAG repository:

```go
type GeppettoReranker struct {
    provider rerank.Provider
}

func (g *GeppettoReranker) Rerank(
    ctx context.Context,
    request ragoperators.RerankRequest,
) ([]ragoperators.RerankScore, error) {
    documents := make([]rerank.Document, len(request.Candidates))
    for i, candidate := range request.Candidates {
        documents[i] = rerank.Document{
            ID: candidate.Chunk.Record.ID,
            Text: candidate.Chunk.Text,
        }
    }

    response, err := g.provider.Rerank(ctx, rerank.Request{
        Model: request.Model,
        Query: request.Query,
        Documents: documents,
        TopN: len(documents),
    })
    if err != nil {
        return nil, err
    }

    scores := make([]ragoperators.RerankScore, len(response.Results))
    for i, result := range response.Results {
        scores[i] = ragoperators.RerankScore{
            ChunkID: result.DocumentID,
            Score: result.Score,
        }
    }
    return scores, nil
}
```

The RAG adapter must separately map `response.Usage` into `rag-query-trace/v2`. If the current RAG interface cannot carry usage, RESEARCHCTL-015 should evolve that interface deliberately rather than dropping telemetry.

## 12. Security design

### 12.1 Protected values

- query text;
- document text;
- bearer/API credentials for future hosted providers;
- private endpoint topology;
- provider response bodies;
- proxy credentials;
- caller-controlled IDs.

### 12.2 Required controls

- Validate the final endpoint with `security.ValidateOutboundURL`.
- Reject URL userinfo.
- Reject or revalidate redirects.
- Honor explicit `AllowHTTP` and `AllowLocalNetworks`; default deny.
- Use the injected/configured HTTP client and context.
- Bound encoded request bytes before creating the request.
- Bound response bytes before decoding.
- Reject trailing JSON.
- Never include query or documents in errors or logs.
- Never include non-2xx response bodies in errors.
- Redact proxy URLs through existing settings helpers.
- Do not log request headers.
- Use safe model and provider identifiers in diagnostics.

### 12.3 DNS limitation

`ValidateOutboundURL` enforces literal IP and hostname policy but does not resolve DNS. A hostname can resolve to a private address after lexical validation. This is an existing broader outbound-security limitation. The first local llama.cpp profile explicitly opts into local networks, so it is not made less safe by DNS resolution. A future hosted reranker adapter should use a transport-level dial policy if protection against DNS rebinding is required.

## 13. Testing strategy

### 13.1 Core API tests

- empty query;
- empty document list;
- empty and duplicate document IDs;
- empty text;
- invalid TopN;
- model mismatch;
- deterministic score ordering;
- equal-score input-order tie break;
- JSON/YAML round trips.

### 13.2 llama.cpp conformance tests

Use `httptest.Server` and an injected client:

- exact method, route, content type, model, query, documents, and top-N;
- index-to-document-ID mapping;
- valid negative and zero scores;
- response cardinality;
- missing index or score;
- duplicate and out-of-range index;
- NaN/infinite handling through crafted JSON where possible;
- response model mismatch;
- usage mapping;
- non-2xx response without body leakage;
- oversized request and response;
- trailing JSON;
- context cancellation;
- timeout;
- redirect rejection;
- local HTTP denied by default and allowed only by explicit options.

### 13.3 Factory and profile tests

- direct `RerankConfig` construction;
- `InferenceSettings` construction;
- YAML profile round trip;
- profile stack inheritance and overlay;
- deep clone independence;
- missing type, model, endpoint, and invalid limits;
- unsupported provider type;
- client timeout and proxy propagation;
- local-network opt-in propagation;
- zero versus unknown model cost.

### 13.4 Goja API tests

- `gp.reranker(settings)` accepts only a registry-resolved settings wrapper;
- sync and async methods produce equivalent response values;
- strict JS argument decoding rejects malformed values before provider calls;
- hidden Go references are not enumerable or writable;
- async work settles only on the runtime owner thread;
- cancellation, runtime shutdown, rejection, and repeated close are race-safe;
- module hard-cut and generated DTS checks include `reranker`;
- the generated declaration template and checked-in `pkg/doc/types/geppetto.d.ts` remain synchronized.

### 13.5 Live opt-in test

```bash
GEPPETTO_LIVE_RERANK=1 \
GEPPETTO_RERANK_BASE_URL=http://127.0.0.1:18012 \
GEPPETTO_RERANK_MODEL=qllama/bge-reranker-v2-m3:q4_k_m \
go test ./pkg/rerank/llamacpp -run TestLive -v -count=1
```

The test skips unless the opt-in variable is exactly enabled. It never falls back to a fixture and never starts external services itself.

### 13.6 Full validation

```bash
go test ./pkg/rerank/... ./pkg/steps/ai/settings ./pkg/engineprofiles ./pkg/js/modules/geppetto -count=1
go test -race ./pkg/rerank/... ./pkg/engineprofiles ./pkg/js/modules/geppetto -count=1
go vet ./...
GOWORK=off go test ./... -count=1
GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run ./...
govulncheck ./...
```

## 14. Implementation phases

### Phase 0: freeze the public API and protocol fixture

1. Review the request, response, usage, model, and error records.
2. Save a sanitized llama.cpp request/response fixture from the verified server revision.
3. Confirm current official endpoint aliases and flags.
4. Decide package name `pkg/rerank` and YAML key `rerank`.
5. Confirm RAG adapter usage requirements.

**Exit gate:** API review accepted; no field relies on RAG types or a single provider's array order.

### Phase 1: implement core package

1. Add public types and `Provider`.
2. Add request validation.
3. Add response mapping/order helpers.
4. Add sentinel errors.
5. Add package documentation and core tests.

**Exit gate:** a fake provider can return deterministic, ID-mapped, provider-neutral responses with usage and cost.

### Phase 2: implement llama.cpp provider

1. Add options and constructor validation.
2. Add bounded request encoding.
3. Add safe endpoint and redirect policy.
4. Add strict response decoding.
5. Add index mapping, score validation, ordering, usage, duration, and cost.
6. Add conformance and security tests.

**Exit gate:** all malformed protocol cases fail safely; the sanitized real response passes.

### Phase 3: integrate settings and profiles

1. Add `RerankConfig` and Glazed fields.
2. Add `InferenceSettings.Rerank` initialization and clone support.
3. Add settings factory and supported-provider list.
4. Add API/client/outbound-option resolution.
5. Add engine-profile YAML, stack, clone, and round-trip tests.
6. Update profile summary output.

**Exit gate:** a rerank-only YAML profile resolves through the normal engine-profile chain and constructs a provider.

### Phase 4: add the Goja API

1. Add the top-level `reranker(settings)` export and hidden provider wrapper.
2. Add strict query/document/options decoding and camelCase response conversion.
3. Add synchronous `rerank(...)` using the runtime lifetime context.
4. Add cancellable `rerankAsync(...)` with Promise settlement on the runtime owner thread.
5. Update the generated DTS template and regenerate `pkg/doc/types/geppetto.d.ts`.
6. Extend hard-cut, export-surface, method-surface, cancellation, race, and example tests.

**Exit gate:** registry-resolved JavaScript can execute sync and async reranking without receiving endpoint or credential capability; DTS and runtime surfaces agree.

### Phase 5: live qualification and downstream adapter proof

1. Start the version-frozen llama.cpp service outside the test process.
2. Run the Go and JavaScript opt-in live tests.
3. Record exact server/model/request evidence.
4. Implement the thin RAG adapter in the RAG ticket.
5. Prove complete score mapping and usage propagation.

**Exit gate:** a real RAG v2 reranking operator uses `geppetto/pkg/rerank` and emits a complete native reranking trace.

### Phase 6: documentation and hardening

1. Add the reranking topic guide.
2. Add Go, synchronous JavaScript, asynchronous JavaScript, and profile examples.
3. Run full unit, race, lint, vet, vulnerability, and module-isolation checks.
4. Check dependency direction, runtime-owner correctness, and secret/body absence.
5. Update ticket diary, tasks, changelog, relations, and publication.

**Exit gate:** documentation and validation pass; Geppetto contains no RAG import and the generated JavaScript declarations match the runtime surface.

## 15. Decision records

### Decision: create `pkg/rerank` rather than use `engine.Engine`

- **Context:** Reranking takes one query and many documents and returns scored relations, not a conversational turn.
- **Options considered:** encode reranking as a prompt through `Engine`; add methods to `Engine`; create a separate primitive.
- **Decision:** Add a separate `pkg/rerank.Provider` interface.
- **Rationale:** It models cardinality, identity mapping, score validation, and usage directly without provider-specific prompt emulation.
- **Consequences:** Geppetto has a third model-service primitive with its own settings and factory.
- **Status:** proposed.

### Decision: require caller-owned document IDs

- **Context:** Provider APIs identify documents by array index, while applications need durable identity.
- **Options considered:** expose only indices; accept plain strings; require ID/text records.
- **Decision:** Require unique non-empty document IDs and return both ID and transport index.
- **Rationale:** It prevents response order or index from becoming application identity and supports exact downstream mapping.
- **Consequences:** Simple callers must create IDs, but correctness is explicit.
- **Status:** proposed.

### Decision: keep scores provider-relative

- **Context:** llama.cpp returned negative scores while Cohere documents normalized scores.
- **Options considered:** force `[0,1]`; normalize scores; preserve finite provider values.
- **Decision:** Accept any finite score and sort descending.
- **Rationale:** Normalization would invent semantics not shared by providers.
- **Consequences:** Callers compare order within a request and must not compare raw scores across models without calibration.
- **Status:** proposed.

### Decision: require explicit TopN

- **Context:** TopN controls whether the caller receives complete or partial scores.
- **Options considered:** default to all documents; allow zero; require explicit `1..N`.
- **Decision:** Require explicit valid TopN.
- **Rationale:** Complete-score callers such as RAG can prove they requested all candidates.
- **Consequences:** The API is slightly more verbose and less ambiguous.
- **Status:** proposed.

### Decision: return a rich Response

- **Context:** Downstream scientific runs need usage, model, cost, duration, and request IDs in addition to scores.
- **Options considered:** return `[]Result`; use callback events; return a response record.
- **Decision:** Return `Response` with optional observation fields.
- **Rationale:** Reranking is non-streaming and one typed response preserves all provider observations.
- **Consequences:** Adapters must map available telemetry and leave unavailable values nil rather than fabricate zero.
- **Status:** proposed.

### Decision: use existing API and client settings

- **Context:** Geppetto already models base URLs, local HTTP/network opt-in, proxies, timeouts, and injected clients.
- **Options considered:** create standalone rerank transport settings; reuse existing settings; hardcode a client.
- **Decision:** Put semantic rerank fields in `RerankConfig` and resolve endpoint/security/client behavior from existing settings.
- **Rationale:** It preserves profile composition and established transport policy.
- **Consequences:** `InferenceSettings` gains an optional rerank section and requires clone/profile tests.
- **Status:** proposed.

### Decision: reject redirects initially

- **Context:** Default redirects can escape the validated endpoint and leak query/document text.
- **Options considered:** default redirect behavior; revalidate each redirect; reject redirects.
- **Decision:** Reject redirects in the first llama.cpp adapter.
- **Rationale:** Local llama.cpp should not redirect, and rejection gives the smallest auditable behavior.
- **Consequences:** Proxied services requiring redirects need an explicit future policy.
- **Status:** proposed.

### Decision: no automatic truncation

- **Context:** Truncation changes model input and is part of application/model semantics.
- **Options considered:** silently truncate by bytes; tokenize in Geppetto; require caller-supplied exact text.
- **Decision:** The first provider submits exact caller text and fails on limits; caller policy remains explicit.
- **Rationale:** A generic transport package should not hide scientific input changes.
- **Consequences:** RAG remains responsible for manifest-matched tokenization and truncation before invocation.
- **Status:** proposed.

### Decision: expose profile-resolved Goja sync and async APIs

- **Context:** Geppetto's JavaScript module already exposes profile-resolved engines and embeddings; reranking must be available to the same scripting environment without exposing provider construction or unnecessarily blocking event-loop applications.
- **Options considered:** no JavaScript API; synchronous only; Promise only; synchronous plus cancellable asynchronous methods.
- **Decision:** Export `reranker(settings)` with `rerank`, `rerankAsync`, and `model`.
- **Rationale:** Synchronous execution matches embeddings and bounded scripts; the async handle matches established session cancellation and runtime-owner patterns.
- **Consequences:** The module needs strict decoders, owner-thread Promise settlement, cancellation/race tests, DTS updates, hard-cut updates, and examples.
- **Status:** proposed.

### Decision: JavaScript cannot construct transport capability

- **Context:** Allowing JavaScript to supply an endpoint, credential, HTTP client, or callback would bypass profile security and make scripts responsible for secret handling.
- **Options considered:** accept inline provider options; accept arbitrary callbacks; require a resolved `InferenceSettings` wrapper.
- **Decision:** `reranker` accepts only registry-resolved settings and builds providers through the Go settings factory.
- **Rationale:** It matches the hard-cut wrapper-first API and keeps credentials and network policy in the host/profile layer.
- **Consequences:** Hosts must configure profile access; JavaScript may choose only among profiles it is authorized to resolve.
- **Status:** proposed.

### Decision: implement llama.cpp first but keep the core portable

- **Context:** A real llama.cpp server and BGE model have already been probed locally.
- **Options considered:** Cohere first; llama.cpp only API; portable core plus llama.cpp adapter.
- **Decision:** Build a portable core and one strict llama.cpp adapter.
- **Rationale:** It gives immediate local value while the Cohere shape validates the abstraction.
- **Consequences:** Hosted authentication and provider-specific document formats remain future adapters.
- **Status:** proposed.

## 16. Alternatives considered

### Keep the adapter in the RAG repository

This is the shortest implementation for one application, but duplicates Geppetto's endpoint, client, profile, and provider responsibilities. It is rejected for the reusable transport layer. RAG still owns a thin domain adapter.

### Use embeddings similarity as reranking

Embedding similarity does not jointly encode query and document and is already a separate retrieval stage. Calling it reranking would misstate the model behavior and prevent cross-encoder qualification.

### Prompt a chat model to rank documents

A generation model can emit a ranking, but cardinality, score semantics, cost, determinism, and malformed output differ from a reranking endpoint. That should be a separately named provider/algorithm if ever implemented, never an implicit fallback.

### Return only ordered documents

Dropping scores and original transport indices prevents downstream traces and validation from explaining changes. The rich result keeps both order and evidence.

### Put RerankConfig only in application YAML

This avoids changing `InferenceSettings` but makes profile registries unable to compose reranker endpoints, models, and clients. Geppetto primitives should be profile-resolvable consistently.

## 17. Intern implementation workflow

A new intern should proceed in this order:

1. Read `pkg/embeddings/embeddings.go` and `settings_factory.go` to understand the nearest provider pattern.
2. Read `pkg/security/outbound_url.go` and its tests.
3. Read `pkg/steps/ai/settings/settings-inference.go`, `settings-client.go`, and `http_client.go`.
4. Read `pkg/engineprofiles/inference_settings_merge.go` and the embedding stack tests.
5. Read the saved llama.cpp probe evidence before writing wire DTOs.
6. Implement core records and tests without importing HTTP.
7. Implement the llama.cpp adapter entirely against `httptest.Server`.
8. Add settings/profile integration only after the provider constructor is stable.
9. Read `module.go`, `api_embeddings.go`, `api_session.go` async settlement, the DTS template, and hard-cut tests.
10. Implement and test the synchronous Goja wrapper before adding async execution.
11. Add async execution with no Goja access from provider goroutines.
12. Run the live opt-in test after every conformance, security, and Goja race test passes.
13. Implement the RAG adapter in the separate RAG ticket.

Do not begin with a live server. Conformance tests must define behavior before operational testing.

## 18. Review checklist

### API

- [ ] Core types import no RAG package.
- [ ] Document IDs are required and unique.
- [ ] TopN is explicit.
- [ ] Scores accept every finite value.
- [ ] Response carries usage, cost, model, duration, and request ID where available.
- [ ] Nil and zero cost remain distinguishable.

### Protocol

- [ ] Provider indices map to caller IDs.
- [ ] Missing, duplicate, and out-of-range indices fail.
- [ ] Cardinality equals TopN.
- [ ] Equal-score ordering is deterministic.
- [ ] Request and response bodies are bounded.
- [ ] Trailing JSON fails.

### Security

- [ ] Endpoint scheme, host, userinfo, query, and fragment are validated.
- [ ] HTTP and local networks are denied by default.
- [ ] Redirects are rejected.
- [ ] Injected client timeout and proxy behavior are retained.
- [ ] Query/document text and response bodies never enter errors.
- [ ] Cancellation terminates active requests.

### Profiles

- [ ] RerankConfig clones deeply.
- [ ] YAML round trip works.
- [ ] Stack overlay works.
- [ ] Missing config fails explicitly.
- [ ] Chat and embedding-only profiles remain valid.

### Goja

- [ ] `reranker` accepts only resolved settings wrappers.
- [ ] JavaScript cannot provide endpoints, credentials, clients, or callbacks.
- [ ] Documents require explicit IDs and text.
- [ ] Sync and async values are structurally equivalent.
- [ ] Async provider work touches no Goja value off the owner thread.
- [ ] Cancellation and runtime close are idempotent and race-safe.
- [ ] DTS, hard-cut, and runtime method surfaces agree.

### Downstream

- [ ] RAG imports Geppetto, never the reverse.
- [ ] RAG requests complete scores where required.
- [ ] Usage reaches RAG traces.
- [ ] No provider fallback exists.

## 19. Risks and open questions

### Reranker usage shape

llama.cpp reports prompt and total tokens. Hosted providers may report search units rather than token counts. The first `Usage` record supports tokens; a future version may need typed provider units. Do not overload token fields with search units.

### Model metadata ownership

`InferenceSettings.ModelInfo` currently describes one selected model. A profile containing chat, embedding, and rerank sections could make one model-info record ambiguous. The first rerank-only profile avoids this. A future profile schema may need capability-specific model metadata.

### HTTP client redirect mutation

An injected `*http.Client` belongs to the caller. The adapter must not mutate its `CheckRedirect` field in place. Clone the client value while retaining its transport, jar, timeout, and other behavior.

### Strict response evolution

llama.cpp marks portions of its HTTP surface as evolving. A strict decoder improves qualification but can reject harmless new fields. Model all currently documented fields and version the adapter when the server protocol changes. Keep sanitized fixtures tied to the tested server revision.

### Request text size versus token limits

Byte limits protect memory and transport but do not prove model context safety. The application must apply exact tokenization/truncation policy before the call. The provider should report clear safe errors for server context-limit failures.

### Cost semantics

Local llama.cpp cost is known zero only under an explicit policy that excludes hardware and energy accounting. Otherwise cost should remain nil. Documentation must not treat nil as zero.

## 20. Deliverables

1. `pkg/rerank` public API and validation.
2. `pkg/rerank/llamacpp` strict provider.
3. `pkg/rerank/config` and Glazed fields.
4. Settings-backed provider factory.
5. `InferenceSettings.Rerank` and profile integration.
6. `require("geppetto").reranker(settings)` with sync and cancellable async methods.
7. Generated TypeScript declarations, hard-cut/export/method parity, and JavaScript examples.
8. Core, conformance, security, cancellation, race, profile, Goja, and live tests.
9. Reranking documentation and examples.
10. Sanitized protocol fixtures and live qualification record.
11. Downstream RAG adapter proof in RESEARCHCTL-015.
12. Ticket tasks, diary, changelog, validation, and reMarkable publication.

## 21. References

### Geppetto

- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/embeddings.go:1-22` — nearest provider interface.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/settings_factory.go:1-246` — direct/settings-backed provider construction and provider selection.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/config/settings.go:1-75` — Glazed/YAML primitive configuration pattern.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/inference/engine/engine.go:1-16` — generation engine boundary.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/steps/ai/settings/settings-inference.go:45-103,345-380` — API settings, optional primitive sections, construction, and cloning.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/steps/ai/settings/settings-client.go:1-82` — HTTP client configuration.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/steps/ai/settings/http_client.go:1-151` — injected/default/custom HTTP client and proxy behavior.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/security/outbound_url.go:1-61` — outbound URL policy.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/engineprofiles/inference_settings_merge.go:1-111` — profile overlay behavior.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/steps/ai/settings/model_info.go:1-139` — limits, pricing, and nil-versus-zero cost semantics.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/js/modules/geppetto/module.go:25-214` — module options, runtime owner/bridge, hidden references, and top-level exports.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/js/modules/geppetto/api_embeddings.go:1-66` — profile-resolved model-service wrapper precedent.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/js/modules/geppetto/api_session.go:682-746` — cancellable Promise handle and owner-thread settlement precedent.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/js/modules/geppetto/dts_parity_test.go:1-194` — generated declaration/runtime export parity.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl:1-280` — declaration source template.
- `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/js/modules/geppetto/module_hardcut_test.go:65-108` — required public surface and runnable hard-cut examples.

### RAG integration evidence

- `/home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/pkg/ragoperators/types.go:129-144` — active RAG reranker boundary.
- `/home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/pkg/ragoperators/rank.go:235-310` — active native reranking semantics and trace.
- `/home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/ttmp/2026/07/15/RAGEVAL-RERANK-001--reranking-stage-for-the-immutable-ttc-rag-laboratory/scripts/02-llamacpp-bge-reranker-probe-results.md:1-67` — real llama.cpp protocol evidence.
- Git object `bf2b567^:pkg/raglab/reranker_llamacpp.go` in `rag-evaluation-system` — deleted prototype adapter used only as extraction evidence.

### External

- [llama.cpp HTTP server documentation](https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md) — reranking route, request fields, and serving requirements.
- [Cohere Rerank v2 API](https://docs.cohere.com/reference/rerank) — portability comparison for query, documents, top-N, indexed results, and scores.
