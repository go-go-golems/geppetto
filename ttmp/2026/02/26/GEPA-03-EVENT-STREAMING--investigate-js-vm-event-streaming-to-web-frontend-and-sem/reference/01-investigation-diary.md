---
Title: Investigation Diary
Ticket: GEPA-03-EVENT-STREAMING
Status: active
Topics:
    - gepa
    - event-streaming
    - js-vm
    - web-frontend
    - sem
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/ws/wsManager.ts
      Note: |-
        Frontend ingest/hydration checkpoints
        Diary records frontend ingest findings
    - Path: go-go-gepa/cmd/gepa-runner/js_runtime.go
      Note: VM construction and module registration checkpoints
    - Path: go-go-gepa/ttmp/2026/02/26/GEPA-03-EVENT-STREAMING--investigate-js-vm-event-streaming-to-web-frontend-and-sem/design-doc/01-gepa-event-streaming-architecture-investigation.md
      Note: Diary references final architecture conclusions
    - Path: go-go-gepa/ttmp/2026/02/26/GEPA-03-EVENT-STREAMING--investigate-js-vm-event-streaming-to-web-frontend-and-sem/scripts/sem-envelope-prototype.js
      Note: Diary reproduces prototype command and output
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: |-
        SEM pass-through/translation behavior verified via explorer
        Diary records stream coordinator behavior findings
    - Path: scripts/sem-envelope-prototype.js
      Note: Ticket-local prototype for envelope validation
ExternalSources: []
Summary: Chronological log of commands, evidence collection, synthesis, and prototype checks for GEPA script-event streaming investigation.
LastUpdated: 2026-02-27T09:55:00-05:00
WhatFor: Preserve step-by-step investigative evidence and reasoning for future continuation
WhenToUse: Use to audit exactly how conclusions were reached and reproduce quick checks
---





# Investigation Diary

## Goal

Determine whether GEPA JS VM scripts can already stream events to frontend through Pinocchio SEM, and if not, define exactly what is missing and how to implement it safely.

## Context

Requested target:

1. Use new docmgr ticket `GEPA-03-EVENT-STREAMING`.
2. Analyze `pinocchio/` and `/home/manuel/code/wesen/corporate-headquarters/go-go-os`.
3. Focus on script path:
   `go-go-gepa/ttmp/2026/02/26/GEPA-02-ANALYZE-RUNNER--analyze-js-runner-and-design-gepa-optimization-tooling/scripts/exp-11-coaching-dataset-generator.js`.
4. Produce deep textbook-style design report + diary + reMarkable upload.

## Quick Reference

## Capability verdict

1. Engine/geppetto events to frontend via SEM: **already working**.
2. GEPA script-level structured event emission into that pipeline: **missing**.
3. Feasibility of adding it: **high**, mostly source-side emission and registration work.

## Most important evidence anchors

1. Pinocchio sink/translation path:
   - `pinocchio/pkg/webchat/conversation_service.go:373`
   - `pinocchio/pkg/webchat/stream_coordinator.go:152`
   - `pinocchio/pkg/webchat/sem_translator.go:133`
2. go-go-os frontend ingest path:
   - `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/ws/wsManager.ts:122`
   - `/home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat/sem/semRegistry.ts:66`
3. GEPA VM/event source gap:
   - `go-go-gepa/cmd/gepa-runner/js_runtime.go:31`
   - `go-go-gepa/cmd/gepa-runner/console.go:17`
   - `go-go-gepa/pkg/optimizer/gepa/optimizer.go:574`

---

## Chronological Log

## 2026-02-27 09:10 to 09:55 - Pinocchio module API migration (`require("pinocchio")` + `require("pnocchio")`)

### Prompt Context

**User prompt (verbatim):** "can we do a require("pnocchio") as well? to be unified in the design?" and "go ahead, no backwards compatiblility needed"

**Assistant interpretation:** Replace timeline JS globals with a module-first API and add alias support so script code can use a consistent `require(...)` contract.

**Inferred user intent:** Ensure timeline SEM reducer/handler registration follows the same go-go-goja module pattern and is reusable in unified engine composition.

**Commit (code):** `b50ec75` — "timeline-js: require pinocchio/pnocchio module for SEM reducers"

### What I did

1. Migrated Pinocchio timeline runtime host bindings from global VM mutation (`vm.Set("registerSemReducer")`, `vm.Set("onSem")`) to native module registration through go-go-goja builder `WithModules(...)`.
2. Registered two module names to the same loader:
   - `require("pinocchio")`
   - `require("pnocchio")` (alias)
