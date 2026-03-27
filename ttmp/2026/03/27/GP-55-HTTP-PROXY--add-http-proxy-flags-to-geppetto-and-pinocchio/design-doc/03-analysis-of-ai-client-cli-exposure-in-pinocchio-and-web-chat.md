---
Title: Analysis of ai-client CLI exposure in Pinocchio and web-chat
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
    - Path: geppetto/pkg/doc/topics/01-profiles.md
      Note: Updated Geppetto concept doc that now explains baseline versus profile-overlay ownership
    - Path: geppetto/pkg/doc/tutorials/09-migrating-cli-commands-to-glazed-bootstrap-profile-resolution.md
      Note: Updated Geppetto migration tutorial that now explains hidden base versus final settings
    - Path: pinocchio/cmd/pinocchio/cmds/js.go
      Note: Shows an engine-backed command that already exposes ai-client through CreateGeppettoSections
    - Path: pinocchio/cmd/pinocchio/cmds/tokens/count.go
      Note: Shows another built-in command that already mounts full Geppetto sections
    - Path: pinocchio/cmd/pinocchio/main.go
      Note: Shows the current root-level profile-only flag exposure and the candidate insertion point for persistent ai-client flags
    - Path: pinocchio/cmd/web-chat/main.go
      Note: |-
        Defines the current web-chat command surface and the hidden-base runtime path
        web-chat command now mounts ai-client and merges parsed client values into its preserved base settings.
    - Path: pinocchio/cmd/web-chat/main_profile_registries_test.go
      Note: |-
        Confirms web-chat currently exposes only profile-related shared flags and not full AI flags
        Regression tests for web-chat ai-client CLI exposure and hidden-base plus parsed-value merge behavior.
    - Path: pinocchio/pkg/cmds/loader.go
      Note: Shows how repository-loaded commands inherit ai-client exposure through CreateGeppettoSections
    - Path: pinocchio/pkg/cmds/profile_base_settings.go
      Note: |-
        Provides the parsed-values base reconstruction helper that is the likely model for web-chat CLI ai-client support
        Command-level helper now delegates to the shared parsed-base overlay helper.
    - Path: pinocchio/pkg/cmds/profilebootstrap/engine_settings.go
      Note: Shows the current hidden-base helper used by web-chat
    - Path: pinocchio/pkg/cmds/profilebootstrap/parsed_base_settings.go
      Note: Shared parsed-base helper that overlays parsed non-profile values onto a hidden base.
    - Path: pinocchio/pkg/cmds/profilebootstrap/parsed_base_settings_test.go
      Note: Regression tests for parsed-base overlay semantics with profile-stripping behavior.
    - Path: pinocchio/pkg/doc/topics/pinocchio-profile-resolution-and-runtime-switching.md
      Note: New Pinocchio canonical topic for hidden base settings and runtime switching
    - Path: pinocchio/pkg/doc/topics/webchat-profile-registry.md
      Note: Updated web-chat doc that now explains the hidden-base caveat for ai-client CLI widening
ExternalSources: []
Summary: Analysis of where ai-client flags are already exposed in Pinocchio, where a broader CLI surface could be added, and why web-chat requires both a public section and a parsed-values-aware base path if explicit ai-client flags should affect runtime behavior.
LastUpdated: 2026-03-27T09:49:06.333721504-04:00
WhatFor: Help future implementation work decide the least disruptive place to expose ai-client settings such as proxy flags across standard Pinocchio commands and the narrower web-chat command surface.
WhenToUse: Use when implementing proxy-related ai-client fields, widening CLI surfaces, or deciding whether settings should be root-level, command-level, config-only, or web-chat-specific.
---



# Analysis of ai-client CLI exposure in Pinocchio and web-chat

## Executive Summary

The short answer is:

1. standard engine-backed Pinocchio command surfaces already have a natural `ai-client` exposure path,
2. a root-level `pinocchio` persistent `ai-client` surface is possible but high-risk because of duplicate-flag and ownership concerns,
3. `web-chat` is different: it does not currently expose `ai-client`, and adding the section alone would not make those CLI flags effective because its base inference settings are rebuilt from config, environment, and defaults only.

