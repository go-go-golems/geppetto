---
Title: Hard-Cutover Release Notes - Profile Registry Rollout
Ticket: GP-25-MIGRATION-DOCS-RELEASE
Status: active
Topics:
    - architecture
    - migration
    - backend
    - chat
    - pinocchio
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/http/profile_api.go
      Note: Canonical profile CRUD + schema endpoint contracts and error semantics.
    - Path: /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/doc/topics/webchat-profile-registry.md
      Note: Operator and integrator-facing webchat profile API guidance.
    - Path: /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/topics/01-profiles.md
      Note: Registry-first profile model and extension conventions.
    - Path: /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/topics/09-middlewares.md
      Note: Middleware configuration model and validation semantics.
ExternalSources: []
Summary: Release-ready cutover notes for profile-registry-first runtime behavior and schema-driven profile editing APIs.
LastUpdated: 2026-02-24T22:59:00-05:00
WhatFor: Give operators and integrators a precise, no-compatibility rollout guide.
WhenToUse: Use for release announcements, upgrade guides, and integration updates.
---

# Hard-Cutover Release Notes - Profile Registry Rollout

## Breaking Changes

- Profile registry middleware integration is always on.
- `PINOCCHIO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE` has been removed.
- Compatibility aliases for renamed runtime/webchat symbols are removed.
- Profile CRUD endpoints are canonical for runtime profile editing.
- Middleware config validation is enforced at profile write time for API handlers wired with middleware definitions.

## New API Surface

- `GET /api/chat/schemas/middlewares`
  - returns middleware definitions and JSON schemas.
- `GET /api/chat/schemas/extensions`
  - returns extension keys and JSON schemas.

These endpoints support schema-driven frontend forms for profile editing.

## Behavioral Changes

- Unknown middleware names in profile create/update return HTTP `400`.
- Schema-invalid middleware config payloads return HTTP `400`.
- Profile selection (`/api/chat/profile`) controls current UI selection state; runtime truth remains request/turn-scoped.

## Operator Action Matrix

| Old behavior | New behavior | Required action |
|---|---|---|
| optional middleware-registry toggle via env var | middleware-registry integration always active | remove env var from deployment configs |
| profile writes may persist invalid middleware payloads (app-dependent) | write-time middleware validation rejects invalid payloads | update clients to read schema endpoints and submit valid payloads |
| no schema discovery endpoint for profile editors | middleware and extension schemas available via API | switch frontend profile editors to schema-driven forms |
| alias-heavy runtime/webchat symbol compatibility | canonical symbol names only | update imports/types to canonical names per migration playbooks |

## Compatibility Floor

- geppetto: includes profile registry + extension key support + middleware schema resolver.
- pinocchio: includes shared profile CRUD handlers with schema endpoints and write-time middleware validation.
- go-go-os inventory chat: includes shared profile CRUD handlers and schema endpoint wiring.

Treat mixed-version deployments across these components as unsupported for profile editing surfaces.

## Rollback Guidance

1. Roll back binaries to the previous known-good release set.
2. Restore profile registry SQLite snapshot if API writes were performed with newer validation semantics.
3. Verify `/api/chat/profiles` and one profile-selected chat request.
4. Re-run smoke checks before reopening traffic.

## Incident-Response Notes

Collect these artifacts first:

- failing request payload (redacted) and full HTTP status/body,
- output of `/api/chat/schemas/middlewares` and `/api/chat/schemas/extensions`,
- profile document from `GET /api/chat/profiles/{slug}`,
- server logs around profile API handler and runtime composer resolution.
