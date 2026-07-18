---
Title: Investigation diary
Ticket: GEPPETTO-RERANKER-001
Status: active
Topics:
    - architecture
    - geppetto
    - inference
    - intern-onboarding
    - providers
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/rerank/errors.go
      Note: Sentinel error categories for safe classification (commit 6c7323b9)
    - Path: repo://pkg/rerank/order.go
      Note: ValidateAndMapResults index mapping, finite-score validation, deterministic ordering, ranks (commit 6c7323b9)
    - Path: repo://pkg/rerank/rerank.go
      Note: Core Provider interface and Request/Response/Result/Usage/Model records (commit 6c7323b9)
    - Path: repo://pkg/rerank/validate.go
      Note: ValidateRequest and ResolveModel invariants (commit 6c7323b9)
ExternalSources: []
Summary: Chronological investigation record for the reusable Geppetto reranker interface, llama.cpp provider, profile integration, validation, and delivery.
LastUpdated: 2026-07-18T18:20:00-04:00
WhatFor: Preserve evidence, decisions, commands, failures, and continuation guidance for GEPPETTO-RERANKER-001.
WhenToUse: Read before implementing or reviewing any task in this ticket.
---


# Diary

## Goal

This diary records the design and eventual implementation of a reusable reranking primitive in Geppetto. The target is a transport-neutral Go interface, strict llama.cpp adapter, normal Geppetto settings/profile integration, safe HTTP behavior, complete tests, and a thin downstream RAG adapter without any RAG dependency in Geppetto.

## Step 1: Create the ticket and define the reranker architecture

The investigation started from a concrete ownership question raised during RESEARCHCTL-015: generation and embeddings were planned through Geppetto, while reranking was still planned as a direct RAG-owned llama.cpp client. Because reranking is a reusable model-service primitive, the provider protocol belongs in Geppetto while RAG-specific evidence and trace semantics remain in the RAG repository.

I created GEPPETTO-RERANKER-001, mapped Geppetto's inference, embeddings, settings, profile, client, security, usage, and pricing systems, inspected the active RAG reranking operator, recovered the deleted prototype llama.cpp adapter as evidence, reviewed the real BGE server probe, and compared the official llama.cpp and Cohere API shapes. I then wrote an intern-facing implementation guide with public APIs, package layout, protocol DTOs, settings/profile integration, security policy, tests, phases, decisions, alternatives, risks, and exact file references.

### Prompt Context

**User prompt (verbatim):** "ok, I added geppetto/ to the workspace, Create a new docmgr ticket in there about creating the reranker interface, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a Geppetto docmgr ticket for a reusable reranker provider interface, deeply inspect relevant Geppetto and RAG code, write an implementation-ready intern guide, maintain a detailed diary and tasks, validate the ticket, and upload the bundle to reMarkable.

**Inferred user intent:** Make Geppetto the common provider substrate for generation, embeddings, and reranking so RAG v2 uses thin domain adapters rather than owning reusable model-service transports.

### What I did

- Confirmed the Geppetto repository, current branch, and existing unrelated untracked history database.
- Ran:

  ```bash
  cd /home/manuel/code/wesen/go-go-golems/geppetto
  docmgr status --summary-only
  docmgr ticket list
  docmgr vocab list
  ```

- Created:

  ```bash
  docmgr ticket create-ticket \
    --ticket GEPPETTO-RERANKER-001 \
    --title 'Reusable reranker provider interface and llama.cpp adapter' \
    --topics architecture,geppetto,inference,providers,intern-onboarding,security

  docmgr doc add --ticket GEPPETTO-RERANKER-001 \
    --doc-type design-doc \
    --title 'Reusable reranker interface architecture and implementation guide'

  docmgr doc add --ticket GEPPETTO-RERANKER-001 \
    --doc-type reference \
    --title 'Investigation diary'
  ```

- Inspected Geppetto's:
  - `pkg/embeddings.Provider`, settings factory, config, caching, and documentation;
  - `pkg/inference/engine.Engine`;
  - `InferenceSettings`, API settings, client settings, and clone path;
  - engine-profile YAML/stack merge behavior;
  - `ModelInfo`, usage, and cost semantics;
  - outbound URL validation and HTTP-client construction.
