---
Title: persist serializable tool definitions on turn data
Ticket: GP-32-TURN-TOOL-DEFINITIONS
Status: active
Topics:
    - geppetto
    - schema
    - persistence
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Persist a durable per-turn snapshot of advertised tool definitions for inspection and debugging, while keeping provider advertisement and execution driven by the live runtime registry.
LastUpdated: 2026-03-10T21:30:26.844981276-04:00
WhatFor: Define and track the Geppetto implementation work required to persist serializable tool definitions on Turn.Data for inspection, without changing the runtime registry authority used for provider advertisement and execution.
WhenToUse: Use when implementing or reviewing per-turn tool-schema persistence, downstream tooling that needs durable tool definitions from persisted turns, or the runtime authority boundary between persisted snapshots and the live registry.
---

# persist serializable tool definitions on turn data

## Overview

Geppetto already stores `ToolConfig` on `Turn.Data` and already carries the live `ToolRegistry` through `context.Context`. What is still missing is a durable, serializable snapshot of the tool definitions themselves. Provider engines should continue to derive tool schemas from the runtime registry in context, but persisted turns should still retain an inspectable copy so old runs can answer the question "what tool schemas were configured around this turn?"

This ticket closes that gap by introducing a typed `Turn.Data` key for serializable tool definitions and wiring the tool loop to stamp that durable representation onto the turn. The runtime registry remains the source of truth for provider advertisement and execution; the persisted definitions are informational and inspectable only. The shipped snapshot format uses a JSON-safe `ToolDefinitionSnapshot` representation so `parameters` survive turn serde round trips.

## Key Links

- Design document: [design-doc/01-implementation-plan-for-persisting-serializable-tool-definitions-on-turn-data.md](./design-doc/01-implementation-plan-for-persisting-serializable-tool-definitions-on-turn-data.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- schema
- persistence

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
