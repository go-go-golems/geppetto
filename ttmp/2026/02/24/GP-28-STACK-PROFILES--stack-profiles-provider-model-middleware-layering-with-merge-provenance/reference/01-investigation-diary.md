---
Title: Investigation diary
Ticket: GP-28-STACK-PROFILES
Status: active
Topics:
    - profile-registry
    - stack-profiles
    - merge-provenance
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/01-profiles.md
      Note: Hard-cutover and profile model documentation inspected
    - Path: geppetto/pkg/inference/middlewarecfg/resolver.go
      Note: Provenance model reference inspected and logged
    - Path: geppetto/pkg/js/modules/geppetto/api_engines.go
      Note: JS API boundary findings logged for stack compatibility planning
    - Path: geppetto/pkg/profiles/registry.go
      Note: Resolver contract evidence captured in diary
    - Path: geppetto/pkg/profiles/runtime_settings_patch_resolver.go
      Note: Merge helper behavior inspected and logged
    - Path: geppetto/pkg/profiles/service.go
      Note: Primary investigation target for current resolve and merge behavior
    - Path: geppetto/pkg/profiles/types.go
      Note: Verified domain shape and absence of stack fields
    - Path: geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/design-doc/01-stack-profiles-architecture-and-merge-provenance-for-provider-model-middleware-layering.md
      Note: Diary documents the derivation path for this proposal
ExternalSources: []
Summary: Chronological log of ticket setup, repository evidence collection, design decisions, and delivery validation steps for stack profiles research.
LastUpdated: 2026-02-25T14:32:35-05:00
WhatFor: Continuation-safe execution log for stack profile research and documentation delivery.
WhenToUse: Use when reviewing how conclusions were derived and reproducing evidence/validation commands.
---


# Investigation diary

## Goal

Produce an evidence-backed design for stack profiles (provider -> model -> middleware layering), including merge semantics and provenance model, store it in ticket docs, validate with docmgr, and publish to reMarkable.

## Chronological log

## Phase 0 - Ticket scaffold confirmation

Time window: 2026-02-24 23:15 to 23:17 (local)

Commands:

```bash
pwd && ls -la
ls -la geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance
find geppetto/ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance -maxdepth 3 -type f | sort
sed -n '1,220p' .../index.md
sed -n '1,240p' .../design-doc/01-stack-profiles-architecture-and-merge-provenance-for-provider-model-middleware-layering.md
sed -n '1,240p' .../reference/01-investigation-diary.md
sed -n '1,200p' .../tasks.md
sed -n '1,200p' .../changelog.md
```

Findings:

1. Ticket scaffold already existed with `index.md`, design doc stub, diary stub, `tasks.md`, and `changelog.md`.
2. Docs were template placeholders only; no substantive research content yet.

## Phase 1 - Skill and quality checklist alignment

Time window: 23:17 to 23:18

Commands:

```bash
sed -n '1,260p' /home/manuel/.codex/skills/ticket-research-docmgr-remarkable/SKILL.md
sed -n '1,260p' /home/manuel/.codex/skills/ticket-research-docmgr-remarkable/references/writing-style.md
sed -n '1,260p' /home/manuel/.codex/skills/ticket-research-docmgr-remarkable/references/deliverable-checklist.md
```

Findings:

1. Required sequence confirmed: evidence-first investigation, long-form design doc, detailed diary, docmgr relate/changelog/tasks updates, doctor pass, dry-run upload, real upload, remote verification.
2. Writing expectations confirmed: explicit sectioning, file-backed claims, concrete implementation plan.

## Phase 2 - Core profile resolver evidence collection

Time window: 23:18 to 23:24

Commands:

```bash
rg -n "type ResolveInput|ResolveEffectiveProfile|ResolveForRuntime|runtimeFingerprint|profile_overlay|request_overrides" geppetto/pkg/profiles -S
rg -n "type Profile struct|RuntimeConfig|InferenceProfileExtensions|MiddlewareProfileExtension" geppetto/pkg/profiles/types.go geppetto/pkg/profiles/middleware_extensions.go -S
rg -n "stack|inherit|extends|overlay|compose|provenance" geppetto/pkg/profiles geppetto/pkg/doc/topics/01-profiles.md -S
nl -ba geppetto/pkg/profiles/types.go | sed -n '1,260p'
nl -ba geppetto/pkg/profiles/registry.go | sed -n '1,220p'
nl -ba geppetto/pkg/profiles/service.go | sed -n '120,260p'
nl -ba geppetto/pkg/profiles/service.go | sed -n '300,430p'
nl -ba geppetto/pkg/profiles/service.go | sed -n '430,660p'
nl -ba geppetto/pkg/profiles/runtime_settings_patch_resolver.go | sed -n '1,260p'
nl -ba geppetto/pkg/profiles/validation.go | sed -n '1,280p'
nl -ba geppetto/pkg/profiles/errors.go | sed -n '1,220p'
nl -ba geppetto/pkg/profiles/metadata.go | sed -n '1,220p'
```

Findings:

1. Resolver contract currently supports only one profile slug (`ResolveInput.ProfileSlug`).
2. No inheritance/stack field exists in `Profile`.
3. Request-override merge exists with policy checks and key canonicalization.
4. Step-settings patch deep merge helper already exists and is reusable for stacked merge semantics.
5. Runtime fingerprint currently hashes one profile payload only.

Interpretation:

1. We can reuse existing merge primitives and policy logic, but need a separate stack expansion + merge layer.
2. Provenance and fingerprint must be extended to include stack lineage.

## Phase 3 - Middleware provenance precedent investigation

Time window: 23:24 to 23:28

Commands:

```bash
nl -ba geppetto/pkg/inference/middlewarecfg/source.go | sed -n '1,220p'
nl -ba geppetto/pkg/inference/middlewarecfg/resolver.go | sed -n '1,260p'
nl -ba geppetto/pkg/inference/middlewarecfg/resolver.go | sed -n '260,420p'
nl -ba geppetto/pkg/inference/middlewarecfg/debug_payload.go | sed -n '1,220p'
nl -ba geppetto/pkg/inference/middlewarecfg/resolver_test.go | sed -n '70,220p'
nl -ba geppetto/pkg/inference/middlewarecfg/resolver_test.go | sed -n '260,460p'
```

