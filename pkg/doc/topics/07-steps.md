---
Title: Working with Steps for AI Inference and Tool Calling
Slug: geppetto-steps-inference-tool-calling
Short: A comprehensive guide to using the step abstraction in Geppetto for building AI inference pipelines, tool calling workflows, and composable operations.
Topics:
- geppetto
- steps
- inference
- tool-calling
- ai
- tutorial
- composition
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Working with Steps for AI Inference and Tool Calling

This tutorial provides a comprehensive guide to using the step abstraction in Geppetto for building AI inference pipelines, tool calling workflows, and composable operations. The step pattern enables you to create modular, reusable components that can be composed into complex AI workflows with features like streaming, cancellation, and event publishing.

## Core Concepts and Architecture

At its heart, Geppetto's step abstraction is designed around functional programming principles, particularly the concept of monads. This approach enables composition of complex operations while maintaining clean separation of concerns, robust error handling, and cancellation support.

### The Step Abstraction

The step abstraction in Geppetto is built around several key components:

- `Step`: A generic interface representing a computation that produces results
- `StepResult`: An interface representing computation results with monadic properties
- `Bind`: A function for composing steps into pipelines
- `PublisherManager`: A component that manages event publishing from steps

The entire system provides:

1. **Type-safe composition**: Steps can be chained with full type safety
2. **Cancellation support**: Operations can be cancelled at any point
3. **Error propagation**: Errors flow through the computation pipeline
4. **Event publishing**: Steps can publish events to notify observers of progress
5. **Streamable results**: Results can be delivered as streams for real-time feedback

### Step Interface

The core interface that all steps implement is defined in `pkg/steps/step.go`:

```go
type Step[T any, U any] interface {
    Start(ctx context.Context, input T) (StepResult[U], error)
    AddPublishedTopic(publisher message.Publisher, topic string) error
}
```

The `Step` interface is parameterized by:
- `T`: The input type
- `U`: The output type

This generic design allows for type-safe composition of steps. A step takes an input of type `T`, performs some computation, and produces a `StepResult` containing values of type `U`.

The `AddPublishedTopic` method enables steps to publish events to a specified topic using a Watermill publisher. This allows external components to observe the step's progress and receive intermediate results.

### StepResult Interface

The `StepResult` interface represents the result of a step's execution:

```go
type StepResult[T any] interface {
    Return() []helpers.Result[T]
    GetChannel() <-chan helpers.Result[T]
    Cancel()
    GetMetadata() *StepMetadata
}
```

This interface embodies several monadic patterns:

1. **List Monad**: It can contain multiple results via the channel, allowing for streaming
2. **Maybe Monad**: Results can be values or errors (via `helpers.Result`)
3. **Cancellation Monad**: Operations can be cancelled through the `Cancel` method
4. **Metadata Monad**: Carries metadata about the computation via `GetMetadata`

The `GetChannel` method returns a channel of `helpers.Result[T]`, which allows consumers to receive results as they become available, enabling streaming workflows.

### StepMetadata Structure

Each step result includes metadata that provides context about the execution:

```go
type StepMetadata struct {
    StepID     uuid.UUID              `json:"step_id"`
    Type       string                 `json:"type"`
    InputType  string                 `json:"input_type"`
    OutputType string                 `json:"output_type"`
    Metadata   map[string]interface{} `json:"meta"`
}
```

This metadata is particularly useful for:
- Tracking step executions across a distributed system
- Providing type information for dynamic handlers
- Storing provider-specific settings and configuration
- Passing context to event subscribers

## Creating Basic Steps

Let's start by exploring how to create and use basic steps in Geppetto.

### Lambda Steps

The simplest type of step is the lambda step, which wraps a function into the `Step` interface. Lambda steps are defined in `pkg/steps/utils/lambda.go` and are perfect for simple transformations:

```go
package main

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/go-go-golems/geppetto/pkg/helpers"
    "github.com/go-go-golems/geppetto/pkg/steps/utils"
)

func main() {
    // Create a lambda step that converts text to uppercase
    uppercaseStep := &utils.LambdaStep[string, string]{
        Function: func(input string) helpers.Result[string] {
            return helpers.NewValueResult(strings.ToUpper(input))
        },
    }
    
    // Execute the step
    ctx := context.Background()
    result, err := uppercaseStep.Start(ctx, "hello world")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    // Process the results
    for _, r := range result.Return() {
        value, err := r.Value()
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        fmt.Println(value) // Outputs: HELLO WORLD
    }
}
```

Lambda steps are ideal for:
- Simple transformations of data
- Filtering results
- Converting between types
- Implementing adapters between different step types

### Manually Implementing a Step

For more complex behavior, you can implement the `Step` interface directly:

```go
package main

import (
    "context"
    
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/go-go-golems/geppetto/pkg/helpers"
    "github.com/go-go-golems/geppetto/pkg/steps"
)

// CounterStep counts from 1 to N and streams the results
type CounterStep struct {
    publisherManager *events.PublisherManager
}

func NewCounterStep() *CounterStep {
    return &CounterStep{
        publisherManager: events.NewPublisherManager(),
    }
}

// Start implements the Step interface
func (s *CounterStep) Start(ctx context.Context, count int) (steps.StepResult[int], error) {
    // Create a channel for results
    resultChan := make(chan helpers.Result[int])
    
    // Set up cancellation
    ctx, cancel := context.WithCancel(ctx)
    
    // Create step metadata
    stepMetadata := &steps.StepMetadata{
        StepID:     uuid.New(),
        Type:       "counter",
        InputType:  "int",
        OutputType: "int",
        Metadata:   map[string]interface{}{},
    }
    
    // Create event metadata
    metadata := events.EventMetadata{
        ID:       uuid.New().String(),
        ParentID: "",
    }
    
    // Start a goroutine to produce results
    go func() {
        defer close(resultChan)
        
        // Publish start event
        s.publisherManager.PublishBlind(
            events.NewStartEvent(metadata, stepMetadata),
        )
        
        for i := 1; i <= count; i++ {
            select {
            case <-ctx.Done():
                // Context was cancelled
                s.publisherManager.PublishBlind(
                    events.NewInterruptEvent(metadata, stepMetadata, ""),
                )
                return
            default:
                // Publish progress event
                s.publisherManager.PublishBlind(
                    events.NewPartialCompletionEvent(
                        metadata, 
                        stepMetadata, 
                        fmt.Sprintf("%d", i),
                        fmt.Sprintf("Counted to %d", i),
                    ),
                )
                
                // Send the result
                resultChan <- helpers.NewValueResult(i)
                
                // Simulate work
                time.Sleep(100 * time.Millisecond)
            }
        }
        
        // Publish completion event
        s.publisherManager.PublishBlind(
            events.NewFinalEvent(
                metadata, 
                stepMetadata, 
                fmt.Sprintf("Finished counting to %d", count),
            ),
        )
    }()
    
    // Create and return the StepResult
    return steps.NewStepResult[int](
        resultChan,
        steps.WithCancel[int](cancel),
        steps.WithMetadata[int](stepMetadata),
    ), nil
}

// AddPublishedTopic implements the Step interface
func (s *CounterStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
    s.publisherManager.RegisterPublisher(topic, publisher)
    return nil
}
```

Custom step implementations are useful when you need:
- Complex internal state management
- Custom cancellation logic
- Fine-grained control over event publishing
- Integration with external systems or resources

## AI Inference Steps

Now let's look at how to create and configure AI inference steps for different providers.

### OpenAI Chat Step

