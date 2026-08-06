---
Title: Diary
Ticket: GEPPETTO-RERANKER-002
Status: active
Topics:
    - geppetto
    - inference
    - providers
    - architecture
    - embeddings
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: Chronological implementation diary for the Cohere reranker adapter port.
LastUpdated: 2026-08-06T00:00:00-04:00
WhatFor: Record investigation and implementation steps, failures, and review guidance for GEPPETTO-RERANKER-002.
WhenToUse: Read before resuming or reviewing work on this ticket.
---

# Diary

## Goal

Track the port of the PR #169 Cohere reranker onto the modern `pkg/rerank` primitive: design doc first, then stepwise implementation (`pkg/rerank/cohere` adapter, factory/validation wiring, Goja parity proof, docs), with a commit and a printed brutalist slip after each step.

## Step 1: Rebase probe, ticket creation, and intern-facing design guide

Before writing any code I established ground truth: PR #169 cherry-picks onto current main with only two trivial conflicts and its tests pass, but its `embeddings.Reranker` interface is architecturally superseded by the GEPPETTO-RERANKER-001 primitive (`pkg/rerank` + llamacpp adapter + factory + Goja API, merged July 2026). I then created GEPPETTO-RERANKER-002 and wrote the full architecture and implementation guide so an intern can implement the port without any prior context.

### Prompt Context

**User prompt (verbatim):** "Rebase https://github.com/go-go-golems/geppetto/pull/169 on top of geppetto/ and see if it integrates cleanly, I know we have maybe been adding reranking as part of our work in ~/code/wesen/go-go-golems/ragkit and ~/code/wesen/go-go-golems/ragopt . Anyway, analyze, and tell me what should best be done" followed by "Create a new docmgr ticket to port over the reranker to th emodern API and lose the deprecated legacy. Create a detailed analysis / design / implementation guide that is for a new intern... Store in the ticket and the nupload to remarkable. Then implement it, step by step, ckeeping a detailed diary as you work, and committinga t appropriate intervals. After each diary entry, print a short brutalist ticket on the printer about what you did and how it went."

**Assistant interpretation:** Probe the PR #169 rebase, decide the right integration strategy, document it as an intern-ready docmgr ticket, publish the docs to reMarkable, then implement the port stepwise with diary entries, commits, and thermal-printed status slips.

**Inferred user intent:** Salvage the year-old Cohere rerank work properly onto the modern rerank primitive, with full documentation hygiene (ticket, diary, reMarkable, printed slips) rather than a quick merge.

**Commit (code):** N/A — docs only in this step.

### What I did
- Fetched `pr-169` (single commit `9567b207` on base `8e2a2034`, May 2025) and cherry-picked it onto `origin/main` on scratch branch `scratch/pr-169-rebase-probe`.
- Resolved the two conflicts by keeping main's versions; verified `go build`, `go vet`, and `go test ./pkg/embeddings/` all pass, including the PR's own Cohere tests.
- Deleted the probe branch after recording results; workspace left clean on `task/add-cohere-reranking`.
- Surveyed `pkg/rerank` (core, llamacpp adapter, config, factory, Goja API), GEPPETTO-RERANKER-001 ticket docs, ragkit's own `rag.Reranker`, and ragopt (no rerank code).
- Created ticket GEPPETTO-RERANKER-002 with `docmgr ticket create-ticket`, added the design doc and this diary, wrote index/tasks.
- Wrote the 10-section intern guide: background, current-state architecture with file references, legacy artifact fate table, Cohere v2 API reference, proposed design with pseudocode, DR-1..DR-5, five-phase plan, 25-row test matrix, risks, references.

### Why
- The rebase probe answers "does it integrate cleanly" with evidence rather than intuition: mechanically yes, architecturally no.
- The guide front-loads every design decision (five decision records) so implementation is mechanical and review can focus on the decision records instead of rediscovering them.

### What worked
- The cherry-pick probe was fast and conclusive: conflicts only in `pkg/embeddings/settings_factory_test.go` (add/add) and `pkg/doc/topics/06-embeddings.md`; the factory merge compiled against the current config because `EmbeddingsConfig` still carries `APIKeys`/`BaseURLs` maps.
- GEPPETTO-RERANKER-001's diary literally pre-wrote the extension recipe ("extend SupportedProviders and the NewProvider switch", "a new outbound/auth policy may be needed"), which anchors the design in prior art.

### What didn't work
- Nothing failed. The probe branch was throwaway by design.

### What I learned
- PR #169's `settings_factory.go` hunk merges cleanly textually and compiles — textual mergeability is a weak signal; the semantic conflict (duplicate rerank API) is the real one.
- ragkit keeps its own `rag.Reranker` (`rag/components.go:80`) with no geppetto dependency; downstream adaptation is a separate ticket's job.
- Cohere's v2 rerank response carries the request ID in the body (`id`) and bills in `search_units`, not tokens — both facts shape DR-3.

### What was tricky to build
- Deciding the usage/cost mapping. Cohere reports `meta.billed_units.search_units`; `rerank.Usage` models tokens. Stuffing searches into token fields would corrupt the scientific-run aggregation semantics the core was built for; extending the core struct for one provider's billing quirk violates the 001 API freeze. DR-3 (Usage nil, optional per-search cost) threads that needle.
- Redirect policy: the 001 diary suggested hosted providers "may legitimately redirect", but failing closed on redirects from a fixed canonical HTTPS endpoint is more auditable (DR-2). I documented the construction-time `BaseURL` override as the migration path.

### What warrants a second pair of eyes
- DR-3: is `Usage == nil` for Cohere acceptable to downstream scientific runs, or do we need a `search_units` field in the core after all?
- DR-2: redirect rejection for a hosted API — confirm we are comfortable failing closed.
- The decision to give `BaseURL` a hosted default (`https://api.cohere.com`) diverges from llamacpp's "no silent endpoint" stance; the guide justifies it, but it is a judgment call.

### What should be done in the future
- After implementation: close PR #169 with a pointer to this ticket.
- Separate ticket for the PR #169 Cohere **embeddings** provider (works via cherry-pick but needs `settings_validation.go` whitelisting).
- ragkit adapter `rag.Reranker` ← `rerank.Provider` (RESEARCHCTL-015 lineage).

### Code review instructions
- Start at `ttmp/2026/08/06/GEPPETTO-RERANKER-002--cohere-reranker-adapter-on-pkg-rerank-port-of-pr-169/design-doc/01-....md`, §6.6 (decision records) — everything else follows from those.
- Cross-check §4 file references against the tree; all paths were verified to exist while writing.

### Technical details
- Probe results: `git cherry-pick 9567b207` onto `a119860d` → conflicts in `pkg/embeddings/settings_factory_test.go` (add/add), `pkg/doc/topics/06-embeddings.md`; `go test ./pkg/embeddings/` ok (0.746s) after taking main's conflicted files.
- Ticket: `docmgr ticket create-ticket --ticket GEPPETTO-RERANKER-002 --topics geppetto,inference,providers,architecture,embeddings`.
