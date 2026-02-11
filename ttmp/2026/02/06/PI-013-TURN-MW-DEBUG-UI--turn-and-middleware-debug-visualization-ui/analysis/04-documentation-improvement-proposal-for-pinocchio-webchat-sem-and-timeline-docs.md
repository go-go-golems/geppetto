---
Title: Documentation Improvement Proposal for Pinocchio Webchat SEM and Timeline Docs
Ticket: PI-013-TURN-MW-DEBUG-UI
Status: active
Topics:
    - websocket
    - middleware
    - turns
    - events
    - frontend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/doc/topics/webchat-sem-and-ui.md
      Note: Primary SEM event format, handler registration, and widget mapping doc
    - Path: pinocchio/pkg/doc/topics/webchat-overview.md
      Note: Documentation index and architecture overview
    - Path: pinocchio/pkg/doc/topics/webchat-backend-reference.md
      Note: StreamCoordinator and ConnectionPool API reference
    - Path: pinocchio/pkg/doc/topics/webchat-backend-internals.md
      Note: Implementation deep-dive for streaming infrastructure
    - Path: pinocchio/pkg/doc/topics/webchat-frontend-integration.md
      Note: WebSocket, Redux, and HTTP integration patterns
    - Path: pinocchio/pkg/doc/topics/webchat-framework-guide.md
      Note: End-to-end framework usage guide
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Event-to-SEM translation with stable ID resolution and handler registration
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: SEM-to-timeline projection with throttling, planning aggregation, role memory
    - Path: pinocchio/pkg/webchat/timeline_store.go
      Note: Durable timeline persistence interface with version-based snapshots
    - Path: pinocchio/pkg/sem/registry/registry.go
      Note: Generic type-safe event handler registry using reflection
    - Path: geppetto/pkg/doc/topics/04-events.md
      Note: Geppetto event system doc that should cross-reference SEM translation
    - Path: go-go-mento/docs/reference/webchat/sem-and-widgets.md
      Note: go-go-mento SEM reference with content that should be ported to pinocchio
    - Path: go-go-mento/docs/reference/webchat/engine-builder.md
      Note: go-go-mento EngineBuilder reference with content that should be ported to pinocchio
    - Path: go-go-mento/docs/reference/webchat/inference-state.md
      Note: go-go-mento InferenceState reference with content that should be ported to pinocchio
ExternalSources: []
Summary: Proposes specific improvements to pinocchio/pkg/doc/topics/webchat-*.md documents to fill gaps around SEM event translation, timeline projection internals, widget registration patterns, the end-to-end event pipeline, and content to port from go-go-mento.
LastUpdated: 2026-02-07T00:15:00-05:00
WhatFor: Guide improvements to pinocchio webchat documentation so that developers can understand and extend the full event-to-UI pipeline.
WhenToUse: Reference when planning documentation sprints for the pinocchio webchat layer, when onboarding developers who need to add new event types or widgets, or when porting remaining content from go-go-mento.
---


# Documentation Improvement Proposal for Pinocchio Webchat SEM and Timeline Docs

## Purpose

The pinocchio webchat documentation (`pinocchio/pkg/doc/topics/webchat-*.md`) covers the streaming infrastructure, backend API, and frontend integration reasonably well. However, several critical pieces of the event-to-UI pipeline are undocumented or only partially documented. Additionally, the go-go-mento repository contains reference documentation (EngineBuilder, InferenceState, detailed SEM-to-widget walkthrough) that was never ported to pinocchio.

This proposal identifies specific gaps and suggests concrete additions. It is a companion to the earlier [Documentation Improvement Proposal for Geppetto Core Docs](03-documentation-improvement-proposal-for-geppetto-core-docs.md), which addressed the geppetto layer. Together they cover the full stack from inference events to UI rendering.

## Audience

- Documentation authors working on pinocchio/webchat docs
- Developers who need to add new SEM event types, timeline entities, or widgets
- Anyone onboarding to the webchat system and finding the existing docs incomplete

## General principles

1. **Document the pipeline, not just the components.** The webchat system is a pipeline: geppetto events → SEM translation → WebSocket → timeline projection → persistence → Redux → widget rendering. Each doc covers one component, but no doc shows the full pipeline in one place.
2. **Port valuable content from go-go-mento.** The go-go-mento docs contain detailed references (EngineBuilder, InferenceState, identity context) that don't exist in pinocchio. Port the concepts, adapted for pinocchio's code paths.
3. **Make "how to extend" a first-class concern.** The most common developer task is adding a new event type end-to-end. This should be a standalone tutorial, not scattered hints across six documents.
4. **Surface hidden behaviors.** The timeline projector has non-obvious behaviors (250ms write throttling, planning aggregation, role memory, stable ID resolution) that are critical for debugging but completely undocumented.

