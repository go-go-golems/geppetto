# Changelog

## 2026-03-16

- Initial workspace created
- Collected all `scopedjs` and `scopedjs-tui-demo` commits since `origin/main`
- Reviewed core `scopedjs` package files, examples, tests, and Pinocchio demo files
- Identified six main cleanup findings:
  - `StateMode` does not match actual runtime lifecycle behavior
  - lazy registration loses manifest-derived capability description
  - eval option override semantics are too weak for booleans
  - Pinocchio demo shell wiring duplicates `scopeddb` demo shell wiring
  - Pinocchio renderer plumbing duplicates `scopeddb` demo renderer plumbing
  - fake example modules are duplicated across examples and may deserve extraction
- Added a detailed intern-facing review, design, and implementation guide
- Added a diary recording commands, evidence, and findings
- Validated the ticket with `docmgr doctor --ticket GP-36`
- Uploaded the ticket bundle to reMarkable at `/ai/2026/03/16/GP-36`
- Started the first cleanup implementation slice in `geppetto/pkg/inference/tools/scopedjs`.
- Replaced description-time `StateMode` prose with registration-driven runtime reuse notes so prebuilt tools now advertise shared-runtime reuse and lazy tools advertise fresh-per-call runtime construction.
- Added `EnvironmentSpec.Describe` as a static manifest-planning hook and taught `NewLazyRegistrar(...)` to build model-facing descriptions from that manifest instead of `EnvironmentManifest{}`.
- Replaced registration override inputs with `EvalOptionOverrides`, using pointer-backed tri-state fields so boolean settings such as `CaptureConsole` can now be overridden explicitly in both directions.
- Updated the runnable `scopedjs` examples and the Pinocchio scopedjs demo environment to provide static manifests for richer lazy/prebuilt descriptions.
- Verified the slice with:
  - `go test ./pkg/inference/tools/scopedjs ./cmd/examples/scopedjs-tool ./cmd/examples/scopedjs-dbserver ./pkg/doc/...` in `geppetto`
  - `go test ./cmd/examples/scopedjs-tui-demo` in `pinocchio`
- Added `pinocchio/cmd/examples/internal/tuidemo` to hold the shared profile-resolution, tool-loop runner, and status-bar scaffolding used by both TUI demos.
- Added `pinocchio/cmd/examples/internal/demorender` to hold the shared markdown renderer factories and presentation helpers used by both demo timelines.
- Cut `cmd/examples/scopeddb-tui-demo/main.go` and `cmd/examples/scopedjs-tui-demo/main.go` over to the shared shell helpers.
- Cut both demo renderer files over to the shared renderer plumbing so they now mostly keep only domain-specific markdown formatting.
- Decided not to extract the fake `webserver` and `obsidian` modules into `geppetto/pkg/inference/tools/scopedjs/scopedjstest` in this pass:
  - the demo modules still mix example-specific behavior with runtime fixtures
  - the shell and renderer duplication was the more urgent maintenance problem
  - the fake-module extraction can happen later once a second non-demo consumer wants the same support layer
- Verified the Pinocchio refactor slice with:
  - `go test ./cmd/examples/scopeddb-tui-demo ./cmd/examples/scopedjs-tui-demo ./cmd/examples/internal/...`
  - `go run ./cmd/examples/scopedjs-tui-demo --list-workspaces`
