---
Title: Reusable reranker provider interface and llama.cpp adapter
Ticket: GEPPETTO-RERANKER-001
Status: active
Topics:
    - architecture
    - geppetto
    - inference
    - intern-onboarding
    - providers
    - security
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Add reranking as a reusable Geppetto model-service primitive with caller-owned document identity, typed usage and cost, strict llama.cpp transport, safe HTTP policy, profile-backed construction, and a thin downstream RAG adapter.
LastUpdated: 2026-07-18T18:20:00-04:00
WhatFor: Track design and implementation of pkg/rerank and its first real provider adapter.
WhenToUse: Use for all implementation, review, qualification, and documentation work on Geppetto reranking.
---

# Reusable reranker provider interface and llama.cpp adapter

## Overview

Geppetto currently provides provider-neutral generation through `pkg/inference/engine` and embeddings through `pkg/embeddings`. This ticket adds the missing reranking primitive: one query and an ordered set of caller-identified documents produce validated scores and deterministic ranks.

The first implementation will provide a strict llama.cpp `/v1/rerank` adapter backed by Geppetto's existing client, profile, model metadata, and outbound-security systems. It will also expose the same profile-resolved provider through `require("geppetto").reranker(settings)`, with synchronous and cancellable asynchronous execution, precise TypeScript declarations, and runtime-owner-safe Promise settlement. The core package remains portable to future Cohere, Jina, or other adapters. Geppetto will not import RAG types or own retrieval, fusion, hydration, citation, trace, or evaluation semantics.

## Current status

- Ticket and documentation structure created.
- Geppetto and downstream RAG architecture mapped.
- Real llama.cpp BGE protocol evidence reviewed.
- Intern-facing design and implementation guide written.
- Thirty-four tasks organized across documentation and Phases 0-6.
- **Implementation complete (Phases 1-6):**
  - Phase 1: `pkg/rerank` core types, validation, deterministic ordering, sentinel errors, and tests (commit 6c7323b9).
  - Phase 2: strict `pkg/rerank/llamacpp` adapter with bounded IO, redirect rejection, outbound URL policy, safe errors, and sanitized BGE fixture (commit 86729b43).
  - Phase 3: `pkg/rerank/config` RerankConfig, `InferenceSettings.Rerank` integration, `pkg/rerank/factory` settings factory, profile stack overlay, and the Glazed section registered in `pkg/sections` and `pkg/cli/bootstrap` (commit 09c438c4).
  - Phase 4: `gp.reranker(settings)` Goja sync and cancellable async API, generated TypeScript declarations, hard-cut/DTS parity, 11 Goja tests, and a runnable example (commits 786e09d1, 3ca8716a).
  - Phase 5/6: opt-in live llama.cpp test, reranking topic guide, and full hardening (unit, race, lint, vet, DTS, dependency-direction) (commit 4ed0d038).
- **Deferred to RESEARCHCTL-015:** the thin downstream RAG adapter and complete-score/usage propagation into the native RAG reranking trace.
- Geppetto contains no RAG or researchctl import; dependency direction is `application -> geppetto/pkg/rerank`.

## Primary guide

- [Reusable reranker interface architecture and implementation guide](./design-doc/01-reusable-reranker-interface-architecture-and-implementation-guide.md)

The guide includes:

- conceptual definitions and package ownership;
- current Geppetto architecture;
- downstream RAG requirements;
- official and live protocol evidence;
- proposed package layout;
- complete Go and JavaScript API sketches;
- synchronous and cancellable asynchronous Goja execution design;
- llama.cpp DTO and transport pseudocode;
- settings, Glazed, and profile integration;
- HTTP and outbound security policy;
- test matrix and live qualification flow;
- five implementation phases with exit gates;
- decision records, alternatives, risks, and exact file references.

## Supporting documents

- [Investigation diary](./reference/01-investigation-diary.md)
- [Task tracker](./tasks.md)
- [Changelog](./changelog.md)

## End-state acceptance

The ticket is complete when:

- `pkg/rerank.Provider` is transport-neutral and has no RAG dependency;
- caller document IDs survive provider index mapping exactly;
- any finite provider score is accepted and sorted deterministically;
- usage, model, optional cost, duration, and request identity are preserved;
- request and response bodies are bounded;
- endpoint, local-network, proxy, redirect, timeout, and cancellation policies are tested;
- `InferenceSettings.Rerank` survives YAML, clone, and profile stack resolution;
- `reranker(settings)` exposes sync and async execution without exposing endpoint or credential capability;
- generated TypeScript, hard-cut, export, and method surfaces agree;
- a rerank-only profile constructs a real llama.cpp provider;
- the opt-in live BGE test passes;
- RAG DSL v2 uses a thin adapter and emits a complete native reranking trace;
- full Geppetto validation and docmgr checks pass.

## Boundary

```text
Geppetto pkg/rerank
  owns: provider API, transport, config, usage, client, security

RAG adapter
  owns: chunks, evidence, model manifests, truncation, trace, evaluation

Researchctl
  owns: generic run lifecycle and artifact custody
```

No dependency points from Geppetto to RAG or researchctl.
