---
Title: Build a Streaming Tool-Loop Agent with Glazed Flags
Slug: build-streaming-tool-loop-agent-glazed-flags
Short: Step-by-step tutorial for building a small Glazed CLI that streams events, runs a tool loop, and keeps engine settings hidden in app bootstrap.
Topics:
- geppetto
- tutorial
- runner
- glazed
- streaming
- tools
- profiles
Commands:
- runner-glazed-registry-flags
Flags:
- prompt
- profile
- profile-registries
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial shows how to build a small streaming agent command with Glazed flags on top of Geppetto's opinionated runner API. The end result is a Cobra command that exposes only business-facing flags such as `prompt`, `profile`, and `profile-registries`, while the application keeps provider configuration and engine bootstrap hidden in app-owned code.

That boundary is important because Geppetto no longer treats profiles as engine-setting overlays. Profiles contribute runtime metadata such as system prompts, middleware uses, and tool names. Your application still owns the final `StepSettings` that create the engine. If you keep that split clear from the beginning, your CLI stays small, your help output stays readable, and your runtime policy remains explicit.

Use this pattern when you want:

- streaming output instead of a blocking one-shot CLI
- tool-calling behavior without hand-assembling `session.Session` and `enginebuilder.Builder`
- a Glazed command surface that exposes only the flags users actually need
- profile registries to select behavior, not provider credentials

The tutorial builds on the current example and helper code in:

- `geppetto/cmd/examples/runner-glazed-registry-flags/main.go`
- `geppetto/cmd/examples/internal/runnerexample/step_settings.go`
- `geppetto/cmd/examples/internal/runnerexample/profiles/basic.yaml`

## What You Will Build

You will build a command with this shape:

```text
glazed flags
  -> decode prompt/profile/profile-registries
  -> app-owned hidden StepSettings bootstrap
  -> resolve runtime metadata from profile registry
  -> register tools
  -> start streaming runner with event sink
  -> wait for final turn
```

The command remains small from the user's perspective:

```bash
go run ./cmd/examples/runner-glazed-registry-flags \
  runner-glazed-registry-flags \
  --profile teacher \
  --prompt "Use the tool if needed and explain the result."
```

## Why This Pattern Matters

There are two tempting mistakes when building this kind of CLI.

The first mistake is exposing the full Geppetto flag surface to every user. That is fine for diagnostics and low-level examples, but it makes small tools noisy. Most operator-facing CLIs do not want to expose provider, timeout, temperature, tool execution, and middleware wiring flags directly.

The second mistake is assuming that profile registries should create or mutate engine settings. That used to be a source of architectural confusion. The current model is simpler:

- app code owns base `StepSettings`
- profile registries own runtime metadata
- `pkg/inference/runner` consumes the already-resolved runtime

This tutorial uses that smaller model throughout.

## Prerequisites

Before you start, make sure you have:

- a Glazed/Cobra command binary or example app
- `OPENAI_API_KEY` set if your hidden base settings use OpenAI
- a profile registry YAML or SQLite source
- familiarity with the basic runner API in [Opinionated Runner API](../topics/10-runner.md)

If you need the background first, read these pages:

- [Opinionated Runner API](../topics/10-runner.md)
- [Profiles](../topics/01-profiles.md)
- [Events and Streaming](../topics/04-events.md)
- [Tools](../topics/07-tools.md)

## Architecture at a Glance

This is the flow we want:

```text
┌──────────────────────────────────────────────────────┐
│ Glazed command                                       │
│ flags: prompt, profile, profile-registries          │
└──────────────────────┬───────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────┐
│ App-owned bootstrap                                  │
│ hidden StepSettings from defaults/config/secrets     │
└──────────────────────┬───────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────┐
│ Profile registry resolution                          │
│ system prompt, middleware uses, tool names           │
└──────────────────────┬───────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────┐
│ runner.New(...).Start(...)                           │
│ engine + middleware + tools + session + sinks        │
└──────────────────────┬───────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────┐
│ Streaming events + final turn                        │
└──────────────────────────────────────────────────────┘
```

The key design choice is that the Glazed command owns the public flag layer, but the application owns engine bootstrap behind the scenes.

## Step 1 — Define the Small Public Flag Surface

Start by defining a Glazed command that only exposes what the operator should control directly. In this tutorial that is:

- the prompt to run
- the profile slug
- the registry source string

That means your command settings struct stays small:

```go
type agentSettings struct {
    Prompt            string `glazed:"prompt"`
    Profile           string `glazed:"profile"`
    ProfileRegistries string `glazed:"profile-registries"`
}
```

Then define a dedicated section for registry selection:

```go
func profileRegistrySettingsSection() (schema.Section, error) {
    return schema.NewSection(
        "profile-settings",
        "Profile settings",
        schema.WithFields(
            fields.New("profile", fields.TypeString,
                fields.WithHelp("Profile slug to resolve"),
                fields.WithDefault("concise"),
            ),
            fields.New("profile-registries", fields.TypeString,
                fields.WithHelp("Comma-separated profile registry sources"),
                fields.WithDefault(runnerexample.ExampleProfileRegistryPath()),
            ),
        ),
    )
}
```

Why this matters:

- it keeps `glaze help` output short and teachable
- it avoids leaking internal engine configuration into the public CLI
- it makes the contract obvious: the user chooses behavior, the app owns infrastructure

## Step 2 — Keep Base StepSettings Hidden in App Bootstrap

This is the architectural center of the tutorial.

Do not ask the registry to create engine settings. Instead, build base `StepSettings` in app code. In a production app that usually means some combination of:

- config file defaults
- environment variables
- secrets or deployment wiring
- a small number of hidden hardcoded defaults

The current example uses a defaults-only bootstrap helper:

```go
stepSettings, err := runnerexample.BaseStepSettingsFromDefaults()
if err != nil {
    return err
}
```

That helper mirrors Pinocchio's bootstrap shape:

```go
func BaseStepSettingsFromDefaults() (*settings.StepSettings, error) {
    sections_, err := geppettosections.CreateGeppettoSections()
    if err != nil { return nil, err }

    schema_ := schema.NewSchema(schema.WithSections(sections_...))
    parsedValues := values.New()

    err = sources.Execute(
        schema_,
        parsedValues,
        sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
    )
    if err != nil { return nil, err }

    return settings.NewStepSettingsFromParsedValues(parsedValues)
}
```

In a real binary, the same pattern often becomes:

```text
defaults
  + app config file
  + env vars
  + optional explicit config override
  -> final base StepSettings
```

That is the correct place to wire provider credentials, default model choice, client timeout, and related engine-level concerns.

## Step 3 — Resolve Runtime Metadata from the Profile Registry

Once you have base `StepSettings`, use the registry only for runtime metadata selection.

The example helper does exactly that:

```go
runtime, closeRegistry, err := runnerexample.ResolveRuntimeFromRegistry(
    ctx,
    stepSettings,
    s.ProfileRegistries,
    s.Profile,
)
if err != nil {
    return err
}
defer closeRegistry()
```

The important thing to understand is what this call contributes.

It does contribute:

- `SystemPrompt`
- `MiddlewareUses`
- `ToolNames`
- `RuntimeKey`
- `RuntimeFingerprint`
- `ProfileVersion`

It does not contribute:

- API keys
- provider selection policy
- HTTP client settings
- engine construction logic

Conceptually, this step looks like:

```text
base StepSettings (app-owned)
  + resolved profile runtime metadata
  -> runner.Runtime
```

That is much easier to reason about than the old “patch engine settings through the profile layer” model.

## Step 4 — Add One or More Tools

A streaming agent usually needs at least one tool to make the loop interesting. With the opinionated runner, the easiest path is to register a function tool when constructing the runner.

Example:

```go
type WeatherRequest struct {
    Location string `json:"location"`
}

type WeatherResponse struct {
    Summary string `json:"summary"`
}

func weatherTool(req WeatherRequest) (WeatherResponse, error) {
    return WeatherResponse{
        Summary: "Sunny and mild in " + req.Location,
    }, nil
}

r := runner.New(
    runner.WithFuncTool(
        "weather",
        "Return a short weather summary for a location",
        weatherTool,
    ),
)
```

This matters because the runner will build the tool registry for you, and the resolved profile can still decide whether the tool should be exposed by including or omitting the tool name in `runtime.ToolNames`.

The pattern is:

```text
tool registrars define what the app can do
profile runtime decides which of those tools are visible in this run
```

That separation keeps app capabilities and profile policy distinct.

## Step 5 — Add a Streaming Event Sink

If you want live output, call `Start(...)` instead of `Run(...)`.

You need an event sink implementation that receives events as inference progresses. For a minimal terminal-oriented example, a tiny stdout sink is enough:

```go
type stdoutSink struct{}

func (s *stdoutSink) PublishEvent(event events.Event) error {
    fmt.Printf("event: %s\n", event.Type())
    return nil
}
```

Then pass it into the start request:

```go
prepared, handle, err := r.Start(ctx, runner.StartRequest{
    Prompt:  s.Prompt,
    Runtime: runtime,
    EventSinks: []events.EventSink{
        &stdoutSink{},
    },
})
if err != nil {
    return err
}
```

Why this matters:

- `Start(...)` gives you immediate access to streaming behavior
- `prepared.Session` is available for inspection or custom coordination
- `handle.Wait()` still gives you the final turn after streaming completes

That makes the API work for both real-time UIs and CLI tools.

## Step 6 — Wait for the Final Turn

After starting the run, wait for completion and print the final turn.

