# Architecture Proposal: Engine-First, Step-Wrapper Design

**Date:** 2025-08-03  
**Author:** Oracle (AI Assistant)  
**Status:** Proposal - Ready for Implementation  

## Executive Summary

This proposal introduces an "Engine-first, Step-wrapper" architecture that replaces the complex Step-based API with a simpler `RunInference` primitive while preserving all watermill event-driven capabilities for pinocchio's UI and streaming features.

**Core Philosophy:** Make `RunInference` the ONLY primitive that actually talks to LLMs. Everything that needs channels or Watermill (UI, old Step API) becomes a thin wrapper around that primitive.

## 1. Core Architecture Overview

### Key Principles
1. **Engine-First Design:** `RunInference` becomes the fundamental primitive
2. **Pluggable Event System:** Event publishing pulled out into configurable `EventSink`
3. **Backwards Compatibility:** Existing Step API becomes a thin adapter
4. **Optional Complexity:** Simple use cases get simple APIs, complex cases retain full power

### Component Hierarchy
```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │ Blocking Mode   │  │ Interactive     │  │ Chat UI      │ │
│  │ (EngineFactory) │  │ (Factory+Events)│  │ (Adapter)    │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                   Compatibility Layer                       │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              StepAdapter (Backwards Compat)             │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                      Core Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │ EngineFactory│  │  EventSink   │  │  Watermill       │   │
│  │     │        │  │ (Null/       │  │  Integration     │   │
│  │     ▼        │  │  Watermill/  │  │                  │   │
│  │   Engine     │  │  Buffer)     │  │                  │   │
│  │ (OpenAI/     │  │              │  │                  │   │
│  │  Claude/     │  │              │  │                  │   │
│  │  Gemini)     │  │              │  │                  │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 2. New Core Interfaces

### 2.1 Engine Interface
```go
package inference // new package in geppetto

// Engine is the fundamental LLM inference primitive
type Engine interface {
    // RunInference executes the request and MAY stream events through the
    // supplied sink. Returns the *final* assistant message.
    RunInference(
        ctx context.Context,
        conversation []*conversation.Message,
        opts ...Option,
    ) (*conversation.Message, error)
}
```

### 2.2 Engine Factory Interface
```go
// EngineFactory creates engines based on settings/configuration
// This allows external control over which provider engine is used
type EngineFactory interface {
    // CreateEngine creates an appropriate engine based on the provided settings
    // The factory determines which provider (OpenAI, Claude, Gemini, etc.) to use
    CreateEngine(settings *settings.StepSettings) (Engine, error)
    
    // SupportedProviders returns a list of AI providers this factory supports
    SupportedProviders() []string
    
    // DefaultProvider returns the default provider when none is specified
    DefaultProvider() string
}

// StandardEngineFactory is the default implementation
type StandardEngineFactory struct{}

func NewStandardEngineFactory() EngineFactory {
    return &StandardEngineFactory{}
}

func (f *StandardEngineFactory) CreateEngine(settings *settings.StepSettings) (Engine, error) {
    // Determine provider from settings.Chat.ApiType or similar
    switch settings.Chat.ApiType {
    case "openai":
        return NewOpenAIEngine(settings.OpenAI), nil
    case "claude", "anthropic":
        return NewClaudeEngine(settings.Claude), nil
    case "gemini":
        return NewGeminiEngine(settings.Gemini), nil
    default:
        // Fallback to OpenAI as default
        return NewOpenAIEngine(settings.OpenAI), nil
    }
}
```

### 2.3 EventSink Interface
```go
// EventSink receives events during inference execution
type EventSink interface {
    Send(events.Event) error
    Close() error
}

// Implementations:
// - NullSink (default, no-op)
// - WatermillSink(publisher, topic) 
// - BufferSink(callback func(Event))
```

### 2.4 Functional Options Pattern
```go
type Option func(*config)

func WithSink(s EventSink) Option
func WithTooling(toolbox.Toolbox) Option        // future proofing
func WithModelParams(p any) Option

// Internal configuration
type config struct {
    sink EventSink
    // model/temperature/system prompt etc.
}
```

## 3. Event Flow Architecture

### 3.1 New Event Flow with Engine
```
┌──────────────┐
│ Engine (OAI) │ ← reads OpenAI/Claude, builds events
└──────┬───────┘
       │ Send(e)
       ▼
┌──────────────────┐
│ WatermillSink    │ (implements inference.EventSink)
└──────┬───────────┘
       │ publisher.Publish(topic, eJSON)
       ▼
 Watermill Router → UI handler → Bubbletea view update
```

### 3.2 Step Adapter Event Flow
```
                     ( 1 )  Δ / final events
