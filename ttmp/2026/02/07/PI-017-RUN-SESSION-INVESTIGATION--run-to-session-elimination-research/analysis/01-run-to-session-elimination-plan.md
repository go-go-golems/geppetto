---
Title: Run to Session Elimination Plan
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
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/07/PI-017-RUN-SESSION-INVESTIGATION--run-to-session-elimination-research/reference/02-run-to-session-inventory.md
      Note: Full inventory of run terminology
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/06/PI-014-CORRELATION-CONTRACT-DEBUG-UI--correlation-contract-and-debug-ui-migration-plan/analysis/01-correlation-contract-tracing-and-migration-implementation-plan.md
      Note: Related debug UI migration plan
ExternalSources: []
Summary: Phased migration plan to eliminate legacy run_id terminology in favor of session_id.
LastUpdated: 2026-02-07T09:00:00-05:00
WhatFor: Guide implementation of run→session terminology migration.
WhenToUse: Use when planning sprints or implementing migration batches.
---

# Run to Session Elimination Plan

## Executive Summary

The codebase uses `run_id` and `RunID` as a legacy alias for `session_id`/`SessionID`. This creates confusion, especially since planning events use `run_id` for a different concept (planning iteration runs). This plan eliminates the legacy alias while preserving the planning domain's use of `run_id`.

**Key Decision:** Planning `run_id` will be kept as-is (not renamed to `planning_id`) because it's already scoped within `PlanningRun` messages and doesn't conflict with session terminology in practice.

## Goals

1. **Eliminate confusion**: Single source of truth for session identification
2. **Preserve planning domain**: Keep `PlanningRun.run_id` as a distinct concept
3. **Backwards compatibility period**: Provide transition period for external consumers
4. **Coordinate with debug UI migration**: Align with PI-013/PI-014 endpoint migration

## Non-Goals

1. Renaming `PlanningRun.run_id` to `planning_id` (low value, high churn)
2. Breaking external APIs immediately (graceful deprecation instead)
3. Renaming process verb usages ("run tests", `RunLoop`, etc.)

---

## Architecture Decision: Two Domains

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Session Domain                             │
├─────────────────────────────────────────────────────────────────────┤
│ Canonical ID: session_id                                            │
│ Scope: Conversation lifetime                                        │
│ Used by: Webchat, TurnStore, AgentMode, StepController              │
│                                                                      │
│ MIGRATION: All run_id/RunID → session_id/SessionID                  │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                          Planning Domain                             │
├─────────────────────────────────────────────────────────────────────┤
│ Canonical ID: PlanningRun.run_id                                    │
│ Scope: Single planning iteration cycle                              │
│ Used by: Planning events, Timeline projector, PlanningWidget        │
│                                                                      │
│ NO CHANGE: Distinct concept, already namespaced                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Migration Batches

### Batch 0: Preparation (Low Risk)

**Goal:** Prepare tooling and add deprecation notices.

| Task | File | Action |
|------|------|--------|
| Add linter rule | `golangci.yml` | Warn on `RunID` in webchat package |
| Add deprecation comments | Multiple | Mark `RunID` fields with `// Deprecated: use SessionID` |
| Create migration tracking issue | GitHub | Track progress across batches |

**Duration:** 1 day  
**Breaking Changes:** None

---

### Batch 1: Backend Internal Names + Logs (Low Risk)

**Goal:** Rename internal symbols without API changes.

| Task | File | Old | New |
|------|------|-----|-----|
| Rename struct field | `conversation.go` | `Conversation.RunID` | `Conversation.SessionID` |
| Update log field references | `router.go` | `.Str("run_id", conv.RunID)` | `.Str("session_id", conv.SessionID).Str("run_id", conv.SessionID)` |
| Rename function | `router.go` | `startRunForPrompt` | `startInferenceForPrompt` |
| Rename test data | `send_queue_test.go` | `RunID: "r1"` | `SessionID: "r1"` |
| Update variable names | Multiple | `run`, `runID`, `runLog` | `session`, `sessionID`, `sessionLog` |

