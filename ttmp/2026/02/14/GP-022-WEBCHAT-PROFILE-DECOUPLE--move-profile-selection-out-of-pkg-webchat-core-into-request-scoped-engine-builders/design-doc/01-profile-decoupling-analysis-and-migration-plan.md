---
Title: Profile Decoupling Analysis and Migration Plan
Ticket: GP-022-WEBCHAT-PROFILE-DECOUPLE
Status: active
Topics:
    - architecture
    - pinocchio
    - chat
    - migration
    - inference
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/main.go
      Note: web-chat app must own profile registration and request policy wiring
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Conversation identity currently stores ProfileSlug and rebuild logic based on it
    - Path: pinocchio/pkg/webchat/engine_from_req.go
      Note: Default request builder is currently profile-aware and needs to become profile-neutral
    - Path: pinocchio/pkg/webchat/router.go
      Note: Core currently hosts profile endpoints and cookie behavior; target is app-owned policy
    - Path: pinocchio/pkg/webchat/send_queue.go
      Note: Queued requests currently persist ProfileSlug and need generic engine identity
    - Path: pinocchio/pkg/webchat/types.go
      Note: Core currently defines profile registry/types that must be removed from pkg/webchat
    - Path: pinocchio/proto/sem/base/ws.proto
      Note: WsHelloV1 currently exposes profile field which should be generalized or removed
    - Path: web-agent-example/cmd/web-agent-example/engine_from_req.go
      Note: Single-profile builder already exists and should map to generic engine key contract
    - Path: web-agent-example/cmd/web-agent-example/main.go
      Note: web-agent-example should no longer depend on core profile APIs
ExternalSources: []
Summary: Detailed design to remove profile string handling from pkg/webchat core and move request policy/config selection to app-level EngineFromReq builders.
LastUpdated: 2026-02-14T15:54:06.698755336-05:00
WhatFor: ""
WhenToUse: ""
---


# Profile Decoupling Analysis and Migration Plan

## Executive Summary

`pinocchio/pkg/webchat` currently mixes three concerns:
1. generic chat runtime (conversation lifecycle, queueing, websocket fanout),
2. engine composition,
3. product policy for "profiles" (slug parsing, cookie handling, profile registry, profile endpoints).

This ticket proposes a hard split: remove profile-specific knowledge from `pkg/webchat` and move it into application-level request policy/builders in:
- `pinocchio/cmd/web-chat` (multi-profile app),
- `web-agent-example/cmd/web-agent-example` (single-profile app with custom middleware).

The target architecture keeps `pkg/webchat` as a profile-agnostic runtime and pushes profile interpretation to `EngineFromReqBuilder` + app-owned handlers. This also removes cross-app cookie coupling (`chat_profile`) and makes `web-agent-example` first-class without profile stubs.

## Problem Statement

The current core package owns profile semantics at multiple layers:
- Data model and registry:
  - `pinocchio/pkg/webchat/types.go` defines `Profile`, `ProfileRegistry`, and stores `Router.profiles`.
- HTTP API:
  - `pinocchio/pkg/webchat/router.go` serves `/api/chat/profiles` and `/api/chat/profile` and writes `chat_profile` cookie.
- Request policy:
  - `pinocchio/pkg/webchat/engine_from_req.go` resolves profile from path/query/cookie/default.
- Conversation identity:
  - `pinocchio/pkg/webchat/conversation.go` stores `Conversation.ProfileSlug`.
  - `pinocchio/pkg/webchat/send_queue.go` stores per-queued-request `ProfileSlug`.
- Engine composition:
  - `pinocchio/pkg/webchat/engine_builder.go` uses profile slug to build config.
- Debug metadata:
  - `pinocchio/pkg/webchat/router_debug_routes.go` emits `profile` field.
- WS protocol:
  - `pinocchio/proto/sem/base/ws.proto` includes `WsHelloV1.profile`.

This coupling creates concrete issues:
1. Core package enforces one product policy (profile slug + cookie) for all apps.
2. `web-agent-example` must fake profile behavior (`ProfileSlug: "default"`) despite not needing profile selection.
3. Shared localhost cookie behavior can leak across apps and cause confusing runtime behavior.
4. Core API surface is larger and harder to reason about than needed.
5. The reusable frontend package (`@pwchat/webchat`) is implicitly tied to profile endpoints.

