# rerank-profile-smoke

Resolves a rerank profile (or overlays rerank flags onto a base profile),
constructs a `geppetto/pkg/rerank` provider, and runs one rerank call against
a real or local llama.cpp `/v1/rerank` server.

This is the runnable counterpart to the opt-in live test in
`pkg/rerank/llamacpp/live_test.go`. Unlike a `_test`, it is a real CLI you can
point at any running llama.cpp reranker endpoint and inspect the output with
Glazed (`--output json`, `--output table`, etc.).

## Prerequisites

A llama.cpp reranker server running with `--embedding --pooling rank --rerank`.
For example, with Ollama's bundled `llama-server`:

```bash
/Applications/Ollama.app/Contents/Resources/llama-server \
  -m /path/to/bge-reranker-v2-m3.gguf \
  --embedding --pooling rank --rerank \
  --host 127.0.0.1 --port 8012
```

If the server runs on a remote host (e.g. a Mac), forward the port over SSH:

```bash
ssh -fN -L 18012:127.0.0.1:8012 user@mac-host
```

## Usage

### Direct profile resolution

Add a rerank profile to `~/.config/pinocchio/profiles.yaml`:

```yaml
profiles:
  bge-reranker-local:
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

Then run:

```bash
go run ./cmd/examples/rerank-profile-smoke run \
  --profile bge-reranker-local \
  --output json
```

### Base-profile overlay (no rerank profile needed)

```bash
go run ./cmd/examples/rerank-profile-smoke run \
  --base-profile ollama-openai-base \
  --rerank-type llamacpp \
  --rerank-engine qllama/bge-reranker-v2-m3:q4_k_m \
  --rerank-base-url http://127.0.0.1:18012 \
  --query "How does TTC calculate a payroll adjustment?" \
  --output json
```

### Custom documents

Each `--document` is `id|text` (pipe separates the caller-controlled document
ID from its text):

```bash
go run ./cmd/examples/rerank-profile-smoke run \
  --profile bge-reranker-local \
  --query "what is a cat?" \
  --document "a|A cat is a small domesticated carnivorous mammal." \
  --document "b|A dog is a domesticated descendant of the wolf." \
  --document "c|Python is a programming language." \
  --output json
```

## Example output

```json
[
  {
    "document_count": 3,
    "duration_ms": 76,
    "model": "qllama/bge-reranker-v2-m3:q4_k_m",
    "profile": "bge-reranker-local",
    "provider": "llama.cpp",
    "query": "How does TTC calculate a payroll adjustment?",
    "results": [
      {"rank": 1, "document_id": "chunk-001", "index": 0, "score": -4.06},
      {"rank": 2, "document_id": "chunk-002", "index": 1, "score": -11.01},
      {"rank": 3, "document_id": "chunk-003", "index": 2, "score": -11.02}
    ],
    "usage_input_tokens": 76,
    "usage_total_tokens": 76
  }
]
```

Notes:
- Scores may be negative and are not probabilities; they are sorted descending.
- The caller-controlled document ID survives provider index mapping exactly
  (`request.Documents[i].ID` → `response.results[*].document_id`).
- `usage_input_tokens`/`usage_total_tokens` come from the llama.cpp `usage`
  object.
