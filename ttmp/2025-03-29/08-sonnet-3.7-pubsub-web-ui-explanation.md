# Web UI Server for Geppetto: Leveraging PubSub with Server-Sent Events

This document explains how the web-ui server in Pinocchio leverages the Watermill PubSub architecture to provide a real-time chat interface using Server-Sent Events (SSE).

## Architecture Overview

The web-ui server combines several components:

1. **Server**: Handles HTTP requests and SSE connections
2. **ChatClient**: Represents a connected client and manages its conversation state
3. **EventRouter**: Routes events from AI steps to connected clients
4. **Templ Components**: Render HTML for the web interface

The architecture follows an event-driven design where:
- User messages trigger AI step execution
- Events from steps are published to topic streams
- Server-Sent Events push real-time updates to the browser

## Server Component

The `Server` struct is the central component that:
- Manages HTTP routes
- Maintains a registry of connected clients
- Handles SSE connections
- Processes user messages

```go
type Server struct {
    router     *events.EventRouter
    clients    map[string]*client.ChatClient
    clientsMux sync.RWMutex
    logger     zerolog.Logger
}
```

### Client Management

The server maintains a map of connected clients identified by unique IDs:

```go
func (s *Server) RegisterClient(client *client.ChatClient) {
    s.clientsMux.Lock()
    defer s.clientsMux.Unlock()
    s.clients[client.ID] = client
    s.logger.Info().Str("client_id", client.ID).Msg("Registered new client")
}
```

When clients disconnect, they are properly unregistered:

```go
func (s *Server) UnregisterClient(clientID string) {
    s.clientsMux.Lock()
    defer s.clientsMux.Unlock()
    if client, ok := s.clients[clientID]; ok {
        close(client.MessageChan)
        close(client.DisconnectCh)
        delete(s.clients, clientID)
        s.logger.Info().Str("client_id", clientID).Msg("Unregistered client")
    }
}
```

### HTTP Endpoints

The server registers three main HTTP endpoints:

1. **/** - Index page that displays the chat interface
2. **/events** - SSE endpoint that streams events to clients
3. **/chat** - Endpoint for submitting chat messages

## SSE Integration with PubSub

The key integration between the PubSub system and the web UI happens through the Server-Sent Events endpoint:

```go
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    
    // Get client by ID
    clientID := r.URL.Query().Get("client_id")
    client_ := s.clients[clientID] // (simplified)
    
    // Stream events
    for {
        select {
        case <-r.Context().Done():
            // Client disconnected
            return
        case msg, ok := <-client_.MessageChan:
            // Send message to client
            fmt.Fprintf(w, "event: message\n")
            for _, line := range strings.Split(msg, "\n") {
                fmt.Fprintf(w, "data: %s\n", line)
            }
            fmt.Fprintf(w, "\n")
            flusher.Flush()
        case <-time.After(30 * time.Second):
            // Heartbeat
            fmt.Fprintf(w, "event: heartbeat\ndata: ping\n\n")
            flusher.Flush()
        }
    }
}
```

This endpoint creates a long-lived HTTP connection that:
1. Receives messages from the client's message channel
2. Formats them as SSE events
3. Sends them to the browser in real-time

## ChatClient: The Bridge Between PubSub and SSE

The `ChatClient` is the critical component that bridges the PubSub system with the SSE stream:

```go
type ChatClient struct {
    ID           string
    MessageChan  chan string
    DisconnectCh chan struct{}
    router       *events.EventRouter
    manager      conversation.Manager
    step         chat.Step
    stepResult   steps.StepResult[*conversation.Message]
    mu           sync.RWMutex
    logger       zerolog.Logger
}
```

### Event Subscription

When a new `ChatClient` is created, it subscribes to events on a dedicated topic:

```go
topic := fmt.Sprintf("chat-%s", id)
if err := client.step.AddPublishedTopic(router.Publisher, topic); err != nil {
    client.logger.Error().Err(err).Msg("Failed to setup event publishing")
    return client
}

// Add handler for this client's events
router.AddHandler(
    topic,
    topic,
    func(msg *message.Message) error {
        // Parse event from JSON
        e, err := chat.NewEventFromJson(msg.Payload)
        
        // Convert event to HTML
        html, err := client.EventToHTML(e)
        
        // Send HTML to message channel for SSE
        client.MessageChan <- html
        return nil
    },
)
```

