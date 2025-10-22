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

Geppetto uses an engine-first architecture. Engines handle provider API calls and emit streaming events. Event delivery is decoupled via sinks (publishers) and an optional Watermill router. Tool orchestration lives in helpers that can also publish events. This page explains the event model, how events are published, how to route them with Watermill, and how to extend the event system with custom event types.

## Event Model

Engines and helpers publish structured chat events defined in `pkg/events/chat-events.go`:

- **start (`EventTypeStart`)**: Inference started for a request.
- **partial (`EventTypePartialCompletion`)**: Streamed delta and accumulated completion.
- **partial-thinking (`EventTypePartialThinking`)**: Streamed reasoning summary delta (Responses API only).
- **tool-call (`EventTypeToolCall`)**: Model requested a tool/function call.
- **tool-result (`EventTypeToolResult`)**: A tool returned its result.
- **error (`EventTypeError`)**: Error during streaming or execution.
- **interrupt (`EventTypeInterrupt`)**: Context cancelled; may include partial text.
- **final (`EventTypeFinal`)**: Inference finished; includes final text.
- status/text exist for legacy/debug use and may be phased out.
### Event Type Cheat Sheet

These are the concrete Go types and commonly used fields you will receive after parsing with `events.NewEventFromJson`:

- `*events.EventPartialCompletionStart`
  - Marks start of an inference stream
- `*events.EventPartialCompletion`
  - Fields: `Delta` (string), `Completion` (string)
- `*events.EventFinal`
  - Fields: `Text` (string)
- `*events.EventToolCall`
  - Fields: `ToolCall` (struct) with `Name` (string), `Input` (string), `ID` (string)
- `*events.EventToolResult`
  - Fields: `ToolResult` (struct) with `ID` (string), `Result` (string)
- `*events.EventError`
  - Fields: `ErrorString` (string)
- `*events.EventInterrupt`
  - Fields: `Text` (string)

See the source for full definitions: `geppetto/pkg/events/chat-events.go`.


Events carry `EventMetadata` only. `EventMetadata` includes:

- `message_id` (stable per stream)
- `run_id` and `turn_id` (correlation identifiers set by the caller/middleware)
- model information, optional stop reason, duration
- typed `Usage` with unified fields across providers
- an `extra` map for provider-specific context (e.g., settings snapshot)

Typed `Usage` (no maps) is defined in `pkg/events/metadata.go` and is populated continuously during streaming by engines:

- `InputTokens`, `OutputTokens`
- `CachedTokens` (OpenAI prompt cache)
- `CacheCreationInputTokens`, `CacheReadInputTokens` (Claude cache)

Providers update `EventMetadata.Usage` as chunks arrive, so UIs can display evolving usage in real-time. Map-based metadata extractors were removed; engines now use provider SDK typed fields directly.

## Publishing Events

There are two complementary ways events are published:

- **Engine-configured sinks**: Pass sinks when creating engines. Engines publish all lifecycle events to these sinks.

```go
  // Create Watermill sink that publishes to topic "chat"
  watermillSink := middleware.NewWatermillSink(router.Publisher, "chat")

  // Attach sink to engine via options
  eng, _ := factory.NewEngineFromParsedLayers(parsed, engine.WithSink(watermillSink))
  ```

- **Context-carried sinks**: Attach sinks to `context.Context` so downstream code (helpers, tools) can publish without plumbing.

```go
  // At call site, attach sinks to context
  ctx = events.WithEventSinks(ctx, watermillSink)

  // Downstream code can publish
  events.PublishEventToContext(ctx, events.NewToolResultEvent(meta, toolResult))
  ```

Engines emit: start, partial, final, interrupt, error (+ tool-call where applicable). Tool helpers emit: tool-call and tool-result. Middleware and tools may also emit lightweight `log` and `info` events for user-visible status.

## Provider Implementations and Event Flow

- **OpenAI (Chat Completions)**: Always streams. Publishes `start`, then `partial` for each delta, finally `final`. On context cancellation it publishes `interrupt`. Tool-call blocks are merged and published as `tool-call` events when complete.
- **OpenAI (Responses)**: Streams multiple channels of information:
  - Reasoning summary boundaries as `info` events ("thinking-started" / "thinking-ended").
  - Reasoning summary deltas as `partial-thinking` events (one per token-like chunk).
  - Output text deltas as `partial` events.
  - Function call arguments as SSE deltas; publishes a `tool-call` when function_call completes.
  - Final usage and reasoning token counts arrive on `response.completed` and are included in `final` metadata.
- **Claude**: Streaming is merged via a content-block merger which emits `start`, `partial` on text deltas, `tool-call` when a tool_use block completes, and `final` on stop.

Both engines publish to configured sinks and also call `events.PublishEventToContext(ctx, …)` so context-carried sinks receive the same events.

## Running the Event Router

Use a Watermill-backed `EventRouter` to route events to handlers. Run it in a goroutine and wait for it to be ready before invoking inference.

