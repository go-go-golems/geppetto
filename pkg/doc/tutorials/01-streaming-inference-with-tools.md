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
- `glaze help geppetto-inference-engines`
- `glaze help geppetto-events-streaming-watermill`

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
- See how conversations are built and updated
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
- Tool helpers: manage the tool-calling loop and conversation updates

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
  - Optional: `ConfigureTools([]engine.ToolDefinition, engine.ToolConfig)` when supported by the provider engine
- Conversation and helpers
  - `builder.NewManagerBuilder().WithSystemPrompt(...).WithPrompt(...).Build()`
  - `toolhelpers.RunToolCallingLoop(ctx, eng, conv, registry, toolConfig)`

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

// Optionally, pass tool schemas to the engine (provider-dependent)
if cfg, ok := eng.(engine.ToolsConfigurable); ok {
    var defs []engine.ToolDefinition
    for _, t := range registry.ListTools() {
        defs = append(defs, engine.ToolDefinition{
            Name: t.Name, Description: t.Description, Parameters: t.Parameters,
        })
    }
    cfg.ConfigureTools(defs, engine.ToolConfig{Enabled: true})
}
```

## Step 5 — Build the Conversation

Create a conversation with a system prompt and the user’s prompt.

```go
mb := builder.NewManagerBuilder().
    WithSystemPrompt("You are a helpful assistant with access to tools.").
    WithPrompt(s.Prompt)
manager, _ := mb.Build()
conv := manager.GetConversation()
```

## Step 6 — Run the Router and Tool-Calling Loop

Run the router and the helper loop concurrently. Attach the sink to the context so helpers and tools can publish events.

```go
eg, groupCtx := errgroup.WithContext(ctx)

eg.Go(func() error { return router.Run(groupCtx) })
eg.Go(func() error {
    <-router.Running()
    runCtx := events.WithEventSinks(groupCtx, sink)
    updated, err := toolhelpers.RunToolCallingLoop(
        runCtx, eng, conv, registry, toolhelpers.NewToolConfig().WithMaxIterations(5),
    )
    if err != nil { return err }
    for _, m := range updated[len(conv):] {
        if err := manager.AppendMessages(m); err != nil { return err }
    }
    return nil
})

if err := eg.Wait(); err != nil { return err }
```

### Sample Text Output

```
assistant: Thinking…
assistant: I will check the current temperature.
call: get_weather {"location":"Paris","units":"celsius"}
result: {"temperature":22}
assistant: It’s about 22°C in Paris right now.
```

## Minimal Variants

- Without tools: call `eng.RunInference(ctx, conv)` and skip the helper loop.
- Structured streaming: use `events.NewStructuredPrinter` with `json` or `yaml` for machine-readable logs.

## Troubleshooting and Tips

- No output streaming? Ensure the engine is constructed with `engine.WithSink(sink)` and that the sink is also placed on the context via `events.WithEventSinks(...)`.
- Blank console? Confirm a handler is registered for the same topic you used when creating the sink (here: `"chat"`).
- Tool schemas not applied? Some providers don’t accept external tool definitions; fall back to the generic helper loop which inspects model output and invokes tools from the registry.
- Infinite loops: cap iterations with `toolhelpers.NewToolConfig().WithMaxIterations(n)`.

## See Also

- Engines and providers: `glaze help geppetto-inference-engines`
- Streaming and printers: `glaze help geppetto-events-streaming-watermill`
- Working example programs:
  - `geppetto/cmd/examples/simple-streaming-inference/main.go`
  - `geppetto/cmd/examples/openai-tools/main.go`
  - `geppetto/cmd/examples/claude-tools/main.go`
  - `geppetto/cmd/examples/generic-tool-calling/main.go`

If you need a full, copy-paste command, use the example apps above as a reference implementation and adapt the snippets here to your project structure.
