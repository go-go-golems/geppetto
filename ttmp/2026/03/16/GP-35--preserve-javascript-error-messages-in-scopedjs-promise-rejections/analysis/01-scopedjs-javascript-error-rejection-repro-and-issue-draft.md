---
Title: Scopedjs JavaScript Error rejection repro and issue draft
Ticket: GP-35
Status: active
Topics:
    - js-bindings
    - tools
    - bug
    - geppetto
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/inference/tools/scopedjs/eval.go
      Note: |-
        Rejected promise values are exported and formatted here
        Promise rejection export and formatting path
    - Path: pkg/inference/tools/scopedjs/runtime_test.go
      Note: |-
        Existing tests currently only cover string rejection
        Existing coverage and missing JS Error cases
    - Path: ttmp/2026/03/16/GP-35--preserve-javascript-error-messages-in-scopedjs-promise-rejections/scripts/repro_scopedjs_js_error_rejection.go
      Note: |-
        Minimal repro used for issue filing
        Minimal repro used in the issue
ExternalSources:
    - https://github.com/go-go-golems/geppetto/issues/302
Summary: Analysis backing the GitHub bug report about scopedjs losing JavaScript Error messages in rejected promises.
LastUpdated: 2026-03-16T21:28:00-04:00
WhatFor: Preserve the exact repro, the narrowed bug boundary, and the likely code paths involved so the eventual fix starts from evidence instead of guesswork.
WhenToUse: Use when implementing or reviewing a fix for scopedjs rejection-message preservation.
---


# Scopedjs JavaScript Error rejection repro and issue draft

## Goal

Capture a minimal and precise bug report for `scopedjs` losing the message of rejected JavaScript `Error` objects, then turn that into a GitHub issue with a reproducible command.

## What the bug is

`scopedjs.RunEval(...)` handles rejected promises, but it does not preserve the message when the rejection value is a real JavaScript `Error` object. Instead, the error string returned to the caller becomes `Promise rejected: map[]`.

The behavior currently splits like this:

- `await Promise.reject("boom")` works and surfaces `Promise rejected: boom`
- `await Promise.reject(new Error("boom"))` loses the message and surfaces `Promise rejected: map[]`
- `throw new Error("boom")` also surfaces `Promise rejected: map[]`

That is a bad LLM-facing contract because JavaScript code normally throws `Error` objects, not strings.

## Minimal repro

The ticket-local repro program is:

- `scripts/repro_scopedjs_js_error_rejection.go`

Run it from the Geppetto repo root:

```bash
go run ./ttmp/2026/03/16/GP-35--preserve-javascript-error-messages-in-scopedjs-promise-rejections/scripts/repro_scopedjs_js_error_rejection.go
```

Observed output:

```json
== string rejection ==
{
  "error": "Promise rejected: boom"
}

== error rejection ==
{
  "error": "Promise rejected: map[]"
}

== throw error ==
{
  "error": "Promise rejected: map[]"
}
```

## Code-path analysis

The likely failure mode is in `pkg/inference/tools/scopedjs/eval.go`.

Relevant flow:

1. `RunEval(...)` executes JS in an async wrapper.
2. `executeEval(...)` detects a promise result.
3. `waitForPromise(...)` snapshots the promise state and rejected value.
4. The rejected value is passed through `exportValue(...)`.
5. The final error string is built with `fmt.Errorf("Promise rejected: %v", snap.Result)`.

The likely bad transformation is:

```go
Result: exportValue(promise.Result())
```

If `promise.Result()` is a JavaScript `Error` object, `goja.Value.Export()` appears to collapse into an empty map-like Go representation. By the time the code formats the rejection, the original message has already been lost.

## Existing test coverage gap

`pkg/inference/tools/scopedjs/runtime_test.go` already asserts the string rejection case:

```go
Code: `await Promise.reject("boom")`
```

What is missing:

- `await Promise.reject(new Error("boom"))`
- `throw new Error("boom")`

Those two cases are exactly the ones that failed in the repro program and in the downstream Pinocchio demo.

## GitHub issue

Filed as:

- `go-go-golems/geppetto#302`

The issue body used for filing is stored at:

- `sources/01-gh-issue-body.txt`

## Suggested fix direction

The rejection path should special-case JavaScript error values before generic export/formatting. A fix could preserve one of:

- `.message`
- `String(error)`
- a structured `{ name, message, stack }` object

The minimum acceptable contract is simpler:

- a rejected JavaScript `Error` must produce a non-empty error string that still contains the original message text

## Acceptance criteria

- `await Promise.reject(new Error("boom"))` returns an `EvalOutput.Error` containing `boom`
- `throw new Error("boom")` returns an `EvalOutput.Error` containing `boom`
- `await Promise.reject("boom")` continues to work unchanged
- regression tests are added for both JavaScript `Error` cases
