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
