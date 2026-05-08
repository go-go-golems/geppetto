---
Title: Claude tool-use duplicate text finalization bug analysis and implementation guide
Ticket: GP-CLAUDE-DUPLICATE-TEXT
Status: active
Topics:
  - geppetto
  - claude
  - streaming
  - observability
DocType: design-doc
Intent: long-term
Owners:
  - manuel
RelatedFiles:
  - Path: pkg/steps/ai/claude/content-block-merger.go
    Note: Claude streaming merger that converts Anthropic SSE events into Geppetto events.
  - Path: pkg/steps/ai/claude/content-block-merger_test.go
    Note: Regression coverage for text/tool-use/message-stop event sequences.
  - Path: pkg/steps/ai/claude/engine_claude.go
    Note: Streaming loop that observes provider events and publishes merger output.
Summary: Analysis and fix plan for duplicate assistant text after Claude/Haiku tool-use turns.
LastUpdated: 2026-05-08T00:05:00-04:00
---

# Claude tool-use duplicate text finalization bug analysis and implementation guide

## Summary

CoinVault's Haiku browser smoke run showed the first assistant text twice around the first tool call. The correlation SQLite artifact proved that the duplicate was not created by the browser reducer. It was already present as a second backend `ChatMessageFinished` event sent over the websocket.

The bug is in the Claude streaming conversion path. Anthropic sends one text content block, then one tool-use content block, then `message_delta` / `message_stop` with `stop_reason=tool_use`. Geppetto's `ContentBlockMerger` currently publishes the accumulated text again at message stop. Downstream code interprets that final text as another text segment after the tool call, producing `chat-msg-*:text:2` with the same text already finalized as `chat-msg-*:text:1`.

## Evidence from the CoinVault artifact

Artifact used for diagnosis:

```text
../2026-03-16--gec-rag/ttmp/2026/05/07/COINVAULT-OBSERVABILITY--add-observer-correlation-export-for-coinvault-web-chat/various/browser-runs/multimodel-correlation-20260507-233459/haiku/debug.sqlite
```

Provider sequence:

```text
content_block_start index=0 text
content_block_delta index=0 "I'll help"
content_block_delta index=0 " you compare ..."
content_block_delta index=0 " to best query this data."
content_block_stop  index=0
content_block_start index=1 tool_use toolu_012...
content_block_delta index=1 tool input JSON chunks
content_block_stop  index=1
message_delta stop_reason=tool_use
message_stop
```

Backend/UI sequence:

```text
ordinal 6 ChatInferenceFinished chat-msg-7:text:1  # correct text block finalization
ordinal 7 ChatToolCallStarted   toolu_012...
ordinal 8 ChatInferenceFinished chat-msg-7:text:2  # duplicate text from message_stop
```

Transport/frontend sequence:

```text
ordinal 8 backend_transport ui_event_sent ChatMessageFinished
ordinal 8 frontend raw websocket ChatMessageFinished chat-msg-7:text:2
ordinal 8 frontend parsed frame ChatMessageFinished chat-msg-7:text:2
ordinal 8 frontend UI mutation upsert chat-msg-7:text:2
```

## Root cause

`pkg/steps/ai/claude/content-block-merger.go` treats message-level stop events as if they always carry final text that should be emitted downstream:

```go
case api.MessageStopType:
    return []events.Event{events.NewFinalEvent(cbm.metadata, cbm.Text())}, nil
```

That is safe for a normal `end_turn` response, but it is wrong for a `tool_use` response. In a tool-use response, the assistant's text content block has already been streamed/finalized before the tool block starts. The message-level stop is the end of the provider message envelope, not a new assistant text segment.

`MessageDeltaType` also emits an empty textual partial with the full accumulated text. For `stop_reason=tool_use`, that event does not represent new text and should not be published as a text event.

## Implementation rule

For Claude/Anthropic streams:

- Text deltas should publish text partials.
- Text content block stop may publish the block-finalization partial used by downstream segment finalization.
- Tool-use block stop should publish a tool-call event.
- Message delta with only `stop_reason` / usage should update metadata but should not publish text.
- Message stop should publish the final lifecycle event for normal end-turn responses, but if the stop reason is `tool_use`, it must not publish a final event because downstream code treats final as a text-segment finalizer.

In pseudocode:

```text
on message_delta:
  update stop_reason / usage metadata
  return no text event

on message_stop:
  update stop_reason / usage metadata
  if stop_reason == "tool_use":
      return no event
  return Final(text=all accumulated text)
```

This preserves a final event for lifecycle consumers while preventing the message-level event from re-emitting text that was already sent through content-block events.

## Validation plan

1. Add a unit test with this sequence:
   - `message_start`
   - text block start/delta/stop
   - tool-use block start/input-delta/stop
   - `message_delta stop_reason=tool_use`
   - `message_stop`
2. Assert that:
   - the text delta is emitted once;
   - the text block stop emits the existing empty-delta partial;
   - the tool call is emitted;
   - `message_delta` emits no text event;
   - no final event is emitted for `tool_use` message stop.
3. Run:

```bash
go test ./pkg/steps/ai/claude -count=1
```

## Follow-up

The CoinVault matrix also showed no non-empty normalized `correlation_key` values for Haiku provider rows. That is separate from the duplicate text bug. It should become a follow-up normalization ticket if Anthropic provider correlation keys need to match the OpenAI/Responses correlation experience.
