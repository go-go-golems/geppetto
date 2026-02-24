---
Title: Implementation Plan - Hard-Cutover Docs and Release
Ticket: GP-25-MIGRATION-DOCS-RELEASE
Status: active
Topics:
    - architecture
    - migration
    - backend
    - chat
    - pinocchio
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/01-profiles.md
      Note: Geppetto profile registry reference docs.
    - Path: geppetto/pkg/doc/topics/09-middlewares.md
      Note: Middleware conceptual reference to align with profile-scoped configuration.
    - Path: geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md
      Note: Registry operations runbook.
    - Path: pinocchio/pkg/webchat/http/profile_api.go
      Note: Authoritative profile CRUD endpoint behavior and error model.
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Runtime/profile wiring and startup defaults to document.
    - Path: pinocchio/cmd/pinocchio/doc/doc.go
      Note: Help-page wiring for docs publication validation.
ExternalSources: []
Summary: Detailed rollout plan for hard-cutover documentation, API/schema references, and release communication for profile-registry adoption.
LastUpdated: 2026-02-24T13:12:02-05:00
WhatFor: Ensure users and integrators can adopt the hard cutover with clear API/schema contracts and operational guidance.
WhenToUse: Use when preparing docs/release artifacts for the registry-first runtime model without legacy conversion paths.
---

# Implementation Plan - Hard-Cutover Docs and Release

## Executive Summary

Technical completion is not enough for this rollout because APIs and symbols changed, aliases were removed, middleware configuration moved into profile-driven behavior, and schema discovery is expanding.

This ticket makes the hard cutover operationally safe by shipping:

1. glazed help pages/playbooks aligned to the final registry-first model,
2. API/schema references for frontend and operator teams,
3. release checklist with explicit breaking-change communication,
4. validation procedures for pinocchio and go-go-os behavior.

## Problem Statement

After cutover, teams need accurate docs for the new steady state. The main risks are:

- stale guidance that still references deprecated compatibility surfaces,
- unclear API contracts for profile CRUD and schema discovery,
- inconsistent understanding of profile/runtime semantics across pinocchio and go-go-os,
- release notes that under-specify required operator action.

Documentation is also distributed across repositories and can drift unless deliberately synchronized around one migration narrative.

Scope decision for this ticket: no legacy conversion tooling work.

## User Segments

- CLI operators running Pinocchio directly on canonical registry files.
- Web-chat operators running Pinocchio and/or Go-Go-OS servers.
- Third-party package consumers affected by symbol renames and alias removals.
- Frontend consumers needing schema endpoints for profile/middleware forms.
- Internal maintainers needing a release gate checklist.

## Proposed Solution

### 1. Documentation Set (Glazed Help Page Style)

Publish/update:

- geppetto profile topic: conceptual model + extension/validation expectations,
- geppetto middleware topic: profile-scoped middleware configuration and resolver behavior,
- geppetto registry operations playbook: canonical operational procedures,
- pinocchio profile registry page: runtime usage and CRUD behavior,
- pinocchio cutover note: removed aliases/env vars and replacement behavior.

All docs should include:

- prerequisites,
- concrete commands,
- expected output snippets,
- troubleshooting section,
- cross-links to schema endpoints and error contracts.

### 2. API and Schema Contracts

Document and stabilize:

- `GET /api/chat/profiles` and profile CRUD behavior,
- profile selection/default semantics,
- middleware schema discovery endpoint contract (`/api/chat/schemas/middlewares`),
- extension schema discovery endpoint contract (`/api/chat/schemas/extensions`),
- validation error payload shape for create/update failures.

### 3. Breaking-Change Communication

Document explicitly:

- removed aliases,
- removed compatibility env vars,
- canonical endpoints/symbol names,
- minimum compatible versions across repos.

Release notes should include a cutover matrix:

```text
old behavior -> new behavior -> required action
```

### 4. Verification and Release Gate

Add a release checklist that blocks rollout unless:

- docs are validated and linked,
- API examples and snippets execute as written,
- manual smoke passes for both pinocchio and go-go-os,
- changelog entries and upgrade notes are complete.

## Design Decisions

1. Legacy conversion tooling is out of scope for GP-25.
2. Glazed help pages are the canonical user-facing docs format.
3. Hard-cutover language is explicit; no soft compatibility messaging.
4. Release is gated by docs/API validation and application-level smoke checks.

## Alternatives Considered

### A. Keep legacy migration tooling as part of this ticket

Rejected because the current direction is hard cutover and this inflates scope with non-goals.

### B. Publish only terse release notes without docs refresh

Rejected because operators and frontend teams need concrete contracts and examples.

### C. Keep compatibility aliases indefinitely

Rejected because it prolongs technical debt and muddies the canonical API model.

## Implementation Plan

### Phase A - Scope Reset and Baseline

1. Remove legacy-tooling references from GP-25 docs/tasks.
2. Anchor scope to hard-cutover deliverables only.
3. Cross-link GP-25 dependencies on GP-24 and GP-27.

### Phase B - Geppetto Docs

1. Update profile topic with registry-first model and extension conventions.
2. Update middleware topic with profile-scoped configuration model.
3. Refresh registry operations playbook with canonical commands.
4. Validate frontmatter and help-page discoverability.

### Phase C - Pinocchio Docs

1. Add/update profile registry topic in pinocchio docs.
2. Add cutover note for symbol/API renames and alias/env-var removals.
3. Include copy-ready snippets for common deployment patterns.

### Phase D - API/Schema Contract Publication

1. Document profile CRUD and selection semantics with examples.
2. Document schema endpoints and response contract examples.
3. Document validation error patterns expected by frontend clients.

### Phase E - Release Notes and Validation

1. Draft release notes with explicit breaking changes and action matrix.
2. Run both servers and execute CRUD/profile selection/schema endpoint smoke.
3. Record outputs and caveats in ticket changelog.

## Open Questions

1. Should schema endpoint examples include both minimal and fully-decorated UI metadata variants?
2. Do we publish a machine-readable upgrade advisory (JSON/YAML) alongside human docs?
3. Which release boundary will be called out as the hard-cutover floor for third-party consumers?

## References

- `geppetto/pkg/doc/topics/01-profiles.md`
- `geppetto/pkg/doc/topics/09-middlewares.md`
- `geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md`
- `pinocchio/pkg/webchat/http/profile_api.go`
