# Tasks

## Done

- [x] Create ticket workspace and design document.
- [x] Investigate current JS turn builder and built turn wrapper.
- [x] Investigate Go `Turn.Clone()` and session continuation ID semantics.
- [x] Write intern-oriented turn append/continuation API guide.
- [x] Implement `gp.turn(existingTurn)` using Go-owned wrapper validation.
- [x] Clear copied `Turn.ID` for continuation builders while preserving `turn.clone()` identity.
- [x] Add runtime tests for immutable append, ID semantics, wrapper rejection, and multimodal continuation.
- [x] Update TypeScript declarations and JS API docs.
- [x] Update multi-turn examples to use the continuation API.

## Follow-up implementation tasks

- [ ] Consider convenience methods such as `turn.appendUser(...)` only if builder-based continuation proves too verbose.
- [ ] Design `gp.turn.fromJSON(...)` separately if plain-object turn import becomes necessary.
