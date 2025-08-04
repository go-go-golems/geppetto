# Engine Middleware Design Proposal

**Date:** 2025-08-03  
**Author:** AI Assistant  
**Status:** Draft Design - Oracle Review Pending  

## Executive Summary

This proposal designs a middleware system for the Engine interface that enables intercepting and modifying input messages and output messages while keeping the core RunInference API clean. The middleware pattern provides extensibility for tool calling, transformations, caching, and other cross-cutting concerns without bloating the Engine interface.

## Design Goals

### Core Principles
1. **Clean Engine Interface**: Keep RunInference signature simple and focused
2. **Composable Middleware**: Support multiple middleware with clear execution order
3. **Event System Integration**: Preserve existing EventSink streaming behavior
4. **Zero Performance Overhead**: Optional middleware with no cost when unused
5. **Type Safety**: Strong typing with compile-time validation

### Use Cases to Support
- **Tool Calling**: Function calling workflows for OpenAI/Claude
- **Input Transformation**: Modify/validate messages before inference
- **Output Processing**: Transform/validate responses after inference  
- **Caching**: Cache responses based on input patterns
- **Logging/Monitoring**: Cross-cutting observability concerns
- **Content Filtering**: Input/output safety and compliance
- **Rate Limiting**: Request throttling and quota management

## Current State Analysis

### Existing Engine Interface
```go
type Engine interface {
    RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error)
}
```

### Existing Configuration System
```go
type Config struct {
    EventSinks []EventSink
}

func WithSink(sink EventSink) Option
```

### Current Event Flow
```
Engine.RunInference() → EventSink.PublishEvent() → UI/Monitoring
```

## Middleware Architecture Design

### 1. Core Middleware Interface (Functional Design)

Based on Oracle feedback, we use a functional approach similar to Go's HTTP middleware pattern:

```go
// HandlerFunc represents a function that can process an inference request.
// This is the core abstraction for both engines and middleware.
// It returns the complete conversation including any intermediate messages added during processing.
type HandlerFunc func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error)

// Middleware is a function that wraps a HandlerFunc with additional functionality.
// It receives a HandlerFunc and returns a new HandlerFunc that includes the middleware logic.
type Middleware func(HandlerFunc) HandlerFunc

// Chain composes multiple middleware into a single HandlerFunc.
// Middleware are applied in order: Chain(m1, m2, m3) results in m1(m2(m3(handler))).
func Chain(handler HandlerFunc, middlewares ...Middleware) HandlerFunc {
    // Apply middlewares in reverse order so they execute in correct order
    for i := len(middlewares) - 1; i >= 0; i-- {
        handler = middlewares[i](handler)
    }
    return handler
}

// EngineHandler adapts an Engine to HandlerFunc interface.
func EngineHandler(engine Engine) HandlerFunc {
    return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
        response, err := engine.RunInference(ctx, messages)
        if err != nil {
            return nil, err
        }
        // Return the original conversation plus the new AI response
        return append(messages, response), nil
    }
}
```

### 2. Engine Wrapper for Middleware Support

