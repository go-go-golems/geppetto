# Goja - Javascript Golang Engine

## 1. Basic Usage
- Runtime initialization
- Running JavaScript code
- Error handling
- Interrupting execution
- Memory management considerations

## 2. Data Type Conversions
- Go to JavaScript conversions
- JavaScript to Go conversions 
- Working with primitive types
- Handling undefined/null values
- Type assertions and checks

## 3. Object Interactions
- Creating JavaScript objects from Go
- Accessing object properties
- Modifying object properties
- Working with arrays
- Handling JSON

## 4. Function Handling
- Exposing Go functions to JavaScript
- Calling JavaScript functions from Go
- Working with constructors
- Handling this context
- Managing function arguments

## 5. Error Handling & Exceptions
- Returning errors from golang
- Try/catch patterns
- Error types
- Stack traces
- Custom error handling
- Runtime errors vs compilation errors

## 6. Advanced Features
- Promises and async operations
- Proxies
- Symbol handling
- ArrayBuffers
- Custom object prototypes

## 7. Event Loop Termination Patterns

When working with the event loop, it's important to properly handle program termination. Here's a pattern for clean termination:

```go
func main() {
    // Create event loop and done channel
    loop := eventloop.NewEventLoop()
    done := make(chan error)
    loop.Start()
    defer loop.Stop()

    // Run your code in the loop
    loop.RunOnLoop(func(vm *goja.Runtime) {
        // Register done callback for JavaScript
        vm.Set("done", func(err ...goja.Value) {
            if len(err) > 0 && !goja.IsUndefined(err[0]) && !goja.IsNull(err[0]) {
                done <- fmt.Errorf("JavaScript error: %v", err[0])
                return
            }
            done <- nil
        })

        // Run your JavaScript code
        _, err := vm.RunString(`
            async function main() {
                try {
                    await someAsyncOperation();
                    done(); // Signal successful completion
                } catch (err) {
                    done(err); // Signal error
                }
            }
            main();
        `)
        if err != nil {
            done <- err
        }
    })

    // Wait for completion or error
    if err := <-done; err != nil {
        log.Error().Err(err).Msg("Error during execution")
        os.Exit(1)
    }
}
```

Key points:
1. Create a done channel to signal completion
2. Register a done callback in JavaScript
3. Use the callback to signal both successful completion and errors
4. Wait for the done signal before exiting
5. Properly handle cleanup with defer loop.Stop()

This pattern ensures:
- Clean program termination
- Proper error handling
- No goroutine leaks
- Graceful shutdown of the event loop

I'll help analyze the documentation outline and provide relevant information from the Goja documentation for each section.

# Documentation Analysis

## 1. Basic Usage

### Runtime initialization
```go
vm := goja.New() // Create new JavaScript runtime
```

### Running JavaScript code
```go
// Run string directly
val, err := vm.RunString("2 + 2")
if err != nil {
    panic(err)
}

// Or compile first for better performance
program, err := goja.Compile("calc.js", "2 + 2", false)
if err != nil {
    panic(err)
}
result, err := vm.RunProgram(program)
```

### Error handling
```go
vm := goja.New()
_, err := vm.RunString(`throw new Error("Custom error");`)
if err != nil {
    if jsErr, ok := err.(*goja.Exception); ok {
        fmt.Printf("JavaScript error: %v\n", jsErr.Value())
    }
}
```

### Interrupting execution
```go
vm := goja.New()
time.AfterFunc(200*time.Millisecond, func() {
    vm.Interrupt("halt")
})

_, err := vm.RunString(`while(true) {}`)
// err will be of type *InterruptedError
```

