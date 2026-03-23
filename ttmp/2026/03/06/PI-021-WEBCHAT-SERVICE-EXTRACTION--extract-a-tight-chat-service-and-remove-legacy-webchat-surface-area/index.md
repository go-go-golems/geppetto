---
Title: extract a tight chat service and remove legacy webchat surface area
Ticket: PI-021-WEBCHAT-SERVICE-EXTRACTION
Status: active
Topics:
    - webchat
    - pinocchio
    - backend
    - cleanup
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-06T09:32:46.200638879-05:00
WhatFor: ""
WhenToUse: ""
---

# extract a tight chat service and remove legacy webchat surface area

## Overview

This ticket tracks the cleanup and migration of `pinocchio/pkg/webchat` toward a tighter shared backend service surface. The immediate objective is to simplify the current package without breaking the active `/chat`, `/ws`, and `/api/timeline` contract used by real integrations. The first implementation slice focuses on collapsing the redundant `ChatService` wrapper while preserving handler behavior.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary design doc**: `design-doc/01-investigation-and-migration-guide-for-tight-webchat-service-extraction.md`
- **Implementation diary**: `reference/01-diary.md`

## Status

Current status: **active**

## Topics

- webchat
- pinocchio
- backend
- cleanup

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
