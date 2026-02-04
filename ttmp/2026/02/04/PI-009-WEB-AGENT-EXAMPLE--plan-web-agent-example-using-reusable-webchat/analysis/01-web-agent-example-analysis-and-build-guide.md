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
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/sem/registry.ts
      Note: SEM event mapping to timeline entities
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx
      Note: UI composition and renderer overrides
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/types.ts
      Note: ChatWidget props
    - Path: ../../../../../../../pinocchio/pkg/inference/events/typed_thinking_mode.go
      Note: Typed thinking mode events emitted by middleware
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: Backend router composition + extension points
    - Path: ../../../../../../../pinocchio/pkg/webchat/sem_translator.go
      Note: SEM translation for thinking mode events
    - Path: ../../../../../../../pinocchio/pkg/webchat/timeline_projector.go
      Note: Timeline snapshot mapping for thinking_mode
ExternalSources: []
Summary: A detailed, intern-ready guide to build a new web-agent-example that reuses the Pinocchio webchat backend + frontend packaging, with a thinking-mode middleware and a custom thinking-mode switch/widget.
LastUpdated: 2026-02-04T16:18:16.334205935-05:00
WhatFor: Provide a code-grounded map of where to look, what to change, and how to wire a new agent around the reusable webchat stack.
WhenToUse: Use when implementing the web-agent-example server + UI, or when onboarding someone to the reusable webchat architecture.
---


# Web Agent Example Analysis and Build Guide

## Executive Summary

This document teaches a brand‑new engineer how to build a **standalone web agent** (the `web-agent-example` repo) by reusing the **Pinocchio webchat backend** and the newly modular **webchat frontend package**. The core goal is to **add a “thinking mode” middleware** on the backend and a **custom thinking‑mode widget + switch** on the frontend, without re‑implementing the chat stack. The guide is intentionally exhaustive: it names exact files, symbols, and data paths, and includes pseudo‑code, diagrams, callouts, and exercises.

> FUNDAMENTAL: Reuse is about seams.
> 
> The safest reuse points are where data changes form. In this architecture those seams are:
> 1) **Event emission** (Go events → SEM payloads),
> 2) **Timeline projection** (SEM events → durable timeline snapshots),
> 3) **Frontend mapping** (timeline snapshots → UI entities).

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

There is an existing “thinking mode” event pipeline. We can reuse it rather than invent a new UI protocol.

Backend event types:

- `pinocchio/pkg/inference/events/typed_thinking_mode.go`
  - `EventThinkingModeStarted`
  - `EventThinkingModeUpdate`
  - `EventThinkingModeCompleted`

Backend SEM translation:

- `pinocchio/pkg/webchat/sem_translator.go`
  - Emits `thinking.mode.started|update|completed`

Backend timeline projection:

- `pinocchio/pkg/webchat/timeline_projector.go`
  - Projects `thinking.mode.*` to `timeline.Kind = thinking_mode`

Frontend mapping and rendering:

- `pinocchio/cmd/web-chat/web/src/sem/registry.ts` (event → entity)
- `pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts` (snapshot → entity props)
- `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx` (`ThinkingModeCard`)

> FUNDAMENTAL: The “thinking mode” UI card appears automatically when the backend emits the corresponding SEM events.

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
    │                        Middleware emits events
    │                               │
    │                               ▼
    │        Event router + sem_translator (SEM frames)
    │                               │
    │                               ▼
    │                       Timeline projector (optional)
    │                               │
    ▼                               ▼
WebSocket stream ◄───────────── SEM frames + snapshots
    │
    ▼
