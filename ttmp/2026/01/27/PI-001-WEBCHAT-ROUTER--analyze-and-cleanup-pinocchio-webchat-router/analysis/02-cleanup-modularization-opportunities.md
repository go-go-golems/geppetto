---
Title: Cleanup & Modularization Opportunities
Ticket: PI-001-WEBCHAT-ROUTER
Status: active
Topics:
    - analysis
    - webchat
    - refactor
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/main.go
      Note: |-
        root mounting and current setup example
        Root prefix mounting pattern
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: |-
        frontend hydration and basePrefix logic
        Hydration contract
    - Path: ../../../../../../../pinocchio/pkg/doc/topics/webchat-framework-guide.md
      Note: |-
        current documentation and API promises
        Doc contract for mounting and API
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
      Note: |-
        getOrCreateConv lifecycle and engine rebuild logic
        Lifecycle coupling
    - Path: ../../../../../../../pinocchio/pkg/webchat/engine_from_req.go
      Note: |-
        profile/conv policy logic (candidate for override)
        Request policy candidates for override
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: |-
        HTTP/WS endpoints, mount, request policy hooks
        Mounting and handler responsibilities
ExternalSources: []
Summary: Analysis of technical debt and design options to simplify and modularize webchat routing and integration.
LastUpdated: 2026-01-27T19:55:00-05:00
WhatFor: Guide future cleanup and packaging work for webchat.
WhenToUse: Before refactors or extracting a reusable webchat module/widget.
---


# Cleanup & Modularization Opportunities

## Executive Summary

The webchat system is functionally strong but structurally busy: responsibilities are centralized in `Router`, internal state is split between `Conversation` and the run loop, and documentation/front-end conventions are drifting from the server implementation. The highest-leverage improvements are:

1. **Unify mount/prefix semantics** across router, CLI, and frontend.
2. **Extract request policy and conversation lifecycle hooks** for integrators.
3. **Decouple frontend widget from server specifics** via a small SDK + NPM package.
4. **Reduce duplicate hydration logic** (SEM buffer vs timeline snapshots).

The rest of this document provides concrete cleanup paths and design options.

---

## 1. Areas That Feel Messy or Inconsistent

### 1.1 Mounting Under a Prefix (server vs router)

**Symptoms**
- `Router.Mount` exists but does not strip prefixes.
- `cmd/web-chat/main.go` manually strips the prefix via `http.StripPrefix`.
- `web/src/utils/basePrefix.ts` guesses the prefix as `/<first path segment>`.

**Why it is messy**
- There is no single authoritative place for prefix logic.
- Relative asset paths and `index.html` behavior depend on how the router is mounted.

**Potential consequences**
- Off-by-one prefix errors when embedding in other servers.
- Frontend paths diverge from backend expectations.

**Cleanup options**

Option A (Router-owned base path):
- Add `Router.WithBasePath(prefix)` or `Router.MountUnder(mux, prefix)` that always does `StripPrefix`.
- Router registers all handlers with `basePath` awareness, and `Handler()` becomes prefix-aware.

Option B (Frontend-owned base path):
- Keep router agnostic, but ship a small `basePrefix` detection function in the frontend SDK.
- Document strict mounting pattern in one place, and delete `Router.Mount` (or make it call the documented pattern).

**Recommendation**: Option A. It centralizes correctness and makes embedding safer.

---

### 1.2 Conversation Lifecycle Hard-coded in `getOrCreateConv`

**Symptoms**
- `getOrCreateConv` does engine build, subscriber build, queue init, stream creation, timeline projector, and rebuild decisions in one method.
- The function is private and not overridable.

**Why it is messy**
- Integrators who want custom conv reuse policies (e.g., “new engine per run” or “preserve certain sessions”) must fork.
- It tightly couples engine config signature logic to the router.

**Cleanup options**

Option A (Hook interface):
- Introduce `ConversationFactory` or `ConversationPolicy` interface on Router.
- Default implementation uses current logic.
- Allow inject via `Router.WithConversationFactory(...)`.

Option B (Split responsibilities):
- Separate `ConvManager` into two layers:
  - `ConversationStore` (lookup, create, save)
  - `ConversationLifecycle` (engine build, stream start)

Option C (Expose getOrCreate as a strategy in EngineFromReqBuilder):
- Request policy returns a `ConversationPlan` that decides reuse/rebuild.

**Recommendation**: Option A. It enables overrides without rewriting the router.

---

### 1.3 Dual Hydration Mechanisms (SEM buffer vs Timeline store)

**Symptoms**
- Conversations contain `semBuf`, but no HTTP endpoint currently uses it.
- Frontend hydrates via `/timeline` and ignores sem buffer.
- Documentation mentions `/hydrate`, which does not exist.

**Why it is messy**
- Multiple hydration pathways exist but only one is operational.
- The unreferenced sem buffer is either dead code or a missing feature.

**Cleanup options**

Option A (Remove sem buffer):
- Delete `semFrameBuffer` and related code.
- Keep `/timeline` as the only hydration path.

Option B (Reintroduce /hydrate using sem buffer):
- Add a `/hydrate` endpoint that returns buffered frames in SEM format.
- Use `semBuf.Snapshot()` for quick short-term hydration.
- Keep `/timeline` for durable long-term replay.

**Recommendation**: Choose one and document it. If timeline is the strategic direction, remove sem buffer.

---

### 1.4 Mixed naming conventions (conv_id, run_id, session_id)

**Symptoms**
- Responses include `run_id` and `session_id` interchangeably.
- Internal code uses `run_id` but sessions are also called `SessionID`.

**Why it is messy**
- API clients must guess which ID is stable.
- Adds accidental complexity to debugging.

