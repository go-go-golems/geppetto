# Streaming Simple Inference with Sink - Implementation Guide

This document collects all the information necessary to write a streaming simple inference using a Sink, based on the existing codebase patterns in `geppetto/cmd/
examples/simple-inference/main.go` and `pinocchio/pkg/cmds/cmd.go`.

## 1. Core Components Overview

### 1.1 EventSink Interface

The `EventSink` interface is the core abstraction that enables engines to publish events without knowing how they will be processed or distributed. This interface provides a clean separation between event generation and event handling, allowing for flexible event routing and processing.

**File**: `geppetto/pkg/events/sink.go`
```go
type EventSink interface {
    PublishEvent(event Event) error
}
```

### 1.2 WatermillSink Implementation

The `WatermillSink` is the primary implementation of the `EventSink` interface that integrates with the Watermill message bus library. This sink serializes events to JSON and publishes them to specific topics, enabling reliable asynchronous event distribution with support for multiple subscribers.

**File**: `geppetto/pkg/inference/middleware/sink_watermill.go`
```go
type WatermillSink struct {
    publisher message.Publisher
    topic     string
}

func NewWatermillSink(publisher message.Publisher, topic string) *WatermillSink
func (w *WatermillSink) PublishEvent(event events.Event) error
```

### 1.3 Engine Options

The engine options system provides a functional programming approach to configuring inference engines with sinks and other capabilities. This design allows engines to be configured with multiple sinks, enabling events to be published to multiple destinations simultaneously.

**File**: `geppetto/pkg/inference/engine/options.go`
```go
type Option func(*Config) error
type Config struct {
    EventSinks []events.EventSink
}

func WithSink(sink events.EventSink) Option
func ApplyOptions(config *Config, options ...Option) error
```

### 1.4 EventRouter

The `EventRouter` is the central coordinator that manages the Watermill message bus and routes events from publishers to subscribers. It provides a high-level interface for setting up event handlers and managing the lifecycle of the message routing system.

**File**: `geppetto/pkg/events/event-router.go`
```go
type EventRouter struct {
    logger     watermill.LoggerAdapter
    Publisher  message.Publisher
    Subscriber message.Subscriber
    router     *message.Router
    verbose    bool
}

func NewEventRouter(options ...EventRouterOption) (*EventRouter, error)
func (e *EventRouter) AddHandler(name string, topic string, f func(msg *message.Message) error)
func (e *EventRouter) Run(ctx context.Context) error
func (e *EventRouter) Close() error
```

## 2. Event Printer Functions

Event printer functions are the final consumers in the event flow, responsible for converting events into human-readable output. These functions handle the display logic for different event types and output formats, providing flexibility in how streaming results are presented to users.

### 2.1 StepPrinterFunc

The `StepPrinterFunc` provides a simple text-based printer that handles the most common event types for streaming inference. It's designed for basic use cases where you want to display streaming text with minimal formatting.

**File**: `geppetto/pkg/events/step-printer-func.go`
```go
func StepPrinterFunc(name string, w io.Writer) func(msg *message.Message) error
```
- Handles partial completion events by printing delta text
- Handles final events with newlines
- Handles tool calls and results with YAML formatting

### 2.2 NewStructuredPrinter

The `NewStructuredPrinter` provides advanced formatting capabilities for events, supporting multiple output formats including JSON and YAML. This printer is useful for debugging, logging, or when structured output is required for further processing.

**File**: `geppetto/pkg/events/printer.go`
```go
type PrinterFormat string
const (
    FormatText PrinterFormat = "text"
    FormatJSON PrinterFormat = "json"
    FormatYAML PrinterFormat = "yaml"
)

type PrinterOptions struct {
    Format          PrinterFormat
    Name            string
    IncludeMetadata bool
    Full            bool
}

func NewStructuredPrinter(w io.Writer, options PrinterOptions) func(msg *message.Message) error
```

## 3. Engine Factory

The engine factory system provides a unified interface for creating inference engines with different AI providers. This abstraction allows applications to switch between different AI providers (OpenAI, Claude, Gemini) without changing the core inference logic, while supporting the addition of sinks and other configuration options.

### 3.1 StandardEngineFactory

The `StandardEngineFactory` is the default implementation that supports multiple AI providers and integrates seamlessly with the sink system. It automatically determines the appropriate engine type based on configuration and applies sink options to enable event streaming.

**File**: `geppetto/pkg/inference/engine/factory/factory.go`
```go
type EngineFactory interface {
    CreateEngine(settings *settings.StepSettings, options ...engine.Option) (engine.Engine, error)
    SupportedProviders() []string
    DefaultProvider() string
}

type StandardEngineFactory struct {
    ClaudeTools []api.Tool
}

func NewStandardEngineFactory(claudeTools ...api.Tool) *StandardEngineFactory
func (f *StandardEngineFactory) CreateEngine(settings *settings.StepSettings, options ...engine.Option) (engine.Engine, error)
```

