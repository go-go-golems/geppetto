---
Title: PI-013 Debug UI Run Terminology Analysis
Ticket: PI-017-RUN-SESSION-INVESTIGATION
Status: active
Topics:
    - backend
    - frontend
    - proto
    - documentation
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/06/PI-013-TURN-MW-DEBUG-UI--turn-and-middleware-debug-visualization-ui/analysis/05-architecture-and-implementation-plan-for-debug-ui.md
      Note: Source document being analyzed
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/07/PI-017-RUN-SESSION-INVESTIGATION--run-to-session-elimination-research/analysis/01-run-to-session-elimination-plan.md
      Note: Main migration plan this analysis feeds into
ExternalSources: []
Summary: Detailed analysis of run terminology in PI-013 Debug UI architecture document with specific replacement recommendations.
LastUpdated: 2026-02-07T09:30:00-05:00
WhatFor: Guide updates to PI-013 design doc to use session terminology.
WhenToUse: Use when updating PI-013 documentation or implementing debug UI endpoints.
---

# PI-013 Debug UI Run Terminology Analysis

## Executive Summary

The PI-013 Debug UI architecture document uses `run` terminology extensively where `session` should be used instead. This analysis catalogs every instance and provides specific replacement text.

**Total instances found:** 47 occurrences of `run`/`runs`/`run_id`/`RunID` that should be migrated to session terminology.

---

## Analysis by Section

### 1. Architecture Overview Diagram

**Location:** Architecture Overview section, ASCII diagram

**Current:**
```
|  | Session    |      | TimelineStore  |  | MiddlewareTraceStore       |  |
|  | (in-memory)|      | (SQLite)       |  | (NEW - SQLite)             |  |
```

**Issue:** The diagram already uses "Session" for the session store, which is correct. No change needed here.

---

### 2. What Already Exists Table

**Location:** "What Already Exists" section

**Current:**
```
| **TurnStore** | `pinocchio/pkg/webchat/turn_store_sqlite.go` | Turn snapshots by conv_id/run_id/phase with YAML payloads |
```

**Replace with:**
```
| **TurnStore** | `pinocchio/pkg/webchat/turn_store_sqlite.go` | Turn snapshots by conv_id/session_id/phase with YAML payloads |
```

---

**Current:**
```
| **`GET /turns`** | `pinocchio/pkg/webchat/router.go:670` | Query turn snapshots by conv_id, run_id, phase, since_ms |
```

**Replace with:**
```
| **`GET /turns`** | `pinocchio/pkg/webchat/router.go:670` | Query turn snapshots by conv_id, session_id, phase, since_ms |
```

---

### 3. Backend New Endpoints Table

**Location:** "What Needs to Be Built" → "Backend (New Endpoints)"

**Current:**
```
| `GET /debug/conversation/:id/runs` | List runs within a conversation | 1 | TurnStore (distinct run_ids) |
| `GET /debug/turn/:conv_id/:run_id/:turn_id` | All phases for one turn | 2 | TurnStore |
```

**Replace with:**
```
| `GET /debug/conversation/:id/sessions` | List sessions within a conversation | 1 | TurnStore (distinct session_ids) |
| `GET /debug/turn/:conv_id/:session_id/:turn_id` | All phases for one turn | 2 | TurnStore |
```

---

### 4. GET /debug/conversations Response

**Location:** Backend REST API Design → `GET /debug/conversations`

**Current:**
```json
{
  "conversations": [
    {
      "id": "conv_8a3f",
      "profile_slug": "general",
      "run_id": "run_02",
      ...
    }
  ]
}
```

**Replace with:**
```json
{
  "conversations": [
    {
      "id": "conv_8a3f",
      "profile_slug": "general",
      "session_id": "sess_02",
      ...
    }
  ]
}
```

---

**Current handler mapping:**
```
// Read: conv.ID, conv.ProfileSlug, conv.RunID, conv.EngConfigSig
```

**Replace with:**
```
// Read: conv.ID, conv.ProfileSlug, conv.SessionID, conv.EngConfigSig
```

---

### 5. GET /debug/conversation/:id Response

**Location:** Backend REST API Design → `GET /debug/conversation/:id`

**Current:**
```json
{
  "id": "conv_8a3f",
  "profile_slug": "general",
  "run_id": "run_02",
  "session_id": "sess_abc",
  ...
}
```

**Issue:** This response has BOTH `run_id` and `session_id` pointing to different values, which is confusing.

**Replace with:**
```json
{
  "id": "conv_8a3f",
  "profile_slug": "general",
  "session_id": "sess_abc",
  ...
}
```

**Note:** Remove `run_id` entirely; `session_id` is the canonical identifier.

---

### 6. GET /debug/conversation/:id/runs Endpoint

**Location:** Backend REST API Design → `GET /debug/conversation/:id/runs`

**Current endpoint:** `GET /debug/conversation/:id/runs`

