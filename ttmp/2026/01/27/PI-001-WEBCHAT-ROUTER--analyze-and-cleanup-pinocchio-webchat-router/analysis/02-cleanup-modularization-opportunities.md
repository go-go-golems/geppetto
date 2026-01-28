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


# Cleanup and Modularization: A Design Analysis

> *"In software, as in writing, the greatest skill is knowing what to leave out. But before you can leave things out, you must understand what you have."*

---

## Executive Summary

The webchat system works well for its intended purpose, but it has accumulated structural complexity that makes it harder to reuse, embed, and extend. This document analyzes the current state and proposes incremental improvements.

**The highest-leverage opportunities are:**

1. **Unify mount/prefix semantics** — Currently split between Router, cmd/main.go, and frontend basePrefix detection
2. **Extract conversation lifecycle hooks** — Allow integrators to customize conversation creation without forking
3. **Decouple the frontend widget** — Package as an embeddable SDK with clean API boundaries
4. **Eliminate dead hydration code** — Choose one hydration path and remove the other

The goal is not to rewrite the system, but to make surgical improvements that increase clarity and enable new use cases.

---

## 1. Understanding Technical Debt

Technical debt, like financial debt, is not inherently bad. It becomes problematic when:
- It compounds silently
- It blocks new features
- It confuses new contributors

The webchat system has three categories of debt:

**Structural debt:** Responsibilities that should be separate are intertwined (e.g., `getOrCreateConv` does six different things).

**Naming debt:** Inconsistent terminology creates confusion (e.g., `run_id` vs `session_id` in API responses).

**Interface debt:** Missing extension points force integrators to fork rather than wrap.

Let us examine each area of concern.

---

## 2. The Mount/Prefix Problem

### 2.1 What Should Happen

When embedding the webchat router under a prefix like `/assistant`, all paths should work correctly:

```
/assistant/            → serve index.html
/assistant/assets/*    → serve static assets
/assistant/chat        → POST chat endpoint
/assistant/ws          → WebSocket upgrade
/assistant/timeline    → GET timeline snapshot
```

The frontend JavaScript, when it runs, should construct API URLs correctly regardless of the mount prefix.

### 2.2 What Actually Happens

Currently, **three different places** handle prefix logic:

**Place 1: `cmd/web-chat/main.go`**

```go
if root != "" {
    mux.Handle(root, http.StripPrefix(root, r.Handler()))
} else {
    mux.Handle("/", r.Handler())
}
```

The main.go manually applies `http.StripPrefix` when a prefix is configured.

**Place 2: `Router.Mount()` method**

```go
func (r *Router) Mount(mux *http.ServeMux, prefix string) {
    mux.Handle(prefix, r.mux)  // No StripPrefix!
}
```

This method exists but does **not** strip the prefix, making it incorrect for nested mounting.

**Place 3: Frontend basePrefix detection**

```typescript
// web/src/utils/basePrefix.ts
export function getBasePrefix(): string {
    // Heuristic: first path segment
    const path = window.location.pathname;
    const match = path.match(/^(\/[^/]+)/);
    return match ? match[1] : '';
}
```

The frontend guesses the prefix from the URL, which is fragile.

### 2.3 Why This Is Problematic

- **Integrators must know the "correct" way** to mount (use `StripPrefix` externally, not `Router.Mount`)
- **Frontend may guess wrong** if the URL structure is unusual
- **Documentation and code diverge** — which is the source of truth?
- **Testing is harder** — behavior depends on how the router is mounted

### 2.4 Proposed Solution: Router-Owned Base Path

Add a method that encapsulates the correct mounting pattern:

```go
// MountUnder attaches the router to a parent mux with correct prefix handling.
func (r *Router) MountUnder(mux *http.ServeMux, prefix string) {
    prefix = normalizePrefix(prefix)  // Ensure leading/trailing slashes
    handler := r.Handler()
    
    // Strip prefix before our handlers see the request
    stripped := http.StripPrefix(strings.TrimRight(prefix, "/"), handler)
    mux.Handle(prefix, stripped)
}

func normalizePrefix(p string) string {
    if p == "" || p == "/" {
        return "/"
    }
    if !strings.HasPrefix(p, "/") {
        p = "/" + p
    }
    if !strings.HasSuffix(p, "/") {
        p = p + "/"
    }
    return p
}
```

