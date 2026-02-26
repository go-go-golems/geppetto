---
Title: Remove legacy profile registry compatibility paths after hard cutover
Ticket: GP-32-PROFILE-REGISTRY-LEGACY-CLEANUP
Status: active
Topics:
    - profile-registry
    - geppetto
    - pinocchio
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Hard-cut cleanup ticket to remove remaining legacy profile-map and multi-registry-YAML compatibility paths and align all profile tooling to strict runtime single-registry YAML + registry-source stacks.
LastUpdated: 2026-02-25T18:40:51.411325163-05:00
WhatFor: Track and execute removal of deprecated profile compatibility code introduced during migration phases but now obsolete after hard-cutover decisions.
WhenToUse: Use when implementing or reviewing profile-system simplification work in geppetto/pinocchio.
---

# Remove legacy profile registry compatibility paths after hard cutover

## Overview

This ticket tracks a hard-cut cleanup pass for the profile subsystem.

Current runtime contracts already require:

- registry-source stacks via `profile-registries`,
- one-file-one-registry runtime YAML (`slug` + `profiles`),
- no runtime use of legacy profile-map files.

But several compatibility/migration paths still exist in code and tests. This ticket removes them to reduce complexity and prevent future drift.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- [Design: hard-cut cleanup inventory and removal plan](./design/01-hard-cut-cleanup-inventory-and-removal-plan.md)

## Status

Current status: **active**

## Topics

- profile-registry
- geppetto
- pinocchio
- migration

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
