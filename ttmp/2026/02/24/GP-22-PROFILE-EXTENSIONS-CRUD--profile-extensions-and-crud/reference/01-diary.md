---
Title: Diary
Ticket: GP-22-PROFILE-EXTENSIONS-CRUD
Status: active
Topics:
    - architecture
    - backend
    - geppetto
    - pinocchio
    - chat
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/profile_policy_test.go
      Note: |-
        Step 5 extension-aware CRUD endpoint and status mapping tests
        Step 5 CRUD endpoint extension/status regression coverage
    - Path: ../../../../../../../pinocchio/pkg/webchat/http/profile_api.go
      Note: |-
        Step 5 shared CRUD handler DTO and response-shape updates for extensions
        Step 5 extension DTO/request/response and list ordering updates
    - Path: pkg/profiles/codec_yaml_test.go
      Note: |-
        Step 4 YAML codec round-trip and unknown extension preservation tests.
        Step 4 YAML extension roundtrip and unknown key preservation coverage
    - Path: pkg/profiles/extensions.go
      Note: Step 2 extension key parser, typed-key helpers, codec registry, and normalization helpers.
    - Path: pkg/profiles/extensions_test.go
      Note: Step 2 parser/codec/normalization regression coverage.
    - Path: pkg/profiles/file_store_yaml_test.go
      Note: |-
        Step 4 YAML store regression for unknown extension preservation across partial updates.
        Step 4 YAML file-store unknown extension persistence on partial updates
    - Path: pkg/profiles/integration_store_parity_test.go
      Note: |-
        Step 4 cross-backend extension behavior parity tests.
        Step 4 extension behavior parity across backends
    - Path: pkg/profiles/registry.go
      Note: Step 3 profile patch extensions field for service update flows.
    - Path: pkg/profiles/service.go
      Note: Step 2 option plumbing and Step 3 create/update extension normalization.
    - Path: pkg/profiles/service_test.go
      Note: Step 3 normalization and error-field contract tests.
    - Path: pkg/profiles/sqlite_store_test.go
      Note: |-
        Step 4 SQLite extension round-trip and partial-update preservation tests.
        Step 4 SQLite extension roundtrip and partial-update persistence
    - Path: pkg/profiles/types.go
      Note: Step 1 profile extensions model field and clone behavior.
    - Path: pkg/profiles/types_clone_test.go
      Note: Step 1 clone mutation-isolation coverage for extensions.
    - Path: pkg/profiles/validation.go
      Note: Step 3 extension-key syntax and payload serializability validation.
    - Path: pkg/profiles/validation_test.go
      Note: Step 3 extension validation field-path assertions.
    - Path: ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/design-doc/01-implementation-plan-profile-extensions-and-crud.md
      Note: Step 1 scope decision to defer registry-level extensions in GP-22.
    - Path: ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md
      Note: |-
        Step 1-4 checklist progress.
        Step 4 checklist progress
        Step 5 CRUD API contract checklist progress
ExternalSources: []
Summary: Implementation diary for GP-22 profile extension and CRUD rollout, including commit checkpoints, failures, and validation commands.
LastUpdated: 2026-02-24T15:09:00-05:00
WhatFor: Track GP-22 task-by-task implementation details and verification evidence.
WhenToUse: Use when reviewing what landed in GP-22 and how to validate each step.
---







# Diary

## Goal

Capture GP-22 implementation progress in small, auditable steps so contributors can reconstruct exactly what changed and why.

## Context

GP-22 follows GP-21 and adds profile-level extension payload support plus CRUD/API contract hardening across Geppetto, Pinocchio, and Go-Go-OS.

## Step 1: Profile Extensions Model Field and Clone Isolation Baseline

This step started GP-22 execution immediately after closing GP-21. The scope was the first model/types task block: introduce profile `extensions`, ensure clone paths deep-copy extension payloads, and lock behavior with regression tests.

This intentionally keeps registry-level extensions out of scope for now and records that decision in the ticket design doc so downstream tasks remain focused.

### Prompt Context

**User prompt (verbatim):** "close GP-21 and start on GP-22."

**Assistant interpretation:** Close GP-21 now, then begin implementing GP-22 tasks incrementally.

**Inferred user intent:** Keep momentum with disciplined ticket transitions and start delivering GP-22 in small committed slices.

