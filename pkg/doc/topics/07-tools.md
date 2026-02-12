---
Title: Tools in Geppetto (Turn-based)
Slug: geppetto-tools
Short: A complete guide to defining, attaching, and executing tools with Turns. Tool registries are carried via `context.Context` (Turn state stays serializable).
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

# Tools in Geppetto (Turn-based)

## Why Tools?

Large language models can generate text, but they can't access databases, call APIs, or perform calculations directly. **Tools** bridge this gap by letting models request specific function calls with structured inputs.

When a model needs information it doesn't have (like today's weather) or wants to perform an action (like sending an email), it emits a **tool call** with the function name and arguments. Your code executes the function and returns the result, allowing the model to continue with fresh information.

**Example flow:**
```
User: "What's the weather in Paris?"
  ↓
Model: tool_call {name: "get_weather", args: {location: "Paris"}}
  ↓
Your code: executes get_weather("Paris") → {temp: 18, conditions: "Cloudy"}
  ↓
Model: "The weather in Paris is 18°C and cloudy."
```

## Architecture Overview

In the Turn-based architecture:
- **Provider engines** emit `tool_call` blocks when models request tools
- **The tool loop runner** executes tools and appends `tool_use` blocks
- **Engines re-run** with the updated Turn to let the model continue

> **Key Pattern:** The runtime `tools.ToolRegistry` is carried via `context.Context` (see `tools.WithRegistry`). Only serializable tool configuration lives on `Turn.Data` (e.g., `engine.KeyToolConfig`). This keeps Turn state persistable while allowing dynamic tools per inference call.

### Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/session"
    "github.com/go-go-golems/geppetto/pkg/inference/toolloop"
    "github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/turns"
)
```

### What you’ll learn

- How to define tools and register them
- How to attach tools to a `Turn` for provider advertisement
- How the tool loop runner executes tools
- How engines map model outputs to Turn blocks

### Key concepts (at a glance)

- Registry: `tools.ToolRegistry` holds callable tools
- Per-Turn tools:
  - Runtime registry: carried via `context.Context` using `tools.WithRegistry(ctx, reg)`
  - Serializable config: stored on `Turn.Data` via `engine.KeyToolConfig`
- Blocks: `llm_text`, `tool_call`, `tool_use`
### OpenAI Responses specifics

When using the OpenAI Responses engine (`ai-api-type=openai-responses`):

- Tools are advertised via the `tools` array on the request. For function tools, schema is top-level: `{type: "function", name, description, parameters}`.
- The engine streams function_call arguments via SSE and emits a `tool-call` event when the function_call completes.
- Reasoning summary is streamed as `partial-thinking` events; UIs can render it between "Thinking started/ended" markers.
- The next iteration (not yet implemented in docs) will include the `assistant:function_call` and `tool:tool_result` blocks in the next request’s `input` to continue tool-driven workflows.
- Payload keys: use `turns.PayloadKeyText`, `turns.PayloadKeyID`, `turns.PayloadKeyName`, `turns.PayloadKeyArgs`, `turns.PayloadKeyResult`, `turns.PayloadKeyError`

---

## Quickstart: From zero to tool execution

1) Define a tool function and register it

```go
type WeatherRequest struct {
    Location string `json:"location" jsonschema:"required"`
    Units    string `json:"units,omitempty" jsonschema:"enum=celsius,enum=fahrenheit,default=celsius"`
}

type WeatherResponse struct {
    Location    string
    Temperature float64
}

func weatherTool(req WeatherRequest) WeatherResponse {
    return WeatherResponse{Location: req.Location, Temperature: 22}
}

reg := tools.NewInMemoryToolRegistry()
def, _ := tools.NewToolFromFunc("get_weather", "Get weather", weatherTool)
_ = reg.RegisterTool("get_weather", *def)
```

2) Run the tool loop (the loop attaches the registry to context and manages tool config on the Turn)

```go
seed := &turns.Turn{}
turns.AppendBlock(seed, turns.NewUserTextBlock("What's the weather in Paris? Use get_weather."))

loopCfg := toolloop.NewLoopConfig().
    WithMaxIterations(5)

toolCfg := tools.DefaultToolConfig().
    WithMaxParallelTools(1).
    WithToolChoice(tools.ToolChoiceAuto).
    WithToolErrorHandling(tools.ToolErrorContinue)

