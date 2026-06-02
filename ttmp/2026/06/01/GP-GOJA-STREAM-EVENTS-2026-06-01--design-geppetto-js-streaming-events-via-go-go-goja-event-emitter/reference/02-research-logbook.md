---
Title: Research logbook
Ticket: GP-GOJA-STREAM-EVENTS-2026-06-01
Status: active
Topics:
    - geppetto
    - goja
    - js-bindings
    - streaming
    - events
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/pkg/doc/17-connected-eventemitters-developer-guide.md
      Note: Upstream connected EventEmitter documentation resource
    - Path: ../../../../../../../go-go-goja/pkg/jsevents/manager.go
      Note: Upstream connected EventEmitter manager resource
    - Path: cmd/examples/geppetto-js-run/main.go
      Note: Example runner Promise-waiting resource
    - Path: pkg/js/modules/geppetto/api_agent.go
      Note: Core runAsync implementation resource
    - Path: pkg/js/modules/geppetto/api_event_emitters.go
      Note: EventEmitter sink adapter resource
    - Path: pkg/js/modules/geppetto/api_event_payloads.go
      Note: Payload mapping resource
    - Path: ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/analysis/01-eventemitter-runasync-code-review-and-intern-guide.md
      Note: Follow-up review summarized in logbook
    - Path: ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/design-doc/01-geppetto-js-streaming-events-design-and-implementation-guide.md
      Note: Primary ticket design guide whose useful/stale sections are cataloged
ExternalSources: []
Summary: Logbook of documents and resources consulted while designing, implementing, documenting, and reviewing Geppetto JS EventEmitter runAsync support.
LastUpdated: 2026-06-02T10:55:00.250305758-04:00
WhatFor: Track which resources were useful, outdated, misleading, or need updates for future EventEmitter/runAsync work.
WhenToUse: Use before resuming EventEmitter streaming cleanup, reviewing the implementation, or onboarding a new engineer/intern.
---


# Research logbook

## Goal

This logbook records the documents, upstream resources, internal docs, examples, and source references used while designing, implementing, documenting, and reviewing the Geppetto JavaScript EventEmitter + `agent.runAsync(...)` work.

For each resource, it records:

- what we were researching;
- what we were looking for in that resource;
- why we chose it and what led us to it;
- what was useful;
- what was not useful;
- what was out of date or wrong;
- what needs updating.

## Context

The ticket goal was to connect Geppetto's structured event system to JavaScript through go-go-goja's connected EventEmitter support. The final first-pass API became:

```js
const EventEmitter = require("events");
const events = new EventEmitter();

events.on("text-delta", ev => process.stdout.write(ev.delta));

const agent = gp.agent()
  .inference(settings)
  .events(events)
  .build();

const handle = agent.runAsync(turn);
const result = await handle.promise;
```

This logbook intentionally includes resources that were useful and resources that turned out to be incomplete, stale, or misleading. Future work should update this log whenever a cleanup task changes the assumptions captured here.

## Quick status index

| Resource | Status | Main reason |
|---|---:|---|
| go-go-goja connected EventEmitters guide | Useful, needs cross-linking | Correct mental model for owner-thread-safe EventEmitter adoption. |
| go-go-goja async patterns guide | Useful, partial | Explains owner-thread scheduling, but not Geppetto-specific run lifecycle. |
| Geppetto event context/sink code | Useful, current | Shows the right injection point for JS EventEmitter sinks. |
| Geppetto JS `api_agent.go` | Useful, needs cleanup | Current implementation exists, but lifecycle and owner-thread issues remain. |
| EventEmitter review doc | Current, actionable | Captures current quality risks and cleanup phases. |
| Initial streaming design guide | Useful but partially superseded | Early per-stream/`handle.on` design is now out of date. |
| JS API reference | Updated, needs continued maintenance | Now documents `runAsync`, but payload contract will drift unless tested. |
| Example runner | Useful, recently fixed | Promise detection bug caused zero-output examples; now fixed. |
| xgoja provider path | Needs updating | Likely lacks EventEmitter manager wiring for provider-created modules. |
| Real-provider examples | Useful, manual only | Good smoke examples, but not automated and provider event behavior varies. |

## Resource entries

### 1. Ticket design guide: `design-doc/01-geppetto-js-streaming-events-design-and-implementation-guide.md`

