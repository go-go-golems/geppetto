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

> **Key Pattern:** The runtime tool registry is carried via `context.Context` (see `tools.WithRegistry`). Only serializable tool configuration belongs in `Turn.Data` (e.g., `engine.KeyToolConfig`).

## Why "Turn" Instead of "Message"

Most LLM frameworks model interactions as a list of chat messages with roles (`user`, `assistant`, `system`). This works for simple chatbots but breaks down for many real-world uses:

- **Document processing** — one input, one output, no conversation at all.
- **Agent loops** — the model calls tools repeatedly without any human input in between.
- **Multi-mode agents** — different instructions and tool sets per mode, switched mid-run.
- **Reasoning/planning** — internal steps that aren't "messages" to anyone.

A **Turn** is a general-purpose container for one inference cycle. It holds everything the model needs to see (input blocks) and everything it produces (output blocks), regardless of whether the interaction is a chat, a batch job, or an agent loop. A single Turn may contain a system prompt, several prior user/assistant exchanges, multiple tool calls and results, and the model's final response — all as ordered blocks in one structure.

The word "Turn" avoids the conversational connotations of "message" and correctly implies that the model takes a turn (like in a board game): it receives context, reasons, and produces output.

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

### How Blocks Accumulate During Inference

A Turn is not static — blocks are **appended in place** as inference proceeds. Understanding this growth is essential for working with middleware and debugging tools.

Here is what a Turn's block list looks like at different moments during a single inference cycle with tool use:

```
Before inference:        [system, user]
After model responds:    [system, user, tool_call]
After tool executes:     [system, user, tool_call, tool_use]
After model finalizes:   [system, user, tool_call, tool_use, llm_text]
```

Step by step:

1. Your application creates the Turn with a system prompt and the user's question.
2. `Engine.RunInference()` calls the LLM API. The model decides to call a tool — the engine appends a `tool_call` block.
3. The tool loop extracts the pending tool call, executes it, and appends a `tool_use` block with the result.
4. The engine runs again (the model now sees the tool result) and appends an `llm_text` block with the final answer.

This all happens on the **same Turn pointer**. The Turn is mutated in place — middlewares see and can modify the evolving context at each step.

The tool loop captures **snapshots** at named phases so you can inspect the Turn's state at each stage:

| Phase | When captured | What the Turn contains |
|-------|--------------|----------------------|
| `pre_inference` | Before engine runs | Input blocks only |
| `post_inference` | After engine returns | Input + model output (text, tool calls) |
| `post_tools` | After tools execute | Input + model output + tool results |
| `final` | After loop completes | Complete Turn |

### Typed Keys

Geppetto uses typed keys for all store access to prevent drift and typos. Three key families exist for the three store types:

- `turns.DataKey[T]` for `Turn.Data`
- `turns.TurnMetaKey[T]` for `Turn.Metadata`
- `turns.BlockMetaKey[T]` for `Block.Metadata`

Keys are defined in generated key-definition files (e.g., `geppetto/pkg/turns/keys_gen.go` and `geppetto/pkg/inference/engine/turnkeys_gen.go`) and accessed via methods:

```go
// Setting a value
err := engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{Enabled: true})

// Structured output config key (typed, engine-owned)
strict := true
err = engine.KeyStructuredOutputConfig.Set(&turn.Data, engine.StructuredOutputConfig{
    Mode:   engine.StructuredOutputModeJSONSchema,
    Name:   "person",
    Schema: map[string]any{"type": "object"},
    Strict: &strict,
})

// Getting a value
config, ok := engine.KeyToolConfig.Get(turn.Data)
```

**Why typed keys?** Direct map access like `turn.Data["config"]` compiles but creates key drift. The `turnsdatalint` analyzer catches these. Always use typed key variables.

`KeyToolConfig` is actively consumed in inference paths today. `KeyStructuredOutputConfig` is available as a typed key and intended for per-turn overrides as provider wiring expands.

## Working with Turns

### Creating Blocks

Use the builder functions for common block types:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
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
    PayloadKeyError            = "error"             // Tool error (string)
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

### How Turns Grow Across a Conversation

Each new Turn starts as a **clone** of the previous Turn's final state, with the new user prompt appended. This means every Turn is a complete snapshot of the full context:

```
Turn 1 (start):          [system, user₁]
Turn 1 (after inference): [system, user₁, llm_text₁]

Turn 2 = clone(Turn 1) + user₂:
Turn 2 (start):          [system, user₁, llm_text₁, user₂]
Turn 2 (after inference): [system, user₁, llm_text₁, user₂, llm_text₂]

Turn 3 = clone(Turn 2) + user₃:
Turn 3 (start):          [system, user₁, llm_text₁, user₂, llm_text₂, user₃]
Turn 3 (after inference): [system, user₁, llm_text₁, user₂, llm_text₂, user₃, llm_text₃]
```

You can look at any Turn in isolation and see the complete context the model had at that point. A diff between Turn N and Turn N+1 shows exactly what was added (new user prompt + model response + any tool interactions).

## Tool Configuration

Tools are configured in two places:

1. **Runtime registry** — Callable functions, attached to `context.Context`
2. **Turn.Data config** — Serializable settings like `Enabled`, `ToolChoice`

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
)

// 1. Create and register tools
registry := tools.NewInMemoryToolRegistry()
// ... register tools ...

// 2. Attach registry to context (engines read this)
ctx = tools.WithRegistry(ctx, registry)

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
    "github.com/go-go-golems/geppetto/pkg/inference/tools" // ToolRegistry + context helpers (tools.WithRegistry/RegistryFrom)
)
```

## See Also

- [Sessions](10-sessions.md) — Managing multi-turn interactions with Turn history
- [Inference Engines](06-inference-engines.md) — How engines use Turns; see "Complete Runtime Flow"
- [Tools](07-tools.md) — Defining and executing tools
- [Middlewares](09-middlewares.md) — Processing Turns with middleware
- [Events](04-events.md) — Streaming events emitted during inference
- [Structured Sinks](11-structured-sinks.md) — Extracting structured data from LLM text streams
- [turnsdatalint](12-turnsdatalint.md) — Linter for typed key usage
