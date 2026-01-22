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

