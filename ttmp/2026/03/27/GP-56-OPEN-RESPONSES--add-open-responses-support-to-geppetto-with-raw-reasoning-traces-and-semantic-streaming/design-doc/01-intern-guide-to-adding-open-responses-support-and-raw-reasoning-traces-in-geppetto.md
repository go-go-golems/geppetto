---
Title: Intern guide to adding Open Responses support and raw reasoning traces in Geppetto
Ticket: GP-56-OPEN-RESPONSES
Status: active
Topics:
    - geppetto
    - open-responses
    - reasoning
    - streaming
    - events
    - tools
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/chat-events.go
      Note: Current reasoning and partial-thinking event types that the new design must account for
    - Path: geppetto/pkg/inference/engine/factory/factory.go
      Note: Provider selection and current openai-responses wiring
    - Path: geppetto/pkg/inference/tokencount/factory/factory.go
      Note: Token-count factory routing that must grow to cover open-responses
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Tool-loop orchestration that consumes tool_call blocks and appends tool_use results
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Current OpenAI Responses runtime
    - Path: geppetto/pkg/steps/ai/openai_responses/helpers.go
      Note: Current turn-to-responses request mapping and reasoning/function-call adjacency logic
    - Path: geppetto/pkg/steps/ai/openai_responses/helpers_test.go
      Note: Regression coverage for reasoning adjacency and assistant-context preservation
    - Path: geppetto/pkg/steps/ai/openai_responses/token_count.go
      Note: Current Responses token count implementation with OpenAI-specific assumptions
    - Path: geppetto/pkg/turns/keys_gen.go
      Note: Current payload keys show reasoning blocks only persist encrypted content today
    - Path: geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/analysis/01-bug-report-missing-thinking-stream-events.md
      Note: Prior ticket documenting current reasoning_text event support and auto-routing context
    - Path: geppetto/ttmp/2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/design-doc/01-root-cause-analysis-missing-reasoning-items-in-follow-up-tool-calls.md
      Note: Prior ticket documenting why reasoning/function-call adjacency is a hard invariant
ExternalSources:
    - https://huggingface.co/blog/open-responses
    - https://www.openresponses.org/
    - https://www.openresponses.org/specification
    - https://www.openresponses.org/reference
Summary: Detailed architecture, design, and implementation guide for introducing Open Responses support on top of Geppetto's existing OpenAI Responses runtime.
LastUpdated: 2026-03-27T17:06:51.594836654-04:00
WhatFor: ""
WhenToUse: ""
---


# Intern guide to adding Open Responses support and raw reasoning traces in Geppetto

## Executive Summary

This ticket is a design-and-implementation guide for adding Hugging Face / Open Responses support to Geppetto without losing the parts of Geppetto that already work well for OpenAI Responses: turn-based state, tool-loop orchestration, event sinks, and canonical inference-result persistence.

The most important fact for a new contributor is that Geppetto already has a substantial Responses-shaped implementation, but it is currently specialized for OpenAI. The existing engine in `pkg/steps/ai/openai_responses` already knows how to:

- build `/v1/responses` requests from `turns.Turn`,
- stream semantic SSE events,
- emit reasoning-related events such as `reasoning-text-delta` and `partial-thinking`,
- persist encrypted reasoning blocks back into the turn,
- preserve tool-call adjacency required by the Responses family of APIs.

The main gap is not "Geppetto has zero Responses support." The real gap is narrower and more architectural:

1. the current implementation is hard-wired to the OpenAI provider contract and OpenAI configuration keys,
2. Geppetto only persists `encrypted_content` for reasoning blocks, not the richer Open Responses reasoning body (`content`, `summary`, `encrypted_content`),
3. Geppetto’s event model is still centered on OpenAI-flavored `reasoning_text` and summary events rather than a provider-neutral Open Responses semantic model,
4. tool and token-count plumbing are tied to the `openai-responses` provider slug instead of a generic Open Responses abstraction.