For the frontend, pass the base prefix explicitly via a `<script>` data attribute or environment variable injected at build time:

```html
<script>window.__WEBCHAT_BASE__ = "{{.BasePrefix}}";</script>
```

Then in TypeScript:

```typescript
export function getBasePrefix(): string {
    return (window as any).__WEBCHAT_BASE__ || '';
}
```

**Benefits:**
- Single authoritative place for prefix logic
- Explicit, not guessed
- Easy to test
- Documentation matches code

---

## 3. The Conversation Lifecycle Monolith

### 3.1 The Problem

`getOrCreateConv` is the most important function in the system. It is also the most complex. In 170 lines, it:

1. Builds engine configuration
2. Computes configuration signature
3. Locks and unlocks the ConvManager
4. Initializes semFrameBuffer and TimelineProjector
5. Detects profile/signature changes and decides whether to rebuild
6. Creates engine, sink, and subscriber
7. Stops/closes old stream coordinator
8. Creates new stream coordinator with callback
9. Starts the stream
10. Initializes the Session with turn history

This function is **private** and **not overridable**. Integrators who need different behavior must fork the entire file.

### 3.2 Why Integrators Need Different Behavior

Consider these legitimate use cases:

**Use case A: Fresh engine per request**

Some applications want a clean slate for each prompt—no conversation history, no engine reuse. Currently impossible without forking.

**Use case B: External conversation storage**

Large deployments may want conversations stored in Redis or a database, not an in-memory map. Currently requires rewriting `ConvManager`.

**Use case C: Custom session initialization**

Applications may want to inject system context or pre-populate turn history from an external source. Currently requires modifying `getOrCreateConv`.

### 3.3 Proposed Solution: ConversationFactory Interface

Introduce an interface that encapsulates conversation lifecycle decisions:

```go
type ConversationRequest struct {
    ConvID      string
    ProfileSlug string
    Overrides   map[string]any
}

type ConversationFactory interface {
    GetOrCreate(ctx context.Context, req ConversationRequest) (*Conversation, error)
}

// Router: allow override
func (r *Router) WithConversationFactory(f ConversationFactory) *Router {
    r.convFactory = f
    return r
}
```

The default implementation wraps the current logic:

```go
type DefaultConversationFactory struct {
    router *Router
}

func (f *DefaultConversationFactory) GetOrCreate(ctx context.Context, req ConversationRequest) (*Conversation, error) {
    // Current getOrCreateConv logic, extracted and cleaned
}
```

**Benefits:**
- Existing behavior unchanged (default factory)
- Integrators can wrap or replace
- Easier to test conversation lifecycle in isolation
- Opens path to distributed conversation stores

### 3.4 Splitting Responsibilities Further

For even cleaner architecture, consider separating:

**ConversationStore:** Pure storage/lookup of conversations

```go
type ConversationStore interface {
    Get(convID string) (*Conversation, bool)
    Put(conv *Conversation) error
    Delete(convID string) error
}
```

**ConversationLifecycle:** Engine/stream creation logic

```go
type ConversationLifecycle interface {
    Initialize(conv *Conversation, config EngineConfig) error
    Rebuild(conv *Conversation, config EngineConfig) error
    Shutdown(conv *Conversation) error
}
```

This separation follows the **Single Responsibility Principle** and makes each piece independently testable.

---

## 4. The Dual Hydration Problem

### 4.1 Current State

The codebase contains **two hydration mechanisms**:

**Mechanism 1: semFrameBuffer**

```go
// In Conversation struct
semBuf *semFrameBuffer
```

Frames are pushed to this buffer as they stream. It's never exposed via HTTP.

**Mechanism 2: TimelineStore**

```go
// GET /timeline?conv_id=...
snap, _ := r.timelineStore.GetSnapshot(ctx, convID, sinceVersion, limit)
```

The frontend uses this endpoint exclusively.

