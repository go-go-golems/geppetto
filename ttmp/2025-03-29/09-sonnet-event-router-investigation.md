# EventRouter: In-Depth Investigation and Refactoring Suggestions

This document explores the `EventRouter` component within Geppetto, examining how it integrates with Watermill, its handler management capabilities, and proposing refactoring suggestions to address some design issues.

## 1. EventRouter Overview

The `EventRouter` is a critical component in Geppetto's event-driven architecture that serves as a wrapper around the Watermill library's message routing capabilities:

```go
type EventRouter struct {
    logger     watermill.LoggerAdapter
    Publisher  message.Publisher
    Subscriber message.Subscriber
    router     *message.Router
    verbose    bool
}
```

### Core Responsibilities

1. **Event Routing**: Connects publishers (event sources) with subscribers (event handlers)
2. **Handler Management**: Registers functions to process events from specific topics
3. **Lifecycle Management**: Controls the starting and stopping of the router

### Initialization

The `EventRouter` is created using a functional options pattern:

```go
func NewEventRouter(options ...EventRouterOption) (*EventRouter, error) {
    ret := &EventRouter{
        logger: watermill.NopLogger{},
    }
    
    for _, o := range options {
        o(ret)
    }
    
    // Create a Go channel-based pub/sub by default
    goPubSub := gochannel.NewGoChannel(gochannel.Config{
        BlockPublishUntilSubscriberAck: true,
    }, ret.logger)
    ret.Publisher = goPubSub
    ret.Subscriber = goPubSub
    
    // Create the Watermill router
    router, err := message.NewRouter(message.RouterConfig{}, ret.logger)
    if err != nil {
        return nil, err
    }
    
    ret.router = router
    
    return ret, nil
}
```

This initialization:
1. Creates a default in-memory pub/sub system using Watermill's `gochannel`
2. Configures the underlying Watermill router
3. Applies any provided options to customize behavior

## 2. Understanding Watermill and Its Router

Before diving deeper into `EventRouter`, it's important to understand how Watermill is designed, examining the actual implementation in `router.go`.

### Watermill's Core Philosophy

According to Watermill's documentation, it was designed with a simple goal:

> "Watermill is a Go library for working efficiently with message streams. It is intended for building event driven applications, enabling event sourcing, RPC over messages, sagas and basically whatever else comes to your mind."

Watermill's design philosophy focuses on:

1. **Simplicity**: Providing a clean, consistent API across different message brokers
2. **Composability**: Building higher-level abstractions from simple primitives
3. **Flexibility**: Supporting various messaging patterns and use cases

### Watermill's Architecture

Watermill has a layered architecture:

1. **Messages**: The core data structure (`message.Message`)
2. **Publisher/Subscriber**: Low-level interfaces for sending and receiving messages
3. **Router**: Connects publishers and subscribers with handlers
4. **Middleware**: Enhances handlers with additional functionality
5. **Higher-level components**: CQRS, Saga, etc.

### The Router in Watermill

Looking at the actual implementation in `router.go`, the Watermill Router is a sophisticated component with multiple capabilities:

```go
type Router struct {
    config RouterConfig

    middlewares     []middleware
    middlewaresLock *sync.RWMutex

    plugins []RouterPlugin

    handlers     map[string]*handler
    handlersLock *sync.RWMutex

    handlersWg *sync.WaitGroup

    runningHandlersWg     *sync.WaitGroup
    runningHandlersWgLock *sync.Mutex

    handlerAdded chan struct{}

    closingInProgressCh chan struct{}
    closedCh            chan struct{}
    closed              bool
    closedLock          sync.Mutex

    logger watermill.LoggerAdapter

    publisherDecorators  []PublisherDecorator
    subscriberDecorators []SubscriberDecorator

    isRunning bool
    running   chan struct{}
}
```

The Router offers these key features:

