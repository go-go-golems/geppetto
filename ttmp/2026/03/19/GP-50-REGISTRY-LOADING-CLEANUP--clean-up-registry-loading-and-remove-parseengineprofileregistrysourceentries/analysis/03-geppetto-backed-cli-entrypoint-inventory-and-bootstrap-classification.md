---
Title: Geppetto backed CLI entrypoint inventory and bootstrap classification
Ticket: GP-50-REGISTRY-LOADING-CLEANUP
Status: active
Topics:
    - profiles
    - glazed
    - cleanup
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/agents/simple-chat-agent/main.go
      Note: Thin command using ResolveFinalInferenceSettings
    - Path: ../../../../../../../pinocchio/cmd/pinocchio/main.go
      Note: |-
        Repository-loaded YAML command entrypoint
        Repository-loaded entrypoint in the inventory
    - Path: ../../../../../../../pinocchio/cmd/web-chat/main.go
      Note: |-
        Standalone app with local config/profile bootstrap helpers
        Standalone app with local bootstrap duplication
    - Path: ../../../../../../../pinocchio/pkg/cmds/cmd.go
      Note: |-
        Loaded-command runtime implementation
        Loaded-command runtime classified in the inventory
    - Path: ../../../../../../../pinocchio/pkg/cmds/helpers/profile_runtime.go
      Note: |-
        Thin/bootstrap helper implementation
        Thin bootstrap helper classified in the inventory
    - Path: cmd/examples/runner-glazed-registry-flags/main.go
      Note: |-
        Shared-section Glazed example representative
        Representative Glazed example for the inventory
ExternalSources: []
Summary: Inventory of current Geppetto-backed CLI entrypoints and their current bootstrap style so the cleanup can target all relevant paths instead of only loaded Pinocchio commands.
LastUpdated: 2026-03-19T10:30:17.538519665-04:00
WhatFor: Classify the current command families and identify which ones need to converge on the shared config/profile/engine bootstrap helpers.
WhenToUse: Use when deciding refactor scope or verifying that a new helper migration covers the intended command surfaces.
---


# Geppetto backed CLI entrypoint inventory and bootstrap classification

## Summary

The current command surfaces split into three real bootstrap styles:

- loaded Pinocchio commands that come from repository YAML and execute through `PinocchioCommand.RunIntoWriter(...)`
- Glazed/Cobra commands that already use sections and parsed values directly
- lightweight/manual entrypoints that bypass the full loaded-command pipeline and bootstrap engines with local helpers

The cleanup should not focus only on `pinocchio/pkg/cmds/cmd.go`. The broader problem is that these styles currently overlap in capability but use different helper stacks, different config discovery rules, and different fallback semantics when no registry is present.

## Classification Rules

For this inventory, commands are classified as:

- `loaded command`
  repository-loaded YAML command path using `pinocchio/pkg/cmds/loader.go`, `cobra.go`, and `cmd.go`
- `Glazed/Cobra command`
  command already expressed primarily as a Glazed/Cobra command with sections and parsed values
- `lightweight/manual bootstrap`
  command or app that constructs its own bootstrap flow, often with local helpers or direct engine creation

## Pinocchio Inventory

### Loaded command family

These commands are discovered from repositories and routed through the loaded-command runtime path:

- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/main.go)
  Uses `BuildCobraCommandWithGeppettoMiddlewares(...)` and then executes `PinocchioCommand.RunIntoWriter(...)`.
- [cobra.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cobra.go)
  Builds the Cobra command around the loaded command description.
- [loader.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/loader.go)
  Builds schema sections and infers defaults from YAML command content.
- [cmd.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd.go)
  Re-resolves profile inputs and optionally overlays a runtime-resolved engine profile before engine creation.

Why this matters:
This path preserves command-local defaults from YAML-loaded command descriptions. Any replacement helper must preserve that property, or loaded commands will silently lose per-command inference defaults.

### Glazed/Cobra command family