---

## New document proposals

### N1: End-to-end "Adding a New Event Type" tutorial (HIGH)

**Why this is needed.**

Adding a new event type touches 6+ files across 3 layers (geppetto, pinocchio backend, frontend). Currently a developer must read parts of sem_translator.go, timeline_projector.go, the proto definitions, the frontend SEM handlers, and the widget registry to piece together the full flow. No single document walks through this.

The go-go-mento `sem-and-widgets.md` has an "Adding a New Widget" section, but it covers only the frontend half. The backend half (SEM translation, projector case, proto message) is not documented anywhere.

**Proposed document:** `webchat-adding-event-types.md` (new topic doc)

**Suggested content outline:**

1. **Overview** — The 7-step pipeline for a new event type (diagram)
2. **Step 1: Define the geppetto event** — Create an event type in `geppetto/pkg/events/`, implement the `Event` interface
3. **Step 2: Define the protobuf message** — Add a proto message in `pinocchio/proto/sem/base/` or `pinocchio/proto/sem/timeline/`, run code generation
4. **Step 3: Register the SEM handler** — Use `semregistry.RegisterByType[*EventType]` in `sem_translator.go`, construct the SEM frame with `wrapSem` + `protoToRaw`
5. **Step 4: Add the timeline projector case** — Add a case in `TimelineProjector.ApplySemFrame()`, choose entity kind, construct snapshot proto
6. **Step 5: Create the frontend SEM handler** — Register via `registerSem('event.type', handler)` in `sem/handlers/`, return `AddCmd` or `UpsertCmd`
7. **Step 6: Create the widget** — Build the React component, call `registerWidgetRenderer(kind, renderer, options)`, choose visibility mode
8. **Step 7: Wire the imports** — Ensure the handler file is imported in the SEM index and the widget is imported in registerAll
9. **Checklist** — Verification steps (WS debug mode, Redux devtools, Storybook story)

**Priority:** HIGH — this is the single most-requested missing doc for developers extending the system.

**Effort:** ~300-400 lines. Requires reading the actual code paths and writing concrete examples.

---

### N2: Timeline Projector reference (HIGH)

**Why this is needed.**

`timeline_projector.go` is a 650-line file with critical non-obvious behaviors that are documented nowhere:

- **Write throttling**: `llm.delta` events are throttled to 250ms minimum between DB writes. This means the DB state can lag behind the in-memory state during fast streaming.
- **Role memory**: The projector stores the role from `llm.start` and applies it to all subsequent `llm.delta` events for the same message ID. If the start event is missed, deltas have no role.
- **Planning aggregation**: Planning events (`planning.start`, `planning.iteration`, `planning.reflection`, `planning.complete`, `execution.start`, `execution.complete`) are aggregated in-memory into a single `planningAgg` struct, then the full snapshot is rebuilt and persisted on every event. Iterations are sorted by index.
- **Stable ID resolution**: The translator resolves stable message IDs using a three-tier fallback (metadata.ID → cached inference/turn/session ID → generated fallback). IDs are cached to maintain consistency across streaming events and cleared on `EventFinal` to prevent memory leaks.
- **Thinking message ID suffix**: Thinking events append `:thinking` to the base message ID to create separate entities from the main assistant message.
- **Tool result split**: Each `tool.result` creates two entities: a completion update for the tool_call entity and a separate tool_result entity.
- **Custom tool handling**: "calc" tool results get `CustomKind: "calc_result"` for special widget rendering.
- **Version/seq semantics**: The `version` passed to the timeline store is the SEM frame's `Seq` value, not an internally-generated counter.

None of this is documented in any existing doc. The `webchat-backend-internals.md` mentions timeline projection in its performance table but doesn't explain any of these behaviors.

**Proposed document:** New section in `webchat-backend-internals.md` or standalone `webchat-timeline-projector.md`

**Suggested content outline:**

1. **Purpose** — Transform ephemeral SEM frames into durable, version-tracked timeline entities
2. **Architecture** — Input (SEM frame JSON), output (TimelineEntityV1 protobuf), store interface
3. **SEM frame to entity mapping table** — Complete mapping of event types to entity kinds
4. **Write throttling** — Why 250ms, what it means for DB vs memory state, impact on hydration
5. **Planning aggregation** — State machine diagram, `planningAgg` lifecycle, iteration sorting
6. **ID resolution and caching** — Three-tier fallback, cache lifecycle, thinking suffix, tool result split
7. **Version semantics** — How `Seq` flows through to `version`, monotonic guarantees
8. **Custom handlers** — `handleTimelineHandlers()` extension point

