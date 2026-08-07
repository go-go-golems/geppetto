---
Title: 'Cohere embeddings provider (port of PR #169)'
Ticket: GEPPETTO-COHERE-EMBEDDINGS-001
Status: active
Topics:
    - geppetto
    - embeddings
    - providers
    - architecture
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: Port the PR #169 Cohere embeddings provider onto the current embeddings factory and profile validation; live-qualified with a real key.
LastUpdated: 2026-08-06T16:48:00.645301951-04:00
WhatFor: Track implementation of the Cohere embeddings provider port.
WhenToUse: Use when reviewing or extending the Cohere embeddings provider.
---

# Cohere embeddings provider (port of PR #169)

## Overview

PR #169 (May 2025) added a Cohere embeddings provider alongside its reranker. The reranker half was ported onto the modern `pkg/rerank` primitive in GEPPETTO-RERANKER-002; this ticket completes the salvage by bringing the embeddings half into the current `pkg/embeddings` package: factory case, profile validation (`cohere-api-key` with embedding-local precedence), docs, and a live API qualification.

## Current status

- Provider + tests salvaged from `pr-169` with one safety fix (non-200 error bodies no longer echoed) — commit `af24d50c`.
- Factory `cohere` case + dimensions-0 semantics; validation accepts cohere with profile-oriented diagnostics.
- Topic guide updated (`06-embeddings.md`) — commit `5d7ea446`.
- Live qualification: `cohere-embedder` profile → real embed-v4.0 vector, 1024/1024 dimensions.
- See [Diary](./reference/01-diary.md) for the full record.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- embeddings
- providers
- architecture

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