- Inspected the active RAG `Reranker` interface and `rerank.cross-encoder/v1` operator.
- Recovered deleted prototype source with:

  ```bash
  git show bf2b567^:pkg/raglab/reranker.go
  git show bf2b567^:pkg/raglab/reranker_llamacpp.go
  ```

- Reviewed the saved real llama.cpp BGE probe: exact route, model, server revision, token usage, negative scores, top-N cardinality, and index mapping.
- Searched current official llama.cpp and Cohere reranking documentation to verify that query + indexed documents + top-N + scores is a portable core.
- Wrote the primary design guide.

### Why

- Geppetto already owns provider-neutral inference, embeddings, client configuration, profile resolution, usage, pricing, and outbound security.
- Reranking is reusable outside RAG and should not be implemented independently by every application.
- The RAG repository must continue to own chunk/evidence identity, tokenization/truncation manifests, complete-score policy, ranking traces, and evaluation.
- A typed request/response API is more precise than encoding reranking as chat generation.

### What worked

- The embedding package provides a strong structural precedent for package, config, factory, and profile integration.
- The previous real probe and deleted adapter provide concrete protocol and validation evidence without imposing compatibility.
- Cohere's API independently confirms the core abstraction while llama.cpp provides the first executable local adapter.
- Existing Geppetto HTTP-client and outbound-security utilities can be reused rather than reimplemented.

### What didn't work

- No current Geppetto reranking package, interface, settings section, or provider factory exists. Repository searches found only inference and embedding model primitives.
- Geppetto's current `ModelInfo` describes one model per inference settings value. A combined chat + embedding + rerank profile could make model metadata ambiguous; the first implementation should use a rerank-only resolved profile.
- The deleted prototype adapter used `http.DefaultClient` when no client was supplied and did not enforce final redirect policy. The new design intentionally tightens those behaviors.

### What I learned

- A reusable reranker must return caller IDs as well as provider indices; response order cannot be treated as durable identity.
- Raw scores are not portable probabilities. The real llama.cpp probe returned valid negative scores, while Cohere documents normalized values.
- `TopN` must remain explicit because it controls complete versus partial score coverage.
- Reranking needs a package-specific usage shape. It generally consumes input tokens but does not produce output tokens.
- Nil cost and known zero cost must remain distinguishable for local services.

### What was tricky to build

- The interface had to remain provider-neutral without losing the exact information RAG needs. The solution is caller-owned document IDs, transport indices, finite scores, deterministic rank, and a rich response with optional usage/cost/model/duration fields.
- Profile integration spans several systems: `InferenceSettings`, clone, YAML, map-based stack merge, Glazed sections, client settings, API endpoint policy, and model metadata. The guide enumerates every required update and test.
- Outbound validation must cover redirects without mutating an injected `*http.Client`. The design proposes cloning the client value and rejecting redirects for the first local adapter.

### What warrants a second pair of eyes

- Whether the package should be named `pkg/rerank` or `pkg/reranking`; the guide proposes the shorter action-oriented name.
- Whether rerank usage should support provider-specific search units in v1 or add them only when a real provider requires them.
- Whether `InferenceSettings.ModelInfo` is sufficient for a rerank-only profile or should gain primitive-specific model metadata in a separate ticket.
- Whether strict unknown-field rejection is appropriate for the evolving llama.cpp response or whether known fields plus bounded ignored extensions are safer.

### What should be done in the future

- Implement the ticket phases in order, beginning with API review and sanitized protocol fixtures.
- Add the RAG adapter only after core and llama.cpp conformance tests pass.
- Consider Cohere or Jina adapters only after the first local provider is complete.

### Code review instructions

- Start with `pkg/embeddings/embeddings.go`, then compare it to the proposed `pkg/rerank/rerank.go` API.
- Review the protocol mapping and security sections before implementation.
- Verify that every proposed dependency points from applications to Geppetto, never from Geppetto to RAG.
- Validate the final ticket with:

  ```bash
  docmgr doctor --ticket GEPPETTO-RERANKER-001 --stale-after 30
  ```

