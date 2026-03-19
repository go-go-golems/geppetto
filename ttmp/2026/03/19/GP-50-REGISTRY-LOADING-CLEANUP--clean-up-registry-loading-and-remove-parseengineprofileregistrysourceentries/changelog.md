# Changelog

## 2026-03-19

- Initial workspace created
- Added the migration analysis and task breakdown for removing `ParseEngineProfileRegistrySourceEntries` and pushing string-list decoding to Glazed where available.

## 2026-03-19

Completed task 1: switched the Glazed runner example helper contract to []string registry sources and adapted the flag-based runner-registry example at the caller boundary.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/internal/runnerexample/inference_settings.go — Shared example helper now accepts []string directly
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-glazed-registry-flags/main.go — Glazed example decodes profile-registries as TypeStringList
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-registry/main.go — Flag-based example now adapts to the slice-based helper contract


## 2026-03-19

Completed the shared profile-settings migration: Geppetto now publishes the canonical section, Geppetto/Pinocchio section owners consume it, and most raw Cobra profile/profile-registries flags were replaced by attaching the shared section. Remaining exceptions are pure flag-package/script entrypoints such as geppetto/cmd/examples/geppetto-js-lab, geppetto/cmd/examples/runner-registry, and pinocchio/scripts/profile-infer-once.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go — Public canonical ProfileSettings section
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/cmds/js.go — JS command reads shared-section profile values instead of manual raw flags
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/main.go — Root inherited flags now come from the shared section
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/switch-profiles-tui/main.go — Plain Cobra TUI now attaches the shared section
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/main.go — Web-chat shared section adoption and []string registry handling


## 2026-03-19

Follow-up cleanup: converted `cmd/examples/runner-registry` off raw `flag` parsing and onto the shared Geppetto `profile-settings` section. The remaining exceptions in this ticket are now limited to standalone non-Cobra/script entrypoints.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-registry/main.go — Cobra example now mounts the shared section for `--profile` and `--profile-registries`
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries/analysis/02-reusable-profile-settings-section-migration-analysis.md — Exception inventory updated after the runner-registry migration

## 2026-03-19

Added a detailed intern-facing design and implementation guide for Pinocchio and Geppetto CLI bootstrap, covering loader history, current runtime/config/profile resolution paths, recommended shared runtime helper design, and the proposed split between app config and profile-registry baseline inference settings.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries/design-doc/01-pinocchio-cli-geppetto-config-and-profile-bootstrap-guide.md — Primary guide delivered for the requested analysis/design/implementation write-up
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd.go — Current loaded-command runtime path analyzed in the guide
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go — Current thin-command runtime path analyzed in the guide

## 2026-03-19

Imported the external guide `geppetto_cli_profile_guide.md` into the ticket, reviewed it as the preferred direction for the next phase, and rewrote the follow-up task list around a narrower simplification goal: baseline app config loading plus engine profile overlay, with runtime profiles explicitly deferred.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries/sources/local/geppetto_cli_profile_guide.md — Imported source adopted as the better design basis for follow-up work
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries/tasks.md — Next-phase tasks rewritten to focus on config/profile simplification and shared engine bootstrap

## 2026-03-19

Completed the first documentation milestone for the next phase and committed it as `34401d6` (`docs(ticket): add bootstrap simplification design notes`). The ticket now contains the explicit scope decision, the CLI entrypoint inventory, the config-versus-registry semantics note, and the minimal first-phase implementation guide.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries/design-doc/02-adopt-imported-cli-profile-guide-and-defer-runtime-profiles.md — Decision note adopting the imported guide and deferring runtime profiles
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries/analysis/03-geppetto-backed-cli-entrypoint-inventory-and-bootstrap-classification.md — Inventory of loaded-command, Glazed/Cobra, and thin bootstrap command families
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries/design-doc/03-baseline-config-and-engine-profile-registry-semantics.md — Baseline config versus engine profile registry semantics and discovery rules
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-registry-loading-and-remove-parseengineprofileregistrysourceentries/design-doc/04-minimal-first-phase-bootstrap-implementation-without-runtime-profiles.md — First-phase implementation guide for shared bootstrap helpers

## 2026-03-19

Completed the first code task from the new plan and committed it as `76ae603` (`refactor(profiles): add shared cli profile selection helper`). Pinocchio now has an explicit `ResolveCLIProfileSelection(...)` helper plus focused precedence/fallback tests, and the thin runtime helper path consumes that shared selection logic instead of duplicating it.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_selection.go — New shared profile-selection contract and resolver
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go — Final inference settings path now delegates selection to the shared helper
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_selection_test.go — New tests covering explicit precedence, config fallback, and XDG fallback

## 2026-03-19

Completed the next helper milestone as `0be81c0` (`refactor(profiles): add shared cli engine settings helper`). The shared bootstrap path now exposes both base and final inference settings, selected profile metadata, resolved engine profile metadata, and an engine-construction helper built on top of the resolved final settings.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_engine_settings.go — Shared final-settings and engine-construction helper layer
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go — Thin runtime helper now wraps the richer shared engine-settings resolver
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime_test.go — Added tests for base/final settings separation and engine creation

## 2026-03-19

