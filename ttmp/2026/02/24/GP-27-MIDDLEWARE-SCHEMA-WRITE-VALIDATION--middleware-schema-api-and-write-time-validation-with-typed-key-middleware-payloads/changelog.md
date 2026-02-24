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

