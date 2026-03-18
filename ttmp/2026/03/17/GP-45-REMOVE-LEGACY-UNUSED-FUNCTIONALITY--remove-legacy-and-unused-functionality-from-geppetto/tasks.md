# Tasks

## Investigation And Planning

- [x] Inventory legacy support paths, unused helpers, and complexity hotspots found during the RuntimeKeyFallback review.
- [x] Classify each candidate by confidence and risk.
- [x] Produce a detailed design and implementation guide for new engineers.
- [x] Convert the broad cleanup inventory into a concrete implementation queue ordered by risk and reviewability.
- [x] Confirm mono-repo or downstream usage for each medium-risk candidate before code deletion.

## Phase 1: Low-risk hard cuts

- [x] Remove the no-op `pkg/sections/profile_registry_source.go` bridge and its tests now that profiles no longer contribute step-setting flags.
- [x] Remove all remaining references to `GatherFlagsFromProfileRegistry` and update any comments/docs that still frame it as a migration bridge.
- [x] Remove `engines.fromProfile` entirely from the JS module surface, generated types, tests, examples, and docs.
- [x] Remove `pkg/profiles/adapters.go` and its tests now that no in-repo callers use the wrapper helpers.
- [x] Remove stale docs that still describe request overrides, old policy fields, or `runtimeKey` resolver options as live behavior.

## Phase 2: Compatibility cleanup with focused validation

- [x] Confirm whether the remaining mutable `ProfileStore` surface should be narrowed to registry/profile upserts only after GP-41 made registry services read-only.
- [x] If safe, remove `DeleteProfile` and `SetDefaultProfile` from `ProfileStore`, memory/sqlite/yaml implementations, and tests.
- [x] Evaluate whether `MirrorLegacyInferenceKeys` in `pkg/inference/engine/run_with_result.go` still has active consumers.
- [x] Evaluate whether `runtimeattrib.AddRuntimeAttributionToExtra` still needs to accept multiple legacy input shapes (`string`, `key`, `slug`, underscore variants).
- [ ] If no active consumers remain, simplify both paths to one canonical metadata envelope.
- [ ] Add or update tests to lock in the canonical shape only.

## Phase 3: Migration-shim cleanup

- [x] Remove bootstrap comments and help text that still frame direct flags as migration escape hatches if that migration is effectively complete.
- [ ] Decide whether the default pinocchio profile-registry auto-discovery path should remain or move to app-owned bootstrapping.

## Phase 4: Over-complex or lightly integrated machinery

- [ ] Confirm whether `StoreRegistry.extensionCodecs` is actually used in runtime code or can be removed.
- [ ] Confirm whether `NormalizeProfileExtensions` has any production callers outside tests.
- [ ] Confirm whether `ProjectRuntimeMiddlewareConfigsToExtensions` and `MiddlewareConfigFromExtensions` are used outside tests and docs.
- [ ] If extension/middleware-extension machinery is still desired, document the real supported flow; otherwise stage removal.
- [ ] Evaluate whether always-on `profile.stack.trace` generation should become opt-in debug output.
- [ ] Evaluate whether unused `ChainedRegistry` fields (`aggregateStore`, `registryOwners`) should be removed.

## Review Checklist

- [x] Each deletion candidate is backed by a grep-based usage check.
- [x] Each remaining compatibility shim has an explicitly stated downstream owner.
- [x] Docs are brought back into sync with actual code behavior.
- [ ] Cleanup is split into reviewable phases rather than one large risky change.
