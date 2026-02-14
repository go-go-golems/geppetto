# Tasks

## Phase 0: Ticket Execution Setup

- [x] Create detailed design doc for profile decoupling and resolver-plan architecture.
- [x] Add explicit clean-cutover policy: retire `BuildEngineFromReq` / `WithEngineFromReqBuilder` after migration.
- [x] Create implementation diary document and keep it updated per implementation slice.
- [ ] Keep `tasks.md` checkboxes synchronized with implementation and commits.

## Phase 1: Core API Cutover (`pinocchio/pkg/webchat`)

- [ ] Introduce new resolver-plan interface in core (request -> conversation plan), centered on:
  - `ConvID`, `Prompt`, `IdempotencyKey`
  - runtime identity (`RuntimeKey`) + rebuild key (`Fingerprint`)
  - runtime constructor closure (`BuildRuntime(...)`)
- [x] Rework chat and WS handlers in core router to use resolver-plan API directly.
- [x] Remove or short-circuit old `BuildEngineFromReq` pathway from runtime execution path.
- [ ] Remove profile concepts from core types/options surface:
  - `Profile`, `ProfileRegistry`, `WithProfileRegistry`, `Router.AddProfile`
- [ ] Remove core-owned profile endpoints from router:
  - `/api/chat/profiles`
  - `/api/chat/profile`
- [ ] Update conversation + queue structs to generic runtime identity naming (remove `ProfileSlug` fields).
- [ ] Move rebuild checks to `Fingerprint` comparison (instead of profile+signature coupling).
- [ ] Update debug API payloads to generic runtime naming (`engine_key` / `runtime_key`).
- [ ] Update WS hello semantics to remove/rename profile field usage.
- [ ] Remove dead code and tests tied only to old profile-centric request builder path.

## Phase 2: `pinocchio/cmd/web-chat` Migration (app-owned profile policy)

- [ ] Add app-local profile registry/types in `cmd/web-chat` (no dependency on core profile registry).
- [ ] Implement app-local resolver that:
  - resolves profile selection policy
  - builds effective runtime config
  - computes fingerprint
  - returns conversation request plan
- [ ] Add app-owned profile API handlers in `cmd/web-chat` and mount with `r.HandleFunc(...)`.
- [ ] Wire new resolver into router setup in `cmd/web-chat/main.go`.
- [ ] Update debug UI frontend mapping for renamed runtime identity fields.
- [ ] Update any WS hello frontend/proto consumers affected by payload rename.

## Phase 3: `web-agent-example` Migration (profile-free)

- [x] Replace custom `engine_from_req.go` builder with resolver-plan implementation.
- [ ] Remove `r.AddProfile(...)` from `web-agent-example/cmd/web-agent-example/main.go`.
- [ ] Keep middleware/disco/thinking composition in app runtime factory closure.
- [ ] Ensure web-agent-example frontend runs with profile selector disabled/optional.
- [ ] Verify no dependency on `/api/chat/profile*` endpoints remains.

## Phase 4: Tests and Validation

- [ ] Update core webchat unit/integration tests for resolver-plan flow and profile-free core behavior.
- [ ] Add/adjust tests for app-owned profile endpoints in `cmd/web-chat`.
- [ ] Add/adjust tests for web-agent-example resolver and runtime behavior.
- [ ] Run and record focused test commands:
  - `go test ./pinocchio/pkg/webchat/...`
  - `go test ./pinocchio/cmd/web-chat/...`
  - `go test ./web-agent-example/...`
- [ ] Run frontend typecheck/build checks for impacted web UIs.

## Phase 5: Documentation and Cleanup

- [ ] Update `pinocchio/pkg/doc/topics/webchat-framework-guide.md` for resolver-plan API.
- [ ] Update `pinocchio/pkg/doc/topics/webchat-user-guide.md` for app-owned profile policy.
- [ ] Update `pinocchio/pkg/doc/tutorials/03-thirdparty-webchat-playbook.md` to remove legacy builder guidance.
- [ ] Ensure docs/examples no longer teach `BuildEngineFromReq` after cutover.
- [ ] Relate changed files in docmgr metadata where needed.

## Implementation Loop (required per slice)

- [ ] For each completed slice: run tests, commit code, check off tasks, update diary, update changelog.
