# Tasks

## TODO


- [x] Design runtimeowner runner package API (Scheduler, Runner, Options, errors) in go-go-goja
- [x] Design geppetto runtime bridge API and integration points for all async VM boundaries
- [x] Implement runtimeowner runner core (Call/Post/Shutdown/IsClosed) with context cancellation and panic recovery
- [x] Add go-go-goja runner unit tests for success, cancellation, scheduler rejection, panic recovery, and closed-runner behavior
- [x] Add go-go-goja runner race/stress tests with concurrent Call/Post workloads
- [x] Create geppetto runtimebridge package with InvokeCallable and ToJSValue helpers backed by runner
- [x] Refactor geppetto module options/runtime init to accept and require runner, then initialize bridge
- [x] Migrate JS engine callback path (engines.fromFunction / jsCallableEngine.RunInference) to bridge
- [x] Migrate JS middleware callback path (including next callback and ctx payload conversion) to bridge
- [x] Migrate JS tool handler invocation and JS tool hook callbacks (before/after/onError) to bridge
- [x] Migrate async event collector payload conversion/listener invocation to bridge to eliminate off-owner value creation
- [x] Audit runAsync/start async paths to ensure no direct VM/callable access remains outside bridge
- [x] Add geppetto async regression tests for runAsync/start with JS callbacks and execute under -race
- [x] Update GP-03 design/planning docs with final file list, signatures, and implementation notes after code changes
