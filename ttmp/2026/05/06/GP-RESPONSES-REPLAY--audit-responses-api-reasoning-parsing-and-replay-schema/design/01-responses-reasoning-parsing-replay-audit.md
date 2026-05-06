---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Streaming and non-streaming parser audited for reasoning item capture
    - Path: pkg/steps/ai/openai_responses/engine_test.go
      Note: Recommended location for incoming parser regression tests
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: Request model and replay builder audited for Responses reasoning items
    - Path: pkg/steps/ai/openai_responses/helpers_test.go
      Note: Recommended location for replay regression tests
    - Path: pkg/turns/keys_gen.go
      Note: Payload keys used for text
    - Path: pkg/turns/types.go
      Note: Turn and Block data model used by Responses parser/replay
    - Path: ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/sources/openai-reasoning-guide.md
      Note: Official source for reasoning context and encrypted reasoning semantics
    - Path: ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/sources/openai-reasoning-items-cookbook.md
      Note: Official cookbook examples for reasoning item replay and function-call context
    - Path: ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/sources/openai-responses-create-api-reference.md
      Note: Official schema source for Reasoning item fields including content reasoning_text
    - Path: ttmp/2026/05/06/GP-RESPONSES-REPLAY--audit-responses-api-reasoning-parsing-and-replay-schema/sources/openai-responses-object-api-reference.md
      Note: Official schema source for Responses object/output items
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Responses API Reasoning Parsing and Replay Audit

## Executive summary

Geppetto's `openai_responses` engine is a hand-written, minimal HTTP/SSE implementation of the OpenAI Responses API. It maps Geppetto's durable `turns.Turn` model into Responses `input[]` items, sends requests, parses streaming or non-streaming provider output, and appends new turn blocks so later calls can replay conversation state.

The recent CoinVault/web-chat history-loading work exposed an important schema boundary problem: we persisted a reasoning block with a synthetic UUID block ID and plaintext local thinking, then replayed that block as a Responses `reasoning` item. The provider rejected the request because it interpreted `input[].id` as a provider reasoning item ID, not an internal Geppetto block ID.

The immediate fix in commit `620f516` made replay safer by separating internal block identity from provider item identity:

- `Block.ID` remains local storage identity.
- `payload.item_id` is the provider item ID captured from provider output.
- replay only uses `payload.item_id` for `input[].id`.
- plaintext-only reasoning blocks are skipped because the current local request struct does not model reasoning `content`.

However, after downloading the official Responses documentation and SDK-derived types, we found a discrepancy: the official `Reasoning` item schema does have an optional `content` field with `reasoning_text` entries. Geppetto's current `responsesInput` struct does not model that field for reasoning items, even though `responsesOutputItem` can parse output `content`. This document explains the system, the official API shape, every known discrepancy, and a staged implementation plan for an intern to make the code correct, testable, and provider-neutral.

## Scope and goals

This ticket is about the Responses API integration in:

- `pkg/steps/ai/openai_responses/helpers.go`
- `pkg/steps/ai/openai_responses/engine.go`
- `pkg/steps/ai/openai_responses/helpers_test.go`
- `pkg/steps/ai/openai_responses/engine_test.go`
- `pkg/turns/*`

The goal is not merely to fix one OpenAI error. The goal is to align the Geppetto turn model and the Responses API item model so that:

- incoming provider output is parsed without losing important item identity or content;
- durable turns distinguish local IDs from provider item IDs;
- replay emits valid Responses `input[]` items;
- replay is robust across OpenAI-compatible providers that may not use OpenAI's exact `rs_...` ID prefix;
- diagnostic previews reveal item types and redacted sensitive fields clearly;
- tests encode the official contract and the local policy decisions.

## Source documents archived in this ticket

The official/web source snapshots are in `sources/`:

- `sources/openai-reasoning-guide.md`
- `sources/openai-reasoning-items-cookbook.md`
- `sources/openai-responses-create-api-reference.md`
- `sources/openai-responses-object-api-reference.md`

Key API reference facts from these documents:

- The reasoning guide says reasoning items should be preserved in context for function-calling flows.
- In stateless mode, requests should include `reasoning.encrypted_content` so output reasoning items contain `encrypted_content` that can be sent in later requests.
- Raw reasoning tokens are not exposed as raw hidden chain-of-thought, but the API can expose summaries and the API reference lists optional `content` entries of type `reasoning_text`.
- The Responses create reference describes a `Reasoning` item with `id`, `summary`, `type: "reasoning"`, optional `content`, optional `encrypted_content`, and optional `status`.

Important excerpts:

> When doing function calling with a reasoning model in the Responses API, we highly recommend you pass back any reasoning items returned with the last function call (in addition to the output of your function).

> You can pass reasoning items from previous responses either using the `previous_response_id` parameter, or by manually passing in all the output items from a past response into the input of a new one.

> Any reasoning items in the output array will now have an `encrypted_content` property, which will contain encrypted reasoning tokens that can be passed along with future conversation turns.

> Reasoning object: `id`, `summary`, `type`, optional `content`, optional `encrypted_content`, optional `status`.

## System overview for a new intern

### The Turn model

Geppetto represents a conversation/inference state as a `turns.Turn`.

```text
Turn
├── ID
├── Blocks[]
│   ├── BlockKindSystem
│   ├── BlockKindUser
│   ├── BlockKindLLMText
│   ├── BlockKindReasoning
│   ├── BlockKindToolCall
│   └── BlockKindToolUse
├── Metadata
└── Data
```

Relevant files:

- `pkg/turns/types.go`
- `pkg/turns/block_kind_gen.go`
- `pkg/turns/keys_gen.go`
- `pkg/turns/helpers_blocks.go`
- `pkg/turns/serde/serde.go`

A `turns.Block` has:

- `ID`: local block identity used by Geppetto persistence and references.
- `Kind`: canonical block kind.
- `Role`: optional role string.
- `Payload`: provider/app-specific content map.
- `Metadata`: additional metadata.

The important payload keys are:

- `text`: message or local thinking text.
- `id`: tool call/call ID for some block kinds.
- `name`: tool name.
- `args`: tool/function arguments.
- `result`: tool result.
- `error`: tool error.
- `images`: image inputs.
- `encrypted_content`: Responses encrypted reasoning blob.
- `summary`: Responses reasoning summary entries.
- `item_id`: provider output item ID.

### The Responses engine

The engine lives under `pkg/steps/ai/openai_responses`.

Important files:

- `helpers.go`: request structs, input builder, tool schema conversion, multimodal message parts.
- `engine.go`: HTTP transport, SSE loop, non-streaming parsing, event publishing, turn mutation.
- `engine_test.go`: streaming and non-streaming behavior tests.
- `helpers_test.go`: request-building tests.

The lifecycle is:

```text
Turn before inference
  ↓ buildResponsesRequest(t)
Responses request JSON
  ↓ POST /v1/responses
SSE stream or JSON response
  ↓ parse provider events/items
Geppetto events for UI/consumers
  ↓ append new Blocks to Turn
Turn after inference
  ↓ persisted by caller
Future request replays latest Turn
```

### Why replay exists

The Responses API can be used in multiple state modes:

1. Server-side state: use `previous_response_id` and let OpenAI retain context.
2. Manual context: include prior output items in the next `input[]`.
3. Stateless/ZDR context: include encrypted reasoning items using `include: ["reasoning.encrypted_content"]` and replay those items later.

Geppetto's current engine primarily implements manual/stateless context replay. It builds the next request from the full `Turn.Blocks` list instead of using `previous_response_id`.

## Official Responses item model versus Geppetto block model

### Official request/response item categories

A Responses `input[]` array can contain several item shapes. The ones relevant here are:

```text
Message-like input
├── role: user | system | developer | assistant
└── content[]
    ├── input_text
    ├── input_image
    ├── input_file
    ├── output_text
    └── refusal

Reasoning item
├── type: reasoning
├── id
├── summary[]
│   └── { type: summary_text, text }
├── content[]              optional
│   └── { type: reasoning_text, text }
├── encrypted_content      optional
└── status                 optional

Function call item
├── type: function_call
├── id                     provider item id
├── call_id                semantic call id
├── name
├── arguments
└── status                 optional

Function call output item
├── type: function_call_output
├── call_id
└── output
```

