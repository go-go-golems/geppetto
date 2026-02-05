---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/router.go
      Note: /chat and /ws entrypoints
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: SEM mapping patterns
    - Path: pinocchio/pkg/webchat/timeline_registry.go
      Note: timeline handler registration
    - Path: web-agent-example/pkg/thinkingmode/middleware.go
      Note: reference middleware pattern
    - Path: web-agent-example/web/src/sem/registerWebAgentSem.ts
      Note: frontend SEM registration
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Disco Internal Dialogue Middleware + Widget

## Goal

Create a middleware + UI widget + SEM protobuf integration that renders an internal “personality dialogue” (inspired by Disco Elysium) where multiple personality facets debate before the assistant responds.

This should be a first-class webchat experience:

- streaming dialogue as SEM events
- durable timeline projection for hydration
- configurable personas and pacing
- a dedicated widget that looks and feels like an internal debate (not a single linear message)

---

## Constraints

- Must integrate with existing Pinocchio webchat router + SEM registry + timeline store.
- Must be configurable via middleware overrides from the UI.
- Should support streaming events (started/line/update/completed) so the UI shows real-time conversation fragments.

---

## High-Level Architecture

```
User prompt
   ↓
Webchat /chat → EngineBuilder + Middlewares
   ↓
DiscoDialogueMiddleware
   ↓ emits SEM events
Sem translator → WS frames
   ↓
Timeline projection (durable) ← sem.tl upserts
   ↓
React widget renders personality debate
```

Key components:

1. **Middleware** (Go) in `web-agent-example/pkg/discodialogue` (or pinocchio pkg if shared)
2. **SEM protobuf** definitions + registry
3. **Timeline projection handlers** to persist snapshots
4. **React widget** to render the debate
5. **Webchat store integration** to register new SEM entity kind

---

## Event Model (SEM + Protobuf)

We need explicit events for the internal dialogue lifecycle. Suggested SEM event types:

- `disco.dialogue.started`
- `disco.dialogue.line`
- `disco.dialogue.updated`
- `disco.dialogue.completed`

These should map to a protobuf payload that includes:

- `conv_id`
- `dialogue_id`
- `line_id`
- `persona` (e.g., “Empathy”, “Logic”, “Instinct”)
- `text`
- `tone`
- `timestamp`
- `status`

### Proposed Protobuf (new file)

`pinocchio/pkg/sem/pb/proto/sem/middleware/disco_dialogue.proto`

```proto
syntax = "proto3";
package sem.middleware;

message DiscoDialogueLineV1 {
  string dialogue_id = 1;
  string line_id = 2;
  string persona = 3;
  string tone = 4;
  string text = 5;
  int64 timestamp_ms = 6;
  string status = 7; // started|updated|completed
}

message DiscoDialogueEventV1 {
  string conv_id = 1;
  DiscoDialogueLineV1 line = 2;
}
```

### SEM Event Envelope

The middleware emits SEM frames like:

```json
{
  "sem": true,
  "event": {
    "type": "disco.dialogue.line",
    "id": "<dialogue_id>:<line_id>",
    "data": { /* protojson */ }
  }
}
```

---

## Timeline Projection

We need a new entity kind in the timeline store, e.g.:

- `disco_dialogue`

Entity fields (protojson in snapshot):

- `dialogue_id`
- `persona`
- `tone`
- `text`
- `status`
- `updated_at_ms`

We will add a timeline projection handler similar to the existing `thinkingmode` handler:

- Register handler in `pinocchio/pkg/webchat/timeline_registry.go`
- On `disco.dialogue.*` events, upsert a `disco_dialogue` entity

---

## Middleware Behavior

### Input

From overrides:

```json
{
  "middlewares": [
    {
      "name": "disco-dialogue",
      "config": {
        "personas": ["Logic", "Empathy", "Paranoia"],
        "max_lines": 4,
        "pace_ms": 120,
        "tone": "noir",
        "style": "disco"
      }
    }
  ]
}
```

### Behavior

1. When a new user prompt starts, emit `disco.dialogue.started` for each persona.
2. Stream `disco.dialogue.line` events as the internal debate progresses.
3. Optionally emit `disco.dialogue.updated` when lines are revised mid-stream.
4. Emit `disco.dialogue.completed` when the debate closes.

### Integration Points

- Hook into inference lifecycle similar to `thinkingmode` middleware.
- Use an internal generator to produce lines before final assistant response.
- Provide deterministic IDs so timeline updates are stable.

---

## Widget / UI

### Visual Layout

- A stacked “internal voices” column above the assistant response.
- Each persona rendered as a card/badge with distinct color/typography.
- Lines appear as streaming text (fade-in, typewriter, or slide-in).

### Component Name

- `DiscoDialogueCard` (new component in `web-agent-example/web/src/components/`)

### Store Integration

- Register SEM handlers similar to `registerWebAgentSem`:
  - `registerDiscoDialogueSem()`
  - Map event type to entity kind `disco_dialogue`

### Props (example)

```ts
interface DiscoDialogueEntry {
  id: string;
  persona: string;
  tone?: string;
  text: string;
  status: "started" | "updated" | "completed";
}
```

---

## Implementation Plan (Detailed)

### Phase 1 — Protobuf + SEM

1. Add proto definition in `pinocchio/pkg/sem/pb/proto/sem/middleware/`.
2. Regenerate protobuf Go code.
3. Register new SEM event type in the registry.

### Phase 2 — Middleware

1. Create package `web-agent-example/pkg/discodialogue`.
2. Implement middleware that emits SEM events.
3. Add config parsing + defaults.

### Phase 3 — Timeline Projection

1. Add timeline handler for new event types.
2. Ensure snapshot hydration stores `disco_dialogue` entities.

### Phase 4 — Frontend Widget

1. Add `DiscoDialogueCard` component.
2. Add SEM mapping + entity type.
3. Extend `ChatWidget` renderers to include the new card.

### Phase 5 — Wiring + Demo

1. Register middleware in `web-agent-example` server.
2. Update UI to send middleware overrides by default.
3. Add a demo prompt + screenshot in docs.

---

## Risks / Open Questions

- How far do we want to simulate “internal debate” vs. strictly real reasoning?
- Do we want a deterministic persona list or should it be model-driven?
- Should the dialogue stream be visible to the model or purely UI-only?

---

## Suggested Tasks (for ticket)

1. Protobuf + SEM event definitions
2. Disco dialogue middleware (Go)
3. Timeline projection handler
4. Frontend widget + SEM registration
5. Wiring and demo test

---

## Related Files

- `pinocchio/pkg/webchat/router.go` — WS and /chat entrypoints
- `pinocchio/pkg/webchat/sem_translator.go` — SEM event mapping
- `pinocchio/pkg/webchat/timeline_registry.go` — projection hooks
- `web-agent-example/pkg/thinkingmode/*` — reference for middleware + events
- `web-agent-example/web/src/sem/registerWebAgentSem.ts` — SEM registration pattern
- `web-agent-example/web/src/components/*` — widget patterns
