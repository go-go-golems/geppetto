---
Title: Unified Conversation Handling Across Repos
Ticket: MO-002-FIX-UP-THINKING-MODELS
Status: active
Topics:
    - bug
    - geppetto
    - go
    - inference
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/turns/types.go
      Note: Turn/Block semantics used by conversation state.
    - Path: moments/backend/pkg/inference/middleware/conversation_compression_middleware.go
      Note: Block/Turn.Data compression mutation.
    - Path: moments/backend/pkg/webchat/loops.go
      Note: ToolCallingLoop turn updates.
    - Path: moments/backend/pkg/webchat/ordering_middleware.go
      Note: Block section reordering rules.
    - Path: moments/backend/pkg/webchat/router.go
      Note: Moments webchat request flow and Turn.Data injection.
    - Path: moments/backend/pkg/webchat/system_prompt_middleware.go
      Note: Idempotent profile prompt insertion.
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: Chat seeding for CLI.
    - Path: pinocchio/pkg/ui/backend.go
      Note: reduceHistory origin and CLI chat mutation flow.
    - Path: pinocchio/pkg/webchat/engine.go
      Note: System prompt middleware composition.
    - Path: pinocchio/pkg/webchat/router.go
      Note: pinocchio webchat turn mutation.
ExternalSources: []
Summary: Proposed unified conversation state and mutation pipeline for geppetto, pinocchio, and moments.
LastUpdated: 2026-01-13T00:00:00Z
WhatFor: Standardize multi-turn handling and prevent Responses ordering errors across repos.
WhenToUse: Use when implementing or refactoring chat flows or tool loops.
---


# Unified Conversation Handling Across Repos

## Executive Summary

`reduceHistory` in pinocchio CLI chat is a bug because it flattens multiple cumulative Turns and duplicates blocks, which violates strict ordering constraints in the OpenAI Responses API. Meanwhile, pinocchio webchat and moments webchat mutate a single cumulative Turn, but they still suffer from inconsistent prompt injection, ordering, and compression logic across repos.

This design proposes a shared, conversation-centric runtime that unifies turn mutation semantics across geppetto, pinocchio, and moments. The core idea is to maintain a single **canonical ConversationState** that yields **stable, validated Turn snapshots** for inference and to enforce ordering invariants (including reasoning adjacency) in one place.

## Problem Statement

We currently have three different multi-turn strategies:

- **Pinocchio CLI chat:** `EngineBackend.reduceHistory` flattens *all prior* Turns. If each Turn is cumulative, this duplicates blocks (system prompts, reasoning blocks, tool calls). Duplicated reasoning blocks can be re-serialized without required followers, which triggers Responses API 400s and leaves the UI streaming forever.
- **Pinocchio webchat:** A single `conv.Turn` is mutated per request. System prompt middleware can insert blocks repeatedly depending on middleware behavior.
- **Moments webchat:** A single `conv.Turn` is mutated, but middlewares reorder and compress blocks; prompt injection is partially idempotent, and ordering rules are different from pinocchio.

Symptoms include:

- `reasoning` items in Responses input with no valid follower.
- duplicated system prompts and repeated blocks across turns.
- hanging UI because a streaming start event is emitted but no final/error event arrives.

We need a unified, conversation-oriented abstraction so each repository handles turns identically and enforces the same invariants.

## Proposed Solution

Introduce a shared **ConversationState** API in geppetto and adopt it in pinocchio and moments. The state owns the canonical block list and produces Turn snapshots for inference. The API enforces ordering constraints and performs deduplication and validation.

### Key goals

- **Single canonical block list** per conversation.
- **Strict invariants** for Responses ordering (reasoning adjacency, tool call pairing).
- **Idempotent system prompt injection** with metadata keys and stable ordering.
- **Unified mutation steps** for user input, tool results, and assistant output.
- **Deterministic snapshotting** of inference Turn data.
- **Replace ad-hoc flattening** (remove `reduceHistory`).

## Design Decisions

### Decision 1: Introduce `conversation` package in geppetto