msgs → StepAdapter.Start ────────────────► sink ───────► publisher
      │                                      │
      │ (2) calls Engine.RunInference        │
      └──────────────────────────── result ◄─┘
                      │
                      ▼
                ResultQueue (final value)
```

## 4. Implementation Details

### 4.1 StepAdapter for Backwards Compatibility
```go
package stepsadapter // lives in geppetto/pkg/steps

// StepAdapter implements the existing chat.Step interface
type StepAdapter struct {
    engine     inference.Engine
    publisher  message.Publisher  // set via AddPublishedTopic
    topic      string
}

func New(eng inference.Engine, opts ...AdapterOption) *StepAdapter

func (sa *StepAdapter) AddPublishedTopic(p message.Publisher, t string) {
    sa.publisher = p
    sa.topic = t
}

func (sa *StepAdapter) Start(
    ctx context.Context,
    msgs []*conversation.Message,
) (steps.StepResult[*conversation.Message], error) {
    
    // Build sink if publisher configured
    var sink inference.EventSink = inference.NullSink{}
    if sa.publisher != nil {
        sink = inference.WatermillSink(sa.publisher, sa.topic)
    }

    // For streaming mode: run inference in goroutine, return channel
    if isStreamingMode(opts) {
        c := make(chan helpers.Result[*conversation.Message])
        go func() {
            defer close(c)
            msg, err := sa.engine.RunInference(ctx, msgs, inference.WithSink(sink))
            if err != nil {
                c <- helpers.NewErrorResult(err)
            } else {
                c <- helpers.NewValueResult(msg)
            }
        }()
        return steps.NewStepResult(c, ...), nil
    }
    
    // For non-streaming: direct call
    msg, err := sa.engine.RunInference(ctx, msgs, inference.WithSink(sink))
    return steps.Resolve(msg), err
}
```

### 4.2 Engine Implementation Examples

#### OpenAI Engine
```go
// engine_openai.go
type OpenAIEngine struct {
    client *openai.Client
    config *settings.OpenAISettings
}

func (o *OpenAIEngine) RunInference(
    ctx context.Context,
    msgs []*conversation.Message, 
    opts ...Option,
) (*conversation.Message, error) {
    
    cfg := defaultConfig()
    for _, fn := range opts { 
        fn(&cfg) 
    }
    
    sink := cfg.sink
    if sink == nil { 
        sink = NullSink{} 
    }

    // Convert messages to OpenAI format
    req := o.buildRequest(msgs)
    
    // Publish start event
    meta := events.NewMetadata(...)
    sink.Send(events.NewStartEvent(meta, nil))

    if o.config.Chat.Stream {
        return o.runStreaming(ctx, req, sink, meta)
    } else {
        return o.runNonStreaming(ctx, req, sink, meta)
    }
}

func (o *OpenAIEngine) runStreaming(
    ctx context.Context,
    req openai.ChatCompletionRequest,
    sink EventSink,
    meta events.Metadata,
) (*conversation.Message, error) {
    
    stream, err := o.client.CreateChatCompletionStream(ctx, req)
    if err != nil {
        sink.Send(events.NewErrorEvent(meta, nil, err))
        return nil, err
    }
    defer stream.Close()

    var completion strings.Builder
    for {
        delta, err := stream.Recv()
        if errors.Is(err, io.EOF) { 
            break 
        }
        if err != nil { 
            sink.Send(events.NewErrorEvent(meta, nil, err))
            return nil, err 
        }
        
        deltaText := delta.Choices[0].Delta.Content
        completion.WriteString(deltaText)
        
        // Send partial completion event
        sink.Send(events.NewPartialCompletionEvent(
            meta, nil, deltaText, completion.String(),
        ))
    }
    
    finalText := completion.String()
    sink.Send(events.NewFinalEvent(meta, nil, finalText))

    assistantMsg := &conversation.Message{
        Role: conversation.RoleAssistant,
        Content: conversation.NewChatMessageContent(finalText),
    }
    return assistantMsg, nil
}
```

#### EventSink Implementations
```go
// sink_watermill.go
type watermillSink struct {
    publisher message.Publisher
    topic     string
}

func WatermillSink(pub message.Publisher, topic string) EventSink {
    return &watermillSink{publisher: pub, topic: topic}
}

func (w *watermillSink) Send(e events.Event) error {
    payload, err := json.Marshal(e)
    if err != nil {
        return err
    }
    
    msg := message.NewMessage(uuid.NewString(), payload)
    msg.Metadata = encodeEventMetadata(e.Metadata())
    
    return w.publisher.Publish(w.topic, msg)
}

