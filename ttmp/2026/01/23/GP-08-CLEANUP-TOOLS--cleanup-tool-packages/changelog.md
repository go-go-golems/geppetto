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


## 2026-01-23

Step 5: Move toolblocks into turns and update downstream imports (geppetto cbb5058; moments 3fa89be6)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/turns/toolblocks/toolblocks.go — New canonical home for tool_call/tool_use Turn helpers
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/moments/backend/pkg/webchat/loops.go — Import path update


## 2026-01-23

Step 6: Delete toolhelpers and update docs to toolloop (geppetto 8e0614d, e4b54a7)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/doc/topics/07-tools.md — Updated docs to LoopConfig + tools.ToolConfig
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/toolhelpers/helpers.go — Deleted legacy loop helper


## 2026-01-23

Step 7: Docs sweep + end-to-end wiring snippet + Pinocchio snippet updates (geppetto d55235f; pinocchio 4b7c3f3)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/doc/topics/07-tools.md — Added end-to-end wiring snippet
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/agents/simple-chat-agent.md — Updated docs to toolloop/session


## 2026-01-23

Step 8: Upload updated GP-08 bundle to reMarkable (/ai/2026/01/23/GP-08-CLEANUP-TOOLS)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-08-CLEANUP-TOOLS--cleanup-tool-packages/reference/01-diary.md — Recorded reMarkable upload commands and remote path