```go
fmt.Printf("session: %s\n", prepared.Session.SessionID)

out, err := handle.Wait()
if err != nil {
    return err
}

turns.FprintTurn(w, out)
```

This is an important point for new contributors: streaming and final result handling are not competing patterns. They are the two halves of the same execution flow.

- the event sink gives you incremental visibility
- `Wait()` gives you the completed final turn

## Step 7 — Put It Together in a Glazed Command

Here is the combined shape in pseudocode:

```go
func (c *agentCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
    s := decodeAgentSettings(parsedValues)

    stepSettings, err := BaseStepSettingsFromDefaults()
    if err != nil {
        return err
    }

    runtime, closeRegistry, err := ResolveRuntimeFromRegistry(
        ctx,
        stepSettings,
        s.ProfileRegistries,
        s.Profile,
    )
    if err != nil {
        return err
    }
    defer closeRegistry()

    r := runner.New(
        runner.WithFuncTool("weather", "Weather lookup", weatherTool),
    )

    prepared, handle, err := r.Start(ctx, runner.StartRequest{
        Prompt:  s.Prompt,
        Runtime: runtime,
        EventSinks: []events.EventSink{
            &stdoutSink{},
        },
    })
    if err != nil {
        return err
    }

    fmt.Fprintf(w, "session: %s\n", prepared.Session.SessionID)

    out, err := handle.Wait()
    if err != nil {
        return err
    }

    turns.FprintTurn(w, out)
    return nil
}
```

This is the opinionated design target for a small agent CLI:

- Glazed owns flag parsing and help
- app code owns hidden engine bootstrap
- registry resolution owns runtime metadata selection
- runner owns execution assembly

## Step 8 — Decide Between Full Flags and Small Flags

Geppetto now has two Glazed example boundaries on purpose.

Use the full-flags pattern when:

- you are debugging engine configuration
- you need a power-user CLI
- the tool is mainly for developers

Use the registry-flags pattern when:

- the tool is operator-facing
- the app already knows how to bootstrap provider config
- you want small, stable help output
- profiles should select behavior, not infrastructure

That is the practical difference between:

- `cmd/examples/runner-glazed-full-flags/`
- `cmd/examples/runner-glazed-registry-flags/`

## Complete File Layout

A clean implementation usually ends up split like this:

```text
cmd/my-agent/main.go
cmd/my-agent/agent_command.go
cmd/my-agent/profile_section.go
internal/myagent/runtime_bootstrap.go
internal/myagent/tools.go
internal/myagent/events.go
pkg/doc/tutorials/build-streaming-tool-loop-agent-with-glazed-flags.md
```

This is useful because it prevents the Cobra/Glazed layer from swallowing all of the runtime logic in one large file.

## Failure Modes to Watch For

New contributors often hit the same mistakes:

- building `StepSettings` from the profile registry instead of from app bootstrap
- exposing too many low-level flags in the public command
- forgetting to close registry handles for SQLite-backed sources
- using `Run(...)` when they actually need streaming behavior from `Start(...)`
- registering tools but forgetting that the profile runtime may still filter visible tool names

If the command feels confused, check whether responsibilities are crossing boundaries:

- app bootstrap
- profile selection
- execution assembly
- event delivery

Each one should stay in its own layer.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| The command streams no events | You called `Run(...)` instead of `Start(...)`, or forgot to attach an event sink | Use `Start(...)` and pass `EventSinks` in the start request |
| The model ignores your tool | The tool was registered, but the resolved runtime filtered it out via `ToolNames` | Check the profile runtime and ensure the tool name appears in the resolved profile |
| The command asks for too many flags | You are using full Geppetto sections for a small CLI | Switch to the registry-flags pattern and keep `StepSettings` bootstrap hidden |
| Provider credentials are missing | The profile registry does not supply engine settings | Bootstrap base `StepSettings` from config/env/secrets in app code |
| The registry works in YAML but not SQLite | The registry handle may not be closed or the source string may be wrong | Verify the `profile-registries` source entry and `defer closeRegistry()` |

## See Also

- [Opinionated Runner API](../topics/10-runner.md) — Runner concepts, `Prepare`, `Start`, and `Run`
- [Profiles](../topics/01-profiles.md) — Registry-first runtime selection model
- [Events and Streaming](../topics/04-events.md) — Event sink and streaming design
- [Tools](../topics/07-tools.md) — Tool registration and runtime filtering
- [Bootstrap binary StepSettings from defaults, config, registries, and profile](../playbooks/08-bootstrap-binary-step-settings-from-defaults-config-registries-profile.md) — Hidden base settings bootstrap pattern

Example files:

- `geppetto/cmd/examples/runner-streaming/main.go`
- `geppetto/cmd/examples/runner-tools/main.go`
- `geppetto/cmd/examples/runner-glazed-registry-flags/main.go`
- `geppetto/cmd/examples/internal/runnerexample/step_settings.go`
