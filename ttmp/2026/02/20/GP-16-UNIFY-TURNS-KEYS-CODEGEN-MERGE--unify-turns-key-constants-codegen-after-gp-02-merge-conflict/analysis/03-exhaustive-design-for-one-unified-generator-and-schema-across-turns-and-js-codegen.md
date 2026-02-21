---
Title: Exhaustive design for one unified generator and schema across turns and JS codegen
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
      Note: Legacy JS constants generator targeted for consolidation
    - Path: cmd/gen-turns/main.go
      Note: Legacy turns generator targeted for consolidation
    - Path: pkg/inference/engine/turnkeys.go
      Note: Manual engine-typed key ownership targeted for generated output
    - Path: pkg/js/modules/geppetto/codec.go
      Note: Manual key short-ID maps targeted for generated output
    - Path: pkg/js/modules/geppetto/spec/js_api_codegen.yaml
      Note: Current JS enum schema baseline for unified manifest design
    - Path: pkg/turns/keys.go
      Note: Manual payload and run metadata key families identified for schema migration
    - Path: pkg/turns/spec/turns_codegen.yaml
      Note: Current turns schema baseline for unified manifest design
    - Path: ttmp/2026/02/20/GP-16-UNIFY-TURNS-KEYS-CODEGEN-MERGE--unify-turns-key-constants-codegen-after-gp-02-merge-conflict/scripts/analyze_codegen_overlap.go
      Note: Experiment script used to inventory current codegen surfaces
ExternalSources: []
Summary: Exhaustive design for replacing cmd/gen-turns and cmd/gen-js-api with one unified generator and one schema that cover all key families, block kinds, JS const exports, Go constants, typed key declarations, and TypeScript outputs.
LastUpdated: 2026-02-20T18:23:00-05:00
WhatFor: Define a consistent, no-backward-compat architecture for all turns/JS codegen and eliminate drift classes seen in GP-02 merge work.
WhenToUse: When implementing the next-generation single codegen system and migrating existing generators and schemas.
---


# Exhaustive design for one unified generator and schema across turns and JS codegen

## Executive summary

We should replace `cmd/gen-turns` and `cmd/gen-js-api` with a single generator command backed by a single manifest.

Current state is better than before (we already import turns schema into JS const generation), but it is still split:

- two binaries,
- two schema models,
- some families still manual (`PayloadKey*`, `RunMetaKey*`),
- typed key ownership split between `turns` and `engine` with manual maintenance.

Because backward compatibility is not required, the clean design is:

1. one schema (`geppetto_codegen.yaml`) for all identity families and JS enum groups,
2. one generator (`cmd/gen-meta` or similar),
3. one intermediate representation (IR),
4. multiple deterministic emitters (Go + TS + JS module const exports),
5. strict validation + golden tests + “generated-files-are-clean” CI guard.

This removes the exact drift class that caused the GP-02 merge conflict and creates a coherent foundation for future growth (provider-specific keys, new block metadata families, richer `.d.ts` generation).

## Why this needs to be bigger than the current partial fix

The recent fix (`gen-js-api --turns-schema`) solved duplicated `BlockKind`/metadata key values, but the architecture still has structural inconsistency:

- `turns_codegen.yaml` models only `block_kinds` + three key scopes.
- `js_api_codegen.yaml` models JS enums only.
- `PayloadKey*` and `RunMetaKey*` still live manually in `pkg/turns/keys.go`.
- Engine-owned typed keys (`pkg/inference/engine/turnkeys.go`) are manual and loosely coupled to turns value constants.
- JS codec short-key maps are still manually curated in `pkg/js/modules/geppetto/codec.go`.

If we want this solved “elegantly” and consistently, we need to unify the whole key/enum surface into one model.

## Baseline inventory (experiment)

A ticket-local experiment script was added and run:

- Script: `ttmp/2026/02/20/GP-16-UNIFY-TURNS-KEYS-CODEGEN-MERGE--unify-turns-key-constants-codegen-after-gp-02-merge-conflict/scripts/analyze_codegen_overlap.go`