**Rationale:** All three repos already share geppetto. A core package makes behavior consistent and testable.

**Proposed package:** `geppetto/pkg/conversation`

### Decision 2: Make `ConversationState` the source of truth

**Rationale:** We must prevent duplication and enforce invariants centrally. Mutating raw Turns in multiple places leads to divergence.

### Decision 3: Support snapshot modes explicitly

**Rationale:** Some engines expect full history; others can accept deltas. Explicit modes avoid accidental duplication.

### Decision 4: Enforce ordering invariants at snapshot time

**Rationale:** Responses API validation is stricter than legacy chat completion. The snapshot builder must guarantee adjacency rules.

## Proposed API (geppetto/pkg/conversation)

### Types

```
type ConversationState struct {
  ID        string
  RunID     string
  Blocks    []turns.Block
  Data      map[turns.TurnDataKey]any
  Metadata  map[turns.TurnMetadataKey]any
  Version   int64
}

// SnapshotConfig controls how a Turn snapshot is produced.
type SnapshotConfig struct {
  IncludeSystemPrompts bool
  IncludeToolBlocks    bool
  IncludeReasoning     bool
  NormalizeOrdering    bool
  EnforceResponsesAdj  bool
}

// Mutation represents a deterministic change to the conversation.
type Mutation interface {
  Apply(cs *ConversationState) error
  Name() string
}
```

### Core operations

```
func NewConversationState(runID string) *ConversationState

func (cs *ConversationState) Apply(m Mutation) error
func (cs *ConversationState) ApplyAll(muts ...Mutation) error

// Snapshot returns a Turn for inference based on the canonical blocks.
func (cs *ConversationState) Snapshot(cfg SnapshotConfig) (*turns.Turn, error)

// Validate runs invariants (reasoning adjacency, tool call pairing, etc).
func (cs *ConversationState) Validate(cfg SnapshotConfig) error
```

### Standard mutations

- `MutateEnsureSystemPrompt(prompt string, key string)`
- `MutateAppendUserText(text string)`
- `MutateAppendAssistantText(text string)`
- `MutateAppendToolCall(id, name, args)`
- `MutateAppendToolResult(id, result)`
- `MutateApplyCompression(opts)`
- `MutateApplyOrdering(sections)`

### Snapshot rules (Responses-specific)

- If `IncludeReasoning` is true, enforce: reasoning block must be immediately followed by assistant text or tool call block.
- If the rule is violated, either:
  - drop reasoning blocks (with log + metadata), or
  - return a validation error and require the caller to correct.

## New Standard Flow

### Pinocchio CLI chat

Replace history flattening with a per-session `ConversationState`.

**Before** (buggy):

```
seed = reduceHistory(history)
seed.Append(UserText(prompt))
updated = RunInference(seed)
history.Append(updated)
```

**After** (unified):

```
cs.Apply(MutateAppendUserText(prompt))
turn = cs.Snapshot(ResponsesSafeConfig)
updated = RunInference(turn)
cs.Apply(MutateAppendAssistantText(updated.Output))
cs.Apply(MutateAppendToolBlocks(updated.ToolBlocks))
```

### Pinocchio webchat

`conv.Turn` is replaced with `conv.ConversationState` and all mutations go through it.

```
conv.State.Apply(MutateAppendUserText(prompt))
turn = conv.State.Snapshot(ResponsesSafeConfig)
updated = RunToolCallingLoop(turn)
conv.State.Apply(MutateMergeTurn(updated))
```

### Moments webchat

Replace ad-hoc `conv.Turn` mutation with a shared `ConversationState`.

Ordering and compression become **Mutations** rather than middleware that directly mutates raw Turns.

```
conv.State.Apply(MutateEnsureSystemPrompt(profilePrompt))
conv.State.Apply(MutateEnsureGlobalPrompt(globalPrompt))
conv.State.Apply(MutateAppendUserText(prompt))
conv.State.Apply(MutateApplyOrdering(sectionRules))
conv.State.Apply(MutateApplyCompression(opts))
turn = conv.State.Snapshot(ResponsesSafeConfig)
updated = ToolCallingLoop(turn)
conv.State.Apply(MutateMergeTurn(updated))
```

