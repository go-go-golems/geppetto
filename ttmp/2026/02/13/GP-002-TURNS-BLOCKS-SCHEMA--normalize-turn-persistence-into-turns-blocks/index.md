---
Title: Normalize Turn Persistence into Turns + Blocks
Ticket: GP-002-TURNS-BLOCKS-SCHEMA
Status: active
Topics:
    - backend
    - persistence
    - turns
    - architecture
    - migration
    - pinocchio
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Deferred ticket to design and later implement normalized turns + blocks sqlite persistence (no backward compatibility) for block-level querying and dedupe.
LastUpdated: 2026-02-13T18:30:00-05:00
WhatFor: Track design and future implementation work for replacing payload-string turn snapshots with normalized storage tables.
WhenToUse: Use when planning or executing GP-002 schema migration and backfill.
---

# Normalize Turn Persistence into Turns + Blocks

## Overview

This ticket captures a deferred storage redesign: replace payload-string turn snapshots with normalized `turns + blocks` persistence (with an ordered relation table), keyed by `block_id + content_hash` to avoid collisions.

This work is intentionally deferred from GP-001 so debug UI migration can ship without taking on schema/backfill risk.

## Key Links

- Planning analysis:
  - `planning/01-turns-and-blocks-normalized-persistence-analysis-deferred.md`
- Tasks:
  - `tasks.md`
- Changelog:
  - `changelog.md`

## Status

Current status: **active**

## Topics

- backend
- persistence
- turns
- architecture
- migration
- pinocchio

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
