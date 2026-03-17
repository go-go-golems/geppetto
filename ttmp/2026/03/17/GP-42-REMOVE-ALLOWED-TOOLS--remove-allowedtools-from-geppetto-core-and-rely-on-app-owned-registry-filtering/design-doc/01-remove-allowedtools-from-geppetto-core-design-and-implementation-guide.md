---
Title: 'Remove AllowedTools from Geppetto core: design and implementation guide'
Ticket: GP-42-REMOVE-ALLOWED-TOOLS
Status: active
Topics:
    - geppetto
    - architecture
    - tools
    - pinocchio
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: 2026-03-16--gec-rag/internal/webchat/tool_catalog.go
      Note: GEC-RAG already filters registries in app code before calling Geppetto
    - Path: geppetto/pkg/inference/engine/types.go
      Note: Mirrored engine ToolConfig that currently persists allowed_tools into turn metadata
    - Path: geppetto/pkg/inference/tools/base_executor.go
      Note: Executor-time allowlist enforcement through ToolConfig.IsToolAllowed
    - Path: geppetto/pkg/inference/tools/config.go
      Note: Core ToolConfig definition and AllowedTools helpers slated for removal
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: Provider-side tool advertisement filtering using AllowedTools
    - Path: pinocchio/pkg/inference/runtime/composer.go
      Note: Pinocchio keeps an app-level AllowedTools concept distinct from Geppetto core tool config
    - Path: temporal-relationships/internal/extractor/httpapi/run_turns_handlers.go
      Note: Inspector path currently exposes engine tool_config.allowed_tools and will need migration
ExternalSources: []
Summary: Evidence-backed implementation guide for removing AllowedTools from Geppetto core. Explains current duplication between core allowlists and app-side filtered registries, proposes a simpler model, and provides file-by-file implementation guidance for new contributors.
LastUpdated: 2026-03-17T14:20:00-04:00
WhatFor: Use this guide to understand the current AllowedTools system, why it should be removed from Geppetto core, and how to implement the change safely.
WhenToUse: Use when onboarding to GP-42, implementing the removal, reviewing the change, or auditing downstream impact.
---


# Remove AllowedTools from Geppetto core: design and implementation guide

## Executive Summary

This ticket proposes removing `AllowedTools` from Geppetto core and leaving tool allowlisting to application-owned registry filtering. Today Geppetto carries `AllowedTools` through multiple layers:

