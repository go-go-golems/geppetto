---
Title: Diary
Ticket: GEPPETTO-COHERE-EMBEDDINGS-001
Status: active
Topics:
    - geppetto
    - embeddings
    - providers
    - architecture
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: Implementation diary for the Cohere embeddings provider port.
LastUpdated: 2026-08-06T00:00:00-04:00
WhatFor: Record implementation steps, failures, and review guidance for GEPPETTO-COHERE-EMBEDDINGS-001.
WhenToUse: Read before resuming or reviewing work on this ticket.
---

# Diary

## Goal

Port the PR #169 Cohere embeddings provider (`pkg/embeddings/cohere.go`) onto the current embeddings factory and profile validation, discard nothing of the modern architecture, live-qualify with a real key, and document.

## Step 1: Provider port, factory case, validation, tests

The earlier rebase probe of PR #169 (GEPPETTO-RERANKER-002 diary, Step 1) had already shown the Cohere embedding provider cherry-picks and compiles against the current `EmbeddingsConfig`. This step salvaged the provider and its tests from `pr-169`, hardened the one unsafe behavior (error-body echo), wired the factory and validation, and extended the existing test tables.

### Prompt Context

**User prompt (verbatim):** "Add support for cohere embeddings as well"

**Assistant interpretation:** Complete the second half of the PR #169 salvage: bring the Cohere embeddings provider into the current embeddings package with factory + validation wiring, matching the conventions established by the reranker port.

**Inferred user intent:** Feature-parity for Cohere across both model-service primitives that PR #169 touched, so profiles can use hosted Cohere for embeddings and reranking alike.

**Commit (code):** `af24d50c` — "feat(embeddings): add Cohere provider, factory case, and profile validation (GEPPETTO-COHERE-EMBEDDINGS-001)"

### What I did
- Extracted `pkg/embeddings/cohere.go` and `cohere_test.go` verbatim from `pr-169`, then fixed one safety issue: non-200 responses no longer decode and echo the provider error body into the returned error (it may contain request text); the body is drained bounded instead.
- Factory: added the `cohere` case to `NewProvider` (key from options or `config.APIKeys["cohere-api-key"]`, optional `cohere-base-url` override, hosted default inside the provider); dimensions default logic now allows `0` for cohere (provider omits `output_dimension`, API uses model default) while openai keeps its 1536 default and other types still error.
- Validation: added `cohereEmbeddingProvider` with `cohereEmbeddingAPIKey` resolution (embedding-local wins over top-level `api.api_keys`), and updated the supported-types message to "openai, ollama and cohere".
- Tests: updated the existing table test that used `cohere` as its "unsupported provider" example (switched to `jina`, same pattern as the rerank factory); added cohere missing-key / complete / embedding-local-key cases; added three factory tests (construct from InferenceSettings with local-key precedence + hosted default URL, base-URL override honored, missing-key diagnostic).

### Why
- The embeddings package is deliberately less ceremonial than `pkg/rerank` (simple interface, no bounded-IO contract), so the port keeps the PR's shape with only the leak fix — over-hardening would diverge from the package's existing openai/ollama style.

### What worked
- `go test ./pkg/embeddings/` green; pre-commit gate (whole repo + lint) green on the commit.
- The probe's prediction held exactly: factory hunk applied without surprises.

### What didn't work
- Two mechanical misses on my part: a stale test assumption (cohere-as-unsupported example) — expected and fixed; a missing `strings` import in the appended factory tests — one build error, fixed immediately.

### What I learned
- The embeddings factory resolves credentials through `EmbeddingsConfig.APIKeys` after merging top-level → embedding-local maps, so the provider case only needs the map lookup; precedence is handled upstream.
- The JS embeddings API (`api_embeddings.go`) constructs through the same factory, so Cohere embeddings are available from JavaScript with zero JS changes — same property as the reranker.

### What was tricky to build
- Dimensions semantics: Cohere's `output_dimension` is optional on the wire (model default when omitted), but the factory errored on `dimensions == 0` for non-openai types. Resolution: cohere `0` means "omit", documented in code and the topic guide.

### What warrants a second pair of eyes
- The decision to keep the PR provider's minimal structure (fresh `http.Client` per call, no byte bounds) rather than imposing rerank-style strictness — confirm the package-level consistency argument holds.
- The error-body drain limit of 1 MiB on non-200 responses.

### What should be done in the future
- A cached-provider integration test with cohere underneath (cache wrapping is provider-agnostic, but untested for cohere specifically).
- Consider documenting `input_type` per-call configurability if retrieval apps need query/document asymmetry from profiles (currently constructor-only).

### Code review instructions
- Start at `pkg/embeddings/cohere.go` (diff against `git show pr-169:pkg/embeddings/cohere.go` — only the non-200 handling changed), then the `cohere` case in `pkg/embeddings/settings_factory.go`, then `settings_validation.go`.
- Validate: `go test ./pkg/embeddings/ -count=1`.

### Technical details
- Cohere v2 embed request: `model`, `input_type` (default `search_document`), `texts`, `embedding_types: ["float"]`, `truncate: "END"`, optional `output_dimension`.

## Step 2: Docs, profile entry, and live Cohere embed qualification

I documented the provider in the embeddings topic guide, added a `cohere-embedder` profile to `~/.pinocchio/config/profiles.yaml`, and made a real Cohere `/embed` call through the profile registry: a real 1024-dimension `embed-v4.0` vector came back, matching the configured dimensions exactly.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the feature: docs, profile, live qualification, bookkeeping.

**Inferred user intent:** Usable, documented hosted Cohere embeddings alongside the reranker.

**Commit (code):** `5d7ea446` — "docs(embeddings): Cohere provider section in topic guide (GEPPETTO-COHERE-EMBEDDINGS-001)"

### What I did
- `pkg/doc/topics/06-embeddings.md`: added "Using Cohere for Hosted Embeddings" (constructor example, input-type note, dimensions-0 semantics, profile YAML keys) and a Cohere row in the provider comparison table.
- `~/.pinocchio/config/profiles.yaml`: added `cohere-embedder` (type cohere, engine embed-v4.0, dimensions 1024, key from `/tmp/cohere.key`).
- Live run: `go run ./cmd/examples/embedding-profile-smoke run --profile-registries ~/.pinocchio/config/profiles.yaml --profile cohere-embedder --text "What is the capital of the United States?"` → real embedding, `actual_dimensions: 1024` matching `configured_dimensions: 1024`.

### Why
- The smoke CLI exercises the production path (registry → validation → factory → provider → API), same qualification pattern as the reranker.

### What worked
- First live call succeeded with plausible float vectors.

### What didn't work
- Nothing failed.

### What I learned
- The embedding-profile-smoke row's `key_configured: false` field refers to inline CLI flags, not profile keys — the profile key was used (the call required auth).

### What was tricky to build
- N/A.

### What warrants a second pair of eyes
- Plaintext key in the user profiles file (consistent with all other providers there).

### What should be done in the future
- Close PR #169: both of its halves (rerank, embeddings) are now superseded by merged-convention implementations with live qualifications.

### Code review instructions
- Docs: `pkg/doc/topics/06-embeddings.md` "Using Cohere for Hosted Embeddings".
- Reproduce: the smoke command above.

### Technical details
- Live result (2026-08-06): provider cohere, model embed-v4.0, configured 1024 = actual 1024, preview `0.0102, 0.0329, 0.0348, 0.0084, -0.0460`.
