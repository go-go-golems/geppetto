---
Title: Debate Round 3 (Q7-9 API surface)
Ticket: 001-TYPED-ACCESSOR-TURN-DATA
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/turns/types.go
      Note: Current Turn struct with public map field (line 99)
    - Path: pkg/turns/serde/serde.go
      Note: Map initialization pattern (line 25)
    - Path: pkg/inference/toolhelpers/helpers.go
      Note: Direct map write pattern (lines 297, 304)
    - Path: pkg/analysis/turnsdatalint/analyzer.go
      Note: Current linter enforcement (typed-key expressions)
ExternalSources: []
Summary: "Debate Round 3: Should Turn.Data be opaque or stay a map? What should we forbid with typed Key[T]? What escape hatches for iteration/persistence?"
LastUpdated: 2025-12-20T01:45:00-05:00
WhatFor: Explore API surface choices (opaque wrapper vs map, linting vs type system, iteration escape hatches) before committing to a design.
WhenToUse: Reference when drafting RFC or reviewing proposals for Turn.Data API surface and access patterns.
---

# Debate Round 3: API Surface and Escape Hatches (Q7-Q9)

## Participants

- **Asha "Strong Types"** (Type-Safety Maximalist)
- **Sam "Small Surface Area"** (API Minimalist)
- **Mina "Make the Linter a Boundary"** (Tooling / Lint Enforcer)
- **Casey "Code Review"** (Maintainer / reviewer / bug-triage)
- **Jordan "Just Let Me Ship"** (Application Engineer / API consumer)

## Questions

7. **Do we want `Turn.Data` to remain a map field or become an opaque type?**
8. **If we introduce typed `Key[T]`, what should we forbid?**
9. **What is the "escape hatch" story?**

---

## Pre-Debate Research

### Research: Current map access patterns

**Query:** How is `Turn.Data` currently accessed?

From `geppetto/pkg/turns/types.go:99`:
```go
// Data stores the application data payload associated with this turn
Data map[TurnDataKey]interface{} `yaml:"data,omitempty"`
```

**Finding:** `Turn.Data` is a **public map field**. Anyone can read/write it directly.

---

**Query:** Where do we initialize the map?

```bash
$ cd geppetto && grep -n 'Data = map\[' pkg/turns/serde/serde.go pkg/inference/toolhelpers/helpers.go
pkg/turns/serde/serde.go:25:		t.Data = map[turns.TurnDataKey]any{}
pkg/inference/toolhelpers/helpers.go:297:		t.Data = map[turns.TurnDataKey]any{}
```

**Finding:** Map initialization is **scattered** across at least 2 files. Each call site checks `if t.Data == nil` and initializes.

---

**Query:** Where do we write to the map?

From `geppetto/pkg/inference/toolhelpers/helpers.go:304`:
```go
t.Data[turns.DataKeyToolConfig] = engine.ToolConfig{
    Enabled:            true,
    ToolChoice:         engine.ToolChoiceAuto,
    MaxIterations:      3,
    MaxParallelTools:   1,
    ToolErrorHandling:  engine.ToolErrorContinue,
}
```

**Finding:** Direct map writes (`t.Data[key] = value`) are common. No helper functions.

---

**Query:** Does anyone iterate over `Turn.Data`?

```bash
$ cd geppetto && grep -r 'for.*range.*\.Data\[' pkg/
```