```go
// EngineWithMiddleware wraps an Engine with a middleware chain.
// It implements the Engine interface while providing middleware functionality.
type EngineWithMiddleware struct {
    handler HandlerFunc
    config  *Config
}

// NewEngineWithMiddleware creates a new engine with middleware support.
func NewEngineWithMiddleware(engine Engine, middlewares ...Middleware) *EngineWithMiddleware {
    // Convert engine to HandlerFunc and apply middleware chain
    handler := EngineHandler(engine)
    chainedHandler := Chain(handler, middlewares...)
    
    return &EngineWithMiddleware{
        handler: chainedHandler,
        config:  NewConfig(),
    }
}

// RunInference executes the middleware chain followed by the underlying engine.
// Returns only the final AI response message for compatibility with Engine interface.
func (e *EngineWithMiddleware) RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error) {
    // Add EventSinks to context for middleware access
    ctx = events.WithSinks(ctx, e.config.EventSinks)
    
    // Clone messages to prevent mutation issues (defensive copy)
    messages = conversation.Clone(messages)
    
    // Execute middleware chain and get complete conversation
    resultConversation, err := e.handler(ctx, messages)
    if err != nil {
        return nil, err
    }
    
    // Return the last message (the final AI response)
    if len(resultConversation) == 0 {
        return nil, fmt.Errorf("middleware returned empty conversation")
    }
    
    return resultConversation[len(resultConversation)-1], nil
}

// RunInferenceWithHistory executes the middleware chain and returns the complete conversation.
// This is useful when you want to see all intermediate messages from tool calling, etc.
func (e *EngineWithMiddleware) RunInferenceWithHistory(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
    // Add EventSinks to context for middleware access
    ctx = events.WithSinks(ctx, e.config.EventSinks)
    
    // Clone messages to prevent mutation issues (defensive copy)
    messages = conversation.Clone(messages)
    
    return e.handler(ctx, messages)
}
```

### 3. Configuration Integration

```go
// WithMiddleware adds middleware to the engine configuration.
func WithMiddleware(middleware Middleware) Option {
    return func(c *Config) error {
        if c.Middlewares == nil {
            c.Middlewares = make([]Middleware, 0)
        }
        c.Middlewares = append(c.Middlewares, middleware)
        return nil
    }
}

// Updated Config to support middleware
type Config struct {
    EventSinks  []EventSink
    Middlewares []Middleware  // New field for middleware chain
}
```

### 4. Factory Integration

```go
// EngineFactory updated to support middleware
func (f *StandardEngineFactory) CreateEngine(settings *settings.StepSettings, options ...Option) (Engine, error) {
    config := NewConfig()
    if err := ApplyOptions(config, options...); err != nil {
        return nil, err
    }

    // Create base engine (OpenAI, Claude, etc.)
    baseEngine, err := f.createBaseEngine(settings, config)
    if err != nil {
        return nil, err
    }

    // Wrap with middleware if any are configured
    if len(config.Middlewares) > 0 {
        return NewEngineWithMiddleware(baseEngine, config.Middlewares...), nil
    }

    return baseEngine, nil
}
```

## Middleware Implementation Examples

### 1. Tool Calling Middleware

```go
// ToolMiddleware handles function calling workflows for OpenAI/Claude.
func NewToolMiddleware(toolbox Toolbox, config ToolConfig) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
            return executeToolWorkflow(ctx, messages, toolbox, config, next)
        }
    }
}

func executeToolWorkflow(
    ctx context.Context,
    messages conversation.Conversation,
    toolbox Toolbox,
    config ToolConfig,
    next HandlerFunc,
) (conversation.Conversation, error) {
    // Prevent infinite tool calling loops
    iterations := 0
    currentMessages := messages
    
    for iterations < config.MaxIterations {
        // Add tool descriptions to the conversation if needed
        enrichedMessages := addToolContext(currentMessages, toolbox)
        
        // Execute inference with tools available
        resultConversation, err := next(ctx, enrichedMessages)
        if err != nil {
            return nil, fmt.Errorf("tool inference failed: %w", err)
        }
        
        // Get the last message (AI response) from the result
        if len(resultConversation) == 0 {
            return nil, fmt.Errorf("empty conversation returned from inference")
        }
        aiResponse := resultConversation[len(resultConversation)-1]
        
        // Check if response contains tool calls
        toolCalls := extractToolCalls(aiResponse)
        if len(toolCalls) == 0 {
            // No more tool calls, return complete conversation
            return resultConversation, nil
        }
        
        // Publish tool calling event
        events.Dispatch(ctx, events.NewToolCallEvent(toolCalls))
        
        // Execute all tool calls
        toolResults, err := executeToolCalls(ctx, toolCalls, toolbox)
        if err != nil {
            return nil, fmt.Errorf("tool execution failed: %w", err)
        }
        
        // Update conversation with AI response and tool results
        currentMessages = resultConversation
        for _, result := range toolResults {
            currentMessages = append(currentMessages, result.ToMessage())
        }
        
        iterations++
    }
    
    return nil, fmt.Errorf("tool calling exceeded maximum iterations (%d)", config.MaxIterations)
}

func executeToolCalls(ctx context.Context, toolCalls []ToolCall, toolbox Toolbox) ([]ToolResult, error) {
    results := make([]ToolResult, len(toolCalls))
    
    for i, call := range toolCalls {
        // Respect context timeout for each tool call
        childCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
        
        result, err := toolbox.ExecuteTool(childCtx, call.Name, call.Arguments)
        if err != nil {
            results[i] = ToolResult{Error: err.Error()}
            events.Dispatch(ctx, events.NewToolErrorEvent(call.Name, err))
        } else {
            results[i] = ToolResult{Content: result}
            events.Dispatch(ctx, events.NewToolResultEvent(call.Name, result))
        }
    }
    
    return results, nil
}
```

