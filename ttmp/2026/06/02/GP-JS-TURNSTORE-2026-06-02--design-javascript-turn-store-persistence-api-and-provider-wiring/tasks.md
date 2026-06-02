# Tasks

## Done

- [x] Create ticket workspace and design document.
- [x] Investigate Geppetto JS module persistence seams.
- [x] Investigate Pinocchio `--turns-dsn` / turn-store implementation.
- [x] Write intern-oriented turn-store API and provider wiring guide.
- [x] Relate key Geppetto and Pinocchio source files.
- [x] Add Geppetto `gp.turnStores` wrappers and host-facing storage interfaces.
- [x] Add `agent.persistTo(...)` and `agent.persistDefault(...)` builder methods.
- [x] Extend xgoja provider config with gated turn storage settings.
- [x] Add JS module, provider, DTS parity, and docs tests for turn-store wrappers.
- [x] Add initial host-storage JS example and docs.

## Follow-up implementation tasks

- [ ] Implement Pinocchio host adapter for DSN-backed turn stores.
- [ ] Add Pinocchio/xgoja integration tests against a temporary SQLite turn store.
- [ ] Run a real host-backed storage smoke once the Pinocchio adapter exists.
