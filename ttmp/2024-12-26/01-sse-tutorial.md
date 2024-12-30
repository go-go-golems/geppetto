# Server-Sent Events (SSE) with HTMX 2.0

This tutorial demonstrates how to implement real-time updates using Server-Sent Events (SSE) with HTMX 2.0. We'll build a simple application that streams events from a Go server to a web browser, with proper error handling and a clean user interface.

## What are Server-Sent Events?

Server-Sent Events (SSE) is a web technology where a browser receives automatic updates from a server via HTTP connection. Unlike WebSockets, SSE is:
- Unidirectional (server to client only)
- Built on regular HTTP
- Automatically handles reconnection
- Simpler to implement than WebSockets
- Works well through firewalls and proxies

## Prerequisites

- Go 1.16 or later
- Basic understanding of HTML and JavaScript
- HTMX 2.0
- HTMX SSE Extension

## Project Structure

```
cmd/tutorials/sse/
├── main.go           # Server implementation
├── templates/
│   └── index.html    # Main HTML template
└── static/
    └── styles.css    # Styling
```

## Step 1: Setting Up the HTML Template

First, we need to set up our HTML template with HTMX and the SSE extension:

```html
<!DOCTYPE html>
<html>
<head>
    <title>SSE Demo with htmx</title>
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
    <script src="https://unpkg.com/htmx-ext-sse@2.2.2/sse.js"></script>
    <link rel="stylesheet" href="/static/styles.css">
</head>
```

Key points about the template:
- We use HTMX 2.0.4 or later
- The SSE extension is required for SSE support
- We include a CSS file for styling

## Step 2: Adding SSE Debug Listeners

To help with debugging SSE events, we add event listeners:

```javascript
document.addEventListener('DOMContentLoaded', function() {
    document.body.addEventListener('htmx:sseBeforeMessage', function (e) {
        console.log('SSE Before Message:', {
            event: e.detail.event,
            data: e.detail.data,
            target: e.detail.elt,
            swap: e.detail.swap
        });
    });
    // ... other event listeners ...
});
```

Important events to listen for:
- `htmx:sseBeforeMessage`: Fired before processing an SSE message
- `htmx:sseMessage`: Fired after processing an SSE message
- `htmx:sseError`: Fired when an error occurs
- `htmx:sseOpen`: Fired when the connection is established

## Step 3: Implementing the Server

The Go server needs several components:

1. Basic setup:
```go
func main() {
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    http.HandleFunc("/", handleHome)
    http.HandleFunc("/events", handleSSE)
    http.HandleFunc("/start-sse", handleStartSSE)
    http.HandleFunc("/clear", handleClear)
}
```

2. SSE handler setup:
```go
func handleSSE(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
}
```

Key server components:
- Event stream headers for SSE
- Goroutine for event generation
- Error group for proper concurrency handling
- Context for cancellation support

## Step 4: Event Generation and Sending

The server generates events in a separate goroutine:

```go
type Event struct {
    Message string
    Time    string
}

// In handleSSE:
g.Go(func() error {
    count := 1
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            event := Event{
                Message: fmt.Sprintf("Event #%d", count),
                Time:    time.Now().Format("15:04:05"),
            }
            // Send event...
            count++
        }
    }
})
```

Important aspects:
- Events are generated every 2 seconds
- Each event has a message and timestamp
- Context cancellation is properly handled
- Events are buffered in a channel

## Step 5: SSE Message Format

SSE messages must follow a specific format:

```
event: Update
data: <div class="event"><span class="time">15:04:05</span> Event #1</div>

```

Key formatting rules:
- Event name is optional (defaults to "message")
- Data line contains the payload
- Double newline (`\n\n`) terminates the message
- Content can be HTML for direct DOM insertion

## Step 6: HTMX SSE Integration

The SSE container in the HTML needs specific attributes:

```html
<div class="sse-container" 
     hx-ext="sse" 
     sse-connect="/events">
    <div id="events" sse-swap="Update">
        <div class="event">Waiting for first event...</div>
    </div>
    <div id="status" sse-swap="TestMessage">
        <div class="status">Connected - Waiting for updates...</div>
    </div>
</div>
```

Important attributes:
- `hx-ext="sse"`: Enables SSE extension
- `sse-connect="/events"`: Specifies SSE endpoint
- `sse-swap="Update"`: Specifies which event to listen for
- `id="events"`: Target for updates

## Step 7: Manual Connection Control

We implement a button to start the SSE connection:

```html
<button class="btn"
        hx-get="/start-sse"
        hx-swap="innerHTML"
        hx-target="#sse-container">
    Start Events
</button>
```

Benefits of manual connection:
- Better user experience
- Reduced server load
- Clear connection state
- Easy to implement reconnection

## Error Handling and Debugging

Important error handling considerations:

1. Server-side:
- Use error group for goroutine management
- Proper context cancellation
- Channel cleanup
- Logging at key points

2. Client-side:
- Event listeners for debugging
- Connection state monitoring
- Error event handling
- Automatic reconnection

## Best Practices

1. Server Implementation:
- Use buffered channels
- Implement proper cleanup
- Handle context cancellation
- Use appropriate timeouts

2. Client Implementation:
- Add error handling
- Monitor connection state
- Use appropriate swap strategies
- Implement reconnection logic

3. General:
- Log important events
- Handle disconnections gracefully
- Use appropriate event names
- Structure HTML for easy updates

## Conclusion

SSE with HTMX provides a simple yet powerful way to implement real-time updates. Key advantages:
- Simple implementation
- Works over HTTP
- Automatic reconnection
- Browser-native technology
- Clean integration with HTMX

Remember to handle errors appropriately and implement proper cleanup on both client and server sides. 
