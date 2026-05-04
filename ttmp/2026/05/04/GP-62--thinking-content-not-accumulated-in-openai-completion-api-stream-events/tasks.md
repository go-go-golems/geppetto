# Tasks

## TODO

- [x] Add tasks here

- [x] Add Completion field to EventReasoningTextDelta (or document why it doesn't need one)
- [ ] Fix runtime_debug_feature.go: pass accumulated content, not just delta, in EventReasoningTextDelta handler
- [x] Add EventReasoningTextDelta handling to JS api_events.go encoder
- [ ] Verify timeline ProjectTimeline accumulation works end-to-end with sessionstream
- [ ] Add integration test for thinking content accumulation across OpenAI completion API
- [x] Consider unifying EventReasoningTextDelta and EventThinkingPartial into a single event type
- [x] Switch runtime_debug_feature.go from EventReasoningTextDelta to EventThinkingPartial
- [x] Delete NewReasoningTextDelta emission from openai/engine_openai.go
- [x] Delete NewReasoningTextDelta emissions from openai_responses/engine.go
- [x] Delete EventReasoningTextDelta type, constant, constructor, and NewEventFromJson case from chat-events.go
- [x] Audit and delete EventReasoningTextDone if no consumers remain
- [x] Compile and run tests across geppetto, pinocchio, coinvault
