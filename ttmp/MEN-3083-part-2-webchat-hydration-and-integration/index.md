---
Title: 'go-go-mento: Webchat/Web hydration and integration reference'
Ticket: MEN-3083-part-2
Status: active
Topics:
    - frontend
    - conversation
    - events
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: go-go-mento/go/pkg/webchat/conversation.go
      Note: WS broadcast; reader; topic chat:<conv_id>
    - Path: go-go-mento/go/pkg/webchat/forwarder.go
      Note: SEM mapping; team analysis typed frames; id stability
    - Path: go-go-mento/go/pkg/webchat/router.go
      Note: WS /rpc/v1/chat/ws; chat handlers; profile resolution
    - Path: go-go-mento/web/src/hooks/useChatStream.ts
      Note: WS connect; trigger hydrate; SEM→Redux mapping
    - Path: go-go-mento/web/src/pages/Chat/timeline/types.ts
      Note: Timeline entity types; optional version
    - Path: go-go-mento/web/src/store/timeline/timelineSlice.ts
      Note: version-gated upsert; hydrateTimelineThunk; mapping
ExternalSources: []
Summary: ""
LastUpdated: 2025-11-04T11:18:23.838560327-05:00
---



# go-go-mento: Webchat/Web hydration and integration reference

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- frontend
- conversation
- events

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
