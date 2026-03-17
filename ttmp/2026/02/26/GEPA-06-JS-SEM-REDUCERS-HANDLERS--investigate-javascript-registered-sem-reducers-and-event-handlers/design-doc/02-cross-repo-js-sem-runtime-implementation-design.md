---
Title: Cross-Repo JavaScript SEM Runtime Implementation Design (Pinocchio and go-go-gepa)
Ticket: GEPA-06-JS-SEM-REDUCERS-HANDLERS
Status: active
Topics:
    - gepa
    - pinocchio
    - geppetto
    - sem
    - event-streaming
    - js-vm
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/timeline_js_runtime.go
      Note: Current Goja runtime implementation for JS reducers/handlers
    - Path: pinocchio/pkg/webchat/timeline_registry.go
      Note: Runtime bridge into projection pipeline
    - Path: pinocchio/cmd/web-chat/timeline_js_runtime_loader.go
      Note: Startup loader for --timeline-js-script
    - Path: pinocchio/cmd/web-chat/llm_delta_projection_harness_test.go
      Note: End-to-end harness for llm.delta projection behavior
    - Path: go-go-gepa/cmd/gepa-runner/stream_cli_integration_test.go
      Note: Existing streaming event validation path in go-go-gepa
ExternalSources: []
Summary: Implementation design for hardening and reusing JavaScript SEM reducer/handler runtime capabilities across pinocchio web-chat and go-go-gepa without forcing direct package coupling.
LastUpdated: 2026-02-26T20:05:00-05:00
WhatFor: Provide a buildable blueprint for productionizing JS SEM projection and enabling cross-repo reuse
WhenToUse: Use when implementing or reviewing JS reducer/runtime portability work
---

# Cross-Repo JavaScript SEM Runtime Implementation Design

## Executive Summary

This document describes a full implementation design for JavaScript-based SEM reducers and event handlers, starting from the current `pinocchio` implementation and extending toward reuse in `go-go-gepa`, which currently does not depend on `pinocchio` packages.

The short version is:

1. The core capability now exists and works in `pinocchio web-chat`.
2. The current implementation is app-adjacent and tied to `timeline` projection types.
3. Reuse in `go-go-gepa` is feasible, but should be done by extracting a small runtime core and adding adapter layers, not by importing `pkg/webchat` directly.
4. The recommended migration path is incremental: harden, extract, adapt, then roll out.

This design intentionally emphasizes operational reliability, compatibility guarantees, and clear API contracts. The goal is not just “make it run,” but “make it safe to evolve and portable across repositories with different projection models.”

---

## Problem Statement

We need JavaScript code to participate in event-driven projection logic and reactive hooks, with two concrete requirements:

1. Register JavaScript SEM reducers/projections.
2. Register JavaScript event handlers for events like `llm.delta` or other SEM event types.

There is now a working runtime inside `pinocchio web-chat`, including:

- `registerSemReducer(eventType, fn)`
- `onSem(eventType, fn)`
- wildcard handler support (`*`)
- reducer consume semantics (`consume: true|false`)
- startup loading via `--timeline-js-script`

The next challenge is architectural reuse. `go-go-gepa` has streaming/event capabilities, but does not currently use `pinocchio` webchat internals. We need a design that avoids accidental hard coupling while preserving behavior parity.

---

## Scope

### In Scope

- JavaScript reducer/handler runtime architecture.
- API contract and behavior semantics.
- Extractable abstraction boundaries for cross-repo reuse.
- Adapters for `pinocchio` timeline projection and `go-go-gepa` stream events.
- Test strategy, operational controls, and rollout plan.

### Out of Scope

- Frontend widget rendering specifics.
- Provider-specific LLM transport implementation details.
- Replacing existing geppetto event models (`start/partial/final`) at this stage.

---

## Fundamentals and Shared Vocabulary

To avoid confusion across repos, we define the core layers precisely.

### Geppetto Event Layer

Geppetto engines publish runtime events such as:

- `start`
- `partial`
- `final`
- other tool and info event classes

These are engine/runtime events, not SEM domain names.

### SEM Translation Layer

In `pinocchio`, translator logic maps geppetto events into SEM envelope events such as:

- `llm.start`
- `llm.delta`
- `llm.final`

This mapping is where the semantic contract for downstream projection begins.

### Projection Layer

Projection consumes SEM events and materializes state snapshots (for example timeline entities).

Projection can be:

- builtin (Go handlers)
- dynamic (JS reducers)
- combined (dynamic add-on with builtin preserved)

### Reducer vs Handler

Reducer:

- returns projection changes (`upserts`)
- may set `consume=true` to suppress builtin projection for that event

Handler:

- side-effect hook with no required output
- useful for metrics, counters, alerts, custom triggers

This split is crucial for deterministic semantics and debugging.

---

## Current State Architecture

