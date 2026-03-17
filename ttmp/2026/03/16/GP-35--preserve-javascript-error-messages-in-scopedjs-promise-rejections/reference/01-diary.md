---
Title: Diary
Ticket: GP-35
Status: active
Topics:
    - js-bindings
    - tools
    - bug
    - geppetto
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/inference/tools/scopedjs/eval.go
      Note: |-
        Main code path implicated by the rejection formatting bug
        Main bug boundary discussed in the diary
    - Path: pkg/inference/tools/scopedjs/runtime_test.go
      Note: |-
        Existing tests that helped narrow the bug to JavaScript Error objects
        Test gap that shaped the repro
    - Path: ttmp/2026/03/16/GP-35--preserve-javascript-error-messages-in-scopedjs-promise-rejections/scripts/repro_scopedjs_js_error_rejection.go
      Note: |-
        Ticket-local repro program used for issue creation
        Command-backed diary artifact
    - Path: ttmp/2026/03/16/GP-35--preserve-javascript-error-messages-in-scopedjs-promise-rejections/sources/01-gh-issue-body.txt
      Note: |-
        Exact issue body submitted with gh
        Exact gh issue content filed upstream
ExternalSources:
    - https://github.com/go-go-golems/geppetto/issues/302
Summary: Diary of narrowing the scopedjs rejection bug, writing a minimal repro, and filing Geppetto issue 302.
LastUpdated: 2026-03-16T21:29:00-04:00
WhatFor: Preserve the exact reasoning, commands, and observed outputs behind issue 302.
WhenToUse: Use when continuing the fix or reviewing how the bug was isolated.
---


# Diary

## Goal

Record how the scopedjs rejection bug was narrowed, reproduced, and turned into a GitHub issue with a concrete local repro.

## Step 1: Reproduce and File the Geppetto Issue

The immediate goal was not to fix `scopedjs`, but to file a high-quality Geppetto issue that somebody else could pick up without first reproducing the failure by hand. The Pinocchio demo had already shown `Promise rejected: map[]`, but that symptom alone was too noisy because it mixed the downstream UI with the upstream runtime.

I narrowed the bug until it was obviously a `scopedjs` contract problem: string promise rejections work, but JavaScript `Error` objects lose their message before they reach `EvalOutput.Error`. Once that boundary was clear, I wrote a minimal ticket-local repro program and used its exact output as the body of the GitHub issue.

### Prompt Context

**User prompt (verbatim):** "Create a ticket in go-go-golems/geppetto with gh, with all the details / repro steps (you can use scripts/ in the ticket folder if needed). Keep a diary afterwards"

**Assistant interpretation:** Create a GitHub issue in the Geppetto repo with a precise bug report, store the local supporting artifacts in a Geppetto ticket workspace, and document the work in a diary.

**Inferred user intent:** Preserve the scopedjs rejection bug as an actionable upstream issue instead of leaving it as an implicit downstream observation from the Pinocchio demo.

**Commit (code):** N/A

### What I did

- Checked `gh auth status` to confirm issue creation would work.
- Read `pkg/inference/tools/scopedjs/eval.go` and `pkg/inference/tools/scopedjs/runtime_test.go`.
- Confirmed the existing tests only cover `await Promise.reject("boom")`.
- Created the Geppetto doc ticket `GP-35`.
- Added `scripts/repro_scopedjs_js_error_rejection.go` under the ticket workspace.
- Ran:

```bash
go run ./ttmp/2026/03/16/GP-35--preserve-javascript-error-messages-in-scopedjs-promise-rejections/scripts/repro_scopedjs_js_error_rejection.go
```

- Captured the exact output showing the split between string rejection and JavaScript `Error` rejection.
- Wrote the initial GitHub issue body as `sources/gh-issue-body.md` with summary, repro, expected behavior, suspected cause, and acceptance criteria.
- Filed the upstream issue with:

```bash
gh issue create \
  --repo go-go-golems/geppetto \
  --title "Preserve JavaScript Error messages in scopedjs promise rejections" \
  --label bug \
  --body-file /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/ttmp/2026/03/16/GP-35--preserve-javascript-error-messages-in-scopedjs-promise-rejections/sources/gh-issue-body.md
```

- Recorded the resulting issue URL: `https://github.com/go-go-golems/geppetto/issues/302`.
- Renamed the stored issue body artifact to `sources/01-gh-issue-body.txt` after `docmgr doctor` reported that a bare `.md` file in `sources/` was being treated as a frontmatter-managed doc.

### Why

- The downstream Pinocchio demo exposed the symptom, but the real bug belongs in Geppetto.
- A ticket without a minimal repro would be easy to misread as "demo renderer weirdness" rather than a runtime export/formatting bug.
- Using a ticket-local script keeps the repro stable and reviewable without touching product code.

### What worked

- `gh` authentication was already valid for the `wesen` account.
- The minimal repro was small and deterministic.
- The repro isolated the exact boundary:
  - string rejection preserved the message
  - JavaScript `Error` rejection lost the message
- GitHub issue creation succeeded on the first try.

### What didn't work

- Searching the codebase for the literal string `Promise rejected: map[]` found nothing, because the string is assembled dynamically from `fmt.Errorf("Promise rejected: %v", snap.Result)`.
- The existing `scopedjs` tests were not enough to prove the bug because they only exercised `Promise.reject("boom")`, not real `Error` objects.

### What I learned

- The bug is narrower than "all promise rejections are broken." It specifically affects rejected JavaScript `Error` objects.
- `exportValue(...)` is a likely lossy step for JS `Error` values when they are exported into Go before formatting.
- `throw new Error("boom")` and `await Promise.reject(new Error("boom"))` both surface the same bad `map[]` behavior, which suggests the async wrapper/promise rejection path is the right place to fix it.

### What was tricky to build

- The tricky part was separating the downstream symptom from the upstream bug. The first observation came from a TUI demo, but filing a good upstream issue required proving the problem without any Pinocchio-specific code in the loop.
- The second tricky part was that one existing test already said promise rejection handling works. That was technically true for strings, so the repro had to show the narrower distinction between string rejections and `Error` objects.

### What warrants a second pair of eyes

- Whether the best contract is plain-string preservation, `String(error)`, or a structured `{ name, message, stack }` payload.
- Whether `exportValue(...)` should special-case JavaScript `Error` values globally or only in the rejection path.
- Whether there are related losses for thrown non-Error objects that should also be covered once the main bug is fixed.

### What should be done in the future

- Add regression tests for:
  - `await Promise.reject(new Error("boom"))`
  - `throw new Error("boom")`
- Decide whether stack traces should be preserved in the final `EvalOutput.Error` contract.

### Code review instructions

- Start with `pkg/inference/tools/scopedjs/eval.go`, especially `waitForPromise(...)` and `exportValue(...)`.
- Then compare against `pkg/inference/tools/scopedjs/runtime_test.go` to see what is currently covered and what is missing.
- Re-run the local repro script:

```bash
go run ./ttmp/2026/03/16/GP-35--preserve-javascript-error-messages-in-scopedjs-promise-rejections/scripts/repro_scopedjs_js_error_rejection.go
```

- Confirm the GitHub issue body in `sources/01-gh-issue-body.txt` still matches the observed output.

### Technical details

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

GitHub issue:

- `go-go-golems/geppetto#302`