### Technical details

- Proposed package: `github.com/go-go-golems/geppetto/pkg/rerank`.
- First provider: `pkg/rerank/llamacpp`.
- Proposed profile key: `inference_settings.rerank`.
- First route: `POST /v1/rerank`.
- Scores: every finite float, sorted descending.
- Identity: caller document ID mapped through provider array index.
- Redirect policy: reject in v1.
- Local HTTP/network policy: explicit opt-in only.

## Step 2: Validate and publish the design baseline

After completing the guide, I related the seven most important evidence files, updated the ticket changelog and task tracker, validated all frontmatter, checked Markdown fences and Git whitespace, and ran `docmgr doctor`. The required dry run and real reMarkable bundle upload both succeeded.

The published bundle contains the ticket overview, 5,482-word design guide, task tracker, diary, and changelog. This records an implementation-ready baseline; the core API and provider code remain open tasks.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete ticket bookkeeping, validate the design package, and deliver it to reMarkable.

**Inferred user intent:** Make the reranker architecture available for review before implementation begins.

### What I did

- Related seven source/evidence files to the focused design document.
- Ran frontmatter validation for the index, design, and diary.
- Ran `docmgr doctor --ticket GEPPETTO-RERANKER-001 --stale-after 30`.
- Verified balanced Markdown code fences and `git diff --check`.
- Ran the reMarkable dry run and real upload.

### Why

- File relations let a future implementer move directly from design claims to source evidence.
- Validation prevents malformed ticket metadata from reducing discoverability.
- A bundled PDF keeps architecture, tasks, and investigation context together.

### What worked

- Every frontmatter validation passed.
- `docmgr doctor` reported all checks passed.
- The upload returned:

  ```text
  OK: uploaded GEPPETTO-RERANKER-001 Interface Design.pdf -> /ai/2026/07/18/GEPPETTO-RERANKER-001
  ```

### What didn't work

- The first commit attempt stopped at the staged whitespace check:

  ```text
  ttmp/2026/07/18/GEPPETTO-RERANKER-001--reusable-reranker-provider-interface-and-llama-cpp-adapter/changelog.md:27: new blank line at EOF.


  Command exited with code 2
  ```

  The failing command was `git diff --cached --check`. I removed the extra final blank line generated by the changelog update and restaged the ticket before retrying.

### What I learned

- The current ticket can be implemented in independently reviewable layers: core API, llama.cpp transport, settings/profiles, live qualification/RAG adapter, then hardening.

### What was tricky to build

- The design needed enough downstream RAG context to prove the interface, without moving RAG evidence concepts into Geppetto. The dependency and adapter sections make that boundary explicit.

### What warrants a second pair of eyes

- Public API naming and usage extensibility should be reviewed before P1 begins because changing them later would affect every provider adapter.

### What should be done in the future

- Begin P0.1-P0.3 by reviewing and freezing the public API and sanitized protocol fixtures.

### Code review instructions

- Read the design's public API, llama.cpp adapter, settings/profile, and decision-record sections first.
- Compare open P0 tasks with the risks around usage units and model metadata.
- Rerun `docmgr doctor` after later documentation changes.

### Technical details

- Remote path: `/ai/2026/07/18/GEPPETTO-RERANKER-001`.
- Bundle: `GEPPETTO-RERANKER-001 Interface Design.pdf`.
- ToC depth: 2.
- Validation result: all checks passed.

## Step 3: Extend the design with the Goja API

The user clarified that reranking must be available through Geppetto's JavaScript module as well as its Go API. I inspected the hard-cut `require("geppetto")` surface, profile-resolved embeddings wrapper, hidden-reference mechanism, runtime owner bridge, cancellable session Promise handle, generated TypeScript template, DTS parity test, hard-cut contract, and JavaScript examples.

I updated the architecture so `gp.reranker(settings)` constructs the same profile-resolved Go provider. The wrapper exposes a synchronous method for bounded command scripts and a cancellable asynchronous handle for event-loop applications. JavaScript cannot construct endpoints, credentials, HTTP clients, or provider callbacks; those remain host/profile capabilities.

