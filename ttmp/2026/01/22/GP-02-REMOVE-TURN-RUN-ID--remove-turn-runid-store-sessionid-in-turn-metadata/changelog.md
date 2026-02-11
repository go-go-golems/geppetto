# Changelog

## 2026-01-22

- Initial workspace created


## 2026-01-22

Created ticket workspace; added initial analysis of replacing Turn.RunID with SessionID in Turn.Metadata; started investigation diary.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/analysis/01-replace-turn-runid-with-sessionid-in-turn-metadata.md — Design analysis and migration checklist
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/reference/01-diary.md — Investigation diary


## 2026-01-22

Updated design with NewSession() (auto-generate SessionID), StartInference must fail on empty seed turn, and TurnPersister drops explicit runID param.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/analysis/01-replace-turn-runid-with-sessionid-in-turn-metadata.md — Design updated with NewSession/StartInference/TunPersister decisions
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/reference/01-diary.md — Diary updated with new requirements
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/tasks.md — Expanded step-by-step refactor task list


## 2026-01-22

Implemented removal of turns.Turn.RunID: added KeyTurnMetaSessionID, NewSession(), StartInference empty-seed failure, updated TurnPersister signature, refactored engines/middleware/tests/examples; geppetto tests pass with GOCACHE override.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/doc/topics/08-turns.md — Updated docs to remove run_id field
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/session/session.go — NewSession + StartInference seed validation + Append metadata
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/session/tool_loop_builder.go — TurnPersister signature + metadata injection
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/turns/keys.go — Added KeyTurnMetaSessionID
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/turns/types.go — Removed Turn.RunID


## 2026-01-22

Implement SessionID-in-metadata refactor (geppetto commit 4b5fe38)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/session/session.go — NewSession() + Append() set session id metadata
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/turns/types.go — Remove Turn.RunID field


## 2026-01-22

Moments backend: rename RunID to SessionID and wire SessionID/InferenceID/TurnID consistently (moments commit 6bb64356)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/moments/backend/pkg/webchat/conversation.go — Conversation + ConvManager rename


## 2026-01-22

Finish remaining runID→sessionID signature cleanup in moments/backend; audit moments/web run_id usage and update cross-ticket docs to reference KeyTurnMetaSessionID.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/analysis/03-moments-web-run-id-usage-audit.md — Inventory and migration guidance for run_id usage
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/moments/backend/pkg/artifact/buffer.go — Rename runID param to sessionID; log both run_id+session_id


## 2026-01-22

Verified complete; closing ticket (tests passed in geppetto/pinocchio/moments/backend on 2026-01-23).

