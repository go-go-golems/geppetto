---
Title: Diary
Ticket: GP-03-REMOVE-TOOL-MIDDLEWARE
Status: active
Topics:
    - geppetto
    - inference
    - tools
    - refactor
    - design
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/middleware/tool_middleware.go
      Note: Investigation target
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: Investigation target
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Investigation target
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T11:37:44.26377543-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Track the investigation and implementation work required to remove `middleware.NewToolMiddleware` and standardize tool execution on the tool-calling loop (and `session.ToolLoopEngineBuilder`).

## Step 1: Initial inventory of “tool calling middleware” vs “tool loop”

I started by confirming what “tool calling middleware” actually is in Geppetto today, and whether it’s meaningfully different from the existing tool-calling loop. The key outcome is that the middleware is already implementing a loop-like control flow (it repeatedly calls `next` and executes tools), but it uses a separate `Toolbox` API instead of the `tools.ToolRegistry` / `tools.ToolExecutor` path.

This matters because we can likely delete the middleware without losing functionality, as long as we update examples/docs/tests to use the `ToolLoopEngineBuilder` (which already runs `toolhelpers.RunToolCallingLoop` when `Registry` is set).

**Commit (code):** N/A

### What I did
- Searched for the middleware and tool-loop entrypoints:
  - `rg -n "NewToolMiddleware\\(" -S`
  - `rg -n "RunToolCallingLoop" geppetto/pkg -S`
  - `rg -n "ToolLoopEngineBuilder" geppetto/pkg -S`
- Read the core implementations:
  - `geppetto/pkg/inference/middleware/tool_middleware.go`
  - `geppetto/pkg/inference/toolhelpers/helpers.go`
  - `geppetto/pkg/inference/session/tool_loop_builder.go`

### Why
- We need to understand whether there are *actually* two distinct implementations (and therefore different semantics/features) before removing one.

### What worked
- Found that `ToolLoopEngineBuilder` already runs the tool loop when `Registry != nil`, so a “standardize on tool loop” refactor has a natural integration point.

### What didn't work
- N/A (investigation-only step).

### What I learned
- `NewToolMiddleware` is already a loop: it calls `next` repeatedly, extracts tool calls from blocks, executes them, appends tool results blocks, and continues.
- The main difference is the API surface: `Toolbox` vs `tools.ToolRegistry`/`tools.ToolExecutor`.

### What was tricky to build
- N/A (investigation-only step).

### What warrants a second pair of eyes
- Whether we want to preserve middleware’s “max-iterations soft cap” semantics when we standardize on `RunToolCallingLoop` (which currently returns an error on cap hit).

### What should be done in the future
- Write a migration plan doc that enumerates:
  - who uses `NewToolMiddleware` today,
  - the exact behavioral gaps between the two paths,
  - the target canonical API and the steps to delete the middleware safely.

### Code review instructions
- Start with:
  - `geppetto/pkg/inference/middleware/tool_middleware.go`
  - `geppetto/pkg/inference/toolhelpers/helpers.go`
  - `geppetto/pkg/inference/session/tool_loop_builder.go`

### Technical details
- Middleware uses `toolblocks.ExtractPendingToolCalls` / `toolblocks.AppendToolResultsBlocks` on turn blocks.
- Tool loop uses `toolcontext.WithRegistry` + `engine.KeyToolConfig` in `Turn.Data` and executes via `tools.ToolExecutor` (which can publish events).

## Step 2: Identify current users of ToolMiddleware

After establishing what the middleware is, I checked which parts of the codebase still use it. The important point is that, in this monorepo workspace, it appears to be used primarily by **examples + docs + middleware tests**, not by a core runtime.

That strongly suggests we can remove it without breaking “real” integration points, as long as we update the examples and documentation that still demonstrate it.

**Commit (code):** N/A

### What I did
- Found call sites excluding `ttmp/` narrative docs:
  - `rg -n "NewToolMiddleware\\(" -S --glob '!**/ttmp/**' /home/manuel/workspaces/2025-10-30/implement-openai-responses-api`
- Opened representative example wiring:
  - `geppetto/cmd/examples/middleware-inference/main.go`
  - `geppetto/cmd/examples/openai-tools/main.go`
  - `geppetto/cmd/examples/claude-tools/main.go`
- Noted test coverage relying on the middleware:
  - `geppetto/pkg/inference/middleware/tool_middleware_test.go`
  - `geppetto/pkg/inference/middleware/tool_middleware_turns_test.go`

### Why
- Removing middleware requires knowing what to migrate and what can be deleted.

### What worked
- Confirmed that non-`geppetto/` packages in this workspace do not import/use `middleware.NewToolMiddleware(...)`.

### What didn't work
- N/A

### What I learned
- “Real” tool calling usage (outside middleware) is already documented and supported via `toolhelpers.RunToolCallingLoop` and `session.ToolLoopEngineBuilder`.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Whether there are out-of-workspace downstream consumers (outside this monorepo) relying on the Toolbox API; if so, we may need a compatibility layer or a long deprecation window.

### What should be done in the future
- Write the analysis doc that compares Toolbox vs ToolRegistry path and proposes a deprecation/removal plan.

### Code review instructions
- Re-run the inventory command:
  - `rg -n "NewToolMiddleware\\(" -S --glob '!**/ttmp/**'`
