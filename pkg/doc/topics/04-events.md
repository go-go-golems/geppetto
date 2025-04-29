---
Title: Steps, PubSub, and Watermill in Geppetto
Slug: geppetto-steps-pubsub-watermill
Short: Explains the step abstraction, event publishing via Watermill, and their use in Geppetto's AI steps.
Topics:
- geppetto
- architecture
- events
- steps
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

# Steps, PubSub, and Watermill in Geppetto

This document explains how the step abstraction in Geppetto leverages publishers and topics, with a focus on AI steps, and the role that the Watermill library plays in this architecture.

## Step Abstraction and Publisher/Topic System

### Core Step Interface

The Step interface in Geppetto is defined as a generic interface with two type parameters:

```go
type Step[T any, U any] interface {
    Start(ctx context.Context, input T) (StepResult[U], error)
    AddPublishedTopic(publisher message.Publisher, topic string) error
}
```

Every Step implementation must provide an `AddPublishedTopic` method, which allows the step to publish events to subscribers. This method associates a Watermill publisher with a specific topic, enabling event-driven communication.

### StepResult - The Return Monad

The `StepResult` interface represents the computation result:

```go
type StepResult[T any] interface {
    Return() []helpers.Result[T]
    GetChannel() <-chan helpers.Result[T]
    Cancel()
    GetMetadata() *StepMetadata
}
```

This interface embodies several monadic patterns:
1. **List Monad**: It can contain multiple results via the channel
2. **Maybe Monad**: Results can be values or errors (via `helpers.Result`)
3. **Cancellation Monad**: Operations can be cancelled
4. **Metadata Monad**: Carries metadata about the computation

The key insight is that while `StepResult` handles the *value* flow through its channel, the *event* flow happens via the publisher/topic system, creating two parallel communication paths.

### How Steps Use Publishers and Topics

Steps use publishers and topics to:

1. Broadcast their internal state changes
2. Report progress during execution
3. Stream partial results
4. Notify about completion or errors

The `AddPublishedTopic` method provides a way for external code to inject a publisher and topic into a step, establishing a channel for the step to communicate its progress and results outside of its primary return value.

### Step Composition with Bind

Steps are designed for composition using the `Bind` function, which implements the monadic bind operator:

```go
func Bind[T any, U any](
    ctx context.Context,
    m StepResult[T],
    step Step[T, U],
) StepResult[U] {
    // Implementation handles chaining steps together
}
```

This function enables complex pipelines by:
1. Taking the result of one step
2. Feeding it into another step
3. Propagating cancellation
4. Handling errors

While values flow through binding, events from each step still flow independently through the publisher system, allowing consumers to observe every step in the chain.

## Context Cancellation and Event Flow

Steps handle context cancellation in a consistent way:

```go
func (csf *ChatStep) Start(ctx context.Context, input conversation.Conversation) (steps.StepResult[*conversation.Message], error) {
    var cancel context.CancelFunc
    cancellableCtx, cancel := context.WithCancel(ctx)
    go func() {
        <-ctx.Done()
        cancel()
    }()
    
    // ... step implementation ...
    
    // When cancellation occurs:
    csf.publisherManager.PublishBlind(chat.NewInterruptEvent(metadata, stepMetadata, message))
}
```

This ensures:
1. Cancellation propagates to child operations
2. Cancellation events are published to subscribers
3. Resources are properly released
4. Long-running steps like AI completions terminate gracefully

## AI Steps and Event Publishing

AI steps (like OpenAI and Claude integrations) make extensive use of the publisher/topic system to stream their execution progress. This is particularly important for several reasons:

1. **Streaming Responses**: AI completions can take time, and streaming allows for progressive rendering of responses
2. **Tool Calling**: AI steps that use tools need to publish when tools are called and when results are returned
3. **Monitoring**: Steps publish metadata about token usage, engines, and other parameters
4. **Caching**: Cached responses need to publish events that mimic what the original execution would have published

