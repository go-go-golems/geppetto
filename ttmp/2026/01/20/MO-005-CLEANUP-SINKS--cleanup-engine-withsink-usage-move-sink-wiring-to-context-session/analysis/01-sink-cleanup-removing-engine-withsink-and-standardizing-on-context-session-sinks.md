---
Title: 'Sink cleanup: removing engine.WithSink and standardizing on context/session sinks'
Ticket: MO-005-CLEANUP-SINKS
Status: active
Topics:
    - inference
    - architecture
    - events
    - webchat
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/context.go
      Note: Context sinks semantics (append-only)
    - Path: geppetto/pkg/inference/core/session.go
      Note: Session EventSinks injection per run
    - Path: geppetto/pkg/inference/engine/options.go
      Note: Engine-config sinks (WithSink) targeted for removal
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: CLI blocking uses WithSink today
    - Path: pinocchio/pkg/ui/runtime/builder.go
      Note: Creates engines with WithSink today
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-20T16:05:26.117984547-05:00
WhatFor: ""
WhenToUse: ""
---


---
Summary: "Inventory of engine.WithSink call sites and a migration plan to remove engine-config sinks, standardizing on run-context (Session) sinks without duplications."
RelatedFiles:
  - Path: geppetto/pkg/events/context.go
    Note: WithEventSinks / PublishEventToContext behavior (append-only; no dedup)
  - Path: geppetto/pkg/inference/core/session.go
    Note: Session already supports EventSinks per run
  - Path: geppetto/pkg/inference/engine/options.go
    Note: Current engine.WithSink / Config.EventSinks to be removed
  - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
    Note: Provider engine currently attaches config sinks into ctx (to be deleted when removing WithSink)
  - Path: pinocchio/pkg/ui/backend.go
    Note: TUI backend starts Session without EventSinks; currently relies on engine.WithSink
  - Path: pinocchio/pkg/ui/runtime/builder.go
    Note: Creates engines with engine.WithSink(uiSink); will need Session sinks instead
  - Path: pinocchio/pkg/cmds/cmd.go
    Note: CLI blocking mode uses engine.WithSink(watermillSink) for event routing
  - Path: pinocchio/cmd/examples/simple-redis-streaming-inference/main.go
    Note: Example uses engine.WithSink(sink)
LastUpdated: 2026-01-20T21:20:00-05:00
WhatFor: "Guide removal of engine.WithSink while preserving event streaming/tool-loop behavior across CLI/TUI/webchat."
WhenToUse: "Before changing sink plumbing, adding new runner types, or debugging duplicate/missing inference events."
---

# Sink cleanup: removing `engine.WithSink` and standardizing on context/session sinks

## Goal

We want to remove the “engine-config sinks” API (`engine.WithSink`, `engine.Config.EventSinks`) and make event publishing depend solely on **run-context sinks** (i.e. `events.WithEventSinks(ctx, ...)`), with the preferred injection point being `core.Session.EventSinks`.

This doc provides:

1) a full inventory of `engine.WithSink` call sites (including the explicit “none in moments” conclusion),
2) an explanation of why the split exists today and what can go wrong (missing events, duplicates),
3) a robust migration design that keeps correctness and avoids accidental duplication,
4) a suggested implementation sequence (phased or “big bang”).

## Non-goals

- This doc does **not** migrate moments webchat to `InferenceState`/`Session` (that’s a separate ticket/sequence).
- This doc does **not** redesign the event model; it only addresses sink wiring.

## Background: two sink injection points today

### Context sinks (already the “unified” publisher API)

`events.WithEventSinks(ctx, sinks...)` attaches sinks to the run context.

All publishers that do:

```go
events.PublishEventToContext(ctx, ev)
```

will publish to whatever sinks are attached to `ctx`.

Important property: `events.WithEventSinks` is **append-only**. If you attach the same sink multiple times (engine + session + caller), you will get duplicate event delivery.

### Engine-config sinks (legacy convenience API we want to remove)

`engine.WithSink(sink)` configures sinks on the engine instance.

Provider engines currently “bridge” engine-config sinks into the context at the start of `RunInference` so tool loops/middleware (which publish via context) share the same sinks.

If we remove `engine.WithSink`, we must ensure that:

- all event publishers only use `PublishEventToContext`,
- the run context always has the sinks needed by the runtime (UI/webchat/logging/persistence).

## Inventory: all `engine.WithSink` call sites

### Scope rules for this inventory

- Includes all code (Go) uses in this monorepo workspace.
- Excludes `ttmp/**` and `pkg/doc/**` references (docs often mention the old API and will be updated later).
- Includes tests and fixtures (because they will break if we delete the API).

