---
Title: 'Debate synthesis: typed Turn.Data/Metadata design'
Ticket: 001-TYPED-ACCESSOR-TURN-DATA
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
    - Path: pkg/analysis/turnsdatalint/analyzer.go
      Note: Current linter implementation
    - Path: pkg/inference/toolcontext/toolcontext.go
      Note: Runtime tool registry pattern (context-carried); illustrates what should stay out of Turn.Data.
    - Path: pkg/turns/keys.go
      Note: Current canonical key definitions
    - Path: pkg/turns/types.go
      Note: Current Turn/Block map structures
    - Path: moments/backend/pkg/inference/middleware/current_user_middleware.go
      Note: Concrete Turn.Data usage in middleware (real call-site pattern).
    - Path: moments/backend/pkg/inference/middleware/thinkingmode/middleware.go
      Note: Concrete Turn.Data read+default+write pattern in middleware.
    - Path: moments/backend/pkg/inference/middleware/compression/turn_data_compressor.go
      Note: Concrete Turn.Data transformation pattern (string-keyed compression step).
    - Path: ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/08-debate-round-1-q1-3-typed-accessors.md
      Note: Debate Round 1 source (invariants)
    - Path: ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/09-debate-round-2-q4-6-key-identity.md
      Note: Debate Round 2 source (key identity)
    - Path: ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/10-debate-round-3-q7-9-api-surface.md
      Note: Debate Round 3 source (API surface)
    - Path: ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/11-debate-round-4-q10-q12-serializability-failures.md
      Note: Debate Round 4 source (serializability)
    - Path: ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/12-debate-round-5-q13-14-tooling-schema.md
      Note: Debate Round 5 source (linter+schema)
ExternalSources: []
Summary: Comprehensive synthesis of 5 debate rounds exploring typed accessors, key identity, API surface, serializability, and tooling for Turn.Data/Metadata/Block.Metadata—surfacing tensions, trade-offs, and actionable design recommendations.
LastUpdated: 2025-12-22T00:00:00-05:00
WhatFor: "A readable, decision-maker-friendly report of what the debate surfaced: the design space, tensions, and concrete options for evolving Turn.Data/Metadata/Block.Metadata."
WhenToUse: "Use after reading the debate rounds to decide direction (key identity, value storage, API surface, and tooling), or when onboarding a new contributor to this ticket."
---


# Debate Synthesis: Typed Turn.Data/Metadata Design

## Executive Summary

This document is the “coming out of the debate” report. It pulls together what we learned across **five rounds** about evolving `Turn.Data`, `Turn.Metadata`, and `Block.Metadata` toward stronger type safety, clearer key identity, and reliable persistence.

The debates didn’t produce a single foregone conclusion. Instead they clarified *why* the current model feels good (it’s flexible and idiomatic) and *where* it repeatedly bites us (type assertions, nil map boilerplate, unclear ownership/meaning of keys, and late serialization failures). Most importantly, they surfaced a small set of decisions that determine everything downstream.

**Scope:** `Turn.Data`, `Turn.Metadata`, `Block.Metadata` only. `Block.Payload` is explicitly out-of-scope.

Today, the model is intentionally “baggy”:

- Maps with typed string keys (`type TurnDataKey string`) and `any` values
- `turnsdatalint` prevents raw-string drift (`t.Data["foo"]`) but does not enforce naming/versioning conventions or value types
- Runtime tool registry is already **out of `Turn.Data`** (it’s carried in `context.Context`)
- Nil map initialization happens in multiple places (serde + helpers), not centrally

The core question is: **can we keep the flexibility of “attach arbitrary per-turn facts” while making usage safer and more discoverable?** In practice, this breaks into a few sub-questions:

- What *must* a key communicate (namespace/slug/version)?
- Do we want serializability to be a best-effort convention, or a structural guarantee?
- Do we want callers to keep touching maps directly, or force access through an API boundary?

What the debates produced is a map of the design space: **7 major design axes** (mostly orthogonal) and a concrete backlog of ideas (lint rules, typed keys, helper APIs, wrappers, and migration sequencing).

### TL;DR (Decisions you must make)

If you only read one thing, make these choices explicit (everything downstream becomes mechanical refactoring):

1. **Key identity**: structured (`TurnDataKeyID{vs, slug, version}`) vs encoded string (e.g. `"namespace.slug@v1"`).
2. **Value storage / serializability**: store `any` (optionally validate on `Set`) vs store `json.RawMessage` (structurally serializable; decode on `Get`).
3. **API boundary**: public maps + helpers (incremental, lint-driven) vs opaque wrapper (central invariants; harder to bypass).

Where to go next:
- For the crisp decision list: see **Decision Framework** (now placed early, below Definitions).
- For option details: see **Major Design Axes**.
- For an exhaustive, grep-friendly backlog: see **Appendix: All Ideas Surfaced**.

### How to read this document

This synthesis is written for multiple audiences:

- **If you’re deciding direction**: start at **Decision Framework**.
- **If you’re implementing**: skim **Major Design Axes**, then jump to **Proposed API Reference (Options)** and **Implementation Approaches**.
- **If you’re reviewing changes**: focus on **Key Tensions** and the **Open Questions**.
- **If you want the raw debate**: jump straight to the linked debate rounds in **Related Documents**.

