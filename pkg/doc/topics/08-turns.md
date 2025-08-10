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

The Turn data model provides a provider-agnostic representation of an interaction, decomposed into ordered `Block`s. Engines read and append blocks; middleware inspects blocks to implement behaviors like tool execution. With the latest refactor, a Turn can also carry a per-turn tool registry and tool configuration in `Turn.Data`, allowing dynamic tools at each step.

### Types

- Run: `{ ID, Name, Metadata, Turns []Turn }`
- Turn: `{ ID, RunID, Metadata map[string]any, Data map[string]any, Blocks []Block }`
- Block: `{ Order, Kind, Role, Payload map[string]any, Metadata []MetadataKV }`
- BlockKind: `User`, `LLMText`, `ToolCall`, `ToolUse`, `System`, `Other`

### Helpers

- `turns.AppendBlock`, `turns.AppendBlocks`
- `turns.FindLastBlocksByKind`
- `turns.SetTurnMetadata`, `turns.SetBlockMetadata`
- Payload keys: `turns.PayloadKeyText`, `turns.PayloadKeyID`, `turns.PayloadKeyName`, `turns.PayloadKeyArgs`, `turns.PayloadKeyResult`
- Data keys: `turns.DataKeyToolRegistry`, `turns.DataKeyToolConfig`
- Conversion:
  - `turns.BuildConversationFromTurn(t)`
  - `turns.BlocksFromConversationDelta(conv, startIdx)`

### Engine mapping

- Engines translate existing context (seeded blocks) into provider messages
- On completion, engines append:
  - `llm_text` for assistant text
  - `tool_call` for structured tool invocation requests
 - Engines read tools from `Turn.Data` to include provider tool definitions for that Turn

### Tool workflow with Turns

1. Engine RunInference appends `tool_call` blocks
2. Middleware extracts pending tool calls (no matching `tool_use` by id)
3. Middleware executes tools and appends `tool_use` blocks
4. Engine is invoked again with the updated Turn to continue
5. Tools advertised for a step are provided via `Turn.Data` (registry/config), enabling dynamic tools per Turn

### Metadata

- Turn-level metadata records request parameters, tracing ids, usage
- Block-level metadata stores provider payload hints, policy annotations

### Guided examples

Create and append common blocks:

```go
seed := &turns.Turn{}
turns.AppendBlock(seed, turns.NewSystemTextBlock("You are a helpful assistant."))
turns.AppendBlock(seed, turns.NewUserTextBlock("What is the weather in Paris?"))
// later: engine appends turns.NewAssistantTextBlock and turns.NewToolCallBlock
```

Use the payload key constants when reading block content:

```go
for i := range seed.Blocks {
    b := &seed.Blocks[i]
    if b.Kind == turns.BlockKindLLMText {
        if txt, ok := b.Payload[turns.PayloadKeyText].(string); ok {
            // use txt
        }
    }
}
```

Attach a tool registry to a Turn (for engines to advertise tools):

```go
reg := tools.NewInMemoryToolRegistry()
// register tools ...
t := &turns.Turn{ Data: map[string]any{} }
t.Data[turns.DataKeyToolRegistry] = reg
t.Data[turns.DataKeyToolConfig] = engine.ToolConfig{ Enabled: true, ToolChoice: engine.ToolChoiceAuto }
```

### Why Turns

- Normalizes provider differences into a common structure
- Enables powerful middleware without parsing opaque text
- Clean separation between provider I/O (engines) and orchestration (middleware)


