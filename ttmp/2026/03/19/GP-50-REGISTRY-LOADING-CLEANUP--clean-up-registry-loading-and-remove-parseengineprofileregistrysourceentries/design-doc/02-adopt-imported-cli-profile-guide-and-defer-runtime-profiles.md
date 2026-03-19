---
Title: Adopt imported CLI profile guide and defer runtime profiles
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
        Loaded-command runtime path that still duplicates profile/bootstrap logic
        Loaded-command bootstrap path discussed in the decision note
    - Path: ../../../../../../../pinocchio/pkg/cmds/helpers/profile_runtime.go
      Note: |-
        Thin bootstrap helper path that overlaps with loaded-command behavior
        Thin bootstrap path discussed in the decision note
    - Path: pkg/sections/profile_sections.go
      Note: |-
        Current shared profile section and bootstrap middleware path
        Current shared profile section and registry fallback behavior
    - Path: ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries/sources/local/geppetto_cli_profile_guide.md
      Note: |-
        Imported guide adopted as the preferred design basis
        Imported guide adopted as the new direction
ExternalSources:
    - local:geppetto_cli_profile_guide.md
Summary: Adopt the imported CLI profile guide as the preferred direction for the next phase, narrow scope to baseline config plus engine profile overlay, and explicitly defer runtime profiles and runtime composition work.
LastUpdated: 2026-03-19T10:29:16.824626395-04:00
WhatFor: Record the explicit design decision that supersedes the earlier proposal and keeps the next implementation phase focused on config/profile simplification only.
WhenToUse: Use when planning, reviewing, or implementing the next profile/bootstrap cleanup steps in Pinocchio and Geppetto.
---


# Adopt imported CLI profile guide and defer runtime profiles

## Executive Summary

The imported guide in `sources/local/geppetto_cli_profile_guide.md` is a better basis for the next phase than the earlier ticket proposal. It makes the important split explicit: baseline CLI/app configuration is one concern, engine profile registries are a second concern, and Pinocchio runtime composition is a third concern. The next implementation phase should only address the first two.

This ticket therefore adopts the imported guide as the working source of truth for follow-up design and implementation. Runtime profiles, runtime middleware composition, prompt/tool profile layering, and other chat-time profile concerns are intentionally deferred. The immediate goal is a smaller and more coherent contract: load baseline config, optionally overlay an engine profile, and create an engine consistently across Geppetto-backed CLI commands.

## Problem Statement

The earlier proposal mixed multiple cleanup goals together. Some parts focused on replacing manual `--profile` and `--profile-registries` flag wiring. Some parts focused on registry loading and string-list parsing. Some parts drifted toward a much larger runtime-profile design where command runtime behavior, middleware composition, and chat-time switching all become part of the profile story.

That broader framing is one reason the current system feels hard to reason about. The code already contains at least two active bootstrap paths:

- `pinocchio/pkg/cmds/cmd.go` handles loaded YAML commands and preserves command-local defaults.
- `pinocchio/pkg/cmds/helpers/profile_runtime.go` handles thinner entrypoints and resolves config plus optional profile overlays.

Those paths do not currently share one small, explicit API. They also do not agree on default behavior when no registry is present. Adding runtime-profile work before simplifying the baseline config and engine-profile parts would increase the surface area before the foundation is stable.

## Proposed Solution

Adopt the imported guide's decomposition and constrain the next phase to the following model:

```text
baseline config/defaults/env/flags
  -> base InferenceSettings
  -> optional engine profile registry overlay
  -> final InferenceSettings
  -> engine
```

That means:

- `config.yaml` remains the baseline application/CLI configuration file.
- `profiles.yaml` remains an engine profile registry file.
- `profile` and `profile-registries` continue to be expressed through the shared Geppetto `profile-settings` section.
- command bootstrap helpers should standardize how they resolve:
  - config files,
  - profile selection,
  - final `InferenceSettings`,
  - and engine creation.

The next phase should not attempt to redesign:

- chat-time runtime profile switching semantics,
- prompt/tool/middleware runtime composition,
- Pinocchio runtime profiles as a first-class new abstraction,
- or non-engine concerns hidden inside `profiles.yaml`.

## Design Decisions

1. The imported guide is now the preferred source of truth for direction.
Reason:
It reflects the real current code structure more accurately than the earlier ticket framing and cleanly separates concerns that were previously blurred together.

2. Runtime profiles are out of scope for the next phase.
Reason:
The fastest path to a simpler CLI bootstrap story is to stabilize baseline config loading and engine profile overlay first. Runtime-profile design would otherwise block or muddy the simpler helper work.

3. `config.yaml` and `profiles.yaml` keep separate semantics.
Reason:
The imported guide is correct that overloading `profiles.yaml` as a baseline config file would create a schema collision between Glazed config sections and engine profile registry structures.

4. The next phase must converge loaded commands and thin/bootstrap commands on the same helper contracts.
Reason:
The current confusion comes from duplicated but slightly different bootstrap logic, not from the absence of one more helper in only one path.

## Alternatives Considered

Continue with the earlier broader proposal and design runtime profiles now.
Rejected because it expands scope before the current config/profile behavior is stabilized and documented.

Treat `profiles.yaml` as the primary baseline inference source.
Rejected because it conflicts with the current engine profile registry schema and weakens the distinction between baseline config and overlay config.

Only refactor the loaded-command path in `cmd.go` and leave thin/bootstrap commands as-is.
Rejected because that would preserve the current split-brain bootstrap behavior and keep the main source of confusion intact.

## Implementation Plan

1. Finish the documentation-first phase in this ticket:
   - inventory entrypoints,
   - document config vs registry semantics,
   - document default discovery rules,
   - define the minimal first implementation shape.
2. Introduce shared helper contracts for:
   - profile selection,
   - final `InferenceSettings` resolution,
   - engine creation.
3. Refactor loaded commands and thin/bootstrap commands onto the same helper contracts.
4. Add tests for precedence and bootstrap parity.
5. Revisit runtime profiles only after the smaller bootstrap contract is stable.

## Open Questions

Whether the next phase should preserve any legacy fallback lookup for `~/.pinocchio/profiles.yaml` when no explicit `profile-registries` are provided.

Whether `switch-profiles-tui` should stay as a stricter registry-required command even after general CLI bootstrap paths allow baseline-only operation without registries.

## References

- [Imported Source - Geppetto CLI Profile Guide](../sources/local/geppetto_cli_profile_guide.md)
- [Pinocchio CLI Geppetto config and profile bootstrap guide](./01-pinocchio-cli-geppetto-config-and-profile-bootstrap-guide.md)
- [Geppetto backed CLI entrypoint inventory and bootstrap classification](../analysis/03-geppetto-backed-cli-entrypoint-inventory-and-bootstrap-classification.md)
- [Baseline config and engine profile registry semantics](./03-baseline-config-and-engine-profile-registry-semantics.md)
- [Minimal first phase bootstrap implementation without runtime profiles](./04-minimal-first-phase-bootstrap-implementation-without-runtime-profiles.md)
