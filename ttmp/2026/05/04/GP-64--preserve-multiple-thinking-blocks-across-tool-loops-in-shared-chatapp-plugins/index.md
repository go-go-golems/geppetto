---
Title: Preserve multiple thinking blocks across tool loops in shared chatapp plugins
Ticket: GP-64
Status: active
Topics:
    - chatapp
    - plugins
    - reasoning
    - tool-calls
    - sessionstream
    - hydration
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: ../../../../../../2026-03-16--gec-rag/web/src/components/timeline/TimelineEntityRow.tsx
      Note: Thinking block renderer for role=thinking ChatMessage entities
    - Path: ../../../../../../2026-03-16--gec-rag/web/src/store/timelineSlice.ts
      Note: Redux upsert-by-ID behavior explains live overwrite/movement
    - Path: ../../../../../../2026-03-16--gec-rag/web/src/ws/parsing.ts
      Note: Frontend live/hydration parsing; will work once backend IDs are segment-aware
    - Path: ../../../../../../pinocchio/pkg/chatapp/chat.go
      Note: Runtime event sink and parent assistant message ID source
    - Path: ../../../../../../pinocchio/pkg/chatapp/plugins/reasoning.go
      Note: Root-cause file; ReasoningPlugin folds all thinking phases into one ID
    - Path: ../../../../../../pinocchio/pkg/chatapp/plugins/toolcall.go
      Note: Contrasting plugin; tool calls already have per-call stable IDs
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-04T15:46:03.16369473-04:00
WhatFor: ""
WhenToUse: ""
---


# Preserve multiple thinking blocks across tool loops in shared chatapp plugins

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- chatapp
- plugins
- reasoning
- tool-calls
- sessionstream
- hydration

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
