---
Title: Debate questions (typed Turn.Data/Metadata)
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
    - Path: geppetto/pkg/turns/serde/serde.go
      Note: Serde normalization and YAML helpers
    - Path: geppetto/pkg/turns/types.go
      Note: Current Turn/Block shapes and map fields
ExternalSources: []
Summary: Debate question set for deciding how (and whether) to make Turn.Data / Turn.Metadata / Block.Metadata typed, opaque, and serializable-only, grounded in current Geppetto code.
LastUpdated: 2025-12-19T19:24:57.829336319-05:00
WhatFor: A ready-to-run moderator script for a design debate on typed accessors / key identity / serializability in turn data+metadata.
WhenToUse: Use when preparing an RFC or before implementing a large refactor of Turn.Data/Metadata access patterns.
---


# Debate questions (typed Turn.Data/Metadata)

## Goal

Provide a **non-performance**, **non-backwards-compatibility** question set to debate the future design of:

- `Turn.Data`
- `Turn.Metadata`
- `Block.Metadata`

**Explicit scope:** `Block.Payload` is out-of-scope for this ticket/debate.

## Context

Ground rules:

- No “how do we keep old YAML/JSON working?” questions — assume we can break APIs and formats.
- No performance framing (allocations, speed, memory) — focus on correctness, ergonomics, boundaries, and maintainability.
- Candidates must cite real code in at least half their answers (types, serde, lints, tool registry contract).

## Quick Reference

### Moderator script (questions)

Use 8–12 questions in a single round; don’t try to do all of them at once.

#### A. What are we actually trying to guarantee?

1. **What invariants should `Turn.Data` guarantee at the boundary?**
   - Examples: “value is serializable”, “key identity is structured”, “typed read always either returns T or a clear error”.
   - **Ask for evidence**: show today’s call-site patterns and where mistakes happen.

2. **Should `Turn.Metadata` and `Block.Metadata` have the same invariants as `Turn.Data`?**
   - If not, propose a crisp separation rule (“Data is persisted app semantics; Metadata is provider/runtime annotations”).

3. **What is the correct UX for “key exists but the stored value is the wrong shape”?**
   - Return `(T, ok=false)`? Return `(T, ok=true, err=...)`? Panic? Log?
   - Focus on *developer experience* and *debuggability*, not performance.

#### B. Key identity: typed string keys vs `{vs,slug,version}`

4. **Should key identity be a structured type or an encoded string convention?**
   - Structured: comparable struct `{vs,slug,version}` with constructors/validation.
   - Encoded: `TurnDataKey` remains a string type but must follow a canonical encoding (lint-enforced).

5. **Where should versioning live?**
   - In the key identity (`@v2`) vs implicit (rename keys) vs “schema registry”.
   - Discuss how to keep the codebase from silently diverging.

6. **Where do “application-specific keys” live, and how do we prevent drift?**
   - E.g., separate packages (like Moments) defining their own keys.
   - Should we require all keys to be defined in one “keys” package, or allow multiple registries?

#### C. API surface: maps vs opaque wrappers

7. **Do we want `Turn.Data` to remain a map field or become an opaque type?**
   - If opaque: what is the minimal API? (e.g., `Get/Set/Range/Delete/Keys/Len`)
   - If map stays: which helper patterns become “blessed”?

8. **If we introduce typed `Key[T]`, what should we forbid?**
   - Examples: forbidding ad-hoc key construction outside canonical definitions.
   - What role should linting play vs the type system?

9. **What is the “escape hatch” story?**
   - Who is allowed to iterate all entries?
   - Is there a “raw view” function, or do we always force typed access?

#### D. Serializability and runtime boundaries (without performance framing)

10. **Should `Turn.Data` enforce serializable-only values structurally?**
   - Options:
     - Validate on `Set` by marshaling.
     - Store canonical serialized values internally (decode on `Get`).
   - Discuss error surfaces and “what becomes impossible” vs “what becomes discoverable”.

11. **Where do runtime-only attachments belong?**
   - Tools already use `context.Context` registry: is that the universal pattern?
   - Do we ever need an explicit `Turn.Runtime` bag, or is context enough?

12. **What’s the right failure mode when an invariant is violated?**
   - “Fail fast at construction time” vs “fail when reading” vs “fail when serializing”.
   - Consider debugging and log clarity (not speed).

#### E. Tooling and enforcement

13. **How should `turnsdatalint` evolve if we change key modeling?**
   - Keep it simple and only ban raw string indexing?
   - Add new rules: “keys must be canonical”, “key encoding must include vs/slug/version”, “ban ad-hoc constructors”.

14. **Do we want a schema registry for keys?**
   - If yes: where does it live, and how does it interact with generics and typed keys?
   - If no: what replaces the “single source of truth” for key → expected type?

### Evidence pack (files candidates should cite)

- `geppetto/pkg/turns/types.go` (current `Turn` / `Block` map shapes)
- `geppetto/pkg/turns/keys.go` and `geppetto/pkg/turns/key_types.go` (typed string key types + canonical const keys)
- `geppetto/pkg/turns/serde/serde.go` (normalization of nil maps; YAML round-trip)
- `geppetto/pkg/analysis/turnsdatalint/analyzer.go` (typed-key enforcement)
- `geppetto/pkg/inference/toolcontext/toolcontext.go` (runtime tool registry lives in context)

## Usage Examples

### Quick “Round 0” checklist (moderator)

- Confirm scope: **Turn.Data/Turn.Metadata/Block.Metadata** only; not `Block.Payload`.
- Pick the 3–5 candidates to participate.
- Choose 8–10 questions from above.
- Require 2+ code citations per candidate.

## Related

- Candidates: `reference/01-debate-candidates-typed-turn-data-metadata.md`
- Ticket analysis: `analysis/01-opaque-turn-data-typed-get-t-accessors.md`
