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

## 2026-02-25

Implemented GP-29 Phase 2A runtime fingerprint propagation (`commit 10b7c8f`).

### What changed

- Added resolver-owned runtime fingerprint propagation across web-chat request/runtime layers:
  - `ResolvedConversationRequest.RuntimeFingerprint`,
  - `ConversationRuntimeRequest.RuntimeFingerprint`,
  - `SubmitPromptInput.RuntimeFingerprint`,
  - `infruntime.ConversationRuntimeRequest.ResolvedProfileFingerprint`.
- Updated `ConvManager.GetOrCreate` and stream hub wiring to pass the resolved fingerprint into runtime composition requests.
- Updated `cmd/web-chat/runtime_composer.go` to prefer `req.ResolvedProfileFingerprint` when provided, with fallback to local fingerprint builder for non-resolver call paths.
- Extended tests:
  - resolver tests now assert runtime fingerprint shape (`sha256:*`),
  - runtime composer test asserts preference for provided resolved fingerprint.
- Verification:
  - `go test ./cmd/web-chat/... ./pkg/webchat/...` passed,
  - full pinocchio pre-commit checks passed.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/runtime_composer.go — Runtime composer now prefers resolver fingerprint
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/runtime_composer_test.go — Added resolver fingerprint preference test
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/api.go — HTTP request plan/runtime fingerprint pass-through
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/conversation.go — Conv manager request includes resolved fingerprint

## 2026-02-25

Implemented GP-29 Phase 2B runtime-composer hard cutover cleanup (`commit 36bedc3`).

### What changed

- Removed local runtime-override parser/merge behavior from pinocchio runtime composer:
  - dropped request-layer system prompt/tool/middleware override parsing in `cmd/web-chat/runtime_composer.go`,
  - removed request-layer middleware merge helper types/functions and request source payload layer usage.
- Runtime composition now consumes only resolver-provided `ResolvedProfileRuntime` + middleware schema resolution.
- Removed stale `RuntimeOverrides` field from `pkg/inference/runtime/composer.go` request contract and from conversation manager runtime-compose call path.
- Updated tests to reflect hard-cut semantics:
  - removed request-override parser tests,
  - replaced override-behavior assertions with resolved-runtime/default behavior assertions,
  - kept middleware schema validation coverage and resolver fingerprint precedence coverage.
- Verification:
  - `go test ./cmd/web-chat/... ./pkg/webchat/...` passed,
  - full pinocchio pre-commit checks passed (`go test ./...`, `go generate ./...`, frontend build, `go build ./...`, `golangci-lint`, `go vet`).

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/runtime_composer.go — Removed local runtime override parser/merge path
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/inference/runtime/composer.go — Removed `RuntimeOverrides` from compose request contract
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/conversation.go — `GetOrCreate` no longer forwards request override maps into runtime composition
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/stream_hub.go — Updated `GetOrCreate` callsites for new signature
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/runtime_composer_test.go — Updated tests for hard-cut runtime composer semantics

## 2026-02-25

Implemented GP-29 Phase 1B request payload hard-cut naming (`commits d1ba9b2, 1ec381a`).

### What changed

- Updated chat/ws resolver payload/query contracts to canonical hard-cut names:
  - body fields:
    - `runtime_key`,
    - `registry_slug`,
    - `request_overrides`;
  - query fields:
    - `runtime_key`,
    - `registry_slug`.
- Removed legacy resolver field/query usage for:
  - `profile`,
  - `registry`,
  - `overrides`,
  - `runtime` query alias.
- Updated resolver tests to assert new payload/query field names and precedence semantics under the hard-cut contract.
- Updated web chat widget payload to send `request_overrides` instead of `overrides` so browser chat requests remain aligned with resolver API.
- Verification:
  - `go test ./cmd/web-chat/... ./pkg/webchat/...` passed,
  - full pre-commit checks passed for both commits.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/api.go — Chat request body contract switched to `runtime_key`/`registry_slug`/`request_overrides`
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go — Resolver now consumes only hard-cut request field/query names
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy_test.go — Updated tests for new payload/query key semantics
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx — Chat payload now emits `request_overrides`

## 2026-02-25

Implemented GP-29 Phase 3 metadata exposure in chat/web runtime paths (`commit 5d3e90e`).

