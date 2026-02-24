---
Title: Hard-Cutover Docs and Release
Ticket: GP-25-MIGRATION-DOCS-RELEASE
Status: active
Topics:
    - architecture
    - migration
    - backend
    - chat
    - pinocchio
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/01-profiles.md
      Note: Core profile-registry user documentation.
    - Path: geppetto/pkg/doc/topics/09-middlewares.md
      Note: Middleware model and configuration documentation to align with profile registry.
    - Path: geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md
      Note: Operational playbook for registry-backed deployments.
    - Path: pinocchio/pkg/webchat/http/profile_api.go
      Note: Profile CRUD and schema endpoint contracts to document for operators and frontend teams.
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Runtime bootstrap defaults and registry integration behavior to document.
    - Path: go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go
      Note: Application integration reference for profile CRUD/runtime behavior.
    - Path: pinocchio/cmd/pinocchio/doc/doc.go
      Note: Embedded help-page loader used for pinocchio glazed docs publishing.
ExternalSources: []
Summary: Documentation and release-readiness track for shipping profile-registry hard cutover without legacy conversion tooling.
LastUpdated: 2026-02-24T13:12:02-05:00
WhatFor: Capture operator playbooks, schema/API references, compatibility-break notices, and release checklist for hard-cutover rollout.
WhenToUse: Use when preparing hard-cutover docs, validating command/API examples, and publishing release notes for profile-registry changes.
---

# Hard-Cutover Docs and Release

## Overview

This ticket packages the technical work into an adoptable release:

- polished help pages and operation playbooks,
- API and schema reference for profile and middleware behavior,
- explicit compatibility-break communication and release checks.

It is the bridge between implementation and successful downstream adoption by internal teams and third-party users.

Legacy conversion tooling is out of scope here. This ticket assumes canonical registry-first configuration and documents hard errors for invalid/unknown middleware/profile payloads.

## Key Links

- Design: [Implementation Plan - Migration Tooling Docs and Release](./design-doc/01-implementation-plan-migration-tooling-docs-and-release.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- migration
- backend
- chat
- pinocchio

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
