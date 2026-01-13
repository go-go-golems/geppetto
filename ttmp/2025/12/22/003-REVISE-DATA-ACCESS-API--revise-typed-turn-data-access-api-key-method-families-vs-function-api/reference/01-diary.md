---
Title: Diary
Ticket: 003-REVISE-DATA-ACCESS-API
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
      Note: Current implementation (wrappers + DataGet/DataSet API) that we are evaluating to revise.
    - Path: geppetto/pkg/turns/poc_split_key_types_test.go
      Note: Proof-of-concept validating the 3 key types + Get/Set methods approach against current implementation.
    - Path: geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md
      Note: Updated design doc proposing 3 key types with Get/Set methods (DataKey/TurnMetaKey/BlockMetaKey).
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/reference/01-diary.md
      Note: Ground truth for what was implemented so far in 002 (commits, scope, and remaining work).
    - Path: geppetto/ttmp/2025/12/22/003-REVISE-DATA-ACCESS-API--revise-typed-turn-data-access-api-key-method-families-vs-function-api/analysis/01-analysis-revise-turn-data-metadata-access-api-key-methods-migration-strategy.md
      Note: Decision analysis: restart vs retroactive refactor and suggested migration strategy.
ExternalSources: []
Summary: "Diary for revising typed Turn.Data/Metadata/Block.Metadata access API toward 3 store-specific key types with Get/Set methods."
LastUpdated: 2025-12-22T00:00:00-05:00
WhatFor: "Capture the exploration, proofs, and decisions needed to revise the API and choose a migration strategy without re-deriving the context."
WhenToUse: "Use when continuing ticket 003 work (API revision + migration strategy/tooling) to understand decisions and validate the approach."
---

# Diary

## Goal

This diary captures the exploration and decisions for revising the typed Turn data access API from a single `Key[T]` + `DataGet/DataSet` function API to **three store-specific key types** with **key receiver methods**:

- `DataKey[T]` for `Turn.Data`
- `TurnMetaKey[T]` for `Turn.Metadata`
- `BlockMetaKey[T]` for `Block.Metadata`

The intended end state is a consistent ergonomic call-site style:

- `v, ok, err := turnkeys.SomeDataKey.Get(t.Data)`
- `err := turnkeys.SomeDataKey.Set(&t.Data, v)`

…and analogous for turn metadata and block metadata.

## Step 1: Identify the ergonomic constraint (no method overloading) and propose 3 key families

This step is the “aha moment” that reframed the earlier Go generics limitation. The blocker wasn’t “methods on generic types” broadly; it was specifically that **methods can’t declare their own type parameters**, and Go also does **not** support method overloading. That means we can’t have a single `Key[T].Get(...)` that works for `Data`, `Metadata`, and `BlockMetadata` with the same method name.

Splitting keys into three store-specific types is the cleanest way to get the ergonomic `.Get/.Set` style **without** sacrificing correctness or introducing awkward method names.

**Commit (code):** N/A — analysis/design step

### What I did
- Reviewed the existing 002 implementation shape (wrappers + `DataGet/DataSet` functions).
- Confirmed the Go limitation (“methods must have no type parameters”) was already addressed in 002 by using generic functions.
- Identified the additional constraint: **no overloads**, so `Key[T].Get(Data)` and `Key[T].Get(Metadata)` cannot coexist.
- Proposed splitting the key type into:
  - `DataKey[T]`, `TurnMetaKey[T]`, `BlockMetaKey[T]`

### Why
- Provide a fluent API (`key.Get(store)` / `key.Set(&store, v)`) without:
  - illegal generic methods on non-generic receivers, and
  - unreadable method naming like `GetFromData` / `GetFromMetadata`.

### What worked
- The design aligns with Go’s constraints and can be validated with a small compile+test PoC.

### What didn't work
- Attempting to keep a single key type forces either:
  - awkward method names, or
  - continued reliance on package-level functions.

### What I learned
- The combination of:
  - “no type parameters on methods” and
  - “no overloads”
  strongly pushes toward **key families** as the most idiomatic design.

### What was tricky to build
- Ensuring the split key families still preserve the existing “opaque wrappers + no bypass” contract.

### What warrants a second pair of eyes
- Confirm the split key families meaningfully reduce category errors (using a metadata key on data) and don’t complicate import-cycle management (e.g. `engine.KeyToolConfig`).

