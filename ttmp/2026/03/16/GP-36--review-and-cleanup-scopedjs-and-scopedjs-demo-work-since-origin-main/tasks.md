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
- [x] Extract shared Pinocchio example shell wiring from `scopeddb` and `scopedjs` demos into a small internal helper package
- [x] Extract shared Pinocchio timeline/render helper plumbing from the two demos so example files keep only domain-specific formatting
- [x] Decide whether fake example modules should move into a reusable test-double/example support package and document the outcome

## TUI Demo Cleanup Execution

- [x] Re-check the current Pinocchio demo state before touching code, including whether helper extraction already happened partially
- [x] Correct the GP-36 task list so it no longer claims the TUI cleanup is already finished
- [x] Extract the duplicated Cobra/logging/profile-resolution/engine-creation shell from the two demo `main.go` files into `cmd/examples/internal/tuidemo`
- [x] Cut `cmd/examples/scopeddb-tui-demo/main.go` over to the shared shell helper
- [x] Cut `cmd/examples/scopedjs-tui-demo/main.go` over to the shared shell helper
- [x] Re-run the demo example tests and a basic `go run --help` smoke check for both binaries
- [x] Reassess whether any meaningful renderer duplication still remains after the shell cleanup; if not, record that outcome explicitly instead of forcing another extraction
- [x] Update the GP-36 diary, changelog, and design guide with what was actually changed
