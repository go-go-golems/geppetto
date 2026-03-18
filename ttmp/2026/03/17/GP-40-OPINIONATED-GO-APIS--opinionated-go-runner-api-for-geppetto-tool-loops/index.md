---
Title: Opinionated Go Runner API for Geppetto Tool Loops
Ticket: GP-40-OPINIONATED-GO-APIS
Status: active
Topics:
    - geppetto
    - pinocchio
    - api-design
    - tools
    - middleware
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: 2026-03-14--cozodb-editor/backend/pkg/hints/engine.go
      Note: CozoDB inference setup (pain points)
    - Path: geppetto/pkg/inference/engine/engine.go
      Note: Core engine interface
    - Path: geppetto/pkg/inference/session/session.go
      Note: Session lifecycle management
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: Engine builder wiring pattern
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Core tool loop orchestration
    - Path: geppetto/pkg/inference/tools/registry.go
      Note: Tool registry interface
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: PinocchioCommand top-level abstraction
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-17T13:17:44.676997236-04:00
WhatFor: ""
WhenToUse: ""
---


# Opinionated Go Runner API for Geppetto Tool Loops

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- pinocchio
- api-design
- tools
- middleware

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