- **What we were researching:** The intended architecture for JavaScript-visible streaming events in Geppetto.
- **What we were looking for:** The proposed public API, event naming, payload mapping, runtime wiring, phases, and the relationship between `agent.stream`, `agent.runAsync`, builder-level `.events(...)`, and per-run emitters.
- **Why we chose it / what led us to it:** This was the primary design deliverable for the ticket and the first place to reconcile implementation direction with user feedback.
- **What was useful:**
  - Captured the core architecture: Geppetto events should flow through `events.EventSink` into go-go-goja EventEmitters.
  - Documented why `handle.on(...)` is racy if the run starts before listener registration.
  - Evolved toward the correct first-pass API: builder-level EventEmitter plus `runAsync`.
  - Contains diagrams, implementation phases, testing guidance, and open questions.
- **What was not useful:**
  - Earlier sections were polluted by the initial `agent.stream(turn, { events })` and `handle.on(...)` design before being revised.
  - Some phase descriptions remained more ambitious than the implementation: per-run sinks, adapter lifecycle events, and richer TypeScript unions.
- **What is out of date / wrong:**
  - Any text implying `agent.stream(...)` is the public execution method is obsolete.
  - Any text implying per-run `{ events: emitter }` is first-pass behavior is obsolete.
  - Any text suggesting generic `RuntimeValues map[string]any` is wrong; the implementation moved to a typed manager/resolver.
- **What needs updating:**
  - Keep the design guide aligned with the final API: `.events(emitter)` + `runAsync`.
  - Add a follow-up section summarizing the code review findings and cleanup priorities.
  - Replace any remaining `stream-*` adapter lifecycle naming with `runasync-*` or defer lifecycle naming entirely.

### 2. EventEmitter runAsync code review: `analysis/01-eventemitter-runasync-code-review-and-intern-guide.md`

- **What we were researching:** The correctness and maintainability of the implemented EventEmitter + `runAsync` work.
- **What we were looking for:** Deprecated code, messy code, missing cleanup, owner-thread problems, host integration gaps, incomplete payload contracts, and untested behavior.
- **Why we chose it / what led us to it:** It was created after implementation to give a new intern a technical map and to identify what remains risky.
- **What was useful:**
  - Clear explanation of the current event flow from JS EventEmitter to Geppetto `EventSink` to provider events and back to JS callbacks.
  - Identifies high-risk issues:
    - builder-level EventEmitter refs are never closed;
    - `runAsync` prepares sessions on a goroutine even though some paths touch goja;
    - xgoja provider path likely lacks EventEmitter manager wiring.
  - Provides a concrete cleanup plan with phases.
- **What was not useful:**
  - It is a review, not an implementation spec. Some recommendations still need task breakdown and acceptance criteria.
- **What is out of date / wrong:**
  - Current as of commit `cf0f4f7b`; it should be rechecked after any lifecycle or xgoja provider cleanup.
- **What needs updating:**
  - Convert review findings into tasks.
  - Update after implementing `agent.close()` or per-run sink scoping.
  - Update after real-provider smoke runs record observed event types.

### 3. Ticket evidence source: `sources/01-code-evidence.md`

- **What we were researching:** Line-numbered evidence for the streaming design and implementation review.
- **What we were looking for:** Anchors for claims about current code: event sinks, session execution, `agent.stream`/`runAsync`, go-go-goja EventEmitter adoption, and runtime module setup.
- **Why we chose it / what led us to it:** The ticket workflow required evidence-first analysis before writing conclusions.
- **What was useful:**
  - Consolidated line-numbered excerpts into one source file.
  - Good for writing the initial design guide without repeatedly opening many source files.
- **What was not useful:**
  - It can become stale quickly after implementation changes.
  - It is a snapshot, not a live index; line numbers may drift.
- **What is out of date / wrong:**
  - Any excerpt from before the `runAsync` implementation may refer to the removed `agent.stream` handle-local collector.
- **What needs updating:**
  - Regenerate after cleanup work, especially after changing `api_agent.go`, provider integration, or runtime wiring.

### 4. Ticket diary: `reference/01-investigation-diary.md`

- **What we were researching:** Chronological implementation history, reasoning, failures, and validation steps.
- **What we were looking for:** Why design decisions changed, what broke, and which commands validated the work.
- **Why we chose it / what led us to it:** The diary is the authoritative continuation record for this ticket.
- **What was useful:**
  - Records the shift from `stream`/per-run emitter thinking to builder-level `.events(emitter)` + `runAsync`.
  - Records the `RuntimeValues` concern and the move to typed manager/resolver plumbing.
  - Records the zero-output runner bug and missing-profile reproduction.
