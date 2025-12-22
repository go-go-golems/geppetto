---
Title: 'composer-1: Go generics - why methods can''t have type parameters (report and study guide)'
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
    - Path: geppetto/pkg/turns/types.go
      Note: Implementation that triggered the generic methods limitation
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/reference/01-diary.md
      Note: |-
        Step 3-4 diary entries documenting the error and workaround
        Implementation context and commits
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/sources/go-generic-methods.md
      Note: Condensed/high-signal version
ExternalSources: []
Summary: Detailed explanation of why Go disallows type parameters on methods, what error you'll see, why it exists, and a study guide for learning Go generics properly.
LastUpdated: 2025-12-22T16:00:00-05:00
WhatFor: Reference when encountering 'method must have no type parameters' errors, understanding Go generics limitations, or learning generics systematically.
WhenToUse: Use when debugging generic method syntax errors, designing APIs that need type-safe accessors, or studying Go generics.
---


# composer-1: Go generics - why methods can't have type parameters (report and study guide)

## Goal

This document explains:
1. **What happened** in Step 3 of the typed Turn.Data/Metadata accessor implementation
2. **What error you'll see** and where it comes from (compiler, not a custom lint rule)
3. **Why Go disallows** type parameters on methods (language design rationale)
4. **How to work around it** (generic functions instead of generic methods)
5. **How to learn Go generics properly** (study guide with canonical resources)

## Context

During implementation of the typed wrapper API for `Turn.Data`, `Turn.Metadata`, and `Block.Metadata`, we attempted to use **generic methods**:

```go
// This is what we tried (and what Go REJECTS)
type Data struct {
    m map[TurnDataKey]any
}

func (d *Data) Set[T any](key Key[T], value T) error { ... }
func (d Data) Get[T any](key Key[T]) (T, bool, error) { ... }
```

The Go compiler rejected this with:
```
syntax error: method must have no type parameters
```

This document explains why that restriction exists and how to work around it.

## Quick Reference

### What's Allowed vs Disallowed

| Pattern | Allowed? | Example |
|---------|----------|---------|
| Generic function | ✅ Yes | `func F[T any](x T) T { return x }` |
| Generic type | ✅ Yes | `type Box[T any] struct { v T }` |
| Method on generic type | ✅ Yes | `func (b Box[T]) Get() T { return b.v }` |
| **Method with type params** | ❌ **No** | `func (s S) M[T any](t T) {}` |

### The Exact Error Message

When you try to declare a method with type parameters, the Go compiler emits:

```
syntax error: method must have no type parameters
```

This is a **language constraint**, not a style preference or custom lint rule. It comes from the Go parser/typechecker.

### Workaround: Generic Functions

Instead of generic methods, use generic functions with explicit receiver arguments:

```go
// Instead of: func (d *Data) Set[T any](key Key[T], value T) error
func DataSet[T any](d *Data, key Key[T], value T) error { ... }

// Instead of: func (d Data) Get[T any](key Key[T]) (T, bool, error)
func DataGet[T any](d Data, key Key[T]) (T, bool, error) { ... }
```

Call sites still get type inference:
```go
v, ok, err := turns.DataGet(t.Data, turns.KeyAgentMode)  // T inferred from Key[T]
err := turns.DataSet(&t.Data, engine.KeyToolConfig, cfg) // T inferred from both args
```

## Usage Examples

### Reproducing the Error

To see the error yourself:

```bash
cat > /tmp/test.go <<'EOF'
package main

type S struct{}

func (s S) F[T any](t T) {}

func main() {}
EOF

go run /tmp/test.go
```

Output:
```
/tmp/test.go:5:13: syntax error: method must have no type parameters
```

### Valid Alternatives

**Option 1: Generic function (what we chose)**
```go
package turns

type Data struct { m map[TurnDataKey]any }

func DataSet[T any](d *Data, key Key[T], value T) error {
    if d.m == nil {
        d.m = make(map[TurnDataKey]any)
    }
    // ... validation and storage
    return nil
}

func DataGet[T any](d Data, key Key[T]) (T, bool, error) {
    // ... retrieval with type assertion
}
```

**Option 2: Method on generic type (if receiver can be generic)**
```go
package example

type Box[T any] struct { v T }

// This IS allowed because Box[T] is a generic type
func (b Box[T]) Get() T { return b.v }

func (b *Box[T]) Set(v T) { b.v = v }
```

Note: Option 2 doesn't work for our use case because `Data` must store heterogeneous values (`any`), not a single generic type.

## Detailed Explanation

### Why Go Disallows Generic Methods

The restriction exists for several reasons:

1. **Method sets and interfaces**: Go's interfaces are based on method sets. If methods could introduce type parameters, questions multiply:
   - Can you define `interface{ M[T any](T) }`?
   - How do you assign a value to such an interface?
   - How does type inference work through interface method calls?

2. **Implementation complexity**: The compiler, `go/types`, `gopls`, `vet`, and the entire toolchain would need to agree on:
   - Instantiation rules (when/how to instantiate `M[T]`)
   - Inference rules (can `T` be inferred from arguments?)
   - Representation (how to store method sets with parameterized methods)

3. **Design philosophy**: Go's generics were intentionally kept minimal in the first release (type params on types and functions only) to avoid ballooning surface area and complexity.

### The Confusion: "Generic Methods" vs "Methods on Generic Types"

Many people confuse two concepts:

- **Generic methods** (disallowed): Methods that declare their own type parameters
  ```go
  type S struct{}
  func (s S) M[T any](t T) {}  // ❌ NOT ALLOWED
  ```

- **Methods on generic types** (allowed): Methods on a receiver that is itself a generic type
  ```go
  type Box[T any] struct { v T }
  func (b Box[T]) Get() T { return b.v }  // ✅ ALLOWED
  ```

