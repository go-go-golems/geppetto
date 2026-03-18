---
Title: Opinionated JavaScript APIs for Geppetto
Ticket: GP-46-OPINIONATED-JS-APIS
Status: active
Topics:
    - geppetto
    - javascript
    - js-bindings
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design and implementation ticket for adding a high-level gp.runner namespace above Geppetto's low-level JavaScript builder/session API. The first implementation slices now include runtime resolution, prepared runs, and blocking execution."
LastUpdated: 2026-03-18T11:03:00-04:00
WhatFor: "Track the implementation of the opinionated JavaScript runner layer that mirrors the new Go runner boundary."
WhenToUse: "Use when implementing or reviewing the gp.runner namespace, or when updating JS docs/examples to make the new runner the default path."
---

# Opinionated JavaScript APIs for Geppetto

## Overview

This ticket analyzes how to add a more opinionated JavaScript layer on top of Geppetto's current `require("geppetto")` module. The current JS API is functional and well-covered, but it is still centered on low-level composition primitives such as `createBuilder`, `createSession`, `runInference`, explicit engine construction, explicit tool registry assembly, and explicit event sink wiring.

That low-level surface is appropriate for advanced hosts and deterministic test harnesses. It is not yet the best app-facing default for small scripts, profile-driven assistants, or simple streaming tools. The purpose of this ticket is to define a smaller, clearer JS API layer that mirrors the new opinionated Go runner boundary:

- the caller owns engine creation explicitly,
- profile resolution contributes runtime metadata only,
- a dedicated runner surface assembles session, middleware, tool loop, and streaming behavior.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary design doc**: [design-doc/01-opinionated-javascript-api-design-and-implementation-guide.md](./design-doc/01-opinionated-javascript-api-design-and-implementation-guide.md)
- **Investigation diary**: [reference/01-manuel-investigation-diary.md](./reference/01-manuel-investigation-diary.md)

## Status

Current status: **active**

Implemented so far:

- `gp.runner.resolveRuntime(...)`
- `gp.runner.prepare(...)`
- prepared-run handles with `session`, `turn`, `runtime`, `run()`, and `start()`
- blocking `gp.runner.run(...)`

Still to do:

- top-level `gp.runner.start(...)`
- public type surface updates
- example scripts and doc rewrites that make `gp.runner` the default JS path
- ticket closeout and refreshed reMarkable upload

## Topics

- geppetto
- javascript
- js-bindings

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
