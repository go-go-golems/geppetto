---
Title: Cleanup tool* packages
Ticket: GP-08-CLEANUP-TOOLS
Status: complete
Topics:
    - geppetto
    - tools
    - toolloop
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/toolblocks/toolblocks.go
      Note: Turn block helpers for tool_call/tool_use extraction and append
    - Path: geppetto/pkg/inference/toolcontext/toolcontext.go
      Note: Context plumbing for passing tool registries to provider engines
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Legacy tool calling loop + legacy config/hook types (candidate for deprecation/compat wrapper)
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Canonical tool loop orchestration (pause points, snapshots, tool execution)
    - Path: geppetto/pkg/inference/toolloop/engine_builder.go
      Note: Canonical session.EngineBuilder implementation for chat-style apps
    - Path: geppetto/pkg/inference/tools/registry.go
      Note: ToolRegistry API + in-memory implementation
    - Path: geppetto/pkg/inference/tools/base_executor.go
      Note: Tool execution orchestration + lifecycle hooks + event publishing
ExternalSources: []
Summary: Inventory and reorganize the geppetto tool* packages; identify deprecated/redundant parts and produce a migration plan for a cleaner, smaller, more coherent tool calling architecture surface.
LastUpdated: 2026-01-23T12:40:48.243132631-05:00
WhatFor: ""
WhenToUse: ""
---


# GP-08-CLEANUP-TOOLS

## Overview

Geppetto currently has five tool* packages under `geppetto/pkg/inference`:

- `toolblocks`
- `toolcontext`
- `toolhelpers`
- `toolloop`
- `tools`

This ticket is a cleanup/reorg starting point: document what each package does, what should be considered canonical vs legacy, what is redundant/confusing, and propose a migration plan toward a smaller and clearer API surface.

## Key docs

- Analysis report: `analysis/01-tool-packages-reorg-report.md`
- Tasks: `tasks.md`
