---
Title: Runtime metadata cleanup implementation plan
Ticket: GP-47-RUNTIME-METADATA-CLEANUP
Status: active
Topics:
    - geppetto
    - javascript
    - js-bindings
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/doc/topics/14-js-api-user-guide.md
      Note: Docs now reflect the execution path described in the plan
    - Path: pkg/js/modules/geppetto/api_profiles.go
      Note: Resolved-profile output now carries refs that the plan expected future assembly to reuse
    - Path: pkg/js/modules/geppetto/api_runtime_metadata.go
      Note: Implements the internal helper strategy proposed by the plan
    - Path: pkg/js/modules/geppetto/api_sessions.go
      Note: Session assembly now consumes runtime metadata centrally
ExternalSources: []
Summary: Narrow cleanup plan for making JS runtime metadata easier to consume correctly and easier to reuse from the future opinionated JS runner API.
LastUpdated: 2026-03-18T10:40:00-04:00
WhatFor: Clarify runtime metadata responsibilities and define a concrete cleanup path before or during the opinionated JS runner implementation.
WhenToUse: Use when implementing the runtime metadata cleanup itself or when deciding how it should feed into GP-46.
---


# Runtime metadata cleanup implementation plan

## Executive Summary

This ticket is about a narrow but important cleanup in the Geppetto JavaScript module: runtime metadata is already resolved, but it is not yet first-class execution input.

Today, `gp.profiles.resolve(...)` returns a useful object with:

- `effectiveRuntime.system_prompt`
- `effectiveRuntime.middlewares`
- `effectiveRuntime.tools`
- `runtimeKey`
- `runtimeFingerprint`

That information is good and worth keeping. The problem is what callers have to do next. They currently need to manually translate the metadata into:

- actual middleware instances,
- actual tool-registry filtering behavior,
- actual stamped runtime identity on runs and turns.

This ticket proposes cleaning up that boundary before or alongside the opinionated JS runner work. The cleanup should introduce internal helpers and sharper contracts so runtime metadata stops being "diagnostic output that callers must manually interpret" and becomes "structured execution input that higher layers can consume consistently."

## Problem Statement

The current JS runtime metadata story has three issues.

### 1. Resolution and consumption are separate, but the consumption side is underdeveloped

The JS module already resolves runtime metadata well in `pkg/js/modules/geppetto/api_profiles.go`. But the module does not provide a correspondingly clean path that consumes that metadata into execution assembly.

That leaves a gap:

```text
profiles.resolve(...)
  -> runtime metadata
  -> ??? manual caller translation ???
  -> builder/session execution
```

### 2. Different kinds of metadata are mixed together

The resolved payload currently contains both:

- metadata that should directly affect execution
- metadata that should mostly be stamped or inspected

These are different categories and should be treated differently.

### 3. The current gap makes the future opinionated JS API harder to build

If the opinionated `gp.runner` layer is implemented without cleaning this up first, it will likely end up embedding ad-hoc translation logic in the runner itself:

- special-case `system_prompt`
- special-case middleware-use materialization
- special-case tool filtering
- special-case runtime metadata stamping

That would work, but it would make the runner implementation do too much in one place.

## Current State

The relevant current behavior is spread across these files:

- `geppetto/pkg/js/modules/geppetto/api_profiles.go`
- `geppetto/pkg/js/modules/geppetto/api_sessions.go`
- `geppetto/pkg/js/modules/geppetto/api_builder_options.go`
- `geppetto/pkg/js/modules/geppetto/api_middlewares.go`
- `geppetto/pkg/js/modules/geppetto/api_tools_registry.go`
- `geppetto/pkg/doc/topics/13-js-api-reference.md`
- `geppetto/pkg/doc/topics/14-js-api-user-guide.md`

### What already exists

- profile resolution works
- runtime metadata is returned to JS
- Go middleware factories exist
- JS event sinks exist
- tool registries exist
- the builder/session layer can consume fully materialized middleware and tools

### What is missing

- no dedicated internal materialization helper for runtime metadata
- no canonical classification of runtime metadata categories
- no shared helper for stamping runtime identity metadata
- no shared helper for tool filtering from runtime tool-name metadata

## Proposed Solution