**Commit (code):** `1888ec5` — "profiles: add profile extensions field and clone isolation coverage"

### What I did

- Closed GP-21 via `docmgr ticket close`.
- Added `Extensions map[string]any` field on `Profile` in `pkg/profiles/types.go`.
- Updated `Profile.Clone()` to deep-copy `Extensions` via `deepCopyStringAnyMap`.
- Extended `pkg/profiles/types_clone_test.go` to assert mutation isolation for:
  - direct profile clone extension payloads,
  - profile extension payloads cloned through `ProfileRegistry.Clone()`.
- Documented decision in GP-22 design doc: defer registry-level `extensions` in this ticket.
- Updated GP-22 task checklist first section to complete.
- Ran validation:
  - `go test ./pkg/profiles/... -count=1`,
  - pre-commit full `go test ./...`, lint, vet.

### Why

- Extension payloads are required for app-specific metadata (for example starter suggestions) without adding new core struct fields repeatedly.
- Deep-copy guarantees are mandatory because extension payloads are nested `any` structures that can alias easily.
- Deferring registry-level extensions keeps scope tight and avoids adding unsupported surface area before we have a concrete consumer.

### What worked

- Targeted and full-repo tests passed.
- Clone behavior was simple to wire because existing deep-copy helpers already handled nested maps/lists.

### What didn't work

- No code-level failures in this step.

### What I learned

- Existing clone architecture was already extension-ready; adding one model field and test assertions was enough to preserve isolation invariants.
- The doc/task workflow remains cleaner when design-scope decisions are documented in the same step they are implemented.

### What was tricky to build

- The subtle part was proving `ProfileRegistry.Clone()` behavior specifically for extensions without introducing redundant logic. I validated this by mutating extension payloads on profiles returned from cloned registries and asserting originals stayed unchanged.

### What warrants a second pair of eyes

- Confirm the defer decision for registry-level extensions aligns with expected GP-23/GP-24 requirements.
- Confirm `map[string]any` is the correct immediate wire type for API/store compatibility before typed-key helpers land.

### What should be done in the future

- Implement GP-22 “Extension Key and Codec Infrastructure” tasks next.

### Code review instructions

- Review `pkg/profiles/types.go` for `Profile.Extensions` and clone behavior.
- Review `pkg/profiles/types_clone_test.go` new mutation-isolation assertions.
- Re-run:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto
go test ./pkg/profiles/... -count=1
```

### Technical details

- Extension payload examples used in tests include nested map/list combinations to expose aliasing bugs.
- No serialization code was required in this step because struct tags on `Profile` are sufficient for YAML/JSON persistence paths.

## Step 2: Extension Keys, Typed Accessors, and Codec Registry Infrastructure

This step implemented the full extension-infrastructure layer used by later validation and CRUD tasks. The focus was to add a canonical extension key format, typed access helpers, and codec-based normalization primitives without yet changing API behavior.

Service-level option plumbing for codec registry injection was added in the same step so downstream create/update flow can adopt normalization without constructor churn.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue GP-22 task-by-task with focused commits and diary updates.

**Inferred user intent:** Build extension support incrementally with clear, testable foundations before wiring it into endpoints.

**Commit (code):** `edfb34d` — "profiles: add extension key typed helpers and codec registry plumbing"

### What I did

- Added `pkg/profiles/extensions.go` with:
  - `ExtensionKey` parse/type helpers for `namespace.feature@vN`,
  - panic-free `NewExtensionKey(...)` and panic `MustExtensionKey(...)`,
  - generic `ProfileExtensionKey[T]` with `Get/Set/Decode/Delete`,
  - `ExtensionCodec` and `ExtensionCodecRegistry` interfaces,
  - `InMemoryExtensionCodecRegistry` with duplicate-key guards,
  - `NormalizeProfileExtensions(...)` for canonical-key normalization + codec decode with unknown-key pass-through.
- Updated `pkg/profiles/service.go`:
  - introduced `StoreRegistryOption`,
  - added `WithExtensionCodecRegistry(...)`,
  - changed `NewStoreRegistry(...)` to accept variadic options and wire codec registry.
- Added `pkg/profiles/extensions_test.go` coverage for:
  - extension key parse/constructor behavior,
  - typed key get/set/decode behavior,
  - duplicate codec registration failure,
  - known-key decode success and decode failure,
  - unknown-key pass-through with deep-copy isolation,
  - service option plumbing behavior.
- Marked all tasks complete in GP-22 “Extension Key and Codec Infrastructure” checklist.
- Ran validation:
  - `go test ./pkg/profiles/... -count=1`,
  - pre-commit full `go test ./...`, lint, vet.

### Why

- Typed extension access avoids repetitive map decode logic in every caller.
- Codec registry centralizes normalization and payload validation semantics per extension key.
- Constructor option plumbing enables incremental service integration without breaking existing call sites.

### What worked

- New infrastructure compiled cleanly and passed both targeted and full-repo hooks.
- Variadic `NewStoreRegistry(..., options...)` kept external call sites source-compatible.

### What didn't work

- No code-level failures in this step.

### What I learned

- The typed-key pattern from `pkg/turns` maps cleanly to profile extensions with minimal adaptation.
- Unknown-key preservation with deep-copy is easy to enforce centrally in a single normalization helper.

### What was tricky to build

- The main nuance was keeping canonicalization strict while still preserving unknown keys. The solution was to always parse/canonicalize keys, then apply codec decoding only when a registered codec exists.

### What warrants a second pair of eyes

- Confirm extension-key regex constraints are strict enough for long-term key hygiene but not too restrictive for app teams.
- Confirm decode errors should remain wrapped as `ValidationError` at this layer versus preserving raw codec error types.

### What should be done in the future

- Next step is GP-22 “Validation and Service Flow”: validate extension keys in `ValidateProfile`, enforce serializability at service boundary, and wire normalization into create/update paths.

### Code review instructions

- Start with `pkg/profiles/extensions.go` for parser/types/registry/normalization contracts.
- Review `pkg/profiles/extensions_test.go` for behavior matrix coverage.
- Check `pkg/profiles/service.go` constructor option changes for compatibility impact.
- Re-run:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto
go test ./pkg/profiles/... -count=1
```

