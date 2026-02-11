---
Title: Toolbox vs ToolRegistry tool-calling paths; migration plan
Ticket: GP-03-REMOVE-TOOL-MIDDLEWARE
Status: active
Topics:
    - geppetto
    - inference
    - tools
    - refactor
    - design
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/middleware/tool_middleware.go
      Note: Toolbox-based loop-like middleware to remove
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: Preferred integration point (ToolLoopEngineBuilder)
    - Path: geppetto/pkg/inference/toolblocks/toolblocks.go
      Note: Block-level tool_call/tool_use extraction+append
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: Canonical tool loop implementation
    - Path: geppetto/pkg/inference/tools/base_executor.go
      Note: Tool execution orchestration + events
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T11:37:44.105454423-05:00
WhatFor: ""
WhenToUse: ""
---


# Toolbox vs ToolRegistry tool-calling paths (and how to remove ToolMiddleware)

## Executive summary

Geppetto currently has **two** tool-execution orchestration surfaces:

1. **Tool-calling middleware**: `middleware.NewToolMiddleware(toolbox, cfg)` (Toolbox-based).
2. **Tool-calling loop**: `toolhelpers.RunToolCallingLoop(ctx, eng, turn, registry, cfg)` (ToolRegistry/ToolExecutor-based), also used by `session.ToolLoopEngineBuilder` when `Registry != nil`.

They both implement a “run inference → extract `tool_call` blocks → execute tools → append `tool_use` blocks → repeat” control flow, but they differ in:

- how tools are represented (ad-hoc `Toolbox` vs typed `tools.ToolRegistry`),
- how execution is configured (middleware-only config vs shared config + executor hooks),
- whether tool execution publishes structured events (loop does; middleware does not),
- concurrency/retry/allowed-tools semantics,
- error semantics (max-iterations currently behaves differently).

This ticket proposes to standardize on the **tool loop** (and/or `ToolLoopEngineBuilder`) and remove the tool middleware, updating examples/docs/tests accordingly.

**Update (2026-01-22):** In this workspace state, the Toolbox-based tool middleware (`middleware.NewToolMiddleware`) has been removed. Examples/docs were migrated to use the tool loop runner (`toolhelpers.RunToolCallingLoop`) via `session.ToolLoopEngineBuilder` with a `tools.ToolRegistry`.

## Glossary

- **Turn**: `*turns.Turn` containing `Blocks` with kinds `tool_call` / `tool_use`.
- **ToolRegistry**: `tools.ToolRegistry` (thread-safe registry of `tools.ToolDefinition`).
- **ToolExecutor**: `tools.ToolExecutor` (executes `tools.ToolCall` against a registry; supports hooks/events via `BaseToolExecutor`).
- **Tool loop**: `toolhelpers.RunToolCallingLoop`.
- **Tool middleware**: `middleware.NewToolMiddleware`.

## Current code (as of 2026-01-22)

### Tool middleware path (Toolbox-based)

**Primary implementation**

- `geppetto/pkg/inference/middleware/tool_middleware.go`

**API shape**

```go
// geppetto/pkg/inference/middleware/tool_middleware.go
type Toolbox interface {
    ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error)
    GetToolDescriptions() []ToolDescription
}

func NewToolMiddleware(toolbox Toolbox, config ToolConfig) Middleware
```

**Behavior**

- Calls `next(ctx, currentTurn)` to run provider inference.
- Uses `toolblocks.ExtractPendingToolCalls(updated)` to find `tool_call` blocks without corresponding `tool_use`.
- Executes each tool call via `toolbox.ExecuteTool(ctx, name, args)` with per-call timeout.
- Appends `tool_use` blocks via `toolblocks.AppendToolResultsBlocks(updated, results)`.
- Repeats up to `config.MaxIterations`.
- If it hits max iterations, it returns the current turn **without error** (“soft cap”).

**Notable “extra” behavior**

- Tool allow-list can be overridden per-turn using `turns.KeyAgentModeAllowedTools` if present.

### Tool loop path (ToolRegistry/ToolExecutor-based)

**Primary implementation**

- `geppetto/pkg/inference/toolhelpers/helpers.go`

**API shape**

```go
// geppetto/pkg/inference/toolhelpers/helpers.go
func RunToolCallingLoop(
    ctx context.Context,
    eng engine.Engine,
    initialTurn *turns.Turn,
    registry tools.ToolRegistry,
    config toolhelpers.ToolConfig,
) (*turns.Turn, error)
```

**Behavior**

- Attaches registry to context: `toolcontext.WithRegistry(ctx, registry)`.
- Writes `engine.KeyToolConfig` into `Turn.Data` (enabled/toolChoice/maxIterations/timeouts/allowedTools/...).
- Runs `eng.RunInference(ctx, turn)` (provider appends `tool_call` blocks).
- Extracts tool calls from blocks via `toolblocks.ExtractPendingToolCalls(updated)`.
- Executes tools via `tools.ToolExecutor` (defaulting to `tools.NewDefaultToolExecutor(tools.DefaultToolConfig())`).
  - `BaseToolExecutor` publishes `tool_call.execute` and `tool_result` events through the context event sink hooks.
  - Supports allow-list checks (`ToolConfig.AllowedTools`), retries/backoff, and parallel execution.
- Appends `tool_use` blocks via `toolblocks.AppendToolResultsBlocks(updated, results)`.
- Repeats up to `config.MaxIterations`.
- If it hits max iterations, it returns an **error** (`fmt.Errorf("max iterations (%d) reached", ...)`).

### Session runner integration

