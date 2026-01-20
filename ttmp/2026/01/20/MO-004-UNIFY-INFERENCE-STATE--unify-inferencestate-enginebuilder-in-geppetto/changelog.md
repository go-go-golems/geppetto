# Changelog

## 2026-01-20

- Initial workspace created


## 2026-01-20

Step 1: create ticket workspace, diary, and migrate inference-core design doc

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/design-doc/03-inferencestate-enginebuilder-core-architecture.md — Design doc moved from MO-003
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/reference/01-diary.md — New MO-004 diary


## 2026-01-20

Step 2: implement geppetto InferenceState + Session runner (commit 453e6af)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/core/session.go — New shared Session runner
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/state/state.go — New shared InferenceState


## 2026-01-20

Step 3: analyze moments router migration (commit 8715e27)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/analysis/01-moments-webchat-router-migration-to-geppetto-inferencestate-session.md — Migration analysis


## 2026-01-20

Step 4: unify provider engine sinks via context (commit 3206cef)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/engine.go — Attach configured sinks to ctx and publish via context


## 2026-01-20

Step 9: migrate geppetto examples to EngineBuilder + Session (commit e009123)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/internal/examplebuilder/builder.go — Example EngineBuilder implementation
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/core/session.go — RunInferenceStarted + cancel behavior


## 2026-01-20

Step 10: pinocchio agent example uses EngineBuilder + Session (commit 03a3043)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go — Tool loop via geppetto core.Session + InferenceState
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/inference/enginebuilder/parsed_layers.go — ParsedLayers-based EngineBuilder


## 2026-01-20

Step 11: pinocchio webchat + TUI migrated to InferenceState + Session (commit 550b073)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/ui/backend.go — TUI backend now uses InferenceState + Session
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Seed construction + Session tool-loop


## 2026-01-20

Step 12: check off example migration tasks (commit 7730444)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/tasks.md — Checked off tasks 11-13


## 2026-01-20

Step 14: remove unused pinocchio runner package (commit 1a835e5)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/inference/runner/runner.go — Deleted; all call sites use geppetto core.Session now


## 2026-01-20

Step 15: check off core migration tasks (commit cade166)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/tasks.md — Checked off tasks 2