- **What was not useful:**
  - Earlier steps contain superseded design assumptions; readers must treat it chronologically, not as final API reference.
- **What is out of date / wrong:**
  - Early diary entries that recommend `agent.stream(turn, { events })` are now historical, not current guidance.
- **What needs updating:**
  - Add entries for any cleanup implementation.
  - Record real-provider smoke outputs and observed event types.

### 5. Geppetto JS API reference: `pkg/doc/topics/13-js-api-reference.md`

- **What we were researching:** Public documentation for the final hard-cut JS API and the new `runAsync` EventEmitter API.
- **What we were looking for:** Whether users can discover `runAsync`, event names, payload fields, handle shape, absent APIs, and example commands.
- **Why we chose it / what led us to it:** The user explicitly requested a JS API reference at minimum.
- **What was useful:**
  - Now documents `RunOptions`, `AgentAsyncHandle`, `runAsync`, builder-level `.events(emitter)`, event names, common payload fields, and deferred APIs.
  - Explicitly states `agent.stream`, per-run `{ events }`, and `handle.on` are absent.
- **What was not useful:**
  - It can only describe payloads that the current encoder supports. The source encoder is hand-maintained, so docs can drift.
- **What is out of date / wrong:**
  - Should be checked whenever `api_event_payloads.go` changes.
  - May over-imply stability for payload fields that do not yet have exhaustive tests.
- **What needs updating:**
  - Add a troubleshooting subsection:
    - missing profile produces `profile not found`;
    - zero deltas may be provider behavior;
    - zero output before commit `5ce221a5` was a runner bug.
  - Add `gp.events.collector()` positioning as advanced/legacy interop.

### 6. Geppetto JS user guide: `pkg/doc/topics/14-js-api-user-guide.md`

- **What we were researching:** Practical user-level explanation of composing settings, agents, turns, tools, multimodal messages, and live events.
- **What we were looking for:** Where to place a narrative `runAsync` section and constraints.
- **Why we chose it / what led us to it:** Users need a recipe, not only a reference table.
- **What was useful:**
  - Now includes a live-events usage section with EventEmitter setup and constraints.
  - Reinforces listener registration before `runAsync`.
- **What was not useful:**
  - It does not go deep into host/runtime setup or lifecycle problems.
- **What is out of date / wrong:**
  - Current enough for public use, but does not mention the cleanup risks from the code review.
- **What needs updating:**
  - Add troubleshooting examples and real-provider caveats after more smoke testing.

### 7. Getting-started tutorial: `pkg/doc/tutorials/05-js-api-getting-started.md`

- **What we were researching:** A step-by-step tutorial path that includes live EventEmitter events.
- **What we were looking for:** Where a beginner should learn `runAsync` relative to sync `run`, tools, and multimodal turns.
- **Why we chose it / what led us to it:** The new API should not be hidden in reference docs only.
- **What was useful:**
  - Now has a dedicated step for EventEmitter + `runAsync`.
  - Includes commands for real-provider examples.
- **What was not useful:**
  - Tutorial intentionally omits intern-level architecture and cleanup caveats.
- **What is out of date / wrong:**
  - Needs updates if example names or runner flags change.
- **What needs updating:**
  - Add a short note that `text-delta` may be absent for some providers.

### 8. Example README: `examples/js/geppetto/README.md`

- **What we were researching:** Discoverability of runnable JS examples.
- **What we were looking for:** Whether a user can find commands to run EventEmitter examples against a real provider.
- **Why we chose it / what led us to it:** User asked how to run the examples and then reported zero output.
- **What was useful:**
  - Lists `31`, `32`, and `33` EventEmitter examples.
  - Includes full `go run ./cmd/examples/geppetto-js-run run ...` commands.
- **What was not useful:**
  - It does not include troubleshooting for Promise waiting or missing profiles.
- **What is out of date / wrong:**
  - Current, but should mention the minimum commit/version containing the runner fix if users may have older checkouts.
- **What needs updating:**
  - Add troubleshooting:
    - no output means old runner or a script that did not return a Promise;
    - missing profile should produce a non-zero error;
    - zero deltas does not necessarily mean failure.

