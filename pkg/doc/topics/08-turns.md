---
Title: Turns and Blocks in Geppetto
Slug: geppetto-turns
Short: Understanding the Run/Turn/Block data model and how engines and middleware use it
Topics:
- geppetto
- turns
- blocks
- architecture
- serialization
- yaml
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

# Turns and Blocks in Geppetto

## Why Turns?

Every AI conversation is a sequence of messages — user prompts, assistant responses, tool calls, and results. Different providers represent these differently: OpenAI uses "messages" with roles, Claude uses "content blocks", Gemini has its own format.

**Turns** provide a unified model that works across all providers. A Turn contains ordered **Blocks** — each representing one piece of the conversation. This abstraction lets you:

- **Write provider-agnostic code** — switch between OpenAI, Claude, Gemini, or Ollama via config
- **Build powerful middleware** — inspect and transform blocks without parsing raw text
- **Serialize conversations** — save/load to YAML for testing, debugging, and persistence
- **Enable dynamic tools** — attach different tools to each inference call

## Core Concepts

### The Data Model

Geppetto organizes conversations into three levels:

```
Run (session)
 └── Turn (one inference cycle)
      └── Block (atomic unit: message, tool call, etc.)
```

| Level | What It Represents | Example |
|-------|-------------------|---------|
| **Run** | A multi-turn session | A complete chat conversation |
| **Turn** | One inference cycle | User asks → Assistant responds |
| **Block** | One atomic piece | "Hello, how can I help?" |

### Block Kinds

Each block has a `Kind` that describes what it contains:

| Kind | Created By | Contains | Payload Keys |
|------|------------|----------|--------------|
| `System` | Your code | System prompt | `text` |
| `User` | Your code | User message | `text`, optionally `images` |
| `LLMText` | Engine | Assistant's text response | `text` |
| `ToolCall` | Engine | Model's request to call a tool | `id`, `name`, `args` |
| `ToolUse` | Middleware/Helper | Result of tool execution | `id`, `result` |
| `Reasoning` | Engine (o1, Claude) | Model's reasoning trace | `encrypted_content`, `item_id` |
| `Other` | Various | Catch-all for unknown types | varies |

### Type Definitions

```go
// Block represents a single atomic unit within a Turn.
type Block struct {
    ID       string                         // Optional unique identifier
    TurnID   string                         // Parent turn reference
    Kind     BlockKind                      // user, llm_text, tool_call, etc.
    Role     string                         // Optional: "user", "assistant", "system"
    Payload  map[string]any                 // Kind-specific content
    Metadata map[BlockMetadataKey]any       // Annotations, provider hints
}

// Turn contains an ordered list of Blocks and associated metadata.
type Turn struct {
    ID       string                         // Optional unique identifier
    RunID    string                         // Parent run reference
    Blocks   []Block                        // Ordered blocks
    Metadata map[TurnMetadataKey]any        // Request params, usage, trace IDs
    Data     map[TurnDataKey]any            // Tool config, app-specific data
}

// Run captures a multi-turn session.
type Run struct {
    ID       string
    Name     string
    Turns    []Turn
    Metadata map[RunMetadataKey]any
}
```

## Working with Turns

### Creating Blocks

Use the builder functions for common block types:

```go
import "github.com/go-go-golems/geppetto/pkg/turns"

// Create a seed turn for inference
seed := &turns.Turn{}
turns.AppendBlock(seed, turns.NewSystemTextBlock("You are a helpful assistant."))
turns.AppendBlock(seed, turns.NewUserTextBlock("What's the weather in Paris?"))
```

Or use the fluent TurnBuilder:

```go
seed := turns.NewTurnBuilder().
    WithSystemPrompt("You are a helpful assistant.").
    WithUserPrompt("What's the weather in Paris?").
    Build()
```

### Reading Block Content

Always use payload key constants — never string literals:

```go
// ✅ Correct: use typed constants
for _, block := range turn.Blocks {
    if block.Kind == turns.BlockKindLLMText {
        if text, ok := block.Payload[turns.PayloadKeyText].(string); ok {
            fmt.Println(text)
        }
    }
}

// ❌ Wrong: string literals (caught by turnsdatalint)
text := block.Payload["text"].(string)
```

### Payload Key Constants

```go
const (
    PayloadKeyText             = "text"              // Text content
    PayloadKeyID               = "id"                // Tool call/result ID
    PayloadKeyName             = "name"              // Tool name
    PayloadKeyArgs             = "args"              // Tool arguments
    PayloadKeyResult           = "result"            // Tool result
    PayloadKeyImages           = "images"            // Attached images
    PayloadKeyEncryptedContent = "encrypted_content" // Reasoning trace
    PayloadKeyItemID           = "item_id"           // Provider item ID
)
```

### Helper Functions

```go
// Append blocks
turns.AppendBlock(t, block)
turns.AppendBlocks(t, block1, block2, block3)
turns.PrependBlock(t, block)

// Find blocks by kind
llmBlocks := turns.FindLastBlocksByKind(turn, turns.BlockKindLLMText)
toolCalls := turns.FindLastBlocksByKind(turn, turns.BlockKindToolCall)

// Metadata helpers
turns.SetTurnMetadata(t, turns.TurnMetaKeyModel, "gpt-4")
turns.SetBlockMetadata(&block, turns.BlockMetaKeyMiddleware, "agent")
```

## Typed Map Keys

Geppetto uses typed keys for all map access to prevent drift and typos:

