---
Title: Opinionated Runner API
Slug: runner
Short: App-facing `pkg/inference/runner` package for building simple and event-driven inference programs from resolved runtime input.
Topics:
- geppetto
- inference
- go-api
- middleware
- tools
- events
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Opinionated Runner API

`pkg/inference/runner` is the recommended starting point for new Geppetto applications that already know their final runtime configuration.

Use it when your application can already decide:

- final `StepSettings`
- final system prompt
- middleware list or middleware uses
- tool registrars and tool-name filtering

The package does not resolve profiles, merge runtime patches, or own application policy. It assembles Geppetto's existing primitives into a simpler app-facing API:

- `Prepare(...)` for advanced callers that need a prepared `session.Session`
- `Start(...)` for async and streaming/event-driven flows
- `Run(...)` for a simple blocking call

## Minimal Shape

```go
stepSettings := ...

r := runner.New(
    runner.WithFuncTool("calculator", "Basic arithmetic calculator", calculatorTool),
)

_, out, err := r.Run(ctx, runner.StartRequest{
    Prompt: "Use the calculator tool to multiply 17 by 23.",
    Runtime: runner.Runtime{
        StepSettings: stepSettings,
        SystemPrompt: "You are a concise assistant.",
        ToolNames:    []string{"calculator"},
    },
})
```

## Event-Driven Shape

```go
prepared, handle, err := r.Start(ctx, runner.StartRequest{
    Prompt: "Explain how streaming event sinks work.",
    Runtime: runner.Runtime{
        StepSettings: stepSettings,
    },
    EventSinks: []events.EventSink{sink},
})
```

Use `prepared.Session` if you need to inspect or extend the session lifecycle after preparation.

## Example Programs

See the focused example programs in:

- `cmd/examples/runner-simple/`
- `cmd/examples/runner-tools/`
- `cmd/examples/runner-streaming/`
- `cmd/examples/runner-registry/`
- `cmd/examples/runner-glazed/`