To create a step that performs inference using OpenAI's chat models:

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func main() {
    // Create and configure step settings
    stepSettings, err := settings.NewStepSettings()
    if err != nil {
        fmt.Printf("Error creating settings: %v\n", err)
        return
    }
    
    // Configure OpenAI settings
    apiType := types.ApiTypeOpenAI
    stepSettings.Chat.ApiType = &apiType
    
    engine := "gpt-4"
    stepSettings.Chat.Engine = &engine
    
    // Enable streaming
    stepSettings.Chat.Stream = true
    
    // Set temperature
    temperature := 0.7
    stepSettings.Chat.Temperature = &temperature
    
    // Set max tokens
    maxTokens := 1000
    stepSettings.Chat.MaxResponseTokens = &maxTokens
    
    // Set API key and base URL
    stepSettings.API.APIKeys = map[string]string{
        "openai-api-key": os.Getenv("OPENAI_API_KEY"),
    }
    stepSettings.API.BaseUrls = map[string]string{
        "openai-base-url": "https://api.openai.com/v1",
    }
    
    // Create the OpenAI chat step
    chatStep, err := openai.NewStep(stepSettings)
    if err != nil {
        fmt.Printf("Error creating step: %v\n", err)
        return
    }
    
    // Create the conversation input
    messages := []*conversation.Message{
        conversation.NewChatMessage(conversation.RoleSystem, 
            "You are a helpful assistant that provides concise answers."),
        conversation.NewChatMessage(conversation.RoleUser, 
            "What is the capital of France?"),
    }
    
    // Run the step
    ctx := context.Background()
    result, err := chatStep.Start(ctx, messages)
    if err != nil {
        fmt.Printf("Error starting step: %v\n", err)
        return
    }
    
    // Process the streaming results
    for res := range result.GetChannel() {
        if res.Error() != nil {
            fmt.Printf("Error: %v\n", res.Error())
            continue
        }
        
        message, _ := res.Value()
        fmt.Print(message.Content.String())
    }
}
```

### Claude Chat Step

To use Anthropic's Claude models instead:

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func main() {
    // Create step settings
    stepSettings, err := settings.NewStepSettings()
    if err != nil {
        fmt.Printf("Error creating settings: %v\n", err)
        return
    }
    
    // Configure Claude settings
    apiType := types.ApiTypeClaude
    stepSettings.Chat.ApiType = &apiType
    
    engine := "claude-3-opus-20240229"
    stepSettings.Chat.Engine = &engine
    
    // Enable streaming
    stepSettings.Chat.Stream = true
    
    // Set temperature
    temperature := 0.5
    stepSettings.Chat.Temperature = &temperature
    
    // Set max tokens
    maxTokens := 2000
    stepSettings.Chat.MaxResponseTokens = &maxTokens
    
    // Set API key and base URL
    stepSettings.API.APIKeys = map[string]string{
        "claude-api-key": os.Getenv("ANTHROPIC_API_KEY"),
    }
    stepSettings.API.BaseUrls = map[string]string{
        "claude-base-url": "https://api.anthropic.com",
    }
    
    // Create Claude chat step
    chatStep, err := claude.NewChatStep(stepSettings, nil)
    if err != nil {
        fmt.Printf("Error creating step: %v\n", err)
        return
    }
    
    // Create conversation input
    messages := []*conversation.Message{
        conversation.NewChatMessage(conversation.RoleSystem, 
            "You are Claude, a helpful AI assistant."),
        conversation.NewChatMessage(conversation.RoleUser, 
            "Explain quantum computing in simple terms."),
    }
    
    // Run the step
    ctx := context.Background()
    result, err := chatStep.Start(ctx, messages)
    if err != nil {
        fmt.Printf("Error starting step: %v\n", err)
        return
    }
    
    // Process streaming results
    for res := range result.GetChannel() {
        if res.Error() != nil {
            fmt.Printf("Error: %v\n", res.Error())
            continue
        }
        
        message, _ := res.Value()
        fmt.Print(message.Content.String())
    }
}
```

### Common Configuration Options

The `StepSettings` struct provides a unified configuration interface for all AI providers:

```go
type StepSettings struct {
    // API settings for different providers
    API *API `json:"api" yaml:"api"`
    
    // Client settings for HTTP requests
    Client *ClientSettings `json:"client" yaml:"client"`
    
    // Chat-specific settings
    Chat *ChatSettings `json:"chat" yaml:"chat"`
    
    // Provider-specific settings
    Claude *ClaudeSettings `json:"claude" yaml:"claude"`
    OpenAI *OpenAISettings `json:"openai" yaml:"openai"`
    // Other providers...
}
```

Key configuration options include:

1. **Basic Parameters**:
   - `ApiType`: The provider to use (OpenAI, Claude, etc.)
   - `Engine`: The specific model to use
   - `Stream`: Whether to stream results or receive them all at once

2. **Generation Parameters**:
   - `Temperature`: Controls randomness (0.0 to 1.0)
   - `TopP`: Controls diversity via nucleus sampling
   - `MaxResponseTokens`: Maximum tokens to generate
   - `Stop`: Custom stop sequences

3. **API Configuration**:
   - `APIKeys`: Map of API keys for different providers
   - `BaseUrls`: Map of base URLs for API endpoints

4. **Client Settings**:
   - `Timeout`: HTTP request timeout
   - `MaxRetries`: Number of retries for failed requests
   - `RetryBackoff`: Backoff strategy for retries

## Composing Steps with Bind

The real power of the step abstraction comes from composition. The `Bind` function in `pkg/steps/step.go` implements monadic binding, allowing you to chain steps together:

```go
func Bind[T any, U any](
    ctx context.Context,
    m StepResult[T],
    step Step[T, U],
) StepResult[U] {
    // Implementation...
}
```

Let's see how to use `Bind` to create a simple pipeline:

```go
package main

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/helpers"
    "github.com/go-go-golems/geppetto/pkg/steps"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/geppetto/pkg/steps/utils"
)

func main() {
    // Create step settings and chat step
    stepSettings := createOpenAISettings()
    chatStep, _ := openai.NewStep(stepSettings)
    
    // Create uppercase lambda step
    uppercaseStep := &utils.LambdaStep[*conversation.Message, string]{
        Function: func(msg *conversation.Message) helpers.Result[string] {
            return helpers.NewValueResult(strings.ToUpper(msg.Content.String()))
        },
    }
    
    // Create a length counting step
    countStep := &utils.LambdaStep[string, int]{
        Function: func(text string) helpers.Result[int] {
            return helpers.NewValueResult(len(text))
        },
    }
    
    // Create input
    messages := []*conversation.Message{
        conversation.NewChatMessage(conversation.RoleUser, "Tell me a short joke."),
    }
    
    // Create the pipeline
    ctx := context.Background()
    
    // Step 1: Generate text with AI
    chatResult, _ := chatStep.Start(ctx, messages)
    
    // Step 2: Convert to uppercase
    uppercaseResult := steps.Bind(ctx, chatResult, uppercaseStep)
    
    // Step 3: Count characters
    countResult := steps.Bind(ctx, uppercaseResult, countStep)
    
    // Process final results
    for res := range countResult.GetChannel() {
        if res.Error() != nil {
            fmt.Printf("Error: %v\n", res.Error())
            continue
        }
        
        length, _ := res.Value()
        fmt.Printf("The joke has %d characters.\n", length)
    }
}
```

This example recreates and extends what's found in `pinocchio/cmd/experiments/agent/uppercase.go`.

### Real-world Example: Uppercase Transformation

Here's the complete example from `pinocchio/cmd/experiments/agent/uppercase.go`, which demonstrates a simple pipeline that:
1. Uses OpenAI to generate a response
2. Converts the response to uppercase
3. Outputs the result

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/helpers"
    "github.com/go-go-golems/geppetto/pkg/steps"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/geppetto/pkg/steps/utils"
    "github.com/go-go-golems/glazed/pkg/cli"
    "github.com/go-go-golems/glazed/pkg/cmds/layers"
    "github.com/go-go-golems/pinocchio/pkg/cmds"
    "github.com/spf13/cobra"
)

var upperCaseCmd = &cobra.Command{
    Use:   "uppercase",
    Short: "uppercase test",
    Run: func(cmd *cobra.Command, args []string) {
        // Create step settings from command flags
        stepSettings, err := settings.NewStepSettings()
        cobra.CheckErr(err)
        
        // Set up layers for configuration
        geppettoLayers, err := cmds.CreateGeppettoLayers(stepSettings, cmds.WithHelpersLayer())
        cobra.CheckErr(err)

        layers_ := layers.NewParameterLayers(layers.WithLayers(geppettoLayers...))

        // Parse command flags
        cobraParser, err := cli.NewCobraParserFromLayers(
            layers_,
            cli.WithCobraMiddlewaresFunc(
                cmds.GetCobraCommandGeppettoMiddlewares,
            ))
        cobra.CheckErr(err)

        parsedLayers, err := cobraParser.Parse(cmd, args)
        cobra.CheckErr(err)

        // Update settings from parsed flags
        err = stepSettings.UpdateFromParsedLayers(parsedLayers)
        cobra.CheckErr(err)

        // Set up context with cancellation
        ctx, cancel := context.WithCancel(cmd.Context())
        defer cancel()
        
        // Create input messages
        messages := []*conversation.Message{
            conversation.NewChatMessage(conversation.RoleUser, "Hello, my friend?"),
        }

        // Enable streaming
        stepSettings.Chat.Stream = true
        
        // Create LLM completion step
        step, err := openai.NewStep(stepSettings)
        cobra.CheckErr(err)

        // Create uppercase lambda step
        uppercaseStep := &utils.LambdaStep[*conversation.Message, string]{
            Function: func(s *conversation.Message) helpers.Result[string] {
                return helpers.NewValueResult(strings.ToUpper(s.Content.String()))
            },
        }

        // Start the LLM completion
        res, err := step.Start(ctx, messages)
        cobra.CheckErr(err)

        // Chain the result through the uppercaseStep
        res_ := steps.Bind[*conversation.Message, string](ctx, res, uppercaseStep)

        // Process and print results
        c := res_.GetChannel()
        for i := range c {
            s, err := i.Value()
            cobra.CheckErr(err)
            fmt.Printf("%s", s)
        }
    },
}
```

## Event Publishing and Subscription

Steps publish events to inform subscribers about their progress and status. The Watermill library provides the messaging infrastructure for this event system.

### Publishing Events

Steps publish events using their `PublisherManager`:

```go
// Inside a step implementation
metadata := events.EventMetadata{
    ID:       uuid.New().String(),
    ParentID: parentID,
}

stepMetadata := &steps.StepMetadata{
    StepID:     uuid.New(),
    Type:       "my-step",
    // ...
}

// Publish start event
publisherManager.PublishBlind(
    events.NewStartEvent(metadata, stepMetadata),
)

// Publish partial completion
publisherManager.PublishBlind(
    events.NewPartialCompletionEvent(
        metadata, 
        stepMetadata, 
        chunk, 
        accumulatedText,
    ),
)