Findings:

1. Middleware config resolver already has canonical layer precedence, deterministic ordering, and path-level trace history.
2. Deterministic debug serialization is test-covered.
3. This model directly informs profile stack provenance design.

Interpretation:

1. Stack profile provenance should mirror `middlewarecfg` trace design for consistency.

## Phase 4 - Registry docs/tests/integration boundary checks

Time window: 23:28 to 23:33

Commands:

```bash
nl -ba geppetto/pkg/doc/topics/01-profiles.md | sed -n '1,220p'
nl -ba geppetto/pkg/doc/topics/01-profiles.md | sed -n '220,420p'
nl -ba geppetto/pkg/profiles/service_test.go | sed -n '1,260p'
nl -ba geppetto/pkg/profiles/service_test.go | sed -n '620,760p'
nl -ba geppetto/pkg/sections/sections.go | sed -n '150,340p'
nl -ba geppetto/pkg/sections/profile_registry_source_test.go | sed -n '180,280p'
```

Findings:

1. Docs clearly describe registry-first hard cutover model and profile precedence.
2. Regression tests verify profile-registry source parity with legacy gather-flags behavior.
3. Settings source ordering is explicit: flags > env > profile > config > defaults in effective outcome.

Interpretation:

1. Stack profile merge must slot inside profile resolution while preserving outer source precedence contract.

## Phase 5 - Middleware extension and persistence constraints

Time window: 23:33 to 23:36

Commands:

```bash
nl -ba geppetto/pkg/profiles/middleware_extensions.go | sed -n '1,260p'
nl -ba geppetto/pkg/profiles/middleware_extensions_test.go | sed -n '1,220p'
nl -ba geppetto/pkg/profiles/codec_yaml.go | sed -n '1,280p'
nl -ba geppetto/pkg/profiles/codec_yaml_test.go | sed -n '1,220p'
nl -ba geppetto/pkg/profiles/sqlite_store.go | sed -n '1,320p'
```

Findings:

1. Middleware instance config is already mapped into typed extension keys by instance slot (`id:` or `index:`).
2. YAML and SQLite persistence store full profile JSON/YAML payloads, so adding `stack` fields is schema-compatible with existing row shape.

Interpretation:

1. Stack fields can be added without changing SQLite table shape.
2. Middleware merge should prefer `id` keys for deterministic stacking.

## Phase 6 - JS API boundary check

Time window: 23:36 to 23:40

Commands:

```bash
find geppetto/pkg/js/modules/geppetto -maxdepth 3 -type f | sort
rg -n "profile|registry|createEngineFactory|middleware schema|resolve" geppetto/pkg/js/modules/geppetto geppetto/examples/js -S
nl -ba geppetto/pkg/js/modules/geppetto/api_engines.go | sed -n '1,360p'
nl -ba geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl | sed -n '1,360p'
nl -ba geppetto/pkg/js/modules/geppetto/module_test.go | sed -n '540,760p'
```

Findings:

1. JS `fromProfile` currently resolves model/provider/env directly and does not read profile registry.
2. Existing tests assert profile precedence only in that model-centric interpretation.

Interpretation:

1. Stack profiles require explicit JS API additions (or semantic cutover) to call registry resolver.

## Phase 7 - Failed probes and corrections

Time window: throughout investigation

Failed commands:

```bash
ls geppetto/pkg/profiles | rg runtime_settings_patch_resolver_test.go -n
nl -ba geppetto/pkg/profiles/runtime_settings_patch_resolver_test.go | sed -n '1,260p'
nl -ba geppetto/pkg/js/modules/geppetto/geppetto.go | sed -n '1,320p'
nl -ba geppetto/pkg/js/modules/geppetto/helpers.go | sed -n '1,320p'
nl -ba geppetto/pkg/profiles/codec_json.go | sed -n '1,320p'
```

Outcomes:

1. Some expected files did not exist; switched to actual file locations (`api_engines.go`, existing tests/codecs).
2. No blocker remained after correcting paths.

## Phase 8 - Drafting the design proposal

Time window: 23:40 onward

Actions:

1. Wrote comprehensive design doc covering:
   1. current-state evidence,
   2. gap analysis,
   3. stack model,
   4. merge semantics,
   5. provenance trace model,
   6. phased implementation plan,
   7. testing strategy,
   8. risks/alternatives/open questions.
2. Included explicit references to evidence files/lines.

## Phase 9 - Remaining delivery steps (to execute now)

Planned commands:

```bash
docmgr doc relate --doc <design-doc> --file-note "/abs/path:reason" ...
docmgr doc relate --doc <diary-doc> --file-note "/abs/path:reason" ...
docmgr changelog update --ticket GP-28-STACK-PROFILES --entry "..." --file-note "/abs/path:reason" ...
docmgr doctor --ticket GP-28-STACK-PROFILES --stale-after 30
remarquee status
remarquee cloud account --non-interactive
remarquee upload bundle --dry-run ...
remarquee upload bundle ...
remarquee cloud ls /ai/2026/02/24/GP-28-STACK-PROFILES --long --non-interactive
```

## Review checklist for this diary

1. Commands are copy/paste-ready.
2. Findings and interpretations are explicitly separated.
3. Failed attempts and corrections are recorded.
4. Delivery/validation steps are listed and reproducible.

## Phase 10 - GP-28 v2 simplification update (geppetto-first)

Time window: 2026-02-25 late evening (local)

### User direction captured

Update GP-28 in light of newer cross-repo research and simplification goals:

1. Geppetto first.
2. Adapt pinocchio CLI/web-chat and go-go-os later.
3. Remove unnecessary complexity (explicitly including overlay when unused).

### Additional evidence checked before rewrite

Commands used:

```bash
rg -n "profile registry|ResolveEffectiveProfile|RegisterProfileAPIHandlers|runtime_composer|profile_policy|newInMemoryProfileService" pinocchio go-go-os geppetto/pkg
nl -ba pinocchio/cmd/web-chat/profile_policy.go | sed -n '430,560p'
nl -ba pinocchio/cmd/web-chat/runtime_composer.go | sed -n '1,220p'
nl -ba geppetto/pkg/profiles/overlay.go | sed -n '1,80p'
```

