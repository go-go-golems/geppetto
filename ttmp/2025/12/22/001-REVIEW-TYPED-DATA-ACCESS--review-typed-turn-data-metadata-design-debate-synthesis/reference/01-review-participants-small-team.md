---
Title: Review participants (small team)
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
      Note: The design synthesis this review pack is meant to review.
ExternalSources: []
Summary: "Small reviewer team (debate-style personas) for reviewing typed Turn.Data/Metadata design."
LastUpdated: 2025-12-22T13:50:40.61012263-05:00
WhatFor: "Assign clear reviewer roles/perspectives, and keep review feedback grounded and non-overlapping."
WhenToUse: "Before running a structured review/debate of the typed Turn.Data/Metadata synthesis; reuse for subsequent RFC reviews."
---

# Review participants (small team)

## Goal

Create a **small, opinionated team of reviewers** (debate-style personas) to review the synthesis doc with **minimal overlap** and **high signal**. Each persona has a bias, a scope, and specific “must-answer” concerns.

## Context

- **Primary input**: the synthesis doc (linked in `RelatedFiles`).
- **Review output**: short notes per persona + concrete “decision + rationale + follow-up tasks”.
- **Non-goal**: re-litigate every debate argument; the synthesis should already carry the bulk of the debate record.

## Quick Reference

### Reviewers (small team; no facilitator)

- **Priya (Go API ergonomics / generics specialist)**
  - **Bias**: simplest API that is still safe; keep call sites clean
  - **Focus**: typed key ergonomics (`Key[T]` inference), method naming (`Set` vs `MustSet` vs `TrySet`), return shapes `(T, bool, error)`
  - **Must validate**:
    - Does the proposed API feel idiomatic Go?
    - Are we forcing type arguments anywhere (and can we avoid that)?
    - Are error semantics unambiguous at call sites?

- **Mina (Tooling / linter maintainer)**
  - **Bias**: enforce conventions with minimal complexity; prevent drift
  - **Focus**: evolution of `turnsdatalint` rules (ban raw string access, ban ad-hoc key construction, deprecation warnings, namespace enforcement)
  - **Must validate**:
    - Can the linter keep enforcement without whole-program analysis?
    - Are there clear “escape hatches” for tests?
    - Does the design rely on linting where the type system can’t help?

- **Noel (Serialization / persistence boundary)**
  - **Bias**: fail-fast + predictable persistence; no late YAML surprises
  - **Focus**: serializability guarantees (`any`+validate vs `json.RawMessage` storage), YAML/JSON round-tripping and failure modes
  - **Must validate**:
    - What invariant do we *actually* want: “serializable to YAML”, “serializable to JSON”, or “best effort”?
    - Where do errors surface (Set-time, Get-time, Marshal-time)?
    - Do we need caching, and if so where?

### Code personification reviewers (grounded in real modules)

- **`turnsdatalint` (the rule-enforcer) — `geppetto/pkg/analysis/turnsdatalint/analyzer.go`**
  - **Bias**: “If it’s important, it should be enforceable.”
  - **Focus**: what must be lint-enforced vs what can be left to convention
  - **Must validate**:
    - Can we prevent drift (raw string keys, ad-hoc key construction, bypasses) with bounded-complexity analysis?
    - What invariants are *not* realistically lintable and therefore must be structural (API boundary)?

- **`turns` data model (the bag we’re changing) — `geppetto/pkg/turns/types.go`**
  - **Bias**: “I need a stable shape and clear semantics.”
  - **Focus**: API boundary decisions (public map vs opaque wrapper), nil-map init, typed access ergonomics
  - **Must validate**:
    - Does the chosen representation keep YAML/persistence predictable?
    - Is the API surface minimal but complete (Get/Set/Delete/Range/Len)?

- **Middlewares (the main consumers) — `moments/backend/pkg/inference/middleware/*.go`, `geppetto/pkg/inference/middleware/*.go`**
  - **Bias**: "We read and write Turn.Data constantly; ergonomics matter."
  - **Focus**: nil-map init patterns, type assertion boilerplate, serialization edge cases (e.g., compression middleware converts to string map)
  - **Must validate**:
    - Does the proposed API reduce boilerplate at middleware call sites?
    - Are there patterns (like compression's string-map conversion) that break with opaque wrappers?
    - Will middleware authors actually use typed helpers, or bypass them?

### Optional “guest reviewer”

- **New Developer (onboarding / discoverability)**
  - **Bias**: “teach me what to do” via API + docs; no tribal knowledge
  - **Focus**: learnability, docs, discoverability, “pit of success”
  - **Must validate**:
    - Can I find key definitions and expected types quickly?
    - Are error messages actionable?
    - Are docs and examples sufficient to avoid misuse?

## Usage Examples

### Example: running a structured review (60–90 minutes)

1. As a group, assign an **owner per section** from `02-review-questions...md` (round-robin).
2. Each owner writes:
   - **Decision**: pick option A/B (big-bang)
   - **Rationale**: 3–5 bullets
   - **Risks**: 1–3 bullets
   - **Follow-ups**: concrete tasks
3. As a group, produce a short outcome note: “what we’re doing next”.

## Related

- Synthesis doc: `geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md`
- Questions pack: `../reference/02-review-questions-for-typed-turn-data-metadata-design.md`
