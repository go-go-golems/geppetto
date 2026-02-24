# Tasks

## Scope and Decisions

- [x] Lock ticket scope: profile-level extensions only (no registry-level extensions).
- [x] Record hard-error policy for unknown middleware names in design doc and ticket index.
- [x] Record that middleware CRUD remains profile CRUD (no separate middleware resource CRUD).
- [x] Record trace/provenance visibility policy: debug-only.

## Write-Time Validation

- [x] Add middleware definition lookup utility for profile write paths.
- [x] Add profile-write validation hook in `CreateProfile`.
- [x] Add profile-write validation hook in `UpdateProfile`.
- [x] Validate each middleware use name against registered definitions at write-time.
- [x] Validate each middleware config payload against definition JSON Schema at write-time.
- [x] Normalize/coerce payload before persistence.
- [x] Return `ErrValidation` with field path when definition is missing.
- [x] Return `ErrValidation` with field path when schema validation fails.
- [x] Add tests for create failure on unknown middleware name.
- [x] Add tests for update failure on unknown middleware name.
- [x] Add tests for schema coercion/validation parity with runtime resolver.

## Typed-Key Middleware Payload Model

- [x] Define canonical middleware extension key format (for example `middleware.<name>.config@v1`).
- [x] Add typed key helpers for middleware config payload access.
- [x] Define mapping between `runtime.middlewares[]` instances and extension payloads.
- [x] Implement write-path projection from legacy `runtime.middlewares[].config` into typed-key payloads.
- [x] Implement read-path for middleware config from typed-key extensions.
- [x] Keep middleware order/enable semantics in `runtime.middlewares`.
- [x] Hard cutover policy: no migration command and no transitional fallback path.

## Schema Discovery API

- [x] Add middleware schema list endpoint (`GET /api/chat/schemas/middlewares`).
- [x] Include middleware name, version metadata, JSON Schema, and display metadata in response.
- [x] Add extension schema list endpoint (`GET /api/chat/schemas/extensions`).
- [ ] Define codec interface extension for schema exposure where needed.
- [x] Return deterministic ordering in schema API responses.
- [x] Add integration tests for middleware schema endpoint contract.
- [x] Add integration tests for extension schema endpoint contract.
- [x] Add frontend runtime decoder/types for schema endpoint payloads.

## Runtime Composer Alignment

- [x] Update Pinocchio runtime composer to consume typed-key middleware config payloads.
- [x] Update Go-Go-OS runtime composer to consume typed-key middleware config payloads.
- [x] Ensure compose-time still hard-fails on unknown middleware (defensive check).
- [x] Ensure write-time and compose-time validator behavior is consistent.
- [ ] Add parity tests across both composers for identical profile payloads.

## Docs and Migration

- [ ] Update GP-20 follow-up cross-reference with resolved decisions.
- [ ] Add help-page section describing middleware schema API endpoints.
- [ ] Update profile migration playbook with typed-key middleware payload migration.
- [ ] Add example profile JSON/YAML snippets for typed-key middleware config.
- [ ] Add troubleshooting section for write-time schema validation failures.

## Verification and Closeout

- [x] Run targeted `go test` for geppetto profiles + middlewarecfg packages.
- [x] Run targeted `go test` for pinocchio web-chat profile API/runtime composer packages.
- [x] Run targeted `go test` for go-go-os inventory runtime composer/profile API packages.
- [ ] Execute manual smoke: create invalid profile -> receive 400; create valid profile -> compose succeeds.
- [x] Update ticket changelog with verification matrix and follow-up items.
- [x] Run `docmgr doctor --ticket GP-27-MIDDLEWARE-SCHEMA-WRITE-VALIDATION`.
