---
Title: Unified Conversation Handling Across Frontends
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
    - Path: ../../../../../../../moments/backend/pkg/promptutil/resolve.go
      Note: |-
        Prompt resolver behavior and Turn.Data slug prefix semantics.
        Prompt resolver semantics used by moments.
    - Path: ../../../../../../../moments/backend/pkg/webchat/conversation.go
      Note: Moments per-conversation Turn, identity session, and engine rebuild logic.
    - Path: ../../../../../../../moments/backend/pkg/webchat/engine.go
      Note: |-
        Moments engine composition with middleware registry and system prompt.
        Moments engine composition with middleware registry.
    - Path: ../../../../../../../moments/backend/pkg/webchat/loops.go
      Note: |-
        Moments tool calling loop with identity-aware executor and step controller.
        Moments tool loop with identity-aware executor.
    - Path: ../../../../../../../moments/backend/pkg/webchat/router.go
      Note: |-
        Moments webchat router, prompt resolver integration, profile handling.
        Moments router profile/prompt resolution and run orchestration.
    - Path: ../../../../../../../pinocchio/pkg/ui/backend.go
      Note: |-
        TUI EngineBackend with ConversationState snapshot/run.
        Pinocchio TUI engine backend and ConversationState snapshotting.
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
      Note: |-
        ConversationState-backed webchat state and snapshotting.
        Pinocchio webchat ConversationState storage.
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: |-
        Webchat router with engine composition, profiles, and tool loop integration.
        Pinocchio webchat router and engine composition.
    - Path: pkg/conversation/state.go
      Note: ConversationState snapshot and validation used by pinocchio.
    - Path: pkg/inference/toolhelpers/helpers.go
      Note: Shared tool calling loop used by pinocchio TUI/webchat.
ExternalSources: []
Summary: Textbook-style analysis of current conversation handling across pinocchio TUI, pinocchio webchat, and moments webchat, plus designs to unify inference orchestration.
LastUpdated: 2026-01-16T10:10:00-05:00
WhatFor: Guide a refactor that centralizes inference orchestration and makes UI transports downstream consumers.
WhenToUse: Use when designing a shared conversation runner or refactoring webchat/TUI pipelines.
---


# Unified Conversation Handling Across Frontends

## Goal

Unify how conversations, tools, middleware, and inference loops are constructed across:

- **pinocchio TUI** (BubbleTea)
- **pinocchio webchat** (WS + REST)
- **moments webchat** (WS + REST + identity + prompt resolver)

The end state is a **central orchestration layer** that spins up the inference loop with the correct model, middlewares, tools, and prompt resolution, while TUI and web transports become thin downstream consumers (event sinks + UI rendering).

## Scope

This document focuses on **conversation lifecycle**, **engine composition**, **middleware/tool registration**, **prompt resolution**, **event routing**, and **tool calling loops**. It intentionally de-emphasizes UI rendering details except where they influence ordering or event semantics.

## Current State (System-by-System)

### 1) pinocchio TUI (BubbleTea)

**Where it lives**

- `pinocchio/pkg/ui/backend.go` (`EngineBackend`)

**How it works**

- Maintains a `conversation.ConversationState` and snapshots a `turns.Turn` per user prompt.
- Invokes `engine.RunInference` directly (no custom tool loop; it expects the engine/middleware to handle tools or external tooling).
- Updates `ConversationState` from the returned Turn.
- Emits UI entities (timeline items) from the Turn for initial seeding.

**Key characteristics**

- **Conversation state**: explicit, in-memory, `ConversationState` based.
- **Engine orchestration**: in the UI backend, not a shared service.
- **Tool calling**: whatever the engine/middlewares implement; no shared tool loop here.
- **Event routing**: uses `events.EventSink` in engine; the UI reacts to events via the runtime backend.

**Simplified flow**

```
User prompt
  -> ConversationState.Snapshot + append user block
  -> Engine.RunInference
  -> Update ConversationState from returned Turn
  -> Emit timeline events / UI entities
```

### 2) pinocchio webchat

**Where it lives**

- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/conversation.go`
- `geppetto/pkg/inference/toolhelpers/helpers.go` (tool loop)

**How it works**

- Websocket connection binds to a conversation; `/chat` POST triggers a run.
- Engine is composed per profile using `composeEngineFromSettings` with middleware factories.
- Conversation state stored as `ConversationState` (not a raw Turn).
- For each prompt:
  - Snapshot Turn from ConversationState
  - Run `toolhelpers.RunToolCallingLoop` (shared tool loop) to handle tool calls
  - Update ConversationState from the returned Turn

**Key characteristics**

- **Conversation state**: `ConversationState` with snapshot and filter of system prompt blocks.
- **Engine orchestration**: in webchat router; engines are composed per profile using middleware factories.
- **Tool calling**: shared geppetto `RunToolCallingLoop` (tool execution from registry).
- **Event routing**: `WatermillSink` for events per conversation and WS broadcast via forwarder.

**Simplified flow**

```
HTTP /chat
  -> ConversationState snapshot + append user
  -> RunToolCallingLoop (engine inference + tool execution)
  -> Update ConversationState
  -> Emit events to WS clients
```

### 3) moments webchat

**Where it lives**

- `moments/backend/pkg/webchat/router.go`
- `moments/backend/pkg/webchat/engine.go`
- `moments/backend/pkg/webchat/loops.go`
- `moments/backend/pkg/webchat/conversation.go`
- `moments/backend/pkg/promptutil/resolve.go`
- `moments/backend/pkg/app/app.go` (registries + prompt resolver wiring)

**How it works**

- `App.InitWebChat` constructs the router using **registries** (middleware/tool/profile registries) and a **PromptResolver**.
- Conversation object stores a raw `*turns.Turn` and identity session details.
- Engine composition uses `composeEngineFromSettings` with registry-backed middlewares.
- Prompt resolution uses `PromptResolver` and `Turn.Data[turnkeys.PromptSlugPrefix]`.
- Tool calling loop is custom: identity-aware executor, step mode, explicit tool result events.

**Key characteristics**

- **Conversation state**: raw Turn, not ConversationState.
- **Engine orchestration**: router + `App` registry wiring; middlewares come from registries.
- **Tool calling**: custom loop with identity-aware executor and step controller.
- **Prompt resolution**: external `PromptResolver` uses Turn.Data and request context (draft bundle, org/person IDs).
- **Event routing**: `events.EventRouter` (redis or in-memory), sink wrappers for SEM extraction and widget events.

**Simplified flow**

```
HTTP /chat or WS
  -> resolve profile, prompt, tool/mw registries
  -> compose engine with middlewares
  -> run ToolCallingLoop (identity-aware executor)
  -> update conv.Turn
  -> publish structured events (SEM)
```

## Summary of Divergence

| Aspect | pinocchio TUI | pinocchio webchat | moments webchat |
|---|---|---|---|
| Conversation state | `ConversationState` | `ConversationState` | raw `*turns.Turn` |
| Engine composition | caller-owned | router-owned | router + App registries |
| Middlewares | factories in router | factories in router | registry snapshots from App |
| Tools | tool registry (per run) | tool registry (per run) | registry injected, identity-aware executor |
| Prompt resolution | system prompt string only | system prompt string only | `PromptResolver` + `Turn.Data` |
| Tool loop | none (engine direct) | shared `RunToolCallingLoop` | custom loop with step controller |
| Events | UI backend events | Watermill sink | SEM pipeline + sink wrappers |

## Why Unify?

- **Consistency**: same conversation semantics and tool calling across UI channels.
- **Correctness**: reasoning/tool ordering issues should be solved in one place (as with the Responses reasoning follower fix).
- **Maintainability**: avoid three different orchestration pipelines.
- **Extensibility**: add a new frontend (Slack, CLI, widget) without rewriting inference glue.

## Design Goals

- **Single orchestration API** for conversation runs.
- **Pluggable middleware/tool registries** without requiring UI-level wiring.
- **Prompt resolution hooks** (e.g., moments’ `PromptResolver`) as optional injection.
- **Strict ordering invariants** handled centrally (tool/assistant reasoning adjacency, system prompt insertion).
- **Transport-agnostic event sinks** (WebSocket, TUI, logging) as downstream subscribers.

## Proposed Unification Architecture

### Core Concept: Conversation Orchestrator

A central component that:

- Owns **ConversationState**
- Builds an **Engine** from a **Profile** (model, middlewares, tool config)
- Injects **PromptResolver** if present
- Runs **ToolCallingLoop** via a common interface
- Emits **events** through injected sinks

#### High-level API (pseudocode)

```go
// Inputs are transport-agnostic

