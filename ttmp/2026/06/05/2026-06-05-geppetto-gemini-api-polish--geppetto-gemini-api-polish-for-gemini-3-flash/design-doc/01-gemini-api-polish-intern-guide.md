---
Title: Gemini API Polish Intern Guide
Ticket: 2026-06-05-geppetto-gemini-api-polish
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
    - Path: go.mod
      Note: Current legacy Gemini SDK dependency
    - Path: pkg/events/canonical_events.go
      Note: Canonical text, reasoning, and provider lifecycle event contract.
    - Path: pkg/events/canonical_tool_events.go
      Note: Canonical tool-call event contract.
    - Path: pkg/inference/engine/inference_config.go
      Note: Canonical per-turn inference overrides used by providers.
    - Path: pkg/steps/ai/gemini/engine_gemini.go
      Note: |-
        Current Gemini provider entrypoint, request builder, tool mapper, and turn replay code.
        Current Gemini provider request construction and legacy SDK use
    - Path: pkg/steps/ai/gemini/engine_gemini_test.go
      Note: |-
        Existing Gemini provider unit tests and fixture patterns.
        Existing Gemini tests and fixture patterns
    - Path: pkg/steps/ai/gemini/stream_helpers.go
      Note: |-
        Current Gemini stream consumption, final block append, and inference_result persistence.
        Current Gemini terminal handling and final block append path
    - Path: pkg/steps/ai/gemini/stream_reducer.go
      Note: |-
        Current Gemini stream reducer for text, function calls, usage, and finish reasons.
        Current Gemini stream reducer missing thought/signature handling
    - Path: pkg/steps/ai/settings/gemini/gemini.yaml
      Note: Current Gemini CLI/profile fields.
    - Path: pkg/steps/ai/settings/gemini/settings.go
      Note: Current Gemini-specific settings registration.
    - Path: pkg/turns/helpers_blocks.go
      Note: Canonical text/tool block constructors.
    - Path: pkg/turns/keys_gen.go
      Note: Existing canonical payload keys relevant to continuation metadata.
    - Path: ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/01-gemini-sdk-capability-probe.sh
      Note: SDK field capability probe supporting the migration recommendation
    - Path: ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/sdk-capability-probe.json
      Note: Probe output showing old SDK lacks modern Gemini 3 fields and new SDK has them
ExternalSources:
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/01-gemini-3-developer-guide.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/02-gemini-thinking.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/03-gemini-thought-signatures.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/04-gemini-function-calling.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/05-generate-content-api.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/06-gemini-api-changelog.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/07-gemini-3-api-updates-blog.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/08-google-genai-go-pkg.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/09-legacy-generative-ai-go-github.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/10-googleapis-go-genai-github.md
Summary: Intern-facing guide for polishing Geppetto's Gemini provider for Gemini 3 Flash / newer Gemini 3 models, including architecture, API differences, design plan, smoke testing, and implementation phases.
LastUpdated: 2026-06-05T09:45:00-04:00
WhatFor: Use before implementing Gemini provider changes so a new engineer understands the current Geppetto provider, the modern Gemini API, and the test/smoke strategy.
WhenToUse: Read when working on Gemini 3 Flash compatibility, thought signatures, thinking configuration, function calling, streaming, or SDK migration in Geppetto.
---



# Gemini API Polish Intern Guide

## Executive Summary

This ticket exists because the current Geppetto Gemini provider is likely behind the current Gemini API surface. The provider currently uses `github.com/google/generative-ai-go/genai` v0.20.1. That library is the older Google Generative AI Go SDK. Current Gemini API documentation and Go package references point to the newer `google.golang.org/genai` SDK, which exposes important fields for Gemini 3 models: `ThinkingConfig`, `Part.Thought`, `Part.ThoughtSignature`, `FunctionCall.ID`, `FunctionResponse.ID`, `GenerateContentResponse.ResponseID`, `UsageMetadata.ThoughtsTokenCount`, and function-call streaming controls.

The immediate goal is not to rewrite the provider blindly. The correct sequence is:

1. Preserve the current behavior with fixture tests.
2. Build direct Geppetto smoke tests for Gemini before routing through `llm-proxy`.
3. Identify whether the old SDK can support Gemini 3 semantics. Early evidence says it cannot fully support them.
4. Design and implement a modern Gemini provider path, likely by migrating to `google.golang.org/genai` or adding a compatibility adapter around it.
5. Validate with live direct Geppetto smokes before any `llm-proxy` smoke.

The most important Gemini 3 compatibility issue is thought-signature preservation. Gemini 3 models can return opaque `thoughtSignature` data on parts, especially function-call parts. When a client sends those parts back in a later request, missing required thought signatures can produce a `400` validation error. Geppetto currently converts Gemini provider output into canonical `Turn` blocks, but it does not preserve Gemini thought signatures or provider-native function-call IDs. This makes Gemini 3 tool loops fragile.

## Problem Statement

Geppetto's Gemini provider currently handles basic text and simple function calls, but modern Gemini 3 models need more complete protocol fidelity.

The current provider:

- creates a legacy `genai.Client` from `github.com/google/generative-ai-go/genai`,
- uses `GenerativeModel.GenerateContentStream`,
- maps `genai.Text` parts to canonical text events,
- maps `genai.FunctionCall` parts to canonical tool-call events,
- synthesizes a UUID for each tool-call ID,
- replays `BlockKindToolCall` as a `genai.FunctionCall`,
- replays `BlockKindToolUse` as a `genai.FunctionResponse`,
- extracts basic prompt and candidate token usage via reflection,
- does not map Gemini thoughts or thought signatures,
- does not map `ThinkingConfig`,
- does not preserve provider response IDs,
- does not capture thoughts token counts.

This is enough for older simple Gemini function calling, but it is not enough for Gemini 3 Flash and related Gemini 3 models. Gemini 3's API adds stricter stateful reasoning behavior. Thought signatures are not optional display metadata; they are continuation data needed for correct multi-turn function-calling and image workflows.

## Current System Map

### Runtime flow

```text
caller / runner / smoke script / llm-proxy
        |
        v
  turns.Turn input
        |
        v
  GeminiEngine.RunInference(ctx, turn)
        |
        +--> build legacy genai.Client
        |
        +--> build legacy genai.GenerativeModel
        |       - temperature
        |       - top_p
        |       - max_output_tokens
        |       - tools as FunctionDeclaration
        |
        +--> build []genai.Part from Turn blocks
        |       - user/system/assistant/reasoning -> genai.Text
        |       - tool_call -> genai.FunctionCall
        |       - tool_use -> genai.FunctionResponse
        |
        +--> GenerateContentStream(ctx, parts...)
        |
        +--> reduceGeminiStreamResponse
        |       - genai.Text -> EventTextDelta
        |       - genai.FunctionCall -> EventToolCallStarted + EventToolCallRequested
        |       - usage -> EventProviderCallMetadataUpdated
        |       - finish reason -> EventProviderCallMetadataUpdated
        |
        +--> completeGeminiStream
        |       - append assistant text block
        |       - append tool_call blocks
        |       - persist InferenceResult
        |       - emit EventProviderCallFinished
        |
        v
  turns.Turn output
```

### Key files

- `pkg/steps/ai/gemini/engine_gemini.go`: request construction, tool registration, turn-to-part mapping, client setup, and `RunInference`.
- `pkg/steps/ai/gemini/stream_reducer.go`: stream chunk reducer.
- `pkg/steps/ai/gemini/stream_helpers.go`: streaming iterator, terminal handling, final turn block assembly.
- `pkg/steps/ai/gemini/engine_gemini_test.go`: current tests for client options, correlation, stream reducer behavior, and terminal errors.
- `pkg/steps/ai/settings/gemini/settings.go` and `gemini.yaml`: Gemini-specific settings registration.
- `pkg/inference/engine/inference_config.go`: canonical inference override fields such as thinking budget and reasoning effort.
- `pkg/events/canonical_events.go` and `pkg/events/canonical_tool_events.go`: target event contract.
- `pkg/turns/keys_gen.go`: canonical payload keys for text, encrypted content, summaries, IDs, and item IDs.

