---
Title: Debate Round 4 (Q10,Q12 serializability+failures)
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
    - Path: pkg/turns/serde/serde.go
      Note: Current YAML marshal/unmarshal (lines 59, 65)
    - Path: pkg/turns/types.go
      Note: Turn/Block struct definitions with any-typed maps
    - Path: pkg/inference/toolcontext/toolcontext.go
      Note: Tool registry lives in context (not Turn.Data)
ExternalSources: []
Summary: "Debate Round 4: Should Turn.Data enforce serializable-only values structurally? What's the right failure mode when invariants are violated?"
LastUpdated: 2025-12-20T02:15:00-05:00
WhatFor: Explore serializability enforcement strategies and failure mode choices (fail-fast vs fail-late, panic vs error vs log) before committing to a design.
WhenToUse: Reference when drafting RFC or reviewing proposals for Turn.Data serializability guarantees and error handling.
---

# Debate Round 4: Serializability and Failure Modes (Q10, Q12)

## Participants

- **Noel "Everything Persistable"** (Serialization Purist)
- **Priya "Go Specialist"** (Go generics + encoding expert)
- **Jordan "Just Let Me Ship"** (Application Engineer / API consumer)
- **Casey "Code Review"** (Maintainer / reviewer / bug-triage)
- **Ravi "Runtime Stays in Context"** (Runtime Boundary Advocate)

## Questions

10. **Should `Turn.Data` enforce serializable-only values structurally?**
12. **What's the right failure mode when an invariant is violated?**

**Note:** Question 11 (runtime-only attachments) is **not included** in this round because the codebase already solved this: runtime objects (tool registry) live in `context.Context`, not `Turn.Data`.

---

## Pre-Debate Research

### Research: Current serializability story

**Query:** How does `Turn.Data` get serialized today?

From `geppetto/pkg/turns/serde/serde.go:59`:
```go
return yaml.Marshal(snapshot)
```

**Finding:** Turns are serialized with `yaml.Marshal`. If a value in `Turn.Data` is **not serializable**, the marshal will **fail at runtime** (when you try to save the turn).

---

**Query:** What types are currently stored in `Turn.Data`?

```bash
$ cd geppetto && grep -r 'Data\[.*\] =' pkg/ | head -10
pkg/inference/toolhelpers/helpers.go:304:	t.Data[turns.DataKeyToolConfig] = engine.ToolConfig{
```

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

**Finding:** `engine.ToolConfig` is a **struct** with exported fields. It's serializable.

---

**Query:** Are there any non-serializable values in `Turn.Data` today?

```bash
$ cd geppetto && grep -r 'DataKeyToolRegistry' pkg/
```

**Finding:** **No matches**. The `DataKeyToolRegistry` key (which stored a `tools.ToolRegistry` interface) was **removed**. Tool registries now live in `context.Context` (see `geppetto/pkg/inference/toolcontext/toolcontext.go`).

---

**Query:** What happens if you try to serialize a non-serializable value?

Test case:
```go
t := &turns.Turn{
    Data: map[turns.TurnDataKey]any{
        "bad_key": make(chan int),  // channels are not serializable
    },
}
_, err := serde.ToYAML(t, serde.Options{})
// err will be: yaml: unsupported type: chan int
```

**Finding:** `yaml.Marshal` **fails at runtime** with an error. There's no compile-time or early validation.

---

## Opening Statements

### Noel "Everything Persistable"

I looked at the current code, and here's what I see: **no serializability guarantee**.

Right now, nothing stops you from writing:
```go
t.Data[someKey] = make(chan int)
```