### Pinocchio (Current Working Baseline)

Key files:

- `pkg/webchat/timeline_js_runtime.go`
- `pkg/webchat/timeline_registry.go`
- `cmd/web-chat/timeline_js_runtime_loader.go`
- `cmd/web-chat/llm_delta_projection_harness_test.go`

Current runtime behavior:

1. Startup loads JS files via `--timeline-js-script`.
2. JS registers reducers/handlers against event types.
3. For each SEM event in projection pipeline:
   - JS handlers run first.
   - JS reducers run next.
   - return payload decoded to timeline upserts.
   - `consume=true` suppresses builtin branch.
4. Callback errors are logged and contained.

### go-go-gepa (Current Baseline)

`go-go-gepa` already has streaming event infrastructure (GEPA-04), but this is not equivalent to pinocchio timeline projection runtime.

Current condition:

- Event streaming exists.
- JS plugin API has event emission support.
- No direct use of `pinocchio/pkg/webchat` timeline runtime.

Therefore, direct reuse requires explicit extraction or adapter layer introduction.

---

## Design Goals

### Primary Goals

1. Preserve current pinocchio behavior with no regression.
2. Define portable runtime core independent of timeline-specific types.
3. Keep the JS API stable across repositories.
4. Enable deterministic tests for consume vs non-consume semantics.
5. Support safe operational controls (fail-fast load, contain callback faults).

### Secondary Goals

1. Keep performance overhead bounded under stream load.
2. Enable future optional stateful reducer context.
3. Support long-term versioned contract evolution.

### Non-Goals

1. Building a distributed sandbox runtime in v1.
2. Introducing hot-reload by default.
3. Unifying all event models into one abstraction immediately.

---

## Functional Requirements

1. JS can register reducer functions keyed by event type.
2. JS can register side-effect handlers keyed by event type.
3. Wildcard subscription must be supported.
4. Reducer return payload must support:
   - boolean consume
   - entity object
   - entity list
   - object wrapper `{consume, upserts}`
5. Runtime callback failures must not crash process.
6. Startup script parse/load failures must fail fast.
7. Adapter layer must map generic reducer outputs to host projection sink.
8. Host should support deterministic ordering guarantees.

---

## Non-Functional Requirements

1. Compatibility: keep existing `registerSemReducer` and `onSem` signatures.
2. Observability: structured logs for script load, callback errors, consume decisions.
3. Performance: avoid unbounded allocations in per-event hot path.
4. Safety: bounded runtime APIs, no implicit filesystem/network grants through host unless explicitly exposed.
5. Testability: unit + integration harness for each host adapter.

---

## Proposed Target Architecture

### Overview

Introduce a portable runtime core plus host adapters:

```text
                +-------------------------------+
                |   JS Runtime Core (portable)  |
                |  - registry (reducers/handlers)|
SEM Event ----> |  - dispatch + decode contract | ----> Host Projection Sink
                |  - consume decision           |
                +-------------------------------+
                           ^
                           |
                  Host API Bridge

Pinocchio Adapter: SEM -> timeline entities (TimelineEntityV2)
go-go-gepa Adapter: stream-event -> gepa projection model / persistence sink
```

### Why This Shape

If we reuse `pkg/webchat` directly in `go-go-gepa`, we import timeline-specific assumptions and webchat coupling. That raises maintainability risk.

A portable core lets both repos share behavior while adapting output types locally.

---

## API Design Sketches

### Go API: Portable Core

```go
// semruntime core
package semruntime

type Event struct {
    Type     string
    ID       string
    Seq      uint64
    StreamID string
    Data     map[string]any
    NowMS    int64
}

type ReducerOutput struct {
    Consume bool
    Upserts []map[string]any // host-agnostic entities
}

type HostSink interface {
    Upsert(ctx context.Context, ev Event, entity map[string]any) error
}

type Runtime interface {
    LoadScriptFile(path string) error
    Handle(ctx context.Context, ev Event, sink HostSink) (handled bool, err error)
}
```

### Go API: Pinocchio Adapter

```go
package webchatadapter

type TimelineSink struct {
    Projector *webchat.TimelineProjector
}

func (s *TimelineSink) Upsert(ctx context.Context, ev semruntime.Event, entity map[string]any) error {
    te := decodeToTimelineEntityV2(entity, ev)
    return s.Projector.Upsert(ctx, ev.Seq, te)
}
```

### Go API: go-go-gepa Adapter

```go
package geparuntime

type DatasetProjectionSink struct {
    Store ProjectionStore
}

func (s *DatasetProjectionSink) Upsert(ctx context.Context, ev semruntime.Event, entity map[string]any) error {
    // map generic entity into gepa projection row/document model
    return s.Store.UpsertProjection(ctx, ev.StreamID, ev.Seq, entity)
}
```

### JavaScript Host Contract (Stable)

