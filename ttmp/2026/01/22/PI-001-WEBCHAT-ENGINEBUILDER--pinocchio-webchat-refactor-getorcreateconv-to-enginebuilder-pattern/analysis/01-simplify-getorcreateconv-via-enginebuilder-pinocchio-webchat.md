---
Title: Simplify getOrCreateConv via EngineBuilder (pinocchio webchat)
Ticket: PI-001-WEBCHAT-ENGINEBUILDER
Status: active
Topics:
    - pinocchio
    - webchat
    - refactor
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/session/builder.go
      Note: Geppetto session.EngineBuilder pattern we want to align with
    - Path: geppetto/pkg/inference/session/tool_loop_builder.go
      Note: Geppetto ToolLoopEngineBuilder as a reusable runner builder
    - Path: go-go-mento/go/pkg/webchat/conversation_manager.go
      Note: Reference ConversationManager pattern (GetOrCreate + config signatures + subscriber factory)
    - Path: go-go-mento/go/pkg/webchat/engine_builder.go
      Note: Reference EngineBuilder pattern (engine+sinks) from go-go-mento
    - Path: moments/backend/pkg/webchat/router.go
      Note: Moments build() closures (sink builder + engine composition + subscriber) showing current complexity
    - Path: pinocchio/pkg/inference/enginebuilder/parsed_layers.go
      Note: Existing ParsedLayersEngineBuilder abstraction we may reuse/reshape
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Current getOrCreateConv implementation and Conversation struct state
    - Path: pinocchio/pkg/webchat/router.go
      Note: Current getOrCreateConv callers build inline build() closures (sink+engine+subscriber)
ExternalSources: []
Summary: "Deep analysis and refactor plan to remove ad-hoc build() closures from Pinocchio webchat getOrCreateConv by adopting a go-go-mento-style EngineBuilder + config signatures + subscriber factory, with a migration path for Moments."
LastUpdated: 2026-01-22T13:11:49.782267796-05:00
WhatFor: "Guide a future refactor ticket: consolidate engine/sink/subscriber composition in Pinocchio webchat and make it reusable for Moments."
WhenToUse: "Use when implementing PI-001 and when evaluating how Moments can migrate away from its current ad-hoc webchat build closures."
---

# Simplify `getOrCreateConv` via EngineBuilder (Pinocchio webchat)

## Problem Statement

Pinocchio’s webchat currently wires conversation creation with **per-callsite** `build := func() (...)` closures that construct:

- a provider `engine.Engine`
- a `WatermillSink` bound to the conversation topic
- a `message.Subscriber` (Redis group subscriber vs in-memory)

`getOrCreateConv` then **caches the first result** it sees for a `conv_id` and returns it on all subsequent calls.

This is hard to maintain because:

1. The closure logic is duplicated across multiple routes (`/ws`, `/chat`, `/chat/{profile}`).
2. Policy decisions (profile defaults, overrides parsing) are mixed with transport wiring (Redis subscriber setup).
3. Once a conversation exists, **profile/override changes are ignored**, which can lead to subtle “engine doesn’t match request” behavior.

The goal of PI-001 is to remove these ad-hoc closures and replace them with a reusable **EngineBuilder + config signature** pattern inspired by go-go-mento (and compatible with Moments’ heavier needs).

## Scope / Non-goals

In scope (this ticket’s refactor target):

- Replace per-callsite `build()` closures with a centralized builder abstraction.
- Make rebuild decisions deterministic (config signature).
- Introduce a reusable pattern that Moments can adopt later, without immediately porting Moments.

Out of scope for PI-001 (but the design should not block these):

- Full Moments migration / consolidation (that’s MO-001).
- DB persistence/hydration logic.
- Step controller integration (separate workstream).

## Current Pinocchio Architecture (As-Is)

### Call graph (simplified)

Pinocchio has three major callsites that all construct a `build()` closure:

