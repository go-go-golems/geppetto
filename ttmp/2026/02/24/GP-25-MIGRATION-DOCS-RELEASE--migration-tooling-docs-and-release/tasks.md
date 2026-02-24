# Tasks

## Scope Reset (No Legacy Migration Tooling)

- [ ] Record scope decision: no legacy conversion tooling work in this ticket.
- [ ] Remove legacy-command references from ticket docs and plan artifacts.
- [ ] Clarify hard-cutover assumption: canonical registry files only.
- [ ] Link GP-25 scope to GP-24 (runtime cutover) and GP-27 (write-time/schema APIs).

## Geppetto Help Pages

- [ ] Update `geppetto/pkg/doc/topics/01-profiles.md` with current registry-first model.
- [ ] Document typed-key extension conventions used for profile payloads.
- [ ] Add/refresh section for middleware config ownership and validation timing.
- [ ] Update `geppetto/pkg/doc/topics/09-middlewares.md` with profile-scoped middleware configuration model.
- [ ] Update `geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md` with operational examples.
- [ ] Add troubleshooting for hard validation errors (unknown middleware, schema failure).

## Pinocchio and App Docs

- [ ] Add/update Pinocchio help page that documents `/api/chat/profiles` CRUD behavior.
- [ ] Add endpoint contract section for current-profile selection and default-profile semantics.
- [ ] Add hard-cutover notes for removed aliases/env vars with replacement guidance.
- [ ] Add Go-Go-OS integration notes describing reuse of shared profile API handlers.
- [ ] Document profile switching semantics (per-turn runtime truth vs conversation defaults).

## Schema API Documentation

- [ ] Document middleware schema endpoint contract (`/api/chat/schemas/middlewares`).
- [ ] Document extension schema endpoint contract (`/api/chat/schemas/extensions`).
- [ ] Add response examples for frontend form-generation consumers.
- [ ] Document error model for create/update profile when schema validation fails.
- [ ] Cross-link schema docs to GP-27 implementation details.

## Release Notes and Cutover Communication

- [ ] Draft hard-cutover release notes for profile registry rollout.
- [ ] Add explicit breaking-changes section (no soft compatibility mode).
- [ ] Add operator action checklist (`old behavior -> new behavior -> required action`).
- [ ] Add compatibility/version floor across geppetto, pinocchio, and go-go-os.
- [ ] Add rollback and incident-response guidance.

## Verification

- [ ] Verify all documented API examples against live endpoints.
- [ ] Verify all documented CLI/help examples execute as written.
- [ ] Run manual smoke for profile CRUD + selection in Pinocchio and Go-Go-OS.
- [ ] Capture validation transcript and commands in changelog.

## Closeout

- [ ] Run `docmgr doctor --ticket GP-25-MIGRATION-DOCS-RELEASE`.
- [ ] Run `docmgr validate frontmatter` on updated docs as needed.
- [ ] Ensure final changelog includes docs updates and release-note links.
