# Tasks

## TODO

- [ ] Remove `middleware.NewEngineWithMiddleware` (and `EngineWithMiddleware`) entirely
- [ ] Fold engine-level middleware wrapping into `session.ToolLoopEngineBuilder` implementation
- [ ] Add `session.NewToolLoopEngineBuilder(...WithOption)` constructor + option helpers
- [ ] Update all `NewEngineWithMiddleware` usages to:
  - [ ] supply `Middlewares` via `ToolLoopEngineBuilder`, or
  - [ ] wrap at the session boundary and return an `InferenceRunner` that also satisfies `engine.Engine`
- [ ] Migrate Geppetto examples to use `session.NewToolLoopEngineBuilder(...)`
- [ ] Update docs/tutorials to reflect the new builder-first middleware composition pattern
- [ ] Run `go test ./...` in `geppetto/`, `pinocchio/`, and `moments/backend/`
- [ ] Upload updated ticket docs to reMarkable
