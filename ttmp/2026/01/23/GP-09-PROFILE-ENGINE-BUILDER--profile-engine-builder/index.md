---
Title: Profile Engine Builder
Ticket: GP-09-PROFILE-ENGINE-BUILDER
Status: complete
Topics:
    - architecture
    - backend
    - go
    - inference
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Design analysis to extract webchat engine/profile policy out of Router behind a request-facing BuildEngineFromReq interface and a ProfileEngineBuilder implementation.
LastUpdated: 2026-02-25T17:31:30.45797984-05:00
WhatFor: Track and document the refactor plan for making webchat engine building a first-class abstraction (not Router glue), with special focus on profile-based engine construction.
WhenToUse: Use when implementing GP-09, reviewing engine-builder boundaries, or porting similar patterns across go-go-mento/pinocchio/moments.
---


# Profile Engine Builder

## Overview

This ticket captures a deep analysis and proposal to further extract webchat engine/profile logic out of Router. The target outcome is that Router delegates to a request-facing `BuildEngineFromReq` interface, which then uses a profile-based engine builder (`profileEngineBuilder`) to produce the engine/sink/config deterministically.

## Key Links

- Analysis: `analysis/01-extract-profile-engine-builder-out-of-router.md`
- Diary: `reference/01-diary.md`

## Status

Current status: **active**

## Topics

- architecture
- backend
- go
- inference

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