## Proposed Solution

### 1) Make `pkg/webchat` profile-agnostic

Remove profile concepts from the core runtime package:
- Remove `Profile`, `ProfileRegistry`, `WithProfileRegistry`, `Router.AddProfile`.
- Remove profile API endpoints from `registerAPIHandlers`:
  - `/api/chat/profiles`
  - `/api/chat/profile`
- Remove `chat_profile` cookie handling from core.
- Remove any path/query parsing logic that assumes profile slug semantics.

### 2) Move profile policy into app-owned request builder(s)

`EngineFromReqBuilder` becomes the authoritative place where app policy is applied:
- Parse request inputs (body/path/query/cookies/custom headers).
- Resolve app-specific configuration identity.
- Validate requested mode/profile against app-owned registry.
- Return generic core input for conversation/runtime layer.

For `pinocchio/cmd/web-chat`, create an app-owned "profile module":
- profile registry/types,
- builder that maps request -> selected profile -> engine config inputs,
- HTTP handlers for profile discovery/selection.

For `web-agent-example`, builder remains single-profile and cookie-independent.

### 3) Replace profile-specific fields with generic engine identity

Core should track "engine identity/config identity", not "profile":
- `Conversation.ProfileSlug` -> `Conversation.EngineKey` (or remove field entirely and use config signature + optional label).
- queued request `ProfileSlug` -> `EngineKey`.
- debug response `profile` -> `engine_key` (or `engine_label`).
- WS hello frame should not expose profile-specific naming.

Recommended minimal generic model:
- `EngineBuildInput`:
  - `ConvID string`
  - `EngineKey string` (opaque app-defined identity; optional but useful for logs/debug)
  - `Overrides map[string]any` (or renamed `RequestOptions`)
- `ConvManager.GetOrCreate(convID, engineKey, overrides)`
- Core uses `EngineKey` only for comparison/logging and delegates config resolution to injected builders.

### 4) Keep engine composition injectable, not profile-driven

Today `Router.BuildConfig(profileSlug, overrides)` is profile-coupled. Move this responsibility to app-owned configuration logic:
- Core receives function(s) from app wiring:
  - resolve engine config from `EngineBuildInput`,
  - build engine/sink from resolved config.
- Core does not inspect application policy dimensions (profile, persona, tier, mode, tenant, etc.).

### 5) Frontend strategy split by app

- `pinocchio/cmd/web-chat` retains profile UX but sourced from app-owned endpoints.
- `web-agent-example` should not be forced into profile UI:
  - either disable selector in `ChatWidget`,
  - or provide a pluggable profile provider with "none" implementation.

This keeps the core reusable for apps with zero, one, or many configuration modes.

## Design Evolution and Detailed API Proposal (How We Got Here)

This section captures the reasoning path that led from the current `BuildConfig/BuildFromConfig` shape to a stronger external API based on resolving a full conversation request plan. The goal is to make the intended layering obvious and reduce ambiguity around where system prompts, overrides, and runtime decisions should live.

### 1) Current control flow and why it was acceptable initially

The current architecture evolved incrementally and made sense at each local step:

1. `EngineFromReqBuilder` extracts request policy pieces (`conv_id`, `profile`, `overrides`) from HTTP/WS request.
2. `ConvManager.GetOrCreate(convID, profileSlug, overrides)` asks `buildConfig(profileSlug, overrides)` for an `EngineConfig`.
3. `EngineConfig.Signature()` is compared to `Conversation.EngConfigSig`.
4. If signature changed, runtime is rebuilt through `buildFromConfig(convID, cfg)`.

This gives deterministic rebuild behavior and avoids rebuilding on every request. It also allowed profile functionality to ship quickly with minimal app wiring.

The problem is not that the flow is wrong. The problem is where policy is anchored. Today the policy axis (profiles, profile cookie, profile endpoints, profile names in debug and WS hello) is embedded in core.

### 2) Coupling signals that show the boundary is misplaced

During the audit, several repeated signs appeared:

