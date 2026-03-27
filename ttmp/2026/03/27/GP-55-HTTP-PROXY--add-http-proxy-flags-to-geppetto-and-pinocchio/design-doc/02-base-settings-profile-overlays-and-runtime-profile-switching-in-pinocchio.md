---
Title: Base settings, profile overlays, and runtime profile switching in Pinocchio
Ticket: GP-55-HTTP-PROXY
Status: active
Topics:
    - geppetto
    - pinocchio
    - glazed
    - config
    - inference
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/cli/bootstrap/bootstrap_test.go
      Note: Provides the parity test showing that resolve-from-base matches the direct resolution path
    - Path: geppetto/pkg/cli/bootstrap/engine_settings.go
      Note: Implements hidden base reconstruction and final profile overlay resolution
    - Path: geppetto/pkg/engineprofiles/inference_settings_merge.go
      Note: Defines merge semantics for overlaying engine profile settings onto a preserved base
    - Path: geppetto/pkg/sections/sections.go
      Note: Defines the shared Geppetto sections used to reconstruct hidden base settings and mount command middleware
    - Path: geppetto/pkg/steps/ai/settings/settings-inference.go
      Note: Defines InferenceSettings and the section-to-struct decoding model that underpins base and final settings
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Demonstrates the hidden base path on a command with a narrower visible CLI surface
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: Shows the standard command startup path from parsed values to base and final settings
    - Path: pinocchio/pkg/cmds/profile_base_settings.go
      Note: Shows how Pinocchio strips profile-derived parse steps to recover a profile-free base
    - Path: pinocchio/pkg/cmds/profilebootstrap/profile_selection.go
      Note: Connects Pinocchio bootstrap to CreateGeppettoSections for hidden base reconstruction
    - Path: pinocchio/pkg/ui/profileswitch/backend.go
      Note: Turns resolved settings into a live engine and session builder during runtime switching
    - Path: pinocchio/pkg/ui/profileswitch/manager.go
      Note: Owns the preserved base and recomputes final settings for each runtime profile switch
ExternalSources: []
Summary: Intern-facing explanation of how Pinocchio and Geppetto reconstruct base inference settings, overlay engine profiles, and support runtime profile switching without losing the underlying non-profile baseline.
LastUpdated: 2026-03-27T09:31:51.335434713-04:00
WhatFor: Explain the slightly non-obvious settings lifecycle behind Pinocchio commands so a contributor can understand what “hidden base settings” are, why profiles are merged on top of them, and how runtime profile switching preserves the baseline.
WhenToUse: Use when adding shared settings such as proxy support, debugging profile precedence, or modifying the runtime profile-switch system in Pinocchio or Geppetto.
---


# Base settings, profile overlays, and runtime profile switching in Pinocchio

## Executive Summary

This document explains a pattern that is easy to misread when first entering the codebase: Pinocchio does not treat "current settings" as one flat object. Instead, it keeps a distinction between:

- a baseline of non-profile settings,
- a profile overlay resolved from the engine-profile registry,
- and the final merged settings that are actually used to build the engine.

There are two related ways the codebase obtains that baseline.

First, Geppetto bootstrap code can rebuild a hidden base `InferenceSettings` from shared sections plus config, environment, and defaults. This is used when the application needs a reliable full AI baseline even if a given CLI command does not visibly expose every AI flag. See [geppetto/pkg/cli/bootstrap/engine_settings.go:26-58](../../../../../../pkg/cli/bootstrap/engine_settings.go).

Second, Pinocchio command code can recover a profile-free base from already parsed command values by removing parse steps whose source is `"profiles"`, then decoding the remaining values back into a fresh `InferenceSettings`. This is used for runtime profile switching and for commands that have already parsed flags through the normal middleware chain. See [pinocchio/pkg/cmds/profile_base_settings.go:12-89](../../../../../../../pinocchio/pkg/cmds/profile_base_settings.go).

That distinction matters for feature work such as HTTP proxy support. If a setting belongs to the shared application baseline, it should live in a shared Geppetto section such as `ai-client`, so it naturally becomes part of the base settings and continues to survive profile changes.

