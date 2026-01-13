---
Title: Debate candidates (typed Turn.Data/Metadata)
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
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Lint rules for typed-key access
    - Path: geppetto/pkg/inference/toolcontext/toolcontext.go
      Note: Tool registry carried in context (runtime boundary)
    - Path: geppetto/pkg/turns/keys.go
      Note: Canonical typed string keys used today
    - Path: geppetto/pkg/turns/types.go
      Note: Current Turn/Block shapes and map fields
ExternalSources: []
Summary: Participants/perspectives for debating how to evolve Turn.Data and Turn/Block Metadata toward typed/opaque/serializable access, grounded in current Geppetto code.
LastUpdated: 2025-12-20T00:28:30-05:00
WhatFor: Kick off a structured, evidence-based debate (candidates + talking points) before changing Turn.Data/Metadata APIs.
WhenToUse: Use before drafting an RFC or starting implementation work on typed accessors / opaque bags for Turn.Data / Turn.Metadata / Block.Metadata.
---


# Debate candidates (typed Turn.Data/Metadata)

## Goal

Provide a **small set of debate participants** (perspectives) that cover the core design tensions in this ticket:

- Typed access ergonomics vs keeping the current “map bag” shape
- Enforcing a stricter key identity (`{vs, slug, version}`) vs typed string keys + linting
- Enforcing “serializable-only” values for persisted state
- Keeping **runtime-only** objects out of `Turn.Data` (already true for tools)

**Explicit scope:** this debate is about `Turn.Data`, `Turn.Metadata`, and `Block.Metadata`. We are **not** changing `Block.Payload`.

## Context

Current code reality (as of this workspace):

- `turns.Turn.Data` is `map[turns.TurnDataKey]any` and is commonly used for **serializable hints/config**, e.g. `turns.DataKeyToolConfig`.
- `turns.Turn.Metadata` is `map[turns.TurnMetadataKey]any` and is used for provider/runtime annotations (usage, model, trace id).
- `turns.Block.Metadata` is `map[turns.BlockMetadataKey]any` and is used for block annotations, including provider-native “original content”.
- The **runtime** tool registry is not in `Turn.Data`; it is carried via `context.Context` (`toolcontext.WithRegistry` / `RegistryFrom`).
- `turnsdatalint` enforces typed-key expressions for `Data`/`Metadata` map indexing (prevents raw string drift).

## Quick Reference

### Candidates

Use these “candidates” as debate participants. Each participant should cite **real code** during the debate.

#### Candidate A — The Type-Safety Maximalist

- **Name**: Asha “Strong Types”
- **Core thesis**: Replace “bags of `any`” with an **opaque wrapper** + **typed accessors** so the compiler helps.
- **Preferred end state**:
  - `Turn.Data` is not a public map; it is an API: `Get/Set/Range`.
  - Typed keys `Key[T]` (or equivalent) so `t.Data.Get(turns.KeyToolConfig)` infers `T`.
  - Key identity includes **`{vs,slug,version}`** (enforced by construction).
- **Red lines**:
  - No “just lint it” if we can make invalid states harder to express in code.
  - Avoid scattering type assertions (`.(T)`) at call sites.
- **What they’ll attack**:
  - Any design that still allows accidental `TurnDataKey("oops")` everywhere with no central validation.

#### Candidate B — The Serialization Purist

- **Name**: Noel “Everything Persistable”
- **Core thesis**: `Turn.Data` and (most) metadata should be **structurally serializable**, not “best-effort”.
- **Preferred end state**:
  - `Set` validates at the boundary (e.g. via JSON/YAML marshal) and fails fast with actionable errors.
  - Optionally store canonical serialized form (e.g. `json.RawMessage`) so serializability is guaranteed.
  - Strong guidance on what belongs in `Metadata` vs `Data`.
- **Red lines**:
  - No runtime objects in serialized state (already a direction in the codebase for tools).
- **What they’ll attack**:
  - Opaque wrapper without “serializable-only” enforcement (it just hides the same problems).

#### Candidate C — The Tooling / Lint Enforcer

- **Name**: Mina “Make the Linter a Boundary”
- **Core thesis**: Keep the map shape if it stays simple, but make **linting + conventions** extremely explicit and non-bypassable.
- **Preferred end state**:
  - Keep `map[TurnDataKey]any` (or wrapper) but enforce:
    - “No raw string indexing” (already enforced by `turnsdatalint`).
    - “No ad-hoc keys outside a canonical keys package” (new lint rule).
    - Optional: enforce `{vs,slug,version}` in the string encoding (new lint rule).
- **Red lines**:
  - Don’t rely on “tribal knowledge” for key naming/versioning.
- **What they’ll attack**:
  - Over-engineered wrappers that don’t materially improve correctness vs lint.

