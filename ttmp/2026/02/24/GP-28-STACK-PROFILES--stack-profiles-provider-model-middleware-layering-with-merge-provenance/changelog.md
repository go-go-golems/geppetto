# Changelog

## 2026-02-24

- Initial workspace created


## 2026-02-24

Completed in-depth stack-profile research: current-state architecture mapping, gap analysis, deterministic merge/provenance proposal, phased implementation plan, and JS API alignment guidance.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/inference/middlewarecfg/resolver.go — Provenance pattern used as design precedent
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service.go — Baseline resolver behavior analyzed
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/design-doc/01-stack-profiles-architecture-and-merge-provenance-for-provider-model-middleware-layering.md — Comprehensive proposal document
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/reference/01-investigation-diary.md — Detailed command-level diary


## 2026-02-24

Validated ticket with docmgr doctor (clean) and uploaded bundle to reMarkable at /ai/2026/02/24/GP-28-STACK-PROFILES.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/changelog.md — Upload and validation evidence recorded
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Delivery checklist marked complete


## 2026-02-25

Updated GP-28 to a v2 hard-cutover, geppetto-first execution spec based on later cross-repo research and simplification decisions.

### What changed

- Reframed implementation order:
  1. Geppetto core stack profiles first,
  2. Geppetto JS API hard cutover second,
  3. downstream pinocchio and go-go-os adaptation later.
- Reduced v1 complexity:
  - removed overlay abstraction from near-term architecture,
  - removed request-time custom stack injection from v1 scope,
  - limited v1 to same-registry stack references.
- Tightened fingerprint/caching guidance around stack lineage inputs.
- Updated remaining task list to reflect implementation work not yet started.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/design-doc/01-stack-profiles-architecture-and-merge-provenance-for-provider-model-middleware-layering.md — Rewritten v2 design spec
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Updated completion/remaining implementation checklist
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/overlay.go — Marked for removal from near-term implementation scope

## 2026-02-26

Scope correction per updated requirements:

1. multi-registry stack references are required,
2. middlewarecfg trace pattern should be used for field-level provenance,
3. request-time overrides remain supported (policy-gated),
4. overlay remains removed from implementation path.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/design-doc/01-stack-profiles-architecture-and-merge-provenance-for-provider-model-middleware-layering.md — Rewritten v3 design with restored full scope
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Updated implementation checklist for v3 scope
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/inference/middlewarecfg/resolver.go — Trace model source adopted for profile provenance
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/overlay.go — Still marked for removal from stack implementation path

## 2026-02-26

Expanded GP-28 implementation tasks into a detailed granular execution backlog.

### What changed

- Replaced high-level remaining checklist with phase-by-phase implementation tasks.
- Added explicit file-level work items for:
  - domain model,
  - validation,
  - stack resolver,
  - merge engine,
  - policy-gated request overrides,
  - middlewarecfg-style provenance,
  - fingerprint updates,
  - JS hard cutover,
  - overlay removal,
  - verification matrix,
  - downstream ticket creation.
- Added explicit definition-of-done checklist.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Granular implementation backlog

## 2026-02-25

Implemented GP-28 Phase 1 domain model changes for stack profile references in Geppetto core.

### What changed

- Added `ProfileRef` in `pkg/profiles/types.go`.
- Added `Profile.Stack []ProfileRef` with JSON/YAML tags.
- Updated `Profile.Clone()` to deep-copy stack references.
- Extended clone tests to assert stack slice deep-copy behavior.
- Verified with `go test ./pkg/profiles`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/types.go — Added stack reference domain model and clone support
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/types_clone_test.go — Added regression assertions for stack deep-copy safety
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 1 checklist complete
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/reference/01-investigation-diary.md — Recorded implementation diary details for this phase

## 2026-02-25

Implemented GP-28 Phase 2 serialization and persistence coverage for stack references.

### What changed

- Confirmed YAML codec required no structural changes beyond `Profile` schema updates from Phase 1.
- Added YAML encode/decode round-trip tests that assert:
  - same-registry refs,
  - cross-registry refs,
  - multi-layer stack refs.
