# Steps in Geppetto

Steps are the fundamental building blocks of data processing pipelines in Geppetto. They represent asynchronous transformations that convert input of type T into a stream of outputs of type U.

## Core Concepts

### Step Interface

A Step is defined by its ability to:
- Transform input of type T into outputs of type U asynchronously
- Stream results through a StepResult monad
- Publish events to Watermill topics during processing
- Carry metadata about the transformation

The basic Step interface looks like this:

```go
type Step interface {
    Start(ctx context.Context, input T) (StepResult[U], error)
    AddPublishedTopic(publisher message.Publisher, topic string) error
}
```

### StepResult Monad

StepResult is a powerful abstraction that combines several aspects of asynchronous processing:

- Error handling through `helpers.Result`
- Streaming through Go channels
- Multiple value outputs (either streamed or batched)
- Cancellation support
- Rich metadata handling

Creating a StepResult:
```go
ret := steps.NewStepResult[string](
    channel,
    steps.WithCancel[string](cancelFunc),
    steps.WithMetadata[string](stepMetadata),
)
```

### Step Metadata

Each step carries metadata that includes:
- Step ID (UUID)
- Step Type (string identifier for the step type)
- Input Type (type information for input)
- Output Type (type information for output)
- Custom key-value metadata pairs

Example metadata structure:
```go
stepMetadata := &steps.StepMetadata{
    StepID:     uuid.New(),
    Type:       "chat",
    InputType:  "conversation.Conversation",
    OutputType: "string",
    Metadata: map[string]interface{}{
        steps.MetadataSettingsSlug: settings.GetMetadata(),
    },
}
```

## Event Publishing

Steps can publish events during their execution using Watermill's message.Publisher interface. This is typically implemented using a PublisherManager that registers with specific topics.

### Publisher Management

```go
type Step struct {
    subscriptionManager *events2.PublisherManager
}

// Register a new publisher for a topic
func (s *Step) AddPublishedTopic(publisher message.Publisher, topic string) error {
    s.subscriptionManager.RegisterPublisher(topic, publisher)
    return nil
}
```

### Chat Events

For LLM chat interactions, the following event types are supported:

```go
EventTypeStart             // Initial event
EventTypeFinal            // Final completion
EventTypePartialCompletion // Intermediate streaming results
EventTypeStatus           // Status updates
EventTypeToolCall         // Tool invocation
EventTypeToolResult       // Tool execution results
EventTypeError           // Error conditions
EventTypeInterrupt       // Processing interruption
```

## Implementation Example: Chat Step

Here's a detailed example of implementing a chat step:

```go
type ChatStep struct {
    Settings            *settings.StepSettings
    Tools               []Tool
    subscriptionManager *events2.PublisherManager
    parentID           conversation.NodeID
    messageID          conversation.NodeID
}

func (cs *ChatStep) Start(ctx context.Context, messages conversation.Conversation) (steps.StepResult[string], error) {
    // 1. Set up cancellation context
    cancellableCtx, cancel := context.WithCancel(ctx)
    
    // 2. Create event channel and metadata
    eventCh := make(chan Event)
    metadata := createEventMetadata()
    stepMetadata := createStepMetadata()
    
    // 3. Set up result channel
    c := make(chan helpers2.Result[string])
    ret := steps.NewStepResult[string](
        c,
        steps.WithCancel[string](cancel),
        steps.WithMetadata[string](stepMetadata),
    )
    
    // 4. Start processing goroutine
    go func() {
        defer close(c)
        defer cancel()
        
        for {
            select {
            case <-cancellableCtx.Done():
                handleCancellation()
                return
                
            case event, ok := <-eventCh:
                if !ok {
                    handleCompletion()
                    return
                }
                
                // Process event and publish updates
                processAndPublishEvent(event)
            }
        }
    }()
    
    return ret, nil
}
```

## Usage in Pipelines

Steps can be:
- Chained together to form processing pipelines
- Run independently for single transformations
- Monitored through their event streams
- Cancelled using context
- Factory-created using StepFactory thunks

### Creating a Pipeline

```go
func CreatePipeline(ctx context.Context) error {
    // Create steps
    step1 := NewStep1()
    step2 := NewStep2()
    
    // Configure event publishing
    publisher := createWatermillPublisher()
    step1.AddPublishedTopic(publisher, "step1-events")
    step2.AddPublishedTopic(publisher, "step2-events")
    
    // Start first step
    result1, err := step1.Start(ctx, input)
    if err != nil {
        return err
    }
    
    // Process results and feed to next step
    for res := range result1.Channel() {
        if res.Error() != nil {
            return res.Error()
        }
        
        // Start second step with result from first
        result2, err := step2.Start(ctx, res.Value())
        if err != nil {
            return err
        }
        
        // Process final results
        processResults(result2)
    }
    
    return nil
}
```

## Best Practices

1. Always handle both success and error paths
   ```go
   if res.Error() != nil {
       publishErrorEvent(res.Error())
       return res.Error()
   }
   ```

2. Use appropriate event types for different processing stages
   ```go
   subscriptionManager.PublishBlind(chat.NewStartEvent(metadata, stepMetadata))
   subscriptionManager.PublishBlind(chat.NewPartialEvent(metadata, stepMetadata, partial))
   subscriptionManager.PublishBlind(chat.NewFinalEvent(metadata, stepMetadata, final))
   ```

3. Implement proper cleanup in case of cancellation
   ```go
   defer close(resultChannel)
   defer cancel()
   select {
   case <-ctx.Done():
       publishInterruptEvent()
       return
   }
   ```

4. Include relevant metadata for debugging and monitoring
   ```go
   metadata := map[string]interface{}{
       "step_id": uuid.New(),
       "timestamp": time.Now(),
       "input_size": len(input),
   }
   ```

5. Consider streaming performance when implementing long-running steps
   - Use buffered channels when appropriate
   - Implement backpressure mechanisms
   - Monitor memory usage in streaming operations

## Error Handling

Steps should handle errors at multiple levels:

1. Initialization errors
```go
if settings == nil {
    return nil, steps.ErrMissingSettings
}
```

2. Runtime errors
```go
if err := processChunk(data); err != nil {
    subscriptionManager.PublishBlind(chat.NewErrorEvent(metadata, stepMetadata, err.Error()))
    return helpers2.NewErrorResult[string](err)
}
```

3. Cleanup errors
```go
defer func() {
    if err := cleanup(); err != nil {
        log.Error().Err(err).Msg("cleanup failed")
    }
}()
```

## See Also

- `geppetto/pkg/steps/step.go` - Core Step definitions
- `geppetto/pkg/steps/ai/chat/events.go` - Chat event types
- `geppetto/pkg/events/publish.go` - Publisher management
- `geppetto/pkg/steps/ai/claude/chat-step.go` - Example implementation of a chat step