# Tasks

## TODO

- [ ] Pinocchio: design `EngineConfig` + `Signature()` for webchat composition
- [ ] Pinocchio: introduce go-go-mento-style `EngineBuilder` (`BuildConfig` / `BuildFromConfig`)
- [ ] Pinocchio: introduce `SubscriberFactory` (Redis vs in-memory)
- [ ] Pinocchio: refactor `getOrCreateConv` into a manager that rebuilds on signature change
- [ ] Pinocchio: remove per-route `build := func() ...` closures from `pinocchio/pkg/webchat/router.go`
- [ ] Pinocchio: switch `Conversation.Sink` to `events.EventSink` (so sinks can be wrapped)
- [ ] Pinocchio: remove `geppetto/pkg/toolbox` dependency from webchat runtime (tools should be registered via factories/registry)
- [ ] Pinocchio: remove/replace any remaining `toolbox` wiring in webchat and examples
- [ ] Tests: signature stability + rebuild behavior (profile change, override change)
- [ ] Moments: write a follow-up migration sketch (how Moments could adopt the same interfaces)
