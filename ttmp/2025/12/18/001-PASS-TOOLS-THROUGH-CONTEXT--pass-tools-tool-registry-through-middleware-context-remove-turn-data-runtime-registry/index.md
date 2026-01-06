---
Title: Pass tools/tool registry through middleware context (remove Turn.Data runtime registry)
Ticket: 001-PASS-TOOLS-THROUGH-CONTEXT
Status: complete
Topics:
    - geppetto
    - turns
    - tools
    - context
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/06-inference-engines.md
      Note: Updated tool helper section to reflect context-carried registry
    - Path: geppetto/pkg/doc/topics/07-tools.md
      Note: Updated to document tool registry via context (toolcontext) instead of Turn.Data
    - Path: geppetto/pkg/doc/topics/08-turns.md
      Note: Updated Turn.Data/tool registry guidance to context-carried registry
    - Path: geppetto/pkg/doc/topics/09-middlewares.md
      Note: Updated middleware docs to describe tool registry via context
    - Path: geppetto/pkg/doc/topics/12-turnsdatalint.md
      Note: Updated linter doc examples after removing DataKeyToolRegistry
    - Path: geppetto/pkg/doc/tutorials/01-streaming-inference-with-tools.md
      Note: Updated tutorial to attach tool registry to runCtx via toolcontext
    - Path: geppetto/pkg/inference/toolcontext/toolcontext.go
      Note: Context helpers for carrying ToolRegistry
ExternalSources: []
Summary: Move runtime ToolRegistry out of Turn.Data by passing it through middleware context; store only serializable tool definitions on the Turn.
LastUpdated: 2026-01-05T18:00:34.151654647-05:00
WhatFor: ""
WhenToUse: ""
---



# Pass tools/tool registry through middleware context (remove Turn.Data runtime registry)

## Overview

Stop storing `tools.ToolRegistry` in `turns.Turn.Data`. Instead, carry the runtime registry through `context.Context` and store only serializable tool definitions in `Turn.Data` so turns/blocks remain persistable and replayable.

## Key Links

- [Analysis: passing tool registry through context](./analysis/01-analysis-passing-tool-registry-through-context.md)
- [Design: context-carried tool registry + serializable Turn.Data](./design-doc/01-design-context-carried-tool-registry-serializable-turn-data.md)
- [Diary](./reference/01-diary.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- turns
- tools
- context
- architecture

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
