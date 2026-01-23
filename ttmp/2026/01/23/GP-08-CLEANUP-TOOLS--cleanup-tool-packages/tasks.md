# Tasks

## Completed (Bookkeeping)

- [x] Create GP-08 ticket workspace (index + tasks + analysis)
- [x] Write initial tool* package inventory + reorg report
- [x] Incorporate decisions: no compat, `tools.ToolConfig` canonical, move `toolcontext`→`tools`, move `toolblocks`→`turns`, builder in `toolloop/enginebuilder`
- [x] Upload GP-08 bundle to reMarkable (and overwrite after updates)

## Execution Plan (Step-by-step)

### Step 1 — Move the session engine builder into `toolloop/enginebuilder`

- [x] Create package: `geppetto/pkg/inference/toolloop/enginebuilder`
- [x] Move `toolloop.EngineBuilder` and builder options into `toolloop/enginebuilder`
  - Target API:
    - `enginebuilder.New(...) *enginebuilder.Builder` (or `NewEngineBuilder`, pick one)
    - `enginebuilder.WithBase(...)`, `enginebuilder.WithMiddlewares(...)`, `enginebuilder.WithToolRegistry(...)`, `enginebuilder.WithToolConfig(...)`, `enginebuilder.WithEventSinks(...)`, `enginebuilder.WithSnapshotHook(...)`, `enginebuilder.WithStepController(...)`, `enginebuilder.WithStepPauseTimeout(...)`, `enginebuilder.WithPersister(...)`
  - Ensure it still satisfies `session.EngineBuilder`
- [x] Update all Geppetto call sites (examples + docs snippets where applicable)
- [x] Update Pinocchio/Moments call sites to the new import path (no wrapper helpers)
- [x] Update/relocate the builder unit tests (avoid import cycles; keep tests in `toolloop/enginebuilder`)

### Step 2 — Make `tools.ToolConfig` canonical (and rename loop config)

- [x] Introduce `toolloop.LoopConfig` (or similar) and remove/rename `toolloop.ToolConfig`
  - Loop config should only cover loop orchestration concerns (e.g. `MaxIterations`)
  - Tool execution/advertisement policy must come from `tools.ToolConfig` (canonical)
- [x] Update `toolloop.Loop` to accept:
  - loop config: `LoopConfig`
  - tool config: `tools.ToolConfig` (or pointer) and set `engine.KeyToolConfig` accordingly
- [x] Decide the authoritative “Turn.Data tool config” type:
  - Preferred: store an `engine.ToolConfig` derived from `tools.ToolConfig` (provider-facing shape stays in engine)
  - Ensure there is exactly one mapping function (single source of truth)
- [x] Update provider engines if they read/assume fields that diverge

### Step 3 — Move registry-in-context from `toolcontext` into `tools`

- [x] Add `tools.WithRegistry(ctx, reg)` and `tools.RegistryFrom(ctx)` (names TBD, but keep short)
- [x] Update providers to import from `tools` (OpenAI/Claude/Gemini/OpenAI-Responses)
- [x] Update tool loop orchestration to use `tools.WithRegistry(...)`
- [x] Delete `geppetto/pkg/inference/toolcontext`
- [x] Run `go test ./...` in `geppetto`

### Step 4 — Move Turn block helpers from `toolblocks` into `turns`

- [x] Create `geppetto/pkg/turns/toolblocks` (preferred) or `geppetto/pkg/turns/tools` (pick one)
- [x] Move:
  - `ExtractPendingToolCalls`
  - `AppendToolResultsBlocks`
  - associated structs
- [x] Update imports in `toolloop` and any remaining internal users
- [x] Delete `geppetto/pkg/inference/toolblocks`
- [ ] Consider improving result block shape (optional follow-up task):
  - avoid `"Error: ..."` string payloads; use typed payload fields if the block model supports it

### Step 5 — Delete `toolhelpers` (hard cutover; no wrappers)

- [x] Update all internal Geppetto call sites (should be none, verify)
- [x] Update downstream repos:
  - `pinocchio`: replace any remaining `toolhelpers` usage with `toolloop.Loop` or `toolloop/enginebuilder` + `tools.ToolExecutor`
  - `moments`: same
  - `go-go-mento`: same
- [x] Delete `geppetto/pkg/inference/toolhelpers`
- [x] Ensure docs/playbooks do not recommend `toolhelpers` anymore (canonical path only)

### Step 6 — Docs + guidance

- [x] Update Geppetto docs to only present canonical surfaces:
  - tool loop: `toolloop.Loop`
  - builder: `toolloop/enginebuilder`
  - primitives: `tools.ToolRegistry`, `tools.ToolExecutor`, `tools.ToolConfig`
- [x] Add a “How to wire tools end-to-end” section with the new API (short, copy/pasteable)
- [x] Update Pinocchio docs snippets (webchat guide, README snippets) to match new paths

### Step 7 — Validation and rollout

- [x] Run tests:
  - `cd geppetto && go test ./... -count=1`
  - `cd pinocchio && go test ./... -count=1`
  - `cd moments/backend && go test ./... -count=1`
  - `cd go-go-mento/go && go test ./... -count=1` (or module-specific)
- [x] Run linters/hooks where applicable (expect lefthook behavior)
- [x] Commit in each repo with clean message grouping (geppetto/pinocchio/moments/go-go-mento)
- [x] Upload updated GP-08 docs to reMarkable (bundle)