### Event Types for AI Steps

AI steps publish a variety of event types, defined in `pkg/events/chat-events.go`. Each corresponds to a specific stage or outcome:

- **`events.EventPartialCompletionStart` (`EventTypeStart`)**: Signals the beginning of a step's execution, particularly one that might stream partial results. (*Note: `EventTypeStart` is used internally by this event type*).
- **`events.EventPartialCompletion` (`EventTypePartialCompletion`)**: Represents an incremental chunk of a response during streaming (typically text).
- **`events.EventToolCall` (`EventTypeToolCall`)**: Published when the AI model requests a tool/function call.
- **`events.EventToolResult` (`EventTypeToolResult`)**: Published when a tool provides its result back to the step.
- **`events.EventFinal` (`EventTypeFinal`)**: Signals the successful completion of the step, usually containing the final aggregated text.
- **`events.EventInterrupt` (`EventTypeInterrupt`)**: Published if the step is interrupted (e.g., via context cancellation).
- **`events.EventError` (`EventTypeError`)**: Published when an error occurs during the step's execution.
- **`events.EventText` (`EventTypeStart` or potentially `EventTypeText`)**: Represents a simple text message event. (*Note: Its exact role and usage might overlap with other events and is under review, see TODOs in `chat-events.go`*).
- (*`EventTypeStatus` is also defined but its usage is currently unclear*).

These events carry metadata (`EventMetadata`) and step metadata (`StepMetadata`) and are serialized to JSON for transport.

### Implementation in AI Steps

Looking at the OpenAI chat step implementation as an example:

```go
func (csf *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
    csf.publisherManager.RegisterPublisher(topic, publisher)
    return nil
}
```

AI steps typically use a `PublisherManager` to handle multiple publishers and topics. When streaming a response:

```go
// Publish the start event
// Note: Constructor names match the event structs
csf.publisherManager.PublishBlind(events.NewStartEvent(metadata, stepMetadata))

// Stream a response
for {
    // ...process response chunk...
    
    // Publish partial completion
    csf.publisherManager.PublishBlind(
        events.NewPartialCompletionEvent(
            metadata,
            stepMetadata,
            delta, message),
    )
}

// Publish completion
csf.publisherManager.PublishBlind(events.NewFinalEvent(metadata, stepMetadata, message))
```

This pattern allows for realtime updates of AI processing status, critical for interactive applications.

## Event Flow Architecture

The full event flow in Geppetto follows this pattern:

1. **Step Creation**: A step is created with a `PublisherManager`
2. **Registration**: Publishers are registered for topics via `AddPublishedTopic`
3. **Execution**: The step is started with input
4. **Event Publishing**: During execution, events are published to all registered topics
5. **Event Routing**: Events are routed to subscribers via Watermill
6. **Consumption**: Subscribers (UI components, loggers, etc.) consume events
7. **Result Return**: Separately, the step returns values through its `StepResult`

This dual-flow architecture (events and values) enables:
- Immediate feedback even for long-running operations
- Detailed monitoring and logging
- Clean separation of control flow and status reporting
- Composition of complex pipelines while maintaining observability

## Watermill's Role

