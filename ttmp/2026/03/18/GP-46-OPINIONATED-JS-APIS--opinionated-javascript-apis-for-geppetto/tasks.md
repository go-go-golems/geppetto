# Tasks

## Research and Documentation

- [x] Create GP-46 ticket workspace and scaffolding
- [x] Inventory current JS module exports, docs, tests, and example scripts
- [x] Write the intern-oriented analysis / design / implementation guide
- [x] Write the Manuel investigation diary entry for the research pass
- [x] Run `docmgr doc relate` for the key JS API evidence files
- [x] Run `docmgr doctor --ticket GP-46-OPINIONATED-JS-APIS --stale-after 30`
- [x] Upload the ticket bundle to reMarkable and verify the remote listing

## Phase 1: Ticket Baseline and Tasking

- [ ] Rewrite the task board into concrete build slices that reflect the post-GP-47 substrate
- [ ] Update the GP-46 changelog to record the implementation kickoff
- [ ] Commit the GP-46 ticket baseline before code work starts

## Phase 2: Runner Namespace and Runtime Resolution

- [ ] Add a new `gp.runner` namespace to the JS module exports in `module.go`
- [ ] Add internal runner ref/types in the JS module for resolved runtime and prepared runs
- [ ] Implement `gp.runner.resolveRuntime(...)` with support for:
  - profile-driven runtime resolution
  - direct `systemPrompt` override/addition
  - direct `middlewares` addition
  - direct `toolNames` override/addition
  - direct runtime identity metadata (`runtimeKey`, `runtimeFingerprint`, `profileVersion`)
- [ ] Reuse the GP-47 runtime-metadata helper layer instead of duplicating translation logic
- [ ] Add focused tests for `gp.runner.resolveRuntime(...)`
- [ ] Commit the namespace + resolveRuntime slice

## Phase 3: Prepared Runs and Blocking Execution

- [ ] Implement `gp.runner.prepare(...)` to:
  - require an explicit engine
  - consume `runtime` from `gp.runner.resolveRuntime(...)`
  - build a session via the existing builder/session substrate
  - append or clone the seed turn
  - expose the prepared session and turn
- [ ] Add a JS-facing prepared-run handle with:
  - `session`
  - `turn`
  - `runtime`
  - `run()`
  - `start()`
- [ ] Implement `gp.runner.run(...)` as the blocking wrapper around `prepare(...)`
- [ ] Add focused tests for `prepare(...)` and `run(...)`
- [ ] Commit the prepared-run slice

## Phase 4: Streaming and Async Start

- [ ] Implement `gp.runner.start(...)` as the event-driven path
- [ ] Decide and implement the runner-specific handle shape:
  - reuse the existing run-handle contract where possible
  - ensure cancellation and event subscriptions stay first-class
- [ ] Add tests for streaming/event delivery and async completion
- [ ] Commit the streaming slice

## Phase 5: Type Surface, Examples, and Docs

- [ ] Extend `geppetto.d.ts.tmpl` and generated `geppetto.d.ts` with `gp.runner` contracts
- [ ] Add example scripts that demonstrate:
  - simple `gp.runner.run(...)`
  - profile-driven runtime assembly
  - streaming `gp.runner.start(...)`
- [ ] Update the JS API reference to make `gp.runner` the default path
- [ ] Update the JS user guide to demote `createBuilder` / `createSession` to advanced usage
- [ ] Run focused validation for JS module tests, docs, and examples
- [ ] Commit the public-surface/docs slice

## Phase 6: Ticket Closeout

- [ ] Update the GP-46 diary with implementation steps, exact commands, failures, and review notes
- [ ] Update the GP-46 changelog with landed commits and rationale
- [ ] Run `docmgr doc relate` for new implementation files and examples
- [ ] Run `docmgr doctor --ticket GP-46-OPINIONATED-JS-APIS --stale-after 30`
- [ ] Upload the refreshed GP-46 bundle to reMarkable
- [ ] Mark the ticket complete if all slices land cleanly
