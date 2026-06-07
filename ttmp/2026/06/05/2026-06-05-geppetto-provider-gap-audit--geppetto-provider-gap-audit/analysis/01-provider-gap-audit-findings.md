---
Title: Provider Gap Audit Findings
Ticket: 2026-06-05-geppetto-provider-gap-audit
Status: active
Topics:
    - geppetto
    - providers
    - reasoning
    - streaming
    - tools
DocType: analysis
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: pkg/events/canonical_events.go
      Note: Canonical provider, text, and reasoning event contracts used as audit baseline.
    - Path: pkg/events/canonical_tool_events.go
      Note: Canonical tool call event contracts used as audit baseline.
    - Path: pkg/inference/engine/inference_result_metadata.go
      Note: |-
        Canonical usage, stop reason, finish class, truncation, and cost mapping helpers.
        Canonical inference_result mapping used as audit baseline
    - Path: pkg/inference/engine/run_with_result.go
      Note: Wrapper that normalizes or synthesizes inference_result metadata.
    - Path: pkg/steps/ai/claude/content-block-merger.go
      Note: |-
        Claude streaming content-block reducer for text, thinking, tools, usage, and stop reasons.
        Claude stream reducer audited for thinking
    - Path: pkg/steps/ai/claude/engine_claude.go
      Note: Claude engine entrypoint and final turn block assembly.
    - Path: pkg/steps/ai/claude/helpers.go
      Note: Claude request builder and turn replay mapper.
    - Path: pkg/steps/ai/claude/token_count.go
      Note: Claude token counter provider implementation.
    - Path: pkg/steps/ai/gemini/engine_gemini.go
      Note: Gemini engine request construction, tool advertisement, usage extraction, and turn replay.
    - Path: pkg/steps/ai/gemini/stream_helpers.go
      Note: Gemini terminal handling, final event emission, and final turn block assembly.
    - Path: pkg/steps/ai/gemini/stream_reducer.go
      Note: |-
        Gemini stream reducer for text, function calls, usage, and finish reason.
        Gemini stream reducer audited for missing reasoning/thought-signature handling
    - Path: pkg/steps/ai/openai/chat_stream.go
      Note: OpenAI-compatible Chat Completions streaming parser.
    - Path: pkg/steps/ai/openai/chat_stream_reducer.go
      Note: OpenAI-compatible Chat Completions event reducer and canonical event mapping.
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: |-
        OpenAI-compatible Chat Completions engine implementation.
        OpenAI Chat-compatible streaming engine audited for tools
    - Path: pkg/steps/ai/openai/helpers.go
      Note: OpenAI-compatible Chat request builder and turn replay mapper.
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: OpenAI Responses engine entrypoint and request lifecycle.
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: OpenAI Responses request builder, reasoning replay, and tool input mapping.
    - Path: pkg/steps/ai/openai_responses/stream_events.go
      Note: |-
        OpenAI Responses provider event mapper for text, reasoning, tools, and usage.
        OpenAI Responses reasoning/tool/usage stream event mapper audited for parity gaps
    - Path: pkg/steps/ai/openai_responses/stream_state.go
      Note: OpenAI Responses terminal metadata and final turn block assembly.
    - Path: pkg/steps/ai/openai_responses/token_count.go
      Note: OpenAI Responses input-token counter provider implementation.
