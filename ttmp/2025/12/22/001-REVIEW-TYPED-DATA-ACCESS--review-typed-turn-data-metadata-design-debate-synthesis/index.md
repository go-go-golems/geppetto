---
Title: Review typed Turn.Data/Metadata design (debate synthesis)
Ticket: 001-REVIEW-TYPED-DATA-ACCESS
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
    - review
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Current linter implementation analyzed
    - Path: geppetto/pkg/turns/types.go
      Note: Current Turn.Data structure analyzed in review
    - Path: geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md
      Note: Primary document under review; this ticket builds the reviewer packet around it.
    - Path: moments/backend/pkg/inference/middleware/compression/turn_data_compressor.go
      Note: Compression middleware transformation pattern
    - Path: moments/backend/pkg/inference/middleware/current_user_middleware.go
      Note: Example middleware access patterns
ExternalSources: []
Summary: Reviewer packet (participants + question set) to review the typed Turn.Data/Metadata synthesis doc and converge on explicit decisions.
LastUpdated: 2025-12-22T13:50:36.042741745-05:00
WhatFor: Run a structured review of the typed Turn.Data/Metadata design space, ending in explicit decisions for a big-bang implementation.
WhenToUse: Use before starting implementation or RFC work for typed Turn.Data/Metadata/Block.Metadata changes, and during subsequent PR/RFC reviews.
---


# Review typed Turn.Data/Metadata design (debate synthesis)

## Overview

This ticket is a **review pack** for the synthesis doc on typed `Turn.Data` / `Turn.Metadata` / `Block.Metadata`.
It provides:

- A **small reviewer team** (personas, roles, biases) to keep feedback focused
- A **question pack** mapped to the synthesis’ decision axes (big-bang; no migration sequencing prompts)

## Key Links

- **Synthesis doc (under review)**: `../../2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md`
- **Participants**: `reference/01-review-participants-small-team.md`
- **Questions**: `reference/02-review-questions-for-typed-turn-data-metadata-design.md`

## Status

Current status: **active**

## Topics

- geppetto
- turns
- go
- architecture
- review

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
