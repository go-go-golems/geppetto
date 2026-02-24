---
Title: Diary
Ticket: GP-21-PROFILE-REGISTRY-CORE
Status: active
Topics:
    - architecture
    - backend
    - geppetto
    - persistence
    - migration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/profiles/service_test.go
      Note: Step 2 service semantics edge-case coverage
    - Path: pkg/profiles/slugs_test.go
      Note: |-
        Text marshal/unmarshal slug round-trip and invalid-input coverage.
        Step 1 text marshal/unmarshal regression coverage
    - Path: pkg/profiles/types_clone_test.go
      Note: |-
        Clone mutation-isolation coverage for profile and registry models.
        Step 1 clone invariants and aliasing regression coverage
    - Path: pkg/profiles/validation_test.go
      Note: |-
        Validation field-path assertions and ErrValidation consistency checks.
        Step 1 validation field-path and error-type contract coverage
    - Path: ttmp/2026/02/24/GP-21-PROFILE-REGISTRY-CORE--profile-registry-core/tasks.md
      Note: |-
        GP-21 checklist progress updates for completed model/validation tasks.
        Step 1 task checklist updates
        Step 2 checklist progress
ExternalSources: []
Summary: Implementation diary for GP-21 profile registry core work, including commit-level changes, validation outcomes, and review guidance.
LastUpdated: 2026-02-24T13:27:54-05:00
WhatFor: Track each implementation step for GP-21 with exact commands, outcomes, and follow-ups.
WhenToUse: Use when reviewing what was implemented for GP-21 and how to reproduce verification.
---



# Diary

## Goal

Capture GP-21 implementation progress step-by-step so a reviewer or intern can reconstruct exactly what changed, why it changed, and how it was validated.

## Context

GP-21 is the “Profile Registry Core” execution ticket. The task list was seeded with broad coverage requirements around clone correctness, validation boundaries, persistence semantics, and metadata/version behavior. This diary records each execution chunk with commit hashes.

## Quick Reference

## Step 1: Close GP-20 Umbrella and Implement Model/Validation Coverage Baseline

This step formalized the planning split by closing GP-20 and starting GP-21 execution. The implementation focus was the first two checklist groups: model/type clone safety and validation error semantics.

I added regression tests that enforce deep-copy behavior for mutable nested payloads and field-specific validation error contracts. This establishes a reliable baseline before moving into service and persistence semantics.

### Prompt Context

**User prompt (verbatim):** "Ok, close GP-20 I guess, and let's work on GP-21 by going through the tasks one by one , committing appropriately and keeping a detailed diary."

**Assistant interpretation:** Close GP-20 immediately, then execute GP-21 incrementally with small commits and continuous diary updates.

**Inferred user intent:** Move from planning to disciplined execution with auditable progress and commit-level traceability.

**Commit (code):** `1e72f9c5fa19ea6be1952c1d975458d650c5861b` — "profiles: add clone and validation regression coverage for GP-21 step 1"

### What I did

- Closed GP-20 via `docmgr ticket close --ticket GP-20-PROFILE-REGISTRY-EXTENSIONS ...`.
- Created the GP-21 diary document scaffold using `docmgr doc add`.
- Added `geppetto/pkg/profiles/types_clone_test.go` with:
  - profile clone deep-copy aliasing checks,
  - registry clone deep-copy aliasing checks,
  - nested map/list middleware config mutation isolation checks.
- Extended `geppetto/pkg/profiles/slugs_test.go` with text marshal/unmarshal coverage and invalid-text rejection checks.
- Extended `geppetto/pkg/profiles/validation_test.go` with:
  - explicit field-path assertions for empty slugs/default requirements/nil entries/key mismatch,
  - whitespace middleware/tool rejection checks,
  - shared helper asserting `errors.Is(err, ErrValidation)` and `*ValidationError` field correctness.
- Marked completed GP-21 checklist items in `.../GP-21.../tasks.md` for model/types and validation sections.
- Ran targeted package tests and pre-commit full repo checks triggered by lefthook.

### Why

- Clone safety regressions are high-impact because profile runtime payloads contain nested `map[string]any` and `[]any` values that are easy to alias accidentally.
- Validation error contract tests (field path + typed error) are required to keep API-layer error mapping deterministic.
- Closing GP-20 prevents ambiguity about which ticket is execution owner.

### What worked

- New clone and validation tests passed immediately without requiring production code changes.
- Full pre-commit pipeline (`go test ./...`, lint, generate, build, vet) passed with the new tests.
- Task checklist now reflects objective completion for the first two sections.

### What didn't work

- Initial frontmatter validation command used a relative path that accidentally doubled `ttmp/` in the path and failed with “no such file or directory”. Re-ran using absolute paths and validation succeeded.

### What I learned

- Existing core implementation was already stronger than the checklist suggested; missing work was mostly explicit regression coverage.
- The clone implementation already deep-copied middleware `Config` payloads correctly; the gap was proof, not behavior.
- Validation methods already returned typed errors, but field-level assertions were under-tested.

### What was tricky to build

- The main sharp edge was asserting deep-copy behavior for nested `any` payloads without writing brittle tests. The approach was to mutate cloned nested maps/lists and assert original payload slices/maps did not change, including nested elements.
- Another subtlety was avoiding false confidence from only checking top-level pointer inequality; tests were written to mutate nested structures directly to expose aliasing.