### Technical details

- `NormalizeProfileExtensions` canonicalizes keys and deep-copies payloads on output to prevent aliasing.
- Known extension keys are codec-decoded; unknown keys are preserved unchanged (except canonical key string normalization).

## Step 3: Validation and Service-Flow Integration for Extensions

This step wired extension validation and codec normalization directly into profile create/update service flows. The scope covered key syntax validation, JSON-serializability checks, patch plumbing for `extensions`, and regression tests for error field paths.

The result is that malformed extension keys and non-serializable payloads are rejected as typed validation errors before persistence, and extension payloads are normalized consistently through the service boundary.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue executing GP-22 tasks in sequence and commit each completed slice.

**Inferred user intent:** Move from extension infrastructure to enforceable runtime behavior with strong validation guarantees.

**Commit (code):** `440fb4f` — "profiles: validate and normalize extensions in service create/update flows"

### What I did

- Updated `pkg/profiles/validation.go`:
  - added `ValidateProfileExtensions(...)`,
  - enforced extension key syntax via `ParseExtensionKey`,
  - enforced JSON-serializable extension payloads with field-specific `ValidationError`.
- Updated `pkg/profiles/registry.go`:
  - added `Extensions *map[string]any` to `ProfilePatch`.
- Updated `pkg/profiles/service.go`:
  - added `normalizeAndValidateProfile(...)` helper,
  - wired extension normalization + validation into `CreateProfile`,
  - wired patch-extension update + normalization into `UpdateProfile`.
- Added/extended tests:
  - `pkg/profiles/validation_test.go` for extension error field paths,
  - `pkg/profiles/service_test.go` for create/update normalization, codec decode failure handling, unknown pass-through, and invalid-key update errors.
- Marked GP-22 “Validation and Service Flow” checklist complete in `tasks.md`.
- Ran validation:
  - `go test ./pkg/profiles/... -count=1`,
  - pre-commit full `go test ./...`, lint, vet.

### Why

- Validation needs to happen before write paths so malformed extension payloads never persist.
- Service-boundary normalization ensures payload handling is consistent regardless of caller path.
- Explicit field-path errors are required for predictable HTTP/API mapping in downstream tickets.

### What worked

