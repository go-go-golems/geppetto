# Step Adapter - Engine-first Architecture Phase 1

This package implements Phase 1 of the Engine-first architecture proposal by providing the `StepAdapter` that enables backwards compatibility for the existing Step API while using new inference engines under the hood.

## Overview

The `StepAdapter` acts as a bridge between the existing complex Step interface and the new simplified Engine interface. This allows:

1. **Zero API Changes**: Existing code using `step.Start(ctx, messages)` continues working without modification
2. **Engine Architecture**: Under the hood, the new inference engine system is used
3. **Event Preservation**: All watermill-based event publishing continues working identically
4. **Streaming Support**: Both streaming and non-streaming modes are preserved

## Key Components

### StepAdapter

The main adapter that wraps an `inference.Engine` and implements the `chat.Step` interface:

```go
type StepAdapter struct {
    engine           inference.Engine
    publisherManager *events.PublisherManager
    metadata         *steps.StepMetadata
}
```

**Key Features:**
- Implements the complete `chat.Step` interface
- Handles `AddPublishedTopic(publisher, topic)` method for event publishing
- Converts `Start(ctx, messages)` to use `RunInference` internally
- Supports both streaming and non-streaming modes
- Preserves exact behavior of existing ChatStep implementations

### EngineStepFactory

A new factory that creates engines and wraps them in adapters:

```go
type EngineStepFactory struct {
    Settings *settings.StepSettings
}
```

**Benefits:**
- Creates the same step instances as before (OpenAI, Claude, Gemini)
- Wraps them in adapters to provide the engine interface
- Preserves all existing step options and configurations
- Maintains caching and other step decorators

### Updated StandardStepFactory

The existing `StandardStepFactory` has been updated to delegate to `EngineStepFactory`:

```go
func (s *StandardStepFactory) NewStep(options ...chat.StepOption) (chat.Step, error) {
    engineFactory := &adapter.EngineStepFactory{Settings: s.Settings}
    return engineFactory.NewStep(options...)
}
```

## Behavior Preservation

### Channel-based Result Handling

The adapter preserves the existing channel-based result system:

- `StepResult[*conversation.Message]` interface is maintained
- `result.GetChannel()` returns streaming results
- `result.Return()` blocks and returns all results
- `result.Cancel()` properly cancels ongoing operations

### Event Publishing

The watermill-based event system continues working identically:

- Start events are published when processing begins
- Partial completion events for streaming results
- Final events with complete responses
- Error and interrupt events for failure cases
- All events include the same metadata as before

### Streaming vs Non-streaming

The adapter detects and handles both modes:

- **Non-streaming**: Direct `RunInference` call, immediate result
- **Streaming**: Background goroutine with channel-based result delivery
- Existing streaming implementations (like OpenAI ChatStep) are delegated to directly

## Usage Examples

### Existing Code (No Changes Required)

```go
// This code continues working without any modifications
factory := &ai.StandardStepFactory{Settings: settings}
step, err := factory.NewStep()
if err != nil {
    return err
}

result, err := step.Start(ctx, messages)
if err != nil {
    return err
}

for messageResult := range result.GetChannel() {
    if messageResult.Error() != nil {
        log.Printf("Error: %v", messageResult.Error())
        continue
    }
    message := messageResult.Unwrap()
    // Process message as before
}
```

### New Engine Interface Access

The adapter also enables access to the simplified engine interface:

```go
if simpleChatStep, ok := step.(chat.SimpleChatStep); ok {
    // Can use the simplified interface directly
    message, err := simpleChatStep.RunInference(ctx, messages)
    // Handle single message response
}
```

### Creating Engine from Step

You can also create engines from existing steps:

```go
engine := adapter.CreateEngineFromStep(existingStep)
message, err := engine.RunInference(ctx, messages)
```

## Implementation Details

### Adapter Logic

The adapter uses a smart delegation strategy:

1. **Check for existing Step interface**: If the engine already implements `chat.Step`, delegate directly after copying publisher configuration
2. **SimpleChatStep wrapping**: For engines that implement `SimpleChatStep`, wrap in the channel-based system with proper event publishing
3. **Basic Engine wrapping**: For basic engines, provide a simple non-streaming wrapper

### Event Publishing Integration

The adapter integrates seamlessly with the existing event system:

- Creates `PublisherManager` instances for watermill integration
- Publishes all the same event types as existing implementations
- Preserves event metadata and sequencing
- Handles cancellation and error events properly

### Error Handling

The adapter preserves all existing error handling patterns:

- Context cancellation is properly propagated
- Errors are wrapped and published as events
- Streaming interruption is handled gracefully
- All error types are preserved from the underlying implementations

## Testing

The adapter includes comprehensive tests:

- `TestStepAdapter_BasicEngine`: Tests basic engine wrapping
- `TestStepAdapter_SimpleChatStep`: Tests SimpleChatStep delegation
- `TestStepAdapter_AddPublishedTopic`: Tests event publishing integration
- `TestStepEngineAdapter`: Tests the reverse adapter (step to engine)

All existing tests in the geppetto codebase continue to pass without modification.

## Migration Path

This implementation provides a smooth migration path:

1. **Phase 1 (Current)**: Existing Step API preserved, engines used internally
2. **Phase 2 (Future)**: Gradual migration to engine-first APIs
3. **Phase 3 (Future)**: Deprecation of complex Step interface in favor of engines

The adapter ensures that no existing functionality is lost during this transition while enabling the benefits of the new engine architecture.
