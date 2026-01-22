# Tasks

## TODO

- [x] Decide canonical ID semantics: `SessionID == conv_id` (stable); `InferenceID` is unique per `RunInference` call (stored in `geppetto.inference_id@v1`)
- [ ] Pinocchio: set `session.SessionID` to stable `conv_id` and propagate `geppetto.inference_id@v1` per inference
- [x] Pinocchio: port `ConnectionPool` semantics (drop-on-error broadcast, idle stop)
- [x] Pinocchio: port `StreamCoordinator` abstraction (ordered callbacks; Redis stream version extraction + SEM fields)
- [x] Pinocchio: add per-conversation fallback cursor `seq` for transports without Redis XID
- [ ] Pinocchio: add `SessionManager` (rebuild-on-config signature changes; eviction loop; load-on-resume hooks)
- [x] Pinocchio: change middleware application order to reverse (match go-go-mento “outermost first”)
- [x] Geppetto: extend `toolhelpers.RunToolCallingLoop` to accept a `tools.ToolExecutor` (and optionally loop hooks)
- [x] Geppetto: add `turns.KeyTurnMetaInferenceID` and populate/read it (session/runner + engines + middleware)
- [ ] Pinocchio: implement step-mode (StepController + pause SEM events + continue/toggle endpoints)
