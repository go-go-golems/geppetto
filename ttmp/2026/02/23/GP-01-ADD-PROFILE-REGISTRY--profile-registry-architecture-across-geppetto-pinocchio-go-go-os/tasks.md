# Tasks

## Completed

- [x] GP01-DONE-001 Create ticket workspace and initial planning/diary docs.
- [x] GP01-DONE-002 Map runtime/profile/config flow across geppetto, pinocchio, and go-go-os.
- [x] GP01-DONE-003 Author long-form ProfileRegistry architecture and migration proposal.
- [x] GP01-DONE-004 Upload bundled ticket docs to reMarkable and verify remote artifact.

## Phase 0: Backlog Refinement and Guardrails

- [ ] GP01-000 Define implementation milestone plan with target order (Geppetto core -> Pinocchio -> Go-Go-OS).
- [ ] GP01-001 Create risk checklist for compatibility regressions in profile precedence.
- [ ] GP01-002 Define deprecation policy for `ai-engine` / `ai-api-type` user-facing flags.
- [ ] GP01-003 Add compatibility matrix document (legacy profiles.yaml vs new registry behavior).
- [ ] GP01-004 Define rollout feature flags and fallback toggles for first integration PRs.

## Phase 1: Geppetto Profile Domain Core

- [x] GP01-100 Create `geppetto/pkg/profiles/types.go` for `Profile`, `ProfileRegistry`, `RuntimeSpec`, `PolicySpec`.
- [x] GP01-101 Create `geppetto/pkg/profiles/store.go` interfaces for registry/profile persistence.
- [x] GP01-102 Create `geppetto/pkg/profiles/registry.go` service interfaces (`List/Get/Resolve/Create/Update/Delete`).
- [x] GP01-103 Create `geppetto/pkg/profiles/errors.go` typed errors (`ErrProfileNotFound`, `ErrVersionConflict`, `ErrPolicyViolation`).
- [x] GP01-104 Create `geppetto/pkg/profiles/validation.go` schema and policy validation.
- [x] GP01-105 Create `geppetto/pkg/profiles/overlay.go` multi-store read overlay and single-store write strategy.
- [x] GP01-106 Create `geppetto/pkg/profiles/metadata.go` provenance fields and version handling.

## Phase 1A: Strong Slug Types

- [x] GP01-120 Create `geppetto/pkg/profiles/slugs.go` with custom types: `RegistrySlug`, `ProfileSlug`, `RuntimeKey`.
- [x] GP01-121 Add parse/normalize constructors (`ParseRegistrySlug`, `ParseProfileSlug`) with validation.
- [x] GP01-122 Add `String()` methods and JSON/YAML marshal/unmarshal for slug types.
- [x] GP01-123 Replace raw-string slug fields in new profile domain structs with custom slug types.
- [x] GP01-124 Add adapter helpers for glazed/string APIs (`ToString`, `FromString`) at boundaries.
- [x] GP01-125 Add tests for slug normalization, invalid values, and serialization round-trips.

## Phase 2: File and Memory Stores

- [x] GP01-200 Implement `InMemoryProfileStore` with thread-safe read/write behavior.
- [ ] GP01-201 Implement YAML codec supporting legacy flat profile map format.
- [ ] GP01-202 Implement YAML codec supporting new registry document format.
- [ ] GP01-203 Implement `YAMLFileProfileStore` load/save with atomic write and backup strategy.
- [ ] GP01-204 Add migration helper from legacy profiles.yaml to new normalized registry objects.
- [ ] GP01-205 Add unit tests for backward compatibility with existing `geppetto/misc/profiles.yaml`-style data.

## Phase 3: Resolver and Effective Runtime Construction

- [ ] GP01-300 Implement `ResolveEffectiveProfile` merge precedence rules.
- [ ] GP01-301 Implement policy enforcement for request overrides (allow-list and deny-list handling).
- [ ] GP01-302 Implement resolved runtime fingerprint generation from effective runtime payload.
- [ ] GP01-303 Add metadata emission (`profile.registry`, `profile.slug`, `profile.version`, `profile.source`).
- [ ] GP01-304 Add golden tests for precedence against current behavior from `GatherFlagsFromProfiles`.
- [ ] GP01-305 Add tests for default-profile fallback and unknown-profile error mapping.

