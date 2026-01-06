---
Title: 'Design: registry-based typed decoding for Turn.Data/Metadata (codec per key)'
Ticket: 003-REVISE-DATA-ACCESS-API
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/engine/turnkeys.go
      Note: |-
        Example of a key whose value type lives in engine (import-cycle constraints for where to register codecs).
        Engine-owned key illustrating where codec registration must live
    - Path: geppetto/pkg/turns/serde/serde.go
      Note: |-
        YAML round-trip path that produces map[string]any for structured values.
        YAML round-trip decode path producing map[string]any
    - Path: geppetto/pkg/turns/types.go
      Note: |-
        Current DataGet/DataSet + YAML decode behavior (map[string]any) causes type mismatch for structured values on round-trip.
        Current strict-assertion DataGet behavior that fails after YAML round-trip for structured values
    - Path: geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md
      Note: Final design baseline; this doc proposes an extension for typed decoding after YAML/JSON round-trips.
ExternalSources: []
Summary: Introduce an optional codec registry keyed by turn key identity to reconstruct typed values (e.g., ToolConfig) from YAML-decoded map[string]any, avoiding implicit global 'decode everything' behavior.
LastUpdated: 2025-12-22T00:00:00-05:00
WhatFor: Fix typed round-trip correctness for structured values stored in Turn.Data/Metadata without weakening type safety or introducing silent conversions.
WhenToUse: Use if we need typed values (structs) to survive YAML/JSON round-trip and be retrievable via typed key APIs without manual casts at call sites.
---


# Design: registry-based typed decoding for Turn.Data/Metadata (codec per key)

## Executive Summary

Today, `Turn.Data` stores `any` values and YAML unmarshalling decodes unknown structured values into `map[string]any`. As a result, a typed read like `DataGet[T](..., Key[T])` can fail after round-trip when the stored value was a struct (e.g., `engine.ToolConfig`) because the stored value is now a map.

It is tempting to make `DataGet[T]` “smart” and decode maps into `T` based solely on the type parameter `T`. However, that creates implicit behavior, performance cost, and correctness ambiguity.

This design proposes an explicit, **opt-in registry of codecs per key** that allows the system to reconstruct typed values when the in-memory value is in a generic decoded form (map/slice of interface values) after YAML/JSON round-trips.

## Problem Statement

We want:

- typed keys with `Get` returning `(T, bool, error)`
- values stored as `any` for runtime performance during normal execution
- values to be JSON/YAML serializable
- **typed round-trip** for selected structured values (at least the ones we care about in tests and persistence), e.g.:
  - `engine.ToolConfig`

Current behavior:

- `DataSet(&t.Data, engine.KeyToolConfig, engine.ToolConfig{...})` stores a concrete struct
- YAML marshal prints a mapping for that struct
- YAML unmarshal decodes it into `map[string]any`
- `DataGet(t.Data, engine.KeyToolConfig)` attempts `value.(engine.ToolConfig)` and fails

## Why “just decode based on T” is not enough

In Go, `DataGet[T]` can obtain runtime type information for `T` using reflection (e.g. `reflect.TypeFor[T]()` / `reflect.TypeOf((*T)(nil)).Elem()`).

So in principle, `DataGet[T]` could do:

1. If `value` is already `T`, return it.
2. Else if `value` is `map[string]any` and `T` is a struct (or pointer-to-struct), attempt `json.Marshal(value)` then `json.Unmarshal(..., &t)`.

But doing this implicitly has major drawbacks:

- **Ambiguity / correctness**:
  - JSON/YAML decoding loses information (e.g., numeric types often become `int`/`float64`, time/duration types become strings, etc.).
  - Interfaces and `any` values cannot be reconstructed without additional type hints.
  - “Decoding into T” may succeed while still being semantically wrong (silent coercions).
- **Performance**:
  - Marshalling/unmarshalling on every `Get` is expensive (allocations, reflection).
  - `Get` is hot-path in middleware; this would regress performance broadly.
- **Surprise behavior**:
  - Callers would no longer have a predictable “store type == get type” contract.
  - Bugs can be masked (a wrong-shaped map that “sort of” decodes).
- **Security / footguns**:
  - Turning arbitrary maps into structs implicitly can become an unexpected deserialization surface.

Therefore, if we add typed decoding, it should be **explicit, per-key**, and ideally only engaged when needed.

## Proposed Solution: Key Type Registry (schema per key identity)

### Concept

Introduce an optional registry that is keyed by turn key identity and can answer:

- **What is the expected Go type for this key?**
- **How do we decode/encode this key’s value across serialization boundaries?** (optional)
- **How do we validate serializability for this key?** (optional)

This registry can then be used in *three* places:

1. **Typed reads**: when `Get[T]` sees a mismatch (e.g., `map[string]any`), consult registry and decode into the expected type.
2. **YAML import**: when unmarshalling YAML into a Turn, consult registry to decode values directly into their expected Go type (so future typed `Get` succeeds).
3. **Write-time validation**: on `Set`, validate:
   - value is assignable to the expected type (if registered)
   - value is serializable (JSON/YAML), using either generic marshal checks or a registry-provided validation function.

