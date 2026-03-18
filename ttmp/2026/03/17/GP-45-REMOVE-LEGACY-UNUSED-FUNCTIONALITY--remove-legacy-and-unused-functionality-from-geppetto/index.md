---
Title: Remove legacy and unused functionality from geppetto
Ticket: GP-45-REMOVE-LEGACY-UNUSED-FUNCTIONALITY
Status: active
Topics:
    - geppetto
    - architecture
    - cleanup
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go
      Note: Normalizes multiple legacy runtime metadata shapes.
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/run_with_result.go
      Note: Mirrors legacy scalar inference keys during migration.
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_registry_source.go
      Note: Explicit migration shim around older profile-file flag loading.
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/adapters.go
      Note: Thin wrappers with no meaningful production usage found.
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions.go
      Note: Contains extension normalization machinery that appears lightly integrated.
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/middleware_extensions.go
      Note: Middleware extension projection helpers appear test-only from current repo usage.
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/stack_trace.go
      Note: Always-on stack trace payload generation for debug metadata.
ExternalSources: []
Summary: Broad cleanup ticket that inventories legacy support paths, unused machinery, and complexity hotspots in geppetto, then turns them into a phased removal plan with evidence and intern-friendly system explanations.
LastUpdated: 2026-03-18T01:21:00-04:00
WhatFor: Use this ticket to plan and stage the removal of low-value compatibility layers and underused machinery from geppetto.
WhenToUse: Use when deciding what cleanup work to pursue after RuntimeKeyFallback removal or when onboarding an engineer into geppetto cleanup work.
---

# Remove legacy and unused functionality from geppetto

## Overview

This ticket packages the broader findings from the `RuntimeKeyFallback` investigation into one cleanup-oriented design set. The focus is not only on clearly obsolete compatibility code, but also on machinery that appears detached from current application value: migration shims that have outlived the migration, output-shape normalizers that preserve old schemas indefinitely, and subsystems that look more elaborate than their demonstrated usage.

The design doc is intentionally detailed for a new intern. It explains the relevant parts of the system first, then categorizes each cleanup candidate by confidence and risk:

- clearly backward-compatibility support,
- likely unused or only test-used machinery,
- probably over-complex but still potentially valuable,
- documentation drift that keeps removed concepts alive.

## Key Links

- Design doc: `design-doc/01-legacy-and-unused-geppetto-functionality-cleanup-analysis-and-implementation-guide.md`
- Diary: `reference/01-investigation-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Current Status

Current status: **active**

The low-risk hard cuts are now landed. GP-45 has removed the dead no-op profile flag bridge, the dead JS `engines.fromProfile` surface, the unused profile adapter helpers, the leftover `DeleteProfile` / `SetDefaultProfile` store API, and the stale docs/JS typings that still taught removed configuration knobs. The remaining work is the compatibility-heavy metadata cleanup plus the higher-risk extension and provenance review.

## Scope

In scope:

- Legacy runtime metadata normalization.
- Legacy scalar inference metadata mirroring.
- Migration shims around older profile loading paths.
- Unused adapter helpers.
- Lightly integrated extension normalization machinery.
- Test-only middleware-extension projection helpers.
- Always-on debug/provenance payload generation.
- Stale docs that still teach removed concepts.

Out of scope:

- Unreviewed deletion of extension schemas if downstream users exist.
- Re-architecting the entire profile system.
- Removing functionality that has active cross-repo consumers without first confirming usage.

## Tasks

See [tasks.md](./tasks.md) for the phased cleanup checklist and review criteria.

## Changelog

See [changelog.md](./changelog.md) for the investigation record and document milestones.

## Structure

- `design-doc/`: broad cleanup analysis and phased implementation guide
- `reference/`: diary and reviewer continuation notes
- `scripts/`: reserved for future ticket-local search or validation helpers
- `archive/`: reserved for superseded inventories or split-out follow-up docs
