# Step Abstraction

The Step abstraction provides a way to compose asynchronous operations in a functional style, particularly useful for AI/LLM interactions and data processing pipelines. It implements a combination of the async monad and list monad patterns.

## Package Structure

The Step abstraction is implemented in:
- `github.com/go-go-golems/geppetto/pkg/steps` - Core step functionality
- `github.com/go-go-golems/geppetto/pkg/helpers` - Result type and utilities

Required imports:
```go
import (
    "context"
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/go-go-golems/geppetto/pkg/steps"
    "github.com/go-go-golems/geppetto/pkg/helpers"
)
```

## Core Concepts

### Step Interface

A Step represents a computation that:
- Takes an input of type T
- Produces multiple outputs of type U asynchronously
- Can be cancelled
- Can be composed with other steps

```go
type Step[T any, U any] interface {
    Start(ctx context.Context, input T) (StepResult[U], error)
    AddPublishedTopic(publisher message.Publisher, topic string) error
}
```

### Result Type

The `helpers.Result[T]` type is used to handle both successful values and errors:

```go
// From github.com/go-go-golems/geppetto/pkg/helpers
type Result[T any] interface {
    Error() error
    Unwrap() T
}

// Create results
successResult := helpers.NewValueResult[string]("success")
errorResult := helpers.NewErrorResult[string](err)
```

### StepResult

StepResult represents the output of a Step execution:

```go
type StepResult[T any] interface {
    Return() []helpers.Result[T]               
    GetChannel() <-chan helpers.Result[T]      
    Cancel()                           
    GetMetadata() *StepMetadata        
}

// Create a new StepResult
result := steps.NewStepResult[string](
    channel,
    steps.WithCancel[string](cancelFunc),
    steps.WithMetadata[string](&steps.StepMetadata{
        Type: "myStep",
        InputType: "string",
        OutputType: "string",
    }),
)
```

## Implementation Example

Here's a complete example of implementing a custom Step:

```go
type MyStep struct {
    // Step configuration
    config MyConfig
}

func NewMyStep(config MyConfig) *MyStep {
    return &MyStep{config: config}
}

// Implement the Step interface
func (s *MyStep) Start(ctx context.Context, input string) (steps.StepResult[string], error) {
    // Create output channel
    out := make(chan helpers.Result[string])
    
    // Create cancellable context
    ctx, cancel := context.WithCancel(ctx)
    
    // Start processing in goroutine
    go func() {
        defer close(out)
        
        // Process input
        result := processInput(input)
        
        select {
        case <-ctx.Done():
            return
        case out <- helpers.NewValueResult[string](result):
        }
    }()
    
    // Return StepResult
    return steps.NewStepResult[string](
        out,
        steps.WithCancel[string](cancel),
        steps.WithMetadata[string](&steps.StepMetadata{
            Type: "MyStep",
            InputType: "string",
            OutputType: "string",
        }),
    ), nil
}

func (s *MyStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
    // Implement if you need event publishing
    return nil
}
```

## Using Steps

### Composing Steps

```go
// Create steps
step1 := NewMyStep(config1)
step2 := NewOtherStep(config2)

// Start first step
result1, err := step1.Start(ctx, input)
if err != nil {
    return err
}

// Chain with second step
finalResult := steps.Bind(ctx, result1, step2)

// Process results
for result := range finalResult.GetChannel() {
    if result.Error() != nil {
        // Handle error
        continue
    }
    // Process result.Unwrap()
}
```

### Error Handling

```go
// Create error result
errorResult := steps.Reject[string](fmt.Errorf("something went wrong"))

// Create success result
successResult := steps.Resolve[string]("success")

// Create empty result
noneResult := steps.ResolveNone[string]()
```

### Context and Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := step.Start(ctx, input)
if err != nil {
    return err
}

// Cancel manually if needed
result.Cancel()
```

## Best Practices

1. Always use context for cancellation
2. Handle both success and error results using `helpers.Result`
3. Close channels when processing is complete
4. Use `steps.Bind` for composing steps
5. Implement proper cleanup in cancellation
6. Use metadata for debugging and monitoring
7. Consider using Watermill for event publishing

The Step abstraction provides a robust foundation for building asynchronous processing pipelines while maintaining type safety and proper error handling. 