## Problem Statement

New contributors often assume the system does this:

1. parse flags into one settings object,
2. maybe replace some fields from a profile,
3. run inference.

That is not quite what happens.

The actual system needs to solve three different problems at once:

1. build a baseline from app-owned config, env, defaults, and sometimes flags,
2. apply a Geppetto-owned engine profile on top of that baseline,
3. support runtime profile switching without permanently losing the original baseline.

Those requirements lead to a slightly more layered design:

- Geppetto owns the reusable AI settings schema and the logic that can reconstruct a baseline.
- engine profiles own only profile overlay data, not the whole application runtime.
- Pinocchio owns runtime control flow and interactive profile switching.

If an intern does not understand those boundaries, it is easy to make one of these mistakes:

- add a shared setting to the wrong section,
- accidentally store infrastructure in profiles,
- mutate final settings in place and lose the original base,
- or re-resolve profiles incorrectly during a runtime switch.

## Mental Model In One Paragraph

Think of the system as "base plus patch equals active runtime."

- The base is the durable app-side configuration: defaults, config files, environment variables, and sometimes command flags.
- The profile is a patch selected from a profile registry.
- The active runtime settings are the result of merging the profile patch on top of the base.

When the user switches profiles at runtime, the code does not treat the currently active settings as the new base. Instead, it goes back to the preserved base, resolves a different profile patch, and computes a new active runtime settings object.

That design prevents profile A from permanently contaminating profile B.

## Glossary

### `InferenceSettings`

The main Geppetto settings struct. It contains the major AI-related sub-objects:

- `API`
- `Chat`
- `Client`
- provider-specific settings such as `OpenAI`, `Claude`, `Gemini`, `Ollama`
- `Embeddings`
- `Inference`

See [geppetto/pkg/steps/ai/settings/settings-inference.go:59-77](../../../../../../pkg/steps/ai/settings/settings-inference.go).

### Base inference settings

An `InferenceSettings` value that contains non-profile baseline configuration. This is the object profile overlays should be merged onto.

### Hidden base inference settings

An internal base `InferenceSettings` reconstructed from shared Geppetto sections plus env/config/defaults, even when those sections are not directly exposed by the current command surface. See [geppetto/pkg/cli/bootstrap/engine_settings.go:26-58](../../../../../../pkg/cli/bootstrap/engine_settings.go).

### Final inference settings

The settings object that comes out of:

```text
final = merge(base, resolved_profile.inference_settings)
```

This is the object used to build the actual engine.

### Profile-derived parse step

A field value inside `values.Values` whose parse log contains a step with source `"profiles"`. Pinocchio strips these steps when reconstructing a profile-free base from already parsed values. See [pinocchio/pkg/cmds/profile_base_settings.go:40-67](../../../../../../../pinocchio/pkg/cmds/profile_base_settings.go).

## The Core Structures

### Schema diagram

```text
Glazed sections
  ai-chat
  ai-client
  openai-chat
  claude-chat
  gemini-chat
  embeddings
  ai-inference
  profile-settings
        |
        v
values.Values
  section -> field -> value + parse log
        |
        +--> DecodeSectionInto(ai-client)     -> ClientSettings
        +--> DecodeSectionInto(ai-chat)       -> ChatSettings
        +--> DecodeSectionInto(openai-chat)   -> openai.Settings
        +--> DecodeSectionInto(claude-chat)   -> claude.Settings
        +--> DecodeSectionInto(gemini-chat)   -> gemini.Settings
        +--> DecodeSectionInto(embeddings)    -> EmbeddingsConfig
        +--> DecodeSectionInto(ai-inference)  -> engine.InferenceConfig
        |
        v
InferenceSettings
  API
  Chat
  Client
  OpenAI
  Claude
  Gemini
  Ollama
  Embeddings
  Inference
        |
        +--> base settings
        +--> profile overlay settings
                 |
                 v
         MergeInferenceSettings(base, overlay)
                 |
                 v
          final runtime InferenceSettings
```

