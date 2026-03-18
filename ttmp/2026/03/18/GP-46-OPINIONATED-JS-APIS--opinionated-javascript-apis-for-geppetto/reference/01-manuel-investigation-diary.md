---
Title: Manuel investigation diary
Ticket: GP-46-OPINIONATED-JS-APIS
Status: active
Topics:
    - geppetto
    - javascript
    - js-bindings
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/js/geppetto/20_events_collector_sink.js
      Note: Diary evidence for current streaming pattern
    - Path: pkg/doc/topics/14-js-api-user-guide.md
      Note: Diary evidence for current recommended workflow
    - Path: pkg/js/modules/geppetto/api_sessions.go
      Note: Diary evidence for builder/session execution path
    - Path: pkg/js/modules/geppetto/module.go
      Note: Diary evidence for export surface
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-18T09:49:10.919317766-04:00
WhatFor: ""
WhenToUse: ""
---


# Manuel investigation diary

## Goal

Capture the research, design reasoning, file-backed evidence, and review guidance for GP-46, which is about designing a more opinionated JavaScript API layer on top of Geppetto's current `require("geppetto")` module.

## Context

The repository just finished a major simplification pass on the Go side:

- `pkg/inference/runner` now provides an opinionated Go API,
- profile registries no longer mutate engine settings,
- profiles resolve runtime metadata instead of applying `StepSettings` patches.

The JS module has not received the equivalent simplification yet. It still exposes the lower-level builder/session model directly.

## Quick Reference

## Step 1: Research the current JS API surface

This step established the current architecture and the likely insertion point for a more opinionated JS layer. The main finding is that the JS module already has the right low-level primitives, but it lacks a small runner-style assembly surface that consumes explicit engine configuration plus resolved runtime metadata.

The best additive design direction is to introduce a new `gp.runner` namespace with `resolveRuntime`, `prepare`, `run`, and `start` methods, while keeping `createBuilder`, `createSession`, and `runInference` as the advanced layer underneath.

### Prompt Context

**User prompt (verbatim):** "Ok, now that we have done this opinionated go API, let's go back over the JS API and analyze how we could do something more opinionated around it. Create a new ticket and keep a diary as you work.

Then create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new Geppetto ticket focused on the JavaScript API, keep a chronological diary while investigating, write a detailed intern-oriented design/implementation guide, and publish the result to reMarkable.

**Inferred user intent:** Carry the same architectural simplification that just landed in the Go API over to the JS module, but do the work as a design-first ticket with full documentation before implementation starts.

### What I did
- Created ticket `GP-46-OPINIONATED-JS-APIS` with docmgr.
- Added the primary design doc and this diary doc.
- Mapped the main JS module exports in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go`.
- Read the low-level session/builder assembly code in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_sessions.go`.
- Read explicit engine construction in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go`.
- Read profile resolution and runtime stack binding in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go`.
- Read builder option parsing, tool registry APIs, event collector APIs, current TS declaration template, examples, and docs.

### Why
- The user explicitly asked for a detailed analysis before implementation.
- The JS layer has enough moving pieces that it is easy to propose the wrong abstraction without reading the current module code, examples, and docs together.
- The new design needs to fit the already-simplified Go architecture instead of reintroducing old profile/runtime confusion under a different name.

### What worked
- The current JS API boundaries were easy to map because the module files are cleanly split by namespace (`api_sessions.go`, `api_engines.go`, `api_profiles.go`, and so on).
- The docs already state the simplified contract that profiles resolve runtime metadata and engines are built explicitly, which gave a solid baseline for the new design.
- The examples showed the main usage patterns clearly:
  - deterministic session scripts,
  - explicit live-provider engine construction,
  - event-collector streaming,
  - profile inspection and runtime stack binding.

### What didn't work
- There was no existing opinionated JS entry point equivalent to the new Go runner package, so there was no direct API surface to extend by analogy.
- Profile middleware metadata is visible from `profiles.resolve(...)`, but there is no first-class JS path that takes those middleware uses and applies them to execution automatically. This means the current JS surface stops halfway through the profile-to-execution flow.

### What I learned
- The current JS API is intentionally explicit, not accidentally messy. The problem is not that it is wrong; the problem is that it is advanced-by-default.
- The best new layer is additive, not a replacement. `createBuilder` and `createSession` should remain as the expert API, while a new `gp.runner` namespace should become the default app-facing API.
- The current `GoMiddlewareFactories` hook is likely the right mechanism for materializing runtime middleware uses in the future JS runner layer.

### What was tricky to build
- The tricky part was identifying the correct conceptual boundary. A shallow read could suggest making `createSession` smarter, but that would just further overload an already-dense option object. The cleaner approach is to keep the current builder/session API low-level and add a separate runner namespace above it.
- Another tricky point was the profile/runtime split. The JS module already follows the post-GP-43 contract in the docs, but not yet in ergonomics. That makes it easy to accidentally propose a design that re-blurs engine creation and profile metadata resolution.

### What warrants a second pair of eyes
- Whether `gp.runner.resolveRuntime(...)` should accept both direct runtime metadata and profile input, or whether those should be separate calls.
- Whether `gp.runner.prepare(...)` is worth adding in the first implementation slice, or whether JS should start with only `run` and `start`.
- How much of the current low-level builder surface should be documented as “advanced” versus merely “available.”

### What should be done in the future
- Finish the design doc and task board.
- Relate the key code and doc files to the ticket.
- Run `docmgr doctor`.
- Upload the ticket bundle to reMarkable.

### Code review instructions
- Start with `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go` to see the exported namespaces.
- Then read `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_sessions.go` and `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_builder_options.go` to understand the current low-level assembly path.
- Compare that against `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/14-js-api-user-guide.md` and `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/20_events_collector_sink.js`.
- Validate this research pass by checking that the design doc's recommended `gp.runner` direction matches the evidence in those files.

### Technical details
- Ticket creation commands:
  - `docmgr ticket create-ticket --ticket GP-46-OPINIONATED-JS-APIS --title "Opinionated JavaScript APIs for Geppetto" --topics geppetto,javascript,goja`
  - `docmgr doc add --ticket GP-46-OPINIONATED-JS-APIS --doc-type design-doc --title "Opinionated JavaScript API design and implementation guide"`
  - `docmgr doc add --ticket GP-46-OPINIONATED-JS-APIS --doc-type reference --title "Manuel investigation diary"`
- Key evidence files:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_sessions.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_builder_options.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_tools_registry.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_events.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/14-js-api-user-guide.md`

## Usage Examples

Use this diary when continuing GP-46:

- read Step 1 before editing the JS module so you preserve the intended boundary,
- update the next step with exact commands and errors if implementation begins,
- treat the “What warrants a second pair of eyes” section as the current risk register.

## Related

- [../design-doc/01-opinionated-javascript-api-design-and-implementation-guide.md](../design-doc/01-opinionated-javascript-api-design-and-implementation-guide.md)
