---
Title: test save-tool-config persistence through turns snapshotting
Ticket: GP-33-SAVE-TOOL-CONFIG-PERSISTENCE
Status: active
Topics:
    - geppetto
    - persistence
    - testing
    - tools
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/examples/geppetto-js-lab/main.go
      Note: Recommended host example binary for adding a persistence probe
    - Path: geppetto/examples/js/geppetto/05_go_tools_from_js.js
      Note: Recommended deterministic fixture with richer generated tool schema
    - Path: geppetto/pkg/inference/engine/types.go
      Note: Defines ToolDefinitionSnapshot and ToolDefinitions
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: Invokes the host TurnPersister after successful completion and wires snapshot hooks into the run context
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Stamps tool_config and tool_definitions onto Turn.Data before first inference and defines the persisted conversion helper
    - Path: geppetto/pkg/turns/serde/serde.go
      Note: Provides the YAML write path a host persister should call
ExternalSources: []
Summary: Research and implementation ticket for verifying that Geppetto persists tool loop configuration and serializable tool definitions into turn snapshots that can be written to disk by a host-side persister.
LastUpdated: 2026-03-11T08:00:00-04:00
WhatFor: Capture the analysis, recommended example harness, and validation runbook for proving end-to-end persistence of `tool_config` and `tool_definitions`.
WhenToUse: Use when implementing or reviewing a persistence probe for the recent save-tool-config work in Geppetto.
---


# test save-tool-config persistence through turns snapshotting

## Overview

This ticket documents how the recent tool-configuration and tool-definition persistence work flows through Geppetto today, and how to verify it using an existing example binary rather than a one-off harness.

Current conclusion:

- the persistence path in core Geppetto is already in place,
- `cmd/examples/geppetto-js-lab` is the best host harness,
- `examples/js/geppetto/05_go_tools_from_js.js` is the best primary fixture,
- the remaining implementation work is host-side persistence wiring plus final artifact upload.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary design doc**: `design-doc/01-intern-guide-to-testing-tool-config-and-tool-schema-persistence-through-turns-snapshotting.md`
- **Investigation diary**: `reference/01-diary.md`

## Status

Current status: **active**

## Topics

- geppetto
- persistence
- testing
- tools

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
