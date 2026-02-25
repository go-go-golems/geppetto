---
Title: Investigation diary
Ticket: GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER
Status: active
Topics:
    - pinocchio
    - profile-registry
    - stack-profiles
    - migration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/profile_policy.go
      Note: |-
        Resolver now delegates to ResolveEffectiveProfile
        Resolver query/body parsing now only accepts runtime_key/registry_slug/request_overrides
    - Path: ../../../../../../../pinocchio/cmd/web-chat/profile_policy_test.go
      Note: |-
        Updated tests for effective runtime assertions
        Updated precedence/payload tests for hard-cut request names
    - Path: ../../../../../../../pinocchio/cmd/web-chat/runtime_composer.go
      Note: |-
        Composer now prefers resolved profile fingerprint
        Removed local runtime override parser/merge behavior
    - Path: ../../../../../../../pinocchio/cmd/web-chat/runtime_composer_test.go
      Note: Updated to resolved-runtime-only semantics
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx
      Note: Widget payload now emits request_overrides
    - Path: ../../../../../../../pinocchio/pkg/inference/runtime/composer.go
      Note: |-
        ConversationRuntimeRequest extended with ResolvedProfileFingerprint
        Removed RuntimeOverrides from runtime compose request
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
      Note: GetOrCreate no longer passes request overrides into runtime composer
    - Path: ../../../../../../../pinocchio/pkg/webchat/http/api.go
      Note: |-
        Runtime fingerprint propagated from request resolver to submit/stream paths
        Chat request contract hard-cut to runtime_key/registry_slug/request_overrides
    - Path: ../../../../../../../pinocchio/pkg/webchat/stream_hub.go
      Note: Updated GetOrCreate calls after runtime override contract removal
    - Path: ttmp/2026/02/25/GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER--pinocchio-stack-profile-resolver-runtime-composer-cutover/tasks.md
      Note: Phase tracking for request resolver cutover
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-25T15:27:00-05:00
WhatFor: Chronological implementation log for GP-29 Pinocchio stack-profile cutover.
WhenToUse: Use when reviewing exactly what changed, why, and how to validate each completed phase.
---





# Investigation diary

## Goal

Track implementation of GP-29 with command-level evidence, commit hashes, failures, and validation guidance.

## Step 1: Request Resolver Cutover to Geppetto Effective Resolution

This step replaced local profile/runtime override merge logic in `cmd/web-chat/profile_policy.go` with geppetto-owned `ResolveEffectiveProfile` behavior. The intent was to stop duplicating policy/override/canonicalization logic in pinocchio and align request resolution with GP-28 stack-profile semantics.

The change preserved existing selection precedence (path/body/query/cookie/default) but moved final runtime computation into the geppetto profile service. This reduces divergence risk and unblocks follow-up work to remove runtime-composer duplication.

### Prompt Context

**User prompt (verbatim):** "ok continue"

**Assistant interpretation:** continue the phased implementation after GP-28 completion, starting with GP-29 downstream adoption work and committing each meaningful phase.

**Inferred user intent:** drive end-to-end migration with strict phase boundaries, frequent diary updates, and validated commits.

**Commit (code):** `294d6ad` — "web-chat: resolve profiles via geppetto effective resolver"

### Commands run

```bash
rg -n "ResolveEffectiveProfile|mergeRuntimeOverrides|RuntimeOverrides|runtimeFingerprint|registrySlug|runtimeKey" pinocchio/cmd/web-chat -S
sed -n '150,520p' pinocchio/cmd/web-chat/profile_policy.go
sed -n '1,620p' pinocchio/cmd/web-chat/profile_policy_test.go
gofmt -w cmd/web-chat/profile_policy.go cmd/web-chat/profile_policy_test.go
go test ./cmd/web-chat/...
git add cmd/web-chat/profile_policy.go cmd/web-chat/profile_policy_test.go
git commit -m "web-chat: resolve profiles via geppetto effective resolver"
```

### What I did

