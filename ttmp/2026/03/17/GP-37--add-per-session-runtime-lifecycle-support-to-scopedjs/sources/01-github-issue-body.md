---
Title: GitHub issue body
Ticket: GP-37
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - js-bindings
DocType: sources
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/tool.go
      Note: Existing registration split referenced directly in the issue text
ExternalSources:
    - https://github.com/go-go-golems/geppetto/issues/304
Summary: Exact GitHub issue body filed for the per-session scopedjs runtime ticket.
LastUpdated: 2026-03-17T08:35:07.251126833-04:00
WhatFor: Preserve the exact upstream issue text so local docs and the GitHub tracker stay synchronized.
WhenToUse: Use when reviewing what was filed upstream or when updating the local ticket to match later GitHub issue edits.
---

# GitHub issue body

Add true per-session runtime lifecycle support to `scopedjs`

## Summary

`scopedjs` currently has two honest lifecycle strategies:

- `RegisterPrebuilt(...)`: one runtime reused for all calls to the registered tool
- `NewLazyRegistrar(...)`: one fresh runtime built per call

What is still missing is a true middle mode where calls in the same Geppetto session reuse one runtime, while different sessions remain isolated. This issue tracks adding that as a real implementation, not just description text.

## Why this matters

Some scoped JS environments are more useful when they can accumulate session-local state:

- in-memory caches
- incrementally configured helper objects
- router or server setup that should survive within one conversation/session
- expensive one-time initialization tied to one session

Today callers have to choose between:

- global shared state across everybody
- no retained state at all

Per-session runtime reuse would provide the missing middle.

## Existing code paths

- `pkg/inference/tools/scopedjs/schema.go`
- `pkg/inference/tools/scopedjs/tool.go`
- `pkg/inference/tools/scopedjs/runtime.go`
- `pkg/inference/tools/scopedjs/eval.go`
- `pkg/inference/tools/scopedjs/tool_test.go`
- `pkg/inference/session/context.go`
- `pkg/inference/session/session.go`

The important current fact is that lifecycle is registration-driven today:

- `RegisterPrebuilt(...)` closes over one built runtime
- `NewLazyRegistrar(...)` builds a runtime on every invocation

That is the right pattern to keep. We should not reintroduce lifecycle semantics as a field on `EvalOptions`.

## Proposed direction

Add a new explicit per-session registration strategy, for example:

```go
type SessionKeyResolver func(ctx context.Context) (string, error)

type PerSessionOptions struct {
    ResolveSessionKey SessionKeyResolver
    IdleTTL           time.Duration
    MaxEntries        int
    OnMissingSession  MissingSessionBehavior
}

func RegisterPerSession[Scope any, Meta any](
    reg tools.ToolRegistry,
    spec EnvironmentSpec[Scope, Meta],
    resolveScope ScopeResolver[Scope],
    evalOpts EvalOptionOverrides,
    perSessionOpts PerSessionOptions,
) error
```

Default session identity should come from:

```go
session.SessionIDFromContext(ctx)
```

## Required behavior

- same session ID => same runtime reused across calls
- different session IDs => different isolated runtimes
- concurrent calls to one session runtime must not race
- idle runtimes must be evictable/cleanable
- prebuilt and lazy behavior must remain unchanged

## Main implementation pieces

1. Add a session-runtime pool keyed by session ID.
2. Add one entry lock per session runtime so `RunEval(...)` stays safe.
3. Add missing-session behavior that is explicit, not silent.
4. Add eviction and cleanup support.
5. Add tests for reuse, isolation, concurrency, and cleanup.

## Suggested tests

- same-session calls increment retained JS state
- different sessions do not see each other's JS state
- missing session ID behavior matches config
- concurrent same-session calls do not race
- eviction closes runtime cleanup hooks
- existing `RegisterPrebuilt(...)` and `NewLazyRegistrar(...)` behavior remains intact

## Notes

There is already a detailed local design and intern-facing implementation guide in:

- `geppetto/ttmp/2026/03/17/GP-37--add-per-session-runtime-lifecycle-support-to-scopedjs/design-doc/01-per-session-scopedjs-runtime-lifecycle-analysis-design-and-intern-implementation-guide.md`
