# Changelog

## 2026-02-14

- Initial workspace created


## 2026-02-14

Created runtime SnapshotDiff integration plan, gap inventory, and execution task list.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-019-DEBUG-UI-SNAPSHOT-DIFF-RUNTIME--integrate-snapshot-diff-and-remaining-debug-ui-surface/design/01-implementation-plan-runtime-snapshot-diff-integration.md — Primary implementation guide
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-019-DEBUG-UI-SNAPSHOT-DIFF-RUNTIME--integrate-snapshot-diff-and-remaining-debug-ui-surface/reference/01-diary.md — Exploration diary


## 2026-02-14

Implemented runtime SnapshotDiff + offline inspector integration, lane data wiring, metadata-first rendering, uiSlice cleanup, and regression tests (commit 58ebcef).

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/components/TurnInspector.tsx — Mounted SnapshotDiff in runtime and wired block selection back to inspector
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/routes/OfflinePage.tsx — Upgraded offline viewer to inspector tabs and timeline lanes
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/store/uiSlice.ts — Removed dead fields/actions and added selectedEntityId


## 2026-02-14

Ticket closed

