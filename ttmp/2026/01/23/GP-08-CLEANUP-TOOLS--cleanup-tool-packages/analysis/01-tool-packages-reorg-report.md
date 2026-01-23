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
LastUpdated: 2026-01-23T00:01:56-05:00
---

# GP-08 report: tool* package inventory and reorg proposal

## Executive summary

Geppetto’s tool-calling stack has converged on a clear “canonical” path (`toolloop` + `tools`) but still carries multiple legacy/overlapping packages and duplicated configuration types. The biggest sources of confusion are:

1) **Too many similarly-named config types** (`engine.ToolConfig`, `tools.ToolConfig`, `toolloop.ToolConfig`, `toolhelpers.ToolConfig`).
2) **Legacy orchestration code** (`toolhelpers.RunToolCallingLoop`) that overlaps almost 1:1 with `toolloop.Loop.RunLoop`.
3) **Small single-purpose packages** (`toolcontext`, `toolblocks`) that are useful but arguably in the “wrong place” (they’re cross-cutting helpers, not “inference features”).

This document inventories the tool* packages and proposes a target organization that reduces redundancy and makes it obvious which APIs are blessed for new code.

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
- `tools.ToolConfig` duplicates `engine.ToolConfig` almost field-for-field, which makes it unclear which one is authoritative.
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
- Provides the canonical `session.EngineBuilder` implementation:
  - `toolloop.EngineBuilder` + `toolloop.NewEngineBuilder(...)`.

**What’s good**
- This is the “right place” for orchestration and step-mode pausing.
- The functional option pattern is consistent with other Geppetto code.

**Needs attention**
- `toolloop.ToolConfig` overlaps with both `engine.ToolConfig` and `tools.ToolConfig` but is not identical. This makes “which config do I use where?” non-obvious.
- Option naming around the builder is still slightly awkward in a couple places:
  - `WithEngineBuilderSnapshotHook(...)` and `WithStepControllerService(...)` exist primarily to avoid name collisions with `Loop` options; we should align naming conventions (see proposal below).

### `geppetto/pkg/inference/toolcontext` (small but critical; placement questionable)

**Responsibility**
- Stores a `tools.ToolRegistry` in `context.Context` (runtime-only) and retrieves it.

**Consumers**
- Provider engines (OpenAI/Claude/Gemini/OpenAI-Responses) check `toolcontext.RegistryFrom(ctx)` to decide what tools to advertise in a request.
- `toolloop.Loop` sets `toolcontext.WithRegistry(ctx, l.registry)` before running inference.

**What’s good**
- Solves a real problem: avoids persisting runtime registries into `Turn.Data`.

**Needs attention**
- This feels like it belongs to the *tools substrate* (either `tools` itself or a sub-area), not a separate top-level tool* package.
- Naming: “toolcontext” is fine, but if we’re trying to reduce package count, this is the easiest merge candidate.

### `geppetto/pkg/inference/toolblocks` (helpful glue; placement questionable)

**Responsibility**
- Extract tool calls from Turn blocks and append tool results as blocks:
  - `ExtractPendingToolCalls(*turns.Turn) []ToolCall`
  - `AppendToolResultsBlocks(*turns.Turn, []ToolResult)`

**What’s good**
- Centralizes the “block walking” logic so tool loop and helpers don’t duplicate it.

**Needs attention**
- This is arguably a `turns` concern (“block helpers”), not an inference concern.
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
- Treat this as deprecated and either:
  - turn it into a strict compatibility wrapper around `toolloop` (preferred for external users), or
  - delete it and do a downstream migration sweep (more disruptive).

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

### Merge candidates (reduce package count)

3) Merge `toolcontext` into `tools`
   - Move:
     - `toolcontext.WithRegistry` -> `tools.WithRegistry` (or `tools.WithToolRegistry`)
     - `toolcontext.RegistryFrom` -> `tools.RegistryFrom` (or `tools.ToolRegistryFrom`)
   - Rationale: provider engines depend on it; it’s part of the core “tools substrate”.

