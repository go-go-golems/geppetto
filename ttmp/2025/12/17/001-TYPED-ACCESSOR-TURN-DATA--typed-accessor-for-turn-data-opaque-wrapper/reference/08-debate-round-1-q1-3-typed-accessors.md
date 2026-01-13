---
Title: Debate Round 1 (Q1-3 typed accessors)
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
    - Path: pkg/analysis/turnsdatalint/analyzer.go
      Note: |-
        Lint enforcement of typed-key expressions
        Linter enforcement (lines 32-39)
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: |-
        Type assertion pattern at line 127-130
        Two-step type assertion pattern (lines 127-130)
    - Path: pkg/turns/serde/serde.go
      Note: |-
        Nil map normalization
        Nil map normalization pattern (line 24)
    - Path: pkg/turns/types.go
      Note: |-
        Current Turn/Block map shapes
        Current Turn/Block map shapes (lines 92-100)
ExternalSources: []
Summary: 'Debate Round 1: What invariants should Turn.Data/Metadata guarantee? Should they match? What UX for type mismatches?'
LastUpdated: 2025-12-20T00:45:00-05:00
WhatFor: Explore the core invariants and error-handling UX for typed access to Turn.Data/Metadata/Block.Metadata before committing to an API design.
WhenToUse: Reference when drafting RFC or reviewing proposals for opaque wrappers / typed accessors.
---


# Debate Round 1: Invariants and Error UX (Q1-Q3)

## Participants

- **Asha "Strong Types"** (Type-Safety Maximalist)
- **Noel "Everything Persistable"** (Serialization Purist)
- **Priya "Go Specialist"** (Go generics + encoding expert)
- **Jordan "Just Let Me Ship"** (Application Engineer / API consumer)
- **Casey "Code Review"** (Maintainer / reviewer / bug-triage)

## Questions

1. **What invariants should `Turn.Data` guarantee at the boundary?**
2. **Should `Turn.Metadata` and `Block.Metadata` have the same invariants as `Turn.Data`?**
3. **What is the correct UX for "key exists but the stored value is the wrong shape"?**

---

## Pre-Debate Research

### Research: Current call-site patterns

**Query:** How do call sites currently access `Turn.Data`?

```bash
$ cd geppetto && grep -n 't\.Data\[' pkg/steps/ai/openai/engine_openai.go
127:		if cfgAny, ok := t.Data[turns.DataKeyToolConfig]; ok && cfgAny != nil {
128:			if cfg, ok := cfgAny.(engine.ToolConfig); ok {
```

**Finding:** OpenAI engine reads `DataKeyToolConfig` with a **two-step pattern**: first check key existence + nil, then type-assert. No error is surfaced if the type assertion fails—the code silently uses a zero `ToolConfig`.

---

**Query:** Where do we initialize nil maps?

```bash
$ cd geppetto && grep -n 'Data == nil' pkg/turns/serde/serde.go pkg/inference/toolhelpers/helpers.go
pkg/turns/serde/serde.go:24:	if t.Data == nil {
pkg/inference/toolhelpers/helpers.go:296:	if t.Data == nil {
```

**Finding:** `serde.NormalizeTurn` (line 24) and `toolhelpers` (line 296) both check for nil maps and initialize them. This boilerplate is scattered.

---

**Query:** What does `turnsdatalint` actually enforce?

```bash
$ cd geppetto && head -50 pkg/analysis/turnsdatalint/analyzer.go | grep -A 5 'Analyzer enforces'
```

**Finding (from analyzer.go:32-39):**
```
// Analyzer enforces that Turn/Run/Block typed-key map access uses a typed key expression,
// and that Block.Payload (map[string]any) uses const string keys.
//
// For typed-key maps, this prevents raw string drift (e.g. t.Data["foo"]) while still allowing
// normal Go patterns like typed conversions, variables, and parameters.
//
// Note: raw string literals like turn.Data["foo"] can compile in Go because untyped string
// constants may be implicitly converted to a defined string type; this analyzer flags them too.
```

