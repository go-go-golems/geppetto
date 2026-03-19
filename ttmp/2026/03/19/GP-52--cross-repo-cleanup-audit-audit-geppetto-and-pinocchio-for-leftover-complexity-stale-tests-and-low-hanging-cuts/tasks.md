# Tasks

## Completed

- [x] Create a shared cross-repo ticket workspace under the Geppetto `ttmp` root
- [x] Inventory Geppetto and Pinocchio structure, large files, and large tests
- [x] Read architecture spine files for both repositories
- [x] Write an intern-facing cleanup audit and phased implementation guide
- [x] Write an investigation diary with evidence and follow-up context

## Recommended Follow-Up

- [ ] Investigate external usage before deleting `geppetto/pkg/sections/profile_sections.go` helpers
- [ ] Investigate external usage before deleting `geppetto/pkg/events` legacy event shapes
- [ ] Investigate downstream imports of `pinocchio/pkg/geppettocompat`
- [ ] Confirm whether the `simple-chat-agent` `debugcmds` build-tag path is still used by anyone
- [ ] Consolidate Pinocchio runtime/bootstrap resolution onto `pkg/cmds/helpers`
- [ ] Migrate `cmd/web-chat/main.go` and tests off deprecated `pkg/webchat` constructors
- [ ] Split current-vs-legacy web-chat policy tests so compatibility removal has a clear boundary
- [ ] Split Geppetto JS module contract tests by namespace

## Publication

- [x] Relate key source files to the ticket documents
- [x] Run `docmgr` validation for the ticket workspace
- [x] Upload the final packet to reMarkable
