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


## 2026-01-21

Step 5: delete engine.WithSink / engine.Option plumbing; update provider constructors + engine factory to use context sinks only (commits d6a0f54, 6ce03ff)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/engine/factory/factory.go — Signature change; no engine options


## 2026-01-21

Step 6: run real-world inference (geppetto examples + pinocchio TUI tmux) and fix pinocchio chat autostart hang (commit da5f276)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/cmds/cmd.go — Disable bobatea autoStartBackend


## 2026-01-21

Postmortem: document MO-007 implementation steps, tricky parts, and WithAutoStartBackend footgun

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/21/MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/analysis/01-postmortem-session-refactor-mo-007.md — Postmortem


## 2026-01-21

Analysis: document bobatea StartBackendMsg no-op and pinocchio TUI integration path with Session API

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/21/MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/analysis/02-pinocchio-tui-bobatea-integrating-session-api-and-fixing-autostart.md — Analysis


## 2026-01-21

Step 7: delete bobatea WithAutoStartBackend/StartBackendMsg/startBackend and remove pinocchio call site (commits c2a08dc, 930b461)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/bobatea/pkg/chat/model.go — Removed AutoStartBackend pipeline


## 2026-02-25

Ticket closed

