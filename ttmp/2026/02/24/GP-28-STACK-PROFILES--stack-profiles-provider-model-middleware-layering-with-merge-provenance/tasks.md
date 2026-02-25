# Tasks

## Completed research/setup

- [x] Create ticket workspace `GP-28-STACK-PROFILES` with design + diary docs
- [x] Collect line-anchored evidence across `pkg/profiles`, `pkg/inference/middlewarecfg`, docs, tests, and JS bindings
- [x] Write comprehensive stack-profile architecture document with merge/provenance design and phased implementation plan
- [x] Write chronological investigation diary with commands, findings, failures, and decision rationale
- [x] Relate key source files to docs via `docmgr doc relate`
- [x] Append ticket changelog entries for research iterations
- [x] Run `docmgr doctor --ticket GP-28-STACK-PROFILES --stale-after 30`
- [x] Upload bundle to reMarkable
- [x] Update GP-28 to v3 scope:
  - multi-registry stack support,
  - middlewarecfg-style per-field trace provenance,
  - policy-gated request-time overrides,
  - overlay removed from implementation path.

## Phase 0 — Pre-implementation guardrails

- [ ] Lock v3 scope in design review notes (no overlay, yes multi-registry, yes full trace, yes policy-gated request overrides).
- [ ] Define stable error taxonomy for stack resolution failures:
  - unknown registry,
  - unknown profile ref,
  - cycle detected,
  - max depth exceeded,
  - invalid stack ref,
  - policy violation on request override.
- [ ] Confirm target invariants for backward behavior on non-stacked profiles (no regression in current resolve output).

## Phase 1 — Profile domain model (Geppetto core)

- [x] Add `ProfileRef` type to `geppetto/pkg/profiles/types.go`.
- [x] Add `Stack []ProfileRef` to `Profile` in `types.go`.
- [x] Update profile clone logic to deep-copy `Stack`.
- [x] Update JSON/YAML tags and comments for `ProfileRef` fields.
- [x] Add/extend tests for clone safety with stack refs (`types_clone_test.go`).

## Phase 2 — Serialization and persistence compatibility

- [x] Update YAML codec handling for stack refs in `geppetto/pkg/profiles/codec_yaml.go`.
- [x] Add YAML round-trip tests with:
  - same-registry ref,
  - cross-registry ref,
  - multi-layer stack.
- [x] Validate SQLite store read/write round-trip does not lose stack refs (`sqlite_store_test.go`).
- [x] Add parity test coverage in `integration_store_parity_test.go` for stack-bearing profiles.

## Phase 3 — Validation

- [x] Extend `geppetto/pkg/profiles/validation.go` with stack ref validation.
- [x] Validate per-ref slug correctness (`registry_slug`, `profile_slug`).
- [x] Add rules for disallowing empty `profile_slug` refs.
- [x] Add cycle detection pre-check API or resolver-linked validation helper.
- [x] Add max-depth guard config/constant and validation failure path.
- [x] Add focused tests for all validation failures in `validation_test.go`.

## Phase 4 — Stack expansion resolver

- [x] Add `geppetto/pkg/profiles/stack_resolver.go`.
- [x] Implement expansion over `(registry, profile)` identities.
- [x] Support empty `registry_slug` as “current registry”.
- [x] Support explicit cross-registry refs.
- [x] Preserve declared order via DFS expansion.
- [x] Deduplicate layers deterministically by first occurrence.
- [x] Emit explicit cycle chain in error payload.
- [x] Add comprehensive resolver tests:
  - linear stack,
  - fan-in stack,
  - cross-registry refs,
  - cycle case,
  - missing registry,
  - missing profile,
  - max depth breach.

## Phase 5 — Merge engine

- [x] Add `geppetto/pkg/profiles/stack_merge.go`.
- [x] Implement deterministic layer merge order (base -> leaf).
- [x] Runtime merge rules:
  - `step_settings_patch` deep merge via `MergeRuntimeStepSettingsPatches`,
  - `system_prompt` last non-empty wins,
  - `tools` replace-on-write,
  - `middlewares` merge by instance key.
- [x] Extensions merge rules:
  - deep object merge,
  - scalar/list replace.
- [x] Policy merge rules:
  - restrictive semantics preserved.
- [x] Middleware key behavior:
  - prefer `name#id`,
  - fallback `name[index]`.
- [x] Add merge-focused unit tests for each field rule and conflict case.

## Phase 6 — Request-time overrides (policy-gated)

