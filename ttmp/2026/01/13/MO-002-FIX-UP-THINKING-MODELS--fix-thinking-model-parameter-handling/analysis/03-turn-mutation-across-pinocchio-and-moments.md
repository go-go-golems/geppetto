---
Title: Turn Mutation Across Pinocchio and Moments
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
    - Path: moments/backend/pkg/inference/middleware/conversation_compression_middleware.go
      Note: Block/Turn.Data compression and mutation.
    - Path: moments/backend/pkg/webchat/engine.go
      Note: Idempotent system prompt middleware composition.
    - Path: moments/backend/pkg/webchat/loops.go
      Note: ToolCallingLoop modifies turns across iterations.
    - Path: moments/backend/pkg/webchat/moments_global_prompt_middleware.go
      Note: Global prompt injection and metadata tagging.
    - Path: moments/backend/pkg/webchat/ordering_middleware.go
      Note: Block section reorder logic.
    - Path: moments/backend/pkg/webchat/router.go
      Note: Moments webchat request handling and Turn.Data injection.
    - Path: moments/backend/pkg/webchat/system_prompt_middleware.go
      Note: EnsureProfileSystemPromptBlock behavior.
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: Chat seeding and runChat flow.
    - Path: pinocchio/pkg/ui/backend.go
      Note: EngineBackend.reduceHistory and Start flow.
    - Path: pinocchio/pkg/webchat/engine.go
      Note: System prompt middleware wiring.
    - Path: pinocchio/pkg/webchat/router.go
      Note: pinocchio webchat Turn mutation.
ExternalSources: []
Summary: Explain how Turns evolve across inference loops in pinocchio CLI/webchat and moments webchat.
LastUpdated: 2026-01-13T00:00:00Z
WhatFor: Provide a precise map of turn mutation so Responses ordering problems can be traced to the source.
WhenToUse: Use when debugging second-turn failures, duplicated system prompts, or reasoning adjacency issues.
---




# Turn Mutation Across Pinocchio and Moments

## Scope and questions answered

This document answers:

- **Where does `reduceHistory` come from, where is it located, and who uses it?**
- **How are Turns constructed and mutated between inference runs** in:
  - pinocchio CLI chat (bubbletea)
  - pinocchio webchat
  - moments backend webchat

The focus is on Turn mutation and ordering (not UI rendering). Diagrams and pseudocode are included for each flow.

## Quick answers (reduceHistory)

- **Location:** `pinocchio/pkg/ui/backend.go`
- **Symbol:** `func (e *EngineBackend) reduceHistory() *turns.Turn`
- **Used by:** `EngineBackend.Start` (pinocchio CLI chat backend)
- **Purpose:** Flattens all prior Turns into a single Turn by concatenating their Blocks in order.

### reduceHistory pseudocode

```
func reduceHistory(history []*turns.Turn) *turns.Turn {
  out := &turns.Turn{}
  for each turn in history:
    if turn == nil: continue
    AppendBlocks(out, turn.Blocks...)
  return out
}
```

## Common vocabulary

- **Turn**: `geppetto/pkg/turns.Turn` — an ordered list of Blocks plus metadata.
- **Block**: `geppetto/pkg/turns.Block` — typed item (system/user/assistant/tool/reasoning).
- **Mutation**: any change to Blocks or Turn.Data between runs (appending user prompts, tool results, compression, reordering).

## Setup A: Pinocchio CLI chat (bubbletea)

### Core files and symbols

- `pinocchio/pkg/ui/backend.go`
  - `EngineBackend.Start`
  - `EngineBackend.reduceHistory`
  - `EngineBackend.SetSeedTurn`
- `pinocchio/pkg/cmds/cmd.go`
  - `runChat`
  - `buildInitialTurnFromBlocksRendered`

### Turn lifecycle across runs

1) **Seed Turn**
   - Built in `runChat` using `buildInitialTurnFromBlocksRendered`.
   - Seed is appended to backend history via `SetSeedTurn`.

2) **User submits a prompt**
   - `EngineBackend.Start` runs.
   - It calls `reduceHistory()` to flatten all prior Turns into one seed.
   - It appends a new user block to the seed.

