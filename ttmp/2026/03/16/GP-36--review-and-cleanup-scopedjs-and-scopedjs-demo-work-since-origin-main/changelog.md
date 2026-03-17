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
