# Tasks

## TODO

- [ ] Decide canonical ID semantics: `SessionID` vs `conv_id` vs `run_id` (per-inference recommended)
- [ ] Pinocchio: change `session.SessionID` to stable `conv_id` and introduce per-inference `run_id`
- [x] Pinocchio: port `ConnectionPool` semantics (drop-on-error broadcast, idle stop)
- [x] Pinocchio: port `StreamCoordinator` abstraction (ordered callbacks; Redis stream version extraction + SEM fields)
- [x] Pinocchio: add per-conversation fallback cursor `seq` for transports without Redis XID
- [ ] Pinocchio: add `SessionManager` (rebuild-on-config signature changes; eviction loop; load-on-resume hooks)
- [x] Pinocchio: change middleware application order to reverse (match go-go-mento “outermost first”)
- [x] Geppetto: extend `toolhelpers.RunToolCallingLoop` to accept a `tools.ToolExecutor` (and optionally loop hooks)
- [ ] Pinocchio: implement step-mode (StepController + pause SEM events + continue/toggle endpoints)