### 2. Transformation Middleware

```go
// NewTransformMiddleware applies transformations to input messages and/or final output message.
func NewTransformMiddleware(inputTransformer, outputTransformer MessageTransformer) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
            // Transform input messages
            transformedMessages := messages
            if inputTransformer != nil {
                transformedMessages = make(conversation.Conversation, len(messages))
                for i, msg := range messages {
                    transformed, err := inputTransformer.Transform(msg)
                    if err != nil {
                        return nil, fmt.Errorf("input transformation failed: %w", err)
                    }
                    transformedMessages[i] = transformed
                }
            }
            
            // Execute inference
            resultConversation, err := next(ctx, transformedMessages)
            if err != nil {
                return nil, err
            }
            
            // Transform final output message if transformer provided
            if outputTransformer != nil && len(resultConversation) > 0 {
                lastIndex := len(resultConversation) - 1
                lastMessage := resultConversation[lastIndex]
                
                transformed, err := outputTransformer.Transform(lastMessage)
                if err != nil {
                    return nil, fmt.Errorf("output transformation failed: %w", err)
                }
                
                // Replace last message with transformed version
                resultConversation[lastIndex] = transformed
            }
            
            return resultConversation, nil
        }
    }
}

type MessageTransformer interface {
    Transform(message *conversation.Message) (*conversation.Message, error)
}

// Example: Uppercase transformer (replacing the Step-based example)
type UppercaseTransformer struct{}

func (t *UppercaseTransformer) Transform(message *conversation.Message) (*conversation.Message, error) {
    if content, ok := message.Content.(*conversation.ChatMessageContent); ok {
        uppercased := &conversation.ChatMessageContent{
            Text: strings.ToUpper(content.Text),
        }
        return &conversation.Message{
            Role:    message.Role,
            Content: uppercased,
        }, nil
    }
    return message, nil
}
```

### 3. Caching Middleware

```go
// CacheMiddleware provides response caching based on input message hashes.
type CacheMiddleware struct {
    cache Cache
    ttl   time.Duration
}

func (m *CacheMiddleware) RunInference(ctx context.Context, messages conversation.Conversation, next Handler) (*conversation.Message, error) {
    // Generate cache key from messages
    cacheKey := m.generateCacheKey(messages)
    
    // Check cache
    if cached, found := m.cache.Get(cacheKey); found {
        return cached.(*conversation.Message), nil
    }
    
    // Execute inference
    response, err := next.RunInference(ctx, messages)
    if err != nil {
        return nil, err
    }
    
    // Cache successful response
    m.cache.Set(cacheKey, response, m.ttl)
    
    return response, nil
}
```

### 4. Logging Middleware

