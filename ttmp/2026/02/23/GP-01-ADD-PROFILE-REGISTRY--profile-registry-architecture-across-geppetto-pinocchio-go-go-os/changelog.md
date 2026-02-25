# Changelog

## 2026-02-23

- Initial workspace created


## 2026-02-23

Completed deep cross-repo analysis and authored the long-form ProfileRegistry architecture plan (3613 words, with APIs, diagrams, pseudocode, migration phases, and DB-backed store strategy); updated diary with detailed step-by-step execution notes.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/planning/01-profileregistry-architecture-and-migration-plan.md — Main analysis and implementation proposal deliverable
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/reference/01-diary.md — Frequent detailed diary entries capturing execution and findings


## 2026-02-23

Expanded tasks.md into a granular phased implementation backlog (GP01-xxx IDs) covering Geppetto core, strong slug types, Pinocchio integration, Go-Go-OS integration, storage, tests, and rollout.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/reference/01-diary.md — Diary update recording granular planning step
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Detailed execution backlog with explicit phase/task IDs


## 2026-02-23

Implemented Phase 1 + Phase 1A foundations in geppetto/pkg/profiles with three commits: typed slug value objects, core profile domain/store/registry abstractions, validation/overlay logic, and boundary adapter helpers; marked corresponding tasks complete.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/overlay.go — Multi-store overlay merge and writer delegation
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/registry.go — Registry service interfaces and resolve contract
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/slugs.go — Strong slug types with parse/normalize and JSON/YAML codecs
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/store.go — Profile store interfaces for persistence layers
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/types.go — Core profile and registry domain model with typed slugs
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/reference/01-diary.md — Implementation diary step with commit hashes and validation
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Phase 1 and 1A tasks marked complete


## 2026-02-23

Implemented GP01-200 by adding InMemoryProfileStore with thread-safe read/write operations, expected-version conflict checks, clone-on-read behavior, and dedicated unit tests.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/memory_store.go — In-memory store implementation for early integration and test usage
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/memory_store_test.go — Lifecycle and versioning tests for in-memory store
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/reference/01-diary.md — Implementation diary step for GP01-200
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Marked GP01-200 complete


## 2026-02-23

Implemented GP01-201..205: YAML registry codec with legacy compatibility, conversion helper, canonical encoding, YAML file-backed store with atomic writes, and compatibility tests against geppetto/misc/profiles.yaml.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/codec_yaml.go — YAML decode/encode and legacy migration helper
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/codec_yaml_test.go — Compatibility tests including legacy fixture decode
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/file_store_yaml.go — YAML file-backed ProfileStore implementation
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/file_store_yaml_test.go — Persistence and reload tests for file store
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/reference/01-diary.md — Diary step documenting YAML codec/file-store implementation
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Marked GP01-201 through GP01-205 complete


## 2026-02-23

Implemented GP01-300..305: added StoreRegistry resolver service with precedence merge, request override policy enforcement, runtime fingerprinting, metadata emission, and golden compatibility tests against GatherFlagsFromProfiles (commit 6a0f1be).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/service.go — Phase 3 resolver and policy implementation
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/service_test.go — Golden compatibility and fallback/error mapping tests
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/step_settings_mapper.go — StepSettings patch mapping and merge logic
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Phase 3 task checkboxes updated


## 2026-02-23

Implemented GP01-400..403: added feature-flagged registry-backed profile middleware in geppetto sections, replaced direct GatherFlagsFromProfiles call with compatibility-preserving adapter path, and added focused section tests (commit 1098b9d).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/sections/profile_registry_source.go — Adapter implementation bridging sections to profiles registry
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/sections/profile_registry_source_test.go — Adapter behavior tests
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/sections/sections.go — Registry middleware wiring point with migration toggle
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Phase 4 task checkboxes updated


## 2026-02-23

Implemented GP01-404: added integration precedence test through GetCobraCommandGeppettoMiddlewares under registry-adapter mode, covering config/profile/env/flag override ordering and migration regression risk (commit d8a93de).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/sections/profile_registry_source_test.go — New precedence integration test and helpers
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — GP01-404 checked


## 2026-02-23

Implemented GP01-405: added profile-first help/deprecation notes to ai-engine and ai-api-type command flags, plus tests and docs updates to lock guidance during migration (commit 8acfb80).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/steps/ai/settings/flags/chat.yaml — Help text updates for engine/provider flags
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/steps/ai/settings/settings-chat_test.go — Help guidance test coverage
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — GP01-405 checked


## 2026-02-23

Implemented GP01-500 and GP01-501 in pinocchio web-chat: replaced local chatProfileRegistry structs with shared geppetto profiles.Registry integration and injected registry-backed resolver wiring (commit eb13816).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Phase 5 task updates
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/main.go — Registry bootstrap and injection in command runtime
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy.go — Shared registry resolver and API handler conversion


## 2026-02-23

Implemented GP01-502 in pinocchio web-chat: request resolver now accepts explicit profile/registry in chat body and query while preserving default/cookie compatibility (commit 3a4b585).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Marked GP01-502 complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy.go — Resolver update for body/query profile+registry parsing
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/http/api.go — Request body contract now includes profile and registry selectors