### 3.2 Factory Helper Functions

The factory helper functions provide convenient shortcuts for common engine creation patterns, reducing boilerplate code and simplifying the integration of engines with sinks and other options.

**File**: `geppetto/pkg/inference/engine/factory/helpers.go`
```go
func NewEngineFromStepSettings(stepSettings *settings.StepSettings, options ...engine.Option) (engine.Engine, error)
func NewEngineFromParsedLayers(parsedLayers *layers.ParsedLayers, options ...engine.Option) (engine.Engine, error)
```

## 4. Implementation Patterns from Existing Code

The codebase provides two main patterns for inference: a simple blocking pattern and a streaming pattern with events. Understanding these patterns helps developers choose the appropriate approach for their use case and provides concrete examples of how to implement streaming inference.

### 4.1 Simple Inference Pattern (Current)

The current simple inference pattern provides a straightforward blocking approach without streaming events. This pattern is useful for simple use cases where real-time streaming is not required and you just need the final result.

**File**: `geppetto/cmd/examples/simple-inference/main.go`
```go
// Current pattern without streaming
engine, err := factory.NewEngineFromParsedLayers(parsedLayers)
if err != nil {
    return errors.Wrap(err, "failed to create engine")
}

msg, err := engine.RunInference(ctx, conversation_)
if err != nil {
    return fmt.Errorf("inference failed: %w", err)
}
```

### 4.2 Pinocchio Command Pattern (With Streaming)

The Pinocchio command pattern demonstrates the full streaming implementation with event routing, multiple sinks, and coordinated execution using errgroup. This pattern shows how to integrate streaming events with proper cleanup and error handling.

**File**: `pinocchio/pkg/cmds/cmd.go`
```go
// Pattern with streaming events
if rc.Router != nil {
    watermillSink := middleware.NewWatermillSink(rc.Router.Publisher, "chat")
    options = append(options, engine.WithSink(watermillSink))

    // Add default printer
    if rc.UISettings == nil || rc.UISettings.Output == "" {
        rc.Router.AddHandler("chat", "chat", events.StepPrinterFunc("", rc.Writer))
    } else {
        printer := events.NewStructuredPrinter(rc.Writer, events.PrinterOptions{
            Format:          events.PrinterFormat(rc.UISettings.Output),
            Name:            "",
            IncludeMetadata: rc.UISettings.WithMetadata,
            Full:            rc.UISettings.FullOutput,
        })
        rc.Router.AddHandler("chat", "chat", printer)
    }

    // Start router in goroutine
    eg := errgroup.Group{}
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    eg.Go(func() error {
        defer cancel()
        defer func(Router *events.EventRouter) {
            _ = Router.Close()
        }(rc.Router)
        return rc.Router.Run(ctx)
    })

    eg.Go(func() error {
        defer cancel()
        <-rc.Router.Running()
        return g.runEngineAndCollectMessages(ctx, rc, options)
    })

    err := eg.Wait()
    if err != nil {
        return nil, err
    }
}
```

## 5. Required Dependencies

The streaming inference system requires several key dependencies and imports to function properly. Understanding these dependencies helps developers set up their projects correctly and avoid common integration issues.

### 5.1 Import Statements

The following import statements provide access to all the necessary components for implementing streaming inference with sinks.

```go
import (
    "context"
    "fmt"
    "io"
    "os"
    
    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/conversation/builder"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/glazed/pkg/cmds/layers"
    "github.com/pkg/errors"
    "github.com/rs/zerolog/log"
    "golang.org/x/sync/errgroup"
)
```

### 5.2 Key Functions to Use

These are the essential functions that developers need to understand and use when implementing streaming inference with sinks.

- `events.NewEventRouter()` - Create event router
- `middleware.NewWatermillSink(router.Publisher, "chat")` - Create watermill sink
- `engine.WithSink(watermillSink)` - Add sink to engine options
- `router.AddHandler("chat", "chat", events.StepPrinterFunc("", w))` - Add printer handler
- `router.Run(ctx)` - Start router
- `factory.NewEngineFromParsedLayers(parsedLayers, options...)` - Create engine with options

## 6. Implementation Steps

The implementation of streaming inference follows a specific sequence of steps that ensure proper initialization, event routing, and cleanup. These steps provide a template for creating robust streaming inference applications.

### 6.1 Basic Streaming Inference Structure

This basic structure demonstrates the minimal setup required for streaming inference with proper error handling and resource cleanup.