### 4.2 The Problem

- **semFrameBuffer is dead code** — it's populated but never read externally
- **Documentation mentions `/hydrate`** — which doesn't exist
- **Cognitive overhead** — developers must understand both paths to know which matters

### 4.3 Proposed Solution: Choose One

**Option A: Remove semFrameBuffer**

If timeline is the strategic direction (which it appears to be), delete:
- `semFrameBuffer` struct
- `semBuf` field in `Conversation`
- All code that pushes to `semBuf`

**Benefits:** Less code, clearer architecture, no confusion.

**Option B: Reintroduce /hydrate endpoint**

If short-term hydration (before timeline projection completes) is valuable:

```go
r.mux.HandleFunc("/hydrate", func(w http.ResponseWriter, r0 *http.Request) {
    convID := r0.URL.Query().Get("conv_id")
    conv, ok := r.cm.GetConversation(convID)
    if !ok {
        http.NotFound(w, r0)
        return
    }
    
    frames := conv.semBuf.Snapshot()
    json.NewEncoder(w).Encode(map[string]any{"frames": frames})
})
```

**Recommendation:** Option A. Timeline-based hydration is more robust and already works. Remove the unused mechanism.

---

## 5. Naming Inconsistencies

### 5.1 run_id vs session_id

In API responses, we return both:

```json
{
  "run_id": "abc-123",
  "session_id": "abc-123"
}
```

They are always the same value! This creates unnecessary confusion:
- Are they different concepts?
- Which should I use?
- What if they diverge in the future?

