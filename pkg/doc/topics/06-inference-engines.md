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
> **Outcome**: You will understand the architecture’s core concepts and learn how to instantiate engines, orchestrate tool calls, and apply best practices in production.

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

You can customize engine creation with options:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
)

func createEngineWithOptions(parsedLayers *layers.ParsedLayers) (engine.Engine, error) {
    // Create event sink for streaming
    watermillSink := middleware.NewWatermillSink(publisher, "chat")

    // Engine options for customization
    engineOptions := []engine.Option{
        engine.WithSink(watermillSink),
    }

    return factory.NewEngineFromParsedLayers(parsedLayers, engineOptions...)
}
```

## Basic Inference Without Tools

For simple text generation without tool calling:

```go
package main

import (
    "context"
    "fmt"

    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/conversation/builder"
    clay "github.com/go-go-golems/clay/pkg"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/glazed/pkg/cmds/logging"
    "github.com/spf13/cobra"
)

func simpleInference(ctx context.Context, parsedLayers *layers.ParsedLayers, prompt string) error {
    // 1. Create engine
    engine, err := factory.NewEngineFromParsedLayers(parsedLayers)
    if err != nil {
        return fmt.Errorf("failed to create engine: %w", err)
    }

    // 2. Build conversation
    builder := builder.NewManagerBuilder().
        WithSystemPrompt("You are a helpful assistant.").
        WithPrompt(prompt)

    manager, err := builder.Build()
    if err != nil {
        return fmt.Errorf("failed to build conversation: %w", err)
    }

    // 3. Run inference
    conversation := manager.GetConversation()
    updatedConversation, err := engine.RunInference(ctx, conversation)
    if err != nil {
        return fmt.Errorf("inference failed: %w", err)
    }

    // 4. Process results
    newMessages := updatedConversation[len(conversation):]
    for _, msg := range newMessages {
        if chatMsg, ok := msg.Content.(*conversation.ChatMessageContent); ok {
            fmt.Printf("%s: %s\n", chatMsg.Role, chatMsg.Text)
        }
    }

    return nil
}

func main() {
    // Minimal pattern for Geppetto-based tools
    root := &cobra.Command{Use: "example", PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        return logging.InitLoggerFromViper()
    }}
    _ = clay.InitViper("pinocchio", root)
    // add commands and execute...
}
```

## Tool Calling with Helpers

The `toolhelpers` package provides utilities for tool calling workflows. This separation allows engines to focus on API calls while helpers handle orchestration.

### Tool Helper Functions

The main helper functions are:

- `ExtractToolCalls`: Parse tool calls from AI responses
- `ExecuteToolCalls`: Execute tools and return results
- `AppendToolResults`: Add tool results to conversations
- `RunToolCallingLoop`: Complete automated workflow

### Appending user messages

To append a user message to a conversation managed by `conversation.Manager`:

```go
import "github.com/go-go-golems/geppetto/pkg/conversation"

if err := manager.AppendMessages(conversation.NewChatMessage(conversation.RoleUser, "Hello")); err != nil {
    // handle error
}
```

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

For fine-grained control, use individual helper functions:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
)

func manualToolCalling(ctx context.Context, engine engine.Engine, registry tools.ToolRegistry) error {
    // Initial conversation
    conversation := getInitialConversation()

    // Run inference
    response, err := engine.RunInference(ctx, conversation)
    if err != nil {
        return err
    }

    // Extract tool calls
    toolCalls := toolhelpers.ExtractToolCalls(response)
    if len(toolCalls) == 0 {
        // No tools called, we're done
        return nil
    }

    // Execute tools
    toolResults := toolhelpers.ExecuteToolCalls(ctx, toolCalls, registry)

    // Append results to conversation
    updatedConversation := toolhelpers.AppendToolResults(response, toolResults)

    // Continue with next iteration if needed...
    return nil
}
```

### Automated Tool Calling Loop

For most use cases, use the automated loop:

```go
import (
    "time"
    "github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
)

func automatedToolCalling(ctx context.Context, engine engine.Engine, registry tools.ToolRegistry, conversation conversation.Conversation) (conversation.Conversation, error) {
    // Configure tool calling behavior
    config := toolhelpers.NewToolConfig().
        WithMaxIterations(5).
        WithTimeout(30 * time.Second).
        WithMaxParallelTools(3).
        WithToolChoice(tools.ToolChoiceAuto).
        WithToolErrorHandling(tools.ToolErrorContinue)

    // Run complete workflow
    return toolhelpers.RunToolCallingLoop(ctx, engine, conversation, registry, config)
}
```

