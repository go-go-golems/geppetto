---
title: "Tasks"
doc_type: tasks
topics:
  - geppetto
  - goja
  - js-bindings
  - streaming
  - events
status: active
intent: task-tracking
owners:
  - manuel
created: 2026-06-01
updated: 2026-06-01
---

# Tasks

## Research and documentation

- [x] Create ticket workspace and core documents
- [x] Collect code evidence for Geppetto JS streaming, event sinks, and go-go-goja EventEmitter support
- [x] Map current Geppetto JS `agent.stream` and `events.collector` behavior
- [x] Map Geppetto event sink and session execution flow
- [x] Map go-go-goja EventEmitter and connected-emitter ownership model
- [x] Write intern-ready analysis/design/implementation guide
- [x] Relate key source files to ticket documents
- [x] Validate docmgr ticket
- [x] Upload initial design bundle to reMarkable and verify remote listing
- [x] Update design to final first-pass API: builder-level EventEmitter plus `agent.runAsync(turn)`
- [x] Re-upload updated bundle to reMarkable after implementation notes are current

## Implementation phase 1 — EventEmitter runtime plumbing and payload adapter

- [x] Add typed EventEmitter manager resolver access to Geppetto module options so module code can lazily find the go-go-goja `jsevents.Manager`
- [x] Install `jsevents.Install()` in the Geppetto JS runtime path used by `pkg/js/runtime.NewRuntime`
- [x] Add `api_event_payloads.go` or equivalent factoring so Geppetto event payload encoding is reusable outside `jsEventCollector`
- [x] Add `api_event_emitters.go` with a `jsEventEmitterSink` implementing `events.EventSink`
- [x] Adopt JavaScript-created EventEmitter values through `jsevents.Manager.AdoptEmitterOnOwner`
- [x] Emit both the general `event` notification and type-specific event names, mapping canonical Geppetto `error` to `inference-error`
- [x] Extend `requireEventSink` so `gp.agent().events(emitter)` accepts a go-go-goja EventEmitter while still accepting existing Go `events.EventSink` refs
- [x] Add synthetic EventEmitter sink tests for `text-delta`, `tool-result-ready`, and `error` event-name mapping
- [x] Commit phase 1

## Implementation phase 2 — `agent.runAsync(turn)` execution path

- [x] Add `agent.runAsync(turn, options?)` as the non-blocking JavaScript execution method
- [x] Keep `agent.run(turn, options?)` synchronous and final-result-only
- [x] Replace/remove the public `agent.stream(turn, options?)` method for this first pass, or leave only a documented internal compatibility alias if required by tests
- [x] Implement runAsync by starting inference, returning `{ promise, cancel, close? }`, and resolving/rejecting the promise on the runtime owner
- [x] Wire `cancel()` to the active `ExecutionHandle.Cancel()` and the outer run context cancel function
- [x] Ensure builder-level event sinks are attached before inference starts
- [x] Add tests proving EventEmitter listeners fire before `runAsync().promise` resolves
- [x] Add tests proving `cancel()` reaches a blocking engine's context
- [x] Commit phase 2

## Implementation phase 3 — Type declarations, docs, and examples

- [x] Update `pkg/doc/types/geppetto.d.ts` to expose `runAsync` and EventEmitter-style event sink input
- [x] Update `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl` in sync with the public declaration file
- [x] Update `pkg/doc/topics/13-js-api-reference.md` to replace `stream` with `runAsync`
- [x] Add or update JS examples showing `new EventEmitter()`, `.events(emitter)`, and `await agent.runAsync(turn).promise`
- [x] Update hard-cut tests and DTS parity tests as needed
- [x] Run focused JS tests and hard-cut contract tests
- [x] Commit phase 3

## Implementation phase 4 — Validation, diary, and delivery

- [x] Run `go test ./pkg/js/... ./cmd/examples/geppetto-js-run -count=1`
- [x] Run `go test -tags geppetto_js_hardcut_contract ./pkg/js/modules/geppetto -run TestHardCutPublicSurfaceContract -count=1`
- [x] Run `docmgr doctor --ticket GP-GOJA-STREAM-EVENTS-2026-06-01 --stale-after 30`
- [x] Update diary with implementation outcomes, failures, and review instructions
- [x] Update changelog with commits and related files
- [x] Upload final updated bundle to reMarkable and verify remote listing
- [x] Commit final ticket documentation updates
