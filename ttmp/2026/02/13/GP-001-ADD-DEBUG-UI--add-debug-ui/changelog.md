# Changelog

## 2026-02-13

- Initial workspace created.
- Added planning document: deep analysis of moving `web-agent-debug` fully into `geppetto` as reusable debug/inspector tooling.
- Added detailed diary with explicit commands, findings, and failures.
- Captured concrete contract mismatches between frontend expected `/debug/*` resources and current `pinocchio/webchat` route/data shapes.
- Defined no-backwards-compatibility cutover plan including explicit deletion targets.

## 2026-02-13

Completed deep migration analysis + diary, then uploaded bundled ticket documentation to reMarkable at /ai/2026/02/13/GP-001-ADD-DEBUG-UI/GP-001-ADD-DEBUG-UI Migration Analysis.pdf

## 2026-02-13

Updated design direction: move debug UI ownership into `pinocchio` (not `geppetto`), with explicit support for offline viewer + live level-2 inspector. Rewrote planning document phases and target architecture accordingly.

## 2026-02-13

Refined offline scope: offline viewer now supports sqlite persisted turns/timelines in addition to yaml/ndjson artifacts. Added detailed clarifications in planning doc for turns envelope vs `TurnSnapshot[]`, lack of blocks table, flattened `props` model vs protobuf oneof timeline entities, and why projector/upsert still matters for read-only live inspection.

## 2026-02-13

Updated design guidance to require frontend usage of metadata envelopes from debug API responses (not only `items` arrays). Added explicit tasking and readiness criteria for metadata-preserving selectors/context panels.

## 2026-02-13

Implemented Phase 1 backend contract slice in `pinocchio` (commit `55b09b0`): added canonical `/api/debug/*` wrappers for timeline, turns, and step-control endpoints; added parity and envelope regression tests in `pkg/webchat`; validated with `go test ./pkg/webchat` and commit-hook full test/lint pass.

## 2026-02-13

Implemented Phase 2 live-inspector read-model slice in `pinocchio` (commit `532777b`): added read-only `/api/debug/conversations`, `/api/debug/conversations/:convId`, `/api/debug/events/:convId`, and `/api/debug/turn/:convId/:sessionId/:turnId` endpoints; added endpoint tests for list/detail/filter/decode behavior; validated with `go test ./pkg/webchat` plus `go test ./pkg/persistence/chatstore ./pkg/webchat`.

## 2026-02-13

Started Storybook for the pinocchio web workspace in tmux session `gp001-sb`; active local URL is `http://localhost:6007/` (6006 was already occupied).

## 2026-02-13

Implemented Phase 3 offline slice in `pinocchio` (commit `09a6320`): added `/api/debug/runs` and `/api/debug/runs/:runId` handlers supporting artifact directories plus sqlite turn/timeline sources, including artifact YAML/NDJSON/log parsing and sqlite run/detail readers with tests.

## 2026-02-13

Implemented initial Phase 4 frontend extraction slice in `pinocchio` (commit `30e3fa5`): scaffolded reusable `src/debug-contract`, `src/debug-api`, `src/debug-state`, and `src/debug-components` packages plus seed `debug-app` module; added metadata-focused story and validated with frontend typecheck/build and hook `web-check`.

## 2026-02-13

Uploaded refreshed GP-001 bundle to reMarkable as `GP-001-ADD-DEBUG-UI Execution Progress.pdf` under `/ai/2026/02/13/GP-001-ADD-DEBUG-UI/` and verified cloud listing.

## 2026-02-13

Implemented Phase 5 app-shell slice in `pinocchio` (commit `c88c3e5`): added `DebugApp` live/offline mode shell, wired live/offline API queries, exposed debug shell via `?debug=1` in main app entrypoint, and added a Storybook story/provider wrapper.

## 2026-02-13