### Prompt Context

**User prompt (verbatim):**

> Add goja API as well
>
> Add goja API as well ,update design doc

**Assistant interpretation:** Extend GEPPETTO-RERANKER-001's design, phases, tasks, tests, and deliverables to include a typed `require("geppetto")` API.

**Inferred user intent:** Make reranking a first-class Geppetto primitive in both Go and JavaScript, consistent with existing inference and embeddings access.

### What I did

- Read:
  - `pkg/js/modules/geppetto/module.go`;
  - `api_embeddings.go` and its tests;
  - `api_session.go`'s asynchronous Promise/cancellation path;
  - `pkg/js/runtimebridge/bridge.go`;
  - `dts_parity_test.go`;
  - `spec/geppetto.d.ts.tmpl`;
  - `module_hardcut_test.go`.
- Added a complete Goja design section covering:
  - `reranker(settings)`;
  - `rerank(query, documents, {topN})`;
  - `rerankAsync(...)` and `{promise,cancel,close}`;
  - precise TypeScript interfaces;
  - strict JS decoding;
  - hidden Go references;
  - runtime-owner-safe Promise settlement;
  - cancellation and runtime shutdown;
  - DTS, hard-cut, export, method, race, and example tests.
- Added a dedicated Goja implementation phase and renumbered qualification and hardening phases.
- Expanded the package layout, decisions, review checklist, workflow, deliverables, references, and full validation commands.
- Added six task-tracker entries and renumbered later task phases.
- Validated the expanded design and force-republished the same reMarkable bundle name after a successful dry run.

### Why

- The existing module already exposes `gp.embeddings(settings)`, so omitting reranking would leave the model-service primitives inconsistent.
- Network reranking can block the runtime owner thread. Supporting both sync and async lets small scripts remain simple while event-loop applications retain cancellation and responsiveness.
- Profile-only construction prevents JavaScript from bypassing endpoint and credential policy.

### What worked

- Existing Geppetto patterns directly support the design:
  - `api_embeddings.go` provides the profile-resolved wrapper precedent;
  - `session.runAsync` provides the cancellable Promise and owner-thread settlement precedent;
  - hidden references prevent forged plain objects from becoming provider capability;
  - DTS parity and hard-cut tests protect the public surface.
- The updated bundle upload succeeded at `/ai/2026/07/18/GEPPETTO-RERANKER-001/GEPPETTO-RERANKER-001 Interface Design.pdf`.

### What didn't work

- The current DTS parity test checks top-level and selected namespace exports, but it does not prove every method on `RerankerProvider` or `RerankAsyncHandle`. The design therefore requires explicit runtime method-surface assertions or a deliberate extension to the parity tooling.
- The first post-changelog validation ended with:

  ```text
  ttmp/2026/07/18/GEPPETTO-RERANKER-001--reusable-reranker-provider-interface-and-llama-cpp-adapter/changelog.md:37: new blank line at EOF.


  Command exited with code 2
  ```

  `docmgr changelog update` appended an extra final blank line. I removed it and reran `git diff --check` before publication and commit.

### What I learned

- No `goja.Value`, Promise resolver, object, or VM conversion may cross into the provider goroutine. The request must be deep-copied to ordinary Go values first, and settlement must be posted back through the runtime owner.
- The synchronous method can reuse the runtime lifetime context, but long-lived event-loop applications should use the asynchronous method.
- The JavaScript API must require `{id,text}` documents; accepting strings would discard the identity guarantee established by the Go API.

### What was tricky to build

- Matching the embeddings API alone would have produced a synchronous network method that blocks the owner thread. The design resolves this by retaining sync compatibility while adding the established asynchronous handle pattern.
- Promise resolution itself is a Goja operation. The background goroutine may call only the provider; conversion, `resolve`, and `reject` must happen on the owner thread.

### What warrants a second pair of eyes

- Review whether both sync and async methods should ship in v1 or whether async should be the primary documented path.
- Review whether to extend the generic DTS parity parser or keep a reranker-specific method-surface test.

### What should be done in the future

