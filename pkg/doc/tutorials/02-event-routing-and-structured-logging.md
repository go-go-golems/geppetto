---
Title: Event Routing and Structured Logging
Slug: geppetto-tutorial-event-routing-logging
Short: Build an application that routes inference events to multiple handlers and produces structured JSON/YAML logs.
Topics:
- geppetto
- tutorial
- events
- logging
- watermill
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Event Routing and Structured Logging

This tutorial teaches you how to build an application that routes inference events to multiple handlers simultaneously — one for real-time console output and another for structured JSON logging suitable for log aggregation systems.

## What You'll Build

A CLI application that:
- Streams assistant responses to the console in real-time
- Logs all events as structured JSON to a file
- Aggregates token usage statistics
- Handles errors gracefully

## Prerequisites

- Basic Go and Cobra knowledge
- Understanding of [Events](../topics/04-events.md)
- A configured provider (OpenAI, Claude, etc.)

## Learning Objectives

- Understand how Watermill routes events to multiple handlers
- Learn the different printer formats (text, JSON, YAML)
- Build custom aggregating handlers
- Separate concerns between UI and logging

## Architecture

```
                    ┌─────────────────────┐
                    │      Engine         │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │    WatermillSink    │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │   EventRouter       │
                    └──────────┬──────────┘
                               │
           ┌───────────────────┼───────────────────┐
           ▼                   ▼                   ▼
    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
    │ Text Printer│    │ JSON Logger │    │ Aggregator  │
    │  (stdout)   │    │  (file)     │    │ (metrics)   │
    └─────────────┘    └─────────────┘    └─────────────┘
```

## Step 1: Set Up the Application Structure

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "sync/atomic"
    "time"

    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/turns"
    "golang.org/x/sync/errgroup"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func run() error {
    ctx := context.Background()
    // ... implementation follows
}
```

## Step 2: Create the Event Router

```go
func run() error {
    ctx := context.Background()

    // Create the event router
    router, err := events.NewEventRouter()
    if err != nil {
        return fmt.Errorf("failed to create router: %w", err)
    }
    defer router.Close()

    // ... add handlers next
}
```

## Step 3: Add the Console Text Printer

The built-in `StepPrinterFunc` streams text deltas to stdout:

```go
    // Handler 1: Real-time text streaming to console
    router.AddHandler(
        "console-printer",    // Unique handler name
        "chat",               // Topic to subscribe to
        events.StepPrinterFunc("", os.Stdout),
    )
```

**What it does:**
- Writes each `partial` event's delta directly to stdout
- Prints a newline after `final` events
- Ignores non-text events (tool calls, errors are not printed)

## Step 4: Add the Structured JSON Logger

Use `NewStructuredPrinter` for machine-readable output:

```go
    // Handler 2: Structured JSON logging to file
    logFile, err := os.Create("inference.log")
    if err != nil {
        return fmt.Errorf("failed to create log file: %w", err)
    }
    defer logFile.Close()

    jsonPrinter := events.NewStructuredPrinter(logFile, events.PrinterOptions{
        Format:          events.FormatJSON,  // JSON lines format
        IncludeMetadata: true,               // Include timing, tokens, etc.
        Full:            true,               // Include all event types
    })

    router.AddHandler(
        "json-logger",
        "chat",
        jsonPrinter,
    )
```

**Printer formats:**
- `events.FormatJSON` — One JSON object per line (JSONL)
- `events.FormatYAML` — YAML documents separated by `---`
- `events.FormatText` — Human-readable text

**Options:**
- `IncludeMetadata: true` — Add `metadata` field with timing, usage, etc.
- `Full: true` — Log all events, not just text
- `Prefix: ">>> "` — Prepend prefix to each line

## Step 5: Add a Custom Aggregator Handler

Build a handler that collects statistics:

```go
    // Handler 3: Metrics aggregator
    var stats struct {
        EventCount   int64
        InputTokens  int64
        OutputTokens int64
        ToolCalls    int64
        Errors       int64
        StartTime    time.Time
    }
    stats.StartTime = time.Now()

    router.AddHandler("aggregator", "chat", func(msg *message.Message) error {
        defer msg.Ack()

        atomic.AddInt64(&stats.EventCount, 1)

        ev, err := events.NewEventFromJson(msg.Payload)
        if err != nil {
            return nil // Skip malformed events
        }

        switch e := ev.(type) {
        case *events.EventFinal:
            atomic.AddInt64(&stats.InputTokens, int64(e.Metadata().Usage.InputTokens))
            atomic.AddInt64(&stats.OutputTokens, int64(e.Metadata().Usage.OutputTokens))

        case *events.EventToolCall:
            atomic.AddInt64(&stats.ToolCalls, 1)

        case *events.EventError:
            atomic.AddInt64(&stats.Errors, 1)
        }

        return nil
    })
```

## Step 6: Create the Sink and Engine

```go
    // Create sink that publishes to the "chat" topic
    sink := middleware.NewWatermillSink(router.Publisher, "chat")

    // Create engine (sinks are attached at runtime via context)
    eng, err := factory.NewEngineFromParsedLayers(parsedLayers)
    if err != nil {
        return fmt.Errorf("failed to create engine: %w", err)
    }
