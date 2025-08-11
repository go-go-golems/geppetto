---
Title: Agent Mode Middleware
Slug: agent-mode-middleware
Short: Configure, switch, and record agent modes with YAML prompts and tool restrictions.
Topics:
- geppetto
- middlewares
- agent-modes
- tools
- yaml
SectionType: Guide
IsTopLevel: false
ShowPerDefault: true
---

## Overview

The Agent Mode middleware lets you define named “modes” for an agent, each with:

- A prompt snippet to inject
- A list of allowed tools to constrain execution
- Optional persistence of mode transitions (per-run audit trail)

It also enables in-band mode switching using YAML. The assistant can output a fenced YAML block describing a proposed switch; the middleware parses, validates, and applies it.

## Key capabilities

- Inject a single, mode-specific user prompt (no system blocks)
- Enforce allowed tools per mode (passed as a hint to tool middleware)
- Parse YAML fenced blocks to detect switches:

```yaml
mode_switch:
  analysis: |
    Provide a long, detailed reasoning for why switching mode helps.
  new_mode: MODE_NAME
```

- Record transitions to SQLite for analytics/visualization

## Architecture

- `Service` merges mode resolution and persistence:
  - `GetMode(ctx, name)` returns `AgentMode { Name, Prompt, AllowedTools }`
  - `GetCurrentMode(ctx, runID)` returns last mode used for a run
  - `RecordModeChange(ctx, change)` appends a transition

- Middleware flow:
  1. Resolve current mode from `Turn.Data["agent_mode"]` or `Service.GetCurrentMode`
  2. Remove previously inserted AgentMode blocks by metadata tag to avoid duplicates
  3. Inject a single user message that combines mode prompt and YAML instructions; tag block metadata (`agentmode_tag=agentmode_user_prompt`, `agentmode=<mode>`) and position it as second-to-last
  4. Expose allowed tools via `Turn.Data["agent_mode_allowed_tools"]`
  5. After inference, parse YAML blocks from assistant messages; if a valid `mode_switch` is found, update Turn and `RecordModeChange`, and emit an Info event

## Data keys

- `agent_mode`: current mode name (string)
- `agent_mode_allowed_tools`: hint read by the Tool middleware to restrict calls

## Package layout

- `middleware.go`: Middleware implementation, YAML parsing, data keys
- `service.go`: `Service`, `StaticService`, `SQLiteService`
- `sqlite_store.go`: SQLite store and schema
- `schema.sql`: Schema example
- `seed.sql`: Seed example

## Example usage

```go
svc := agentmode.NewStaticService([]*agentmode.AgentMode{
  {Name: "chat",  AllowedTools: []string{"echo"},     Prompt: "You are in chat mode; prefer concise helpful answers."},
  {Name: "clock", AllowedTools: []string{"time_now"}, Prompt: "You are in clock mode; you may use time_now when necessary."},
})

mw := agentmode.NewMiddleware(svc, agentmode.DefaultConfig())
toolMw := middleware.NewToolMiddleware(tb, middleware.ToolConfig{MaxIterations: 3})
engine := middleware.NewEngineWithMiddleware(base, mw, toolMw)

t := &turns.Turn{Data: map[string]any{}}
t.Data[agentmode.DataKeyAgentMode] = "clock"
```

## YAML parser

This middleware uses the shared parser `parse.ExtractYAMLBlocks` to extract fenced YAML code blocks from assistant text. This makes the feature reusable for other firmwares or middlewares that want to parse model-structured YAML.

## SQLite schema and seed

- See `schema.sql` for the example schema.
- See `seed.sql` for inserting example modes and mode changes.

To initialize:

```sql
.read schema.sql
.read seed.sql
```

## Debugging and logging

- Unknown mode names are logged at warn level
- Detected switches are logged at debug with from/to
- On insertion/removal and on switch, the middleware emits streaming Log/Info events (see `geppetto/pkg/events/chat-events.go`), which can be consumed by UIs to render status.

## Writing good docs

This document follows the internal guidance from `glaze help how-to-write-good-documentation-pages` by:

- Providing a clear purpose and overview
- Explaining architecture, data flow, and configuration keys
- Showing copy-pasteable examples
- Referencing code locations and runtimes


