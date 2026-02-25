---
Title: Ordered multi-source profile registries and single-registry YAML cutover
Ticket: GP-31-PROFILE-REGISTRIES-CHAIN
Status: active
Topics:
    - profile-registry
    - pinocchio
    - geppetto
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/codec_yaml.go
      Note: YAML runtime format and decode behavior
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/sections/sections.go
      Note: CLI profile settings middleware chain
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go
      Note: web-chat runtime request resolution behavior
ExternalSources: []
Summary: Planning ticket for ordered multi-source profile registry loading with single-registry YAML runtime format and profile-name-first resolution by chain order.
LastUpdated: 2026-02-25T16:24:30-05:00
WhatFor: Track design and implementation planning for GP-31 registry source chaining.
WhenToUse: Use when implementing or reviewing GP-31 registry source chain changes.
---

# Ordered multi-source profile registries and single-registry YAML cutover

## Overview

This ticket defines how to support `--profile-registries file1,file2,file3` across Pinocchio/Geppetto with deterministic ordered resolution. YAML sources are hard-cut to single-registry docs, SQLite sources may contain multiple registries, and runtime selection resolves profile names in source order when registry is unspecified.

## Key Links

- Design guide: `design-doc/01-implementation-guide-ordered-profile-registries-chain-and-single-registry-yaml-cutover.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`
- Diary: `reference/01-investigation-diary.md`

## Status

Current status: **active**

## Topics

- profile-registry
- pinocchio
- geppetto
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
