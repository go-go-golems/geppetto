# Go generics and the “method must have no type parameters” wall

You already captured the key empirical fact: **the compiler rejects methods that declare their own type parameter list** with:

```
syntax error: method must have no type parameters
```

That error isn’t a lint preference; it’s a **language rule** rooted in how Go defines method declarations, method sets, and interface satisfaction. The “Step 4” workaround—moving the polymorphism to **generic functions**—is the idiomatic escape hatch, and it’s explicitly recommended by the original generics design. ([go.googlesource.com][1])

Below is a deeper, research-backed explanation with lots of examples and pointers.

---

## 1) The rule, precisely: what Go allows vs forbids

### Allowed: type parameters on **functions**

```go
func Identity[T any](v T) T { return v }
```

This is exactly what the spec describes: a **type parameter list** belongs to a *generic function* (or type). ([Go.dev][2])

### Allowed: type parameters on **types**

```go
type Box[T any] struct{ v T }

func (b Box[T]) Get() T { return b.v }
func (b *Box[T]) Set(v T) { b.v = v }
```

This works because **the receiver type is generic** and the method uses the receiver’s already-declared `T`. The spec explicitly notes that type parameters “may also be declared by the receiver specification of a method declaration associated with a generic type.” ([Go.dev][2])

### Forbidden: methods that introduce **their own** type parameters

```go
type S struct{}

func (s S) F[T any](t T) {} // ❌ rejected
```

The design doc is unambiguous:

* “methods may not themselves have additional type parameters”
* if you want that, “write a suitably parameterized top-level function” ([go.googlesource.com][1])

And it reiterates the same in the “No parameterized methods” section. ([go.googlesource.com][1])

### Also forbidden: interface methods with type parameters

Even if you wanted to express this at the interface level, the toolchain has a dedicated error path. The compiler’s own syntax testdata includes cases like:

```go
type I interface {
    m[P any]()
}
```

…which is annotated to produce: **“interface method must have no type parameters”**. ([Go.dev][3])

---

## 2) Why the compiler error exists (the “real” reason)

The short version is: **methods are deeply tied to interfaces in Go**, and generic methods create a hard question:

> If a type has a parameterized method, can it implement an interface that requires that method?

Go’s generics design explicitly calls this out as the blocker. Methods are one of the main mechanisms for interface satisfaction, but with method-level type parameters it’s unclear how to make interface calls reliably instantiate the right method bodies. ([go.googlesource.com][1])

### The design-doc thought experiment (paraphrased)

The generics proposal presents a multi-package scenario where:

* Package A defines a type with a parameterized method (e.g., `Identity[T]`).
* Package B defines an interface that mentions that parameterized method.
* Package C takes `any`, does a type assertion to that interface, then calls `Identity[int]`.

At the call site in package C, the concrete type is only known through an interface value. The question becomes: **where does the compiler instantiate the `int` version of the method?**

The design doc explores options (link-time instantiation, run-time instantiation), and argues they become untenable—especially when reflection enters the picture. That’s the core rationale for the restriction. ([go.googlesource.com][1])

---

## 3) “Generic methods” vs “methods on generic types” (the classic confusion)

This distinction is worth burning in:

### ✅ Methods on generic types

Receiver is `Box[T]`, method uses `T`:

```go
type Box[T any] struct{ v T }
func (b Box[T]) Get() T { return b.v }
```

### ❌ Generic methods

Receiver is non-generic, method tries to introduce `T`:

```go
type Data struct{}
func (d Data) Get[T any](k Key[T]) (T, bool) // not allowed
```

The spec’s method-declaration grammar makes it visible why there’s no place to put a method-local type parameter list: a method is `func Receiver Name Signature`, and the type parameter list concept is defined for generic functions/types, not methods. ([Go.dev][2])

---

## 4) Your heterogeneous container problem: why you *wanted* method-level generics

Your `Turn.Data` is a textbook case where method-level generics would be perfect ergonomically:

* `Data` must store **heterogeneous** values → internal `map[key]any`
* you want **typed** accessors keyed by `Key[T]`