### What warrants a second pair of eyes

- Review whether the invariants in `types_clone_test.go` are sufficiently exhaustive for all mutable fields in future model additions.
- Confirm validation field-path strings are stable contracts expected by upstream API handlers.

### What should be done in the future

- Continue with GP-21 “Service Semantics” tasks next, especially explicit resolve fallback and read-only mutation policy tests.

### Code review instructions

- Start with `geppetto/pkg/profiles/types_clone_test.go` for clone invariants and aliasing checks.
- Then review `geppetto/pkg/profiles/validation_test.go` for field-path and error-type contract assertions.
- Finally review `geppetto/pkg/profiles/slugs_test.go` additions for text codec coverage.

Validation commands:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto
go test ./pkg/profiles/... -count=1
```

Pre-commit pipeline that executed during commit:

```bash
go test ./...
go generate ./...
go build ./...
golangci-lint run -v --max-same-issues=100
go vet -vettool=/tmp/geppetto-lint ./...
```

### Technical details

- Added tests for `MarshalText/UnmarshalText` paths on `RegistrySlug`, `ProfileSlug`, and `RuntimeKey`.
- Added validation helper `requireValidationField(...)` to enforce:
  - `errors.Is(err, ErrValidation)`,
  - concrete `*ValidationError` type,
  - expected `Field` path.

## Step 2: Service Semantics Regression Coverage

This step targeted the GP-21 service semantics checklist. I added explicit coverage for default/fallback profile resolution behavior, policy-denied mutations, optimistic-locking conflicts, and deterministic registry summary ordering.

The tests were written against `StoreRegistry` APIs to keep coverage at service boundary level rather than store-internal behavior only.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue GP-21 incrementally with one task group at a time, commit each chunk, and keep detailed diary records.

**Inferred user intent:** Ensure service-level profile behavior is stable and explicitly tested before persistence/integration tasks.

**Commit (code):** `04c51203a7e8838b63de45d6b76af2e8efb4c0d7` — "profiles: cover service semantics edge cases for GP-21 step 2"

### What I did

- Added service tests in `geppetto/pkg/profiles/service_test.go` for:
  - empty profile request + no default slug + explicit `default` profile fallback,
  - empty profile request + no fallback available -> validation error with field assertion,
  - update of read-only profile -> `ErrPolicyViolation`,
  - delete of read-only profile -> `ErrPolicyViolation`,
  - update with stale expected version -> `ErrVersionConflict`,
  - deterministic `ListRegistries` ordering by slug.
- Updated GP-21 task checklist to mark all `Service Semantics` tasks complete.

### Why

- Service behavior is where API and UI expectations converge. Missing edge-case tests here can cause runtime profile selection bugs even if store logic is correct.
- Explicit ordering and policy/conflict tests prevent regressions in list endpoints and write-path safety.

### What worked

- New service tests passed after adjustments.
- Full repository pre-commit pipeline (`go test ./...`, lint, vet, generate, build) passed on commit.
- Service semantics checklist section is now fully checked.

### What didn't work

- Initial fallback tests attempted to create registries without `default_profile_slug` through normal upsert paths; validation rejected this as expected.
- To test fallback logic paths that handle legacy/invalid in-memory states, tests were updated to inject intentionally invalid registry state directly into in-memory store internals (`store.registries[...] = ...`) inside package-level tests.

### What I learned

- `resolveProfileSlugForRegistry` contains fallback behavior that is not reachable through strict validation/write paths, but is still useful to guard for defensive compatibility.
- Policy and version conflict handling in `UpdateProfile`/`DeleteProfile` is straightforward and now explicitly asserted.

### What was tricky to build

- The tricky part was testing fallback branches that are logically valid but structurally blocked by current validation rules.
- The solution was to avoid weakening validation and instead construct controlled invalid state directly in tests within the same package, so fallback code remains tested without changing production contracts.

### What warrants a second pair of eyes

- Confirm that testing fallback via injected invalid state is acceptable long-term for defensive branches.
- Verify whether fallback-to-`default` branch should eventually be removed if validation invariants are guaranteed at all boundaries.

### What should be done in the future

- Next: Persistence coverage tasks (YAML and SQLite robustness tests), then metadata/version invariants and integration parity baseline.

### Code review instructions

- Start with `geppetto/pkg/profiles/service_test.go` new tests around fallback/policy/conflict/ordering.
- Review the two injected invalid-state tests and ensure they reflect intended defensive semantics.
- Re-run:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto
go test ./pkg/profiles/... -count=1
```

### Technical details

- Added explicit `ValidationError.Field` assertion for empty-profile/no-default path.
- Added deterministic ordering assertion for `ListRegistries` summaries.
- Added policy and version-conflict checks at service API level (not just store-level tests).

## Usage Examples

Use this diary to review GP-21 execution:

1. Read the latest step section.
2. Inspect referenced commit hash.
3. Run listed validation commands.
4. Continue from “What should be done in the future”.

## Related

- `../tasks.md`
- `../design-doc/01-implementation-plan-profile-registry-core.md`
