---
Title: Correlation Contract and Debug UI Migration Plan
Ticket: PI-014-CORRELATION-CONTRACT-DEBUG-UI
Status: active
Topics:
    - backend
    - middleware
    - turns
    - events
    - frontend
    - websocket
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Implement deterministic inference-scoped event-to-snapshot correlation, tracing integration details, endpoint migration to /debug-only APIs, and separate debug app scaffolding.
LastUpdated: 2026-02-07T11:25:00-05:00
WhatFor: Planning and execution hub for Critical 1 and associated follow-up decisions from PI-013.
WhenToUse: Use when implementing debug correlation and tracing architecture or reviewing migration progress.
---

# Correlation Contract and Debug UI Migration Plan

## Overview

This ticket defines implementation work for deterministic correlation between SEM events and turn snapshots, clarifies tracing integration, applies debug endpoint migration decisions, and tracks separate debug app setup.

## Key Links

- Analysis: `analysis/01-correlation-contract-tracing-and-migration-implementation-plan.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
