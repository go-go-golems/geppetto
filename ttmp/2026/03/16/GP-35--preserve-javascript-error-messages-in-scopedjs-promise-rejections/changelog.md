# Changelog

## 2026-03-16

- Initial workspace created
- Isolated the bug to `scopedjs` handling of rejected JavaScript `Error` objects rather than string promise rejections.
- Added a minimal repro program at `scripts/repro_scopedjs_js_error_rejection.go`.
- Captured the exact repro output showing:
  - `await Promise.reject("boom")` -> `Promise rejected: boom`
  - `await Promise.reject(new Error("boom"))` -> `Promise rejected: map[]`
  - `throw new Error("boom")` -> `Promise rejected: map[]`
- Drafted the GitHub issue body in `sources/gh-issue-body.md`.
- Filed `go-go-golems/geppetto#302` with `gh issue create`.
- Implemented a local fix in `pkg/inference/tools/scopedjs/eval.go`:
  - promise rejections now retain the raw `goja.Value` until formatting time
  - rejected JS `Error` values are rendered with their JavaScript string form instead of `Export()`
  - returned and console-logged JS `Error` values now also preserve text instead of degrading to `map[]`
- Added regression coverage in `pkg/inference/tools/scopedjs/runtime_test.go` for:
  - `await Promise.reject(new Error("boom"))`
  - `throw new Error("boom")`
  - `return new Error("boom")`
  - `console.error(new Error("boom"))`
- Verified the fix with `go test ./pkg/inference/tools/scopedjs`.