Introduce a dedicated runtime metadata cleanup layer inside the JS module before exposing any new public API.

The new internal layer should do three things.

### 1. Classify runtime metadata explicitly

The resolved runtime payload should be thought of as three categories.

#### Execution metadata

This should be consumable by execution assembly:

- `systemPrompt`
- `middlewares`
- `toolNames`

#### Identity metadata

This should be stamped onto turns/events/runs:

- `runtimeKey`
- `runtimeFingerprint`
- `profileVersion`

#### Inspection metadata

This should remain visible for debugging and UIs:

- `registrySlug`
- `profileSlug`
- raw `metadata`

This classification matters because it keeps the cleanup from becoming fuzzy.

### 2. Add internal helpers to materialize execution metadata

Internal helpers should convert resolved runtime metadata into execution-ready pieces:

- `systemPrompt` -> middleware
- middleware uses -> actual Go middleware instances
- tool names -> filtered execution registry

These helpers should not necessarily be public by themselves. They exist so both:

- the current low-level API,
- and the future `gp.runner`

can reuse the same logic.

### 3. Add internal helpers to stamp runtime identity metadata

The cleanup should define a canonical helper that stamps runtime identity metadata onto:

- prepared seed turns
- run context or event metadata where appropriate

That avoids repeated ad-hoc "set runtimeKey/runtimeFingerprint somewhere" logic in future APIs.

## Design Decisions

### Decision 1: keep `gp.profiles.resolve(...)`

The cleanup does not remove `resolve`.

`resolve` still has a clear role:

- inspection
- debugging
- tests
- profile browsers
- advanced hosts

What changes is its role in the default execution path. It should become the lower-level inspection primitive, not the normal app-facing "next step."

### Decision 2: keep the cleanup mostly internal

This cleanup should mostly produce internal helpers first, not a large new public surface.

Why:

- it keeps the change narrow,
- it reduces API churn,
- it gives GP-46 a reusable substrate.

### Decision 3: avoid putting all runtime translation logic directly into `gp.runner`

This is the main architectural reason for GP-47.

If the cleanup is done separately, the opinionated runner can stay focused on:

- input validation,
- execution assembly,
- sync/async start/run behavior.

It will not need to become a dumping ground for every piece of runtime metadata translation.

## Alternatives Considered

### Alternative A: do nothing and let `gp.runner` handle runtime metadata internally

This is viable, but less clean.

Why it is weaker:

- all runtime cleanup logic gets buried inside the future runner implementation,
- harder to test in isolation,
- harder to reuse from existing advanced codepaths.

### Alternative B: make `gp.profiles.resolve(...)` do more

For example:

- return already-materialized middleware instances,
- return already-filtered tools,
- attach execution helpers to the resolved object.

This is not recommended.

Why:

- it blurs resolution and execution again,
- it over-promotes a low-level inspection API into an execution API,
- it makes `profiles.resolve(...)` too smart.

### Alternative C: make the cleanup fully public before building `gp.runner`

For example:

- `gp.runtime.materialize(...)`
- `gp.runtime.filterTools(...)`

This could work, but it creates another public namespace to support and explain.

The cleaner path is:

- internal cleanup first,
- public opinionated runner second.

## Detailed Implementation Plan

### Phase 1: add internal runtime metadata classification helpers

Add internal structs or helper functions that separate:

- execution metadata
- identity metadata
- inspection metadata

Suggested file:

- `pkg/js/modules/geppetto/api_runtime_metadata.go`

### Migration note for GP-46

This cleanup is now the first real implementation slice of the future opinionated JavaScript runner.

Without GP-47, a future `gp.runner` implementation would still need to embed several low-level translations:

- build the system-prompt middleware from `effectiveRuntime.system_prompt`
- resolve profile middleware uses into concrete Go middleware instances
- clone and filter execution registries from runtime `tools`
- stamp runtime identity metadata onto prepared seed turns

With GP-47 landed, the future runner can stay focused on:

- validating app-facing inputs
- composing sessions or prepared runs
- exposing synchronous and asynchronous run helpers

In other words, GP-47 moves the JS module from:

```text
profiles.resolve(...)
  -> caller reinterprets runtime metadata manually
  -> session/builder execution
```

