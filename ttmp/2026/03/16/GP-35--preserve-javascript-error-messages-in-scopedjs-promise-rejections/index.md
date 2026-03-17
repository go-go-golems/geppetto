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
Summary: GitHub bug report plus local repro material for scopedjs losing JavaScript Error messages and returning Promise rejected: map[].
LastUpdated: 2026-03-16T21:17:17.370109399-04:00
WhatFor: Track the scopedjs rejection-message bug from discovery through GitHub issue filing, including the exact repro command and the code paths most likely involved.
WhenToUse: Use when fixing or reviewing promise rejection handling in scopedjs, especially around JavaScript Error objects.
---

# Preserve JavaScript Error messages in scopedjs promise rejections

## Overview

This ticket captures a bug in `scopedjs` promise rejection handling. String rejections survive, but real JavaScript `Error` objects lose their message and end up surfaced to callers as `Promise rejected: map[]`.

The GitHub issue is filed as `go-go-golems/geppetto#302`, and this workspace contains the exact repro script and issue body used to open it.

## Key Links

- GitHub issue: `go-go-golems/geppetto#302`
- Issue body source: `sources/01-gh-issue-body.txt`
- Minimal repro: `scripts/repro_scopedjs_js_error_rejection.go`
- Analysis doc: `analysis/01-scopedjs-javascript-error-rejection-repro-and-issue-draft.md`
- Diary: `reference/01-diary.md`

## Status

Current status: **active**

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
