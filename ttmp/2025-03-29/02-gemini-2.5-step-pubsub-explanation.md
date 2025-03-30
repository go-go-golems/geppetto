# Explanation of Step Publishing and Watermill in Geppetto

This document explains how the `steps.Step` abstraction in Geppetto, particularly AI-related steps, utilizes publishing mechanisms via the `AddPublishedTopic` method and the role the Watermill library plays in this process.

## 1. Publishing in `steps.Step` via `AddPublishedTopic`

The `steps.Step[T, U]` interface (defined in `geppetto/pkg/steps/step.go`) includes the method:

```go
AddPublishedTopic(publisher message.Publisher, topic string) error
```

*   **Purpose:** This method serves as an injection point. It allows the system coordinating the steps to provide a specific `message.Publisher` instance (from the Watermill library) and a `topic` string to a `Step`. The step is then expected to use this publisher to broadcast events related to its execution lifecycle onto the given topic.
*   **Implementation Strategy:** Concrete step implementations, especially AI steps like `openai.ChatStep`, `claude.MessagesStep`, and various caching/utility steps, implement this method. They typically don't use the provided `publisher` and `topic` directly within the method. Instead, they often delegate the registration to an internal helper object, commonly an `events.PublisherManager` (from `geppetto/pkg/events/publish.go`).
    ```go
    // Example from openai.ChatStep
    func (csf *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
        csf.publisherManager.RegisterPublisher(topic, publisher) // Delegate registration
        return nil
    }
    ```
*   **Event Emission:** The core logic happens within the step's `Start` method. During execution, the step uses its internal `PublisherManager` (which now holds the registered publishers/topics) to publish various lifecycle events. It calls methods like `publisherManager.Publish(eventPayload)` or `publisherManager.PublishBlind(eventPayload)`.
*   **Event Types:** Steps publish events corresponding to different stages of their execution:
    *   **Start:** Indicates the step has begun processing (e.g., `chat.NewStartEvent`).
    *   **Partial Results:** For steps that stream output (like AI chat), events containing chunks of the result (e.g., `chat.NewPartialCompletionEvent`).
    *   **Final Result:** Signifies successful completion, often containing the final output (e.g., `chat.NewFinalEvent`).
    *   **Errors:** Published when an error occurs during execution (e.g., `chat.NewErrorEvent`).
    *   **Interrupts:** Emitted if the step's execution context is canceled (e.g., `chat.NewInterruptEvent`).
    These events usually encapsulate metadata about the step, its inputs/outputs, timestamps, and the specific payload for that event type.

## 2. Role of Watermill

Watermill (`github.com/ThreeDotsLabs/watermill`) is a fundamental Go library used by Geppetto for building event-driven applications. It provides the core pub/sub infrastructure.

*   **Abstraction Layer:** Watermill's primary role here is to provide standard interfaces like `message.Publisher` and `message.Subscriber`. This decouples the application logic (the Geppetto steps) from the underlying message transport mechanism. Geppetto steps don't need to know if messages are being sent via in-memory channels, Kafka, RabbitMQ, or another system; they just interact with the Watermill interface.
*   **Publisher Interface:** The `message.Publisher` type required by `AddPublishedTopic` comes directly from Watermill.
*   **Message Handling:** When a step calls `publisher.Publish(topic, message)` (usually indirectly via the `PublisherManager`), Watermill takes over. It handles constructing the `message.Message` object (often adding metadata like UUIDs) and routes it through the configured backend (e.g., the `gochannel` pubsub for in-memory communication, as configured in `geppetto/pkg/events/event-router.go`).
*   **Decoupling:** The use of Watermill is key to Geppetto's modularity. Steps (publishers) are decoupled from the services or components that consume their events (subscribers). A UI component, a logging service, or another analytical tool can subscribe to step events using a Watermill `message.Subscriber` without the step needing any direct reference to them.

## Summary

1.  The `AddPublishedTopic` method on `steps.Step` allows steps to be configured with a destination (topic) and a means (Watermill `message.Publisher`) to announce their status.
2.  AI steps implement this, often using an `events.PublisherManager` to store these configurations.
3.  During their `Start` execution, steps publish lifecycle events (start, partial, final, error, interrupt) using the manager.
4.  Watermill provides the core `message.Publisher` interface and handles the actual message transport, decoupling event producers (steps) from event consumers and the specific pub/sub technology used. 