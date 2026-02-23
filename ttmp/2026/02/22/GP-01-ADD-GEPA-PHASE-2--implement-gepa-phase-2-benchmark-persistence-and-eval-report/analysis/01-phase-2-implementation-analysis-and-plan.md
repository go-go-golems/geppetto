---
Title: 'Phase 2 Implementation Analysis: GEPA Benchmark Persistence and eval-report'
Ticket: GP-01-ADD-GEPA-PHASE-2
Status: active
Topics:
    - architecture
    - geppetto
    - inference
    - persistence
    - tools
    - sqlite
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: 2026-02-18--cozodb-extraction/cozo-relationship-js-runner/run_recorder.go
      Note: Reference recorder structure and schema lifecycle patterns
    - Path: 2026-02-18--cozodb-extraction/cozo-relationship-js-runner/eval_report.go
      Note: Reference for report command query and table/json dual output
    - Path: geppetto/cmd/gepa-runner/main.go
      Note: Current optimize command where recorder flags/lifecycle will be integrated
    - Path: geppetto/cmd/gepa-runner/eval_command.go
      Note: Current eval command where per-example persistence will be integrated
ExternalSources: []
Summary: Implementation plan for Phase 2 GEPA persistence, recorder wiring, and eval-report command in local geppetto.
LastUpdated: 2026-02-23T21:25:00-05:00
WhatFor: Define exact storage model, command contract, and migration steps before coding Phase 2.
WhenToUse: Use when implementing or reviewing GEPA Phase 2 benchmark persistence and reporting.
---

# Phase 2 Implementation Analysis

## 1. Scope

Phase 2 maps to section `10.2` from GP-01 analysis:

1. Introduce benchmark persistence with SQLite.
2. Add `eval-report` command for recorded runs.
3. Reuse proven COZO runner patterns, but keep schema GEPA-specific.

## 2. Current State (End of Phase 1)

`cmd/gepa-runner` currently provides:

1. `optimize` command.
2. `eval` command.
3. JSON file output (`--out-report`) only.

No persistent run history exists yet, so cross-run analysis (trend, plugin performance summary, candidate lineage tracking) is not possible.

## 3. Design Goals

1. Keep Phase 1 command behavior compatible.
2. Add optional persistence via flags (`--record`, `--record-db`).
3. Record enough data for:
   - run-level trend analysis,
   - candidate-level optimize analysis,
   - example-level eval debugging.
4. Provide first-class reporting via `gepa-runner eval-report`.

## 4. Proposed Storage Schema

Three tables are sufficient for Phase 2:

1. `gepa_runs`
   - one row per optimize/eval run.
2. `gepa_candidate_metrics`
   - optimize candidate lineage rows from `Result.Candidates`.
3. `gepa_eval_examples`
   - eval command per-example results for inspected candidate.

### 4.1 `gepa_runs`

Primary fields:

1. `run_id` (TEXT PK)
2. `mode` (`optimize` | `eval`)
3. `status` (`completed` | `failed`)
4. `started_at_ms`, `finished_at_ms`, `duration_ms`
5. `plugin_id`, `plugin_name`, `profile`
6. `dataset_size`
7. optimize metrics: `calls_used`, `best_mean_score`, `best_n`
8. eval metrics: `mean_score`, `mean_n`
9. `objective`, `max_evals`, `batch_size`
10. `error` (failed run diagnostics)

### 4.2 `gepa_candidate_metrics`

Primary fields:

1. `run_id`, `candidate_id` (composite PK)
2. `parent_id`, `candidate_hash`
3. `mean_score`, `n`, `evals_cached`
4. `mean_objectives_json`
5. `candidate_json`
6. `reflection_raw`
7. `is_best` (0/1)

### 4.3 `gepa_eval_examples`

Primary fields:

1. `run_id`, `candidate_hash`, `example_index` (composite PK)
2. `score`
3. `objectives_json`
4. `feedback`, `evaluator_notes`
5. `output_json`, `trace_json`, `raw_json`

## 5. Recorder Lifecycle

Pseudocode:

```text
if --record:
  recorder = open(record_db)
  recorder.start(run meta)

run optimize/eval

if optimize:
  recorder.record_optimize_result(result)
if eval:
  recorder.record_eval_result(stats, examples)

recorder.close(success/error)
```

Failure path requirement: a run row still gets written with `status=failed` and `error`.

## 6. `eval-report` Command Contract

Command:

1. `gepa-runner eval-report --db <path> --limit-runs N --format table|json`

Outputs:

1. Recent run table (or JSON array) from `gepa_runs`.
2. Plugin summary aggregate grouped by `plugin_id`.
3. Optional latest optimize candidate count summary derived from `gepa_candidate_metrics`.

## 7. Compatibility and Risks

1. Existing `--out-report` behavior must remain unchanged.
2. `--record` default is false to avoid surprising filesystem writes.
3. Recorder errors should fail command by default (Phase 2 strict mode) to prevent silent data loss.
4. SQLite schema migration is additive (`CREATE TABLE IF NOT EXISTS`).

## 8. Implementation Plan

1. Add recorder module and schema initialization.
2. Wire optimize/eval flags and recorder usage.
3. Add `eval-report` command and SQL query logic.
4. Add tests and smoke artifacts.
5. Update README and ticket docs.

## 9. Acceptance Criteria

1. `go test ./cmd/gepa-runner -count=1` passes.
2. `go build ./cmd/gepa-runner` passes.
3. Optimize + eval runs with `--record` create rows in all expected tables.
4. `eval-report` produces valid table and JSON output.
5. Ticket diary/changelog/tasks are complete and linked.
