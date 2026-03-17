---
Title: Review and cleanup scopedjs and scopedjs demo work since origin main
Ticket: GP-36
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - js-bindings
    - pinocchio
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/pinocchio/cmd/examples/scopeddb-tui-demo/renderers.go
      Note: Baseline renderer used for comparison
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/pinocchio/cmd/examples/scopedjs-tui-demo/renderers.go
      Note: Demo rendering plumbing reviewed for extraction
    - Path: pkg/inference/tools/scopedjs/description.go
      Note: Generated tool descriptions and lifecycle prose
    - Path: pkg/inference/tools/scopedjs/runtime.go
      Note: Runtime construction flow
ExternalSources: []
Summary: Review ticket for the scopedjs and scopedjs-tui-demo work landed since origin/main, focused on cleanup, architectural clarity, and removable duplication.
LastUpdated: 2026-03-16T21:47:59.353681659-04:00
WhatFor: Capture the review findings, evidence, and cleanup plan before more code starts depending on the first scopedjs implementation shape.
WhenToUse: Use when onboarding to scopedjs, planning a cleanup pass, or reviewing whether newly added abstractions should remain stable or be simplified.
---


# Review and cleanup scopedjs and scopedjs demo work since origin main

## Overview

This ticket captures a cleanup-oriented review of the `scopedjs` package in Geppetto and the `scopedjs-tui-demo` example in Pinocchio. The review scope is everything added on the `task/add-scoped-js` branch since `origin/main`, with special attention to duplicated code, fuzzy lifecycle semantics, unidiomatic or misleading abstractions, and compatibility or migration baggage that can still be removed before the feature becomes harder to change.

The central output is an intern-friendly review and implementation guide in [design-doc/01-scopedjs-and-demo-review-cleanup-analysis-design-and-implementation-guide.md](./design-doc/01-scopedjs-and-demo-review-cleanup-analysis-design-and-implementation-guide.md). That guide explains the current system, documents the main findings, and proposes a cleanup plan that preserves the useful parts of the work while trimming the confusing parts.

## Key Links

- Main review guide: [design-doc/01-scopedjs-and-demo-review-cleanup-analysis-design-and-implementation-guide.md](./design-doc/01-scopedjs-and-demo-review-cleanup-analysis-design-and-implementation-guide.md)
- Investigation diary: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)
- Related code: see the frontmatter `RelatedFiles` fields in the design doc and diary

## Status

Current status: **active**

Current state of work:

- ticket created
- review diff collected from both repositories
- main findings documented
- cleanup plan drafted
- validation completed with `docmgr doctor`
- bundle uploaded to reMarkable and verified in the remote folder

## Topics

- geppetto
- tools
- architecture
- js-bindings
- pinocchio

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Reserved for future design docs if the cleanup spawns follow-up designs
- design-doc/ - Main long-form review and implementation guide
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts

## Review Focus

The review specifically targets:

- `StateMode` lifecycle promises versus actual runtime behavior
- lazy registration description quality versus prebuilt registration quality
- `EvalOptions` override semantics, especially boolean merges
- duplication between the `scopeddb` and `scopedjs` Pinocchio demos
- duplicated fake runtime capability modules across examples
- whether any of the newly introduced abstractions should be simplified before more code depends on them
