---
Title: Pinocchio Unification Progress and Architecture
Ticket: MO-003-UNIFY-INFERENCE
Status: active
Topics:
    - inference
    - architecture
    - webchat
    - prompts
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/inference/runner/runner.go
      Note: Shared runner architecture described in this doc
    - Path: pinocchio/pkg/ui/backend.go
      Note: TUI backend runner integration
    - Path: pinocchio/pkg/webchat/router.go
      Note: Webchat runner integration and tool loop wiring
ExternalSources: []
Summary: Explains the shared runner architecture, TUI/webchat integration, and how the unified inference flow now works end-to-end.
LastUpdated: 2026-01-16T15:29:00-05:00
WhatFor: Understand what was built so far for pinocchio unification and how the pieces fit together.
WhenToUse: When onboarding to the unified inference runner or planning follow-on work.
---




# Pinocchio Unification Progress and Architecture

## Goal

Describe the unification work completed so far in pinocchio, explain how the shared runner is built, and detail how TUI and webchat now flow through the same inference orchestration. This document is meant to be a “single source of truth” for how the new pieces fit together.

## What we changed (summary)

- Introduced a **shared inference runner** in `pinocchio/pkg/inference/runner` that owns:
  - snapshot creation from `ConversationState`,
  - optional tool calling loop execution,
  - event sink wiring + snapshot hook wiring, and
  - canonical conversation state updates.
- Migrated **TUI backend** to the shared runner.
- Migrated **webchat run loop** to the shared runner and removed redundant snapshot/update helpers.

## Key components and where they live

### 1) Shared runner

File: `pinocchio/pkg/inference/runner/runner.go`

The runner defines two public helper functions and a structured options object:

- `Run(ctx, eng, state, prompt, opts)` — performs snapshot + inference (+ tools) + state update.
- `SnapshotForPrompt(state, prompt)` — builds a snapshot `Turn` from `ConversationState` with a user prompt appended.
- `UpdateStateFromTurn(state, turn, opts)` — persists the updated turn into `ConversationState` (optionally filtering blocks).

#### Runner options

```
type RunOptions struct {
  ToolRegistry geptools.ToolRegistry
  ToolConfig   *toolhelpers.ToolConfig
  SnapshotHook toolhelpers.SnapshotHook
  EventSinks   []events.EventSink
  Update       UpdateOptions
}

type UpdateOptions struct {
  FilterBlocks func([]turns.Block) []turns.Block
}
```

This structure lets both TUI and webchat pass their unique concerns (tools, sinks, hooks, system-prompt filtering) without duplicating orchestration logic.

### 2) Conversation state

File: `geppetto/pkg/conversation/state.go`

Pinocchio relies on `ConversationState` as the canonical multi-turn container. The runner takes a pointer to this state and ensures it is updated after inference, so the next prompt is always grounded in the latest block sequence.

### 3) TUI backend integration

File: `pinocchio/pkg/ui/backend.go`

The TUI now calls the runner directly:

```
updated, err := runner.Run(ctx, engine, &e.state, prompt, runner.RunOptions{})
```

This replaces TUI-specific snapshot/update logic and guarantees the same snapshot + update behavior as webchat.

### 4) Webchat integration

Files:
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/conversation.go`

The webchat run loop now uses the shared runner and passes:

- tool registry for tool-calling,
- tool config (max iterations / timeout),
- event sinks (Watermill sink for UI streaming),
- snapshot hook (for Turn YAML dumps), and
- block filtering (to exclude system prompt middleware blocks when persisting state).

The old helpers (`snapshotForPrompt`, `updateStateFromTurn`) were removed from `conversation.go` because the runner now provides that logic.

## How the unified flow works end-to-end

### A) TUI flow (high-level)

```
User prompt
  -> EngineBackend.Start
  -> runner.Run(ctx, engine, &state, prompt, opts)
     -> SnapshotForPrompt
     -> engine.RunInference
     -> UpdateStateFromTurn
  -> UI events stream via Watermill (sink configured at engine creation)
```

### B) Webchat flow (high-level)

```
HTTP /chat
  -> Router builds tool registry
  -> runCtx attaches EventSink + SnapshotHook
  -> runner.Run(ctx, engine, &conv.State, prompt, opts)
     -> SnapshotForPrompt
     -> toolhelpers.RunToolCallingLoop (if tool registry provided)
     -> UpdateStateFromTurn (with system prompt filtering)
  -> Websocket UI receives Watermill events from the sink
```

## Event sinks and streaming

Both TUI and webchat rely on Watermill-based event sinks for UI streaming:

- `middleware.NewWatermillSink` publishes `events.Event` objects as JSON to a topic.
- `events.WithEventSinks` attaches the sink to the run context, so the engine and loop can publish inference events during execution.

This means inference events are emitted consistently regardless of frontend.

## Why this design matters

- **Single orchestration path**: snapshot → inference → update is identical for TUI + webchat.
- **Reduced divergence**: tool loops, snapshot hooks, and state updates are no longer duplicated.
- **Future-ready**: Moments can be migrated by routing its loop through the same runner options.

## What still remains (short list)

- Centralize tool registry creation + overrides in a shared helper (currently still built in webchat).
- Consolidate event sink wiring (single builder for TUI and webchat).
- Document a Moments migration plan that maps its tool loop to runner options.

## Appendix: Annotated pseudocode

### Runner core

```
Run(ctx, engine, state, prompt, opts):
  seed = SnapshotForPrompt(state, prompt)
  ctx = attach_event_sinks(ctx, opts.EventSinks)
  ctx = attach_snapshot_hook(ctx, opts.SnapshotHook)
  if opts.ToolRegistry:
     updated = RunToolCallingLoop(ctx, engine, seed, opts.ToolRegistry, toolcfg)
  else:
     updated = engine.RunInference(ctx, seed)
  UpdateStateFromTurn(state, updated, opts.Update)
  return updated
```

### Webchat usage

```
registry = buildToolRegistry(...)
opts = RunOptions{
  ToolRegistry: registry,
  ToolConfig: cfg,
  SnapshotHook: hook,
  EventSinks: []EventSink{conv.Sink},
  Update: { FilterBlocks: filterSystemPromptBlocks },
}
runner.Run(runCtx, conv.Eng, &conv.State, prompt, opts)
```
