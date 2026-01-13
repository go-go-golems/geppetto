---
Title: Build a Streaming Inference Command with Tool Calling
Slug: geppetto-streaming-inference-tools
Short: Step-by-step tutorial to build a Cobra command that streams model output and supports tool calling using Geppetto.
Topics:
- geppetto
- tutorial
- inference
- streaming
- tools
- events
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Build a Streaming Inference Command with Tool Calling

This tutorial explains how to build a Cobra command that performs streaming inference and supports tool calling using Geppetto. We follow the engine-first architecture: engines handle provider I/O and emit events, while helpers orchestrate tools. The focus is on concepts, small runnable snippets, and the key APIs you will use, not a single large code dump. See the style guide for expectations around examples and structure: `glaze help how-to-write-good-documentation-pages`.

For foundational background, see:
- [Inference Engines](../topics/06-inference-engines.md)
- [Events and Streaming](../topics/04-events.md)

Note: In current Geppetto, provider engines learn about available tools from the tool registry attached to `context.Context` (see `toolcontext.WithRegistry`). This tutorial shows that wiring explicitly in Step 6.

## What You’ll Build

- A CLI command that:
  - Parses provider config layers (Pinocchio profiles)
  - Starts a Watermill-backed event router and console printers
  - Creates an engine configured with an EventSink for streaming
  - Registers a sample tool and runs a tool-calling loop
  - Streams assistant output (and tool activity) to the console

## Learning Objectives

- Understand the streaming event flow and why the sink matters
- Learn how to register tools and enable tool calls with streaming
- See how Turns are built and updated
- Know which APIs to use when composing your own command

## Prerequisites

- Basic Go and Cobra knowledge
- Configured Pinocchio profile(s) for your provider
- Geppetto modules in your project

## Architecture at a Glance

- Engine: runs inference against a provider and emits events
- Event router: transports events (Watermill) and drives printers
- Event sink: connects the engine and helpers to the router
- Tool registry: in-memory store of callable tools
- Tool helpers: manage the tool-calling loop and Turn updates

## Key APIs You’ll Use

- Engine and sink
  - `factory.NewEngineFromParsedLayers(parsed, engine.WithSink(sink))`
  - `middleware.NewWatermillSink(publisher, topic)`
- Events and printers
  - `events.NewEventRouter()`
  - `events.StepPrinterFunc(prefix, w)` or `events.NewStructuredPrinter(w, options)`
- Tools
  - `tools.NewInMemoryToolRegistry()`
  - `tools.NewToolFromFunc(name, description, func)`
  - `toolcontext.WithRegistry(ctx, registry)` (attach runtime registry to `context.Context`)
- Turns and helpers
  - `turns.NewSystemTextBlock(...)` / `turns.NewUserTextBlock(...)`
  - `toolhelpers.RunToolCallingLoop(ctx, eng, turn, registry, toolConfig)`

## Step 1 — Define the CLI Command

Create a command description with arguments and flags, including a `pinocchio-profile` to load provider layers.

```go
// inside NewStreamingCmd()
desc := cmds.NewCommandDescription(
    "stream-with-tools",
    cmds.WithShort("Streaming inference with tools"),
    cmds.WithArguments(
        parameters.NewParameterDefinition("prompt", parameters.ParameterTypeString,
            parameters.WithHelp("Prompt")),
    ),
    cmds.WithFlags(
        parameters.NewParameterDefinition("pinocchio-profile", parameters.ParameterTypeString,
            parameters.WithDefault("4o-mini")),
        parameters.NewParameterDefinition("output-format", parameters.ParameterTypeString,
            parameters.WithDefault("text")),
        parameters.NewParameterDefinition("with-metadata", parameters.ParameterTypeBool,
            parameters.WithDefault(false)),
    ),
    cmds.WithLayersList(geppettolayers.CreateGeppettoLayers()...),
)
```

## Step 2 — Start the Event Router and Printers

The event router moves tokens and tool events through Watermill. Attach a human-readable printer or a structured one.

```go
router, _ := events.NewEventRouter()
defer router.Close()

// Text printer for humans
router.AddHandler("chat", "chat", events.StepPrinterFunc("", os.Stdout))

// OR machine-readable output
// printer := events.NewStructuredPrinter(os.Stdout, events.PrinterOptions{
//   Format: events.PrinterFormat("json"), IncludeMetadata: false,
// })
// router.AddHandler("chat", "chat", printer)

sink := middleware.NewWatermillSink(router.Publisher, "chat")
```

Why this matters: the sink ties your engine and helpers to the router so that tokens and tool activity can be streamed and printed as they happen.

## Step 3 — Create the Engine (Streaming Enabled)

Pass the sink to the engine so it can emit streaming events.

```go
eng, err := factory.NewEngineFromParsedLayers(parsed, engine.WithSink(sink))
if err != nil { return err }
```

