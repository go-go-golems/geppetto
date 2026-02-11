---
Title: Implement typed Turn.Data/Metadata accessors
Ticket: 002-IMPLEMENT-TYPE-DATA-ACCESSOR
Status: complete
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Linter that will need enhancement for new API
    - Path: geppetto/pkg/turns/keys.go
      Note: Current canonical key definitions (to be migrated to typed Key[T] pattern)
    - Path: geppetto/pkg/turns/serde/serde.go
      Note: Serialization normalization that initializes nil maps
    - Path: geppetto/pkg/turns/types.go
      Note: |-
        Core Turn/Block type definitions with current map-based Data/Metadata fields
        Wrapper API implementation that triggered the generic methods limitation
        Final implementation using generic functions (DataSet/DataGet/etc) instead of generic methods
    - Path: geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md
      Note: Design doc specifying the typed wrapper API to implement
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/analysis/01-codebase-analysis-turn-data-metadata-access-locations.md
      Note: Full inventory of migration sites
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/reference/01-diary.md
      Note: Step 3/4 where the generic methods issue was encountered and resolved
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/sources/go-generic-methods.md
      Note: Generics constraint note referenced by Step 3/4
    - Path: moments/backend/pkg/inference/middleware/compression/turn_data_compressor.go
      Note: Compression middleware that needs refactoring for typed API
    - Path: moments/backend/pkg/inference/middleware/current_user_middleware.go
      Note: Example middleware with Turn.Data access patterns
    - Path: moments/backend/pkg/turnkeys/data_keys.go
      Note: Moments-specific Turn.Data keys (to migrate)
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-25T09:04:20.13525101-05:00
WhatFor: ""
WhenToUse: ""
---








# Implement typed Turn.Data/Metadata accessors

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- turns
- go
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