Findings:

1. Downstream apps are already app-owned resolver/composer integrations and can be adapted later with bounded blast radius.
2. Overlay abstraction remains a separate store mechanism and is not required for stack profile implementation.
3. Geppetto core is still the only correct place to implement stack semantics/fingerprint contracts first.

### Changes made in this step

1. Rewrote GP-28 design doc to v2 hard-cutover geppetto-first plan:
   - `design-doc/01-stack-profiles-architecture-and-merge-provenance-for-provider-model-middleware-layering.md`
2. Reduced v1 scope in doc:
   - no request-time stack override,
   - no cross-registry stack refs,
   - minimal provenance payload for v1,
   - explicit overlay removal from near-term plan.
3. Updated task list to reflect implementation still pending (core first, downstream later):
   - `tasks.md`
4. Appended changelog entry documenting v2 scope/order simplification:
   - `changelog.md`

### Outcome

GP-28 now reflects current execution strategy:

1. implement stack profiles in Geppetto first,
2. hard-cut JS profile semantics to registry-backed behavior,
3. remove overlay complexity,
4. defer pinocchio/go-go-os adaptation to follow-on implementation tickets.

## Phase 11 - Scope correction to v3 (restore full stack scope, keep overlay removal)

Time window: 2026-02-26 early morning (local)

### Requirement update

User clarified final requirements for GP-28:

1. multiple profile registries are required,
2. field provenance should use middlewarecfg trace pattern,
3. request-time overrides should remain supported per `PolicySpec`,
4. simplification to keep: remove overlay complexity.

### Impact on previous v2

The previous v2 simplification removed too much scope (especially multi-registry and full trace). This step restores those capabilities while retaining the overlay-removal simplification.

### Changes made

1. Rewrote design doc to v3 hard-cutover geppetto-first with:
   - cross-registry stack refs (`ProfileRef.RegistrySlug`),
   - middlewarecfg-style `Trace[path].Steps[]` provenance model,
   - explicit policy-gated request override behavior,
   - lineage-inclusive fingerprint requirements,
   - overlay removal still in implementation plan.
2. Updated tasks to reflect restored scope.
3. Appended changelog entry for v3 scope correction.

### Net decision

Design direction is now:

1. mostly back to original full-stack architecture,
2. but with overlay removed from implementation path,
3. and still sequenced geppetto-first, downstream later.

## Phase 12 - Detailed granular task plan authoring

Time window: 2026-02-26 early morning (local)

### Objective

Translate v3 scope into an execution-grade granular checklist so implementation can proceed without ambiguity.

### Work performed

1. Replaced `tasks.md` with a phased backlog from pre-implementation guardrails through downstream ticket creation.
2. Added file-targeted tasks and test gates for each major subsystem:
   - `pkg/profiles` domain/validation/resolver/merge/provenance/fingerprint,
   - overlay removal,
   - JS hard cutover,
   - docs and verification.
3. Added explicit Definition of Done checklist aligned with v3 requirements.

### Outcome

GP-28 now has a detailed implementation task list suitable for direct execution and progress tracking.

## Phase 13 - Implementation kickoff: Phase 1 domain model

Time window: 2026-02-25 13:45 to 13:55 (local)

### Objective

Execute the first implementation slice from the granular backlog: add stack reference fields to the core profile model and verify clone safety.

### Commands run

```bash
git -C /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto status --short
go test ./pkg/profiles
rg --files ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance
sed -n '1,220p' ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/changelog.md
sed -n '1,260p' ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/reference/01-investigation-diary.md
sed -n '1,320p' ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md
git -C /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto diff -- pkg/profiles/types.go pkg/profiles/types_clone_test.go
date -Iseconds
```

### Code changes completed

1. `pkg/profiles/types.go`
   - added `ProfileRef` with `registry_slug` (optional) and `profile_slug` (required),
   - added `Profile.Stack []ProfileRef`,
   - updated `Profile.Clone()` to deep-copy `Stack`.
2. `pkg/profiles/types_clone_test.go`
   - extended deep-copy test to include `Stack`,
   - mutated cloned refs and asserted original stack refs remain unchanged.

### Verification

1. `go test ./pkg/profiles` passed after the changes.

### Ticket updates completed

1. Marked all Phase 1 task checkboxes complete in `tasks.md`.
2. Added changelog entry documenting this implementation slice and verification.
3. Appended this execution record to the investigation diary.

### Notes

1. Existing uncommitted ticket workspace files from GP-21/GP-28 remained untouched outside intentional doc updates for this phase.
2. Next implementation slice is Phase 2 (serialization and persistence round-trip coverage for `stack` fields).

## Phase 14 - Implementation Phase 2 serialization/persistence coverage

Time window: 2026-02-25 13:55 to 13:59 (local)

### Objective

Prove stack references persist cleanly through YAML and SQLite stores (including cross-registry refs), and verify backend parity across memory/yaml/sqlite stores.

### Commands run

```bash
sed -n '1,260p' pkg/profiles/codec_yaml.go
sed -n '1,320p' pkg/profiles/codec_yaml_test.go
sed -n '1,320p' pkg/profiles/sqlite_store_test.go
sed -n '1,360p' pkg/profiles/integration_store_parity_test.go
sed -n '1,340p' pkg/profiles/sqlite_store.go
rg --files pkg/profiles | rg 'yaml|store'
sed -n '1,360p' pkg/profiles/file_store_yaml.go
sed -n '1,360p' pkg/profiles/file_store_yaml_test.go
gofmt -w pkg/profiles/codec_yaml_test.go pkg/profiles/sqlite_store_test.go pkg/profiles/integration_store_parity_test.go
go test ./pkg/profiles
date -Iseconds
```

### Findings before edits

1. YAML and SQLite stores already serialize full `Profile` payloads via schema-driven marshal/unmarshal.
2. No additional store schema/migration changes were required for `stack`.
3. Coverage gaps existed for stack-specific round-trip assertions.

### Code changes completed

1. `pkg/profiles/codec_yaml_test.go`
   - added `TestEncodeDecodeYAML_PreservesStackRefs`.
