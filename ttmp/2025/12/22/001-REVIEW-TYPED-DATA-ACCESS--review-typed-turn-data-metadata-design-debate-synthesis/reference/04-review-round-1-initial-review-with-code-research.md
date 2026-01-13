---
Title: 'Review round 1: initial review with code research'
Ticket: 001-REVIEW-TYPED-DATA-ACCESS
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
    - review
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md
      Note: The synthesis document being reviewed
    - Path: geppetto/pkg/turns/types.go
      Note: Current Turn.Data structure
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Current linter implementation
    - Path: moments/backend/pkg/inference/middleware/current_user_middleware.go
      Note: Example middleware accessing Turn.Data
    - Path: moments/backend/pkg/inference/middleware/compression/turn_data_compressor.go
      Note: Example middleware transforming Turn.Data
ExternalSources: []
Summary: "Animated debate round 1: reviewers research codebase, present opening arguments, and engage in structured discussion about typed Turn.Data/Metadata design decisions."
LastUpdated: 2025-12-22T13:58:11.04454605-05:00
WhatFor: "First structured review round: gather evidence, surface tensions, converge on explicit decisions for big-bang implementation."
WhenToUse: "Use as template for structured review rounds; reference when implementing decisions."
---

# Review Round 1: Initial Review with Code Research

## Pre-Review Research

Each reviewer conducted independent research before the meeting. Here's what they found:

### Priya (Go API Ergonomics Specialist)

**Research queries:**
- Searched for `t.Data[...]` patterns across middleware codebase
- Analyzed type assertion patterns: `value.(string)`, `value.(engine.ToolConfig)`
- Counted nil-map initialization boilerplate

**Findings:**
- **17 middleware files** directly access `Turn.Data`
- **Pattern 1**: Nil-check + init appears in 80% of middleware:
  ```go
  if t.Data == nil {
      t.Data = map[turns.TurnDataKey]any{}
  }
  ```
- **Pattern 2**: Two-step type assertion everywhere:
  ```go
  modeName, _ := t.Data[DataKeyAgentMode].(string)
  if modeName == "" { /* fallback */ }
  ```
- **Pattern 3**: Silent failures (type assertion returns zero value, no error)
- **Observation**: No middleware uses helper functions yet—all direct map access

**Code example from `current_user_middleware.go:70-78`:**
```go
if id := strings.TrimSpace(user.ID); id != "" {
    t.Data[turnkeys.PersonID] = id
}
if e := strings.TrimSpace(user.Email); e != "" {
    t.Data[turnkeys.UserPrimaryEmail] = e
}
if d := strings.TrimSpace(user.DisplayName); d != "" {
    t.Data[turnkeys.UserDisplayName] = d
}
```
*No type assertions here because we're writing strings, but reading requires assertions.*

---

### Mina (Linter Maintainer)

**Research queries:**
- Analyzed `turnsdatalint` current rules
- Searched for `TurnDataKey("...")` ad-hoc constructions
- Checked for raw string literals in `t.Data[...]` (should be caught by linter)

**Findings:**
- **Current linter**: Only prevents raw string literals (`t.Data["foo"]`)
- **Gap**: Does NOT prevent `TurnDataKey("oops")` ad-hoc construction
- **Gap**: Does NOT enforce canonical keys (can use `var myKey = TurnDataKey("custom")`)
- **Gap**: Does NOT validate namespace/version format
- **Found**: 3 instances of ad-hoc key construction in test files (acceptable, but shows drift risk)

**Code example from `turnsdatalint/analyzer.go:141-146`:**
```go
pass.Reportf(
    idx.Lbrack,
    `%s key must be of type %q (not a raw string literal or untyped string constant)`,
    sel.Sel.Name,
    keyTypeStr,
)
```
*Current rule is simple: "typed key expression, not literal". Need to add "canonical key only" rule.*

---

### Noel (Serialization Boundary)

**Research queries:**
- Searched for non-serializable types stored in `Turn.Data`
- Analyzed YAML marshal/unmarshal paths
- Checked compression middleware's `map[string]any` conversion

