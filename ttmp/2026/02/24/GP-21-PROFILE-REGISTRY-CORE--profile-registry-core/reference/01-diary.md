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
    - Path: pkg/profiles/file_store_yaml_test.go
      Note: Step 3 YAML store robustness coverage
    - Path: pkg/profiles/integration_store_parity_test.go
      Note: Step 5 cross-backend lifecycle parity baseline coverage
    - Path: pkg/profiles/memory_store_test.go
      Note: Step 5 metadata/version attribution and monotonicity coverage
    - Path: pkg/profiles/service_test.go
      Note: Step 2 service semantics edge-case coverage
    - Path: pkg/profiles/slugs_test.go
      Note: |-
        Text marshal/unmarshal slug round-trip and invalid-input coverage.
        Step 1 text marshal/unmarshal regression coverage
    - Path: pkg/profiles/sqlite_store_test.go
      Note: Step 4 sqlite store robustness coverage
    - Path: pkg/profiles/types_clone_test.go
      Note: |-
        Clone mutation-isolation coverage for profile and registry models.
        Step 1 clone invariants and aliasing regression coverage
    - Path: pkg/profiles/validation_test.go
      Note: |-
        Validation field-path assertions and ErrValidation consistency checks.
        Step 1 validation field-path and error-type contract coverage
    - Path: ttmp/2026/02/24/GP-21-PROFILE-REGISTRY-CORE--profile-registry-core/reference/02-core-behavior-matrix.md
      Note: Step 5 behavior matrix contract for downstream tickets
    - Path: ttmp/2026/02/24/GP-21-PROFILE-REGISTRY-CORE--profile-registry-core/tasks.md
      Note: |-
        GP-21 checklist progress updates for completed model/validation tasks.
        Step 1 task checklist updates
        Step 2 checklist progress
        Step 3 YAML checklist completion
        Step 4 sqlite checklist completion
ExternalSources: []
Summary: Implementation diary for GP-21 profile registry core work, including commit-level changes, validation outcomes, and review guidance.
LastUpdated: 2026-02-24T13:36:26-05:00
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

## Step 3: YAML Store Robustness Coverage

This step completed the YAML persistence checklist for GP-21. The tests now exercise file lifecycle behavior, malformed input handling, multi-registry reload parity, atomic temp-file rename behavior, and post-close guards.

The focus was to move YAML store confidence from basic happy-path persistence to explicit failure and lifecycle semantics.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue executing GP-21 tasks sequentially, committing each completed task group and recording progress in a detailed diary.

**Inferred user intent:** Ensure persistence behaviors are fully test-proven before downstream integration relies on them.

**Commit (code):** `5d825aeea7acafd342fc77a1c2fe74dd483d2afb` — "profiles: harden yaml store coverage for GP-21 step 3"

### What I did

- Extended `geppetto/pkg/profiles/file_store_yaml_test.go` with new tests:
  - `TestYAMLFileProfileStore_MissingFileInitialization`
  - `TestYAMLFileProfileStore_ParseFailureSurfacing`
  - `TestYAMLFileProfileStore_WriteThenReloadParity_MultipleRegistries`
  - `TestYAMLFileProfileStore_AtomicTempRenameBehavior`
  - `TestYAMLFileProfileStore_CloseStateGuards`
- Updated GP-21 task checklist to mark all `Persistence: YAML Store` tasks completed.
- Ran `go test ./pkg/profiles/... -count=1`, then full pre-commit test/lint pipeline via lefthook.

### Why

- YAML store is a primary operator-facing persistence path; missing edge-case coverage can produce silent data-loss risks or confusing runtime errors.
- Lifecycle tests (`missing file`, `closed store`) prevent regressions in initialization and shutdown paths.

### What worked

- All added YAML tests passed.
- Final file decode checks confirmed post-write readability and no `.tmp` residue.
- Multi-registry parity test validated durable write/reload behavior beyond a single default registry.

### What didn't work

- No code-level failures occurred in this step.

### What I learned

