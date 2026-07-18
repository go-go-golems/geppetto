# Tasks

## TODO

- [x] D0.1 Map Geppetto inference, embeddings, settings, profiles, security, usage, and HTTP client architecture <!-- t:i45o -->
- [x] D0.2 Inspect active RAG reranking requirements and proven llama.cpp protocol evidence <!-- t:vlpk -->
- [x] D0.3 Write the intern-facing reranker architecture and implementation guide <!-- t:akmt -->
- [x] D0.4 Relate evidence files, update bookkeeping, and pass docmgr validation <!-- t:xii2 -->
- [x] D0.5 Upload the design bundle to reMarkable <!-- t:14hq -->
- [x] D0.6 Extend the design with the profile-resolved Goja sync/async API and republish the bundle <!-- t:m05r -->
- [x] P0.1 Review and freeze pkg/rerank public request, response, usage, model, provider, and error APIs <!-- t:xuf0 -->
- [x] P0.2 Save sanitized llama.cpp request and response protocol fixtures tied to the tested server revision <!-- t:20br -->
- [x] P0.3 Confirm package naming, profile key, route aliases, and first adapter version <!-- t:sb7k -->
- [x] P1.1 Implement pkg/rerank public types and Provider interface <!-- t:62lw -->
- [x] P1.2 Implement request validation and stable sentinel error categories <!-- t:ssa5 -->
- [x] P1.3 Implement response index mapping, finite-score validation, deterministic ordering, and ranks <!-- t:odwu -->
- [x] P1.4 Add core API, malformed input, ordering, JSON, and YAML tests <!-- t:2cz9 -->
- [x] P2.1 Implement pkg/rerank/llamacpp options and constructor validation <!-- t:8xdd -->
- [x] P2.2 Implement bounded request encoding and strict bounded response decoding <!-- t:cqd8 -->
- [x] P2.3 Implement outbound URL policy, injected client handling, and redirect rejection <!-- t:4tae -->
- [x] P2.4 Implement model, usage, duration, optional cost, and safe error mapping <!-- t:7vru -->
- [x] P2.5 Add llama.cpp conformance, security, cancellation, timeout, and body-limit tests <!-- t:ty0v -->
- [x] P3.1 Add RerankConfig and Glazed rerank fields <!-- t:ahxl -->
- [x] P3.2 Add InferenceSettings.Rerank initialization, cloning, and validation <!-- t:oxdo -->
- [x] P3.3 Implement settings-backed rerank ProviderFactory with explicit supported providers <!-- t:8gtj -->
- [x] P3.4 Add engine-profile YAML round-trip, stack overlay, clone, API, client, and local-network tests <!-- t:5pu9 -->
- [x] P3.5 Update inference-settings summaries and profile documentation <!-- t:maiw -->
- [x] P4.1 Add the top-level reranker(settings) Goja factory and hidden provider wrapper <!-- t:bc09 -->
- [x] P4.2 Add strict JavaScript request decoding and synchronous rerank response conversion <!-- t:i38u -->
- [x] P4.3 Add cancellable rerankAsync Promise handles with owner-thread settlement <!-- t:xkkw -->
- [x] P4.4 Update generated DTS, hard-cut, export-surface, and method-surface parity checks <!-- t:ys9e -->
- [x] P4.5 Add Goja sync, async, cancellation, runtime-close, race, and runnable example tests <!-- t:ikwn -->
- [x] P5.1 Add opt-in live llama.cpp qualification test and record exact model/server evidence <!-- t:2127 -->
- [ ] P5.2 (deferred to RESEARCHCTL-015) Implement and validate the thin downstream RAG adapter under RESEARCHCTL-015 <!-- t:ueay -->
- [ ] P5.3 (deferred to RESEARCHCTL-015) Prove complete score and usage propagation into the native RAG reranking trace <!-- t:qo7o -->
- [x] P6.1 Add Geppetto reranking topic documentation and Go/JavaScript copy-paste examples <!-- t:0r8r -->
- [x] P6.2 Run full unit, race, module-isolation, lint, vet, vulnerability, and dependency checks <!-- t:67dj -->
- [x] P6.3 Complete diary, changelog, tasks, doctor validation, and final reMarkable publication <!-- t:tq7v -->
- [x] P5.1b Add rerank-profile-smoke runnable example CLI for live qualification <!-- t:k307 -->
