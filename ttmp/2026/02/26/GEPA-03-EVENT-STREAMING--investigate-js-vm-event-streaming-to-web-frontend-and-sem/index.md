---
Title: Investigate JS VM Event Streaming to Web Frontend and SEM
Ticket: GEPA-03-EVENT-STREAMING
Status: active
Topics:
    - geppetto
    - pinocchio
    - js-bindings
    - events
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: design-doc/01-gepa-event-streaming-architecture-investigation.md
      Note: Primary deep analysis
    - Path: reference/01-investigation-diary.md
      Note: Chronological command/finding log
    - Path: scripts/sem-envelope-prototype.js
      Note: Prototype SEM envelope validation experiment
ExternalSources: []
Summary: 'Investigation concludes that engine events already stream via Pinocchio SEM/websocket, while GEPA JS script event streaming requires a new host emission API and bridge integration.'
LastUpdated: 2026-03-16T22:08:46-04:00
WhatFor: Track the architecture and execution plan for script-level event streaming into frontend
WhenToUse: Use when implementing GEPA->SEM streaming capabilities or reviewing event pipeline design decisions
---

# Investigate JS VM Event Streaming to Web Frontend and SEM

## Overview

This ticket investigates whether scripts running in GEPA JS VM (for example `exp-11-coaching-dataset-generator.js`) can stream events to the web frontend using existing Pinocchio SEM infrastructure.

Canonical ticket home: this ticket was moved from `pinocchio/ttmp` to `geppetto/ttmp` on 2026-03-16 because the missing piece is a Geppetto-side JS host emission API and runtime bridge.

## Final Verdict

1. Engine/geppetto event streaming to frontend: **already works today**.
2. GEPA script-level structured event streaming: **not wired yet**.
3. Required implementation is clear and feasible: add source-side `emitEvent` API + bridge + optional projection registration.

## Primary Documents

1. `design-doc/01-gepa-event-streaming-architecture-investigation.md`
2. `reference/01-investigation-diary.md`
3. `scripts/sem-envelope-prototype.js`

## Tasks and Changelog

1. See [tasks.md](./tasks.md)
2. See [changelog.md](./changelog.md)

## Structure

- `design/` and `design-doc/`: architecture deliverables
- `reference/`: diary and operational notes
- `scripts/`: experiments and helper scripts
- `sources/`: external artifacts if needed
- `archive/`: deprecated ticket artifacts
