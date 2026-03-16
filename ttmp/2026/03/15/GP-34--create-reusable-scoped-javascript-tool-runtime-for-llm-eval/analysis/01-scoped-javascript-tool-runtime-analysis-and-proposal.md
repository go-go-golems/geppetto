---
Title: Scoped JavaScript tool runtime analysis and proposal
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
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/13-js-api-reference.md
      Note: |-
        Current public JS API docs surface
        Existing JS API docs surface to extend or mirror
    - Path: geppetto/pkg/inference/tools/registry.go
      Note: |-
        Registry contract the new eval tool helpers should target
        Target ToolRegistry contract for the new package
    - Path: geppetto/pkg/inference/tools/scopeddb/tool.go
      Note: |-
        Concrete precedent for the reusable registration shape to mirror
        Reusable prebuilt and lazy tool registration pattern to mirror for JS
    - Path: geppetto/pkg/js/modules/geppetto/api_tools_registry.go
      Note: |-
        Current JS-side registry and cross-runtime tool invocation contract
        Current JS tool registry surface and overlap with requested eval tool
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: |-
        Existing host module wiring and runtimeowner bridge usage
        Existing runtime-bound JS module and bridge usage
    - Path: geppetto/pkg/js/runtime/runtime.go
      Note: |-
        Existing owned runtime bootstrap and initializer hooks
        Owned runtime bootstrap with require and initializer hooks
    - Path: geppetto/pkg/js/runtimebridge/bridge.go
      Note: |-
        Thread-safe callback bridge for running Go and JS across the owned runtime
        Owner-thread bridge relevant for eval execution
ExternalSources: []
Summary: Initial codebase analysis for a reusable Geppetto package that exposes configured goja runtimes as one LLM-facing eval tool, modeled after the new scoped DB package.
LastUpdated: 2026-03-15T23:14:16.473078648-04:00
WhatFor: Give implementers a concrete map of the existing code seams and the likely package responsibilities for a reusable scoped JavaScript eval tool.
WhenToUse: Use when implementing GP-34 or when deciding how to package goja modules, globals, bootstrap scripts, and runtime documentation into one Geppetto tool.
---


# Scoped JavaScript tool runtime analysis and proposal

## Executive Summary

The new `scopeddb` package in Geppetto is the right immediate precedent. It extracted a repeated application pattern into a reusable package that:

- owns the common registration mechanics,
- keeps application-specific data loading local to the app,
- and exposes both prebuilt and lazy registration helpers.

The JavaScript side already has most of the lower-level pieces needed for an analogous extraction, but they are not yet assembled into one reusable LLM-facing tool package. Today Geppetto can:

- build an owned goja runtime with `require` support,
- register native modules,
- bridge JS callbacks onto the runtime owner thread,
- and create tool registries from JavaScript itself.

What is missing is the package that takes those pieces and turns them into one bounded tool surface such as `eval_dbserver`, with a stable input/output contract and a generated description of the available modules, globals, and helper functions.

## Current Codebase Findings

### 1. The scoped DB registration pattern is already the template

`geppetto/pkg/inference/tools/scopeddb/tool.go` is the clearest model for the new package shape. It exposes:

- `RegisterPrebuilt(...)` for already-constructed resources,
- `NewLazyRegistrar(...)` for context-derived resources,
- and a clear split between Geppetto-owned registration and app-owned materialization.

That same split fits the requested JS tool well. The JS equivalent would be:

- host-owned runtime spec and module/global wiring,
- Geppetto-owned eval tool registration,
- and optional lazy runtime construction from request/session context.

### 2. The JS runtime bootstrap is already centralized

`geppetto/pkg/js/runtime/runtime.go` already creates an owned runtime, enables `require`, optionally enables default go-go-goja modules, registers the `geppetto` module, and runs additional runtime initializers.

That means GP-34 does not need to invent runtime ownership. The reusable package can likely build on top of the existing `jsruntime.NewRuntime(...)` path, or extract a thinner host-neutral builder from it if the current API is too Geppetto-module-centric.

### 3. The Geppetto JS module proves the host-bridge side is real

`geppetto/pkg/js/modules/geppetto/module.go` and `geppetto/pkg/js/runtimebridge/bridge.go` show that Geppetto already has a working pattern for:

- owner-thread-safe goja execution,
- JS-to-Go callback bridging,
- and runtime-bound APIs that carry host state.

This matters because an `eval_xxx` tool will almost certainly need to:

- execute inside the owned runtime,
- keep module/global state coherent during one tool call,
- and safely translate Go errors and JS exceptions into tool results.

### 4. JS can already create tool registries, but that is not the same as a scoped eval tool

`geppetto/pkg/js/modules/geppetto/api_tools_registry.go` exposes `geppetto.tools.createRegistry()` and lets JS register handlers or import host Go tools. That is useful for scripting, but it is not yet the requested product surface.

The user request is for a single tool that lets the LLM operate inside one prepared runtime, not for the LLM to assemble or manage individual JS tools itself. The package therefore needs to sit one layer above the existing JS registry helper.

### 5. Documentation already has a home

`geppetto/pkg/doc/topics/13-js-api-reference.md` already documents the current JS surface. GP-34 should reuse that discipline and make runtime documentation first-class input to the tool description, not an afterthought.

## Recommended Shape

The likely package should live near the DB precedent:

```text
geppetto/pkg/inference/tools/scopedjs
```

or a similarly explicit name such as `scopedruntime`.

Its responsibilities should be:

1. Define a host-owned spec for one bounded JS runtime.
2. Build or resolve that runtime.
3. Register a single `ToolDefinition` such as `eval_dbserver`.
4. Render runtime documentation into the tool description.
5. Execute code inside the owned runtime with predictable timeout and error behavior.

## Likely Spec Split

The same separation used by `scopeddb` should apply here.

- App-owned:
  - which modules to register,
  - which globals to inject,
  - which JS bootstrap files to load,
  - what documentation to expose,
  - how to derive runtime scope from context.
- Geppetto-owned:
  - tool registration,
  - eval input/output schema,
  - runtime description rendering,
  - lifecycle and timeout handling,
  - prebuilt versus lazy registrar helpers.

## First-Pass Working Model

The requested tool appears to be conceptually:

```text
eval_<environment-name>
```

where the model receives one description containing:

- what globals exist,
- what modules can be required,
- which helper functions are available,
- which bootstrap files have already been loaded,
- and example snippets or starter tasks.

That is analogous to the scoped DB tool describing tables and starter queries. Here the "schema" is the runtime surface itself.

## Open Design Questions

- Should the tool accept raw JavaScript source only, or a richer request object with an operation name, code, and maybe structured arguments?
- Should modules be exposed only through `require(...)`, or should the package also support injecting documented globals such as `db` directly?
- Should bootstrap scripts run once at runtime construction time, or once per tool invocation?
- Should the package allow long-lived mutable runtime state, or should it rebuild/clone a clean runtime for each call?
- How should host documentation be authored: plain markdown fragments, structured descriptors, or both?

## Working Understanding Of The Task

The task is to design a reusable Geppetto package that does for JavaScript runtime tools what `scopeddb` now does for bounded SQLite query tools.

Concretely, the package should let an application declare a named runtime environment, register goja modules and globals into it, preload supporting JS files, attach human-readable documentation for the exposed capabilities, and then publish the whole environment as one LLM-facing tool such as `eval_dbserver`. That tool would allow the model to write and run bounded JavaScript against the prepared environment, for example standing up a small webserver that queries a scoped database and interacts with Obsidian controls, without forcing each application to reimplement the runtime bootstrap and registration mechanics from scratch.
