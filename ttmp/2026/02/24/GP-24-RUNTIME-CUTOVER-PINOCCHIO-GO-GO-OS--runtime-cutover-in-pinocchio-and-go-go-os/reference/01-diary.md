---
Title: Diary
Ticket: GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS
Status: active
Topics:
    - architecture
    - backend
    - pinocchio
    - chat
    - migration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/app_owned_chat_integration_test.go
      Note: Profile selection integration tests for next-conversation and in-flight runtime transitions.
    - Path: go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go
      Note: Inventory server integration test for in-flight runtime rebuild on selected profile change.
    - Path: geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/tasks.md
      Note: GP-24 execution checklist synchronized with completed implementation slices.
    - Path: geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/design-doc/01-implementation-plan-runtime-cutover-in-pinocchio-and-go-go-os.md
      Note: Policy decision record for in-flight conversation runtime behavior.
ExternalSources: []
Summary: Diary of GP-24 runtime cutover execution, with profile-selection behavior decisions and verification traces.
LastUpdated: 2026-02-24T15:20:00-05:00
WhatFor: Capture what was implemented, why it was done, and how to re-run verification quickly.
WhenToUse: Use when onboarding or reviewing GP-24 cutover work and profile-selection runtime semantics.
---

# GP-24 Implementation Diary

## 2026-02-24

### Step 1 - Baseline Reconciliation

- Verified ticket sequencing and state: GP-23 closed and GP-24 active in `docmgr`.
- Audited existing GP-24 scope against current code to avoid duplicating already-landed work.
- Confirmed both binaries mount shared profile CRUD handlers via `webhttp.RegisterProfileAPIHandlers(...)`.
- Confirmed route coverage by existing integration tests in:
  - `pinocchio/cmd/web-chat/app_owned_chat_integration_test.go`
  - `go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go`

### Step 2 - In-Flight Profile Policy Test Coverage

- Added Pinocchio integration tests:
  - `TestAppOwnedProfileSelection_InFlightConversation_RebuildsRuntime`
  - `TestAppOwnedProfileSelection_AffectsNextConversationCreation`
- Added Go-Go-OS integration test:
  - `TestProfileE2E_SelectedProfileChange_RebuildsInFlightConversationRuntime`
- Core validated behavior:
  - Profile selection via `POST /api/chat/profile` is not only stored as cookie state; it also affects runtime composition.
  - Reusing the same conversation ID after selection change triggers runtime key transition when runtime fingerprint changes.

### Step 3 - Verification Commands

- Pinocchio:
  - `go test ./cmd/web-chat -run 'TestAppOwnedProfileSelection_(InFlightConversation_RebuildsRuntime|AffectsNextConversationCreation)' -count=1`
- Go-Go-OS:
  - `go test ./go-inventory-chat/cmd/hypercard-inventory-server -run 'TestProfileE2E_SelectedProfileChange_RebuildsInFlightConversationRuntime|TestProfileE2E_ListSelectChat_RuntimeKeyReflectsSelection' -count=1`

All commands passed.

### Step 4 - Documentation + Task Synchronization

- Updated GP-24 task list with completed backend/runtime/profile-selection items.
- Recorded explicit in-flight policy decision in design doc.
- Added changelog entry with decisions, verification commands, and related files.

### Step 5 - Go-Go-OS Selector Server Sync

- Updated `useSetProfile` to call `setCurrentProfile` so selector state follows server-confirmed profile slug rather than optimistic local-only updates.
- Updated `ChatConversationWindow` profile selector callback to invoke async profile writes (`void setProfile(...)`) and preserve current registry context in state.
- Added/ran frontend runtime tests to lock request/decode contract for current-profile API.
- Verified `packages/engine` typecheck passes after hook API change.

### Step 6 - Hard Cutover Alias Removal

- Removed `profile` alias fallback from shared profile API request payloads in Pinocchio webchat HTTP handlers.
- Create-profile and current-profile writes now parse `slug` only, which aligns with hard-cutover API cleanup goals.
- Re-ran pinocchio command/package tests to confirm no regression in profile route behavior or build.

### Step 7 - Compatibility Docs Audit

- Audited maintained docs for stale references to removed profile-registry middleware toggle and alias behavior.
- Verified no active docs in `pinocchio/pkg/doc` or `geppetto/pkg/doc` still instruct users to use removed middleware-switch env vars.
- Marked compatibility cleanup documentation task complete; historical references remain only in archived ticket artifacts under `ttmp/`.

### Step 8 - Pinocchio Frontend Profile Contract + Selection Reconciliation

- Updated `pinocchio/cmd/web-chat/web/src/store/profileApi.ts` so profile list/get/set decoding follows shared API semantics:
  - list supports canonical array and legacy indexed-object payload shapes,
  - current-profile decoding accepts `slug` and legacy `profile` fallback fields,
  - malformed payloads now fail fast in frontend contract decoding.
- Updated `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx` to reconcile selected profile from three sources:
  - local app state,
  - server current-profile response,
  - available profile options.
- Added profile-set failure fallback: when server rejects a profile switch, widget refreshes current profile and re-syncs selection so UI does not remain stale.

New tests added:

- `pinocchio/cmd/web-chat/web/src/store/profileApi.test.ts`
- `pinocchio/cmd/web-chat/web/src/webchat/profileSelection.test.ts`

Validation:

- `npm run typecheck` (in `pinocchio/cmd/web-chat/web`): pass.
- `npx vitest run src/store/profileApi.test.ts src/webchat/profileSelection.test.ts`: pass.

Task impact:

- Completed GP-24 tasks 26, 27, 28, 29.

### Step 9 - Go-Go-OS Selector Behavior Tests

- Added a pure selector-state helper module to make profile selector behavior deterministic and testable outside DOM harnesses.
- Wired `ChatConversationWindow` profile selector through those helpers.
- Added test coverage for:
  - switching `inventory -> default -> inventory`,
  - selecting a profile before any message is sent,
  - fallback to default when current selection is stale.

New files:

- `go-go-os/packages/engine/src/chat/components/profileSelectorState.ts`
- `go-go-os/packages/engine/src/chat/components/profileSelectorState.test.ts`

Updated file:

- `go-go-os/packages/engine/src/chat/components/ChatConversationWindow.tsx`

Validation:

- `pnpm exec vitest run src/chat/components/profileSelectorState.test.ts src/chat/runtime/profileApi.test.ts src/chat/runtime/useProfiles.test.ts`: pass.
- `pnpm exec tsc -b`: pass.

Task impact:

- Completed GP-24 tasks 24 and 25.
