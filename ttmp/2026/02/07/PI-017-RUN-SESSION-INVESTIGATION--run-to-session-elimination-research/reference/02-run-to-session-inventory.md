---
Title: Run to Session Inventory
Ticket: PI-017-RUN-SESSION-INVESTIGATION
Status: active
Topics:
    - backend
    - frontend
    - proto
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go
      Note: startRunForPrompt, run_id in API responses
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/conversation.go
      Note: Conversation.RunID field
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/turn_store.go
      Note: TurnSnapshot.RunID, TurnQuery.RunID
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/turn_store_sqlite.go
      Note: SQL schema with run_id column
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/send_queue.go
      Note: run_id in queue responses
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/proto/sem/middleware/planning.proto
      Note: PlanningRun.run_id (planning domain)
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/inference/events/typed_planning.go
      Note: EventPlanningStart.RunID (planning domain)
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/events/chat-events.go
      Note: LegacyRunID in EventMetadata
ExternalSources: []
Summary: Complete classified inventory of all run terminology in pinocchio, geppetto, and web-agent-example.
LastUpdated: 2026-02-07T08:45:00-05:00
WhatFor: Reference for run-to-session migration planning.
WhenToUse: Use when planning or implementing run→session terminology migration.
---

# Run to Session Inventory

## Goal

Provide a complete, classified inventory of all `run` terminology across pinocchio, geppetto, and web-agent-example repositories.

## Context

The codebase has two distinct uses of "run":

1. **Legacy Session Alias**: `run_id` / `RunID` as an alias for `session_id` / `SessionID`
2. **Planning Domain**: `run_id` in planning events identifies a planning run (iteration cycle)

This inventory classifies every hit to enable targeted migration.

## Classification Legend

| Category | Description | Action |
|----------|-------------|--------|
| **LEGACY_SESSION_ALIAS** | `run_id`/`RunID` meaning session | Must migrate to session terminology |
| **PLANNER_RUN_DOMAIN** | Planning run identifier | Keep or rename to `planning_id` |
| **PROCESS_VERB** | Generic "run" verb (go run, npm run) | Ignore |
| **GENERATED_FROM_PROTO** | Generated code from proto | Change proto source only |
| **DOC_STALE** | Documentation using run terminology | Update documentation |
| **TRUE_SESSION_IDENTIFIER** | Already uses session correctly | No change needed |

---

## LEGACY_SESSION_ALIAS (Must Migrate)

These are the highest priority migration targets.

### Backend: Conversation Struct

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/pkg/webchat/conversation.go:27` | `Conversation.RunID` | Session identifier | Yes | Rename to `SessionID` | Internal | backend |
| `pinocchio/pkg/webchat/conversation.go:219-220` | `run := e.Metadata().SessionID` | Session check | Yes | Rename `run` to `sessionID` | Internal | backend |
| `pinocchio/pkg/webchat/conversation.go:250` | `RunID: runID` | Session assignment | Yes | Rename to `SessionID` | Internal | backend |
| `pinocchio/pkg/webchat/conversation.go:292-293` | `run` variable | Session filter | Yes | Rename to `sessionID` | Internal | backend |

### Backend: Router

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/pkg/webchat/router.go:408` | `sessionID = c.RunID` | Session from conv | Yes | Access `c.SessionID` | Internal | backend |
| `pinocchio/pkg/webchat/router.go:444` | `sessionID = c.RunID` | Session from conv | Yes | Access `c.SessionID` | Internal | backend |
| `pinocchio/pkg/webchat/router.go:681` | `run_id` query param | API input | Yes | Accept both, prefer `session_id` | **Yes** | backend |
| `pinocchio/pkg/webchat/router.go:707` | `RunID: runID` | Query filter | Yes | Rename to `SessionID` | Internal | backend |
| `pinocchio/pkg/webchat/router.go:720` | `"run_id": runID` | API response | Yes | Remove, keep only `session_id` | **Yes** | backend |
| `pinocchio/pkg/webchat/router.go:785` | log `run_id` | Logging | Yes | Keep both for transition | Low | backend |
| `pinocchio/pkg/webchat/router.go:927` | `startRunForPrompt` | Function name | Yes | Rename to `startInferenceForPrompt` | Internal | backend |
| `pinocchio/pkg/webchat/router.go:932` | log `run_id` | Logging | Yes | Keep both for transition | Low | backend |
| `pinocchio/pkg/webchat/router.go:1012` | `conv.RunID` | Step controller scope | Yes | Use `conv.SessionID` | Internal | backend |
| `pinocchio/pkg/webchat/router.go:1029` | "starting run loop" | Log message | Yes | "starting inference" | Low | backend |
| `pinocchio/pkg/webchat/router.go:1046-1047` | `"run_id"` in response | API response | Yes | Remove legacy field | **Yes** | backend |
| `pinocchio/pkg/webchat/router.go:1083` | `SessionID: conv.RunID` | Metadata assignment | Yes | Use `SessionID` | Internal | backend |
| `pinocchio/pkg/webchat/router.go:1091,1093` | log `run loop` | Log messages | Yes | "inference" | Low | backend |