The key point is that `values.Values` is not just a map of values. It also stores parse provenance. That provenance is what allows Pinocchio to later say, "show me the same parsed settings, but remove the pieces that came from the profile middleware."

## Data Ownership

### Data ownership diagram

```text
Application-owned baseline
  Owner: Pinocchio app + Geppetto shared section system
  Sources:
    - defaults
    - config files
    - environment variables
    - command flags (when available on that command)
  Typical fields:
    - ai-client.*
    - ai-chat defaults
    - provider base URLs / API keys
    - embeddings defaults
    - ai-inference defaults

Profile-owned overlay
  Owner: Geppetto engine profile registry
  Sources:
    - selected engine profile
  Typical fields:
    - chat.engine
    - chat.api_type
    - model-specific request defaults
    - provider-specific inference settings
  Should generally NOT own:
    - app transport infrastructure
    - credentials
    - unrelated app runtime policy

Runtime-owned active state
  Owner: Pinocchio runtime / profile switch backend
  Sources:
    - merged result of baseline + resolved profile
    - current selected profile metadata
  Typical fields:
    - final InferenceSettings used to build the current engine
    - selected registry slug / profile slug / profile version
    - live session builder
```

This separation is why proxy support belongs in `ai-client`. A proxy is part of the app-owned baseline. It should survive profile changes and should not need to be duplicated across profiles.

## Where The Baseline Comes From

There are two different baseline-building paths in the codebase. They are related, but not identical.

### Path A: rebuild a hidden base from shared sections

This path lives in [geppetto/pkg/cli/bootstrap/engine_settings.go:26-58](../../../../../../pkg/cli/bootstrap/engine_settings.go).

`ResolveBaseInferenceSettings(...)` does this:

1. calls `cfg.BuildBaseSections()`,
2. builds a new Glazed schema from those sections,
3. creates a fresh `values.Values`,
4. resolves config files from the current bootstrap config,
5. executes only:
   - environment variables,
   - config files,
   - defaults,
6. decodes the result into a brand-new `InferenceSettings`.

Important detail: this function does not replay Cobra flags. It is intentionally constructing a clean internal baseline from shared sections and app-level config sources.

This is what I meant by "hidden" in the earlier note:

- it is reconstructed internally,
- it uses sections that may not all be visible on the current command,
- and it exists to give the app a complete AI baseline even when the command surface is narrower.

### Why Path A exists

Some commands do not expose the full shared Geppetto AI surface on their own CLI, but they still need an AI baseline. `web-chat` is the clearest example.

`web-chat` mounts only a small visible section set, but it still calls `profilebootstrap.ResolveBaseInferenceSettings(parsed)` and logs the result as "resolved hidden web-chat base inference settings" in [pinocchio/cmd/web-chat/main.go:173-180](../../../../../../../pinocchio/cmd/web-chat/main.go).

That means:

- the command does not need to expose every AI flag directly,
- but the application can still reconstruct the shared baseline behind the scenes.

### Path B: recover a base from already parsed values by stripping profiles

This path lives in [pinocchio/pkg/cmds/profile_base_settings.go:12-89](../../../../../../../pinocchio/pkg/cmds/profile_base_settings.go).

`baseSettingsFromParsedValuesWithBase(...)` does this:

1. clones the already parsed `values.Values`,
2. walks every field's parse log,
3. removes the last-applied entries whose source is `"profiles"`,
4. keeps the last non-profile parse step for each field,
5. decodes the remaining values into a fresh `InferenceSettings`.

This is a different kind of base:

- it starts from the actual command parse result,
- so it includes whatever Cobra flags and command-layer values were already applied,
- but it deliberately removes profile contributions.

### Why Path B exists

Interactive profile switching needs a profile-free baseline that already reflects the command as it was actually launched.

If the system only stored "current final settings", then switching from profile A to profile B would effectively become:

```text
final_after_B = merge(final_after_A, profile_B)
```

