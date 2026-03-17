---
Title: "Bootstrap binary StepSettings from defaults, config, registries, and profile"
Slug: bootstrap-binary-step-settings-defaults-config-registries-profile
Short: Playbook for wiring a Cobra/Glazed binary so schema defaults feed config, config feeds registry selection, and the selected profile produces the final StepSettings.
Topics:
  - configuration
  - profiles
  - step-settings
  - glazed
Commands:
  - geppetto
  - pinocchio
Flags:
  - config-file
  - profile
  - profile-registries
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Playbook
---

## Goal

Set up a binary so runtime configuration resolves in a predictable order:

1. schema defaults create the base shape,
2. config files and env refine that base,
3. registry selection resolves the effective profile,
4. the profile patch produces the final `*settings.StepSettings`.

This matters because many regressions come from mixing these layers together. A binary that parses flags, config, and profiles in one pass usually ends up with unclear precedence, hidden fallbacks, or profile patches applied twice.

## Before you start

- Decide which settings stay public on the CLI.
- Keep `--config-file`, `--profile`, and `--profile-registries` if you want a small public surface.
- Decide whether your binary should support `PINOCCHIO_*` or another env prefix.
- Use layered config shape (`ai-chat:`, `openai-chat:`, `claude-chat:`), not legacy flat keys.

## Recommended architecture

Treat runtime bootstrapping as two separate phases:

- **Phase 1: public command parse**
  Use Cobra/Glazed to parse the binary's public flags.
- **Phase 2: hidden runtime parse**
  Reuse Geppetto sections internally to build base `StepSettings`, then hand that base to profile resolution.

The resulting precedence should look like this:

```text
Geppetto section defaults
        ->
config files
        ->
env
        ->
base StepSettings
        ->
profile registry chain
        ->
selected effective profile
        ->
resolved.EffectiveStepSettings
```

If you also support request overrides, they belong inside profile resolution, not as a post-processing patch in the binary.

## Step 1: Keep the public CLI small

Parse only the flags your binary should expose directly. A common pattern is:

- server/runtime flags owned by the binary,
- `--config-file`,
- `--profile`,
- `--profile-registries`.

Do not expose every AI/provider field just because Geppetto has a section for it. Those settings can still participate in resolution through the hidden parse.

Example Cobra parser setup:

```go
command, err := cli.BuildCobraCommand(cmdDef, cli.WithParserConfig(cli.CobraParserConfig{
	AppName: "pinocchio",
	ConfigFilesFunc: func(_ *values.Values, _ *cobra.Command, _ []string) ([]string, error) {
		// Disable builder-owned config loading when you want to own
		// the hidden runtime parse yourself.
		return nil, nil
	},
}))
if err != nil {
	return err
}
```

Why this matters: if both the public parser and your hidden runtime bootstrap load config files, you will get duplicated or conflicting precedence.

## Step 2: Build Geppetto sections for the hidden runtime parse

Use `sections.CreateGeppettoSections()` as the schema source of truth. That keeps defaults and field names aligned with the rest of Geppetto.

```go
geppettoSections, err := geppettosections.CreateGeppettoSections()
if err != nil {
	return err
}

runtimeSchema := schema.NewSchema(schema.WithSections(geppettoSections...))
runtimeValues := values.New()
```

Why this matters: `settings.NewStepSettings()` creates the struct shape, but parsed section defaults are still the most reliable way to reproduce the same defaults/config/env behavior the rest of the Glazed stack expects.

## Step 3: Resolve config files explicitly

Build the config file list yourself in low-to-high precedence order. For example:

- implicit app config such as `~/.pinocchio/config.yaml`,
- explicit `--config-file`.

Example helper:

```go
func resolveRuntimeConfigFiles(parsed *values.Values) []string {
	files := []string{}

	if implicit := defaultPinocchioConfigFileIfPresent(); implicit != "" {
		files = append(files, implicit)
	}

	commandSettings := &cli.CommandSettings{}
	if err := parsed.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err == nil {
		if explicit := strings.TrimSpace(commandSettings.ConfigFile); explicit != "" {
			files = append(files, explicit)
		}
	}

	return files
}
```

If your app config contains non-section top-level keys, add a config mapper that filters them before handing the file to Geppetto sections.

## Step 4: Execute the hidden parse in the right middleware order

Geppetto/Glazed source middleware is ordered by precedence, not by visual reading order. The pattern below yields:

- defaults first,
- then config,
- then env.

```go
configFiles := resolveRuntimeConfigFiles(parsed)

if err := cmdsources.Execute(
	runtimeSchema,
	runtimeValues,
	cmdsources.FromEnv("PINOCCHIO", fields.WithSource("env")),
	cmdsources.FromFiles(
		configFiles,
		cmdsources.WithConfigFileMapper(configMapper),
		cmdsources.WithParseOptions(fields.WithSource("config")),
	),
	cmdsources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
); err != nil {
	return err
}
```

Then convert the parsed values into base settings:

```go
baseStepSettings, err := settings.NewStepSettingsFromParsedValues(runtimeValues)
if err != nil {
	return err
}
```

At this point you have the base runtime state for the binary, before profile-specific policy and patching.

## Step 5: Build the registry chain separately from base settings

Registry selection is not the same problem as building base settings. Keep it explicit.

```go
entries, err := profiles.ParseProfileRegistrySourceEntries(profileRegistries)
if err != nil {
	return err
}

specs, err := profiles.ParseRegistrySourceSpecs(entries)
if err != nil {
	return err
}

registry, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
if err != nil {
	return err
}
defer func() { _ = registry.Close() }()
```