- Implement core Go and settings phases before the Goja wrapper so JavaScript remains a thin adapter over a stable provider API.

### Code review instructions

- Start at design section 8.6, then review Phase 4 and the two new JavaScript decision records.
- Compare the proposed async flow with `agentSessionRef.startAsync` and `moduleRuntime.postOnOwner`.
- Confirm the task list contains one coherent P4 Goja phase and renumbered P5/P6 work.

### Technical details

- Top-level factory: `gp.reranker(settings)`.
- Sync method: `rerank(query, documents, {topN, model?})`.
- Async method: `rerankAsync(...) -> {promise,cancel,close}`.
- Provider construction: registry-resolved `InferenceSettings` only.
- Type source: `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`.
- Updated design size: 6,846 words.
- Publication used `--force` because it replaces the earlier design-only PDF with the Goja-expanded design.

## Step 4: Implement the core pkg/rerank package (Phase 1)

With the design frozen and the Goja API added, I began implementation. Phase 1 establishes the transport-neutral core: public records, the Provider interface, request validation, response mapping and deterministic ordering, sentinel errors, and JSON/YAML round-trip tests. No HTTP and no RAG import belong here; a fake in-process provider proves the deterministic ID-mapped responses.

I grounded every type in the design's section 8 and in the real BGE probe evidence: scores may be negative (the probe returned -3.32, -9.83, -11.01), so validation checks finiteness only, never a `[0,1]` range. The deleted prototype's `mapLlamaCPPResults` informed the index/duplicate/non-finite checks, but the new core is provider-neutral and returns a rich `Response` rather than `[]Result`.

**Commit (code):** 6c7323b91033f65d6ba82b7b96c9c09dc29367da — "feat(rerank): add core pkg/rerank types, validation, and ordering"

### Prompt Context

**User prompt (verbatim):** "Implement the GEPPETTO-RERANKER-001 docmgr ticket. keep a detailed diary as you work, commit at appropriate intervals. Work in the geppetto/ directory (also for docmgr). Read skills to make sure you use the right commands and diary format."

**Assistant interpretation:** Implement the GEPPETTO-RERANKER-001 ticket phase by phase, keeping a detailed diary, committing at appropriate intervals, and using docmgr from the geppetto/ directory with the correct skill commands and diary format.

**Inferred user intent:** Move from design to a working, tested, committed reranker primitive in Geppetto, with diary evidence and ticket bookkeeping at each phase.

### What I did

- Read the `diary` and `docmgr` skill references to confirm the diary step format and docmgr command set.
- Oriented on the ticket: 6 docs tasks done, 28 implementation tasks open; baseline `go build ./...` and the embeddings/settings/engineprofiles/js-module test suites all green.
- Read the reference implementations: `pkg/embeddings/embeddings.go`, `settings_factory.go`, `config/settings.go`, `ollama.go` (HTTP+security pattern), `pkg/security/outbound_url.go`, `pkg/steps/ai/settings/settings-inference.go`, `settings-client.go`, `http_client.go`, `model_info.go`, and `pkg/engineprofiles/inference_settings_merge.go`.
- Read the JS module precedent: `module.go`, `api_embeddings.go`, `api_inference_settings.go`, `api_session.go` async settlement, `api_owner_bridge.go`, `codec.go`, the DTS template, the DTS parity test, and the hard-cut test.
- Recovered the deleted prototype reranker (`reranker.go`, `reranker_llamacpp.go`) and the real BGE probe results as wire-DTO and validation evidence.
- Implemented `pkg/rerank`:
  - `doc.go` package documentation and dependency-direction contract;
  - `rerank.go` `Document`, `Request`, `Result`, `Usage`, `Response`, `Model`, and `Provider`;
  - `errors.go` sentinel categories;
  - `validate.go` `ValidateRequest` and `ResolveModel`;
  - `order.go` `ValidateAndMapResults`, `SortResults`, `AssignRanks`, `RawResult`;
  - `options.go` shared `ProviderOption`/`ProviderOptions`;
  - `rerank_test.go` and `codec_test.go` covering empty query/docs, duplicate IDs, empty text, invalid TopN, model mismatch, index-to-ID mapping, equal-score tie breaks, wrong cardinality, missing index/score, out-of-range and duplicate index, zero/negative scores, and JSON/YAML round trips.
