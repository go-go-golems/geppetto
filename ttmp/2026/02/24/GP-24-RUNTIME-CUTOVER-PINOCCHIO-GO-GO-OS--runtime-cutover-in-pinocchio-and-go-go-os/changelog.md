# Changelog

## 2026-02-24

- Initial workspace created


## 2026-02-24

Populated ticket with cross-application runtime cutover plan, shared CRUD route integration, and parity task matrix.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/design-doc/01-implementation-plan-runtime-cutover-in-pinocchio-and-go-go-os.md — Application rollout architecture
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/tasks.md — Granular cutover and parity checklist

## 2026-02-24

Advanced GP-24 execution with explicit profile-selection runtime policy tests in both apps and synchronized task checklist state.

### Decisions

- In-flight conversation policy is now explicit: reusing the same `conv_id` after selected-profile change triggers runtime rebuild when runtime fingerprint differs.
- `POST /api/chat/profile` selection is validated as effective for both new conversation creation and in-flight conversation runtime transitions.

### Verification

- `go test ./cmd/web-chat -run 'TestAppOwnedProfileSelection_(InFlightConversation_RebuildsRuntime|AffectsNextConversationCreation)' -count=1` (pinocchio): pass.
- `go test ./go-inventory-chat/cmd/hypercard-inventory-server -run 'TestProfileE2E_SelectedProfileChange_RebuildsInFlightConversationRuntime|TestProfileE2E_ListSelectChat_RuntimeKeyReflectsSelection' -count=1` (go-go-os): pass.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/app_owned_chat_integration_test.go — Added integration tests for selected-profile effects on next and in-flight conversation runtime.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go — Added integration test proving in-flight conversation runtime rebuild on selected-profile change.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/tasks.md — Checked completed GP-24 work items.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/design-doc/01-implementation-plan-runtime-cutover-in-pinocchio-and-go-go-os.md — Recorded explicit in-flight conversation policy decision.

## 2026-02-24

Integrated server-confirmed current-profile writes into Go-Go-OS profile selector state flow.

### Verification

- `pnpm exec vitest run src/chat/runtime/profileApi.test.ts src/chat/runtime/useProfiles.test.ts` (go-go-os/packages/engine): pass.
- `pnpm exec tsc -b` (go-go-os/packages/engine): pass.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/runtime/useSetProfile.ts — `useSetProfile` now calls `setCurrentProfile` and updates local state from server-confirmed slug.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/components/ChatConversationWindow.tsx — Selector now invokes async server-backed profile selection updates.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/runtime/profileApi.test.ts — Added current-profile API decode/request contract test.

## 2026-02-24

Removed shared profile API payload alias fallback (`profile`) to enforce slug-only request contract under hard cutover policy.

### Verification

- `go test ./cmd/web-chat -count=1` (pinocchio): pass.
- `go test ./pkg/webchat/http -count=1` (pinocchio): pass (`no test files`, compile validation).

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/pkg/webchat/http/profile_api.go — Removed `profile` alias fields and fallback parsing in create-profile and set-current-profile payload decoding.

## 2026-02-24

Completed compatibility cleanup audit for obsolete toggle references in maintained docs.

### Verification

- `rg -n "PINOCCHIO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE|profile alias|\"profile\" field|legacy middleware switch|env var" pinocchio/pkg/doc geppetto/pkg/doc go-go-os -g'*.md'`
- Result: no maintained docs reference the removed middleware-switch toggle; only historical ticket artifacts remain under `ttmp/`.

## 2026-02-24

Hardened Pinocchio web-chat frontend profile contract handling and selector reconciliation to match shared profile API semantics.

### Verification

