---
Title: Go-go-os stack-profile resolver/runtime composer cutover
Ticket: GP-30-GO-GO-OS-STACK-PROFILE-CUTOVER
Status: active
Topics:
    - go-go-os
    - profile-registry
    - stack-profiles
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go
      Note: Request resolution path to align with GP-28 stack profile inputs
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go
      Note: Runtime composition/fingerprint path to align with resolver outputs
    - Path: go-go-os/go-inventory-chat/cmd/go-go-os-launcher/main_integration_test.go
      Note: End-to-end integration expectations for runtime composition
    - Path: geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/index.md
      Note: Upstream core contract this ticket adopts
ExternalSources: []
Summary: Downstream go-go-os adoption ticket for GP-28 stack profile resolver/runtime-composer contracts.
LastUpdated: 2026-02-25T14:30:20.890182044-05:00
WhatFor: Track go-go-os migration to registry-backed stack profile resolution and lineage-aware runtime composition.
WhenToUse: Use when implementing or reviewing profile/runtime behavior in go-go-os pinoweb and launcher integration flows.
---

# Go-go-os stack-profile resolver/runtime composer cutover

## Overview

This ticket tracks go-go-os-side adoption of GP-28 stack profile behavior. The goal is to align `request_resolver` and `runtime_composer` with geppetto resolver outputs and remove duplicated composition/fingerprint logic.

Primary outcomes:

1. request resolution supports registry/runtimeKey/requestOverrides inputs in hard-cut mode,
2. runtime composition uses stack-aware resolved runtime + lineage-aware fingerprints,
3. integration tests validate policy-gated overrides and multi-registry selection behavior.

## Key Links

- **Upstream core ticket**: `GP-28-STACK-PROFILES`
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- go-go-os
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