### 9. Example scripts: `31_event_emitter_run_async.js`, `32_event_emitter_progress_summary.js`, `33_event_emitter_multiturn_run_async.js`

- **What we were researching:** How users should consume EventEmitter events in real JavaScript.
- **What we were looking for:** Minimal, progress-summary, and multi-turn examples that exercise the final API shape.
- **Why we chose it / what led us to it:** The user asked for examples showcasing streaming events.
- **What was useful:**
  - `31` demonstrates the smallest real-provider smoke.
  - `32` records generic and type-specific event counts.
  - `33` shows one builder-level emitter reused across two explicit `runAsync` turns.
- **What was not useful:**
  - They are manual examples; no automated real-provider CI.
  - They do not prove all providers stream token deltas.
- **What is out of date / wrong:**
  - Current as examples, but they rely on runner promise-waiting support from commit `5ce221a5`.
- **What needs updating:**
  - Add a wrapper script to run all EventEmitter examples.
  - Add expected-output examples with notes about provider variability.

### 10. go-go-goja connected EventEmitters developer guide: `pkg/doc/17-connected-eventemitters-developer-guide.md`

- **What we were researching:** How go-go-goja expects Go code to connect to JavaScript-created EventEmitters.
- **What we were looking for:** Correct adoption API, owner-thread restrictions, `EmitterRef` lifecycle, and emission patterns.
- **Why we chose it / what led us to it:** The ticket explicitly proposed using go-go-goja connected EventEmitters instead of direct callback invocation.
- **What was useful:**
  - Established the correct mental model: adopt JS EventEmitter on the owner thread, then emit through a Go ref that schedules work back to the owner.
  - Supported the design decision to avoid calling JS callbacks directly from provider goroutines.
- **What was not useful:**
  - It is upstream/general; it does not tell Geppetto how to scope sinks or how to map canonical event payloads.
- **What is out of date / wrong:**
  - It may not mention Geppetto's final `runAsync` naming or builder-level-only first pass.
- **What needs updating:**
  - Cross-link from Geppetto JS docs to this guide, or add a short Geppetto-specific connected EventEmitter section.

### 11. go-go-goja async patterns guide: `pkg/doc/03-async-patterns.md`

- **What we were researching:** Owner-thread scheduling and asynchronous JavaScript execution patterns.
- **What we were looking for:** Why sync `run()` cannot deliver live JS callbacks and why `runAsync` needs to return control to the owner.
- **Why we chose it / what led us to it:** The user asked whether `stream()`/`runAsync` was needed if `run()` exists.
- **What was useful:**
  - Reinforced owner-thread constraints and asynchronous scheduling patterns.
  - Helped explain why synchronous `run()` cannot be a live callback API.
- **What was not useful:**
  - It does not cover Geppetto sessions, `ExecutionHandle`, or provider streaming.
- **What is out of date / wrong:**
  - Not wrong; just not Geppetto-specific.
- **What needs updating:**
  - Add Geppetto-specific examples elsewhere; do not change upstream doc unless go-go-goja wants a Geppetto example.

### 12. go-go-goja `pkg/jsevents/manager.go`

- **What we were researching:** The actual connected EventEmitter implementation.
- **What we were looking for:** `AdoptEmitterOnOwner`, `EmitWithBuilder`, lifecycle close semantics, and runtime manager lookup.
- **Why we chose it / what led us to it:** Needed to implement `jsEventEmitterSink` correctly.
- **What was useful:**
  - `AdoptEmitterOnOwner` proves adoption validates go-go-goja EventEmitter values.
  - `EmitWithBuilder` schedules delivery through `owner.Post`.
  - `Close` unregisters the ref from the manager.
- **What was not useful:**
  - Does not solve Geppetto's builder-level lifetime problem by itself.
- **What is out of date / wrong:**
  - Current enough, but Geppetto `go.mod` had to align to `go-go-goja v0.7.2` for workspace/provider API compatibility.
- **What needs updating:**
  - Geppetto should use `Close` deterministically, either per-run or via `agent.close()`.

### 13. go-go-goja `modules/events/events.go`

