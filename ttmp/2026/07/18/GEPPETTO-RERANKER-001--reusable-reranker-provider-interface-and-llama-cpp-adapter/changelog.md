# Changelog

## 2026-07-18

- Initial workspace created


## 2026-07-18

Created the reusable Geppetto reranker ticket, mapped provider/profile/security boundaries, and wrote the intern-facing architecture and implementation guide.

### Related Files

- /home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/embeddings.go — Provider API precedent
- /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/07/18/GEPPETTO-RERANKER-001--reusable-reranker-provider-interface-and-llama-cpp-adapter/design-doc/01-reusable-reranker-interface-architecture-and-implementation-guide.md — Primary design deliverable
- /home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/pkg/ragoperators/rank.go — Downstream native reranking semantics


## 2026-07-18

Validated and published the reranker interface design bundle to /ai/2026/07/18/GEPPETTO-RERANKER-001/GEPPETTO-RERANKER-001 Interface Design.pdf.

### Related Files

- /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/07/18/GEPPETTO-RERANKER-001--reusable-reranker-provider-interface-and-llama-cpp-adapter/design-doc/01-reusable-reranker-interface-architecture-and-implementation-guide.md — Published primary architecture guide
- /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/07/18/GEPPETTO-RERANKER-001--reusable-reranker-provider-interface-and-llama-cpp-adapter/reference/01-investigation-diary.md — Recorded validation and publication evidence

## 2026-07-18

Extended the reranker design with profile-resolved synchronous and cancellable asynchronous Goja APIs, precise DTS, runtime-owner safety, tests, and a dedicated implementation phase.

### Related Files

- /home/manuel/code/wesen/go-go-golems/geppetto/pkg/js/modules/geppetto/api_session.go — Async cancellation and owner-thread settlement precedent
- /home/manuel/code/wesen/go-go-golems/geppetto/pkg/js/modules/geppetto/module.go — Public module and hidden-reference integration point
- /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/07/18/GEPPETTO-RERANKER-001--reusable-reranker-provider-interface-and-llama-cpp-adapter/design-doc/01-reusable-reranker-interface-architecture-and-implementation-guide.md — Expanded primary design with Goja architecture

## 2026-07-18

Republished the expanded Goja-aware reranker design bundle at /ai/2026/07/18/GEPPETTO-RERANKER-001/GEPPETTO-RERANKER-001 Interface Design.pdf.

### Related Files

- /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/07/18/GEPPETTO-RERANKER-001--reusable-reranker-provider-interface-and-llama-cpp-adapter/design-doc/01-reusable-reranker-interface-architecture-and-implementation-guide.md — Published expanded Go and Goja API design
- /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/07/18/GEPPETTO-RERANKER-001--reusable-reranker-provider-interface-and-llama-cpp-adapter/reference/01-investigation-diary.md — Recorded Goja design and republication evidence

## 2026-07-18

Step 4: Implemented pkg/rerank core package (Phase 1, P1.1-P1.4): Provider interface, records, validation, deterministic ordering, sentinel errors, JSON/YAML tests. Commit 6c7323b9.

### Related Files

- /home/manuel/code/wesen/go-go-golems/geppetto/pkg/rerank/order.go — Response mapping and deterministic ordering (commit 6c7323b9)
- /home/manuel/code/wesen/go-go-golems/geppetto/pkg/rerank/rerank.go — Core Provider interface and records (commit 6c7323b9)

## 2026-07-18

Step 5: Implemented strict llama.cpp /v1/rerank adapter (Phase 2, P2.1-P2.5): bounded IO, strict JSON, outbound URL policy, redirect rejection, safe errors, sanitized BGE fixture. Commit 86729b43.

### Related Files

- /home/manuel/code/wesen/go-go-golems/geppetto/pkg/rerank/llamacpp/provider.go — Strict adapter with bounded IO and redirect rejection (commit 86729b43)

## 2026-07-18

Step 6: Integrated RerankConfig, settings factory, and profile stack overlay (Phase 3, P3.1-P3.5). Commit 09c438c4.

### Related Files

- /home/manuel/code/wesen/go-go-golems/geppetto/pkg/rerank/factory/settings_factory.go — Settings factory and InferenceSettings validation (commit 09c438c4)
