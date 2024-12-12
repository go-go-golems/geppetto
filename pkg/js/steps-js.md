# JavaScript Step API

## What is a Step?

A Step is a fundamental abstraction for asynchronous computation that combines features of both async operations and list monads. It represents a computation that:

1. Takes a single input
2. Produces zero or more outputs asynchronously
3. Can be cancelled
4. Can be composed with other steps
5. Supports streaming results
6. Carries metadata about its execution

Steps are particularly useful for:
- LLM interactions (streaming responses)
- Data processing pipelines
- Parallel computations
- Operations that need cancellation
- Event-driven processing

### Core Step Concepts

#### 1. Multiple Results
A Step can produce multiple results over time, similar to an async iterator:

```javascript
const step = steps.createMapStep((x) => x * 2);
const cancel = step.startWithCallbacks([1, 2, 3], {
    onResult: (result) => console.log(result) // Prints: 2, 4, 6
});
```

#### 2. Composition
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

#### 3. Cancellation
All step operations support cancellation:

```javascript
// With callbacks
const cancel = step.startWithCallbacks(input, callbacks);
// Later...
cancel();

// With promises
const controller = new AbortController();
const promise = step.startAsync(input, { signal: controller.signal });
// Later...
controller.abort();
```

#### 4. Metadata
Steps carry metadata about their execution:

```javascript
const result = await step.startAsync(input);
console.log(result.metadata); // Execution details, timing, etc.
```

## API Overview

Each registered step provides three ways to interact with the underlying Go Step:

1. Promise-based async API (`startAsync`)
2. Synchronous blocking API (`startBlocking`) 
3. Callback-based streaming API (`startWithCallbacks`)

### Promise-based API

Best for single-result operations or when you want to wait for all results:

```javascript
// Async/await style
try {
    const promise = doubleStep.startAsync(input);
    console.log("Promise created");
    const results = await promise;
    console.log("Results:", results);
} catch (err) {
    console.error("Error:", err);
}
```

### Synchronous API

Use when you need blocking behavior and have all results immediately:

```javascript
try {
    const results = step.startBlocking(input);
    console.log("Results:", results);
} catch (err) {
    console.error("Error:", err);
}
```

### Callback-based Streaming API

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

## Implementation Details

### Step Registration

Steps are registered using the `RegisterStep` function:

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

The function requires:
- A Goja runtime
- An event loop for proper Promise handling
- A name for the step in JavaScript
- The Go Step implementation
- Converter functions for input/output types

### Event Loop Integration

All JavaScript callbacks and Promise resolutions must happen on the event loop:

```go
loop.RunOnLoop(func(*goja.Runtime) {
    // Resolve or reject Promise
    resolve(w.runtime.ToValue(result))
})
```

### Type Conversion

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

### Error Handling

Errors are properly propagated to JavaScript:
- Promise rejections for async API
- Exceptions for blocking API
- Error callbacks for streaming API

## Example Use Cases

### 1. Embeddings Generation

```javascript
// Register the embeddings step
const embeddingsStep = embeddings.createStep({
    model: "text-embedding-3-small",
    dimensions: 1536
});

// Promise-based usage
const vectors = await embeddingsStep.startAsync("Hello, world!");
console.log("Embedding vectors:", vectors);

// Streaming usage for batch processing
const cancel = embeddingsStep.startWithCallbacks(
    ["Hello", "World", "!"],
    {
        onResult: (embedding) => {
            console.log("Got embedding:", embedding);
        },
        onDone: () => {
            console.log("All embeddings generated");
        }
    }
);
```

### 2. LLM Chat Completion

```javascript
// Register the chat step
const chatStep = claude.createStep({
    model: "claude-3-opus-20240229",
    temperature: 0.7
});

// Stream responses
const cancel = chatStep.startWithCallbacks(
    {
        messages: [
            { role: "user", content: "Explain quantum computing" }
        ]
    },
    {
        onResult: (chunk) => {
            process.stdout.write(chunk);
        },
        onError: (err) => {
            console.error("Chat error:", err);
        },
        onDone: () => {
            console.log("\nChat complete");
        }
    }
);
```

### 3. Custom Processing Step

```javascript
// Create a custom processing step
const processingStep = steps.createLambdaStep(
    (input) => {
        // Process input
        return processedResult;
    }
);

// Use it in a pipeline
const results = await processingStep.startAsync(data);
```

