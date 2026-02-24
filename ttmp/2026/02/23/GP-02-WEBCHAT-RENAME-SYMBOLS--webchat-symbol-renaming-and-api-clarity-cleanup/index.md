---
Title: Webchat Symbol Renaming and API Clarity Cleanup
Ticket: GP-02-WEBCHAT-RENAME-SYMBOLS
Status: active
Topics:
    - architecture
    - backend
    - chat
    - pinocchio
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/profile_policy.go
      Note: Main resolver symbol rename surface
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Main composer symbol rename surface
    - Path: pinocchio/pkg/webchat/http/api.go
      Note: DTO naming cleanup boundary
    - Path: pinocchio/pkg/inference/runtime/composer.go
      Note: Runtime compose contract field rename boundary
ExternalSources: []
Summary: Ticket workspace tracking staged symbol/API renames in Pinocchio webchat runtime path to improve naming clarity without changing behavior.
LastUpdated: 2026-02-23T15:30:51.557623112-05:00
WhatFor: Execute and track a safe, staged naming migration for confusing webchat profile/runtime symbols and contracts.
WhenToUse: Use when implementing or reviewing GP-02 rename tasks and compatibility/deprecation changes.
---

# Webchat Symbol Renaming and API Clarity Cleanup

## Overview

This ticket tracks a behavior-preserving naming cleanup for webchat profile/runtime APIs and symbols.  
The goal is to reduce ambiguity in resolver/composer/request contracts so contributors can reason about profile selection and runtime composition quickly.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- architecture
- backend
- chat
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