### Backend: TurnStore

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/pkg/webchat/turn_store.go:8` | `TurnSnapshot.RunID` | Session identifier | Yes | Rename to `SessionID` | Internal | backend |
| `pinocchio/pkg/webchat/turn_store.go:18` | `TurnQuery.RunID` | Session filter | Yes | Rename to `SessionID` | Internal | backend |
| `pinocchio/pkg/webchat/turn_store_sqlite.go:50` | `run_id TEXT` | SQL column | Yes | Rename to `session_id` | **Yes (schema)** | backend |
| `pinocchio/pkg/webchat/turn_store_sqlite.go:58` | `turns_by_run` | Index name | Yes | Rename to `turns_by_session` | **Yes (schema)** | backend |
| `pinocchio/pkg/webchat/turn_store_sqlite.go:93` | SQL INSERT | Column name | Yes | Use `session_id` | Internal | backend |
| `pinocchio/pkg/webchat/turn_store_sqlite.go:106,123-124` | Query params | Filter | Yes | Use `SessionID` | Internal | backend |

### Backend: Send Queue

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/pkg/webchat/send_queue.go:127-128` | `"run_id"` in response | API response | Yes | Remove legacy field | **Yes** | backend |
| `pinocchio/pkg/webchat/send_queue.go:144-145` | `"run_id"` in response | API response | Yes | Remove legacy field | **Yes** | backend |
| `pinocchio/pkg/webchat/send_queue_test.go:13,34,46,59` | `RunID` in tests | Test data | Yes | Update to `SessionID` | Internal | backend |

### Backend: Turn Persister

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/pkg/webchat/turn_persister.go:27` | `runID: conv.RunID` | Session for persistence | Yes | Use `SessionID` | Internal | backend |

### Backend: Timeline Projector

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/pkg/webchat/timeline_projector.go:53` | `planning map[string]*planningAgg` | Key is run_id | Mixed | Clarify: planning run, not session | See notes | backend |

### Geppetto Events (Backwards Compat)

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `geppetto/pkg/events/chat-events.go:387` | `LegacyRunID` | JSON compat field | Yes | Remove after migration period | **Yes** | backend |
| `geppetto/pkg/events/chat-events.go:393-394` | Legacy fallback | Compat logic | Yes | Remove after migration | **Yes** | backend |
| `geppetto/pkg/events/chat-events.go:401,403` | `run_id` log field | Log format | Yes | Keep both for transition | Low | backend |

### Simple Chat Agent

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/cmd/agents/simple-chat-agent/main.go:170` | `sessionRunID` | Variable name | Yes | Already uses `session` prefix | None | backend |
| `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go:128-187` | log `run_id` | Logging | Yes | Keep both for transition | Low | backend |
| `pinocchio/cmd/agents/simple-chat-agent/pkg/ui/app.go:393` | `run:` status | UI display | Yes | Change to `session:` | Low | frontend |

### Agent Mode

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/pkg/middlewares/agentmode/sqlite_store.go:30` | `run_id TEXT` | SQL column | Yes | Rename to `session_id` | **Yes (schema)** | backend |
| `pinocchio/pkg/middlewares/agentmode/sqlite_store.go:38,44,57-58` | SQL queries | Session filter | Yes | Use `session_id` | Internal | backend |
| `pinocchio/pkg/middlewares/agentmode/middleware.go:39` | `ModeChange.RunID` | Session identifier | Yes | Rename to `SessionID` | Internal | backend |
| `pinocchio/pkg/middlewares/agentmode/middleware.go:81,168,190-217` | `run_id` in logs/code | Session | Yes | Use `session_id` | Internal | backend |
| `pinocchio/pkg/middlewares/agentmode/service.go:51,89` | `RunID` in map/change | Session key | Yes | Use `SessionID` | Internal | backend |
| `pinocchio/pkg/middlewares/agentmode/schema.sql:19,27-28` | SQL schema | Column/index | Yes | Rename to `session_id` | **Yes (schema)** | backend |
| `pinocchio/pkg/middlewares/agentmode/seed.sql:16-18` | Seed data | Test data | Yes | Use `session-1` | Internal | backend |

