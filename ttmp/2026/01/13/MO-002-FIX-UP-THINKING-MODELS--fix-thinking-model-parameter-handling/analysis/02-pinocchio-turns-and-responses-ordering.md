---
Title: Pinocchio Turns and Responses Ordering
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
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: RunInference error path and block append order.
    - Path: geppetto/pkg/steps/ai/openai_responses/helpers.go
      Note: Input item ordering and reasoning follower logic.
    - Path: geppetto/pkg/turns/types.go
      Note: Turn/Block definitions and append semantics.
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: Chat seeding and Turn construction in runChat.
    - Path: pinocchio/pkg/ui/backend.go
      Note: EngineBackend history reduction and Start prompt append behavior.
    - Path: pinocchio/pkg/webchat/router.go
      Note: conv.Turn update flow for webchat requests.
ExternalSources: []
Summary: Trace how pinocchio chat/webchat build Turns and how Responses input items must be ordered.
LastUpdated: 2026-01-13T00:00:00Z
WhatFor: Explain block ordering constraints for Responses API and how pinocchio constructs conversation Turns.
WhenToUse: Use when debugging Responses validation errors or hanging chat UI on later turns.
---


# Pinocchio Turns and Responses Ordering

## Goal

This document explains how the pinocchio CLI chat UI and webchat construct `turns.Turn` objects and how those turns are converted into OpenAI Responses API input items. It is written as a reference for ordering and validation rules so we can avoid errors like:

```
Item '<id>' of type 'reasoning' was provided without its required following item.
```

## Glossary

- **Turn**: A container for ordered conversation blocks. In geppetto, this is `turns.Turn`.
- **Block**: A typed unit of content appended to a Turn (system text, user text, assistant output, tool calls, reasoning, etc).
- **Responses input item**: One element of the `input` array in OpenAI Responses API. There are two styles:
  - **Message-style**: `{ role, content }`
  - **Item-style**: `{ type: "reasoning" | "function_call" | "function_call_output" | "message", ... }`

## Core data types (geppetto)

### Turn and Block

Source: `geppetto/pkg/turns/types.go`

- `Turn` is an ordered list of `Block` values.
- `Block.Kind` determines its semantic role.
- `Block.Payload` is a map that carries text, ids, tool args, encrypted reasoning, etc.

Block kinds (selected):
- `BlockKindSystem`: system messages
- `BlockKindUser`: user messages
- `BlockKindLLMText`: assistant output text
- `BlockKindReasoning`: model reasoning (encrypted content)
- `BlockKindToolCall`: assistant tool calls
- `BlockKindToolUse`: tool results

Block helpers (source: `geppetto/pkg/turns/helpers_blocks.go`):
- `turns.NewSystemTextBlock(text)`
- `turns.NewUserTextBlock(text)`
- `turns.NewAssistantTextBlock(text)`
- `turns.NewToolCallBlock(id, name, args)`
- `turns.NewToolUseBlock(id, result)`

Append functions (source: `geppetto/pkg/turns/types.go`):
- `turns.AppendBlock(t, b)` assigns order and appends.
- `turns.AppendBlocks(t, blocks...)` appends multiple in order.

## Pinocchio CLI chat (bubbletea)

### High-level flow

```
User input -> bobatea UI -> EngineBackend.Start
  -> reduceHistory (flatten prior turns)
  -> Append user block
  -> Engine.RunInference (Responses)
  -> events -> UI timeline
```

### Seed construction and chat runtime

- Seed Turn for chat is built in `pinocchio/pkg/cmds/cmd.go`:
  - `buildInitialTurnFromBlocksRendered` / `buildInitialTurnFromBlocks`
  - It appends system prompt, YAML message blocks, and optional user prompt.
- `runChat` sets this seed into the backend using `sess.Backend.SetSeedTurn(seed)`.

Relevant symbols:
- `pinocchio/pkg/cmds/cmd.go:buildInitialTurnFromBlocksRendered`
- `pinocchio/pkg/cmds/cmd.go:runChat` (seed setup)
- `pinocchio/pkg/ui/backend.go:SetSeedTurn`