Observed output:

```text
== Codegen Surface Inventory ==
turns.namespace: geppetto
turns.block_kinds: 7
turns.keys[data]: 8
turns.keys[turn_meta]: 8
turns.keys[block_meta]: 5
js.enums: 4
js.enum.names: EventType, HookAction, ToolChoice, ToolErrorHandling
manual.payload_keys(keys.go): 9
manual.run_meta_keys(keys.go): 1
```

Meaning:

- major families are generated,
- but payload and run metadata key families are still manual,
- and generator concerns are still split across two commands.

## Design goals (no backward compatibility constraint)

1. Single source of truth for all key and enum identities.
2. Single generator command for all outputs.
3. All key families represented explicitly (including payload and run metadata).
4. Explicit ownership model for typed keys to avoid import cycles.
5. Deterministic generated outputs with stable ordering.
6. Eliminate hand-edited generated-identity files.
7. Keep API naming internally consistent (allow breaking rename cleanup).

## Non-goals

1. No attempt to infer full JS runtime API dynamically from goja wiring in this phase.
2. No immediate replacement of all hand-authored `.d.ts` method signatures with reflection.
3. No automatic migration compatibility layer.

## Proposed target architecture

## 1) New single manifest

Proposed path:

- `pkg/spec/geppetto_codegen.yaml`

It should include all identity families and export policies.

### Proposed schema shape (conceptual)

