---
Title: "GP-08 report: tool* package inventory and reorg proposal"
Ticket: GP-08-CLEANUP-TOOLS
Status: active
Topics:
  - geppetto
  - tools
  - toolloop
  - architecture
DocType: analysis
Intent: long-term
Owners: []
Summary: Detailed inventory of Geppetto’s tool* packages, with findings on deprecated/redundant areas and a proposed reorganization + migration plan.
LastUpdated: 2026-01-23T08:19:28-05:00
---

# GP-08 report: tool* package inventory and reorg proposal

## Executive summary

Geppetto’s tool-calling stack has converged on a clear “canonical” path (`toolloop` + `tools`) but still carries multiple legacy/overlapping packages and duplicated configuration types. The biggest sources of confusion are:

1) **Too many similarly-named config types** (`engine.ToolConfig`, `tools.ToolConfig`, `toolloop.ToolConfig`, `toolhelpers.ToolConfig`).
2) **Legacy orchestration code** (`toolhelpers.RunToolCallingLoop`) that overlaps almost 1:1 with `toolloop.Loop.RunLoop`.
3) **Small single-purpose packages** (`toolcontext`, `toolblocks`) that are useful but arguably in the “wrong place” (they’re cross-cutting helpers, not “inference features”).

This document inventories the tool* packages and proposes a target organization that reduces redundancy and makes it obvious which APIs are blessed for new code.

## Decisions (locked-in for GP-08)

The following decisions were made after the initial report draft. The rest of this document is written to match them:

- `tools.ToolConfig` is canonical (tool advertisement + execution policy).
- The tool loop has a separate configuration concept (not called `ToolConfig`).
- `toolcontext` should become part of `tools`.
- `toolblocks` should become part of `turns`.
- The session engine builder should live in a **toolloop subpackage** to avoid `With*` naming clashes.
- **No backwards compatibility**: no wrappers; update downstream repos and cut over.

## Scope

This report covers the following packages under `geppetto/pkg/inference`:

- `toolblocks`
- `toolcontext`
- `toolhelpers`
- `toolloop`
- `tools`

It also references the provider engines in `geppetto/pkg/steps/ai/*`, because they consume `toolcontext` and the “tool config in Turn.Data” contract.

## Inventory (what exists today)

### `geppetto/pkg/inference/tools` (core primitives; mostly OK)

**Responsibility**
- Tool definition model (`ToolDefinition`), registry (`ToolRegistry`), and execution (`ToolExecutor` / `BaseToolExecutor`).
- Provider-format adapters (`OpenAIToolAdapter`, `ClaudeToolAdapter`, `GeminiToolAdapter`) used by provider engines to serialize the “tools list” into provider requests.

**What’s good**
- Registry and executor are cleanly separated from orchestration (tool loop).
- `BaseToolExecutor` provides hook points and consistent event publishing.

**Needs attention**
- `tools.ToolConfig` duplicates `engine.ToolConfig` almost field-for-field, which makes it unclear which one is authoritative. GP-08’s direction is to make `tools.ToolConfig` canonical and converge other “tool config” representations onto it.
- Adapter APIs include partial/unimplemented stubs (e.g. `ConvertFromProviderResponse` returning “not implemented”), which is OK if intentionally unused, but should be made explicit.

**Dependency notes**
- Provider engines import `tools` (for adapters and tool registry types).

### `geppetto/pkg/inference/toolloop` (canonical orchestration; OK but still settling)

**Responsibility**
- Orchestrates the iterative “LLM -> tool_call -> execute -> tool_use -> continue” loop.
- Defines step-mode pause semantics:
  - `after_inference` (when tool calls are pending)
  - `after_tools`
- Owns a shared, application-owned `StepController` and publishes a Geppetto-native `debugger.pause` event.
- Provides the canonical engine builder implementation for session-style apps, but **in a dedicated subpackage** to avoid `With*` name clashes:
  - `toolloop/enginebuilder` (e.g. `enginebuilder.New(...)` + `enginebuilder.WithBase(...)`, etc.).

