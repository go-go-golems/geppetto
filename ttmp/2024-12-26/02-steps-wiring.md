# Creating and Running Steps with Event Publishing in Geppetto

## Overview

Geppetto uses a powerful abstraction called Steps to handle asynchronous operations that transform inputs into streams of outputs. These steps can publish events during their execution using the Watermill framework, which are then routed to handlers like UI components or terminal output.

## Step Creation and Event Publishing

### Basic Step Structure

A Step in Geppetto must implement the Step interface:

```go
type Step[T any, U any] interface {
    Start(ctx context.Context, input T) (StepResult[U], error)
    AddPublishedTopic(publisher message.Publisher, topic string) error
}
```

Key components:
- `Start`: Initiates the step's operation
- `AddPublishedTopic`: Registers a Watermill publisher for event broadcasting
- `StepResult`: A monad combining streaming, error handling, and metadata

### Publishing Events

Steps publish events through Watermill topics. Here's the typical flow:

1. Register a publisher:
```go
type ChatStep struct {
    subscriptionManager *events.PublisherManager
}

func (s *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
    return s.subscriptionManager.RegisterPublisher(topic, publisher)
}
```

2. Publish events during execution:
```go
// Example of publishing a start event
s.subscriptionManager.PublishBlind(chat.NewStartEvent(metadata, stepMetadata))

// Example of publishing partial completion
s.subscriptionManager.PublishBlind(chat.NewPartialEvent(metadata, stepMetadata, partial))
```

## Event Types and Structure

Events in Geppetto follow a structured format defined in `events.go`:

### Core Event Types:
- `EventTypeStart`: Initial event when step begins
- `EventTypePartialCompletion`: Intermediate streaming results
- `EventTypeFinal`: Final completion event
- `EventTypeToolCall`: Tool invocation events
- `EventTypeToolResult`: Tool execution results
- `EventTypeError`: Error conditions
- `EventTypeInterrupt`: Processing interruption

### Event Structure:
```go
type Event interface {
    Type() EventType
    Metadata() EventMetadata
    StepMetadata() *steps.StepMetadata
    Payload() []byte
}
```

Each event carries:
- Type information
- Event metadata (message ID, parent ID)
- Step metadata (step ID, type, input/output types)
- Payload specific to the event type

## Setting Up Event Routing

### Creating an Event Router

In the GeppettoCommand's `RunIntoWriter`, a router is created and configured:

```go
router, err := events.NewEventRouter()
if err != nil {
    return err
}
defer router.Close()

// Add handlers for different topics
router.AddHandler("chat", "chat", chat.StepPrinterFunc("", w))
```

### Connecting Steps to Handlers

1. Create a step with publishing capability:
```go
chatStep, err = stepFactory.NewStep(chat.WithPublishedTopic(router.Publisher, "chat"))
```

2. Set up handlers to process events:
```go
// Example from backend.go for UI handling
router.AddHandler("ui", "ui", StepChatForwardFunc(p))
```

## Event Handling and UI Integration

### Converting Events to UI Messages

The `StepChatForwardFunc` in `backend.go` shows how to transform Watermill events into UI messages:

1. Parse incoming events:
```go
e, err := chat.NewEventFromJson(msg.Payload)
```

2. Convert to appropriate UI messages:
```go
switch e_ := e.(type) {
case *chat.EventPartialCompletion:
    p.Send(conversation2.StreamCompletionMsg{
        StreamMetadata: metadata,
        Delta:         e_.Delta,
        Completion:    e_.Completion,
    })
// ... handle other event types
}
```

## Complete Flow Example

1. Create and configure a step:
```go
stepFactory := &ai.StandardStepFactory{
    Settings: stepSettings,
}

router, _ := events.NewEventRouter()
chatStep, _ := stepFactory.NewStep(
    chat.WithPublishedTopic(router.Publisher, "chat"))
```

2. Set up event handling:
```go
router.AddHandler("chat", "chat", chat.StepPrinterFunc("", w))
```

3. Run the step:
```go
result, _ := chatStep.Start(ctx, input)
for res := range result.GetChannel() {
    // Process results while events are published asynchronously
}
```

## Best Practices

1. Event Publishing:
   - Always include appropriate metadata with events
   - Use specific event types for different operations
   - Handle errors through the event system

2. Event Routing:
   - Set up handlers before starting steps
   - Use appropriate topics for different types of handlers
   - Clean up resources using defer statements

3. UI Integration:
   - Transform events into appropriate UI messages
   - Handle all relevant event types
   - Maintain proper error handling and resource cleanup

This system provides a flexible and powerful way to handle asynchronous operations while maintaining good separation of concerns between processing, event publishing, and UI updates.
