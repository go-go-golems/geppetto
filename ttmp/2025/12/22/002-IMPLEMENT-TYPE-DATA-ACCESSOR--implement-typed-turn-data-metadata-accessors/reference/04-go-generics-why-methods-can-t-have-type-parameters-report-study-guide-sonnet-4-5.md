---
Title: 'Go generics: why methods can''t have type parameters (report + study guide, sonnet-4.5)'
Ticket: 002-IMPLEMENT-TYPE-DATA-ACCESSOR
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
    - Path: geppetto/geppetto/pkg/turns/types.go
      Note: Final implementation using generic functions (DataSet/DataGet/MetadataSet/etc) instead of generic methods
    - Path: geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md
      Note: Design doc updated to match compiler restriction
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/01-diary.md
      Note: Implementation diary; Step 3/4 describe hitting this limitation
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/sources/go-generic-methods.md
      Note: Shorter/high-signal generics note
ExternalSources:
    - https://go.dev/ref/spec#Type_parameter_declarations
    - https://go.dev/ref/spec#Method_sets
Summary: Detailed report explaining why Go methods cannot declare their own type parameters, the exact compiler error triggered, what's actually allowed vs disallowed, and a self-study guide with exercises to understand the design constraints.
LastUpdated: 2025-12-22T17:00:00-05:00
WhatFor: Understanding why we use generic functions (DataSet/DataGet) instead of generic methods (d.Set/d.Get) in the typed Turn.Data wrapper API
WhenToUse: Reference when confused about 'method must have no type parameters' error or why API uses package-level functions instead of methods
---


# Go Generics: Why Methods Can't Have Type Parameters

## Goal

Explain the exact compiler constraint that prevents methods from declaring their own type parameters in Go, clarify the common confusion between "generic methods" (disallowed) and "methods on generic types" (allowed), and provide a self-study guide for understanding the language design rationale.

## Context

During implementation of the typed `Turn.Data`/`Metadata` wrapper API (ticket 002-IMPLEMENT-TYPE-DATA-ACCESSOR), we initially tried:

```go
type Data struct {
    m map[TurnDataKey]any
}

// This was our first attempt (INVALID Go):
func (d *Data) Set[T any](key Key[T], value T) error { ... }
func (d Data) Get[T any](key Key[T]) (T, bool, error) { ... }
```

This failed with the compiler error:

```
syntax error: method must have no type parameters
```

This document explains **why** this fails, what's actually allowed, and how to research this limitation further.

---

## Report: What Happened, What Error Fired, and Why

### What You Saw in the Diary

In Step 3 of the implementation diary, I initially wrote the wrapper API with **generic methods**, i.e., methods that declare their own type parameter:

```go
// INVALID: Methods cannot declare type parameters
func (d *Data) Set[T any](key Key[T], value T) error { ... }
func (d Data) Get[T any](key Key[T]) (T, bool, error) { ... }
```

### The Exact Rule/Error That Fired

This is **not** a custom lint rule or analyzer warning. It's the **Go compiler parser/type-checker** rejecting the syntax at compile time.

We reproduced it directly with the local Go toolchain (go1.24.2):

```bash
cat > /tmp/go-generic-method-check.go <<'EOF'
package main

type S struct{}

// This is a method with its own type parameter: NOT VALID Go
func (s S) F[T any](t T) {}

func main() {}
EOF

go run /tmp/go-generic-method-check.go
```

Compiler output:

```
# command-line-arguments
/tmp/go-generic-method-check.go:5:13: syntax error: method must have no type parameters
```

That exact text ("**method must have no type parameters**") is what surfaces through IDE tooling (Cursor/gopls). It's a **language constraint**, not a style preference.

### Why It Happened (Key Concept)

Go (as of Go 1.18+) supports **type parameters on:**

- **Types** (generic types): `type Box[T any] struct { v T }`
- **Functions** (generic functions): `func Swap[T any](a, b T) (T, T)`