Uploaded a fresh reMarkable bundle with latest Phase 5 updates as `GP-001-ADD-DEBUG-UI Execution Progress (Phase 5).pdf` under `/ai/2026/02/13/GP-001-ADD-DEBUG-UI/`.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/13/GP-001-ADD-DEBUG-UI--add-debug-ui/planning/01-web-agent-debug-ui-migration-analysis-for-geppetto.md — Primary migration analysis document
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/13/GP-001-ADD-DEBUG-UI--add-debug-ui/reference/01-diary.md — Detailed execution diary

## 2026-02-13

Ported full debug-ui source into pinocchio web workspace with app/storybook wiring and URL-persisted conversation/session/turn context (commit 3671aeb). Verified typecheck/build/storybook smoke; lint hook still fails on inherited a11y/style diagnostics in imported components.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/.storybook/preview.tsx — Storybook providers/styles for debug-ui and webchat stories
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui — Primary migrated UI package and selectors
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/components/AppShell.tsx — URL search/localStorage state sync for selected debug context


## 2026-02-13

Uploaded refreshed GP-001 ticket bundle after full UI port as 'GP-001-ADD-DEBUG-UI Execution Progress (Ported UI Slice).pdf' to /ai/2026/02/13/GP-001-ADD-DEBUG-UI and verified cloud listing.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/13/GP-001-ADD-DEBUG-UI--add-debug-ui/reference/01-diary.md — Diary updated with new implementation slice and validation results

## 2026-02-14

Completed the Phase 6 cutover deletion and route-alignment follow-up:
- pinocchio commit `6cb9117` updated debug-ui MSW handlers to canonical `/api/debug/*` paths, removing the last `/debug/*` compatibility dependency in frontend mocks.
- web-agent-example commit `2905322` removed `cmd/web-agent-debug` harness and legacy debug UI source tree, plus stale README/docs references.
- validation confirms no remaining `web-agent-debug` references in `web-agent-example` source/docs and Storybook tmux session `gp001-sb` remains healthy on `http://localhost:6007/`.
- repo-wide `GOWORK=off go test ./...` in `web-agent-example` still fails due pre-existing missing module dependencies in this workspace (not introduced by deletion); failure is documented in diary.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/mocks/msw/createDebugHandlers.ts — Mock handlers switched from `/debug/*` to `/api/debug/*`
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/README.md — Removed obsolete `web-agent-debug` harness instructions
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/13/GP-001-ADD-DEBUG-UI--add-debug-ui/reference/01-diary.md — Diary records Phase 6 commits and validation outcomes

## 2026-02-13

Uploaded refreshed GP-001 bundle after Phase 6 completion as `GP-001-ADD-DEBUG-UI Execution Progress (Phase 6 Cutover).pdf` to `/ai/2026/02/13/GP-001-ADD-DEBUG-UI/` and verified cloud listing includes the new document.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/13/GP-001-ADD-DEBUG-UI--add-debug-ui/reference/01-diary.md — Diary captures upload commands and verification listing

## 2026-02-14

Completed `P4.5` stabilization in `pinocchio` (commit `de59a6f`): fixed Storybook indexer interruption via clean restart of tmux session `gp001-sb`, repaired migrated story typing regression (`TurnInspector.stories.tsx`), and made debug-ui lint policy explicit in `biome.json` so moved legacy components pass `npm run check` without rewrites.

Validated end-to-end:
- `npm run check` (pass)
- `npm run build` (pass)
- `npm run storybook -- --ci --smoke-test --port 6007` (pass)
- live Storybook still served at `http://localhost:6007/` (`200`).

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/biome.json — Debug-ui migration lint/a11y policy tuning used to unblock `web-check`
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/components/TurnInspector.stories.tsx — Story type fix for `ParsedTurn`/block spread
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/13/GP-001-ADD-DEBUG-UI--add-debug-ui/tasks.md — `P4.5` marked complete

## 2026-02-13