1. **Core request parsing knows app policy semantics.**
   - `profile` query/path/cookie fallbacks are product decisions, not runtime mechanics.
2. **Core API surface exports product concepts.**
   - `/api/chat/profile` and `/api/chat/profiles` are UI/product endpoints.
3. **Core identity uses product naming.**
   - `ProfileSlug` appears in conversation state, queue records, logs, debug payloads, and WS hello.
4. **Single-profile app still pays profile tax.**
   - `web-agent-example` must emit fake `ProfileSlug: "default"` while intentionally ignoring profile behavior.
5. **Ambiguous override semantics.**
   - `overrides` currently contains a mix of engine composition data (`system_prompt`, middlewares, tools) and execution behavior (`step_mode`), which means one map is driving two lifecycles.

These are all boundary smells: core is prescribing too much about how apps choose and represent configuration.

### 3) Principle used to reframe the API

We can separate concerns by asking one question:

**What is the minimal thing webchat core needs to run a conversation safely and efficiently?**

Core needs:
- conversation key (`conv_id`)
- a stable rebuild fingerprint
- a runtime factory to create engine/sink when needed
- per-request execution data (prompt, idempotency, runtime options)

Core does not need:
- profile terminology
- cookie semantics
- path conventions (`/chat/{profile}`)
- UI profile endpoints
- system prompt policy logic

From this, the API should move from:
- "give me partial policy knobs (`profileSlug`, `overrides`) and I will decide config shape"

to:
- "give me the fully resolved plan for this request; I will execute lifecycle mechanics."

### 4) Why `BuildEngineFromReq` is close but still incomplete

`BuildEngineFromReq` was an important step, but it currently returns:
- `ConvID`
- `ProfileSlug`
- `Overrides`
- parsed request body

This still leaks two issues:

1. **Core continues to interpret returned fields semantically.**
   - `ProfileSlug` carries specific policy naming.
2. **Core still owns config composition.**
   - `BuildConfig/BuildFromConfig` are invoked by core and currently encode system prompt and middleware/tool policy.

So, even though request parsing moved outward, configuration policy still partially lives in core.

### 5) Proposed replacement: resolve a complete Conversation Request Plan

The cleaner contract is an app-owned resolver that returns everything core needs in one object.

```go
type ConversationRequestResolver interface {
    Resolve(req *http.Request) (*ConversationRequestPlan, error)
}

type ConversationRequestPlan struct {
    // routing
    ConvID string

    // request execution data
    Prompt         string
    IdempotencyKey string
    RuntimeOptions map[string]any

    // runtime identity
    RuntimeKey  string // optional, debug-friendly identity
    Fingerprint string // required, stable rebuild key

    // runtime factory (app-owned policy already applied)
    BuildRuntime func(ctx context.Context, convID string) (engine.Engine, events.EventSink, error)
}
```

This is intentionally opinionated:
- `Fingerprint` is mandatory for deterministic reuse/rebuild.
- `BuildRuntime` closes over policy choices (system prompt, tools, middlewares, profile, tenant, etc.) so core never needs to parse them.
- `RuntimeOptions` remains request-scoped and is consumed by runtime execution stages (for example `step_mode`).

### 6) Split the current `overrides` map into two categories

A key source of confusion today is that `overrides` mixes composition and execution concerns.

The proposed split:

1. **Runtime composition inputs** (go into `Fingerprint` + `BuildRuntime`)
   - system prompt
   - middleware set and middleware config
   - tool allowlist / tool config
   - provider/model options if app chooses per-request model switching
2. **Runtime execution options** (stay as request data)
   - step mode toggle
   - debug flags
   - future control-plane options that affect execution flow but should not force runtime rebuild

This split removes accidental rebuilds and clarifies ownership.

### 7) Detailed pseudocode: new end-to-end flow

#### 7.1 Resolver (app layer)

