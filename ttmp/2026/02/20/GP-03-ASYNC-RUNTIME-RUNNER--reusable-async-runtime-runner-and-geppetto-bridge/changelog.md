# Changelog

## 2026-02-20

- Initial workspace created


## 2026-02-20

Step 1 complete: added go-go-goja runtimeowner runner package + unit/race tests (commit 03a723b).

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/go-go-goja/pkg/runtimeowner/runner.go — Core owner-thread call/post implementation
- /home/manuel/workspaces/2026-02-12/geppetto-js/go-go-goja/pkg/runtimeowner/runner_race_test.go — Concurrent race stress coverage
- /home/manuel/workspaces/2026-02-12/geppetto-js/go-go-goja/pkg/runtimeowner/runner_test.go — Behavioral coverage for cancellation/panic/closed paths


## 2026-02-20

Step 2 complete: integrated geppetto runtime bridge and migrated async JS callback boundaries to owner-thread runner calls; added async runAsync/start JS callback regressions (commit aad992c).

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/api.go — Owner-thread migration for engine/middleware/tool hooks/tool handler/event collector/start/runAsync
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/module.go — Options refactor from Loop to Runner and bridge wiring
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/module_test.go — Async regression tests and promise polling helpers
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/runtimebridge/bridge.go — Bridge helpers for Call/Post/InvokeCallable/ToJSValue

