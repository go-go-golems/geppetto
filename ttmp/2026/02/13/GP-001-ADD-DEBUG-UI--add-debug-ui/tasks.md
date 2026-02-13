# Tasks

## Completed

- [x] Create ticket workspace `GP-001-ADD-DEBUG-UI`.
- [x] Create planning and diary docs for ticket.
- [x] Audit `web-agent-debug` frontend/backend surface area with concrete metrics.
- [x] Audit `pinocchio/webchat` runtime and projection pipelines.
- [x] Audit `geppetto/cmd/llm-runner` serve/frontend integration and cleanup opportunities.
- [x] Write deep migration analysis with no-backwards-compatibility plan.
- [x] Record detailed multi-step diary with commands, failures, and review guidance.
- [x] Upload ticket analysis bundle to reMarkable and verify cloud listing.

## Execution Board

### Phase 1: Canonical Debug API (Pinocchio backend)

- [x] `P1.1` Add canonical `/api/debug/*` routes in `pinocchio/pkg/webchat/router.go`.
- [x] `P1.2` Keep existing live handlers as implementation source, but expose new namespace wrappers:
- [x] `P1.2.a` `GET /api/debug/timeline`
- [x] `P1.2.b` `GET /api/debug/turns`
- [x] `P1.2.c` `POST /api/debug/step/enable`
- [x] `P1.2.d` `POST /api/debug/step/disable`
- [x] `P1.2.e` `POST /api/debug/continue`
- [x] `P1.3` Add endpoint tests for all new paths (status code + shape parity).
- [x] `P1.4` Add envelope metadata regression test for `/api/debug/turns` (`conv_id/session_id/phase/since_ms/items`).
- [x] `P1.5` Run `go test ./pkg/webchat` and commit.

### Phase 2: Live Inspector Read Models (Pinocchio backend)

- [ ] `P2.1` Add `GET /api/debug/conversations` from `ConvManager` snapshots.
- [ ] `P2.2` Add `GET /api/debug/conversations/:convId` detail shape.
- [ ] `P2.3` Add `GET /api/debug/events/:convId` backed by sem buffer snapshot with `since_seq` + `limit` filtering.
- [ ] `P2.4` Add `GET /api/debug/turn/:convId/:sessionId/:turnId` helper on top of turn store list + decode.
- [ ] `P2.5` Add tests for conversation/events/turn detail handlers.
- [ ] `P2.6` Run targeted + package tests and commit.

### Phase 3: Offline Viewer Sources (Pinocchio backend)

- [ ] `P3.1` Add offline debug source abstraction (`artifacts` and `sqlite`).
- [ ] `P3.2` Implement filesystem artifact reader endpoints (yaml/ndjson/log).
- [ ] `P3.3` Implement sqlite readers for persisted turns/timelines.
- [ ] `P3.4` Add `runs` and `run detail` APIs using shared envelope DTOs.
- [ ] `P3.5` Add tests with fixture files/sqlite db and commit.

### Phase 4: Frontend Package Extraction In Pinocchio Web Workspace

- [ ] `P4.1` Create package structure under `pinocchio/cmd/web-chat/web`:
- [ ] `P4.1.a` `src/debug-contract`
- [ ] `P4.1.b` `src/debug-api`
- [ ] `P4.1.c` `src/debug-state`
- [ ] `P4.1.d` `src/debug-components`
- [ ] `P4.2` Port baseline debug UI components from `web-agent-example` into pinocchio-owned packages.
- [ ] `P4.3` Update data adapters for timeline proto shape + metadata envelopes.
- [ ] `P4.4` Ensure frontend uses envelope metadata in selectors/context panels (not only `items`).
- [ ] `P4.5` Add package-level tests/story stories and run `npm run check && npm run build`.
- [ ] `P4.6` Commit frontend extraction slice.

### Phase 5: Pinocchio Debug App Shell + Storybook

- [ ] `P5.1` Add pinocchio debug app shell (offline mode + live level-2 mode switch).
- [ ] `P5.2` Wire app shell to canonical `/api/debug/*` endpoints.
- [ ] `P5.3` Add routing/state persistence for selected conversation/run context.
- [ ] `P5.4` Start Storybook in `pinocchio/cmd/web-chat/web` in tmux and keep it running during iteration.
- [ ] `P5.5` Validate key inspector flows manually in Storybook and dev server.
- [ ] `P5.6` Commit app shell slice.

### Phase 6: Cutover + Deletion

- [ ] `P6.1` Remove `web-agent-example/cmd/web-agent-debug` frontend and Go harness.
- [ ] `P6.2` Remove obsolete references/scripts/docs tied to deleted debug app.
- [ ] `P6.3` Ensure no compatibility dependency on old `/debug/*` namespace in frontend.
- [ ] `P6.4` Run repo-level checks impacted by removal and commit.

### Phase 7: Ticket Hygiene Per Slice

- [ ] `P7.1` After each code slice: update diary with commands, failures, commit hash, and review instructions.
- [ ] `P7.2` After each code slice: update changelog and check completed tasks.
- [ ] `P7.3` Upload refreshed ticket docs to reMarkable at major milestones.
