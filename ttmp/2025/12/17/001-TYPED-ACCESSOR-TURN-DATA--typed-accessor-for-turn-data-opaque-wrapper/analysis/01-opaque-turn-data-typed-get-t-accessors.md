---
Title: 'Opaque Turn.Data: typed Get[T] accessors'
Ticket: 001-TYPED-ACCESSOR-TURN-DATA
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Lint rules that shape the key story today
    - Path: geppetto/pkg/inference/toolcontext/toolcontext.go
      Note: Runtime tool registry lives in context (not Turn.Data)
    - Path: geppetto/pkg/turns/keys.go
      Note: Current canonical TurnDataKey/MetadataKey const keys
    - Path: geppetto/pkg/turns/serde/serde.go
      Note: Serde normalization of nil maps and YAML round-trip
    - Path: geppetto/pkg/turns/types.go
      Note: Current Turn/Block map shapes (Data/Metadata/Payload)
ExternalSources: []
Summary: Reality-check + analysis for evolving Turn.Data and Turn/Block Metadata toward typed/opaque access and structured key identity ({vs,slug,version}), with a serializable-values constraint. Block.Payload is explicitly out-of-scope.
LastUpdated: 2025-12-20T00:27:00-05:00
WhatFor: ""
WhenToUse: ""
---


## Context

`geppetto/pkg/turns.Turn.Data` is currently a plain map:

- Keys are **typed** (`type TurnDataKey string`) and the repo encourages **canonical `const` keys** like `turns.DataKeyToolConfig`.
- Values are **`any`** (`interface{}`), which makes reads/writes flexible but pushes correctness checks into each call site.

This ticket asks: can we keep the “hashmap of arbitrary per-turn attachments” idea, but make access **opaque + typed** with an idiomatic Go `Get[T]` accessor?

Also relevant: `turns.Turn.Metadata` and `turns.Block.Metadata` are the other “bags of attributes” in this model.

**Explicit scope note (updated):** we are **not** changing `turns.Block.Payload` in this ticket. `Block.Payload` remains a flexible `map[string]any` used for block content (text, tool args/results, etc.).

## Current state (what hurts)

### Value-side type safety is manual

Call sites need explicit nil checks and type assertions, e.g.:

- Engines/middleware read `Turn.Data` entries via `any` + type assertions (example: `engine.ToolConfig` under `turns.DataKeyToolConfig`)
- Similar patterns exist for `Turn.Metadata` and `Block.Metadata` where values are attached as `any`

The result is repetitive boilerplate and scattered conventions about “what type is stored under key X?”

### Map initialization boilerplate

Because `Turn.Data` is a map field, write sites need to ensure it’s initialized (nil map assignment panics). Some code already does:

- `if t.Data == nil { t.Data = map[turns.TurnDataKey]any{} }`

`serde.NormalizeTurn` does this for YAML load/store, but plenty of non-serde code still needs to do it.

## Goals

- **Opaque**: hide the raw map behind a small API so we can enforce conventions (initialization, type assertions, iteration).
- **Typed reads**: a `Get[T]` accessor that returns a strongly typed value (plus ok/error).
- **Ergonomic**: avoid making call sites worse (especially around generic type argument verbosity).
- **Serialization compatibility**: preserve the current YAML shape (`data:` as a mapping from string-ish keys to values) and keep `turns/serde` round-trips stable.
- **Support iteration**: some code persists/inspects *all* key/value pairs.

### Hard requirement: keys must encode `{vs, slug, version}`

You want the key identity to always contain **a namespace (“vs”), a slug, and a version**, e.g. `{vs:"mento", slug:"network", version:2}`.

Important Go reality: it’s not possible to make “unset fields” *literally impossible* in all cases because every type has a **zero value** (and callers can always declare `var k T`). What we *can* do, idiomatically, is:

- Make it **impossible for other packages** to construct a key without providing all parts (unexported fields + constructor).
- Make it **fail-fast** if an invalid/zero key is used at runtime (validate in `Data.Set/Get/Range` boundaries).
- Make it **hard to bypass** in the codebase via a linter (reject `var k turns.TurnDataKeyID` and reject calling constructors outside the canonical key definition file).

### Hard requirement: values must be serializable (Turn + Block bags)

You also require that the **types stored** in:

- `Turn.Data`
- `Turn.Metadata`
- `Block.Metadata`

are **serializable**.

For this ticket, “serializable” should mean:

- **Round-trippable** through Geppetto’s primary interchange formats (YAML today, and usually JSON for persistence/transport).
- No runtime-only objects (channels, funcs, open files, DB handles, tool registries as interface objects, etc.) living inside these bags.