**Finding:** The linter prevents raw string indexing but **does not** enforce type correctness of values. It only enforces key types.

---

**Query:** Current type definitions

From `geppetto/pkg/turns/types.go:92-100`:
```go
type Turn struct {
	ID     string  `yaml:"id,omitempty"`
	RunID  string  `yaml:"run_id,omitempty"`
	Blocks []Block `yaml:"blocks"`
	// Metadata stores arbitrary metadata about the turn
	Metadata map[TurnMetadataKey]interface{} `yaml:"metadata,omitempty"`
	// Data stores the application data payload associated with this turn
	Data map[TurnDataKey]interface{} `yaml:"data,omitempty"`
}
```

**Finding:** Both `Data` and `Metadata` are plain `map[TypedKey]interface{}`. No wrapper, no validation.

---

## Opening Statements

### Asha "Strong Types"

I looked at `engine_openai.go:127-130`, and what I see is exactly the problem we're trying to solve. The code does:

```go
if cfgAny, ok := t.Data[turns.DataKeyToolConfig]; ok && cfgAny != nil {
    if cfg, ok := cfgAny.(engine.ToolConfig); ok {
        toolCfg = cfg
    }
}
```

That's **two** nil/ok checks just to read a config struct. If the type assertion fails, we silently get a zero `ToolConfig`—no error, no log, nothing. This is a footgun.

**My answer to Q1:** `Turn.Data` should guarantee:
1. **Values are typed at the boundary** — `Get[T](key)` returns `T`, not `any`.
2. **Type mismatches are surfaced** — if you stored `string` but read as `int`, you get an error, not silent zero.
3. **Nil maps are impossible** — the wrapper handles initialization internally.

**Q2:** Yes, `Turn.Metadata` and `Block.Metadata` should have the **same** invariants. They're all "bags of typed attributes"—the only difference is semantic (what you store), not structural (how you access it).

**Q3:** When the value is the wrong shape, return `(T, ok=true, err=DecodeError)`. The `ok=true` tells you "key exists", the error tells you "but I couldn't decode it as T". This is **debuggable**: you can log the error and see exactly what went wrong.

If we keep the current `map[TurnDataKey]any` shape, we're just pushing this boilerplate into every call site. An opaque wrapper with typed `Get/Set` centralizes it.

---

### Noel "Everything Persistable"

I agree with Asha on one thing: the current pattern is repetitive. But I'm more concerned about what happens when we **serialize** these turns.

Look at `serde.NormalizeTurn` (line 24-26):
```go
if t.Data == nil {
    t.Data = map[turns.TurnDataKey]any{}
}
```

This is defensive programming because we can't trust that `Data` is always initialized. But here's the deeper issue: **we also can't trust that the values inside `Data` are serializable**.

Right now, nothing stops you from doing:
```go
t.Data[someKey] = make(chan int)  // compiles, but yaml.Marshal will fail
```

**My answer to Q1:** `Turn.Data` should guarantee:
1. **Values are serializable** — structurally, not "best-effort". If `Set` succeeds, the value **is** serializable.
2. **Typed reads** — yes, but secondary to serializability.

**Q2:** `Turn.Metadata` and `Block.Metadata` should have the **same** serializability guarantee. The difference is semantic:
- `Data` = application-level hints/config (e.g., tool config, agent mode).
- `Metadata` = provider/runtime annotations (e.g., usage, model, trace ID).

Both get persisted in YAML, so both must be serializable.

**Q3:** For type mismatches, I want `(T, ok=true, err=DecodeError)` like Asha said. But I also want **encode errors** surfaced at `Set` time. If you try to `Set` a non-serializable value, it should fail immediately, not when you try to save the turn later.

The cleanest way to enforce this: store values as `json.RawMessage` internally. Then serializability is guaranteed by construction.

---

### Priya "Go Specialist"

Let me ground this in Go's actual type system, because I'm seeing some wishful thinking.

**Q1 invariants:** I agree we want typed reads, but let's be precise about what Go can and can't do.

