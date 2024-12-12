# JavaScript Chat Step Factory

This package provides a JavaScript wrapper for creating and managing chat steps in Geppetto. It allows you to create chat steps with various configurations and use them in a JavaScript environment.

## Usage

### Basic Usage

```javascript
// Create a new factory instance
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

// Use callback-based API for streaming
step.startWithCallbacks(
    { 
        messages: [
            { role: "user", content: "Hello, how can I help you?" }
        ]
    },
    {
        onResult: (result) => {
            console.log("Got chunk:", result);
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

### With Custom Options

```javascript
// Create step with custom options
const step = factory.newStep([
    (step) => {
        // Custom option function that can modify the step
        // Return null or undefined if no error
        // Return an error if something goes wrong
        return null;
    }
]);
```

### Error Handling

```javascript
// Using Promise API
step.startAsync(input)
    .then(result => {
        console.log("Success:", result);
    })
    .catch(err => {
        console.error("Error:", err);
    });

// Using callback API
step.startWithCallbacks(
    input,
    {
        onResult: (result) => {
            console.log("Result:", result);
        },
        onError: (err) => {
            console.error("Error:", err);
            // Handle error appropriately
        },
        onDone: () => {
            console.log("Complete");
        }
    }
);
```

### Cancellation

The chat step operations support cancellation through the JavaScript AbortController API when using promises:

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

## API Reference

### ChatStepFactory

Constructor for creating new chat step factories.

```javascript
const factory = new ChatStepFactory();
```

### factory.newStep([options])

Creates a new chat step with optional configuration options.

- `options`: Array of option functions that can modify the step
- Returns: Chat step object

### Chat Step Methods

#### startAsync(input)

Starts the chat step and returns a Promise.

- `input`: Object containing chat messages and configuration
- Returns: Promise that resolves with the chat result or rejects with an error

#### startWithCallbacks(input, callbacks)

Starts the chat step with callback-based handling.

- `input`: Object containing chat messages and configuration
- `callbacks`: Object containing callback functions:
  - `onResult(result)`: Called when a result chunk is available
  - `onError(error)`: Called when an error occurs
  - `onDone()`: Called when the operation completes
- Returns: undefined

## Input Format

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

## Error Handling

The wrapper provides comprehensive error handling through both promises and callbacks. Common error scenarios include:

- Invalid input format
- Network errors
- Model errors
- Rate limiting
- Context cancellation

Make sure to properly handle errors in your application code using either the Promise `.catch()` method or the `onError` callback. 