1. Reworked `ProfileRequestResolver` chat/ws resolution paths to call `r.profileRegistry.ResolveEffectiveProfile(...)`.
2. Kept registry/profile selection precedence, but removed local runtime/defaults merge computation.
3. Added resolver helper logic to:
   - build `ResolveInput`,
   - map resolved profile metadata to `ProfileVersion`,
   - map geppetto validation/policy errors into `RequestResolutionError` with HTTP 400.
4. Removed obsolete local helpers:
   - `runtimeDefaultsFromProfile`,
   - `mergeRuntimeOverrides`.
5. Updated resolver tests to assert resolved runtime behavior instead of legacy merged `plan.Overrides` behavior.
6. Fixed post-refactor lint break (`unused resolveDefaultProfileSlug`) by deleting the dead function.

### Why

1. GP-28 already defines the canonical profile runtime resolution path (stack merge + policy-gated request overrides + key normalization).
2. Pinocchio’s local merge logic duplicated behavior and risked divergence from geppetto contracts.

### What worked

1. `go test ./cmd/web-chat/...` passed after cutover.
2. Full pinocchio pre-commit pipeline passed on commit:
   - `go test ./...`,
   - `go generate ./...`,
   - frontend build,
   - `go build ./...`,
   - `golangci-lint`,
   - `go vet`.

### What didn't work

1. First commit attempt failed lint due dead code:
   - error: `func (*ProfileRequestResolver).resolveDefaultProfileSlug is unused`
   - fix: deleted the unused function and re-ran formatting/tests/commit.

### What I learned

1. Request resolver can fully rely on geppetto effective resolution without changing route-level profile selection precedence.
2. Pre-commit lint in pinocchio quickly catches stale helpers left behind by cutovers.

### What was tricky to build

The key tricky point was preserving external request behavior while changing internal ownership:
1. selection precedence remained app-owned,
2. final runtime computation moved to geppetto,
3. tests had to switch from asserting merged override maps to asserting effective resolved runtime fields.

### What warrants a second pair of eyes

1. Error-message compatibility for clients expecting old wording on override-policy rejections.
2. Whether any downstream caller still depends on `plan.Overrides` carrying merged defaults.

### What should be done in the future

1. Implement GP-29 Phase 2 runtime composer adoption:
   - stop local override parsing where redundant,
   - consume resolver-provided fingerprint and metadata contracts.

### Code review instructions