Timeline store + UI renderers
```

### The key seams (where you hook in)

- **Middleware seam** (Go): emit `EventThinkingMode*`
- **UI renderer seam** (React): override the card for `thinking_mode`
- **UI composer seam** (React): add a “thinking mode switch” control
- **Request seam** (Go): accept a mode override in the request body

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
  - SEM bridging for `thinking.mode.*`
- `pinocchio/pkg/webchat/timeline_projector.go`
  - Snapshot mapping for `thinking_mode`

### Backend: Middleware/Event infrastructure

- `geppetto/pkg/inference/middleware/middleware.go`
  - The middleware type you must implement
- `geppetto/pkg/events/context.go`
  - `events.PublishEventToContext` for SEM dispatch
- `pinocchio/pkg/inference/events/typed_thinking_mode.go`
  - Typed thinking mode events you will emit
- Example middleware patterns:
  - `pinocchio/pkg/middlewares/agentmode/middleware.go`

### Frontend: Reusable UI package

- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
  - The composition root and `renderers` override map
- `pinocchio/cmd/web-chat/web/src/webchat/types.ts`
  - `ChatWidgetProps`, `ChatWidgetComponents`, `ChatWidgetRenderers`
- `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx`
  - Base cards (including `ThinkingModeCard`)
- `pinocchio/cmd/web-chat/web/src/sem/registry.ts`
  - Event → entity mapping (SEM → UI)
- `pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts`
  - Snapshot → entity mapping

### Reference Docs (read these first)

- `geppetto/ttmp/2026/02/02/PI-006-REUSABLE-WEBCHAT--reusable-webchat-modular-themable/design-doc/01-reusable-webchat-modular-themable-architecture-plan.md`
  - Explains the reusable `webchat/` package structure, CSS tokens, and slots
- `geppetto/ttmp/2026/01/25/GP-015-WEBCHAT-PACKAGE--webchat-packaging-reusable-npm-package/analysis/01-webchat-packaging-into-a-reusable-npm-package.md`
  - Deep packaging analysis and the intended public API shape
- `geppetto/ttmp/2026/02/02/PI-007-WEBCHAT-BACKEND-REFACTOR--webchat-backend-refactor/analysis/03-textbook-the-new-webchat-router.md`
  - Deep understanding of the router and request flow

## Build Plan for `web-agent-example`

Below is a minimal‑but‑complete plan for the new agent. It is written as if you will implement it from scratch in the `web-agent-example` repo.

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

#### 1.3 Wire configuration layers

Use the same parameter layer approach as `pinocchio/cmd/web-chat/main.go`:

- `addr`
- `timeline-dsn` / `timeline-db`
- `evict-idle-seconds`
- etc.

The goal is to keep compatibility with the existing router config interface, even if the `web-agent-example` binary only exposes a subset.

### Phase 2 — Thinking mode middleware (backend)

**Goal:** emit thinking‑mode events around inference so the UI sees “thinking mode” states.

#### 2.1 Choose the middleware insertion point

A geppetto middleware wraps the inference handler:

```go
type Middleware func(HandlerFunc) HandlerFunc
```

A thinking‑mode middleware can:

1. Emit `EventThinkingModeStarted` before calling the inner handler
2. Emit `EventThinkingModeUpdate` mid‑way (optional)
3. Emit `EventThinkingModeCompleted` on success or failure

#### 2.2 Use existing typed events

Use the typed events from:

- `pinocchio/pkg/inference/events/typed_thinking_mode.go`

Example pseudo‑code:

```go
func ThinkingModeMiddleware() middleware.Middleware {
  return func(next middleware.HandlerFunc) middleware.HandlerFunc {
    return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
      meta := events.NewEventMetadataFromTurn(t) // or build from t.ID + timestamps

      events.PublishEventToContext(ctx, events.NewThinkingModeStarted(meta, t.ID, &events.ThinkingModePayload{
        Mode: "deliberate",
        Phase: "planning",
        Reasoning: "Starting analysis...",
      }))

      out, err := next(ctx, t)

      if err != nil {
        events.PublishEventToContext(ctx, events.NewThinkingModeCompleted(meta, t.ID, nil, false, err.Error()))
        return out, err
      }

      events.PublishEventToContext(ctx, events.NewThinkingModeCompleted(meta, t.ID, nil, true, ""))
      return out, nil
    }
  }
}
```

> FUNDAMENTAL: **Events are the contract.** Once you emit `thinking.mode.*`, everything downstream already knows how to display it.

#### 2.3 Support “mode switching”

The UI should be able to set a user‑selected thinking mode. There are two clean options:

1. **Overrides in the chat request body**
   - The `ChatRequestBody` already includes `Overrides map[string]any`.
   - You can pass `overrides.thinking_mode = "fast"` from the UI.
2. **Separate HTTP endpoint**
   - Register your own route: `router.HandleFunc("/api/chat/thinking-mode", ...)`.
   - Store it in conversation state or a per‑session config.

If you use overrides, the thinking middleware can read them by attaching overrides to the `turns.Turn` metadata or by extending `EngineConfig` to store them.

Pseudocode for override extraction:

```go
// when building EngineConfig:
if mode, ok := overrides["thinking_mode"].(string); ok {
  cfg.Metadata["thinking_mode"] = mode
}
```

Then in middleware:

```go
mode := t.Metadata["thinking_mode"]
```

### Phase 3 — Custom thinking mode widget + switch (frontend)

**Goal:** present a UI control to switch thinking modes and render the timeline entity with your own visualization.

#### 3.1 Use the reusable ChatWidget package

Import from the package in the frontend:

```ts
import { ChatWidget, type ChatWidgetRenderers } from '@org/webchat-react'
```

In this workspace the module lives at:

- `pinocchio/cmd/web-chat/web/src/webchat`

So your frontend can use a local path alias or a workspace link (until the package is published).

#### 3.2 Override the renderer for thinking mode

The `ChatWidget` accepts a `renderers` map keyed by entity kind:

```ts
const renderers: ChatWidgetRenderers = {
  thinking_mode: ThinkingModeSwitchCard,
};