```yaml
version: 1
namespace: geppetto

families:
  block_kinds:
    - id: user
      go_const: BlockKindUser
    - id: llm_text
      go_const: BlockKindLLMText
    - id: tool_call
      go_const: BlockKindToolCall
    - id: tool_use
      go_const: BlockKindToolUse
    - id: system
      go_const: BlockKindSystem
    - id: reasoning
      go_const: BlockKindReasoning
    - id: other
      go_const: BlockKindOther

  keys:
    data:
      - value: tool_config
        go_value_const: ToolConfigValueKey
        typed_key:
          owner: engine
          name: KeyToolConfig
          type_expr: ToolConfig
      - value: structured_output_config
        go_value_const: StructuredOutputConfigValueKey
        typed_key:
          owner: engine
          name: KeyStructuredOutputConfig
          type_expr: StructuredOutputConfig
      - value: inference_config
        go_value_const: InferenceConfigValueKey
        typed_key:
          owner: engine
          name: KeyInferenceConfig
          type_expr: InferenceConfig
      - value: claude_inference_config
        go_value_const: ClaudeInferenceConfigValueKey
        typed_key:
          owner: engine
          name: KeyClaudeInferenceConfig
          type_expr: ClaudeInferenceConfig
      - value: openai_inference_config
        go_value_const: OpenAIInferenceConfigValueKey
        typed_key:
          owner: engine
          name: KeyOpenAIInferenceConfig
          type_expr: OpenAIInferenceConfig
      - value: agent_mode_allowed_tools
        go_value_const: AgentModeAllowedToolsValueKey
        typed_key:
          owner: turns
          name: KeyAgentModeAllowedTools
          type_expr: "[]string"
      - value: agent_mode
        go_value_const: AgentModeValueKey
        typed_key:
          owner: turns
          name: KeyAgentMode
          type_expr: string
      - value: responses_server_tools
        go_value_const: ResponsesServerToolsValueKey
        typed_key:
          owner: turns
          name: KeyResponsesServerTools
          type_expr: "[]any"

    turn_meta:
      - value: provider
        go_value_const: TurnMetaProviderValueKey
        typed_key: { owner: turns, name: KeyTurnMetaProvider, type_expr: string }
      - value: runtime
        go_value_const: TurnMetaRuntimeValueKey
        typed_key: { owner: turns, name: KeyTurnMetaRuntime, type_expr: any }
      - value: session_id
        go_value_const: TurnMetaSessionIDValueKey
        typed_key: { owner: turns, name: KeyTurnMetaSessionID, type_expr: string }
      - value: inference_id
        go_value_const: TurnMetaInferenceIDValueKey
        typed_key: { owner: turns, name: KeyTurnMetaInferenceID, type_expr: string }
      - value: trace_id
        go_value_const: TurnMetaTraceIDValueKey
        typed_key: { owner: turns, name: KeyTurnMetaTraceID, type_expr: string }
      - value: usage
        go_value_const: TurnMetaUsageValueKey
        typed_key: { owner: turns, name: KeyTurnMetaUsage, type_expr: any }
      - value: stop_reason
        go_value_const: TurnMetaStopReasonValueKey
        typed_key: { owner: turns, name: KeyTurnMetaStopReason, type_expr: string }
      - value: model
        go_value_const: TurnMetaModelValueKey
        typed_key: { owner: turns, name: KeyTurnMetaModel, type_expr: string }

    block_meta:
      - value: claude_original_content
        go_value_const: BlockMetaClaudeOriginalContentValueKey
        typed_key: { owner: turns, name: KeyBlockMetaClaudeOriginalContent, type_expr: any }
      - value: tool_calls
        go_value_const: BlockMetaToolCallsValueKey
        typed_key: { owner: turns, name: KeyBlockMetaToolCalls, type_expr: any }
      - value: middleware
        go_value_const: BlockMetaMiddlewareValueKey
        typed_key: { owner: turns, name: KeyBlockMetaMiddleware, type_expr: string }
      - value: agentmode_tag
        go_value_const: BlockMetaAgentModeTagValueKey
        typed_key: { owner: turns, name: KeyBlockMetaAgentModeTag, type_expr: string }
      - value: agentmode
        go_value_const: BlockMetaAgentModeValueKey
        typed_key: { owner: turns, name: KeyBlockMetaAgentMode, type_expr: string }

    run_meta:
      - value: trace_id
        go_value_const: RunMetaKeyTraceID
        typed_key:
          owner: turns
          name: ""
          type_expr: ""

  payload_keys:
    - value: text
      go_value_const: PayloadKeyText
    - value: id
      go_value_const: PayloadKeyID
    - value: name
      go_value_const: PayloadKeyName
    - value: args
      go_value_const: PayloadKeyArgs
    - value: result
      go_value_const: PayloadKeyResult
    - value: error
      go_value_const: PayloadKeyError
    - value: images
      go_value_const: PayloadKeyImages
    - value: encrypted_content
      go_value_const: PayloadKeyEncryptedContent
    - value: item_id
      go_value_const: PayloadKeyItemID

js_enums:
  - name: ToolChoice
    doc: How the model should choose tools
    values:
      - key: AUTO
        value: auto
      - key: NONE
        value: none
      - key: REQUIRED
        value: required
  - name: ToolErrorHandling
    doc: How to handle tool execution errors
    values:
      - key: CONTINUE
        value: continue
      - key: ABORT
        value: abort
      - key: RETRY
        value: retry
  - name: HookAction
    doc: Actions returned from tool hook callbacks
    values:
      - key: ABORT
        value: abort
      - key: RETRY
        value: retry
      - key: CONTINUE
        value: continue
  - name: EventType
    doc: Streaming event types for RunHandle.on()
    values:
      - key: START
        value: start
      - key: PARTIAL
        value: partial
      - key: FINAL
        value: final
      - key: TOOL_CALL
        value: tool-call
      - key: TOOL_RESULT
        value: tool-result
      - key: ERROR
        value: error

js_exports:
  const_groups:
    - name: BlockKind
      source: families.block_kinds
    - name: TurnDataKeys
      source: families.keys.data
    - name: TurnMetadataKeys
      source: families.keys.turn_meta
    - name: BlockMetadataKeys
      source: families.keys.block_meta
    - name: RunMetadataKeys
      source: families.keys.run_meta
    - name: PayloadKeys
      source: families.payload_keys
    - name: ToolChoice
      source: js_enums.ToolChoice
    - name: ToolErrorHandling
      source: js_enums.ToolErrorHandling
    - name: HookAction
      source: js_enums.HookAction
    - name: EventType
      source: js_enums.EventType

outputs:
  turns_go:
    file: pkg/turns/generated_identities.go
  engine_go:
    file: pkg/inference/engine/turnkeys_gen.go
  turns_dts:
    file: pkg/doc/types/turns.d.ts
  geppetto_consts_go:
    file: pkg/js/modules/geppetto/consts_gen.go
  geppetto_dts:
    file: pkg/doc/types/geppetto.d.ts
  geppetto_codec_maps_go:
    file: pkg/js/modules/geppetto/codec_keys_gen.go
```

