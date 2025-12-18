---
Title: Diary
Ticket: 005-RELAX-TURNSDATALINT
Status: active
Topics:
    - infrastructure
    - inference
    - bug
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-18T18:20:44.817294314-05:00
---

# Diary

## Goal

Track the step-by-step work to relax `turnsdatalint` from **const-identity enforcement** to **typed-key enforcement** (while still rejecting raw string literals / untyped string constants) and keep downstream repos (like Pinocchio) from accumulating workaround APIs.

## Step 1: Create ticket + write rationale (doc-first)

This step set up the Geppetto-side workspace for the change and captured the rationale in one place. The goal is to make the upcoming analyzer change easy to review and safe to ship, with explicit tests and docs updates.

**Commit (docs):** a1cd2937cd8fe4eac3c4e4b495945df690c1c8a0 — "Docs: add ticket to relax turnsdatalint to typed-key enforcement"

### What I did
- Created ticket `005-RELAX-TURNSDATALINT` under `geppetto/ttmp/2025/12/18/...`
- Added analysis doc: `analysis/01-rationale-relax-turnsdatalint-to-typed-key-enforcement.md`

### Why
- The linter behavior is an upstream contract; changing it should be documented and test-driven.

### What worked
- Ticket scaffold + rationale doc provide a stable place to link code and downstream context.

### What warrants a second pair of eyes
- Ensure the intended new rule still blocks actual drift cases (raw string literals / untyped string consts).

## Step 2: Implement analyzer change + update tests/docs (in progress)

This step updates `turnsdatalint` so typed expressions (vars/params/conversions) are accepted for typed-key maps, while still rejecting raw string literals and untyped string constants. It also updates analysistest fixtures and the topic documentation.

**Commit (code):** N/A — implemented, commit pending

### What I’m changing
- `geppetto/pkg/analysis/turnsdatalint/analyzer.go`: switch from const-identity to type-based key checks; remove helper allowlist
- `geppetto/pkg/analysis/turnsdatalint/testdata/src/a/a.go`: adjust expected violations (raw string literals + untyped string consts) and allow typed vars/params/conversions
- `geppetto/pkg/doc/topics/12-turnsdatalint.md`: document the relaxed rule and update examples/flags

### Code review instructions
- Start with `pkg/analysis/turnsdatalint/analyzer.go` (core semantic change), then review the testdata diff to confirm the new behavior is enforced.

### What I did
- Implemented type-based key checking and explicit rejection of:
  - raw string literals (`t.Data["raw"]`)
  - untyped string const identifiers (`const k = "raw"; t.Data[k]`)
- Updated analysistest fixture expectations in `testdata/src/a/a.go`
- Updated the topic documentation in `pkg/doc/topics/12-turnsdatalint.md`
- Ran:
  - `gofmt -w pkg/analysis/turnsdatalint/analyzer.go pkg/analysis/turnsdatalint/testdata/src/a/a.go`
  - `go test ./pkg/analysis/turnsdatalint -count=1`
  - `make build`
  - `go test ./... -count=1`
  - `make linttool`

### What worked
- All tests passed and `make linttool` succeeded.

