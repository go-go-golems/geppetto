# Tasks

## Scope and Decisions

- [ ] Lock ticket scope: profile-level extensions only (no registry-level extensions).
- [ ] Record hard-error policy for unknown middleware names in design doc and ticket index.
- [ ] Record that middleware CRUD remains profile CRUD (no separate middleware resource CRUD).
- [ ] Record trace/provenance visibility policy: debug-only.

## Write-Time Validation

- [ ] Add middleware definition lookup utility for profile write paths.
- [ ] Add profile-write validation hook in `CreateProfile`.
- [ ] Add profile-write validation hook in `UpdateProfile`.
- [ ] Validate each middleware use name against registered definitions at write-time.
- [ ] Validate each middleware config payload against definition JSON Schema at write-time.
- [ ] Normalize/coerce payload before persistence.
- [ ] Return `ErrValidation` with field path when definition is missing.
- [ ] Return `ErrValidation` with field path when schema validation fails.
- [ ] Add tests for create failure on unknown middleware name.
- [ ] Add tests for update failure on unknown middleware name.
- [ ] Add tests for schema coercion/validation parity with runtime resolver.

## Typed-Key Middleware Payload Model

- [ ] Define canonical middleware extension key format (for example `middleware.<name>.config@v1`).
- [ ] Add typed key helpers for middleware config payload access.
- [ ] Define mapping between `runtime.middlewares[]` instances and extension payloads.
- [ ] Implement write-path projection from legacy `runtime.middlewares[].config` into typed-key payloads.
- [ ] Implement read-path for middleware config from typed-key extensions.
- [ ] Keep middleware order/enable semantics in `runtime.middlewares`.
- [ ] Add migration command/update path for existing registries to typed-key config shape.
- [ ] Add tests for mixed legacy+typed-key payload handling (if transitional read path is kept).
- [ ] Add tests for strict hard-cut mode (if legacy fallback disabled).

## Schema Discovery API

- [ ] Add middleware schema list endpoint (`GET /api/chat/schemas/middlewares`).
- [ ] Include middleware name, version metadata, JSON Schema, and display metadata in response.
- [ ] Add extension schema list endpoint (`GET /api/chat/schemas/extensions`).
- [ ] Define codec interface extension for schema exposure where needed.
- [ ] Return deterministic ordering in schema API responses.
- [ ] Add integration tests for middleware schema endpoint contract.
- [ ] Add integration tests for extension schema endpoint contract.
- [ ] Add frontend runtime decoder/types for schema endpoint payloads.

## Runtime Composer Alignment

- [ ] Update Pinocchio runtime composer to consume typed-key middleware config payloads.
- [ ] Update Go-Go-OS runtime composer to consume typed-key middleware config payloads.
- [ ] Ensure compose-time still hard-fails on unknown middleware (defensive check).
- [ ] Ensure write-time and compose-time validator behavior is consistent.
- [ ] Add parity tests across both composers for identical profile payloads.

## Docs and Migration

- [ ] Update GP-20 follow-up cross-reference with resolved decisions.
- [ ] Add help-page section describing middleware schema API endpoints.
- [ ] Update profile migration playbook with typed-key middleware payload migration.
- [ ] Add example profile JSON/YAML snippets for typed-key middleware config.
- [ ] Add troubleshooting section for write-time schema validation failures.

## Verification and Closeout

- [ ] Run targeted `go test` for geppetto profiles + middlewarecfg packages.
- [ ] Run targeted `go test` for pinocchio web-chat profile API/runtime composer packages.
- [ ] Run targeted `go test` for go-go-os inventory runtime composer/profile API packages.
- [ ] Execute manual smoke: create invalid profile -> receive 400; create valid profile -> compose succeeds.
- [ ] Update ticket changelog with verification matrix and follow-up items.
- [ ] Run `docmgr doctor --ticket GP-27-MIDDLEWARE-SCHEMA-WRITE-VALIDATION`.