### Geppetto block mapping today

Current builder logic in `helpers.go` maps:

```text
BlockKindSystem    → message role=system content=input_text
BlockKindUser      → message role=user content=input_text/input_image/input_file
BlockKindLLMText   → message role=assistant content=output_text
BlockKindReasoning → item type=reasoning summary/encrypted_content/id-from-payload.item_id
BlockKindToolCall  → item type=function_call call_id/name/arguments/id-from-payload.item_id
BlockKindToolUse   → item type=function_call_output call_id/output
```

Current parser logic in `engine.go` maps:

```text
response.output_item.added type=reasoning
  → thinking-started info event
  → currentReasoningItemID = item.id
  → latestEncryptedContent = item.encrypted_content if present

response.reasoning_text.delta / done
  → EventThinkingPartial
  → currentReasoningText buffer

response.reasoning_summary_text.delta
  → EventThinkingPartial
  → currentReasoningSummary buffer

response.output_item.done type=reasoning
  → BlockKindReasoning with:
      ID = provider item id if available
      payload.item_id = provider item id if available
      payload.text = currentReasoningText if non-empty
      payload.encrypted_content = encrypted content if available
      payload.summary = item.summary or current summary buffer if available

response.output_text.delta / done
  → assistant partial/final text buffers

response.output_item.done type=function_call
  → pending tool call for final Turn append

end of stream
  → append assistant BlockKindLLMText if message text exists
  → append tool call blocks if captured
```

## How incoming parsing works today

### Streaming flow

The streaming path starts in `Engine.RunInference` after it sends a request with `stream: true`.

Simplified pseudocode:

```text
send request
create stream buffers:
  message
  thinkBuf
  summaryBuf
  currentReasoningText
  currentReasoningSummary
  currentReasoningItemID
  latestEncryptedContent
  latestMessageItemID
  callsByItem
  finalCalls

for each SSE event:
  normalize event name
  parse JSON payload into map
  switch event type:
    output_item.added:
      if reasoning: reset reasoning buffers, capture item id, maybe encrypted_content
      if message: capture message item id
      if web_search_call: publish search events

    reasoning_text.delta/done:
      update currentReasoningText
      publish thinking partial

    reasoning_summary_text.delta:
      update currentReasoningSummary and summaryBuf
      publish thinking partial

    output_item.done:
      if reasoning: append BlockKindReasoning
      if message: backfill assistant text
      if function_call: create pending tool call

    output_text.delta/done:
      update assistant message buffers
      publish assistant partials

    function_call_arguments.delta/done:
      update pending call args

    completed/failed/error:
      update usage or error state

on stream end:
  append assistant message block
  append tool call blocks
  persist inference_result metadata
  publish final event
```

Diagram:

```text
SSE events
  ├── reasoning item lifecycle
  │   ├── output_item.added(reasoning)
  │   ├── reasoning_text.delta*
  │   ├── reasoning_summary_text.delta*
  │   └── output_item.done(reasoning)
  │       └── Turn.Blocks += BlockKindReasoning
  │
  ├── assistant message lifecycle
  │   ├── output_item.added(message)
  │   ├── output_text.delta*
  │   └── output_item.done(message)
  │       └── Turn.Blocks += BlockKindLLMText at stream end
  │
  └── function call lifecycle
      ├── function_call_arguments.delta*
      └── output_item.done(function_call)
          └── Turn.Blocks += BlockKindToolCall at stream end
```

### Non-streaming flow

The non-streaming path unmarshals the whole JSON response into `responsesResponse`.

Simplified pseudocode:

```text
unmarshal response into rr
for output item in rr.Output:
  if type == reasoning:
    create BlockKindReasoning
    copy id to Block.ID and payload.item_id
    copy output content text into payload.text
    copy summary into payload.summary
    copy encrypted_content into payload.encrypted_content
    append block

  if type == message:
    remember latest message item id

  for content item in output item content:
    if output_text/text: append to assistant message
    if output_json: append JSON string

append assistant message block if non-empty
persist usage and inference result
publish final event
```