If we define:
```go
func (d Data) Get[T any](key TurnDataKey) (T, bool)
```

Go **cannot infer `T`** from the assignment target. Every call site must write:
```go
cfg, ok := t.Data.Get[engine.ToolConfig](turns.DataKeyToolConfig)
```

That's noisy. If we want ergonomic reads, we need **typed keys**:
```go
type Key[T any] struct { id TurnDataKeyID }
func (d Data) Get[T any](k Key[T]) (T, bool, error)
```

Then:
```go
cfg, ok, err := t.Data.Get(turns.KeyToolConfig)  // T inferred from key
```

**Q2:** Should `Metadata` have the same invariants? **Mostly yes**, but with one caveat: `Metadata` is often populated by provider engines, not application code. We need to make sure the API doesn't make it harder for engines to attach annotations.

**Q3:** For type mismatches, I want `(T, ok=true, err=DecodeError)`. But here's the Go reality: if we store `json.RawMessage` internally (like Noel wants), then `Get` becomes "decode JSON into T". That means:
- Decode errors are **runtime** errors, not compile-time.
- You need to handle `err` at every call site.

That's fine for correctness, but it's more ceremony than the current pattern. We should be honest about that trade-off.

Also, about Noel's "store as `json.RawMessage`" idea: that's a **big** change. It means every `Set` does a JSON marshal, and every `Get` does a JSON unmarshal. It's not free (though you said no performance framing, so I'll leave it there).

---

### Jordan "Just Let Me Ship"

Okay, I'm the person who actually writes middleware and app logic. Let me tell you what I care about:

**Q1:** I don't want to think about "invariants" when I'm attaching a hint to a turn. I just want to write:
```go
t.SetToolConfig(engine.ToolConfig{Enabled: true})
```

Not:
```go
if err := t.Data.Set(turns.KeyToolConfig, engine.ToolConfig{Enabled: true}); err != nil {
    return fmt.Errorf("failed to set tool config: %w", err)
}
```

If we're going to add typed accessors, they need to be **ergonomic**. The current pattern (`t.Data[key] = value`) is simple. If the new pattern is more verbose, it better be **way** safer.

**Q2:** Should `Metadata` have the same invariants? I don't care, as long as the API is consistent. If I have to learn two different patterns for `Data` vs `Metadata`, that's annoying.

**Q3:** For type mismatches, I want **clear error messages**. If I stored a `string` and tried to read it as `int`, the error should say:
```
Turn.Data[tool_config]: expected engine.ToolConfig, got string "oops"
```

Not:
```
json: cannot unmarshal string into Go value of type engine.ToolConfig
```

The first error tells me **what key** and **what I did wrong**. The second error is just a JSON library error.

Also, I'm worried about Noel's "store as JSON" idea. Does that mean I can't store types that don't round-trip through JSON perfectly? Like, what about `time.Time` with nanosecond precision?

---

### Casey "Code Review"

I review a lot of code, and I triage a lot of bugs. Here's what I want:

**Q1:** The invariant I care about most is: **"what type is stored under this key?" should be discoverable**.

Right now, if I see:
```go
t.Data[turns.DataKeyToolConfig] = someValue
```

I have to grep for `DataKeyToolConfig` usage to figure out what type `someValue` should be. That's not great.

If we move to typed keys like:
```go
var KeyToolConfig = Key[engine.ToolConfig]{...}
```

Then I can **jump to definition** and immediately see the expected type. That's a big win for reviewability.

**Q2:** Should `Metadata` have the same invariants? **Yes**, for consistency. But I also want a **crisp separation rule** in the docs:
- `Data` = application semantics (config, hints, agent mode).
- `Metadata` = provider/runtime annotations (usage, model, trace ID).

That way, when I'm reviewing a PR that adds a new key, I can ask: "Is this really `Data`, or should it be `Metadata`?"

**Q3:** For type mismatches, I want `(T, ok=true, err=DecodeError)` **plus** good error messages. But I also want:
- **Fail-fast at `Set` time** if the value is obviously wrong (e.g., non-serializable).
- **Log warnings** if we detect a mismatch during `Get` (so we can catch bugs in tests/staging).

