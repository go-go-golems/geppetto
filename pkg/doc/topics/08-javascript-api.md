---
Title: JavaScript API for Geppetto - Complete Reference Guide
Slug: geppetto-javascript-api
Short: A comprehensive guide to using Geppetto's JavaScript API for conversations, embeddings, steps, and chat functionality through Goja bindings.
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
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# JavaScript API for Geppetto - Complete Reference Guide

This guide provides comprehensive documentation for Geppetto's JavaScript API, which allows you to interact with conversations, generate embeddings, execute steps, and perform AI chat operations from JavaScript code through Goja bindings.

## Overview

Geppetto's JavaScript API exposes four main components:

1. **Conversation API**: Create and manage conversations with messages, tool uses, and tool results
2. **Embeddings API**: Generate vector embeddings from text using various embedding models
3. **Steps API**: Execute asynchronous computation steps with streaming, cancellation, and composition
4. **Chat Step Factory**: Create and manage chat completion steps for AI interactions

These APIs provide both synchronous and asynchronous interfaces, with full support for streaming, error handling, and cancellation patterns.

## Core Architecture

### JavaScript-Go Integration

The JavaScript API is built on top of Goja, a JavaScript engine for Go, with special integration for Node.js-style event loops. This enables:

- **Promise-based async operations**: Using goja_nodejs eventloop package
- **Streaming results**: Real-time data processing with callbacks
- **Proper cancellation**: Context-based cancellation from Go propagated to JavaScript
- **Type conversion**: Seamless conversion between Go and JavaScript types
- **Event loop integration**: All callbacks and Promise resolutions happen on the event loop

### Common Patterns

All APIs follow consistent patterns:

```javascript
// Promise-based API for single results
const result = await api.operationAsync(input);

// Synchronous API for immediate results
const result = api.operationBlocking(input);

// Callback-based API for streaming results
const cancel = api.operationWithCallbacks(input, {
    onResult: (result) => { /* handle result */ },
    onError: (error) => { /* handle error */ },
    onDone: () => { /* operation complete */ },
    onCancel: () => { /* operation cancelled */ }
});
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

The Steps API provides JavaScript access to Geppetto's step abstraction - a powerful system for asynchronous computation that combines features of async operations and list monads.

### Core Step Concepts

A Step represents a computation that:

1. **Takes a single input** and produces zero or more outputs asynchronously
2. **Can be cancelled** at any point during execution
3. **Can be composed** with other steps to create pipelines
4. **Supports streaming results** for real-time feedback
5. **Carries metadata** about its execution

#### Key Step Characteristics

##### 1. Multiple Results
A Step can produce multiple results over time, similar to an async iterator:

```javascript
const step = steps.createMapStep((x) => x * 2);
const cancel = step.startWithCallbacks([1, 2, 3], {
    onResult: (result) => console.log(result) // Prints: 2, 4, 6
});
```

This multiple-result capability enables streaming processing where results are delivered incrementally rather than all at once.

##### 2. Composition
Steps can be chained together using `bind` operations:

```javascript
// Create a pipeline that:
// 1. Generates embeddings
// 2. Searches similar documents
// 3. Summarizes results with an LLM
const pipeline = steps.compose([
    embedStep,
    (embeddings) => searchStep.startAsync(embeddings),
    (documents) => llmStep.startAsync({
        messages: [{
            role: "user",
            content: `Summarize these documents: ${documents.join('\n')}`
        }]
    })
]);
```

Steps are particularly useful for:
- LLM interactions with streaming responses
- Data processing pipelines
- Parallel computations
- Operations requiring cancellation
- Event-driven processing

### Step Execution APIs

Each registered step provides three execution methods:

#### Promise-based API
Best for single-result operations or when you want to wait for all results:

```javascript
// Async/await style
try {
    const promise = step.startAsync(input);
    console.log("Promise created");
    const results = await promise;
    console.log("Results:", results);
} catch (err) {
    console.error("Error:", err);
}
```

#### Synchronous API
Use when you need blocking behavior and have all results immediately:

```javascript
try {
    const results = step.startBlocking(input);
    console.log("Results:", results);
} catch (err) {
    console.error("Error:", err);
}
```

#### Callback-based Streaming API
Best for handling streaming results or long-running operations:

```javascript
const cancel = step.startWithCallbacks(input, {
    onResult: (result) => {
        console.log("Got result:", result);
    },
    onError: (err) => {
        console.error("Error occurred:", err);
    },
    onDone: () => {
        console.log("Processing complete");
    },
    onCancel: () => {
        console.log("Operation cancelled");
    }
});