## How replay works today

Replay is performed by `buildInputItemsFromTurn(t)` in `helpers.go`.

The function walks `t.Blocks` in order and appends Responses `input[]` items.

Important behavior:

- Regular messages are emitted as role-based items with content parts.
- Tool calls are emitted as `function_call` items.
- Tool outputs are emitted as `function_call_output` items.
- Reasoning is only emitted when followed immediately by an assistant message or tool call group.
- The builder tries to preserve provider ordering constraints by keeping a reasoning item next to its associated message/tool-call items.

Current reasoning replay policy after commit `620f516`:

```text
reasoningItem(block):
  enc = payload.encrypted_content
  summary = payload.summary
  itemID = payload.item_id

  if enc == "" and summary is empty:
    skip item

  item = { type: "reasoning", summary: summary or [] }

  if itemID exists:
    item.id = itemID

  if enc exists:
    item.encrypted_content = enc

  return item
```

This is safer than the previous policy because it never sends `Block.ID` as a provider item ID. But it is not fully aligned to the official schema because it drops plaintext reasoning text instead of replaying it as `content: [{ type: "reasoning_text", text: ... }]`.

## Discrepancy inventory

### D1. Reasoning `content` is missing from request model

Official API reference:

```json
{
  "type": "reasoning",
  "id": "...",
  "summary": [{"type":"summary_text","text":"..."}],
  "content": [{"type":"reasoning_text","text":"..."}],
  "encrypted_content": "...",
  "status": "completed"
}
```

Current Geppetto `responsesInput` has `Content []responsesContentPart`, but that field is treated as message content. It can technically serialize `content`, but `responsesContentPart` does not distinguish reasoning content from message content in the builder, and `reasoningItem` never populates it.

Impact:

- plaintext local reasoning is lost on replay;
- plaintext-only reasoning blocks become invalid/no-op if not skipped;
- current comment saying the API has no plaintext reasoning field is inaccurate.

Recommended fix:

- Rename or generalize content structs so reasoning content can be represented explicitly.
- Add `reasoning_text` support in replay.
- Decide policy for when plaintext reasoning is allowed to be replayed.

Pseudocode:

```text
type ResponsesInputItem struct:
  role
  type
  id
  content[] of ResponsesContent
  summary[]
  encrypted_content
  status
  call_id
  name
  arguments
  output

type ResponsesContent struct:
  type
  text
  image_url
  file_id
  detail
```

Then:

```text
if block.kind == Reasoning:
  content = []
  if payload.text != "":
    content.append({type: "reasoning_text", text: payload.text})
  item.content = content
```

Open question:

- Does OpenAI accept client-replayed `reasoning_text` in `input[]`, or is the schema only present because output items can be echoed back unchanged? The official `ResponseReasoningItemParam` says it is an input param, but we should verify with a live test before enabling by default.

### D2. Provider item ID and local block ID were historically conflated

The bug that triggered this audit was:

```text
input[8].id = 87d2ce2a-bfbb-413d-be09-bf612998ba12
```

That UUID was an internal `Block.ID`, not a provider item ID.

Current fix:

- capture provider reasoning IDs into `payload.item_id`;
- replay from `payload.item_id` only.

Remaining follow-up:

- migrate or tolerate older stored turns where provider IDs are only in `Block.ID`.
- never use ID prefix heuristics like `strings.HasPrefix(id, "rs")` for provider-neutral correctness.

Recommended compatibility policy:

```text
if payload.item_id exists:
  use it
else if migration setting enabled and block metadata says provider=openai-responses and block.ID came from provider:
  use block.ID
else:
  omit id
```

Do not infer provider origin from the textual prefix alone.

### D3. `latestEncryptedContent` is global across streaming reasoning items

In `engine.go`, streaming state has one `latestEncryptedContent` string outside the per-reasoning-item buffers. On `output_item.added(reasoning)`, the code resets text and summary buffers but does not reset `latestEncryptedContent` to empty. If a later reasoning item lacks encrypted content, it may accidentally inherit encrypted content from an earlier item.

Current code shape:

