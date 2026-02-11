---
Title: Refactor postmortem
Ticket: GP-04-ENGINE-WITH-MIDDLEWARE-MOOT
Status: active
Topics:
    - geppetto
    - inference
    - middleware
    - refactor
    - design
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/examples/simple-inference/main.go
      Note: Example migrated to builder-first middleware wiring
    - Path: geppetto/pkg/inference/middleware/middleware.go
      Note: Remaining middleware primitives (HandlerFunc/Middleware/Chain)
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: Builder now owns middleware chaining (replacement for EngineWithMiddleware)
    - Path: geppetto/pkg/inference/session/tool_loop_builder_options.go
      Note: NewToolLoopEngineBuilder functional options
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T16:39:53.968113401-05:00
WhatFor: ""
WhenToUse: ""
---


# Refactor postmortem

## Goal

Document the engineering details of the `EngineWithMiddleware` removal refactor:

- what changed, why it changed, and what it enables
- what worked, what was tricky, and what could still be wrong
- how to review/validate and how to migrate downstream code

## Context

Geppetto previously exposed an engine-level middleware wrapper API:

- `middleware.NewEngineWithMiddleware(base, mws...)` (returning an `engine.Engine`)

At the same time, Geppetto’s session integration already had a “composition root”:

- `session.ToolLoopEngineBuilder{ Base, Middlewares, EventSinks, Registry, ... }`

This created two competing abstraction boundaries for attaching cross-cutting concerns (logging, system prompt injection, agent-mode, sqlite tool middleware, etc.). The refactor removed the engine-level wrapper entirely and folded the “engine → handler → Chain(...)” logic into `ToolLoopEngineBuilder`.

In addition, we introduced a functional-options constructor to standardize builder creation:

- `session.NewToolLoopEngineBuilder(...)`
- `session.WithToolLoopBase(...)`, `session.WithToolLoopMiddlewares(...)`, …

## What I changed

### API changes

- Removed: `middleware.NewEngineWithMiddleware` / `middleware.EngineWithMiddleware`.
- Kept: `middleware.HandlerFunc`, `middleware.Middleware`, `middleware.Chain`.
- New: `session.NewToolLoopEngineBuilder(opts...) *ToolLoopEngineBuilder` and option helpers in `geppetto/pkg/inference/session/tool_loop_builder_options.go`.

### Behavioral intent (what should be the same)

- Middleware ordering semantics remain: `Middlewares` are applied **in-order** around the base engine (the same order as `middleware.Chain`).
- Tool loop execution is still owned by the builder (`Registry` toggles `RunToolCallingLoop`).
- Event sinks remain context-only; builder attaches sinks onto the run context.

### Key commits (for review)

- Code removal + fold into builder: `bdc03c1` — "tool-loop: remove NewEngineWithMiddleware helper"
- Docs + ticket updates: `2017adf` — "docs: move middleware composition to ToolLoopEngineBuilder"
- Pinocchio doc updates: `fbfc743` — "docs: remove NewEngineWithMiddleware references"
- Ticket diary update: `468f555` — "docs(GP-04): update diary for EngineWithMiddleware removal"

## What worked well

- The builder already owned the real integration concerns (tool loop, sinks, persistence), so it was the right place to attach middleware chaining.
- Because `InferenceRunner` and `engine.Engine` both expose `RunInference(ctx, *turns.Turn) (*turns.Turn, error)`, returning the built runner is a workable replacement for previous “engine composition” call sites that needed something engine-shaped.
- Updating examples to use `session.NewToolLoopEngineBuilder(...)` made the “happy path” easier to copy/paste correctly.

## What went wrong / sharp edges

- **Downstream composition helpers** (e.g. “composeEngineFromSettings”) previously returned an `engine.Engine` and used repeated wrapping.
  - After removal, those functions need to either:
    - return a built runner (`InferenceRunner` that also satisfies `engine.Engine`), or
    - be reworked so callers hold onto builder/session instead of an engine.
- **Middleware ordering** is easy to accidentally invert when migrating away from repeated wrapping patterns.
  - The prior approach often did `for i := len(uses)-1; i >= 0; i-- { eng = wrap(eng, mw) }` specifically to preserve “first listed = outermost”.
  - In the new world, you must be explicit about whether your `[]Middleware` is ordered outermost→innermost, and ensure it matches the old behavior.

## What warrants a second pair of eyes

- Verify middleware ordering semantics for app-specific stacks:
  - system prompt middleware should remain close to provider inference (typically innermost)
  - reorder middleware should remain close to provider inference (often innermost), unless it’s intended to reorder after other middlewares mutate blocks
  - idempotent system prompt middleware semantics should not rely on being “first”
- Confirm that returning a built runner where an `engine.Engine` was expected does not cause subtle type/interface mismatches in downstream packages (especially if they perform type assertions).
- Ensure docs no longer mention removed symbols in any “user-facing” documentation trees (non-`ttmp/`).

## Validation / how to review

### Commands

- `cd geppetto && go test ./... -count=1`
- `cd pinocchio && go test ./... -count=1`
- `cd moments/backend && go test ./... -count=1`

### Grep checks

- Ensure no non-temporary references remain:
  - `rg -n "NewEngineWithMiddleware\\(" -S --glob '!**/ttmp/**'`

## Quick Reference

### New canonical builder construction pattern

```go
builder := session.NewToolLoopEngineBuilder(
    session.WithToolLoopBase(baseEngine),
    session.WithToolLoopMiddlewares(logMw, sysPromptMw, toolMw),
    session.WithToolLoopEventSinks(sink),
    session.WithToolLoopRegistry(registry), // optional: enables tool loop
    session.WithToolLoopToolConfig(cfg),    // optional
)
```

## Usage Examples

### Replace `NewEngineWithMiddleware` in examples

Before:

```go
eng := middleware.NewEngineWithMiddleware(base, logMw)
sess.Builder = &session.ToolLoopEngineBuilder{Base: eng}
```

After:

```go
sess.Builder = session.NewToolLoopEngineBuilder(
    session.WithToolLoopBase(base),
    session.WithToolLoopMiddlewares(logMw),
)
```

## Related

- Ticket tasks: `geppetto/ttmp/2026/01/22/GP-04-ENGINE-WITH-MIDDLEWARE-MOOT--assess-enginewithmiddleware-vs-session-toolloopenginebuilder-middlewares/tasks.md`
- Analysis: `geppetto/ttmp/2026/01/22/GP-04-ENGINE-WITH-MIDDLEWARE-MOOT--assess-enginewithmiddleware-vs-session-toolloopenginebuilder-middlewares/analysis/01-is-enginewithmiddleware-obsolete-migration-to-session-builder-middlewares.md`
