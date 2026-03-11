---
Title: implementation plan for persisting serializable tool definitions on turn data
Ticket: GP-32-TURN-TOOL-DEFINITIONS
Status: active
Topics:
    - geppetto
    - schema
    - persistence
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/engine/types.go
      Note: Engine-owned runtime and persisted tool-definition types define the inspection contract and the runtime/provider boundary
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Tool loop already stamps ToolConfig and is the natural place to stamp persisted tool definitions
    - Path: geppetto/pkg/inference/tools/definition.go
      Note: Runtime tool definitions originate here and need conversion into a serializable per-turn representation
    - Path: geppetto/pkg/js/modules/geppetto/codec.go
      Note: JS turn-data short-key codec should expose the new persisted tool_definitions key
    - Path: geppetto/pkg/spec/geppetto_codegen.yaml
      Note: Generated turn keys and JS/TS constants need a new tool_definitions entry
    - Path: geppetto/pkg/steps/ai/claude/engine_claude.go
      Note: Claude provider path continues to advertise from the live registry in context
    - Path: geppetto/pkg/steps/ai/gemini/engine_gemini.go
      Note: Gemini provider path continues to read the live registry in context
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: OpenAI provider path now uses a shared context-only helper for runtime advertisement
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Responses provider path now uses a shared context-only helper for runtime advertisement
    - Path: geppetto/pkg/turns/serde/serde_test.go
      Note: Serde regression coverage should prove tool_definitions round-trip through YAML
ExternalSources: []
Summary: Implementation plan for adding a durable `tool_definitions` turn-data contract in Geppetto for inspection and debugging, while keeping provider tool schemas sourced from the live runtime registry.
LastUpdated: 2026-03-10T21:30:30.306664583-04:00
WhatFor: Explain how to add a durable `tool_definitions` turn-data contract in Geppetto for inspection and debugging without changing the live runtime-registry-based provider advertisement model.
WhenToUse: Use when implementing or reviewing the Geppetto change that persists tool schemas on `Turn.Data` while preserving context-carried runtime execution and provider advertisement.
---


# implementation plan for persisting serializable tool definitions on turn data

## Executive Summary

Geppetto already made the important architectural move of taking the live `ToolRegistry` out of `Turn.Data` and carrying it through `context.Context` for runtime execution. That solved the "runtime object inside a persisted turn" problem, but it left one gap unresolved in observability: persisted turns still do not carry a durable snapshot of tool definitions for inspection.

The recommended implementation is:

1. introduce a new typed `Turn.Data` key for `tool_definitions`,
2. stamp a serializable snapshot of the registry's advertised definitions onto the turn before inference,
3. keep provider engines advertising from the live registry in context,
4. keep the live registry in context exclusively for execution.

This gives Geppetto a clean split:

- `Turn.Data` = durable, inspectable snapshot state
- `context.Context` = live executable runtime services

That is the best long-term base for downstream inspectors like Temporal Relationships, Pinocchio debug tooling, or replay systems.

## Implementation Outcome

The implemented shape kept the intended authority boundary and made one important persistence correction:

- `tool_definitions` is persisted on `Turn.Data` as `engine.ToolDefinitions`
- each entry is an `engine.ToolDefinitionSnapshot`
- `parameters` are stored as `map[string]any`, not `*jsonschema.Schema`
- runtime provider advertisement remains sourced from the live registry in `context.Context`
- OpenAI and OpenAI Responses now use a shared context-only helper to derive runtime-advertised tool definitions

The `map[string]any` choice for `parameters` was necessary because `*jsonschema.Schema` did not round-trip safely through the typed-map YAML/JSON serde path.

## Problem Statement

Today Geppetto has an incomplete state split.

What already exists:

- `engine.KeyToolConfig` is written to `Turn.Data`.
- The live `tools.ToolRegistry` is attached to `context.Context`.
- The December 2025 design docs explicitly propose persisting serializable tool definitions on the turn.

What is still true in code:

- OpenAI, Claude, Gemini, and OpenAI Responses engines still read the registry from context to build provider tool schemas.
- There is no first-class `tool_definitions` typed key on `Turn.Data`.
- Persisted turns do not fully answer "which tool schemas were configured for this turn?" unless you can reconstruct the runtime registry from the original code and runtime composition.

That means:

- historical turn inspection is incomplete,
- deterministic replay is weaker than it should be,
- downstream apps have to choose between showing only `ToolConfig` or reverse-engineering live registries,
- Geppetto's own intended "serializable turn truth" model is not finished for tool inspection.

## Proposed Solution

### 1. Add a new typed turn-data key

Add a `tool_definitions` key to the generated turn key schema in `pkg/spec/geppetto_codegen.yaml`.

Recommended key shape:

- value key: `tool_definitions`
- typed key: `KeyToolDefinitions`
- family: `Turn.Data`
- owner: likely `engine` for minimal provider-engine friction

### 2. Choose the persisted payload type deliberately

There are three plausible representations:

1. `[]tools.ToolDefinition`
2. `[]engine.ToolDefinition`
3. a new dedicated serializable type such as `[]engine.ToolAdvertisement`

Implementation outcome:

- use a dedicated serializable snapshot type
- store `parameters` as a plain JSON object map
- exclude runtime-only fields entirely from the persisted representation

The persisted fields should include:

- `name`
- `description`
- `parameters`
- optionally `examples`
- optionally `tags`
- optionally `version`

The persisted fields must not include:

- executable function pointers
- registry internals
- runtime-only state

### 3. Stamp definitions onto the turn inside the tool loop

The tool loop already:

- attaches the live registry to context,
- writes `ToolConfig` to `Turn.Data`.

It should also write `tool_definitions` derived from `registry.ListTools()`.

Recommended timing:

- stamp definitions before the first engine inference call in `toolloop.RunLoop`,
- keep them on the same evolving turn so all persisted snapshots carry the same advertised schema set for that iteration series.

### 4. Keep provider engines on the live registry

Provider engines should continue to resolve advertised tools from the live registry in `context.Context`.

The new persisted `tool_definitions` key is explicitly for:

- inspection,
- persistence,
- debugging,
- downstream readback APIs,
- historical comparison.

It is not the authoritative input for provider request building in this ticket.

### 5. Keep runtime execution on context registry

No change to execution ownership:

- tool calls are still executed against the live registry in context,
- `tool_definitions` are for inspection only,
- if the persisted definitions diverge from the live registry at runtime, execution still depends on the registry.

The long-term invariant is:

- definitions explain what was configured around the turn,
- registry explains what the runtime can execute.

## Design Decisions

### Decision: persist tool definitions on `Turn.Data`

Rationale:

- turns are the durable unit of truth in Geppetto,
- downstream inspection and replay need a stable snapshot,
- this matches the prior context-carried-registry design docs.

### Decision: keep the live registry in context for execution

Rationale:

- executors are runtime services, not serializable data,
- execution needs actual code, not just schemas,
- mixing runtime registry objects back into turn state would regress the design.

### Decision: provider advertisement remains sourced from the live registry

Rationale:

- it avoids changing runtime semantics in the same ticket,
- the live registry remains the clear source of truth for what can be advertised and executed,
- the persisted snapshot can lag or differ without affecting model/runtime behavior.

### Decision: keep `responses_server_tools` separate for now

Rationale:

- OpenAI Responses built-in server-side tools are already modeled with a dedicated turn key,
- function tool definitions and built-in server tools have different shapes and lifecycles,
- conflating them in the first patch would increase scope and blur semantics.

## Alternatives Considered

### Alternative A: leave schemas derived from context registry only and do not persist them

Rejected because it leaves persisted turns incomplete and makes downstream inspection depend on reconstructing runtime state from code/config.

### Alternative B: store the entire registry object back on the turn

Rejected because it reintroduces non-serializable runtime state into `Turn.Data`, undoing the main architecture cleanup.

### Alternative C: have provider engines switch to persisted turn definitions as the source of truth

Rejected because that changes runtime behavior and authority boundaries in a ticket whose goal is observability, not execution semantics.

### Alternative D: have downstream apps derive schemas from profiles/tool factories

Rejected because it is not historically trustworthy. The code and profile stack may change after a turn is persisted, so post-hoc derivation can lie about what the model actually saw.

### Alternative E: add a separate persistence table for tool schemas

Rejected for the first implementation because the turn is already the natural unit of durable inference state, and a separate table would duplicate storage and indexing concerns without clear benefit.

### Alternative F: persist only tool names / allowlist

Rejected because names and allowlists are not enough to reconstruct the actual advertised schema contract; downstream inspection needs descriptions and parameters too.

## Implementation Plan

### Phase 1. Define the new typed key and persisted type

Files:

- `geppetto/pkg/spec/geppetto_codegen.yaml`
- generated outputs:
  - `geppetto/pkg/turns/keys_gen.go`
  - `geppetto/pkg/inference/engine/turnkeys_gen.go`
  - `geppetto/pkg/doc/types/turns.d.ts`
  - `geppetto/pkg/js/modules/geppetto/consts_gen.go`
  - `geppetto/pkg/doc/types/geppetto.d.ts`

Steps:

1. Add a new data-family key entry for `tool_definitions`.
2. Choose the owning package and concrete type expression.
3. Regenerate key/constant/type outputs.

Recommended naming:

- value key: `ToolDefinitionsValueKey`
- typed key: `KeyToolDefinitions`

### Phase 2. Add the serializable tool-definition type and conversion helpers

Files:

- likely `geppetto/pkg/inference/engine`
- possibly `geppetto/pkg/inference/tools`

Steps:

1. Define the persisted representation if an existing type is not clean enough.
2. Add helper(s) to convert:
   - `tools.ToolDefinition` -> persisted snapshot representation
   - `context.Context` -> runtime-advertised `engine.ToolDefinition`
3. Ensure function/executor fields are never populated from persisted state.