Key idea: a single manifest can still keep package ownership correct by declaring `typed_key.owner`.

## 2) One generator command

Proposed binary:

- `cmd/gen-meta/main.go`

CLI:

- `--schema <path>` required
- `--section all|turns-go|engine-go|turns-dts|js-consts-go|geppetto-dts|codec-go`
- `--check` (validate generated files are up to date, do not write)
- `--write` default mode

No more separate `cmd/gen-turns` and `cmd/gen-js-api`.

## 3) Intermediate representation (IR)

Generator should parse the YAML into a normalized IR first, then emit all outputs from IR.

Core IR types:

- `Namespace`
- `BlockKinds []Kind`
- `KeyFamilies map[Scope][]KeyDef` where scopes include `data`, `turn_meta`, `block_meta`, `run_meta`, `payload`
- `TypedKeyDefs []TypedKeyDef` with package owner
- `JSEnums []EnumDef`
- `JSExports []ExportGroup`

All emitters use the same IR to prevent divergent logic.

## 4) Emitters and generated files

### Turns Go emitter

Generate one file containing:

- namespace constant,
- all value constants for all key families (including payload/run),
- turns-owned typed key declarations,
- block kind enum and string/YAML methods.

Because no backward compatibility is required, we can consolidate current `keys.go`, `keys_gen.go`, and possibly `block_kind_gen.go` into one generated file.

### Engine Go emitter

Generate `pkg/inference/engine/turnkeys_gen.go` from typed key defs with `owner: engine`.

This removes manual synchronization between engine key vars and turns constants.

### Turns TS emitter

Generate turns declaration constants for all key families and block kinds, including run/payload families if desired (or a filtered export list).

### JS consts Go emitter

Generate `gp.consts` groups from `js_exports.const_groups` exactly.

Because no backward compatibility is required, we should remove `MetadataKeys` alias and keep explicit names only (`TurnMetadataKeys`, etc.).

### Geppetto TS emitter

Generate const groups portion from same `js_exports.const_groups` and merge into main `geppetto.d.ts` template sections.

### JS codec map emitter (strongly recommended)

Generate `turnDataShortToID`, `turnMetaShortToID`, `blockMetaShortToID` (and reverse maps) from IR into a generated file.

This removes manual map drift in `pkg/js/modules/geppetto/codec.go`.

## 5) Validation model

Strict validation should fail generation early on:

1. Duplicate values within family.
2. Duplicate const names.
3. Duplicate typed key names across packages.
4. Invalid ownership (`engine` typed key with missing `type_expr`).
5. js export source references missing family/enum.
6. js export group name collisions.
7. invalid JS key identifier derivation if explicit key is provided.
8. output file targets collide.

Also generate a “manifest hash” comment into outputs for debugging provenance.

## 6) Naming and consistency policy (breaking allowed)

Because backward compatibility is explicitly not required:

1. Eliminate ambiguous names:
- keep `TurnMetadataKeys`
- remove alias-only `MetadataKeys`

2. Use consistent family names everywhere:
- Go: `*ValueKey` constants,
- JS const groups: explicit `<Family>Keys` names,
- TS: matching names.

3. Generate all key-value constants from one manifest, including payload/run families.

## 7) Migration plan (concrete)

## Phase A: Introduce unified generator in parallel

- Add `cmd/gen-meta` + tests + templates.
- Keep current generators temporarily.
- Generate to temporary files (`*_gen_next.go`, `*.next.d.ts`) and diff.