<ChatWidget renderers={renderers} />
```

Your `ThinkingModeSwitchCard` can be a brand‑new component, or a composition around the default `ThinkingModeCard` from `webchat/cards.tsx`.

#### 3.3 Add a thinking mode switch in the Composer

Override the Composer slot so you can add a dropdown or toggle:

```ts
const ThinkingModeComposer = (props: ComposerSlotProps) => {
  const [mode, setMode] = useState('fast');

  return (
    <div>
      <select value={mode} onChange={(e) => setMode(e.target.value)}>
        <option value="fast">Fast</option>
        <option value="deliberate">Deliberate</option>
      </select>
      <DefaultComposer {...props} />
    </div>
  );
};

<ChatWidget components={{ Composer: ThinkingModeComposer }} />
```

#### 3.4 Wire the selected mode into the request

You need to ensure the selected mode is included in the POST payload.

Two options:

1. **Extend ChatWidget** (preferred)
   - Add a prop to `ChatWidgetProps` like `buildOverrides?: () => Record<string, any>`.
   - Use it when building the POST body:

```ts
const payload = {
  conv_id: app.convId || convIdFromLocation(),
  prompt: text,
  overrides: buildOverrides?.(),
};
```

2. **Wrap ChatWidget**
   - Fork ChatWidget into your app and keep the same interface.
   - This is more work but avoids changing the reusable package.

> FUNDAMENTAL: A UI switch is useless unless you serialize its state.
> 
> The contract boundary for that state is the **chat request payload**.

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

- [ ] Add a `ThinkingModeMiddleware` in `web-agent-example` (new package)
- [ ] Register it with `Router.RegisterMiddleware("thinking-mode", ...)`
- [ ] Decide how the user‑selected mode is stored and passed (overrides vs endpoint)
- [ ] Emit `EventThinkingModeStarted/Completed` around inference
- [ ] Verify that `thinking.mode.*` frames appear over the websocket

### Frontend checklist

- [ ] Build a small React app that imports `ChatWidget`
- [ ] Override the `thinking_mode` renderer with a custom card
- [ ] Override the Composer to add a dropdown/toggle
- [ ] Ensure selected mode is serialized in the POST request
- [ ] Build into `static/dist` for embedding

## Worked Pseudocode: Full Loop

```
func handleUserMessage(prompt string, mode string) {
  // UI sends: { prompt, overrides: { thinking_mode: mode } }

  // Backend: BuildConfig sees overrides.thinking_mode
  cfg.Metadata["thinking_mode"] = mode

  // Middleware reads cfg.Metadata + emits SEM events
  emit thinking.mode.started (mode, phase)
  run inference
  emit thinking.mode.completed (success)

  // Frontend receives timeline entity
  kind = thinking_mode → render custom card
}
```

## Exercises and Quizzes

### Exercise 1 — Event tracing

Find the line in `pinocchio/pkg/webchat/sem_translator.go` where a `thinking.mode.started` SEM frame is created. Write down the struct type being serialized and list its fields.

### Exercise 2 — Timeline mapping

In `pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts`, identify how `thinking_mode` is mapped to entity props. What fields are used to compute success/error?

### Exercise 3 — Custom renderer

Write a new React component `MyThinkingModeCard` that renders `mode`, `phase`, and `status` in a compact banner. Then wire it into `ChatWidget renderers` in a minimal `App.tsx`.

### Quiz (short answers)

1. What function builds the webchat router in the backend?
2. Which type represents the HTTP request body for chat messages?
3. Where does the UI map SEM events into timeline entities?
4. What is the main file that embeds the web UI static assets?

## Appendix: Why this architecture is stable

- **Events are typed**: They don’t leak UI concerns into inference logic.
- **Timeline projection is pure**: events → snapshots → stable UI behavior.
- **ChatWidget is composable**: you can override small pieces without forking the entire UI.
- **Static assets are embedded**: no runtime asset server needed for the Go binary.

If you follow this guide, the new `web-agent-example` can focus on *new behavior* (thinking modes) rather than re‑creating existing infrastructure.