func (w *watermillSink) Close() error {
    return nil // Watermill publisher handles its own lifecycle
}

// sink_null.go
type NullSink struct{}

func (n NullSink) Send(e events.Event) error { return nil }
func (n NullSink) Close() error              { return nil }
```

## 5. Pinocchio Integration Patterns

### 5.1 Blocking Mode (Simplest - No Events)
```go
// In pinocchio/pkg/cmds/cmd.go - runBlocking()
func (g *PinocchioCommand) runBlocking(ctx context.Context, rc *run.RunContext) ([]*conversation.Message, error) {
    // Create engine factory
    factory := inference.NewStandardEngineFactory()
    
    // Create appropriate engine based on settings (external control)
    engine, err := factory.CreateEngine(rc.StepSettings)
    if err != nil {
        return nil, err
    }
    
    // Simple inference call - no events needed
    conversation := rc.ConversationManager.GetConversation()
    answer, err := engine.RunInference(ctx, conversation)
    if err != nil {
        return nil, err
    }
    
    // Append and return
    if err := rc.ConversationManager.AppendMessages(answer); err != nil {
        return nil, err
    }
    
    return rc.ConversationManager.GetConversation(), nil
}
```

### 5.2 Streaming UI Mode (Chat/Interactive)
```go
// In pinocchio/pkg/cmds/cmd.go - runChat()
func (g *PinocchioCommand) runChat(ctx context.Context, rc *run.RunContext) ([]*conversation.Message, error) {
    // Create router (unchanged)
    router := rc.Router
    
    // Create engine factory
    factory := inference.NewStandardEngineFactory()
    
    // Create appropriate engine based on settings (external control)
    engine, err := factory.CreateEngine(rc.StepSettings)
    if err != nil {
        return nil, err
    }
    
    // Create watermill sink for events
    sink := inference.WatermillSink(router.Publisher, "ui")
    
    // For UI backend, we can still use StepAdapter during migration
    step := stepsadapter.New(engine, stepsadapter.WithPublisher(router.Publisher, "ui"))
    backend := ui.NewStepBackend(step)
    
    // Or eventually, use engine directly:
    // backend := ui.NewEngineBackend(engine, sink)
    
    // Rest of UI setup unchanged...
}
```

### 5.3 Backwards Compatibility (Existing Code)
```go
// In existing Step factory functions
func (sf *StandardStepFactory) NewStep(options ...chat.StepOption) (chat.Step, error) {
    // Create engine factory
    engineFactory := inference.NewStandardEngineFactory()
    
    // Create appropriate engine based on step settings (external control)
    engine, err := engineFactory.CreateEngine(sf.Settings)
    if err != nil {
        return nil, err
    }
    
    // Wrap in adapter
    adapter := stepsadapter.New(engine)
    
    // Apply existing options (AddPublishedTopic, etc.)
    for _, opt := range options {
        opt(adapter)  // StepAdapter implements chat.Step interface
    }
    
    return adapter, nil
}
```

## 6. ChatRunner Integration

### 6.1 Updated ChatSession
```go
// In pinocchio/pkg/chatrunner/chat_runner.go
type ChatSession struct {
    ctx           context.Context
    engineFactory inference.EngineFactory        // Changed to use factory interface
    stepSettings  *settings.StepSettings         // Settings for engine creation
    manager       geppetto_conversation.Manager
    uiOptions     []bobachat.ModelOption
    // ... rest unchanged
}

func (cs *ChatSession) runBlockingInternal() error {
    // Create engine via factory - provider determined by settings
    engine, err := cs.engineFactory.CreateEngine(cs.stepSettings)
    if err != nil {
        return err
    }
    
    conversation := cs.manager.GetConversation()
    answer, err := engine.RunInference(cs.ctx, conversation)
    if err != nil {
        return err
    }
    
    if err := cs.manager.AppendMessages(answer); err != nil {
        return err
    }
    
    // Print output
    fmt.Fprintln(cs.outputWriter, answer.Content.View())
    return nil
}

