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

**Important:** The runtime tool registry is carried via `context.Context` (see `toolcontext.WithRegistry`). Only serializable tool configuration belongs in `Turn.Data` (e.g., `turns.DataKeyToolConfig`).

### Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/go-go-golems/geppetto/pkg/inference/toolcontext"
)
```

### Types

- Run: `{ ID, Name, Metadata map[RunMetadataKey]any, Turns []Turn }`
- Turn: `{ ID, RunID, Metadata map[TurnMetadataKey]any, Data map[TurnDataKey]any, Blocks []Block }`
- Block: `{ ID, TurnID, Kind, Role, Payload map[string]any, Metadata map[BlockMetadataKey]any }`
- BlockKind: `User`, `LLMText`, `ToolCall`, `ToolUse`, `System`, `Other`

**Note:** Map keys (`TurnDataKey`, `TurnMetadataKey`, `BlockMetadataKey`, `RunMetadataKey`) are typed in Go for compile-time safety, but serialize as strings in YAML. Use typed constants (e.g., `turns.DataKeyToolConfig`) rather than string literals.


### Helpers

- `turns.AppendBlock`, `turns.AppendBlocks`
- `turns.FindLastBlocksByKind`
- `turns.SetTurnMetadata(t *Turn, key TurnMetadataKey, value any)` - Set turn-level metadata with typed key
- `turns.SetBlockMetadata(b *Block, key BlockMetadataKey, value any)` - Set block-level metadata with typed key
- `turns.WithBlockMetadata(b Block, kvs map[BlockMetadataKey]any) Block` - Return block with metadata added
- `turns.HasBlockMetadata(b Block, key BlockMetadataKey, value string) bool` - Check if block has metadata key/value
- `turns.RemoveBlocksByMetadata(t *Turn, key BlockMetadataKey, values ...string) int` - Remove blocks by metadata
- Payload keys: `turns.PayloadKeyText`, `turns.PayloadKeyID`, `turns.PayloadKeyName`, `turns.PayloadKeyArgs`, `turns.PayloadKeyResult`
- Data keys: `turns.DataKeyToolConfig`
- Conversion:
  - `turns.BuildConversationFromTurn(t)`
  - `turns.BlocksFromConversationDelta(conv, startIdx)`

### Engine mapping

- Engines translate existing context (seeded blocks) into provider messages
- On completion, engines append:
  - `llm_text` for assistant text
  - `tool_call` for structured tool invocation requests
- Engines read tools from `context.Context` (tool registry) to include provider tool definitions for that Turn

### Tool workflow with Turns

1. Engine RunInference appends `tool_call` blocks
2. Middleware extracts pending tool calls (no matching `tool_use` by id)
3. Middleware executes tools and appends `tool_use` blocks
4. Engine is invoked again with the updated Turn to continue
5. Tools advertised for a step are provided via `context.Context` (registry) plus `Turn.Data` (config), enabling dynamic tools per Turn

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

Attach a tool registry to context (for engines to advertise tools) and tool config to the Turn:

```go
reg := tools.NewInMemoryToolRegistry()
// register tools ...
t := &turns.Turn{ Data: map[turns.TurnDataKey]any{} }
t.Data[turns.DataKeyToolConfig] = engine.ToolConfig{ Enabled: true, ToolChoice: engine.ToolChoiceAuto }

ctx = toolcontext.WithRegistry(ctx, reg)
```

### Why Turns

- Normalizes provider differences into a common structure
- Enables powerful middleware without parsing opaque text
- Clean separation between provider I/O (engines) and orchestration (middleware)


