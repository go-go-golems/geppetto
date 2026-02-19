---
Title: Implementation Plan
Ticket: GP-008-JS-SESSION-HISTORY
Status: active
Topics:
    - geppetto
    - goja
    - javascript
    - session
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Plan for exposing session history inspection APIs to JS
LastUpdated: 2026-02-12T11:03:45.592161464-05:00
WhatFor: Implement a safe JS-visible session history API for multi-turn inspection and replay workflows.
WhenToUse: Use when implementing or reviewing JS APIs that expose session turn history and transcript snapshots.
---

# Implementation Plan

## Goal

Expose full session history inspection from the JS `geppetto` module, not just `latest()`, while preserving session integrity.

## Problem Statement

Current JS session APIs allow:

- append turns
- run inference
- inspect `latest()`

But they do not expose an official history surface. Users can run multi-turn flows but cannot query complete turn history from JS without maintaining a parallel history themselves.

## Target API

- `session.turns()` -> returns all turns as cloned snapshots.
- `session.turnCount()` -> returns current number of turns.
- `session.getTurn(index)` -> returns cloned snapshot at index, or `null` if out of range.
- Optional: `session.turnsRange(start, end)` if implementation cost is low and behavior is clear.

## Semantics

- Returned turns are snapshots (defensive clones), not mutable references to internal `session.Turns`.
- Indexing is 0-based.
- Negative indices are allowed only if explicitly documented; default behavior is out-of-range -> `null`.
- History order is append order.

## Scope

In scope:

- API methods on JS session wrapper.
- Encoding helpers for `[]*turns.Turn` to JS values.
- Unit tests covering multi-turn growth and immutability.
- One JS probe script in ticket `scripts/` to validate behavior.

Out of scope:

- Persistence to disk (covered by transcript-persistence phase).
- New inference core session semantics.

## Implementation Approach

1. Extend `newSessionObject` in `pkg/js/modules/geppetto/api.go` with new methods.
2. Add helper in codec path to encode turn slices consistently.
3. Ensure clone-on-read behavior before conversion to JS values.
4. Add tests in `pkg/js/modules/geppetto/module_test.go` for:
   - multi-turn append/run history count,
   - `getTurn` bounds behavior,
   - mutation safety (JS changes do not mutate internal history).

## Testing Plan

- `go test ./pkg/js/modules/geppetto -count=1`
- Ticket probe script:
  - `node geppetto/ttmp/2026/02/12/GP-008-JS-SESSION-HISTORY--js-api-session-history-inspection/scripts/test_session_history_smoke.js`

## Risks and Mitigations

- Risk: returning live references allows accidental corruption.
  - Mitigation: always clone before returning.
- Risk: large histories can increase allocation cost.
  - Mitigation: keep API simple now; add range/pagination only if needed.

## Exit Criteria

- New session history methods available and documented.
- Tests prove correct behavior and immutability guarantees.
- Smoke script passes and is recorded in diary/changelog.
