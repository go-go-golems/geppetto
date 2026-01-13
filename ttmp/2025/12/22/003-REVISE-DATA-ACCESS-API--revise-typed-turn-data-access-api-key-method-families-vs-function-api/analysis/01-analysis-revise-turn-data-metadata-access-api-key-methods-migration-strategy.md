---
Title: 'Analysis: revise Turn.Data/Metadata access API (key methods) + migration strategy'
Ticket: 003-REVISE-DATA-ACCESS-API
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/turns/types.go
      Note: Current API + wrapper implementation that would be refactored to 3-key-types
    - Path: geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md
      Note: Updated design doc reflecting 3 key types with Get/Set methods
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/reference/01-diary.md
      Note: Current implementation ground truth used for restart-vs-refactor decision
ExternalSources: []
Summary: Decision analysis for revising the typed Turn.Data/Metadata/Block.Metadata access API to 3 key types with methods; recommends retroactive refactor (not restart) and outlines migration strategy.
LastUpdated: 2025-12-22T00:00:00-05:00
WhatFor: Decide whether to restart or refactor the current in-flight migration now that we understand the ergonomic + correctness benefits of key-method families.
WhenToUse: Use before doing further large-scale migrations of moments/pinocchio/geppetto call sites, to avoid compounding churn.
---


# Analysis: Revise Turn.Data/Metadata access API (3 key types w/ Get/Set) + migration strategy

## Context / Why this ticket exists

Ticket `002-IMPLEMENT-TYPE-DATA-ACCESSOR` implemented the core refactor: `Turn.Data`, `Turn.Metadata`, and `Block.Metadata` are now **opaque wrappers** and access is mediated via typed keys and typed accessors. During that work we hit (and documented) the Go constraint that methods cannot declare their own type parameters, which led to the current API shape: `turns.DataGet/DataSet`, `turns.MetadataGet/MetadataSet`, `turns.BlockMetadataGet/BlockMetadataSet` with a single `Key[T]`.

While exploring ergonomics, we validated a key insight: **methods on a generic receiver type are legal** in Go (e.g. `func (k Key[T]) Get(d Data) ...`). The blocker was not generics per se—it was *generic methods on non-generic receiver types* like `Data`.

However, Go does not support method overloading, so a single `Key[T].Get(...)` cannot target `Data`, `Metadata`, and `BlockMetadata` with the same method name. That pushes us toward the cleanest ergonomic + correctness improvement:

- **Split typed keys into three store-specific key types**, each with `.Get/.Set` methods:
  - `DataKey[T]` for `Turn.Data`
  - `TurnMetaKey[T]` for `Turn.Metadata`
  - `BlockMetaKey[T]` for `Block.Metadata`

This analysis answers the meta-question: **do we restart** the in-flight refactor from the beginning with the improved design, or **retroactively refactor what we already built**?

## Current state (as of 002 diary)

Ground truth for where we are comes from `geppetto/ttmp/2025/12/22/002-IMPLEMENT.../reference/01-diary.md`:

- **Wrappers are already in place** (breaking change landed):
  - `Turn.Data` is `turns.Data` (opaque)
  - `Turn.Metadata` is `turns.Metadata` (opaque)
  - `Block.Metadata` is `turns.BlockMetadata` (opaque)
- **Typed access currently exists as generic functions**:
  - `turns.DataGet/DataSet`
  - `turns.MetadataGet/MetadataSet`
  - `turns.BlockMetadataGet/BlockMetadataSet`
- **Keys are already namespaced+versioned**:
  - Format: `"namespace.value@vN"`
  - Geppetto keys moved to namespace/value const model (and `engine.KeyToolConfig` exists to avoid import cycle)
  - Moments keys migration started (and several high-impact middlewares were already migrated)
- **Helper functions were removed** (per breaking-change spec), and call sites replaced with wrapper API.
- **We are far from done**:
  - Moments has the largest remaining surface area (many more call sites to migrate)
  - Linter enhancement work is still pending/partial
  - There are special-case components (e.g. turn-data compression middleware) still needing refactor