to:

```text
profiles.resolve(...)
  -> shared internal materialization helpers
  -> session/builder execution
  -> future gp.runner reuses the same substrate
```

Suggested internal shape:

```go
type resolvedRuntimeExecution struct {
    SystemPrompt string
    MiddlewareUses []goMiddlewareUse
    ToolNames []string
}

type resolvedRuntimeIdentity struct {
    RuntimeKey string
    RuntimeFingerprint string
    ProfileVersion uint64
}

type resolvedRuntimeView struct {
    RegistrySlug string
    ProfileSlug string
    Metadata map[string]any
}
```

### Phase 2: materialize middleware from runtime metadata

Implement a helper that:

1. accepts resolved runtime metadata,
2. builds a middleware slice,
3. resolves Go middleware factories by name,
4. injects a system prompt middleware if needed.

This should centralize current and future behavior.

### Phase 3: materialize filtered tools from runtime metadata

Implement a helper that:

1. accepts a source registry,
2. copies tools into a fresh in-memory registry,
3. filters by runtime `toolNames` if provided.

This avoids:

- in-place mutation,
- repeated caller logic,
- inconsistent policy.

### Phase 4: stamp runtime identity metadata consistently

Implement a helper that:

1. accepts a turn,
2. merges canonical runtime identity metadata,
3. preserves caller-supplied nested runtime fields where appropriate.

This should mirror the direction already used in the Go runner work.

### Phase 5: test the helpers directly

Add focused tests for:

- no runtime metadata
- system prompt only
- middleware uses only
- tool-name filtering only
- mixed runtime metadata
- unknown middleware names
- identity metadata stamping and merge behavior

### Phase 6: reframe docs

Update the JS docs to say:

- `gp.profiles.resolve(...)` is primarily for inspection and advanced resolution
- execution should normally happen through the future opinionated path
- runtime metadata categories are distinct and intentional

## Concrete Task Order

1. Create `api_runtime_metadata.go`
2. Add runtime metadata classification helpers
3. Add middleware materialization helper
4. Add registry clone/filter helper
5. Add runtime identity stamping helper
6. Add focused tests
7. Update JS docs
8. Reuse the helpers from GP-46 implementation

## How This Relates to GP-46

This is not fully orthogonal, and it is not a hard prerequisite either.

The right way to think about it is:

- GP-46 is the public API design (`gp.runner`)
- GP-47 is the internal cleanup that makes GP-46 cleaner to implement

So:

- GP-47 helps GP-46 materially
- but GP-46 could still be built without doing GP-47 first

More concretely:

### What GP-47 helps with

- cleaner internal boundaries
- smaller `gp.runner` implementation
- reusable runtime metadata helpers
- easier tests
- better doc framing

### What GP-47 does not block

- deciding on `gp.runner` naming
- deciding on `run` / `start` / `prepare`
- deciding to keep `profiles.resolve(...)`

So my recommendation is:

- treat GP-47 as the first implementation slice inside the GP-46 roadmap,
- but keep it as a separate ticket because it has a clear cleanup scope and can be reviewed independently.

## Open Questions

1. Should the runtime metadata helper be purely internal, or should a small subset become public later?
2. Should tool filtering happen only in the future runner, or should current builder-based paths also gain an internal helper wrapper?
3. Should runtime identity stamping be limited to turns, or should the event collector payloads also be normalized through the same helper?

## References

- `geppetto/pkg/js/modules/geppetto/api_profiles.go`
- `geppetto/pkg/js/modules/geppetto/api_sessions.go`
- `geppetto/pkg/js/modules/geppetto/api_builder_options.go`
- `geppetto/pkg/js/modules/geppetto/api_middlewares.go`
- `geppetto/pkg/js/modules/geppetto/api_tools_registry.go`
- `geppetto/pkg/doc/topics/13-js-api-reference.md`
- `geppetto/pkg/doc/topics/14-js-api-user-guide.md`
- `geppetto/ttmp/2026/03/18/GP-46-OPINIONATED-JS-APIS--opinionated-javascript-apis-for-geppetto/design-doc/01-opinionated-javascript-api-design-and-implementation-guide.md`
