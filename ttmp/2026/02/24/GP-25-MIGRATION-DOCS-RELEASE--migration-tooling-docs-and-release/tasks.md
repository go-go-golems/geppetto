# Tasks

## Scope Reset (No Legacy Migration Tooling)

- [x] Record scope decision: no legacy conversion tooling work in this ticket.
- [x] Remove legacy-command references from ticket docs and plan artifacts.
- [x] Clarify hard-cutover assumption: canonical registry files only.
- [x] Link GP-25 scope to GP-24 (runtime cutover) and GP-27 (write-time/schema APIs).

## Geppetto Help Pages

- [x] Update `geppetto/pkg/doc/topics/01-profiles.md` with current registry-first model.
- [x] Document typed-key extension conventions used for profile payloads.
- [x] Add/refresh section for middleware config ownership and validation timing.
- [x] Update `geppetto/pkg/doc/topics/09-middlewares.md` with profile-scoped middleware configuration model.
- [x] Update `geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md` with operational examples.
- [x] Add troubleshooting for hard validation errors (unknown middleware, schema failure).

## Pinocchio and App Docs

- [x] Add/update Pinocchio help page that documents `/api/chat/profiles` CRUD behavior.
- [x] Add endpoint contract section for current-profile selection and default-profile semantics.
- [x] Add hard-cutover notes for removed aliases/env vars with replacement guidance.
- [x] Add Go-Go-OS integration notes describing reuse of shared profile API handlers.
- [x] Document profile switching semantics (per-turn runtime truth vs conversation defaults).

## Schema API Documentation

- [x] Document middleware schema endpoint contract (`/api/chat/schemas/middlewares`).
- [x] Document extension schema endpoint contract (`/api/chat/schemas/extensions`).
- [x] Add response examples for frontend form-generation consumers.
- [x] Document error model for create/update profile when schema validation fails.
- [x] Cross-link schema docs to GP-27 implementation details.

## Release Notes and Cutover Communication

- [x] Draft hard-cutover release notes for profile registry rollout.
- [x] Add explicit breaking-changes section (no soft compatibility mode).
- [x] Add operator action checklist (`old behavior -> new behavior -> required action`).
- [x] Add compatibility/version floor across geppetto, pinocchio, and go-go-os.
- [x] Add rollback and incident-response guidance.

## Verification

- [x] Verify all documented API examples against live endpoints.
- [x] Verify all documented CLI/help examples execute as written.
- [x] Run manual smoke for profile CRUD + selection in Pinocchio and Go-Go-OS.
- [x] Capture validation transcript and commands in changelog.

## Closeout

- [x] Run `docmgr doctor --ticket GP-25-MIGRATION-DOCS-RELEASE`.
- [x] Run `docmgr validate frontmatter` on updated docs as needed.
- [x] Ensure final changelog includes docs updates and release-note links.
