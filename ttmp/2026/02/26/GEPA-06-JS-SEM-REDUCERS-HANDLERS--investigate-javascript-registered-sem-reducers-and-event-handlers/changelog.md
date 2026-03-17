# Changelog

## 2026-03-16

- Moved the canonical ticket workspace from `pinocchio/ttmp` to `geppetto/ttmp`.
- Reason: the ticket is primarily about reusable Geppetto-side runtime and binding architecture, with Pinocchio and go-go-os as downstream consumers.

## 2026-02-27

- Migrated `pinocchio` timeline JS runtime to go-go-goja owned runtime pattern (`engine` + runtime owner runner execution path) and added runtime close lifecycle on registry set/clear.
- Updated `pinocchio` timeline projection to handle delta-only `llm.delta` frames by reconstructing cumulative content from prior state.
- Added `geppetto/pkg/js/runtime` helper for builder-owned JS runtime bootstrap with `require("geppetto")` wiring and initializer hooks.
- Migrated `geppetto` JS lab example + module tests to the builder-owned runtime flow.
- Added cross-binding runtime test proving geppetto bindings and custom host bindings can coexist in one VM.
- Re-ran targeted tests in both repos (`pinocchio` webchat + harness tests, `geppetto` JS runtime/module tests) successfully.

## 2026-02-26

- Implemented Option C Task 2 in `pinocchio`: startup wiring for `--timeline-js-script` plus loader tests (`f33fb55`).
- Implemented Option C Task 3 in `pinocchio`: `gpt-5-nano` resolver/runtime validation test (`4b1a649`).
- Added GEPA-06 ticket-local validation scripts and output captures under `scripts/exp-03-*`.
- Added dedicated llm.delta projection harness script/output (`scripts/exp-04-*`) and corresponding `pinocchio` integration harness tests.
- Implemented Option C Task 4 in `pinocchio`: JS runtime contract + troubleshooting docs (`381ffb7`).
- Follow-up fix in `pinocchio`: removed duplicate `profile-registries` flag registration so `web-chat web-chat --help` and runtime startup path work (`4a87c5f`).
- Implemented Option C Task 1 in `pinocchio`: added Goja-based JS SEM runtime bridge and committed as `99c2bfd`.
- Created GEPA-06 ticket for JS-registered SEM reducers and JS event-handler architecture investigation.
- Completed deep cross-repo analysis across `geppetto`, `pinocchio`, `go-go-os`, and `go-go-gepa`.
- Incorporated and validated GEPA-04 streaming-event baseline in code.
- Added prototype script `scripts/js-sem-reducer-handler-prototype.js` and captured behavior.
- Published comprehensive design doc and chronological diary with implementation roadmap.

## 2026-02-26

Completed layered investigation of JS SEM reducers and event handlers: confirmed geppetto JS event subscriptions, pinocchio Go-owned backend projection, go-go-os frontend runtime registration capabilities, and GEPA-04 streaming-event baseline. Added architecture roadmap and prototype for composable reducer/handler semantics.

### Related Files

- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/go-go-gepa/ttmp/2026/02/26/GEPA-06-JS-SEM-REDUCERS-HANDLERS--investigate-javascript-registered-sem-reducers-and-event-handlers/design-doc/01-javascript-registered-sem-reducers-and-event-handler-architecture.md — Primary architecture analysis
- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/go-go-gepa/ttmp/2026/02/26/GEPA-06-JS-SEM-REDUCERS-HANDLERS--investigate-javascript-registered-sem-reducers-and-event-handlers/reference/01-investigation-diary.md — Chronological command and evidence diary
- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/go-go-gepa/ttmp/2026/02/26/GEPA-06-JS-SEM-REDUCERS-HANDLERS--investigate-javascript-registered-sem-reducers-and-event-handlers/scripts/js-sem-reducer-handler-prototype.js — Prototype demonstrating current overwrite behavior and composable alternative
