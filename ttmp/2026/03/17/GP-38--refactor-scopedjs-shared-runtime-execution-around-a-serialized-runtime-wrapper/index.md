---
Title: Refactor scopedjs shared-runtime execution around a serialized runtime wrapper
Ticket: GP-38
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
    - Path: geppetto/pkg/doc/tutorials/07-build-scopedjs-eval-tools.md
      Note: |-
        Developer guide that should document the serialized shared-runtime wrapper
        Tutorial updated to explain BuildResult.Executor and the safe shared-runtime path
    - Path: geppetto/pkg/inference/tools/scopedjs/eval.go
      Note: Current multi-phase eval flow that is unsafe to interleave on one shared runtime
    - Path: geppetto/pkg/inference/tools/scopedjs/executor.go
      Note: New serialized runtime wrapper that owns reused-runtime eval locking
    - Path: geppetto/pkg/inference/tools/scopedjs/schema.go
      Note: Public BuildResult type that will expose the clearer runtime wrapper
    - Path: geppetto/pkg/inference/tools/scopedjs/tool.go
      Note: |-
        Prebuilt tool registration path that currently shares one runtime across calls
        RegisterPrebuilt now evaluates through the wrapper rather than the raw runtime
    - Path: geppetto/pkg/inference/tools/scopedjs/tool_test.go
      Note: |-
        Regression coverage for prebuilt and lazy registration behavior
        Concurrent prebuilt regression test proving whole-eval serialization on a shared runtime
ExternalSources: []
Summary: Cleanup ticket for introducing a clear serialized runtime wrapper around shared scopedjs runtime execution and validating it with concurrency regression tests.
LastUpdated: 2026-03-17T10:05:03.587010227-04:00
WhatFor: Replace the ad hoc prebuilt shared-runtime execution path with an explicit wrapper that owns serialization semantics and is reusable for future lifecycle modes.
WhenToUse: Use when reviewing or extending shared scopedjs runtime execution, especially around prebuilt runtimes, future per-session pooling, and concurrency safety.
---


# Refactor scopedjs shared-runtime execution around a serialized runtime wrapper

## Overview

This ticket tracks a focused cleanup in `pkg/inference/tools/scopedjs`: move shared-runtime evaluation behind a small explicit wrapper that serializes access to one runtime. The immediate bug is concurrent prebuilt evals interleaving across `prepare -> eval -> wait -> cleanup` phases, but the deeper goal is architectural clarity. Shared runtime ownership should live in one reusable type instead of being an incidental property of `RegisterPrebuilt(...)`.

The intended outcome is:

- `BuildRuntime(...)` still returns the raw runtime for compatibility
- `BuildResult` also exposes a clearer executor/wrapper for safe reused-runtime evals
- `RegisterPrebuilt(...)` uses that wrapper
- future lifecycle modes such as per-session runtime pools can reuse the same wrapper type

## Key Links

- Main design and implementation guide: [design-doc/01-serialized-shared-runtime-executor-cleanup-plan.md](./design-doc/01-serialized-shared-runtime-executor-cleanup-plan.md)
- Implementation diary: [reference/01-implementation-diary.md](./reference/01-implementation-diary.md)
- Tasks: [tasks.md](./tasks.md)

## Status

Current status: **active**

Current state:

- ticket created
- implementation shape chosen
- detailed tasks drafted
- code work pending

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

- design/ - Reserved for follow-up design material if this slice branches
- design-doc/ - Main implementation guide
- reference/ - Diary and quick reference material
- playbooks/ - Validation procedures if needed later
- scripts/ - Reproduction helpers if later needed
- various/ - Working notes
- archive/ - Deprecated artifacts
