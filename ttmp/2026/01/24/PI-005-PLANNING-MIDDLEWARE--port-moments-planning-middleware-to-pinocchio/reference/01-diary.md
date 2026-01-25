---
Title: Diary
Ticket: PI-005-PLANNING-MIDDLEWARE
Status: active
Topics:
    - backend
    - middleware
    - moments
    - pinocchio
    - planning
    - protobuf
    - webchat
    - websocket
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: moments
      Note: Source project to analyze
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Profiles/middleware wiring for web-chat
    - Path: pinocchio/pkg/webchat/router.go
      Note: Where planning stubs were gated; will wire real middleware
    - Path: pinocchio/proto/sem/middleware/planning.proto
      Note: SEM planning/execution event schemas
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-24T21:05:52.656865135-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Keep a detailed, step-by-step diary for PI-005-PLANNING-MIDDLEWARE: analyze the planning middleware in `moments` and port the real behavior into Pinocchio so the webchat planning widget is driven by actual middleware/LLM output (not stubs).

## Context

Pinocchio currently supports rendering planning UI (`planning.*` + `execution.*` SEM events) and can hydrate planning snapshots via `sem.timeline.*`, but the backend previously emitted **stub planning events** that were identical every run. Those stubs are now gated behind `--emit-planning-stubs` and default off (PI-004).

This ticket ports the *real planning middleware* behavior from `moments` into Pinocchio so:
- planning is only present when enabled/configured
- emitted planning events carry meaningful, model-derived content
- the timeline projector can persist/hydrate planning snapshots consistently

## Step 1: Create PI-005 ticket scaffold

Created the new ticket, seeded an initial task breakdown, and set up a diary + a design doc to capture the analysis (what moments does) and the port plan (what we implement in pinocchio).

### Prompt Context

**User prompt (verbatim):** "analyze planning middleware in moments (i think?) and port into pinocchio. Add tasks, check them off, commit, etc..."

**Assistant interpretation:** Create tracked work, analyze `moments` planning middleware, then implement the equivalent middleware and wiring in the Pinocchio webchat backend, committing incrementally and checking tasks off.

**Inferred user intent:** Get “Moments-grade” planning UX in Pinocchio without hacks: real SEM planning streams, compatible with hydration and the React planning widget.

**Commit (docs):** pending

### What I did
- Created docmgr ticket `PI-005-PLANNING-MIDDLEWARE`.
- Added docs:
  - `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/design-doc/01-moments-planning-middleware-analysis-port-plan.md`
  - `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/reference/01-diary.md`
- Added initial task list in `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/tasks.md`.
- Related key files/dirs for review context (moments source, Pinocchio planning protos, web-chat wiring).

### Why
- This work spans multiple modules (moments + pinocchio) and needs durable traceability (tasks + diary + commits).

### What worked
- Ticket scaffold and tasks are in place; next step is code archaeology in `moments`.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Review ticket setup:
  - `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/index.md`
  - `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/tasks.md`
