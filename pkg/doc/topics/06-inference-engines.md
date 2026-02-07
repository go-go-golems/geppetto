---
Title: Understanding and Using the Geppetto Inference Engine Architecture (Turn-based)
Slug: geppetto-inference-engines
Short: A comprehensive guide to the Turn-based inference engine architecture, covering engines, streaming, and provider implementations.
Topics:
  - geppetto
  - inference
  - engines
  - turns
  - tools
  - architecture
  - tutorial
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Understanding and Using the Geppetto Inference Engine Architecture

> **Audience**: Developers familiar with Go who want to embed AI capabilities into their applications using Geppetto.<br/>
> **Outcome**: You will understand the architecture's core concepts and learn how to instantiate engines, orchestrate tool calls, and apply best practices in production.

## 30-Second Overview

```go
// 1. Create an engine from configuration
engine, _ := factory.NewEngineFromParsedLayers(parsedLayers)

// 2. Build a Turn with your prompt
turn := &turns.Turn{}
turns.AppendBlock(turn, turns.NewSystemTextBlock("You are a helpful assistant."))
turns.AppendBlock(turn, turns.NewUserTextBlock("Hello!"))

// 3. Run inference
result, _ := engine.RunInference(ctx, turn)

// 4. Read the response
for _, block := range result.Blocks {
    if block.Kind == turns.BlockKindLLMText {
        fmt.Println(block.Payload[turns.PayloadKeyText])
    }
}
```

That's it. The engine handles provider-specific API calls, streaming, and response parsing. You work with Turns and Blocks.

For chat-style, multi-turn apps, prefer `session.Session` (it centralizes “clone latest + append user
prompt” via `AppendNewTurnFromUserPrompt(s)` and runs inference against the latest appended turn
in-place via `StartInference`).

