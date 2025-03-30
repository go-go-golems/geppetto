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

AI steps publish a variety of event types:

- `EventTypeStart`: When the step begins execution
- `EventTypePartialCompletion`: During streaming, as chunks of the response arrive
- `EventTypeToolCall`: When an AI model calls a tool
- `EventTypeToolResult`: When a tool returns a result
- `EventTypeFinal`: When the step has completed
- `EventTypeInterrupt`: When the step is interrupted
- `EventTypeError`: When an error occurs

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
csf.publisherManager.PublishBlind(chat.NewStartEvent(metadata, stepMetadata))

// Stream a response
for {
    // ...process response chunk...
    
    // Publish partial completion
    csf.publisherManager.PublishBlind(
        chat.NewPartialCompletionEvent(
            metadata,
            stepMetadata,
            delta, message),
    )
}

// Publish completion
csf.publisherManager.PublishBlind(chat.NewFinalEvent(metadata, stepMetadata, message))
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

Geppetto uses Watermill's message router for connecting publishers to subscribers:

```go
func (e *EventRouter) AddHandler(name string, topic string, f func(msg *message.Message) error) {
    e.router.AddNoPublisherHandler(name, topic, e.Subscriber, f)
}
```

The router manages the flow of messages and provides features like middleware, metrics, and recovery.

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

1. When the AI decides to call a tool, a `ToolCallEvent` is published
2. The tool executes and a `ToolResultEvent` is published
3. The AI continues processing with the tool result

This enables monitoring of the entire tool calling flow.

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