// Cancel the operation when needed
setTimeout(() => {
    cancel();
}, 5000);
```

### Step Composition

Steps can be chained together using composition patterns:

```javascript
// Create a pipeline that:
// 1. Generates embeddings
// 2. Searches similar documents  
// 3. Summarizes results with an LLM
const pipeline = steps.compose([
    embedStep,
    (embeddings) => searchStep.startAsync(embeddings),
    (documents) => llmStep.startAsync({
        messages: [{
            role: "user",
            content: `Summarize these documents: ${documents.join('\n')}`
        }]
    })
]);
```

### Cancellation Support

All step operations support cancellation:

```javascript
// With callbacks
const cancel = step.startWithCallbacks(input, callbacks);
// Later...
cancel();

// With promises using AbortController
const controller = new AbortController();
const promise = step.startAsync(input, { signal: controller.signal });
// Later...
controller.abort();
```

### Metadata and Results

Steps carry metadata about their execution:

```javascript
const result = await step.startAsync(input);
console.log(result.metadata); // Execution details, timing, etc.
```

### Step Registration Implementation

Steps are registered using the `RegisterStep` function in Go:

```go
func RegisterStep[T any, U any](
    runtime *goja.Runtime,
    loop *eventloop.EventLoop,
    name string,
    step steps.Step[T, U],
    inputConverter func(goja.Value) T,
    outputConverter func(U) goja.Value,
) error
```

This registration process:
- Integrates with the JavaScript event loop for proper Promise handling
- Provides type conversion between Go and JavaScript
- Enables all three execution APIs (async, blocking, callbacks)
- Handles error propagation and cancellation

### Step Factory Pattern

Steps can be created from factories for reusability:

```javascript
// Create a reusable step factory
const createProcessingStep = (options) => steps.createStep({
    input: options.preprocessor,
    process: options.processor,
    output: options.postprocessor
});

// Create instances with different configurations
const textStep = createProcessingStep({
    preprocessor: (text) => text.toLowerCase(),
    processor: async (text) => /* process */,
    postprocessor: (result) => result.toString()
});
```

### Advanced Step Patterns

#### Event Publishing and Monitoring
Steps can publish events for monitoring and debugging:

```javascript
const monitoredStep = steps.withEvents(step, {
    onStart: (input) => console.log("Starting with:", input),
    onResult: (result) => console.log("Produced:", result),
    onComplete: () => console.log("Step completed"),
    onError: (err) => console.error("Step failed:", err)
});
```

#### Parallel Processing
```javascript
// Process multiple inputs in parallel
const parallelStep = steps.createParallelStep({
    maxConcurrency: 3,
    step: processingStep
});

const results = await parallelStep.startAsync(inputs);
```

#### Error Recovery
```javascript
const robustStep = steps.withRetry(step, {
    maxAttempts: 3,
    backoff: (attempt) => Math.pow(2, attempt) * 1000,
    shouldRetry: (error) => error.isTransient
});
```

#### State Management
```javascript
const statefulStep = steps.withState({
    initialState: { count: 0 },
    step: (input, state) => {
        state.count++;
        return processWithState(input, state);
    }
});
```

## Chat Step Factory

The Chat Step Factory provides a specialized interface for creating chat completion steps that integrate with various LLM providers.

### Basic Usage

```javascript
// Create a factory instance
const factory = new ChatStepFactory();

// Create a new chat step
const step = factory.newStep();

