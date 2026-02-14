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
