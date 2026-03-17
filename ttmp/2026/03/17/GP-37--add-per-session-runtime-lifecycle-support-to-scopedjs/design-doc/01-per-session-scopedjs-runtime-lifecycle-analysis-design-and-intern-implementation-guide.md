---
Title: Per-session scopedjs runtime lifecycle analysis design and intern implementation guide
Ticket: GP-37
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - js-bindings
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/schema.go
      Note: Public API surface for scopedjs environment and eval options
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/tool.go
      Note: Current registrar entry points that would gain a per-session strategy
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/runtime.go
      Note: Runtime build path reused by a future session pool
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/eval.go
      Note: Eval execution path that must remain safe under runtime reuse
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/tool_test.go
      Note: Current tests for prebuilt and lazy lifecycle semantics
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/session/context.go
      Note: Existing session metadata context helpers
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/session/session.go
      Note: Existing stable session model in Geppetto
ExternalSources: []
Summary: Detailed design and implementation guide for adding true per-session runtime reuse to scopedjs.
LastUpdated: 2026-03-17T08:35:07.138577781-04:00
WhatFor: Explain the current scopedjs lifecycle system and provide a concrete future implementation plan for per-session runtime support.
WhenToUse: Use when onboarding to scopedjs internals, planning per-session runtime work, or reviewing lifecycle and cleanup tradeoffs before implementation begins.
---

# Per-session scopedjs runtime lifecycle analysis design and intern implementation guide

## Executive Summary

`scopedjs` currently supports two lifecycle strategies:

- one shared prebuilt runtime reused across all calls
- one fresh runtime built for each call

Those are both real behaviors implemented in code today. What does not yet exist is a runtime that is shared only within one Geppetto session. This document explains how the current system works, why a true per-session mode is useful, what new moving parts it needs, and how to implement it without repeating the old mistake of advertising lifecycle behavior the code does not actually enforce.

For a new intern, the most important takeaway is this: the hard part is not evaluating JavaScript. The hard part is deciding who owns the runtime, when it is reused, how it is cleaned up, and how that behavior is communicated honestly through the API.

## Problem Statement

### What exists today

The current `scopedjs` package lives under `geppetto/pkg/inference/tools/scopedjs`.

Its core responsibilities are:

- let the host application declare a JavaScript runtime environment
- register modules, globals, and bootstrap JavaScript
- expose that environment as one LLM-facing tool such as `eval_dbserver`
- run model-provided JavaScript against the prepared environment

The important current entry points are:

- `EnvironmentSpec[Scope, Meta]` in `schema.go`
- `BuildRuntime(...)` in `runtime.go`
- `RegisterPrebuilt(...)` in `tool.go`
- `NewLazyRegistrar(...)` in `tool.go`
- `RunEval(...)` in `eval.go`

### What is missing

There is no mode where:

- caller A in session `s-1` reuses prior state from earlier calls in `s-1`
- caller B in session `s-2` gets a different isolated runtime
- callers in unrelated sessions never observe each other's runtime state

That missing behavior is what this ticket calls `per_session`.

### Why the feature matters

Some JS-backed tools are more useful when they can accumulate state across a session:

- an in-memory cache of parsed data
- an incrementally configured local server/router object
- loaded helper modules or dynamically created objects
- expensive one-time initialization tied to the session context

Using one global prebuilt runtime for this would leak state across users or conversations. Using one fresh runtime per call throws away too much useful session-local state. Per-session reuse is the missing middle.

## Current System Walkthrough

### 1. Host application describes the environment

The host application builds an `EnvironmentSpec` in `schema.go`.

Conceptually:

```go
type EnvironmentSpec[Scope any, Meta any] struct {
    RuntimeLabel string
    Tool         ToolDefinitionSpec
    DefaultEval  EvalOptions
    Describe     func() (EnvironmentManifest, error)
    Configure    func(ctx context.Context, b *Builder, scope Scope) (Meta, error)
}
```

Important concepts:

- `Scope`: host-owned inputs used to build the runtime
- `Meta`: host-owned metadata returned after runtime construction
- `Describe`: static manifest for docs, especially useful before a runtime exists
- `Configure`: the actual builder callback that wires modules, globals, and bootstrap files

### 2. `Builder` accumulates runtime ingredients

`builder.go` contains the `Builder` methods used inside `Configure(...)`.

Main methods:

- `AddModule(...)`
- `AddNativeModule(...)`
- `AddGlobal(...)`
- `AddInitializer(...)`
- `AddBootstrapSource(...)`
- `AddBootstrapFile(...)`
- `AddHelper(...)`

The builder collects two things at once:

- real runtime setup instructions
- a manifest used to generate model-facing documentation

### 3. `BuildRuntime(...)` creates one owned runtime

`runtime.go` converts the builder state into a `go-go-goja` runtime:

```text
EnvironmentSpec.Configure(...)
    -> Builder accumulates modules/globals/bootstrap
    -> gojengine.NewBuilder()
    -> factory.NewRuntime(ctx)
    -> bootstrap JS files run
    -> BuildResult returned
```

Returned shape:

```go
type BuildResult[Meta any] struct {
    Runtime  *gojengine.Runtime
    Meta     Meta
    Manifest EnvironmentManifest
    Cleanup  func() error
}
```

That `BuildResult` is the right seam for a future session pool. A session entry can own one runtime plus cleanup metadata.

### 4. Registration currently determines lifecycle

This is the most important current fact.

`tool.go` exposes two registration strategies:

```text
RegisterPrebuilt(...)
    caller already built one runtime
    -> tool closure reuses that same runtime forever

NewLazyRegistrar(...)
    tool closure resolves scope per call
    -> BuildRuntime(...) per call
    -> cleanup immediately after the call
```

Diagram:

```text
                +----------------------+
                |  EnvironmentSpec     |
                +----------+-----------+
                           |
                           v
                   +---------------+
                   | BuildRuntime  |
                   +-------+-------+
                           |
          +----------------+----------------+
          |                                 |
          v                                 v
+--------------------+             +----------------------+
| RegisterPrebuilt   |             | NewLazyRegistrar    |
| holds one runtime  |             | builds every call   |
| shared by all call |             | fresh per call      |
+--------------------+             +----------------------+
```

This design is honest because the lifecycle behavior is encoded in the registration path itself. The future per-session feature should follow the same rule.

### 5. `RunEval(...)` evaluates JS against one runtime

`eval.go` handles one evaluation:

- install temporary `input`
- optionally replace `console`
- run wrapped async function
- wait for promise resolution if needed
- export the result
- restore console and clean temporary globals

Important point: `RunEval(...)` assumes it has exclusive practical access to the runtime during that call. That assumption matters for per-session support because the same runtime would be reused repeatedly. A session-owned runtime must therefore not be evaluated concurrently without coordination.

## What `per_session` Actually Means

### Intended semantics

For a given tool registration:

- same session ID => reuse the same runtime instance
- different session ID => different runtime instances
- missing session ID => follow a documented fallback or error contract

### Behavior table

| Strategy | Runtime reuse boundary | Isolation | Current status |
| --- | --- | --- | --- |
| prebuilt | all callers of one tool instance | none between callers | implemented |
| lazy | one call | full per-call isolation | implemented |
| per-session | one session ID | isolated between sessions | future |

### Example

Suppose tool code runs:

```js
globalThis.counter = (globalThis.counter || 0) + 1;
return { counter: globalThis.counter };
```

Expected behavior:

- prebuilt: values might be `1`, `2`, `3` across any callers
- lazy: every call returns `1`
- per-session:
  - session `s-1` sees `1`, then `2`
  - session `s-2` separately sees `1`, then `2`

## Proposed Solution

### High-level design

Add a new registration strategy that owns a runtime pool keyed by session ID. Each session ID maps to one built runtime plus bookkeeping.

High-level components:

- `SessionKeyResolver`: resolves the session key from `context.Context`
- `SessionRuntimePool`: stores runtime entries by session key
- `SessionRuntimeEntry`: owns one runtime and its metadata
- `RegisterPerSession(...)` or equivalent registrar helper
- eviction and shutdown support

### Recommended API direction

Do not add lifecycle back to `EvalOptions`.

Recommended shape:

```go
type SessionKeyResolver func(ctx context.Context) (string, error)

type PerSessionOptions struct {
    ResolveSessionKey SessionKeyResolver
    IdleTTL           time.Duration
    MaxEntries        int
    OnMissingSession  MissingSessionBehavior
}

type MissingSessionBehavior string

const (
    MissingSessionError        MissingSessionBehavior = "error"
    MissingSessionPerCall      MissingSessionBehavior = "per_call"
)

func RegisterPerSession[Scope any, Meta any](
    reg tools.ToolRegistry,
    spec EnvironmentSpec[Scope, Meta],
    resolveScope ScopeResolver[Scope],
    evalOpts EvalOptionOverrides,
    perSessionOpts PerSessionOptions,
) error
```

Why this direction is recommended:

- lifecycle remains a registration concern
- existing prebuilt and lazy APIs stay intact
- advanced callers can customize session key behavior without changing eval inputs