- **What we were researching:** What JavaScript receives from `require("events")` and how EventEmitter values are represented internally.
- **What we were looking for:** Constructor behavior, `FromValue`, supported methods, and event semantics.
- **Why we chose it / what led us to it:** Needed to know what `.events(emitter)` can safely accept.
- **What was useful:**
  - Confirmed the EventEmitter is Go-backed and can be unwrapped by `eventsmodule.FromValue`.
  - Listed supported JS methods: `on`, `once`, `off`, `removeAllListeners`, `emit`, etc.
- **What was not useful:**
  - It does not include Geppetto-specific TypeScript types or event payload docs.
- **What is out of date / wrong:**
  - No known wrong content.
- **What needs updating:**
  - Geppetto docs should say the accepted EventEmitter is the go-go-goja EventEmitter from `require("events")`, not an arbitrary object with `.emit`.

### 14. go-go-goja `pkg/jsevents` tests and examples (`manager_test.go`, `fswatch.go`, `watermill.go`)

- **What we were researching:** Existing examples of adopting EventEmitters and connecting Go resources to JS EventEmitter callbacks.
- **What we were looking for:** Test patterns for adoption, async emit, close, and helper modules.
- **Why we chose it / what led us to it:** Needed implementation and testing patterns for `api_event_emitters_test.go`.
- **What was useful:**
  - `manager_test.go` shows owner-thread adoption and `EmitSync` assertions.
  - `fswatch.go` and `watermill.go` show real resource-to-EventEmitter connection patterns.
- **What was not useful:**
  - These examples are resource connectors, not inference-run connectors; their lifecycle model does not directly decide Geppetto's agent/run scope.
- **What is out of date / wrong:**
  - Not wrong, but they may encourage persistent resource refs if copied without considering Geppetto run lifetimes.
- **What needs updating:**
  - Geppetto should add its own tests for per-run cleanup or `agent.close()` rather than relying on upstream examples.

### 15. Geppetto event code: `pkg/events/context.go`, `chat-events.go`, `canonical_events.go`, `canonical_tool_events.go`, `metadata.go`

- **What we were researching:** Canonical Geppetto event types, metadata, and sink publication.
- **What we were looking for:** Event type names, struct fields, correlation metadata, and where sinks are attached/retrieved.
- **Why we chose it / what led us to it:** JavaScript payload mapping must be based on canonical Go events.
- **What was useful:**
  - `context.go` gave the sink injection point.
  - `chat-events.go` lists canonical event names such as `text-delta`, `provider-call-started`, and `error`.
  - canonical event structs define typed fields that should be carried to JS.
- **What was not useful:**
  - There is no single generated payload contract for JavaScript.
  - Some events have fields not yet mapped by `api_event_payloads.go`.
- **What is out of date / wrong:**
  - The event system is current, but the JS payload encoder is incomplete relative to it.
- **What needs updating:**
  - Add payload coverage tests for all canonical event types used by JS.
  - Consider a registry-driven or interface-driven payload encoder.

### 16. Geppetto session/enginebuilder code: `pkg/inference/session/execution.go`, `session.go`, `toolloop/enginebuilder/builder.go`, `options.go`

- **What we were researching:** How an agent run starts, waits, cancels, and receives event sinks.
- **What we were looking for:** `ExecutionHandle`, `Cancel`, `Wait`, `WithEventSinks`, and the run context path.
- **Why we chose it / what led us to it:** `runAsync` needed a promise/cancel handle backed by the actual inference execution handle.
- **What was useful:**
  - `ExecutionHandle.Cancel()` is the right cancellation target.
  - `enginebuilder.WithEventSinks(...)` already injects sinks into the run context.
- **What was not useful:**
  - The session APIs do not directly express JavaScript lifecycle or owner-thread requirements.
- **What is out of date / wrong:**
  - Not wrong; the risk is in how `api_agent.go` uses it from a goroutine.
- **What needs updating:**
  - Add a helper that separates owner-thread preparation from background waiting.
  - Use runtime lifetime context as a base context where possible.

### 17. Geppetto JS `api_agent.go`

- **What we were researching:** The agent builder/object implementation and the new `runAsync` control flow.
- **What we were looking for:** Public methods, event sink storage, cancellation, run setup, and Promise settlement.
- **Why we chose it / what led us to it:** This is the main implementation file for the public API.
- **What was useful:**
  - Clear path for `.events(...)` to append sinks.
  - `runAsync` returns the intended minimal handle.
  - Cancellation reaches both `ExecutionHandle.Cancel()` and run context cancel.