**What’s good**
- This is the “right place” for orchestration and step-mode pausing.
- The functional option pattern is consistent with other Geppetto code.

**Needs attention**
- `toolloop.ToolConfig` should not exist as a type name. The tool loop needs its own config, but GP-08’s direction is that **`tools.ToolConfig` is canonical** and the loop config should be a separate type (e.g. `toolloop.LoopConfig`).
- `toolloop/enginebuilder` should own the ergonomic `With*` option surface for builder composition without leaking naming workarounds into the main `toolloop` package.

### `geppetto/pkg/inference/toolcontext` (small but critical; placement questionable)

**Responsibility**
- Stores a `tools.ToolRegistry` in `context.Context` (runtime-only) and retrieves it.

**Consumers**
- Provider engines (OpenAI/Claude/Gemini/OpenAI-Responses) check `toolcontext.RegistryFrom(ctx)` to decide what tools to advertise in a request.
- `toolloop.Loop` sets `toolcontext.WithRegistry(ctx, l.registry)` before running inference.

**What’s good**
- Solves a real problem: avoids persisting runtime registries into `Turn.Data`.

**Needs attention**
- GP-08 decision: `toolcontext` moves into `tools` (no separate package).

### `geppetto/pkg/inference/toolblocks` (helpful glue; placement questionable)

**Responsibility**
- Extract tool calls from Turn blocks and append tool results as blocks:
  - `ExtractPendingToolCalls(*turns.Turn) []ToolCall`
  - `AppendToolResultsBlocks(*turns.Turn, []ToolResult)`

**What’s good**
- Centralizes the “block walking” logic so tool loop and helpers don’t duplicate it.

**Needs attention**
- GP-08 decision: `toolblocks` moves into `turns` (block model helpers live with the block model).
- It currently hard-codes a string-y “Error: …” payload shape when appending tool results, which is convenient but may not match what all providers/UI layers want long-term.

### `geppetto/pkg/inference/toolhelpers` (legacy; redundant)

**Responsibility**
- Provides a standalone `RunToolCallingLoop(...)` plus:
  - `toolhelpers.ToolConfig`
  - `toolhelpers.SnapshotHook` + `WithTurnSnapshotHook`
  - `ExecuteToolCallsTurn` helper

**Current usage**
- Within `geppetto` itself, current Go code no longer imports `toolhelpers` (only the package and its tests exist).

**Why this is redundant**
- `toolhelpers.RunToolCallingLoop` is essentially the same loop that now exists in `toolloop.Loop.RunLoop`, but without step-mode semantics and without the canonical EngineBuilder integration.
- It introduces a fourth “tool config” type and a second snapshot hook mechanism (which we already have in `toolloop`).

**Recommendation**
- GP-08 decision: delete it and do a downstream migration sweep (no wrappers, no compat layer).

## Cross-cutting problems (why this feels messy)

### 1) Config type duplication

We currently have overlapping configuration structs:

- `engine.ToolConfig` (stored in `Turn.Data` under `engine.KeyToolConfig`)
- `tools.ToolConfig` (executor + tool-level gating concerns)
- `toolloop.ToolConfig` (loop-level parameters + executor override)
- `toolhelpers.ToolConfig` (legacy copy of loop-level config)

This leads to unclear questions:
- Which config is “the source of truth” for providers when advertising tools?
- Which config is “the source of truth” for the executor when running tool calls?
- Which config do applications mutate (and where: Turn.Data vs builder options vs context)?

### 2) Too many places to learn “how tools work”

From a user’s perspective, tool calling requires understanding:
- Turn blocks conventions (`tool_call`, `tool_use`)
- Registry plumbing (context vs Turn.Data)
- EngineBuilder composition (middleware + sinks + tool registry)
- Tool loop orchestration
- Tool executor behavior and events

The package boundaries should help guide the reader, but right now they scatter the mental model.

## Proposed target organization (starting point)

This proposal focuses on clarity and a smaller set of “blessed” packages for new code.

### Canonical packages (new code should use these)

