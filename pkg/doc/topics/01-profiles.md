---
Title: Engine Profiles in Geppetto
Slug: profiles
Short: Engine-only profile registries for resolving `InferenceSettings` in Geppetto.
Topics:
- configuration
- engineprofiles
- inference
- registry
Commands:
- geppetto
Flags:
- profile
- profile-registries
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Engine Profiles in Geppetto

Geppetto now treats profiles as **engine profiles** only.

An engine profile answers one question:

- what `InferenceSettings` should be used to build the engine?

It does **not** answer:

- what system prompt to inject
- what middlewares to enable
- what tools to expose
- what runtime key or runtime fingerprint to stamp

Those are application concerns and belong in Pinocchio, GEC-RAG, Temporal Relationships, or another host.

## Core Model

The engine profile domain lives in [pkg/engineprofiles](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles).

The key types are:

- `EngineProfile`
- `EngineProfileRegistry`
- `ResolvedEngineProfile`
- `ResolveInput`
- `InferenceSettings`

Conceptually:

```text
engine profile registry
  -> resolve one engine profile slug
  -> expand stack layers
  -> merge engine settings
  -> produce final InferenceSettings
```

## Data Shape

An engine profile contains:

- `slug`
- `display_name`
- `description`
- `stack`
- `inference_settings`
- `metadata`
- `extensions`

Minimal YAML:

```yaml
slug: provider-openai
profiles:
  default:
    slug: default
    inference_settings:
      api:
        api_keys:
          openai-api-key: demo-openai-key
      chat:
        api_type: openai
        engine: gpt-4o-mini
```

Stacked profile:

```yaml
slug: team-agent
profiles:
  assistant:
    slug: assistant
    stack:
      - registry_slug: provider-openai
        profile_slug: default
    inference_settings:
      chat:
        api_type: openai-responses
        engine: gpt-5-mini
```

## What Resolution Returns

`ResolveEngineProfile(...)` returns:

- `registrySlug`
- `profileSlug`
- merged `inferenceSettings`
- metadata like `profile.registry`, `profile.slug`, `profile.version`, and stack lineage

It no longer returns:

- `effectiveRuntime`
- `runtimeKey`
- `runtimeFingerprint`

Those were part of the older mixed runtime model and were removed in the hard cut.

## Why the Split Matters

The old model conflated two different domains:

- engine selection and provider configuration
- app runtime behavior

That caused Geppetto to know too much about system prompts, middleware registries, tool allowlists, and app-level runtime metadata.

The current split is:

```text
Geppetto
  engine profile registry
  -> InferenceSettings
  -> engine

App
  system prompt
  middlewares
  tools
  runtime metadata
  -> actual run behavior
```

## Package Boundaries

Geppetto owns:

- engine profile registry loading
- YAML / SQLite / in-memory registry stores
- stack expansion and merge
- `InferenceSettings`
- engine construction

Apps own:

- prompt policy
- middleware selection
- tool registries and filtering
- runtime keys / fingerprints / cache identity
- per-app YAML for runtime presets if they want that

## API References

Relevant files:

- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/types.go)
- [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/registry.go)
- [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/service.go)
- [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/stack_merge.go)
- [inference_settings_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/inference_settings_merge.go)
- [settings-inference.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/settings-inference.go)

## Typical Flow

Go pseudocode:

```go
entries, _ := engineprofiles.ParseEngineProfileRegistrySourceEntries(rawSources)
specs, _ := engineprofiles.ParseRegistrySourceSpecs(entries)
chain, _ := engineprofiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
defer chain.Close()

resolved, _ := chain.ResolveEngineProfile(ctx, engineprofiles.ResolveInput{
    EngineProfileSlug: engineprofiles.MustEngineProfileSlug("assistant"),
})

eng, _ := enginefactory.NewEngineFromSettings(resolved.InferenceSettings)
```

Then the app adds its own runtime behavior on top:

```text
resolved engine profile -> engine
app runtime config -> prompt + middleware + tools
engine + app runtime -> session / runner
```

## Base Settings vs Profile Overlay

This section explains the most important lifecycle distinction behind the profile system.

Geppetto profile resolution is not "the profile is the whole runtime." The profile is an overlay. The host application is expected to keep a baseline `InferenceSettings` object and merge the resolved profile on top of it.

Conceptually:

```text
app-owned base InferenceSettings
  + resolved engine-profile InferenceSettings overlay
  = final InferenceSettings used to build the engine
```

That is why profile docs should be read together with bootstrap docs:

- Geppetto owns the shared settings schema and the merge contract.
- The host application owns how the baseline is collected from defaults, config, environment, and flags.

This distinction matters whenever a setting is cross-profile and should survive profile changes. Transport and operator settings such as `ai-client.*` belong in the app-owned baseline, not in engine profiles. Model-selection defaults such as `chat.engine` are much more natural as profile overlays.

## Hidden Base Settings

Applications sometimes need a baseline even when a command does not visibly expose the whole Geppetto AI surface on its CLI.

The bootstrap helpers in `geppetto/pkg/cli/bootstrap` support that pattern by reconstructing a hidden base `InferenceSettings` from the shared Geppetto sections plus config, environment, and defaults. The important consequence is:

- a setting can still be part of the effective base even when the command intentionally exposes a narrower public CLI.

That is why the ownership boundary matters so much. If a field belongs in a shared section such as `ai-client`, it can participate naturally in the hidden base lifecycle.

## Practical Rule: What Belongs In Profiles

Use this rule of thumb:

- Put model and provider behavior in profiles.
- Put app/operator infrastructure in the shared baseline.

Examples that fit well in profiles:

- `chat.engine`
- `chat.api_type`
- provider-specific request defaults

Examples that fit better in the shared baseline:

- `ai-client.timeout`
- provider credentials and base URLs
- proxy configuration

## See Also

- `geppetto/pkg/doc/tutorials/09-migrating-cli-commands-to-glazed-bootstrap-profile-resolution.md` for the bootstrap/base/final settings lifecycle
- `pinocchio/pkg/doc/topics/pinocchio-profile-resolution-and-runtime-switching.md` for the application-side runtime profile-switch pattern built on top of this Geppetto model