One more thing: if we make `Data` opaque, we need a **clear escape hatch** for persistence code that needs to iterate all entries. Something like:
```go
t.Data.Range(func(key TurnDataKeyID, value json.RawMessage) bool { ... })
```

---

## Rebuttals

### Asha responds to Priya

Priya, you're right that Go can't infer `T` from assignment. That's why I said we need **typed keys** `Key[T]`. I'm glad we agree on that.

But you're worried about the "ceremony" of handling `err` at every call site. Here's my counter: **the ceremony is already there**, it's just hidden. Look at `engine_openai.go:127-130` again—that's two `if` statements and a silent failure. The new pattern makes the error **explicit** instead of **silent**.

I'd rather have:
```go
cfg, ok, err := t.Data.Get(turns.KeyToolConfig)
if err != nil {
    log.Warn().Err(err).Msg("failed to decode tool config")
}
```

Than:
```go
if cfgAny, ok := t.Data[turns.DataKeyToolConfig]; ok && cfgAny != nil {
    if cfg, ok := cfgAny.(engine.ToolConfig); ok {
        // use cfg
    }
    // silent failure if type assertion fails
}
```

The first pattern is **debuggable**. The second pattern is **silent**.

---

### Noel responds to Jordan

Jordan, you asked: "Does storing as JSON mean I can't store types that don't round-trip perfectly?"

Good question. The answer is: **if it doesn't round-trip through JSON, it probably shouldn't be in `Turn.Data`**.

`time.Time` actually round-trips fine through JSON (it uses RFC3339 encoding). But if you have a type with custom semantics that JSON can't represent, then yes, you'd need to either:
1. Implement `MarshalJSON`/`UnmarshalJSON` on that type.
2. Store a different representation (e.g., a struct that **is** JSON-friendly).

This is a **feature**, not a bug. It forces us to think about what we're persisting.

And to your point about error messages: I agree, we should wrap JSON errors with context. Something like:
```go
return fmt.Errorf("Turn.Data[%s]: failed to decode as %T: %w", key, zero, err)
```

That gives you the key, the expected type, and the underlying error.

---

### Priya responds to Asha

Asha, I'm not saying the ceremony is bad—I'm saying we should be **honest** about it. The current pattern is:
```go
if cfgAny, ok := t.Data[key]; ok && cfgAny != nil {
    if cfg, ok := cfgAny.(T); ok {
        // use cfg
    }
}
```

The new pattern (with opaque wrapper + typed keys) is:
```go
cfg, ok, err := t.Data.Get(turns.KeyToolConfig)
if err != nil {
    // handle decode error
}
if ok {
    // use cfg
}
```

That's **better** (explicit error), but it's not **less** code. We're trading silent failures for explicit error handling. That's a good trade, but let's not pretend it's "simpler".

Also, about storing `json.RawMessage` internally: that's a **structural** guarantee of serializability, which is great. But it also means:
- Every `Set` does a JSON marshal (allocates).
- Every `Get` does a JSON unmarshal (allocates).

If we're okay with that (and you said no performance framing), then fine. But it's a real trade-off.

---

### Jordan responds to Casey

Casey, I like your "crisp separation rule" idea:
- `Data` = application semantics.
- `Metadata` = provider/runtime annotations.

But here's my worry: in practice, the line is blurry. For example, is `trace_id` `Data` or `Metadata`? It's set by the application (for distributed tracing), but it's also used by providers (for logging).

I think we need **examples** in the docs, not just a rule. Show me 5-10 real keys and where they belong.

Also, about the escape hatch for iteration: yes, we need that. But I want it to be **typed**, not just `json.RawMessage`. Something like:
```go
t.Data.Range(func(key TurnDataKeyID, value any) bool { ... })
```

So persistence code can still inspect values without decoding them first.

---

### Casey responds to Jordan

Jordan, you're right that the line between `Data` and `Metadata` can be blurry. Here's how I'd draw it:

- **`Data`**: "If I serialize this turn to YAML and load it in a different process, does this value affect **application behavior**?" If yes, it's `Data`.
- **`Metadata`**: "Is this value only useful for **observability** (logs, traces, debugging)?" If yes, it's `Metadata`.

Example:
- `tool_config` → `Data` (affects which tools are called).
- `usage` (token counts) → `Metadata` (observability only).
- `trace_id` → **`Metadata`** (it's for correlation, not application logic).

About the escape hatch: I see your point, but if we store `json.RawMessage` internally (like Noel wants), then `Range` **has** to expose raw bytes. Otherwise, we'd have to decode every value just to iterate, which defeats the purpose.

Maybe we provide **two** escape hatches:
1. `Range(func(key, json.RawMessage) bool)` — for persistence (fast, no decode).
2. `RangeTyped(func(key, any) bool)` — for debugging (decodes each value).

---

## Moderator Summary

### Key Tensions

1. **Ergonomics vs Explicitness**
   - Current pattern: simple syntax (`t.Data[key] = value`), but silent failures.
   - Proposed pattern: explicit errors (`Get/Set` with `err`), but more ceremony.
   - **Consensus**: The trade-off is worth it, but we need to minimize verbosity (typed keys help).

2. **Serializability: validate vs enforce**
   - Noel wants **structural** enforcement (store `json.RawMessage`).
   - Others want **validation** at `Set` time (marshal to check, but store `any`).
   - **Tension**: Structural enforcement is airtight but adds runtime cost. Validation is cheaper but less foolproof.

3. **`Data` vs `Metadata` invariants**
   - **Consensus**: They should have the **same** API shape (typed access, serializability).
   - **Open question**: Should they have the **same** internal representation, or can `Metadata` be more permissive?

4. **Error UX for type mismatches**
   - **Consensus**: `(T, ok=true, err=DecodeError)` is the right signature.
   - **Requirement**: Error messages must include **key name** and **expected type**.
   - **Open question**: Should we log warnings automatically, or leave that to call sites?

### Interesting Ideas

- **Typed keys `Key[T]`** for inference (Asha, Priya): Strong support. This makes `Get` ergonomic without explicit type args.
- **Crisp separation rule** for `Data` vs `Metadata` (Casey): "Does it affect application behavior?" vs "Is it observability-only?"
- **Two escape hatches** for iteration (Casey): `Range(json.RawMessage)` for persistence, `RangeTyped(any)` for debugging.
- **Fail-fast at `Set`** (Noel, Casey): Validate serializability at write time, not read time.

### Open Questions

1. **Should we store `json.RawMessage` internally, or store `any` and validate on `Set`?**
   - Noel: structural enforcement (store JSON).
   - Priya: validation is cheaper (store `any`, marshal to check).

2. **How do we handle types that don't round-trip through JSON perfectly?**
   - Noel: "If it doesn't round-trip, it shouldn't be in `Data`."
   - Jordan: "What about edge cases like `time.Time` nanoseconds?"

3. **Should `Metadata` be more permissive than `Data`?**
   - Casey: "Maybe `Metadata` allows non-serializable values for runtime-only annotations?"
   - Noel: "No, both should be serializable—use `context.Context` for runtime-only."

4. **What's the migration path?**
   - If we change `Turn.Data` from `map[TurnDataKey]any` to an opaque wrapper, how do we migrate existing code?
   - Do we need a compatibility shim, or can we break the API cleanly?

### Next Steps

- **Debate Round 2**: Key identity and versioning (Q4-Q5).
- **Prototype**: Build a small proof-of-concept with typed keys `Key[T]` and opaque wrapper to test ergonomics.
- **RFC**: Draft a proposal based on these findings.

---

## Related

- Candidates: `reference/01-debate-candidates-typed-turn-data-metadata.md`
- Questions: `reference/02-debate-questions-typed-turn-data-metadata.md`
- Ticket analysis: `analysis/01-opaque-turn-data-typed-get-t-accessors.md`
