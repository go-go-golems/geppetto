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
    - Path: pkg/profiles/types.go
      Note: |-
        Step 1 profile extensions model field and clone behavior.
        Step 1 profile extensions model field and clone deep-copy behavior
    - Path: pkg/profiles/types_clone_test.go
      Note: |-
        Step 1 clone mutation-isolation coverage for extensions.
        Step 1 extension clone mutation-isolation coverage
    - Path: ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/design-doc/01-implementation-plan-profile-extensions-and-crud.md
      Note: |-
        Step 1 decision to defer registry-level extensions in GP-22.
        Step 1 registry-level extension scope decision
    - Path: ttmp/2026/02/24/GP-22-PROFILE-EXTENSIONS-CRUD--profile-extensions-and-crud/tasks.md
      Note: |-
        Step 1 checklist progress for model/type tasks.
        Step 1 task progress
ExternalSources: []
Summary: Implementation diary for GP-22 profile extension and CRUD rollout, including commit checkpoints, failures, and validation commands.
LastUpdated: 2026-02-24T13:47:00-05:00
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

## Related

- `../tasks.md`
- `../design-doc/01-implementation-plan-profile-extensions-and-crud.md`
