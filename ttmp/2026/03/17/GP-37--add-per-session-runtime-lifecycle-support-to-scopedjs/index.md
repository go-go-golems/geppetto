---
Title: Add per-session runtime lifecycle support to scopedjs
Ticket: GP-37
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - js-bindings
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/session/context.go
      Note: |-
        Existing session identifier plumbing available from context
        Provides the default session identifier source for a future per-session registrar
    - Path: geppetto/pkg/inference/session/session.go
      Note: Existing long-lived session model that motivates per-session scoped runtimes
    - Path: geppetto/pkg/inference/tools/scopedjs/eval.go
      Note: |-
        Eval execution path that must remain safe under per-session runtime reuse
        RunEval assumes practical exclusive access to one runtime during evaluation
    - Path: geppetto/pkg/inference/tools/scopedjs/runtime.go
      Note: |-
        Runtime construction path that per-session pooling would reuse
        BuildRuntime returns the owned runtime handle that a future session pool would cache
    - Path: geppetto/pkg/inference/tools/scopedjs/tool.go
      Note: |-
        Current prebuilt and lazy registration strategies that define runtime reuse behavior
        Current registration strategies define runtime reuse behavior and are the main seam for a future per-session mode
ExternalSources:
    - https://github.com/go-go-golems/geppetto/issues/304
Summary: Design and planning ticket for adding true per-session runtime reuse semantics to scopedjs without reintroducing misleading lifecycle API promises.
LastUpdated: 2026-03-17T08:34:23.029561701-04:00
WhatFor: Capture the analysis, risks, and implementation plan for a future per-session runtime mode in scopedjs before expanding the public API again.
WhenToUse: Use when implementing, reviewing, or onboarding to the future scopedjs per-session runtime feature, especially around session identity, runtime pooling, and cleanup semantics.
---


# Add per-session runtime lifecycle support to scopedjs

## Overview

This ticket tracks a future enhancement to `pkg/inference/tools/scopedjs`: support a true per-session runtime lifecycle, where calls associated with the same Geppetto session reuse one prepared JavaScript runtime, while different sessions remain isolated from each other.

This is intentionally a future ticket rather than an immediate implementation slice. The current `scopedjs` package already supports two honest runtime behaviors:

- prebuilt registration, which reuses one runtime across all calls to that tool instance
- lazy registration, which builds a fresh runtime for each call

What is still missing is the middle ground: a session-scoped runtime pool keyed by a stable session identifier from `context.Context`. That feature touches lifecycle ownership, synchronization, cleanup, and public API shape, so it deserves a dedicated design and implementation plan before code lands.

## Key Links

- Main design and implementation guide: [design-doc/01-per-session-scopedjs-runtime-lifecycle-analysis-design-and-intern-implementation-guide.md](./design-doc/01-per-session-scopedjs-runtime-lifecycle-analysis-design-and-intern-implementation-guide.md)
- Investigation diary: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- GitHub issue body: [sources/01-github-issue-body.md](./sources/01-github-issue-body.md)
- GitHub issue: `go-go-golems/geppetto#304`
- Related Files: see the frontmatter `RelatedFiles` field for the core code paths

## Status

Current status: **active**

Current state:

- local ticket created
- existing `scopedjs` lifecycle behavior reviewed
- existing Geppetto session plumbing reviewed
- intern-friendly design and implementation guide drafted
- GitHub issue filed from the same design material

## Topics

- geppetto
- tools
- architecture
- js-bindings

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Reserved for extra follow-up design slices if the plan branches
- design-doc/ - Main long-form analysis and implementation guide
- reference/ - Chronological notes and quick reference material
- playbooks/ - Command sequences and validation procedures
- scripts/ - Reproduction or benchmarking helpers if later needed
- sources/ - Exact issue text and supporting source material
- various/ - Working notes and scratch research
- archive/ - Deprecated or reference-only artifacts

## Problem Focus

The future implementation needs to solve all of the following clearly:

- derive a stable session key for runtime reuse from existing Geppetto context and session mechanisms
- keep runtime state isolated between sessions
- serialize or otherwise safely manage concurrent access to a session-owned runtime
- clean up idle or poisoned runtimes without surprising the caller
- expose the feature through an honest API that matches real behavior
- preserve backward compatibility for existing prebuilt and lazy registrations
