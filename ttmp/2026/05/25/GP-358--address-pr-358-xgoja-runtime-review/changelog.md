# Changelog

## 2026-05-25

- Initial workspace created.
- Retrieved PR 358 review threads:
  - `IncludeDefaultModules=false` leaked go-go-goja data-only default modules.
  - nil runtime initializers were now rejected by the builder instead of skipped.
- Updated `pkg/js/runtime/runtime.go` so default module inclusion is fully controlled by `Options.IncludeDefaultModules` and nil runtime initializers are filtered before builder registration.
- Added regression tests in `pkg/js/runtime/runtime_test.go`.
- Added ELI5 explanation in `reference/01-eli5-pr-358-runtime-review-comments.md`.
- Validation passed: `go test ./pkg/js/runtime -count=1` and `go test ./pkg/js/... -count=1`.
- Committed and pushed fix commit `4de12305 fix: preserve geppetto runtime option contracts` to PR 358.
- Resolved both Codex review threads and posted PR follow-up comment: https://github.com/go-go-golems/geppetto/pull/358#issuecomment-4538749186
