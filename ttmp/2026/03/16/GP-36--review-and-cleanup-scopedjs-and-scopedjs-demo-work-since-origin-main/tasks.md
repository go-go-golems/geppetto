# Tasks

## Completed

- [x] Create GP-36 review ticket workspace in `geppetto/ttmp`
- [x] Identify all `geppetto` commits since `origin/main` related to `scopedjs`
- [x] Identify all `pinocchio` commits since `origin/main` related to `scopedjs-tui-demo`
- [x] Collect high-signal code references in `scopedjs` package files
- [x] Compare `scopedjs-tui-demo` shell wiring against `scopeddb-tui-demo`
- [x] Compare `scopedjs-tui-demo` renderer wiring against `scopeddb-tui-demo`
- [x] Draft a detailed review and cleanup guide for a new intern
- [x] Record the evidence and commands in a chronological diary
- [x] Run `docmgr doctor` for GP-36
- [x] Upload the ticket bundle to reMarkable

## Follow-Up Work Suggested By This Review

- [x] Clarify `scopedjs` runtime reuse semantics so tool descriptions reflect registration behavior instead of the current misleading `StateMode` story
- [x] Introduce a static manifest-planning path so lazy and prebuilt tools expose the same capability description
- [x] Replace `EvalOptions` boolean override merging with an explicit tri-state override model and add focused tests
- [ ] Extract shared Pinocchio example shell wiring from `scopeddb` and `scopedjs` demos into a small internal helper package
- [ ] Extract shared Pinocchio timeline/render helper plumbing from the two demos so example files keep only domain-specific formatting
- [ ] Decide whether fake example modules should move into a reusable test-double/example support package and document the outcome

## Current Implementation Slices

- [x] Slice 1: update GP-36 tasks/changelog/diary to track concrete cleanup implementation work rather than only the review findings
- [x] Slice 2: fix `scopedjs` registration semantics and lazy description planning in `geppetto/pkg/inference/tools/scopedjs`
- [x] Slice 3: fix eval option override semantics and extend `scopedjs` tests
- [ ] Slice 4: extract shared Pinocchio demo shell helper package and cut both demo `main.go` files over to it
- [ ] Slice 5: extract shared Pinocchio timeline/render helper package and cut both demo renderer files over to it
- [ ] Slice 6: document the fake-module reuse decision and capture the final implementation diary plus changelog
