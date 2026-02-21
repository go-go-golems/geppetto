---
Title: Analysis of gen-js-api vs gen-turns unification and generated block metadata key exports
Ticket: GP-16-UNIFY-TURNS-KEYS-CODEGEN-MERGE
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - inference
    - turns
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/gen-js-api/main.go
      Note: Current JS API constants generator analyzed for overlap and extension points
    - Path: cmd/gen-turns/main.go
      Note: Turns-domain generator analyzed as canonical source for key/value constants
    - Path: pkg/js/modules/geppetto/consts_gen.go
      Note: Generated gp.consts surface that would gain BlockMetadataKeys group
    - Path: pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl
      Note: Template to update for new generated key groups in JS module typings
    - Path: pkg/js/modules/geppetto/spec/js_api_codegen.yaml
      Note: JS schema currently duplicating some turns-domain values
    - Path: pkg/turns/keys_gen.go
      Note: Generated key constants showing turns-domain source of truth outputs
    - Path: pkg/turns/spec/turns_codegen.yaml
      Note: Canonical turns schema proposed as source for imported key groups
ExternalSources: []
Summary: Detailed comparison of cmd/gen-js-api and cmd/gen-turns, with a unification design that auto-exports turns key groups (including block metadata keys) to the JS API constants surface.
LastUpdated: 2026-02-20T17:58:00-05:00
WhatFor: Clarify generator responsibilities, reduce duplicated schemas, and define an implementation path for generated BlockMetadataKeys exports in gp.consts.
WhenToUse: When implementing generator unification or extending JS constants for turns data/metadata key groups.
---


# Analysis of `gen-js-api` vs `gen-turns` unification and generated block metadata key exports

## Purpose

This analysis explains:

1. What `cmd/gen-js-api/main.go` and `cmd/gen-turns/main.go` each do today.
2. Where they overlap and drift.
3. How to unify them safely.
4. How to add generated block metadata key constants (`BlockMetadataKeys`) in the JS API, plus related key groups.

The immediate practical driver is that key evolution (for example GP-02 inference keys) now requires synchronized updates across multiple schemas/files and has already produced merge friction.

## Current state: side-by-side

## `cmd/gen-turns`

`cmd/gen-turns` is the canonical generator for turns-domain identity data:

- Input schema: `pkg/turns/spec/turns_codegen.yaml`
- Outputs:
  - `pkg/turns/block_kind_gen.go`
  - `pkg/turns/keys_gen.go`
  - `pkg/doc/types/turns.d.ts`
- Owns:
  - `BlockKind` values
  - geppetto namespace value-key constants by scope (`data`, `turn_meta`, `block_meta`)
  - typed keys that are safe to declare in `turns`

## `cmd/gen-js-api`

`cmd/gen-js-api` is focused on JS module constants export and JS-facing d.ts generation:

- Input schema: `pkg/js/modules/geppetto/spec/js_api_codegen.yaml`
- Outputs:
  - `pkg/js/modules/geppetto/consts_gen.go` (`gp.consts.*` installer)
  - `pkg/doc/types/geppetto.d.ts`
- Owns:
  - JS API enum groups (`ToolChoice`, `ToolErrorHandling`, `HookAction`, `EventType`)
  - Also currently duplicates turns-domain values (`BlockKind`, `MetadataKeys`)

## Core mismatch

There are effectively two schema authorities for overlapping concepts:

- turns schema (`turns_codegen.yaml`) for turns identity data
- JS API schema (`js_api_codegen.yaml`) for some of the same values

This duplication creates drift risk and makes merge resolution harder when key sets expand (as seen with new inference config key constants).

## Concrete overlap and drift points

## Overlap 1: `BlockKind`

- Canonical in turns: `block_kinds` from `turns_codegen.yaml`
- Duplicated in JS schema: `enums: BlockKind` in `js_api_codegen.yaml`

## Overlap 2: turn metadata key value names

- Canonical in turns: `keys` entries with `scope: turn_meta`
- Duplicated in JS schema: `enums: MetadataKeys`

