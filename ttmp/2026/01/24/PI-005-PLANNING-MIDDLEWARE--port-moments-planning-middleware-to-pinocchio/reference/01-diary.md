---
Title: Diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-mento/go/pkg/agents/runner_event_bridge.go
      Note: Reference planning lifecycle emission from runner trace
    - Path: go-go-mento/go/pkg/persistence/timelinehydration/projector.go
      Note: Reference timelinehydration projection (messages/tools)
    - Path: go-go-mento/go/pkg/webchat/handlers/planning.go
      Note: Reference SEM forwarder mapping for planning/execution
    - Path: moments
      Note: Source project to analyze
    - Path: moments/docs/web/event-driven-widgets.md
      Note: Moments documentation of planning widget pipeline
    - Path: moments/web/src/platform/sem/handlers/planning.ts
      Note: Frontend planning aggregate handler (reference semantics)
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Profiles/middleware wiring for web-chat
    - Path: pinocchio/pkg/middlewares/planning/directive_middleware.go
      Note: Pinocchio directive injection into system prompt
    - Path: pinocchio/pkg/middlewares/planning/lifecycle_engine.go
      Note: Pinocchio planning lifecycle wrapper (commit d80ef03)
    - Path: pinocchio/pkg/webchat/engine.go
      Note: Engine composition special-cases planning (commit d80ef03)
    - Path: pinocchio/pkg/webchat/router.go
      Note: |-
        Where planning stubs were gated; will wire real middleware
        Publish execution.complete on toolloop error when planning enabled
    - Path: pinocchio/proto/sem/middleware/planning.proto
      Note: SEM planning/execution event schemas
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-24T21:05:52.656865135-05:00
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Keep a detailed, step-by-step diary for PI-005-PLANNING-MIDDLEWARE: analyze the planning middleware in `moments` and port the real behavior into Pinocchio so the webchat planning widget is driven by actual middleware/LLM output (not stubs).

## Context

Pinocchio currently supports rendering planning UI (`planning.*` + `execution.*` SEM events) and can hydrate planning snapshots via `sem.timeline.*`, but the backend previously emitted **stub planning events** that were identical every run. Those stubs are now gated behind `--emit-planning-stubs` and default off (PI-004).

This ticket ports the *real planning middleware* behavior from `moments` into Pinocchio so:
- planning is only present when enabled/configured
- emitted planning events carry meaningful, model-derived content
- the timeline projector can persist/hydrate planning snapshots consistently

## Step 1: Create PI-005 ticket scaffold

Created the new ticket, seeded an initial task breakdown, and set up a diary + a design doc to capture the analysis (what moments does) and the port plan (what we implement in pinocchio).

### Prompt Context

**User prompt (verbatim):** "analyze planning middleware in moments (i think?) and port into pinocchio. Add tasks, check them off, commit, etc..."

**Assistant interpretation:** Create tracked work, analyze `moments` planning middleware, then implement the equivalent middleware and wiring in the Pinocchio webchat backend, committing incrementally and checking tasks off.

**Inferred user intent:** Get “Moments-grade” planning UX in Pinocchio without hacks: real SEM planning streams, compatible with hydration and the React planning widget.

**Commit (docs):** pending

### What I did
- Created docmgr ticket `PI-005-PLANNING-MIDDLEWARE`.
- Added docs:
  - `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/design-doc/01-moments-planning-middleware-analysis-port-plan.md`
  - `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/reference/01-diary.md`
- Added initial task list in `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/tasks.md`.
- Related key files/dirs for review context (moments source, Pinocchio planning protos, web-chat wiring).

### Why
- This work spans multiple modules (moments + pinocchio) and needs durable traceability (tasks + diary + commits).

### What worked
- Ticket scaffold and tasks are in place; next step is code archaeology in `moments`.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Review ticket setup:
  - `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/index.md`
  - `geppetto/ttmp/2026/01/24/PI-005-PLANNING-MIDDLEWARE--port-moments-planning-middleware-to-pinocchio/tasks.md`

