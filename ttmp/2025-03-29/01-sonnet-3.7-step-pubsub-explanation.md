# Explanation of Step Publishing and Watermill in Geppetto

This document explains the publisher-subscriber (pub-sub) mechanism in Geppetto's Step abstraction, focusing on how AI steps leverage this architecture to communicate events and the role Watermill plays as the underlying messaging system.

## 1. Steps and Publishing Mechanism

The Geppetto framework is built around the core abstraction of `Step`, which represents a computational unit that performs a particular task, such as generating text with an AI model or processing data. The Step interface includes:

```go
type Step[T any, U any] interface {
    Start(ctx context.Context, input T) (StepResult[U], error)
    AddPublishedTopic(publisher message.Publisher, topic string) error
}
```

### How Steps Use Publishers and Topics

1. **Publisher Registration**: Steps receive a publisher and topic through the `AddPublishedTopic` method, which they use to emit events during execution.

2. **Implementation Pattern**: AI steps typically store the publisher and topic in a `PublisherManager`, which manages multiple publisher/topic pairs.

3. **Event Publication**: During execution, steps publish various events (start, progress, completion, errors) to inform subscribers about their state.

4. **Event Types**: AI steps publish events like:
   - `chat.NewStartEvent`: When processing begins
   - `chat.NewPartialCompletionEvent`: For streaming partial results
   - `chat.NewFinalEvent`: When processing completes
   - `chat.NewErrorEvent`: When errors occur
   - `chat.NewInterruptEvent`: When processing is interrupted

### Example from OpenAI Chat Step

The OpenAI `ChatStep` implementation demonstrates how an AI step uses publishing:

```go
// During initialization
func (csf *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
    csf.publisherManager.RegisterPublisher(topic, publisher)
    return nil
}

// During execution
func (csf *ChatStep) Start(ctx context.Context, messages conversation.Conversation) (steps.StepResult[*conversation.Message], error) {
    // Setup...
    
    // Announce start
    csf.publisherManager.PublishBlind(chat.NewStartEvent(metadata, stepMetadata))
    
    // During streaming
    csf.publisherManager.PublishBlind(
        chat.NewPartialCompletionEvent(
            metadata,
            stepMetadata,
            delta, message),
    )
    
    // On completion
    csf.publisherManager.PublishBlind(chat.NewFinalEvent(
        metadata,
        stepMetadata,
        message,
    ))
    
    // On error
    csf.publisherManager.PublishBlind(chat.NewErrorEvent(metadata, stepMetadata, err.Error()))
}
```

## 2. Role of Watermill

Watermill (`github.com/ThreeDotsLabs/watermill`) is a Go library that provides the messaging infrastructure for Geppetto's pub-sub system:

### Key Aspects of Watermill

1. **Abstraction**: Watermill provides standard interfaces like `message.Publisher` and `message.Subscriber`, decoupling steps from specific message transport mechanisms.

2. **Message Formats**: Messages in Watermill consist of:
   - A UUID
   - Payload (binary data)
   - Metadata (key-value pairs)

3. **Transport Agnostic**: Watermill supports various message brokers (Go channels, AMQP, Kafka, etc.), allowing Geppetto to use the most appropriate transport without changing step logic.

4. **Message Routing**: Watermill's router component connects publishers to subscribers based on topic names, handling message delivery.

### How Geppetto Uses Watermill

1. **PublisherManager**: Geppetto wraps Watermill's publisher in a `PublisherManager` class that:
   - Manages multiple publisher/topic pairs
   - Serializes event objects to JSON
   - Adds sequence numbers to messages
   - Provides error handling

2. **Event Router**: Geppetto's `EventRouter` creates Watermill publishers and subscribers, connects them via topics, and allows components to register handlers for events.

3. **Decoupling**: Using Watermill allows steps to publish events without knowing who is listening or how messages are transported, making the system more flexible and modular.

## 3. Integration Flow

The complete flow from AI step execution to event handling works as follows:

1. A client creates an AI step (like `ChatStep`) and configures it.

2. The client registers a publisher and topic with the step via `AddPublishedTopic`.

3. When executed, the step processes its input and publishes events during key moments (start, progress, completion, errors).

4. Watermill delivers these events to all subscribers of the specified topic.

5. Subscribers (like UIs, loggers, or other steps) receive and process these events, updating displays, logging information, or triggering further actions.

This architecture enables:
- Real-time streaming of AI outputs
- Monitoring of AI processing
- Loose coupling between components
- Flexible handling of events by multiple subscribers

For example, a web UI might subscribe to these events to show typing animations, a logger might record them for auditing, and another component might collect metrics—all without the AI step needing to know about any of these consumers. 