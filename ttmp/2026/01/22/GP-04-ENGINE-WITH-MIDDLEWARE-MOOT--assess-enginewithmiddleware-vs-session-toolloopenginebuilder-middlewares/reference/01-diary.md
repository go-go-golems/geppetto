---
Title: Diary
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
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T12:37:42.830073638-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Track the investigation work for whether `middleware.EngineWithMiddleware` should be treated as a legacy/public API (discouraged in app code) now that `session.ToolLoopEngineBuilder` accepts middlewares.

## Step 1: Inventory `EngineWithMiddleware` usage and builder integration

I started by locating the implementation of `EngineWithMiddleware`, confirming what it actually does (an `engine.Engine` → `HandlerFunc` adapter + `Chain(...)`), and then finding where it is used. The key outcome is that it is *not* currently “moot” as an implementation detail: `session.ToolLoopEngineBuilder` uses it internally when configured with `Middlewares`.

This reframes the question from “can we delete it?” to “should callers stop reaching for this wrapper directly, and instead configure middlewares via the session builder for chat-style apps?”

**Commit (code):** N/A

### What I did
- Read the wrapper implementation:
  - `sed -n '1,220p' geppetto/pkg/inference/middleware/middleware.go`
- Confirmed session-level composition entrypoints:
  - `sed -n '1,220p' geppetto/pkg/inference/session/builder.go`
  - `sed -n '1,240p' geppetto/pkg/inference/session/tool_loop_builder.go`
- Inventoried call sites (excluding `ttmp/`):
  - `rg -n "NewEngineWithMiddleware\\(" geppetto -S --glob '!**/ttmp/**'`
  - `rg -n "NewEngineWithMiddleware\\(" pinocchio go-go-mento moments -S --glob '!**/ttmp/**'`

### Why
- We can’t judge “obsolescence” without knowing:
  - whether the session builder already covers the use cases, and
  - how much downstream code depends on direct wrapper composition.

### What worked
- Found that `ToolLoopEngineBuilder` already supports a middleware slice and applies it via `middleware.NewEngineWithMiddleware`.

### What didn't work
- N/A (investigation-only step).

### What I learned
- `EngineWithMiddleware` is used in:
  - Geppetto core session builder (internal)
  - Geppetto examples and docs (public-facing “how to compose” guidance)
  - Downstream repos’ engine composition code (Pinocchio/Moments/go-go-mento)

### What was tricky to build
- N/A (investigation-only step).

### What warrants a second pair of eyes
- If we change docs/examples to prefer builder-based composition, confirm we’re not missing a non-session “engine composition” use case that should remain first-class.

### What should be done in the future
- Decide whether:
  - to keep `NewEngineWithMiddleware` as a low-level convenience API, or
  - to deprecate it for app-level usage (keeping it internal), and migrate examples/docs accordingly.

### Code review instructions
- Start at:
  - `geppetto/pkg/inference/middleware/middleware.go`
  - `geppetto/pkg/inference/session/tool_loop_builder.go`

### Technical details
- `ToolLoopEngineBuilder.Build` currently does:
  - `eng := b.Base`
  - `if len(b.Middlewares) > 0 { eng = middleware.NewEngineWithMiddleware(eng, b.Middlewares...) }`

# Step 2: Write up analysis and propose migration plan

Captured the findings in an analysis doc under this ticket, focused on the distinction between:

- `EngineWithMiddleware` being *moot as a public recommendation* (for chat-style apps), vs
- `EngineWithMiddleware` being *still required as an internal helper* of the builder today.

**Commit (code):** N/A
