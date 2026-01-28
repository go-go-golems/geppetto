---
Title: Webchat getOrCreateConv vs EngineFromReqBuilder analysis
Ticket: PI-004-WEBCHAT-ENGINE-BUILD
Status: active
Topics:
  - webchat
  - refactor
  - analysis
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
  - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
    Note: getOrCreateConv builds engine/sink/subscriber and stores profile/config signature
  - Path: ../../../../../../../pinocchio/pkg/webchat/engine_from_req.go
    Note: EngineFromReqBuilder resolves conv/profile/overrides from HTTP/WS request
  - Path: ../../../../../../../pinocchio/pkg/webchat/engine_builder.go
    Note: BuildConfig and BuildFromConfig compose EngineConfig and engine/sink
  - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
    Note: Handlers call BuildEngineFromReq then getOrCreateConv
  - Path: ../../../../../../../pinocchio/pkg/webchat/engine_config.go
    Note: EngineConfig signature drives rebuild decisions
  - Path: ../../../../../../../pinocchio/pkg/webchat/types.go
    Note: Router fields and Profile definition shape ownership
  - Path: ../../../../../../../pinocchio/cmd/web-chat/README.md
    Note: Documentation describing getOrCreateConv and engine creation
ExternalSources: []
Summary: |
  Textbook-style analysis of current webchat engine-building flow and options to
  decouple request policy from conversation lifecycle.
LastUpdated: 2026-01-27T22:59:30-05:00
WhatFor: Map responsibilities between request policy, engine composition, and conversation lifecycle; identify refactor options.
WhenToUse: Before changing webchat conversation/engine creation responsibilities.
---

# Webchat getOrCreateConv vs EngineFromReqBuilder: A Textbook Treatment

> "Separate the questions 'what is requested' and 'how to fulfill it', and you will see the seams where systems want to split."

This document is a didactic, system-level analysis of the webchat backend. It focuses on the boundary between request policy (conv_id/profile/overrides) and engine composition (middlewares/tools/step settings), and on the role that `getOrCreateConv` currently plays in joining those domains.

If you are short on time, read Sections 2, 4, and 7. They capture the core architecture, the coupling, and a recommended path forward.

---

## 1) Problem Statement, in Precise Terms

We want to simplify the webchat engine-building flow. The specific question is:

> Should engine creation logic be moved out of `getOrCreateConv`, possibly into `EngineFromReqBuilder` (or another component), so that `getOrCreateConv` only retrieves an existing conversation?

The underlying design intent is to cleanly separate responsibilities:

- Request policy: interpret HTTP and WebSocket requests
- Engine policy: resolve profiles, overrides, and step settings
- Conversation lifecycle: create or fetch conversation state and manage runtime attachments

The current system blends these responsibilities, which makes the boundaries harder to reason about and to evolve.

---

## 2) The Current Architecture, by Layer

### 2.1 The Four-Layer Model (Backend Only)

```
┌───────────────────────────────────────────────────────────────────────┐
│ Layer 4: HTTP/WS Handlers (router.go)                                  │
│ - Parses request, handles errors, logs, routes to conv runtime         │
└───────────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌───────────────────────────────────────────────────────────────────────┐
│ Layer 3: Request Policy (engine_from_req.go)                           │
│ - Derive conv_id, profile slug, overrides                              │
│ - Preserve existing profile if conversation already exists             │
└───────────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌───────────────────────────────────────────────────────────────────────┐
│ Layer 2: Conversation Runtime (conversation.go)                        │
│ - Create/reuse conversation                                             │
│ - Build engine + sink + subscriber                                      │
│ - Rebuild on config signature changes                                  │
└───────────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌───────────────────────────────────────────────────────────────────────┐
│ Layer 1: Engine Composition (engine_builder.go)                        │
│ - BuildConfig(profile, overrides)                                      │
│ - BuildFromConfig(convID, config)                                      │
└───────────────────────────────────────────────────────────────────────┘
```