These commands already use Cobra and shared sections, but do not go through the loaded-command runtime:

- [js.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/cmds/js.go)
  Inherits the shared profile section and manually maps inherited flags into parsed values.
- [helpers.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/cmds/tokens/helpers.go)
  Reuses Geppetto command-building helpers for token-related commands.
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/switch-profiles-tui/main.go)
  Cobra command that uses shared profile flags but intentionally requires registries to exist.

Why this matters:
These commands are close to the intended end state, but some still do raw `cmd.Flags().GetStringSlice(...)` work and some keep stricter semantics than the rest of the CLI.

### Lightweight/manual bootstrap family

These commands do not rely on the loaded-command runtime path and instead bootstrap settings or engines locally:

- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/agents/simple-chat-agent/main.go)
  Calls `ResolveFinalInferenceSettings(...)`.
- [profile_runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go)
  Resolves config files, profile settings, and final inference settings outside the loaded-command path.
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/main.go)
  Has its own local variants of config-file discovery, profile-registry normalization, and base inference resolution.
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/examples/simple-chat/main.go)
  Uses `ParseGeppettoLayers(...)`, a transitional helper that rebuilds part of the stack manually.
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/examples/simple-redis-streaming-inference/main.go)
  Uses `factory.NewEngineFromParsedValues(...)`, which is not profile-registry-aware.

Why this matters:
This family proves the problem is broader than loaded commands. There are already multiple local bootstrap implementations in Pinocchio.

## Geppetto Inventory

### Glazed/Cobra examples already close to the shared-section model

- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-glazed-registry-flags/main.go)
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-glazed-full-flags/main.go)
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-registry/main.go)

These are useful as migration examples because they already mount shared sections and deal in parsed values rather than raw string normalization.

### Manual engine bootstrap examples

- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/inference/main.go)
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/streaming-inference/main.go)
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/advanced/claude-tools/main.go)
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/advanced/openai-tools/main.go)
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/advanced/middleware-inference/main.go)
- [main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/advanced/generic-tool-calling/main.go)

These still demonstrate or rely on `factory.NewEngineFromParsedValues(...)`. That is valid for plain parsed-values creation but does not solve the registry-aware profile overlay problem by itself.

## Current Bootstrap Surface Map

```text
Pinocchio loaded commands
  -> loader.go
  -> BuildCobraCommandWithGeppettoMiddlewares(...)
  -> cmd.go
  -> engine/profile overlay logic duplicated locally

Thin Pinocchio apps
  -> profile_runtime.go
  -> ResolveFinalInferenceSettings(...)
  -> engine

Standalone Pinocchio app variants
  -> web-chat local helpers
  -> partial duplication of config/profile discovery

Geppetto examples
  -> direct parsed values
  -> factory.NewEngineFromParsedValues(...)
```

## Findings

1. There is no single bootstrap contract that all Geppetto-backed CLIs already use.
2. The loaded-command path and the thin/bootstrap path both need to survive, but they should share helper contracts.
3. `web-chat` is a concrete example of why purely fixing `cmd.go` is insufficient.
4. Geppetto examples are useful migration targets because they reveal where profile-aware engine construction should replace plain `factory.NewEngineFromParsedValues(...)`.

## Recommended Scope For The Next Refactor

- First-class refactor targets:
  - `pinocchio/pkg/cmds/cmd.go`
  - `pinocchio/pkg/cmds/helpers/profile_runtime.go`
  - `pinocchio/pkg/cmds/loader.go`
- Early adopters for parity verification:
  - `pinocchio/cmd/agents/simple-chat-agent/main.go`
  - `pinocchio/cmd/web-chat/main.go`
  - `geppetto/cmd/examples/runner-glazed-registry-flags/main.go`
- Deferred or illustrative-only paths:
  - older examples using `ParseGeppettoLayers(...)`
  - examples that intentionally demonstrate raw parsed-values engine creation without profile overlays
