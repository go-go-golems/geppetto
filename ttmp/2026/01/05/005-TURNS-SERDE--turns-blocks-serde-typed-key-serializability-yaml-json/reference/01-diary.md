---
Title: Diary
Ticket: 005-TURNS-SERDE
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - turns
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/inference/engine/types.go
      Note: Custom UnmarshalJSON to parse duration strings in ToolConfig/RetryConfig
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: Diary records tool_config mismatch regression
    - Path: pkg/turns/key_families.go
      Note: Diary records typed key mismatch behavior
    - Path: pkg/turns/serde/key_decode_regression_test.go
      Note: Regression coverage for duration strings in YAML fixtures
    - Path: pkg/turns/serde/serde.go
      Note: Diary tracks serde behavior changes and verification
    - Path: pkg/turns/types.go
      Note: Diary records wrapper UnmarshalYAML behavior
ExternalSources: []
Summary: Implementation diary for investigating and fixing typed-key serializability for Turn/Block data+metadata (YAML/JSON).
LastUpdated: 2026-01-05T19:40:34.295785454-05:00
WhatFor: Record each decision and experiment while making Turn/Block serde compatible with typed keys (including failures and exact commands).
WhenToUse: Update on every meaningful investigation or change; use during review and when continuing work after a pause.
---



# Diary

## Goal

Investigate why YAML-sourced turns/blocks can no longer be decoded into typed-key values (e.g. `engine.ToolConfig`), and propose concrete implementation options to restore a smooth fixture/runner workflow without weakening typed-key safety for normal code.

## Step 1: Create ticket + identify YAML → typed-key mismatch root cause

This step sets up a new docmgr ticket specifically focused on serde. The trigger was a regression reported in `pkg/steps/ai/openai/engine_openai.go`: `engine.KeyToolConfig.Get(t.Data)` now returns a type mismatch error for YAML-loaded turns, causing inference to abort instead of falling back to defaults.

The core issue is that our YAML unmarshal for `turns.Data` / `turns.Metadata` intentionally decodes values into `any` (`map[string]any`, `[]any`, scalars). Typed keys then do a strict Go type assertion (`value.(T)`) and treat mismatches as errors. That is correct for in-memory typed usage, but it breaks YAML workflows unless we add a decode/retyping layer for known keys.

**Commit (code):** N/A — analysis-only (so far)

### What I did
- Created ticket `005-TURNS-SERDE` and added an analysis doc + this diary:
  - `docmgr ticket create-ticket --ticket 005-TURNS-SERDE --title "Turns/Blocks serde: typed key serializability (YAML/JSON)" --topics architecture,geppetto,go,turns,inference`
  - `docmgr doc add --ticket 005-TURNS-SERDE --doc-type analysis --title "Analysis: serializable Turn/Block data+metadata with typed keys"`
  - `docmgr doc add --ticket 005-TURNS-SERDE --doc-type reference --title "Diary"`
- Located the failing access pattern in OpenAI engine:
  - `geppetto/pkg/steps/ai/openai/engine_openai.go` (around the `engine.KeyToolConfig.Get(t.Data)` call)
- Inspected YAML serde path:
  - `geppetto/pkg/turns/serde/serde.go` (`FromYAML` uses `yaml.Unmarshal` into `turns.Turn`)
  - `geppetto/pkg/turns/types.go` (`Data.UnmarshalYAML`, `Metadata.UnmarshalYAML`, `BlockMetadata.UnmarshalYAML` decode into `map[string]any`)
  - `geppetto/pkg/turns/key_families.go` (`DataKey[T].Get` does strict `value.(T)` and returns `(zero, true, error)` on mismatch)
- Ran a small probe to confirm YAML key naming and decoding behavior for `engine.ToolConfig`:
  - `go run /tmp/turns_serde_yaml_decode_probe.go`
  - `go run /tmp/turns_serde_yaml_decode_probe3.go`

### Why
- We need a design that preserves typed-key ergonomics and strictness in Go code, while still allowing fixtures (`cmd/llm-runner/fixtures/*.yaml`) and other YAML workflows to specify structured values (tool config, lists) without manual JSON strings or brittle casing.

