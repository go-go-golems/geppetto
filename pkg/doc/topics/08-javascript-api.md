---
Title: JavaScript API for Geppetto - Complete Reference Guide
Slug: geppetto-javascript-api
Short: A comprehensive guide to using Geppetto's watermill-based JavaScript RuntimeEngine for conversations, embeddings, steps, and chat functionality through Goja bindings.
Topics:
- geppetto
- javascript
- api
- conversations
- embeddings
- steps
- chat
- bindings
- goja
- watermill
- runtime-engine
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# JavaScript API for Geppetto - Complete Reference Guide

This guide provides comprehensive documentation for Geppetto's JavaScript API, which allows you to interact with conversations, generate embeddings, execute steps, and perform AI chat operations from JavaScript code through a watermill-based RuntimeEngine.

## Overview

Geppetto's JavaScript API exposes four main components through the **RuntimeEngine**:

1. **Conversation API**: Create and manage conversations with messages, tool uses, and tool results
2. **Embeddings API**: Generate vector embeddings from text using various embedding models
3. **Steps API**: Execute asynchronous computation steps with watermill event streaming
4. **Setup Functions**: Modular JavaScript environment configuration

These APIs provide event-driven streaming capabilities through Watermill pub/sub architecture with automatic handler lifecycle management.

## Core Architecture

### RuntimeEngine - Watermill-Based JavaScript Execution

The new RuntimeEngine provides a clean, event-driven approach to JavaScript execution:

```go
// Create and configure engine
engine := js.NewRuntimeEngine()
defer engine.Close()

// Add setup functions
engine.AddSetupFunction(js.SetupDoubleStep())
engine.AddSetupFunction(js.SetupConversation())
engine.AddSetupFunction(js.SetupEmbeddings(stepSettings))
engine.AddSetupFunction(js.SetupDoneCallback())

// Start engine (blocking until completion)
engine.Start()

// Or execute JavaScript code on running loop
err := engine.RunOnLoop("console.log('Hello World');")
```

### Key Features

- **Watermill Integration**: Uses Watermill pub/sub for event streaming with automatic topic management
- **Per-Step Handlers**: Each step gets its own handler that auto-registers/unregisters on completion
- **Event Loop Management**: Proper `Loop.Run()` usage for setup and `RunOnLoop()` for execution
- **Modular Setup**: Setup functions for different components (steps, conversations, embeddings)
- **Auto-Cleanup**: Handlers and subscriptions automatically cleaned up when steps complete

### JavaScript-Go Integration

The JavaScript API is built on top of Goja with watermill event streaming:

- **Event-driven execution**: Steps publish events to watermill topics
- **Automatic handler management**: Per-step handlers auto-register and cleanup
- **Real-time streaming**: Events flow through watermill pub/sub system
- **Type conversion**: Seamless conversion between Go and JavaScript types
- **Event loop integration**: All callbacks happen on the managed event loop

### Common Patterns

The new API uses event-driven patterns with watermill:

```javascript
// Event-driven step execution with streaming
const stepID = step.runWithEvents(input, function(event) {
    console.log("Event:", event.type, event);
    
    switch(event.type) {
        case "start":
            console.log("Step started");
            break;
        case "partial-completion":
            console.log("Partial result:", event.delta);
            break;
        case "final":
            console.log("Final result:", event.text);
            break;
        case "error":
            console.error("Step error:", event.error);
            break;
    }
});

console.log("Step ID:", stepID);
```

## Conversation API

The Conversation API provides a JavaScript interface for creating and managing conversations with messages, tool uses, and tool results.

### Creating and Managing Conversations

```javascript
// Create a new conversation
const conv = new Conversation();

// Add a simple chat message
const msgId = conv.AddMessage("user", "Hello, how can I help you?");

// Add a message with options
const msgWithOptions = conv.AddMessage("system", "System prompt", {
    metadata: { source: "config" },
    parentID: "parent-message-id",
    time: "2024-01-01T00:00:00Z",
    id: "custom-id"  // optional, will generate UUID if not provided
});

// Add a message with an image
const msgWithImage = conv.AddMessageWithImage(
    "user",
    "Here's an image",
    "/path/to/image.jpg"  // supports local files and URLs
);
```

### Message Options

The `MessageOptions` interface provides flexible configuration:

```typescript
interface MessageOptions {
    metadata?: Record<string, any>;  // Additional metadata
    parentID?: string;               // Parent message ID
    time?: string;                   // RFC3339 format timestamp
    id?: string;                     // Custom message ID
}
```