- `tools.ToolConfig.AllowedTools` in [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go#L5)
- `engine.ToolConfig.AllowedTools` in [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/types.go#L43)
- provider-engine advertisement filtering in [engine_openai.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/openai/engine_openai.go#L447) and [engine_claude.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/claude/engine_claude.go#L287)
- executor-time allow checks in [base_executor.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/base_executor.go#L72)
- tool-loop bridging in [loop.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/loop.go#L175)
- JS builder options in [api_builder_options.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_builder_options.go#L205)

At the same time, the newer application code already filters registries before Geppetto sees them. GEC-RAG builds a filtered registry in [tool_catalog.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/tool_catalog.go#L132) and uses it in [configurable_loop_runner_prepare.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go#L100). Temporal Relationships computes the allowed tool names in [run_chat_transport.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_chat_transport.go#L568). Pinocchio carries an app-level `ComposedRuntime.AllowedTools` in [composer.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/inference/runtime/composer.go#L20).

The duplication means the same concept exists twice:

1. app code decides which tools should be exposed,
2. Geppetto re-enforces an allowlist internally.

The proposed design is to make Geppetto trust the registry it receives. If an app wants only three tools available, it should construct a registry containing only those three tools. Geppetto should then advertise and execute exactly what is present in that registry, without an extra `AllowedTools` layer.

## Problem Statement

### What does `AllowedTools` do today?

`AllowedTools` is a slice of tool names attached to Geppetto tool configuration:

```go
type ToolConfig struct {
    Enabled          bool
    ToolChoice       ToolChoice
    MaxIterations    int
    ExecutionTimeout time.Duration
    MaxParallelTools int
    AllowedTools     []string
    ...
}
```

See [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go#L5).

In Geppetto core, that field has two direct effects:

1. it filters tool definitions before they are advertised to the provider,
2. it blocks tool execution if the model tries to call a tool that is not on the allowlist.

The helper methods that implement that are:

- `IsToolAllowed` in [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go#L100)
- `FilterTools` in [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go#L115)

### Why is that a problem?

Because the applications already do a more concrete and more understandable thing: they build the registry they actually want.

For example, GEC-RAG:

1. has a catalog of possible tools,
2. computes the allowed tool names,
3. constructs a registry containing only those tools,
4. passes that filtered registry into the runner.

That behavior is visible in [tool_catalog.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/tool_catalog.go#L132) and [configurable_loop_runner_prepare.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go#L100).

So the system effectively has two authorization layers:

```text
application layer
  -> filtered registry

Geppetto core
  -> AllowedTools re-filter
  -> AllowedTools execution guard
```

That duplication has real costs:

- two sources of truth,
- more code paths to reason about,
- duplicated tests,
- duplicated docs,
- leaked configuration into turn metadata,
- a more confusing API for interns and app authors.

### Why not keep both for safety?

Because the extra safety is weak compared to the complexity cost.

If app code passes the wrong registry, Geppetto cannot infer the app’s intent anyway. The reliable contract should be:

- the app owns registry construction,
- Geppetto executes what is registered.

That is a cleaner boundary.

## Scope

### In scope

- Removing `AllowedTools` from `tools.ToolConfig`
- Removing `AllowedTools` from `engine.ToolConfig`
- Removing provider advertisement filtering that depends on `AllowedTools`
- Removing executor allow checks that depend on `AllowedTools`
- Removing JS builder exposure of `allowedTools`
- Updating docs/examples/tests that encode Geppetto-owned allowlists

### Out of scope

- App-level runtime metadata like Pinocchio `ComposedRuntime.AllowedTools`
- App-specific registry filtering helpers
- General tool authorization systems beyond registry construction

## Current-State Architecture

### Geppetto tool config layer

The first place `AllowedTools` appears is [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go#L5).

Important behaviors:

- `DefaultToolConfig()` sets `AllowedTools: nil`, where `nil` means “all tools allowed” in [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go#L17)
- `WithAllowedTools(...)` exposes it as a fluent builder in [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go#L60)
- `IsToolAllowed` and `FilterTools` implement the allowlist behavior

### Engine-layer mirrored config

The tool loop persists a second mirrored config type in [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/types.go#L43). That means `AllowedTools` is not only runtime state; it is also serialized turn metadata.

This matters because removing the field affects:

- runtime behavior,
- JSON/YAML shape,
- turn inspectors and docs,
- tests that deserialize tool config from turn data.

### Tool loop bridging

The tool loop converts the loop-level tool config into engine-level tool config in [loop.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/loop.go#L175), including:

```go
AllowedTools: cfg.AllowedTools,
```

This is one of the key bridges that would disappear after the removal.

### Executor-time enforcement

`BaseToolExecutor.IsAllowed(...)` delegates directly to `config.IsToolAllowed(call.Name)` in [base_executor.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/base_executor.go#L72). Later in `ExecuteToolCall`, that check becomes a runtime rejection:

```go
if !b.ToolExecutorExt.IsAllowed(ctx, call) {
    return &ToolResult{..., Error: fmt.Sprintf("tool not allowed: %s", call.Name)}, nil
}
```

See [base_executor.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/base_executor.go#L150).

### Provider-side advertisement filtering

The OpenAI and Claude engines both:

1. convert the provider-agnostic tool definitions into `tools.ToolDefinition`,
2. reconstruct a `tools.ToolConfig`,
3. call `FilterTools(...)`,
4. convert the filtered list to provider format.

That logic lives in:

- [engine_openai.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/openai/engine_openai.go#L447)
- [engine_claude.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/claude/engine_claude.go#L287)

So `AllowedTools` is enforced before the model even sees the tool list.

### JS and example exposure

The JS builder options parser allows `allowedTools` and writes it into `toolCfg.AllowedTools` in [api_builder_options.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_builder_options.go#L233).

Geppetto examples also teach `WithAllowedTools(...)`, for example in [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/generic-tool-calling/main.go#L405).

This is important for migration because if the core field is removed, the examples and JS API must stop teaching it.

## Downstream Usage Analysis

### GEC-RAG

GEC-RAG already embodies the simpler design. Its `ToolCatalog.BuildRegistry(...)` filters tool registrations by name in [tool_catalog.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/tool_catalog.go#L132). The runner preparation path then uses that filtered registry in [configurable_loop_runner_prepare.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go#L100), while still using a plain default Geppetto tool config in [configurable_loop_runner_prepare.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go#L137).

Observed conclusion:

- GEC-RAG does not need Geppetto `ToolConfig.AllowedTools`.
- It already chooses tools by registry construction.

### Pinocchio

Pinocchio’s runtime composer exposes an app-level `AllowedTools` field on `ComposedRuntime` in [composer.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/inference/runtime/composer.go#L20). That field is carried in webchat state in [llm_state.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/llm_state.go#L64).

This is an app-owned concept, not a Geppetto core concept. The intended direction is:

- Pinocchio computes its allowlist,
- Pinocchio builds a filtered registry,
- Geppetto runs that registry without its own allowlist field.

Observed conclusion:

- Pinocchio can retain the runtime-level concept while still removing the core Geppetto field.

### Temporal Relationships

Temporal Relationships computes allowed tool names in [run_chat_transport.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_chat_transport.go#L568) and writes them into its runtime request path in [run_chat_transport.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_chat_transport.go#L223).

Separately, its run-turns inspector path exposes `toolConfig.AllowedTools` from turn data in [run_turns_handlers.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_turns_handlers.go#L285). That is a downstream inspection consequence of Geppetto’s mirrored `engine.ToolConfig.AllowedTools`.

Observed conclusion:

- runtime behavior can stay intact without Geppetto `AllowedTools`,
- the turn-inspector surface will need to adapt because that metadata field would disappear.

## Adjacent surface: `agent_mode_allowed_tools`

There is a separate Geppetto turn-data key:

- [keys_gen.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/turns/keys_gen.go#L35)

Pinocchio middleware writes this hint in [middleware.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/middlewares/agentmode/middleware.go#L177).

Important observation:

- I did not find an active Geppetto runtime path in this workspace that reads this key and turns it into live allowlist enforcement.

So for this ticket, the recommended scope is:

- remove `ToolConfig.AllowedTools` from Geppetto core,
- treat `agent_mode_allowed_tools` as adjacent cleanup or dead metadata review,
- do not let that adjacent key block the main simplification.

## Why removal is the right direction

### Simpler contract

The correct contract should be:

```text
application owns registry filtering
Geppetto executes registered tools
```

Not:

```text
application computes allowlist
application maybe filters registry
Geppetto re-filters provider tool list
Geppetto re-checks execution allowlist
Geppetto persists allowlist into turn tool config
```

### Reduced duplication

Removing `AllowedTools` from Geppetto core deletes:

- one field from `tools.ToolConfig`
- one field from `engine.ToolConfig`
- one builder helper
- one filter helper
- one execution helper
- provider-side filtering branches
- several tests/docs/examples

### Better separation of concerns

Application code knows why a tool should or should not be available:

- product plan,
- profile selection,
- tenant policy,
- feature flag,
- conversation mode,
- session context.

Geppetto does not know that intent. It should be given the final registry and run it.

## Proposed Design

### High-level decision

Geppetto core should stop modeling tool allowlists as part of `ToolConfig`.

Instead:

1. applications compute the allowed tool set,
2. applications construct a filtered registry,
3. Geppetto advertises all tools in that registry,
4. Geppetto executes tool calls only against that registry.

### Proposed API changes

Current:

```go
type ToolConfig struct {
    Enabled          bool
    ToolChoice       ToolChoice
    MaxIterations    int
    ExecutionTimeout time.Duration
    MaxParallelTools int
    AllowedTools     []string
    ...
}
```

Proposed:

```go
type ToolConfig struct {
    Enabled          bool
    ToolChoice       ToolChoice
    MaxIterations    int
    ExecutionTimeout time.Duration
    MaxParallelTools int
    ...
}
```

The same removal applies to `engine.ToolConfig`.

### Runtime flow after the change

```text
app runtime/profile logic
  -> decide allowed tool names
  -> build filtered registry
  -> pass registry to Geppetto

Geppetto
  -> advertise all registered tools
  -> execute only registered tools
```

### Pseudocode for app-side filtering

```go
func BuildFilteredRegistry(all []ToolRegistrar, allowed []string) tools.ToolRegistry {
    reg := tools.NewInMemoryToolRegistry()
    allowedSet := make(map[string]struct{}, len(allowed))
    for _, name := range allowed {
        allowedSet[name] = struct{}{}
    }
    for _, registrar := range all {
        if len(allowedSet) > 0 {
            if _, ok := allowedSet[registrar.Name()]; !ok {
                continue
            }
        }
        _ = registrar.Register(reg)
    }
    return reg
}
```

This is already close to what GEC-RAG does in [tool_catalog.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/tool_catalog.go#L132).

## Design Decisions

### Decision 1: remove Geppetto-owned allowlist enforcement completely

Rationale:

- partial removal would leave ambiguity,
- a single source of truth is the whole point of the cleanup.

### Decision 2: keep app-level allowlist concepts out of this core cleanup

Rationale:

- Pinocchio `ComposedRuntime.AllowedTools` is part of app composition, not Geppetto core enforcement,
- apps can keep whatever metadata helps them build registries.

### Decision 3: treat inspector/turn-metadata fallout as downstream migration work

Rationale:

- removing `engine.ToolConfig.AllowedTools` means some inspection UIs will stop seeing that field,
- that is expected because the source of truth moves to app-owned registry construction.

## Alternatives Considered

### Alternative A: keep `AllowedTools` for provider filtering only

Why rejected:

- still duplicates app-owned filtering,
- still leaks the concept into Geppetto API and docs,
- still creates divergence risk between registry and allowlist.

### Alternative B: keep `AllowedTools` for executor safety only

Why rejected:

- execution-time checks cannot help with tool advertisement mismatch,
- the registry already provides the concrete set of executable tools.

### Alternative C: move allowlist logic to a Geppetto registry helper

This is plausible if Geppetto later wants a reusable registry-filter utility, but it should not remain part of `ToolConfig`.

Why not the main recommendation:

- the user explicitly wants app code to own the filtering,
- a helper can be added later without preserving the core `AllowedTools` field.

## Practical Real-World Examples

### Example 1: GEC-RAG chat

Current good pattern:

- runtime composition computes `runtime.AllowedTools`
- `ToolCatalog.BuildRegistry(runtime.AllowedTools)` builds the concrete registry
- Geppetto runs the registry

After cleanup:

- the same app flow remains,
- Geppetto `ToolConfig` loses a redundant field.

### Example 2: Pinocchio webchat

Desired behavior:

- profile/runtime composer computes app-level allowed tools
- webchat session builder constructs a filtered registry
- Geppetto loop runs that registry with a normal tool config

What is no longer needed:

- a second allowlist inside Geppetto tool config,
- turn metadata that pretends Geppetto owns the allowlist.

### Example 3: Temporal Relationships run-chat

Desired behavior:

- session/tool setup determines which tools should be exposed
- app code builds the right registry
- Geppetto runs it

The only downstream cleanup is inspector/UI adaptation if they currently read `tool_config.allowed_tools`.

## System Diagram

### Current system

```text
app runtime logic
  -> allowed tool names
  -> maybe filtered registry
  -> Geppetto ToolConfig.AllowedTools
         |
         +-> provider tool filtering
         +-> executor allow check
         +-> engine.ToolConfig persisted on turn
```

### Proposed system

```text
app runtime logic
  -> allowed tool names
  -> filtered registry
         |
         v
Geppetto
  -> advertise all tools in registry
  -> execute tools from registry
  -> no separate allowlist field
```

## File-by-File Implementation Guide

This section is written for a new intern. It explains what each file does and why it matters.

### 1. Remove the field from `tools.ToolConfig`

Start with [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go#L5).

What to change:

- remove `AllowedTools`
- remove `WithAllowedTools`
- remove `IsToolAllowed`
- remove `FilterTools`

Why:

- this is the root core definition,
- if the field survives here, the simplification is not real.

### 2. Remove the field from `engine.ToolConfig`

See [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/types.go#L43).

What to change:

- remove `AllowedTools` from the struct
- remove JSON unmarshal support for `allowed_tools`

Why:

- this mirrored type is used for persisted turn metadata,
- leaving it behind would preserve dead configuration shape.

### 3. Remove the loop bridge

See [loop.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/loop.go#L175).

What to change:

- stop copying `cfg.AllowedTools` into `engine.ToolConfig`

Why:

- the engine config should no longer contain that field.

### 4. Remove executor allow checks

See [base_executor.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/base_executor.go#L72).

What to change:

- make the default `IsAllowed` return `true`,
- remove the “tool not allowed” branch from the default allowlist model,
- keep the extension hook so custom executors can still impose app-specific checks if they want.

Important nuance:

`ToolExecutorExt.IsAllowed` is still useful as a customization hook. The ticket only removes the default built-in allowlist based on `ToolConfig.AllowedTools`.

### 5. Remove provider filtering

See:

- [engine_openai.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/openai/engine_openai.go#L447)
- [engine_claude.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/claude/engine_claude.go#L287)

What to change:

- remove temporary reconstruction of `tools.ToolConfig` only for `FilterTools`
- convert all tool definitions from the registry directly to provider format

Conceptual pseudocode:

```go
func prepareProviderTools(toolDefs []engine.ToolDefinition, config engine.ToolConfig) []ProviderTool {
    converted := convertDefinitions(toolDefs)
    return convertAllToProviderFormat(converted)
}
```

### 6. Remove JS builder exposure

See [api_builder_options.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_builder_options.go#L205).

What to change:

- remove parsing of `allowedTools`
- update JS docs/types/examples accordingly

Why:

- the JS API should not expose deleted core configuration.

### 7. Update examples

Search for `WithAllowedTools` and `allowedTools` in Geppetto examples and docs.

What to change:

- replace Geppetto-owned allowlist examples with registry-filter examples,
- or simplify examples so they show the default path without this concept.

### 8. Downstream follow-up adaptation

These are not all Geppetto-core edits, but they matter to the migration story:

- Pinocchio docs that say `AllowedTools` is consumed by the inference loop should be updated.
- Temporal Relationships inspector code that surfaces serialized `tool_config.allowed_tools` should be updated.

The runtime behavior in apps should remain valid if they already build filtered registries.

## Recommended Implementation Phases

### Phase 1: Geppetto core type and runtime cleanup

Files:

- [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go)
- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/types.go)
- [base_executor.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/base_executor.go)
- [loop.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/loop.go)
- provider engines

Goal:

- remove the core field and enforcement logic.

### Phase 2: JS and example cleanup

Files:

- [api_builder_options.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_builder_options.go)
- Geppetto examples and docs found by the inventory script

Goal:

- remove stale surface area and teaching material.

### Phase 3: downstream doc and inspector cleanup

Files:

- Pinocchio and GEC-RAG docs referencing “inference loop uses AllowedTools”
- Temporal Relationships inspector/transport code that serializes or displays the removed field

Goal:

- keep app behavior while removing assumptions that Geppetto persists allowlists in tool config.

## Testing and Validation Strategy

### Core tests

Run:

```bash
go test ./geppetto/pkg/inference/tools/... -count=1
go test ./geppetto/pkg/inference/toolloop/... -count=1
go test ./geppetto/pkg/steps/ai/openai/... -count=1
go test ./geppetto/pkg/steps/ai/claude/... -count=1
go test ./geppetto/pkg/js/modules/geppetto/... -count=1
```

### Test cases to update or add

1. provider engines advertise all tools in the passed registry
2. executor no longer rejects calls based on built-in allowlist config
3. turn tool config no longer serializes `allowed_tools`
4. JS builder options reject or ignore `allowedTools` depending on chosen migration strategy
5. app-level filtered registries still result in the correct runtime tool set

### Manual verification checklist

1. GEC-RAG still exposes only its filtered registry tools.
2. Pinocchio conversation runtime still works when the registry is pre-filtered.
3. Temporal Relationships run-chat still exposes the intended tools, even if turn inspector payload changes.

## Risks and Mitigations

### Risk 1: external callers rely on Geppetto JS `allowedTools`

Mitigation:

- decide whether to hard-remove or briefly deprecate,
- update examples and docs in the same change.

### Risk 2: turn inspectors lose expected metadata

Mitigation:

- document the behavior change,
- if needed, apps can persist their own allowlist metadata separately from Geppetto tool config.

### Risk 3: some code relied on executor-time allow checks as a fallback

Mitigation:

- confirm the registry passed into loops is already filtered in app-owned code,
- keep `ToolExecutorExt.IsAllowed` as a customization hook for special cases.

## Open Questions

1. Should JS `allowedTools` be removed immediately or accepted-and-ignored for one release?
2. Should the adjacent `agent_mode_allowed_tools` turn-data key be cleaned up in the same patch or a follow-up ticket?
3. Do any downstream inspectors need replacement app-owned metadata before `engine.ToolConfig.AllowedTools` disappears?

## Quick-start instructions for a new intern

Follow this order:

1. Read [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go) and [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/types.go) to see the duplicated field.
2. Read [engine_openai.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/openai/engine_openai.go) and [engine_claude.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/claude/engine_claude.go) to see provider filtering.
3. Read [base_executor.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/base_executor.go) to see execution filtering.
4. Read [tool_catalog.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/tool_catalog.go) to see the target app-side pattern.
5. Run the ticket-local inventory script from `scripts/`.
6. Make the core field-removal changes first.
7. Update JS, docs, and examples second.
8. Finish with downstream inspector/doc cleanup.

## References

- [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go)
- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/types.go)
- [base_executor.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/base_executor.go)
- [loop.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/loop.go)
- [engine_openai.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/openai/engine_openai.go)
- [engine_claude.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/claude/engine_claude.go)
- [api_builder_options.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_builder_options.go)
- [tool_catalog.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/tool_catalog.go)
- [configurable_loop_runner_prepare.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/configurable_loop_runner_prepare.go)
- [composer.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/inference/runtime/composer.go)
- [run_chat_transport.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_chat_transport.go)
- [run_turns_handlers.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_turns_handlers.go)
- [keys_gen.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/turns/keys_gen.go)