### What worked
- Confirmed the immediate mismatch mechanism:
  - YAML `turn.data[geppetto.tool_config@v1]` decodes as `map[string]any`, but `engine.KeyToolConfig` expects `engine.ToolConfig`.
  - YAML lists decode as `[]any`, which will also break keys like `turns.KeyAgentModeAllowedTools` (expects `[]string`).
- Confirmed YAML decoding for `engine.ToolConfig` is sensitive to key naming:
  - `tool_choice: auto` does *not* populate `ToolChoice` without `yaml` tags (probe showed defaults).
  - Keys like `toolchoice: auto` (lowercased, no underscores) *do* decode into `ToolChoice` (probe succeeded).

### What didn't work
- Direct YAML decode using snake_case keys into `engine.ToolConfig` without `yaml` tags:
  - `go run /tmp/turns_serde_yaml_decode_probe.go` decoded only `enabled` and left the rest as zero values.

### What I learned
- The typed key “serializability” check (`json.Marshal` in `Key.Set`) is necessary but not sufficient: YAML input will naturally create `map[string]any` / `[]any` values that are JSON-serializable but not *type-assertable* to the target `T`.
- To keep YAML fixtures ergonomic, we likely need either:
  - a registry-based “retyping” pass in serde (`FromYAML` post-processing), or
  - key-level decoding logic that can coerce from generic YAML shapes into `T` for known keys.

### What was tricky to build
- Understanding the interaction between `yaml.v3` decoding defaults (maps/lists become `any`) and our typed-key design (strict runtime type assertions).

### What warrants a second pair of eyes
- Whether the “right” fix should live in:
  - serde (preferred, keeps typed key `Get` strict), or
  - `Key.Get` (more ergonomic, but risks hiding genuine in-memory misuse).

### What should be done in the future
- Add a minimal reproduction fixture that includes `data.geppetto.tool_config@v1` and `data.geppetto.agent_mode_allowed_tools@v1`, and decide on a canonical YAML schema (likely snake_case) for structured key values.

### Code review instructions
- Start with:
  - `geppetto/pkg/turns/types.go` (YAML unmarshal for `Data`/`Metadata`/`BlockMetadata`)
  - `geppetto/pkg/turns/key_families.go` (`Key.Get` / `Key.Set` semantics)
  - `geppetto/pkg/turns/serde/serde.go` (`FromYAML`)
  - `geppetto/pkg/steps/ai/openai/engine_openai.go` (fatal `KeyToolConfig.Get` usage)
  - `geppetto/pkg/inference/engine/types.go` (shape of `ToolConfig`)
- Validate experiments with:
  - `go run /tmp/turns_serde_yaml_decode_probe.go`
  - `go run /tmp/turns_serde_yaml_decode_probe3.go`

### Technical details
- Probe 1 YAML (snake_case keys) produced mostly zero values:
  - Input: `tool_choice: auto`, `max_iterations: 3`, `execution_timeout: 2s`, ...
  - Output: `Decoded: {Enabled:true ToolChoice: MaxIterations:0 ExecutionTimeout:0s ...}`
- Probe 2 YAML (lowercased concatenated keys) decoded fully:
  - Input: `toolchoice: auto`, `maxiterations: 3`, `executiontimeout: 2s`, ...
  - Output: `Decoded: {Enabled:true ToolChoice:auto MaxIterations:3 ExecutionTimeout:2s ...}`

## Step 2: Brainstorming: where should decoding live, and what does “type safe” mean for YAML?

This step captured the design brainstorming around whether typed keys should expose a decoding surface, and how to reconcile strict typed-key safety with permissive YAML fixtures. The key tension is that YAML decoding naturally produces “generic” representations (`map[string]any`, `[]any`), while typed keys (correctly) want concrete Go types (`T`).