**Priority:** HIGH — critical for anyone debugging timeline persistence issues.

**Effort:** ~200-300 lines.

---

### N3: Cross-layer bridge section in geppetto `04-events.md` (MEDIUM)

**Why this is needed.**

The geppetto `04-events.md` explains the event system thoroughly but stops at "events go to sinks." It never mentions that in the webchat context, these events get translated into SEM frames. A reader of the geppetto docs has no idea that the event system connects to a UI rendering pipeline.

**Proposed addition:** A new section "Where Events Go: The SEM Translation Layer" at the end of `04-events.md`, just before "See Also".

**Suggested content:**

```markdown
## Where Events Go: The SEM Translation Layer

In the webchat application (pinocchio), geppetto events flow through an additional
translation step before reaching the UI. The pipeline:

1. Middleware and engines emit events via `events.Publish(ctx, event)`.
2. The event sink (typically a Watermill sink) publishes to a message bus.
3. A StreamCoordinator subscribes to the conversation's topic.
4. For each event, the **SEM translator** (`sem_translator.go`) converts it to a
   normalized JSON frame using a type-based handler registry.
5. SEM frames are broadcast to connected WebSocket clients.
6. In parallel, a **timeline projector** converts SEM frames to durable timeline
   entity snapshots persisted via a TimelineStore.

This means every geppetto event type has a potential second life as a UI-facing event.
When designing new event types, consider:

- Does this event need a visual representation? If so, register a SEM handler.
- Should the UI see this event in real-time (streaming) or only after completion?
- Does the event carry data that should be persisted in the timeline?

See the pinocchio webchat documentation for details:
- [Webchat SEM and UI](pinocchio/pkg/doc/topics/webchat-sem-and-ui.md)
- [Webchat Backend Internals](pinocchio/pkg/doc/topics/webchat-backend-internals.md)
```

**Priority:** MEDIUM — connects the two doc layers, but readers can discover the SEM docs independently.

**Effort:** ~30-50 lines added to existing doc.

---

## Improvements to existing documents

### E1: Add "What Is Webchat?" elevator pitch to `webchat-overview.md` (MEDIUM)

**Current state:** The overview jumps straight into "Quick Start" without explaining what webchat is or what problem it solves.

**What is missing:** A 2-3 paragraph introduction explaining:
- Webchat is a framework for building web-based LLM interfaces using the geppetto inference pipeline
- It provides a streaming event pipeline from backend inference to frontend timeline widgets
- It supports profiles, middleware composition, tool registration, and durable timeline persistence

**Effort:** ~20-30 lines.

### E2: Add SEM frame payload examples to `webchat-sem-and-ui.md` (MEDIUM)

**Current state:** Shows the envelope format but not complete payload examples for complex events like `tool.start` (with structured input), `planning.iteration` (with nested fields), or `thinking.mode.started`.

**What is missing:** 3-4 complete JSON frame examples showing real payloads with actual field values, plus brief explanation of each field.

**Effort:** ~60-80 lines.

### E3: Add debugging section to `webchat-sem-and-ui.md` (LOWER)

**Current state:** No debugging guidance. The go-go-mento version has a "Debugging Checklist" section with `?ws_debug=1` flag and step-by-step troubleshooting.

**What is missing:** Port the debugging checklist from go-go-mento's `sem-and-widgets.md`:
- Enable WS debug logging
- Check for `event:routed` logs
- Inspect Redux state via devtools
- Verify handler registration via imports

**Effort:** ~30-40 lines.

### E4: Add error handling patterns to `webchat-frontend-integration.md` (LOWER)

**Current state:** No discussion of what happens when the WebSocket drops mid-inference, how to handle 409 conflicts, or retry strategies.

**What is missing:**
- WebSocket reconnection behavior
- Hydration recovery after disconnect
- 409 conflict handling (inference already running)

**Effort:** ~40-60 lines.

---

## Content to port from go-go-mento

### P1: Port EngineBuilder reference (MEDIUM)

**Source:** `go-go-mento/docs/reference/webchat/engine-builder.md`

**What it covers:** EngineBuilder composition hub — responsibility overview, API methods (`Build`, `BuildConfig`, `BuildFromConfig`), override parsing details, sink wrapping pipeline, signature-based recomposition, error handling, testing strategies.

**Target:** New `webchat-engine-builder.md` in pinocchio docs, or a new section in `webchat-framework-guide.md`.

