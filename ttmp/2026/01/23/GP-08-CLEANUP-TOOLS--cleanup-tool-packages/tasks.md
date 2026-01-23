# Tasks

## TODO

- [ ] Confirm desired end-state package layout (minimum # of packages, naming, compatibility expectations)
- [ ] Deprecation strategy for `geppetto/pkg/inference/toolhelpers` (wrapper vs deletion; external consumers)
- [ ] Decide canonical tool config type(s) and remove duplication across:
  - `engine.ToolConfig`
  - `tools.ToolConfig`
  - `toolloop.ToolConfig`
  - `toolhelpers.ToolConfig`
- [ ] Decide whether `toolcontext` becomes:
  - part of `tools` (preferred), or
  - stays as its own package but gets renamed (e.g. `toolsctx`)
- [ ] Decide whether `toolblocks` becomes:
  - part of `turns` (preferred), or
  - stays as its own package but gets renamed (e.g. `turntools`)
- [ ] Update docs to only present the canonical surfaces (`toolloop.Loop`, `toolloop.EngineBuilder`, `tools.ToolRegistry`, `tools.ToolExecutor`)
- [ ] Add a “compatibility + migration” doc for downstream repos (go-go-mento, moments, pinocchio) and provide mechanical rewrite guidance
- [ ] Add explicit deprecation markers (`// Deprecated:`) and migration notes in GoDoc
- [ ] Optionally: add a minimal `go vet`/linter rule or `rg` check to prevent new uses of deprecated packages
