---
Title: Implementation diary
Ticket: GP-CLAUDE-DUPLICATE-TEXT
Status: active
Topics:
  - geppetto
  - claude
  - streaming
DocType: reference
Intent: long-term
Owners:
  - manuel
Summary: Chronological diary for fixing Claude tool-use duplicate text finalization.
LastUpdated: 2026-05-08T00:05:00-04:00
---

# Implementation diary

## 2026-05-08 00:05 — Ticket creation and root-cause framing

Created this ticket after correlating a CoinVault Haiku duplicate-message artifact through Geppetto provider records, backend pipeline records, transport fanout, frontend raw websocket records, parsed frames, UI mutations, and timeline entities.

The duplicate appears when Anthropic/Claude streams one text block, then one tool-use block, then ends the provider message with `stop_reason=tool_use`. Geppetto's Claude `ContentBlockMerger` publishes the accumulated text again at message stop, which downstream consumers treat as a new text segment after the tool call.

The implementation plan is to make `MessageDeltaType` metadata-only and make `MessageStopType` emit an empty final text payload when the stop reason is `tool_use`.

## 2026-05-08 00:15 — Implemented Claude tool-use finalization fix

I updated `pkg/steps/ai/claude/content-block-merger.go` and added a regression case in `pkg/steps/ai/claude/content-block-merger_test.go`.

### What changed

- `MessageDeltaType` now updates `stop_reason`, `stop_sequence`, and usage metadata but returns no text event. Anthropic `message_delta` can be metadata-only, and publishing accumulated text there risks downstream duplicate text updates.
- `MessageStopType` still emits a final event for lifecycle consumers, but if the accumulated stop reason is `tool_use`, it emits `Final(text="")` instead of `Final(text=cbm.Text())`.
- Added `stopReasonIsToolUse()` helper to keep the message-stop decision local to the merger.

### Regression test

The new test covers this provider sequence:

```text
message_start
content_block_start index=0 text
content_block_delta index=0 "I'll inspect the schema first."
content_block_stop  index=0
content_block_start index=1 tool_use sql_doc
content_block_delta index=1 {"topic":"inventory"}
content_block_stop  index=1
message_delta stop_reason=tool_use
message_stop
```

Expected emitted events:

```text
start
partial text delta
empty-delta block-finalization partial
tool-call
final with empty text
```

That preserves the tool-use lifecycle but prevents re-emitting the already-finalized text at message stop.

### Validation

```bash
go test ./pkg/steps/ai/claude -run 'TestContentBlockMerger' -count=1
go test ./pkg/steps/ai/claude -count=1
```

Both passed.

## 2026-05-08 00:20 — Broader package validation

After the targeted Claude package tests passed, I ran the broader AI steps test suite:

```bash
go test ./pkg/steps/ai/... -count=1
```

It passed for Claude, Claude API, Gemini, OpenAI, OpenAI Responses, runtime attribution, settings, and stream helpers. This gives confidence that the changed final-text behavior did not break adjacent provider packages.

## 2026-05-08 00:55 — Playwright retest showed empty final still duplicates downstream

I re-ran CoinVault Haiku through Playwright after commit `ae94308`. Because the workspace has `../go.work` including `./geppetto`, CoinVault should have picked up the local Geppetto checkout. The retest still showed duplicate intro text.

The new artifact was:

```text
../2026-03-16--gec-rag/ttmp/2026/05/07/COINVAULT-OBSERVABILITY--add-observer-correlation-export-for-coinvault-web-chat/various/browser-runs/haiku-after-geppetto-fix-20260507-235210/haiku/debug.sqlite
```

The Geppetto/provider sequence now showed:

```text
record 56 provider_routed_event message_delta stop_reason=tool_use
record 57 provider_routed_event message_stop
record 58 geppetto_publish_started final
```

Backend pipeline still had:

```text
ordinal 6 ChatInferenceFinished chat-msg-1:text:1
ordinal 7 ChatToolCallStarted
ordinal 8 ChatInferenceFinished chat-msg-1:text:2
```

This means the first fix was too weak. Emitting `EventFinal` with empty text still causes downstream Pinocchio/CoinVault logic to finalize cached accumulated text as a new segment. For `stop_reason=tool_use`, Claude `message_stop` must emit no final event at all. The lifecycle/tool-turn boundary is already represented by the tool-call event and the next inference turn.

I updated the fix accordingly: `MessageStopType` now returns no events when the stop reason is `tool_use`, and the regression test now expects no final event for that case.