## Table of Contents
1. [Core Architecture Principles](#core-architecture-principles)
2. [The Engine Interface](#the-engine-interface)
3. [Creating Engines with Factories](#creating-engines-with-factories)
4. [Basic Inference Without Tools](#basic-inference-without-tools)
5. [Tool Calling with Helpers](#tool-calling-with-helpers)
6. [Provider-Specific Implementations](#provider-specific-implementations)
7. [Middleware and Cross-Cutting Concerns](#middleware-and-cross-cutting-concerns)
8. [Testing and Mocking](#testing-and-mocking)
9. [Best Practices](#best-practices)
10. [Debugging and Troubleshooting](#debugging-and-troubleshooting)
11. [Conclusion](#conclusion)

This tutorial explains the Turn-based inference architecture in Geppetto. Engines operate on a `Turn` (ordered `Block`s plus metadata), handle provider I/O, and publish streaming events via sinks. Tool orchestration can be handled by middleware or helpers. The result is simpler, more testable, and provider-agnostic.

### Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/turns"
)
```

## Core Architecture Principles

The Geppetto inference architecture is built around a clean separation of concerns:

- **Engines**: Handle provider-specific API calls and streaming (emit events)
- **Tool Helpers**: Manage tool calling orchestration and workflows
- **Factories**: Create engines from configuration layers
- **Middleware**: Add cross-cutting concerns like logging and event publishing

### Key Benefits

- **Simplicity**: Single `RunInference` method on engines
- **Provider Agnostic**: Works with OpenAI, Claude, Gemini, or any provider
- **Testable**: Easy to mock engines for testing
- **Composable**: Mix and match engines, helpers, and middleware

### Context-Based Dependency Injection

Geppetto uses `context.Context` to carry runtime dependencies rather than global state or struct fields:

- **Event sinks**: `events.WithEventSinks(ctx, sink)` — engines and middleware publish streaming events to sinks found on the context.
- **Tool registries**: `tools.WithRegistry(ctx, registry)` — engines discover available tools from the context.
- **Snapshot hooks**: `toolloop.WithTurnSnapshotHook(ctx, hook)` — the tool loop invokes snapshot callbacks found on the context.

This pattern avoids global state, makes testing straightforward (just pass a different context), and supports multiple parallel inference runs with independent configuration.

## The Engine Interface

The heart of the architecture is the simple `Engine` interface (no explicit streaming method; streaming happens when sinks are configured on the engine):

```go
import (
    "context"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

// Engine processes a Turn and returns the updated Turn.
type Engine interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

### Engine Responsibilities

Engines focus solely on API communication:

1. **API Calls**: Make provider-specific HTTP requests
2. **Response Parsing**: Convert API responses to Turn blocks (`llm_text`, `tool_call`)
3. **Streaming**: Publish events for real-time updates
4. **Error Handling**: Manage API-level errors and retries

Engines do **NOT** handle:

- Tool execution
- Tool calling loops
- Complex orchestration logic

## Creating Engines with Factories

The factory pattern creates engines from configuration layers, providing a provider-agnostic way to instantiate engines:

```go
package main

import (
    "context"
    "fmt"

    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/glazed/pkg/cmds/layers"
)

func createEngine(parsedLayers *layers.ParsedLayers) (engine.Engine, error) {
    // Create engine from configuration - works with any provider
    baseEngine, err := factory.NewEngineFromParsedLayers(parsedLayers)
    if err != nil {
        return nil, fmt.Errorf("failed to create engine: %w", err)
    }

    return baseEngine, nil
}
```

### Engine Options

Provider engines are created without options. Event sinks are attached to the runtime
`context.Context`:

```go
func createEngine(parsedLayers *layers.ParsedLayers) (engine.Engine, error) {
    return factory.NewEngineFromParsedLayers(parsedLayers)
}

func runWithSinks(ctx context.Context, eng engine.Engine, sink events.EventSink, seed *turns.Turn) (*turns.Turn, error) {
    runCtx := events.WithEventSinks(ctx, sink)
    return eng.RunInference(runCtx, seed)
}
```

## Basic Inference Without Tools

For simple text generation without tool calling:

```go
import (
    "context"
    "fmt"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

func simpleInference(ctx context.Context, parsedLayers *layers.ParsedLayers, prompt string) error {
    e, err := factory.NewEngineFromParsedLayers(parsedLayers)
    if err != nil { return fmt.Errorf("failed to create engine: %w", err) }

    seed := &turns.Turn{}
    turns.AppendBlock(seed, turns.NewSystemTextBlock("You are a helpful assistant."))
    turns.AppendBlock(seed, turns.NewUserTextBlock(prompt))

    updated, err := e.RunInference(ctx, seed)
    if err != nil { return fmt.Errorf("inference failed: %w", err) }

    for _, block := range updated.Blocks {
        if block.Kind != turns.BlockKindLLMText {
            continue
        }
        if text, ok := block.Payload[turns.PayloadKeyText].(string); ok {
            fmt.Println(text)
        }
    }
    return nil
}
```

## Tool Calling with the Tool Loop (Per-Turn tools)

Geppetto’s canonical tool orchestration lives in `toolloop.Loop`. Engines focus on provider I/O, while the loop handles extracting tool calls, executing tools, appending tool results, and iterating.

Providers learn about available tools from the **runtime registry attached to `context.Context`** (see `tools.WithRegistry`) plus any **serializable tool config** stored on `Turn.Data` (written automatically by the loop via `engine.KeyToolConfig`).

### Tool Calling Building Blocks

The main building blocks are:

- `toolblocks.ExtractPendingToolCalls`: find tool calls without matching tool_use blocks
- `tools.NewDefaultToolExecutor`: execute tool calls against a registry
- `toolblocks.AppendToolResultsBlocks`: append `tool_use` blocks from results
- `toolloop.Loop.RunLoop`: complete automated Turn-based workflow

### Setting Up Tools

First, create and register your tools:

```go
package main

import (
    "context"
    "encoding/json"

    "github.com/go-go-golems/geppetto/pkg/inference/tools"
)

// WeatherRequest represents the input for the weather tool
type WeatherRequest struct {
    Location string `json:"location" jsonschema:"required,description=The city to get weather for"`
    Units    string `json:"units,omitempty" jsonschema:"description=Temperature units,default=celsius,enum=celsius,enum=fahrenheit"`
}

// WeatherResponse represents the weather tool's response
type WeatherResponse struct {
    Location    string  `json:"location"`
    Temperature float64 `json:"temperature"`
    Conditions  string  `json:"conditions"`
    Units       string  `json:"units"`
}

// weatherTool is a mock weather tool
func weatherTool(req WeatherRequest) WeatherResponse {
    // Mock implementation
    return WeatherResponse{
        Location:    req.Location,
        Temperature: 22.0,
        Conditions:  "Sunny",
        Units:       req.Units,
    }
}

func setupTools() (tools.ToolRegistry, error) {
    // Create registry
    registry := tools.NewInMemoryToolRegistry()

    // Create tool definition from function
    weatherToolDef, err := tools.NewToolFromFunc(
        "get_weather",
        "Get current weather information for a specific location",
        weatherTool,
    )
    if err != nil {
        return nil, err
    }

    // Register tool
    err = registry.RegisterTool("get_weather", *weatherToolDef)
    if err != nil {
        return nil, err
    }

    return registry, nil
}
```

### Manual Tool Calling

For fine-grained control, use the Turn helpers directly:

```go
import (
    "context"
    "encoding/json"

    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/turns/toolblocks"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

func manualToolCalling(ctx context.Context, eng engine.Engine, registry tools.ToolRegistry, seed *turns.Turn) (*turns.Turn, error) {
    // Run inference
    updated, err := eng.RunInference(ctx, seed)
    if err != nil {
        return nil, err
    }

    calls := toolblocks.ExtractPendingToolCalls(updated)
    if len(calls) == 0 {
        return updated, nil
    }

    exec := tools.NewDefaultToolExecutor(tools.DefaultToolConfig())
    var execCalls []tools.ToolCall
    for _, call := range calls {
        args, _ := json.Marshal(call.Arguments)
        execCalls = append(execCalls, tools.ToolCall{
            ID: call.ID, Name: call.Name, Arguments: args,
        })
    }

    results, err := exec.ExecuteToolCalls(ctx, execCalls, registry)
    if err != nil {
        return nil, err
    }

    var shared []toolblocks.ToolResult
    for _, res := range results {
        if res == nil {
            continue
        }
        content := ""
        if res.Result != nil {
            if b, err := json.Marshal(res.Result); err == nil {
                content = string(b)
            }
        }
        shared = append(shared, toolblocks.ToolResult{
            ID: res.ID, Content: content, Error: res.Error,
        })
    }

    toolblocks.AppendToolResultsBlocks(updated, shared)
    return updated, nil
}
```

### Automated Tool Calling Loop

For most use cases, use the automated loop:

```go
import (
    "context"
    "time"

    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/toolloop"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

func automatedToolCalling(ctx context.Context, eng engine.Engine, registry tools.ToolRegistry, seed *turns.Turn) (*turns.Turn, error) {
    loop := toolloop.New(
        toolloop.WithEngine(eng),
        toolloop.WithRegistry(registry),
        toolloop.WithLoopConfig(toolloop.NewLoopConfig().WithMaxIterations(5)),
        toolloop.WithToolConfig(tools.DefaultToolConfig().
            WithExecutionTimeout(30*time.Second).
            WithMaxParallelTools(3).
            WithToolChoice(tools.ToolChoiceAuto)),
    )

    return loop.RunLoop(ctx, seed)
}
```

## Complete Tool Calling Example (with per-Turn tools)

Here's a complete example showing tool calling with streaming events (engine emits start/partial/final). Tools are attached to the Turn instead of the Engine.

```go
package main

import (
    "context"
    "fmt"
    "io"
    "time"

    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/inference/toolloop"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/turns"
    "golang.org/x/sync/errgroup"
)

func completeToolCallingExample(ctx context.Context, parsedLayers *layers.ParsedLayers, prompt string, w io.Writer) error {
    // 1. Create event router for streaming
    router, err := events.NewEventRouter()
    if err != nil {
        return fmt.Errorf("failed to create event router: %w", err)
    }
    defer router.Close()

    // 2. Add console printer for events (or use structured printer)
    // Handler signature is func(*message.Message) error
    router.AddHandler("chat", "chat", events.StepPrinterFunc("", w))
    // Alternative structured printer:
    // printer := events.NewStructuredPrinter(w, events.PrinterOptions{Format: events.FormatText})
    // router.AddHandler("chat", "chat", printer)

    // 3. Create watermill sink for publishing events
    watermillSink := middleware.NewWatermillSink(router.Publisher, "chat")

    // 4. Create engine (sinks are attached at runtime via context)
    baseEngine, err := factory.NewEngineFromParsedLayers(parsedLayers)
    if err != nil {
        return fmt.Errorf("failed to create engine: %w", err)
    }

    // 5. Set up tools
    registry := tools.NewInMemoryToolRegistry()

    // Register weather tool (from previous example)
    weatherToolDef, err := tools.NewToolFromFunc(
        "get_weather",
        "Get current weather information for a specific location",
        weatherTool,
    )
    if err != nil {
        return fmt.Errorf("failed to create weather tool: %w", err)
    }

    err = registry.RegisterTool("get_weather", *weatherToolDef)
    if err != nil {
        return fmt.Errorf("failed to register weather tool: %w", err)
    }

    // 6. Build seed Turn
    seed := &turns.Turn{Data: map[turns.TurnDataKey]any{}}
    turns.AppendBlock(seed, turns.NewSystemTextBlock(
        "You are a helpful assistant with access to weather information. Use the get_weather tool when users ask about weather.",
    ))
    turns.AppendBlock(seed, turns.NewUserTextBlock(prompt))

    // 7. Run inference with streaming in parallel
    eg := errgroup.Group{}
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    // Start event router
    eg.Go(func() error {
        defer cancel()
        return router.Run(ctx)
    })

    // Run inference with tool calling
    eg.Go(func() error {
        defer cancel()
        <-router.Running() // Wait for router to be ready

        // Attach engine sink to context so engine can stream
        runCtx := events.WithEventSinks(ctx, watermillSink)

        loop := toolloop.New(
            toolloop.WithEngine(baseEngine),
            toolloop.WithRegistry(registry),
            toolloop.WithLoopConfig(toolloop.NewLoopConfig().WithMaxIterations(5)),
            toolloop.WithToolConfig(tools.DefaultToolConfig().
                WithExecutionTimeout(30*time.Second).
                WithMaxParallelTools(3).
                WithToolChoice(tools.ToolChoiceAuto)),
        )

        // Run complete tool calling workflow
        updatedTurn, err := loop.RunLoop(runCtx, seed)
        if err != nil {
            return fmt.Errorf("tool calling failed: %w", err)
        }

        // Process final results
        for _, block := range updatedTurn.Blocks {
            if block.Kind == turns.BlockKindLLMText {
                if text, ok := block.Payload[turns.PayloadKeyText].(string); ok {
                    fmt.Fprintln(w, text)
                }
            }
        }

        return nil
    })

    return eg.Wait()
}
```

## Complete Runtime Flow

The sections above describe individual components. Here is how they connect into a single request flow, from user prompt to final response, in a multi-turn application:

```
1. Session.AppendNewTurnFromUserPrompt("question")
   ├── Clones the latest turn (preserving full conversation history)
   ├── Appends a new user block
   └── Assigns a new TurnID

2. Session.StartInference(ctx)
   ├── Creates an ExecutionHandle (tracks async result)
   └── Launches goroutine with runner.RunInference()

3. Runner setup
   ├── Attaches event sinks to context
   ├── Sets SessionID and InferenceID on Turn metadata
   └── Creates toolloop.Loop (if tool registry is present)

4. Tool loop iterates (up to maxIterations):
   │
   ├─ a. Snapshot: "pre_inference"
   │     Turn captured before any processing
   │
   ├─ b. Middleware chain (pre-processing)
   │     System prompt → agent mode → tool reorder → ...
   │     Each middleware can inspect/mutate the Turn
   │
   ├─ c. Engine.RunInference()
   │     ├── Translates Turn blocks to provider wire format
   │     ├── Calls LLM API
   │     ├── Streams events: start → delta → delta → ...
   │     └── Appends output blocks: llm_text, tool_call
   │
   ├─ d. Middleware chain (post-processing)
   │     Each middleware can inspect/mutate the result
   │
   ├─ e. Snapshot: "post_inference"
   │     Turn captured with model output
   │
   ├─ f. Extract pending tool calls
   │     If none: loop exits (done)
   │
   ├─ g. Execute tools in parallel
   │     Append tool_use blocks with results
   │
   ├─ h. Snapshot: "post_tools"
   │     Turn captured with tool results
   │
   └─ i. Loop back to (a) with updated Turn

5. Final turn persisted (if persister configured)

6. ExecutionHandle receives result
   Caller retrieves via handle.Wait()
```

This flow shows why snapshot phases are valuable for debugging: you can see the Turn at each critical moment and understand exactly what the model received and produced.

## Provider-Specific Implementations

The factory automatically selects the correct provider based on configuration:

### OpenAI Engine (Chat Completions)

```yaml
# Configuration for OpenAI
api:
  openai:
    api_key: "your-openai-key"
    model: "gpt-4"
    base_url: "https://api.openai.com/v1"
```

### OpenAI Responses Engine (Reasoning + Tools)

The OpenAI Responses API is supported via a dedicated engine package and is selected by setting `ai-api-type` to `openai-responses`. This engine streams reasoning summary ("thinking") and tool-call arguments in addition to normal output text deltas.

Key notes:
- Use `--ai-api-type=openai-responses` and a reasoning-capable model (e.g., `o4-mini`).
- The engine omits `temperature` and `top_p` for `o3/o4` families (these models reject sampling params).
- For function tools, the engine omits `tool_choice` (vendor-only values like `file_search` are not applicable).
- At trace log level, the engine prints a full YAML dump of the request payload to aid debugging.

Example (tools):

```bash
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type=openai-responses \
  --ai-engine=o4-mini \
  --mode=tools \
  --prompt='Please use get_weather to check the weather in San Francisco, in celsius.' \
  --log-level trace --verbose
```

Example (thinking only):

```bash
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type=openai-responses \
  --ai-engine=o4-mini \
  --mode=thinking \
  --prompt='Prove the sum of first n odd numbers equals n^2, stream reasoning summary.' \
  --log-level info
```

### Claude Engine

```yaml
# Configuration for Claude
api:
  claude:
    api_key: "your-claude-key"
    model: "claude-3-opus-20240229"
    base_url: "https://api.anthropic.com"
```

### Gemini Engine

```yaml
# Configuration for Gemini
api:
  gemini:
    api_key: "your-gemini-key"
    model: "gemini-pro"
```

## Middleware and Cross-Cutting Concerns

Add middleware for logging, metrics, and other cross-cutting concerns:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/session"
    "github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

func addMiddleware(baseEngine engine.Engine) session.EngineBuilder {
    // Add logging middleware
    loggingMiddleware := func(next middleware.HandlerFunc) middleware.HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            log.Info().Int("block_count", len(t.Blocks)).Msg("Starting inference")

            result, err := next(ctx, t)
            if err != nil {
                log.Error().Err(err).Msg("Inference failed")
            } else {
                log.Info().Int("result_count", len(result.Blocks)).Msg("Inference completed")
            }

            return result, err
        }
    }

    return enginebuilder.New(
        enginebuilder.WithBase(baseEngine),
        enginebuilder.WithMiddlewares(loggingMiddleware),
    )
}
```

## Testing and Mocking

The simple engine interface makes testing straightforward:

```go
type MockEngine struct{ add func(*turns.Turn) }

func (m *MockEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    if m.add != nil { m.add(t) }
    return t, nil
}
```

## Best Practices

When working with the inference engine architecture:

### Engine Design

- **Keep engines focused**: Only handle API communication
- **Use factories**: Always create engines through the factory pattern
- **Provider agnostic**: Write code that works with any provider
- **Error handling**: Handle API-specific errors appropriately

### Tool Calling

- **Use canonical surfaces**: wire tools via `toolloop.Loop` + `toolloop/enginebuilder` (no `toolhelpers`)
- **Configure limits**: Set reasonable iteration and timeout limits
- **Handle errors**: Configure appropriate error handling strategies
- **Test tools**: Test tool functions independently

### Performance

- **Streaming**: Use event sinks for real-time updates
- **Parallel execution**: Allow parallel tool execution when possible
- **Caching**: Consider caching for repeated operations
- **Timeouts**: Set appropriate timeouts for all operations

### Development

- **Mock engines**: Use mock engines for testing
- **Logging**: Add logging middleware for debugging
- **Configuration**: Use configuration layers for flexibility
- **Error handling**: Implement comprehensive error handling

## Debugging and Troubleshooting

### Enable Debug Logging

```go
import "github.com/rs/zerolog/log"

// Set log level to debug
log.Logger = log.Level(zerolog.DebugLevel)
```

### Debug Tool Execution

Tool execution emits logs and (optionally) events depending on your wiring (engine event sinks on `context.Context`, and tool execution settings in `tools.ToolConfig`):

```go
// Tool calling components can log detailed information about:
// - Tool call extraction
// - Tool execution steps
// - Result processing
// - Error handling
```

### Event Monitoring

Monitor events for real-time debugging through the Watermill router:

```go
// Add debug event handler that parses messages into events
router.AddHandler("debug", "chat", func(msg *message.Message) error {
    e, err := events.NewEventFromJson(msg.Payload)
    if err != nil { return err }
    log.Debug().Interface("event", e).Msg("Received event")
    msg.Ack()
    return nil
})
```

## Conclusion

The Geppetto inference engine architecture provides a clean, testable, and provider-agnostic foundation for AI applications. By separating API communication (engines) from orchestration logic (tool loop + middleware), the architecture achieves:

- **Simplicity**: Easy to understand and maintain
- **Flexibility**: Works with any AI provider
- **Testability**: Simple interfaces enable comprehensive testing
- **Composability**: Mix and match components as needed

The combination of engines, tool loop, factories, and middleware provides all the tools needed to build sophisticated AI applications while maintaining clean separation of concerns.

## See Also

- [Turns and Blocks](08-turns.md) — The Turn data model that engines operate on; see "How Blocks Accumulate"
- [Sessions](10-sessions.md) — Multi-turn session management built on top of engines
- [Tools](07-tools.md) — Defining and executing tools
- [Events](04-events.md) — How engines publish streaming events
- [Middlewares](09-middlewares.md) — Adding cross-cutting behavior; see "Middleware as Composable Prompting"
- [Structured Sinks](11-structured-sinks.md) — Extracting structured data from LLM text streams
- [Streaming Tutorial](../tutorials/01-streaming-inference-with-tools.md) — Complete working example
- Examples: `geppetto/cmd/examples/simple-streaming-inference/`, `geppetto/cmd/examples/generic-tool-calling/`
