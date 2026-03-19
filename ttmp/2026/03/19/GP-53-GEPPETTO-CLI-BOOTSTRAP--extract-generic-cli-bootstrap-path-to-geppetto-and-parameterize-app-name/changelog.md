# Changelog

## 2026-03-19

- Initial workspace created
# Changelog

## 2026-03-19

Created the follow-up ticket for extracting the generic CLI bootstrap path from Pinocchio into Geppetto and parameterizing the application identity. Added the initial design/implementation guide covering the current code split, the proposed Geppetto package boundary, the app-configurable bootstrap surface, migration phases, and the main file inventory that will need to change.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-53-GEPPETTO-CLI-BOOTSTRAP--extract-generic-cli-bootstrap-path-to-geppetto-and-parameterize-app-name/design-doc/01-generic-geppetto-cli-bootstrap-extraction-and-app-name-parameterization-guide.md — Primary analysis and implementation guide for the extraction
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go — Current Pinocchio-owned generic profile-selection candidate
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go — Current Pinocchio-owned generic engine-settings candidate
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go — Existing Geppetto bootstrap path with Pinocchio-specific assumptions
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/internal/runnerexample/inference_settings.go — Current Geppetto registry-only helper path

## 2026-03-19

Expanded the GP-53 task list into smaller execution steps so the extraction can be implemented and reviewed in clean layers: package creation and config surface first, generic resolver porting second, Geppetto tests third, and Pinocchio wrapper cutover last.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-53-GEPPETTO-CLI-BOOTSTRAP--extract-generic-cli-bootstrap-path-to-geppetto-and-parameterize-app-name/tasks.md — Refined implementation checklist for the extraction
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-53-GEPPETTO-CLI-BOOTSTRAP--extract-generic-cli-bootstrap-path-to-geppetto-and-parameterize-app-name/reference/01-diary.md — New implementation diary for the extraction work

## 2026-03-19

Implemented the Geppetto-owned generic CLI bootstrap package in `pkg/cli/bootstrap` and cut Pinocchio over to a thin wrapper config. The new package owns the generic resolved profile/engine contracts, config discovery, profile selection, base inference settings resolution, final engine settings resolution, and engine construction. Application identity is now caller-configurable through app name and env prefix instead of being hardcoded to Pinocchio.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/config.go — New app bootstrap config surface with app name/env prefix/config mapper/section builder callbacks
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/profile_selection.go — Generic config-file discovery, profile selection, and CLI selection value helpers
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/engine_settings.go — Generic base/final engine settings resolution and engine-construction helpers
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/bootstrap_test.go — Geppetto tests covering app/env parameterization, no-fallback behavior, and from-base parity
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go — Pinocchio wrapper config now binds the generic Geppetto package to `pinocchio` / `PINOCCHIO`
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go — Pinocchio wrapper now delegates generic engine-resolution behavior to Geppetto

## 2026-03-19

Closed the remaining GP-53 decision tasks. The old middleware helpers in `pkg/sections` are now explicitly treated as legacy wiring for existing examples, not the preferred home for new bootstrap logic. The example split is also now explicit: direct full-flag examples continue to use `factory.NewEngineFromParsedValues(...)`, while registry-aware or config-aware example paths are the candidates for the new `pkg/cli/bootstrap` package.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go — Marked the legacy Cobra middleware helper as non-preferred for new applications
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_sections.go — Marked the legacy profile middleware helper as non-preferred for new applications
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-53-GEPPETTO-CLI-BOOTSTRAP--extract-generic-cli-bootstrap-path-to-geppetto-and-parameterize-app-name/tasks.md — All GP-53 tasks are now checked off
