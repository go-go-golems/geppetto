---
Title: Events, Streaming, and Watermill in Geppetto
Slug: geppetto-events-streaming-watermill
Short: Engine-first event publishing with EventSink, Watermill routing, and context-carried sinks for streaming AI.
Topics:
- geppetto
- architecture
- events
- watermill
- pubsub
- ai
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Events, Streaming, and Watermill in Geppetto

## Why Events?

When you run AI inference, you want to see results as they stream in — not wait for the entire response. This enables:

- **Responsive UIs** — show tokens as they arrive, not after 10 seconds of silence
- **Progress feedback** — show "Thinking..." or tool execution status
- **Debugging** — trace exactly what the model is doing and when
- **Real-time metrics** — display token counts as they accumulate

Geppetto's event system publishes structured events as inference progresses. Every token, tool call, and error becomes an event that flows through a unified pipeline.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Your Application                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────┐     ┌──────────────┐     ┌───────────────────┐  │
│   │  Engine  │────▶│  Event Sink  │────▶│  Watermill Router │  │
│   └──────────┘     └──────────────┘     └─────────┬─────────┘  │
│        │                                          │             │
│        ▼                                          ▼             │
│   ┌──────────┐                            ┌─────────────┐       │
│   │ Helpers/ │─────────────────────────▶ │  Handlers   │       │
│   │  Tools   │   (via context sinks)      │ (printers,  │       │
│   └──────────┘                            │  custom)    │       │
│                                           └─────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

**Key components:**

| Component | Role |
|-----------|------|
| **Engine** | Makes provider API calls, emits lifecycle events (start, partial, final) |
| **Event Sink** | Receives events and publishes to a topic |
| **Watermill Router** | Routes events to registered handlers |
| **Handlers** | Process events (print to console, aggregate, forward) |
| **Context Sinks** | Let helpers/tools emit events without explicit plumbing |

## Event Types

### Core Lifecycle Events

| Event | Type Constant | When Emitted | Key Fields |
|-------|---------------|--------------|------------|
| **start** | `EventTypeStart` | Inference begins | `Metadata` |
| **partial** | `EventTypePartialCompletion` | Each streamed chunk | `Delta`, `Completion` |
| **partial-thinking** | `EventTypePartialThinking` | Reasoning summary delta (Responses API) | `Delta` |
| **final** | `EventTypeFinal` | Inference completes | `Text`, `Metadata.Usage` |
| **interrupt** | `EventTypeInterrupt` | Context cancelled | `Text` (partial) |
| **error** | `EventTypeError` | Error occurs | `ErrorString` |

### Tool Events

| Event | Type Constant | When Emitted | Key Fields |
|-------|---------------|--------------|------------|
| **tool-call** | `EventTypeToolCall` | Model requests tool | `ToolCall.Name`, `ToolCall.Input`, `ToolCall.ID` |
| **tool-result** | `EventTypeToolResult` | Tool returns result | `ToolResult.ID`, `ToolResult.Result` |
| **tool-call-execute** | `EventTypeToolCallExecute` | Execution starts | `ToolCall` |
| **tool-call-execution-result** | `EventTypeToolCallExecutionResult` | Execution finishes | `ToolResult` |

### Extended Events

| Category | Events | Purpose |
|----------|--------|---------|
| **Reasoning** | `partial-thinking`, `reasoning-text-delta`, `reasoning-text-done` | o1/Claude thinking traces |
| **Web Search** | `web-search-started`, `web-search-searching`, `web-search-done` | Built-in web search progress |
| **File Search** | `file-search-started`, `file-search-done` | Built-in file search progress |
| **Code Interpreter** | `code-interpreter-*` | Code execution progress |
| **MCP** | `mcp-*`, `mcp-list-tools-*` | MCP tool progress |
| **Image Gen** | `image-generation-*` | Image generation progress |
| **Status** | `log`, `info`, `status`, `agent-mode-switch` | UI/debug status |

### Event Type Cheat Sheet

Common concrete Go types when parsing with `events.NewEventFromJson`:

- `*events.EventPartialCompletionStart` → stream start
- `*events.EventPartialCompletion` → `Delta`, `Completion`
- `*events.EventFinal` → `Text`
- `*events.EventToolCall` → `ToolCall` with `Name`, `Input`, `ID`
- `*events.EventToolResult` → `ToolResult` with `ID`, `Result`
- `*events.EventError` → `ErrorString`
- `*events.EventInterrupt` → `Text`

