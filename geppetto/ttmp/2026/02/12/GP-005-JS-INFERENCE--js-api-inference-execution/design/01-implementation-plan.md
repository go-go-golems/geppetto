---
Title: Implementation Plan
Ticket: GP-005-JS-INFERENCE
Status: active
Topics:
    - geppetto
    - javascript
    - goja
    - inference
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-12T10:00:32-05:00
WhatFor: Implement phase-2 JS API inference lifecycle APIs and validation tests.
WhenToUse: Use when implementing or validating JS session run/runAsync/cancel behavior.
---

# Implementation Plan

## Goal

Deliver phase 2 inference execution from JS:

- create engine/session from JS API,
- run blocking and async inference,
- preserve session invariants and cancellation behavior.

All inference tests for this ticket must run with:

- `PINOCCHIO_PROFILE=gemini-2.5-flash-lite`

## Scope

In scope:

- JS wrappers for `createEngine`, `createSession`, `run`, `runAsync`, `cancel`.
- Minimal error taxonomy for inference calls.
- One or more smoke tests that execute existing inference examples.

Out of scope:

- JS middleware implementation details.
- Tool registry orchestration.

## Work Packages

1. Engine/session JS wrapper contract
2. Sync run path
3. Async run + cancellation path
4. Error/timeout surface
5. Inference smoke scripts using required profile

## Deliverables

- Inference API shape and validation checklist.
- Command matrix for smoke tests (single-pass baseline).
- Runnable JS smoke script under ticket `scripts/`.

## Testing Plan

- Run inference smoke script:
  - `node geppetto/ttmp/2026/02/12/GP-005-JS-INFERENCE--js-api-inference-execution/scripts/test_inference_smoke.js`
- Script enforces `PINOCCHIO_PROFILE=gemini-2.5-flash-lite` in environment.

## Risks and Mitigations

- Risk: provider credentials unavailable in CI/local run.
  - Mitigation: mark as skipped with explicit reason in output; fail only for non-auth runtime errors.
- Risk: async cancellation race conditions.
  - Mitigation: include explicit cancellation smoke cases in follow-up implementation ticket.

## Exit Criteria

- Detailed task list is complete.
- JS inference script executed and result captured in diary.
- Profile requirement is explicitly documented and used.
