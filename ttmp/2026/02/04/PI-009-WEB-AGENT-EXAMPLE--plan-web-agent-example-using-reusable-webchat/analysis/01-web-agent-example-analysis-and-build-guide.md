---
Title: Web Agent Example Analysis and Build Guide
Ticket: PI-009-WEB-AGENT-EXAMPLE
Status: active
Topics:
    - webchat
    - frontend
    - backend
    - agent
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go
      Note: Backend router composition + extension points
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/sem_translator.go
      Note: SEM translation for thinking mode events
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/timeline_projector.go
      Note: Timeline snapshot mapping for thinking_mode
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/inference/events/typed_thinking_mode.go
      Note: Typed thinking mode events emitted by middleware
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx
      Note: UI composition and renderer overrides
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/webchat/types.ts
      Note: ChatWidget props, slots, and renderer types
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/sem/registry.ts
      Note: SEM event mapping to timeline entities
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts
      Note: Redux timeline source of truth used by ChatWidget
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/proto/sem/middleware/thinking_mode.proto
      Note: Protobuf contract for Go to TS SEM payload boundary
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/web/src/sem/registerWebAgentSem.ts
      Note: Web-agent SEM handlers decode protobuf payloads with fromJson before Redux upsert
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/pkg/thinkingmode/sem_test.go
      Note: Go boundary check for protobuf encode/decode round-trip
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/pkg/discodialogue/sem_test.go
      Note: Go boundary check for disco dialogue protobuf encode/decode round-trip
ExternalSources: []
Summary: "A detailed, intern-ready guide to build a new web-agent-example that reuses the Pinocchio webchat backend + frontend packaging, with Redux state ownership in the frontend and protobuf-defined SEM contracts at the Go/TS boundary."
LastUpdated: 2026-02-04T16:18:16.334205935-05:00
WhatFor: "Provide a code-grounded map of where to look, what to change, and how to wire a new web-agent around the reusable webchat stack, including Redux-first state handling, protobuf-backed SEM contracts, and custom middleware/UI."
WhenToUse: "Use when implementing the web-agent-example server + UI, or when onboarding someone to the reusable webchat architecture."
---

# Web Agent Example Analysis and Build Guide

## Executive Summary

This document teaches a brand-new engineer how to build a **standalone web agent** (the `web-agent-example` repo) by reusing the **Pinocchio webchat backend** and the modular **webchat frontend package**. The core goal is to **add a custom thinking-mode middleware and event set** on the backend, plus a **custom ThinkingModeCard and thinking-mode switch UI** on the frontend. The guide is intentionally exhaustive: it names exact files, symbols, and data paths, and includes pseudo-code, diagrams, callouts, and exercises.

> FUNDAMENTAL: Reuse is about seams.
> 
> The safest reuse points are where data changes form. In this architecture those seams are:
> 1) **Event emission** (Go events → SEM payloads),
> 2) **Timeline projection** (SEM events → durable timeline snapshots),
> 3) **Frontend mapping** (timeline snapshots → UI entities).

## Non-Negotiable Requirements (PI-009 Update)

The ticket now has three explicit constraints that should drive all implementation choices:

1. **Redux is the state authority in the web-agent frontend**
   - The UI must rely on the existing `@pwchat` Redux store (`timelineSlice`, app slice) as the durable runtime state.
   - Component-local state is allowed only for transient controls (for example an unsubmitted dropdown value).
   - Do not introduce parallel global state stores for timeline entities.
2. **Protobuf defines the Go/TS SEM boundary**
   - Custom middleware SEM payloads must be declared in protobuf under `pinocchio/proto/sem/middleware/*`.
   - Go should serialize SEM payloads from generated protobuf types.
   - TS should decode those payloads using generated schemas (`fromJson`) before projecting into entities.
3. **The guide itself must reflect these constraints**
   - Every custom middleware section should name the protobuf contract and TS decoder path.
   - Every frontend section should state Redux ownership and where entities are written.

### Boundary contract pattern (Go → WS JSON → TS)

