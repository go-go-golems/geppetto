// Package rerank provides a transport-neutral reranking primitive.
//
// Reranking is the third Geppetto model-service primitive alongside inference
// (pkg/inference/engine) and embeddings (pkg/embeddings). Given one query and
// an ordered set of caller-owned documents, a cross-encoder reranker scores
// and reorders those documents.
//
// The package is deliberately retrieval-agnostic. It knows about query text,
// document text, and durable caller-controlled document IDs. It does not know
// about chunks, evidence, fusion, collapse, citations, traces, or evaluation.
// Those concerns belong to the caller (for example, a RAG system that adapts
// this primitive through a thin domain adapter).
//
// Dependency direction:
//
//	application (e.g. rag-evaluation-system/pkg/ragproviders)
//	    -> geppetto/pkg/rerank
//
//	geppetto/pkg/rerank
//	    -X-> any RAG or application package
//
// Core invariants enforced by this package:
//
//   - Document IDs are required, non-empty, and unique within a request.
//   - TopN is explicit and must be in [1, len(Documents)].
//   - Scores may be any finite value (including negative); they are not
//     interpreted as probabilities.
//   - Provider array indices map back to caller document IDs; response order is
//     never used as durable identity.
//   - Results are deterministically ordered: score descending, then input index
//     ascending, then document ID ascending.
//   - Ranks are assigned from 1 after sorting.
package rerank
