---
Title: Cross-App Profile API and Runtime Parity Report
Ticket: GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS
Status: active
Topics:
    - architecture
    - backend
    - pinocchio
    - chat
    - migration
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/app_owned_chat_integration_test.go
      Note: Pinocchio integration coverage for profile CRUD and runtime-switch behavior.
    - Path: go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go
      Note: Go-Go-OS integration coverage for profile CRUD and runtime-switch behavior.
    - Path: geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/sources/01-profile-parity-fixture.yaml
      Note: Shared fixture specification used as parity reference model.
ExternalSources: []
Summary: Contract parity report for profile CRUD/current-profile/runtime-switch behavior between Pinocchio and Go-Go-OS.
LastUpdated: 2026-02-24T15:44:40.704035915-05:00
WhatFor: Provide concrete parity evidence for GP-24 cross-app endpoint/runtime behavior.
WhenToUse: Use when validating that both apps expose compatible profile APIs and runtime-switch semantics.
---

# Cross-App Profile API and Runtime Parity Report

## Scope

This report validates GP-24 parity tasks across:

- Pinocchio `cmd/web-chat`,
- Go-Go-OS `hypercard-inventory-server`.

Validated surfaces:

1. profile CRUD endpoints,
2. current-profile selection endpoint,
3. runtime marker behavior after selection and in-flight conversation switches.

## Parity Fixture

Reference fixture definition:

- `sources/01-profile-parity-fixture.yaml`

The fixture captures a canonical three-profile setup (`default`, `inventory`, `analyst`) and is used as the semantic baseline for parity checks, even where app-local test fixtures use different concrete slugs.

## Verification Commands

Executed:

```bash
cd pinocchio
go test ./cmd/web-chat -run 'TestProfileAPI_CRUDRoutesAreMounted|TestAppOwnedProfileSelection_InFlightConversation_RebuildsRuntime|TestAppOwnedProfileSelection_AffectsNextConversationCreation' -count=1

cd go-go-os
go test ./go-inventory-chat/cmd/hypercard-inventory-server -run 'TestProfileAPI_CRUDRoutesAreMounted|TestProfileE2E_ListSelectChat_RuntimeKeyReflectsSelection|TestProfileE2E_SelectedProfileChange_RebuildsInFlightConversationRuntime' -count=1
```

Result: both command groups passed.

## Findings

### A) CRUD surface parity

Both apps cover:

- `GET /api/chat/profiles`,
- `POST /api/chat/profiles`,
- `GET /api/chat/profiles/{slug}`,
- `PATCH /api/chat/profiles/{slug}`,
- `POST /api/chat/profiles/{slug}/default`,
- `DELETE /api/chat/profiles/{slug}`.

Observed parity:

- status codes align (`200/201/204/404` paths),
- response contract keys align with shared handler (`registry`, `slug`, `runtime`, `metadata`, `extensions`, `is_default` for documents; list item contract for list).

### B) Current-profile selection parity

Both apps validate:

- `POST /api/chat/profile` writes selected profile,
- subsequent chat request in same session/cookie context uses selected runtime.

### C) Runtime marker parity after selection/switch

Both apps validate:

- next-conversation runtime follows selected profile,
- in-flight conversation runtime rebuild occurs when selected profile changes and same `conv_id` is reused.

Note on marker format:

- Pinocchio tests assert runtime marker like `default` / `agent`.
- Go-Go-OS tests assert runtime marker like `inventory@v0` / `analyst@v0`.

Normalized comparison rule (strip optional `@version` suffix) shows semantic parity: marker base slug matches selected profile in both apps.

## Task Closure Mapping

- Task 34: parity fixture added (`sources/01-profile-parity-fixture.yaml`).
- Task 35: CRUD parity validated via targeted integration suites in both apps.
- Task 36: current-profile selection parity validated via targeted integration suites in both apps.
- Task 37: post-switch runtime marker parity validated via targeted integration suites in both apps.
- Task 38: this report and changelog update provide captured parity evidence.

## Residual Gaps

- This report relies on integration suites rather than a single cross-process harness executing both servers in one script.
- Runtime marker formatting differs (`slug` vs `slug@vN`) but normalized semantics are equivalent.