```go
func (r *WebChatResolver) Resolve(req *http.Request) (*ConversationRequestPlan, error) {
    body := parseChatBodyOrWS(req)
    convID := resolveConvID(body, req)
    idem := resolveIdempotencyKey(req, body)

    // app policy dimension, not core concern
    profile := r.resolveProfile(req, body)

    // app composes effective runtime config (prompt/mw/tools/model/etc.)
    cfg := r.resolveRuntimeConfig(profile, body)
    fp := cfg.Fingerprint() // deterministic + secret-safe

    // request-time options (do not define runtime composition)
    runOpts := map[string]any{
        "step_mode": body.StepMode,
    }

    plan := &ConversationRequestPlan{
        ConvID:         convID,
        Prompt:         body.Prompt,
        IdempotencyKey: idem,
        RuntimeOptions: runOpts,
        RuntimeKey:     profile, // optional display/debug key
        Fingerprint:    fp,
        BuildRuntime: func(ctx context.Context, convID string) (engine.Engine, events.EventSink, error) {
            return r.runtimeFactory.Build(ctx, convID, cfg)
        },
    }
    return plan, nil
}
```

#### 7.2 Core handler

```go
func (h *ChatHandler) HandleChat(w http.ResponseWriter, req *http.Request) {
    plan, err := h.resolver.Resolve(req)
    if err != nil { writePolicyError(w, err); return }

    conv, err := h.cm.GetOrCreate(plan.ConvID, plan.Fingerprint, plan.RuntimeKey, plan.BuildRuntime)
    if err != nil { writeServerError(w, err); return }

    prep, err := conv.PrepareSessionInference(
        plan.IdempotencyKey,
        plan.Prompt,
        plan.RuntimeOptions,
    )
    if err != nil { writeServerError(w, err); return }
    if !prep.Start { writeQueued(w, prep); return }

    resp, err := h.startInference(conv, plan.Prompt, plan.RuntimeOptions, plan.IdempotencyKey)
    if err != nil { writeServerError(w, err); return }
    writeJSON(w, resp)
}
```

#### 7.3 Core conversation manager

```go
func (cm *ConvManager) GetOrCreate(
    convID string,
    fingerprint string,
    runtimeKey string,
    build BuildRuntimeFn,
) (*Conversation, error) {
    c := cm.lookup(convID)
    if c == nil {
        eng, sink, err := build(cm.baseCtx, convID)
        if err != nil { return nil, err }
        return cm.create(convID, fingerprint, runtimeKey, eng, sink), nil
    }

    if c.Fingerprint != fingerprint {
        eng, sink, err := build(cm.baseCtx, convID)
        if err != nil { return nil, err }
        c.replaceRuntime(eng, sink)
        c.Fingerprint = fingerprint
        c.RuntimeKey = runtimeKey
    }
    return c, nil
}
```

### 8) ASCII timeline diagram

```text
Client
  |
  | POST /chat (or WS attach)
  v
Webchat Core (generic transport/lifecycle)
  |
  | Resolve(req)
  v
App Resolver -----------------------------------------------+
  |                                                         |
  | build ConversationRequestPlan                           |
  |  - conv_id                                              |
  |  - prompt / idempotency_key / runtime_options           |
  |  - runtime_key / fingerprint                            |
  |  - buildRuntime closure                                 |
  |                                                         |
  +---------------------------------------------------------+
  |
  v
ConvManager.GetOrCreate(conv_id, fingerprint, runtime_key, buildRuntime)
  |
  | conversation exists?
  |   no  -> buildRuntime -> create runtime
  |   yes -> compare fingerprint
  |            same -> reuse runtime
  |            diff -> buildRuntime -> replace runtime
  v
PrepareSessionInference(idempotency, prompt, runtime_options)
  |
  | running?
  |   yes -> enqueue
  |   no  -> start inference
  v
Session/ToolLoop RunInference
  |
  v
EventSink -> StreamCoordinator -> WS connections / debug stream
```

### 9) Why this resolves the "system prompt in core" concern

With this shape:
- system prompt is never interpreted by core,
- profile selection is never interpreted by core,
- middleware/tool composition is never interpreted by core.

Those are encapsulated in resolver/runtime-factory logic owned by each app.

For `pinocchio/cmd/web-chat`, profile can still exist as an app feature.
For `web-agent-example`, no profile layer is needed at all.

Both apps share one core lifecycle implementation without forcing shared policy semantics.

### 10) Compatibility with current implementation and migration approach

A practical migration path without a flag day rewrite:

1. Introduce `ConversationRequestPlan` internally while keeping `EngineBuildInput`.
2. Add adapter resolver that maps old `BuildEngineFromReq + BuildConfig/BuildFromConfig` into plan.
3. Move `pinocchio/cmd/web-chat` to native resolver first.
4. Move `web-agent-example` to native resolver.
5. Remove old profile-centric fields and adapter code.

This allows iterative refactoring while preserving momentum, but given ticket preference for no long-lived compatibility, adapter phase should be short and removed within same ticket or immediate follow-up.

### 11) Tradeoffs and design consequences

Benefits:
- clear ownership boundaries,
- easier integration of new apps with non-profile policy axes,
- easier testing (resolver can be unit tested independent of core),
- reduced core API churn when product policy changes.

Costs:
- app boilerplate increases slightly (must implement resolver and runtime builder),
- initial refactor touches handler and conversation manager signatures,
- need to clearly define what belongs in `RuntimeOptions` versus fingerprinted config.

Given current code shape and the pain points already visible, this tradeoff is favorable.

### 12) Naming recommendation

If we keep "engine" in names, people will continue to think in provider terms instead of conversation lifecycle terms. Prefer conversation-centric naming for the external contract:

- `ConversationRequestResolver.Resolve(req)`
- `ConversationRequestPlan`
- `BuildRuntime(...)`
- `Fingerprint`

This aligns with what core actually does: it manages conversation runtime lifecycles, not provider-level request parsing policy.

## App Migration Map (pinocchio/cmd/web-chat and web-agent-example)

This section maps the concrete app-side changes needed once core exposes the new resolver-plan API. The intent is to make implementation parallelizable and remove ambiguity about ownership.

### A) `pinocchio/cmd/web-chat` migration map

`cmd/web-chat` keeps profile UX as an app feature, but profile behavior is no longer owned by `pkg/webchat`.

#### A.1 Backend wiring changes

1. `pinocchio/cmd/web-chat/main.go`
   - Remove `r.AddProfile(...)` calls.
   - Build an app-local profile registry/config source.
   - Construct and pass app resolver to router (new router option, for example `WithConversationRequestResolver(...)`).
   - Mount app-owned profile endpoints via `r.HandleFunc(...)`:
     - `GET /api/chat/profiles`
     - `GET/POST /api/chat/profile`
2. New app files under `pinocchio/cmd/web-chat/` (proposed):
   - `profiles.go` (profile model + registry)
   - `request_resolver.go` (request -> plan resolution)
   - `profile_handlers.go` (profile API handlers for UI)
   - `runtime_factory.go` (build runtime from resolved app config)

#### A.2 Resolver behavior for web-chat

The app resolver should:
1. Parse request body/query/path/cookie according to product policy.
2. Resolve selected profile slug from app policy order (path/query/cookie/default/existing conversation as desired).
3. Build effective runtime config:
   - system prompt
   - middleware list and configs
   - tool allowlist
   - model/provider options
4. Compute deterministic fingerprint from effective runtime config.
5. Return plan:
   - `ConvID`, `Prompt`, `IdempotencyKey`
   - `RuntimeOptions` (for execution-time options such as step mode)
   - `RuntimeKey` (human-friendly key, e.g. profile slug)
   - `Fingerprint`
   - `BuildRuntime(...)` closure

#### A.3 Frontend impact for web-chat

Because web-chat still has profile UX, frontend behavior remains largely unchanged; ownership changes.

Files to verify/update:
1. `pinocchio/cmd/web-chat/web/src/store/profileApi.ts`
   - Keep endpoints unchanged from frontend perspective.
   - Backend provider changes from core-owned to app-owned handlers.