3. Exported timeline API under:
   - `p.timeline.registerSemReducer(eventType, fn)`
   - `p.timeline.onSem(eventType, fn)`
4. Updated all runtime and harness tests to use the module API (including alias test coverage via `require("pnocchio")`).
5. Updated web-chat runtime docs/troubleshooting docs to reflect module-based usage.
6. Ran real tests:
   - `go test ./pkg/webchat -count=1`
   - `go test ./cmd/web-chat -count=1 -run 'TestConfigureTimelineJSScripts|TestLLMDeltaProjectionHarness'`
   - all passed.

### Why

The migration removes reliance on global host bindings and aligns timeline scripting with module registration patterns already used in go-go-goja/geppetto runtimes, making mixed-module runtime composition predictable.

### What worked

1. Alias module registration (`pnocchio`) worked without additional runtime shims.
2. Existing harness tests validated real `llm.delta` projection behavior with module API scripts.
3. Runtime remained fail-fast on script-load errors and non-fatal on handler/reducer runtime exceptions.

### What didn't work

1. First `git commit` attempt ran pre-commit hooks and did not return a clear final status in the initial interactive session; re-checking log showed commit actually completed (`b50ec75`).

### What I learned

1. Native module registration in go-go-goja builder is sufficient for timeline JS host APIs; global injection is no longer required.
2. Keeping `timeline` namespace under module exports avoids future API crowding while preserving script readability.

### What was tricky to build

The main edge was ensuring module availability before script execution. Loading scripts without forcing module registration first can cause script-time `require` failures depending on initialization order. The fix was to explicitly require the module on the runtime owner path before `RunScript(...)`.

### What warrants a second pair of eyes

1. Whether to keep top-level module shortcuts (`exports.registerSemReducer`, `exports.onSem`) or remove them for strict API minimalism.
2. Whether geppetto runtime should expose a shared `ModuleSpec` composition entrypoint so pinocchio/geppetto modules can be registered together with zero custom glue.

### What should be done in the future

1. Apply the same module-spec composition pattern in geppetto runtime options so host apps can plug both bindings in one engine instance.
2. Add one integration test that composes both module specs in one runtime and validates registration/order behavior.

### Code review instructions

1. Start at `pinocchio/pkg/webchat/timeline_js_runtime.go` (module registration, loader, require-ensure path).
2. Review updated test scripts in:
   - `pinocchio/pkg/webchat/timeline_js_runtime_test.go`
   - `pinocchio/cmd/web-chat/timeline_js_runtime_loader_test.go`
   - `pinocchio/cmd/web-chat/llm_delta_projection_harness_test.go`
3. Validate with:
   - `go test ./pkg/webchat -count=1`
   - `go test ./cmd/web-chat -count=1 -run 'TestConfigureTimelineJSScripts|TestLLMDeltaProjectionHarness'`

### Technical details

Canonical JS usage:

```js
const p = require("pinocchio"); // or require("pnocchio")
p.timeline.onSem("*", (ev, ctx) => { /* observe */ });
p.timeline.registerSemReducer("llm.delta", (ev, ctx) => ({
  consume: false,
  upserts: [{ id: ev.id + "-side", kind: "llm.side", props: { cumulative: ev.data?.cumulative } }],
}));
```

## 2026-02-26 17:18 to 17:26 - Ticket setup and workflow bootstrap

### Commands

```bash
docmgr status --summary-only
docmgr ticket create-ticket --ticket GEPA-03-EVENT-STREAMING --title "Investigate JS VM Event Streaming to Web Frontend and SEM" --topics gepa,event-streaming,js-vm,web-frontend,sem
docmgr doc add --ticket GEPA-03-EVENT-STREAMING --doc-type design-doc --title "GEPA Event Streaming Architecture Investigation"
docmgr doc add --ticket GEPA-03-EVENT-STREAMING --doc-type reference --title "Investigation Diary"
```

### Observations

1. Ticket workspace created successfully under:
   `go-go-gepa/ttmp/2026/02/26/GEPA-03-EVENT-STREAMING--investigate-js-vm-event-streaming-to-web-frontend-and-sem`.
2. Required docs scaffolded (`index.md`, `tasks.md`, `changelog.md`, design doc, diary).

### Notes

1. Loaded skill references:
   - `ticket-research-docmgr-remarkable/SKILL.md`
   - `references/writing-style.md`
   - `references/deliverable-checklist.md`

---

## 2026-02-26 17:26 to 17:31 - Initial broad scan

### Commands

