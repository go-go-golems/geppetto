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

**Important:** The runtime tool registry is carried via `context.Context` (see `toolcontext.WithRegistry`). Only serializable tool configuration belongs in `Turn.Data` (e.g., `engine.KeyToolConfig`).

### Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/toolcontext"
)
```

### Types

- Run: `{ ID, Name, Metadata map[RunMetadataKey]any, Turns []Turn }`
- Turn: `{ ID, RunID, Metadata turns.Metadata, Data turns.Data, Blocks []Block }`
- Block: `{ ID, TurnID, Kind, Role, Payload map[string]any, Metadata turns.BlockMetadata }`
- BlockKind: `User`, `LLMText`, `ToolCall`, `ToolUse`, `System`, `Other`

**Note:** Store keys (`TurnDataKey`, `TurnMetadataKey`, `BlockMetadataKey`, `RunMetadataKey`) are typed in Go for compile-time safety, but serialize as strings in YAML. Turn stores (`turns.Data`, `turns.Metadata`, `turns.BlockMetadata`) are opaque wrappers; access them via typed keys and key methods (`key.Get/key.Set`), not map indexing.

### Typed keys

Geppetto uses three store-specific key families:

- `turns.DataKey[T]` for `Turn.Data`
- `turns.TurnMetaKey[T]` for `Turn.Metadata`
- `turns.BlockMetaKey[T]` for `Block.Metadata`

Define keys in key-definition files (for geppetto: `geppetto/pkg/turns/keys.go` and `geppetto/pkg/inference/engine/turnkeys.go`) and reuse the canonical variables everywhere else.

### Helpers

- `turns.AppendBlock`, `turns.AppendBlocks`
- `turns.FindLastBlocksByKind`
- Payload keys: `turns.PayloadKeyText`, `turns.PayloadKeyID`, `turns.PayloadKeyName`, `turns.PayloadKeyArgs`, `turns.PayloadKeyResult`
- Wrapper-store helpers:
  - `turns.Data.Len`, `turns.Data.Range`, `turns.Data.Clone`
  - `turns.Metadata.Len`, `turns.Metadata.Range`, `turns.Metadata.Clone`
  - `turns.BlockMetadata.Len`, `turns.BlockMetadata.Range`, `turns.BlockMetadata.Clone`

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
t := &turns.Turn{}
_ = engine.KeyToolConfig.Set(&t.Data, engine.ToolConfig{Enabled: true, ToolChoice: engine.ToolChoiceAuto})

ctx = toolcontext.WithRegistry(ctx, reg)
```

### Why Turns

- Normalizes provider differences into a common structure
- Enables powerful middleware without parsing opaque text
- Clean separation between provider I/O (engines) and orchestration (middleware)