### Registry entry interface

We keep it minimal and explicit:

```go
// Entry is per-key schema/codec metadata.
type Entry interface {
    // ExpectedType returns the concrete Go type expected for this key.
    // Used for type-checking and for decode targets.
    ExpectedType() reflect.Type

    // DecodeFromYAML takes a YAML-decoded representation and returns a typed Go value.
    // Typical input forms: map[string]any, []any, string, float64, bool, nil.
    // Returning (nil, false, nil) means "not applicable" (e.g., already typed).
    DecodeFromYAML(decoded any) (any, bool, error)

    // ValidateOnSet validates that the provided value is acceptable.
    // This can include type checks and serializability checks.
    ValidateOnSet(value any) error
}
```

We then bind codecs to specific key ID types (Data/TurnMeta/BlockMeta) or to an underlying canonical string ID.

### API sketch (Data only shown)

```go
// RegisterDataType associates schema/codec info with a Turn.Data key.
func RegisterDataType(key TurnDataKey, entry Entry)

// DataGet[T] uses:
// 1) direct type assertion to T (fast path)
// 2) if mismatch: consult registry for key.id and attempt DecodeFromYAML(value)
// 3) if DecodeFromYAML returns a value of type T: return it (optionally cache back into Data)
// 4) otherwise: return type mismatch error
```

### Using the registry for YAML import (turn wrapper UnmarshalYAML)

When `Data.UnmarshalYAML` (and `Metadata.UnmarshalYAML`, `BlockMetadata.UnmarshalYAML`) receives a decoded `map[string]any`, it can:

1. Parse the key string into the appropriate key ID type (`TurnDataKey`, etc.).
2. Look up a registry entry for that key ID.
3. If found, attempt to decode the decoded value into the expected Go type.
4. Store the typed value into the wrapper map.

This makes YAML import behave as “typed reconstruction at the edge”, which avoids doing repeated decode work in `Get` during execution.

**Import-cycle note:** the registry must allow registrations from packages that own the types (e.g., `engine` for `engine.ToolConfig`), because `turns` cannot import `engine`.

### Using the registry for YAML serializability validation (Set-time)

If a key is registered, `Set` can validate:

- **Type correctness**: `reflect.TypeOf(value)` is assignable to `ExpectedType()`
- **Serializability**: either
  - generic `json.Marshal` + optionally `yaml.Marshal` checks, or
  - entry-specific `ValidateOnSet` logic if we want to constrain YAML compatibility (e.g., “JSON-shaped YAML only”).

This turns “serializability” from a best-effort check into a *schema-aware* invariant for keys that opt in.

### Registration and import cycles

Key ownership and import cycles matter:

- `engine.KeyToolConfig` lives in `engine` (because the value type is `engine.ToolConfig`)
- `turns` must not import `engine`

Therefore:

- The codec registration for `ToolConfig` must happen in the `engine` package (or a package that can import both).

We have two viable patterns:

#### Pattern A: init-time registration (simple, but relies on init)

In `engine`:

```go
func init() {
    turns.RegisterDataType(turns.NewTurnDataKey(...), toolConfigEntry{})
}
```

Pros: easiest.
Cons: hidden side effects, order-of-init concerns in tests.

#### Pattern B: explicit wiring in application bootstrap (preferred for visibility)

Expose a function in `engine`:

```go
func RegisterTurnCodecs() {
    turns.RegisterDataType(KeyToolConfigID, toolConfigEntry{})
}
```

Then call it from:

- CLI main(s)
- tests that rely on typed round-trip

Pros: explicit.
Cons: requires remembering to call it.

We can combine with a helper in `turns/serde` tests to keep behavior stable.

### Default entry helper (optional)

Provide a standard codec implementation that uses JSON round-tripping:

```go
// JSONRoundTripEntry is a convenience entry for "decoded form -> JSON -> typed struct".
// It is explicit and per-key, but uses json marshal/unmarshal to reconstruct the type.
type JSONRoundTripEntry[T any] struct{}
```

The core idea remains: keep reconstruction **explicit** and **per-key**, even if the implementation uses JSON internally.

This keeps codecs easy to define, but still explicit and per-key.

## Interaction with “3 key families” API

If we adopt `DataKey[T].Get(...)`, the decoding logic should live behind that method (or behind the underlying `DataGet` equivalent). The registry should still key off the underlying string identity (e.g., `TurnDataKey` string), but we should avoid allowing “cross-store” codecs.

## Design Decisions

### Per-key registry, not “decode based on T everywhere”

We want:

- predictable semantics
- limited performance impact
- explicitness at the edges where decoding matters (round-trip, persistence)

### Decode only on mismatch, and only when value looks “decoded”

We should only attempt codecs when:

- key exists
- direct assertion fails
- value is one of expected “decoded forms” (map/slice/string/float/bool), configurable