## Phase B: Switch consumers

- Update `go:generate` in:
  - `pkg/turns/generate.go`
  - `pkg/js/modules/geppetto/generate.go`
  - add generation call for engine turnkeys.
- Point all outputs to canonical files.

## Phase C: Remove legacy generators and schemas

- Delete:
  - `cmd/gen-turns`
  - `cmd/gen-js-api`
  - `pkg/turns/spec/turns_codegen.yaml`
  - `pkg/js/modules/geppetto/spec/js_api_codegen.yaml`
- Replace with single manifest path.

## Phase D: Cleanup manual legacy files

- Delete or minimize manual `pkg/turns/keys.go`.
- Remove manual engine `turnkeys.go` if fully generated.
- Move manual codec key maps to generated file.

## Phase E: Enforce in CI

- Add `make generate` target and CI check:
  - run generator,
  - fail if git diff is non-empty.

## 8) Testing strategy

## Unit tests for parser/IR

- schema parse,
- ownership rules,
- group resolution,
- JS key transformation.

## Golden tests per emitter

- turns_go,
- engine_go,
- js_consts_go,
- turns_dts,
- geppetto_dts,
- codec maps.

Golden tests should compare exact output text with fixtures.

## Integration tests

- `go generate ./...` from clean tree,
- `go test ./...`,
- compile check for all generated targets.

## 9) Risk analysis

### Risk: one manifest becomes too large/hard to edit

Mitigation:

- support `include` files in schema (`families/*.yaml`, `js_enums/*.yaml`) merged by loader,
- keep canonical merged output for reproducibility.

### Risk: template complexity explosion

Mitigation:

- keep emitters thin and separate templates by target,
- shared helper functions for identifier/value transforms.

### Risk: import-cycle regressions with generated engine keys

Mitigation:

- ownership validation + compile tests,
- explicit package-specific emitter for typed key sections.

### Risk: accidental runtime break when removing aliases

Mitigation:

- this is acceptable per no-backward-compat policy, but still document rename in release notes and JS reference docs.

## 10) Optional extension: typed `.d.ts` fragments from Go types

A future phase can add `go/packages`-based extraction for struct-shaped API fragments (contexts/options payloads) to reduce manual TypeScript maintenance.

This should remain an additional emitter over IR, not a replacement for schema-driven dynamic API export generation.

## 11) Recommended decisions

1. Approve move to one generator command.
2. Approve single manifest with all key families and JS enums.
3. Approve generation of `run_meta` and `payload_keys` families (remove manual constants file responsibility).
4. Approve generating engine turnkeys and JS codec short-key maps from manifest.
5. Approve explicit JS group names only (drop ambiguous aliases).

## 12) Immediate next implementation tasks (derived)

1. Scaffold `cmd/gen-meta` with parser + IR + one emitter (turns_go) to prove architecture.
2. Port current turns outputs to `gen-meta` and lock golden tests.
3. Port js const generation + geppetto d.ts const sections.
4. Add engine turnkeys emitter.
5. Add codec map emitter.
6. Cut over `go:generate` and delete legacy generators/schemas.

## Appendix A: files directly implicated

- `cmd/gen-turns/main.go`
- `cmd/gen-js-api/main.go`
- `pkg/turns/spec/turns_codegen.yaml`
- `pkg/js/modules/geppetto/spec/js_api_codegen.yaml`
- `pkg/turns/keys.go`
- `pkg/turns/keys_gen.go`
- `pkg/inference/engine/turnkeys.go`
- `pkg/js/modules/geppetto/consts_gen.go`
- `pkg/doc/types/turns.d.ts`
- `pkg/doc/types/geppetto.d.ts`
- `pkg/js/modules/geppetto/codec.go`
- `pkg/turns/spec/README.md`

## Appendix B: experiment artifact

- `ttmp/2026/02/20/GP-16-UNIFY-TURNS-KEYS-CODEGEN-MERGE--unify-turns-key-constants-codegen-after-gp-02-merge-conflict/scripts/analyze_codegen_overlap.go`
