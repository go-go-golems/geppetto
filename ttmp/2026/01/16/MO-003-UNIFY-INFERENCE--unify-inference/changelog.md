# Changelog

## 2026-01-16

- Initial workspace created


## 2026-01-16

Step 1: add shared inference runner and migrate TUI backend (commit 2df3b2c)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/inference/runner/runner.go — Shared runner for TUI


## 2026-01-16

Step 2: migrate pinocchio webchat to shared runner (commit 0fdcb56)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Runner-based webchat loop


## 2026-01-16

Step 5: make system prompt middleware idempotent (commit 4594a4b)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/middleware/systemprompt_middleware.go — Skip reinsertion when systemprompt metadata present


## 2026-01-16

Step 7: analyze go-go-mento webchat conversation manager and Run alignment

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/analysis/11-go-go-mento-webchat-conversation-manager-and-run-alignment.md — Detailed architecture analysis
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/go-go-mento/go/pkg/webchat/conversation_manager.go — Lifecycle and engine recomposition


## 2026-01-16

Step 8: draft InferenceState + EngineBuilder core design

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/design-doc/03-inferencestate-enginebuilder-core-architecture.md — Design doc
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/go-go-mento/go/pkg/webchat/inference_state.go — InferenceState reused as core


## 2026-01-20

Step 10: refine inference core Session API (Runner interface, persister runID, move core types to geppetto)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/design-doc/03-inferencestate-enginebuilder-core-architecture.md — Updated API and ownership decisions


## 2026-01-25

Ticket closed