**Proposed fix:** Choose one canonical name. Suggestion: `session_id` (aligns with Geppetto's `Session` concept).

For backward compatibility, keep `run_id` as a deprecated alias for one release cycle:

```go
resp := map[string]any{
    "session_id": conv.RunID,
    "run_id":     conv.RunID,  // Deprecated: use session_id
}
```

### 5.2 Conversation vs ChatSession

The name `Conversation` is accurate for the concept (a persistent chat thread). However, the struct conflates two concerns:
- **Identity:** `ID`, `ProfileSlug`, `EngConfigSig`
- **Runtime state:** `Sess`, `Eng`, `Sink`, `stream`, `pool`

Consider whether the name should change based on which concern is primary. If the struct primarily represents runtime state, `ChatSession` might be clearer.

**Recommendation:** Keep `Conversation` but add comments clarifying that it represents both identity and runtime state.

### 5.3 EngineFromReqBuilder

This name describes the implementation, not the purpose. Consider:
- `RequestPolicy` — emphasizes that it's about interpreting requests
- `RequestResolver` — focuses on the resolution aspect
- `ChatRequestParser` — more specific

**Recommendation:** Rename to `RequestPolicy` to match the abstraction level.

---

## 6. The Router Does Too Much

### 6.1 Current Responsibilities

The `Router` struct owns:

1. **Event router** (Watermill pub/sub)
2. **HTTP mux** (request routing)
3. **Static file serving** (embedded FS)
4. **Profile registry** (configurations)
5. **Tool registry** (registered tools)
6. **Middleware registry** (registered middlewares)
7. **Conversation manager** (active conversations)
8. **Timeline store** (persistence)
9. **Request policy** (how to parse requests)
10. **Step controller** (debug stepping)
11. **Various runtime flags**

This violates the **Interface Segregation Principle**. Users who want only a subset must take the whole thing.

### 6.2 Proposed Modularization

**Option A: Modular constructors**

Introduce a minimal core and compose upward:

```go
// Core: just the event bus and conversation runtime
type RouterCore struct {
    baseCtx  context.Context
    router   *events.EventRouter
    cm       *ConvManager
}

func NewRouterCore(ctx context.Context, eventRouter *events.EventRouter) *RouterCore

// Full router: adds HTTP, static assets, profiles
type Router struct {
    *RouterCore
    mux       *http.ServeMux
    profiles  ProfileRegistry
    // ...
}

func NewRouter(ctx, parsed, staticFS) *Router {
    core := NewRouterCore(ctx, buildEventRouter(parsed))
    return &Router{RouterCore: core, ...}
}
```

**Benefits:**
- Reuse core in non-HTTP contexts (e.g., gRPC, CLI)
- Clearer separation of concerns
- Easier testing

**Option B: Sub-routers**

Split by API surface:

```go
type ChatAPI struct { /* handles POST /chat */ }
type WsAPI struct { /* handles /ws */ }
type TimelineAPI struct { /* handles /timeline */ }

type Router struct {
    chat     *ChatAPI
    ws       *WsAPI
    timeline *TimelineAPI
}
```

**Benefits:**
- Each API is independently testable
- Clear ownership of each endpoint

**Recommendation:** Start with Option A (modular constructors). It's less invasive and provides most of the benefit.

---

## 7. Packaging the Frontend as an NPM Widget

### 7.1 The Vision

External developers should be able to embed the webchat UI with a simple API:

```typescript
import { mountWebchat } from '@pinocchio/webchat-widget';

mountWebchat({
    target: '#chat-container',
    baseUrl: 'https://api.example.com/chat',
    profile: 'default',
    theme: 'dark',
    cssVars: {
        '--webchat-accent': '#ff5500',
    },
});
```

### 7.2 Required Refactoring

**Step 1: Extract SDK layer**

Create a small SDK that handles communication:

```typescript
// @pinocchio/webchat-sdk
export function connectWs(baseUrl: string, convId: string): WebSocket
export function postChat(baseUrl: string, payload: ChatPayload): Promise<ChatResponse>
export function getTimeline(baseUrl: string, convId: string): Promise<TimelineSnapshot>
```

This decouples network logic from UI components.

**Step 2: Parameterize baseUrl**

Currently, URLs are constructed from `window.location`:

```typescript
const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
const url = `${proto}://${window.location.host}${basePrefix}/ws`;
```

Change to accept explicit baseUrl:

```typescript
export function createWsUrl(baseUrl: string, convId: string): string {
    const parsed = new URL(baseUrl);
    parsed.protocol = parsed.protocol === 'https:' ? 'wss:' : 'ws:';
    parsed.pathname = parsed.pathname.replace(/\/$/, '') + '/ws';
    parsed.searchParams.set('conv_id', convId);
    return parsed.href;
}
```

**Step 3: Package structure**

```
@pinocchio/webchat-widget/
  dist/
    webchat.js       # UMD bundle
    webchat.esm.js   # ESM bundle
    webchat.css      # Styles
  src/
    index.ts         # Public API
    widget.tsx       # Main component
    sdk.ts           # Network layer
    theme.ts         # Theming system
  types/
    index.d.ts       # TypeScript declarations
```

### 7.3 Styling Strategy

**CSS Variables (recommended for first iteration):**

```css
:root {
    --webchat-bg: #ffffff;
    --webchat-text: #1a1a1a;
    --webchat-accent: #0066cc;
    --webchat-radius: 8px;
    --webchat-font: system-ui, sans-serif;
}
```

Consumers can override:

```css
:root {
    --webchat-accent: #ff5500;
}
```

**Shadow DOM (optional, for stricter encapsulation):**

Later, offer a Shadow DOM mode that prevents style leakage in either direction.

---

## 8. Documentation Drift

### 8.1 Known Mismatches

| Documentation Says | Reality |
|--------------------|---------|
| `/hydrate` endpoint exists | Only `/timeline` exists |
| `Router.Mount()` handles prefixes | It doesn't strip prefixes |
| Use `run_id` for session identity | Both `run_id` and `session_id` are returned |

### 8.2 Fixes

1. **Update docs to remove `/hydrate`** or implement it
2. **Deprecate `Router.Mount()`** in favor of `MountUnder()` (or fix it)
3. **Document canonical ID naming** in API reference

### 8.3 Living Documentation

Consider generating API documentation from code:
- OpenAPI spec from handler annotations
- TypeScript types generated from protobuf
- Examples extracted from tests

This reduces drift by making docs derive from source.

---

## 9. Module Layout Proposal

For long-term maintainability, consider reorganizing:

```
/webchat
  /router      # HTTP + WS endpoints, mounting
  /runtime     # Conversation, ConnectionPool, StreamCoordinator
  /engine      # EngineConfig, EngineBuilder, profiles
  /timeline    # TimelineProjector, TimelineStore implementations
  /sem         # SEM translator, registry integration
  /sdk         # (future) Go SDK for programmatic use
```

**Benefits:**
- Clear responsibility boundaries
- Each package testable in isolation
- Easier to understand for new contributors
- Natural places for new features

---

## 10. Incremental Cleanup Plan

Change should be incremental and safe. Here is a phased approach:

### Phase 1: Low Risk, High Clarity

**Effort:** 1-2 days

1. **Fix or replace `Router.Mount()`**
   - Add `MountUnder()` with correct StripPrefix
   - Deprecate `Mount()` with a comment
   
2. **Remove semFrameBuffer (or implement /hydrate)**
   - Audit all references
   - Delete if unused
   
3. **Normalize run_id/session_id**
   - Document which is canonical
   - Add deprecation comment to the other

4. **Update documentation**
   - Remove /hydrate mentions (or implement it)
   - Document mounting correctly

### Phase 2: Interface Extraction

**Effort:** 3-5 days

1. **Introduce `ConversationFactory` interface**
   - Extract current logic to `DefaultConversationFactory`
   - Add `WithConversationFactory()` to Router
   
2. **Rename `EngineFromReqBuilder` to `RequestPolicy`**
   - Search and replace
   - Update docs

3. **Add explicit baseUrl to frontend**
   - Inject via template or environment
   - Remove guessing logic

### Phase 3: Structural Improvements

**Effort:** 1-2 weeks

1. **Extract `RouterCore`**
   - Pull out non-HTTP concerns
   - Allow reuse in non-HTTP contexts
   
2. **Split sub-routers (optional)**
   - `ChatAPI`, `WsAPI`, `TimelineAPI`
   - Only if testing benefits justify complexity

3. **Package frontend widget**
   - Create SDK layer
   - Set up NPM package structure
   - Add styling API

---

## 11. Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| Remove semFrameBuffer | Low | Audit all references first |
| Add MountUnder() | Low | Additive; doesn't change existing behavior |
| ConversationFactory interface | Medium | Ensure default implementation matches current behavior exactly |
| Rename RequestPolicy | Medium | Search codebase for all references |
| Extract RouterCore | Medium-High | Integration tests essential |
| NPM widget packaging | Medium | Existing UI continues to work; widget is new code |

---

## 12. Success Criteria

How do we know the cleanup succeeded?

1. **New integrators can embed webchat in 5 minutes** — no forking required
2. **Mounting under a prefix works on first try** — single documented method
3. **Frontend widget can be added via npm install** — no copy-paste
4. **Each package has clear, testable responsibilities**
5. **Documentation matches reality** — no "but actually..." moments

---

## Conclusion

The webchat system is fundamentally sound. Its problems are not deep architectural flaws but accumulated complexity from organic growth. By systematically extracting interfaces, clarifying naming, and removing dead code, we can transform it from a working-but-coupled implementation into a reusable framework.

The key insight is that **good abstractions are discovered, not invented**. The current code reveals where the seams should be. Our job is to make those seams explicit.

---

## Appendix: Quick Reference

### Files to Modify by Phase

**Phase 1:**
- `router.go` — add `MountUnder()`, deprecate `Mount()`
- `conversation.go` — remove semBuf references (if chosen)
- `sem_buffer.go` — delete (if chosen)
- Documentation files

**Phase 2:**
- `conversation.go` — extract to `DefaultConversationFactory`
- `engine_from_req.go` — rename file and type
- Frontend `wsManager.ts`, `basePrefix.ts`

**Phase 3:**
- New file: `router_core.go`
- Split handlers into sub-packages
- New package: `@pinocchio/webchat-widget`

### Compatibility Checklist

Before each phase, verify:
- [ ] Existing webchat command works unchanged
- [ ] Frontend hydration works
- [ ] Profile switching works
- [ ] Queue/idempotency behavior unchanged
- [ ] Timeline persistence works
- [ ] No breaking changes to existing integrations
