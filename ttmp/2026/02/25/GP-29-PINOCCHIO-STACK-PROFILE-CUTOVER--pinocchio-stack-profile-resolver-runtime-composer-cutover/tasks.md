# Tasks

## Phase 0 - Scope lock and baseline

- [x] Confirm hard-cut migration target against GP-28:
  - no legacy profile fallback path,
  - registry-backed resolve required,
  - stack lineage/fingerprint metadata consumed downstream.
- [x] Capture baseline behavior and tests in:
  - `cmd/web-chat/profile_policy_test.go`,
  - `cmd/web-chat/runtime_composer_test.go`,
  - `cmd/web-chat/app_owned_chat_integration_test.go`.

## Phase 1 - Request resolver adoption

- [x] Refactor `cmd/web-chat/profile_policy.go` to delegate request override policy checks to geppetto profile resolution output instead of local duplication.
- [x] Ensure request payload supports:
  - `registrySlug`,
  - `runtimeKey`,
  - policy-gated `requestOverrides`.
- [x] Remove stale local override merge logic that duplicates geppetto canonicalization/policy semantics.

## Phase 2 - Runtime composer adoption

- [x] Refactor `cmd/web-chat/runtime_composer.go` to consume stack-aware resolved runtime outputs from geppetto resolver.
- [x] Remove redundant local runtime override parsers where behavior is now centralized in geppetto.
- [x] Ensure composed runtime fingerprints use resolver-provided lineage-aware fingerprint.

## Phase 3 - API surface and response metadata

- [x] Ensure chat/web APIs expose resolved profile metadata needed by clients:
  - `profile.stack.lineage`,
  - `profile.stack.trace`,
  - runtime fingerprint.
- [x] Confirm default profile selection and explicit profile selection behavior with multi-registry support.

## Phase 4 - Verification

- [x] Run: `go test ./cmd/web-chat/...`.
- [x] Add/adjust tests for:
  - stack profile resolution in web-chat flows,
  - denied/allowed request override behavior,
  - fingerprint and metadata propagation.
- [ ] Smoke-check CLI/web-chat runtime behavior manually where automated coverage is insufficient.

## Phase 5 - Documentation and rollout

- [ ] Update pinocchio runtime/profile docs for hard-cut stack profile behavior.
- [ ] Record migration notes for operators (removed legacy paths, required request fields).
- [ ] Link outcomes back to GP-28 changelog.