### What should be done in the future
- N/A

### Code review instructions
- Read the 002 diary section about generic methods and the current API surface, then review the updated 001 design doc sections showing the split key types.

## Step 2: Update the final design doc (`001`) to the 3 key types + key methods API

This step updated the canonical “final design” documentation so that future work does not continue implementing the function API by inertia. The doc now describes the split key types and shows call-site patterns using `.Get/.Set`.

**Commit (docs):** N/A — documentation update only

### What I did
- Updated `001` final design doc to:
  - replace `Key[T]` with `DataKey[T]`, `TurnMetaKey[T]`, `BlockMetaKey[T]`
  - replace `DataGet/DataSet` usage in examples with `.Get/.Set` on keys
  - adjust linter rule wording to enforce `DataK/TurnMetaK/BlockMetaK`
  - adjust compression section to use `[]turns.DataKey[any]`

### Why
- Prevent design drift: if the docs still show the function style, ongoing migrations in moments will keep compounding churn.

### What worked
- The doc can now serve as the “north star” for implementing the revised API in code.

### What didn't work
- N/A

### What I learned
- The compression section is a useful forcing function: it makes it obvious that Data keys should be distinct from metadata keys.

### What was tricky to build
- Ensuring all doc examples remain consistent after the API change (migration guide, lint rules, compression).

### What warrants a second pair of eyes
- Confirm the updated doc doesn’t accidentally imply `turn.Data.Set(...)` methods exist (the real API is key methods).

### What should be done in the future
- Update any other downstream docs or examples that still show `DataGet/DataSet` as the canonical pattern.

### Code review instructions
- Start at `001/.../03-final-design-typed-turn-data-metadata-accessors.md` and search for `DataKey[` and `.Get(` patterns.

## Step 3: Proof-of-concept Go implementation (test-only) for split key types + Get/Set methods

This step provides a concrete “does it actually compile and behave?” checkpoint. We implemented a small test-only Go file that defines `DataKey/TurnMetaKey/BlockMetaKey` and wires their `.Get/.Set` methods to the *current* implementation (delegating through the existing `Key[T]` + `DataGet/DataSet/...` functions).

The goal is to validate feasibility before changing production code or running large-scale migrations again.

**Commit (code):** N/A — exploration only (not committed by this step)

### What I did
- Added `geppetto/pkg/turns/poc_split_key_types_test.go`:
  - defined the three key families
  - implemented `.Get/.Set` methods for each
  - verified behavior with `go test`
- Fixed a test build footgun: ensured `geppetto/pkg/turns/key_methods_experiment_test.go` has a `package` clause (it was intentionally emptied earlier).

### Why
- Confirm the revised design is compatible with Go’s type system and our existing wrappers, without first doing a large refactor.

### What worked
- `cd geppetto && go test ./pkg/turns -count=1` passed.
- The key-method approach matches current semantics (including type-mismatch errors and serializability validation).

### What didn't work
- N/A (PoC compiled cleanly)

### What I learned
- The design is mechanically implementable:
  - key receiver methods compile
  - store-specific key families can be enforced by types (compile-time category separation)

### What was tricky to build
- Keeping this as a *pure PoC* by delegating to the existing implementation rather than rewriting production types in the same step.

### What warrants a second pair of eyes
- Verify that the conversion bridging used in the PoC (casting key IDs across `TurnDataKey`/`TurnMetadataKey`/`BlockMetadataKey`) is acceptable in the real implementation, or if we should restructure internals to avoid cross-casts entirely.

### What should be done in the future
- Implement the real production API in `turns`:
  - introduce the 3 key types in production code
  - update canonical key definitions in geppetto/moments/pinocchio
  - migrate call sites (ideally via a refactoring tool)

### Code review instructions
- Start in `geppetto/pkg/turns/poc_split_key_types_test.go`.
- Validate with:
  - `cd geppetto && go test ./pkg/turns -count=1`

## Step 4: Decide “restart vs retroactive refactor” for 002 work

This step created a dedicated analysis doc that evaluates whether we should throw away the in-flight 002 migration and restart with the revised API, or whether we should retrofit the revised API on top of the existing foundation.