### Definitions (quick)

We use these terms consistently below:

- **bag**: a map-like “attach arbitrary things” container (`Data`, `Metadata`).
- **canonical key**: the blessed key definition in a keys package (as opposed to ad-hoc `TurnDataKey("oops")`).
- **typed key**: `Key[T]` wrapper that carries the key identity *and* the expected value type (for inference and reviewability).
- **encoded key**: key identity stored in a string format (e.g. `namespace.slug@v1`).
- **structured key**: key identity stored as fields (e.g. `{vs, slug, version}`).
- **serializable-only**: values can be persisted and round-tripped without special casing (YAML today; often JSON elsewhere).

---

## Decision Framework (What to Decide)

This section is intentionally practical: it translates the debate into the small set of choices that actually determine the shape of the implementation. If you make these decisions explicitly, the rest of the work becomes a series of mechanical refactors instead of endless bikeshedding.

### Critical Decisions (Must Decide First)

**Decision 1: Structured keys vs encoded strings?**
- **If structured:** Commit to `TurnDataKeyID{vs, slug, version}`, `var` keys, `MarshalText`
- **If encoded:** Keep `TurnDataKey string`, add namespace/version to format, lint enforcement
- **Hybrid option:** Start encoded, migrate to structured later

**Implication:** This choice sets how we talk about keys everywhere—how keys are authored, how they’re reviewed, and how we prevent collisions. It also determines whether “canonical keys” can be `const` (encoded) or must be `var` (structured).

**Decision 2: Store `any` or `json.RawMessage`?**
- **If `any`:** Optionally validate on `Set`, fast reads, but not structurally guaranteed
- **If `json.RawMessage`:** Structurally guaranteed, unmarshal on `Get`, consider caching (or caller-side caching)

**Implication:** This is really a decision about *where* serializability is enforced (boundary validation vs structural storage) and *when* decode errors appear (rarely at persistence edges vs potentially at read sites).

**Decision 3: Opaque wrapper or public map?**
- **If opaque:** Breaking change, centralized invariants, clear API
- **If public map:** Incremental helpers, no breaking change, enforcement relies on linting + convention

**Implication:** This choice sets the enforcement mechanism. Opaque wrappers make “the right thing” hard to bypass. Public maps keep Go ergonomics but require tooling to prevent drift.

### Secondary Decisions (Depend on Above)

**Decision 4: Error handling for `Set`**
- One API (return error or panic)?
- Two APIs (`Set` + `TrySet` or `MustSet` + `Set`)?

**Decision 5: Versioning policy**
- Required from day one?
- Optional (defaults to v1 if omitted)?

**Decision 6: Linter enhancements**
- Ban ad-hoc keys?
- Enforce naming conventions?
- Warn on deprecated keys?
- Configurable strictness?

**Decision 7: Schema registry**
- None (typed keys are enough)?
- Build-time (generated file)?
- Linter report mode (on-demand)?

---

## Linting Strategy (Summary)

Today `turnsdatalint` prevents raw string drift (`t.Data["foo"]`) but it does **not** enforce canonical key ownership, naming/versioning conventions, or value-shape expectations.

If we keep **public maps**, linting becomes a key guardrail. The debate’s most practical lint rules were:

- **Ban ad-hoc key construction**: forbid `TurnDataKey("oops")` outside canonical keys packages.
- **Enforce key naming conventions**: e.g. `^[a-z]+\.[a-z_]+(@v\d+)?$` (namespace + slug + optional version).
- **Deprecation warnings**: parse `// Deprecated:` comments and warn at usage sites.
- **Configurable strictness**: strict in CI; permissive locally if needed.
- **Test key policy**: decide whether tests may use `test.*` keys, or require canonical test keys.

---

## Proposed API Reference (Options)

This is intentionally not a decision—just a compact reference for what each choice implies at call sites.

### Option A: Public map + helpers (incremental; lint-driven)

```go
// Turn.Data stays: map[TurnDataKey]any
func GetData[T any](t *Turn, key Key[T]) (T, bool, error)
func SetData[T any](t *Turn, key Key[T], value T) error
func MustGetData[T any](t *Turn, key Key[T]) T
func MustSetData[T any](t *Turn, key Key[T], value T)
```

### Option B: Opaque wrapper (central invariants; no bypass)

```go
type Turn struct {
    Data Data `yaml:"data,omitempty"`
}

type Data struct { /* private storage */ }

func (d Data) Get[T any](key Key[T]) (T, bool, error)
func (d Data) MustGet[T any](key Key[T]) T
func (d *Data) Set[T any](key Key[T], value T) error
func (d *Data) MustSet[T any](key Key[T], value T)
func (d Data) Len() int
func (d Data) Range(fn func(/* key identity */, /* raw representation */) bool)
```

---

## Problem Statement

### What Hurts Today (Evidence from Code)

The pain points below are not theoretical—they’re the patterns that repeatedly show up at call sites and in helpers. They also explain why “just use the map” starts simple but accumulates edge cases.

1. **Type safety is manual**
   - Call sites do two-step type assertions: check existence, then assert type
   - Example: `engine_openai.go:127-130` shows nested `if ok` checks
   - Silent failures: type assertion fails → zero value used with no error/log

