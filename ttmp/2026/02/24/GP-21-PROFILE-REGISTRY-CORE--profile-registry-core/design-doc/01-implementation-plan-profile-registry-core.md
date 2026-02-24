---
Title: Implementation Plan - Profile Registry Core
Ticket: GP-21-PROFILE-REGISTRY-CORE
Status: active
Topics:
    - architecture
    - backend
    - geppetto
    - persistence
    - migration
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/types.go
      Note: Source of truth for profile/registry model fields.
    - Path: geppetto/pkg/profiles/slugs.go
      Note: Typed slug parsing, normalization, and serialization.
    - Path: geppetto/pkg/profiles/validation.go
      Note: Model invariant validation and error boundaries.
    - Path: geppetto/pkg/profiles/service.go
      Note: Resolve/update semantics and runtime merge behavior.
    - Path: geppetto/pkg/profiles/memory_store.go
      Note: In-memory reference implementation.
    - Path: geppetto/pkg/profiles/file_store_yaml.go
      Note: YAML persistence implementation.
    - Path: geppetto/pkg/profiles/sqlite_store.go
      Note: SQLite persistence implementation.
ExternalSources: []
Summary: Detailed implementation plan for profile registry core correctness, invariants, and persistence behavior.
LastUpdated: 2026-02-25T00:50:00-05:00
WhatFor: Break down all foundational registry-core work that must land before extension and middleware tracks.
WhenToUse: Use when implementing model, validation, and storage changes in geppetto/pkg/profiles.
---

# Implementation Plan - Profile Registry Core

## Executive Summary

The profile registry core must be hardened first so all downstream tickets can rely on deterministic behavior. This plan focuses on model clarity, strict invariants, reproducible storage semantics, and test coverage around defaults, versioning, and mutation behavior.

## Problem Statement

Current profile registry support exists, but the project is entering a larger cutover where profile registries drive runtime behavior broadly. That requires stronger guarantees than “works in happy path”:

- default profile selection must be deterministic and explicit,
- serialization and round-trip behavior must be stable across YAML and SQLite,
- validation errors must be precise and actionable,
- metadata/version updates must be consistent for all write paths,
- service behavior should be concurrency-safe and predictable.

## Proposed Solution

### 1. Model Contracts

- Keep typed slugs (`RegistrySlug`, `ProfileSlug`, `RuntimeKey`) as mandatory boundary types.
- Ensure `ProfileRegistry` contracts are explicit:
  - if profiles exist, default profile must be set and present,
  - map key slug must equal profile slug field,
  - nil profiles are rejected.

### 2. Validation Hardening

- Normalize all validation failures to `ValidationError`/`PolicyViolationError`/`VersionConflictError` where appropriate.
- Ensure field path strings are stable and test-covered.
- Validate middleware/tool name non-emptiness and trim semantics consistently.

### 3. Persistence Guarantees

- YAML store:
  - preserve deterministic serialization and read-after-write semantics,
  - verify behavior when file absent/corrupt/partial write recovery path.
- SQLite store:
  - ensure payload consistency, slug matching, migration idempotency,
  - test load behavior on malformed rows and schema drift errors.

### 4. Metadata and Versioning

- Ensure all mutating operations increment metadata version and updated timestamp exactly once.
- Standardize actor/source propagation through write options.
- Cover created/updated attribution behavior in tests.

### 5. Resolution Behavior

- Guarantee fallback behavior when requested profile slug is empty:
  - use registry default,
  - fallback to explicit `default` profile only when policy allows and present,
  - otherwise return validation error.
- Preserve deterministic runtime fingerprint generation for stable cache keys.

## Design Decisions

1. **Strong typed slugs remain mandatory** to avoid stringly-typed drift.
2. **Validation should fail early and loudly** with precise field paths.
3. **SQLite stores full registry payload JSON** (no column-per-field expansion in this phase).
4. **Read-modify-write always passes through validation** before persistence.
5. **No implicit mutation behavior** (all writes through explicit service methods).

## Alternatives Considered

### A. Relaxed validation for easier app onboarding

Rejected because weak invariants in core would push complexity to every consumer.

### B. SQL-normalized tables for registries/profiles immediately

Deferred: current payload JSON row model is sufficient and less migration-heavy for this phase.

### C. Keep metadata/version updates store-specific

Rejected: version/provenance semantics should be consistent independent of storage backend.

## Implementation Plan

### Phase A - Model and Validation

1. Audit and tighten `types.go` and `validation.go` contracts.
2. Add tests for empty/default mismatch and map-key mismatch.
3. Add tests for middleware/tool normalization semantics.

### Phase B - Service Semantics

1. Harden `StoreRegistry.ResolveEffectiveProfile` fallback behavior.
2. Ensure deterministic error type mapping for CRUD and resolution calls.
3. Add conflict/policy tests for update and delete paths.

### Phase C - Persistence Robustness

1. Expand YAML store tests for missing/corrupt file and recovery cases.
2. Expand SQLite tests for malformed payload and slug mismatch.
3. Add parity tests across in-memory/YAML/SQLite service behavior.

### Phase D - Metadata/Version Contracts

1. Validate version increments across create/update/delete/default-set paths.
2. Validate actor/source propagation consistency.
3. Add regression tests for timestamp and attribution fields.

### Phase E - Integration Baseline

1. Add integration smoke for list/get/create/update/delete/default flows.
2. Publish a core behavior matrix for downstream tickets.

## Open Questions

1. Do we need optimistic locking on registry-level writes in this phase, or profile-level is sufficient?
2. Should runtime fingerprint include registry metadata tags or only runtime-effective fields?
3. Do we enforce deterministic profile ordering in list endpoints at service layer only, or also in stores?

## References

- `geppetto/pkg/profiles/types.go`
- `geppetto/pkg/profiles/validation.go`
- `geppetto/pkg/profiles/service.go`
- `geppetto/pkg/profiles/file_store_yaml.go`
- `geppetto/pkg/profiles/sqlite_store.go`