### EngineBackend behavior

Source: `pinocchio/pkg/ui/backend.go`

Key methods:

1) `SetSeedTurn(t *turns.Turn)`
- Appends the seed Turn into history (`e.history`)
- Emits UI entities for user/assistant text blocks (system/reasoning/tool are skipped)

2) `Start(ctx, prompt)`
- `reduceHistory()` flattens all prior Turns into a single seed Turn
- Appends a new `turns.NewUserTextBlock(prompt)`
- Calls `engine.RunInference(ctx, seed)`
- Appends the updated Turn into history

Pseudocode (simplified):

```
func Start(prompt):
  seed = reduceHistory(history)
  if prompt != "": seed.Append(UserText(prompt))
  updated = RunInference(seed)
  history.Append(updated)
  return BackendFinished
```

### Important implication: history duplication

`reduceHistory()` concatenates blocks from **every prior Turn**. If each Turn is already cumulative (includes the full conversation up to that point), then flattening all turns will duplicate content. The log snippet with two system prompts strongly suggests this duplication is happening.

Consequences:
- System prompts can appear multiple times.
- Earlier reasoning blocks can reappear in the flattened input.
- Ordering around the "latest reasoning block" becomes sensitive to duplicated content.

## Pinocchio webchat

### High-level flow

```
Browser POST /chat -> router appends user block
  -> RunToolCallingLoop(conv.Turn)
  -> Responses engine
  -> events -> WebSocket -> store.js
```

### Turn handling

Source: `pinocchio/pkg/webchat/router.go`

- Each conversation has a `conv.Turn` that is **reused** across turns.
- On each /chat request:
  - `turns.AppendBlock(conv.Turn, turns.NewUserTextBlock(body.Prompt))`
  - `toolhelpers.RunToolCallingLoop(..., conv.Turn, ...)` mutates the same Turn
  - `conv.Turn` is replaced with the updated turn from the engine

### Middleware effects

- Webchat uses `composeEngineFromSettings` which includes `middleware.NewSystemPromptMiddleware(sys)`.
- System prompt middleware may inject a new system block on each run.
- Because the Turn is reused, repeated system blocks can accumulate unless the middleware avoids duplicates.

## Responses API input conversion

### Entry points

- `geppetto/pkg/steps/ai/openai_responses/engine.go:RunInference`
- `geppetto/pkg/steps/ai/openai_responses/helpers.go:buildResponsesRequest`
- `geppetto/pkg/steps/ai/openai_responses/helpers.go:buildInputItemsFromTurn`

### Mapping: Blocks -> Responses input items

Rules as implemented in `buildInputItemsFromTurn`:

- `BlockKindSystem` -> message item `{ role: "system", content: [{ type: "input_text" }] }`
- `BlockKindUser` -> message item `{ role: "user", content: [{ type: "input_text" }] }`
- `BlockKindLLMText` -> message item `{ role: "assistant", content: [{ type: "output_text" }] }`
- `BlockKindToolCall` -> item `{ type: "function_call", call_id, name, arguments }`
- `BlockKindToolUse` -> item `{ type: "function_call_output", call_id, output }`
- `BlockKindReasoning` -> item `{ type: "reasoning", id, encrypted_content, summary: [] }`

### Responses reasoning rule

The Responses API requires a **reasoning item** to be immediately followed by one of:
- `type: "message"` with role `assistant`, or
- `type: "function_call"`

If a reasoning item appears without this follower, the API returns:

```
invalid_request_error: Item '<id>' of type 'reasoning' was provided without its required following item.
```

### How the helper enforces this

`buildInputItemsFromTurn` does extra work for reasoning:

1) Find the latest reasoning block.
2) Emit all prior blocks as normal (skipping older reasoning).
3) If the latest reasoning is followed by assistant text or tool calls, emit:
   - `type: reasoning`, then
   - `type: message` or `type: function_call`
4) Skip duplicated assistant text when it was used as the reasoning follower.
5) Emit remaining blocks after the reasoning group.

Pseudocode (simplified):