**Updated to match current codebase:** Geppetto already removed the “runtime tool registry in `Turn.Data`” pattern. Provider engines learn about tools from a registry carried via `context.Context` (`toolcontext.WithRegistry` / `RegistryFrom`). `Turn.Data` holds serializable per-turn config/hints (e.g. `turns.DataKeyToolConfig`).

Non-goals (for this ticket):

- Designing a full schema system for every key/value (though we can leave room for it).

## Core constraint: Go generics and type inference

If we define:

```go
func (d Data) Get[T any](key turns.TurnDataKey) (T, bool)
```

Go generally **cannot infer `T`** from the assignment target, so call sites must write:

```go
cfg, ok := t.Data.Get[engine.ToolConfig](turns.DataKeyToolConfig)
```

That is workable, but noticeably noisy at scale.

## Design space

### Key modeling choices (independent axis)

There are two largely independent decisions:

1. **How we type values** (raw `any` map vs opaque wrapper vs typed `Get`).
2. **What the key identity is** (today: a typed string; desired: `{vs, slug, version}`).

This ticket’s new requirement forces (2) to change.

#### Key model A (status quo): `type TurnDataKey string`

- **Pros**: can use `const` keys; YAML round-trip is simple; existing linter (`turnsdatalint`) enforces *typed-key expressions* (no raw string literals and no untyped string const identifiers) for typed-key maps.
- **Cons**: cannot guarantee presence of `vs/slug/version` in the type system; you can only enforce it by convention or linting of the string format.

#### Key model B (recommended): a structured comparable key id + text encoding

Define a comparable key ID:

```go
// Turns data key identity: vs + slug + version.
// Fields are unexported so external packages cannot construct it as a literal.
type TurnDataKeyID struct {
	vs      string
	slug    string
	version uint16
}
```

Then store `Turn.Data` as `map[TurnDataKeyID]any` (or behind an opaque wrapper). For YAML, implement `encoding.TextMarshaler` / `encoding.TextUnmarshaler` so the key still serializes as a readable scalar like:

- `mento/network@v2` (or `mento:network@v2`, pick one canonical encoding)

This satisfies the “must include all parts” requirement by construction + validation.

**Key tradeoff:** Go `const` cannot be used for struct values, so “canonical keys” become `var` (or `func` returning a key). That would require evolving the linting story from “typed-key expressions” toward “only canonical key declarations / typed `Key[T]` values”, plus forbidding ad-hoc constructors outside the canonical keys file(s).

### Value modeling choices (serializability axis)

With “must be serializable,” we need to decide what we actually store internally.

#### Value model 1: store typed Go values, validate on `Set` (best-effort)

Keep storing `any` (or `T` in `Set[T]`) but enforce at the boundary:

- `Data.Set(...)` attempts `yaml.Marshal(v)` and/or `json.Marshal(v)` and rejects on error.

This is easy, but it’s not airtight:

- YAML/JSON marshal success depends on dynamic types and can be surprising (interfaces, custom marshalers, time types).
- You still risk “it marshals, but not how we want” across formats.

#### Value model 2 (recommended): store canonical bytes (JSON), decode on `Get`

Store **only serialized bytes** internally (e.g. `json.RawMessage`) and use typed accessors to encode/decode.

This makes the serializability requirement *structural*:

- If `Set` succeeds, the value is serializable by definition (in the chosen canonical format).

To keep YAML human-friendly:

- For YAML marshal: decode each raw JSON value into `any` (via `json.Unmarshal`) and emit it as YAML.
- For YAML unmarshal: decode YAML into `any`, then `json.Marshal` and store raw JSON.

This implies “YAML allowed subset” must be JSON-compatible (scalars, sequences, mappings with string keys).

#### Value model 3: store `yaml.Node` (YAML-native), derive JSON at edges

Store `*yaml.Node` internally and decode on `Get`. This is great for YAML fidelity but makes JSON behavior second-class and is harder to validate/standardize.

Given persistence needs, **JSON-bytes** tends to be the most robust.

### Option 0: helper functions only (non-opaque, minimal change)

Keep `Turn.Data` as a map, add helpers:

```go
func GetData[T any](t *Turn, key TurnDataKey) (T, bool)
func SetData(t *Turn, key TurnDataKey, v any)
```

- **Pros**: minimal refactor; easy incremental adoption.
- **Cons**: doesn’t prevent direct map access, doesn’t centralize iteration semantics, still leans on `Get[T]` type args (or duplicates typed getters per key).

### Option 1: opaque wrapper with `Get[T](TurnDataKey)` (simple opaque)

Change the field to a wrapper:

```go
type Data struct{ m map[TurnDataKey]any }
type Turn struct { Data Data `yaml:"data,omitempty"` }
```

Provide:

```go
func (d *Data) Set(key TurnDataKey, v any)
func (d Data) Get[T any](key TurnDataKey) (T, bool)
func (d Data) Range(fn func(TurnDataKey, any) bool)
```

