---
Title: Debate synthesis v2 (concise rewrite)
Ticket: 001-REVIEW-TYPED-DATA-ACCESS
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
    - review
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md
      Note: Original long-form synthesis (source material for this rewrite)
    - Path: geppetto/pkg/turns/types.go
      Note: Current Turn/Block map structures (`Turn.Data`, `Turn.Metadata`, `Block.Metadata`)
    - Path: geppetto/pkg/turns/keys.go
      Note: Current canonical key definitions
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Current linter (prevents raw string drift)
    - Path: moments/backend/pkg/inference/middleware/current_user_middleware.go
      Note: Concrete middleware Turn.Data writes (identity fields)
    - Path: moments/backend/pkg/inference/middleware/thinkingmode/middleware.go
      Note: Concrete middleware read+default+write pattern
    - Path: moments/backend/pkg/inference/middleware/compression/turn_data_compressor.go
      Note: Concrete Turn.Data transformation pattern (string-keyed compression step)
ExternalSources: []
Summary: "Concise rewrite of the typed Turn.Data/Metadata synthesis: highlights the minimal decision set, the option space, and the tooling boundary—optimized for readability and review." 
LastUpdated: 2025-12-22T00:00:00-05:00
WhatFor: "A shorter, clearer synthesis of the typed Turn.Data/Metadata debate: make the design space navigable and decision-ready without reading 1000+ lines." 
WhenToUse: "Use when onboarding reviewers/implementers or when drafting follow-up RFCs; treat the original synthesis as an appendix/reference." 
---

# Debate synthesis v2 (concise rewrite)

## Executive Summary

We want to keep the flexibility of `Turn.Data`, `Turn.Metadata`, and `Block.Metadata` (map-like “bags”), while reducing three recurring costs:

- **Manual type safety**: repeated `if ok { v, ok := any.(T) }` patterns.
- **Boilerplate**: repeated nil-map initialization.
- **Late failures**: non-serializable values stored in bags that fail only when persisting (YAML).

The debate converged on a simple framing: **three primary decisions** determine almost everything else.

### TL;DR (three decisions)

1. **Key identity**: structured key ID (`{namespace, slug, version}`) vs encoded string (`"namespace.slug@vN"`).
2. **Value storage / serializability**: store `any` (maybe validate on write) vs store `json.RawMessage` (structurally serializable; decode on read).
3. **API boundary**: keep public maps (lint + conventions) vs opaque wrapper (central invariants; harder to bypass).

Everything else (naming, error semantics, lint rules, optional helpers) flows from those.

### How to read this document

- **If you’re deciding direction**: read **Decision Framework**.
- **If you’re implementing**: read **Decision Framework**, then **API surface options** and **Tooling/Linting**.
- **If you want the full archive**: see the linked original synthesis.

## Problem Statement (What hurts today)

Today’s model is intentionally “baggy”:

- `Turn.Data` is a map keyed by a typed string (`TurnDataKey`) storing `any`.
- `turnsdatalint` prevents raw string drift at map indexing sites (e.g. `t.Data["foo"]`).
- Nil map init is not centralized.
- Serializability is not structurally enforced.

Concrete middleware patterns show the pain:

### Example A — read + default + write requires boilerplate

From a typical middleware pattern:

```go
if t.Data == nil {
    t.Data = make(map[turns.TurnDataKey]any)
}
modeName, _ := t.Data[turnkeys.ThinkingMode].(string)
modeName = strings.TrimSpace(modeName)
if modeName == "" {
    modeName = ModeExploring
    t.Data[turnkeys.ThinkingMode] = modeName
}
```

Costs:
- nil-init boilerplate
- silent type assertion failures
- no single place to attach better error context

### Example B — transformations often want string-keyed views

Compression tooling is naturally written against `map[string]any`:

```go
func (tdc *TurnDataCompressor) Compress(ctx context.Context, data map[string]any) TurnDataCompressionOutcome {
    for key := range data {
        if s, ok := data[key].(string); ok {
            data[key] = strings.TrimSpace(s)
        }
    }
    return out
}
```

This becomes relevant if we introduce an opaque wrapper: we may still need an escape hatch that produces a string-keyed view.

## Decision Framework

### Decision 1 — Key identity

**Option A: encoded strings**

- Keep `type TurnDataKey string` and encode identity in the string (namespace + slug + version):
  - Example: `"geppetto.tool_config@v1"`.
