---
Title: Remove Turn.RunID; store SessionID in Turn metadata
Ticket: GP-02-REMOVE-TURN-RUN-ID
Status: active
Topics:
    - geppetto
    - turns
    - inference
    - refactor
    - design
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T09:57:27.405855024-05:00
WhatFor: ""
WhenToUse: ""
---

# Remove Turn.RunID; store SessionID in Turn metadata

## Overview

This ticket explores removing the top-level `RunID` field from `turns.Turn` and instead storing session correlation (`SessionID`) in `turns.Turn.Metadata` using a typed key, set at `session.Session.Append` time.

Primary motivations:

- Make `turns.Turn` portable/independent at construction time (no baked-in “parent run/session” field).
- Resolve the naming mismatch between `RunID` (Turn) and `SessionID` (Session).
- Keep downstream correlation (events/logging/persistence) by reading the value from metadata instead.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Analysis**: `analysis/01-replace-turn-runid-with-sessionid-in-turn-metadata.md`
- **Diary**: `reference/01-diary.md`

## Status

Current status: **active**

## Topics

- geppetto
- turns
- inference
- refactor
- design

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