## Complete Tool Calling Example

Here's a complete example showing tool calling with streaming events (engine emits start/partial/final; helpers emit tool-call/tool-result via context):

```go
package main

import (
    "context"
    "fmt"
    "io"
    "time"

    "github.com/go-go-golems/geppetto/pkg/conversation/builder"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
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

    // 4. Create engine with streaming
    engineOptions := []engine.Option{
        engine.WithSink(watermillSink),
    }

    baseEngine, err := factory.NewEngineFromParsedLayers(parsedLayers, engineOptions...)
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

    // 6. Configure engine for tool calling (if supported)
    if configurableEngine, ok := baseEngine.(interface {
        ConfigureTools([]engine.ToolDefinition, engine.ToolConfig)
    }); ok {
        // Convert registry tools to engine format
        var engineTools []engine.ToolDefinition
        for _, tool := range registry.ListTools() {
            engineTool := engine.ToolDefinition{
                Name:        tool.Name,
                Description: tool.Description,
                Parameters:  tool.Parameters,
            }
            engineTools = append(engineTools, engineTool)
        }

        // Configure tools on engine
        engineConfig := engine.ToolConfig{
            Enabled:           true,
            ToolChoice:        engine.ToolChoiceAuto,
            MaxIterations:     1,
            ExecutionTimeout:  30 * time.Second,
            MaxParallelTools:  3,
            ToolErrorHandling: engine.ToolErrorContinue,
        }
        configurableEngine.ConfigureTools(engineTools, engineConfig)
    }

    // 7. Create tool helper configuration
    helperConfig := toolhelpers.NewToolConfig().
        WithMaxIterations(5).
        WithTimeout(30 * time.Second).
        WithMaxParallelTools(3).
        WithToolChoice(tools.ToolChoiceAuto).
        WithToolErrorHandling(tools.ToolErrorContinue)

    // 8. Build conversation
    conversationBuilder := builder.NewManagerBuilder().
        WithSystemPrompt("You are a helpful assistant with access to weather information. Use the get_weather tool when users ask about weather.").
        WithPrompt(prompt)

    manager, err := conversationBuilder.Build()
    if err != nil {
        return fmt.Errorf("failed to build conversation: %w", err)
    }

    conversation := manager.GetConversation()

    // 9. Run inference with streaming in parallel
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

        // Attach engine sink to context so helpers/tools can publish too
        runCtx := events.WithEventSinks(ctx, watermillSink)

        // Run complete tool calling workflow
        updatedConversation, err := toolhelpers.RunToolCallingLoop(
            runCtx, baseEngine, conversation, registry, helperConfig,
        )
        if err != nil {
            return fmt.Errorf("tool calling failed: %w", err)
        }

        // Process final results
        newMessages := updatedConversation[len(conversation):]
        for _, msg := range newMessages {
            if err := manager.AppendMessages(msg); err != nil {
                return fmt.Errorf("failed to append message: %w", err)
            }
        }

        return nil
    })

    return eg.Wait()
}
```

## Provider-Specific Implementations

The factory automatically selects the correct provider based on configuration:

### OpenAI Engine

```yaml
# Configuration for OpenAI
api:
  openai:
    api_key: "your-openai-key"
    model: "gpt-4"
    base_url: "https://api.openai.com/v1"
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
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
)

func addMiddleware(baseEngine engine.Engine) engine.Engine {
    // Add logging middleware
    loggingMiddleware := func(next middleware.HandlerFunc) middleware.HandlerFunc {
        return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
            log.Info().Int("message_count", len(messages)).Msg("Starting inference")

            result, err := next(ctx, messages)
            if err != nil {
                log.Error().Err(err).Msg("Inference failed")
            } else {
                log.Info().Int("result_count", len(result)).Msg("Inference completed")
            }

            return result, err
        }
    }

    return middleware.NewEngineWithMiddleware(baseEngine, loggingMiddleware)
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

- **Use helpers**: Let `toolhelpers` handle orchestration
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

The `toolhelpers` package includes extensive debug logging:

```go
// Tool calling helpers log detailed information about:
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

The Geppetto inference engine architecture provides a clean, testable, and provider-agnostic foundation for AI applications. By separating API communication (engines) from orchestration logic (helpers), the architecture achieves:

- **Simplicity**: Easy to understand and maintain
- **Flexibility**: Works with any AI provider
- **Testability**: Simple interfaces enable comprehensive testing
- **Composability**: Mix and match components as needed

The combination of engines, helpers, factories, and middleware provides all the tools needed to build sophisticated AI applications while maintaining clean separation of concerns.