2. `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
   - Keep profile selector behavior.
   - Ensure no assumptions remain about core-managed profile semantics.
3. `pinocchio/cmd/web-chat/web/src/store/store.ts`
   - Keep profile API reducer/middleware wiring.
4. Debug UI mapping updates for renamed runtime identity fields:
   - `pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts`
   - `pinocchio/cmd/web-chat/web/src/debug-ui/types/index.ts`
   - `pinocchio/cmd/web-chat/web/src/debug-ui/components/ConversationCard.tsx`
5. WS hello schema consumer updates if `profile` is removed/renamed:
   - `pinocchio/cmd/web-chat/web/src/sem/pb/proto/sem/base/ws_pb.ts`

### B) `web-agent-example` migration map

`web-agent-example` should be profile-free: one runtime policy, no profile APIs, no profile cookie coupling.

#### B.1 Backend wiring changes

1. `web-agent-example/cmd/web-agent-example/main.go`
   - Remove `r.AddProfile(...)` block completely.
   - Replace `WithEngineFromReqBuilder(...)` usage with resolver-plan equivalent.
   - Keep `WithEventSinkWrapper(...)` as-is (orthogonal concern).
2. `web-agent-example/cmd/web-agent-example/engine_from_req.go`
   - Replace with resolver-based file (proposed rename `request_resolver.go`).
   - Stop emitting profile-centric fields; emit runtime key/fingerprint semantics.
3. New/adjusted helper module (optional):
   - `runtime_factory.go` to centralize runtime composition for thinking/disco middleware.

#### B.2 Resolver behavior for web-agent-example

The app resolver should:
1. Resolve `ConvID`, `Prompt`, `IdempotencyKey`.
2. Build effective runtime config from app defaults + middleware overrides (`buildOverrides` inputs).
3. Compute fingerprint from effective runtime config.
4. Return plan with:
   - stable `Fingerprint`
   - optional `RuntimeKey` (could be `"default"` or empty)
   - `BuildRuntime(...)` closure
   - request data and runtime options

No profile endpoints or profile cookie behavior is required.

#### B.3 Frontend impact for web-agent-example

1. `web-agent-example/web/src/App.tsx`
   - Continue using `ChatWidget`, but disable/hide profile selector.
2. `@pwchat/webchat` integration strategy:
   - Preferred: add explicit prop for profile mode (for example `profileMode="none"` or `profilesEnabled={false}`).
   - Alternative: custom header/statusbar slots that omit selector.
3. No `profileApi` dependency should be required by this app.

### C) Cross-app sequencing and ownership

To minimize breakage and rework, execute app migrations in this order:

1. Introduce core resolver-plan interfaces and adapter glue (short-lived).
2. Migrate `pinocchio/cmd/web-chat` backend ownership of profile behavior.
3. Migrate `web-agent-example` to profile-free resolver.
4. Update frontend package/profile selector optionality.
5. Remove adapter and old profile-centric paths in core.

### D) Acceptance criteria by app

#### D.1 `pinocchio/cmd/web-chat`

1. Profile switching still works in UI.
2. `/api/chat/profile*` endpoints are served by app package, not `pkg/webchat`.
3. Core debug payloads use generic runtime identity naming.
4. Chat and WS flows reuse/rebuild runtime based on fingerprint only.

#### D.2 `web-agent-example`

1. App runs without any profile registry setup.
2. No dependency on `/api/chat/profile*`.
3. Chat widget UI works without profile selector.
4. Runtime rebuild behavior is fingerprint-driven and deterministic.

## Design Decisions

### Decision A: No backward compatibility layer

Per ticket direction, we should do a hard cut:
- remove profile APIs/types from core,
- update both known apps immediately (`web-chat`, `web-agent-example`),
- update docs/tests in same change series.

Rationale:
- avoids indefinite dual pathways,
- simplifies code and mental model quickly,
- this repo already controls both primary consumers.

### Decision B: Keep an opaque `EngineKey` in core (not `ProfileSlug`)

We still need a lightweight identity for:
- logging rebuild decisions,
- queued request tracking,
- debug visibility.

Using generic `EngineKey` preserves utility without profile semantics.

### Decision C: App owns profile endpoints

`/api/chat/profiles` and `/api/chat/profile` are product UX APIs, not core runtime APIs. They should be mounted by the app package via `r.HandleFunc(...)`.

### Decision D: Default core request builder should be neutral

Current default builder is profile-aware. Replace it with either:
1. a minimal neutral builder (conv ID + prompt body only), or
2. no default at all (constructor requires explicit builder).

Preferred: neutral default builder + explicit docs, so simple integrations still work.

### Decision E: Remove profile naming from protocol/debug payloads

`profile` fields in WS hello and debug APIs should become generic (`engine_key`) or be omitted if unused.

Rationale:
- avoids leaking deprecated concept into tooling,
- aligns payload naming with core abstraction.

### Decision F: Clean cutover to resolver-plan API and retire `BuildEngineFromReq`

This migration should be a clean cutover, not a long-lived dual API period.

Concretely:
1. Prefer the new broad, consistent interface:
   - `ConversationRequestResolver.Resolve(req) -> ConversationRequestPlan`
2. Treat `BuildEngineFromReq` as transitional only if needed for a very short bridge.
3. Remove `BuildEngineFromReq` from core once both in-repo app consumers are migrated:
   - `pinocchio/cmd/web-chat`
   - `web-agent-example`
4. Remove `WithEngineFromReqBuilder(...)` from the stable external surface if it no longer has active users.

Why this is important:
1. `BuildEngineFromReq` encodes a narrower and older shape (`ConvID/ProfileSlug/Overrides`) that keeps policy leakage alive.
2. The resolver-plan API better matches real lifecycle needs:
   - request data
   - runtime fingerprint
   - runtime construction closure
3. Maintaining both interfaces for long increases complexity, documentation drift, and ambiguity for new integrations.

Cutover policy:
1. Short bridge allowed only to land migrations safely in one change window.
2. No commitment to backward compatibility for the old builder model.
3. Documentation and examples should exclusively teach resolver-plan after cutover.

## Alternatives Considered

### Alternative 1: Keep profiles in core, just hide them

Rejected because:
- core still encodes app policy,
- `web-agent-example` still pays complexity tax,
- does not solve cookie/profile leakage.

### Alternative 2: Rename `ProfileSlug` to `Mode` but keep same architecture

Rejected because:
- this is semantic relabeling, not decoupling,
- still bakes one policy dimension into core.

### Alternative 3: Add more hooks but keep profile endpoints in core

Rejected because:
- duplicates ownership across core and apps,
- hard to reason about which layer is source of truth.

### Alternative 4: Move all config resolution to app and pass full `EngineConfig` in request builder output

Viable long-term, but larger immediate refactor (WS/chat paths, config caching, request options separation). The recommended plan keeps current shape with a generic key first, then optionally moves to full-config request outputs in a later cleanup.

## Implementation Plan

### Phase 0: Inventory and safety net

1. Add/adjust tests around request builder + conversation rebuild logic to validate generic engine-key behavior.
2. Capture current behavior in integration tests for:
   - `/chat`,
   - `/ws`,
   - queue/idempotency,
   - debug conversation listing.

### Phase 1: Core API refactor (`pinocchio/pkg/webchat`)

1. Remove profile types/registry from `types.go`.
2. Remove `WithProfileRegistry` from `router_options.go`.
3. Replace profile-centric fields:
   - `EngineBuildInput.ProfileSlug` -> `EngineBuildInput.EngineKey`
   - `Conversation.ProfileSlug` -> `Conversation.EngineKey`
   - `queuedChat.ProfileSlug` -> `queuedChat.EngineKey`
4. Update `conversation.go` rebuild comparison/logging to use engine key + config signature.
5. Update `router.go` handlers and logs to use `engine_key` naming.
6. Remove `/api/chat/profiles` and `/api/chat/profile` from core route registration.
7. Update debug routes payload shape (`profile` -> `engine_key`).
8. Update ws hello semantics:
   - replace/remove `profile` field usage in server emit path.
   - if protobuf is changed, regenerate Go/TS artifacts.

### Phase 2: App-level profile module in `pinocchio/cmd/web-chat`

1. Introduce local profile registry/type(s) in cmd package.
2. Implement app-specific `EngineFromReqBuilder`:
   - parse `/chat/{profile}` and `/ws?profile=...`,
   - optionally parse app cookie for UX persistence,
   - validate profile existence.
3. Provide app-owned handlers:
   - `/api/chat/profiles`
   - `/api/chat/profile`
4. Wire builder + handlers into router setup in `main.go`.
5. Keep profile UX in `cmd/web-chat/web` unchanged functionally, but endpoint ownership now lives in app layer.

### Phase 3: `web-agent-example` alignment

1. Remove `r.AddProfile(...)` usage from `web-agent-example/cmd/web-agent-example/main.go`.
2. Keep/update custom builder to emit a single fixed `EngineKey`.
3. Ensure web UI does not require profile endpoints:
   - disable profile selector in `@pwchat/webchat` for this app, or
   - inject a no-profile header/statusbar variant.
4. Validate end-to-end conversation flow with middleware overrides still works.

### Phase 4: Documentation and cleanup

1. Update docs that currently teach core profile APIs:
   - `pinocchio/pkg/doc/topics/webchat-framework-guide.md`
   - `pinocchio/pkg/doc/topics/webchat-user-guide.md`
   - `pinocchio/pkg/doc/tutorials/03-thirdparty-webchat-playbook.md`
2. Remove stale references to `chat_profile` cookie as core behavior.
3. Update debug UI field expectations from `profile` to generic engine identity.

### Phase 5: Verification matrix

1. `go test ./pinocchio/pkg/webchat/...`
2. `go test ./pinocchio/cmd/web-chat/...`
3. `go test ./web-agent-example/...`
4. Manual smoke:
   - `pinocchio web-chat` with profile switching works via app-owned handlers.
   - `web-agent-example` runs without profile APIs/cookies.
   - WS attach and chat POST both create/join same conversation.

## Detailed API Delta (Proposed)

### Core package (`pkg/webchat`)

Remove:
- `type Profile`
- `type ProfileRegistry`
- `func (r *Router) AddProfile(...)`
- `func WithProfileRegistry(...)`
- default profile endpoints in router.

Change:
- `EngineBuildInput` fields (profile -> generic engine key).
- conversation/debug payload fields (profile -> generic key).

### `pinocchio/cmd/web-chat`

Add:
- app-local profile registry module,
- app-local request-policy builder,
- app-local profile HTTP handlers.

### `web-agent-example`

Change:
- no dependency on core profile APIs,
- single-key builder only,
- UI profile control explicitly disabled/removed.

## Risks and Mitigations

1. Risk: frontend package currently assumes profile endpoints exist.
   - Mitigation: make profile UI optional; default to hidden unless provider/endpoints configured.

2. Risk: websocket hello/profile field compatibility break.
   - Mitigation: migrate consumers in same PR set; use generic `engine_key` field where needed.

3. Risk: hidden dependency in docs/tests/internal tools on `profile` naming.
   - Mitigation: repo-wide grep for `ProfileSlug`, `/api/chat/profile`, `chat_profile`, `WsHelloV1.profile`.

4. Risk: request builder complexity shifts to apps.
   - Mitigation: provide small reusable helper(s) in app package, not core runtime.

## Rollout Recommendation

Single coordinated refactor in one ticket branch is viable because there are two primary app consumers in-repo. Do not stage a long-lived compatibility shim.

Recommended commit slices:
1. Core type/API rename + endpoint removal.
2. `cmd/web-chat` profile module + endpoint wiring.
3. `web-agent-example` simplification + UI profile optionality.
4. Tests and docs update.

## Open Questions

1. Should core websocket hello carry any engine identity field at all, or remain minimal (`conv_id`, `server_time`)?
2. Should `EngineKey` be required or optional in `EngineBuildInput`?
3. Should request `overrides` be split into:
   - engine composition overrides,
   - runtime options (ex: step mode)?
4. Do we want a small shared helper package (outside core) for profile-style app policy to avoid duplication between apps?

## References

- `pinocchio/pkg/webchat/types.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/engine_from_req.go`
- `pinocchio/pkg/webchat/engine_builder.go`
- `pinocchio/pkg/webchat/conversation.go`
- `pinocchio/pkg/webchat/send_queue.go`
- `pinocchio/pkg/webchat/router_options.go`
- `pinocchio/pkg/webchat/router_debug_routes.go`
- `pinocchio/proto/sem/base/ws.proto`
- `pinocchio/cmd/web-chat/main.go`
- `web-agent-example/cmd/web-agent-example/main.go`
- `web-agent-example/cmd/web-agent-example/engine_from_req.go`
