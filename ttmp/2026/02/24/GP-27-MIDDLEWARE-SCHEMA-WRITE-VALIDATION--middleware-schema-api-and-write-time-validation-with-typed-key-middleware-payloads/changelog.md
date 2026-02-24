# Changelog

## 2026-02-24

- Initial workspace created
- Added ticket scope decisions from GP-20 follow-up:
  - no registry-level extensions for this phase,
  - write-time middleware validation in profile CRUD,
  - hard errors for unknown middleware names,
  - typed-key extension payloads for middleware config with namespacing,
  - debug-only trace/provenance output.
- Added design document with architecture, API proposal, migration plan, and task breakdown.
- Added granular task checklist for validation, schema APIs, composer alignment, docs, and verification.
- Clarified API model: middleware lifecycle remains profile-scoped CRUD; no separate middleware CRUD resource exists today.