### Default session resolver

By default, the session key should come from:

```go
session.SessionIDFromContext(ctx)
```

Why:

- Geppetto already uses this for session-aware flows
- it avoids inventing a second session identity concept
- it makes the feature line up with existing `session.Session` behavior

### Runtime pool entry shape

Pseudocode:

```go
type sessionRuntimeEntry[Meta any] struct {
    key        string
    handle     *BuildResult[Meta]
    createdAt  time.Time
    lastUsedAt time.Time
    poisoned   error
    mu         sync.Mutex
}
```

Pool sketch:

```go
type sessionRuntimePool[Meta any] struct {
    mu      sync.Mutex
    entries map[string]*sessionRuntimeEntry[Meta]
    ttl     time.Duration
    max     int
}
```

### Call flow

Diagram:

```text
tool call
  |
  v
resolve session id from context
  |
  +--> missing?
  |      |
  |      +--> error or per-call fallback
  |
  v
look up pool entry by session id
  |
  +--> not found?
  |      |
  |      +--> resolve scope
  |      +--> BuildRuntime(...)
  |      +--> store entry
  |
  v
lock entry
  |
  v
RunEval(...)
  |
  +--> fatal runtime error?
  |      |
  |      +--> mark entry poisoned or evict immediately
  |
  v
unlock entry
  |
  v
update last-used timestamp
```

## Design Decisions

### Decision 1: lifecycle belongs at registration time, not eval time

Rationale:

- runtime ownership is broader than a single eval call
- `EvalOptions` should remain about call-level behavior such as timeout and console capture
- this avoids repeating the earlier misleading `StateMode` mistake

### Decision 2: default to existing session context plumbing

Rationale:

- `pkg/inference/session/context.go` already provides `SessionIDFromContext(ctx)`
- existing Geppetto flows already attach session metadata
- reuse of the established session mechanism lowers conceptual overhead for new contributors

### Decision 3: serialize access per session runtime

Rationale:

- a single `goja` runtime should not be treated as safely concurrent
- `RunEval(...)` mutates temporary globals and replaces `console`
- session-local reuse is valuable, but only if it is deterministic and race-free

Practical implication:

- one session runtime should have one lock around eval execution
- different sessions may still execute concurrently on different runtimes

### Decision 4: support explicit cleanup and idle eviction

Rationale:

- session-local runtimes can otherwise accumulate without bound
- long-lived runtimes may hold memory, files, or external handles
- cleanup semantics must be visible and testable from day one

## Alternatives Considered

### Alternative A: put `per_session` back into `EvalOptions`

Rejected because:

- it suggests lifecycle is a property of one call
- it hides runtime ownership in the wrong layer
- it recreates the same conceptual problem that GP-36 just cleaned up

### Alternative B: silently infer session behavior inside `NewLazyRegistrar(...)`

Rejected because:

- it would make lazy behavior less predictable
- it would overload one API with three different lifetime policies
- callers deserve to opt into runtime reuse explicitly

### Alternative C: make missing session ID always fall back to per-call

Not recommended as the only behavior because:

- it can hide configuration mistakes
- callers may think they are getting persistent session state when they are not

Better approach:

- expose explicit `OnMissingSession` behavior
- default to a conservative error unless there is a strong adoption reason otherwise

### Alternative D: one global mutex for the whole pool

Rejected because:

- it would serialize unrelated sessions unnecessarily
- per-session isolation should still allow cross-session parallelism

## Implementation Plan

### Phase 0: Finalize API and behavior contract

Goals:

- decide public API name and option shape
- decide missing-session behavior
- decide eviction defaults

Steps:

1. Add a design note in code comments near the new API explaining lifecycle semantics.
2. Keep `RegisterPrebuilt(...)` and `NewLazyRegistrar(...)` unchanged for backward compatibility.
3. Add a new explicit entry point for per-session registration.

### Phase 1: Add the runtime pool

Goals:

- own one runtime entry per session key
- support lookup, creation, reuse, and deletion

Pseudocode:

```go
func (p *sessionRuntimePool[Meta]) getOrCreate(
    ctx context.Context,
    key string,
    build func(context.Context) (*BuildResult[Meta], error),
) (*sessionRuntimeEntry[Meta], error) {
    p.mu.Lock()
    entry := p.entries[key]
    if entry == nil {
        entry = &sessionRuntimeEntry[Meta]{key: key}
        p.entries[key] = entry
    }
    p.mu.Unlock()

    entry.mu.Lock()
    defer entry.mu.Unlock()

    if entry.handle == nil {
        handle, err := build(ctx)
        if err != nil {
            return nil, err
        }
        entry.handle = handle
        entry.createdAt = time.Now()
    }
    entry.lastUsedAt = time.Now()
    return entry, nil
}
```

