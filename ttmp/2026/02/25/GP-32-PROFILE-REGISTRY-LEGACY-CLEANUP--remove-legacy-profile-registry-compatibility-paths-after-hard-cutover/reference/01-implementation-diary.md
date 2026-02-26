---
Title: Implementation diary
Ticket: GP-32-PROFILE-REGISTRY-LEGACY-CLEANUP
Status: active
Topics:
    - profile-registry
    - geppetto
    - pinocchio
    - migration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Step-by-step implementation diary for GP-32 hard-cut profile cleanup.
LastUpdated: 2026-02-25T19:18:00.000000000-05:00
WhatFor: Capture implementation progress, exact commands, and validation evidence while removing legacy profile compatibility paths.
WhenToUse: Read when reviewing GP-32 implementation decisions, commit boundaries, and test evidence.
---

# Implementation diary

## Goal

Execute GP-32 task-by-task with hard-cut semantics and keep a precise execution trail.

## Context

Runtime behavior is already single-registry YAML + profile-registry stacks, but compatibility code still exists in migration tools/codecs/helpers/tests. This diary tracks removal work by phase.

## Quick Reference

## Phase Status

1. Phase 1 (migrate command output contract): completed
2. Phase 2 (codec/store compatibility removal): completed
3. Phase 3 (pinocchio legacy helper/CLI cleanup): completed
4. Phase 4 (script/docs final alignment + validation): completed

## Commands Used So Far

```bash
go test ./cmd/pinocchio/cmds -run MigrateLegacyProfiles -count=1
go test ./pkg/profiles -count=1
go test ./cmd/pinocchio -count=1
go run ./cmd/pinocchio profiles --help
scripts/profile_registry_cutover_smoke.sh

# manual smoke
go run ./cmd/pinocchio profiles migrate-legacy \
  --input <legacy-profiles.yaml> \
  --output <out.yaml> \
  --registry default
```

## Usage Examples

## Step 1: Change `profiles migrate-legacy` to output runtime single-registry YAML

Date: 2026-02-25

Files changed:

1. `pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy.go`
2. `pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy_test.go`

What changed:

1. Command help/description updated to runtime single-registry output.
2. Output contract changed from bundle (`registries:`) to runtime YAML (`slug` + `profiles`).
3. Default output filename changed from `<input>.registry.yaml` to `<input>.runtime.yaml`.
4. Canonical bundle input is now rejected in non-skip mode with a hard-cut error.
5. Command no longer depends on `DecodeYAMLRegistries` / `EncodeYAMLRegistries`; it now:
   - converts legacy map input directly to one runtime registry,
   - decodes existing single-registry runtime YAML via `DecodeRuntimeYAMLSingleRegistry`,
   - encodes output as runtime YAML with no `default_profile_slug`.
6. Tests updated to assert:
   - no `registries:`,
   - no `default_profile_slug:`,
   - explicit error on canonical bundle input without `--skip-if-not-legacy`.

Validation:

1. `go test ./cmd/pinocchio/cmds -run MigrateLegacyProfiles -count=1` passed.
2. Manual command smoke confirmed output shape:

```yaml
slug: default
profiles:
  ...
```

## Step 2: Remove codec/store legacy compatibility and enforce runtime YAML hard-cut

Date: 2026-02-25

Files changed:

1. `geppetto/pkg/profiles/codec_yaml.go`
2. `geppetto/pkg/profiles/codec_yaml_runtime.go`
3. `geppetto/pkg/profiles/file_store_yaml.go`
4. `geppetto/pkg/profiles/codec_yaml_test.go`
5. `geppetto/pkg/profiles/file_store_yaml_test.go`
6. `geppetto/pkg/profiles/integration_store_parity_test.go`

What changed:

1. Removed legacy and bundle decode behavior from `DecodeYAMLRegistries` by routing through strict runtime decoding.
2. Removed legacy-map conversion support (`ConvertLegacyProfilesMapToRegistry`) from the codec surface.
3. Added `EncodeRuntimeYAMLSingleRegistry(...)` and switched compatibility encoding to one-registry-per-file semantics.
4. Updated `YAMLFileProfileStore` to:
   - load only runtime single-registry YAML,
   - persist only one registry,
   - reject operations against non-default registry slugs.
