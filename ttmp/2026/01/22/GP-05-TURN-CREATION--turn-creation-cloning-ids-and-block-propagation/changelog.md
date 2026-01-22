# Changelog

## 2026-01-22

- Initial workspace created


## 2026-01-22

Capture intent and begin mapping turn cloning/ID propagation paths; document current behavior (Turn.ID stable across inferences; block TurnID/InferenceID not stamped by default).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/session/session.go — StartInference clones and stamps turn metadata
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/turns/types.go — AppendBlock does not set Block.TurnID


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

