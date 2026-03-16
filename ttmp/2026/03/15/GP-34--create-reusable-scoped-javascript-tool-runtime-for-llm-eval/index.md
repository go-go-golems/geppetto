---
Title: Create reusable scoped JavaScript tool runtime for LLM eval
Ticket: GP-34
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - backend
    - js-bindings
    - go-api
    - security
    - inference
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/tools/scopeddb/tool.go
      Note: Current reusable pattern for prebuilt and lazy tool registration
    - Path: geppetto/pkg/inference/tools/registry.go
      Note: Shared Geppetto tool registry contract the new package must target
    - Path: geppetto/pkg/js/runtime/runtime.go
      Note: Owned runtime bootstrap with require registry and runtime initializers
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Existing runtime-owned geppetto module and JS to Go bridge surface
    - Path: geppetto/pkg/js/modules/geppetto/api_tools_registry.go
      Note: Existing JS-side tool registry and host Go tool interop
    - Path: geppetto/pkg/js/runtimebridge/bridge.go
      Note: Owner-thread callback bridge relevant for eval execution
    - Path: geppetto/pkg/doc/topics/13-js-api-reference.md
      Note: Existing JS API documentation surface to extend or mirror
ExternalSources: []
Summary: Ticket workspace for designing a reusable Geppetto package that exposes a configured goja runtime as a single scoped JavaScript eval tool for LLM use.
LastUpdated: 2026-03-15T23:14:15.882329582-04:00
WhatFor: Track the analysis and design work needed to turn Geppetto's current JS runtime pieces into a reusable scoped eval tool analogous to the new scoped database tool package.
WhenToUse: Use when designing or implementing a reusable eval-style JavaScript tool that bundles goja modules, globals, bootstrap scripts, and documentation behind one LLM-facing tool.
---

# Create reusable scoped JavaScript tool runtime for LLM eval

## Overview

This ticket scopes a reusable Geppetto package for exposing a configured goja runtime as a single LLM-facing tool such as `eval_dbserver`. The package should mirror the shape of the recently extracted scoped DB tool pattern: applications provide app-owned wiring, while Geppetto owns the generic registration, runtime bootstrap, description generation, and tool execution contract.

The motivating use case is a bounded JavaScript environment that can be assembled from registered goja modules and globals, preloaded JS files, and runtime documentation. A host application should be able to register pieces such as an `fs` module, a `db` global, Obsidian controls, and a webserver module, then expose the whole environment through one eval tool that the model can use coherently.

## Key Links

- **Primary design guide**: `design-doc/01-scoped-javascript-eval-tools-architecture-design-and-implementation-guide.md`
- **Primary analysis**: `analysis/01-scoped-javascript-tool-runtime-analysis-and-proposal.md`
- **Investigation diary**: `reference/01-investigation-diary.md`
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- tools
- architecture
- backend
- js-bindings
- go-api
- security
- inference

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- analysis/ - Research and design synthesis
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
