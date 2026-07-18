---
Title: Understanding and Using the Rerank Package
Slug: geppetto-rerank-package
Short: A guide to the rerank package in Geppetto, covering cross-encoder reranking, the llama.cpp adapter, profile integration, and the JavaScript API.
Topics:
- geppetto
- rerank
- retrieval
- ai
- tutorial
- providers
- security
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Understanding and Using the Rerank Package

## What is Reranking?

Reranking is a retrieval stage that takes one query and an ordered set of candidate documents and scores them with a cross-encoder model. Unlike embedding retrieval (where query and document vectors are produced independently), a cross-encoder scores the query and each document *together*, producing a relevance score per document. This typically yields higher-quality ordering than first-stage vector or BM25 retrieval.

Reranking is Geppetto's third model-service primitive, alongside inference (`pkg/inference/engine`) and embeddings (`pkg/embeddings`).

**Use cases:**
- **RAG** — reorder retrieved chunks before generation
- **Search** — improve first-stage retrieval precision
- **Evidence selection** — pick the most relevant passages

## Core Concepts

### Caller-owned document identity

The rerank request carries only a caller-controlled unique ID and exact text per document. Application metadata and first-stage retrieval scores stay outside the provider request. This keeps the interface reusable and reduces accidental disclosure.

### Provider index mapping

HTTP reranker protocols identify results by the zero-based position of the submitted document. Array position is transport identity, not application identity. The adapter maps:

```
request.Documents[i].ID
    -> provider request documents[i]
    -> provider response result.index == i
    -> response.Results[*].DocumentID
```

A caller must never infer application identity from response order alone.

### TopN

`TopN` controls response cardinality. Some providers return only the highest-scoring `TopN` documents. A caller that needs one score per input must set `TopN == len(Documents)`. The package validates the actual response against the requested cardinality. `TopN` is never defaulted: explicit cardinality lets complete-score callers prove they requested all candidates.

### Scores

Scores are provider- and model-specific. They may be negative and must not be interpreted as probabilities. The package accepts any finite score and sorts descending.

### Usage and cost

Reranking usually consumes input tokens but produces no generated output tokens. The `Usage` record carries input and total tokens when the provider reports them. A nil `Cost` means pricing is unknown; a pointer to zero means the provider is explicitly free/local under the selected pricing policy. nil and zero are intentionally distinguishable.

## Quick Start (Go)

```go
import (
    "github.com/go-go-golems/geppetto/pkg/rerank"
    "github.com/go-go-golems/geppetto/pkg/rerank/llamacpp"
    "github.com/go-go-golems/geppetto/pkg/security"
)

provider, err := llamacpp.New(llamacpp.Options{
    BaseURL: "http://127.0.0.1:18012",
    Model:   "qllama/bge-reranker-v2-m3:q4_k_m",
    OutboundURL: security.OutboundURLOptions{
        AllowHTTP:          true,
        AllowLocalNetworks: true,
    },
})
if err != nil { panic(err) }

resp, err := provider.Rerank(ctx, rerank.Request{
    Query: "How does TTC calculate a payroll adjustment?",
    Documents: []rerank.Document{
        {ID: "chunk-001", Text: "A payroll adjustment corrects wages or deductions."},
        {ID: "chunk-002", Text: "Cypress trees tolerate dry conditions."},
    },
    TopN: 2,
})
if err != nil { panic(err) }

for _, r := range resp.Results {
    fmt.Println(r.Rank, r.DocumentID, r.Score)
}
```

## Profile-based Construction (Go)

Reranking integrates with Geppetto's engine profile system. A rerank-only profile stacks a base API profile:

```yaml
inference_settings:
  api:
    base_urls:
      rerank-base-url: http://127.0.0.1:18012
    allow_http:
      rerank: true
    allow_local_networks:
      rerank: true
  rerank:
    type: llamacpp
    engine: qllama/bge-reranker-v2-m3:q4_k_m
    max_request_bytes: 2097152
    max_response_bytes: 1048576
```

Construct from resolved `InferenceSettings`:

```go
import (
    rerankfactory "github.com/go-go-golems/geppetto/pkg/rerank/factory"
)

factory, err := rerankfactory.NewSettingsFactoryFromInferenceSettings(resolvedSettings)
if err != nil { panic(err) }
provider, err := factory.NewProvider()
if err != nil { panic(err) }
```

`ValidateInferenceSettingsForRerank` gives profile-oriented diagnostics before construction.

## JavaScript API

The `require("geppetto")` module exposes `reranker(settings)`, consistent with `embeddings(settings)`:

```javascript
const gp = require("geppetto");
const settings = gp.inferenceProfiles
  .load("~/.config/pinocchio/profiles.yaml")
  .resolve("bge-reranker-local");

const reranker = gp.reranker(settings);

// Synchronous (for bounded scripts):
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

### Asynchronous execution

For event-loop applications, use `rerankAsync` to avoid blocking the runtime owner thread:

```javascript
const handle = reranker.rerankAsync(query, documents, {topN: documents.length});
try {
  const response = await handle.promise;
  // ...
} finally {
  handle.close();
}
```

The handle exposes `cancel()` and `close()` for cancellation and runtime shutdown. The provider goroutine touches no JavaScript value; Promise settlement happens on the runtime owner thread.

### Security constraints

JavaScript cannot supply an endpoint, credential, HTTP client, local-network exception, or provider implementation directly. Those capabilities remain in host/profile configuration. The `settings` argument must be the hidden-reference `InferenceSettings` wrapper returned by `inferenceProfiles.resolve`.

## Security

The llama.cpp adapter enforces:

- **Bounded IO**: request and response bodies are size-limited (`MaxRequestBytes`, `MaxResponseBytes`).
- **Strict decoding**: unknown JSON fields and trailing data are rejected.
- **Outbound URL policy**: scheme, host, userinfo, query, and fragment are validated; HTTP and local networks are denied by default.
- **Redirect rejection**: local model endpoints should not redirect; redirects are rejected to prevent validated-endpoint escape.
- **Safe errors**: query text, document text, credentials, and response bodies never enter error messages.
- **Cancellation**: context cancellation terminates active requests.

## Supported Providers

Currently only `llamacpp` is supported. The core package is transport-neutral; future adapters (Cohere, Jina) can be added without changing the `Provider` interface.

## Live Qualification

Use the runnable example for interactive qualification. It accepts either a
resolved rerank profile or complete inline rerank settings and emits one row
per ranked document, so both JSON and table output remain readable:

```bash
go run ./cmd/examples/rerank-profile-smoke run \
  --rerank-type llamacpp \
  --rerank-engine qllama/bge-reranker-v2-m3:q4_k_m \
  --rerank-base-url http://127.0.0.1:18012 \
  --output json
```

The opt-in test remains the automation guard:

```bash
GEPPETTO_LIVE_RERANK=1 \
GEPPETTO_RERANK_BASE_URL=http://127.0.0.1:18012 \
GEPPETTO_RERANK_MODEL=qllama/bge-reranker-v2-m3:q4_k_m \
go test ./pkg/rerank/llamacpp -run TestLive -v -count=1
```

It skips unless `GEPPETTO_LIVE_RERANK=1` is set exactly, never falls back to a fixture, and never starts external services itself.

## Downstream Integration

Applications (such as a RAG system) adapt Geppetto reranking through a thin domain adapter. The dependency direction is:

```
application -> geppetto/pkg/rerank
geppetto/pkg/rerank -X-> any application package
```

The application owns evidence IDs, manifests, truncation, complete-score requirements, traces, citations, and evaluation. Geppetto owns the provider transport.