1. **Concurrent Handler Execution**: Handlers are executed in parallel for multiple incoming messages
2. **Middleware Support**: Both router-level and handler-level middleware can be added
3. **Graceful Shutdown**: Router can be closed with proper cleanup of resources
4. **Handler Management**: Handlers can be added, started and stopped individually 
5. **Decorators**: Publishers and Subscribers can be wrapped with additional functionality
6. **Recovery from Panics**: Handler execution is protected with panic recovery
7. **Pluggable Design**: Plugins can be added to enhance router functionality

## 3. Connection to Watermill's Router

The `EventRouter` primarily serves as a simplified interface to Watermill's more complex `message.Router`. Key connections include:

### Router Delegation

Most of the `EventRouter` methods directly delegate to Watermill's router:

```go
func (e *EventRouter) Run(ctx context.Context) error {
    return e.router.Run(ctx)
}

func (e *EventRouter) RunHandlers(ctx context.Context) error {
    return e.router.RunHandlers(ctx)
}

func (e *EventRouter) Running() chan struct{} {
    return e.router.Running()
}
```

### Handler Registration

The `AddHandler` method wraps Watermill's `AddNoPublisherHandler` to simplify the handler registration process:

```go
func (e *EventRouter) AddHandler(name string, topic string, f func(msg *message.Message) error) {
    e.router.AddNoPublisherHandler(name, topic, e.Subscriber, f)
}
```

This method simplifies the Watermill API by:
1. Only requiring the handler name, topic, and function
2. Automatically using the router's preconfigured subscriber
3. Using a simpler function signature (no message publishing)

### Simplification Choices

The `EventRouter` makes specific simplifications to Watermill's full capabilities:

1. **No Message Publishing from Handlers**: Uses `AddNoPublisherHandler` instead of Watermill's full `AddHandler` which would allow returning messages from handlers. In Watermill's implementation, handlers can return messages to be published:

   ```go
   // From watermill/message/router.go
   type HandlerFunc func(msg *Message) ([]*Message, error)
   ```

2. **Single Pub/Sub Backend**: Uses a single publisher/subscriber pair compared to Watermill's flexibility to mix and match different implementations.

3. **Limited Configuration Options**: Exposes only a subset of Watermill's configuration parameters. For example, Watermill's router supports configuration like `CloseTimeout` which is not exposed in `EventRouter`.

4. **No Decorator Support**: Watermill Router supports decorating publishers and subscribers:

   ```go
   // From watermill/message/router.go
   func (r *Router) AddPublisherDecorators(dec ...PublisherDecorator) {
       r.publisherDecorators = append(r.publisherDecorators, dec...)
   }
   ```

5. **No Plugin Support**: Watermill's plugin system is not exposed through `EventRouter`:

   ```go
   // From watermill/message/router.go
   func (r *Router) AddPlugin(p ...RouterPlugin) {
       r.plugins = append(r.plugins, p...)
   }
   ```

### Missing Router Close Implementation

An important observation is that `EventRouter.Close()` doesn't properly close the underlying Watermill router:

```go
func (e *EventRouter) Close() error {
    err := e.Publisher.Close()
    if err != nil {
        log.Error().Err(err).Msg("Failed to close pubsub")
        // not returning just yet
    }

    return nil
}
```

This method only closes the publisher but not the router itself, which could lead to resource leaks. Looking at Watermill's implementation, the router's `Close()` method does much more:

```go
// From watermill/message/router.go
func (r *Router) Close() error {
    r.closedLock.Lock()
    defer r.closedLock.Unlock()

    r.handlersLock.Lock()
    defer r.handlersLock.Unlock()

    if r.closed {
        return nil
    }
    r.closed = true

    r.logger.Info("Closing router", nil)
    defer r.logger.Info("Router closed", nil)

    close(r.closingInProgressCh)
    defer close(r.closedCh)

    timeouted := r.waitForHandlers()
    if timeouted {
        return errors.New("router close timeout")
    }

    return nil
}
```

This implementation properly waits for handlers to finish processing, respecting the configured timeout. A proper `EventRouter.Close()` should call `e.router.Close()`.

## 4. Handler Management

### Adding Handlers

Handlers can be added using the `AddHandler` method:

```go
router.AddHandler("ui-stdout", "ui", func(msg *message.Message) error {
    // Process the message
    return nil
})
```

Each handler:
1. Must have a unique name across the router
2. Subscribes to a specific topic
3. Processes messages using a provided function

When a handler is added in Watermill, it returns a `Handler` struct that can be used to control it:

```go
// From watermill/message/router.go
func (r *Router) AddHandler(
    handlerName string,
    subscribeTopic string,
    subscriber Subscriber,
    publishTopic string,
    publisher Publisher,
    handlerFunc HandlerFunc,
) *Handler {
    // ...implementation...
    
    return &Handler{
        router:  r,
        handler: newHandler,
    }
}
```

However, `EventRouter.AddHandler` doesn't return this handler, limiting control options:

```go
// EventRouter.AddHandler doesn't return the handler
func (e *EventRouter) AddHandler(name string, topic string, f func(msg *message.Message) error) {
    e.router.AddNoPublisherHandler(name, topic, e.Subscriber, f)
}
```

### Removing Handlers

**Current Limitation**: The `EventRouter` does not provide a method to remove handlers once they're added.

Watermill provides a way to stop individual handlers using the `Handler.Stop()` method:

```go
// From watermill/message/router.go
// Stop stops the handler.
// Stop is asynchronous.
// You can check if handler was stopped with Stopped() function.
func (h *Handler) Stop() {
    if !h.handler.started {
        panic("handler is not started")
    }

    h.handler.stopFn()
}
```

Since `EventRouter.AddHandler` doesn't return the handler object, there's no way to access this functionality through the `EventRouter` API.

### Handler Execution

Looking at the Watermill implementation, each handler is executed in its own goroutine for each incoming message:

```go
// From watermill/message/router.go
func (h *handler) run(ctx context.Context, middlewares []middleware) {
    // ...

    for msg := range h.messagesCh {
        h.runningHandlersWgLock.Lock()
        h.runningHandlersWg.Add(1)
        h.runningHandlersWgLock.Unlock()

        go h.handleMessage(msg, middlewareHandler)
    }

    // ...
}
```

This concurrent execution model allows for high throughput processing of messages. The `EventRouter` preserves this behavior by delegating to Watermill's router.

### Handler Context and Metadata

Watermill's Router adds context values to messages processed by handlers:

```go
// From watermill/message/router.go
func (h *handler) addHandlerContext(messages ...*Message) {
    for i, msg := range messages {
        ctx := msg.Context()

        if h.name != "" {
            ctx = context.WithValue(ctx, handlerNameKey, h.name)
        }
        if h.publisherName != "" {
            ctx = context.WithValue(ctx, publisherNameKey, h.publisherName)
        }
        // ... more context values ...
        
        messages[i].SetContext(ctx)
    }
}
```

These context values can be useful for debugging and tracing, but are not directly accessible through the `EventRouter` API.

## 5. Middleware Capabilities in Watermill

One of the most powerful features of Watermill's Router is its support for middleware. Looking at the implementation, Watermill supports both router-level and handler-level middleware:

```go
// From watermill/message/router.go
func (r *Router) AddMiddleware(m ...HandlerMiddleware) {
    r.logger.Debug("Adding middleware", watermill.LogFields{"count": fmt.Sprintf("%d", len(m))})
    r.addRouterLevelMiddleware(m...)
}

// From watermill/message/router.go
func (h *Handler) AddMiddleware(m ...HandlerMiddleware) {
    handler := h.handler
    handler.logger.Debug("Adding middleware to handler", watermill.LogFields{
        "count":       fmt.Sprintf("%d", len(m)),
        "handlerName": handler.name,
    })

    h.router.addHandlerLevelMiddleware(handler.name, m...)
}
```

Middleware in Watermill follows a decorator pattern:

```go
// From watermill/message/router.go
type HandlerMiddleware func(h HandlerFunc) HandlerFunc
```

This allows for powerful composability of handler behavior. Middlewares are applied in order, with the first middleware wrapping the outermost layer:

