---
Title: Extract shared ReasoningPlugin and ToolCallPlugin into pinocchio/pkg/chatapp/plugins
Ticket: GP-63
Status: active
Topics:
    - chatapp
    - plugins
    - refactoring
    - reuse
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: ../../../../../../2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go
      Note: CoinVault debug feature to be replaced
    - Path: ../../../../../../2026-03-16--gec-rag/web/src/pb/external/pinocchio/chat_pb.ts
      Note: Generated TS protobuf schemas for shared ToolCall payloads
    - Path: ../../../../../../2026-03-16--gec-rag/web/src/ws/parsing.ts
      Note: CoinVault frontend parser for shared reasoning/tool-call events
    - Path: ../../../../../../pinocchio/cmd/web-chat/reasoning_chat_feature.go
      Note: Current reasoning plugin to be moved
    - Path: ../../../../../../pinocchio/pkg/chatapp/chat.go
      Note: Engine
    - Path: ../../../../../../pinocchio/pkg/chatapp/features.go
      Note: ChatPlugin interface definition
    - Path: ../../../../../../pinocchio/pkg/chatapp/pb/proto/pinocchio/chatapp/v1/chat.pb.go
      Note: Proto types to be extended with tool call messages
    - Path: ../../../../../../pinocchio/pkg/chatapp/plugins/reasoning.go
      Note: Shared ReasoningPlugin implementation
    - Path: ../../../../../../pinocchio/pkg/chatapp/plugins/toolcall.go
      Note: Shared ToolCallPlugin implementation
    - Path: ../../../../../../pinocchio/pkg/chatapp/plugins/toolcall_test.go
      Note: Unit tests for ToolCallPlugin runtime and projection behavior
    - Path: ../../../../../../pinocchio/pkg/ui/forwarders/agent/forwarder.go
      Note: Old TUI forwarder with same logic
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-04T14:39:53.989622783-04:00
WhatFor: ""
WhenToUse: ""
---



# Extract shared ReasoningPlugin and ToolCallPlugin into pinocchio/pkg/chatapp/plugins

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
- refactoring
- reuse

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
