---
Title: Extract scoped DB tool pattern into reusable geppetto package
Ticket: GP-33
Status: active
Topics:
    - geppetto
    - tooldb
    - sqlite
    - architecture
    - backend
    - refactor
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket workspace for the GP-33 design and migration plan to extract scoped database tool mechanics from temporal-relationships into Geppetto.
LastUpdated: 2026-03-15T15:44:45.472262429-04:00
WhatFor: Track the research, design, and delivery artifacts for the scoped database tool extraction proposal.
WhenToUse: Use when reviewing GP-33, onboarding an implementer, or locating the primary design guide and investigation diary.
---

# Extract scoped DB tool pattern into reusable geppetto package

## Overview

This ticket captures the analysis and implementation plan for extracting the scoped SQLite query tool pattern from `temporal-relationships` into a reusable Geppetto package. The target outcome is a Geppetto-level package that lets applications define scoped, read-only SQLite snapshot tools without reimplementing schema bootstrapping, query safety, and tool registration from scratch.

The ticket is currently documentation-complete for the analysis/design phase. The implementation itself is still future work.

## Key Links

- **Primary design guide**: `design-doc/01-scoped-database-tools-extraction-analysis-design-and-implementation-guide.md`
- **Investigation diary**: `reference/01-investigation-diary.md`
- **Tasks**: `tasks.md`
- **Changelog**: `changelog.md`

## Status

Current status: **active**

Completed in this ticket so far:

- created the GP-33 workspace in `geppetto/ttmp`,
- mapped the current `temporal-relationships` implementation,
- mapped the Geppetto and Pinocchio integration points,
- wrote the detailed design and migration guide.

Still intentionally open:

- package implementation in `geppetto`,
- migration of the `temporal-relationships` call sites,
- final API naming validation during coding.

## Topics

- geppetto
- tooldb
- sqlite
- architecture
- backend
- refactor
- documentation

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