1) `geppetto/pkg/inference/tools`
   - Keep as “core primitives”: registry + executor + tool definitions + adapters.

2) `geppetto/pkg/inference/toolloop`
   - Keep as “orchestration”: loop + step control + builder + snapshot hook.
   - Builder should move to `geppetto/pkg/inference/toolloop/enginebuilder`.

### Merge candidates (reduce package count)

3) Move `toolcontext` into `tools` (decision)
   - Make registry-in-context part of the canonical “tools substrate”.

4) Move `toolblocks` into `turns` (decision)
   - Prefer placing helpers next to the Turn/block model to reduce mental hops and package count.

### Deprecation strategy

5) Remove `toolhelpers` (decision)
   - Do a hard cutover: update downstream repos (Pinocchio, Moments, go-go-mento, etc.) and delete `toolhelpers` entirely.
   - This is more disruptive short-term but avoids carrying and documenting legacy surfaces indefinitely.

## Proposed API normalization (reduce confusion)

### Make config layering explicit

We should distinguish:

- **Provider request config** (what is advertised/allowed; what the provider sees)
- **Loop control config** (max iterations, pause semantics, etc.)
- **Executor config** (timeouts, retries, masking, event publishing)

A cleaner end state could be:

```go
// canonical: used for provider advertisement + tool execution policy
// (this is tools.ToolConfig)
type ToolConfig struct {
  Enabled           bool
  ToolChoice        ToolChoice
  AllowedTools      []string
  MaxParallelTools  int
  ToolErrorHandling ToolErrorHandling
  ExecutionTimeout  time.Duration
  RetryConfig       RetryConfig
}

// used by toolloop orchestration (separate from tools.ToolConfig)
type LoopConfig struct {
  MaxIterations int
  // Step/pause config is owned elsewhere (StepController + pause timeout).
}
```

The key is to avoid “same struct defined three times in three packages”: GP-08’s direction is to make `tools.ToolConfig` canonical and keep only a minimal loop config in `toolloop`.

### Builder options naming

Right now `toolloop` has both Loop options and EngineBuilder options. To avoid clashes, GP-08’s direction is to move the builder to a subpackage and keep the main loop options focused on the loop:

- `toolloop.New(...)` / `toolloop.WithEngine(...)` / `toolloop.WithRegistry(...)` for the loop.
- `toolloop/enginebuilder.New(...)` / `enginebuilder.WithBase(...)` / `enginebuilder.WithMiddlewares(...)` for the builder.

This removes the need for “odd” names whose only purpose is disambiguation.

## Migration plan (mechanical steps)

1) Move `toolcontext` into `tools`; update provider engines + tool loop to import the new location.
2) Move `toolblocks` into `turns`; update tool loop and any other consumers.
3) Move engine builder into `toolloop/enginebuilder`; update call sites (Pinocchio, Moments, examples).
4) Make `tools.ToolConfig` canonical and remove `toolloop.ToolConfig`/`toolhelpers.ToolConfig` types.
5) Delete `toolhelpers`; update external users (go-go-mento, etc.) to `toolloop` + `tools`.
6) Update docs/examples to only show canonical surfaces (`toolloop.Loop`, `toolloop/enginebuilder`, `tools.ToolRegistry`, `tools.ToolExecutor`).

## What is deprecated vs redundant vs OK-ish (quick classification)

**Deprecated (strong)**
- `geppetto/pkg/inference/toolhelpers` (delete; no compat wrappers)

**Redundant / likely merge targets**
- `geppetto/pkg/inference/toolcontext` (move into `tools`)
- “Config duplication” across `engine`, `tools`, `toolloop`, `toolhelpers`

**OK-ish but warrants attention**
- `geppetto/pkg/inference/toolblocks` (move into `turns`; revisit result block shape)
- `geppetto/pkg/inference/tools/adapters.go` (make “not implemented” parts explicit or finish them)

**Canonical / OK**
- `geppetto/pkg/inference/toolloop` (orchestration + step control + builder)
- `geppetto/pkg/inference/tools` (registry + executor + tool definitions)
