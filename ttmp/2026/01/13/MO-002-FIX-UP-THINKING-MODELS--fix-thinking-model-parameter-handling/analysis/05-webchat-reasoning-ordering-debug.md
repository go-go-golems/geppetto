---
Title: Webchat Reasoning Ordering Debug
Ticket: MO-002-FIX-UP-THINKING-MODELS
Status: active
Topics:
    - bug
    - geppetto
    - go
    - inference
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
      Note: ConversationState snapshot/update for webchat.
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: |-
        Webchat run loop and snapshot hook wiring.
        Run loop snapshot hook wiring and Responses error reproduction.
    - Path: pkg/inference/toolhelpers/helpers.go
      Note: Tool loop phases and snapshot hook support.
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Responses SSE handling and reasoning block append order.
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: buildInputItemsFromTurn reasoning adjacency and item-based follower logic.
ExternalSources: []
Summary: Deep-dive on the webchat Responses reasoning ordering failure in multi-turn flows.
LastUpdated: 2026-01-14T22:35:58.000000000-05:00
WhatFor: Explain why Responses rejects reasoning items on the second webchat prompt and define a concrete debug plan.
WhenToUse: Use when diagnosing 400s about reasoning adjacency in pinocchio webchat.
---


# Webchat Reasoning Ordering Debug

## Goal

Identify why pinocchio webchat still sends invalid Responses inputs (reasoning without required follower) on the second prompt, despite ConversationState changes, and define a targeted debug path to capture the exact Turn ordering.

## Repro summary

User log excerpt (second prompt fails):

```
2026-01-14T21:56:54.691183864-05:00 INF /chat received component=webchat conv_id=conv-1768445812588-1ihlft profile=default prompt_len=5
2026-01-14T21:56:54.691378409-05:00 INF starting run loop component=webchat conv_id=conv-1768445812588-1ihlft run_id=34c684f8-955d-4367-97a8-f74b15ff7c7c
2026-01-14T21:56:58.131318201-05:00 INF run loop finished component=webchat conv_id=conv-1768445812588-1ihlft run_id=34c684f8-955d-4367-97a8-f74b15ff7c7c
2026-01-14T21:57:00.859905392-05:00 INF /chat received component=webchat conv_id=conv-1768445812588-1ihlft profile=default prompt_len=5
2026-01-14T21:57:00.860114409-05:00 INF starting run loop component=webchat conv_id=conv-1768445812588-1ihlft run_id=34c684f8-955d-4367-97a8-f74b15ff7c7c
2026-01-14T21:57:01.486262469-05:00 ERR RunToolCallingLoop: engine inference failed error="responses api error: status=400 body=map[error:map[code:<nil> message:Item 'rs_0968a5f5acb9fffd0069685777dac081a19a59cc8abe65cfec' of type 'reasoning' was provided without its required following item. param:input type:invalid_request_error]]" iteration=1
```

## Expected ordering contract

OpenAI Responses requires:

```
...,
{ "type": "reasoning", "id": "rs_..." },
{ "type": "message", "role": "assistant", "content": [...] }
```

Or:

```
...,
{ "type": "reasoning", "id": "rs_..." },
{ "type": "function_call", ... }
```

Any other follower (role-based message, tool output, system/user item) triggers the 400 error.

## Where the ordering is constructed

### Webchat state flow

```
HTTP /chat
  -> ConversationState snapshot (conv.snapshotForPrompt)
  -> toolhelpers.RunToolCallingLoop
     -> engine.RunInference
        -> openai_responses.buildInputItemsFromTurn
```

Key code:
- `pinocchio/pkg/webchat/conversation.go` (ConversationState snapshot/update)
- `pinocchio/pkg/webchat/router.go` (run loop)
- `geppetto/pkg/steps/ai/openai_responses/helpers.go` (Responses input builder)

### Responses input builder (current logic)

Pseudo-view of `buildInputItemsFromTurn`:

```
latestReasoningIdx = last reasoning block index in Turn
if latestReasoningIdx == -1:
  emit all blocks as role-based messages/tool items
else:
  emit blocks before latestReasoningIdx (skip reasoning, and skip "last assistant before reasoning")
  if next block after reasoning is:
     - assistant text -> emit reasoning + item-based message
     - tool_call -> emit reasoning + tool_call/tool_use group
     - otherwise -> emit no reasoning
  emit remaining blocks after reasoning group
```

This should only emit reasoning when a valid follower exists.

## Why the 400 still happens (hypotheses)

The error proves a reasoning item is present in the Responses input **without** a valid follower immediately afterward. That means one of the following is true:

1) **Tool-call follower is missing required fields**
   - The Turn has `tool_call` blocks after reasoning, but `buildInputItemsFromTurn` fails to emit them because `call_id` or `name` is empty.
   - Result: reasoning is emitted, but no function_call items are appended.

