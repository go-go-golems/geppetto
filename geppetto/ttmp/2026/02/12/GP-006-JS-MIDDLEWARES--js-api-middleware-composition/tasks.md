# Tasks

## TODO

- [x] Define JS middleware function signature and expected return contract
- [x] Define bridge adapter from JS middleware callback to Go `middleware.Middleware`
- [x] Specify immutable vs mutable turn handling policy across middleware boundaries
- [x] Document exact chain ordering semantics for mixed Go+JS middleware
- [x] Define middleware error wrapping with JS source context preservation
- [x] Create JS smoke script for `cmd/examples/middleware-inference`
- [x] Ensure middleware smoke script sets `PINOCCHIO_PROFILE=gemini-2.5-flash-lite`
- [x] Run middleware smoke script and capture output in diary
- [x] Add conformance checklist for mixed-chain regression tests
- [x] Update changelog with middleware planning baseline
