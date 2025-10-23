---
Title: Tools in Geppetto (Turn-based)
Slug: geppetto-tools
Short: A complete guide to defining, attaching, and executing tools with Turns. Tools are discoverable per Turn via `Turn.Data`.
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

Tools enable models to call functions with structured inputs. In the Turn-based architecture, provider engines emit `tool_call` blocks; middleware (or helpers) execute tools and append `tool_use` blocks. As of this refactor, tools are attached per Turn: the engine reads which tools are available for that Turn from `Turn.Data`. This allows dynamic tools per step without mutating the engine’s state. We follow the Glaze documentation guidelines for clarity and completeness [[memory:5699956]].

### Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
    "github.com/go-go-golems/geppetto/pkg/turns"
)
```

### What you’ll learn

- How to define tools and register them
- How to attach tools to a `Turn` for provider advertisement
- How middleware and helpers execute tools
- How engines map model outputs to Turn blocks

### Key concepts (at a glance)

- Registry: `tools.ToolRegistry` holds callable tools
- Per-Turn tools: `turns.DataKeyToolRegistry` and `turns.DataKeyToolConfig` on `Turn.Data`
- Blocks: `llm_text`, `tool_call`, `tool_use`
### OpenAI Responses specifics

When using the OpenAI Responses engine (`ai-api-type=openai-responses`):

- Tools are advertised via the `tools` array on the request. For function tools, schema is top-level: `{type: "function", name, description, parameters}`.
- The engine streams function_call arguments via SSE and emits a `tool-call` event when the function_call completes.
- Reasoning summary is streamed as `partial-thinking` events; UIs can render it between "Thinking started/ended" markers.
- The next iteration (not yet implemented in docs) will include the `assistant:function_call` and `tool:tool_result` blocks in the next request’s `input` to continue tool-driven workflows.
- Payload keys: use `turns.PayloadKeyText`, `turns.PayloadKeyID`, `turns.PayloadKeyName`, `turns.PayloadKeyArgs`, `turns.PayloadKeyResult`

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

2) Attach the registry (and optional tool config) to a Turn

```go
seed := &turns.Turn{ Data: map[string]any{} }
seed.Data[turns.DataKeyToolRegistry] = reg
seed.Data[turns.DataKeyToolConfig] = engine.ToolConfig{
    Enabled:          true,
    ToolChoice:       engine.ToolChoiceAuto,
    MaxParallelTools: 1,
}
```

3) Run the engine and execute tools with middleware or helpers

```go
// Middleware route (Turn-native)
tb := middleware.NewMockToolbox()
tb.RegisterTool("echo", "Echo text", map[string]any{"text": {"type": "string"}},
    func(ctx context.Context, args map[string]any) (any, error) { return args["text"], nil })
mw := middleware.NewToolMiddleware(tb, middleware.ToolConfig{MaxIterations: 5})
wrapped := middleware.NewEngineWithMiddleware(e, mw)

turns.AppendBlock(seed, turns.NewUserTextBlock("Use echo with 'hello'"))
updated, _ := wrapped.RunInference(ctx, seed)

// Helper route (conversation-first)
cfg := toolhelpers.NewToolConfig().WithMaxIterations(5)
finalConv, _ := toolhelpers.RunToolCallingLoop(ctx, e, initialConversation, reg, cfg)
```

---

## Guided walkthrough: End-to-end example

The following example shows how to:
- Define a tool with JSON Schema inferred from a Go function
- Seed a Turn with a per-Turn registry
- Let the engine advertise tools to the provider
- Execute tools via middleware and return results as `tool_use` blocks

```go
package main

import (
    "context"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
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

    // 2) Seed a Turn with registry and minimal tool config
    t := &turns.Turn{ Data: map[string]any{} }
    t.Data[turns.DataKeyToolRegistry] = reg
    t.Data[turns.DataKeyToolConfig] = engine.ToolConfig{ Enabled: true }
    turns.AppendBlock(t, turns.NewUserTextBlock("Please use add with a=2 and b=3"))

    // 3) Attach tool middleware to execute tool_use blocks
    tb := middleware.NewMockToolbox()
    tb.RegisterTool("add", "Add two numbers", map[string]any{
        "a": {"type": "number"},
        "b": {"type": "number"},
    }, func(ctx context.Context, args map[string]any) (any, error) {
        return args["a"].(float64) + args["b"].(float64), nil
    })
    e = middleware.NewEngineWithMiddleware(e, middleware.NewToolMiddleware(tb, middleware.ToolConfig{MaxIterations: 3}))

    // 4) Run inference (engine may emit tool_call; middleware executes and appends tool_use)
    _, err := e.RunInference(ctx, t)
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

Override whichever hooks you need; the base executor handles the rest (context cancellation, event emission, timings, and retries). For most projects, `tools.NewDefaultToolExecutor` remains sufficient, and helper utilities such as `toolhelpers.RunToolCallingLoop` continue to use it under the hood.

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
```

Engine discovery keys in `Turn.Data`:

```go
turns.DataKeyToolRegistry // tools.ToolRegistry
turns.DataKeyToolConfig   // engine.ToolConfig
```

---

## Best practices

- Define precise JSON Schemas; mark required params judiciously
- Keep tool inputs small (provider payload limits apply)
- Log tool execution steps and responses
- Use timeouts and iteration limits to prevent loops
- Prefer middleware for Turn-native use; use helpers for conversation-first flows

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

## Troubleshooting and tips

- If an engine doesn’t seem to advertise your tools, ensure `Turn.Data[turns.DataKeyToolRegistry]` is set and non-empty
- When reading payloads, always use the payload key constants to avoid typos
- To change tools at runtime, modify the Turn’s registry or config before calling `RunInference`
