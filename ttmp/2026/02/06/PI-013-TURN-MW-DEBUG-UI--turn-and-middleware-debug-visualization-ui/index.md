---
Title: Turn and Middleware Debug Visualization UI
Ticket: PI-013-TURN-MW-DEBUG-UI
Status: complete
Topics:
    - websocket
    - middleware
    - turns
    - events
    - frontend
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - local:ui-design-turn.md
Summary: Build a dedicated debugging UI to visualize turn snapshots, middleware mutations, event flows, structured sink extraction, and timeline projection behavior, including reviewed architecture guidance, post-review design decisions, and a full frontend code review audit.
LastUpdated: 2026-02-25T18:07:24.153384994-05:00
WhatFor: Central index for PI-013 documentation and handoff artifacts.
WhenToUse: Use when orienting to this ticket or starting implementation/design work.
---



# Turn and Middleware Debug Visualization UI

## Overview

PI-013 defines a dedicated developer-facing web UI for inspecting how turns and blocks are transformed across middleware/tool-loop phases and how those transformations propagate through events, SEM translation, timeline projection, and hydration.

The ticket currently contains:

- A detailed analysis/specification doc for design and implementation handoff.
- A designer primer explaining turns/blocks/middlewares/structured events in plain language.
- A designer-provided architecture and implementation proposal.
- A deep engineering review of the proposal with corrected implementation guidance.
- A full frontend React code review report of the PI-013 UX implementation commits.
- A frequent, step-by-step implementation diary plus a separate review diary artifact.

## Key Links

- Analysis: `analysis/01-turn-and-middleware-debug-ui-requirements-and-ux-specification.md`
- Designer primer: `analysis/02-designer-primer-turns-blocks-middlewares-and-structured-events.md`
- Architecture proposal: `analysis/05-architecture-and-implementation-plan-for-debug-ui.md`
- Engineering review: `analysis/06-engineering-review-of-architecture-and-implementation-plan-for-debug-ui.md`
- Frontend code review report: `analysis/07-ux-frontend-react-code-review-report.md`
- Follow-up implementation ticket: `../PI-014-CORRELATION-CONTRACT-DEBUG-UI--correlation-contract-and-debug-ui-migration-plan/index.md`
- EventStore postmortem ticket: `../PI-015-EVENTSTORE-POSTMORTEM--eventstore-for-postmortem-debug-mode/index.md`
- SEM/event performance ticket: `../PI-016-SEM-EVENT-PERF--sem-and-event-pipeline-performance-deep-dive/index.md`
- Implementation diary: `reference/01-diary.md`
- Frontend code review diary (separate): `reference/02-frontend-code-review-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- websocket
- middleware
- turns
- events
- frontend

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
