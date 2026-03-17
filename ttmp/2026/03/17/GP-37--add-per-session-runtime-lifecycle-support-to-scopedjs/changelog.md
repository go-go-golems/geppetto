# Changelog

## 2026-03-17

- Created `GP-37` to track future `scopedjs` per-session runtime lifecycle support.
- Reviewed the current `scopedjs` lifecycle split:
  - prebuilt registration reuses one runtime instance
  - lazy registration builds a fresh runtime per call
- Reviewed existing Geppetto session identity plumbing in `pkg/inference/session/context.go` and `pkg/inference/session/session.go`.
- Drafted a detailed design and implementation guide aimed at a new intern.
- Drafted the upstream GitHub issue body from the same analysis so the local ticket and GitHub issue can stay aligned.
- Filed `go-go-golems/geppetto#304` for the future per-session scopedjs runtime work.
- Validated the ticket with `docmgr doctor --root geppetto/ttmp --ticket GP-37 --stale-after 30`.
