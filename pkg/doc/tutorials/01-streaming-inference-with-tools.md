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

This tutorial shows how to build a Cobra command that performs streaming inference and supports tool calling using Geppetto. It follows the engine-first architecture: engines handle provider I/O and emit events; helpers orchestrate tools.

For background, see:
- `glaze help geppetto-inference-engines`
- `glaze help geppetto-events-streaming-watermill`

## What You’ll Build

- A Cobra command that:
  - Parses provider config layers (Pinocchio profiles)
  - Starts a Watermill-backed event router
  - Creates an engine with an EventSink for streaming
  - Registers a sample tool
  - Runs the tool-calling loop with streaming
  - Prints streamed output to the console

## Prerequisites

- Basic Go and Cobra CLI knowledge
- Configured Pinocchio profile(s) for your provider
- Geppetto modules in your project

## 1) CLI Skeleton

```go
package main

import (
    "context"
    "fmt"
    "github.com/go-go-golems/geppetto/pkg/conversation/builder"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    geppettolayers "github.com/go-go-golems/geppetto/pkg/layers"
    "github.com/go-go-golems/glazed/pkg/cli"
    "github.com/go-go-golems/glazed/pkg/cmds"
    "github.com/go-go-golems/glazed/pkg/cmds/layers"
    "github.com/go-go-golems/glazed/pkg/cmds/parameters"
    "github.com/go-go-golems/glazed/pkg/help"
    "github.com/go-go-golems/glazed/pkg/cmds/logging"
    help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
    clay "github.com/go-go-golems/clay/pkg"
    "github.com/pkg/errors"
    "github.com/rs/zerolog/log"
    "github.com/spf13/cobra"
    "golang.org/x/sync/errgroup"
    "io"
)

type StreamingCmd struct { *cmds.CommandDescription }

type Settings struct {
    PinocchioProfile string `glazed.parameter:"pinocchio-profile"`
    Prompt           string `glazed.parameter:"prompt"`
    OutputFormat     string `glazed.parameter:"output-format"`
    WithMetadata     bool   `glazed.parameter:"with-metadata"`
}

func NewStreamingCmd() (*StreamingCmd, error) {
    geLayers, err := geppettolayers.CreateGeppettoLayers()
    if err != nil { return nil, err }
    desc := cmds.NewCommandDescription(
        "stream-with-tools",
        cmds.WithShort("Streaming inference with tools"),
        cmds.WithArguments(
            parameters.NewParameterDefinition("prompt", parameters.ParameterTypeString, parameters.WithHelp("Prompt")),
        ),
        cmds.WithFlags(
            parameters.NewParameterDefinition("pinocchio-profile", parameters.ParameterTypeString, parameters.WithDefault("4o-mini")),
            parameters.NewParameterDefinition("output-format", parameters.ParameterTypeString, parameters.WithDefault("text")),
            parameters.NewParameterDefinition("with-metadata", parameters.ParameterTypeBool, parameters.WithDefault(false)),
        ),
        cmds.WithLayersList(geLayers...),
    )
    return &StreamingCmd{CommandDescription: desc}, nil
}

func (c *StreamingCmd) RunIntoWriter(ctx context.Context, parsed *layers.ParsedLayers, w io.Writer) error {
    // Parse settings
    s := &Settings{}
    if err := parsed.InitializeStruct(layers.DefaultSlug, s); err != nil { return errors.Wrap(err, "init settings") }

    // 1) Event router + sink
    router, err := events.NewEventRouter()
    if err != nil { return errors.Wrap(err, "router") }
    defer router.Close()

    // Console printers
    if s.OutputFormat == "" || s.OutputFormat == "text" {
        // Handler signature is func(*message.Message) error
        router.AddHandler("chat", "chat", events.StepPrinterFunc("", w))
    } else {
        printer := events.NewStructuredPrinter(w, events.PrinterOptions{Format: events.PrinterFormat(s.OutputFormat), IncludeMetadata: s.WithMetadata})
        router.AddHandler("chat", "chat", printer)
    }

    sink := middleware.NewWatermillSink(router.Publisher, "chat")

    // 2) Engine
    eng, err := factory.NewEngineFromParsedLayers(parsed, engine.WithSink(sink))
    if err != nil { return errors.Wrap(err, "engine") }

    // 3) Tools (registry + one sample tool)
    registry := tools.NewInMemoryToolRegistry()
    weatherDef, err := tools.NewToolFromFunc("get_weather", "Get weather for a location", func(req struct{ Location, Units string }) struct{ Temperature float64 } {
        return struct{ Temperature float64 }{Temperature: 22.0}
    })
    if err != nil { return errors.Wrap(err, "tool") }
    if err := registry.RegisterTool("get_weather", *weatherDef); err != nil { return errors.Wrap(err, "register tool") }

    // Optionally configure engine tools (if provider supports built-in tool schemas)
    // ConfigureTools lets you pass tool definitions to engines that support it (e.g., OpenAI function calling).
    if cfg, ok := eng.(interface{ ConfigureTools([]engine.ToolDefinition, engine.ToolConfig) }); ok {
        var defs []engine.ToolDefinition
        for _, t := range registry.ListTools() {
            defs = append(defs, engine.ToolDefinition{Name: t.Name, Description: t.Description, Parameters: t.Parameters})
        }
        cfg.ConfigureTools(defs, engine.ToolConfig{Enabled: true})
    }

    // 4) Conversation
    mb := builder.NewManagerBuilder().
        WithSystemPrompt("You are a helpful assistant with access to tools.").
        WithPrompt(s.Prompt)
    manager, err := mb.Build()
    if err != nil { return errors.Wrap(err, "build conversation") }
    conv := manager.GetConversation()

    // 5) Run router + inference with tool-calling in parallel
    eg, groupCtx := errgroup.WithContext(ctx)

    eg.Go(func() error { return router.Run(groupCtx) })
    eg.Go(func() error {
        <-router.Running()
        runCtx := events.WithEventSinks(groupCtx, sink) // allow helpers/tools to publish
        updated, err := toolhelpers.RunToolCallingLoop(
            runCtx, eng, conv, registry, toolhelpers.NewToolConfig().WithMaxIterations(5),
        )
        if err != nil { return err }
        // Append new messages
        for _, m := range updated[len(conv):] {
            if err := manager.AppendMessages(m); err != nil { return err }
        }
        return nil
    })

    if err := eg.Wait(); err != nil { return err }
    log.Info().Msg("Finished")
    return nil
}

func main() {
    root := &cobra.Command{Use: "stream-with-tools", PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        return logging.InitLoggerFromViper()
    }}
    helpSystem := help.NewHelpSystem()
    help_cmd.SetupCobraRootCommand(helpSystem, root)
    _ = clay.InitViper("pinocchio", root)

    c, err := NewStreamingCmd()
    cobra.CheckErr(err)
    command, err := cli.BuildCobraCommand(c, cli.WithCobraMiddlewaresFunc(geppettolayers.GetCobraCommandGeppettoMiddlewares))
    cobra.CheckErr(err)
    root.AddCommand(command)
    cobra.CheckErr(root.Execute())
}
```

Key points:
- The engine is created with `engine.WithSink(...)`. This enables streaming events.
- The same sink is attached to the context with `events.WithEventSinks(...)` so tool helpers and tools can publish events.
- `toolhelpers.RunToolCallingLoop` manages the tool-calling workflow; you simply append the new messages to your `manager`.

## 2) Minimal Variants

- Without tools: call `eng.RunInference(ctx, conv)` and skip the helper loop.
- Structured output: use `events.NewStructuredPrinter` with `json`/`yaml` formats for machine-readable streams.

## 3) Explore Working Examples

Check the example programs shipped with the repo for complete references:
- `geppetto/cmd/examples/simple-streaming-inference/main.go`
- `geppetto/cmd/examples/openai-tools/main.go`
- `geppetto/cmd/examples/claude-tools/main.go`
- `geppetto/cmd/examples/generic-tool-calling/main.go`

These demonstrate provider-specific nuances, logging middleware, and different output formatting strategies.




