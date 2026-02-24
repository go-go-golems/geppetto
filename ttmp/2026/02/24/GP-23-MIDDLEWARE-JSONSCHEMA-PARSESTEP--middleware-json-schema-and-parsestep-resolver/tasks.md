# Tasks

## Package Setup

- [x] Create `geppetto/pkg/inference/middlewarecfg` package.
- [x] Add registry interface for middleware definitions.
- [x] Add in-memory definition registry implementation.
- [x] Add duplicate definition registration guard.
- [x] Add lookup and list APIs with deterministic ordering.

## Definition Contracts

- [x] Add `Definition` interface with `ConfigJSONSchema()` and `Build(...)`.
- [x] Add `BuildDeps` carrier type for app-owned dependencies.
- [x] Add middleware use instance struct (`name`, `id`, `enabled`, `config`).
- [x] Extend profile middleware-use model to include instance `id` and `enabled`.
- [x] Add validation for empty middleware names and duplicate instance IDs.
- [x] Add validation tests for middleware instance key rules.

## JSON Schema Resolver Core

- [x] Add source interface for layered middleware config sources.
- [x] Implement canonical source precedence ordering.
- [x] Implement schema default extraction as first source layer.
- [x] Implement schema projection for source payload writes by JSON pointer.
- [x] Implement per-write coercion/validation against schema fragments.
- [x] Implement final object validation against full schema.
- [x] Add deterministic path ordering in resolver output.

## ParseStep Provenance

- [x] Introduce resolver ParseStep model (`source`, `layer`, `path`, `raw`, `value`, `metadata`).
- [x] Add per-path log append behavior on every applied write.
- [x] Add trace store in resolver result object.
- [x] Add helper for retrieving latest winning value and full history per path.
- [x] Add tests for provenance ordering across multiple source layers.
- [x] Add tests for coercion visibility in provenance (`raw` vs `value`).

## Build Chain Integration

- [x] Implement `BuildChain` that converts resolved instances to middleware chain.
- [x] Skip disabled middleware instances in chain build.
- [x] Ensure stable middleware execution order from resolved list.
- [x] Include instance key in build errors for diagnostics.
- [x] Add tests for repeated middleware name with unique IDs.

## Pinocchio Integration

- [ ] Refactor `pinocchio/cmd/web-chat/runtime_composer.go` to consume resolver output.
- [ ] Remove ad-hoc middleware override parsing in runtime composer.
- [ ] Remove legacy map-based builder plumbing where replaced.
- [ ] Add runtime composer tests for resolver precedence behavior.
- [ ] Add runtime composer tests for invalid middleware schema payload failures.

## Go-Go-OS Integration

- [ ] Refactor `go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go` to consume resolver.
- [ ] Ensure profile runtime middlewares are no longer ignored in inventory runtime path.
- [ ] Add integration tests proving middleware profile defaults apply in inventory server.
- [ ] Add parity tests against Pinocchio for identical middleware-use inputs.

## Glazed Adapter and Tooling

- [ ] Add JSON-schema-to-Glazed-section adapter for CLI/help integration.
- [ ] Add adapter tests for required/default/enum mapping.
- [ ] Verify generated sections preserve human-readable docs/help text.
- [ ] Document adapter limitations where JSON schema constructs do not map 1:1.

## Debug and Observability

- [ ] Add debug payload serializer for resolved middleware config traces.
- [ ] Add structured log event when resolver rejects source payload.
- [ ] Include path and source details in resolver error messages.
- [ ] Add tests for error payload clarity and stability.

## Cleanup and Hard Cutover

- [ ] Remove compatibility flags or toggles related to legacy middleware parsing.
- [ ] Delete dead helper functions used only by previous ad-hoc parser path.
- [ ] Rename/replace `step_settings_mapper` symbol family with resolver-accurate naming.
- [ ] Update imports and compile references across geppetto/pinocchio/go-go-os.

## Verification

- [ ] Run unit tests for resolver package.
- [ ] Run Pinocchio runtime composer tests.
- [ ] Run Go-Go-OS integration tests touching middleware runtime paths.
- [ ] Execute manual smoke validating layered source precedence.
- [ ] Record verification matrix in ticket changelog.
