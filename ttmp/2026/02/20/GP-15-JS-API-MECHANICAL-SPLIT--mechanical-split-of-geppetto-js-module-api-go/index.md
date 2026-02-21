---
Title: Mechanical split of geppetto JS module api.go
Ticket: GP-15-JS-API-MECHANICAL-SPLIT
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - inference
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-20T12:04:15.245792199-05:00
WhatFor: ""
WhenToUse: ""
---

# Mechanical split of geppetto JS module api.go

## Overview

This ticket performs a move-only split of `pkg/js/modules/geppetto/api.go` into domain-focused files under the same package (`geppetto`). No API or behavior changes are intended in this ticket; it is strictly structural to improve maintainability and reviewability.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

Progress:

- Analysis and split plan complete
- Mechanical split implementation complete
- Targeted tests and race checks complete
- Ticket docs/diary/changelog update in progress

## Topics

- architecture
- geppetto
- go
- inference

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