Conceptually ideal API (but illegal in Go):

```go
func (d *Data) Set[T any](key Key[T], value T) error
func (d Data)  Get[T any](key Key[T]) (T, bool, error)
```

So you moved the type parameter to a **top-level generic function**:

```go
func DataSet[T any](d *Data, key Key[T], value T) error
func DataGet[T any](d Data, key Key[T]) (T, bool, error)
```

This is *exactly* the workaround the generics design doc points people toward: when you want extra type params “on a method”, write a parameterized function instead. ([go.googlesource.com][1])

---

## 5) A solid “typed-key map” implementation (minimal but complete)

Here’s a compact reference implementation that shows the core mechanics: nil-map init, storage as `any`, retrieval with a checked assertion, and error messages that tell you what happened.

```go
package typedmap

import (
	"fmt"
)

type Key[T any] struct {
	id string
}

func NewKey[T any](id string) Key[T] { return Key[T]{id: id} }

type Data struct {
	m map[string]any
}

func DataSet[T any](d *Data, key Key[T], value T) {
	if d.m == nil {
		d.m = make(map[string]any)
	}
	d.m[key.id] = value
}

func DataGet[T any](d Data, key Key[T]) (T, bool, error) {
	var zero T
	if d.m == nil {
		return zero, false, nil
	}
	v, ok := d.m[key.id]
	if !ok {
		return zero, false, nil
	}
	tv, ok := v.(T)
	if !ok {
		return zero, true, fmt.Errorf("typedmap: key %q stored %T, not requested type", key.id, v)
	}
	return tv, true, nil
}
```

Usage:

```go
var (
	KeyAgentMode = typedmap.NewKey[string]("agent_mode")
	KeyTurnID    = typedmap.NewKey[int64]("turn_id")
)

func example() error {
	var d typedmap.Data

	typedmap.DataSet(&d, KeyAgentMode, "debug")
	typedmap.DataSet(&d, KeyTurnID, int64(42))

	mode, ok, err := typedmap.DataGet(d, KeyAgentMode) // T inferred = string
	_, _, _ = mode, ok, err

	// Intentional mismatch:
	_, _, err = typedmap.DataGet[int64](d, KeyAgentMode)
	return err
}
```

Notice: you **rarely** need explicit type arguments at the call site because `T` is inferable from `Key[T]` (and the value for `Set`). That’s one big reason the generic-function workaround stays ergonomic.

---

## 6) An even more ergonomic pattern (still 100% legal Go): put methods on `Key[T]`

If you want something that *feels* method-y without illegal method type params, you can attach methods to the **generic key type**, not to `Data`:

```go
func (k Key[T]) Get(d Data) (T, bool, error) {
	return DataGet(d, k)
}

func (k Key[T]) Set(d *Data, v T) {
	DataSet(d, k, v)
}
```

Now call sites look like:

```go
KeyAgentMode.Set(&d, "debug")
mode, ok, err := KeyAgentMode.Get(d)
```

This works because `Key[T]` is a generic type, and its methods can use `T` without introducing a new type parameter list. ([Go.dev][2])

(For your “typed Turn.Data accessor” use case, this pattern can be a nice compromise between discoverability and Go’s generics limits.)

---

## 7) Related generics boundaries that look similar (and bite people)

### “Interface method must have no type parameters”

Covered above; it’s a direct toolchain restriction. ([Go.dev][3])

### “Function type must have no type parameters”

Go lets you declare **generic functions**, but a *function type literal* can’t itself be parameterized (you can’t have `type F func[T any](T)`).
The compiler’s own syntax tests include this restriction too. ([Go.dev][3])

### Generic interfaces are real (but different)

Go *does* support interfaces with type parameters (e.g. `type Set[E any] interface { Add(E) }`), which is powerful for constraints and API design, but that’s distinct from “generic interface methods.” ([Go.dev][4])

---

## 8) Status of “generic methods” as a language feature

This is not “secretly supported in newer Go.” It remains an active design topic, and proposals continue to show up.

