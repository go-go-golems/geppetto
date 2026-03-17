---
Title: Preserve JavaScript Error messages in scopedjs promise rejections
Ticket: GP-35
Status: active
Topics:
    - js-bindings
    - tools
    - bug
    - geppetto
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/eval.go
      Note: Promise rejection handling currently collapses JavaScript Error objects to map[]
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/runtime_test.go
      Note: Existing tests cover string promise rejection but not JavaScript Error object rejection
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/ttmp/2026/03/16/GP-35--preserve-javascript-error-messages-in-scopedjs-promise-rejections/scripts/repro_scopedjs_js_error_rejection.go
      Note: Minimal reproducible example used in the GitHub issue
ExternalSources:
    - https://github.com/go-go-golems/geppetto/issues/302
Summary: Tracks the scopedjs JavaScript Error export bug from repro and issue filing through the local fix that now preserves JS error text for rejected, thrown, returned, and console-logged Error values.
LastUpdated: 2026-03-16T22:08:46-04:00
WhatFor: Track the scopedjs JavaScript Error formatting bug, the upstream GitHub issue, the local code fix, and the regression coverage for future reviewers.
WhenToUse: Use when fixing or reviewing JavaScript Error handling in scopedjs, especially around rejected promises and exported goja values.
---

# Preserve JavaScript Error messages in scopedjs promise rejections

## Overview

This ticket captures a bug in `scopedjs` JavaScript Error handling. String rejections survived, but real JavaScript `Error` objects lost their message and ended up surfaced to callers as `Promise rejected: map[]`.

The GitHub issue is filed as `go-go-golems/geppetto#302`, and this workspace now contains both the original repro material and the local implementation fix. The fix preserves useful text for:

- `await Promise.reject(new Error("boom"))`
- `throw new Error("boom")`
- `return new Error("boom")`
- `console.error(new Error("boom"))`

## Key Links

- GitHub issue: `go-go-golems/geppetto#302`
- Issue body source: `sources/01-gh-issue-body.txt`
- Minimal repro: `scripts/repro_scopedjs_js_error_rejection.go`
- Analysis doc: `analysis/01-scopedjs-javascript-error-rejection-repro-and-issue-draft.md`
- Diary: `reference/01-diary.md`

## Status

Current status: **active**

Local state:

- GitHub issue filed upstream
- local `scopedjs` fix implemented
- regression tests added and passing

## Topics

- js-bindings
- tools
- bugs
- geppetto

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
