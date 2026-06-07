---
Title: Intern Guide to Geppetto Provider Architecture and Gap Audit Plan
Ticket: 2026-06-05-geppetto-provider-gap-audit
Status: active
Topics:
    - geppetto
    - providers
    - reasoning
    - streaming
    - tools
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: pkg/events/canonical_events.go
      Note: Canonical streaming event contract
    - Path: pkg/inference/engine/engine.go
      Note: Canonical engine interface
    - Path: pkg/inference/engine/run_with_result.go
      Note: Canonical inference result wrapper
    - Path: pkg/inference/tools/advertisement.go
      Note: Tool advertisement contract
    - Path: pkg/steps/ai/claude/engine_claude.go
      Note: Claude provider entrypoint
    - Path: pkg/steps/ai/gemini/engine_gemini.go
      Note: Gemini provider entrypoint
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: OpenAI Chat-compatible provider entrypoint
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: OpenAI Responses provider entrypoint
    - Path: pkg/turns/helpers_blocks.go
      Note: Canonical turn block constructors
ExternalSources: []
Summary: Intern-facing technical guide and implementation plan for auditing Geppetto provider parity across streaming, tool calls, reasoning, usage, and continuation metadata.
LastUpdated: 2026-06-05T08:10:00-04:00
WhatFor: Use before starting the provider gap audit so a new engineer understands the Geppetto provider stack, canonical contracts, and audit method.
WhenToUse: Read when auditing or changing Geppetto provider implementations, especially OpenAI Chat, OpenAI Responses, Claude, Gemini, and future providers.
---


# Intern Guide to Geppetto Provider Architecture and Gap Audit Plan

## Executive Summary

Geppetto is an inference orchestration library that normalizes several model provider APIs behind a common `engine.Engine` interface. Provider engines translate between provider-specific protocols and Geppetto's canonical concepts: `turns.Turn`, canonical events, tool registries, inference settings, and `InferenceResult` metadata.

This ticket is for a systematic provider gap audit. The immediate trigger was the `llm-proxy` prototype: OpenAI-compatible Chat Completions and tool-call smoke tests worked for OpenAI Chat-compatible backends and OpenAI Responses, but Anthropic Claude exposed two provider-specific gaps. First, the Claude engine consumed the streaming Messages API but could send `stream: false`, producing a `no response` error. Second, Claude extended thinking returned a documented `thinking` content block and `thinking_delta` / `signature_delta` events that were not parsed into Geppetto's canonical reasoning model.

The audit must not start by changing code. It should first map the provider architecture, define a parity matrix, collect provider-specific API references, and then test each provider against the same canonical expectations. This document is the onboarding and implementation guide for that work.

## Problem Statement

Geppetto supports multiple provider APIs, but each provider has a different shape for:

- request construction,
- streaming event parsing,
- text output,
- tool advertisement and tool calls,
- reasoning/thinking output,
- continuation metadata,
- usage and finish reasons,
- provider-specific constraints.

The current codebase has strong canonical abstractions, but provider implementations have grown at different times. This creates parity risk: one provider may support canonical reasoning blocks, another may silently drop reasoning, and another may crash on a documented provider event.

The purpose of the audit is to answer, for every Geppetto provider engine:

1. What canonical Geppetto features does this provider implement?
2. What provider-native features are intentionally unsupported?
3. What provider-native features are partially supported or silently dropped?
4. What gaps need fixes, tests, docs, or explicit errors?
5. What behavior must downstream systems such as `llm-proxy` rely on?

## System Overview

### Core runtime shape

At the center is a small interface:

```go
type Engine interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

Reference: `pkg/inference/engine/engine.go`.

The engine receives a `turns.Turn`, appends generated blocks, publishes events through sinks in `context.Context`, and returns the updated turn. The helper `RunInferenceWithResult` wraps provider engines and guarantees canonical inference metadata where possible.

Reference: `pkg/inference/engine/run_with_result.go`.

### Text diagram

```text
caller / runner / proxy
        |
        v
  turns.Turn input
        |
        v
  engine.Engine.RunInference(ctx, turn)
        |
        +--> provider request mapper
        |       - messages / input / prompt
        |       - tools
        |       - reasoning settings
        |       - sampling and max tokens
        |
        +--> provider streaming or non-streaming client
        |       - SSE events
        |       - JSON responses
        |       - provider error payloads
        |
        +--> canonical event publication
        |       - EventTextDelta
        |       - EventToolCallStarted
        |       - EventToolCallArgumentsDelta
        |       - EventReasoningDelta
        |       - EventProviderCallFinished
        |
        +--> canonical turn blocks
        |       - BlockKindLLMText
        |       - BlockKindToolCall
        |       - BlockKindToolUse
        |       - BlockKindReasoning
        |
        v
  turns.Turn output + InferenceResult
```

### The canonical data model

A `turns.Turn` is an ordered list of blocks. Provider engines should append blocks in the same semantic order the provider produced them.

Important block kinds:

- `BlockKindUser`: user input.
- `BlockKindSystem`: system or developer directives.
- `BlockKindLLMText`: assistant-visible text.
- `BlockKindToolCall`: model-requested tool invocation.
- `BlockKindToolUse`: tool result supplied by caller/runtime.
- `BlockKindReasoning`: provider reasoning, thinking, or reasoning summary material.

Reference files:

- `pkg/turns/types.go`
- `pkg/turns/block.go`
- `pkg/turns/helpers_blocks.go`
- `pkg/turns/keys_gen.go`

### Canonical events

Provider streaming events should be normalized into canonical Geppetto events. These are what UIs, CLIs, debug sinks, and proxies observe during inference.

Important canonical events:

- `EventProviderCallStarted`
- `EventProviderCallMetadataUpdated`
- `EventProviderCallFinished`
- `EventTextSegmentStarted`
- `EventTextDelta`
- `EventTextSegmentFinished`
- `EventToolCallStarted`
- `EventToolCallArgumentsDelta`
- `EventToolCallRequested`
- `EventReasoningSegmentStarted`
- `EventReasoningDelta`
- `EventReasoningSegmentFinished`
- `EventError`

Reference files:

- `pkg/events/canonical_events.go`
- `pkg/events/canonical_tool_events.go`
- `pkg/events/chat-events.go`
- `pkg/events/context.go`

### Inference settings and profile resolution

Provider configuration is carried by `settings.InferenceSettings`. It contains common chat settings, provider-specific settings, client settings, API keys/base URLs, inference defaults, and model info.

Important fields:

- `InferenceSettings.Chat`: provider type, model/engine, streaming, max response tokens, temperature, top-p, stop, structured output.
- `InferenceSettings.API`: API keys and base URLs keyed by provider.
- `InferenceSettings.Client`: timeout and HTTP client.
- `InferenceSettings.OpenAI`, `Claude`, `Gemini`, `Ollama`: provider-specific settings.
- `InferenceSettings.Inference`: canonical per-turn defaults such as `ThinkingBudget`, `ReasoningEffort`, and `ReasoningSummary`.
- `InferenceSettings.ModelInfo`: model-level capabilities such as reasoning support.

Reference files:

- `pkg/steps/ai/settings/settings-inference.go`
- `pkg/steps/ai/settings/settings-chat.go`
- `pkg/steps/ai/settings/settings-client.go`
- `pkg/steps/ai/settings/model_info.go`
- `pkg/inference/engine/inference_config.go`

### Engine factory

Engine creation is centralized so profile configuration can select the provider implementation.

Reference files:

- `pkg/inference/engine/factory/factory.go`
- `pkg/inference/engine/factory/helpers.go`

Simplified pseudocode:

```go
func CreateEngine(settings *InferenceSettings) (engine.Engine, error) {
    switch settings.Chat.ApiType {
    case "openai":
        return openai.NewEngine(settings)
    case "openai-responses":
        return openai_responses.NewEngine(settings)
    case "claude":
        return claude.NewClaudeEngine(settings)
    case "gemini":
        return gemini.NewEngine(settings)
    default:
        return nil, fmt.Errorf("unsupported api type")
    }
}
```

## Provider Implementations to Audit

### OpenAI Chat-compatible provider

Primary files:

- `pkg/steps/ai/openai/engine_openai.go`
- `pkg/steps/ai/openai/chat_stream.go`
- `pkg/steps/ai/openai/chat_stream_reducer.go`
- `pkg/steps/ai/openai/helpers.go`
- `pkg/steps/ai/openai/chat_types.go`

Current expected responsibilities:

- Build Chat Completions-compatible requests.
- Attach tools from the runtime tool registry.
- Parse streaming chat chunks.
- Normalize text deltas.
- Normalize tool-call deltas.
- Normalize provider-native reasoning if the backend emits reasoning fields.
- Persist final text/tool/reasoning blocks.
- Produce canonical inference metadata.

Known area to inspect:

- Reasoning support is present in `chat_stream_reducer.go`, but provider-specific backends vary in whether they emit compatible reasoning deltas.
- The audit should verify which OpenAI-compatible models/providers actually support `reasoning_effort`, `thinking`, or `reasoning` fields.

### OpenAI Responses provider

Primary files:

- `pkg/steps/ai/openai_responses/engine.go`
- `pkg/steps/ai/openai_responses/helpers.go`
- `pkg/steps/ai/openai_responses/streaming.go`
- `pkg/steps/ai/openai_responses/stream_events.go`
- `pkg/steps/ai/openai_responses/stream_state.go`
- `pkg/steps/ai/openai_responses/request_tools.go`
- `pkg/steps/ai/openai_responses/usage.go`

Current expected responsibilities:

- Build Responses API requests.
- Include `reasoning.encrypted_content` for stateless continuation.
- Map reasoning effort, reasoning summary, and max reasoning tokens.
- Parse text, tool calls, reasoning text, reasoning summaries, encrypted content, and usage.
- Preserve reasoning items around tool-call flows.
- Emit canonical reasoning and text events.

Known area to inspect:

- Responses has the deepest reasoning support, including tests for reasoning summaries, encrypted content, and aliases. The audit should verify that those tests still match current OpenAI docs and live behavior.

### Anthropic Claude provider

Primary files:

- `pkg/steps/ai/claude/engine_claude.go`
- `pkg/steps/ai/claude/helpers.go`
- `pkg/steps/ai/claude/content-block-merger.go`
- `pkg/steps/ai/claude/api/messages.go`
- `pkg/steps/ai/claude/api/streaming.go`
- `pkg/steps/ai/claude/api/content.go`

Current expected responsibilities:

- Build Anthropic Messages API requests.
- Force streaming mode when using `RunInference`, because the implementation consumes SSE events.
- Attach tools from the runtime tool registry.
- Map `thinking_budget` to Anthropic `thinking` request parameters.
- Parse text, tool use, tool input JSON deltas, thinking deltas, signatures, stop reasons, and usage.
- Persist text, tool calls, and reasoning blocks.

Recent issue that motivated this audit:

- Claude streaming with `chat.stream: false` returned `no response` until `RunInference` forced `req.Stream = true`.
- Claude extended thinking returned `content_block.type: "thinking"`, which required explicit parser support.

### Gemini provider

Primary files:

- `pkg/steps/ai/gemini/engine_gemini.go`
- `pkg/steps/ai/gemini/helpers.go`
- `pkg/steps/ai/gemini/stream_reducer.go`
- `pkg/steps/ai/gemini/stream_helpers.go`

Current expected responsibilities:

- Build Gemini requests.
- Handle Gemini tool/function declarations and calls.
- Parse streaming text and tool output.
- Emit canonical events and append canonical turn blocks.

Known area to inspect:

- Determine whether Gemini thinking/reasoning metadata exists in the currently supported API and whether it should map to `BlockKindReasoning`.
- Verify tool-call argument streaming and final tool-call block behavior against canonical OpenAI/Claude/Responses behavior.

### Token counting providers

Primary files:

- `pkg/steps/ai/claude/token_count.go`
- `pkg/steps/ai/openai_responses/token_count.go`
- `pkg/inference/tokencount/factory/factory.go`

Audit scope:

- Verify provider token count APIs align with request builders.
- Confirm tools, reasoning settings, system messages, and multimodal content are included or intentionally omitted.

## Canonical Contracts for the Audit

Each provider should be evaluated against these contracts.

### Contract 1: Request mapping

For every provider, document how canonical turn blocks become provider request objects.

Checklist:

- System blocks.
- User text and multimodal blocks.
- Assistant text blocks.
- Reasoning blocks.
- Tool-call blocks.
- Tool-result blocks.
- Provider-specific metadata needed for continuation.

Pseudocode:

```go
for block in turn.Blocks {
    switch block.Kind {
    case BlockKindSystem:
        mapToProviderSystem(block)
    case BlockKindUser:
        mapToProviderUserMessage(block)
    case BlockKindLLMText:
        mapToProviderAssistantText(block)
    case BlockKindReasoning:
        mapToProviderReasoningIfSupported(block)
    case BlockKindToolCall:
        mapToProviderAssistantToolCall(block)
    case BlockKindToolUse:
        mapToProviderToolResult(block)
    }
}
```

### Contract 2: Streaming event normalization

Streaming provider events should become canonical Geppetto events without provider leakage unless the canonical event explicitly carries provider metadata.

Checklist:

- Text starts, deltas, and finishes.
- Tool-call starts, argument deltas, and final requested events.
- Reasoning starts, deltas, and finishes.
- Provider metadata updates.
- Final provider call finished.
- Provider error events.

### Contract 3: Turn persistence

After `RunInference` returns, the output turn should contain durable blocks for the generated result.

Expected block order:

```text
[input blocks...]
[reasoning block(s), if provider returned reasoning]
[assistant text block(s), if provider returned text]
[tool call block(s), if provider requested tools]
```

Provider-specific continuation data may be stored in block payload or metadata. The audit must list each such field and determine whether it is stable enough for downstream use.

### Contract 4: Tool calling

Tool support has two independent surfaces:

1. Tool advertisement to the provider.
2. Tool-call output from the provider.

Reference files:

- `pkg/inference/tools/registry.go`
- `pkg/inference/tools/context.go`
- `pkg/inference/tools/advertisement.go`
- `pkg/inference/tools/adapters.go`
- `pkg/inference/tools/config.go`

Important invariant:

- Live runtime tool definitions should come from the context registry, not only a persisted `Turn.Data` snapshot.

Pseudocode:

```go
registry, ok := tools.RegistryFrom(ctx)
if ok {
    providerRequest.Tools = convert(registry.ListTools())
}
```

### Contract 5: Reasoning and thinking

Reasoning must be explicitly categorized by provider.

Provider examples:

- OpenAI Responses:
  - raw reasoning tokens are not exposed,
  - summaries can be requested,
  - encrypted reasoning items can be included for stateless continuation.
- OpenAI Chat-compatible:
  - some backends expose reasoning deltas or fields, but this is provider/model-specific.
- Claude:
  - extended thinking streams `thinking` blocks,
  - `thinking_delta` carries thinking text when display allows it,
  - `signature_delta` carries verification/continuation material,
  - thinking blocks may need preservation in tool-use flows.

Audit checklist:

- Does the provider support reasoning request controls?
- Does it emit reasoning events?
- Does Geppetto parse them?
- Does Geppetto persist them?
- Does Geppetto avoid leaking reasoning through normal assistant text?
- Does Geppetto preserve continuation material?

### Contract 6: Usage and finish metadata

Every provider should populate `InferenceResult` as consistently as possible.

Reference files:

- `pkg/inference/engine/inference_result_metadata.go`
- `pkg/inference/engine/run_with_result.go`
- `pkg/turns/inference_result.go`

Audit fields:

- provider,
- model,
- stop reason,
- finish class,
- usage tokens,
- max tokens,
- truncation,
- duration,
- provider-specific extras.

## Audit Matrix Template

Use this table for each provider.

| Provider | Request text | Stream text | Tools advertised | Tool calls parsed | Reasoning request | Reasoning stream | Reasoning persisted | Usage | Continuation metadata | Gaps |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| OpenAI Chat | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |
| OpenAI Responses | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |
| Claude | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |
| Gemini | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |

For each `TBD`, fill one of:

- `supported`,
- `partial`,
- `unsupported-by-provider`,
- `not-implemented`,
- `unknown-needs-live-smoke`,
- `intentionally-suppressed`.

## Recommended Investigation Workflow

### Phase 1: Source and code map

1. Read provider docs and store sources under `sources/`.
2. Read the provider engine files listed above.
3. Read the associated tests before changing code.
4. Fill the audit matrix with current evidence.

Commands:

```bash
rg -n "Reasoning|thinking|tool|ToolCall|EventReasoning|EventTool" pkg/steps/ai pkg/inference pkg/events pkg/turns -S
rg -n "RunInference|RunInferenceWithResult|PublishEvent|PersistInferenceResult" pkg/steps/ai -S
```

### Phase 2: Static gap analysis

For each provider, inspect four functions or modules:

1. Request builder.
2. Streaming parser/reducer.
3. Final turn block append logic.
4. Inference result / usage mapping.

Write findings with file and line references. Do not infer support just because a canonical type exists; find the provider-specific mapping.

### Phase 3: Fixture tests

For every gap, prefer a fake provider fixture before a live test.

Example fixture strategy:

```go
server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    assertRequestShape(r)
    w.Header().Set("Content-Type", "text/event-stream")
    writeProviderSSE(w, []string{
        providerStartEvent,
        providerReasoningDelta,
        providerTextDelta,
        providerDone,
    })
}))
```

### Phase 4: Live smoke tests

Use live smokes only after fixture tests establish expected behavior. Store scripts under `scripts/` and response artifacts under `scripts/artifacts/`.

Minimum live smoke cases:

- plain text,
- streaming text,
- tool call,
- streaming tool call,
- reasoning/thinking if provider supports it,
- tool + reasoning continuation if provider supports it.

### Phase 5: Fixes and docs

For every code fix:

- add provider fixture tests,
- run targeted tests/lint,
- run live smoke if credentials and model access exist,
- update this ticket's diary,
- update provider docs or comments if behavior is subtle.

## Known Starting Findings

These are not final audit conclusions. They are known context from the `llm-proxy` work that this ticket should verify and expand.

1. Claude `RunInference` now forces streaming requests because the implementation consumes SSE.
2. Claude extended thinking required explicit parsing of `thinking`, `thinking_delta`, and `signature_delta`.
3. OpenAI Responses has mature reasoning support, including summaries and encrypted content.
4. OpenAI Chat-compatible reasoning support exists in the reducer path, but live backend behavior is model/provider-specific.
5. Gemini needs a dedicated pass for reasoning/thinking parity and tool-call stream parity.
6. Full Geppetto repository validation is currently blocked by an unrelated missing `github.com/go-go-golems/go-go-goja/engine` dependency; targeted provider package validation may be necessary until that is fixed.

## Implementation Plan

### Step 1: Build the provider evidence table

Output: one analysis doc under this ticket.

- Add source documents for OpenAI, Anthropic, Gemini, and any other provider under `sources/`.
- Fill the audit matrix.
- Link each matrix cell to code and/or docs.

### Step 2: Add fixture tests for confirmed gaps

Output: failing tests first, then fixes.

Priority order:

1. Crashes on documented provider events.
2. Reasoning leakage into assistant text.
3. Lost tool-call arguments.
4. Missing continuation metadata.
5. Incorrect usage/finish reason.

### Step 3: Normalize provider continuation metadata

Output: design decision record.

Decide whether provider-specific fields such as Claude `signature` and OpenAI `encrypted_content` should remain raw payload keys or get typed key helpers.

### Step 4: Live smoke scripts

Output: scripts and artifacts.

All live smokes must be reproducible without committing secrets. Scripts should read local profile config or environment variables and write redacted artifacts.

### Step 5: Final report

Output: final provider gap report uploaded to reMarkable.

The report should include:

- provider matrix,
- verified behavior,
- gaps fixed,
- gaps remaining,
- suggested follow-up tickets,
- validation commands,
- artifact index.

## Design Decisions

### Decision 1: Audit against canonical contracts, not provider similarity

Providers are not expected to expose identical native APIs. The audit should measure each provider against Geppetto's canonical contracts and mark provider-native limitations explicitly.

Status: accepted.

### Decision 2: Preserve provider-specific continuation data, but do not expose it as assistant text

Reasoning continuation material is operational metadata. It may be required for provider quality or correctness, but it should not leak through assistant-visible text APIs.

Status: accepted.

### Decision 3: Prefer fixture tests before live smokes

Live smokes are useful, but they are expensive and can change with provider behavior. Fixture tests make regressions deterministic.

Status: accepted.

### Decision 4: Store every script and artifact in the ticket

Ad-hoc `/tmp` scripts are not enough for a provider audit. Every script and non-secret artifact should be copied into this ticket.

Status: accepted.

## Open Questions

1. Should Geppetto add generated typed keys for provider-specific reasoning continuation fields such as Claude `signature`?
2. Should Chat Completions compatibility layers expose reasoning through non-standard fields, or always suppress it?
3. Should `RunInference` implementations be allowed to override profile `stream` settings when their implementation requires streaming?
4. Which providers should support tool + reasoning continuation in the same assistant turn?
5. What is the expected behavior for providers that emit reasoning summaries but not raw reasoning text?

## References

Primary code references:

- `pkg/inference/engine/engine.go`
- `pkg/inference/engine/run_with_result.go`
- `pkg/inference/engine/inference_config.go`
- `pkg/events/canonical_events.go`
- `pkg/events/canonical_tool_events.go`
- `pkg/turns/helpers_blocks.go`
- `pkg/steps/ai/openai/engine_openai.go`
- `pkg/steps/ai/openai/chat_stream_reducer.go`
- `pkg/steps/ai/openai_responses/engine.go`
- `pkg/steps/ai/openai_responses/stream_events.go`
- `pkg/steps/ai/claude/engine_claude.go`
- `pkg/steps/ai/claude/content-block-merger.go`
- `pkg/steps/ai/gemini/engine_gemini.go`
- `pkg/inference/tools/advertisement.go`
- `pkg/inference/tools/context.go`
- `pkg/inference/tools/registry.go`

Related recent commits:

- `fb2b9ed402ab680beac78b77ffd398e7b6292b66` — force Claude streaming requests.
- `6928c321a3e30c1b402c71d94366d2101e7e514e` — support Claude extended thinking streams.

Related `llm-proxy` ticket artifacts:

- `/home/manuel/workspaces/2026-06-04/llm-proxy/llm-proxy/ttmp/2026/06/04/2026-06-04-llm-proxy-openai-compatible-geppetto-proxy--openai-compatible-llm-proxy-backed-by-geppetto/sources/01-anthropic-extended-thinking.md`
- `/home/manuel/workspaces/2026-06-04/llm-proxy/llm-proxy/ttmp/2026/06/04/2026-06-04-llm-proxy-openai-compatible-geppetto-proxy--openai-compatible-llm-proxy-backed-by-geppetto/sources/04-openai-dev-reasoning-guide.md`
- `/home/manuel/workspaces/2026-06-04/llm-proxy/llm-proxy/ttmp/2026/06/04/2026-06-04-llm-proxy-openai-compatible-geppetto-proxy--openai-compatible-llm-proxy-backed-by-geppetto/scripts/artifacts/llm-proxy-thinking-smoke-summary-after-claude-thinking-fix.json`
