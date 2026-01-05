# Tasks

## TODO

- [x] Bootstrap ticket workspace (docs + initial tasks)

### `turns` production API (no backwards compatibility)

- [x] Implement production key families: `DataKey[T]`, `TurnMetaKey[T]`, `BlockMetaKey[T]` (store-specific ids)
- [x] Implement key receiver methods: `Get(store)` / `Set(&store, value)` for all three key families
- [x] Add store-specific constructors: `DataK/TurnMetaK/BlockMetaK` (replace `turns.K`)
- [x] Decide and implement shared key-id constructor shape (`NewKeyString` vs keep `NewTurnDataKey` + casts) and update callers

### Canonical key definitions (must type-check downstream)

- [x] Migrate geppetto canonical keys in `geppetto/pkg/turns/keys.go` to `DataK/TurnMetaK/BlockMetaK`
- [x] Migrate engine escape-hatch key: `geppetto/pkg/inference/engine/turnkeys.go` `KeyToolConfig` → `turns.DataK`
- [x] Migrate any tests/examples still using `turns.K[...]` (e.g. `geppetto/pkg/turns/serde/serde_test.go`)

### Call-site migration (mechanical rewrite, then fix compile)

- [x] Run `turnsrefactor` (dry-run) on `geppetto` and review diff size/safety
- [x] Run `turnsrefactor` (write) on `geppetto` and fix any compile errors
- [x] Run `turnsrefactor` (write) on `moments/backend` and fix any compile errors
- [x] Run `turnsrefactor` (write) on `pinocchio` and fix any compile errors

### Key constructor migration (mechanical rewrite)

- [x] Migrate constructors in `moments/backend/pkg/turnkeys/*` (`turns.K` → correct store-specific constructor)
- [x] Migrate constructors in pinocchio key files (`turns.K` → correct store-specific constructor)
- [x] Decide whether to extend `turnsrefactor` to rewrite constructors (`turns.K` → `turns.{Data,TurnMeta,BlockMeta}K`) and implement if worthwhile

### Delete old API (hard cut)

- [x] Delete legacy turns API: `Key[T]`, `K[T]`, and `DataGet/DataSet/MetadataGet/MetadataSet/BlockMetadataGet/BlockMetadataSet`
- [ ] Remove/adjust any remaining references in code + docs (no shims)
- [ ] Update/retire `turnsrefactor` verification mode once old API is removed (it currently verifies no `*.DataGet/...` remain)

### Lint + docs (keep policy aligned)

- [ ] Update `turnsdatalint` to enforce canonical constructor usage (ban ad-hoc `DataK/TurnMetaK/BlockMetaK` outside key-definition files)
- [ ] Add/extend `turnsdatalint` tests for constructor policy + key-family map indexing
- [ ] Update `geppetto/pkg/doc/topics/08-turns.md` to reflect the new production API and remove references to deleted helpers/APIs

### Validation gates (must pass before declaring “done”)

- [x] `cd geppetto && go test ./... -count=1`
- [x] `cd moments/backend && go test ./... -count=1`
- [x] `cd pinocchio && go test ./... -count=1`
