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

## Step 2: Add `gp.runner.resolveRuntime(...)`

This step started the implementation proper. The goal was to introduce the new `gp.runner` namespace without yet touching execution. I wanted the first code slice to prove the runtime-boundary shape before taking on session assembly.

### What I did
- Updated `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go` to export a new `gp.runner` namespace.
- Added `runnerResolvedRuntimeRef` in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_types.go`.
- Added `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runner.go`.
- Refactored `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_middlewares.go` so middleware refs can be decoded and re-materialized cleanly instead of having ad-hoc one-way conversions.
- Implemented `gp.runner.resolveRuntime(...)` with support for:
  - profile-driven runtime resolution,
  - direct `systemPrompt`,
  - direct middleware additions,
  - direct tool-name overrides,
  - direct runtime identity metadata.
- Added focused tests in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module_test.go`.

### Commands
- `go test ./pkg/js/modules/geppetto -count=1`
- `./.bin/golangci-lint run ./pkg/js/modules/geppetto`

### What went wrong
- I first wrote the middleware-array export using `mwArray.Set(int64(i), v)`. Goja object property setting here wants string keys, so that had to become `fmt.Sprintf("%d", i)`.
- I also had a nil-guard issue in the runtime builder path where I touched metadata before verifying it existed.

### Why this slice matters
- It proves that the new `gp.runner` namespace can sit above the low-level module without redesigning the underlying session/builder APIs.
- It establishes a JS-native runtime object that can be passed into later `prepare`, `run`, and `start` calls.

## Step 3: Add prepared runs and blocking execution

This step built the first real execution layer. The design goal was to make `gp.runner.prepare(...)` thin: it should reuse the existing `sessionRef` path instead of building a second execution subsystem inside the JS module.

### What I did
- Added `preparedRunRef` in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_types.go`.
- Extended `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runner.go` with:
  - `gp.runner.prepare(...)`
  - `gp.runner.run(...)`
  - internal runtime application into the existing `builderRef`
  - prepared-turn cloning and runtime metadata stamping
  - JS prepared-run objects exposing:
    - `session`
    - `turn`
    - `runtime`
    - `run()`
    - `start()`
- Added focused tests for:
  - profile-driven prepared runs,
  - blocking `runner.run(...)`,
  - runtime metadata stamping on prepared turns,
  - tool-loop execution flowing through the prepared-run path.

### Commands
- `gofmt -w pkg/js/modules/geppetto/api_types.go pkg/js/modules/geppetto/module.go pkg/js/modules/geppetto/api_runner.go pkg/js/modules/geppetto/module_test.go`
- `go test ./pkg/js/modules/geppetto -count=1`

### Failures and fixes
- First failure:
  - error: `cannot use func(turn *turns.Turn) (*turns.Turn, error) as middleware.HandlerFunc`
  - cause: I copied the wrong middleware signature into the new test. The JS module middleware path expects `func(ctx context.Context, turn *turns.Turn) (*turns.Turn, error)`.
  - fix: changed the test middleware to the real handler signature and reused the typed metadata helper key.
- Second failure:
  - panic from `buildPreparedTurn` because I called `.String()` on `obj.Get("prompt")` when the property was undefined.
  - fix: guard goja property access before calling `.Export()` or `.String()`.
- Third failure:
  - panic from `prepareRunnerOptions` because I called `.Export()` on an undefined `sessionId`.
  - fix: the same guard pattern as above.
- Fourth failure:
  - `runner.run` ignored a direct `systemPrompt`.
  - cause: `resolveRuntime` was recording `SystemPrompt` but not rebuilding the corresponding `systemPrompt` middleware ref.
  - fix: added a `setRunnerSystemPrompt(...)` helper that rewrites the middleware spec list consistently for both profile-derived and direct prompt values.

### Review guidance
- Start with `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runner.go`.
- Verify that `prepareRunnerOptions(...)` uses the existing `applyBuilderOptions(...)` and `sessionRef` execution flow instead of duplicating session logic.
- Confirm that `buildPreparedTurn(...)` mirrors the Go runner behavior:
  - clone the seed turn,
  - clear the historical id,
  - append prompt text if needed,
  - stamp runtime metadata before execution.

## Step 4: Add top-level `gp.runner.start(...)`

This step finished the core execution surface. The design decision here was to reuse the existing session start-handle contract instead of inventing a second async handle shape just for the runner namespace.

### What I did
- Updated `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go` to export `gp.runner.start(...)`.
- Extended `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runner.go` so:
  - top-level `runner.start(...)` prepares once and then delegates to the existing `sessionRef.start(...)`,
  - prepared-run `start()` now augments the returned handle with `session`, `turn`, and `runtime`,
  - top-level `runner.start(...)` uses the same augmentation helper.
- Added a focused streaming test in `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module_test.go`.

### Commands
- `gofmt -w pkg/js/modules/geppetto/module.go pkg/js/modules/geppetto/api_runner.go pkg/js/modules/geppetto/module_test.go`
- `go test ./pkg/js/modules/geppetto -count=1`
- `./.bin/golangci-lint run ./pkg/js/modules/geppetto`

### Why this shape
- The existing `session.start(...)` contract is already good:
  - `promise`
  - `cancel()`
  - `on(eventType, callback)`
- Reusing that contract means the opinionated API feels consistent with the advanced API instead of subtly different.
- Attaching `session`, `turn`, and `runtime` to the handle keeps the top-level runner path debuggable and inspectable without forcing callers back down to `prepare(...)` when they only wanted to start immediately.

### What worked
- The implementation was very small because the prepared-run slice already had the right abstraction.
- The new top-level `runner.start(...)` stayed almost purely compositional: prepare once, then delegate.

### What warrants review
- Check that `attachPreparedRunToHandle(...)` is not mutating any semantics of the existing start handle beyond adding non-conflicting fields.
- Check that the top-level runner handle remains safe to use with the current owner-thread model, since all handle augmentation still happens synchronously on the JS owner thread before the background run settles.

## Related

- [../design-doc/01-opinionated-javascript-api-design-and-implementation-guide.md](../design-doc/01-opinionated-javascript-api-design-and-implementation-guide.md)