## Step 4 — Register a Tool

Use the in-memory registry and define a simple tool. Providers that support built-in tool schemas can be configured with those definitions.

```go
registry := tools.NewInMemoryToolRegistry()
getWeather, _ := tools.NewToolFromFunc(
    "get_weather",
    "Get weather for a location",
    func(req struct{ Location, Units string }) struct{ Temperature float64 } {
        return struct{ Temperature float64 }{Temperature: 22.0}
    },
)
_ = registry.RegisterTool("get_weather", *getWeather)
```

## Step 5 — Build the Turn

Create a Turn with a system block, a user block, and tool config stored in `Turn.Data`.

```go
seed := &turns.Turn{Data: map[turns.TurnDataKey]any{}}
seed.Data[turns.DataKeyToolConfig] = engine.ToolConfig{
    Enabled:          true,
    ToolChoice:       engine.ToolChoiceAuto,
    MaxParallelTools: 1,
}
turns.AppendBlock(seed, turns.NewSystemTextBlock(
    "You are a helpful assistant with access to tools.",
))
turns.AppendBlock(seed, turns.NewUserTextBlock(s.Prompt))
```

## Step 6 — Run the Router and Tool-Calling Loop

Run the router and the helper loop concurrently using `errgroup`. This pattern ensures proper coordination:

```go
eg, groupCtx := errgroup.WithContext(ctx)

// Goroutine 1: Run the event router
// The router blocks until its context is cancelled
eg.Go(func() error { return router.Run(groupCtx) })

// Goroutine 2: Run inference after router is ready
eg.Go(func() error {
    // CRITICAL: Wait for router to be ready before publishing events
    <-router.Running()
    
    // Attach the sink to context so helpers and tools can publish events
    runCtx := events.WithEventSinks(groupCtx, sink)
    
    // Attach the tool registry to context so engines know what tools are available
    // (Engines read from context, not from Turn.Data)
    runCtx = toolcontext.WithRegistry(runCtx, registry)
    
    // Run the tool-calling loop:
    // 1. Calls RunInference with the Turn
    // 2. If model emits tool_call blocks, executes tools
    // 3. Appends tool_use blocks with results
    // 4. Re-invokes inference until no more tool calls (or max iterations)
    updated, err := toolhelpers.RunToolCallingLoop(
        runCtx, eng, seed, registry, 
        toolhelpers.NewToolConfig().WithMaxIterations(5),
    )
    if err != nil { return err }
    
    // 'updated' now contains the full conversation:
    // [system] → [user] → [llm_text] → [tool_call] → [tool_use] → [llm_text (final)]
    _ = updated
    return nil
})

// Wait for both goroutines to complete
// If either fails, the other is cancelled via groupCtx
if err := eg.Wait(); err != nil { return err }
```

**Why errgroup?**
- `errgroup.WithContext` creates a derived context that cancels when any goroutine fails
- If the inference fails, the router stops; if the router fails, inference stops
- `eg.Wait()` returns the first error from any goroutine

### Sample Text Output

```
assistant: Thinking…
assistant: I will check the current temperature.
call: get_weather {"location":"Paris","units":"celsius"}
result: {"temperature":22}
assistant: It’s about 22°C in Paris right now.
```

## Minimal Variants

- Without tools: call `eng.RunInference(ctx, seed)` and skip the helper loop.
- Structured streaming: use `events.NewStructuredPrinter` with `json` or `yaml` for machine-readable logs.

## Troubleshooting and Tips

- No output streaming? Ensure the engine is constructed with `engine.WithSink(sink)` and that the sink is also placed on the context via `events.WithEventSinks(...)`.
- Blank console? Confirm a handler is registered for the same topic you used when creating the sink (here: `"chat"`).
- Tool schemas not applied? Some providers don’t accept external tool definitions; fall back to the generic helper loop which inspects model output and invokes tools from the registry.
- Infinite loops: cap iterations with `toolhelpers.NewToolConfig().WithMaxIterations(n)`.

## See Also

- [Inference Engines](../topics/06-inference-engines.md) — Engine architecture and factory patterns
- [Events and Streaming](../topics/04-events.md) — Event types, routing, and printers
- [Tools](../topics/07-tools.md) — Tool definitions and registry patterns
- [Turns and Blocks](../topics/08-turns.md) — The Turn data model
- [Middlewares](../topics/09-middlewares.md) — Alternative to helper-based tool execution

**Working example programs:**
- `geppetto/cmd/examples/simple-streaming-inference/main.go`
- `geppetto/cmd/examples/openai-tools/main.go`
- `geppetto/cmd/examples/claude-tools/main.go`
- `geppetto/cmd/examples/generic-tool-calling/main.go`

If you need a full, copy-paste command, use the example apps above as a reference implementation and adapt the snippets here to your project structure.
