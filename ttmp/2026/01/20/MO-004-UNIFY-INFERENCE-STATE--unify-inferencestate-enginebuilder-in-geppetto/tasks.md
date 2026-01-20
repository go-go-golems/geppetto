# Tasks

## TODO

- [ ] Add tasks here

- [ ] Extract InferenceState into geppetto/pkg/inference/state (with StartRun/FinishRun/CancelRun, RunID, Turn, Eng)
- [ ] Define geppetto EngineBuilder contract/interface (no ConversationManager injection) and minimal engine+sink composition inputs
- [ ] Move ToolCallingLoop into geppetto/pkg/inference/core (or equivalent), decouple from go-go-mento webchat package
- [ ] Define Runner interface RunInference(ctx, seed) -> (turn, error) and Session implementation capturing State/Registry/LoopOpts/Persister
- [ ] Update go-go-mento webchat to use geppetto InferenceState/ToolCallingLoop/Runner types (remove local duplicates)
- [ ] Update pinocchio TUI/webchat to use geppetto inference core Session/Runner (replace pinocchio runner)
- [ ] Add a reference persister implementation(s): no-op persister, filesystem persister for debugging
- [ ] Add targeted tests for Session.RunInference single-pass and tool-loop paths (mock engine/tool registry)
- [ ] Document migration notes and any breaking API changes in MO-004 design/docs
- [ ] Migrate geppetto cmd/examples to use geppetto/pkg/inference/state + core.Session + builder.EngineBuilder
- [ ] Migrate pinocchio examples/commands that run agent/tool loops to use geppetto inference core (InferenceState + Session + EngineBuilder)
- [ ] Validate examples run paths (go run) and ensure tool-loop + event sinks still work
