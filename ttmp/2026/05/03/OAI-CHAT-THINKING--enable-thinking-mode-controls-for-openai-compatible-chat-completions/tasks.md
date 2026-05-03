# Tasks

## Completed analysis tasks

- [x] Create docmgr ticket workspace.
- [x] Add primary design/implementation guide.
- [x] Add chronological implementation diary.
- [x] Store DeepSeek thinking-mode source documentation in `sources/`.
- [x] Store redacted Wafer thinking-parameter probe in `sources/`.
- [x] Map current OpenAI chat request/settings/streaming architecture with file references.

## Follow-up implementation tasks

- [ ] Add `thinking` and `reasoning_effort` fields to the OpenAI Chat Completions request type.
- [ ] Add settings/flags/profile schema for OpenAI Chat Completions thinking toggle and effort.
- [ ] Wire settings and optional per-turn overrides into `MakeCompletionRequestFromTurn`.
- [ ] Add request JSON tests for enabled/disabled/high/max/omitted behavior.
- [ ] Validate live against Wafer DeepSeek-V4-Pro.
- [ ] Update Geppetto and Pinocchio docs with profile examples.
