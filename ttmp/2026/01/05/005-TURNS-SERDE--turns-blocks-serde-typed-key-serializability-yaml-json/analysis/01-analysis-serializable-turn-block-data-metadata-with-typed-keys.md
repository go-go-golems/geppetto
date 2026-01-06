---
Title: 'Analysis: serializable Turn/Block data+metadata with typed keys'
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
    - Path: geppetto/cmd/llm-runner/fixtures/simple.yaml
      Note: Representative YAML fixture; currently no data section
    - Path: geppetto/pkg/inference/engine/turnkeys.go
      Note: Defines KeyToolConfig typed key
    - Path: geppetto/pkg/inference/engine/types.go
      Note: ToolConfig schema; candidate for yaml tags
    - Path: geppetto/pkg/inference/fixtures/fixtures.go
      Note: Loads fixtures and uses serde.FromYAML
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: Current failure surface; KeyToolConfig.Get aborts on YAML type mismatch
    - Path: geppetto/pkg/turns/key_families.go
      Note: Typed key Get/Set contract; type mismatch becomes error
    - Path: geppetto/pkg/turns/serde/serde.go
      Note: YAML entrypoint used by cmd/llm-runner and fixtures
    - Path: geppetto/pkg/turns/types.go
      Note: Wrapper store YAML marshal/unmarshal; values decode as any
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-05T19:40:34.153705597-05:00
WhatFor: Analyze how typed Turn/Block keys interact with YAML/JSON serialization, why YAML decode currently produces type mismatches, and enumerate concrete implementation options (with tradeoffs) to restore fixture usability without weakening type safety.
WhenToUse: Use before implementing any serde changes; use during design review to pick an approach and to define correctness/compatibility expectations for fixtures and offline workflows.
---


# Analysis: serializable Turn/Block data+metadata with typed keys

## Problem statement

We recently migrated Turn/Block data access to store-specific typed key families:

- `turns.Data` is an opaque wrapper holding a `map[turns.TurnDataKey]any`
- `turns.Metadata` is an opaque wrapper holding a `map[turns.TurnMetadataKey]any`
- `turns.BlockMetadata` is an opaque wrapper holding a `map[turns.BlockMetadataKey]any`
- typed keys (`turns.DataKey[T]`, `turns.TurnMetaKey[T]`, `turns.BlockMetaKey[T]`) use strict runtime type assertion (`value.(T)`) and treat mismatches as errors.

This strictness is desirable for in-memory Go usage, because it catches accidental misuse of keys across stores and prevents “silent” bad types from propagating.

However, YAML deserialization of `turns.Turn` currently decodes `data`/`metadata` values into generic `any` representations (`map[string]any`, `[]any`, scalars). This makes YAML-sourced turns incompatible with typed-key reads for any non-scalar key types (structs, slices, maps), which breaks workflows like `cmd/llm-runner` fixtures.

Concrete regression: `pkg/steps/ai/openai/engine_openai.go` now fails hard on YAML-sourced `geppetto.tool_config@v1` because `engine.KeyToolConfig.Get(t.Data)` sees a `map[string]any` instead of an `engine.ToolConfig` and returns `(ok=true, err!=nil)`.

## Current behavior (why it fails)

### YAML serde path

- `pkg/turns/serde/serde.go:FromYAML` does `yaml.Unmarshal(b, &turns.Turn{})`
- `pkg/turns/types.go` provides `UnmarshalYAML` for `turns.Data` / `turns.Metadata` / `turns.BlockMetadata`
  - they decode into `map[string]any`
  - then convert the keys to `TurnDataKey`/`TurnMetadataKey`/`BlockMetadataKey` and store the values “as-is”

### Typed key read path

- `pkg/turns/key_families.go:DataKey[T].Get`:
  - missing key => `(zero, false, nil)`
  - present key but wrong type => `(zero, true, error)`

This is the key interaction that makes YAML fixtures brittle: YAML decode produces “present key but wrong Go type” for most structured values.

## Reproduction snippets

### 1) ToolConfig in YAML becomes `map[string]any`

Minimal YAML turn:

```yaml
id: t
blocks: []
data:
  geppetto.tool_config@v1:
    enabled: true
    tool_choice: auto
```

After `serde.FromYAML`, the raw stored value for the key is a `map[string]any` (not `engine.ToolConfig`). So `engine.KeyToolConfig.Get(t.Data)` returns `(ok=true, err=type mismatch)` and `engine_openai.go` aborts.

### 2) `[]string` in YAML becomes `[]any`

Minimal YAML turn:

```yaml
id: t
blocks: []
data:
  geppetto.agent_mode_allowed_tools@v1: [search, calc]
```

After `serde.FromYAML`, the raw stored value is `[]any{"search","calc"}`. Typed keys like `turns.KeyAgentModeAllowedTools` expect `[]string`, so `Get` fails with a type mismatch.

## Constraints and goals

### Goals

- Keep typed-key strictness for in-memory Go usage (type mismatch should remain an error in normal code paths).
- Make YAML/JSON-based workflows usable again:
  - fixtures should be hand-writable
  - fixtures should prefer snake_case (consistent with existing turn YAML docs)
  - structured types (structs, slices, maps) should decode into their typed Go forms when possible