`session.ToolLoopEngineBuilder` is already the canonical integration point for “chat-style apps”:

- `geppetto/pkg/inference/session/tool_loop_builder.go`

Behavior:

- If `ToolLoopEngineBuilder.Registry == nil`, runs a single `eng.RunInference`.
- Else runs `toolhelpers.RunToolCallingLoop(...)`.

So: the “tool loop” is already the preferred orchestration path at the session level.

## Who used ToolMiddleware?

Prior to removal (excluding `ttmp/` docs), it was used by:

- Examples:
  - `geppetto/cmd/examples/middleware-inference/main.go`
  - `geppetto/cmd/examples/openai-tools/main.go`
  - `geppetto/cmd/examples/claude-tools/main.go`
- Package tests:
  - `geppetto/pkg/inference/middleware/tool_middleware_test.go`
  - `geppetto/pkg/inference/middleware/tool_middleware_turns_test.go`
- Documentation pages mention it (migration candidates):
  - `geppetto/pkg/doc/topics/07-tools.md`
  - `geppetto/pkg/doc/topics/09-middlewares.md`

In this workspace state, these call sites have been migrated and the middleware/tests were deleted. (Out-of-workspace downstream consumers may still exist.)

## Toolbox vs ToolRegistry (differences that matter)

### 1) Tool representation

**Toolbox**

- Tools are identified by `name` and executed by passing a `map[string]interface{}`.
- There is no strong schema: descriptions are `map[string]interface{}` and not wired into provider integration directly.
- No standardized lifecycle hooks (pre-execute mutation, masking, publish events, retries).

**ToolRegistry + ToolDefinition**

- Tools are registered as `tools.ToolDefinition`:
  - Schema: `*jsonschema.Schema`
  - Exec: `ToolFunc` with type-safe invocation and optional `context.Context` first parameter.
- Tool calls are `tools.ToolCall{ID, Name, Arguments json.RawMessage}`, so the argument boundary is explicitly JSON.

### 2) Execution orchestration and events

**Toolbox path**

- Execution is synchronous, serial, and local to the middleware.
- It does not publish tool call / tool result events (only the engine and other helpers may publish).
- Timeout is per-call via `context.WithTimeout`.

**ToolExecutor path**

- Execution can be parallelized, retried with backoff, aborted/continued based on policy.
- `BaseToolExecutor` publishes tool execution events (`events.NewToolCallExecuteEvent`, `events.NewToolCallExecutionResultEvent`) via the context event sinks.

### 3) Configuration surface

**ToolMiddleware.ToolConfig**

- `MaxIterations`, per-call `Timeout`, `ToolFilter`.
- Also reads turn-scoped allow-list via `turns.KeyAgentModeAllowedTools`.

**Tool loop config**

- `toolhelpers.ToolConfig` maps to `engine.ToolConfig` (in `Turn.Data`) plus executor knobs:
  - tool choice, parallelism, allowed tools, error handling, executor, etc.

### 4) Error semantics

Today:

- Middleware: max-iterations is a soft cap (returns last turn without error).
- Tool loop: max-iterations is an error.

If we remove middleware, we need to decide which behavior we want as canonical and align (either make the tool loop soft-cap, or adjust call sites to treat “max iterations reached” as a non-fatal terminal state).

## Recommended target state

### Standard orchestration API

Primary:

- Session apps: use `session.ToolLoopEngineBuilder{Registry: reg, ToolConfig: &cfg}` and do not install `NewToolMiddleware`.

Secondary (lower-level):

- Non-session code: call `toolhelpers.RunToolCallingLoop(ctx, eng, turn, reg, cfg)` directly.

### Remove

- Remove `geppetto/pkg/inference/middleware/tool_middleware.go` and related tests/docs/examples that depend on it.

Optional:

- Keep a small compatibility adapter package (NOT middleware) if there are external users of Toolbox:
  - `toolboxadapter` could wrap a `tools.ToolRegistry` and `tools.ToolExecutor` to expose a `Toolbox`-like `ExecuteTool` (but this undermines the goal of removing dual APIs).

## Migration plan (concrete)

1. Update docs to stop recommending tool middleware:
   - Replace snippets in `geppetto/pkg/doc/topics/07-tools.md` and `geppetto/pkg/doc/topics/09-middlewares.md` with either:
     - `ToolLoopEngineBuilder.Registry` usage, or
     - direct `RunToolCallingLoop`.
2. Update examples that currently wrap engines with tool middleware:
   - `geppetto/cmd/examples/openai-tools/main.go`
   - `geppetto/cmd/examples/claude-tools/main.go`
   - `geppetto/cmd/examples/middleware-inference/main.go`
   to use `ToolLoopEngineBuilder.Registry` and/or `RunToolCallingLoop` directly.
3. Decide max-iteration semantics and implement consistently:
   - Option A: change `RunToolCallingLoop` to return `(turn, nil)` when the cap is hit.
   - Option B: keep the error but ensure callers treat it as expected/terminal for tool loops.
4. Delete `tool_middleware.go` and its tests, or convert the tests to validate the loop path.
5. Run `go test ./...` and `make lint` from `geppetto/`.

## Open questions (need explicit decisions)

1. Do we still need “tool execution as middleware” as a concept for engine composition, or is “tool loop runner” sufficient?
2. Should `RunToolCallingLoop` be the only supported tool execution path, and `ToolLoopEngineBuilder` the recommended integration?
3. If we keep `turns.KeyAgentModeAllowedTools`, where should it be enforced (executor hook vs tool loop config building)?
