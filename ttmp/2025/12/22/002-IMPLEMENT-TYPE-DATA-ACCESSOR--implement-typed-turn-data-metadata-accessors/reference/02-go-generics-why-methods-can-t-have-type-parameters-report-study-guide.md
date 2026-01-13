---
Title: 'Go generics: why methods can''t have type parameters (report + study guide)'
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
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/reference/01-diary.md
      Note: Where the implementation hit the compiler error
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/sources/go-generic-methods.md
      Note: Deeper references and examples
ExternalSources: []
Summary: Clarifies Go's generics limitation around methods with type parameters, shows the exact compiler error, and provides a self-study guide for understanding generics vs method sets/interfaces.
LastUpdated: 2025-12-22T15:49:19.224799529-05:00
WhatFor: Unblock understanding of the Step 3/4 migration and provide a durable reference for generics-related design decisions in this repo.
WhenToUse: Use when you see `method must have no type parameters`, when designing typed-access APIs, or when reviewing generic constraints and method/interface interactions.
---


# Go generics: why methods can't have type parameters (report + study guide)

## Goal

Explain why the initial “generic method” implementation attempt failed, what produced the error, and provide a study/research path to understand Go’s generics restrictions (especially around methods, method sets, and interfaces).

## Context

In ticket `002-IMPLEMENT-TYPE-DATA-ACCESSOR`, we introduced typed access to `Turn.Data`, `Turn.Metadata`, and `Block.Metadata`.

The most ergonomic API shape would be generic **methods** like:

```go
// This is what we wanted ergonomically, but it is NOT valid Go:
func (d *Data) Set[T any](key Key[T], value T) error
func (d Data) Get[T any](key Key[T]) (T, bool, error)
```

But Go rejects methods that declare their own type parameter list with the exact error:

> `syntax error: method must have no type parameters`

This is a **Go language/compiler restriction**, not a repo-specific lint preference.

## Quick Reference

### 1) The exact failing syntax (method with type parameters)

```go
package main

type S struct{}

// INVALID in Go: method introduces its own type parameter list.
func (s S) F[T any](t T) {}

func main() {}
```

Running:

```bash
go run ./main.go
```

produces:

```
syntax error: method must have no type parameters
```

### 2) What *is* allowed: methods on generic receiver types

Go **does** allow methods on a *generic type* (the receiver itself is parameterized):

```go
type Box[T any] struct{ v T }

// VALID Go: method uses the receiver’s type parameter T.
func (b Box[T]) Get() T { return b.v }
```

What Go does **not** allow is a method that declares *new* type parameters at the method level.

### 3) The workaround used in this repo: generic functions

Because `Turn.Data` is heterogeneous internally (`map[TurnDataKey]any`), we kept wrapper structs non-generic and expressed typed access as **package-level generic functions**:

```go
func DataSet[T any](d *Data, key Key[T], value T) error
func DataGet[T any](d Data, key Key[T]) (T, bool, error)

func MetadataSet[T any](m *Metadata, key Key[T], value T) error
func MetadataGet[T any](m Metadata, key Key[T]) (T, bool, error)

func BlockMetadataSet[T any](bm *BlockMetadata, key Key[T], value T) error
func BlockMetadataGet[T any](bm BlockMetadata, key Key[T]) (T, bool, error)
```

Call sites stay type-inferred and readable:

```go
mode, ok, err := turns.DataGet(t.Data, turns.KeyAgentMode)
err := turns.DataSet(&t.Data, engine.KeyToolConfig, cfg)
```

### 4) What happened in Step 3/4 (short)

- **Step 3 (attempt):** implement `Set[T]`/`Get[T]` as methods on wrapper types
- **Failure:** compiler rejects with `method must have no type parameters`
- **Step 4 (fix):** switch to generic functions `DataSet/DataGet/...` and keep non-generic methods (`Len`, `Range`, `Delete`, YAML marshal/unmarshal)

## Usage Examples

### When you see the error in code review

- If you see a method like `func (x X) M[T any](...)`, it will not compile in Go. Replace it with a generic function form.

### When designing an API

- If your receiver type is naturally generic (e.g., `Box[T]`), methods are fine and can use `T`.
- If your receiver is *not* generic and you want typed behavior (e.g., heterogeneous map wrapper), use generic **functions**.

## Study guide (how to research this yourself)

### Read the spec sections (in order)

- `https://go.dev/ref/spec#Type_parameter_declarations`
- `https://go.dev/ref/spec#Instantiations`
- `https://go.dev/ref/spec#Method_sets`
- `https://go.dev/ref/spec#Interface_types`

### Hands-on exercises

1) Write the failing method example (`S.F[T]`) and confirm the exact compiler error.
2) Write the valid “generic receiver” example (`Box[T].Get()`).
3) Sketch a typed-key wrapper with `DataSet/DataGet` and observe type inference at call sites.

### Research direction

Search terms that reliably land on the right discussions:

- `Go parameterized methods proposal`
- `golang/go generic methods`
- `method must have no type parameters`

When reading, focus on:
- method sets and interface satisfaction implications
- type inference behavior through interfaces
- reasons the initial generics release constrained surface area

## Related

- Ticket diary: `reference/01-diary.md` (Step 3/4 notes)
- Design doc: `ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md`
