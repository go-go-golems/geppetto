---
Title: Add a New Event Handler
Slug: geppetto-playbook-add-event-handler
Short: Step-by-step guide to implement a custom event handler, subscribe it to the router, and parse incoming events.
Topics:
- geppetto
- events
- playbook
- watermill
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Add a New Event Handler

This playbook walks through adding a custom event handler to your Geppetto application. By the end, your handler will receive streaming events from inference and process them as needed (logging, aggregation, forwarding, etc.).

## Prerequisites

- A working Geppetto engine setup (see [Inference Engines](../topics/06-inference-engines.md))
- Understanding of the event system (see [Events](../topics/04-events.md))

## Steps

### Step 1: Create the Event Router

The router transports events via Watermill. Create it early in your application:

```go
import "github.com/go-go-golems/geppetto/pkg/events"

router, err := events.NewEventRouter()
if err != nil {
    return fmt.Errorf("failed to create router: %w", err)
}
defer router.Close()
```

**Options:**
- `events.WithVerbose(true)` — Enable Watermill debug logging
- `events.WithPublisher(pub)` / `events.WithSubscriber(sub)` — Use external transport (NATS, Redis, Kafka)

### Step 2: Implement Your Handler Function

A handler receives a `*message.Message` from Watermill and must call `msg.Ack()` when done:

```go
import (
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/go-go-golems/geppetto/pkg/events"
)

func myCustomHandler(msg *message.Message) error {
    defer msg.Ack() // Always acknowledge the message
    
    // Parse the raw JSON into a typed event
    ev, err := events.NewEventFromJson(msg.Payload)
    if err != nil {
        return fmt.Errorf("failed to parse event: %w", err)
    }
    
    // Handle specific event types
    switch e := ev.(type) {
    case *events.EventPartialCompletionStart:
        fmt.Println("Stream started")
        
    case *events.EventPartialCompletion:
        fmt.Printf("Delta: %s\n", e.Delta)
        
    case *events.EventFinal:
        fmt.Printf("Final text: %s\n", e.Text)
        fmt.Printf("Tokens: %d input, %d output\n", 
            e.Metadata().Usage.InputTokens, 
            e.Metadata().Usage.OutputTokens)
        
    case *events.EventToolCall:
        fmt.Printf("Tool call: %s(%s)\n", e.ToolCall.Name, e.ToolCall.Input)
        
    case *events.EventToolResult:
        fmt.Printf("Tool result [%s]: %s\n", e.ToolResult.ID, e.ToolResult.Result)
        
    case *events.EventError:
        return fmt.Errorf("inference error: %s", e.ErrorString)
    }
    
    return nil
}
```

### Step 3: Register the Handler

Add your handler to the router with a unique name and topic:

```go
router.AddHandler(
    "my-custom-handler",  // Handler name (unique within router)
    "chat",               // Topic to subscribe to
    myCustomHandler,      // Your handler function
)
```

**Important:** The topic must match the one used when creating the sink (Step 5).

### Step 4: Create the Event Sink

The sink connects your engine to the router:

```go
import "github.com/go-go-golems/geppetto/pkg/inference/middleware"

sink := middleware.NewWatermillSink(router.Publisher, "chat")
```

### Step 5: Wire the Engine with the Sink

Pass the sink when creating the engine:

```go
import "github.com/go-go-golems/geppetto/pkg/inference/engine"

eng, err := factory.NewEngineFromParsedLayers(parsedLayers, engine.WithSink(sink))
if err != nil {
    return err
}
```

### Step 6: Run Router and Inference Concurrently

Use `errgroup` to coordinate the router and inference:

```go
import "golang.org/x/sync/errgroup"

eg, groupCtx := errgroup.WithContext(ctx)

// Start the router
eg.Go(func() error {
    return router.Run(groupCtx)
})

// Run inference after router is ready
eg.Go(func() error {
    <-router.Running() // Wait for router to be ready
    
    // Attach sink to context for helpers/tools to publish
    runCtx := events.WithEventSinks(groupCtx, sink)
    
    // Run your inference
    _, err := eng.RunInference(runCtx, turn)
    return err
})

if err := eg.Wait(); err != nil {
    return err
}
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/turns"
    "golang.org/x/sync/errgroup"
)

func main() {
    ctx := context.Background()
    
    // 1. Create router
    router, _ := events.NewEventRouter()
    defer router.Close()
    
    // 2. Add built-in text printer
    router.AddHandler("printer", "chat", events.StepPrinterFunc("", os.Stdout))
    
    // 3. Add custom aggregator
    var tokenCount int
    router.AddHandler("aggregator", "chat", func(msg *message.Message) error {
        defer msg.Ack()
        ev, _ := events.NewEventFromJson(msg.Payload)
        if final, ok := ev.(*events.EventFinal); ok {
            tokenCount = final.Metadata().Usage.InputTokens + final.Metadata().Usage.OutputTokens
        }
        return nil
    })
    
    // 4. Create sink and engine
    sink := middleware.NewWatermillSink(router.Publisher, "chat")
    eng, _ := factory.NewEngineFromParsedLayers(parsedLayers, engine.WithSink(sink))
    
    // 5. Build Turn
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Hello!"))
    
    // 6. Run concurrently
    eg, groupCtx := errgroup.WithContext(ctx)
    eg.Go(func() error { return router.Run(groupCtx) })
    eg.Go(func() error {
        <-router.Running()
        _, err := eng.RunInference(events.WithEventSinks(groupCtx, sink), turn)
        return err
    })
    
    _ = eg.Wait()
    fmt.Printf("\nTotal tokens: %d\n", tokenCount)
}
```

## Advanced: Custom Event Types

Register custom event types to flow through the same infrastructure:

```go
import "github.com/go-go-golems/geppetto/pkg/events"

// Define custom event
type MyProgressEvent struct {
    events.EventImpl
    Progress float64 `json:"progress"`
    Message  string  `json:"message"`
}

// Register decoder in init()
func init() {
    _ = events.RegisterEventFactory("my-progress", func() events.Event {
        return &MyProgressEvent{EventImpl: events.EventImpl{Type_: "my-progress"}}
    })
}

// Publish from your code
func publishProgress(ctx context.Context, progress float64, msg string) {
    ev := &MyProgressEvent{
        EventImpl: events.EventImpl{
            Type_: "my-progress",
            Metadata_: events.EventMetadata{ID: uuid.New()},
        },
        Progress: progress,
        Message:  msg,
    }
    events.PublishEventToContext(ctx, ev)
}

// Handle in your handler
func myHandler(msg *message.Message) error {
    defer msg.Ack()
    ev, _ := events.NewEventFromJson(msg.Payload)
    if progress, ok := ev.(*MyProgressEvent); ok {
        fmt.Printf("Progress: %.0f%% - %s\n", progress.Progress*100, progress.Message)
    }
    return nil
}
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| No events received | Router not running | Wait for `<-router.Running()` before inference |
| Events dropped | Topic mismatch | Ensure sink and handler use same topic |
| Duplicate events | Multiple handlers same name | Use unique handler names |
| Handler not called | Handler added after Run | Add handlers before calling `router.Run()` |
| Context cancelled | Router stopped early | Check errgroup errors |

## Built-in Handlers

Geppetto provides ready-to-use handlers:

```go
// Text streaming to stdout
router.AddHandler("chat", "chat", events.StepPrinterFunc("", os.Stdout))

// Structured output (JSON/YAML/text)
printer := events.NewStructuredPrinter(os.Stdout, events.PrinterOptions{
    Format:          events.FormatJSON,
    IncludeMetadata: true,
})
router.AddHandler("structured", "chat", printer)

// Raw event dumping (debugging)
router.AddHandler("debug", "chat", router.DumpRawEvents)
```

## See Also

- [Events](../topics/04-events.md) — Full events reference
- [Streaming Tutorial](../tutorials/01-streaming-inference-with-tools.md) — Complete streaming example
- Example: `geppetto/cmd/examples/simple-streaming-inference/main.go`

