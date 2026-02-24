# Tasks

## Model and Types

- [x] Add `extensions` map field to `geppetto/pkg/profiles/types.go` profile model with JSON/YAML tags.
- [x] Decide and document whether registry-level `extensions` is included in this ticket or deferred.
- [x] Add deep-copy behavior for extensions in `Profile.Clone()`.
- [x] Add deep-copy behavior for extensions in `ProfileRegistry.Clone()`.
- [x] Add tests proving mutation isolation of cloned extension payloads.

## Extension Key and Codec Infrastructure

- [x] Add extension key type and parser (`namespace.feature@vN`) with validation tests.
- [x] Add helper constructor for extension keys with panic-free parse API.
- [x] Add generic typed-key helpers for profile extension get/set/decode.
- [x] Add extension codec interface and in-memory registry.
- [x] Add duplicate key registration guard for codec registry.
- [x] Add service option plumbing for codec registry injection.
- [x] Add tests for known-key decode success and decode failure.
- [x] Add tests for unknown-key pass-through behavior.

## Validation and Service Flow

- [x] Extend `ValidateProfile` to validate extension-key syntax.
- [x] Validate extension payload serializability at service boundary.
- [x] Wire codec normalization into create-profile path.
- [x] Wire codec normalization into update-profile path.
- [x] Ensure service errors map to typed validation/policy/conflict errors.
- [x] Add tests for extension validation field paths in returned errors.

## Persistence and Round-Trip

- [ ] Extend YAML codec tests to include extension payload round-trip.
- [ ] Add YAML regression test for preserving unknown extension keys.
- [ ] Extend SQLite store tests to include extension payload round-trip.
- [ ] Add SQLite regression test for preserving unknown extension keys on partial profile updates.
- [ ] Add parity tests proving same extension behavior in memory/YAML/SQLite stores.

## CRUD API Contract

- [ ] Add `extensions` field to profile API list DTO.
- [ ] Add `extensions` field to profile API get DTO.
- [ ] Add `extensions` field to profile API create request DTO.
- [ ] Add `extensions` field to profile API create response DTO.
- [ ] Add `extensions` field to profile API patch request DTO.
- [ ] Add `extensions` field to profile API patch response DTO.
- [ ] Ensure list responses serialize as arrays, not map-index objects.
- [ ] Ensure list ordering is deterministic by slug.
- [ ] Verify status code mapping for invalid extension payload (400/422 as defined).
- [ ] Verify status code mapping for missing registry/profile (404).
- [ ] Verify status code mapping for version conflicts (409).
- [ ] Add endpoint tests for create/update/delete/default flows with extension payloads.

## Shared Route Reuse Across Apps

- [ ] Confirm Pinocchio web-chat mounts shared CRUD handlers only via package API.
- [ ] Mount the same shared CRUD handlers in Go-Go-OS inventory server.
- [ ] Ensure Go-Go-OS mount uses same registry selection semantics as Pinocchio.
- [ ] Add integration test in Go-Go-OS server for CRUD list/get/create/update/delete/default endpoints.
- [ ] Add integration test in Pinocchio web-chat for equivalent endpoint behavior.
- [ ] Add contract comparison test to catch cross-app response-shape drift.

## Frontend Client Integration

- [ ] Update Go-Go-OS `profileApi.ts` types to include `extensions`.
- [ ] Add TS runtime decoder guards for extension payload presence/absence.
- [ ] Add TS tests for list response parsing with multiple profiles.
- [ ] Add TS tests for create/update responses with extension payloads.
- [ ] Validate profile selector behavior after CRUD writes and default switch.

## Documentation

- [ ] Document extension key naming/versioning conventions in ticket docs.
- [ ] Add profile CRUD API examples with extension payloads.
- [ ] Add starter-suggestions extension example payload and expected UI contract.
- [ ] Document error semantics for extension validation failures.

## Release Readiness

- [ ] Run `go test ./...` for geppetto profiles and pinocchio webchat packages.
- [ ] Run Go-Go-OS tests for profile API client and chat runtime slices.
- [ ] Execute manual smoke: create profile -> set default -> select profile -> send message.
- [ ] Record verification output in ticket changelog before close.
