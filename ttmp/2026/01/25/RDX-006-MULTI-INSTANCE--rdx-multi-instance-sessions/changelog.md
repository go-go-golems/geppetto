# Changelog

## 2026-01-25

- Initial workspace created


## 2026-01-25

Step 1: expanded spec and created tasks for multi-instance work

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/25/RDX-006-MULTI-INSTANCE--rdx-multi-instance-sessions/analysis/01-multi-instance-sessions-spec.md — Expanded spec
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/25/RDX-006-MULTI-INSTANCE--rdx-multi-instance-sessions/tasks.md — Task list


## 2026-01-25

Step 2: add tail timeout/count and dual-mode output (commit 5a8be95)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/rdx/cmd/rdx/commands.go — Tail flags and dual-mode
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/rdx/cmd/rdx/tail_runtime.go — Plain tail output


## 2026-01-25

Step 3: implement session registry, selectors, and sessions commands (commit b241104)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/rdx/cmd/rdx/selector_runtime.go — Selector resolution
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/rdx/cmd/rdx/sessions_commands.go — Sessions commands
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/rdx/pkg/rtk/session_registry/registry.go — Registry implementation


## 2026-01-25

Step 4: rename selector flag to avoid Glazed conflict

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/rdx/cmd/rdx/commands.go — Use --instance-select instead of --select


## 2026-01-25

Step 4: rename selector flag to --instance-select (commit 29fddb7)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/rdx/cmd/rdx/commands.go — Renamed selector flags


## 2026-01-25

Step 5: fix watch argument order for optional selector (commit e379358)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/rdx/cmd/rdx/commands.go — Reordered watch args


## 2026-01-25

Step 6: Add debug-raw command for raw SocketCluster frames (commit 63798fd)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/rdx/cmd/rdx/debug_raw_runtime.go — Raw debug runtime with timeout/count

