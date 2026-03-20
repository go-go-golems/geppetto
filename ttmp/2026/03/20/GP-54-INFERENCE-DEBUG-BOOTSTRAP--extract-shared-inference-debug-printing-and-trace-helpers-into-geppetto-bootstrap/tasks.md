# Tasks

## TODO

## Completed

- [x] Create Geppetto ticket `GP-54-INFERENCE-DEBUG-BOOTSTRAP`.
- [x] Add a detailed design doc for the extraction.
- [x] Add a chronological diary for the research work.
- [x] Map the current ownership split across Geppetto bootstrap, Pinocchio wrappers, Pinocchio call sites, and a downstream backend.
- [x] Identify the clean-cut boundary: Geppetto owns generic inference debug behavior; Pinocchio keeps only app-specific bootstrap configuration and broader app settings.
- [x] Produce an intern-oriented implementation guide with prose, API sketches, pseudocode, diagrams, phased migration steps, and file references.
- [x] Simplify the target design to one debug flag, one combined output, and plain `***` masking.
- [x] Break the implementation into granular repo-by-repo execution tasks with commit boundaries.
- [x] Task 1.1: Create `geppetto/pkg/cli/bootstrap/inference_debug.go`.
- [x] Task 1.2: Move `InferenceSettingSource`, `BuildInferenceSettingsSourceTrace(...)`, and the private trace helpers out of `pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go`.
- [x] Task 1.3: Add `BuildInferenceTraceParsedValues(...)` in Geppetto so callers can rebuild hidden-base parsed values through `AppBootstrapConfig`.
- [x] Task 1.4: Add `InferenceDebugSettings`, `InferenceDebugSectionSlug`, and `NewInferenceDebugSection()` in Geppetto with only `--print-inference-settings`.
- [x] Task 1.5: Add one combined debug renderer in Geppetto that emits final settings together with source provenance and masks sensitive values as `***`.
- [x] Task 1.6: Add `HandleInferenceDebugOutput(...)` in Geppetto so callers can print-and-exit through one entrypoint.
- [x] Task 1.7: Build and format `geppetto/`, then commit the shared-helper extraction.
- [x] Task 2.1: Switch `pinocchio/pkg/cmds/cmd.go` to call the Geppetto helper instead of printing locally.
- [x] Task 2.2: Switch `pinocchio/cmd/pinocchio/cmds/js.go` to mount `bootstrap.NewInferenceDebugSection()` and use `HandleInferenceDebugOutput(...)`.
- [x] Task 2.3: Remove `pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go`.
- [x] Task 2.4: Remove `cmdlayers.NewInferenceDebugParameterLayer()` and the old `print-inference-settings-sources` plumbing from Pinocchio helper settings.
- [x] Task 2.5: Update any now-stale Pinocchio CLI assertions that still expect `--print-inference-settings-sources`.
- [x] Task 2.6: Build and test `pinocchio/`, then commit the Pinocchio clean cut.
- [x] Task 3.1: Replace the CozoDB backend’s local debug section with the shared Geppetto section or a thin local wrapper that only preserves app-local flag placement.
- [x] Task 3.2: Remove `buildInferenceTraceParsedValues(...)`, `writeRedactedYAML(...)`, and the old debug-only redaction helpers from the backend once the shared helper is wired.
- [x] Task 3.3: Remove the backend’s compatibility alias flags for source-only debug output.
- [x] Task 3.4: Re-run backend validation commands and update the backend ticket diary if behavior changes materially.
- [x] Task 3.5: Commit the downstream backend migration.
- [x] Task 4.1: Update GP-54 design docs, tasks, changelog, and diary with the implemented API names and any deviations discovered during the code move.
- [x] Task 4.2: Run `docmgr doctor --ticket GP-54-INFERENCE-DEBUG-BOOTSTRAP --stale-after 30`.
- [x] Task 4.3: Re-upload the finished ticket bundle to reMarkable.