3) **Inference runs**
   - `engine.RunInference(ctx, seed)` returns an updated Turn.
   - That updated Turn is appended to history.

4) **Next prompt**
   - Steps repeat, but `reduceHistory()` now concatenates all prior Turns again.

### Pseudocode: EngineBackend.Start

```
seed = reduceHistory(history)
AppendBlock(seed, UserText(prompt))
updated = engine.RunInference(ctx, seed)
history.Append(updated)
return BackendFinished
```

### Diagram: CLI chat inference loop

```
User -> UI -> EngineBackend.Start
   -> reduceHistory(history)
   -> append user block
   -> RunInference(seed)
   -> history += updated Turn
```

### Mutation characteristics

- **State store:** `EngineBackend.history []*turns.Turn`
- **Mutation per run:**
  - new user block appended to a flattened seed
  - updated Turn appended to history
- **Risk:** If each updated Turn already contains full conversation history,
  `reduceHistory()` will duplicate blocks (including system prompts and reasoning).

## Setup B: Pinocchio webchat

### Core files and symbols

- `pinocchio/pkg/webchat/router.go`
  - `handleChatRequest` (implicit; request path in router)
- `pinocchio/pkg/webchat/engine.go`
  - `composeEngineFromSettings` (system prompt middleware)

### Turn lifecycle across runs

1) **Conversation state**
   - Each conversation has `conv.Turn` stored in memory.
   - `conv.Turn` persists across requests.

2) **Incoming user message**
   - Router appends `turns.NewUserTextBlock` to `conv.Turn`.
   - `RunToolCallingLoop` (or direct engine) runs against the same Turn.

3) **Inference updates**
   - The engine returns an updated Turn.
   - `conv.Turn = updatedTurn` (the conversation state is replaced).

### Pseudocode: pinocchio webchat chat request

```
conv = getOrCreateConversation(convID)
if conv.Turn == nil: conv.Turn = new Turn
AppendBlock(conv.Turn, UserText(prompt))
updated = RunToolCallingLoop(ctx, conv.Eng, conv.Turn)
conv.Turn = updated
```

### Diagram: pinocchio webchat

```
POST /chat
  -> append user block to conv.Turn
  -> RunToolCallingLoop(conv.Turn)
  -> conv.Turn = updated
```

### Mutation characteristics

- **State store:** `conv.Turn` (single Turn, cumulative)
- **Mutation per run:**
  - append user block
  - engine may append assistant text, tool calls, tool results
  - updated Turn replaces conv.Turn
- **System prompt insertion:**
  - `composeEngineFromSettings` wraps the engine with `NewSystemPromptMiddleware(sys)`
  - This may insert system blocks on each run depending on middleware behavior.

## Setup C: Moments backend webchat

### Core files and symbols

- `moments/backend/pkg/webchat/router.go`
  - `handleChatRequest` (appends user blocks, sets Turn.Data)
  - `EnsureProfileSystemPromptBlock`
- `moments/backend/pkg/webchat/conversation.go`
  - `Conversation.Turn` state
- `moments/backend/pkg/webchat/loops.go`
  - `ToolCallingLoop`
- `moments/backend/pkg/webchat/engine.go`
  - `composeEngineFromSettings` (idempotent system prompt)
- Turn-mutating middlewares:
  - `moments/backend/pkg/webchat/system_prompt_middleware.go`
  - `moments/backend/pkg/webchat/moments_global_prompt_middleware.go`
  - `moments/backend/pkg/webchat/ordering_middleware.go`
  - `moments/backend/pkg/inference/middleware/conversation_compression_middleware.go`

### Turn lifecycle across runs

1) **Conversation state**
   - `conv.Turn` is initialized in `getOrCreateConv`.
   - It persists across requests.

2) **Incoming user message**
   - `handleChatRequest` appends `NewUserTextBlock` to `conv.Turn`.
   - Additional Turn.Data is injected: page context, timezone, draft bundle, profile slug, user scope.

3) **Prompt injection**
   - `EnsureProfileSystemPromptBlock` is called before the run loop.
   - `composeEngineFromSettings` applies `NewIdempotentSystemPromptMiddleware` (profile prompt).
   - `moments_global_prompt` middleware adds a global system prompt once (idempotent by metadata).

