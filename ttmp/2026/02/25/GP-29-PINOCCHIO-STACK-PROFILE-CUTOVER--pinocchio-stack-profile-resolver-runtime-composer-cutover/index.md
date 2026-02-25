---
Title: Pinocchio stack-profile resolver/runtime composer cutover
Ticket: GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER
Status: complete
Topics:
    - pinocchio
    - profile-registry
    - stack-profiles
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/profile_policy.go
      Note: App-owned request resolver currently duplicates override policy handling
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Runtime composition and fingerprint behavior to be aligned with GP-28 resolver outputs
    - Path: pinocchio/cmd/web-chat/runtime_composer_test.go
      Note: Baseline test suite for runtime composition behaviors
    - Path: geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/index.md
      Note: Upstream core contract this ticket adopts
ExternalSources: []
Summary: Downstream Pinocchio adoption ticket for GP-28 stack profile resolver/runtime-composer contracts.
LastUpdated: 2026-02-25T17:31:13.586863652-05:00
WhatFor: Track Pinocchio migration to registry-backed stack profile resolution and lineage-aware runtime composition.
WhenToUse: Use when implementing or reviewing profile/runtime behavior in pinocchio web-chat after GP-28 core changes.
---


# Pinocchio stack-profile resolver/runtime composer cutover

## Overview

This ticket tracks Pinocchio-side adoption of GP-28 stack profile behavior. The goal is to remove duplicated profile override/runtime composition logic in `cmd/web-chat` and consume geppetto resolver outputs directly.

Primary outcomes:

1. request resolver uses registry-backed stack profile resolution,
2. runtime composer relies on resolved runtime + lineage-aware fingerprint metadata,
3. web-chat API/tests reflect policy-gated request overrides and multi-registry profile selection.

## Key Links

- **Upstream core ticket**: `GP-28-STACK-PROFILES`
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- pinocchio
- profile-registry
- stack-profiles
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
