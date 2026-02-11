# Tasks

## TODO

- [x] Pinocchio: design `EngineConfig` + `Signature()` for webchat composition
- [x] Pinocchio: introduce go-go-mento-style `EngineBuilder` (`BuildConfig` / `BuildFromConfig`)
- [x] Pinocchio: introduce `SubscriberFactory` (Redis vs in-memory)
- [x] Pinocchio: refactor `getOrCreateConv` into a manager that rebuilds on signature change
- [x] Pinocchio: remove per-route `build := func() ...` closures from `pinocchio/pkg/webchat/router.go`
- [x] Pinocchio: switch `Conversation.Sink` to `events.EventSink` (so sinks can be wrapped)
- [x] Pinocchio: remove `geppetto/pkg/toolbox` dependency from webchat runtime (tools should be registered via factories/registry)
- [x] Pinocchio: remove/replace any remaining `toolbox` wiring in webchat and examples
- [ ] Tests: signature stability + rebuild behavior (profile change, override change)
- [ ] Moments: write a follow-up migration sketch (how Moments could adopt the same interfaces)
