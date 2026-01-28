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
    - Path: pinocchio/cmd/web-chat/README.md
      Note: |-
        Documentation describing getOrCreateConv and engine creation
        Documentation of webchat flow
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: |-
        getOrCreateConv builds engine/sink/subscriber and stores profile/config signature
        getOrCreateConv builds engine/sink/subscriber and manages rebuilds
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: |-
        BuildConfig and BuildFromConfig compose EngineConfig and engine/sink
        BuildConfig/BuildFromConfig responsibilities
    - Path: pinocchio/pkg/webchat/engine_config.go
      Note: |-
        EngineConfig signature drives rebuild decisions
        EngineConfig signature semantics
    - Path: pinocchio/pkg/webchat/engine_from_req.go
      Note: |-
        EngineFromReqBuilder resolves conv/profile/overrides from HTTP/WS request
        Request policy resolution for conv/profile/overrides
    - Path: pinocchio/pkg/webchat/router.go
      Note: Handlers call BuildEngineFromReq then getOrCreateConv
    - Path: pinocchio/pkg/webchat/types.go
      Note: |-
        Router fields and Profile definition shape ownership
        Router/profile ownership boundaries
ExternalSources: []
Summary: |
    Detailed analysis of current webchat engine-building flow and options to decouple
    request policy from conversation lifecycle.
LastUpdated: 2026-01-27T22:46:30-05:00
WhatFor: Map responsibilities between request policy, engine composition, and conversation lifecycle; identify refactor options.
WhenToUse: Before changing webchat conversation/engine creation responsibilities.
---


# Webchat getOrCreateConv vs EngineFromReqBuilder

This analysis is intentionally didactic and explicit; it is meant to be read like a “systems textbook” for the current webchat architecture and the proposed simplification.

## 1) The Core Question

The question is not “where do we build engines?” but rather **what layer owns which responsibility**:

- **Request policy** (derive conv_id, profile, overrides)
- **Engine configuration** (resolve profile defaults + overrides + step settings)
- **Conversation lifecycle** (create/lookup conversation state, queueing, streaming, session)
- **Engine composition** (construct engine + sink + subscriber; rebuild on config changes)

Today, these responsibilities are split across multiple components, with `getOrCreateConv` doing more than just “conversation lookup.”

## 2) System Model (Current State)

### 2.1 Data Structures and Roles

**Conversation (pinocchio/pkg/webchat/conversation.go)**
- Holds per-conversation state:
  - `Eng`, `Sink`, `sub` (engine + stream wiring)
  - `ProfileSlug`, `EngConfigSig` (used for rebuild logic)
  - Queue, idempotency, sem buffer, timeline projector
  - `Sess` (enginebuilder session object)

**EngineFromReqBuilder (pinocchio/pkg/webchat/engine_from_req.go)**
- Resolves **request policy**:
  - Conv ID (query/body)
  - Profile slug (path/cookie, or existing conversation)
  - Overrides (from chat body)
- Output: `EngineBuildInput{ConvID, ProfileSlug, Overrides}`

**Router EngineBuilder (pinocchio/pkg/webchat/engine_builder.go)**
- `BuildConfig(profileSlug, overrides)` → `EngineConfig`
- `BuildFromConfig(convID, config)` → `(engine.Engine, events.EventSink)`
- This is where **profile defaults**, **override validation**, and **step settings** are applied.

### 2.2 Call Graph (WS and Chat)

**WebSocket Join (GET /ws)**

```
HTTP /ws ──> BuildEngineFromReq(req)
            ├─ resolve conv_id (query)
            ├─ resolve profile (query/cookie/conv profile)
            └─ return EngineBuildInput

input ──> getOrCreateConv(convID, profileSlug, overrides=nil)
          ├─ BuildConfig(profileSlug, overrides=nil)
          ├─ compute EngineConfig.Signature()
          ├─ if conversation exists and signature changed → rebuild engine/sink/subscriber
          └─ else create new conversation + engine + sink + subscriber
```