The recommended implementation strategy is:

- do not add root-level persistent `ai-client` flags to `pinocchio` first,
- rely on the existing `CreateGeppettoSections()` path for engine-backed commands such as `pinocchio js`, repository-loaded commands, and token-count commands,
- if `web-chat` must expose proxy flags explicitly, add a public `ai-client` section and a parsed-values-aware base-resolution path so those flags actually reach the preserved base inference settings used by request resolution.

## Problem Statement

The user goal is not just "make proxy configuration possible somewhere." The user goal is:

- let cross-profile settings such as proxy transport live at the CLI/config layer,
- keep them outside engine profiles,
- and ensure both standard Pinocchio commands and `web-chat` can actually use them.

Those requirements force two related questions:

1. Where should the flags be visible?
2. Which runtime settings path actually consumes them?

Those are not the same question.

In `pinocchio`, many engine-backed commands already expose the full Geppetto section set. In `web-chat`, the public CLI is intentionally narrower and the runtime baseline comes from a hidden-base reconstruction path. That means the same `ai-client` feature interacts with two different command architectures.

## Current State

### Standard Pinocchio engine-backed commands

Several modern `pinocchio` command surfaces already mount the full Geppetto sections. That means `ai-client` is already the natural place for cross-profile CLI settings there.

Concrete evidence:

- `pinocchio js` mounts `CreateGeppettoSections()` in [pinocchio/cmd/pinocchio/cmds/js.go:62-103](../../../../../../../pinocchio/cmd/pinocchio/cmds/js.go).
- repository-loaded commands build their sections from `CreateGeppettoSections(...)` in [pinocchio/pkg/cmds/loader.go:65-85](../../../../../../../pinocchio/pkg/cmds/loader.go).
- `tokens count` mounts `CreateGeppettoSections()` in [pinocchio/cmd/pinocchio/cmds/tokens/count.go:31-69](../../../../../../../pinocchio/cmd/pinocchio/cmds/tokens/count.go).
- repository-loaded commands run through the shared Geppetto middleware path in [pinocchio/cmd/pinocchio/main.go:245-252](../../../../../../../pinocchio/cmd/pinocchio/main.go).

So for those commands, once `ai-client` gains proxy flags in Geppetto, the public CLI exposure story is already largely solved.

### Root `pinocchio` command

The root command currently adds only `profile-settings` directly to the root Cobra command in [pinocchio/cmd/pinocchio/main.go:161-164](../../../../../../../pinocchio/cmd/pinocchio/main.go).

That means there is no current root-level persistent `ai-client` surface. Instead, `ai-client` exposure comes from individual subcommands that mount full Geppetto sections.

### `web-chat`

`web-chat` intentionally exposes only:

- `profile-settings`
- `redis`

in [pinocchio/cmd/web-chat/main.go:75-103](../../../../../../../pinocchio/cmd/web-chat/main.go).

Its tests explicitly assert that AI flags like `ai-engine` are not exposed on that CLI in [pinocchio/cmd/web-chat/main_profile_registries_test.go:95-123](../../../../../../../pinocchio/cmd/web-chat/main_profile_registries_test.go).

At runtime, `web-chat` resolves:

- profile selection from parsed values in [pinocchio/cmd/web-chat/main.go:117-126](../../../../../../../pinocchio/cmd/web-chat/main.go),
- hidden base inference settings through `profilebootstrap.ResolveBaseInferenceSettings(parsed)` in [pinocchio/cmd/web-chat/main.go:173-180](../../../../../../../pinocchio/cmd/web-chat/main.go).

That base-resolution helper delegates to Geppetto bootstrap in [pinocchio/pkg/cmds/profilebootstrap/engine_settings.go:15-17](../../../../../../../pinocchio/pkg/cmds/profilebootstrap/engine_settings.go), which rebuilds the base from env, config files, and defaults only in [geppetto/pkg/cli/bootstrap/engine_settings.go:31-57](../../../../../../pkg/cli/bootstrap/engine_settings.go).