```javascript
registerSemReducer("llm.delta", function(event, ctx) {
  return {
    consume: false,
    upserts: [{
      id: event.id + "-delta",
      kind: "llm.delta.projection",
      props: {
        cumulative: event.data && event.data.cumulative
      }
    }]
  };
});

onSem("*", function(event, ctx) {
  // side effects only
});
```

---

## Reducer Return Semantics

Support matrix:

1. `true`
2. `false`
3. `{consume: true}`
4. `{id, kind, props, ...}`
5. `[{...}, {...}]`
6. `{consume, upserts}`

Normalization algorithm:

```pseudocode
function normalizeReturn(raw):
  if raw is null:
    return {consume:false, upserts:[]}

  if raw is boolean:
    return {consume:raw, upserts:[]}

  if raw is array:
    return {consume:false, upserts:decodeEntities(raw)}

  if raw is object:
    consume = toBool(raw.consume)
    if "upserts" in raw:
      if upserts is array: return {consume, upserts:decodeEntities(upserts)}
      if upserts is object: return {consume, upserts:[decodeEntity(upserts)]}
      return {consume, upserts:[]}

    if looksLikeEntity(raw):
      return {consume, upserts:[decodeEntity(raw)]}

  return {consume:false, upserts:[]}
```

This is already close to pinocchio behavior and should be preserved for compatibility.

---

## Event Dispatch Semantics

Order is intentional:

1. handlers
2. reducers
3. host sink upserts
4. consume decision returned to pipeline

Reasoning:

- handlers can update runtime-local counters/state read by reducers.
- this was validated by harness tests.

Dispatch pseudocode:

```pseudocode
function handleEvent(ev):
  consume = false

  for handler in handlers[ev.type] + handlers["*"]:
    try handler(ev, ctx)
    catch err -> log warn continue

  for reducer in reducers[ev.type] + reducers["*"]:
    try raw = reducer(ev, ctx)
    catch err -> log warn continue

    out = normalizeReturn(raw)
    if out.consume:
      consume = true

    for entity in out.upserts:
      try sink.upsert(ev, entity)
      catch err -> log warn continue

  return consume
```

---

## Adapter Strategy for go-go-gepa

### Current Constraint

`go-go-gepa` does not currently import `pinocchio` webchat internals.

### Recommended Path

1. Extract core runtime package to shared location (or small standalone module).
2. Keep pinocchio adapter local to pinocchio.
3. Build go-go-gepa adapter local to go-go-gepa.
4. Reuse JS contract and regression fixtures across both hosts.

### Adapter Mapping Considerations

`go-go-gepa` may not use `TimelineEntityV2`. It can still consume generic entity maps and project into:

- sqlite projection table
- event-augmented dataset logs
- output documents

Key is that runtime upserts remain host-neutral until adapter decode.

---

## Compatibility Model

### Behavioral Compatibility

Guarantee the following across extraction:

1. same reducer/handler registration API
2. same consume semantics
3. same wildcard matching semantics
4. same callback error containment
5. same load failure behavior

### Contract Versioning

Introduce optional runtime contract metadata:

```go
const ContractVersion = "semruntime.v1"
```

Potential future additions:

- `getState/setState`
- richer context fields
- transaction/batch upserts

Versioning prevents silent drift.

---

## Testing Strategy

### Unit Tests (Core)

- registration validation
- wildcard dispatch
- normalization matrix for reducer returns
- consume aggregation behavior
- error containment behavior

### Adapter Tests (Pinocchio)

- map decode to `TimelineEntityV2`
- consume true suppresses builtin projection branch
- consume false preserves builtin + side projections
- handler-before-reducer ordering

### Adapter Tests (go-go-gepa)

- stream event to semruntime event mapping
- projection sink persistence semantics
- compatibility fixture parity with pinocchio scripts

### End-to-End Harness

Current pinocchio harness already validates real route flow:

- `POST /chat/default`
- translated SEM frames
- JS runtime behavior
- `GET /api/timeline` assertions

Equivalent harness should be added in go-go-gepa once adapter is wired.

---

## Operational and Security Design

### Runtime Safety Defaults

1. Fail fast on script load syntax/runtime compile errors.
2. Contain callback runtime exceptions.
3. Log callback errors with event type and script source context when available.
4. Do not auto-expose filesystem/network globals to JS unless intentionally registered.

### Operational Guardrails

- startup log includes loaded script paths
- metrics counters (recommended):
  - events processed
  - reducer errors
  - handler errors
  - upsert errors
  - consume decisions per event type

### Incident Modes

- Broken script at startup: server/process refuses to continue (expected fail-fast).
- Runtime errors under traffic: continue stream, degraded custom projection only.

---

## Performance Design

### Hot Path Concerns

- frequent event payload map conversions
- repeated map-to-entity decoding
- lock contention in shared VM usage

