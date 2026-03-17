# Changelog

## 2026-03-17

- Created `GP-38` to track the shared-runtime execution cleanup in `scopedjs`.
- Added a new `RuntimeExecutor` wrapper in `pkg/inference/tools/scopedjs/executor.go`.
- Extended `BuildResult` so `BuildRuntime(...)` now returns both the raw runtime and a serialized executor wrapper.
- Switched `RegisterPrebuilt(...)` to evaluate through the wrapper instead of calling `RunEval(...)` on the raw runtime directly.
- Switched the lazy path to use the same wrapper shape for consistency without changing lazy lifecycle semantics.
- Added a concurrency regression test proving prebuilt shared-runtime evals serialize across the whole eval lifecycle instead of interleaving.
- Updated the scopedjs tutorial to explain `BuildResult.Executor` and the safe shared-runtime path.
- Validated with:
  - `go test ./pkg/inference/tools/scopedjs`
  - `go test ./pkg/doc`
  - `docmgr doctor --root geppetto/ttmp --ticket GP-38 --stale-after 30`
- Committed the slice as `6f620b2` (`refactor(scopedjs): add serialized runtime executor`).