ExternalSources:
    - ttmp/2026/06/05/2026-06-05-geppetto-provider-gap-audit--geppetto-provider-gap-audit/sources/01-gemini-generate-content.md
    - ttmp/2026/06/05/2026-06-05-geppetto-provider-gap-audit--geppetto-provider-gap-audit/sources/02-gemini-function-calling.md
    - ttmp/2026/06/05/2026-06-05-geppetto-provider-gap-audit--geppetto-provider-gap-audit/sources/03-gemini-thinking.md
    - ttmp/2026/06/05/2026-06-05-geppetto-provider-gap-audit--geppetto-provider-gap-audit/sources/04-gemini-thought-signatures.md
    - ttmp/2026/06/05/2026-06-05-geppetto-provider-gap-audit--geppetto-provider-gap-audit/sources/05-openai-reasoning.md
    - ttmp/2026/06/05/2026-06-05-geppetto-provider-gap-audit--geppetto-provider-gap-audit/sources/06-openai-streaming-responses.md
    - ttmp/2026/06/05/2026-06-05-geppetto-provider-gap-audit--geppetto-provider-gap-audit/sources/07-anthropic-extended-thinking.md
    - ttmp/2026/06/05/2026-06-05-geppetto-provider-gap-audit--geppetto-provider-gap-audit/sources/08-anthropic-messages-streaming.md
Summary: Static provider gap audit for Geppetto's OpenAI Chat-compatible, OpenAI Responses, Claude, Gemini, and token-count paths, focused on streaming, tools, reasoning, usage, stop reasons, continuation metadata, and inference_result behavior.
LastUpdated: 2026-06-05T09:15:00-04:00
WhatFor: Use as the working evidence table before adding fixture tests or making provider parity fixes.
WhenToUse: Read before changing provider engines, adding reasoning/tool continuation support, or deciding provider-specific test priorities.
---


# Provider Gap Audit Findings

## Executive Summary

This document is the first static audit pass for Geppetto provider parity. It compares the provider engines against the canonical contracts defined by `turns.Turn`, canonical streaming events, tool-call events, reasoning blocks, provider-call metadata, token usage, stop reasons, and `InferenceResult` metadata.

The strongest implementations are OpenAI Responses and OpenAI-compatible Chat Completions. Claude is now close after the recent streaming and extended-thinking fixes, but its reasoning continuation metadata is still represented with raw payload keys and its replay shape needs a fixture test for tool-use loops. Gemini has the largest confirmed gaps: it supports text streaming, function call detection, usage, finish reasons, and final tool-call blocks, but it currently does not map Gemini thinking, thought signatures, or provider-native function-call IDs into canonical reasoning and continuation data.

## Audit Method

The audit used three evidence types:

1. Canonical Geppetto contracts in `pkg/events`, `pkg/turns`, and `pkg/inference/engine`.
2. Provider implementation code in `pkg/steps/ai/openai`, `pkg/steps/ai/openai_responses`, `pkg/steps/ai/claude`, and `pkg/steps/ai/gemini`.
3. Official provider documentation captured under this ticket's `sources/` directory.

Line-anchored code excerpts were also captured in:

- `ttmp/2026/06/05/2026-06-05-geppetto-provider-gap-audit--geppetto-provider-gap-audit/scripts/artifacts/01-provider-gap-evidence-line-anchors.md`

## Canonical Baseline

The canonical event model already has the right abstractions for this audit:

- provider lifecycle: `EventProviderCallStarted`, `EventProviderCallMetadataUpdated`, `EventProviderCallFinished`;
- text streaming: `EventTextSegmentStarted`, `EventTextDelta`, `EventTextSegmentFinished`;
- reasoning streaming: `EventReasoningSegmentStarted`, `EventReasoningDelta`, `EventReasoningSegmentFinished`;
- tool calls: `EventToolCallStarted`, `EventToolCallArgumentsDelta`, `EventToolCallRequested`.

Evidence:

- `pkg/events/canonical_events.go:73-210`
- `pkg/events/canonical_tool_events.go:1-98`

The `RunInferenceWithResult` wrapper can synthesize or normalize a result if a provider forgets to persist one, but provider engines should still persist accurate `InferenceResult` metadata themselves. This matters because synthesized metadata can infer tool-call pending status, but cannot recover provider-specific usage or cost if those fields were not captured.

Evidence:

- `pkg/inference/engine/run_with_result.go:32-132`
- `pkg/inference/engine/inference_result_metadata.go:10-51`

## Provider Matrix

Legend:

- `supported`: implemented in code and aligned with canonical contracts.
- `partial`: implemented for common cases, but with known limitations.
- `not-implemented`: provider or docs indicate capability, but Geppetto does not map it yet.
- `unsupported-by-provider`: provider-native API does not expose the feature in this mode.
- `unknown-needs-live-smoke`: code suggests support but live behavior should be verified.
- `intentionally-suppressed`: data is intentionally kept out of assistant-visible text.

| Provider | Request text | Stream text | Tools advertised | Tool calls parsed | Reasoning request | Reasoning stream | Reasoning persisted | Usage | Stop / finish | Continuation metadata | `InferenceResult` | Main gaps |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| OpenAI Chat-compatible | supported | supported | supported | supported | partial | partial | supported | partial | supported | partial | supported | Reasoning fields are provider-specific; no encrypted/signed reasoning continuation; named tool choice is not represented in the current canonical tool config. |
| OpenAI Responses | supported | supported | supported | supported | supported | supported | supported | supported | supported | partial | partial | Persisted `InferenceResult.FinishClass` can diverge from Responses-specific `ProviderCallFinished.FinishClass`; reasoning replay is constrained to encrypted content / summaries and adjacency rules. |
| Claude | supported | supported | supported | supported | partial | supported | supported | supported | supported | partial | supported | Reasoning uses raw `signature` payload key; replay shape for thinking + tool use needs a fixture test; Claude tool-choice and newer adaptive/display thinking controls are not mapped. |
| Gemini | supported | supported | supported | partial | not-implemented | not-implemented | not-implemented | partial | supported | not-implemented | supported | No ThinkingConfig / includeThoughts mapping, no thought-signature capture/replay, synthetic tool-call IDs, no reasoning block emission, usage omits thought/cached detail. |
| Token counters | n/a | n/a | partial | n/a | partial | n/a | n/a | supported where implemented | n/a | partial | n/a | Only Claude and OpenAI Responses have token counters; OpenAI Chat-compatible and Gemini token counters are absent. |

## OpenAI Chat-compatible Provider

### Current behavior

OpenAI-compatible Chat Completions is implemented as a streaming-first engine. `RunInference` forces `req.Stream = true`, sets `stream_options.include_usage` unless the model name contains `mistral`, advertises tools from `tools.AdvertisedToolDefinitionsFromContext(ctx)`, maps the canonical `ToolConfig` choices `none|required|auto`, and maps `MaxParallelTools` into `parallel_tool_calls`.

Evidence:

- `pkg/steps/ai/openai/engine_openai.go:72-179`

The stream parser extracts text deltas, non-standard reasoning deltas from `delta.reasoning` or `delta.reasoning_content`, tool call deltas, usage, and finish reason.

Evidence:

- `pkg/steps/ai/openai/chat_stream.go:248-268`
- `pkg/steps/ai/openai/chat_stream.go:321-341`

The reducer emits canonical text, reasoning, usage, and tool events. Tool argument fragments accumulate in `ToolArgBuffers`; a final `EventToolCallRequested` is emitted when the stream reaches EOF. Usage maps prompt/completion/cached tokens to `events.Usage`, and reasoning tokens are stored in event metadata extras rather than `events.Usage`.

Evidence:

- `pkg/steps/ai/openai/chat_stream_reducer.go:127-260`
- `pkg/steps/ai/openai/chat_stream_reducer.go:264-349`
- `pkg/steps/ai/openai/engine_openai.go:412-429`

Final turn blocks are appended as reasoning, assistant text, and then tool-call blocks when the stream ended normally. The engine persists an `InferenceResult` using the canonical metadata builder.

Evidence:

- `pkg/steps/ai/openai/engine_openai.go:330-340`
- `pkg/steps/ai/openai/engine_openai.go:346-370`

The request builder replays canonical reasoning blocks as `reasoning_content` on an assistant message when pending tool calls are flushed. It also maps common inference overrides such as temperature, top-p, max response tokens, stop sequences, seed, reasoning effort, DeepSeek-style `thinking.type`, `n`, and penalties.