That would be wrong, because `final_after_A` already contains profile A's changes.

Instead, Pinocchio uses:

```text
base_without_profiles = strip_profile_steps(parsed_values)
final_after_B = merge(base_without_profiles, profile_B)
```

That preserves the original non-profile baseline and prevents profile cross-contamination.

## The Shared Section Layer

The reusable Geppetto AI sections are assembled in [geppetto/pkg/sections/sections.go:35-127](../../../../../../pkg/sections/sections.go). The returned list includes:

- `ai-chat`
- `ai-client`
- provider sections
- `embeddings`
- `ai-inference`
- `profile-settings`

This section list is important because it defines what counts as shared baseline-capable AI configuration. If a field is not represented in these shared sections, it cannot participate naturally in the reconstructed base.

For Pinocchio bootstrap, the app config passed by [pinocchio/pkg/cmds/profilebootstrap/profile_selection.go:18-29](../../../../../../../pinocchio/pkg/cmds/profilebootstrap/profile_selection.go) sets `BuildBaseSections` to `CreateGeppettoSections()`. That is why the hidden base includes `ai-client`.

## Startup Flow For Standard Commands

The normal command path has two parallel concepts:

- `stepSettings`, built from the full parsed command values,
- `baseSettings`, rebuilt or recovered so profiles can be applied cleanly.

You can see this in [pinocchio/pkg/cmds/cmd.go:225-265](../../../../../../../pinocchio/pkg/cmds/cmd.go).

### Sequence diagram: startup resolution

```text
User
  |
  | runs pinocchio command
  v
Cobra + Glazed middleware
  |
  | parse flags/args/env/config/defaults
  | apply profile middleware
  v
parsedValues
  |
  | Decode into stepSettings
  | [cmd.go:225-233]
  v
stepSettings (may already include profile effects)
  |
  | Recover profile-free base
  | baseSettingsFromParsedValuesWithBase(...)
  | [cmd.go:238-244]
  v
baseSettings
  |
  | Resolve selected profile and merge onto base
  | ResolveCLIEngineSettingsFromBase(...)
  | [cmd.go:250-264]
  v
ResolvedCLIEngineSettings
  |
  | BaseInferenceSettings = profile-free base
  | FinalInferenceSettings = base + resolved profile overlay
  v
engine factory
  |
  v
runtime engine
```

### What `stepSettings` is for

`stepSettings` is the "what the command currently sees" settings object. It is useful for ordinary command execution because it reflects the parsed command layers directly.

However, once profiles participate, `stepSettings` is not a safe baseline for future profile switching unless you first remove the profile-derived values.

### What `baseSettings` is for

`baseSettings` is the stable foundation for:

- resolving the startup profile correctly,
- reconstructing active settings,
- and switching profiles later.

## Bootstrap Flow For Narrower Commands

Some command surfaces need the baseline but do not mount the full visible AI settings surface. That is where the hidden base reconstruction path is useful.

### Sequence diagram: hidden bootstrap base

```text
Current app command
  |
  | provides AppBootstrapConfig
  | BuildBaseSections = CreateGeppettoSections()
  v
ResolveBaseInferenceSettings(...)
  |
  | Build fresh schema from shared sections
  | [engine_settings.go:31-35]
  |
  | Resolve config file list
  | [engine_settings.go:37-40]
  |
  | Execute env + files + defaults only
  | [engine_settings.go:41-51]
  v
hidden parsed values
  |
  | Decode into new InferenceSettings
  | [engine_settings.go:54-57]
  v
hidden base inference settings
```

This hidden base is especially useful in `web-chat`, which logs exactly that concept in [pinocchio/cmd/web-chat/main.go:173-180](../../../../../../../pinocchio/cmd/web-chat/main.go).

## How Profile Overlay Merging Works

Profile merging is handled by [geppetto/pkg/engineprofiles/inference_settings_merge.go:20-49](../../../../../../pkg/engineprofiles/inference_settings_merge.go).

At a high level:

1. clone/serialize base and overlay into maps,
2. recursively merge them,
3. convert the merged map back into `InferenceSettings`.

