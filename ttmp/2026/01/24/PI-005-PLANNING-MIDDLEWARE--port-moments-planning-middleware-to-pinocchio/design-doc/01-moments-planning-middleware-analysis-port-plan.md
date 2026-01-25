---
Title: 'Moments Planning Middleware: Analysis + Port Plan'
Ticket: PI-005-PLANNING-MIDDLEWARE
Status: active
Topics:
    - backend
    - middleware
    - moments
    - pinocchio
    - planning
    - protobuf
    - webchat
    - websocket
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-24T21:05:52.363391064-05:00
WhatFor: ""
WhenToUse: ""
---

# Moments Planning Middleware: Analysis + Port Plan

## Executive Summary

The “planning widget” that you see in the Moments UI is driven by a typed SEM event contract (`sem.middleware.planning.*`) and a frontend aggregate handler (`planning.start|iteration|reflection|complete` + `execution.start|complete` → a single evolving `planning` timeline entity). In the Moments repo, this is primarily a **frontend** feature plus documentation; the **canonical backend implementation** of emitting planning events lives in the older `go-go-mento` / “mento-playground” codebase, where planning events are produced from an agent runner trace and forwarded over the webchat pipeline.

Pinocchio already has:
- the protobuf schemas for planning/execution SEM payloads,
- a React planning widget that renders those events,
- and durable hydration via `sem.timeline.*` snapshots (PI-004), including planning snapshots.

What Pinocchio lacked was a real backend producer of planning events (as opposed to earlier stubs). This design ports the planning lifecycle concept into Pinocchio by adding a **planning lifecycle engine wrapper** that:
- runs a dedicated “planner call” once per inference/run,
- emits typed Geppetto events (`planning.*`, `execution.*`) correlated by `run_id`,
- injects the final directive into the execution turn as a system block section,
- and relies on existing Pinocchio SEM translation + timeline projection to stream and persist the widget state.

## Problem Statement

Pinocchio webchat can render planning widgets, but there is no real “planning middleware” in the backend that produces meaningful content. Previously, Pinocchio emitted fixed, stub planning events on every run, which was misleading and made planning appear always active even when no planning logic existed.

We want to port the *real* Moments/go-go-mento planning pipeline semantics into Pinocchio so that:
- planning events reflect actual model-driven planning output,
- planning only appears when configured/enabled,
- planning events are correlated and progressive (start → iteration(s) → complete; execution start/complete),
- and the resulting planning widget state is compatible with Pinocchio’s durable hydration (`GET /timeline`).

## Proposed Solution

### A. Treat go-go-mento as the reference backend implementation

In go-go-mento, the planning widget is powered by:
- **Runner bridge** that emits planning/execution lifecycle events from the runner trace:
  - `go-go-mento/go/pkg/agents/runner_event_bridge.go`
- **Forwarder handlers** that map typed events to protobuf payloads and SEM frames:
  - `go-go-mento/go/pkg/webchat/handlers/planning.go`

Key semantics to preserve:
- A stable `run_id` correlation ID is the entity ID on the frontend.
- Each planning iteration updates the same planning entity (aggregate keyed by run_id).
- `execution.*` is nested under the same “planning run” concept.

### B. Treat Moments as the reference frontend implementation

In Moments, the planning widget semantics are:
- `planning.*` / `execution.*` SEM frames are decoded via protobuf JSON (`fromJson(…Schema, ev.data)`).
- An in-memory aggregate keyed by `runId` materializes a single `PlanningEntity` timeline entity:
  - `moments/web/src/platform/sem/handlers/planning.ts`

Pinocchio’s React widget registry already follows the same “aggregate into one widget entity” pattern.

### C. Implement a Pinocchio planning lifecycle producer

Pinocchio does not have an agent runner trace like go-go-mento; it is a “chat + toolloop” engine. So, the port needs a Pinocchio-native implementation that still emits the same planning/execution lifecycle events. The pragmatic approach is:

1) **Planner call (once per inference/run)**:
   - Clone the current turn and run a dedicated “planner prompt” against the same provider engine.
   - Parse a JSON envelope describing iterations + final directive.
   - Emit typed events:
     - `planning.start`
     - `planning.iteration` (N times)
     - `planning.complete`
   - Store the `final_directive` in `Turn.Data` (typed key).

2) **Execution phase**:
   - Emit `execution.start` once the directive exists.
   - Inject the directive into the system prompt as a marked section (idempotent rewrite).
   - Ensure `execution.complete` exists even if the toolloop errors (e.g. max-iterations).

This keeps the contract stable for the frontend and for the `sem.timeline` projector, while staying within Pinocchio’s architecture.

### D. Hydration / persistence

go-go-mento includes a separate “timeline hydration” persistence pipeline backed by Postgres:
- `go-go-mento/go/pkg/persistence/timelinehydration/*`
- It currently projects **message/tool** events to `sem.timeline.*` snapshots.

Pinocchio already implements durable hydration with SQLite and projects **messages, tools, thinking_mode, and planning/execution** into `sem.timeline.*` snapshots (PI-004). Therefore, we do not port go-go-mento’s persistence layer; we only port the planning event emission semantics.

## Design Decisions

### Use the existing SEM protobuf contract (no new schema)

Pinocchio already uses `pinocchio/proto/sem/middleware/planning.proto` and has frontend handlers for these events. We keep the same contract so no migration is required.

### Use `inference_id` as the `run_id`

Pinocchio sessions already assign a stable `inference_id` per run. Reusing it as `run_id` keeps correlation straightforward and aligns with the semantics expected by the UI and the timeline projector.

### Planner call must not leak as an assistant “message”

If the planner call streamed `llm.*` events into the normal sink, the user would see “planner output” as a chat message. The planner call must run without the webchat event sinks while *still* emitting the planning lifecycle events via the normal sinks.

## Alternatives Considered

### 1) True “middleware that calls the engine”

Geppetto middleware wraps a handler but does not provide access to the underlying engine, so it cannot easily perform an additional inference call without recursion hazards. This pushes us toward an engine wrapper/lifecycle layer rather than a pure middleware.

### 2) Port go-go-mento’s full agent runner / planner trace

This would be the “most faithful” port, but it is significantly more invasive: Pinocchio would need an agent orchestration subsystem and trace format. The engine-wrapper approach gives us useful planning UX now with minimal disruption.

## Implementation Plan

1) Implement the planning lifecycle engine wrapper + directive injector (Pinocchio backend).
2) Wire the planning lifecycle into Pinocchio webchat engine composition when enabled by profile/middleware config.
3) Ensure the router emits `execution.complete` on toolloop errors (so the widget doesn’t get stuck “running”).
4) Add tests for lifecycle event emission.
5) Validate end-to-end via:
   - `go test ./...`
   - web typecheck/build
   - Storybook planning story / live server with the planning profile.

## Open Questions

1) Do we want a dedicated planner model distinct from the executor model (go-go-mento supports that via workflow configuration)? Pinocchio currently uses the same provider/model.
2) Should we emit `planning.reflection` events (separate from `planning.iteration.reflection_text`)?
3) Should we store `tokens_used` / other telemetry for `execution.complete`? Today this depends on provider support in the engine event stream.

## References

- Moments docs: `moments/docs/web/event-driven-widgets.md`
- Moments frontend SEM handler: `moments/web/src/platform/sem/handlers/planning.ts`
- go-go-mento runner bridge: `go-go-mento/go/pkg/agents/runner_event_bridge.go`
- go-go-mento forwarder handlers: `go-go-mento/go/pkg/webchat/handlers/planning.go`
- go-go-mento hydration/persistence: `go-go-mento/go/pkg/persistence/timelinehydration/*`
- Pinocchio SEM planning schema: `pinocchio/proto/sem/middleware/planning.proto`
- Pinocchio hydration baseline (PI-004): `geppetto/ttmp/2026/01/24/PI-004-ACTUAL-HYDRATION--pinocchio-webchat-durable-timeline-hydration-sem-timeline-snapshots/`