### Memory management considerations
Important notes from docs([1](https://pkg.go.dev/github.com/dop251/goja)):
- Runtime is not goroutine-safe
- Cannot pass object values between runtimes
- WeakMap/WeakRef have special implementation considerations due to Go runtime limitations

## 2. Data Type Conversions

### Go to JavaScript conversions
- Primitive types (numbers, strings, bool) convert to JS primitives
- Structs/maps convert to JS objects
- Slices/arrays convert to JS arrays
- Functions convert to JS functions
- nil converts to null

### JavaScript to Go conversions
```go
// Export to interface{}
val := vm.Get("someVar")
exported := val.Export()

// Export to specific type
var target string
err := vm.ExportTo(val, &target)
```

### Working with primitive types
Covered by automatic conversions between Go and JavaScript primitives.

### Handling undefined/null values
```go
val := vm.Get("someVar")
if goja.IsUndefined(val) || goja.IsNull(val) {
    // Handle undefined/null case
}
```

### Type assertions and checks
```go
if obj, ok := val.(*goja.Object); ok {
    // Handle object type
}

if fn, ok := goja.AssertFunction(val); ok {
    // Handle function type
}
```

## Detailed Type Conversions

### Go → JavaScript Conversions
```go
vm := goja.New()

// Primitive Types
vm.Set("number", 42)              // number
vm.Set("string", "hello")         // string
vm.Set("boolean", true)           // boolean
vm.Set("null", nil)               // null

// Structs
type Person struct {
    Name string
    Age  int
}
vm.Set("person", Person{"John", 30})  // converts to JS object

// Maps
vm.Set("map", map[string]interface{}{
    "key": "value",
    "num": 123,
})  // converts to JS object

// Slices/Arrays
vm.Set("slice", []int{1, 2, 3})       // converts to JS array
vm.Set("array", [3]string{"a", "b", "c"}) // converts to JS array

// Functions
vm.Set("goFunc", func(call goja.FunctionCall) goja.Value {
    return vm.ToValue("result")
}) // converts to JS function

// Channels (not directly supported)
// Must be wrapped in Go functions for use

// Interfaces
// Converted based on underlying type
```

### JavaScript → Go Conversions
```go
vm := goja.New()

// Primitive Types
numVal := vm.Get("num").ToInteger()     // int64
floatVal := vm.Get("num").ToFloat()     // float64
strVal := vm.Get("str").String()        // string
boolVal := vm.Get("bool").ToBoolean()   // bool

// Objects
obj := vm.Get("obj").ToObject(vm)
// Access properties:
prop := obj.Get("propertyName")

// Arrays
arr := vm.Get("arr").ToObject(vm)
length := arr.Get("length").ToInteger()
// Access elements:
element := arr.Get("0")

// Functions
fn, ok := goja.AssertFunction(vm.Get("fn"))
if ok {
    result, err := fn(goja.Undefined()) // Call with no args
}

// Type Checking
val := vm.Get("something")
switch {
case goja.IsUndefined(val):
    // handle undefined
case goja.IsNull(val):
    // handle null
case val.ExportType().Kind() == reflect.Func:
    // handle function
case val.ExportType().Kind() == reflect.Map:
    // handle map/object
case val.ExportType().Kind() == reflect.Slice:
    // handle array
}
```

## 3. Object Interactions

### Creating JavaScript objects from Go
```go
vm := goja.New()

// Create empty object
obj := vm.NewObject()
obj.Set("prop", "value")

// Create from struct
type Config struct {
    Host string
    Port int
}
config := &Config{"localhost", 8080}
vm.Set("config", config)

// Create from map
mapObj := vm.ToValue(map[string]interface{}{
    "name": "test",
    "data": []int{1, 2, 3},
}).ToObject(vm)
```

### Accessing and Modifying Properties
```go
vm := goja.New()
obj := vm.NewObject()

// Set properties
obj.Set("name", "value")
obj.Set("nested", map[string]interface{}{
    "key": "value",
})

// Get properties
val := obj.Get("name")
nestedObj := obj.Get("nested").ToObject(vm)
nestedVal := nestedObj.Get("key")

// Delete properties
obj.Delete("name")

// Check property existence
if obj.Has("prop") {
    // Property exists
}
```

## 4. Function Handling

### Exposing Go functions to JavaScript
```go
vm := goja.New()

// Simple function
vm.Set("greet", func(call goja.FunctionCall) goja.Value {
    name := call.Argument(0).String()
    return vm.ToValue("Hello, " + name)
})

// Method with this context
vm.Set("calculator", vm.NewObject())
calculator := vm.Get("calculator").ToObject(vm)
calculator.Set("add", func(call goja.FunctionCall) goja.Value {
    this := call.This
    x := call.Argument(0).ToInteger()
    y := call.Argument(1).ToInteger()
    return vm.ToValue(x + y)
})
```

### Calling JavaScript functions from Go
```go
vm := goja.New()
_, _ = vm.RunString(`
    function add(x, y) { 
        return x + y; 
    }
`)

// Get and call function
fn, ok := goja.AssertFunction(vm.Get("add"))
if ok {
    result, err := fn(goja.Undefined(), vm.ToValue(2), vm.ToValue(3))
    if err != nil {
        // Handle error
    }
    fmt.Println(result.Export()) // 5
}
```

## 5. Promise Handling & Async Operations

### Basic Promise Creation
```go
// Create a new promise with resolver and rejecter
promise, resolve, reject := vm.NewPromise()

// Simple resolution
resolve(vm.ToValue("success"))

// Simple rejection
reject(vm.ToValue("error occurred"))
```

### Promise States and Results
```go
// Check promise state
state := promise.State()
switch state {
case goja.PromiseStatePending:
    // Still running
case goja.PromiseStateFulfilled:
    // Get fulfilled value
    result := promise.Result()
case goja.PromiseStateRejected:
    // Get rejection reason
    reason := promise.Result()
}
```

### Event Loop Integration
```go
// Create event loop for async operations
loop := eventloop.NewEventLoop()
loop.Start()
defer loop.Stop()

// Run async operation with proper event loop handling
loop.RunOnLoop(func(runtime *goja.Runtime) {
    promise, resolve, reject := runtime.NewPromise()
    
    go func() {
        // Async work here
        result, err := someAsyncOperation()
        
        // Must resolve/reject on event loop
        loop.RunOnLoop(func(*goja.Runtime) {
            if err != nil {
                reject(runtime.ToValue(err.Error()))
                return
            }
            resolve(runtime.ToValue(result))
        })
    }()
})
```

### Promise Chaining
```go
// In JavaScript:
vm.RunString(`
    promise
        .then(value => {
            return processValue(value);
        })
        .catch(err => {
            console.error("Error:", err);
        })
        .finally(() => {
            cleanup();
        });
`)

// Creating chainable promises in Go
func createChainablePromise(vm *goja.Runtime, loop *eventloop.EventLoop) *goja.Promise {
    promise, resolve, reject := vm.NewPromise()
    
    go func() {
        // First async operation
        result1, err := firstOperation()
        if err != nil {
            loop.RunOnLoop(func(*goja.Runtime) {
                reject(vm.ToValue(err.Error()))
            })
            return
        }
        
        // Second async operation
        result2, err := secondOperation(result1)
        if err != nil {
            loop.RunOnLoop(func(*goja.Runtime) {
                reject(vm.ToValue(err.Error()))
            })
            return
        }
        
        loop.RunOnLoop(func(*goja.Runtime) {
            resolve(vm.ToValue(result2))
        })
    }()
    
    return promise
}
```

### Promise Error Handling
```go
// Proper error propagation
func handlePromiseError(vm *goja.Runtime, err error) goja.Value {
    switch e := err.(type) {
    case *goja.Exception:
        // JavaScript error
        return vm.ToValue(e.Value())
    case *CustomError:
        // Custom error type
        return vm.ToValue(map[string]interface{}{
            "type": "CustomError",
            "message": e.Error(),
            "details": e.Details,
        })
    default:
        // Generic error
        return vm.ToValue(err.Error())
    }
}

// Usage in promise rejection
loop.RunOnLoop(func(runtime *goja.Runtime) {
    if err != nil {
        reject(handlePromiseError(runtime, err))
        return
    }
})
```

### Promise Race and All
```go
// Implementing Promise.all behavior
func promiseAll(vm *goja.Runtime, loop *eventloop.EventLoop, promises []*goja.Promise) *goja.Promise {
    allPromise, resolve, reject := vm.NewPromise()
    results := make([]goja.Value, len(promises))
    completed := 0
    
    for i, p := range promises {
        idx := i // Capture loop variable
        
        // Handle each promise
        _, err := vm.RunString(fmt.Sprintf(`
            promises[%d].then(
                function(result) {
                    results[%d] = result;
                    completed++;
                    if (completed === promises.length) {
                        resolve(results);
                    }
                },
                function(err) {
                    reject(err);
                }
            )
        `, idx, idx))
        
        if err != nil {
            loop.RunOnLoop(func(*goja.Runtime) {
                reject(vm.ToValue(err.Error()))
            })
            return allPromise
        }
    }
    
    return allPromise
}

// Implementing Promise.race behavior
func promiseRace(vm *goja.Runtime, loop *eventloop.EventLoop, promises []*goja.Promise) *goja.Promise {
    racePromise, resolve, reject := vm.NewPromise()
    resolved := false
    
    for _, p := range promises {
        // Handle each promise
        _, err := vm.RunString(`
            promise.then(
                function(result) {
                    if (!resolved) {
                        resolved = true;
                        resolve(result);
                    }
                },
                function(err) {
                    if (!resolved) {
                        resolved = true;
                        reject(err);
                    }
                }
            )
        `)
        
        if err != nil {
            loop.RunOnLoop(func(*goja.Runtime) {
                reject(vm.ToValue(err.Error()))
            })
            return racePromise
        }
    }
    
    return racePromise
}
```

### Promise Cancellation
```go
// Cancellable promise with context
func createCancellablePromise(
    ctx context.Context,
    vm *goja.Runtime,
    loop *eventloop.EventLoop,
) *goja.Promise {
    promise, resolve, reject := vm.NewPromise()
    
    go func() {
        select {
        case <-ctx.Done():
            loop.RunOnLoop(func(*goja.Runtime) {
                reject(vm.ToValue("Operation cancelled"))
            })
            return
        default:
            // Perform async work
            result, err := someAsyncWork()
            loop.RunOnLoop(func(*goja.Runtime) {
                if err != nil {
                    reject(vm.ToValue(err.Error()))
                    return
                }
                resolve(vm.ToValue(result))
            })
        }
    }()
    
    return promise
}

// Usage with AbortController in JavaScript
vm.RunString(`
    const controller = new AbortController();
    const promise = startAsyncOperation({ signal: controller.signal });
    
    // Cancel after timeout
    setTimeout(() => {
        controller.abort();
    }, 5000);
`)
```

### Promise Rejection Tracking
```go
// Track unhandled promise rejections
vm.SetPromiseRejectionTracker(func(p *goja.Promise, operation goja.PromiseRejectionOperation) {
    switch operation {
    case goja.PromiseRejectionReject:
        log.Warn().
            Interface("reason", p.Result()).
            Msg("Unhandled promise rejection")
    case goja.PromiseRejectionHandle:
        log.Info().Msg("Previously unhandled promise rejection was handled")
    }
})
```

### Promise Interop with Go Context
```go
// Using Go context with promises for timeouts and cancellation
func createTimeoutPromise(vm *goja.Runtime, loop *eventloop.EventLoop, duration time.Duration) *goja.Promise {
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()

    promise, resolve, reject := vm.NewPromise()
    
    go func() {
        select {
        case <-ctx.Done():
            loop.RunOnLoop(func(*goja.Runtime) {
                switch ctx.Err() {
                case context.DeadlineExceeded:
                    reject(vm.ToValue("timeout"))
                case context.Canceled:
                    reject(vm.ToValue("canceled"))
                }
            })
        case result := <-doAsyncWork():
            loop.RunOnLoop(func(*goja.Runtime) {
                resolve(vm.ToValue(result))
            })
        }
    }()

    return promise
}
```

### Basic Usage Examples

1. Simple Runtime Setup
```go
// Basic runtime initialization and script execution
vm := goja.New()
val, err := vm.RunString("2 + 2")
if err != nil {
    panic(err)
}
fmt.Println(val.Export()) // Output: 4
```

2. Error Handling Pattern
```go
vm := goja.New()
_, err := vm.RunString(`
    throw new Error("Custom error");
`)
if err != nil {
    if jsErr, ok := err.(*goja.Exception); ok {
        fmt.Printf("JavaScript error: %v\n", jsErr.Value())
    }
}
```

3. Data Type Conversion
```go
// Go struct to JavaScript object
type Person struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

vm := goja.New()
person := &Person{Name: "John", Age: 30}
vm.Set("person", person)

// Access from JavaScript
val, _ := vm.RunString(`
    console.log(person.name + " is " + person.age + " years old");
    person.age += 1;
    person.age;
`)
fmt.Printf("New age: %v\n", val.Export())
```

4. Function Exposure
```go
// Exposing Go function to JavaScript
vm := goja.New()
vm.Set("multiply", func(call goja.FunctionCall) goja.Value {
    x := call.Argument(0).ToInteger()
    y := call.Argument(1).ToInteger()
    return vm.ToValue(x * y)
})

result, _ := vm.RunString(`multiply(3, 4)`)
fmt.Println(result.Export()) // Output: 12
```

5. Promise Handling
```go
vm := goja.New()
promise, resolve, reject := vm.NewPromise()

// Simulate async operation
go func() {
    time.Sleep(1 * time.Second)
    resolve("Operation completed")
}()

// Handle promise in JavaScript
vm.RunString(`
    promise.then(function(result) {
        console.log("Promise resolved:", result);
    }).catch(function(error) {
        console.log("Promise rejected:", error);
    });
`)
```

6. Object Property Access
```go
vm := goja.New()
_, _ = vm.RunString(`
    var obj = {
        name: "test",
        nested: {
            value: 42
        }
    };
`)

obj := vm.Get("obj").ToObject(vm)
name := obj.Get("name").String()
nestedValue := obj.Get("nested").ToObject(vm).Get("value").ToInteger()
```

7. Array Manipulation
```go
vm := goja.New()
array := vm.NewArray()
_ = array.Set("0", "first")
_ = array.Set("1", "second")

// Access from JavaScript
result, _ := vm.RunString(`
    var arr = array;
    arr.push("third");
    arr.join(", ");
`)
```

8. Custom Constructor
```go
vm := goja.New()
vm.Set("MyClass", func(call goja.ConstructorCall) *goja.Object {
    // Initialize instance
    this := call.This
    _ = this.Set("value", call.Argument(0).String())
    
    // Add method
    _ = this.Set("getValue", func() string {
        return this.Get("value").String()
    })
    
    return nil // Return nil to use call.This
})

_, _ = vm.RunString(`
    var instance = new MyClass("test");
    console.log(instance.getValue()); // "test"
`)
```

