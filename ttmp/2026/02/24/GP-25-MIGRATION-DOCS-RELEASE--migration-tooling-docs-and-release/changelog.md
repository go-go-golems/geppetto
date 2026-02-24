# Changelog

## 2026-02-24

- Initial workspace created


## 2026-02-24

Populated ticket with migration-tooling/docs/release execution plan and detailed rollout tasks.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-25-MIGRATION-DOCS-RELEASE--migration-tooling-docs-and-release/design-doc/01-implementation-plan-migration-tooling-docs-and-release.md — Migration and release strategy
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-25-MIGRATION-DOCS-RELEASE--migration-tooling-docs-and-release/tasks.md — Granular migration/docs/release tasks

## 2026-02-24

- Re-scoped GP-25 to hard-cutover docs/release work only; removed legacy conversion-tooling deliverables from scope.
- Updated index, design plan, and task list to focus on:
  - registry-first canonical documentation,
  - profile CRUD and schema API contracts,
  - breaking-change communication and release gating.
- Explicitly documented that legacy migration command hardening is not part of this ticket.

## 2026-02-24

Completed hard-cutover docs/release pack: updated geppetto profile and middleware help pages, updated SQLite operations playbook with schema/write-validation checks, updated pinocchio webchat profile-registry guide with schema endpoints + per-turn runtime semantics + hard-cutover notes, and added ticket release artifacts (release notes + operator checklist). Verification executed with profile API integration tests in pinocchio/go-go-os and help-page discoverability check for webchat-profile-registry.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md — Operational checks for schema endpoints and validation failures
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/topics/01-profiles.md — Registry-first profile model updates
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/topics/09-middlewares.md — Profile-scoped middleware config
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-25-MIGRATION-DOCS-RELEASE--migration-tooling-docs-and-release/playbooks/01-operator-cutover-checklist.md — Operator go/no-go checklist and rollback packet
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-25-MIGRATION-DOCS-RELEASE--migration-tooling-docs-and-release/reference/01-hard-cutover-release-notes.md — Release notes and breaking-change matrix
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-25-MIGRATION-DOCS-RELEASE--migration-tooling-docs-and-release/tasks.md — Marked task list complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/doc/topics/webchat-profile-registry.md — CRUD/schema contracts


## 2026-02-24

Ticket closed after hard-cutover documentation, release notes, operator checklist, and verification updates.

