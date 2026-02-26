---
Title: Migrate Geppetto and Pinocchio to Glazed Facade Packages
Ticket: GP-001-UPDATE-GLAZED
Status: complete
Topics:
    - migration
    - glazed
    - geppetto
    - pinocchio
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Workspace for planning and executing migration from legacy glazed APIs to facade packages across geppetto and pinocchio.
LastUpdated: 2026-02-25T17:31:25.980296724-05:00
WhatFor: Central navigation for analysis, diary, tasks, and evidence artifacts for the migration.
WhenToUse: Use as the entry point for migration implementation and review.
---


# Migrate Geppetto and Pinocchio to Glazed Facade Packages

## Overview

This ticket tracks migration from legacy Glazed APIs (`layers`, `parameters`, `cmds/middlewares`, `ParsedLayers`) to facade packages (`schema`, `fields`, `cmds/sources`, `values`) across:

1. `geppetto/` first
2. `pinocchio/` second

## Primary Documents

- Analysis: `analysis/01-migration-analysis-old-glazed-to-facade-packages-geppetto-then-pinocchio.md`
- Diary: `reference/01-diary.md`

## Evidence Artifacts

Machine-generated inventories and validation logs are stored under:

- `sources/local/`

Key files:

- `sources/local/00-counts.txt`
- `sources/local/09-count-breakdown.txt`
- `sources/local/14-failure-extracts.txt`

## Status

Current status: **active**

## Topics

- migration
- glazed
- geppetto
- pinocchio

## Tasks

See [tasks.md](./tasks.md) for the execution checklist.

## Changelog

See [changelog.md](./changelog.md) for chronological updates.

## Structure

- design/ - Architecture and design documents
- reference/ - Context and operational references
- playbooks/ - Command sequences and procedures
- scripts/ - Temporary utility scripts
- sources/ - Raw evidence and imported sources
- various/ - Working notes
- archive/ - Deprecated/reference-only material
