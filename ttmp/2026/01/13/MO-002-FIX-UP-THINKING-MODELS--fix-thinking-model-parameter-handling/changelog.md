# Changelog

## 2026-01-13

- Initial workspace created


## 2026-01-13

Step 1: Capture GPT-5/o-series parameter errors and begin chat-mode gating (no commit).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai/helpers.go — Introduce reasoning-model gating for max_completion_tokens and sampling params.


## 2026-01-13

Step 2: Exclude GPT-5/o-series sampling params in Responses helper (no commit).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/helpers.go — Expand allowSampling gating for reasoning models.


## 2026-01-13

Step 3: Run GPT-5 Responses example in tmux; completed successfully (no code changes).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/openai-tools/main.go — Validated GPT-5 responses run using server-tools mode.


## 2026-01-13

Step 4: Document Responses thinking stream event flow and pinocchio UI gap (no code changes).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/01-responses-thinking-stream-event-flow.md — Analysis of event path and UI mismatch.


## 2026-01-13

Step 5: Render thinking events in pinocchio chat UI (commit 7b38883).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/ui/backend.go — Handle thinking-started/ended and reasoning summary deltas.