This handler:
1. Receives events published by the AI step
2. Converts them to HTML using templ components
3. Sends the HTML through the message channel to the SSE endpoint

### Message Processing

When a user sends a message, `SendUserMessage` is called:

```go
func (c *ChatClient) SendUserMessage(ctx context.Context, message string) error {
    // Add user message to conversation
    userMsg := conversation.NewChatMessage(conversation.RoleUser, message)
    c.manager.AppendMessages(userMsg)
    
    // Cancel existing step if running
    if c.stepResult != nil {
        c.stepResult.Cancel()
    }
    
    // Start chat step with conversation
    result, err := c.step.Start(ctx, c.manager.GetConversation())
    
    // Process results in background
    go func() {
        for result := range result.GetChannel() {
            // Process result (error handling, etc.)
        }
    }()
    
    return nil
}
```

This function:
1. Updates the conversation with the user message
2. Starts an AI step with the entire conversation history
3. Sets up a goroutine to monitor the step results

## Event Processing Chain

The complete event flow from user input to UI update is:

1. User submits a message through the `/chat` endpoint
2. Server processes the message and calls `SendUserMessage` on the client
3. Client starts the AI step with the conversation
4. Step processes the conversation and publishes events to the client's topic
5. Client's event handler converts events to HTML
6. HTML is sent through the message channel to the SSE endpoint
7. Browser receives the SSE events and updates the UI in real-time

## Conversation Model for Web

For web display, the system converts the internal conversation model to a web-friendly format:

```go
func ConvertConversation(conv conversation.Conversation) (*WebConversation, error) {
    webConv := &WebConversation{
        Messages: make([]*WebMessage, 0, len(conv)),
    }

    for _, msg := range conv {
        webMsg, err := ConvertMessage(msg)
        if err != nil {
            return nil, err
        }
        webConv.Messages = append(webConv.Messages, webMsg)
    }

    return webConv, nil
}
```

This conversion handles different message types:
- Chat messages (user and assistant)
- Tool use messages
- Tool result messages

## UI Components with HTMX

The UI is built with HTMX for interactivity:

1. **EventContainer**: Sets up the SSE connection and displays streaming events
   ```html
   <div id="events" hx-ext="sse" sse-connect="/events?client_id=...">
       <div class="assistant-response" sse-swap="message"></div>
   </div>
   ```

2. **ChatInput**: Submits messages via AJAX
   ```html
   <form hx-post="/chat" hx-swap="outerHTML">
       <input type="hidden" name="client_id" value="..."/>
       <input type="text" name="message" class="form-control"/>
       <button type="submit" class="btn btn-primary">Send</button>
   </form>
   ```

3. **ConversationHistory**: Displays the conversation history
   ```html
   <div id="conversation-history" hx-swap-oob="#conversation-history">
       <!-- Message components -->
   </div>
   ```

## Event Conversion to HTML

The `EventToHTML` method converts chat events to HTML:

```go
func (c *ChatClient) EventToHTML(e chat.Event) (string, error) {
    var buf strings.Builder
    
    switch e_ := e.(type) {
    case *chat.EventPartialCompletion:
        // Render partial completion for streaming
        components.AssistantMessage(time.Now(), e_.Completion).Render(context.Background(), &buf)
    case *chat.EventFinal:
        // Add message to conversation and render full history
        c.manager.AppendMessages(conversation.NewChatMessage(conversation.RoleAssistant, e_.Text))
        conv := c.manager.GetConversation()
        webConv, err := web_conversation.ConvertConversation(conv)
        components.ConversationHistory(webConv, true).Render(context.Background(), &buf)
    case *chat.EventError:
        // Render error
        // ...
    }
    
    return buf.String(), nil
}
```

This method handles different event types:
- Partial completions for streaming responses
- Final messages that update the conversation history
- Error messages

## Conclusion

The web-ui server demonstrates how the PubSub architecture in Geppetto can be leveraged to create responsive web interfaces:

1. Steps publish events through the Watermill event router
2. Chat clients subscribe to these events and convert them to HTML
3. Server-Sent Events stream the HTML to the browser
4. HTMX updates the UI in real-time without full page reloads

This architecture enables:
- Real-time streaming of AI responses
- Tool call visualization
- Stateful conversation management
- Clean separation of concerns between AI processing and UI updates

The event-driven design allows for scalability and flexibility, as new features can be added by introducing new event types and corresponding UI components. 