```text
var latestEncryptedContent string

on output_item.added(reasoning):
  reset text and summary
  if item.encrypted_content exists:
    latestEncryptedContent = item.encrypted_content

on output_item.done(reasoning):
  enc = latestEncryptedContent
  if done item has encrypted_content:
    enc = done encrypted_content
```

Recommended fix:

```text
on output_item.added(reasoning):
  currentReasoningEncryptedContent = ""
  if item.encrypted_content exists:
    currentReasoningEncryptedContent = item.encrypted_content

on output_item.done(reasoning):
  enc = currentReasoningEncryptedContent
  if item.encrypted_content exists:
    enc = item.encrypted_content
```

Also rename `latestEncryptedContent` to `currentReasoningEncryptedContent`.

### D4. Streaming parser ignores `item.content` reasoning_text on `output_item.done`

The non-streaming parser reads `oi.Content` for reasoning items. The streaming parser primarily reads `response.reasoning_text.delta/done` and the current buffers. If `output_item.done(reasoning)` includes a `content` array with `reasoning_text`, but no separate `response.reasoning_text.*` events were seen, the streaming parser may miss it.

Recommended fix:

```text
on output_item.done(reasoning):
  text = currentReasoningText
  if item.content contains reasoning_text:
    backfill missing suffix into text buffer
  payload.text = text
```

This mirrors the assistant message backfill behavior already implemented for `output_item.done(message)`.

### D5. Summary parsing reuses a payload helper on provider item maps

`reasoningSummaryEntriesFromPayload` is used both for internal block payloads and provider item maps. That works only because both use the key `summary`, but the function name and accepted shapes are geared toward Geppetto payloads.

Recommended fix:

Split into:

```text
reasoningSummaryEntriesFromBlockPayload(payload)
reasoningSummaryEntriesFromProviderItem(item)
```

Both may share a lower-level normalizer.

### D6. Response item `status` is parsed but not stored/replayed

The official Reasoning item has `status`. Function calls and messages also often have status. Geppetto currently does not preserve reasoning status in payload or replay it.

Impact:

- probably low for completed items;
- could matter for incomplete/background/compaction flows;
- useful for debugging.

Recommended fix:

- introduce `PayloadKeyStatus` or a provider-scoped metadata key;
- store status for reasoning, message, and function_call output items;
- replay status only if official input schema allows/needs it.

### D7. Function-call item grouping is approximate

The official guide says when function calls happen, pass back reasoning items, function call items, and function call output items since the last user message, untouched. Current builder walks blocks and groups a reasoning item with immediately following tool calls and tool outputs. This handles simple contiguous cases, but durable turns do not explicitly store provider response boundaries or grouping identity.

Risk:

- multiple reasoning/tool-call groups in one response are hard to reconstruct without group metadata;
- future server tools or multi-call flows may violate expectations.

Middleware reordering is intentionally out of scope. If a middleware reorders Responses blocks into a provider-invalid sequence, that middleware is responsible for preserving or repairing provider ordering. This audit focuses on making the Responses parser and replay builder preserve the order they receive.

Recommended fix:

Store OpenAI Responses-specific output group metadata in block metadata, not in general-purpose payload fields:

```text
metadata[openai_responses.response_id]
metadata[openai_responses.output_index]
metadata[openai_responses.item_type]
metadata[openai_responses.status]
```

Keep `payload.item_id` because it matches the Responses API/event field name and is the actual replay payload for `input[].id`. Use metadata for fields that are OpenAI Responses bookkeeping rather than generic block payload.

Then replay can group by response ID/output sequence rather than just adjacent block kind.

### D8. Request preview hides item type and sensitive fields

The debug preview in `engine.go` only logs `role` and `parts`. Reasoning items appear as:

```text
role="" parts=[]
```

This made a reasoning item look like an empty assistant message. It also hid whether `id`, `summary`, or `encrypted_content` were present.

Recommended fix:

Create a `previewResponsesInputItem` helper that logs:

- `type`
- `role`
- redacted `id`
- content part types and text lengths
- `has_encrypted_content`
- `summary_count`
- `call_id`
- `name`

Pseudocode:

```text
preview(item):
  return {
    type: item.type,
    role: item.role,
    id: redact(item.id),
    call_id: redact(item.call_id),
    name: item.name,
    content: summarizeContent(item.content),
    has_encrypted_content: item.encrypted_content != "",
    encrypted_content_len: len(item.encrypted_content),
    summary_count: len(item.summary),
  }
```

### D9. `include: ["reasoning.encrypted_content"]` is unconditional

Geppetto currently always adds `reasoning.encrypted_content`. For this ticket, keep that behavior. It is aligned with the stateless/ZDR continuation mode that Geppetto is implementing, and adding capability plumbing now would distract from the parser/replay schema fixes.

If a future provider rejects the include parameter, address that in a provider-configuration ticket. Do not add capability flags in this implementation pass.

## Recommended target design

### Data model

Add or formalize provider item metadata without overloading `Block.ID`.

Preferred payload keys:

```text
item_id             provider output item id; keep this name because it matches the Responses spec
encrypted_content   encrypted reasoning continuation blob
summary             reasoning summary array
text                local/renderable text or reasoning_text
```

OpenAI Responses-specific response metadata should live in block metadata, not in general payload keys:

```text
metadata[openai_responses.response_id]
metadata[openai_responses.output_index]
metadata[openai_responses.item_type]
metadata[openai_responses.status]
```

Keep provider wire names like `item_id` where they match the Responses spec. Do not introduce `provider_item_id` or migration compatibility for this ticket.

### Request model

Refactor HTTP structs for clarity:

```text
ResponsesRequest
  model
  input[] ResponsesInputItem
  reasoning
  include
  tools
  ...

ResponsesInputItem
  // common
  type
  id
  status

  // message
  role
  content[] ResponsesContentPart

  // reasoning
  summary[] ReasoningSummaryPart
  encrypted_content

  // function_call
  call_id
  name
  arguments

  // function_call_output
  output

ResponsesContentPart
  type
  text
  image_url
  file_id
  detail
  annotations?
```

Important: use the same `content` field for message content and reasoning content, but encode part types accurately:

```json
{"type":"reasoning_text","text":"..."}
```

### Replay policy

Recommended default policy:

1. Replay encrypted reasoning if present.
2. Replay summary if present.
3. Replay provider item ID only from `payload.item_id`.
4. Replay `reasoning_text` according to the official schema once implemented and tested.
5. Until `reasoning_text` replay is implemented, continue skipping plaintext-only reasoning blocks to avoid empty reasoning items.
6. Preserve contiguous provider item groups for tool-call flows.

Pseudocode:

```text
buildReasoningInput(block, capabilities):
  item = { type: "reasoning" }

  if payload.item_id:
    item.id = payload.item_id

  if payload.summary:
    item.summary = normalizeSummary(payload.summary)
  else:
    item.summary = []

  if payload.encrypted_content:
    item.encrypted_content = payload.encrypted_content

  if payload.text != "":
    item.content = [{ type: "reasoning_text", text: payload.text }]

  if item has no id, no encrypted_content, no summary entries, no content:
    return nil

  return item
```

For OpenAI specifically, run a small live verification before enabling plaintext reasoning replay by default. The official SDK type suggests it is valid, but reasoning text may be sensitive and provider/model-dependent.

### Incoming parsing policy

Streaming parser should capture all item properties from both lifecycle events and terminal item payloads.

Pseudocode:

```text
on output_item.added(reasoning):
  current = new ReasoningAccumulator
  current.itemID = item.id
  current.status = item.status
  current.encryptedContent = item.encrypted_content
  current.outputIndex = event.output_index

on reasoning_text.delta/done:
  current.text += normalized delta/backfill
  publish thinking partial

on reasoning_summary_text.delta/done:
  current.summaryText += normalized delta/backfill
  publish thinking partial

on output_item.done(reasoning):
  merge terminal item into current:
    item.id
    item.status
    item.encrypted_content
    item.summary
    item.content[].reasoning_text
  append BlockKindReasoning with:
    ID = local block ID policy, preferably provider ID if known but not relied upon
    payload.item_id = item.id
    payload.text = merged reasoning text
    payload.summary = merged summary
    payload.encrypted_content = merged encrypted content
    payload.status = item.status
    metadata.response_id/output_index if available
  clear current
```

