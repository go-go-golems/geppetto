---
Title: Run-to-Session Elimination Research
Ticket: PI-017-RUN-SESSION-INVESTIGATION
Status: active
Topics:
    - backend
    - frontend
    - proto
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/chat-events.go
      Note: EventMetadata with LegacyRunID backwards compat
    - Path: pinocchio/pkg/inference/events/typed_planning.go
      Note: Planning events with RunID (planning domain)
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Conversation struct with RunID field
    - Path: pinocchio/pkg/webchat/router.go
      Note: Core router with startRunForPrompt and run_id in API responses
    - Path: pinocchio/pkg/webchat/turn_store.go
      Note: TurnStore interface with RunID
    - Path: pinocchio/pkg/webchat/turn_store_sqlite.go
      Note: SQLite schema with run_id column
    - Path: pinocchio/proto/sem/middleware/planning.proto
      Note: Proto with PlanningRun.run_id (planning domain)
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-07T08:33:47.927810681-05:00
WhatFor: ""
WhenToUse: ""
---


# Run-to-Session Elimination Research

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- backend
- frontend
- proto
- documentation

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