- WS connect (`/ws`)
- Start run (`/chat`) with cookie/default profile + overrides
- Start run (`/chat/{profile}`) with explicit profile + overrides

Each callsite calls:

```
conv, err := r.getOrCreateConv(convID, build)
```

### Responsibility split today

#### `getOrCreateConv` (pinocchio)

File: `pinocchio/pkg/webchat/conversation.go`

Responsibilities:

- “Create once per conv_id” cache (no rebuild logic).
- Assign a generated `RunID` (effectively the session id).
- Attach:
  - connection pool
  - stream coordinator
  - session (`*session.Session`) seeded with a first turn and `ToolLoopEngineBuilder` (base engine + event sinks)
- Start the stream coordinator immediately.

#### `build()` closures (callsite-local)

File: `pinocchio/pkg/webchat/router.go`

Duplicated responsibilities:

1. **Transport wiring**
   - `EnsureGroupAtTail` (Redis)
   - build group subscriber (Redis) vs use in-memory subscriber (router.Subscriber)
2. **Sink creation**
   - `middleware.NewWatermillSink(r.router.Publisher, topicForConv(convID))`
3. **Settings + policy application**
   - StepSettings from `ParsedLayers`
   - profile default system prompt and middleware list
   - apply overrides (system_prompt, middlewares)
4. **Engine composition**
   - `composeEngineFromSettings(stepSettings, sys, uses, r.mwFactories)`

### Observed issues

#### A) “Create-once cache” ignores profile/override changes

Because `getOrCreateConv` does not compare a config signature, the first route to create a conversation “wins”.
For example:

- A client joins via WS using profile A (creates conv + engine A).
- Later, a client POSTs `/chat` with overrides (expects engine B).
- `getOrCreateConv` returns the existing conversation, and the overrides are silently ignored for the base engine.

Moments partially addresses this by rebuilding on a signature change, but Pinocchio does not today.

#### B) The closure return type forces `*middleware.WatermillSink`

Pinocchio stores `Sink *middleware.WatermillSink` on the conversation, not `events.EventSink`. That makes it awkward to introduce
sink wrapper pipelines (extractors, filtering, fanout) without changing types.

go-go-mento treats WatermillSink as the base sink and returns `events.EventSink` from its builder.

#### C) Policy and transport are entangled

Redis group management and subscriber creation are embedded in the same closure as profile parsing and engine composition.
This is exactly the kind of “wiring glue” that becomes brittle as the code grows.

## Reference Patterns (What We Want To Reuse)

### go-go-mento: EngineBuilder + EngineConfig + ConversationManager

Files:

- `go-go-mento/go/pkg/webchat/engine_builder.go`
- `go-go-mento/go/pkg/webchat/engine_config.go`
- `go-go-mento/go/pkg/webchat/conversation_manager.go`

Key ideas:

1. **EngineConfig as a first-class object**
   - Capture all inputs that influence engine composition.
   - Serialize to JSON and use the JSON string as a debuggable signature.
2. **EngineBuilder centralizes composition**
   - `BuildConfig(profileSlug, overrides) -> EngineConfig`
   - `BuildFromConfig(convID, EngineConfig) -> (engine, eventsink)`
3. **Subscriber creation is independent**
   - `SubscriberFactory(convID) -> subscriber`
4. **ConversationManager owns rebuild decisions**
   - Rebuild engine/sink/subscriber if profile or signature changes.

This pattern is the closest direct match for what Pinocchio needs.

### Moments: similar needs, but “ad-hoc closure” wiring remained

Files:

- `moments/backend/pkg/webchat/router.go`
- `moments/backend/pkg/webchat/conversation.go`

Moments already has:

- a `getOrCreateConv` that can rebuild on “signature changes”
- a concept of profile slug and a per-conversation engine config signature field
- more state (identity session refresh, step controller, doc lens extractors, etc.)

But Moments still relies on:

- callsite-local `build := func() (engine, sink, subscriber)` closures
- a weak signature: `profileSlug + constantSuffix`

