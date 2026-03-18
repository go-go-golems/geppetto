---
Title: Remove request-level profile override functionality from Geppetto profile resolution
Ticket: GP-41-REMOVE-PROFILE-OVERRIDES
Status: complete
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
LastUpdated: 2026-03-17T15:55:00-04:00
WhatFor: "Use this ticket to understand why request-level profile overrides add complexity today, why the implementation pivoted further toward a read-only registry architecture, what concrete surfaces need to change, and how to execute the cleanup safely."
WhenToUse: "Use when implementing, reviewing, or validating the removal of request_overrides, profile policy, and registry mutation APIs from the Geppetto profile stack, or when onboarding someone into the runtime-related code paths."
---

# Remove request-level profile override functionality from Geppetto profile resolution

## Overview

This ticket captures the analysis and implementation plan for removing request-level profile overrides from Geppetto profile resolution. The core finding is that the actual product usage pattern is coarse-grained profile switching, while the override machinery adds policy, parsing, validation, JS binding, and HTTP contract complexity that current downstream apps do not rely on as a real runtime feature.

The primary deliverables are the detailed intern-oriented design guides in `design-doc/`, including the new read-only registry implementation plan, plus a separate Manuel diary in `reference/`.

## Key Links

- Primary guide: `design-doc/01-remove-geppetto-request-level-profile-overrides-design-and-implementation-guide.md`
- Runtime glossary: `design-doc/02-runtime-glossary-across-geppetto-and-pinocchio.md`
- Read-only registry implementation plan: `design-doc/03-read-only-profile-registry-pivot-implementation-plan.md`
- Diary: `reference/01-manuel-investigation-diary.md`
- Surface inventory script: `scripts/01-override-surface-inventory.sh`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **complete**

Analysis and delivery are complete, and implementation is now following a broader read-only registry pivot. The ticket now contains the original removal guide, runtime glossary, a dedicated pivot implementation plan, the diary, an updated task board, reMarkable-ready bundle inputs, and the first round of concrete code changes that remove `RequestOverrides` from the Geppetto resolution path.

## Scope

In scope:

- Geppetto profile service request override plumbing.
- Geppetto profile policy fields used only for request override control.
- Geppetto registry mutation APIs that are no longer justified by real product usage.
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

That simpler usage is visible in Pinocchio, GEC-RAG, and Temporal Relationships. Temporal Relationships does not expose request overrides on its HTTP surface at all. During implementation, that same evidence also supported a second conclusion: the registry layer itself should probably be read-only, because current downstream product usage does not justify the embedded CRUD/defaulting service APIs either.

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