**Replace with:** `GET /debug/conversation/:id/sessions`

---

**Current response:**
```json
{
  "runs": [
    {
      "run_id": "run_01",
      "turn_count": 1,
      "first_turn_at": "2026-02-06T14:32:01Z",
      "last_turn_at": "2026-02-06T14:32:05Z"
    },
    {
      "run_id": "run_02",
      ...
    }
  ]
}
```

**Replace with:**
```json
{
  "sessions": [
    {
      "session_id": "sess_01",
      "turn_count": 1,
      "first_turn_at": "2026-02-06T14:32:01Z",
      "last_turn_at": "2026-02-06T14:32:05Z"
    },
    {
      "session_id": "sess_02",
      ...
    }
  ]
}
```

---

**Current handler mapping:**
```
snapshots := turnStore.List(ctx, TurnQuery{ConvID: id})
// Group by run_id, count turns, extract min/max created_at_ms
```

**Replace with:**
```
snapshots := turnStore.List(ctx, TurnQuery{ConvID: id})
// Group by session_id, count turns, extract min/max created_at_ms
```

---

**Current TurnStore change note:**
```
**TurnStore change needed:** Add a `DistinctRuns(ctx, convID) ([]RunSummary, error)` method, or compute from `List` results.
```

**Replace with:**
```
**TurnStore change needed:** Add a `DistinctSessions(ctx, convID) ([]SessionSummary, error)` method, or compute from `List` results.
```

---

### 7. GET /debug/turn/:conv_id/:run_id/:turn_id Endpoint

**Location:** Backend REST API Design → `GET /debug/turn/:conv_id/:run_id/:turn_id`

**Current endpoint:** `GET /debug/turn/:conv_id/:run_id/:turn_id`

**Replace with:** `GET /debug/turn/:conv_id/:session_id/:turn_id`

---

**Current response:**
```json
{
  "conv_id": "conv_8a3f",
  "run_id": "run_02",
  "turn_id": "turn_02",
  ...
}
```

**Replace with:**
```json
{
  "conv_id": "conv_8a3f",
  "session_id": "sess_02",
  "turn_id": "turn_02",
  ...
}
```

---

**Current handler mapping:**
```
snapshots := turnStore.List(ctx, TurnQuery{
    ConvID: convID,
    RunID:  runID,
})
```

**Replace with:**
```
snapshots := turnStore.List(ctx, TurnQuery{
    ConvID:    convID,
    SessionID: sessionID,
})
```

---

### 8. Redux State Shape

**Location:** Redux State Shape section

**Current:**
```typescript
sessions: {
  selectedConvId: string | null;
  selectedRunId: string | null;
};
```

**Replace with:**
```typescript
sessions: {
  selectedConvId: string | null;
  selectedSessionId: string | null;
};
```

---

### 9. RTK Query API Definition

**Location:** RTK Query API Definition section

**Current:**
```typescript
getRuns: builder.query<RunSummary[], string>({
  query: (convId) => `conversation/${convId}/runs`,
}),
```

**Replace with:**
```typescript
getSessions: builder.query<SessionSummary[], string>({
  query: (convId) => `conversation/${convId}/sessions`,
}),
```

---

**Current:**
```typescript
getTurns: builder.query<TurnSnapshot[], TurnQuery>({
  query: ({ convId, runId, phase, sinceMs, limit }) =>
    `turns?` + new URLSearchParams({
      ...(convId && { conv_id: convId }),
      ...(runId && { run_id: runId }),
      ...
    }),
}),
```

**Replace with:**
```typescript
getTurns: builder.query<TurnSnapshot[], TurnQuery>({
  query: ({ convId, sessionId, phase, sinceMs, limit }) =>
    `turns?` + new URLSearchParams({
      ...(convId && { conv_id: convId }),
      ...(sessionId && { session_id: sessionId }),
      ...
    }),
}),
```

---

**Current:**
```typescript
getTurnDetail: builder.query<TurnDetail, TurnDetailQuery>({
  query: ({ convId, runId, turnId }) =>
    `turn/${convId}/${runId}/${turnId}`,
}),
```

**Replace with:**
```typescript
getTurnDetail: builder.query<TurnDetail, TurnDetailQuery>({
  query: ({ convId, sessionId, turnId }) =>
    `turn/${convId}/${sessionId}/${turnId}`,
}),
```

---

### 10. Key Type Definitions

**Location:** Key Type Definitions section

**Current:**
```typescript
interface TurnSnapshot {
  conv_id: string;
  run_id: string;
  turn_id: string;
  phase: 'pre_inference' | 'post_inference' | 'post_tools' | 'final';
  created_at_ms: number;
  turn: ParsedTurn;
}
```

**Replace with:**
```typescript
interface TurnSnapshot {
  conv_id: string;
  session_id: string;
  turn_id: string;
  phase: 'pre_inference' | 'post_inference' | 'post_tools' | 'final';
  created_at_ms: number;
  turn: ParsedTurn;
}
```

