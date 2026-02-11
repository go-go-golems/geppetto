---
Title: Oak + Git history → SQLite PR analysis (geppetto)
Ticket: MO-005-OAK-GIT-HISTORY
Status: active
Topics:
    - infrastructure
    - tools
    - geppetto
    - go
    - persistence
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: oak-git-db/README.md
      Note: Standalone repo README
    - Path: oak-git-db/cmd/oakgitdb/main.go
      Note: Standalone CLI entrypoint
    - Path: oak-git-db/docs/design.md
      Note: Standalone design documentation
    - Path: oak-git-db/docs/implementation.md
      Note: Standalone implementation documentation
    - Path: oak-git-db/docs/usage.md
      Note: Standalone usage documentation
    - Path: oak-git-db/pkg/oakgitdb/builder.go
      Note: Standalone pipeline + schema
ExternalSources:
    - local:oakgitdb usage (markdown).md
    - local:oakgitdb design (markdown).md
    - local:oakgitdb implementation (markdown).md
Summary: ""
LastUpdated: 2026-01-21T17:10:14.100094608-05:00
WhatFor: ""
WhenToUse: ""
---





# Oak + Git history → SQLite PR analysis (geppetto)

## Overview

Build a PR-focused SQLite database for `geppetto/` that combines:

- git PR facts (commits, changed files, rename/copy metadata)
- oak (tree-sitter) matches for code definitions (base + head snapshots)
- Go typed analysis for head snapshot symbols + call edges

Primary output:

- `various/pr-vs-origin-main.db`

Tool + detailed docs have been moved into the standalone repo:

- `oak-git-db/`

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Design doc**: `planning/01-design-oak-git-history-sqlite-database-for-pr-vs-origin-main.md`
- **Diary**: `reference/01-diary.md`

## Status

Current status: **active**

## Topics

- infrastructure
- tools
- geppetto
- go
- persistence

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
