---
Title: JS API Feature Validation Guide and geppetto-js-lab Scripts
Ticket: GP-17-JS-LAB-API-EXAMPLES
Status: active
Topics:
    - geppetto
    - inference
    - tools
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/examples/geppetto-js-lab/main.go
      Note: Harness used to run all validation scripts
    - Path: examples/js/geppetto/07_context_and_constants.js
      Note: Baseline example referenced by context validation script
    - Path: ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/01_handles_consts_and_turns.js
      Note: Validates hidden handles and generated constants
    - Path: ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/02_context_hooks_and_run_options.js
      Note: Validates context and run option propagation
    - Path: ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/03_async_surface_smoke.js
      Note: Validates runAsync/start surface
    - Path: ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/run_all.sh
      Note: Runs all GP-17 validation scripts in one command
ExternalSources: []
Summary: End-to-end validation guide for recently added Geppetto JS API features, with runnable geppetto-js-lab scripts and expected outcomes.
LastUpdated: 2026-02-20T17:46:29.43222161-05:00
WhatFor: Provide a practical, reproducible verification playbook for JS API improvements delivered in GP-01/GP-16.
WhenToUse: Use when validating JS API behavior after refactors, upgrading dependencies, or onboarding developers.
---


# JS API Feature Validation Guide and `geppetto-js-lab` Scripts

## Goal

This document provides a concrete validation plan for the new Geppetto JS API capabilities delivered in recent tickets. It combines:

- a feature inventory,
- step-by-step test commands,
- new ticket-local validation scripts,
- expected outputs and troubleshooting notes.

The scripts in this ticket are intentionally focused on **fast smoke/integration checks** that can be executed locally with the same harness developers use for manual JS experimentation.

## Scope of features covered

This validation pack targets the features introduced across the JS API improvement work:

1. Opaque Go handle safety (hidden `__geppetto_ref`)
2. Generated constants surface (`gp.consts.*`) and key groups
3. Context plumbing to middleware/tool handlers/tool hooks
4. Per-run options (`timeoutMs`, `tags`) through `session.run()`/`session.start()`
5. Async entrypoint surfaces (`runAsync()`, `start()`, `RunHandle` methods)

## Test harness

All scripts are executed with:

```bash
go run ./cmd/examples/geppetto-js-lab --script <script-path>
```

Harness behavior assumptions:

- `require("geppetto")` is pre-registered.
- Global helpers available:
  - `assert(cond, msg)`
  - `console.log`, `console.error`
  - `ENV`
- Built-in Go tools available via `tools.useGoTools()`:
  - `go_double`
  - `go_concat`

## New scripts in this ticket

Location:

- `ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts`

### Script matrix

| Script | Primary coverage | Why it matters |
|---|---|---|
| `01_handles_consts_and_turns.js` | Hidden handle behavior + const groups + turns/session sanity | Catches regressions in opaque ref semantics and generated constants exports |
| `02_context_hooks_and_run_options.js` | Middleware/tool/hook context + run tags/deadline propagation | Validates observability and context-aware middleware/tool integration |
| `03_async_surface_smoke.js` | `runAsync()`, `start()`, `RunHandle` shape, event subscriptions, cancel path | Ensures async public API remains wired and callable |
| `run_all.sh` | Executes all three scripts sequentially | One-command smoke run for local/dev CI workflows |

## How to run

### Run full suite

```bash
./ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/run_all.sh
```

### Run individual scripts

```bash
go run ./cmd/examples/geppetto-js-lab \
  --script ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/01_handles_consts_and_turns.js

go run ./cmd/examples/geppetto-js-lab \
  --script ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/02_context_hooks_and_run_options.js

go run ./cmd/examples/geppetto-js-lab \
  --script ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/03_async_surface_smoke.js
```

## Expected outcomes

Each run should end with:

- `PASS: <script-name>.js`

Additional script-specific evidence:

- Script 01 prints engine keys (without `__geppetto_ref`) and echo output.
- Script 02 prints serialized middleware/tool/hook context payloads containing session/inference IDs and tags.
- Script 03 prints promise/cancel surface details and confirms `RunHandle` methods exist.

## Notes on async validation depth

`geppetto-js-lab` is ideal for API-level smoke checks, but script-level async validation is intentionally shallow in this pack:

- it verifies public async API shape and basic callability,
- it does not attempt long-running deterministic event-order assertions in pure JS.

For deep async correctness (race/deadlock/order guarantees), use package tests in:

- `pkg/js/modules/geppetto/module_test.go`

## Regression checklist (quick review)

When a script fails, check these first:

1. `go run ./cmd/examples/geppetto-js-lab --list-go-tools` still shows expected built-ins.
2. `gp.consts` groups still include:
   - `ToolChoice`, `ToolErrorHandling`, `BlockKind`, `TurnDataKeys`, `TurnMetadataKeys`, `BlockMetadataKeys`, `RunMetadataKeys`, `PayloadKeys`, `EventType`
3. Context fields still propagated:
   - `sessionId`, `inferenceId`, `deadlineMs`, `tags`
4. Async surface still exports:
   - `session.runAsync()` promise-like result
   - `session.start()` handle with `.promise`, `.on()`, `.cancel()`

## Relationship to existing example suite

Repository examples in `examples/js/geppetto/*.js` continue to be canonical demonstrations. The GP-17 scripts are a **ticket-local validation bundle** optimized for feature regression checks and onboarding verification.

## Suggested future extension

If we want this to run in CI as a named smoke suite, add a small target that executes `scripts/run_all.sh` and fails on non-zero exit.
