---
Title: Fix OpenAI Responses helper context and stream error handling bugs
Ticket: PI-020-OPENAI-RESPONSES-BUGFIX
Status: complete
Topics:
    - backend
    - bugfix
    - openai
    - responses
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Bug 2 fix implementation
    - Path: pkg/steps/ai/openai_responses/engine_test.go
      Note: Bug 2 regression tests
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: Bug 1 fix implementation
    - Path: pkg/steps/ai/openai_responses/helpers_test.go
      Note: Bug 1 regression tests
ExternalSources: []
Summary: 'Fixes two regressions in OpenAI Responses handling: assistant context loss before reasoning and streaming failures returning success.'
LastUpdated: 2026-02-25T17:31:26.535301278-05:00
WhatFor: ""
WhenToUse: ""
---



# Fix OpenAI Responses helper context and stream error handling bugs

## Overview

This ticket fixes two independent regressions in `pkg/steps/ai/openai_responses`:

1. `buildInputItemsFromTurn` dropped all assistant context blocks before the latest reasoning block instead of dropping only the intended single block.
2. Streaming inference treated SSE `error` / `response.failed` as success by emitting a final event and returning `nil` error.

Both fixes were implemented test-first with explicit failing regression tests, then validated with focused package tests and full pre-commit checks.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- backend
- bugfix
- openai
- responses

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
