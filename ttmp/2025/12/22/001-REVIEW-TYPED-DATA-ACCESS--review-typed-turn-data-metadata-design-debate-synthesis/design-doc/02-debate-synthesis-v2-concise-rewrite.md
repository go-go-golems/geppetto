---
Title: 'Debate synthesis v2: concise rewrite'
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
      Note: Original synthesis document (1001 lines); this is a concise rewrite (~400 lines)
    - Path: geppetto/pkg/turns/types.go
      Note: Current Turn.Data structure
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Current linter implementation
    - Path: moments/backend/pkg/inference/middleware/current_user_middleware.go
      Note: Real middleware usage patterns
    - Path: moments/backend/pkg/inference/middleware/compression/turn_data_compressor.go
      Note: Compression middleware transformation pattern
ExternalSources: []
Summary: "Concise rewrite of the typed Turn.Data/Metadata synthesis: decisions first, condensed axes, consolidated linting, API reference, and real code examples. ~60% shorter than original."
LastUpdated: 2025-12-22T15:00:00-05:00
WhatFor: "Quick decision-making reference: TL;DR decisions, condensed design axes, consolidated linting strategy, API options, and real-world code patterns."
WhenToUse: "Use when you need the essential decisions and trade-offs without the exhaustive debate record. For full detail, see the original synthesis document."
---

# Debate Synthesis v2: Typed Turn.Data/Metadata Design (Concise)

## Executive Summary

This document synthesizes **five debate rounds** exploring typed accessors, key identity, API surface, serializability, and tooling for `Turn.Data`/`Turn.Metadata`/`Block.Metadata`.

**Scope:** `Turn.Data`, `Turn.Metadata`, `Block.Metadata` only. `Block.Payload` is out-of-scope.

**Current state:** Maps with typed string keys (`TurnDataKey`) and `any` values. `turnsdatalint` prevents raw strings but doesn't enforce naming/versioning or value types. Nil-map init is scattered.

**Core question:** Can we keep flexibility while making usage safer and more discoverable?

---

## TL;DR (Decisions You Must Make)

**Three critical decisions** determine everything downstream:

1. **Key identity**: Structured (`TurnDataKeyID{vs, slug, version}`) vs encoded string (`"namespace.slug@v1"`)
2. **Value storage**: `any` (validate on `Set`) vs `json.RawMessage` (structurally serializable; decode on `Get`)
3. **API boundary**: Public map + helpers (incremental) vs opaque wrapper (central invariants)

**Consensus:** Typed keys `Key[T]` are essential (enables inference, removes type assertion boilerplate).

**No consensus:** Opaque vs public map (split 4-4), `json.RawMessage` vs `any` (split), error handling (split).

