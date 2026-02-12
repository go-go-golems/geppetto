---
Title: Implementation Plan
Ticket: GP-006-JS-MIDDLEWARES
Status: active
Topics:
    - geppetto
    - javascript
    - goja
    - middleware
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-12T10:00:32-05:00
WhatFor: Implement phase-3 JS middleware authoring and mixed Go+JS middleware composition semantics.
WhenToUse: Use when implementing or reviewing middleware adapter behavior and ordering.
---

# Implementation Plan

## Goal

Deliver phase 3 middleware support in JS API:

- register middleware implemented in JS,
- compose JS and Go middleware predictably,
- preserve middleware ordering semantics from Go core.

## Scope

In scope:

- JS middleware adapter contracts.
- Mixed chain composition model.
- Error propagation and context metadata pass-through.
- JS smoke script running existing middleware example command.

Out of scope:

- Tool registry API (handled in builder/tools ticket).
- Turn mapper internals (handled in setup/turn ticket).

## Work Packages

1. Middleware function contract and signature
2. Adapter bridge JS function -> Go middleware
3. Chain order rules and conformance checks
4. Error wrapping and stack preservation
5. Middleware smoke script

## Deliverables

- Middleware API contract doc and order guarantees.
- Checklist for cross-language chain behavior.
- Runnable JS script that validates middleware example execution.

## Testing Plan

- Run middleware smoke script:
  - `node geppetto/ttmp/2026/02/12/GP-006-JS-MIDDLEWARES--js-api-middleware-composition/scripts/test_middleware_smoke.js`
- Inference-oriented middleware tests use:
  - `PINOCCHIO_PROFILE=gemini-2.5-flash-lite`

## Risks and Mitigations

- Risk: chain order mismatch between JS expectations and Go runtime.
  - Mitigation: document `Chain(h, m1, m2) == m1(m2(h))` rule in tests and API docs.
- Risk: bridge hides middleware-origin errors.
  - Mitigation: preserve JS function name and source location in wrapped errors.

## Exit Criteria

- Detailed task list completed.
- JS smoke script executed and logged in diary.
- Ordering and error rules explicitly documented.