2. `pkg/profiles/sqlite_store_test.go`
   - added `TestSQLiteProfileStore_RegistryRoundTrip_PreservesStackRefs`.
3. `pkg/profiles/integration_store_parity_test.go`
   - added `TestStoreRegistryStackRefParityAcrossBackends`.

### Verification

1. `go test ./pkg/profiles` passed.

### Ticket updates completed

1. Marked Phase 2 checklist items complete in `tasks.md`.
2. Added changelog entry for Phase 2 implementation and test verification.

### Notes

1. Phase 2 satisfied by test coverage because persistence layer shape already handled `Profile.Stack` via existing YAML/JSON serialization.
2. Next implementation slice is Phase 3 validation (stack ref validation, cycle/depth safeguards).

## Phase 15 - Implementation Phase 3 validation

Time window: 2026-02-25 13:59 to 14:02 (local)

### Objective

Implement stack-specific validation guarantees: per-ref slug validity, missing-ref detection, cycle detection, and max-depth guard paths.

### Commands run

```bash
sed -n '1,360p' pkg/profiles/validation.go
sed -n '1,420p' pkg/profiles/validation_test.go
sed -n '1,280p' pkg/profiles/errors.go
rg -n "cycle|max depth|stack|validation" pkg/profiles -S
gofmt -w pkg/profiles/validation.go pkg/profiles/validation_test.go
go test ./pkg/profiles
date -Iseconds
```

### Code changes completed

1. `pkg/profiles/validation.go`
   - added `DefaultProfileStackValidationMaxDepth`,
   - added `StackValidationOptions`,
   - added `ValidateProfileRef`,
   - extended `ValidateProfile` to validate `Stack` refs,
   - added `ValidateProfileStackTopology` helper with:
     - missing ref validation,
     - cycle detection,
     - depth overflow validation,
   - wired `ValidateRegistry` to run topology validation for single-registry checks with unresolved external refs allowed.
2. `pkg/profiles/validation_test.go`
   - added tests for:
     - empty stack `profile_slug`,
     - invalid stack `registry_slug`,
     - missing same-registry ref,
     - cycle detection,
     - max-depth overflow,
     - missing cross-registry ref,
     - allow-unresolved cross-registry mode.

### Verification

1. `go test ./pkg/profiles` passed.

### Ticket updates completed

1. Marked Phase 3 task checklist complete in `tasks.md`.
2. Added changelog entry for validation implementation and test coverage.

### Notes

1. Validation layer now has a reusable topology check API that later resolver integration can invoke directly for preflight.
2. Next implementation slice is Phase 4 stack expansion resolver.

## Phase 16 - Implementation Phase 4 stack expansion resolver

Time window: 2026-02-25 14:02 to 14:05 (local)

### Objective

Add a deterministic stack expansion resolver that can traverse cross-registry refs and emit explicit resolver errors for cycles, missing refs, and max-depth breaches.

### Commands run

```bash
sed -n '1,360p' pkg/profiles/service.go
sed -n '360,760p' pkg/profiles/service.go
sed -n '1,340p' pkg/profiles/registry.go
rg -n "ResolveEffectiveProfile|resolve.*profile|stack|overlay" pkg/profiles -S
sed -n '1,260p' pkg/profiles/memory_store.go
gofmt -w pkg/profiles/stack_resolver.go pkg/profiles/stack_resolver_test.go
go test ./pkg/profiles
date -Iseconds
```

### Code changes completed

1. Added `pkg/profiles/stack_resolver.go` with:
   - `StackResolverOptions`,
   - `ProfileStackLayer`,
   - `(*StoreRegistry).ExpandProfileStack(...)`.
2. Resolver behavior implemented:
   - DFS-based expansion,
   - same-registry default for empty `registry_slug`,
   - explicit cross-registry target resolution,
   - deterministic dedupe by first occurrence,
   - base->leaf layer ordering,
   - cycle error with explicit chain string,
   - missing registry/profile errors bound to stack field paths,
   - depth overflow checks.
3. Added `pkg/profiles/stack_resolver_test.go` with comprehensive matrix:
   - linear stack,
   - fan-in dedupe,
   - cross-registry refs,
   - cycle,
   - missing registry,
   - missing profile,
   - max-depth breach.

### Verification

1. `go test ./pkg/profiles` passed.

### Ticket updates completed

1. Marked Phase 4 checklist complete in `tasks.md`.
2. Added changelog entry describing resolver implementation and tests.

### Notes

1. Resolver now exists as a standalone primitive to plug into `ResolveEffectiveProfile` in later phases.
2. Next implementation slice is Phase 5 merge engine.

## Phase 17 - Implementation Phase 5 merge engine

Time window: 2026-02-25 14:05 to 14:08 (local)

### Objective

Add deterministic stack-layer merge logic for runtime, policy, and extensions with explicit conflict semantics.

### Commands run

```bash
sed -n '1,280p' pkg/profiles/types.go
rg -n "deep merge|Merge.*map|merge.*extensions|middleware key|name#id|index\\]" pkg/profiles pkg/inference/middlewarecfg -S
gofmt -w pkg/profiles/stack_merge.go pkg/profiles/stack_merge_test.go
go test ./pkg/profiles
date -Iseconds
```

### Code changes completed

1. Added `pkg/profiles/stack_merge.go`:
   - `StackMergeResult`,
   - `MergeProfileStackLayers`,
   - runtime merge helpers,
   - middleware key merge helpers,
   - extension deep merge helpers,
   - restrictive policy merge helpers.
2. Added `pkg/profiles/stack_merge_test.go`:
   - runtime merge behavior tests,
   - middleware merge key replacement tests,
   - extension deep/object-vs-scalar merge tests,
   - restrictive policy merge tests.

### Verification

1. `go test ./pkg/profiles` passed.

### Ticket updates completed

1. Marked Phase 5 checklist complete in `tasks.md`.
2. Added changelog entry for merge engine + tests.

### Notes

1. Merge engine now provides deterministic base->leaf materialization independent of request overrides.
2. Next implementation slice is Phase 6 request-time overrides integration on merged runtime.

## Phase 18 - Implementation Phase 6 request-time override integration

Time window: 2026-02-25 14:08 to 14:10 (local)