- [x] Integrate stack-merged runtime with existing request override path in `service.go`.
- [x] Preserve existing policy gate semantics (`AllowOverrides`, denied/allowed key constraints).
- [x] Ensure canonical key normalization still applies to overrides.
- [x] Add tests for:
  - allowed request override over stack result,
  - denied request override over stack result,
  - denied-key rejection,
  - allowed-key allow-list behavior.

## Phase 7 — Provenance trace (middlewarecfg pattern)

- [x] Add provenance types in `geppetto/pkg/profiles` (new file: `stack_trace.go` or equivalent).
- [x] Implement path-level step accumulation mirroring `middlewarecfg` resolver trace style:
  - deterministic ordered paths,
  - per-path ordered steps,
  - final winner value per path.
- [x] Include layer metadata on each step (`registry`, `profile`, `version`, source/layer info).
- [x] Add deterministic serialization helper (similar spirit to `middlewarecfg/debug_payload.go`).
- [x] Add tests that assert stable ordering and payload determinism.

## Phase 8 — Runtime fingerprint updates

- [x] Update fingerprint payload generation in `geppetto/pkg/profiles/service.go`.
- [x] Include ordered layer lineage (`registry/profile/version`).
- [x] Include effective runtime + effective step settings after stack + overrides.
- [x] Add fingerprint sensitivity tests:
  - upstream layer version bump changes fingerprint,
  - layer reorder changes fingerprint,
  - override change changes fingerprint,
  - non-stack profile remains stable with prior semantics where expected.

## Phase 9 — `ResolveEffectiveProfile` integration

- [x] Refactor `ResolveEffectiveProfile` in `service.go` to call stack expansion + merge + provenance + fingerprint.
- [x] Keep public resolver entrypoint unchanged unless strictly necessary.
- [x] Ensure `ResolveForRuntime` uses stack-aware resolved output.
- [x] Keep non-stack behavior equivalent (regression tests required).

## Phase 10 — Remove overlay abstraction

- [x] Delete `geppetto/pkg/profiles/overlay.go`.
- [x] Delete `geppetto/pkg/profiles/overlay_test.go`.
- [x] Remove any references/imports/docs that suggest overlay as composition path.
- [x] Run package-wide tests to verify no dead references remain.

## Phase 11 — Geppetto JS API hard cutover

- [x] Update `geppetto/pkg/js/modules/geppetto/api_engines.go`:
  - make `engines.fromProfile` registry-backed,
  - remove model/env fallback semantics from this method.
- [x] Ensure resolver output can expose stack provenance/fingerprint to JS where needed.
- [x] Update module/runtime options wiring to provide required registry dependencies.
- [x] Update TypeScript surface in `geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`.
- [x] Add JS module tests for:
  - multi-registry stack resolution,
  - provenance payload shape,
  - fingerprint exposure,
  - policy-gated request override behavior.

## Phase 12 — Documentation updates

- [x] Update `geppetto/pkg/doc/topics/01-profiles.md` with stack + multi-registry model and override policy behavior.
- [x] Update JS API docs (`13-js-api-reference.md`, `14-js-api-user-guide.md`) for hard-cut `fromProfile` semantics.
- [x] Add section on overlay removal rationale to avoid future reintroduction confusion.

## Phase 13 — Verification matrix

- [x] Run: `go test ./pkg/profiles`.
- [x] Run: `go test ./pkg/inference/middlewarecfg` (ensure trace model assumptions still hold).
- [x] Run: `go test ./pkg/js/modules/geppetto`.
- [x] Run focused downstream compile/tests where feasible to estimate adaptation blast radius:
  - `pinocchio/cmd/web-chat/...`,
  - `go-go-os/go-inventory-chat/...`.
- [x] Record all command outputs and failures in diary.

## Phase 14 — Downstream adaptation tickets (post-core)

- [x] Create Pinocchio follow-up ticket(s):
  - resolver adoption,
  - runtime composer adoption,
  - profile API behavior verification.
- [x] Create go-go-os follow-up ticket(s):
  - request resolver adoption,
  - runtime composer adoption,
  - integration tests for runtime key/fingerprint behavior.
- [x] Link follow-up tickets back to GP-28 in changelog/index.

## Definition of done for GP-28 implementation

- [x] Geppetto profile domain supports stack refs across registries.
- [x] `ResolveEffectiveProfile` returns deterministic stack-merged runtime with policy-gated request override support.
- [x] Provenance trace follows middlewarecfg-style per-field step history.
- [x] Runtime fingerprint is lineage-aware and cache-safe.
- [x] Overlay abstraction removed from implementation path.
- [x] JS `engines.fromProfile` is hard-cut to registry-backed semantics.
- [x] Non-stack profiles continue to behave correctly.
- [x] Downstream adaptation tickets are created and linked.
