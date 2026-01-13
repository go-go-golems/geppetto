---
Title: Quiz - Yesterday's Work on Ticket 003
Ticket: 003-REVISE-DATA-ACCESS-API
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
    - quiz
DocType: reference
Intent: learning
---

# Quiz: Revising the Typed Turn Data Access API

This quiz walks you through yesterday's work on ticket 003, which focused on revising the typed Turn data access API from a function-based approach to a key-method-based approach with three store-specific key types.

## Part 1: Understanding the Core Problem

Before diving into the technical details, it's important to understand what problem we're solving. Ticket 003 is about revising the typed Turn data access API. The current implementation (see [`geppetto/pkg/turns/types.go`](../../../../../../pkg/turns/types.go)) uses a single `Key[T]` type with package-level functions like `DataGet` and `DataSet`. The goal is to move to a more ergonomic API where keys themselves have methods, resulting in call sites like `turnkeys.SomeDataKey.Get(t.Data)` instead of `turns.DataGet(t.Data, turnkeys.SomeKey)`. See the [API ergonomics principles](../../../../../../docs/concepts/api-ergonomics-principles.md) guide for background on method-based vs function-based APIs.

This revision introduces three separate key types, each specific to a different store: `DataKey[T]` for `Turn.Data`, `TurnMetaKey[T]` for `Turn.Metadata`, and `BlockMetaKey[T]` for `Block.Metadata`. The updated design is documented in the [final design doc](../001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md), and understanding why we need three types instead of one is crucial to grasping the design decisions made. For context on what was implemented previously, see the [002 diary](../002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/reference/01-diary.md). The [store-specific key types](../../../../../../docs/concepts/store-specific-key-types.md) documentation explains the rationale for this split.

<form id="part1-core-problem">
name: Understanding the Core Problem
description: Test your understanding of what problem we're solving and why

fields:
  - name: q1_goal
    label: What is the primary goal of ticket 003?
    type: radio
    options:
      - To add new features to the Turn data access API
      - To revise the API from `Key[T]` + `DataGet/DataSet` functions to three store-specific key types with `.Get/.Set` methods
      - To fix bugs in the existing implementation
      - To improve performance of data access
    correct: To revise the API from `Key[T]` + `DataGet/DataSet` functions to three store-specific key types with `.Get/.Set` methods
    required: true

  - name: q2_target_api
    label: What is the intended end-state call-site style?
    type: radio
    options:
      - `v, ok, err := turns.DataGet(t.Data, turnkeys.SomeKey)`
      - `v, ok, err := turnkeys.SomeDataKey.Get(t.Data)`
      - `v := t.Data.Get(turnkeys.SomeKey)`
      - `v, ok := turnkeys.SomeKey.Get(t.Data)`
    correct: `v, ok, err := turnkeys.SomeDataKey.Get(t.Data)`
    required: true

  - name: q3_key_types
    label: What are the three store-specific key types being introduced?
    type: checkbox
    options:
      - `DataKey[T]` for `Turn.Data`
      - `TurnMetaKey[T]` for `Turn.Metadata`
      - `BlockMetaKey[T]` for `Block.Metadata`
      - `Key[T]` for all stores
      - `MetaKey[T]` for metadata stores
    correct:
      - `DataKey[T]` for `Turn.Data`
      - `TurnMetaKey[T]` for `Turn.Metadata`
      - `BlockMetaKey[T]` for `Block.Metadata`
    required: true
</form>

## Part 2: The Design Constraint That Led to Three Key Families

The decision to split into three key families wasn't arbitrary—it was driven by fundamental constraints in Go's type system. The "aha moment" came when we realized that the blocker wasn't just "methods on generic types" broadly, but specifically the combination of two limitations: methods can't declare their own type parameters, and Go doesn't support method overloading. This is documented in [Step 1 of the diary](./01-diary.md#step-1-identify-the-ergonomic-constraint-no-method-overloading-and-propose-3-key-families). For background on these constraints, see the [Go generics method limitations guide](../../../../../../docs/concepts/go-generics-method-limitations.md) and [method overloading alternatives](../../../../../../docs/concepts/method-overloading-alternatives.md).

