---
Title: Core Behavior Matrix
Ticket: GP-21-PROFILE-REGISTRY-CORE
Status: active
Topics:
    - architecture
    - backend
    - geppetto
    - persistence
    - migration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/types_clone_test.go
      Note: Clone deep-copy and mutation-isolation invariants.
    - Path: geppetto/pkg/profiles/validation_test.go
      Note: Validation field-path and error-type contract coverage.
    - Path: geppetto/pkg/profiles/service_test.go
      Note: Service fallback, policy, conflict, and ordering semantics.
    - Path: geppetto/pkg/profiles/file_store_yaml_test.go
      Note: YAML persistence lifecycle and robustness checks.
    - Path: geppetto/pkg/profiles/sqlite_store_test.go
      Note: SQLite durability, corruption detection, and close-state checks.
    - Path: geppetto/pkg/profiles/integration_store_parity_test.go
      Note: Lifecycle parity flow across in-memory, YAML, and SQLite backends.
ExternalSources: []
Summary: Short matrix of guaranteed GP-21 profile-registry-core behaviors validated by tests, for downstream ticket design and API consumers.
LastUpdated: 2026-02-24T13:34:48-05:00
WhatFor: Provide a compact contract of core model/service/store guarantees established in GP-21.
WhenToUse: Use when implementing downstream extension/runtime tickets that depend on profile registry behavior guarantees.
---

# Core Behavior Matrix

## Goal

Document the core guarantees established by GP-21 so downstream work can rely on explicit behavior instead of assumptions.

## Context

GP-21 focuses on profile-registry-core correctness: model clone safety, validation contracts, service semantics, and persistence parity across storage backends.

## Quick Reference

| Area | Guaranteed Behavior | Evidence |
| --- | --- | --- |
| Clone semantics | `Profile.Clone()` and `ProfileRegistry.Clone()` deep-copy mutable nested payloads (`map[string]any`, `[]any`, middleware config payloads, slices). | `geppetto/pkg/profiles/types_clone_test.go` |
| Slug codecs | `RegistrySlug`, `ProfileSlug`, and `RuntimeKey` round-trip through JSON, YAML, and text codecs; invalid text/JSON input fails. | `geppetto/pkg/profiles/slugs_test.go` |
| Validation contract | Validation failures are typed (`ErrValidation` / `*ValidationError`) and include stable field paths (e.g. `registry.default_profile_slug`, `runtime.middlewares[0].name`). | `geppetto/pkg/profiles/validation_test.go` |
| Resolve fallback | Empty requested profile resolves to registry default when present; fallback-to-`default` branch and missing-default error path are explicitly tested. | `geppetto/pkg/profiles/service_test.go` |
| Policy enforcement | Read-only profile update/delete operations fail with `ErrPolicyViolation`. | `geppetto/pkg/profiles/service_test.go` |
| Version conflicts | Service/store write operations honor expected-version checks and return `ErrVersionConflict` on mismatch. | `geppetto/pkg/profiles/service_test.go`, `geppetto/pkg/profiles/sqlite_store_test.go` |
| Registry ordering | Registry summaries are deterministic and sorted by slug. | `geppetto/pkg/profiles/service_test.go` |
| YAML store lifecycle | Missing-file initialization, malformed-file failure surfacing, multi-registry write/reload parity, temp-file rename behavior, and closed-store guards are covered. | `geppetto/pkg/profiles/file_store_yaml_test.go` |
| SQLite store lifecycle | Migration idempotency, malformed payload handling, row/payload slug mismatch rejection, CRUD durability across reopen, delete persistence, and close guards are covered. | `geppetto/pkg/profiles/sqlite_store_test.go` |
| Metadata/version invariants | Registry/profile metadata version increments on mutation, actor/source propagation works, `CreatedAtMs` remains immutable, `UpdatedAtMs` is monotonic. | `geppetto/pkg/profiles/memory_store_test.go` |
| Cross-backend parity | Same lifecycle flow (`create -> update -> set-default -> delete`) passes against in-memory, YAML, and SQLite backends with matching outcomes. | `geppetto/pkg/profiles/integration_store_parity_test.go` |

## Usage Examples

Downstream ticket usage:

1. GP-22 (extensions/CRUD): rely on validation field-path stability and clone safety when adding extension payloads.
2. GP-23 (middleware schema resolver): rely on service policy/conflict handling and backend parity guarantees.
3. GP-24 (runtime cutover): rely on deterministic list ordering and profile resolution behavior.

## Related

- `../tasks.md`
- `../design-doc/01-implementation-plan-profile-registry-core.md`
- `./01-diary.md`
