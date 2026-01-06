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
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: Diary records tool_config mismatch regression
    - Path: geppetto/pkg/turns/key_families.go
      Note: Diary records typed key mismatch behavior
    - Path: geppetto/pkg/turns/serde/serde.go
      Note: Diary tracks serde behavior changes and verification
    - Path: geppetto/pkg/turns/types.go
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