### What changed

- Added resolved profile metadata propagation from resolver to runtime/service layers:
  - resolver now emits resolved metadata payload on request plan,
  - service/runtime request contracts now carry resolved profile metadata map.
- Exposed runtime metadata to chat responses:
  - `runtime_fingerprint`,
  - `profile_metadata` (including resolver metadata keys such as `profile.stack.lineage` and `profile.stack.trace`).
- Updated conversation/runtime plumbing to persist and return resolved metadata through `ConversationHandle`.
- Extended tests to assert:
  - resolver plans include stack lineage/trace metadata keys,
  - stream hub returns resolved profile metadata in handles,
  - submit/queue responses include runtime fingerprint + profile metadata.
- Verification:
  - `go test ./cmd/web-chat/... ./pkg/webchat/...` passed,
  - full pinocchio pre-commit checks passed.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go — Resolver request plan now includes resolved profile metadata
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/api.go — HTTP handlers now pass metadata through request/service boundaries
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/conversation_service.go — Chat responses now include runtime fingerprint and resolved profile metadata
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/conversation.go — Conversation state persists resolved profile metadata
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/stream_hub.go — Handles now return resolved profile metadata

## 2026-02-25

Implemented GP-29 Phase 5 documentation rollout and migration notes (`commit e586ac0`, plus GP-28 changelog backlink update).

### What changed

- Updated pinocchio runtime/profile docs for hard-cut stack profile behavior:
  - `cmd/web-chat/README.md`,
  - `pkg/doc/topics/webchat-http-chat-setup.md`.
- Documented operator-facing migration notes:
  - required request payload keys (`runtime_key`, `registry_slug`, `request_overrides`),
  - removed legacy resolver aliases (`profile`, `registry`, `overrides`, `runtime` query alias),
  - runtime metadata fields now exposed in responses (`runtime_fingerprint`, `profile_metadata`).
- Linked GP-29 downstream outcomes back to GP-28 changelog for cross-ticket traceability.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/README.md — Updated request contract and response metadata documentation
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/doc/topics/webchat-http-chat-setup.md — Updated canonical API contract docs
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/changelog.md — Added GP-29 linkage entry

## 2026-02-25

Completed GP-29 Phase 4 manual smoke-check and closed verification backlog.

### What changed

- Ran live `web-chat` smoke checks against a running server with SQLite-backed profile registry:
  - default request resolves `default` profile metadata/fingerprint,
  - explicit `runtime_key=agent` resolves agent metadata/fingerprint,
  - removed legacy query alias `runtime` is ignored (default profile remains selected),
  - invalid runtime key returns `400 invalid runtime_key: ...`,
  - invalid registry slug returns `400 invalid registry: ...`,
  - missing registry returns `404 registry not found`.
- Confirmed response payload contract on `POST /chat` includes:
  - `runtime_fingerprint`,
  - `profile_metadata` with `profile.slug`, `profile.registry`, `profile.stack.lineage`, `profile.stack.trace`.
- Marked Phase 4 task complete in ticket `tasks.md`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go — Request resolver hard-cut behavior verified against live server
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/api.go — Chat request/response field contract validated in smoke run
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER--pinocchio-stack-profile-resolver-runtime-composer-cutover/tasks.md — Phase 4 completion recorded

## 2026-02-25

Added an operator automation script for profile migration + end-to-end smoke checks (`commit 21ce15a`).

### What changed

- Added `pinocchio/scripts/profile_registry_cutover_smoke.sh` to automate:
  - profile YAML backup,
  - legacy-to-canonical registry conversion (`profiles migrate-legacy`),
  - canonical registry import into SQLite DB,
  - web-chat startup on that DB and HTTP smoke checks,
  - pinocchio `--print-parsed-fields` smoke run against migrated profile source.
- Script validates hard-cut behavior with concrete checks:
  - `runtime_key`/`registry_slug` request selection,
  - runtime fingerprint and profile metadata presence,
  - invalid runtime slug validation (`400`),
  - profile middleware load marker (`mode: profile-registry`) in parsed fields.
- Script writes artifacts (DB/YAML/JSON/logs) into a work directory and prints paths at completion.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/scripts/profile_registry_cutover_smoke.sh — End-to-end backup/migrate/import/web-chat/pinocchio smoke automation
