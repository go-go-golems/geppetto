---
Title: Tools in Geppetto (Turn-based)
Slug: geppetto-tools
Short: A complete guide to defining, registering, configuring, and executing tools with Turns
Topics:
- geppetto
- tools
- turns
- middleware
- helpers
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

## Tools in Geppetto (Turn-based)

Tools enable models to call functions with structured inputs. In the Turn-based architecture, provider engines emit `tool_call` blocks; middleware (or helpers) execute tools and append `tool_use` blocks containing results.

### Overview

- Tool definitions: name, description, JSON Schema for parameters, optional examples/tags/version
- Registry: stores callable tools for execution
- Engine config: providers need tool definitions to advertise capabilities
- Middleware/helpers: detect tool_call blocks, execute tools, append tool_use, re-enter engine step

### Define and register a tool

```go
type WeatherRequest struct {
    Location string `json:"location" jsonschema:"required"`
    Units    string `json:"units,omitempty" jsonschema:"enum=celsius,enum=fahrenheit,default=celsius"`
}

type WeatherResponse struct { Location string; Temperature float64 }

func weatherTool(req WeatherRequest) WeatherResponse { return WeatherResponse{Location: req.Location, Temperature: 22} }

reg := tools.NewInMemoryToolRegistry()
def, _ := tools.NewToolFromFunc("get_weather", "Get weather", weatherTool)
_ = reg.RegisterTool("get_weather", *def)
```

### Pass tools to provider engines

Engines include tools in API requests so models can emit structured calls.

```go
// Convert registry entries to engine.ToolDefinition and configure on engine (if supported)
var defs []engine.ToolDefinition
for _, t := range reg.ListTools() {
    defs = append(defs, engine.ToolDefinition{Name: t.Name, Description: t.Description, Parameters: t.Parameters})
}
if cfg, ok := e.(interface{ ConfigureTools([]engine.ToolDefinition, engine.ToolConfig) }); ok {
    cfg.ConfigureTools(defs, engine.ToolConfig{Enabled: true, ToolChoice: engine.ToolChoiceAuto, MaxParallelTools: 1})
}
```

### Turn blocks emitted by engines

- `llm_text`: assistant text
- `tool_call`: model requests invoking a tool `{id,name,args}`
- `tool_use`: result of tool execution `{id,result}`

### Executing tools with middleware

```go
tb := middleware.NewMockToolbox()
tb.RegisterTool("echo", "Echo text", map[string]any{"text": {"type":"string"}}, func(ctx context.Context, args map[string]any) (any, error) { return args["text"], nil })

mw := middleware.NewToolMiddleware(tb, middleware.ToolConfig{MaxIterations: 5})
wrapped := middleware.NewEngineWithMiddleware(e, mw)

seed := &turns.Turn{}
turns.AppendBlock(seed, turns.Block{Kind: turns.BlockKindUser, Payload: map[string]any{"text": "Use echo with 'hello'"}})
updated, _ := wrapped.RunInference(ctx, seed)
```

### Executing tools with helpers (from conversations)

For compatibility with conversation flows, `toolhelpers` provides an automated loop. Internally this now seeds a Turn, runs the engine, and translates results back to conversation.

```go
cfg := toolhelpers.NewToolConfig().WithMaxIterations(5)
finalConv, _ := toolhelpers.RunToolCallingLoop(ctx, e, initialConversation, reg, cfg)
```

### Best practices

- Define precise JSON Schemas; prefer required fields only when necessary
- Keep tool inputs small; providers have payload limits
- Log tool execution and results for observability
- Use timeouts and iteration limits to prevent loops
- Prefer middleware for Turn-native apps; use helpers when starting from conversations