```go
router, _ := events.NewEventRouter()
defer router.Close()

// Add a simple printer
router.AddHandler("chat", "chat", events.StepPrinterFunc("", os.Stdout))

// Create sink and engine
sink := middleware.NewWatermillSink(router.Publisher, "chat")
eng, _ := factory.NewEngineFromParsedLayers(parsed, engine.WithSink(sink))

eg, groupCtx := errgroup.WithContext(ctx)

eg.Go(func() error { return router.Run(groupCtx) })
eg.Go(func() error {
    <-router.Running()
    // Attach same sink to context so helpers/tools can publish
    runCtx := events.WithEventSinks(groupCtx, sink)
    _, err := eng.RunInference(runCtx, messages)
    return err
})

_ = eg.Wait()
```

## Watermill’s Role

[Watermill](https://github.com/ThreeDotsLabs/watermill) provides the message router, publisher, and subscriber. Geppetto’s `EventRouter` wraps Watermill to simplify adding handlers:

```go
// Handler signature is func(*message.Message) error
router.AddHandler("chat", "chat", events.StepPrinterFunc("", w))
// Or structured output (text/json/yaml):
printer := events.NewStructuredPrinter(w, events.PrinterOptions{Format: events.FormatText})
router.AddHandler("chat", "chat", printer)
```

Use `middleware.NewWatermillSink(publisher, topic)` to publish engine events into the router.

Note: A legacy `PublisherManager` exists for lower-level control but is not required for engine-first workflows.

## Client-side Consumption Patterns

Clients consume events by adding handlers to the `EventRouter`. The router parses Watermill messages and your handler can print, aggregate, or transform events.

### 1) Console streaming printer (simple)

```go
// Text streaming (writes deltas and final newline)
router.AddHandler("chat", "chat", events.StepPrinterFunc("", os.Stdout))

// Or a structured printer (text/json/yaml), optionally with metadata
printer := events.NewStructuredPrinter(os.Stdout, events.PrinterOptions{
    Format:          events.FormatText, // or events.FormatJSON / events.FormatYAML
    IncludeMetadata: false,
    Full:            false,
})
router.AddHandler("chat", "chat", printer)
```

This is the same pattern used in the example client (see `geppetto/cmd/examples/simple-streaming-inference/main.go`).

### 2) Custom handler (aggregate deltas, handle errors)

```go
// import "github.com/ThreeDotsLabs/watermill/message"

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

`ToolCall.Input` and `ToolResult.Result` are strings. If they contain JSON, pretty-print them for readability:

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

### 3) Cross-process consumption

`EventRouter` defaults to an in-process GoChannel pub/sub. For cross-process setups, initialize with a different Watermill transport and pass it via options:

```go
pub := /* e.g., NATS/Kafka publisher */
sub := /* e.g., NATS/Kafka subscriber */
router, _ := events.NewEventRouter(events.WithPublisher(pub), events.WithSubscriber(sub))

// Your handlers remain the same
router.AddHandler("chat", "chat", events.StepPrinterFunc("", os.Stdout))
```

With a networked publisher/subscriber, any client subscribed to the `chat` topic can consume events produced by engines running elsewhere.

## Event Extensibility and Custom Events

Geppetto provides an event registry that allows you to register custom event types beyond the built-in events. This enables libraries, tools, and application-specific code to emit domain-specific events that flow through the same event infrastructure.

### The Event Registry

The registry is a thread-safe, singleton store of decoders and encoders. When `events.NewEventFromJson` deserializes an event, it first checks the registry for a custom decoder matching the event's `type` field. If found, the custom decoder runs; otherwise, Geppetto's built-in event types are used.

### Registration Methods

The `pkg/events` package provides three registration functions:

#### 1. RegisterEventCodec (full control)

Register a decoder function with complete control over deserialization:

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

For standard JSON unmarshalling, use the factory helper:

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

The factory approach handles unmarshalling and payload storage automatically.

#### 3. RegisterEventEncoder (outbound serialization)

Register an encoder for custom serialization logic:

```go
func init() {
    encoder := func(ev events.Event) ([]byte, error) {
        // Custom serialization logic
        return json.Marshal(ev)
    }
    
    _ = events.RegisterEventEncoder("custom-progress", encoder)
}
```

### Publishing Custom Events

Custom events are published the same way as built-in events:

```go
// Create custom event with metadata
meta := events.EventMetadata{
    ID:     uuid.New(),
    RunID:  "run-123",
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

Custom events flow through the same Watermill router and handlers:

```go
router.AddHandler("custom-collector", "chat", func(msg *message.Message) error {
    defer msg.Ack()
    
    e, err := events.NewEventFromJson(msg.Payload)
    if err != nil {
        return err
    }
    
    // Type-assert to your custom event
    if progressEv, ok := e.(*CustomProgressEvent); ok {
        fmt.Printf("Progress: %.0f%% - %s\n", 
            progressEv.Progress*100, progressEv.Status)
    }
    
    return nil
})
```

### Best Practices for Custom Events

- **Embed EventImpl**: All custom events should embed `events.EventImpl` to satisfy the `Event` interface and carry standard metadata.
- **Register in init()**: Use `init()` functions to register codecs/factories at package initialization time.
- **Unique type names**: Choose distinctive type strings to avoid collisions (e.g., `myapp-progress` rather than `progress`).
- **Metadata consistency**: Always populate `EventMetadata` with `message_id`, and optionally `run_id`/`turn_id` for correlation.
- **Handle registration errors**: Check the error return from registration functions; duplicate registrations will fail.

### Use Cases

Custom events are useful for:

- **Tool-specific progress**: Tools can emit granular progress events (e.g., database query progress, file upload status)
- **Domain events**: Application-specific events like "user-action", "workflow-step-completed"
- **Integration events**: Events from external systems flowing through the same infrastructure
- **Debugging events**: Instrumentation and profiling events during development

## Practical Tips

- Always wait for `router.Running()` before invoking inference to avoid dropped events.
- Attach the same sink to both the engine and the context so helpers/tools can publish.
- Prefer `events.NewStructuredPrinter` for machine-readable output during tests.
- Register custom event types in `init()` functions to ensure they're available before event processing begins.
- Enable debug logging (`zerolog.DebugLevel`) to see when events are published to context sinks and when no sinks are available.
- For related guidance, see: `glaze help geppetto-inference-engines`.

## Full Example: Router + Engine + Helpers

While the sections above detail the internal mechanisms, a common practical pattern is needed when *using* the `EventRouter` in an application.

The `router.Run(ctx)` method is blocking and listens for events until its context is canceled. Therefore, it must typically be run in a background goroutine.

Simultaneously, the application logic that triggers event publishing (e.g., calling `step.Start` or a higher-level function like `llm.Generate` which uses steps internally) needs to execute. This logic often depends on the router being active to receive published events.

To coordinate these concurrent operations and ensure clean startup, shutdown, and avoid race conditions, it's recommended to use `golang.org/x/sync/errgroup`:

1.  Create a parent `context.Context` with cancellation (`context.WithCancel`).
2.  Create an `errgroup.Group` associated with this cancellable context (`eg, groupCtx := errgroup.WithContext(ctx)`). Using `errgroup.WithContext` automatically handles cancellation propagation if any goroutine in the group returns an error.
3.  Launch `router.Run(groupCtx)` in one goroutine managed by the `errgroup` (`eg.Go`).
    *   **Crucially**, use `defer router.Close()` inside this goroutine to ensure the router's resources are released when it stops.
    *   It's also good practice to include `defer cancel()` here, although `errgroup.WithContext` handles cancellation on error, this ensures cancellation if the goroutine exits cleanly.
4.  Launch the main event-producing task (e.g., `llm.Generate(groupCtx, ...)` or `step.Start(groupCtx, ...)` ) in another goroutine managed by the `errgroup`.
    *   **Wait for the router**: Before the task starts publishing events, wait for the router to be ready by reading from its `Running()` channel: `<-router.Running()`.
    *   Use `defer cancel()` in this goroutine as well to signal the router goroutine to stop if this task finishes or fails.
5.  Call `eg.Wait()` in the main thread. This blocks until all goroutines launched via `eg.Go` have finished (either successfully or due to an error/cancellation).

This pattern ensures that:
- The router is confirmed to be running before the main task starts publishing.
- The router stays alive as long as the main task is running.
- If the main task finishes or fails, the router is signalled to shut down via context cancellation.
- If the router fails unexpectedly, the main task is signalled to stop via context cancellation.
- The router's `Close()` method is called reliably upon shutdown.
- The application waits for both components to terminate cleanly before exiting.

Example structure:

```go
ctx, cancel := context.WithCancel(context.Background())
// NOTE: No defer cancel() here, errgroup handles it via groupCtx

eg, groupCtx := errgroup.WithContext(ctx)
router, _ := events.NewEventRouter() // Assume router creation

// Goroutine for the router
eg.Go(func() error {
    defer func() {
        log.Info().Msg("Closing event router")
        _ = router.Close() // Ensure router is closed
        log.Info().Msg("Event router closed")
        // cancel() // Optional: Signal cancellation if router exits cleanly
    }()

    log.Info().Msg("Starting event router")
    runErr := router.Run(groupCtx) // Use group's context
    log.Info().Err(runErr).Msg("Event router stopped")
    // Don't return context.Canceled as a fatal error from the group
    if runErr != nil && !errors.Is(runErr, context.Canceled) {
        return runErr // Return other errors
    }
    return nil
})

// Goroutine for the main task (e.g., LLM call)
eg.Go(func() error {
    defer cancel() // Signal router to stop when this task is done

    // Wait for router to be ready
    <-router.Running()
    log.Info().Msg("Event router is running, proceeding with main task")

    // Run inference (helpers will also publish via context sinks)
    _, err := eng.RunInference(events.WithEventSinks(groupCtx, watermillSink), conversation)
    if err != nil {
        return err
    }
    
    log.Info().Msg("Main task finished")
    return nil 
})

// Wait for both goroutines to complete
log.Info().Msg("Waiting for goroutines to finish")
if err := eg.Wait(); err != nil {
    log.Error().Err(err).Msg("Application finished with error")
    // Handle error (errgroup returns the first non-nil error)
} else {
    log.Info().Msg("Application finished successfully")
}
```