Plus YAML marshal/unmarshal to preserve `data:` mapping compatibility.

- **Pros**: truly opaque; initialization becomes internal; iteration can be controlled.
- **Cons**: `Get[T]` remains type-arg-heavy, which may discourage adoption.

### Option 2: opaque wrapper + typed keys for inference (recommended)

Introduce a *typed key wrapper* that carries the runtime key string but also a phantom type parameter:

```go
type Key[T any] struct{ id TurnDataKeyID }
func K[T any](id TurnDataKeyID) Key[T] { return Key[T]{id: id} }

// canonical typed keys (vars are fine; TurnDataKeyID cannot be const)
var KeyToolConfig = K[engine.ToolConfig](MustDataKeyID("geppetto", "tool_config", 1))
var KeyAgentMode  = K[string](MustDataKeyID("geppetto", "agent_mode", 1))
```

Now define:

```go
func (d Data) Get[T any](k Key[T]) (T, bool)
```

Call sites become:

```go
cfg, ok := t.Data.Get(turns.KeyToolConfig) // T inferred from key
mode, ok := t.Data.Get(turns.KeyAgentMode)
```

This hits a sweet spot:

- Keeps **typed canonical string keys** (existing convention + linter still useful).
- Adds **typed value access** with **good ergonomics** (no explicit type args).
- Still allows a raw escape hatch for iteration/persistence via `Range`.

### Option 2b (recommended refinement): typed keys + opaque map + serialized values

To satisfy **serializable values**, we should refine the opaque design:

- `Data` stores `json.RawMessage` (canonical) rather than arbitrary `any`.
- `Get[T]` becomes “decode into T” and should surface decode errors.
- `Set[T]` becomes “encode T” and should surface encode errors.

This makes it impossible to stash non-serializable runtime objects in `Turn.Data` (same for metadata if we adopt the same pattern). `Block.Payload` remains out-of-scope for this ticket.

### Option 3: “schema registry” for keys (optional strictness)

In addition to typed `Get`, maintain an internal registry of key → expected type (or a validator function). The wrapper could:

- Reject `Set` with mismatched types (panic in debug, error in prod)
- Improve error messages (“expected engine.ToolConfig, got map[string]any”)

This adds complexity and may be too opinionated for a general-purpose “turn attachment” map, but it can be layered on later.

## Serialization and compatibility considerations

### YAML (current default)

Today, `yaml.v3` happily round-trips:

- `map[TurnDataKey]any` keyed by a defined string type

If we move to an opaque wrapper and/or structured keys, we should preserve the same YAML surface:

- Keep `data:` as a YAML mapping.
- Serialize keys as a readable scalar string (via `MarshalText`/`UnmarshalText` on the key type).
- Implement `Data.MarshalYAML`/`Data.UnmarshalYAML` to marshal/unmarshal the underlying map.

This keeps `turns/serde` stable and avoids breaking existing YAML documents.

### JSON / persistence

Some persistence code iterates over `Turn.Data` and serializes values. Historically this required special-casing runtime-only objects (notably the tool registry when it lived in `Turn.Data`); current Geppetto code carries the tool registry in `context.Context` instead, which is the direction we want to preserve. With an opaque wrapper:

- Provide `Range`/`Keys`/`Len` so persistence can still enumerate entries.
- Consider adding a deliberate escape hatch like `AsMapCopy()` so storage boundaries can snapshot values without sharing the mutable internal map.

With the “serializable values” requirement, we should *not* have runtime-only objects in these maps anymore; instead:

- Store a serializable representation (e.g., tool definitions, tool IDs, or a registry spec).
- Reconstruct runtime registries at execution time from that representation.

## Recommendation

Implement **Option 2b**:

- Make each attribute bag an **opaque wrapper** with the same shape:
  - `Turn.Data`
  - `Turn.Metadata`
  - `Block.Metadata`
- Use **structured key IDs** `{vs, slug, version}` where keys are typed (for Turn/Block metadata and Turn data).
- Enforce **serializable values** by storing a canonical serialized form (`json.RawMessage`) internally.
- Provide typed `Get/Set` with inference using `Key[T]`.
- Preserve YAML readability by rendering raw JSON values back into YAML-friendly values during marshal.

This is idiomatic in Go for “typed access to untyped bags”:

- Similar in spirit to `context.Context` key patterns, but improved with generics for compile-time type safety.
- Centralizes initialization + error reporting.

## Suggested API sketch (pseudocode)