## Alternatives Considered

### Decode implicitly in DataGet based on T (no registry)

Rejected for ambiguity + performance + surprise behavior (see above).

### Registry by type T, not by key

Rejected because:

- same type can appear under different keys with different encoding contracts
- `any` and interfaces still ambiguous
- makes it harder to keep import-cycle boundaries clear

### Store JSON blobs (`json.RawMessage`) in Turn.Data

This alternative is the “no registry” route: instead of storing `any`, store JSON bytes and always decode on `Get[T]`.

#### Why it helps

- Round-trip through YAML/JSON becomes naturally typed again because the stored representation is the bytes, not the intermediate decoded map form.

#### Why we still might not want it

- It moves decode cost into the hot path (`Get`).
- It tends to harm YAML readability unless we add extra marshal/unmarshal adapters (decode bytes just to print YAML, then re-encode on read).

**Relationship to this design:** if we choose RawMessage storage, the registry becomes far less necessary (though codecs may still be useful for special cases like custom encodings). This design assumes we keep `any` storage for hot-path performance and add an explicit, narrow mechanism to restore typed reconstruction only when needed.

#### RawMessage + YAML bridge (how to keep YAML human-friendly anyway)

If we *do* choose `json.RawMessage` storage, we can still keep YAML output human-friendly by explicitly bridging:

- **YAML marshal path**: decode JSON bytes → `any` → `yaml.Marshal`
- **YAML unmarshal path**: `yaml.Unmarshal` → JSON-compatible `any` → `json.Marshal` → store bytes

This keeps the *persisted canonical form* as JSON bytes while keeping the *debug snapshot* (YAML) readable.

##### Marshal to YAML (readable snapshot)

Given internal storage:

```go
// Example storage
m map[TurnDataKey]json.RawMessage
```

To produce YAML:

1. Build `map[string]any`:
   - key: `string(TurnDataKey)`
   - value: `json.Unmarshal(raw, &decodedAny)`
2. `yaml.Marshal(map[string]any)` as usual

Notes:

- `json.Unmarshal` yields:
  - `map[string]any` for objects
  - `[]any` for arrays
  - `float64` for numbers
  - `string/bool/nil` for primitives
  These map cleanly into YAML.
- This decode happens only on snapshotting / persistence boundaries, not on hot-path `Get`.

##### Unmarshal from YAML (store canonical JSON bytes)

On YAML input, we must ensure the YAML subset is JSON-compatible (because we will re-encode to JSON):

1. Unmarshal into a `map[string]any` (not `any`), so mapping keys are constrained to strings:

```go
var raw map[string]any
if err := yaml.Unmarshal(yamlBytes, &raw); err != nil { ... }
```

2. For each entry:
   - validate the value is JSON-compatible (recursively):
     - scalars: `nil/bool/string/float64/int` (normalize ints to float64 or leave as-is; `encoding/json` accepts both)
     - sequences: `[]any`
     - mappings: `map[string]any` only (no non-string keys)
   - `json.Marshal(value)` and store into `json.RawMessage`

If we allow arbitrary YAML features (anchors, merge keys, tags, non-string keys), the JSON encoding step becomes ambiguous or lossy. The safest contract is:

- **YAML is accepted as “JSON-shaped YAML” only**, i.e. YAML that can be represented as JSON without loss of meaning.

##### Trade-offs of the YAML bridge

- **Pros**:
  - YAML remains readable even with RawMessage storage.
  - Typed round-trip becomes robust (canonical form is JSON bytes).
- **Cons**:
  - YAML (un)marshal now does extra work:
    - marshal: JSON decode of every stored value
    - unmarshal: JSON encode of every incoming value
  - You must define and enforce the “JSON-shaped YAML” contract.

##### Impact on the registry design

With RawMessage + YAML bridge, the *registry-based typed decoding* is largely unnecessary for structured values because the canonical storage is bytes and `Get[T]` can always unmarshal into `T`.

However, a registry can still make sense for:

- custom encodings (non-JSON, compressed blobs, versioned schema upgrades),
- or “fast path” cached typed values for extremely hot keys (at the cost of caching complexity).

## Implementation Plan

- [ ] Decide whether registry is required for all stores or only Turn.Data (start with Data).
- [ ] Add a small registry to `turns` (map + mutex, or sync.Map).
- [ ] Add a codec interface + default JSONCodec helper.
- [ ] Add explicit registration hook for engine-owned structured values (`ToolConfig`).
- [ ] Update serde tests to call codec registration (if using explicit wiring).
- [ ] Document performance expectations and “do not register codecs for hot keys unless needed”.

## Open Questions

- Do we need to support decoding for Block.Metadata too, or is Data sufficient?
- Should registry be global (package-level) or scoped (passed into serde / app context)?
- Do we want to cache decoded results back into the wrapper map to avoid repeated decode work?
  - (This is attractive but introduces mutation-on-read and concurrency considerations.)