- Existing YAML store implementation already had robust behavior; the main gap was explicit coverage for edge cases and lifecycle guards.
- `persistLocked` temp-file + rename strategy is verifiable in tests by asserting no lingering `.tmp` and successful decode of final file.

### What was tricky to build

- The tricky part was writing assertions that test atomic write intent without race-prone filesystem assumptions.
- I used deterministic post-condition checks (`.tmp` does not exist, final file decodes) instead of timing-sensitive checks.

### What warrants a second pair of eyes

- Review whether parse-failure assertions should enforce stricter error-shape guarantees (currently checks for yaml context, not exact message).
- Review if additional corruption scenarios (partial file truncation) should be simulated in future.

### What should be done in the future

- Next: finish SQLite persistence robustness tests, then metadata/version invariants, then integration parity baseline.

### Code review instructions

- Start with new tests in `geppetto/pkg/profiles/file_store_yaml_test.go`.
- Verify each new test maps 1:1 to GP-21 YAML checklist items.
- Re-run:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto
go test ./pkg/profiles/... -count=1
```

### Technical details

- Added close-guard assertions across all public YAML store methods to ensure `ensureOpen` contract remains enforced.
- Added reload parity checks across two registries (`default`, `team`) to confirm cross-registry serialization integrity.

## Step 4: SQLite Store Robustness Coverage

This step completed the SQLite persistence checklist for GP-21. I added tests for migration idempotency, malformed payload and slug-mismatch detection at load time, CRUD persistence across reopen cycles, delete-persistence behavior, and post-close method guards.

The scope intentionally mirrored the YAML robustness matrix so behavior guarantees remain consistent across store backends.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue task-by-task execution with commit-level checkpoints and detailed diary updates.

**Inferred user intent:** Verify SQLite durability and error handling before relying on registry storage in production workflows.

**Commit (code):** `160c8b785990b1503f19c25feb2b6985072892fe` — "profiles: expand sqlite store resilience tests for GP-21 step 4"

### What I did

- Extended `geppetto/pkg/profiles/sqlite_store_test.go` with tests for:
  - migration idempotency (`store.migrate()` repeated calls),
  - malformed JSON payload row handling,
  - row slug vs payload slug mismatch rejection,
  - CRUD persistence after reopen,
  - delete profile persistence updating registry row/default slug state,
  - close idempotency + post-close operation guard checks across all public methods.
- Updated GP-21 checklist to mark all `Persistence: SQLite Store` tasks complete.

### Why

- SQLite is the durable backend used for production profile registries; load-time corruption checks and reopen parity are critical.
- Close-state guard behavior needs explicit assertions to avoid hidden nil-pointer/closed-db runtime failures.

### What worked

- All new SQLite tests passed after one assertion refinement.
- Full pre-commit pipeline passed (`go test ./...`, lint, vet, generate, build).
- SQLite task section is now fully complete in ticket checklist.

### What didn't work

- Initial malformed-payload assertion expected `"invalid"` substring only, but runtime error surfaced as `"unexpected end of JSON input"`.
- Assertion was broadened to accept equivalent JSON parse-failure indicators (`json`, `unexpected`, or `invalid`) while still enforcing failure clarity.

### What I learned

- Defensive load behavior in `loadFromDB` is strong and catches both payload malformed JSON and row/payload slug mismatches.
- Delete-profile persistence semantics include clearing default profile slug when deleting the current default, and this now survives reopen checks.

### What was tricky to build

- Crafting realistic malformed-row scenarios required bypassing normal service/store write paths and inserting raw rows directly using `database/sql`.
- The challenge was ensuring failure-mode tests stayed deterministic without overfitting to one exact parser error string.

### What warrants a second pair of eyes

- Review whether malformed-payload error assertions should remain flexible or if we want stricter wrapped error contracts in store implementation.
- Verify that direct `store.migrate()` coverage is acceptable as an internal-behavior test.

### What should be done in the future

- Next: metadata/versioning invariants and integration baseline parity tests across memory/YAML/SQLite service behavior.

### Code review instructions

- Review new SQLite tests in `geppetto/pkg/profiles/sqlite_store_test.go`.
- Check that each test maps directly to one checklist item in the SQLite section.
- Validate with:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto
go test ./pkg/profiles/... -count=1
```

