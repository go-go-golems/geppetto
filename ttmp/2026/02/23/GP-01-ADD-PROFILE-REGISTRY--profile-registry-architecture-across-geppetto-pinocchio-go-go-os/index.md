---
Title: Profile Registry Architecture Across Geppetto Pinocchio Go-Go-OS
Ticket: GP-01-ADD-PROFILE-REGISTRY
Status: active
Topics:
    - architecture
    - geppetto
    - pinocchio
    - chat
    - inference
    - persistence
    - migration
    - backend
    - frontend
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/planning/01-profileregistry-architecture-and-migration-plan.md
      Note: Primary architecture proposal and migration plan
    - Path: geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/planning/03-phase-0-rollout-guardrails-and-compatibility-plan.md
      Note: Phase 0 guardrails, compatibility matrix, and rollout posture
    - Path: geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/playbook/01-db-profile-store-ops-notes-and-gp-01-release-notes.md
      Note: DB-backed operations notes and release notes
    - Path: geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/reference/01-diary.md
      Note: Detailed diary of actions
    - Path: geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md
      Note: Granular phased implementation backlog with task IDs
ExternalSources: []
Summary: Cross-repo architecture ticket defining a reusable ProfileRegistry design and migration plan for geppetto, pinocchio web-chat, and go-go-os profile UX.
LastUpdated: 2026-02-23T13:57:00-05:00
WhatFor: Drive implementation of registry-backed profile resolution, profile CRUD APIs, and profile-aware web clients.
WhenToUse: Use this index to navigate the primary planning doc and detailed implementation diary.
---



# Profile Registry Architecture Across Geppetto Pinocchio Go-Go-OS

## Overview

This ticket defines how to replace flag-heavy AI runtime configuration with reusable profile registries. The work covers geppetto (registry core and middleware integration), pinocchio (runtime resolution and profile APIs), and go-go-os (profile listing/selection/create flows for chat clients).

Current status: analysis complete and implementation proposal delivered.

## Key Links

- [ProfileRegistry Architecture and Migration Plan](./planning/01-profileregistry-architecture-and-migration-plan.md)
- [Phase 0 Rollout Guardrails and Compatibility Plan](./planning/03-phase-0-rollout-guardrails-and-compatibility-plan.md)
- [DB Profile Store Ops Notes and GP-01 Release Notes](./playbook/01-db-profile-store-ops-notes-and-gp-01-release-notes.md)
- [Implementation Diary](./reference/01-diary.md)
- **Related Files**: See frontmatter RelatedFiles field for the key source files used in analysis.
- **External Sources**: N/A (analysis based on local codebase).

## Status

Current status: **active**

## Topics

- architecture
- geppetto
- pinocchio
- chat
- inference
- persistence
- migration
- backend
- frontend

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- planning/ - Long-form architecture proposal and migration strategy
- playbook/ - Operations and release communication runbooks
- reference/ - Detailed diary and analysis artifacts
- scripts/ - Temporary tooling (none added yet)
- archive/ - Deprecated or historical material
