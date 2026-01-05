---
Title: 'Generic turn types: store-specific key families + key methods'
Ticket: 001-GENERIC-TURN-TYPES
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - turns
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Migrate turns to store-specific key families + key methods (Get/Set), then remove legacy Key[T] + DataGet/DataSet API."
LastUpdated: 2026-01-05T17:15:18.264870597-05:00
WhatFor: ""
WhenToUse: ""
---

# Generic turn types: store-specific key families + key methods

## Overview

Implement a new production API in `geppetto/pkg/turns`:

- Store-specific key families: `DataKey[T]`, `TurnMetaKey[T]`, `BlockMetaKey[T]`
- Key receiver methods: `key.Get(store)` / `key.Set(&store, value)`

Then migrate canonical key definitions (geppetto + downstream) and run the one-shot rewrite tool (`geppetto/cmd/turnsrefactor`) to convert call sites from `turns.DataGet/DataSet/...` to `key.Get/key.Set`, and finally delete the legacy API.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Analysis**: `analysis/01-analysis-implement-store-specific-key-families-key-methods.md`
- **Diary**: `reference/01-diary.md`

## Status

Current status: **active**

## Topics

- architecture
- geppetto
- go
- turns

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