2) **Assistant follower skipped due to text filtering**
   - The Turn has `llm_text` after reasoning, but payload text is empty/whitespace.
   - The builder skips the item-based message, leaving reasoning with no follower.

3) **Turn ordering is not what we think**
   - Reasoning blocks might be followed by `tool_use` or `user` blocks (from tool loop or new prompt append), producing invalid adjacency.
   - This should cause reasoning to be omitted, but may still be emitted if the code misidentifies the follower.

4) **Latest reasoning block is not the one you think**
   - Multiple reasoning blocks exist; the "latest" one might be the final block in the Turn and lack a follower, while an earlier reasoning+assistant pair exists.
   - The builder only considers the latest reasoning; earlier reasoning blocks are skipped.

5) **ConversationState update filters too aggressively**
   - Filtering only `middleware=systemprompt` blocks is correct, but if other blocks accidentally carry that metadata, we could be dropping assistant text blocks and breaking adjacency.

## How to prove the root cause

Capture the exact Turn ordering **before inference** on the second run.

### Snapshot hook instrumentation

Enable snapshot logging using the env var:

```
PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR=/tmp/pinocchio-webchat-snapshots
```

The router now attaches a `toolhelpers.WithTurnSnapshotHook` during the run loop and writes per-phase YAML snapshots:

```
/tmp/pinocchio-webchat-snapshots/<conv_id>/<run_id>/<timestamp>-<phase>-<turnID>.yaml
```

Phases include `pre_inference`, `post_inference`, `post_tools` (from `RunToolCallingLoop`).

### What to inspect

For the **second prompt** run:

1) Inspect the **pre_inference** snapshot
2) Locate the latest `reasoning` block
3) Check the immediate next block in the YAML
4) Confirm:
   - `llm_text` with non-empty `payload.text`, or
   - `tool_call` with `payload.id` and `payload.name`

If not, the reason for the 400 is confirmed.

## Candidate fixes (once ordering is confirmed)

1) **Enforce reasoning adjacency at snapshot time**
   - Add `EnforceResponsesAdj` in ConversationState snapshot config for webchat.
   - Drop reasoning blocks that do not have a valid follower.

2) **Adjust input builder to avoid invalid reasoning emission**
   - In `buildInputItemsFromTurn`, only emit reasoning if a **serialized** follower is actually appended.
   - If tool_call cannot be serialized (missing id/name), do not include reasoning.

3) **Normalize tool_call fields**
   - Ensure tool_call blocks always carry `payload.id` and `payload.name` after tool loop.

## Immediate next steps

- Capture the snapshot files and identify the exact adjacency failure.
- Record the ordering in this doc with a concrete example from YAML.
- Decide whether to fix in ConversationState snapshot validation or in buildInputItemsFromTurn.

## DebugTap findings (raw request evidence)

Using `PINOCCHIO_WEBCHAT_DEBUG_TAP_DIR`, the raw request for the failing second prompt showed this input sequence:

```
[
  { role: "system", ... },
  { role: "user", ... },
  { type: "reasoning", id: "rs_..." },
  { type: "message", role: "assistant", content: [ ... ] },   // missing id
  { role: "user", ... }
]
```

The prior response contained a message output item with an ID (e.g. `msg_...`), but we never persisted it into the Turn. That means the reasoning item is followed by a message **without** the required output item ID.

Raw file example (pre-fix):
- `/tmp/pinocchio-debugtap/conv-debug-5/2cf1b63b-a561-4f6b-b0eb-941a53c4646b/run-2/raw/turn-1-http-request.json`

## Root cause

The Responses API requires reasoning items to be followed by the **exact** output item they were paired with (same item ID). We were emitting the follower message without its `id`, so the API rejected the reasoning item as missing its required follower.

## Fix implemented

1) Capture message output item IDs in the Responses engine:
   - Streaming: `response.output_item.added` and `response.output_item.done`
   - Non-streaming: `responsesResponse.Output` item IDs
2) Persist that ID on the assistant text block payload (`PayloadKeyItemID`).
3) When emitting the item-based follower after reasoning, include `ID` in the `type:"message"` input item.

Relevant files:
- `geppetto/pkg/steps/ai/openai_responses/engine.go`
- `geppetto/pkg/steps/ai/openai_responses/helpers.go`

## Validation

After the fix, the raw request includes a message item with `id: "msg_..."` immediately after the reasoning item, and the second prompt completes without 400 errors.

Raw file example (post-fix):
- `/tmp/pinocchio-debugtap/conv-debug-6/fc4b4f7d-df0d-4372-800e-b23e719b5898/run-2/raw/turn-1-http-request.json`
