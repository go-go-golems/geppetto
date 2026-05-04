---
Title: Thinking content not accumulated in OpenAI completion API stream events
Ticket: GP-62
Status: active
Topics:
    - openai
    - thinking
    - streaming
    - bug
    - sessionstream
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: ../../../../../../2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go
      Note: CoinVault debug feature that handles EventReasoningTextDelta with content=ev.Delta (BUG)
    - Path: ../../../../../../pinocchio/cmd/web-chat/reasoning_chat_feature.go
      Note: Pinocchio reasoning plugin that handles EventThinkingPartial correctly with ev.Completion
    - Path: pkg/events/chat-events.go
      Note: 'Event types: EventReasoningTextDelta (delta only) vs EventThinkingPartial (delta + completion)'
    - Path: pkg/js/modules/geppetto/api_events.go
      Note: JS event encoder that does NOT handle EventReasoningTextDelta
    - Path: pkg/steps/ai/openai/chat_stream.go
      Note: SSE stream decoder and normalizeChatStreamEvent that extracts reasoning deltas
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: OpenAI completion engine that emits EventReasoningTextDelta and EventThinkingPartial during streaming
    - Path: pkg/steps/ai/streamhelpers/reasoning_markdown.go
      Note: NormalizeReasoningDelta used to accumulate thinking content
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-04T11:44:50.500138191-04:00
WhatFor: ""
WhenToUse: ""
---


# Thinking content not accumulated in OpenAI completion API stream events

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- openai
- thinking
- streaming
- bug
- sessionstream

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