## Sequence Diagrams

### Unified inference loop

```
User input
  -> ConversationState.Apply(AppendUserText)
  -> Snapshot(ResponsesSafe)
  -> Engine.RunInference
  -> ConversationState.Apply(MergeTurn)
```

### Responses-safe snapshot

```
Blocks (canonical)
  -> normalize ordering
  -> enforce reasoning adjacency
  -> emit Turn snapshot
```

## Migration Plan

### Phase 1: Implement conversation package in geppetto

- Add `geppetto/pkg/conversation` with `ConversationState`, mutations, and snapshot logic.
- Add unit tests for reasoning adjacency and tool call pairing.

### Phase 2: Pinocchio CLI chat

- Replace `EngineBackend.history` + `reduceHistory` with `ConversationState`.
- Update UI seeding to read from `ConversationState.Blocks`.
- Ensure event metadata carries RunID/TurnID consistently.

### Phase 3: Pinocchio webchat

- Replace `conv.Turn` with `conv.State`.
- Migrate middleware behavior to explicit mutations (system prompt injection, tool result reorder).

### Phase 4: Moments webchat

- Replace `conv.Turn` with `conv.State`.
- Convert ordering and compression into mutations or snapshot config.
- Preserve idempotent system/global prompt behavior via mutation metadata keys.

## Alternatives Considered

### Alternative A: Keep per-repo logic and just fix reduceHistory

- Pros: smaller change.
- Cons: does not unify ordering rules, leaves webchat stacks divergent.

### Alternative B: Only update Responses input builder to tolerate bad ordering

- Pros: localized change.
- Cons: hides structural bugs and still risks tool/reasoning pairing issues.

### Alternative C: Use append-only Turn deltas everywhere

- Pros: no duplication.
- Cons: requires each engine to accept deltas and rebuild history internally; more invasive across engines.

## Open Questions

- Should snapshot builder drop reasoning blocks on adjacency violations or hard error?
- Do we need per-provider snapshots (Responses vs ChatCompletions) or a single unified snapshot with optional flags?
- How should tool-call grouping interact with ordering and compression mutations?

## API References and File Pointers

- Pinocchio CLI:
  - `pinocchio/pkg/ui/backend.go` (`EngineBackend.Start`, `reduceHistory`)
  - `pinocchio/pkg/cmds/cmd.go` (`runChat` seed)
- Pinocchio webchat:
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/engine.go`
- Moments webchat:
  - `moments/backend/pkg/webchat/router.go`
  - `moments/backend/pkg/webchat/loops.go`
  - `moments/backend/pkg/webchat/system_prompt_middleware.go`
  - `moments/backend/pkg/webchat/moments_global_prompt_middleware.go`
  - `moments/backend/pkg/webchat/ordering_middleware.go`
  - `moments/backend/pkg/inference/middleware/conversation_compression_middleware.go`

## Implementation Plan (Detailed Steps)

1) **Create `geppetto/pkg/conversation`**
   - Define `ConversationState` and snapshot logic.
   - Add unit tests for Responses adjacency rules.

2) **Pinocchio CLI migration**
   - Remove `reduceHistory` usage; keep for compatibility only if needed.
   - Store a single `ConversationState` in `EngineBackend`.

3) **Pinocchio webchat migration**
   - Replace `conv.Turn` with `conv.State`.
   - Convert system prompt injection to idempotent mutation.

4) **Moments webchat migration**
   - Replace `conv.Turn` with `conv.State`.
   - Move ordering/compression to explicit mutations or snapshot config.

5) **Integration tests**
   - Multi-turn Responses test with reasoning blocks.
   - Tool loop with reasoning + tool call adjacency.

## Non-Goals

- Rewriting the entire inference engine interface.
- Changing tool execution semantics.
- Adding new UI behavior beyond correct conversation ordering.