Why this matters: the binary should be able to answer two separate questions clearly:

- what are the base settings before profile selection?
- what profile registry chain should be consulted?

## Step 6: Resolve the effective profile with `BaseStepSettings`

This is the key step. Do not manually reimplement patch layering if you can avoid it. Let the registry service resolve the effective profile and hand it the base settings you already computed.

```go
resolved, err := registry.ResolveEffectiveProfile(ctx, profiles.ResolveInput{
	RegistrySlug:     selectedRegistrySlug,
	ProfileSlug:      selectedProfileSlug,
	BaseStepSettings: baseStepSettings,
})
if err != nil {
	return err
}

effective := resolved.EffectiveStepSettings
```

This gives you a final `StepSettings` with the correct layering:

- base defaults/config/env already applied,
- profile stack already expanded,
- profile `runtime.step_settings_patch` already merged.

## Step 7: Use `resolved.EffectiveStepSettings` as the only runtime truth

Once profile resolution succeeds, stop reading raw flags/config for AI runtime fields. Use:

- `resolved.EffectiveStepSettings` for engine construction,
- `resolved.EffectiveRuntime` for system prompt, tools, and middleware selections,
- `resolved.RuntimeFingerprint` / `resolved.Metadata` for tracing and caching.

Example:

```go
eng, err := infruntime.BuildEngineFromSettingsWithMiddlewares(
	ctx,
	resolved.EffectiveStepSettings,
	resolved.EffectiveRuntime.SystemPrompt,
	resolvedMiddlewares,
)
if err != nil {
	return err
}
```

Why this matters: once the binary falls back to reading `parsed` or ad-hoc config again after resolution, precedence becomes impossible to reason about.

## Complete bootstrap sketch

This is the end-to-end shape:

```go
func buildEffectiveStepSettings(
	ctx context.Context,
	parsed *values.Values,
	profileRegistries string,
	selectedProfile profiles.ProfileSlug,
) (*settings.StepSettings, error) {
	geppettoSections, err := geppettosections.CreateGeppettoSections()
	if err != nil {
		return nil, err
	}

	runtimeSchema := schema.NewSchema(schema.WithSections(geppettoSections...))
	runtimeValues := values.New()
	configFiles := resolveRuntimeConfigFiles(parsed)

	if err := cmdsources.Execute(
		runtimeSchema,
		runtimeValues,
		cmdsources.FromEnv("PINOCCHIO", fields.WithSource("env")),
		cmdsources.FromFiles(
			configFiles,
			cmdsources.WithConfigFileMapper(configMapper),
			cmdsources.WithParseOptions(fields.WithSource("config")),
		),
		cmdsources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	); err != nil {
		return nil, err
	}

	baseStepSettings, err := settings.NewStepSettingsFromParsedValues(runtimeValues)
	if err != nil {
		return nil, err
	}

	entries, err := profiles.ParseProfileRegistrySourceEntries(profileRegistries)
	if err != nil {
		return nil, err
	}
	specs, err := profiles.ParseRegistrySourceSpecs(entries)
	if err != nil {
		return nil, err
	}

	registry, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = registry.Close() }()

	resolved, err := registry.ResolveEffectiveProfile(ctx, profiles.ResolveInput{
		ProfileSlug:       selectedProfile,
		BaseStepSettings:  baseStepSettings,
	})
	if err != nil {
		return nil, err
	}

	return resolved.EffectiveStepSettings, nil
}
```

## Failure modes to avoid

- **Parsing config twice**
  One parse in the Cobra layer and another hidden parse will produce surprising overrides.
- **Applying profile patches manually after `ResolveEffectiveProfile`**
  The registry service already did that merge.
- **Building engines from raw parsed values after profile resolution**
  This skips stack merge behavior.
- **Mixing legacy flat config keys with layered section config**
  `openai-api-key: ...` at top level is not the same as `openai-chat.openai-api-key`.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| final runtime ignores `--config-file` | builder-owned config loading was disabled, but hidden parse never consumed `command-settings.config-file` | decode `cli.CommandSettingsSlug` from the public parse and build the hidden config file list explicitly |
| profile patch seems to do nothing | binary builds engine from `baseStepSettings` instead of `resolved.EffectiveStepSettings` | treat `resolved.EffectiveStepSettings` as the only post-resolution runtime source |
| defaults disappeared after refactor | binary switched from parsed section flow to `settings.NewStepSettings()` only | restore hidden section parsing with `CreateGeppettoSections()` + `NewStepSettingsFromParsedValues(...)` |
| runtime differs between local shell and CI | env vars are shaping the base layer differently | log config file list and selected metadata, then compare env in both environments |
| unknown profile fails late | binary delays registry resolution until engine construction | resolve the effective profile before engine build and surface profile errors directly |
| config file loads but some keys are ignored | top-level YAML does not match section slugs or needs a mapper | use layered config shape or add `WithConfigFileMapper(...)` |

## See Also

- [Profile Registry in Geppetto](../topics/01-profiles.md)
- [Migrate to Geppetto Sections and Values](../tutorials/05-migrating-to-geppetto-sections-and-values.md)
- [Migrate legacy profiles.yaml to registry](../playbooks/05-migrate-legacy-profiles-yaml-to-registry.md)
- [Operate SQLite-backed profile registry](../playbooks/06-operate-sqlite-profile-registry.md)
- [Wire provider credentials for JS and go runner](../playbooks/07-wire-provider-credentials-for-js-and-go-runner.md)