The important point is that Layer 2 (conversation) currently calls into Layer 1 (engine composition). There is no separate boundary object between them.

### 2.2 The Key Interfaces (Contracts)

**EngineFromReqBuilder**

```go
// request policy
BuildEngineFromReq(req) -> (EngineBuildInput, *ChatRequestBody, error)
```

Output:
```
EngineBuildInput{
  ConvID,
  ProfileSlug,
  Overrides,
}
```

**EngineBuilder**

```go
BuildConfig(profileSlug, overrides) -> EngineConfig
BuildFromConfig(convID, config) -> (engine.Engine, events.EventSink)
```

**Conversation Lifecycle**

```go
getOrCreateConv(convID, profileSlug, overrides) -> *Conversation
```

This method both:
- Computes config (`BuildConfig`)
- Composes engine/sink (`BuildFromConfig`)
- Rebuilds on config signature changes

---

## 3) Data Flow: What Actually Happens at Runtime

### 3.1 WebSocket Join

```
HTTP GET /ws?conv_id=...&profile=...
   │
   ├─ BuildEngineFromReq -> (conv_id, profile, overrides=nil)
   │
   └─ getOrCreateConv(conv_id, profile, nil)
         ├─ BuildConfig(profile, nil)
         ├─ config.Signature()
         ├─ if existing conv signature changed -> rebuild engine + sink + sub
         └─ else create conversation and attach runtime components
```

### 3.2 Chat Request

```
HTTP POST /chat
   │
   ├─ BuildEngineFromReq -> (conv_id, profile, overrides) + body
   │
   └─ getOrCreateConv(conv_id, profile, overrides)
         ├─ BuildConfig(profile, overrides)
         ├─ config.Signature()
         ├─ if existing conv signature changed -> rebuild engine + sink + sub
         └─ else create conversation and attach runtime components
```

This is a common pattern: request policy runs first, then conversation ensures engine wiring.

---

## 4) The Hidden Coupling: Why `getOrCreateConv` Is "Too Heavy"

The design tension is not cosmetic. It is structural. Consider the following chain:

```
convID -> topicForConv(convID) -> event sink -> subscriber -> stream coordinator
```

The engine must emit events to a conversation-specific topic, which ties engine composition to the conversation runtime. This forces `getOrCreateConv` to do more than retrieval.

**Therefore, `getOrCreateConv` is effectively:**

- a conversation cache,
- a runtime attachment manager,
- and a configuration-dependent engine factory.

That is a lot of responsibility for one method, especially when its name suggests simple retrieval.

---

## 5) What EngineFromReqBuilder Is and Is Not

The name *EngineFromReqBuilder* invites the assumption that it builds engines. It does not.

It is a policy resolver that answers:

> "Given this request, which conversation ID, profile, and overrides should we use?"

It does not decide:

- Which middlewares to apply
- Which tools to include
- What step settings are in scope
- Whether an engine rebuild is needed

Put simply:

```
EngineFromReqBuilder = request policy
BuildConfig/BuildFromConfig = engine policy
getOrCreateConv = lifecycle + engine wiring
```

So the current naming is part of the confusion: it sounds like a builder, but acts like a policy resolver.

---

## 6) Design Options (With Trade-offs)

We want to simplify ownership boundaries without losing behavior. Below are four options, ordered by increasing scope.

### Option A: Introduce an EnginePlan (Recommended)

**Key idea:** Convert "request policy" + "engine policy" into a single **plan** that gets applied by the conversation runtime.

```
type EnginePlan struct {
  ConvID      string
  ProfileSlug string
  Overrides   map[string]any
  Config      EngineConfig
  ConfigSig   string
}
```

Then:

```go
plan, body := r.BuildPlanFromReq(req)
conv := r.getOrCreateConvWithPlan(plan)
```