## Important Gemini API Concepts

### GenerateContent and streaming

Gemini's text, multimodal, tool, and thinking workflows are under `generateContent` and streaming `streamGenerateContent`. In the new Go SDK, this is represented by `client.Models.GenerateContent(ctx, model, contents, config)` and `client.Models.GenerateContentStream(ctx, model, contents, config)`.

Important new SDK types:

```go
type GenerateContentConfig struct {
    Temperature    *float32
    TopP           *float32
    MaxOutputTokens int32 or pointer-shaped fields, depending on SDK version
    Tools          []*Tool
    ToolConfig     *ToolConfig
    ThinkingConfig *ThinkingConfig
}

type GenerateContentResponse struct {
    Candidates    []*Candidate
    UsageMetadata *GenerateContentResponseUsageMetadata
    ResponseID    string
}
```

References:

- `sources/05-generate-content-api.md`
- `sources/08-google-genai-go-pkg.md`
- local module evidence: `/home/manuel/go/pkg/mod/google.golang.org/genai@v1.58.0/types.go`

### ThinkingConfig

Gemini thinking models can reason internally before producing visible text. Current docs distinguish older budget-style thinking controls from newer Gemini 3 thinking levels. The new Go SDK exposes `ThinkingConfig` with at least:

```go
type ThinkingConfig struct {
    IncludeThoughts bool
    ThinkingBudget  *int32
}
```

The current Geppetto Gemini provider does not set this because the legacy SDK does not expose the same modern fields. That means Geppetto cannot currently request visible thought parts for smoke testing through the Gemini provider.

References:

- `sources/02-gemini-thinking.md`
- `sources/07-gemini-3-api-updates-blog.md`
- `sources/08-google-genai-go-pkg.md`
- local module evidence: `google.golang.org/genai@v1.58.0/types.go:2484-2490`, `types.go:2702`

### Thought parts and thought signatures

In the new SDK, response parts can carry:

```go
type Part struct {
    Text             string
    FunctionCall     *FunctionCall
    FunctionResponse *FunctionResponse
    Thought          bool
    ThoughtSignature []byte
}
```

`Thought` marks a part as model thinking rather than assistant-visible answer text. `ThoughtSignature` is an opaque continuation token. Gemini 3 documentation says thought signatures must be preserved and sent back in later requests, especially when function calling is involved. Missing required signatures can cause a 400-class validation failure.

Important rule for Geppetto: thought text and signatures must not be merged into normal assistant text. They should map to canonical reasoning blocks and provider-specific continuation metadata.

References:

- `sources/03-gemini-thought-signatures.md`
- `sources/12-cloud-thought-signatures.md`
- `sources/07-gemini-3-api-updates-blog.md`
- local module evidence: `google.golang.org/genai@v1.58.0/types.go:1412-1442`

### Function calls and function responses

The old provider maps function calls by name and arguments only. The new SDK exposes IDs:

```go
type FunctionCall struct {
    Name string
    Args map[string]any
    ID   string
}

type FunctionResponse struct {
    Name     string
    Response map[string]any
    ID       string
}
```

The ID matters because a function response may need to match a function call. The current provider synthesizes UUIDs and therefore loses the provider-native ID if the provider returned one. For Gemini 3, that loss is risky because function-call parts can also carry thought signatures. Geppetto needs a representation that preserves:

- provider function-call ID,
- function name,
- arguments,
- thought signature on the function-call part,
- response ID / output index if exposed,
- association between tool result and original call.

References:

- `sources/04-gemini-function-calling.md`
- `sources/08-google-genai-go-pkg.md`
- local module evidence: `google.golang.org/genai@v1.58.0/types.go:1220-1226`, `types.go:1299-1317`

### Usage metadata

The current provider extracts prompt and candidate token counts. The new SDK exposes richer usage, including `ThoughtsTokenCount` and `ResponseID`.