// Publish final event
publisherManager.PublishBlind(
    events.NewFinalEvent(metadata, stepMetadata, finalText),
)

// Publish error event
publisherManager.PublishBlind(
    events.NewErrorEvent(metadata, stepMetadata, err),
)
```

### Subscribing to Events

To subscribe to events from a step:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/ThreeDotsLabs/watermill"
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
)

func main() {
    // Create Watermill publisher/subscriber
    pubSub := gochannel.NewGoChannel(
        gochannel.Config{},
        watermill.NewStdLogger(false, false),
    )
    
    // Create chat step (configuration omitted)
    chatStep, _ := openai.NewStep(stepSettings)
    
    // Register publisher for a topic
    topic := "chat-events"
    err := chatStep.AddPublishedTopic(pubSub, topic)
    if err != nil {
        fmt.Printf("Error adding published topic: %v\n", err)
        return
    }
    
    // Subscribe to events
    messages, err := pubSub.Subscribe(context.Background(), topic)
    if err != nil {
        fmt.Printf("Error subscribing: %v\n", err)
        return
    }
    
    // Process events in a goroutine
    go func() {
        for msg := range messages {
            eventJSON := msg.Payload
            event, _ := events.NewEventFromJson(eventJSON)
            
            // Handle different event types
            switch event.Type {
            case events.EventTypeStart:
                fmt.Println("Step started")
                
            case events.EventTypePartialCompletion:
                partialEvent, _ := events.NewPartialCompletionEventFromJSON(eventJSON)
                fmt.Printf("Partial: %s\n", partialEvent.Delta)
                
            case events.EventTypeFinal:
                finalEvent, _ := events.NewFinalEventFromJSON(eventJSON)
                fmt.Printf("Final: %s\n", finalEvent.Content)
                
            case events.EventTypeToolCall:
                toolCallEvent, _ := events.NewToolCallEventFromJSON(eventJSON)
                fmt.Printf("Tool call: %s(%s)\n", 
                    toolCallEvent.ToolCall.Name, 
                    toolCallEvent.ToolCall.Input)
                
            case events.EventTypeToolResult:
                toolResultEvent, _ := events.NewToolResultEventFromJSON(eventJSON)
                fmt.Printf("Tool result: %s\n", toolResultEvent.ToolResult.Result)
                
            case events.EventTypeError:
                fmt.Printf("Error: %s\n", event.Content)
                
            case events.EventTypeInterrupt:
                fmt.Printf("Interrupted: %s\n", event.Content)
            }
            
            msg.Ack()
        }
    }()
    
    // Run the chat step (code omitted)
}
```

### Event Types

Steps publish various event types defined in `pkg/events/chat-events.go`:

1. `EventTypeStart`: Signals the beginning of a step's execution
2. `EventTypePartialCompletion`: Represents an incremental chunk during streaming
3. `EventTypeToolCall`: Published when the AI requests a tool/function call
4. `EventTypeFinal`: Signals successful completion with the final result
5. `EventTypeToolResult`: Published when a tool provides its result
6. `EventTypeInterrupt`: Published if the step is interrupted via cancellation
7. `EventTypeError`: Published when an error occurs during execution

## Tool Calling in Steps

Tool calling enables AI models to invoke functions. Geppetto provides structured steps for implementing and using tools.

### Implementing Tool Functions

First, define your tool functions with proper JSON schema annotations:

```go
// Weather data structure with JSON Schema annotations
type WeatherParams struct {
    Location string `json:"location" jsonschema:"description=The city and state/country to get weather for,required"`
    Units    string `json:"units,omitempty" jsonschema:"description=The units to use (celsius or fahrenheit),enum=celsius,enum=fahrenheit"`
}

// Function to get current weather
func getWeather(params WeatherParams) string {
    // Implementation to fetch weather data
    return fmt.Sprintf("The weather in %s is currently sunny and 75°F.", params.Location)
}

// Function to get weather on a specific day
type WeatherDayParams struct {
    Location string `json:"location" jsonschema:"description=The city and state/country,required"`
    Date     string `json:"date" jsonschema:"description=The date to get weather for (YYYY-MM-DD),required,format=date"`
}

func getWeatherOnDay(params WeatherDayParams) string {
    // Implementation to fetch historical weather
    return fmt.Sprintf("On %s, the weather in %s was partly cloudy with a high of 68°F.", 
        params.Date, params.Location)
}
```

### Using ChatToolStep with OpenAI

The `ChatToolStep` in OpenAI implementation creates a step that can call tools:

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
    "github.com/invopop/jsonschema"
)