---

## PLANNER_RUN_DOMAIN (Keep or Rename to planning_id)

These refer to a **planning run** (iteration cycle), which is a different concept from session.

### Proto Definitions (Source of Truth)

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/proto/sem/middleware/planning.proto:11` | `PlanningRun.run_id` | Planning run ID | **No** | Keep or rename to `planning_id` | **Proto change** | proto |
| `pinocchio/proto/sem/middleware/planning.proto:19,25,39,48` | `PlanningRun run` | Nested message | **No** | Keep or rename field to `planning` | **Proto change** | proto |
| `pinocchio/proto/sem/middleware/planning.proto:58,66` | `run_id` in Execution* | Planning run ref | **No** | Keep or rename to `planning_id` | **Proto change** | proto |
| `pinocchio/proto/sem/timeline/planning.proto:35` | `run_id` in entity | Planning run ref | **No** | Keep or rename to `planning_id` | **Proto change** | proto |

### Go Events (Source)

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/pkg/inference/events/typed_planning.go:12,38,71,92,118,139` | `RunID` field | Planning run ID | **No** | Keep or rename to `PlanningID` | Internal | backend |
| `pinocchio/pkg/inference/events/typed_planning.go:25,57,81,106,127,150` | `RunID:` assignment | Planning run ID | **No** | Keep or rename | Internal | backend |

### SEM Translator (Mapping)

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/pkg/webchat/sem_translator.go:577,606,631,648,668,682` | `ev.RunID` | Planning run ID | **No** | Keep or rename | Internal | backend |
| `pinocchio/pkg/webchat/sem_translator.go:588,625,642,663,677,693` | Proto mapping | Planning events | **No** | Keep or rename | Internal | backend |

### Generated Code (Change Proto Instead)

| Path | Category | Notes |
|------|----------|-------|
| `pinocchio/pkg/sem/pb/proto/sem/middleware/planning.pb.go` | GENERATED_FROM_PROTO | Generated from `planning.proto` |
| `pinocchio/pkg/sem/pb/proto/sem/timeline/planning.pb.go` | GENERATED_FROM_PROTO | Generated from `planning.proto` |
| `pinocchio/cmd/web-chat/web/src/sem/pb/proto/sem/middleware/planning_pb.ts` | GENERATED_FROM_PROTO | Generated TypeScript |
| `pinocchio/cmd/web-chat/web/src/sem/pb/proto/sem/timeline/planning_pb.ts` | GENERATED_FROM_PROTO | Generated TypeScript |
| `pinocchio/web/src/sem/pb/proto/sem/middleware/planning_pb.ts` | GENERATED_FROM_PROTO | Generated TypeScript |
| `pinocchio/web/src/sem/pb/proto/sem/timeline/planning_pb.ts` | GENERATED_FROM_PROTO | Generated TypeScript |

### Frontend Registry (Consumers)

| Path | Symbol/Field | Current Meaning | Maps to Session? | Proposed Change | Breaking? | Owner |
|------|--------------|-----------------|------------------|-----------------|-----------|-------|
| `pinocchio/cmd/web-chat/web/src/sem/registry.ts:326-381` | `pb?.run?.runId` | Planning run ID | **No** | Follow proto change | Internal | frontend |

---

## PROCESS_VERB (Ignore)

These are generic uses of "run" as an English verb and are not identifiers in the runtime model.

| Pattern | Count | Examples |
|---------|-------|----------|
| `go run` | ~50 | Build/test commands |
| `npm run` | ~30 | Frontend commands |
| `docker run` | ~5 | Container commands |
| Function names (`RunLoop`, `RunInference`, `RunIntoWriter`) | ~40 | Execution methods |
| Comments ("run tests", "run the server") | ~200+ | Documentation |
| Package names (`pkg/cmds/run`) | 10 | Execution context package |
| Log messages ("run complete") | ~20 | Operational messages |

---

## DOC_STALE (Update Documentation)

| Path | Issue | Proposed Change |
|------|-------|-----------------|
| `pinocchio/cmd/web-chat/README.md:123` | `run_id` in query params | Update to `session_id` |
| `pinocchio/cmd/web-chat/README.md:181` | `startRunForPrompt` reference | Update function name |
| `pinocchio/pkg/doc/topics/webchat-framework-guide.md:236` | `"run_id": "<uuid>"` | Remove legacy field |
| `pinocchio/pkg/doc/topics/webchat-framework-guide.md:347` | "run loop" terminology | Use "inference" |
| `pinocchio/pkg/doc/topics/webchat-backend-reference.md:255` | `RunID` in struct example | Use `SessionID` |
| `pinocchio/pkg/doc/topics/webchat-backend-reference.md:273,276,302,320` | "run" references | Use "inference" or "session" |
| `pinocchio/pkg/doc/topics/webchat-sem-and-ui.md:73-76,285` | `{ run }` in event payloads | Clarify as planning run |
| `geppetto/pkg/doc/topics/04-events.md:118` | `SessionID   string // legacy name: run_id` | Clarify migration status |
| `geppetto/pkg/doc/topics/08-turns.md` | `Run (session)` comment | Keep as clarification |
| `geppetto/changelog.md:5` | "RunContext" reference | Historical, no change needed |

