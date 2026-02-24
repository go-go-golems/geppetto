---
Title: Migration Tooling, Docs, and Release
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
    - Path: pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy.go
      Note: CLI migration verb for converting legacy profiles.yaml into registry YAML.
    - Path: pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy_test.go
      Note: Migration correctness and idempotency tests.
    - Path: geppetto/pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md
      Note: Geppetto migration playbook in glazed help page format.
    - Path: geppetto/pkg/doc/topics/01-profiles.md
      Note: Core profile-registry user documentation.
    - Path: pinocchio/cmd/pinocchio/main.go
      Note: CLI root command integration point for migration tooling and docs discoverability.
    - Path: pinocchio/cmd/pinocchio/doc/doc.go
      Note: Embedded help-page loader used for pinocchio glazed docs.
ExternalSources: []
Summary: Final migration, documentation, and release-readiness track for shipping profile registry cutover safely to users and integrators.
LastUpdated: 2026-02-24T13:12:02-05:00
WhatFor: Capture operator playbooks, CLI tooling, compatibility break notices, and release checklist for profile-registry rollout.
WhenToUse: Use when preparing migration guides, running conversion tooling, and publishing release notes for profile-registry changes.
---

# Migration Tooling, Docs, and Release

## Overview

This ticket packages the technical work into an adoptable release:

- migration tooling for legacy profile files,
- polished help pages and migration playbooks,
- explicit compatibility-break communication and release checks.

It is the bridge between implementation and successful downstream adoption by internal teams and third-party users.

## Key Links

- Design: [Implementation Plan - Migration Tooling Docs and Release](./design-doc/01-implementation-plan-migration-tooling-docs-and-release.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- documentation
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
