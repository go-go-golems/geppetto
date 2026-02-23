# Tasks

## Phase 2 Scope (`10.2`)

Implement GEPA benchmark persistence and reporting in local `cmd/gepa-runner`, borrowing proven structure from `cozo-relationship-js-runner` while keeping GEPA-specific schemas and outputs.

## Track A: Analysis and Contract

- [x] Write Phase 2 analysis document describing architecture, storage schema, and command UX.
- [x] Define SQLite schema for GEPA runs, candidate metrics, and eval example metrics.
- [x] Define `eval-report` output contract (`table|json`) and summary aggregates.
- [x] Add detailed implementation checklist mapped to this ticket.

## Track B: Recorder Implementation

- [x] Add `cmd/gepa-runner/run_recorder.go` with SQLite init and insert APIs.
- [x] Add stable run ID generation and candidate hash helper.
- [x] Persist optimize run summary (run-level metrics + candidate entries).
- [x] Persist eval run summary (run-level metrics + per-example eval rows).
- [x] Ensure recorder cleanly captures failure status and error message on command failure.

## Track C: Runner Integration

- [x] Add `--record` and `--record-db` flags to `optimize`.
- [x] Add `--record` and `--record-db` flags to `eval`.
- [x] Wire recorder lifecycle into optimize/eval execution paths.
- [x] Keep existing optimize/eval JSON report file behavior unchanged.

## Track D: Reporting Command

- [x] Add `cmd/gepa-runner/eval_report.go`.
- [x] Register `eval-report` cobra command in `cmd/gepa-runner/main.go`.
- [x] Implement SQL queries for recent runs and plugin summary aggregates.
- [x] Support `--format table|json`, `--db`, `--limit-runs`.

## Track E: Validation and Docs

- [x] Add tests for recorder schema/init + write/read path.
- [x] Add tests for report query/format path using temp SQLite fixtures.
- [x] Run validation commands:
- [x] `go test ./cmd/gepa-runner -count=1`
- [x] `go build ./cmd/gepa-runner`
- [x] Capture smoke artifacts in ticket `sources/` for optimize/eval + eval-report.
- [x] Update `cmd/gepa-runner/README.md` with recording and report usage.

## Ticket Hygiene / Delivery

- [x] Keep `reference/01-diary.md` updated with each implementation step.
- [x] Update `changelog.md` after each major milestone.
- [x] Mark Phase 2 done only after all tracks pass and artifacts are linked.