- Ran `gofmt -w`, `go test ./pkg/rerank/... -count=1 -v` (20 tests pass), and `go vet ./pkg/rerank/...`.
- Checked for RAG/researchctl imports: none (only a doc-comment reference to the dependency direction).
- Committed Phase 1 code, then checked off P1.1-P1.4 and related the four core files to the diary.

### Why

- Phase 1 defines the contracts every later phase depends on. Building and testing it in isolation (no HTTP, no settings, no Goja) keeps the core provably provider-neutral and lets the llama.cpp adapter be a thin transport.
- The fake provider test proves the exit gate of Phase 1: a fake provider can return deterministic, ID-mapped, provider-neutral responses with usage and cost.

### What worked

- The design's section 8 types translated directly into Go with no surprises.
- The deleted prototype's validation logic was a useful cross-check for index/duplicate/non-finite handling, but the new core returns a rich `Response` and is transport-neutral.
- JSON/YAML round-trip tests confirmed `omitempty` keeps nil cost and nil usage absent while preserving a non-nil pointer-to-zero cost (the nil-vs-zero distinction the design requires).

### What didn't work

- The first `git commit` exited with code 1 from the parallel lefthook pre-commit (`make lintmax` + `make test`), even though both ultimately passed when run individually. The failure was a hook race, not a code defect; re-running the commit on an already-compiled tree succeeded.

### What I learned

- `RawResult` needs `HasIndex`/`HasScore` bools (pointer-like presence flags) to distinguish a missing provider field from a valid zero index or zero score. Plain `int`/`float64` zero values would silently accept a malformed response.
- `sort.SliceStable` plus a three-key comparator (score desc, index asc, ID asc) gives deterministic ordering even when the provider returns results out of score order, matching the probe evidence where response order happened to already be score order.

### What was tricky to build

- The nil-vs-zero cost distinction is enforced structurally: `Cost *float64` with `omitempty` serializes a nil cost away entirely and serializes a pointer-to-zero as `0`. The round-trip test `TestResponse_JSONRoundTrip_WithOptionals` pins this so a future change cannot silently collapse them.
- `ValidateRequest` must not mutate `Request.Model`; it only validates. `ResolveModel` is a separate non-mutating helper so callers can fill the model before encoding without surprising side effects.

### What warrants a second pair of eyes

- Confirm the three-key tie-break order (score desc, index asc, ID asc) matches downstream RAG expectations before the llama.cpp adapter and RAG adapter rely on it.
- Confirm `ValidateRequest`'s model-mismatch rule (strict equality when both non-empty) is the right first behavior versus allowing the request model to override the provider model.

### What should be done in the future

- Phase 2 (llama.cpp adapter) should reuse `ValidateRequest` and `ValidateAndMapResults` rather than re-implementing validation, so the core invariants stay authoritative.
- Add a property/fuzz test for `ValidateAndMapResults` once the llama.cpp fixtures exist, to cover arbitrary index permutations.

### Code review instructions

- Start in `pkg/rerank/rerank.go` (types), then `validate.go` (`ValidateRequest`), then `order.go` (`ValidateAndMapResults`, `SortResults`).
- Validate with:

  ```bash
  go test ./pkg/rerank/... -count=1 -v
  go vet ./pkg/rerank/...
  ```

- Confirm no RAG/researchctl import: `rg -n "rag|researchctl" pkg/rerank/` (only a doc-comment dependency-direction reference is expected).

### Technical details

- Sentinel errors: `ErrInvalidRequest`, `ErrInvalidResponse`, `ErrUnavailable`, `ErrRequestTooLarge`, `ErrResponseTooLarge`.
- Default request byte bound: `DefaultMaxRequestBytes = 2 << 20` (2 MiB).
- Tie-break comparator: score descending, then input index ascending, then document ID ascending.
- Ranks assigned from 1 after sorting.
- Real BGE probe scores used as the negative-score design anchor: -3.32784366607666, -9.837879180908203, -11.012685775756836.
