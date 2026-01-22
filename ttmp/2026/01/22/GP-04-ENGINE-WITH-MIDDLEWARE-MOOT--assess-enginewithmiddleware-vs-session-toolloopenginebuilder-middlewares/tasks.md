# Tasks

## TODO

- [ ] Inventory current `NewEngineWithMiddleware` call sites (geppetto + downstream repos)
- [ ] Clarify what “moot” means: internal helper vs public API recommendation
- [ ] Decide target composition API for apps:
  - [ ] Prefer `session.ToolLoopEngineBuilder{Middlewares: ...}` for chat-style apps
  - [ ] Keep `middleware.NewEngineWithMiddleware` for non-session use cases (or deprecate)
- [ ] Update docs/examples to use builder-provided middlewares (where applicable)
- [ ] Consider deprecation strategy for direct wrapper usage (notes + timeline)
- [ ] Upload ticket docs to reMarkable
