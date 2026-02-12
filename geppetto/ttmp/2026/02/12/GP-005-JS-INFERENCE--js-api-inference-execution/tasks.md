# Tasks

## TODO

- [x] Define JS API for engine/session lifecycle (`createEngine`, `createSession`, `run`, `runAsync`, `cancel`)
- [x] Specify single-active-inference behavior and rejection semantics for concurrent starts
- [x] Define async handle contract (`wait`, `cancel`, state flags)
- [x] Document context timeout/cancellation mapping across Goja boundary
- [x] Specify standardized inference error envelope for JS callers
- [x] Create JS smoke script that executes `cmd/examples/simple-inference`
- [x] Ensure inference smoke script sets `PINOCCHIO_PROFILE=gemini-2.5-flash-lite`
- [x] Run inference smoke script and record pass/skip/failure details in diary
- [x] Add follow-up tasks for async cancellation stress tests
- [x] Update changelog with inference baseline completion