2. **Map initialization boilerplate**
   - Scattered `if t.Data == nil { t.Data = map[...]{} }` checks
   - Found in `serde.go:25`, `toolhelpers.go:297`, and likely more

3. **No versioning story**
   - Keys are simple strings (`"tool_config"`, `"person_id"`)
   - No explicit versioning (no `@v1`, no version field)
   - Legacy keys exist (`PersonIDLegacy`, `PersonIDCamelCase`) for compatibility

4. **No namespace enforcement**
   - Geppetto keys: `"tool_config"` (no namespace)
   - Moments keys: `"mento.person_id"` (namespace prefix by convention)
   - No linter rule preventing collisions

5. **Value types are opaque**
   - To find what type a key expects, you grep for usage
   - No single source of truth mapping keys → expected types

6. **Serializability is not enforced**
   - Nothing prevents storing `make(chan int)` in `Turn.Data`
   - Fails late (at serialization time) with `yaml: unsupported type`

---

## Code Examples (Real-world call-site patterns)

These examples are meant to make the abstract trade-offs concrete. They are *not* prescriptive; they illustrate the patterns the design is trying to improve.

### Middleware pattern: read + default + write (nil map init)

```go
// Example shape (from moments thinking_mode middleware):
if t.Data == nil {
    t.Data = make(map[turns.TurnDataKey]any)
}
modeName, _ := t.Data[turnkeys.ThinkingMode].(string)
if strings.TrimSpace(modeName) == "" {
    modeName = ModeExploring
    t.Data[turnkeys.ThinkingMode] = modeName
}
```

### Middleware pattern: transform values (string-focused compression)

```go
// Example shape (from moments turn_data_compressor):
func (tdc *TurnDataCompressor) Compress(ctx context.Context, data map[string]any) TurnDataCompressionOutcome {
    // ... drop fields ...
    for key := range data {
        switch v := data[key].(type) {
        case string:
            // summarize / truncate
            data[key] = strings.TrimSpace(v)
        }
    }
    return out
}
```

---

## Major Design Axes (7 Dimensions)

Most debate disagreements weren’t about one single mechanism—they were about *where* to put the boundary and *how strong* to make the guarantees. The seven axes below are mostly independent: you can mix-and-match (e.g., encoded keys + opaque wrapper; structured keys + public map; `any` storage + typed keys, etc.). That’s useful, because it lets us iterate in phases instead of requiring a big-bang rewrite.

### 1. Key Identity: Structured vs Encoded

Key identity is the foundation: it determines whether keys are merely “names” or whether they encode meaning (namespace and version) that tools and humans can rely on. This choice affects how we prevent collisions across packages (Geppetto vs Moments), how we version values when their shape changes, and how reviewable the code feels when you jump to a key definition.

**Option A: Structured type (compile-time enforcement)**
```go
type TurnDataKeyID struct {
    vs      string  // namespace
    slug    string  // identifier
    version uint16  // version
}
```

**Pros:**
- Fields are unexported (impossible to construct without all parts)
- Validation at construction time (in `MustDataKeyID`)
- Clear separation of concerns (vs/slug/version are distinct fields)

