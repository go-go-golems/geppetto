---
Title: SEM and Event Pipeline Performance Deep Dive and Plan
Ticket: PI-016-SEM-EVENT-PERF
Status: active
Topics:
    - backend
    - events
    - architecture
    - websocket
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Event consume loop, seq derivation, and frame dispatch path
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Event-to-SEM conversion and envelope generation
    - Path: pinocchio/pkg/webchat/sem_buffer.go
      Note: In-memory frame storage and overflow copy behavior
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Synchronous projection and throttle logic
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: SQLite upsert/snapshot IO characteristics
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Stream callback path to pool, sem buffer, and projector
    - Path: pinocchio/pkg/webchat/timeline_upsert.go
      Note: timeline.upsert broadcast path
ExternalSources: []
Summary: Deep dive into SEM/event pipeline performance bottlenecks and a staged optimization plan across buffering, translation, projection, and persistence.
LastUpdated: 2026-02-07T11:05:00-05:00
WhatFor: Establish a clear performance model and implementation roadmap for reducing latency, allocation churn, and throughput bottlenecks in debug/event pipelines.
WhenToUse: Use before and during SEM/event performance optimization work.
---

# SEM and Event Pipeline Performance Deep Dive and Plan

## Executive summary

The SEM/event path is correct functionally, but there are avoidable costs in hot paths:

1. envelope parse/rebuild cost on every frame enrichment,
2. synchronous projector writes in consume loop,
3. sem buffer overflow copying,
4. repeated JSON decode/encode in downstream consumers.

These are manageable for low traffic but become visible under high token-rate streams and multi-conversation load.

## Pipeline map

Hot path per event:

1. `StreamCoordinator.consume` decodes event from watermill payload.
2. translator generates one or more SEM frames.
3. `SemanticEventsFromEventWithCursor` unmarshals and re-marshals each frame to inject `seq`/`stream_id`.
4. callback path in conversation broadcasts to WS, appends to `semFrameBuffer`, and applies timeline projection.
5. timeline projector may write to SQLite synchronously.

## Bottleneck analysis

### 1) Envelope rewrite overhead

Location: cursor enrichment helper in stream coordinator.

Issue:

- every SEM frame incurs JSON unmarshal + marshal to inject cursor fields.
- for high-frequency `llm.delta`, this adds CPU and allocation overhead.

Potential improvements:

- generate cursor fields directly at translation time where possible,
- use lightweight envelope struct and avoid map round-trips,
- consider pooled buffers in hot path.

### 2) Synchronous projector writes in consume loop

Location: conversation stream callback -> `timelineProj.ApplySemFrame`.

Issue:

- projection and persistence run on the same flow that fans out events.
- if SQLite stalls, websocket stream and sem buffering are delayed.

Potential improvements:

- decouple projector write path with internal queue and bounded worker,
- preserve ordering using per-conversation worker serialization,
- keep direct mode available for strict debug correctness scenarios.

### 3) semFrameBuffer overflow copying

Location: `sem_buffer.go`.

Issue:

- overflow path creates a new slice and copies on trim.
- sustained high traffic causes allocation churn.

Potential improvements:

- implement true ring buffer with head/tail indices,
- support snapshot extraction without full clone where feasible.

### 4) Repeated payload parse in debug/event endpoints

Issue:

- debug event views and projector both parse JSON payloads repeatedly.
- no cached parsed envelope representation exists.

Potential improvements:

- store parsed envelope alongside raw bytes in buffer,
- optional parsed cache keyed by seq for debug mode.

### 5) Timeline upsert fan-out overhead

Location: `timeline_upsert.go`.

Issue:

- each upsert emits another SEM frame and broadcast.
- increases event volume and client-side processing.

Potential improvements:

- add configurable sampling/coalescing for frequent update kinds,
- preserve final-state guarantees.

## Measurement plan

### Metrics to add

Per conversation:

- events ingested/sec
- SEM frames emitted/sec
- consume-loop latency p50/p95
- projector apply latency p50/p95
- sqlite upsert latency p50/p95
- sem buffer drop/trim count
- websocket send queue depth and drop count

Process-wide:

- allocations/sec by package
- GC pause time under synthetic token stream

### Benchmarks/tests

1. microbench: envelope enrich function (`-benchmem`)
2. microbench: sem buffer add/snapshot under overflow
3. integration load test: 1/5/20 concurrent conversations streaming deltas
4. regression test: ordering and seq monotonicity under optimized path

## Optimization roadmap

### Phase 1: observability first

- add metrics and tracing spans around consume path and projector path,
- add benchmark harness.

### Phase 2: low-risk hot path wins

- replace sem buffer copy-trim with ring buffer,
- reduce JSON map conversions in cursor enrichment.

### Phase 3: projector decoupling

- queue + worker model for projection persistence,
- strict-order mode maintained.

### Phase 4: advanced tuning

- coalesced timeline upserts for high-frequency delta updates,
- parsed-envelope cache for debug endpoints.

## Risk assessment

1. Optimization accidentally breaks ordering.
Mitigation: explicit monotonicity tests + per-conversation serialized workers.

2. Queue-based decoupling introduces drops under pressure.
Mitigation: bounded queues, policy modes, and telemetry.

3. Complexity growth hurts maintainability.
Mitigation: phase-gated rollout with benchmarks and clear feature flags.

## Deliverables

1. Performance baseline report.
2. Implemented low-risk improvements with benchmark deltas.
3. Architecture proposal and prototype for decoupled projection writes.
4. Updated docs with throughput/latency guidance.

## Acceptance criteria

1. Baseline and post-change benchmark artifacts checked into ticket docs.
2. sem buffer optimization reduces allocations materially in overflow benchmark.
3. Consume-loop p95 latency improves under representative load.
4. No ordering regressions in seq monotonicity and timeline consistency tests.
