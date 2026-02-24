# Tasks

## TODO

- [ ] Add profile/registry extension fields (`extensions`) to `geppetto/pkg/profiles/types.go` with JSON/YAML tags.
- [ ] Extend `Profile.Clone()` and `ProfileRegistry.Clone()` to deep-copy extension payload maps safely.
- [ ] Add extension key type (`ProfileExtensionKey`) with canonical key constructor and parser.
- [ ] Implement generic typed extension key helper (`ExtK[T]`) with `Decode/Get/Set` functions modeled after turns typed keys.
- [ ] Add unit tests for extension key validation, typed decode behavior, and serializability errors.
- [ ] Extend `ValidateProfile` to validate extension keys and payload serializability.
- [ ] Extend `ValidateRegistry` to validate registry-level extensions (if enabled in phase 1).
- [ ] Introduce extension codec registry interfaces (`ExtensionCodec`, `ExtensionRegistry`) in geppetto profiles package.
- [ ] Implement default in-memory codec registry with collision protection on duplicate key registration.
- [ ] Wire codec-based normalization/validation into profile create/update flows.
- [ ] Ensure unknown extension keys are preserved in pass-through mode (default behavior).
- [ ] Add strict validation mode toggle for tests/tooling (unknown key rejection/warnings) if needed.
- [ ] Update YAML codec tests to verify extension round-trip for known and unknown keys.
- [ ] Update SQLite store tests to verify extension round-trip for known and unknown keys.
- [ ] Add regression tests to ensure no data loss on read-modify-write cycles with unknown extension keys.
- [ ] Add extension fields to profile CRUD request/response DTOs and handlers.
- [ ] Add API validation errors with clear field paths for extension key/payload failures.
- [ ] Add OpenAPI/help-page style examples for extension payloads in CRUD endpoints.
- [ ] Add `StoreRegistry` option plumbing to register one or more extension resolvers.
- [ ] Define and implement `ExtensionResolver` lifecycle invocation during `ResolveEffectiveProfile`.
- [ ] Add resolver error contract and tests for deterministic failure behavior.
- [ ] Implement first concrete extension key in app layer (`webchat.starter_suggestions@v1`) as reference.
- [ ] Expose starter suggestions through webchat API payloads where appropriate.
- [ ] Integrate starter suggestion rendering in pinocchio/go-go-os webchat UI once consumer contract is stable.
- [ ] Document namespace conventions for extension keys (`webchat.*`, app-specific prefixes, vendor prefixes).
- [ ] Create migration mapping from existing env/config toggles to extension keys and profile fields.
- [ ] Add pinocchio CLI/tool command for migrating legacy `profiles.yaml` to registry + extension shape.
- [ ] Write migration playbook for third-party consumers (aliases removed, new API usage, data migration steps).
- [ ] Add glazed help pages in geppetto and pinocchio for extensible profile registry concepts and examples.
- [ ] Add end-to-end tests for profile list/select/update flow using extension-backed profile data.
- [ ] Add compatibility tests for pinocchio and go-go-os mounting shared CRUD routes.
- [ ] Add release note entry and upgrade checklist for extension-enabled profile registry rollout.
- [ ] Run full integration smoke (`go test`, profile CRUD manual checks, UI selection sanity) before closing ticket.

## Middleware Config Unification

- [ ] Create `geppetto/pkg/inference/middlewarecfg` package with `Definition`, `Use`, `Registry`, and typed-definition helper.
- [ ] Keep `middleware.Middleware` runtime type unchanged (no wrapper/alias layer).
- [ ] Extend `geppetto/pkg/profiles/types.go` `MiddlewareUse` with optional `id` and `enabled` fields.
- [ ] Add middleware-use validation for duplicate instance IDs and invalid names.
- [ ] Implement registry duplicate-registration guards and thread-safe lookup semantics.
- [ ] Implement schema assembly for active middleware uses (`SchemaForUses`).
- [ ] Implement layered config parsing flow using glazed sources for middleware sections.
- [ ] Implement chain builder (`BuildChain`) that decodes typed config and instantiates middleware per instance.
- [ ] Refactor `pinocchio/cmd/web-chat/runtime_composer.go` to use middlewarecfg instead of ad-hoc override parsing.
- [ ] Refactor `go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go` to consume `ResolvedProfileRuntime.Middlewares`.
- [ ] Remove `Router.RegisterMiddleware`, `Server.RegisterMiddleware`, and `Router.mwFactories` dead state from `pinocchio/pkg/webchat`.
- [ ] Remove legacy `map[string]MiddlewareBuilder` middleware composition path after composer migration.
- [ ] Migrate all in-repo bootstrap call sites to registry-definition based composer wiring in one cutover.
- [ ] Add integration tests that prove identical middleware profile behavior between pinocchio web-chat and go-go-os inventory server.
- [ ] Add documentation/help page section describing profile-scoped middleware defaults and override precedence.

## JSON Schema + Provenance

- [ ] Make JSON Schema the canonical middleware parameter contract (`ConfigJSONSchema()` per middleware definition).
- [ ] Implement schema-based resolver that applies layered sources and coerces/validates values by JSON pointer path.
- [ ] Implement provenance ledger model (`ValueSetStep`) equivalent to Glazed `FieldValue.Log`.
- [ ] Ensure each applied value records source/layer/raw/coerced/metadata in trace output.
- [ ] Add resolver output type containing both final materialized config and per-path trace history.
- [ ] Add debug serialization endpoint/tooling to inspect resolved middleware config provenance.
- [ ] Implement JSON Schema -> Glazed section adapter for CLI/help generation where needed.
- [ ] Add tests for precedence, coercion, required/default enforcement, and provenance ordering.
- [ ] Remove duplicated ad-hoc middleware decode/override parsing once schema resolver is wired.
