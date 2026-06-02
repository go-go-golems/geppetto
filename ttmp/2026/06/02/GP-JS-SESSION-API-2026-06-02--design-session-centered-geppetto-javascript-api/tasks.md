# Tasks

## Done

- [x] Create ticket workspace.
- [x] Add session-centered JS API design document.
- [x] Add investigation diary.
- [x] Map existing Go `session.Session` lifecycle and invariants.
- [x] Map current JS agent, turn builder, turn-store, provider, and DTS surfaces.
- [x] Specify proposed session builder, `session.next()`, `session.fork()`, and `resumeLatest()` APIs.
- [x] Specify implementation phases, test strategy, risks, alternatives, and decision records.

## Follow-up implementation tasks

- [ ] Implement `agent.session()` and session wrappers in `pkg/js/modules/geppetto/api_session.go`.
- [ ] Implement `session.next().run()` / `runAsync()` using existing session execution machinery.
- [ ] Implement session builder storage/resume/base semantics.
- [ ] Implement `session.fork()`.
- [ ] Hard-cut public `gp.turn`, `agent.run`, and `agent.runAsync`.
- [ ] Update TypeScript declarations, docs, examples, and hard-cut public surface tests.
- [ ] Add real-provider session multi-turn and EventEmitter smoke examples.
- [ ] Add storage-enabled resume/fork integration tests after the Pinocchio adapter exists.