## Step 2: Confirm reference implementation (go-go-mento) + persistence story

This step reconciles an ambiguity: Moments contains the frontend widget + docs, but the backend that actually emits planning lifecycle events is (historically) implemented in go-go-mento / mento-playground. Once we treat go-go-mento as canonical, the port target for Pinocchio becomes clearer: emit the same `planning.*` + `execution.*` lifecycle events with a stable `run_id` correlation, and let the existing Pinocchio SEM translator + timeline projector do the rest.

I also verified that go-go-mento *does* have a durable hydration/persistence subsystem (“timeline hydration”), but it projects primarily message/tool entities. Pinocchio’s PI-004 hydration work already covers planning snapshots, so we should not port the go-go-mento persistence layer.

### Prompt Context

**User prompt (verbatim):** "maybe the middleware is in go-go-mento too? maybe even with hydration / persistence?"

**Assistant interpretation:** Treat go-go-mento as a likely source of the real planning lifecycle pipeline (beyond Moments frontend), and check whether it also has durability/hydration logic that influences how we should port to Pinocchio.

**Inferred user intent:** Use the most complete/production-proven implementation as the blueprint, and avoid rebuilding features that already exist (or that Pinocchio already implements differently).

**Commit (docs):** pending

### What I did
- Located the planning lifecycle forwarder in go-go-mento:
  - `go-go-mento/go/pkg/webchat/handlers/planning.go`
  - Confirms: typed events → protobuf payloads (`sem.middleware.planning`) → SEM frames (`planning.*`, `execution.*`), with stable `run_id` correlation.
- Located the planning event production path in go-go-mento:
  - `go-go-mento/go/pkg/agents/runner_event_bridge.go`
  - Confirms: planning lifecycle events are emitted from a runner trace (not a Geppetto middleware wrapper).
- Located the durability/hydration subsystem in go-go-mento:
  - `go-go-mento/go/pkg/persistence/timelinehydration/*`
  - Confirms: Postgres-backed projection store with versioning and a projector that produces `sem.timeline.*` snapshots (primarily messages/tools).
- Cross-checked the Moments frontend handler that consumes the same SEM contract:
  - `moments/web/src/platform/sem/handlers/planning.ts`
  - Confirms: a single aggregate per `runId` that drives the planning widget.

### Why
- The port should preserve the event contract and correlation semantics that Moments/go-go-mento already use, rather than inventing a new shape.
- We only want to port what Pinocchio lacks (real planning event emission), not duplicate persistence/hydration systems Pinocchio already has (PI-004 SQLite timeline).

### What worked
- The repo evidence is consistent:
  - Moments: widget + handler (frontend).
  - go-go-mento: lifecycle emission + forwarder mapping (backend).
  - Pinocchio: schema + translator + widget + timeline persistence; missing only the producer.

### What didn't work
- Searching Moments backend for a “planning middleware” implementation didn’t turn up a direct equivalent; planning appears to be documented/consumed there, not produced.

### What I learned
- go-go-mento’s persistence/hydration projector does not cover planning snapshots, so Pinocchio’s PI-004 work is already “more complete” for planning persistence.

### What was tricky to build
- N/A (analysis step)

### What warrants a second pair of eyes
- Whether we want to mirror go-go-mento’s “analysis_id” legacy alias fields in Pinocchio’s SEM frames. (Today Pinocchio uses protobuf-first payloads and the planning widget consumes those directly.)

### What should be done in the future
- N/A

### Code review instructions
- Read in order:
  - `moments/docs/web/event-driven-widgets.md`
  - `go-go-mento/go/pkg/agents/runner_event_bridge.go`
  - `go-go-mento/go/pkg/webchat/handlers/planning.go`
  - `go-go-mento/go/pkg/persistence/timelinehydration/projector.go`

## Step 3: Implement real planning lifecycle in Pinocchio (no stubs)

This step implements the missing backend piece in Pinocchio: a real producer of `planning.*` and `execution.*` events. The implementation is a lightweight planning lifecycle wrapper around the composed chat engine. It performs a planner call once per inference, parses a strict JSON envelope, emits planning iterations, stores the final directive in `Turn.Data`, and injects that directive into the system prompt right before execution.