| Map | Key Type | Example Constants |
|-----|----------|-------------------|
| `Turn.Data` | `TurnDataKey` | `DataKeyToolConfig`, `DataKeyAgentMode` |
| `Turn.Metadata` | `TurnMetadataKey` | `TurnMetaKeyModel`, `TurnMetaKeyUsage` |
| `Block.Metadata` | `BlockMetadataKey` | `BlockMetaKeyMiddleware`, `BlockMetaKeyAgentMode` |
| `Run.Metadata` | `RunMetadataKey` | `RunMetaKeyTraceID` |
| `Block.Payload` | `string` (const) | `PayloadKeyText`, `PayloadKeyArgs` |

**Why typed keys?** Go allows implicit conversion from untyped strings to these types, so `turn.Data["raw"]` compiles but creates key drift. The `turnsdatalint` analyzer catches these. Always use the typed constants.

## Tool Configuration

Tools are configured in two places:

1. **Runtime registry** — Callable functions, attached to `context.Context`
2. **Turn.Data config** — Serializable settings like `Enabled`, `ToolChoice`

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/toolcontext"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
)

// 1. Create and register tools
registry := tools.NewInMemoryToolRegistry()
// ... register tools ...

// 2. Attach registry to context (engines read this)
ctx = toolcontext.WithRegistry(ctx, registry)

// 3. Configure tool behavior on Turn.Data
turn := &turns.Turn{Data: map[turns.TurnDataKey]any{}}
turn.Data[turns.DataKeyToolConfig] = engine.ToolConfig{
    Enabled:          true,
    ToolChoice:       engine.ToolChoiceAuto,
    MaxParallelTools: 3,
}
```

This separation keeps Turn state serializable while allowing dynamic tool changes per inference call.

## Tool Workflow

The tool calling loop follows this pattern:

```
1. You create a Turn with user/system blocks
2. Engine.RunInference() processes it
3. Engine appends llm_text and/or tool_call blocks
4. Middleware/helpers extract pending tool_call blocks
5. Tools execute, tool_use blocks are appended
6. Engine is called again with the updated Turn
7. Repeat until no more tool calls
```

Matching: A `tool_call` block is "pending" if no `tool_use` block exists with the same `id`.

## Serialization (YAML)

Turns serialize to human-readable YAML for testing, debugging, and persistence:

```yaml
version: 1
id: turn_001
run_id: run_abc
blocks:
  - kind: system
    role: system
    payload: { text: "You are a helpful assistant." }
  - kind: user
    role: user
    payload: { text: "What's 2+2?" }
  - kind: tool_call
    payload:
      id: fc_1
      name: calculator
      args: { expression: "2+2" }
  - kind: tool_use
    payload:
      id: fc_1
      result: { answer: 4 }
  - kind: llm_text
    role: assistant
    payload: { text: "2+2 equals 4." }
metadata: {}
data: {}
```

### Serde Helpers

```go
import "github.com/go-go-golems/geppetto/pkg/turns/serde"

// Save to file
err := serde.SaveTurnYAML("turn.yaml", turn, serde.Options{OmitData: false})

// Load from file  
loaded, err := serde.LoadTurnYAML("turn.yaml")
```

Use YAML fixtures in `testdata/` folders for regression tests and offline analysis.

## Engine Mapping

Engines translate between Turns and provider-specific formats:

| Turn Block | OpenAI | Claude | Gemini |
|------------|--------|--------|--------|
| `System` | `system` message | system parameter | `system_instruction` |
| `User` | `user` message | `user` role | `user` role |
| `LLMText` | `assistant` message | `assistant` role | `model` role |
| `ToolCall` | `tool_calls` array | `tool_use` block | `functionCall` |
| `ToolUse` | `tool` message | `tool_result` block | `functionResponse` |

Engines handle this mapping internally — your code just works with Turns.

## Metadata

### Turn-Level Metadata

```go
// Set after inference
turns.SetTurnMetadata(turn, turns.TurnMetaKeyModel, "gpt-4-turbo")
turns.SetTurnMetadata(turn, turns.TurnMetaKeyUsage, usageStats)
turns.SetTurnMetadata(turn, turns.TurnMetaKeyTraceID, traceID)
```

Common keys: `TurnMetaKeyProvider`, `TurnMetaKeyRuntime`, `TurnMetaKeyTraceID`, `TurnMetaKeyUsage`, `TurnMetaKeyStopReason`, `TurnMetaKeyModel`

### Block-Level Metadata

```go
// Mark blocks for filtering
turns.SetBlockMetadata(&block, turns.BlockMetaKeyMiddleware, "agentmode")
turns.SetBlockMetadata(&block, turns.BlockMetaKeyAgentMode, "research")
```

Common keys: `BlockMetaKeyMiddleware`, `BlockMetaKeyAgentModeTag`, `BlockMetaKeyAgentMode`, `BlockMetaKeyToolCalls`

## Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/turns"           // Core types, builders, helpers
    "github.com/go-go-golems/geppetto/pkg/turns/serde"     // YAML serialization
    "github.com/go-go-golems/geppetto/pkg/inference/toolcontext" // Context-based registry
)
```

## See Also

- [Inference Engines](06-inference-engines.md) — How engines use Turns
- [Tools](07-tools.md) — Defining and executing tools
- [Middlewares](09-middlewares.md) — Processing Turns with middleware
- [turnsdatalint](12-turnsdatalint.md) — Linter for typed key usage