The conclusion is to **not restart**: the foundation (wrappers, YAML behavior, key format, helper removals) is correct; the API surface can be revised with a mostly-mechanical migration, and doing it now is less churn than later.

**Commit (docs):** N/A — analysis doc

### What I did
- Wrote the analysis document in ticket 003:
  - `analysis/01-analysis-revise-turn-data-metadata-access-api-key-methods-migration-strategy.md`

### Why
- Prevent compounding churn: moments still has a large remaining migration surface area.

### What worked
- The analysis makes the tradeoff explicit and grounds the recommendation in 002’s actual progress.

### What warrants a second pair of eyes
- Confirm the recommendation fits your preferred workflow constraints (e.g. “no backwards compatibility” vs allowing a short-lived dual API to ease migration).

### What should be done in the future
- N/A

---

## Step 5: Build a CLI refactor tool (go/packages + AST rewrite) and keep repo green while iterating

This step started the actual engineering work to make the “all at once” migration feasible: a CLI tool that can rewrite `turns.{Data,Metadata,BlockMetadata}{Get,Set}` calls into the new key-method call form. While wiring the tool in, we hit an unrelated-but-blocking test issue in `turns/serde` due to Go generic type inference, and then a follow-on test assertion failure. We’ll fix these immediately so the repo stays green while we iterate on the tooling.

**Commit (code):** N/A — in-progress

### What I did
- Drafted a design doc for the refactor tool in ticket 003 (go/packages + type-aware rewrites + imports cleanup).
- Added a new CLI entrypoint `geppetto/cmd/turnsrefactor` and a library package `geppetto/pkg/analysis/turnsrefactor`.
- Ran `cd geppetto && go test ./... -count=1` and observed failures in `geppetto/pkg/turns/serde`.

### Why
- We want a deterministic, repeatable “one shot” migration with minimal human error and no long-lived dual-API period.
- Keeping tests green reduces churn and ensures we’re not accidentally breaking unrelated behavior while building tooling.

### What worked
- The new packages compile and most of geppetto tests ran.

### What didn't work
- `pkg/turns/serde/serde_test.go` failed to compile due to Go generic inference with `Key[any]` + `[]any` literal.
- After a minimal inference fix, the same test started failing at runtime (assertion) — needs investigation.

### What I learned
- Some existing tests depend on subtle generic inference behavior; as we change APIs, we should expect and quickly patch these to keep the suite trustworthy.

### What was tricky to build
- Avoiding partial migrations: tooling needs to be type-aware, but our local code edits can still trigger unrelated failures that block iteration.

### What warrants a second pair of eyes
- Whether the refactor tool should verify by re-loading packages and doing type-based verification (stronger) vs textual verification (fast but weaker).

### What should be done in the future
- N/A

---

## Step 6: Evaluate “decode into T” and design a per-key codec registry

This step addressed a core tension in the typed-access design: after YAML/JSON round-trips, structured values often come back as `map[string]any` rather than their original concrete struct type. That makes a strict type assertion in `Get[T]` fail (e.g. `engine.ToolConfig` coming back as a map).

The obvious idea—“`Get[T]` knows `T`, so it can decode into `T` automatically”—is technically possible via reflection and marshal/unmarshal, but it introduces implicit behavior, performance costs, and ambiguity. Instead, we designed an explicit **per-key codec registry** so only selected keys opt into typed reconstruction.

**Commit (docs):** N/A — design-only

### What I did
- Observed the failure mode: YAML decode yields `map[string]any` for structured values, causing typed `Get` to error.
- Wrote a dedicated design doc proposing an explicit codec registry keyed by turn key identity:
  - `design-doc/02-design-registry-based-typed-decoding-for-turns-data.md`

### Why
- Avoid silent and expensive “decode everything on Get” behavior in middleware hot paths.
- Keep type reconstruction explicit and per-key (only for keys that need round-trip fidelity).
- Respect import-cycle constraints (e.g. `engine.ToolConfig` codec must be registered from `engine`).

### What worked
- The design cleanly explains why implicit decoding is risky and gives a concrete, implementable registry API and wiring strategy.

### What didn't work
- N/A (design step)

### What I learned
- “Typed key methods” and “typed reconstruction after persistence” are separate concerns:
  - the former is about call-site ergonomics and category safety
  - the latter needs explicit schema/codec ownership to be correct