We discussed a few candidate homes for the fix: a serde-only retyping pass (keeps typed keys strict), a registry of per-key codecs (more explicit), or letting the key type expose a decode method itself (more ergonomic). We also explicitly called out that “just YAML-unmarshal into &T{}” won’t work as a generic approach without tags, and that JSON tags are a pragmatic bridge for now because many structs already have `json:"..."`.

**Commit (code):** N/A — discussion-only

### What I did
- Evaluated possible solutions:
  - Add a per-key decode surface (`key.Decode(raw)`), optionally used by `Get`.
  - Use `mapstructure` decoding keyed off `json` tags with decode hooks (duration, weak typing).
  - Use JSON re-marshal/unmarshal as a fast universal coercion step for YAML-derived values.
- Identified a sharp edge that matters for `engine.ToolConfig` specifically:
  - `yaml.v3` does not map `tool_choice` into `ToolChoice` without `yaml` tags.
  - `encoding/json` *does* map `tool_choice` into `ToolChoice` thanks to `json:"tool_choice"` tags.

### Why
- The goal is to restore fixture usability (YAML authoring) without throwing away typed-key safety for normal in-memory usage.

### What worked
- Clear articulation of the core mechanism: YAML “any” values + strict `value.(T)` assertions are incompatible without a decode/retyping layer.

### What didn't work
- The naive “just decode YAML into T everywhere” idea: `&T{}` is not valid for type parameters; plus YAML tags don’t exist on many config structs.

### What I learned
- A key-level `Decode` method can exist and can use `new(T)`; but we still have to choose whether it runs:
  - only in serde ingest (safer), or
  - on every `Get` (more ergonomic, but can mask misuse).

### What was tricky to build
- Understanding how different decoders behave:
  - YAML decoding uses field-name heuristics, not `json` tags.
  - JSON decoding uses `json` tags, but won’t handle `time.Duration` strings without custom logic.

### What warrants a second pair of eyes
- Whether allowing decode-on-`Get` is an acceptable tradeoff (masking in-memory mis-typed values).

### What should be done in the future
- Decide on a long-term schema for structured values (especially durations), and whether to use mapstructure+hooks rather than JSON re-marshal.

### Code review instructions
- Review the options analysis in:
  - `geppetto/ttmp/2026/01/05/005-TURNS-SERDE--turns-blocks-serde-typed-key-serializability-yaml-json/analysis/01-analysis-serializable-turn-block-data-metadata-with-typed-keys.md`

## Step 3: Prototype: `new(T)` + JSON re-marshal decode for typed keys

This step implemented the “wide reach” prototype we discussed: when a typed key reads a value that is present but not already of type `T`, it attempts to decode it by JSON-marshaling the raw `any` value and JSON-unmarshaling into `new(T)`. This is specifically aimed at YAML-derived values where structs become `map[string]any` and string slices become `[]any`.

The result is that YAML fixtures can once again specify structured key values (e.g. `geppetto.tool_config@v1` as a YAML map) and typed reads can succeed without needing a separate codec registry in the short term.

**Commit (code):** N/A — not committed yet in this step

### What I did
- Implemented `Decode(raw any) (T, error)` for all key families and used it in `Get`:
  - `geppetto/pkg/turns/key_families.go`
- Added regression coverage for YAML fixtures:
  - `geppetto/pkg/turns/serde/key_decode_regression_test.go`
- Updated the existing serde round-trip test to reflect the new decode behavior:
  - `geppetto/pkg/turns/serde/serde_test.go`
- Validated:
  - `cd geppetto && go test ./... -count=1`

### Why
- This directly addresses the reported regression: `engine.KeyToolConfig.Get(t.Data)` should not hard-fail for turns loaded from YAML when the YAML encodes `geppetto.tool_config@v1` as a map.

### What worked
- YAML containing:
  - `geppetto.tool_config@v1: {enabled: true, tool_choice: required, max_parallel_tools: 2}`
  - `geppetto.agent_mode_allowed_tools@v1: [search, calc]`
  now decodes successfully through typed keys during tests.

### What didn't work
- Known limitation (not solved by this prototype): types with `time.Duration` fields will not decode from JSON strings like `"2s"` unless the type provides custom JSON unmarshaling or we switch to a decode-hook approach (e.g. mapstructure).

