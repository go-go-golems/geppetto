---
Title: 'JS API Improvements: handles, constants, context, streaming'
Ticket: GP-01-JS-IMPROVEMENTS
Status: active
Topics:
    - geppetto
    - javascript
    - goja
    - api-design
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/chat-events.go
    - Path: geppetto/pkg/events/context.go
    - Path: geppetto/pkg/events/sink.go
    - Path: geppetto/pkg/inference/middleware/middleware.go
    - Path: geppetto/pkg/inference/session/execution.go
    - Path: geppetto/pkg/inference/session/session.go
    - Path: geppetto/pkg/inference/toolloop/loop.go
    - Path: geppetto/pkg/inference/tools/config.go
    - Path: geppetto/pkg/js/modules/geppetto/api.go
      Note: Core async/session code analyzed in bug report
    - Path: geppetto/pkg/js/modules/geppetto/codec.go
    - Path: geppetto/pkg/js/modules/geppetto/module.go
    - Path: geppetto/pkg/turns/block_kind_gen.go
    - Path: geppetto/pkg/turns/keys_gen.go
    - Path: geppetto/ttmp/2026/02/20/GP-01-JS-IMPROVEMENTS--js-api-improvements-handles-constants-context-streaming/analysis/01-bug-report-js-async-inference-runtime-thread-safety-runasync-start.md
      Note: Long-form bug report on async JS runtime safety
    - Path: geppetto/ttmp/2026/02/20/GP-01-JS-IMPROVEMENTS--js-api-improvements-handles-constants-context-streaming/planning/01-intern-guide-reusable-async-runtime-safety-runner-go-go-goja-geppetto.md
      Note: Intern-focused architecture and integration guide for reusable async VM safety runner
ExternalSources: []
Summary: |
    Four improvement areas for the geppetto JS/goja integration: (5.1) make opaque handles truly hidden via DefineDataProperty, (5.2) export enums/constants and ship .d.ts type definitions, (5.3) forward context (session/inference/turn IDs, timing) to middleware, tool handlers, and tool hooks, (5.4) add RunHandle with event streaming, per-run cancellation, and per-run options.
LastUpdated: 2026-02-20T07:37:50.199347415-05:00
WhatFor: ""
WhenToUse: ""
---



# JS API Improvements: handles, constants, context, streaming

## Overview

Four improvement areas for the geppetto JS/goja integration layer that make it
harder to write correct, observable, and ergonomic JS scripts:

1. **5.1 Opaque handles leak** — `__geppetto_ref` is enumerable/writable/discoverable
2. **5.2 Stringly-typed configs** — no constants, no IDE guidance, runtime-only failures
3. **5.3 Missing context** — middleware and tool handlers lack session/inference/turn IDs
4. **5.4 No streaming/RunHandle** — `runAsync()` returns bare Promise with no events or cancellation

All four are analyzed in detail with code locations and modification strategies in the design document.

## Key Links

- [Design: Codebase Analysis](./design/01-js-api-improvements-codebase-analysis.md) — detailed analysis of all four areas
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- javascript
- goja
- api-design

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
