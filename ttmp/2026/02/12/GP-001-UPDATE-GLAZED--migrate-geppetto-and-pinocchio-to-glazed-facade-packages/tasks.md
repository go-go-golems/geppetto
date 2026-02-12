# Tasks

## Completed

- [x] Create docmgr ticket `GP-001-UPDATE-GLAZED`
- [x] Capture baseline failures with `make test` and `make lint` in both repos
- [x] Produce exhaustive migration analysis and inventory artifacts
- [x] Create and maintain detailed implementation diary

## Automation

- [x] Build and run an automated `go/ast` + `gopls` migration analyzer from `scripts/`, and store generated reports under `sources/local/`

## Geppetto Migration (Phase 1)

- [x] Migrate `geppetto/pkg/layers/layers.go` from legacy `layers/parameters/middlewares` to `schema/fields/sources/values`
- [x] Migrate geppetto settings section constructors and defaults init (`pkg/steps/ai/settings/*`, `pkg/embeddings/config/settings.go`)
- [x] Migrate geppetto runtime decode helpers to `values.DecodeSectionInto` (`pkg/steps/ai/settings/settings-step.go`, `pkg/embeddings/settings_factory.go`, `pkg/inference/engine/factory/helpers.go`)
- [x] Migrate geppetto commands/examples signatures and schema wiring (`cmd/examples/*`, `cmd/llm-runner/*`)
- [x] Update geppetto docs snippets to new APIs (`pkg/doc/topics/06-embeddings.md`, `pkg/doc/topics/06-inference-engines.md`)
- [x] Validate geppetto with `make test` and `make lint`

## Pinocchio Migration (Phase 2)

- [x] Migrate pinocchio command model/core loader to `schema/fields/sources/values` (`pkg/cmds/*`)
- [x] Migrate pinocchio command implementations in `cmd/pinocchio/cmds/*`
- [x] Migrate webchat + redis settings paths (`pkg/webchat/*`, `pkg/redisstream/redis_layer.go`, `cmd/web-chat/main.go`)
- [ ] Migrate examples and agent commands (`cmd/examples/*`, `cmd/agents/*`)
- [ ] Validate pinocchio with `make test` and `make lint`

## Follow-up / Out of Scope for This Ticket

- [ ] Separate ticket for missing geppetto package imports in pinocchio (`toolhelpers`, `toolcontext`, `conversation`)
- [ ] Separate ticket for `RunID` incompatibility in `pinocchio/pkg/middlewares/agentmode/middleware.go`
- [ ] Separate ticket for `clay`/`glazed` logging bootstrap compatibility (`logging.AddLoggingLayerToRootCommand` missing)
- [ ] Separate ticket for `prompto` module compatibility with facade packages (`github.com/go-go-golems/prompto@v0.1.13` still imports `cmds/layers` and `cmds/parameters`)
