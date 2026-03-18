---
Title: Remove AllowedTools from Geppetto core and rely on app-owned registry filtering
Ticket: GP-42-REMOVE-ALLOWED-TOOLS
Status: complete
Topics:
    - geppetto
    - architecture
    - tools
    - pinocchio
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed removal ticket for Geppetto core AllowedTools handling. Includes evidence for current core enforcement, downstream app-owned registry filtering patterns, a proposed simplification plan, migration guidance, and an intern-oriented implementation guide."
LastUpdated: 2026-03-17T14:20:00-04:00
WhatFor: "Use this ticket to understand why AllowedTools is duplicated between Geppetto core and app code, what to remove from Geppetto, and how to migrate safely to app-owned registry filtering."
WhenToUse: "Use when implementing, reviewing, or validating the removal of ToolConfig.AllowedTools and related Geppetto core allowlist behavior."
---

# Remove AllowedTools from Geppetto core and rely on app-owned registry filtering

## Overview

This ticket captures the analysis and implementation plan for removing `AllowedTools` from Geppetto core. The main finding is that Geppetto currently enforces tool allowlists twice:

- once by filtering or checking `AllowedTools` inside Geppetto core,
- again by app code that already constructs a filtered registry before starting the loop.

That duplication exists in provider preparation, tool execution, turn metadata, JS bindings, docs, and tests. The ticket recommends simplifying Geppetto so it only operates on the registry it is given. Applications should decide which tools belong in that registry.

The primary deliverable is the detailed intern-oriented design and implementation guide in `design-doc/`, plus a separate Manuel diary in `reference/`.

## Key Links

- Primary guide: `design-doc/01-remove-allowedtools-from-geppetto-core-design-and-implementation-guide.md`
- Diary: `reference/01-manuel-investigation-diary.md`
- Surface inventory script: `scripts/01-allowed-tools-surface-inventory.sh`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **complete**

Analysis and delivery are complete. The ticket now contains a full removal guide, evidence inventory, diary, task list, and reMarkable-ready bundle inputs.

## Scope

In scope:

- `tools.ToolConfig.AllowedTools` in Geppetto core.
- `engine.ToolConfig.AllowedTools` and its turn-serialization consequences.
- Provider-side tool filtering in Geppetto engines.
- Executor-side allowlist checks in Geppetto tool execution.
- JS builder options and examples that expose core `allowedTools`.
- Documentation and tests that encode Geppetto-owned tool allowlisting.

Out of scope:

- App-owned runtime concepts such as Pinocchio `ComposedRuntime.AllowedTools`.
- Application profile logic that computes which tools should be allowed.
- Generic registry-building abstractions for all apps.

## Deliverables

- A detailed design and implementation guide for a new intern.
- A chronological diary capturing evidence gathering and decisions.
- A small ticket-local script to inventory `AllowedTools`-related surfaces.
- Validation through `docmgr doctor`.
- Bundle upload to reMarkable.

## Tasks

See [tasks.md](./tasks.md) for the task checklist and completion status.

## Changelog

See [changelog.md](./changelog.md) for ticket milestones and delivery notes.

## Structure

- `design-doc/` contains the main analysis and implementation guide.
- `reference/` contains the diary.
- `scripts/` contains ticket-local support tooling for future continuation.
- `archive/`, `design/`, `playbooks/`, `sources/`, and `various/` remain available for follow-up work if implementation proceeds.
