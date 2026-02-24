---
Title: Phase 0 Rollout Guardrails and Compatibility Plan
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
DocType: planning
Intent: long-term
Owners: []
RelatedFiles:
  - Path: geppetto/pkg/profiles/service.go
    Note: Canonical profile resolution, precedence, policy, and fingerprint semantics.
  - Path: geppetto/pkg/sections/sections.go
    Note: Final middleware integration path (always registry-backed, no env gating).
  - Path: pinocchio/cmd/web-chat/main.go
    Note: Server-side profile registry bootstrapping and Glazed profile store settings.
  - Path: pinocchio/cmd/web-chat/profile_policy.go
    Note: Request selection precedence and profile CRUD route wiring entrypoints.
  - Path: pinocchio/pkg/webchat/http/profile_api.go
    Note: Reusable CRUD handler behavior and error mapping.
  - Path: geppetto/pkg/doc/topics/01-profiles.md
    Note: User-facing profile model and migration/deprecation guidance.
ExternalSources: []
Summary: Final Phase 0 guardrails for implementation order, regression risks, flag deprecation policy, compatibility matrix, and rollout posture for registry-first profiles.
LastUpdated: 2026-02-23T22:02:00-05:00
WhatFor: Close GP01-000 through GP01-004 with an explicit plan that reflects the final implementation state.
WhenToUse: Use when validating rollout safety, writing release comms, or checking behavior parity expectations between legacy profile maps and registry-first resolution.
---

## Scope

This document closes and consolidates the initial guardrail tasks:

- `GP01-000` milestone order,
- `GP01-001` risk checklist,
- `GP01-002` deprecation policy for `ai-engine` / `ai-api-type`,
- `GP01-003` compatibility matrix,
- `GP01-004` rollout toggles/fallback strategy.

It reflects the final state after implementation, not an early draft state.

## 1. Milestone Order (GP01-000)

The implementation followed this order and this remains the recommended sequence for future registry-heavy rollouts:

1. Geppetto core domain and stores (`pkg/profiles`).
2. Geppetto resolution semantics + sections integration parity tests.
3. Pinocchio request resolver/runtime composer integration.
4. Pinocchio reusable profile CRUD HTTP routes.
5. Go-Go-OS runtime transport + UI state + selector integration.
6. End-to-end and regression matrix verification.
7. Docs and migration tooling.

Why this order worked:

- profile semantics had to stabilize before exposing HTTP mutation APIs,
- resolver/composer correctness had to stabilize before client UX work,
- CRUD route reuse had to exist before go-go-os server integration,
- migration/docs were safest after behavior and tests were fixed.

## 2. Compatibility Risk Checklist (GP01-001)

Use this as a pre-merge or pre-release checklist for profile-related changes.

| Risk Area | Failure Mode | Guardrail | Evidence |
|---|---|---|---|
| Precedence drift | Different effective runtime from legacy expectations | Keep golden parity tests and sections precedence integration tests green | `geppetto/pkg/profiles/service_test.go`, `geppetto/pkg/sections/profile_registry_source_test.go` |
| Resolver ambiguity | Wrong profile chosen from body/query/runtime/cookie/default | Keep explicit precedence tests for resolver inputs | `pinocchio/cmd/web-chat/profile_policy_test.go` |
| Runtime staleness | Profile mutation does not rebuild conversation runtime | Keep profile-version-sensitive fingerprint tests | `pinocchio/cmd/web-chat/runtime_composer_test.go`, `pinocchio/pkg/webchat/conversation_service_test.go` |
| Unsafe mutations | Read-only profile modified or deny-list ignored | Enforce policy in service + API mapping tests | `geppetto/pkg/profiles/service.go`, `pinocchio/pkg/webchat/http/profile_api_test.go` |
| Lost update | Concurrent edits overwrite each other | Require `expected_version` for safe updates and map conflicts to `409` | `geppetto/pkg/profiles/sqlite_store_test.go` |
| Store corruption | SQLite file copied unsafely while writes active | Snapshot backup procedure + WAL checkpoint in ops runbook | `playbook/01-db-profile-store-ops-notes-and-gp-01-release-notes.md` |

## 3. Deprecation Policy for `ai-engine` / `ai-api-type` (GP01-002)

### Policy statement

`--ai-engine` and `--ai-api-type` remain supported as direct override inputs, but they are no longer the primary operator workflow. Profile selection is primary, direct flags are secondary escape hatches.

### Practical rules

- New examples and docs should start with profile selection.
- Existing flags remain functional to avoid abrupt breakage for scripts.
- Profile definitions should carry stable defaults so scripts do not need to set engine/provider per invocation.
- Deprecation messaging should avoid hard removal dates until all first-party apps complete registry migration and downstream breakage risk is low.

### Contract for app maintainers

- Treat direct provider/engine flags as emergency/manual override tools.
- Avoid building new product UX around these low-level flags.
- Prefer profile CRUD + selection surfaces for user-facing runtime changes.

## 4. Compatibility Matrix (GP01-003)

| Capability | Legacy profile-map flow | Registry-first flow | Notes |
|---|---|---|---|
| Profile storage | Flat YAML map keyed by profile | Canonical registries/profiles model; memory/YAML/SQLite | Legacy YAML still loadable via codec/migration |
| Profile selection | profile + profile-file parsing in middleware | Explicit registry/profile selection plus resolution service | Selection precedence remains explicit and test-backed |
| Runtime composition | Flag overlay heavy, app-specific mappings | `ResolveEffectiveProfile` + typed runtime spec | Centralized policy and metadata |
| Override policy | Mostly app-specific checks | Shared `PolicySpec` + typed policy errors | Consistent `403` behavior from APIs |
| Mutation API | Limited or app-local | Reusable CRUD routes (`/api/chat/profiles...`) | Shared handlers for pinocchio and go-go-os |
| Persistence | YAML-only patterns in many flows | SQLite first-class + in-memory bootstrap fallback | DB ops guidance in GP01-903 runbook |
| Runtime rebuild trigger | Often runtime-key focused | Fingerprint includes profile version/effective inputs | Prevents stale runtimes after profile changes |

## 5. Rollout Toggles and Fallback Strategy (GP01-004)

### Final posture

There is no environment-based middleware toggle anymore. Registry-backed middleware integration is always on.

### What replaced toggle-based rollout

- strong automated regression coverage,
- explicit migration command and backup path for legacy files,
- store-level rollout choice via standard Glazed config (`in-memory` bootstrap vs SQLite-backed persistence),
- release-level rollback (binary/version rollback) instead of runtime env-flag switching.

### Operational fallback model

1. Keep backups of profile storage (YAML or SQLite) before rollout.
2. If a release regresses profile behavior, rollback deployment to previous binary.
3. Restore profile storage snapshot only if data shape drift occurred.
4. Re-run regression matrix and e2e profile scenarios before re-promoting.

This strategy keeps runtime behavior deterministic and avoids hidden branch logic in production paths.

## 6. Task Closure Mapping

- `GP01-000`: Section 1 (milestone order and rationale)
- `GP01-001`: Section 2 (risk checklist)
- `GP01-002`: Section 3 (deprecation policy)
- `GP01-003`: Section 4 (compatibility matrix)
- `GP01-004`: Section 5 (rollout and fallback strategy without env toggle)

## See Also

- `planning/01-profileregistry-architecture-and-migration-plan.md`
- `planning/02-implementation-postmortem-and-intern-guide.md`
- `playbook/01-db-profile-store-ops-notes-and-gp-01-release-notes.md`
