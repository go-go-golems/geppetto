---
Title: Registry loading cleanup analysis and migration inventory
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
    - Path: ../../../../../../../pinocchio/pkg/cmds/helpers/profile_runtime.go
      Note: Downstream Glazed-backed caller to migrate later
    - Path: ../../../../../../../temporal-relationships/internal/extractor/gorunner/config.go
      Note: Downstream caller that already has []string before rejoining
    - Path: cmd/examples/geppetto-js-lab/main.go
      Note: Non-Glazed CLI example that still needs local string normalization
    - Path: cmd/examples/runner-glazed-registry-flags/main.go
      Note: Reference Glazed TypeStringList example and migration target
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-19T06:53:42.67853887-04:00
WhatFor: Inventory current uses of ParseEngineProfileRegistrySourceEntries, distinguish Glazed-backed vs non-Glazed entrypoints, and define the migration path for removing the exported helper.
WhenToUse: Use when updating registry-loading code to accept []string inputs directly and deciding where Glazed should own string-list decoding.
---


# Registry Loading Cleanup Analysis

## Decision

`ParseEngineProfileRegistrySourceEntries` should be removed from `pkg/engineprofiles`.

Reason:

- The function is not registry-domain logic. It only trims a raw string, splits on commas, trims entries, and rejects empty members.
- `ParseRegistrySourceSpecs([]string)` is already the better API boundary for registry loading because it operates on normalized entries.
- In Glazed-backed commands, `fields.TypeStringList` already produces `[]string`, so the helper duplicates parsing work that the flag/config layer can already do.

## Current Geppetto Uses

### 1. Glazed example path

Files:

- `cmd/examples/runner-glazed-registry-flags/main.go`
- `cmd/examples/internal/runnerexample/inference_settings.go`

Current state:

- The public command already wants `profile-registries` to behave like a list.
- The in-progress code in this workspace has already switched the command struct to `[]string` and the field type to `fields.TypeStringList`.
- `ResolveInferenceSettingsFromRegistry` still calls the removed helper and is currently inconsistent with that change.

Migration:

- Keep `profile-registries` as `TypeStringList`.
- Accept `[]string` end-to-end in the example helper.
- Call `ParseRegistrySourceSpecs(entries)` directly.

### 2. Geppetto bootstrap/profile-settings path

File:

- `pkg/sections/sections.go`

Current state:

- The bootstrap profile settings struct still decodes `profile-registries` as `string`.
- The section field is still declared as `fields.TypeString`.
- Bootstrap then calls `ParseEngineProfileRegistrySourceEntries(profileSettings.ProfileRegistries)` before validating the result.

Migration:

- Change the section field to `fields.TypeStringList`.
- Change the bootstrap struct field to `[]string`.
- Let Glazed decode defaults/config/env/Cobra into `[]string`.
- Validate `len(profileSettings.ProfileRegistries)` and pass the slice straight to `ParseRegistrySourceSpecs` or downstream consumers.

This is the highest-value cleanup because it removes the helper from Geppetto's main Glazed-backed path, not just the example.

### 3. Non-Glazed CLI example

File:

- `cmd/examples/geppetto-js-lab/main.go`

Current state:

- Uses `flag.String`, so the input is still a raw comma-separated string.

Migration:

- Keep string acceptance in the example.
- Replace the engineprofiles helper with a small local normalization function in this package.

This preserves the example's UX while keeping the registry package focused on typed inputs.

### 4. JS API entrypoint

File:

- `pkg/js/modules/geppetto/api_profiles.go`

Current state:

- `profiles.connectStack(...)` accepts either a comma-separated string or a JS array.
- The string case currently delegates to `ParseEngineProfileRegistrySourceEntries`.

Migration:

- Keep support for both JS input shapes.
- Normalize strings locally in the JS module.
- Continue treating arrays as first-class and pass the final `[]string` to `ParseRegistrySourceSpecs`.

## Downstream Callers Outside Geppetto

These are not part of the Geppetto package removal itself, but they matter because deleting the exported helper will force follow-up migrations.

### Glazed-backed callers that should also move to []string

- `pinocchio/pkg/cmds/helpers/profile_runtime.go`
- `pinocchio/cmd/pinocchio/cmds/js.go`

Observation:

- These paths derive profile settings from parsed Glazed values and should eventually decode registry sources as `[]string`, mirroring the Geppetto cleanup.

Recommended follow-up:

- Change their profile settings sections/structs to `fields.TypeStringList` plus `[]string`.
- Remove the extra string normalization layer.

### Raw-string callers that need local normalization

- `pinocchio/pkg/cmds/helpers/parse-helpers.go`
- `temporal-relationships/internal/extractor/gorunner/config.go`
- `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go`
- `temporal-relationships/internal/extractor/httpapi/server.go`
- `2026-03-16--gec-rag/internal/webchat/profiles.go`

Observation:

- Several of these files already have `[]string` in hand, then `strings.Join(...)` them only to call `ParseEngineProfileRegistrySourceEntries(...)` and recover the same slice shape.

Recommended follow-up:

- If the caller already has `[]string`, pass it directly to `ParseRegistrySourceSpecs`.
- If the caller still starts from a raw string, keep normalization local to that package instead of depending on Geppetto for CSV splitting.

## Scope For This Ticket

In scope now:

- Remove the exported helper from Geppetto.
- Move Geppetto Glazed-backed paths to `[]string`.
- Keep non-Glazed Geppetto entrypoints working via local normalization.
- Update Geppetto tests and docs to show `[]string` as the handoff into `ParseRegistrySourceSpecs`.

Recorded for follow-up:

- Pinocchio and other workspace repos that still import the helper.
- Any Glazed profile settings surfaces outside Geppetto that still use `TypeString`.

## Implementation Notes

- `ParseRegistrySourceSpecs` remains the stable typed API for registry loading.
- Empty-entry rejection still matters; it just moves to the specific input boundary that owns raw-string parsing.
- Documentation should stop demonstrating `raw string -> ParseEngineProfileRegistrySourceEntries -> ParseRegistrySourceSpecs` and instead show either:
  - Glazed-decoded `[]string -> ParseRegistrySourceSpecs`
  - local string normalization -> `[]string -> ParseRegistrySourceSpecs`
