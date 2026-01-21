# Tasks

## TODO

- [x] Finalize design: Session/EngineBuilder/ExecutionHandle APIs + invariants
- [x] Implement new `pkg/inference/session` and migrate geppetto callers (no compatibility)
- [x] Remove `engine.WithSink` and legacy sink plumbing (use context sinks)
- [x] Remove old lifecycle APIs (`pkg/inference/state`, `pkg/inference/core.Session`)
- [x] Migrate pinocchio TUI + webchat to Session/ExecutionHandle + ToolLoopEngineBuilder
- [x] Migrate geppetto/pinocchio examples to Session/ExecutionHandle
- [x] Run real-world tests per MO-004 playbook (OpenAI Responses + tools + TUI)
- [x] Update MO-007 design doc to reflect the final API surface
- [x] Implement ToolLoopEngineBuilder using base engine.Engine + middleware.Middleware (no EngineFactoryLike)
- [x] Add unit tests for new Session/ExecutionHandle lifecycle (cancel+wait) and ToolLoopEngineBuilder
