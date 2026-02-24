---
Title: GP-27 implementation diary (typed-key middleware payload scope)
Ticket: GP-27-MIDDLEWARE-SCHEMA-WRITE-VALIDATION
Status: active
DocType: reference
Owners: []
LastUpdated: 2026-02-24T23:35:00Z
---

# GP-27 implementation diary (typed-key middleware payload scope)

## Context

The ticket already had write-time middleware validation and schema endpoints in place, but the typed-key middleware payload model was still not implemented in code. Runtime composition still depended on `runtime.middlewares[].config` as canonical storage.

## Work log

### 1) Added shared typed-key middleware payload helpers in Geppetto

Files:

- `geppetto/pkg/profiles/middleware_extensions.go`
- `geppetto/pkg/profiles/middleware_extensions_test.go`

What was added:

- canonical middleware config extension key convention:
  - `middleware.<middleware_name>_config@v1`
- typed payload model:
  - `MiddlewareConfigExtensionPayload{Instances map[string]map[string]any}`
- instance-slot mapping:
  - `id:<middleware-id>` when `MiddlewareUse.ID` is set
  - `index:<i>` otherwise
- helper APIs:
  - `MiddlewareConfigExtensionKey`
  - `MiddlewareConfigInstanceSlot`
  - `ProjectRuntimeMiddlewareConfigsToExtensions`
  - `MiddlewareConfigFromExtensions`
  - `SetMiddlewareConfigInExtensions`

Behavior:

- write projection moves inline `runtime.middlewares[*].config` into typed-key extensions and clears inline config.
- read helper resolves config from typed-key extension payload by middleware instance slot.

### 2) Updated shared profile CRUD handlers (Pinocchio webchat HTTP package)

File:

- `pinocchio/pkg/webchat/http/profile_api.go`

What changed:

- Create and Patch flow now run a canonical middleware validation/projection pass:
  - projects inline middleware config into typed-key extensions,
  - validates middleware existence and schema using registered middleware definitions,
  - stores normalized/coerced config back into typed-key extensions.
- Patch flow now merges with current profile state before validation so runtime updates can persist projected extension payloads without dropping existing extension fields.
- `/api/chat/schemas/extensions` now includes:
  - explicitly configured extension schemas,
  - auto-generated middleware typed-key extension schemas derived from middleware definition schemas.

### 3) Wired typed-key read-path into request resolvers used by both apps

Files:

- `pinocchio/cmd/web-chat/profile_policy.go`
- `go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go`

What changed:

- `profileRuntimeSpec(...)` hydration now fills middleware config from typed-key extensions when inline config is absent.
- `runtimeDefaultsFromProfile(...)` in web-chat also hydrates middleware config from typed-key extensions, so profile defaults continue to flow into runtime override assembly.

### 4) Tests updated

File:

- `pinocchio/cmd/web-chat/profile_policy_test.go`

Added/updated assertions:

- CRUD create test now verifies projection of middleware config to extension key:
  - `middleware.agentmode_config@v1`
  - nested `instances["id:primary"]`
- schema endpoint test now verifies extension schema catalog includes:
  - explicit app extension schema
  - middleware typed-key extension schemas (e.g. `middleware.agentmode_config@v1`, `middleware.sqlite_config@v1`)

Geppetto tests added for projection/read-slot behavior and key validation.

## Verification commands and outcomes

Targeted test matrix run:

- `go test ./pkg/profiles -count=1` (geppetto) ✅
- `go test ./pkg/webchat/http ./cmd/web-chat -count=1` (pinocchio) ✅
- `go test ./go-inventory-chat/internal/pinoweb ./go-inventory-chat/cmd/hypercard-inventory-server -count=1` (go-go-os) ✅

During commit hooks:

- Geppetto pre-commit initially failed on `gofmt` for new test file; fixed with:
  - `gofmt -w geppetto/pkg/profiles/middleware_extensions.go geppetto/pkg/profiles/middleware_extensions_test.go`
- Subsequent pre-commit hooks passed.

## Commits produced

- `geppetto`: `b887394` — profiles typed-key middleware extension helpers
- `pinocchio`: `0cf75e5` — profile API projection/validation + resolver hydration + tests
- `go-go-os`: `0c2c7ad` — request resolver hydration from typed-key middleware extension payloads

## Open follow-ups left in GP-27

- migration command for existing registries to typed-key middleware payload shape,
- explicit version/display metadata in middleware schema API contract,
- composer parity tests (cross-app same payload behavior),
- docs/migration playbook completion and `docmgr doctor` closeout.