**Findings:**
- **Compression middleware** (`turn_data_compressor.go:33`) converts `Turn.Data` to `map[string]any`:
  ```go
  func (tdc *TurnDataCompressor) Compress(ctx context.Context, data map[string]any)
  ```
  *This bypasses typed keys entirely—converts to string keys for processing.*
- **YAML serialization**: `types.go:99` shows `Data map[TurnDataKey]interface{}` with `yaml:"data,omitempty"`
- **No validation**: Nothing prevents storing `chan int`, `func()`, or other non-serializable types
- **Late failures**: Would fail at `yaml.Marshal` time, not at write time

**Code example from `compression/turn_data_compressor.go:64-78`:**
```go
value := data[key]
switch v := value.(type) {
case string:
    trimmed := strings.TrimSpace(v)
    if tdc.summarizer != nil && shouldSummarizeString(trimmed, opts.MaxTurnDataStringLength) {
        summary, err := tdc.summarizer.SummarizeText(ctx, trimmed, ...)
        if err == nil {
            data[key] = summary  // mutates map[string]any
        }
    }
```
*Compression works on `map[string]any`, not `map[TurnDataKey]any`. This is a compatibility concern.*

---

### Middlewares (Code Persona)

**Research queries:**
- Analyzed all middleware access patterns
- Identified transformation patterns (read → modify → write)
- Checked for patterns that would break with opaque wrapper

**Findings:**
- **Nil-init pattern**: Every middleware checks `if t.Data == nil` before access
- **Read patterns**: Type assertions with fallbacks:
  ```go
  modeName, _ := t.Data[DataKeyAgentMode].(string)
  if modeName == "" { modeName = cfg.DefaultMode }
  ```
