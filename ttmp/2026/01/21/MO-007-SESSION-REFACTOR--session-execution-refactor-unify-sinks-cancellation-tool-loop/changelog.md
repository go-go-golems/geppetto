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


## 2026-01-21

Step 2: migrate geppetto cmd/examples off core.Session/InferenceState to session.Session + ToolLoopEngineBuilder (commit 5cd95af)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/simple-inference/main.go — Uses Session.StartInference().Wait()
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/simple-streaming-inference/main.go — Streaming example uses ToolLoopEngineBuilder.EventSinks


## 2026-01-21

Step 3: migrate pinocchio TUI off InferenceState/core.Session to geppetto session.Session + ExecutionHandle; stop using engine.WithSink for TUI engine creation (geppetto 388e976, pinocchio 0c6041a)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/session/session.go — Add IsRunning() for UIs
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/ui/backend.go — TUI backend now drives inference via session.Session
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/ui/runtime/builder.go — No engine.WithSink; pass sink to backend


## 2026-01-21

Step 4: migrate pinocchio webchat off InferenceState/core.Session to session.Session + ToolLoopEngineBuilder (pinocchio d3c0684)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/conversation.go — Conversation state now stores *session.Session
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — /chat now drives Session.StartInference + Wait logging

