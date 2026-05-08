---
Title: Analysis and Implementation Guide
Ticket: GEPPETTO-CORRELATION-KEYS
Status: active
Topics:
  - observability
  - reasoning
  - openai-compatibility
  - streaming
  - pinocchio
  - webchat
DocType: design-doc
Intent: long-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Design and implementation plan for normalized provider correlation keys across Geppetto, Pinocchio, web-chat, and CoinVault.
LastUpdated: 2026-05-07T20:45:00-04:00
WhatFor: Explain why Chat Completions streams need fallback correlation keys and how to propagate them without faking provider item IDs.
WhenToUse: Read before implementing or reviewing provider correlation key propagation.
---

# Analysis and Implementation Guide

## Executive summary

OpenAI Responses streams expose provider-native output item identity. OpenAI-compatible Chat Completions streams usually do not. The absence of item IDs makes existing provider-to-browser reasoning joins sparse for providers such as DeepSeek V4 Pro, even though the thinking deltas are present and rendered correctly.

The fix is not to invent `item_id`. `item_id` should remain provider-native. Instead, Geppetto should emit a normalized correlation identity alongside provider-native IDs. Downstream applications can use that identity when native item IDs are absent.

The implementation introduces fields such as:

```text
choice_index
stream_kind
correlation_key
tool_call_id
tool_call_index
```

Geppetto is the right layer to compute these fields because Geppetto sees provider stream objects before normalization. Pinocchio should forward them into protobuf UI payloads, web-chat should store/export them, and CoinVault should consume the updated generated frontend protobuf/debug payload shape.

## Problem statement

The DeepSeek V4 Pro CoinVault full-trace artifact showed this pattern:

- provider JSON objects had response IDs;
- provider JSON objects had tool-call IDs;
- provider JSON objects had no provider item IDs;
- browser reasoning updates rendered correctly;
- `geppetto_reasoning_to_frontend` was empty because it is built around provider item identity.

In OpenAI Responses, `item_id` refers to an output item inside a response. In Chat Completions, the stream is flatter:

```text
response id -> choices[index] -> delta.content / delta.reasoning_content / delta.tool_calls
```

There is no native output item object. Using `item_id` for a synthetic value would make traces lie. But not providing any stable fallback identity makes SQL analysis harder.

## Proposed solution

Add explicit normalized correlation fields.

### Field semantics

| Field | Owner | Meaning |
|---|---|---|
| `response_id` | provider-native | Provider response/stream id. |
| `item_id` | provider-native | Provider output item id, only when the provider emits one. |
| `output_index` | provider-native | Provider output index, when emitted. |
| `summary_index` | provider-native | Provider reasoning summary index, when emitted. |
| `choice_index` | Chat Completions-native | `choices[n].index` for OpenAI-compatible chunks. |
| `stream_kind` | Geppetto-normalized | Logical stream family: `reasoning`, `content`, `tool_call`, `finish`, `unknown`. |
| `correlation_key` | Geppetto-normalized | Stable join key for provider streams when `item_id` is absent or insufficient. |
| `tool_call_id` | provider-native when present | Provider tool/function call id. |
| `tool_call_index` | Chat Completions-native | `delta.tool_calls[n].index` for streamed tool argument assembly. |

### Correlation key examples

For Chat Completions reasoning:

```text
openai-chat:<response_id>:choice:<choice_index>:reasoning
```

For Chat Completions visible content:

```text
openai-chat:<response_id>:choice:<choice_index>:content
```

For Chat Completions tool calls:

```text
openai-chat:<response_id>:choice:<choice_index>:tool:<tool_call_id>
```

If a tool-call ID is not present yet but a tool index is present:

```text
openai-chat:<response_id>:choice:<choice_index>:tool-index:<tool_call_index>
```

For Responses API, we can set `correlation_key` to an item-based identity without replacing item ID:

```text
openai-responses:<response_id>:item:<item_id>
```

When item ID is absent, fall back to output/summary identity:

```text
openai-responses:<response_id>:output:<output_index>:summary:<summary_index>
```

## ASCII structure comparison

### Responses API

```text
[user]
  |
  v
[response resp_123]
  |
  +--> [item rs_001: reasoning]
  |       +--> summary delta: "The user wants..."
  |       +--> summary delta: " me to compare..."
  |
  +--> [item msg_002: assistant text]
  |       +--> text delta: "Let me start"
  |       +--> text delta: " by checking..."
  |
  +--> [item fc_003: function call]
          +--> args delta: "{\"sql\":\"SELECT"
          +--> args delta: " COUNT(*) ...\"}"
```

### Chat Completions

```text
[user]
  |
  v
[chat.completion stream id=e44f...]
  |
  +--> choice[0].delta.reasoning_content: "The user wants..."
  +--> choice[0].delta.reasoning_content: " me to compare..."
  |
  +--> choice[0].delta.content: "Let me start"
  +--> choice[0].delta.content: " by checking..."
  |
  +--> choice[0].delta.tool_calls[0].id: call_abc
  +--> choice[0].delta.tool_calls[0].function.arguments: "{\"sql\":\"SELECT"
  +--> choice[0].finish_reason: "tool_calls"
```

## Implementation plan

1. Geppetto observability fields:
   - Extend `observability.Record` with `ChoiceIndex`, `StreamKind`, `CorrelationKey`, `ToolCallID`, and `ToolCallIndex`.
   - Add helper functions for provider data maps and normalized correlation key construction.

2. Geppetto OpenAI Chat Completions:
   - Extract `choice_index`, reasoning/content/tool stream kind, and tool call IDs/indexes from decoded provider chunks.
   - Attach correlation metadata to `EventThinkingPartial` metadata and `EventInfo` thinking start/end events.
   - Optionally attach content correlation metadata to partial completion events.
   - Attach tool correlation metadata to tool-call events where possible.

3. Geppetto OpenAI Responses:
   - Preserve native `item_id` fields.
   - Add `correlation_key` derived from `response_id` + `item_id` or output/summary fallback.

4. Pinocchio protobuf and reasoning plugin:
   - Add correlation fields to `ReasoningUpdate`.
   - Parse fields from Geppetto metadata.
   - Include `correlation_key` in the reasoning segment key when present.

5. Pinocchio tool/message payloads:
   - Add relevant correlation fields to `ChatMessageUpdate` and `ToolCallUpdate` if needed by the frontend/debug export.
   - Keep backward-compatible optional/additive protobuf fields.

6. Pinocchio web-chat frontend/debug SQLite:
   - Regenerate Go/TypeScript protobufs.
   - Preserve new fields in typed frontend payload decoding.
   - Add SQLite columns/views for correlation keys.

7. CoinVault:
   - Copy/regenerate external Pinocchio chatapp protobuf TypeScript.
   - Update debug analysis SQL to use fallback correlation keys.

## Design decisions

- Do not overload `item_id` with synthetic values.
- Put provider-derived correlation in Geppetto because Geppetto sees provider-native shape.
- Put UI mapping in Pinocchio because Pinocchio owns chatapp payloads.
- Do not change Sessionstream; it already carries protobuf payloads opaquely and records raw JSON.
- Keep all fields additive for compatibility.

## Validation strategy

- Geppetto unit tests for Chat Completions correlation fields on reasoning/content/tool chunks.
- Geppetto unit tests for Responses correlation key fallback.
- Pinocchio reasoning plugin tests for correlation key propagation and segmentation.
- Web frontend typecheck/tests after generated TS protobuf updates.
- CoinVault typecheck after generated TS protobuf update.
- If time permits, run a browser-backed DeepSeek smoke and verify SQL joins by `correlation_key`.
