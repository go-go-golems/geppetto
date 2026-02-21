---
Title: 'Root cause analysis: missing reasoning items in follow-up tool calls'
Ticket: GP-05-REASONING-TOOL-CALL
Status: active
Topics:
    - backend
    - chat
    - openai
    - debugging
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../pkg/steps/ai/openai_responses/helpers.go
      Note: Contains the root-cause fix in request input assembly
    - Path: ../../../../../../pkg/steps/ai/openai_responses/helpers_test.go
      Note: Regression tests for reasoning/function_call adjacency
    - Path: 2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/sources/01-log-excerpt-followup-400.txt
      Note: Primary log evidence for provider 400
    - Path: 2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/sources/02-turn-block-sequence-pre-inference.txt
      Note: Turn block sequence showing missing reasoning predecessor
    - Path: 2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/sources/snapshots/gpt-5.log
      Note: Captured runtime debug log snapshot
ExternalSources: []
Summary: OpenAI Responses follow-up requests failed because request input kept older function_call items while dropping their required reasoning items.
LastUpdated: 2026-02-20T16:33:00-05:00
WhatFor: Diagnose and fix missing reasoning-item bug in follow-up tool-call turns.
WhenToUse: When debugging OpenAI Responses 400 errors about required reasoning items.
---


# Root cause analysis: missing reasoning items in follow-up tool calls

## Executive Summary

A follow-up request to OpenAI Responses failed with HTTP 400 on February 20, 2026 because input item `fc_0577e434...` (`function_call`) was sent without required reasoning item `rs_0577e434...`.

Root cause was in `pkg/steps/ai/openai_responses/helpers.go`: `buildInputItemsFromTurn` only preserved the latest reasoning block and dropped older reasoning blocks, but it still retained older `tool_call`/`function_call` items. This produced an invalid input sequence for Responses.

Implemented fix: switch to ordered block-walk processing so each reasoning block is preserved when it has a valid immediate follower (`llm_text` or `tool_call` chain), preventing orphaned function calls.

## Problem Statement

Observed production/debug error:

- `Item 'fc_0577e434078b396c006998cee71410819398e7a53dadd00bb2' of type 'function_call' was provided without its required 'reasoning' item: 'rs_0577e434078b396c006998cedf97c881938a4973922a4eab57'.`

Context from log evidence (`sources/01-log-excerpt-followup-400.txt`):

- Inference start: `2026-02-20T16:18:00.591...` with `turn_id=2a85dbb6-985d-4e45-a813-c31c1e95e0c8`
- Request summary: `input_items=10`
- Provider response: `status=400` with missing reasoning error above.

## Evidence Correlation

### 1) Turn snapshot contains multiple reasoning blocks

From `sources/02-turn-block-sequence-pre-inference.txt` for turn `2a85dbb6-985d-4e45-a813-c31c1e95e0c8`:

- Ordinal 7: `reasoning` id `rs_...df97...`
- Ordinal 8: `tool_call` item id `fc_...e714...`
- Ordinal 9: `tool_use` for same call
- Ordinal 10: newer `reasoning` id `rs_...e83a...`
- Ordinal 11: assistant `llm_text`
- Ordinal 12: user follow-up

### 2) Previous algorithm dropped older reasoning by design

Old logic in `pkg/steps/ai/openai_responses/helpers.go`:

- Located only `latestReasoningIdx`.
- Skipped all older `turns.BlockKindReasoning` in pre-context.
- Still appended pre-context `tool_call` and `tool_use` blocks.

Effect: request included `function_call` with `id=fc_...` but omitted required reasoning `rs_...df97...`.

### 3) “empty role” preview entries were diagnostic noise, not root cause

`engine.go` preview logger only prints `{role, parts}` and not item `type`, so item-based rows (`reasoning`, `function_call`, `function_call_output`) appear as `{"role":"","parts":[]}` in logs.

Reference: `pkg/steps/ai/openai_responses/engine.go:122-134`.

## Implemented Solution

### Code change

File changed: `pkg/steps/ai/openai_responses/helpers.go`

`buildInputItemsFromTurn` now processes blocks in order (single pass) instead of latest-reasoning partitioning:

- For each `reasoning` block:
  - If immediate follower is `llm_text`, emit:
    - `type:"reasoning"`
    - followed immediately by item-based assistant `type:"message"`
  - If immediate follower is `tool_call`, emit:
    - `type:"reasoning"`
    - followed by contiguous `tool_call`/`tool_use` chain as `function_call`/`function_call_output`
  - Otherwise omit that reasoning block (same safety behavior for invalid follower shapes).
- Non-reasoning blocks are still emitted as before.

This preserves required reasoning/function-call adjacency across the full historical turn context.

### Tests

File changed: `pkg/steps/ai/openai_responses/helpers_test.go`

- Updated assistant-context test to reflect ordered behavior:
  - `TestBuildInputItemsFromTurn_PreservesAssistantContextWithReasoningFollower`
- Added regression test reproducing multi-reasoning chain:
  - `TestBuildInputItemsFromTurn_PreservesReasoningForOlderFunctionCallChains`
  - Verifies older `function_call` is immediately preceded by its reasoning item.

Validation run:

- `go test ./pkg/steps/ai/openai_responses -count=1` -> PASS.

## Design Decisions

1. Preserve semantic chronology over “latest reasoning only” heuristics.
2. Keep reasoning omission guard when follower is invalid, to avoid known provider 400s.
3. Limit change scope to request-building layer; no timeline or turn-storage schema changes.

## Alternatives Considered

1. Keep latest-only strategy and selectively re-inject required older reasoning for tool calls.
- Rejected: brittle matching logic and easier to regress.

2. Strip all historical tool calls from follow-up context.
- Rejected: risks losing tool execution continuity and model context fidelity.

3. Attempt provider-side fallback/retry by removing orphaned function calls.
- Rejected: masks invariant break and may degrade response quality.

## Implementation Plan Status

1. Capture reproducible artifacts and evidence. Completed.
2. Correlate logs and DB turn history. Completed.
3. Patch request-builder logic. Completed.
4. Add regression tests and run package tests. Completed.
5. Re-run full inventory server scenario end-to-end. Pending in ticket task.

## Open Questions

1. Should we add a request preflight validator that asserts every `function_call` requiring reasoning has a matching predecessor before HTTP send?
2. Should debug preview logging include `type` and `id` to make invalid sequences obvious?

## References

- Code: `pkg/steps/ai/openai_responses/helpers.go`
- Tests: `pkg/steps/ai/openai_responses/helpers_test.go`
- Log evidence: `ttmp/2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/sources/01-log-excerpt-followup-400.txt`
- Turn evidence: `ttmp/2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/sources/02-turn-block-sequence-pre-inference.txt`
- Snapshot bundle: `ttmp/2026/02/20/GP-05-REASONING-TOOL-CALL--investigate-lost-reasoning-blocks-before-follow-up-tool-call-in-openai-responses/sources/snapshots`
