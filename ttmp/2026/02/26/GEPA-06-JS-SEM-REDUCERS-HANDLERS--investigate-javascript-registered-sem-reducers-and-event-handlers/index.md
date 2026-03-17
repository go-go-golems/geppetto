---
Title: Investigate JavaScript-Registered SEM Reducers and Event Handlers
Ticket: GEPA-06-JS-SEM-REDUCERS-HANDLERS
Status: active
Topics:
    - geppetto
    - pinocchio
    - go-go-os
    - js-bindings
    - events
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: design-doc/01-javascript-registered-sem-reducers-and-event-handler-architecture.md
      Note: Primary investigation and architecture recommendation
    - Path: design-doc/02-cross-repo-js-sem-runtime-implementation-design.md
      Note: Cross-repo implementation design for shared runtime/binding model
    - Path: reference/01-investigation-diary.md
      Note: Chronological command log and findings
    - Path: scripts/js-sem-reducer-handler-prototype.js
      Note: Prototype showing handler overwrite vs composable model
ExternalSources: []
Summary: 'Clarifies current layer ownership and now includes implemented builder-owned runtime alignment across pinocchio+geppetto plus coexistence strategy for geppetto + timeline bindings in one VM.'
LastUpdated: 2026-03-16T22:08:46-04:00
WhatFor: Define implementation path for JavaScript SEM reducer/handler extensibility
WhenToUse: Use when planning dynamic event reaction/projection features across geppetto/pinocchio/go-go-os
---

# Investigate JavaScript-Registered SEM Reducers and Event Handlers

## Overview

This ticket investigates where SEM projection and event reaction logic live today, and what is required to support JavaScript-registered reducers and handlers.

Canonical ticket home: this ticket was moved from `pinocchio/ttmp` to `geppetto/ttmp` on 2026-03-16 because the main abstraction under discussion is reusable JS runtime and event/binding infrastructure.

## High-Level Conclusions

1. `geppetto` already supports JavaScript event handlers for geppetto event stream (`start`/`partial`/`final`).
2. `pinocchio` currently performs backend SEM projection in Go; backend JS reducer registration is not present.
3. `go-go-os` app runtime already supports JS/TS SEM handler registration, but with single-handler overwrite semantics.
4. GEPA-04 streaming events are now part of the baseline and were incorporated into this analysis.

## Primary Artifacts

1. `design-doc/01-javascript-registered-sem-reducers-and-event-handler-architecture.md`
2. `reference/01-investigation-diary.md`
3. `scripts/js-sem-reducer-handler-prototype.js`

## Tracking

1. See [tasks.md](./tasks.md)
2. See [changelog.md](./changelog.md)