**Duration:** 2 days  
**Breaking Changes:** None (internal only)  
**Testing:** All existing tests pass

---

### Batch 2: Store/Query/Schema Fields (Medium Risk)

**Goal:** Migrate storage layer to session terminology.

| Task | File | Old | New | Migration |
|------|------|-----|-----|-----------|
| Rename struct field | `turn_store.go` | `TurnSnapshot.RunID` | `TurnSnapshot.SessionID` | |
| Rename query field | `turn_store.go` | `TurnQuery.RunID` | `TurnQuery.SessionID` | |
| SQL schema | `turn_store_sqlite.go` | `run_id TEXT` | `session_id TEXT` | ALTER TABLE |
| SQL index | `turn_store_sqlite.go` | `turns_by_run` | `turns_by_session` | DROP/CREATE INDEX |
| AgentMode schema | `agentmode/sqlite_store.go` | `run_id TEXT` | `session_id TEXT` | ALTER TABLE |
| AgentMode struct | `agentmode/middleware.go` | `ModeChange.RunID` | `ModeChange.SessionID` | |
| Persister | `turn_persister.go` | `runID: conv.RunID` | `sessionID: conv.SessionID` | |

**Schema Migration Script:**
```sql
-- Turn store
ALTER TABLE turns RENAME COLUMN run_id TO session_id;
DROP INDEX IF EXISTS turns_by_run;
CREATE INDEX turns_by_session ON turns(session_id, created_at_ms DESC);

-- AgentMode store
ALTER TABLE agent_mode_changes RENAME COLUMN run_id TO session_id;
DROP INDEX IF EXISTS idx_agent_mode_changes_run_id_at;
CREATE INDEX idx_agent_mode_changes_session_id_at ON agent_mode_changes(session_id, at);
```

**Duration:** 3 days  
**Breaking Changes:** Schema migration required (add to startup)  
**Testing:** Integration tests with migration

---

### Batch 3: API Contracts (High Risk)

**Goal:** Update HTTP/WS API responses.

| Task | File | Old | New | Deprecation Strategy |
|------|------|-----|-----|---------------------|
| POST /chat response | `router.go` | `"run_id": "..."` | Remove field | Return only `session_id` |
| GET /turns query | `router.go` | `run_id` param | Accept both, prefer `session_id` | Warn on old param |
| Queue responses | `send_queue.go` | `"run_id": "..."` | Remove field | |

**Deprecation Period:**
- Phase A (2 weeks): Return both `run_id` and `session_id`, log warning when `run_id` is used
- Phase B (2 weeks): Return only `session_id`, break on `run_id` query param
- Phase C: Remove all `run_id` handling

**Duration:** 4 weeks total  
**Breaking Changes:** Yes, after deprecation period  
**Testing:** API contract tests, frontend integration

---

### Batch 4: Geppetto EventMetadata (Medium Risk)

**Goal:** Remove backwards-compat `LegacyRunID`.

| Task | File | Old | New |
|------|------|-----|-----|
| Remove JSON compat | `chat-events.go` | `LegacyRunID` field | Remove entirely |
| Update log marshaling | `chat-events.go` | `e.Str("run_id", em.SessionID)` | Remove `run_id` log field |

**Deprecation Period:**
- Coordinate with external consumers
- Announce in release notes 2 releases prior

**Duration:** 1 day (after deprecation period)  
**Breaking Changes:** Yes, for JSON consumers expecting `run_id`  
**Testing:** Event serialization tests

---

### Batch 5: Documentation Cleanup (Low Risk)

**Goal:** Update all documentation to use session terminology.