```go
// LoggingMiddleware provides request/response logging and metrics.
type LoggingMiddleware struct {
    logger  Logger
    metrics Metrics
}

func (m *LoggingMiddleware) RunInference(ctx context.Context, messages conversation.Conversation, next Handler) (*conversation.Message, error) {
    start := time.Now()
    
    // Log request
    m.logger.Info("inference_started", map[string]interface{}{
        "message_count": len(messages),
        "timestamp":     start,
    })
    
    // Execute inference
    response, err := next.RunInference(ctx, messages)
    
    duration := time.Since(start)
    
    // Log response
    if err != nil {
        m.logger.Error("inference_failed", map[string]interface{}{
            "error":    err.Error(),
            "duration": duration,
        })
        m.metrics.IncrementCounter("inference_errors")
    } else {
        m.logger.Info("inference_completed", map[string]interface{}{
            "duration":      duration,
            "response_size": len(response.Content.View()),
        })
        m.metrics.RecordDuration("inference_duration", duration)
    }
    
    return response, nil
}
```

## Event System Integration

### Middleware Event Publishing

Middleware can participate in the event system by implementing EventSink or by publishing custom events:

```go
// Middleware with event publishing capability
type EventAwareMiddleware struct {
    eventSinks []EventSink
}

func (m *EventAwareMiddleware) RunInference(ctx context.Context, messages conversation.Conversation, next Handler) (*conversation.Message, error) {
    // Publish custom middleware events
    m.publishEvent(events.NewCustomEvent("middleware_started", metadata))
    
    response, err := next.RunInference(ctx, messages)
    
    if err != nil {
        m.publishEvent(events.NewErrorEvent("middleware_error", metadata, err))
    } else {
        m.publishEvent(events.NewCustomEvent("middleware_completed", metadata))
    }
    
    return response, err
}

func (m *EventAwareMiddleware) publishEvent(event events.Event) {
    for _, sink := range m.eventSinks {
        _ = sink.PublishEvent(event) // Fire and forget
    }
}
```

## Usage Examples

### Basic Middleware Usage

```go
// Create base engine
factory := inference.NewStandardEngineFactory()
baseEngine, err := factory.CreateEngine(settings)

// Create middleware chain
toolMiddleware := NewToolMiddleware(toolbox, toolConfig)
loggingMiddleware := NewLoggingMiddleware(logger, metrics)
cacheMiddleware := NewCacheMiddleware(cache, 5*time.Minute)

// Wrap engine with middleware
engine := NewEngineWithMiddleware(
    baseEngine,
    loggingMiddleware,    // Executes first (outermost)
    cacheMiddleware,     // Executes second
    toolMiddleware,      // Executes third (innermost)
)

// Use like any engine - returns only final AI response
response, err := engine.RunInference(ctx, messages)

// Or get complete conversation history including tool calls
fullConversation, err := engine.RunInferenceWithHistory(ctx, messages)
// fullConversation contains: original messages + AI tool call + tool results + final AI response
```

### Factory Integration

```go
// Using middleware through factory options
engine, err := factory.CreateEngine(
    settings,
    inference.WithSink(uiSink),
    inference.WithMiddleware(NewToolMiddleware(toolbox, toolConfig)),
    inference.WithMiddleware(NewLoggingMiddleware(logger, metrics)),
)
```

### Tool Calling Example

```go
// Define tools
toolbox := toolbox.New()
toolbox.RegisterTool("weather", weatherTool)
toolbox.RegisterTool("calculator", calculatorTool)

// Create tool middleware
toolMiddleware := NewToolMiddleware(toolbox, ToolConfig{
    MaxIterations: 5,
    Timeout:      30 * time.Second,
})

// Create engine with tool support
engine := NewEngineWithMiddleware(baseEngine, toolMiddleware)

// Use with tool-capable messages
messages := conversation.New(
    conversation.NewUserMessage("What's the weather in San Francisco and what's 25 * 47?"),
)

// Get just the final response
response, err := engine.RunInference(ctx, messages)
// response contains: "The weather in San Francisco is 72°F and sunny. 25 * 47 = 1,175."

// Or get the complete conversation history
fullConversation, err := engine.RunInferenceWithHistory(ctx, messages)
// fullConversation contains:
// 1. User: "What's the weather in San Francisco and what's 25 * 47?"
// 2. AI: [tool calls for weather and calculator]
// 3. Tool result: weather data
// 4. Tool result: calculation result
// 5. AI: "The weather in San Francisco is 72°F and sunny. 25 * 47 = 1,175."
```