### Mitigations

1. Keep runtime lock scope minimal.
2. Avoid unnecessary JSON round-trips after decode.
3. Consider optional per-script runtime sharding if lock contention appears.
4. Batch sink upserts where host supports it.

### Performance Targets

Initial pragmatic target:

- no significant regression in steady-state event throughput for baseline workloads.

Formal targets can be set after first profiling pass.

---

## Migration Plan

### Phase 0: Baseline Freeze

- keep current pinocchio runtime behavior as reference baseline.
- lock behavioral tests as contract tests.

### Phase 1: Core Extraction

- create `semruntime` core package.
- move registration/dispatch/normalization logic there.
- retain pinocchio adapter wrappers.

### Phase 2: Pinocchio Stabilization

- switch pinocchio to extracted core.
- run existing and harness tests.
- validate no behavior drift.

### Phase 3: go-go-gepa Adapter

- define host projection sink model in go-go-gepa.
- map stream events to semruntime envelope.
- wire startup script loader for go-go-gepa command surface.

### Phase 4: Cross-Repo Contract Suite

- shared JS fixture set for parity tests.
- CI gating for both repos.

---

## Implementation Task Breakdown

### Core Extraction Tasks

1. Introduce `semruntime` package with runtime interfaces.
2. Port reducer/handler dispatch and return normalization.
3. Add core test table for reducer return forms.

### Pinocchio Adapter Tasks

1. Implement `TimelineSink` adapter.
2. Keep `--timeline-js-script` startup UX unchanged.
3. Re-run harness and existing tests.

### go-go-gepa Adapter Tasks

1. Define projection sink interface for gepa domain.
2. Wire command flags for JS script loading.
3. Add integration test with `gpt-5-nano` profile registry and stream flow.

### Documentation Tasks

1. Contract doc for JS runtime APIs.
2. Troubleshooting doc for consume/fallback behavior.
3. Migration notes for host adapter authors.

---

## Open Questions

1. Should runtime support stateful reducers in v1 (`getState/setState`) or remain stateless?
2. Should multi-script load order be explicit and deterministic by flag order only?
3. Should go-go-gepa projection sink persist generic entity JSON or normalized schema per kind?
4. Is hot-reload a requirement, or startup-only loading sufficient?

---

## Risks and Mitigations

### Risk: Hidden Contract Drift

Mitigation:

- shared fixture tests and strict compatibility assertions.

### Risk: Over-coupling to Pinocchio Types

Mitigation:

- keep core package host-neutral and adapter-driven.

### Risk: Runtime Script Quality in Production

Mitigation:

- preflight compile checks in CI.
- staged rollouts with canary script sets.

### Risk: Performance Regression on High Event Rates

Mitigation:

- profile with representative event streams.
- optimize decode and lock strategy before broad rollout.

---

## Example End-State Developer Experience

### Pinocchio

```bash
go run ./cmd/web-chat web-chat \
  --profile-registries ./profiles.yaml \
  --timeline-js-script ./scripts/timeline-delta.js
```

### go-go-gepa (target state)

```bash
go run ./cmd/gepa-runner stream run \
  --profile gpt-5-nano \
  --profile-registries ./profiles.yaml \
  --sem-js-script ./scripts/gepa-projection.js
```

JS script reused with minimal or zero changes when both hosts honor the same contract.

---

## Acceptance Criteria

1. Pinocchio behavior unchanged after core extraction (all existing tests pass).
2. Pinocchio harness tests continue to pass for:
   - non-consume add-on projection
   - consume suppression
   - handler-before-reducer order
3. go-go-gepa can load JS reducers/handlers through new adapter path.
4. Shared contract fixture suite passes in both repos.
5. Operational docs cover startup failure and runtime fault modes.

---

## Appendix A: Minimal Cross-Host JS Fixture

```javascript
onSem("llm.delta", function(ev) {
  // side-effect marker
});

registerSemReducer("llm.delta", function(ev) {
  return {
    consume: false,
    upserts: [{
      id: ev.id + "-projection",
      kind: "llm.delta.projection",
      props: {
        delta: ev.data && ev.data.delta,
        cumulative: ev.data && ev.data.cumulative
      }
    }]
  };
});
```

---

## Appendix B: Extraction Checklist

1. Copy logic, do not change behavior first.
2. Move tests with logic to prevent accidental semantics drift.
3. Keep adapter conversion thin and explicit.
4. Preserve error strings where operational tooling depends on them.
5. Run pinocchio + go-go-gepa integration tests before merge.

---

## Final Recommendation

Proceed with extraction-backed reuse (portable core + host adapters). Do not couple go-go-gepa directly to `pinocchio/pkg/webchat` internals. The current pinocchio implementation is a good production baseline and already validated by harness tests; treat it as the semantic reference, then generalize carefully.