Pinocchio can provide a cleaner reusable pattern that Moments can later adopt.

### Geppetto: `session.EngineBuilder` (runner builder) is related but distinct

File: `geppetto/pkg/inference/session/builder.go`

Geppetto’s “builder” builds a **runner** (`InferenceRunner`) for a session, not an engine/sink/subscriber triple.
It’s still relevant because Pinocchio webchat ultimately needs to run inference via `session.Session.StartInference`.

But the immediate PI-001 refactor target is the webchat “composition glue” around `getOrCreateConv`.

## Proposed Design (To-Be)

### Design Principle

Replace:

- “per-callsite closure that returns engine/sink/subscriber”

With:

- “centralized EngineBuilder (engine + sink)”
- “centralized SubscriberFactory (subscriber)”
- “ConversationManager (get-or-create + rebuild-on-signature)”

### Diagram: To-Be components

```
                     +------------------------------+
HTTP/WS handlers --> | ConversationManager          |
                     | - GetOrCreate(conv_id, ...)  |
                     | - rebuild on signature       |
                     +---------------+--------------+
                                     |
                  +------------------+------------------+
                  |                                     |
         +--------v---------+                   +-------v--------+
         | EngineBuilder    |                   | Subscriber      |
         | - BuildConfig    |                   | Factory         |
         | - BuildFromConfig|                   | - Build(conv_id)|
         +--------+---------+                   +-------+--------+
                  |                                     |
                  v                                     v
          (engine, eventsink)                     (subscriber)
                  |
                  v
         +--------+---------+
         | Conversation     |
         | - engine         |
         | - sink           |
         | - subscriber     |
         | - stream         |
         | - session        |
         +------------------+
```

### Proposed Pinocchio EngineConfig (minimal common denominator)

We want something that matches go-go-mento closely enough that migration is obvious:

```go
type EngineConfig struct {
    ProfileSlug  string                 `json:"profile_slug"`
    SystemPrompt string                 `json:"system_prompt"`
    Middlewares  []MiddlewareUse        `json:"middlewares"`
    StepSettings *settings.StepSettings `json:"step_settings"`
    // Optional: Tools []string (Pinocchio tool registry is currently built per-run)
}

func (c EngineConfig) Signature() string // returns JSON string
```

Notes:

- StepSettings must be included (or a stable derivative of it), otherwise the signature is incomplete.
- Tools are currently assembled per run from `Router.toolFactories`; we can add them later if we want rebuilds to
  occur when tool configuration changes.

### Proposed EngineBuilder interface (Pinocchio)

This is intentionally isomorphic to go-go-mento:

```go
type EngineBuilder interface {
    BuildConfig(profileSlug string, overrides map[string]any) (EngineConfig, error)
    BuildFromConfig(convID string, cfg EngineConfig) (engine.Engine, events.EventSink, error)
}
```

Implementation dependencies likely include:

- `parsed *layers.ParsedLayers`
- `profiles ProfileRegistry`
- `mwFactories map[string]MiddlewareFactory`
- `publisher message.Publisher` (to build WatermillSink)
- optional sink wrapping hooks (for future Moments reuse)

### Proposed SubscriberFactory (Pinocchio)

Keep transport details out of the EngineBuilder:

```go
type SubscriberFactory func(convID string) (message.Subscriber, error)
```

Pinocchio already has two distinct strategies:

- Redis: ensure group + build group subscriber with consumer name `ws-forwarder:<conv_id>`
- In-memory: reuse `router.Subscriber`

### Proposed ConversationManager (Pinocchio)

This can start as a simplified port of go-go-mento’s manager:

```go
type ConversationManager struct {
    mu            sync.Mutex
    conversations map[string]*Conversation
    builder       EngineBuilder
    subscriber    SubscriberFactory
    baseCtx       context.Context
    idleTimeout   time.Duration
}

func (cm *ConversationManager) GetOrCreate(
    ctx context.Context,
    convID string,
    profileSlug string,
    overrides map[string]any,
) (*Conversation, error)
```

