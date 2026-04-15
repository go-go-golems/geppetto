---
Title: CLI Command Migration Guide (Glazed + Bootstrap + Profiles)
Slug: geppetto-cli-bootstrap-profile-migration
Short: Step-by-step guide to migrate a Cobra command to the current Glazed + bootstrap path with profile resolution and shared inference-debug output.
Topics:
- geppetto
- cli
- glazed
- bootstrap
- profiles
- config
- migration
- tutorial
Commands:
- geppetto
Flags:
- config-file
- profile
- profile-registries
- print-parsed-fields
- print-inference-settings
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial shows how to migrate a CLI command from raw Cobra flags to the current Glazed + bootstrap path. The core APIs live in `geppetto/pkg/cli/bootstrap`, and the concrete reference implementations are the current Pinocchio JS command plus a downstream app-owned backend.

The target is not just “support `--profile`.” The target is a command that:

- parses flags and positional arguments through Glazed,
- resolves config and profiles through a single bootstrap contract,
- creates engines from the final merged `InferenceSettings`,
- and can print one combined inference-debug document for troubleshooting.

## What You Are Building

A migrated command has five layers:

1. A Glazed `CommandDescription` for the command’s own flags and arguments.
2. Shared Geppetto sections for inference settings and profile selection.
3. An app-owned middleware chain that merges Cobra, arguments, environment, config files, and defaults.
4. A single `ResolveCLIEngineSettings(...)` call that produces the final merged `InferenceSettings`.
5. An optional shared inference-debug path owned by `geppetto/pkg/cli/bootstrap`.

The most useful reference files are:

- [pkg/cli/bootstrap/config.go](../../cli/bootstrap/config.go)
- [pkg/cli/bootstrap/engine_settings.go](../../cli/bootstrap/engine_settings.go)
- [pkg/cli/bootstrap/inference_debug.go](../../cli/bootstrap/inference_debug.go)
- [pkg/sections/sections.go](../../sections/sections.go)
- `pinocchio/cmd/pinocchio/cmds/js.go`

## Ownership Boundary

This is the most important concept to keep straight.

Geppetto owns the generic bootstrap behavior:

- `AppBootstrapConfig`
- profile/config resolution
- final merged engine settings
- inference-debug output

Applications own only their app-specific bootstrap wiring:

- app name and env prefix
- config-file mapper
- command-specific runtime behavior

That means the reusable migration pattern belongs in Geppetto. Application packages should pass an app-owned `AppBootstrapConfig` into the Geppetto helper rather than recreating local copies of bootstrap logic.

## Why This Migration Matters

Raw Cobra commands drift over time in predictable ways:

- config loading differs from one command to another,
- profile selection and registry handling diverge,
- engine creation bypasses merged settings,
- and debugging becomes inconsistent.

The shared bootstrap path gives every command the same answers to the same questions:

- Which config file was loaded?
- Which profile registries were used?
- Which profile actually resolved?
- What are the final `InferenceSettings`?
- Why does a specific field have its current value?

## Step 1: Replace Local Cobra Flags with a Glazed Command Description

Stop declaring the normal command flags with `cmd.Flags().StringVar(...)`. Move them into a `CommandDescription`.

```go
type MyCommand struct {
	*cmds.CommandDescription
}

func newMyCommand() (*MyCommand, error) {
	baseSections, err := geppettosections.CreateGeppettoSections()
	if err != nil {
		return nil, err
	}
	inferenceDebugSection, err := geppettobootstrap.NewInferenceDebugSection()
	if err != nil {
		return nil, err
	}

	commandOptions := []cmds.CommandDescriptionOption{
		cmds.WithShort("..."),
		cmds.WithFlags(
			fields.New("my-flag", fields.TypeString),
		),
		cmds.WithSections(inferenceDebugSection),
	}
	commandOptions = append(commandOptions, cmds.WithSections(baseSections...))

	return &MyCommand{
		CommandDescription: cmds.NewCommandDescription("my-command", commandOptions...),
	}, nil
}
```

That does two things:

- the command-specific flags become declarative and Glazed-owned,
- and the command becomes compatible with `cli.BuildCobraCommand(...)`.

## Step 2: Reuse the Shared Geppetto Sections

For a command that needs inference/profile resolution, start with:

- `geppetto/pkg/sections.CreateGeppettoSections()`

That gives you:

- `ai-chat`
- provider sections like `openai-chat`
- `ai-inference`
- `profile-settings`

If you also want inference debug output, add:

- `geppetto/pkg/cli/bootstrap.NewInferenceDebugSection()`

That gives you:

- `--print-inference-settings`