Use a per-item accumulator map if provider can interleave multiple reasoning items:

```text
reasoningByItemID: map[string]*ReasoningAccumulator
currentReasoningItemID: string
```

If events include `item_id`, use it. If not, use current item while inside a reasoning item.

### Replay grouping design

Manual Responses context should preserve provider output item groups. A durable representation should know which output items came from the same provider response.

A robust algorithm:

```text
collect blocks since last user message or since selected context boundary
partition provider-originated blocks by response_id and output_index order
for each group:
  emit items in original output order:
    reasoning
    function_call
    function_call_output if present after local tool execution
    message
append new user message
```

When `response_id` is missing, fall back to current adjacency logic.

Diagram:

```text
Provider response R1 output[]
  0: reasoning rs_1
  1: function_call fc_1 / call_1

Local execution
  function_call_output call_1

Replay request
  input[]
    previous messages...
    {type: reasoning, id: rs_1, encrypted_content: ...}
    {type: function_call, id: fc_1, call_id: call_1, ...}
    {type: function_call_output, call_id: call_1, output: ...}
    {role: user, content: ...}
```

## Implementation guide

### Phase 0: Preserve current safety

Before deeper changes, ensure current safety is locked down:

- No replay path uses `Block.ID` as `input[].id`.
- Plaintext-only reasoning is skipped unless explicitly enabled.
- Error messages surface in callers/UIs.

Tests:

- synthetic UUID block ID is not emitted as `id`.
- plaintext-only reasoning is omitted under default capabilities.
- encrypted reasoning with `payload.item_id` replays `id` and `encrypted_content`.

### Phase 1: Refactor structs and preview helpers

Modify `helpers.go`:

- rename `responsesInput` to `responsesInputItem` if desired;
- add explicit comments for common/message/reasoning/tool fields;
- ensure `Content` can carry `reasoning_text`;
- add `Status string`;
- add typed preview helper.

Pseudocode:

```text
func previewResponsesInput(items []responsesInput) []map[string]any:
  previews = []
  for item in items:
    previews.append(previewResponsesInputItem(item))
  return previews
```

Modify `engine.go` logging to use this helper.

Tests:

- preview of reasoning item shows `type=reasoning`, `has_encrypted_content=true`, `summary_count=N`, no raw encrypted blob.

### Phase 2: Capture incoming reasoning content completely

Streaming:

- replace loose variables with `ReasoningAccumulator`.
- reset encrypted content per item.
- merge `item.content[].reasoning_text` on terminal event.
- store provider `status`.

Non-streaming:

- store `payload.item_id`, `payload.text`, `payload.summary`, `payload.encrypted_content`, `payload.status`.
- ignore or separately store unsupported content types instead of silently concatenating all text.

Tests:

- streaming reasoning text only in `output_item.done.content` is persisted.
- two reasoning items do not leak encrypted content between each other.
- summary arrays from terminal item are preserved.

### Phase 3: Implement reasoning_text replay

The official Responses schema includes optional reasoning `content` entries with `type: "reasoning_text"`. Do not hide this behind a capability flag in this pass. Implement the official shape directly for OpenAI Responses, then validate it with unit tests and, if possible, an opt-in live conformance test.

Pseudocode:

```text
if text != "":
  item.Content = append(item.Content, {Type: "reasoning_text", Text: text})
```

Tests:

- plaintext reasoning replays as `content:[{type:"reasoning_text", text:"..."}]`.
- a reasoning item with encrypted content and text serializes both official fields.
- a reasoning item with no id, no summary, no encrypted content, and no text is omitted.

### Phase 4: Response grouping metadata

Add provider group metadata capture:

- response ID from `response.created`/`response.completed` payloads if present.
- output index from item events.
- item status.

Then update replay builder to prefer grouping by provider response/output index.

Tests:

- multiple function calls since last user replay in documented order.
- old turns without metadata still use adjacency fallback.

### Phase 5: Live provider conformance tests

