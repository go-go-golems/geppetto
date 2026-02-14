---
Title: Add Debug UI
Ticket: GP-001-ADD-DEBUG-UI
Status: completed
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
LastUpdated: 2026-02-14T16:05:00-05:00
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

Current status: **completed**

Progress snapshot:

- Backend canonical debug APIs, live read models, and offline artifact/sqlite run viewers implemented.
- Migrated debug UI fully lives in pinocchio web workspace with Storybook and canonical `/api/debug/*` wiring.
- Legacy `web-agent-example/cmd/web-agent-debug` harness/UI removed.
- Post-port regressions (URL history loop, parsed turn block decoding) fixed and regression-tested.
- Ticket docs/diary/changelog fully updated, with milestone bundles uploaded and verified in reMarkable cloud.

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