It also ensures `execution.complete` is emitted when the tool loop exits with an error, so the UI doesn’t get stuck in an “execution running” state.

### Prompt Context

**User prompt (verbatim):** "analyze planning middleware in moments (i think?) and port into pinocchio. Add tasks, check them off, commit, etc..."

**Assistant interpretation:** Implement the backend planning lifecycle so Pinocchio’s existing planning widget is powered by real events, and make it configurable (enabled by profile / middleware config), then commit and track progress in the ticket.

**Inferred user intent:** Achieve “Moments-grade” planning UX in Pinocchio: real planning trace, stable correlation IDs, and consistent widget updates and hydration.

**Commit (code):** `d80ef03` — "web-chat: add real planning lifecycle"

### What I did
- Added a Pinocchio planning middleware package:
  - `pinocchio/pkg/middlewares/planning/lifecycle_engine.go` — wrapper that emits `planning.*` and `execution.*`
  - `pinocchio/pkg/middlewares/planning/directive_middleware.go` — injects a final directive into the system prompt (idempotent, marker-based)
  - `pinocchio/pkg/middlewares/planning/config.go` — config + default planner prompt (strict JSON envelope)
  - `pinocchio/pkg/middlewares/planning/middleware_keys.go` — typed Turn.Data key for the directive
  - `pinocchio/pkg/middlewares/planning/lifecycle_engine_test.go` — event emission test
- Wired planning into webchat engine composition:
  - `pinocchio/pkg/webchat/engine.go` treats a `middlewares` entry named `"planning"` as a special lifecycle wrapper (not a standard middleware factory), and adds the directive injector at the innermost position so directives land after the base system prompt.
- Wired a `planning` profile into the `web-chat` command for easy manual testing:
  - `pinocchio/cmd/web-chat/main.go` adds profile `planning` and `GET /planning` to set `chat_profile=planning`.
- Ensured toolloop error paths still produce `execution.complete`:
  - `pinocchio/pkg/webchat/router.go` publishes `execution.complete` on `handle.Wait()` errors when planning is enabled.
- Verified correctness:
  - `cd pinocchio && go test ./... -count=1`
  - `cd pinocchio/cmd/web-chat/web && npm run typecheck`

### Why
- Pinocchio needs a real producer for planning events; the UI side already exists.
- The planner call must not leak as an assistant message, so it runs without the webchat sinks while the lifecycle events still publish normally.

### What worked
- The new planning profile can be activated via `GET /planning`, and the UI receives real `planning.*` events.
- Lint rules for typed keys are satisfied by moving the `turns.DataK` definition into a `*_keys.go` file.

### What didn't work
- Initial implementation triggered a custom linter error:
  - `do not call turns.DataK outside key-definition files`
  - Fix: renamed key file to `middleware_keys.go` to match the project’s key-definition convention.

### What I learned
- In this codebase, “middleware” is often overloaded: some behaviors that conceptually act like middleware are better expressed as engine wrappers because they need to perform additional inference calls.

### What was tricky to build
- Avoiding planner-call contamination:
  - Because context-attached sinks are append-only, the planner call must run in a fresh context without sinks; otherwise its `llm.*` events would render as normal chat messages.

### What warrants a second pair of eyes
- Whether the planner prompt should be made configurable via Glazed parameters (instead of only via middleware config objects).

### What should be done in the future
- Add a small UI affordance (“Profiles” dropdown) to switch between `default`/`agent`/`planning` without manual fetches.

### Code review instructions
- Start with the new package:
  - `pinocchio/pkg/middlewares/planning/lifecycle_engine.go`
  - `pinocchio/pkg/middlewares/planning/directive_middleware.go`
- Then review webchat integration:
  - `pinocchio/pkg/webchat/engine.go`
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/cmd/web-chat/main.go`
- Validate:
  - `cd pinocchio && go test ./... -count=1`