### Objective

Wire request-time overrides to run against stack-merged runtime/policy in the main `ResolveEffectiveProfile` path.

### Commands run

```bash
sed -n '1,260p' pkg/profiles/service_test.go
sed -n '260,760p' pkg/profiles/service_test.go
gofmt -w pkg/profiles/service.go pkg/profiles/service_test.go
go test ./pkg/profiles
date -Iseconds
```

### Code changes completed

1. `pkg/profiles/service.go`
   - `ResolveEffectiveProfile` now:
     - resolves stack layers via `ExpandProfileStack`,
     - merges layers via `MergeProfileStackLayers`,
     - applies request overrides using merged runtime + merged policy.
2. `pkg/profiles/service_test.go`
   - added tests for stack-aware override behavior:
     - allowed override over stack result,
     - denied override key over stack result,
     - allow-list policy enforcement over stack result,
     - `AllowOverrides=false` in stack blocking overrides.

### Verification

1. `go test ./pkg/profiles` passed.

### Ticket updates completed

1. Marked Phase 6 checklist complete in `tasks.md`.
2. Added changelog entry for stack-aware override integration.

### Notes

1. Request override gate semantics are still enforced in one place (`resolveRuntimeSpec`), now fed by merged stack policy.
2. Next implementation slice is Phase 7 provenance trace model.

## Phase 19 - Implementation Phase 7 provenance trace model

Time window: 2026-02-25 14:10 to 14:13 (local)

### Objective

Add middlewarecfg-style path-level provenance tracing for stack merges, with deterministic ordered debug serialization.

### Commands run

```bash
sed -n '1,320p' pkg/inference/middlewarecfg/resolver.go
sed -n '1,260p' pkg/inference/middlewarecfg/debug_payload.go
rg -n "Trace|steps|path" pkg/inference/middlewarecfg -S
sed -n '1,320p' pkg/inference/middlewarecfg/resolver_test.go
gofmt -w pkg/profiles/stack_trace.go pkg/profiles/stack_merge.go pkg/profiles/stack_trace_test.go
go test ./pkg/profiles
date -Iseconds
```

### Code changes completed

1. Added `pkg/profiles/stack_trace.go` with:
   - `ProfileStackTrace`,
   - `ProfilePathTrace`,
   - `ProfileStackTraceStep`,
   - path write collection,
   - history/latest-value helpers,
   - deterministic debug payload + JSON marshal helpers.
2. Updated `pkg/profiles/stack_merge.go`:
   - added `MergeProfileStackLayersWithTrace` returning merge result + trace.
3. Added `pkg/profiles/stack_trace_test.go`:
   - path history ordering assertions,
   - final winner-value assertions,
   - deterministic serialization assertions.

### Verification

1. `go test ./pkg/profiles` passed.

### Ticket updates completed

1. Marked Phase 7 checklist complete in `tasks.md`.
2. Added changelog entry for trace model and tests.

### Notes

1. Trace model now exists and can be surfaced through `ResolveEffectiveProfile` metadata in subsequent integration work.
2. Next implementation slice is Phase 8 fingerprint updates.

## Phase 20 - Implementation Phase 8 fingerprint lineage updates

Time window: 2026-02-25 14:13 to 14:15 (local)

### Objective

Make runtime fingerprints stack-lineage-aware and verify sensitivity to stack topology/version/override changes.

### Commands run

```bash
gofmt -w pkg/profiles/service.go pkg/profiles/service_test.go
go test ./pkg/profiles
date -Iseconds
```

### Code changes completed

1. `pkg/profiles/service.go`
   - updated `runtimeFingerprint` to include ordered stack lineage entries:
     - `registry_slug`,
     - `profile_slug`,
     - `version`,
     - `source`.
2. `pkg/profiles/service_test.go`
   - added fingerprint behavior tests for:
     - upstream layer version bump sensitivity,
     - layer order sensitivity,
     - request override sensitivity,
     - non-stack identical run stability.

### Verification

1. `go test ./pkg/profiles` passed.

### Ticket updates completed

1. Marked Phase 8 checklist complete in `tasks.md`.
2. Added changelog entry for lineage-aware fingerprint behavior.

### Notes

1. Fingerprint contract now covers stack identity ordering and layer version/source changes.
2. Next implementation slice is Phase 9 resolve integration finalization (trace/fingerprint metadata wiring and regression confirmation).

## Phase 21 - Implementation Phase 9 resolve integration finalization

Time window: 2026-02-25 14:15 to 14:16 (local)

### Objective

Finalize stack-aware resolve integration by surfacing lineage/trace metadata while preserving existing resolver contract.

### Commands run

```bash
rg -n "ResolveForRuntime|ResolveEffectiveProfile|RuntimeComposer|Resolve.*Runtime" pkg/profiles pkg -S
gofmt -w pkg/profiles/service.go pkg/profiles/service_test.go
go test ./pkg/profiles
date -Iseconds
```

### Code changes completed

1. `pkg/profiles/service.go`
   - switched resolve merge call to `MergeProfileStackLayersWithTrace`,
   - added response metadata fields:
     - `profile.stack.lineage`,
     - `profile.stack.trace`,
   - reused lineage helper for fingerprint payload consistency.
2. `pkg/profiles/service_test.go`
   - added metadata assertions for non-stack and stacked resolve flows:
     - lineage presence and ordering,
     - trace payload presence/type.

### Verification

1. `go test ./pkg/profiles` passed.

### Ticket updates completed

1. Marked Phase 9 checklist complete in `tasks.md`.
2. Added changelog entry for resolve integration finalization.

### Notes

1. `ResolveForRuntime` is not a separate function in current codebase; stack-aware behavior is exercised via `ResolveEffectiveProfile` and downstream call sites.
2. Next implementation slice is Phase 10 overlay abstraction removal.

## Phase 22 - Implementation Phase 10 overlay removal

Time window: 2026-02-25 14:16 to 14:18 (local)

### Objective

Delete unused overlay abstraction and confirm package/test health without it.

### Commands run

```bash
rg -n "overlay|OverlayStore|NewOverlayStore" pkg/profiles pkg -S
rg -n "overlay|OverlayStore|overlay store" pkg/doc/topics/01-profiles.md pkg/doc/topics -S
go test ./pkg/profiles
go test ./...
date -Iseconds
```

