# Tasks

## Shared CRUD Route Mounting

- [x] Audit shared profile handler registration function for full CRUD + current-profile coverage.
- [x] Mount shared profile handlers in `pinocchio/cmd/web-chat/main.go`.
- [x] Mount shared profile handlers in `go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go`.
- [x] Ensure both apps use the same default registry slug behavior.
- [x] Ensure both apps parse `registry` query parameter consistently.
- [x] Add route smoke test in Pinocchio for all mounted profile endpoints.
- [x] Add route smoke test in Go-Go-OS server for all mounted profile endpoints.

## Runtime Composer Cutover

- [x] Remove app-local hardcoded middleware list where profile runtime should drive behavior.
- [x] Refactor Pinocchio runtime composer to consume resolved profile runtime end-to-end.
- [x] Refactor Go-Go-OS runtime composer to consume same resolved profile runtime semantics.
- [x] Ensure middleware chain build order is deterministic across both apps.
- [x] Ensure tool registry defaults and profile tool overrides merge consistently.
- [x] Add tests asserting runtime fingerprint changes when profile version changes.
- [x] Add tests asserting no engine rebuild when profile version unchanged.

## Profile Selection Semantics

- [x] Verify `POST /api/chat/profile` persists selected profile state (cookie/session).
- [x] Verify `GET /api/chat/profile` returns selected profile and default fallback correctly.
- [x] Add integration test for selection change affecting next conversation creation.
- [x] Decide and document policy for in-flight conversations when selected profile changes.
- [x] Add tests for chosen in-flight conversation policy.
- [x] Ensure profile selector UI reflects server-selected profile after write response.

## Go-Go-OS Frontend Integration

- [x] Align Go-Go-OS profile API client with shared backend contract.
- [x] Fix profile list parsing edge cases (array vs map-index legacy shape).
- [ ] Ensure profile selector can switch between at least three seeded profiles.
- [ ] Add UI test for switching from `inventory` to `default` and back.
- [ ] Add UI test for switching selected profile before first message send.

## Pinocchio Frontend Integration

- [ ] Ensure web-chat frontend uses the same profile list/get/set APIs.
- [ ] Ensure selected profile widget updates when server rejects invalid profile.
- [ ] Add frontend tests for default profile display and selection changes.
- [ ] Add regression test for stale selected profile cookie value.

## Compatibility Cleanup (Hard Cutover)

- [x] Remove remaining compatibility toggles related to legacy middleware/profile behavior.
- [x] Remove dead route aliases or fallback handlers superseded by shared CRUD routes.
- [x] Remove obsolete comments/docs referencing removed toggles.
- [x] Verify no `os.Getenv` reads remain for legacy profile middleware switching.

## Cross-App Parity Testing

- [ ] Build a parity fixture with same registry/profile set for both apps.
- [ ] Run same CRUD sequence against both servers and compare responses.
- [ ] Run same profile selection sequence and compare selected-profile responses.
- [ ] Run one inference request after profile switch and compare effective runtime markers.
- [ ] Capture parity report in ticket docs/changelog.

## Manual QA Checklist

- [ ] Start Pinocchio web-chat and verify profile list/create/update/delete/default/select manually.
- [ ] Start Go-Go-OS inventory server and verify same manual profile flow.
- [ ] Verify a true `default` profile with no middlewares runs without middleware-specific events.
- [ ] Verify switching to middleware-enabled profile produces expected middleware events.
- [ ] Verify error handling for invalid profile slug and invalid registry slug.

## Closeout

- [ ] Update changelog with cutover commits and compatibility removals.
- [ ] Add post-cutover troubleshooting notes to ticket docs.
- [ ] Confirm downstream teams notified of stable endpoint surface.
