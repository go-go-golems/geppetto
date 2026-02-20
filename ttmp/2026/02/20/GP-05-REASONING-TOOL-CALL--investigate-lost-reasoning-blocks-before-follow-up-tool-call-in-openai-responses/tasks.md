# Tasks

## In Progress

- [ ] Re-run the hypercard inventory server scenario and verify no 400 from OpenAI Responses for the same follow-up turn shape.

## Done

- [x] Capture a local snapshot of runtime artifacts (`/tmp/gpt-5.log`, `/tmp/timeline3.db`, `/tmp/turns.db`) into the ticket.
- [x] Reconstruct the failing turn block sequence from `turns.db` and correlate it with the OpenAI 400 log.
- [x] Identify root cause in `buildInputItemsFromTurn` where older `reasoning` items were dropped while their `function_call` items were retained.
- [x] Implement request-builder fix to preserve reasoning/function-call adjacency across the full turn history.
- [x] Add regression tests for multi-reasoning + tool-call chains.