type ConversationRunner interface {
  Run(ctx context.Context, req RunRequest) (*RunResult, error)
}

type RunRequest struct {
  ConversationID string
  Prompt string
  ProfileSlug string
  Overrides map[string]any
  Context map[string]any // identity/session, UI metadata, timezone, draft bundle
  ToolsOverride []ToolDefinition
}

type RunResult struct {
  Turn *turns.Turn
  Events []events.Event // optional; usually emitted via sink
}
```

#### Orchestrator responsibilities

- Snapshot and validate conversation state
- Resolve prompts (system + middleware prompts) via injected resolver
- Compose engine with middlewares and tool registry
- Run a **common tool loop** that allows identity-aware executor injection
- Update conversation state

### Key Building Blocks

#### 1) ConversationState (shared)

- Use `geppetto/pkg/conversation` as the canonical state container.
- Moments should migrate from `conv.Turn` to `ConversationState` to inherit ordering validation.

#### 2) Profile Registry

- Move profile definitions into a shared registry or descriptor struct:
  - model
  - default prompt
  - middleware list (with config)
  - tool policy (allowed tools)
  - prompt slug prefix (moments)

#### 3) Middleware Registry

- Use registry-based middleware descriptors everywhere:
  - pinocchio currently uses `map[string]MiddlewareFactory`
  - moments already uses registry snapshots

Unify via a shared `MiddlewareRegistry` interface:

```go
type MiddlewareRegistry interface {
  Get(name string) (MiddlewareDescriptor, bool)
  List() []MiddlewareDescriptor
}
```

#### 4) Tool Registry and Executor

- Pinocchio uses `geptools.InMemoryToolRegistry` + `toolhelpers.RunToolCallingLoop`.
- Moments uses identity-aware executor (auth, step mode).

Unify by allowing `ToolExecutor` injection in the tool loop:

```go
type ToolExecutor interface {
  Execute(ctx context.Context, calls []tools.ToolCall, reg tools.ToolRegistry) ([]*tools.ToolResult, error)
}
```

#### 5) Prompt Resolver Hook

- Provide optional `PromptResolver` injection:
  - if present, it resolves system/middleware prompts based on Turn.Data context
  - if absent, fallback to static prompt strings

The orchestrator should accept a `PromptResolver` and `ResolveOptions` derived from request context (draft bundle, person/org).

#### 6) Event Sink Pipeline

- The orchestrator should emit events via an injected sink interface:

```go
type EventSink interface {
  PublishEvent(events.Event) error
}
```

- The transport layer (WS, TUI) attaches sinks as needed.

## Detailed Unified Flow

```
Client (TUI/Web) submits prompt
  -> Transport Handler (WS or HTTP)
     -> Conversation Orchestrator
        1) Load ConversationState
        2) Apply context: identity session, prompt slug prefix, draft bundle
        3) Resolve system prompt (PromptResolver, optional)
        4) Compose engine with middlewares/tool policy
        5) Run ToolCallingLoop (with injected ToolExecutor)
        6) Update ConversationState
        7) Emit events to sinks
  <- Transport returns run_id/conv_id