```go
// Go: emit protobuf-backed SEM payload
msg := &semMw.ThinkingModeStarted{ItemId: ev.ItemID, Data: data}
raw, _ := protojson.Marshal(msg)
frame := map[string]any{
  "sem": true,
  "event": map[string]any{
    "type": string(EventThinkingStarted),
    "id":   ev.ItemID,
    "data": json.RawMessage(raw),
  },
}
```

```ts
// TS: decode protobuf payload, then dispatch Redux update
const data = decodeProto<ThinkingModeStarted>(ThinkingModeStartedSchema, ev.data);
dispatch(timelineSlice.actions.upsertEntity({
  id: ev.id,
  kind: 'webagent_thinking_mode',
  createdAt: Date.now(),
  props: { mode: data?.data?.mode, phase: data?.data?.phase, status: 'started' },
}));
```

Boundary checks now live in Go tests:

- `web-agent-example/pkg/thinkingmode/sem_test.go`
- `web-agent-example/pkg/discodialogue/sem_test.go`

## What Already Exists (You Are Reusing)

### 1) Webchat frontend is already modular and reusable

The reusable React package lives in the Pinocchio frontend at:

- `pinocchio/cmd/web-chat/web/src/webchat/`

Key exports:

- `ChatWidget` (root component)
- `ChatWidgetRenderers` (map of entity kind → card component)
- `ChatWidgetComponents` (slot overrides for header/status/composer)
- `ThinkingModeCard` (default card for `thinking_mode` entities)
- Theme tokens and parts in `webchat/styles/`

State ownership in this package is Redux-first. `ChatWidget` writes entities through Redux actions and reads entities through selectors, with `timelineSlice` as the canonical source of timeline truth.

Where to read:

- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/types.ts`
- `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/styles/webchat.css`
- `pinocchio/cmd/web-chat/web/src/webchat/styles/theme-default.css`

### 2) Webchat backend is reusable as a library

The backend is a package (not just a binary) at:

- `pinocchio/pkg/webchat`

It already exposes a composable router and server:

- `webchat.NewRouter(ctx, parsedLayers, staticFS)`
- `Router.RegisterMiddleware(name, factory)`
- `Router.RegisterTool(name, factory)`
- `Router.AddProfile(profile)`
- `Router.BuildHTTPServer()`
- `webchat.NewServer(ctx, parsedLayers, staticFS)`

Concrete example of assembly is in:

- `pinocchio/cmd/web-chat/main.go`

> FUNDAMENTAL: “Reusable backend” means **you compose it**, not just run it.
> 
> You choose which middlewares/tools/profiles to register, and you decide whether to embed a UI or only serve API + websocket routes.

### 3) Thinking mode is already wired end‑to‑end (events → UI)

There is an existing “thinking mode” event pipeline. We will **not** reuse its default card or default event types; instead we’ll **fork the pipeline** with our own event names and custom UI card while still leveraging the same structural seams (events → SEM → timeline → UI).

Existing baseline (for reference only):

- Typed events: `pinocchio/pkg/inference/events/typed_thinking_mode.go`
- SEM translation: `pinocchio/pkg/webchat/sem_translator.go`
- Timeline projection: `pinocchio/pkg/webchat/timeline_projector.go`
- UI mapping: `pinocchio/cmd/web-chat/web/src/sem/registry.ts`
- UI card: `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx`

For new middleware features, the SEM payload contract should be protobuf-first (`pinocchio/proto/sem/middleware/*`) so Go emission and TS decode stay in lockstep.

> FUNDAMENTAL: You can add your own event types and cards without touching the rest of the webchat stack.

## Core Data Flow (End‑to‑End Diagram)

```
User Input (Browser)
    │
    ▼
ChatWidget (React) ──POST /chat──► Webchat Router (Go)
    │                               │
    │                               ▼
    │                         Conversation + Engine
    │                               │
    │                               ▼
    │                Custom middleware emits custom events
    │                               │
    │                               ▼
    │        Custom SEM translation + timeline projection
    │                               │
    ▼                               ▼
WebSocket stream ◄───────────── SEM frames + snapshots
    │
    ▼
Timeline store + custom UI card
```

### The key seams (where you hook in)

- **Middleware seam** (Go): emit *custom* `thinking_mode.*` events
- **SEM seam** (Go): translate your custom events into SEM frames
- **Timeline seam** (Go): project your custom SEM frames into timeline snapshots
- **UI renderer seam** (React): render a custom ThinkingModeCard
- **UI composer seam** (React): add a thinking‑mode switch control
- **Request seam** (Go/React): send “current thinking mode” in the request payload

## Where to Look (Annotated Map)

### Backend: Reusable server/routers

- `pinocchio/pkg/webchat/router.go`
  - `NewRouter` (construction)
  - `Router.RegisterMiddleware`, `Router.RegisterTool`, `Router.AddProfile`
  - `registerAPIHandlers` (HTTP endpoints)
  - `registerUIHandlers` (static assets)
- `pinocchio/pkg/webchat/server.go`
  - `NewServer`, `Server.Run` (lifecycle)
- `pinocchio/pkg/webchat/conversation.go`
  - `Conversation` state, `ConvManager.GetOrCreate`
- `pinocchio/pkg/webchat/engine_from_req.go`
  - `ChatRequestBody` includes `Overrides map[string]any`
- `pinocchio/pkg/webchat/sem_translator.go`
  - Existing SEM bridging patterns (you will add your own branch)
- `pinocchio/pkg/webchat/timeline_projector.go`
  - Existing snapshot mapping patterns (you will add your own branch)

### Backend: Middleware/Event infrastructure

- `geppetto/pkg/inference/middleware/middleware.go`
  - The middleware type you must implement
- `geppetto/pkg/events/context.go`
  - `events.PublishEventToContext` for SEM dispatch
- **You will add new event types** in `web-agent-example` (not in Pinocchio)

### Frontend: Reusable UI package

- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
  - The composition root and `renderers` override map
- `pinocchio/cmd/web-chat/web/src/webchat/types.ts`
  - `ChatWidgetProps`, `ChatWidgetComponents`, `ChatWidgetRenderers`
- `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx`
  - Base cards (you will write your own thinking mode card instead)
- `pinocchio/cmd/web-chat/web/src/sem/registry.ts`
  - Event → entity mapping (SEM → UI)
- `pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts`
  - Snapshot → entity mapping

### Reference Docs (read these first)

- `geppetto/ttmp/2026/02/02/PI-006-REUSABLE-WEBCHAT--reusable-webchat-modular-themable/design-doc/01-reusable-webchat-modular-themable-architecture-plan.md`
- `geppetto/ttmp/2026/01/25/GP-015-WEBCHAT-PACKAGE--webchat-packaging-reusable-npm-package/analysis/01-webchat-packaging-into-a-reusable-npm-package.md`
- `geppetto/ttmp/2026/02/02/PI-007-WEBCHAT-BACKEND-REFACTOR--webchat-backend-refactor/analysis/03-textbook-the-new-webchat-router.md`

## Build Plan for `web-agent-example`

Below is a minimal‑but‑complete plan for the new agent, with explicit guidance for **custom events** and a **custom ThinkingModeCard**.

### Phase 1 — Boot the reusable backend

**Goal:** get a running Go server using `pinocchio/pkg/webchat`.

#### 1.1 Add server entrypoint

Edit `web-agent-example/cmd/web-agent-example/main.go` and create a server akin to `pinocchio/cmd/web-chat/main.go`:

```go
// pseudo-code
func main() {
  ctx := context.Background()

  parsed := buildLayers()  // use geppetto layers + webchat params
  staticFS := embedStatic() // embed built web UI under /static

  srv, _ := webchat.NewServer(ctx, parsed, staticFS)
  r := srv.Router()

  // register middleware/tools/profiles here
  r.RegisterMiddleware("thinking-mode", NewThinkingModeMiddleware)

  // run
  _ = srv.Run(ctx)
}
```

Symbols to use:

- `webchat.NewServer`
- `Router.RegisterMiddleware`
- `Router.RegisterTool`
- `Router.AddProfile`

#### 1.2 Embed the UI assets

Follow the pattern in `pinocchio/cmd/web-chat/main.go`:

```go
//go:embed static
var staticFS embed.FS
```

Your `web-agent-example` repo should include a `static/` directory with the same structure used by the webchat frontend build:

```
web-agent-example/
  cmd/web-agent-example/main.go
  static/
    index.html
    dist/
      assets/
        ... (Vite build output)
```

You can build these assets using a small `web/` frontend project that imports the reusable `ChatWidget` package (see Phase 3).

> FUNDAMENTAL: The backend does not care if the UI is React, Vue, or raw HTML.
> 
> It only serves files out of the embedded `static` filesystem.

### Phase 2 — Custom thinking mode events + middleware (backend)

**Goal:** emit **your own** thinking‑mode event types (not the built‑in ones), then translate those into SEM frames and timeline snapshots.

#### 2.1 Define your custom events

Create a new package in `web-agent-example` for custom events, for example:

```
web-agent-example/pkg/events/thinkingmode
```

Define your own event types and event factory registration. Example (pseudo‑code):

```go
// pkg/events/thinkingmode/events.go
package thinkingmode

type ThinkingModePayload struct {
  Mode string
  Phase string
  Reason string
  Extra map[string]any
}

type ThinkingModeStarted struct { ... }
// register factory: events.RegisterEventFactory("webagent.thinking.started", ...)
```

Use a unique event namespace to avoid collisions:

- `webagent.thinking.started`
- `webagent.thinking.update`
- `webagent.thinking.completed`

> FUNDAMENTAL: Your custom event types are the stable contract for this agent.

#### 2.2 Implement the middleware

A geppetto middleware wraps the inference handler:

```go
type Middleware func(HandlerFunc) HandlerFunc
```

Implement your own middleware that emits your custom events:

```go
func ThinkingModeMiddleware() middleware.Middleware {
  return func(next middleware.HandlerFunc) middleware.HandlerFunc {
    return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
      meta := events.NewEventMetadataFromTurn(t)

      events.PublishEventToContext(ctx, thinkingmode.NewStarted(meta, t.ID, &thinkingmode.Payload{...}))
      out, err := next(ctx, t)
      if err != nil {
        events.PublishEventToContext(ctx, thinkingmode.NewCompleted(meta, t.ID, nil, false, err.Error()))
        return out, err
      }
      events.PublishEventToContext(ctx, thinkingmode.NewCompleted(meta, t.ID, nil, true, ""))
      return out, nil
    }
  }
}
```

#### 2.3 Translate your custom events into SEM

You need a SEM translator hook similar to `pinocchio/pkg/webchat/sem_translator.go`. There are two options:

1. **Add a new translation branch** in your own code that registers with the SEM registry.
2. **Wrap the existing translator** and extend it before you build the router.

The minimum behavior: for each custom event, emit a SEM frame whose `type` is something like:

- `webagent.thinking.started`
- `webagent.thinking.update`
- `webagent.thinking.completed`

This allows the frontend `sem/registry.ts` to map these events to a timeline entity.

#### 2.4 Project timeline snapshots

In the timeline projector, add a case that consumes your custom SEM event types and writes a timeline entity of kind `webagent_thinking_mode` (or keep `thinking_mode` if you want to reuse the slot name).

Example intent:

```
case "webagent.thinking.started":
  upsert kind="webagent_thinking_mode" with status="started"
```

> FUNDAMENTAL: The timeline entity **kind** is how the UI decides what card to render.

### Phase 3 — Custom ThinkingModeCard + switch UI (frontend)

**Goal:** render a custom card and add a thinking‑mode switch to the composer.

#### 3.1 Build a custom ThinkingModeCard

Create your own card in the `web-agent-example` frontend, not in the Pinocchio shared package:

```tsx
export function WebAgentThinkingModeCard({ e }: { e: RenderEntity }) {
  return (
    <div className="webagent-thinking-card">
      <div className="header">Mode: {String(e.props?.mode ?? '')}</div>
      <div className="phase">Phase: {String(e.props?.phase ?? '')}</div>
      <div className="reason">{String(e.props?.reasoning ?? '')}</div>
    </div>
  );
}
```

Then register it via the `renderers` prop:

```tsx
const renderers: ChatWidgetRenderers = {
  webagent_thinking_mode: WebAgentThinkingModeCard,
};

<ChatWidget renderers={renderers} />
```

#### 3.2 Add a thinking‑mode switch to the composer

Override the Composer slot so you can add your control:

```tsx
const ThinkingModeComposer = (props: ComposerSlotProps) => {
  const [mode, setMode] = useState('fast');

  return (
    <div>
      <label>
        Thinking Mode
        <select value={mode} onChange={(e) => setMode(e.target.value)}>
          <option value="fast">Fast</option>
          <option value="deliberate">Deliberate</option>
        </select>
      </label>
      <DefaultComposer {...props} />
    </div>
  );
};
```

#### 3.3 Serialize the selected mode in the request

Your custom composer must write the selected mode into the POST payload. You can do that in two ways:

1. **Extend ChatWidget** (preferred): add a `buildOverrides` callback prop and use it to build `overrides`.
2. **Wrap ChatWidget**: clone ChatWidget into your app and modify the send logic.

Payload example:

```ts
const payload = {
  conv_id: app.convId || convIdFromLocation(),
  prompt: text,
  overrides: { thinking_mode: mode },
};
```

> FUNDAMENTAL: The UI switch is only real if it gets serialized into the request.

### Phase 4 — Packaging the UI in `web-agent-example`

**Goal:** produce `static/` assets that the Go server embeds.

Recommended structure inside `web-agent-example`:

```
web-agent-example/
  web/
    src/
      App.tsx  // imports ChatWidget, overrides renderer & composer
    package.json
    vite.config.ts
  static/
    index.html
    dist/
      assets/...
```

Build step (run from `web/`):

```
npm run build
```

Then copy the output to `static/dist` at repo root. The Go binary will embed it at build time.

## Implementation Checklist (Intern‑Level Detail)

### Backend checklist

- [ ] Create **custom thinking mode event types** in `web-agent-example/pkg/events/...`
- [ ] Register event factories with unique type names (`webagent.thinking.*`)
- [ ] Implement **custom thinking mode middleware** that emits these events
- [ ] Extend SEM translation to handle your new event types
- [ ] Extend timeline projection to map your event types to a custom entity kind
- [ ] Verify SEM frames arrive over websocket for your custom events

### Frontend checklist

- [ ] Build a small React app that imports `ChatWidget`
- [ ] Create a **custom ThinkingModeCard** and register it under your custom entity kind
- [ ] Override Composer to add a thinking‑mode switch
- [ ] Ensure selected mode is serialized in the POST request
- [ ] Build into `static/dist` for embedding

## Worked Pseudocode: Full Loop

```
func handleUserMessage(prompt string, mode string) {
  // UI sends: { prompt, overrides: { thinking_mode: mode } }

  // Backend: BuildConfig sees overrides.thinking_mode
  cfg.Metadata["thinking_mode"] = mode

  // Middleware emits custom events
  emit webagent.thinking.started
  run inference
  emit webagent.thinking.completed

  // Timeline projects to kind: webagent_thinking_mode
  // UI renders WebAgentThinkingModeCard
}
```

## Exercises and Quizzes

### Exercise 1 — Event definition

Define a new event type `webagent.thinking.started` with a payload that includes `mode`, `phase`, and `reason`. Register the factory and write a unit test that serializes and re‑hydrates it.

### Exercise 2 — Timeline mapping

Add a new timeline mapping branch that turns `webagent.thinking.*` SEM frames into `webagent_thinking_mode` entities. Confirm that a `status` field is set correctly.

### Exercise 3 — Custom renderer

Write `WebAgentThinkingModeCard` and register it in `ChatWidget renderers`. Confirm the card appears when you fake a timeline entity in Storybook.

### Quiz (short answers)

1. Where will you register the custom thinking mode event factories?
2. Which file handles the POST request payload in the backend?
3. What determines the renderer used for a timeline entity in the UI?
4. Why do you need a custom entity kind if you want a custom card?

## Appendix: Why this architecture is stable

- **Events are typed**: You can create custom event types without touching the engine.
- **Timeline projection is pure**: events → snapshots → stable UI behavior.
- **ChatWidget is composable**: you can override renderers and slots without forking the UI.
- **Static assets are embedded**: no runtime asset server needed for the Go binary.

If you follow this guide, the new `web-agent-example` can focus on *new behavior* (custom thinking modes + UI) rather than re‑creating existing infrastructure.
