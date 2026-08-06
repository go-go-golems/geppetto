# Tasks

## TODO

- [x] P1: Create pkg/rerank/cohere adapter core (doc.go, protocol.go, provider.go)
- [x] P2: Adapter unit tests with mock HTTP server (test matrix rows 1-19), -race clean
- [ ] P3: Factory + validation wiring (cohere type, API key resolution, diagnostics) + tests
- [ ] P4: Goja parity test proving gp.reranker works with type=cohere unchanged
- [ ] P5: Extend pkg/doc/topics/15-reranking.md with Cohere section; opt-in live test
- [ ] P6: Final validation (build, vet, lint, full test suite, docmgr doctor), close-out; recommend closing PR #169