1. Start in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy_test.go`
2. Validate with:
   - `go test ./cmd/web-chat/...` (from `pinocchio` repo root).

### Technical details

1. Commit hash: `294d6ad`.
2. Ticket files updated this step:
   - `tasks.md`,
   - `changelog.md`,
   - `reference/01-investigation-diary.md`.

## Step 2: Propagate Resolver Fingerprint Through Runtime Composition

This step wired the resolver-generated runtime fingerprint from HTTP request resolution all the way to runtime composition. The goal was to stop recomputing a potentially divergent runtime fingerprint in pinocchio when geppetto already produced a lineage-aware cache key.

The implementation kept fallback behavior for non-resolver call paths by retaining local fingerprint construction if no resolver fingerprint is provided.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** continue with the next GP-29 phase and commit incrementally after each concrete migration slice.

**Inferred user intent:** complete the downstream cutover with phase-level checkpoints, not just one bulk change.

**Commit (code):** `10b7c8f` — "web-chat: propagate resolved runtime fingerprint to composer"

### Commands run

```bash
gofmt -w cmd/web-chat/profile_policy.go cmd/web-chat/runtime_composer.go pkg/inference/runtime/composer.go pkg/webchat/http/api.go pkg/webchat/stream_hub.go pkg/webchat/conversation.go pkg/webchat/conversation_service.go
go test ./cmd/web-chat/... ./pkg/webchat/...
git add cmd/web-chat/profile_policy.go cmd/web-chat/profile_policy_test.go cmd/web-chat/runtime_composer.go cmd/web-chat/runtime_composer_test.go pkg/inference/runtime/composer.go pkg/webchat/conversation.go pkg/webchat/conversation_service.go pkg/webchat/http/api.go pkg/webchat/stream_hub.go
git commit -m "web-chat: propagate resolved runtime fingerprint to composer"
```

### What I did

1. Added runtime fingerprint fields to request/plan contracts:
   - `ResolvedConversationRequest.RuntimeFingerprint`,
   - `ConversationRuntimeRequest.RuntimeFingerprint`,
   - `SubmitPromptInput.RuntimeFingerprint`,
   - `infruntime.ConversationRuntimeRequest.ResolvedProfileFingerprint`.
2. Propagated these fields through:
   - HTTP handlers,
   - stream hub,
   - conversation service,
   - conversation manager runtime compose request.
3. Updated runtime composer to prefer `ResolvedProfileFingerprint` from resolver.
4. Added/updated tests validating:
   - resolver returns hash-shaped runtime fingerprint (`sha256:*`),
   - runtime composer uses provided resolved fingerprint.

### Why

1. GP-28 fingerprint is lineage-aware and includes stacked profile provenance.
2. Recomputing fingerprint downstream from partial inputs can miss lineage change signals and cause stale runtime cache behavior.

### What worked

1. `go test ./cmd/web-chat/... ./pkg/webchat/...` passed.
2. Full pinocchio pre-commit pipeline passed on commit.

### What didn't work

N/A

### What I learned

1. The runtime fingerprint path crosses multiple internal types; without explicit propagation fields the resolver output is dropped before runtime composition.

### What was tricky to build

The main complexity was cross-layer contract propagation:
1. multiple package-level request structs had to stay aligned,
2. existing tests used different entrypoints (HTTP helper tests, composer tests, service tests),
3. fallback behavior had to remain intact for callsites that bypass resolver.

### What warrants a second pair of eyes

1. Any external integration that serializes/deserializes these request structures should confirm the added fields are ignored/handled as expected.

### What should be done in the future

1. Complete remaining Phase 2 item:
   - remove redundant runtime override parser/merge paths in composer once all callsites rely on resolver-applied effective runtime.

### Code review instructions

1. Start in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/api.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/conversation.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/runtime_composer.go`
2. Validate with:
   - `go test ./cmd/web-chat/... ./pkg/webchat/...` (from `pinocchio` repo root).

### Technical details

1. Commit hash: `10b7c8f`.

## Step 3: Remove Runtime Composer Local Override Parsers (Hard-Cut Phase 2B)

This step completed the remaining GP-29 Phase 2 cleanup by removing request-override parser/merge logic from pinocchio runtime composition. After GP-28, geppetto already canonicalizes and policy-gates request overrides and returns an effective runtime spec; leaving parser logic in composer created duplicate behavior and drift risk.

The cut keeps runtime composer focused on:
1. consuming `ResolvedProfileRuntime`,
2. resolving middleware config via schema,
3. composing the engine,
4. using resolver-owned runtime fingerprint when available.

### Prompt Context

**User prompt (verbatim):** "ok continue"

**Assistant interpretation:** proceed with the next open GP-29 implementation task and commit per phase.

**Inferred user intent:** continue strict hard-cut migration work with granular commits and diary updates.

**Commit (code):** `36bedc3` — "web-chat: remove local runtime override parser path"

### Commands run

```bash
rg -n "override|Override|ResolvedProfileFingerprint|RuntimeFingerprint|parse" cmd/web-chat/runtime_composer.go cmd/web-chat/profile_policy.go pkg/inference/runtime/composer.go pkg/webchat -S
go test ./cmd/web-chat/... ./pkg/webchat/...
gofmt -w cmd/web-chat/runtime_composer.go cmd/web-chat/runtime_composer_test.go pkg/inference/runtime/composer.go pkg/webchat/conversation.go pkg/webchat/stream_hub.go
go test ./cmd/web-chat/... ./pkg/webchat/...
git add cmd/web-chat/runtime_composer.go cmd/web-chat/runtime_composer_test.go pkg/inference/runtime/composer.go pkg/webchat/conversation.go pkg/webchat/stream_hub.go
git commit -m "web-chat: remove local runtime override parser path"
```