See full catalog: `geppetto/pkg/events/chat-events.go`

## Event Metadata

Every event carries `EventMetadata`:

```go
type EventMetadata struct {
    ID       uuid.UUID // Stable per stream
    SessionID   string // Correlation ID for the session (legacy name: run_id)
    InferenceID string // Correlation ID for the inference call
    TurnID   string    // Correlation ID for the turn
    Model    string    // Model identifier (e.g., "gpt-4")
    Duration time.Duration
    
    Usage    Usage     // Token counts (updated continuously)
    Extra    map[string]any // Provider-specific context
}

type Usage struct {
    InputTokens              int
    OutputTokens             int
    CachedTokens             int // OpenAI prompt cache
    CacheCreationInputTokens int // Claude cache
    CacheReadInputTokens     int // Claude cache
}
```

**Note**: `Usage` is updated as chunks arrive, so UIs can display evolving token counts in real-time.

## Publishing Events

Geppetto uses **context-carried sinks**:

- Provider engines publish via `events.PublishEventToContext(ctx, ...)`.
- Helper layers (tool loops, middleware) also publish via context.

Attach sinks to `context.Context` at runtime:

```go
watermillSink := middleware.NewWatermillSink(router.Publisher, "chat")
runCtx := events.WithEventSinks(ctx, watermillSink)

// Anywhere downstream can publish
events.PublishEventToContext(runCtx, events.NewToolResultEvent(meta, toolResult))
```

## Provider Event Flow

| Provider | Streaming Behavior |
|----------|-------------------|
| **OpenAI (Chat)** | `start` → multiple `partial` → `final`. Tool calls merged and emitted as `tool-call` when complete. |
| **OpenAI (Responses)** | Adds `info` events for reasoning boundaries, `partial-thinking` for summary deltas. Function args streamed via SSE. |
| **Claude** | Content-block merger emits `start` → `partial` → `tool-call` (when complete) → `final`. |

All providers publish via context sinks.

## Running the Event Router

Use a Watermill-backed `EventRouter` to route events to handlers:

```go
router, _ := events.NewEventRouter()
defer router.Close()

// Add a simple printer
router.AddHandler("chat", "chat", events.StepPrinterFunc("", os.Stdout))

// Create sink and engine
sink := middleware.NewWatermillSink(router.Publisher, "chat")
eng, _ := factory.NewEngineFromParsedLayers(parsed)

eg, groupCtx := errgroup.WithContext(ctx)

eg.Go(func() error { return router.Run(groupCtx) })
eg.Go(func() error {
    <-router.Running()
    runCtx := events.WithEventSinks(groupCtx, sink)
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Say hello."))
    _, err := eng.RunInference(runCtx, turn)
    return err
})

_ = eg.Wait()
```

## Client-side Consumption Patterns

### 1) Console Streaming Printer

```go
// Text streaming (writes deltas and final newline)
router.AddHandler("chat", "chat", events.StepPrinterFunc("", os.Stdout))

// Or structured output (text/json/yaml)
printer := events.NewStructuredPrinter(os.Stdout, events.PrinterOptions{
    Format:          events.FormatText, // or FormatJSON / FormatYAML
    IncludeMetadata: false,
    Full:            false,
})
router.AddHandler("chat", "chat", printer)
```

### 2) Custom Handler

```go
router.AddHandler("collector", "chat", func(msg *message.Message) error {
    defer msg.Ack()
    e, err := events.NewEventFromJson(msg.Payload)
    if err != nil { return err }

    switch ev := e.(type) {
    case *events.EventPartialCompletionStart:
        // initialize buffer / UI
    case *events.EventPartialCompletion:
        // append ev.Delta
    case *events.EventFinal:
        // flush buffer or update UI with ev.Text
    case *events.EventToolCall:
        // access via ev.ToolCall.Name and ev.ToolCall.Input (string)
    case *events.EventToolResult:
        // access via ev.ToolResult.ID and ev.ToolResult.Result (string)
    case *events.EventError:
        return fmt.Errorf(ev.ErrorString)
    }
    return nil
})
```

#### Pretty-printing tool payloads (JSON-aware)

`ToolCall.Input` and `ToolResult.Result` are strings. If they contain JSON, pretty-print them:

```go
func prettyJSONIfPossible(s string) string {
    t := strings.TrimSpace(s)
    if strings.HasPrefix(t, "{") || strings.HasPrefix(t, "[") {
        var v interface{}
        if err := json.Unmarshal([]byte(s), &v); err == nil {
            if b, err := json.MarshalIndent(v, "", "  "); err == nil {
                return string(b)
            }
        }
    }
    return s
}
```

### 3) Cross-process Consumption

For distributed systems, swap the in-process transport for NATS, Kafka, etc.:

```go
pub := /* e.g., NATS/Kafka publisher */
sub := /* e.g., NATS/Kafka subscriber */
router, _ := events.NewEventRouter(events.WithPublisher(pub), events.WithSubscriber(sub))

// Your handlers remain the same
router.AddHandler("chat", "chat", events.StepPrinterFunc("", os.Stdout))
```

## Custom Event Types

Geppetto provides an event registry for custom event types that flow through the same infrastructure.

### Registration Methods

#### 1. RegisterEventCodec (full control)

```go
import "github.com/go-go-golems/geppetto/pkg/events"

type CustomProgressEvent struct {
    events.EventImpl
    Progress float64 `json:"progress"`
    Status   string  `json:"status"`
}

func init() {
    decoder := func(b []byte) (events.Event, error) {
        var ev CustomProgressEvent
        if err := json.Unmarshal(b, &ev); err != nil {
            return nil, err
        }
        ev.SetPayload(b)
        return &ev, nil
    }
    
    _ = events.RegisterEventCodec("custom-progress", decoder)
}
```

#### 2. RegisterEventFactory (convenience)

```go
type CustomStatusEvent struct {
    events.EventImpl
    Phase string `json:"phase"`
}

func init() {
    factory := func() events.Event {
        return &CustomStatusEvent{
            EventImpl: events.EventImpl{Type_: "custom-status"},
        }
    }
    
    _ = events.RegisterEventFactory("custom-status", factory)
}
```

#### 3. RegisterEventEncoder (outbound serialization)

```go
func init() {
    encoder := func(ev events.Event) ([]byte, error) {
        return json.Marshal(ev)
    }
    
    _ = events.RegisterEventEncoder("custom-progress", encoder)
}
```

### Publishing Custom Events

```go
meta := events.EventMetadata{
    ID:     uuid.New(),
    SessionID:  "session-123",
    InferenceID: "inference-456",
    TurnID: "turn-456",
}

customEvent := &CustomProgressEvent{
    EventImpl: events.EventImpl{
        Type_:     "custom-progress",
        Metadata_: meta,
    },
    Progress: 0.75,
    Status:   "processing",
}

// Publish via context sinks
events.PublishEventToContext(ctx, customEvent)

// Or publish directly to a configured sink
_ = sink.PublishEvent(customEvent)
```

### Consuming Custom Events

```go
router.AddHandler("custom-collector", "chat", func(msg *message.Message) error {
    defer msg.Ack()
    
    e, err := events.NewEventFromJson(msg.Payload)
    if err != nil { return err }
    
    if progressEv, ok := e.(*CustomProgressEvent); ok {
        fmt.Printf("Progress: %.0f%% - %s\n", 
            progressEv.Progress*100, progressEv.Status)
    }
    
    return nil
})
```

### Best Practices for Custom Events

- **Embed EventImpl**: All custom events should embed `events.EventImpl` to satisfy the `Event` interface.
- **Register in init()**: Use `init()` functions to register at package initialization.
- **Unique type names**: Choose distinctive strings (e.g., `myapp-progress` not `progress`).
- **Metadata consistency**: Always populate `EventMetadata` with `ID`, optionally `SessionID`/`InferenceID`/`TurnID`.
- **Handle registration errors**: Duplicate registrations will fail.

### Use Cases for Custom Events

- **Tool-specific progress**: Database query progress, file upload status
- **Domain events**: "user-action", "workflow-step-completed"
- **Integration events**: Events from external systems
- **Debugging events**: Instrumentation during development

## Full Example: Router + Engine + Helpers

The `router.Run(ctx)` method blocks until cancelled. Use `errgroup` to coordinate:

```go
ctx, cancel := context.WithCancel(context.Background())
// NOTE: No defer cancel() here, errgroup handles it via groupCtx

eg, groupCtx := errgroup.WithContext(ctx)
router, _ := events.NewEventRouter()

// Goroutine for the router
eg.Go(func() error {
    defer func() {
        log.Info().Msg("Closing event router")
        _ = router.Close()
        log.Info().Msg("Event router closed")
    }()

    log.Info().Msg("Starting event router")
    runErr := router.Run(groupCtx)
    log.Info().Err(runErr).Msg("Event router stopped")
    // Don't return context.Canceled as a fatal error
    if runErr != nil && !errors.Is(runErr, context.Canceled) {
        return runErr
    }
    return nil
})

// Goroutine for the main task
eg.Go(func() error {
    defer cancel() // Signal router to stop when done

    <-router.Running()
    log.Info().Msg("Event router is running, proceeding with main task")

    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Say hello."))
    _, err := eng.RunInference(events.WithEventSinks(groupCtx, watermillSink), turn)
    if err != nil {
        return err
    }
    
    log.Info().Msg("Main task finished")
    return nil 
})

log.Info().Msg("Waiting for goroutines to finish")
if err := eg.Wait(); err != nil {
    log.Error().Err(err).Msg("Application finished with error")
} else {
    log.Info().Msg("Application finished successfully")
}
```

**Why this pattern?**
- Router is confirmed running before publishing starts
- Router stays alive while the task runs
- If task finishes/fails, router shuts down via context cancellation
- If router fails, task stops via context cancellation
- `router.Close()` is called reliably

## Structured Data Extraction with Filtering Sinks

Geppetto ships a filtering sink that extracts structured payloads from streaming text and emits typed events.

### Tag Format

Blocks are delimited with XML-like tags:

~~~text
<geppetto:citations:v1>
```yaml
citations:
  - title: GPT-4 Technical Report
    authors: [OpenAI]
```
</geppetto:citations:v1>
~~~

### Extractor Interface

Each extractor declares the `(package, type, version)` tuple it handles:

```go
type Extractor interface {
    TagPackage() string
    TagType() string
    TagVersion() string
    NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession
}

type ExtractorSession interface {
    OnStart(ctx context.Context) []events.Event
    OnRaw(ctx context.Context, chunk []byte) []events.Event
    OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event
}
```

### Sink Setup

```go
next := /* your downstream events.EventSink */
sink := structuredsink.NewFilteringSink(next, structuredsink.Options{
    Malformed: structuredsink.MalformedErrorEvents,
}, &citationsExtractor{})
```

### Parsing Helpers

The `parsehelpers` package includes a debounced YAML parser for progressive updates:

```go
ctrl := parsehelpers.NewDebouncedYAML[citationsPayload](parsehelpers.DebounceConfig{
    SnapshotEveryBytes: 512,
    SnapshotOnNewline:  true,
    MaxBytes:           64 << 10,
})
```

Use `OnRaw` to feed bytes and emit "best-so-far" events, then `OnCompleted` to parse the final payload. If a block is malformed, the sink applies the configured policy (`MalformedErrorEvents`, `MalformedReconstructText`, or `MalformedIgnore`).

## Practical Tips

- Always wait for `router.Running()` before invoking inference to avoid dropped events.
- Attach the same sink to both the engine and the context so helpers/tools can publish.
- Prefer `events.NewStructuredPrinter` for machine-readable output during tests.
- Register custom event types in `init()` functions.
- Enable debug logging (`zerolog.DebugLevel`) to see event publishing.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| No events received | Router not running | Call `<-router.Running()` before inference |
| Missing tool events | Sink not on context | Use `events.WithEventSinks(ctx, sink)` |
| Dropped events | Wrong topic | Match topic in `NewWatermillSink` and `AddHandler` |
| Events stop mid-stream | Context cancelled | Check for deadline or explicit cancellation |

## Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/events"               // Core events, router, printers
    "github.com/go-go-golems/geppetto/pkg/inference/middleware" // WatermillSink
    "github.com/go-go-golems/geppetto/pkg/events/structuredsink" // Filtering + extraction
)
```

## See Also

- [Inference Engines](06-inference-engines.md) — How engines emit events
- [Tools](07-tools.md) — Tool events and execution
- [Streaming Tutorial](../tutorials/01-streaming-inference-with-tools.md) — Complete example
- Example: `geppetto/cmd/examples/simple-streaming-inference/main.go`