Geppetto should continue mapping prompt tokens to `events.Usage.InputTokens` and candidate tokens to `events.Usage.OutputTokens`, but it should also preserve thoughts token count in metadata extras until the canonical usage struct grows a first-class field.

References:

- `sources/08-google-genai-go-pkg.md`
- local module evidence: `google.golang.org/genai@v1.58.0/types.go:3297-3317`, `types.go:3378-3397`

## Current Gaps

### Gap 1: Legacy SDK lacks modern Gemini 3 state fields

`go.mod` currently depends on:

```text
github.com/google/generative-ai-go v0.20.1
```

The modern package is:

```text
google.golang.org/genai
```

The new package is already present in the module cache from other projects and exposes the fields needed for Gemini 3: `ThinkingConfig`, `Part.Thought`, `Part.ThoughtSignature`, `FunctionCall.ID`, `FunctionResponse.ID`, `GenerateContentResponse.ResponseID`, and `UsageMetadata.ThoughtsTokenCount`.

The old SDK's public `genai.Part` abstraction is an interface over a smaller set of veneer types. The old public `FunctionCall` has no visible ID or thought signature. The old `partFromProto` path even panics on provider `FunctionResponse` parts in one conversion branch. This makes it a poor base for Gemini 3 parity.

### Gap 2: Geppetto drops thought signatures

Current `reduceGeminiStreamResponse` only switches over:

```go
case genai.Text:
case genai.FunctionCall:
```

There is no `Thought` / `ThoughtSignature` handling. Current `appendGeminiFinalTurnBlocks` appends only assistant text and tool calls. Current `buildPartsFromTurn` treats `BlockKindReasoning` as normal `genai.Text`, which is wrong for provider reasoning continuation.

### Gap 3: Geppetto synthesizes tool-call IDs

Current code uses:

```go
id := uuid.NewString()
```

for every Gemini function call. That makes local correlation work, but it loses provider-native ID semantics. New SDK `FunctionCall.ID` should become the canonical tool-call ID when present.

### Gap 4: Tool configuration is incomplete

The current provider attaches `model.Tools`, but explicit `FunctionCallingConfig` was removed for SDK compatibility. The modern SDK exposes `FunctionCallingConfigModeAuto`, `Any`, `None`, and `Validated`, and `StreamFunctionCallArguments` where supported. The Gemini API docs say streaming function-call arguments are not supported in Gemini API for at least some paths, so this needs careful API-specific testing rather than blind enabling.

### Gap 5: Direct live smokes are missing

The prior provider-gap ticket identified Gemini gaps statically. This ticket must run direct Geppetto smokes before any `llm-proxy` smoke. The direct smokes should exercise the provider through Geppetto's engine/factory/profile path so we observe canonical events and turn blocks without OpenAI-compatible mapping layers hiding Gemini-specific problems.

## Proposed Architecture

### Recommended direction: migrate Gemini provider to the new SDK behind a local adapter

Do not spread `google.golang.org/genai` types across unrelated Geppetto packages. Keep the provider-local implementation in `pkg/steps/ai/gemini` and create a small adapter layer inside that package.

Proposed internal modules:

```text
pkg/steps/ai/gemini/
  engine_gemini.go              existing entrypoint, eventually smaller
  sdk_client_legacy.go          optional old SDK adapter if migration is staged
  sdk_client_genai.go           new SDK adapter
  request_builder.go            Turn + settings -> Gemini request model
  stream_reducer.go             provider stream -> canonical events/state
  continuation.go               thought signatures, provider IDs, replay helpers
  usage.go                      usage/finish/response metadata mapping
  smoke/ or testdata/           fixture JSON/SSE-like raw provider payloads
```

The adapter should hide SDK details behind a provider-local interface:

```go
type geminiClient interface {
    StreamGenerateContent(ctx context.Context, req geminiRequest) (geminiStream, error)
}

type geminiStream interface {
    Next() (*geminiChunk, error)
    Close() error
}

type geminiChunk struct {
    ResponseID string
    Candidates []geminiCandidate
    Usage geminiUsage
    Raw any
}

type geminiPart struct {
    Kind string // text, thought, function_call, function_response, inline_data, unknown
    Text string
    Thought bool
    ThoughtSignature []byte
    FunctionCall *geminiFunctionCall
}

type geminiFunctionCall struct {
    ID string
    Name string
    Args map[string]any
    PartialArgs string
}
```

This allows fixture tests to target Geppetto's reducer without requiring live Google calls or a specific SDK response iterator.

### Canonical mapping rules

#### Text part

Provider text part where `Thought == false`:

- emit `EventTextSegmentStarted` once per text segment,
- emit `EventTextDelta` for streamed text,
- append final `turns.NewAssistantTextBlock(text)`,
- keep thought signatures, if present on non-function-call text parts, as provider metadata on the block rather than visible text.

#### Thought part

Provider text part where `Thought == true`:

- emit `EventReasoningSegmentStarted` with source `thinking` or `gemini-thinking`,
- emit `EventReasoningDelta` with thought text only if the request asked to include thoughts,
- append a `BlockKindReasoning` block,
- put thought text under `turns.PayloadKeyText`,
- put `thought_signature` in a typed/generated provider-specific key if added, or a clearly named payload key if the typed key is deferred,
- never include thought text in `FullText()` / assistant-visible answer text.

#### Function call part

Provider function-call part:

- canonical tool-call ID = provider `FunctionCall.ID` if non-empty; otherwise synthesize a stable ID and record that it was synthesized,
- emit `EventToolCallStarted`,
- if the provider streams partial arguments and the SDK exposes them, emit `EventToolCallArgumentsDelta`,
- emit `EventToolCallRequested` when the provider call is complete,
- append `BlockKindToolCall`,
- preserve thought signature on the tool-call block because Gemini 3 may require the same part to be replayed.

#### Function response replay

When the next request includes a `BlockKindToolUse`, the Gemini request builder must find the matching `BlockKindToolCall` and replay:

- provider function-call ID,
- function name,
- function response ID if required by the SDK/API,
- thought signature on the original function-call part if the function-call part itself is replayed in the current turn,
- result payload as object, not as lossy string when possible.

### Turn replay model

The current `buildPartsFromTurn` flattens all blocks into `[]genai.Part`. That loses role and turn grouping. The new SDK uses `[]*genai.Content`, each with a role and `[]*Part`. Gemini 3 thought-signature validation is sensitive to whether a function call and function response are in the current turn or previous history.

Recommended replay shape:

```go
func buildContentsFromTurn(t *turns.Turn) []*genai.Content {
    var contents []*genai.Content
    var current *genai.Content

    for _, block := range t.Blocks {
        switch block.Kind {
        case BlockKindSystem:
            // Prefer GenerateContentConfig.SystemInstruction if supported.
        case BlockKindUser:
            flushCurrentIfRoleChanges("user")
            current.Parts = append(current.Parts, textPart(block))
        case BlockKindLLMText:
            flushCurrentIfRoleChanges("model")
            current.Parts = append(current.Parts, modelTextPart(block))
        case BlockKindReasoning:
            flushCurrentIfRoleChanges("model")
            current.Parts = append(current.Parts, thoughtPart(block))
        case BlockKindToolCall:
            flushCurrentIfRoleChanges("model")
            current.Parts = append(current.Parts, functionCallPart(block))
        case BlockKindToolUse:
            flushCurrentIfRoleChanges("user")
            current.Parts = append(current.Parts, functionResponsePart(block))
        }
    }

    return contents
}
```

Do not assume this pseudocode is final. It needs fixture and live validation against Gemini 3's rules for current-turn thought signatures. The key idea is to stop treating every block as a flat part with no role boundary.

## Smoke Testing Strategy

The user explicitly requested a lot of smoke testing, with Geppetto itself first before `llm-proxy`. The correct order is below.

### Stage 0: compile-time capability probe