Note for an intern: the real implementation should avoid duplicate builds if two goroutines race to create the same key. That may require an entry placeholder plus careful lock ordering.

### Phase 2: Integrate session resolution

Goals:

- default to `session.SessionIDFromContext(ctx)`
- allow custom resolvers

Steps:

1. Add `SessionKeyResolver`.
2. If the resolver is nil, use the default Geppetto session resolver.
3. If the resolved key is empty:
   - either error immediately
   - or use explicit per-call fallback if configured

### Phase 3: Add per-session registrar

Goals:

- expose one LLM-facing tool
- reuse one runtime per session

Pseudocode:

```go
func RegisterPerSession(...) error {
    pool := newSessionRuntimePool(...)

    def, err := tools.NewToolFromFunc(
        spec.Tool.Name,
        BuildDescription(..., "Calls with the same session reuse one runtime; different sessions are isolated."),
        func(ctx context.Context, in EvalInput) (EvalOutput, error) {
            key, err := resolveSessionKey(ctx)
            if err != nil {
                return EvalOutput{Error: err.Error()}, nil
            }

            scope, err := resolveScope(ctx)
            if err != nil {
                return EvalOutput{Error: err.Error()}, nil
            }

            entry, err := pool.getOrCreate(ctx, key, func(buildCtx context.Context) (*BuildResult[Meta], error) {
                return BuildRuntime(buildCtx, spec, scope)
            })
            if err != nil {
                return EvalOutput{Error: err.Error()}, nil
            }

            entry.mu.Lock()
            defer entry.mu.Unlock()
            return RunEval(ctx, entry.handle.Runtime, in, evalOpts)
        },
    )
    ...
}
```

### Phase 4: Add eviction and shutdown

Goals:

- close idle runtimes
- prevent unbounded growth

Possible design:

- evict on access when checking timestamps
- optionally expose `Close()` on the registrar-managed pool or a manager object
- mark poisoned runtimes for immediate rebuild or eviction after fatal failures

### Phase 5: Test thoroughly

Minimum test matrix:

- same session retains JS state across calls
- different sessions do not share state
- missing session ID errors or falls back according to config
- concurrent calls to the same session do not race
- concurrent calls to different sessions can proceed independently
- eviction closes runtime cleanup hooks
- current prebuilt and lazy behavior is unchanged

### Phase 6: Document and teach the feature

Update:

- `pkg/doc/tutorials/07-build-scopedjs-eval-tools.md`
- any adoption playbook that compares runtime lifecycle strategies
- example programs if one is added later

## Intern Implementation Guide

### Mental model

Think of `scopedjs` as two layers:

1. environment construction
2. runtime ownership strategy

Environment construction already exists and works. Your job in this future ticket would mostly be to add a new ownership strategy.

### Safe order of work

1. Read `tool.go`, `runtime.go`, and `tool_test.go`.
2. Read `session/context.go`.
3. Write the failing tests first for same-session reuse and different-session isolation.
4. Add the smallest possible pool implementation.
5. Add cleanup and eviction only after the lifecycle tests pass.
6. Update docs last, once the API names are final.

### Common mistakes to avoid

- Do not put lifecycle configuration back into `EvalOptions`.
- Do not use one global mutex for all sessions.
- Do not silently fall back to per-call mode unless the API explicitly says so.
- Do not forget to call runtime cleanup when evicting entries.
- Do not assume `goja` runtime access is safe concurrently.

## Open Questions

- Should missing session ID default to error or per-call fallback?
- Should idle eviction happen only on access, or also in a background sweeper?
- Does the public API need an explicit close/shutdown hook for tests and host applications?
- Should fatal runtime errors poison an entry permanently or trigger one rebuild attempt on next access?

## References

- `geppetto/pkg/inference/tools/scopedjs/schema.go`
- `geppetto/pkg/inference/tools/scopedjs/tool.go`
- `geppetto/pkg/inference/tools/scopedjs/runtime.go`
- `geppetto/pkg/inference/tools/scopedjs/eval.go`
- `geppetto/pkg/inference/tools/scopedjs/tool_test.go`
- `geppetto/pkg/inference/session/context.go`
- `geppetto/pkg/inference/session/session.go`
- `geppetto/ttmp/2026/03/16/GP-36--review-and-cleanup-scopedjs-and-scopedjs-demo-work-since-origin-main/design-doc/01-scopedjs-and-demo-review-cleanup-analysis-design-and-implementation-guide.md`
