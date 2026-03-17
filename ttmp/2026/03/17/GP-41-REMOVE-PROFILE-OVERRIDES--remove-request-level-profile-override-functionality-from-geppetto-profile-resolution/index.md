---
Title: Remove request-level profile override functionality from Geppetto profile resolution
Ticket: GP-41-REMOVE-PROFILE-OVERRIDES
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - pinocchio
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed removal ticket for Geppetto request-level profile overrides. Includes evidence that downstream products primarily switch whole profiles, a proposed API simplification plan, migration guidance, a runtime glossary, and intern-oriented implementation guides."
LastUpdated: 2026-03-17T17:07:00-04:00
WhatFor: "Use this ticket to understand why request-level profile overrides add complexity today, what concrete surfaces need to change to remove them, how the overloaded Runtime* vocabulary is used across Geppetto and Pinocchio, and how to execute the cleanup safely."
WhenToUse: "Use when implementing, reviewing, or validating the removal of request_overrides and override policy machinery from Geppetto profile resolution, or when onboarding someone into the runtime-related code paths."
---

# Remove request-level profile override functionality from Geppetto profile resolution

## Overview

This ticket captures the analysis and implementation plan for removing request-level profile overrides from Geppetto profile resolution. The core finding is that the actual product usage pattern is coarse-grained profile switching, while the override machinery adds policy, parsing, validation, JS binding, and HTTP contract complexity that current downstream apps do not rely on as a real runtime feature.

The primary deliverable is the detailed intern-oriented design and implementation guide in `design-doc/`, plus a separate Manuel diary in `reference/`.

## Key Links

- Primary guide: `design-doc/01-remove-geppetto-request-level-profile-overrides-design-and-implementation-guide.md`
- Runtime glossary: `design-doc/02-runtime-glossary-across-geppetto-and-pinocchio.md`
- Diary: `reference/01-manuel-investigation-diary.md`
- Surface inventory script: `scripts/01-override-surface-inventory.sh`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

Analysis and delivery are complete, and Phase 1 implementation is now in progress in Geppetto core. The ticket now contains the original removal guide, evidence inventory, diary, task list, reMarkable-ready bundle inputs, and the first round of concrete code changes that remove `RequestOverrides` from the Geppetto resolution path.

## Scope

In scope:

- Geppetto profile service request override plumbing.
- Geppetto profile policy fields used only for request override control.
- JS bindings and examples that expose request overrides.
- Pinocchio and GEC-RAG integration cleanup required after Geppetto simplification.
- Documentation and tests that currently encode override behavior.

Out of scope:

- Removing profile-based `runtime.step_settings_patch`.
- Replacing profiles themselves with a new runtime system.
- Changing downstream profile switching UX.

## Why this ticket exists

The relevant Geppetto resolution path currently accepts `ResolveInput.RequestOverrides`, merges override policy, applies normalized override keys, and mutates the resolved runtime before final step settings are produced. This is implemented in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go` and `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go`.

The concrete downstream products mostly do something simpler:

- choose a profile,
- resolve the effective runtime,
- apply the resolved profile runtime wholesale.

That simpler usage is visible in Pinocchio, GEC-RAG, and Temporal Relationships. Temporal Relationships does not expose request overrides on its HTTP surface at all.

## Deliverables

- A detailed design and implementation guide for a new intern.
- A chronological diary capturing evidence gathering and decisions.
- A small ticket-local script to inventory override-related surfaces.
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
