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
RelatedFiles:
    - Path: repo://pkg/embeddings/cohere.go
      Note: Review hardening—endpoint derivation, outbound policy, injected client, cardinality (commit 7abdc347)
    - Path: repo://pkg/embeddings/settings_factory.go
      Note: Factory client/policy wiring and blank-local precedence (commit 7abdc347)
    - Path: repo://pkg/sections/sections_test.go
      Note: CLI cohere-api-key registration regression test (commit 7abdc347)
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

## Step 3: Documentation audit follow-up

A user prompt ("Did you update whatever documentation of geppetto was necessary, btw? in pkg/doc") triggered a full doc/config audit, which caught two real gaps the implementation had missed.

### Prompt Context

**User prompt (verbatim):** "Did you update whatever documentation of geppetto was necessary, btw? in pkg/doc"

**Assistant interpretation:** Audit pkg/doc (and adjacent config) for anything that enumerates embedding/rerank providers and could be stale after adding cohere.

**Inferred user intent:** Trust but verify — make sure no provider whitelist or doc index was left behind.

**Commit (code):** `f3ff3a4b` — "fix(embeddings): add cohere to embeddings-type choices; index reranking topic doc (GEPPETTO-COHERE-EMBEDDINGS-001)"

### What I did
- Grepped pkg/doc for provider enumerations; audited the JS guides, profiles guide, docs index, and the embeddings flags YAML.
- **Gap 1 (functional):** `pkg/embeddings/config/flags/embeddings.yaml` — `embeddings-type` is a `choice` flag listing only `openai`/`ollama`; CLI/parsed-values construction with `type: cohere` would fail choice validation. Added `cohere` and updated the dimensions help text.
- **Gap 2 (navigational, predates this ticket):** `pkg/doc/topics/00-docs-index.md` never listed `15-reranking.md` (missing since GEPPETTO-RERANKER-001). Added a Reranking row.
- Confirmed `13-js-api-reference.md`/`14-js-api-user-guide.md` do not cover the reranker/embeddings JS APIs (DTS + `15-reranking.md` JS section carry that; a pre-existing gap, not made worse).

### What worked / what didn't
- The audit found real issues the implementation phase missed — the choice-flag whitelist is exactly the same class of omission as `settings_validation.go`, one layer down.

### What I learned
- Provider additions in geppetto have a checklist shape: factory case → settings validation → **Glazed flag choices** → topic docs → docs index. The flag choices are easy to forget because profile YAML bypasses them.

### What warrants a second pair of eyes
- Whether `embeddings-type` choice expansion affects any snapshot/golden CLI tests (full suite passed, so none caught it).

### What should be done in the future
- Add the five-point checklist above to a playbook or AGENT.md note for future provider ports.

### Code review instructions
- `git show f3ff3a4b` — two files, one choice list, one index row.

## Step 4: Address all six Cohere embeddings review findings

PR #408 received six P2 inline findings, all on the embeddings half of the change. I treated them as one coherent boundary-hardening pass rather than six isolated patches: the provider now has the same base-URL, outbound-policy, and host-owned-client semantics as the Cohere reranker; its batch response contract is explicit; and direct CLI credentials plus profile precedence are internally consistent.

The implementation changed the new Cohere constructor to return an error because endpoint validation belongs at construction time, before any text or bearer credential can be sent. This is a deliberate API correction inside the still-open PR, not a compatibility shim.

### Prompt Context

**User prompt (verbatim):** "address code review comments on the geppetto PR"

**Assistant interpretation:** Fetch every inline review comment on geppetto PR #408, implement each requested correction with regression coverage, update docs and ticket records, then push and resolve/reply to the threads.

**Inferred user intent:** Make PR #408 safe and review-ready rather than merely acknowledging automated findings.

**Commit (code):** `7abdc347` — "fix(embeddings): address Cohere PR review findings"

