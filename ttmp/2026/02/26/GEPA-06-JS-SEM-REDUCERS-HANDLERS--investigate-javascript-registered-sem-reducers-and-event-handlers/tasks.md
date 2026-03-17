# Tasks

## Completed

- [x] Create ticket and seed design/diary docs.
- [x] Investigate `geppetto` event model and JS event subscription APIs.
- [x] Investigate `pinocchio` SEM translation/projection ownership and extension points.
- [x] Investigate `go-go-os` SEM handler/reducer registration model and runtime module surfaces.
- [x] Validate GEPA-04 streaming-event additions in current `go-go-gepa` code.
- [x] Run prototype experiment demonstrating handler overwrite vs composable reducer/handler model.
- [x] Produce exhaustive architecture document and chronological diary.

## Option C Implementation Plan (Pinocchio Backend JS Reducer Runtime)

### Task 1 - Backend JS runtime core + registry bridge

- [x] Add `pkg/webchat/timeline_js_runtime.go` implementing a Goja-backed runtime that supports:
  - `registerSemReducer(eventType, fn)`
  - `onSem(eventType, fn)`
  - wildcard `*` handlers
  - reducer return decoding into `TimelineEntityV2` upserts
- [x] Extend `pkg/webchat/timeline_registry.go` to support a pluggable runtime bridge and execution ordering.
- [x] Add focused unit tests for runtime registration, wildcard handling, reducer output decoding, and error containment.
- [x] Commit Task 1 as isolated commit (`pinocchio` commit `99c2bfd`).

### Task 2 - Startup wiring + configuration surface

- [x] Add startup wiring in `cmd/web-chat/main.go` for loading one or more JS reducer files.
- [x] Add CLI/config options for reducer script paths.
- [x] Register runtime before chat service/stream processing starts.
- [x] Add tests for loader/wiring behavior.
- [x] Commit Task 2 as isolated commit (`pinocchio` commits `f33fb55`, `4a87c5f` follow-up duplicate-flag fix).

### Task 3 - gpt-5-nano profile validation scripts + integration checks

- [x] Add ticket scripts under `scripts/` to run a local validation flow with gpt-5-nano profile.
- [x] Exercise runtime with at least one reducer/handler script reacting to llm delta semantics.
- [x] Capture command outputs and findings in diary.
- [x] Commit Task 3 as isolated commit (`pinocchio` commit `4b1a649`).

### Task 4 - Docs/help and operational guardrails

- [x] Update docs/help with JS reducer runtime contract, safety notes, and examples.
- [x] Add runbook/troubleshooting notes for bad reducer scripts and fallback behavior.
- [x] Commit Task 4 as isolated commit (`pinocchio` commit `381ffb7`).

### Task 5 - Runtime builder alignment across pinocchio + geppetto

- [x] Refactor `pinocchio` JS timeline runtime to run on owned go-go-goja runtime lifecycle and runner execution path.
- [x] Add runtime lifecycle shutdown wiring in timeline registry clear/set flows.
- [x] Update projection behavior to tolerate delta-only SEM events when cumulative is omitted.
- [x] Add `geppetto/pkg/js/runtime` helper to bootstrap a runtime that exposes `require(\"geppetto\")` with shared runtime-owner wiring.
- [x] Migrate geppetto JS lab example and tests to builder-owned runtime flow.
- [x] Run targeted regression tests in both repos and record outputs in diary.

## Follow-up (Post-MVP)

- [ ] Decide whether to expose reducer hot-reload vs startup-only behavior.
- [ ] Decide whether to add stateful reducer context (getState/setState) in v2.
- [ ] Evaluate optional integration with GEPA plugin stream-event -> SEM bridge for end-to-end dynamic projections.