Create a small manual or gated integration test script under the ticket or `cmd/examples` that can be run with real credentials.

Scenarios:

1. Stateless encrypted reasoning:
   - request with `include: ["reasoning.encrypted_content"]`, `store:false` if possible;
   - capture reasoning item;
   - replay encrypted reasoning item with follow-up;
   - expect success.

2. Plaintext reasoning content replay:
   - construct input with `type: reasoning`, `content:[{type:"reasoning_text", text:"..."}]`;
   - expect either success or documented provider rejection.

3. Tool-call grouping:
   - model calls a function;
   - replay reasoning + function_call + function_call_output;
   - expect success.

Keep this integration test opt-in because it uses paid API calls.

## Concrete file references

### Request construction

`pkg/steps/ai/openai_responses/helpers.go`

- `responsesRequest`
- `responsesInput`
- `responsesContentPart`
- `buildResponsesRequest`
- `buildInputItemsFromTurn`
- `buildResponsesMessageParts`
- `toolUsePayloadToJSONString`

### Streaming and non-streaming parsing

`pkg/steps/ai/openai_responses/engine.go`

- `RunInference`
- SSE event switch over `normalizeResponsesEventName(eventName)`
- reasoning parsing cases:
  - `response.output_item.added`
  - `response.reasoning_text.delta`
  - `response.reasoning_text.done`
  - `response.reasoning_summary_text.delta`
  - `response.output_item.done`
- final turn append logic
- non-streaming `responsesResponse` parsing

### Turn model

`pkg/turns/types.go`

- `Block`
- `Turn`
- `Data`
- `Metadata`

`pkg/turns/keys_gen.go`

- `PayloadKeyText`
- `PayloadKeyEncryptedContent`
- `PayloadKeySummary`
- `PayloadKeyItemID`

### Tests

`pkg/steps/ai/openai_responses/helpers_test.go`

- request shape tests;
- reasoning replay tests;
- function-call grouping tests.

`pkg/steps/ai/openai_responses/engine_test.go`

- streaming parser tests;
- non-streaming parser tests;
- SSE event compatibility tests.

## Suggested task checklist

- [ ] Add a typed `reasoning_text` content representation in request items.
- [ ] Add a redacted Responses input preview helper.
- [ ] Replace `latestEncryptedContent` with per-reasoning-item encrypted content state.
- [ ] Parse reasoning `content[].reasoning_text` from streaming `output_item.done`.
- [ ] Add status capture for reasoning items.
- [ ] Add official `reasoning_text` replay for plaintext reasoning content.
- [ ] Add tests for plaintext reasoning replay and truly empty reasoning omission.
- [ ] Add tests for encrypted content not leaking across multiple reasoning items.
- [ ] Add tests for provider item IDs coming only from `payload.item_id`.
- [ ] Add opt-in live conformance script or example.
- [ ] Update `pkg/doc/topics/06-inference-engines.md` and `pkg/doc/topics/08-turns.md` after implementation.

## Review guidance

When reviewing an implementation, ask these questions:

1. Does any replay path still use `Block.ID` as provider `input[].id`?
2. Can a reasoning item serialize with no useful fields?
3. Are encrypted reasoning blobs redacted in logs?
4. Are per-item streaming accumulators reset correctly?
5. Does streaming and non-streaming parsing produce equivalent blocks for equivalent provider output?
6. Are old persisted turns tolerated?
7. Are OpenAI-specific assumptions kept out of generic provider-neutral paths?
8. Is every official schema field we intentionally ignore documented?

## Current status after this audit

Current code is safer than before the invalid UUID incident, but not fully spec-aligned.

Safe today:

- provider IDs are replayed from `payload.item_id`, not local `Block.ID`;
- plaintext-only reasoning is skipped, avoiding empty reasoning replay;
- encrypted content and summaries continue to replay.

Needs work:

- official `reasoning_text` content is not modeled for replay;
- streaming parser should merge terminal `content[].reasoning_text`;
- encrypted-content state should be per reasoning item;
- OpenAI-specific response metadata should be preserved in `metadata[openai_responses.*]`;
- debug previews should show item type and redacted replay-relevant fields.