- `npm run typecheck` (pinocchio/cmd/web-chat/web): pass.
- `npx vitest run src/store/profileApi.test.ts src/webchat/profileSelection.test.ts` (pinocchio/cmd/web-chat/web): pass.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/web/src/store/profileApi.ts — Added shared-contract response decoding for profile list/current-profile endpoints, including indexed-object list fallback.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx — Added selected-profile reconciliation logic and server-refresh fallback after profile set failures.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/web/src/webchat/profileSelection.ts — Added deterministic selection resolver for app/server/options profile reconciliation.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/web/src/store/profileApi.test.ts — Added frontend API contract tests for list/get/set profile behavior.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/web/src/webchat/profileSelection.test.ts — Added tests covering default display, selection change stability, and stale selection fallback.

## 2026-02-24

Added Go-Go-OS profile-selector behavior tests for bidirectional switching and pre-send selection flow.

### Verification

- `pnpm exec vitest run src/chat/components/profileSelectorState.test.ts src/chat/runtime/profileApi.test.ts src/chat/runtime/useProfiles.test.ts` (go-go-os/packages/engine): pass.
- `pnpm exec tsc -b` (go-go-os/packages/engine): pass.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/components/profileSelectorState.ts — Added pure selector-state helpers for selected-profile value and change-target resolution.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/components/profileSelectorState.test.ts — Added tests covering inventory/default switching, pre-send selection, and stale-selection fallback.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/packages/engine/src/chat/components/ChatConversationWindow.tsx — Wired selector UI logic through tested helper functions.

## 2026-02-24

Captured cross-app profile API/runtime parity evidence and added explicit parity fixture specification.

### Verification

- `go test ./cmd/web-chat -run 'TestProfileAPI_CRUDRoutesAreMounted|TestAppOwnedProfileSelection_InFlightConversation_RebuildsRuntime|TestAppOwnedProfileSelection_AffectsNextConversationCreation' -count=1` (pinocchio): pass.
- `go test ./go-inventory-chat/cmd/hypercard-inventory-server -run 'TestProfileAPI_CRUDRoutesAreMounted|TestProfileE2E_ListSelectChat_RuntimeKeyReflectsSelection|TestProfileE2E_SelectedProfileChange_RebuildsInFlightConversationRuntime' -count=1` (go-go-os): pass.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/sources/01-profile-parity-fixture.yaml — Shared parity fixture specification for profile registry semantics.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/analysis/01-cross-app-profile-api-and-runtime-parity-report.md — Parity report comparing CRUD, profile selection, and runtime marker behavior.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/tasks.md — Checked parity task block complete.

## 2026-02-24

Added invalid-input API coverage, post-cutover troubleshooting guidance, and explicit cutover commit index.

### Verification

- `go test ./cmd/web-chat -run 'TestProfileAPI_InvalidSlugAndRegistry_ReturnBadRequest' -count=1` (pinocchio): pass.
- `go test ./go-inventory-chat/cmd/hypercard-inventory-server -run 'TestProfileAPI_InvalidSlugAndRegistry_ReturnBadRequest' -count=1` (go-go-os): pass.

### Cutover Commit Index

Pinocchio:

- `673b8ad` — mount shared profile CRUD and verify route contract lifecycle.
- `cc2b10c` — profile-selection integration tests for next and in-flight conversation behavior.
- `036a128` — hard-cutover compatibility removal for profile alias fallback in profile API payloads.
- `3077aa5` — frontend contract decoding + selector reconciliation fixes.

Go-Go-OS:

- `903a5fe` — shared profile CRUD integration and contract checks.
- `df0a590` — in-flight runtime rebuild integration coverage.
- `7801932` — selector state synced with server-confirmed current profile response.
- `8928527` — selector switching tests (inventory/default and pre-send selection) via pure helper module.

### Related Files

- /home/manuel/workspaces/2026-02-23/add-profile-registry/pinocchio/cmd/web-chat/app_owned_chat_integration_test.go — Added invalid registry/slug API behavior integration test.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go — Added invalid registry/slug API behavior integration test.
- /home/manuel/workspaces/2026-02-23/add-profile-registry/geppetto/ttmp/2026/02/24/GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS--runtime-cutover-in-pinocchio-and-go-go-os/design-doc/01-implementation-plan-runtime-cutover-in-pinocchio-and-go-go-os.md — Added post-cutover troubleshooting section with command checklist.