This is the most important constraint in the whole analysis.

## Exposure Matrix

| Surface | Current public `ai-client` exposure | Runtime can consume `ai-client` config/env | Runtime can consume explicit `ai-client` CLI flags today | Notes |
|---|---|---|---|---|
| `pinocchio js` | Yes, via full Geppetto sections | Yes | Yes, once fields exist in `ai-client` | Best existing model |
| repository-loaded prompt commands | Yes, via full Geppetto sections | Yes | Yes, once fields exist in `ai-client` | Inherits standard Geppetto path |
| `tokens count` | Yes, via full Geppetto sections | Yes | Yes, once fields exist in `ai-client` | API counting path still needs transport wiring separately |
| root `pinocchio` command itself | No persistent `ai-client` flags | N/A | N/A | Only `profile-settings` is mounted at root |
| `web-chat` | No | Yes, through hidden base config/env/defaults | No | Mounting flags alone is insufficient |

## Proposed Solution

### Recommendation for standard `pinocchio`

Do not start by adding root-level persistent `ai-client` flags to the root command.

Instead, treat the existing engine-backed subcommand surfaces as the main public CLI exposure for `ai-client`:

- `pinocchio js`
- repository-loaded commands
- built-in commands that already mount full Geppetto sections

Why this is the safest first step:

- the section ownership is already correct,
- these commands already decode and carry `ai-client`,
- no new root-level flag collision policy is required,
- and the implementation focus stays on the real missing piece: transport wiring in provider engines.

### Recommendation for root-level `pinocchio`

If a future product decision demands "global" persistent `--proxy-url` or other `ai-client` flags on the root command itself, the most obvious insertion point is [pinocchio/cmd/pinocchio/main.go:161-164](../../../../../../../pinocchio/cmd/pinocchio/main.go), immediately next to the existing direct root mount of `profile-settings`.

But this should be treated as a second-phase design, not the first implementation.

Risks:

- many engine-backed subcommands already mount `ai-client` through `CreateGeppettoSections()`,
- adding the same flags persistently at the root may create duplicate-flag conflicts or confusing help surfaces,
- non-engine commands such as `clip` do not benefit from those flags,
- the root command would need a clear policy for which subcommands inherit those flags and which do not.

Recommendation:

- do not add root-level persistent `ai-client` flags in the first implementation,
- revisit only if product ergonomics require "set once at root and use everywhere."

### Recommendation for `web-chat`

If the goal is only to let `web-chat` consume `ai-client` settings from config and environment, no public CLI changes are needed. The hidden base already gives that command a config/env/defaults path.

If the goal is to let `web-chat` accept explicit CLI flags such as `--proxy-url`, then two changes are required.

#### Change 1: mount an `ai-client` section on the command

The public CLI insertion point is [pinocchio/cmd/web-chat/main.go:75-103](../../../../../../../pinocchio/cmd/web-chat/main.go), where the command description is assembled.

Options:

- construct `settings.NewClientValueSection()` directly and add it next to `profileSettingsSection` and `redisLayer`,
- or create a small shared helper that returns just the `ai-client` section for commands that intentionally want cross-profile client flags without the rest of the full Geppetto AI surface.

I recommend the second shape if the feature is implemented, because `web-chat` clearly does not want the entire full-flags surface.

#### Change 2: preserve parsed `ai-client` CLI values when building the base

This is the part that is easy to miss.

Today `web-chat` gets its preserved baseline from `ResolveBaseInferenceSettings(parsed)`, and that helper rebuilds from:

- env
- config files
- defaults

It does not replay parsed Cobra values into the base. So if `web-chat` simply mounted `ai-client` flags and nothing else changed, those flag values would exist in the parsed command values but would not reach the hidden base used by request resolution.

That means the runtime would still ignore them.

The implementation therefore needs a parsed-values-aware base path for `web-chat`.

Recommended shape:

1. keep `ResolveBaseInferenceSettings(parsed)` as the hidden config/env/defaults baseline,
2. then overlay parsed non-profile `ai-client` values onto that baseline before constructing the runtime composer and request resolver.

The cleanest shared abstraction would likely be to generalize the existing stripped-base helper in [pinocchio/pkg/cmds/profile_base_settings.go:21-89](../../../../../../../pinocchio/pkg/cmds/profile_base_settings.go) so it can be reused outside the command runner path.

Conceptually:

```go
hiddenBase, configFiles, err := profilebootstrap.ResolveBaseInferenceSettings(parsed)

// new shared helper: preserve parsed CLI/env/config/default values except profile-derived ones,
// starting from the already reconstructed hidden base
baseWithCLI, err := profilebootstrap.ResolveParsedBaseInferenceSettingsWithBase(parsed, hiddenBase)
```

Then `web-chat` would pass `baseWithCLI` into:

- `newProfileRuntimeComposer(...)`
- `newProfileRequestResolver(...)`

instead of the current `hiddenBase`.

## Design Decisions

### Decision 1: treat standard `pinocchio` exposure and `web-chat` exposure as different problems

Rationale:

- standard engine-backed commands already mount full Geppetto sections,
- `web-chat` intentionally does not.

### Decision 2: do not recommend root-level persistent `ai-client` flags first

Rationale:

- higher duplication risk,
- weaker command-surface hygiene,
- no need to solve it before the provider transport layer itself is fixed.

### Decision 3: require a parsed-values-aware base path for `web-chat` if public `ai-client` flags are added

Rationale:

- without it, the flags would be visible but ineffective,
- which is worse than leaving them absent.

## Alternatives Considered

### Alternative A: add `ai-client` only at the root `pinocchio` command

Rejected for the first cut.

This would create a broad persistent surface before the codebase has a clean policy for root-vs-subcommand flag ownership.

### Alternative B: add `ai-client` to `web-chat` and stop there

Rejected.

That would not help standard Pinocchio commands, which already have a better exposure path, and it would still be incomplete unless the base-resolution path changes too.

### Alternative C: keep `web-chat` config/env-only forever

Viable as a product choice, but not the best match if the explicit user requirement is "I want to pass proxy flags to Pinocchio and have it be used."

If `web-chat` is expected to accept those flags explicitly, it needs the two-part change described above.

## Implementation Plan

### Phase 1: rely on existing standard Pinocchio command surfaces

1. Add proxy-related fields to Geppetto `ai-client`.
2. Let `pinocchio js`, repository commands, and other full-section verbs inherit those flags automatically.
3. Update docs to describe `ai-client` as the cross-profile CLI/config surface.

### Phase 2: decide whether `web-chat` should stay config/env-only or expose `ai-client` on its CLI

If no:

- document that `web-chat` consumes `ai-client` through config/env/defaults only.

If yes:

1. add a public `ai-client` section to `web-chat`,
2. add a parsed-values-aware base helper,
3. pass the resulting base into web-chat runtime composition and request resolution,
4. add tests proving the CLI flags actually affect the resolved base.

### Phase 3: only then reconsider root-level persistent `pinocchio` flags

Do this only if operator ergonomics still require a root-level "set once, apply everywhere" model.

## Open Questions

1. Should `web-chat` remain intentionally profile-only on its visible CLI and rely on config/env for `ai-client`?
2. If `web-chat` grows explicit `ai-client` flags, should those be limited to `ai-client` only, or should it reuse a broader Geppetto section set?
3. If root-level persistent `pinocchio` flags are later added, should engine-backed subcommands stop mounting the same flags locally to avoid duplication?

## References

- `geppetto/pkg/cli/bootstrap/engine_settings.go`
- `pinocchio/cmd/pinocchio/main.go`
- `pinocchio/cmd/pinocchio/cmds/js.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/count.go`
- `pinocchio/pkg/cmds/loader.go`
- `pinocchio/pkg/cmds/profile_base_settings.go`
- `pinocchio/pkg/cmds/profilebootstrap/engine_settings.go`
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/cmd/web-chat/main_profile_registries_test.go`