Our use case needs the first (heterogeneous container with typed accessors), but Go only allows the second (when the receiver type itself is generic).

### Why Our Design Needs Generic Methods (Conceptually)

We want:
- A **non-generic** container (`Data` stores `map[TurnDataKey]any`)
- **Typed accessors** that infer `T` from `Key[T]` and `value`

The ideal API would be:
```go
t.Data.Set(key, value)  // T inferred from key and value
v, ok, err := t.Data.Get(key)  // T inferred from key
```

But since generic methods aren't allowed, we use generic functions:
```go
turns.DataSet(&t.Data, key, value)
v, ok, err := turns.DataGet(t.Data, key)
```

This preserves type safety and inference, just with slightly different call-site syntax.

## Study Guide: Learning Go Generics Properly

### 1. Core Concepts to Master First

Before diving into "why methods can't have type parameters," understand:

- **Type parameters**: What `[T any]` means, constraints (`comparable`, `constraints.Ordered`), and type inference
- **Generic functions vs generic types**: When to use each
- **Method sets and interfaces**: How Go decides interface satisfaction

### 2. Canonical Reading (Official Docs)

**Go Language Specification:**
- [Type parameter declarations](https://go.dev/ref/spec#Type_parameter_declarations)
- [Instantiations](https://go.dev/ref/spec#Instantiations)
- [Method sets](https://go.dev/ref/spec#Method_sets)
- [Interface types](https://go.dev/ref/spec#Interface_types)

**Go Blog / Tutorials:**
- Search: `go blog generics` for official blog posts
- [Go Tour: Generics](https://go.dev/tour/generics/1) (if available)

**Proposal Process:**
- [Go Generics Proposal](https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md) - The original design document

### 3. Practical Experiments (Do These Locally)

**Experiment 1: Confirm what's allowed**
```go
// ✅ Should compile
func F[T any](x T) T { return x }

// ❌ Should fail
type S struct{}
func (s S) M[T any](t T) {}

// ✅ Should compile
type Box[T any] struct { v T }
func (b Box[T]) Get() T { return b.v }
```

**Experiment 2: Explore interface method sets**
Try to express an interface that would require a parameterized method:
```go
// What would this mean? Can you assign a value to it?
type I interface {
    M[T any](T)
}
```
Observe that you can't even express this in Go.

**Experiment 3: Build a typed-key map wrapper**
- First try the generic method version (observe the failure)
- Then implement the generic function version
- Compare call-site ergonomics

### 4. Research Guide: Digging Deeper

**Search terms** that reliably get you to design discussions:
- `"parameterized methods" Go proposal`
- `"generic methods" golang/go issue`
- `"method must have no type parameters" go`

**Where to look:**
- [golang/go issue tracker](https://github.com/golang/go/issues) - Search for "generic methods" or "parameterized methods"
- [Go proposal repository](https://github.com/golang/proposal) - Design documents and discussions
- [Go forums / Reddit](https://www.reddit.com/r/golang/) - Community discussions (grain of salt)

**What to read:**
- Design notes / proposals (often long but contain real tradeoffs)
- Issue tracker threads (especially closed/rejected proposals)
- Community discussions (to understand practical pain points)

### 5. Mini-Exercises (To Make It Stick)

**Exercise 1: Typed registry**
Build a small "registry of typed keys" package:
- Define `Key[T]` wrapper
- Implement `Get[T]` and `Set[T]` as generic functions
- See how type inference behaves at call sites

**Exercise 2: Error reporting**
Add one intentional type mismatch:
```go
// Store as string
turns.DataSet(&d, keyString, "hello")

// Try to retrieve as int
v, ok, err := turns.DataGet[int](d, keyString)
```
Observe the error message format (`expected %T got %T`).

**Exercise 3: Compare with other languages**
If you know Rust, C++, or TypeScript:
- How do they handle "generic methods"?
- What are the tradeoffs?
- Why might Go have chosen differently?

### 6. Common Pitfalls to Avoid

1. **Confusing "methods on generic types" with "generic methods"**
   - Remember: `func (b Box[T]) Get() T` is allowed because `Box[T]` is generic
   - `func (s S) M[T any](...)` is disallowed because `S` is not generic

2. **Assuming "newer Go versions" will add it**
   - As of Go 1.24.2 (our toolchain), this is still disallowed
   - There's no indication this will change soon
   - Design your APIs assuming it won't

3. **Trying to work around it with reflection**
   - Reflection can't give you compile-time type safety
   - Stick to generic functions for type-safe APIs

## Related

- **Diary entries**: See `reference/01-diary.md` Step 3-4 for the implementation context
- **Design doc**: `001-REVIEW-TYPED-DATA-ACCESS/design-doc/03-final-design-typed-turn-data-metadata-accessors.md` for the overall API design
- **Implementation**: `geppetto/pkg/turns/types.go` shows the final generic-function-based API

## Appendix: FAQ

**Q: Will Go ever support generic methods?**  
A: There's no official indication. The restriction is intentional and well-reasoned. Design your APIs assuming it won't change.

**Q: Is there a performance difference between generic methods and generic functions?**  
A: No. Both compile to the same machine code after monomorphization. The difference is purely syntactic/ergonomic.

**Q: Can I use type aliases or embedding to work around this?**  
A: No. The restriction is at the language level. Embedding or aliasing doesn't change method declaration rules.

**Q: What about `go:generate` or code generation?**  
A: Code generation can create type-specific methods, but you lose type inference and compile-time type safety. Generic functions are preferable.

**Q: How do other languages handle this?**  
A: Rust allows generic methods (`impl<T> Trait for Type { fn method<U>(...) }`). C++ allows template methods. TypeScript allows generic methods. Go chose simplicity and consistency over this feature.
