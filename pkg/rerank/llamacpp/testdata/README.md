# llama.cpp rerank protocol fixture

Date: 2026-07-16 (captured); 2026-07-18 (sanitized fixture saved)

## Server revision

| Property | Value |
| --- | --- |
| Server binary | `/Applications/Ollama.app/Contents/Resources/llama-server` |
| Server version | `1 (cb295bf59)`, AppleClang 21.0.0.21000099, Darwin arm64 |
| Model blob | `qllama/bge-reranker-v2-m3:q4_k_m` |
| Server flags | `--embedding --pooling rank --rerank --host 127.0.0.1 --port 8012` |
| Request route | `POST /v1/rerank` |

## Sanitized response fixture

Stored alongside this file as `bge-reranker-v2-m3-response.json`. It contains
only the documented response fields (model, object, usage, results). No query
text, document text, credentials, or host topology are present.

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

## Decoder contract

- `results[*].index` is the zero-based position of the submitted document.
- `results[*].relevance_score` is finite but may be negative; it is not a
  probability and must not be clamped to `[0,1]`.
- Response cardinality equals the requested `top_n`.
- `usage.prompt_tokens` and `usage.total_tokens` are reported and equal here;
  the adapter maps them into `rerank.Usage{InputTokens, TotalTokens}`.

## Provenance

Evidence originally captured by
`rag-evaluation-system/ttmp/2026/07/15/RAGEVAL-RERANK-001/.../scripts/02-llamacpp-bge-reranker-probe-results.md`.
This fixture is the sanitized, test-loadable form tied to the tested server
revision.