## Oracle Architectural Review Summary

The Oracle provided comprehensive feedback that led to significant design improvements:

### Key Oracle Recommendations Implemented

1. **Functional Middleware Pattern**: Switched from interface-based to function-based middleware (like Go HTTP) for zero-allocation composition and better ergonomics.

2. **Context-Based Event Integration**: EventSinks are now embedded in context via `events.WithSinks(ctx, sinks)` rather than manually plumbed through middleware structs.

3. **Defensive Message Cloning**: Messages are cloned before middleware execution to prevent mutation issues between middleware.

4. **Timeout Handling**: Tool calling respects context timeouts with `context.WithTimeout()` for each tool execution.

5. **Infinite Loop Prevention**: Tool middleware enforces `MaxIterations` limit to prevent infinite tool calling loops.

6. **Thread Safety**: Middleware implementations are required to be stateless or properly guarded for concurrent use.

### Critical Design Decisions

**Streaming Consideration**: The current design handles request/response interception but not token-stream interception. For real-time streaming requirements, we would need to:
- Propagate token streams through middleware as `io.Reader` or channels, or  
- Expose separate `StreamEngine` interface, or
- Broker everything through EventSink with stronger guarantees

**Decided**: Keep current design focused on message-level interception. Streaming events continue through existing EventSink system.

### Security and Reliability Features

1. **Error Wrapping**: Middleware errors are properly wrapped with context using `fmt.Errorf("context: %w", err)`
2. **Panic Recovery**: Can be added as outermost middleware for safety
3. **Resource Management**: Context timeouts prevent runaway tool executions
4. **Observability**: Rich event publishing for monitoring and debugging

## Performance Considerations

### Zero-Cost Abstraction
- Function composition has zero allocation overhead
- No reflection or dynamic dispatch during inference  
- Function calls are inlined by Go compiler when possible
- Memory allocation is minimal (function closures only)

### Streaming Preservation
- Middleware executes before and after inference, not during streaming
- EventSink system continues to handle streaming events normally
- No impact on streaming performance

### Memory Management
- Middleware should avoid retaining references to large messages
- Handler chain is lightweight (function pointers + interfaces)
- Caching middleware manages its own memory lifecycle

## Testing Strategy

### Unit Testing
```go
func TestToolMiddleware(t *testing.T) {
    // Create mock engine and toolbox
    mockEngine := &MockEngine{}
    mockToolbox := &MockToolbox{}
    
    // Create middleware
    middleware := NewToolMiddleware(mockToolbox, ToolConfig{})
    
    // Create handler
    handler := &engineHandler{engine: mockEngine}
    
    // Test tool calling workflow
    messages := conversation.New(
        conversation.NewUserMessage("Use the calculator tool"),
    )
    
    response, err := middleware.RunInference(ctx, messages, handler)
    
    assert.NoError(t, err)
    assert.Contains(t, response.Content.View(), "calculated result")
}
```

### Integration Testing
```go
func TestMiddlewareChain(t *testing.T) {
    // Create real engine
    engine := createTestEngine()
    
    // Create middleware chain
    chain := NewEngineWithMiddleware(
        engine,
        NewLoggingMiddleware(testLogger),
        NewToolMiddleware(testToolbox, ToolConfig{}),
    )
    
    // Test complete workflow
    response, err := chain.RunInference(ctx, testMessages)
    
    assert.NoError(t, err)
    assert.NotNil(t, response)
}
```

## Migration Path

### Phase 1: Core Middleware Infrastructure
1. Implement basic Middleware interface and EngineWithMiddleware wrapper
2. Add middleware support to Config and Options
3. Update EngineFactory to handle middleware options
4. Create comprehensive unit tests