**Finding:** **No iteration** found in current Geppetto code. (Historically, persistence code iterated to special-case the tool registry, but that's been removed since the registry moved to `context.Context`.)

---

**Query:** What does the linter enforce?

From `geppetto/pkg/analysis/turnsdatalint/analyzer.go:32-39`:
```
// Analyzer enforces that Turn/Run/Block typed-key map access uses a typed key expression,
// and that Block.Payload (map[string]any) uses const string keys.
//
// For typed-key maps, this prevents raw string drift (e.g. t.Data["foo"]) while still allowing
// normal Go patterns like typed conversions, variables, and parameters.
```

**Finding:** The linter prevents **raw string indexing** but **does not** prevent:
- Ad-hoc key construction (`turns.TurnDataKey("oops")`).
- Direct map writes (`t.Data[key] = value`).
- Nil map panics (if `Data` is nil and you write to it).

---

## Opening Statements

### Asha "Strong Types"

I looked at the current code, and here's what I see: **no encapsulation**.

`Turn.Data` is a public map field. Anyone can write:
```go
t.Data = nil  // oops
t.Data[turns.TurnDataKey("oops")] = "bad"  // linter allows this
```

The linter prevents `t.Data["oops"]` (raw string), but it **doesn't** prevent `turns.TurnDataKey("oops")` (typed conversion). That's a loophole.

**My answer to Q7:** `Turn.Data` should be **opaque**. Change it from a public map to a wrapper:
```go
type Turn struct {
    Data Data `yaml:"data,omitempty"`  // Data is a wrapper, not a map
}

type Data struct {
    m map[TurnDataKeyID]json.RawMessage  // private
}
```

Minimal API:
- `Get[T](Key[T]) (T, bool, error)` — typed read
- `Set[T](Key[T], T) error` — typed write
- `Delete(TurnDataKeyID)` — remove entry
- `Range(func(TurnDataKeyID, json.RawMessage) bool)` — iterate for persistence
- `Len() int` — size

**Q8:** If we introduce typed `Key[T]`, we should **forbid**:
- Ad-hoc key construction (`TurnDataKey("oops")`).
- Keys defined outside canonical packages (`geppetto/pkg/turns`, `moments/backend/pkg/turnkeys`).

The linter should enforce: "All keys must be canonical vars defined in a keys package."

**Q9:** The escape hatch is `Range`. Persistence code can iterate over raw `json.RawMessage` values without decoding them. That's fast and safe.

Making `Data` opaque centralizes initialization (no more scattered `if t.Data == nil { ... }`), enforces typed access, and makes it impossible to bypass the API.

---

### Sam "Small Surface Area"

Asha, I hear you, but I think you're solving a problem we don't have.

Look at the current usage: `t.Data[key] = value`. That's **simple**. Everyone understands it. It's Go's standard map syntax.

If we make `Data` opaque, every call site becomes:
```go
if err := t.Data.Set(turns.KeyToolConfig, cfg); err != nil {
    return err
}
```

That's **more code** for the same operation. And now I have to handle an error that I didn't have to handle before.

**My answer to Q7:** Keep `Turn.Data` as a **map field**, but add **helper functions**:
```go
func SetData[T any](t *Turn, key Key[T], value T) error {
    if t.Data == nil {
        t.Data = map[TurnDataKey]any{}
    }
    // validate serializability if needed
    t.Data[key.id] = value
    return nil
}

func GetData[T any](t *Turn, key Key[T]) (T, bool, error) {
    if t.Data == nil {
        var zero T
        return zero, false, nil
    }
    v, ok := t.Data[key.id]
    if !ok {
        var zero T
        return zero, false, nil
    }
    // type assertion or decode
    return v.(T), true, nil
}
```

This gives you **typed access** without breaking the existing pattern. Call sites can choose:
- `t.Data[key] = value` (direct, for simple cases).
- `SetData(t, key, value)` (helper, for validation/error handling).

**Q8:** If we introduce typed `Key[T]`, we should **forbid** ad-hoc key construction via **linting**, not the type system. The linter should enforce: "All keys must be canonical consts/vars."

**Q9:** The escape hatch is: **the map is public**. If you need to iterate, just do `for k, v := range t.Data { ... }`. No special API needed.

The advantage of this approach: **incremental adoption**. Existing code keeps working. New code can use helpers. We don't force a big refactor.

---

### Mina "Make the Linter a Boundary"

I'm with Sam on keeping the map public, but I want to strengthen the linter.

Right now, the linter prevents `t.Data["oops"]` but **allows** `t.Data[turns.TurnDataKey("oops")]`. That's a loophole.

**My answer to Q7:** Keep `Turn.Data` as a **map field**, but add a **new lint rule**:
- "All `TurnDataKey` values must be canonical consts/vars defined in a keys package."
- "Forbid ad-hoc key construction (`TurnDataKey(...)`) outside keys packages."

This is similar to how we enforce `context.Context` keys: you can't just do `context.WithValue(ctx, "oops", value)` because the linter flags it. You have to use a typed key.

**Q8:** We should forbid:
- Ad-hoc key construction (via linting).
- Keys defined in random files (via linting: "keys must live in `*/keys.go` or `*/turnkeys/*.go`").

**Q9:** The escape hatch is: **the map is public**. But we should add a **helper** for safe iteration:
```go
func RangeData(t *Turn, fn func(TurnDataKey, any) bool) {
    if t.Data == nil {
        return
    }
    for k, v := range t.Data {
        if !fn(k, v) {
            return
        }
    }
}
```

This makes iteration explicit and gives us a place to add logging/metrics if needed.

The key insight: **linting is enforcement**. We don't need an opaque wrapper to prevent bad patterns—we just need better lint rules.

---

### Casey "Code Review"

I review a lot of PRs, and here's what I care about: **reviewability** and **fail-fast**.

Right now, when I see:
```go
t.Data[turns.DataKeyToolConfig] = cfg
```

I have to ask: "Is `t.Data` initialized? What if it's nil?" That's a **footgun**.

**My answer to Q7:** I prefer **opaque** (`Data` wrapper), but I want it to be **ergonomic**:
```go
t.Data.Set(turns.KeyToolConfig, cfg)  // no error handling for simple cases
```

If `Set` needs to return an error (for serializability validation), then fine. But don't make me write:
```go
if err := t.Data.Set(key, value); err != nil {
    return fmt.Errorf("set turn data: %w", err)
}
```

...for every single write. That's too much ceremony.

Maybe we have **two APIs**:
- `Set(key, value)` — panics on error (for "this should never fail" cases).
- `TrySet(key, value) error` — returns error (for "validate this" cases).

**Q8:** If we introduce typed `Key[T]`, we should forbid:
- Ad-hoc key construction (via linting).
- Keys defined outside canonical packages (via linting).
- **Deprecated keys** (via linting: "KeyPersonID is deprecated; use KeyPersonIDV2").

**Q9:** The escape hatch is `Range`, but I want **two variants**:
- `Range(func(key, json.RawMessage) bool)` — for persistence (fast, no decode).
- `RangeTyped(func(key, any) bool)` — for debugging (decodes each value).

This gives us flexibility: persistence code uses the fast path, debugging tools use the typed path.

Also, I want the wrapper to **log warnings** when it detects problems:
- "Key X has no value (nil or empty)."
- "Key X failed to decode as T."

This helps catch bugs in tests/staging before they hit production.

---

### Jordan "Just Let Me Ship"

Okay, I'm the person who actually writes application code. Let me tell you what I care about:

**Q7:** I don't care if `Data` is a map or a wrapper, as long as the **syntax is simple**. Right now, I do:
```go
t.Data[turns.DataKeyToolConfig] = cfg
```

If we make it opaque, I'll do:
```go
t.Data.Set(turns.KeyToolConfig, cfg)
```

That's fine. But if I have to write:
```go
if err := t.Data.Set(turns.KeyToolConfig, cfg); err != nil {
    log.Warn().Err(err).Msg("failed to set tool config")
    // now what? do I return? do I continue?
}
```

...then I'm going to be **annoyed**. Most of the time, setting a config value **should not fail**. If it does fail (e.g., serializability check), that's a **bug in my code**, not a runtime condition I need to handle gracefully.

So I'm with Casey: give me a `Set` that panics on error, and a `TrySet` that returns an error for cases where I actually need to handle it.

**Q8:** If we introduce typed `Key[T]`, I want the linter to prevent:
- Ad-hoc key construction (`TurnDataKey("oops")`).
- Using deprecated keys (warn me if I use `KeyPersonID` instead of `KeyPersonIDV2`).

But I **don't** want the linter to prevent me from defining keys in my own package. If I'm working on a feature and I need a temporary key for testing, I should be able to define it locally:
```go
const MyTestKey turns.TurnDataKey = "test.my_feature"
```

The linter can warn me if it doesn't follow the naming convention, but it shouldn't **block** me.

**Q9:** The escape hatch is: **I don't need one**. I'm never going to iterate over `Turn.Data` in application code. That's a persistence/debugging concern. So I don't care if the map is public or if there's a `Range` function.

But if we make `Data` opaque, **please** give me a way to inspect it in tests:
```go
assert.Equal(t, cfg, t.Data.MustGet(turns.KeyToolConfig))  // panics if not found
```

I don't want to write:
```go
cfg, ok, err := t.Data.Get(turns.KeyToolConfig)
require.NoError(t, err)
require.True(t, ok)
assert.Equal(t, expectedCfg, cfg)
```

That's too much boilerplate for a test.

---

## Rebuttals

### Asha responds to Sam

Sam, you said "we're solving a problem we don't have." But we **do** have a problem: **nil map panics**.

Look at `geppetto/pkg/turns/serde/serde.go:25` and `geppetto/pkg/inference/toolhelpers/helpers.go:297`. Both files have:
```go
if t.Data == nil {
    t.Data = map[turns.TurnDataKey]any{}
}
```

That's **defensive programming** because the map is public and can be nil. If we make `Data` opaque, that boilerplate goes away. The wrapper handles initialization internally.

Also, you said "every call site becomes `if err := t.Data.Set(...); err != nil { ... }`". But that's only if `Set` returns an error. We can make `Set` **not** return an error for common cases:
```go
func (d *Data) Set[T any](key Key[T], value T) {
    d.ensure()  // initialize map if nil
    d.m[key.id] = value
}
```

No error, no ceremony. The only time we return an error is if we're validating serializability, and that's **optional** (we can add a `TrySet` variant).

---

### Sam responds to Asha

Asha, I agree that nil map panics are annoying. But here's the thing: **we already have a solution** for that. It's called `serde.NormalizeTurn`:
```go
func NormalizeTurn(t *turns.Turn) {
    if t.Data == nil {
        t.Data = map[turns.TurnDataKey]any{}
    }
    // ...
}
```

Just call `NormalizeTurn` once (e.g., after loading from YAML), and you're done. No nil panics.

If we make `Data` opaque, we're forcing **every call site** to use the wrapper API, even if they don't need typed access or validation. That's a **big change** for a small benefit.

Also, about your "no error, no ceremony" argument: if `Set` doesn't return an error, then how do you validate serializability? You can't. So either:
- `Set` always validates (and returns an error).
- `Set` never validates (and we're back to the current problem: non-serializable values can sneak in).

You can't have it both ways.

---

### Mina responds to Casey

Casey, you want two `Range` variants: one for raw bytes, one for typed values. That's a good idea, but here's my worry: **which one do people use?**

If `Range(func(key, json.RawMessage) bool)` is the default, then most code will use it (because it's simpler). But then we lose the benefit of typed access.

If `RangeTyped(func(key, any) bool)` is the default, then we're decoding every value just to iterate, which is **slow** for persistence.

I think we should have **one** `Range` function that exposes the raw map:
```go
func (d Data) Range(fn func(TurnDataKey, any) bool) {
    for k, v := range d.m {
        if !fn(k, v) {
            return
        }
    }
}
```

If you need typed access, use `Get[T]` inside the loop:
```go
t.Data.Range(func(k TurnDataKey, v any) bool {
    if k == turns.DataKeyToolConfig {
        cfg, ok, err := t.Data.Get(turns.KeyToolConfig)
        if err == nil && ok {
            // use cfg
        }
    }
    return true
})
```

That's more flexible and doesn't force a choice between "fast" and "typed".

---

### Casey responds to Jordan

Jordan, you said "I don't want the linter to block me from defining keys locally for testing." I get that, but here's the problem: **how do we prevent those temporary keys from leaking into production?**

If you define:
```go
const MyTestKey turns.TurnDataKey = "test.my_feature"
```

...and then use it in a test, that's fine. But what if someone else copies your test code into production code? Now we have an ad-hoc key in production.

I think the linter should **warn** (not block) on keys defined outside canonical packages. And we should have a **naming convention** for test keys:
- Production keys: `"namespace.slug@vN"` (e.g., `"mento.person_id@v1"`).
- Test keys: `"test.namespace.slug"` (e.g., `"test.mento.my_feature"`).

The linter can allow `"test.*"` keys in test files only.

About your "MustGet for tests" idea: I like it. We should have:
- `Get[T](key) (T, bool, error)` — for production (returns error).
- `MustGet[T](key) T` — for tests (panics if not found or decode fails).

That makes test assertions cleaner.

---

### Jordan responds to Mina

Mina, you said "linting is enforcement." I agree, but here's my worry: **linting is slow**.

Right now, I can write:
```go
t.Data[turns.DataKeyToolConfig] = cfg
```

...and the compiler tells me immediately if I made a mistake (wrong type for `cfg`, typo in `DataKeyToolConfig`, etc.).

If we rely on linting to catch ad-hoc key construction, I have to:
1. Write the code.
2. Run `make lint`.
3. Wait for the linter to finish.
4. Fix the error.
5. Repeat.

That's a **slower feedback loop** than compile-time errors.

With Asha's opaque wrapper + typed keys, the **compiler** catches mistakes:
```go
t.Data.Set(turns.KeyToolConfig, "oops")  // compile error: wrong type
```

That's **instant feedback**. I prefer that over linting.

---

## Moderator Summary

### Key Tensions

1. **Opaque wrapper vs public map**
   - **Asha**: Opaque for encapsulation, centralized initialization, typed access.
   - **Sam**: Public map for simplicity, incremental adoption, no breaking changes.
   - **Mina**: Public map + stronger linting (forbid ad-hoc keys).
   - **Casey**: Opaque, but with ergonomic API (no ceremony).
   - **Jordan**: Don't care, as long as syntax is simple.
   - **Tension**: Opaque is safer but requires refactor; public map is simpler but relies on linting.

2. **Error handling for `Set`**
   - **Asha**: `Set` doesn't return error (or has `TrySet` variant).
   - **Sam**: If `Set` doesn't return error, can't validate serializability.
   - **Casey**: Two APIs: `Set` (panics) and `TrySet` (returns error).
   - **Jordan**: `Set` should panic on error (bugs, not runtime conditions).
   - **Tension**: Ergonomics (no error handling) vs safety (validate serializability).

3. **Linting vs type system**
   - **Asha, Jordan**: Type system is better (compile-time errors, instant feedback).
   - **Mina, Sam**: Linting is sufficient (no breaking changes, incremental).
   - **Tension**: Compile-time enforcement (requires opaque wrapper) vs runtime enforcement (linting).

4. **Escape hatches for iteration**
   - **Asha**: `Range(func(key, json.RawMessage) bool)` for persistence.
   - **Sam**: Public map, no special API needed.
   - **Mina**: `Range(func(key, any) bool)` exposes raw map.
   - **Casey**: Two variants (`Range` for raw bytes, `RangeTyped` for typed values).
   - **Tension**: Fast (raw bytes) vs typed (decode on iterate).

### Interesting Ideas

- **Two `Set` APIs** (Casey, Jordan): `Set` (panics on error) and `TrySet` (returns error).
- **`MustGet` for tests** (Jordan, Casey): Panics if key not found or decode fails.
- **Test key naming convention** (Casey): `"test.namespace.slug"` allowed in test files only.
- **Linter warns on ad-hoc keys** (Mina, Casey): Forbid `TurnDataKey("oops")` outside keys packages.
- **Incremental adoption** (Sam): Keep public map, add helpers, migrate gradually.
- **Centralized initialization** (Asha): Opaque wrapper handles `if t.Data == nil { ... }` internally.

### Open Questions

1. **Should `Set` return an error, or should we have two APIs (`Set` + `TrySet`)?**
   - Casey/Jordan: Two APIs (panics vs returns error).
   - Sam: If `Set` doesn't return error, can't validate serializability.

2. **How do we validate serializability without making every call site handle errors?**
   - Option 1: `Set` validates and panics on error (fail-fast).
   - Option 2: `Set` doesn't validate; `TrySet` validates and returns error.
   - Option 3: Validation is optional (enabled via a flag or wrapper variant).

3. **What's the migration path if we make `Data` opaque?**
   - Do we provide a compatibility shim (e.g., `Turn.Data` becomes a method that returns a map view)?
   - Do we break the API cleanly and update all call sites?

4. **How do we prevent test keys from leaking into production?**
   - Casey: Naming convention (`"test.*"`) + linter rule (only in test files).
   - Jordan: Linter should warn, not block.

5. **Should `Range` expose raw bytes or typed values?**
   - Asha: Raw bytes (`json.RawMessage`) for fast persistence.
   - Mina: Typed values (`any`) for flexibility.
   - Casey: Two variants (fast + typed).

### Next Steps

- **Prototype**: Build a small proof-of-concept with both approaches:
  1. Opaque wrapper with `Set/Get/Range` and typed keys.
  2. Public map with helper functions and stronger linting.
- **Measure**: Compare ergonomics (lines of code, error handling ceremony) and safety (what mistakes are prevented).
- **RFC**: Draft a proposal based on findings, including migration path.

---

## Related

- Candidates: `reference/01-debate-candidates-typed-turn-data-metadata.md`
- Questions: `reference/02-debate-questions-typed-turn-data-metadata.md`
- Debate Round 1: `reference/08-debate-round-1-q1-3-typed-accessors.md`
- Debate Round 2: `reference/09-debate-round-2-q4-6-key-identity.md`
- Ticket analysis: `analysis/01-opaque-turn-data-typed-get-t-accessors.md`
