---
Title: Clean up JavaScript runtime metadata resolution and consumption
Ticket: GP-47-RUNTIME-METADATA-CLEANUP
Status: active
Topics:
    - geppetto
    - javascript
    - js-bindings
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/js/geppetto/21_resolved_profile_session.js
      Note: Executable example of resolved profile runtime assembly
    - Path: pkg/doc/topics/13-js-api-reference.md
      Note: Reference docs updated to describe resolvedProfile as execution input
    - Path: pkg/doc/topics/14-js-api-user-guide.md
      Note: User guide updated to demote profiles.resolve to inspection/advanced use
    - Path: pkg/js/modules/geppetto/api_builder_options.go
      Note: Options-path entry point for resolvedProfile
    - Path: pkg/js/modules/geppetto/api_middlewares.go
      Note: Current middleware materialization path
    - Path: pkg/js/modules/geppetto/api_profiles.go
      Note: Current runtime metadata resolution entry point
    - Path: pkg/js/modules/geppetto/api_runtime_metadata.go
      Note: New internal helper layer for materializing and stamping resolved runtime metadata
    - Path: pkg/js/modules/geppetto/api_sessions.go
      Note: |-
        Current execution assembly layer that lacks first-class runtime metadata consumption
        Builder/session integration point for resolvedProfile materialization and turn stamping
    - Path: pkg/js/modules/geppetto/api_tools_registry.go
      Note: Current tool registry path and future filtering point
    - Path: pkg/js/modules/geppetto/module_test.go
      Note: Focused regression coverage for runtime metadata materialization
    - Path: ttmp/2026/03/18/GP-46-OPINIONATED-JS-APIS--opinionated-javascript-apis-for-geppetto/design-doc/01-opinionated-javascript-api-design-and-implementation-guide.md
      Note: Upstream design ticket this cleanup supports
ExternalSources: []
Summary: Implements the JS runtime-metadata cleanup that lets resolved profiles materialize middlewares, registry filtering, and stamped turn metadata before the future opinionated gp.runner API lands.
LastUpdated: 2026-03-18T10:56:00-04:00
WhatFor: Track the runtime-metadata cleanup slice that shrinks the gap between gp.profiles.resolve(...) and execution assembly.
WhenToUse: Use when reviewing the GP-47 implementation or when wiring the future opinionated JavaScript runner onto the cleaned runtime-metadata substrate.
---



# Clean up JavaScript runtime metadata resolution and consumption

## Overview

This ticket isolates the cleanup around JavaScript runtime metadata in `require("geppetto")`. Today, `gp.profiles.resolve(...)` returns useful runtime metadata such as `system_prompt`, `middlewares`, `tools`, `runtimeKey`, and `runtimeFingerprint`, but the module does not provide a first-class path that consumes that metadata into execution assembly.

The cleanup goal is to make that boundary explicit and smaller:

- keep runtime metadata resolution as an inspection primitive,
- define which runtime metadata should be automatically consumable by execution,
- remove ad-hoc/manual metadata application from normal callers,
- prepare the module for the future opinionated `gp.runner` layer.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Implementation plan**: [design-doc/01-runtime-metadata-cleanup-implementation-plan.md](./design-doc/01-runtime-metadata-cleanup-implementation-plan.md)

## Status

Current status: **active**

## Topics

- geppetto
- javascript
- js-bindings

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Diary

See [reference/01-manuel-investigation-diary.md](./reference/01-manuel-investigation-diary.md) for the step-by-step implementation diary.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