func main() {
    // Create step settings
    stepSettings, err := settings.NewStepSettings()
    if err != nil {
        fmt.Printf("Error creating settings: %v\n", err)
        return
    }
    
    // Configure settings
    apiType := types.ApiTypeOpenAI
    stepSettings.Chat.ApiType = &apiType
    engine := "gpt-4"
    stepSettings.Chat.Engine = &engine
    stepSettings.Chat.Stream = true
    
    // Set API credentials
    stepSettings.API.APIKeys = map[string]string{
        "openai-api-key": os.Getenv("OPENAI_API_KEY"),
    }
    stepSettings.API.BaseUrls = map[string]string{
        "openai-base-url": "https://api.openai.com/v1",
    }
    
    // Create JSON schema reflector
    reflector := &jsonschema.Reflector{
        DoNotReference: true,
    }
    
    // Create tool step with functions
    toolStep, err := openai.NewChatToolStep(
        stepSettings,
        openai.WithReflector(reflector),
        openai.WithToolFunctions(map[string]any{
            "getWeather":      getWeather,
            "getWeatherOnDay": getWeatherOnDay,
        }),
    )
    if err != nil {
        fmt.Printf("Error creating tool step: %v\n", err)
        return
    }
    
    // Create conversation input
    messages := conversation.NewManager(
        conversation.WithMessages(
            conversation.NewChatMessage(
                conversation.RoleSystem,
                "You are a helpful weather assistant.",
            ),
            conversation.NewChatMessage(
                conversation.RoleUser,
                "What's the weather in Boston today and what was it on November 9th, 1924?",
            ),
        ),
    )

    // Run the tool step
    ctx := context.Background()
    result, err := toolStep.Start(ctx, messages.GetConversation())
    if err != nil {
        fmt.Printf("Error starting step: %v\n", err)
        return
    }
    
    // Process results
    for res := range result.GetChannel() {
        if res.Error() != nil {
            fmt.Printf("Error: %v\n", res.Error())
            continue
        }
        
        message, _ := res.Value()
        fmt.Println(message.Content.String())
    }
}
```

### Understanding the Tool Step Implementation

The `ChatToolStep` in `geppetto/pkg/steps/ai/openai/chat-execute-tool-step.go` combines several steps internally to handle the tool calling workflow:

1.  `ChatWithToolsStep`: Sends the prompt and tool definitions to the AI model. It receives either a text response or requests to call specific tools.
2.  `ExecuteToolStep`: If tool calls are requested, this step invokes the actual Go functions associated with the tools and collects their results.
3.  `LambdaStep` (internal helper): Converts the raw tool execution results into a structured `conversation.Message` with `RoleTool`. This message can then be processed further or sent back to the AI model in subsequent turns if needed.

The flow works like this:

1.  User input is sent to the AI model along with available tool definitions.
2.  The AI generates text or decides to call one or more tools based on the prompt.
3.  If tool calls are generated, they are extracted and passed to the corresponding Go functions for execution.
4.  The results from the executed tools are collected.
5.  These results are formatted into a `conversation.Message` (or multiple messages).
6.  This final message (containing tool results or the AI's text response) is emitted by the `ChatToolStep`.

### Claude Tool Support

Claude's tool calling works similarly conceptually but uses Claude-specific API formats for defining tools. You configure the `claude.ChatStep` with these definitions:

```go
// Create Claude tool definitions
tools := []api.Tool{
    {
        Name:        "getWeather",
        Description: "Get the current weather for a location",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "location": map[string]interface{}{
                    "type":        "string",
                    "description": "The city and state/country to get weather for",
                },
            },
            "required": []string{"location"},
        },
    },
    // Add more tool definitions as needed
}

// Create Claude chat step configured to use tools
// Note: This configures the step to *send* tool definitions to Claude.
// Handling the *execution* if Claude requests a tool call requires
// additional steps to be bound after this chat step.
chatStep, err := claude.NewChatStep(stepSettings, tools)
if err != nil {
    fmt.Printf("Error creating step: %v\n", err)
    return
}
```

## Creating a Complete Chat Application

Building on these concepts, let's create a complete chat application with tool calling, based on `pinocchio/cmd/experiments/tool-ui/main.go`:

```go
package main

import (
    "context"
    "os"

    "github.com/ThreeDotsLabs/watermill/message"
    bobachat "github.com/go-go-golems/bobatea/pkg/chat"
    clay "github.com/go-go-golems/clay/pkg"
    geppetto_conversation "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
    glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
    "github.com/go-go-golems/glazed/pkg/cmds/layers"
    "github.com/go-go-golems/glazed/pkg/cmds/parameters"
    "github.com/go-go-golems/pinocchio/pkg/chatrunner"
    pinocchio_cmds "github.com/go-go-golems/pinocchio/pkg/cmds"
    "github.com/invopop/jsonschema"
    "github.com/pkg/errors"
    "github.com/spf13/cobra"
)