### Technical details

- Added raw-row fixture setup with `database/sql` for malformed/mismatch load-path tests.
- Added close-guard matrix to validate all public `SQLiteProfileStore` methods fail cleanly after close.

## Step 5: Metadata/Version Contracts and Cross-Backend Parity Baseline

This step completed the remaining GP-21 checklist: metadata/version invariants and integration baseline parity across in-memory, YAML, and SQLite stores.

I added focused metadata attribution/version tests in memory store coverage and introduced a cross-backend lifecycle parity test that runs one canonical CRUD flow against all three stores. I also added a standalone behavior matrix reference note for downstream tickets.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the remaining GP-21 tasks sequentially, with commits and detailed diary continuity.

**Inferred user intent:** Leave GP-21 in a fully completed state with both executable tests and concise downstream contract documentation.

**Commit (code):** `fcbf87593f1557675829fe0d1c7578eb56f8f4a8` — "profiles: add metadata and cross-backend parity coverage for GP-21 step 5"

### What I did

- Extended `geppetto/pkg/profiles/memory_store_test.go` with:
  - registry metadata/version mutation coverage (version increments, actor/source propagation, created/updated timestamp behavior),
  - profile metadata/version mutation coverage with explicit created/updated attribution checks.
- Added `geppetto/pkg/profiles/integration_store_parity_test.go`:
  - canonical lifecycle flow (`create -> update -> set-default -> delete`) run for each backend (memory/YAML/SQLite),
  - identical end-state assertions for all backends.
- Created behavior matrix reference doc:
  - `.../reference/02-core-behavior-matrix.md`,
  - summarizes guaranteed invariants and their test evidence.
- Marked all remaining GP-21 tasks complete in `tasks.md`.

### Why

- Metadata/version contracts are critical for conflict handling, auditability, and API-level debugging.
- Parity tests ensure backend choice does not alter service-level behavior.
- The behavior matrix provides a compact contract for GP-22..GP-25 implementation decisions.

### What worked

- New tests passed locally and under full pre-commit pipeline.
- Task list reached 100% completion.
- Behavior matrix doc gives a one-page downstream reference of guarantees + source tests.

### What didn't work

- No failing iterations occurred in this step.

### What I learned

- Metadata propagation semantics are deterministic and testable without mocking time by asserting immutability and monotonic non-decrease.
- A single well-defined lifecycle parity test provides high confidence across storage adapters with minimal duplication.

### What was tricky to build

- The subtle part was choosing a lifecycle flow that remains valid under current registry invariants while still covering create/update/set-default/delete operations.
- I used a flow that deletes the non-default profile after switching default to avoid producing invalid registry states in parity assertions.

### What warrants a second pair of eyes

- Review whether parity test should expand to include reopen checks within the same flow for all backends.
- Review whether registry metadata source-overwrite behavior on each mutation is desired long-term or should be additive.

### What should be done in the future

- GP-21 implementation tasks are complete.
- Next action is ticket close or targeted follow-up only if new core invariants are requested.

### Code review instructions

- Start with `geppetto/pkg/profiles/memory_store_test.go` new metadata/version sections.
- Review `geppetto/pkg/profiles/integration_store_parity_test.go` for lifecycle parity logic.
- Read `.../reference/02-core-behavior-matrix.md` for the compact invariant contract.
- Validate:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto
go test ./pkg/profiles/... -count=1
```

### Technical details

- Metadata assertions verify:
  - version increments on each mutation,
  - `CreatedAtMs` immutability,
  - `UpdatedAtMs` monotonicity,
  - `CreatedBy` stickiness and `UpdatedBy` refresh,
  - source propagation behavior.
- Parity assertions verify stable final state across memory/YAML/SQLite with identical service calls.

## Usage Examples

Use this diary to review GP-21 execution:

1. Read the latest step section.
2. Inspect referenced commit hash.
3. Run listed validation commands.
4. Continue from “What should be done in the future”.

## Related

- `../tasks.md`
- `../design-doc/01-implementation-plan-profile-registry-core.md`
