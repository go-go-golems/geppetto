# Changelog

## 2026-02-24

- Initial workspace created
- Added ticket scope decisions from GP-20 follow-up:
  - no registry-level extensions for this phase,
  - write-time middleware validation in profile CRUD,
  - hard errors for unknown middleware names,
  - typed-key extension payloads for middleware config with namespacing,
  - debug-only trace/provenance output.
- Added design document with architecture, API proposal, migration plan, and task breakdown.
- Added granular task checklist for validation, schema APIs, composer alignment, docs, and verification.
- Clarified API model: middleware lifecycle remains profile-scoped CRUD; no separate middleware CRUD resource exists today.

## 2026-02-24

Implemented profile write-time middleware schema validation in shared profile CRUD handlers and added schema discovery endpoints (/api/chat/schemas/middlewares and /api/chat/schemas/extensions). Wired middleware definitions and extension schema catalogs in pinocchio web-chat and go-go-os inventory server.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go — Go-go-os wiring for middleware definitions and extension schemas
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go — Expose middleware definition registry for API wiring
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy.go — Pinocchio wiring for middleware definitions and extension schemas
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy_test.go — API tests for middleware validation and schema endpoints
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/http/profile_api.go — Write-time validation and schema endpoints

## 2026-02-24

Implemented typed-key middleware payload model and runtime hydration path.

### Highlights

- Added Geppetto typed-key middleware extension helpers:
  - canonical key convention `middleware.<name>_config@v1`
  - projection from inline `runtime.middlewares[*].config` to typed-key extension payloads
  - per-instance slot mapping (`id:<id>` / `index:<i>`)
  - read/write helper APIs for middleware config extension payloads
- Updated shared profile CRUD handler logic to:
  - project inline middleware config to typed-key extensions during create/patch writes,
  - validate middleware config from canonical typed-key payloads against definition schema,
  - normalize resolved payloads back into typed-key extension storage.
- Updated request resolver hydration in both web-chat and inventory app to load middleware config from typed-key extensions when runtime inline config is absent.
- Extended extension schema endpoint to include generated middleware typed-key extension schemas.
- Added/updated tests for projection and schema endpoint expectations.

### Verification matrix

- `go test ./pkg/profiles -count=1` (geppetto) ✅
- `go test ./pkg/webchat/http ./cmd/web-chat -count=1` (pinocchio) ✅
- `go test ./go-inventory-chat/internal/pinoweb ./go-inventory-chat/cmd/hypercard-inventory-server -count=1` (go-go-os) ✅

### Related commits

- `geppetto` `b887394` — profiles typed-key middleware extension helpers
- `pinocchio` `0cf75e5` — profile API typed-key projection/validation + resolver hydration + tests
- `go-go-os` `0c2c7ad` — inventory resolver hydration from typed-key middleware extensions

### Related files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/middleware_extensions.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/pkg/profiles/middleware_extensions_test.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/http/profile_api.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy_test.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-27-MIDDLEWARE-SCHEMA-WRITE-VALIDATION--middleware-schema-api-and-write-time-validation-with-typed-key-middleware-payloads/reference/01-diary.md

## 2026-02-24

Recorded hard-cutover policy and implemented middleware schema metadata response fields.

### Highlights

- Ticket scope clarified to hard cutover:
  - no migration command for existing registries,
  - no transitional fallback path.
- Middleware schema endpoint (`GET /api/chat/schemas/middlewares`) now includes:
  - `name`,
  - `version`,
  - `display_name`,
  - `description`,
  - `schema`.
- Added metadata provider methods on app middleware definitions so schema endpoint has stable display metadata for both web-chat and inventory middleware catalogs.
- Updated API contract test expectations for middleware schema metadata fields.

### Verification matrix

- `go test ./pkg/webchat/http ./cmd/web-chat -count=1` (pinocchio) ✅
- `go test ./go-inventory-chat/internal/pinoweb ./go-inventory-chat/cmd/hypercard-inventory-server -count=1` (go-go-os) ✅

### Related files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/http/profile_api.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/middleware_definitions.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/profile_policy_test.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/internal/pinoweb/middleware_definitions.go
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-27-MIDDLEWARE-SCHEMA-WRITE-VALIDATION--middleware-schema-api-and-write-time-validation-with-typed-key-middleware-payloads/tasks.md
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-27-MIDDLEWARE-SCHEMA-WRITE-VALIDATION--middleware-schema-api-and-write-time-validation-with-typed-key-middleware-payloads/design-doc/01-implementation-plan-middleware-schema-api-write-time-validation-and-typed-key-middleware-payloads.md
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-27-MIDDLEWARE-SCHEMA-WRITE-VALIDATION--middleware-schema-api-and-write-time-validation-with-typed-key-middleware-payloads/index.md