// Weather functions (implementation omitted)
func getWeather(params WeatherParams) string {
    // Implementation
}

func getWeatherOnDay(params WeatherDayParams) string {
    // Implementation
}

// ToolUiCommand struct
type ToolUiCommand struct {
    *glazed_cmds.CommandDescription
}

var _ glazed_cmds.BareCommand = (*ToolUiCommand)(nil)

// NewToolUiCommand creates a new command for the tool UI
func NewToolUiCommand() (*ToolUiCommand, error) {
    stepSettings, err := settings.NewStepSettings()
    if err != nil {
        return nil, err
    }
    
    geppettoLayers, err := pinocchio_cmds.CreateGeppettoLayers(stepSettings, pinocchio_cmds.WithHelpersLayer())
    if err != nil {
        return nil, err
    }

    return &ToolUiCommand{
        CommandDescription: glazed_cmds.NewCommandDescription(
            "tool-ui",
            glazed_cmds.WithShort("Tool UI Example using ChatRunner"),
            glazed_cmds.WithFlags(
                parameters.NewParameterDefinition(
                    "ui",
                    parameters.ParameterTypeBool,
                    parameters.WithDefault(false),
                    parameters.WithHelp("start in interactive chat UI mode")),
            ),
            glazed_cmds.WithLayersList(geppettoLayers...),
        ),
    }, nil
}

// Settings for the tool UI
type ToolUiSettings struct {
    UI bool `glazed.parameter:"ui"`
}

// Run implements the BareCommand interface
func (t *ToolUiCommand) Run(
    ctx context.Context,
    parsedLayers *layers.ParsedLayers,
) error {
    // Parse settings
    settings_ := &ToolUiSettings{}
    err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings_)
    if err != nil {
        return err
    }

    // Create step settings from parsed layers
    stepSettings, err := settings.NewStepSettingsFromParsedLayers(parsedLayers)
    if err != nil {
        return err
    }
    
    // Configure settings
    stepSettings.Chat.Stream = true
    engine := "gpt-4o-mini"
    stepSettings.Chat.Engine = &engine
    apiType := types.ApiTypeOpenAI
    stepSettings.Chat.ApiType = &apiType

    // Create conversation manager with initial prompt
    manager := geppetto_conversation.NewManager(
        geppetto_conversation.WithMessages(
            geppetto_conversation.NewChatMessage(
                geppetto_conversation.RoleUser,
                "Give me the weather in Boston on november 9th 1924, please, including the windspeed for me, an old ass american. Also, the weather in paris today, with temperature.",
            ),
        ))

    // Set up JSON schema reflector
    reflector := &jsonschema.Reflector{
        DoNotReference: true,
    }
    err = reflector.AddGoComments("github.com/go-go-golems/pinocchio", "./cmd/experiments/tool-ui")
    if err != nil {
        log.Warn().Err(err).Msg("Could not add go comments")
    }

    // Create step factory for the chat runner
    stepFactory := func(publisher message.Publisher, topic string) (chat.Step, error) {
        toolStep, err := openai.NewChatToolStep(
            stepSettings.Clone(),
            openai.WithReflector(reflector),
            openai.WithToolFunctions(map[string]any{
                "getWeather":      getWeather,
                "getWeatherOnDay": getWeatherOnDay,
            }),
        )
        if err != nil {
            return nil, errors.Wrap(err, "failed to create tool step")
        }

        if publisher != nil && topic != "" {
            err = toolStep.AddPublishedTopic(publisher, topic)
            if err != nil {
                return nil, errors.Wrapf(err, "failed to add published topic %s", topic)
            }
        }
        return toolStep, nil
    }

    // Determine the run mode (UI or blocking)
    var mode chatrunner.RunMode
    if settings_.UI {
        mode = chatrunner.RunModeChat
    } else {
        mode = chatrunner.RunModeBlocking
    }

    // Build the chat session
    builder := chatrunner.NewChatBuilder().
        WithMode(mode).
        WithManager(manager).
        WithStepFactory(stepFactory).
        WithContext(ctx).
        WithOutputWriter(os.Stdout).
        WithUIOptions(bobachat.WithTitle("Tool UI Chat"))

    // Create and run the session
    session, err := builder.Build()
    if err != nil {
        return errors.Wrap(err, "failed to build chat session")
    }

    err = session.Run()
    if err != nil {
        return errors.Wrap(err, "chat session failed")
    }

    return nil
}