Before changing provider code, create a small ticket script that imports the current old SDK and the proposed new SDK and prints whether fields exist. This should be a compile-time probe, not reflection if possible.

Expected result:

- old SDK does not expose modern thought/signature fields,
- new SDK does expose them.

Store output in:

```text
scripts/artifacts/sdk-capability-probe.json
```

### Stage 1: fixture tests before live provider calls

Add tests in `pkg/steps/ai/gemini` for:

1. text streaming unchanged,
2. function call with provider ID,
3. function call with thought signature,
4. thought text part maps to reasoning block/event,
5. usage with thoughts token count,
6. finish reason mapping for normal stop, max tokens, malformed function call, safety, and errors,
7. replay of function call + function response with required thought signature.

Fixture tests should not need Google credentials.

### Stage 2: direct Geppetto live smokes

Create ticket scripts under:

```text
scripts/01-generate-gemini-smoke-profiles.py
scripts/02-gemini-geppetto-smoke/main.go
scripts/artifacts/
```

The direct smoke should instantiate Geppetto profiles and engines directly. It should not call `llm-proxy`.

Minimum live smoke matrix:

| Case | Model | Purpose | Expected |
|---|---|---|---|
| plain-text | `gemini-3.5-flash` or available Gemini 3 Flash profile | Basic output | assistant text block and usage |
| streaming-text | same | Event stream | text start/delta/finish events |
| tool-call | same | Function calling | tool call block with provider ID if exposed |
| tool-loop | same | Tool result continuation | final answer after tool result |
| thinking-visible | thinking-capable Gemini 3 model | `includeThoughts` / reasoning | reasoning events and reasoning block, no leakage into assistant text |
| thinking-tool-loop | Gemini 3 model | thought signature preservation | no 400 error; signature round-trips |
| max-tokens | same | finish reason | max-token finish class/truncation |
| malformed args or forced tool | same | error handling | provider error captured cleanly |

Each run should save:

- redacted profile used,
- request summary,
- canonical events NDJSON,
- final turn YAML/JSON,
- `InferenceResult`,
- raw provider chunks if debug tap is enabled,
- redacted error response on failure.

### Stage 3: `llm-proxy` smoke only after direct Geppetto passes

Only after direct Geppetto smokes pass should we route through `llm-proxy`. The proxy smoke should validate that OpenAI-compatible Chat Completions tool loops can use a Gemini-backed profile without losing tool-call IDs or continuation metadata. Because OpenAI Chat Completions has no standard field for Gemini thought signatures, the proxy must not expose them as assistant text. It should rely on Geppetto's turn state for continuation rather than trying to round-trip them through an OpenAI client unless a design explicitly supports that.

## Implementation Plan

### Phase 1: Ticket setup and research

Status: in progress.

- Create ticket.
- Download official sources with Defuddle.
- Write this intern guide.
- Upload guide to reMarkable.
- Create smoke-test plan and tasks.

### Phase 2: SDK capability decision

Implement a small capability probe and write a short decision record:

Options:

1. Keep old `github.com/google/generative-ai-go/genai` and add reflection hacks.
2. Migrate provider to `google.golang.org/genai`.
3. Add a raw REST Gemini client just for Gemini 3 features.
4. Support both old and new SDKs temporarily behind an adapter.

Recommended default: migrate to `google.golang.org/genai` behind a provider-local adapter. Avoid reflection hacks for correctness-critical fields such as thought signatures.

Decision criteria:

- Can we preserve thought signatures?
- Can we preserve function-call IDs?
- Can we request visible thoughts for smoke tests?
- Can we capture thoughts token usage?
- Can we keep existing Geppetto profile semantics stable?
- Can we keep tests deterministic?

### Phase 3: Fixture tests and adapter scaffolding

Before live smoke, add failing fixture tests around the reducer and request builder. The fixture tests should define the expected canonical behavior independently of the SDK.

Suggested first tests:

```go
func TestGeminiReducerThoughtPartEmitsReasoningEvents(t *testing.T)
func TestGeminiReducerFunctionCallPreservesProviderIDAndSignature(t *testing.T)
func TestGeminiRequestBuilderReplaysThoughtSignatureForToolLoop(t *testing.T)
func TestGeminiUsageMapsThoughtsTokenCountToMetadataExtra(t *testing.T)
```

### Phase 4: Implement modern request/response mapping

Implement the smallest adapter that supports:

- client creation from existing `InferenceSettings.API` keys/base URLs,
- `GenerateContentConfig` from chat settings and per-turn inference settings,
- tool declarations from `tools.RegistryFrom(ctx)`,
- `FunctionCallingConfig` from canonical `ToolConfig`,
- `ThinkingConfig` from canonical `InferenceConfig` and optional Gemini-specific settings,
- role-preserving content construction from `turns.Turn`,
- stream chunk conversion to provider-local `geminiChunk`,
- canonical event emission,
- final turn block append,
- `InferenceResult` persistence.

### Phase 5: Direct Geppetto live smoke

Run direct smokes through Geppetto. Do not start with `llm-proxy`.

A smoke script should accept:

```text
--profiles <path>
--profile <slug>
--case plain-text|tool-call|tool-loop|thinking|thinking-tool-loop
--out-dir <artifact-dir>
--redact
```

The script should write one summary JSON per run:

```json
{
  "case": "thinking-tool-loop",
  "profile": "gemini-3-flash-thinking-smoke",
  "model": "gemini-3.5-flash",
  "ok": true,
  "event_counts": {
    "EventReasoningDelta": 3,
    "EventToolCallRequested": 1,
    "EventTextDelta": 4
  },
  "final_blocks": ["reasoning", "tool_call", "tool_use", "llm_text"],
  "has_provider_tool_id": true,
  "has_thought_signature": true,
  "inference_result": {
    "finish_class": "completed",
    "stop_reason": "STOP"
  }
}
```

### Phase 6: Proxy smoke

After direct Geppetto passes, test through `llm-proxy` with the same profile slug. Proxy smoke should focus on OpenAI-compatible behavior:

- `/v1/chat/completions` non-streaming text,
- streaming text,
- client-driven tool loop,
- error mapping for provider 400s,
- no reasoning leakage into `message.content`.

## Design Decisions

### Decision 1: Smoke Geppetto directly before `llm-proxy`

Status: accepted.

The provider must be correct before testing compatibility layers. `llm-proxy` maps Geppetto events and turns into OpenAI-compatible responses; it can hide provider bugs or introduce separate mapping bugs. Direct provider smokes isolate Gemini behavior.

### Decision 2: Preserve Gemini thought signatures as continuation metadata

Status: proposed.

Thought signatures are not user-visible text. They should be preserved in turn/block metadata or payload under typed keys and replayed exactly when the provider requires them. They should not appear in assistant text.

### Decision 3: Prefer new SDK migration over reflection hacks

Status: proposed.

The old SDK appears structurally unable to expose the fields needed for Gemini 3 correctness. The new SDK exposes them directly. Reflection should only be used for compatibility around optional fields, not core state such as thought signatures.

### Decision 4: Treat provider function-call IDs as canonical tool-call IDs when present

Status: proposed.

If Gemini returns `FunctionCall.ID`, Geppetto should use it as `turns.PayloadKeyID`. A synthesized ID should be a fallback only, and the block should record that it was synthesized.

### Decision 5: Add Gemini-specific settings only when canonical settings are insufficient

Status: proposed.

Use existing canonical `InferenceConfig` fields where possible:

- `ThinkingBudget` -> Gemini `ThinkingConfig.ThinkingBudget` when supported.
- `ReasoningEffort` or a new field -> Gemini 3 thinking level if needed.
- `MaxResponseTokens`, `Temperature`, `TopP`, `Stop` -> generation config.

Add provider-specific Gemini settings for API-specific controls that do not fit canonical fields, such as `include_thoughts`, thinking level if it cannot map cleanly, or media resolution.

## Risks

