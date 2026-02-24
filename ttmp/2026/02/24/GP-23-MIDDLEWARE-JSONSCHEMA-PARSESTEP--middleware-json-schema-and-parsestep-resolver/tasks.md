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

- [x] Refactor `pinocchio/cmd/web-chat/runtime_composer.go` to consume resolver output.
- [x] Remove ad-hoc middleware override parsing in runtime composer.
- [x] Remove legacy map-based builder plumbing where replaced.
- [x] Add runtime composer tests for resolver precedence behavior.
- [x] Add runtime composer tests for invalid middleware schema payload failures.

## Go-Go-OS Integration

- [x] Refactor `go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go` to consume resolver.
- [x] Ensure profile runtime middlewares are no longer ignored in inventory runtime path.
- [x] Add integration tests proving middleware profile defaults apply in inventory server.
- [x] Add parity tests against Pinocchio for identical middleware-use inputs.

## Glazed Adapter and Tooling

- [x] Add JSON-schema-to-Glazed-section adapter for CLI/help integration.
- [x] Add adapter tests for required/default/enum mapping.
- [x] Verify generated sections preserve human-readable docs/help text.
- [x] Document adapter limitations where JSON schema constructs do not map 1:1.

## Debug and Observability

- [x] Add debug payload serializer for resolved middleware config traces.
- [x] Add structured log event when resolver rejects source payload.
- [x] Include path and source details in resolver error messages.
- [x] Add tests for error payload clarity and stability.

## Cleanup and Hard Cutover

- [x] Remove compatibility flags or toggles related to legacy middleware parsing.
- [x] Delete dead helper functions used only by previous ad-hoc parser path.
- [x] Rename/replace `step_settings_mapper` symbol family with resolver-accurate naming.
- [x] Update imports and compile references across geppetto/pinocchio/go-go-os.

## Verification

- [x] Run unit tests for resolver package.
- [x] Run Pinocchio runtime composer tests.
- [x] Run Go-Go-OS integration tests touching middleware runtime paths.
- [x] Execute manual smoke validating layered source precedence.
- [x] Record verification matrix in ticket changelog.