Evidence:

- `pkg/steps/ai/openai/helpers.go:135-180`
- `pkg/steps/ai/openai/helpers.go:542-603`

### Assessment

OpenAI Chat-compatible support is good for the proxy use case: text, streaming, tools, reasoning deltas, usage, stop reasons, and final turn blocks all have canonical mappings.

The main caveat is that reasoning support here is not one official OpenAI Chat Completions contract. The code supports the shapes used by OpenAI-compatible providers such as DeepSeek and Together (`reasoning` / `reasoning_content`), and supports `reasoning_effort`, but there is no provider-agnostic encrypted continuation field in this path. This is acceptable for display and single-turn operation, but it should not be treated as equivalent to OpenAI Responses encrypted reasoning or Claude signed thinking.

### Gaps and tests to add

1. Add a fixture test for streamed reasoning + text + usage + tool call in one Chat Completions stream.
2. Add a fixture test that verifies `reasoning_content` is replayed correctly before tool calls.
3. Decide whether canonical tool config needs named-function `tool_choice`; current OpenAI code only maps `none|required|auto`.
4. Document that Chat-compatible reasoning continuation is provider-specific and may not be round-trippable.

## OpenAI Responses Provider

### Current behavior

OpenAI Responses builds a `/v1/responses` request from canonical turn blocks, forces streaming, includes encrypted reasoning content in every request, maps common chat settings, maps reasoning effort/summary/max tokens, applies structured output, and maps per-turn inference overrides.

Evidence:

- `pkg/steps/ai/openai_responses/engine.go:44-161`
- `pkg/steps/ai/openai_responses/helpers.go:180-218`
- `pkg/steps/ai/openai_responses/helpers.go:242-280`

Tool advertisement is implemented through `attachToolsToResponsesRequest`. It converts context-advertised tools, attaches built-in Responses server tools when present, and maps `MaxParallelTools` into `parallel_tool_calls`.

Evidence:

- `pkg/steps/ai/openai_responses/request_tools.go:12-50`

The streaming mapper handles response items for messages, function calls, reasoning, reasoning summaries, web search, function-call argument deltas, usage, and stop reason. Reasoning text and summaries produce canonical `EventReasoningDelta` events. Finished reasoning items append `BlockKindReasoning` blocks carrying text, summary, encrypted content, provider item ID, output index, response ID, and provider status metadata.

Evidence:

- `pkg/steps/ai/openai_responses/stream_events.go:140-170`
- `pkg/steps/ai/openai_responses/stream_events.go:298-359`
- `pkg/steps/ai/openai_responses/stream_events.go:367-431`
- `pkg/steps/ai/openai_responses/stream_events.go:480-523`
- `pkg/steps/ai/openai_responses/stream_events.go:641-669`

Final metadata maps usage, cached tokens, reasoning token counts, thinking text, saying text, optional reasoning summaries, stop reason, and duration. Final turn blocks replay assistant text and tool calls with provider metadata.

Evidence:

- `pkg/steps/ai/openai_responses/stream_state.go:106-135`
- `pkg/steps/ai/openai_responses/stream_state.go:156-186`

The request replay logic deliberately avoids sending plaintext reasoning text back to OpenAI Responses because live Responses requests reject non-empty reasoning input content. It preserves replayable encrypted content and summaries, and only emits reasoning items when the next block shape is a valid assistant message or function-call sequence.

Evidence:

- `pkg/steps/ai/openai_responses/helpers.go:488-518`
- `pkg/steps/ai/openai_responses/helpers.go:522-579`

### Assessment

OpenAI Responses has the most complete reasoning and continuation implementation. It captures the provider's operational continuation metadata and keeps it out of assistant-visible text. That matches the audit guide decision: continuation data is preserved, but not exposed as normal assistant output.