1. **SDK migration risk:** `google.golang.org/genai` has a different API and may require refactoring tests and client setup.
2. **Profile compatibility risk:** existing profile YAML should continue to work. Avoid renaming existing keys unless necessary.
3. **Thought signature validation risk:** Gemini 3 validation can depend on exact turn structure, not just whether a signature exists somewhere.
4. **Tool loop role risk:** current flat parts may not model `user` vs `model` turns correctly for Gemini 3.
5. **Credential and quota risk:** live smokes may fail because of unavailable model access, not code bugs. Artifact summaries must distinguish provider access failures from implementation failures.
6. **Proxy mismatch risk:** OpenAI-compatible clients cannot naturally carry Gemini thought signatures. Keep proxy testing separate from provider correctness.

## Review Checklist for an Intern

Before changing code:

- [ ] Read this guide.
- [ ] Read `sources/01-gemini-3-developer-guide.md`.
- [ ] Read `sources/03-gemini-thought-signatures.md`.
- [ ] Read `sources/08-google-genai-go-pkg.md` sections for `Part`, `FunctionCall`, `FunctionResponse`, `ThinkingConfig`, and `GenerateContentResponse`.
- [ ] Read `pkg/steps/ai/gemini/engine_gemini.go`.
- [ ] Read `pkg/steps/ai/gemini/stream_reducer.go`.
- [ ] Read `pkg/steps/ai/gemini/stream_helpers.go`.
- [ ] Read `pkg/steps/ai/gemini/engine_gemini_test.go`.
- [ ] Write or run SDK capability probe.

Before live smoke:

- [ ] Add fixture tests for any planned behavior change.
- [ ] Run `go test ./pkg/steps/ai/gemini -count=1`.
- [ ] Store smoke profiles outside git or redact secrets.
- [ ] Ensure artifacts go under this ticket's `scripts/artifacts/` directory.

Before proxy smoke:

- [ ] Direct Geppetto text smoke passes.
- [ ] Direct Geppetto tool-call smoke passes.
- [ ] Direct Geppetto tool-loop smoke passes.
- [ ] Direct Geppetto thinking/tool-signature smoke passes or is explicitly marked blocked by model access.

## References

### Ticket sources

- `sources/01-gemini-3-developer-guide.md`
- `sources/02-gemini-thinking.md`
- `sources/03-gemini-thought-signatures.md`
- `sources/04-gemini-function-calling.md`
- `sources/05-generate-content-api.md`
- `sources/06-gemini-api-changelog.md`
- `sources/07-gemini-3-api-updates-blog.md`
- `sources/08-google-genai-go-pkg.md`
- `sources/09-legacy-generative-ai-go-github.md`
- `sources/10-googleapis-go-genai-github.md`
- `sources/11-cloud-gemini-3-flash.md`
- `sources/12-cloud-thought-signatures.md`

### Current Geppetto files

- `pkg/steps/ai/gemini/engine_gemini.go`
- `pkg/steps/ai/gemini/stream_reducer.go`
- `pkg/steps/ai/gemini/stream_helpers.go`
- `pkg/steps/ai/gemini/engine_gemini_test.go`
- `pkg/steps/ai/gemini/observability.go`
- `pkg/steps/ai/settings/gemini/settings.go`
- `pkg/steps/ai/settings/gemini/gemini.yaml`
- `pkg/inference/engine/inference_config.go`
- `pkg/inference/tools/advertisement.go`
- `pkg/inference/tools/context.go`
- `pkg/inference/tools/registry.go`
- `pkg/events/canonical_events.go`
- `pkg/events/canonical_tool_events.go`
- `pkg/turns/helpers_blocks.go`
- `pkg/turns/keys_gen.go`

## Initial Task List

1. Add SDK capability probe script and artifact.
2. Add fixture tests for new Gemini part/continuation behavior.
3. Decide SDK migration path.
4. Implement request builder and stream reducer changes.
5. Run direct Geppetto smoke tests.
6. Run `llm-proxy` smoke tests only after direct provider smokes pass.
7. Write final report and upload it to reMarkable.