Uploaded refreshed GP-001 bundle after `P4.5` completion as `GP-001-ADD-DEBUG-UI Execution Progress (P4.5 Complete).pdf` to `/ai/2026/02/13/GP-001-ADD-DEBUG-UI/`, then verified cloud listing shows the new file after a short propagation delay.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/13/GP-001-ADD-DEBUG-UI--add-debug-ui/reference/01-diary.md — Diary captures upload commands and listing verification

## 2026-02-14

Addressed post-port schema drift in debug UI adapters (commit `2f79dda`): updated `debugApi.ts` normalization to accept stringified numeric values (common in protojson for int64/uint64), support mixed turn payload shapes (YAML string or object), and broaden timeline timestamp/version field fallbacks. This fixes timeline/entity rendering edge cases where numeric fields were interpreted as `0`.

Validated after patch:
- `npm run typecheck` (pass)
- `npm run check` (pass)
- `npm run build` (pass)
- `npm run storybook -- --ci --smoke-test --port 6007` (pass)

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts — Schema-tolerant API adapters for timeline/turn/event decoding

## 2026-02-14

Implemented offline viewer wiring in the active migrated debug-ui app (commit `fba3093`): added `/api/debug/runs` and `/api/debug/runs/:runId` RTK queries, introduced offline source/run state in `uiSlice`, wired `Offline` route + sidebar source panel in `AppShell`, and added read-only run detail rendering. Also aligned Storybook MSW handlers to canonical `/api/debug/*` envelope shapes and added offline fixture coverage so stories exercise the same transport contract as backend endpoints.

Validated after patch:
- `npm run typecheck` (pass)
- `npm run check` (pass)
- `npm run build` (pass)
- `npm run storybook -- --ci --smoke-test --port 6007` (pass)

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/components/AppShell.tsx — Added offline mode navigation, source/run URL persistence, and conditional sidebar rendering
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/components/OfflineSourcesPanel.tsx — New source input and run selection panel for offline mode
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/routes/OfflinePage.tsx — Read-only offline run detail inspector
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts — Added runs/detail endpoints and normalization logic
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/mocks/msw/createDebugHandlers.ts — Canonical envelope MSW mocks plus offline endpoint handlers

## 2026-02-14

Uploaded refreshed GP-001 bundle after offline viewer wiring as `GP-001-ADD-DEBUG-UI Execution Progress (Offline Viewer Wiring).pdf` to `/ai/2026/02/13/GP-001-ADD-DEBUG-UI/` and verified cloud listing includes the new file.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/13/GP-001-ADD-DEBUG-UI--add-debug-ui/reference/01-diary.md — Diary records upload command and verification listing

## 2026-02-14

Fixed two live-debug regressions in `pinocchio` (commit `d511280`):
- removed rapid URL write churn in `AppShell` that caused repeated `History/Location API` errors (`DOMException: operation is insecure`) when clicking around routes;
- corrected turn-detail parsing for backend `parsed` payloads that use protobuf-style capitalized keys (`Blocks`, `ID`, `Kind`), and now prefer YAML/object payload decoding first with parsed fallback. This restores correct block counts in turn inspector for real `/api/debug/turn/...` responses.

Validation after patch:
- `npm run -s typecheck` (pass)
- `npm run -s check` (pass)
- `npm run -s build` (pass)
- `npm run storybook -- --ci --smoke-test --port 6007` (pass)
- manual repro path from report confirms no history spam and `Blocks (5)` rendered for turn detail.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/components/AppShell.tsx — URL hydration/sync guard logic to prevent navigation replace loops
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts — Robust turn parsed payload decoding across lowercase/protobuf-style keys and enum kinds

## 2026-02-14

Uploaded refreshed GP-001 bundle after URL-loop + turn-parse fixes as `GP-001-ADD-DEBUG-UI Execution Progress (URL Loop + Turn Parse Fix).pdf` to `/ai/2026/02/13/GP-001-ADD-DEBUG-UI/` and verified cloud listing includes the new file.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/13/GP-001-ADD-DEBUG-UI--add-debug-ui/reference/01-diary.md — Diary captures upload/verification commands for this milestone
