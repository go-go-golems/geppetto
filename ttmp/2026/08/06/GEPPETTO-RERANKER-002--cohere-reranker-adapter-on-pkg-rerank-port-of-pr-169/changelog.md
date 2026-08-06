# Changelog

## 2026-08-06

- Initial workspace created


## 2026-08-06

Step 1: Rebase probe of PR #169 (2 trivial conflicts, tests pass, architecturally superseded); ticket created; intern-facing architecture and implementation guide written with DR-1..DR-5 and 25-row test matrix

### Related Files

- /home/manuel/workspaces/2026-08-06/add-cohere-reranking/geppetto/ttmp/2026/08/06/GEPPETTO-RERANKER-002--cohere-reranker-adapter-on-pkg-rerank-port-of-pr-169/design-doc/01-cohere-reranker-adapter-architecture-and-implementation-guide.md — Primary implementation guide


## 2026-08-06

Step 2: pkg/rerank/cohere adapter core + 30 mock-server tests, -race clean (commits 1a5a9639, 163c69be)

### Related Files

- /home/manuel/workspaces/2026-08-06/add-cohere-reranking/geppetto/pkg/rerank/cohere/provider.go — Strict Cohere v2 /rerank adapter implementing rerank.Provider
- /home/manuel/workspaces/2026-08-06/add-cohere-reranking/geppetto/pkg/rerank/cohere/provider_test.go — 30-test suite covering matrix rows 1-19


## 2026-08-06

Step 3: factory/validation wiring, Goja parity test, topic docs + live test (commits 66b4e650, f0b0ca69, fb0cba6e)

### Related Files

- /home/manuel/workspaces/2026-08-06/add-cohere-reranking/geppetto/pkg/doc/topics/15-reranking.md — Cohere provider user docs
- /home/manuel/workspaces/2026-08-06/add-cohere-reranking/geppetto/pkg/js/modules/geppetto/api_reranker_test.go — Goja parity proof for type=cohere
- /home/manuel/workspaces/2026-08-06/add-cohere-reranking/geppetto/pkg/rerank/factory/settings_factory.go — Cohere provider case, API key resolution, per-type validation

