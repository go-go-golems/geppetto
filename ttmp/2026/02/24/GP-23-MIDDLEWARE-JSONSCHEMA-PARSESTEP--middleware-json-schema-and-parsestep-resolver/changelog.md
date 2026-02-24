# Changelog

## 2026-02-24

- Initial workspace created


## 2026-02-24

Populated ticket with JSON-schema-first middleware resolver design and ParseStep provenance task plan.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP--middleware-json-schema-and-parsestep-resolver/design-doc/01-implementation-plan-middleware-json-schema-and-parsestep-resolver.md — Canonical resolver design and hard-cutover policy
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP--middleware-json-schema-and-parsestep-resolver/tasks.md — Granular resolver and integration tasks


## 2026-02-24

Step 1 completed: middleware definition package scaffold and profile middleware instance contract baseline landed.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/definition.go — Added `Definition` interface and `BuildDeps` carrier type
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/registry.go — Added definition registry interface and in-memory implementation with duplicate guards and deterministic listing
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/registry_test.go — Added registration/lookup/sort/duplicate regression coverage
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/types.go — Extended `MiddlewareUse` with `id` and `enabled`, plus clone behavior
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/validation.go — Added middleware instance ID validation rules
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/validation_test.go — Added middleware instance ID validation tests
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/service.go — Parsed override middleware `id`/`enabled` fields and validation pathing
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/service_test.go — Added request-override duplicate ID regression test
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/types_clone_test.go — Added clone isolation assertions for middleware `id`/`enabled`


## 2026-02-24

Step 2 completed: JSON-schema resolver core landed with layered source precedence, default extraction, JSON-pointer write projection, per-write coercion, final validation, and deterministic path ordering.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/source.go — Added source interface, canonical source layers, and precedence ordering
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/resolver.go — Added schema-first resolver core, write projection, coercion/validation, and deterministic output paths
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/resolver_test.go — Added precedence/defaults/coercion/required-field/path-ordering regression tests
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP--middleware-json-schema-and-parsestep-resolver/tasks.md — Marked JSON Schema Resolver Core checklist complete


## 2026-02-24

Step 3 completed: ParseStep provenance landed with per-path trace history, raw/coerced value tracking, helper APIs for latest/history lookups, and regression coverage.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/resolver.go — Added `ParseStep`, `PathTrace`, trace storage, and history/latest helper APIs
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/resolver_test.go — Added provenance ordering and coercion visibility tests
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP--middleware-json-schema-and-parsestep-resolver/tasks.md — Marked ParseStep Provenance checklist complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP--middleware-json-schema-and-parsestep-resolver/reference/01-diary.md — Added Step 3 diary entry and verification details


## 2026-02-24

Step 4 completed: BuildChain integration landed for resolved middleware instances, including disabled-instance skipping, stable order preservation, instance-keyed build diagnostics, and repeated-name/unique-ID regression coverage.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/chain.go — Added `ResolvedInstance`, `BuildChain`, and instance key/enablement helpers
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/inference/middlewarecfg/chain_test.go — Added chain build ordering/skip/error/repeated-name tests
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP--middleware-json-schema-and-parsestep-resolver/tasks.md — Marked Build Chain Integration checklist complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP--middleware-json-schema-and-parsestep-resolver/reference/01-diary.md — Added Step 4 diary entry and review instructions


## 2026-02-24

Step 5 completed: Pinocchio web-chat runtime composer migrated to resolver output + middleware definition registry, legacy map-based middleware factory path removed from composition wiring, and resolver precedence/schema-failure tests added.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/runtime_composer.go — Refactored runtime composition to resolve middleware configs and build chain via `middlewarecfg`
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/middleware_definitions.go — Added app-owned middleware definitions (`agentmode`, `sqlite`) with JSON schemas and dependency-aware builders
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/main.go — Replaced legacy middleware factory map wiring with definition registry and build deps
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy.go — Included middleware `id`/`enabled` fields in default runtime override payload shape
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/runtime_composer_test.go — Added resolver precedence and invalid schema payload runtime tests
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/inference/runtime/engine.go — Added `BuildEngineFromSettingsWithMiddlewares` helper for pre-resolved middleware chains
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP--middleware-json-schema-and-parsestep-resolver/tasks.md — Marked Pinocchio Integration checklist complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP--middleware-json-schema-and-parsestep-resolver/reference/01-diary.md — Added Step 5 implementation diary and verification notes