The recommended implementation path is to generalize the existing `openai_responses` engine into a provider-neutral Responses layer, then add a first-class `open-responses` provider on top of that generalized layer. This preserves working code, minimizes rework, and gives us a clear place to normalize raw reasoning traces from providers that expose them.

## Problem Statement

Geppetto currently supports two distinct OpenAI-family runtime paths:

- classic Chat Completions through `pkg/steps/ai/openai`,
- OpenAI Responses through `pkg/steps/ai/openai_responses`.

That second path already covers some reasoning functionality, but it is explicitly specialized to OpenAI:

- the provider enum uses `openai-responses`, not `open-responses` ([`geppetto/pkg/steps/ai/types/types.go:5`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/types/types.go#L5)),
- the factory maps only `openai-responses` to the Responses engine ([`geppetto/pkg/inference/engine/factory/factory.go:77`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/inference/engine/factory/factory.go#L77)),
- the engine hardcodes OpenAI URL and key lookup defaults ([`geppetto/pkg/steps/ai/openai_responses/engine.go:108`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/engine.go#L108)),
- the token counter also hardcodes OpenAI-specific `/responses/input_tokens` assumptions ([`geppetto/pkg/steps/ai/openai_responses/token_count.go:67`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/token_count.go#L67)).

Open Responses, as described by Hugging Face and the Open Responses specification, is intentionally broader:

- it is multi-provider, not OpenAI-only,
- it models streaming as semantic events,
- it treats reasoning as a first-class item with optional raw `content`, `summary`, and `encrypted_content`,
- it allows routers and providers to expose additional provider-specific capabilities while sharing a common schema.

That broader contract matters because the user request is specifically about reasoning/thinking delta traces. Geppetto’s current storage model for reasoning is still too thin for Open Responses:

- `turns.BlockKindReasoning` exists ([`geppetto/pkg/turns/block_kind_gen.go:13`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/turns/block_kind_gen.go#L13)),
- but the canonical payload keys only include `encrypted_content` and `item_id` ([`geppetto/pkg/turns/keys_gen.go:8`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/turns/keys_gen.go#L8)),
- and the request builder only round-trips encrypted reasoning blocks back into outgoing input items ([`geppetto/pkg/steps/ai/openai_responses/helpers.go:332`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/helpers.go#L332)).

So the design problem is:

How do we evolve Geppetto from "OpenAI Responses integration with some reasoning support" into "provider-neutral Open Responses support with full raw reasoning trace handling, semantic event normalization, and preserved tool-loop compatibility"?

### Scope

In scope:

- engine/factory/token-count support for a new Open Responses provider mode,
- reasoning item persistence and replay,
- semantic stream-event normalization,
- explicit design guidance for tool loops, tests, and rollout.

Out of scope for this ticket:

- implementing every provider-specific hosted tool,
- redesigning the entire event system,
- changing Geppetto away from `Turn`/`Block` as the core internal state model.

### External Contract Summary

Based on the Hugging Face announcement and the Open Responses docs:

- the format is an open, multi-provider inference standard based on the Responses API,
- reasoning items can expose raw reasoning `content`, `encrypted_content`, and `summary`,
- clients should expect semantic streaming events instead of only text deltas,
- stateless operation remains important, but routers and providers may add provider-specific extensions.

Important nuance:

- The Hugging Face blog shows an example event named `response.reasoning.delta`.
- The Open Responses reference/spec currently models reasoning text as `reasoning_text` content and the stream semantics as `response.<content_type>.delta`.

This looks like a naming mismatch or transitional documentation drift, not something we should ignore. We should deliberately code for canonicalization here instead of assuming one naming shape forever.

## Proposed Solution

Build a generic Responses runtime layer by refactoring the current OpenAI-specific implementation into two parts:

1. a provider-neutral core package that understands the shared Responses/Open Responses turn mapping, item mapping, streaming parser, and reasoning persistence,
2. thin provider adapters that supply base URLs, auth/header behavior, feature flags, and any provider-specific event aliases or tool item names.

### High-Level Design

Keep the Geppetto execution model:

- `Turn` remains the durable conversation unit,
- `Block` remains the atomic state unit,
- engines still mutate a `Turn`,
- `enginebuilder` and `toolloop` remain the orchestration entrypoints,
- events still flow through `events.WithEventSinks(...)`.

Replace the provider specialization point:

- today: `openai_responses.Engine` is both the generic Responses implementation and the OpenAI adapter,
- target: a generic `responses` engine becomes the protocol implementation, and OpenAI/Open Responses become provider configurations on top.

### Proposed Component Layout

Recommended file/module split:

1. `pkg/steps/ai/responses/`
   - shared request/response structs,
   - shared SSE parser,
   - shared turn-to-item conversion,
   - shared item-to-turn conversion,
   - shared tool attachment logic,
   - shared usage extraction.
2. `pkg/steps/ai/responses/provider.go`
   - provider adapter interface and normalization helpers.
3. `pkg/steps/ai/openai_responses/`
   - thin OpenAI-flavored adapter or compatibility wrapper.
4. new `pkg/steps/ai/open_responses/`
   - Hugging Face / Open Responses adapter with version-header behavior and Open Responses-specific defaults.
5. `pkg/inference/engine/factory/`
   - add provider selection for `open-responses`.
6. `pkg/inference/tokencount/factory/`
   - route token counting through the generic Responses token counter where supported.

### Provider Adapter Interface

Pseudocode:

```go
type ResponsesProviderAdapter interface {
    Name() string
    ResolveBaseURL(ss *settings.InferenceSettings) (string, error)
    ResolveAPIKey(ss *settings.InferenceSettings) (string, error)
    ExtraRequestHeaders(ss *settings.InferenceSettings) map[string]string
    NormalizeSSEEventName(eventName string, payload map[string]any) string
    SupportsInputTokenCount() bool
    InputTokenCountPath() string
    SupportsEncryptedReasoning() bool
    SupportsRawReasoningContent() bool
    ProviderToolItemPrefix() string
}
```

This keeps the main engine ignorant of whether the upstream is:

- OpenAI proper,
- Hugging Face Inference Providers exposing Open Responses,
- another router implementing the same schema.

### Turn / Block Data Model Changes

Current state:

- reasoning blocks carry only `encrypted_content`,
- assistant text is stored in `llm_text` blocks,
- function calls and tool results are represented by `tool_call` and `tool_use` blocks,
- request rebuilding relies on ordered block walks for Responses adjacency correctness ([`geppetto/pkg/steps/ai/openai_responses/helpers.go:342`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/helpers.go#L342)).

Proposed change:

- extend reasoning block payloads to store all reasoning channels we care about,
- keep one `BlockKindReasoning`, but enrich its payload instead of inventing several new block kinds.

Recommended new payload keys:

```go
const (
    PayloadKeyReasoningContent          = "reasoning_content"
    PayloadKeyReasoningSummary          = "reasoning_summary"
    PayloadKeyReasoningContentParts     = "reasoning_content_parts"
    PayloadKeyReasoningSummaryParts     = "reasoning_summary_parts"
    PayloadKeyReasoningProviderType     = "reasoning_provider_type"
    PayloadKeyReasoningStreamItemStatus = "reasoning_status"
)
```

Recommended payload semantics:

- `encrypted_content`: encrypted provider trace for stateless continuation,
- `reasoning_content`: flattened raw reasoning text for simple render/debug cases,
- `reasoning_summary`: flattened summary text for UI and logs,
- `reasoning_content_parts`: exact structured content parts from provider,
- `reasoning_summary_parts`: exact summary parts from provider,
- `reasoning_provider_type`: useful when provider emits namespaced items or aliases,
- `reasoning_status`: `in_progress`, `completed`, `incomplete`, etc.

Reason for storing both flattened text and structured parts:

- flattened text keeps existing CLI/UI/debug flows simple,
- structured parts avoid data loss and make future spec changes survivable,
- interns will otherwise be tempted to choose one or the other and create unnecessary migrations later.

### Event Normalization Design

Current event model:

- `partial` for assistant output,
- `partial-thinking` for summary/thinking compatibility,
- `reasoning-text-delta` and `reasoning-text-done` for raw reasoning text,
- generic `info` events for lifecycle markers like `thinking-started` and `reasoning-summary-started`.

This already works for current OpenAI Responses flows, but it does not fully model Open Responses as a semantic, provider-neutral stream.

Recommended design:

1. Keep existing outward-facing events for now because current tools, examples, and tests consume them.
2. Add an internal normalization layer that classifies stream updates before emission.
3. Emit normalized metadata on every reasoning-related event so downstream consumers can distinguish:
   - raw reasoning,
   - reasoning summary,
   - provider-encrypted reasoning,
   - provider-specific extended items.

Suggested normalized metadata fields in `EventMetadata.Extra`:

```json
{
  "responses_item_type": "reasoning",
  "responses_content_type": "reasoning_text",
  "responses_channel": "raw_reasoning",
  "responses_provider": "open-responses",
  "responses_sequence_number": 81
}
```

Suggested parsing strategy:

```go
switch normalizedEventName {
case "response.reasoning_text.delta", "response.reasoning.delta":
    appendRawReasoningDelta(...)
    publish(EventReasoningTextDelta(...))
    publish(EventThinkingPartial(...)) // temporary compatibility path

case "response.reasoning_summary_text.delta":
    appendReasoningSummaryDelta(...)
    publish(EventThinkingPartial(...))

case "response.output_item.added":
    if item.type == "reasoning" { publish thinking-started }

case "response.output_item.done":
    if item.type == "reasoning" {
        finalize reasoning block with content+summary+encrypted_content
        publish thinking-ended
    }
}
```

### Request-Building Design

The most delicate invariant in the current code is the ordered block walk used to preserve reasoning adjacency for function-call chains. That logic should be preserved, not discarded.

Relevant evidence:

- request rebuilding is block-order-sensitive in [`geppetto/pkg/steps/ai/openai_responses/helpers.go:342`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/helpers.go#L342),
- regression coverage exists in [`geppetto/pkg/steps/ai/openai_responses/helpers_test.go:170`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/helpers_test.go#L170),
- a prior bug ticket documents why losing old reasoning blocks breaks follow-up tool calls.

The generic Responses/Open Responses request builder should therefore:

1. keep ordered block traversal,
2. keep the rule that a reasoning block is only replayed when it has a valid immediate follower,
3. extend the reasoning item serializer to include:
   - raw `content` when allowed and available,
   - `summary`,
   - `encrypted_content`,
4. allow provider adapters to strip fields that a provider rejects.

Pseudocode:

```go
func buildInputItemsFromTurnWithAdapter(t *turns.Turn, adapter ResponsesProviderAdapter) []responsesInput {
    for each block in t.Blocks in order {
        switch block.Kind {
        case Reasoning:
            follower := nextMeaningfulFollower(...)
            if !validFollower(follower) {
                continue
            }
            item := serializeReasoningBlock(block)
            item = adapter.NormalizeOutgoingReasoningItem(item)
            append(item)
            appendFollowerChain(follower)

        case ToolCall:
            appendFunctionCall(...)

        case ToolUse:
            appendFunctionCallOutput(...)

        default:
            appendRoleMessage(...)
        }
    }
}
```

### Streaming Parser Design

The current parser in [`geppetto/pkg/steps/ai/openai_responses/engine.go:174`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/engine.go#L174) is a manual SSE loop with a large event-name switch. That is acceptable for the initial implementation, but Open Responses support is the right time to split it into a reusable parser layer.

Recommended split:

1. `sse_reader.go`
   - responsible only for SSE framing.
2. `event_normalizer.go`
   - turns provider event names into canonical names.
3. `stream_aggregator.go`
   - maintains in-progress message text, reasoning text, summary text, tool call args, item ids.
4. `turn_mutator.go`
   - converts finalized items into `Turn` block mutations.

This matters because raw reasoning support is not just one more switch case. It requires state accumulation across:

- output item lifecycle,
- reasoning content parts,
- reasoning summary parts,
- completed item payloads,
- final usage and stop-reason metadata.

### Open Responses-Specific Request Additions

Open Responses-specific behavior to add in the new provider adapter:

- set `OpenResponses-Version` header when configured or when adapter default requires it,
- support provider-routing parameters when the upstream supports them,
- support provider-specific extension passthrough in a typed config object instead of naked `[]any`,
- do not assume OpenAI-only hosted-tool item names like bare `web_search_call`; allow namespaced types such as `openai:web_search_call`.

Recommended turn data key:

```go
KeyResponsesProviderOptions = DataK[map[string]any](GeppettoNamespaceKey, "responses_provider_options", 1)
```

That gives us a place to store temporary provider-specific knobs without polluting the base chat settings.

### Token Counting

Token counting must move with the engine. Right now the token counter is effectively another OpenAI adapter.

Required changes:

1. move request construction reuse into the generic Responses package,
2. let provider adapters choose whether `/responses/input_tokens` exists,
3. let adapters control auth/base URL/header behavior,
4. return `provider=open-responses` when that mode is selected.

### Diagrams

#### Current Runtime

```text
Turn
  -> enginebuilder.runner
     -> toolloop.RunLoop (optional)
        -> openai_responses.Engine.RunInference
           -> buildResponsesRequest
           -> POST /v1/responses
           -> parse SSE
           -> publish Geppetto events
           -> append reasoning/message/tool_call blocks
           -> persist inference_result
```

#### Target Runtime

```text
Turn
  -> enginebuilder.runner
     -> toolloop.RunLoop (optional)
        -> responses.Engine(adapter=open-responses | openai-responses)
           -> build generic Responses/Open Responses request
           -> adapter adds auth/base URL/version headers
           -> parse SSE via reusable normalizer
           -> publish normalized Geppetto events
           -> append rich reasoning/message/tool_call blocks
           -> persist inference_result
```

#### Reasoning Data Flow

```text
provider SSE delta
  -> normalized event name
  -> stream aggregator buffers raw reasoning + summary
  -> Geppetto events emitted for live consumers
  -> output_item.done finalizes one reasoning item
  -> Turn receives one reasoning block
  -> next request rebuilds reasoning item from that block
```

## Design Decisions

### 1. Reuse the existing Responses engine shape instead of building a second engine from scratch

Reason:

- the existing implementation already solved several hard problems,
- it already integrates with `Turn`, `Block`, `events`, `toolloop`, and `inference_result`,
- the prior bugfixes around reasoning adjacency are too easy to regress if we start over.

### 2. Store richer reasoning state on `BlockKindReasoning` instead of introducing multiple new block kinds immediately

Reason:

- `BlockKindReasoning` already exists and is used by the request builder,
- one enriched reasoning block is easier to serialize and debug,
- new block kinds would force much wider downstream changes for little benefit.

### 3. Add a new provider mode instead of pretending Open Responses is just another OpenAI base URL

Reason:

- Open Responses is broader than OpenAI’s Responses API,
- it can require version headers and provider-routing metadata,
- the naming difference matters in factories, flags, and token counting,
- an explicit provider name makes logs and bug reports much easier to read.

### 4. Preserve current event compatibility while introducing internal normalization

Reason:

- Geppetto already has examples and tests that consume `partial-thinking` and `reasoning-text-*`,
- the repo currently depends on those signals,
- the compatibility layer should be viewed as migration scaffolding, not the final semantic model.

### 5. Treat the blog/spec event-name mismatch as a real integration risk

Reason:

- the Hugging Face blog shows `response.reasoning.delta`,
- the reference/spec currently describes reasoning content more like `reasoning_text`,
- routers and providers may adopt different names during standardization.

Implementation implication:

- canonicalize event names in one place,
- test both aliases,
- keep trace logs for real provider captures in ticket artifacts.

### 6. Keep stateless turn rebuilding as the default design center

Reason:

- the current OpenAI Responses engine is explicitly designed around stateless continuation via reasoning items,
- older Geppetto design work already noted that `previous_response_id` creates tension with middleware and turn mutation,
- Open Responses also values stateless, portable request semantics.

This ticket should not re-open the `previous_response_id` path unless a provider forces it later.

## Alternatives Considered

### Alternative A: Add a brand-new `open_responses` engine and leave `openai_responses` untouched

Rejected for now.

Why:

- duplicates request-building logic,
- duplicates SSE parser logic,
- duplicates token-count logic,
- increases the chance that one engine gets reasoning fixes while the other drifts.

### Alternative B: Treat Open Responses as just `openai-responses` with a different base URL

Rejected.

Why:

- hides protocol-level differences behind config hacks,
- makes version headers and provider routing awkward,
- keeps the code semantically OpenAI-centric,
- guarantees confusing future bug reports.

### Alternative C: Persist only raw reasoning text and ignore structured parts

Rejected.

Why:

- loses fidelity if providers emit structured summary parts,
- makes replay less trustworthy,
- forces another migration when richer item structures become necessary.

### Alternative D: Persist only structured parts and never store flattened text

Rejected.

Why:

- makes simple rendering and debugging harder,
- forces every caller to reconstruct text repeatedly,
- increases friction for existing event/UI consumers.

### Alternative E: Remove `partial-thinking` and force all consumers onto new raw reasoning events immediately

Rejected for this rollout.

Why:

- unnecessary breakage for examples and existing UI paths,
- makes the migration larger than the actual provider-support work,
- not required to unlock Open Responses support.

## Implementation Plan

### Phase 1: Create the provider abstraction

Files to touch:

- [`geppetto/pkg/steps/ai/types/types.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/types/types.go)
- [`geppetto/pkg/steps/ai/settings/flags/chat.yaml`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/settings/flags/chat.yaml)
- [`geppetto/pkg/inference/engine/factory/factory.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/inference/engine/factory/factory.go)
- [`geppetto/pkg/inference/tokencount/factory/factory.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/inference/tokencount/factory/factory.go)

Tasks:

1. Add `ApiTypeOpenResponses`.
2. Expose it in CLI/profile settings.
3. Route it in the engine factory.
4. Route token counting through the Responses token-counter path.

Pseudocode:

```go
const ApiTypeOpenResponses ApiType = "open-responses"

switch provider {
case "openai-responses":
    return responses.NewEngine(OpenAIResponsesAdapter{}, settings)
case "open-responses":
    return responses.NewEngine(OpenResponsesAdapter{}, settings)
}
```

### Phase 2: Extract the generic Responses core

Files to create/refactor:

- `pkg/steps/ai/responses/engine.go`
- `pkg/steps/ai/responses/helpers.go`
- `pkg/steps/ai/responses/token_count.go`
- adapter files under `pkg/steps/ai/responses/`

Tasks:

1. Move shared request structs out of `openai_responses`.
2. Move shared turn-input mapping out of `openai_responses`.
3. Move shared SSE framing and aggregation out of `openai_responses`.
4. Leave OpenAI-specific details in an adapter.

Success criterion:

- `openai-responses` behavior remains unchanged after the extraction.

### Phase 3: Extend reasoning block persistence

Files to touch:

- [`geppetto/pkg/turns/keys_gen.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/turns/keys_gen.go)
- [`geppetto/pkg/steps/ai/responses/engine.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/responses/engine.go)
- [`geppetto/pkg/steps/ai/responses/helpers.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/responses/helpers.go)
- [`geppetto/pkg/turns/helpers_print.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/turns/helpers_print.go)
- [`geppetto/pkg/turns/pretty_printer.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/turns/pretty_printer.go)

Tasks:

1. Add payload keys for raw reasoning and summary.
2. Capture raw reasoning and summary in stream buffers.
3. Finalize enriched reasoning blocks on `output_item.done`.
4. Rebuild outgoing reasoning items from the richer payload.

Guardrail:

- keep encrypted reasoning support intact because stateless continuation depends on it.

### Phase 4: Normalize Open Responses stream events

Files to touch:

- `pkg/steps/ai/responses/event_normalizer.go`
- [`geppetto/pkg/events/chat-events.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/events/chat-events.go)
- [`geppetto/pkg/doc/topics/04-events.md`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/doc/topics/04-events.md)

Tasks:

1. Canonicalize `response.reasoning.delta` and `response.reasoning_text.delta`.
2. Emit consistent metadata for reasoning channel/source/type.
3. Keep current events but document the new normalized semantics.
4. Update event docs so they no longer read as OpenAI-only.

### Phase 5: Add Open Responses provider adapter

Files to add:

- `pkg/steps/ai/open_responses/adapter.go`
- possibly `pkg/steps/ai/open_responses/engine.go` if you want a thin wrapper package for clarity.

Tasks:

1. Resolve provider-specific base URL and auth config.
2. Inject `OpenResponses-Version` header when configured.
3. Support provider-specific extensions through typed options.
4. Normalize provider-specific event aliases and namespaced item types.

### Phase 6: Strengthen tests

Files to add/update:

- request-builder tests,
- parser tests,
- engine tests,
- factory tests,
- token-count tests,
- fixture traces under the ticket `sources/` directory or package testdata.

Test matrix:

1. OpenAI Responses raw reasoning event names currently supported by Geppetto.
2. Open Responses blog-style reasoning delta alias.
3. Summary-only reasoning provider.
4. Encrypted-only reasoning provider.
5. Mixed reasoning + tool-call chain replay.
6. Namespaced provider item types such as `openai:web_search_call`.

### Phase 7: Update examples and docs

Files to touch:

- [`geppetto/cmd/examples/advanced/openai-tools/main.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/cmd/examples/advanced/openai-tools/main.go)
- provider docs and settings docs,
- any profile examples for Responses-based providers.

Tasks:

1. Add an example that explicitly demonstrates raw reasoning traces.
2. Add an example that uses the `open-responses` provider mode.
3. Document what is stored on reasoning blocks and why.

## Testing and Validation Strategy

### Unit Tests

Request building:

- verify reasoning blocks serialize all supported fields,
- verify older reasoning blocks are preserved before function calls,
- verify invalid reasoning followers are still omitted safely.

Parser:

- verify alias normalization between `response.reasoning.delta` and `response.reasoning_text.delta`,
- verify summary deltas and raw reasoning deltas are separated correctly,
- verify `response.output_item.done` finalizes stored reasoning payloads.

Factory/settings:

- verify `open-responses` provider selection,
- verify token count factory routing,
- verify missing-provider-config errors are readable.

### Integration Tests

Use mocked HTTP transports as the existing Responses tests do:

- this is already the repo pattern in [`geppetto/pkg/steps/ai/openai_responses/engine_test.go:288`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/engine_test.go#L288),
- extend that style to Open Responses event payloads and headers.

### Trace-Based Verification

For at least one real provider capture:

1. save the raw trace in the ticket `sources/` folder,
2. capture emitted Geppetto events,
3. capture final turn YAML,
4. confirm the next request replays the reasoning item correctly.

### Human Review Checklist

1. Does the final turn contain a reasoning block with raw content, summary, and encrypted content where available?
2. Does the final event metadata distinguish raw reasoning from summary reasoning?
3. Does a follow-up tool-call request preserve required reasoning adjacency?
4. Do existing `partial-thinking` consumers still behave acceptably?

## Risks, Alternatives, and Migration Notes

### Risk: provider event-name drift

Mitigation:

- normalize aliases centrally,
- store real traces,
- test both canonical and alternate names.

### Risk: raw reasoning traces may be sensitive

Mitigation:

- do not print raw reasoning by default in pretty-printers,
- allow redaction/obfuscation hooks,
- keep encrypted content support for providers that require protected reasoning continuity.

### Risk: storing raw reasoning bloats turns

Mitigation:

- store both structured and flattened data only when present,
- consider truncation or debug-mode gates later if storage becomes an issue,
- keep this out of Phase 1 unless data size becomes immediately problematic.

### Risk: Open Responses provider-specific tool items fragment the parser

Mitigation:

- require adapter-level aliasing,
- keep provider-specific item logic in adapters,
- keep the generic engine focused on canonical item families.

### Migration Note

A new intern should understand this clearly:

- Geppetto already has working code in `openai_responses`.
- The goal is to generalize and enrich that code, not replace the entire inference stack.
- The hardest correctness constraint remains turn/block ordering around reasoning and tool calls.
- If that ordering regresses, the API will return provider errors even when everything else looks correct.

## Open Questions

1. Should the canonical event alias be `response.reasoning.delta` or `response.reasoning_text.delta`, or must we support both permanently?
2. Should raw reasoning content ever be replayed to providers when `encrypted_content` is present, or should replay remain encrypted-only by default?
3. Should provider-specific hosted-tool items become first-class Geppetto events immediately, or should they stay in a generic extension lane first?
4. Does every Open Responses provider support an input-token counting endpoint comparable to `/responses/input_tokens`, or do we need per-provider capability checks?
5. Do we want a typed `ResponsesProviderOptions` struct now, or should the first cut use a `map[string]any` plus strict adapter validation?

## References

### Key Geppetto Files

- [`geppetto/pkg/inference/engine/factory/factory.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/inference/engine/factory/factory.go)
- [`geppetto/pkg/inference/tokencount/factory/factory.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/inference/tokencount/factory/factory.go)
- [`geppetto/pkg/inference/toolloop/loop.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/inference/toolloop/loop.go)
- [`geppetto/pkg/inference/toolloop/enginebuilder/builder.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/inference/toolloop/enginebuilder/builder.go)
- [`geppetto/pkg/steps/ai/openai_responses/engine.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/engine.go)
- [`geppetto/pkg/steps/ai/openai_responses/helpers.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/helpers.go)
- [`geppetto/pkg/steps/ai/openai_responses/helpers_test.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/helpers_test.go)
- [`geppetto/pkg/steps/ai/openai_responses/engine_test.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/engine_test.go)
- [`geppetto/pkg/steps/ai/openai_responses/token_count.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai_responses/token_count.go)
- [`geppetto/pkg/steps/ai/types/types.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/types/types.go)
- [`geppetto/pkg/steps/ai/settings/flags/chat.yaml`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/settings/flags/chat.yaml)
- [`geppetto/pkg/events/chat-events.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/events/chat-events.go)
- [`geppetto/pkg/doc/topics/04-events.md`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/doc/topics/04-events.md)
- [`geppetto/pkg/turns/types.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/turns/types.go)
- [`geppetto/pkg/turns/helpers_blocks.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/turns/helpers_blocks.go)
- [`geppetto/pkg/turns/keys_gen.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/turns/keys_gen.go)
- [`geppetto/pkg/turns/block_kind_gen.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/turns/block_kind_gen.go)
- [`geppetto/cmd/examples/advanced/openai-tools/main.go`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/cmd/examples/advanced/openai-tools/main.go)

### Relevant Prior Tickets

- [`geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/analysis/01-bug-report-missing-thinking-stream-events.md`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/analysis/01-bug-report-missing-thinking-stream-events.md)
- [`geppetto/ttmp/2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/design-doc/01-root-cause-analysis-missing-reasoning-items-in-follow-up-tool-calls.md`](/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/ttmp/2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/design-doc/01-root-cause-analysis-missing-reasoning-items-in-follow-up-tool-calls.md)

### External Sources

- https://huggingface.co/blog/open-responses
- https://www.openresponses.org/
- https://www.openresponses.org/specification
- https://www.openresponses.org/reference
