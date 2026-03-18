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

The package does not resolve profiles into engine settings, merge runtime patches, or own application policy. It assembles Geppetto's existing primitives into a simpler app-facing API:

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

In practice this usually means one of two app-owned bootstrap patterns:

- full-flags CLI: expose full Geppetto sections and build `StepSettings` from parsed Glazed values
- small CLI: keep `StepSettings` bootstrap hidden in app code or app config, and expose only profile selection or a few business flags publicly

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

## Registry-Only Glazed Shape

The small-CLI pattern is important: profiles no longer build provider settings for you. Instead, the app bootstraps hidden base `StepSettings`, then lets registries contribute prompt/tool/middleware runtime metadata.

```go
func run(ctx context.Context, parsedValues *values.Values) error {
    profileSettings := decodeProfileSettings(parsedValues)

    // App-owned hidden bootstrap. In a real app this might read config files,
    // secrets, or deployment defaults. The example uses defaults-only bootstrap.
    stepSettings, err := runnerexample.BaseStepSettingsFromDefaults()
    if err != nil { return err }

    runtime, closeRegistry, err := runnerexample.ResolveRuntimeFromRegistry(
        ctx,
        stepSettings,
        profileSettings.ProfileRegistries,
        profileSettings.Profile,
    )
    if err != nil { return err }
    defer closeRegistry()

    _, out, err := runner.New().Run(ctx, runner.StartRequest{
        Prompt:  profileSettings.Prompt,
        Runtime: runtime,
    })
    if err != nil { return err }
    _ = out
    return nil
}
```

That is the intended opinionated shape for apps like Pinocchio:

- hidden `StepSettings` bootstrap stays app-owned
- profile registries stay focused on runtime metadata
- `pkg/inference/runner` consumes the already-resolved runtime

## Example Programs

See the focused example programs in:

- `cmd/examples/runner-simple/`
- `cmd/examples/runner-tools/`
- `cmd/examples/runner-streaming/`
- `cmd/examples/runner-registry/`
- `cmd/examples/runner-glazed-full-flags/`
- `cmd/examples/runner-glazed-registry-flags/` for the small-CLI Glazed pattern where only registry selection is public
