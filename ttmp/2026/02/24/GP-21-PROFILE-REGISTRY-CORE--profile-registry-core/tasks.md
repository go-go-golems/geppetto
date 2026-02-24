# Tasks

## Model and Types

- [x] Audit `geppetto/pkg/profiles/types.go` for core registry/profile field invariants and document each invariant inline in tests.
- [x] Ensure `Profile.Clone()` deep-copies all mutable nested fields and add regression tests for aliasing.
- [x] Ensure `ProfileRegistry.Clone()` deep-copies profiles map and nested payloads; add mutation-isolation tests.
- [x] Verify `RuntimeSpec` clone behavior covers middleware config deep-copy for map/list payload values.
- [x] Add tests proving typed slug marshal/unmarshal behavior across JSON, YAML, and text.

## Validation

- [x] Add explicit tests for empty registry slug rejection with field path assertion.
- [x] Add explicit tests for empty profile slug rejection with field path assertion.
- [x] Add tests for `default_profile_slug` required when profiles map is non-empty.
- [x] Add tests for `default_profile_slug` pointing to missing profile.
- [x] Add tests for nil profile entries in registry map.
- [x] Add tests for map-key slug mismatch (`registry.profiles[key].slug`).
- [x] Add tests for whitespace-only middleware names.
- [x] Add tests for whitespace-only tool names.
- [x] Verify error type consistency for validation failures (`ValidationError`).

## Service Semantics

- [x] Add tests for resolve behavior when requested profile slug is empty and registry default exists.
- [x] Add tests for resolve behavior when requested profile slug is empty and registry default missing.
- [x] Add tests for fallback to `default` slug only when present.
- [x] Add tests for policy violation on update of read-only profile.
- [x] Add tests for policy violation on delete of read-only profile.
- [x] Add tests for version conflict behavior when expected version mismatches.
- [x] Add tests for list ordering determinism of registry summaries.

## Persistence: YAML Store

- [x] Add tests for missing YAML file initialization behavior.
- [x] Add tests for YAML parse failure surfacing and error clarity.
- [x] Add tests for write-then-reload parity with multiple registries.
- [x] Add tests for atomic temp-file rename behavior (no partial final file).
- [x] Add tests for close-state behavior (`ensureOpen`).

## Persistence: SQLite Store

- [ ] Add tests for schema migration idempotency.
- [ ] Add tests for malformed payload JSON row handling.
- [ ] Add tests for row slug != payload slug mismatch error path.
- [ ] Add tests for persistence after profile CRUD operations.
- [ ] Add tests for delete profile causing registry row update persistence.
- [ ] Add tests for `Close()` idempotency and post-close guard behavior.

## Metadata and Versioning

- [ ] Add tests for registry metadata version bump on each mutating operation.
- [ ] Add tests for profile metadata version bump on each mutating operation.
- [ ] Add tests for created/updated actor/source attribution propagation.
- [ ] Add tests for created timestamp immutability and updated timestamp monotonicity.

## Integration Baseline

- [ ] Add backend integration test covering create -> update -> set-default -> delete lifecycle.
- [ ] Add parity test suite that runs same expectations against in-memory, YAML, and SQLite stores.
- [ ] Produce a short behavior matrix note in ticket docs summarizing guaranteed invariants for downstream tickets.
