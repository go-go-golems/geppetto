---
Title: Remove tool calling middleware; standardize on tool loop
Ticket: GP-03-REMOVE-TOOL-MIDDLEWARE
Status: complete
Topics:
    - geppetto
    - inference
    - tools
    - refactor
    - design
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/examples/generic-tool-calling/main.go
      Note: Example that uses tool loop
    - Path: geppetto/cmd/examples/middleware-inference/main.go
      Note: Example that uses tool loop + middleware
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: Session runner that uses tool loop when Registry set
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Tool calling loop (ToolRegistry/ToolExecutor-based)
    - Path: geppetto/pkg/doc/topics/07-tools.md
      Note: Tools docs (builder/loop usage)
    - Path: geppetto/pkg/doc/topics/09-middlewares.md
      Note: Middleware docs (tool loop integration)
ExternalSources: []
Summary: "Completed: removed ToolMiddleware/Toolbox, migrated examples/docs to ToolLoopEngineBuilder + RunToolCallingLoop, updated smoke tests (incl. Claude tool calling)."
LastUpdated: 2026-01-22T19:38:33-05:00
WhatFor: ""
WhenToUse: ""
---


# Remove tool calling middleware; standardize on tool loop

## Overview

This ticket removes the legacy tool calling middleware (ToolMiddleware/Toolbox) and standardizes tool calling on the tool loop runner via `session.ToolLoopEngineBuilder` and `toolhelpers.RunToolCallingLoop`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete**

## Topics

- geppetto
- inference
- tools
- refactor
- design

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