### Tool Integration

The Conversation API supports tool use and tool result messages for AI function calling:

```javascript
// Add a tool use
const toolUseId = conv.AddToolUse(
    "tool123",
    "searchCode",
    { query: "find main function" }
);

// Add a tool result
const resultId = conv.AddToolResult(
    "tool123",
    "Found main function in main.go"
);
```

### Working with Messages

```javascript
// Get all messages
const messages = conv.GetMessages();
// Returns an array of message objects

// Get formatted view of a specific message
const messageView = conv.GetMessageView(msgId);
// Returns formatted string based on message type:
// - Chat: "[role]: text"
// - Tool Use: "ToolUseContent{...}"
// - Tool Result: "ToolResultContent{...}"

// Update message metadata
conv.UpdateMetadata(msgId, { processed: true });

// Get conversation as a single prompt string
const prompt = conv.GetSinglePrompt();

// Convert back to Go conversation object
const goConv = conv.toGoConversation();
```

### Message Object Structure

Messages returned by `GetMessages()` have different structures based on their type:

#### Common Fields
All message objects include:
```javascript
{
    id: string,          // Unique message ID
    parentID: string,    // Parent message ID
    time: Date,          // Creation timestamp
    lastUpdate: Date,    // Last update timestamp
    metadata: object,    // Additional metadata
    type: string        // Message type: "chat-message", "tool-use", or "tool-result"
}
```

#### Chat Message (type: "chat-message")
```javascript
{
    ...commonFields,
    role: string,        // "system", "assistant", "user", or "tool"
    text: string,        // Message content
    images?: [{          // Optional array of images
        imageURL: string,
        imageName: string,
        mediaType: string,
        detail: string
    }]
}
```

#### Tool Use (type: "tool-use")
```javascript
{
    ...commonFields,
    toolID: string,      // Tool identifier
    name: string,        // Tool name
    input: object,       // Tool input parameters
    toolType: string     // Tool type (e.g., "function")
}
```

#### Tool Result (type: "tool-result")
```javascript
{
    ...commonFields,
    toolID: string,      // Tool identifier
    result: string       // Tool execution result
}
```

### Image Support

The conversation API supports adding images to messages:

```javascript
// Add message with image
const msgWithImage = conv.AddMessageWithImage(
    "user",
    "What's in this image?",
    "/path/to/image.jpg"
);
```

**Supported formats**: PNG, JPEG, WebP, and GIF
**Maximum file size**: 20MB
**Sources**: Local file paths and URLs
**Constraints**: Images are automatically validated for format and size

## Embeddings API

The Embeddings API provides JavaScript bindings for generating vector embeddings from text using various embedding models.

### Core Concepts

Embeddings are vector representations of text that capture semantic meaning in a high-dimensional space. They're useful for:
- Semantic search and similarity comparison
- Document clustering and classification
- Information retrieval systems
- Machine learning features

### Model Information

Each embeddings provider exposes information about its model:

```javascript
const model = embeddings.getModel();
// Returns: { name: string, dimensions: number }
console.log("Using model:", model.name);
console.log("Vector dimensions:", model.dimensions);
```

### Synchronous API

For simple, blocking operations:

```javascript
const text = "Hello, world!";
try {
    const embedding = embeddings.generateEmbedding(text);
    // Returns: Float32Array of dimensions length
    console.log("Embedding dimensions:", embedding.length);
} catch (err) {
    console.error("Failed to generate embedding:", err);
}
```

### Asynchronous Promise API

Promise-based API for better error handling and non-blocking operations:

```javascript
async function generateEmbedding(text) {
    try {
        const embedding = await embeddings.generateEmbeddingAsync(text);
        console.log("Embedding dimensions:", embedding.length);
        return embedding;
    } catch (err) {
        console.error("Failed to generate embedding:", err);
        throw err;
    }
}

// Usage
const embedding = await generateEmbedding("Hello, world!");
```

### Callback API with Cancellation

For operations that need cancellation support:

```javascript
const text = "Hello, world!";
const cancel = embeddings.generateEmbeddingWithCallbacks(text, {
    onSuccess: (embedding) => {
        console.log("Embedding generated:", embedding);
    },
    onError: (err) => {
        console.error("Error:", err);
    }
});

// Cancel the operation if needed
setTimeout(() => {
    cancel();
}, 5000);
```

### Batch Processing

Process multiple texts efficiently:

```javascript
// Process multiple texts with Promise.all
const texts = [
    "First document",
    "Second document", 
    "Third document"
];

async function batchProcess(texts) {
    try {
        const embeddings = await Promise.all(
            texts.map(text => embeddings.generateEmbeddingAsync(text))
        );
        return embeddings;
    } catch (err) {
        console.error("Batch processing failed:", err);
        throw err;
    }
}

const allEmbeddings = await batchProcess(texts);
```

### Semantic Search Example

Implementing semantic search using embeddings:

```javascript
// Function to compute cosine similarity between vectors
function cosineSimilarity(a, b) {
    let dotProduct = 0;
    let normA = 0;
    let normB = 0;
    
    for (let i = 0; i < a.length; i++) {
        dotProduct += a[i] * b[i];
        normA += a[i] * a[i];
        normB += b[i] * b[i];
    }
    
    return dotProduct / (Math.sqrt(normA) * Math.sqrt(normB));
}

// Async semantic search implementation
async function semanticSearch(query, documents) {
    try {
        // Generate query embedding
        const queryEmbedding = await embeddings.generateEmbeddingAsync(query);
        
        // Generate document embeddings
        const documentEmbeddings = await Promise.all(
            documents.map(doc => embeddings.generateEmbeddingAsync(doc))
        );
        
        // Calculate similarities
        const similarities = documentEmbeddings.map(docEmb => 
            cosineSimilarity(queryEmbedding, docEmb)
        );
        
        // Find best match
        const bestMatchIndex = similarities.indexOf(Math.max(...similarities));
        return {
            document: documents[bestMatchIndex],
            similarity: similarities[bestMatchIndex]
        };
    } catch (err) {
        console.error("Semantic search failed:", err);
        throw err;
    }
}

// Usage
const result = await semanticSearch("machine learning", [
    "Deep learning uses neural networks",
    "Cooking recipes for beginners", 
    "AI and machine learning concepts"
]);
```

### Error Handling and Best Practices

```javascript
// Always handle errors appropriately
try {
    const embedding = await embeddings.generateEmbeddingAsync(text);
    // Process embedding...
} catch (err) {
    console.error("Failed to generate embedding:", err);
    // Handle error appropriately
}

// Resource management - consider memory usage
const model = embeddings.getModel();
const memoryPerEmbedding = model.dimensions * 4; // 4 bytes per float32

// Calculate memory for batch processing
const batchSize = 1000;
const estimatedMemory = memoryPerEmbedding * batchSize;
console.log(`Estimated memory usage: ${estimatedMemory / 1024 / 1024} MB`);

// Use cancellation for long-running operations
let cancel;

function startEmbedding() {
    cancel = embeddings.generateEmbeddingWithCallbacks(text, {
        onSuccess: handleSuccess,
        onError: handleError
    });
}

function stopEmbedding() {
    if (cancel) {
        cancel();
        cancel = null;
    }
}
```

## Steps API

The Steps API provides JavaScript access to Geppetto's step abstraction through the watermill-based RuntimeEngine. Steps now use event-driven execution with automatic handler lifecycle management.

### Core Step Concepts

A Step in the new architecture:

1. **Event-driven execution**: Steps publish events to watermill topics for real-time feedback
2. **Automatic handler management**: Each step gets a unique handler that auto-registers/unregisters
3. **Streaming support**: Real-time event flow through watermill pub/sub system
4. **Per-step topics**: Each step execution gets its own watermill topic for isolation
5. **Carries metadata**: Events include both step and event metadata

#### Key Step Characteristics

##### 1. Event Streaming
Steps now stream events through watermill rather than returning results directly:

```javascript
const stepID = step.runWithEvents(input, function(event) {
    console.log("Event type:", event.type);
    console.log("Event data:", event);
});

console.log("Step ID:", stepID);
```

This event-driven approach enables real-time feedback and better observability.

##### 2. Event Types
Steps publish various event types during execution:

- **`start`**: Step execution begins
- **`partial-completion`**: Incremental results (for streaming steps)
- **`final`**: Step completed successfully with final result
- **`error`**: Step encountered an error
- **`interrupt`**: Step was interrupted/cancelled
- **`tool-call`**: AI step is calling a tool (for AI steps)
- **`tool-result`**: Tool call completed (for AI steps)

##### 3. Automatic Cleanup
Each step execution automatically:
- Registers a handler on the watermill router
- Creates a unique topic (`step.{stepID}`)
- Unregisters the handler when the step completes
- Cleans up subscriptions and resources

