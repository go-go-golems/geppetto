---
Title: Investigation Diary
Ticket: GEPA-06-JS-SEM-REDUCERS-HANDLERS
Status: active
Topics:
    - gepa
    - event-streaming
    - js-vm
    - sem
    - pinocchio
    - geppetto
    - go-go-os
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts
      Note: |-
        Frontend SEM handler registration and overwrite behavior
        Diary records frontend handler semantics evidence
    - Path: geppetto/pkg/js/modules/geppetto/api_events.go
      Note: JS event collector and wildcard behavior
    - Path: go-go-gepa/cmd/gepa-runner/stream_cli_integration_test.go
      Note: Diary captures GEPA-04 validation evidence
    - Path: go-go-gepa/ttmp/2026/02/26/GEPA-06-JS-SEM-REDUCERS-HANDLERS--investigate-javascript-registered-sem-reducers-and-event-handlers/design-doc/01-javascript-registered-sem-reducers-and-event-handler-architecture.md
      Note: |-
        Final synthesis document
        Diary links final synthesis
    - Path: go-go-gepa/ttmp/2026/02/26/GEPA-06-JS-SEM-REDUCERS-HANDLERS--investigate-javascript-registered-sem-reducers-and-event-handlers/scripts/js-sem-reducer-handler-prototype.js
      Note: Diary logs prototype command/output
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: |-
        Backend SEM projection flow
        Diary records projection ownership evidence
    - Path: scripts/js-sem-reducer-handler-prototype.js
      Note: Ticket-local prototype for handler/reducer semantics
ExternalSources: []
Summary: Chronological investigation log for JS-registered SEM reducers and event handler feasibility across geppetto, pinocchio, and go-go-os.
LastUpdated: 2026-02-27T10:20:00-05:00
WhatFor: Preserve exact commands, findings, and reasoning trail for follow-up implementation
WhenToUse: Use when continuing implementation planning for JS reducer/event-handler architecture
---






# Investigation Diary

## Goal

Investigate whether and where JavaScript can register SEM reducers/projection code and event handlers (including reactions to `llm.delta`), and clarify layer ownership (`geppetto` vs `pinocchio` vs frontend runtime).

## Context

User request asked for:

1. A separate expansive report.
2. Direct focus on JS-registered SEM reducers/projection and JS event handlers.
3. Clarification of which system owns SEM projection.
4. Ongoing diary + reMarkable delivery.

User also added an important update during the investigation:

- **GEPA-04 already added streaming events**.

This was incorporated and validated in code (not just accepted as assumption).

---

## Quick Reference

## Core outcome

1. `geppetto`: JS event handlers for geppetto events already exist (`on(eventType, callback)`), including delta-equivalent stream events (`partial`).
2. `pinocchio` backend: SEM projection exists, but registration is Go-only today.
3. `go-go-os` frontend app runtime: JS/TS SEM handler registration exists; however, handler map is single-handler-per-type and sandbox plugins do not expose SEM subscription APIs.
4. `go-go-gepa` GEPA-04: plugin stream events exist (`emitEvent`, `events.emit`, `--stream`) but are not automatic backend SEM reducer registration.

## Most important files

1. `geppetto/pkg/js/modules/geppetto/api_sessions.go`
2. `geppetto/pkg/js/modules/geppetto/api_events.go`
3. `pinocchio/pkg/webchat/timeline_projector.go`
4. `pinocchio/pkg/webchat/timeline_registry.go`
5. `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts`
6. `go-go-gepa/cmd/gepa-runner/plugin_loader.go`
7. `go-go-gepa/pkg/jsbridge/emitter.go`

---

## Chronological Log

## 2026-02-26 17:52-17:56 - Ticket setup

### Commands

```bash
docmgr status --summary-only
find go-go-gepa/ttmp/2026/02/26 -maxdepth 1 -type d | sort
docmgr ticket create-ticket --ticket GEPA-06-JS-SEM-REDUCERS-HANDLERS --title "Investigate JavaScript-Registered SEM Reducers and Event Handlers" --topics gepa,event-streaming,js-vm,sem,pinocchio,geppetto,go-go-os
docmgr doc add --ticket GEPA-06-JS-SEM-REDUCERS-HANDLERS --doc-type design-doc --title "JavaScript-Registered SEM Reducers and Event Handler Architecture"
docmgr doc add --ticket GEPA-06-JS-SEM-REDUCERS-HANDLERS --doc-type reference --title "Investigation Diary"
```

