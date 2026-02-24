# Changelog

## 2026-02-24

- Initial workspace created


## 2026-02-24

Populated ticket with detailed extension+CRUD implementation plan and granular task breakdown.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/design-doc/01-implementation-plan-profile-extensions-and-crud.md — Core architecture and phased rollout plan
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Granular implementation checklist


## 2026-02-24

Step 1 completed: profile-level extensions model and clone-isolation baseline landed (commit 1888ec5).

### Related Files

- pkg/profiles/types.go — Added Profile.Extensions and deep-copy in Profile.Clone
- pkg/profiles/types_clone_test.go — Added extension payload mutation-isolation tests for profile and registry clone paths
- ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/design-doc/01-implementation-plan-profile-extensions-and-crud.md — Recorded decision to defer registry-level extensions in GP-22
- ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Marked Model and Types checklist items complete


## 2026-02-24

Step 2 completed: extension key parser, typed-key helpers, codec registry infrastructure, and service option plumbing landed (commit edfb34d).

### Related Files

- pkg/profiles/extensions.go — Added ExtensionKey parser/types
- pkg/profiles/extensions_test.go — Added parser/constructor/registry/normalization/service-option tests
- pkg/profiles/service.go — Added StoreRegistryOption and WithExtensionCodecRegistry plumbing
- ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Marked extension infrastructure checklist complete


## 2026-02-24

Step 3 completed: extension validation and service create/update normalization wiring landed (commit 440fb4f).

### Related Files

- pkg/profiles/registry.go — Added ProfilePatch.Extensions for update flow
- pkg/profiles/service.go — Wired extension normalization/validation into create and update paths
- pkg/profiles/service_test.go — Added create/update extension normalization and field-path error tests
- pkg/profiles/validation.go — Added extension-key syntax and payload serializability validation
- pkg/profiles/validation_test.go — Added extension validation field-path assertions
- ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Marked validation/service-flow checklist complete