## 2026-02-23

Implemented GP01-503: pinocchio web-chat runtime composer now consumes typed resolved profile runtime passed end-to-end from resolver through conversation pipeline (commit 2ac2dc6).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Marked GP01-503 complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/runtime_composer.go — Runtime composition now starts from resolved profile runtime spec
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/inference/runtime/composer.go — Extended runtime compose request contract with ResolvedRuntime
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/http/api.go — Added ResolvedRuntime field to request plan and forwarding path


## 2026-02-23

Implemented GP01-504: runtime fingerprint/rebuild path now incorporates profile version and is verified by conversation-service coverage for same-version reuse vs new-version rebuild (commit ec779f8).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Marked GP01-504 complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/runtime_composer.go — Fingerprint payload now includes profile version
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/inference/runtime/composer.go — Runtime compose request contract extended with profile version
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/conversation_service_test.go — Added version-driven rebuild verification


## 2026-02-23

Authored detailed intern-facing implementation postmortem (8+ page long-form) covering foundations, architecture, implementation timeline, API/testing guidance, and onboarding runbook; uploaded both postmortem and diary to reMarkable (folder: /ai/2026/02/23/GP-01-ADD-PROFILE-REGISTRY).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/planning/02-implementation-postmortem-and-intern-guide.md — New comprehensive postmortem deliverable
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/reference/01-diary.md — Added documentation and upload execution diary steps


## 2026-02-23

Implemented GP01-505..508 in pinocchio: added /api/chat/profiles CRUD handlers, explicit error/status mapping, compatibility endpoint retained, and resolver/endpoint precedence+conflict tests (commit c25bcd2724f5e51136d968f3dda8c593cf1f252e).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Marked GP01-505/506/507/508 complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy.go — Added CRUD routes and API error mapping over geppetto Registry
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy_test.go — Added lifecycle CRUD


## 2026-02-23

Completed GP01-600..604: added geppetto SQLiteProfileStore with schema/migration, persistent registry payload storage, and integration coverage for version conflicts/default updates (commit 2b3685600e73fe3977cb70cc86fa039d98bcfa90).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/sqlite_store.go — New SQLite-backed ProfileStore with migrate/load/persist and DSN helper
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/sqlite_store_test.go — Integration tests for roundtrip persistence and optimistic-version semantics
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Marked GP01-600/601/602/603/604 complete


## 2026-02-23

Completed GP01-605 in pinocchio web-chat: added glazed settings for profile-registry SQLite DSN/DB and switched profile service bootstrap to SQLite store when configured, with seed-on-first-boot behavior (commit fd9171a6ff6a644ec9fef0ef7e9e74cae3237d26).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Marked GP01-605 complete
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/main.go — Added profile-registry-dsn/profile-registry-db glazed flags and startup wiring
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy.go — Added SQLite profile service initialization helpers and bootstrap registry builder
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy_test.go — Added bootstrap/reopen test for SQLite profile service


## 2026-02-23

Completed Phase 8 (GP01-800..804) with inventory e2e profile tests and geppetto legacy-vs-registry regression matrix; updated geppetto and pinocchio help docs for registry-first workflows and CRUD API reference (commits e768f24, b03096d, c501145, 9ba5c17).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/topics/01-profiles.md — Registry-first conceptual and migration documentation.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/sections/profile_registry_source_test.go — Legacy-vs-registry regression matrix coverage.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go — Phase 8 e2e tests for profile selection
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go — Profile-aware request resolution feeding runtime key/version/runtime.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/doc/topics/webchat-profile-registry.md — Detailed webchat profile registry and CRUD payload documentation.


## 2026-02-23

Added legacy profiles migration operator workflow: new playbook and new 'pinocchio profiles migrate-legacy' command with tests and command wiring; completed GP01-901.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md — Step-by-step migration playbook.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/tasks.md — Marked GP01-901 complete.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy.go — Legacy profile map to registry conversion command implementation.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/pinocchio/cmds/profiles_migrate_legacy_test.go — Migration command regression tests.


## 2026-02-23

Completed remaining documentation tasks: added Phase 0 guardrails (milestones/risk/deprecation/compatibility/rollout), DB-backed profile store ops runbook, and GP-01 release notes; updated geppetto/pinocchio help docs and removed stale feature-flag guidance in postmortem.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md — Adds persistent docs runbook
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/doc/topics/01-profiles.md — Updated profile doc with always-on registry middleware note
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/planning/03-phase-0-rollout-guardrails-and-compatibility-plan.md — Closes GP01-000..004
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/23/GP-01-ADD-PROFILE-REGISTRY--profile-registry-architecture-across-geppetto-pinocchio-go-go-os/playbook/01-db-profile-store-ops-notes-and-gp-01-release-notes.md — Closes GP01-903/904
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/doc/topics/webchat-profile-registry.md — Added SQLite ops and rollout notes


## 2026-02-25

Ticket closed

