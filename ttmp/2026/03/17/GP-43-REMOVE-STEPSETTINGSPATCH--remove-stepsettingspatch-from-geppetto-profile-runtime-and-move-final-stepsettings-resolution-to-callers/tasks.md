# Tasks

## Phase 1: Inventory and contract decisions

- [x] Create a dedicated Geppetto ticket for deleting `StepSettingsPatch`.
- [x] Inventory all Geppetto, Pinocchio, GEC-RAG, and Temporal Relationships references to `StepSettingsPatch`, `EffectiveStepSettings`, `BaseStepSettings`, and patch helpers.
- [x] Write a detailed design and implementation guide describing the target architecture and migration plan.
- [x] Add a ticket-local script that re-runs the `StepSettingsPatch` surface inventory.
- [x] Decide the replacement caller-owned API shape for final runtime resolution (`StepSettings`, runtime identity, middleware values, tool registry).
- [x] Decide that Geppetto profile resolution should return runtime metadata only, with no temporary compatibility adapter in core.
- [x] Fold `RuntimeKeyFallback` removal into the same hard-cut series instead of treating it as a separate later cleanup.

## Phase 2: Geppetto core API changes

- [x] Remove `StepSettingsPatch` from `geppetto/pkg/profiles.RuntimeSpec`.
- [x] Remove `BaseStepSettings` from `geppetto/pkg/profiles.ResolveInput`.
- [x] Remove `RuntimeKeyFallback` from `geppetto/pkg/profiles.ResolveInput`.
- [x] Remove `EffectiveStepSettings` from `geppetto/pkg/profiles.ResolvedProfile`.
- [x] Delete `geppetto/pkg/profiles/runtime_settings_patch_resolver.go` after all consumers are migrated.
- [x] Delete runtime-key fallback synthesis from `ResolveEffectiveProfile`.
- [x] Remove stack merge and stack trace handling for `/runtime/step_settings_patch`.
- [x] Update validation and clone logic that still references `StepSettingsPatch`.

## Phase 3: Geppetto caller-facing and compatibility surface cleanup

- [x] Update `geppetto/pkg/sections/profile_registry_source.go` to stop converting profile patch maps into Glazed source maps.
- [x] Update JS bindings to stop exposing `effectiveStepSettings`, `effectiveRuntime.step_settings_patch`, and `runtimeKeyFallback` as profile-resolution API concepts.
- [x] Update generated JS type definitions and examples.
- [x] Remove or rewrite Geppetto examples that depend on `step_settings_patch`.
- [x] Update profile YAML examples and migration playbooks to stop teaching `runtime.step_settings_patch` and Geppetto-owned runtime identity fallback.

## Phase 4: Downstream caller migration

- [x] Pinocchio and GEC-RAG: stop populating `RuntimeKeyFallback` in live webchat resolvers and derive runtime keys directly in app code.
- [x] Pinocchio: introduce an app-owned resolved runtime object that includes final `StepSettings`, runtime key, and runtime fingerprint.
- [x] Pinocchio: replace profile patch application in `cmd/web-chat/runtime_composer.go` with caller-owned final `StepSettings` resolution.
- [x] Pinocchio: update profile switch, TUI helpers, and scripts that currently rely on `resolved.EffectiveStepSettings` or Geppetto-owned runtime keys.
- [x] GEC-RAG: move final `StepSettings` resolution and runtime identity into resolver/app code and stop applying `ResolvedProfileRuntime.StepSettingsPatch` in runtime composition.
- [x] Temporal Relationships: move final `StepSettings` resolution and runtime identity into command/runtime setup and stop applying profile patch data during run-chat runtime composition.
- [x] Audit any remaining external helper code that still calls `ResolveEffectiveProfile(... BaseStepSettings: ...)` or depends on `RuntimeKeyFallback`.

## Phase 5: Migration and persistence/documentation cleanup

- [x] Update legacy profile migration code so it no longer writes `runtime.step_settings_patch`.
- [x] Decide how to handle existing YAML files containing `step_settings_patch`: hard error, soft warning, or one-shot migration.
- [x] Decide how to handle existing callers or scripts that still pass `runtimeKeyFallback`: hard error or immediate compile/runtime break.
- [x] Update README files, topic docs, and playbooks across Geppetto and Pinocchio.
- [x] Update tests and fixtures that assert merged effective step settings from profile patches or Geppetto-owned runtime key synthesis.

## Phase 6: Validation

- [x] Run focused Go tests in `geppetto/pkg/profiles`, `geppetto/pkg/js/modules/geppetto`, and `geppetto/pkg/sections`.
- [x] Run focused Pinocchio tests for web chat runtime composition, profile switching, and helper commands.
- [x] Run focused GEC-RAG and Temporal Relationships tests covering runtime resolution and engine composition.
- [x] Run `docmgr doctor` on the new ticket documents after implementation.
