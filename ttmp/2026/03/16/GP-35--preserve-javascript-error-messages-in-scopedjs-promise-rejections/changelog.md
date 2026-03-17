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
