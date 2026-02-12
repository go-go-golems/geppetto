# Changelog

## 2026-02-12

- Initial workspace created


## 2026-02-12

Created implementation plan and detailed builder/tools task list; added and executed JS builder+tools smoke script using PINOCCHIO_PROFILE=gemini-2.5-flash-lite.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-007-JS-BUILDER-TOOLS--js-api-builder-and-tools-orchestration/design/01-implementation-plan.md — Builder and toolloop plan
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-007-JS-BUILDER-TOOLS--js-api-builder-and-tools-orchestration/reference/01-diary.md — Recorded execution results
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-007-JS-BUILDER-TOOLS--js-api-builder-and-tools-orchestration/scripts/test_builder_tools_smoke.js — Runnable JS builder/tools smoke script
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-007-JS-BUILDER-TOOLS--js-api-builder-and-tools-orchestration/tasks.md — Detailed builder/tools tasks


## 2026-02-12

Implemented JS builder+tools orchestration (createBuilder/withTools/toolLoop mapping), JS tool registry, Go-tool import/call, and toolloop integration tests.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-007-JS-BUILDER-TOOLS--js-api-builder-and-tools-orchestration/reference/01-diary.md — Step 3 implementation diary
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/api.go — Builder and tools registry implementation
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/codec.go — Native JS value conversion for callback mutation semantics
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/module_test.go — Builder/tools and Go tool invocation tests

