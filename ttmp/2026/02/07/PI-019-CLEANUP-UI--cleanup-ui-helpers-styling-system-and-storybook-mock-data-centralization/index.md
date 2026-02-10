---
Title: Cleanup UI helpers, styling system, and Storybook mock data centralization
Ticket: PI-019-CLEANUP-UI
Status: complete
Topics:
    - frontend
    - architecture
    - middleware
    - websocket
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Plan ticket for reducing debug UI code size by unifying duplicated helpers, extracting reusable CSS design system layers, and centralizing Storybook mock data generation.
LastUpdated: 2026-02-07T20:32:00-05:00
WhatFor: Central coordination document for PI-019 cleanup implementation and closeout evidence.
WhenToUse: Use when reviewing PI-019 delivery scope, validation evidence, and resulting architecture guardrails.
---

# Cleanup UI helpers, styling system, and Storybook mock data centralization

## Overview

PI-019 is a cleanup/planning ticket focused on reducing frontend code size and maintenance burden in the PI-013 debug UI React codebase.

Scope:

- unify duplicated presentation/format helpers across components,
- extract embedded component CSS into reusable design-system style layers,
- centralize Storybook mock data generation and MSW handler composition,
- align cleanup approach with reusable styling patterns used in `pinocchio` webchat.

Current status: implementation complete; all phases and validation gates closed.

## Key Links

- Implementation plan: `analysis/01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup.md`
- Diary: `reference/01-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`
- Reference styling pattern (pinocchio): `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/webchat/styles/`

## Status

Current status: **complete**

## Topics

- frontend
- architecture
- middleware
- websocket

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
