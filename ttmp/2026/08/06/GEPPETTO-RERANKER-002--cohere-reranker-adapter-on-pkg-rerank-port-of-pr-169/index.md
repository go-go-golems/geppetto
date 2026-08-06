---
Title: 'Cohere reranker adapter on pkg/rerank (port of PR #169)'
Ticket: GEPPETTO-RERANKER-002
Status: active
Topics:
    - geppetto
    - inference
    - providers
    - architecture
    - embeddings
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources:
    - https://github.com/go-go-golems/geppetto/pull/169
    - https://docs.cohere.com/reference/rerank
Summary: Port the PR #169 Cohere reranker onto the modern pkg/rerank primitive as pkg/rerank/cohere, wire it through the settings factory and profile validation, prove the Goja path needs no changes, and discard the deprecated embeddings.Reranker legacy.
LastUpdated: 2026-08-06T15:53:57.048456716-04:00
WhatFor: Track design and implementation of the Cohere rerank provider for geppetto.
WhenToUse: Use for all implementation, review, and documentation work on the Cohere rerank adapter.
---

# Cohere reranker adapter on pkg/rerank (port of PR #169)

## Overview

Geppetto's reranking primitive (`pkg/rerank`, GEPPETTO-RERANKER-001) shipped with a single llama.cpp provider and an explicitly reserved slot for hosted providers. This ticket adds the second provider: a strict Cohere v2 `/rerank` adapter in `pkg/rerank/cohere`, constructed through the existing settings factory from engine profiles and `InferenceSettings`, exposed to JavaScript through the existing `gp.reranker(settings)` API with no JS changes.

PR #169 (May 2025) attempted Cohere reranking as `pkg/embeddings.CohereReranker` behind a small `embeddings.Reranker` interface that predates `pkg/rerank`. A rebase probe (2026-08-06) showed it cherry-picks with only two trivial conflicts, but the interface is a strict regression versus `rerank.Provider` (no caller-owned document IDs, no usage/cost propagation, no deterministic ordering, no bounded IO, no outbound URL policy, no profile integration). This ticket ports the protocol knowledge and **discards the legacy API**; PR #169 should be closed once this ships.

## Current status

- Rebase probe of PR #169 completed (2 trivial conflicts; tests pass; architecturally superseded).
- Architecture mapped: core (`pkg/rerank`), reference adapter (`pkg/rerank/llamacpp`), config/factory/validation, Goja surface, outbound security.
- Intern-facing architecture and implementation guide written (see below).
- Implementation: not started.

## Primary guide

- [Cohere reranker adapter: architecture and implementation guide](./design-doc/01-cohere-reranker-adapter-architecture-and-implementation-guide.md)

The guide covers: retrieval-pipeline background, the `pkg/rerank` contract, the llamacpp adapter as reference implementation, the Cohere v2 API, package layout, provider/factory/validation pseudocode, five decision records (DR-1..DR-5), a five-phase implementation plan, a 25-row test matrix, risks and open questions.

## Supporting documents

- [Diary](./reference/01-diary.md)
- [Task tracker](./tasks.md)
- [Changelog](./changelog.md)

## End-state acceptance

The ticket is complete when:

- `pkg/rerank/cohere` implements `rerank.Provider` with bounded IO, strict decoding, redirect rejection, safe errors, and Authorization-header auth;
- the factory constructs it from `rerank.type: cohere` with `cohere-api-key` from `InferenceSettings.API` and an optional `cohere-base-url` override;
- `ValidateInferenceSettingsForRerank` accepts `cohere` with profile-oriented diagnostics;
- the full test matrix (§8 of the guide) passes under `-race`;
- a Goja test proves `gp.reranker(settings)` works with `type: cohere` unchanged;
- `pkg/doc/topics/15-reranking.md` documents the Cohere provider;
- no `embeddings.Reranker` legacy is present anywhere in the tree.

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.