```go
// From watermill/message/router.go
middlewareHandler := h.handlerFunc
// first added middlewares should be executed first (so should be at the top of call stack)
for i := len(middlewares) - 1; i >= 0; i-- {
    currentMiddleware := middlewares[i]
    isValidHandlerLevelMiddleware := currentMiddleware.HandlerName == h.name
    if currentMiddleware.IsRouterLevel || isValidHandlerLevelMiddleware {
        middlewareHandler = currentMiddleware.Handler(middlewareHandler)
    }
}
```

The `EventRouter` doesn't expose this middleware capability at all, missing out on a key feature of Watermill.

## 6. Real-World Usage in Geppetto

### Web UI Server

In the web UI server, the `EventRouter` connects AI processing steps with the web frontend:

```go
// Create and configure router
router, err := events.NewEventRouter(events.WithVerbose(true))

// Create server with router
server := NewServer(router)

// Each client registers its own topic and handler
topic := fmt.Sprintf("chat-%s", clientID)
router.AddHandler(topic, topic, func(msg *message.Message) error {
    // Parse event and convert to HTML
    e, err := chat.NewEventFromJson(msg.Payload)
    html, err := client.EventToHTML(e)
    client.MessageChan <- html
    return nil
})

// Connect Step's events to the topic
client.step.AddPublishedTopic(router.Publisher, topic)
```

This pattern enables:
1. Unique topics per client/conversation
2. Dynamic handler registration as clients connect
3. Streaming events from AI steps to web UI

### Tool UI

The tool UI demonstrates how the `EventRouter` handles different event types:

```go
t.eventRouter.AddHandler("raw-events-stdout", "ui", t.eventRouter.DumpRawEvents)
t.eventRouter.AddHandler("ui-stdout", "ui", func(msg *message.Message) error {
    // Process different event types
    e, _ := chat.NewEventFromJson(msg.Payload)
    switch e_ := e.(type) {
    case *chat.EventPartialCompletion:
        // Handle streaming completion
    case *chat.EventToolCall:
        // Handle tool calls
    // ...other event types
    }
    return nil
})
```

## 7. Limitations of the Current Design

After examining the actual Watermill router implementation, several limitations of the `EventRouter` become even more apparent:

### 1. No Handler Removal

As mentioned, handlers cannot be removed dynamically, which could lead to resource leaks in long-running applications. While Watermill supports stopping individual handlers via `Handler.Stop()`, `EventRouter` doesn't expose this functionality.

### 2. Name Confusion

The name `EventRouter` doesn't clearly reflect its full responsibility as both a router and a pub/sub factory. Watermill itself makes a clearer distinction between the router and the pub/sub components.

### 3. Limited Configuration

The `EventRouter` enforces a 1:1 relationship between Publisher and Subscriber, which may not fit all use cases. Watermill's router is designed to work with multiple different pub/sub implementations simultaneously, which is not exposed in `EventRouter`.

### 4. Mixing of Concerns

The `EventRouter` combines:
- Pub/Sub creation and management
- Message routing
- Event handling
- Serialization/deserialization (via the `DumpRawEvents` method)

This contrasts with Watermill's cleaner separation of concerns.

### 5. Inconsistent Relationship with PublisherManager

A separate `PublisherManager` exists which overlaps with some `EventRouter` responsibilities, creating confusion:

```go
// From events/publish.go
// NOTE(manuel, 2024-03-24) This might be worth moving / integrating into the event router
// It sounds also logical that this is the thing that would add sequence numbers to events?
type PublisherManager struct {
    Publishers     map[string][]message.Publisher
    sequenceNumber uint64
    mutex          sync.Mutex
}
```

### 6. Missing Middleware Support

Watermill has powerful middleware capabilities that allow for:
- Correlation ID tracking
- Error handling/retries
- Metrics/logging
- Rate limiting
- Poison queue handling

`EventRouter` doesn't expose this functionality, limiting its extensibility.

### 7. No Decorator Support

Watermill's publisher and subscriber decorator capabilities are not exposed:

```go
// From watermill/message/router.go
func (r *Router) AddPublisherDecorators(dec ...PublisherDecorator) {
    r.publisherDecorators = append(r.publisherDecorators, dec...)
}

func (r *Router) AddSubscriberDecorators(dec ...SubscriberDecorator) {
    r.subscriberDecorators = append(r.subscriberDecorators, dec...)
}
```

These decorators could be used to add functionality like metrics, logging, or retries to all publishers and subscribers.

### 8. No Plugin Support

Watermill's plugin system is not exposed through `EventRouter`:

```go
// From watermill/message/router.go
func (r *Router) AddPlugin(p ...RouterPlugin) {
    r.plugins = append(r.plugins, p...)
}
```

Plugins can provide additional functionality like signal handling or health checks.

### 9. Incomplete Close Method

As noted earlier, `EventRouter.Close()` doesn't close the underlying Watermill router, potentially leading to resource leaks.

## 8. Refactoring Suggestions

Based on the analysis of Watermill's actual implementation, here are refined refactoring suggestions:

### 1. Separate Pub/Sub Creation from Routing

```go
// Proposed new types
type EventBus interface {
    Publisher() message.Publisher
    Subscriber() message.Subscriber
    Close() error
}

type EventDispatcher interface {
    AddHandler(name, topic string, handler HandlerFunc) *Handler
    RunHandlers(ctx context.Context) error
    Run(ctx context.Context) error
    Close() error
    
    // Expose additional Watermill functionality
    AddMiddleware(middleware HandlerMiddleware)
    AddPlugin(plugin RouterPlugin)
    AddPublisherDecorators(decorators ...PublisherDecorator)
    AddSubscriberDecorators(decorators ...SubscriberDecorator)
}
```

### 2. Return Handlers and Add Handler Removal Support

Leverage Watermill's existing handler control mechanisms by directly returning the handler object:

```go
// Return the handler from AddHandler
func (d *EventDispatcher) AddHandler(name, topic string, handlerFunc HandlerFunc) *Handler {
    return d.router.AddNoPublisherHandler(name, topic, d.subscriber, adaptHandlerFunc(handlerFunc))
}

// Example usage
handler := dispatcher.AddHandler("my-handler", "topic", myHandlerFunc)

// Later, stop the handler when needed
handler.Stop()

// You can also check if handler has stopped
<-handler.Stopped()
```

### 3. Rename Components for Clarity

Proposed new naming:
- `EventRouter` → `EventDispatcher` (for routing functionality)
- `PublisherManager` → `TopicPublisher` (for publishing to multiple subscribers)
- Add `EventBus` (for pub/sub creation and management)

### 4. Expose Full Router Configuration

Allow configuration of all Watermill router options:

```go
type EventDispatcherConfig struct {
    // Forward Watermill's router config
    RouterConfig message.RouterConfig
    
    // Additional Geppetto-specific config
    SequenceNumbering bool
    VerboseLogging    bool
}

func NewEventDispatcher(bus EventBus, config EventDispatcherConfig) (*EventDispatcher, error) {
    router, err := message.NewRouter(config.RouterConfig, bus.Logger())
    if err != nil {
        return nil, err
    }
    
    // ...
}
```

### 5. Integrate PublisherManager

Merge the functionality of `PublisherManager` into the core event system, preserving sequence numbering:

```go
type TopicPublisher struct {
    publisher message.Publisher
    seqNum    atomic.Uint64
    mu        sync.Mutex
}

func (p *TopicPublisher) PublishEvent(topic string, event Event) error {
    // Add sequence number
    seqNum := p.seqNum.Add(1)
    event.SetSequenceNumber(seqNum)
    
    // Serialize event
    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    // Create Watermill message
    msg := message.NewMessage(watermill.NewUUID(), payload)
    msg.Metadata.Set("sequence_number", fmt.Sprintf("%d", seqNum))
    
    // Publish through Watermill
    return p.publisher.Publish(topic, msg)
}
```

### 6. Add Full Middleware Support

Expose Watermill's middleware capabilities:

