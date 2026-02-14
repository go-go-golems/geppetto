---
Title: Add Debug UI
Ticket: GP-001-ADD-DEBUG-UI
Status: active
Topics:
    - frontend
    - geppetto
    - migration
    - conversation
    - events
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/llm-runner/serve.go
      Note: Current geppetto serve entrypoint assessed for integration and cleanup
    - Path: pinocchio/pkg/webchat/router.go
      Note: Primary runtime API surface assessed for migration
    - Path: web-agent-example/cmd/web-agent-debug/web/src/api/debugApi.ts
      Note: Primary frontend contract expectations assessed for migration
ExternalSources: []
Summary: Ticket workspace for migrating web-agent debug UI into pinocchio as a first-class visualization/debugging tool and reusable React/RTK inspector package stack for offline (filesystem/sqlite) and live level-2 modes.
LastUpdated: 2026-02-13T18:06:00-05:00
WhatFor: Track planning, decisions, and execution details for GP-001 no-backwards-compatibility migration.
WhenToUse: Use when implementing or reviewing the GP-001 migration and cleanup work.
---


# Add Debug UI

## Overview

This ticket captures the migration strategy to move the debug UI currently in `web-agent-example` into `pinocchio` as:

- a pinocchio-owned visualization and debugging application,
- a reusable React/RTK package foundation for conversation + timeline projection inspectors in offline (filesystem/sqlite) and live level-2 modes,
- frontend contracts that preserve and use response metadata envelopes (not only `items` arrays),
- and a no-backwards-compatibility cutover (legacy surfaces can be removed).

## Key Links

- Planning analysis:
  - `planning/01-web-agent-debug-ui-migration-analysis-for-geppetto.md`
- Diary:
  - `reference/01-diary.md`
- Tasks:
  - `tasks.md`
- Changelog:
  - `changelog.md`

## Status

Current status: **active**

Progress snapshot:

- Ticket and docs created.
- Deep migration analysis completed and stored in planning doc.
- Diary populated with step-by-step execution details and command outcomes.
- reMarkable upload completed and verified in cloud storage.

## Topics

- frontend
- geppetto
- migration
- conversation
- events
- architecture

## Structure

- design/ - Architecture and design documents
- reference/ - Diaries, contracts, context summaries
- playbooks/ - Command sequences and validation procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated/reference artifacts