## Missing in JS constants surface today

`gp.consts` does not currently expose generated groups for:

- turn data key value names
- block metadata key value names

This is exactly where your request applies: generate block metadata key data for JS constants so app code can avoid raw string literals.

## Additional observation: dead field in `gen-js-api` schema model

`enumValueSchema` includes `GoConst` but the generator currently does not consume it in output rendering or validation. This indicates either:

- a planned but unfinished cross-link mechanism, or
- leftover schema complexity not being enforced.

Either way, it supports the case for simplifying and clarifying ownership.

## Target architecture

Unify around one domain authority and one composition layer:

1. `turns_codegen.yaml` remains canonical for turns-domain vocabulary.
2. `js_api_codegen.yaml` becomes JS-specific overlay/config, not a duplicate key catalog.
3. `gen-js-api` composes:
- turns-domain groups imported from turns schema
- JS-only enums defined in JS schema

This keeps runtime export ergonomics while removing duplicate maintenance.

## Proposed ownership split

## Canonical turns-owned data (from `turns_codegen.yaml`)

- `BlockKind`
- `TurnDataKeys` (from `scope: data`)
- `TurnMetadataKeys` (from `scope: turn_meta`)
- `BlockMetadataKeys` (from `scope: block_meta`)

## JS-only enums (from `js_api_codegen.yaml`)

- `ToolChoice`
- `ToolErrorHandling`
- `HookAction`
- `EventType`
- any future JS module-specific groups not represented in turns

## What this enables immediately

`gp.consts` can provide:

- `gp.consts.BlockKind.*`
- `gp.consts.TurnDataKeys.*`
- `gp.consts.TurnMetadataKeys.*`
- `gp.consts.BlockMetadataKeys.*`
- existing JS-only groups unchanged

This directly addresses "generate the blocks metadata key data" by introducing `BlockMetadataKeys` as generated constants.

## Naming recommendation for JS exports

Use explicit group names to avoid ambiguity:

- `TurnDataKeys`
- `TurnMetadataKeys`
- `BlockMetadataKeys`

Keep `MetadataKeys` as a backward-compat alias only if needed. Since backward compatibility is not required in the current GP-03 context and was explicitly deprioritized earlier, the cleaner option is to move directly to explicit names.

## Unification design options

## Option 1: Keep two binaries, share data contracts (recommended)

- `cmd/gen-turns` stays as-is for turns files.
- `cmd/gen-js-api` gains optional `--turns-schema` input.
- `gen-js-api` composes merged enum groups internally.

Pros:

- low operational disruption
- minimal command migration
- easy incremental rollout

Cons:

- still two CLIs, though with clearer boundaries

## Option 2: One unified CLI (single command with sections)

Merge both tools into one command (for example `cmd/gen-meta` with `turns`, `js-api`, `all` sections).

Pros:

- one entrypoint
- shared validation/render plumbing by default

Cons:

- larger migration
- unnecessary coupling of independent generation flows

Given current repo state and ongoing merge work, Option 1 is safer and faster.

## Proposed schema evolution for JS generator

Keep JS schema focused on export configuration and JS-only groups. Example conceptually:

```yaml
imports:
  turns:
    schema: pkg/turns/spec/turns_codegen.yaml
    groups:
      - source: block_kinds
        name: BlockKind
      - source: data_keys
        name: TurnDataKeys
      - source: turn_meta_keys
        name: TurnMetadataKeys
      - source: block_meta_keys
        name: BlockMetadataKeys

enums:
  - name: ToolChoice
    values: ...
  - name: ToolErrorHandling
    values: ...
  - name: HookAction
    values: ...
  - name: EventType
    values: ...
```

If schema changes are too much for first pass, a simpler first implementation is a CLI flag:

- `gen-js-api --turns-schema pkg/turns/spec/turns_codegen.yaml`

with hardcoded imported group defaults.

## Key transformation rules for imported groups

When importing turns keys into JS enum-like groups:

1. Value source comes from `value` in turns schema (for example `claude_original_content`).
2. JS key identifier is derived by upper snake conversion:
- `claude_original_content` -> `CLAUDE_ORIGINAL_CONTENT`
- `session_id` -> `SESSION_ID`
3. Preserve source ordering from schema for stable diffs.

## Pseudocode for composed generation

```go
turns := loadTurnsSchema(turnsSchemaPath)      // block_kinds + keys[]
js := loadJSSchema(jsSchemaPath)               // js-only enums + import config

enums := []Enum{}

// Imported turns groups
if js.importTurns.BlockKind {
  enums = append(enums, enumFromBlockKinds("BlockKind", turns.BlockKinds))
}
if js.importTurns.TurnDataKeys {
  enums = append(enums, enumFromKeys("TurnDataKeys", turns.Keys, scopeData))
}
if js.importTurns.TurnMetadataKeys {
  enums = append(enums, enumFromKeys("TurnMetadataKeys", turns.Keys, scopeTurnMeta))
}
if js.importTurns.BlockMetadataKeys {
  enums = append(enums, enumFromKeys("BlockMetadataKeys", turns.Keys, scopeBlockMeta))
}

// JS-only groups
enums = append(enums, js.Enums...)

validateNoDuplicateEnumNames(enums)
validateNoDuplicateValuesWithinEnum(enums)

renderGoConsts(enums)
renderGeppettoDTS(enums)
```

## Relationship to current conflict and GP-02

This unification directly reduces the conflict class we just hit:

- New turns keys added once in `turns_codegen.yaml`.
- `gen-turns` updates Go/TS turns artifacts.
- `gen-js-api` imports those turns key groups, so JS constants stay in sync automatically.

This prevents manual duplication between:

- `pkg/turns/keys.go`/`keys_gen.go`
- `pkg/js/modules/geppetto/spec/js_api_codegen.yaml`

and removes one major drift vector.

## Implementation impact map

Likely touched files for the implementation phase:

- `cmd/gen-js-api/main.go`
- `cmd/gen-js-api/main_test.go`
- `pkg/js/modules/geppetto/spec/js_api_codegen.yaml`
- `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
- `pkg/js/modules/geppetto/consts_gen.go` (generated)
- `pkg/doc/types/geppetto.d.ts` (generated)
- optional docs:
  - `pkg/doc/topics/13-js-api-reference.md`

No change needed to turns typed-key ownership in `pkg/inference/engine/turnkeys.go`.

## Validation plan

## Generator tests

- `go test ./cmd/gen-js-api`
- Extend tests to cover imported turns groups and naming conversion.

## Generated artifacts check

- `go generate ./pkg/turns`
- `go generate ./pkg/js/modules/geppetto`
- confirm `gp.consts.BlockMetadataKeys.*` appears in:
  - `consts_gen.go`
  - `geppetto.d.ts`

## Runtime sanity

- `go test ./pkg/js/modules/geppetto -count=1`
- add/extend module tests to verify new group presence.

## Risk assessment

- Risk: accidental breaking rename (`MetadataKeys` -> `TurnMetadataKeys`) for existing JS users.
  - Mitigation: either temporary alias or explicit breaking-change note.

- Risk: imported group includes keys intentionally not meant for JS surface.
  - Mitigation: allow include/exclude filters in JS import config.

- Risk: generator complexity increases.
  - Mitigation: shared helper package for schema loading/validation transforms, keep CLI surface small.

## Recommendation

Proceed with a phased unification:

1. Keep both generators.
2. Extend `gen-js-api` to import turns schema groups.
3. Add generated `TurnDataKeys`, `TurnMetadataKeys`, and `BlockMetadataKeys` exports.
4. Remove duplicated `BlockKind` and metadata keys from JS schema once imported groups are stable.
5. Optionally clean up unused `go_const` field or make it meaningful with validation.

This approach is low-risk, directly addresses your block metadata key request, and aligns codegen with a single turns-domain source of truth.
