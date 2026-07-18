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
RelatedFiles: []
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