- Added SQLite registry persistence/reload test that verifies stack refs are preserved across close/reopen.
- Added backend parity test (memory/yaml/sqlite) validating stack refs survive store operations across implementations.
- Verified with `go test ./pkg/profiles`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/codec_yaml_test.go — Added stack YAML round-trip coverage
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/sqlite_store_test.go — Added SQLite round-trip stack persistence coverage
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/integration_store_parity_test.go — Added cross-backend stack parity coverage
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 2 checklist complete

## 2026-02-25

Implemented GP-28 Phase 3 validation for stack references, topology, and depth limits.

### What changed

- Added stack reference validation in `ValidateProfile`:
  - empty `profile_slug` rejected,
  - invalid `profile_slug` / `registry_slug` rejected with field-level errors.
- Added `ValidateProfileRef` helper for per-ref validation.
- Added stack topology validation helper:
  - `ValidateProfileStackTopology`,
  - `StackValidationOptions`,
  - `DefaultProfileStackValidationMaxDepth`.
- Topology validation now covers:
  - missing referenced profile detection,
  - cycle detection with cycle path reporting,
  - max-depth overflow detection with traversal context.
- `ValidateRegistry` now runs topology validation for single-registry checks while allowing unresolved external refs.
- Added focused tests for stack validation failure/success paths.
- Verified with `go test ./pkg/profiles`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/validation.go — Added stack ref + topology validation logic and depth guard
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/validation_test.go — Added stack validation and topology tests
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 3 checklist complete

## 2026-02-25

Implemented GP-28 Phase 4 stack expansion resolver.

### What changed

- Added `ExpandProfileStack` on `StoreRegistry` in new `stack_resolver.go`.
- Resolver behavior now includes:
  - DFS expansion across `(registry, profile)` identities,
  - support for empty `registry_slug` refs as same-registry references,
  - support for explicit cross-registry refs,
  - deterministic first-occurrence deduplication,
  - base->leaf output ordering,
  - explicit cycle chain errors,
  - missing registry/profile validation failures with stack path fields,
  - max-depth guard via `StackResolverOptions.MaxDepth`.
- Added comprehensive resolver tests in `stack_resolver_test.go`.
- Verified with `go test ./pkg/profiles`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/stack_resolver.go — Added stack expansion resolver implementation
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/stack_resolver_test.go — Added resolver behavior test matrix
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 4 checklist complete

## 2026-02-25

Implemented GP-28 Phase 5 stack merge engine.

### What changed

- Added `stack_merge.go` with `MergeProfileStackLayers`.
- Merge behavior now implements:
  - deterministic base->leaf layer merge,
  - runtime patch deep merge using `MergeRuntimeStepSettingsPatches`,
  - last non-empty `system_prompt`,
  - tools replace-on-write semantics,
  - middleware merge by key (`name#id`, fallback `name[index]`),
  - extension deep object merge with scalar/list replacement,
  - restrictive policy merge (allow AND, deny union, allow intersection minus deny, read_only OR).
- Added merge-focused tests in `stack_merge_test.go` covering runtime, middleware, extension, and policy conflict cases.
- Verified with `go test ./pkg/profiles`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/stack_merge.go — Added stack merge engine and merge helpers
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/stack_merge_test.go — Added merge rule regression tests
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 5 checklist complete

## 2026-02-25

Implemented GP-28 Phase 6 request-time override integration on top of stacked runtime.

### What changed

- Updated `ResolveEffectiveProfile` to:
  - expand stack layers (`ExpandProfileStack`),
  - merge layers (`MergeProfileStackLayers`),
  - apply request overrides against merged runtime + merged policy.
- Preserved existing override normalization/policy enforcement path (`resolveRuntimeSpec`, canonical key handling).
- Added service-level tests for stack+override behavior:
  - allowed override over stacked runtime,
  - denied override key over stacked runtime,
  - allow-list restriction over stacked runtime,
  - `AllowOverrides=false` in any stack layer blocking overrides.
