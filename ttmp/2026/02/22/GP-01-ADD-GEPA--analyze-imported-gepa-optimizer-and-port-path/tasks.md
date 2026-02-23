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
- [x] Port optimizer plugin contract support into local geppetto plugin module (`defineOptimizerPlugin`, `OPTIMIZER_PLUGIN_API_VERSION`).
- [x] Register `geppetto/plugins` optimizer helper exports in local module registration path.
- [x] Add/extend JS module tests to validate optimizer descriptor contract checks (`apiVersion`, `kind`, `id`, `name`, `create`).
- [x] Port and adapt `toy_math_optimizer.js` example into local repo as reference script for optimizer plugin authors.
- [x] Add/update documentation page(s) showing optimizer plugin contract and example usage.

## Track B: Medium-Value Refit (`9.2`)

- [x] Create local `cmd/gepa-runner` command skeleton in `geppetto/cmd/gepa-runner` (do not copy imported code blindly).
- [x] Re-implement runtime bootstrap using current local APIs (`goja` event loop + geppetto module registration) and verify type signatures.
- [x] Re-implement optimize command flags using current glazed field types (replace invalid imported `fields.TypeInt` usage).
- [x] Re-implement eval command for one-shot prompt benchmarking over dataset.
- [x] Rebuild plugin loader for optimizer descriptors and evaluator invocation (`dataset()` optional, `evaluate()` required).
- [x] Rebuild evaluator result decode path with no duplicate map type-switch cases.
- [x] Implement dataset loader (`.json`, `.jsonl`) with clear parse errors and line context for JSONL failures.
- [x] Wire profile and provider option bridging (`resolvePinocchioProfile`, `applyProfileEnvironment`, `resolveEngineOptions`) against current local sections.
- [x] Add optimize/eval smoke tests (or scripted checks) that run with deterministic fake engine/evaluator where possible.
- [x] Add command help/readme for `cmd/gepa-runner` including contract and sample commands.
- [x] Ensure `go build ./cmd/gepa-runner` passes in local `geppetto/`.

## Track C: Minimal Viable Integration Gates (`10.1`)

- [x] Integrate GEPA package and runner without changing unrelated OpenAI responses or JS builder hook behavior in same PR.
- [x] Keep output format stable: best prompt output plus JSON report file from optimize/eval commands.
- [x] Verify end-to-end local run path with sample script (`optimize` then `eval`) and capture artifacts in ticket `sources/`.
- [x] Document known limitations for MVP (single primary param mutation, no persistent benchmark DB, no crossover/merge).
- [x] Add CI or local checklist commands to ticket docs:
- [x] `go test ./pkg/optimizer/gepa`
- [x] `go test ./pkg/js/modules/geppetto`
- [x] `go build ./cmd/gepa-runner`
- [x] Mark MVP done only when all gates above pass and artifacts are linked in ticket changelog.

## Ticket Hygiene / Delivery

- [x] Update `changelog.md` after each implementation chunk with precise file-level notes.
- [x] Keep `analysis` doc synced with any architecture deviations discovered during implementation.
- [x] Upload Phase 1 implementation summary to reMarkable after code is merged/ready for review.
