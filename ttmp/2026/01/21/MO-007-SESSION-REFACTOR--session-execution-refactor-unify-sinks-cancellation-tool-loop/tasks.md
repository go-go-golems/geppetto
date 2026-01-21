# Tasks

## TODO

- [ ] Finalize design: Session/EngineBuilder/ExecutionHandle APIs + invariants
- [ ] Implement new `pkg/inference/session` and migrate geppetto callers (no compatibility)
- [ ] Remove `engine.WithSink` and legacy sink plumbing (use context sinks)
- [ ] Remove old lifecycle APIs (`pkg/inference/state`, `pkg/inference/core.Session`)
- [ ] Migrate pinocchio TUI + webchat to Session/ExecutionHandle + ToolLoopEngineBuilder
- [ ] Migrate geppetto/pinocchio examples to Session/ExecutionHandle
- [ ] Run real-world tests per MO-004 playbook (OpenAI Responses + tools + TUI)
- [ ] Update MO-007 design doc to reflect the final API surface
- [x] Implement ToolLoopEngineBuilder using base engine.Engine + middleware.Middleware (no EngineFactoryLike)
- [x] Add unit tests for new Session/ExecutionHandle lifecycle (cancel+wait) and ToolLoopEngineBuilder