| Task | File | Change |
|------|------|--------|
| API docs | `webchat-framework-guide.md` | Remove `run_id` from examples |
| Backend reference | `webchat-backend-reference.md` | Update struct examples |
| README | `web-chat/README.md` | Update query param examples |
| SEM docs | `webchat-sem-and-ui.md` | Clarify planning run vs session |
| Events docs | `04-events.md` | Remove "legacy" comment |
| Turns docs | `08-turns.md` | Keep `Run (session)` as clarification |

**Duration:** 1 day  
**Breaking Changes:** None  
**Testing:** Doc link validation

---

## Coordination with Debug UI Migration (PI-013/PI-014)

The debug UI migration already plans:
1. Migrate `/turns` and `/timeline` to `/debug/turns` and `/debug/timeline`
2. No backwards compatibility for legacy routes
3. `DistinctRuns` API for session summaries

**Alignment:**
- Batch 2 (schema migration) should complete before debug UI implementation
- Batch 3 (API contracts) should include the debug endpoints
- The `DistinctRuns` method should be named `DistinctSessions` in implementation

---

## Validation Checklist

After complete migration:

- [ ] No `run_id` in webchat/session APIs (unless marked legacy and scheduled for deletion)
- [ ] Session-only terminology in debug endpoints and UI
- [ ] No mixed `run_id` + `session_id` payloads in the same contract
- [ ] Docs match implementation
- [ ] All tests pass
- [ ] No linter warnings for `RunID` in webchat package

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| External consumers break on API change | High | 4-week deprecation period, announce in release notes |
| Schema migration fails | Medium | Test migration script in staging, backup before migration |
| Frontend breaks on missing `run_id` | Medium | Update frontend first in Batch 3 |
| Planning run confusion | Low | Clear documentation that `PlanningRun.run_id` is distinct |

---

## Delete List

After migration is complete, these symbols should be deleted:

### Backend (Go)

- `Conversation.RunID` → replaced by `SessionID`
- `TurnSnapshot.RunID` → replaced by `SessionID`
- `TurnQuery.RunID` → replaced by `SessionID`
- `ModeChange.RunID` → replaced by `SessionID`
- `LegacyRunID` in `EventMetadata` → removed
- `startRunForPrompt()` → replaced by `startInferenceForPrompt()`

### SQL Schema

- `turns.run_id` column → replaced by `session_id`
- `turns_by_run` index → replaced by `turns_by_session`
- `agent_mode_changes.run_id` column → replaced by `session_id`
- `idx_agent_mode_changes_run_id_at` index → replaced by `*_session_id_at`

### API Contracts

- `run_id` field in POST /chat response → removed
- `run_id` query parameter in GET /turns → removed

### Documentation

- All `run_id` examples → replaced by `session_id`

---

## Terms That Remain

| Term | Location | Justification |
|------|----------|---------------|
| `PlanningRun.run_id` | Proto | Planning run is a distinct domain; already namespaced |
| `EventPlanningStart.RunID` | Events | Maps to proto; follows proto naming |
| `planning map[string]*planningAgg` | Timeline projector | Key is planning run ID, not session |
| `RunLoop()` | toolloop | Function name for execution, not identifier |
| `RunInference()` | engine | Function name for execution, not identifier |
| `pkg/cmds/run` | Package | Execution context package |

---

## Rollout Order

```
Week 1:  Batch 0 (Preparation)
         Batch 1 (Internal names)
Week 2:  Batch 2 (Schema migration)
Week 3-6: Batch 3 Phase A (Deprecation period, return both)
Week 7:  Batch 3 Phase B (Break on old param)
         Batch 4 (EventMetadata)
         Batch 5 (Documentation)
Week 8:  Batch 3 Phase C (Remove all legacy handling)
         Validation
```

---

## Success Criteria

1. **Zero LEGACY_SESSION_ALIAS hits** in inventory scan
2. **All tests pass** without `run_id` expectations
3. **No runtime errors** related to missing `run_id` field
4. **Documentation accuracy** verified by manual review
5. **External consumers** successfully migrated (based on deprecation period feedback)
