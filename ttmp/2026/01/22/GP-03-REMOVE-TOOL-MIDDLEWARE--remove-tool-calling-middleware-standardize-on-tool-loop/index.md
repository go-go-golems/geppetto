---
Title: Remove tool calling middleware; standardize on tool loop
Ticket: GP-03-REMOVE-TOOL-MIDDLEWARE
Status: active
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
      Note: Example that uses ToolMiddleware
    - Path: geppetto/pkg/inference/middleware/tool_middleware.go
      Note: Legacy tool-calling middleware (Toolbox-based)
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: Session runner that uses tool loop when Registry set
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Tool calling loop (ToolRegistry/ToolExecutor-based)
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T11:37:36.226414528-05:00
WhatFor: ""
WhenToUse: ""
---


# Remove tool calling middleware; standardize on tool loop

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

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
