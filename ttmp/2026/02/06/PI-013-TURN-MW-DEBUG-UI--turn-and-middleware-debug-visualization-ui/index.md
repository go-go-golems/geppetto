---
Title: Turn and Middleware Debug Visualization UI
Ticket: PI-013-TURN-MW-DEBUG-UI
Status: active
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
ExternalSources: []
Summary: Build a dedicated debugging UI to visualize turn snapshots, middleware mutations, event flows, structured sink extraction, and timeline projection behavior.
LastUpdated: 2026-02-06T21:45:00-05:00
WhatFor: Central index for PI-013 documentation and handoff artifacts.
WhenToUse: Use when orienting to this ticket or starting implementation/design work.
---

# Turn and Middleware Debug Visualization UI

## Overview

PI-013 defines a dedicated developer-facing web UI for inspecting how turns and blocks are transformed across middleware/tool-loop phases and how those transformations propagate through events, SEM translation, timeline projection, and hydration.

The ticket currently contains:

- A detailed analysis/specification doc for design and implementation handoff.
- A designer primer explaining turns/blocks/middlewares/structured events in plain language.
- A frequent, step-by-step diary of the work completed so far.

## Key Links

- Analysis: `analysis/01-turn-and-middleware-debug-ui-requirements-and-ux-specification.md`
- Designer primer: `analysis/02-designer-primer-turns-blocks-middlewares-and-structured-events.md`
- Diary: `reference/01-diary.md`
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
