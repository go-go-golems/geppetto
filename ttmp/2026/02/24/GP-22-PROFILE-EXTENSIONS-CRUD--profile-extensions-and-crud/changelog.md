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


## 2026-02-24

Step 4 completed: extension persistence round-trip and cross-backend parity coverage landed (commit 09bc4ca).

### Related Files

- pkg/profiles/codec_yaml_test.go — Added YAML extension round-trip and unknown key preservation tests
- pkg/profiles/file_store_yaml_test.go — Added YAML store partial-update regression preserving unknown extensions
- pkg/profiles/integration_store_parity_test.go — Added extension behavior parity test across memory/YAML/SQLite
- pkg/profiles/sqlite_store_test.go — Added SQLite extension round-trip and partial-update preservation tests
- ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Marked persistence checklist complete


## 2026-02-24

Step 5 completed: Pinocchio shared CRUD API contract updated for extensions, deterministic list ordering, and endpoint/status regression coverage (commit 834fa5c).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy_test.go — Added extension-aware CRUD lifecycle and status mapping assertions
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/http/profile_api.go — Added extensions in list/get/create/patch DTOs and deterministic list sorting
- ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Marked CRUD API contract checklist complete


## 2026-02-24

Step 6 completed: shared profile CRUD route reuse validated across Pinocchio and Go-Go-OS with full integration lifecycle and response-shape contract assertions (commits 673b8ad, 903a5fe).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/app_owned_chat_integration_test.go — Mounted shared package handlers in integration server and added CRUD lifecycle contract test
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go — Expanded mounted CRUD integration test to cover create/get/patch/delete/default with contract-key assertions
- ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Marked shared-route reuse checklist complete


## 2026-02-24

Step 7 completed: Go-Go-OS frontend profile client now decodes/guards extension-aware API payloads and hardens selector fallback behavior after CRUD/default changes (commit cb20eff).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/runtime/profileApi.ts — Added runtime decoders for list/document/current-profile payloads with extension map handling and malformed-payload errors
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/runtime/profileApi.test.ts — Added list/create/update/default decoder coverage including indexed-object fallback parsing
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/runtime/profileTypes.ts — Added extension fields on list/document types
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/runtime/useProfiles.ts — Added deterministic selection resolution helper for CRUD/default-switch refreshes
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/runtime/useProfiles.test.ts — Added profile selection fallback regression tests
- ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Marked frontend integration checklist complete


## 2026-02-24

Step 8 completed: ticket documentation expanded with extension naming/versioning conventions, CRUD examples, starter-suggestions extension contract, and extension validation error semantics.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/design-doc/01-implementation-plan-profile-extensions-and-crud.md — Added implementation-facing conventions, payload examples, and error mapping details
- ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Marked documentation checklist complete


## 2026-02-24

Step 9 verification completed: release-readiness checks passed for Go backend packages, Go-Go-OS frontend runtime tests, and manual end-to-end HTTP smoke flow.

### Verification Commands

- `go test ./pkg/profiles/... -count=1` (geppetto) -> pass
- `go test ./cmd/web-chat ./pkg/webchat/... -count=1` (pinocchio) -> pass
- `npm run test -w packages/engine -- src/chat/runtime/profileApi.test.ts src/chat/runtime/useProfiles.test.ts src/chat/runtime/http.test.ts src/chat/state/profileSlice.test.ts src/chat/state/selectors.test.ts` (go-go-os) -> pass
- Manual smoke against running `hypercard-inventory-server`:
  - `GET /api/chat/profiles?registry=default` -> `200`
  - `POST /api/chat/profiles` (create `smokeops`) -> `201`
  - `POST /api/chat/profiles/smokeops/default?registry=default` -> `200`
  - `POST /api/chat/profile` (select `smokeops`) -> `200`
  - `POST /chat` (prompt with selected profile cookie) -> `200`

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md — Marked release-readiness checklist complete

## 2026-02-24

Ticket closed