### What was tricky to build
- Balancing explicit registration (avoids hidden init side effects) with practicality (tests and CLIs must remember to call registration).

### What warrants a second pair of eyes
- Whether we should cache decoded values back into the wrapper map (mutation-on-read tradeoff).
- Whether registry should be global or passed as explicit dependency into serde/load paths.

### What should be done in the future
- If we adopt the registry, implement it in `turns` and add explicit registration for `engine.ToolConfig`.

---

## Step 7: Revisit `json.RawMessage` storage as an alternative to registry-based decoding

This step re-opened one of the debate proposals: storing values as `json.RawMessage` (JSON bytes) instead of storing `any`. The motivation is straightforward: it makes typed reconstruction on `Get[T]` natural (always unmarshal bytes into `T`), and avoids needing a per-key codec registry for common struct values.

The trade-off is equally clear: decoding moves into the `Get` hot path and can meaningfully slow middleware. It also complicates “human-friendly YAML” unless we add adapters that decode bytes for YAML presentation and re-encode on read.

**Commit (docs):** N/A — documentation update only

### What I did
- Updated the canonical final design doc (`001`) to expand the `json.RawMessage` alternative with concrete pros/cons and the YAML readability implications.
- Updated the codec-registry design doc (`003`) to clarify how RawMessage storage relates (it would reduce/replace most of the registry’s need).

### Why
- We observed in practice that strict type-assertion `Get` fails after YAML round-trip for structured values (e.g., `engine.ToolConfig` coming back as `map[string]any`).
- The RawMessage alternative is the most direct “make round-trip typed” approach, so it deserves explicit analysis in the docs.

### What worked
- The docs now clearly present RawMessage as a viable variant with explicit costs and constraints.

### What warrants a second pair of eyes
- Whether the project’s priorities favor:
  - hot-path performance (prefer `any` + optional registry) vs
  - persistence fidelity (prefer RawMessage).

### What should be done in the future
- If we decide to adopt RawMessage:
  - define the exact YAML marshaling strategy to preserve readability (decode-to-any for YAML output, encode-from-any on YAML input)
  - decide whether to add a cache layer to avoid repeated decode on `Get`.

---

## Step 8: Spell out the RawMessage ↔ YAML bridge (decode to `any` for YAML output, re-encode on YAML input)

This step answered a practical follow-up: if we store values as `json.RawMessage`, can we still produce readable YAML snapshots? Yes: we can decode JSON bytes into `any` when marshaling to YAML, and on YAML input constrain ourselves to “JSON-shaped YAML” and re-encode each value back into JSON bytes.

This keeps the canonical internal storage as JSON bytes (good for typed round-trip), while keeping YAML readable for debugging and inspection—at the cost of extra encode/decode work on snapshot boundaries.

**Commit (docs):** N/A — documentation update only

### What I did
- Updated the codec-registry design doc to include a concrete RawMessage↔YAML bridging algorithm and its constraints:
  - `design-doc/02-design-registry-based-typed-decoding-for-turns-data.md`

### Why
- We discovered in practice that YAML round-trips tend to produce decoded “map forms” that break strict typed reads.
- RawMessage storage solves typed reconstruction, but we didn’t want to assume we must give up YAML readability.

### What worked
- The updated doc provides an explicit marshal/unmarshal strategy and calls out the necessary contract:
  - YAML input must be JSON-compatible (string keys, JSON primitives, sequences, mappings)

### What warrants a second pair of eyes
- Whether “JSON-shaped YAML only” is acceptable for all current YAML usage in this repo.
- Whether we should enforce this strictly (error on non-JSON YAML) vs best-effort conversion (risk of silent lossy behavior).

---

## Step 9: Inventory real usage of Turn YAML serde across geppetto/moments/pinocchio

This step answered a key question: is Turn YAML import/export a production dependency, or mostly a developer-experience artifact? We searched the entire workspace for actual call sites of `serde.ToYAML/FromYAML/SaveTurnYAML/LoadTurnYAML` and imports of `geppetto/pkg/turns/serde`.

The outcome: YAML serde is used in **geppetto tooling** (fixtures + `llm-runner` artifact generation and run viewer parsing), but there are **no direct runtime call sites** in `moments/` or `pinocchio/` in this workspace.

