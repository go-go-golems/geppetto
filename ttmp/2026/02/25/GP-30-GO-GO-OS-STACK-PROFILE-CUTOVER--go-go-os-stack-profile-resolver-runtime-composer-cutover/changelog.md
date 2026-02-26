# Changelog

## 2026-02-26

Implemented the first hard-cut go-go-os resolver/composer slice on branch `task/geppetto-profile-registry-js` (go-go-os commit `3b977e7`).

### What changed

- Replaced legacy chat request selectors (`profile`, `registry`, `overrides`) in `StrictRequestResolver` with canonical request handling for:
  - `runtime_key`,
  - `request_overrides`,
  - ignored `registry_slug` selectors at request time.
- Switched resolver runtime computation to `profileRegistry.ResolveEffectiveProfile(...)` and mapped geppetto validation/policy errors to HTTP 400s.
- Removed runtime-composer override rejection guard so override policy remains centralized in profile resolution.
- Updated resolver unit tests and launcher integration tests to:
  - use `runtime_key` request payload/query forms,
  - assert unknown `registry_slug` selectors are ignored,
  - verify policy-gated request override rejection behavior.
- Verified with:
  - `GOWORK=off go test ./internal/pinoweb/... ./cmd/go-go-os-launcher/...`
  - `GOWORK=off go test ./...`

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go — Hard-cut request resolver migration to `runtime_key` + `ResolveEffectiveProfile`.
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/go-go-os/go-inventory-chat/internal/pinoweb/request_resolver_test.go — Resolver contract updates (runtime key precedence, registry selector ignore, policy errors).
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go — Removed local override rejection duplication.
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/go-go-os/go-inventory-chat/cmd/go-go-os-launcher/main_integration_test.go — Integration payload/query updates for runtime key cutover.

## 2026-02-25

- Initial workspace created

## 2026-02-25

Initialized downstream migration backlog from GP-28 stack profile implementation.

### What changed

- Added phased cutover tasks for:
  - request resolver adoption,
  - runtime composer adoption,
  - metadata/fingerprint propagation,
  - verification and docs rollout.
- Linked ticket scope explicitly to GP-28 core contracts.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go — Request override/profile selection integration target
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go — Runtime composition/fingerprint integration target
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/index.md — Upstream contract and implementation source
