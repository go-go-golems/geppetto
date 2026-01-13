---
Title: 'Report: prototype typed key Decode via JSON re-marshal'
Ticket: 005-TURNS-SERDE
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - turns
    - inference
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/turns/key_families.go
      Note: Prototype Decode(raw)->T via JSON re-marshal
    - Path: geppetto/pkg/turns/serde/key_decode_regression_test.go
      Note: Regression tests for YAML fixture decoding
    - Path: geppetto/pkg/turns/serde/serde_test.go
      Note: Updated to reflect new decode-on-Get behavior
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-05T20:03:03.261947337-05:00
WhatFor: ""
WhenToUse: ""
---


# Report: prototype typed key Decode via JSON re-marshal

## Goal

Prototype a minimal, wide-reach fix for YAML-sourced turns where structured values under `turns.Data` / `turns.Metadata` / `turns.BlockMetadata` decode as `any` (maps/slices), causing typed key reads to fail. The approach is “best-effort typed decode” using `new(T)` plus JSON re-marshal (`json.Marshal(raw)` then `json.Unmarshal(..., *T)`).

## Motivation / Trigger

`pkg/steps/ai/openai/engine_openai.go` currently does:

- `engine.KeyToolConfig.Get(t.Data)`

If a turn was loaded from YAML (fixtures via `turns/serde.FromYAML`), `turns.Data.UnmarshalYAML` stores `geppetto.tool_config@v1` as a `map[string]any`. Previously, a mismatch could be ignored; now typed keys treat “present but wrong type” as an error and inference aborts.

## Prototype summary

Implemented a `Decode(raw any) (T, error)` method on:

- `turns.DataKey[T]`
- `turns.TurnMetaKey[T]`
- `turns.BlockMetaKey[T]`

and changed `Get` to call `Decode` instead of a direct `value.(T)` assertion.

Decode behavior:

1. If `raw` is already type `T`, return it.
2. Otherwise marshal `raw` to JSON and unmarshal into `new(T)`, then return `*ptr`.

This supports:

- `map[string]any` → structs with `json:"..."` tags (e.g. `engine.ToolConfig`)
- `[]any` → `[]string` (when all elements are strings)
- numeric coercions via JSON (within the limitations of `encoding/json`)

## What I changed

- `geppetto/pkg/turns/key_families.go`
  - added `decodeViaJSON[T]`
  - added `Decode` methods for all three key families
  - updated `Get` methods to use the decode path

## Tests / Validation

Added regression tests that load YAML with structured `data` values and verify typed reads succeed:

- `geppetto/pkg/turns/serde/key_decode_regression_test.go`
  - confirms `engine.KeyToolConfig.Get(turn.Data)` succeeds when YAML contains `geppetto.tool_config@v1` as a map
  - confirms `turns.KeyAgentModeAllowedTools.Get(turn.Data)` succeeds when YAML contains a list of strings
  - confirms uncoercible list elements still produce an error

Also updated an existing round-trip test to reflect the new behavior:

- `geppetto/pkg/turns/serde/serde_test.go`

Validation command:

- `cd geppetto && go test ./... -count=1`

## Limitations / Known gaps

- This is intentionally “wide reach”, but it is not perfect:
  - `time.Duration` inside structs (e.g. `engine.ToolConfig.ExecutionTimeout`) will not decode from JSON strings like `"2s"` using `encoding/json` alone; it would require either:
    - custom `UnmarshalJSON` on a wrapper type, or
    - a mapstructure/decode-hook-based approach.
- Because `Get` now attempts decoding, in-memory misuse can be masked:
  - if a caller stores a `map[string]any` for a key that expects a struct, `Get` may succeed instead of failing fast.
  - If that’s unacceptable, we should move decoding into `serde.FromYAML` post-processing or gate the behavior behind an opt-in flag.

## Recommendation / Next steps

- Short term: keep this prototype if the main priority is unblocking YAML fixtures and tool_config usage.
- Medium term: decide whether to:
  - keep decode-in-`Get`, or
  - move the decode into serde (only applied to YAML/JSON ingestion), and restore strict `Get` for in-memory usage.
- If we want user-friendly YAML for durations, evaluate switching from JSON re-marshal to `mapstructure` with:
  - `TagName: "json"` (reuse existing tags)
  - `WeaklyTypedInput: true`
  - decode hooks like `mapstructure.StringToTimeDurationHookFunc()`
