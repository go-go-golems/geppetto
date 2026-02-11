---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/session/session.go
      Note: ErrSessionEmptyTurn check
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Seed turn creation
    - Path: pinocchio/pkg/webchat/router.go
      Note: Start run appends prompt and triggers StartInference
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Seed turn missing when starting run

## Summary
Starting a webchat run with an empty prompt fails with `session has no seed turn (or seed turn is empty)`.

## Impact
- First run can fail when `/chat` is called with an empty prompt (prompt_len=0).
- Webchat cannot start inference without a non-empty seed turn.

## Observed behavior
The server logs show `start run failed` with `session has no seed turn (or seed turn is empty)` immediately after the request.

## Expected behavior
Webchat should seed the session with a system prompt so inference can start even if the user prompt is empty.

## Logs
```
2026-02-02T23:31:05.447215664-05:00 ERR start run failed error="session has no seed turn (or seed turn is empty)" component=webchat conv_id=38b904cb-2dd4-449c-9603-2e426e554496 run_id=a48062f3-86b7-4062-a425-d57bae941103 session_id=a48062f3-86b7-4062-a425-d57bae941103
2026-02-02T23:31:10.694466864-05:00 INF ws connect request component=webchat conv_id=38b904cb-2dd4-449c-9603-2e426e554496 profile=default remote=127.0.0.1:33670
2026-02-02T23:31:10.69451189-05:00 INF ws joining conversation component=webchat conv_id=38b904cb-2dd4-449c-9603-2e426e554496 profile=default remote=127.0.0.1:33670
2026-02-02T23:31:10.695330165-05:00 INF ws connected component=webchat conv_id=38b904cb-2dd4-449c-9603-2e426e554496 profile=default remote=127.0.0.1:33670
2026-02-02T23:31:11.155994936-05:00 INF /chat received component=webchat conv_id=38b904cb-2dd4-449c-9603-2e426e554496 profile=default prompt_len=0
2026-02-02T23:31:11.15753841-05:00 INF starting run loop component=webchat conv_id=38b904cb-2dd4-449c-9603-2e426e554496 idempotency_key=73f19664-fa26-4d5b-b5a8-a1e0e1bf42ae run_id=a48062f3-86b7-4062-a425-d57bae941103 session_id=a48062f3-86b7-4062-a425-d57bae941103
2026-02-02T23:31:11.157563036-05:00 ERR start run failed error="session has no seed turn (or seed turn is empty)" component=webchat conv_id=38b904cb-2dd4-449c-9603-2e426e554496 run_id=a48062f3-86b7-4062-a425-d57bae941103 session_id=a48062f3-86b7-4062-a425-d57bae941103
```

## Suspected cause
- `Session.StartInference` fails when the latest turn has zero blocks.
- `AppendNewTurnFromUserPrompt` does not append a block when the prompt is empty.
- Webchat creates a seed turn with no blocks and does not inject a system prompt into the seed turn before starting inference.

## Proposed fix
- Seed the session with a system prompt block derived from the profile (default: "You are an assistant").
- Ensure each profile has a non-empty system prompt configured so the system prompt middleware is always present.
