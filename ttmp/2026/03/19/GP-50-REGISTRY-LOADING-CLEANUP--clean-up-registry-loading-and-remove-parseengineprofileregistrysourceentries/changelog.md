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
