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
