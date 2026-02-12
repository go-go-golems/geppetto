# Changelog

## 2026-02-12

- Initial workspace created


## 2026-02-12

Created implementation plan and detailed inference task list; added and executed JS inference smoke script using PINOCCHIO_PROFILE=gemini-2.5-flash-lite.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-005-JS-INFERENCE--js-api-inference-execution/design/01-implementation-plan.md — Inference lifecycle and profile constraints
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-005-JS-INFERENCE--js-api-inference-execution/reference/01-diary.md — Recorded execution results
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-005-JS-INFERENCE--js-api-inference-execution/scripts/test_inference_smoke.js — Runnable JS inference smoke script
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-005-JS-INFERENCE--js-api-inference-execution/tasks.md — Detailed inference tasks


## 2026-02-12

Implemented JS inference lifecycle APIs (createSession/run/runAsync/cancel/isRunning) and validated inference smoke tests with required profile.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-005-JS-INFERENCE--js-api-inference-execution/reference/01-diary.md — Step 3 implementation diary
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/api.go — Session and inference wrapper methods
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/module_test.go — Inference lifecycle tests

