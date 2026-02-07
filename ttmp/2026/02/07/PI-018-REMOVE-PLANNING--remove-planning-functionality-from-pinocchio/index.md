---
Title: Remove Planning Functionality from Pinocchio
Ticket: PI-018-REMOVE-PLANNING
Status: active
Topics:
    - pinocchio
    - refactoring
    - cleanup
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/inference/events/typed_planning.go
      Note: Core planning event types to DELETE
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Remove 6 planning event handlers
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Remove planning aggregation logic
    - Path: pinocchio/proto/sem/middleware/planning.proto
      Note: Middleware proto to DELETE
    - Path: pinocchio/proto/sem/timeline/planning.proto
      Note: Timeline proto to DELETE
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-07T08:46:42.347152363-05:00
WhatFor: ""
WhenToUse: ""
---


# Remove Planning Functionality from Pinocchio

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- pinocchio
- refactoring
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