- **What was not useful:**
  - `runAsync` currently starts session setup on a goroutine; this is suspicious because `buildSession()` can touch `a.api.vm` when `toolLoop` options exist.
  - No explicit agent close/lifecycle.
  - Promise rejection values are strings, not JS `Error` objects.
- **What is out of date / wrong:**
  - No public `stream` remains, which is correct. Any nearby comments/docs referencing stream should be removed.
- **What needs updating:**
  - Add owner-thread-safe preparation.
  - Add lifecycle cleanup.
  - Add nil output checks.
  - Improve rejection values.

### 18. Geppetto JS `api_event_emitters.go`

- **What we were researching:** EventEmitter-backed EventSink adapter.
- **What we were looking for:** Manager lookup, adoption, publish behavior, and close behavior.
- **Why we chose it / what led us to it:** This is the bridge between Geppetto events and JS EventEmitter listeners.
- **What was useful:**
  - Small implementation; easy to reason about.
  - Uses `jsevents.Manager.AdoptEmitterOnOwner` and `EmitterRef.EmitWithBuilder`, which is the right upstream API.
  - Maps every Geppetto event to both `event` and a type-specific event name.
- **What was not useful:**
  - Does not log publish failures; errors are often swallowed by `PublishEventToContext`.
  - Close exists but is not called for builder-level refs.
  - Uses `context.Background()` for emits.
- **What is out of date / wrong:**
  - Not wrong, but incomplete lifecycle behavior makes the adapter unsafe for long-lived agent churn.
- **What needs updating:**
  - Add deterministic close path.
  - Log emit failures.
  - Consider run-scoped adoption/close while preserving JS EventEmitter callbacks.

### 19. Geppetto JS `api_event_payloads.go`

- **What we were researching:** JavaScript payload encoding for Geppetto events.
- **What we were looking for:** Which event types are mapped and how fields are named.
- **Why we chose it / what led us to it:** The API reference and examples depend on this payload contract.
- **What was useful:**
  - Centralizes payload encoding shared by EventEmitter sinks and the old collector.
  - Maps text, reasoning, tool, error, and interrupt events.
  - Maps canonical `error` type-specific emission to `inference-error`.
- **What was not useful:**
  - Hand-maintained switch statement misses many canonical event fields.
  - No compile-time reminder when a new event type is added.
- **What is out of date / wrong:**
  - Incomplete relative to Geppetto's full canonical event set.
- **What needs updating:**
  - Add table-driven tests and richer provider/run lifecycle fields.
  - Consider generated or interface-based payload mapping.

### 20. Geppetto JS runtime: `pkg/js/runtime/runtime.go`

- **What we were researching:** How owned JavaScript runtimes install Geppetto and go-go-goja modules.
- **What we were looking for:** Where to install `jsevents.Install()` and how to pass the manager to the Geppetto module.
- **Why we chose it / what led us to it:** EventEmitter adoption needs a per-runtime `jsevents.Manager`.
- **What was useful:**
  - Correct place for the example runner and owned runtime path.
  - Installs `jsevents.Install()` by default unless already provided.
  - Provides typed resolver closure.
- **What was not useful:**
  - Only covers `pkg/js/runtime.NewRuntime`, not every xgoja provider host path.
- **What is out of date / wrong:**
  - Current implementation is OK for owned runtime path.
- **What needs updating:**
  - Add tests for duplicate initializer behavior.
  - Ensure provider-hosted runtimes have an equivalent manager injection path.

### 21. xgoja provider: `pkg/js/modules/geppetto/provider/provider.go` and `provider_test.go`

- **What we were researching:** Whether generated xgoja hosts can use the new EventEmitter support.
- **What we were looking for:** Host service boundaries and module-loader construction.
- **Why we chose it / what led us to it:** The JS module is exposed both through owned runtime and provider integration.
- **What was useful:**
  - Shows profile registry config and host-provided options path.
  - Shows the provider returns `geppettomodule.NewLoader(opts)` without direct runtime context access.
- **What was not useful:**
  - Does not yet explain how EventEmitter manager/resolver should be supplied for provider-created modules.
- **What is out of date / wrong:**
  - Tests had to be updated for newer go-go-goja `HostServices.AssetResolver()` expectations.
  - The provider integration likely needs explicit EventEmitter manager support.
- **What needs updating:**
  - Add provider integration test that verifies `.events(new EventEmitter())` works.
  - Add host service extension or documented host option for EventEmitter manager/resolver.

