# Changelog

## 2026-03-17

- Created ticket `GP-42-REMOVE-ALLOWED-TOOLS` for removing `AllowedTools` from Geppetto core and standardizing on app-owned registry filtering.
- Added the primary design doc and a separate Manuel diary.
- Completed an evidence pass over Geppetto tool config, engine tool config, provider engines, tool loop bridging, JS builder options, and downstream registry filtering code.
- Wrote a detailed implementation guide explaining the current duplication, why core `AllowedTools` should be removed, what code paths must change, and how apps should continue filtering tools outside Geppetto.
- Added a ticket-local inventory script for quickly re-scanning `AllowedTools`-related code paths.
- Related the key evidence files to the ticket documents.
- Validated the ticket docs with `docmgr doctor`.
- Uploaded the bundle to reMarkable and verified the remote destination.

## 2026-03-17

Hardened repository lint/security workflows to ignore ticket workspaces and switched lint package discovery to tracked Go files only so ttmp and unrelated nested repos stop contaminating lint and gosec (commit 73b18910b0388e0b553792ac6702afdb5ad3d653).

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/.golangci.yml — Excluded ttmp from golangci-lint
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/Makefile — Scoped lint and gosec targets to tracked Go package directories


## 2026-03-17

Implemented Geppetto core AllowedTools removal, removed duplicated enforcement from provider preparation and executor logic, and updated JS/example/test surfaces (commit b15088f1ffd3893db286d90349b7a1abc1bef35a).

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/types.go — Removed mirrored engine.ToolConfig.AllowedTools
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/base_executor.go — Stopped executor-side allowlist rejection
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/config.go — Removed ToolConfig.AllowedTools and related helpers
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/openai/engine_openai.go — Stopped provider-side tool allowlist filtering


## 2026-03-17

Updated Temporal Relationships turn-inspector readback to stop exposing stale tool_config.allowed_tools after the Geppetto core removal and kept app-owned runtime registry filtering intact (commit da69b8f31f1e53884f857cb1f53cd5d68b71bc7e).

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/httpapi/run_turns_handlers.go — Removed allowed_tools from turn tool-config JSON payloads
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/ui/src/components/inspector/TurnToolContextCard.tsx — Removed stale allowed-tools badge from the inspector UI

