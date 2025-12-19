---
Title: Typed accessor for Turn.Data (opaque wrapper)
Ticket: 001-TYPED-ACCESSOR-TURN-DATA
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Investigate turning turns.Turn.Data into an opaque wrapper with typed access (Get[T]) while preserving YAML compatibility and iteration needs."
LastUpdated: 2025-12-17T17:35:01.569057594-05:00
---

# Typed accessor for Turn.Data (opaque wrapper)

## Overview

Explore how to evolve `turns.Turn.Data` (currently `map[turns.TurnDataKey]any`) into an **opaque** structure with **typed accessors** (e.g., `Get[T]`) while keeping serialization and iteration semantics intact.

## Key Links

- [Analysis: Opaque Turn.Data: typed Get[T] accessors](./analysis/01-opaque-turn-data-typed-get-t-accessors.md)
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