4) Move `toolblocks` into `turns` (or rename for clarity)
   - Option A (preferred): `geppetto/pkg/turns/toolblocks` or `geppetto/pkg/turns/tools`
   - Option B: keep package but rename to make it obvious it’s about *Turn blocks*, not execution (e.g. `turntools`).

### Deprecation strategy

5) Deprecate `toolhelpers`
   - Add `// Deprecated:` GoDoc on package and exported identifiers.
   - Implement as a thin wrapper around `toolloop`:
     - `RunToolCallingLoop(...)` calls `toolloop.New(WithEngine, WithRegistry, WithConfig, WithExecutor?)`.
     - `toolhelpers.ToolConfig` becomes a type alias of `toolloop.ToolConfig` (or is removed).
     - `WithTurnSnapshotHook` becomes a wrapper around `toolloop.WithTurnSnapshotHook`.
   - This keeps external call sites working while preventing the reintroduction of new semantics.

## Proposed API normalization (reduce confusion)

### Make config layering explicit

We should distinguish:

- **Provider request config** (what is advertised/allowed; what the provider sees)
- **Loop control config** (max iterations, pause semantics, etc.)
- **Executor config** (timeouts, retries, masking, event publishing)

A cleaner end state could be:

```go
// used for provider advertisement and stored on Turn.Data (typed key)
type ToolPolicy struct {
  ToolChoice       ToolChoice
  AllowedTools     []string
  MaxParallelTools int
  ErrorHandling    ToolErrorHandling
}

// used by toolloop orchestration
type LoopConfig struct {
  MaxIterations int
  Timeout       time.Duration
}

// used by tools executor (retries, masking, etc.)
type ExecutorConfig struct {
  ExecutionTimeout time.Duration
  Retry            RetryConfig
}
```

Whether these live in `tools` or a dedicated `toolconfig` package is a design choice; the key is to avoid “same struct defined three times in three packages”.

### Builder options naming

Right now `toolloop` has both Loop options and EngineBuilder options. We should make naming consistent and predictable:

- `toolloop.New(...)` / `toolloop.WithEngine(...)` / `toolloop.WithRegistry(...)` for the loop.
- `toolloop.NewEngineBuilder(...)` / `toolloop.WithBase(...)` / `toolloop.WithMiddlewares(...)` for the builder.

The few “odd” names (`WithEngineBuilderSnapshotHook`, `WithStepControllerService`) should be revisited so they read naturally at call sites while still being unambiguous.

## Migration plan (mechanical steps)

1) Decide end-state package boundaries (merge toolcontext? move toolblocks?).
2) Add deprecation markers on `toolhelpers` and implement wrapper behavior.
3) Normalize config types (pick canonical owners; add adapter helpers for transitional period).
4) Update provider engines to import the new registry-from-context location (if toolcontext is merged).
5) Update docs and examples to only show canonical surfaces:
   - `toolloop.EngineBuilder` / `toolloop.NewEngineBuilder`
   - `toolloop.Loop`
   - `tools.ToolRegistry` / `tools.ToolExecutor`
6) Optional: add a simple `rg` check in CI or a lint rule to prevent new imports of deprecated packages.

## What is deprecated vs redundant vs OK-ish (quick classification)

**Deprecated (strong)**
- `geppetto/pkg/inference/toolhelpers` (keep only as compat wrapper, or delete after downstream migrations)

**Redundant / likely merge targets**
- `geppetto/pkg/inference/toolcontext` (merge into `tools`)
- “Config duplication” across `engine`, `tools`, `toolloop`, `toolhelpers`

**OK-ish but warrants attention**
- `geppetto/pkg/inference/toolblocks` (location/ownership and result block shape)
- `geppetto/pkg/inference/tools/adapters.go` (make “not implemented” parts explicit or finish them)

**Canonical / OK**
- `geppetto/pkg/inference/toolloop` (orchestration + step control + builder)
- `geppetto/pkg/inference/tools` (registry + executor + tool definitions)
