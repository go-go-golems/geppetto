# Geppetto Step Abstraction: Pub/Sub Architecture with Watermill Integration

## Architectural Overview

The Geppetto framework uses a sophisticated publish-subscribe pattern combined with monadic step processing to handle AI operations. This system enables:

1. **Asynchronous processing** of LLM interactions
2. **Real-time event streaming** for UI updates and monitoring
3. **Decoupled architecture** for extensibility

### Core Components

![Architecture Diagram](https://mermaid.ink/svg/pako:eNpVkE1PwzAMhv9KlNd1H5u0aRMSEhKHgTjAYRzSJm5Hk7ZJN6GE-O9x-4F0suX3eZ_tVHY4QmM7VJqG4A1qKqXCd3zBwQeHlalx8BZf8QkHqXG0HXZQ4wQdDlDhCBYq_MQ3fMZeaqyNw8k5_MIX7KTCSrfYW4uT7fEDX1FLhY1usTcWR9viO75hKxU2psXeWBxsix_4iq1U2JoWe2NxsC1-4gu2UmFnWuyNxcG2-IHP2EmFvWmxNxYH2-I7PmEnFQ6mxd5YHGyLb_iIrVQ4mhZ7Y3GwLb7hA7ZS4WRa7I3Fwbb4ivfYSIWzabE3Fgfb4gveYSMVLqbF3lgcbIsveIu1VLiaFntjcbAtvuANVlLhZlrsjcXBtviM11hKhbtpsTcWB9viE15hIRUepsXeWBxsi494gblUeJoWe2NxsC0-4DlmUuFlWuyNxcG2eI9nmEqFt2mxNxYH2-IpniBS4QN6HGyH_1uLfwc5bSU)

**Key Elements:**
1. `ChatStep` (AI operation handler)
2. `PublisherManager` (Event multiplexer)
3. Watermill (Message bus)
4. Event Consumers (UI, Monitoring, etc.)

## Implementation Deep Dive

### 1. Event Publishing in AI Steps (`chat-step.go`)

AI steps like `ChatStep` leverage the publisher system through three key phases:

```go
func (csf *ChatStep) Start(ctx context.Context, messages conversation.Conversation) (steps.StepResult[*conversation.Message], error) {
    // Phase 1: Event initialization
    metadata := chat.EventMetadata{
        ID:       conversation.NewNodeID(),
        ParentID: parentID,
        LLMMessageMetadata: conversation.LLMMessageMetadata{
            Engine: string(*csf.Settings.Chat.Engine),
        },
    }
    
    // Phase 2: Event publishing
    csf.publisherManager.PublishBlind(chat.NewStartEvent(metadata, stepMetadata))
    
    // Phase 3: Streaming handling
    for {
        response, err := stream.Recv()
        if err == io.EOF {
            csf.publisherManager.PublishBlind(chat.NewFinalEvent(...))
            break
        }
        csf.publisherManager.PublishBlind(chat.NewPartialCompletionEvent(...))
    }
}
```

**Key Features:**
- Strong typing with `EventMetadata` and `StepMetadata`
- Context-aware cancellation propagation
- Automatic error handling and event sequencing

### 2. PublisherManager Mechanics (`publish.go`)

The `PublisherManager` acts as a smart router:

```go
// Registration example
pm := events.NewPublisherManager()
pm.RegisterPublisher("chat_events", watermillPublisher)

// Publishing logic
func (s *PublisherManager) Publish(payload interface{}) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    // Serialization
    b, _ := json.Marshal(payload)
    
    // Watermill message creation
    msg := message.NewMessage(watermill.NewUUID(), b)
    msg.Metadata.Set("sequence_number", fmt.Sprintf("%d", s.sequenceNumber))
    
    // Fan-out to all registered publishers
    for topic, subs := range s.Publishers {
        for _, sub := range subs {
            sub.Publish(topic, msg) // Watermill interface
        }
    }
    
    s.sequenceNumber++
    return nil
}
```

**Critical Features:**
- Automatic sequence numbering for event ordering
- Thread-safe publisher registration
- Blind publishing pattern (fire-and-forget with error logging)

### 3. Watermill Integration

Watermill serves as the messaging backbone through:

1. **Standardized Interface**
```go
type Step interface {
    AddPublishedTopic(publisher message.Publisher, topic string) error
}
```

2. **Message Structure**
```go
msg := message.NewMessage(
    watermill.NewUUID(),      // Unique message ID
    serializedPayload,        // JSON-encoded event
)
msg.Metadata.Set("sequence_number", "123") // Ordered delivery
```

3. **Pluggable Transports**
```yaml
# Example configuration
pubsub:
  redis:
    client: "default"
    marshaler: "json"
```

**Supported Backends:**
- Redis Streams
- Google Cloud Pub/Sub
- NATS
- Kafka
- In-memory (for testing)

### 4. Event Type System (`events.go`)

The event hierarchy enables complex state tracking:

```go
type Event interface {
    Type() EventType
    Metadata() EventMetadata
    StepMetadata() *steps.StepMetadata
}

// Example concrete type
type EventPartialCompletion struct {
    EventImpl
    Delta       string `json:"delta"`
    Completion  string `json:"completion"`
}
```

**Event Lifecycle:**
1. `Start` → Initializes conversation
2. `PartialCompletion` (0..n) → Streaming updates
3. Terminal Event:
   - `Final` → Successful completion
   - `Error` → Operation failure
   - `Interrupt` → User cancellation

## Why This Architecture?

### Benefits
1. **Observability**
   - Sequence numbers enable event replay
   - Structured logging via zerolog integration
   ```go
   func (e EventPartialCompletion) MarshalZerologObject(ev *zerolog.Event) {
       e.EventImpl.MarshalZerologObject(ev)
       ev.Str("delta", e.Delta)
          .Str("completion", e.Completion)
   }
   ```

2. **Extensibility**
   - Add new consumers without modifying steps
   - Example: Add monitoring subscriber
   ```go
   pm.RegisterPublisher("chat_events", monitoringPublisher)
   ```

3. **Error Resilience**
   - Blind publishing prevents event system failures from crashing steps
   - Automatic context cancellation propagation

4. **Multi-modal Support**
   - Handle text and tool calls simultaneously
   ```go
   type EventToolCall struct {
       EventImpl
       ToolCall ToolCall `json:"tool_call"`
   }
   ```

### Tradeoffs
1. **Eventual Consistency**
   - No strict ordering across topics
   - Limited transactional guarantees

2. **Performance Considerations**
   - JSON serialization overhead
   - Watermill middleware chain adds latency

3. **Memory Management**
   - Care required with channel buffer sizes
   - Potential for goroutine leaks if not properly managed

## Typical Usage Pattern

**1. Step Initialization**
```go
chatStep := openai.NewStep(
    settings,
    openai.WithSubscriptionManager(pm),
)
```

**2. Event Consumption**
```go
messages, _ := subscriber.Subscribe(ctx, "chat_events")
for msg := range messages {
    event, err := chat.NewEventFromJson(msg.Payload)
    if err != nil {
        msg.Nack()
        continue
    }
    
    switch e := event.(type) {
    case *chat.EventPartialCompletion:
        fmt.Printf("Delta: %s\n", e.Delta)
    case *chat.EventFinal:
        fmt.Printf("Final output: %s\n", e.Text)
    }
    
    msg.Ack()
}
```

**3. Monitoring Integration**
```go
pm.RegisterPublisher("chat_events", prometheusAdapter)
pm.RegisterPublisher("chat_events", elasticsearchAdapter)
```

## Debugging Tips

1. **Sequence Inspection**
```bash
watermill inspect --metadata sequence_number
```

2. **Event Replay**
```go
// Replay last 100 events
subscriber.Subscribe(ctx, "chat_events", watermill.WithReplay(100))
```

3. **Context Tracing**
```go
msg.Metadata.Set("trace_id", traceID) // Propagate across services
```

This architecture enables Geppetto to handle complex AI workflows while maintaining observability and extensibility. The Watermill integration provides enterprise-grade messaging capabilities without locking into specific infrastructure choices.