**Pros**
- Explicit boundary: request policy -> plan -> conversation runtime
- Removes `BuildConfig` from inside `getOrCreateConv`
- Keeps runtime behavior unchanged (still rebuilds on config signature)

**Cons**
- Slightly expands the request builder surface
- Requires touching handler code paths

### Option B: Split getOrCreateConv into Two Methods

```
conv := r.getOrCreateConversationState(convID)
r.ensureConversationRuntime(conv, plan)
```

**Pros**
- `getOrCreateConv` name becomes truthful
- Conversation retrieval is cleanly separated

**Cons**
- Two calls everywhere, easy to misuse
- Requires careful locking to avoid inconsistent runtime state

### Option C: Expand EngineFromReqBuilder to Return EngineConfig

```
BuildEngineFromReq(req) -> (EngineConfig, EngineBuildInput, Body)
```

**Pros**
- Single call yields everything needed
- `getOrCreateConv` can be simplified to only apply config

**Cons**
- Request builder now depends on parsed layers and step settings
- Harder to test and mock

### Option D: Split Conversation Into State and Runtime

Create a `ConversationRuntime` for engine/sink/subscriber and keep `Conversation` as state/queue only.

**Pros**
- Clean layering; reasoning becomes simpler

**Cons**
- Large refactor; higher risk and change surface

---

## 7) Recommended Direction

**Recommendation: Option A (EnginePlan)**  
It is the minimum refactor that makes ownership explicit without large architectural upheaval.

It preserves the useful invariant:

> "Engine wiring is driven by a configuration signature that is stable and derivable from request inputs."

It also directly addresses the user intuition:

> "getOrCreateConv should not decide profiles or middlewares."

With a plan, it does not. It simply compares the plan to the existing runtime and rebuilds when the signature changes.

---

## 8) Practical Guidance: What Would Change

### 8.1 New Flow (Sketch)

```
req -> BuildPlanFromReq
     -> plan = {conv_id, profile, overrides, config, sig}
     -> getOrCreateConvWithPlan(plan)
```

### 8.2 Updated Responsibilities

| Responsibility | Proposed Owner |
| --- | --- |
| request parsing | EngineFromReqBuilder |
| config composition | EnginePlan builder |
| rebuild decision | getOrCreateConvWithPlan |
| engine construction | BuildFromConfig |
| conversation retrieval | getOrCreateConvWithPlan (or split) |

### 8.3 Renaming for Clarity

If we keep `getOrCreateConv` doing runtime wiring, consider renaming it to:

- `getOrCreateConversationRuntime`

If we split it, consider:

- `getOrCreateConversationState`
- `ensureConversationRuntime`

This is not cosmetic; it is a way of encoding intent in the API.

---

## 9) Risks and Invariants

### Invariants That Must Hold

1) The sink must publish to the correct conversation topic (`chat:{convID}`).
2) The subscriber must follow that same topic.
3) Rebuilds must occur if and only if the config signature changes.
4) A conversation without a runtime is not useful to clients.

### Risks

- If the plan/config signature is computed in multiple places, the rebuild logic may diverge.
- If profile fallback logic (existing conversation profile) is removed or changed, user workflows could break.

---

## 10) Summary in One Paragraph

`EngineFromReqBuilder` currently resolves request policy but does not build engines; `getOrCreateConv` composes engines as part of conversation lifecycle, creating a responsibility overlap that makes the system feel coupled. The cleanest simplification is to introduce an explicit **EnginePlan** that combines request policy and engine configuration, so that `getOrCreateConv` merely applies a plan instead of deriving one. This keeps behavior intact while making the ownership boundary explicit and testable.

---

## Appendix: Glossary

- **ConvID**: Stable conversation identifier, used for topics and state lookup.
- **Profile**: Named configuration (prompt, tools, middlewares).
- **Overrides**: Per-request modifications to profile defaults.
- **EngineConfig**: Canonical configuration that influences engine composition.
- **Signature**: Deterministic string from EngineConfig used to decide rebuilds.

