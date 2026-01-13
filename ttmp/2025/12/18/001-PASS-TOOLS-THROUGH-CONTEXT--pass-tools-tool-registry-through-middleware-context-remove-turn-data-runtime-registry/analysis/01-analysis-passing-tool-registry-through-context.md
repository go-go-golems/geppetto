---
Title: 'Analysis: passing tool registry through context'
Ticket: 001-PASS-TOOLS-THROUGH-CONTEXT
Status: active
Topics:
    - geppetto
    - turns
    - tools
    - context
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Inventory and analyze current ToolRegistry-in-Turn.Data usage; evaluate passing runtime tool registry through context while keeping Turn/Block bags serializable."
LastUpdated: 2025-12-18T12:07:47.038724844-05:00
---

## What we’re trying to fix

We want **all values stored in**:

- `turns.Turn.Data`
- `turns.Turn.Metadata`
- `turns.Block.Payload`
- `turns.Block.Metadata`

to be **serializable** (YAML/JSON round-trippable). Today, we store a runtime-only `tools.ToolRegistry` interface value in `Turn.Data`, which violates that constraint.

At the same time, we still need a *runtime* registry (executors, dynamic behavior) to reach:

- engines (to advertise tools)
- middleware (to add/adjust tool availability)
- tool execution (to actually run tools)

## Current behavior: where the registry is written/read (non-doc code)

### Writes to `Turn.Data[turns.DataKeyToolRegistry]`

- `moments/backend/pkg/webchat/router.go`: attaches registry to `conv.Turn.Data`
- `geppetto/pkg/inference/toolhelpers/helpers.go`: attaches registry to the turn inside the tool-calling loop
- `go-go-mento/go/pkg/webchat/router.go`: attaches registry to turn
- `go-go-mento/go/pkg/drive1on1/http/handlers/agent_chat.go`: attaches registry to turn
- `go-go-mento/go/pkg/drive1on1/http/handlers/agent_summarize.go`: attaches registry to turn

### Reads from `Turn.Data[turns.DataKeyToolRegistry]`

Engines (advertise tools to providers):

- `geppetto/pkg/steps/ai/openai/engine_openai.go`
- `geppetto/pkg/steps/ai/claude/engine_claude.go`
- `geppetto/pkg/steps/ai/gemini/engine_gemini.go`

Middleware/instrumentation:

- `moments/backend/pkg/webchat/langfuse_middleware.go` (reads registry to list tools for tracing)
- `go-go-mento/go/pkg/inference/middleware/langfuse_middleware.go` (same pattern)

Middleware that mutates registry:

- `pinocchio/pkg/middlewares/sqlitetool/middleware.go` (ensures SQL tool exists in the registry)

Persistence / serialization boundary:

- `pinocchio/cmd/agents/simple-chat-agent/pkg/store/sqlstore.go`
  - serializes `reg.ListTools()` as JSON
  - explicitly skips storing the raw registry object when iterating `t.Data`

## Observation: there are two different needs hidden behind “registry”

1. **Tool descriptions** (“advertising”): a serializable list of definitions (`[]ToolDefinition`) is sufficient.
2. **Tool executors** (“execution”): you need a runtime mapping from tool name/id → implementation. This is not serializable (it is code).

So “store ToolRegistry in Turn.Data” is really conflating (1) and (2).

## Proposed split of responsibilities

### Serializable: store tool *definitions* on the Turn (or derived from config/profile)

Replace `Turn.Data[turns.DataKeyToolRegistry] = reg` with something serializable, e.g.:

- `Turn.Data[turns.DataKeyToolDefinitions] = []tools.ToolDefinition{...}`

Engines and tracing middleware then read only the definitions list.

### Runtime-only: pass tool *executors/registry* through `context.Context`

Pass the real runtime registry through middleware chain context:

- Context carries `tools.ToolRegistry` (or a narrower interface) for the duration of the request/turn execution.
- Tool execution reads registry from `ctx`, not from `Turn.Data`.

This keeps Turn/Block bags serializable without losing runtime flexibility.

## Risks / sharp edges

- **Reproducibility**: storing tool defs on the turn makes snapshots more deterministic; passing runtime registry via `ctx` is ephemeral. That’s fine, but it means “replay from persisted turn” needs a way to reconstruct runtime registry from stored defs/profile.
- **Naming/id stability**: if we rely on tool names as identifiers, ensure no drift. If we move to typed tool IDs later, this becomes cleaner.
- **Middleware ordering**: middlewares that “add tools” must run before execution. If they formerly mutated `Turn.Data` registry, they now need to mutate the runtime registry in `ctx` (or mutate the defs list plus have a builder).

## Migration outline (high level)

- Introduce a small `toolcontext` helper (or similar) that stores/loads registry from `context.Context`.
- Introduce a new serializable key (e.g. `turns.DataKeyToolDefinitions`) and store `[]tools.ToolDefinition` there.
- Update engines to read tool definitions from Turn.Data instead of reading a registry object.
- Update middleware:
  - tracing reads defs from Turn.Data
  - execution reads registry from `ctx`
  - any “registry mutating” middleware must mutate runtime registry via `ctx` (or shift to mutating defs + rebuilding registry).
- Update persistence to stop special-casing raw registry objects (since they no longer exist in Turn.Data).