**Chat Run (POST /chat)**

```
HTTP /chat ──> BuildEngineFromReq(req)
              ├─ parse body (prompt, conv_id, overrides)
              ├─ resolve profile (path/cookie/conv profile)
              └─ return EngineBuildInput + body

input ──> getOrCreateConv(convID, profileSlug, overrides)
          ├─ BuildConfig(profileSlug, overrides)
          ├─ compute EngineConfig.Signature()
          ├─ if existing conv signature differs → rebuild engine/sink/subscriber
          └─ else create conversation + engine + sink + subscriber

conv ──> enqueue/run prompt
```

### 2.3 Responsibility Matrix (Current)

| Responsibility | Current Owner | Notes |
| --- | --- | --- |
| Parse request body/params | `EngineFromReqBuilder` | conv id + profile + overrides |
| Validate overrides | `BuildConfig` | denies if profile disallows |
| Compose EngineConfig | `BuildConfig` | profile defaults + overrides + step settings |
| Decide rebuild | `getOrCreateConv` | compares `EngConfigSig` |
| Build engine/sink/sub | `getOrCreateConv` via `BuildFromConfig` | rebuilds inside conv lock |
| Store per-conv metadata | `Conversation` | `ProfileSlug`, `EngConfigSig` |

**Key observation:**  
`EngineFromReqBuilder` is **not** an engine builder; it’s a *policy resolver* for request semantics. Engine composition and rebuild are *not* in that layer.

## 3) Why This Feels “Wrong” (User Intuition)

The intuition “getOrCreateConv should only retrieve existing conversation” comes from a clean separation of concerns:

> “Conversation lifecycle” should not imply “engine creation policy.”

Today, however, **engine composition is bound to conversation creation** because:
- the engine must publish to a conversation-specific topic (`chat:{convID}`)
- conversation streaming is wired to the engine’s event sink
- rebuilds are triggered on *config* changes, which are currently detected inside `getOrCreateConv`.

So the “conversation” is not just a conversation; it is a **fully wired stream + engine instance**.

## 4) How EngineFromReqBuilder Relates (and Why It Doesn’t Replace getOrCreateConv)

`EngineFromReqBuilder` answers:

> “Given this HTTP request, which conv_id/profile/overrides should we use?”

It **does not** answer:

- What engine should be built?
- What middlewares/tools are applied?
- How do overrides affect step settings?
- When should a running conversation be rebuilt?

In other words:

```
EngineFromReqBuilder == request policy
BuildConfig/BuildFromConfig == engine policy
getOrCreateConv == lifecycle + engine wiring
```

Therefore, **EngineFromReqBuilder cannot “replace” getOrCreateConv** without expanding its scope to include engine configuration and construction.

## 5) Root Cause of “Too Much Responsibility”

The coupling comes from this chain:

```
convID -> topicForConv(convID) -> sink/subscriber -> engine -> conversation
```

Because the sink and subscriber are conv-specific, engine construction is currently *anchored* to the conversation object. This leads to:

- `getOrCreateConv` both retrieving and *building/rewiring* engine resources.
- `BuildConfig` being invoked inside `getOrCreateConv`, even though config is derived from request policy.

## 6) What “Simplify” Could Mean (Design Options)

Below are concrete refactor options; each keeps existing behavior while clarifying ownership.

### Option A — Introduce an explicit EnginePlan (recommended minimal change)

**Idea:** Separate “engine plan” from “conversation lookup.”

1) Expand request policy to return a **plan**:
   - `EnginePlan{ConvID, ProfileSlug, Overrides, Config, ConfigSig}`
2) `getOrCreateConv` only accepts a plan and does **no BuildConfig** internally.

Pseudo-API:

```go
type EnginePlan struct {
    ConvID string
    ProfileSlug string
    Overrides map[string]any
    Config EngineConfig
    ConfigSig string
}

func (r *Router) BuildPlanFromReq(req *http.Request) (EnginePlan, *ChatRequestBody, error)
func (r *Router) getOrCreateConvWithPlan(plan EnginePlan) (*Conversation, error)
```

