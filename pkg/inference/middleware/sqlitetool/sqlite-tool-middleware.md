---
Title: SQLite Tool Middleware
Slug: sqlite-tool-middleware
Short: Load a SQLite DB, inject schema/prompts, and expose a sql_query tool (read-only or read-write).
Topics:
- geppetto
- middlewares
- sqlite
- tools
SectionType: Guide
IsTopLevel: false
ShowPerDefault: true
---

## Overview

The SQLite Tool middleware attaches a SQLite database to a Turn and advertises a `sql_query` tool (instance-named) for querying that database. The tool description is precomputed once at middleware creation from the current database schema and any prompts found in a `_prompts` table, so no system blocks are injected at runtime anymore. This keeps the conversation cleaner and avoids duplicate prompt text across turns. It can be configured as read-only (default) or read-write.

## Key capabilities

- Precomputes a tool description that embeds the DB schema (excluding `_prompts`).
- Includes additional prompts from a `_prompts(prompt TEXT)` table in the description; optional per-turn prompts from `Turn.Data` may still be appended inside the tool description if desired.
- Registers a provider-advertised tool with instance-specific name, e.g. `sql_query` or `sql_query_finance`.
- Executes tool calls directly, returning tabular results (header + rows) or rows_affected for write statements.
- Read-only enforcement: DSN `mode=ro`, `PRAGMA query_only=ON`, and allowlist of SQL verbs (SELECT/WITH/EXPLAIN/PRAGMA).

## Architecture

- Data keys on `Turn.Data`:
  - `sqlite_dsn` (string): DSN/file for the attached SQLite database (overrides Config.DSN)
  - `sqlite_prompts` ([]string): optional extra snippets to be included in the tool description for this turn
  - `turns.DataKeyToolRegistry`: a ToolRegistry used by engines to advertise the `sql_query_*` tool

- Config (`sqlitetool.Config`):
  - `DSN string`: fallback DSN when `sqlite_dsn` is not provided
  - `DB DBLike`: optional pre-opened DB (implements `QueryContext`/`ExecContext`); not closed by middleware
  - `Name string`: instance label, used to derive the tool name: `sql_query` (empty) or `sql_query_<normalized-name>`
  - `ReadOnly bool`: default true; when true, sets `PRAGMA query_only=ON` and only allows SELECT/WITH/EXPLAIN/PRAGMA
  - `MaxRows int`: cap returned rows for SELECT-like statements (default 200)
  - `ExecutionTimeout time.Duration`: per-call timeout

- Middleware flow:
  1. Resolve DB from `Config.DB` or open from DSN / `sqlite_dsn`
  2. Precompute tool description once at middleware creation (schema + prompts)
  3. Register instance-specific tool definition in the Turn registry with the precomputed description
  4. After `next`, scan for pending `tool_call` blocks matching the instance tool name and execute them

## Example usage

```go
import (
  engpkg "github.com/go-go-golems/geppetto/pkg/inference/engine"
  "github.com/go-go-golems/geppetto/pkg/inference/middleware"
  "github.com/go-go-golems/geppetto/pkg/inference/session"
  "github.com/go-go-golems/pinocchio/pkg/middlewares/sqlitetool"
  "github.com/go-go-golems/geppetto/pkg/inference/tools"
  "github.com/go-go-golems/geppetto/pkg/turns"
)

// Build registry and Turn
reg := tools.NewInMemoryToolRegistry()
t := &turns.Turn{Data: map[string]any{turns.DataKeyToolRegistry: reg}}
// Attach DSN and optional prompts
t.Data["sqlite_dsn"] = "file:demo.db?mode=ro&immutable=1"
t.Data["sqlite_prompts"] = []string{"Return concise tabular results."}

// Compose middleware
cfg := sqlitetool.DefaultConfig()
cfg.Name = "finance"      // tool becomes sql_query_finance
cfg.ReadOnly = true        // allow only SELECT/WITH/EXPLAIN/PRAGMA
mw := sqlitetool.NewMiddleware(cfg)
runner, err := toolloop.NewEngineBuilder(
  toolloop.WithBase(baseEngine),
  toolloop.WithMiddlewares(mw),
).Build(ctx, "")
if err != nil {
  panic(err)
}

// Run inference
updated, _ := runner.RunInference(ctx, t)
```

## Tool name and prompts

- Tool is registered as `sql_query` when `Config.Name` is empty; otherwise `sql_query_<normalized-name>` with lowercased spaces → underscores.
- The schema and any access recommendations are included inside the tool description, not injected as system messages.
- Prompts are pulled from `_prompts` and can be augmented from `Turn.Data["sqlite_prompts"]`, all merged into the tool description.

## Read-only vs read-write

- Read-only (default):
  - Open DSN with `mode=ro`/`immutable=1` when possible
  - Set `PRAGMA query_only=ON`
  - Only allows `SELECT`, `WITH`, `EXPLAIN`, `PRAGMA`; others return `error: read-only mode ...`

- Read-write:
  - SELECT-like uses `QueryContext` and returns tabular results (capped by `MaxRows`)
  - Other statements use `ExecContext` and return `rows_affected=N`

## Schema and seed

- Ready-made examples:
  - `schema.sql`: creates `_prompts`, `authors`, `books`, plus an index
  - `seed.sql`: inserts helpful prompts and sample data

Initialize:

```sql
sqlite3 demo.db < schema.sql
sqlite3 demo.db < seed.sql
```

## Notes and best practices

- Keep the tool read-only in production unless a write path is explicitly needed.
- Consider setting `immutable=1` for static datasets.
- Avoid returning huge result sets; tune `MaxRows` and encourage summarization in prompts.
- Use instance `Name` for multiple databases to avoid tool name collisions (e.g., `sql_query_finance`, `sql_query_hr`).

## Writing good docs

This guide follows the internal `glaze help how-to-write-good-documentation-pages` guidance by providing purpose, architecture, data keys, examples, and repeatable setup steps.
