# Tasks

## TODO

- [x] Remove `middleware.NewEngineWithMiddleware` (and `EngineWithMiddleware`) entirely
- [x] Fold engine-level middleware wrapping into `session.ToolLoopEngineBuilder` implementation
- [x] Add `session.NewToolLoopEngineBuilder(...WithOption)` constructor + option helpers
- [x] Update all `NewEngineWithMiddleware` usages to:
- [x] supply `Middlewares` via `ToolLoopEngineBuilder`, or
- [x] wrap at the session boundary and return an `InferenceRunner` that also satisfies `engine.Engine`
- [x] Migrate Geppetto examples to use `session.NewToolLoopEngineBuilder(...)`
- [x] Update docs/tutorials to reflect the new builder-first middleware composition pattern
- [x] Run `go test ./...` in `geppetto/`, `pinocchio/`, and `moments/backend/`
- [ ] Upload updated ticket docs to reMarkable