### What I did

1. Removed runtime composer request-override parsing/merge code from `cmd/web-chat/runtime_composer.go`:
   - dropped validation/parsing for `system_prompt`, `middlewares`, `tools`,
   - removed runtime middleware override merge helpers and request-layer config source path.
2. Simplified middleware input construction to resolved-profile-only inputs (`runtimeMiddlewareInputsFromProfile`).
3. Removed `RuntimeOverrides` from `pkg/inference/runtime/composer.go` request contract.
4. Updated conversation manager/runtime compose boundary to stop forwarding override maps:
   - `pkg/webchat/conversation.go`,
   - `pkg/webchat/stream_hub.go`.
5. Updated runtime composer tests:
   - removed tests asserting request override application in composer,
   - replaced with resolved-runtime/default behavior assertions.

### Why

1. Geppetto `ResolveEffectiveProfile` is now the canonical point for override policy enforcement and effective runtime materialization.
2. Duplicate parser paths in pinocchio risk split behavior and future regressions.
3. Hard-cut migration explicitly favors removing legacy/duplicate behavior over compatibility shims.

### What worked

1. Targeted suite passed after refactor:
   - `go test ./cmd/web-chat/... ./pkg/webchat/...`.
2. Full pre-commit pipeline passed during commit:
   - `go test ./...`,
   - `go generate ./...`,
   - frontend build,
   - `go build ./...`,
   - `golangci-lint`,
   - `go vet`.

### What didn't work

Initial targeted tests failed immediately after code removal because old tests still expected request-override behavior in composer:
1. `TestWebChatRuntimeComposer_OverridesResolvedRuntimeSpec`,
2. `TestWebChatRuntimeComposer_UsesResolverPrecedenceForMiddlewareConfig`.

Fix: rewrote tests for resolved-runtime/default semantics and profile-owned middleware config assertions.

### What I learned

1. The old runtime override parser path had become fully redundant after resolver cutover.
2. The cleanest guarantee against regression is removing the `RuntimeOverrides` field from runtime composer request types, not just ignoring it at runtime.

### What was tricky to build

Ensuring the cleanup remained narrowly scoped:
1. remove override parsing from runtime composition,
2. keep queue/idempotency request plumbing untouched for now,
3. avoid accidental behavior changes outside composition.

### What warrants a second pair of eyes

1. Any integration that still expected runtime composer request-layer middleware overriding should now move fully to resolver-side request overrides.
2. Remaining `Overrides` fields in higher-level webchat request structs are now primarily for non-composer concerns and may be candidates for later cleanup.

### What should be done in the future

1. Complete GP-29 Phase 3:
   - expose `profile.stack.lineage` and `profile.stack.trace` through chat/web APIs.
2. Complete Phase 1 payload hardening:
   - align request payload naming with registry/runtime key contracts for hard-cut API shape.

### Code review instructions

