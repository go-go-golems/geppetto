---
Title: 'Analysis: OpenAI Responses helper context loss and streaming failure handling'
Ticket: PI-020-OPENAI-RESPONSES-BUGFIX
Status: active
Topics:
    - backend
    - bugfix
    - openai
    - responses
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Contains streaming tail that incorrectly returned success on SSE failures
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: Contains the pre-reasoning assistant skip logic root cause
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-10T18:18:10.604263577-05:00
WhatFor: ""
WhenToUse: ""
---


# Analysis

## Question

Did we already fix either of the two reported bugs in current `HEAD`?

## Result

No. Both bugs are still present as of 2026-02-10.

## Bug A: Assistant Context Dropped Before Reasoning

- File: `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/helpers.go`
- Area: `buildInputItemsFromTurn`, pre-context loop before latest reasoning.
- Current behavior:
  - `lastAssistantBeforeReasoning` is updated for each assistant text in the same loop where assistant blocks are conditionally skipped.
  - The branch checks `if i == lastAssistantBeforeReasoning { continue }` immediately after updating that value for assistant blocks.
  - Outcome: every assistant text prior to reasoning is skipped, not only the last one.
- Risk:
  - Multi-turn context loses assistant outputs that should remain in history.
  - Responses request history becomes incomplete and can change model behavior.

## Bug B: Streaming Errors Return Success

- File: `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/engine.go`
- Area: streaming SSE branch in `RunInference`.
- Current behavior:
  - `streamErr` is set on `event:error` or `event:response.failed`.
  - After loop exits, code still emits `events.NewFinalEvent(...)` and returns `nil` error.
- Risk:
  - Callers interpret failed streams as successful completions.
  - Incomplete output can be persisted as if request succeeded.

## Execution Plan

1. Add regression test for Bug A (fail first).
2. Fix Bug A and run focused tests.
3. Commit Bug A changes.
4. Add regression test for Bug B (fail first).
5. Fix Bug B and run focused tests.
6. Commit Bug B changes.
7. Update diary/changelog/tasks and attach related files.
