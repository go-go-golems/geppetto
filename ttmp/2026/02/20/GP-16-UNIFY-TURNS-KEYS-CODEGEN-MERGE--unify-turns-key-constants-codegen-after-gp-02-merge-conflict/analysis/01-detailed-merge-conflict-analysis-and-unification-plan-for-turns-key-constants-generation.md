---
Title: Detailed merge-conflict analysis and unification plan for turns key constants generation
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
    - Path: pkg/doc/types/turns.d.ts
      Note: Generated TypeScript declarations for turns constants
    - Path: pkg/inference/engine/turnkeys.go
      Note: Engine-owned typed keys referencing turns value constants
    - Path: pkg/turns/keys.go
      Note: Current unresolved merge conflict file
    - Path: pkg/turns/keys_gen.go
      Note: Generated key constants output that should become canonical
    - Path: pkg/turns/spec/turns_codegen.yaml
      Note: Schema source of truth for generated turns key constants
ExternalSources: []
Summary: Analysis of GP-02 merge conflict between manual turns key constants and schema/codegen-based key generation, with a recommended unification strategy.
LastUpdated: 2026-02-20T17:44:00-05:00
WhatFor: Resolve GP-02 merge conflict elegantly and prevent future key-definition drift.
WhenToUse: When reconciling turns key constants, codegen, and inference config key additions after merging main.
---


# Detailed merge-conflict analysis and unification plan for turns key constants generation

## Executive summary

The merge conflict is not only a text conflict in `pkg/turns/keys.go`; it is a source-of-truth conflict.

- GP-02 introduced new inference key value constants (`InferenceConfigValueKey`, `ClaudeInferenceConfigValueKey`, `OpenAIInferenceConfigValueKey`) by editing `pkg/turns/keys.go` directly.
- Mainline now treats the geppetto namespace/value constants and typed keys as generated output in `pkg/turns/keys_gen.go`, with `pkg/turns/spec/turns_codegen.yaml` as the intended source of truth.
- The current generator schema does not yet include the new inference value constants, so generated files lag GP-02's manual additions.

The elegant resolution is to unify on schema-driven generation and stop editing generated key constants manually. Concretely: keep `keys.go` for payload/run metadata only, add new inference value constants to `turns_codegen.yaml`, regenerate `keys_gen.go` and `pkg/doc/types/turns.d.ts`, and resolve the merge to that model.

## What changed on each side

### Side A: GP-02 branch (local merge side)

- Added inference-config feature and engine turn keys in `pkg/inference/engine/turnkeys.go`:
  - `KeyInferenceConfig`
  - `KeyClaudeInferenceConfig`
  - `KeyOpenAIInferenceConfig`
- These keys reference constants in `turns`:
  - `turns.InferenceConfigValueKey`
  - `turns.ClaudeInferenceConfigValueKey`
  - `turns.OpenAIInferenceConfigValueKey`

### Side B: merged mainline

- Introduced (or solidified) generated turns key constants:
  - `pkg/turns/spec/turns_codegen.yaml` (schema)
  - `pkg/turns/keys_gen.go` (generated Go constants and typed keys)
  - `pkg/doc/types/turns.d.ts` (generated TS declarations)
- In this model, constants under geppetto namespace/value keys are expected to be generated.

## Current observed conflict and why it happened

There is an unresolved conflict in `pkg/turns/keys.go` (`UU`).

Conflict content shows:

- **Ours** (`:2`) keeps `keys.go` minimal (payload keys + run metadata key).
- **Theirs** (`:3`) keeps the old/manual full constants section in `keys.go`, including the three new inference constants.

Meanwhile, `pkg/turns/keys_gen.go` already exists and currently defines many namespace/value constants, but it does **not** include the three new inference constants because the schema file does not include them yet.

So conflict resolution by "just choosing theirs" or "just choosing ours" is insufficient without addressing schema/generated output consistency.

## Architectural constraints that matter

1. Import-cycle constraint remains valid.
- Typed keys with engine-owned types (ToolConfig, InferenceConfig, provider configs) must remain in `pkg/inference/engine/turnkeys.go`.
- `turns` can own only string value constants and typed keys that do not force an import cycle.

2. There should be one source of truth for geppetto namespace value-key constants.
- If both manual and generated definitions exist, drift is guaranteed.
- Drift impacts both Go (`keys_gen.go`) and TS (`turns.d.ts`) consumers.

3. JS codec and external API docs rely on stable key identifiers.
- If constants diverge between files, subtle behavior mismatch appears in serialization and JS interop.

## Why this conflict is risky if resolved naively

### Naive resolution A: Accept `theirs` in `keys.go`

This may compile, but it keeps dual authority:
- manual constants in `keys.go`
- generated constants in `keys_gen.go`

Risks:
- duplicate declarations or future redeclaration conflicts
- future edits accidentally made in the wrong file
- generated TS declarations (`turns.d.ts`) may still miss new keys