1. Start in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/runtime_composer.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/inference/runtime/composer.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/conversation.go`
2. Validate with:
   - `go test ./cmd/web-chat/... ./pkg/webchat/...` (from `pinocchio` repo root).

### Technical details

1. Commit hash: `36bedc3`.

## Step 4: Hard-Cut Request Payload Naming (`runtime_key` / `registry_slug` / `request_overrides`)

This step completed the remaining Phase 1 payload hardening work by moving request resolution to explicit hard-cut field names aligned with GP-28 terminology, and removing legacy resolver aliases (`profile`, `registry`, `overrides`, `runtime` query alias) from chat/ws request resolution paths.

Because the web-chat frontend still emitted `overrides`, a follow-up commit updated the widget payload key to `request_overrides`.

### Prompt Context

**User prompt (verbatim):** "ok continue"

**Assistant interpretation:** continue with next open GP-29 phase item after runtime-composer cleanup.

**Inferred user intent:** complete phased hard-cut migration with granular commits and detailed diary history.

**Commits (code):**
1. `d1ba9b2` — "web-chat: hard-cut resolver payload to runtime_key"
2. `1ec381a` — "web-chat: send request_overrides in chat widget payload"

### Commands run

```bash
gofmt -w cmd/web-chat/profile_policy.go cmd/web-chat/profile_policy_test.go pkg/webchat/http/api.go
go test ./cmd/web-chat/... ./pkg/webchat/...
git add cmd/web-chat/profile_policy.go cmd/web-chat/profile_policy_test.go pkg/webchat/http/api.go cmd/web-chat/web/src/webchat/ChatWidget.tsx
git commit -m "web-chat: hard-cut request payload to runtime_key"
# commit failed due parallel pre-commit web-check/lint race
git reset HEAD cmd/web-chat/web/src/webchat/ChatWidget.tsx
git commit -m "web-chat: hard-cut resolver payload to runtime_key"
git add cmd/web-chat/web/src/webchat/ChatWidget.tsx
git commit -m "web-chat: send request_overrides in chat widget payload"
```

### What I did

1. Updated `pkg/webchat/http/api.go` `ChatRequestBody` to use:
   - `runtime_key`,
   - `registry_slug`,
   - `request_overrides`.
2. Updated `cmd/web-chat/profile_policy.go` resolver extraction paths:
   - chat body now reads `RuntimeKey`, `RegistrySlug`, `RequestOverrides`,
   - query parsing now reads only `runtime_key` and `registry_slug`,
   - parse error messaging now reports `invalid runtime_key`.
3. Updated resolver tests in `cmd/web-chat/profile_policy_test.go` to reflect new body/query names and precedence behavior.
4. Updated chat widget request payload in `cmd/web-chat/web/src/webchat/ChatWidget.tsx` to send `request_overrides`.

### Why

1. Hard-cut migration explicitly removes legacy naming and compatibility aliases.
2. Consistent naming with GP-28 contracts simplifies downstream migration planning for go-go-os and pinocchio surfaces.

### What worked

1. Targeted suite passed:
   - `go test ./cmd/web-chat/... ./pkg/webchat/...`.
2. Both resulting commits passed pre-commit checks.

### What didn't work

The first attempt to commit Go + web changes together failed due a pre-commit race:
1. `lefthook` pre-commit is configured with `parallel: true`.
2. `web-check` (`npm run check`) and `lint` (`make lintmax`) ran concurrently.
3. `lint` runs `go generate`, which performs frontend install/build and races against `web-check`, causing transient TypeScript lib-not-found errors.

Fix:
1. split commit into Go-only (skips `web-check`),
2. commit web file separately (runs only `web-check` and passes).

### What I learned

1. With current hook configuration, staging both Go and web files can trigger nondeterministic frontend hook failures due parallelized install/check paths.
2. Operationally safer commit sequencing in this repo is:
   - Go changes first,
   - web changes second.

### What was tricky to build

Ensuring hard-cut naming without accidentally changing profile-management API semantics:
1. request resolver path changed for `/chat` and `/ws`,
2. profile CRUD/list endpoints still intentionally use their own `registry` query/body naming.

### What warrants a second pair of eyes

1. External callers still posting `profile`/`registry`/`overrides` to `/chat` will now fail and must be updated.
2. WS consumers that relied on `profile`/`runtime` query aliases must switch to `runtime_key`.

### What should be done in the future

1. Complete GP-29 Phase 3 metadata exposure (`profile.stack.lineage`, `profile.stack.trace` in chat/web responses).
2. Optionally update lefthook pre-commit sequencing to avoid web-check/lint race when mixed Go+web files are staged.

### Code review instructions

1. Start in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/api.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy_test.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
2. Validate with:
   - `go test ./cmd/web-chat/... ./pkg/webchat/...`,
   - `cd cmd/web-chat/web && npm run check`.

### Technical details

1. Commit hashes: `d1ba9b2`, `1ec381a`.

## Related

- `GP-28-STACK-PROFILES` (upstream core contract)
- `GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER` tasks/changelog