### Code changes completed

1. Deleted `pkg/profiles/overlay.go`.
2. Deleted `pkg/profiles/overlay_test.go`.

### Verification

1. `go test ./pkg/profiles` passed.
2. `go test ./...` passed.

### Ticket updates completed

1. Marked Phase 10 checklist complete in `tasks.md`.
2. Added changelog entry documenting overlay removal and verification.

### Notes

1. Overlay removal did not require downstream code changes in this repo.
2. Next implementation slice is Phase 11 JS API hard cutover.

## Phase 23 - Implementation Phase 11 JS API hard cutover

Time window: 2026-02-25 14:18 to 14:25 (local)

### Objective

Hard-cut Geppetto JS `engines.fromProfile` to profile-registry resolution semantics, wire runtime dependencies, and verify with module tests plus repository pre-commit gates.

### Prompt Context

**User prompt (verbatim):** "Alright, let's work on this, task by task, keeping commits after each phase or whenever appropriate, and keeping a frequent detailed diary."

**Assistant interpretation:** execute GP-28 implementation phases sequentially with a commit per meaningful phase and keep the ticket diary/changelog/tasks synchronized.

**Inferred user intent:** complete the stack-profile rollout in an auditable way where each phase is self-contained, test-validated, and easy to review/continue.

**Commit (code):** `0f7a7a9` — "js: hard-cut engines.fromProfile to profile registry"

### Commands run

```bash
rg -n "deepCopyStringAnyMap|metadata" pkg/js/modules/geppetto/api_engines.go pkg/js/modules/geppetto/api_types.go pkg/js/modules/geppetto/module.go pkg/js/modules/geppetto/module_test.go
sed -n '1,520p' pkg/js/modules/geppetto/api_engines.go
go test ./pkg/js/modules/geppetto
gofmt -w pkg/js/modules/geppetto/api_engines.go
go test ./pkg/js/modules/geppetto
go test ./pkg/profiles
git add pkg/js/modules/geppetto/api_engines.go pkg/js/modules/geppetto/api_types.go pkg/js/modules/geppetto/module.go pkg/js/modules/geppetto/module_test.go pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl
git commit -m "js: hard-cut engines.fromProfile to profile registry"
date -Iseconds
```

### What I did

1. Fixed JS module compile break by replacing missing helper calls (`deepCopyStringAnyMap`) with existing package clone helper (`cloneJSONMap`).
2. Completed `fromProfile` hard cutover so it resolves through `profiles.ResolveEffectiveProfile`.
3. Added profile registry wiring into module options/runtime (`Options.ProfileRegistry`, `moduleRuntime.profileRegistry`).
4. Updated engine refs to carry metadata and exposed it to JS engine objects.
5. Updated JS typings to introduce `ProfileEngineOptions` and removed legacy profile option from generic engine config.
6. Expanded/updated module tests for registry-backed resolution paths and missing-registry error behavior.
7. Committed phase code and validated via full lefthook pre-commit pipeline.

### Why

1. The old JS `fromProfile` behavior was model/env-based and bypassed the profile registry stack/provenance/fingerprint pipeline.
2. Hard cutover is required by current scope: no backward-compat shims, explicit registry semantics, and parity with Go resolver behavior.

### What worked

1. Registry-backed `fromProfile` implementation compiled and passed module tests after helper fix.
2. Full pre-commit gates passed (tests/lint/vet/build), confirming repository-wide health with the JS cutover in place.
3. Metadata payload wiring gives JS access to resolver outputs needed for downstream caching/diagnostics.

### What didn't work

1. Initial test run failed to compile due to unresolved helper references:
   - command: `go test ./pkg/js/modules/geppetto`
   - error: `undefined: deepCopyStringAnyMap` in `api_engines.go` call sites.
2. Resolved by switching to existing `cloneJSONMap` helper in this package.

### What I learned

1. JS module already had a stable JSON-clone helper path (`cloneJSONValue`/`cloneJSONMap`), so introducing profile metadata did not require new utility code.
2. Registry-backed `fromProfile` can preserve live inference usability by hydrating missing provider keys from env after resolver output cloning.

### What was tricky to build

The tricky edge was preserving semantic boundaries while hard-cutting behavior:
1. `fromProfile` must stop honoring legacy model/env precedence.
2. `fromConfig` must continue to support direct ad-hoc engine configuration.
3. Resolver output must be surfaced to JS without aliasing mutable maps.

The compile failure surfaced because the first implementation used an internal helper that exists in `pkg/profiles` but is not exported to JS module code. Reusing package-local clone helpers fixed both compilation and map safety constraints.

### What warrants a second pair of eyes

1. Metadata contract shape exposed to JS (`metadata.runtimeFingerprint`, `metadata.resolvedMetadata`) should be reviewed against downstream consumer expectations.
2. Env API-key hydration for resolved settings should be reviewed for any provider-specific edge behavior beyond current coverage.

### What should be done in the future

1. Continue with Phase 12 documentation updates for hard-cut JS semantics and overlay removal rationale.
2. Complete Phase 13 verification matrix including downstream compile/tests in pinocchio and go-go-os.

### Code review instructions

1. Start with:
   - `pkg/js/modules/geppetto/api_engines.go`,
   - `pkg/js/modules/geppetto/module.go`,
   - `pkg/js/modules/geppetto/module_test.go`,
   - `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`.
2. Validate with:
   - `go test ./pkg/js/modules/geppetto`,
   - `go test ./pkg/profiles`,
   - `go test ./...` (already run by pre-commit).

### Technical details

1. Commit: `0f7a7a9`.
2. Phase tasks/changelog were updated in:
   - `ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md`,
   - `ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/changelog.md`.

## Phase 24 - Phase 11 follow-up TS artifact/type consistency

Time window: 2026-02-25 14:25 to 14:27 (local)

### Objective

Close API contract drift introduced by Phase 11 by synchronizing generated TS artifacts and declaring `engine.metadata` in the public JS type contract.

### Prompt Context

**User prompt (verbatim):** (see Step 23)

**Assistant interpretation:** continue phase-based delivery and commit follow-up fixes when needed to keep API/documentation consistency.