**Commit (docs):** N/A — analysis doc

### What I did
- Searched for call sites of `serde.*YAML*` and the `turns/serde` import path across the workspace.
- Created an analysis document in ticket 003:
  - `analysis/02-analysis-where-turn-yaml-serde-is-used-geppetto-vs-moments-vs-pinocchio.md`

### Why
- If YAML is not a runtime dependency for moments/pinocchio, we can treat “YAML readability” as a tooling concern rather than a core product requirement.

### What I learned
- YAML is an “artifact interchange” format for geppetto’s `llm-runner` and fixture workflows (write YAML, then read it later for UI/reporting).

---

## Step 10: Extend the registry to be a key type registry (use it for typed YAML import + serializability validation)

This step generalized the codec-registry idea into a full “typekey registry” (key schema registry). The key insight: the registry already needs to know *how to get from decoded forms to typed values*, so we can also use it at YAML import time to decode directly into the expected type, and at Set time to enforce type correctness and serializability constraints.

This provides a third path between the extremes:

- `any` storage (fast reads, but weak round-trip typing)
- `json.RawMessage` storage (strong round-trip typing, but decode cost moves around)

A key type registry can keep `any` storage fast while still reconstructing typed values at import boundaries and enforcing schema-aware serializability where it matters.

**Commit (docs):** N/A — design doc update only

### What I did
- Updated `design-doc/02-design-registry-based-typed-decoding-for-turns-data.md` to:
  - reframe it as a **key type registry**
  - add a section on using it in wrapper `UnmarshalYAML` to decode values into their expected Go type
  - add a section on using it in `Set` to validate type correctness + serializability (YAML/JSON)

### Why
- We hit a concrete failure: YAML round-trip decodes structs into `map[string]any`, breaking strict typed reads.
- We want schema-aware correctness without pushing decode cost into every `Get` call.

### What warrants a second pair of eyes
- Registry wiring: do we want init-time registration or explicit wiring (preferred) to avoid hidden side effects?
- Whether UnmarshalYAML should cache decoded typed values (mutate-on-import) and whether we should ever mutate-on-read.

---

## Step 11: Restore repo health (YAML omitempty for opaque wrappers + serde round-trip expectations)

This step fixed a real breakage we hit while iterating: after introducing opaque wrapper types (with no exported fields), `yaml:",omitempty"` treated them as empty structs and omitted `data:`/`metadata:` entirely during `yaml.Marshal`. That made the YAML round-trip tests fail and obscured the intended serde contract.

We fixed this by adding `IsZero()` methods so YAML can correctly treat “non-empty internal map” as non-zero. We also clarified the serde test expectation for `engine.ToolConfig`: without a key type registry or `json.RawMessage` storage, YAML import will decode struct-shaped data as `map[string]any`, so a strict typed read should error (but the map form is still present).

**Commit (code):** 3f6d39282a5b49c608ac96870de37903348e272e — "turns: make YAML omitempty work; clarify serde roundtrip"

### What I did
- Updated `geppetto/pkg/turns/types.go`:
  - added `IsZero()` methods to `Data`, `Metadata`, `BlockMetadata`
- Updated `geppetto/pkg/turns/serde/serde_test.go`:
  - clarified that typed `Get` for `engine.ToolConfig` fails after YAML round-trip (current semantics)
  - asserted the decoded `map[string]any` form contains expected fields
- Ran:
  - `cd geppetto && go test ./... -count=1`
  - pre-commit hook (`lefthook`) succeeded (test + lint)

### Why
- Keep the repo green while we explore and land API/tooling changes.
- Preserve a clear and honest test contract: typed reconstruction of struct values requires either a registry/typekey decode at import time or RawMessage storage.

### What learned
- `gopkg.in/yaml.v3` treats “struct with no exported fields” as empty for `omitempty` unless `IsZero()` is provided.

### What was tricky to build
- Keeping “readable YAML” and “typed reconstruction” separate concerns in tests; otherwise tests accidentally force an architectural decision.

### What warrants a second pair of eyes
- Whether `IsZero()` is the preferred mechanism vs switching wrapper fields to pointers (we chose IsZero to keep field shape stable).

## Quick Reference

<!-- Provide copy/paste-ready content, API contracts, or quick-look tables -->

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->