- Verified with `go test ./pkg/profiles`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service.go — Integrated stack expansion/merge into resolve flow
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service_test.go — Added stack override policy tests
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 6 checklist complete

## 2026-02-25

Implemented GP-28 Phase 7 stack provenance trace model.

### What changed

- Added `stack_trace.go` with middlewarecfg-style path tracing model:
  - per-path ordered write history,
  - final winner value per path,
  - stable ordered paths,
  - layer metadata per step (`registry`, `profile`, `source`, `version`, `layer_index`).
- Added deterministic debug payload helpers:
  - `BuildDebugPayload`,
  - `MarshalDebugPayload`.
- Added `MergeProfileStackLayersWithTrace` in `stack_merge.go` to return merge output plus trace.
- Added `stack_trace_test.go` to assert:
  - ordered path history behavior,
  - final winner semantics,
  - deterministic payload serialization.
- Verified with `go test ./pkg/profiles`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/stack_trace.go — Added stack trace model + deterministic payload helpers
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/stack_merge.go — Added merge-with-trace helper
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/stack_trace_test.go — Added trace determinism and history tests
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 7 checklist complete

## 2026-02-25

Implemented GP-28 Phase 8 runtime fingerprint lineage updates.

### What changed

- Updated `runtimeFingerprint` payload in `service.go` to include ordered stack lineage entries (`registry_slug`, `profile_slug`, `version`, `source`).
- Fingerprint payload continues to include effective runtime + step settings, now computed from stack-merged + override-applied runtime.
- Added fingerprint sensitivity tests in `service_test.go`:
  - upstream layer version changes fingerprint,
  - stack layer reorder changes fingerprint,
  - request override changes fingerprint,
  - non-stack identical resolves remain stable.
- Verified with `go test ./pkg/profiles`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service.go — Fingerprint payload now includes stack lineage
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service_test.go — Added fingerprint sensitivity regression tests
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 8 checklist complete

## 2026-02-25

Implemented GP-28 Phase 9 `ResolveEffectiveProfile` stack integration finalization.

### What changed

- `ResolveEffectiveProfile` now uses:
  - stack expansion (`ExpandProfileStack`),
  - stack merge + provenance (`MergeProfileStackLayersWithTrace`),
  - stack-aware fingerprint (`runtimeFingerprint` with lineage),
  - stack metadata exposure in response metadata:
    - `profile.stack.lineage`,
    - `profile.stack.trace`.
- Public resolver entrypoint/signature remained unchanged.
- Added metadata-level tests ensuring:
  - lineage ordering matches base->leaf order,
  - trace payload is present and typed.
- Existing non-stack behavior regression coverage remains green.
- Verified with `go test ./pkg/profiles`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service.go — Final resolve integration with stack trace/lineage metadata
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service_test.go — Added stack metadata resolve assertions
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 9 checklist complete

## 2026-02-25

Implemented GP-28 Phase 10 overlay abstraction removal.

### What changed

- Removed deprecated/unused overlay implementation files:
  - `pkg/profiles/overlay.go`,
  - `pkg/profiles/overlay_test.go`.
- Verified no remaining code references depend on overlay abstraction.
- Ran verification:
  - `go test ./pkg/profiles`,
  - `go test ./...`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/overlay.go — Deleted
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/overlay_test.go — Deleted
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 10 checklist complete

## 2026-02-25

Implemented GP-28 Phase 11 Geppetto JS API hard cutover (`commit 0f7a7a9`).

### What changed

- Hard-cut `engines.fromProfile` to call registry-backed resolution (`ResolveEffectiveProfile`) instead of model/env fallback logic.
- Added JS module runtime dependency wiring for `ProfileRegistry` through `Options` and `moduleRuntime`.
- Exposed profile-resolution metadata on engine refs returned by `fromProfile`:
  - `profileRegistry`,
  - `profileSlug`,
  - `runtimeFingerprint`,
  - `resolvedMetadata`.