**Cons:**
- Cannot use `const` (Go limitation: structs aren't const-able)
- Canonical keys become `var` (not `const`)
- Linter must evolve from "enforce const keys" to "enforce canonical vars"
- Requires `MarshalText`/`UnmarshalText` for YAML serialization

**Option B: Encoded string (convention + linting)**
```go
type TurnDataKey string
const DataKeyToolConfig TurnDataKey = "geppetto.tool_config@v1"
```

**Pros:**
- Keeps `const` support (familiar Go pattern)
- Incremental migration (add namespace/version to existing keys gradually)
- Simple YAML serialization (already works)
- Linter can parse format with regex

**Cons:**
- Namespace/slug/version are encoded (not separate fields)
- Validation is runtime (linter parses strings)
- Possible to construct ad-hoc keys (`TurnDataKey("oops")`)

**Trade-off:** Compile-time enforcement (structured) vs incremental migration (encoded).

---

### 2. Value Storage: `any` vs `json.RawMessage`

This axis is about whether serializability is a convention (“we try not to put weird things here”) or a structural guarantee (“it is literally impossible to store non-serializable values”). It also determines *where* errors appear: at `Set`, at `Get`, or later during persistence.

**Option A: Store `any`, validate on `Set`**
```go
type Data struct {
    m map[TurnDataKeyID]any
}

func (d *Data) Set[T any](key Key[T], value T) error {
    if _, err := json.Marshal(value); err != nil {
        return fmt.Errorf("not serializable: %w", err)
    }
    d.m[key.id] = value
    return nil
}
```

**Pros:**
- Fast reads (no unmarshal, just type assertion or direct use)
- Familiar pattern (similar to current `any` maps)
- Fail-fast (serializability checked at write time)

**Cons:**
- Validates by marshaling but doesn't store marshaled form (wasteful)
- Could marshal to JSON but not to YAML (rare, but possible)
- Still stores `any` (not structurally guaranteed to be serializable)

**Option B: Store `json.RawMessage`, decode on `Get`**
```go
type Data struct {
    m map[TurnDataKeyID]json.RawMessage
}

func (d Data) Get[T any](key Key[T]) (T, bool, error) {
    var zero T
    b, ok := d.m[key.id]
    if !ok { return zero, false, nil }
    if err := json.Unmarshal(b, &zero); err != nil {
        return zero, true, err
    }
    return zero, true, nil
}
```

**Pros:**
- Structurally guaranteed serializable (if `Set` succeeds, value is JSON)
- Marshal only once (at `Set` time, not twice for validation + serialization)
- Clear boundary (only serialized data in map)

**Cons:**
- Unmarshal cost on every `Get` (allocation + CPU)
- If same key read multiple times, repeated unmarshal (unless cached)
- Caching adds complexity (when to invalidate?)

**Trade-off:** Structural guarantee (JSON bytes) vs performance (store `any`).

---

### 3. Typed Access: Inference via `Key[T]`

This is the ergonomics axis. It’s where the debate found the clearest win: typed keys can remove the repetitive “check + type assert” pattern while keeping call sites readable. In Go specifically, the key trick is using `Key[T]` so `T` can be inferred from the key argument.

**Current pattern:**
```go
if cfgAny, ok := t.Data[turns.DataKeyToolConfig]; ok && cfgAny != nil {
    if cfg, ok := cfgAny.(engine.ToolConfig); ok {
        // use cfg
    }
}
```

**Proposed pattern (typed keys):**
```go
type Key[T any] struct { id TurnDataKeyID }
var KeyToolConfig = Key[engine.ToolConfig]{id: ...}

cfg, ok, err := t.Data.Get(turns.KeyToolConfig)  // T inferred from key
```

**Why this works:** Go can infer `T` from the `Key[T]` parameter, so you don't need explicit type args.

**Without typed keys (worse):**
```go
cfg, ok, err := t.Data.Get[engine.ToolConfig](turns.DataKeyToolConfig)  // explicit type arg
```

**Consensus:** Typed keys `Key[T]` are essential for ergonomic inference.

---

### 4. API Surface: Opaque vs Public Map

This axis is about boundaries. If you keep a public map, you preserve idiomatic Go ergonomics but you rely on linting and conventions to prevent drift. If you make the bag opaque, you can enforce invariants centrally (initialization, serializability checks, consistent error messages), but you pay with a larger refactor and a new API surface.

**Option A: Opaque wrapper**
```go
type Turn struct {
    Data Data `yaml:"data,omitempty"`  // Data is a wrapper, not a map
}

type Data struct {
    m map[TurnDataKeyID]json.RawMessage  // private
}

// API: Get/Set/Range/Delete/Len
```

**Pros:**
- Prevents direct map access (no bypass)
- Centralizes initialization (no scattered nil checks)
- Can enforce invariants at boundaries (typed access, serializability)
- Clear API contract

**Cons:**
- Breaking change (all call sites must migrate)
- More verbose (`t.Data.Set(key, val)` vs `t.Data[key] = val`)
- Requires YAML marshal/unmarshal implementation

**Option B: Public map + helpers**
```go
type Turn struct {
    Data map[TurnDataKey]any `yaml:"data,omitempty"`  // still public
}

// Helpers
func GetData[T any](t *Turn, key Key[T]) (T, bool, error) { ... }
func SetData[T any](t *Turn, key Key[T], value T) error { ... }
```

**Pros:**
- Non-breaking (existing code keeps working)
- Incremental adoption (use helpers where needed)
- Simple migration path

**Cons:**
- Doesn't prevent direct map access (can bypass helpers)
- Nil map checks still scattered (unless helpers handle it)
- No centralized enforcement

**Trade-off:** Safety/encapsulation (opaque) vs simplicity/migration (public map).

---

### 5. Error Handling: Panic vs Return Error

Once we add invariants, we have to decide how violations surface to developers. The debate consistently preferred “fail early” over “fail during YAML serialization”, but it split on the UX: panics are blunt but low-ceremony; errors are explicit but add boilerplate at call sites. A two-API approach is a common compromise in Go.

**For `Set` (write operations):**

**Option A: Always return error**
```go
func (d *Data) Set[T any](key Key[T], value T) error
```
- **Pro:** Testable, no process crashes, caller decides how to handle
- **Con:** Ceremony at every call site (must handle error)

**Option B: Panic on error**
```go
func (d *Data) Set[T any](key Key[T], value T)  // panics if not serializable
```
- **Pro:** Simple syntax, no error handling for common cases
- **Con:** Crashes process, harder to test error cases

**Option C: Two APIs**
```go
func (d *Data) Set[T any](key Key[T], value T)              // panics on error
func (d *Data) TrySet[T any](key Key[T], value T) error     // returns error
// OR
func (d *Data) MustSet[T any](key Key[T], value T)          // panics on error
func (d *Data) Set[T any](key Key[T], value T) error        // returns error
```
- **Pro:** Best of both worlds (simple for common cases, safe for edge cases)
- **Con:** Two APIs to remember

**Debate insight:** Most participants favor **Option C** (two APIs), but disagree on naming:
- Jordan/Casey prefer: `Set` panics (common case), `TrySet` returns error (validate)
- Alternative: `MustSet` panics (assert), `Set` returns error (default)

**For `Get` (read operations):**

**Consensus:** Return `(T, bool, error)`:
- `ok=false` → key not found
- `ok=true, err=nil` → success
- `ok=true, err=DecodeError` → key exists but wrong shape

---

### 6. Versioning Strategy

Versioning is the “what happens when we change our mind” policy. It’s less about the syntax and more about avoiding silent divergence: a version should make it obvious when a value’s shape changed, and deprecations should be visible enough that new code doesn’t keep using the old form forever.

**Current reality:** No versioning in key strings; legacy keys show pain (`PersonIDLegacy`, `PersonIDCamelCase`).

**Consensus:** Versioning should be **in the key identity** (not implicit).

**Option A: Struct field**
```go
type TurnDataKeyID struct {
    vs      string
    slug    string
    version uint16  // required
}
```

**Option B: String suffix**
```go
const DataKeyToolConfig TurnDataKey = "geppetto.tool_config@v1"
```

**Option C: Optional string suffix**
```go
const DataKeyToolConfig TurnDataKey = "geppetto.tool_config"  // implicitly v1
const DataKeyToolConfigV2 TurnDataKey = "geppetto.tool_config@v2"
```

**Trade-offs:**
- **Struct field:** Explicit and enforced, but requires `var` (not `const`)
- **String suffix (required):** Explicit, keeps `const`, but all keys need updating
- **String suffix (optional):** Gradual migration, but implicit v1 is invisible

**Deprecation story:**
- Mark old keys with `// Deprecated: use KeyXV2 instead`
- Linter warns on usage of deprecated keys
- Migration guide with timeline

---

### 7. Application-Specific Keys

This axis is about ownership and scale. Geppetto has a handful of shared keys; applications like Moments have many more. The debate strongly favored letting apps define their own keys in their own packages—but only if we can prevent collisions and drift (via namespace conventions and tooling).

**Current reality:**
- Geppetto: 4 keys in `pkg/turns/keys.go` (no namespace prefix)
- Moments: 20+ keys in `backend/pkg/turnkeys/` (namespace prefix `"mento.*"`)

**Consensus:** Application-specific keys live in their own packages.

**Namespace enforcement options:**

**Option A: Linter-based**
- Each package declares its namespace (via comment or config)
- Linter enforces: "All keys in `moments/backend/pkg/turnkeys` must start with `mento.`"

**Option B: Convention-based**
- Infer namespace from package path (`turnkeys` → use parent dir name)
- No explicit declaration needed

**Option C: Code-based registry**
- Central file lists all valid namespaces
- Keys reference namespace via const

**Trade-offs:**
- Linter-based: Flexible, no central file, but requires package-level metadata
- Convention-based: Automatic, but not foolproof (what if package moves?)
- Code-based: Explicit, but creates dependencies and maintenance burden

---

## Key Tensions (Across All Rounds)

If the design axes are “what knobs exist”, these tensions are “what knobs fight each other.” They’re useful both for decision-making (“which trade-off do we accept?”) and for reviewing proposals (“which tension is this proposal resolving, and what does it make worse?”).

### 1. Type Safety vs Ergonomics

**The tension:** Stronger type safety often means more ceremony.

**Evidence:**
- **Current:** `t.Data[key] = value` (simple, but unsafe)
- **Proposed:** `if err := t.Data.Set(key, value); err != nil { ... }` (safe, but verbose)

**Resolution paths:**
- Use typed keys `Key[T]` for inference (reduces type arg noise)
- Two APIs: simple for common cases (`Set` panics), safe for edge cases (`TrySet` returns error)
- Opaque wrapper centralizes nil checks (removes one source of boilerplate)

---

### 2. Structural Guarantees vs Performance

**The tension:** Storing `json.RawMessage` guarantees serializability but adds unmarshal cost on reads.

**Evidence:**
- Store `any` + validate: Fast reads, but not structurally guaranteed
- Store JSON bytes: Guaranteed serializable, but unmarshal on every `Get`

**Resolution paths:**
- Accept the trade-off (structural guarantee is worth the cost)
- Add caching (decode once, cache result—but adds complexity)
- Encourage explicit local caching (caller responsibility)
- Make validation optional (strict mode for tests/CI, permissive for hot paths)

---

### 3. Compile-Time vs Runtime Enforcement

**The tension:** Type system enforcement requires bigger changes; linting is incremental.

**Evidence:**
- Structured keys require changing `const` to `var`, updating all call sites, implementing `MarshalText`
- Linting can enforce conventions without breaking existing code

**Resolution paths:**
- Type system for **value types** (typed keys `Key[T]`)
- Linting for **key conventions** (naming, namespaces, deprecation)
- Both are needed (complementary, not competing)

---

### 4. Opaque vs Public: Safety vs Migration

**The tension:** Opaque wrappers prevent misuse but require large refactors.

**Evidence:**
- Opaque: No bypass, centralized logic, clear API contract
- Public map: Incremental adoption, no breaking changes, familiar pattern

**Resolution paths:**
- Hybrid: Start with public map + helpers, migrate to opaque long-term
- Two-phase migration: Add helpers first, change field type later
- Provide compatibility shim during transition

---

### 5. Fail-Fast vs Fail-Late

**The tension:** When should we catch serializability violations?

**Evidence:**
- **Fail-late (current):** Fails at `yaml.Marshal` time (production surprise)
- **Fail-fast:** Fails at `Set` time (immediate feedback)

**Consensus:** Fail-fast is better. Debate is about **how**:
- Panic (treats non-serializable as bug)
- Return error (treats non-serializable as runtime condition)
- Two APIs (let caller decide)

---

### 6. Linter Complexity vs Human Review

**The tension:** Comprehensive linting catches mistakes early but adds maintenance burden.

**Evidence:**
- Current linter: Simple (ban raw strings), fast, maintainable
- Proposed enhancements: Naming conventions, canonical keys, deprecation warnings

**Resolution paths:**
- Incremental linter evolution (add rules gradually)
- Configurable strictness (strict in CI, permissive locally)
- Keep linter focused (don't do whole-program analysis)

---

### 7. Schema Registry vs Typed Keys

**The tension:** Do we need a central registry mapping keys → expected types?

**Evidence:**
- Typed keys `Key[T]` already carry type information (in the type parameter)
- Jump to definition shows expected type immediately
- No grep needed

**Consensus:** No runtime schema registry (typed keys are enough).

**Build-time registry debate:**
- **Pro:** Discoverability (list all keys), collision detection
- **Con:** Generated file maintenance, import cycles
- **Alternative:** Linter report mode (scan code, output JSON)

---

## Appendix: All Ideas Surfaced (Organized by Category)

Treat this section as an indexable backlog. It’s intentionally exhaustive: you can skim headings to find a theme (typed access, serializability, key identity, tooling), then jump to the detailed debate round(s) if you want the original rationale.

### A. Typed Access Mechanisms

1. **Typed keys `Key[T]` for inference** (Asha, Priya)
   - Enables `t.Data.Get(turns.KeyToolConfig)` with inferred `T`
   - No explicit type args (`Get[engine.ToolConfig]`)

2. **Three-value return for `Get`** (Consensus)
   - `(T, bool, error)` distinguishes "not found" vs "decode error"
   - `ok=false` → key missing
   - `ok=true, err=nil` → success
   - `ok=true, err!=nil` → key exists but wrong shape

3. **`MustGet` for tests** (Jordan, Casey)
   - Panics if key not found or decode fails
   - Cleaner test assertions (no `require.NoError` boilerplate)

4. **Context-aware error messages** (Jordan, Noel)
   - Include key name and expected type in errors
   - `"Turn.Data[tool_config]: expected engine.ToolConfig, got string"`

---

### B. Serializability Enforcement

5. **Store `json.RawMessage` internally** (Noel, Casey)
   - Structurally guaranteed serializable
   - Marshal once (at `Set`), unmarshal on `Get`

6. **Validate on `Set` by marshaling** (Priya, Jordan)
   - Store `any`, but check serializability at write time
   - Fail-fast without unmarshal cost on reads

7. **Round-trip validation (opt-in)** (Casey)
   - `Set` marshals then unmarshals to check round-trip fidelity
   - Enabled in tests/staging, disabled in production

8. **Two `Set` APIs** (Jordan, Casey)
   - `Set` panics (for "this should never fail" cases)
   - `TrySet` returns error (for "validate this" cases)
   - OR: `MustSet` panics, `Set` returns error

---

### C. Key Identity & Versioning

9. **Namespace prefixes** (Mina, Casey)
   - Format: `"namespace.slug@vN"`
   - Examples: `"geppetto.tool_config@v1"`, `"mento.person_id@v1"`

10. **Structured key ID** (Asha, Priya)
    - `TurnDataKeyID{vs, slug, version}`
    - Enforced at construction via unexported fields

11. **Optional versioning** (Mina)
    - `@v1` defaults if omitted
    - Gradual migration (add versions incrementally)

12. **Explicit versioning required** (Asha, Casey)
    - Version is mandatory (struct field or string suffix)
    - Forces clarity from day one

13. **Deprecation markers** (Casey, Priya)
    - `// Deprecated: use KeyXV2 instead`
    - Linter parses comments and warns at usage sites

14. **Migration timeline** (Casey)
    - "Migrate from KeyX to KeyXV2 by YYYY-Q1"
    - Clear communication for developers

---

### D. API Surface & Wrappers

15. **Opaque wrapper with minimal API** (Asha)
    - `Get/Set/Range/Delete/Len`
    - Private map, no bypass

16. **Public map with helper functions** (Sam)
    - Keep `map[TurnDataKey]any` public
    - Add `GetData[T]`, `SetData[T]` helpers
    - Incremental adoption

17. **Two `Range` variants** (Casey)
    - `Range(func(key, json.RawMessage) bool)` for fast persistence
    - `RangeTyped(func(key, any) bool)` for debugging

18. **Single `Range` exposing raw map** (Mina)
    - `Range(func(key, any) bool)` gives access to underlying map
    - Caller uses `Get[T]` inside loop for typed access

---

### E. Linter Evolution

19. **Ban ad-hoc key construction** (Mina, Casey, Asha)
    - Forbid `TurnDataKey("oops")` outside keys packages
    - Linter checks: "Key construction only in `*/keys.go` or `*/turnkeys/*.go`"

20. **Enforce naming conventions** (Mina, Casey)
    - Regex: `^[a-z]+\.[a-z_]+(@v\d+)?$`
    - Ensures all keys have namespace + slug + optional version

21. **Deprecation warnings** (Casey, Priya, Mina)
    - Parse `// Deprecated:` comments
    - Warn at usage sites

22. **Configurable strictness** (Casey)
    - Strict mode (CI): block on violations
    - Permissive mode (local dev): warn only

23. **Test key exemption** (Casey)
    - Allow `"test.namespace.slug"` keys in test files only
    - Linter enforces file-based rules

---

### F. Namespace & Collision Prevention

24. **Linter-based namespace registry** (Priya, Casey)
    - Each package declares namespace (via comment or config)
    - Linter enforces that all keys use declared namespace

25. **Automatic namespace inference** (Jordan)
    - Infer from package path (`moments/backend/pkg/turnkeys` → `"mento"`)
    - Convention-based (no explicit declaration)

26. **Build-time registry (generated file)** (Casey)
    - List all keys with metadata (namespace, slug, version, type, package)
    - Used for discoverability, collision detection, docs generation

27. **Linter report mode** (Priya)
    - Linter outputs JSON report of all keys (no generated file)
    - On-demand generation (run when needed)

---

### G. Migration & Compatibility

28. **Hybrid approach** (Jordan)
    - Short-term: Add namespace prefixes to string keys (encoded)
    - Long-term: Migrate to structured keys (`TurnDataKeyID`)
    - Gives benefits of both (incremental + structured)

29. **Legacy key pattern** (from Moments code)
    - Keep old keys for compatibility: `PersonIDLegacy turns.TurnDataKey = "person_id"`
    - Mark with `// Deprecated:` comment
    - Linter warns on usage

30. **Incremental linter evolution** (Mina, Sam)
    - Add rules one at a time (naming → canonical keys → deprecation)
    - Each step is independently useful
    - No big-bang refactor

---

## Decision Framework (What to Decide)
Moved to the top of this document (after the Executive Summary) to reduce redundancy and make the “what to decide” section easier to find.

---

## Implementation Approaches (Phased vs Big-bang)

The debates surfaced a pragmatic sequencing: get value early (typed access + better discoverability), then decide whether to “harden” the model (structured keys, opaque wrapper, structural serializability).

Two implementation styles are compatible with the same end-state:
- **Phased**: do the work in stages to de-risk adoption.
- **Big-bang**: apply the same checklist in one change-set (higher coordination cost; faster convergence).

Below is the phased checklist (useful even if you choose big-bang—treat it as an ordered checklist).

### Phase 1: Foundation (Minimal Breaking Change)

**Goal:** Add typed access without breaking existing code.

**Changes:**
1. Add typed key wrappers:
   ```go
   type Key[T any] struct { id TurnDataKey }  // uses existing TurnDataKey
   func K[T any](id TurnDataKey) Key[T] { return Key[T]{id: id} }
   ```

2. Add helper functions (not opaque yet):
   ```go
   func GetData[T any](t *Turn, key Key[T]) (T, bool, error)
   func SetData[T any](t *Turn, key Key[T], value T) error
   func MustSetData[T any](t *Turn, key Key[T], value T)  // panics
   func MustGetData[T any](t *Turn, key Key[T]) T         // panics
   ```

3. Convert canonical keys to typed:
   ```go
   var KeyToolConfig = K[engine.ToolConfig]("tool_config")
   var KeyAgentMode = K[string]("agent_mode")
   ```

4. Enhance linter:
   - Ban ad-hoc key construction (`TurnDataKey("oops")`)
   - Enforce: keys must be canonical consts/vars

**Migration:** Incremental. Existing code keeps working; new code uses typed helpers.

---

### Phase 2: Structured Keys & Namespaces

**Goal:** Add namespace and versioning to key identity.

**Changes:**
1. Introduce `TurnDataKeyID`:
   ```go
   type TurnDataKeyID struct {
       vs      string
       slug    string
       version uint16
   }
   func (k TurnDataKeyID) String() string {
       return fmt.Sprintf("%s/%s@v%d", k.vs, k.slug, k.version)
   }
   ```

2. Update `Key[T]` to use `TurnDataKeyID`:
   ```go
   type Key[T any] struct { id TurnDataKeyID }
   var KeyToolConfig = Key[engine.ToolConfig]{
       id: MustDataKeyID("geppetto", "tool_config", 1),
   }
   ```

3. Update linter:
   - Enforce: keys must include namespace (no bare `"tool_config"`)
   - Enforce: version must be ≥ 1

**Migration:** Update all canonical keys. Call sites don't change (still use `KeyToolConfig`).

---

### Phase 3: Opaque Wrapper & Serializability

**Goal:** Make `Data` opaque and enforce serializability structurally.

**Changes:**
1. Change `Turn.Data` from map to wrapper:
   ```go
   type Turn struct {
       Data Data `yaml:"data,omitempty"`
   }
   
   type Data struct {
       m map[TurnDataKeyID]json.RawMessage  // private
   }
   ```

2. Implement wrapper API:
   ```go
   func (d *Data) Set[T any](key Key[T], value T) error
   func (d *Data) MustSet[T any](key Key[T], value T)
   func (d Data) Get[T any](key Key[T]) (T, bool, error)
   func (d Data) MustGet[T any](key Key[T]) T
   func (d Data) Range(fn func(TurnDataKeyID, json.RawMessage) bool)
   func (d Data) Len() int
   ```

3. Implement YAML marshal/unmarshal:
   - `MarshalYAML`: decode JSON → `any`, emit as YAML (human-readable)
   - `UnmarshalYAML`: decode YAML → `any`, encode to JSON, store bytes

**Migration:** Breaking change. All `t.Data[key] = value` → `t.Data.Set(key, value)`.

---

## Open Questions (Require Decisions)

### High Priority

1. **Error handling: `Set` vs `TrySet` or `MustSet` vs `Set`?**
   - Which should be the "default" API?
   - What should the naming convention be?

2. **Caching strategy for `json.RawMessage` storage?**
   - No caching (unmarshal on every `Get`)?
   - Internal cache (when to invalidate)?
   - Encourage explicit local caching (caller responsibility)?

3. **YAML vs JSON canonical format?**
   - Store JSON bytes, render as YAML?
   - Or store YAML nodes, derive JSON at edges?
   - Can we assume JSON ⊆ YAML for our use cases?

### Medium Priority

4. **Should `Metadata` have same invariants as `Data`?**
   - Same API surface?
   - Same serializability enforcement?
   - Or more permissive (for provider annotations)?

5. **Deprecation policy enforcement?**
   - Linter warns?
   - Linter blocks?
   - Documentation only?

6. **Test key handling?**
   - Allow `"test.*"` keys in test files only?
   - Or trust developers (no special rules)?

7. **Namespace registry approach?**
   - Linter-based (package declares namespace)?
   - Convention-based (infer from path)?
   - Code-based (central file)?

### Low Priority

8. **Should we support legacy keys long-term?**
   - Keep `PersonIDLegacy` pattern?
   - Or force migration with deprecation timeline?

9. **Build-time key registry?**
    - Generated file?
    - Linter report mode?
    - Neither?

---

## Implementation Notes (Non-prescriptive)

This synthesis is intentionally about the **design space** and decision points, not a mandated execution plan. If you do implement changes, treat the **Implementation Approaches** section as an ordered checklist and tailor it to your constraints.

### Success Criteria (if you implement any of the proposals)

- **Fewer type assertion bugs** (measure: grep for `.(T)` patterns in Turn.Data access)
- **Fewer nil map panics** (measure: crash reports, test failures)
- **Clearer key ownership** (measure: time to find key definition)
- **Faster code review** (measure: reviewer questions about key types)

---

## Appendix: Participant Positions (Summary Table)

| Participant | Key Identity | Value Storage | API Surface | Error Handling | Linting |
|-------------|--------------|---------------|-------------|----------------|---------|
| **Asha** (Type-Safety) | Structured (`TurnDataKeyID`) | `json.RawMessage` | Opaque wrapper | Two APIs (panic + error) | Strong enforcement |
| **Noel** (Serialization) | Prefer structured | **`json.RawMessage`** (must) | Opaque or helpers | Return error | Moderate |
| **Priya** (Go Specialist) | Either works | Validate on `Set` | Opaque or helpers | Return error | Feasible rules only |
| **Mina** (Linter) | Encoded string | Either | **Public map** + lint | Return error | **Strong enforcement** |
| **Sam** (Minimalist) | Encoded string | `any` (no validation) | **Public map** + helpers | Return error | **Minimal** |
| **Jordan** (API Consumer) | Don't care | Don't care | Simple syntax | **Panic** (bugs) | Warn, don't block |
| **Casey** (Maintainer) | Prefer structured | `json.RawMessage` | Opaque (ergonomic) | Two APIs | Strong + configurable |
| **Ravi** (Runtime Boundary) | Either | Serializable (yes) | Either | Fail-fast | Moderate |

**Key insights from table:**
- **No consensus on opaque vs public map** (split 4-4)
- **Strong consensus on typed keys** `Key[T]` (all support)
- **Split on `json.RawMessage`** (4 prefer, 3 prefer `any` + validate)
- **Split on error handling** (3 prefer two APIs, 2 prefer panic, 3 prefer error)

---

## Related Documents

### Debate Materials
- Candidates: `reference/01-debate-candidates-typed-turn-data-metadata.md`
- Questions: `reference/02-debate-questions-typed-turn-data-metadata.md`
- Round 1 (Q1-3): `reference/08-debate-round-1-q1-3-typed-accessors.md`
- Round 2 (Q4-6): `reference/09-debate-round-2-q4-6-key-identity.md`
- Round 3 (Q7-9): `reference/10-debate-round-3-q7-9-api-surface.md`
- Round 4 (Q10,Q12): `reference/11-debate-round-4-q10-q12-serializability-failures.md`
- Round 5 (Q13-14): `reference/12-debate-round-5-q13-14-tooling-schema.md`

### Analysis
- Original analysis: `analysis/01-opaque-turn-data-typed-get-t-accessors.md`

### Next Documents to Create
- **RFC**: Detailed proposal based on chosen approach
- **Prototype report**: Findings from Phase 1 prototype
- **Migration guide**: Step-by-step migration for existing code