**Inferred user intent:** avoid partial phase output where runtime behavior and type surfaces diverge.

**Commit (code):** `a05c587` — "docs: regenerate geppetto JS type definitions"  
**Commit (code):** `1f5cac5` — "js: add engine metadata to TypeScript contract"

### Commands run

```bash
git diff -- pkg/doc/types/geppetto.d.ts
git add pkg/doc/types/geppetto.d.ts
git commit -m "docs: regenerate geppetto JS type definitions"
apply_patch pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl
apply_patch pkg/doc/types/geppetto.d.ts
go test ./pkg/js/modules/geppetto
git add pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl pkg/doc/types/geppetto.d.ts
git commit -m "js: add engine metadata to TypeScript contract"
```

### What I did

1. Captured and committed regenerated `pkg/doc/types/geppetto.d.ts` output produced by prior generation hooks.
2. Added optional `metadata` field to `Engine` type in both TS template and generated docs copy.
3. Re-ran module tests to confirm no runtime/type regression.

### Why

1. Phase 11 exposed metadata at runtime, but TS contract still declared only `name`.
2. Keeping generated artifacts committed avoids spurious working-tree diffs and review confusion.

### What worked

1. Both follow-up commits were clean and isolated.
2. Module tests remained green after type-surface update.

### What didn't work

N/A

### What I learned

1. `go generate` in pre-commit can leave non-staged generated docs diffs unless explicitly committed after the phase.

### What was tricky to build

No algorithmic complexity; the tricky part was sequencing:
1. commit generated artifact drift first,
2. then declare the missing type surface and regenerate/patch consistently.

### What warrants a second pair of eyes

1. Whether any downstream TS consumers depend on a stricter `Engine` shape and need lint/builder updates.

### What should be done in the future

1. Keep generated TS artifact sync as part of the JS API phase checklist by default.

### Code review instructions

1. Review:
   - `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`,
   - `pkg/doc/types/geppetto.d.ts`.
2. Validate:
   - `go test ./pkg/js/modules/geppetto`.

### Technical details

1. Commits: `a05c587`, `1f5cac5`.

## Phase 25 - Implementation Phase 12 documentation updates

Time window: 2026-02-25 14:27 to 14:29 (local)

### Objective

Update Geppetto profiles and JS API docs for stack profiles, hard-cut `fromProfile`, policy-gated request overrides, and explicit overlay removal rationale.

### Prompt Context

**User prompt (verbatim):** (see Step 23)

**Assistant interpretation:** keep implementation and documentation moving in lockstep with commit boundaries and detailed diary evidence.

**Inferred user intent:** ensure design/usage docs reflect the actual shipped API so downstream migration work can proceed without ambiguity.

**Commit (code):** `189eff0` — "docs: update profile stack and JS fromProfile hard-cut semantics"

### Commands run

```bash
rg -n "fromProfile|overlay|stack|registry|request override|requestOverrides" pkg/doc/topics/01-profiles.md pkg/doc/topics/13-js-api-reference.md pkg/doc/topics/14-js-api-user-guide.md -S
apply_patch pkg/doc/topics/01-profiles.md
apply_patch pkg/doc/topics/13-js-api-reference.md
apply_patch pkg/doc/topics/14-js-api-user-guide.md
git add pkg/doc/topics/01-profiles.md pkg/doc/topics/13-js-api-reference.md pkg/doc/topics/14-js-api-user-guide.md
git commit -m "docs: update profile stack and JS fromProfile hard-cut semantics"
date -Iseconds
```

### What I did

1. Updated `01-profiles.md` to document:
   - `Profile.Stack` / `ProfileRef`,
   - multi-registry stack semantics,
   - stack-aware resolve outputs (`profile.stack.lineage`, `profile.stack.trace`),
   - merged-policy request override behavior,
   - overlay removal rationale.
2. Updated `13-js-api-reference.md` to document:
   - `fromProfile` hard-cut registry semantics,
   - `ProfileEngineOptions`,
   - required `Options.ProfileRegistry`,
   - `engine.metadata` payload fields.
3. Updated `14-js-api-user-guide.md` with hard-cut usage guidance and troubleshooting row for missing profile registry.
4. Marked Phase 12 complete in ticket tasks and appended changelog entries.

### Why

1. Phase 11 changed runtime and API contracts; docs needed to match to avoid downstream integration errors.
2. GP-28 scope explicitly calls for removing future-facing complexity and documenting overlay removal.

### What worked

1. Documentation now matches current hard-cut runtime behavior.
2. Cross-doc terminology is consistent (`registry-backed`, `policy-gated`, `stack lineage/trace`).

### What didn't work

N/A

### What I learned

1. JS API reference previously encoded legacy `fromProfile` precedence assumptions that are now fully obsolete.

### What was tricky to build

The subtle part was keeping examples valid against actual override key normalization. I corrected request override example keys to canonicalizable runtime keys (`systemPrompt` -> `system_prompt`) rather than invalid nested keys.

### What warrants a second pair of eyes

1. Documentation examples should be cross-checked by downstream teams (pinocchio/go-go-os) against their host wiring conventions for `Options.ProfileRegistry`.

### What should be done in the future

1. Execute Phase 13 downstream verification matrix and record compile/test outcomes for pinocchio + go-go-os.

### Code review instructions

1. Review:
   - `pkg/doc/topics/01-profiles.md`,
   - `pkg/doc/topics/13-js-api-reference.md`,
   - `pkg/doc/topics/14-js-api-user-guide.md`.
2. Validate:
   - compare docs against current runtime/type contract in:
     - `pkg/js/modules/geppetto/api_engines.go`,
     - `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`,
     - `pkg/profiles/service.go`.

### Technical details

1. Commit: `189eff0`.

## Phase 26 - Implementation Phase 13 verification matrix

Time window: 2026-02-25 14:29 to 14:31 (local)

### Objective

Execute the Phase 13 verification matrix across geppetto core packages and focused downstream targets (pinocchio, go-go-os).

### Prompt Context

**User prompt (verbatim):** (see Step 23)

**Assistant interpretation:** continue phase-by-phase execution with explicit validation evidence before moving into downstream-ticket handoff.