### Notes

1. New ticket created successfully with standard doc structure.
2. Existing date folder already contained GEPA-04 and GEPA-05 tickets; picked GEPA-06 for this separate investigation.

---

## 2026-02-26 17:56-18:05 - Parallel deep code exploration (sub-agents)

### Explorer tasks launched

1. Pinocchio SEM projection ownership and extension model.
2. Geppetto event architecture + JS hook capabilities.
3. go-go-os SEM runtime registration and reducer path.

### Key outputs

1. **Geppetto explorer findings**
   - Geppetto canonical events are `start`/`partial`/`final`, not `llm.*` names.
   - JS run handle already supports `.on(eventType, callback)`.
   - JS collector supports wildcard `*` listeners.
2. **go-go-os explorer findings**
   - `registerSem` exists and `llm.delta` default handler is registered.
   - `registerSem` currently overwrites same-type handler (`Map.set` semantics).
   - Runtime app modules can register extensions; sandbox plugin runtime cannot.
3. **Pinocchio explorer findings**
   - Backend SEM projection path is `ApplySemFrame` in Go.
   - Custom timeline handlers are Go functions via `RegisterTimelineHandler`.
   - No backend JS plugin runtime for SEM reducers today.

---

## 2026-02-26 18:00 - Mid-investigation user update incorporated

User note:

> "btw we added streaming event in GEPA-04"

Action taken:

1. Explicitly validated in current code before concluding anything.
2. Added section in final design document to distinguish GEPA-04 capabilities from new requirements.

---

## 2026-02-26 18:06-18:14 - GEPA-04 validation in source code

### Commands

```bash
rg -n "emitEvent|events\.emit|stream-event|plugin_event|--stream|EventSink" go-go-gepa/cmd/gepa-runner go-go-gepa/pkg -S
find go-go-gepa/ttmp/2026/02/26/GEPA-04-ASYNC-PLUGIN-PROMISES--enable-promise-based-js-plugin-execution-and-streaming-events -maxdepth 2 -type f | sort
sed -n '1,240p' .../GEPA-04.../planning/01-promise-aware-plugin-bridge-and-streaming-events-implementation-plan.md
```

### Findings

1. `plugin_loader.go` injects `options.emitEvent` and `options.events.emit` across plugin methods.
2. `dataset/generator/plugin_loader.go` does same for `generateOne`.
3. `jsbridge/emitter.go` wraps payload with sequence/timestamp/plugin metadata.
4. `plugin_stream.go` prints `stream-event { ... }` when enabled.
5. Integration tests validate these stream lines (`stream_cli_integration_test.go`).

Interpretation:

- GEPA-04 provides source event emission and CLI stream visibility.
- It does not itself provide SEM reducer registration in pinocchio backend.

---

## 2026-02-26 18:14-18:18 - Focused line-anchored evidence extraction

### Commands

```bash
nl -ba geppetto/pkg/js/modules/geppetto/api_sessions.go | sed -n '450,620p'
nl -ba geppetto/pkg/js/modules/geppetto/api_events.go | sed -n '1,220p'
nl -ba geppetto/pkg/events/chat-events.go | sed -n '1,120p'
nl -ba pinocchio/pkg/webchat/timeline_projector.go | sed -n '70,190p'
nl -ba pinocchio/pkg/webchat/timeline_registry.go | sed -n '1,120p'
nl -ba /home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts | sed -n '1,220p'
nl -ba /home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts | sed -n '300,420p'
```

### Why this mattered

1. Captured exact line anchors for all major claims.
2. Verified naming mismatch (`partial` vs `llm.delta`) with source truth.
3. Verified front-end overwrite behavior and backend Go-only handler contracts.

---

## 2026-02-26 18:18-18:20 - Prototype experiment