* The long-running proposal to allow type parameters in methods is tracked as a language-change issue. ([GitHub][5])
* A newer 2025 proposal was closed as a duplicate of that same issue, indicating the topic is still alive but unresolved. ([GitHub][6])
* Meanwhile, Go continues to evolve generics in other directions (e.g., Go 1.24 added **generic type aliases**), but **not** method-level type params. ([Go.dev][7])

---

## 9) How to study this systematically (with canonical sources)

If you want the mental model that makes this “obvious in hindsight,” focus on three pillars:

1. **Type parameter declarations / constraints / instantiation**
   Spec section: type parameter declarations + how constraints work. ([Go.dev][2])

2. **Method declarations and method sets**

* Method declarations grammar. ([Go.dev][2])
* Method sets: how Go decides what methods can be called and how interface membership is computed. ([Go.dev][2])

3. **The generics design rationale**
   Read the original accepted design doc, specifically:

* “Methods may not take additional type arguments” ([go.googlesource.com][1])
* “No parameterized methods” (the interface/instantiation argument) ([go.googlesource.com][1])
  And then follow the linked proposal issue thread for evolving discussion. ([GitHub][5])

For an “official gentle ramp” (useful even if you’re advanced, because it anchors terminology), Go’s tutorial is still the canonical baseline. ([Go.dev][8])

---

## 10) Mini-exercises that map directly to your `Turn.Data` design

1. **Reproduce the exact compiler failure**

   * Write the illegal method form; observe the parser error.

2. **Rewrite it two ways**

   * top-level `DataGet/DataSet`
   * `Key[T].Get/Set` (methods on generic type)

3. **Stress-test inference**

   * calls where `T` is inferred from key only (`Get`)
   * calls where `T` is inferred from key + value (`Set`)
   * one intentional mismatch; ensure your error includes the stored dynamic type

4. **Think in “method sets”**

   * explain to yourself why Go cares so much about methods implementing interfaces (it’s *the* composition mechanism), using the spec definition of method sets and interface type sets. ([Go.dev][2])

---

## References (high signal)

* Go spec: Type parameter declarations, method declarations, method sets, interface types. ([Go.dev][2])
* Type Parameters Proposal (design doc): “Methods may not take additional type arguments” + “No parameterized methods”. ([go.googlesource.com][1])
* Proposal issue tracking “type parameters in methods” (#49085) and a 2025 duplicate (#75526). ([GitHub][5])
* Compiler syntax testdata showing dedicated errors for type-parameterized function types and interface methods. ([Go.dev][3])
* Go 1.24 release notes (example of generics evolution that *did* land: generic type aliases). ([Go.dev][7])
* Go blog: intro to generics / type sets; and generic interfaces. ([Go.dev][9])

If you want, I can adapt the examples above into a drop-in `turns`-style package sketch that mirrors your `Key[T]` + `TurnDataKey` naming and includes the “JSON marshal validation” hook you mentioned—without changing the core API shape you settled on.

[1]: https://go.googlesource.com/proposal/%2B/refs/heads/master/design/43651-type-parameters.md "Type Parameters Proposal"
[2]: https://go.dev/ref/spec "The Go Programming Language Specification - The Go Programming Language"
[3]: https://go.dev/src/cmd/compile/internal/syntax/testdata/issue48382.go " - The Go Programming Language"
[4]: https://go.dev/blog/generic-interfaces "Generic interfaces - The Go Programming Language"
[5]: https://github.com/golang/go/issues/49085 "proposal: spec: allow type parameters in methods · Issue #49085 · golang/go · GitHub"
[6]: https://github.com/golang/go/issues/75526 "proposal: allow type parameters on methods (generic methods) · Issue #75526 · golang/go · GitHub"
[7]: https://go.dev/doc/go1.24 "Go 1.24 Release Notes - The Go Programming Language"
[8]: https://go.dev/doc/tutorial/generics "Tutorial: Getting started with generics - The Go Programming Language"
[9]: https://go.dev/blog/intro-generics "An Introduction To Generics - The Go Programming Language"
