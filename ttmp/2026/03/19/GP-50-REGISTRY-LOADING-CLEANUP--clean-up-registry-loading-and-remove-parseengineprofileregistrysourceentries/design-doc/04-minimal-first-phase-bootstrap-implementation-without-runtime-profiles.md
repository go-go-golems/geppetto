---
Title: Minimal first phase bootstrap implementation without runtime profiles
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
    - Path: ../../../../../../../pinocchio/pkg/cmds/cmd.go
      Note: |-
        Loaded-command path that must preserve command-local defaults
        Loaded-command path targeted by the first-phase helper refactor
    - Path: ../../../../../../../pinocchio/pkg/cmds/helpers/profile_runtime.go
      Note: |-
        Thin command path that already expresses most of the desired helper flow
        Thin command path targeted by the first-phase helper refactor
    - Path: ../../../../../../../pinocchio/pkg/cmds/loader.go
      Note: |-
        Loader path that currently re-parses YAML to recover default inference settings
        Loader defaults issue documented in the first-phase plan
    - Path: pkg/inference/engine/factory/helpers.go
      Note: |-
        Current plain parsed-values engine creation helper
        Current engine creation helper contrasted with profile-aware flow
ExternalSources: []
Summary: Detailed first-phase implementation shape for unifying config/profile/bootstrap behavior without introducing runtime profiles or runtime composition abstractions.
LastUpdated: 2026-03-19T10:30:17.552274315-04:00
WhatFor: Provide a concrete implementation guide for the next refactor phase after the initial documentation work is complete.
WhenToUse: Use when beginning code changes that consolidate baseline config loading, engine profile overlay, and engine creation.
---


# Minimal first phase bootstrap implementation without runtime profiles

## Executive Summary

The first implementation phase should converge the current bootstrap paths on a small shared contract without trying to solve chat-time runtime profiles. The minimum useful abstraction is:

- resolve baseline config-backed `InferenceSettings`
- resolve selected engine profile inputs
- optionally overlay an engine profile on the baseline settings
- create an engine from the final settings

That is enough to simplify most command paths and remove duplicated bootstrap logic. It is also small enough to keep loaded-command default preservation intact.

## Problem Statement

Today there are two overlapping but not identical bootstrap paths:

- `PinocchioCommand.RunIntoWriter(...)` reconstructs profile selection, builds a baseline snapshot, and applies profile overlays locally
- `ResolveFinalInferenceSettings(...)` in `profile_runtime.go` resolves baseline config and optional profile overlay for thin/bootstrap commands

At the same time, `loader.go` reconstructs inference defaults by re-parsing command YAML, and some examples still use `factory.NewEngineFromParsedValues(...)` directly. The result is a system where each entrypoint style has slightly different behavior even though they all want the same end product: a ready engine plus enough metadata to explain how it was chosen.

## Proposed Solution

Implement a first-phase shared helper layer with the following responsibilities.

### 1. Resolve profile selection

Input:
- parsed values from the shared `profile-settings` section
- config file discovery result

Output:
- selected profile slug
- normalized registry sources
- provenance notes if needed later for debugging/tests

Suggested shape:

```go
type ResolvedProfileSelection struct {
    Profile string
    ProfileRegistries []string
    ConfigFiles []string
}
```

### 2. Resolve final inference settings

Input:
- baseline parsed values or baseline `InferenceSettings`
- resolved profile selection

Output:
- base `InferenceSettings`
- final merged `InferenceSettings`
- optional resolved engine profile handle
- cleanup function for registry chains

Suggested shape:

```go
type ResolvedCLIEngineSettings struct {
    Base *settings.InferenceSettings
    Final *settings.InferenceSettings
    ResolvedEngineProfile *engineprofiles.ResolvedEngineProfile
    ConfigFiles []string
    Close func()
}
```

### 3. Create engine quickly

Input:
- resolved final settings

Output:
- ready engine

The important change is not that engine creation becomes complicated. It is that profile-aware paths should stop pretending `factory.NewEngineFromParsedValues(...)` alone solves the registry overlay problem.

### 4. Support two caller styles

The helper layer must support:

- parsed-values callers
  loaded/full commands where command-local defaults are already embedded in parsed values
- bootstrap callers
  thin/manual commands that first need to build hidden parsed values from defaults, config, and env

The imported guide is right that both styles should exist. The cleanup target is shared helper contracts, not forcing every command through one exact parser entrypoint.

## Design Decisions

1. Preserve loaded-command defaults.
Reason:
`loader.go` injects inference defaults derived from the YAML command definition. Any helper that rebuilds baseline state from only config/env would lose those defaults for loaded commands.

2. Use `profile_runtime.go` as the conceptual starting point, not the exact final API.
Reason:
It already models baseline config plus optional profile overlay. It just needs to be generalized and aligned with the loaded-command path.

3. Defer runtime-profile state.
Reason:
Chat-time switching and runtime composition introduce extra state and UI concerns. They should build on top of the simplified baseline/profile/bootstrap layer later.

4. Make loader defaults explicit later in the phase.
Reason:
`loader.go` currently re-parses YAML to recreate defaults. That is a smell, but it can be isolated after the shared helper contract is clear.

## Alternatives Considered

Replace everything with `ResolveFinalInferenceSettings(...)` immediately.
Rejected because loaded commands need to preserve per-command defaults that are already present in parsed values.

Replace everything with `factory.NewEngineFromParsedValues(...)`.
Rejected because that helper does not resolve engine profile registries or profile selection overlays by itself.

Design the final runtime-profile system first and then back into CLI bootstrap.
Rejected because it expands the problem and delays the simpler shared-helper cleanup.

## Implementation Plan

1. Add a shared profile-selection resolver that all paths can call.
2. Add a shared final-settings resolver that can start either from parsed values or from a baseline settings object.
3. Add a tiny engine-creation helper for resolved final settings.
4. Refactor `cmd.go` to call the shared resolvers.
5. Refactor `profile_runtime.go` to become a consumer of the shared resolvers rather than a parallel implementation.
6. Refactor `loader.go` to make command-derived defaults explicit rather than rediscovering them by reparsing YAML.
7. Add tests proving:
   - loaded-command defaults are preserved
   - baseline-only operation works without registries
   - explicit profile overlays take precedence when registries exist
   - thin/bootstrap commands and loaded commands agree on final results

## Sketch

```text
loaded command
  parsed values already contain command defaults
    -> resolve profile selection
    -> compute base settings from parsed values
    -> optional profile overlay
    -> engine

thin/bootstrap command
  hidden parse from defaults + config + env + explicit flags
    -> resolve profile selection
    -> compute base settings
    -> optional profile overlay
    -> engine
```

## Open Questions

Whether the shared helper should return both parsed-values-derived base settings and final settings for all callers, or only for loaded commands that need runtime switching context.

Whether `web-chat` should be migrated directly to the new helpers in the same phase or after `cmd.go` and `profile_runtime.go` are unified.

## References

- [Adopt imported CLI profile guide and defer runtime profiles](./02-adopt-imported-cli-profile-guide-and-defer-runtime-profiles.md)
- [Geppetto backed CLI entrypoint inventory and bootstrap classification](../analysis/03-geppetto-backed-cli-entrypoint-inventory-and-bootstrap-classification.md)
- [Baseline config and engine profile registry semantics](./03-baseline-config-and-engine-profile-registry-semantics.md)
- [Pinocchio CLI Geppetto config and profile bootstrap guide](./01-pinocchio-cli-geppetto-config-and-profile-bootstrap-guide.md)