### Script created

`scripts/js-sem-reducer-handler-prototype.js`

Purpose:

1. Simulate current single-handler map behavior (overwrite risk).
2. Simulate proposed composable reducer + handler chains with wildcard support.

### Command

```bash
node scripts/js-sem-reducer-handler-prototype.js
```

### Output summary

1. Current model output: `"custom:hello"` only (default handler replaced).
2. Proposed model output: multiple handlers run + reducer chain state evolves.

Interpretation:

- Confirms that composition semantics are a real requirement if additive JS behavior is expected.

---

## 2026-02-26 18:20-18:27 - Final synthesis writing

Work completed:

1. Wrote full design doc with:
   - direct yes/no answers,
   - layer ownership map,
   - capability matrix,
   - GEPA-04 baseline integration,
   - architecture options and staged recommendation,
   - API sketches and implementation phases,
   - risk and test strategy,
   - open questions.
2. Updated this diary with command chronology and interpretation.

---

## 2026-02-26 18:51-18:57 - Option C Task 1 implementation and commit

### Scope

Implemented backend Goja runtime bridge in `pinocchio` for JS SEM reducers + event handlers (Option C Task 1).

### Commands

```bash
go test ./pkg/webchat -run 'Timeline|JSTimeline'
git add pkg/webchat/timeline_registry.go pkg/webchat/timeline_js_runtime.go pkg/webchat/timeline_js_runtime_test.go
git commit -m "webchat: add js SEM reducer runtime bridge"
```

### Notes and debugging

1. First focused test run failed in `TestJSTimelineRuntime_HandlerErrorIsContained` because runtime executed reducers before handlers.
2. Runtime execution order was patched to run JS handlers first, then reducers (error-containment behavior preserved).
3. First commit attempt failed pre-commit lint:
   - `errcheck` required handling `r.vm.Set(...)` errors.
   - `gofmt` required formatting the new test file.
4. Applied lint fixes (`if err := r.vm.Set(...); err != nil { panic(err) }`) and ran `gofmt`.
5. Re-ran focused tests successfully.
6. Re-ran commit; pre-commit suite passed full test/lint and commit succeeded.

### Result

1. Commit: `pinocchio` `99c2bfd` (`webchat: add js SEM reducer runtime bridge`)
2. Files:
   - `pkg/webchat/timeline_js_runtime.go` (new)
   - `pkg/webchat/timeline_js_runtime_test.go` (new)
   - `pkg/webchat/timeline_registry.go` (runtime bridge hooks)

---

## 2026-02-26 18:58-19:00 - Option C Task 2 startup wiring + loader tests

### Work completed

1. Added `--timeline-js-script` (`TypeStringList`) to `cmd/web-chat/main.go`.
2. Added startup wiring via `configureTimelineJSScripts(...)` before app server start.
3. Added `cmd/web-chat/timeline_js_runtime_loader.go`:
   - path normalization (repeat flag + comma support),
   - fail-fast script loading (`LoadScriptFile`),
   - runtime registration via `webchat.SetTimelineRuntime`.
4. Added `cmd/web-chat/timeline_js_runtime_loader_test.go` for:
   - path normalization,
   - successful runtime load + projection behavior,
   - missing-script error path.

### Verification

```bash
go test ./cmd/web-chat -run 'TimelineJSScript|RuntimeConfig|AppOwned'
go test ./pkg/webchat -run 'Timeline|JSTimeline'
```

Result:

1. Focused tests passed.
2. Commit succeeded with full pre-commit checks.
3. Commit: `pinocchio` `f33fb55` (`web-chat: wire startup loading for timeline JS scripts`).

---

## 2026-02-26 19:00-19:03 - Option C Task 3 gpt-5-nano validation scripts + llm.delta checks

### Work completed

1. Updated loader test to react on `llm.delta` semantics (not generic event type).
2. Added `TestProfileResolver_GPT5NanoProfileIsResolvedForChatRequest` in `cmd/web-chat` tests.
3. Added ticket scripts in `scripts/`:
   - `exp-03-timeline-llm-delta-reducer.js`
   - `exp-03-profile-registry-gpt5nano.yaml`
   - `exp-03-validate-option-c-gpt5nano.sh`
