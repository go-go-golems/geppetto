---
Title: Review questions for typed Turn.Data/Metadata design
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
      Note: The synthesis doc being reviewed; these questions are mapped to its “design axes”.
ExternalSources: []
Summary: "Structured question set for reviewing the typed Turn.Data/Metadata synthesis: decisions and risks (big-bang implementation)."
LastUpdated: 2025-12-22T13:50:40.726019248-05:00
WhatFor: "Drive a consistent, evidence-based review that ends in explicit design decisions for a big-bang implementation."
WhenToUse: "Use during PR/RFC review for Turn.Data/Metadata/Block.Metadata typing/serialization/key-identity changes."
---

# Review questions for typed Turn.Data/Metadata design

## Goal

Provide a **question pack** for reviewing the synthesis doc, forcing explicit decisions on the **key axes** and making sure we surface risks (performance, serialization, tooling).

## Context

- This is a *review* question set, not a new debate prompt.
- Expected output is **decisions** + **follow-up tasks**, not prose.
- Assumption for this ticket: **we are doing it all at once** (no phased rollout questions).

## Quick Reference

### How to use

- Answer the **Must-answer** section first.
- For each question, capture:
  - **Decision**: A/B/Hybrid/Defer
  - **Rationale**: bullets
  - **Risks**: bullets
  - **Follow-ups**: tasks

### Must-answer (top 10)

1. **Key identity**: structured (`TurnDataKeyID{vs, slug, version}`) vs encoded string (`"ns.slug@vN"`)?

2. **Value storage**: store `any` (validate on Set) vs store `json.RawMessage` (decode on Get) vs “strict mode only”?

3. **API surface boundary**: keep public map + helpers vs opaque wrapper?

4. **Default write API**: `Set` returns error vs `Set` panics and `TrySet` returns error vs `MustSet` panics and `Set` returns error?

5. **Default read API**: is `(T, bool, error)` the right standard everywhere? Do we need `MustGet` (tests only)?

6. **Versioning policy**: is an explicit `vN` required in key identity from day one?

7. **Namespace policy**: required prefixes (e.g. `geppetto.*`, `mento.*`), and how do we enforce it?

8. **Linting scope**: what new rules are required in `turnsdatalint` to make the chosen design safe (and what rules are explicitly out-of-scope)?

9. **Serialization contract**: what persistence format is the contract (YAML, JSON, “YAML via JSON”, other)? What are the failure modes and where do we want them?

10. **Success criteria**: how will we know the change helped (metrics/grep checks/test failures reduction)?

---

## Deep-dive questions (by synthesis axis)

### Axis 1 — Key identity (structured vs encoded)

- **A1.1**: If we choose **structured key IDs**, are we OK with canonical keys becoming `var` (not `const`)? What does that do to:
  - reviewability?
  - linter implementation?
  - import cycles / initialization order?

- **A1.2**: If we choose **encoded strings**, do we require `@vN` always, or allow no-suffix as implicit v1? What is the migration plan for existing keys?
- **A1.2**: If we choose **encoded strings**, do we require `@vN` always, or allow no-suffix as implicit v1?

- **A1.3**: Do we need a single global format (all packages), or can apps (Moments) diverge as long as namespaces are enforced?

### Axis 2 — Value storage (`any` vs `json.RawMessage`)

- **A2.1**: Which invariant do we want to guarantee?
  - “can be marshaled to YAML”
  - “can be marshaled to JSON”
  - “can be round-tripped (marshal+unmarshal)”

- **A2.2**: Where should serialization errors surface by default?
  - at write time (`Set`)
  - at read time (`Get`)
  - at persistence boundary (marshal/unmarshal)

- **A2.3**: Performance: is repeated unmarshal on each `Get` acceptable? If not, where does caching live (internal cache vs caller cache)?

### Axis 3 — Typed access (`Key[T]`)

- **A3.1**: Does `Key[T]` cover the main pain points (type assertion + nil boilerplate)? What call sites are still awkward?

- **A3.2**: Is there a standard for “optional values” (e.g. `*T`, `[]T`, `map[...]...`) and error semantics that remains consistent?

- **A3.3**: Do we need “schema-ish” discoverability beyond typed keys (e.g. linter report mode), or is “jump to definition” sufficient?

### Axis 4 — Opaque wrapper vs public map

- **A4.1**: If we keep a public map, what bypasses are we willing to tolerate, and what lint rules must exist to prevent drift?

- **A4.2**: If we go opaque, what is the minimal API surface we commit to (Get/Set/Delete/Range/Len)? Any missing needs (copy, merge, clone)?

- **A4.3**: Where do nil-map initialization responsibilities live (helpers vs wrapper vs serde)?

### Axis 5 — Error handling (panic vs error)

- **A5.1**: Which operations are “bugs” (panic OK) vs “runtime condition” (error required)?

- **A5.2**: Should panic-style APIs be limited to tests, or allowed in production call sites for “should never fail” invariants?

- **A5.3**: What should error messages include (key name, expected type, actual type, path)? Is there a standard error type?

### Axis 6 — Versioning strategy

- **A6.1**: Is versioning always required and always explicit (no implicit v1)?

- **A6.2**: When a value shape changes, is the policy “always bump version” (v1 → v2), or do we allow in-place changes?

### Axis 7 — App-specific keys + collision prevention

- **A7.1**: Do we need a namespace registry? If yes, where does it live (lint config vs code vs convention)?

- **A7.2**: How do tests define keys? Do we allow `test.*` namespaces, or require canonical test keys?

---

## Reviewer scorecard (copy/paste)

```text
Reviewer:
Key identity:
Value storage:
API surface:
Set API:
Get API:
Versioning policy:
Namespace policy:
Lint rules (required):
Success criteria:
Top risks:
Follow-up tasks:
```

## Usage Examples

### Example: short decision note (per reviewer)

```text
Decision: Big-bang, encoded keys, any+validate on Set, public map + helpers.
Rationale:
- Typed keys remove most type assertion boilerplate.
Risks:
- Helpers can be bypassed; relies on linter to prevent drift.
Follow-ups:
- Add linter rule banning TurnDataKey("...") outside canonical key packages.
- Add 2–3 call-site conversions to measure ergonomics.
```

## Related

- Participants: `../reference/01-review-participants-small-team.md`
- Synthesis doc: `geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md`