In other words: we have a working foundation, partial call-site migrations, and significant remaining work.

### 002 work inventory (what is already “real” and shouldn’t be thrown away)

From the 002 diary, the following substantial chunks are already completed and validated in code:

- **Core structural refactor** (geppetto):
  - Opaque wrappers for `Turn.Data`, `Turn.Metadata`, `Block.Metadata`
  - YAML marshal/unmarshal behavior preserved (empty wrappers omit)
  - JSON serializability validation on Set
  - Generic-method limitation handled via package-level generic functions
- **Key model migrated** (geppetto):
  - Namespaced + versioned keys (`namespace.value@vN`)
  - Import-cycle handled via engine-owned key for `ToolConfig`
- **Helper removal + call-site migrations**:
  - Geppetto call sites migrated off raw map access and removed helper functions
  - Pinocchio migrated off removed helpers and onto wrapper API
  - Moments key package migrated + several high-traffic middlewares migrated (current_user, thinkingmode, promptutil, memory_extraction, memory_context)

This is “foundational plumbing” work. Rewinding would mostly re-run the same change set, plus reintroduce risk in the already-migrated middleware semantics.

### PoC checkpoint (ticket 003)

We additionally validated the revised design with a compile+test PoC:

- `geppetto/pkg/turns/poc_split_key_types_test.go`
- `cd geppetto && go test ./pkg/turns -count=1`

This confirms the split-key-family API is mechanically implementable against the current code shape.

## Revised design: 3 key types with `.Get/.Set` methods

The improved design is:

- Replace single `Key[T]` with three typed key wrappers:
  - `DataKey[T]`
  - `TurnMetaKey[T]`
  - `BlockMetaKey[T]`
- Each key type exposes:
  - `.Get(store)` → `(T, bool, error)`
  - `.Set(&store, value)` → `error`
- Wrapper types (`Data`, `Metadata`, `BlockMetadata`) remain unchanged.

### Why this is better than “single Key + functions”

- **Ergonomics**: Call sites read naturally and reduce boilerplate:
  - `mode, ok, err := turnkeys.ThinkingMode.Get(t.Data)`
  - `err := turnkeys.ThinkingMode.Set(&t.Data, mode)`
  - vs `turns.DataGet(t.Data, turnkeys.ThinkingMode)` / `turns.DataSet(&t.Data, ...)`
- **Correctness / compile-time safety**: A `TurnMetaKey[T]` cannot be accidentally used against `Turn.Data` (and vice versa). Today, because metadata keys are effectively “the same” underlying ID (and can be cast), it’s easier to make a category error.
- **Linting becomes sharper**: We can enforce that the *store-specific* key constructor is used (`DataK/TurnMetaK/BlockMetaK`), and ban casts between key ID types.
- **Avoids the Go generic-method limitation** cleanly: methods live on generic receiver types, not on `Data`/`Metadata`/`BlockMetadata`.

## Restart vs refactor: decision analysis

### Option A: Restart from the beginning (revert 002 work, re-implement)

**What it means**

- Roll back to pre-wrapper state, then re-apply the entire refactor with the revised key design from day one.

**Pros**

- Conceptually “clean slate” with one consistent design throughout.
- Potentially simpler story for reviewers (one big change set with final API).

**Cons**

- **Throws away validated work**:
  - wrapper design, YAML marshal/unmarshal, key namespacing decisions, import-cycle resolution, and multiple migrated call sites
- **High risk of reintroducing bugs already burned down** in 002 (serde behavior, idempotency logic in middleware migrations, etc.)
- **Massive churn**: you still need to redo all the same call-site migrations, plus re-land the same helper removals, plus re-sync with moments/pinocchio changes.
- **No real technical upside** vs refactor: the bulk of the work done is not invalidated by the key-type split.

**Conclusion**

Restarting is only justified if the existing foundation is fundamentally wrong (it isn’t). It is mostly process/psychological cleanliness, not technical necessity.

### Option B: Retroactively refactor what we have (introduce the new key types and migrate)

