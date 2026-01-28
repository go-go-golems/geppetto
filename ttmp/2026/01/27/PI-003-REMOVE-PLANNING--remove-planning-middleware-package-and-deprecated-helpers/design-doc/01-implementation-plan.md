---
Title: Implementation Plan
Ticket: PI-003-REMOVE-PLANNING
Status: active
Topics:
    - refactor
    - cleanup
    - webchat
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/main.go
      Note: |-
        planning profile currently configured
        Planning profile wiring
    - Path: ../../../../../../../pinocchio/pkg/middlewares/planning/lifecycle_engine.go
      Note: |-
        Planning middleware core (to remove)
        Planning lifecycle engine to remove
    - Path: ../../../../../../../pinocchio/pkg/middlewares/planning/middleware_keys.go
      Note: |-
        Planning middleware keys (to remove)
        Planning keys to remove
    - Path: ../../../../../../../pinocchio/pkg/webchat/engine.go
      Note: |-
        planningConfigFromAny + lifecycle engine wiring
        Planning wiring and helper
ExternalSources: []
Summary: Plan to remove planning middleware package and clean dependent wiring.
LastUpdated: 2026-01-27T20:35:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Implementation Plan — Remove Planning Middleware

## Executive Summary

We no longer use planning in pinocchio. This change removes the entire planning middleware package and all wiring points that reference it (webchat engine composition, profile presets, and helper functions). The goal is a clean build with no planning features or dead code.

## Problem Statement

The `pinocchio/pkg/middlewares/planning` package adds a planning lifecycle engine, prompt rewriting, and UI‑specific events. It is currently unused, but still referenced by webchat engine composition and profile definitions. Retaining it adds unnecessary complexity and risk, while new work is focused on simpler runtime behavior.

## Proposed Solution

1. Remove the planning middleware package and its tests.
2. Remove all code paths that reference planning (e.g., `planningConfigFromAny`, `NewLifecycleEngine`, `KeyDirective` usage, planning profile in webchat CLI).
3. Update documentation and any example config that references planning.
4. Run tests and lint to confirm no residual dependencies.

## Design Decisions

- **Removal over deprecation**: We are explicitly not supporting planning; keeping dead code is counterproductive.
- **Keep API surface small**: Replace planning-related options with no‑ops only if necessary for compatibility. Prefer removal.
- **Scope to pinocchio**: This is focused on pinocchio’s planning package; other systems remain unaffected.

## Alternatives Considered

- **Deprecate but keep**: rejected because it retains complexity with no usage.
- **Keep only keys/config**: rejected; still confusing and implies support.

## Implementation Plan

1. Inventory all planning references (imports, profiles, docs).
2. Remove `pinocchio/pkg/middlewares/planning` package.
3. Remove planning wiring in `pinocchio/pkg/webchat/engine.go` and delete `planningConfigFromAny` helper.
4. Remove planning profile from `pinocchio/cmd/web-chat/main.go`.
5. Update docs that mention planning (if any).
6. Run `go test ./...` and `golangci-lint` if applicable.