func (cs *ChatSession) runChatInternal() error {
    router := cs.router
    // ... router setup unchanged

    // Create engine via factory - provider determined by settings  
    engine, err := cs.engineFactory.CreateEngine(cs.stepSettings)
    if err != nil {
        return err
    }
    
    // During migration: use adapter
    uiStep := stepsadapter.New(engine)
    uiStep.AddPublishedTopic(router.Publisher, "ui")
    
    // Eventually: direct engine + sink
    // sink := inference.WatermillSink(router.Publisher, "ui")
    // backend := ui.NewEngineBackend(engine, sink)
    
    backend := ui.NewStepBackend(uiStep)
    // ... rest unchanged
}
```

## 7. Migration Strategy

### Phase 0: Foundation (Week 1-2)
- [x] Create `geppetto/pkg/inference` package
- [x] Implement `Engine` interface with OpenAI and Claude implementations
- [x] Implement `EventSink` interface with Null and Watermill sinks
- [x] Implement `EngineFactory` interface for provider abstraction
- [x] Add `StandardEngineFactory` for automatic engine selection
- [ ] Add comprehensive tests for new components

### Phase 1: Adapter Introduction (Week 3)
- [x] Implement `StepAdapter` in `geppetto/pkg/steps/adapter`
- [x] Update existing `StepFactory` implementations to return adapters
- [x] Integrate `EngineFactory` into step creation process
- [ ] Verify all existing tests pass with zero functional changes
- [ ] Add adapter-specific tests

### Phase 2: Pinocchio Blocking Mode (Week 4)
- [ ] Refactor `runBlocking()` in `pinocchio/pkg/cmds/cmd.go` to use EngineFactory
- [ ] Update `ChatSession.runBlockingInternal()` to use EngineFactory
- [ ] Update ChatBuilder to accept EngineFactory instead of step factories
- [ ] Remove Step framework dependency from blocking paths
- [ ] Performance testing and validation

### Phase 3: Interactive/Chat Gradual Migration (Week 5-6)
- [ ] Create `ui.EngineBackend` as alternative to `ui.StepBackend`
- [ ] Update `runChat()` and `runInteractive()` to support both backends
- [ ] Gradual rollout with feature flags
- [ ] Comprehensive UI testing with streaming events

### Phase 4: Cleanup and Deprecation (Week 7-8)
- [ ] Remove `StepAdapter` usage from active code paths
- [ ] Deprecate Step API in geppetto (with migration guide)
- [ ] Performance optimization and final testing
- [ ] Documentation updates

## 8. Benefits and Trade-offs

### Benefits
1. **Simplified API Surface:** `RunInference(ctx, msgs) → (msg, error)` for basic cases
2. **Provider Abstraction:** `EngineFactory` allows external control of AI provider selection
3. **Flexible Event Publishing:** Optional, configurable, testable event sinks
4. **Clear Separation of Concerns:** Factory creates engines, engines do inference, sinks handle events
5. **Backwards Compatibility:** Existing code continues working during migration
6. **Performance:** Direct engine calls avoid channel/goroutine overhead for blocking cases
7. **Testability:** Easy to test inference without event infrastructure
8. **External Configuration:** AI provider selection controlled by settings, not hard-coded

### Trade-offs
1. **Initial Complexity:** Two APIs exist during migration period
2. **Code Duplication:** Some logic duplicated between Engine and StepAdapter
3. **Learning Curve:** Developers need to understand new patterns

## 9. Testing Strategy

### 9.1 Unit Tests
- [ ] Engine implementations with mock LLM APIs
- [ ] EventSink implementations with captured events
- [ ] StepAdapter compatibility with existing Step test suites

### 9.2 Integration Tests
- [ ] End-to-end pinocchio workflows with new architecture
- [ ] Watermill event flow validation
- [ ] UI integration with streaming events

### 9.3 Performance Tests
- [ ] Benchmark Engine vs Step for blocking operations
- [ ] Memory usage comparison
- [ ] Event publishing overhead measurement

## 10. Success Criteria

### Technical Success
- [ ] All existing pinocchio functionality preserved
- [ ] Performance improvement for blocking operations (>20% faster)
- [ ] Reduced memory usage in non-streaming scenarios
- [ ] 100% test coverage for new components

### Developer Experience Success
- [ ] Simple blocking inference requires <5 lines of code
- [ ] Clear migration path with comprehensive documentation
- [ ] No breaking changes during transition period
- [ ] Positive developer feedback on API simplicity

## Conclusion

This "Engine-first, Step-wrapper" architecture provides a clear path to simplify pinocchio's LLM integration while preserving all existing capabilities. The design enables gradual migration, maintains backwards compatibility, and provides significant simplification for common use cases while retaining the full power of the event-driven system for complex UI scenarios.

The proposal addresses all requirements:
- ✅ Simple RunInference API for basic usage
- ✅ Optional watermill event publishing with multiple sink support
- ✅ Backwards compatibility during migration via StepAdapter
- ✅ Flexible event integration (on/off toggle)
- ✅ Support for all existing pinocchio modes
- ✅ Preservation of existing event types
- ✅ Provider abstraction via EngineFactory for external control

Implementation can begin immediately with Phase 0, providing value at each step of the migration process.