Behavior:

- Build EngineConfig from profile + overrides.
- Compute signature.
- If conversation exists:
  - If profile changed OR signature changed: rebuild engine/sink, rebuild subscriber, reattach stream coordinator.
  - Else: reuse.
- If conversation does not exist:
  - create new conversation, attach pool/stream/session.

### “Sink type” choice: store `events.EventSink` not `*WatermillSink`

Recommendation:

- Conversation should store `events.EventSink` (which can be a WatermillSink or a wrapped sink).
- The base Watermill sink should be an internal detail of the EngineBuilder.

This aligns with go-go-mento and makes Moments adoption easier (Moments uses sink wrapper pipelines heavily).

## How This Helps Moments Later

Moments has the same refactor pain, but amplified:

- more sink wrapping
- more “profile resolution” complexity
- identity session refresh
- step controller
- persistence/hydration hooks

If Pinocchio adopts the go-go-mento-style builder/manager split, Moments can migrate incrementally:

1. Introduce a Moments EngineBuilder that implements the same interface:
   - BuildConfig(profileSlug, overrides) returns a JSON-signatured config including resolved prompt/middleware/tool lists.
   - BuildFromConfig(convID, cfg) composes engine + sink pipeline over a base Watermill sink.
2. Swap Moments’ `getOrCreateConv` to call the builder, and delete ad-hoc callsite closures.
3. Keep Moments-only concerns (identity, step mode) in the conversation manager layer, not in the builder.

This allows a “clean core” to exist even if Moments’ package remains messy in the short term.

## Migration Plan (Implementation Checklist for PI-001)

No code is written in this ticket yet; this is an implementation plan for the next phase.

1. Define `EngineConfig` + `Signature()` in Pinocchio webchat (or a shared pinocchio package).
2. Introduce `EngineBuilder` interface matching go-go-mento’s API.
3. Implement a Pinocchio `EngineBuilder` using:
   - `settings.NewStepSettingsFromParsedLayers(r.parsed)`
   - profile default prompt/middlewares
   - override parsing logic currently in `/chat` handlers
   - `composeEngineFromSettings(...)`
   - `middleware.NewWatermillSink(r.router.Publisher, topicForConv(convID))`
4. Introduce `SubscriberFactory` for Redis vs in-memory subscriber creation.
5. Replace `Router.getOrCreateConv(convID, build)` with `ConversationManager.GetOrCreate(...)`.
6. Remove the duplicated `build := func() ...` blocks from `pinocchio/pkg/webchat/router.go`.
7. Add tests for:
   - config signature stability
   - rebuild on signature change
   - no rebuild on identical inputs

## Open Questions

1. Should Pinocchio rebuild conversations on:
   - WS join profile (cookie) changes?
   - `/chat` overrides changes mid-session?
   The go-go-mento answer is “yes, rebuild when signature changes”.
2. Do we want a “session manager” layer (go-go-mento style) in Pinocchio soon anyway?
   If yes, PI-001 should structure code to make that addition natural.

## Appendix: Where to look in code (As-Is)

- Pinocchio:
  - `pinocchio/pkg/webchat/router.go` (three duplicated build closures)
  - `pinocchio/pkg/webchat/conversation.go` (`getOrCreateConv` cache semantics)
  - `pinocchio/pkg/webchat/engine.go` (`composeEngineFromSettings`)
  - `pinocchio/pkg/inference/enginebuilder/parsed_layers.go` (existing minimal builder)
- go-go-mento:
  - `go-go-mento/go/pkg/webchat/engine_builder.go`
  - `go-go-mento/go/pkg/webchat/engine_config.go`
  - `go-go-mento/go/pkg/webchat/conversation_manager.go`
- Moments:
  - `moments/backend/pkg/webchat/router.go`
  - `moments/backend/pkg/webchat/conversation.go`
