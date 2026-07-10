---
Title: Geppetto history for last three months
Ticket: HIST-2026-03-14-TO-2026-06-14
Status: active
Topics:
    - git-history
    - docmgr
    - research
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: "SQLite-backed git and docmgr history for Geppetto from 2026-03-14 through 2026-06-14."
LastUpdated: 2026-06-14T12:55:00-04:00
WhatFor: "Use this ticket to inspect or regenerate the last-three-month project history from git commits and docmgr metadata."
WhenToUse: "Use when reviewing recent project direction, looking for hot paths, or needing a reproducible SQLite-backed history report."
---

# Geppetto history for last three months

## Overview

This ticket contains a reproducible, SQLite-backed history of Geppetto activity from 2026-03-14 through 2026-06-14. It combines git commits, commit file statistics, docmgr ticket metadata, document metadata, and changelog entries into `various/history.sqlite`, then renders the narrative report in `analysis/01-last-three-months-history.md`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- git-history
- docmgr
- research

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
