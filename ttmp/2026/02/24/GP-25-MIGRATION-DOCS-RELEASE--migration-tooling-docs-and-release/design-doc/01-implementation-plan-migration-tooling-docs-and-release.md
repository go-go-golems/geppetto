---
Title: Implementation Plan - Migration Tooling Docs and Release
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
    - Path: pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy.go
      Note: Core migration command implementation.
    - Path: pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy_test.go
      Note: Command behavior, format detection, and conversion tests.
    - Path: geppetto/pkg/doc/topics/01-profiles.md
      Note: Geppetto profile registry reference docs.
    - Path: geppetto/pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md
      Note: Practical migration runbook and command examples.
    - Path: pinocchio/cmd/pinocchio/main.go
      Note: CLI command tree where migration verb is mounted.
    - Path: pinocchio/cmd/pinocchio/doc/doc.go
      Note: Help-page wiring used for docs publication validation.
ExternalSources: []
Summary: Detailed rollout plan for migration tooling, help-page documentation, and release operations for the profile-registry cutover.
LastUpdated: 2026-02-24T13:12:02-05:00
WhatFor: Ensure users and integrators can migrate with minimal friction and clear operational guidance.
WhenToUse: Use when preparing documentation updates, running migration conversions, and finalizing release communication.
---

# Implementation Plan - Migration Tooling Docs and Release

## Executive Summary

Technical completion is not enough for this rollout because APIs and symbols changed, aliases were removed, and profile storage format evolved.

This ticket makes the cutover operationally safe by shipping:

1. migration CLI tooling with robust input-shape detection,
2. glazed help pages/playbooks for both geppetto and pinocchio users,
3. release checklist with explicit breaking-change communication,
4. validation procedures for both direct users and third-party integrators.

## Problem Statement

Users still have legacy `profiles.yaml` files and older API/symbol assumptions. Without structured migration support, they face:

- unclear conversion steps,
- uncertainty around what changed and why,
- high-risk manual edits of profile files,
- production incidents from stale automation scripts.

Documentation is also distributed across repositories and can drift unless deliberately synchronized around one migration narrative.

## User Segments

- CLI operators running Pinocchio directly.
- Web-chat operators running Pinocchio and/or Go-Go-OS servers.
- Third-party package consumers affected by symbol renames and alias removals.
- Internal maintainers who need a release gate checklist.

## Proposed Solution

### 1. Migration Command Hardening

Use `pinocchio profiles migrate-legacy` as the canonical conversion command.

Required command capabilities:

- detect legacy vs single-registry vs canonical multi-registry input,
- convert legacy maps to canonical registry format,
- support explicit registry slug override,
- support in-place and output-path modes,
- emit summary metrics (`registry_count`, `profile_count`, output path).

Expected operator flow:

```bash
pinocchio profiles migrate-legacy --input profiles.yaml --output profiles.registry.yaml
pinocchio profiles migrate-legacy --input profiles.registry.yaml --check
```

### 2. Documentation Set (Glazed Help Page Style)

Publish/update:

- geppetto profile topic: conceptual model + schema expectations,
- geppetto migration playbook: command-by-command conversion workflow,
- pinocchio profile registry page: runtime usage and CRUD behavior,
- pinocchio migration playbook: renamed symbols, removed aliases, practical before/after examples.

All docs should include:

- prerequisites,
- concrete commands,
- expected output snippets,
- rollback guidance,
- troubleshooting section.

### 3. Breaking-Change Communication

Document explicitly:

- removed aliases,
- removed compatibility env vars,
- new canonical endpoints/symbol names,
- minimum version requirements across repos.

Release notes should include a migration matrix:

```text
old behavior -> new behavior -> required user action
```

### 4. Verification and Release Gate

Add a release checklist that blocks rollout unless:

- migration command passes against fixture corpus,
- docs are validated and linked,
- manual smoke passes for both Pinocchio and Go-Go-OS,
- changelog entries and upgrade notes are complete.

## Design Decisions

1. One canonical migration command (`migrate-legacy`) instead of multiple ad-hoc scripts.
2. Glazed help pages are the canonical user-facing docs format.
3. Hard-cutover language is explicit; we do not hide breaking changes behind soft wording.
4. Release is gated by migration and docs verification, not code tests only.

## Alternatives Considered

### A. Rely on manual YAML editing instructions only

Rejected because it is error-prone and difficult to support at scale.

### B. Keep compatibility aliases indefinitely

Rejected because it prolongs technical debt and muddies API understanding.

### C. Publish migration docs only in one repo

Rejected because users consume both geppetto and pinocchio contexts.

## Implementation Plan

### Phase A - Command and Fixture Validation

1. Audit migration command flags and output contracts.
2. Add/expand fixture corpus for legacy/single/multi-registry input files.
3. Add idempotency tests and failure-mode tests.

### Phase B - Geppetto Docs

1. Update profile topic to include registry-first model and extension basics.
2. Update migration playbook with current command usage and caveats.
3. Validate frontmatter and help-page discoverability.

### Phase C - Pinocchio Docs

1. Add/update profile registry topic in pinocchio docs.
2. Add migration playbook for symbol/API renames and alias removals.
3. Include copy-ready command snippets for common deployment patterns.

### Phase D - Release Notes and Upgrade Matrix

1. Draft release notes with explicit breaking changes.
2. Add upgrade matrix old->new with required operator actions.
3. Add troubleshooting section for common migration failures.

### Phase E - Operational Validation

1. Run migration command on sample real-world profile files.
2. Run both servers with migrated profiles and execute CRUD/profile selection smoke.
3. Record outputs and known caveats in ticket changelog.

## Open Questions

1. Should migration command support automatic backup creation by default when `--in-place` is used?
2. Do we publish a machine-readable upgrade advisory (JSON/YAML) alongside human docs?
3. Which release version boundary will be called out as the hard-cutover floor for third-party consumers?

## References

- `pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy.go`
- `pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy_test.go`
- `geppetto/pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md`
- `geppetto/pkg/doc/topics/01-profiles.md`