4) **Ordering**
   - `ordering` middleware reorders blocks by section:
     `system -> user_context -> team_context -> conversation -> post_conversation`.

5) **Tool loop**
   - `ToolCallingLoop` calls `eng.RunInference` repeatedly.
   - After each inference, tool calls are executed and tool result blocks appended.
   - `currentTurn` is updated to the new Turn each iteration.

6) **State update**
   - The final `updatedTurn` replaces `conv.Turn`.

### Pseudocode: moments handleChatRequest

```
conv = getOrCreateConversation(convID)
if conv.Turn == nil: conv.Turn = new Turn
Set Turn.Data fields (context, timezone, draft bundle, profile slug)
AppendBlock(conv.Turn, UserText(prompt))
EnsureProfileSystemPromptBlock(conv.Turn, resolvedPrompt)
updated = ToolCallingLoop(ctx, conv.Eng, conv.Turn)
conv.Turn = updated
```

### Pseudocode: moments ToolCallingLoop

```
current = conv.Turn
for i in maxIterations:
  updated = eng.RunInference(ctx, current)
  calls = ExtractPendingToolCalls(updated)
  if no calls: return updated
  results = ExecuteTools(calls)
  AppendToolResultsBlocks(updated, results)
  current = updated
return current
```

### Diagram: moments webchat

```
POST /chat
  -> append user block
  -> ensure profile system prompt
  -> middlewares (global prompt, ordering, compression, ...)
  -> ToolCallingLoop:
       RunInference
       append tool result blocks
       repeat
  -> conv.Turn = updated
```

### Mutation characteristics

- **State store:** `conv.Turn` (single Turn, cumulative)
- **Mutation per run:**
  - append user block
  - Turn.Data enrichment (context, timezone, user scope)
  - system prompt injection (idempotent)
  - ordering middleware reorders blocks
  - optional compression middleware can drop/summarize blocks
  - tool result blocks appended during tool loop

## Comparative summary

| Setup | State Store | Mutation Strategy | Ordering Risks |
| --- | --- | --- | --- |
| pinocchio CLI chat | `history []*Turn` | flatten + append user + append updated Turn | **duplication risk** if updated Turns are cumulative |
| pinocchio webchat | `conv.Turn` | append user + replace Turn | system prompt middleware may add blocks each run |
| moments webchat | `conv.Turn` | append user + middleware mutation + tool loop updates | ordering middleware reorders blocks; compression may drop blocks |

## Implications for Responses API ordering

Responses validation is strict about reasoning adjacency. The above flows create different risks:

- **pinocchio CLI chat:** duplicate reasoning blocks can surface without a valid follower.
- **pinocchio webchat:** repeated system prompt insertion can shift block order around reasoning items.
- **moments webchat:** ordering middleware can move system blocks to the front; compression middleware can drop or summarize blocks, potentially removing required followers if not scoped carefully.

## Diagnostic checklist

- Print block kinds and indices before each `RunInference` call.
- Verify that each `BlockKindReasoning` is immediately followed by `BlockKindLLMText` or `BlockKindToolCall`.
- Track which middlewares mutate `t.Blocks` or `t.Data` (ordering, compression, global prompt).
- For pinocchio CLI chat, verify whether each updated Turn is cumulative; if so, `reduceHistory()` will duplicate content.

## Source references

- `pinocchio/pkg/ui/backend.go` — `reduceHistory`, `EngineBackend.Start`
- `pinocchio/pkg/cmds/cmd.go` — `runChat` seeding
- `pinocchio/pkg/webchat/router.go` — `conv.Turn` updates
- `pinocchio/pkg/webchat/engine.go` — system prompt middleware
- `moments/backend/pkg/webchat/router.go` — request handling and Turn.Data injection
- `moments/backend/pkg/webchat/loops.go` — `ToolCallingLoop`
- `moments/backend/pkg/webchat/system_prompt_middleware.go` — `EnsureProfileSystemPromptBlock`
- `moments/backend/pkg/webchat/moments_global_prompt_middleware.go`
- `moments/backend/pkg/webchat/ordering_middleware.go`
- `moments/backend/pkg/inference/middleware/conversation_compression_middleware.go`