**Porting notes:**
- Code paths are in `pinocchio/pkg/webchat/` not `go/pkg/webchat/`
- Factory/profile semantics may differ slightly — validate against pinocchio code
- The sink wrapping pipeline (Watermill → extractors) is a key concept that explains how events reach the SEM translator

**Priority:** MEDIUM — important for backend developers extending engine composition.

**Effort:** ~80-120 lines (adapted from ~108-line go-go-mento doc).

### P2: Port InferenceState reference (MEDIUM)

**Source:** `go-go-mento/docs/reference/webchat/inference-state.md`

**What it covers:** Per-conversation inference lifecycle — `StartRun`, `FinishRun`, `IsRunning`, `CancelRun`, error types, Router integration, cancellation entry points.

**Target:** New section in `webchat-backend-reference.md` or standalone doc.

**Porting notes:**
- Pinocchio may use `session.Session` instead of `InferenceState` for some lifecycle management — validate
- The error types (`ErrInferenceRunning`, `ErrInferenceNotRunning`) are important for HTTP handler responses
- Cancellation flow is critical for understanding how inference cleanup works

**Priority:** MEDIUM — important for understanding conversation lifecycle.

**Effort:** ~60-80 lines (adapted from ~89-line go-go-mento doc).

### P3: Port SEM widget catalog and "Adding a New Widget" guide (HIGH — overlaps with N1)

**Source:** `go-go-mento/docs/reference/webchat/sem-and-widgets.md` (Part 2)

**What it covers:** Complete widget mapping table (entity kind → widget file → SEM event), entity type definitions, SEM → entity mapping table, hydration vs streaming merge rules, "Adding a New Widget" 5-step guide, debugging checklist.

**Target:** Enrich existing `webchat-sem-and-ui.md` with the detailed widget catalog and step-by-step guide.

**Porting notes:**
- Widget paths differ: pinocchio uses `pinocchio/cmd/web-chat/web/src/` while the platform layer in moments uses `moments/web/src/platform/timeline/widgets/`
- The go-go-mento doc has a more complete widget catalog (20+ widgets) than pinocchio's current doc
- The "Adding a New Widget" section should be expanded into the full end-to-end tutorial (N1) rather than just frontend-focused steps

**Priority:** HIGH — directly addresses the user's question about registering widgets to SEM events.

**Effort:** ~100-150 lines of adapted content.

---

## Summary table

| ID | Type | Title | Target Doc | Priority | Effort |
|----|------|-------|-----------|----------|--------|
| N1 | New doc | End-to-end "Adding a New Event Type" tutorial | `webchat-adding-event-types.md` | HIGH | ~350 lines |
| N2 | New doc/section | Timeline Projector reference | `webchat-backend-internals.md` or standalone | HIGH | ~250 lines |
| N3 | Addition | Cross-layer bridge in geppetto events doc | `geppetto/pkg/doc/topics/04-events.md` | MEDIUM | ~40 lines |
| E1 | Enhancement | "What Is Webchat?" elevator pitch | `webchat-overview.md` | MEDIUM | ~25 lines |
| E2 | Enhancement | SEM frame payload examples | `webchat-sem-and-ui.md` | MEDIUM | ~70 lines |
| E3 | Enhancement | Debugging section | `webchat-sem-and-ui.md` | LOWER | ~35 lines |
| E4 | Enhancement | Error handling patterns | `webchat-frontend-integration.md` | LOWER | ~50 lines |
| P1 | Port | EngineBuilder reference | New doc or framework guide | MEDIUM | ~100 lines |
| P2 | Port | InferenceState reference | Backend reference or standalone | MEDIUM | ~70 lines |
| P3 | Port | Widget catalog and "Adding a New Widget" | `webchat-sem-and-ui.md` | HIGH | ~125 lines |

**Recommended execution order:**

1. **N1** (end-to-end tutorial) — highest impact, enables developers to extend the system
2. **P3** (port widget catalog) — directly addresses widget-to-SEM registration question
3. **N2** (projector reference) — surfaces critical hidden behaviors
4. **N3** (cross-layer bridge) — small addition with high conceptual value
5. **E1** + **E2** (overview pitch + payload examples) — quick wins
6. **P1** + **P2** (port EngineBuilder + InferenceState) — important for backend work
7. **E3** + **E4** (debugging + error handling) — quality-of-life improvements

## Validation approach

For each improvement:
1. Verify code paths and APIs against current pinocchio source (not go-go-mento assumptions)
2. Ensure cross-references between pinocchio and geppetto docs are bidirectional
3. Run `docmgr doctor` after updates to catch stale references or vocabulary issues
4. For ported content, verify that pinocchio-specific paths, types, and patterns are used (not go-go-mento paths)
