# Tasks

## Done

- [x] Create ticket workspace and design document.
- [x] Investigate current JS turn builder and built turn wrapper.
- [x] Investigate Go `Turn.Clone()` and session continuation ID semantics.
- [x] Write intern-oriented turn append/continuation API guide.

## Follow-up implementation tasks

- [ ] Implement `gp.turn(existingTurn)` using Go-owned wrapper validation.
- [ ] Clear copied `Turn.ID` for continuation builders while preserving `turn.clone()` identity.
- [ ] Add runtime tests for immutable append, ID semantics, wrapper rejection, and multimodal continuation.
- [ ] Update TypeScript declarations and JS API docs.
- [ ] Update multi-turn examples to use the continuation API.