```go
// Add global middleware
func (d *EventDispatcher) AddMiddleware(m HandlerMiddleware) {
    d.router.AddMiddleware(m)
}

// Example usage with Watermill's built-in middleware
dispatcher.AddMiddleware(middleware.Recoverer)
dispatcher.AddMiddleware(middleware.Retry{
    MaxRetries:      3,
    InitialInterval: time.Second,
    MaxInterval:     time.Minute,
}.Middleware)

// Example handler-specific middleware
handler := dispatcher.AddHandler("my-handler", "topic", myHandlerFunc)
handler.AddMiddleware(middleware.Throttle(10).Middleware)
```

### 7. Add Support for Publisher Decorators and Plugins

Expose Watermill's plugin and decorator systems:

```go
// Add plugin
func (d *EventDispatcher) AddPlugin(p RouterPlugin) {
    d.router.AddPlugin(p)
}

// Add publisher decorator
func (d *EventDispatcher) AddPublisherDecorators(decorators ...PublisherDecorator) {
    d.router.AddPublisherDecorators(decorators...)
}

// Add subscriber decorator
func (d *EventDispatcher) AddSubscriberDecorators(decorators ...SubscriberDecorator) {
    d.router.AddSubscriberDecorators(decorators...)
}

// Example usage
dispatcher.AddPlugin(plugins.SignalsHandler)
dispatcher.AddPublisherDecorators(metrics.PublisherDecorator("my_app"))
```

### 8. Fix the Close Method

Ensure proper resource cleanup by delegating to Watermill's router:

```go
func (d *EventDispatcher) Close() error {
    // First close the router
    if err := d.router.Close(); err != nil {
        return fmt.Errorf("failed to close router: %w", err)
    }
    
    return nil
}
```

### 9. Support for Multiple Publisher Handlers

Support handlers that can publish messages, by using Watermill's full `AddHandler` method:

```go
func (d *EventDispatcher) AddPublishingHandler(
    name string,
    subscribeTopic string,
    publishTopic string,
    handler func(msg *message.Message) ([]*message.Message, error),
) *Handler {
    return d.router.AddHandler(
        name,
        subscribeTopic,
        d.subscriber,
        publishTopic,
        d.publisher,
        handler,
    )
}

// Example usage
dispatcher.AddPublishingHandler(
    "transform-handler",
    "input-topic",
    "output-topic",
    func(msg *message.Message) ([]*message.Message, error) {
        // Transform message
        transformed := message.NewMessage(
            watermill.NewUUID(),
            []byte("Transformed: " + string(msg.Payload)),
        )
        return []*message.Message{transformed}, nil
    },
)
```

## 9. Implementation Plan

A phased approach to implementation:

1. **Phase 1**: Create new interfaces without changing existing code
   - Define `EventBus`, `EventDispatcher`, etc.
   - Implement new components alongside existing ones

2. **Phase 2**: Create a compatibility layer
   - Implement new components that work with existing code
   - Provide adapters for smooth migration

3. **Phase 3**: Gradually migrate client code to new interfaces
   - Update services one by one
   - Test and validate each migration step

4. **Phase 4**: Deprecate old interfaces
   - Mark old components as deprecated
   - Provide migration guides

5. **Phase 5**: Remove deprecated code
   - Remove old components after sufficient migration time
   - Finalize new API

This approach ensures backward compatibility while improving the design.

## Conclusion

The `EventRouter` provides essential functionality for Geppetto's event-driven architecture but has several design limitations compared to the full capabilities of Watermill's Router. By refactoring it into more specialized components with clearer responsibilities, we can improve maintainability, flexibility, and usability.

The proposed changes would better align with the single responsibility principle and provide a more intuitive API for event handling while maintaining compatibility with Watermill's robust messaging infrastructure. By properly exposing Watermill's capabilities like middleware, decorators, and plugins while adding Geppetto-specific requirements like sequence numbering, we can create a more powerful and flexible event system.

A well-designed event system will make it easier to add new features and debug issues in Geppetto's event-driven architecture. 