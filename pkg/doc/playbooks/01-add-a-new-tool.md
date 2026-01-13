---
Title: Add a New Tool
Slug: geppetto-playbook-add-tool
Short: Step-by-step guide to define a tool, register it, attach the registry to context, and configure tool calling on a Turn.
Topics:
- geppetto
- tools
- playbook
- turns
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Add a New Tool

This playbook walks through adding a new callable tool to your Geppetto application. By the end, your tool will be registered, advertised to the model, and executable during inference.

## Prerequisites

- A working Geppetto engine setup (see [Inference Engines](../topics/06-inference-engines.md))
- Basic understanding of Turns and Blocks (see [Turns](../topics/08-turns.md))

## Steps

### Step 1: Define the Tool Function

Create a Go function that implements your tool's logic. The function can optionally accept `context.Context` as its first parameter:

```go
package main

import (
    "context"
    "fmt"
)

// Tool input struct - fields become JSON schema properties
type GetWeatherInput struct {
    Location string `json:"location" description:"City name or coordinates"`
    Units    string `json:"units" description:"celsius or fahrenheit"`
}

// Tool output struct
type GetWeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Conditions  string  `json:"conditions"`
}

// Tool function - can accept context as first parameter
func getWeather(ctx context.Context, input GetWeatherInput) (GetWeatherOutput, error) {
    // Your implementation here
    // Access context for cancellation, deadlines, or request-scoped values
    select {
    case <-ctx.Done():
        return GetWeatherOutput{}, ctx.Err()
    default:
    }
    
    return GetWeatherOutput{
        Temperature: 22.0,
        Conditions:  "Partly cloudy",
    }, nil
}
```

**Why struct inputs?** Geppetto auto-generates a JSON schema from the struct, which the model uses to produce valid arguments.

### Step 2: Create a ToolDefinition

Convert your function into a `ToolDefinition` using `NewToolFromFunc`:

```go
import "github.com/go-go-golems/geppetto/pkg/inference/tools"

toolDef, err := tools.NewToolFromFunc(
    "get_weather",                           // Tool name (model uses this)
    "Get current weather for a location",    // Description (helps model decide when to use it)
    getWeather,                              // Your function
)
if err != nil {
    return fmt.Errorf("failed to create tool: %w", err)
}
```

### Step 3: Create and Populate the Registry

Create an in-memory registry and register your tool:

```go
import "github.com/go-go-golems/geppetto/pkg/inference/tools"

registry := tools.NewInMemoryToolRegistry()

if err := registry.RegisterTool("get_weather", *toolDef); err != nil {
    return fmt.Errorf("failed to register tool: %w", err)
}

// Register additional tools as needed
// registry.RegisterTool("search_web", *searchDef)
// registry.RegisterTool("run_code", *codeDef)
```

### Step 4: Attach Registry to Context

The registry must be attached to `context.Context` so engines and middleware can access it:

```go
import "github.com/go-go-golems/geppetto/pkg/inference/toolcontext"

// Before calling RunInference or RunToolCallingLoop
ctx = toolcontext.WithRegistry(ctx, registry)
```

**Why context, not Turn?** The registry contains function pointers (not serializable). Keeping it in context separates runtime state from persistable Turn data.

### Step 5: Configure Tool Calling on the Turn

Store tool configuration in `Turn.Data` using typed keys:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

turn := &turns.Turn{
    Data: map[turns.TurnDataKey]any{},
}

turn.Data[turns.DataKeyToolConfig] = engine.ToolConfig{
    Enabled:          true,
    ToolChoice:       engine.ToolChoiceAuto,  // or ToolChoiceRequired, ToolChoiceNone
    MaxParallelTools: 1,                       // How many tools can run in parallel
}

turns.AppendBlock(turn, turns.NewSystemTextBlock("You are a helpful assistant with weather tools."))
turns.AppendBlock(turn, turns.NewUserTextBlock("What's the weather in Paris?"))
```

### Step 6: Run Inference with Tool Calling

Use `toolhelpers.RunToolCallingLoop` for automatic tool execution:

```go
import "github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"

config := toolhelpers.NewToolConfig().
    WithMaxIterations(5).           // Max tool-call rounds
    WithToolChoice("auto")          // Let model decide

result, err := toolhelpers.RunToolCallingLoop(ctx, engine, turn, registry, config)
if err != nil {
    return fmt.Errorf("tool calling failed: %w", err)
}

// result contains: [system] → [user] → [llm_text] → [tool_call] → [tool_use] → [llm_text]
```

**What happens:**
1. Engine calls the model with your Turn
2. Model emits a `tool_call` block requesting `get_weather`
3. Helper executes `getWeather()` with the model's arguments
4. Helper appends a `tool_use` block with the result
5. Engine re-runs with the updated Turn
6. Model generates final response using the weather data

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/toolcontext"
    "github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

type GetWeatherInput struct {
    Location string `json:"location"`
    Units    string `json:"units"`
}

type GetWeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Conditions  string  `json:"conditions"`
}

func getWeather(ctx context.Context, input GetWeatherInput) (GetWeatherOutput, error) {
    return GetWeatherOutput{Temperature: 22.0, Conditions: "Sunny"}, nil
}

func main() {
    ctx := context.Background()
    
    // 1. Create engine (assumes parsed layers from CLI or config)
    eng, _ := factory.NewEngineFromParsedLayers(parsedLayers)
    
    // 2. Create and register tool
    toolDef, _ := tools.NewToolFromFunc("get_weather", "Get weather for location", getWeather)
    registry := tools.NewInMemoryToolRegistry()
    _ = registry.RegisterTool("get_weather", *toolDef)
    
    // 3. Attach registry to context
    ctx = toolcontext.WithRegistry(ctx, registry)
    
    // 4. Build Turn with tool config
    turn := &turns.Turn{Data: map[turns.TurnDataKey]any{}}
    turn.Data[turns.DataKeyToolConfig] = engine.ToolConfig{
        Enabled:    true,
        ToolChoice: engine.ToolChoiceAuto,
    }
    turns.AppendBlock(turn, turns.NewSystemTextBlock("You have access to weather tools."))
    turns.AppendBlock(turn, turns.NewUserTextBlock("What's the weather in Tokyo?"))
    
    // 5. Run tool calling loop
    config := toolhelpers.NewToolConfig().WithMaxIterations(5)
    result, err := toolhelpers.RunToolCallingLoop(ctx, eng, turn, registry, config)
    if err != nil {
        panic(err)
    }
    
    // 6. Print final response
    for _, block := range result.Blocks {
        if block.Kind == turns.BlockKindLLMText {
            fmt.Println(block.Payload[turns.PayloadKeyText])
        }
    }
}
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| Tool not called | Registry not on context | Add `ctx = toolcontext.WithRegistry(ctx, registry)` |
| Tool not advertised | ToolConfig not on Turn | Set `turn.Data[turns.DataKeyToolConfig]` |
| "tool not found" error | Name mismatch | Ensure `RegisterTool` name matches model's request |
| Infinite loop | No max iterations | Use `WithMaxIterations(n)` in config |
| Context values lost | Wrong context | Pass the enriched `ctx` to `RunToolCallingLoop` |

## See Also

- [Tools](../topics/07-tools.md) — Full tools reference
- [Turns and Blocks](../topics/08-turns.md) — Turn data model
- [Streaming Tutorial](../tutorials/01-streaming-inference-with-tools.md) — Complete streaming example
- Example: `geppetto/cmd/examples/generic-tool-calling/main.go`