### What I learned
- `new(T)` makes this approach work for a wide range of `T`, including pointers, slices, and structs, because `encoding/json` can unmarshal into `*T` (or `**U` when `T` itself is a pointer).

### What was tricky to build
- Updating existing tests that previously asserted strict type mismatch behavior for YAML round-trips; the semantics of typed key `Get` changed by design in this prototype.

### What warrants a second pair of eyes
- Confirm we want decode-on-`Get` globally (it can mask in-memory misuse), vs limiting this behavior to serde ingestion only.

### What should be done in the future
- Decide whether to:
  - keep this behavior, or
  - move decoding into `serde.FromYAML` post-processing and restore strict `Get`.
- If we keep YAML as the authoring format for structured values, decide how to represent and decode durations cleanly.

### Code review instructions
- Start with:
  - `geppetto/pkg/turns/key_families.go`
  - `geppetto/pkg/turns/serde/key_decode_regression_test.go`
  - `geppetto/pkg/turns/serde/serde_test.go`
  - `geppetto/ttmp/2026/01/05/005-TURNS-SERDE--turns-blocks-serde-typed-key-serializability-yaml-json/analysis/02-report-prototype-typed-key-decode-via-json-re-marshal.md`
- Validate:
  - `cd geppetto && go test ./... -count=1`

## Step 4: Add custom JSON unmarshalling for duration fields in `engine.ToolConfig`

After landing the JSON re-marshal prototype, the next obvious sharp edge is durations: YAML fixtures naturally express durations as strings (e.g. `execution_timeout: 2s`), but `encoding/json` cannot unmarshal JSON strings into `time.Duration` (it expects a number of nanoseconds). That means the prototype would still fail for `engine.ToolConfig.ExecutionTimeout` and `engine.RetryConfig.BackoffBase` when fixtures use human-friendly duration strings.

This step adds `UnmarshalJSON` implementations for `engine.ToolConfig` and `engine.RetryConfig` that accept either:

- string durations (`"2s"`, `"100ms"`) via `time.ParseDuration`
- numeric durations (treated as `time.Duration` nanoseconds, preserving the default JSON behavior)

**Commit (code):** N/A — not committed yet in this step

### What I did
- Added custom JSON unmarshalling:
  - `geppetto/pkg/inference/engine/types.go`
    - `ToolConfig.UnmarshalJSON`
    - `RetryConfig.UnmarshalJSON`
    - shared helper `unmarshalJSONDuration`
- Extended YAML regression coverage to include duration strings:
  - `geppetto/pkg/turns/serde/key_decode_regression_test.go`
- Validated:
  - `cd geppetto && go test ./... -count=1`

### Why
- This makes YAML fixtures with human-readable durations compatible with the typed-key `Decode` path without requiring mapstructure or per-key codec registries yet.

### What worked
- `engine.KeyToolConfig.Get(turn.Data)` now successfully reads YAML-sourced `execution_timeout: 2s` and `retry_config.backoff_base: 100ms`.

### What didn't work
- N/A

### What I learned
- Custom `UnmarshalJSON` on the struct is a clean way to “teach” the JSON re-marshal prototype about YAML-friendly representations without changing the typed key layer.

### What was tricky to build
- Keeping compatibility with both representations:
  - YAML fixtures want string durations.
  - Existing JSON encoding of `time.Duration` uses numbers (nanoseconds).

### What warrants a second pair of eyes
- Confirm we’re happy treating numeric values as nanoseconds (matching `time.Duration`), and that accepting float64 inputs is not too permissive.

### What should be done in the future
- Consider adding explicit YAML tags or a mapstructure-based decoder if we want YAML output (ToYAML) to use snake_case fields rather than the current default-lowercased field names for structs.

### Code review instructions
- Start with:
  - `geppetto/pkg/inference/engine/types.go`
  - `geppetto/pkg/turns/serde/key_decode_regression_test.go`
- Validate:
  - `cd geppetto && go test ./... -count=1`
