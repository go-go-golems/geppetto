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

