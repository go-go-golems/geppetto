# Tasks

## Phase 1 Scope

Implement analysis sections `9.1`, `9.2`, and `10.1` as the first shippable GEPA integration in local `geppetto/`.

## Track A: High-Value Ports (`9.1`)

- [x] Port `pkg/optimizer/gepa/*` from `imported/geppetto-main` into `geppetto/pkg/optimizer/gepa`.
- [x] Add package-level tests for `pkg/optimizer/gepa` (config defaults, cache reuse, aggregate stats, Pareto front behavior).
- [x] Fix reflection fenced-block parsing in `reflector.go` so plain fenced prompt text does not lose the first word.
- [x] Add regression tests for fenced parsing (` ```text` and ` ```\\ncontent\\n``` ` forms).
- [x] Add no-progress/stagnation guard in optimizer loop (break or fallback when no new evals are consumed and no accepted child is produced).
- [x] Add regression test that proves loop exits cleanly in repeated-child / cache-hit scenarios.
- [x] Ensure `pkg/optimizer/gepa` compiles and tests pass under local workspace (`go test ./pkg/optimizer/gepa`).
- [ ] Port optimizer plugin contract support into local geppetto plugin module (`defineOptimizerPlugin`, `OPTIMIZER_PLUGIN_API_VERSION`).
- [ ] Register `geppetto/plugins` optimizer helper exports in local module registration path.
- [ ] Add/extend JS module tests to validate optimizer descriptor contract checks (`apiVersion`, `kind`, `id`, `name`, `create`).
- [ ] Port and adapt `toy_math_optimizer.js` example into local repo as reference script for optimizer plugin authors.
- [ ] Add/update documentation page(s) showing optimizer plugin contract and example usage.

## Track B: Medium-Value Refit (`9.2`)

- [ ] Create local `cmd/gepa-runner` command skeleton in `geppetto/cmd/gepa-runner` (do not copy imported code blindly).
- [ ] Re-implement runtime bootstrap using current local APIs (`goja` event loop + geppetto module registration) and verify type signatures.
- [ ] Re-implement optimize command flags using current glazed field types (replace invalid imported `fields.TypeInt` usage).
- [ ] Re-implement eval command for one-shot prompt benchmarking over dataset.
- [ ] Rebuild plugin loader for optimizer descriptors and evaluator invocation (`dataset()` optional, `evaluate()` required).
- [ ] Rebuild evaluator result decode path with no duplicate map type-switch cases.
- [ ] Implement dataset loader (`.json`, `.jsonl`) with clear parse errors and line context for JSONL failures.
- [ ] Wire profile and provider option bridging (`resolvePinocchioProfile`, `applyProfileEnvironment`, `resolveEngineOptions`) against current local sections.
- [ ] Add optimize/eval smoke tests (or scripted checks) that run with deterministic fake engine/evaluator where possible.
- [ ] Add command help/readme for `cmd/gepa-runner` including contract and sample commands.
- [ ] Ensure `go build ./cmd/gepa-runner` passes in local `geppetto/`.

## Track C: Minimal Viable Integration Gates (`10.1`)

- [ ] Integrate GEPA package and runner without changing unrelated OpenAI responses or JS builder hook behavior in same PR.
- [ ] Keep output format stable: best prompt output plus JSON report file from optimize/eval commands.
- [ ] Verify end-to-end local run path with sample script (`optimize` then `eval`) and capture artifacts in ticket `sources/`.
- [ ] Document known limitations for MVP (single primary param mutation, no persistent benchmark DB, no crossover/merge).
- [ ] Add CI or local checklist commands to ticket docs:
- [ ] `go test ./pkg/optimizer/gepa`
- [ ] `go test ./pkg/js/modules/geppetto`
- [ ] `go build ./cmd/gepa-runner`
- [ ] Mark MVP done only when all gates above pass and artifacts are linked in ticket changelog.

## Ticket Hygiene / Delivery

- [ ] Update `changelog.md` after each implementation chunk with precise file-level notes.
- [ ] Keep `analysis` doc synced with any architecture deviations discovered during implementation.
- [ ] Upload Phase 1 implementation summary to reMarkable after code is merged/ready for review.