The main gap is terminal classification consistency. `completeResponsesStream` computes a Responses-specific `finishClass` for `EventProviderCallFinished`, including `stream_closed`, `cancelled`, `failed`, and `tool_calls_pending`. But `persistResponsesInferenceResult` uses the generic `BuildInferenceResultFromEventMetadata` helper before that Responses-specific `finishClass` is applied. If the stream closes without `response.completed`, the event can say `stream_closed` while the persisted `InferenceResult` is inferred only from stop reason and tool-call presence.

Evidence:

- `pkg/steps/ai/openai_responses/streaming.go:129-141`
- `pkg/steps/ai/openai_responses/stream_state.go:189-214`

### Gaps and tests to add

1. Add a fixture test where the stream reaches EOF without `response.completed`; assert that turn metadata `InferenceResult.FinishClass` matches the provider-call finish class.
2. Add fixture coverage for reasoning encrypted content replay before a tool call and before an assistant message.
3. Add a regression test for `reasoning_summary_text.delta` and `reasoning_text.done` backfill behavior.
4. Decide whether `encrypted_content` should remain a generic payload key or receive a generated typed key wrapper.

## Claude Provider

### Current behavior

Claude builds a Messages API request from canonical turn blocks, maps system content, user/assistant text, tool calls, tool results, and reasoning blocks. It supports common inference overrides, including max tokens, stop sequences, temperature, top-p, and `ThinkingBudget`, and validates Claude-specific constraints: at most one of temperature/top-p and temperature must be unset or `1.0` when thinking is enabled.

Evidence:

- `pkg/steps/ai/claude/helpers.go:95-145`
- `pkg/steps/ai/claude/helpers.go:177-338`

`RunInference` now forces streaming because the implementation consumes Anthropic SSE. Tools are advertised from `tools.RegistryFrom(ctx)` and converted into Claude tool definitions.

Evidence:

- `pkg/steps/ai/claude/engine_claude.go:95-120`

The content-block merger handles `message_delta` stop reason and usage, `content_block_start` for text/tool/thinking, `content_block_delta` for text, tool JSON input, thinking deltas, and signature deltas, and `content_block_stop` for final text, tool use, and reasoning segment completion.

Evidence:

- `pkg/steps/ai/claude/content-block-merger.go:217-252`
- `pkg/steps/ai/claude/content-block-merger.go:290-393`

Final turn blocks are appended for assistant text, tool calls, and reasoning. Reasoning payloads carry canonical text under `turns.PayloadKeyText` and Claude's signature under the raw payload key `signature`. Usage and stop reason become canonical `InferenceResult` metadata.

Evidence:

- `pkg/steps/ai/claude/engine_claude.go:223-274`

Official Anthropic docs confirm that streaming thinking uses `thinking_delta`, signature data arrives as `signature_delta`, omitted display can send only signatures, and thinking blocks must be preserved during tool-use loops.

Evidence:

- `sources/07-anthropic-extended-thinking.md:164-190`
- `sources/07-anthropic-extended-thinking.md:228-299`
- `sources/07-anthropic-extended-thinking.md:321-337`
- `sources/08-anthropic-messages-streaming.md:132-149`

### Assessment

Claude is now functionally strong for the gaps that triggered this ticket: it forces streaming and maps extended thinking into canonical reasoning events and final reasoning blocks. Usage, stop reasons, text, tool calls, and tool results are supported.

The remaining risk is continuation shape. The replay mapper emits `BlockKindReasoning` as its own assistant message with one `thinking` content block, and emits `BlockKindToolCall` as another assistant message with one `tool_use` content block. Anthropic's documentation emphasizes preserving thinking blocks during tool use, and provider examples typically represent thinking and tool use in the same assistant turn. This may work or may be rejected depending on Anthropic's message validation; it needs a deterministic fixture test and then a live smoke for a thinking + tool continuation loop.

The other design gap is typing. Claude's signature is carried under a raw payload key `signature`, while OpenAI Responses uses `turns.PayloadKeyEncryptedContent`. A generated typed key would make continuation code safer and easier to discover.

### Gaps and tests to add

