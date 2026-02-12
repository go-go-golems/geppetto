---
Title: Implementation Plan
Ticket: GP-003-JS-TURNS-CODEGEN
Status: active
Topics:
    - geppetto
    - turns
    - codegen
    - go
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/gen-turns/main.go
      Note: Generator implementation
    - Path: cmd/gen-turns/main_test.go
      Note: Generator test coverage
    - Path: pkg/turns/block_kind_gen.go
      Note: Generated kind mapper
    - Path: pkg/turns/keys.go
      Note: Handwritten constants retained
    - Path: pkg/turns/keys_gen.go
      Note: Generated key mappers
    - Path: pkg/turns/spec/turns_codegen.yaml
      Note: Source schema
    - Path: pkg/turns/types.go
      Note: Handwritten code reduced after generation adoption
ExternalSources: []
Summary: Stepwise plan to introduce code generation for turns BlockKind and key-id mappers with incremental commits per task.
LastUpdated: 2026-02-12T14:35:00-05:00
WhatFor: Drive implementation of generated mappers while preserving backward-compatible symbols and behavior.
WhenToUse: Use when implementing and reviewing turns codegen changes in this ticket.
---


# Implementation Plan

## Goal

Introduce deterministic code generation for:

1. `turns.BlockKind` string/YAML mapping logic.
2. Geppetto-owned turns key-id mapper constants and typed key variables.

while preserving all existing exported symbol names currently used by the repo.

## Non-goals

- Changing key naming conventions.
- Changing runtime behavior of key decoding/encoding.
- Refactoring unrelated turns APIs.

## Constraints

- Generated code must be committed.
- Existing references to constants like `ToolConfigValueKey` and vars like `KeyTurnMetaSessionID` must continue compiling.
- Work is split into sequential tasks with one commit per task.

## Task Breakdown

1. Task 1: Generator scaffold + manifest + go:generate wiring.
2. Task 2: Generate and adopt BlockKind mapper code.
3. Task 3: Generate and adopt turns key-id mappers/constants/typed keys.
4. Task 4: Add generator validation tests + regen checks + docs/diary finalization.

## Proposed File Additions

- `pkg/turns/spec/turns_codegen.yaml`
- `pkg/turns/generate.go` (go:generate entrypoint)
- `cmd/gen-turns/main.go` (generator)
- `pkg/turns/block_kind_gen.go` (generated)
- `pkg/turns/keys_gen.go` (generated)
- `pkg/turns/spec/README.md` (manifest semantics)

## Migration Strategy

- Generate code first in parallel with manual definitions.
- Replace manual definitions in small, compile-safe steps.
- Keep comments noting generated ownership.
- Verify with targeted tests each task.

## Validation Commands

- `go run ./cmd/gen-turns --schema pkg/turns/spec/turns_codegen.yaml --out pkg/turns`
- `go generate ./pkg/turns`
- `go test ./pkg/turns/... ./pkg/inference/... -count=1`

## Risks and Mitigations

- Risk: symbol drift causing compile break.
  - Mitigation: preserve exported names in manifest and generated templates.
- Risk: duplicate definitions during transition.
  - Mitigation: replace in isolated task commits with compile checks.
- Risk: stale generated outputs.
  - Mitigation: add test/check that generated files are current.
