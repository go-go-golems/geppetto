---
Title: Turns and Blocks in Geppetto
Slug: geppetto-turns
Short: Understanding the Run/Turn/Block data model and how engines and middleware use it
Topics:
- geppetto
- turns
- blocks
- architecture
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

## Turns and Blocks in Geppetto

The Turn data model provides a provider-agnostic representation of an interaction, decomposed into ordered `Block`s. Engines read and append blocks; middleware inspects blocks to implement behaviors like tool execution.

### Types

- Run: `{ ID, Name, Metadata, Turns []Turn }`
- Turn: `{ ID, RunID, Metadata map[string]any, Data map[string]any, Blocks []Block }`
- Block: `{ Order, Kind, Role, Payload map[string]any, Metadata []MetadataKV }`
- BlockKind: `User`, `LLMText`, `ToolCall`, `ToolUse`, `System`, `Other`

### Helpers

- `turns.AppendBlock`, `turns.AppendBlocks`
- `turns.FindLastBlocksByKind`
- `turns.SetTurnMetadata`, `turns.UpsertBlockMetadata`
- Conversion:
  - `turns.BuildConversationFromTurn(t)`
  - `turns.BlocksFromConversationDelta(conv, startIdx)`

### Engine mapping

- Engines translate existing context (seeded blocks) into provider messages
- On completion, engines append:
  - `llm_text` for assistant text
  - `tool_call` for structured tool invocation requests

### Tool workflow with Turns

1. Engine RunInference appends `tool_call` blocks
2. Middleware extracts pending tool calls (no matching `tool_use` by id)
3. Middleware executes tools and appends `tool_use` blocks
4. Engine is invoked again with the updated Turn to continue

### Metadata

- Turn-level metadata records request parameters, tracing ids, usage
- Block-level metadata stores provider payload hints, policy annotations

### Why Turns

- Normalizes provider differences into a common structure
- Enables powerful middleware without parsing opaque text
- Clean separation between provider I/O (engines) and orchestration (middleware)


