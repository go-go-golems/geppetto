# Changelog

## 2026-06-02

- Initial workspace created


## 2026-06-02

Created intern-oriented design guide for Geppetto/xgoja turn-store persistence API and provider wiring.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/module.go — DefaultPersister and proposed storage option seam
- /home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/cmds/chat_persistence.go — Pinocchio turns-dsn implementation evidence


## 2026-06-02

Uploaded turn-store design guide bundle to reMarkable at /ai/2026/06/02/GP-JS-TURNSTORE-2026-06-02.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/02/GP-JS-TURNSTORE-2026-06-02--design-javascript-turn-store-persistence-api-and-provider-wiring/design-doc/01-javascript-turn-store-persistence-design-and-implementation-guide.md — Uploaded design guide source


## 2026-06-02

Implemented Geppetto JS turn-store wrappers, agent persistence selection, provider storage gating, tests, docs, and example (commit cf09f49e).

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_turn_store.go — Runtime wrapper API
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_turn_store_test.go — Regression tests
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/provider/provider.go — Provider config gating


## 2026-06-02

Wired Pinocchio js --turns-dsn/--turns-db into Geppetto JS gp.turnStores.default() via the existing SQLite turn store (Pinocchio commit 16e7f7b).

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/cmd/pinocchio/cmds/js.go — CLI flags and runtime registration
- /home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/cmd/pinocchio/cmds/js_turn_store.go — Host adapter implementation
- /home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/cmd/pinocchio/cmds/js_turn_store_test.go — Adapter/runtime regression coverage


## 2026-06-02

Validated Pinocchio JS two-process --turns-db storage/resume smoke with live provider; second process resumed one stored turn and persisted a second (session pinocchio-js-storage-1780440401).

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/cmd/pinocchio/cmds/js.go — CLI storage path exercised by smoke
- /home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/cmd/pinocchio/cmds/js_turn_store.go — Adapter persistence/resume path exercised by smoke