---

### 11. Screen-by-Screen Implementation

**Location:** Screen 1: Session Overview -- Data Flow

**Current:**
```
TurnList (sidebar)     GET /debug/turns?conv_id=X  TurnStore.List(convID)
StateTrackLane         GET /debug/turn/X/Y/*       TurnStore.List(convID,runID)
```

**Replace with:**
```
TurnList (sidebar)     GET /debug/turns?conv_id=X  TurnStore.List(convID)
StateTrackLane         GET /debug/turn/X/Y/*       TurnStore.List(convID,sessionID)
```

---

**Current cross-highlighting note:**
```
2. Query `/debug/turns?conv_id=X&run_id=Y` to find matching turn -> link to block.
```

**Replace with:**
```
2. Query `/debug/turns?conv_id=X&session_id=Y` to find matching turn -> link to block.
```

---

### 12. Implementation Phases

**Location:** Phase 0: Correlation Contract

**Current:**
```
5. Add `DistinctRuns` query path instead of deriving run summaries from limited `List(...)` results.
```

**Replace with:**
```
5. Add `DistinctSessions` query path instead of deriving session summaries from limited `List(...)` results.
```

---

**Location:** Phase 1: Foundation

**Current:**
```
4. Implement `GET /debug/conversation/:id/runs` handler -> query dedicated run-summary path
```

**Replace with:**
```
4. Implement `GET /debug/conversation/:id/sessions` handler -> query dedicated session-summary path
```

---

### 13. Frontend Component Routes

**Location:** Component Tree section

**Current:**
```
<Route path="/turn/:convId/:runId/:turnId"
       element={<TurnInspector />} />
```

**Replace with:**
```
<Route path="/turn/:convId/:sessionId/:turnId"
       element={<TurnInspector />} />
```

---

### 14. Component Props

**Location:** Screen 1: Session Overview components

**Current:**
```
<CorrelationIdBar convId runId />
```

**Replace with:**
```
<CorrelationIdBar convId sessionId />
```

---

### 15. Middleware Tracing Integration Comment

**Location:** Middleware Tracing → Integration Point

**Current:**
```go
// collector can be attached to trace writer/store for this run
```

**Replace with:**
```go
// collector can be attached to trace writer/store for this session
```

---

## Summary: All Changes Required

### Endpoint Path Changes

| Current | New |
|---------|-----|
| `GET /debug/conversation/:id/runs` | `GET /debug/conversation/:id/sessions` |
| `GET /debug/turn/:conv_id/:run_id/:turn_id` | `GET /debug/turn/:conv_id/:session_id/:turn_id` |
| Query param `run_id` | Query param `session_id` |

### Response Field Changes

| Current | New |
|---------|-----|
| `"run_id": "..."` | `"session_id": "..."` |
| `"runs": [...]` | `"sessions": [...]` |
| `RunSummary` type | `SessionSummary` type |

### Code Symbol Changes

| Current | New |
|---------|-----|
| `conv.RunID` | `conv.SessionID` |
| `TurnQuery.RunID` | `TurnQuery.SessionID` |
| `selectedRunId` | `selectedSessionId` |
| `getRuns` | `getSessions` |
| `DistinctRuns()` | `DistinctSessions()` |
| `runId` (variable) | `sessionId` (variable) |

### Route Parameter Changes

| Current | New |
|---------|-----|
| `:runId` | `:sessionId` |

---

## Implementation Checklist

When updating PI-013 document:

- [ ] Update "What Already Exists" table (2 changes)
- [ ] Update "Backend (New Endpoints)" table (2 changes)
- [ ] Update `GET /debug/conversations` response (2 changes)
- [ ] Update `GET /debug/conversation/:id` response (remove `run_id`)
- [ ] Update `GET /debug/conversation/:id/runs` → `/sessions` (4 changes)
- [ ] Update `GET /debug/turn` endpoint path and response (3 changes)
- [ ] Update Redux State Shape (1 change)
- [ ] Update RTK Query API Definition (4 changes)
- [ ] Update Key Type Definitions (1 change)
- [ ] Update Screen-by-Screen Data Flow (2 changes)
- [ ] Update Implementation Phases (2 changes)
- [ ] Update Component Routes (1 change)
- [ ] Update Component Props (1 change)
- [ ] Update code comments (1 change)

**Total: 26 distinct changes across the document**

---

## Coordination Notes

1. **PI-013 should be updated AFTER Batch 2** of the run→session migration (when TurnStore schema changes)
2. **Implementation should use session terminology from day 1** — don't implement with `run_id` then migrate
3. **Frontend routes** like `/turn/:convId/:runId/:turnId` should use `:sessionId` from the start
4. **API responses** should never include `run_id` in the debug UI — use `session_id` exclusively