- Avoid adding a backwards-compatibility layer for the deleted legacy turns API.

### Non-goals

- We do not need full schema enforcement at YAML load time for *all* keys; unknown keys can remain `any`.
- We do not need to support arbitrary Go types; the existing `Key.Set` already constrains values to be JSON-serializable.

## Options

### Option A: Make the OpenAI engine tolerate mismatches (quick patch)

Change `pkg/steps/ai/openai/engine_openai.go` to treat a type mismatch for `engine.KeyToolConfig.Get(t.Data)` as non-fatal (log a warning and continue with defaults).

Pros:
- Fastest way to restore `cmd/llm-runner` fixtures.
- Localized change.

Cons:
- Doesn’t address the general problem (other keys will still be broken).
- Risks papering over genuine in-memory misuse (if a caller sets the wrong type programmatically, inference may silently ignore it).

### Option B: Add `yaml` tags to structured config types (necessary but not sufficient)

Add `yaml:"tool_choice"` etc tags to `engine.ToolConfig` (and nested types) so snake_case YAML can decode into the struct.

Pros:
- Makes YAML encoding/decoding for `ToolConfig` consistent and ergonomic.
- Improves `ToYAML` output readability for any Turn that already holds a typed `ToolConfig`.

Cons:
- By itself it does not fix the typed-key mismatch, because `turns.Data.UnmarshalYAML` currently decodes values into `any` instead of decoding to the expected struct type.

### Option C: Add a serde “retyping” pass via a registry of per-key decoders (recommended)

Add a registry that maps key ids (e.g. `"geppetto.tool_config@v1"`) to decode/coercion logic. `serde.FromYAML` would:

1. `yaml.Unmarshal` into `turns.Turn` (keeping current permissive `any` decode)
2. Normalize defaults (`serde.NormalizeTurn`)
3. Apply the registry to retype known keys:
   - if a known key value is `map[string]any`, decode to its typed struct and replace the stored value
   - if a known key value is `[]any`, coerce to `[]string` (etc)

This keeps typed key `Get` strict (it will still fail if no retyping happened), and it centralizes YAML-specific behavior in serde.

Pros:
- Preserves strictness for normal in-memory code.
- Restores YAML workflows without requiring YAML to embed JSON strings.
- Scales: we can incrementally add codecs for keys that need them.
- Avoids introducing “best-effort” behavior into every typed key `Get`.

Cons:
- Requires designing and maintaining a codec registry API (including init-order considerations).
- Requires explicit coverage: if a key is not registered, it will remain `any` and typed `Get` will still fail.

Implementation sketch (non-binding):
- In `turns` package:
  - expose registry registration functions like:
    - `RegisterDataDecoder(id turns.TurnDataKey, fn func(raw any) (any, bool, error))`
    - `RegisterTurnMetaDecoder(...)`
    - `RegisterBlockMetaDecoder(...)`
  - provide a `RetypeKnownKeys(t *turns.Turn)` helper that mutates the wrappers.
- In `turns/serde`:
  - call `turns.RetypeKnownKeys(&t)` from `FromYAML`.
- In `engine` (and other packages that own key types):
  - register decoders for their keys in `init()`.

### Option D: Make typed keys “smart” (decode-on-Get)

Modify `DataKey[T].Get` (and friends) to attempt decoding/coercion when `value.(T)` fails. For example, if the raw value is a `map[string]any`, JSON-marshal it and JSON-unmarshal into `T`, or mapstructure-decode with tag support.

Pros:
- Most ergonomic for callers; “it just works” regardless of source.
- No need to ensure `serde.FromYAML` ran a post-pass.

Cons:
- Blurs the contract of typed keys and can hide misuse in normal code paths.
- Hard to do generically and correctly for all `T` (durations/enums, nested maps, numeric conversions, `[]any` → `[]string`, etc).
- Potentially expensive and surprising at runtime.

## Recommendation

Pursue Option C (serde post-process + registry), and pair it with Option B for `engine.ToolConfig` so fixtures can use snake_case keys and retyping can decode into the struct reliably.

As a short-term stabilization measure, Option A can be used to unblock CI/fixtures while the serde registry work is implemented and tested.

## Open questions

- Should the registry live in `turns` (to avoid import cycles and allow other packages to register decoders), or in `turns/serde` (simpler but harder to extend across packages)?
- Do we want YAML to be the “authoring format” (snake_case everywhere), with JSON as a secondary format, or the opposite?
- Should unknown keys remain as `any` always, or should we optionally preserve raw YAML nodes to allow deferred decoding?

## Related files (starting points)

- `geppetto/pkg/turns/types.go` — YAML marshal/unmarshal for wrapper stores
- `geppetto/pkg/turns/key_families.go` — typed key `Get`/`Set` contract
- `geppetto/pkg/turns/serde/serde.go` — YAML entrypoint used by fixtures and runner
- `geppetto/pkg/steps/ai/openai/engine_openai.go` — current failure surface (`KeyToolConfig.Get`)
- `geppetto/pkg/inference/engine/types.go` — `ToolConfig` schema (candidate for `yaml` tags)
- `geppetto/pkg/inference/engine/turnkeys.go` — typed key definition for tool config
