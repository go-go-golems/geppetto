# Changelog

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

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go — Request override policy integration target
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/runtime_composer.go — Runtime composition/fingerprint integration target
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/index.md — Upstream contract and implementation source

## 2026-02-25

Implemented GP-29 Phase 1A request resolver cutover (`commit 294d6ad`).

### What changed

- Refactored `cmd/web-chat/profile_policy.go` to resolve chat/ws profile runtime via:
  - `profiles.ResolveEffectiveProfile`,
  - resolver-owned registry/profile selection precedence,
  - geppetto-owned request-override policy validation and canonicalization.
- Removed duplicated local merge path:
  - deleted `runtimeDefaultsFromProfile`,
  - deleted `mergeRuntimeOverrides`,
  - removed profile runtime/manual policy merge behavior from request resolver.
- Updated request resolver tests for new semantics:
  - assert effective runtime system prompt from resolved profile runtime,
  - assert request policy rejection message from geppetto policy enforcement,
  - stop asserting legacy merged `plan.Overrides` payload for defaults.
- Verification:
  - `go test ./cmd/web-chat/...` passed,
  - full pinocchio pre-commit checks passed (`go test ./...`, `go generate ./...`, frontend build, `go build ./...`, `golangci-lint`, `go vet`).

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go — Resolver now delegates to geppetto effective profile resolution
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy_test.go — Updated assertions for resolver cutover semantics
