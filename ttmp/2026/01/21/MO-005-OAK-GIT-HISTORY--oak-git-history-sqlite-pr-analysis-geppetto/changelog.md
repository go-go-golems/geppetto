# Changelog

## 2026-01-21

- Initial workspace created

- Step 2: Implemented cmd/oakgitdb MVP (git + oak + Go typed call edges) and generated various/pr-vs-origin-main.db.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/oakgitdb/main.go — CLI entrypoint
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/analysis/oakgitdb/builder.go — Schema + ingestion pipeline
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/21/MO-005-OAK-GIT-HISTORY--oak-git-history-sqlite-pr-analysis-geppetto/various/pr-vs-origin-main.db — Generated DB artifact

## 2026-01-21

Step 3: Moved oakgitdb code+docs into standalone repo directory oak-git-db/ (added go.mod, updated go.work, left ticket doc stubs).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/oak-git-db/cmd/oakgitdb/main.go — Standalone CLI
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/oak-git-db/docs/usage.md — Standalone usage docs
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/oak-git-db/pkg/oakgitdb/builder.go — Standalone builder


## 2026-01-21

Step 4: Committed oak-git-db repo (e417b29), uploaded bundled PDF to reMarkable (/ai/2026/01/21/MO-005-OAK-GIT-HISTORY), and imported markdown docs into ticket sources/local.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/oak-git-db/docs/design.md — Design doc
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/oak-git-db/docs/implementation.md — Implementation guide
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/oak-git-db/docs/usage.md — Usage guide


## 2026-01-21

Added task list for multi-repo DB support (geppetto + pinocchio): multi-root build + repo namespacing/schema changes.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/21/MO-005-OAK-GIT-HISTORY--oak-git-history-sqlite-pr-analysis-geppetto/tasks.md — Track multi-repo DB work


## 2026-01-21

Step 5: Implemented multi-repo DB support in oak-git-db (repeatable --repo, schema_version=2 with repo_id-scoped commits/paths). Commit b6e0313.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/oak-git-db/cmd/oakgitdb/main.go — Repeatable --repo flag
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/oak-git-db/docs/usage.md — Multi-repo usage + PR selection queries
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/oak-git-db/pkg/oakgitdb/builder.go — Multi-repo loop + schema changes


## 2026-02-25

Ticket closed