…but **NOT on methods**. The method form `func (recv T) M[U any](...)` is rejected by the compiler.

---

## The Common Confusion: "Generic Methods" vs "Methods on Generic Types"

Go *does* allow methods on a **generic receiver type**. Example:

```go
type Box[T any] struct {
    v T
}

// This is a method on a generic type, and it is VALID Go.
func (b Box[T]) Get() T {
    return b.v
}

func (b *Box[T]) Set(val T) {
    b.v = val
}
```

This works because the **type** `Box[T]` is generic, and the methods operate on that instantiated type. The methods themselves don't introduce new type parameters.

### What We Actually Needed (And Why It Didn't Work)

Our use case requires a **heterogeneous container** (`Turn.Data`) that stores many different value types keyed by `Key[T]`. The container can't be `Data[T]` because it needs to hold *multiple* types simultaneously (a `Key[string]` and a `Key[int]` and a `Key[ToolConfig]` all in the same map).

So `Data` itself is **not generic**:

```go
type Data struct {
    m map[TurnDataKey]any  // stores values of different types
}
```

Once `Data` is non-generic, you might want "**method-level generics**" to provide type-safe accessors:

```go
// This would be perfect ergonomically, but Go does not allow it:
func (d *Data) Set[T any](key Key[T], value T) error
func (d Data) Get[T any](key Key[T]) (T, bool, error)
```

Since that's **illegal syntax**, we implemented the exact same semantics as **generic functions** (package-level functions with type parameters):

```go
// VALID Go: generic functions (not methods)
func DataSet[T any](d *Data, key Key[T], value T) error
func DataGet[T any](d Data, key Key[T]) (T, bool, error)

func MetadataSet[T any](m *Metadata, key Key[T], value T) error
func MetadataGet[T any](m Metadata, key Key[T]) (T, bool, error)

func BlockMetadataSet[T any](bm *BlockMetadata, key Key[T], value T) error
func BlockMetadataGet[T any](bm BlockMetadata, key Key[T]) (T, bool, error)
```

These provide identical type safety and inference but are called as:

```go
err := turns.DataSet(&t.Data, turnkeys.KeyThinkingMode, "exploring")
mode, ok, err := turns.DataGet(t.Data, turnkeys.KeyThinkingMode)
```

instead of the (impossible) method form:

```go
// INVALID Go syntax:
err := t.Data.Set(turnkeys.KeyThinkingMode, "exploring")
mode, ok, err := t.Data.Get(turnkeys.KeyThinkingMode)
```

---

## Why Go Disallows Method Type Parameters

This is a research-y area, but here are the practical reasons you'll see discussed in the Go proposal/issue tracker:

### 1. Method Sets + Interfaces Get Harder

Go's interfaces are based on **method sets**. If methods could introduce type parameters, many questions arise:

- Does `interface{ M[T any](T) }` exist as a valid interface type?
- How do you assign a concrete value to it?
- How does type inference work through interface calls?
- What does it mean for a type to "satisfy" such an interface?

Example of the ambiguity:

```go
// Hypothetical (not valid Go):
type Processor interface {
    Process[T any](T) T
}

// Does this satisfy Processor?
type MyProcessor struct{}
func (m MyProcessor) Process[T any](v T) T { return v }

// How would you call it through the interface?
var p Processor = MyProcessor{}
result := p.Process(42)  // what is T here?
```

These questions multiply when you consider:
- Multiple type parameters
- Type constraints
- Inference rules
- Reflection

### 2. Implementation Complexity

The compiler, `go/types`, `gopls`, `vet`, and the whole ecosystem would need to agree on:
- Instantiation rules (when/how are methods instantiated)
- Inference rules (how T is determined at call sites)
- Representation (how method sets are encoded)

### 3. Ambiguity / Readability Tradeoffs

Go intentionally kept the first generics release (Go 1.18) relatively **minimal**:
- Type parameters on types and functions only
- No higher-kinded types
- No specialized syntax beyond `[T any]` and constraints

