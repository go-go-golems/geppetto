---
Title: Webchat Responses Reasoning Follower Bug Report
Ticket: MO-002-FIX-UP-THINKING-MODELS
Status: active
Topics:
    - bug
    - geppetto
    - pinocchio
    - openai
    - responses
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: Webchat run loop, DebugTap wiring, and /chat handling.
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
      Note: Conversation run sequencing for debug capture.
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Capture output message item IDs during streaming and non-streaming.
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: Emit message item IDs when pairing reasoning with assistant output.
ExternalSources: []
Summary: Bug report and resolution for Responses reasoning follower validation failure on second webchat prompt.
LastUpdated: 2026-01-14T22:37:00-05:00
WhatFor: Record the reproduction, root cause, and fix for the Responses reasoning follower validation error.
WhenToUse: Use when validating multi-turn Responses behavior or debugging similar 400s.
---



# Bug Report: Responses reasoning follower validation failure in webchat

## Summary

Multi-turn webchat conversations using the OpenAI Responses API fail on the second prompt for GPT-5 models. The API returns a 400 error: “Item '<rs_...>' of type 'reasoning' was provided without its required following item.” The webchat UI then shows a stuck “generating” state because the engine aborts mid-run.

The root cause was that we did not persist the prior assistant message item ID from the Responses output. On the next turn we emitted a reasoning item followed by a message item that lacked its `id`, which violates the Responses validation rule that a reasoning item must be immediately followed by the paired output item (same `id`).

## Severity

High. It breaks multi-turn chat for Responses-backed reasoning models (gpt-5, o-series).

## Environment

- App: pinocchio webchat (`cmd/web-chat`)
- API: OpenAI Responses (`--ai-api-type openai-responses`)
- Models: `gpt-5-mini` (also seen with `gpt-5`)
- Streaming enabled (default)

## Steps to reproduce

1) Start webchat:

```
PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR=/tmp/pinocchio-turns \
PINOCCHIO_WEBCHAT_DEBUG_TAP_DIR=/tmp/pinocchio-debugtap \
go run ./cmd/web-chat --log-level debug --with-caller --log-file /tmp/pinocchio-webchat.log \
  web-chat --addr :8090 --ai-api-type openai-responses --ai-engine gpt-5-mini --ai-max-response-tokens 512
```

2) Send first prompt:

```
curl -s -X POST http://localhost:8090/chat \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"hello","conv_id":"conv-debug","overrides":{}}'
```

3) Send second prompt with same `conv_id`:

```
curl -s -X POST http://localhost:8090/chat \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"What is going on here?","conv_id":"conv-debug","overrides":{}}'
```

## Expected behavior

The assistant responds to the second prompt and the conversation continues normally.

## Actual behavior

Second prompt fails with a 400 error. Example log:

```
responses api error: status=400 ... Item 'rs_...' of type 'reasoning' was provided without its required following item.
```

## Investigation and evidence

### Snapshot evidence

Turn snapshots (pre-inference) showed the expected order:

- user: "hello"
- reasoning block
- assistant text
- user: "What is going on here?"

### DebugTap evidence (root cause)

Raw request captured by DebugTap showed this sequence for the failing second prompt:

```
[
  { role: "system", ... },
  { role: "user", ... },
  { type: "reasoning", id: "rs_..." },
  { type: "message", role: "assistant", content: [ ... ] },   // missing id
  { role: "user", ... }
]
```

File:
- `/tmp/pinocchio-debugtap/conv-debug-5/.../run-2/raw/turn-1-http-request.json`

The first response contained a `message` output item with an ID (e.g. `msg_...`), but that ID was not stored in the Turn, so we could not re-emit it as the required follower.

## Root cause

We did not persist the assistant message output item ID from Responses responses. On the next turn, `buildInputItemsFromTurn` emitted a reasoning item followed by a message item without `id`. The Responses API treats that as a missing required follower for the reasoning item.

## Fix

1) Persist the Responses message output item ID on assistant text blocks.
2) When emitting the item-based follower after a reasoning item, include the message `id`.

### Code changes

- `geppetto/pkg/steps/ai/openai_responses/engine.go`
  - Capture message item IDs in streaming (`response.output_item.added` / `response.output_item.done`).
  - Capture message item IDs in non-streaming output.
  - Store `msg_...` in assistant block payload (`PayloadKeyItemID`).

- `geppetto/pkg/steps/ai/openai_responses/helpers.go`
  - Emit `ID` on the `type:"message"` follower when pairing reasoning with assistant output.

- DebugTap wiring (for verification):
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/conversation.go`

## Verification

After the fix, the second prompt completes without 400 errors and the Responses stream ends normally. The raw request now includes a message item with `id: "msg_..."` immediately after the reasoning item:

- `/tmp/pinocchio-debugtap/conv-debug-6/.../run-2/raw/turn-1-http-request.json`

## What needed to be done

- Collect raw request bodies to confirm the missing message item ID (DebugTap).
- Track message output item IDs and persist them in the Turn.
- Re-emit message IDs when constructing the reasoning follower item.

## Notes

- The error reproduces with multiple GPT-5 models and appears specific to the Responses validation rules for reasoning continuity.
- DebugTap is opt-in via `PINOCCHIO_WEBCHAT_DEBUG_TAP_DIR` and can be removed later if no longer needed.