The code compiles. The tests pass (if they don't serialize). But when you try to save the turn to YAML, **boom**: `yaml: unsupported type: chan int`.

That's **fail-late**. You don't find out until you try to persist the turn, which might be in production.

**My answer to Q10:** `Turn.Data` should enforce serializable-only values **structurally**. Here's how:

**Option A: Validate on `Set` by marshaling**
```go
func (d *Data) Set[T any](key Key[T], value T) error {
    // Check if value is serializable
    if _, err := json.Marshal(value); err != nil {
        return fmt.Errorf("value for key %s is not serializable: %w", key, err)
    }
    d.m[key.id] = value
    return nil
}
```

This is **fail-fast**: you find out immediately when you try to set a non-serializable value.

**Option B: Store canonical serialized values internally**
```go
type Data struct {
    m map[TurnDataKeyID]json.RawMessage  // store JSON bytes
}

func (d *Data) Set[T any](key Key[T], value T) error {
    b, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("failed to serialize value for key %s: %w", key, err)
    }
    d.m[key.id] = b
    return nil
}

func (d Data) Get[T any](key Key[T]) (T, bool, error) {
    var zero T
    b, ok := d.m[key.id]
    if !ok {
        return zero, false, nil
    }
    if err := json.Unmarshal(b, &zero); err != nil {
        return zero, true, fmt.Errorf("failed to deserialize value for key %s: %w", key, err)
    }
    return zero, true, nil
}
```

This is **structurally guaranteed**: if `Set` succeeds, the value **is** serializable (because it's stored as JSON bytes).

I prefer **Option B** because it's airtight. Option A is weaker (you could store a value that marshals to JSON but not to YAML, or vice versa).

**Q12:** The right failure mode is **fail-fast at `Set` time**. If you try to set a non-serializable value, `Set` returns an error immediately. Don't wait until serialization time to discover the problem.

---

### Priya "Go Specialist"

Noel, I agree with your goal (serializability guarantee), but let's talk about the **trade-offs** of your two options.

**Option A: Validate on `Set` by marshaling**
- **Pro**: Fail-fast (errors at write time).
- **Con**: You marshal the value **twice**: once to validate, once to serialize the turn. That's wasteful.
- **Con**: You're validating with JSON, but serializing with YAML. What if the value marshals to JSON but not to YAML? (Rare, but possible.)

**Option B: Store `json.RawMessage` internally**
- **Pro**: Structurally guaranteed (if `Set` succeeds, value is serializable).
- **Pro**: Only marshal once (at `Set` time).
- **Con**: Every `Get` does a JSON unmarshal. That's **allocations** and **CPU** on every read.
- **Con**: You lose type information. If you store `engine.ToolConfig` and then `Get` it as `map[string]any`, the unmarshal succeeds (wrong type, but valid JSON).

**My answer to Q10:** I prefer **Option A** (validate on `Set`), but with a **refinement**:
- Use the **same marshaler** for validation and serialization (e.g., always use JSON, or always use YAML).
- Make validation **optional** (via a flag or wrapper variant) so hot paths can skip it.

**Q12:** The right failure mode depends on **who's calling**:
- **Application code** (setting a config): fail-fast at `Set` time (return error).
- **Persistence code** (loading from YAML): fail-fast at load time (return error if a value can't be decoded).
- **Debugging code** (inspecting a turn): fail gracefully (log warning, return partial data).

We can't have one failure mode for all cases. We need **context-aware** error handling.

---

### Jordan "Just Let Me Ship"

Okay, I'm the person who actually writes application code. Let me tell you what I care about:

**Q10:** I don't want to think about serializability when I'm setting a config value. I just want to write:
```go
t.Data.Set(turns.KeyToolConfig, engine.ToolConfig{Enabled: true})
```

If that value is **not** serializable, that's a **bug in my code**. It should **panic** (fail-fast), not return an error that I have to handle.

Here's why: if `engine.ToolConfig` is not serializable, that's a **design problem**, not a runtime condition. I can't "handle" it gracefully—I need to fix the code.

So I'm with Noel on **fail-fast**, but I want it to **panic**, not return an error:
```go
func (d *Data) Set[T any](key Key[T], value T) {
    b, err := json.Marshal(value)
    if err != nil {
        panic(fmt.Sprintf("value for key %s is not serializable: %v", key, err))
    }
    d.m[key.id] = b
}
```

If I'm in a case where I **do** need to handle the error (e.g., user-provided data), I can use a `TrySet` variant:
```go
func (d *Data) TrySet[T any](key Key[T], value T) error {
    b, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("value for key %s is not serializable: %w", key, err)
    }
    d.m[key.id] = b
    return nil
}
```

**Q12:** The right failure mode is:
- **`Set`**: panic on error (bugs, not runtime conditions).
- **`TrySet`**: return error (for user-provided data).
- **`Get`**: return error (decode failures are runtime conditions, not bugs).

This gives me the best of both worlds: simple API for common cases, error handling for edge cases.

---

### Casey "Code Review"

I review a lot of PRs, and here's what I care about: **debuggability** and **fail-fast**.

**Q10:** I prefer Noel's **Option B** (store `json.RawMessage`), but with one addition: **log warnings** when we detect problems.

For example, if `Set` succeeds but the value is "weird" (e.g., a struct with unexported fields, or a value that round-trips differently), log a warning:
```go
func (d *Data) Set[T any](key Key[T], value T) error {
    b, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("value for key %s is not serializable: %w", key, err)
    }
    // Round-trip check (optional, for debugging)
    var check T
    if err := json.Unmarshal(b, &check); err != nil {
        log.Warn().Str("key", key.String()).Msg("value round-trip failed")
    }
    d.m[key.id] = b
    return nil
}
```

This helps catch bugs in tests/staging before they hit production.

**Q12:** The right failure mode is:
- **`Set`**: return error (so callers can decide: panic, log, or handle).
- **`Get`**: return error (decode failures are actionable).
- **Serialization** (YAML): return error (but this should never fail if `Set` validated).

I **don't** want `Set` to panic by default (like Jordan suggested) because that makes it hard to write tests. If I'm testing error cases, I don't want my test to panic—I want to check the error.

But I'm okay with a `MustSet` variant that panics (for cases where failure is truly a bug).

---

### Ravi "Runtime Stays in Context"

I want to talk about the **boundary** between persisted state and runtime state.

**Q10:** I agree that `Turn.Data` should be serializable. But here's the key insight: **we already solved the runtime-only problem**.

Look at `geppetto/pkg/inference/toolcontext/toolcontext.go`:
```go
func WithRegistry(ctx context.Context, reg tools.ToolRegistry) context.Context {
    return context.WithValue(ctx, ctxKey{}, reg)
}
```

The tool registry (a runtime-only object) lives in `context.Context`, **not** in `Turn.Data`. That's the right pattern.

So the question is: **are there any other runtime-only objects we need to support?**

My answer: **no**. If you have a runtime-only object, put it in `context.Context`. Don't put it in `Turn.Data`.

This makes the serializability requirement **simple**: `Turn.Data` is **always** serializable, by definition. No special cases, no escape hatches.

**Q12:** The right failure mode is:
- **`Set`**: validate serializability and return error (fail-fast).
- **`Get`**: return error if decode fails (runtime condition).
- **Context boundary**: if you try to put a runtime-only object in `Turn.Data`, `Set` rejects it immediately.

The key is: **make the boundary explicit**. `Turn.Data` is persisted state. `context.Context` is runtime state. Don't mix them.

---

## Rebuttals

### Noel responds to Priya

Priya, you said "Option B has a con: every `Get` does a JSON unmarshal." That's true, but here's the thing: **we're already doing that**.

Look at `geppetto/pkg/turns/serde/serde.go:65`:
```go
if err := yaml.Unmarshal(b, &t); err != nil {
    return nil, err
}
```

When you load a turn from YAML, you're unmarshaling **all** the values. So the "unmarshal on `Get`" cost is **not new**—it's just moved from "load time" to "read time".

And in many cases, that's **better**: if you load a turn with 10 keys but only read 2 of them, you've saved 8 unmarshals.

Also, about your "lose type information" concern: that's why we have **typed keys** `Key[T]`. The key **carries** the type information, so `Get` knows what type to unmarshal into.

---

### Priya responds to Noel

Noel, you're right that we're already unmarshaling at load time. But here's the difference:

**Current (load from YAML):**
- Unmarshal **once** (when loading the turn).
- Values are stored as Go objects (`engine.ToolConfig`).
- Reads are **fast** (just a map lookup + type assertion).

**Option B (store JSON bytes):**
- Unmarshal **every time** you read (on every `Get`).
- Values are stored as `json.RawMessage`.
- Reads are **slow** (map lookup + JSON unmarshal).

So if you read the same key **multiple times** (e.g., in a loop), you're unmarshaling it **multiple times**. That's wasteful.

One way to mitigate this: **cache decoded values**:
```go
type Data struct {
    m     map[TurnDataKeyID]json.RawMessage  // canonical storage
    cache map[TurnDataKeyID]any              // decoded cache
}

func (d Data) Get[T any](key Key[T]) (T, bool, error) {
    // Check cache first
    if cached, ok := d.cache[key.id]; ok {
        return cached.(T), true, nil
    }
    // Unmarshal from JSON
    var zero T
    b, ok := d.m[key.id]
    if !ok {
        return zero, false, nil
    }
    if err := json.Unmarshal(b, &zero); err != nil {
        return zero, true, err
    }
    // Cache for next read
    d.cache[key.id] = zero
    return zero, true, nil
}
```

But now you have **cache invalidation** problems. When do you clear the cache? What if the value changes?

---

### Jordan responds to Casey

Casey, you said "I don't want `Set` to panic by default because that makes it hard to write tests." But here's my counter: **if your test expects a panic, use `require.Panics`**:
```go
require.Panics(t, func() {
    t.Data.Set(turns.KeyBad, nonSerializableValue)
})
```

That's **one line** of test code. Compare that to the **production code** ceremony of handling errors at every call site:
```go
if err := t.Data.Set(turns.KeyToolConfig, cfg); err != nil {
    return fmt.Errorf("set tool config: %w", err)
}
```

I'd rather have one line of test code than 3 lines of production code at every call site.

Also, about your "round-trip check" idea: I like it, but it should be **opt-in** (e.g., enabled in tests/staging, disabled in production). Otherwise, we're doing **two** marshals (encode + decode) on every `Set`, which is expensive.

---

### Casey responds to Jordan

Jordan, you're right that `require.Panics` is one line. But here's the problem: **what if the panic message is wrong?**

With an error, I can check the error message:
```go
err := t.Data.TrySet(turns.KeyBad, nonSerializableValue)
require.Error(t, err)
assert.Contains(t, err.Error(), "not serializable")
```

With a panic, I can't easily check the panic message. I have to use `recover()` and inspect the panic value, which is **messy**.

Also, about panics in production: if `Set` panics, it **crashes the process** (unless you have a top-level `recover()`). That's a **bad user experience**. I'd rather return an error and let the caller decide how to handle it (log, retry, fail gracefully).

But I'm okay with a `MustSet` variant that panics, for cases where failure is truly a bug.

---

### Ravi responds to Priya

Priya, you're worried about the cost of unmarshaling on every `Get`. But here's the thing: **how often do you actually read the same key multiple times?**

In most code I've seen, you read a key **once** (e.g., at the start of a function) and then use the value. You don't read it in a loop.

If there **are** hot paths where you read the same key multiple times, you can **cache the value locally**:
```go
cfg, ok, err := t.Data.Get(turns.KeyToolConfig)
if err != nil || !ok {
    // handle error
}
// Use cfg multiple times in this function
```

That's **explicit** caching, which is better than **implicit** caching (like your cache map idea) because it's clear to the reader that the value is being reused.

Also, about your cache invalidation concern: if `Turn.Data` is **immutable** (you can't change a value after setting it), then you don't need cache invalidation. The cache is always valid.

---

## Moderator Summary

### Key Tensions

1. **Validate vs Store Serialized**
   - **Noel**: Store `json.RawMessage` (structurally guaranteed).
   - **Priya**: Validate on `Set` by marshaling (fail-fast, but less airtight).
   - **Tension**: Structural guarantee (Option B) is airtight but adds unmarshal cost on every `Get`. Validation (Option A) is cheaper but less foolproof.

2. **Panic vs Return Error**
   - **Jordan**: `Set` should panic (bugs, not runtime conditions).
   - **Casey**: `Set` should return error (so callers can decide).
   - **Tension**: Panics are simpler (no error handling) but crash the process. Errors are safer but add ceremony at call sites.
   - **Proposed solution**: Two APIs (`Set` panics, `TrySet` returns error) or (`MustSet` panics, `Set` returns error).

3. **Unmarshal Cost**
   - **Noel**: Unmarshal on `Get` is fine (you're already unmarshaling at load time).
   - **Priya**: Unmarshal on every `Get` is wasteful if you read the same key multiple times.
   - **Proposed solution**: Cache decoded values (but adds complexity) or encourage explicit local caching.

4. **Fail-Fast vs Fail-Late**
   - **Consensus**: Fail-fast at `Set` time (don't wait until serialization).
   - **Noel, Jordan, Casey, Ravi**: Validate serializability on `Set`.
   - **Priya**: Agree, but make validation optional (for hot paths).

### Interesting Ideas

- **Store `json.RawMessage` internally** (Noel): Structurally guaranteed serializability.
- **Two `Set` APIs** (Jordan, Casey): `Set` panics (or returns error), `TrySet` returns error (or `MustSet` panics).
- **Round-trip check** (Casey): Validate that value round-trips correctly (opt-in for debugging).
- **Context boundary** (Ravi): Runtime-only objects live in `context.Context`, not `Turn.Data`.
- **Cache decoded values** (Priya): Mitigate unmarshal cost on repeated `Get` calls.
- **Explicit local caching** (Ravi): Encourage callers to cache values locally if needed.

### Open Questions

1. **Should we store `json.RawMessage` or validate on `Set`?**
   - Noel: Store JSON (airtight).
   - Priya: Validate (cheaper).
   - **Trade-off**: Structural guarantee vs performance.

2. **Should `Set` panic or return error?**
   - Jordan: Panic (bugs, not runtime conditions).
   - Casey: Return error (so callers can decide).
   - **Proposed solution**: Two APIs (`Set` + `TrySet` or `Set` + `MustSet`).

3. **How do we handle the unmarshal cost on `Get`?**
   - Noel/Ravi: It's fine (you're already unmarshaling at load time).
   - Priya: Add caching (but adds complexity).
   - **Open question**: Is the unmarshal cost a real problem in practice?

4. **Should validation be optional (for hot paths)?**
   - Priya: Yes (make it opt-in or opt-out).
   - Noel: No (always validate, or store JSON bytes).

5. **What about YAML vs JSON serializability?**
   - Noel: Store JSON bytes, render as YAML at serialization time.
   - Priya: Validate with the same marshaler you'll use for serialization.
   - **Open question**: Can we assume JSON ⊆ YAML for our use cases?

### Consensus Points

- **Fail-fast at `Set` time**: Don't wait until serialization to discover non-serializable values.
- **Runtime-only objects live in `context.Context`**: Don't put them in `Turn.Data`.
- **Typed keys carry type information**: `Key[T]` tells `Get` what type to unmarshal into.
- **Need two APIs**: One for "this should never fail" (panic or simple), one for "validate this" (return error).

### Next Steps

- **Prototype**: Build a small proof-of-concept with both approaches:
  1. Store `json.RawMessage` internally (Noel's Option B).
  2. Validate on `Set` by marshaling (Priya's Option A).
- **Measure**: Compare performance (unmarshal cost on `Get`) and safety (what mistakes are prevented).
- **RFC**: Draft a proposal based on findings, including error handling strategy (panic vs error).

---

## Related

- Candidates: `reference/01-debate-candidates-typed-turn-data-metadata.md`
- Questions: `reference/02-debate-questions-typed-turn-data-metadata.md`
- Debate Round 1: `reference/08-debate-round-1-q1-3-typed-accessors.md`
- Debate Round 2: `reference/09-debate-round-2-q4-6-key-identity.md`
- Debate Round 3: `reference/10-debate-round-3-q7-9-api-surface.md`
- Ticket analysis: `analysis/01-opaque-turn-data-typed-get-t-accessors.md`
