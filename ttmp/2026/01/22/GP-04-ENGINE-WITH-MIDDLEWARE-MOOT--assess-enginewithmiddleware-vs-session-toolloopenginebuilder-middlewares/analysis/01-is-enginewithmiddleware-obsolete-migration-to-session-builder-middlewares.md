---
Title: Is EngineWithMiddleware obsolete? Migration to Session builder middlewares
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
      Note: EngineWithMiddleware definition and chain wrapper
    - Path: geppetto/pkg/inference/session/builder.go
      Note: EngineBuilder interface
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: ToolLoopEngineBuilder applies middlewares via NewEngineWithMiddleware
    - Path: moments/backend/pkg/webchat/engine.go
      Note: Downstream app engine composition pattern (repeated wrapping)
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T12:37:42.646554215-05:00
WhatFor: ""
WhenToUse: ""
---


# Is `EngineWithMiddleware` “moot” now that we have a Session `EngineBuilder` with middlewares?

## Goal

Assess whether callers should stop using `middleware.NewEngineWithMiddleware(...)` directly, now that the standard session integration (`session.ToolLoopEngineBuilder`) accepts a `Middlewares []middleware.Middleware` field.

This is not purely a “can we delete it?” question. `ToolLoopEngineBuilder` currently uses `NewEngineWithMiddleware` internally, so “moot” here primarily means: **is this wrapper still a recommended app-level composition API, or should it become an internal helper?**

## Current APIs (signatures + behavior)

### Middleware package

Defined in `geppetto/pkg/inference/middleware/middleware.go`:

- `type HandlerFunc func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)`
- `type Middleware func(HandlerFunc) HandlerFunc`
- `func Chain(handler HandlerFunc, middlewares ...Middleware) HandlerFunc`
- `func NewEngineWithMiddleware(e engine.Engine, middlewares ...Middleware) *EngineWithMiddleware`
- `type EngineWithMiddleware struct { handler HandlerFunc }`
- `func (e *EngineWithMiddleware) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)`

Key properties:

- It’s just an adapter: `engine.Engine` → `HandlerFunc` → `Chain(...)`.
- It returns a concrete `*EngineWithMiddleware`, which itself implements `engine.Engine`.
- It is “session-unaware” by design: it neither creates nor enforces session identifiers; it simply wraps `RunInference`.

### Session builder package

Defined in `geppetto/pkg/inference/session/builder.go`:

- `type EngineBuilder interface { Build(ctx context.Context, sessionID string) (InferenceRunner, error) }`

The main builder used by chat-style apps is `session.ToolLoopEngineBuilder` in `geppetto/pkg/inference/session/tool_loop_builder.go`, which includes:

- `Middlewares []middleware.Middleware`
- `Build(ctx, sessionID)` currently applies them by wrapping the base engine:
  - `eng = middleware.NewEngineWithMiddleware(eng, b.Middlewares...)`

## Where it is used today (“who uses it?”)

### Inside Geppetto (core)

- `geppetto/pkg/inference/session/tool_loop_builder.go`
  - uses `middleware.NewEngineWithMiddleware` when `b.Middlewares` is non-empty.

So **it is not moot as an implementation detail** of the standard builder.

### In Geppetto (examples + docs)

Representative call sites:

- `geppetto/cmd/examples/simple-inference/main.go`
- `geppetto/cmd/examples/simple-streaming-inference/main.go`
- `geppetto/cmd/examples/middleware-inference/main.go`
- `geppetto/cmd/examples/generic-tool-calling/main.go`
- `geppetto/cmd/examples/openai-tools/main.go`
- `geppetto/cmd/examples/claude-tools/main.go`
- Docs/tutorials in `geppetto/pkg/doc/topics/*` and middleware package docs

Pattern: examples frequently show “wrap the engine directly”, rather than “configure a session builder”.

### Outside Geppetto (workspace downstream repos)

In this monorepo workspace, additional usage exists in:

- `pinocchio/...` (examples + webchat engine composition)
- `moments/backend/...` (webchat engine composition)
- `go-go-mento/go/...` (webchat engine composition)

