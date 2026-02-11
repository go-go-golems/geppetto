# Changelog

## 2025-12-17

- Initial workspace created


## 2025-12-17

Added analysis doc on making Turn.Data opaque with typed Get[T] access; recommends typed-key wrapper for inference and YAML-compatible marshaling.

### Related Files

- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/analysis/01-opaque-turn-data-typed-get-t-accessors.md — Analysis document


## 2025-12-17

Updated ticket index summary/links; related core files for quick navigation.

### Related Files

- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/analysis/01-opaque-turn-data-typed-get-t-accessors.md — Analysis doc
- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/index.md — Ticket index


## 2025-12-17

Updated analysis to require structured Turn.Data keys {vs, slug, version}; proposes comparable TurnDataKeyID with text encoding plus typed Key[T] for inference.

### Related Files

- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/analysis/01-opaque-turn-data-typed-get-t-accessors.md — Expanded analysis (structured keys + typed access)


## 2025-12-17

Updated analysis: require all values stored in Turn/Block Data+Metadata (and Block.Payload) to be serializable; proposes JSON-bytes storage with YAML-friendly rendering and removal/splitting of runtime-only attachments.

### Related Files

- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/analysis/01-opaque-turn-data-typed-get-t-accessors.md — Expanded analysis (serializable values constraint)


## 2025-12-19

Updated analysis doc to match current codebase (tool registry via context; linter semantics; Block.Payload out-of-scope) and added debate candidates/questions reference docs.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/analysis/01-opaque-turn-data-typed-get-t-accessors.md — Aligned analysis with current implementation
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/01-debate-candidates-typed-turn-data-metadata.md — Debate participants
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/02-debate-questions-typed-turn-data-metadata.md — Debate questions


## 2025-12-19

Expanded debate candidate list with normal human personas (Go specialist, API consumer, maintainer/reviewer).

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/01-debate-candidates-typed-turn-data-metadata.md — Added candidates F/G/H


## 2025-12-19

Completed debate round 1 on questions 1-3 (invariants, metadata scope, wrong-shape UX) with 5 participants citing real code patterns.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/03-debate-round-1-invariants-and-boundaries-q1-q3.md — Debate round document


## 2025-12-19

Completed debate round 2 on questions 4-5 (key identity: structured vs encoded string, versioning location) with 5 participants.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/04-debate-round-2-key-identity-and-versioning-q4-q5.md — Debate round document


## 2025-12-19

Completed debate round 3 on question 6 (application-specific keys: where they live and how to prevent drift) with 5 participants citing Moments turnkeys package.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/05-debate-round-3-application-specific-keys-q6.md — Debate round document


## 2025-12-19

Completed debate round 1 on invariants and error UX (Q1-Q3) with 5 candidates, real code evidence, and consensus on (T, bool, error) API with structured errors.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/06-debate-round-1-invariants-and-error-ux-q1-q3.md — Debate round 1


## 2025-12-19

Debate Round 1 (Q1-3): Explored invariants, Data vs Metadata separation, and error UX for type mismatches with 5 participants.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/08-debate-round-1-q1-3-typed-accessors.md — Debate round 1 document


## 2025-12-19

Debate Round 2 (Q4-6): Explored structured vs encoded keys, versioning strategies, and application-specific key organization with 5 participants.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/09-debate-round-2-q4-6-key-identity.md — Debate round 2 document


## 2025-12-19

Debate Round 3 (Q7-9): Explored opaque wrapper vs public map, linting vs type system enforcement, and iteration escape hatches with 5 participants.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/10-debate-round-3-q7-9-api-surface.md — Debate round 3 document


## 2025-12-19

Debate Round 4 (Q10,Q12): Explored structural serializability enforcement (store JSON vs validate on Set) and failure modes (panic vs error, fail-fast vs fail-late) with 5 participants.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/11-debate-round-4-q10-q12-serializability-failures.md — Debate round 4 document


## 2025-12-19

Debate Round 5 (Q13-14): Explored linter evolution strategies (naming conventions, canonical keys, deprecation) and schema registry trade-offs (runtime vs build-time vs none) with 5 participants.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/reference/12-debate-round-5-q13-14-tooling-schema.md — Debate round 5 document


## 2025-12-19

Uploaded all debate materials (candidates, questions, 5 rounds) to reMarkable device at ai/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA/reference/.


## 2025-12-19

Created comprehensive debate synthesis document organizing 30+ ideas, 7 design axes, decision framework, and phased migration recommendation from 5 debate rounds.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md — Synthesis design document


## 2025-12-22

Refined debate synthesis writing style: added reader map + definitions, added narrative context/bridges throughout axes and tensions, clarified decision implications and phased path.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md — Improved narrative/clarity per technical writing guidelines


## 2025-12-22

Edited synthesis doc for clarity/conciseness: added TL;DR + early Decision Framework, added linting summary + API reference + real-world code patterns, de-duplicated later Decision Framework section, and neutralized implementation section language.

### Related Files

- geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/design-doc/01-debate-synthesis-typed-turn-data-metadata-design.md — Restructure and clarity edits


## 2026-01-25

Ticket closed