- Added fallback API-key hydration for resolved step settings from environment in JS module path to preserve runtime usability.
- Updated TypeScript contract:
  - introduced `ProfileEngineOptions` (`registrySlug`, `runtimeKey`, `requestOverrides`),
  - changed `engines.fromProfile` options type,
  - removed legacy `profile` field from generic `EngineOptions`.
- Extended JS module tests to cover:
  - registry-backed profile resolution,
  - cross-registry targeting,
  - required profile registry dependency behavior,
  - live Gemini integration via registry-backed profile lookup.
- Full pre-commit gate passed:
  - `go test ./...`,
  - `go generate ./...`,
  - `go build ./...`,
  - `golangci-lint run`,
  - vet with `geppetto-lint`.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/js/modules/geppetto/api_engines.go — Registry-backed `fromProfile`, metadata exposure, env key hydration helper
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/js/modules/geppetto/module.go — Profile registry option/runtime wiring
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/js/modules/geppetto/api_types.go — `engineRef.Metadata` support
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/js/modules/geppetto/module_test.go — Updated and expanded JS profile resolution tests
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl — JS API type hard-cut update
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 11 checklist complete

## 2026-02-25

Aligned generated/type docs with Phase 11 JS runtime metadata exposure.

### What changed

- Regenerated committed JS type artifact to match updated `fromProfile` options (`commit a05c587`).
- Added `Engine.metadata?: Record<string, any>` to TypeScript contract and generated copy (`commit 1f5cac5`).

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl — Engine metadata typing + profile option shape
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/doc/types/geppetto.d.ts — Generated type artifact sync

## 2026-02-25

Implemented GP-28 Phase 12 documentation update pass (`commit 189eff0`).

### What changed

- Updated profile registry docs for:
  - stack profile domain model (`ProfileRef`, `Profile.Stack`),
  - stack expansion/merge behavior,
  - merged-policy request override semantics,
  - lineage/trace/fingerprint outputs,
  - explicit overlay removal rationale.
- Updated JS API reference and user guide for hard-cut `fromProfile` behavior:
  - registry-backed semantics,
  - required host `ProfileRegistry` dependency,
  - `ProfileEngineOptions`,
  - engine metadata payload expectations,
  - troubleshooting for missing profile registry.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/doc/topics/01-profiles.md — Stack/multi-registry and overlay-removal documentation
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/doc/topics/13-js-api-reference.md — Hard-cut `fromProfile` reference updates
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/doc/topics/14-js-api-user-guide.md — Hard-cut behavior and host wiring guidance
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 12 checklist complete

## 2026-02-25

Completed GP-28 Phase 13 verification matrix (core + downstream focused tests).

### What changed

- Executed required geppetto verification commands:
  - `go test ./pkg/profiles`,
  - `go test ./pkg/inference/middlewarecfg`,
  - `go test ./pkg/js/modules/geppetto`.
- Executed downstream focused verification:
  - `go test ./cmd/web-chat/...` in pinocchio,
  - `go test ./go-inventory-chat/...` in go-go-os.
- All commands passed in current workspace state.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/runtime_composer.go — Downstream runtime composer compile/test target
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go — Downstream runtime composer compile/test target
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md — Marked Phase 13 checklist complete

## 2026-02-25

Completed GP-28 Phase 14 by creating downstream adaptation tickets and linking them back to GP-28.

### What changed

- Created `GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER` with phased migration tasks for:
  - request resolver adoption,
  - runtime composer adoption,
  - metadata/fingerprint propagation,
  - verification/docs.
- Created `GP-30-GO-GO-OS-STACK-PROFILE-CUTOVER` with equivalent downstream migration phases.
- Updated GP-28 index to include downstream follow-up ticket links.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER--pinocchio-stack-profile-resolver-runtime-composer-cutover/index.md — Pinocchio downstream follow-up ticket
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-30-GO-GO-OS-STACK-PROFILE-CUTOVER--go-go-os-stack-profile-resolver-runtime-composer-cutover/index.md — go-go-os downstream follow-up ticket
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/index.md — Added downstream linkage
