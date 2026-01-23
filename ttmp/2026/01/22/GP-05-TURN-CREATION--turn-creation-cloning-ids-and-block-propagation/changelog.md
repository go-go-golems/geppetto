# Changelog

## 2026-01-22

- Initial workspace created


## 2026-01-22

Capture intent and begin mapping turn cloning/ID propagation paths; document current behavior (Turn.ID stable across inferences; block TurnID/InferenceID not stamped by default).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/session/session.go — StartInference stamps turn metadata
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/turns/types.go — Block.TurnID removed; Turn.Clone added for safe cloning


## 2026-01-22

Write deep analysis of follow-up turn creation within a session: where turns are cloned, how Turn.ID is (not) regenerated, current gaps in Block.TurnID stamping, and proposal for block-level InferenceID attribution.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/analysis/01-turn-creation-turnid-propagation.md — Primary analysis document
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/reference/01-diary.md — Frequent investigation diary


## 2026-01-22

Add appendix mapping where blocks are appended/modified (middlewares, tool loop, engines) and document current Block.TurnID stamping gaps.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/analysis/01-turn-creation-turnid-propagation.md — Added Appendix section on block mutations and TurnID stamping
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/reference/01-diary.md — Added Step 6 diary entry for the inventory

## 2026-01-23

Simplify turn creation + cloning: remove `Block.TurnID`, centralize turn cloning, and centralize follow-up prompt turn creation in the Session API.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/turns/types.go — Remove Block.TurnID; add `Turn.Clone()`
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/session/session.go — Add `AppendNewTurnFromUserPrompt(s)`; run `StartInference` in-place on latest appended turn
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Webchat now uses `AppendNewTurnFromUserPrompt`