**Cleanup options**

Option A: Standardize on `session_id` in public JSON.
Option B: Standardize on `run_id` but explicitly define it as a session.

**Recommendation**: Pick one and update response shaping + docs consistently.

---

### 1.5 Single Router type doing too much

**Symptoms**
- Router owns event router, HTTP mux, profile registry, tool/middleware registries, conversation manager, timeline store, request policy, and debug endpoints.

**Why it is messy**
- Makes reuse harder; forced to use Router even if you want only a subset.

**Cleanup options**

Option A (Modular constructors):
- Introduce `NewRouterCore(...)` returning a struct with only core wiring.
- `NewRouter` composes core + HTTP + static assets.

Option B (Split into sub-routers):
- `ChatAPI` (HTTP POST)
- `WsAPI` (WebSocket join)
- `TimelineAPI` (hydration)
- A top-level `Router` composes these.

**Recommendation**: Option A. It preserves existing API while enabling reuse.

---

## 2. How to Mount Under a Prefix (Clean Design)

**Current situation**
- `cmd/web-chat/main.go` manually uses `http.StripPrefix` when `--root` is set.
- `Router.Mount` simply does `mux.Handle(prefix, r.mux)` (no strip), which would break paths.

**Proposed standard behavior**

```
func (r *Router) MountUnder(mux *http.ServeMux, prefix string) {
  prefix = normalize(prefix) // ensure leading + trailing slash
  mux.Handle(prefix, http.StripPrefix(strings.TrimRight(prefix, "/"), r.Handler()))
}
```

**Benefits**
- Single canonical mounting helper.
- Frontend paths remain consistent.
- Removes the “two places to do it” bug class.

---

## 3. Overriding getOrCreateConv (Customization Strategy)

**What integrators might want**
- Create a fresh `Session` per chat request (no conversation reuse).
- Use a different engine per request or per profile.
- Inject custom conversation metadata or storage.

**Minimal interface proposal**

```
type ConversationFactory interface {
  GetOrCreate(ctx context.Context, req ConversationRequest) (*Conversation, error)
}

// Router: allow override
func (r *Router) WithConversationFactory(f ConversationFactory) *Router
```

**Default implementation**
- Keep existing logic, but make it the default factory.

**Why this helps**
- Extensible without forking.
- Integrators can wrap current logic (e.g., instrumentation, caching, custom TTLs).

---

## 4. Packaging the Web App as an NPM Widget

### 4.1 Goals
- Embed a minimal webchat widget into any site.
- Support CSS customization without editing source.
- Provide a stable API for base URL, profile, and conv id.

### 4.2 Suggested Package Shape

**Package layout (monorepo or standalone)**

```
@pinocchio/webchat-widget
  /dist
    webchat.js
    webchat.css
  /src
    index.ts
    widget.tsx
    theme.css
  /types
    index.d.ts
```

**Embedding API**

```
import { mountWebchat } from '@pinocchio/webchat-widget';

mountWebchat({
  target: '#chat',
  baseUrl: 'https://example.com/chat',
  profile: 'default',
  theme: 'dark',
  cssVars: {
    '--webchat-accent': '#ff5500',
  },
});
```

### 4.3 Styling Strategy

Option A: CSS variables (simplest)
- Expose a small set of CSS variables in `:root`.
- Allow overrides through `cssVars` option.

Option B: Shadow DOM
- Encapsulate styles with Shadow DOM.
- Provide explicit API for theme overrides.

Option C: CSS modules + theme tokens
- Export a `theme` object used by the widget; external CSS is applied via CSS variables.

**Recommendation**: Option A for first iteration, add Shadow DOM as an opt-in later.

### 4.4 Webchat SDK Layer

Create a tiny “SDK” package that handles:
- `connectWs(baseUrl, convId)`
- `postChat(baseUrl, payload)`
- `getTimeline(baseUrl, convId)`

This decouples UI and makes the widget portable.

---

## 5. Other Renaming and Structure Improvements

### 5.1 Names to simplify
- `Conversation` -> `ChatSession` (if it truly represents one run) or keep `Conversation` but ensure it survives multiple runs.
- `RunID` vs `SessionID`: pick one canonical name.
- `EngineFromReqBuilder`: consider `RequestPolicy`.

### 5.2 Module layout

Potential module split:

```
/webchat
  /router (HTTP + WS endpoints)
  /runtime (conversation, stream, pool)
  /engine (build + config)
  /timeline (projector + store)
  /sem (translator + registry helpers)
```

This makes responsibilities visible and easier to test.

---

## 6. Documentation Gaps and Drift

Observed mismatches:
- Docs mention `/hydrate` while frontend uses `/timeline` and backend only implements `/timeline`.
- `Router.Mount` exists but the documented pattern uses `StripPrefix` externally.

**Actionable fixes**
- Update docs to remove `/hydrate` or reintroduce it.
- Delete or fix `Router.Mount` to match documentation.

---

## 7. Recommended Incremental Cleanup Plan

**Phase 1 (low risk)**
- Fix `Router.Mount` or replace with `MountUnder`.
- Align docs with actual endpoints.
- Remove or reintroduce sem buffer (choose one).

**Phase 2 (medium risk)**
- Introduce `ConversationFactory` interface and `RequestPolicy` naming.
- Normalize `run_id` vs `session_id` in responses.

**Phase 3 (larger)**
- Extract frontend widget into NPM package + SDK.
- Split Router into subcomponents for reuse.

---

## 8. Summary

The webchat system is close to a reusable framework, but it needs clearer seams: explicit mounting behavior, configurable conversation lifecycle, and a better separation between backend core and frontend widget. Addressing these items will make it easier to embed in other applications and to maintain over time.