// Use Promise-based API
step.startAsync({ 
    messages: [
        { role: "user", content: "Hello, how can I help you?" }
    ]
})
.then(result => {
    console.log("Response:", result);
})
.catch(err => {
    console.error("Error:", err);
});
```

### Streaming Chat Responses

Chat steps excel at streaming responses for real-time output:

```javascript
step.startWithCallbacks(
    { 
        messages: [
            { role: "user", content: "Explain quantum computing" }
        ]
    },
    {
        onResult: (result) => {
            console.log("Got chunk:", result);
            // Display streaming text in UI
        },
        onError: (err) => {
            console.error("Error occurred:", err);
        },
        onDone: () => {
            console.log("Chat complete");
        }
    }
);
```

### Conversation Integration

The Chat Step Factory supports two input formats:

#### Using Conversation Objects (Recommended)
```javascript
const conv = new Conversation();

// Add messages with full conversation management
conv.AddMessage("system", "You are a helpful assistant");
conv.AddMessage("user", "What is quantum computing?");

// Add messages with metadata
conv.AddMessage("user", "Hello", {
    metadata: { source: "user-input" },
    time: "2024-03-20T15:04:05Z"
});

// Add messages with images
conv.AddMessageWithImage("user", "What's in this image?", "path/to/image.jpg");

// Add tool usage
conv.AddToolUse("tool123", "search", { query: "quantum computing" });
conv.AddToolResult("tool123", "Found relevant articles...");

// Use with chat step
const response = await step.startAsync(conv);
```

#### Legacy Format (Backward Compatibility)
```javascript
const input = {
    messages: [
        { role: "system", content: "You are a helpful assistant" },
        { role: "user", content: "What is quantum computing?" }
    ]
};

const response = await step.startAsync(input);
```

### Custom Configuration

Create steps with custom options:

```javascript
const step = factory.newStep([
    (step) => {
        // Custom option function that can modify the step
        // Return null or undefined if no error
        // Return an error if something goes wrong
        return null;
    }
]);
```

### Cancellation Support

Chat operations support cancellation through AbortController:

```javascript
const controller = new AbortController();

step.startAsync(input, { signal: controller.signal })
    .then(result => {
        console.log("Success:", result);
    })
    .catch(err => {
        if (err.name === 'AbortError') {
            console.log("Operation cancelled");
        } else {
            console.error("Error:", err);
        }
    });

// Cancel the operation
setTimeout(() => {
    controller.abort();
}, 5000);
```

### Complete Chat Application Example

```javascript
// Create factory and step
const factory = new ChatStepFactory();
const chatStep = factory.newStep([
    {
        temperature: 0.7,
        maxTokens: 2000
    }
]);

// Create and setup conversation
const conversation = new Conversation();
conversation.AddMessage("system", 
    "You are a helpful AI assistant. Be concise and clear in your responses."
);

// Function to send user message and get response
async function chat(userInput) {
    // Add user message
    conversation.AddMessage("user", userInput);
    
    // Stream response
    let response = "";
    const cancel = chatStep.startWithCallbacks(conversation, {
        onResult: (chunk) => {
            response += chunk;
            // Update UI with streaming response
            updateUI(response);
        },
        onError: (err) => {
            console.error("Chat error:", err);
            handleError(err);
        },
        onDone: () => {
            // Add assistant's response to conversation
            conversation.AddMessage("assistant", response);
            // Update UI to show completion
            markComplete();
        }
    });
    
    // Return cancel function for cleanup
    return cancel;
}

// Usage
const cancelChat = await chat("Explain quantum computing briefly");

// Cancel if needed
setTimeout(() => {
    cancelChat();
}, 10000);
```

### Input Format

The input object should follow this structure:

```javascript
{
    messages: [
        {
            role: string,    // "system", "user", "assistant"
            content: string  // The message content
        }
    ],
    // Additional configuration options depending on the chat model
    temperature?: number,
    maxTokens?: number,
    // etc...
}
```

### Error Handling Best Practices

```javascript
// With callbacks
chatStep.startWithCallbacks(conversation, {
    onResult: (chunk) => { /* ... */ },
    onError: (err) => {
        console.error("LLM error:", err);
        // Handle specific error cases
        if (err.includes("rate limit")) {
            // Handle rate limiting
        } else if (err.includes("context length")) {
            // Handle context length errors
        }
    }
});

// With promises
try {
    await chatStep.startAsync(conversation);
} catch (err) {
    if (err.includes("context length")) {
        // Handle context length errors
    } else if (err.includes("invalid api key")) {
        // Handle authentication errors
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
