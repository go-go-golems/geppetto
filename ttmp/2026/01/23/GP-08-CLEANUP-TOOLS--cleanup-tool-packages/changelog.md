# Changelog


## 2026-01-23

Step 1: Add diary + restructure tasks into step-by-step execution plan

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-08-CLEANUP-TOOLS--cleanup-tool-packages/reference/01-diary.md — Created and started the implementation diary
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-08-CLEANUP-TOOLS--cleanup-tool-packages/tasks.md — Converted generic TODOs into an explicit execution plan


## 2026-01-23

Step 1: Add diary + restructure tasks into step-by-step execution plan (commit ed48dec)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-08-CLEANUP-TOOLS--cleanup-tool-packages/reference/01-diary.md — Diary step 1
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-08-CLEANUP-TOOLS--cleanup-tool-packages/tasks.md — Execution plan


## 2026-01-23

Step 2: Move toolloop.EngineBuilder to toolloop/enginebuilder + migrate downstream call sites (geppetto fe9c0af; pinocchio cc40488; moments 20a6d194)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/toolloop/enginebuilder/builder.go — New package + implementation
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/moments/backend/pkg/webchat/engine.go — Cutover to new import path
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Cutover to new import path


## 2026-01-23

Step 3: Split loop orchestration (LoopConfig) from tool policy (tools.ToolConfig) and update downstream call sites (geppetto 9ec5cdaa; pinocchio dc6053a; moments aa0d50ca)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/toolloop/loop.go — Loop config + tool config split and Turn.Data mapping
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/tools/config.go — Ergonomic With* helpers
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/moments/backend/pkg/webchat/loops.go — Pass LoopConfig + tools.ToolConfig
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Pass LoopConfig + tools.ToolConfig


## 2026-01-23

Step 4: Move toolcontext into tools (WithRegistry/RegistryFrom), migrate imports, delete toolcontext (geppetto d6f1baa; pinocchio 2fdfcc9; moments 6cad48fd)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/toolcontext/toolcontext.go — Deleted
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/tools/context.go — New canonical context helpers
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/moments/backend/pkg/webchat/router.go — Updated to tools.WithRegistry
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/middlewares/sqlitetool/middleware.go — Updated to tools.RegistryFrom