#### Candidate D — The Runtime Boundary Advocate

- **Name**: Ravi “Runtime Stays in Context”
- **Core thesis**: Keep runtime state and persisted state separated; avoid contaminating Turn state with execution-only structures.
- **Preferred end state**:
  - `Turn.Data` is intentionally **persisted / serializable**.
  - Runtime attachments live outside (e.g. `context.Context`, or a new explicit runtime carrier if needed).
  - Tool registry remains context-carried; configs/hints remain in `Turn.Data`.
- **Red lines**:
  - Don’t reintroduce runtime registries inside `Turn.Data`.
- **What they’ll attack**:
  - Any design that makes it easy to stash runtime interfaces again “because it’s convenient”.

#### Candidate E — The API Minimalist

- **Name**: Sam “Small Surface Area”
- **Core thesis**: The current model is readable and flexible; improve it with **small helpers** instead of a big redesign.
- **Preferred end state**:
  - Add small helper functions and patterns, but keep the data model obvious.
  - If generics are used, they must be ergonomic (no noisy `Get[T]` everywhere).
- **Red lines**:
  - Don’t create a framework that every caller must learn before they can attach a single hint.
- **What they’ll attack**:
  - Key registries, schema systems, or multi-layer wrappers unless they clearly reduce bugs.

#### Candidate F — The Go Specialist (generics + encoding)

- **Name**: Priya “Go Specialist”
- **Core thesis**: Design must respect Go’s real constraints (zero values, `const` limitations, type inference rules, encoding contracts), otherwise it will be painful or leaky.
- **Preferred end state**:
  - A minimal-but-precise API that composes well with Go tooling (`go/analysis`, `encoding.TextMarshaler`, `yaml.v3` behavior).
  - Clear “invalid key” handling (constructor validation, and/or boundary validation at `Get/Set/Range`).
  - Typed access that doesn’t fight inference (strong preference for typed keys `Key[T]` if we want ergonomic reads).
- **Red lines**:
  - Relying on “Go will infer `T` from assignment” (it won’t, in general).
  - Key modeling that assumes struct keys can be `const` (they can’t).
- **What they’ll attack**:
  - APIs that look clean in pseudocode but don’t survive real call sites and Go’s type system rules.

#### Candidate G — The Application Engineer (API consumer)

- **Name**: Jordan “Just Let Me Ship”
- **Core thesis**: Whatever we do must make common usage simpler (less boilerplate, fewer footguns) for engineers writing middleware and app logic.
- **Preferred end state**:
  - Copy/pasteable patterns for “set config on a turn” and “read a hint safely”.
  - Errors that are easy to debug (“expected ToolConfig under key X, got Y”).
  - A way to evolve/rename/version keys without every caller being a schema expert.
- **Red lines**:
  - Making every call site specify generic type arguments (`Get[Foo]`) all over the place.
  - A design where you need to understand 5 new types just to attach one hint.
- **What they’ll attack**:
  - Overly abstract registries/wrappers that don’t translate into day-to-day ergonomics.

#### Candidate H — The Maintainer (reviewer/bug-triage)

- **Name**: Casey “Code Review”
- **Core thesis**: Optimize for long-term correctness via enforceable conventions and reviewability: “what does this key mean?” should be answerable quickly.
- **Preferred end state**:
  - A single obvious place to discover keys, their intended types, and versioning policy.
  - Lint rules that prevent drift (not just “please don’t do X”).
  - A clear line between persisted state (`Turn.Data`) and runtime state (context-carried, etc.).
- **Red lines**:
  - Hidden implicit behavior that only the original author understands.
- **What they’ll attack**:
  - Anything that makes it harder to review changes or to diagnose production/test failures involving turn state.

## Usage Examples

### How to use these candidates in a debate round

- **Step 1**: pick 3–5 candidates from above (keep it small).
- **Step 2**: require each candidate to cite at least 2 code references:
  - current types (`turns.Turn`, `turns.Block`)
  - key definitions (`turns/keys.go`)
  - `turnsdatalint` rules
  - tool registry contract (`toolcontext.WithRegistry`)
- **Step 3**: run the questions in `reference/02-debate-questions-typed-turn-data-metadata.md`.

## Related

- Ticket analysis: `analysis/01-opaque-turn-data-typed-get-t-accessors.md`
- Key code to cite:
  - `geppetto/pkg/turns/types.go`
  - `geppetto/pkg/turns/key_types.go`
  - `geppetto/pkg/turns/keys.go`
  - `geppetto/pkg/turns/serde/serde.go`
  - `geppetto/pkg/analysis/turnsdatalint/analyzer.go`
  - `geppetto/pkg/inference/toolcontext/toolcontext.go`
