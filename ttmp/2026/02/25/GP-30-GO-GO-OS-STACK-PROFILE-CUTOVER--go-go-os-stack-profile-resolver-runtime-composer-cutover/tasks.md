# Tasks

## Phase 0 - Scope lock and baseline

- [ ] Confirm hard-cut migration target against GP-28:
  - registry-backed profile resolution,
  - stack lineage/trace metadata propagation,
  - lineage-aware runtime fingerprint usage.
- [ ] Capture baseline behavior/tests in:
  - `internal/pinoweb/request_resolver_test.go`,
  - `internal/pinoweb/runtime_composer_test.go`,
  - `cmd/go-go-os-launcher/main_integration_test.go`.

## Phase 1 - Request resolver adoption

- [ ] Refactor `internal/pinoweb/request_resolver.go` to pass through:
  - `registrySlug`,
  - `runtimeKey`,
  - policy-gated `requestOverrides`.
- [ ] Remove local override policy duplication in favor of geppetto resolver outcomes.

## Phase 2 - Runtime composer adoption

- [ ] Refactor `internal/pinoweb/runtime_composer.go` to consume stack-aware resolved runtime outputs.
- [ ] Replace local fingerprint input shaping with resolver-provided runtime fingerprint where possible.
- [ ] Ensure middleware/tool/system prompt composition respects already-merged stack runtime.

## Phase 3 - API/integration surface

- [ ] Confirm web API payloads and launcher integration support multi-registry profile selection.
- [ ] Ensure response metadata includes stack lineage/trace and runtime fingerprint where clients need diagnostics/caching.

## Phase 4 - Verification

- [ ] Run: `go test ./go-inventory-chat/...`.
- [ ] Add/adjust tests for:
  - stack profile selection and defaulting,
  - request override policy enforcement,
  - fingerprint stability/sensitivity in launcher runtime flows.

## Phase 5 - Documentation and rollout

- [ ] Update go-go-os docs/runbooks for stack profile behavior and required request fields.
- [ ] Link completion notes back to GP-28 changelog/index.
