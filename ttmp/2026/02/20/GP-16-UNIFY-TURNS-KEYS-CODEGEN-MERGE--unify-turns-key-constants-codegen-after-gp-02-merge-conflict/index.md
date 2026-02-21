---
Title: Unify turns key constants/codegen after GP-02 merge conflict
Ticket: GP-16-UNIFY-TURNS-KEYS-CODEGEN-MERGE
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - inference
    - turns
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-20T12:15:21.973717791-05:00
WhatFor: ""
WhenToUse: ""
---

# Unify turns key constants/codegen after GP-02 merge conflict

## Overview

This ticket analyzes and resolves the merge conflict between manual turns key constant additions from GP-02 and the schema/codegen-based key generation now used in main. The objective is to re-establish a single source of truth for key constants while keeping engine-owned typed keys in `pkg/inference/engine/turnkeys.go`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- architecture
- geppetto
- go
- inference
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