```

## Textbook-Style Sequence Diagrams

### A) TUI (pinocchio) — unified

```
User -> TUI: prompt
TUI -> Orchestrator: Run(prompt, profile)
Orchestrator -> ConversationState: Snapshot + append user
Orchestrator -> EngineFactory: compose engine
Orchestrator -> ToolLoop: run inference + tools
ToolLoop -> Engine: RunInference
ToolLoop -> ToolExecutor: ExecuteTools
Orchestrator -> ConversationState: Update from Turn
Orchestrator -> EventSink(TUI): publish events
TUI -> User: render events
```

### B) Webchat (pinocchio/moments) — unified

```
Browser -> /chat: POST prompt
Router -> Orchestrator: Run(prompt, profile, context)
Orchestrator -> PromptResolver?: resolve system + middleware prompts
Orchestrator -> EngineFactory: compose
Orchestrator -> ToolLoop: run inference + tools
ToolLoop -> EventSink(WS): publish events
WS -> Browser: stream events
```

## Proposed Design Options

### Option A: Shared `ConversationRunner` in geppetto

- Add a new package: `geppetto/pkg/inference/runner`.
- Implement a generic runner with hooks:
  - PromptResolver
  - MiddlewareRegistry
  - ToolRegistry + ToolExecutor
  - EventSink

**Pros**
- Single source of truth for conversation semantics.
- Minimal duplication between pinocchio and moments.

**Cons**
- Requires moments to adopt geppetto runner assumptions (may be sizable refactor).

### Option B: “Engine Orchestrator” in each app, but common core helpers

- Extract common building blocks into geppetto:
  - ConversationState utilities
  - Unified tool loop interface
  - Prompt resolution adapter hooks
- Keep app-specific orchestration logic in moments/pinocchio.

**Pros**
- Lower risk for moments (keeps their custom features).

**Cons**
- Still leaves some duplication in router flows.

### Option C: “Conversation Service” abstraction in moments, reused in pinocchio

- Build a dedicated service in moments that exposes:
  - `RunConversation` API
  - `AttachEventSink` and `AttachTransport`
- Port pinocchio to use that service (or share code).

**Pros**
- Matches moments’ current service-centered architecture.

**Cons**
- More coupling to moments codebase.

## Recommended Design

**Option A** (Shared `ConversationRunner` in geppetto) with adapters:

- `PromptResolverAdapter` for moments (wraps `prompts.Resolver` + `ResolveOptions`).
- `ToolExecutorAdapter` for moments (identity-aware executor).
- `ProfileAdapter` for pinocchio (profiles from CLI/webchat).

This yields a true single orchestration path, and supports both tool loops and prompt resolution patterns.

## Migration Plan

1) **Extract orchestrator interface in geppetto**
   - Implement `ConversationRunner` + base tool loop.

2) **Migrate pinocchio TUI to runner**
   - Replace `EngineBackend.Start` with `runner.Run(...)`.

3) **Migrate pinocchio webchat to runner**
   - Replace router-internal engine composition + tool loop.

4) **Introduce runner adapter in moments**
   - Replace `ToolCallingLoop` / `conv.Turn` with runner.
   - Migrate to `ConversationState`.
   - Keep prompt resolver hook and identity tool executor.

## Risks and Mitigations

- **Prompt resolver ordering**: moments relies on Turn.Data for scope; ensure orchestrator writes `turnkeys.PromptSlugPrefix` and identity fields before resolution.
- **Identity session context**: moments tool executor requires identity session; runner must support request-scoped context injection.
- **Event schema differences**: maintain SEM event mapping as a downstream concern.
- **Backward compatibility**: keep existing endpoints stable; only change internal orchestration.

## Appendix: Suggested Interfaces

```go
// Prompt resolver hook

type PromptResolver interface {
  Resolve(ctx context.Context, t *turns.Turn, slugSuffix string, opts any) (string, error)
}

// Orchestrator

type Orchestrator struct {
  Profiles ProfileRegistry
  Middlewares MiddlewareRegistry
  Tools ToolRegistry
  PromptResolver PromptResolver
  ToolExecutor ToolExecutor
  EventSinks []events.EventSink
}

func (o *Orchestrator) Run(ctx context.Context, req RunRequest) (*RunResult, error) {
  // 1) load ConversationState
  // 2) apply context / profile
  // 3) resolve prompt
  // 4) compose engine
  // 5) run tool loop
  // 6) update state
}
```

## Conclusion

The current divergence across pinocchio TUI, pinocchio webchat, and moments webchat makes correctness fixes (like the Responses reasoning follower issue) harder to apply uniformly. A shared `ConversationRunner` with injected prompt resolver and tool executor hooks would make the transport layer thin, preserve special capabilities (moments identity, prompt resolver), and consolidate ordering constraints in one place.