### Phase 2: Essential Middleware Implementations
1. Implement ToolMiddleware for OpenAI/Claude tool calling
2. Create LoggingMiddleware for observability
3. Implement basic TransformMiddleware
4. Add integration tests

### Phase 3: Advanced Middleware
1. Implement CacheMiddleware with various cache backends
2. Create ContentFilterMiddleware for safety
3. Add RateLimitingMiddleware for quota management
4. Performance testing and optimization

### Phase 4: Migration and Cleanup
1. Migrate existing Step-based tool calling to ToolMiddleware
2. Replace transformation Steps with TransformMiddleware
3. Update documentation and examples
4. Remove deprecated Step-based implementations

## Benefits and Trade-offs

### Benefits
1. **Clean Architecture**: Core Engine interface remains simple and focused
2. **Composability**: Multiple middleware can be combined in any order
3. **Extensibility**: New functionality added without changing Engine interface
4. **Reusability**: Middleware components are reusable across different engines
5. **Testability**: Each middleware is independently testable
6. **Performance**: Zero overhead when middleware not used

### Trade-offs
1. **Complexity**: Additional abstraction layer to understand
2. **Debugging**: Error traces may span multiple middleware layers
3. **Configuration**: More complex setup for advanced use cases
4. **Memory**: Slight memory overhead for handler chain

## Alternative Patterns Considered

### 1. Engine Interface Extension
```go
type EngineWithTools interface {
    Engine
    RunInferenceWithTools(ctx context.Context, messages conversation.Conversation, tools []Tool) (*conversation.Message, error)
}
```
**Rejected**: Bloats Engine interface, not composable

### 2. Option-Based Configuration
```go
func (e *Engine) RunInference(ctx context.Context, messages conversation.Conversation, options ...InferenceOption) (*conversation.Message, error)
```
**Rejected**: Changes core API signature, less type-safe

### 3. Event-Based Hooks
```go
type Engine interface {
    OnBeforeInference(hook func(messages conversation.Conversation) conversation.Conversation)
    OnAfterInference(hook func(response *conversation.Message) *conversation.Message)
    RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error)
}
```
**Rejected**: Limited composability, stateful Engine interface

## Future Extensions

### Advanced Middleware Patterns
1. **Conditional Middleware**: Execute based on message content or context
2. **Parallel Middleware**: Execute multiple middleware concurrently
3. **Retry Middleware**: Automatic retry with exponential backoff
4. **Circuit Breaker**: Fail-fast for degraded services

### Provider-Specific Middleware
1. **OpenAI Middleware**: Function calling, response formatting
2. **Claude Middleware**: Tool use patterns, safety filtering
3. **Gemini Middleware**: Multimodal handling, grounding

### Observability Enhancements
1. **Tracing Middleware**: Distributed tracing integration
2. **Metrics Middleware**: Detailed performance metrics
3. **Audit Middleware**: Compliance and security logging

## Conclusion

The middleware pattern provides a clean, extensible architecture for the Engine interface that addresses the key requirements identified in the Step API analysis:

1. **Tool Calling**: ToolMiddleware preserves complex tool calling workflows
2. **Transformations**: TransformMiddleware replaces Step-based transformations
3. **Cross-cutting Concerns**: Logging, caching, filtering handled cleanly
4. **Composability**: Multiple middleware combine naturally
5. **Performance**: Zero overhead when not used, minimal overhead when used

This design keeps the Engine interface clean while providing the extensibility needed to replace all Step API functionality. The pattern is well-established, testable, and provides a clear migration path from the existing Step-based architecture.

---

## Next Steps

1. **Oracle Review**: Validate architectural approach and identify potential issues
2. **Prototype Implementation**: Create working middleware system
3. **Tool Calling Migration**: Implement ToolMiddleware for OpenAI/Claude
4. **Integration Testing**: Validate with existing pinocchio usage
5. **Documentation**: Create migration guides and examples

This middleware design positions the Engine interface for long-term extensibility while maintaining the simplicity that makes the Engine-first architecture successful.
