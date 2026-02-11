---
Title: Remove EngineWithMiddleware; fold into ToolLoopEngineBuilder
Ticket: GP-04-ENGINE-WITH-MIDDLEWARE-MOOT
Status: active
Topics:
    - geppetto
    - inference
    - middleware
    - refactor
    - design
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/middleware/middleware.go
      Note: Middleware core (`HandlerFunc`, `Middleware`, `Chain`)
    - Path: geppetto/pkg/inference/session/builder.go
      Note: EngineBuilder interface
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: ToolLoopEngineBuilder folds middleware chaining into builder
    - Path: geppetto/pkg/inference/session/tool_loop_builder_options.go
      Note: NewToolLoopEngineBuilder + functional options
    - Path: moments/backend/pkg/webchat/engine.go
      Note: Downstream app engine composition now uses builder (no NewEngineWithMiddleware)
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T12:37:42.646554215-05:00
WhatFor: ""
WhenToUse: ""
---

# Remove `EngineWithMiddleware` and standardize middleware composition on `ToolLoopEngineBuilder`

## Decision

- Delete the exported `middleware.NewEngineWithMiddleware` / `middleware.EngineWithMiddleware` API.
- Make middleware composition a responsibility of the session builder (`session.ToolLoopEngineBuilder`).
- Provide a constructor + functional options so callers can build a `ToolLoopEngineBuilder` without brittle struct literals:
  - `session.NewToolLoopEngineBuilder(...)`
  - `session.WithToolLoopBase(...)`, `session.WithToolLoopMiddlewares(...)`, etc.

## Why

Having both:

- “engine-level wrappers” (EngineWithMiddleware), and
- “session-level composition” (ToolLoopEngineBuilder),

creates an ongoing ambiguity about where middleware belongs. Middleware interacts with:

- tool loop orchestration,
- sinks (context-only),
- persistence/snapshots,

which are already owned by the builder. Standardizing on a single composition boundary reduces surface area and removes “two ways to do it”.

## Replacement design (pseudocode)

Inside `ToolLoopEngineBuilder.Build`:

```text
base := b.Base
handler := func(ctx, t) { return base.RunInference(ctx, t) }
handler = middleware.Chain(handler, b.Middlewares...)

runner := &engineWithMiddlewares{handler: handler} // unexported; implements engine.Engine
```

The returned runner is still an `engine.Engine` (same `RunInference` signature), so existing “I need an engine-like thing” call sites can use a built runner when needed.

## New builder construction pattern

```text
builder := session.NewToolLoopEngineBuilder(
  session.WithToolLoopBase(baseEngine),
  session.WithToolLoopMiddlewares(logMw, sysPromptMw),
  session.WithToolLoopEventSinks(sink),
  session.WithToolLoopRegistry(reg),
  session.WithToolLoopToolConfig(cfg),
)
```

## Migration checklist

- Update all call sites to stop calling `middleware.NewEngineWithMiddleware(...)`.
- For “engine composition” call sites (e.g. Pinocchio/Moments webchat engine builders), replace the wrapper chain with:
  - `session.NewToolLoopEngineBuilder(...).Build(context.Background(), "")` and return the resulting runner (it satisfies `engine.Engine`).
- Update examples and docs to use the builder constructor and `With...` options.