This means we can't have a single `Key[T]` type with a `Get` method that works for both `Data` and `Metadata` stores, because we'd need method overloading (which Go doesn't support) to distinguish between `Key[T].Get(Data)` and `Key[T].Get(Metadata)`. The alternatives—awkward method names like `GetFromData` or continued reliance on package-level functions—were rejected in favor of splitting into three type families, which provides compile-time category safety. The proof-of-concept validating this approach can be found in [`geppetto/pkg/turns/poc_split_key_types_test.go`](../../../../../../pkg/turns/poc_split_key_types_test.go). See also the [type family design patterns](../../../../../../docs/concepts/type-family-design-patterns.md) documentation.

<form id="part2-design-constraint">
name: The Design Constraint
description: Understanding why we need three separate key types instead of one

fields:
  - name: q4_constraint_1
    label: What was the "aha moment" that reframed the Go generics limitation?
    type: radio
    options:
      - Methods can't declare their own type parameters
      - Go doesn't support method overloading
      - Both: methods can't declare type parameters AND Go doesn't support method overloading
      - Go doesn't support generics at all
    correct: Both: methods can't declare type parameters AND Go doesn't support method overloading
    required: true

  - name: q5_constraint_2
    label: Why can't we have a single `Key[T].Get(...)` method that works for Data, Metadata, and BlockMetadata?
    type: radio
    options:
      - Because Go doesn't support generics
      - Because we can't have `Key[T].Get(Data)` and `Key[T].Get(Metadata)` coexist (no method overloading)
      - Because methods can't have type parameters
      - Because it would be too slow
    correct: Because we can't have `Key[T].Get(Data)` and `Key[T].Get(Metadata)` coexist (no method overloading)
    required: true

  - name: q6_alternative_names
    label: What alternatives were considered but rejected for a single key type?
    type: checkbox
    options:
      - Awkward method names like `GetFromData` / `GetFromMetadata`
      - Continued reliance on package-level functions
      - Using reflection to determine the store type
      - Using runtime type checking
    correct:
      - Awkward method names like `GetFromData` / `GetFromMetadata`
      - Continued reliance on package-level functions
    required: true

  - name: q7_benefit
    label: What benefit does splitting into three key families provide?
    type: radio
    options:
      - Better performance
      - Compile-time category safety (reduces category errors like using a metadata key on data)
      - Smaller binary size
      - Better error messages
    correct: Compile-time category safety (reduces category errors like using a metadata key on data)
    required: true
</form>

## Part 3: Migration Strategy Decision

Once the design direction was clear, a critical decision had to be made: should we throw away the existing work from ticket 002 and restart, or should we retrofit the revised API on top of the existing foundation? This wasn't just a technical question—it had implications for ongoing work, especially in the moments codebase which still has a large remaining migration surface area. The full analysis is documented in [`analysis/01-analysis-revise-turn-data-metadata-access-api-key-methods-migration-strategy.md`](../analysis/01-analysis-revise-turn-data-metadata-access-api-key-methods-migration-strategy.md). See also the [migration strategy patterns](../../../../../../docs/concepts/migration-strategy-patterns.md) guide.

The analysis concluded that restarting would be wasteful. The foundation built in 002—including wrappers, YAML behavior, key format, and helper removals—is correct. The API surface can be revised with a mostly-mechanical migration, and doing it now creates less churn than waiting until later. To make this feasible, a CLI refactor tool using go/packages and AST rewriting is being built (see [`geppetto/cmd/turnsrefactor`](../../../../../../cmd/turnsrefactor) and [`geppetto/pkg/analysis/turnsrefactor`](../../../../../../pkg/analysis/turnsrefactor)) to enable a deterministic, repeatable migration. This is covered in [Step 5 of the diary](./01-diary.md#step-5-build-a-cli-refactor-tool-gopackages--ast-rewrite-and-keep-repo-green-while-iterating). For background on AST rewriting approaches, see the [go/packages AST transformation guide](../../../../../../docs/concepts/go-packages-ast-transformation.md).

<form id="part3-migration-strategy">
name: Migration Strategy
description: Understanding the decision to retrofit vs restart

fields:
  - name: q8_migration_decision
    label: What was the conclusion about restarting vs retrofitting the 002 work?
    type: radio
    options:
      - Throw away 002 work and restart with the revised API
      - Retrofit the revised API on top of the existing 002 foundation
      - Keep both APIs indefinitely
      - Abandon the revision
    correct: Retrofit the revised API on top of the existing 002 foundation
    required: true

  - name: q9_why_not_restart
    label: Why was restarting rejected?
    type: checkbox
    options:
      - The foundation (wrappers, YAML behavior, key format, helper removals) is correct
      - The API surface can be revised with a mostly-mechanical migration
      - Doing it now is less churn than later
      - The 002 work was completely wrong
      - Moments still has a large remaining migration surface area
    correct:
      - The foundation (wrappers, YAML behavior, key format, helper removals) is correct
      - The API surface can be revised with a mostly-mechanical migration
      - Doing it now is less churn than later
      - Moments still has a large remaining migration surface area
    required: true

  - name: q10_refactor_tool
    label: What approach is being taken to make the migration feasible?
    type: radio
    options:
      - Manual migration file by file
      - A CLI refactor tool using go/packages + AST rewrite
      - Waiting for an IDE refactoring feature
      - Using sed/awk scripts
    correct: A CLI refactor tool using go/packages + AST rewrite
    required: true
</form>

## Part 4: The Persistence Round-Trip Problem

A critical issue emerged when working with the typed access API: what happens to structured values after they go through YAML or JSON serialization and deserialization? The answer is that they don't come back as their original concrete struct types—instead, they come back as `map[string]any`. This breaks strict type assertions in `Get[T]`, causing failures when trying to retrieve values like `engine.ToolConfig` that have been through a round-trip. The serialization/deserialization logic lives in [`geppetto/pkg/turns/serde`](../../../../../../pkg/turns/serde). This is a common issue discussed in the [serialization round-trip type preservation](../../../../../../docs/concepts/serialization-roundtrip-type-preservation.md) guide.

The obvious solution—having `Get[T]` automatically decode into `T`—was considered but rejected. While technically possible via reflection and marshal/unmarshal, it introduces implicit behavior, adds performance costs, and creates ambiguity. This led to exploring explicit alternatives that maintain type safety without hidden costs. This problem and its implications are detailed in [Step 6 of the diary](./01-diary.md#step-6-evaluate-decode-into-t-and-design-a-per-key-codec-registry). See the [implicit vs explicit decoding trade-offs](../../../../../../docs/concepts/implicit-vs-explicit-decoding.md) documentation for more on why implicit decoding was rejected.

<form id="part4-persistence-problem">
name: The Persistence Round-Trip Problem
description: Understanding the challenge with YAML/JSON round-trips and typed access

fields:
  - name: q11_roundtrip_problem
    label: What happens to structured values after a YAML/JSON round-trip?
    type: radio
    options:
      - They stay as their original concrete struct type
      - They come back as `map[string]any` rather than their original concrete struct type
      - They become strings
      - They are lost entirely
    correct: They come back as `map[string]any` rather than their original concrete struct type
    required: true

  - name: q12_impact
    label: What is the impact of this round-trip behavior on typed `Get[T]`?
    type: radio
    options:
      - No impact, everything works fine
      - Strict type assertion in `Get[T]` fails (e.g., `engine.ToolConfig` coming back as a map)
      - Performance is improved
      - Values are automatically decoded
    correct: Strict type assertion in `Get[T]` fails (e.g., `engine.ToolConfig` coming back as a map)
    required: true

  - name: q13_implicit_decode
    label: Why was the idea of "`Get[T]` automatically decoding into `T`" rejected?
    type: checkbox
    options:
      - It introduces implicit behavior
      - It adds performance costs
      - It creates ambiguity
      - It's technically impossible
      - It would require reflection and marshal/unmarshal
    correct:
      - It introduces implicit behavior
      - It adds performance costs
      - It creates ambiguity
      - It would require reflection and marshal/unmarshal
    required: true
</form>

## Part 5: Alternatives Considered - Codec Registry vs RawMessage

Two main approaches were evaluated to solve the round-trip typing problem. The first is a codec registry: an explicit per-key registry where only selected keys opt into typed reconstruction. This avoids silent and expensive "decode everything on Get" behavior, keeps type reconstruction explicit, and respects import-cycle constraints (e.g., `engine.ToolConfig` codec must be registered from `engine`). The registry design is documented in [`design-doc/02-design-registry-based-typed-decoding-for-turns-data.md`](../design-doc/02-design-registry-based-typed-decoding-for-turns-data.md). For background on registry patterns, see the [codec registry patterns](../../../../../../docs/concepts/codec-registry-patterns.md) guide.

The second approach is storing values as `json.RawMessage` (JSON bytes) instead of `any`. This makes typed reconstruction on `Get[T]` natural and avoids needing a per-key registry, but it moves decoding into the `Get` hot path (potentially slowing middleware) and complicates human-friendly YAML unless adapters are added. The RawMessage alternative is discussed in [Step 7 of the diary](./01-diary.md#step-7-revisit-jsonrawmessage-storage-as-an-alternative-to-registry-based-decoding) and expanded in the [final design doc](../001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md). See the [RawMessage storage patterns](../../../../../../docs/concepts/rawmessage-storage-patterns.md) documentation for more details. Understanding the trade-offs between these approaches is key to making the right architectural decision, as explored in the [storage strategy comparison](../../../../../../docs/concepts/storage-strategy-comparison.md).

<form id="part5-alternatives">
name: Alternatives Considered
description: Understanding the trade-offs between different approaches

fields:
  - name: q14_registry_approach
    label: What is the codec registry approach?
    type: radio
    options:
      - Store everything as JSON bytes
      - An explicit per-key codec registry so only selected keys opt into typed reconstruction
      - Automatic decoding for all keys
      - No decoding at all
    correct: An explicit per-key codec registry so only selected keys opt into typed reconstruction
    required: true

  - name: q15_registry_benefits
    label: What are the benefits of the codec registry approach?
    type: checkbox
    options:
      - Avoids silent and expensive "decode everything on Get" behavior
      - Keeps type reconstruction explicit and per-key
      - Respects import-cycle constraints (e.g., `engine.ToolConfig` codec must be registered from `engine`)
      - Makes all reads faster
      - Requires no registration code
    correct:
      - Avoids silent and expensive "decode everything on Get" behavior
      - Keeps type reconstruction explicit and per-key
      - Respects import-cycle constraints (e.g., `engine.ToolConfig` codec must be registered from `engine`)
    required: true

  - name: q16_rawmessage_approach
    label: What is the `json.RawMessage` storage approach?
    type: radio
    options:
      - Storing values as `any` type
      - Storing values as `json.RawMessage` (JSON bytes) instead of `any`
      - Storing values as strings
      - Not storing values at all
    correct: Storing values as `json.RawMessage` (JSON bytes) instead of `any`
    required: true

  - name: q17_rawmessage_pros
    label: What are the advantages of `json.RawMessage` storage?
    type: checkbox
    options:
      - Makes typed reconstruction on `Get[T]` natural (always unmarshal bytes into `T`)
      - Avoids needing a per-key codec registry for common struct values
      - Strong round-trip typing
      - Makes all reads faster
      - No YAML readability concerns
    correct:
      - Makes typed reconstruction on `Get[T]` natural (always unmarshal bytes into `T`)
      - Avoids needing a per-key codec registry for common struct values
      - Strong round-trip typing
    required: true

  - name: q18_rawmessage_cons
    label: What are the disadvantages of `json.RawMessage` storage?
    type: checkbox
    options:
      - Decoding moves into the `Get` hot path and can slow middleware
      - Complicates "human-friendly YAML" unless adapters are added
      - Requires a codec registry anyway
      - Makes reads slower
      - Breaks existing code
    correct:
      - Decoding moves into the `Get` hot path and can slow middleware
      - Complicates "human-friendly YAML" unless adapters are added
    required: true
</form>

## Part 6: YAML Bridge and Usage Analysis

If we adopt `json.RawMessage` storage, can we still produce readable YAML snapshots? Yes—by decoding JSON bytes into `any` when marshaling to YAML, and on YAML input constraining ourselves to "JSON-shaped YAML" (string keys, JSON primitives, sequences, mappings) and re-encoding each value back into JSON bytes. This keeps canonical internal storage as JSON bytes while maintaining YAML readability for debugging, at the cost of extra encode/decode work on snapshot boundaries. The bridging algorithm is detailed in [Step 8 of the diary](./01-diary.md#step-8-spell-out-the-rawmessage--yaml-bridge-decode-to-any-for-yaml-output-re-encode-on-yaml-input) and the [registry design doc](../design-doc/02-design-registry-based-typed-decoding-for-turns-data.md). See the [YAML-JSON bridge patterns](../../../../../../docs/concepts/yaml-json-bridge-patterns.md) guide for implementation details.

An important discovery was made about where YAML is actually used: it's primarily in geppetto tooling (fixtures and `llm-runner` artifact generation), not in moments or pinocchio runtime code. This means "YAML readability" can be treated as a tooling concern rather than a core product requirement, which affects how we evaluate the trade-offs. The full usage analysis is documented in [`analysis/02-analysis-where-turn-yaml-serde-is-used-geppetto-vs-moments-vs-pinocchio.md`](../analysis/02-analysis-where-turn-yaml-serde-is-used-geppetto-vs-moments-vs-pinocchio.md), as described in [Step 9 of the diary](./01-diary.md#step-9-inventory-real-usage-of-turn-yaml-serde-across-geppettomomentspinocchio). This aligns with the [tooling vs runtime concerns](../../../../../../docs/concepts/tooling-vs-runtime-concerns.md) documentation.

<form id="part6-yaml-bridge">
name: YAML Bridge and Usage
description: Understanding how YAML readability can be preserved and where YAML is actually used

fields:
  - name: q19_yaml_bridge
    label: How can readable YAML be preserved if we store values as `json.RawMessage`?
    type: radio
    options:
      - We can't preserve YAML readability with RawMessage
      - Decode JSON bytes into `any` when marshaling to YAML, and on YAML input constrain to "JSON-shaped YAML" and re-encode back into JSON bytes
      - Store everything twice (once as bytes, once as YAML)
      - Use a different storage format
    correct: Decode JSON bytes into `any` when marshaling to YAML, and on YAML input constrain to "JSON-shaped YAML" and re-encode back into JSON bytes
    required: true

  - name: q20_yaml_constraint
    label: What constraint does the RawMessage ↔ YAML bridge require?
    type: radio
    options:
      - YAML input must be JSON-compatible (string keys, JSON primitives, sequences, mappings)
      - YAML input can be any valid YAML
      - YAML input must be valid Go structs
      - No constraints
    correct: YAML input must be JSON-compatible (string keys, JSON primitives, sequences, mappings)
    required: true

  - name: q21_yaml_usage
    label: Where is Turn YAML serde actually used in the codebase?
    type: checkbox
    options:
      - In geppetto tooling (fixtures + `llm-runner` artifact generation and run viewer parsing)
      - In moments runtime code
      - In pinocchio runtime code
      - Nowhere, it's unused
    correct:
      - In geppetto tooling (fixtures + `llm-runner` artifact generation and run viewer parsing)
    required: true

  - name: q22_yaml_implication
    label: What does the YAML usage analysis imply?
    type: radio
    options:
      - YAML is a critical runtime dependency
      - YAML is an "artifact interchange" format for geppetto tooling, so "YAML readability" can be treated as a tooling concern rather than a core product requirement
      - YAML must be removed entirely
      - YAML is used everywhere
    correct: YAML is an "artifact interchange" format for geppetto tooling, so "YAML readability" can be treated as a tooling concern rather than a core product requirement
    required: true
</form>

## Part 7: Key Type Registry Evolution

The codec registry concept evolved into something more comprehensive: a full "key type registry" (or key schema registry). The key insight was that the registry already needs to know how to get from decoded forms to typed values, so we can also use it at YAML import time to decode directly into the expected type, and at `Set` time to enforce type correctness and serializability constraints. This evolution is documented in [Step 10 of the diary](./01-diary.md#step-10-extend-the-registry-to-be-a-key-type-registry-use-it-for-typed-yaml-import--serializability-validation). See the [schema registry evolution](../../../../../../docs/concepts/schema-registry-evolution.md) guide for the design rationale.

This provides a "third path" between the extremes of `any` storage (fast reads but weak round-trip typing) and `json.RawMessage` storage (strong round-trip typing but decode cost in hot paths). A key type registry can keep `any` storage fast while still reconstructing typed values at import boundaries and enforcing schema-aware serializability where it matters. The registry design and its three uses are detailed in [`design-doc/02-design-registry-based-typed-decoding-for-turns-data.md`](../design-doc/02-design-registry-based-typed-decoding-for-turns-data.md). The preferred approach for registry wiring is explicit (not init-time registration) to avoid hidden side effects, and would integrate with the wrapper types in [`geppetto/pkg/turns/types.go`](../../../../../../pkg/turns/types.go). See the [explicit dependency injection patterns](../../../../../../docs/concepts/explicit-dependency-injection.md) documentation for why explicit wiring is preferred.

<form id="part7-registry-evolution">
name: Key Type Registry Evolution
description: Understanding how the registry concept evolved

fields:
  - name: q23_registry_evolution
    label: How was the codec registry concept generalized?
    type: radio
    options:
      - It became a full "key type registry" (key schema registry)
      - It was abandoned
      - It became simpler
      - It merged with RawMessage storage
    correct: It became a full "key type registry" (key schema registry)
    required: true

  - name: q24_registry_uses
    label: What are the three uses of the key type registry?
    type: checkbox
    options:
      - Decoding from decoded forms to typed values (original use)
      - Using it at YAML import time to decode directly into the expected type
      - Using it at Set time to enforce type correctness and serializability constraints
      - Caching decoded values
      - Performance optimization
    correct:
      - Decoding from decoded forms to typed values (original use)
      - Using it at YAML import time to decode directly into the expected type
      - Using it at Set time to enforce type correctness and serializability constraints
    required: true

  - name: q25_third_path
    label: What does the key type registry provide as a "third path"?
    type: radio
    options:
      - A compromise between `any` storage and `json.RawMessage` storage
      - A way to keep `any` storage fast while still reconstructing typed values at import boundaries and enforcing schema-aware serializability
      - Both of the above
      - A replacement for both approaches
    correct: Both of the above
    required: true

  - name: q26_registry_wiring
    label: What is the preferred approach for registry wiring?
    type: radio
    options:
      - Init-time registration (hidden side effects)
      - Explicit wiring (preferred) to avoid hidden side effects
      - Automatic discovery via reflection
      - No registration needed
    correct: Explicit wiring (preferred) to avoid hidden side effects
    required: true
</form>

## Part 8: Implementation Steps and Proof of Concept

Before changing production code or running large-scale migrations, a proof-of-concept was implemented to validate feasibility. The PoC defined the three key families (`DataKey`, `TurnMetaKey`, `BlockMetaKey`) and wired their `.Get/.Set` methods to delegate through the existing `Key[T]` + `DataGet/DataSet/...` functions. This kept it as a pure PoC without rewriting production types. The PoC implementation is in [`geppetto/pkg/turns/poc_split_key_types_test.go`](../../../../../../pkg/turns/poc_split_key_types_test.go), as described in [Step 3 of the diary](./01-diary.md#step-3-proof-of-concept-go-implementation-test-only-for-split-key-types---getset-methods). See the [proof-of-concept patterns](../../../../../../docs/concepts/proof-of-concept-patterns.md) guide for best practices on validating designs before large-scale changes.

The PoC validated that the design is mechanically implementable: key receiver methods compile, store-specific key families can be enforced by types (compile-time category separation), and the key-method approach matches current semantics including type-mismatch errors and serializability validation. Additionally, the final design doc was updated before implementing code to prevent design drift—if docs still show the function style, ongoing migrations will keep compounding churn. The design doc update is covered in [Step 2 of the diary](./01-diary.md#step-2-update-the-final-design-doc-001-to-the-3-key-types---key-methods-api) and the updated doc is at [`../001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md`](../001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md). See the [design doc maintenance guide](../../../../../../docs/concepts/design-doc-maintenance.md) for why keeping docs in sync prevents drift.

<form id="part8-implementation">
name: Implementation Steps
description: Understanding what was actually built and validated

fields:
  - name: q27_poc_purpose
    label: What was the purpose of the proof-of-concept implementation?
    type: radio
    options:
      - To replace the production code immediately
      - To validate feasibility before changing production code or running large-scale migrations
      - To fix bugs
      - To improve performance
    correct: To validate feasibility before changing production code or running large-scale migrations
    required: true

  - name: q28_poc_approach
    label: How did the PoC validate the design?
    type: radio
    options:
      - By rewriting all production code
      - By defining the three key families and wiring their `.Get/.Set` methods to delegate through the existing `Key[T]` + `DataGet/DataSet/...` functions
      - By using reflection
      - By copying code from another project
    correct: By defining the three key families and wiring their `.Get/.Set` methods to delegate through the existing `Key[T]` + `DataGet/DataSet/...` functions
    required: true

  - name: q29_poc_result
    label: What was the result of the PoC?
    type: checkbox
    options:
      - `go test` passed
      - The key-method approach matches current semantics (including type-mismatch errors and serializability validation)
      - The design is mechanically implementable
      - Key receiver methods compile
      - Store-specific key families can be enforced by types (compile-time category separation)
      - It failed to compile
    correct:
      - `go test` passed
      - The key-method approach matches current semantics (including type-mismatch errors and serializability validation)
      - The design is mechanically implementable
      - Key receiver methods compile
      - Store-specific key families can be enforced by types (compile-time category separation)
    required: true

  - name: q30_design_doc_update
    label: Why was the final design doc (`001`) updated before implementing the code?
    type: radio
    options:
      - To prevent design drift: if docs still show function style, ongoing migrations will keep compounding churn
      - Because the code was already written
      - To fix typos
      - To add examples
    correct: To prevent design drift: if docs still show function style, ongoing migrations will keep compounding churn
    required: true
</form>

## Part 9: Key Insights and Learnings

Several important insights emerged from this work. First, there's a crucial separation of concerns: typed key methods (about call-site ergonomics and category safety) are separate from typed reconstruction after persistence (which needs explicit schema/codec ownership). These are different problems requiring different solutions. This insight is captured in [Step 6 of the diary](./01-diary.md#step-6-evaluate-decode-into-t-and-design-a-per-key-codec-registry). See the [separation of concerns patterns](../../../../../../docs/concepts/separation-of-concerns-patterns.md) guide for more on this principle.

The compression section of the design doc provided a useful forcing function—it made it obvious that Data keys should be distinct from metadata keys. This is mentioned in [Step 2 of the diary](./01-diary.md#step-2-update-the-final-design-doc-001-to-the-3-key-types---key-methods-api) and relates to how keys are handled in compression scenarios. See the [compression key design](../../../../../../docs/concepts/compression-key-design.md) documentation for details. While building the refactor tool, an unrelated-but-blocking issue surfaced: test failures in [`geppetto/pkg/turns/serde/serde_test.go`](../../../../../../pkg/turns/serde/serde_test.go) due to Go generic type inference. This taught us that some existing tests depend on subtle generic inference behavior, and as we change APIs, we should expect and quickly patch these to keep the suite trustworthy. This issue is documented in [Step 5 of the diary](./01-diary.md#step-5-build-a-cli-refactor-tool-gopackages--ast-rewrite-and-keep-repo-green-while-iterating). See the [test maintenance during API changes](../../../../../../docs/concepts/test-maintenance-during-api-changes.md) guide for strategies.

<form id="part9-insights">
name: Key Insights and Learnings
description: Understanding the deeper insights from yesterday's work

fields:
  - name: q31_separation_of_concerns
    label: What two concerns were identified as separate?
    type: checkbox
    options:
      - Typed key methods (about call-site ergonomics and category safety)
      - Typed reconstruction after persistence (needs explicit schema/codec ownership)
      - Performance and correctness
      - YAML and JSON
    correct:
      - Typed key methods (about call-site ergonomics and category safety)
      - Typed reconstruction after persistence (needs explicit schema/codec ownership)
    required: true

  - name: q32_compression_insight
    label: What insight did the compression section provide?
    type: radio
    options:
      - Data keys should be compressed differently
      - Data keys should be distinct from metadata keys (makes it obvious they're different)
      - Compression is not needed
      - All keys should be compressed the same way
    correct: Data keys should be distinct from metadata keys (makes it obvious they're different)
    required: true

  - name: q33_test_issue
    label: What unrelated-but-blocking issue was encountered while building the refactor tool?
    type: radio
    options:
      - A compilation error in the refactor tool itself
      - Test failures in `turns/serde` due to Go generic type inference
      - Performance issues
      - Import cycle errors
    correct: Test failures in `turns/serde` due to Go generic type inference
    required: true

  - name: q34_lesson_learned
    label: What lesson was learned about tests and API changes?
    type: radio
    options:
      - Tests never break when APIs change
      - Some existing tests depend on subtle generic inference behavior; as we change APIs, we should expect and quickly patch these to keep the suite trustworthy
      - All tests should be rewritten
      - Tests are not important
    correct: Some existing tests depend on subtle generic inference behavior; as we change APIs, we should expect and quickly patch these to keep the suite trustworthy
    required: true
</form>

## Part 10: Open Questions and Future Work

Several open questions remain that need to be resolved before finalizing the implementation. For the registry approach, we need to decide whether to cache decoded values back into the wrapper map (mutation-on-read tradeoff), whether the registry should be global or passed as an explicit dependency into serde/load paths, and whether `UnmarshalYAML` should cache decoded typed values (mutate-on-import). See the [mutation-on-read patterns documentation](../../../../../../docs/concepts/mutation-on-read-patterns.md) for discussion of these trade-offs.

For `json.RawMessage` storage, open questions include whether the project's priorities favor hot-path performance (prefer `any` + optional registry) versus persistence fidelity (prefer RawMessage), whether "JSON-shaped YAML only" is acceptable for all current YAML usage, and whether we should enforce JSON-shaped YAML strictly (error on non-JSON YAML) versus best-effort conversion. The [performance vs fidelity trade-offs guide](../../../../../../docs/concepts/performance-vs-fidelity-tradeoffs.md) explores these considerations. Future work includes implementing the real production API, updating canonical key definitions, migrating call sites via the refactoring tool, fixing test failures, and deciding between registry and RawMessage approaches. The [migration checklist](../../../../../../docs/concepts/turn-api-migration-checklist.md) outlines the remaining steps.

<form id="part10-future-work">
name: Open Questions and Future Work
description: Understanding what remains to be decided and implemented

fields:
  - name: q35_open_question_1
    label: What is an open question about the registry approach?
    type: checkbox
    options:
      - Whether we should cache decoded values back into the wrapper map (mutation-on-read tradeoff)
      - Whether registry should be global or passed as explicit dependency into serde/load paths
      - Whether UnmarshalYAML should cache decoded typed values (mutate-on-import)
      - Whether we should use RawMessage instead
    correct:
      - Whether we should cache decoded values back into the wrapper map (mutation-on-read tradeoff)
      - Whether registry should be global or passed as explicit dependency into serde/load paths
      - Whether UnmarshalYAML should cache decoded typed values (mutate-on-import)
    required: true

  - name: q36_open_question_2
    label: What is an open question about RawMessage storage?
    type: checkbox
    options:
      - Whether the project's priorities favor hot-path performance (prefer `any` + optional registry) vs persistence fidelity (prefer RawMessage)
      - Whether "JSON-shaped YAML only" is acceptable for all current YAML usage
      - Whether we should enforce JSON-shaped YAML strictly (error on non-JSON YAML) vs best-effort conversion
      - Whether RawMessage is faster
    correct:
      - Whether the project's priorities favor hot-path performance (prefer `any` + optional registry) vs persistence fidelity (prefer RawMessage)
      - Whether "JSON-shaped YAML only" is acceptable for all current YAML usage
      - Whether we should enforce JSON-shaped YAML strictly (error on non-JSON YAML) vs best-effort conversion
    required: true

  - name: q37_future_work
    label: What future work remains?
    type: checkbox
    options:
      - Implement the real production API in `turns` (introduce the 3 key types in production code)
      - Update canonical key definitions in geppetto/moments/pinocchio
      - Migrate call sites (ideally via the refactoring tool)
      - Fix the test failures in `turns/serde`
      - Decide between registry and RawMessage approaches
    correct:
      - Implement the real production API in `turns` (introduce the 3 key types in production code)
      - Update canonical key definitions in geppetto/moments/pinocchio
      - Migrate call sites (ideally via the refactoring tool)
      - Fix the test failures in `turns/serde`
      - Decide between registry and RawMessage approaches
    required: true
</form>

---

## Summary

This quiz covered the key aspects of yesterday's work on ticket 003:

1. **The Core Problem**: Revising the API from function-based to key-method-based with three store-specific key types
2. **The Design Constraint**: Go's lack of method overloading led to the need for three key families
3. **Migration Strategy**: Decision to retrofit rather than restart
4. **Persistence Challenges**: YAML/JSON round-trips break typed access
5. **Alternatives**: Codec registry vs `json.RawMessage` storage
6. **YAML Bridge**: How to preserve readability with RawMessage
7. **Registry Evolution**: From codec registry to full key type registry
8. **Implementation**: PoC validation and design doc updates
9. **Key Insights**: Separation of concerns and test considerations
10. **Future Work**: Open questions and remaining implementation tasks

Review your answers and check the diary document for detailed explanations of each step!

