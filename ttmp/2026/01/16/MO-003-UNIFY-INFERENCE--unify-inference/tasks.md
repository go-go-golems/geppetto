# Tasks

## TODO

- [ ] Add tasks here

- [ ] Inventory current inference loops and conversation state usage (pinocchio TUI/webchat, moments webchat)
- [ ] Define unified inference loop API contract (inputs, outputs, middleware/tool registries, event sink)
- [ ] Design prompt-resolution middleware (tag schema, slug resolution, templating, draft bundle support)
- [ ] Implement shared conversation builder in geppetto (state snapshot + turn builder)
- [ ] Migrate pinocchio TUI to shared inference builder
- [ ] Migrate pinocchio webchat to shared inference builder
- [ ] Author Moments follow-up plan doc (no code) for future migration
- [ ] Add tests/fixtures for unified inference ordering + prompt resolution
- [ ] Update docs/designs with final unified flow diagrams
- [ ] Add webchat DebugTap to persist pre-inference turn snapshots under /tmp/conversations/<conv>/NN-*.yaml
- [ ] Define shared inference runner for pinocchio TUI + webchat (single entrypoint, shared options)
- [ ] Refactor pinocchio TUI backend to use shared runner and event sink wiring
- [ ] Refactor pinocchio webchat router to use shared runner and event sink wiring
- [ ] Align pinocchio TUI + webchat snapshot/ConversationState handling in shared runner
- [ ] Document Moments migration plan (no code) to move webchat loop onto shared runner