4. Executed script and captured outputs:
   - `scripts/exp-03-summary.txt`
   - `scripts/exp-03-out/pinocchio-cmd-web-chat.txt`
   - `scripts/exp-03-out/pinocchio-pkg-webchat.txt`
   - `scripts/exp-03-out/go-go-gepa-streaming.txt`

### Verification commands (direct and via script)

```bash
go test ./cmd/web-chat -run 'TimelineJSScript|GPT5NanoProfile|ProfileResolver'
go test ./pkg/webchat -run 'TestJSTimelineRuntime_NonConsumingReducerAllowsBuiltinProjection|TestJSTimelineRuntime_ReducerCreatesEntityAndConsumesEvent'
go test ./cmd/gepa-runner -run 'TestDatasetGenerateStreamCLIOutput'
./scripts/exp-03-validate-option-c-gpt5nano.sh
```

Result:

1. All tests passed.
2. `gpt-5-nano` profile path validated in local/offline resolver and go-go-gepa streaming integration test.
3. Commit: `pinocchio` `4b1a649` (`web-chat: validate gpt-5-nano profile runtime wiring`).

---

## 2026-02-26 19:03-19:04 - Option C Task 4 docs/help + operational guardrails

### Work completed

1. Updated `cmd/web-chat/README.md` with:
   - `--timeline-js-script` usage,
   - reducer/handler API contract,
   - consume/fallback semantics,
   - startup/runtime safety notes,
   - run example.
2. Updated `pkg/doc/topics/webchat-debugging-and-ops.md` with:
   - JS runtime failure modes,
   - warning log signatures,
   - troubleshooting steps for accidental consume/overrides.

### Result

1. Commit: `pinocchio` `381ffb7` (`docs: document timeline JS runtime contract and troubleshooting`).

---

## 2026-02-26 19:06-19:52 - Follow-up: duplicate profile flag regression fix

### Issue found during validation

Running:

```bash
go run ./cmd/web-chat web-chat --help
```

returned:

- duplicate `profile-registries` flag registration error.

### Fix

1. Removed redundant explicit `profile-registries` flag from `cmd/web-chat/main.go`.
2. Kept profile registry sourcing from geppetto profile-settings section (single canonical flag provider).

### Re-check

```bash
go test ./cmd/web-chat -run 'TimelineJSScript|GPT5NanoProfile|ProfileResolver|AppOwned'
go run ./cmd/web-chat web-chat --help
```

Result:

1. Tests passed.
2. CLI help now loads correctly and includes `--timeline-js-script`.
3. Commit: `pinocchio` `4a87c5f` (`web-chat: avoid duplicate profile-registries flag`).

---

## 2026-02-26 20:03-20:04 - Added dedicated llm.delta projection harness

### Scope

Added a dedicated integration harness for `pinocchio/cmd/web-chat` to validate real `llm.delta` SEM projection behavior under JS runtime consume modes.

### Implemented tests

1. `TestLLMDeltaProjectionHarness_NonConsumingReducerAddsSideProjection`
2. `TestLLMDeltaProjectionHarness_ConsumingReducerSuppressesBuiltinDeltaProjection`
3. `TestLLMDeltaProjectionHarness_HandlerRunsBeforeReducer`

Harness characteristics:

1. Uses a deterministic streaming test engine that emits `start` + `partial` geppetto events.
2. Flows through real web-chat route stack (`POST /chat/default`, `/api/timeline` polling).
3. Loads JS runtime script through command loader path (`configureTimelineJSScripts`) from temp file.
4. Asserts timeline entity outcomes for consume/non-consume semantics and handler ordering.

### Commands

```bash
go test ./cmd/web-chat -run 'LLMDeltaProjectionHarness' -v
go test ./cmd/web-chat -run 'TimelineJSScript|LLMDeltaProjectionHarness|GPT5NanoProfile|ProfileResolver'
go test ./pkg/webchat -run 'JSTimeline|Timeline'
./scripts/exp-04-run-llm-delta-projection-harness.sh
```

