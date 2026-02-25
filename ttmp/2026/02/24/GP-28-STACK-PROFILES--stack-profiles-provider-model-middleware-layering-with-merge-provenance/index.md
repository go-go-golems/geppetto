---
Title: stack profiles provider-model-middleware layering with merge provenance
Ticket: GP-28-STACK-PROFILES
Status: active
Topics:
    - profile-registry
    - stack-profiles
    - merge-provenance
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/service.go
      Note: Main implementation target for stack-aware resolver changes
    - Path: geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/design-doc/01-stack-profiles-architecture-and-merge-provenance-for-provider-model-middleware-layering.md
      Note: Primary architecture proposal
    - Path: geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/reference/01-investigation-diary.md
      Note: Chronological investigation log
ExternalSources: []
Summary: Research ticket for implementing stacked profiles with deterministic merge rules and profile-level provenance in Geppetto.
LastUpdated: 2026-02-25T00:37:00-05:00
WhatFor: Guide implementation of provider/model/middleware profile composition and auditable runtime resolution.
WhenToUse: Use when designing or implementing stack profile behavior in registry, runtime resolver, and JS bindings.
---


# stack profiles provider-model-middleware layering with merge provenance

## Overview

This ticket researches how to implement stack profiles in Geppetto so operators can compose provider defaults, model variants, and middleware overlays without duplicating profile payloads.

Primary outputs:

1. architecture/design proposal,
2. command-level investigation diary,
3. delivery validation and reMarkable publication artifacts.

## Key Links

- Design document: `design-doc/01-stack-profiles-architecture-and-merge-provenance-for-provider-model-middleware-layering.md`
- Diary: `reference/01-investigation-diary.md`
- Task tracker: `tasks.md`
- Changelog: `changelog.md`
- Downstream follow-up: `GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER`
- Downstream follow-up: `GP-30-GO-GO-OS-STACK-PROFILE-CUTOVER`

## Status

Current status: **active**

## Topics

- profile-registry
- stack-profiles
- merge-provenance

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