```go
func runStreamingInference(ctx context.Context, prompt string, w io.Writer) error {
    // 1. Create event router
    router, err := events.NewEventRouter()
    if err != nil {
        return errors.Wrap(err, "failed to create event router")
    }
    defer router.Close()

    // 2. Create watermill sink
    watermillSink := middleware.NewWatermillSink(router.Publisher, "chat")
    
    // 3. Add printer handler
    router.AddHandler("chat", "chat", events.StepPrinterFunc("", w))

    // 4. Create engine with sink
    engine, err := factory.NewEngineFromParsedLayers(parsedLayers, 
        engine.WithSink(watermillSink))
    if err != nil {
        return errors.Wrap(err, "failed to create engine")
    }

    // 5. Start router and run inference in parallel
    eg := errgroup.Group{}
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    eg.Go(func() error {
        defer cancel()
        return router.Run(ctx)
    })

    eg.Go(func() error {
        defer cancel()
        <-router.Running()
        
        // Run inference
        msg, err := engine.RunInference(ctx, conversation_)
        if err != nil {
            return fmt.Errorf("inference failed: %w", err)
        }
        
        // Append result
        if err := manager.AppendMessages(msg); err != nil {
            return fmt.Errorf("failed to append message: %w", err)
        }
        
        return nil
    })

    return eg.Wait()
}
```

### 6.2 Advanced Configuration Options

Advanced configuration options demonstrate how to customize the streaming behavior for different use cases, including structured output, custom topics, and multiple sinks.

```go
// With structured output
printer := events.NewStructuredPrinter(w, events.PrinterOptions{
    Format:          events.FormatJSON,
    IncludeMetadata: true,
    Full:            false,
})
router.AddHandler("chat", "chat", printer)

// With custom topic
customSink := middleware.NewWatermillSink(router.Publisher, "custom-topic")
router.AddHandler("custom-topic", "custom-topic", events.StepPrinterFunc("", w))

// With multiple sinks
sink1 := middleware.NewWatermillSink(router.Publisher, "chat")
sink2 := middleware.NewWatermillSink(router.Publisher, "logging")
engine, err := factory.NewEngineFromParsedLayers(parsedLayers,
    engine.WithSink(sink1),
    engine.WithSink(sink2))
```

## 7. Event Types and Flow

Understanding the event types and flow is crucial for implementing custom handlers and debugging streaming inference. The event system provides a rich set of event types that capture different aspects of the inference process.

### 7.1 Event Flow

The event flow describes how events move through the system from generation to consumption, providing a clear understanding of the streaming architecture.

1. **Engine** runs inference and generates events
2. **EventSink** (WatermillSink) publishes events to watermill topic
3. **EventRouter** routes events to registered handlers
4. **Handlers** (StepPrinterFunc/StructuredPrinter) process and display events

### 7.2 Event Types

The event system supports various event types that capture different aspects of the inference process, from streaming text to tool calls and errors.

- `EventPartialCompletion` - Streaming text chunks
- `EventFinal` - Final completion
- `EventError` - Error events
- `EventToolCall` - Tool call events
- `EventToolResult` - Tool result events

## 8. Error Handling and Cleanup

Proper error handling and cleanup are essential for robust streaming inference applications. The event-driven nature of the system requires careful management of resources and graceful shutdown procedures.

### 8.1 Proper Cleanup Pattern

This pattern ensures that resources are properly cleaned up even in error conditions, preventing resource leaks and ensuring reliable application behavior.

```go
defer func() {
    if router != nil {
        _ = router.Close()
    }
}()
```

### 8.2 Context Cancellation

Context cancellation provides a coordinated way to shut down the streaming system gracefully, ensuring that all goroutines are properly terminated.

```go
ctx, cancel := context.WithCancel(ctx)
defer cancel()

// Use errgroup for coordinated shutdown
eg := errgroup.Group{}
eg.Go(func() error {
    defer cancel()
    return router.Run(ctx)
})
```

## 9. Testing and Debugging

Testing and debugging streaming inference requires special considerations due to the asynchronous nature of events. The system provides several tools and patterns for effective debugging and testing.

### 9.1 Debug Mode

Debug mode enables verbose logging and provides detailed information about event flow, helping developers understand and troubleshoot streaming behavior.

```go
// Enable verbose logging
router, err := events.NewEventRouter(events.WithVerbose(true))
```

### 9.2 Raw Event Dumping

Raw event dumping allows developers to see the exact events being generated and processed, providing deep insights into the streaming behavior.

```go
// Add raw event dumper for debugging
router.AddHandler("chat", "chat", router.DumpRawEvents)
```

## 10. Integration with Existing Simple Inference

The integration guide provides a step-by-step approach for modifying existing simple inference examples to support streaming, ensuring backward compatibility while adding streaming capabilities.

To modify the existing simple inference example to support streaming:

1. **Add router creation** after settings initialization
2. **Create watermill sink** with router publisher
3. **Add engine.WithSink()** to engine creation options
4. **Add printer handler** to router
5. **Wrap engine.RunInference()** in errgroup with router.Run()
6. **Handle cleanup** with proper defer statements

This pattern allows the simple inference example to stream events while maintaining the same interface and functionality. 