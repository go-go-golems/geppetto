---
Title: Operator Checklist - Profile Registry Hard Cutover
Ticket: GP-25-MIGRATION-DOCS-RELEASE
Status: active
Topics:
    - migration
    - backend
    - chat
    - pinocchio
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md
      Note: SQLite-backed operations and troubleshooting procedures.
    - Path: /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/doc/topics/webchat-profile-registry.md
      Note: CRUD, selection semantics, and schema endpoint contracts.
ExternalSources: []
Summary: Deploy-time and post-deploy checklist for profile-registry hard cutover.
LastUpdated: 2026-02-24T22:59:00-05:00
WhatFor: Give on-call and release operators a concrete go/no-go process.
WhenToUse: Use during rollout windows and post-deploy validation.
---

# Operator Checklist - Profile Registry Hard Cutover

## Pre-Deploy

- [ ] Confirm env var `PINOCCHIO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE` is removed everywhere.
- [ ] Confirm target binaries include schema endpoints and write-time validation behavior.
- [ ] Snapshot profile registry DB (`profiles.db`) before deploy.
- [ ] Confirm service account write access for registry DB and backup directory.

## Deploy

- [ ] Deploy pinocchio and go-go-os with matching release set.
- [ ] Ensure profile API handlers are mounted (`/api/chat/profiles*`, `/api/chat/profile`).
- [ ] Ensure schema endpoints are mounted:
  - [ ] `/api/chat/schemas/middlewares`
  - [ ] `/api/chat/schemas/extensions`

## Immediate Validation

- [ ] `GET /api/chat/profiles` returns expected list.
- [ ] `GET /api/chat/schemas/middlewares` returns non-empty schema list.
- [ ] `GET /api/chat/schemas/extensions` returns extension schema list.
- [ ] `POST /api/chat/profiles` with unknown middleware returns `400`.
- [ ] `POST /api/chat/profiles` with schema-invalid middleware config returns `400`.
- [ ] Valid profile create/patch succeeds and is reflected in list/get responses.

## Runtime Behavior Validation

- [ ] Set current profile through `/api/chat/profile` and verify cookie route behavior.
- [ ] Send one chat request with explicit profile and verify runtime key selection.
- [ ] Switch profile and verify subsequent turn behavior matches selected profile.

## Rollback Procedure

- [ ] Roll back binaries to previous stable release set.
- [ ] Restore pre-deploy profile DB snapshot.
- [ ] Verify CRUD list endpoint and one profile-selected chat request after rollback.

## Escalation Packet

Provide all of:

- failing request/response pair,
- current middleware schema endpoint payload,
- profile document payload for affected slug,
- runtime composer and profile API logs.
