---
Title: Baseline config and engine profile registry semantics
Ticket: GP-50-REGISTRY-LOADING-CLEANUP
Status: active
Topics:
    - profiles
    - glazed
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../glazed/pkg/config/resolve.go
      Note: |-
        Shared baseline app config path discovery
        Baseline config discovery rules documented here
    - Path: ../../../../../../../pinocchio/cmd/web-chat/main.go
      Note: |-
        Local duplicate semantics and discovery rules
        Duplicate local discovery rules documented here
    - Path: ../../../../../../../pinocchio/pkg/cmds/helpers/profile_runtime.go
      Note: |-
        Current baseline config and registry overlay helper
        Current baseline-only and overlay behavior documented here
    - Path: pkg/sections/profile_sections.go
      Note: |-
        Current default profile-registry fallback and middleware bootstrap
        Current registry fallback behavior documented here
ExternalSources:
    - local:geppetto_cli_profile_guide.md
Summary: Defines the intended separation between baseline app config and engine profile registries, including discovery rules and precedence guidance for the next implementation phase.
LastUpdated: 2026-03-19T10:30:17.546190135-04:00
WhatFor: Prevent schema drift by making the baseline config file and the engine profile registry file explicit, separate concepts.
WhenToUse: Use when implementing config/profile discovery, reviewing precedence behavior, or deciding whether a setting belongs in config or in an engine profile registry.
---


# Baseline config and engine profile registry semantics

## Executive Summary

The next phase should formalize two separate file roles:

- `config.yaml` is the baseline application and inference configuration file
- `profiles.yaml` is the engine profile registry file

They should not be overloaded to mean the same thing. The imported guide is correct that the current system becomes confusing mainly when these concepts blur together. The cleanup should simplify bootstrap by making the boundary explicit in code, tests, and docs.

## Problem Statement

Current code already hints at two distinct concepts, but the distinction is inconsistent in practice:

- Glazed's shared config resolution looks for `config.yaml`.
- Geppetto's profile section has a default fallback for `~/.config/pinocchio/profiles.yaml`.
- `web-chat` has its own local `config.yaml` lookup and local profile-registry lookup helpers.
- some older reasoning and docs drifted toward treating `profiles.yaml` as a possible baseline inference source.

That creates two risks:

1. schema confusion
   `config.yaml` uses section-based Glazed configuration, while `profiles.yaml` uses engine profile registry schema
2. behavior confusion
   commands do not agree on whether the absence of a registry file is an error or whether baseline config alone is sufficient

## Proposed Solution

Adopt the following semantics as the ticket's design target.

### File roles

`config.yaml`
- purpose:
  baseline application/CLI configuration
- schema:
  Glazed section-based config
- examples:
  `ai-chat`, `ai-client`, `ai-inference`, provider sections, command settings
- ownership:
  app/CLI baseline

`profiles.yaml`
- purpose:
  engine profile registry overlay source
- schema:
  Geppetto engine profile registry schema
- examples:
  named engine profiles such as `default`, `fast`, or `claude-sonnet`
- ownership:
  profile registry overlay

### Resolution model

```text
defaults
  <- config.yaml
  <- environment
  <- explicit CLI flags
  = baseline parsed values / base InferenceSettings

base InferenceSettings
  <- optional selected profile from profiles.yaml or other registry sources
  = final InferenceSettings
```

### Discovery rules

For baseline config:

- use `glazed/pkg/config.ResolveAppConfigPath("pinocchio", ...)`
- default search order remains:
  - `$XDG_CONFIG_HOME/pinocchio/config.yaml`
  - `$HOME/.pinocchio/config.yaml`
  - `/etc/pinocchio/config.yaml`
- explicit `--config` should be appended as a higher-precedence file after the default discovered file when both exist

For profile registries:

- explicit `profile-settings.profile-registries` remains the strongest source
- environment and config should be able to populate that section through the shared profile-settings section
- a legacy fallback to `$XDG_CONFIG_HOME/pinocchio/profiles.yaml` may remain during this cleanup if it is clearly documented as a registry fallback, not as baseline config discovery

## Design Decisions

1. Baseline config and engine profiles stay separate.
Reason:
They represent different schemas, different responsibilities, and different precedence boundaries.

2. Baseline-only operation must remain valid for general CLI commands.
Reason:
`pinocchio/pkg/cmds/helpers/profile_runtime.go` already supports this, and it is the simpler operator model when a user only wants one configured engine without maintaining registries.

3. Registry fallback, if retained, must be described as legacy convenience.
Reason:
The fallback exists in code today, but it should not obscure the main contract that registries are optional overlays rather than the primary config file.

4. Registry-required commands may still opt into stricter semantics.
Reason:
Tools like profile switchers are specifically about registries and should be allowed to fail early when registries are absent.

## Alternatives Considered

Treat `profiles.yaml` as the baseline inference settings file.
Rejected because the schema is different and because the imported guide correctly separates engine overlay from baseline config.

Require profile registries for all commands.
Rejected because that would make ordinary single-engine CLI usage more cumbersome and would conflict with the more permissive helper path that already exists.

Remove all profile-registry fallback behavior immediately.
Deferred rather than rejected. It may be the right long-term cleanup, but the next phase can keep a clearly documented fallback while the helper APIs are consolidated.

## Implementation Plan

1. Standardize and document the current discovery rules in shared helpers.
2. Remove local duplicate discovery implementations where shared helpers can be used instead.
3. Make tests assert the intended baseline-only and optional-overlay semantics.
4. Keep runtime-profile work out of this phase so the file-role boundary stays clear.

## Open Questions

Should the fallback registry lookup include both XDG and `~/.pinocchio/profiles.yaml`, or only the current XDG path?

Should the explicit `--config` path continue to merge on top of a discovered default config, or should it replace it entirely for some command families?

## References

- [Imported Source - Geppetto CLI Profile Guide](../sources/local/geppetto_cli_profile_guide.md)
- [resolve.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/glazed/pkg/config/resolve.go)
- [profile_sections.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_sections.go)
- [profile_runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go)
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/main.go)
