---
Title: Reusable profile settings section migration analysis
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
    - Path: ../../../../../../../pinocchio/cmd/pinocchio/main.go
      Note: Pinocchio root now attaches the shared section to inherited Cobra flags
    - Path: ../../../../../../../pinocchio/cmd/switch-profiles-tui/main.go
      Note: Plain Cobra TUI now attaches the shared section instead of raw profile flags
    - Path: ../../../../../../../pinocchio/cmd/web-chat/main.go
      Note: Web-chat uses the shared section and decoded []string registry sources
    - Path: cmd/examples/runner-glazed-registry-flags/main.go
      Note: Geppetto example now consumes the shared section with defaults
    - Path: pkg/sections/sections.go
      Note: Canonical shared profile-settings section and slug now live here
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-19T06:57:38.435034092-04:00
WhatFor: Define the shared Glazed section API for `--profile` and `--profile-registries`, inventory duplicate section owners and raw-flag owners, and set the migration scope for Geppetto and Pinocchio.
WhenToUse: Use when replacing hand-written profile selection sections or raw Cobra flags with the canonical Geppetto `profile-settings` section.
---


# Reusable Profile Settings Section Migration Analysis

## Decision

`geppetto/pkg/sections` should publish the canonical Glazed section for:

- `profile-settings.profile`
- `profile-settings.profile-registries`

The shared API should then replace:

- duplicated `schema.NewSection("profile-settings", ...)` definitions
- most hand-written Cobra flags for `--profile` and `--profile-registries`

## Canonical API Shape

The shared section should live in `geppetto/pkg/sections/sections.go` and be public.

Recommended surface:

- `ProfileSettingsSectionSlug`
- `ProfileSettings`
- `NewProfileSettingsSection(...)`

The canonical field types should be:

- `profile`: `fields.TypeString`
- `profile-registries`: `fields.TypeStringList`

That keeps the parsing responsibility in Glazed and gives callers a normalized `[]string` instead of repeatedly re-splitting comma-separated strings.

## Current Owners Of A Manual `profile-settings` Section

### Geppetto

- `geppetto/pkg/sections/sections.go`
- `geppetto/cmd/examples/runner-glazed-registry-flags/main.go`

### Pinocchio

- `pinocchio/pkg/cmds/helpers/profile_runtime.go`
- `pinocchio/cmd/web-chat/main.go`
- test-only section construction in `pinocchio/pkg/cmds/cmd_profile_registry_test.go`

These should all move to the single public Geppetto constructor.

## Raw Flag Owners

### Strong candidates to migrate now

These are already Glazed/Cobra-oriented and should stop defining profile flags manually:

- `pinocchio/cmd/pinocchio/main.go`
  - currently adds a root persistent raw `--profile-registries`
- `pinocchio/cmd/pinocchio/cmds/js.go`
  - currently adds a local raw `--profile`
  - manually constructs parsed section values afterward
- `pinocchio/cmd/examples/internal/tuidemo/cli.go`
  - plain Cobra, but can still add the shared section via `AddSectionToCobraCommand`

### Lower-priority or likely exception entrypoints

These are small standalone `flag`-based programs rather than Glazed/Cobra commands:

- `geppetto/cmd/examples/geppetto-js-lab/main.go`
- `pinocchio/scripts/profile-infer-once/main.go`

Follow-up note: `geppetto/cmd/examples/runner-registry/main.go` was later converted to a small Cobra command so it could mount the shared section directly.

## Migration Strategy

### 1. Publish the shared section in Geppetto

- Export the section slug constant.
- Export a shared decoded settings struct.
- Export a constructor with optional defaults so examples can still set:
  - default profile slug
  - default registry source entries

### 2. Replace duplicated section definitions

Use the shared constructor in:

- Geppetto example commands
- Pinocchio helper/bootstrap code
- Pinocchio web-chat
- tests that currently hand-build a local `profile-settings` section

### 3. Replace raw flags where the command shape already supports it

For existing Cobra commands:

- use the shared section to add the flags to the Cobra command
- stop duplicating the `profile` / `profile-registries` field definitions

This does not require every command to become a full `cmds.CommandDescription`; plain Cobra commands can still attach the shared section through the Glazed `CobraSection` interface.

## Type Consequences

Making `profile-registries` canonical as `TypeStringList` means decoded settings should become `[]string` in the shared paths.

Implications:

- Geppetto bootstrap can validate `len(ProfileRegistries)` directly.
- Pinocchio helper types that decode the shared section should move from `string` to `[]string`.
- Boundaries that still call string-based APIs can `strings.Join(entries, ",")` locally until those downstream APIs are cleaned up too.

## In-Scope Code Paths For This Pass

- `geppetto/pkg/sections/sections.go`
- `geppetto/cmd/examples/runner-glazed-registry-flags/main.go`
- `pinocchio/pkg/cmds/helpers/profile_runtime.go`
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/cmd/pinocchio/main.go`
- `pinocchio/cmd/pinocchio/cmds/js.go`
- `pinocchio/cmd/examples/simple-chat/main.go`
- `pinocchio/cmd/agents/simple-chat-agent/main.go`
- relevant tests for the above

## Explicit Exceptions For Now

- pure `flag`-package binaries and scripts
- shell scripts and documentation examples

Those should be recorded, but not block adoption of the shared section in the main Glazed/Cobra paths.