### Artifacts

1. `scripts/exp-04-run-llm-delta-projection-harness.sh`
2. `scripts/exp-04-harness-output.txt`

### Result

1. Harness tests passed.
2. Reproducible ticket script added and executed successfully.

---

## 2026-02-27 09:35-10:20 - Runtime builder alignment in pinocchio + geppetto

### Scope

Implemented follow-up runtime alignment so both systems can share the owned go-go-goja runtime pattern and support co-locating geppetto bindings with custom (Pinocchio-style) timeline bindings in one VM.

### Code changes

1. `pinocchio/pkg/webchat/timeline_js_runtime.go`
   - moved runtime bootstrap to go-go-goja owned runtime (`engine` factory + owner runner),
   - switched script execution + reducer/handler dispatch through runner calls,
   - added explicit `Close(context.Context)` lifecycle.
2. `pinocchio/pkg/webchat/timeline_registry.go`
   - runtime clear/set now closes previous runtime if it supports close semantics.
3. `pinocchio/pkg/webchat/timeline_projector.go`
   - `llm.delta` now reconstructs cumulative text from prior cached content when cumulative is omitted and only `delta` is present.
4. `pinocchio/cmd/web-chat/timeline_js_runtime_loader.go`
   - runtime loader now passes `require.WithGlobalFolders(...)` for script directory + `node_modules`.
5. `geppetto/pkg/js/runtime/runtime.go` (new)
   - added reusable runtime bootstrap helper that returns an owned runtime exposing `require("geppetto")`.
6. `geppetto/pkg/js/runtime/runtime_test.go` (new)
   - added coexistence test: geppetto module + custom host binding (`registerSemReducer`) in same VM.
7. `geppetto/cmd/examples/geppetto-js-lab/main.go`
   - migrated to shared runtime helper + initializer hook path.
8. `geppetto/pkg/js/modules/geppetto/module_test.go`
   - migrated harness bootstrap to builder-owned runtime flow.
9. `geppetto/pkg/doc/topics/13-js-api-reference.md`
   - updated host wiring section to document shared runtime bootstrap usage.

### Validation commands and results

```bash
# pinocchio
go test ./pkg/webchat -run 'TestJSTimelineRuntime|TestTimelineProjector' -count=1
go test ./cmd/web-chat -run 'TestConfigureTimelineJSScripts|TestProfileResolver_GPT5NanoProfileIsResolvedForChatRequest|TestLLMDeltaProjectionHarness' -count=1

# geppetto
go test ./pkg/js/runtime -count=1
go test ./pkg/js/modules/geppetto -run 'TestTurnsCodecAndHelpers|TestSessionRunWithEchoEngine' -count=1
go test ./cmd/examples/geppetto-js-lab -count=1
```

Observed:

1. Pinocchio targeted tests passed.
2. Geppetto targeted tests passed.
3. `geppetto-js-lab` package builds (`[no test files]`).

### gpt-5-nano note

1. The resolver/profile-path test for `gpt-5-nano` was re-run and passed (`TestProfileResolver_GPT5NanoProfileIsResolvedForChatRequest`).
2. A live provider call with `gpt-5-nano` was **not** executed in this environment because `OPENAI_API_KEY` is currently not set.

---

## Usage Examples

## Re-check where projection is done

```bash
rg -n "ApplySemFrame|RegisterTimelineHandler|registerSem\(|on\(" pinocchio geppetto /home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat -S
```

## Re-run prototype

```bash
node scripts/js-sem-reducer-handler-prototype.js
```

## Re-open design doc

```bash
sed -n '1,260p' design-doc/01-javascript-registered-sem-reducers-and-event-handler-architecture.md
```

---

## Related

1. `design-doc/01-javascript-registered-sem-reducers-and-event-handler-architecture.md`
2. `scripts/js-sem-reducer-handler-prototype.js`
3. `../GEPA-04-ASYNC-PLUGIN-PROMISES--enable-promise-based-js-plugin-execution-and-streaming-events/planning/01-promise-aware-plugin-bridge-and-streaming-events-implementation-plan.md`