Important semantics:

- the overlay wins for conflicting scalar values,
- nested maps merge recursively,
- some normalization happens for fields such as client timeout in [geppetto/pkg/engineprofiles/inference_settings_merge.go:86-105](../../../../../../pkg/engineprofiles/inference_settings_merge.go).

Pseudocode:

```go
func MergeInferenceSettings(base, overlay *InferenceSettings) *InferenceSettings {
    if base == nil { return clone(overlay) }
    if overlay == nil { return clone(base) }

    baseMap := inferenceSettingsToMap(base)
    overlayMap := inferenceSettingsToMap(overlay)
    mergedMap := recursiveMerge(baseMap, overlayMap) // overlay wins on conflicts
    return inferenceSettingsFromMap(mergedMap)
}
```

That means profile overlays behave like patches. They are not full replacements of the whole settings tree.

## Runtime Profile Switching

The runtime switching code lives in [pinocchio/pkg/ui/profileswitch/manager.go:14-188](../../../../../../../pinocchio/pkg/ui/profileswitch/manager.go) and [pinocchio/pkg/ui/profileswitch/backend.go:19-236](../../../../../../../pinocchio/pkg/ui/profileswitch/backend.go).

### What the manager owns

`profileswitch.Manager` owns:

- a profile registry,
- the original base `InferenceSettings`,
- the currently resolved runtime selection metadata.

The critical field is `base *settings.InferenceSettings` in [pinocchio/pkg/ui/profileswitch/manager.go:14-19](../../../../../../../pinocchio/pkg/ui/profileswitch/manager.go).

That field is the whole reason switching can work cleanly.

### What happens during `Resolve(...)`

In [pinocchio/pkg/ui/profileswitch/manager.go:127-177](../../../../../../../pinocchio/pkg/ui/profileswitch/manager.go), the manager:

1. resolves a profile from the registry,
2. merges that profile's `InferenceSettings` onto `m.base`,
3. returns a new `Resolved` value containing:
   - registry slug,
   - profile slug,
   - merged `InferenceSettings`,
   - profile version metadata.

That method does not mutate `m.base`.

### What happens during `Switch(...)`

`Switch(...)` simply calls `Resolve(...)`, stores the resulting resolved state, and returns it. See [pinocchio/pkg/ui/profileswitch/manager.go:180-187](../../../../../../../pinocchio/pkg/ui/profileswitch/manager.go).

### What the backend does

The backend turns a resolved settings object into a live engine/session builder.

In [pinocchio/pkg/ui/profileswitch/backend.go:142-170](../../../../../../../pinocchio/pkg/ui/profileswitch/backend.go), `applyResolved(...)`:

1. creates a new engine from the resolved `InferenceSettings`,
2. creates a new `enginebuilder.Builder`,
3. stores the resolved metadata,
4. swaps the session builder.

The runtime session builder changes. The preserved base does not.

### Sequence diagram: runtime profile switch

```text
User selects new profile
  |
  v
profileswitch.Backend.SwitchProfile(profileSlug)
  |
  | guard: session must be idle
  | [backend.go:125-131]
  v
profileswitch.Manager.Switch(profileSlug)
  |
  | Resolve profile from registry
  | [manager.go:148-153]
  |
  | Merge resolved profile settings onto preserved m.base
  | [manager.go:162-165]
  v
Resolved{
  InferenceSettings = merge(base, profileOverlay)
}
  |
  v
Backend.applyResolved(...)
  |
  | factory.NewEngineFromSettings(res.InferenceSettings)
  | [backend.go:150-153]
  |
  | replace session builder
  | [backend.go:158-170]
  v
New active runtime profile
```

## Why The System Does Not Just Mutate One Settings Object

Because that would destroy the baseline.

Suppose the starting baseline is:

```yaml
client:
  timeout_second: 60
chat:
  api_type: openai
  engine: base-model
```

And profile A says:

```yaml
chat:
  engine: gpt-4.1
```

And profile B says:

```yaml
chat:
  engine: claude-3-7-sonnet
```

If the program simply mutated the active settings in place, then after applying profile A it would be unclear whether `chat.engine` still represented baseline or overlay data. Later, when switching to profile B, you could accidentally end up with "profile B merged onto a settings object already polluted by profile A."

Preserving a separate base avoids that ambiguity.

## The Two Base Mechanisms Side By Side

This is the single most important comparison in the document.

| Mechanism | Source material | Includes profile values? | Includes command-parse flags? | Main use |
|---|---|---|---|---|
| Hidden bootstrap base | shared sections + env + config + defaults | No | Not directly | construct app-wide AI baseline even when command surface is narrow |
| Parsed-values base after stripping profiles | already parsed `values.Values` minus `"profiles"` steps | No | Yes | runtime switching and commands that already parsed their real flag surface |

If you remember only one thing, remember this:

- hidden base is reconstructed from canonical shared sections,
- stripped base is recovered from actual parsed runtime values.

Both are valid. They serve different moments in the lifecycle.

## How To Reason About A New Setting

When adding a field, ask these questions in order.

1. Is this field part of the reusable application baseline?
   - If yes, it belongs in a shared Geppetto section and should appear in base settings.

2. Is this field model/profile behavior or app/operator infrastructure?
   - Profile behavior may belong in engine profile overlays.
   - App/operator infrastructure should usually stay in baseline settings.

3. Does the field need to survive runtime profile changes unchanged?
   - If yes, it almost certainly belongs in the base, not in the profile.

Applied to proxy support:

- proxy is transport infrastructure,
- it should survive profile changes,
- therefore it belongs in `ai-client`,
- and provider engine construction should consume it from the final merged settings whose base already carried it forward.

## Common Mistakes

### Mistake 1: confusing "currently active" with "baseline"

The currently active settings may already include a profile overlay. They are not automatically safe to reuse as the new base.

### Mistake 2: putting app infrastructure into profiles

If the value is something like proxy transport, credentials, or app middleware policy, it is usually the wrong thing to store in an engine profile.

### Mistake 3: assuming every command exposes the whole shared AI surface

Some commands use hidden base reconstruction specifically because their visible CLI is narrower than the shared AI schema.

### Mistake 4: forgetting parse provenance

`values.Values` carries parse logs. That provenance is part of the design, not an incidental implementation detail.

## Debugging Checklist

When debugging a settings-precedence problem, inspect these layers in order:

1. Which sections are actually mounted for this command?
   - [geppetto/pkg/sections/sections.go](../../../../../../pkg/sections/sections.go)
   - command-specific setup such as [pinocchio/pkg/cmds/loader.go](../../../../../../../pinocchio/pkg/cmds/loader.go)

2. Is the baseline coming from hidden bootstrap reconstruction or parsed-values stripping?
   - [geppetto/pkg/cli/bootstrap/engine_settings.go](../../../../../../pkg/cli/bootstrap/engine_settings.go)
   - [pinocchio/pkg/cmds/profile_base_settings.go](../../../../../../../pinocchio/pkg/cmds/profile_base_settings.go)

3. Which fields came from source `"profiles"`?
   - inspect parse logs in `values.Values` if you need exact provenance

4. What did the profile overlay actually contain?
   - registry resolution path in [geppetto/pkg/cli/bootstrap/engine_settings.go:112-139](../../../../../../pkg/cli/bootstrap/engine_settings.go)
   - runtime switch path in [pinocchio/pkg/ui/profileswitch/manager.go:148-165](../../../../../../../pinocchio/pkg/ui/profileswitch/manager.go)

5. Which settings object built the current engine?
   - startup path in [pinocchio/pkg/cmds/cmd.go:250-265](../../../../../../../pinocchio/pkg/cmds/cmd.go)
   - runtime switch path in [pinocchio/pkg/ui/profileswitch/backend.go:150-170](../../../../../../../pinocchio/pkg/ui/profileswitch/backend.go)

## Design Decisions

### Decision 1: preserve a separate baseline