5. Replaced codec tests to explicitly assert hard-cut behavior:
   - reject legacy profile-map YAML,
   - reject canonical `registries:` bundle YAML,
   - preserve runtime fields/extensions/stack refs in single-registry roundtrips.
6. Updated YAML store tests to reject legacy file loading and reject second registry slug writes.
7. Updated stack-ref parity test backend matrix to run multi-registry parity only on memory/sqlite backends.

Validation:

1. `go test ./pkg/profiles -count=1` passed.

## Step 3: Remove pinocchio legacy helper/profile command wiring

Date: 2026-02-25

Files changed:

1. `pinocchio/pkg/cmds/helpers/parse-helpers.go`
2. `pinocchio/cmd/pinocchio/main.go`
3. `pinocchio/scripts/profile_registry_cutover_smoke.sh`

What changed:

1. Removed helper-level legacy profile-file flow (`WithProfileFile`, `sources.GatherFlagsFromProfiles`).
2. Added helper-level registry-stack parsing using `PINOCCHIO_PROFILE_REGISTRIES` and default `~/.config/pinocchio/profiles.yaml` when present.
3. Switched helper profile middleware to `GatherFlagsFromProfileRegistry(...)`.
4. Removed Clay legacy profiles command setup and legacy initial profile template from `cmd/pinocchio/main.go`.
5. Added a native `profiles` command group and kept `profiles migrate-legacy` under that group.
6. Updated smoke script to migrate and import runtime single-registry YAML directly.

Validation:

1. `go test ./pkg/cmds/helpers ./cmd/pinocchio ./cmd/pinocchio/cmds -run MigrateLegacyProfiles -count=1` passed.
2. `go run ./cmd/pinocchio profiles --help` showed expected command surface (`migrate-legacy` only).

## Step 4: Final docs alignment and end-to-end validation

Date: 2026-02-25

Files changed:

1. `geppetto/pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md`

What changed:

1. Rewrote migration playbook to hard-cut runtime single-registry output contract.
2. Removed canonical bundle migration flow language and replaced it with runtime YAML verification/activation steps.

Validation:

1. `scripts/profile_registry_cutover_smoke.sh` passed end-to-end:
   - legacy migration,
   - sqlite import,
   - web-chat profile selection + chat checks,
   - pinocchio `--print-parsed-fields` registry metadata checks.
2. Pre-commit validation on changed repos passed:
   - `go test ./...` in geppetto,
   - `go test ./...` in pinocchio.

## Step 5: Remove Glazed profile-settings flag injection (`profile-file`) and own profile flags in geppetto section

Date: 2026-02-25

Files changed:

1. `geppetto/pkg/sections/sections.go`
2. `geppetto/pkg/sections/profile_registry_source_test.go`
3. `geppetto/cmd/examples/simple-inference/main.go`
4. `pinocchio/cmd/pinocchio/main.go`
5. `pinocchio/cmd/pinocchio/main_profile_registries_test.go`

What changed:

1. Added profile section ownership directly in `CreateGeppettoSections()` by appending custom `profile-settings` with fields:
   - `profile`
   - `profile-registries`
2. Removed command wiring that injected Glazed profile section (`cli.WithProfileSettingsSection()`), which was introducing legacy `--profile-file`.
3. Switched geppetto section internals to local `profile-settings` slug constant, decoupling from Glazed profile section constants.
4. Updated section tests to avoid adding `cli.NewProfileSettingsSection()` and added explicit regression assertion:
   - `--profile` and `--profile-registries` exist,
   - `--profile-file` does not exist.
5. Added pinocchio regression test to assert `--profile-file` fails as unknown flag.

Validation:

1. `go test ./pkg/sections ./cmd/examples/simple-inference -count=1` in geppetto passed.
2. `go test ./cmd/pinocchio -count=1` in pinocchio passed.
3. Manual CLI check:
   - `go run ./cmd/pinocchio code professional --profile-file ...` now fails with `unknown flag: --profile-file`.

## Related

1. `../design/01-hard-cut-cleanup-inventory-and-removal-plan.md`
2. `../tasks.md`