## Phase 4: Geppetto CLI Middleware Integration

- [ ] GP01-400 Add registry-backed middleware adapter in `geppetto/pkg/sections` pipeline.
- [ ] GP01-401 Replace direct `sources.GatherFlagsFromProfiles` use with adapter while preserving compatibility.
- [ ] GP01-402 Keep bootstrap parsing for profile selection, but source profile payload from registry service.
- [ ] GP01-403 Add feature flag to toggle old/new middleware path during migration.
- [ ] GP01-404 Add integration tests for env/config/flags/profile ordering.
- [ ] GP01-405 Update command help/deprecation notes for profile-first configuration path.

## Phase 5: Pinocchio Web-Chat Integration

- [ ] GP01-500 Replace local `chatProfileRegistry` structs in `pinocchio/cmd/web-chat/profile_policy.go`.
- [ ] GP01-501 Inject shared profile registry service into request resolver.
- [ ] GP01-502 Extend request parsing to accept explicit `profile` and `registry` fields in chat body/query.
- [ ] GP01-503 Update runtime composer to consume resolved profile runtime instead of local defaults.
- [ ] GP01-504 Keep runtime fingerprint rebuild behavior in `ConvManager` and verify with profile version changes.
- [ ] GP01-505 Add profile CRUD HTTP endpoints (`GET/POST/PATCH/DELETE /api/chat/profiles...`).
- [ ] GP01-506 Keep compatibility endpoint `/api/chat/profile` (cookie-based current selection).
- [ ] GP01-507 Add resolver tests for cookie/query/body/path precedence.
- [ ] GP01-508 Add endpoint tests for validation, policy failures, and version conflict handling.

## Phase 6: Profile Persistence (SQLite First)

- [ ] GP01-600 Create profile registry SQLite schema and migration files.
- [ ] GP01-601 Implement `SQLiteProfileStore` read/list/create/update/delete operations.
- [ ] GP01-602 Implement optimistic concurrency update path (`expected_version`).
- [ ] GP01-603 Implement default-profile mutation and registry-level metadata updates.
- [ ] GP01-604 Add integration tests with sqlite temp DB.
- [ ] GP01-605 Add optional profile-store settings flags/fields in server config.

## Phase 7: Go-Go-OS Client Integration

- [ ] GP01-700 Add profile runtime API client module in `go-go-os/packages/engine/src/chat/runtime`.
- [ ] GP01-701 Extend `submitPrompt` payload to include `profile`/`registry` (optional).
- [ ] GP01-702 Extend websocket connect URL builder to include profile query when configured.
- [ ] GP01-703 Add profile redux slice (available profiles, selected profile, loading/error state).
- [ ] GP01-704 Add profile hooks (`useProfiles`, `useCurrentProfile`, `useSetProfile`).
- [ ] GP01-705 Add profile selector UI integration in `ChatConversationWindow` header actions.
- [ ] GP01-706 Integrate inventory app with explicit profile selection UX.
- [ ] GP01-707 Add tests for runtime/http contract changes and profile reducer behavior.

## Phase 8: End-to-End and Regression Testing

- [ ] GP01-800 Add e2e test: list profiles -> select profile -> send chat -> runtime key reflects selection.
- [ ] GP01-801 Add e2e test: create profile from web API -> appears in list -> usable immediately.
- [ ] GP01-802 Add e2e test: profile update increments version and triggers runtime rebuild on next request.
- [ ] GP01-803 Add e2e test: read-only profile mutation rejected with stable error code/message.
- [ ] GP01-804 Add regression test suite comparing legacy and registry-backed profile resolution outputs.

## Phase 9: Documentation and Rollout

- [ ] GP01-900 Update geppetto/pinocchio user docs for registry-first profile workflows.
- [ ] GP01-901 Add migration guide for legacy `profiles.yaml` users.
- [ ] GP01-902 Add API docs for profile CRUD endpoints and payload schema.
- [ ] GP01-903 Add ops notes for DB-backed profile storage (backup/recovery/permissions).
- [ ] GP01-904 Publish release notes including deprecations and fallback strategy.