```

## Step 7: Build the Turn and Run Inference

```go
    // Build the Turn
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewSystemTextBlock("You are a helpful assistant."))
    turns.AppendBlock(turn, turns.NewUserTextBlock("Explain event-driven architecture in 3 sentences."))

    // Run router and inference concurrently
    eg, groupCtx := errgroup.WithContext(ctx)

    eg.Go(func() error {
        return router.Run(groupCtx)
    })

    eg.Go(func() error {
        <-router.Running() // Wait for router to be ready

        // Attach sink to context for any helpers that publish
        runCtx := events.WithEventSinks(groupCtx, sink)

        _, err := eng.RunInference(runCtx, turn)
        return err
    })

    if err := eg.Wait(); err != nil {
        return fmt.Errorf("inference failed: %w", err)
    }

    // Print aggregated stats
    duration := time.Since(stats.StartTime)
    fmt.Printf("\n\n--- Statistics ---\n")
    fmt.Printf("Duration: %v\n", duration)
    fmt.Printf("Events: %d\n", stats.EventCount)
    fmt.Printf("Tokens: %d in, %d out\n", stats.InputTokens, stats.OutputTokens)
    fmt.Printf("Tool calls: %d\n", stats.ToolCalls)
    fmt.Printf("Errors: %d\n", stats.Errors)

    return nil
}
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    "sync/atomic"
    "time"

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

    // 2. Console printer (real-time streaming)
    router.AddHandler("console", "chat", events.StepPrinterFunc("", os.Stdout))

    // 3. JSON logger (structured logs)
    logFile, _ := os.Create("inference.log")
    defer logFile.Close()
    jsonPrinter := events.NewStructuredPrinter(logFile, events.PrinterOptions{
        Format:          events.FormatJSON,
        IncludeMetadata: true,
        Full:            true,
    })
    router.AddHandler("json-logger", "chat", jsonPrinter)

    // 4. Metrics aggregator
    var totalTokens int64
    router.AddHandler("metrics", "chat", func(msg *message.Message) error {
        defer msg.Ack()
        ev, _ := events.NewEventFromJson(msg.Payload)
        if final, ok := ev.(*events.EventFinal); ok {
            atomic.AddInt64(&totalTokens, int64(
                final.Metadata().Usage.InputTokens+final.Metadata().Usage.OutputTokens,
            ))
        }
        return nil
    })

    // 5. Create sink and engine
    sink := middleware.NewWatermillSink(router.Publisher, "chat")
    eng, _ := factory.NewEngineFromParsedLayers(parsedLayers)

    // 6. Build Turn
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Hello!"))

    // 7. Run
    eg, groupCtx := errgroup.WithContext(ctx)
    eg.Go(func() error { return router.Run(groupCtx) })
    eg.Go(func() error {
        <-router.Running()
        _, err := eng.RunInference(events.WithEventSinks(groupCtx, sink), turn)
        return err
    })
    _ = eg.Wait()

    fmt.Printf("\n\nTotal tokens: %d\n", totalTokens)
}
```

## Sample Output

**Console (stdout):**
```
Hello! I'm here to help. What would you like to know?
```

**Log file (inference.log):**
```json
{"type":"start","metadata":{"id":"abc123","model":"gpt-4"}}
{"type":"partial","delta":"Hello","completion":"Hello","metadata":{...}}
{"type":"partial","delta":"!","completion":"Hello!","metadata":{...}}
{"type":"final","text":"Hello! I'm here to help...","metadata":{"usage":{"input_tokens":12,"output_tokens":15}}}
```

## Advanced: Multiple Topics

Route different event types to different handlers:

```go
// Tool events go to a separate topic
toolSink := middleware.NewWatermillSink(router.Publisher, "tools")

// Add tool-specific handler
router.AddHandler("tool-logger", "tools", func(msg *message.Message) error {
    defer msg.Ack()
    // Log tool calls to separate file
    return nil
})
```

## Advanced: External Transports

For distributed systems, use external Watermill transports:

```go
import "github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"

// Create Redis publisher/subscriber
pub, _ := redisstream.NewPublisher(redisstream.PublisherConfig{...}, logger)
sub, _ := redisstream.NewSubscriber(redisstream.SubscriberConfig{...}, logger)

// Pass to router
router, _ := events.NewEventRouter(
    events.WithPublisher(pub),
    events.WithSubscriber(sub),
)
```

Now events flow through Redis and can be consumed by any connected service.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| No console output | Handler not registered | Check `AddHandler` called before `Run` |
| Missing log entries | File not flushed | Ensure `defer logFile.Close()` |
| Duplicate events | Same handler name | Use unique names per handler |
| Events out of order | Async handlers | Events arrive in order per handler |

## See Also

- [Events](../topics/04-events.md) — Full events reference
- [Streaming Tutorial](01-streaming-inference-with-tools.md) — Basic streaming
- [Add Event Handler Playbook](../playbooks/02-add-a-new-event-handler.md) — Step-by-step guide
- Example: `geppetto/cmd/examples/simple-streaming-inference/main.go`