Rationale:

- avoids profile cross-contamination,
- makes runtime profile switching deterministic,
- keeps operator/app configuration separate from profile patches.

### Decision 2: support both a hidden reconstructed base and a stripped parsed-values base

Rationale:

- some commands need a full internal AI baseline even when their visible CLI is narrow,
- other flows need a baseline that faithfully reflects already parsed command flags.

### Decision 3: treat profile data as overlays, not the whole runtime state

Rationale:

- profile registries should stay focused on engine/profile behavior,
- app runtime ownership remains in the application layer.

## Alternatives Considered

### Alternative A: keep only one mutable settings object

Rejected.

This would make runtime switching fragile because the active object would already contain previous profile effects.

### Alternative B: reconstruct the base from scratch for every runtime switch

Rejected as the primary runtime mechanism.

That would ignore command-specific parse state already present in `values.Values` and would make switching less faithful to how the command was launched.

### Alternative C: store all runtime/app settings inside engine profiles

Rejected.

That collapses the ownership boundary between app baseline and profile overlay.

## Implementation Plan

This document is explanatory rather than proposing a new subsystem, but if you need to modify this area safely, follow this order:

1. Decide whether your field belongs in the baseline or in the profile overlay.
2. If it belongs in the baseline, add it to the appropriate shared Geppetto section and `InferenceSettings` sub-struct.
3. Confirm that hidden bootstrap reconstruction picks it up.
4. Confirm that stripped parsed-values base recovery also preserves it when it is not profile-derived.
5. Confirm that profile overlays do not accidentally own or overwrite it unless that is intentional.
6. Add tests for both startup resolution and runtime switch behavior.

The existing bootstrap parity test in [geppetto/pkg/cli/bootstrap/bootstrap_test.go:208-273](../../../../../../pkg/cli/bootstrap/bootstrap_test.go) is a good starting point because it proves that resolving from an explicit base matches resolving through the direct path.

## Open Questions

1. Should the codebase make the names "bootstrap base" and "runtime stripped base" more explicit in comments or type names?
2. Should there be a small shared debug helper for visualizing the exact provenance of fields in `values.Values` during profile switching?
3. For future infrastructure fields such as proxy support, should profile validation eventually reject those fields inside engine profiles?

## References

- [geppetto/pkg/steps/ai/settings/settings-inference.go](../../../../../../pkg/steps/ai/settings/settings-inference.go)
- [geppetto/pkg/sections/sections.go](../../../../../../pkg/sections/sections.go)
- [geppetto/pkg/cli/bootstrap/engine_settings.go](../../../../../../pkg/cli/bootstrap/engine_settings.go)
- [geppetto/pkg/engineprofiles/inference_settings_merge.go](../../../../../../pkg/engineprofiles/inference_settings_merge.go)
- [geppetto/pkg/cli/bootstrap/bootstrap_test.go](../../../../../../pkg/cli/bootstrap/bootstrap_test.go)
- [pinocchio/pkg/cmds/cmd.go](../../../../../../../pinocchio/pkg/cmds/cmd.go)
- [pinocchio/pkg/cmds/profile_base_settings.go](../../../../../../../pinocchio/pkg/cmds/profile_base_settings.go)
- [pinocchio/pkg/cmds/profilebootstrap/profile_selection.go](../../../../../../../pinocchio/pkg/cmds/profilebootstrap/profile_selection.go)
- [pinocchio/pkg/cmds/loader.go](../../../../../../../pinocchio/pkg/cmds/loader.go)
- [pinocchio/pkg/ui/profileswitch/manager.go](../../../../../../../pinocchio/pkg/ui/profileswitch/manager.go)
- [pinocchio/pkg/ui/profileswitch/backend.go](../../../../../../../pinocchio/pkg/ui/profileswitch/backend.go)
- [pinocchio/cmd/web-chat/main.go](../../../../../../../pinocchio/cmd/web-chat/main.go)

## Problem Statement

<!-- Describe the problem this design addresses -->

## Proposed Solution

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