1. Add a fixture test for a Claude stream containing `thinking` start, `thinking_delta`, `signature_delta`, text, and usage.
2. Add a fixture or request-construction test for replaying `[thinking] + [tool_use] + [tool_result]` without changing turn semantics.
3. Decide whether to introduce `turns.PayloadKeySignature` or a provider-specific typed key for Claude thinking signatures.
4. Decide whether to map newer Anthropic thinking controls such as adaptive thinking and display mode. Current Geppetto maps only `thinking: {type: "enabled", budget_tokens: N}` via `ThinkingBudget`.
5. Decide whether Claude tool choice should be represented in canonical tool config; current code advertises tools but reports no explicit tool-choice support.

## Gemini Provider

### Current behavior

Gemini builds a `genai.GenerativeModel`, applies temperature/top-p/max output tokens from profile settings and per-turn inference overrides, advertises tools by converting context registry entries into `genai.FunctionDeclaration`, and prepends a textual tool signature hint to the prompt.

Evidence:

- `pkg/steps/ai/gemini/engine_gemini.go:188-238`
- `pkg/steps/ai/gemini/engine_gemini.go:242-296`

The stream reducer reads `genai.Text` parts into canonical text events, reads `genai.FunctionCall` parts into canonical tool-call started/requested events, extracts usage metadata, and extracts finish reason.

Evidence:

- `pkg/steps/ai/gemini/stream_reducer.go:35-99`
- `pkg/steps/ai/gemini/engine_gemini.go:433-469`

Finalization closes active text, appends assistant text and tool-call blocks, emits provider-call finished, persists `InferenceResult`, and maps terminal errors to `InferenceFinishClassError`.

Evidence:

- `pkg/steps/ai/gemini/stream_helpers.go:47-94`

The replay mapper converts text-like blocks into `genai.Text`, tool-call blocks into `genai.FunctionCall`, and tool-use blocks into `genai.FunctionResponse`.

Evidence:

- `pkg/steps/ai/gemini/engine_gemini.go:498-565`

Official Gemini docs captured in this ticket indicate that Gemini supports thinking controls, thought summaries, thought signatures, function-call IDs, function responses with IDs, and strict thought-signature preservation during function-calling loops.

Evidence:

- `sources/03-gemini-thinking.md`
- `sources/04-gemini-thought-signatures.md:70-97`
- `sources/04-gemini-thought-signatures.md:473-526`
- `sources/02-gemini-function-calling.md:544-623`
- `sources/02-gemini-function-calling.md:1438-1454`

### Assessment

Gemini is adequate for basic text and simple function-call detection, but it is not yet parity-complete with modern Gemini reasoning and function-calling continuation requirements.

Confirmed gaps:

1. No request-side `ThinkingConfig` mapping. `ThinkingBudget`, `ReasoningEffort`, `ReasoningSummary`, or a Gemini-specific `includeThoughts` option are not mapped into the Gemini request.
2. No response-side thought/reasoning mapping. The reducer only switches on `genai.Text` and `genai.FunctionCall`; it does not inspect thought parts, thought summaries, or thought signatures.
3. No canonical `BlockKindReasoning` output for Gemini.
4. No thought-signature capture or replay. The docs state that thought signatures must be preserved, especially for function-call turns. The current replay code drops any such metadata.
5. Tool-call IDs are synthetic UUIDs. The reducer does not preserve a provider-native function-call ID, and replay does not send a matching ID on `FunctionResponse` where the provider expects one.
6. Usage maps prompt and candidate tokens only. It does not expose thought tokens, cache tokens, or other usage metadata details if present in newer Gemini responses.
7. Tool-call streaming is coarse. The reducer emits `ToolCallStarted` and `ToolCallRequested` in the same chunk and does not emit argument deltas. That may be acceptable if the SDK only exposes completed function call parts, but it is not equivalent to the OpenAI streaming delta contract.

### Gaps and tests to add