### 22. Example runner: `cmd/examples/geppetto-js-run/main.go`

- **What we were researching:** Why EventEmitter examples produced zero output.
- **What we were looking for:** Whether returned JS Promises were detected and waited on.
- **Why we chose it / what led us to it:** User reported running examples produced no output.
- **What was useful:**
  - Exposes profile registry flags and default profile to examples.
  - After the fix, detects a returned JS Promise by exporting the `goja.Value` on the owner thread and waits for it.
  - Missing profiles now surface as visible errors.
- **What was not useful:**
  - Polls Promise state every 10ms; sufficient for examples, not a general async runtime design.
  - Detects native goja Promise, not arbitrary thenables.
- **What is out of date / wrong:**
  - Before commit `5ce221a5`, Promise detection was wrong and examples could exit silently.
- **What needs updating:**
  - Add a regression test or small async example runner test.
  - Document that async example scripts should return a Promise.

### 23. Type declarations: `pkg/doc/types/geppetto.d.ts` and `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`

- **What we were researching:** Public TypeScript shape for `runAsync` and EventEmitter sinks.
- **What we were looking for:** Whether declaration files match runtime exports.
- **Why we chose it / what led us to it:** The hard-cut JS API has DTS parity tests; stale declarations previously caused confusion.
- **What was useful:**
  - Now exposes `AgentAsyncHandle`, `EventEmitterLike`, and `runAsync`.
  - Removes public `stream`/`handle.on` from declarations.
- **What was not useful:**
  - Event payload types are still `any`-ish; there is no discriminated union for event payloads.
- **What is out of date / wrong:**
  - Current for the runtime surface, but incomplete for typed event payloads.
- **What needs updating:**
  - Add structured event payload types after payload coverage is hardened.

### 24. Obsidian hard-cut article: `ARTICLE - Geppetto JS Bindings - Wrapper First Hard Cutover.md`

- **What we were researching:** Background context for the hard-cut wrapper-first API.
- **What we were looking for:** The principles that should constrain the EventEmitter work: Go-owned wrappers, explicit turns, registry-backed settings, and no legacy map/session/runner API.
- **Why we chose it / what led us to it:** The EventEmitter API needed to fit the existing hard-cut architecture.
- **What was useful:**
  - Reinforced that JS APIs should be explicit and wrapper-first.
  - Helped reject adding broad legacy-ish shims.
- **What was not useful:**
  - It predates EventEmitter `runAsync` and does not explain streaming events.
- **What is out of date / wrong:**
  - It should be updated or followed by a new note if EventEmitter streaming becomes a stable API.
- **What needs updating:**
  - Add a follow-up Obsidian note after cleanup work stabilizes `runAsync` and event lifecycle.

### 25. reMarkable upload skill/reference

- **What we were researching:** How to upload ticket documents as a PDF bundle to reMarkable.
- **What we were looking for:** The correct `remarquee upload bundle` command and remote path conventions.
- **Why we chose it / what led us to it:** The user requested upload to reMarkable.
- **What was useful:**
  - Current command guidance says to upload directly with `--non-interactive` and avoid unnecessary status/list calls.
  - Avoided extra auth/status tool calls.
- **What was not useful:**
  - Operational only; not relevant to EventEmitter architecture.
- **What is out of date / wrong:**
  - No issue encountered.
- **What needs updating:**
  - N/A for the EventEmitter ticket.

## Maintenance checklist for this logbook

When resuming this ticket, update this logbook if any of these happen:

- `agent.close()` or per-run EventEmitter sink scoping is implemented.
- xgoja provider EventEmitter manager integration changes.
- `api_event_payloads.go` adds or removes payload fields.
- `gp.events.collector()` is deprecated, removed, or repositioned.
- Real-provider smoke output is recorded.
- The examples runner changes its async/Promise behavior.
- go-go-goja EventEmitter APIs change.

## Related documents

- `design-doc/01-geppetto-js-streaming-events-design-and-implementation-guide.md`
- `analysis/01-eventemitter-runasync-code-review-and-intern-guide.md`
- `reference/01-investigation-diary.md`
- `sources/01-code-evidence.md`
- `pkg/doc/topics/13-js-api-reference.md`
- `pkg/doc/topics/14-js-api-user-guide.md`
- `pkg/doc/tutorials/05-js-api-getting-started.md`