- All new tests passed with targeted and full pre-commit checks.
- Existing typed error handling (`ErrValidation`, `ErrPolicyViolation`, `ErrVersionConflict`) remained stable.

### What didn't work

- No code-level failures in this step.

### What I learned

- Extension normalization fits cleanly as a pre-validation step and avoids duplicated logic across create/update methods.
- Field-path assertions in tests are essential because map-key input errors are otherwise easy to regress silently.

### What was tricky to build

- The subtle part was preserving update semantics while adding extension patch support. The approach was to introduce `ProfilePatch.Extensions` as an optional pointer map, apply it only when present, and then run one shared normalize/validate helper for final consistency.

### What warrants a second pair of eyes

- Confirm `ProfilePatch.Extensions` semantics are correct for “clear all extensions” versus “no change” behavior in API layers.
- Confirm JSON-serializability checks are sufficient for downstream storage/runtime expectations.

### What should be done in the future

- Next step is GP-22 “Persistence and Round-Trip” coverage for YAML/SQLite and cross-backend parity with extension payloads.

### Code review instructions

- Start with `pkg/profiles/validation.go` and `pkg/profiles/service.go` integration points.
- Review `pkg/profiles/service_test.go` new extension-flow tests and `pkg/profiles/validation_test.go` field-path assertions.
- Re-run:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto
go test ./pkg/profiles/... -count=1
```

### Technical details

- Service create/update flows now normalize extension keys and codec-decode known keys before validation and persistence.
- Invalid extension keys and non-serializable values now surface as `ValidationError` with `profile.extensions[...]` field paths.

## Step 4: Persistence and Round-Trip Coverage for Extension Payloads

This step completed the persistence matrix for extension payload behavior across YAML and SQLite stores, plus backend parity checks. The work focused on ensuring extension payloads survive encode/decode, reload, and partial profile update flows without key drift or data loss.

The tests explicitly cover unknown extension keys because forward-compatible pass-through is a core requirement for mixed-version deployments.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue GP-22 execution and complete the next checklist block with commit checkpoints.

**Inferred user intent:** Prove that extension payload guarantees remain stable across all persistence backends before API/client integration.

**Commit (code):** `09bc4ca` — "profiles: add extension persistence round-trip coverage across backends"

### What I did

- Extended `pkg/profiles/codec_yaml_test.go`:
  - added extension payload assertions to YAML encode/decode roundtrip,
  - added regression test for preserving unknown extension keys.
- Extended `pkg/profiles/file_store_yaml_test.go`:
  - added service-driven partial update regression ensuring unknown extension payload survives YAML reload.
- Extended `pkg/profiles/sqlite_store_test.go`:
  - added SQLite extension roundtrip test,
  - added partial-update regression ensuring unknown extension payload survives reopen.
- Extended `pkg/profiles/integration_store_parity_test.go`:
  - added cross-backend parity test for extension behavior across memory/YAML/SQLite.
- Marked all tasks complete in GP-22 “Persistence and Round-Trip” checklist.
- Ran validation:
  - `go test ./pkg/profiles/... -count=1`,
  - pre-commit full `go test ./...`, lint, vet.

### Why

- Persistence behavior is the contract boundary used by web-chat and Go-Go-OS. Any extension data loss or key mutation here would break profile-scoped features.
- Unknown-key retention is required so third-party and future extensions can coexist safely.

### What worked

- All new tests passed in targeted and full pre-commit pipelines.
- Extension key canonicalization and unknown-payload pass-through stayed consistent across all backends.

### What didn't work

- No code-level failures in this step.

### What I learned

- Service-level partial updates are a useful regression lens for persistence because they emulate real API usage better than direct store-only tests.
- Backend parity tests keep expectations aligned and reduce drift risk between adapters.

### What was tricky to build

- The tricky part was asserting parity without relying on backend-specific value concrete types. I focused assertions on canonical keys and nested payload content semantics rather than strict concrete Go types.

### What warrants a second pair of eyes

- Confirm parity assertions are strict enough for downstream API contract confidence, especially around nested `any` payload values.
- Confirm no additional reopen/close lifecycle variants are needed for extension-heavy payloads.

### What should be done in the future

- Next step is GP-22 CRUD API contract implementation in Pinocchio webchat handlers and DTOs (`extensions` fields and status-code shape checks).

### Code review instructions

- Start with `pkg/profiles/codec_yaml_test.go`, `pkg/profiles/file_store_yaml_test.go`, and `pkg/profiles/sqlite_store_test.go`.
- Review `pkg/profiles/integration_store_parity_test.go` extension parity test.
- Re-run:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto
go test ./pkg/profiles/... -count=1
```

