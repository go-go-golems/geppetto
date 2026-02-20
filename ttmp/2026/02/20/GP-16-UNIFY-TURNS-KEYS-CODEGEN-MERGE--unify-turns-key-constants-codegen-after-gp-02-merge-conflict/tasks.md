# Tasks

## TODO

- [x] Phase 1 - Authoritative spec and generator core
- [x] 1.1 Create `pkg/spec/geppetto_codegen.yaml` as the single manifest for block kinds, key families, JS enums, and JS const export groups
- [x] 1.2 Implement schema parsing and validation in `cmd/gen-meta` (required fields, duplicate detection, owner validation, source references)
- [x] 1.3 Implement a normalized internal render model (`turnsKeysRenderData`, `turnsDTSRenderData`, `engineTurnkeysRenderData`, `jsConstsRenderData`)
- [x] 1.4 Implement sectioned generation CLI surface (`--section all|turns-go|engine-go|turns-dts|js-go|js-dts`)
- [x] 1.5 Add focused unit tests in `cmd/gen-meta/main_test.go` for schema validation, enum/group expansion, and JS key normalization
- [x] Phase 2 - Generate all targets from one source
- [x] 2.1 Generate turns Go block kinds to `pkg/turns/block_kind_gen.go`
- [x] 2.2 Generate turns value constants and turns-owned typed keys to `pkg/turns/keys_gen.go`
- [x] 2.3 Generate engine-owned typed keys to `pkg/inference/engine/turnkeys_gen.go`
- [x] 2.4 Generate turns TypeScript constants to `pkg/doc/types/turns.d.ts` via `pkg/turns/spec/turns.d.ts.tmpl`
- [x] 2.5 Generate geppetto JS const installer to `pkg/js/modules/geppetto/consts_gen.go`
- [x] 2.6 Generate geppetto TypeScript const section to `pkg/doc/types/geppetto.d.ts` via `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
- [x] Phase 3 - Wire and migrate call sites
- [x] 3.1 Rewire `go:generate` entrypoints in turns/js/engine packages to call `cmd/gen-meta`
- [x] 3.2 Remove manual duplicate key surfaces now owned by generation (`pkg/turns/keys.go`, `pkg/inference/engine/turnkeys.go`)
- [x] 3.3 Update lint policy allowlist in `pkg/analysis/turnsdatalint/analyzer.go` for generated key-definition files
- [x] 3.4 Remove legacy generators and split manifests (`cmd/gen-turns`, `cmd/gen-js-api`, `pkg/turns/spec/turns_codegen.yaml`, `pkg/js/modules/geppetto/spec/js_api_codegen.yaml`)
- [x] Phase 4 - Reconcile behavior and docs
- [x] 4.1 Update tests and runtime expectations to new const groups (`TurnMetadataKeys`, `BlockMetadataKeys`, `RunMetadataKeys`, `PayloadKeys`)
- [x] 4.2 Update docs to reference the unified generator/schema and new const group naming
- [x] 4.3 Regenerate all generated outputs and verify no stale generated headers remain
- [x] Phase 5 - Validation and ticket bookkeeping
- [x] 5.1 Run focused verification (`go test ./cmd/gen-meta`, key package tests, generator invocations)
- [x] 5.2 Run full repository verification (`go test ./...`)
- [x] 5.3 Update `reference/01-diary.md` with implementation narrative, commands, failures, and review instructions
- [x] 5.4 Update GP-16 changelog and mark completed tasks
- [x] 5.5 Run `docmgr doctor --ticket GP-16-UNIFY-TURNS-KEYS-CODEGEN-MERGE` and resolve/report findings

## Done

- [x] Resolve `pkg/turns/keys.go` merge conflict by keeping payload/run-only manual constants
- [x] Add missing inference value keys to `pkg/turns/spec/turns_codegen.yaml`
- [x] Regenerate turns artifacts (`keys_gen.go`, `turns.d.ts`) via `go generate ./pkg/turns`
- [x] Validate compile/tests for turns + inference/engine + provider helpers
- [x] Extend `cmd/gen-js-api` with `--turns-schema` imports for turns-domain const groups
- [x] Generate `TurnDataKeys`, `TurnMetadataKeys`, and `BlockMetadataKeys` in `gp.consts`
- [x] Regenerate geppetto constants outputs and update module/generator tests
- [x] Update JS API reference docs for the new key groups
- [x] Add a short contributor note documenting schema/codegen as the single source of truth for turns value constants