Because `CreateGeppettoSections()` already includes `profile-settings`, you do not need to define your own `--profile` or `--profile-registries` flags.

## Step 3: Build Cobra Through Glazed

This is the point where `--print-parsed-fields` starts working.

```go
func NewMyCommand() *cobra.Command {
	command, err := newMyCommand()
	if err != nil {
		panic(err)
	}
	cobraCommand, err := cli.BuildCobraCommand(command, cli.WithParserConfig(cli.CobraParserConfig{
		MiddlewaresFunc: myMiddlewares,
	}))
	if err != nil {
		panic(err)
	}
	return cobraCommand
}
```

This enables:

- `--print-parsed-fields`
- `--print-schema`
- `--print-yaml`
- normal Glazed command parsing

If the command still bypasses `cli.BuildCobraCommand(...)`, it will keep behaving like a special-case Cobra command.

## Step 4: Define an App-Owned Bootstrap Contract

Every application needs to define how Geppetto should load its config and sections. That contract is `AppBootstrapConfig`.

```go
func appBootstrapConfig() geppettobootstrap.AppBootstrapConfig {
	return geppettobootstrap.AppBootstrapConfig{
		AppName:          "my-app",
		EnvPrefix:        "MY_APP",
		ConfigFileMapper: myConfigMapper,
		NewProfileSection: func() (schema.Section, error) {
			return geppettosections.NewProfileSettingsSection()
		},
		BuildBaseSections: func() ([]schema.Section, error) {
			return geppettosections.CreateGeppettoSections()
		},
	}
}
```

Two details matter:

- `ConfigFileMapper` must translate your app’s config file into section maps.
- `BuildBaseSections` must return the hidden base sections that participate in inference resolution and provenance.

If you are working in Pinocchio, the application-specific bootstrap contract already exists as `profilebootstrap.BootstrapConfig()`.

## Hidden Base vs Final Settings

This section explains the central lifecycle concept behind the bootstrap APIs before the tutorial continues with middleware wiring.

The bootstrap path intentionally keeps two different settings objects in play:

- `BaseInferenceSettings`
- `FinalInferenceSettings`

The mental model is:

```text
shared sections + app config/env/defaults
  -> base inference settings

base inference settings + resolved engine profile overlay
  -> final inference settings
```

That is the reason `AppBootstrapConfig.BuildBaseSections` matters so much. It defines which shared Geppetto sections are eligible to participate in the hidden base.

Sequence sketch:

```text
BuildBaseSections()
  -> fresh schema
  -> parse env + config + defaults
  -> BaseInferenceSettings
  -> resolve profile registry selection
  -> merge profile overlay
  -> FinalInferenceSettings
```

Practical implications:

- Put cross-profile settings in shared Geppetto sections such as `ai-client`.
- Build engines from `FinalInferenceSettings`, not from raw profile payloads.
- Do not assume the profile itself is the whole runtime configuration.

Some host applications then add an additional runtime pattern on top of this: they preserve a profile-free base reconstructed from already parsed values so they can switch profiles later without losing CLI-supplied overrides. That runtime-switch pattern is application-owned, not Geppetto-owned. Pinocchio documents that companion pattern in `pinocchio/pkg/doc/topics/pinocchio-profile-resolution-and-runtime-switching.md`.

## Step 5: Use an App-Aware Config Middleware

The middleware chain should load:

- Cobra values,
- positional arguments,
- environment variables,
- config files,
- and defaults.

The exact config-file mapper is application-specific. A generic shape looks like this:

```go
func myMiddlewares(parsed *values.Values, cmd *cobra.Command, args []string) ([]cmd_sources.Middleware, error) {
	return []cmd_sources.Middleware{
		cmd_sources.FromCobra(cmd, fields.WithSource("cobra")),
		cmd_sources.FromArgs(args, fields.WithSource("arguments")),
		cmd_sources.FromEnv("MY_APP", fields.WithSource("env")),
		cmd_sources.FromConfigPlanBuilder(
			appBootstrapConfig().ConfigPlanBuilder,
			cmd_sources.WithConfigFileMapper(appBootstrapConfig().ConfigFileMapper),
			cmd_sources.WithParseOptions(fields.WithSource("config")),
		),
		cmd_sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	}, nil
}
```

If you are working in Pinocchio, the important application-specific detail is still:

- `profilebootstrap.MapPinocchioConfigFile(...)`

because Pinocchio config files contain non-section keys like `repositories`.

## Step 6: Resolve Final Engine Settings Once

Do not reimplement profile merging inside the command.

After parsing, call:

```go
resolved, err := geppettobootstrap.ResolveCLIEngineSettings(ctx, appBootstrapConfig(), parsed)
if err != nil {
	return err
}
if resolved.Close != nil {
	defer resolved.Close()
}
```

The result gives you:

- `resolved.ProfileSelection`
- `resolved.BaseInferenceSettings`
- `resolved.FinalInferenceSettings`
- `resolved.ResolvedEngineProfile`

For a normal engine-backed command, create engines from `resolved.FinalInferenceSettings`.

The rule to keep is simple:

- build engines from the final merged settings, not from raw profile payload alone.

## Step 7: Use the Shared Inference Debug Helper

The current debug surface is intentionally simple:

- `--print-parsed-fields` from Glazed
- `--print-inference-settings` from `geppetto/pkg/cli/bootstrap`

`--print-inference-settings` prints a combined document with:

- `settings`
- `sources`

Sensitive values are masked as `***`.

The generic pattern is:

```go
debugSettings := &geppettobootstrap.InferenceDebugSettings{}
if err := parsed.DecodeSectionInto(geppettobootstrap.InferenceDebugSectionSlug, debugSettings); err != nil {
	return err
}
if debugSettings.PrintInferenceSettings {
	_, err := geppettobootstrap.HandleInferenceDebugOutput(
		w,
		appBootstrapConfig(),
		parsed,
		*debugSettings,
		resolved,
		geppettobootstrap.InferenceDebugOutputOptions{},
	)
	return err
}
```

Two details matter here:

1. The helper is Geppetto-owned, not app-owned.
2. The helper reconstructs hidden-base parsed values for provenance internally, so callers usually do not need to do that themselves.

If your command injects an extra command-specific baseline before profile overlay, pass it as `InferenceDebugOutputOptions.CommandBase`. Otherwise leave it nil.

## Step 8: If the Command Exposes Runtime Profile APIs, Pass the Defaults Through

This is the subtle runtime bug that showed up in Pinocchio JS, but the rule is generic.

If your runtime can create engines later from selected profiles, do not feed it raw profile payload alone. Pass the CLI’s final merged settings into the runtime as defaults and merge from there.

Again, the rule is:

- build engines from the final merged settings, not from raw profile payload alone.

## Pinocchio Companion

If you are starting from the Pinocchio repository, also read the companion guide in `pinocchio/pkg/doc/tutorials/07-migrating-cli-verbs-to-glazed-profile-bootstrap.md`. That document focuses only on the Pinocchio-specific deltas:

- `profilebootstrap.BootstrapConfig()`
- `profilebootstrap.MapPinocchioConfigFile(...)`
- `profilebootstrap.ResolveCLIEngineSettings(...)`
- Pinocchio JS runtime defaults

## Validation Checklist

Run these after the migration:

1. `go run ./cmd/<app> <command> --help --long-help`
2. `go run ./cmd/<app> <command> --print-parsed-fields`
3. `go run ./cmd/<app> <command> --print-inference-settings`
4. Confirm the `--print-inference-settings` output includes both `settings:` and `sources:`
5. Confirm sensitive values are masked as `***`
6. Run the repo-local tests for the command and bootstrap package

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `unknown flag: --print-parsed-fields` | The command still bypasses `cli.BuildCobraCommand(...)` | Convert it to a Glazed command and build Cobra through `cli.BuildCobraCommand(...)` |
| App config parsing fails on non-section keys | The config-file mapper is missing or too naive | Define an app-specific `ConfigFileMapper` in `AppBootstrapConfig` |
| `must be configured when profile-settings.profile is set` | A profile was selected without any registry sources | Mount `profile-settings` and configure `profile-registries` via config or flags |
| Final engine misses API key or base URL | The command built from raw profile data instead of merged settings | Create engines from `resolved.FinalInferenceSettings` |
| `--print-inference-settings` shows weak provenance | The command did not resolve settings through the shared bootstrap path | Use `ResolveCLIEngineSettings(...)` and `HandleInferenceDebugOutput(...)` from `geppetto/pkg/cli/bootstrap` |
| Downstream app loads the wrong config namespace | The app copied another app’s assumptions instead of defining its own bootstrap config | Create an app-owned `AppBootstrapConfig` with the right app name, env prefix, and config mapper |

## See Also

- [pkg/cli/bootstrap/config.go](../../cli/bootstrap/config.go)
- [pkg/cli/bootstrap/engine_settings.go](../../cli/bootstrap/engine_settings.go)
- [pkg/cli/bootstrap/inference_debug.go](../../cli/bootstrap/inference_debug.go)
- `pinocchio/cmd/pinocchio/cmds/js.go`