## Best Practices

1. **Event Loop Usage**:
   - Always use the event loop for JavaScript callbacks
   - Resolve/reject Promises on the event loop
   - Keep JavaScript operations on the event loop thread

2. **Error Handling**:
   - Properly propagate Go errors to JavaScript
   - Use appropriate error handling for each API
   - Include error details in rejections/callbacks

3. **Resource Cleanup**:
   - Cancel operations when no longer needed
   - Clean up resources in onDone/onCancel
   - Use defer for cleanup in Go code

4. **Type Safety**:
   - Provide proper type converters
   - Validate types before conversion
   - Handle conversion errors gracefully

5. **Performance**:
   - Use streaming for large datasets
   - Avoid blocking the event loop
   - Consider batching for multiple operations

## Advanced Concepts

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

### Event Publishing

Steps can publish events for monitoring and debugging:

```javascript
const monitoredStep = steps.withEvents(step, {
    onStart: (input) => console.log("Starting with:", input),
    onResult: (result) => console.log("Produced:", result),
    onComplete: () => console.log("Step completed"),
    onError: (err) => console.error("Step failed:", err)
});
```

### Parallel Processing

Steps support parallel execution patterns:

```javascript
// Process multiple inputs in parallel
const parallelStep = steps.createParallelStep({
    maxConcurrency: 3,
    step: processingStep
});

// Use with array of inputs
const results = await parallelStep.startAsync(inputs);
```

### Error Recovery

Steps can include error recovery logic:

```javascript
const robustStep = steps.withRetry(step, {
    maxAttempts: 3,
    backoff: (attempt) => Math.pow(2, attempt) * 1000,
    shouldRetry: (error) => error.isTransient
});
```

### State Management

Steps can maintain state across executions:

```javascript
const statefulStep = steps.withState({
    initialState: { count: 0 },
    step: (input, state) => {
        state.count++;
        return processWithState(input, state);
    }
});
```

## Integration with Go Steps

The JavaScript Step API is a direct mapping to Go's Step implementation:

- Go's channels → JavaScript async iterators/callbacks
- Go's context cancellation → JavaScript AbortController
- Go's generics → JavaScript type definitions
- Go's error handling → JavaScript promises/try-catch

This allows seamless integration between Go and JavaScript code while maintaining the same programming model. 

The Step abstraction can be used from JavaScript code through Goja bindings. This enables writing processing steps in JavaScript while maintaining the same async patterns.

### Promise-based Execution

Using the goja_nodejs eventloop package, Steps can be executed asynchronously using Promises:

```go
import (
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/eventloop"
    "github.com/go-go-golems/geppetto/pkg/steps"
)

// Create event loop
loop := eventloop.NewEventLoop()
loop.Start()
defer loop.Stop()

// Create and register step
myStep := NewMyStep(config)

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

The JavaScript code can then use the Promise:

```javascript
stepPromise
    .then(result => {
        console.log("Step completed:", result);
    })
    .catch(err => {
        console.error("Step failed:", err);
    });
```

### Event Loop Considerations

1. The event loop must be started before using Promises and stopped when done
2. All Promise resolutions must happen on the event loop using `RunOnLoop`
3. The event loop is not goroutine-safe - all JS operations must happen on the loop
4. Long-running operations should be executed in goroutines to avoid blocking the loop
5. Error handling should account for both Step errors and Promise rejection

### Cancellation

Steps started through Promises can be cancelled using the context:

```go
ctx, cancel := context.WithCancel(context.Background())

loop.RunOnLoop(func(vm *goja.Runtime) {
    p, resolve, reject := vm.NewPromise()
    
    go func() {
        result, err := myStep.Start(ctx, input)
        // ... handle results ...
    }()
    
    // Cancel after timeout
    go func() {
        time.Sleep(5 * time.Second)
        cancel()
    }()
    
    vm.Set("stepPromise", p)
})
```

The JavaScript code will receive the cancellation as a rejected Promise:

```javascript
stepPromise
    .catch(err => {
        console.error("Step was cancelled:", err);
    });
```

The Step abstraction combined with goja_nodejs provides a robust way to bridge Go and JavaScript async operations while maintaining proper error handling and cancellation support. 