This was to:
- Avoid ballooning surface area
- Get real-world feedback before extending
- Keep the mental model tractable

### 4. Workarounds Are Sufficient (In Practice)

The Go team's position has been that **generic functions** + **methods on generic types** cover the vast majority of use cases, and the cases where you'd want method-level generics can use package-level functions instead.

Our API is a perfect example: the semantics are identical, the only difference is call-site syntax (`turns.DataSet(&t.Data, k, v)` vs `t.Data.Set(k, v)`).

---

## Study Guide: How to Learn This Properly

### 1. Core Concepts (What to Learn First)

- **Type parameters**: what `[T any]` means, constraints (`[T Comparable]`), inference rules
- **Generic functions vs generic types**: when each fits, how they're instantiated
- **Method sets & interfaces**: how Go decides interface satisfaction, what a method set is

### 2. Canonical Reading (Official Docs)

Start here (read in order):

1. **Go Tour: Generics**
   - [tour.golang.org/generics/1](https://tour.golang.org/generics/1) (interactive examples)

2. **Go spec: Type parameters**
   - [Type parameter declarations](https://go.dev/ref/spec#Type_parameter_declarations)
   - [Instantiations](https://go.dev/ref/spec#Instantiations)

3. **Go spec: Method sets**
   - [Method sets](https://go.dev/ref/spec#Method_sets)
   - [Interface types](https://go.dev/ref/spec#Interface_types)

4. **Go blog posts on generics**
   - Search "golang blog generics" or browse [go.dev/blog](https://go.dev/blog)
   - Key posts:
     - "An Introduction To Generics"
     - "When To Use Generics"

### 3. Practical Experiments (Do These Locally)

Create a scratch module and try these experiments:

#### Experiment 1: Confirm What's Allowed

```bash
mkdir /tmp/go-generics-test && cd /tmp/go-generics-test
go mod init test
```

Create `main.go`:

```go
package main

import "fmt"

// 1. Generic function: VALID
func Swap[T any](a, b T) (T, T) {
    return b, a
}

// 2. Generic type: VALID
type Box[T any] struct {
    v T
}

// 3. Method on generic type: VALID
func (b Box[T]) Get() T {
    return b.v
}

// 4. Method with its own type param: INVALID (uncomment to see error)
// type Container struct{}
// func (c Container) Store[T any](v T) {}

func main() {
    // Generic function call
    a, b := Swap(1, 2)
    fmt.Println(a, b)

    // Generic type usage
    box := Box[string]{v: "hello"}
    fmt.Println(box.Get())
}
```

Run `go run main.go` and observe:
- Cases 1, 2, 3 compile fine
- Uncommenting case 4 produces: `syntax error: method must have no type parameters`

#### Experiment 2: Explore Interface Method Sets

Try expressing an interface that would require a parameterized method:

```go
// This is NOT valid Go (and you can't work around it):
// type GenericProcessor interface {
//     Process[T any](T) T
// }

// Instead, interfaces in Go look like this:
type StringProcessor interface {
    Process(string) string
}

type IntProcessor interface {
    Process(int) int
}

// No way to express "Process[T any] for any T" in an interface
```

#### Experiment 3: Heterogeneous Container (Our Use Case)

Implement a simplified version of our `Data` wrapper:

```go
package main

import (
    "encoding/json"
    "fmt"
)

type Key[T any] struct {
    name string
}

type Data struct {
    m map[string]any
}

// Generic function approach (this works):
func DataSet[T any](d *Data, key Key[T], value T) error {
    if d.m == nil {
        d.m = make(map[string]any)
    }
    if _, err := json.Marshal(value); err != nil {
        return fmt.Errorf("value not serializable: %w", err)
    }
    d.m[key.name] = value
    return nil
}

func DataGet[T any](d Data, key Key[T]) (T, bool, error) {
    var zero T
    if d.m == nil {
        return zero, false, nil
    }
    value, ok := d.m[key.name]
    if !ok {
        return zero, false, nil
    }
    typed, ok := value.(T)
    if !ok {
        return zero, true, fmt.Errorf("expected %T, got %T", zero, value)
    }
    return typed, true, nil
}

func main() {
    var d Data
    
    // Keys with types
    keyName := Key[string]{name: "name"}
    keyAge := Key[int]{name: "age"}
    
    // Set values (type inferred from Key[T] and value)
    _ = DataSet(&d, keyName, "Alice")
    _ = DataSet(&d, keyAge, 30)
    
    // Get values (type inferred from Key[T])
    name, _, _ := DataGet(d, keyName)
    age, _, _ := DataGet(d, keyAge)
    
    fmt.Printf("%s is %d years old\n", name, age)
}
```

Run this and observe:
- Type safety: you can't pass `int` to `keyName` or `string` to `keyAge`
- Inference works: you don't write `DataSet[string]` or `DataGet[int]`, it's inferred
- The only ergonomic difference from methods is the call site syntax

### 4. How This Applies to Our Design

In our typed `Turn.Data` API:

**What we wanted:**
```go
t.Data.Set(turnkeys.KeyThinkingMode, "exploring")
mode, ok, err := t.Data.Get(turnkeys.KeyThinkingMode)
```

**What we got (equivalent semantics, different syntax):**
```go
turns.DataSet(&t.Data, turnkeys.KeyThinkingMode, "exploring")
mode, ok, err := turns.DataGet(t.Data, turnkeys.KeyThinkingMode)
```

This preserves:
- Type inference (`T` inferred from `Key[T]` and `value`)
- Centralized nil-map initialization
- JSON marshal validation (fail-fast)
- No direct map access (opaque wrapper)

### 5. Research Guide (How to Dig Deeper on "Generic Methods" Specifically)

Search terms that will get you to the right design discussions:

- `"parameterized methods" Go proposal`
- `"generic methods" golang/go issue`
- `"method must have no type parameters" go language design`
- `site:github.com/golang/go "method type parameters"`

Key resources:

1. **Go proposals repo**: [github.com/golang/proposal](https://github.com/golang/proposal)
   - Search for "generics" and read the accepted proposal
   - Look for discussion of why method-level generics were excluded

2. **Go issue tracker**: [github.com/golang/go/issues](https://github.com/golang/go/issues)
   - Search for "generic methods" or "parameterized methods"
   - Read the threads (often long but contain real tradeoffs)

3. **Go Nuts mailing list / Reddit r/golang**
   - Search for "method type parameters" discussions
   - Lots of practical Q&A about why this limitation exists

### 6. Mini-Exercises (To Make It Stick)

Do these in a scratch module:

**Exercise 1: Reproduce the Error**
- Implement a typed-key map wrapper
- First try the generic method version (observe the failure and exact error text)
- Then implement the generic function version (observe it compiles)

**Exercise 2: Explore Inference**
- Build a small "registry of typed keys" package
- Write test cases showing how inference behaves at call sites
- Try intentionally passing wrong types to see how errors present

**Exercise 3: Compare Ergonomics**
- Implement the same API both ways (generic functions vs hypothetical generic methods)
- Measure: lines of code, clarity, ease of use
- Conclusion: syntax differs, but semantics/safety are identical

**Exercise 4: Type Mismatch Error Reporting**
- Add one intentional type mismatch to your wrapper
- Observe the error format (`expected %T got %T` style)
- Compare to how a direct map `map[string]any` would fail (requires manual checking)

---

## Quick Reference: Valid vs Invalid

### ✅ VALID Go Syntax

```go
// Generic function
func Swap[T any](a, b T) (T, T)

// Generic type
type Box[T any] struct { v T }

// Method on generic type
func (b Box[T]) Get() T

// Generic function with constraints
func Min[T constraints.Ordered](a, b T) T

// Multiple type parameters
func Zip[A, B any](as []A, bs []B) []Pair[A, B]
```

### ❌ INVALID Go Syntax

```go
// Method with its own type parameter
type Container struct{}
func (c Container) Store[T any](v T) {}
// Error: method must have no type parameters

// Interface with parameterized method
type Processor interface {
    Process[T any](T) T
}
// Error: method must have no type parameters

// Method declaring new type param not on receiver
type Box[T any] struct{}
func (b Box[T]) Transform[U any](fn func(T) U) Box[U] {}
// Error: method must have no type parameters
```

---

## Usage Examples

### When You Hit This Error

**Scenario**: You're implementing a wrapper and try:

```go
type Wrapper struct {
    data map[string]any
}

func (w *Wrapper) Set[T any](key string, val T) error {
    // ...
}
```

**Compiler says**:
```
syntax error: method must have no type parameters
```

**Fix**: Change to generic function:

```go
func WrapperSet[T any](w *Wrapper, key string, val T) error {
    if w.data == nil {
        w.data = make(map[string]any)
    }
    w.data[key] = val
    return nil
}

// Call as:
err := WrapperSet(&w, "key", 42)
```

### When You See "Methods on Generic Types" and Think It's the Same

**Misunderstanding**: "Go supports generic methods because I saw `func (b Box[T]) Get() T`"

**Clarification**: That's a **method on a generic type** (the receiver `Box[T]` is generic), not a **generic method** (a method with its own new type parameter).

```go
// This is allowed (method on generic type):
type Box[T any] struct { v T }
func (b Box[T]) Get() T { return b.v }

// This is NOT allowed (generic method):
type Box struct {}
func (b Box) Get[T any]() T { var zero T; return zero }
```

---

## Related Documents

- **Implementation diary**: `01-diary.md` (Steps 3/4 describe encountering and fixing this issue)
- **Final design doc**: `../../001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md`
- **Code**: `../../../../../../geppetto/pkg/turns/types.go` (final implementation using generic functions)

---

## Appendix: Why This Matters for Our API

### The Design Constraint

We need:
1. A **single container** (`Turn.Data`) that stores **multiple types** (string, int, ToolConfig, etc.)
2. **Type-safe accessors** that prevent `string` where `int` is expected
3. **Centralized nil-map init** and validation

### Why `Data[T]` Doesn't Work

If we made `Data` generic:

```go
type Data[T any] struct {
    m map[TurnDataKey]T
}
```

Then:
- `Turn` would need to be `Turn[T]` to hold `Data[T]`
- We could only store **one type** per `Turn` (all strings, or all ints, but not both)
- That's useless—we need heterogeneous storage

### Why Generic Methods Would Be Perfect (But Aren't Allowed)

Ideally:

```go
type Data struct {
    m map[TurnDataKey]any  // heterogeneous storage
}

// INVALID: method with type parameter
func (d *Data) Set[T any](key Key[T], value T) error
func (d Data) Get[T any](key Key[T]) (T, bool, error)
```

This would give us:
- Heterogeneous storage (`any` under the hood)
- Type-safe accessors (generic methods with `Key[T]` typing)
- Ergonomic call sites (`t.Data.Set(...)`)

But Go **doesn't allow it**.

### The Actual Solution (Generic Functions)

Since generic methods are disallowed, we use **generic functions** with the same type signature:

```go
func DataSet[T any](d *Data, key Key[T], value T) error
func DataGet[T any](d Data, key Key[T]) (T, bool, error)
```

Call sites:

```go
// Instead of: t.Data.Set(key, value)
err := turns.DataSet(&t.Data, key, value)

// Instead of: val, ok, err := t.Data.Get(key)
val, ok, err := turns.DataGet(t.Data, key)
```

**Tradeoffs**:
- ✅ Identical type safety and inference
- ✅ Identical runtime behavior
- ✅ Centralized validation and nil-map init
- ❌ Slightly less ergonomic syntax (package prefix + explicit receiver parameter)

**Conclusion**: The semantics are identical; only the call-site syntax differs. This is the idiomatic Go way when you need method-level generics.

---

**Document generated by Claude Sonnet 4.5 (2025-12-22)**