1. Add a Gemini fixture/reducer test for a function-call part with provider ID and verify canonical tool-call ID preservation if the SDK exposes it.
2. Add a request-builder test for function responses that verifies `id` is preserved when known.
3. Add a design decision for Gemini thought signatures: likely a provider-specific payload key plus typed helper, because the signature can attach to function calls and text parts.
4. Add support for request-side thinking configuration if the current Go SDK exposes it; otherwise document the SDK version limitation and add a raw REST fallback only if justified.
5. Add response-side handling for thought/thinking parts if exposed by the SDK.
6. Add usage extraction for any available thought/cached token fields.

## Token Counting Providers

### Current behavior

Only Claude and OpenAI Responses have token counter implementations.

Claude builds a Messages count-tokens request from the same projection as the inference request, includes `ThinkingBudget` when present, includes Claude metadata, includes advertised tools, and calls `/v1/messages/count_tokens`.

Evidence:

- `pkg/steps/ai/claude/token_count.go:18-126`

OpenAI Responses builds the same Responses request input as inference, attaches tools, and posts to `/responses/input_tokens`.

Evidence:

- `pkg/steps/ai/openai_responses/token_count.go:34-111`

### Assessment

Token counting is useful where implemented, but it is not provider-complete. There is no OpenAI Chat-compatible token counter and no Gemini token counter. That may be acceptable if token counting is currently an optional subsystem, but it should be explicit in docs and factory behavior.

### Gaps and tests to add

1. Confirm factory behavior when token count is requested for OpenAI Chat-compatible and Gemini profiles.
2. Add OpenAI Chat-compatible token counting only if the target API supports a stable input-token endpoint or if an estimator is acceptable.
3. Add Gemini token counting if the Go SDK exposes `CountTokens` for the current request shape.
4. Add contract tests that token counters use the same turn projection as inference for text, tools, reasoning, and images.

## Cross-Provider Findings

### Finding 1: Streaming-first engines should declare that they override profile stream settings

OpenAI Chat, OpenAI Responses, Claude, and Gemini all effectively run through streaming internally. OpenAI Chat and Claude explicitly force request streaming because their parsers consume streaming responses. OpenAI Responses also forces `stream: true`. Gemini uses `GenerateContentStream`.

This is the right implementation shape for canonical event normalization, but it should be documented as a provider-engine invariant: `chat.stream` in a profile is not necessarily a provider request flag for these engines.

### Finding 2: Reasoning continuation metadata needs a typed-key decision

OpenAI Responses uses `turns.PayloadKeyEncryptedContent` and `turns.PayloadKeySummary`; Claude uses raw `signature`; Gemini needs thought signatures and possibly thought summaries. This should not be allowed to grow as scattered string keys.

Recommended decision:

- Keep assistant-visible reasoning text in `turns.PayloadKeyText`.
- Keep provider continuation material out of `FullText()`.
- Add typed/generated keys for provider continuation fields that affect correctness:
  - Claude signature.
  - Gemini thought signature.
  - OpenAI Responses encrypted content already has a generic key, but may need provider metadata helpers for item ID/output index/status.

### Finding 3: `InferenceResult.FinishClass` should match provider-call terminal semantics

OpenAI Chat, Claude, and Gemini mostly use `BuildInferenceResultFromEventMetadata`, which is adequate when stop reason and tool-call presence fully determine the finish class. OpenAI Responses has richer terminal states (`stream_closed`, `cancelled`, `failed`) but currently persists a generic result before applying the Responses-specific finish class.

Recommended fix:

- Add a provider-specific finish-class override path for OpenAI Responses persisted `InferenceResult`.
- Add tests for normal completion, tool-call pending, provider error, cancellation, and EOF without `response.completed`.

### Finding 4: Tool-call IDs are provider-continuation data, not just local correlation IDs

OpenAI Chat and OpenAI Responses preserve provider call IDs. Claude preserves `tool_use.id`. Gemini synthesizes a UUID for each `genai.FunctionCall` and maps tool results by block ID to name. Official Gemini docs show function-call IDs and function-response IDs. If the SDK exposes these fields, Geppetto should preserve them; if the SDK does not, this should be documented as an SDK limitation.