### Results (code)

#### pinocchio (production)

- `pinocchio/pkg/cmds/cmd.go:324` — blocking CLI mode appends `engine.WithSink(watermillSink)`
- `pinocchio/pkg/cmds/cmd.go:448` — interactive initial step uses `engine.WithSink(chatSink)`
- `pinocchio/pkg/ui/runtime/builder.go:152` — chat builder creates engine with `engine.WithSink(uiSink)`
- `pinocchio/pkg/ui/runtime/builder.go:196` — chat builder creates engine with `engine.WithSink(uiSink)`

#### pinocchio (examples)

- `pinocchio/cmd/examples/simple-redis-streaming-inference/main.go:161` — `factory.NewEngineFromParsedLayers(..., engine.WithSink(sink))`

#### geppetto (tests/fixtures)

- `geppetto/pkg/inference/engine/factory/helpers_test.go:34` — `NewEngineFromStepSettings(..., engine.WithSink(nullSink))`
- `geppetto/pkg/inference/engine/factory/helpers_test.go:52` — same
- `geppetto/pkg/inference/engine/factory/helpers_test.go:56` — same
- `geppetto/pkg/inference/fixtures/fixtures.go:172` — `engOpts := []engine.Option{engine.WithSink(sink)}`
- `geppetto/pkg/inference/fixtures/fixtures.go:219` — `openai_responses.NewEngine(..., engine.WithSink(sink2))`

#### moments (production)

No `engine.WithSink` call sites found in `moments/**` (excluding docs).

This is consistent with moments and go-go-mento already favoring `events.WithEventSinks(...)` at run start:

- `moments/backend/pkg/webchat/router.go:884` — attaches sinks via context
- `go-go-mento/go/pkg/webchat/router.go:726` — attaches sinks via context

## Why move sinks to Session (not InferenceState)

We have three plausible “homes” for event sinks:

1) **Engine** (engine-config sinks) — what we’re removing.
2) **InferenceState** — tempting because it’s “the session state holder”.
3) **Session** — what we already have (`core.Session.EventSinks`).

### Recommendation: sinks belong on Session

Sinks are “stable dependencies” (configuration) that shape *how runs are observed*, not *the mutable run state* itself.

`InferenceState` is intentionally minimal and concurrency-focused:

- engine handle
- current turn
- run id
- running/cancel bookkeeping

Putting sinks into `InferenceState` introduces:

- additional mutable config that must be guarded (or treated as read-only),
- new ordering questions (set sinks before/after StartRun? what if changed mid-run?),
- unclear ownership when multiple runners share the same state.

`core.Session` is already the place where we capture stable deps:

- `EventSinks []events.EventSink`
- tool registry/config
- snapshot hook
- persister

So the clean model is:

- **Session owns sink injection** (per run).
- **State owns cancellation + “current turn”**.
- **Engines publish via context only**.

## Robust sink strategy: invariants and failure modes

### Invariants we want

1) Every run has a well-defined set of sinks (UI/logging/etc).
2) Tool loop and provider engine events go to the same sinks.
3) No duplicate delivery unless explicitly requested.
4) “Cancel” semantics remain correct (sink changes should not break CancelRun).

### Known failure modes today

- **Missing events**: tool loops publish via ctx but ctx has no sinks (or only engine-config sinks exist).
- **Duplicate events**: same sink attached at multiple layers (engine + session + caller ctx).
- **Context reuse accumulation**: because `WithEventSinks` appends, if a long-lived base context already has sinks and each run re-attaches, duplicates can accumulate across turns.

### Recommended invariant: base contexts are sinkless; Session injects sinks per run

In other words:

- don’t put sinks on a long-lived `baseCtx`,
- always create a per-run `runCtx`,
- let `Session.RunInference` / `RunInferenceStarted` inject sinks once.

This avoids accidental accumulation due to append semantics.

If a system needs multiple sinks (router sink + printer sink), that’s fine: pass them as `Session.EventSinks`.

## Migration design options

### Option A (recommended): “Session-only sinks” (remove engine sinks)

Changes:

1) Delete `engine.WithSink` and `engine.Config.EventSinks`.
2) Delete provider-engine “attach config sinks into ctx” glue.
3) Ensure every run path constructs a Session (or explicitly wraps ctx with `events.WithEventSinks`).

Where to attach sinks:

- Prefer: `sess := &core.Session{..., EventSinks: []events.EventSink{...}}`.
- Acceptable for simple one-offs: `runCtx := events.WithEventSinks(ctx, sinks...); eng.RunInference(runCtx, seed)`.

### Option B (transitional): keep engine sinks, but treat them as “default session sinks”