### Step Execution API

Steps provide the new `runWithEvents` method for watermill-based execution:

#### Event-driven API
The primary method for executing steps with real-time event streaming:

```javascript
const stepID = step.runWithEvents(input, function(event) {
    // Handle different event types
    switch(event.type) {
        case "start":
            console.log("Step started");
            break;
            
        case "partial-completion":
            console.log("Partial result:", event.delta);
            console.log("Full completion so far:", event.completion);
            break;
            
        case "final":
            console.log("Final result:", event.text);
            break;
            
        case "error":
            console.error("Step failed:", event.error);
            break;
            
        case "tool-call":
            console.log("Tool call:", event.toolCall.name, event.toolCall.input);
            break;
            
        case "tool-result":
            console.log("Tool result:", event.toolResult.result);
            break;
    }
});

console.log("Step execution started with ID:", stepID);
```

#### Event Object Structure
Events have a common structure with type-specific fields:

```javascript
// Common fields
{
    type: string,           // Event type
    meta: {                 // Event metadata
        messageId: string,
        parentId: string,
        engine: string
    },
    step: {                 // Step metadata
        stepId: string,
        type: string,
        inputType: string,
        outputType: string,
        metadata: object
    }
}

// Type-specific fields for partial-completion
{
    ...commonFields,
    delta: string,          // New text added
    completion: string      // Full text so far
}

// Type-specific fields for final
{
    ...commonFields,
    text: string           // Final result text
}

// Type-specific fields for tool-call
{
    ...commonFields,
    toolCall: {
        id: string,
        name: string,
        input: object
    }
}

// Type-specific fields for tool-result
{
    ...commonFields,
    toolResult: {
        id: string,
        result: any
    }
}
```

### Step Registration and Setup

Steps are now registered through setup functions rather than direct registration:

#### Creating Setup Functions
```go
// In Go - create a setup function
func SetupMyStep() js.SetupFunction {
    return func(vm *goja.Runtime, engine *js.RuntimeEngine) {
        // Create your step
        myStep := &MyStep{...}
        
        // Create watermill step object factory
        stepFactory := js.CreateWatermillStepObject(
            engine,
            myStep,
            inputConverter,
            outputConverter,
        )
        
        // Register in VM
        stepObj := stepFactory(vm)
        vm.Set("myStep", stepObj)
    }
}

// Add to engine
engine.AddSetupFunction(SetupMyStep())
```

#### Built-in Setup Functions
Geppetto provides several built-in setup functions:

```go
// Basic test step
engine.AddSetupFunction(js.SetupDoubleStep())

// Conversation API
engine.AddSetupFunction(js.SetupConversation())

// Embeddings API
engine.AddSetupFunction(js.SetupEmbeddings(stepSettings))

// Done callback for script completion
engine.AddSetupFunction(js.SetupDoneCallback())
```

### Watermill Integration Details

#### Topic Management
- Each step execution gets a unique topic: `step.{stepID}`
- Topics are automatically created and cleaned up
- Events are published to the step's topic during execution

#### Handler Lifecycle
1. **Registration**: Handler registered when `runWithEvents` is called
2. **Event Processing**: Handler receives and processes events from watermill
3. **JavaScript Callback**: Events converted to JavaScript and passed to callback
4. **Cleanup**: Handler automatically unregistered when step completes

#### Error Handling
```javascript
const stepID = step.runWithEvents(input, function(event) {
    if (event.type === "error") {
        console.error("Step failed:", event.error);
        // Handle error appropriately
        return;
    }
    
    // Process successful events
    console.log("Event:", event.type, event);
});
```

### Migration from Legacy API

The new watermill-based API replaces the previous Promise/callback patterns:

#### Legacy Pattern (Removed)
```javascript
// Old Promise-based API (no longer available)
const results = await step.startAsync(input);

// Old callback API (no longer available)
const cancel = step.startWithCallbacks(input, {
    onResult: (result) => console.log(result),
    onError: (error) => console.error(error)
});
```

#### New Watermill Pattern
```javascript
// New event-driven API
const stepID = step.runWithEvents(input, function(event) {
    switch(event.type) {
        case "final":
            console.log("Result:", event.text);
            break;
        case "error":
            console.error("Error:", event.error);
            break;
    }
});
```

### Advanced Patterns

#### Event Filtering
```javascript
const stepID = step.runWithEvents(input, function(event) {
    // Only handle specific event types
    if (event.type === "partial-completion") {
        updateUI(event.delta);
    } else if (event.type === "final") {
        showFinalResult(event.text);
    }
});
```

