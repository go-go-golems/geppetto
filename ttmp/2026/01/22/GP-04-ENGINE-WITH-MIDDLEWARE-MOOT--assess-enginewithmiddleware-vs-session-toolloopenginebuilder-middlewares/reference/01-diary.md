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

Track the investigation and implementation work to remove `middleware.NewEngineWithMiddleware` and standardize middleware composition on `session.ToolLoopEngineBuilder` (with a `NewToolLoopEngineBuilder(...WithOption)` constructor).

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

## Step 3: Decide to remove EngineWithMiddleware and fold into the builder

The direction changed from “should we discourage this wrapper?” to “we should delete it.” The main reason is API clarity: having both engine-level wrappers and builder-level middleware creates two competing composition roots. Since the builder already owns tool-loop behavior, sinks, persistence, and snapshots, it’s the right place for middleware chaining too.

This step corresponded to implementing the removal and moving the wrapper logic into `ToolLoopEngineBuilder.Build` as an unexported adapter.

**Commit (code):** bdc03c1 — "tool-loop: remove NewEngineWithMiddleware helper"

### What I did
- Removed the exported `middleware.NewEngineWithMiddleware` / `EngineWithMiddleware` API.
- Folded “engine -> HandlerFunc -> Chain(...)” into `session.ToolLoopEngineBuilder`.

### Why
- Avoid two public composition APIs (“engine-first” vs “builder-first”) and standardize on the session builder boundary.

### What worked
- Keeping the wrapper unexported preserves semantics while shrinking the public API.

### What didn't work
- N/A

### What I learned
- `InferenceRunner` is structurally compatible with `engine.Engine` (both expose `RunInference(ctx,*turns.Turn)`), which makes returning a built runner a workable replacement for “engine composition” call sites.

### What was tricky to build
- Preserving middleware ordering semantics (outermost vs innermost) while migrating call sites that previously wrapped engines repeatedly.

### What warrants a second pair of eyes
- Confirming that middleware ordering in downstream repos (Pinocchio/Moments) matches the historical wrapper semantics.

### What should be done in the future
- Convert remaining downstream repos that still reference `NewEngineWithMiddleware` (if any) or explicitly mark them “legacy/not supported”.

### Code review instructions
- Start with `geppetto/pkg/inference/session/tool_loop_builder.go` and confirm middleware chaining is internal and unexported.

### Technical details
- The builder now adapts `engine.Engine` to `middleware.HandlerFunc` and applies `middleware.Chain(...)` inside `Build`.

## Step 4: Add `NewToolLoopEngineBuilder` options constructor and migrate docs/examples

After the API removal, I updated public-facing examples and docs to demonstrate the new canonical pattern: create a builder via `session.NewToolLoopEngineBuilder(...)` and pass middlewares with `session.WithToolLoopMiddlewares(...)`.

**Commit (docs):** 2017adf — "docs: move middleware composition to ToolLoopEngineBuilder"

### What I did
- Updated Geppetto docs and middleware package docs to remove `NewEngineWithMiddleware` references and show builder-first composition.
- Updated the GP-04 ticket tasks and analysis doc to reflect the new direction.

### Why
- Keep the docs aligned with the new API so users don’t copy/paste removed symbols.

### What worked
- The new `With...` option helpers make snippets shorter and reduce struct-literal churn in examples.

### What didn't work
- N/A

### What I learned
- A “builder-first” story is much easier to keep coherent once we remove engine-level wrappers from the public API.

### What was tricky to build
- Updating examples that previously “wrapped and re-wrapped” engines, while still preserving middleware execution order.

### What warrants a second pair of eyes
- Review doc snippets for correctness (imports, intended ordering, and whether the examples still demonstrate best practices).

### What should be done in the future
- N/A

### Code review instructions
- Review the doc changes:
  - `geppetto/pkg/doc/topics/09-middlewares.md`
  - `geppetto/pkg/doc/topics/06-inference-engines.md`
  - `geppetto/pkg/doc/topics/07-tools.md`
  - `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md`
