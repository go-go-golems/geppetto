---
Title: Fix Claude tool-use duplicate text finalization
Ticket: GP-CLAUDE-DUPLICATE-TEXT
Status: active
Topics:
    - geppetto
    - claude
    - observability
    - streaming
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: pkg/steps/ai/claude/content-block-merger.go
      Note: Claude stream merger that emits duplicate final text for tool-use turns
    - Path: pkg/steps/ai/claude/content-block-merger_test.go
      Note: Regression tests for Claude text/tool-use/message-stop sequences
    - Path: pkg/steps/ai/claude/engine_claude.go
      Note: Streaming loop publishing merger events and provider observability
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-07T23:49:06.043940424-04:00
WhatFor: ""
WhenToUse: ""
---


# Fix Claude tool-use duplicate text finalization

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- claude
- observability
- streaming

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