// Main function sets up and runs the command
func main() {
    toolUiCommand, err := NewToolUiCommand()
    cobra.CheckErr(err)

    toolUICobraCommand, err := pinocchio_cmds.BuildCobraCommandWithGeppettoMiddlewares(toolUiCommand)
    cobra.CheckErr(err)

    err = clay.InitViper("pinocchio", toolUICobraCommand)
    cobra.CheckErr(err)

    err = toolUICobraCommand.Execute()
    cobra.CheckErr(err)
}
```

## Best Practices

When working with steps in Geppetto, follow these best practices for robust, maintainable code.

### Context Handling

Always manage contexts correctly to prevent resource leaks:

```go
// Create a cancellable context
ctx, cancel := context.WithCancel(parentCtx)
defer cancel() // Ensure cancellation when function returns

// Forward cancellation to child operations
go func() {
    <-ctx.Done()
    // Clean up resources
}()

// Include context in step execution
result, err := step.Start(ctx, input)
```

### Step Design Principles

1. **Single Responsibility**: Each step should do one thing well
2. **Statelessness**: Minimize internal state in steps
3. **Resource Management**: Always clean up resources on cancellation
4. **Error Handling**: Propagate errors clearly and consistently
5. **Event Richness**: Publish events for all significant state changes

### Composition Patterns

Prefer composing multiple small steps over complex monolithic ones:

```go
// Bad: Single monolithic step
result, _ := complexStep.Start(ctx, input)

// Good: Composition of simpler steps
result1, _ := step1.Start(ctx, input)
result2 := steps.Bind(ctx, result1, step2)
result3 := steps.Bind(ctx, result2, step3)
```

### Type Safety

Leverage Go's type system for safety:

```go
// Explicitly specify type parameters
result := steps.Bind[*conversation.Message, string](ctx, chatResult, uppercaseStep)

// Create type-safe adapters between incompatible steps
adaptStep := &utils.LambdaStep[TypeA, TypeB]{
    Function: func(a TypeA) helpers.Result[TypeB] {
        // Convert TypeA to TypeB
    },
}
```

### Memory Management

Handle resource cleanup properly:

```go
// In your step implementation
func (s *MyStep) Start(ctx context.Context, input T) (steps.StepResult[U], error) {
    // Create resources
    resource := createResource()
    
    // Create cancellation context
    childCtx, cancel := context.WithCancel(ctx)
    
    // Set up cleanup
    cleanup := func() {
        cancel()
        resource.Close()
    }
    
    // Ensure cleanup on context cancellation
    go func() {
        <-ctx.Done()
        cleanup()
    }()
    
    // Create result with cleanup handler
    result := steps.NewStepResult[U](
        resultChannel,
        steps.WithCancel[U](cleanup),
    )
    
    return result, nil
}
```

### Error Handling

Propagate errors properly:

```go
// Return errors from Start immediately
func (s *MyStep) Start(ctx context.Context, input T) (steps.StepResult[U], error) {
    if !validInput(input) {
        return nil, fmt.Errorf("invalid input: %v", input)
    }
    
    // Continue with valid input
}

// Send errors through the result channel for runtime errors
go func() {
    defer close(resultChan)
    
    result, err := performOperation()
    if err != nil {
        // Publish error event
        s.publisherManager.PublishBlind(
            events.NewErrorEvent(metadata, stepMetadata, err),
        )
        
        // Send error to channel
        resultChan <- helpers.NewErrorResult[U](err)
        return
    }
    
    // Send success result
    resultChan <- helpers.NewValueResult(result)
}()
```

### Tool Function Design

Design tool functions with clear types and JSON schema annotations:

```go
// Use descriptive types with JSON schema annotations
type WeatherParams struct {
    Location string `json:"location" jsonschema:"description=The location to get weather for,required"`
    Units    string `json:"units,omitempty" jsonschema:"description=The units to use (celsius or fahrenheit),enum=celsius,enum=fahrenheit"`
}

// Return informative results or errors
func getWeather(params WeatherParams) (string, error) {
    if params.Location == "" {
        return "", errors.New("location is required")
    }
    
    // Validate units
    units := params.Units
    if units == "" {
        units = "fahrenheit"
    } else if units != "celsius" && units != "fahrenheit" {
        return "", fmt.Errorf("invalid units: %s (must be celsius or fahrenheit)", units)
    }
    
    // Implementation
}
```

## Conclusion

The step abstraction in Geppetto provides a powerful way to build compositional AI pipelines with features like streaming, cancellation, and tool calling. By understanding and leveraging this pattern, you can create sophisticated AI applications that handle complex workflows while maintaining type safety and composability.

The key advantages of using steps include:

1. **Compositional Design**: Build complex workflows from simple, reusable components
2. **Type Safety**: Leverage Go's type system for compile-time guarantees
3. **Cancellation Support**: Properly handle interruptions and cleanup resources
4. **Event Publishing**: Monitor progress and status in real-time
5. **Streaming Results**: Process results as they become available