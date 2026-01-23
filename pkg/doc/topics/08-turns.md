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

> **Key Pattern:** The runtime tool registry is carried via `context.Context` (see `toolcontext.WithRegistry`). Only serializable tool configuration belongs in `Turn.Data` (e.g., `engine.KeyToolConfig`).

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

### Type Definitions

```go
// Block represents a single atomic unit within a Turn.
type Block struct {
    ID       string                  // Optional unique identifier
    Kind     BlockKind               // user, llm_text, tool_call, etc.
    Role     string                  // Optional: "user", "assistant", "system"
    Payload  map[string]any          // Kind-specific content
    Metadata turns.BlockMetadata     // Annotations, provider hints (opaque store)
}

// Turn contains an ordered list of Blocks and associated metadata.
type Turn struct {
    ID       string            // Optional unique identifier
    Blocks   []Block           // Ordered blocks
    Metadata turns.Metadata    // Request params, usage, trace IDs (opaque store)
    Data     turns.Data        // Tool config, app-specific data (opaque store)
}

// Run captures a multi-turn session.
type Run struct {
    ID       string
    Name     string
    Turns    []Turn
    Metadata map[RunMetadataKey]any
}
```

**Note:** `turns.Data`, `turns.Metadata`, and `turns.BlockMetadata` are opaque wrapper stores. Access them via typed keys and key methods (`key.Get`/`key.Set`), not map indexing.

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

### Typed Keys

Geppetto uses typed keys for all store access to prevent drift and typos. Three key families exist for the three store types:

- `turns.DataKey[T]` for `Turn.Data`
- `turns.TurnMetaKey[T]` for `Turn.Metadata`
- `turns.BlockMetaKey[T]` for `Block.Metadata`

Keys are defined in key-definition files (e.g., `geppetto/pkg/turns/keys.go` and `geppetto/pkg/inference/engine/turnkeys.go`) and accessed via methods:

```go
// Setting a value
err := engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{Enabled: true})

// Getting a value
config, ok := engine.KeyToolConfig.Get(turn.Data)
```

**Why typed keys?** Direct map access like `turn.Data["config"]` compiles but creates key drift. The `turnsdatalint` analyzer catches these. Always use typed key variables.

## Working with Turns

### Creating Blocks

Use the builder functions for common block types:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/toolcontext"
)

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
t := &turns.Turn{}

// Append blocks
turns.AppendBlock(t, block)
turns.AppendBlocks(t, block1, block2, block3)
turns.PrependBlock(t, block)

// Find blocks by kind
llmBlocks := turns.FindLastBlocksByKind(*t, turns.BlockKindLLMText)
toolCalls := turns.FindLastBlocksByKind(*t, turns.BlockKindToolCall)

// Clone a Turn for safe mutation (rarely needed directly in apps; prefer session helpers below)
cloned := t.Clone()

// Store helpers (for opaque wrappers)
turn.Data.Len()
turn.Data.Range(func(k, v any) bool { ... })
turn.Data.Clone()
```

## Multi-turn Sessions (Chat-style apps)

For multi-turn interactions (user prompt → inference → repeat), prefer the `session.Session` API:

- Use `Session.AppendNewTurnFromUserPrompt(...)` (or `AppendNewTurnFromUserPrompts(...)`) to create the
  next prompt turn by cloning the latest turn (full snapshot) and appending one user block per prompt.
- Then call `Session.StartInference(ctx)` to run the tool loop/engine against the **latest appended
  turn in-place**. Middlewares are allowed to mutate the turn, and those mutations become the base
  for the next prompt.

```go
import "github.com/go-go-golems/geppetto/pkg/inference/session"

sess := session.NewSession()
sess.Builder = builder // e.g. enginebuilder.Builder

_, _ = sess.AppendNewTurnFromUserPrompt("Hello")
handle, _ := sess.StartInference(ctx)
updated, _ := handle.Wait()
_ = updated // == sess.Latest()
```

This model keeps a history of snapshots (`sess.Turns`), but only the newest snapshot is mutated
while an inference is running.

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

// 3. Configure tool behavior on Turn.Data using typed key
turn := &turns.Turn{}
err := engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{
    Enabled:          true,
    ToolChoice:       engine.ToolChoiceAuto,
    MaxParallelTools: 3,
})
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
metadata:
  geppetto.session_id@v1: sess_abc
data: {}
```

### Serde and Typed Keys

When you load Turns from YAML (`turns/serde.FromYAML`), `data`/`metadata` values decode into generic Go shapes (`map[string]any`, `[]any`, scalars). Typed keys (`key.Get`) will best-effort decode these into their target type `T` via JSON re-marshal/unmarshal.

If a struct type needs special decoding (e.g. `time.Duration` from `"2s"` strings), implement `UnmarshalJSON` on that struct. `engine.ToolConfig` does this so YAML fixtures can use `execution_timeout: 2s`.

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
// Using typed keys (new API)
err := turns.KeyTurnMetaModel.Set(&turn.Metadata, "gpt-4-turbo")
model, ok := turns.KeyTurnMetaModel.Get(turn.Metadata)
```

Common keys: `KeyTurnMetaProvider`, `KeyTurnMetaRuntime`, `KeyTurnMetaTraceID`, `KeyTurnMetaUsage`, `KeyTurnMetaStopReason`, `KeyTurnMetaModel`

Session correlation key: `KeyTurnMetaSessionID` (stored as `geppetto.session_id@v1` in YAML).

### Block-Level Metadata

```go
// Using typed keys
err := turns.KeyBlockMetaMiddleware.Set(&block.Metadata, "agentmode")
middleware, ok := turns.KeyBlockMetaMiddleware.Get(block.Metadata)
```

Common keys: `KeyBlockMetaMiddleware`, `KeyBlockMetaAgentModeTag`, `KeyBlockMetaAgentMode`, `KeyBlockMetaToolCalls`

## Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/turns"           // Core types, builders, helpers
    "github.com/go-go-golems/geppetto/pkg/turns/serde"     // YAML serialization
    "github.com/go-go-golems/geppetto/pkg/inference/engine" // KeyToolConfig
    "github.com/go-go-golems/geppetto/pkg/inference/toolcontext" // Context-based registry
)
```

## See Also

- [Inference Engines](06-inference-engines.md) — How engines use Turns
- [Tools](07-tools.md) — Defining and executing tools
- [Middlewares](09-middlewares.md) — Processing Turns with middleware
- [turnsdatalint](12-turnsdatalint.md) — Linter for typed key usage
