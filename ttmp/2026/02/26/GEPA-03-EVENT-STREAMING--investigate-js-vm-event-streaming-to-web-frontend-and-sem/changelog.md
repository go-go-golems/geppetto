# Changelog

## 2026-03-16

- Moved the canonical ticket workspace from `pinocchio/ttmp` to `geppetto/ttmp`.
- Reason: the implementation center of gravity is the Geppetto JS VM host bridge, not the Pinocchio consumer side.

## 2026-02-26

- Initial workspace created.
- Added design doc `design-doc/01-gepa-event-streaming-architecture-investigation.md` with detailed architecture and implementation plan.
- Added investigation diary `reference/01-investigation-diary.md` with full command chronology and findings.
- Added runnable prototype script `scripts/sem-envelope-prototype.js` and captured validation output.
- Recorded core conclusion: engine events already stream through SEM; GEPA script-level event emission contract/bridge still needs implementation.

## 2026-02-26

Completed deep cross-repo investigation of JS VM event streaming path; documented current capability (engine events yes, script events not yet), produced phased implementation architecture, and added SEM envelope prototype experiment.

### Related Files

- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/go-go-gepa/ttmp/2026/02/26/GEPA-03-EVENT-STREAMING--investigate-js-vm-event-streaming-to-web-frontend-and-sem/design-doc/01-gepa-event-streaming-architecture-investigation.md — Primary architecture and implementation guidance
- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/go-go-gepa/ttmp/2026/02/26/GEPA-03-EVENT-STREAMING--investigate-js-vm-event-streaming-to-web-frontend-and-sem/reference/01-investigation-diary.md — Chronological evidence log
- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/go-go-gepa/ttmp/2026/02/26/GEPA-03-EVENT-STREAMING--investigate-js-vm-event-streaming-to-web-frontend-and-sem/scripts/sem-envelope-prototype.js — Prototype envelope validation script

## 2026-02-27

- Updated documentation with implementation follow-through for module-based timeline JS API in Pinocchio.
- Recorded successful real test runs for runtime and web-chat harness projections.
- Documented new canonical usage:
  - `require("pinocchio")`
  - `require("pnocchio")` alias
  - `p.timeline.registerSemReducer(...)`
  - `p.timeline.onSem(...)`

### Related Files

- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/pinocchio/pkg/webchat/timeline_js_runtime.go — Native module registration for timeline reducer/handler APIs
- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/pinocchio/pkg/webchat/timeline_js_runtime_test.go — Module API + alias coverage in runtime tests
- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/pinocchio/cmd/web-chat/llm_delta_projection_harness_test.go — Real projection harness updated to module API
- /home/manuel/workspaces/2026-02-22/add-gepa-optimizer/go-go-gepa/ttmp/2026/02/26/GEPA-03-EVENT-STREAMING--investigate-js-vm-event-streaming-to-web-frontend-and-sem/reference/01-investigation-diary.md — New diary step with commands/results