```
latest = lastIndex(BlockKindReasoning)
if no reasoning:
  emit all blocks as normal
else:
  emit pre-context
  if next block is assistant text:
    emit reasoning + item message
  else if next block is tool call:
    emit reasoning + tool call(s)
  else:
    omit reasoning
  emit remaining blocks
```

## Sequence diagrams

### CLI chat (pinocchio + Responses)

```
User
  |
  v
bobatea chat UI
  |
  v
EngineBackend.Start
  |
  | reduceHistory + append user block
  v
OpenAI Responses engine (RunInference)
  |
  | buildResponsesRequest -> buildInputItemsFromTurn
  v
/v1/responses
  |
  v
SSE events -> EventPartialCompletion / EventThinkingPartial
  |
  v
UI timeline
```

### Webchat (browser)

```
Browser POST /chat
  |
  v
router.go (append user block to conv.Turn)
  |
  v
RunToolCallingLoop(conv.Turn)
  |
  v
OpenAI Responses engine
  |
  v
Event Router -> webchat forwarder -> WebSocket
  |
  v
web/src/store.js (llm.* + llm.thinking.*)
```

## Why the second prompt can hang at "Generating..."

Observed behavior:
- `EventPartialCompletionStart` is emitted and UI creates an assistant entity.
- The Responses API returns HTTP 400 for invalid input (reasoning without follower).
- The UI remains in "Generating..." because it never receives `EventFinal` or `EventError`.

Relevant code:
- `pinocchio/pkg/ui/backend.go:Start` logs the error and returns `BackendFinishedMsg`.
- `geppetto/pkg/steps/ai/openai_responses/engine.go` returns the error after HTTP 400 **without publishing** an `EventError` in the streaming path.

So the UI gets a start event but no terminal event to end the stream.

## Ordering guidance for Responses API

To avoid validation errors:

1) Ensure every reasoning block is immediately followed in the Turn by:
   - an assistant text block (`BlockKindLLMText`), or
   - a tool call block (`BlockKindToolCall`).

2) Do not append new user/system blocks directly after a reasoning block.

3) Tool call blocks must include valid `call_id` and `name` so they serialize into `function_call` items.

4) Avoid duplicate history merges that can reorder or repeat reasoning blocks.

## Turn construction gotchas (pinocchio)

### CLI chat

- `EngineBackend` stores **every updated Turn** in `history`, then flattens all of them.
- If each updated Turn already contains full history, flattening duplicates blocks.
- Duplicate reasoning blocks increase the chance that the "latest reasoning" block is not followed by an assistant/tool item.

### Webchat

- `conv.Turn` is reused and grows over time; system prompt middleware can keep adding system blocks.
- Repeated system blocks are allowed by the API but make debugging ordering harder.

## Suggested diagnostics

- Inspect the flattened Turn (before conversion):
  - `engine.DebugTap` in `openai_responses/engine.go` calls `tap.OnTurnBeforeConversion`.
- Log the sequence of block kinds to verify adjacency around reasoning.
- For debugging, search for `BlockKindReasoning` and print the next block kind.

Example check (pseudocode):

```
for i, b := range t.Blocks:
  if b.Kind == Reasoning:
    next = t.Blocks[i+1] if exists
    assert next.Kind in {LLMText, ToolCall}
```

## References (code symbols)

- `pinocchio/pkg/ui/backend.go`
  - `EngineBackend.Start`
  - `EngineBackend.reduceHistory`
  - `EngineBackend.SetSeedTurn`
- `pinocchio/pkg/cmds/cmd.go`
  - `runChat`
  - `buildInitialTurnFromBlocksRendered`
- `pinocchio/pkg/webchat/router.go`
  - conversation `conv.Turn` update + `RunToolCallingLoop`
- `geppetto/pkg/steps/ai/openai_responses/helpers.go`
  - `buildInputItemsFromTurn`
- `geppetto/pkg/steps/ai/openai_responses/engine.go`
  - `RunInference` (error path and block appends)
- `geppetto/pkg/turns/types.go`
  - `Turn`, `Block`, `AppendBlock`
