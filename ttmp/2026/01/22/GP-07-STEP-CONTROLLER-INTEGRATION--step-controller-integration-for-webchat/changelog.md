# Changelog

## 2026-01-22

- Initial workspace created
- Added StepController deep-dive analysis and research diary; updated index and tasks to reflect “integrate stepping into session” direction (no API backwards-compat).

## 2026-01-22

Added analysis proposing ToolCallingLoop struct + With* options and moving step control into the canonical tool loop (not middleware / not conversation-owned). Updated diary with verbatim prompt + intent per new diary rules.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-07-STEP-CONTROLLER-INTEGRATION--step-controller-integration-for-webchat/analysis/02-toolcallingloop-struct-step-control-integration.md — New design analysis
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-07-STEP-CONTROLLER-INTEGRATION--step-controller-integration-for-webchat/reference/01-diary.md — Diary Step 5 with Prompt Context


## 2026-01-22

Uploaded new analysis PDF to reMarkable under /ai/2026/01/23/GP-07-STEP-CONTROLLER-INTEGRATION (via remarquee upload md).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-07-STEP-CONTROLLER-INTEGRATION--step-controller-integration-for-webchat/reference/01-diary.md — Diary Step 6 records upload commands and destination


## 2026-01-22

Added design+implementation plan for moving tool loop + step control into a new toolloop package (WithEngine/WithRegistry options), making tool loop publish Geppetto-native debugger.pause events, and wiring Pinocchio continue to a shared StepController. Updated GP-07 tasks to match.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-07-STEP-CONTROLLER-INTEGRATION--step-controller-integration-for-webchat/design/01-move-tool-loop-step-control-into-toolloop-package-implementation-plan.md — New design doc + step-by-step plan
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-07-STEP-CONTROLLER-INTEGRATION--step-controller-integration-for-webchat/tasks.md — Revised tasks to reflect toolloop package approach


## 2026-01-22

Uploaded GP-07 design/implementation-plan PDF to reMarkable under /ai/2026/01/23/GP-07-STEP-CONTROLLER-INTEGRATION.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-07-STEP-CONTROLLER-INTEGRATION--step-controller-integration-for-webchat/design/01-move-tool-loop-step-control-into-toolloop-package-implementation-plan.md — Uploaded as PDF via remarquee

