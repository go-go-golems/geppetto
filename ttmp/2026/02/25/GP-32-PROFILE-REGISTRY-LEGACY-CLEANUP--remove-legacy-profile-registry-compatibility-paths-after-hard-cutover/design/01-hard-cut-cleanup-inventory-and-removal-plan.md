---
Title: Hard-cut cleanup inventory and removal plan
Ticket: GP-32-PROFILE-REGISTRY-LEGACY-CLEANUP
Status: active
Topics:
    - profile-registry
    - geppetto
    - pinocchio
    - migration
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy.go
      Note: Migrate command output contract currently emits bundle format
    - Path: ../../../../../../../pinocchio/cmd/pinocchio/main.go
      Note: Old clay profiles command and legacy template still mounted
    - Path: ../../../../../../../pinocchio/pkg/cmds/helpers/parse-helpers.go
      Note: Legacy profile-file helper path still uses GatherFlagsFromProfiles
    - Path: ../../../../../../../pinocchio/scripts/profile_registry_cutover_smoke.sh
      Note: Smoke script assumes bundle conversion/import flow
    - Path: pkg/profiles/codec_yaml.go
      Note: Legacy bundle and legacy-map YAML compatibility branches to remove
    - Path: pkg/profiles/file_store_yaml.go
      Note: YAML store currently reads/writes bundle semantics; align to strict single-registry
    - Path: pkg/profiles/service_test.go
      Note: Contains legacy parity test anchored on GatherFlagsFromProfiles
ExternalSources: []
Summary: Inventory of legacy profile compatibility paths and phased removal plan for strict single-registry runtime.
LastUpdated: 2026-02-25T18:41:07.2014072-05:00
WhatFor: Define exactly which compatibility paths to remove and provide an implementation sequence that preserves runtime behavior while reducing profile-system complexity.
WhenToUse: Use before and during GP-32 implementation to keep scope focused on hard-cut simplification.
---


# Hard-Cut Cleanup Inventory And Removal Plan

## Decision

Hard-cut profile behavior is now authoritative:

1. Runtime YAML is single-registry only (`slug` + `profiles`).
2. Runtime profile loading uses registry-source stacks (`profile-registries`).
3. Legacy map format and bundle (`registries:`) are migration artifacts, not runtime contracts.

GP-32 removes remaining compatibility code that violates or muddies those rules.

## Inventory: Compatibility Paths To Remove

## A) Multi-registry/legacy YAML codec paths in geppetto profiles

- `geppetto/pkg/profiles/codec_yaml.go`
  - `DecodeYAMLRegistries(...)` currently accepts:
    - top-level `registries:`,
    - single-registry docs,
    - legacy profile maps.
  - `EncodeYAMLRegistries(...)` emits top-level bundle format.
  - `ConvertLegacyProfilesMapToRegistry(...)` keeps legacy-map conversion active.

## B) YAML file store still bundle-oriented

- `geppetto/pkg/profiles/file_store_yaml.go`
  - reads via `DecodeYAMLRegistries(...)`,
  - writes via `EncodeYAMLRegistries(...)`.

This keeps bundle semantics alive in persistence behavior.

## C) Migrate command currently emits bundle output by default

- `pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy.go`
  - help text and behavior describe canonical bundle output (`registries.<slug>...`).

This conflicts with the desired operator expectation that migration gives runtime-ready single-registry YAML.

## D) Legacy profile-file helper path still exists

- `pinocchio/pkg/cmds/helpers/parse-helpers.go`
  - `WithProfileFile`,
  - `ParseGeppettoLayers` using `sources.GatherFlagsFromProfiles`.

This is the old profile-file model and should be removed or rewritten for registry sources.

## E) Old clay profiles command still mounted

- `pinocchio/cmd/pinocchio/main.go`
  - `clay_profiles.NewProfilesCommand(...)`,
  - `pinocchioInitialProfilesContent()` template for legacy profile maps.

This keeps old semantics and docs reachable from the main CLI.

## F) Legacy-focused tests anchor old behavior

- `geppetto/pkg/profiles/codec_yaml_test.go` (legacy map + bundle tests)
- `geppetto/pkg/profiles/file_store_yaml_test.go` (legacy load, multi-registry parity)
- `pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy_test.go` (expects bundle output)
- `geppetto/pkg/profiles/service_test.go` (`GoldenAgainstGatherFlagsFromProfiles`)

## G) Script tooling still assumes bundle migration output

- `pinocchio/scripts/profile_registry_cutover_smoke.sh`
  - currently converts legacy to canonical bundle, then imports bundle into sqlite via helper.

## Implementation Plan

## Phase 1: Change migration output contract

1. Make `profiles migrate-legacy` emit runtime single-registry YAML by default.
2. Keep `--registry` as the output `slug` when input is legacy map.
3. If input has multiple registries (`registries:`), fail unless an explicit registry-selection flag is provided (or remove support entirely and require pre-split input).
4. Update command help, tests, and docs.

## Phase 2: Remove codec compatibility branches

1. Remove bundle and legacy-map support from runtime-facing decode paths.
2. Replace `DecodeYAMLRegistries`/`EncodeYAMLRegistries` with strict single-registry codec(s), or keep names but enforce strict shape.
3. Remove `ConvertLegacyProfilesMapToRegistry`.

## Phase 3: Remove old profile-file and clay profile command paths

1. Remove `parse-helpers` profile-file compatibility options/functions.
2. Remove old clay profiles command wiring from pinocchio main entrypoint.
3. Ensure all active commands rely on `sections.GetCobraCommandGeppettoMiddlewares` registry path.

## Phase 4: Test and script cleanup

1. Rewrite tests to validate only supported hard-cut behavior.
2. Update smoke scripts to avoid bundle import assumptions.
3. Re-run geppetto + pinocchio test suites and profile smoke flows.

## Acceptance Criteria

1. No runtime path accepts top-level `registries:` or legacy profile-map YAML.
2. `profiles migrate-legacy` produces runtime-ready single-registry YAML by default.
3. No `GatherFlagsFromProfiles`/`profile-file` runtime flow remains in active code paths.
4. Old clay `profiles` command is removed from pinocchio main CLI.
5. Tests/docs/scripts are aligned to hard-cut behavior only.