#### Event Aggregation
```javascript
let fullText = "";

const stepID = step.runWithEvents(input, function(event) {
    if (event.type === "partial-completion") {
        // Build up complete text from deltas
        fullText += event.delta;
        console.log("Current text length:", fullText.length);
    } else if (event.type === "final") {
        console.log("Final text:", fullText);
    }
});
```

#### Step Monitoring
```javascript
const stepStats = {
    startTime: null,
    eventCount: 0,
    errors: []
};

const stepID = step.runWithEvents(input, function(event) {
    stepStats.eventCount++;
    
    if (event.type === "start") {
        stepStats.startTime = Date.now();
    } else if (event.type === "error") {
        stepStats.errors.push(event.error);
    } else if (event.type === "final") {
        const duration = Date.now() - stepStats.startTime;
        console.log("Step completed in", duration, "ms");
        console.log("Total events:", stepStats.eventCount);
    }
});
```

## RuntimeEngine Setup and Configuration

The RuntimeEngine provides a modular approach to setting up the JavaScript environment through setup functions.

### Engine Lifecycle

```go
// 1. Create engine (does not start event loop)
engine := js.NewRuntimeEngine()
defer engine.Close()

// 2. Add setup functions
engine.AddSetupFunction(js.SetupDoubleStep())
engine.AddSetupFunction(js.SetupConversation())
engine.AddSetupFunction(js.SetupEmbeddings(stepSettings))

// 3. Start engine and run setup (blocking until completion)
engine.Start()

// Or add custom JavaScript execution
engine.AddSetupFunction(func(vm *goja.Runtime, engine *js.RuntimeEngine) {
    _, err := vm.RunString("console.log('Custom setup complete');")
    if err != nil {
        panic(err)
    }
})
```

### Setup Functions

Setup functions allow modular configuration of the JavaScript environment:

#### Built-in Setup Functions

```go
// Test step that doubles numbers
js.SetupDoubleStep()

// Conversation management API
js.SetupConversation()

// Embeddings generation API
js.SetupEmbeddings(stepSettings)

// Done callback for script completion signaling
js.SetupDoneCallback()
```

#### Custom Setup Functions

```go
func MyCustomSetup() js.SetupFunction {
    return func(vm *goja.Runtime, engine *js.RuntimeEngine) {
        // Set up custom JavaScript objects
        customObj := vm.NewObject()
        customObj.Set("version", "1.0.0")
        customObj.Set("author", "My App")
        vm.Set("myApp", customObj)
        
        // Register custom steps
        myStep := &MyCustomStep{}
        stepFactory := js.CreateWatermillStepObject(
            engine,
            myStep,
            func(v goja.Value) MyInput { /* convert input */ },
            func(output MyOutput) goja.Value { /* convert output */ },
        )
        stepObj := stepFactory(vm)
        vm.Set("myCustomStep", stepObj)
    }
}

// Add to engine
engine.AddSetupFunction(MyCustomSetup())
```

### Console and Utilities

The RuntimeEngine automatically sets up basic console functionality:

```javascript
// Available in all scripts
console.log("Hello", "World");
console.error("Error message");

// Done callback (when SetupDoneCallback is used)
done(); // Signals script completion
```

### Event Loop Management

The RuntimeEngine properly manages the Goja event loop:

- **`engine.Start()`**: Calls `Loop.Run()` with all setup functions
- **Setup Phase**: All setup functions called within the event loop
- **Execution Phase**: Scripts can be executed via setup functions
- **Cleanup Phase**: Event loop terminates when all work is complete

### Error Handling

```go
// Setup functions can handle errors
func MySetupWithErrorHandling() js.SetupFunction {
    return func(vm *goja.Runtime, engine *js.RuntimeEngine) {
        defer func() {
            if r := recover(); r != nil {
                log.Error().Interface("panic", r).Msg("Setup function panicked")
            }
        }()
        
        // Setup code that might fail
        _, err := vm.RunString("potentially.failing.code();")
        if err != nil {
            log.Error().Err(err).Msg("JavaScript execution failed in setup")
            // Handle error appropriately
        }
    }
}
```

## Integration with Go Steps

The JavaScript Step API provides a direct mapping to Go's Step implementation, enabling seamless integration:

- **Go channels** → JavaScript async iterators/callbacks
- **Go context cancellation** → JavaScript AbortController
- **Go generics** → JavaScript type definitions
- **Go error handling** → JavaScript promises/try-catch

