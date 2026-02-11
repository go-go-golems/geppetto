---
Title: Run to Session Open Questions
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
    - Path: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/07/PI-017-RUN-SESSION-INVESTIGATION--run-to-session-elimination-research/analysis/01-run-to-session-elimination-plan.md
      Note: Migration plan with proposed decisions
ExternalSources: []
Summary: Open questions requiring decisions before run→session migration.
LastUpdated: 2026-02-07T09:15:00-05:00
WhatFor: Track unresolved decisions blocking migration.
WhenToUse: Review before starting migration batches.
---

# Run to Session Open Questions

## Goal

Document questions that require stakeholder input before proceeding with migration.

## Open Questions

### Q1: Should `PlanningRun.run_id` be renamed to `planning_id`?

**Context:**  
The planning domain uses `run_id` within `PlanningRun` messages. This is conceptually distinct from session ID, but the naming could still cause confusion.

**Options:**

| Option | Pros | Cons |
|--------|------|------|
| Keep `PlanningRun.run_id` | No proto breaking change; already namespaced by message type | Potential confusion in logs/debugging |
| Rename to `planning_id` | Clearer separation of concerns | Proto breaking change; requires regenerating all generated code |
| Rename to `planning_run_id` | Explicit about domain | Still has "run" in name; longest option |

**Recommendation:** Keep `PlanningRun.run_id` — the namespacing is sufficient and avoids proto churn.

**Status:** Proposed decision in plan, awaiting confirmation.

---

### Q2: How long should the API deprecation period be?

**Context:**  
Batch 3 proposes a 4-week deprecation period for the `/chat` API `run_id` field.

**Options:**

| Option | Pros | Cons |
|--------|------|------|
| 2 weeks | Faster migration | May not give external consumers enough time |
| 4 weeks | Standard deprecation window | Longer maintenance burden |
| 6 weeks | Very conservative | Delays cleanup |
| Immediate | No legacy code | Breaking change for all consumers |

**Recommendation:** 4 weeks, with announcement 2 releases prior.

**Status:** Proposed decision in plan, awaiting confirmation.

---

### Q3: Should debug UI use `session_id` or `run_id` for its new endpoints?

**Context:**  
PI-013/PI-014 proposes new `/debug/turns` and `/debug/timeline` endpoints. The current plan references `run_id` in some places.

**Coordination Points:**
- `DistinctRuns` method name in TurnStore — should be `DistinctSessions`
- `/debug/conversation/:id/runs` endpoint — should be `/debug/conversation/:id/sessions`
- Query parameter `run_id` — should be `session_id`

**Recommendation:** Debug UI should use session terminology exclusively.

**Status:** Requires update to PI-013/PI-014 design docs.

---

### Q4: Should `LegacyRunID` backwards compatibility in EventMetadata have a deprecation period?

**Context:**  
`geppetto/pkg/events/chat-events.go` has a `LegacyRunID` field that maps `run_id` JSON key to `SessionID` during unmarshaling.

**Impact:**  
Any external system sending events with `"run_id"` in JSON will break if this is removed without warning.

**Options:**

| Option | Pros | Cons |
|--------|------|------|
| Remove immediately | Clean codebase | Breaks external consumers |
| 2-release deprecation | Gives warning time | Must maintain compat code |
| Keep indefinitely | Never breaks anyone | Perpetuates confusion |

**Recommendation:** 2-release deprecation with release note warning.

**Status:** Awaiting decision.

---

### Q5: How should schema migration be deployed?

**Context:**  
Batch 2 requires renaming `run_id` columns to `session_id` in SQLite databases.

**Options:**

| Option | Pros | Cons |
|--------|------|------|
| Automatic on startup | Seamless for users | Risk of data loss if migration fails |
| Manual migration script | User controls timing | Extra step for deployment |
| Dual-write period | Zero downtime | Complex implementation |

**Recommendation:** Automatic on startup with backup prompt in release notes.

**Status:** Awaiting decision.

---

### Q6: Should the simple-chat-agent UI status bar show "session:" or "run:"?

**Context:**  
`pinocchio/cmd/agents/simple-chat-agent/pkg/ui/app.go:393` displays `run:` in the status bar.

**Options:**

| Option | Impact |
|--------|--------|
| Change to `session:` | Consistent with new terminology |
| Change to `sess:` | Shorter, fits status bar better |
| Keep `run:` | Minimal UI change |

**Recommendation:** Change to `session:` for consistency.

**Status:** Low priority, awaiting decision.

---

### Q7: Should AgentMode's seed data use realistic session IDs?

**Context:**  
`pinocchio/pkg/middlewares/agentmode/seed.sql:16-18` uses `run-1` and `run-2` as test session IDs.

**Options:**

| Option | Impact |
|--------|--------|
| Change to `session-1`, `session-2` | Consistent naming |
| Change to UUIDs | More realistic |
| Keep as-is | Minimal change, test data only |

**Recommendation:** Change to `session-1`, `session-2` for consistency.

**Status:** Low priority, part of Batch 2.

---

## Resolved Questions

(None yet — questions will move here as decisions are made.)

---

## Decision Log Template

When a question is resolved, document:

```markdown
### QN: [Question Title]

**Decision:** [Chosen option]
**Rationale:** [Why this option was chosen]
**Date:** [YYYY-MM-DD]
**Decided by:** [Stakeholder(s)]
**Implementation:** [Batch N / specific PR]
```