**What it means**

- Keep wrappers and current key IDs and YAML behavior intact.
- Change the typed-key API surface:
  - introduce 3 key types + constructors
  - migrate key definitions + call sites to the method form
  - remove (or keep temporarily, then remove) the function form

**Pros**

- **Preserves already-landed foundation** (wrappers, YAML behavior, keys, major call-site migrations).
- **The change is mostly mechanical**:
  - `turns.DataGet(t.Data, K)` → `K.Get(t.Data)`
  - `turns.DataSet(&t.Data, K, v)` → `K.Set(&t.Data, v)`
  - similarly for metadata and block metadata
- **Best timing is now**: we are “far from done”, so revising now reduces total churn compared to doing it after migrating the remaining ~100+ moments sites.
- Enables better long-term guarantees and a cleaner public API.

**Cons / risks**

- Requires **touching many files** again (churn), including already-migrated ones.
- If we insist on “no compatibility even transiently”, we may need a single large atomic change.
- Tooling: a mechanical migration is easy to botch if done ad hoc; better to use either:
  - a small `go/analysis` refactoring tool (see draft design doc in this ticket), or
  - carefully-scoped scripted rewrite + compilation gate.

**Conclusion**

Retroactive refactor is the pragmatic choice. It keeps momentum and reduces overall rework, while still moving us to the better API.

## Recommendation

**Do NOT restart.** Proceed with a **retroactive refactor** to the “3 key types + key methods” API now, before migrating more of moments.

The key reason: the foundation is correct; the API surface is what we’re improving. A restart would re-run the entire migration gauntlet unnecessarily.

## Proposed migration strategy (minimize pain)

This is a suggested execution approach; it does not commit us to keeping backward compatibility long-term.

### Phase 0: Align design docs + linter rules

- Update the final design doc (`001`) to the 3-key-types design (done separately in this workstream).
- Update/extend `turnsdatalint` rule spec accordingly (ban cross-store key casts, require store-specific constructors).

### Phase 1: Implement new key types in `turns` (mechanically derivable from current implementation)

- Add:
  - `DataKey[T]`, `TurnMetaKey[T]`, `BlockMetaKey[T]`
  - constructors `DataK/TurnMetaK/BlockMetaK`
  - `.Get/.Set` methods
- Decide whether `DataGet/DataSet/...` remain as:
  - short-lived migration helpers (immediately removed after migration), or
  - permanent convenience wrappers (not recommended if we want a single canonical style).

### Phase 2: Migrate key definitions first (low risk, unblocks call-site rewrite)

- `geppetto/pkg/turns/keys.go`
- `geppetto/pkg/inference/engine/turnkeys.go`
- `moments/backend/pkg/turnkeys/*.go`
- any pinocchio-local keys

### Phase 3: Migrate call sites (bulk mechanical rewrite)

- Use either:
  - a `go/analysis` refactoring tool (recommended for safety), or
  - a deterministic search/replace strategy with compilation as the gate.
- Gate each repo chunk with `go test ./... -count=1` to avoid silent breakage.

### Phase 4: Remove old API surface

- Remove `DataGet/DataSet/...` if we decide key methods are canonical.
- Update docs and examples to avoid drifting back to the function style.

## What warrants a second pair of eyes (review-critical)

- **Atomicity vs incremental compatibility**: do we allow an intermediate commit where both APIs exist to ease migration, as long as the final state removes the old one?
- **Key ownership and import cycles**: confirm the engine-owned `ToolConfig` key story still works cleanly with `DataKey[ToolConfig]`.
- **Linter enforcement**: ensure the revised linter rules make it *harder* (not easier) to bypass invariants, especially around:
  - key construction
  - cross-store key misuse
  - lingering direct map access in YAML or helper code

## Next actions (immediate)

- Fill out the ticket `003` design docs with the exact API + constructor naming decisions, and decide whether we are building a refactoring tool or doing a scripted rewrite.
- Pause further moments middleware migrations until the API decision lands (otherwise churn compounds).