Suggested helpers:

```go
func AdvertisedToolDefinitionsFromContext(ctx context.Context) []engine.ToolDefinition
func persistedToolDefinitions(defs []tools.ToolDefinition) engine.ToolDefinitions
```

### Phase 3. Stamp definitions in the tool loop

Files:

- `geppetto/pkg/inference/toolloop/loop.go`

Steps:

1. After attaching the registry to context and before engine inference, derive definitions from `l.registry.ListTools()`.
2. Write them to `Turn.Data` via `engine.KeyToolDefinitions.Set(&t.Data, defs)`.
3. Keep the existing `KeyToolConfig` write intact.

Acceptance criteria:

- any turn run through the tool loop has both `tool_config` and `tool_definitions` in persisted turn data.

### Phase 4. Keep engines on the live registry and add non-regression coverage

Files:

- `geppetto/pkg/steps/ai/openai/engine_openai.go`
- `geppetto/pkg/steps/ai/openai_responses/engine.go`
- `geppetto/pkg/steps/ai/claude/engine_claude.go`
- `geppetto/pkg/steps/ai/gemini/engine_gemini.go`
- token-count / request-estimation paths that currently inspect the registry

Steps:

1. Audit provider paths and token-count helpers to confirm they still read advertisement data from the live registry.
2. Add targeted tests or assertions showing that introducing `tool_definitions` on `Turn.Data` does not change provider advertisement behavior.
3. If helpful, add comments/docs clarifying that persisted definitions are informational and not consumed by engines in this ticket.

Acceptance criteria:

- provider requests remain built from the live registry,
- persisted `tool_definitions` do not silently become authoritative advertisement input.

### Phase 5. Extend serde and provider tests

Files:

- `geppetto/pkg/turns/serde/serde_test.go`
- provider-specific tests:
  - `geppetto/pkg/steps/ai/openai_responses/token_count_test.go`
  - any OpenAI/Claude/Gemini request-building tests

Steps:

1. Add YAML round-trip coverage for the new `tool_definitions` key.
2. Add tests verifying that `tool_definitions` round-trip independently of the live registry.
3. Keep explicit execution-path tests proving the runtime registry is still required for tool execution and advertisement.

### Phase 6. Update JS codec and docs

Files:

- `geppetto/pkg/js/modules/geppetto/codec.go`
- `geppetto/pkg/doc/topics/07-tools.md`
- `geppetto/pkg/doc/topics/08-turns.md`
- `geppetto/pkg/doc/tutorials/01-streaming-inference-with-tools.md`

Steps:

1. Add the new short-key mapping to the JS codec map.
2. Document the new invariant:
   - `ToolConfig` and `tool_definitions` live on the turn,
   - live registry lives in context,
   - provider advertisement still uses the live registry,
   - execution still requires a live registry.

## Open Questions

- Should the persisted representation reuse `engine.ToolDefinition`, or is it worth introducing a new explicitly serializable type to avoid carrying a `Function` field at all? My recommendation is: use a dedicated serializable type if the churn is acceptable; otherwise use `engine.ToolDefinition` as a pragmatic first implementation and leave unification for a later cleanup.
- Should middleware be allowed to mutate `tool_definitions` directly on the turn, or should the loop remain the canonical writer? My recommendation is: let middleware mutate `ToolConfig` and runtime registry/defs when needed, but keep the tool loop as the canonical initial stamp point.
- Do we want any warning or diagnostic when persisted `tool_definitions` differ from the live registry-derived definitions? My recommendation is no in the first cut unless a downstream inspector explicitly asks for a diff helper.
- Should `responses_server_tools` eventually be folded into a broader persisted advertisement contract, or remain separate? My recommendation is keep them separate until there is a concrete need to unify function and provider-built-in tools.

## References

- `geppetto/pkg/spec/geppetto_codegen.yaml`
- `geppetto/pkg/inference/toolloop/loop.go`
- `geppetto/pkg/inference/tools/context.go`
- `geppetto/pkg/inference/tools/definition.go`
- `geppetto/pkg/inference/engine/types.go`
- `geppetto/pkg/steps/ai/openai/engine_openai.go`
- `geppetto/pkg/steps/ai/openai_responses/engine.go`
- `geppetto/pkg/steps/ai/claude/engine_claude.go`
- `geppetto/pkg/steps/ai/gemini/engine_gemini.go`
- `geppetto/pkg/turns/serde/serde_test.go`
- `geppetto/ttmp/2025/12/18/001-PASS-TOOLS-THROUGH-CONTEXT--pass-tools-tool-registry-through-middleware-context-remove-turn-data-runtime-registry/design-doc/01-design-context-carried-tool-registry-serializable-turn-data.md`
- `geppetto/ttmp/2025/12/18/001-PASS-TOOLS-THROUGH-CONTEXT--pass-tools-tool-registry-through-middleware-context-remove-turn-data-runtime-registry/analysis/01-analysis-passing-tool-registry-through-context.md`