---

## Summary Statistics

| Category | Count | Action |
|----------|-------|--------|
| LEGACY_SESSION_ALIAS | ~42 | Must migrate |
| PLANNER_RUN_DOMAIN | ~58 | Keep or rename to `planning_id` |
| PROCESS_VERB | ~400 | Ignore |
| GENERATED_FROM_PROTO | ~30 | Change proto source |
| DOC_STALE | ~20 | Update documentation |
| TRUE_SESSION_IDENTIFIER | ~100+ | Already correct |

---

## Data Flow Traces

### Legacy Session Alias Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│ Conversation Creation                                                │
├─────────────────────────────────────────────────────────────────────┤
│ NewConversation(runID)  →  Conversation.RunID = runID               │
│                                                                      │
│ conv.RunID propagates to:                                           │
│   ├── TurnStore.Save(convID, runID, ...)                            │
│   │       └── SQLite: INSERT INTO turns(run_id, ...)                │
│   ├── API response: {"run_id": "...", "session_id": "..."}          │
│   ├── StepController scope: StepScope{SessionID: conv.RunID}        │
│   ├── AgentMode store: agent_mode_changes.run_id                    │
│   └── Logs: Str("run_id", conv.RunID)                               │
└─────────────────────────────────────────────────────────────────────┘
```

### Planning Run Flow (Separate Domain)

```
┌─────────────────────────────────────────────────────────────────────┐
│ Planning Middleware                                                  │
├─────────────────────────────────────────────────────────────────────┤
│ NewPlanningStart(runID=UUID)  →  EventPlanningStart.RunID           │
│                                                                      │
│ RunID propagates to:                                                │
│   ├── SEM events: planning.start, planning.iteration, etc.          │
│   ├── Proto: PlanningRun.run_id                                     │
│   ├── Timeline entity: TimelineEntityV1.run_id                      │
│   └── Frontend: PlanningWidget aggregates by runId                  │
│                                                                      │
│ NOTE: This is NOT the same as session_id!                           │
│       A single session can have multiple planning runs.             │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Usage Examples

### Querying This Inventory

To find all backend files that need session migration:
```bash
grep -E "LEGACY_SESSION_ALIAS.*backend" reference/02-run-to-session-inventory.md
```

### Verifying Migration Progress

After migration, re-run inventory scan and expect:
- Zero LEGACY_SESSION_ALIAS hits
- PLANNER_RUN_DOMAIN hits unchanged (or renamed to `planning_id`)
- Documentation updated to reflect session terminology
