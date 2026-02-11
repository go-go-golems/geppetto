# Changelog

## 2025-12-22

- Initial workspace created


## 2025-12-22

Adjusted review packet for big-bang implementation: removed migration-planner persona and stripped migration/maintenance sequencing questions; updated links and framing.


## 2025-12-22

Updated reviewer roster: removed facilitator role and dropped Asha/Jordan; added code-persona reviewers (turnsdatalint, turns types, toolcontext).


## 2025-12-22

Completed review round 1: all reviewers answered assigned questions with code research. Consensus: encoded keys, json.RawMessage storage, opaque wrapper, two Set APIs. Open: compression middleware compatibility (Range vs AsMapCopy).


## 2025-12-22

Completed review round 1: animated debate with code research, opening statements, rebuttals, and decision framework. Key tensions: opaque wrapper vs public map, json.RawMessage vs any storage, compression compatibility.


## 2025-12-22

Completed review round 2: meta-review of synthesis document focusing on clarity, conciseness, and structure. Identified redundancies, proposed new document structure, and specific edits to improve readability.


## 2025-12-22

Created new concise synthesis v2 doc (fresh rewrite) instead of iterating on original; incorporates TL;DR/decision framing, concrete middleware examples, and consolidated tooling notes.

### Related Files

- geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/01-debate-synthesis-v2-concise-rewrite.md — New v2 synthesis


## 2025-12-22

Created concise rewrite of synthesis document (~400 lines vs 1001): decisions first, condensed axes, consolidated linting, API reference, real code examples. Removed redundancy and phased migration language.

### Related Files

- geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/02-debate-synthesis-v2-concise-rewrite.md — Fresh concise rewrite


## 2025-12-22

Created final design document with concrete decisions: opaque wrapper, encoded string keys, any storage with validation, two-API error handling, required versioning, strong linting. Includes complete API design, implementation details, migration guide, and implementation plan.

### Related Files

- geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md — Final prescriptive design


## 2025-12-22

Updated final design: use structured TurnDataKeyID internally, encoded string format only for YAML/JSON serialization. Keys are vars (not consts), use MustDataKeyID constructor.


## 2025-12-22

Removed AsStringMap() escape hatch and MustGet() panic variant. Get() always returns error - no panic variants. Compression middleware must refactor to work with typed API.


## 2025-12-22

Removed TrySet variant. Set() always returns error - no panic variants. Consistent error handling across all operations.


## 2025-12-22

Updated key identity to use fixed NamespaceKey and ValueKey types instead of string-based namespaces. Two-level key system: namespace keys (e.g., MentoNamespaceKey) and value keys (e.g., UserDisplayNameValueKey) combined with version.


## 2025-12-22

Updated key identity to use string consts for namespace and value keys (like current design). Keys constructed via K[T](namespaceConst, valueConst, version) - linter enforces const usage.


## 2026-01-25

Ticket closed


## 2026-01-25

Ticket closed