**Inferred user intent:** keep core completion quality high by proving compatibility with major downstream consumers before declaring GP-28 implementation complete.

### Commands run

```bash
go test ./pkg/profiles
go test ./pkg/inference/middlewarecfg
go test ./pkg/js/modules/geppetto
go test ./cmd/web-chat/...                      # in pinocchio
go test ./go-inventory-chat/...                 # in go-go-os
```

### What I did

1. Ran all required core verification commands listed in GP-28 Phase 13.
2. Ran focused downstream compile/test commands in pinocchio and go-go-os.
3. Recorded results and marked Phase 13 checklist complete in GP-28 tasks.

### Why

1. GP-28 implementation changes affect shared profile/runtime contracts used by both downstream applications.
2. A green downstream baseline is required before creating adaptation handoff tickets.

### What worked

1. All core verification commands passed.
2. Both downstream focused test targets passed:
   - pinocchio `cmd/web-chat/...`,
   - go-go-os `go-inventory-chat/...`.

### What didn't work

N/A

### What I learned

1. Current workspace state is internally consistent for stack profiles and JS cutover at compile/test level across the two primary downstream consumers.

### What was tricky to build

No implementation complexity in this phase; the main risk was ensuring verification commands targeted the correct rebased downstream paths (`cmd/web-chat/...` and `go-inventory-chat/...`).

### What warrants a second pair of eyes

1. Runtime behavior under real multi-registry stack payloads should still be validated through downstream feature-level tests once adaptation work starts.

### What should be done in the future

1. Execute downstream adaptation implementation in dedicated tickets (Phase 14 handoff).

### Code review instructions

1. Re-run the verification matrix commands listed above in the same workspaces.
2. Compare results with GP-28 changelog entries for Phase 13.

### Technical details

1. No code commit in this phase (verification-only).

## Phase 27 - Implementation Phase 14 downstream adaptation tickets

Time window: 2026-02-25 14:31 to 14:32 (local)

### Objective

Create downstream follow-up tickets for pinocchio and go-go-os adoption work, and link them back to GP-28.

### Prompt Context

**User prompt (verbatim):** (see Step 23)

**Assistant interpretation:** complete GP-28 phased backlog through ticket handoff and cross-ticket linkage.

**Inferred user intent:** finish geppetto-first delivery and establish clean downstream execution queues without blocking current branch progress.

### Commands run

```bash
docmgr ticket list
docmgr ticket create-ticket --ticket GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER --title "Pinocchio stack-profile resolver/runtime composer cutover" --topics pinocchio,profile-registry,stack-profiles,migration
docmgr ticket create-ticket --ticket GP-30-GO-GO-OS-STACK-PROFILE-CUTOVER --title "Go-go-os stack-profile resolver/runtime composer cutover" --topics go-go-os,profile-registry,stack-profiles,migration
rg -n "ResolveEffectiveProfile|runtime_composer|profile_policy|runtimeFingerprint|request_resolver" pinocchio/cmd/web-chat go-go-os/go-inventory-chat -S
apply_patch ttmp/.../GP-29.../tasks.md
apply_patch ttmp/.../GP-29.../index.md
apply_patch ttmp/.../GP-29.../changelog.md
apply_patch ttmp/.../GP-30.../tasks.md
apply_patch ttmp/.../GP-30.../index.md
apply_patch ttmp/.../GP-30.../changelog.md
apply_patch ttmp/.../GP-28.../index.md
apply_patch ttmp/.../GP-28.../tasks.md
apply_patch ttmp/.../GP-28.../changelog.md
docmgr doctor --ticket GP-28-STACK-PROFILES --stale-after 30
docmgr doctor --ticket GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER --stale-after 30
docmgr doctor --ticket GP-30-GO-GO-OS-STACK-PROFILE-CUTOVER --stale-after 30
```

### What I did

1. Created:
   - `GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER`,
   - `GP-30-GO-GO-OS-STACK-PROFILE-CUTOVER`.
2. Filled each downstream ticket with phased tasks for:
   - resolver adoption,
   - runtime composer adoption,
   - metadata/fingerprint propagation,
   - verification and docs rollout.
3. Added relevant related files and GP-28 linkage in each ticket index/changelog.
4. Updated GP-28:
   - marked Phase 14 complete,
   - linked downstream tickets in index/changelog,
   - updated definition-of-done checkboxes for completed core outcomes.

### Why

1. GP-28 scope explicitly requires downstream adaptation tickets after core geppetto completion.
2. Dedicated downstream tickets keep GP-28 focused while enabling parallel migration work.

### What worked

1. Both tickets were created successfully with docmgr scaffolding.
2. Cross-ticket links are now present in GP-28 index/changelog.
3. `docmgr doctor` checks passed for GP-28, GP-29, and GP-30.

### What didn't work

N/A

### What I learned

1. The most relevant downstream touchpoints are already concentrated in:
   - pinocchio `profile_policy.go` / `runtime_composer.go`,
   - go-go-os `request_resolver.go` / `runtime_composer.go`.

### What was tricky to build

The key challenge was ensuring ticket scopes were narrow and implementation-oriented (not generic research buckets) while still mapping cleanly to GP-28 phase language and existing test entry points.

### What warrants a second pair of eyes

1. Ticket granularity/prioritization in GP-29 and GP-30 should be reviewed by downstream owners before implementation starts.

### What should be done in the future

1. Begin GP-29 implementation first (pinocchio web-chat path), then GP-30 as follow-on or parallel stream.

### Code review instructions

1. Review newly created tickets:
   - `ttmp/2026/02/25/GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER--pinocchio-stack-profile-resolver-runtime-composer-cutover/`,
   - `ttmp/2026/02/25/GP-30-GO-GO-OS-STACK-PROFILE-CUTOVER--go-go-os-stack-profile-resolver-runtime-composer-cutover/`.
2. Confirm GP-28 links/checklists in:
   - `ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/index.md`,
   - `ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/tasks.md`,
   - `ttmp/2026/02/24/GP-28-STACK-PROFILES--stack-profiles-provider-model-middleware-layering-with-merge-provenance/changelog.md`.

### Technical details

1. No code commit in this phase (ticketing/documentation handoff).