```go
// turns/data.go
type Data struct {
	m map[TurnDataKeyID]json.RawMessage
}

// TurnDataKeyID is the stable identity of a Turn.Data entry: vs + slug + version.
// It must be comparable to be usable as a Go map key (strings + uint16 are comparable).
type TurnDataKeyID struct {
	vs      string
	slug    string
	version uint16
}

func MustDataKeyID(vs, slug string, version uint16) TurnDataKeyID {
	// validate non-empty, version >= 1, slug charset, etc.
	// panic is acceptable for defining package-level canonical keys.
	return TurnDataKeyID{vs: vs, slug: slug, version: version}
}

func (k TurnDataKeyID) String() string {
	// canonical encoding; used for logs and (via MarshalText) for YAML map keys
	return fmt.Sprintf("%s/%s@v%d", k.vs, k.slug, k.version)
}

func (k TurnDataKeyID) MarshalText() ([]byte, error) { return []byte(k.String()), nil }
func (k *TurnDataKeyID) UnmarshalText(b []byte) error {
	// parse "vs/slug@vN" into fields; validate; assign
	return nil
}

// Key carries the runtime key identity plus a phantom type for inference.
type Key[T any] struct{ id TurnDataKeyID }
func K[T any](id TurnDataKeyID) Key[T] { return Key[T]{id: id} }

func (d *Data) ensure() {
	if d.m == nil { d.m = map[TurnDataKeyID]json.RawMessage{} }
}

// Typed Set: the compiler ensures value matches the key's declared type.
func (d *Data) Set[T any](key Key[T], v T) error {
	d.ensure()
	// validate key.id (vs/slug/version present) to catch zero-values early
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	d.m[key.id] = b
	return nil
}

func (d Data) Get[T any](key Key[T]) (T, bool, error) {
	var zero T
	if d.m == nil { return zero, false, nil }
	b, ok := d.m[key.id]
	if !ok || len(b) == 0 {
		return zero, false, nil
	}
	if err := json.Unmarshal(b, &zero); err != nil {
		return zero, true, err
	}
	return zero, true, nil
}

func (d Data) Range(fn func(TurnDataKeyID, json.RawMessage) bool) {
	if d.m == nil { return }
	for k, v := range d.m {
		if !fn(k, v) { return }
	}
}

func (d Data) MarshalYAML() (any, error) { return d.m, nil }
func (d *Data) UnmarshalYAML(n *yaml.Node) error {
	if n == nil { d.m = nil; return nil }
	var tmp map[TurnDataKeyID]json.RawMessage
	if err := n.Decode(&tmp); err != nil { return err }
	d.m = tmp
	return nil
}
```

Note: the YAML marshal/unmarshal above is only a sketch. To keep YAML human-readable, `MarshalYAML` likely needs to convert `map[TurnDataKeyID]json.RawMessage` into `map[TurnDataKeyID]any` by decoding each JSON value into `any` first; similarly, `UnmarshalYAML` should decode into `map[TurnDataKeyID]any` and then `json.Marshal` each entry into raw JSON.

Canonical typed keys would live next to `keys.go`:

```go
var KeyToolConfig   = K[engine.ToolConfig](MustDataKeyID("geppetto", "tool_config", 1))
```

## What happens to runtime-only attachments (e.g., tool registry)?

If **everything in `Turn.Data` must be serializable**, then `tools.ToolRegistry` objects cannot live there.

Two common patterns to keep the ergonomics:

1. **Store a serializable spec, reconstruct at runtime**
   - Store `[]tools.ToolDefinition` (or tool IDs + config) in `Turn.Data`.
   - At runtime, build a registry object from definitions and pass it directly to the engine/middleware, or store it in a separate non-serialized structure.

2. **Split “data” vs “runtime”**
   - Keep `Turn.Data` strictly serializable.
   - Add `Turn.Runtime` (not serialized) for ephemeral attachments.
   - This cleanly separates concerns but requires touching the core types and serializers.

Given your new requirement, (1) is the strictest and simplest: keep one bag, but only store serializable representations.

## Migration sketch (incremental)

- Add `turns.Data` + `turns.Key[T]` in `geppetto/pkg/turns`.
- Introduce `TurnDataKeyID` (vs/slug/version) + canonical encoding (`vs/slug@vN`), and update YAML serde to use it.
- Convert `Turn.Data` field type and update `turns/serde.NormalizeTurn` (or replace it with `Data.ensure()` usage).
- Mechanically update key call sites:
  - `t.Data[oldKey] = v` → `t.Data.Set(turns.KeyX, v)` (typed)
  - `vAny, ok := t.Data[oldKey]` → `v, ok := t.Data.Get(turns.KeyX)`
  - `for k, v := range t.Data { ... }` → `t.Data.Range(func(k, v) bool { ...; return true })`
- Update linting approach:
  - Existing `turnsdatalint` is tailored to “typed-key expressions for typed-key maps” (and separately enforces const keys for `Block.Payload`).
  - With structured keys (non-const), prefer linting “only use canonical `turns.Key...` typed keys” and/or “ban `MustDataKeyID(...)` outside `turns/keys.go`”.