Pattern: those codebases compose multiple middlewares by repeatedly re-wrapping:

```text
eng = middleware.NewEngineWithMiddleware(eng, mw1)
eng = middleware.NewEngineWithMiddleware(eng, mw2)
...
```

## Why `EngineWithMiddleware` feels redundant (as a public “mainline” API)

If your application is already operating at the “session orchestration” level (long-lived chat, tool calling loop, persistence, event sinks), the composition surface is now:

- `session.Session` + `session.EngineBuilder` (most commonly `ToolLoopEngineBuilder`)

and therefore:

- **middlewares can be passed into the builder**, without requiring the caller to wrap provider engines directly.

This makes `NewEngineWithMiddleware` *functionally redundant* for app code in the common case, because:

- builders already need to own policy around tools, sinks, persistence, and safety caps
- the builder can consistently apply middleware once (in one place), avoiding repeated wrapping patterns spread across call sites

## Why it is still useful (and likely shouldn’t be deleted immediately)

There are valid use cases for the wrapper:

- Non-session or non-chat usage where you just want a single `engine.Engine` instance with a few cross-cutting concerns.
- Tests where you want a minimal wrapper chain without introducing session machinery.
- Legacy integrations (including downstream repos) where “engine composition” is still the primary abstraction.

Also: `ToolLoopEngineBuilder` currently depends on it internally, so deletion implies either:

- duplicating the chain logic inside session, or
- moving middleware chaining into a lower-level helper (which is essentially the same thing)

## Proposed direction (migration path)

### Recommendation for applications

- Prefer `session.ToolLoopEngineBuilder` (or another `session.EngineBuilder`) as the composition root for chat-style applications.
- Use `ToolLoopEngineBuilder.Middlewares` rather than wrapping `Base` with `NewEngineWithMiddleware` at call sites.

### Recommendation for `middleware.NewEngineWithMiddleware`

Treat it as:

- a low-level convenience helper (still supported), but
- not the primary “happy path” showcased in docs/examples for chat+tools.

If we want to be explicit, the doc/story changes could be:

- Update examples to build sessions with `ToolLoopEngineBuilder` and pass middlewares there.
- Leave `NewEngineWithMiddleware` in place for lower-level/advanced usage.

### Potential API/documentation cleanup

Some docs show the signature as returning `engine.Engine` (rather than `*EngineWithMiddleware`). That’s semantically fine (it implements the interface), but mismatched to the actual signature.

Options:

- Keep code as-is; fix docs to match.
- Or change `NewEngineWithMiddleware` to return `engine.Engine` for an interface-first public API (note: would be a breaking change for callers relying on the concrete type).

## Concrete “how would code look” (pseudocode)

### Before (direct wrapping)

```text
provider := openai.NewEngine(...)
eng := middleware.NewEngineWithMiddleware(provider, logMw, sysPromptMw)

// somewhere else, tool loop/persistence wiring happens out-of-band
```

### After (builder owns composition)

```text
builder := &session.ToolLoopEngineBuilder{
  Base: provider,
  Middlewares: []middleware.Middleware{logMw, sysPromptMw},
  Registry: registry,          // enable tool loop
  ToolConfig: cfg,
  EventSinks: sinks,
  SnapshotHook: hook,
  Persister: persister,
}

s := session.NewSession()
s.Builder = builder
s.Append(seedTurn)
handle := s.StartInference(...)
```

## Open questions / decisions to make

1. Do we want to actively deprecate “engine-level composition” in docs, or just provide both patterns?
2. Should `EngineWithMiddleware` remain exported, or become an internal helper used by session builders?
3. If we keep it exported, should we standardize on a single “composition root” recommendation:
   - “Use session builder for chat-like apps; use wrapper for single-run scripts/tests”?

## Suggested next steps (ticket-level)

- Update the example programs to demonstrate the session builder path.
- Update docs that currently teach “wrap the engine” as the main way to attach cross-cutting concerns.
- (Optional) add a short section to middleware docs: “If you are using sessions, prefer configuring builder middlewares.”