Completed the loaded-command migration as `a755724` (`refactor(profiles): share loaded command bootstrap`). `PinocchioCommand.RunIntoWriter(...)` now delegates profile selection and profile overlay to the shared bootstrap path while preserving command-local defaults through `baseSettingsFromParsedValues(...)`, and the shared implementation was moved into a new cycle-free `profilebootstrap` package so both `cmds` and `helpers` can depend on it.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd.go — Loaded-command runtime path now uses the shared parsed-values/base-settings resolver flow
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go — Cycle-free shared profile-selection implementation
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go — Cycle-free shared final-settings and engine-construction implementation
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_selection.go — Compatibility wrapper onto the shared `profilebootstrap` implementation
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_engine_settings.go — Compatibility wrapper onto the shared `profilebootstrap` implementation

## 2026-03-19

Completed the next hard-cutover bootstrap milestone as `0a1056d` (`refactor(profiles): cut over thin bootstrap commands`) and `475131e` (`test(profiles): codify no registry fallback`). Thin/manual bootstrap paths now use the shared `profilebootstrap` helpers, the implicit `profiles.yaml` fallback is removed, `profile` without registries now fails explicitly, and `profile_runtime.go` no longer carries its own config-file resolution logic.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go — Removed implicit registry fallback and exported shared config-file helpers for thin bootstrap reuse
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go — Shared engine-settings path now enforces explicit registries when a profile is selected
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/parse-helpers.go — Legacy thin helper now reuses shared config-file bootstrap and `[]string` registry inputs
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go — Reduced to the runtime-specific compatibility wrapper only
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/cmds/js.go — JS bootstrap path now consumes shared profile/bootstrap helpers while keeping unrelated local runtime edits unstaged
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/examples/internal/tuidemo/cli.go — Plain Cobra example now builds parsed values with `config-file` plus shared profile section
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/examples/simple-chat/main.go — Example wrapper no longer injects a fake default profile and now passes registry slices directly
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/main.go — Web-chat startup now uses shared selection/base-settings helpers and tolerates baseline-only mode
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/profile_policy.go — Request resolver now distinguishes baseline-only mode from explicit invalid profile/registry selection
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_selection_test.go — No-fallback behavior is now codified directly in helper tests

## 2026-03-19

Completed the SQLite follow-up as `a8763be` (`refactor(commands): remove dead sqlite tool path`). The remaining command references to the deleted `sqlitetool` package were removed from `web-chat` and `simple-chat-agent`, the web-chat schema tests were updated to match the new middleware surface, and the previously blocked command packages now verify successfully.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/middleware_definitions.go — Removed the dead `sqlite` middleware definition from the web-chat runtime registry
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/main.go — Removed startup SQLite dependency injection that only existed for the deleted middleware
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/agents/simple-chat-agent/main.go — Removed the dead SQLite tool middleware path from the agent command
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/main_profile_registries_test.go — Fixed the shared-profile test helper to construct valid Glazed section values
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/profile_policy_test.go — Schema endpoint expectations now match the post-SQLite middleware inventory

## 2026-03-19

Completed the remaining profile-cleanup implementation tasks via `eb02e18` (`refactor(loader): preserve loaded command base settings`) and `5466c5b` (`refactor(profiles): share cli selection value assembly`). Loaded YAML commands now carry explicit base inference settings from the loader, manual/bootstrap commands now build config/profile selection values through one shared helper, and targeted parity tests cover the shared bootstrap contract.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/examples/internal/tuidemo/cli.go — TUI demo manual bootstrap path now reuses the shared CLI selection-values helper
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/cmds/js.go — JS manual bootstrap path now reuses the shared CLI selection-values helper
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd.go — Loaded command runtime now prefers loader-provided base settings over parse-log reconstruction
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/loader.go — Loader now stores explicit baseline inference settings on loaded commands
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings_test.go — Parity test covers internal versus from-base engine resolution paths
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go — Shared builder for config-file plus profile-selection parsed values


## 2026-03-19

Added `01b8780` (`feat(debug): print resolved inference settings`) so loaded Pinocchio commands can print the final merged inference settings after config/env/flags/profile resolution and before engine creation. This was validated both with `go test ./pkg/cmds -count=1` and by running `go run ./cmd/pinocchio code professional test --chat --profile gpt-5-mini --print-inference-settings`.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd.go — Loaded command path now prints final resolved inference settings and exits before engine creation
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd_profile_registry_test.go — Regression test covers printed merged settings and early exit before engine creation
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmdlayers/helpers.go — New helper-layer debug flag for final inference settings printing

## 2026-03-19

Added `--print-inference-settings-sources` so loaded Pinocchio commands can print the final merged inference settings together with per-setting source logs. The new trace shows command baselines, config/env/default parse steps, and the final profile overlay step in one YAML view, which closes the observability gap between `--print-parsed-fields` and `--print-inference-settings`.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go — New source-trace builder that maps parsed fields and profile overlays onto final inference-setting paths
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd.go — Loaded command path now prints traced final inference settings and exits before engine creation
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmdlayers/helpers.go — New helper-layer debug flag for traced inference settings printing
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd_profile_registry_test.go — Regression test covers command/config/profile provenance ordering for final settings