**Pros**
- Clean, explicit boundary: request → plan → conversation.
- `getOrCreateConv` no longer reaches into profiles/overrides directly.
- Minimal change in flow; no need to rework WebSocket/chat handlers heavily.

**Cons**
- Slightly larger “request builder” surface (it now pulls in BuildConfig).
- Needs careful plan creation for existing conversation case.

### Option B — Make `getOrCreateConv` purely retrieval, add `EnsureEngine` method

Split into:

```go
conv := r.getOrCreateConv(convID)
r.ensureEngine(conv, plan)
```

**Pros**
- Clear separation: retrieval vs engine wiring.
- `getOrCreateConv` name matches behavior.

**Cons**
- Adds two-step call sequence everywhere; easy to forget to call `EnsureEngine`.
- Requires careful locking choreography to keep lifecycle safe.

### Option C — Expand EngineFromReqBuilder into full EngineFromReqPlanner

Make request builder return `EngineConfig` directly:

```go
BuildEngineFromReq(req) (EngineConfig, EngineBuildInput, *ChatRequestBody, error)
```

**Pros**
- A single object yields both request policy and engine config.
- getOrCreateConv can be simplified to take `EngineConfig`.

**Cons**
- Request builder now depends on parsed layers + settings builder; it is no longer a “policy-only” interface.
- Harder to test in isolation (needs step settings).

### Option D — Move engine creation entirely out of conversation, keep conv as pure state

Create a separate `ConversationRuntime` (engine + sink + subscriber) and make
`Conversation` purely state/queue. This is a bigger architecture shift.

**Pros**
- Clean layering; easier to test each layer independently.

**Cons**
- Large refactor with higher risk; likely not needed right now.

## 7) Recommended Direction (Short-Term)

**Recommendation: Option A (EnginePlan)**  
It is the smallest change that makes ownership explicit:

- Request handlers remain mostly intact.
- `BuildPlanFromReq` bridges request policy → engine config.
- `getOrCreateConv` no longer decides config; it only reconciles “plan vs current”.

This aligns with the user’s intuition: **conversation retrieval stops doing policy**.
The plan is explicit and testable.

## 8) Additional Cleanups (If Doing This Refactor)

### 8.1 Rename for clarity

- `getOrCreateConv` → `getOrCreateConversationRuntime` (if it still builds engines)
- or if split:
  - `getOrCreateConversationState` (pure lookup)
  - `ensureConversationRuntime` (engine/sink/subscriber wiring)

### 8.2 Push config signature comparisons to a single place

Right now the comparison happens inside `getOrCreateConv`. If you move to plans, this becomes:

```
if conv.EngineConfigSig != plan.ConfigSig -> rebuild
```

This is still valid but now derived from the plan rather than rebuilt ad-hoc.

### 8.3 Make “profile” identity stable

`EngineFromReqBuilder` uses existing conversation profile as a fallback.
That is correct, but it means **profile selection is stateful**. In a plan-based
design, that becomes explicit:

```
plan.ProfileSlug = existing.ProfileSlug if set
```

This should be documented as a contract.

## 9) Key Takeaways (Callouts)

**Callout 1: Policy vs Composition**  
`EngineFromReqBuilder` is not an engine builder; it is a request-policy resolver.

**Callout 2: Conversation != Engine**  
The conversation object currently bundles state, streaming infrastructure, and
engine wiring; separating these simplifies reasoning.

**Callout 3: Config Signature is the real “engine identity”**  
Engine rebuild should be keyed to `EngineConfig.Signature()` rather than to
the profile alone.

## 10) Decision Checklist

Use this checklist to decide whether to refactor:

- Do we want `getOrCreateConv` to be purely a state lookup?
- Are we comfortable expanding request builders to return `EngineConfig`?
- Do we need a single “plan” object to carry policy + configuration?
- Can we keep engine rebuild logic in one place without double-building configs?

If “yes” to the above, proceed with Option A (EnginePlan).