If we need a gradual migration, we can treat engine-config sinks as a default set that a Session copies into its own `EventSinks` once (at Session construction).

This still leaves two sources of truth and keeps the duplicate risk, so it’s strictly transitional.

### Option C (InferenceState sinks)

Store sinks in `InferenceState` and have Session read them.

This is workable but blurs responsibilities (state vs configuration). It also complicates future persistence/serialization of state (sinks aren’t serializable).

## Concrete per-call-site migration notes (how to remove WithSink without regressions)

### 1) Pinocchio TUI (chat builder + backend)

Current situation:

- `pinocchio/pkg/ui/runtime/builder.go` creates engine with `engine.WithSink(uiSink)`.
- `pinocchio/pkg/ui/backend.go` creates a Session with `State: e.inf` and **no EventSinks**.
- Therefore, the system currently relies on engine-config sinks to get events to the UI.

Migration:

- Make the backend/session carry the sink:
  - `EngineBackend` should store `eventSinks []events.EventSink` (or a `*core.Session` instance).
  - When it builds a Session in `Start`, pass `EventSinks: e.eventSinks`.
- Then remove `engine.WithSink(uiSink)` from `runtime/builder.go`.

Pseudocode target:

```go
backend := NewEngineBackend(eng, uiSink) // store sinks
sess := &core.Session{State: backend.inf, EventSinks: backend.sinks}
sess.RunInferenceStarted(runCtx, seed)
```

### 2) Pinocchio CLI blocking mode (PinocchioCommand)

Current:

- Creates `WatermillSink` and passes it via `engine.WithSink`.
- Calls `engine.RunInference(ctx, seed)` directly.

Migration options:

- Simple: wrap the context:

```go
runCtx := events.WithEventSinks(ctx, watermillSink)
updated, err := eng.RunInference(runCtx, seed)
```

- Better (consistent): use a Session even for single-pass:

```go
inf := state.NewInferenceState(runID, nil, eng)
sess := &core.Session{State: inf, EventSinks: []events.EventSink{watermillSink}}
updated, err := sess.RunInference(ctx, seed)
```

### 3) pinocchio example (simple-redis-streaming-inference)

Replace `engine.WithSink(sink)` with a context sink injection, or use a Session.

### 4) geppetto fixtures + engine factory tests

These tests/fixtures exist largely to validate “engine factory can build engines”.

Migration:

- Remove `engine.WithSink(nullSink)` and instead:
  - test event publishing by attaching a sink to context, or
  - remove sink assertions entirely if those tests don’t truly cover sink delivery.

## Provider-engine implications

After removal:

- Provider engines should no longer store sink config.
- Provider engines should publish exclusively via `events.PublishEventToContext(ctx, ev)`.
- Any code currently doing `ctx = events.WithEventSinks(ctx, e.config.EventSinks...)` is deleted.

This also means:

- engine constructors no longer need `options ...engine.Option` for sinks.
- `engine/options.go` may be deleted entirely if it becomes unused (or repurposed).

## Suggested implementation sequence

### Phase 1: migrate all call sites (no API deletion yet)

- Update pinocchio:
  - TUI builder/backend: move sink to Session.
  - CLI blocking: wrap ctx or use Session.
  - example: wrap ctx or use Session.
- Update geppetto fixtures/tests to not depend on WithSink.

At the end of Phase 1, `engine.WithSink` is unused in code (only docs mention it).

### Phase 2: delete `engine.WithSink` and all config-sink glue

- Delete `engine/options.go` or remove sink fields/options.
- Delete provider-engine “attach config sinks to ctx” lines.
- Delete any factory signatures/options if they become dead weight.

### Phase 3: docs + smoke tests

- Update docs that mention `engine.WithSink` to instead recommend `Session.EventSinks` or `events.WithEventSinks`.
- Add/update smoke tests:
  - ensure tool-loop events are still delivered
  - ensure streaming UI still receives events

## Open questions / decisions (to settle early)

1) Do we want to allow non-Session direct engine usage long-term?
   - If yes, we should document “wrap ctx with sinks”.
   - If no, we can simplify: all callers must go through `core.Session`.

2) Do we need sink deduplication in `events.WithEventSinks`?
   - If we enforce “Session is the only injector”, we can avoid dedup code.
   - If we allow multiple injection points, we likely need a dedup API (but it’s non-trivial to do generically for interface values).

3) Do we want sinks on `Session` only, or also on `EngineBuilder` (to standardize construction)?
   - Likely yes: `EngineBuilder.Build(...)` can return an engine, and Session wiring can happen alongside it (or via a higher-level “SessionBuilder”).