### What I did
- Added `cohere-api-key` to the registered `embeddings` Glazed section. The existing `*-api-key` wildcard decode into `APISettings.APIKeys` makes `--embeddings-type cohere --cohere-api-key ...` usable through `CreateGeppettoSections`; a section-level regression test proves the field is present.
- Changed `cohere-base-url` to shared base semantics: default/base is `https://api.cohere.com`, and embeddings derive `/v2/embed` with `url.JoinPath`, matching rerank's `/v2/rerank` behavior.
- Changed `NewCohereProvider` to return `(*CohereProvider, error)` and validate the final endpoint at construction using `security.ValidateOutboundURL`.
- Added `WithCohereOutboundURL`; the factory derives policy from `settings.OutboundURLOptions(f.api, "embeddings")`, so HTTP/local destinations remain fail-closed unless explicitly opted in.
- Added `WithCohereHTTPClient`; `NewSettingsFactoryFromInferenceSettings` carries `InferenceSettings.Client`, calls `settings.EnsureHTTPClient`, and injects the resulting timeout/proxy/TLS-aware client into the provider.
- Enforced exact Cohere response cardinality (`len(vectors) == len(texts)`) to protect cache wrappers from nil holes and out-of-range panics.
- Skipped blank/whitespace embedding-local API keys and base URLs during profile-map overlay so validation fallback and factory precedence agree.
- Updated `06-embeddings.md` for the error-returning constructor, CLI key, shared-base contract, and outbound policy.
- Added tests for short/oversized/empty response mismatch, shared endpoint derivation, fail-closed URL policy plus explicit local opt-in, injected-client timeout behavior, blank local fallback, endpoint/base split, and registered CLI credentials.

### Why
- All six findings exposed one root problem: the minimal PR #169 embeddings salvage did not inherit the stricter provider-boundary conventions introduced by the newer rerank subsystem.
- Rejecting an invalid destination at construction time prevents accidental disclosure before a request exists and gives callers a deterministic configuration error.
- Exact response cardinality is part of the `Provider.GenerateBatchEmbeddings` contract, not optional defensive checking: both cache wrappers assume positional one-to-one output.

### What worked
- `GOWORK=off go test -race ./pkg/embeddings ./pkg/sections -count=1` passed.
- `GOWORK=off go test ./... -count=1`, targeted vet, and `git diff --check` passed.
- The commit's lefthook gate passed the full test suite, golangci-lint, geppetto-lint, and glazed-lint.
- The user's unstaged glazed v1.4.2 `go.mod`/`go.sum` bump was detected during diff audit and deliberately excluded from this focused commit.

### What didn't work
- The first targeted run failed because an existing factory test still expected the old endpoint-as-base representation:
  `--- FAIL: TestNewSettingsFactoryFromInferenceSettingsConstructsCohereProvider (0.00s)`
  `settings_factory_test.go:178: base URL = "https://api.cohere.com", want hosted default`
  I updated it to assert both fields separately: base `https://api.cohere.com`, endpoint `https://api.cohere.com/v2/embed`.

### What I learned
- `CreateGeppettoSections` does not need a separate embeddings credential section: the registered `embeddings` section is already decoded into both `EmbeddingsConfig` and `APISettings`; the latter's `glazed:"*-api-key"` wildcard captures `cohere-api-key`, exactly like provider chat sections.
- A shared `cohere-base-url` key only works when every Cohere capability treats it as a base rather than an endpoint.

### What was tricky to build
- CLI credential routing is indirect: the flag lives in the embeddings YAML, `UpdateStepSettingsFromParsedValues` decodes the same section into `ss.API`, and a wildcard struct tag stores it in a map. Registering the pre-existing `NewEmbeddingsApiKeyValue` instead would not have worked without adding its separate slug to the API decode loop and could duplicate `openai-api-key`; adding the field to the existing section is the smaller correct path.
- Tests use HTTP loopback servers, which the production policy correctly rejects. Test construction therefore needs an explicit `AllowHTTP + AllowLocalNetworks` option helper; this keeps production defaults fail-closed and makes each local opt-in visible.

### What warrants a second pair of eyes
- Confirm `"embeddings"` is the desired profile key for `api.allow_http` / `api.allow_local_networks`; it mirrors rerank's capability key and is documented, but some users may expect a provider key such as `cohere`.
- Confirm changing the newly introduced `NewCohereProvider` constructor to return an error is acceptable before merge; all in-repo callers and docs are updated.

### What should be done in the future
- Consider consolidating hosted-provider endpoint/client/security options into a shared constructor pattern so embeddings and rerank cannot drift again.

### Code review instructions
- Start with `pkg/embeddings/cohere.go` (`NewCohereProvider`, `GenerateBatchEmbeddings`), then `pkg/embeddings/settings_factory.go` (policy/client/precedence wiring).
- Review regression contracts in `pkg/embeddings/cohere_test.go`, `pkg/embeddings/settings_factory_test.go`, and `pkg/sections/sections_test.go`.
- Validate with `GOWORK=off go test -race ./pkg/embeddings ./pkg/sections -count=1 && GOWORK=off go test ./... -count=1`.

### Technical details
- Review comments addressed: `3731910731`, `3731910733`, `3731910738`, `3731910744`, `3731910748`, `3731910754`.
- Public endpoint derivation: `url.JoinPath(baseURL, "/v2/embed")`; hosted default remains `https://api.cohere.com/v2/embed`.