### Type Conversion Details

Input and output values are converted between Go and JavaScript using converter functions:

```go
// Example for float64 type
inputConverter := func(v goja.Value) float64 {
    return v.ToFloat()
}
outputConverter := func(v float64) goja.Value {
    return runtime.ToValue(v)
}
```

### Event Loop Considerations

When using the JavaScript API:

1. **Event loop must be started** before using Promises and stopped when done
2. **All Promise resolutions** must happen on the event loop using `RunOnLoop`
3. **Event loop is not goroutine-safe** - all JS operations must happen on the loop
4. **Long-running operations** should be executed in goroutines to avoid blocking
5. **Error handling** should account for both Step errors and Promise rejection

**Critical Implementation Detail**: All JavaScript callbacks and Promise resolutions must happen on the event loop:

```go
loop.RunOnLoop(func(*goja.Runtime) {
    // Resolve or reject Promise
    resolve(w.runtime.ToValue(result))
})
```

### Promise-based Execution Example

```go
import (
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/eventloop"
)

// Create event loop
loop := eventloop.NewEventLoop()
loop.Start()
defer loop.Stop()

// Run step in event loop
loop.RunOnLoop(func(vm *goja.Runtime) {
    // Create promise and get resolver
    p, resolve, reject := vm.NewPromise()
    
    // Start step execution
    go func() {
        result, err := myStep.Start(ctx, input)
        if err != nil {
            // Must resolve/reject on the event loop
            loop.RunOnLoop(func(*goja.Runtime) {
                reject(vm.ToValue(err.Error()))
            })
            return
        }
        
        // Process results
        for r := range result.GetChannel() {
            if r.Error() != nil {
                loop.RunOnLoop(func(*goja.Runtime) {
                    reject(vm.ToValue(r.Error().Error()))
                })
                return
            }
            
            // Resolve with success value
            loop.RunOnLoop(func(*goja.Runtime) {
                resolve(vm.ToValue(r.Unwrap()))
            })
        }
    }()
    
    // Make promise available to JS code
    vm.Set("stepPromise", p)
})
```

## Error Types and Handling

Common errors across all APIs:

### Embeddings API Errors
1. Invalid input text
2. Provider API errors (rate limiting, authentication)
3. Network connectivity issues
4. Model loading errors
5. Cancellation errors

### Steps API Errors
1. Step initialization failures
2. Input validation errors
3. Execution runtime errors
4. Context cancellation
5. Resource exhaustion

### Chat API Errors
1. Invalid conversation format
2. Model configuration errors
3. Token limit exceeded
4. API authentication failures
5. Rate limiting and quota errors

### Conversation API Errors
1. Invalid message format
2. Missing required fields
3. Unsupported image formats
4. File access errors
5. Metadata validation errors

## Best Practices

### 1. Resource Management
```javascript
// Always clean up resources
const cancel = operation.startWithCallbacks(input, callbacks);

// Clean up when done
window.addEventListener('beforeunload', () => {
    if (cancel) cancel();
});
```

### 2. Error Handling
```javascript
// Use appropriate error handling for each API
try {
    const result = await api.operationAsync(input);
    // Process result
} catch (err) {
    console.error("Operation failed:", err);
    // Handle error appropriately
}
```

### 3. Performance Optimization
```javascript
// Batch similar requests
const results = await Promise.all(
    inputs.map(input => api.operationAsync(input))
);

// Use streaming for large datasets
const cancel = api.operationWithCallbacks(largeInput, {
    onResult: processChunk,
    onDone: finalizeResults
});
```

### 4. Memory Management
```javascript
// Consider memory usage for embeddings
const model = embeddings.getModel();
const memoryPerEmbedding = model.dimensions * 4; // bytes

// Use cancellation to prevent unnecessary work
const controller = new AbortController();
const promise = api.operationAsync(input, { 
    signal: controller.signal 
});
```

### 5. Choosing the Right API
- Use **synchronous APIs** for simple, fast operations
- Use **Promise APIs** for single results with proper error handling
- Use **callback APIs** when you need streaming or cancellation support
- Use **Conversation objects** instead of raw message arrays for chat operations

This comprehensive JavaScript API enables powerful AI application development while maintaining clean separation between JavaScript application logic and Go-based AI infrastructure. The consistent patterns across all APIs make it easy to build complex, responsive applications that can handle streaming data, provide real-time feedback, and gracefully handle errors and cancellation.
