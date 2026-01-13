---
Title: Fix Version Number Detection
Ticket: 004-FIX-VERSION-NUMBER
Status: complete
Topics:
    - versioning
    - git
    - releases
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/.github/workflows/release.yml
      Note: GitHub Actions release workflow
    - Path: geppetto/.github/workflows/tag-release-notes.yml
      Note: Artifact-free GitHub Release creation on tag push + manual backfill
    - Path: geppetto/.goreleaser.yaml
      Note: GoReleaser config (currently does not specify main package)
    - Path: geppetto/Makefile
      Note: Uses svu current in release target
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-05T18:01:01.831743709-05:00
WhatFor: ""
WhenToUse: ""
---




# Fix Version Number Detection

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- versioning
- git
- releases

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
