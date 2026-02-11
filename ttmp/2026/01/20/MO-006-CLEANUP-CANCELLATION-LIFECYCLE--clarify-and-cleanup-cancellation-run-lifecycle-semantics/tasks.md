# Tasks

## TODO

- [ ] Add tasks here

- [ ] Document current lifecycle + cancellation (StartRun/FinishRun/CancelRun/HasCancel) across TUI/webchat
- [ ] Propose renamed/cleaned model: Conversation vs InferenceRun; define ownership and invariants
- [ ] Add tests/smoke scripts for cancel behavior (tmux + webchat)
- [x] Add minimal unit tests for core.Session: EventSinks injection + cancellation behavior (no real providers)
- [x] Add minimal unit test for toolhelpers.RunToolCallingLoop using a fake engine + a simple echo tool
- [x] Update MO-006 compendium to link to MO-004 inference testing playbook and note which commands exercise sinks/cancel/tool-loop