```bash
rg --files pinocchio | head -n 200
rg -n "\b(goja|engine|event|stream|websocket|sse|sem|runner|vm)\b" pinocchio -S
rg --files /home/manuel/code/wesen/corporate-headquarters/go-go-os | head -n 200
rg -n "\b(goja|engine|event|stream|websocket|sse|sem|runner|vm)\b" /home/manuel/code/wesen/corporate-headquarters/go-go-os -S
```

### What worked

1. Confirmed there is substantial existing SEM/websocket architecture in both repos.
2. Located high-signal areas quickly (`pkg/webchat/*`, `wsManager.ts`, `semRegistry.ts`, proto-generated types).

### What was tricky

1. Broad pattern search produced very large noisy output.
2. Decided to pivot to focused explorer agents for line-anchored authoritative extraction.

---

## 2026-02-26 17:31 to 17:36 - Focused explorer investigations

### Sub-agents launched

1. `pinocchio` event pipeline and SEM translation explorer.
2. `go-go-os` websocket + SEM frontend consumption explorer.
3. `go-go-gepa` JS VM/script event capability explorer.

### High-confidence findings from explorer outputs

1. **Pinocchio**
   - Engine events are already attached to sink and routed through stream coordinator.
   - Coordinator supports both direct SEM envelopes and geppetto-event translation.
   - Unknown families require registration for projection.
2. **go-go-os**
   - WS manager enforces SEM envelope shape and drives sem registry.
   - Event bus and Event Viewer already tap raw envelopes for debug.
   - Custom families are feasible but need explicit handlers/renderers.
3. **go-go-gepa**
   - goja runtime exists with `require(...)` module surface.
   - No first-class script structured event emission API currently.
   - Optimizer emits structured Go events, but default exposure is textual/log-style output.

### Interpretation

The consumer half (Pinocchio + go-go-os) is mostly production-ready for custom events. Missing work is mostly producer-side emission contract and bridge from GEPA runner to stream transport.

---

## 2026-02-26 17:36 to 17:39 - Prototype experiment

### Command

```bash
node --version
```

### Output

`v22.21.0`

### Why this mattered

Confirmed local Node runtime available to execute quick ticket-local prototype checks.

### Prototype script created

`scripts/sem-envelope-prototype.js`

Purpose:

1. Show canonical envelope requirements (`sem: true`, `event.id`, `event.type`).
2. Show family classification behavior.
3. Demonstrate `timeline.upsert` can be emitted from script intent.

### Run command

```bash
node scripts/sem-envelope-prototype.js
```

### Output

```text
OK | type=gepa.script.progress | family=other | reason=valid
OK | type=timeline.upsert | family=timeline | reason=valid
FAIL | type=gepa.bad | family=other | reason=missing sem=true
FAIL | type=<missing> | family=other | reason=missing event.type
```

### Interpretation

1. Envelope-level correctness checks are simple and should live in host bridge.
2. `timeline.upsert` remains lowest-friction path for immediate frontend projection.
3. Custom families default to non-special classification unless explicitly mapped.

---

## 2026-02-26 17:39 to 17:47 - Synthesis and design doc drafting

### Work performed

1. Wrote full design doc with:
   - architecture map,
   - capability matrix,
   - option analysis,
   - recommended hybrid model,
   - pseudocode/API contracts,
   - phased implementation plan,
   - risks/tests/open questions.
2. Embedded local evidence references and external documentation links.
3. Added diagrams (flow + sequence) for producer->transport->frontend path clarity.

### Reasoning checkpoint

The right first milestone is to add host-normalized SEM emission from GEPA with optional `timeline.upsert`, because this minimizes dependency on new translators and gives immediate visible frontend results.

---

## Usage Examples

## Re-run this investigation quickly

1. Refresh code evidence:

```bash
# Pinocchio
rg -n "SemanticEventsFromEventWithCursor|StreamCoordinator|TimelineProjector" pinocchio/pkg/webchat -S

# go-go-os
rg -n "handleSem|timeline.upsert|eventBus|WsManager" /home/manuel/code/wesen/corporate-headquarters/go-go-os/packages/engine/src/chat -S

# go-go-gepa
rg -n "newJSRuntime|console\.error|OptimizerEvent|SetEventHook" go-go-gepa -S
```

2. Re-run prototype contract check:

```bash
node scripts/sem-envelope-prototype.js
```

3. Re-open primary design:

```bash
sed -n '1,260p' design-doc/01-gepa-event-streaming-architecture-investigation.md
```

---

## Related

1. `design-doc/01-gepa-event-streaming-architecture-investigation.md`
2. `scripts/sem-envelope-prototype.js`