### Naive resolution B: Accept `ours` in `keys.go` only

This keeps generated model, but without schema update the three inference constants are absent in `keys_gen.go`.

Risks:
- compile failures in `pkg/inference/engine/turnkeys.go`
- incomplete generated TS constants

## Recommended unification strategy (elegant path)

Adopt a strict rule:

- `pkg/turns/keys.go` is manual and limited to non-generated payload/run metadata constants.
- `pkg/turns/spec/turns_codegen.yaml` is the single source for geppetto namespace value keys and typed turns metadata/data keys.
- `pkg/turns/keys_gen.go` and `pkg/doc/types/turns.d.ts` are generated artifacts and never hand-edited.

### Add these entries to schema (`scope: data`)

- `InferenceConfigValueKey` = `inference_config`
- `ClaudeInferenceConfigValueKey` = `claude_inference_config`
- `OpenAIInferenceConfigValueKey` = `openai_inference_config`

`typed_key` should remain empty for these three, because typed key declarations for their concrete types stay in `pkg/inference/engine/turnkeys.go` (import-cycle-safe layering).

## Proposed implementation sequence (for the follow-up implementation task)

1. Resolve conflict markers in `pkg/turns/keys.go` by keeping the minimal/manual-only version.
2. Edit `pkg/turns/spec/turns_codegen.yaml`:
- Add the 3 inference value-key constants under `keys` with `scope: data`, empty `typed_key`, empty `type_expr`.
3. Regenerate turns artifacts:
- `go generate ./pkg/turns`
4. Verify generated outputs now include all three new constants:
- `pkg/turns/keys_gen.go`
- `pkg/doc/types/turns.d.ts`
5. Run compile/tests at minimum for changed surfaces:
- `go test ./pkg/turns ./pkg/inference/engine ./pkg/steps/ai/... -count=1`
6. Run full pre-commit checks (or repo standard hook) before final merge commit.

## Detailed validation checklist

### Compile/link validation

- `pkg/inference/engine/turnkeys.go` resolves:
  - `turns.InferenceConfigValueKey`
  - `turns.ClaudeInferenceConfigValueKey`
  - `turns.OpenAIInferenceConfigValueKey`
- No duplicate constant declarations between `keys.go` and `keys_gen.go`.

### Generated-output validation

- `pkg/turns/keys_gen.go` contains the three constants exactly once.
- `pkg/doc/types/turns.d.ts` contains TS declarations for all three constants.

### Behavioral regression validation

- GP-02 inference tests still pass (`helpers_test.go` in provider packages and engine inference config tests).
- Turn serialization/deserialization tests still pass (`pkg/turns/serde/*`).

## Additional hardening recommendations

1. Add a short developer note in `pkg/turns/generate.go` or CONTRIBUTING section:
- "Do not hand-edit geppetto namespace key constants; edit `spec/turns_codegen.yaml` and regenerate."

2. Add a CI guard (later):
- fail build if `go generate ./pkg/turns` changes tracked files.

3. Optional linter rule (later):
- disallow definitions of `*ValueKey` constants in `pkg/turns/keys.go` except payload/run metadata constants.

## File-by-file reconciliation map

- `pkg/turns/keys.go`
  - Keep payload constants (`PayloadKey*`) and run metadata constants (`RunMetaKey*`) only.
  - Remove or avoid manual geppetto namespace/value-key blocks.

- `pkg/turns/spec/turns_codegen.yaml`
  - Add missing inference value-key entries (source of truth).

- `pkg/turns/keys_gen.go`
  - Regenerated output must include new constants.

- `pkg/doc/types/turns.d.ts`
  - Regenerated output must include new constants.

- `pkg/inference/engine/turnkeys.go`
  - Keep typed key declarations where they are (correct ownership boundary).

## Risk matrix

- Risk: duplicate declarations remain after merge
  - Mitigation: search for all `*InferenceConfigValueKey` declarations before commit.

- Risk: generated file not refreshed
  - Mitigation: mandatory `go generate ./pkg/turns` and diff check.

- Risk: future contributors reintroduce manual edits
  - Mitigation: explicit doc note + CI guard.

## Recommended ticket scope for implementation

This GP-16 ticket should focus on unification and merge conflict resolution only:

- unify source of truth,
- regenerate artifacts,
- validate compile/tests,
- document workflow.

It should not expand into broader inference-config behavior changes; that belongs in GP-02 follow-up tickets.

## Appendix: concise decision statement

Decision: **Adopt schema-driven key constant generation as the canonical mechanism and migrate GP-02 inference value keys into `turns_codegen.yaml`; keep engine typed keys in `pkg/inference/engine/turnkeys.go`; resolve `keys.go` conflict to payload/run-only manual constants.**

This gives one authority for key string constants, keeps import boundaries clean, and minimizes future merge friction.