- **Write patterns**: Direct assignment after nil-check
- **Transformation concern**: Compression middleware converts `map[TurnDataKey]any` → `map[string]any` for processing
- **Range patterns**: None found (middlewares don't iterate over all keys)

**Code example from `thinkingmode/middleware.go:47-64`:**
```go
modeName, found := getModeFromTurn(t)  // helper extracts from t.Data
modeName = strings.TrimSpace(modeName)
if modeName == "" {
    modeName = ModeExploring
}
if !found {
    if t.Data == nil {
        t.Data = make(map[turns.TurnDataKey]any)
    }
    t.Data[turnkeys.ThinkingMode] = modeName
}
```
*Typical pattern: read with type assertion, default if missing, write if not found.*

---

### `turnsdatalint` (Code Persona)

**Research queries:**
- Analyzed current AST inspection logic
- Evaluated feasibility of "canonical keys only" rule
- Checked for whole-program analysis requirements

**Findings:**
- **Current approach**: AST-level inspection, no whole-program analysis
- **Feasible additions**:
  - Ban `TurnDataKey("...")` conversions outside `*/keys.go` or `*/turnkeys/*.go`
  - Enforce namespace format regex: `^[a-z]+\.[a-z_]+(@v\d+)?$`
- **Challenges**:
  - Need to identify "canonical key packages" (config or convention?)
  - Deprecation warnings require comment parsing (doable)
- **Performance**: Current analyzer is fast; new rules won't slow it down

**Code example from `turnsdatalint/analyzer.go:152-184`:**
```go
func isAllowedTypedKeyExpr(pass *analysis.Pass, e ast.Expr, wantPkgPath, wantName string) bool {
    // Disallow raw string literals
    if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
        return false
    }
    // Disallow untyped string const identifiers
    if isUntypedStringConstExpr(pass, e) {
        return false
    }
    // Check if expr is of the right named type
    // ...
}
```
*Current logic is AST-based and fast. Can extend to check "is this a canonical key var/const?".*

---

### `turns` Data Model (Code Persona)

**Research queries:**
- Analyzed current `Turn.Data` structure and YAML serialization
- Checked nil-map initialization points
- Evaluated API surface needs

**Findings:**
- **Current structure**: `Data map[TurnDataKey]interface{}` with `yaml:"data,omitempty"`
- **Nil handling**: Scattered across codebase (serde, helpers, middleware)
- **YAML serialization**: Works automatically (map serializes as YAML map)
- **API needs**: Minimal—just Get/Set/Delete/Range/Len
- **Concern**: If we go opaque, need custom `MarshalYAML`/`UnmarshalYAML` to preserve human-readable YAML

**Code example from `types.go:91-100`:**
```go
type Turn struct {
    ID     string  `yaml:"id,omitempty"`
    RunID  string  `yaml:"run_id,omitempty"`
    Blocks []Block `yaml:"blocks"`
    Metadata map[TurnMetadataKey]interface{} `yaml:"metadata,omitempty"`
    Data map[TurnDataKey]interface{} `yaml:"data,omitempty"`
}
```
*Simple map serialization. If we wrap, we need to preserve this YAML shape.*

---

## Opening Statements

### Priya: "The API Should Feel Natural"

*Priya leans forward, gesturing at her screen.*

"Look, I've seen the patterns. Every middleware does this dance:

```go
if t.Data == nil {
    t.Data = map[turns.TurnDataKey]any{}
}
modeName, _ := t.Data[DataKeyAgentMode].(string)
if modeName == "" {
    modeName = cfg.DefaultMode
    t.Data[DataKeyAgentMode] = modeName
}
```

**That's three lines of boilerplate for one read-with-default.** The synthesis proposes typed keys `Key[T]` with inference—that's good! But I need to see the actual API.

**My position:**
1. **Typed keys `Key[T]` are essential**—they enable inference so callers don't write `Get[engine.ToolConfig](key)`
2. **Two APIs for `Set`**: `Set` panics (common case), `TrySet` returns error (validation)
3. **`Get` returns `(T, bool, error)`**: `ok=false` means not found, `err!=nil` means decode error
4. **Public map + helpers first**: Let's validate ergonomics before committing to opaque wrapper

**Question for the group**: If we go opaque wrapper, how do we handle compression middleware's `map[string]any` conversion? Do we add `AsStringMap()` method, or force compression to work with typed keys?

---

### Mina: "Enforcement Must Be Automatic"

*Mina taps her keyboard, showing the linter code.*

"I've analyzed `turnsdatalint`. Right now it's simple: ban raw strings. But the synthesis wants more: ban ad-hoc keys, enforce namespaces, warn on deprecation.

**My position:**
1. **Linter can enforce canonical keys**: Check that key expressions come from `*/keys.go` or `*/turnkeys/*.go` packages
2. **Namespace format regex**: `^[a-z]+\.[a-z_]+(@v\d+)?$`—enforceable via AST inspection
3. **Deprecation warnings**: Parse `// Deprecated:` comments, warn at usage sites
4. **BUT**: Linter can't prevent bypasses if we keep public map. If we want structural guarantees, we need opaque wrapper.

**Question for the group**: Do we want linter-based enforcement (can be bypassed) or structural enforcement (opaque wrapper)? Can't have both—pick one.

**Also**: Compression middleware converts to `map[string]any`. If we enforce typed keys, does compression break? Or do we allow `AsStringMap()` escape hatch?

---

### Noel: "Serialization Failures Must Fail Early"

*Noel points at a YAML error log.*

"I've seen production failures: `yaml: unsupported type: chan int`. Happens at serialization time, not at write time. That's too late.

**My position:**
1. **Store `json.RawMessage` internally**: Structurally guaranteed serializable
2. **Marshal at `Set`, unmarshal at `Get`**: One marshal, multiple reads (but unmarshal cost)
3. **Alternative**: Store `any`, validate by marshaling at `Set` time (fail-fast, but not structurally guaranteed)
4. **YAML rendering**: Decode JSON → `any`, emit as YAML (human-readable)

**Question for the group**: Is unmarshal-on-every-`Get` acceptable? Or do we need caching? Compression middleware reads values—will repeated unmarshal hurt performance?

**Also**: Compression converts to `map[string]any`. If we store `json.RawMessage`, how does compression work? Do we unmarshal all keys first?

---

### Middlewares: "We Need Direct Access Patterns"

*The middlewares persona speaks with the voice of accumulated patterns.*

"We're the ones touching `Turn.Data` in every request. We've seen the patterns:

1. **Nil-init**: Every middleware checks `if t.Data == nil`
2. **Read-with-default**: `value, _ := t.Data[key].(T); if value == zero { value = default }`
3. **Write-after-check**: `if condition { t.Data[key] = value }`
4. **Transformation**: Compression converts entire map to `map[string]any`

**Our position:**
1. **Opaque wrapper is risky**: Compression middleware needs `map[string]any`—how do we get that?
2. **Typed helpers are good**: But we need `MustGet` for tests (panic if missing)
3. **Nil-init should be automatic**: Wrapper should handle this
4. **Range support**: We don't need it now, but might later

**Question for the group**: If we go opaque wrapper, how do we support compression's `map[string]any` conversion? Do we add `AsStringMap()`? Or force compression to work with typed keys (breaking change)?

**Also**: If `Get` returns `(T, bool, error)`, how do we handle "not found but that's OK" vs "decode error"? The `bool` helps, but error messages need to be clear.

---

### `turnsdatalint`: "I Can Enforce, But Not Prevent"

*The linter speaks in terms of AST nodes and type checks.*

"I can inspect AST and flag violations. But I can't prevent bypasses if the map is public.

**My position:**
1. **Can enforce canonical keys**: Check that key expressions are vars/consts from canonical packages
2. **Can enforce namespace format**: Regex on key string values
3. **Can warn on deprecation**: Parse comments, flag usage
4. **CANNOT prevent**: Direct map access bypassing helpers (if map is public)

**Question for the group**: Do we want "best-effort enforcement" (linter) or "structural enforcement" (opaque wrapper)? If opaque, I become less critical—the API boundary prevents bypasses.

**Also**: Compression middleware's `map[string]any` conversion—if we enforce typed keys, does this break? Or is compression exempt from lint rules?

---

### `turns` Data Model: "I Need a Stable Shape"

*The data model speaks with the weight of persistence.*

"I'm serialized to YAML in production. I need a predictable shape.

**My position:**
1. **Current shape**: `data: { key: value, ... }` in YAML (human-readable)
2. **If opaque wrapper**: Need custom `MarshalYAML`/`UnmarshalYAML` to preserve shape
3. **Nil handling**: Should be centralized (wrapper or serde), not scattered
4. **API surface**: Get/Set/Delete/Range/Len is sufficient

**Question for the group**: If we store `json.RawMessage` internally, how do we render as YAML? Do we unmarshal to `any` then let YAML encoder handle it? That works, but adds decode step.

**Also**: Compression middleware converts to `map[string]any`. If we're opaque, how does compression access the data? Do we expose `AsStringMap()` or force typed access?

---

## Rebuttals and Discussion

### Priya responds to Middlewares

**Priya**: "You're worried about compression's `map[string]any` conversion. I get it—that's a real pattern. But here's the thing: if we go opaque wrapper, we can add `AsStringMap()` method that returns `map[string]any` for compatibility. It's an escape hatch, but it's explicit.

**Better question**: Do we want compression to work on typed keys, or is string-map conversion acceptable? If acceptable, `AsStringMap()` solves it. If not, compression needs refactoring."

**Middlewares**: "`AsStringMap()` works, but it feels like a workaround. Why not keep public map + helpers? Then compression keeps working, and new code uses helpers."

**Priya**: "Because then helpers can be bypassed. Mina's linter can't prevent `t.Data[someKey] = value` if the map is public. We need structural enforcement, or we accept drift."

---

### Noel responds to Middlewares (performance concern)

**Noel**: "You're worried about unmarshal-on-every-`Get`. Valid concern. But compression reads values once per request—unmarshal cost is negligible compared to LLM inference.

**If we store `json.RawMessage`, compression can call `Get` for each key, unmarshal, build `map[string]any`. Or we add `AsStringMap()` that does bulk unmarshal. Either works."

**Middlewares**: "What about hot paths? If a middleware reads `PersonID` 10 times in one request, do we unmarshal 10 times?"

**Noel**: "Yes, unless we cache. But caching adds complexity—when to invalidate? I'd rather accept the cost and let callers cache locally if needed. Most reads are once-per-key anyway."

---

### Mina responds to `turnsdatalint`

**Mina**: "You said you can enforce canonical keys. How? Do you check that the key var/const is defined in `*/keys.go`?"

**`turnsdatalint`**: "Yes. I can check the package path and filename. If key is defined in `pkg/turns/keys.go` or `moments/backend/pkg/turnkeys/*.go`, it's canonical. Otherwise, flag it."

**Mina**: "What about test files? Do we allow ad-hoc keys in tests?"

**`turnsdatalint`**: "We can add a rule: allow `TurnDataKey("test.*")` in `*_test.go` files only. Or require canonical test keys. I prefer the latter—keeps tests honest."

---

### `turns` Data Model responds to Noel (YAML rendering)

**`turns` Data Model**: "You said we'd unmarshal JSON → `any`, then let YAML encoder handle it. That works, but adds a decode step. Is that acceptable?"

**Noel**: "Yes. YAML encoding is already expensive (converts Go values to YAML nodes). One extra decode step is negligible. The benefit: structurally guaranteed serializable storage."

**`turns` Data Model**: "What about human readability? If we store JSON bytes, can we still render nice YAML?"

**Noel**: "Yes. Unmarshal JSON → `any`, then YAML encoder renders it. Result is same as current: `data: { key: value }`. Human-readable."

---

### Middlewares responds to Priya (bypass concern)

**Middlewares**: "You're worried about bypasses if map is public. But Mina's linter can flag `t.Data[key] = value` and require helpers. That's enforcement, just not structural."

**Priya**: "Linter enforcement can be ignored (warnings vs errors). Structural enforcement can't be bypassed. But I hear you—if linter is strict enough, maybe public map + helpers is fine."

**Mina**: "I can make linter errors (not warnings) for raw map access. But that requires config change. And it's still bypassable if someone ignores linter. Structural enforcement is stronger."

---

## Key Tensions Surfaced

1. **Opaque wrapper vs public map**: Structural enforcement vs migration simplicity
2. **`json.RawMessage` vs `any`**: Structural guarantee vs performance (unmarshal cost)
3. **Compression compatibility**: `map[string]any` conversion needs escape hatch or refactor
4. **Linter vs structural**: Can linter be strict enough, or do we need opaque wrapper?
5. **Error handling**: Two APIs (`Set`/`TrySet`) vs single API with error return

---

## Decisions Needed

### Critical (Must Decide)

1. **Opaque wrapper or public map?**
   - **Opaque**: Structural enforcement, but compression needs `AsStringMap()` escape hatch
   - **Public map**: Compression works, but helpers can be bypassed (linter enforcement only)

2. **Store `json.RawMessage` or `any`?**
   - **`json.RawMessage`**: Structurally guaranteed, but unmarshal on every `Get`
   - **`any`**: Fast reads, but validate on `Set` (not structurally guaranteed)

3. **Typed keys `Key[T]`**: **CONSENSUS**—everyone agrees this is essential for ergonomics

### Secondary (Can Decide Later)

4. **Error handling**: `Set` panics + `TrySet` returns error, or single API?
5. **Namespace enforcement**: Required from day one, or optional default-v1?
6. **Compression compatibility**: `AsStringMap()` escape hatch, or refactor compression?

---

## Next Steps

1. **Prototype opaque wrapper** with `AsStringMap()` escape hatch
2. **Benchmark unmarshal cost** for `json.RawMessage` storage (measure real middleware patterns)
3. **Test compression middleware** with opaque wrapper + `AsStringMap()`
4. **Decide**: Opaque + `json.RawMessage` vs public map + `any` + validate

**Timeline**: One week for prototype + benchmarks, then reconvene for decision.
