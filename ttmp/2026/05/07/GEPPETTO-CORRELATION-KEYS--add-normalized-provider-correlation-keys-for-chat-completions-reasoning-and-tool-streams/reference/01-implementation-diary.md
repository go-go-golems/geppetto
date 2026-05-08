---
Title: Implementation Diary
Ticket: GEPPETTO-CORRELATION-KEYS
Status: active
Topics:
  - observability
  - reasoning
  - openai-compatibility
  - streaming
  - pinocchio
  - webchat
DocType: reference
Intent: long-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Chronological diary for normalized provider correlation key implementation.
LastUpdated: 2026-05-07T20:45:00-04:00
WhatFor: Record implementation steps, validation, failures, and follow-ups.
WhenToUse: Read before resuming or reviewing GEPPETTO-CORRELATION-KEYS.
---

# Implementation Diary

## Step 1: Ticket creation and design

The user asked to create a Geppetto ticket and then implement normalized provider correlation keys across Geppetto, Pinocchio, Pinocchio web-chat, and CoinVault. Sessionstream is intentionally out of scope because it already transports typed payloads and debug JSON without needing to understand provider identities.

I created ticket `GEPPETTO-CORRELATION-KEYS` under `geppetto/ttmp`, added the design guide, and added tasks covering analysis, Geppetto fields, Pinocchio propagation, web-chat export, CoinVault updates, and validation.

Key design decision: keep `item_id` provider-native and add a separate `correlation_key` for normalized/fallback joins.

## Step 2: Geppetto correlation fields

Implemented the first Geppetto slice.

### What changed

- Extended `observability.Record` with:
  - `choice_index`
  - `stream_kind`
  - `correlation_key`
  - `tool_call_id`
  - `tool_call_index`
- Extended the OpenAI-compatible Chat Completions stream decoder to retain `choices[0].index` as `ChoiceIndex`.
- Added Chat Completions correlation-key construction:
  - `openai-chat:<response_id>:choice:<choice_index>:reasoning`
  - `openai-chat:<response_id>:choice:<choice_index>:content`
  - `openai-chat:<response_id>:choice:<choice_index>:tool:<tool_call_id>`
  - `openai-chat:<response_id>:choice:<choice_index>:tool-index:<tool_call_index>`
- Attached correlation metadata to reasoning and content publish events.
- Attached correlation metadata to final merged tool-call events.
- Added Responses API correlation keys while preserving provider-native `item_id` semantics.

### Validation

```bash
go test ./pkg/steps/ai/openai ./pkg/steps/ai/openai_responses ./pkg/observability -count=1
```

Result: passed.

### Notes

This slice keeps `item_id` provider-native. Chat Completions fallback identity is represented by `correlation_key`, not by a fake item ID.