### Technical details

- Added extension regression coverage for:
  - YAML encode/decode registry codec layer,
  - YAML file store reload after service partial update,
  - SQLite reopen durability and partial-update preservation,
  - backend parity across memory/YAML/SQLite service flows.

## Step 5: CRUD API Contract Extensions in Pinocchio Webchat

This step applied the extension-aware CRUD contract at the shared webchat HTTP API layer in Pinocchio. DTOs and request payloads now carry `extensions`, list output is explicitly sorted by slug in handler code, and CRUD endpoint tests validate extension behavior end-to-end.

The focus here was API-level contract stability: response shape, status-code mappings, and extension field presence across list/get/create/patch flows.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue GP-22 into API contract tasks and commit the next completed slice.

**Inferred user intent:** Ensure extension support is usable through the real HTTP interface consumed by webchat and downstream clients.

**Commit (code):** `834fa5c` — "webchat: include profile extensions in CRUD API contract"

### What I did

- Updated `/pinocchio/pkg/webchat/http/profile_api.go`:
  - added `extensions` to `ProfileListItem`, `ProfileDocument`, `CreateProfileRequest`, and `PatchProfileRequest`,
  - passed create/patch extension payloads into geppetto profile service calls,
  - added handler-level slug sorting in list endpoint for deterministic order,
  - added extension payload projection in list/get/create/patch responses.
- Updated `/pinocchio/cmd/web-chat/profile_policy_test.go`:
  - extended CRUD lifecycle test to include extension payload create/get/patch assertions,
  - added list-response ordering assertion,
  - added error/status checks for invalid extension payload (`400`) and missing registry (`404`) while keeping conflict check (`409`).
- Verified Pinocchio package tests:
  - `go test ./cmd/web-chat -count=1`,
  - plus full pre-commit pipeline during commit.

### Why

- Clients cannot consume extension metadata if API DTOs omit it.
- Deterministic list ordering should be enforced at the API boundary, not only assumed from backend implementations.
- Extension-specific validation/status checks are required to lock the HTTP contract.

### What worked

- Handler and tests compiled cleanly with no extra migration layer.
- Existing error mapping logic (`ErrValidation` => `400`, `ErrRegistryNotFound/ErrProfileNotFound` => `404`, `ErrVersionConflict` => `409`) remained intact and now has explicit regression coverage for extension paths.

### What didn't work

- No code-level failures in this step.

### What I learned

- The webchat handler already had a strong structure for extending DTOs; adding extension support was mostly straightforward field plumbing.
- API-level deterministic sorting is cheap and removes hidden dependencies on store ordering behavior.

### What was tricky to build

- The main subtlety was keeping extension payload output safe without exposing shared mutable maps. The handler uses map cloning before writing JSON responses to avoid accidental aliasing.

### What warrants a second pair of eyes

- Confirm handler-level map cloning strategy is acceptable long-term for large extension payloads.
- Confirm no additional API consumers rely on previous unsorted list order.

### What should be done in the future

- Next step is GP-22 shared-route reuse and Go-Go-OS integration tasks, then frontend client typing/runtime decoder updates.

### Code review instructions

- Review `/pinocchio/pkg/webchat/http/profile_api.go` first for DTO/request/response updates and list sort behavior.
- Review `/pinocchio/cmd/web-chat/profile_policy_test.go` for extension-aware CRUD and status assertions.
- Re-run:

```bash
cd /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio
go test ./cmd/web-chat -count=1
```

### Technical details

- `extensions` now flows through:
  - `POST /api/chat/profiles` request and response,
  - `PATCH /api/chat/profiles/{slug}` request and response,
  - `GET /api/chat/profiles` list items,
  - `GET /api/chat/profiles/{slug}` profile document.
- List endpoint enforces deterministic ordering via `sort.Slice` by profile slug before encoding JSON response.

## Related

- `../tasks.md`
- `../design-doc/01-implementation-plan-profile-extensions-and-crud.md`