### Finding 5: Usage metadata is uneven

OpenAI Responses captures input/output/cached and reasoning tokens. OpenAI Chat captures prompt/completion/cached and stores reasoning token counts in metadata extras. Claude captures input/output/cache creation/cache read tokens. Gemini captures prompt/candidates only.

Recommended follow-up:

- Extend `events.Usage` or metadata conventions to represent reasoning/thinking tokens consistently.
- Add provider-specific usage detail extraction tests.

## Prioritized Fix Plan

### Priority 1: Deterministic fixture tests

1. OpenAI Responses terminal finish-class consistency test.
2. Claude thinking + tool-use continuation replay test.
3. Gemini function-call ID preservation test, if the SDK exposes IDs.
4. Gemini thought-signature response/replay test, if the SDK exposes signatures.
5. OpenAI Chat reasoning + tool stream mixed fixture test.

### Priority 2: Low-risk correctness fixes

1. Apply Responses-specific finish class to persisted `InferenceResult`.
2. Add typed key(s) for Claude signatures.
3. Preserve provider-native Gemini function-call IDs if available.
4. Make Gemini tool-call streaming limitations explicit in comments/tests.

### Priority 3: Feature parity work

1. Add Gemini thinking request mapping.
2. Add Gemini thought/thinking response mapping to canonical reasoning events and blocks.
3. Add Gemini thought-signature preservation across tool loops.
4. Add token counters or explicit unsupported behavior for OpenAI Chat-compatible and Gemini.
5. Add provider docs describing streaming-first behavior and reasoning continuation rules.

## Suggested Fixture Test Inventory

| Test | Provider | Expected assertion |
|---|---|---|
| `TestResponsesInferenceResultFinishClassMatchesStreamClosed` | OpenAI Responses | EOF without `response.completed` persists `FinishClass=stream_closed`. |
| `TestResponsesReasoningEncryptedContentReplayBeforeToolCall` | OpenAI Responses | Reasoning item with encrypted content is replayed before the associated function call. |
| `TestClaudeThinkingToolUseReplayPreservesSingleAssistantTurnSemantics` | Claude | Reasoning signature and tool use are replayed in a provider-valid structure. |
| `TestClaudeThinkingStreamWithSignatureDelta` | Claude | `thinking_delta` and `signature_delta` emit canonical reasoning events and a reasoning block with signature metadata. |
| `TestOpenAIChatMixedReasoningTextToolStream` | OpenAI Chat | Reasoning deltas, text deltas, tool arg deltas, usage, finish reason, and final blocks all survive one stream. |
| `TestGeminiFunctionCallPreservesProviderIDWhenAvailable` | Gemini | Provider function-call ID becomes canonical tool-call ID and is reused for function response. |
| `TestGeminiThinkingPartMapsToReasoningBlock` | Gemini | Provider thought/thinking parts emit reasoning events and final `BlockKindReasoning`. |
| `TestTokenCountersUseInferenceProjection` | Token counting | Count-token request includes the same messages/tools/reasoning controls as inference. |

## Open Questions

1. Does the current `google/generative-ai-go/genai` SDK expose Gemini thought parts, thought signatures, and function-call IDs? If not, should Geppetto wait for SDK support or add a REST-level Gemini path?
2. Should `events.Usage` gain `ReasoningTokens` / `ThinkingTokens`, or should these remain provider-specific `metadata.Extra` fields?
3. Should canonical `ToolConfig` support named tool choice, or should named tool choice remain provider-specific?
4. Should Claude adaptive thinking and display mode be represented in `InferenceConfig`, in provider-specific Claude settings, or not at all until needed?
5. Should provider engines expose an explicit capability descriptor for reasoning, continuation metadata, token counting, and streaming-first behavior?

## Validation Status

This document is a static audit artifact. No provider code was changed in this step. The next implementation step should add fixture tests for the highest-confidence gaps before making fixes.