[Watermill](https://github.com/ThreeDotsLabs/watermill) is a Go library for working with message streams. It provides abstractions for:

1. Publishing messages
2. Subscribing to messages
3. Routing messages between publishers and subscribers

### Key Watermill Components Used in Geppetto

#### Publishers and Subscribers

In Geppetto, Watermill provides the publisher and subscriber interfaces:

```go
type Publisher interface {
    Publish(topic string, messages ...*Message) error
    Close() error
}

type Subscriber interface {
    Subscribe(ctx context.Context, topic string) (<-chan *Message, error)
    Close() error
}
```

These interfaces allow for different message transport implementations (in-memory Go channels, NATS, Kafka, etc.) with a consistent API.

#### Message Router

Geppetto uses Watermill's message router, often via the `EventRouter` abstraction (`pkg/events/event-router.go`), for connecting publishers to subscribers:

```go
// From EventRouter
func (e *EventRouter) AddHandler(name string, topic string, f func(msg *message.Message) error) {
    e.router.AddNoPublisherHandler(name, topic, e.Subscriber, f)
}
```

The router will publish events to all registered topics.

The router manages the flow of messages and provides features like middleware, metrics, and recovery.

#### ChatEventHandler Interface

The `EventRouter` also defines a specific interface for handling common chat events:

```go
// Defined in pkg/events/event-router.go
type ChatEventHandler interface {
    HandlePartialCompletion(ctx context.Context, e *EventPartialCompletion) error
    HandleText(ctx context.Context, e *EventText) error
    HandleFinal(ctx context.Context, e *EventFinal) error
    HandleError(ctx context.Context, e *EventError) error
    HandleInterrupt(ctx context.Context, e *EventInterrupt) error
    // Note: HandleToolCall and HandleToolResult are not yet part of this standard interface.
}
```

The `EventRouter.RegisterChatEventHandler` method simplifies connecting a `chat.Step` to an implementation of this interface. It creates a dispatcher that receives messages, parses them using `events.NewEventFromJson`, and calls the appropriate `Handle*` method.

*Important*: While `EventToolCall` and `EventToolResult` events *exist* and are published by steps, the standard `ChatEventHandler` interface and its dispatcher in `EventRouter` may not yet have dedicated methods to handle them. Custom handlers attached via `AddHandler` would be needed to process these currently, or the interface would need to be extended.

### PublisherManager

Geppetto extends Watermill with a `PublisherManager` that manages multiple publishers for different topics:

```go
type PublisherManager struct {
    Publishers     map[string][]message.Publisher
    sequenceNumber uint64
    mutex          sync.Mutex
}
```

The manager:
1. Allows registering publishers for specific topics
2. Adds sequence numbers to messages for ordering
3. Broadcasts messages to all relevant publishers
4. Handles serialization of event payloads to JSON

#### Sequence Numbers

The `PublisherManager` adds sequence numbers to each message:

```go
msg := message.NewMessage(watermill.NewUUID(), b)
msg.Metadata.Set("sequence_number", fmt.Sprintf("%d", s.sequenceNumber))
s.sequenceNumber++
```

This ensures that consumers can reconstruct the order of events even when they arrive out of order, critical for displaying streamed content correctly.

## Practical Examples

### Streaming Chat Completion

When an OpenAI chat step is executed with streaming enabled:

1. The step is initialized with a `PublisherManager`
2. External code calls `AddPublishedTopic` to register a publisher and topic
3. When `Start` is called, the step publishes a `StartEvent`
4. As chunks of the response arrive, the step publishes `PartialCompletionEvent`s
5. When complete, the step publishes a `FinalEvent`

This allows a UI or other consumer to display the AI response as it's generated, rather than waiting for the entire response.

### Tool Calling

For AI steps that support tool calling:

1. When the AI decides to call a tool, an `events.EventToolCall` is published.
2. The tool executes (often managed by a separate step or mechanism).
3. When the result is available, an `events.EventToolResult` is published (often consumed by the original AI step to continue processing).

This enables monitoring of the entire tool calling flow. Subscribers can listen for these specific event types.

### Composing AI Steps with Tools

When composing an AI step with tool execution:

1. An AI step starts and generates content or tool calls
2. Tool calls are captured and published as events
3. Another step executes the tools
4. Tool results are passed back to the original AI step
5. The AI continues processing with the tool results
6. Events from all steps flow independently to subscribers

This composition is possible because:
- The value flow is handled by the Step/StepResult system
- The event flow is handled by the publisher/topic system
- Context cancellation propagates correctly through the entire chain

## Practical Usage: Running the Event Router

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

    // ... your logic that publishes events ...
    // err := llm.Generate(groupCtx, ...)
    // if err != nil { 
    //    return err // errgroup will trigger cancellation
    // }
    
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
