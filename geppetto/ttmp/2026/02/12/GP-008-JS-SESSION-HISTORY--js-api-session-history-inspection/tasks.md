# Tasks

## TODO


- [x] Define JS contract for session history APIs: session.turns(), session.turnCount(), session.getTurn(index)
- [x] Define snapshot immutability policy and cloning semantics for returned turn history
- [x] Implement history API methods on session object in pkg/js/modules/geppetto/api.go
- [x] Specify index behavior (0-based, negative indices, and out-of-range handling)
- [x] Add codec helpers for encoding slices of turns to native JS arrays
- [x] Add optional range/slice helper (session.turnsRange(start,end)) if low-cost
- [x] Add unit tests for multi-turn append/run with history inspection
- [x] Add unit tests proving JS-side mutation does not mutate stored session history
- [x] Create ticket JS script to probe/print multi-turn history behavior
- [x] Run go test + script, then update diary/changelog with outputs
