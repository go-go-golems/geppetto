# Tasks

## TODO

- [x] Read `moments/backend` StepController code and summarize responsibilities + API surface
- [x] Identify what “a step” means in Moments (pause boundary definition)
- [x] Map Moments StepController semantics onto Geppetto post-MO-007 architecture (session + context sinks + tool loop)
- [x] Propose integration options and recommend a design (breaking changes allowed)
- [x] Pinocchio forwarder: map Geppetto EventDebuggerPause (debugger.pause) into SEM frames (pause_id/phase/summary/deadline)
- [x] Create new geppetto/pkg/inference/toolloop package: Loop struct (RunLoop) + options WithEngine/WithRegistry/WithConfig + cancellation-safe StepController
- [x] Refactor geppetto tool loop: move logic from toolhelpers.RunToolCallingLoop into toolloop.Loop.RunLoop and publish Geppetto-native debugger.pause events at pause points
- [x] Pinocchio webchat: wire dev-gated continue handler directly to shared toolloop.StepController (pause_id -> Continue), not via session.Session/ExecutionHandle
- [x] Pinocchio webchat: add dev-gated endpoints to enable/disable step mode (per session) and to continue by pause_id via shared toolloop.StepController
- [x] Tests: toolloop StepController wait/continue/cancel/disable + toolloop Loop pause-point behavior (after_inference when tools pending; after_tools)
- [x] Geppetto events: add EventDebuggerPause (type=debugger.pause) with session_id/inference_id/turn_id metadata and payload fields pause_id/phase/summary/deadline_ms/extra
- [x] Session runner: switch ToolLoopEngineBuilder runner to use toolloop.New(...).RunLoop(...) instead of toolhelpers.RunToolCallingLoop