loop := toolloop.New(
    toolloop.WithEngine(e),
    toolloop.WithRegistry(reg),
    toolloop.WithLoopConfig(loopCfg),
    toolloop.WithToolConfig(toolCfg),
)
updated, err := loop.RunLoop(ctx, seed)
```

3) Alternatively: run via the session builder

```go
runner, _ := enginebuilder.New(
    enginebuilder.WithBase(e),
    enginebuilder.WithToolRegistry(reg),
    enginebuilder.WithLoopConfig(loopCfg),
    enginebuilder.WithToolConfig(toolCfg),
).Build(ctx, "demo-session")
updated, _ := runner.RunInference(ctx, seed)
```

## How to wire tools end-to-end

This is the minimal “wire it up and run” pattern. It assumes you already have an `engine.Engine` (via `factory.NewEngineFromParsedValues(...)` or your own builder) and a populated `tools.ToolRegistry`:

```go
import (
    "context"
    "time"

    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/toolloop"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

func RunWithTools(ctx context.Context, eng engine.Engine, reg tools.ToolRegistry, seed *turns.Turn, sinks ...events.EventSink) (*turns.Turn, error) {
    if len(sinks) > 0 {
        ctx = events.WithEventSinks(ctx, sinks...)
    }

    loopCfg := toolloop.NewLoopConfig().WithMaxIterations(5)
    toolCfg := tools.DefaultToolConfig().WithExecutionTimeout(60 * time.Second)

    loop := toolloop.New(
        toolloop.WithEngine(eng),
        toolloop.WithRegistry(reg),
        toolloop.WithLoopConfig(loopCfg),
        toolloop.WithToolConfig(toolCfg),
    )
    return loop.RunLoop(ctx, seed)
}
```

---

## Guided walkthrough: End-to-end example

The following example shows how to:
- Define a tool with JSON Schema inferred from a Go function
- Seed a Turn with a per-Turn registry
- Let the engine advertise tools to the provider
- Execute tools via the tool loop and return results as `tool_use` blocks

```go
package main

	import (
	    "context"
	    "github.com/go-go-golems/geppetto/pkg/inference/engine"
		    "github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	    "github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	    "github.com/go-go-golems/geppetto/pkg/inference/tools"
	    "github.com/go-go-golems/geppetto/pkg/turns"
	)

type AddRequest struct { A, B float64 `json:"a" jsonschema:"required"` }
type AddResponse struct { Sum float64 `json:"sum"` }

func addTool(req AddRequest) AddResponse { return AddResponse{Sum: req.A + req.B} }

	func run(ctx context.Context, e engine.Engine) error {
	    // 1) Create registry and register the tool
	    reg := tools.NewInMemoryToolRegistry()
	    def, _ := tools.NewToolFromFunc("add", "Add two numbers", addTool)
	    _ = reg.RegisterTool("add", *def)

	    // 2) Seed a Turn
	    t := &turns.Turn{}
	    turns.AppendBlock(t, turns.NewUserTextBlock("Please use add with a=2 and b=3"))

	    // 3) Configure the tool loop
	    loopCfg := toolloop.NewLoopConfig().
	        WithMaxIterations(3)

	    toolCfg := tools.DefaultToolConfig().
	        WithMaxParallelTools(1).
	        WithToolChoice(tools.ToolChoiceAuto).
	        WithToolErrorHandling(tools.ToolErrorContinue)

	    // 4) Build a runner that owns tool execution
		    builder := enginebuilder.New(
		        enginebuilder.WithBase(e),
		        enginebuilder.WithToolRegistry(reg),
		        enginebuilder.WithLoopConfig(loopCfg),
		        enginebuilder.WithToolConfig(toolCfg),
		    )
		    runner, err := builder.Build(ctx, "demo-session")
	    if err != nil {
	        return err
	    }

	    // 5) Run inference (engine may emit tool_call; tool loop executes and appends tool_use)
	    _, err = runner.RunInference(ctx, t)
	    return err
	}
```

---

## Tool executors and lifecycle hooks

Once a provider emits `tool_call` blocks, a `tools.ToolExecutor` turns those calls into actual function invocations. Geppetto now ships two composable executors:

- `tools.DefaultToolExecutor` wraps the standard behavior (argument masking, event publishing, retries, and parallelism driven by `ToolConfig`)
- `tools.BaseToolExecutor` provides the orchestration plus overridable lifecycle hooks so you can inject authorization, observability, or custom retry heuristics

The `ToolConfig` you attach to the Turn still governs concurrency (`MaxParallelTools`), error handling (`ToolErrorAbort` vs `ToolErrorRetry`), and retry backoff. `DefaultToolExecutor` simply wires those settings into the base implementation.

If you need custom behavior, embed the base executor and override only the hooks you care about. Remember to point the base executor back to the outer type so the overrides run.

```go
import (
    "context"
    "encoding/json"

    "github.com/go-go-golems/geppetto/pkg/inference/tools"
)

type Session interface {
    Bearer() string
}

type AuthorizedExecutor struct {
    *tools.BaseToolExecutor
    sess Session
}

func NewAuthorizedExecutor(cfg tools.ToolConfig, sess Session) *AuthorizedExecutor {
    base := tools.NewBaseToolExecutor(cfg)
    exec := &AuthorizedExecutor{BaseToolExecutor: base, sess: sess}
    base.ToolExecutorExt = exec // enable hook overrides
    return exec
}

func (a *AuthorizedExecutor) PreExecute(ctx context.Context, call tools.ToolCall, _ tools.ToolRegistry) (tools.ToolCall, error) {
    // Inject auth into the argument payload before execution
    var args map[string]any
    _ = json.Unmarshal(call.Arguments, &args)
    if args == nil {
        args = map[string]any{}
    }
    args["auth"] = map[string]string{"bearer_token": a.sess.Bearer()}
    call.Arguments, _ = json.Marshal(args)
    return call, nil
}

func (a *AuthorizedExecutor) MaskArguments(ctx context.Context, call tools.ToolCall) string {
    // Redact secrets when events are published
    var args map[string]any
    _ = json.Unmarshal(call.Arguments, &args)
    if auth, ok := args["auth"].(map[string]any); ok {
        auth["bearer_token"] = "***"
    }
    masked, _ := json.Marshal(args)
    return string(masked)
}
```

Available hooks on `BaseToolExecutor`:
- `PreExecute` mutate or reject calls before lookup
- `IsAllowed` add authorization beyond `ToolConfig.AllowedTools`
- `MaskArguments`, `PublishStart`, `PublishResult` tune event payloads
- `ShouldRetry` implement bespoke retry policies
- `MaxParallel` override concurrency control per batch

Override whichever hooks you need; the base executor handles the rest (context cancellation, event emission, timings, and retries). For most projects, `tools.NewDefaultToolExecutor` remains sufficient, and higher-level orchestration (via `toolloop.Loop` or `toolloop/enginebuilder`) wires it in under the hood.

---

## Context-aware tool functions

`tools.NewToolFromFunc` recognises optional `context.Context` parameters. Supported signatures include:

- `func(Input) (Output, error)`
- `func(context.Context, Input) (Output, error)`
- `func(context.Context) (Output, error)` (no JSON payload)
- `func() (Output, error)`

When you register a tool, Geppetto generates JSON Schema for the first non-context parameter and compiles both context-free and context-aware executors. That means providers that pass Go contexts can propagate deadlines, auth tokens, or tracing spans straight into your tool implementation.

```go
func searchDocs(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("tool.searchDocs")
	return index.Search(ctx, req.Query)
}

def, _ := tools.NewToolFromFunc("search_docs", "Search internal documentation", searchDocs)
```

If a tool has no JSON input (for example `func(context.Context) (Result, error)`), the generated schema becomes an empty object so the provider can still advertise the tool.

---

## Reference: payload and data keys

When reading/writing block payloads, always use the constants:

```go
turns.PayloadKeyText
turns.PayloadKeyID
turns.PayloadKeyName
turns.PayloadKeyArgs
turns.PayloadKeyResult
turns.PayloadKeyError
```

Engine discovery keys in `Turn.Data`:

```go
engine.KeyToolConfig      // engine.ToolConfig stored in Turn.Data
```

---

## Best practices

- Define precise JSON Schemas; mark required params judiciously
- Keep tool inputs small (provider payload limits apply)
- Log tool execution steps and responses
- Use timeouts and iteration limits to prevent loops
- Prefer middleware for Turn-native automation; use helpers when you want explicit control over the loop

### Emitting Custom Events from Tools

Tools can emit custom progress or status events using the event registry. This is useful for long-running operations where you want to provide real-time feedback to users:

```go
import "github.com/go-go-golems/geppetto/pkg/events"

type ToolProgressEvent struct {
    events.EventImpl
    ToolName string  `json:"tool_name"`
    Progress float64 `json:"progress"`
    Message  string  `json:"message"`
}

func init() {
    _ = events.RegisterEventFactory("tool-progress", func() events.Event {
        return &ToolProgressEvent{EventImpl: events.EventImpl{Type_: "tool-progress"}}
    })
}

func longRunningTool(ctx context.Context, req ToolRequest) (ToolResponse, error) {
    // Emit progress events
    progressEvent := &ToolProgressEvent{
        EventImpl: events.EventImpl{Type_: "tool-progress", Metadata_: metadata},
        ToolName:  "long_running_tool",
        Progress:  0.5,
        Message:   "Processing data...",
    }
    events.PublishEventToContext(ctx, progressEvent)
    
    // ... tool implementation
}
```

For details on event extensibility, see: `glaze help geppetto-events-streaming-watermill`

## Troubleshooting and Tips

| Problem | Solution |
|---------|----------|
| Tools not advertised | Ensure `ctx = tools.WithRegistry(ctx, reg)` before `RunInference` |
| Tool call not executed | Check middleware is attached or run the tool loop explicitly via `toolloop.New(...).RunLoop(...)` |
| Payload key errors | Use constants like `turns.PayloadKeyArgs`, never string literals |
| Dynamic tools not working | Modify registry before calling `RunInference`; Turn.Data for config only |

## See Also

- [Turns and Blocks](08-turns.md) — The Turn data model and block kinds
- [Inference Engines](06-inference-engines.md) — How engines emit tool_call blocks
- [Events](04-events.md) — Tool events and custom event emission
- [Middlewares](09-middlewares.md) — Tool middleware for automatic execution
- [Streaming Tutorial](../tutorials/01-streaming-inference-with-tools.md) — Complete working example
