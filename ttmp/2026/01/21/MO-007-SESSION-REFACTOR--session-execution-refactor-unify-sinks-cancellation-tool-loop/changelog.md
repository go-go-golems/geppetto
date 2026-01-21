# Changelog

## 2026-01-21

- Initial workspace created


## 2026-01-21

Wrote MO-007 design doc defining Session(SessionID)+EngineBuilder+ExecutionHandle and a no-compat step-by-step migration plan (supersedes MO-005/006).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/21/MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/design-doc/01-session-refactor-sessionid-enginebuilder-executionhandle.md — Primary design doc


## 2026-01-21

Step 1: introduce pkg/inference/session (Session + ExecutionHandle) and ToolLoopEngineBuilder; add unit tests; enforce gofmt via pre-commit (commit 158e4be)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/session/session.go — Async StartInference + single-active invariant
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/session/tool_loop_builder.go — Orchestration moved to builder/runner

