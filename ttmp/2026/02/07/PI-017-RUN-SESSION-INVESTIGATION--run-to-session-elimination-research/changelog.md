# Changelog

## 2026-02-07

- Initial workspace created


## 2026-02-07

Completed run-to-session terminology investigation: inventory scan (674 hits), classification (42 LEGACY_SESSION_ALIAS, 58 PLANNER_RUN_DOMAIN), elimination plan with 5 migration batches, and open questions document

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Primary migration target with startRunForPrompt and API run_id


## 2026-02-07

Added PI-013 Debug UI run terminology analysis: 47 instances of run terminology found, 26 distinct changes needed across endpoints, responses, types, and components

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/06/PI-013-TURN-MW-DEBUG-UI--turn-and-middleware-debug-visualization-ui/analysis/05-architecture-and-implementation-plan-for-debug-ui.md — Source document analyzed for run terminology


## 2026-02-07

Updated PI-013 design document: replaced all run terminology with session terminology (26 changes across endpoints, types, routes, components)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/06/PI-013-TURN-MW-DEBUG-UI--turn-and-middleware-debug-visualization-ui/analysis/05-architecture-and-implementation-plan-for-debug-ui.md — Updated to use session_id instead of run_id throughout


## 2026-02-07

Switched PI-017 into hard-cut implementation mode, added detailed execution checklist, and captured fresh post-PI-018 baseline inventory for code migration

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/07/PI-017-RUN-SESSION-INVESTIGATION--run-to-session-elimination-research/tasks.md — Detailed implementation checklist with task 0 completed
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/07/PI-017-RUN-SESSION-INVESTIGATION--run-to-session-elimination-research/reference/01-diary.md — Hard-cut kickoff step and baseline scan commands


## 2026-02-07

Executed hard-cut run→session migration across pinocchio webchat, persistence, frontend terminology, and geppetto metadata/logging; removed session-meaning `run_id` aliases from API/docs while keeping SQLite schema rename shims for legacy DBs

### Commits

- `4e1729e` (pinocchio): webchat session hard-cut (`RunID`→`SessionID`, `/turns` session_id-only, response payload cleanup, queue/router symbol renames)
- `bc50725` (pinocchio): agentmode/session persistence migration + frontend state/log key cleanup
- `e9f6b4c` (pinocchio): webchat docs/examples terminology cleanup to session-only identifiers
- `d14f3ad` (geppetto): remove `LegacyRunID` compatibility and session-alias `run_id` log emission from event metadata/middlewares

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — API + queue + inference entrypoint migration
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/turn_store_sqlite.go — Legacy `run_id`→`session_id` column/index migration shim
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/middlewares/agentmode/sqlite_store.go — AgentMode persistence migration shim
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/events/chat-events.go — Removed LegacyRunID + `run_id` metadata log emission

