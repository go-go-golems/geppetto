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

The first implementation will provide a strict llama.cpp `/v1/rerank` adapter backed by Geppetto's existing client, profile, model metadata, and outbound-security systems. The core package remains portable to future Cohere, Jina, or other adapters. Geppetto will not import RAG types or own retrieval, fusion, hydration, citation, trace, or evaluation semantics.

## Current status

- Ticket and documentation structure created.
- Geppetto and downstream RAG architecture mapped.
- Real llama.cpp BGE protocol evidence reviewed.
- Intern-facing design and implementation guide written.
- Twenty-eight tasks organized across documentation and Phases 0-5.
- Implementation has not started.

## Primary guide

- [Reusable reranker interface architecture and implementation guide](./design-doc/01-reusable-reranker-interface-architecture-and-implementation-guide.md)

The guide includes:

- conceptual definitions and package ownership;
- current Geppetto architecture;
- downstream RAG requirements;
- official and live protocol evidence;
- proposed package layout;
- complete public API sketches;
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
