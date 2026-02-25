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
LastUpdated: 2026-02-25T18:47:33.461980462-05:00
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

1. Phase 1 (migrate command output contract): in progress
2. Phase 2 (codec/store compatibility removal): pending
3. Phase 3 (pinocchio legacy helper/CLI cleanup): pending
4. Phase 4 (script/docs final alignment + validation): pending

## Commands Used So Far

```bash
go test ./cmd/pinocchio/cmds -run MigrateLegacyProfiles -count=1

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

## Related

1. `../design/01-hard-cut-cleanup-inventory-and-removal-plan.md`
2. `../tasks.md`