- **Pros**: const keys; incremental adoption; simple YAML.
- **Cons**: identity fields are encoded; validation is tooling/runtime.

**Option B: structured IDs**

- Use a structured key ID (`{namespace, slug, version}`) with controlled construction.
- **Pros**: explicit identity; validation at construction; easier to reason about collisions.
- **Cons**: no `const`; keys become `var`; requires marshal/unmarshal plumbing.

**Decision lens**: do we prioritize incremental migration and const keys (encoded) or stronger identity invariants (structured)?

### Decision 2 — Value storage / serializability

**Option A: store `any`, validate on write (optional)**

- Fast reads; familiar to existing code.
- Validation can be “fail-fast” by marshaling at `Set` time.
- Still not a structural guarantee (map can hold anything if bypassed).

**Option B: store `json.RawMessage` internally**

- Structural serializability: if `Set` succeeds, persistence cannot fail due to unsupported Go types.
- Reads decode on-demand.
- Can still render as YAML by decoding JSON → `any` at marshal time.

**Decision lens**: do we want serializability to be a convention (A) or a guarantee (B)?

### Decision 3 — API boundary

**Option A: public maps + helpers**

- Keep `Turn.Data` as `map[TurnDataKey]any`.
- Add typed helpers using `Key[T]` for inference.
- Enforcement depends on linting + code review.

**Option B: opaque wrapper**

- Change `Turn.Data` to an opaque type with `Get/Set/Delete/Range/Len`.
- Centralizes nil init, error context, and serializability checks.
- Strongest way to prevent bypasses.

**Decision lens**: are we willing to accept “possible bypass” in exchange for simplicity, or do we want invariants enforced structurally?

## API surface options (call-site ergonomics)

Typed keys are the main ergonomics win: `Key[T]` enables type inference at call sites.

### Common building block: typed key wrapper

```go
type Key[T any] struct {
    // id is either TurnDataKey (encoded) or TurnDataKeyID (structured).
    //
    // This is pseudo-code: do not actually use `any` here in the implementation.
    // The real type should encode identity precisely.
    id any
}
```

(Exact representation depends on Decision 1.)

### Option A — public map + helper functions

```go
func GetData[T any](t *Turn, key Key[T]) (T, bool, error)
func SetData[T any](t *Turn, key Key[T], value T) error
func MustGetData[T any](t *Turn, key Key[T]) T
func MustSetData[T any](t *Turn, key Key[T], value T)
```

### Option B — opaque wrapper

```go
type Turn struct {
    Data Data `yaml:"data,omitempty"`
}

type Data struct { /* private */ }

func (d Data) Get[T any](key Key[T]) (T, bool, error)
func (d Data) MustGet[T any](key Key[T]) T
func (d *Data) Set[T any](key Key[T], value T) error
func (d *Data) MustSet[T any](key Key[T], value T)
func (d *Data) Delete(/* key identity */)
func (d Data) Len() int
func (d Data) Range(fn func(/* key identity */, /* raw */) bool)
```

## Tooling / linting (what’s realistically enforceable)

Today `turnsdatalint` ensures map indexing uses a typed key expression (no raw string literal drift). The debate surfaced a pragmatic tooling backlog:

- **Ban ad-hoc key construction**: forbid `TurnDataKey("...")` outside canonical key packages.
- **Enforce naming conventions** (if encoded strings): `namespace.slug(@vN)`.
- **Deprecation warnings**: parse `// Deprecated:` at key definitions and warn on usage.
- **Configurable strictness**: strict in CI; permissive locally (optional).
- **Test keys**: decide whether `test.*` is allowed in `_test.go`.

A useful mental model: **lint is great at preventing drift; it is not a substitute for an API boundary** if you truly need “no bypass” invariants.

## Open Questions (documented, not decided)

- **Error semantics**: should `Set` return error by default, or have both `Set` + `MustSet` / `TrySet`?
- **Caching** (if `json.RawMessage` storage): internal cache vs caller cache vs none.
- **YAML vs JSON contract**: what exactly must round-trip?
- **Metadata parity**: should `Turn.Metadata` and `Block.Metadata` share the same invariants and APIs as `Turn.Data`?

## Appendix (what got cut to stay concise)

This v2 is intentionally shorter. The original synthesis contains:

- exhaustive “All Ideas Surfaced” inventory
- long-form phased migration narrative
- participant position table

Use it as reference material:
- `geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md`