**Next:** See [Decision Framework](#decision-framework-what-to-decide) for detailed rationale, or [Major Design Axes](#major-design-axes-7-dimensions) for option details.

---

## Decision Framework (What to Decide)

### Critical Decisions (Must Decide First)

**Decision 1: Structured keys vs encoded strings?**
- **Structured:** `TurnDataKeyID{vs, slug, version}`, `var` keys, `MarshalText` required
- **Encoded:** `TurnDataKey string`, namespace/version in format, lint enforcement
- **Implication:** Determines whether canonical keys can be `const` (encoded) or must be `var` (structured)

**Decision 2: Store `any` or `json.RawMessage`?**
- **`any`:** Fast reads, validate on `Set` (not structurally guaranteed)
- **`json.RawMessage`:** Structurally guaranteed, unmarshal on `Get` (performance cost)
- **Implication:** Where serializability is enforced (boundary vs storage) and when errors appear

**Decision 3: Opaque wrapper or public map?**
- **Opaque:** Breaking change, centralized invariants, clear API, no bypass
- **Public map:** Incremental helpers, no breaking change, lint-driven enforcement
- **Implication:** Enforcement mechanism (structural vs tooling)

### Secondary Decisions

4. **Error handling:** One API (error/panic) vs two APIs (`Set`/`TrySet` or `MustSet`/`Set`)?
5. **Versioning:** Required from day one vs optional (defaults to v1)?
6. **Linter enhancements:** Ban ad-hoc keys? Enforce naming? Warn on deprecation? Configurable strictness?
7. **Schema registry:** None (typed keys enough) vs build-time vs linter report mode?

---

## Problem Statement

### What Hurts Today

1. **Type safety is manual**: Two-step type assertions, silent failures (zero value used)
2. **Nil-map boilerplate**: Scattered `if t.Data == nil` checks
3. **No versioning**: Keys are simple strings, legacy keys exist for compatibility
4. **No namespace enforcement**: Geppetto (`"tool_config"`) vs Moments (`"mento.person_id"`) by convention only
5. **Value types opaque**: Grep to find expected types
6. **Serializability not enforced**: Can store `chan int`, fails at YAML marshal time

---

## Code Examples (Real-World Patterns)

### Current Pattern (Middleware)

```go
// current_user_middleware.go
if t.Data == nil {
    t.Data = map[turns.TurnDataKey]any{}
}
if id := strings.TrimSpace(user.ID); id != "" {
    t.Data[turnkeys.PersonID] = id
}

// thinkingmode/middleware.go
modeName, found := getModeFromTurn(t)  // helper does: t.Data[key].(string)
if !found {
    if t.Data == nil {
        t.Data = make(map[turns.TurnDataKey]any)
    }
    t.Data[turnkeys.ThinkingMode] = modeName
}
```

**Pain points:** Nil-check boilerplate, type assertion in helpers, no error on wrong type.

### Proposed Pattern (Typed Keys)

```go
// With typed keys Key[T] and helpers
mode, ok, err := turns.GetData(t, turns.KeyThinkingMode)
if err != nil {
    return nil, fmt.Errorf("decode error: %w", err)
}
if !ok {
    mode = ModeExploring
    turns.MustSetData(t, turns.KeyThinkingMode, mode)
}
```

**Benefits:** Type inferred from key, explicit error handling, no nil-check (wrapper handles it).

### Edge Case: Compression Middleware

```go
// compression/turn_data_compressor.go converts to map[string]any
func (tdc *TurnDataCompressor) Compress(ctx context.Context, data map[string]any)
```

**Question:** If we go opaque wrapper, how does compression work? Options:
- `AsStringMap()` escape hatch (explicit conversion)
- Refactor compression to work with typed keys (breaking change)

---

## Proposed API Reference (Options)

### Option A: Public Map + Helpers

```go
// Turn.Data stays: map[TurnDataKey]any
func GetData[T any](t *Turn, key Key[T]) (T, bool, error)
func SetData[T any](t *Turn, key Key[T], value T) error
func MustGetData[T any](t *Turn, key Key[T]) T
func MustSetData[T any](t *Turn, key Key[T], value T)
```

**Call site:** `mode, ok, err := turns.GetData(t, turns.KeyThinkingMode)`

### Option B: Opaque Wrapper

```go
type Turn struct {
    Data Data `yaml:"data,omitempty"`
}

type Data struct { /* private: map[TurnDataKeyID]json.RawMessage or map[TurnDataKeyID]any */ }

func (d Data) Get[T any](key Key[T]) (T, bool, error)
func (d Data) MustGet[T any](key Key[T]) T
func (d *Data) Set[T any](key Key[T], value T) error
func (d *Data) MustSet[T any](key Key[T], value T)
func (d Data) Len() int
func (d Data) Range(fn func(/* key */, /* raw */) bool)
```

**Call site:** `mode, ok, err := t.Data.Get(turns.KeyThinkingMode)`

**Note:** If storing `json.RawMessage`, compression needs `AsStringMap()` or refactor.

---

## Linting Strategy (Summary)

**Current:** `turnsdatalint` prevents raw strings (`t.Data["foo"]`) but doesn't enforce canonical keys, naming, or versioning.

**If public map:** Linting becomes critical guardrail. Proposed rules:

- **Ban ad-hoc keys**: `TurnDataKey("oops")` only allowed in `*/keys.go` or `*/turnkeys/*.go`
- **Enforce naming**: Regex `^[a-z]+\.[a-z_]+(@v\d+)?$` (namespace + slug + optional version)
- **Deprecation warnings**: Parse `// Deprecated:` comments, warn at usage
- **Configurable strictness**: Strict in CI, permissive locally
- **Test key policy**: Allow `test.*` keys in tests, or require canonical test keys?

**If opaque wrapper:** Linting less critical (API boundary prevents bypass), but still useful for key conventions.

---

## Major Design Axes (7 Dimensions)

### 1. Key Identity: Structured vs Encoded

**Structured:** `TurnDataKeyID{vs, slug, version}` (unexported fields, compile-time enforcement, but `var` keys, requires `MarshalText`)

**Encoded:** `TurnDataKey string` with format `"namespace.slug@v1"` (keeps `const`, simple YAML, but runtime validation, possible ad-hoc construction)

**Trade-off:** Compile-time enforcement vs incremental migration.

### 2. Value Storage: `any` vs `json.RawMessage`

**`any`:** Validate on `Set` by marshaling (fast reads, fail-fast, but not structurally guaranteed)

**`json.RawMessage`:** Marshal at `Set`, unmarshal on `Get` (structurally guaranteed, but unmarshal cost on every read)

**Trade-off:** Structural guarantee vs performance.

### 3. Typed Access: `Key[T]` for Inference

**Consensus:** Typed keys `Key[T]` enable inference (`t.Data.Get(key)` not `t.Data.Get[Type](key)`), remove type assertion boilerplate.

**Return shape:** `(T, bool, error)` where `ok=false` means not found, `err!=nil` means decode error.

### 4. API Surface: Opaque vs Public Map

**Opaque:** Private map, `Get/Set/Range/Delete/Len` API, no bypass, centralized nil-init

**Public map:** Keep `map[TurnDataKey]any`, add helpers, incremental adoption, but bypassable (relies on linting)

**Trade-off:** Safety/encapsulation vs simplicity/migration.

### 5. Error Handling: Panic vs Error

**Two APIs (consensus):** `Set` panics (common case), `TrySet` returns error (validation), OR `MustSet` panics, `Set` returns error

**For `Get`:** Return `(T, bool, error)` (consensus)

### 6. Versioning Strategy

**Required:** Version mandatory (struct field or `@vN` suffix)

**Optional:** `@v1` defaults if omitted (gradual migration)

**Deprecation:** `// Deprecated:` comments, linter warns/errors

### 7. Application-Specific Keys

**Consensus:** Apps define keys in own packages (e.g., `moments/backend/pkg/turnkeys`), namespace prevents collisions.

**Enforcement:** Linter-based (package declares namespace) vs convention-based (infer from path) vs code-based (central registry)

---

## Key Tensions

1. **Type safety vs ergonomics**: Stronger safety = more ceremony (resolved by typed keys `Key[T]`)
2. **Structural guarantees vs performance**: `json.RawMessage` guarantees serializability but unmarshal cost
3. **Compile-time vs runtime enforcement**: Structured keys = compile-time, encoded = linting
4. **Opaque vs public**: Safety vs migration simplicity
5. **Fail-fast vs fail-late**: Serialization errors at `Set` vs `Marshal` time (consensus: fail-fast)
6. **Linter complexity vs human review**: Comprehensive linting vs maintainability

---

## Open Questions

### High Priority

1. **Error handling naming:** `Set`/`TrySet` vs `MustSet`/`Set`?
2. **Caching for `json.RawMessage`:** No cache vs internal cache vs caller cache?
3. **YAML vs JSON canonical:** Store JSON bytes, render YAML? Or store YAML nodes?

### Medium Priority

4. **`Metadata` same invariants as `Data`?** Same API? Same serializability?
5. **Deprecation enforcement:** Linter warns vs blocks vs docs only?
6. **Test key handling:** Allow `test.*` or require canonical?
7. **Namespace registry:** Linter-based vs convention vs code-based?

---

## Implementation Notes (Non-Prescriptive)

**Typed keys `Key[T]` are essential**—this is the core win (removes type assertion boilerplate, enables inference).

**Everything else is a trade-off:**
- Structured vs encoded keys (compile-time vs migration)
- `json.RawMessage` vs `any` (guarantee vs performance)
- Opaque vs public map (safety vs simplicity)

**Success criteria:**
- Fewer type assertion bugs (grep `.(T)` patterns)
- Fewer nil map panics (crash reports)
- Clearer key ownership (time to find definition)
- Faster code review (fewer questions about types)

---

## Appendix: Participant Positions

| Participant | Key Identity | Value Storage | API Surface | Error Handling | Linting |
|-------------|--------------|---------------|-------------|----------------|---------|
| **Asha** | Structured | `json.RawMessage` | Opaque wrapper | Two APIs | Strong |
| **Noel** | Prefer structured | `json.RawMessage` | Opaque or helpers | Return error | Moderate |
| **Priya** | Either | Validate on `Set` | Opaque or helpers | Return error | Feasible |
| **Mina** | Encoded string | Either | Public map + lint | Return error | Strong |
| **Sam** | Encoded string | `any` (no validation) | Public map + helpers | Return error | Minimal |
| **Jordan** | Don't care | Don't care | Simple syntax | Panic | Warn |
| **Casey** | Prefer structured | `json.RawMessage` | Opaque | Two APIs | Strong + config |
| **Ravi** | Either | Serializable | Either | Fail-fast | Moderate |

**Insights:**
- **Consensus:** Typed keys `Key[T]` (all support)
- **Split:** Opaque vs public map (4-4), `json.RawMessage` vs `any` (4-3), error handling (3-2-3)

---

## Related Documents

- **Original synthesis:** `geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md` (1001 lines, exhaustive)
- **Debate rounds:** See original synthesis for links to all 5 debate rounds
- **Review rounds:** `reference/04-review-round-1-initial-review-with-code-research.md`, `reference/05-review-round-2-document-clarity-and-conciseness.md`
